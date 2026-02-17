package types

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// BankKeeper defines the expected bank module keeper interface.
type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// KnowledgeKeeper defines the expected knowledge module keeper interface for BVM opcodes.
type KnowledgeKeeper interface {
	// Read methods
	GetFact(ctx context.Context, factId string) (fact interface{}, found bool)
	GetFactsByDomain(ctx context.Context, domain string) []interface{}

	// Write methods for BVM opcode integration
	SubmitClaimFromBVM(ctx context.Context, submitterDID, domain, contentHash, evidenceHash, logicZone string, stakeAmount uint64) (claimId string, err error)
	CommitVoteFromBVM(ctx context.Context, validatorDID, roundId, commitmentHash string) error
	RevealVoteFromBVM(ctx context.Context, validatorDID, roundId, verdict string, confidence uint64, salt string) error
	ChallengeFactFromBVM(ctx context.Context, challengerDID, factId, evidenceHash string, stakeAmount uint64) (roundId string, err error)
}

// StakingKeeper defines the expected staking module keeper interface for BVM opcodes.
type StakingKeeper interface {
	GetValidatorStake(ctx context.Context, validatorAddr string) (uint64, bool)
}

// SessionCapabilities defines what a session key is allowed to do within BVM.
// BVM-local copy to avoid cross-module type import from x/auth/types.
type SessionCapabilities struct {
	CanTransfer     bool
	CanStake        bool
	CanSubmitClaims bool
	CanVote         bool
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
