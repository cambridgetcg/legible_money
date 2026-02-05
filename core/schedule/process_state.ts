/**
 * Schedule Process State (SPS 131313)
 *
 * Manages the state of all active schedule processes.
 * From Patent: "SPS 131313 refers to the current state of all processes
 * corresponding to the Schedule Messages initiated by End Users 105
 * or converted from smart contracts"
 *
 * Provides:
 * - Process storage and retrieval
 * - Index management for efficient queries
 * - State root computation for Merkle tree
 */

import { createHash } from "node:crypto";
import type {
  ScheduleProcess,
  ScheduleProcessState,
  ScheduleProcessStatus,
  ScheduleType,
  ScheduleConfig,
  ScheduleEvent,
  // LinkedEntityType,
} from "./types.js";
import { DEFAULT_SCHEDULE_CONFIG, ScheduleError } from "./types.js";

/**
 * Schedule Process State Manager
 * Implements SPS 131313 from the patent
 */
export class ScheduleProcessStateManager {
  private state: ScheduleProcessState;
  private config: ScheduleConfig;
  private eventListeners: Array<(event: ScheduleEvent) => void> = [];

  // CRIT-002 FIX: Track current block timestamp for deterministic state updates
  private currentBlockTimestamp: number = 0;

  constructor(config: Partial<ScheduleConfig> = {}) {
    this.config = { ...DEFAULT_SCHEDULE_CONFIG, ...config };
    this.state = this.createEmptyState();
  }

  /**
   * Set the current block timestamp (called by SPU at block processing time)
   * All state modifications will use this timestamp for determinism
   */
  setBlockTimestamp(timestamp: number): void {
    this.currentBlockTimestamp = timestamp;
  }

  // ============ State Initialization ============

  private createEmptyState(): ScheduleProcessState {
    return {
      processes: new Map(),
      byTime: new Map(),
      byAccount: new Map(),
      byContract: new Map(),
      byType: new Map(),
      byStatus: new Map(),
      byClaimId: new Map(),
      byDisputeId: new Map(),
      byVestingId: new Map(),
      watchedKeys: new Map(),
      totalProcesses: 0,
      activeProcesses: 0,
      completedProcesses: 0,
      totalExecutions: 0,
      stateRoot: this.computeEmptyRoot(),
    };
  }

  private computeEmptyRoot(): string {
    return createHash("sha256").update("EMPTY_SPS").digest("hex");
  }

  // ============ Process Management ============

  /**
   * Add a new schedule process
   */
  addProcess(process: ScheduleProcess): void {
    // Validate
    this.validateProcess(process);

    // Check account limits
    const accountProcesses =
      this.state.byAccount.get(process.creator) ?? new Set();
    if (accountProcesses.size >= this.config.maxActiveSchedulesPerAccount) {
      throw new ScheduleError(
        `Account ${process.creator} has reached maximum active schedules`,
        "MAX_SCHEDULES_EXCEEDED",
        { limit: this.config.maxActiveSchedulesPerAccount },
      );
    }

    // Store process
    this.state.processes.set(process.id, process);

    // Update indexes
    this.addToIndexes(process);

    // Update metrics
    this.state.totalProcesses++;
    if (process.status === "active" || process.status === "pending") {
      this.state.activeProcesses++;
    }

    // Update state root
    this.updateStateRoot();

    // Emit event
    this.emit({ type: "process_created", process });
  }

  /**
   * Get a process by ID
   */
  getProcess(processId: string): ScheduleProcess | undefined {
    return this.state.processes.get(processId);
  }

  /**
   * Update a process
   */
  updateProcess(processId: string, updates: Partial<ScheduleProcess>): void {
    const process = this.state.processes.get(processId);
    if (!process) {
      throw new ScheduleError(
        `Process ${processId} not found`,
        "PROCESS_NOT_FOUND",
      );
    }

    const oldStatus = process.status;

    // Remove from old indexes if status changed
    if (updates.status && updates.status !== oldStatus) {
      this.removeFromStatusIndex(process);
    }

    // Apply updates
    // CRIT-002 FIX: Use deterministic block timestamp instead of Date.now()
    const updatedProcess: ScheduleProcess = {
      ...process,
      ...updates,
      modifiedAt: this.currentBlockTimestamp || process.modifiedAt,
    };

    this.state.processes.set(processId, updatedProcess);

    // Update status index
    if (updates.status && updates.status !== oldStatus) {
      this.addToStatusIndex(updatedProcess);
      this.updateActiveCount(oldStatus, updates.status);
    }

    // Update state root
    this.updateStateRoot();
  }

