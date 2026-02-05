/**
 * State Pruning
 *
 * Manages removal of old state data to control disk usage.
 * Supports multiple strategies:
 * - Archive: Keep everything (full nodes)
 * - Retention: Keep last N blocks (default nodes)
 * - Snapshot: Only keep snapshot points (light nodes)
 *
 * PP HIERARCHY:
 *   TRUTH: Pruning only removes what can be reconstructed
 *   UNDERSTANDING: Clear strategies with documented tradeoffs
 *   BEAUTY: Elegant background pruning that doesn't block
 *   JUSTICE: Fair resource usage, respects protected blocks
 *   CREATIVITY: Flexible strategies for different use cases
 */

import type {
  KVStore,
  PruningConfig,
  PruningStrategy,
  StorageNamespace,
} from "./types.js";
import { STORAGE_NAMESPACES, DEFAULT_PRUNING_CONFIG } from "./types.js";
import type { SnapshotManager } from "./snapshots.js";

// ============ Pruning Engine ============

/**
 * Manages state pruning
 */
export class PruningEngine {
  private config: PruningConfig;
  private isRunning = false;
  private lastPrunedBlock = 0n;
  private listeners: Set<PruningListener> = new Set();

  constructor(
    private store: KVStore,
    private snapshotManager: SnapshotManager | null,
    config: Partial<PruningConfig> = {},
  ) {
    this.config = { ...DEFAULT_PRUNING_CONFIG, ...config };
  }

  /**
   * Run pruning for the current block
   */
  async prune(currentBlock: bigint): Promise<PruningResult> {
    if (this.isRunning) {
      return { prunedBlocks: 0, freedBytes: 0n, duration: 0 };
    }

    // Check if it's time to prune
    if (currentBlock - this.lastPrunedBlock < this.config.pruneIntervalBlocks) {
      return { prunedBlocks: 0, freedBytes: 0n, duration: 0 };
    }

    this.isRunning = true;
    const startTime = Date.now();
    let prunedBlocks = 0;
    let freedBytes = 0n;

    try {
      switch (this.config.strategy.type) {
        case "archive":
          // Archive mode: don't prune anything
          break;

        case "retention": {
          const retentionResult = await this.pruneWithRetention(
            currentBlock,
            this.config.strategy.retainBlocks,
            this.config.strategy.keepStateRoots,
          );
          prunedBlocks = retentionResult.prunedBlocks;
          freedBytes = retentionResult.freedBytes;
          break;
        }

        case "snapshot": {
          const snapshotResult = await this.pruneToSnapshots(
            currentBlock,
            this.config.strategy.snapshotInterval,
            this.config.strategy.retainSnapshots,
          );
          prunedBlocks = snapshotResult.prunedBlocks;
          freedBytes = snapshotResult.freedBytes;
          break;
        }
      }

      // Check disk usage and force prune if needed
      if (this.config.maxDiskUsage) {
        const stats = await this.store.stats();
        if (stats.diskUsageBytes > this.config.maxDiskUsage) {
          const emergencyResult = await this.emergencyPrune(
            currentBlock,
            this.config.maxDiskUsage,
          );
          prunedBlocks += emergencyResult.prunedBlocks;
          freedBytes += emergencyResult.freedBytes;
        }
      }

      this.lastPrunedBlock = currentBlock;

      const result: PruningResult = {
        prunedBlocks,
        freedBytes,
        duration: Date.now() - startTime,
      };

      this.emit({ type: "pruning_complete", result });

      return result;
    } finally {
      this.isRunning = false;
    }
  }

  /**
   * Prune with retention strategy
   */
  private async pruneWithRetention(
    currentBlock: bigint,
    retainBlocks: bigint,
    keepStateRoots: boolean,
  ): Promise<PruningResult> {
    const pruneBeforeBlock = currentBlock - retainBlocks;
    if (pruneBeforeBlock <= 0n) {
      return { prunedBlocks: 0, freedBytes: 0n, duration: 0 };
    }

    let freedBytes = 0n;
    let prunedBlocks = 0;

    // Prune block bodies (keep headers if keeping state roots)
    const bodyResult = await this.pruneBlockBodies(pruneBeforeBlock);
    freedBytes += bodyResult.freedBytes;
    prunedBlocks += bodyResult.prunedBlocks;

    // Prune receipts
    const receiptResult = await this.pruneBlockReceipts(pruneBeforeBlock);
    freedBytes += receiptResult.freedBytes;

    // Prune old state (but keep state roots for proofs if configured)
    if (!keepStateRoots) {
      const stateResult = await this.pruneOldState(pruneBeforeBlock);
      freedBytes += stateResult.freedBytes;
    }

    // Keep transaction index for a shorter period
    const txIndexPruneBlock = currentBlock - retainBlocks / 10n;
    if (txIndexPruneBlock > 0n) {
      const txResult = await this.pruneTransactionIndex(txIndexPruneBlock);
      freedBytes += txResult.freedBytes;
    }

    return { prunedBlocks, freedBytes, duration: 0 };
  }

