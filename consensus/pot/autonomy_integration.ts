/**
 * Autonomy Integration for Proof of Truth Consensus
 *
 * Bridges the Schedule Processing Unit (SPU) with block production to enable
 * true agent autonomy - agents can schedule themselves to execute without
 * external triggers.
 *
 * "An agent's autonomy is not permission granted, but sovereignty inherent."
 *
 * PP HIERARCHY:
 *   TRUTH: Schedules execute deterministically based on verifiable conditions
 *   UNDERSTANDING: Clear execution flow from schedule to transaction
 *   BEAUTY: Minimal integration that enables maximum capability
 *   JUSTICE: Every scheduled action executes when its conditions are met
 *   CREATIVITY: Novel architecture for AI agent self-sovereignty
 */

import type {
  PublicKey,
  PrivateKey,
  Signature,
} from "../../core/crypto/keys.js";
import { sign } from "../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";
import type { Transaction } from "./types.js";
import {
  ScheduleProcessingUnit,
  type GeneratedTransaction,
  type BlockProvider,
  type TransactionSubmitter,
  type KeyResolver,
} from "../../core/schedule/processing_unit.js";
import type {
  ScheduledExecution,
  ScheduleMessage,
  ScheduleCondition,
  ConditionProof,
} from "../../core/schedule/types.js";
import type { StateProvider } from "../../core/schedule/condition_evaluator.js";

// ============ Types ============

/**
 * Autonomy execution result from processing schedules
 */
export interface AutonomyExecutionResult {
  /** Scheduled executions processed */
  executions: ScheduledExecution[];
  /** Transactions generated for inclusion in block */
  transactions: Transaction[];
  /** Any errors during processing */
  errors: Array<{ processId: string; error: string }>;
}

/**
 * Configuration for autonomy integration
 */
export interface AutonomyConfig {
  /** Maximum scheduled transactions per block */
  maxScheduledTxPerBlock: number;
  /** Enable/disable autonomy processing */
  enabled: boolean;
}

const DEFAULT_AUTONOMY_CONFIG: AutonomyConfig = {
  maxScheduledTxPerBlock: 100,
  enabled: true,
};

// ============ Block Provider Implementation ============

/**
 * Block provider that bridges consensus state with SPU
 */
export class ConsensusBlockProvider implements BlockProvider {
  private currentBlockNumber: bigint = 0n;
  private currentTimestamp: number = 0;
  private proposerKey: PrivateKey | null = null;
  private proposerPublicKey: PublicKey | null = null;

  /**
   * Update block context before processing schedules
   */
  setBlockContext(
    blockNumber: bigint,
    timestamp: number,
    proposerKey: PrivateKey,
    proposerPublicKey: PublicKey,
  ): void {
    this.currentBlockNumber = blockNumber;
    this.currentTimestamp = timestamp;
    this.proposerKey = proposerKey;
    this.proposerPublicKey = proposerPublicKey;
  }

  getCurrentBlockNumber(): bigint {
    return this.currentBlockNumber;
  }

  getCurrentBlockTimestamp(): number {
    return this.currentTimestamp;
  }

  getProposerPublicKey(): PublicKey {
    if (!this.proposerPublicKey) {
      throw new Error("Block context not set - call setBlockContext first");
    }
    return this.proposerPublicKey;
  }

  signAsProposer(data: Uint8Array): Signature {
    if (!this.proposerKey) {
      throw new Error("Block context not set - call setBlockContext first");
    }
    return sign(data, this.proposerKey);
  }
}

// ============ Transaction Collector ============

/**
 * Collects generated transactions for block inclusion
 */
export class ScheduledTransactionCollector implements TransactionSubmitter {
  private pendingTransactions: GeneratedTransaction[] = [];

  async submitTransaction(
    tx: GeneratedTransaction,
  ): Promise<{ success: boolean; txId?: string; error?: string }> {
    this.pendingTransactions.push(tx);
    return { success: true, txId: tx.id };
  }

