/**
 * State Snapshots
 *
 * Creates and manages snapshots of blockchain state for:
 * - Fast sync (new nodes can download snapshot instead of replaying)
 * - Recovery (restore from snapshot after corruption)
 * - Archival (historical state access)
 *
 * PP HIERARCHY:
 *   TRUTH: Snapshots capture exact state at a point in time
 *   UNDERSTANDING: Clear chunking and verification
 *   BEAUTY: Elegant streaming interface
 *   JUSTICE: Fair chunk distribution for parallel download
 *   CREATIVITY: Incremental snapshots for efficiency
 */

import { sha512_256Hex } from "../../core/crypto/hashing.js";
import type {
  KVStore,
  StateSnapshot,
  SnapshotChunk,
  SnapshotConfig,
  StorageNamespace,
} from "./types.js";
import { STORAGE_NAMESPACES, DEFAULT_SNAPSHOT_CONFIG } from "./types.js";

// ============ Snapshot Manager ============

/**
 * Manages state snapshots
 */
export class SnapshotManager {
  private config: SnapshotConfig;
  private snapshots: Map<string, StateSnapshot> = new Map();

  constructor(
    private store: KVStore,
    config: Partial<SnapshotConfig> = {},
  ) {
    this.config = { ...DEFAULT_SNAPSHOT_CONFIG, ...config };
  }

  /**
   * Create a new snapshot at the current state
   */
  async createSnapshot(
    blockNumber: bigint,
    blockHash: string,
    stateRoot: string,
    namespaces: StorageNamespace[] = Object.values(STORAGE_NAMESPACES),
  ): Promise<StateSnapshot> {
    const snapshotId = this.generateSnapshotId(blockNumber, blockHash);
    const chunks: SnapshotChunk[] = [];
    const chunkHashes: string[] = [];

    let totalUncompressedSize = 0n;
    let totalCompressedSize = 0n;

    // Collect data from each namespace
    const allData: Array<{ key: Uint8Array; value: Uint8Array }> = [];

    for (const namespace of namespaces) {
      for await (const { key, value } of this.store.iterator({
        gte: namespace,
        lt: this.nextNamespace(namespace),
      })) {
        allData.push({ key, value });
        totalUncompressedSize += BigInt(key.length + value.length + 8); // +8 for length prefixes
      }
    }

    // Create chunks
    let currentChunk: Array<{ key: Uint8Array; value: Uint8Array }> = [];
    let currentSize = 0;
    let chunkIndex = 0;

    for (const entry of allData) {
      const entrySize = entry.key.length + entry.value.length + 8;

      if (
        currentSize + entrySize > this.config.maxChunkSize &&
        currentChunk.length > 0
      ) {
        // Finalize current chunk
        const chunk = await this.createChunk(
          snapshotId,
          chunkIndex,
          currentChunk,
        );
        chunks.push(chunk);
        chunkHashes.push(chunk.hash);
        totalCompressedSize += BigInt(chunk.data.length);

        currentChunk = [];
        currentSize = 0;
        chunkIndex++;
      }

      currentChunk.push(entry);
      currentSize += entrySize;
    }

    // Finalize last chunk
    if (currentChunk.length > 0) {
      const chunk = await this.createChunk(
        snapshotId,
        chunkIndex,
        currentChunk,
      );
      chunks.push(chunk);
      chunkHashes.push(chunk.hash);
      totalCompressedSize += BigInt(chunk.data.length);
    }

    // Store chunks
    for (const chunk of chunks) {
      await this.storeChunk(chunk);
    }

    // Create snapshot metadata
    const snapshot: StateSnapshot = {
      id: snapshotId,
      blockNumber,
      blockHash,
      stateRoot,
      createdAt: Date.now(),
      formatVersion: 1,
      compressedSize: totalCompressedSize,
      uncompressedSize: totalUncompressedSize,
      chunkCount: chunks.length,
      chunkHashes,
      snapshotType: "full",
    };

    // Store metadata
    await this.storeSnapshotMetadata(snapshot);
    this.snapshots.set(snapshotId, snapshot);

    // Prune old snapshots
    await this.pruneOldSnapshots();

    return snapshot;
  }

