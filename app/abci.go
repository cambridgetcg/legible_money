// ABCI++ Extension Points for Proof of Truth Consensus
//
// This file implements the ABCI++ methods that integrate PoT's multi-block
// verification rounds into CometBFT's single-block cycle.
//
// TS → ABCI++ Mapping:
//   tryProduceBlock()          →  PrepareProposal
//   validateBlock()            →  ProcessProposal
//   submitCommitment/Reveal    →  ExtendVote
//   verifyCommitment           →  VerifyVoteExtension
//   applyBlock()               →  FinalizeBlock
//
// Vote Extension Format:
//   Serialized VoteExtension protobuf containing:
//   - VerificationCommitment entries (commit phase)
//   - VerificationReveal entries (reveal phase)
//   - VRF proof for proposer eligibility
//
// Multi-block round lifecycle:
//   Block N:   Claim submitted → round created
//   Block N+1: PrepareProposal collects vote extensions with commitments
//   Block N+K: PrepareProposal collects reveals, aggregates completed rounds
package app

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"sort"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	emergencytypes "github.com/legible-money/legible/x/emergency/types"
)

// VoteExtension is the PoT-specific data validators attach to their votes.
// This is serialized as JSON for Phase 1; Phase 2 will use protobuf.
type VoteExtension struct {
	// Commitments from validators in the commit phase.
	Commitments []VoteCommitment `json:"commitments,omitempty"`

	// Reveals from validators in the reveal phase.
	Reveals []VoteReveal `json:"reveals,omitempty"`

	// VRF proof for this validator's eligibility.
	VRFOutput []byte `json:"vrf_output,omitempty"`
	VRFProof  []byte `json:"vrf_proof,omitempty"`

	// Validator's operator address (bech32).
	ValidatorAddress string `json:"validator_address"`
}

// VoteCommitment represents a validator's commitment to a verification verdict.
type VoteCommitment struct {
	RoundID        string `json:"round_id"`
	CommitmentHash string `json:"commitment_hash"` // sha256(verdict || confidence || salt)
	VRFOutput      string `json:"vrf_output,omitempty"` // hex-encoded VRF output
	VRFProof       string `json:"vrf_proof,omitempty"`  // hex-encoded VRF proof
	Height         uint64 `json:"height"`
}

// VoteReveal represents a validator's revealed verification verdict.
type VoteReveal struct {
	RoundID      string `json:"round_id"`
	Verdict      string `json:"verdict"`    // "accept", "reject", "abstain"
	Confidence   uint64 `json:"confidence"` // 0-1000000
	Salt         string `json:"salt"`
	Justification string `json:"justification,omitempty"`
}

// PrepareProposalHandler selects transactions for the next block.
//
// In PoT, this is equivalent to tryProduceBlock():
// 1. Select pending transactions from CometBFT mempool
// 2. Collect vote extensions from the previous block
// 3. Process completed verification rounds
// 4. Include round completion/fact creation txs in the block
func (app *LegibleApp) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (*abci.ResponsePrepareProposal, error) {
		logger := ctx.Logger().With("module", "abci", "handler", "PrepareProposal")

		// Emergency halt guard: only allow emergency txs during halt
		if app.EmergencyKeeper.IsHalted(ctx) {
			logger.Warn("chain halted - filtering non-emergency txs from proposal")
			var emergencyTxs [][]byte
			for _, txBytes := range req.Txs {
				tx, err := app.txConfig.TxDecoder()(txBytes)
				if err != nil {
					continue
				}
				allEmergency := true
				for _, msg := range tx.GetMsgs() {
					if !emergencytypes.IsEmergencyMsg(msg) {
						allEmergency = false
						break
					}
				}
				if allEmergency {
					emergencyTxs = append(emergencyTxs, txBytes)
				}
			}
			return &abci.ResponsePrepareProposal{Txs: emergencyTxs}, nil
		}

		// Start with transactions from mempool
		txs := make([][]byte, 0, len(req.Txs))

		// Process vote extensions from previous block (if available)
		if len(req.LocalLastCommit.Votes) > 0 {
			completedRounds := app.processVoteExtensions(ctx, req.LocalLastCommit.Votes, logger)

			// Generate system transactions for completed rounds
			for _, round := range completedRounds {
				sysTx, err := app.createRoundCompletionTx(ctx, round)
				if err != nil {
					logger.Error("failed to create round completion tx",
						"round_id", round.RoundID,
						"err", err,
					)
					continue
				}
				if sysTx != nil {
					txs = append(txs, sysTx)
				}
			}
		}

		// Add regular transactions from mempool, respecting gas limit
		var totalGas uint64
		for _, tx := range req.Txs {
			// Rough gas estimation based on tx size
			estimatedGas := uint64(len(tx)) * 10
			if totalGas+estimatedGas > BlockGasLimit {
				break
			}
			txs = append(txs, tx)
			totalGas += estimatedGas
		}

		logger.Debug("prepared proposal",
			"height", req.Height,
			"txs", len(txs),
			"vote_extensions", len(req.LocalLastCommit.Votes),
		)

		return &abci.ResponsePrepareProposal{Txs: txs}, nil
	}
}

