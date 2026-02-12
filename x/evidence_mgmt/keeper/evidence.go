package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/evidence_mgmt/types"
)

// SubmitEvidence validates and stores a new evidence submission.
func (k Keeper) SubmitEvidence(ctx sdk.Context, evidence types.Evidence) error {
	params := k.GetParams(ctx)

	// Validate that the claim exists in x/knowledge
	if k.knowledgeKeeper != nil && !k.knowledgeKeeper.HasClaim(ctx, evidence.ClaimId) {
		return fmt.Errorf("%w: %s", types.ErrClaimNotFound, evidence.ClaimId)
	}

	// Check max evidence per claim
	count := k.GetEvidenceCountForClaim(ctx, evidence.ClaimId)
	if count >= params.MaxEvidencePerClaim {
		return fmt.Errorf("%w: claim %s already has %d evidence submissions",
			types.ErrMaxEvidenceReached, evidence.ClaimId, count)
	}

	// Check for duplicate content hash within this claim
	existing := k.GetEvidenceByClaimId(ctx, evidence.ClaimId)
	for _, e := range existing {
		if e.ContentHash == evidence.ContentHash {
			return fmt.Errorf("%w: content hash %s already submitted for claim %s",
				types.ErrDuplicateEvidence, evidence.ContentHash, evidence.ClaimId)
		}
	}

	// Validate evidence type
	if !types.IsValidEvidenceType(types.EvidenceType(evidence.EvidenceType)) {
		return fmt.Errorf("%w: invalid evidence type %s",
			types.ErrInvalidEvidence, evidence.EvidenceType)
	}

	// Set initial state
	evidence.Status = string(types.EvidenceStatusSubmitted)
	evidence.AuditCount = 0
	evidence.Reliability = 0

	k.SetEvidence(ctx, evidence)

	// Write expiry index for efficient EndBlock scanning
	expiryHeight := evidence.SubmittedAt + params.AuditWindow
	k.SetExpiryIndex(ctx, expiryHeight, evidence.Id)

	k.Logger(ctx).Info("evidence submitted",
		"evidence_id", evidence.Id,
		"claim_id", evidence.ClaimId,
		"type", evidence.EvidenceType,
		"submitter", evidence.Submitter,
	)

	return nil
}

// UpdateEvidenceStatus updates the status of an evidence record.
func (k Keeper) UpdateEvidenceStatus(ctx sdk.Context, id string, status types.EvidenceStatus) error {
	evidence, found := k.GetEvidence(ctx, id)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrEvidenceNotFound, id)
	}

	evidence.Status = string(status)
	if status == types.EvidenceStatusVerified {
		evidence.VerifiedAt = uint64(ctx.BlockHeight())
	}

	k.SetEvidence(ctx, evidence)
	return nil
}

// ComputeReliability computes the reliability score for evidence from its audits.
// Returns the stake-weighted average of audit scores.
func (k Keeper) ComputeReliability(ctx sdk.Context, evidenceId string) (uint64, error) {
	evidence, found := k.GetEvidence(ctx, evidenceId)
	if !found {
		return 0, fmt.Errorf("%w: %s", types.ErrEvidenceNotFound, evidenceId)
	}

	audits := k.GetAuditsForEvidence(ctx, evidenceId)
	if len(audits) == 0 {
		return 0, nil
	}

	// Compute stake-weighted average score
	totalStake := new(big.Int)
	weightedSum := new(big.Int)

	for _, audit := range audits {
		if audit.Status == string(types.AuditStatusExpired) {
			continue
		}
		stake := new(big.Int)
		if _, ok := stake.SetString(audit.Stake, 10); !ok {
			continue
		}
		score := new(big.Int).SetUint64(audit.Score)
		contribution := new(big.Int).Mul(stake, score)
		weightedSum.Add(weightedSum, contribution)
		totalStake.Add(totalStake, stake)
	}

	if totalStake.Sign() == 0 {
		return 0, nil
	}

	// weighted_score = weightedSum / totalStake
	reliability := new(big.Int).Div(weightedSum, totalStake)

	// Update evidence record
	evidence.Reliability = reliability.Uint64()
	k.SetEvidence(ctx, evidence)

	return reliability.Uint64(), nil
}