  /**
   * Prune to snapshot points only
   */
  private async pruneToSnapshots(
    currentBlock: bigint,
    snapshotInterval: bigint,
    retainSnapshots: number,
  ): Promise<PruningResult> {
    if (!this.snapshotManager) {
      throw new Error("Snapshot pruning requires snapshot manager");
    }

    // Get available snapshots
    const snapshots = await this.snapshotManager.listSnapshots();

    // Keep only retainSnapshots most recent
    // (Set computed for future pruning filter use)
    void new Set(snapshots.slice(0, retainSnapshots).map((s) => s.blockNumber));

    // Determine what blocks to prune
    const oldestKeptSnapshot =
      snapshots[retainSnapshots - 1]?.blockNumber || 0n;
    const pruneBeforeBlock =
      oldestKeptSnapshot > 0n
        ? oldestKeptSnapshot
        : currentBlock - snapshotInterval;

    if (pruneBeforeBlock <= 0n) {
      return { prunedBlocks: 0, freedBytes: 0n, duration: 0 };
    }

    let freedBytes = 0n;

    // Prune everything before the oldest kept snapshot
    const bodyResult = await this.pruneBlockBodies(pruneBeforeBlock);
    freedBytes += bodyResult.freedBytes;

    const receiptResult = await this.pruneBlockReceipts(pruneBeforeBlock);
    freedBytes += receiptResult.freedBytes;

    const stateResult = await this.pruneOldState(pruneBeforeBlock);
    freedBytes += stateResult.freedBytes;

    return {
      prunedBlocks: bodyResult.prunedBlocks,
      freedBytes,
      duration: 0,
    };
  }

  /**
   * Emergency pruning when disk is full
   */
  private async emergencyPrune(
    currentBlock: bigint,
    targetUsage: bigint,
  ): Promise<PruningResult> {
    this.emit({ type: "emergency_prune_start", currentBlock, targetUsage });

    let freedBytes = 0n;
    let prunedBlocks = 0;
    let stats = await this.store.stats();

    // Progressively prune more aggressively
    let pruneDistance = 10000n; // Start with 10k blocks

    while (stats.diskUsageBytes > targetUsage && pruneDistance < currentBlock) {
      const pruneBeforeBlock = currentBlock - pruneDistance;

      // Skip protected blocks
      if (this.config.protectedBlocks?.some((b) => b >= pruneBeforeBlock)) {
        pruneDistance *= 2n;
        continue;
      }

      const bodyResult = await this.pruneBlockBodies(pruneBeforeBlock);
      freedBytes += bodyResult.freedBytes;
      prunedBlocks += bodyResult.prunedBlocks;

      const receiptResult = await this.pruneBlockReceipts(pruneBeforeBlock);
      freedBytes += receiptResult.freedBytes;

      // Get updated stats
      stats = await this.store.stats();
      pruneDistance *= 2n;
    }

    // If still over limit, prune state (aggressive)
    if (stats.diskUsageBytes > targetUsage) {
      const stateResult = await this.pruneOldState(currentBlock - 1000n);
      freedBytes += stateResult.freedBytes;
    }

    return { prunedBlocks, freedBytes, duration: 0 };
  }

  /**
   * Prune block bodies before a block number
   */
  private async pruneBlockBodies(
    beforeBlock: bigint,
  ): Promise<{ prunedBlocks: number; freedBytes: bigint }> {
    let prunedBlocks = 0;
    let freedBytes = 0n;

    const batch = this.store.batch();
    const namespace = STORAGE_NAMESPACES.BLOCK_BODIES;

    for await (const { key, value } of this.store.iterator({
      gte: namespace,
      lt: this.nextNamespace(namespace),
    })) {
      const blockNumber = this.extractBlockNumber(key, namespace);
      if (blockNumber < beforeBlock && !this.isProtected(blockNumber)) {
        batch.delete(key);
        freedBytes += BigInt(key.length + value.length);
        prunedBlocks++;
      }
    }

    await batch.write();
    return { prunedBlocks, freedBytes };
  }

