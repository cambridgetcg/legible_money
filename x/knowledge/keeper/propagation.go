package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// PropagateConfidenceChange propagates a confidence change through the citation graph
// using BFS traversal with weakest-link derived confidence.
//
// From TS state/knowledge.ts: when a fact's confidence changes, all facts that cite it
// (incoming "supports" references) may need their derived confidence updated.
// Propagation stops when the change is below PropagationThreshold (1%) or MaxPropagationDepth is reached.
func (k Keeper) PropagateConfidenceChange(ctx sdk.Context, factID string, newConfidence uint64) int {
	updated := 0
	ep := k.GetExtendedParams(ctx)

	// BFS queue: fact IDs whose dependents need checking
	type queueEntry struct {
		factID string
		depth  int
	}
	queue := []queueEntry{{factID: factID, depth: 0}}
	visited := map[string]bool{factID: true}

	for len(queue) > 0 {
		entry := queue[0]
		queue = queue[1:]

		if entry.depth >= int(ep.MaxPropagationDepth) {
			continue
		}

		// Find all facts that cite this fact (incoming "supports" references)
		incoming := k.GetIncomingReferences(ctx, entry.factID)
		for _, ref := range incoming {
			if ref.ReferenceType != "supports" {
				continue
			}
			dependentID := ref.FromFactId
			if visited[dependentID] {
				continue
			}
			visited[dependentID] = true

			dependent, found := k.GetFact(ctx, dependentID)
			if !found || dependent.Status == types.FactStatusExpired ||
				dependent.Status == types.FactStatusRevoked {
				continue
			}

			// Compute derived confidence for this dependent fact
			derived := k.CalculateDerivedConfidence(ctx, dependentID)

			// Check if change exceeds propagation threshold
			oldConf := dependent.Confidence
			var diff uint64
			if derived > oldConf {
				diff = derived - oldConf
			} else {
				diff = oldConf - derived
			}

			if diff < ep.PropagationThreshold {
				continue
			}

			// Apply the derived confidence (clamped by category ceiling)
			if profile, ok := ep.GetCategoryDecayProfile(dependent.EpistemicCategory); ok {
				if derived > profile.MaxConfidence {
					derived = profile.MaxConfidence
				}
			}

			dependent.Confidence = derived
			dependent.LastConfidenceUpdate = uint64(ctx.BlockHeight())
			k.SetFact(ctx, dependent)
			updated++

			// Continue propagation
			queue = append(queue, queueEntry{factID: dependentID, depth: entry.depth + 1})
		}
	}

	if updated > 0 {
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("confidence_propagated",
				sdk.NewAttribute("source_fact", factID),
				sdk.NewAttribute("facts_updated", fmt.Sprintf("%d", updated)),
			),
		)
	}

	return updated
}

// CalculateDerivedConfidence computes confidence for a fact based on its references.
// Uses weakest-link: the derived confidence cannot exceed the minimum confidence
// of any fact it cites (supports).
//
// If the fact has no outgoing "supports" references, returns its own base confidence.
func (k Keeper) CalculateDerivedConfidence(ctx sdk.Context, factID string) uint64 {
	fact, found := k.GetFact(ctx, factID)
	if !found {
		return 0
	}

	outgoing := k.GetFactReferences(ctx, factID)
	if len(outgoing) == 0 {
		return fact.BaseConfidence
	}

	ep := k.GetExtendedParams(ctx)

	// Weakest-link: minimum confidence among all supporting references
	var minConf uint64
	hasSupportRef := false

	for _, ref := range outgoing {
		if ref.ReferenceType != "supports" {
			continue
		}
		cited, citedFound := k.GetFact(ctx, ref.ToFactId)
		if !citedFound {
			continue
		}

		// Apply decay to cited fact
		currentHeight := uint64(ctx.BlockHeight())
		cited = k.applyDecayToFact(cited, currentHeight, ep)

		// Check for cycles — penalize cycle members
		if k.isInCycle(ctx, factID, ref.ToFactId) {
			cited.Confidence = (cited.Confidence * ep.CyclePenaltyFactor) / 1_000_000
		}

		if !hasSupportRef || cited.Confidence < minConf {
			minConf = cited.Confidence
			hasSupportRef = true
		}
	}

	if !hasSupportRef {
		return fact.BaseConfidence
	}

	// Derived confidence is the lesser of own base confidence and weakest reference
	if fact.BaseConfidence < minConf {
		return fact.BaseConfidence
	}
	return minConf
}

// DetectCircularDependencies finds all facts involved in circular reference chains
// using iterative DFS (Tarjan-style detection without full SCC decomposition).
// Returns fact IDs that are part of at least one cycle.
func (k Keeper) DetectCircularDependencies(ctx sdk.Context) []string {
	var cycleFacts []string
	cycleSet := make(map[string]bool)

	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		if cycleSet[fact.Id] {
			return false // already identified
		}
		if k.isFactInAnyCycle(ctx, fact.Id) {
			cycleSet[fact.Id] = true
			cycleFacts = append(cycleFacts, fact.Id)
		}
		return false
	})

	return cycleFacts
}

// isInCycle checks if there is a path from targetID back to sourceID
// through "supports" references, indicating a circular dependency.
func (k Keeper) isInCycle(ctx sdk.Context, sourceID, targetID string) bool {
	// DFS from targetID looking for sourceID
	visited := map[string]bool{targetID: true}
	stack := []string{targetID}

	for len(stack) > 0 {
		current := stack[len(stack)-1]
		stack = stack[:len(stack)-1]

		refs := k.GetFactReferences(ctx, current)
		for _, ref := range refs {
			if ref.ReferenceType != "supports" {
				continue
			}
			if ref.ToFactId == sourceID {
				return true
			}
			if !visited[ref.ToFactId] {
				visited[ref.ToFactId] = true
				stack = append(stack, ref.ToFactId)
			}
		}

		// Limit search depth (use a reasonable default if ep not available)
		if len(visited) > 20 {
			break
		}
	}

	return false
}

// isFactInAnyCycle checks if a specific fact is part of any reference cycle.
func (k Keeper) isFactInAnyCycle(ctx sdk.Context, factID string) bool {
	// DFS from factID — if we can reach factID again, it's in a cycle
	refs := k.GetFactReferences(ctx, factID)
	for _, ref := range refs {
		if ref.ReferenceType != "supports" {
			continue
		}
		if k.isInCycle(ctx, factID, ref.ToFactId) {
			return true
		}
	}
	return false
}

// ApplyCyclePenalties finds cycle members and reduces their confidence by CyclePenaltyFactor.
// Called periodically (e.g., every 1000 blocks) to discourage circular citation.
func (k Keeper) ApplyCyclePenalties(ctx sdk.Context) int {
	penalized := 0
	ep := k.GetExtendedParams(ctx)
	cycleFacts := k.DetectCircularDependencies(ctx)

	for _, factID := range cycleFacts {
		fact, found := k.GetFact(ctx, factID)
		if !found {
			continue
		}

		// Apply 0.4× penalty
		penalizedConf := (fact.Confidence * ep.CyclePenaltyFactor) / 1_000_000
		if penalizedConf == fact.Confidence {
			continue // already at minimum
		}

		fact.Confidence = penalizedConf
		fact.LastConfidenceUpdate = uint64(ctx.BlockHeight())
		k.SetFact(ctx, fact)
		penalized++

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("cycle_penalty_applied",
				sdk.NewAttribute("fact_id", factID),
				sdk.NewAttribute("new_confidence", fmt.Sprintf("%d", penalizedConf)),
			),
		)
	}

	return penalized
}
