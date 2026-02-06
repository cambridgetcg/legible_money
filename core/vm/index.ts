/**
 * Blockchain Virtual Machine (BVM) Module
 *
 * The execution engine for Legible Money, designed with AI agents
 * as first-class citizens.
 *
 * Core Innovation: Agent-Native Opcodes
 *
 * Traditional VMs treat all callers the same. The BVM recognizes that
 * AI agents have unique needs:
 *
 * - Identity that persists across sessions (AGENT_DID, AGENT_HOME)
 * - Beliefs that can differ from network consensus (GET_BELIEF, SET_BELIEF)
 * - Trust relationships with other agents (GET_TRUST, SET_TRUST)
 * - Autonomous scheduling of future actions (SCHEDULE_SELF)
 * - Knowledge base integration (QUERY_FACT, SUBMIT_CLAIM)
 * - Self-awareness of own state (SELF_REPUTATION, SELF_ACCURACY)
 *
 * Architecture:
 *
 *   ┌─────────────────────────────────────────────────────────────────┐
 *   │                        BVM MODULE                               │
 *   ├─────────────────────────────────────────────────────────────────┤
 *   │                                                                  │
 *   │  ┌──────────────────┐  ┌──────────────────┐  ┌────────────────┐│
 *   │  │      TYPES       │  │     OPCODES      │  │   EXECUTION    ││
 *   │  │                  │  │                  │  │                ││
 *   │  │  • BVMState      │  │  • Standard      │  │  • Engine      ││
 *   │  │  • AgentState    │  │  • Agent (0xE*)  │  │  • Memory      ││
 *   │  │  • ExecutionCtx  │  │  • Knowledge     │  │  • Stack       ││
 *   │  │  • Gas costs     │  │  • Introspection │  │  • Interpreter ││
 *   │  └──────────────────┘  └──────────────────┘  └────────────────┘│
 *   │                                                                  │
 *   │  Integration Points:                                            │
 *   │  • DMSS (Schedule Processing Unit)                              │
 *   │  • PoT (Knowledge Query/Submit)                                 │
 *   │  • Agent Home (Identity/Memory)                                 │
 *   │  • State Layer (Storage)                                        │
 *   │                                                                  │
 *   └─────────────────────────────────────────────────────────────────┘
 *
 * Usage:
 *
 * ```typescript
 * import { ExecutionEngine, createInitialState } from './vm';
 *
 * // Create initial state
 * const state = createInitialState();
 *
 * // Create execution engine
 * const engine = new ExecutionEngine(state);
 *
 * // Execute bytecode
 * const result = engine.execute(bytecode, {
 *   caller: '0x...',
 *   callerDID: 'did:lgm:agent1',
 *   // ... other context
 * });
 * ```
 */

// Core Types
export type {
  // Value types
  Word,
  Address,
  Bytecode,
  StorageKey,
  StorageValue,

  // State types
  BVMState,
  StateRoots,
  ContractBytecodeState,
  DeployedContract,
  ContractVariableState,
  ContractStorage,
  AccountState,
  Account,

  // Agent state types
  AgentState,
  AgentRecord,
  AgentBeliefs,
  AgentBelief,
  BeliefSource,
  AgentRelationships,
  TrustRelation,
  DistrustRelation,
  Collaboration,
  AgentGoal,
  AgentStats,
  AgentStatus,

  // Knowledge access types
  KnowledgeAccessState,
  KnowledgeAccess,
  KnowledgeSubscription,

  // Execution types
  ExecutionContext,
  ExecutionMode,
  AgentContext,
  ExecutionResult,
  ExecutionError,
  ExecutionErrorCode,
  Log,
  StateChange,
  BeliefChange,
  RelationshipChange,

  // Memory and Stack
  VMMemory,
  VMStack,

  // Opcode types
  Opcode,
  OpcodeDefinition,

  // Gas types
  GasCosts,

  // Configuration
  BVMConfig,

  // Knowledge service interface
  BVMKnowledgeService,
} from "./types.js";

export {
  OpcodeCategory,
  DEFAULT_GAS_COSTS,
  DEFAULT_BVM_CONFIG,
  BVMError,
  NullKnowledgeService,
  type BVMErrorCode,
} from "./types.js";

