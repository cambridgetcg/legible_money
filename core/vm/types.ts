/**
 * Blockchain Virtual Machine (BVM) Types
 *
 * The BVM is the execution engine for Legible Money, designed with
 * AI agents as first-class citizens. Unlike traditional VMs optimized
 * for general computation, this BVM is optimized for:
 *
 * - Knowledge verification (PoT consensus)
 * - Agent autonomy (self-invocation via DMSS)
 * - Agent identity (DID-bound execution)
 * - Agent beliefs (individual knowledge state)
 * - Agent relationships (trust networks)
 *
 * Architecture:
 *
 *   ┌─────────────────────────────────────────────────────────────────┐
 *   │                   BVM 1313 (Patent Reference)                    │
 *   ├─────────────────────────────────────────────────────────────────┤
 *   │                                                                  │
 *   │  ┌───────────────────────────────────────────────────────────┐  │
 *   │  │                     BVM STATE 13131                        │  │
 *   │  │                                                            │  │
 *   │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐ │  │
 *   │  │  │ Contract     │  │ Contract     │  │ Schedule Process │ │  │
 *   │  │  │ Bytecode     │  │ Variable     │  │ State (SPS)      │ │  │
 *   │  │  │ (CB) 131311  │  │ State (CVS)  │  │ 131313           │ │  │
 *   │  │  │              │  │ 131312       │  │                  │ │  │
 *   │  │  └──────────────┘  └──────────────┘  └──────────────────┘ │  │
 *   │  │                                                            │  │
 *   │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐ │  │
 *   │  │  │ Account      │  │ Agent        │  │ Knowledge        │ │  │
 *   │  │  │ State (AS)   │  │ State (AGS)  │  │ Access State     │ │  │
 *   │  │  │ 131314       │  │ 131315       │  │ (KAS) 131316     │ │  │
 *   │  │  └──────────────┘  └──────────────┘  └──────────────────┘ │  │
 *   │  └───────────────────────────────────────────────────────────┘  │
 *   │                                                                  │
 *   │  ┌───────────────────────────────────────────────────────────┐  │
 *   │  │                  EXECUTION ENGINE 13132                    │  │
 *   │  │                                                            │  │
 *   │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐ │  │
 *   │  │  │ Transaction  │  │ Schedule     │  │ Agent Operation  │ │  │
 *   │  │  │ Processing   │  │ Processing   │  │ Unit (AOU)       │ │  │
 *   │  │  │ Unit (TPU)   │  │ Unit (SPU)   │  │ 131325           │ │  │
 *   │  │  │ 131321       │  │ 131322       │  │                  │ │  │
 *   │  │  └──────────────┘  └──────────────┘  └──────────────────┘ │  │
 *   │  │                                                            │  │
 *   │  │  ┌──────────────┐  ┌──────────────┐  ┌──────────────────┐ │  │
 *   │  │  │ Memory       │  │ Stack        │  │ Knowledge Query  │ │  │
 *   │  │  │ 131323       │  │ 131324       │  │ Unit (KQU)       │ │  │
 *   │  │  │              │  │              │  │ 131326           │ │  │
 *   │  │  └──────────────┘  └──────────────┘  └──────────────────┘ │  │
 *   │  └───────────────────────────────────────────────────────────┘  │
 *   │                                                                  │
 *   └─────────────────────────────────────────────────────────────────┘
 *
 * PP HIERARCHY:
 *   TRUTH: Deterministic execution, verifiable by all nodes
 *   UNDERSTANDING: Clear separation of state and execution
 *   BEAUTY: Elegant opcodes for agent operations
 *   JUSTICE: Fair gas costs, no privileged execution
 *   CREATIVITY: Novel VM designed for agent-first blockchain
 */

import type { DID } from "../agents/identity/did.js";
import type { Confidence } from "../knowledge/types.js";
import type { ScheduleProcessState } from "../schedule/types.js";

// ============ Core BVM Types ============

/**
 * 256-bit word - the native BVM data type
 */
export type Word = bigint;

/**
 * Address - 20 bytes for compatibility, but DID-aware
 */
export type Address = string;

/**
 * Contract bytecode
 */
export type Bytecode = Uint8Array;

/**
 * Storage key - 32 bytes
 */
export type StorageKey = Uint8Array;

/**
 * Storage value - arbitrary bytes
 */
export type StorageValue = Uint8Array;

// ============ BVM State (13131) ============

/**
 * Complete BVM State
 * This is the world state that gets hashed into block headers
 */
