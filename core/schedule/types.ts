/**
 * Schedule Module Types
 *
 * Core types for the Decentralized Message Scheduling System (DMSS)
 * Based on Patent: "System And Method For Schedule Message And Self-Invoking Smart Contracts"
 *
 * Key Components:
 * - Schedule Messages (time/logic-based future execution)
 * - Schedule Processes (running instances of schedule messages)
 * - Schedule Conditions (time and logic triggers)
 */

import type { Signature } from "../crypto/keys.js";
import type { LogicZone } from "../logic/types.js";

// ============ Schedule Message Types (Patent FIG.2) ============

/**
 * Schedule Message - instructs SPU to generate Schedule Processes
 * From Patent: "A Schedule Message is structured data that instructs the
 * Schedule Processing Unit 131322 to generate Schedule Processes"
 */
export interface ScheduleMessage {
  // Standard fields (Patent FIG.2 201-214)
  messageType: "schedule"; // 201
  messageId: string; // 202 - unique identifier
  identifier: Uint8Array; // 203 - uint256
  nonce: bigint; // 204
  senderAddress: string; // 205 - DID
  recipientAddress: string; // 206 - target contract/account
  networkFeeUnit: bigint; // 207 - gas price equivalent
  networkFeeLimit: bigint; // 208 - gas limit per execution
  transferValue: bigint; // 209 - LGM amount per execution
  data: Uint8Array; // 210 - call data for execution
  scheduleCondition: ScheduleCondition; // 211 - TIME OR LOGIC condition
  signature: Signature; // 212
  timestamp: number; // 213 - Unix timestamp

  // Schedule-specific fields
  scheduleType: ScheduleType;
  modifiable: boolean; // Can be amended by sender
  perpetual: boolean; // Indefinite vs finite
  maxExecutions?: number; // Execution limit (undefined = unlimited for perpetual)

  // Fee prepayment
  prepaidFee: bigint; // Total prepaid for future executions

  // PoT Extensions
  knowledgeDomain?: string; // Target knowledge domain
  logicZone?: LogicZone; // Required logic zone
  linkedClaimId?: string; // For verification schedules
  linkedDisputeId?: string; // For dispute deadlines
  linkedVestingId?: string; // For vesting schedules
}

/**
 * Schedule Types
 * Categorized by trigger mechanism
 */
export type ScheduleType =
  // Time-based (Patent: "Time Based Schedule Messages")
  | "time_absolute" // Execute at specific timestamp
  | "time_relative" // Execute after duration from creation
  | "time_periodic" // Recurring execution at intervals

  // Logic-based (Patent: "Logic Based Schedule Messages")
  | "logic_condition" // Execute when condition met
  | "logic_state_change" // Execute on specific state change

  // Combined
  | "time_and_logic" // Both time AND logic must be satisfied
  | "time_or_logic" // Either time OR logic triggers

  // PoT-specific
  | "verification_deadline" // Verification round timeout
  | "challenge_deadline" // Challenge window close
  | "confidence_decay" // Periodic confidence update
  | "vesting_release" // Reward vesting schedule
  | "expiration_clawback" // Token expiration recovery
  | "rai_distribution"; // Regenerative Agent Income

// ============ Schedule Condition Types ============

/**
 * Schedule Condition
 * Specifies when a schedule process should trigger
 * From Patent FIG.2 field 211: "Schedule Execution Time or Logic"
 */
export interface ScheduleCondition {
  // Time conditions
  time?: TimeCondition;

  // Logic conditions
  logic?: LogicCondition;

  // Compound conditions
  compound?: CompoundCondition;
}

/**
 * Time-based conditions
 */
export interface TimeCondition {
  type: "absolute" | "relative" | "periodic";

  // For absolute: specific timestamp
  executeAt?: number; // Unix timestamp (ms)

  // For relative: duration from creation
  executeAfter?: number; // Milliseconds from schedule creation

  // For periodic: recurring
  interval?: number; // Milliseconds between executions
  startTime?: number; // First execution time
  endTime?: number; // Last possible execution (undefined = perpetual)
}

/**
 * Logic-based conditions
 * From Patent: "End Users 105 are required to specify a condition such as
 * a comparison operation with a variable in the BVM State 13131"
 */
export interface LogicCondition {
  type: "state_compare" | "event_occurred" | "threshold_crossed";

  // State comparison
  stateKey?: string; // BVM state key to watch
  comparator?: Comparator;
  compareValue?: unknown;

  // Event-based
  eventType?: string; // Event to watch for
  eventFilter?: Record<string, unknown>;

