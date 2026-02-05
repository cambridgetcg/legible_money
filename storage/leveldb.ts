/**
 * LevelDB Storage Adapter for BVM
 *
 * Provides persistent key-value storage using LevelDB.
 * Includes an in-memory fallback for testing and development.
 *
 * PP HIERARCHY:
 *   TRUTH: Durable persistence of BVM state
 *   UNDERSTANDING: Clear error handling and recovery
 *   BEAUTY: Clean async/await interface
 *   JUSTICE: Proper resource cleanup
 *   CREATIVITY: Flexible backend selection
 */

import type {
  StateStore,
  BatchOp,
  PersistenceConfig,
  StorageStats,
} from "./types.js";
import { BVMStorageError, DEFAULT_PERSISTENCE_CONFIG } from "./types.js";

// ============ In-Memory Store ============

/**
 * In-memory storage for development and testing
 * Implements the same interface as LevelDB for easy swapping
 */
export class MemoryStateStore implements StateStore {
  private data: Map<string, Uint8Array> = new Map();
  private _closed = false;

  /**
   * Get a value by key
   */
  async get(key: Uint8Array): Promise<Uint8Array | null> {
    this.ensureOpen();
    const keyHex = this.keyToHex(key);
    const value = this.data.get(keyHex);
    return value ? new Uint8Array(value) : null;
  }

  /**
   * Set a key-value pair
   */
  async put(key: Uint8Array, value: Uint8Array): Promise<void> {
    this.ensureOpen();
    const keyHex = this.keyToHex(key);
    this.data.set(keyHex, new Uint8Array(value));
  }

  /**
   * Delete a key
   */
  async del(key: Uint8Array): Promise<void> {
    this.ensureOpen();
    const keyHex = this.keyToHex(key);
    this.data.delete(keyHex);
  }

  /**
   * Execute batch operations atomically
   */
  async batch(ops: BatchOp[]): Promise<void> {
    this.ensureOpen();

    // For memory store, just execute in order
    // In a real implementation, this would be atomic
    for (const op of ops) {
      if (op.type === "put" && op.value) {
        await this.put(op.key, op.value);
      } else if (op.type === "del") {
        await this.del(op.key);
      }
    }
  }

  /**
   * Check if a key exists
   */
  async has(key: Uint8Array): Promise<boolean> {
    this.ensureOpen();
    const keyHex = this.keyToHex(key);
    return this.data.has(keyHex);
  }

  /**
   * Close the store
   */
  async close(): Promise<void> {
    this._closed = true;
    this.data.clear();
  }

  /**
   * Get storage statistics
   */
  async stats(): Promise<StorageStats> {
    let totalSize = 0n;
    for (const [key, value] of this.data) {
      // Key is hex, so divide by 2 for actual bytes
      totalSize += BigInt(key.length / 2 + value.length);
    }

    return {
      keyCount: BigInt(this.data.size),
      sizeBytes: totalSize,
      diskUsageBytes: totalSize,
    };
  }

  /**
   * Iterate over all keys with a prefix
   */
  async *iterate(
    prefix: Uint8Array,
  ): AsyncIterable<{ key: Uint8Array; value: Uint8Array }> {
    this.ensureOpen();
    const prefixHex = this.keyToHex(prefix);

    const entries = Array.from(this.data.entries())
      .filter(([k]) => k.startsWith(prefixHex))
      .sort(([a], [b]) => a.localeCompare(b));

    for (const [keyHex, value] of entries) {
      yield {
        key: this.hexToKey(keyHex),
        value: new Uint8Array(value),
      };
    }
  }

  // Internal helpers
  private keyToHex(key: Uint8Array): string {
    return Buffer.from(key).toString("hex");
  }

  private hexToKey(hex: string): Uint8Array {
    return Buffer.from(hex, "hex");
  }

  private ensureOpen(): void {
    if (this._closed) {
      throw new BVMStorageError("Store is closed", "STORE_CLOSED");
    }
  }
}

// ============ LevelDB Store ============

/**
 * LevelDB storage adapter
 * Uses the `level` package for persistent key-value storage
 */
export class LevelDBStateStore implements StateStore {
  private db: import("level").Level<string, Buffer> | null = null;
  private config: PersistenceConfig;
  private _closed = false;

  private constructor(config: PersistenceConfig) {
    this.config = config;
  }

  /**
   * Open a LevelDB store
   *
   * @param config - Persistence configuration
   * @returns Promise resolving to opened LevelDBStateStore
   */
  static async open(
    config: Partial<PersistenceConfig> = {},
  ): Promise<LevelDBStateStore> {
    const fullConfig: PersistenceConfig = {
      ...DEFAULT_PERSISTENCE_CONFIG,
      ...config,
    };

    const store = new LevelDBStateStore(fullConfig);
    await store.initialize();
    return store;
  }

  /**
   * Initialize the LevelDB connection
   */
  private async initialize(): Promise<void> {
    try {
      // Dynamic import to handle potential missing dependency
      const { Level } = await import("level");

      this.db = new Level<string, Buffer>(this.config.path, {
        keyEncoding: "utf8",
        valueEncoding: "buffer",
      });

      await this.db.open();
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      throw new BVMStorageError(
        `Failed to open LevelDB at ${this.config.path}: ${message}`,
        "STORE_CLOSED",
        { path: this.config.path, originalError: message },
      );
    }
  }