  /**
   * Remove a process (after completion/expiration)
   */
  removeProcess(processId: string): void {
    const process = this.state.processes.get(processId);
    if (!process) {
      return;
    }

    // Remove from all indexes
    this.removeFromIndexes(process);

    // Remove process
    this.state.processes.delete(processId);

    // Update metrics
    if (process.status === "completed") {
      this.state.completedProcesses++;
    }

    // Update state root
    this.updateStateRoot();
  }

  // ============ Index Management ============

  private addToIndexes(process: ScheduleProcess): void {
    // Account index
    this.addToSetIndex(this.state.byAccount, process.creator, process.id);

    // Contract index
    this.addToSetIndex(
      this.state.byContract,
      process.targetAddress,
      process.id,
    );

    // Type index
    const scheduleType = this.inferScheduleType(process);
    this.addToSetIndex(this.state.byType, scheduleType, process.id);

    // Status index
    this.addToStatusIndex(process);

    // Time index (for time-based conditions)
    if (process.nextCheckTime) {
      const bucket = this.getTimeBucket(process.nextCheckTime);
      this.addToSetIndex(this.state.byTime, bucket, process.id);
    }

    // Watched keys index (for logic conditions)
    for (const key of process.watchedStateKeys) {
      this.addToSetIndex(this.state.watchedKeys, key, process.id);
    }

    // PoT entity indexes
    if (process.linkedEntityId) {
      switch (process.linkedEntityType) {
        case "claim":
          this.addToSetIndex(
            this.state.byClaimId,
            process.linkedEntityId,
            process.id,
          );
          break;
        case "dispute":
          this.addToSetIndex(
            this.state.byDisputeId,
            process.linkedEntityId,
            process.id,
          );
          break;
        case "vesting":
          this.addToSetIndex(
            this.state.byVestingId,
            process.linkedEntityId,
            process.id,
          );
          break;
      }
    }
  }

  private removeFromIndexes(process: ScheduleProcess): void {
    // Account index
    this.removeFromSetIndex(this.state.byAccount, process.creator, process.id);

    // Contract index
    this.removeFromSetIndex(
      this.state.byContract,
      process.targetAddress,
      process.id,
    );

    // Type index
    const scheduleType = this.inferScheduleType(process);
    this.removeFromSetIndex(this.state.byType, scheduleType, process.id);

    // Status index
    this.removeFromStatusIndex(process);

    // Time index
    if (process.nextCheckTime) {
      const bucket = this.getTimeBucket(process.nextCheckTime);
      this.removeFromSetIndex(this.state.byTime, bucket, process.id);
    }

    // Watched keys index
    for (const key of process.watchedStateKeys) {
      this.removeFromSetIndex(this.state.watchedKeys, key, process.id);
    }

    // PoT entity indexes
    if (process.linkedEntityId) {
      switch (process.linkedEntityType) {
        case "claim":
          this.removeFromSetIndex(
            this.state.byClaimId,
            process.linkedEntityId,
            process.id,
          );
          break;
        case "dispute":
          this.removeFromSetIndex(
            this.state.byDisputeId,
            process.linkedEntityId,
            process.id,
          );
          break;
        case "vesting":
          this.removeFromSetIndex(
            this.state.byVestingId,
            process.linkedEntityId,
            process.id,
          );
          break;
      }
    }
  }

  private addToStatusIndex(process: ScheduleProcess): void {
    this.addToSetIndex(this.state.byStatus, process.status, process.id);
  }

  private removeFromStatusIndex(process: ScheduleProcess): void {
    this.removeFromSetIndex(this.state.byStatus, process.status, process.id);
  }

  private addToSetIndex<K>(
    map: Map<K, Set<string>>,
    key: K,
    value: string,
  ): void {
    let set = map.get(key);
    if (!set) {
      set = new Set();
      map.set(key, set);
    }
    set.add(value);
  }

  private removeFromSetIndex<K>(
    map: Map<K, Set<string>>,
    key: K,
    value: string,
  ): void {
    const set = map.get(key);
    if (set) {
      set.delete(value);
      if (set.size === 0) {
        map.delete(key);
      }
    }
  }

  private getTimeBucket(timestamp: number): number {
    return (
      Math.floor(timestamp / this.config.timeBucketSizeMs) *
      this.config.timeBucketSizeMs
    );
  }

  private inferScheduleType(process: ScheduleProcess): ScheduleType {
    const condition = process.condition;

    if (condition.compound) {
      return condition.compound.operator === "and"
        ? "time_and_logic"
        : "time_or_logic";
    }

    if (condition.time && condition.logic) {
      return "time_and_logic";
    }

    if (condition.time) {
      switch (condition.time.type) {
        case "absolute":
          return "time_absolute";
        case "relative":
          return "time_relative";
        case "periodic":
          return "time_periodic";
      }
    }

    if (condition.logic) {
      return condition.logic.type === "state_compare"
        ? "logic_condition"
        : "logic_state_change";
    }

    return "logic_condition";
  }