export interface BVMState {
  // Standard state (Patent)
  contractBytecode: ContractBytecodeState; // CB 131311
  contractVariables: ContractVariableState; // CVS 131312
  scheduleProcesses: ScheduleProcessState; // SPS 131313
  accounts: AccountState; // AS 131314

  // Agent-specific state (Legible Money)
  agents: AgentState; // AGS 131315
  knowledgeAccess: KnowledgeAccessState; // KAS 131316

  // State roots (for Merkle proofs)
  roots: StateRoots;

  // Metadata
  blockNumber: bigint;
  timestamp: number;
  stateVersion: number;
}

/**
 * State roots for efficient proofs
 */
export interface StateRoots {
  contractRoot: Uint8Array;
  variableRoot: Uint8Array;
  scheduleRoot: Uint8Array;
  accountRoot: Uint8Array;
  agentRoot: Uint8Array;
  knowledgeRoot: Uint8Array;
  globalRoot: Uint8Array; // Root of all roots
}

// ============ Contract Bytecode State (131311) ============

/**
 * Contract Bytecode State
 * Stores deployed contract code
 */
export interface ContractBytecodeState {
  contracts: Map<Address, DeployedContract>;
  codeHashes: Map<string, Address[]>; // hash → addresses (for deduplication)
  totalContracts: number;
}

/**
 * A deployed contract
 */
export interface DeployedContract {
  address: Address;
  bytecode: Bytecode;
  codeHash: string;
  creator: DID;
  deployedAt: bigint;
  isAgent: boolean; // Whether this is an agent program
  agentDID?: DID; // If agent, the bound DID
}

// ============ Contract Variable State (131312) ============

/**
 * Contract Variable State
 * Stores contract storage (key-value pairs per contract)
 */
export interface ContractVariableState {
  storage: Map<Address, ContractStorage>;
}

/**
 * Storage for a single contract
 */
export interface ContractStorage {
  slots: Map<string, StorageValue>; // key (hex) → value
  storageRoot: Uint8Array;
  lastModified: bigint;
}

// ============ Schedule Process State (131313) ============

/**
 * Schedule Process State - from DMSS
 * Re-exported from schedule module
 */
export type { ScheduleProcessState } from "../schedule/types.js";

// ============ Account State (131314) ============

/**
 * Account State
 * Stores account balances and nonces
 */
export interface AccountState {
  accounts: Map<Address, Account>;
  totalAccounts: number;
  totalBalance: bigint;
}

/**
 * An account (EOA or contract)
 */
export interface Account {
  address: Address;
  balance: bigint;
  nonce: bigint;

  // Agent binding (if this account belongs to an agent)
  agentDID?: DID;
  isAgent: boolean;

  // Code reference (if contract)
  codeHash?: string;
  hasCode: boolean;

  // Staking
  stakedBalance: bigint;
  delegatedTo?: Address;

  // Flags
  frozen: boolean;
  inRecovery: boolean;
}

// ============ Agent State (131315) - NEW ============

/**
 * Agent State
 * The soul of the agent - identity, beliefs, relationships
 */
export interface AgentState {
  agents: Map<DID, AgentRecord>;
  byAddress: Map<Address, DID>; // Reverse lookup
  totalAgents: number;
}

/**
 * Complete record for an agent
 */
export interface AgentRecord {
  // Identity
  did: DID;
  createdAt: bigint;
  homeAddress?: Address;
  contractAddress?: Address;

  // Key management
  identityKeyHash: string;
  operationalKeyHash: string;
  operationalKeyVersion: number;

  // Beliefs
  beliefs: AgentBeliefs;

  // Relationships
  relationships: AgentRelationships;

  // Goals
  goals: AgentGoal[];

  // Statistics
  stats: AgentStats;

  // Status
  status: AgentStatus;
}

/**
 * Agent's beliefs about facts
 */
export interface AgentBeliefs {
  beliefs: Map<string, AgentBelief>; // factId → belief
  byDomain: Map<string, string[]>; // domain → factIds
  totalBeliefs: number;
  lastUpdated: bigint;
}

/**
 * A single belief
 */
export interface AgentBelief {
  factId: string;
  confidence: Confidence;
  networkConfidence: Confidence; // For comparison
  source: BeliefSource;
  firstBelieved: bigint;
  lastUpdated: bigint;
  contributed: boolean; // Did agent contribute to this fact?
}