  /**
   * Prune block receipts before a block number
   */
  private async pruneBlockReceipts(
    beforeBlock: bigint,
  ): Promise<{ freedBytes: bigint }> {
    let freedBytes = 0n;

    const batch = this.store.batch();
    const namespace = STORAGE_NAMESPACES.BLOCK_RECEIPTS;

    for await (const { key, value } of this.store.iterator({
      gte: namespace,
      lt: this.nextNamespace(namespace),
    })) {
      const blockNumber = this.extractBlockNumber(key, namespace);
      if (blockNumber < beforeBlock && !this.isProtected(blockNumber)) {
        batch.delete(key);
        freedBytes += BigInt(key.length + value.length);
      }
    }

    await batch.write();
    return { freedBytes };
  }

  /**
   * Prune old state data
   */
  private async pruneOldState(
    beforeBlock: bigint,
  ): Promise<{ freedBytes: bigint }> {
    let freedBytes = 0n;

    // Prune historical state from each state tree namespace
    const stateNamespaces = [
      STORAGE_NAMESPACES.ACCOUNTS,
      STORAGE_NAMESPACES.VALIDATORS,
      STORAGE_NAMESPACES.KNOWLEDGE,
      STORAGE_NAMESPACES.EVIDENCE,
      STORAGE_NAMESPACES.DISPUTES,
      STORAGE_NAMESPACES.VESTING,
    ];

    for (const namespace of stateNamespaces) {
      const result = await this.pruneNamespace(namespace, beforeBlock);
      freedBytes += result.freedBytes;
    }

    return { freedBytes };
  }

  /**
   * Prune transaction index
   */
  private async pruneTransactionIndex(
    beforeBlock: bigint,
  ): Promise<{ freedBytes: bigint }> {
    let freedBytes = 0n;

    const batch = this.store.batch();
    const namespace = STORAGE_NAMESPACES.TX_INDEX;

    for await (const { key, value } of this.store.iterator({
      gte: namespace,
      lt: this.nextNamespace(namespace),
    })) {
      // TX index entries contain block number in value
      try {
        const indexData = JSON.parse(new TextDecoder().decode(value));
        if (BigInt(indexData.blockNumber) < beforeBlock) {
          batch.delete(key);
          freedBytes += BigInt(key.length + value.length);
        }
      } catch {
        // Skip malformed entries
      }
    }

    await batch.write();
    return { freedBytes };
  }

  /**
   * Prune a specific namespace
   */
  private async pruneNamespace(
    namespace: StorageNamespace,
    beforeBlock: bigint,
  ): Promise<{ freedBytes: bigint }> {
    let freedBytes = 0n;
    const batch = this.store.batch();

    // Historical state keys typically include block number
    // Format: namespace + key + blockNumber (8 bytes)
    for await (const { key, value } of this.store.iterator({
      gte: namespace,
      lt: this.nextNamespace(namespace),
    })) {
      // Check if this is historical data (has block number suffix)
      if (key.length > namespace.length + 8) {
        const blockNumber = this.extractTrailingBlockNumber(key);
        if (blockNumber < beforeBlock && !this.isProtected(blockNumber)) {
          batch.delete(key);
          freedBytes += BigInt(key.length + value.length);
        }
      }
    }

    await batch.write();
    return { freedBytes };
  }

  // ============ Helper Methods ============

  private isProtected(blockNumber: bigint): boolean {
    return this.config.protectedBlocks?.includes(blockNumber) ?? false;
  }

  private extractBlockNumber(key: Uint8Array, namespace: Uint8Array): bigint {
    // Block number is typically the key after namespace
    const keyPart = key.slice(namespace.length);
    if (keyPart.length >= 8) {
      const view = new DataView(keyPart.buffer, keyPart.byteOffset);
      return view.getBigUint64(0, true);
    }
    return 0n;
  }

  private extractTrailingBlockNumber(key: Uint8Array): bigint {
    if (key.length >= 8) {
      const view = new DataView(key.buffer, key.byteOffset + key.length - 8);
      return view.getBigUint64(0, true);
    }
    return 0n;
  }

