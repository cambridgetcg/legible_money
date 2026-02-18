package app

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	vrfcrypto "github.com/legible-money/legible/x/knowledge/crypto"
	"github.com/legible-money/legible/x/knowledge/evaluation"
	knowledgekeeper "github.com/legible-money/legible/x/knowledge/keeper"
	knowledgetypes "github.com/legible-money/legible/x/knowledge/types"
)

// VoteExtensionConfig holds per-validator configuration for vote extensions.
type VoteExtensionConfig struct {
	// ValidatorAddress is the validator's bech32 operator address.
	ValidatorAddress string

	// ValidatorPrivateKey is the Ed25519 private key for VRF generation.
	// This is the 32-byte seed or 64-byte full private key.
	ValidatorPrivateKey []byte

	// LocalStore holds commitment salts between commit and reveal phases.
	LocalStore *LocalCommitmentStore
}

// ExtendVoteHandler creates the ABCI++ ExtendVote handler that attaches
// PoT verification data to validator votes.
//
// For each active verification round:
//   - Commit phase: Generate VRF, check selection, create commitment hash
//   - Reveal phase: Retrieve local salt, create reveal
func ExtendVoteHandler(
	knowledgeKeeper knowledgekeeper.Keeper,
	config *VoteExtensionConfig,
) sdk.ExtendVoteHandler {
	return func(ctx sdk.Context, req *abci.RequestExtendVote) (resp *abci.ResponseExtendVote, err error) {
		logger := ctx.Logger().With("module", "abci", "handler", "ExtendVote")

		// Panic recovery: a panic in ExtendVote must not crash the validator.
		// Return an empty extension so the vote can still be cast.
		defer func() {
			if r := recover(); r != nil {
				logger.Error("PANIC in ExtendVote — returning empty extension",
					"height", req.Height, "panic", fmt.Sprintf("%v", r))
				resp = &abci.ResponseExtendVote{}
				err = nil
			}
		}()

		ext := VoteExtension{
			ValidatorAddress: config.ValidatorAddress,
		}

		// Skip if not configured (e.g., non-validator node)
		if config.ValidatorAddress == "" || len(config.ValidatorPrivateKey) == 0 {
			return emptyVoteExtension()
		}

		currentHeight := uint64(ctx.BlockHeight())

		// Get active verification rounds
		activeRounds := knowledgeKeeper.GetActiveRoundsForHeight(ctx, currentHeight)

		// In adversarial mode, only process challenge-type rounds via vote extensions.
		// Legacy verification rounds are handled by the deterministic BeginBlocker.
		params := knowledgeKeeper.GetParams(ctx)
		adversarialEnabled := params.AdversarialVerificationEnabled

		for _, round := range activeRounds {
			// Skip non-challenge rounds in adversarial mode
			if adversarialEnabled && round.Type != knowledgetypes.RoundTypeChallenge {
				continue
			}

			switch round.Phase {
			case "commit":
				commitment, err := handleCommitPhase(ctx, knowledgeKeeper, config, round)
				if err != nil {
					logger.Debug("skipping commit for round",
						"round_id", round.Id, "reason", err.Error())
					continue
				}
				if commitment != nil {
					ext.Commitments = append(ext.Commitments, *commitment)
				}

			case "reveal":
				reveal, err := handleRevealPhase(config, round)
				if err != nil {
					logger.Debug("skipping reveal for round",
						"round_id", round.Id, "reason", err.Error())
					continue
				}
				if reveal != nil {
					ext.Reveals = append(ext.Reveals, *reveal)
				}
			}
		}

		// Clean up expired local commitments
		config.LocalStore.CleanupExpired(currentHeight, 100)

		// Serialize
		bz, err := json.Marshal(ext)
		if err != nil {
			logger.Error("failed to marshal vote extension", "err", err)
			return emptyVoteExtension()
		}

		logger.Debug("extended vote",
			"height", currentHeight,
			"commitments", len(ext.Commitments),
			"reveals", len(ext.Reveals),
		)

		return &abci.ResponseExtendVote{VoteExtension: bz}, nil
	}
}