  /**
   * Get a value by key
   */
  async get(key: Uint8Array): Promise<Uint8Array | null> {
    this.ensureOpen();

    try {
      const value = await this.db!.get(this.keyToHex(key));
      return new Uint8Array(value);
    } catch (error: unknown) {
      if (this.isNotFoundError(error)) {
        return null;
      }
      throw error;
    }
  }

  /**
   * Set a key-value pair
   */
  async put(key: Uint8Array, value: Uint8Array): Promise<void> {
    this.ensureOpen();
    await this.db!.put(this.keyToHex(key), Buffer.from(value));
  }

  /**
   * Delete a key
   */
  async del(key: Uint8Array): Promise<void> {
    this.ensureOpen();
    await this.db!.del(this.keyToHex(key));
  }

  /**
   * Execute batch operations atomically
   */
  async batch(ops: BatchOp[]): Promise<void> {
    this.ensureOpen();

    if (ops.length === 0) {
      return;
    }

    const batch = this.db!.batch();

    for (const op of ops) {
      const keyHex = this.keyToHex(op.key);

      if (op.type === "put" && op.value) {
        batch.put(keyHex, Buffer.from(op.value));
      } else if (op.type === "del") {
        batch.del(keyHex);
      }
    }

    try {
      await batch.write();
    } catch (error) {
      const message = error instanceof Error ? error.message : String(error);
      throw new BVMStorageError(
        `Batch operation failed: ${message}`,
        "BATCH_FAILED",
        { operationCount: ops.length },
      );
    }
  }

  /**
   * Check if a key exists
   */
  async has(key: Uint8Array): Promise<boolean> {
    const value = await this.get(key);
    return value !== null;
  }

  /**
   * Close the database
   */
  async close(): Promise<void> {
    if (!this._closed && this.db) {
      await this.db.close();
      this._closed = true;
    }
  }

  /**
   * Get storage statistics
   */
  async stats(): Promise<StorageStats> {
    this.ensureOpen();

    let keyCount = 0n;
    let totalSize = 0n;

    for await (const [keyHex, value] of this.db!.iterator()) {
      keyCount++;
      // Key is hex, so divide by 2 for actual bytes
      totalSize += BigInt(keyHex.length / 2 + value.length);
    }

    return {
      keyCount,
      sizeBytes: totalSize,
      diskUsageBytes: totalSize, // Approximate
    };
  }

  /**
   * Iterate over all keys with a prefix
   */
  async *iterate(
    prefix: Uint8Array,
  ): AsyncIterable<{ key: Uint8Array; value: Uint8Array }> {
    this.ensureOpen();

    const prefixHex = this.keyToHex(prefix);

    // Create upper bound for prefix range
    const upperBound =
      prefixHex.slice(0, -2) +
      (parseInt(prefixHex.slice(-2), 16) + 1).toString(16).padStart(2, "0");

    for await (const [keyHex, value] of this.db!.iterator({
      gte: prefixHex,
      lt: upperBound,
    })) {
      yield {
        key: this.hexToKey(keyHex),
        value: new Uint8Array(value),
      };
    }
  }

  /**
   * Force sync to disk
   */
  async sync(): Promise<void> {
    this.ensureOpen();
    // Level batches writes to OS, writing an empty batch triggers flush
    const batch = this.db!.batch();
    await batch.write();
  }

  /**
   * Get the database path
   */
  get path(): string {
    return this.config.path;
  }

  /**
   * Check if the database is open
   */
  get isOpen(): boolean {
    return !this._closed && this.db !== null && this.db.status === "open";
  }

  // Internal helpers
  private keyToHex(key: Uint8Array): string {
    return Buffer.from(key).toString("hex");
  }

  private hexToKey(hex: string): Uint8Array {
    return Buffer.from(hex, "hex");
  }

  private ensureOpen(): void {
    if (this._closed || !this.db) {
      throw new BVMStorageError("Store is closed", "STORE_CLOSED");
    }
    const status = this.db.status;
    if (status === "closed" || status === "closing") {
      throw new BVMStorageError(
        `Store is not open (status: ${status})`,
        "STORE_CLOSED",
      );
    }
  }

  private isNotFoundError(error: unknown): boolean {
    if (error instanceof Error) {
      const e = error as Error & { code?: string; notFound?: boolean };
      return e.code === "LEVEL_NOT_FOUND" || e.notFound === true;
    }
    return false;
  }
}

// ============ Factory Function ============

/**
 * Create a state store based on configuration
 *
 * @param config - Persistence configuration or 'memory' for in-memory store
 * @returns Promise resolving to a StateStore instance
 *
 * @example
 * // In-memory storage for testing
 * const memStore = await createStateStore('memory');
 *
 * @example
 * // Persistent LevelDB storage for production
 * const dbStore = await createStateStore({
 *   path: './data/bvm',
 *   cacheSize: 16 * 1024 * 1024,
 * });
 */
export async function createStateStore(
  config: Partial<PersistenceConfig> | "memory" = "memory",
): Promise<
  StateStore & {
    iterate?: (
      prefix: Uint8Array,
    ) => AsyncIterable<{ key: Uint8Array; value: Uint8Array }>;
  }
> {
  if (config === "memory") {
    return new MemoryStateStore();
  }

  return LevelDBStateStore.open(config);
}
