package keeper

import (
	"math"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_defense/types"
)

// CalculateDomainReputationScore computes a composite reputation score for a validator in a domain.
// Score components (basis points): accuracy (50%), consistency (25%), challenge defense (15%), calibration (10%)
func CalculateDomainReputationScore(
	totalVerifications, correctVerifications uint64,
	verdictConsistency, challengesReceived, challengesLost uint64,
	confidenceCalibration uint64,
	currentBlock, lastActivityBlock uint64,
) (score, decayedScore, scoreConfidence uint64) {
	if totalVerifications == 0 {
		return types.BaseScore, types.BaseScore, 0
	}

	// Accuracy component (with Laplace smoothing: (correct+1)/(total+2))
	accuracyScore := ((correctVerifications + 1) * types.MaxScore) / (totalVerifications + 2)

	// Consistency component
	consistencyScore := verdictConsistency * types.MaxScore / 1_000_000

	// Challenge defense component
	var challengeScore uint64
	if challengesReceived == 0 {
		challengeScore = types.MaxScore // No challenges = perfect defense
	} else {
		defended := challengesReceived - challengesLost
		challengeScore = (defended * types.MaxScore) / challengesReceived
	}

	// Calibration component
	calibrationScore := confidenceCalibration * types.MaxScore / 1_000_000

	// Weighted composite
	score = (accuracyScore*types.AccuracyWeight +
		consistencyScore*types.ConsistencyWeight +
		challengeScore*types.ChallengeDefenseWeight +
		calibrationScore*types.CalibrationWeight) / 1_000_000

	if score > types.MaxScore {
		score = types.MaxScore
	}

	// Time decay: decayedScore = baseScore + (score - baseScore) * 0.5^(age/halfLife)
	decayedScore = applyDecay(score, currentBlock, lastActivityBlock)

	// Score confidence increases with verification count (saturates at ~100)
	scoreConfidence = min64(1_000_000, totalVerifications*1_000_000/100)

	return score, decayedScore, scoreConfidence
}

// applyDecay applies exponential half-life decay to a score.
func applyDecay(score, currentBlock, lastActivityBlock uint64) uint64 {
	if currentBlock <= lastActivityBlock {
		return score
	}
	age := currentBlock - lastActivityBlock
	if age == 0 {
		return score
	}

	halfLife := float64(types.ReputationHalfLifeBlocks)
	decayFactor := math.Pow(0.5, float64(age)/halfLife)

	base := float64(types.BaseScore)
	decayed := base + (float64(score)-base)*decayFactor

	result := uint64(decayed)
	if result < types.MinDecayedScore {
		result = types.MinDecayedScore
	}
	return result
}

// GetEffectiveReputation returns the effective reputation for a validator in a domain,
// using the domain→stratum→global fallback chain.
func (k Keeper) GetEffectiveReputation(ctx sdk.Context, validatorAddr, domain string) types.EffectiveReputationResult {
	currentBlock := uint64(ctx.BlockHeight())
	rep := k.GetOrCreateReputation(ctx, validatorAddr, currentBlock)

	// Try domain-specific reputation first
	if domainRep, ok := rep.DomainReputations[domain]; ok {
		if domainRep.TotalVerifications >= types.MinVerificationsForDomainRep {
			return types.EffectiveReputationResult{
				Score:      domainRep.DecayedScore,
				Source:     types.SourceDomain,
				Discount:   1_000_000, // 100% = no discount
				Confidence: domainRep.ScoreConfidence,
			}
		}
	}

	// Try stratum reputation
	stratum := k.getDomainStratum(ctx, domain)
	if stratumRep, ok := rep.StratumReputations[stratum]; ok {
		if stratumRep.TotalVerifications >= types.MinVerificationsForDomainRep {
			decayed := applyDecay(stratumRep.Score, currentBlock, stratumRep.LastActivityBlock)
			conf := min64(1_000_000, stratumRep.TotalVerifications*1_000_000/100)
			return types.EffectiveReputationResult{
				Score:      decayed,
				Source:     types.SourceStratum,
				Discount:   types.StratumReputationDiscount,
				Confidence: conf,
			}
		}
	}

	// Fall back to global reputation
	globalDecayed := applyDecay(rep.GlobalReputation.Score, currentBlock, rep.LastUpdatedBlock)
	conf := min64(1_000_000, rep.GlobalReputation.TotalVerifications*1_000_000/100)
	return types.EffectiveReputationResult{
		Score:      globalDecayed,
		Source:     types.SourceGlobal,
		Discount:   types.GlobalReputationDiscount,
		Confidence: conf,
	}
}

