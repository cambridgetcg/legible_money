package evaluation

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// evaluateLogicZone computes the zone confidence ceiling and checks for
// Godelian incompleteness requirements.
//
// Stage 4 of the evaluation pipeline.
func evaluateLogicZone(ctx sdk.Context, evalCtx *EvaluationContext, claim *types.Claim) StageResult {
	result := StageResult{
		Name: "logic_zone",
	}

	zone := claim.LogicZone

	// Get zone ceiling (0-1,000,000 scale, same as knowledge module)
	var zoneCeiling uint64
	if evalCtx.GetZoneConfidenceCeiling != nil && zone != "" {
		zoneCeiling = evalCtx.GetZoneConfidenceCeiling(ctx, zone)
	} else if zone == "" {
		zoneCeiling = 1_000_000 // default: 100%
	} else {
		// Zone specified but no keeper: use conservative default
		zoneCeiling = 700_000 // 70%
	}

	if zoneCeiling > 1_000_000 {
		zoneCeiling = 1_000_000
	}

	result.Ceiling = zoneCeiling
	result.Justification = "logic zone ceiling applied"

	// Godelian handling: zones where incompleteness applies
	if evalCtx.RequiresIncompletenessAck != nil && zone != "" {
		if evalCtx.RequiresIncompletenessAck(ctx, zone) {
			// G-2: Relevance-aware grounding — referenced fact must be:
			//   1. Verified or active
			//   2. In the same or a more formal (lower formality rank) zone
			//   3. In the same domain (if both have domains set)
			hasAxiomaticGrounding := false
			for _, refId := range claim.References {
				if evalCtx.GetFact != nil {
					fact, found := evalCtx.GetFact(ctx, refId)
					if !found || (fact.Status != types.FactStatusVerified &&
						fact.Status != types.FactStatusActive) {
						continue
					}

					// Zone relevance: referenced fact must be in same or more formal zone
					if !isZoneCompatibleGrounding(fact.LogicZone, zone) {
						continue
					}

					// Domain relevance: if both have domains, they must match
					if fact.Domain != "" && claim.Domain != "" && fact.Domain != claim.Domain {
						continue
					}

					hasAxiomaticGrounding = true
					break
				}
			}

			if !hasAxiomaticGrounding {
				result.ForceVerdict = "abstain"
				result.Justification = "Godel-zone claim with no relevant axiomatic grounding"
			}
		}
	}

	return result
}

// isZoneCompatibleGrounding returns true if a fact in refZone can ground a claim
// in claimZone. A fact in a more formal (lower formality rank) zone can ground
// claims in less formal zones, but not the reverse.
// Example: a propositional fact CAN ground a Peano claim (axioms ground theorems),
//
//	but an empirical fact CANNOT ground a Peano claim.
func isZoneCompatibleGrounding(refZone, claimZone string) bool {
	if refZone == "" || claimZone == "" {
		return true // no zone = no restriction (backward compatible)
	}
	refRank := zoneFormality(refZone)
	claimRank := zoneFormality(claimZone)
	if refRank < 0 || claimRank < 0 {
		return true // unknown zone = allow (forward compatible)
	}
	return refRank <= claimRank
}

// zoneFormality returns the formality rank of a logic zone string.
// Lower values = more formal. Matches the ordering in x/ontology/types.
func zoneFormality(zone string) int {
	switch zone {
	case "propositional":
		return 0
	case "presburger":
		return 1
	case "peano":
		return 2
	case "set_theory":
		return 3
	case "empirical":
		return 4
	default:
		return -1
	}
}
