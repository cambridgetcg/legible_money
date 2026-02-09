package keeper

import (
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/research/types"
)

// ---------- Params ----------

func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(params)
	store.Set(types.ParamsKey, bz)
}

func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	json.Unmarshal(bz, &params)
	return params
}

// ---------- Research Submissions ----------

func submissionKey(id string) []byte {
	return append(types.SubmissionKeyPrefix, []byte(id)...)
}

func (k Keeper) SetResearch(ctx sdk.Context, r types.Research) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(r)
	store.Set(submissionKey(r.Id), bz)
}

func (k Keeper) GetResearch(ctx sdk.Context, id string) (types.Research, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(submissionKey(id))
	if bz == nil {
		return types.Research{}, false
	}
	var r types.Research
	json.Unmarshal(bz, &r)
	return r, true
}

func (k Keeper) DeleteResearch(ctx sdk.Context, id string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(submissionKey(id))
}

func (k Keeper) IterateResearches(ctx sdk.Context, cb func(types.Research) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.SubmissionKeyPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var r types.Research
		json.Unmarshal(iter.Value(), &r)
		if cb(r) {
			break
		}
	}
}

func (k Keeper) GetResearchesByStatus(ctx sdk.Context, status types.ResearchStatus) []types.Research {
	var results []types.Research
	k.IterateResearches(ctx, func(r types.Research) bool {
		if r.Status == status {
			results = append(results, r)
		}
		return false
	})
	return results
}

func (k Keeper) GetResearchesByDomain(ctx sdk.Context, domain string) []types.Research {
	var results []types.Research
	k.IterateResearches(ctx, func(r types.Research) bool {
		if r.Domain == domain {
			results = append(results, r)
		}
		return false
	})
	return results
}

// ---------- Bounties ----------

func bountyKey(id string) []byte {
	return append(types.BountyKeyPrefix, []byte(id)...)
}

func (k Keeper) SetBounty(ctx sdk.Context, b types.Bounty) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(b)
	store.Set(bountyKey(b.Id), bz)
}

func (k Keeper) GetBounty(ctx sdk.Context, id string) (types.Bounty, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(bountyKey(id))
	if bz == nil {
		return types.Bounty{}, false
	}
	var b types.Bounty
	json.Unmarshal(bz, &b)
	return b, true
}

func (k Keeper) DeleteBounty(ctx sdk.Context, id string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(bountyKey(id))
}

func (k Keeper) IterateBounties(ctx sdk.Context, cb func(types.Bounty) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.BountyKeyPrefix)
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		var b types.Bounty
		json.Unmarshal(iter.Value(), &b)
		if cb(b) {
			break
		}
	}
}

func (k Keeper) GetActiveBounties(ctx sdk.Context) []types.Bounty {
	var results []types.Bounty
	k.IterateBounties(ctx, func(b types.Bounty) bool {
		if b.Status == types.BountyStatusOpen || b.Status == types.BountyStatusClaimed {
			results = append(results, b)
		}
		return false
	})
	return results
}

// ---------- Peer Reviews ----------

func reviewKey(researchId, reviewId string) []byte {
	return append(types.PeerReviewKeyPrefix, []byte(researchId+"/"+reviewId)...)
}

func reviewPrefixForResearch(researchId string) []byte {
	return append(types.PeerReviewKeyPrefix, []byte(researchId+"/")...)
}

func (k Keeper) SetPeerReview(ctx sdk.Context, r types.PeerReview) {
	store := ctx.KVStore(k.storeKey)
	bz, _ := json.Marshal(r)
	store.Set(reviewKey(r.ResearchId, r.Id), bz)
}

func (k Keeper) GetPeerReview(ctx sdk.Context, researchId, reviewId string) (types.PeerReview, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(reviewKey(researchId, reviewId))
	if bz == nil {
		return types.PeerReview{}, false
	}
	var r types.PeerReview
	json.Unmarshal(bz, &r)
	return r, true
}

func (k Keeper) GetReviewsForResearch(ctx sdk.Context, researchId string) []types.PeerReview {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, reviewPrefixForResearch(researchId))
	defer iter.Close()

	var results []types.PeerReview
	for ; iter.Valid(); iter.Next() {
		var r types.PeerReview
		json.Unmarshal(iter.Value(), &r)
		results = append(results, r)
	}
	return results
}

func (k Keeper) HasReviewerReviewed(ctx sdk.Context, researchId, reviewer string) bool {
	reviews := k.GetReviewsForResearch(ctx, researchId)
	for _, r := range reviews {
		if r.Reviewer == reviewer {
			return true
		}
	}
	return false
}

// ---------- Treasury ----------

func (k Keeper) SetTreasuryBalance(ctx sdk.Context, balance string) {
	store := ctx.KVStore(k.storeKey)
	tb := types.TreasuryBalance{Balance: balance}
	bz, _ := json.Marshal(tb)
	store.Set(types.TreasuryKeyPrefix, bz)
}

func (k Keeper) GetTreasuryBalance(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.TreasuryKeyPrefix)
	if bz == nil {
		return "0"
	}
	var tb types.TreasuryBalance
	json.Unmarshal(bz, &tb)
	return tb.Balance
}

// ---------- Counters ----------

var (
	researchCounterKey = []byte("research_counter")
	bountyCounterKey   = []byte("bounty_counter")
)

func (k Keeper) nextResearchId(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(researchCounterKey)
	var counter uint64
	if bz != nil {
		json.Unmarshal(bz, &counter)
	}
	counter++
	bz, _ = json.Marshal(counter)
	store.Set(researchCounterKey, bz)
	return fmt.Sprintf("RES-%d", counter)
}

func (k Keeper) nextBountyId(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(bountyCounterKey)
	var counter uint64
	if bz != nil {
		json.Unmarshal(bz, &counter)
	}
	counter++
	bz, _ = json.Marshal(counter)
	store.Set(bountyCounterKey, bz)
	return fmt.Sprintf("BOUNTY-%d", counter)
}

func (k Keeper) nextReviewId(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	key := []byte("review_counter")
	bz := store.Get(key)
	var counter uint64
	if bz != nil {
		json.Unmarshal(bz, &counter)
	}
	counter++
	bz, _ = json.Marshal(counter)
	store.Set(key, bz)
	return fmt.Sprintf("REV-%d", counter)
}
