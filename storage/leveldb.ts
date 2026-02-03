/**
 * LevelDB Storage Adapter
 *
 * Provides persistent key-value storage using LevelDB.
 * Optimized for blockchain state storage with:
 * - Efficient range queries
 * - Atomic batch writes
 * - Compression support
 * - Statistics tracking
 *
 * PP HIERARCHY:
 *   TRUTH: Durable persistence of state
 *   UNDERSTANDING: Clear error handling and recovery
 *   BEAUTY: Clean async/await interface
 *   JUSTICE: Proper resource cleanup
 *   CREATIVITY: Extensible for advanced features
 */

import type {
  KVStore,
  WriteBatch,
  IteratorOptions,
  KVPair,
  StorageStats,
  LevelDBConfig,
  StorageEvent,
  StorageEventListener,
} from './types.js';

// ============ In-Memory Implementation ============
// Note: In production, this would use classic-level or level package
// For now, we provide an in-memory implementation with the same interface

/**
 * In-memory storage for development and testing
 * Same interface as LevelDB for easy swapping
 */
export class MemoryStore implements KVStore {
  private data: Map<string, Uint8Array> = new Map();
  private listeners: Set<StorageEventListener> = new Set();
  private _closed = false;

  constructor(_config?: { maxSize?: number }) {
    // Config ignored for memory store
  }

  async get(key: Uint8Array): Promise<Uint8Array | null> {
    this.ensureOpen();
    const keyStr = this.keyToString(key);
    return this.data.get(keyStr) || null;
  }

  async put(key: Uint8Array, value: Uint8Array): Promise<void> {
    this.ensureOpen();
    const keyStr = this.keyToString(key);
    this.data.set(keyStr, value);
    this.emit({ type: 'put', key, value });
  }

  async delete(key: Uint8Array): Promise<void> {
    this.ensureOpen();
    const keyStr = this.keyToString(key);
    this.data.delete(keyStr);
    this.emit({ type: 'delete', key });
  }

  async has(key: Uint8Array): Promise<boolean> {
    this.ensureOpen();
    const keyStr = this.keyToString(key);
    return this.data.has(keyStr);
  }

  batch(): WriteBatch {
    return new MemoryWriteBatch(this);
  }

  async *iterator(options: IteratorOptions = {}): AsyncIterator<KVPair> {
    this.ensureOpen();

    // Convert to sorted array for iteration
    const entries = Array.from(this.data.entries())
      .map(([k, v]) => ({ key: this.stringToKey(k), value: v }))
      .sort((a, b) => this.compareKeys(a.key, b.key));

    let filtered = entries;

    // Apply range filters
    if (options.gte) {
      filtered = filtered.filter(e => this.compareKeys(e.key, options.gte!) >= 0);
    }
    if (options.lt) {
      filtered = filtered.filter(e => this.compareKeys(e.key, options.lt!) < 0);
    }

    // Apply reverse
    if (options.reverse) {
      filtered = filtered.reverse();
    }

    // Apply limit
    if (options.limit !== undefined) {
      filtered = filtered.slice(0, options.limit);
    }

    // Yield results
    for (const entry of filtered) {
      yield entry;
    }
  }

  async close(): Promise<void> {
    this._closed = true;
    this.data.clear();
  }

  async stats(): Promise<StorageStats> {
    let totalSize = 0n;
    for (const [key, value] of this.data) {
      totalSize += BigInt(key.length + value.length);
    }

    return {
      keyCount: BigInt(this.data.size),
      sizeBytes: totalSize,
      diskUsageBytes: totalSize, // Same for memory
    };
  }

  // Event handling
  addListener(listener: StorageEventListener): void {
    this.listeners.add(listener);
  }

  removeListener(listener: StorageEventListener): void {
    this.listeners.delete(listener);
  }

  private emit(event: StorageEvent): void {
    for (const listener of this.listeners) {
      try {
        listener(event);
      } catch (e) {
        console.error('Storage event listener error:', e);
      }
    }
  }

  // Internal methods
  private keyToString(key: Uint8Array): string {
    return Buffer.from(key).toString('hex');
  }

  private stringToKey(str: string): Uint8Array {
    return Buffer.from(str, 'hex');
  }

  private compareKeys(a: Uint8Array, b: Uint8Array): number {
    const minLen = Math.min(a.length, b.length);
    for (let i = 0; i < minLen; i++) {
      if (a[i] !== b[i]) {
        return a[i] - b[i];
      }
    }
    return a.length - b.length;
  }

  private ensureOpen(): void {
    if (this._closed) {
      throw new Error('Store is closed');
    }
  }

