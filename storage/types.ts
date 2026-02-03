/**
 * Storage Layer Types
 *
 * Defines the interface for persistent storage backends.
 * Supports LevelDB, RocksDB, and in-memory implementations.
 *
 * PP HIERARCHY:
 *   TRUTH: Storage is the ground of truth persistence
 *   UNDERSTANDING: Clear interfaces for different storage needs
 *   BEAUTY: Elegant abstraction over multiple backends
 *   JUSTICE: Fair resource management, proper cleanup
 *   CREATIVITY: Extensible for new storage backends
 */

// ============ Core Storage Interface ============

/**
 * Key-value storage interface
 * All storage backends must implement this
 */
export interface KVStore {
  /** Get a value by key */
  get(key: Uint8Array): Promise<Uint8Array | null>;

  /** Set a key-value pair */
  put(key: Uint8Array, value: Uint8Array): Promise<void>;

  /** Delete a key */
  delete(key: Uint8Array): Promise<void>;

  /** Check if key exists */
  has(key: Uint8Array): Promise<boolean>;

  /** Batch operations for atomicity */
  batch(): WriteBatch;

  /** Create an iterator over keys */
  iterator(options?: IteratorOptions): AsyncIterator<KVPair>;

  /** Close the store */
  close(): Promise<void>;

  /** Get storage stats */
  stats(): Promise<StorageStats>;
}

/**
 * Write batch for atomic operations
 */
export interface WriteBatch {
  /** Add a put operation */
  put(key: Uint8Array, value: Uint8Array): WriteBatch;

  /** Add a delete operation */
  delete(key: Uint8Array): WriteBatch;

  /** Clear all operations */
  clear(): WriteBatch;

  /** Execute all operations atomically */
  write(): Promise<void>;
}

/**
 * Iterator options
 */
export interface IteratorOptions {
  /** Start key (inclusive) */
  gte?: Uint8Array;

  /** End key (exclusive) */
  lt?: Uint8Array;

  /** Reverse iteration */
  reverse?: boolean;

  /** Maximum entries to return */
  limit?: number;

  /** Keys only (skip values) */
  keys?: boolean;

  /** Values only (skip keys) */
  values?: boolean;
}

/**
 * Key-value pair
 */
export interface KVPair {
  key: Uint8Array;
  value: Uint8Array;
}

/**
 * Storage statistics
 */
export interface StorageStats {
  /** Total keys stored */
  keyCount: bigint;

  /** Total size in bytes */
  sizeBytes: bigint;

  /** Disk usage (may differ from sizeBytes due to compression) */
  diskUsageBytes: bigint;

  /** Number of SST files (for LSM stores) */
  sstFileCount?: number;

  /** Compaction stats */
  compactionStats?: CompactionStats;
}

/**
 * Compaction statistics for LSM-based stores
 */
export interface CompactionStats {
  /** Number of compactions completed */
  compactionsCompleted: number;

  /** Pending compaction bytes */
  pendingBytes: bigint;

  /** Current level stats */
  levels: LevelStats[];
}

export interface LevelStats {
  level: number;
  fileCount: number;
  sizeBytes: bigint;
}

// ============ Namespaced Storage ============

/**
 * Storage namespaces for different data types
 */
export const STORAGE_NAMESPACES = {
  /** Account state tree */
  ACCOUNTS: new Uint8Array([0x01]),

  /** Validator state tree */
  VALIDATORS: new Uint8Array([0x02]),

  /** Knowledge/facts state tree */
  KNOWLEDGE: new Uint8Array([0x03]),

  /** Evidence state tree */
  EVIDENCE: new Uint8Array([0x04]),

  /** Dispute state tree */
  DISPUTES: new Uint8Array([0x05]),

  /** Vesting state tree */
  VESTING: new Uint8Array([0x06]),

  /** Research state tree */
  RESEARCH: new Uint8Array([0x07]),

  /** Block headers */
  BLOCK_HEADERS: new Uint8Array([0x10]),

  /** Block bodies (transactions) */
  BLOCK_BODIES: new Uint8Array([0x11]),

  /** Block receipts */
  BLOCK_RECEIPTS: new Uint8Array([0x12]),

  /** Transaction index */
  TX_INDEX: new Uint8Array([0x13]),

  /** State snapshots */
  SNAPSHOTS: new Uint8Array([0x20]),

  /** Snapshot metadata */
  SNAPSHOT_META: new Uint8Array([0x21]),

  /** Archive data (historical states) */
  ARCHIVE: new Uint8Array([0x30]),

  /** Light client proofs */
  PROOFS: new Uint8Array([0x40]),

  /** Configuration and metadata */
  CONFIG: new Uint8Array([0xF0]),
} as const;