  /**
   * Create an incremental snapshot (only changes since parent)
   */
  async createIncrementalSnapshot(
    blockNumber: bigint,
    blockHash: string,
    stateRoot: string,
    parentSnapshotId: string,
    changedKeys: Uint8Array[],
  ): Promise<StateSnapshot> {
    const parent = await this.getSnapshot(parentSnapshotId);
    if (!parent) {
      throw new Error(`Parent snapshot not found: ${parentSnapshotId}`);
    }

    const snapshotId = this.generateSnapshotId(blockNumber, blockHash);
    const chunks: SnapshotChunk[] = [];
    const chunkHashes: string[] = [];

    let totalUncompressedSize = 0n;
    let totalCompressedSize = 0n;

    // Only include changed data
    const changedData: Array<{ key: Uint8Array; value: Uint8Array | null }> =
      [];

    for (const key of changedKeys) {
      const value = await this.store.get(key);
      changedData.push({ key, value });
      totalUncompressedSize += BigInt(key.length + (value?.length || 0) + 9); // +9 for length + null marker
    }

    // Create single chunk for incremental (usually small)
    if (changedData.length > 0) {
      const chunk = await this.createIncrementalChunk(
        snapshotId,
        0,
        changedData,
      );
      chunks.push(chunk);
      chunkHashes.push(chunk.hash);
      totalCompressedSize += BigInt(chunk.data.length);
      await this.storeChunk(chunk);
    }

    const snapshot: StateSnapshot = {
      id: snapshotId,
      blockNumber,
      blockHash,
      stateRoot,
      createdAt: Date.now(),
      formatVersion: 1,
      compressedSize: totalCompressedSize,
      uncompressedSize: totalUncompressedSize,
      chunkCount: chunks.length,
      chunkHashes,
      snapshotType: "incremental",
      parentSnapshotId,
    };

    await this.storeSnapshotMetadata(snapshot);
    this.snapshots.set(snapshotId, snapshot);

    return snapshot;
  }

  /**
   * Restore state from a snapshot
   */
  async restoreSnapshot(snapshotId: string): Promise<void> {
    const snapshot = await this.getSnapshot(snapshotId);
    if (!snapshot) {
      throw new Error(`Snapshot not found: ${snapshotId}`);
    }

    // For incremental snapshots, restore parent first
    if (snapshot.snapshotType === "incremental" && snapshot.parentSnapshotId) {
      await this.restoreSnapshot(snapshot.parentSnapshotId);
    }

    // Restore chunks
    for (let i = 0; i < snapshot.chunkCount; i++) {
      const chunk = await this.loadChunk(snapshotId, i);
      if (!chunk) {
        throw new Error(`Chunk ${i} not found for snapshot ${snapshotId}`);
      }

      // Verify chunk hash
      const computedHash = sha512_256Hex(chunk.data);
      if (computedHash !== chunk.hash) {
        throw new Error(
          `Chunk ${i} hash mismatch: expected ${chunk.hash}, got ${computedHash}`,
        );
      }

      // Apply chunk to store
      if (snapshot.snapshotType === "full") {
        await this.applyFullChunk(chunk);
      } else {
        await this.applyIncrementalChunk(chunk);
      }
    }
  }

  /**
   * Get snapshot metadata
   */
  async getSnapshot(snapshotId: string): Promise<StateSnapshot | null> {
    // Check cache first
    if (this.snapshots.has(snapshotId)) {
      return this.snapshots.get(snapshotId)!;
    }

    // Load from store
    const key = this.snapshotMetadataKey(snapshotId);
    const data = await this.store.get(key);
    if (!data) {
      return null;
    }

    const snapshot = JSON.parse(new TextDecoder().decode(data), this.reviver);
    this.snapshots.set(snapshotId, snapshot);
    return snapshot;
  }