  private updateActiveCount(
    oldStatus: ScheduleProcessStatus,
    newStatus: ScheduleProcessStatus,
  ): void {
    const wasActive = oldStatus === "active" || oldStatus === "pending";
    const isActive = newStatus === "active" || newStatus === "pending";

    if (wasActive && !isActive) {
      this.state.activeProcesses--;
    } else if (!wasActive && isActive) {
      this.state.activeProcesses++;
    }
  }

  // ============ Query Methods ============

  /**
   * Get processes due for execution at or before given time
   */
  getProcessesDueByTime(currentTime: number): ScheduleProcess[] {
    const result: ScheduleProcess[] = [];
    const currentBucket = this.getTimeBucket(currentTime);

    // Check all buckets up to and including current
    for (const [bucket, processIds] of this.state.byTime) {
      if (bucket <= currentBucket) {
        for (const processId of processIds) {
          const process = this.state.processes.get(processId);
          if (
            process &&
            process.nextCheckTime &&
            process.nextCheckTime <= currentTime
          ) {
            if (process.status === "active" || process.status === "pending") {
              result.push(process);
            }
          }
        }
      }
    }

    return result;
  }

  /**
   * Get processes watching a specific state key
   */
  getProcessesWatchingKey(stateKey: string): ScheduleProcess[] {
    const processIds = this.state.watchedKeys.get(stateKey);
    if (!processIds) {
      return [];
    }

    return Array.from(processIds)
      .map((id) => this.state.processes.get(id))
      .filter(
        (p): p is ScheduleProcess =>
          p !== undefined && (p.status === "active" || p.status === "pending"),
      );
  }

  /**
   * Get processes by account
   */
  getProcessesByAccount(did: string): ScheduleProcess[] {
    const processIds = this.state.byAccount.get(did);
    if (!processIds) {
      return [];
    }

    return Array.from(processIds)
      .map((id) => this.state.processes.get(id))
      .filter((p): p is ScheduleProcess => p !== undefined);
  }

  /**
   * Get processes by status
   */
  getProcessesByStatus(status: ScheduleProcessStatus): ScheduleProcess[] {
    const processIds = this.state.byStatus.get(status);
    if (!processIds) {
      return [];
    }

    return Array.from(processIds)
      .map((id) => this.state.processes.get(id))
      .filter((p): p is ScheduleProcess => p !== undefined);
  }

  /**
   * Get processes by linked claim
   */
  getProcessesByClaimId(claimId: string): ScheduleProcess[] {
    const processIds = this.state.byClaimId.get(claimId);
    if (!processIds) {
      return [];
    }

    return Array.from(processIds)
      .map((id) => this.state.processes.get(id))
      .filter((p): p is ScheduleProcess => p !== undefined);
  }

  /**
   * Get processes by linked dispute
   */
  getProcessesByDisputeId(disputeId: string): ScheduleProcess[] {
    const processIds = this.state.byDisputeId.get(disputeId);
    if (!processIds) {
      return [];
    }

    return Array.from(processIds)
      .map((id) => this.state.processes.get(id))
      .filter((p): p is ScheduleProcess => p !== undefined);
  }

  /**
   * Get all active processes
   */
  getActiveProcesses(): ScheduleProcess[] {
    return [
      ...this.getProcessesByStatus("active"),
      ...this.getProcessesByStatus("pending"),
    ];
  }

  // ============ State Root ============

  private updateStateRoot(): void {
    // Compute Merkle root of all processes
    const processHashes = Array.from(this.state.processes.values())
      .sort((a, b) => a.id.localeCompare(b.id))
      .map((p) => this.hashProcess(p));

    if (processHashes.length === 0) {
      this.state.stateRoot = this.computeEmptyRoot();
      return;
    }

    // Simple hash combination for now (would use proper Merkle tree in production)
    const combinedHash = createHash("sha256");
    for (const hash of processHashes) {
      combinedHash.update(hash);
    }
    this.state.stateRoot = combinedHash.digest("hex");
  }

  private hashProcess(process: ScheduleProcess): string {
    const data = JSON.stringify({
      id: process.id,
      sourceMessageId: process.sourceMessageId,
      creator: process.creator,
      status: process.status,
      executionCount: process.executionCount,
      generatedTxNonce: process.generatedTxNonce.toString(),
      remainingFee: process.remainingFee.toString(),
    });
    return createHash("sha256").update(data).digest("hex");
  }

  /**
   * Get current state root
   */
  getStateRoot(): string {
    return this.state.stateRoot;
  }