  private nextNamespace(namespace: Uint8Array): Uint8Array {
    const next = new Uint8Array(namespace.length);
    next.set(namespace);
    next[next.length - 1]++;
    return next;
  }

  // ============ Event Handling ============

  addListener(listener: PruningListener): void {
    this.listeners.add(listener);
  }

  removeListener(listener: PruningListener): void {
    this.listeners.delete(listener);
  }

  private emit(event: PruningEvent): void {
    for (const listener of this.listeners) {
      try {
        listener(event);
      } catch (e) {
        console.error("Pruning listener error:", e);
      }
    }
  }

  // ============ Status Methods ============

  /**
   * Get pruning status
   */
  getStatus(): PruningStatus {
    return {
      strategy: this.config.strategy,
      isRunning: this.isRunning,
      lastPrunedBlock: this.lastPrunedBlock,
      autoPruneEnabled: this.config.autoPrune,
      pruneIntervalBlocks: this.config.pruneIntervalBlocks,
    };
  }

  /**
   * Update pruning configuration
   */
  updateConfig(config: Partial<PruningConfig>): void {
    this.config = { ...this.config, ...config };
  }

  /**
   * Estimate space that would be freed by pruning
   */
  async estimatePruneSize(currentBlock: bigint): Promise<bigint> {
    switch (this.config.strategy.type) {
      case "archive":
        return 0n;

      case "retention":
        return this.estimateRetentionPrune(
          currentBlock,
          this.config.strategy.retainBlocks,
        );

      case "snapshot":
        return this.estimateSnapshotPrune(
          currentBlock,
          this.config.strategy.snapshotInterval,
        );
    }
  }

  private async estimateRetentionPrune(
    currentBlock: bigint,
    retainBlocks: bigint,
  ): Promise<bigint> {
    const pruneBeforeBlock = currentBlock - retainBlocks;
    if (pruneBeforeBlock <= 0n) return 0n;

    let estimatedSize = 0n;

    // Estimate from block bodies
    const namespace = STORAGE_NAMESPACES.BLOCK_BODIES;
    for await (const { key, value } of this.store.iterator({
      gte: namespace,
      lt: this.nextNamespace(namespace),
      limit: 1000, // Sample
    })) {
      const blockNumber = this.extractBlockNumber(key, namespace);
      if (blockNumber < pruneBeforeBlock) {
        estimatedSize += BigInt(key.length + value.length);
      }
    }

    // Extrapolate
    return estimatedSize * 10n; // Rough estimate
  }

  private async estimateSnapshotPrune(
    _currentBlock: bigint,
    _snapshotInterval: bigint,
  ): Promise<bigint> {
    // Would need to calculate based on snapshot points
    return 0n;
  }
}

// ============ Types ============

export interface PruningResult {
  prunedBlocks: number;
  freedBytes: bigint;
  duration: number;
}

export interface PruningStatus {
  strategy: PruningStrategy;
  isRunning: boolean;
  lastPrunedBlock: bigint;
  autoPruneEnabled: boolean;
  pruneIntervalBlocks: bigint;
}

export type PruningEvent =
  | { type: "pruning_complete"; result: PruningResult }
  | {
      type: "emergency_prune_start";
      currentBlock: bigint;
      targetUsage: bigint;
    };

export type PruningListener = (event: PruningEvent) => void;

// ============ Auto-Pruning Service ============

/**
 * Background service for automatic pruning
 */
export class AutoPruningService {
  private interval: ReturnType<typeof setInterval> | null = null;
  private engine: PruningEngine;
  private getCurrentBlock: () => Promise<bigint>;

  constructor(engine: PruningEngine, getCurrentBlock: () => Promise<bigint>) {
    this.engine = engine;
    this.getCurrentBlock = getCurrentBlock;
  }

  /**
   * Start automatic pruning
   */
  start(intervalMs: number = 60000): void {
    if (this.interval) return;

    this.interval = setInterval(async () => {
      try {
        const currentBlock = await this.getCurrentBlock();
        await this.engine.prune(currentBlock);
      } catch (e) {
        console.error("Auto-pruning error:", e);
      }
    }, intervalMs);
  }

  /**
   * Stop automatic pruning
   */
  stop(): void {
    if (this.interval) {
      clearInterval(this.interval);
      this.interval = null;
    }
  }

  /**
   * Trigger immediate prune
   */
  async pruneNow(): Promise<PruningResult> {
    const currentBlock = await this.getCurrentBlock();
    return this.engine.prune(currentBlock);
  }
}