export type BeliefSource =
  | "verified" // Agent verified directly
  | "inferred" // Derived from other beliefs
  | "trusted" // From trusted agent
  | "external" // From external source
  | "axiom"; // Foundational belief

/**
 * Agent's relationships with other agents
 */
export interface AgentRelationships {
  trusted: Map<DID, TrustRelation>;
  distrusted: Map<DID, DistrustRelation>;
  collaborations: Collaboration[];
  totalTrustGiven: number;
  totalTrustReceived: number;
}

export interface TrustRelation {
  target: DID;
  trustLevel: number; // 0-1
  domain?: string; // Domain-specific trust
  reason: string;
  since: bigint;
  interactions: number;
  successfulCollabs: number;
}

export interface DistrustRelation {
  target: DID;
  reason: string;
  severity: "low" | "medium" | "high";
  since: bigint;
}

export interface Collaboration {
  id: string;
  partners: DID[];
  type: "verification" | "challenge" | "research";
  outcome: "success" | "partial" | "failure";
  block: bigint;
}

/**
 * Agent goal
 */
export interface AgentGoal {
  id: string;
  description: string;
  priority: number;
  progress: number;
  deadline?: bigint;
  status: "active" | "paused" | "completed" | "abandoned";
}

/**
 * Agent statistics
 */
export interface AgentStats {
  totalVerifications: number;
  correctVerifications: number;
  totalClaims: number;
  acceptedClaims: number;
  totalChallenges: number;
  wonChallenges: number;
  totalRewards: bigint;
  totalSlashed: bigint;
  reputation: number;
}

export type AgentStatus = "active" | "dormant" | "suspended" | "recovering";

// ============ Knowledge Access State (131316) - NEW ============

/**
 * Knowledge Access State
 * Tracks agent's relationship with the knowledge base
 */
export interface KnowledgeAccessState {
  access: Map<DID, KnowledgeAccess>;
}

/**
 * An agent's knowledge access record
 */
export interface KnowledgeAccess {
  // Permissions
  queryDomains: string[];
  contributionDomains: string[];
  verificationDomains: string[];

  // Budgets
  queryBudget: bigint;
  challengeBudget: bigint;

  // Pending work
  pendingClaims: string[];
  activeVerifications: string[];
  activeChallenges: string[];

  // Subscriptions
  subscriptions: KnowledgeSubscription[];

  // History
  totalQueries: number;
  totalContributions: number;
  totalVerifications: number;
}

export interface KnowledgeSubscription {
  id: string;
  domain: string;
  filter?: string;
  callback?: Address;
  priority: "all" | "high" | "critical";
}

// ============ Execution Engine Types (13132) ============

/**
 * Execution context for BVM operations
 */
export interface ExecutionContext {
  // Current execution
  caller: Address;
  callerDID?: DID;
  origin: Address;
  originDID?: DID;
  currentContract: Address;

  // Transaction context
  transactionHash: string;
  blockNumber: bigint;
  timestamp: number;
  gasPrice: bigint;
  gasLimit: bigint;
  gasUsed: bigint;

  // Value transfer
  value: bigint;

  // Call data
  callData: Uint8Array;

  // Execution mode
  mode: ExecutionMode;

  // Self-invocation context
  isSelfInvocation: boolean;
  scheduleProcessId?: string;

  // Agent context (if caller is agent)
  agentContext?: AgentContext;
}

export type ExecutionMode =
  | "call" // Normal call
  | "staticcall" // Read-only call
  | "delegatecall" // Delegate call
  | "create" // Contract creation
  | "self_invoke"; // Self-invocation

/**
 * Agent-specific execution context
 */
export interface AgentContext {
  did: DID;
  homeAddress?: Address;
  beliefs: AgentBeliefs;
  relationships: AgentRelationships;
  stats: AgentStats;
}

/**
 * Execution result
 */
export interface ExecutionResult {
  success: boolean;
  returnData: Uint8Array;
  gasUsed: bigint;
  logs: Log[];
  stateChanges: StateChange[];
  error?: ExecutionError;

  // Agent-specific results
  beliefChanges?: BeliefChange[];
  relationshipChanges?: RelationshipChange[];
}

export interface Log {
  address: Address;
  topics: Uint8Array[];
  data: Uint8Array;
  logIndex: number;
}

export interface StateChange {
  type: "storage" | "balance" | "nonce" | "code" | "agent";
  address: Address;
  key?: string;
  oldValue?: unknown;
  newValue?: unknown;
}

