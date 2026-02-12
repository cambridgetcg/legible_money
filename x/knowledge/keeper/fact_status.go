package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Confidence adjustment constants (shared with disputes module).
// All values in basis points (1_000_000 = 100%).
const (
	RetractionThresholdBps uint64 = 100_000   // 10% — below this, fact is revoked
	ConfidenceBoostBps     uint64 = 110_000   // 11% — upheld boost
	MaxConfidenceBps       uint64 = 1_000_000 // 100%
	CascadeReductionBps    uint64 = 220_000   // 22% — cascade reduction rate
)

// MarkFactChallenged transitions a fact's status to "challenged" when a dispute is opened.
// Only facts with status "verified" or "active" can be challenged.
func (k Keeper) MarkFactChallenged(ctx sdk.Context, factId string) error {
	fact, found := k.GetFact(ctx, factId)
	if !found {
		return fmt.Errorf("fact not found: %s", factId)
	}
	if fact.Status != "verified" && fact.Status != "active" {
		return fmt.Errorf("fact %s cannot be challenged: status is %s", factId, fact.Status)
	}

	fact.Status = "challenged"
	k.SetFact(ctx, fact)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"fact_challenged",
			sdk.NewAttribute("fact_id", factId),
			sdk.NewAttribute("previous_status", "verified"),
		),
	)

	return nil
}

// ResolveFactDispute adjusts a fact's confidence and status based on a dispute outcome.
// This encapsulates all invariants: confidence reduction/boost, retraction threshold,
// status transitions, and event emission.
//
// Parameters:
//   - factId: the fact being disputed
//   - outcome: "overturned", "upheld", or "inconclusive"
//   - reductionBps: for overturned — confidence reduction in basis points (e.g. 550_000 = 55%)
//
// Returns the new confidence value for the caller to use in events.
func (k Keeper) ResolveFactDispute(
	ctx sdk.Context,
	factId string,
	outcome string,
	reductionBps uint64,
) (newConfidence uint64, err error) {
	fact, found := k.GetFact(ctx, factId)
	if !found {
		return 0, fmt.Errorf("fact not found: %s", factId)
	}

	previousConfidence := fact.Confidence
	fact.ChallengeCount++

	switch outcome {
	case "overturned":
		fact.Confidence = fact.Confidence * (MaxConfidenceBps - reductionBps) / MaxConfidenceBps
		if fact.Confidence < RetractionThresholdBps {
			fact.Status = "revoked"
		} else {
			fact.Status = "contested"
		}

	case "upheld":
		fact.Confidence = fact.Confidence * (MaxConfidenceBps + ConfidenceBoostBps) / MaxConfidenceBps
		if fact.Confidence > MaxConfidenceBps {
			fact.Confidence = MaxConfidenceBps
		}
		if fact.Status == "contested" || fact.Status == "challenged" {
			fact.Status = "active"
		}

	case "inconclusive":
		// No change to confidence
		if fact.Status == "challenged" {
			fact.Status = "contested"
		}

	default:
		return 0, fmt.Errorf("unknown dispute outcome: %s", outcome)
	}

	k.SetFact(ctx, fact)

	// Emit confidence change event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"fact_confidence_changed",
			sdk.NewAttribute("fact_id", factId),
			sdk.NewAttribute("previous_confidence", fmt.Sprintf("%d", previousConfidence)),
			sdk.NewAttribute("new_confidence", fmt.Sprintf("%d", fact.Confidence)),
			sdk.NewAttribute("outcome", outcome),
			sdk.NewAttribute("challenge_count", fmt.Sprintf("%d", fact.ChallengeCount)),
		),
	)

	// Emit retraction event if fact was revoked
	if fact.Status == "revoked" {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"fact_retracted",
				sdk.NewAttribute("fact_id", factId),
				sdk.NewAttribute("final_confidence", fmt.Sprintf("%d", fact.Confidence)),
				sdk.NewAttribute("challenge_count", fmt.Sprintf("%d", fact.ChallengeCount)),
			),
		)
		k.Logger(ctx).Info("fact retracted via dispute",
			"fact_id", factId,
			"final_confidence", fact.Confidence,
		)
	}

	return fact.Confidence, nil
}

// CascadeFactConfidence reduces confidence of facts that reference a retracted fact.
// Uses a 22% cascade reduction (from TS CONFIDENCE_CASCADE_RATE).
// Returns the IDs of affected facts.
func (k Keeper) CascadeFactConfidence(
	ctx sdk.Context,
	retractedFactId string,
	reductionBps uint64,
) ([]string, error) {
	// Get facts that reference the retracted fact via incoming references
	refs := k.GetIncomingReferences(ctx, retractedFactId)
	if len(refs) == 0 {
		return nil, nil
	}

	var affected []string
	for _, ref := range refs {
		refFactId := ref.FromFactId
		fact, found := k.GetFact(ctx, refFactId)
		if !found {
			continue
		}
		if fact.Status == "revoked" {
			continue
		}

		previousConfidence := fact.Confidence
		fact.Confidence = fact.Confidence * (MaxConfidenceBps - reductionBps) / MaxConfidenceBps

		if fact.Confidence < RetractionThresholdBps {
			fact.Status = "revoked"
		} else if fact.Confidence < previousConfidence {
			fact.Status = "contested"
		}

		k.SetFact(ctx, fact)
		affected = append(affected, refFactId)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"fact_confidence_cascaded",
				sdk.NewAttribute("fact_id", refFactId),
				sdk.NewAttribute("retracted_reference", retractedFactId),
				sdk.NewAttribute("previous_confidence", fmt.Sprintf("%d", previousConfidence)),
				sdk.NewAttribute("new_confidence", fmt.Sprintf("%d", fact.Confidence)),
				sdk.NewAttribute("status", fact.Status),
			),
		)
	}

	if len(affected) > 0 {
		k.Logger(ctx).Info("confidence cascade applied",
			"retracted_fact", retractedFactId,
			"affected_facts", len(affected),
		)
	}

	return affected, nil
}
