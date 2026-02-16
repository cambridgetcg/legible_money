// Package testutil provides shared test utilities for the Legible Money chain.
package testutil

import (
	"encoding/json"
	"fmt"
	"testing"

	corestore "cosmossdk.io/core/store"
	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

// MigrationTestSuite provides reusable infrastructure for testing in-place
// store migrations. Create one per test, seed v1 data, run the migration
// function, then assert v2 data is correct.
//
// Usage:
//
//	suite := testutil.NewMigrationTestSuite(t, "my_module")
//	// Seed v1 state
//	suite.SetKV([]byte("prefix/key1"), oldJSON)
//	// Run migration
//	err := v2.MigrateStore(suite.Ctx, suite.StoreService)
//	require.NoError(t, err)
//	// Assert v2 state
//	got := suite.GetKV([]byte("prefix/key1"))
//	// ... verify got
type MigrationTestSuite struct {
	T            *testing.T
	Ctx          sdk.Context
	StoreService corestore.KVStoreService
	StoreKey     *storetypes.KVStoreKey
}

// NewMigrationTestSuite creates an in-memory KV store and SDK context
// suitable for migration testing. The storeKeyName should match the
// module's store key (e.g. "lgm_knowledge").
func NewMigrationTestSuite(t *testing.T, storeKeyName string) *MigrationTestSuite {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(storeKeyName)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, nil)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	storeService := runtime.NewKVStoreService(storeKey)

	return &MigrationTestSuite{
		T:            t,
		Ctx:          ctx,
		StoreService: storeService,
		StoreKey:     storeKey,
	}
}

// SetKV writes a key-value pair to the module's KV store.
func (s *MigrationTestSuite) SetKV(key, value []byte) {
	s.T.Helper()
	kvStore := s.StoreService.OpenKVStore(s.Ctx)
	if err := kvStore.Set(key, value); err != nil {
		s.T.Fatalf("SetKV failed: %v", err)
	}
}

// GetKV reads a value from the module's KV store. Returns nil if not found.
func (s *MigrationTestSuite) GetKV(key []byte) []byte {
	s.T.Helper()
	kvStore := s.StoreService.OpenKVStore(s.Ctx)
	bz, err := kvStore.Get(key)
	if err != nil {
		s.T.Fatalf("GetKV failed: %v", err)
	}
	return bz
}

// HasKV returns true if the key exists in the store.
func (s *MigrationTestSuite) HasKV(key []byte) bool {
	s.T.Helper()
	kvStore := s.StoreService.OpenKVStore(s.Ctx)
	has, err := kvStore.Has(key)
	if err != nil {
		s.T.Fatalf("HasKV failed: %v", err)
	}
	return has
}

// DeleteKV removes a key from the store.
func (s *MigrationTestSuite) DeleteKV(key []byte) {
	s.T.Helper()
	kvStore := s.StoreService.OpenKVStore(s.Ctx)
	if err := kvStore.Delete(key); err != nil {
		s.T.Fatalf("DeleteKV failed: %v", err)
	}
}

// SetJSON marshals v as JSON and stores it at key.
func (s *MigrationTestSuite) SetJSON(key []byte, v interface{}) {
	s.T.Helper()
	bz, err := json.Marshal(v)
	if err != nil {
		s.T.Fatalf("SetJSON marshal failed: %v", err)
	}
	s.SetKV(key, bz)
}

// GetJSON reads and unmarshals the JSON value at key into dest.
func (s *MigrationTestSuite) GetJSON(key []byte, dest interface{}) {
	s.T.Helper()
	bz := s.GetKV(key)
	if bz == nil {
		s.T.Fatalf("GetJSON: key not found")
	}
	if err := json.Unmarshal(bz, dest); err != nil {
		s.T.Fatalf("GetJSON unmarshal failed: %v", err)
	}
}

// SeedKVBatch writes multiple key-value pairs in a single call.
func (s *MigrationTestSuite) SeedKVBatch(entries map[string][]byte) {
	s.T.Helper()
	for k, v := range entries {
		s.SetKV([]byte(k), v)
	}
}

// IteratePrefix reads all keys with the given prefix and returns them.
func (s *MigrationTestSuite) IteratePrefix(prefix []byte) map[string][]byte {
	s.T.Helper()
	kvStore := s.StoreService.OpenKVStore(s.Ctx)
	iter, err := kvStore.Iterator(prefix, prefixEndBytes(prefix))
	if err != nil {
		s.T.Fatalf("IteratePrefix failed: %v", err)
	}
	defer iter.Close()

	result := make(map[string][]byte)
	for ; iter.Valid(); iter.Next() {
		keyCopy := make([]byte, len(iter.Key()))
		copy(keyCopy, iter.Key())
		valCopy := make([]byte, len(iter.Value()))
		copy(valCopy, iter.Value())
		result[string(keyCopy)] = valCopy
	}
	return result
}

// AssertKeyExists fails the test if the key does not exist.
func (s *MigrationTestSuite) AssertKeyExists(key []byte, msgAndArgs ...interface{}) {
	s.T.Helper()
	if !s.HasKV(key) {
		msg := fmt.Sprintf("expected key %x to exist", key)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf("%s: %v", msg, msgAndArgs[0])
		}
		s.T.Fatal(msg)
	}
}

// AssertKeyNotExists fails the test if the key exists.
func (s *MigrationTestSuite) AssertKeyNotExists(key []byte, msgAndArgs ...interface{}) {
	s.T.Helper()
	if s.HasKV(key) {
		msg := fmt.Sprintf("expected key %x to not exist", key)
		if len(msgAndArgs) > 0 {
			msg = fmt.Sprintf("%s: %v", msg, msgAndArgs[0])
		}
		s.T.Fatal(msg)
	}
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