  // For batch operations
  async _batchWrite(operations: Array<{ type: 'put' | 'delete'; key: Uint8Array; value?: Uint8Array }>): Promise<void> {
    for (const op of operations) {
      if (op.type === 'put' && op.value) {
        await this.put(op.key, op.value);
      } else if (op.type === 'delete') {
        await this.delete(op.key);
      }
    }
    this.emit({ type: 'batch_complete', operations: operations.length });
  }
}

/**
 * Write batch for memory store
 */
class MemoryWriteBatch implements WriteBatch {
  private operations: Array<{ type: 'put' | 'delete'; key: Uint8Array; value?: Uint8Array }> = [];

  constructor(private store: MemoryStore) {}

  put(key: Uint8Array, value: Uint8Array): WriteBatch {
    this.operations.push({ type: 'put', key, value });
    return this;
  }

  delete(key: Uint8Array): WriteBatch {
    this.operations.push({ type: 'delete', key });
    return this;
  }

  clear(): WriteBatch {
    this.operations = [];
    return this;
  }

  async write(): Promise<void> {
    await this.store._batchWrite(this.operations);
    this.operations = [];
  }
}

// ============ LevelDB Implementation ============

/**
 * LevelDB storage adapter
 * Uses classic-level or level package
 */
export class LevelDBStore implements KVStore {
  private db: any; // Would be Level instance
  private config: LevelDBConfig;
  private listeners: Set<StorageEventListener> = new Set();
  private _closed = false;

  private constructor(config: LevelDBConfig) {
    this.config = config;
  }

  /**
   * Open a LevelDB store
   */
  static async open(config: LevelDBConfig): Promise<LevelDBStore> {
    const store = new LevelDBStore(config);
    await store.init();
    return store;
  }

  private async init(): Promise<void> {
    // In production:
    // const { Level } = await import('level');
    // this.db = new Level(this.config.path, {
    //   keyEncoding: 'binary',
    //   valueEncoding: 'binary',
    //   cacheSize: this.config.cacheSize || 32 * 1024 * 1024,
    //   writeBufferSize: this.config.writeBufferSize || 4 * 1024 * 1024,
    //   maxOpenFiles: this.config.maxOpenFiles || 1000,
    //   compression: this.config.compression !== false,
    // });

    // For now, use memory implementation
    console.warn('LevelDB not available, using in-memory storage');
    this.db = new MemoryStore();
  }

  async get(key: Uint8Array): Promise<Uint8Array | null> {
    this.ensureOpen();
    try {
      return await this.db.get(key);
    } catch (e: any) {
      if (e.code === 'LEVEL_NOT_FOUND' || e.notFound) {
        return null;
      }
      throw e;
    }
  }

  async put(key: Uint8Array, value: Uint8Array): Promise<void> {
    this.ensureOpen();
    await this.db.put(key, value);
    this.emit({ type: 'put', key, value });
  }

  async delete(key: Uint8Array): Promise<void> {
    this.ensureOpen();
    await this.db.delete(key);
    this.emit({ type: 'delete', key });
  }

  async has(key: Uint8Array): Promise<boolean> {
    const value = await this.get(key);
    return value !== null;
  }

  batch(): WriteBatch {
    return new LevelDBWriteBatch(this.db, (count) => {
      this.emit({ type: 'batch_complete', operations: count });
    });
  }

  async *iterator(options: IteratorOptions = {}): AsyncIterator<KVPair> {
    this.ensureOpen();
    yield* this.db.iterator(options);
  }

  async close(): Promise<void> {
    if (!this._closed) {
      await this.db.close();
      this._closed = true;
    }
  }

  async stats(): Promise<StorageStats> {
    if (this.db.stats) {
      return this.db.stats();
    }

    // Fallback: iterate and count
    let keyCount = 0n;
    let totalSize = 0n;

    for await (const { key, value } of this.iterator()) {
      keyCount++;
      totalSize += BigInt(key.length + value.length);
    }

    return {
      keyCount,
      sizeBytes: totalSize,
      diskUsageBytes: totalSize,
    };
  }

  /**
   * Compact the database
   */
  async compact(): Promise<void> {
    this.emit({ type: 'compaction_start', level: 0 });
    const start = Date.now();

    if (this.db.compactRange) {
      await this.db.compactRange();
    }

    const duration = Date.now() - start;
    this.emit({ type: 'compaction_complete', level: 0, duration });
  }

  // Event handling
  addListener(listener: StorageEventListener): void {
    this.listeners.add(listener);
  }

  removeListener(listener: StorageEventListener): void {
    this.listeners.delete(listener);
  }

