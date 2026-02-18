package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/bvm/types"
)

// --- Contract CRUD ---

// SetContract stores a deployed contract and maintains the creator index.
func (k Keeper) SetContract(ctx sdk.Context, contract *types.DeployedContract) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(contract)
	if err != nil {
		panic(err)
	}
	if err := store.Set(types.ContractKey(contract.Address), bz); err != nil {
		panic(fmt.Sprintf("failed to set contract: %v", err))
	}

	// Creator index
	if err := store.Set(types.ContractCreatorIndexKey(contract.Creator, contract.Address), []byte{1}); err != nil {
		panic(fmt.Sprintf("failed to set contract creator index: %v", err))
	}
}

// GetContract retrieves a deployed contract by address.
func (k Keeper) GetContract(ctx sdk.Context, address string) (*types.DeployedContract, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ContractKey(address))
	if err != nil || bz == nil {
		return nil, false
	}
	var contract types.DeployedContract
	if err := proto.Unmarshal(bz, &contract); err != nil {
		panic(err)
	}
	return &contract, true
}

// DeleteContract removes a deployed contract and decrements the shared code's
// RefCount. If RefCount reaches zero, the code entry is garbage-collected.
func (k Keeper) DeleteContract(ctx sdk.Context, contract *types.DeployedContract) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.ContractKey(contract.Address)); err != nil {
		panic(fmt.Sprintf("failed to delete contract: %v", err))
	}
	if err := store.Delete(types.ContractCreatorIndexKey(contract.Creator, contract.Address)); err != nil {
		panic(fmt.Sprintf("failed to delete contract creator index: %v", err))
	}

	// Decrement code RefCount; garbage-collect at zero
	if contract.CodeHash != "" {
		code, found := k.GetCode(ctx, contract.CodeHash)
		if found {
			if code.RefCount <= 1 {
				k.DeleteCode(ctx, contract.CodeHash)
			} else {
				code.RefCount--
				k.SetCode(ctx, code)
			}
		}
	}
}

// IterateContracts iterates over all contracts.
func (k Keeper) IterateContracts(ctx sdk.Context, cb func(*types.DeployedContract) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ContractKeyPrefix, prefixEndBytes(types.ContractKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var contract types.DeployedContract
		if err := proto.Unmarshal(iter.Value(), &contract); err != nil {
			panic(err)
		}
		if cb(&contract) {
			break
		}
	}
}

// GetContractsByCreator returns all contracts created by a given address.
func (k Keeper) GetContractsByCreator(ctx sdk.Context, creator string) []*types.DeployedContract {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.ContractCreatorPrefix(creator)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return nil
	}
	defer iter.Close()

	var contracts []*types.DeployedContract
	for ; iter.Valid(); iter.Next() {
		// Extract contract address from index key: prefix + creator + "/" + address
		key := iter.Key()
		address := string(key[len(prefix):])
		contract, found := k.GetContract(ctx, address)
		if found {
			contracts = append(contracts, contract)
		}
	}
	return contracts
}

// GetNextContractNonce returns and increments the contract deployment counter.
func (k Keeper) GetNextContractNonce(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ContractCounterKey)
	var counter uint64
	if err == nil && bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	next := counter + 1
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, next)
	if err := store.Set(types.ContractCounterKey, buf); err != nil {
		panic(fmt.Sprintf("failed to set contract counter: %v", err))
	}
	return counter
}

// --- Code CRUD ---

// SetCode stores contract code keyed by its hash.
func (k Keeper) SetCode(ctx sdk.Context, code *types.ContractCode) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(code)
	if err != nil {
		panic(err)
	}
	if err := store.Set(types.CodeKey(code.CodeHash), bz); err != nil {
		panic(fmt.Sprintf("failed to set code: %v", err))
	}
}

// GetCode retrieves contract code by hash.
func (k Keeper) GetCode(ctx sdk.Context, codeHash string) (*types.ContractCode, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.CodeKey(codeHash))
	if err != nil || bz == nil {
		return nil, false
	}
	var code types.ContractCode
	if err := proto.Unmarshal(bz, &code); err != nil {
		panic(err)
	}
	return &code, true
}

// DeleteCode removes contract code by hash.
func (k Keeper) DeleteCode(ctx sdk.Context, codeHash string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.CodeKey(codeHash)); err != nil {
		panic(fmt.Sprintf("failed to delete code: %v", err))
	}
}

