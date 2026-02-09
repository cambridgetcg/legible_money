package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// IsFactExpired checks 3 expiration conditions (from TS knowledge.ts:837):
//  1. Explicit expiry: ExpiresAtBlock > 0 && currentBlock >= ExpiresAtBlock
//  2. Confidence decay: decayed confidence < MinConfidenceThreshold (30%)
//  3. Reverification staleness: age > StalenessMultiplier × halfLife without reverification
func IsFactExpired(fact types.Fact, currentHeight uint64) (bool, string) {
	// Condition 1: Explicit expiry
	if fact.ExpiresAtBlock > 0 && currentHeight >= fact.ExpiresAtBlock {
		return true, "explicit_expiry"
	}

	// Only check active/verified facts
	if fact.Status != types.FactStatusActive &&
		fact.Status != types.FactStatusVerified &&
		fact.Status != types.FactStatusContested {
		return false, ""
	}

	profile, ok := types.CATEGORY_DECAY[fact.EpistemicCategory]
	if !ok {
		return false, ""
	}

	// Eternal categories don't expire from decay
	if profile.HalfLifeBlocks == 0 {
		return false, ""
	}

	// Calculate current decayed confidence
	base := fact.BaseConfidence
	if base == 0 {
		base = fact.Confidence
	}
	var blocksSince uint64
	lastVerified := fact.LastVerifiedHeight
	if lastVerified == 0 {
		lastVerified = fact.VerifiedAtHeight
	}
	if currentHeight > lastVerified {
		blocksSince = currentHeight - lastVerified
	}

	decayed := CalculateDecayedConfidence(base, blocksSince, fact.EpistemicCategory)

	// Condition 2: Confidence below minimum
	if decayed < types.MinConfidenceThreshold {
		return true, "confidence_decay"
	}

	// Condition 3: Reverification staleness
	if profile.RequiresReverification && profile.HalfLifeBlocks > 0 {
		if blocksSince > types.StalenessMultiplier*profile.HalfLifeBlocks {
			return true, "reverification_staleness"
		}
	}

	return false, ""
}

// ExpireStaleFactsInEndBlocker scans for expired facts and transitions them.
// Amortized: processes at most maxPerBlock facts per EndBlocker call.
func (k Keeper) ExpireStaleFactsInEndBlocker(ctx sdk.Context, maxPerBlock int) int {
	currentHeight := uint64(ctx.BlockHeight())
	expired := 0

	k.IterateFacts(ctx, func(fact types.Fact) bool {
		if expired >= maxPerBlock {
			return true // stop iteration
		}

		// Skip already-expired or terminal-state facts
		if fact.Status == types.FactStatusExpired ||
			fact.Status == types.FactStatusRevoked ||
			fact.Status == types.FactStatusDisproven ||
			fact.Status == types.FactStatusSuperseded {
			return false
		}

		isExpired, reason := IsFactExpired(fact, currentHeight)
		if isExpired {
			fact.Status = types.FactStatusExpired
			k.SetFact(ctx, fact)
			expired++

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("fact_expired",
					sdk.NewAttribute("fact_id", fact.Id),
					sdk.NewAttribute("domain", fact.Domain),
					sdk.NewAttribute("reason", reason),
					sdk.NewAttribute("category", fact.EpistemicCategory),
				),
			)
		}

		return false
	})

	return expired
}

// GetFactsNeedingReverification returns facts below the reverification threshold.
func (k Keeper) GetFactsNeedingReverification(ctx sdk.Context, maxResults int) []types.Fact {
	currentHeight := uint64(ctx.BlockHeight())
	var facts []types.Fact

	k.IterateFacts(ctx, func(fact types.Fact) bool {
		if len(facts) >= maxResults {
			return true
		}

		// Only check active facts
		if fact.Status != types.FactStatusActive && fact.Status != types.FactStatusVerified {
			return false
		}

		// Apply decay to get current confidence
		decayed := k.applyDecayToFact(fact, currentHeight)

		if NeedsReverification(decayed, currentHeight) {
			facts = append(facts, decayed)
		}
		return false
	})

	return facts
}

// ReverifyFact resets a fact's verification clock after successful re-verification.
func (k Keeper) ReverifyFact(ctx sdk.Context, factID string, newConfidence uint64, currentHeight uint64) error {
	fact, found := k.GetFact(ctx, factID)
	if !found {
		return types.ErrFactNotFound
	}

	// Apply reverification bonus
	fact = ApplyReverificationBonus(fact)

	// If new verification confidence provided, use it
	if newConfidence > 0 {
		profile, ok := types.CATEGORY_DECAY[fact.EpistemicCategory]
		if ok && newConfidence > profile.MaxConfidence {
			newConfidence = profile.MaxConfidence
		}
		fact.BaseConfidence = newConfidence
		fact.Confidence = newConfidence
	}

	fact.LastVerifiedHeight = currentHeight
	fact.VerifierCount++
	fact.LastConfidenceUpdate = currentHeight
	fact.Maturity = CalculateMaturity(fact.Confidence, currentHeight-fact.CreatedAtHeight, uint64(fact.VerifierCount))

	// Reactivate if expired
	if fact.Status == types.FactStatusExpired {
		fact.Status = types.FactStatusActive
	}

	k.SetFact(ctx, fact)

	ctx.EventManager().EmitEvent(
		sdk.NewEvent("fact_reverified",
			sdk.NewAttribute("fact_id", factID),
			sdk.NewAttribute("new_confidence", fmt.Sprintf("%d", fact.Confidence)),
			sdk.NewAttribute("verifier_count", fmt.Sprintf("%d", fact.VerifierCount)),
		),
	)

	return nil
}