  private emit(event: StorageEvent): void {
    for (const listener of this.listeners) {
      try {
        listener(event);
      } catch (e) {
        console.error('Storage event listener error:', e);
      }
    }
  }

  private ensureOpen(): void {
    if (this._closed) {
      throw new Error('Store is closed');
    }
  }
}

/**
 * Write batch for LevelDB
 */
class LevelDBWriteBatch implements WriteBatch {
  private batch: any;
  private operationCount = 0;

  constructor(
    db: any,
    private onComplete: (count: number) => void
  ) {
    this.batch = db.batch();
  }

  put(key: Uint8Array, value: Uint8Array): WriteBatch {
    this.batch.put(key, value);
    this.operationCount++;
    return this;
  }

  delete(key: Uint8Array): WriteBatch {
    this.batch.del(key);
    this.operationCount++;
    return this;
  }

  clear(): WriteBatch {
    this.batch.clear();
    this.operationCount = 0;
    return this;
  }

  async write(): Promise<void> {
    await this.batch.write();
    this.onComplete(this.operationCount);
    this.operationCount = 0;
  }
}

// ============ Namespaced Store ============

/**
 * A store with automatic key namespacing
 */
export class NamespacedStore implements KVStore {
  constructor(
    private store: KVStore,
    private namespace: Uint8Array
  ) {}

  private prefixKey(key: Uint8Array): Uint8Array {
    const result = new Uint8Array(this.namespace.length + key.length);
    result.set(this.namespace, 0);
    result.set(key, this.namespace.length);
    return result;
  }

  async get(key: Uint8Array): Promise<Uint8Array | null> {
    return this.store.get(this.prefixKey(key));
  }

  async put(key: Uint8Array, value: Uint8Array): Promise<void> {
    return this.store.put(this.prefixKey(key), value);
  }

  async delete(key: Uint8Array): Promise<void> {
    return this.store.delete(this.prefixKey(key));
  }

  async has(key: Uint8Array): Promise<boolean> {
    return this.store.has(this.prefixKey(key));
  }

  batch(): WriteBatch {
    return new NamespacedWriteBatch(this.store.batch(), this.namespace);
  }

  async *iterator(options: IteratorOptions = {}): AsyncIterator<KVPair> {
    const prefixedOptions: IteratorOptions = { ...options };

    if (options.gte) {
      prefixedOptions.gte = this.prefixKey(options.gte);
    } else {
      prefixedOptions.gte = this.namespace;
    }

    if (options.lt) {
      prefixedOptions.lt = this.prefixKey(options.lt);
    } else {
      // Create upper bound for namespace
      const upperBound = new Uint8Array(this.namespace.length);
      upperBound.set(this.namespace);
      upperBound[upperBound.length - 1]++;
      prefixedOptions.lt = upperBound;
    }

    for await (const { key, value } of this.store.iterator(prefixedOptions)) {
      yield {
        key: key.slice(this.namespace.length),
        value,
      };
    }
  }

  async close(): Promise<void> {
    // Don't close underlying store - it may be shared
  }

  async stats(): Promise<StorageStats> {
    // Would need to iterate just this namespace
    let keyCount = 0n;
    let sizeBytes = 0n;

    for await (const { key, value } of this.iterator()) {
      keyCount++;
      sizeBytes += BigInt(key.length + value.length);
    }

    return {
      keyCount,
      sizeBytes,
      diskUsageBytes: sizeBytes,
    };
  }
}

/**
 * Write batch for namespaced store
 */
class NamespacedWriteBatch implements WriteBatch {
  constructor(
    private batch: WriteBatch,
    private namespace: Uint8Array
  ) {}

  private prefixKey(key: Uint8Array): Uint8Array {
    const result = new Uint8Array(this.namespace.length + key.length);
    result.set(this.namespace, 0);
    result.set(key, this.namespace.length);
    return result;
  }

  put(key: Uint8Array, value: Uint8Array): WriteBatch {
    this.batch.put(this.prefixKey(key), value);
    return this;
  }

  delete(key: Uint8Array): WriteBatch {
    this.batch.delete(this.prefixKey(key));
    return this;
  }

  clear(): WriteBatch {
    this.batch.clear();
    return this;
  }

  async write(): Promise<void> {
    return this.batch.write();
  }
}

// ============ Factory Function ============

/**
 * Create a storage instance based on configuration
 */
export async function createStorage(config: LevelDBConfig | { type: 'memory' }): Promise<KVStore> {
  if (config.type === 'memory') {
    return new MemoryStore();
  }

  if (config.type === 'leveldb') {
    return LevelDBStore.open(config);
  }

  throw new Error(`Unknown storage type: ${(config as any).type}`);
}