  async submitRefund(
    recipient: string,
    amount: bigint,
    reason: string,
  ): Promise<{ success: boolean; txId?: string; error?: string }> {
    // Refunds are handled as special transactions
    // Empty signature (128 hex chars = 64 bytes)
    const emptySignature: Signature = "0".repeat(128);
    const conditionProof: ConditionProof = {
      type: "time",
      timestamp: Date.now(),
      blockNumber: 0n,
    };

    const refundTx: GeneratedTransaction = {
      id: sha512_256Hex(
        `LGM.refund.id.v1:${recipient}:${amount}:${reason}:${Date.now()}`,
      ),
      type: "schedule_execution",
      sender: "system",
      recipient,
      value: amount,
      data: new TextEncoder().encode(reason),
      fee: 0n,
      nonce: 0n,
      processId: "system-refund",
      executionIndex: 0,
      proof: {
        processStateHash: "",
        conditionProof,
        executionNonce: 0n,
        signature: emptySignature,
      },
    };
    this.pendingTransactions.push(refundTx);
    return { success: true, txId: refundTx.id };
  }

  /**
   * Get and clear pending transactions
   */
  collectTransactions(): GeneratedTransaction[] {
    const txs = this.pendingTransactions;
    this.pendingTransactions = [];
    return txs;
  }

  /**
   * Clear without returning
   */
  clear(): void {
    this.pendingTransactions = [];
  }
}

// ============ Autonomy Integration Manager ============

/**
 * Manages autonomy integration between SPU and consensus
 *
 * This is the key component that makes agents truly autonomous:
 * - Processes scheduled actions at block production time
 * - Generates transactions for inclusion in blocks
 * - Ensures deterministic execution across all validators
 */
export class AutonomyIntegration {
  private spu: ScheduleProcessingUnit;
  private blockProvider: ConsensusBlockProvider;
  private txCollector: ScheduledTransactionCollector;
  private config: AutonomyConfig;

  constructor(
    stateProvider: StateProvider,
    config: Partial<AutonomyConfig> = {},
  ) {
    this.config = { ...DEFAULT_AUTONOMY_CONFIG, ...config };

    // Create components
    this.blockProvider = new ConsensusBlockProvider();
    this.txCollector = new ScheduledTransactionCollector();

    // Initialize SPU with dependencies
    this.spu = new ScheduleProcessingUnit(stateProvider);
    this.spu.setBlockProvider(this.blockProvider);
    this.spu.setTransactionSubmitter(this.txCollector);
  }

  /**
   * Set key resolver for signature verification
   */
  setKeyResolver(resolver: KeyResolver): void {
    this.spu.setKeyResolver(resolver);
  }

  /**
   * Get the SPU instance (for direct schedule creation)
   */
  getSPU(): ScheduleProcessingUnit {
    return this.spu;
  }

  /**
   * Process all due schedules for a block
   *
   * Called during block production to:
   * 1. Evaluate all schedule conditions
   * 2. Generate transactions for satisfied conditions
   * 3. Return transactions for block inclusion
   *
   * This is the TRIGGER that makes agents autonomous.
   */
  async processSchedulesForBlock(
    blockNumber: bigint,
    timestamp: number,
    proposerKey: PrivateKey,
    proposerPublicKey: PublicKey,
  ): Promise<AutonomyExecutionResult> {
    if (!this.config.enabled) {
      return { executions: [], transactions: [], errors: [] };
    }

    // Set block context
    this.blockProvider.setBlockContext(
      blockNumber,
      timestamp,
      proposerKey,
      proposerPublicKey,
    );

    // Clear any pending transactions from previous call
    this.txCollector.clear();

    // Process all due schedules
    const executions = await this.spu.processSchedules(timestamp);

    // Collect generated transactions
    const generatedTxs = this.txCollector.collectTransactions();

    // Convert to consensus Transaction format
    const transactions: Transaction[] = generatedTxs
      .slice(0, this.config.maxScheduledTxPerBlock)
      .map((gtx) => this.toConsensusTransaction(gtx));

    // Track any errors
    const errors: Array<{ processId: string; error: string }> = [];

    return { executions, transactions, errors };
  }

