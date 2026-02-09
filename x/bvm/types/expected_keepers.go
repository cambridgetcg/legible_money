package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module keeper interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
}

// KnowledgeKeeper defines the expected knowledge module keeper interface for BVM opcodes.
type KnowledgeKeeper interface {
	GetFact(ctx context.Context, factId string) (fact interface{}, found bool)
	GetFactsByDomain(ctx context.Context, domain string) []interface{}
}

// StakingKeeper defines the expected staking module keeper interface for BVM opcodes.
type StakingKeeper interface {
	GetValidatorStake(ctx context.Context, validatorAddr string) (uint64, bool)
}

// AuthKeeper defines the expected auth module keeper interface for BVM opcodes.
type AuthKeeper interface {
	GetAccountByDID(ctx context.Context, did string) (interface{}, bool)
}
