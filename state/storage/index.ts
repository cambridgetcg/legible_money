/**
 * Storage Layer
 *
 * Persistent storage for blockchain state with:
 * - LevelDB/RocksDB backends
 * - State tree persistence
 * - Write-ahead logging for durability
 * - Recovery on startup
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
 *
 * GAP-001 Implementation Status:
 *   Phase 1: LevelDB integration ✅
 *   Phase 2: State tree persistence ✅
 *   Phase 3: Write-ahead logging ✅
 *   Phase 4: Recovery protocol ✅
 */

export * from "./types.js";
export * from "./leveldb.js";
export * from "./snapshots.js";
export * from "./pruning.js";
export * from "./state_persistence.js";
export * from "./write_ahead_log.js";
export * from "./recovery.js";

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
} from "./types.js";

export {
  STORAGE_NAMESPACES,
  namespacedKey,
  extractKey,
  DEFAULT_SNAPSHOT_CONFIG,
  DEFAULT_PRUNING_CONFIG,
} from "./types.js";

export {
  MemoryStore,
  LevelDBStore,
  NamespacedStore,
  createStorage,
  createNamespacedStorage,
} from "./leveldb.js";

export { SnapshotManager, SnapshotSync } from "./snapshots.js";

export { PruningEngine, AutoPruningService } from "./pruning.js";

// Phase 2: State persistence
export {
  StatePersistenceManager,
  ModifiedEntryTracker,
  STATE_NAMESPACES,
} from "./state_persistence.js";

export type { PersistenceStatistics } from "./state_persistence.js";

// Phase 3: Write-ahead logging
export { WriteAheadLog, WALTransaction } from "./write_ahead_log.js";

export type {
  WALEntry,
  WALOperation,
  WALEntryStatus,
  WALStatistics,
} from "./write_ahead_log.js";

// Phase 4: Recovery
export {
  RecoveryManager,
  verifyStateIntegrity,
  createCheckpoint,
  loadCheckpoint,
} from "./recovery.js";

export type {
  RecoveryResult,
  RecoveredState,
  StateMetadata,
  IntegrityResult,
} from "./recovery.js";