// ResolveEvidence resolves evidence status based on audit consensus.
// Called when enough audits have been submitted and the audit window has closed.
func (k Keeper) ResolveEvidence(ctx sdk.Context, evidenceId string) error {
	evidence, found := k.GetEvidence(ctx, evidenceId)
	if !found {
		return fmt.Errorf("%w: %s", types.ErrEvidenceNotFound, evidenceId)
	}

	if evidence.Status != string(types.EvidenceStatusSubmitted) {
		return nil // Already resolved
	}

	params := k.GetParams(ctx)

	// Check if audit window has passed
	if uint64(ctx.BlockHeight()) < evidence.SubmittedAt+params.AuditWindow {
		return nil // Window still open
	}

	audits := k.GetAuditsForEvidence(ctx, evidenceId)
	if uint32(len(audits)) < params.MinAuditsForConsensus {
		// Not enough audits — expire the evidence
		evidence.Status = string(types.EvidenceStatusExpired)
		k.SetEvidence(ctx, evidence)
		k.RemoveExpiryIndex(ctx, evidence.SubmittedAt+params.AuditWindow, evidence.Id)
		return nil
	}

	// Compute consensus
	consensus, err := k.ComputeConsensus(ctx, evidenceId)
	if err != nil {
		return fmt.Errorf("failed to compute consensus for evidence %s: %w", evidenceId, err)
	}

	// Store consensus
	k.SetConsensus(ctx, consensus)

	// Resolve audits: reward accurate, slash inaccurate
	if err := k.ResolveAudits(ctx, evidenceId, consensus.WeightedScore); err != nil {
		return fmt.Errorf("failed to resolve audits for evidence %s: %w", evidenceId, err)
	}

	// Update evidence with reliability and status
	reliability := consensus.WeightedScore
	evidence.Reliability = reliability
	evidence.AuditCount = consensus.AuditCount

	// Threshold: if reliability > 500000 (50%), consider verified
	if reliability > 500000 {
		evidence.Status = string(types.EvidenceStatusVerified)
		evidence.VerifiedAt = uint64(ctx.BlockHeight())
	} else {
		evidence.Status = string(types.EvidenceStatusRejected)
	}

	k.SetEvidence(ctx, evidence)

	// Clean up expiry index — evidence is now resolved
	k.RemoveExpiryIndex(ctx, evidence.SubmittedAt+params.AuditWindow, evidence.Id)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"evidence_resolved",
			sdk.NewAttribute("evidence_id", evidenceId),
			sdk.NewAttribute("status", string(evidence.Status)),
			sdk.NewAttribute("reliability", fmt.Sprintf("%d", reliability)),
			sdk.NewAttribute("audit_count", fmt.Sprintf("%d", consensus.AuditCount)),
		),
	)

	k.Logger(ctx).Info("evidence resolved",
		"evidence_id", evidenceId,
		"status", evidence.Status,
		"reliability", reliability,
	)

	return nil
}

// ExpireStaleEvidence resolves evidence whose audit window has closed.
// Uses the expiry index for O(expired) scanning instead of iterating all evidence.
func (k Keeper) ExpireStaleEvidence(ctx sdk.Context) {
	currentHeight := uint64(ctx.BlockHeight())

	// Collect IDs first to avoid mutating the index during iteration
	var expiredIds []string
	k.IterateExpiredEvidence(ctx, currentHeight, func(evidenceId string) bool {
		expiredIds = append(expiredIds, evidenceId)
		return false
	})

	for _, id := range expiredIds {
		if err := k.ResolveEvidence(ctx, id); err != nil {
			k.Logger(ctx).Error("failed to resolve stale evidence",
				"evidence_id", id, "error", err)
		}
	}
}
