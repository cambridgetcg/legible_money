package keeper

import (
	"fmt"

	sdkmath "cosmossdk.io/math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/knowledge/types"
)

// AggregateVerificationResult computes the final outcome of a verification round
// using stake-weighted confidence aggregation.
//
// Algorithm:
//  1. Weight each reveal's confidence by the validator's effective stake
//  2. Compute agreement ratio = majority_weight / total_weight
//  3. Determine outcome: accepted, rejected, contested, insufficient, inconclusive
//  4. Calculate rewards/slashes for each participant
func (k Keeper) AggregateVerificationResult(
	ctx sdk.Context,
	round types.VerificationRound,
) (types.VerificationResult, error) {
	params := k.GetParams(ctx)

	// Resolve domain from claim for reputation weighting
	var roundDomain string
	if k.captureDefenseKeeper != nil && round.ClaimId != "" {
		if claim, found := k.GetClaim(ctx, round.ClaimId); found {
			roundDomain = claim.Domain
		}
	}

	// Gather stake weights for all revealing validators, adjusted by domain reputation
	stakeMap := make(map[string]uint64, len(round.Reveals))
	for validatorAddr := range round.Reveals {
		stake := k.GetEffectiveStakeForValidator(ctx, validatorAddr)
		// If capture defense keeper is wired, adjust by domain reputation
		if k.captureDefenseKeeper != nil && roundDomain != "" {
			repScore := k.captureDefenseKeeper.GetReputationScore(ctx, validatorAddr, roundDomain)
			if repScore > 0 {
				// Reputation-weighted: stake * repScore / 1_000_000
				stake = (stake * repScore) / 1_000_000
				if stake == 0 {
					stake = 1
				}
			}
		}
		stakeMap[validatorAddr] = stake
	}

	// Compute stake-weighted verdicts
	var acceptWeight, rejectWeight, abstainWeight, totalWeight uint64
	for validatorAddr, reveal := range round.Reveals {
		stake := stakeMap[validatorAddr]
		if stake == 0 {
			stake = 1 // Prevent zero-weight for Tier 0 validators
		}
		weightedConf := reveal.Confidence * stake / 1_000_000 // normalize

		switch reveal.Verdict {
		case "accept":
			acceptWeight += weightedConf
		case "reject":
			rejectWeight += weightedConf
		case "abstain":
			abstainWeight += weightedConf
		}
		totalWeight += weightedConf
	}

	height := uint64(ctx.BlockHeight())

	result := types.VerificationResult{
		ParticipantCount:  uint32(len(round.Reveals)),
		Rewards:           make(map[string]uint64),
		Slashes:           make(map[string]uint64),
		CompletedAtHeight: height,
	}

	if totalWeight == 0 {
		result.Verdict = "insufficient"
		return result, nil
	}

	// Participation ratio: reveals / selected validators
	participationRatio := uint64(1_000_000) // Default 100% if no selection tracking
	if len(round.SelectedValidators) > 0 {
		participationRatio = (uint64(len(round.Reveals)) * 1_000_000) / uint64(len(round.SelectedValidators))
	}

	// Check quorum
	if participationRatio < params.QuorumThreshold {
		result.Verdict = "insufficient"
		result.Confidence = 0
		return result, nil
	}

	result.QuorumReached = true

	// Determine majority verdict and agreement ratio
	var majorWeight uint64
	var majorVerdict string
	if acceptWeight >= rejectWeight && acceptWeight >= abstainWeight {
		majorWeight = acceptWeight
		majorVerdict = "accepted"
	} else if rejectWeight > acceptWeight && rejectWeight >= abstainWeight {
		majorWeight = rejectWeight
		majorVerdict = "rejected"
	} else {
		majorWeight = abstainWeight
		majorVerdict = "inconclusive"
	}

	agreementRatio := (majorWeight * 1_000_000) / totalWeight
	result.AgreementRatio = agreementRatio
	result.TotalWeight = totalWeight

	// Apply thresholds to determine outcome
	if agreementRatio >= params.AcceptanceThreshold {
		result.Verdict = majorVerdict
		result.Confidence = agreementRatio
	} else if agreementRatio >= 550000 { // 55% contested threshold
		result.Verdict = "contested"
		result.Confidence = agreementRatio
	} else {
		result.Verdict = "inconclusive"
		result.Confidence = agreementRatio
	}

	// Apply stratum ceiling (C11): clamp confidence by domain's stratum
	if roundDomain != "" {
		result.Confidence = k.applyStratumCeiling(ctx, roundDomain, result.Confidence)
	}

	// Apply category ceiling from epistemic profile
	if round.ClaimId != "" {
		if claim, found := k.GetClaim(ctx, round.ClaimId); found {
			if profile, ok := types.CATEGORY_DECAY[claim.EpistemicCategory]; ok {
				if result.Confidence > profile.MaxConfidence {
					result.Confidence = profile.MaxConfidence
				}
			}
		}
	}

	// Apply evidence bonus (additive, capped at MaxEvidenceBonus = 15%)
	// Evidence is queried from x/evidence_mgmt via the cross-module keeper interface.
	// The bonus rewards claims that are supported by high-quality, audited evidence.
	if k.evidenceMgmtKeeper != nil && round.ClaimId != "" {
		evidenceBonus := k.computeEvidenceBonus(ctx, round.ClaimId)
		if evidenceBonus > 0 {
			result.EvidenceBonus = evidenceBonus
			result.Confidence += evidenceBonus
			// Hard cap at 1,000,000 (100%) — confidence cannot exceed certainty
			if result.Confidence > 1_000_000 {
				result.Confidence = 1_000_000
			}
		}
	}

	// Calculate rewards and slashes
	k.calculateRewardsAndSlashes(ctx, round, &result, stakeMap)

	return result, nil
}