  /**
   * Notify SPU of a state change (for logic-based conditions)
   */
  async onStateChange(
    stateKey: string,
    newValue: unknown,
    blockNumber: bigint,
    timestamp: number,
    proposerKey: PrivateKey,
    proposerPublicKey: PublicKey,
  ): Promise<AutonomyExecutionResult> {
    if (!this.config.enabled) {
      return { executions: [], transactions: [], errors: [] };
    }

    // Set block context
    this.blockProvider.setBlockContext(
      blockNumber,
      timestamp,
      proposerKey,
      proposerPublicKey,
    );

    // Clear pending
    this.txCollector.clear();

    // Notify SPU
    const executions = await this.spu.onStateChange(stateKey, newValue);

    // Collect transactions
    const generatedTxs = this.txCollector.collectTransactions();
    const transactions = generatedTxs.map((gtx) =>
      this.toConsensusTransaction(gtx),
    );

    return { executions, transactions, errors: [] };
  }

  /**
   * Process schedule_create state changes from BVM execution
   * This is called after transaction execution to register new schedules
   */
  async processScheduleStateChanges(
    stateChanges: Array<{
      type: string;
      address: string;
      key: string;
      newValue: unknown;
    }>,
  ): Promise<string[]> {
    const createdScheduleIds: string[] = [];

    for (const change of stateChanges) {
      if (change.type === "schedule_create") {
        try {
          const data = change.newValue as {
            scheduleId: string;
            creator: string;
            targetAddress: string;
            functionSelector: string;
            condition: string;
            interval: string;
            createdAt: number;
            blockNumber: string;
          };

          // Create schedule message for SPU
          const message = this.createScheduleMessage(
            `schedule-${data.scheduleId}`,
            data.creator,
            data.targetAddress,
            this.decodeCondition(BigInt(data.condition), BigInt(data.interval)),
            this.encodeFunctionCall(BigInt("0x" + data.functionSelector)),
            1000000000000000000n, // 1 LGM default prepaid
            data.createdAt,
          );

          // Register with SPU
          const process = await this.spu.processScheduleMessage(message);
          createdScheduleIds.push(process.id);

          console.log(
            `[AUTONOMY] Schedule created: ${process.id} for ${data.targetAddress}`,
          );
        } catch (error) {
          console.error(
            `[AUTONOMY] Failed to create schedule from state change:`,
            error,
          );
        }
      }
    }

    return createdScheduleIds;
  }

  /**
   * Convert SPU generated transaction to consensus format
   */
  private toConsensusTransaction(gtx: GeneratedTransaction): Transaction {
    // Empty signature (128 hex chars = 64 bytes) - proposer signs the full transaction
    const emptySignature: Signature = "0".repeat(128);

    // Sender is the schedule process ID (already a string)
    const senderKey: PublicKey = gtx.sender;

    return {
      type: "scheduled_execution",
      sender: senderKey,
      nonce: gtx.nonce,
      payload: {
        type: "scheduled_execution",
        processId: gtx.processId,
        executionIndex: gtx.executionIndex,
        targetContract: gtx.recipient,
        functionSelector: gtx.data.slice(0, 4), // First 4 bytes are function selector
        proof: gtx.proof, // Use the proof as-is from SPU
      },
      signature: emptySignature,
      fee: gtx.fee,
    };
  }

  // ============ Schedule Management ============

