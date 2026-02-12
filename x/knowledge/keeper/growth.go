package keeper

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// ApplyConfidenceGrowth applies survival-based confidence growth to all verified facts.
// Called every ConfidenceGrowthEpoch blocks (1,111 blocks, ~47 min) in BeginBlocker.
//
// Growth formula:
//
//	effective_growth = growth_per_epoch * survival_multiplier / 1,000,000
//	survival_multiplier = min(3.0, 1.0 + survived_challenges * 0.5)
//
// Caps:
//   - During challenge window: confidence frozen at structural score
//   - After window closes unchallenged: growth up to MaxSurvivalConfidence (77%)
//   - After surviving 1+ challenges: cap raised to 88%
//   - After reaching established maturity: cap raised to category ceiling
func (k Keeper) ApplyConfidenceGrowth(ctx sdk.Context, currentHeight uint64) {
	params := k.GetParams(ctx)
	growthPerEpoch := params.ConfidenceGrowthPerEpoch
	if growthPerEpoch == 0 {
		growthPerEpoch = types.ConfidenceGrowthPerEpochDefault
	}

	maxSurvival := params.MaxSurvivalConfidence
	if maxSurvival == 0 {
		maxSurvival = types.MaxSurvivalConfidenceDefault
	}

	k.IterateFacts(ctx, func(fact types.Fact) bool {
		// Skip non-verified facts
		if fact.Status != types.FactStatusVerified && fact.Status != types.FactStatusActive {
			return false
		}
		// Skip provisional facts (confidence frozen during challenge window)
		if fact.IsProvisional {
			return false
		}

		// Calculate survival multiplier: min(3.0, 1.0 + survived * 0.5)
		// In basis points: min(3_000_000, 1_000_000 + survived * 500_000)
		survivalMultiplier := uint64(1_000_000) + uint64(fact.SurvivedChallenges)*500_000
		if survivalMultiplier > 3_000_000 {
			survivalMultiplier = 3_000_000
		}

		// effective_growth = growthPerEpoch * survivalMultiplier / 1_000_000
		effectiveGrowth := (growthPerEpoch * survivalMultiplier) / 1_000_000

		// Determine confidence cap
		cap := maxSurvival
		if fact.SurvivedChallenges > 0 {
			cap = types.SurvivedChallengeConfidenceCap // 88%
		}
		if fact.Maturity == types.MaturityEstablished || fact.Maturity == types.MaturityCanonical {
			categoryCeiling := getCategoryCeiling(fact.EpistemicCategory)
			if categoryCeiling > cap {
				cap = categoryCeiling
			}
		}

		// Apply zone ceiling if ontology keeper available
		if k.ontologyKeeper != nil && fact.LogicZone != "" {
			zoneCeiling := k.ontologyKeeper.GetZoneConfidenceCeiling(ctx, fact.LogicZone)
			if zoneCeiling < cap {
				cap = zoneCeiling
			}
		}

		// Apply category ceiling
		categoryCeiling := getCategoryCeiling(fact.EpistemicCategory)
		if categoryCeiling < cap {
			cap = categoryCeiling
		}

		// Apply growth
		newConfidence := fact.Confidence + effectiveGrowth
		if newConfidence > cap {
			newConfidence = cap
		}

		if newConfidence != fact.Confidence {
			fact.Confidence = newConfidence
			fact.LastConfidenceUpdate = currentHeight
			k.SetFact(ctx, fact)
		}

		return false
	})
}

