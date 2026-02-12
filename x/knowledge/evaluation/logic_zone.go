package evaluation

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// defaultZoneCeiling is used when no zone is specified or the zone is unknown.
// 700,000 = 70% confidence ceiling (matches empirical zone).
const defaultZoneCeiling = 700_000

// evaluateLogicZone computes the zone confidence ceiling and checks for
// Godelian incompleteness requirements.
//
// Stage 4 of the evaluation pipeline.
func evaluateLogicZone(ctx sdk.Context, evalCtx *EvaluationContext, claim types.Claim) StageResult {
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
			// Check if claim has any references to established facts
			hasAxiomaticGrounding := false
			for _, refId := range claim.References {
				if evalCtx.GetFact != nil {
					fact, found := evalCtx.GetFact(ctx, refId)
					if found && (fact.Status == types.FactStatusVerified ||
						fact.Status == types.FactStatusActive) {
						hasAxiomaticGrounding = true
						break
					}
				}
			}

			if !hasAxiomaticGrounding {
				result.ForceVerdict = "abstain"
				result.Justification = "Godel-zone claim with no axiomatic grounding"
			}
		}
	}

	return result
}