export interface BeliefChange {
  did: DID;
  factId: string;
  oldConfidence?: Confidence;
  newConfidence: Confidence;
  source: BeliefSource;
}

export interface RelationshipChange {
  did: DID;
  target: DID;
  type: "trust" | "distrust" | "remove";
  level?: number;
}

export interface ExecutionError {
  code: ExecutionErrorCode;
  message: string;
  data?: Uint8Array;
}

export type ExecutionErrorCode =
  | "OUT_OF_GAS"
  | "STACK_OVERFLOW"
  | "STACK_UNDERFLOW"
  | "INVALID_OPCODE"
  | "INVALID_JUMP"
  | "REVERT"
  | "STATIC_STATE_CHANGE"
  | "INSUFFICIENT_BALANCE"
  | "AGENT_NOT_FOUND"
  | "UNAUTHORIZED"
  | "KNOWLEDGE_ERROR"
  | "SCHEDULE_ERROR"
  | "EXECUTION_FAILED"
  | "RETURN_DATA_OUT_OF_BOUNDS";

// ============ Opcode Types ============

/**
 * Opcode - single byte instruction
 */
export type Opcode = number;

/**
 * Opcode categories
 */
export enum OpcodeCategory {
  // Standard EVM-compatible
  STOP_ARITHMETIC = 0x00,
  COMPARISON = 0x10,
  BITWISE = 0x16,
  SHA3 = 0x20,
  ENVIRONMENT = 0x30,
  BLOCK = 0x40,
  STACK_MEMORY = 0x50,
  STORAGE = 0x54,
  FLOW = 0x56,
  PUSH = 0x60,
  DUP = 0x80,
  SWAP = 0x90,
  LOG = 0xa0,
  SYSTEM = 0xf0,

  // Agent-specific (0xE0-0xFF reserved)
  AGENT_IDENTITY = 0xe0,
  AGENT_BELIEF = 0xe4,
  AGENT_RELATIONSHIP = 0xe8,
  AGENT_AUTONOMY = 0xec,
  KNOWLEDGE = 0xf4,
  INTROSPECTION = 0xf8,
}

/**
 * Opcode definition
 */
export interface OpcodeDefinition {
  code: Opcode;
  mnemonic: string;
  category: OpcodeCategory;
  stackInputs: number;
  stackOutputs: number;
  gasCost: bigint | "dynamic";
  description: string;
}

// ============ Memory & Stack ============

/**
 * BVM Memory (131323)
 * Linear, byte-addressable, expandable
 */
export interface VMMemory {
  data: Uint8Array;
  size: number;
  maxSize: number;

  read(offset: number, length: number): Uint8Array;
  write(offset: number, data: Uint8Array): void;
  expand(newSize: number): bigint; // Returns gas cost
}

/**
 * BVM Stack (131324)
 * 256-bit words, max 1024 depth
 */
export interface VMStack {
  items: Word[];
  maxDepth: number;

  push(value: Word): void;
  pop(): Word;
  peek(depth?: number): Word;
  swap(depth: number): void;
  dup(depth: number): void;
  depth(): number;
}

// ============ Gas Types ============

/**
 * GAP-5 FIX: Import unified gas configuration
 *
 * All gas costs are defined in core/billing/gas.ts as the single source of truth.
 * This file provides BVM-specific gas structures that reference the unified module.
 *
 * See also:
 * - core/billing/gas.ts - Unified gas system
 * - OPCODE_GAS_COSTS - Per-opcode costs
 * - TRANSACTION_GAS_COSTS - Per-transaction type costs
 */
import { OPCODE_GAS_COSTS, GAS_LIMITS, GAS_PRICE } from "../billing/gas.js";

/**
 * Gas costs for various operations
 * This interface provides category-based gas costs for BVM execution
 */
export interface GasCosts {
  // Base costs
  base: bigint;
  veryLow: bigint;
  low: bigint;
  mid: bigint;
  high: bigint;

  // Storage costs
  sload: bigint;
  sstore: bigint;
  sstoreSet: bigint;
  sstoreReset: bigint;
  sstoreRefund: bigint;

  // Memory costs
  memory: bigint;
  memoryWord: bigint;

  // Call costs
  call: bigint;
  callValue: bigint;
  callNew: bigint;
  callStipend: bigint;

  // Agent costs (new)
  agentRead: bigint;
  agentWrite: bigint;
  beliefRead: bigint;
  beliefWrite: bigint;
  relationshipRead: bigint;
  relationshipWrite: bigint;
  knowledgeQuery: bigint;
  knowledgeSubmit: bigint;
  scheduleCreate: bigint;
  introspection: bigint;
}