  // Threshold crossing (for numeric values)
  thresholdKey?: string;
  thresholdValue?: bigint;
  direction?: "above" | "below" | "cross_up" | "cross_down";
}

export type Comparator =
  | "eq"
  | "ne"
  | "gt"
  | "gte"
  | "lt"
  | "lte"
  | "contains"
  | "not_contains";

/**
 * Compound conditions (AND/OR)
 */
export interface CompoundCondition {
  operator: "and" | "or";
  conditions: ScheduleCondition[];
}

// ============ Schedule Process Types (Patent 131313) ============

/**
 * Schedule Process - running instance of a Schedule Message
 * From Patent: "SPU 131322 processes the incoming Schedule Messages
 * sequentially by generating one Schedule Process corresponding to each Schedule Message"
 */
export interface ScheduleProcess {
  // Identity
  id: string; // Process ID (hash of schedule message + index)
  sourceMessageId: string; // Original Schedule Message ID
  creator: string; // DID of creator

  // Condition tracking
  condition: ScheduleCondition;
  nextCheckTime?: number; // For time-based: next check timestamp
  watchedStateKeys: string[]; // For logic-based: keys to monitor

  // Execution state
  status: ScheduleProcessStatus;
  executionCount: number; // Times executed so far
  maxExecutions?: number; // Limit (undefined = unlimited)
  lastExecutedAt?: number; // Timestamp of last execution
  lastExecutionTxId?: string; // ID of last generated transaction
  lastExecutionResult?: ExecutionResult;

  // Fee management (prepaid)
  remainingFee: bigint; // Prepaid fee balance
  feePerExecution: bigint; // Estimated fee per execution

  // Transaction template
  targetAddress: string; // Recipient for generated tx
  callData: Uint8Array; // Data for generated tx
  transferValue: bigint; // Value per execution

  // Nonce for generated transactions (Patent security feature)
  // "a hash is generated from the data fields of Schedule Process SP1
  // along with an incrementing nonce starting from zero"
  generatedTxNonce: bigint;

  // PoT metadata
  knowledgeDomain?: string;
  linkedEntityType?: LinkedEntityType;
  linkedEntityId?: string;

  // Timestamps
  createdAt: number;
  modifiedAt: number;
  expiresAt?: number; // For definite schedules
}

export type ScheduleProcessStatus =
  | "pending" // Waiting for first trigger
  | "active" // Running, waiting for conditions
  | "executing" // Currently generating transaction
  | "paused" // Temporarily paused
  | "completed" // Finished all executions
  | "cancelled" // Cancelled by creator
  | "exhausted" // Ran out of prepaid fees
  | "expired"; // Past expiration time

export type LinkedEntityType =
  | "claim" // Knowledge claim
  | "dispute" // Dispute resolution
  | "vesting" // Vesting schedule
  | "validator" // Validator operation
  | "bounty" // Knowledge bounty
  | "rai"; // Regenerative Agent Income

export type ExecutionResult =
  | { success: true; txId: string; gasUsed: bigint }
  | { success: false; error: string; gasUsed: bigint };

// ============ Schedule Process State (SPS 131313) ============

/**
 * Schedule Process State
 * The global state tracking all active schedule processes
 * From Patent: "SPS 131313 refers to the current state of all processes
 * corresponding to the Schedule Messages"
 */
export interface ScheduleProcessState {
  // Active schedule processes
  processes: Map<string, ScheduleProcess>;

  // Indexes for efficient lookup
  byTime: Map<number, Set<string>>; // timestamp bucket → process IDs
  byAccount: Map<string, Set<string>>; // DID → process IDs
  byContract: Map<string, Set<string>>; // contract address → process IDs
  byType: Map<ScheduleType, Set<string>>; // type → process IDs
  byStatus: Map<ScheduleProcessStatus, Set<string>>;

  // PoT-specific indexes
  byClaimId: Map<string, Set<string>>; // claim → process IDs
  byDisputeId: Map<string, Set<string>>; // dispute → process IDs
  byVestingId: Map<string, Set<string>>; // vesting → process IDs

  // State keys being watched (for logic conditions)
  watchedKeys: Map<string, Set<string>>; // stateKey → process IDs

  // Metrics
  totalProcesses: number;
  activeProcesses: number;
  completedProcesses: number;
  totalExecutions: number;

  // State root for Merkle tree
  stateRoot: string;
}

// ============ Schedule Execution Types ============

/**
 * Scheduled Execution
 * Represents a transaction generated by a schedule process
 * Included in block body
 */