// handleCommitPhase generates a commitment for a round in commit phase.
// Returns nil if the validator is not selected via VRF for this round.
//
// P0-1 FIX: If a commitment already exists in the local store for this round,
// re-use it to prevent false equivocation slashing. Without this check, a
// validator selected in consecutive blocks would generate different salts,
// producing different commitment hashes that StoreCommitmentInRound treats
// as equivocation.
func handleCommitPhase(
	ctx sdk.Context,
	keeper knowledgekeeper.Keeper,
	config *VoteExtensionConfig,
	round knowledgetypes.VerificationRound,
) (*VoteCommitment, error) {
	// Generate VRF to check if this validator is selected
	prevBlockHash := getPrevBlockHashForVote(ctx)
	vrfOutput, vrfProof, err := knowledgekeeper.GenerateValidatorVRF(
		config.ValidatorPrivateKey,
		round.ClaimId,
		round.StartHeight,
		prevBlockHash,
	)
	if err != nil {
		return nil, fmt.Errorf("VRF generation failed: %w", err)
	}

	// Get this validator's effective stake (must match VerifyValidatorVRFSelection)
	totalStake := keeper.GetTotalStake(ctx)
	effectiveStake := keeper.GetStakingKeeper().GetEffectiveStake(ctx, config.ValidatorAddress)
	if effectiveStake == 0 {
		return nil, fmt.Errorf("validator %s not found or has zero stake", config.ValidatorAddress)
	}

	params := keeper.GetParams(ctx)

	// Check if we're selected
	selected, _ := vrfcrypto.IsValidatorSelected(
		vrfOutput,
		effectiveStake,
		totalStake,
		params.MinVerifiers,
	)

	if !selected {
		return nil, fmt.Errorf("not selected for round %s", round.Id)
	}

	// P0-1 FIX: Re-use existing commitment to prevent equivocation.
	// If we already generated a commitment for this round (in a prior block),
	// re-emit the same hash so StoreCommitmentInRound sees an idempotent
	// duplicate rather than a conflicting commitment.
	if existing, found := config.LocalStore.Get(round.Id); found {
		commitmentHash := ComputeCommitmentHash(round.Id, existing.Verdict, existing.Confidence, existing.Salt)
		return &VoteCommitment{
			RoundID:        round.Id,
			CommitmentHash: commitmentHash,
			VRFOutput:      hex.EncodeToString(vrfOutput),
			VRFProof:       hex.EncodeToString(vrfProof),
			Height:         uint64(ctx.BlockHeight()),
		}, nil
	}

	// Evaluate claim via the deterministic evaluation engine.
	// All validators produce identical results for the same on-chain state.
	var verdict string
	var confidence uint64

	claim, claimFound := keeper.GetClaim(ctx, round.ClaimId)
	if !claimFound {
		verdict = "abstain"
		confidence = 0
	} else {
		evalCtx := NewEvaluationContextFromKeeper(keeper)
		evalResult, err := evaluation.EvaluateClaim(ctx, evalCtx, round, claim)
		if err != nil {
			verdict = "abstain"
			confidence = 0
		} else {
			verdict = evalResult.Verdict
			confidence = evalResult.Confidence
		}
	}

	// Generate random salt
	saltBytes := make([]byte, 16)
	if _, err := rand.Read(saltBytes); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	salt := hex.EncodeToString(saltBytes)

	// Compute commitment hash
	commitmentHash := ComputeCommitmentHash(round.Id, verdict, confidence, salt)

	// Store locally for reveal phase
	config.LocalStore.Store(LocalCommitment{
		RoundID:    round.Id,
		Verdict:    verdict,
		Confidence: confidence,
		Salt:       salt,
		Height:     uint64(ctx.BlockHeight()),
	})

	return &VoteCommitment{
		RoundID:        round.Id,
		CommitmentHash: commitmentHash,
		VRFOutput:      hex.EncodeToString(vrfOutput),
		VRFProof:       hex.EncodeToString(vrfProof),
		Height:         uint64(ctx.BlockHeight()),
	}, nil
}

