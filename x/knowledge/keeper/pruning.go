package keeper

import (
	"fmt"
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// ============================================================================
// Knowledge Maintenance Score (KMS) — Immunity + Calculation
// ============================================================================

// IsFactImmune returns true if the fact cannot be pruned (Godelian boundaries).
// Immune facts: eternal categories (HalfLifeBlocks == 0), high citation count,
// or terminal statuses that are already handled.
func (k Keeper) IsFactImmune(ctx sdk.Context, fact types.Fact) bool {
	// Terminal statuses — already handled, skip pruning.
	// Note: FactStatusPruned is NOT immune — pruned facts still need hard-deletion processing.
	if fact.Status == types.FactStatusExpired ||
		fact.Status == types.FactStatusDisproven ||
		fact.Status == types.FactStatusRevoked {
		return true
	}

	// Eternal categories: analytic, formal, protocol, computational (HalfLifeBlocks == 0)
	if profile, ok := types.CATEGORY_DECAY[fact.EpistemicCategory]; ok {
		if profile.HalfLifeBlocks == 0 {
			return true
		}
	}

	// Fundamentality immunity: facts cited by many others are foundational
	ep := k.GetExtendedParams(ctx)
	threshold := ep.FundamentalityImmunityThreshold
	if threshold == 0 {
		threshold = 50 // default
	}
	if fact.CitationCount >= threshold {
		return true
	}

	// Prophetic facts are immune from pruning — they were vindicated by citations
	if fact.IsProphetic {
		return true
	}

	return false
}

// CalculateKMS computes the Knowledge Maintenance Score for a fact (0-1,000,000 basis points).
// KMS = w_c*Contribution + w_u*Utilization + w_t*Trajectory + w_m*Maturity + w_n*Novelty + patronage_bonus
func (k Keeper) CalculateKMS(ctx sdk.Context, fact types.Fact) uint64 {
	ep := k.GetExtendedParams(ctx)

	contribution := k.calculateContributionScore(ctx, fact)
	utilization := k.calculateUtilizationScore(ctx, fact)
	trajectory := k.calculateTrajectoryScore(ctx, fact)
	maturity := k.calculateMaturityBonus(ctx, fact)
	novelty := calculateNoveltyScore(fact)

	// Weighted sum (5 components, weights sum to 1,000,000)
	kms := (ep.ContributionWeight*contribution +
		ep.UtilizationWeight*utilization +
		ep.TrajectoryWeight*trajectory +
		ep.MaturityWeight*maturity +
		ep.NoveltyWeight*novelty) / 1_000_000

	// Add patronage bonus (capped at 300,000 bps = 30%)
	kms += fact.PatronageBonus

	// Clamp to 0-1,000,000
	if kms > 1_000_000 {
		kms = 1_000_000
	}

	return kms
}

// calculateNoveltyScore computes the novelty component of KMS from the
// fact's stored novelty and bridge scores. Returns 0-1,000,000.
func calculateNoveltyScore(fact types.Fact) uint64 {
	// Combine novelty_score and bridge_score with prophetic bonus
	score := fact.NoveltyScore

	// Bridge score adds directly (already in bps)
	if fact.BridgeScore > 0 {
		bridgeContrib := fact.BridgeScore * 2 // Bridge facts are extra valuable for KMS
		if bridgeContrib > 500_000 {
			bridgeContrib = 500_000
		}
		score += bridgeContrib
	}

	// Prophetic facts get a major KMS boost (hard to prune vindicated knowledge)
	if fact.IsProphetic {
		score += 300_000
	}

	if score > 1_000_000 {
		score = 1_000_000
	}
	return score
}

// calculateContributionScore measures how much a fact contributes to the knowledge graph.
// Based on incoming citation count + fundamentality tier + active dependent ratio.
// Returns 0-1,000,000.
func (k Keeper) calculateContributionScore(ctx sdk.Context, fact types.Fact) uint64 {
	// Incoming citations (capped at 100 for scoring)
	citations := fact.CitationCount
	if citations > 100 {
		citations = 100
	}

	// Score scales with sqrt of citations for diminishing returns
	// sqrt(100) = 10, so max contribution from citations = 10/10 * 600,000 = 600,000
	sqrtCitations := uint64(math.Sqrt(float64(citations)))
	citationScore := (sqrtCitations * 600_000) / 10

	// Fundamentality tier bonus
	fundScore := k.CalculateFundamentalityScore(ctx, fact.Id)
	var fundBonus uint64
	switch {
	case fundScore >= types.FundamentalityAxiomatic:
		fundBonus = 400_000
	case fundScore >= types.FundamentalityFundamental:
		fundBonus = 300_000
	case fundScore >= types.FundamentalityFoundational:
		fundBonus = 200_000
	case fundScore >= types.FundamentalityNotable:
		fundBonus = 100_000
	}

	score := citationScore + fundBonus
	if score > 1_000_000 {
		score = 1_000_000
	}
	return score
}

// calculateUtilizationScore measures how recently and frequently a fact is used.
// Based on last_cited_block recency + citation_count frequency.
// Returns 0-1,000,000.
func (k Keeper) calculateUtilizationScore(ctx sdk.Context, fact types.Fact) uint64 {
	currentHeight := uint64(ctx.BlockHeight())

	// Use last_utilized_block (or last_cited_block as fallback)
	lastUsed := fact.LastUtilizedBlock
	if lastUsed == 0 {
		lastUsed = fact.LastCitedBlock
	}
	if lastUsed == 0 {
		lastUsed = fact.CreatedAtHeight
	}

	// Recency score: exponential decay based on blocks since last use
	// Full score if used within last day (34,272 blocks), decays to ~0 after 100 days
	blocksSinceUse := currentHeight - lastUsed
	if currentHeight < lastUsed {
		blocksSinceUse = 0
	}

	var recencyScore uint64
	if blocksSinceUse == 0 {
		recencyScore = 1_000_000
	} else {
		// Decay with half-life of ~10 days (342,720 blocks)
		halfLife := uint64(342_720)
		// Simple decay: score = 1,000,000 / (1 + blocksSinceUse / halfLife)
		recencyScore = 1_000_000 * halfLife / (halfLife + blocksSinceUse)
	}

	// Frequency component: citation count (capped at 50)
	freq := fact.CitationCount
	if freq > 50 {
		freq = 50
	}
	frequencyScore := (freq * 1_000_000) / 50

	// Weighted combination: 60% recency, 40% frequency
	score := (recencyScore*600_000 + frequencyScore*400_000) / 1_000_000
	if score > 1_000_000 {
		score = 1_000_000
	}
	return score
}

// calculateTrajectoryScore measures whether fact confidence is stable/growing vs decaying.
// Returns 0-1,000,000.
func (k Keeper) calculateTrajectoryScore(ctx sdk.Context, fact types.Fact) uint64 {
	// If confidence is above base → growing (survived challenges boost)
	if fact.Confidence > fact.BaseConfidence && fact.BaseConfidence > 0 {
		// Growing: scale based on how much above base
		growth := fact.Confidence - fact.BaseConfidence
		// Cap growth bonus at 200,000 bps of delta
		score := uint64(500_000) + (growth*500_000)/1_000_000
		if score > 1_000_000 {
			score = 1_000_000
		}
		return score
	}

	// Stable: confidence == base
	if fact.Confidence == fact.BaseConfidence || fact.BaseConfidence == 0 {
		return 500_000
	}

	// Decaying: confidence below base
	if fact.BaseConfidence > 0 {
		ratio := (fact.Confidence * 1_000_000) / fact.BaseConfidence
		// Scale 0-500,000 based on how much confidence remains
		score := (ratio * 500_000) / 1_000_000
		return score
	}

	return 500_000 // neutral default
}

// calculateMaturityBonus rewards older, more established facts.
// Returns 0-1,000,000.
func (k Keeper) calculateMaturityBonus(ctx sdk.Context, fact types.Fact) uint64 {
	currentHeight := uint64(ctx.BlockHeight())
	var score uint64

	// Age component (capped at ~100 days = 3,427,200 blocks)
	age := currentHeight - fact.CreatedAtHeight
	if currentHeight < fact.CreatedAtHeight {
		age = 0
	}
	maxAge := uint64(3_427_200) // ~100 days
	if age > maxAge {
		age = maxAge
	}
	ageScore := (age * 400_000) / maxAge

	// Maturity level bonus
	var maturityBonus uint64
	switch fact.Maturity {
	case types.MaturityCanonical:
		maturityBonus = 400_000
	case types.MaturityEstablished:
		maturityBonus = 200_000
	case types.MaturityEmerging:
		maturityBonus = 50_000
	}

	// Survived challenges bonus (max 200,000 for 4+ survived)
	challengeBonus := uint64(fact.SurvivedChallenges) * 50_000
	if challengeBonus > 200_000 {
		challengeBonus = 200_000
	}

	score = ageScore + maturityBonus + challengeBonus
	if score > 1_000_000 {
		score = 1_000_000
	}
	return score
}

// ============================================================================
// Pruning Lifecycle: at_risk → pruned → hard_deleted
// ============================================================================

// GetGracePeriod returns the grace period in blocks for a fact's epistemic category.
// Grace = max(GracePeriodBlocks, HalfLifeBlocks * GraceMultiplier)
func (k Keeper) GetGracePeriod(ctx sdk.Context, category string) uint64 {
	ep := k.GetExtendedParams(ctx)
	grace := ep.GracePeriodBlocks
	if grace == 0 {
		grace = 1_028_160 // default ~30 days
	}

	if profile, ok := types.CATEGORY_DECAY[category]; ok && profile.HalfLifeBlocks > 0 {
		multiplier := ep.GraceMultiplier
		if multiplier == 0 {
			multiplier = 2
		}
		categoryGrace := profile.HalfLifeBlocks * multiplier
		if categoryGrace > grace {
			grace = categoryGrace
		}
	}

	return grace
}

// GetGracePeriodForFact returns the grace period accounting for novelty.
// Novel facts (NoveltyScore above threshold) get an extended grace period
// via NoveltyGraceMultiplier, protecting ahead-of-time knowledge from premature pruning.
func (k Keeper) GetGracePeriodForFact(ctx sdk.Context, fact types.Fact) uint64 {
	baseGrace := k.GetGracePeriod(ctx, fact.EpistemicCategory)

	ep := k.GetExtendedParams(ctx)
	noveltyThreshold := ep.NoveltyImmunityThresholdBps
	if noveltyThreshold == 0 {
		noveltyThreshold = types.NoveltyImmunityThreshold
	}
	noveltyMult := ep.NoveltyGraceMultiplier
	if noveltyMult == 0 {
		noveltyMult = types.NoveltyGraceMultiplier
	}

	// Novel facts or prophetic facts get extended grace
	if fact.NoveltyScore >= noveltyThreshold || fact.IsProphetic {
		return baseGrace * noveltyMult
	}

	return baseGrace
}

// MarkFactAtRisk sets the fact status to at_risk and records the grace start block.
func (k Keeper) MarkFactAtRisk(ctx sdk.Context, fact *types.Fact) {
	fact.Status = types.FactStatusAtRisk
	fact.PruningGraceStart = uint64(ctx.BlockHeight())
	k.SetFact(ctx, *fact)

	// Add to pruning queue index for efficient batch lookup
	gracePeriod := k.GetGracePeriod(ctx, fact.EpistemicCategory)
	graceEnd := fact.PruningGraceStart + gracePeriod
	k.SetPruningQueueEntry(ctx, graceEnd, fact.Id)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("fact_at_risk",
			sdk.NewAttribute("fact_id", fact.Id),
			sdk.NewAttribute("kms_threshold", "below"),
			sdk.NewAttribute("grace_start", fmt.Sprintf("%d", fact.PruningGraceStart)),
			sdk.NewAttribute("grace_end", fmt.Sprintf("%d", graceEnd)),
		),
	)
}

