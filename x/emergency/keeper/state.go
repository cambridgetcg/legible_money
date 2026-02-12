package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/emergency/types"
)

// --- Emergency Status ---

// GetEmergencyStatus returns the current emergency status.
func (k Keeper) GetEmergencyStatus(ctx sdk.Context) types.EmergencyStatus {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.HaltStatusKey)
	if err != nil || bz == nil {
		return types.StatusNormal
	}
	return types.EmergencyStatus(bz)
}

// SetEmergencyStatus sets the current emergency status.
func (k Keeper) SetEmergencyStatus(ctx sdk.Context, status types.EmergencyStatus) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Set(types.HaltStatusKey, []byte(status)); err != nil {
		panic("failed to set emergency status: " + err.Error())
	}
}

// IsHalted returns true if the chain is in any halted state.
func (k Keeper) IsHalted(ctx sdk.Context) bool {
	status := k.GetEmergencyStatus(ctx)
	switch status {
	case types.StatusHalted, types.StatusRevertVoting, types.StatusReverting, types.StatusResumeVoting:
		return true
	default:
		return false
	}
}

// --- Ceremonies ---

// SetCeremony stores a ceremony by ID.
func (k Keeper) SetCeremony(ctx sdk.Context, ceremony types.EmergencyCeremony) error {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(ceremony)
	if err != nil {
		return fmt.Errorf("failed to marshal emergency ceremony: %w", err)
	}
	if err := store.Set(types.CeremonyKey(ceremony.Id), bz); err != nil {
		return fmt.Errorf("failed to set emergency ceremony: %w", err)
	}
	return nil
}

// GetCeremony retrieves a ceremony by ID.
func (k Keeper) GetCeremony(ctx sdk.Context, id string) (types.EmergencyCeremony, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.CeremonyKey(id))
	if err != nil || bz == nil {
		return types.EmergencyCeremony{}, false
	}
	var ceremony types.EmergencyCeremony
	if err := json.Unmarshal(bz, &ceremony); err != nil {
		return types.EmergencyCeremony{}, false
	}
	return ceremony, true
}

// DeleteCeremony removes a ceremony by ID.
func (k Keeper) DeleteCeremony(ctx sdk.Context, id string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.CeremonyKey(id)); err != nil {
		return
	}
}

// GetActiveCeremony returns the currently active (non-terminal) ceremony, if any.
func (k Keeper) GetActiveCeremony(ctx sdk.Context) (types.EmergencyCeremony, bool) {
	var active types.EmergencyCeremony
	found := false
	k.IterateCeremonies(ctx, func(c types.EmergencyCeremony) bool {
		if c.Phase != string(types.PhaseFinalized) && c.Phase != string(types.PhaseFailed) {
			active = c
			found = true
			return true // stop iteration
		}
		return false
	})
	return active, found
}

// GetAllCeremonies returns all stored ceremonies.
func (k Keeper) GetAllCeremonies(ctx sdk.Context) []types.EmergencyCeremony {
	var ceremonies []types.EmergencyCeremony
	k.IterateCeremonies(ctx, func(c types.EmergencyCeremony) bool {
		ceremonies = append(ceremonies, c)
		return false
	})
	return ceremonies
}

// IterateCeremonies iterates over all ceremonies. Return true from cb to stop.
func (k Keeper) IterateCeremonies(ctx sdk.Context, cb func(types.EmergencyCeremony) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.CeremonyKeyPrefix, prefixEndBytes(types.CeremonyKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var ceremony types.EmergencyCeremony
		if err := json.Unmarshal(iter.Value(), &ceremony); err != nil {
			continue
		}
		if cb(ceremony) {
			break
		}
	}
}

// --- Active Halt Ceremony ID ---

// GetActiveHaltCeremonyId returns the ID of the current active halt ceremony.
func (k Keeper) GetActiveHaltCeremonyId(ctx sdk.Context) string {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ActiveHaltCeremonyIdKey)
	if err != nil || bz == nil {
		return ""
	}
	return string(bz)
}

