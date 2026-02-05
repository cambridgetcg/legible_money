/**
 * Write-Ahead Log (WAL)
 *
 * Implements Phase 3 of the persistence backend (GAP-001).
 * Provides transaction durability by logging operations before applying them.
 *
 * WAL Entry Lifecycle:
 * 1. beginTransaction() - Create uncommitted entry
 * 2. addOperation() - Add operations to entry
 * 3. prepare() - Write entry to disk (uncommitted)
 * 4. commit() - Mark entry as committed
 * 5. cleanup() - Remove old committed entries
 *
 * Recovery:
 * - On startup, scan for uncommitted entries
 * - Replay operations from uncommitted entries
 * - Or rollback if entry was never prepared
 *
 * WAL Key Format:
 * - `wal:entry:{entryId}` - WAL entry data
 * - `wal:latest` - Latest entry ID
 * - `wal:checkpoint` - Last checkpointed entry
 *
 * @module storage/write_ahead_log
 */

import type { KVStore } from "./types.js";
import { namespacedKey } from "./types.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";

// ============ Types ============

/**
 * A single operation in the WAL
 */
export interface WALOperation {
  /** Namespace for the operation */
  namespace: number;
  /** Key being modified */
  key: string;
  /** Operation type */
  type: "put" | "delete";
  /** Value for put operations (base64 encoded) */
  value?: string;
}

/**
 * WAL entry status
 */
export type WALEntryStatus =
  | "pending"
  | "prepared"
  | "committed"
  | "rolled_back";

/**
 * A WAL entry representing a transaction
 */
export interface WALEntry {
  /** Unique entry ID */
  id: string;
  /** Block number this entry is for */
  blockNumber: bigint;
  /** Entry status */
  status: WALEntryStatus;
  /** Operations in this entry */
  operations: WALOperation[];
  /** Timestamp when entry was created */
  createdAt: number;
  /** Timestamp when entry was prepared (written to disk) */
  preparedAt?: number;
  /** Timestamp when entry was committed */
  committedAt?: number;
  /** Checksum of operations for integrity verification */
  checksum: string;
}

/**
 * Serialized WAL entry for storage
 */
interface SerializedWALEntry {
  id: string;
  blockNumber: string;
  status: WALEntryStatus;
  operations: WALOperation[];
  createdAt: number;
  preparedAt?: number;
  committedAt?: number;
  checksum: string;
}

// ============ WAL Namespace ============

const WAL_NAMESPACE = new Uint8Array([0xfe]); // WAL uses its own namespace

// ============ Serialization ============

function serializeEntry(entry: WALEntry): Uint8Array {
  const serialized: SerializedWALEntry = {
    ...entry,
    blockNumber: entry.blockNumber.toString(),
  };
  return new TextEncoder().encode(JSON.stringify(serialized));
}

function deserializeEntry(data: Uint8Array): WALEntry {
  const serialized: SerializedWALEntry = JSON.parse(
    new TextDecoder().decode(data),
  );
  return {
    ...serialized,
    blockNumber: BigInt(serialized.blockNumber),
  };
}

function stringToKey(str: string): Uint8Array {
  return new TextEncoder().encode(str);
}

// ============ Checksum ============

function computeChecksum(operations: WALOperation[]): string {
  const data = JSON.stringify(operations);
  return sha512_256Hex(data);
}

// ============ Write-Ahead Log ============

/**
 * Write-Ahead Log for transaction durability
 */
export class WriteAheadLog {
  private store: KVStore;
  private currentEntry: WALEntry | null = null;
  private entryCounter: bigint = 0n;

  constructor(store: KVStore) {
    this.store = store;
  }

  /**
   * Initialize the WAL - load latest entry counter
   */
  async initialize(): Promise<void> {
    const latestKey = namespacedKey(WAL_NAMESPACE, stringToKey("latest"));
    const data = await this.store.get(latestKey);
    if (data) {
      const json = JSON.parse(new TextDecoder().decode(data));
      this.entryCounter = BigInt(json.counter);
    }
  }

  /**
   * Begin a new transaction
   */
  beginTransaction(blockNumber: bigint): string {
    if (this.currentEntry !== null) {
      throw new Error("Transaction already in progress");
    }

    this.entryCounter++;
    const entryId = `wal-${this.entryCounter.toString().padStart(16, "0")}`;

    this.currentEntry = {
      id: entryId,
      blockNumber,
      status: "pending",
      operations: [],
      createdAt: Date.now(),
      checksum: "",
    };

    return entryId;
  }