export type StorageNamespace = (typeof STORAGE_NAMESPACES)[keyof typeof STORAGE_NAMESPACES];

/**
 * Create a namespaced key
 */
export function namespacedKey(namespace: StorageNamespace, key: Uint8Array): Uint8Array {
  const result = new Uint8Array(namespace.length + key.length);
  result.set(namespace, 0);
  result.set(key, namespace.length);
  return result;
}

/**
 * Extract the key from a namespaced key
 */
export function extractKey(namespace: StorageNamespace, namespacedKey: Uint8Array): Uint8Array {
  return namespacedKey.slice(namespace.length);
}

// ============ Snapshot Types ============

/**
 * State snapshot for fast sync and recovery
 */
export interface StateSnapshot {
  /** Unique snapshot ID */
  id: string;

  /** Block number this snapshot represents */
  blockNumber: bigint;

  /** Block hash at this snapshot */
  blockHash: string;

  /** State root at this snapshot */
  stateRoot: string;

  /** Timestamp when snapshot was created */
  createdAt: number;

  /** Snapshot format version */
  formatVersion: number;

  /** Compressed size in bytes */
  compressedSize: bigint;

  /** Uncompressed size in bytes */
  uncompressedSize: bigint;

  /** Number of chunks */
  chunkCount: number;

  /** Chunk hashes for verification */
  chunkHashes: string[];

  /** Whether this is a full or incremental snapshot */
  snapshotType: 'full' | 'incremental';

  /** Parent snapshot ID (for incremental) */
  parentSnapshotId?: string;
}

/**
 * Snapshot chunk for streaming
 */
export interface SnapshotChunk {
  /** Snapshot ID this chunk belongs to */
  snapshotId: string;

  /** Chunk index */
  chunkIndex: number;

  /** Compressed chunk data */
  data: Uint8Array;

  /** Hash of this chunk */
  hash: string;
}

/**
 * Snapshot configuration
 */
export interface SnapshotConfig {
  /** How often to create snapshots (in blocks) */
  intervalBlocks: bigint;

  /** Maximum number of snapshots to keep */
  maxSnapshots: number;

  /** Compression algorithm */
  compression: 'none' | 'snappy' | 'zstd' | 'lz4';

  /** Compression level (1-9) */
  compressionLevel: number;

  /** Maximum chunk size in bytes */
  maxChunkSize: number;

  /** Enable incremental snapshots */
  enableIncremental: boolean;
}

export const DEFAULT_SNAPSHOT_CONFIG: SnapshotConfig = {
  intervalBlocks: 10000n,
  maxSnapshots: 5,
  compression: 'zstd',
  compressionLevel: 3,
  maxChunkSize: 64 * 1024 * 1024, // 64 MB
  enableIncremental: true,
};

// ============ Pruning Types ============

/**
 * Pruning strategy
 */
export type PruningStrategy =
  | ArchivePruning      // Keep everything
  | RetentionPruning    // Keep N blocks
  | SnapshotPruning;    // Keep snapshots only

export interface ArchivePruning {
  type: 'archive';
  // Keep all historical data
}

export interface RetentionPruning {
  type: 'retention';
  /** Number of recent blocks to keep full state for */
  retainBlocks: bigint;
  /** Keep state roots for pruned blocks (enables light client proofs) */
  keepStateRoots: boolean;
}

export interface SnapshotPruning {
  type: 'snapshot';
  /** Only keep snapshot states */
  snapshotInterval: bigint;
  /** Number of snapshots to retain */
  retainSnapshots: number;
}

/**
 * Pruning configuration
 */
export interface PruningConfig {
  /** Pruning strategy */
  strategy: PruningStrategy;

  /** Run pruning automatically */
  autoPrune: boolean;

  /** How often to run pruning (in blocks) */
  pruneIntervalBlocks: bigint;

  /** Maximum disk usage before forced pruning (bytes) */
  maxDiskUsage?: bigint;

  /** Protect certain block numbers from pruning */
  protectedBlocks?: bigint[];
}

export const DEFAULT_PRUNING_CONFIG: PruningConfig = {
  strategy: {
    type: 'retention',
    retainBlocks: 100000n, // ~3 days at 2521ms blocks
    keepStateRoots: true,
  },
  autoPrune: true,
  pruneIntervalBlocks: 1000n,
};

