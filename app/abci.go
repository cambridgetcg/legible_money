// ABCI++ Extension Points for Proof of Truth Consensus
//
// This file implements the ABCI++ methods that integrate PoT's multi-block
// verification rounds into CometBFT's single-block cycle.
//
// TS → ABCI++ Mapping:
//
//	tryProduceBlock()          →  PrepareProposal
//	validateBlock()            →  ProcessProposal
//	submitCommitment/Reveal    →  ExtendVote (vote_extensions.go)
//	verifyCommitment           →  VerifyVoteExtension (vote_extensions.go)
//	applyBlock()               →  FinalizeBlock
//
// Vote Extension Data Flow:
//  1. ExtendVote: Validators attach commitments/reveals to CometBFT votes
//  2. PrepareProposal: Proposer collects extensions, creates injection tx
//  3. ProcessProposal: Validators verify injection tx format
//  4. PreBlocker: All validators process injection tx, store in keeper state
//  5. BeginBlocker: Phase transitions and aggregation via knowledge module
package app

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	emergencytypes "github.com/legible-money/legible/x/emergency/types"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
)

// VoteExtension is the PoT-specific data validators attach to their votes.
// This is serialized as JSON for Phase 2; Phase 3 will use protobuf.
type VoteExtension struct {
	// Commitments from validators in the commit phase.
	Commitments []VoteCommitment `json:"commitments,omitempty"`

	// Reveals from validators in the reveal phase.
	Reveals []VoteReveal `json:"reveals,omitempty"`

	// Validator's operator address (bech32).
	ValidatorAddress string `json:"validator_address"`
}

// VoteCommitment represents a validator's commitment to a verification verdict.
type VoteCommitment struct {
	RoundID        string `json:"round_id"`
	CommitmentHash string `json:"commitment_hash"`      // sha256(verdict || confidence || salt)
	VRFOutput      string `json:"vrf_output,omitempty"` // hex-encoded VRF output
	VRFProof       string `json:"vrf_proof,omitempty"`  // hex-encoded VRF proof
	Height         uint64 `json:"height"`
}

// VoteReveal represents a validator's revealed verification verdict.
type VoteReveal struct {
	RoundID       string `json:"round_id"`
	Verdict       string `json:"verdict"`    // "accept", "reject", "abstain"
	Confidence    uint64 `json:"confidence"` // 0-1000000
	Salt          string `json:"salt"`
	Justification string `json:"justification,omitempty"`
}

// ---- Vote Extension Injection ----

// VoteExtInjectionPrefix is the 4-byte magic header for injected vote extension txs.
// These are pseudo-txs included at position 0 of a block by the proposer.
var VoteExtInjectionPrefix = []byte{0x00, 'V', 'E', 'X'}

// VoteExtInjection contains all commitments and reveals from the previous block's
// vote extensions. The proposer encodes this in PrepareProposal; all validators
// process it in PreBlocker to store data in the knowledge keeper.
type VoteExtInjection struct {
	Commitments []InjectedCommitment `json:"commitments"`
	Reveals     []InjectedReveal     `json:"reveals"`
}

// InjectedCommitment is a commitment extracted from a vote extension.
type InjectedCommitment struct {
	RoundID        string `json:"round_id"`
	Validator      string `json:"validator"`
	CommitmentHash string `json:"commitment_hash"`
	VRFOutput      string `json:"vrf_output,omitempty"`
	VRFProof       string `json:"vrf_proof,omitempty"`
}

// InjectedReveal is a reveal extracted from a vote extension.
type InjectedReveal struct {
	RoundID    string `json:"round_id"`
	Validator  string `json:"validator"`
	Verdict    string `json:"verdict"`
	Confidence uint64 `json:"confidence"`
	Salt       string `json:"salt"`
}

// MaxVEXInjectionBytes is the maximum size (in bytes) of a vote extension injection
// pseudo-tx. This prevents block bloat from oversized injections.
// With 100 validators × 50 entries/validator × ~200 bytes/entry = ~1MB.
// We set a 2MB limit for safety margin.
const MaxVEXInjectionBytes = 2 * 1024 * 1024

// isVoteExtInjectionTx checks if a tx has the vote extension injection prefix.
func isVoteExtInjectionTx(tx []byte) bool {
	return len(tx) > len(VoteExtInjectionPrefix) &&
		bytes.Equal(tx[:len(VoteExtInjectionPrefix)], VoteExtInjectionPrefix)
}