// SetActiveHaltCeremonyId stores the active halt ceremony ID.
func (k Keeper) SetActiveHaltCeremonyId(ctx sdk.Context, id string) {
	store := k.storeService.OpenKVStore(ctx)
	if id == "" {
		_ = store.Delete(types.ActiveHaltCeremonyIdKey)
	} else {
		_ = store.Set(types.ActiveHaltCeremonyIdKey, []byte(id))
	}
}

// --- Audit Log ---

// AddAuditEntry appends an audit entry to the log.
func (k Keeper) AddAuditEntry(ctx sdk.Context, entry types.EmergencyAuditEntry) {
	store := k.storeService.OpenKVStore(ctx)
	height := entry.BlockNumber
	// Find next available index for this height
	var index uint32
	for {
		key := types.AuditLogKey(height, index)
		bz, err := store.Get(key)
		if err != nil || bz == nil {
			data, err := json.Marshal(entry)
			if err != nil {
				return
			}
			_ = store.Set(key, data)
			return
		}
		index++
	}
}

// GetAuditLog returns all audit entries.
func (k Keeper) GetAuditLog(ctx sdk.Context) []types.EmergencyAuditEntry {
	var entries []types.EmergencyAuditEntry
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.AuditLogKeyPrefix, prefixEndBytes(types.AuditLogKeyPrefix))
	if err != nil {
		return entries
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var entry types.EmergencyAuditEntry
		if err := json.Unmarshal(iter.Value(), &entry); err != nil {
			continue
		}
		entries = append(entries, entry)
	}
	return entries
}

// --- Anti-Abuse Tracking ---

// GetGuardianProposalCount returns how many proposals a guardian has made this epoch.
func (k Keeper) GetGuardianProposalCount(ctx sdk.Context, addr string) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.GuardianProposalCountKey(addr))
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// IncrementGuardianProposalCount increments a guardian's proposal count.
func (k Keeper) IncrementGuardianProposalCount(ctx sdk.Context, addr string) {
	count := k.GetGuardianProposalCount(ctx, addr) + 1
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	_ = store.Set(types.GuardianProposalCountKey(addr), bz)
}

// GetEpochProposalCount returns the global proposal count for this epoch.
func (k Keeper) GetEpochProposalCount(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.EpochProposalCountKey)
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// IncrementEpochProposalCount increments the global epoch proposal count.
func (k Keeper) IncrementEpochProposalCount(ctx sdk.Context) {
	count := k.GetEpochProposalCount(ctx) + 1
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	_ = store.Set(types.EpochProposalCountKey, bz)
}

// GetLastProposalBlock returns the block height of the last proposal.
func (k Keeper) GetLastProposalBlock(ctx sdk.Context) uint64 {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.LastProposalBlockKey)
	if err != nil || bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// SetLastProposalBlock stores the block height of the last proposal.
func (k Keeper) SetLastProposalBlock(ctx sdk.Context, block uint64) {
	store := k.storeService.OpenKVStore(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, block)
	_ = store.Set(types.LastProposalBlockKey, bz)
}

// ResetEpochCounters clears all per-epoch anti-abuse tracking.
func (k Keeper) ResetEpochCounters(ctx sdk.Context) {
	store := k.storeService.OpenKVStore(ctx)
	// Clear epoch count
	_ = store.Delete(types.EpochProposalCountKey)
	// Clear per-guardian counts
	iter, err := store.Iterator(types.GuardianProposalCountPrefix, prefixEndBytes(types.GuardianProposalCountPrefix))
	if err != nil {
		return
	}
	defer iter.Close()
	var keys [][]byte
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, iter.Key())
	}
	for _, key := range keys {
		_ = store.Delete(key)
	}
}

// --- Params ---

// GetParams returns the emergency module parameters.
func (k Keeper) GetParams(ctx sdk.Context) types.Params {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.ParamsKey)
	if err != nil || bz == nil {
		return types.DefaultParams()
	}
	var params types.Params
	if err := json.Unmarshal(bz, &params); err != nil {
		return types.DefaultParams()
	}
	return params
}

// SetParams stores the emergency module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := json.Marshal(params)
	if err != nil {
		panic("failed to marshal emergency params: " + err.Error())
	}
	if err := store.Set(types.ParamsKey, bz); err != nil {
		panic("failed to set emergency params: " + err.Error())
	}
}
