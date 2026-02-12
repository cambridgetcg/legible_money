package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromModuleToModule(ctx context.Context, senderModule string, recipientModule string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

// KnowledgeKeeper provides fact data for pricing and citation rewards.
type KnowledgeKeeper interface {
	GetFactConfidence(ctx sdk.Context, factId string) (confidence uint64, found bool)
	GetFactCitationCount(ctx sdk.Context, factId string) (count uint64, found bool)
	GetFactSubmitter(ctx sdk.Context, factId string) (submitter string, found bool)
	GetFactCreatedBlock(ctx sdk.Context, factId string) (block uint64, found bool)
	IncrementCitationCount(ctx sdk.Context, factId string) error
}
