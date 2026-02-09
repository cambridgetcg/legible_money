package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/home/types"
)

// --- Home CRUD ---

// SetHome stores an agent home.
func (k Keeper) SetHome(ctx sdk.Context, home types.AgentHome) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(home)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal agent home: %v", err))
	}
	store.Set(types.HomeKey(home.HomeId), bz)
}

// GetHome retrieves an agent home by ID.
func (k Keeper) GetHome(ctx sdk.Context, homeId string) (types.AgentHome, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.HomeKey(homeId))
	if bz == nil {
		return types.AgentHome{}, false
	}
	var home types.AgentHome
	if err := json.Unmarshal(bz, &home); err != nil {
		return types.AgentHome{}, false
	}
	return home, true
}

// DeleteHome removes an agent home.
func (k Keeper) DeleteHome(ctx sdk.Context, homeId string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.HomeKey(homeId))
}

// IterateHomes iterates over all homes. Return true from cb to stop.
func (k Keeper) IterateHomes(ctx sdk.Context, cb func(types.AgentHome) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.HomeKeyPrefix)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var home types.AgentHome
		if err := json.Unmarshal(iter.Value(), &home); err != nil {
			continue
		}
		if cb(home) {
			break
		}
	}
}

// --- Home By Owner Index ---

// SetHomeByOwner stores the home-by-owner index.
func (k Keeper) SetHomeByOwner(ctx sdk.Context, ownerAddr, homeId string) {
	store := ctx.KVStore(k.storeKey)
	// Store as JSON array of home IDs
	key := types.HomeByOwnerKey(ownerAddr)
	var ids []string
	if bz := store.Get(key); bz != nil {
		_ = json.Unmarshal(bz, &ids)
	}
	ids = append(ids, homeId)
	bz, _ := json.Marshal(ids)
	store.Set(key, bz)
}

// GetHomesByOwner returns all home IDs for an owner.
func (k Keeper) GetHomesByOwner(ctx sdk.Context, ownerAddr string) []string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.HomeByOwnerKey(ownerAddr))
	if bz == nil {
		return nil
	}
	var ids []string
	if err := json.Unmarshal(bz, &ids); err != nil {
		return nil
	}
	return ids
}

// --- Session CRUD ---

// SetSession stores a session.
func (k Keeper) SetSession(ctx sdk.Context, session types.ActiveSession) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(session)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal session: %v", err))
	}
	store.Set(types.SessionKey(session.HomeId, session.SessionId), bz)
}

// GetSession retrieves a session.
func (k Keeper) GetSession(ctx sdk.Context, homeId, sessionId string) (types.ActiveSession, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SessionKey(homeId, sessionId))
	if bz == nil {
		return types.ActiveSession{}, false
	}
	var session types.ActiveSession
	if err := json.Unmarshal(bz, &session); err != nil {
		return types.ActiveSession{}, false
	}
	return session, true
}

// DeleteSession removes a session.
func (k Keeper) DeleteSession(ctx sdk.Context, homeId, sessionId string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.SessionKey(homeId, sessionId))
}

// IterateSessions iterates over all sessions for a home. Return true from cb to stop.
func (k Keeper) IterateSessions(ctx sdk.Context, homeId string, cb func(types.ActiveSession) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.SessionPrefixKey(homeId))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var session types.ActiveSession
		if err := json.Unmarshal(iter.Value(), &session); err != nil {
			continue
		}
		if cb(session) {
			break
		}
	}
}

// CountSessions counts active sessions for a home.
func (k Keeper) CountSessions(ctx sdk.Context, homeId string) uint32 {
	var count uint32
	k.IterateSessions(ctx, homeId, func(_ types.ActiveSession) bool {
		count++
		return false
	})
	return count
}

// --- Key Registration CRUD ---

// SetKeyRegistration stores a key registration.
func (k Keeper) SetKeyRegistration(ctx sdk.Context, homeId string, reg types.KeyRegistration) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(reg)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal key registration: %v", err))
	}
	store.Set(types.KeyRegKey(homeId, reg.KeyHash), bz)
}