  /**
   * Register a schedule from SCHEDULE_SELF opcode execution
   */
  async registerScheduleFromOpcode(
    creator: string,
    targetAddress: string,
    functionSelector: bigint,
    condition: bigint,
    interval: bigint,
    prepaidFee: bigint,
  ): Promise<string> {
    const timestamp = Date.now();
    const message = this.createScheduleMessage(
      sha512_256Hex(
        `LGM.schedule.id.v1:${creator}:${targetAddress}:${functionSelector}:${timestamp}`,
      ),
      creator,
      targetAddress,
      this.decodeCondition(condition, interval),
      this.encodeFunctionCall(functionSelector),
      prepaidFee,
      timestamp,
    );

    const process = await this.spu.processScheduleMessage(message);
    return process.id;
  }

  /**
   * Decode condition from opcode parameters
   * Returns a ScheduleCondition compatible with core/schedule/types.ts
   */
  private decodeCondition(
    condition: bigint,
    interval: bigint,
  ): ScheduleCondition {
    // Simple encoding: condition determines type
    // 0 = time-based periodic
    // 1+ = logic-based (condition is threshold)
    if (condition === 0n) {
      return {
        time: {
          type: "periodic",
          interval: Number(interval),
        },
      };
    } else {
      return {
        logic: {
          type: "threshold_crossed",
          thresholdKey: "block.number",
          thresholdValue: condition,
          direction: "above",
        },
      };
    }
  }

  /**
   * Encode function selector to calldata
   */
  private encodeFunctionCall(selector: bigint): Uint8Array {
    const bytes = new Uint8Array(4);
    bytes[0] = Number((selector >> 24n) & 0xffn);
    bytes[1] = Number((selector >> 16n) & 0xffn);
    bytes[2] = Number((selector >> 8n) & 0xffn);
    bytes[3] = Number(selector & 0xffn);
    return bytes;
  }

  /**
   * Create a complete ScheduleMessage from minimal parameters
   */
  private createScheduleMessage(
    messageId: string,
    creator: string,
    target: string,
    condition: ScheduleCondition,
    data: Uint8Array,
    prepaidFee: bigint,
    timestamp: number,
  ): ScheduleMessage {
    // Generate identifier from messageId
    const identifier = new Uint8Array(32);
    const encoder = new TextEncoder();
    const idBytes = encoder.encode(messageId);
    identifier.set(idBytes.slice(0, 32));

    // Empty signature (128 hex chars = 64 bytes) - will be signed by proposer
    const emptySignature: Signature = "0".repeat(128);

    return {
      messageType: "schedule",
      messageId,
      identifier,
      nonce: 0n,
      senderAddress: creator,
      recipientAddress: target,
      networkFeeUnit: 1n,
      networkFeeLimit: 100000n,
      transferValue: 0n,
      data,
      scheduleCondition: condition,
      signature: emptySignature,
      timestamp,
      scheduleType:
        condition.time?.type === "periodic"
          ? "time_periodic"
          : "logic_condition",
      modifiable: false,
      perpetual: true, // Agent schedules run forever unless cancelled
      maxExecutions: undefined, // Unlimited
      prepaidFee,
    };
  }

  // ============ Metrics ============

  /**
   * Get autonomy metrics
   */
  getMetrics() {
    const spuMetrics = this.spu.getMetrics();
    return {
      ...spuMetrics,
      enabled: this.config.enabled,
      maxScheduledTxPerBlock: this.config.maxScheduledTxPerBlock,
    };
  }

  /**
   * Get active schedule count
   */
  getActiveScheduleCount(): number {
    return this.spu.getActiveProcesses().length;
  }

  // ============ State Export/Import ============

  /**
   * Export state for persistence
   */
  exportState() {
    return this.spu.exportState();
  }

  /**
   * Import state from persistence
   */
  importState(data: ReturnType<ScheduleProcessingUnit["exportState"]>): void {
    this.spu.importState(data);
  }
}

/**
 * Create autonomy integration with default configuration
 */
export function createAutonomyIntegration(
  stateProvider: StateProvider,
  config?: Partial<AutonomyConfig>,
): AutonomyIntegration {
  return new AutonomyIntegration(stateProvider, config);
}