  /**
   * Add an operation to the current transaction
   */
  addOperation(operation: WALOperation): void {
    if (!this.currentEntry) {
      throw new Error("No transaction in progress");
    }
    this.currentEntry.operations.push(operation);
  }

  /**
   * Add a put operation
   */
  addPut(namespace: number, key: string, value: Uint8Array): void {
    this.addOperation({
      namespace,
      key,
      type: "put",
      value: Buffer.from(value).toString("base64"),
    });
  }

  /**
   * Add a delete operation
   */
  addDelete(namespace: number, key: string): void {
    this.addOperation({
      namespace,
      key,
      type: "delete",
    });
  }

  /**
   * Prepare the transaction (write to disk, but don't commit)
   */
  async prepare(): Promise<void> {
    if (!this.currentEntry) {
      throw new Error("No transaction in progress");
    }

    // Compute checksum
    this.currentEntry.checksum = computeChecksum(this.currentEntry.operations);
    this.currentEntry.status = "prepared";
    this.currentEntry.preparedAt = Date.now();

    // Write entry to disk
    const entryKey = namespacedKey(
      WAL_NAMESPACE,
      stringToKey(`entry:${this.currentEntry.id}`),
    );
    await this.store.put(entryKey, serializeEntry(this.currentEntry));

    // Update latest counter
    const latestKey = namespacedKey(WAL_NAMESPACE, stringToKey("latest"));
    await this.store.put(
      latestKey,
      new TextEncoder().encode(
        JSON.stringify({ counter: this.entryCounter.toString() }),
      ),
    );
  }

  /**
   * Commit the transaction (mark as committed)
   */
  async commit(): Promise<void> {
    if (!this.currentEntry) {
      throw new Error("No transaction in progress");
    }

    if (this.currentEntry.status !== "prepared") {
      throw new Error("Transaction must be prepared before commit");
    }

    this.currentEntry.status = "committed";
    this.currentEntry.committedAt = Date.now();

    // Update entry on disk
    const entryKey = namespacedKey(
      WAL_NAMESPACE,
      stringToKey(`entry:${this.currentEntry.id}`),
    );
    await this.store.put(entryKey, serializeEntry(this.currentEntry));

    // Clear current entry
    this.currentEntry = null;
  }

  /**
   * Rollback the current transaction
   */
  async rollback(): Promise<void> {
    if (!this.currentEntry) {
      return; // Nothing to rollback
    }

    if (this.currentEntry.status === "prepared") {
      // Mark as rolled back on disk
      this.currentEntry.status = "rolled_back";
      const entryKey = namespacedKey(
        WAL_NAMESPACE,
        stringToKey(`entry:${this.currentEntry.id}`),
      );
      await this.store.put(entryKey, serializeEntry(this.currentEntry));
    }

    this.currentEntry = null;
  }

  /**
   * Get the current transaction entry (if any)
   */
  getCurrentEntry(): WALEntry | null {
    return this.currentEntry;
  }

  /**
   * Get a specific entry by ID
   */
  async getEntry(entryId: string): Promise<WALEntry | null> {
    const entryKey = namespacedKey(
      WAL_NAMESPACE,
      stringToKey(`entry:${entryId}`),
    );
    const data = await this.store.get(entryKey);
    if (!data) return null;
    return deserializeEntry(data);
  }

  /**
   * Find all uncommitted entries for recovery
   */
  async findUncommittedEntries(): Promise<WALEntry[]> {
    const uncommitted: WALEntry[] = [];
    const prefix = namespacedKey(WAL_NAMESPACE, stringToKey("entry:"));
    const prefixEnd = namespacedKey(WAL_NAMESPACE, stringToKey("entry:\xff"));

    for await (const { value } of this.store.iterator({
      gte: prefix,
      lt: prefixEnd,
    })) {
      const entry = deserializeEntry(value);
      if (entry.status === "prepared") {
        uncommitted.push(entry);
      }
    }

    // Sort by block number for replay order
    uncommitted.sort((a, b) => {
      if (a.blockNumber < b.blockNumber) return -1;
      if (a.blockNumber > b.blockNumber) return 1;
      return 0;
    });

    return uncommitted;
  }

