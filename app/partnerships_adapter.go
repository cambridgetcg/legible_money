package app

import (
	"context"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"

	partnershipstypes "github.com/legible-money/legible/x/partnerships/types"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
)

// PartnershipsStakingAdapter wraps the Legible staking keeper to satisfy
// the partnerships module's StakingKeeper interface.
type PartnershipsStakingAdapter struct {
	keeper legiblestakingkeeper.Keeper
}

// NewPartnershipsStakingAdapter creates a new adapter.
func NewPartnershipsStakingAdapter(k legiblestakingkeeper.Keeper) *PartnershipsStakingAdapter {
	return &PartnershipsStakingAdapter{keeper: k}
}

var _ partnershipstypes.StakingKeeper = (*PartnershipsStakingAdapter)(nil)

// GetLastValidatorPower returns the last recorded power for a validator.
// Maps from the Legible staking keeper's TotalStake to int64 power.
func (a *PartnershipsStakingAdapter) GetLastValidatorPower(ctx context.Context, operator sdk.ValAddress) (int64, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	v, found := a.keeper.GetValidator(sdkCtx, operator.String())
	if !found {
		return 0, nil
	}
	// Convert TotalStake string to int64 power (in ulgm)
	bi := new(big.Int)
	if _, ok := bi.SetString(v.TotalStake, 10); !ok {
		return 0, nil
	}
	// Divide by 1e6 to get power in whole LGM tokens
	divisor := new(big.Int).SetInt64(1_000_000)
	bi.Div(bi, divisor)
	if !bi.IsInt64() {
		return 1<<62 - 1, nil // cap at large value
	}
	return bi.Int64(), nil
}