// handleRevealPhase generates a reveal for a round in reveal phase.
// Returns nil if no local commitment exists (validator didn't commit).
func handleRevealPhase(
	config *VoteExtensionConfig,
	round knowledgetypes.VerificationRound,
) (*VoteReveal, error) {
	// Retrieve local commitment
	localCommit, found := config.LocalStore.Get(round.Id)
	if !found {
		return nil, fmt.Errorf("no local commitment for round %s", round.Id)
	}

	return &VoteReveal{
		RoundID:    round.Id,
		Verdict:    localCommit.Verdict,
		Confidence: localCommit.Confidence,
		Salt:       localCommit.Salt,
	}, nil
}

// VerifyVoteExtensionHandler creates the ABCI++ VerifyVoteExtension handler
// that validates PoT vote extensions from other validators.
//
// Validates:
//   - Structural validity (well-formed JSON)
//   - VRF proof validity and selection (for commitments)
//   - Reveal-matches-commitment (for reveals)
//   - Validator eligibility
func VerifyVoteExtensionHandler(
	keeper knowledgekeeper.Keeper,
) sdk.VerifyVoteExtensionHandler {
	return func(ctx sdk.Context, req *abci.RequestVerifyVoteExtension) (resp *abci.ResponseVerifyVoteExtension, err error) {
		logger := ctx.Logger().With("module", "abci", "handler", "VerifyVoteExtension")

		// Panic recovery: a panic in VerifyVoteExtension must not crash the node.
		// Accept the extension to avoid halting consensus over a local error.
		defer func() {
			if r := recover(); r != nil {
				logger.Error("PANIC in VerifyVoteExtension — accepting extension to preserve liveness",
					"height", req.Height, "panic", fmt.Sprintf("%v", r))
				resp = acceptExtension()
				err = nil
			}
		}()

		// Empty extensions are always valid
		if len(req.VoteExtension) == 0 {
			return acceptExtension(), nil
		}

		// Deserialize
		var ext VoteExtension
		if err := json.Unmarshal(req.VoteExtension, &ext); err != nil {
			logger.Warn("invalid vote extension format", "err", err)
			return rejectExtension(), nil
		}

		// Reject oversized extensions (prevent DoS via unbounded VRF verification)
		const MaxCommitmentsPerExtension = 50
		const MaxRevealsPerExtension = 50
		if len(ext.Commitments) > MaxCommitmentsPerExtension {
			logger.Warn("too many commitments in vote extension",
				"count", len(ext.Commitments), "max", MaxCommitmentsPerExtension)
			return rejectExtension(), nil
		}
		if len(ext.Reveals) > MaxRevealsPerExtension {
			logger.Warn("too many reveals in vote extension",
				"count", len(ext.Reveals), "max", MaxRevealsPerExtension)
			return rejectExtension(), nil
		}

		// Validate commitments
		for _, c := range ext.Commitments {
			if c.RoundID == "" || c.CommitmentHash == "" {
				return rejectExtension(), nil
			}
			if len(c.CommitmentHash) != 64 {
				return rejectExtension(), nil
			}

			// GAP-1 FIX: VRF proof is now REQUIRED for all commitments.
			// Previously "if present" was optional, allowing unverified commitments
			// from validators who skipped VRF proof generation. Without a VRF proof
			// the protocol cannot verify that the validator was legitimately selected.
			if c.VRFOutput == "" || c.VRFProof == "" {
				logger.Warn("commitment missing VRF proof — rejecting extension",
					"round_id", c.RoundID,
					"validator", ext.ValidatorAddress,
				)
				return rejectExtension(), nil
			}

			round, found := keeper.GetVerificationRound(ctx, c.RoundID)
			if !found {
				// Round may have been cleaned up between extension creation and verification.
				// Accept with benefit of doubt — the round cannot be verified.
				logger.Debug("round not found for commitment — accepting (may be cleaned up)",
					"round_id", c.RoundID)
				continue
			}

			err := keeper.VerifyValidatorVRFSelection(
				ctx, round, ext.ValidatorAddress, c.VRFOutput, c.VRFProof,
			)
			if err != nil {
				logger.Warn("VRF verification failed",
					"round_id", c.RoundID,
					"validator", ext.ValidatorAddress,
					"err", err,
				)
				return rejectExtension(), nil
			}
		}

		// Validate reveals
		for _, r := range ext.Reveals {
			if r.RoundID == "" || r.Salt == "" {
				return rejectExtension(), nil
			}

			switch r.Verdict {
			case "accept", "reject", "abstain":
			default:
				return rejectExtension(), nil
			}

			if r.Confidence > 1_000_000 {
				return rejectExtension(), nil
			}

			// Verify reveal matches prior commitment
			round, found := keeper.GetVerificationRound(ctx, r.RoundID)
			if found {
				commitment, hasCommitment := round.Commitments[ext.ValidatorAddress]
				if hasCommitment {
					expectedHash := ComputeCommitmentHash(r.RoundID, r.Verdict, r.Confidence, r.Salt)
					if expectedHash != commitment.CommitmentHash {
						logger.Warn("reveal does not match commitment",
							"round_id", r.RoundID,
							"validator", ext.ValidatorAddress,
						)
						return rejectExtension(), nil
					}
				}
			}
		}

		logger.Debug("verified vote extension",
			"height", req.Height,
			"commitments", len(ext.Commitments),
			"reveals", len(ext.Reveals),
		)

		return acceptExtension(), nil
	}
}

