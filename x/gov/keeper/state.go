package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/gov/types"
)

// --- LIP CRUD ---

// SetLIP stores a LIP by ID.
func (k Keeper) SetLIP(ctx sdk.Context, lip types.LIP) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(lip)
	if err != nil {
		panic("failed to marshal LIP: " + err.Error())
	}
	store.Set(lipKey(lip.Id), bz)
}

// GetLIP retrieves a LIP by ID.
func (k Keeper) GetLIP(ctx sdk.Context, id string) (types.LIP, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(lipKey(id))
	if bz == nil {
		return types.LIP{}, false
	}
	var lip types.LIP
	if err := json.Unmarshal(bz, &lip); err != nil {
		return types.LIP{}, false
	}
	return lip, true
}

// GetAllLIPs returns all stored LIPs.
func (k Keeper) GetAllLIPs(ctx sdk.Context) []types.LIP {
	var lips []types.LIP
	k.IterateLIPs(ctx, func(lip types.LIP) bool {
		lips = append(lips, lip)
		return false
	})
	return lips
}

// GetLIPsByStage returns LIPs filtered by stage.
func (k Keeper) GetLIPsByStage(ctx sdk.Context, stage types.LIPStage) []types.LIP {
	var lips []types.LIP
	k.IterateLIPs(ctx, func(lip types.LIP) bool {
		if lip.Stage == stage {
			lips = append(lips, lip)
		}
		return false
	})
	return lips
}

// IterateLIPs iterates over all LIPs. Return true from cb to stop.
func (k Keeper) IterateLIPs(ctx sdk.Context, cb func(types.LIP) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.LIPKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var lip types.LIP
		if err := json.Unmarshal(iter.Value(), &lip); err != nil {
			continue
		}
		if cb(lip) {
			break
		}
	}
}

// --- Vote CRUD ---

// SetVote stores a quadratic vote.
func (k Keeper) SetVote(ctx sdk.Context, vote types.QuadraticVote) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(vote)
	if err != nil {
		panic("failed to marshal vote: " + err.Error())
	}
	store.Set(voteKey(vote.VoteId), bz)
}

// GetVote retrieves a vote by ID.
func (k Keeper) GetVote(ctx sdk.Context, voteId string) (types.QuadraticVote, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(voteKey(voteId))
	if bz == nil {
		return types.QuadraticVote{}, false
	}
	var vote types.QuadraticVote
	if err := json.Unmarshal(bz, &vote); err != nil {
		return types.QuadraticVote{}, false
	}
	return vote, true
}

// GetVotesForProposal returns all votes for a given proposal ID.
func (k Keeper) GetVotesForProposal(ctx sdk.Context, proposalId string) []types.QuadraticVote {
	var votes []types.QuadraticVote
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.VoteKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var vote types.QuadraticVote
		if err := json.Unmarshal(iter.Value(), &vote); err != nil {
			continue
		}
		if vote.ProposalId == proposalId {
			votes = append(votes, vote)
		}
	}
	return votes
}

// HasVoted checks if a voter has already voted on a proposal.
func (k Keeper) HasVoted(ctx sdk.Context, proposalId, voter string) bool {
	store := ctx.KVStore(k.storeKey)
	return store.Has(voteDedupeKey(proposalId, voter))
}

// SetVoteDedupe marks a voter as having voted on a proposal.
func (k Keeper) SetVoteDedupe(ctx sdk.Context, proposalId, voter string) {
	store := ctx.KVStore(k.storeKey)
	store.Set(voteDedupeKey(proposalId, voter), []byte{1})
}

// GetAllVotes returns all stored votes.
func (k Keeper) GetAllVotes(ctx sdk.Context) []types.QuadraticVote {
	var votes []types.QuadraticVote
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.VoteKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var vote types.QuadraticVote
		if err := json.Unmarshal(iter.Value(), &vote); err != nil {
			continue
		}
		votes = append(votes, vote)
	}
	return votes
}

// --- Disbursement CRUD ---

// SetDisbursement stores a disbursement proposal.
func (k Keeper) SetDisbursement(ctx sdk.Context, disb types.DisbursementProposal) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(disb)
	if err != nil {
		panic("failed to marshal disbursement: " + err.Error())
	}
	store.Set(disbursementKey(disb.Id), bz)
}

// GetDisbursement retrieves a disbursement by ID.
func (k Keeper) GetDisbursement(ctx sdk.Context, id string) (types.DisbursementProposal, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(disbursementKey(id))
	if bz == nil {
		return types.DisbursementProposal{}, false
	}
	var disb types.DisbursementProposal
	if err := json.Unmarshal(bz, &disb); err != nil {
		return types.DisbursementProposal{}, false
	}
	return disb, true
}

// GetAllDisbursements returns all stored disbursements.
func (k Keeper) GetAllDisbursements(ctx sdk.Context) []types.DisbursementProposal {
	var disbs []types.DisbursementProposal
	k.IterateDisbursements(ctx, func(d types.DisbursementProposal) bool {
		disbs = append(disbs, d)
		return false
	})
	return disbs
}

// IterateDisbursements iterates over all disbursements. Return true from cb to stop.
func (k Keeper) IterateDisbursements(ctx sdk.Context, cb func(types.DisbursementProposal) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.DisbursementKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var disb types.DisbursementProposal
		if err := json.Unmarshal(iter.Value(), &disb); err != nil {
			continue
		}
		if cb(disb) {
			break
		}
	}
}

// --- Counters ---

// GetNextLIPNumber returns the next LIP sequence number.
func (k Keeper) GetNextLIPNumber(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LIPCounterKey)
	if bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

// SetNextLIPNumber sets the next LIP sequence number.
func (k Keeper) SetNextLIPNumber(ctx sdk.Context, n uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, n)
	store.Set(types.LIPCounterKey, bz)
}

// GetNextDisbNumber returns the next disbursement sequence number.
func (k Keeper) GetNextDisbNumber(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.DisbCounterKey)
	if bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

// SetNextDisbNumber sets the next disbursement sequence number.
func (k Keeper) SetNextDisbNumber(ctx sdk.Context, n uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, n)
	store.Set(types.DisbCounterKey, bz)
}

// --- Params ---

// GetParams returns the governance module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ParamsKey)
	if bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams stores the governance module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic("failed to marshal governance params: " + err.Error())
	}
	store.Set(types.ParamsKey, bz)
}

// --- Key construction helpers ---

func lipKey(id string) []byte {
	return append(types.LIPKeyPrefix, []byte(id)...)
}

func voteKey(id string) []byte {
	return append(types.VoteKeyPrefix, []byte(id)...)
}

func disbursementKey(id string) []byte {
	return append(types.DisbursementKeyPrefix, []byte(id)...)
}

func voteDedupeKey(proposalId, voter string) []byte {
	return append(types.VoteDedupeKeyPrefix, []byte(fmt.Sprintf("%s/%s", proposalId, voter))...)
}
