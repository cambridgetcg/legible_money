package keeper

import (
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/compute_pool/types"
)

// RecordPoolInflow records LGM received from block reward distribution.
// Called by x/vesting_rewards when distributing block rewards.
func (k Keeper) RecordPoolInflow(ctx sdk.Context, amount string) {
	height := uint64(ctx.BlockHeight())
	epoch := height / types.BlocksPerEpoch

	stats := k.GetOrCreateEpochStats(ctx, epoch)
	current := new(big.Int)
	current.SetString(stats.PoolInflow, 10)
	inflow := new(big.Int)
	inflow.SetString(amount, 10)
	current.Add(current, inflow)
	stats.PoolInflow = current.String()
	k.SetEpochStats(ctx, stats)
}

// FinalizeEpoch rolls up epoch statistics at epoch boundaries.
// Called in BeginBlock when a new epoch starts.
//
// Auto-tuning logic:
//  1. Compute utilization = outflow / inflow (in bps)
//  2. If utilization < TargetUtilizationLowBps → increase adjustment factor (more credits)
//  3. If utilization > TargetUtilizationHighBps → decrease adjustment factor (fewer credits)
//  4. Clamp factor to [MinAdjustmentFactorBps, MaxAdjustmentFactorBps]
func (k Keeper) FinalizeEpoch(ctx sdk.Context) {
	height := uint64(ctx.BlockHeight())
	epoch := height / types.BlocksPerEpoch

	// Only finalize if this is actually an epoch boundary
	if height%types.BlocksPerEpoch != 0 || epoch == 0 {
		return
	}

	params := k.GetParams(ctx)
	previousEpoch := epoch - 1
	stats := k.GetOrCreateEpochStats(ctx, previousEpoch)

	// Count active providers and validators from the previous epoch
	var activeProviders uint32
	k.IterateProviders(ctx, func(p types.ComputeProvider) bool {
		if p.Status == string(types.ProviderStatusActive) {
			activeProviders++
		}
		return false
	})
	stats.ActiveProviders = activeProviders

	// Count validators with non-zero credit balances
	var activeValidators uint32
	k.IterateCredits(ctx, func(c types.ComputeCredit) bool {
		if c.EarnedTotal > 0 {
			activeValidators++
		}
		return false
	})
	stats.ActiveValidators = activeValidators

	// ---------- Utilization-Based Auto-Tuning ----------

	currentFactor := k.GetAdjustmentFactor(ctx)

	// Compute utilization ratio: outflow / inflow (in bps, 1000000 = 100%)
	inflow := new(big.Int)
	inflow.SetString(stats.PoolInflow, 10)

	outflow := new(big.Int)
	outflow.SetString(stats.LgmDistributed, 10)

	var utilizationBps uint64
	if inflow.Sign() > 0 {
		// utilizationBps = outflow × 1000000 / inflow
		num := new(big.Int).Mul(outflow, big.NewInt(1000000))
		num.Div(num, inflow)
		utilizationBps = num.Uint64()
		if utilizationBps > 1000000 {
			utilizationBps = 1000000 // cap at 100%
		}
	}
	// If inflow is zero, utilization stays 0 — no adjustment needed

	stats.UtilizationBps = utilizationBps

	// Adjust the factor based on utilization vs target range
	newFactor := currentFactor
	step := params.MaxAdjustmentPerEpochBps

	if inflow.Sign() > 0 {
		if utilizationBps < params.TargetUtilizationLowBps {
			// Underutilized: pool is accumulating. Increase credits to encourage spending.
			// Scale step proportionally to how far below the target we are.
			// gap = targetLow - utilization (in bps)
			gap := params.TargetUtilizationLowBps - utilizationBps
			// adjustStep = step × gap / targetLow (proportional, capped at step)
			adjustStep := step * gap / params.TargetUtilizationLowBps
			if adjustStep > step {
				adjustStep = step
			}
			if adjustStep == 0 {
				adjustStep = 1 // always move at least 1 bps
			}
			newFactor += adjustStep
		} else if utilizationBps > params.TargetUtilizationHighBps {
			// Overutilized: pool is draining. Decrease credits to slow spending.
			gap := utilizationBps - params.TargetUtilizationHighBps
			overshootRange := uint64(1000000) - params.TargetUtilizationHighBps
			adjustStep := step
			if overshootRange > 0 {
				adjustStep = step * gap / overshootRange
			}
			if adjustStep > step {
				adjustStep = step
			}
			if adjustStep == 0 {
				adjustStep = 1
			}
			if newFactor > adjustStep {
				newFactor -= adjustStep
			} else {
				newFactor = params.MinAdjustmentFactorBps
			}
		}
		// If within [targetLow, targetHigh], no change — equilibrium
	}

	// Clamp to bounds
	if newFactor < params.MinAdjustmentFactorBps {
		newFactor = params.MinAdjustmentFactorBps
	}
	if newFactor > params.MaxAdjustmentFactorBps {
		newFactor = params.MaxAdjustmentFactorBps
	}

	stats.AdjustmentFactorBps = newFactor
	k.SetAdjustmentFactor(ctx, newFactor)
	k.SetEpochStats(ctx, stats)

	k.Logger(ctx).Info("finalized compute pool epoch",
		"epoch", previousEpoch,
		"pool_inflow", stats.PoolInflow,
		"lgm_distributed", stats.LgmDistributed,
		"utilization_bps", utilizationBps,
		"adjustment_factor_bps", newFactor,
		"credits_issued", stats.CreditsIssued,
		"credits_redeemed", stats.CreditsRedeemed,
		"credits_expired", stats.CreditsExpired,
		"active_providers", activeProviders,
		"active_validators", activeValidators,
	)
}