// GetEffectiveStakeForValidator returns the effective stake for a validator,
// including virtual stake for Tier 0/1 validators.
func (k Keeper) GetEffectiveStakeForValidator(ctx sdk.Context, validatorAddr string) uint64 {
	return k.stakingKeeper.GetEffectiveStake(ctx, validatorAddr)
}

// calculateRewardsAndSlashes computes reward and slash amounts for each
// participant based on their verdict alignment with the final outcome.
//
// Reward tiers (from TS chain consensus/pot/state_integration.ts):
//   - Correct verdict: base_reward * tier_multiplier
//   - Wrong verdict: slashed proportionally
//   - Missed reveal (committed but didn't reveal): missed_verification slash
func (k Keeper) calculateRewardsAndSlashes(
	ctx sdk.Context,
	round types.VerificationRound,
	result *types.VerificationResult,
	stakeMap map[string]uint64,
) {
	if result.Verdict == "insufficient" || result.Verdict == "inconclusive" {
		return
	}

	// Base reward per correct verification (in ulgm), configurable via params (G-9 fix)
	params := k.GetParams(ctx)
	baseReward := params.VerificationRewardAmount
	if baseReward == 0 {
		baseReward = 100000 // fallback default: 0.1 LGM
	}

	for validatorAddr, reveal := range round.Reveals {
		// Check if validator's verdict aligns with final outcome
		correct := (result.Verdict == "accepted" && reveal.Verdict == "accept") ||
			(result.Verdict == "rejected" && reveal.Verdict == "reject")

		if correct {
			// Apply tier-based reward multiplier (TS: 0.1x/0.5x/1.0x/1.5x)
			reward := baseReward
			if info, found := k.stakingKeeper.GetValidatorInfo(ctx, validatorAddr); found {
				reward = applyTierMultiplier(reward, info.Tier)
			}
			result.Rewards[validatorAddr] = reward
		} else if reveal.Verdict != "abstain" {
			// Slash wrong verdict (abstain is not slashed)
			stake := stakeMap[validatorAddr]
			slashAmount := stake / 100 // 1% slash
			if slashAmount > 0 {
				result.Slashes[validatorAddr] = slashAmount
			}
		}
	}

	// Slash validators who committed but failed to reveal
	for validatorAddr := range round.Commitments {
		if _, revealed := round.Reveals[validatorAddr]; !revealed {
			// Lookup stake for non-revealing validator (not in stakeMap)
			stake := stakeMap[validatorAddr]
			if stake == 0 {
				stake = k.GetEffectiveStakeForValidator(ctx, validatorAddr)
			}
			// Missed reveal penalty (harsher than wrong verdict)
			slashAmount := stake / 50 // 2% slash for missed reveal
			if slashAmount > 0 {
				result.Slashes[validatorAddr] = slashAmount
			}
		}
	}
}

// ApplyRewardsAndSlashes distributes rewards and applies slashes from a
// verification result to validator accounts.
func (k Keeper) ApplyRewardsAndSlashes(
	ctx sdk.Context,
	result types.VerificationResult,
) error {
	params := k.GetParams(ctx)
	researchTaxRate := params.ResearchTaxRate
	if researchTaxRate == 0 {
		researchTaxRate = 220000 // fallback: 22%
	}

	// Apply rewards via BankKeeper (module → validator account)
	for validatorAddr, amount := range result.Rewards {
		if err := k.RecordVerificationOutcome(ctx, validatorAddr, true); err != nil {
			k.Logger(ctx).Error("failed to record correct verification",
				"validator", validatorAddr, "error", err)
		}
		// Transfer reward from module account to validator (after research tax)
		if k.bankKeeper != nil && amount > 0 {
			// Deduct research tax (22% default) and send to research fund
			taxAmount := amount * researchTaxRate / 1_000_000
			validatorAmount := amount - taxAmount

			if taxAmount > 0 {
				taxCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(int64(taxAmount))))
				if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "research", taxCoins); err != nil {
					k.Logger(ctx).Error("failed to send research tax",
						"validator", validatorAddr, "tax", taxAmount, "error", err)
					// On failure, give full reward to validator
					validatorAmount = amount
				}
			}

			if validatorAmount > 0 {
				valAddr, err := sdk.AccAddressFromBech32(validatorAddr)
				if err == nil {
					rewardCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(int64(validatorAmount))))
					if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, valAddr, rewardCoins); err != nil {
						k.Logger(ctx).Error("failed to send reward",
							"validator", validatorAddr, "amount", validatorAmount, "error", err)
					}
				}
			}
		}
	}

	// Apply slashes
	for validatorAddr, amount := range result.Slashes {
		if err := k.SlashMissedVerification(ctx, validatorAddr, fmt.Sprintf("%d", amount)); err != nil {
			k.Logger(ctx).Error("failed to apply slash",
				"validator", validatorAddr, "amount", amount, "error", err)
		}
		if err := k.RecordVerificationOutcome(ctx, validatorAddr, false); err != nil {
			k.Logger(ctx).Error("failed to record incorrect verification",
				"validator", validatorAddr, "error", err)
		}
	}

	return nil
}

