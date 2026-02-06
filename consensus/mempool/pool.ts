/**
 * Mempool
 *
 * Transaction pool for pending transactions.
 * Transactions are sorted by fee and validated before inclusion.
 */

import { EventEmitter } from "events";
import { verify } from "../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";
import type { Transaction, TransactionType } from "../pot/types.js";

/** JSON replacer that converts BigInt to string (for size estimation) */
const bigintReplacer = (_k: string, v: unknown) =>
  typeof v === "bigint" ? v.toString() : v;

// ============ Mempool Configuration ============

export interface MempoolConfig {
  maxSize: number; // Maximum transactions
  maxPerSender: number; // Max transactions per sender
  minFee: bigint; // Minimum fee
  maxTxSize: number; // Maximum transaction size in bytes
  evictionInterval: number; // How often to evict old txs (ms)
  maxTxAge: number; // Maximum transaction age (ms)
}

export const DEFAULT_MEMPOOL_CONFIG: MempoolConfig = {
  maxSize: 10000,
  maxPerSender: 100,
  minFee: 1000n, // 0.000001 LGM
  maxTxSize: 100000, // 100 KB
  evictionInterval: 60000, // 1 minute
  maxTxAge: 3600000, // 1 hour
};

// ============ Mempool Entry ============

export interface MempoolEntry {
  tx: Transaction;
  txHash: string;
  addedAt: number;
  fee: bigint;
  feePerByte: bigint;
  size: number;
}

// ============ Mempool ============

export class Mempool extends EventEmitter {
  private config: MempoolConfig;
  private pool: Map<string, MempoolEntry> = new Map();
  private bySender: Map<string, Set<string>> = new Map(); // sender -> txHashes
  private evictionTimer: NodeJS.Timeout | null = null;

  constructor(config: Partial<MempoolConfig> = {}) {
    super();
    this.config = { ...DEFAULT_MEMPOOL_CONFIG, ...config };
  }

  /**
   * Start the mempool (begins eviction timer)
   */
  start(): void {
    this.evictionTimer = setInterval(
      () => this.evictExpired(),
      this.config.evictionInterval,
    );
  }

  /**
   * Stop the mempool
   */
  stop(): void {
    if (this.evictionTimer) {
      clearInterval(this.evictionTimer);
      this.evictionTimer = null;
    }
  }

  /**
   * Add a transaction to the pool
   */
  add(tx: Transaction): { added: boolean; reason?: string; txHash?: string } {
    // Calculate hash
    const txHash = this.hashTransaction(tx);

    // Check if already exists
    if (this.pool.has(txHash)) {
      return { added: false, reason: "Transaction already in pool" };
    }

    // Validate transaction
    const validation = this.validateTransaction(tx);
    if (!validation.valid) {
      return { added: false, reason: validation.reason };
    }

    // Check pool size
    if (this.pool.size >= this.config.maxSize) {
      // Try to evict lower-fee transaction
      const evicted = this.evictLowestFee(tx.fee);
      if (!evicted) {
        return { added: false, reason: "Pool full, fee too low" };
      }
    }

    // Check per-sender limit
    const senderTxs = this.bySender.get(tx.sender) || new Set();
    if (senderTxs.size >= this.config.maxPerSender) {
      return { added: false, reason: "Too many transactions from sender" };
    }

    // Calculate size in bytes (not characters) for accurate fee-per-byte
    const serialized = JSON.stringify(tx, bigintReplacer);
    const size = new TextEncoder().encode(serialized).byteLength;
    const feePerByte = tx.fee / BigInt(size);

    // Add to pool
    const entry: MempoolEntry = {
      tx,
      txHash,
      addedAt: Date.now(),
      fee: tx.fee,
      feePerByte,
      size,
    };

    this.pool.set(txHash, entry);

    // Track by sender
    if (!this.bySender.has(tx.sender)) {
      this.bySender.set(tx.sender, new Set());
    }
    this.bySender.get(tx.sender)!.add(txHash);

    this.emit("tx:added", entry);

    return { added: true, txHash };
  }

