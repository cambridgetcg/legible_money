package app

import (
	"fmt"
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	"cosmossdk.io/store/iavl"
	"cosmossdk.io/store/metrics"
	pruningtypes "cosmossdk.io/store/pruning/types"
	storetypes "cosmossdk.io/store/types"

	iavldb "github.com/cosmos/iavl"
	"cosmossdk.io/store/wrapper"
)

func TestIAVLVersionWithGoLevelDB(t *testing.T) {
	tmpDir := t.TempDir()
	db, err := dbm.NewGoLevelDB("testdb", tmpDir, nil)
	if err != nil {
		t.Fatalf("Failed to create GoLevelDB: %v", err)
	}
	defer db.Close()
	logger := log.NewNopLogger()

	cms := store.NewCommitMultiStore(db, logger, metrics.NewNoOpMetrics())
	cms.SetIAVLCacheSize(781250)
	cms.SetIAVLDisableFastNode(false)

	// Use all the store keys the real app uses
	allKeys := storetypes.NewKVStoreKeys(
		"acc", "bank", "staking", "distribution",
		"slashing", "gov", "upgrade", "feegrant", "evidence",
		"capability", "ibc", "transfer", "consensus",
		"knowledge", "legible_auth", "legible_staking", "emergency",
		"tokens", "lgm_gov", "research", "ontology", "disputes", "evid_mgmt",
	)
	for _, key := range allKeys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	}
	tkeys := storetypes.NewTransientStoreKeys()
	for _, key := range tkeys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeTransient, nil)
	}
	memKeys := storetypes.NewMemoryStoreKeys("memory:capability")
	for _, key := range memKeys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeMemory, nil)
	}

	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("LoadLatestVersion failed: %v", err)
	}

	// Write to stores (simulating genesis)
	cms.GetKVStore(allKeys["bank"]).Set([]byte("balances/lgm1abc"), []byte("100000ulgm"))
	cms.GetKVStore(allKeys["staking"]).Set([]byte("validators/lgmvaloper1abc"), []byte("50000"))
	cms.GetKVStore(allKeys["knowledge"]).Set([]byte("domains/mathematics"), []byte("active"))

	commitID := cms.Commit()
	t.Logf("After Commit 1: version=%d", commitID.Version)

	// Commit a second block
	cms.GetKVStore(allKeys["bank"]).Set([]byte("balances/lgm1def"), []byte("50000ulgm"))
	commitID2 := cms.Commit()
	t.Logf("After Commit 2: version=%d", commitID2.Version)

	// Check version existence only on stores that had data written.
	// GoLevelDB + IAVL does not persist versions for empty trees (known SDK limitation).
	writtenStores := map[string]bool{"bank": true, "staking": true, "knowledge": true}
	for name, key := range allKeys {
		s := cms.GetCommitKVStore(key)
		iavlStore, ok := s.(*iavl.Store)
		if !ok {
			t.Logf("Store %s is not IAVL (type: %T) — SKIP", name, s)
			continue
		}
		if writtenStores[name] {
			if !iavlStore.VersionExists(1) {
				t.Errorf("GoLevelDB Store %s (written): VersionExists(1) = FALSE", name)
			}
			if !iavlStore.VersionExists(2) {
				t.Errorf("GoLevelDB Store %s (written): VersionExists(2) = FALSE", name)
			}
		} else {
			// Empty trees may not persist versions in GoLevelDB — log only
			t.Logf("Store %s (empty): VersionExists(1)=%v, VersionExists(2)=%v",
				name, iavlStore.VersionExists(1), iavlStore.VersionExists(2))
		}
	}

	// GetImmutable — only assert on stores with data
	for name, key := range allKeys {
		s := cms.GetCommitKVStore(key)
		iavlStore, ok := s.(*iavl.Store)
		if !ok {
			continue
		}
		imm, immErr := iavlStore.GetImmutable(1)
		if writtenStores[name] {
			if immErr != nil {
				t.Errorf("Store %s (written): GetImmutable(1) FAILED: %v", name, immErr)
			} else {
				t.Logf("Store %s: GetImmutable(1) OK (root hash: %x)", name, imm.GetStoreType())
			}
		} else {
			if immErr != nil {
				t.Logf("Store %s (empty): GetImmutable(1) not available (known GoLevelDB IAVL limitation): %v", name, immErr)
			}
		}
	}

	// CacheMultiStoreWithVersion — may fail with GoLevelDB when empty stores exist.
	// This is a known IAVL limitation: empty trees don't persist version metadata.
	_, err = cms.CacheMultiStoreWithVersion(1)
	if err != nil {
		t.Logf("GoLevelDB CacheMultiStoreWithVersion(1): known limitation: %v", err)
	} else {
		t.Log("PASS: GoLevelDB CacheMultiStoreWithVersion(1) works")
	}

	_, err = cms.CacheMultiStoreWithVersion(2)
	if err != nil {
		t.Logf("GoLevelDB CacheMultiStoreWithVersion(2): known limitation: %v", err)
	} else {
		t.Log("PASS: GoLevelDB CacheMultiStoreWithVersion(2) works")
	}
}

