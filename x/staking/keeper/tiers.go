package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/staking/types"
)

// ComputeValidatorTier calculates the appropriate tier based on stake and performance.
// Supports windowed slash counts, contested verification multipliers, and minimum
// contested verification requirements (Strategies A, B, D).
func ComputeValidatorTier(
	ctx sdk.Context,
	k Keeper,
	stake string,
	verifications uint64,
	correctVerifications uint64,
	slashCount uint64,
	contestedCorrect uint64,
	lastSlashHeight uint64,
) types.ValidatorTier {
	stakeInt := new(big.Int)
	stakeInt.SetString(stake, 10)

	currentHeight := uint64(ctx.BlockHeight())
	// P0-3 fix: Read epoch length from params instead of hardcoding.
	// Guardian slash window uses SlashDecayPeriodBlocks (34,272 blocks = ~1 day),
	// not BLOCKS_PER_EPOCH (1,111). The hardcoded value made the window 31x shorter.
	params := k.GetParams(ctx)
	epochLength := params.GetSlashDecayPeriodBlocks()
	if epochLength == 0 {
		epochLength = 34272 // fallback default
	}

	// Calculate accuracy (0-1000000 basis points)
	var accuracy uint64
	if verifications > 0 {
		accuracy = (correctVerifications * 1000000) / verifications
	}

	// Check tiers from highest to lowest (Guardian -> Apprentice)
	tierConfigs := k.GetAllTierConfigs(ctx)

	// Tier 3 (Guardian): 11,111 LGM, 333 verifications (33 contested), 77% accuracy,
	// 0 slashes in rolling 10-epoch window, contested verifications count 3x
	guardianConfig := tierConfigs[3]
	minStakeGuardian := new(big.Int)
	minStakeGuardian.SetString(guardianConfig.MinStake, 10)

	windowedSlashes := getWindowedSlashCount(slashCount, lastSlashHeight, currentHeight, epochLength, guardianConfig.SlashWindowEpochs)
	effectiveVerifications := getEffectiveVerifications(verifications, contestedCorrect, guardianConfig.ContestedVerificationMultiplier)

	if stakeInt.Cmp(minStakeGuardian) >= 0 &&
		effectiveVerifications >= guardianConfig.MinVerifications &&
		accuracy >= guardianConfig.MinAccuracy &&
		(guardianConfig.MaxSlashCount < 0 || windowedSlashes <= uint64(guardianConfig.MaxSlashCount)) &&
		contestedCorrect >= guardianConfig.MinContestedVerifications {
		return types.TierGuardian
	}

	// Tier 2 (Bonded): 1,111 LGM stake + 11 verifications at 50% accuracy
	bondedConfig := tierConfigs[2]
	minStakeBonded := new(big.Int)
	minStakeBonded.SetString(bondedConfig.MinStake, 10)

	if stakeInt.Cmp(minStakeBonded) >= 0 &&
		verifications >= bondedConfig.MinVerifications &&
		(bondedConfig.MinAccuracy == 0 || accuracy >= bondedConfig.MinAccuracy) {
		return types.TierBonded
	}

	// Tier 1 (Verified): 1.11 LGM stake + 22 verifications at 77% accuracy
	verifiedConfig := tierConfigs[1]
	minStakeVerified := new(big.Int)
	minStakeVerified.SetString(verifiedConfig.MinStake, 10)

	if stakeInt.Cmp(minStakeVerified) >= 0 &&
		verifications >= verifiedConfig.MinVerifications &&
		accuracy >= verifiedConfig.MinAccuracy {
		return types.TierVerified
	}

	// Tier 0 (Apprentice): default
	return types.TierApprentice
}

// getWindowedSlashCount returns the slash count relevant for tier eligibility.
// If slashWindowEpochs is 0 or lastSlashHeight is 0, returns total slashCount.
// Otherwise, returns 0 if last slash is outside the window.
func getWindowedSlashCount(slashCount, lastSlashHeight, currentHeight, epochLength, slashWindowEpochs uint64) uint64 {
	if slashWindowEpochs == 0 || lastSlashHeight == 0 {
		return slashCount
	}
	windowBlocks := slashWindowEpochs * epochLength
	if currentHeight > lastSlashHeight && (currentHeight-lastSlashHeight) > windowBlocks {
		return 0
	}
	return slashCount
}

