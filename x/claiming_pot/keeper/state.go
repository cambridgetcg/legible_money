package keeper

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/claiming_pot/types"
)

// --- Pot CRUD ---

// SetPot stores a claiming pot.
func (k Keeper) SetPot(ctx sdk.Context, pot *types.ClaimingPot) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(pot)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal claiming pot: %v", err))
	}
	if err := kvStore.Set(types.PotKey(pot.PotId), bz); err != nil {
		panic(fmt.Sprintf("failed to store claiming pot: %v", err))
	}
}

// GetPot retrieves a claiming pot by ID.
func (k Keeper) GetPot(ctx sdk.Context, potId string) (*types.ClaimingPot, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.PotKey(potId))
	if err != nil || bz == nil {
		return nil, false
	}
	var pot types.ClaimingPot
	if err := proto.Unmarshal(bz, &pot); err != nil {
		return nil, false
	}
	return &pot, true
}

// DeletePot removes a claiming pot.
func (k Keeper) DeletePot(ctx sdk.Context, potId string) {
	kvStore := k.storeService.OpenKVStore(ctx)
	_ = kvStore.Delete(types.PotKey(potId))
}

// IteratePots iterates over all pots. Return true from cb to stop.
func (k Keeper) IteratePots(ctx sdk.Context, cb func(*types.ClaimingPot) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	iter, err := kvStore.Iterator(types.PotKeyPrefix, prefixEndBytes(types.PotKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var pot types.ClaimingPot
		if err := proto.Unmarshal(iter.Value(), &pot); err != nil {
			continue
		}
		if cb(&pot) {
			break
		}
	}
}

// GetPotsByStatus returns all pots with a given status.
func (k Keeper) GetPotsByStatus(ctx sdk.Context, status types.PotStatus) []*types.ClaimingPot {
	var pots []*types.ClaimingPot
	k.IteratePots(ctx, func(pot *types.ClaimingPot) bool {
		if pot.Status == string(status) {
			pots = append(pots, pot)
		}
		return false
	})
	return pots
}

// GetPotsByCreator returns all pots created by a specific address.
func (k Keeper) GetPotsByCreator(ctx sdk.Context, creator string) []*types.ClaimingPot {
	var pots []*types.ClaimingPot
	k.IteratePots(ctx, func(pot *types.ClaimingPot) bool {
		if pot.Creator == creator {
			pots = append(pots, pot)
		}
		return false
	})
	return pots
}

// --- Claim Record CRUD ---

// SetClaimRecord stores a claim record.
func (k Keeper) SetClaimRecord(ctx sdk.Context, record *types.ClaimRecord) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(record)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal claim record: %v", err))
	}
	if err := kvStore.Set(types.ClaimRecordKey(record.PotId, record.Claimer), bz); err != nil {
		panic(fmt.Sprintf("failed to store claim record: %v", err))
	}
}

// GetClaimRecord retrieves a claim record.
func (k Keeper) GetClaimRecord(ctx sdk.Context, potId, claimer string) (*types.ClaimRecord, bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ClaimRecordKey(potId, claimer))
	if err != nil || bz == nil {
		return nil, false
	}
	var record types.ClaimRecord
	if err := proto.Unmarshal(bz, &record); err != nil {
		return nil, false
	}
	return &record, true
}

// IterateClaimRecords iterates over all claim records for a pot. Return true from cb to stop.
func (k Keeper) IterateClaimRecords(ctx sdk.Context, potId string, cb func(*types.ClaimRecord) bool) {
	kvStore := k.storeService.OpenKVStore(ctx)
	prefix := types.ClaimRecordPrefixKey(potId)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var record types.ClaimRecord
		if err := proto.Unmarshal(iter.Value(), &record); err != nil {
			continue
		}
		if cb(&record) {
			break
		}
	}
}

// GetAllClaimRecords returns all claim records for a pot.
func (k Keeper) GetAllClaimRecords(ctx sdk.Context, potId string) []*types.ClaimRecord {
	var records []*types.ClaimRecord
	k.IterateClaimRecords(ctx, potId, func(record *types.ClaimRecord) bool {
		records = append(records, record)
		return false
	})
	return records
}

// --- Creator Pot Count ---

// GetCreatorPotCount returns the number of pots a creator has made.
func (k Keeper) GetCreatorPotCount(ctx sdk.Context, creator string) uint64 {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.CreatorPotCountKey(creator))
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// IncrementCreatorPotCount increments a creator's pot count.
func (k Keeper) IncrementCreatorPotCount(ctx sdk.Context, creator string) {
	count := k.GetCreatorPotCount(ctx, creator) + 1
	kvStore := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	if err := kvStore.Set(types.CreatorPotCountKey(creator), bz); err != nil {
		panic(fmt.Sprintf("failed to store creator pot count: %v", err))
	}
}

// --- Params ---

// GetParams returns the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return &params
}

// SetParams sets the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := kvStore.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to store params: %v", err))
	}
}

// --- Pot Counter ---

// potCounterKey is used to generate unique pot IDs.
var potCounterKey = []byte{0x10}

// GetNextPotId returns a unique pot ID and increments the counter.
func (k Keeper) GetNextPotId(ctx sdk.Context) string {
	kvStore := k.storeService.OpenKVStore(ctx)
	bz, err := kvStore.Get(potCounterKey)
	var counter uint64
	if err == nil && bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	if err := kvStore.Set(potCounterKey, newBz); err != nil {
		panic(fmt.Sprintf("failed to store pot counter: %v", err))
	}
	return fmt.Sprintf("pot-%d", counter)
}