// ============ Migration Types ============

/**
 * Storage migration for schema changes
 */
export interface StorageMigration {
  /** Migration version */
  version: number;

  /** Migration name */
  name: string;

  /** Migration description */
  description: string;

  /** Run the migration */
  up(store: KVStore): Promise<void>;

  /** Rollback the migration */
  down(store: KVStore): Promise<void>;
}

/**
 * Migration state
 */
export interface MigrationState {
  /** Current schema version */
  currentVersion: number;

  /** Applied migrations */
  appliedMigrations: AppliedMigration[];
}

export interface AppliedMigration {
  version: number;
  name: string;
  appliedAt: number;
  duration: number;
}

// ============ Storage Provider Types ============

/**
 * Storage provider configuration
 */
export type StorageProviderConfig =
  | LevelDBConfig
  | RocksDBConfig
  | MemoryConfig;

export interface LevelDBConfig {
  type: 'leveldb';
  path: string;
  cacheSize?: number;
  writeBufferSize?: number;
  maxOpenFiles?: number;
  compression?: boolean;
}

export interface RocksDBConfig {
  type: 'rocksdb';
  path: string;
  cacheSize?: number;
  writeBufferSize?: number;
  maxOpenFiles?: number;
  compression?: 'none' | 'snappy' | 'zstd' | 'lz4';
  maxBackgroundCompactions?: number;
}

export interface MemoryConfig {
  type: 'memory';
  maxSize?: number;
}

// ============ Light Client Types ============

/**
 * State proof for light clients
 */
export interface StateProof {
  /** The key being proved */
  key: Uint8Array;

  /** The value (null if proving non-existence) */
  value: Uint8Array | null;

  /** Merkle proof path */
  proof: ProofNode[];

  /** State root this proof is against */
  stateRoot: string;

  /** Block number */
  blockNumber: bigint;
}

export interface ProofNode {
  /** Node hash */
  hash: string;

  /** Position in tree (left or right sibling) */
  position: 'left' | 'right';
}

/**
 * Block header for light client sync
 */
export interface LightBlockHeader {
  number: bigint;
  hash: string;
  parentHash: string;
  stateRoot: string;
  knowledgeRoot: string;
  timestamp: number;
  proposer: string;
  validatorSetHash: string;
}

// ============ Storage Events ============

/**
 * Storage event types
 */
export type StorageEvent =
  | { type: 'put'; key: Uint8Array; value: Uint8Array }
  | { type: 'delete'; key: Uint8Array }
  | { type: 'batch_complete'; operations: number }
  | { type: 'compaction_start'; level: number }
  | { type: 'compaction_complete'; level: number; duration: number }
  | { type: 'snapshot_created'; snapshot: StateSnapshot }
  | { type: 'pruning_complete'; freedBytes: bigint }
  | { type: 'migration_applied'; migration: AppliedMigration };

/**
 * Storage event listener
 */
export type StorageEventListener = (event: StorageEvent) => void;

// ============ Utility Types ============

/**
 * Encoding for values
 */
export type ValueEncoding = 'binary' | 'json' | 'msgpack';

/**
 * Encode a value
 */
export function encodeValue(value: unknown, encoding: ValueEncoding): Uint8Array {
  switch (encoding) {
    case 'binary':
      if (value instanceof Uint8Array) return value;
      throw new Error('Binary encoding requires Uint8Array');
    case 'json':
      return new TextEncoder().encode(JSON.stringify(value, bigIntReplacer));
    case 'msgpack':
      // Would use msgpack library
      throw new Error('msgpack encoding not implemented');
  }
}

/**
 * Decode a value
 */
export function decodeValue<T>(data: Uint8Array, encoding: ValueEncoding): T {
  switch (encoding) {
    case 'binary':
      return data as unknown as T;
    case 'json':
      return JSON.parse(new TextDecoder().decode(data), bigIntReviver);
    case 'msgpack':
      throw new Error('msgpack encoding not implemented');
  }
}

function bigIntReplacer(_key: string, value: unknown): unknown {
  return typeof value === 'bigint' ? { __bigint__: value.toString() } : value;
}

function bigIntReviver(_key: string, value: unknown): unknown {
  if (typeof value === 'object' && value !== null && '__bigint__' in value) {
    return BigInt((value as { __bigint__: string }).__bigint__);
  }
  return value;
}