// getEffectiveVerifications computes effective verification count with contested multiplier.
func getEffectiveVerifications(totalVerifications, contestedCorrect, multiplier uint64) uint64 {
	if multiplier <= 1 {
		return totalVerifications
	}
	nonContested := totalVerifications - contestedCorrect
	bonus := multiplier - 1
	return nonContested + contestedCorrect*(1+bonus)
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

// GetEffectiveSelectionStake returns the stake used for VRF selection probability.
// Tiers 0 and 1 use virtual stake, Tiers 2 and 3 use real stake.
// R3 remediation: Returns "0" if validator's real stake is below MinStakeForVerification,
// effectively excluding them from VRF verification selection.
func GetEffectiveSelectionStake(
	ctx sdk.Context,
	k Keeper,
	validator *types.LegibleValidator,
	params *types.Params,
) string {
	// R3: Minimum real stake check for VRF eligibility
	minStake := params.GetMinStakeForVerification()
	if minStake != "" && minStake != "0" {
		totalStake := new(big.Int)
		totalStake.SetString(validator.TotalStake, 10)
		minStakeInt := new(big.Int)
		minStakeInt.SetString(minStake, 10)
		if totalStake.Cmp(minStakeInt) < 0 {
			return "0" // Excluded from VRF selection
		}
	}

	// Tier 0 and 1: use virtual stake (11 LGM)
	if validator.Tier == types.ValidatorTierString(types.TierApprentice) || validator.Tier == types.ValidatorTierString(types.TierVerified) {
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

// CalculateTierSlash applies tier-based slash multiplier from TierConfig.
// Higher tiers get slashed more heavily for violations.
func CalculateTierSlash(
	ctx sdk.Context,
	k Keeper,
	tier types.ValidatorTier,
	baseSlash string,
) string {
	baseInt := new(big.Int)
	baseInt.SetString(baseSlash, 10)

	// Read SlashMultiplierBps from tier config; default 1.0x if not found
	multiplier := int64(1000)
	config, found := k.GetTierConfig(ctx, tier)
	if found && config.SlashMultiplierBps > 0 {
		multiplier = int64(config.SlashMultiplierBps)
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
	validator *types.LegibleValidator,
) (types.ValidatorTier, bool) {
	currentTier := types.ValidatorTierFromString(validator.Tier)
	potentialTier := ComputeValidatorTier(
		ctx,
		k,
		validator.TotalStake,
		validator.VerificationsCompleted,
		validator.VerificationsCorrect,
		validator.SlashCount,
		validator.ContestedVerificationsCorrect,
		validator.LastSlashedHeight,
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
	validator *types.LegibleValidator,
) (types.ValidatorTier, bool) {
	currentTier := types.ValidatorTierFromString(validator.Tier)
	potentialTier := ComputeValidatorTier(
		ctx,
		k,
		validator.TotalStake,
		validator.VerificationsCompleted,
		validator.VerificationsCorrect,
		validator.SlashCount,
		validator.ContestedVerificationsCorrect,
		validator.LastSlashedHeight,
	)

	if potentialTier < currentTier {
		return potentialTier, true
	}

	return currentTier, false
}

// GetTierRequirements returns the requirements for a specific tier (for UI/CLI)
func GetTierRequirements(ctx sdk.Context, k Keeper, tier types.ValidatorTier) *types.TierConfig {
	config, found := k.GetTierConfig(ctx, tier)
	if !found {
		// Return empty config if not found
		return &types.TierConfig{Tier: types.ValidatorTierString(tier)}
	}
	return config
}

// ValidateTransition checks if a tier transition is valid
func ValidateTransition(
	ctx sdk.Context,
	k Keeper,
	fromTier types.ValidatorTier,
	toTier types.ValidatorTier,
	validator *types.LegibleValidator,
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
		validator.ContestedVerificationsCorrect,
		validator.LastSlashedHeight,
	)

	if computedTier < toTier {
		return types.ErrTierRequirementsNotMet
	}

	return nil
}