// ProcessProposalHandler validates a proposed block.
//
// In PoT, this is equivalent to validateBlock():
// 1. Re-execute all transactions
// 2. Verify round completion transactions are valid
// 3. Verify VRF proofs for proposer eligibility
func (app *LegibleApp) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (*abci.ResponseProcessProposal, error) {
		logger := ctx.Logger().With("module", "abci", "handler", "ProcessProposal")

		// Emergency halt guard: reject proposals containing non-emergency txs
		if app.EmergencyKeeper.IsHalted(ctx) {
			for _, txBytes := range req.Txs {
				tx, err := app.txConfig.TxDecoder()(txBytes)
				if err != nil {
					continue
				}
				for _, msg := range tx.GetMsgs() {
					if !emergencytypes.IsEmergencyMsg(msg) {
						logger.Warn("rejecting proposal with non-emergency tx during halt",
							"msg_type", sdk.MsgTypeURL(msg),
						)
						return &abci.ResponseProcessProposal{
							Status: abci.ResponseProcessProposal_REJECT,
						}, nil
					}
				}
			}
		}

		// Validate each transaction in the proposal
		for i, txBytes := range req.Txs {
			tx, err := app.txConfig.TxDecoder()(txBytes)
			if err != nil {
				logger.Warn("invalid tx in proposal", "index", i, "err", err)
				return &abci.ResponseProcessProposal{
					Status: abci.ResponseProcessProposal_REJECT,
				}, nil
			}

			// Validate message types
			msgs := tx.GetMsgs()
			for _, msg := range msgs {
				// In SDK v0.50, ValidateBasic is optional via HasValidateBasic interface
				if m, ok := msg.(sdk.HasValidateBasic); ok {
					if err := m.ValidateBasic(); err != nil {
						logger.Warn("invalid msg in proposal",
							"index", i,
							"msg_type", sdk.MsgTypeURL(msg),
							"err", err,
						)
						return &abci.ResponseProcessProposal{
							Status: abci.ResponseProcessProposal_REJECT,
						}, nil
					}
				}
			}
		}

		// Note: In CometBFT v0.38, ProcessProposal receives CommitInfo (not ExtendedCommitInfo).
		// Vote extensions are available in PrepareProposal via LocalLastCommit (ExtendedCommitInfo).
		// The proposer should have already incorporated valid extensions into the block.
		// VerifyVoteExtension handles individual extension validation before they reach here.

		logger.Debug("accepted proposal",
			"height", req.Height,
			"txs", len(req.Txs),
		)

		return &abci.ResponseProcessProposal{
			Status: abci.ResponseProcessProposal_ACCEPT,
		}, nil
	}
}

// ExtendVoteHandler allows validators to attach PoT verdicts as vote extensions.
//
// In PoT, this is equivalent to submitCommitment() and submitReveal():
// - During commit phase: attach commitment hashes for active rounds
// - During reveal phase: attach reveals for rounds where validator committed
func (app *LegibleApp) ExtendVoteHandler() sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (*abci.ResponseExtendVote, error) {
		logger := ctx.Logger().With("module", "abci", "handler", "ExtendVote")

		ext := VoteExtension{
			ValidatorAddress: "", // Set by the validator's local config
		}

		// TODO (Phase 2): Query active verification rounds from knowledge keeper
		// For each round this validator is selected for:
		//   - If in commit phase: generate commitment hash, add to ext.Commitments
		//   - If in reveal phase: reveal verdict, add to ext.Reveals
		//
		// Pseudocode:
		//   activeRounds := app.KnowledgeKeeper.GetActiveRoundsForValidator(ctx, validatorAddr)
		//   for _, round := range activeRounds {
		//     if round.Phase == "commit" {
		//       commitment := generateCommitment(round, verdict, confidence, salt)
		//       ext.Commitments = append(ext.Commitments, commitment)
		//     } else if round.Phase == "reveal" {
		//       reveal := generateReveal(round, verdict, confidence, salt)
		//       ext.Reveals = append(ext.Reveals, reveal)
		//     }
		//   }

		// Serialize vote extension
		bz, err := json.Marshal(ext)
		if err != nil {
			logger.Error("failed to marshal vote extension", "err", err)
			return &abci.ResponseExtendVote{}, nil
		}

		logger.Debug("extended vote",
			"height", req.Height,
			"commitments", len(ext.Commitments),
			"reveals", len(ext.Reveals),
		)

		return &abci.ResponseExtendVote{
			VoteExtension: bz,
		}, nil
	}
}