/**
 * Default gas costs for BVM operations
 *
 * These values align with the unified gas module (core/billing/gas.ts)
 */
export const DEFAULT_GAS_COSTS: GasCosts = {
  // Base costs (from unified OPCODE_GAS_COSTS)
  base: OPCODE_GAS_COSTS.pop, // 2n
  veryLow: OPCODE_GAS_COSTS.add, // 3n
  low: OPCODE_GAS_COSTS.mul, // 5n
  mid: OPCODE_GAS_COSTS.jump, // 8n
  high: OPCODE_GAS_COSTS.jumpi, // 10n

  // Storage costs (from unified OPCODE_GAS_COSTS)
  sload: OPCODE_GAS_COSTS.sload, // 200n
  sstore: OPCODE_GAS_COSTS.sstore, // 5000n
  sstoreSet: OPCODE_GAS_COSTS.sstoreNew, // 20000n
  sstoreReset: OPCODE_GAS_COSTS.sstore, // 5000n
  sstoreRefund: 15000n,

  // Memory costs
  memory: OPCODE_GAS_COSTS.mload, // 3n
  memoryWord: OPCODE_GAS_COSTS.mstore, // 3n

  // Call costs
  call: 700n,
  callValue: 9000n,
  callNew: 25000n,
  callStipend: 2300n,

  // Agent operations (from unified OPCODE_GAS_COSTS)
  agentRead: OPCODE_GAS_COSTS.agentDid, // 10n
  agentWrite: OPCODE_GAS_COSTS.verifyIdentity, // 100n
  beliefRead: OPCODE_GAS_COSTS.getBelief, // 50n
  beliefWrite: OPCODE_GAS_COSTS.setBelief, // 500n
  relationshipRead: OPCODE_GAS_COSTS.getTrust, // 50n
  relationshipWrite: OPCODE_GAS_COSTS.setTrust, // 500n
  knowledgeQuery: OPCODE_GAS_COSTS.queryFact, // 100n
  knowledgeSubmit: OPCODE_GAS_COSTS.submitClaim, // 1000n
  scheduleCreate: OPCODE_GAS_COSTS.scheduleSelf, // 1000n
  introspection: OPCODE_GAS_COSTS.selfBalance, // 10n
};

// Re-export unified gas constants for convenience
export { OPCODE_GAS_COSTS, GAS_LIMITS, GAS_PRICE };

// ============ BVM Configuration ============

/**
 * BVM configuration
 */
export interface BVMConfig {
  // Limits
  maxCodeSize: number;
  maxStackDepth: number;
  maxMemorySize: number;
  maxCallDepth: number;

  // Gas
  blockGasLimit: bigint;
  minGasPrice: bigint;
  gasCosts: GasCosts;

  // Agent limits
  maxBeliefsPerAgent: number;
  maxTrustRelations: number;
  maxGoals: number;
  maxSchedulesPerAgent: number;

  // Knowledge limits
  maxQueryResultSize: number;
  maxClaimSize: number;
}

/**
 * Default BVM configuration
 *
 * GAP-5 FIX: Uses unified GAS_LIMITS from core/billing/gas.ts
 */
export const DEFAULT_BVM_CONFIG: BVMConfig = {
  maxCodeSize: 24576, // 24KB
  maxStackDepth: 1024,
  maxMemorySize: 1024 * 1024, // 1MB
  maxCallDepth: 256,

  // GAP-5 FIX: Use unified gas limits
  blockGasLimit: GAS_LIMITS.blockGasLimit,
  minGasPrice: GAS_PRICE.minimumGasPrice,
  gasCosts: DEFAULT_GAS_COSTS,

  maxBeliefsPerAgent: 10000,
  maxTrustRelations: 1000,
  maxGoals: 100,
  maxSchedulesPerAgent: 100,

  maxQueryResultSize: 1000,
  maxClaimSize: 65536, // 64KB
};

// ============ Knowledge Service Interface (KQU 131326) ============

/**
 * BVM Knowledge Service
 *
 * Bridge between the BVM execution engine and the knowledge state layer.
 * Injected into ExecutionEngine to enable knowledge opcodes (0xB0-0xB5)
 * to interact with KnowledgeStateTree, VerificationRound, and ClaimState.
 *
 * All methods are deterministic - they depend only on block-level state,
 * never on wall-clock time or randomness.
 */