// GetReputationScore returns the effective reputation score for a validator in a domain,
// with discount applied. Returns basis points (0-1,000,000).
func (k Keeper) GetReputationScore(ctx sdk.Context, validatorAddr, domain string) uint64 {
	result := k.GetEffectiveReputation(ctx, validatorAddr, domain)
	// Apply source discount: score * discount / 1_000_000
	return (result.Score * result.Discount) / 1_000_000
}

// UpdateDomainReputation updates a validator's reputation after a verification.
func (k Keeper) UpdateDomainReputation(
	ctx sdk.Context,
	validatorAddr, domain string,
	stratum uint32,
	wasCorrect bool,
	verdictConf, finalConf uint64,
	currentBlock uint64,
) {
	rep := k.GetOrCreateReputation(ctx, validatorAddr, currentBlock)

	// Update global reputation
	rep.GlobalReputation.TotalVerifications++
	if wasCorrect {
		rep.GlobalReputation.CorrectVerifications++
	}
	if rep.GlobalReputation.TotalVerifications > 0 {
		rep.GlobalReputation.Accuracy = (rep.GlobalReputation.CorrectVerifications * 1_000_000) / rep.GlobalReputation.TotalVerifications
	}
	rep.GlobalReputation.Score = computeGlobalScore(rep.GlobalReputation)

	// Update stratum reputation
	sr, ok := rep.StratumReputations[stratum]
	if !ok {
		sr = types.StratumReputation{
			Stratum:                stratum,
			Score:                  types.BaseScore,
			FirstVerificationBlock: currentBlock,
			LastDecayBlock:         currentBlock,
		}
	}
	sr.TotalVerifications++
	if wasCorrect {
		sr.CorrectVerifications++
	}
	if sr.TotalVerifications > 0 {
		sr.Accuracy = (sr.CorrectVerifications * 1_000_000) / sr.TotalVerifications
	}
	sr.Score = (sr.Accuracy * types.MaxScore) / 1_000_000
	sr.LastActivityBlock = currentBlock
	rep.StratumReputations[stratum] = sr

	// Update domain reputation
	dr, ok := rep.DomainReputations[domain]
	if !ok {
		dr = types.DomainReputation{
			Domain:                 domain,
			Stratum:                stratum,
			Score:                  types.BaseScore,
			FirstVerificationBlock: currentBlock,
			LastDecayBlock:         currentBlock,
			VerdictConsistency:     700_000, // Start at 70%
			ConfidenceCalibration:  500_000, // Start at 50%
		}
	}
	dr.TotalVerifications++
	if wasCorrect {
		dr.CorrectVerifications++
	}
	if dr.TotalVerifications > 0 {
		dr.Accuracy = (dr.CorrectVerifications * 1_000_000) / dr.TotalVerifications
	}

	// Update confidence calibration: how close is validator's confidence to final?
	if finalConf > 0 {
		var diff uint64
		if verdictConf > finalConf {
			diff = verdictConf - finalConf
		} else {
			diff = finalConf - verdictConf
		}
		calibration := uint64(1_000_000) - diff
		// Exponential moving average with alpha=0.1 (approximated)
		dr.ConfidenceCalibration = (dr.ConfidenceCalibration*9 + calibration) / 10
	}

	dr.Score, dr.DecayedScore, dr.ScoreConfidence = CalculateDomainReputationScore(
		dr.TotalVerifications, dr.CorrectVerifications,
		dr.VerdictConsistency, dr.ChallengesReceived, dr.ChallengesLost,
		dr.ConfidenceCalibration,
		currentBlock, dr.LastVerificationBlock,
	)
	dr.LastVerificationBlock = currentBlock
	rep.DomainReputations[domain] = dr

	rep.LastUpdatedBlock = currentBlock
	k.SetIsolatedReputation(ctx, rep)
}

// ApplyPeriodicDecay decays all reputations toward the base score.
func (k Keeper) ApplyPeriodicDecay(ctx sdk.Context, currentBlock uint64) {
	k.IterateReputations(ctx, func(rep types.IsolatedReputation) bool {
		changed := false

		// Decay global
		decayed := applyDecay(rep.GlobalReputation.Score, currentBlock, rep.LastUpdatedBlock)
		if decayed != rep.GlobalReputation.Score {
			rep.GlobalReputation.Score = decayed
			changed = true
		}

		// Decay stratum reputations
		for s, sr := range rep.StratumReputations {
			d := applyDecay(sr.Score, currentBlock, sr.LastActivityBlock)
			if d != sr.Score {
				sr.Score = d
				sr.LastDecayBlock = currentBlock
				rep.StratumReputations[s] = sr
				changed = true
			}
		}

		// Decay domain reputations
		for dom, dr := range rep.DomainReputations {
			d := applyDecay(dr.Score, currentBlock, dr.LastVerificationBlock)
			if d != dr.DecayedScore {
				dr.DecayedScore = d
				dr.LastDecayBlock = currentBlock
				rep.DomainReputations[dom] = dr
				changed = true
			}
		}

		if changed {
			rep.LastUpdatedBlock = currentBlock
			k.SetIsolatedReputation(ctx, rep)
		}
		return false
	})
}

