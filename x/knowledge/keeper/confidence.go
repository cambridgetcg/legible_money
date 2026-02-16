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
		Domain:            roundDomain,
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

	// AH-4: Method diversity check — penalize suspiciously uniform verification.
	// Applied after all other confidence adjustments but before rewards.
	if len(round.Reveals) >= types.MinDiversityParticipants {
		diversityAdj := computeMethodDiversityAdjustment(round.Reveals)
		if diversityAdj != 0 {
			result.MethodDiversityPenalty = diversityAdj
			if diversityAdj < 0 {
				penalty := uint64(-diversityAdj)
				if result.Confidence > penalty {
					result.Confidence -= penalty
				} else {
					result.Confidence = 0
				}
			} else {
				result.Confidence += uint64(diversityAdj)
				if result.Confidence > 1_000_000 {
					result.Confidence = 1_000_000
				}
			}
		}
		// Record unique method count for transparency
		result.UniqueMethodCount = countUniqueMethods(round.Reveals)
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
		baseReward = 1000000 // fallback default: 1.0 LGM
	}

	// Slash rates from params (basis points / 1,000,000), aligned with TS prototype:
	// TS wrongVerificationSlash: 0.06 (6%) → 60000 bps
	// TS missedVerificationSlash: 0.0001 (0.01%) → 100 bps (Cosmos-aligned downtime)
	wrongSlashBps := params.WrongVerificationSlashBps
	if wrongSlashBps == 0 {
		wrongSlashBps = 60000 // fallback: 6% (matches TS wrongVerificationSlash)
	}
	missedSlashBps := params.MissedRevealSlashBps
	if missedSlashBps == 0 {
		missedSlashBps = 100 // fallback: 0.01% (Cosmos-aligned downtime penalty)
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
			// Difficulty bonus: harder verifications (lower confidence) pay more.
			// A verifier who correctly evaluates a borderline claim did more
			// valuable work than one who rubber-stamped an obvious truth.
			// bonus = reward * (1_000_000 - avgConfidence) / 1_000_000
			// → at confidence 500k: +50%, at 900k: +10%, at 300k: +70%
			if result.Confidence > 0 && result.Confidence < 1_000_000 {
				difficulty := 1_000_000 - result.Confidence
				reward = reward + reward*difficulty/1_000_000
			}
			result.Rewards[validatorAddr] = reward
		} else if reveal.Verdict != "abstain" {
			// Slash wrong verdict: 6% of stake (abstain is not slashed)
			stake := stakeMap[validatorAddr]
			slashAmount := stake * wrongSlashBps / 1_000_000
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
			// Missed reveal penalty: 0.01% of stake (Cosmos-aligned downtime)
			slashAmount := stake * missedSlashBps / 1_000_000
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
		if k.verificationHooks != nil {
			if err := k.verificationHooks.AfterVerificationCompleted(ctx, validatorAddr, result.Domain, true); err != nil {
				k.Logger(ctx).Error("verification hook failed for correct verification",
					"validator", validatorAddr, "error", err)
			}
		}
		// Transfer reward from module account to validator (after research tax)
		if k.bankKeeper != nil && amount > 0 {
			// Deduct research tax (22% default) and send to research fund
			taxAmount := amount * researchTaxRate / 1_000_000
			validatorAmount := amount - taxAmount

			if taxAmount > 0 {
				taxCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(int64(taxAmount))))
				if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "research_fund", taxCoins); err != nil {
					// Research tax is a protocol invariant (22% of all verification rewards).
					// Silent bypass would let validators receive full rewards, corrupting economics.
					// Withhold entire reward — the module retains funds for retry next round.
					k.Logger(ctx).Error("research tax transfer failed — reward withheld",
						"validator", validatorAddr, "tax", taxAmount, "reward", amount, "error", err)
					continue
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
		if k.verificationHooks != nil {
			if err := k.verificationHooks.AfterVerificationCompleted(ctx, validatorAddr, result.Domain, false); err != nil {
				k.Logger(ctx).Error("verification hook failed for incorrect verification",
					"validator", validatorAddr, "error", err)
			}
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
// Tier 0 (Apprentice): 10%, Tier 1 (Verified): 50%, Tier 2 (Bonded): 100%, Tier 3 (Guardian): 200%.
func applyTierMultiplier(baseReward uint64, tier uint32) uint64 {
	switch tier {
	case 0:
		return baseReward / 10 // 0.1x
	case 1:
		return baseReward / 2 // 0.5x
	case 2:
		return baseReward // 1.0x
	case 3:
		return baseReward * 2 // 2.0x
	default:
		return baseReward
	}
}

// computeMethodDiversityAdjustment detects suspiciously uniform verification
// results and computes a confidence adjustment (negative = penalty, positive = bonus).
//
// The check only activates when at least one reveal explicitly declares a
// VerificationMethod. If all reveals omit the method (legacy mode / deterministic
// pipeline), no penalty or bonus is applied — identical confidence values are
// expected in deterministic evaluation.
//
// When methods are declared:
//  1. Confidence uniformity + single declared method: full penalty
//     (UniformityPenaltyBps + SingleMethodPenaltyBps = -15%).
//  2. Single declared method but varied confidence: partial penalty
//     (SingleMethodPenaltyBps = -5%).
//  3. Two distinct methods: neutral (no adjustment).
//  4. Three or more distinct methods: diversity bonus (DiversityBonusBps = +3%).
func computeMethodDiversityAdjustment(reveals map[string]*types.VerificationReveal) int64 {
	if len(reveals) < types.MinDiversityParticipants {
		return 0
	}

	// Count how many reveals declare an explicit method
	var declaredCount int
	declaredMethods := make(map[string]int, len(reveals))
	for _, reveal := range reveals {
		if reveal.VerificationMethod != "" {
			declaredCount++
			declaredMethods[reveal.VerificationMethod]++
		}
	}

	// Legacy mode: no methods declared at all → no adjustment
	if declaredCount == 0 {
		return 0
	}

	var adjustment int64
	uniqueDeclared := uint32(len(declaredMethods))

	// Check confidence uniformity (only penalize when combined with method concerns)
	confidenceValues := make(map[uint64]struct{}, len(reveals))
	for _, reveal := range reveals {
		confidenceValues[reveal.Confidence] = struct{}{}
	}
	allIdenticalConfidence := len(confidenceValues) == 1

	if uniqueDeclared == 1 {
		// All declaring validators use the same method — suspicious
		adjustment -= int64(types.SingleMethodPenaltyBps)
		if allIdenticalConfidence {
			// Same method AND identical confidence — very suspicious
			adjustment -= int64(types.UniformityPenaltyBps)
		}
	} else if uniqueDeclared >= 3 {
		// Genuine methodological diversity — reward
		adjustment += int64(types.DiversityBonusBps)
	}
	// uniqueDeclared == 2: neutral

	return adjustment
}

// countUniqueMethods returns the number of distinct verification methods
// declared across all reveals. Empty/omitted methods are counted as one group.
func countUniqueMethods(reveals map[string]*types.VerificationReveal) uint32 {
	methods := make(map[string]struct{}, len(reveals))
	for _, reveal := range reveals {
		method := reveal.VerificationMethod
		if method == "" {
			method = "_undeclared"
		}
		methods[method] = struct{}{}
	}
	return uint32(len(methods))
}