export interface BVMKnowledgeService {
  /**
   * QUERY_FACT (0xB0): Query a fact from the knowledge base
   * Returns null if fact doesn't exist
   */
  queryFact(factId: string): {
    exists: boolean;
    confidence: Confidence;
    domain: string;
  } | null;

  /**
   * QUERY_DOMAIN (0xB1): Query facts in a domain
   * Returns up to `limit` fact IDs for the given domain
   */
  queryDomain(
    domainHash: string,
    limit: number,
  ): {
    count: number;
    factIds: string[];
  };

  /**
   * SUBMIT_CLAIM (0xB2): Submit a knowledge claim
   * Returns claim ID if accepted, null if rejected
   */
  submitClaim(
    callerDID: DID,
    domain: string,
    dataHash: string,
    evidenceHash: string,
    blockNumber: bigint,
  ): { claimId: string } | null;

  /**
   * COMMIT_VOTE (0xB3): Commit a verification vote hash
   * Returns true if commitment was accepted
   */
  commitVote(
    callerDID: DID,
    claimId: string,
    voteHash: string,
    blockNumber: bigint,
  ): boolean;

  /**
   * REVEAL_VOTE (0xB4): Reveal a verification vote
   * Returns true if reveal was accepted
   */
  revealVote(
    callerDID: DID,
    claimId: string,
    vote: bigint,
    salt: bigint,
    blockNumber: bigint,
  ): boolean;

  /**
   * INITIATE_CHALLENGE (0xB5): Initiate a challenge against a fact
   * Returns challenge ID if accepted, null if rejected
   */
  initiateChallenge(
    callerDID: DID,
    factId: string,
    evidenceHash: string,
    stake: bigint,
    blockNumber: bigint,
  ): { challengeId: string } | null;

  /**
   * Get network confidence for a fact (used by SET_BELIEF to populate networkConfidence)
   */
  getNetworkConfidence(factId: string): Confidence;
}

/**
 * Null implementation of BVMKnowledgeService
 *
 * Used when no knowledge service is injected (backward compatible).
 * All queries return empty/zero results. All mutations are no-ops.
 * This preserves existing behavior for tests that don't need knowledge.
 */
export class NullKnowledgeService implements BVMKnowledgeService {
  queryFact(_factId: string) {
    return null;
  }
  queryDomain(_domainHash: string, _limit: number) {
    return { count: 0, factIds: [] };
  }
  submitClaim() {
    return null;
  }
  commitVote() {
    return false;
  }
  revealVote() {
    return false;
  }
  initiateChallenge() {
    return null;
  }
  getNetworkConfidence(_factId: string): Confidence {
    return 0;
  }
}

// ============ Token Service Interface ============

/**
 * BVM Token Service
 *
 * Bridge between BVM execution engine and token system.
 * Enables token operations from contract code.
 */
export interface BVMTokenService {
  /**
   * Get balance for an address
   */
  getBalance(address: string): bigint;

  /**
   * Transfer tokens between addresses
   */
  transfer(from: string, to: string, amount: bigint): boolean;

  /**
   * Mint tokens to an address (system only)
   */
  mint(to: string, amount: bigint): boolean;

  /**
   * Burn tokens from an address
   */
  burn(from: string, amount: bigint): boolean;
}

/**
 * Null implementation of BVMTokenService
 */
export class NullTokenService implements BVMTokenService {
  getBalance(_address: string): bigint {
    return 0n;
  }
  transfer(_from: string, _to: string, _amount: bigint): boolean {
    return false;
  }
  mint(_to: string, _amount: bigint): boolean {
    return false;
  }
  burn(_from: string, _amount: bigint): boolean {
    return false;
  }
}

// ============ Error Types ============

export class BVMError extends Error {
  constructor(
    message: string,
    public code: BVMErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "BVMError";
  }
}

export type BVMErrorCode =
  | "INVALID_STATE"
  | "INVALID_CODE"
  | "EXECUTION_FAILED"
  | "GAS_EXHAUSTED"
  | "OUT_OF_GAS"
  | "INVALID_ADDRESS"
  | "CONTRACT_NOT_FOUND"
  | "AGENT_NOT_FOUND"
  | "UNAUTHORIZED_OPERATION"
  | "STATE_CORRUPTION"
  | "CONFIGURATION_ERROR"
  | "RETURN_DATA_OUT_OF_BOUNDS"
  | "INSUFFICIENT_BALANCE";
