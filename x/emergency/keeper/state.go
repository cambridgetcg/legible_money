package keeper

import (
	"encoding/binary"
	"encoding/json"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/emergency/types"
)

// --- Emergency Status ---

// GetEmergencyStatus returns the current emergency status.
func (k Keeper) GetEmergencyStatus(ctx sdk.Context) types.EmergencyStatus {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.HaltStatusKey)
	if bz == nil {
		return types.StatusNormal
	}
	return types.EmergencyStatus(bz)
}

// SetEmergencyStatus sets the current emergency status.
func (k Keeper) SetEmergencyStatus(ctx sdk.Context, status types.EmergencyStatus) {
	store := ctx.KVStore(k.storeKey)
	store.Set(types.HaltStatusKey, []byte(status))
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
func (k Keeper) SetCeremony(ctx sdk.Context, ceremony types.EmergencyCeremony) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(ceremony)
	if err != nil {
		panic("failed to marshal emergency ceremony: " + err.Error())
	}
	store.Set(types.CeremonyKey(ceremony.Id), bz)
}

// GetCeremony retrieves a ceremony by ID.
func (k Keeper) GetCeremony(ctx sdk.Context, id string) (types.EmergencyCeremony, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.CeremonyKey(id))
	if bz == nil {
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
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.CeremonyKey(id))
}

// GetActiveCeremony returns the currently active (non-terminal) ceremony, if any.
func (k Keeper) GetActiveCeremony(ctx sdk.Context) (types.EmergencyCeremony, bool) {
	var active types.EmergencyCeremony
	found := false
	k.IterateCeremonies(ctx, func(c types.EmergencyCeremony) bool {
		if c.Phase != types.PhaseFinalized && c.Phase != types.PhaseFailed {
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
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.CeremonyKeyPrefix)
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
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.ActiveHaltCeremonyIdKey)
	if bz == nil {
		return ""
	}
	return string(bz)
}

// SetActiveHaltCeremonyId stores the active halt ceremony ID.
func (k Keeper) SetActiveHaltCeremonyId(ctx sdk.Context, id string) {
	store := ctx.KVStore(k.storeKey)
	if id == "" {
		store.Delete(types.ActiveHaltCeremonyIdKey)
	} else {
		store.Set(types.ActiveHaltCeremonyIdKey, []byte(id))
	}
}

// --- Audit Log ---

// AddAuditEntry appends an audit entry to the log.
func (k Keeper) AddAuditEntry(ctx sdk.Context, entry types.EmergencyAuditEntry) {
	store := ctx.KVStore(k.storeKey)
	height := entry.BlockNumber
	// Find next available index for this height
	var index uint32
	for {
		key := types.AuditLogKey(height, index)
		if !store.Has(key) {
			bz, err := json.Marshal(entry)
			if err != nil {
				return
			}
			store.Set(key, bz)
			return
		}
		index++
	}
}

// GetAuditLog returns all audit entries.
func (k Keeper) GetAuditLog(ctx sdk.Context) []types.EmergencyAuditEntry {
	var entries []types.EmergencyAuditEntry
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.AuditLogKeyPrefix)
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
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.GuardianProposalCountKey(addr))
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// IncrementGuardianProposalCount increments a guardian's proposal count.
func (k Keeper) IncrementGuardianProposalCount(ctx sdk.Context, addr string) {
	count := k.GetGuardianProposalCount(ctx, addr) + 1
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.GuardianProposalCountKey(addr), bz)
}

// GetEpochProposalCount returns the global proposal count for this epoch.
func (k Keeper) GetEpochProposalCount(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.EpochProposalCountKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// IncrementEpochProposalCount increments the global epoch proposal count.
func (k Keeper) IncrementEpochProposalCount(ctx sdk.Context) {
	count := k.GetEpochProposalCount(ctx) + 1
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count)
	store.Set(types.EpochProposalCountKey, bz)
}

// GetLastProposalBlock returns the block height of the last proposal.
func (k Keeper) GetLastProposalBlock(ctx sdk.Context) uint64 {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.LastProposalBlockKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// SetLastProposalBlock stores the block height of the last proposal.
func (k Keeper) SetLastProposalBlock(ctx sdk.Context, block uint64) {
	store := ctx.KVStore(k.storeKey)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, block)
	store.Set(types.LastProposalBlockKey, bz)
}

// ResetEpochCounters clears all per-epoch anti-abuse tracking.
func (k Keeper) ResetEpochCounters(ctx sdk.Context) {
	store := ctx.KVStore(k.storeKey)
	// Clear epoch count
	store.Delete(types.EpochProposalCountKey)
	// Clear per-guardian counts
	iter := storetypes.KVStorePrefixIterator(store, types.GuardianProposalCountPrefix)
	defer iter.Close()
	var keys [][]byte
	for ; iter.Valid(); iter.Next() {
		keys = append(keys, iter.Key())
	}
	for _, key := range keys {
		store.Delete(key)
	}
}

// --- Params ---

// GetParams returns the emergency module parameters.
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

// SetParams stores the emergency module parameters.
func (k Keeper) SetParams(ctx sdk.Context, params types.Params) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(params)
	if err != nil {
		panic("failed to marshal emergency params: " + err.Error())
	}
	store.Set(types.ParamsKey, bz)
}