// ---- Helpers ----

func emptyVoteExtension() (*abci.ResponseExtendVote, error) {
	return &abci.ResponseExtendVote{}, nil
}

func acceptExtension() *abci.ResponseVerifyVoteExtension {
	return &abci.ResponseVerifyVoteExtension{
		Status: abci.ResponseVerifyVoteExtension_ACCEPT,
	}
}

func rejectExtension() *abci.ResponseVerifyVoteExtension {
	return &abci.ResponseVerifyVoteExtension{
		Status: abci.ResponseVerifyVoteExtension_REJECT,
	}
}

// getPrevBlockHashForVote returns the previous block's hash for VRF seed generation.
// Under CometBFT's instant finality, LastBlockId.Hash is deterministic once committed
// and cannot change due to reorgs. See crypto/vrf.go:GenerateVRFSeed for fork-sensitivity
// analysis if porting to a non-instant-finality consensus engine.
func getPrevBlockHashForVote(ctx sdk.Context) []byte {
	header := ctx.BlockHeader()
	if len(header.LastBlockId.Hash) > 0 {
		return header.LastBlockId.Hash
	}
	return make([]byte, 32)
}

// NewEvaluationContextFromKeeper wires keeper methods into the evaluation
// engine's context interface. All functions are nil-safe: if a sub-keeper
// is not wired, the evaluation engine uses conservative defaults.
func NewEvaluationContextFromKeeper(k knowledgekeeper.Keeper) *evaluation.EvaluationContext {
	ctx := &evaluation.EvaluationContext{
		GetClaim:                k.GetClaim,
		GetFact:                 k.GetFact,
		GetSupportingFacts:      k.GetSupportingFacts,
		GetContradictingFacts:   k.GetContradictingFacts,
		GetSubmitterCalibration: k.GetCalibrationRecord,
	}

	// Wire evidence keeper (nil-safe)
	if emk := k.GetEvidenceMgmtKeeper(); emk != nil {
		ctx.GetEvidenceByClaimId = emk.GetEvidenceByClaimId
	}

	// Wire ontology keeper (nil-safe)
	if ok := k.GetOntologyKeeper(); ok != nil {
		ctx.GetZoneConfidenceCeiling = ok.GetZoneConfidenceCeiling
		ctx.RequiresIncompletenessAck = ok.RequiresIncompletenessAck
		ctx.GetConfidenceCeiling = ok.GetConfidenceCeiling
	}

	// Wire negative knowledge lookups (Stage 8: counter-fact analysis)
	ctx.GetCounterFactsByFact = k.GetCounterFactsByFact
	ctx.GetBidirectionalConfidence = k.GetBidirectionalConfidence

	return ctx
}
