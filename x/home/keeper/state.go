package keeper

import (
	"encoding/binary"
	"encoding/json"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/protobuf/proto"

	"github.com/legible-money/legible/x/home/types"
)

// --- Home CRUD ---

// SetHome stores an agent home.
func (k Keeper) SetHome(ctx sdk.Context, home *types.AgentHome) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(home)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal agent home: %v", err))
	}
	if err := store.Set(types.HomeKey(home.HomeId), bz); err != nil {
		panic(fmt.Sprintf("failed to set agent home: %v", err))
	}
}

// GetHome retrieves an agent home by ID.
func (k Keeper) GetHome(ctx sdk.Context, homeId string) (*types.AgentHome, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.HomeKey(homeId))
	if err != nil || bz == nil {
		return nil, false
	}
	var home types.AgentHome
	if err := proto.Unmarshal(bz, &home); err != nil {
		return nil, false
	}
	return &home, true
}

// DeleteHome removes an agent home.
func (k Keeper) DeleteHome(ctx sdk.Context, homeId string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.HomeKey(homeId)); err != nil {
		panic(fmt.Sprintf("failed to delete agent home: %v", err))
	}
}

// IterateHomes iterates over all homes. Return true from cb to stop.
func (k Keeper) IterateHomes(ctx sdk.Context, cb func(*types.AgentHome) bool) {
	store := k.storeService.OpenKVStore(ctx)
	iter, err := store.Iterator(types.HomeKeyPrefix, prefixEndBytes(types.HomeKeyPrefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var home types.AgentHome
		if err := proto.Unmarshal(iter.Value(), &home); err != nil {
			continue
		}
		if cb(&home) {
			break
		}
	}
}

// --- Home By Owner Index ---

// SetHomeByOwner stores the home-by-owner index.
func (k Keeper) SetHomeByOwner(ctx sdk.Context, ownerAddr, homeId string) {
	store := k.storeService.OpenKVStore(ctx)
	// Store as JSON array of home IDs
	key := types.HomeByOwnerKey(ownerAddr)
	var ids []string
	if bz, err := store.Get(key); err == nil && bz != nil {
		_ = json.Unmarshal(bz, &ids)
	}
	ids = append(ids, homeId)
	bz, _ := json.Marshal(ids)
	if err := store.Set(key, bz); err != nil {
		panic(fmt.Sprintf("failed to set home-by-owner index: %v", err))
	}
}

// GetHomesByOwner returns all home IDs for an owner.
func (k Keeper) GetHomesByOwner(ctx sdk.Context, ownerAddr string) []string {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.HomeByOwnerKey(ownerAddr))
	if err != nil || bz == nil {
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
func (k Keeper) SetSession(ctx sdk.Context, session *types.ActiveSession) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(session)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal session: %v", err))
	}
	if err := store.Set(types.SessionKey(session.HomeId, session.SessionId), bz); err != nil {
		panic(fmt.Sprintf("failed to set session: %v", err))
	}
}

// GetSession retrieves a session.
func (k Keeper) GetSession(ctx sdk.Context, homeId, sessionId string) (*types.ActiveSession, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.SessionKey(homeId, sessionId))
	if err != nil || bz == nil {
		return nil, false
	}
	var session types.ActiveSession
	if err := proto.Unmarshal(bz, &session); err != nil {
		return nil, false
	}
	return &session, true
}

// DeleteSession removes a session.
func (k Keeper) DeleteSession(ctx sdk.Context, homeId, sessionId string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.SessionKey(homeId, sessionId)); err != nil {
		panic(fmt.Sprintf("failed to delete session: %v", err))
	}
}

// IterateSessions iterates over all sessions for a home. Return true from cb to stop.
func (k Keeper) IterateSessions(ctx sdk.Context, homeId string, cb func(*types.ActiveSession) bool) {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.SessionPrefixKey(homeId)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var session types.ActiveSession
		if err := proto.Unmarshal(iter.Value(), &session); err != nil {
			continue
		}
		if cb(&session) {
			break
		}
	}
}

// CountSessions counts active sessions for a home.
func (k Keeper) CountSessions(ctx sdk.Context, homeId string) uint32 {
	var count uint32
	k.IterateSessions(ctx, homeId, func(_ *types.ActiveSession) bool {
		count++
		return false
	})
	return count
}

// --- Key Registration CRUD ---

// SetKeyRegistration stores a key registration.
func (k Keeper) SetKeyRegistration(ctx sdk.Context, homeId string, reg *types.KeyRegistration) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(reg)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal key registration: %v", err))
	}
	if err := store.Set(types.KeyRegKey(homeId, reg.KeyHash), bz); err != nil {
		panic(fmt.Sprintf("failed to set key registration: %v", err))
	}
}