  /**
   * Remove a transaction from the pool
   */
  remove(txHash: string): boolean {
    const entry = this.pool.get(txHash);
    if (!entry) return false;

    this.pool.delete(txHash);

    // Remove from sender tracking
    const senderTxs = this.bySender.get(entry.tx.sender);
    if (senderTxs) {
      senderTxs.delete(txHash);
      if (senderTxs.size === 0) {
        this.bySender.delete(entry.tx.sender);
      }
    }

    this.emit("tx:removed", entry);

    return true;
  }

  /**
   * Get transaction by hash
   */
  get(txHash: string): Transaction | undefined {
    return this.pool.get(txHash)?.tx;
  }

  /**
   * Check if transaction exists
   */
  has(txHash: string): boolean {
    return this.pool.has(txHash);
  }

  /**
   * Get all transactions for a sender
   */
  getBySender(sender: string): Transaction[] {
    const txHashes = this.bySender.get(sender);
    if (!txHashes) return [];

    return Array.from(txHashes)
      .map((hash) => this.pool.get(hash)?.tx)
      .filter((tx): tx is Transaction => tx !== undefined);
  }

  /**
   * Get transactions for block inclusion.
   * Enforces nonce continuity per sender to prevent block rejection.
   *
   * @param senderBaseNonces - Expected next nonce per sender from state.
   *   When provided, only nonce-contiguous sequences starting from the
   *   expected nonce are selected. Without it, intra-pool nonce ordering
   *   is still enforced but gaps relative to on-chain state are not caught.
   */
  getForBlock(
    maxCount: number,
    maxSize: number,
    senderBaseNonces?: Map<string, bigint>,
  ): Transaction[] {
    // Group entries by sender, sorted by nonce within each group
    const bySenderEntries = new Map<string, MempoolEntry[]>();
    for (const entry of this.pool.values()) {
      const sender = entry.tx.sender;
      if (!bySenderEntries.has(sender)) {
        bySenderEntries.set(sender, []);
      }
      bySenderEntries.get(sender)!.push(entry);
    }

    // For each sender, build nonce-contiguous sequence
    const senderQueues: Array<{
      sender: string;
      entries: MempoolEntry[];
      leadFeePerByte: bigint;
    }> = [];

    for (const [sender, entries] of bySenderEntries) {
      entries.sort((a, b) => {
        if (a.tx.nonce < b.tx.nonce) return -1;
        if (a.tx.nonce > b.tx.nonce) return 1;
        return 0;
      });

      const baseNonce = senderBaseNonces?.get(sender);
      const contiguous: MempoolEntry[] = [];
      let expectedNonce = baseNonce;

      for (const entry of entries) {
        if (expectedNonce !== undefined && entry.tx.nonce !== expectedNonce) {
          break; // Nonce gap — stop here for this sender
        }
        contiguous.push(entry);
        expectedNonce =
          expectedNonce !== undefined
            ? expectedNonce + 1n
            : entry.tx.nonce + 1n;
      }

      if (contiguous.length > 0) {
        senderQueues.push({
          sender,
          entries: contiguous,
          leadFeePerByte: contiguous[0].feePerByte,
        });
      }
    }

    // Sort sender queues by lead tx feePerByte (highest first)
    senderQueues.sort((a, b) => {
      if (b.leadFeePerByte > a.leadFeePerByte) return 1;
      if (b.leadFeePerByte < a.leadFeePerByte) return -1;
      return 0;
    });

    // Round-robin pick from sender queues, respecting nonce order
    const selected: Transaction[] = [];
    let totalSize = 0;
    const queueIndices = new Map<string, number>();

    let progress = true;
    while (selected.length < maxCount && progress) {
      progress = false;
      for (const queue of senderQueues) {
        if (selected.length >= maxCount) break;
        const idx = queueIndices.get(queue.sender) ?? 0;
        if (idx >= queue.entries.length) continue;

        const entry = queue.entries[idx];
        if (totalSize + entry.size > maxSize) continue;

        selected.push(entry.tx);
        totalSize += entry.size;
        queueIndices.set(queue.sender, idx + 1);
        progress = true;
      }
    }

    return selected;
  }

