package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"

	stakingtypes "github.com/legible-money/legible/x/staking/types"
)

// StakingKeeper defines the expected staking module keeper interface.
type StakingKeeper interface {
	GetValidator(ctx sdk.Context, operatorAddr string) (*stakingtypes.LegibleValidator, bool)
	GetActiveValidatorSet(ctx sdk.Context) []*stakingtypes.LegibleValidator
	IterateValidators(ctx sdk.Context, cb func(*stakingtypes.LegibleValidator) bool)
}
