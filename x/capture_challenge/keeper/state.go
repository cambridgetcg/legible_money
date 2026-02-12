package keeper

import (
	"encoding/json"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/capture_challenge/types"
)

// --- Challenge CRUD ---

// GetChallenge returns a challenge by ID.
func (k Keeper) GetChallenge(ctx sdk.Context, challengeId string) (types.CaptureChallenge, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ChallengeKey(challengeId))
	if err != nil || bz == nil {
		return types.CaptureChallenge{}, false
	}
	var challenge types.CaptureChallenge
	if err := json.Unmarshal(bz, &challenge); err != nil {
		return types.CaptureChallenge{}, false
	}
	return challenge, true
}

// SetChallenge stores a challenge.
func (k Keeper) SetChallenge(ctx sdk.Context, challenge types.CaptureChallenge) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(challenge)
	if err != nil {
		panic("failed to marshal challenge: " + err.Error())
	}
	if err := store.Set(types.ChallengeKey(challenge.Id), bz); err != nil {
		panic("failed to set challenge: " + err.Error())
	}
}

// DeleteChallenge removes a challenge.
func (k Keeper) DeleteChallenge(ctx sdk.Context, challengeId string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.ChallengeKey(challengeId)); err != nil {
		panic("failed to delete challenge: " + err.Error())
	}
}

// IterateChallenges iterates over all challenges.
func (k Keeper) IterateChallenges(ctx sdk.Context, cb func(types.CaptureChallenge) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ChallengeKeyPrefix, prefixEndBytes(types.ChallengeKeyPrefix))
	if err != nil {
		panic("failed to create challenge iterator: " + err.Error())
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var challenge types.CaptureChallenge
		if err := json.Unmarshal(iter.Value(), &challenge); err != nil {
			continue
		}
		if cb(challenge) {
			break
		}
	}
}

// --- Bounty Pool CRUD ---

// GetBountyPool returns a domain's bounty pool.
func (k Keeper) GetBountyPool(ctx sdk.Context, domain string) (types.DomainBountyPool, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.BountyPoolKey(domain))
	if err != nil || bz == nil {
		return types.DomainBountyPool{}, false
	}
	var pool types.DomainBountyPool
	if err := json.Unmarshal(bz, &pool); err != nil {
		return types.DomainBountyPool{}, false
	}
	return pool, true
}

// SetBountyPool stores a domain's bounty pool.
func (k Keeper) SetBountyPool(ctx sdk.Context, pool types.DomainBountyPool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(pool)
	if err != nil {
		panic("failed to marshal bounty pool: " + err.Error())
	}
	if err := store.Set(types.BountyPoolKey(pool.Domain), bz); err != nil {
		panic("failed to set bounty pool: " + err.Error())
	}
}

// GetOrCreateBountyPool returns existing pool or creates a new one.
func (k Keeper) GetOrCreateBountyPool(ctx sdk.Context, domain string, currentBlock uint64) types.DomainBountyPool {
	pool, found := k.GetBountyPool(ctx, domain)
	if found {
		return pool
	}
	return types.DomainBountyPool{
		Domain:           domain,
		Balance:          "0",
		TotalRewardsPaid: "0",
		CreatedAtBlock:   currentBlock,
		LastUpdatedBlock: currentBlock,
	}
}

// IterateBountyPools iterates over all bounty pools.
func (k Keeper) IterateBountyPools(ctx sdk.Context, cb func(types.DomainBountyPool) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.BountyPoolKeyPrefix, prefixEndBytes(types.BountyPoolKeyPrefix))
	if err != nil {
		panic("failed to create bounty pool iterator: " + err.Error())
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var pool types.DomainBountyPool
		if err := json.Unmarshal(iter.Value(), &pool); err != nil {
			continue
		}
		if cb(pool) {
			break
		}
	}
}

// --- Params CRUD ---

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKeyPrefix)
	if err != nil || bz == nil {
		p := types.DefaultParams()
		return *p
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		p := types.DefaultParams()
		return *p
	}
	return params
}

// SetParams stores the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(params)
	if err != nil {
		panic("failed to marshal params: " + err.Error())
	}
	if err := store.Set(types.ParamsKeyPrefix, bz); err != nil {
		panic("failed to set params: " + err.Error())
	}
}
