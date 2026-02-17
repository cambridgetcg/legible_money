package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

// StakingKeeper defines the expected staking module interface.
type StakingKeeper interface {
	// GetValidatorByAddress returns a validator and whether it exists.
	GetValidatorByAddress(ctx sdk.Context, addr string) (tier string, stake string, found bool)
}

// KnowledgeKeeper defines the expected knowledge module interface.
type KnowledgeKeeper interface {
	// GetClaimComplexity returns the evidence count and cross-domain reference count
	// for a given claim, used to calculate compute credit grants.
	GetClaimComplexity(ctx sdk.Context, claimId string) (evidenceCount uint64, crossDomainRefs uint64, found bool)
}