  /**
   * Mark an entry as committed (for recovery)
   */
  async markCommitted(entryId: string): Promise<void> {
    const entry = await this.getEntry(entryId);
    if (!entry) {
      throw new Error(`Entry ${entryId} not found`);
    }

    entry.status = "committed";
    entry.committedAt = Date.now();

    const entryKey = namespacedKey(
      WAL_NAMESPACE,
      stringToKey(`entry:${entryId}`),
    );
    await this.store.put(entryKey, serializeEntry(entry));
  }

  /**
   * Cleanup old committed entries
   */
  async cleanup(keepCount: number = 100): Promise<number> {
    const entries: WALEntry[] = [];
    const prefix = namespacedKey(WAL_NAMESPACE, stringToKey("entry:"));
    const prefixEnd = namespacedKey(WAL_NAMESPACE, stringToKey("entry:\xff"));

    // Collect all entries
    for await (const { key: _key, value } of this.store.iterator({
      gte: prefix,
      lt: prefixEnd,
    })) {
      const entry = deserializeEntry(value);
      entries.push(entry);
    }

    // Sort by creation time (oldest first)
    entries.sort((a, b) => a.createdAt - b.createdAt);

    // Find committed entries to delete
    const committed = entries.filter((e) => e.status === "committed");
    const toDelete = committed.slice(
      0,
      Math.max(0, committed.length - keepCount),
    );

    // Delete old entries
    const batch = this.store.batch();
    for (const entry of toDelete) {
      const entryKey = namespacedKey(
        WAL_NAMESPACE,
        stringToKey(`entry:${entry.id}`),
      );
      batch.delete(entryKey);
    }
    await batch.write();

    return toDelete.length;
  }

  /**
   * Get WAL statistics
   */
  async getStatistics(): Promise<WALStatistics> {
    let pending = 0;
    let prepared = 0;
    let committed = 0;
    let rolledBack = 0;

    const prefix = namespacedKey(WAL_NAMESPACE, stringToKey("entry:"));
    const prefixEnd = namespacedKey(WAL_NAMESPACE, stringToKey("entry:\xff"));

    for await (const { value } of this.store.iterator({
      gte: prefix,
      lt: prefixEnd,
    })) {
      const entry = deserializeEntry(value);
      switch (entry.status) {
        case "pending":
          pending++;
          break;
        case "prepared":
          prepared++;
          break;
        case "committed":
          committed++;
          break;
        case "rolled_back":
          rolledBack++;
          break;
      }
    }

    return {
      entryCounter: this.entryCounter,
      pending,
      prepared,
      committed,
      rolledBack,
      hasActiveTransaction: this.currentEntry !== null,
    };
  }

  /**
   * Verify integrity of an entry
   */
  verifyEntry(entry: WALEntry): boolean {
    const expectedChecksum = computeChecksum(entry.operations);
    return entry.checksum === expectedChecksum;
  }
}

// ============ Types ============

export interface WALStatistics {
  entryCounter: bigint;
  pending: number;
  prepared: number;
  committed: number;
  rolledBack: number;
  hasActiveTransaction: boolean;
}

// ============ WAL Integration Helper ============

/**
 * Helper for integrating WAL with state operations
 */
export class WALTransaction {
  private wal: WriteAheadLog;
  private entryId: string;
  private completed: boolean = false;

  constructor(wal: WriteAheadLog, blockNumber: bigint) {
    this.wal = wal;
    this.entryId = wal.beginTransaction(blockNumber);
  }

  /**
   * Get the entry ID
   */
  getId(): string {
    return this.entryId;
  }

  /**
   * Add a put operation
   */
  put(namespace: number, key: string, value: Uint8Array): void {
    this.ensureActive();
    this.wal.addPut(namespace, key, value);
  }

  /**
   * Add a delete operation
   */
  delete(namespace: number, key: string): void {
    this.ensureActive();
    this.wal.addDelete(namespace, key);
  }

  /**
   * Prepare and commit the transaction
   */
  async commit(): Promise<void> {
    this.ensureActive();
    await this.wal.prepare();
    await this.wal.commit();
    this.completed = true;
  }

  /**
   * Rollback the transaction
   */
  async rollback(): Promise<void> {
    if (!this.completed) {
      await this.wal.rollback();
      this.completed = true;
    }
  }

  private ensureActive(): void {
    if (this.completed) {
      throw new Error("Transaction already completed");
    }
  }
}
