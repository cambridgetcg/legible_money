package evaluation

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

const (
	supportBonus            = 30_000  // per supporting verified reference
	maxSupportBonus         = 150_000 // cap on total support bonus
	discreditPenalty        = 50_000  // per disproven/revoked reference
	maxDiscreditPenalty     = 300_000 // cap on total discredit penalty
	brokenRefPenalty        = 10_000  // per nonexistent reference
	contradictionPenalty    = 50_000  // per high-confidence contradiction
	maxContradictionPenalty = 250_000 // cap on total contradiction penalty

	// contradictionHighConf is the threshold above which a contradicting
	// fact significantly penalizes the claim.
	contradictionHighConf = 700_000

	// maxBFSDepth for circular reference detection.
	// ZD-K01 FIX: Increased from 5 to 10 — shallow depth allowed attackers
	// to create cycles at depth 6+ that evaded detection.
	maxBFSDepth = 10

	// Domain coherence penalties (AH-3: anti-hallucination)
	// Applied when references come from domains unrelated to the claim's domain.
	crossDomainSameStratumPenalty = 5_000   // minor: same stratum, different domain
	crossDomainDiffStratumPenalty = 20_000  // major: different stratum entirely
	maxDomainIncoherencePenalty   = 100_000 // cap on total domain coherence penalty
)

// evaluateFactGraph analyzes the claim's references against the fact graph.
// Stage 3 of the evaluation pipeline.
func evaluateFactGraph(ctx sdk.Context, evalCtx *EvaluationContext, claim types.Claim) StageResult {
	result := StageResult{
		Name: "fact_graph",
	}

	if len(claim.References) == 0 {
		result.Justification = "no references to evaluate"
		return result
	}

	var totalAdjustment int64
	var supportTotal uint64
	var discreditTotal uint64
	var contradictionTotal uint64

	for _, refId := range claim.References {
		fact, found := evalCtx.GetFact(ctx, refId)
		if !found {
			// Broken reference
			totalAdjustment -= int64(brokenRefPenalty)
			continue
		}

		switch fact.Status {
		case types.FactStatusVerified, types.FactStatusActive:
			// Supporting reference
			if supportTotal < maxSupportBonus {
				bonus := supportBonus
				if supportTotal+uint64(bonus) > maxSupportBonus {
					bonus = int(maxSupportBonus - supportTotal)
				}
				supportTotal += uint64(bonus)
				totalAdjustment += int64(bonus)
			}
		case types.FactStatusDisproven, types.FactStatusRevoked:
			// Discredited reference
			if discreditTotal < maxDiscreditPenalty {
				penalty := discreditPenalty
				if discreditTotal+uint64(penalty) > maxDiscreditPenalty {
					penalty = int(maxDiscreditPenalty - discreditTotal)
				}
				discreditTotal += uint64(penalty)
				totalAdjustment -= int64(penalty)
			}
		}

		// Check for high-confidence contradictions (capped at maxContradictionPenalty)
		if evalCtx.GetContradictingFacts != nil && contradictionTotal < maxContradictionPenalty {
			contradictions := evalCtx.GetContradictingFacts(ctx, refId)
			for _, contra := range contradictions {
				if contradictionTotal >= maxContradictionPenalty {
					break
				}
				contraFact, contraFound := evalCtx.GetFact(ctx, contra.FromFactId)
				if contraFound && contraFact.Confidence > contradictionHighConf {
					penalty := uint64(contradictionPenalty)
					if contradictionTotal+penalty > maxContradictionPenalty {
						penalty = maxContradictionPenalty - contradictionTotal
					}
					contradictionTotal += penalty
					totalAdjustment -= int64(penalty)
				}
			}
		}
	}

	// Domain coherence check (AH-3): penalize references from unrelated domains,
	// BUT reward verified cross-domain bridges (novelty-aware AH-3).
	// Unverified cross-domain refs are suspicious (hallucination). Well-verified ones
	// from different domains represent genuine interdisciplinary synthesis.
	ep := evalCtx.ExtendedParams
	if claim.Domain != "" && evalCtx.GetFact != nil {
		var incoherencePenalty uint64
		for _, refId := range claim.References {
			if incoherencePenalty >= maxDomainIncoherencePenalty {
				break
			}
			refFact, found := evalCtx.GetFact(ctx, refId)
			if !found || refFact.Domain == "" || refFact.Domain == claim.Domain {
				continue
			}

			// Verified bridge: high-confidence cross-domain ref → skip penalty.
			// The novelty stage (stage 6) awards a positive bonus for these.
			// Here we just avoid penalizing what's actually valuable synthesis.
			if refFact.Confidence >= ep.VerifiedBridgeThresholdBps &&
				(refFact.Status == types.FactStatusVerified || refFact.Status == types.FactStatusActive) {
				continue
			}

			// Unverified cross-domain ref: apply AH-3 penalty as before
			claimStratum := inferStratum(claim.Domain)
			refStratum := inferStratum(refFact.Domain)
			var penalty uint64
			if claimStratum == refStratum {
				penalty = crossDomainSameStratumPenalty
			} else {
				penalty = crossDomainDiffStratumPenalty
			}
			if incoherencePenalty+penalty > maxDomainIncoherencePenalty {
				penalty = maxDomainIncoherencePenalty - incoherencePenalty
			}
			incoherencePenalty += penalty
		}
		totalAdjustment -= int64(incoherencePenalty)
	}

	// Circular reference detection via bounded BFS
	if detectCycle(ctx, evalCtx, claim.References) {
		result.Justification = "circular reference detected in fact graph"
		if totalAdjustment > 0 {
			// Apply cycle penalty factor: 40% reduction
			// CyclePenaltyFactor = 400,000 means multiply by 0.4 (i.e., lose 60%)
			cyclePenalty := int64(uint64(totalAdjustment) * (1_000_000 - ep.CyclePenaltyFactor) / 1_000_000)
			totalAdjustment -= cyclePenalty
		} else {
			// Negative or zero adjustment with a cycle: apply a fixed penalty
			// so circular references always have a cost, even when the graph
			// already penalizes the claim.
			totalAdjustment -= int64(brokenRefPenalty)
		}
	}

	result.Adjustment = totalAdjustment
	if result.Justification == "" {
		result.Justification = "fact graph analysis complete"
	}

	return result
}

// detectCycle performs bounded BFS to check for circular references.
// Returns true if any referenced fact ID appears in its own transitive references.
func detectCycle(ctx sdk.Context, evalCtx *EvaluationContext, references []string) bool {
	if evalCtx.GetFact == nil {
		return false
	}

	// BFS from each referenced fact
	for _, startId := range references {
		visited := make(map[string]bool)
		visited[startId] = true

		queue := []bfsEntry{{id: startId, depth: 0}}

		for len(queue) > 0 {
			current := queue[0]
			queue = queue[1:]

			if current.depth >= maxBFSDepth {
				continue
			}

			fact, found := evalCtx.GetFact(ctx, current.id)
			if !found {
				continue
			}

			for _, nextId := range fact.References {
				if visited[nextId] {
					// Found a cycle: we've seen this ID before in this traversal
					// Check if it's one of the original references
					for _, origId := range references {
						if nextId == origId {
							return true
						}
					}
					continue
				}
				visited[nextId] = true
				queue = append(queue, bfsEntry{id: nextId, depth: current.depth + 1})
			}
		}
	}

	return false
}

type bfsEntry struct {
	id    string
	depth int
}