// IterateCode iterates over all stored code.
func (k Keeper) IterateCode(ctx sdk.Context, cb func(*types.ContractCode) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.CodeKeyPrefix, prefixEndBytes(types.CodeKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var code types.ContractCode
		if err := proto.Unmarshal(iter.Value(), &code); err != nil {
			panic(err)
		}
		if cb(&code) {
			break
		}
	}
}

// --- Contract State CRUD ---

// SetContractState stores a key-value entry in a contract's state.
func (k Keeper) SetContractState(ctx sdk.Context, contractAddress, key, value string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.ContractStateKey(contractAddress, key), []byte(value)); err != nil {
		panic(fmt.Sprintf("failed to set contract state: %v", err))
	}
}

// GetContractState retrieves a value from a contract's state.
func (k Keeper) GetContractState(ctx sdk.Context, contractAddress, key string) (string, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ContractStateKey(contractAddress, key))
	if err != nil || bz == nil {
		return "", false
	}
	return string(bz), true
}

// DeleteContractState removes a key-value entry from a contract's state.
func (k Keeper) DeleteContractState(ctx sdk.Context, contractAddress, key string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.ContractStateKey(contractAddress, key)); err != nil {
		panic(fmt.Sprintf("failed to delete contract state: %v", err))
	}
}

// IterateContractState iterates over all state entries for a contract.
func (k Keeper) IterateContractState(ctx sdk.Context, contractAddress string, cb func(key, value string) bool) {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.ContractStatePrefix(contractAddress)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key := string(iter.Key()[len(prefix):])
		value := string(iter.Value())
		if cb(key, value) {
			break
		}
	}
}

// CountContractState counts the number of state entries for a contract.
func (k Keeper) CountContractState(ctx sdk.Context, contractAddress string) uint64 {
	var count uint64
	k.IterateContractState(ctx, contractAddress, func(_, _ string) bool {
		count++
		return false
	})
	return count
}

// --- Schedule CRUD ---

// SetSchedule stores a contract schedule and maintains the block index.
func (k Keeper) SetSchedule(ctx sdk.Context, schedule *types.ContractSchedule) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(schedule)
	if err != nil {
		panic(err)
	}
	if err := store.Set(types.ScheduleKey(schedule.ScheduleId), bz); err != nil {
		panic(fmt.Sprintf("failed to set schedule: %v", err))
	}

	// Block index (only for pending schedules)
	if !schedule.Executed && !schedule.Cancelled {
		if err := store.Set(types.ScheduleBlockIndexKey(schedule.ExecuteAtBlock, schedule.ScheduleId), []byte{1}); err != nil {
			panic(fmt.Sprintf("failed to set schedule block index: %v", err))
		}
	}
}

// GetSchedule retrieves a schedule by ID.
func (k Keeper) GetSchedule(ctx sdk.Context, scheduleId string) (*types.ContractSchedule, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ScheduleKey(scheduleId))
	if err != nil || bz == nil {
		return nil, false
	}
	var schedule types.ContractSchedule
	if err := proto.Unmarshal(bz, &schedule); err != nil {
		panic(err)
	}
	return &schedule, true
}

// DeleteSchedule removes a schedule.
func (k Keeper) DeleteSchedule(ctx sdk.Context, schedule *types.ContractSchedule) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.ScheduleKey(schedule.ScheduleId)); err != nil {
		panic(fmt.Sprintf("failed to delete schedule: %v", err))
	}
	if err := store.Delete(types.ScheduleBlockIndexKey(schedule.ExecuteAtBlock, schedule.ScheduleId)); err != nil {
		panic(fmt.Sprintf("failed to delete schedule block index: %v", err))
	}
}

// IterateSchedules iterates over all schedules.
func (k Keeper) IterateSchedules(ctx sdk.Context, cb func(*types.ContractSchedule) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.ScheduleKeyPrefix, prefixEndBytes(types.ScheduleKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var schedule types.ContractSchedule
		if err := proto.Unmarshal(iter.Value(), &schedule); err != nil {
			panic(err)
		}
		if cb(&schedule) {
			break
		}
	}
}

