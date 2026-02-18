package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// IncrementCitationCount updates a fact's citation metrics when it is referenced by a new claim.
func (k Keeper) IncrementCitationCount(ctx sdk.Context, factID string) error {
	fact, found := k.GetFact(ctx, factID)
	if !found {
		return types.ErrFactNotFound
	}

	fact.CitationCount++
	fact.LastCitedBlock = uint64(ctx.BlockHeight())
	fact.LastUtilizedBlock = uint64(ctx.BlockHeight())
	k.SetFact(ctx, fact)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("fact_cited",
			sdk.NewAttribute("fact_id", factID),
			sdk.NewAttribute("citation_count", fmt.Sprintf("%d", fact.CitationCount)),
		),
	)

	return nil
}

// DistributeCitationRewards distributes a portion of verification rewards to cited facts.
// The total citation pool is CitationShareBPS (15%) of the verification reward.
// Each ancestor level gets CitationDecayPerLevel (50%) less than the previous.
// Protocol floor ensures at least CitationProtocolFloorBPS (2%) stays in protocol.
//
// FARM-3 fix: citingSubmitter is checked against each cited fact's submitter.
// When they match (self-citation), a 50% discount is applied to prevent citation pyramids.
//
// From TS core/knowledge/types.ts: CITATION_ECONOMICS.
func (k Keeper) DistributeCitationRewards(
	ctx sdk.Context,
	factID string,
	totalReward uint64,
	citingSubmitter string,
) uint64 {
	if totalReward == 0 {
		return 0
	}

	ep := k.GetExtendedParams(ctx)

	// Calculate citation pool (15% of total reward)
	citationPool := (totalReward * ep.CitationShareBps) / 1_000_000
	if citationPool == 0 {
		return 0
	}

	// Get outgoing references (facts cited by the newly verified fact)
	refs := k.GetFactReferences(ctx, factID)
	if len(refs) == 0 {
		return 0
	}

	// Distribute rewards with decay per level
	var totalDistributed uint64
	levelShare := citationPool

	for level := 0; level < int(ep.MaxPropagationDepth) && levelShare > 0; level++ {
		// Find facts at this level
		var levelRefs []*types.FactReference
		if level == 0 {
			// Level 0: direct references from the new fact
			for _, ref := range refs {
				if ref.ReferenceType == "supports" {
					levelRefs = append(levelRefs, ref)
				}
			}
		} else {
			// Level N: references of references (already handled by BFS below)
			break
		}

		if len(levelRefs) == 0 {
			break
		}

		// Split level share equally among cited facts at this level
		perFactShare := levelShare / uint64(len(levelRefs))
		if perFactShare == 0 {
			break
		}

		for _, ref := range levelRefs {
			fact, found := k.GetFact(ctx, ref.ToFactId)
			if !found || fact.Status == types.FactStatusExpired {
				continue
			}

			// Apply fundamentality bonus
			bonus := k.calculateFundamentalityBonus(ctx, ref.ToFactId)
			adjustedShare := (perFactShare * bonus) / 1_000_000

			// FARM-3 fix: Self-citation discount — when the citing submitter is the
			// same as the cited fact's submitter, apply SelfCitationDiscountBPS (50%)
			// reduction. This prevents agents from building self-referential citation
			// pyramids for passive income.
			if citingSubmitter != "" && fact.Submitter == citingSubmitter {
				adjustedShare = (adjustedShare * (1_000_000 - ep.SelfCitationDiscountBps)) / 1_000_000
			}

			// Enforce protocol floor
			protocolFloor := (totalReward * ep.CitationProtocolFloorBps) / 1_000_000
			if adjustedShare < protocolFloor {
				adjustedShare = protocolFloor
			}

			// Cap at remaining pool
			if totalDistributed+adjustedShare > citationPool {
				adjustedShare = citationPool - totalDistributed
			}

			if adjustedShare == 0 {
				continue
			}

			// Transfer reward to the cited fact's submitter
			if k.bankKeeper != nil {
				submitterAddr, err := sdk.AccAddressFromBech32(fact.Submitter)
				if err == nil {
					rewardCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(int64(adjustedShare))))
					if err := k.bankKeeper.SendCoinsFromModuleToAccount(
						ctx, types.ModuleName, submitterAddr, rewardCoins,
					); err != nil {
						k.Logger(ctx).Error("failed to send citation reward",
							"fact_id", ref.ToFactId, "amount", adjustedShare, "error", err)
						continue
					}
				}
			}

			// Update fact citation metrics
			fact.TotalCitationRewards += adjustedShare
			fact.LastCitedBlock = uint64(ctx.BlockHeight())
			k.SetFact(ctx, fact)

			totalDistributed += adjustedShare

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("citation_reward",
					sdk.NewAttribute("cited_fact", ref.ToFactId),
					sdk.NewAttribute("citing_fact", factID),
					sdk.NewAttribute("amount", fmt.Sprintf("%d", adjustedShare)),
					sdk.NewAttribute("level", fmt.Sprintf("%d", level)),
				),
			)
		}

		// Decay for next level
		levelShare = (levelShare * ep.CitationDecayPerLevel) / 1_000_000
	}

	return totalDistributed
}

// calculateFundamentalityBonus returns the reward multiplier for a fact based on
// how many other facts depend on it (incoming citation count as proxy for fundamentality).
// Facts cited by 50+ other facts get 2.0x; 20+ get 1.5x; 10+ get 1.25x; 3+ get 1.1x.
func (k Keeper) calculateFundamentalityBonus(ctx sdk.Context, factID string) uint64 {
	ep := k.GetExtendedParams(ctx)
	score := k.CalculateFundamentalityScore(ctx, factID, ep)
	return GetFundamentalityMultiplier(score, ep)
}

// CalculateFundamentalityScore computes a fundamentality score for a fact
// based on its incoming citation count (how many facts depend on it).
func (k Keeper) CalculateFundamentalityScore(ctx sdk.Context, factID string, ep types.ExtendedParams) uint64 {
	incoming := k.GetIncomingReferences(ctx, factID)

	// Count supporting references
	supportCount := uint64(0)
	for _, ref := range incoming {
		if ref.ReferenceType == "supports" {
			supportCount++
		}
	}

	// Map to fundamentality level
	switch {
	case supportCount >= 50:
		return ep.FundamentalityAxiomaticBps // 0.90
	case supportCount >= 20:
		return ep.FundamentalityFundamentalBps // 0.75
	case supportCount >= 10:
		return ep.FundamentalityFoundationalBps // 0.50
	case supportCount >= 3:
		return ep.FundamentalityNotableBps // 0.25
	default:
		return 0
	}
}

// GetFundamentalityMultiplier returns the reward multiplier for a fundamentality score.
func GetFundamentalityMultiplier(score uint64, ep types.ExtendedParams) uint64 {
	switch {
	case score >= ep.FundamentalityAxiomaticBps:
		return ep.FundamentalityBonusAxiomaticBps // 2.0x
	case score >= ep.FundamentalityFundamentalBps:
		return ep.FundamentalityBonusFundamentalBps // 1.5x
	case score >= ep.FundamentalityFoundationalBps:
		return ep.FundamentalityBonusFoundationalBps // 1.25x
	case score >= ep.FundamentalityNotableBps:
		return ep.FundamentalityBonusNotableBps // 1.1x
	default:
		return 1_000_000 // 1.0x
	}
}