  // ============ Metrics ============

  /**
   * Get state metrics
   */
  getMetrics(): {
    totalProcesses: number;
    activeProcesses: number;
    completedProcesses: number;
    totalExecutions: number;
    processCountByStatus: Record<string, number>;
    processCountByType: Record<string, number>;
  } {
    const byStatus: Record<string, number> = {};
    for (const [status, ids] of this.state.byStatus) {
      byStatus[status] = ids.size;
    }

    const byType: Record<string, number> = {};
    for (const [type, ids] of this.state.byType) {
      byType[type] = ids.size;
    }

    return {
      totalProcesses: this.state.totalProcesses,
      activeProcesses: this.state.activeProcesses,
      completedProcesses: this.state.completedProcesses,
      totalExecutions: this.state.totalExecutions,
      processCountByStatus: byStatus,
      processCountByType: byType,
    };
  }

  /**
   * Increment total executions count
   */
  incrementExecutions(): void {
    this.state.totalExecutions++;
  }

  // ============ Validation ============

  private validateProcess(process: ScheduleProcess): void {
    if (!process.id) {
      throw new ScheduleError("Process ID is required", "INVALID_CONDITION");
    }

    if (!process.creator) {
      throw new ScheduleError("Creator DID is required", "INVALID_CONDITION");
    }

    if (!process.condition) {
      throw new ScheduleError(
        "Schedule condition is required",
        "INVALID_CONDITION",
      );
    }

    if (process.remainingFee < 0n) {
      throw new ScheduleError(
        "Remaining fee cannot be negative",
        "INSUFFICIENT_FEE",
      );
    }

    // Check compound condition depth
    if (process.condition.compound) {
      const depth = this.getConditionDepth(process.condition);
      if (depth > this.config.maxCompoundConditionDepth) {
        throw new ScheduleError(
          `Compound condition depth ${depth} exceeds maximum ${this.config.maxCompoundConditionDepth}`,
          "CONDITION_TOO_COMPLEX",
        );
      }
    }

    // Check watched keys limit
    if (
      process.watchedStateKeys.length > this.config.maxWatchedKeysPerProcess
    ) {
      throw new ScheduleError(
        `Watched keys count ${process.watchedStateKeys.length} exceeds maximum ${this.config.maxWatchedKeysPerProcess}`,
        "CONDITION_TOO_COMPLEX",
      );
    }
  }

  private getConditionDepth(condition: {
    compound?: { conditions: unknown[] };
  }): number {
    if (!condition.compound) {
      return 1;
    }
    const childDepths = condition.compound.conditions.map((c) =>
      this.getConditionDepth(c as { compound?: { conditions: unknown[] } }),
    );
    return 1 + Math.max(...childDepths);
  }

  // ============ Events ============

  /**
   * Subscribe to schedule events
   */
  onEvent(listener: (event: ScheduleEvent) => void): () => void {
    this.eventListeners.push(listener);
    return () => {
      const index = this.eventListeners.indexOf(listener);
      if (index >= 0) {
        this.eventListeners.splice(index, 1);
      }
    };
  }

  private emit(event: ScheduleEvent): void {
    for (const listener of this.eventListeners) {
      try {
        listener(event);
      } catch {
        // Ignore listener errors
      }
    }
  }

  // ============ Serialization ============

  /**
   * Export state for persistence
   */
  exportState(): {
    processes: Array<[string, ScheduleProcess]>;
    metrics: {
      totalProcesses: number;
      activeProcesses: number;
      completedProcesses: number;
      totalExecutions: number;
    };
  } {
    return {
      processes: Array.from(this.state.processes.entries()),
      metrics: {
        totalProcesses: this.state.totalProcesses,
        activeProcesses: this.state.activeProcesses,
        completedProcesses: this.state.completedProcesses,
        totalExecutions: this.state.totalExecutions,
      },
    };
  }

  /**
   * Import state from persistence
   */
  importState(data: {
    processes: Array<[string, ScheduleProcess]>;
    metrics: {
      totalProcesses: number;
      activeProcesses: number;
      completedProcesses: number;
      totalExecutions: number;
    };
  }): void {
    // Reset state
    this.state = this.createEmptyState();

    // Import metrics
    this.state.totalProcesses = data.metrics.totalProcesses;
    this.state.activeProcesses = data.metrics.activeProcesses;
    this.state.completedProcesses = data.metrics.completedProcesses;
    this.state.totalExecutions = data.metrics.totalExecutions;

    // Import processes (this rebuilds indexes)
    for (const [id, process] of data.processes) {
      this.state.processes.set(id, process);
      this.addToIndexes(process);
    }

    // Update state root
    this.updateStateRoot();
  }
}