// applyStratumCeiling clamps confidence by the domain's stratum ceiling.
// Queries the ontology module first; falls back to hardcoded mapping for bootstrap domains.
func (k Keeper) applyStratumCeiling(ctx sdk.Context, domain string, confidence uint64) uint64 {
	// Try ontology module first (dynamic, user-defined domains)
	if k.ontologyKeeper != nil {
		maxConf, _, err := k.ontologyKeeper.GetConfidenceCeiling(ctx, domain)
		if err == nil && maxConf > 0 {
			if confidence > maxConf {
				return maxConf
			}
			return confidence
		}
		// Fall through to hardcoded on error (domain not in ontology yet)
	}

	// Fallback: hardcoded mapping for bootstrap domains
	domainStratum := inferDomainStratum(domain)
	ceiling, ok := types.STRATUM_CONFIDENCE_CEILING[domainStratum]
	if !ok {
		return confidence // no ceiling for unknown stratum
	}
	if confidence > ceiling {
		return ceiling
	}
	return confidence
}

// inferDomainStratum maps known domains to their ontological stratum.
func inferDomainStratum(domain string) string {
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
		return "technological" // default to highest ceiling
	}
}

// computeEvidenceBonus computes an additive confidence bonus based on evidence
// quality for a claim. The bonus is the average of (typeStrength * reliability)
// across all verified evidence for the claim, capped at MaxEvidenceBonus.
//
// Formula:
//
//	bonus = sum(typeStrength_i * reliability_i / 1_000_000) / count
//	capped at MaxEvidenceBonus (150,000 = 15%)
//
// Only evidence with status "verified" or "submitted" is considered.
// Unaudited evidence (reliability=0) contributes at a base reliability of 10%.
func (k Keeper) computeEvidenceBonus(ctx sdk.Context, claimId string) uint64 {
	evidences := k.evidenceMgmtKeeper.GetEvidenceByClaimId(ctx, claimId)
	if len(evidences) == 0 {
		return 0
	}

	var totalScore uint64
	var validCount uint64

	for _, ev := range evidences {
		// Skip rejected or expired evidence
		if ev.Status == "rejected" || ev.Status == "expired" {
			continue
		}

		typeStrength, ok := types.EVIDENCE_TYPE_STRENGTH[ev.EvidenceType]
		if !ok {
			typeStrength = 400_000 // conservative default for unknown types (0.4)
		}

		reliability := ev.Reliability
		if reliability == 0 {
			// Unaudited evidence: assign base reliability (10%)
			// so submitting evidence still matters before audits complete
			reliability = 100_000
		}

		// Score = typeStrength * reliability / 1_000_000
		score := (typeStrength * reliability) / 1_000_000
		totalScore += score
		validCount++
	}

	if validCount == 0 {
		return 0
	}

	// Average across all valid evidence pieces
	bonus := totalScore / validCount

	// Cap at MaxEvidenceBonus
	if bonus > types.MaxEvidenceBonus {
		bonus = types.MaxEvidenceBonus
	}

	k.Logger(ctx).Debug("evidence bonus computed",
		"claim_id", claimId,
		"evidence_count", validCount,
		"bonus", bonus,
	)

	return bonus
}

// applyTierMultiplier scales a reward by the validator's tier.
// Tier 0 (Apprentice): 10%, Tier 1 (Verified): 50%, Tier 2 (Bonded): 100%, Tier 3 (Guardian): 150%.
func applyTierMultiplier(baseReward uint64, tier uint32) uint64 {
	switch tier {
	case 0:
		return baseReward / 10 // 0.1x
	case 1:
		return baseReward / 2 // 0.5x
	case 2:
		return baseReward // 1.0x
	case 3:
		return baseReward * 3 / 2 // 1.5x
	default:
		return baseReward
	}
}