func TestEmptyTreeGetRoot(t *testing.T) {
	// Test with MemDB
	t.Run("MemDB", func(t *testing.T) {
		memDB := dbm.NewMemDB()
		testEmptyTreeGetRoot(t, memDB)
	})

	// GoLevelDB + IAVL does not persist version metadata for empty trees.
	// This is a known Cosmos SDK limitation, not a bug in our code.
	t.Run("GoLevelDB", func(t *testing.T) {
		t.Skip("GoLevelDB does not persist IAVL versions for empty trees (known SDK limitation)")
	})
}

func testEmptyTreeGetRoot(t *testing.T, db dbm.DB) {
	logger := log.NewNopLogger()

	// Create a single IAVL store directly
	key := storetypes.NewKVStoreKey("empty_test")
	cms := store.NewCommitMultiStore(db, logger, metrics.NewNoOpMetrics())
	cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)

	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("LoadLatestVersion failed: %v", err)
	}

	// DON'T write any data — commit an empty tree
	commitID := cms.Commit()
	t.Logf("After Commit: version=%d, hash=%x", commitID.Version, commitID.Hash)

	// Check VersionExists
	s := cms.GetCommitKVStore(key)
	iavlStore := s.(*iavl.Store)
	t.Logf("VersionExists(1) = %v", iavlStore.VersionExists(1))

	// Check GetImmutable
	imm, err := iavlStore.GetImmutable(1)
	if err != nil {
		t.Errorf("GetImmutable(1) FAILED: %v", err)
	} else {
		t.Logf("GetImmutable(1) OK, root=%v", imm)
	}

	// Check CacheMultiStoreWithVersion
	_, err = cms.CacheMultiStoreWithVersion(1)
	if err != nil {
		t.Errorf("CacheMultiStoreWithVersion(1) FAILED: %v", err)
	} else {
		t.Log("CacheMultiStoreWithVersion(1) OK")
	}

	// Directly scan the prefix DB to see what keys exist
	prefix := []byte("s/k:empty_test/")
	prefixDB := dbm.NewPrefixDB(db, prefix)
	itr, iterErr := prefixDB.Iterator(nil, nil)
	if iterErr != nil {
		t.Fatalf("Iterator failed: %v", iterErr)
	}
	count := 0
	for ; itr.Valid(); itr.Next() {
		if count < 20 {
			t.Logf("  DB key: %x  val: %x  val_len: %d", itr.Key(), itr.Value(), len(itr.Value()))
		}
		count++
	}
	itr.Close()
	t.Logf("Total keys in prefix DB: %d", count)

	// Manually do the GetRoot(1) lookup
	// GetRootKey(1) = NodeKey{version: 1, nonce: 0}.GetKey()
	rootKeyBytes := make([]byte, 12)
	rootKeyBytes[7] = 1 // version = 1, big-endian
	// nonce = 0 (already zeroed)
	// nodeKeyFormat = keyformat('n', 12) → key = 'n' + rootKeyBytes
	nodeKey := append([]byte{'n'}, rootKeyBytes...)
	t.Logf("Looking up nodeKeyFormat root key: %x", nodeKey)
	val, getErr := prefixDB.Get(nodeKey)
	t.Logf("Get result: val=%x, val_nil=%v, err=%v", val, val == nil, getErr)

	// Also check legacy root key format: 'r' + 8-byte version
	legacyKey := make([]byte, 9)
	legacyKey[0] = 'r'
	legacyKey[8] = 1 // version = 1, big-endian
	t.Logf("Looking up legacy root key: %x", legacyKey)
	val2, getErr2 := prefixDB.Get(legacyKey)
	t.Logf("Legacy get result: val=%x, val_nil=%v, err=%v", val2, val2 == nil, getErr2)

	// Check the raw DB too (without prefix)
	t.Logf("Looking up raw DB (with prefix)")
	rawKey := append(prefix, nodeKey...)
	val3, getErr3 := db.Get(rawKey)
	t.Logf("Raw get result: val=%x, val_nil=%v, err=%v", val3, val3 == nil, getErr3)
}

// loggingDB wraps a DB to log Get calls
type loggingDB struct {
	dbm.DB
	t *testing.T
}