// RescueFact clears the pruning grace and restores the fact to its previous status.
func (k Keeper) RescueFact(ctx sdk.Context, fact *types.Fact, rescueType string) {
	if fact.PruningGraceStart > 0 {
		// Remove from pruning queue
		gracePeriod := k.GetGracePeriod(ctx, fact.EpistemicCategory)
		graceEnd := fact.PruningGraceStart + gracePeriod
		k.DeletePruningQueueEntry(ctx, graceEnd, fact.Id)
	}

	fact.Status = types.FactStatusVerified
	fact.PruningGraceStart = 0
	k.SetFact(ctx, *fact)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("fact_rescued",
			sdk.NewAttribute("fact_id", fact.Id),
			sdk.NewAttribute("rescue_type", rescueType),
		),
	)
}

// ArchiveFact sets the fact status to "pruned", pauses vesting, and severs active citation chains.
func (k Keeper) ArchiveFact(ctx sdk.Context, factID string) error {
	fact, found := k.GetFact(ctx, factID)
	if !found {
		return types.ErrFactNotFound
	}

	fact.Status = types.FactStatusPruned
	k.SetFact(ctx, fact)

	// Pause vesting if vesting_rewards keeper is available
	if k.vestingRewardsKeeper != nil && fact.OriginalClaimId != "" {
		if err := k.vestingRewardsKeeper.PauseVestingByClaimId(ctx, fact.OriginalClaimId); err != nil {
			k.Logger(ctx).Error("failed to pause vesting on archive",
				"fact_id", factID, "claim_id", fact.OriginalClaimId, "error", err)
		}
	}

	// Remove from pruning queue (if still there)
	if fact.PruningGraceStart > 0 {
		gracePeriod := k.GetGracePeriod(ctx, fact.EpistemicCategory)
		graceEnd := fact.PruningGraceStart + gracePeriod
		k.DeletePruningQueueEntry(ctx, graceEnd, fact.Id)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("fact_archived",
			sdk.NewAttribute("fact_id", factID),
			sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)

	k.Logger(ctx).Info("fact archived (pruned)",
		"fact_id", factID,
		"domain", fact.Domain,
		"category", fact.EpistemicCategory,
		"height", ctx.BlockHeight(),
	)

	return nil
}

// HardDeleteFact removes an archived fact after the retention period.
func (k Keeper) HardDeleteFact(ctx sdk.Context, factID string) {
	// Settle any patronage records before deletion
	k.SettlePatronageOnDelete(ctx, factID)

	// Delete the fact and its indexes
	k.DeleteFact(ctx, factID)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("fact_hard_deleted",
			sdk.NewAttribute("fact_id", factID),
			sdk.NewAttribute("height", fmt.Sprintf("%d", ctx.BlockHeight())),
		),
	)
}

