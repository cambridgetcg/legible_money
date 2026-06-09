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
	ep := k.GetExtendedParams(ctx)
	growthPerEpoch := params.ConfidenceGrowthPerEpoch
	if growthPerEpoch == 0 {
		growthPerEpoch = types.ConfidenceGrowthPerEpochDefault
	}

	maxSurvival := params.MaxSurvivalConfidence
	if maxSurvival == 0 {
		maxSurvival = types.MaxSurvivalConfidenceDefault
	}

	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		// Skip non-verified facts
		if fact.Status != types.FactStatusVerified && fact.Status != types.FactStatusActive {
			return false
		}
		// Skip provisional facts (confidence frozen during challenge window)
		if fact.IsProvisional {
			return false
		}

		// CONF-002 guard: facts with no verifiers and no survived challenges have no
		// epistemic backing. Their confidence must not grow beyond base_confidence.
		// A fact verified by zero actors is structurally unanchored — growth without
		// verification is false confidence.
		if fact.VerifierCount == 0 && fact.SurvivedChallenges == 0 {
			if fact.Confidence > fact.BaseConfidence {
				// Drift has already occurred (legacy data). Clamp back to base.
				fact.Confidence = fact.BaseConfidence
				fact.LastConfidenceUpdate = currentHeight
				k.SetFact(ctx, fact)
			}
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
			cap = ep.SurvivedChallengeConfCap // 88%
		}
		if fact.Maturity == types.MaturityEstablished || fact.Maturity == types.MaturityCanonical {
			categoryCeiling := getCategoryCeilingFromParams(fact.EpistemicCategory, ep)
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
		categoryCeiling := getCategoryCeilingFromParams(fact.EpistemicCategory, ep)
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
	ep := k.GetExtendedParams(ctx)
	growthEpoch := ep.ConfidenceGrowthEpochBlocks
	staleThreshold := ep.StaleContestedDecayEpochs * growthEpoch

	k.IterateFacts(ctx, func(fact *types.Fact) bool {
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
		decayAmount := (fact.Confidence * ep.StaleContestedDecayRateBps) / 1_000_000
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

// DetectPropheticFacts scans verified facts for prophetic recognition.
// A fact is prophetic if it was barely accepted (confidence near the acceptance threshold)
// but has since accumulated significant citations — ahead of its time, later vindicated.
// Called every ConfidenceGrowthEpoch blocks in BeginBlocker.
func (k Keeper) DetectPropheticFacts(ctx sdk.Context, currentHeight uint64) {
	ep := k.GetExtendedParams(ctx)
	propheticThreshold := ep.PropheticBaseThresholdBps
	citationThreshold := ep.PropheticCitationThreshold
	propheticCap := ep.PropheticConfidenceCapBps

	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		// Skip if already prophetic or not verified
		if fact.IsProphetic {
			return false
		}
		if fact.Status != types.FactStatusVerified && fact.Status != types.FactStatusActive {
			return false
		}

		// Check: was this fact barely accepted?
		// BaseConfidence represents the structural score at acceptance time.
		// If it was near the acceptance threshold (600k), it was a close call.
		baseConf := fact.BaseConfidence
		if baseConf == 0 {
			baseConf = fact.Confidence
		}
		if baseConf > propheticThreshold {
			return false // wasn't barely accepted
		}

		// Check: has it since gained significant citations?
		if fact.CitationCount < citationThreshold {
			return false
		}

		// FARM-13: Require citation diversity for prophetic status.
		// Prophetic recognition is about community vindication, not self-promotion.
		if !k.ValidatePropheticDiversity(ctx, fact, ep) {
			return false
		}

		// Prophetic! Mark it and apply confidence boost.
		fact.IsProphetic = true

		// Calculate citation growth rate (citations per epoch since creation)
		ageBlocks := currentHeight - fact.CreatedAtHeight
		if ageBlocks > 0 {
			epochBlocks := ep.ConfidenceGrowthEpochBlocks
			epochsAlive := ageBlocks / epochBlocks
			if epochsAlive == 0 {
				epochsAlive = 1
			}
			// Rate in bps where 1,000,000 = 1 citation/epoch
			fact.CitationGrowthRate = fact.CitationCount * 1_000_000 / epochsAlive
		}

		// Boost confidence toward prophetic cap
		if fact.Confidence < propheticCap {
			// Gradual boost: move 10% closer to cap per detection
			gap := propheticCap - fact.Confidence
			boost := gap / 10
			if boost == 0 {
				boost = 1
			}
			fact.Confidence += boost
			fact.LastConfidenceUpdate = currentHeight
		}

		k.SetFact(ctx, fact)

		ctx.EventManager().EmitEvent(
			sdk.NewEvent("prophetic_fact_detected",
				sdk.NewAttribute("fact_id", fact.Id),
				sdk.NewAttribute("base_confidence", fmt.Sprintf("%d", baseConf)),
				sdk.NewAttribute("citation_count", fmt.Sprintf("%d", fact.CitationCount)),
				sdk.NewAttribute("new_confidence", fmt.Sprintf("%d", fact.Confidence)),
			),
		)

		return false
	})
}

// ApplyPropheticGrowth gives prophetic facts accelerated confidence growth
// and an elevated confidence cap. Called alongside ApplyConfidenceGrowth.
func (k Keeper) ApplyPropheticGrowth(ctx sdk.Context, currentHeight uint64) {
	ep := k.GetExtendedParams(ctx)
	propheticCap := ep.PropheticConfidenceCapBps

	params := k.GetParams(ctx)
	growthPerEpoch := params.ConfidenceGrowthPerEpoch
	if growthPerEpoch == 0 {
		growthPerEpoch = types.ConfidenceGrowthPerEpochDefault
	}

	k.IterateFacts(ctx, func(fact *types.Fact) bool {
		if !fact.IsProphetic {
			return false
		}
		if fact.Status != types.FactStatusVerified && fact.Status != types.FactStatusActive {
			return false
		}
		if fact.IsProvisional {
			return false
		}

		// FARM-13: Re-validate prophetic diversity each growth epoch.
		// If citations become concentrated (e.g., sybil accounts banned,
		// citations removed), revoke prophetic status.
		if !k.ValidatePropheticDiversity(ctx, fact, ep) {
			fact.IsProphetic = false
			k.SetFact(ctx, fact)
			ctx.EventManager().EmitEvent(
				sdk.NewEvent("prophetic_status_revoked",
					sdk.NewAttribute("fact_id", fact.Id),
					sdk.NewAttribute("reason", "FARM-13: citation diversity below threshold"),
				),
			)
			return false
		}

		// Prophetic facts get 2× growth rate
		effectiveGrowth := growthPerEpoch * 2

		// Apply survived challenges multiplier (same as regular growth)
		survivalMultiplier := uint64(1_000_000) + uint64(fact.SurvivedChallenges)*500_000
		if survivalMultiplier > 3_000_000 {
			survivalMultiplier = 3_000_000
		}
		effectiveGrowth = (effectiveGrowth * survivalMultiplier) / 1_000_000

		newConfidence := fact.Confidence + effectiveGrowth
		if newConfidence > propheticCap {
			newConfidence = propheticCap
		}

		if newConfidence != fact.Confidence {
			fact.Confidence = newConfidence
			fact.LastConfidenceUpdate = currentHeight
			k.SetFact(ctx, fact)
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
	ep := k.GetExtendedParams(ctx)
	k.IterateFacts(ctx, func(fact *types.Fact) bool {
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
			if fact.Confidence >= ep.MaturityEstablishedMinConf &&
				age >= ep.MaturityEstablishedMinAge &&
				effectiveVerifications >= uint32(ep.MaturityEstablishedMinVerif) {
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
			if fact.Confidence >= ep.MaturityCanonicalMinConf &&
				age >= ep.MaturityCanonicalMinAge &&
				effectiveVerifications >= uint32(ep.MaturityCanonicalMinVerif) {
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

// ValidatePropheticDiversity checks if a fact's citation sources meet the
// diversity requirements for prophetic status (FARM-13).
// Returns true if diversity check is disabled or if citations are sufficiently diverse.
func (k Keeper) ValidatePropheticDiversity(ctx sdk.Context, fact *types.Fact, ep types.ExtendedParams) bool {
	if ep.PropheticMinUniqueCiters == 0 {
		return true // diversity check disabled
	}

	record, found := k.GetCitationSourceRecord(ctx, fact.Id)
	if !found {
		return false // no citation source data → can't verify diversity
	}

	// Check unique citers
	uniqueCiters := uint64(len(record.SourceCounts))
	if uniqueCiters < ep.PropheticMinUniqueCiters {
		return false
	}

	// Check no single address dominates
	if ep.PropheticMaxSingleCiterShare > 0 && record.TotalCites > 0 {
		for _, count := range record.SourceCounts {
			// share = count * 1_000_000 / totalCites
			// Cross-multiply to avoid division: count * 1_000_000 > totalCites * maxShare
			if count*1_000_000 > record.TotalCites*ep.PropheticMaxSingleCiterShare {
				return false
			}
		}
	}

	return true
}
