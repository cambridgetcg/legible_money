package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/gov/types"
)

// --- LIP CRUD ---

// SetLIP stores a LIP by ID.
func (k Keeper) SetLIP(ctx sdk.Context, lip *types.LIP) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(lip)
	if err != nil {
		panic("failed to marshal LIP: " + err.Error())
	}
	if err := store.Set(lipKey(lip.Id), bz); err != nil {
		panic("failed to set LIP: " + err.Error())
	}
}

// GetLIP retrieves a LIP by ID.
func (k Keeper) GetLIP(ctx sdk.Context, id string) (*types.LIP, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(lipKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var lip types.LIP
	if err := proto.Unmarshal(bz, &lip); err != nil {
		return nil, false
	}
	return &lip, true
}

// GetAllLIPs returns all stored LIPs.
func (k Keeper) GetAllLIPs(ctx sdk.Context) []*types.LIP {
	var lips []*types.LIP
	k.IterateLIPs(ctx, func(lip *types.LIP) bool {
		lips = append(lips, lip)
		return false
	})
	return lips
}

// GetLIPsByStage returns LIPs filtered by stage.
func (k Keeper) GetLIPsByStage(ctx sdk.Context, stage types.HandwrittenLIPStage) []*types.LIP {
	var lips []*types.LIP
	k.IterateLIPs(ctx, func(lip *types.LIP) bool {
		if lip.Stage == string(stage) {
			lips = append(lips, lip)
		}
		return false
	})
	return lips
}

// IterateLIPs iterates over all LIPs. Return true from cb to stop.
func (k Keeper) IterateLIPs(ctx sdk.Context, cb func(*types.LIP) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.LIPKeyPrefix, prefixEndBytes(types.LIPKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var lip types.LIP
		if err := proto.Unmarshal(iter.Value(), &lip); err != nil {
			continue
		}
		if cb(&lip) {
			break
		}
	}
}

// --- Vote CRUD ---

// SetVote stores a quadratic vote.
func (k Keeper) SetVote(ctx sdk.Context, vote *types.QuadraticVote) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(vote)
	if err != nil {
		panic("failed to marshal vote: " + err.Error())
	}
	if err := store.Set(voteKey(vote.VoteId), bz); err != nil {
		panic("failed to set vote: " + err.Error())
	}
}

// GetVote retrieves a vote by ID.
func (k Keeper) GetVote(ctx sdk.Context, voteId string) (*types.QuadraticVote, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(voteKey(voteId))
	if err != nil || bz == nil {
		return nil, false
	}
	var vote types.QuadraticVote
	if err := proto.Unmarshal(bz, &vote); err != nil {
		return nil, false
	}
	return &vote, true
}

// GetVotesForProposal returns all votes for a given proposal ID.
func (k Keeper) GetVotesForProposal(ctx sdk.Context, proposalId string) []*types.QuadraticVote {
	var votes []*types.QuadraticVote
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.VoteKeyPrefix, prefixEndBytes(types.VoteKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var vote types.QuadraticVote
		if err := proto.Unmarshal(iter.Value(), &vote); err != nil {
			continue
		}
		if vote.ProposalId == proposalId {
			votes = append(votes, &vote)
		}
	}
	return votes
}

// HasVoted checks if a voter has already voted on a proposal.
func (k Keeper) HasVoted(ctx sdk.Context, proposalId, voter string) bool {
	store := k.storeService.OpenKVStore(ctx)
	has, err := store.Has(voteDedupeKey(proposalId, voter))
	if err != nil {
		return false
	}
	return has
}

// SetVoteDedupe marks a voter as having voted on a proposal.
func (k Keeper) SetVoteDedupe(ctx sdk.Context, proposalId, voter string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(voteDedupeKey(proposalId, voter), []byte{1}); err != nil {
		panic("failed to set vote dedupe: " + err.Error())
	}
}

// GetAllVotes returns all stored votes.
func (k Keeper) GetAllVotes(ctx sdk.Context) []*types.QuadraticVote {
	var votes []*types.QuadraticVote
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.VoteKeyPrefix, prefixEndBytes(types.VoteKeyPrefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var vote types.QuadraticVote
		if err := proto.Unmarshal(iter.Value(), &vote); err != nil {
			continue
		}
		votes = append(votes, &vote)
	}
	return votes
}