// GetKeyRegistration retrieves a key registration.
func (k Keeper) GetKeyRegistration(ctx sdk.Context, homeId, keyHash string) (*types.KeyRegistration, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.KeyRegKey(homeId, keyHash))
	if err != nil || bz == nil {
		return nil, false
	}
	var reg types.KeyRegistration
	if err := proto.Unmarshal(bz, &reg); err != nil {
		return nil, false
	}
	return &reg, true
}

// IterateKeyRegistrations iterates over all key registrations for a home.
func (k Keeper) IterateKeyRegistrations(ctx sdk.Context, homeId string, cb func(*types.KeyRegistration) bool) {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.KeyRegPrefixKey(homeId)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var reg types.KeyRegistration
		if err := proto.Unmarshal(iter.Value(), &reg); err != nil {
			continue
		}
		if cb(&reg) {
			break
		}
	}
}

// CountKeyRegistrations counts non-revoked key registrations for a home.
func (k Keeper) CountKeyRegistrations(ctx sdk.Context, homeId string) uint32 {
	var count uint32
	k.IterateKeyRegistrations(ctx, homeId, func(reg *types.KeyRegistration) bool {
		if !reg.Revoked {
			count++
		}
		return false
	})
	return count
}

// --- Alert CRUD ---

// SetAlert stores an alert.
func (k Keeper) SetAlert(ctx sdk.Context, alert *types.Alert) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(alert)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal alert: %v", err))
	}
	if err := store.Set(types.AlertKey(alert.HomeId, alert.AlertId), bz); err != nil {
		panic(fmt.Sprintf("failed to set alert: %v", err))
	}
}

// GetAlert retrieves an alert.
func (k Keeper) GetAlert(ctx sdk.Context, homeId, alertId string) (*types.Alert, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.AlertKey(homeId, alertId))
	if err != nil || bz == nil {
		return nil, false
	}
	var alert types.Alert
	if err := proto.Unmarshal(bz, &alert); err != nil {
		return nil, false
	}
	return &alert, true
}

// DeleteAlert removes an alert.
func (k Keeper) DeleteAlert(ctx sdk.Context, homeId, alertId string) {
	store := k.storeService.OpenKVStore(ctx)
	if err := store.Delete(types.AlertKey(homeId, alertId)); err != nil {
		panic(fmt.Sprintf("failed to delete alert: %v", err))
	}
}

// IterateAlerts iterates over all alerts for a home. Return true from cb to stop.
func (k Keeper) IterateAlerts(ctx sdk.Context, homeId string, cb func(*types.Alert) bool) {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.AlertPrefixKey(homeId)
	iter, err := store.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return
	}
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		var alert types.Alert
		if err := proto.Unmarshal(iter.Value(), &alert); err != nil {
			continue
		}
		if cb(&alert) {
			break
		}
	}
}

// CountPendingAlerts counts unacknowledged alerts for a home.
func (k Keeper) CountPendingAlerts(ctx sdk.Context, homeId string) uint32 {
	var count uint32
	k.IterateAlerts(ctx, homeId, func(alert *types.Alert) bool {
		if !alert.Acknowledged {
			count++
		}
		return false
	})
	return count
}

// --- Spending Limit CRUD ---

// SetSpendingLimit stores a spending limit.
func (k Keeper) SetSpendingLimit(ctx sdk.Context, homeId string, limit *types.SpendingLimit) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := proto.Marshal(limit)
	if err != nil {
		panic(fmt.Sprintf("failed to marshal spending limit: %v", err))
	}
	if err := store.Set(types.SpendLimitKey(homeId, limit.KeyType), bz); err != nil {
		panic(fmt.Sprintf("failed to set spending limit: %v", err))
	}
}

// GetSpendingLimit retrieves a spending limit.
func (k Keeper) GetSpendingLimit(ctx sdk.Context, homeId, keyType string) (*types.SpendingLimit, bool) {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.SpendLimitKey(homeId, keyType))
	if err != nil || bz == nil {
		return nil, false
	}
	var limit types.SpendingLimit
	if err := proto.Unmarshal(bz, &limit); err != nil {
		return nil, false
	}
	return &limit, true
}

// --- Home Counter ---

// GetNextHomeId returns a unique home ID and increments the counter.
func (k Keeper) GetNextHomeId(ctx sdk.Context) string {
	store := k.storeService.OpenKVStore(ctx)
	bz, err := store.Get(types.HomeCounterKey)
	var counter uint64
	if err == nil && bz != nil {
		counter = binary.BigEndian.Uint64(bz)
	}
	counter++
	newBz := make([]byte, 8)
	binary.BigEndian.PutUint64(newBz, counter)
	if err := store.Set(types.HomeCounterKey, newBz); err != nil {
		panic(fmt.Sprintf("failed to set home counter: %v", err))
	}
	return fmt.Sprintf("home-%d", counter)
}
