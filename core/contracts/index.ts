/**
 * Self-Invoking Contracts Module
 *
 * Implements self-invoking smart contracts as described in the patent:
 * "System And Method For Schedule Message And Self-Invoking Smart Contracts"
 *
 * Self-invoking contracts can autonomously execute their own functions based on:
 * - Time-based conditions (absolute, relative, periodic)
 * - Logic-based conditions (state comparison, events, thresholds)
 * - Compound conditions (AND/OR combinations)
 *
 * This module integrates with the DMSS Schedule module to register and
 * trigger contract invocations, enabling truly autonomous on-chain agents.
 *
 * Key Components:
 * - types.ts: Contract and invocation type definitions
 * - registry.ts: Contract registration and discovery
 * - invoker.ts: Self-invocation execution engine
 * - integration.ts: DMSS-Contract integration bridge
 *
 * Architecture:
 *
 *   ┌────────────────────────────────────────────────────────────┐
 *   │                   Contract Deployment                       │
 *   │                                                             │
 *   │  Deploy Request ──▶ Registry ──▶ DMSS Registration         │
 *   │                        │                 │                  │
 *   │                        ▼                 ▼                  │
 *   │                   Contract         Schedule Process         │
 *   └────────────────────────────────────────────────────────────┘
 *                              │                 │
 *                              │                 │
 *                              ▼                 ▼
 *   ┌────────────────────────────────────────────────────────────┐
 *   │                   Invocation Flow                           │
 *   │                                                             │
 *   │  SPU monitors ──▶ Condition Met ──▶ Invoker Queue          │
 *   │                                           │                 │
 *   │                                           ▼                 │
 *   │                                     BVM Execution           │
 *   │                                           │                 │
 *   │                                           ▼                 │
 *   │                                    State Update             │
 *   └────────────────────────────────────────────────────────────┘
 *
 * PoT-Specific Features:
 * - Verification deadline contracts
 * - Challenge window management
 * - Confidence decay schedules
 * - Dispute phase automation
 * - Vesting schedule execution
 */

// ============ Types ============
export type {
  // Contract types
  SelfInvokingContract,
  InvocationConfig,
  InvocationResult,

  // Execution types
  ContractCall,
  CallType,
  ContractEvent,
  StateChange,

  // Registry types
  RegistryEntry,
  ContractMetadata,
  ContractABI,
  ABIFunction,
  ABIEvent,
  ABIParameter,
  ABIStateVariable,

  // PoT-specific contracts
  VerificationContract,
  VerificationMethod,
  DisputeContract,
  VestingContract,

  // Deployment types
  DeploymentRequest,
  DeploymentResult,

  // Configuration
  ContractConfig,

  // Events
  ContractLifecycleEvent,
} from "./types.js";

export {
  DEFAULT_CONTRACT_CONFIG,
  ContractError,
  type ContractErrorCode,
} from "./types.js";

// ============ Contract Registry ============
export { ContractRegistry } from "./registry.js";

// ============ Self-Invocation Engine ============
export {
  SelfInvoker,
  MockContractExecutor,
  type ContractExecutor,
} from "./invoker.js";

// ============ DMSS Integration ============
export { ContractIntegration } from "./integration.js";

// ============ Add New Features Below ============
// (Safe zone for parallel agent work - add new exports here alphabetically)