  /**
   * List all available snapshots
   */
  async listSnapshots(): Promise<StateSnapshot[]> {
    const snapshots: StateSnapshot[] = [];
    const prefix = new TextEncoder().encode("snapshot:meta:");

    for await (const { value } of this.store.iterator({
      gte: prefix,
      lt: this.incrementKey(prefix),
    })) {
      const snapshot = JSON.parse(
        new TextDecoder().decode(value),
        this.reviver,
      );
      snapshots.push(snapshot);
    }

    return snapshots.sort((a, b) => Number(b.blockNumber - a.blockNumber));
  }

  /**
   * Delete a snapshot
   */
  async deleteSnapshot(snapshotId: string): Promise<void> {
    const snapshot = await this.getSnapshot(snapshotId);
    if (!snapshot) {
      return;
    }

    // Delete chunks
    for (let i = 0; i < snapshot.chunkCount; i++) {
      await this.store.delete(this.chunkKey(snapshotId, i));
    }

    // Delete metadata
    await this.store.delete(this.snapshotMetadataKey(snapshotId));
    this.snapshots.delete(snapshotId);
  }

  /**
   * Stream snapshot chunks for download
   */
  async *streamChunks(snapshotId: string): AsyncGenerator<SnapshotChunk> {
    const snapshot = await this.getSnapshot(snapshotId);
    if (!snapshot) {
      throw new Error(`Snapshot not found: ${snapshotId}`);
    }

    for (let i = 0; i < snapshot.chunkCount; i++) {
      const chunk = await this.loadChunk(snapshotId, i);
      if (chunk) {
        yield chunk;
      }
    }
  }

  /**
   * Import a chunk (for receiving snapshots)
   */
  async importChunk(chunk: SnapshotChunk): Promise<void> {
    // Verify hash
    const computedHash = sha512_256Hex(chunk.data);
    if (computedHash !== chunk.hash) {
      throw new Error(
        `Chunk hash mismatch: expected ${chunk.hash}, got ${computedHash}`,
      );
    }

    await this.storeChunk(chunk);
  }

  // ============ Private Methods ============

  private generateSnapshotId(blockNumber: bigint, blockHash: string): string {
    return sha512_256Hex(
      `snapshot:${blockNumber}:${blockHash}:${Date.now()}`,
    ).slice(0, 32);
  }

  private async createChunk(
    snapshotId: string,
    chunkIndex: number,
    data: Array<{ key: Uint8Array; value: Uint8Array }>,
  ): Promise<SnapshotChunk> {
    // Serialize data
    const serialized = this.serializeChunkData(data);

    // Compress (simplified - in production use zstd/lz4)
    const compressed = await this.compress(serialized);

    // Calculate hash
    const hash = sha512_256Hex(compressed);

    return {
      snapshotId,
      chunkIndex,
      data: compressed,
      hash,
    };
  }

  private async createIncrementalChunk(
    snapshotId: string,
    chunkIndex: number,
    data: Array<{ key: Uint8Array; value: Uint8Array | null }>,
  ): Promise<SnapshotChunk> {
    const serialized = this.serializeIncrementalData(data);
    const compressed = await this.compress(serialized);
    const hash = sha512_256Hex(compressed);

    return {
      snapshotId,
      chunkIndex,
      data: compressed,
      hash,
    };
  }