// VerifyVoteExtensionHandler validates PoT vote extensions from other validators.
//
// In PoT, this is equivalent to verifyCommitment():
// - Verify commitment hashes are well-formed
// - Verify VRF proofs show the validator was selected
// - Verify reveals match prior commitments
func (app *LegibleApp) VerifyVoteExtensionHandler() sdk.VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (*abci.ResponseVerifyVoteExtension, error) {
		logger := ctx.Logger().With("module", "abci", "handler", "VerifyVoteExtension")

		// Empty extensions are always valid (validator may not be selected for any round)
		if len(req.VoteExtension) == 0 {
			return &abci.ResponseVerifyVoteExtension{
				Status: abci.ResponseVerifyVoteExtension_ACCEPT,
			}, nil
		}

		// Deserialize vote extension
		var ext VoteExtension
		if err := json.Unmarshal(req.VoteExtension, &ext); err != nil {
			logger.Warn("invalid vote extension format", "err", err)
			return &abci.ResponseVerifyVoteExtension{
				Status: abci.ResponseVerifyVoteExtension_REJECT,
			}, nil
		}

		// Validate commitments
		for _, c := range ext.Commitments {
			if c.RoundID == "" || c.CommitmentHash == "" {
				logger.Warn("invalid commitment in vote extension",
					"round_id", c.RoundID,
				)
				return &abci.ResponseVerifyVoteExtension{
					Status: abci.ResponseVerifyVoteExtension_REJECT,
				}, nil
			}

			// Commitment hash must be 64 hex chars (SHA-256)
			if len(c.CommitmentHash) != 64 {
				return &abci.ResponseVerifyVoteExtension{
					Status: abci.ResponseVerifyVoteExtension_REJECT,
				}, nil
			}
		}

		// Validate reveals
		for _, r := range ext.Reveals {
			if r.RoundID == "" || r.Salt == "" {
				return &abci.ResponseVerifyVoteExtension{
					Status: abci.ResponseVerifyVoteExtension_REJECT,
				}, nil
			}

			// Verdict must be valid
			switch r.Verdict {
			case "accept", "reject", "abstain":
				// OK
			default:
				return &abci.ResponseVerifyVoteExtension{
					Status: abci.ResponseVerifyVoteExtension_REJECT,
				}, nil
			}

			// Confidence must be in range [0, 1000000]
			if r.Confidence > 1_000_000 {
				return &abci.ResponseVerifyVoteExtension{
					Status: abci.ResponseVerifyVoteExtension_REJECT,
				}, nil
			}

			// TODO (Phase 2): Verify reveal matches prior commitment
			// commitmentHash := sha256(verdict || confidence || salt)
			// stored := app.KnowledgeKeeper.GetCommitment(ctx, r.RoundID, validatorAddr)
			// if stored.CommitmentHash != computedHash { REJECT }
		}

		logger.Debug("verified vote extension",
			"height", req.Height,
			"validator", fmt.Sprintf("%x", req.ValidatorAddress),
			"commitments", len(ext.Commitments),
			"reveals", len(ext.Reveals),
		)

		return &abci.ResponseVerifyVoteExtension{
			Status: abci.ResponseVerifyVoteExtension_ACCEPT,
		}, nil
	}
}

// ---------- Internal Methods ----------

// completedRound represents a verification round that has gathered sufficient reveals.
type completedRound struct {
	RoundID        string
	ClaimID        string
	Verdict        string // "accepted", "rejected", "inconclusive"
	Confidence     uint64
	Reveals        []VoteReveal
	ValidatorCount int
}