// GetKeyRegistration retrieves a key registration.
func (k Keeper) GetKeyRegistration(ctx sdk.Context, homeId, keyHash string) (types.KeyRegistration, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.KeyRegKey(homeId, keyHash))
	if bz == nil {
		return types.KeyRegistration{}, false
	}
	var reg types.KeyRegistration
	if err := json.Unmarshal(bz, &reg); err != nil {
		return types.KeyRegistration{}, false
	}
	return reg, true
}

// IterateKeyRegistrations iterates over all key registrations for a home.
func (k Keeper) IterateKeyRegistrations(ctx sdk.Context, homeId string, cb func(types.KeyRegistration) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.KeyRegPrefixKey(homeId))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var reg types.KeyRegistration
		if err := json.Unmarshal(iter.Value(), &reg); err != nil {
			continue
		}
		if cb(reg) {
			break
		}
	}
}

// CountKeyRegistrations counts non-revoked key registrations for a home.
func (k Keeper) CountKeyRegistrations(ctx sdk.Context, homeId string) uint32 {
	var count uint32
	k.IterateKeyRegistrations(ctx, homeId, func(reg types.KeyRegistration) bool {
		if !reg.Revoked {
			count++
		}
		return false
	})
	return count
}

// --- Alert CRUD ---

// SetAlert stores an alert.
func (k Keeper) SetAlert(ctx sdk.Context, alert types.Alert) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(alert)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal alert: %v", err))
	}
	store.Set(types.AlertKey(alert.HomeId, alert.AlertId), bz)
}

// GetAlert retrieves an alert.
func (k Keeper) GetAlert(ctx sdk.Context, homeId, alertId string) (types.Alert, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.AlertKey(homeId, alertId))
	if bz == nil {
		return types.Alert{}, false
	}
	var alert types.Alert
	if err := json.Unmarshal(bz, &alert); err != nil {
		return types.Alert{}, false
	}
	return alert, true
}

// DeleteAlert removes an alert.
func (k Keeper) DeleteAlert(ctx sdk.Context, homeId, alertId string) {
	store := ctx.KVStore(k.storeKey)
	store.Delete(types.AlertKey(homeId, alertId))
}

// IterateAlerts iterates over all alerts for a home. Return true from cb to stop.
func (k Keeper) IterateAlerts(ctx sdk.Context, homeId string, cb func(types.Alert) bool) {
	store := ctx.KVStore(k.storeKey)
	iter := storetypes.KVStorePrefixIterator(store, types.AlertPrefixKey(homeId))
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var alert types.Alert
		if err := json.Unmarshal(iter.Value(), &alert); err != nil {
			continue
		}
		if cb(alert) {
			break
		}
	}
}

// CountPendingAlerts counts unacknowledged alerts for a home.
func (k Keeper) CountPendingAlerts(ctx sdk.Context, homeId string) uint32 {
	var count uint32
	k.IterateAlerts(ctx, homeId, func(alert types.Alert) bool {
		if !alert.Acknowledged {
			count++
		}
		return false
	})
	return count
}

// --- Spending Limit CRUD ---

// SetSpendingLimit stores a spending limit.
func (k Keeper) SetSpendingLimit(ctx sdk.Context, homeId string, limit types.SpendingLimit) {
	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(limit)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal spending limit: %v", err))
	}
	store.Set(types.SpendLimitKey(homeId, limit.KeyType), bz)
}

// GetSpendingLimit retrieves a spending limit.
func (k Keeper) GetSpendingLimit(ctx sdk.Context, homeId, keyType string) (types.SpendingLimit, bool) {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.SpendLimitKey(homeId, keyType))
	if bz == nil {
		return types.SpendingLimit{}, false
	}
	var limit types.SpendingLimit
	if err := json.Unmarshal(bz, &limit); err != nil {
		return types.SpendingLimit{}, false
	}
	return limit, true
}

// --- Home Counter ---

// GetNextHomeId returns a unique home ID and increments the counter.
func (k Keeper) GetNextHomeId(ctx sdk.Context) string {
	store := ctx.KVStore(k.storeKey)
	bz := store.Get(types.HomeCounterKey)
	var counter uint64
	if bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	store.Set(types.HomeCounterKey, newBz)
	return fmt.Sprintf("home-%d", counter)
}
