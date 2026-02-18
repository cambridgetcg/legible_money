package evaluation

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// evaluateNovelty assesses the information gain and cross-domain bridge value
// of a claim. Inserted between fact_graph (stage 5) and logic_zone (stage 6).
//
// The novelty stage rewards claims that:
//  1. Introduce genuinely new information (high information gain)
//  2. Bridge verified knowledge across domains (cross-domain synthesis)
//
// Both are gated by evidence quality — claims with weak evidence get no novelty
// credit, preventing hallucinated "novel" claims from gaming the bonus.
func evaluateNovelty(ctx sdk.Context, evalCtx *EvaluationContext, claim *types.Claim, evidenceAdjustment int64) StageResult {
	result := StageResult{
		Name: "novelty",
	}

	// Gate: no novelty bonus without positive evidence quality.
	// The evidence stage returns a positive adjustment when valid evidence exists.
	// A zero or negative adjustment means no evidence or weak/rejected evidence.
	// This prevents hallucinated "novel" claims with no backing from earning bonus.
	if evidenceAdjustment <= 0 {
		result.Justification = fmt.Sprintf("evidence quality %d insufficient for novelty bonus", evidenceAdjustment)
		return result
	}

	var totalBonus int64
	var bridgeScore uint64

	// ========================================================================
	// 1. Information Gain: how much does this claim reduce uncertainty?
	// ========================================================================
	//
	// Information gain is highest when:
	//   - The claim is in a domain with few existing facts (sparse = more informative)
	//   - The claim references facts from distant domains (synthesis)
	//   - The claim introduces new connections between isolated clusters
	//
	// Formula: info_gain = domain_sparsity × reference_novelty
	//   domain_sparsity = 1.0 - (facts_in_domain / total_facts) clamped to [0.1, 1.0]
	//   reference_novelty = unique_referenced_domains / total_references

	infoGainBps := calculateInformationGain(ctx, evalCtx, claim)

	// ========================================================================
	// 2. Bridge Score: verified cross-domain references
	// ========================================================================
	//
	// Claims that cite well-verified facts from other domains get a bonus.
	// This rewards genuine interdisciplinary synthesis while the AH-3 penalty
	// still applies to unverified cross-domain refs.

	bridgeScore = calculateBridgeScore(ctx, evalCtx, claim)

	ep := evalCtx.ExtendedParams
	// Combined novelty = (info_gain + bridge_score) / 2, capped at NoveltyMaxBonusBps
	combined := (infoGainBps + bridgeScore) / 2
	if combined > ep.NoveltyMaxBonusBps {
		combined = ep.NoveltyMaxBonusBps
	}

	// FARM-7: Scale novelty bonus by domain contributor count.
	// Domains with fewer than MinDomainContributorsForNovelty unique contributors
	// get proportionally reduced novelty bonuses to prevent domain squatting.
	if evalCtx.GetDomainUniqueContributors != nil && claim.Domain != "" {
		contributors := evalCtx.GetDomainUniqueContributors(ctx, claim.Domain)
		if contributors < ep.MinDomainContributorsForNovelty {
			combined = (combined * contributors) / ep.MinDomainContributorsForNovelty
		}
	}

	totalBonus = int64(combined)

	result.Adjustment = totalBonus
	result.Justification = fmt.Sprintf(
		"novelty assessment: info_gain=%d bridge=%d combined=%d",
		infoGainBps, bridgeScore, combined,
	)

	return result
}

// calculateInformationGain measures how much new information a claim contributes.
// Returns a value in basis points (0-1,000,000).
func calculateInformationGain(ctx sdk.Context, evalCtx *EvaluationContext, claim *types.Claim) uint64 {
	// Domain sparsity: fewer facts in this domain = more information gain
	var domainSparsityBps uint64 = 1_000_000 // default: full sparsity if no domain context

	if evalCtx.GetFactCountByDomain != nil && evalCtx.GetDomainCount != nil && claim.Domain != "" {
		factsInDomain := evalCtx.GetFactCountByDomain(ctx, claim.Domain)
		totalDomains := evalCtx.GetDomainCount(ctx)

		if factsInDomain > 0 && totalDomains > 0 {
			// Average facts per domain as baseline
			// If this domain has fewer than average → high sparsity
			// Sparsity = 1.0 - min(factsInDomain, cap) / cap
			// Cap at 100 facts per domain to avoid zero sparsity for mature domains
			cap := uint64(100)
			bounded := factsInDomain
			if bounded > cap {
				bounded = cap
			}
			domainSparsityBps = 1_000_000 - (bounded * 1_000_000 / cap)
		}
	}

	// Floor at 10% — even well-populated domains can have novel claims
	if domainSparsityBps < 100_000 {
		domainSparsityBps = 100_000
	}

	// Reference novelty: how many unique domains are referenced?
	var referenceNoveltyBps uint64
	if len(claim.References) > 0 && evalCtx.GetFact != nil {
		domainSet := make(map[string]bool)
		for _, refId := range claim.References {
			fact, found := evalCtx.GetFact(ctx, refId)
			if found && fact.Domain != "" {
				domainSet[fact.Domain] = true
			}
		}
		uniqueDomains := uint64(len(domainSet))
		totalRefs := uint64(len(claim.References))

		if totalRefs > 0 {
			// Ratio of unique domains to total refs (cross-domain synthesis)
			referenceNoveltyBps = uniqueDomains * 1_000_000 / totalRefs
		}
	}

	// Combined: geometric mean of sparsity and reference novelty (or just sparsity if no refs)
	if referenceNoveltyBps > 0 {
		// Geometric mean: sqrt(sparsity * reference_novelty)
		product := float64(domainSparsityBps) * float64(referenceNoveltyBps)
		combined := uint64(math.Sqrt(product))
		return combined
	}

	// No references: halve the sparsity bonus (references prove cross-pollination)
	return domainSparsityBps / 2
}

// calculateBridgeScore computes the cross-domain bridge value of a claim.
// Only verified references (above VerifiedBridgeThreshold) from different domains count.
// Returns a value in basis points (0-1,000,000).
func calculateBridgeScore(ctx sdk.Context, evalCtx *EvaluationContext, claim *types.Claim) uint64 {
	if len(claim.References) == 0 || evalCtx.GetFact == nil || claim.Domain == "" {
		return 0
	}

	ep := evalCtx.ExtendedParams
	var bridgeBonus uint64

	for _, refId := range claim.References {
		if bridgeBonus >= ep.MaxBridgeBonusBps {
			break
		}

		refFact, found := evalCtx.GetFact(ctx, refId)
		if !found || refFact.Domain == "" || refFact.Domain == claim.Domain {
			continue
		}

		// Only well-verified cross-domain refs count as bridges
		if refFact.Confidence < ep.VerifiedBridgeThresholdBps {
			continue
		}

		// Verified bridge: bonus proportional to ref confidence
		// Scale: at threshold = base bonus, at 1M = 1.43× bonus
		bonus := ep.CrossDomainBridgeBonusBps * refFact.Confidence / ep.VerifiedBridgeThresholdBps
		if bonus > ep.CrossDomainBridgeBonusBps*3/2 {
			bonus = ep.CrossDomainBridgeBonusBps * 3 / 2 // cap at 1.5× per ref
		}

		if bridgeBonus+bonus > ep.MaxBridgeBonusBps {
			bonus = ep.MaxBridgeBonusBps - bridgeBonus
		}
		bridgeBonus += bonus
	}

	return bridgeBonus
}
