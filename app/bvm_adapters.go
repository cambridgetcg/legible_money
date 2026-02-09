package app

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	legibleauthkeeper "github.com/legible-money/legible/x/auth/keeper"
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