// ============================================================================
// EndBlocker Pruning Scanner
// ============================================================================

// PruneUnusedFactsInEndBlocker scans facts and applies the pruning lifecycle.
// Returns the number of facts processed (at_risk + pruned + deleted).
func (k Keeper) PruneUnusedFactsInEndBlocker(ctx sdk.Context, maxPerBlock int) int {
	ep := k.GetExtendedParams(ctx)
	if !ep.PruningEnabled {
		return 0
	}

	currentHeight := uint64(ctx.BlockHeight())
	processed := 0

	k.IterateFacts(ctx, func(fact types.Fact) bool {
		if processed >= maxPerBlock {
			return true // stop iteration
		}

		// Skip immune facts
		if k.IsFactImmune(ctx, fact) {
			return false
		}

		switch fact.Status {
		case types.FactStatusVerified, types.FactStatusActive:
			// Check if KMS is below threshold
			kms := k.CalculateKMS(ctx, fact)
			if kms < ep.PruningThresholdBps {
				k.MarkFactAtRisk(ctx, &fact)
				processed++
			}

		case types.FactStatusAtRisk:
			// Check if grace period expired (novelty-aware)
			gracePeriod := k.GetGracePeriodForFact(ctx, fact)
			graceEnd := fact.PruningGraceStart + gracePeriod

			if currentHeight >= graceEnd {
				// Re-check KMS — might have recovered
				kms := k.CalculateKMS(ctx, fact)
				if kms >= ep.PruningThresholdBps {
					k.RescueFact(ctx, &fact, "kms_recovered")
				} else {
					if err := k.ArchiveFact(ctx, fact.Id); err != nil {
						k.Logger(ctx).Error("failed to archive fact", "fact_id", fact.Id, "error", err)
					}
				}
				processed++
			} else {
				// Still in grace — check if KMS recovered
				kms := k.CalculateKMS(ctx, fact)
				if kms >= ep.PruningThresholdBps {
					k.RescueFact(ctx, &fact, "kms_recovered")
					processed++
				}
			}

		case types.FactStatusPruned:
			// Check if archive retention expired → hard delete
			if fact.PruningGraceStart > 0 {
				gracePeriod := k.GetGracePeriod(ctx, fact.EpistemicCategory)
				archiveStart := fact.PruningGraceStart + gracePeriod
				if currentHeight >= archiveStart+ep.ArchiveRetentionBlocks {
					k.HardDeleteFact(ctx, fact.Id)
					processed++
				}
			}
		}

		return false
	})

	return processed
}