// GetPendingSchedules returns all schedules due at or before the given block.
// Uses a range iterator to catch overdue schedules from chain halts.
func (k Keeper) GetPendingSchedules(ctx sdk.Context, upToBlock uint64) []*types.ContractSchedule {
	store := k.storeService.OpenKVStore(ctx)

	// Range: [ScheduleBlockIndexPrefix, ScheduleBlockIndexPrefix + Uint64ToBytes(upToBlock+1))
	start := make([]byte, len(types.ScheduleBlockIndexPrefix))
	copy(start, types.ScheduleBlockIndexPrefix)

	end := make([]byte, 0, len(types.ScheduleBlockIndexPrefix)+8)
	end = append(end, types.ScheduleBlockIndexPrefix...)
	end = append(end, types.Uint64ToBytes(upToBlock+1)...)

	iter, err := store.Iterator(start, end)
	if err != nil {
		return nil
	}
	defer iter.Close()

	// Key format: 0x11 + 8-byte block + scheduleId
	prefixLen := len(types.ScheduleBlockIndexPrefix) + 8

	var schedules []*types.ContractSchedule
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if len(key) <= prefixLen {
			continue
		}
		scheduleId := string(key[prefixLen:])
		schedule, found := k.GetSchedule(ctx, scheduleId)
		if found && !schedule.Executed && !schedule.Cancelled {
			schedules = append(schedules, schedule)
		}
	}
	return schedules
}

// GetNextScheduleId returns and increments the schedule counter.
func (k Keeper) GetNextScheduleId(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ScheduleCounterKey)
	var counter uint64
	if err == nil && bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	next := counter + 1
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, next)
	if err := store.Set(types.ScheduleCounterKey, buf); err != nil {
		panic(fmt.Sprintf("failed to set schedule counter: %v", err))
	}
	return counter
}

// CountContractSchedules counts the number of active (non-executed, non-cancelled) schedules for a contract.
func (k Keeper) CountContractSchedules(ctx sdk.Context, contractAddress string) uint64 {
	var count uint64
	k.IterateSchedules(ctx, func(s *types.ContractSchedule) bool {
		if s.ContractAddress == contractAddress && !s.Executed && !s.Cancelled {
			count++
		}
		return false
	})
	return count
}

// --- Schedule Capabilities ---

// SetScheduleCapabilities stores the caller's capabilities at schedule creation time.
// Used by P1-2 to prevent privilege escalation in scheduled executions.
func (k Keeper) SetScheduleCapabilities(ctx sdk.Context, scheduleId string, caps types.SessionCapabilities) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(caps)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal schedule capabilities: %v", err))
	}
	if err := store.Set(types.ScheduleCapabilityKey(scheduleId), bz); err != nil {
		panic(fmt.Sprintf("failed to set schedule capabilities: %v", err))
	}
}

// GetScheduleCapabilities retrieves the stored capabilities for a schedule.
// Returns false if no capabilities are stored (pre-fix schedules).
func (k Keeper) GetScheduleCapabilities(ctx sdk.Context, scheduleId string) (types.SessionCapabilities, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ScheduleCapabilityKey(scheduleId))
	if err != nil || bz == nil {
		return types.SessionCapabilities{}, false
	}
	var caps types.SessionCapabilities
	if err := json.Unmarshal(bz, &caps); err != nil {
		return types.SessionCapabilities{}, false
	}
	return caps, true
}

// --- Gas Schedule ---

// SetGasSchedule stores a governance gas schedule override.
// The schedule maps opcode mnemonics to gas costs.
func (k Keeper) SetGasSchedule(ctx sdk.Context, schedule map[string]uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(schedule)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal gas schedule: %v", err))
	}
	if err := store.Set(types.GasScheduleKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set gas schedule: %v", err))
	}
}

// GetGasSchedule retrieves the governance gas schedule override.
// Returns nil if no schedule is set (use default opcode costs).
func (k Keeper) GetGasSchedule(ctx sdk.Context) map[string]uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.GasScheduleKey)
	if err != nil || bz == nil {
		return nil
	}
	var schedule map[string]uint64
	if err := json.Unmarshal(bz, &schedule); err != nil {
		return nil
	}
	return schedule
}

// --- Params ---

// SetParams stores the module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params *types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(params)
	if err != nil {
		panic(err)
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic(fmt.Sprintf("failed to set params: %v", err))
	}
}

// GetParams retrieves the module parameters.
func (k Keeper) GetParams(ctx sdk.Context) *types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		p := types.DefaultParams()
		return &p
	}
	var params types.Params
	if err := proto.Unmarshal(bz, &params); err != nil {
		panic(err)
	}
	return &params
}
