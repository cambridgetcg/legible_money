/**
 * BVM Storage Types
 *
 * Defines the interface for BVM state persistence.
 * Provides a clean abstraction over different storage backends.
 *
 * PP HIERARCHY:
 *   TRUTH: Durable state persistence for BVM
 *   UNDERSTANDING: Clear interfaces for state operations
 *   BEAUTY: Elegant abstraction over storage complexity
 *   JUSTICE: Fair resource management, proper cleanup
 *   CREATIVITY: Extensible for new storage backends
 */

// ============ Core Storage Interface ============

/**
 * Low-level key-value state store interface
 * All storage backends must implement this
 */
export interface StateStore {
  /** Get a value by key */
  get(key: Uint8Array): Promise<Uint8Array | null>;

  /** Set a key-value pair */
  put(key: Uint8Array, value: Uint8Array): Promise<void>;

  /** Delete a key */
  del(key: Uint8Array): Promise<void>;

  /** Batch operations for atomicity */
  batch(ops: BatchOp[]): Promise<void>;

  /** Close the store */
  close(): Promise<void>;

  /** Check if key exists */
  has?(key: Uint8Array): Promise<boolean>;

  /** Get storage statistics */
  stats?(): Promise<StorageStats>;
}

/**
 * Batch operation for atomic writes
 */
export interface BatchOp {
  type: "put" | "del";
  key: Uint8Array;
  value?: Uint8Array;
}

/**
 * Persistence configuration
 */
export interface PersistenceConfig {
  /** Path to the storage directory */
  path: string;

  /** LRU cache size in bytes (default: 8MB) */
  cacheSize?: number;

  /** Enable compression (default: true) */
  compression?: boolean;

  /** Write buffer size in bytes (default: 4MB) */
  writeBufferSize?: number;

  /** Maximum number of open files (default: 1000) */
  maxOpenFiles?: number;
}

/**
 * Default persistence configuration
 */
export const DEFAULT_PERSISTENCE_CONFIG: PersistenceConfig = {
  path: "./data/bvm",
  cacheSize: 8 * 1024 * 1024, // 8MB
  compression: true,
  writeBufferSize: 4 * 1024 * 1024, // 4MB
  maxOpenFiles: 1000,
};

/**
 * Storage statistics
 */
export interface StorageStats {
  /** Total number of keys */
  keyCount: bigint;

  /** Total size in bytes */
  sizeBytes: bigint;

  /** Disk usage (may differ due to compression) */
  diskUsageBytes: bigint;

  /** Number of SST files (for LSM stores) */
  sstFileCount?: number;
}

// ============ State Key Prefixes ============

/**
 * Storage key prefixes for different BVM state types
 * These ensure proper key separation and efficient range queries
 */
export const BVM_STORAGE_PREFIXES = {
  /** Contract bytecode (CB 131311) */
  CONTRACT: new Uint8Array([0x01]),

  /** Contract storage slots (CVS 131312) */
  STORAGE: new Uint8Array([0x02]),

  /** Account state (AS 131314) */
  ACCOUNT: new Uint8Array([0x03]),

  /** Agent state (AGS 131315) */
  AGENT: new Uint8Array([0x04]),

  /** Knowledge access state (KAS 131316) */
  KNOWLEDGE_ACCESS: new Uint8Array([0x05]),

  /** Schedule process state (SPS 131313) */
  SCHEDULE: new Uint8Array([0x06]),

  /** State snapshots */
  SNAPSHOT: new Uint8Array([0x10]),

  /** State metadata */
  METADATA: new Uint8Array([0xf0]),

  /** State roots */
  STATE_ROOT: new Uint8Array([0xf1]),
} as const;

export type BVMStoragePrefix =
  (typeof BVM_STORAGE_PREFIXES)[keyof typeof BVM_STORAGE_PREFIXES];

// ============ Key Construction Utilities ============

/**
 * Create a namespaced key by prepending a prefix
 */
export function createKey(prefix: Uint8Array, key: Uint8Array): Uint8Array {
  const result = new Uint8Array(prefix.length + key.length);
  result.set(prefix, 0);
  result.set(key, prefix.length);
  return result;
}

/**
 * Create a key from a string address
 */
export function addressToKey(prefix: Uint8Array, address: string): Uint8Array {
  const addressBytes = new TextEncoder().encode(address);
  return createKey(prefix, addressBytes);
}

/**
 * Create a storage slot key: PREFIX + ADDRESS + SLOT_KEY
 */
export function storageSlotKey(
  address: string,
  slotKey: Uint8Array,
): Uint8Array {
  const addressBytes = new TextEncoder().encode(address);
  const combined = new Uint8Array(addressBytes.length + 1 + slotKey.length);
  combined.set(addressBytes, 0);
  combined[addressBytes.length] = 0x00; // Separator
  combined.set(slotKey, addressBytes.length + 1);
  return createKey(BVM_STORAGE_PREFIXES.STORAGE, combined);
}

/**
 * Create a block height key (8 bytes big-endian)
 */
export function heightToKey(height: bigint): Uint8Array {
  const buffer = new ArrayBuffer(8);
  const view = new DataView(buffer);
  view.setBigUint64(0, height, false); // big-endian for proper sorting
  return new Uint8Array(buffer);
}

/**
 * Parse a block height from key bytes
 */
export function keyToHeight(key: Uint8Array): bigint {
  const view = new DataView(key.buffer, key.byteOffset, key.byteLength);
  return view.getBigUint64(0, false);
}

// ============ Serialization Utilities ============

/**
 * BigInt-safe JSON replacer
 */
export function bigIntReplacer(_key: string, value: unknown): unknown {
  if (typeof value === "bigint") {
    return { __bigint__: value.toString() };
  }
  if (value instanceof Map) {
    return {
      __map__: Array.from(value.entries()).map(([k, v]) => [k, v]),
    };
  }
  return value;
}

/**
 * BigInt-safe JSON reviver
 */
export function bigIntReviver(_key: string, value: unknown): unknown {
  if (typeof value === "object" && value !== null) {
    if ("__bigint__" in value) {
      return BigInt((value as { __bigint__: string }).__bigint__);
    }
    if ("__map__" in value) {
      return new Map((value as { __map__: [unknown, unknown][] }).__map__);
    }
  }
  return value;
}

/**
 * Encode a value to bytes using JSON serialization
 */
export function encodeValue(value: unknown): Uint8Array {
  const json = JSON.stringify(value, bigIntReplacer);
  return new TextEncoder().encode(json);
}

/**
 * Decode bytes to a value using JSON deserialization
 */
export function decodeValue<T>(data: Uint8Array): T {
  const json = new TextDecoder().decode(data);
  return JSON.parse(json, bigIntReviver) as T;
}

// ============ Error Types ============

/**
 * BVM storage error
 */
export class BVMStorageError extends Error {
  constructor(
    message: string,
    public code: BVMStorageErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "BVMStorageError";
  }
}

export type BVMStorageErrorCode =
  | "STORE_CLOSED"
  | "KEY_NOT_FOUND"
  | "SERIALIZATION_ERROR"
  | "DESERIALIZATION_ERROR"
  | "BATCH_FAILED"
  | "SNAPSHOT_NOT_FOUND"
  | "ROLLBACK_FAILED"
  | "COMMIT_FAILED"
  | "CORRUPTION_DETECTED"
  | "INVALID_STATE";
