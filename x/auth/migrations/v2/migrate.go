package v2

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/core/store"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/auth/types"
)

// MigrateStore performs in-place store migrations from v1 to v2.
//
// v2 changes:
// - RecoveryConfig entries gain an AccountAddress field (previously only stored in KV key).
// - Params gain BootstrapEnabled and BootstrapAmount fields (default values applied).
func MigrateStore(ctx sdk.Context, storeService store.KVStoreService) error {
	kvStore := storeService.OpenKVStore(ctx)

	// Migrate RecoveryConfigs: add AccountAddress field from KV key
	if err := migrateRecoveryConfigs(ctx, kvStore); err != nil {
		return fmt.Errorf("failed to migrate recovery configs: %w", err)
	}

	// Migrate Params: ensure new fields have defaults
	if err := migrateParams(ctx, kvStore); err != nil {
		return fmt.Errorf("failed to migrate params: %w", err)
	}

	return nil
}

func migrateRecoveryConfigs(ctx sdk.Context, kvStore store.KVStore) error {
	prefix := types.RecoveryConfigPrefix
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		return err
	}
	defer iter.Close()

	type entry struct {
		key     []byte
		address string
		config  *types.RecoveryConfig
	}

	var entries []entry
	for ; iter.Valid(); iter.Next() {
		config := new(types.RecoveryConfig)
		if err := json.Unmarshal(iter.Value(), config); err != nil {
			ctx.Logger().Error("skipping malformed recovery config during migration", "error", err)
			continue
		}

		// Extract address from key
		key := iter.Key()
		address := string(key[len(prefix):])

		// Only migrate if AccountAddress is empty
		if config.AccountAddress == "" {
			config.AccountAddress = address
			keyCopy := make([]byte, len(key))
			copy(keyCopy, key)
			entries = append(entries, entry{key: keyCopy, address: address, config: config})
		}
	}

	// Write back outside of iteration
	for i := range entries {
		e := entries[i]
		bz, err := json.Marshal(e.config)
		if err != nil {
			return fmt.Errorf("failed to marshal recovery config for %s: %w", e.address, err)
		}
		if err := kvStore.Set(e.key, bz); err != nil {
			return fmt.Errorf("failed to store recovery config for %s: %w", e.address, err)
		}
	}

	return nil
}

func migrateParams(ctx sdk.Context, kvStore store.KVStore) error {
	bz, err := kvStore.Get(types.ParamsKey)
	if err != nil || bz == nil {
		// No params stored — nothing to migrate
		return nil
	}

	params := new(types.Params)
	if err := json.Unmarshal(bz, params); err != nil {
		return fmt.Errorf("failed to unmarshal params: %w", err)
	}

	// Apply defaults for new fields if not already set
	defaults := types.DefaultParams()
	changed := false

	if params.BootstrapAmount == "" {
		params.BootstrapAmount = defaults.BootstrapAmount
		params.BootstrapEnabled = defaults.BootstrapEnabled
		changed = true
	}

	if changed {
		bz, err := json.Marshal(params)
		if err != nil {
			return fmt.Errorf("failed to marshal params: %w", err)
		}
		if err := kvStore.Set(types.ParamsKey, bz); err != nil {
			return fmt.Errorf("failed to store params: %w", err)
		}
	}

	return nil
}

// prefixEndBytes returns the end key for a prefix scan (exclusive upper bound).
func prefixEndBytes(prefix []byte) []byte {
	if len(prefix) == 0 {
		return nil
	}
	end := make([]byte, len(prefix))
	copy(end, prefix)
	for i := len(end) - 1; i >= 0; i-- {
		end[i]++
		if end[i] != 0 {
			return end
		}
	}
	return nil
}