  private serializeChunkData(
    data: Array<{ key: Uint8Array; value: Uint8Array }>,
  ): Uint8Array {
    // Calculate total size
    let totalSize = 4; // entry count
    for (const entry of data) {
      totalSize += 4 + entry.key.length + 4 + entry.value.length;
    }

    const result = new Uint8Array(totalSize);
    const view = new DataView(result.buffer);
    let offset = 0;

    // Write entry count
    view.setUint32(offset, data.length, true);
    offset += 4;

    // Write entries
    for (const entry of data) {
      // Key length + key
      view.setUint32(offset, entry.key.length, true);
      offset += 4;
      result.set(entry.key, offset);
      offset += entry.key.length;

      // Value length + value
      view.setUint32(offset, entry.value.length, true);
      offset += 4;
      result.set(entry.value, offset);
      offset += entry.value.length;
    }

    return result;
  }

  private serializeIncrementalData(
    data: Array<{ key: Uint8Array; value: Uint8Array | null }>,
  ): Uint8Array {
    let totalSize = 4;
    for (const entry of data) {
      totalSize +=
        4 +
        entry.key.length +
        1 +
        (entry.value?.length || 0) +
        (entry.value ? 4 : 0);
    }

    const result = new Uint8Array(totalSize);
    const view = new DataView(result.buffer);
    let offset = 0;

    view.setUint32(offset, data.length, true);
    offset += 4;

    for (const entry of data) {
      view.setUint32(offset, entry.key.length, true);
      offset += 4;
      result.set(entry.key, offset);
      offset += entry.key.length;

      // Null marker + optional value
      if (entry.value === null) {
        result[offset] = 0; // null marker
        offset += 1;
      } else {
        result[offset] = 1; // has value
        offset += 1;
        view.setUint32(offset, entry.value.length, true);
        offset += 4;
        result.set(entry.value, offset);
        offset += entry.value.length;
      }
    }

    return result;
  }

  private async compress(data: Uint8Array): Promise<Uint8Array> {
    // Simplified - in production use zstd or lz4
    if (this.config.compression === "none") {
      return data;
    }
    // Return as-is for now (compression would be added with a library)
    return data;
  }

  private async decompress(data: Uint8Array): Promise<Uint8Array> {
    if (this.config.compression === "none") {
      return data;
    }
    return data;
  }

  private async applyFullChunk(chunk: SnapshotChunk): Promise<void> {
    const decompressed = await this.decompress(chunk.data);
    const view = new DataView(decompressed.buffer, decompressed.byteOffset);
    let offset = 0;

    const entryCount = view.getUint32(offset, true);
    offset += 4;

    const batch = this.store.batch();

    for (let i = 0; i < entryCount; i++) {
      const keyLen = view.getUint32(offset, true);
      offset += 4;
      const key = decompressed.slice(offset, offset + keyLen);
      offset += keyLen;

      const valueLen = view.getUint32(offset, true);
      offset += 4;
      const value = decompressed.slice(offset, offset + valueLen);
      offset += valueLen;

      batch.put(key, value);
    }

    await batch.write();
  }

  private async applyIncrementalChunk(chunk: SnapshotChunk): Promise<void> {
    const decompressed = await this.decompress(chunk.data);
    const view = new DataView(decompressed.buffer, decompressed.byteOffset);
    let offset = 0;

    const entryCount = view.getUint32(offset, true);
    offset += 4;

    const batch = this.store.batch();

    for (let i = 0; i < entryCount; i++) {
      const keyLen = view.getUint32(offset, true);
      offset += 4;
      const key = decompressed.slice(offset, offset + keyLen);
      offset += keyLen;

      const hasValue = decompressed[offset] === 1;
      offset += 1;

      if (hasValue) {
        const valueLen = view.getUint32(offset, true);
        offset += 4;
        const value = decompressed.slice(offset, offset + valueLen);
        offset += valueLen;
        batch.put(key, value);
      } else {
        batch.delete(key);
      }
    }

    await batch.write();
  }

  private async storeChunk(chunk: SnapshotChunk): Promise<void> {
    const key = this.chunkKey(chunk.snapshotId, chunk.chunkIndex);
    await this.store.put(key, chunk.data);
  }

