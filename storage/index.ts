/**
 * Storage Layer
 *
 * Persistent storage for blockchain state with:
 * - LevelDB/RocksDB backends
 * - State snapshots for fast sync
 * - Pruning strategies for disk management
 * - Light client proof generation
 *
 * PP HIERARCHY:
 *   TRUTH: Durable, verifiable state persistence
 *   UNDERSTANDING: Clear APIs for different use cases
 *   BEAUTY: Elegant abstraction over storage complexity
 *   JUSTICE: Fair resource management
 *   CREATIVITY: Extensible for future backends
 */

export * from './types.js';
export * from './leveldb.js';
export * from './snapshots.js';
export * from './pruning.js';

// Re-export key types for convenience
export type {
  KVStore,
  WriteBatch,
  IteratorOptions,
  KVPair,
  StorageStats,
  StateSnapshot,
  SnapshotChunk,
  SnapshotConfig,
  PruningConfig,
  PruningStrategy,
  StateProof,
  LightBlockHeader,
} from './types.js';

export {
  STORAGE_NAMESPACES,
  namespacedKey,
  extractKey,
  DEFAULT_SNAPSHOT_CONFIG,
  DEFAULT_PRUNING_CONFIG,
} from './types.js';

export {
  MemoryStore,
  LevelDBStore,
  NamespacedStore,
  createStorage,
} from './leveldb.js';

export {
  SnapshotManager,
  SnapshotSync,
} from './snapshots.js';

export {
  PruningEngine,
  AutoPruningService,
} from './pruning.js';