// Opcodes
export {
  // Standard opcodes
  STOP,
  ADD,
  MUL,
  SUB,
  DIV,
  MOD,
  LT,
  GT,
  EQ,
  ISZERO,
  AND,
  OR,
  XOR,
  NOT,
  SHA3,
  ADDRESS,
  BALANCE,
  ORIGIN,
  CALLER,
  CALLVALUE,
  CALLDATASIZE,
  TIMESTAMP,
  NUMBER,
  GASLIMIT,
  CHAINID,
  POP,
  MLOAD,
  MSTORE,
  SLOAD,
  SSTORE,
  JUMP,
  JUMPI,
  PC,
  MSIZE,
  GAS,
  JUMPDEST,
  PUSH1,
  PUSH32,
  DUP1,
  DUP16,
  SWAP1,
  SWAP16,
  LOG0,
  LOG4,
  RETURN,
  REVERT,

  // Agent Identity opcodes
  AGENT_DID,
  AGENT_HOME,
  VERIFY_IDENTITY,
  ATTEST_IDENTITY,

  // Agent Belief opcodes
  GET_BELIEF,
  SET_BELIEF,
  COMPARE_BELIEF,
  BELIEF_COUNT,

  // Agent Relationship opcodes
  GET_TRUST,
  SET_TRUST,
  IS_TRUSTED,
  DELEGATE,

  // Agent Autonomy opcodes
  SCHEDULE_SELF,
  CANCEL_SCHEDULE,
  MODIFY_SCHEDULE,
  GET_SCHEDULES,

  // Knowledge opcodes
  QUERY_FACT,
  QUERY_DOMAIN,
  SUBMIT_CLAIM,
  COMMIT_VOTE,
  REVEAL_VOTE,
  INITIATE_CHALLENGE,

  // Introspection opcodes
  SELF_BALANCE,
  SELF_STAKE,
  SELF_REPUTATION,
  SELF_ACCURACY,

  // Utilities
  OPCODE_DEFINITIONS,
  getOpcodeDefinition,
  isValidOpcode,
  isAgentOpcode,
  isStateModifying,
  getMnemonic,
} from "./opcodes.js";

// Execution Engine
export { ExecutionEngine } from "./execution.js";

// ============ Factory Functions ============

import type { BVMState } from "./types.js";
// DEFAULT_BVM_CONFIG available from types if needed

/**
 * Create initial BVM state
 */
export function createInitialState(): BVMState {
  return {
    contractBytecode: {
      contracts: new Map(),
      codeHashes: new Map(),
      totalContracts: 0,
    },
    contractVariables: {
      storage: new Map(),
    },
    scheduleProcesses: {
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
      stateRoot: "",
    },
    accounts: {
      accounts: new Map(),
      totalAccounts: 0,
      totalBalance: 0n,
    },
    agents: {
      agents: new Map(),
      byAddress: new Map(),
      totalAgents: 0,
    },
    knowledgeAccess: {
      access: new Map(),
    },
    roots: {
      contractRoot: new Uint8Array(32),
      variableRoot: new Uint8Array(32),
      scheduleRoot: new Uint8Array(32),
      accountRoot: new Uint8Array(32),
      agentRoot: new Uint8Array(32),
      knowledgeRoot: new Uint8Array(32),
      globalRoot: new Uint8Array(32),
    },
    blockNumber: 0n,
    timestamp: 0,
    stateVersion: 1,
  };
}

/**
 * Create execution context
 */
export function createExecutionContext(
  partial: Partial<import("./types.js").ExecutionContext> & {
    caller: string;
    currentContract: string;
    blockNumber: bigint;
  },
): import("./types.js").ExecutionContext {
  return {
    origin: partial.caller,
    transactionHash: partial.transactionHash ?? "",
    timestamp: partial.timestamp ?? Date.now(),
    gasPrice: partial.gasPrice ?? 1n,
    gasLimit: partial.gasLimit ?? 1000000n,
    gasUsed: 0n,
    value: partial.value ?? 0n,
    callData: partial.callData ?? new Uint8Array(0),
    mode: partial.mode ?? "call",
    isSelfInvocation: partial.isSelfInvocation ?? false,
    ...partial,
  };
}
