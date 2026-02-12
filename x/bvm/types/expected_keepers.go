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

// SessionCapabilities defines what a session key is allowed to do within BVM.
// BVM-local copy to avoid cross-module type import from x/auth/types.
type SessionCapabilities struct {
	CanTransfer    bool
	CanStake       bool
	CanSubmitClaims bool
	CanVote        bool
}

// AuthKeeper defines the expected auth module keeper interface for BVM opcodes.
type AuthKeeper interface {
	GetAccountByDID(ctx context.Context, did string) (interface{}, bool)
	// GetAccountDID resolves a bech32 address to its DID. Returns ("", false) if unknown.
	GetAccountDID(ctx context.Context, address string) (string, bool)
	// GetSessionCapabilities returns the active session key capabilities for an owner at a block height.
	GetSessionCapabilities(ctx context.Context, owner string, blockHeight uint64) (SessionCapabilities, bool)
	// RegisterSessionKey creates a session key in the canonical x/auth store.
	RegisterSessionKey(ctx context.Context, ownerAddress, keyHash, pubKey string, caps SessionCapabilities, expiresAtBlock uint64) error
	// RevokeSessionKey removes a session key from the canonical x/auth store.
	RevokeSessionKey(ctx context.Context, ownerAddress, keyHash string) error
}
