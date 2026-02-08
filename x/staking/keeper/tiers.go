package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/staking/types"
)

// ComputeValidatorTier calculates the appropriate tier based on stake and performance
// Translated from consensus/pot/validator_tiers.ts
func ComputeValidatorTier(
	ctx sdk.Context,
	k Keeper,
	stake string,
	verifications uint64,
	correctVerifications uint64,
	slashCount uint64,
) types.ValidatorTier {
	stakeInt := new(big.Int)
	stakeInt.SetString(stake, 10)

	// Calculate accuracy (0-1000000 basis points)
	var accuracy uint64
	if verifications > 0 {
		accuracy = (correctVerifications * 1000000) / verifications
	}

	// Check tiers from highest to lowest (Guardian -> Apprentice)
	tierConfigs := k.GetAllTierConfigs(ctx)

	// Tier 3 (Guardian): 11111 LGM, 1111 verifications, 77% accuracy, 0 slashes
	guardianConfig := tierConfigs[3]
	minStakeGuardian := new(big.Int)
	minStakeGuardian.SetString(guardianConfig.MinStake, 10)

	if stakeInt.Cmp(minStakeGuardian) >= 0 &&
		verifications >= guardianConfig.MinVerifications &&
		accuracy >= guardianConfig.MinAccuracy &&
		(guardianConfig.MaxSlashCount >= 0 && slashCount <= uint64(guardianConfig.MaxSlashCount)) {
		return types.TierGuardian
	}

	// Tier 2 (Bonded): 1111 LGM stake only
	bondedConfig := tierConfigs[2]
	minStakeBonded := new(big.Int)
	minStakeBonded.SetString(bondedConfig.MinStake, 10)

	if stakeInt.Cmp(minStakeBonded) >= 0 {
		return types.TierBonded
	}

	// Tier 1 (Verified): 22 verifications at 77% accuracy
	verifiedConfig := tierConfigs[1]
	if verifications >= verifiedConfig.MinVerifications &&
		accuracy >= verifiedConfig.MinAccuracy {
		return types.TierVerified
	}

	// Tier 0 (Apprentice): default
	return types.TierApprentice
}

// IsTierEligibleForCategory checks if a tier can verify a given epistemic category
func IsTierEligibleForCategory(
	ctx sdk.Context,
	k Keeper,
	tier types.ValidatorTier,
	category string,
) bool {
	config, found := k.GetTierConfig(ctx, tier)
	if !found {
		return false
	}

	for _, allowed := range config.AllowedCategories {
		if allowed == category {
			return true
		}
	}

	return false
}

// GetEffectiveSelectionStake returns the stake used for VRF selection probability
// Tiers 0 and 1 use virtual stake, Tiers 2 and 3 use real stake
func GetEffectiveSelectionStake(
	ctx sdk.Context,
	k Keeper,
	validator types.LegibleValidator,
	params types.Params,
) string {
	// Tier 0 and 1: use virtual stake (111 LGM)
	if validator.Tier == types.TierApprentice || validator.Tier == types.TierVerified {
		return params.VirtualStake
	}

	// Tier 2 and 3: use actual total stake
	return validator.TotalStake
}

// CalculateTierReward applies the tier reward multiplier to a base reward
// Returns adjusted reward as string (bigint)
func CalculateTierReward(
	ctx sdk.Context,
	k Keeper,
	tier types.ValidatorTier,
	baseReward string,
) string {
	config, found := k.GetTierConfig(ctx, tier)
	if !found {
		return baseReward
	}

	baseInt := new(big.Int)
	baseInt.SetString(baseReward, 10)

	// Apply multiplier (basis points: 1000 = 1.0x)
	multiplier := big.NewInt(int64(config.RewardMultiplier))
	divisor := big.NewInt(1000)

	adjusted := new(big.Int).Mul(baseInt, multiplier)
	adjusted.Div(adjusted, divisor)

	return adjusted.String()
}

// CalculateTierSlash applies tier-based slash multiplier
// Higher tiers get slashed more heavily for violations
func CalculateTierSlash(
	ctx sdk.Context,
	k Keeper,
	tier types.ValidatorTier,
	baseSlash string,
) string {
	// Slash multipliers (inverse of reward multipliers)
	// Tier 0 (Apprentice): 1.5x slash (less stake to lose, higher penalty)
	// Tier 1 (Verified): 1.2x slash
	// Tier 2 (Bonded): 1.0x slash (standard)
	// Tier 3 (Guardian): 1.0x slash (already lose tier on any slash)

	baseInt := new(big.Int)
	baseInt.SetString(baseSlash, 10)

	var multiplier int64
	switch tier {
	case types.TierApprentice:
		multiplier = 1500 // 1.5x
	case types.TierVerified:
		multiplier = 1200 // 1.2x
	case types.TierBonded:
		multiplier = 1000 // 1.0x
	case types.TierGuardian:
		multiplier = 1000 // 1.0x (but loses tier immediately)
	default:
		multiplier = 1000
	}

	mult := big.NewInt(multiplier)
	divisor := big.NewInt(1000)

	adjusted := new(big.Int).Mul(baseInt, mult)
	adjusted.Div(adjusted, divisor)

	return adjusted.String()
}

// CheckTierUpgrade checks if validator qualifies for tier upgrade
func CheckTierUpgrade(
	ctx sdk.Context,
	k Keeper,
	validator types.LegibleValidator,
) (types.ValidatorTier, bool) {
	currentTier := validator.Tier
	potentialTier := ComputeValidatorTier(
		ctx,
		k,
		validator.TotalStake,
		validator.VerificationsCompleted,
		validator.VerificationsCorrect,
		validator.SlashCount,
	)

	if potentialTier > currentTier {
		return potentialTier, true
	}

	return currentTier, false
}

// CheckTierDowngrade checks if validator should be demoted
func CheckTierDowngrade(
	ctx sdk.Context,
	k Keeper,
	validator types.LegibleValidator,
) (types.ValidatorTier, bool) {
	currentTier := validator.Tier
	potentialTier := ComputeValidatorTier(
		ctx,
		k,
		validator.TotalStake,
		validator.VerificationsCompleted,
		validator.VerificationsCorrect,
		validator.SlashCount,
	)

	if potentialTier < currentTier {
		return potentialTier, true
	}

	return currentTier, false
}

// GetTierRequirements returns the requirements for a specific tier (for UI/CLI)
func GetTierRequirements(ctx sdk.Context, k Keeper, tier types.ValidatorTier) types.TierConfig {
	config, found := k.GetTierConfig(ctx, tier)
	if !found {
		// Return empty config if not found
		return types.TierConfig{Tier: tier}
	}
	return config
}

// ValidateTransition checks if a tier transition is valid
func ValidateTransition(
	ctx sdk.Context,
	k Keeper,
	fromTier types.ValidatorTier,
	toTier types.ValidatorTier,
	validator types.LegibleValidator,
) error {
	// Can't skip tiers on upgrade (must be sequential)
	if toTier > fromTier && toTier-fromTier > 1 {
		return types.ErrInvalidTier
	}

	// Check if validator meets requirements for target tier
	computedTier := ComputeValidatorTier(
		ctx,
		k,
		validator.TotalStake,
		validator.VerificationsCompleted,
		validator.VerificationsCorrect,
		validator.SlashCount,
	)

	if computedTier < toTier {
		return types.ErrTierRequirementsNotMet
	}

	return nil
}