// DecayStaleContestedFacts applies confidence decay to facts stuck in "contested" status
// without recent activity (R-5). Called every ConfidenceGrowthEpoch blocks in BeginBlocker.
//
// A contested fact is "stale" if no confidence change, challenge, or counter-fact
// activity has occurred for StaleContestedDecayEpochs epochs. Once stale, confidence
// decays by StaleContestedDecayRateBps (5%) per epoch. If confidence falls below
// RetractionThresholdBps (10%), the fact is revoked.
func (k Keeper) DecayStaleContestedFacts(ctx sdk.Context, currentHeight uint64) {
	growthEpoch := uint64(types.ConfidenceGrowthEpoch)
	staleThreshold := types.StaleContestedDecayEpochs * growthEpoch

	k.IterateFacts(ctx, func(fact types.Fact) bool {
		if fact.Status != types.FactStatusContested {
			return false
		}

		// Determine last activity height
		lastActivity := fact.LastConfidenceUpdate
		if fact.LastVerifiedHeight > lastActivity {
			lastActivity = fact.LastVerifiedHeight
		}

		// Not stale yet
		if lastActivity == 0 || currentHeight < lastActivity+staleThreshold {
			return false
		}

		// Apply decay
		decayAmount := (fact.Confidence * types.StaleContestedDecayRateBps) / 1_000_000
		if decayAmount == 0 {
			decayAmount = 1 // Always decay at least 1 unit
		}

		if fact.Confidence <= decayAmount || fact.Confidence-decayAmount < RetractionThresholdBps {
			// Decayed below retraction threshold → revoke
			fact.Status = types.FactStatusRevoked
			fact.Confidence = 0
			fact.LastConfidenceUpdate = currentHeight
			k.SetFact(ctx, fact)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("stale_contested_fact_revoked",
					sdk.NewAttribute("fact_id", fact.Id),
					sdk.NewAttribute("stale_blocks", fmt.Sprintf("%d", currentHeight-lastActivity)),
				),
			)

			// Cascade to dependents
			k.cascadeConfidenceReduction(ctx, fact.Id, CascadeReductionBps)
		} else {
			fact.Confidence -= decayAmount
			fact.LastConfidenceUpdate = currentHeight
			k.SetFact(ctx, fact)

			ctx.EventManager().EmitEvent(
				sdk.NewEvent("stale_contested_fact_decayed",
					sdk.NewAttribute("fact_id", fact.Id),
					sdk.NewAttribute("new_confidence", fmt.Sprintf("%d", fact.Confidence)),
					sdk.NewAttribute("decay_amount", fmt.Sprintf("%d", decayAmount)),
				),
			)
		}

		return false
	})
}

// AdvanceMaturityLevels checks facts for maturity advancement.
// Called every ConfidenceGrowthEpoch blocks in BeginBlocker.
//
// Maturity criteria:
//   - Emerging → Established: confidence ≥ 80%, age ≥ 1 day, ≥ 3 verifications
//   - Established → Canonical: confidence ≥ 90%, age ≥ 10 days, ≥ 10 verifications
//
// In adversarial verification, "verifications" includes:
//   - survived_challenges (each survived challenge = 1 verification)
//   - verifier_count (original verification round participants)
//   - citation_count (being cited = passive verification)
func (k Keeper) AdvanceMaturityLevels(ctx sdk.Context, currentHeight uint64) {
	k.IterateFacts(ctx, func(fact types.Fact) bool {
		if fact.Status != types.FactStatusVerified && fact.Status != types.FactStatusActive {
			return false
		}
		if fact.IsProvisional {
			return false
		}

		// Calculate effective verifications
		effectiveVerifications := uint32(fact.VerifierCount) + fact.SurvivedChallenges
		if fact.CitationCount > 0 {
			// Citations count as partial verifications (each citation = 0.5 verification)
			citationVerifications := uint32(fact.CitationCount / 2)
			if citationVerifications > 5 {
				citationVerifications = 5 // cap citation-based verifications
			}
			effectiveVerifications += citationVerifications
		}

		age := uint64(0)
		if currentHeight > fact.CreatedAtHeight {
			age = currentHeight - fact.CreatedAtHeight
		}

		switch fact.Maturity {
		case types.MaturityEmerging, "":
			if fact.Confidence >= types.MaturityEstablishedMinConf &&
				age >= types.MaturityEstablishedMinAge &&
				effectiveVerifications >= types.MaturityEstablishedMinVerifications {
				fact.Maturity = types.MaturityEstablished
				k.SetFact(ctx, fact)

				ctx.EventManager().EmitEvent(
					sdk.NewEvent("fact_maturity_advanced",
						sdk.NewAttribute("fact_id", fact.Id),
						sdk.NewAttribute("new_maturity", types.MaturityEstablished),
					),
				)
			}

		case types.MaturityEstablished:
			if fact.Confidence >= types.MaturityCanonicalMinConf &&
				age >= types.MaturityCanonicalMinAge &&
				effectiveVerifications >= types.MaturityCanonicalMinVerifications {
				fact.Maturity = types.MaturityCanonical
				k.SetFact(ctx, fact)

				ctx.EventManager().EmitEvent(
					sdk.NewEvent("fact_maturity_advanced",
						sdk.NewAttribute("fact_id", fact.Id),
						sdk.NewAttribute("new_maturity", types.MaturityCanonical),
						sdk.NewAttribute("effective_verifications", fmt.Sprintf("%d", effectiveVerifications)),
					),
				)
			}
		}

		return false
	})
}