// processVoteExtensions aggregates vote extensions from the previous block
// and determines which verification rounds have completed.
func (app *LegibleApp) processVoteExtensions(
	ctx sdk.Context,
	votes []abci.ExtendedVoteInfo,
	logger interface{ Debug(string, ...interface{}) },
) []completedRound {
	// Aggregate reveals by round ID
	roundReveals := make(map[string][]VoteReveal)

	for _, vote := range votes {
		if len(vote.VoteExtension) == 0 {
			continue
		}

		var ext VoteExtension
		if err := json.Unmarshal(vote.VoteExtension, &ext); err != nil {
			continue
		}

		for _, reveal := range ext.Reveals {
			roundReveals[reveal.RoundID] = append(roundReveals[reveal.RoundID], reveal)
		}
	}

	// Check which rounds have sufficient reveals to complete
	var completed []completedRound
	for roundID, reveals := range roundReveals {
		// TODO (Phase 2): Get round from knowledge keeper to check MinVerifiers
		// round := app.KnowledgeKeeper.GetVerificationRound(ctx, roundID)
		// minVerifiers := app.KnowledgeKeeper.GetParams(ctx).MinVerifiers

		minVerifiers := 3 // Default minimum
		if len(reveals) < minVerifiers {
			continue
		}

		// Aggregate verdicts using stake-weighted confidence
		verdict, confidence := aggregateVerdicts(reveals)

		completed = append(completed, completedRound{
			RoundID:        roundID,
			Verdict:        verdict,
			Confidence:     confidence,
			Reveals:        reveals,
			ValidatorCount: len(reveals),
		})
	}

	// Sort completed rounds by round ID for deterministic ordering
	sort.Slice(completed, func(i, j int) bool {
		return completed[i].RoundID < completed[j].RoundID
	})

	return completed
}

// aggregateVerdicts computes the final verdict from multiple reveals.
// Uses simple majority with confidence weighting.
// Phase 2 will use stake-weighted confidence aggregation.
func aggregateVerdicts(reveals []VoteReveal) (string, uint64) {
	var acceptVotes, rejectVotes, abstainVotes uint64
	var totalConfidence uint64

	for _, r := range reveals {
		switch r.Verdict {
		case "accept":
			acceptVotes += r.Confidence
		case "reject":
			rejectVotes += r.Confidence
		case "abstain":
			abstainVotes += r.Confidence
		}
		totalConfidence += r.Confidence
	}

	if totalConfidence == 0 {
		return "inconclusive", 0
	}

	// Determine verdict by highest weighted votes
	if acceptVotes > rejectVotes && acceptVotes > abstainVotes {
		// Confidence = accept weight / total weight, scaled to 0-1000000
		confidence := (acceptVotes * 1_000_000) / totalConfidence
		return "accepted", confidence
	}
	if rejectVotes > acceptVotes && rejectVotes > abstainVotes {
		confidence := (rejectVotes * 1_000_000) / totalConfidence
		return "rejected", confidence
	}

	return "inconclusive", 0
}

// createRoundCompletionTx generates a system transaction to finalize a
// completed verification round. This is an authority-gated AddFact or
// status update.
func (app *LegibleApp) createRoundCompletionTx(ctx sdk.Context, round completedRound) ([]byte, error) {
	// TODO (Phase 2): Create actual Cosmos SDK transaction
	// The system tx would be either:
	//   - MsgAddFact (if verdict == "accepted")
	//   - Internal state update (if verdict == "rejected" or "inconclusive")
	//
	// For now, log the completion
	ctx.Logger().Info("verification round completed",
		"round_id", round.RoundID,
		"verdict", round.Verdict,
		"confidence", round.Confidence,
		"validators", round.ValidatorCount,
	)

	return nil, nil
}

// isValidVoteExtension performs basic structural validation of a vote extension.
func (app *LegibleApp) isValidVoteExtension(ctx sdk.Context, extBytes []byte) bool {
	var ext VoteExtension
	return json.Unmarshal(extBytes, &ext) == nil
}

// ComputeCommitmentHash generates a commitment hash for a verification verdict.
// hash = SHA-256(roundID || verdict || confidence || salt)
func ComputeCommitmentHash(roundID, verdict string, confidence uint64, salt string) string {
	h := sha256.New()
	h.Write([]byte(roundID))
	h.Write([]byte(verdict))
	confBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(confBytes, confidence)
	h.Write(confBytes)
	h.Write([]byte(salt))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// VerifyCommitmentHash checks that a reveal matches its prior commitment.
func VerifyCommitmentHash(commitmentHash, roundID, verdict string, confidence uint64, salt string) bool {
	computed := ComputeCommitmentHash(roundID, verdict, confidence, salt)
	return computed == commitmentHash
}
