package keeper

import (
	"encoding/binary"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/schedule/types"
)

// ---------- Params ----------

// SetParams sets module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(&params)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal params: %v", err))
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// GetParams returns module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// ---------- Sequence ----------

// SetSequence sets the auto-increment sequence.
func (k Keeper) SetSequence(ctx sdk.Context, seq uint64) {
	store := k.storeService.OpenKVStore(ctx)
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, seq)
	if err := store.Set(types.SequenceKey, buf); err != nil {
		panic(fmt.Sprintf("failed to set sequence: %v", err))
	}
}

// GetSequence returns the current sequence.
func (k Keeper) GetSequence(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.SequenceKey)
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// NextSequence increments and returns the next sequence.
func (k Keeper) NextSequence(ctx sdk.Context) uint64 {
	seq := k.GetSequence(ctx) + 1
	k.SetSequence(ctx, seq)
	return seq
}

// ---------- Process CRUD ----------

// SetProcess stores a schedule process with all indexes.
func (k Keeper) SetProcess(ctx sdk.Context, process types.ScheduleProcess) {
	store := k.storeService.OpenKVStore(ctx)

	// Primary key
	key := processKey(process.Id)
	bz, err := proto.Marshal(&process)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal process: %v", err))
	}
	if err := store.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set process: %v", err))
	}

	// Account index
	acctKey := accountIndexKey(process.Creator, process.Id)
	if err := store.Set(acctKey, []byte(process.Id)); err != nil {
		panic(fmt.Sprintf("failed to set account index: %v", err))
	}

	// Status index
	statusKey := statusIndexKey(string(process.Status), process.Id)
	if err := store.Set(statusKey, []byte(process.Id)); err != nil {
		panic(fmt.Sprintf("failed to set status index: %v", err))
	}

	// Time index (for next execution)
	if process.Status == types.StatusActive && process.NextExecuteAt > 0 {
		timeKey := timeIndexKey(process.NextExecuteAt, process.Id)
		if err := store.Set(timeKey, []byte(process.Id)); err != nil {
			panic(fmt.Sprintf("failed to set time index: %v", err))
		}
	}

	// Linked entity index
	if process.LinkedEntityType != "" && process.LinkedEntityId != "" {
		linkedKey := linkedEntityIndexKey(process.LinkedEntityType, process.LinkedEntityId, process.Id)
		if err := store.Set(linkedKey, []byte(process.Id)); err != nil {
			panic(fmt.Sprintf("failed to set linked entity index: %v", err))
		}
	}
}

// GetProcess retrieves a process by ID.
func (k Keeper) GetProcess(ctx sdk.Context, id string) (types.ScheduleProcess, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := processKey(id)
	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.ScheduleProcess{}, false
	}
	var process types.ScheduleProcess
	if err := proto.Unmarshal(bz, &process); err != nil {
		return types.ScheduleProcess{}, false
	}
	return process, true
}

// DeleteProcess removes a process and its indexes.
func (k Keeper) DeleteProcess(ctx sdk.Context, process types.ScheduleProcess) {
	store := k.storeService.OpenKVStore(ctx)

	_ = store.Delete(processKey(process.Id))
	_ = store.Delete(accountIndexKey(process.Creator, process.Id))
	_ = store.Delete(statusIndexKey(string(process.Status), process.Id))

	if process.NextExecuteAt > 0 {
		_ = store.Delete(timeIndexKey(process.NextExecuteAt, process.Id))
	}

	if process.LinkedEntityType != "" && process.LinkedEntityId != "" {
		_ = store.Delete(linkedEntityIndexKey(process.LinkedEntityType, process.LinkedEntityId, process.Id))
	}
}

// RemoveTimeIndex removes the time index entry for a process.
func (k Keeper) RemoveTimeIndex(ctx sdk.Context, blockHeight uint64, processId string) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(timeIndexKey(blockHeight, processId))
}

// RemoveStatusIndex removes the status index entry for a process.
func (k Keeper) RemoveStatusIndex(ctx sdk.Context, status string, processId string) {
	store := k.storeService.OpenKVStore(ctx)
	_ = store.Delete(statusIndexKey(status, processId))
}

// IterateProcesses iterates all processes. Return true from cb to stop.
func (k Keeper) IterateProcesses(ctx sdk.Context, cb func(types.ScheduleProcess) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ProcessKeyPrefix, prefixEndBytes(types.ProcessKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var process types.ScheduleProcess
		if err := proto.Unmarshal(iter.Value(), &process); err != nil {
			continue
		}
		if cb(process) {
			break
		}
	}
}

// GetProcessesByCreator returns all processes for a creator.
func (k Keeper) GetProcessesByCreator(ctx sdk.Context, creator string) []types.ScheduleProcess {
	store := k.storeService.OpenKVStore(ctx)
	prefix := append(types.ByAccountIndexPrefix, []byte(creator+":")...)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var processes []types.ScheduleProcess
	for ; iter.Valid(); iter.Next() {
		processId := string(iter.Value())
		process, found := k.GetProcess(ctx, processId)
		if found {
			processes = append(processes, process)
		}
	}
	return processes
}

// CountActiveByCreator counts active/paused schedules for an account.
func (k Keeper) CountActiveByCreator(ctx sdk.Context, creator string) uint32 {
	processes := k.GetProcessesByCreator(ctx, creator)
	var count uint32
	for _, p := range processes {
		if p.Status == types.StatusActive || p.Status == types.StatusPaused {
			count++
		}
	}
	return count
}

// GetDueProcesses returns process IDs due at or before the given block height.
func (k Keeper) GetDueProcesses(ctx sdk.Context, blockHeight uint64) []string {
	store := k.storeService.OpenKVStore(ctx)
	// Iterate from beginning of time index to blockHeight+1
	endKey := timeIndexKey(blockHeight+1, "")
	iter, err := store.Iterator(types.ByTimeIndexPrefix, endKey)
	if err != nil {
		return nil
	}
	defer iter.Close()

	var ids []string
	for ; iter.Valid(); iter.Next() {
		ids = append(ids, string(iter.Value()))
	}
	return ids
}

// ---------- Key Construction Helpers ----------

func processKey(id string) []byte {
	return append(types.ProcessKeyPrefix, []byte(id)...)
}

func timeIndexKey(blockHeight uint64, processId string) []byte {
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, blockHeight)
	key := append(types.ByTimeIndexPrefix, buf...)
	if processId != "" {
		key = append(key, []byte(processId)...)
	}
	return key
}

func accountIndexKey(creator, processId string) []byte {
	return append(types.ByAccountIndexPrefix, []byte(creator+":"+processId)...)
}

func statusIndexKey(status, processId string) []byte {
	return append(types.ByStatusIndexPrefix, []byte(status+":"+processId)...)
}

func linkedEntityIndexKey(entityType, entityId, processId string) []byte {
	return append(types.ByLinkedEntityIndexPrefix, []byte(entityType+":"+entityId+":"+processId)...)
}
