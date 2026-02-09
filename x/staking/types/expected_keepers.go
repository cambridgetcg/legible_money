package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// AccountKeeper defines the expected Cosmos SDK account keeper (v0.50 uses context.Context)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	SetAccount(ctx context.Context, acc sdk.AccountI)
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected Cosmos SDK bank keeper (v0.50 uses context.Context)
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

// LegibleAuthKeeper defines the expected custom auth keeper.
// Currently unused by staking keeper; reserved for future cross-module calls.
type LegibleAuthKeeper interface{}

// TokensKeeper defines the expected tokens module keeper for period-based balance tracking.
// The staking module calls this to maintain the period overlay alongside BankKeeper transfers.
type TokensKeeper interface {
	DebitBalance(ctx sdk.Context, address, amount string) error
	CreditBalance(ctx sdk.Context, address, amount string, blockHeight uint64) error
	CreditBalanceToSpecificPeriod(ctx sdk.Context, address, amount string, period uint64) error
	GetCurrentPeriod(ctx sdk.Context) uint64
}