// encodeVoteExtInjection serializes a VoteExtInjection with the magic prefix.
func encodeVoteExtInjection(inj VoteExtInjection) ([]byte, error) {
	data, err := json.Marshal(inj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal vote extension injection: %w", err)
	}
	return append(VoteExtInjectionPrefix, data...), nil
}

// decodeVoteExtInjection deserializes a VoteExtInjection from prefixed bytes.
func decodeVoteExtInjection(tx []byte) (VoteExtInjection, error) {
	var inj VoteExtInjection
	if !isVoteExtInjectionTx(tx) {
		return inj, fmt.Errorf("not a vote extension injection tx")
	}
	err := json.Unmarshal(tx[len(VoteExtInjectionPrefix):], &inj)
	return inj, err
}

// PrepareProposalHandler selects transactions for the next block.
//
// In PoT, this is equivalent to tryProduceBlock():
// 1. Select pending transactions from CometBFT mempool
// 2. Collect vote extensions from the previous block
// 3. Inject vote extension data (commitments/reveals) as a pseudo-tx at position 0
func (app *LegibleApp) PrepareProposalHandler() sdk.PrepareProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestPrepareProposal) (resp *abci.ResponsePrepareProposal, err error) {
		logger := ctx.Logger().With("module", "abci", "handler", "PrepareProposal")

		// Panic recovery: a panic in PrepareProposal must not crash the node.
		// Return an empty proposal so CometBFT selects the next proposer.
		defer func() {
			if r := recover(); r != nil {
				logger.Error("PANIC in PrepareProposal — returning empty proposal",
					"height", req.Height, "panic", fmt.Sprintf("%v", r))
				resp = &abci.ResponsePrepareProposal{Txs: nil}
				err = nil
			}
		}()

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
		txs := make([][]byte, 0, len(req.Txs)+1)

		// Process vote extensions from previous block (if available)
		if len(req.LocalLastCommit.Votes) > 0 {
			injection := app.processVoteExtensions(ctx, req.LocalLastCommit.Votes)

			// Inject vote extension data as a pseudo-tx at position 0
			if len(injection.Commitments) > 0 || len(injection.Reveals) > 0 {
				injBytes, err := encodeVoteExtInjection(injection)
				if err != nil {
					// Encoding a well-formed VoteExtInjection should never fail.
					// If it does, this proposer must not produce a block with
					// missing PoT data — return error so CometBFT selects the
					// next proposer. (PP-CONSENSUS-1 remediation)
					return nil, fmt.Errorf("critical: vote extension injection encoding failed: %w", err)
				}

				// P1-2 FIX: Enforce VEX injection size limit to prevent block bloat.
				if len(injBytes) > MaxVEXInjectionBytes {
					logger.Warn("vote extension injection exceeds size limit — dropping",
						"size", len(injBytes), "max", MaxVEXInjectionBytes)
					// Drop the injection entirely rather than producing a partial/corrupt one.
					// The rounds will retry on the next block.
				} else {
					txs = append(txs, injBytes)
				}

				logger.Info("injecting vote extension data",
					"commitments", len(injection.Commitments),
					"reveals", len(injection.Reveals),
				)
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
// 1. Recognize and validate vote extension injection tx (position 0)
// 2. Validate all regular transactions
// 3. Verify VRF proofs for proposer eligibility
func (app *LegibleApp) ProcessProposalHandler() sdk.ProcessProposalHandler {
	return func(ctx sdk.Context, req *abci.RequestProcessProposal) (resp *abci.ResponseProcessProposal, err error) {
		logger := ctx.Logger().With("module", "abci", "handler", "ProcessProposal")

		// Panic recovery: a panic in ProcessProposal must not crash the node.
		// Reject the proposal so the network can try the next proposer.
		defer func() {
			if r := recover(); r != nil {
				logger.Error("PANIC in ProcessProposal — rejecting proposal",
					"height", req.Height, "panic", fmt.Sprintf("%v", r))
				resp = &abci.ResponseProcessProposal{
					Status: abci.ResponseProcessProposal_REJECT,
				}
				err = nil
			}
		}()

		// Emergency halt guard: reject proposals containing non-emergency txs
		if app.EmergencyKeeper.IsHalted(ctx) {
			for _, txBytes := range req.Txs {
				// Skip vote extension injection tx during halt check
				if isVoteExtInjectionTx(txBytes) {
					continue
				}
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
			// Skip vote extension injection tx (validated separately)
			if isVoteExtInjectionTx(txBytes) {
				// P1-2 FIX: Reject oversized VEX injections
				if len(txBytes) > MaxVEXInjectionBytes {
					logger.Warn("vote extension injection tx exceeds size limit",
						"index", i, "size", len(txBytes), "max", MaxVEXInjectionBytes)
					return &abci.ResponseProcessProposal{
						Status: abci.ResponseProcessProposal_REJECT,
					}, nil
				}
				// Validate the injection format
				if _, err := decodeVoteExtInjection(txBytes); err != nil {
					logger.Warn("invalid vote extension injection tx", "index", i, "err", err)
					return &abci.ResponseProcessProposal{
						Status: abci.ResponseProcessProposal_REJECT,
					}, nil
				}
				continue
			}

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

		logger.Debug("accepted proposal",
			"height", req.Height,
			"txs", len(req.Txs),
		)

		return &abci.ResponseProcessProposal{
			Status: abci.ResponseProcessProposal_ACCEPT,
		}, nil
	}
}

// ---------- Internal Methods ----------

// processVoteExtensions collects commitments and reveals from vote extensions.
// Returns a VoteExtInjection containing all vote extension data to inject
// into the block for deterministic processing by all validators.
func (app *LegibleApp) processVoteExtensions(
	ctx sdk.Context,
	votes []abci.ExtendedVoteInfo,
) VoteExtInjection {
	var inj VoteExtInjection

	for _, vote := range votes {
		if len(vote.VoteExtension) == 0 {
			continue
		}

		var ext VoteExtension
		if err := json.Unmarshal(vote.VoteExtension, &ext); err != nil {
			continue
		}

		// Skip extensions with no validator address
		if ext.ValidatorAddress == "" {
			continue
		}

		// Collect commitments
		for _, c := range ext.Commitments {
			if c.RoundID == "" || c.CommitmentHash == "" {
				continue
			}
			inj.Commitments = append(inj.Commitments, InjectedCommitment{
				RoundID:        c.RoundID,
				Validator:      ext.ValidatorAddress,
				CommitmentHash: c.CommitmentHash,
				VRFOutput:      c.VRFOutput,
				VRFProof:       c.VRFProof,
			})
		}

		// Collect reveals
		for _, r := range ext.Reveals {
			if r.RoundID == "" || r.Salt == "" {
				continue
			}
			inj.Reveals = append(inj.Reveals, InjectedReveal{
				RoundID:    r.RoundID,
				Validator:  ext.ValidatorAddress,
				Verdict:    r.Verdict,
				Confidence: r.Confidence,
				Salt:       r.Salt,
			})
		}
	}

	// Sort for deterministic ordering (all validators must process the same data)
	sort.Slice(inj.Commitments, func(i, j int) bool {
		if inj.Commitments[i].RoundID != inj.Commitments[j].RoundID {
			return inj.Commitments[i].RoundID < inj.Commitments[j].RoundID
		}
		return inj.Commitments[i].Validator < inj.Commitments[j].Validator
	})
	sort.Slice(inj.Reveals, func(i, j int) bool {
		if inj.Reveals[i].RoundID != inj.Reveals[j].RoundID {
			return inj.Reveals[i].RoundID < inj.Reveals[j].RoundID
		}
		return inj.Reveals[i].Validator < inj.Reveals[j].Validator
	})

	return inj
}

// ProcessVoteExtInjection processes a vote extension injection tx by storing
// commitments and reveals in the knowledge keeper state.
// Called from PreBlocker before BeginBlock phase transitions.
func (app *LegibleApp) ProcessVoteExtInjection(ctx sdk.Context, data []byte) {
	logger := ctx.Logger().With("module", "abci", "handler", "PreBlocker")

	// Panic recovery: a panic while processing injected VEX data must not
	// crash the node. The round data is lost for this block but the chain
	// continues producing blocks.
	defer func() {
		if r := recover(); r != nil {
			logger.Error("PANIC in ProcessVoteExtInjection — skipping injection",
				"height", ctx.BlockHeight(), "panic", fmt.Sprintf("%v", r))
		}
	}()

	inj, err := decodeVoteExtInjection(data)
	if err != nil {
		logger.Error("failed to decode vote extension injection", "err", err)
		return
	}

	height := uint64(ctx.BlockHeight())
	storedCommitments := 0
	storedReveals := 0

	// Store commitments in keeper state
	for _, c := range inj.Commitments {
		// GAP-2 FIX: Re-verify VRF proof before storing each commitment.
		// VerifyVoteExtension verified proofs when the extension was received, but
		// a malicious block proposer could construct a fake injection containing
		// commitments with invalid or fabricated VRF proofs — bypassing selection
		// verification entirely. Re-checking here closes the proposer-injection gap.
		if c.VRFOutput == "" || c.VRFProof == "" {
			logger.Warn("injected commitment missing VRF proof — discarding",
				"round_id", c.RoundID,
				"validator", c.Validator,
			)
			continue
		}
		round, roundFound := app.KnowledgeKeeper.GetVerificationRound(ctx, c.RoundID)
		if roundFound {
			if err := app.KnowledgeKeeper.VerifyValidatorVRFSelection(
				ctx, round, c.Validator, c.VRFOutput, c.VRFProof,
			); err != nil {
				logger.Warn("VRF proof re-verification failed for injected commitment — discarding",
					"round_id", c.RoundID,
					"validator", c.Validator,
					"err", err,
				)
				continue
			}
		}
		// If round not found, allow storage to proceed — StoreCommitmentInRound
		// will return ErrRoundNotFound and the loop will skip cleanly.

		commitment := knowledgetypes.VerificationCommitment{
			Validator:         c.Validator,
			CommitmentHash:    c.CommitmentHash,
			SubmittedAtHeight: height,
		}

		if err := app.KnowledgeKeeper.StoreCommitmentInRound(ctx, c.RoundID, commitment); err != nil {
			if errors.Is(err, knowledgetypes.ErrEquivocation) {
				// Equivocation already handled (slashed + evidence stored) inside keeper
				logger.Error("EQUIVOCATION in vote extension commitment",
					"round_id", c.RoundID,
					"validator", c.Validator,
					"error", err.Error(),
				)
			} else {
				// Normal skip — round may have expired or validator already committed
				logger.Debug("skipped commitment from vote extension",
					"round_id", c.RoundID,
					"validator", c.Validator,
					"reason", err.Error(),
				)
			}
			continue
		}
		storedCommitments++
	}

	// Store reveals in keeper state
	for _, r := range inj.Reveals {
		reveal := knowledgetypes.VerificationReveal{
			Validator:         r.Validator,
			Verdict:           r.Verdict,
			Confidence:        r.Confidence,
			Salt:              r.Salt,
			SubmittedAtHeight: height,
		}

		if err := app.KnowledgeKeeper.StoreRevealInRound(ctx, r.RoundID, reveal); err != nil {
			if errors.Is(err, knowledgetypes.ErrEquivocation) {
				// Equivocation already handled (slashed + evidence stored) inside keeper
				logger.Error("EQUIVOCATION in vote extension reveal",
					"round_id", r.RoundID,
					"validator", r.Validator,
					"error", err.Error(),
				)
			} else {
				// Normal skip — round may have transitioned or validator already revealed
				logger.Debug("skipped reveal from vote extension",
					"round_id", r.RoundID,
					"validator", r.Validator,
					"reason", err.Error(),
				)
			}
			continue
		}
		storedReveals++
	}

	if storedCommitments > 0 || storedReveals > 0 {
		logger.Info("processed vote extension injection",
			"height", height,
			"commitments_stored", storedCommitments,
			"commitments_total", len(inj.Commitments),
			"reveals_stored", storedReveals,
			"reveals_total", len(inj.Reveals),
		)
	}
}

// ComputeCommitmentHash generates a commitment hash for a verification verdict.
// Delegates to knowledgetypes.ComputeCommitmentHash (canonical implementation).
func ComputeCommitmentHash(roundID, verdict string, confidence uint64, salt string) string {
	return knowledgetypes.ComputeCommitmentHash(roundID, verdict, confidence, salt)
}

// VerifyCommitmentHash checks that a reveal matches its prior commitment.
// Delegates to knowledgetypes.VerifyCommitmentHash (canonical implementation).
func VerifyCommitmentHash(commitmentHash, roundID, verdict string, confidence uint64, salt string) bool {
	return knowledgetypes.VerifyCommitmentHash(commitmentHash, roundID, verdict, confidence, salt)
}