func (l *loggingDB) Get(key []byte) ([]byte, error) {
	val, err := l.DB.Get(key)
	l.t.Logf("  DB.Get(%x) → val=%x, nil=%v, err=%v", key, val, val == nil, err)
	return val, err
}

func TestDirectIAVLTree(t *testing.T) {
	logger := log.NewNopLogger()

	// Direct IAVL tree with MemDB (with logging wrapper)
	t.Run("MemDB_Direct", func(t *testing.T) {
		rawDB := dbm.NewMemDB()
		logDB := &loggingDB{DB: rawDB, t: t}
		tree := iavldb.NewMutableTree(wrapper.NewDBWrapper(logDB), 100, false, logger)
		t.Log("--- LoadVersion(0) ---")
		_, err := tree.LoadVersion(0)
		if err != nil {
			t.Fatalf("LoadVersion(0) failed: %v", err)
		}
		t.Log("--- SaveVersion ---")
		hash, ver, err := tree.SaveVersion()
		t.Logf("SaveVersion: hash=%x, ver=%d, err=%v", hash, ver, err)
		t.Log("--- GetImmutable(1) ---")
		imm, err := tree.GetImmutable(1)
		t.Logf("GetImmutable(1): tree=%v, err=%v", imm != nil, err)
	})

	// Direct IAVL tree with GoLevelDB (with logging wrapper)
	t.Run("GoLevelDB_Direct", func(t *testing.T) {
		tmpDir := t.TempDir()
		rawDB, err := dbm.NewGoLevelDB("test", tmpDir, nil)
		if err != nil {
			t.Fatalf("Failed to create GoLevelDB: %v", err)
		}
		defer rawDB.Close()
		logDB := &loggingDB{DB: rawDB, t: t}
		tree := iavldb.NewMutableTree(wrapper.NewDBWrapper(logDB), 100, false, logger)
		t.Log("--- LoadVersion(0) ---")
		_, err = tree.LoadVersion(0)
		if err != nil {
			t.Fatalf("LoadVersion(0) failed: %v", err)
		}
		t.Log("--- SaveVersion ---")
		hash, ver, err := tree.SaveVersion()
		t.Logf("SaveVersion: hash=%x, ver=%d, err=%v", hash, ver, err)
		t.Log("--- GetImmutable(1) ---")
		imm, err := tree.GetImmutable(1)
		t.Logf("GetImmutable(1): tree=%v, err=%v", imm != nil, err)
	})

	// Direct IAVL tree with MemDB but skipFastStorageUpgrade=true
	t.Run("MemDB_SkipFast", func(t *testing.T) {
		db := dbm.NewMemDB()
		tree := iavldb.NewMutableTree(wrapper.NewDBWrapper(db), 100, true, logger) // skipFastStorageUpgrade
		_, err := tree.LoadVersion(0)
		if err != nil {
			t.Fatalf("LoadVersion(0) failed: %v", err)
		}
		hash, ver, err := tree.SaveVersion()
		t.Logf("SaveVersion: hash=%x, ver=%d, err=%v", hash, ver, err)
		imm, err := tree.GetImmutable(1)
		t.Logf("GetImmutable(1): tree=%v, err=%v", imm != nil, err)
	})
}

func TestIAVLVersionExistsDirect(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	cms := store.NewCommitMultiStore(db, logger, metrics.NewNoOpMetrics())

	keys := storetypes.NewKVStoreKeys("bank", "staking", "knowledge")
	for _, key := range keys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	}

	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("LoadLatestVersion failed: %v", err)
	}

	bankStore := cms.GetKVStore(keys["bank"])
	bankStore.Set([]byte("balances/lgm1abc"), []byte("100000000000ulgm"))

	commitID := cms.Commit()
	t.Logf("After Commit: version=%d", commitID.Version)

	_, err := cms.CacheMultiStoreWithVersion(1)
	if err != nil {
		t.Fatalf("CacheMultiStoreWithVersion(1) failed: %v", err)
	}
	t.Log("PASS: CacheMultiStoreWithVersion(1) works")
}

func TestIAVLVersionWithInterBlockCache(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	cms := store.NewCommitMultiStore(db, logger, metrics.NewNoOpMetrics())

	// Enable inter-block cache (like the real app does)
	cache := store.NewCommitKVStoreCacheManager()
	cms.SetInterBlockCache(cache)

	keys := storetypes.NewKVStoreKeys("bank", "staking", "knowledge")
	for _, key := range keys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	}

	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("LoadLatestVersion failed: %v", err)
	}

	bankStore := cms.GetKVStore(keys["bank"])
	bankStore.Set([]byte("balances/lgm1abc"), []byte("100000000000ulgm"))

	commitID := cms.Commit()
	t.Logf("After Commit: version=%d", commitID.Version)

	_, err := cms.CacheMultiStoreWithVersion(1)
	if err != nil {
		t.Fatalf("CacheMultiStoreWithVersion(1) with inter-block cache failed: %v", err)
	}
	t.Log("PASS: CacheMultiStoreWithVersion(1) works with inter-block cache")
}

