package evaluation

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// evaluateCounterFacts checks if any references in the claim point to facts
// that have verified counter-facts (negative knowledge).
//
// Stage 8 of the evaluation pipeline. Penalizes claims that build upon
// knowledge that has been partially or fully disproven.
//
// Penalty tiers (per counter-fact):
//   - High confidence (>= 700k): -80,000
//   - Medium confidence (>= 400k): -40,000
//   - Low confidence (< 400k): -10,000
//
// Total penalty capped at MaxCounterFactPenalty (400,000).
//
// Additionally:
//   - Claims referencing a fact whose bidirectional status is "disproven"
//     receive a forced "reject" verdict.
//   - Claims submitted about facts that are already fully disproven
//     receive the AlreadyDisprovenPenalty (200,000).
func evaluateCounterFacts(ctx sdk.Context, evalCtx *EvaluationContext, claim *types.Claim) StageResult {
	result := StageResult{
		Name: "counter_facts",
	}

	if evalCtx.GetCounterFactsByFact == nil {
		result.Justification = "counter-fact lookup not available"
		return result
	}

	ep := evalCtx.ExtendedParams
	// Check if this claim's own target is already known to be false.
	// This applies when a claim references a fact that is itself disproven.
	var totalPenalty uint64
	var disprovenRefFound bool

	for _, refId := range claim.References {
		if totalPenalty >= ep.MaxCounterFactPenalty {
			break
		}

		counterFacts := evalCtx.GetCounterFactsByFact(ctx, refId)
		if len(counterFacts) == 0 {
			continue
		}

		for _, cf := range counterFacts {
			if cf.Status != types.CounterFactStatusVerified {
				continue
			}

			var penalty uint64
			if cf.Confidence >= ep.CounterFactHighConfidenceBps {
				penalty = ep.CounterFactHighPenalty
			} else if cf.Confidence >= ep.CounterFactMediumConfidenceBps {
				penalty = ep.CounterFactMediumPenalty
			} else {
				penalty = ep.CounterFactLowPenalty
			}

			if totalPenalty+penalty > ep.MaxCounterFactPenalty {
				penalty = ep.MaxCounterFactPenalty - totalPenalty
			}
			totalPenalty += penalty
		}

		// Check bidirectional status: if any referenced fact is "disproven",
		// force rejection — the claim builds on false foundations.
		if evalCtx.GetBidirectionalConfidence != nil {
			bidir := evalCtx.GetBidirectionalConfidence(ctx, refId)
			if bidir != nil && bidir.Status == "disproven" {
				disprovenRefFound = true
			}
		}
	}

	if disprovenRefFound {
		result.ForceVerdict = "reject"
		result.Justification = "claim references a disproven fact (bidirectional confidence status = disproven)"
		result.Adjustment = -int64(totalPenalty)
		return result
	}

	result.Adjustment = -int64(totalPenalty)

	if totalPenalty > 0 {
		result.Justification = "counter-fact penalties applied to referenced facts"
	} else {
		result.Justification = "no counter-facts found on referenced facts"
	}

	return result
}