// ============================================================================
// Patronage Settlement
// ============================================================================

// SettlePatronageOnDelete settles all patronage records when a fact is hard deleted.
// 50% burned, 50% sent to research fund.
func (k Keeper) SettlePatronageOnDelete(ctx sdk.Context, factID string) {
	records := k.GetPatronageRecordsByFact(ctx, factID)
	for _, record := range records {
		if record.Settled {
			continue
		}

		// Mark as settled
		record.Settled = true
		k.SetPatronageRecord(ctx, record)

		// Note: actual burn + research fund transfer would use bankKeeper
		// For now we just mark settled and emit event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent("patronage_settled",
				sdk.NewAttribute("fact_id", factID),
				sdk.NewAttribute("patron", record.Patron),
				sdk.NewAttribute("amount", record.Amount),
				sdk.NewAttribute("outcome", "fact_deleted"),
			),
		)
	}
}

// SettlePatronageOnRescue returns patronage stake to patron when a fact is rescued.
func (k Keeper) SettlePatronageOnRescue(ctx sdk.Context, factID string) {
	records := k.GetPatronageRecordsByFact(ctx, factID)
	for _, record := range records {
		if record.Settled {
			continue
		}

		record.Settled = true
		k.SetPatronageRecord(ctx, record)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("patronage_returned",
				sdk.NewAttribute("fact_id", factID),
				sdk.NewAttribute("patron", record.Patron),
				sdk.NewAttribute("amount", record.Amount),
			),
		)
	}
}