// --- Disbursement CRUD ---

// SetDisbursement stores a disbursement proposal.
func (k Keeper) SetDisbursement(ctx sdk.Context, disb *types.DisbursementProposal) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(disb)
	if err != nil {
		panic("failed to marshal disbursement: " + err.Error())
	}
	if err := store.Set(disbursementKey(disb.Id), bz); err != nil {
		panic("failed to set disbursement: " + err.Error())
	}
}

// GetDisbursement retrieves a disbursement by ID.
func (k Keeper) GetDisbursement(ctx sdk.Context, id string) (*types.DisbursementProposal, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(disbursementKey(id))
	if err != nil || bz == nil {
		return nil, false
	}
	var disb types.DisbursementProposal
	if err := proto.Unmarshal(bz, &disb); err != nil {
		return nil, false
	}
	return &disb, true
}

// GetAllDisbursements returns all stored disbursements.
func (k Keeper) GetAllDisbursements(ctx sdk.Context) []*types.DisbursementProposal {
	var disbs []*types.DisbursementProposal
	k.IterateDisbursements(ctx, func(d *types.DisbursementProposal) bool {
		disbs = append(disbs, d)
		return false
	})
	return disbs
}

// IterateDisbursements iterates over all disbursements. Return true from cb to stop.
func (k Keeper) IterateDisbursements(ctx sdk.Context, cb func(*types.DisbursementProposal) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.DisbursementKeyPrefix, prefixEndBytes(types.DisbursementKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var disb types.DisbursementProposal
		if err := proto.Unmarshal(iter.Value(), &disb); err != nil {
			continue
		}
		if cb(&disb) {
			break
		}
	}
}

// --- Counters ---

// GetNextLIPNumber returns the next LIP sequence number.
func (k Keeper) GetNextLIPNumber(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.LIPCounterKey)
	if err != nil || bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

// SetNextLIPNumber sets the next LIP sequence number.
func (k Keeper) SetNextLIPNumber(ctx sdk.Context, n uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, n)
	if err := store.Set(types.LIPCounterKey, bz); err != nil {
		panic("failed to set LIP counter: " + err.Error())
	}
}

// GetNextDisbNumber returns the next disbursement sequence number.
func (k Keeper) GetNextDisbNumber(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.DisbCounterKey)
	if err != nil || bz == nil {
		return 1
	}
	return binary.BigEndian.Uint64(bz)
}

// SetNextDisbNumber sets the next disbursement sequence number.
func (k Keeper) SetNextDisbNumber(ctx sdk.Context, n uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, n)
	if err := store.Set(types.DisbCounterKey, bz); err != nil {
		panic("failed to set disb counter: " + err.Error())
	}
}

// --- Params ---

// GetParams returns the governance module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// SetParams stores the governance module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic("failed to marshal governance params: " + err.Error())
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic("failed to set governance params: " + err.Error())
	}
}

// --- Upgrade Plan CRUD ---

// SetUpgradePlan stores an upgrade plan associated with a LIP.
func (k Keeper) SetUpgradePlan(ctx sdk.Context, lipId string, plan types.UpgradePlan) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(plan)
	if err != nil {
		panic("failed to marshal upgrade plan: " + err.Error())
	}
	if err := store.Set(upgradePlanKey(lipId), bz); err != nil {
		panic("failed to set upgrade plan: " + err.Error())
	}
}

// GetUpgradePlan retrieves the upgrade plan for a LIP.
func (k Keeper) GetUpgradePlan(ctx sdk.Context, lipId string) (types.UpgradePlan, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(upgradePlanKey(lipId))
	if err != nil || bz == nil {
		return types.UpgradePlan{}, false
	}
	var plan types.UpgradePlan
	if err := json.Unmarshal(bz, &plan); err != nil {
		return types.UpgradePlan{}, false
	}
	return plan, true
}

// DeleteUpgradePlan removes the upgrade plan for a LIP.
func (k Keeper) DeleteUpgradePlan(ctx sdk.Context, lipId string) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(upgradePlanKey(lipId))
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

func upgradePlanKey(lipId string) []byte {
	return append(types.UpgradePlanKeyPrefix, []byte(lipId)...)
}