// --- Cross-Stratum Validation ---

// ValidateCrossStratumCoverage checks if the validator set covers required strata
// for a claim at a given stratum level.
func (k Keeper) ValidateCrossStratumCoverage(
	ctx sdk.Context,
	claimStratum uint32,
	validatorStrata map[string]uint32,
) (bool, uint64) {
	req := getCrossStratumRequirement(claimStratum)
	if req == nil {
		return true, 1_000_000 // No requirements for this stratum
	}

	// Check each required stratum has enough validators
	stratumCounts := make(map[uint32]uint32)
	for _, s := range validatorStrata {
		stratumCounts[s]++
	}

	allMet := true
	for _, reqStratum := range req.RequiredStrata {
		minCount, ok := req.MinValidatorsPerStratum[reqStratum]
		if !ok {
			minCount = 1
		}
		if stratumCounts[reqStratum] < minCount {
			allMet = false
			break
		}
	}

	if allMet {
		return true, 1_000_000
	}

	// Partial coverage — apply confidence penalty
	return true, 1_000_000 - types.PartialCoverageConfPenalty
}

// getCrossStratumRequirement returns requirements for a claim at a given stratum.
// Stratum IDs: 0=technological, 1=social, 2=cognitive, 3=biological, 4=chemical, 5=physical, 6=fundamental
func getCrossStratumRequirement(stratum uint32) *types.CrossStratumRequirement {
	switch stratum {
	case 0: // technological
		return &types.CrossStratumRequirement{
			ClaimStratum:            0,
			RequiredStrata:          []uint32{0},
			MinValidatorsPerStratum: map[uint32]uint32{0: 3},
			LowerStratumWeightBonus: 1_000_000,
		}
	case 1: // social
		return &types.CrossStratumRequirement{
			ClaimStratum:            1,
			RequiredStrata:          []uint32{1, 2},
			MinValidatorsPerStratum: map[uint32]uint32{1: 2, 2: 1},
			LowerStratumWeightBonus: 1_100_000,
		}
	case 2: // cognitive
		return &types.CrossStratumRequirement{
			ClaimStratum:            2,
			RequiredStrata:          []uint32{2, 3},
			MinValidatorsPerStratum: map[uint32]uint32{2: 2, 3: 1},
			LowerStratumWeightBonus: 1_100_000,
		}
	case 3: // biological
		return &types.CrossStratumRequirement{
			ClaimStratum:            3,
			RequiredStrata:          []uint32{3, 4},
			MinValidatorsPerStratum: map[uint32]uint32{3: 2, 4: 2},
			LowerStratumWeightBonus: 1_150_000,
		}
	case 4: // chemical
		return &types.CrossStratumRequirement{
			ClaimStratum:            4,
			RequiredStrata:          []uint32{4, 5},
			MinValidatorsPerStratum: map[uint32]uint32{4: 2, 5: 2},
			LowerStratumWeightBonus: 1_150_000,
		}
	case 5: // physical
		return &types.CrossStratumRequirement{
			ClaimStratum:            5,
			RequiredStrata:          []uint32{5, 6},
			MinValidatorsPerStratum: map[uint32]uint32{5: 2, 6: 1},
			LowerStratumWeightBonus: 1_200_000,
		}
	case 6: // fundamental
		return &types.CrossStratumRequirement{
			ClaimStratum:            6,
			RequiredStrata:          []uint32{6},
			MinValidatorsPerStratum: map[uint32]uint32{6: 3},
			LowerStratumWeightBonus: 1_000_000,
		}
	}
	return nil
}

// computeGlobalScore computes a simple global reputation score from accuracy.
func computeGlobalScore(g types.GlobalReputation) uint64 {
	if g.TotalVerifications == 0 {
		return types.BaseScore
	}
	return (g.Accuracy * types.MaxScore) / 1_000_000
}

func min64(a, b uint64) uint64 {
	if a < b {
		return a
	}
	return b
}