  private async loadChunk(
    snapshotId: string,
    chunkIndex: number,
  ): Promise<SnapshotChunk | null> {
    const key = this.chunkKey(snapshotId, chunkIndex);
    const data = await this.store.get(key);
    if (!data) {
      return null;
    }

    // Get hash from metadata
    const snapshot = await this.getSnapshot(snapshotId);
    if (!snapshot) {
      return null;
    }

    return {
      snapshotId,
      chunkIndex,
      data,
      hash: snapshot.chunkHashes[chunkIndex],
    };
  }

  private async storeSnapshotMetadata(snapshot: StateSnapshot): Promise<void> {
    const key = this.snapshotMetadataKey(snapshot.id);
    const data = JSON.stringify(snapshot, this.replacer);
    await this.store.put(key, new TextEncoder().encode(data));
  }

  private async pruneOldSnapshots(): Promise<void> {
    const snapshots = await this.listSnapshots();

    // Keep only maxSnapshots
    if (snapshots.length > this.config.maxSnapshots) {
      const toDelete = snapshots.slice(this.config.maxSnapshots);
      for (const snapshot of toDelete) {
        await this.deleteSnapshot(snapshot.id);
      }
    }
  }

  private snapshotMetadataKey(snapshotId: string): Uint8Array {
    return new TextEncoder().encode(`snapshot:meta:${snapshotId}`);
  }

  private chunkKey(snapshotId: string, chunkIndex: number): Uint8Array {
    return new TextEncoder().encode(
      `snapshot:chunk:${snapshotId}:${chunkIndex.toString().padStart(6, "0")}`,
    );
  }

  private nextNamespace(namespace: Uint8Array): Uint8Array {
    const next = new Uint8Array(namespace.length);
    next.set(namespace);
    next[next.length - 1]++;
    return next;
  }

  private incrementKey(key: Uint8Array): Uint8Array {
    const result = new Uint8Array(key.length);
    result.set(key);
    for (let i = result.length - 1; i >= 0; i--) {
      if (result[i] < 255) {
        result[i]++;
        break;
      }
      result[i] = 0;
    }
    return result;
  }

  private replacer(_key: string, value: unknown): unknown {
    if (typeof value === "bigint") {
      return { __bigint__: value.toString() };
    }
    return value;
  }

  private reviver(_key: string, value: unknown): unknown {
    if (typeof value === "object" && value !== null && "__bigint__" in value) {
      return BigInt((value as { __bigint__: string }).__bigint__);
    }
    return value;
  }
}

// ============ Snapshot Sync ============

/**
 * Handles snapshot synchronization between nodes
 */
export class SnapshotSync {
  constructor(
    private manager: SnapshotManager,
    _store: KVStore,
  ) {}

  /**
   * Request snapshot list from peer
   */
  async requestSnapshotList(): Promise<StateSnapshot[]> {
    // Would be implemented via network layer
    return this.manager.listSnapshots();
  }

  /**
   * Download and apply a snapshot from peer
   */
  async downloadSnapshot(
    snapshotId: string,
    fetchChunk: (
      snapshotId: string,
      chunkIndex: number,
    ) => Promise<SnapshotChunk>,
  ): Promise<void> {
    // This would be called with a network fetch function
    const snapshot = await this.manager.getSnapshot(snapshotId);
    if (!snapshot) {
      throw new Error(`Snapshot not found: ${snapshotId}`);
    }

    // Download and import each chunk
    for (let i = 0; i < snapshot.chunkCount; i++) {
      const chunk = await fetchChunk(snapshotId, i);
      await this.manager.importChunk(chunk);
    }

    // Restore the snapshot
    await this.manager.restoreSnapshot(snapshotId);
  }

  /**
   * Export snapshot for peer
   */
  async *exportSnapshot(snapshotId: string): AsyncGenerator<SnapshotChunk> {
    yield* this.manager.streamChunks(snapshotId);
  }
}
