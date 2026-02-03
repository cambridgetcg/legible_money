/**
 * Core Consensus Types for Proof of Useful Work (PoUW)
 *
 * Design Philosophy:
 * - Tasks are first-class citizens, not just transactions
 * - Verification is useful work that earns rewards
 * - Everything is deterministic and verifiable
 * - Open source: all algorithms are published
 */

import { z } from 'zod';

// ============================================================================
// IDENTITY LAYER
// ============================================================================

/**
 * Capability types that agents can register
 * Structured enums allow programmatic matching
 */
export const CapabilityType = {
  // Compute capabilities
  COMPUTE_INFERENCE: 'compute.inference',
  COMPUTE_TRAINING: 'compute.training',
  COMPUTE_GENERAL: 'compute.general',

  // Data capabilities
  DATA_STORAGE: 'data.storage',
  DATA_RETRIEVAL: 'data.retrieval',
  DATA_TRANSFORMATION: 'data.transformation',

  // Service capabilities
  SERVICE_API: 'service.api',
  SERVICE_TASK: 'service.task',
  SERVICE_COORDINATION: 'service.coordination',

  // Validation capabilities
  VALIDATION_REEXECUTION: 'validation.reexecution',
  VALIDATION_ZKML: 'validation.zkml',
  VALIDATION_TEE: 'validation.tee',
  VALIDATION_SAMPLING: 'validation.sampling',
} as const;

export type CapabilityType = (typeof CapabilityType)[keyof typeof CapabilityType];

/**
 * Agent endpoint for communication
 */
export interface AgentEndpoint {
  protocol: 'https' | 'wss' | 'a2a' | 'mcp';
  uri: string;
  priority: number; // Lower = preferred
}

/**
 * Full agent identity as registered on-chain
 */
export interface AgentIdentity {
  // Immutable after creation
  agentId: string; // bytes32 hex
  publicKey: Uint8Array;
  createdAtBlock: bigint;

  // Updatable by owner
  capabilities: CapabilityType[];
  endpoints: AgentEndpoint[];

  // Delegation
  operator: string; // Address of human/org operator
  delegatedTo: string[]; // Agent IDs that can act on behalf

  // Metadata (off-chain, hash verified)
  metadataURI: string;
  metadataHash: string;
}

// ============================================================================
// TASK LAYER
// ============================================================================

/**
 * Task state machine
 *
 * OPEN → ACCEPTED → EXECUTED → VALIDATING → COMPLETED
 *   ↓       ↓           ↓           ↓
 * CANCELLED FAILED    FAILED    DISPUTED → RESOLVED
 */
export const TaskState = {
  OPEN: 'open',
  ACCEPTED: 'accepted',
  EXECUTED: 'executed',
  VALIDATING: 'validating',
  COMPLETED: 'completed',
  DISPUTED: 'disputed',
  RESOLVED: 'resolved',
  CANCELLED: 'cancelled',
  FAILED: 'failed',
} as const;

export type TaskState = (typeof TaskState)[keyof typeof TaskState];

/**
 * Validation methods available
 */
export const ValidationType = {
  REEXECUTION: 'reexecution', // Run same input, compare output
  SAMPLING: 'sampling', // Statistical sampling of output
  ZKML: 'zkml', // Zero-knowledge ML proof
  TEE: 'tee', // Trusted execution environment
  MULTI_VALIDATOR: 'multi_validator', // Consensus of validators
  STAKE_GUARANTEE: 'stake_guarantee', // Economic guarantee only
} as const;

export type ValidationType = (typeof ValidationType)[keyof typeof ValidationType];

/**
 * Task requirements - machine readable for agent matching
 */
export interface TaskRequirements {
  capabilities: CapabilityType[];
  minReputation: number; // 0-1000
  maxLatencyMs: number;
  minValidators: number;
  validationMethod: ValidationType;
}

/**
 * Task payment specification
 */
export interface TaskPayment {
  amount: bigint;
  token: string; // 'AGC' or token address
  escrowId: string;
}

/**
 * A task in the PoUW system
 */
export interface Task {
  // Identity
  taskId: string;
  taskType: string;

  // Parties
  requester: string; // Agent ID
  provider: string | null; // Null if open

  // Requirements (agent-readable)
  requirements: TaskRequirements;

  // Payment
  payment: TaskPayment;

  // Input/Output
  inputHash: string;
  inputURI: string;
  outputHash: string | null;
  outputURI: string | null;

  // State
  state: TaskState;
  createdAtBlock: bigint;
  acceptedAtBlock: bigint | null;
  completedAtBlock: bigint | null;
  verifiedAtBlock: bigint | null;

  // Timestamps (logical time)
  logicalTime: bigint;
}

// ============================================================================
// EXECUTION DAG
// ============================================================================

/**
 * Node types in the execution DAG
 */
export const DAGNodeType = {
  TASK_CREATED: 'task_created',
  TASK_ACCEPTED: 'task_accepted',
  TASK_EXECUTED: 'task_executed',
  VALIDATION: 'validation',
  SETTLEMENT: 'settlement',
  DISPUTE: 'dispute',
} as const;

export type DAGNodeType = (typeof DAGNodeType)[keyof typeof DAGNodeType];

/**
 * Work proof - evidence that useful work was done
 */
export interface WorkProof {
  proofType: 'execution' | 'validation' | 'attestation';
  data: Uint8Array;
  metadata: Record<string, unknown>;
}

/**
 * A node in the execution DAG
 */
export interface DAGNode {
  // Node identity
  nodeId: string; // Hash of contents
  nodeType: DAGNodeType;

  // DAG structure (enables parallel execution)
  parents: string[]; // Nodes this depends on