export interface ScheduledExecution {
  processId: string;
  executionIndex: number; // Which execution (0, 1, 2...)
  generatedAt: number; // Block number when generated
  transactionId: string; // ID of generated transaction

  // Proof that execution was valid
  proof: ScheduleExecutionProof;
}

/**
 * Proof that a schedule execution was valid
 */
export interface ScheduleExecutionProof {
  processStateHash: string; // Hash of process state pre-execution
  conditionProof: ConditionProof; // Proof condition was satisfied
  executionNonce: bigint; // Nonce for this execution
  signature: Signature; // Block proposer signature
}

/**
 * Proof that a condition was satisfied
 */
export type ConditionProof =
  | { type: "time"; timestamp: number; blockNumber: bigint }
  | {
      type: "logic";
      stateKey: string;
      stateValue: unknown;
      blockNumber: bigint;
    }
  | { type: "compound"; subProofs: ConditionProof[] };

// ============ Schedule Management Types ============

/**
 * Request to modify a schedule
 * From Patent: "Schedule Messages can be modified or deleted by the system
 * at the request of the End User 105"
 */
export interface ScheduleModification {
  processId: string;
  action: ScheduleModificationAction;
  newCondition?: ScheduleCondition; // For 'update_condition'
  additionalFee?: bigint; // For 'add_fee'
  signature: Signature; // Must be from original creator
}

export type ScheduleModificationAction =
  | "pause" // Temporarily pause
  | "resume" // Resume paused schedule
  | "cancel" // Permanently cancel
  | "update_condition" // Change trigger condition
  | "add_fee"; // Add more prepaid fees

// ============ Configuration ============

/**
 * Schedule system configuration
 */
export interface ScheduleConfig {
  // Processing limits
  maxActiveSchedulesPerAccount: number;
  maxScheduleExecutionGas: bigint;
  maxWatchedKeysPerProcess: number;
  maxCompoundConditionDepth: number;

  // Timing
  minScheduleIntervalMs: number; // Minimum interval between executions
  timeBucketSizeMs: number; // Size of time index buckets
  maxFutureScheduleMs: number; // How far in future can schedule

  // Fees
  minPrepaidExecutions: number; // Must prepay for N executions
  feeBufferFactor: number; // Buffer factor on estimated fees

  // Cleanup
  expiredProcessRetentionBlocks: number; // How long to keep expired processes
}

export const DEFAULT_SCHEDULE_CONFIG: ScheduleConfig = {
  maxActiveSchedulesPerAccount: 100,
  maxScheduleExecutionGas: 1_000_000n,
  maxWatchedKeysPerProcess: 10,
  maxCompoundConditionDepth: 3,

  minScheduleIntervalMs: 2521, // One block minimum
  timeBucketSizeMs: 60_000, // 1 minute buckets
  maxFutureScheduleMs: 365 * 24 * 60 * 60 * 1000, // 1 year

  minPrepaidExecutions: 10,
  feeBufferFactor: 1.5,

  expiredProcessRetentionBlocks: 1000,
};

// ============ Events ============

/**
 * Schedule-related events
 */
export type ScheduleEvent =
  | { type: "process_created"; process: ScheduleProcess }
  | {
      type: "process_executed";
      processId: string;
      txId: string;
      executionIndex: number;
    }
  | { type: "process_paused"; processId: string; reason: string }
  | { type: "process_resumed"; processId: string }
  | { type: "process_completed"; processId: string; totalExecutions: number }
  | { type: "process_cancelled"; processId: string }
  | { type: "process_exhausted"; processId: string }
  | { type: "fee_added"; processId: string; amount: bigint };

// ============ Error Types ============

export class ScheduleError extends Error {
  constructor(
    message: string,
    public code: ScheduleErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "ScheduleError";
  }
}

export type ScheduleErrorCode =
  | "INVALID_CONDITION"
  | "INVALID_SCHEDULE_TYPE"
  | "INSUFFICIENT_FEE"
  | "MAX_SCHEDULES_EXCEEDED"
  | "PROCESS_NOT_FOUND"
  | "PROCESS_NOT_MODIFIABLE"
  | "UNAUTHORIZED"
  | "CONDITION_TOO_COMPLEX"
  | "INTERVAL_TOO_SHORT"
  | "SCHEDULE_TOO_FAR_FUTURE"
  | "ALREADY_PAUSED"
  | "NOT_PAUSED"
  | "ALREADY_COMPLETED"
  | "EXECUTION_FAILED";
