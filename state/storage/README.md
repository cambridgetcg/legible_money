# Storage Module

Persistent storage layer for blockchain state.

## Overview

The storage module provides durable, verifiable state persistence with:

- **LevelDB backend** - Fast key-value storage
- **Write-ahead logging (WAL)** - Crash recovery
- **State snapshots** - Fast sync for new nodes
- **Pruning strategies** - Disk management
- **Light client proofs** - Merkle proof generation

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                    Application Layer                     │
├─────────────────────────────────────────────────────────┤
│                 StatePersistenceManager                  │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────────┐  │
│  │   Accounts  │  │  Validators │  │    Knowledge    │  │
│  └─────────────┘  └─────────────┘  └─────────────────┘  │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────┐    │
│  │              Write-Ahead Log (WAL)              │    │
│  └─────────────────────────────────────────────────┘    │
├─────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────────────────┐   │
│  │   LevelDBStore  │  │      SnapshotManager       │   │
│  └─────────────────┘  └─────────────────────────────┘   │
├─────────────────────────────────────────────────────────┤
│                    LevelDB / RocksDB                     │
└─────────────────────────────────────────────────────────┘
```

## Components

### LevelDBStore

Key-value storage with namespacing support.

```typescript
import { LevelDBStore, createStorage } from './storage';

// Create store
const store = await createStorage({
  type: 'leveldb',
  path: './data/state',
});

// Basic operations
await store.put('key', Buffer.from('value'));
const value = await store.get('key');
await store.delete('key');

// Batch writes (atomic)
const batch = store.batch();
batch.put('key1', Buffer.from('value1'));
batch.put('key2', Buffer.from('value2'));
await batch.write();
```

### NamespacedStore

Isolate different data types.

```typescript
import { createNamespacedStorage, STORAGE_NAMESPACES } from './storage';

const stores = await createNamespacedStorage('./data');

// Access by namespace
await stores.accounts.put('did:lgm:abc', accountData);
await stores.validators.put('pubkey', validatorData);
await stores.facts.put('fact:123', factData);
```

### StatePersistenceManager

Persist Sparse Merkle Trees to disk.

```typescript
import { StatePersistenceManager } from './storage';

const manager = new StatePersistenceManager(store);

// Persist state trees
await manager.persistState({
  accounts: accountTree,
  validators: validatorTree,
  knowledge: knowledgeTree,
}, blockNumber);

// Load state trees
const state = await manager.loadState(blockNumber);
```

### WriteAheadLog

Crash recovery with write-ahead logging.

```typescript
import { WriteAheadLog, WALTransaction } from './storage';

const wal = new WriteAheadLog('./data/wal');
await wal.open();

// Start transaction
const tx = new WALTransaction(wal);
await tx.begin();

try {
  await tx.put('key1', Buffer.from('value1'));
  await tx.put('key2', Buffer.from('value2'));
  await tx.commit();
} catch (error) {
  await tx.rollback();
}
```

### RecoveryManager

Recover state after crash.

```typescript
import { RecoveryManager, verifyStateIntegrity } from './storage';

const recovery = new RecoveryManager(store, wal);

// On startup
const result = await recovery.recover();
if (result.status === 'recovered') {
  console.log('Recovered to block:', result.blockNumber);
}

// Verify integrity
const integrity = await verifyStateIntegrity(store);
if (!integrity.valid) {
  console.error('Corruption detected:', integrity.errors);
}
```

### SnapshotManager

Fast sync with state snapshots.

```typescript
import { SnapshotManager, SnapshotSync } from './storage';

// Create snapshot
const snapshots = new SnapshotManager(store, {
  interval: 100,  // Every 100 blocks
  keepCount: 5,   // Keep 5 snapshots
});

await snapshots.createSnapshot(blockNumber, stateRoot);

// Sync from snapshot
const sync = new SnapshotSync(store);
const chunks = await sync.requestSnapshot(blockNumber);
await sync.applySnapshot(chunks);
```

### PruningEngine

Manage disk usage.

```typescript
import { PruningEngine, AutoPruningService } from './storage';

const pruning = new PruningEngine(store, {
  strategy: 'default',
  keepRecent: 1000,
  keepSnapshots: 5,
});

// Manual prune
const freed = await pruning.prune();
console.log('Freed:', freed, 'bytes');

// Auto-pruning service
const autoPrune = new AutoPruningService(pruning, {
  checkInterval: 3600000,  // Every hour
  threshold: 0.9,          // 90% disk usage
});
autoPrune.start();
```

## Configuration

### Storage Configuration

```typescript
interface StorageConfig {
  type: 'leveldb' | 'rocksdb' | 'memory';
  path: string;
  cacheSize?: number;      // Default: 64MB
  maxOpenFiles?: number;   // Default: 1000
  compression?: boolean;   // Default: true
}
```

### Pruning Configuration

```typescript
interface PruningConfig {
  strategy: 'nothing' | 'default' | 'everything';
  keepRecent: number;      // Blocks to keep
  keepSnapshots: number;   // Snapshots to keep
}

// Strategies:
// - nothing: Keep everything (archive node)
// - default: Keep recent + snapshots
// - everything: Minimal storage (light node)
```

### Snapshot Configuration

```typescript
interface SnapshotConfig {
  interval: number;        // Blocks between snapshots
  keepCount: number;       // Max snapshots to keep
  chunkSize: number;       // Bytes per chunk (for sync)
  compressionLevel: number;
}
```

## Namespaces

| Namespace | Prefix | Description |
|-----------|--------|-------------|
| `accounts` | `acc:` | Account states |
| `validators` | `val:` | Validator states |
| `facts` | `fct:` | Verified facts |
| `blocks` | `blk:` | Block data |
| `txindex` | `txi:` | Transaction index |
| `metadata` | `meta:` | Chain metadata |

## Error Handling

```typescript
import { StorageError, RecoveryError } from './storage';

try {
  await store.get('key');
} catch (error) {
  if (error instanceof StorageError) {
    switch (error.code) {
      case 'NOT_FOUND':
        // Key doesn't exist
        break;
      case 'CORRUPTED':
        // Data corruption
        break;
      case 'DISK_FULL':
        // No space
        break;
    }
  }
}
```

## Performance

### Batch Operations

Always use batches for multiple writes:

```typescript
// Bad: Individual writes
for (const item of items) {
  await store.put(item.key, item.value);  // Slow!
}

// Good: Batch writes
const batch = store.batch();
for (const item of items) {
  batch.put(item.key, item.value);
}
await batch.write();  // Fast!
```

### Iterator Usage

Use iterators for range queries:

```typescript
// Iterate over all accounts
for await (const { key, value } of store.iterator({
  gte: 'acc:',
  lt: 'acc:\xff',
})) {
  const account = deserialize(value);
  // Process account
}
```

## Testing

```bash
# Run storage tests
npm run test -- tests/unit/storage/

# Test specific component
npm run test -- tests/unit/storage/leveldb.test.ts
npm run test -- tests/unit/storage/persistence.test.ts
```

## Files

| File | Description |
|------|-------------|
| `types.ts` | Type definitions |
| `leveldb.ts` | LevelDB store implementation |
| `state_persistence.ts` | State tree persistence |
| `write_ahead_log.ts` | WAL for crash recovery |
| `recovery.ts` | Recovery manager |
| `snapshots.ts` | Snapshot management |
| `pruning.ts` | Disk pruning strategies |

## See Also

- [State Management](../README.md) - State layer overview
- [GLOSSARY.md](../../docs/GLOSSARY.md) - Terminology

---

*Storage module implementation (GAP-001)*
