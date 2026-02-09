package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/claiming_pot/types"
)

// --- Pot CRUD ---

// SetPot stores a claiming pot.
func (k Keeper) SetPot(ctx sdk.Context, pot types.ClaimingPot) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(pot)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal claiming pot: %v", err))
	}
	store.Set(types.PotKey(pot.PotId), bz)
}

// GetPot retrieves a claiming pot by ID.
func (k Keeper) GetPot(ctx sdk.Context, potId string) (types.ClaimingPot, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.PotKey(potId))
	if bz == nil {
		return types.ClaimingPot{}, false
	}
	var pot types.ClaimingPot
	if err := json.Unmarshal(bz, &pot); err != nil {
		return types.ClaimingPot{}, false
	}
	return pot, true
}

// DeletePot removes a claiming pot.
func (k Keeper) DeletePot(ctx sdk.Context, potId string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.PotKey(potId))
}

// IteratePots iterates over all pots. Return true from cb to stop.
func (k Keeper) IteratePots(ctx sdk.Context, cb func(types.ClaimingPot) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.PotKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var pot types.ClaimingPot
		if err := json.Unmarshal(iter.Value(), &pot); err != nil {
			continue
		}
		if cb(pot) {
			break
		}
	}
}

// GetPotsByStatus returns all pots with a given status.
func (k Keeper) GetPotsByStatus(ctx sdk.Context, status types.PotStatus) []types.ClaimingPot {
	var pots []types.ClaimingPot
	k.IteratePots(ctx, func(pot types.ClaimingPot) bool {
		if pot.Status == status {
			pots = append(pots, pot)
		}
		return false
	})
	return pots
}

// GetPotsByCreator returns all pots created by a specific address.
func (k Keeper) GetPotsByCreator(ctx sdk.Context, creator string) []types.ClaimingPot {
	var pots []types.ClaimingPot
	k.IteratePots(ctx, func(pot types.ClaimingPot) bool {
		if pot.Creator == creator {
			pots = append(pots, pot)
		}
		return false
	})
	return pots
}

// --- Claim Record CRUD ---

// SetClaimRecord stores a claim record.
func (k Keeper) SetClaimRecord(ctx sdk.Context, record types.ClaimRecord) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal claim record: %v", err))
	}
	store.Set(types.ClaimRecordKey(record.PotId, record.Claimer), bz)
}

// GetClaimRecord retrieves a claim record.
func (k Keeper) GetClaimRecord(ctx sdk.Context, potId, claimer string) (types.ClaimRecord, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ClaimRecordKey(potId, claimer))
	if bz == nil {
		return types.ClaimRecord{}, false
	}
	var record types.ClaimRecord
	if err := json.Unmarshal(bz, &record); err != nil {
		return types.ClaimRecord{}, false
	}
	return record, true
}

// IterateClaimRecords iterates over all claim records for a pot. Return true from cb to stop.
func (k Keeper) IterateClaimRecords(ctx sdk.Context, potId string, cb func(types.ClaimRecord) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.ClaimRecordPrefixKey(potId))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var record types.ClaimRecord
		if err := json.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		if cb(record) {
			break
		}
	}
}

// GetAllClaimRecords returns all claim records for a pot.
func (k Keeper) GetAllClaimRecords(ctx sdk.Context, potId string) []types.ClaimRecord {
	var records []types.ClaimRecord
	k.IterateClaimRecords(ctx, potId, func(record types.ClaimRecord) bool {
		records = append(records, record)
		return false
	})
	return records
}

// --- Creator Pot Count ---

// GetCreatorPotCount returns the number of pots a creator has made.
func (k Keeper) GetCreatorPotCount(ctx sdk.Context, creator string) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CreatorPotCountKey(creator))
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// IncrementCreatorPotCount increments a creator's pot count.
func (k Keeper) IncrementCreatorPotCount(ctx sdk.Context, creator string) {
	count := k.GetCreatorPotCount(ctx, creator) + 1
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.CreatorPotCountKey(creator), bz)
}

// --- Params ---

// GetParams returns the module parameters.
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

// SetParams sets the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	store.Set(types.ParamsKey, bz)
}

// --- Pot Counter ---

// potCounterKey is used to generate unique pot IDs.
var potCounterKey = []byte{0x10}

// GetNextPotId returns a unique pot ID and increments the counter.
func (k Keeper) GetNextPotId(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(potCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(potCounterKey, newBz)
	return fmt.Sprintf("pot-%d", counter)
}
