package app

import (
	"context"
	"crypto/sha256"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	legibleauthkeeper "github.com/legible-money/legible/x/auth/keeper"
	authtypes "github.com/legible-money/legible/x/auth/types"
	bvmtypes "github.com/legible-money/legible/x/bvm/types"
	knowledgekeeper "github.com/legible-money/legible/x/knowledge/keeper"
	legiblestakingkeeper "github.com/legible-money/legible/x/staking/keeper"
)

// --- BVM Knowledge Adapter ---

// BVMKnowledgeAdapter wraps x/knowledge keeper to satisfy bvm types.KnowledgeKeeper.
type BVMKnowledgeAdapter struct {
	keeper knowledgekeeper.Keeper
}

func NewBVMKnowledgeAdapter(keeper knowledgekeeper.Keeper) *BVMKnowledgeAdapter {
	return &BVMKnowledgeAdapter{keeper: keeper}
}

var _ bvmtypes.KnowledgeKeeper = (*BVMKnowledgeAdapter)(nil)

func (a *BVMKnowledgeAdapter) GetFact(ctx context.Context, factId string) (interface{}, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	fact, found := a.keeper.GetFact(sdkCtx, factId)
	if !found {
		return nil, false
	}
	return fact, true
}

func (a *BVMKnowledgeAdapter) GetFactsByDomain(ctx context.Context, domain string) []interface{} {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	facts := a.keeper.GetFactsByDomain(sdkCtx, domain)
	result := make([]interface{}, len(facts))
	for i, f := range facts {
		result[i] = f
	}
	return result
}

func (a *BVMKnowledgeAdapter) SubmitClaimFromBVM(ctx context.Context, submitterDID, domain, contentHash, evidenceHash, logicZone string, stakeAmount uint64) (string, error) {
	return a.keeper.SubmitClaimFromBVM(ctx, submitterDID, domain, contentHash, evidenceHash, logicZone, stakeAmount)
}

func (a *BVMKnowledgeAdapter) CommitVoteFromBVM(ctx context.Context, validatorDID, roundId, commitmentHash string) error {
	return a.keeper.CommitVoteFromBVM(ctx, validatorDID, roundId, commitmentHash)
}

func (a *BVMKnowledgeAdapter) RevealVoteFromBVM(ctx context.Context, validatorDID, roundId, verdict string, confidence uint64, salt string) error {
	return a.keeper.RevealVoteFromBVM(ctx, validatorDID, roundId, verdict, confidence, salt)
}

func (a *BVMKnowledgeAdapter) ChallengeFactFromBVM(ctx context.Context, challengerDID, factId, evidenceHash string, stakeAmount uint64) (string, error) {
	return a.keeper.ChallengeFactFromBVM(ctx, challengerDID, factId, evidenceHash, stakeAmount)
}

// --- BVM Staking Adapter ---

// BVMStakingAdapter wraps x/staking keeper to satisfy bvm types.StakingKeeper.
type BVMStakingAdapter struct {
	keeper legiblestakingkeeper.Keeper
}

func NewBVMStakingAdapter(keeper legiblestakingkeeper.Keeper) *BVMStakingAdapter {
	return &BVMStakingAdapter{keeper: keeper}
}

var _ bvmtypes.StakingKeeper = (*BVMStakingAdapter)(nil)

func (a *BVMStakingAdapter) GetValidatorStake(ctx context.Context, validatorAddr string) (uint64, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	validator, found := a.keeper.GetValidator(sdkCtx, validatorAddr)
	if !found {
		return 0, false
	}
	return bigIntStringToUint64(validator.TotalStake), true
}

// --- BVM Auth Adapter ---

// BVMAuthAdapter wraps x/auth keeper to satisfy bvm types.AuthKeeper.
type BVMAuthAdapter struct {
	keeper legibleauthkeeper.Keeper
}

func NewBVMAuthAdapter(keeper legibleauthkeeper.Keeper) *BVMAuthAdapter {
	return &BVMAuthAdapter{keeper: keeper}
}

var _ bvmtypes.AuthKeeper = (*BVMAuthAdapter)(nil)

func (a *BVMAuthAdapter) GetAccountByDID(ctx context.Context, did string) (interface{}, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	account, found := a.keeper.GetAccountByDID(sdkCtx, did)
	if !found {
		return nil, false
	}
	return account, true
}

// GetAccountDID resolves a bech32 address to its DID.
func (a *BVMAuthAdapter) GetAccountDID(ctx context.Context, address string) (string, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	account, found := a.keeper.GetAccount(sdkCtx, address)
	if !found || account.Did == "" {
		return "", false
	}
	return account.Did, true
}

// GetSessionCapabilities returns the active session capabilities for an owner at a given block height.
// Finds the first valid (non-expired) session key and returns its capabilities.
func (a *BVMAuthAdapter) GetSessionCapabilities(ctx context.Context, owner string, blockHeight uint64) (bvmtypes.SessionCapabilities, bool) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sessions := a.keeper.GetSessionKeysForOwner(sdkCtx, owner)
	for _, session := range sessions {
		if session.ExpiresAtBlock > blockHeight {
			return bvmtypes.SessionCapabilities{
				CanTransfer:     session.Capabilities.CanTransfer,
				CanStake:        session.Capabilities.CanStake,
				CanSubmitClaims: session.Capabilities.CanSubmitClaims,
				CanVote:         session.Capabilities.CanVote,
			}, true
		}
	}
	return bvmtypes.SessionCapabilities{}, false
}

// RegisterSessionKey creates a session key in the canonical x/auth store.
func (a *BVMAuthAdapter) RegisterSessionKey(ctx context.Context, ownerAddress, keyHash, pubKey string, caps bvmtypes.SessionCapabilities, expiresAtBlock uint64) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Generate key hash if not provided
	if keyHash == "" {
		h := sha256.Sum256([]byte(pubKey))
		keyHash = fmt.Sprintf("%x", h)
	}

	session := authtypes.SessionKey{
		KeyHash:        keyHash,
		PublicKey:      pubKey,
		Owner:          ownerAddress,
		ExpiresAtBlock: expiresAtBlock,
		CreatedAtBlock: uint64(sdkCtx.BlockHeight()),
		Capabilities: &authtypes.SessionCapabilities{
			CanTransfer:     caps.CanTransfer,
			CanStake:        caps.CanStake,
			CanSubmitClaims: caps.CanSubmitClaims,
			CanVote:         caps.CanVote,
		},
	}
	a.keeper.SetSessionKey(sdkCtx, &session)

	// Increment session key count on the account
	if account, found := a.keeper.GetAccount(sdkCtx, ownerAddress); found {
		account.SessionKeyCount++
		a.keeper.SetAccount(sdkCtx, account)
	}

	return nil
}

// RevokeSessionKey removes a session key from the canonical x/auth store.
func (a *BVMAuthAdapter) RevokeSessionKey(ctx context.Context, ownerAddress, keyHash string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	a.keeper.DeleteSessionKey(sdkCtx, ownerAddress, keyHash)

	// Decrement session key count on the account
	if account, found := a.keeper.GetAccount(sdkCtx, ownerAddress); found {
		if account.SessionKeyCount > 0 {
			account.SessionKeyCount--
			a.keeper.SetAccount(sdkCtx, account)
		}
	}

	return nil
}
