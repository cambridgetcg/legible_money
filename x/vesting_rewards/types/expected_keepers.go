package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module keeper interface.
type BankKeeper interface {
	MintCoins(ctx context.Context, moduleName string, amounts sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetSupply(ctx context.Context, denom string) sdk.Coin
}

// StakingKeeper defines the expected staking module keeper interface.
// Used to query active validator count for block reward scaling.
type StakingKeeper interface {
	GetActiveValidatorCount(ctx context.Context) uint32
}

// AutopoiesisKeeper defines the expected autopoiesis keeper interface.
// The vesting_rewards module reads the blockreward multiplier to scale emissions.
type AutopoiesisKeeper interface {
	// GetMultiplier returns the current multiplier value for a path (BPS, 1M = 1.0x).
	GetMultiplier(ctx sdk.Context, path string) uint64
}
