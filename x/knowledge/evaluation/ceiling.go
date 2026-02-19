package evaluation

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// applyCeilings computes the effective ceiling as the minimum of three
// independent ceilings: zone, stratum, and category.
//
// Stage 5 of the evaluation pipeline.
func applyCeilings(ctx sdk.Context, evalCtx *EvaluationContext, claim *types.Claim, zoneCeiling uint64) StageResult {
	result := StageResult{
		Name: "ceiling_application",
	}

	minCeiling := uint64(1_000_000) // start at max

	// 1. Zone ceiling (from Stage 4, already in 0-1,000,000 scale)
	if zoneCeiling > 0 && zoneCeiling < minCeiling {
		minCeiling = zoneCeiling
	}

	// 2. Stratum ceiling (from ontology module, already 0-1,000,000 scale)
	stratumCeiling := getStratumCeiling(ctx, evalCtx, claim.Domain)
	if stratumCeiling > 0 && stratumCeiling < minCeiling {
		minCeiling = stratumCeiling
	}

	// 3. Category ceiling (from category decay profile MaxConfidence)
	if profile, ok := evalCtx.ExtendedParams.GetCategoryDecayProfile(claim.EpistemicCategory); ok {
		if profile.MaxConfidence > 0 && profile.MaxConfidence < minCeiling {
			minCeiling = profile.MaxConfidence
		}
	}

	result.Ceiling = minCeiling
	result.Justification = "triple ceiling: zone, stratum, category"

	return result
}

// getStratumCeiling queries the ontology module for the domain's stratum ceiling.
// Falls back to the params-based stratum confidence ceiling map.
func getStratumCeiling(ctx sdk.Context, evalCtx *EvaluationContext, domain string) uint64 {
	if evalCtx.GetConfidenceCeiling != nil && domain != "" {
		maxConf, _, err := evalCtx.GetConfidenceCeiling(ctx, domain)
		if err == nil && maxConf > 0 {
			return maxConf
		}
	}

	// Fallback: infer stratum from domain name
	stratum := inferStratum(domain)
	if ceiling, ok := evalCtx.ExtendedParams.GetStratumConfidenceCeiling(stratum); ok {
		return ceiling
	}

	return 1_000_000 // no ceiling
}

// inferStratum maps known domain names to their ontological stratum.
// Mirrors the logic in x/knowledge/keeper/confidence.go:inferDomainStratumHardcoded.
// FARM-12: default changed from "technological" to "social" to prevent ceiling abuse.
func inferStratum(domain string) string {
	switch domain {
	case "mathematics", "logic", "formal_systems":
		return "fundamental"
	case "physics", "astronomy", "mechanics":
		return "physical"
	case "chemistry", "materials_science":
		return "chemical"
	case "biology", "genetics", "medicine":
		return "biological"
	case "cognitive_science", "neuroscience", "psychology":
		return "cognitive"
	case "sociology", "economics", "political_science":
		return "social"
	case "computer_science", "engineering", "technology":
		return "technological"
	default:
		return "social" // FARM-12: default to most restrictive ceiling
	}
}