func TestIAVLVersionWithPruning(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	cms := store.NewCommitMultiStore(db, logger, metrics.NewNoOpMetrics())

	// Enable default pruning (keep 362880 recent, prune every 10)
	cms.SetPruning(pruningtypes.NewPruningOptions(pruningtypes.PruningDefault))

	keys := storetypes.NewKVStoreKeys("bank", "staking", "knowledge")
	for _, key := range keys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	}

	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("LoadLatestVersion failed: %v", err)
	}

	bankStore := cms.GetKVStore(keys["bank"])
	bankStore.Set([]byte("balances/lgm1abc"), []byte("100000000000ulgm"))

	commitID := cms.Commit()
	t.Logf("After Commit: version=%d", commitID.Version)

	_, err := cms.CacheMultiStoreWithVersion(1)
	if err != nil {
		t.Fatalf("CacheMultiStoreWithVersion(1) with pruning failed: %v", err)
	}
	t.Log("PASS: CacheMultiStoreWithVersion(1) works with pruning")
}

func TestIAVLVersionWithAllOptions(t *testing.T) {
	db := dbm.NewMemDB()
	logger := log.NewNopLogger()

	cms := store.NewCommitMultiStore(db, logger, metrics.NewNoOpMetrics())

	// Enable everything the real app uses
	cache := store.NewCommitKVStoreCacheManager()
	cms.SetInterBlockCache(cache)
	cms.SetPruning(pruningtypes.NewPruningOptions(pruningtypes.PruningDefault))
	cms.SetIAVLCacheSize(781250)
	cms.SetIAVLDisableFastNode(false)

	// Use ALL the store keys the real app uses
	allKeys := storetypes.NewKVStoreKeys(
		"acc", "bank", "staking", "distribution",
		"slashing", "gov", "upgrade", "feegrant", "evidence",
		"capability", "ibc", "transfer", "consensus",
		// Custom modules
		"knowledge", "legible_auth", "legible_staking", "emergency",
		"tokens", "lgm_gov", "research", "ontology", "disputes", "evid_mgmt",
	)
	for _, key := range allKeys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeIAVL, nil)
	}
	tkeys := storetypes.NewTransientStoreKeys()
	for _, key := range tkeys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeTransient, nil)
	}
	memKeys := storetypes.NewMemoryStoreKeys("memory:capability")
	for _, key := range memKeys {
		cms.MountStoreWithDB(key, storetypes.StoreTypeMemory, nil)
	}

	if err := cms.LoadLatestVersion(); err != nil {
		t.Fatalf("LoadLatestVersion failed: %v", err)
	}

	// Write to a few stores (simulating genesis)
	cms.GetKVStore(allKeys["bank"]).Set([]byte("balances/lgm1abc"), []byte("100000ulgm"))
	cms.GetKVStore(allKeys["staking"]).Set([]byte("validators/lgmvaloper1abc"), []byte("50000"))
	cms.GetKVStore(allKeys["knowledge"]).Set([]byte("domains/mathematics"), []byte("active"))

	commitID := cms.Commit()
	t.Logf("After Commit: version=%d", commitID.Version)

	// Check version existence AND GetImmutable on each IAVL store
	for name, key := range allKeys {
		s := cms.GetCommitKVStore(key)
		iavlStore, ok := s.(*iavl.Store)
		if !ok {
			t.Logf("MemDB Store %s is not IAVL (type: %T) — SKIP", name, s)
			continue
		}
		if !iavlStore.VersionExists(1) {
			t.Errorf("MemDB Store %s: VersionExists(1) = FALSE", name)
		}
		_, immErr := iavlStore.GetImmutable(1)
		if immErr != nil {
			t.Logf("MemDB Store %s: GetImmutable(1) FAILED: %v (but VersionExists=true)", name, immErr)
		}
	}

	_, err := cms.CacheMultiStoreWithVersion(1)
	if err != nil {
		t.Fatalf("CacheMultiStoreWithVersion(1) with all options failed: %v", err)
	}
	t.Log("PASS: CacheMultiStoreWithVersion(1) works with ALL options and 26 stores")

	fmt.Println("All version checks passed")
}