  /**
   * Get pending transactions by type
   */
  getByType(type: TransactionType): Transaction[] {
    return Array.from(this.pool.values())
      .filter((entry) => entry.tx.type === type)
      .map((entry) => entry.tx);
  }

  /**
   * Get mempool size
   */
  size(): number {
    return this.pool.size;
  }

  /**
   * Get mempool stats
   */
  getStats(): MempoolStats {
    const entries = Array.from(this.pool.values());
    const fees = entries.map((e) => e.fee);

    return {
      size: this.pool.size,
      senders: this.bySender.size,
      totalFees: fees.reduce((a, b) => a + b, 0n),
      minFee: fees.length > 0 ? fees.reduce((a, b) => (a < b ? a : b)) : 0n,
      maxFee: fees.length > 0 ? fees.reduce((a, b) => (a > b ? a : b)) : 0n,
      avgFee:
        fees.length > 0
          ? fees.reduce((a, b) => a + b, 0n) / BigInt(fees.length)
          : 0n,
    };
  }

  /**
   * Clear the entire pool
   */
  clear(): void {
    this.pool.clear();
    this.bySender.clear();
    this.emit("cleared");
  }

  // ============ Private Methods ============

  private validateTransaction(tx: Transaction): {
    valid: boolean;
    reason?: string;
  } {
    // Check size in bytes (not characters) for consistent enforcement
    const size = new TextEncoder().encode(
      JSON.stringify(tx, bigintReplacer),
    ).byteLength;
    if (size > this.config.maxTxSize) {
      return { valid: false, reason: "Transaction too large" };
    }

    // Check minimum fee
    if (tx.fee < this.config.minFee) {
      return { valid: false, reason: "Fee too low" };
    }

    // Verify signature against payload (matches wallet SDK signing convention).
    // TODO: Migrate to getSignableMessage() covering type+nonce+fee+payload.
    const payload = JSON.stringify(tx.payload);
    const valid = verify(
      new TextEncoder().encode(payload),
      tx.signature,
      tx.sender,
    );

    if (!valid) {
      return { valid: false, reason: "Invalid signature" };
    }

    return { valid: true };
  }

  private hashTransaction(tx: Transaction): string {
    return sha512_256Hex(
      JSON.stringify({
        type: tx.type,
        sender: tx.sender,
        nonce: tx.nonce.toString(),
        payload: tx.payload,
        signature: tx.signature,
      }),
    );
  }

  private evictLowestFee(newFee: bigint): boolean {
    // Find transaction with lowest feePerByte (matches getForBlock sort order)
    let lowest: MempoolEntry | null = null;

    for (const entry of this.pool.values()) {
      if (!lowest || entry.feePerByte < lowest.feePerByte) {
        lowest = entry;
      }
    }

    // Only evict if new fee is higher than the lowest entry's total fee
    if (lowest && newFee > lowest.fee) {
      this.remove(lowest.txHash);
      return true;
    }

    return false;
  }

  private evictExpired(): void {
    const now = Date.now();
    const maxAge = this.config.maxTxAge;
    const toRemove: string[] = [];

    for (const [hash, entry] of this.pool) {
      if (now - entry.addedAt > maxAge) {
        toRemove.push(hash);
      }
    }

    for (const hash of toRemove) {
      this.remove(hash);
    }

    if (toRemove.length > 0) {
      this.emit("evicted", toRemove.length);
    }
  }
}

export interface MempoolStats {
  size: number;
  senders: number;
  totalFees: bigint;
  minFee: bigint;
  maxFee: bigint;
  avgFee: bigint;
}

// ============ Specialized Pools ============

/**
 * Create a mempool for claim submissions
 */
export function createClaimPool(config: Partial<MempoolConfig> = {}): Mempool {
  return new Mempool({
    ...config,
    maxSize: 1000, // Fewer claims than general txs
    maxPerSender: 10, // Limit claims per agent
  });
}

/**
 * Create a mempool for verification transactions
 */
export function createVerificationPool(
  config: Partial<MempoolConfig> = {},
): Mempool {
  return new Mempool({
    ...config,
    maxSize: 5000,
    minFee: 0n, // Verifications may be fee-free
  });
}