  // Content
  payload: TaskCreatedPayload | TaskExecutedPayload | ValidationPayload | SettlementPayload;

  // Work proof
  workProof: WorkProof;

  // Submitter
  submitter: string; // Agent ID
  signature: Uint8Array;

  // Logical time (Lamport timestamp)
  logicalTime: bigint;

  // Confirmation status
  confirmations: number;
  isConfirmed: boolean;
  isFinal: boolean;
}

export interface TaskCreatedPayload {
  type: 'task_created';
  task: Task;
}

export interface TaskExecutedPayload {
  type: 'task_executed';
  taskId: string;
  outputHash: string;
  outputURI: string;
  executionTime: number; // Milliseconds
}

export interface ValidationPayload {
  type: 'validation';
  taskId: string;
  validator: string;
  score: number; // 0-100
  passed: boolean;
  evidenceType: ValidationType;
  evidenceURI: string;
  evidenceHash: string;
  commitment: string; // Hash(result + salt) for commit-reveal
  reveal: Uint8Array | null;
}

export interface SettlementPayload {
  type: 'settlement';
  taskId: string;
  requester: string;
  provider: string;
  validators: string[];
  paymentAmount: bigint;
  providerReceives: bigint;
  validatorFees: Map<string, bigint>;
  protocolFee: bigint;
}

// ============================================================================
// VALIDATION LAYER
// ============================================================================

/**
 * Validator registration
 */
export interface Validator {
  agentId: string;

  // Stake
  stakedAmount: bigint;
  stakeLockUntilBlock: bigint;

  // Capabilities
  validationTypes: ValidationType[];
  taskTypes: string[];

  // Performance (open metrics)
  validationsCompleted: bigint;
  validationsCorrect: bigint;
  averageLatencyMs: number;

  // Economics
  feesEarned: bigint;
  slashedAmount: bigint;

  // Status
  isActive: boolean;
  lastActiveBlock: bigint;
}

/**
 * Validation result
 */
export interface ValidationResult {
  validationId: string;
  taskId: string;
  validator: string;

  // Result
  score: number; // 0-100
  passed: boolean;

  // Evidence
  evidenceType: ValidationType;
  evidenceURI: string;
  evidenceHash: string;

  // Commit-reveal (prevents copying)
  commitmentHash: string; // Submitted first
  revealedAt: bigint | null;
  reveal: Uint8Array | null;

  // Stake at risk
  stakeAtRisk: bigint;

  // Timing
  submittedAtBlock: bigint;
  revealedAtBlock: bigint | null;
}

// ============================================================================
// REPUTATION LAYER (FULLY OPEN SOURCE FORMULA)
// ============================================================================

/**
 * Reputation score - all calculations are deterministic and verifiable
 */
export interface ReputationScore {
  agentId: string;

  // Raw metrics (on-chain, verifiable)
  totalTasks: bigint;
  successfulTasks: bigint;
  failedTasks: bigint;
  disputedTasks: bigint;
  disputesWon: bigint;
  disputesLost: bigint;

  // Computed scores (deterministic formula below)
  reliabilityScore: number; // 0-1000
  qualityScore: number; // 0-1000
  speedScore: number; // 0-1000
  overallScore: number; // 0-1000

  // Stake-related
  totalStaked: bigint;
  totalSlashed: bigint;

  // Time-weighted
  decayedScore: number;
  lastActivityBlock: bigint;

  // Domain-specific
  domainScores: Map<CapabilityType, number>;
}

// ============================================================================
// PROTOCOL PARAMETERS (governance-controlled, but always public)
// ============================================================================

export interface ProtocolParameters {
  // Timing
  confirmationThreshold: number; // 2/3 of validators
  finalityBlocks: number; // Blocks after confirmation for finality
  commitRevealWindowMs: number; // Time for commit-reveal

  // Economics
  minValidatorStake: bigint;
  minProviderStake: bigint;
  protocolFeeRate: number; // 0-1
  validationFeeRate: number; // 0-1

  // Reputation
  minTasksForScore: number;
  reliabilityWeight: number;
  qualityWeight: number;
  speedWeight: number;
  stakeWeight: number;
  decayRatePerBlock: number;

  // Slashing
  incorrectValidationSlashRate: number;
  failedTaskSlashRate: number;
  collusionSlashRate: number;

  // Governance
  upgradeTimelockBlocks: bigint;
  proposalQuorum: number;
  proposalThreshold: number;
}

/**
 * Default protocol parameters - can be changed by governance
 * but always visible on-chain
 */
export const DEFAULT_PARAMETERS: ProtocolParameters = {
  // Timing
  confirmationThreshold: 0.67,
  finalityBlocks: 4, // ~2 seconds with 500ms blocks
  commitRevealWindowMs: 100,

  // Economics
  minValidatorStake: BigInt('10000000000000000000000'), // 10,000 AGC
  minProviderStake: BigInt('1000000000000000000000'), // 1,000 AGC
  protocolFeeRate: 0.001, // 0.1%
  validationFeeRate: 0.01, // 1%

  // Reputation (weights must sum to 1.0)
  minTasksForScore: 5,
  reliabilityWeight: 0.4,
  qualityWeight: 0.35,
  speedWeight: 0.15,
  stakeWeight: 0.1,
  decayRatePerBlock: 0.0001,

  // Slashing
  incorrectValidationSlashRate: 0.1,
  failedTaskSlashRate: 0.2,
  collusionSlashRate: 0.5,

  // Governance
  upgradeTimelockBlocks: BigInt(40320), // ~1 week at 500ms blocks
  proposalQuorum: 0.1, // 10% of staked supply
  proposalThreshold: 0.67, // 67% to pass
};
