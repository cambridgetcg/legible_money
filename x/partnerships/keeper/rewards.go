package keeper

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/partnerships/types"
)

// GetLockMultiplier returns the reward multiplier for a given lock tier (basis points, 1000000 = 1x).
func GetLockMultiplier(lockTier uint32) *big.Int {
	if lockTier > 5 {
		lockTier = 5
	}
	return new(big.Int).SetUint64(types.LockTiers[lockTier].MultiplierBps)
}

// DistributeReward distributes a gross reward through the partnership.
//
// Steps:
// 1. Apply lock multiplier based on tier
// 2. 22% (CommonPotShareBps) goes to common pot
// 3. Remaining 78% split per partnership split (default 50/50)
// 4. Update metrics (TotalEarned, CommonPotBalance)
//
// Returns (humanShare, agentShare, commonPotAdd) as string amounts.
func (k Keeper) DistributeReward(
	ctx sdk.Context,
	partnership *types.Partnership,
	grossAmount string,
	sourceType string,
) (humanShare string, agentShare string, commonPotAdd string, err error) {
	gross := new(big.Int)
	if _, ok := gross.SetString(grossAmount, 10); !ok || gross.Sign() <= 0 {
		return "", "", "", fmt.Errorf("invalid gross amount: %s", grossAmount)
	}

	params := k.GetParams(ctx)

	// Step 1: Apply lock multiplier by lock tier
	multiplier := GetLockMultiplier(partnership.LockTier)
	// adjusted = gross * multiplier / 1000000
	adjusted := new(big.Int).Mul(gross, multiplier)
	adjusted.Div(adjusted, bpsDenom)

	// Step 2: Calculate common pot share (22% = 220000 bps)
	commonPotShareBps := new(big.Int).SetUint64(params.CommonPotShareBps)
	potShare := new(big.Int).Mul(adjusted, commonPotShareBps)
	potShare.Div(potShare, bpsDenom)

	// Step 3: Remaining goes to individual splits
	remaining := new(big.Int).Sub(adjusted, potShare)

	// Human share = remaining * splitHumanBps / 1000000
	humanBps := new(big.Int).SetUint64(partnership.SplitHumanBps)
	humanAmt := new(big.Int).Mul(remaining, humanBps)
	humanAmt.Div(humanAmt, bpsDenom)

	// Agent share = remaining - humanShare (avoids rounding loss)
	agentAmt := new(big.Int).Sub(remaining, humanAmt)

	// Step 4: Update partnership metrics
	// Update common pot balance
	currentPot := new(big.Int)
	if partnership.CommonPotBalance != "" {
		currentPot.SetString(partnership.CommonPotBalance, 10)
	}
	currentPot.Add(currentPot, potShare)
	partnership.CommonPotBalance = currentPot.String()

	// Update total earned
	currentEarned := new(big.Int)
	if partnership.TotalEarned != "" {
		currentEarned.SetString(partnership.TotalEarned, 10)
	}
	currentEarned.Add(currentEarned, adjusted)
	partnership.TotalEarned = currentEarned.String()

	// Persist updated partnership
	k.SetPartnership(ctx, partnership)

	k.Logger(ctx).Info("partnership reward distributed",
		"partnership_id", partnership.Id,
		"gross", grossAmount,
		"adjusted", adjusted.String(),
		"common_pot", potShare.String(),
		"human_share", humanAmt.String(),
		"agent_share", agentAmt.String(),
		"source", sourceType,
	)

	return humanAmt.String(), agentAmt.String(), potShare.String(), nil
}
