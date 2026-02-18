package keeper

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// CalculateDecayedConfidence computes the current confidence after time decay.
// Uses exponential half-life decay: C(t) = C_min + (C_0 - C_min) × 0.5^(t/halfLife)
// Eternal categories (halfLife=0) return the original confidence unchanged.
// All values in basis points (0-1,000,000).
func CalculateDecayedConfidence(baseConfidence, blocksSince uint64, category string, ep types.ExtendedParams) uint64 {
	return calculateDecayedConfidenceWithMultiplier(baseConfidence, blocksSince, category, 1, ep)
}

// CalculateDecayedConfidenceNovel computes confidence decay for novel facts.
// Novel facts get an extended half-life (2× by default) so their knowledge
// persists longer before requiring reverification.
func CalculateDecayedConfidenceNovel(baseConfidence, blocksSince uint64, category string, halfLifeMultiplier uint64, ep types.ExtendedParams) uint64 {
	if halfLifeMultiplier == 0 {
		halfLifeMultiplier = 1
	}
	return calculateDecayedConfidenceWithMultiplier(baseConfidence, blocksSince, category, halfLifeMultiplier, ep)
}

// calculateDecayedConfidenceWithMultiplier is the shared implementation
// for standard and novelty-aware decay.
func calculateDecayedConfidenceWithMultiplier(baseConfidence, blocksSince uint64, category string, halfLifeMultiplier uint64, ep types.ExtendedParams) uint64 {
	profile, ok := ep.GetCategoryDecayProfile(category)
	if !ok {
		return baseConfidence // unknown category — no decay
	}

	// Eternal categories don't decay
	if profile.HalfLifeBlocks == 0 {
		// Still enforce ceiling
		if baseConfidence > profile.MaxConfidence {
			return profile.MaxConfidence
		}
		return baseConfidence
	}

	if blocksSince == 0 {
		conf := baseConfidence
		if conf > profile.MaxConfidence {
			conf = profile.MaxConfidence
		}
		return conf
	}

	// Exponential decay: C(t) = C_min + (C_0 - C_min) × 0.5^(age/halfLife)
	// Novel facts get an extended half-life via multiplier
	halfLife := float64(profile.HalfLifeBlocks * halfLifeMultiplier)
	decayFactor := math.Pow(0.5, float64(blocksSince)/halfLife)

	minConf := float64(profile.MinConfidence)
	base := float64(baseConfidence)
	if base > float64(profile.MaxConfidence) {
		base = float64(profile.MaxConfidence)
	}

	decayed := minConf + (base-minConf)*decayFactor
	result := uint64(decayed)

	// Enforce floor
	if result < profile.MinConfidence {
		result = profile.MinConfidence
	}

	return result
}

// GetFactWithDecay retrieves a fact and applies lazy confidence decay.
// The decayed confidence is returned but NOT persisted (lazy evaluation).
func (k Keeper) GetFactWithDecay(ctx sdk.Context, factID string) (types.Fact, bool) {
	fact, found := k.GetFact(ctx, factID)
	if !found {
		return fact, false
	}

	ep := k.GetExtendedParams(ctx)
	currentHeight := uint64(ctx.BlockHeight())
	fact = k.applyDecayToFact(fact, currentHeight, ep)
	return fact, true
}

// applyDecayToFact computes the current decayed confidence for a fact.
func (k Keeper) applyDecayToFact(fact types.Fact, currentHeight uint64, ep types.ExtendedParams) types.Fact {
	// Use BaseConfidence if set, otherwise use Confidence as the base
	base := fact.BaseConfidence
	if base == 0 {
		base = fact.Confidence
	}

	// Calculate age since last verification
	var blocksSince uint64
	if fact.LastVerifiedHeight > 0 && currentHeight > fact.LastVerifiedHeight {
		blocksSince = currentHeight - fact.LastVerifiedHeight
	} else if fact.VerifiedAtHeight > 0 && currentHeight > fact.VerifiedAtHeight {
		blocksSince = currentHeight - fact.VerifiedAtHeight
	}

	// Novel or prophetic facts get slower decay (extended half-life)
	var decayed uint64
	if fact.NoveltyScore >= ep.NoveltyImmunityThresholdBps || fact.IsProphetic {
		decayed = CalculateDecayedConfidenceNovel(base, blocksSince, fact.EpistemicCategory, ep.NoveltyDecayHalfLifeMultiplier, ep)
	} else {
		decayed = CalculateDecayedConfidence(base, blocksSince, fact.EpistemicCategory, ep)
	}
	fact.Confidence = decayed
	fact.LastConfidenceUpdate = currentHeight

	// Update maturity based on current state
	fact.Maturity = CalculateMaturity(decayed, blocksSince, uint64(fact.VerifierCount), ep)

	return fact
}

// CalculateMaturity determines fact maturity level based on confidence, age, and verifications.
func CalculateMaturity(confidence, ageBlocks, verificationCount uint64, ep types.ExtendedParams) string {
	if confidence >= ep.MaturityCanonicalMinConf &&
		ageBlocks >= ep.MaturityCanonicalMinAge &&
		verificationCount >= uint64(ep.MaturityCanonicalMinVerif) {
		return types.MaturityCanonical
	}
	if confidence >= ep.MaturityEstablishedMinConf &&
		ageBlocks >= ep.MaturityEstablishedMinAge &&
		verificationCount >= uint64(ep.MaturityEstablishedMinVerif) {
		return types.MaturityEstablished
	}
	return types.MaturityEmerging
}

// NeedsReverification checks if a fact's confidence has decayed below the reverification threshold.
func NeedsReverification(fact types.Fact, currentHeight uint64, ep types.ExtendedParams) bool {
	profile, ok := ep.GetCategoryDecayProfile(fact.EpistemicCategory)
	if !ok || !profile.RequiresReverification {
		return false
	}

	// Check confidence threshold
	if fact.Confidence < ep.ReverificationThreshold {
		return true
	}

	// Check staleness: age > 2× halfLife without reverification
	if profile.HalfLifeBlocks > 0 {
		lastVerified := fact.LastVerifiedHeight
		if lastVerified == 0 {
			lastVerified = fact.VerifiedAtHeight
		}
		if currentHeight > lastVerified {
			age := currentHeight - lastVerified
			if age > uint64(ep.StalenessMultiplier)*profile.HalfLifeBlocks {
				return true
			}
		}
	}

	return false
}

// ApplyReverificationBonus applies the category-specific reverification bonus.
func ApplyReverificationBonus(fact types.Fact, ep types.ExtendedParams) types.Fact {
	profile, ok := ep.GetCategoryDecayProfile(fact.EpistemicCategory)
	if !ok {
		return fact
	}

	newConf := fact.BaseConfidence + profile.ReverificationBonus
	if newConf > profile.MaxConfidence {
		newConf = profile.MaxConfidence
	}
	fact.BaseConfidence = newConf
	fact.Confidence = newConf
	return fact
}
