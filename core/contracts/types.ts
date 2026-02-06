/**
 * Self-Invoking Contracts Module Types
 *
 * Implements the patent's concept of self-invoking smart contracts:
 * "Self-Invoking Smart Contracts are smart contracts that are set up with a
 * schedule message and make use of SPU 131322 to trigger the execution
 * of a smart contract function with predefined parameters at a specified
 * time or when a logic condition is satisfied."
 *
 * This enables contracts to autonomously execute without external triggers,
 * creating truly autonomous on-chain agents.
 *
 * PP HIERARCHY:
 *   TRUTH: Contracts execute deterministically based on verifiable conditions
 *   UNDERSTANDING: Clear separation of contract definition and invocation
 *   BEAUTY: Elegant integration of scheduling with contract execution
 *   JUSTICE: All invocations are transparent and verifiable
 *   CREATIVITY: Enables autonomous on-chain agents (novel for PoT)
 */

import type { Signature } from "../crypto/keys.js";
import type { ScheduleCondition, ScheduleType } from "../schedule/types.js";
import type { DID } from "../agents/identity/did.js";

// ============ Contract Definition Types ============

/**
 * Self-Invoking Contract definition
 * A contract that can schedule its own function calls
 */
export interface SelfInvokingContract {
  /** Unique contract address (derived from creator + nonce) */
  address: string;

  /** Contract creator DID */
  creator: DID;

  /** Contract name */
  name: string;

  /** Contract version */
  version: string;

  /** Contract code hash (for verification) */
  codeHash: string;

  /** Whether contract is active */
  isActive: boolean;

  /** Self-invocation configurations */
  invocations: InvocationConfig[];

  /** Total invocations executed */
  totalInvocations: number;

  /** Contract state hash */
  stateHash: string;

  /** Block when deployed */
  deployedAtBlock: bigint;

  /** Block when last invoked */
  lastInvokedAtBlock?: bigint;

  /** Gas limit for contract operations */
  gasLimit: bigint;

  /** Prepaid gas balance */
  prepaidGas: bigint;

  /** PoT-specific: Associated knowledge domain */
  knowledgeDomain?: string;

  /** PoT-specific: Required verification level */
  requiredVerificationLevel?: number;
}

/**
 * Invocation configuration - defines how a contract invokes itself
 */
export interface InvocationConfig {
  /** Unique invocation ID within contract */
  id: string;

  /** Function to invoke */
  functionName: string;

  /** Function selector (first 4 bytes of function signature hash) */
  functionSelector: Uint8Array;

  /** Encoded function parameters */
  parameters: Uint8Array;

  /** Schedule condition for invocation */
  condition: ScheduleCondition;

  /** Schedule type (from DMSS) */
  scheduleType: ScheduleType;

  /** Whether this invocation is enabled */
  enabled: boolean;

  /** Maximum invocations (undefined = unlimited) */
  maxInvocations?: number;

  /** Current invocation count */
  invocationCount: number;

  /** Gas limit for this specific invocation */
  gasLimit: bigint;

  /** Value to send with invocation (LGM) */
  value: bigint;

  /** Linked schedule process ID (from DMSS) */
  scheduleProcessId?: string;

  /** Last successful invocation block */
  lastInvokedBlock?: bigint;

  /** Last invocation result */
  lastResult?: InvocationResult;
}

/**
 * Result of a self-invocation
 */
export interface InvocationResult {
  /** Whether invocation succeeded */
  success: boolean;

  /** Return data (if any) */
  returnData?: Uint8Array;

  /** Gas used */
  gasUsed: bigint;

  /** Error message (if failed) */
  error?: string;

  /** Events emitted */
  events: ContractEvent[];

  /** State changes */
  stateChanges: StateChange[];

  /** Block when executed */
  executedAtBlock: bigint;

  /** Transaction ID */
  transactionId: string;
}

// ============ Contract Execution Types ============

/**
 * Contract call - internal or external
 */
export interface ContractCall {
  /** Call ID */
  id: string;

  /** Target contract address */
  target: string;

  /** Caller address (contract or account) */
  caller: string;

  /** Function selector */
  functionSelector: Uint8Array;

  /** Encoded call data */
  callData: Uint8Array;

  /** Value sent with call */
  value: bigint;

  /** Gas provided */
  gas: bigint;

  /** Call type */
  callType: CallType;

  /** Whether this is a self-invocation */
  isSelfInvocation: boolean;

  /** Schedule process ID (if from DMSS) */
  scheduleProcessId?: string;
}

export type CallType =
  | "external" // From external transaction
  | "internal" // Contract-to-contract
  | "self_invoke" // Self-invocation via DMSS
  | "delegate" // Delegate call
  | "static"; // Static call (read-only)

/**
 * Contract event emitted during execution
 */
export interface ContractEvent {
  /** Contract that emitted the event */
  contract: string;

  /** Event signature */
  signature: string;

  /** Indexed topics */
  topics: Uint8Array[];

  /** Event data */
  data: Uint8Array;

  /** Log index in transaction */
  logIndex: number;

  /** Block number */
  blockNumber: bigint;

  /** Transaction ID */
  transactionId: string;
}

/**
 * State change during contract execution
 */
export interface StateChange {
  /** Contract address */
  contract: string;

  /** Storage key */
  key: Uint8Array;

  /** Previous value (if any) */
  previousValue?: Uint8Array;

  /** New value */
  newValue: Uint8Array;
}

// ============ Contract Registry Types ============

/**
 * Contract registry entry
 */
export interface RegistryEntry {
  /** Contract address */
  address: string;

  /** Contract metadata */
  metadata: ContractMetadata;

  /** Verification status */
  verified: boolean;

  /** Block when registered */
  registeredAtBlock: bigint;

  /** Registry tags for discovery */
  tags: string[];
}

/**
 * Contract metadata for registry
 */
export interface ContractMetadata {
  /** Contract name */
  name: string;

  /** Contract description */
  description: string;

  /** Contract version */
  version: string;

  /** Author DID */
  author: DID;

  /** Source code hash (for verification) */
  sourceHash?: string;

  /** ABI (Application Binary Interface) */
  abi: ContractABI;

  /** License */
  license?: string;

  /** Project URL */
  projectUrl?: string;
}

/**
 * Contract ABI - describes contract interface
 */
export interface ContractABI {
  /** Contract functions */
  functions: ABIFunction[];

  /** Contract events */
  events: ABIEvent[];

  /** State variables (readable) */
  stateVariables: ABIStateVariable[];
}

/**
 * ABI function definition
 */
export interface ABIFunction {
  /** Function name */
  name: string;

  /** Function selector (4 bytes) */
  selector: string;

  /** Input parameters */
  inputs: ABIParameter[];

  /** Output parameters */
  outputs: ABIParameter[];

  /** State mutability */
  stateMutability: "pure" | "view" | "nonpayable" | "payable";

  /** Whether function can be self-invoked */
  selfInvokable: boolean;
}

/**
 * ABI event definition
 */
export interface ABIEvent {
  /** Event name */
  name: string;

  /** Event signature hash */
  signatureHash: string;

  /** Event parameters */
  inputs: ABIParameter[];

  /** Whether event is anonymous */
  anonymous: boolean;
}

/**
 * ABI parameter
 */
export interface ABIParameter {
  /** Parameter name */
  name: string;

  /** Parameter type */
  type: string;

  /** Whether indexed (for events) */
  indexed?: boolean;

  /** Component types (for tuples/arrays) */
  components?: ABIParameter[];
}

/**
 * ABI state variable
 */
export interface ABIStateVariable {
  /** Variable name */
  name: string;

  /** Variable type */
  type: string;

  /** Getter function name (if public) */
  getter?: string;
}

// ============ PoT-Specific Contract Types ============

/**
 * Knowledge Verification Contract
 * A self-invoking contract specifically for PoT verification
 */
export interface VerificationContract extends SelfInvokingContract {
  /** Knowledge domain this contract verifies */
  verificationDomain: string;

  /** Types of claims this contract can verify */
  claimTypes: string[];

  /** Verification method */
  verificationMethod: VerificationMethod;

  /** Required evidence types */
  requiredEvidenceTypes: string[];

  /** Minimum confidence threshold */
  minConfidenceThreshold: number;
}

/**
 * Verification method for knowledge claims
 */
export interface VerificationMethod {
  /** Method type */
  type: "deterministic" | "probabilistic" | "oracle" | "committee";

  /** Oracle address (if oracle-based) */
  oracleAddress?: string;

  /** Committee addresses (if committee-based) */
  committeeAddresses?: string[];

  /** Threshold for committee consensus */
  committeeThreshold?: number;
}

/**
 * Dispute Resolution Contract
 * Self-invoking contract for managing dispute timelines
 */
export interface DisputeContract extends SelfInvokingContract {
  /** Dispute types this contract handles */
  disputeTypes: string[];

  /** Phase durations (in blocks) */
  phaseDurations: Map<string, bigint>;

  /** Escalation thresholds */
  escalationThresholds: Map<string, bigint>;

  /** Arbiter selection method */
  arbiterSelection: "random" | "reputation" | "stake_weighted";
}

/**
 * Vesting Contract
 * Self-invoking contract for reward vesting
 */
export interface VestingContract extends SelfInvokingContract {
  /** Vesting schedule type */
  vestingType: "linear" | "exponential" | "cliff" | "custom";

  /** Beneficiary DID */
  beneficiary: DID;

  /** Total amount to vest */
  totalAmount: bigint;

  /** Amount already vested */
  vestedAmount: bigint;

  /** Cliff period (if any) */
  cliffPeriod?: bigint;

  /** Vesting duration */
  vestingDuration: bigint;
}

// ============ Contract Deployment Types ============

/**
 * Contract deployment request
 */
export interface DeploymentRequest {
  /** Deployer DID */
  deployer: DID;

  /** Contract bytecode */
  bytecode: Uint8Array;

  /** Constructor arguments */
  constructorArgs: Uint8Array;

  /** Contract metadata */
  metadata: ContractMetadata;

  /** Initial invocation configs */
  invocations: Omit<
    InvocationConfig,
    "id" | "invocationCount" | "scheduleProcessId"
  >[];

  /** Initial prepaid gas */
  prepaidGas: bigint;

  /** Salt for deterministic address (optional) */
  salt?: Uint8Array;

  /** Signature from deployer */
  signature: Signature;
}

/**
 * Deployment result
 */
export interface DeploymentResult {
  /** Deployed contract address */
  address: string;

  /** Contract instance */
  contract: SelfInvokingContract;

  /** Created schedule process IDs */
  scheduleProcessIds: string[];

  /** Deployment transaction ID */
  transactionId: string;

  /** Gas used for deployment */
  gasUsed: bigint;

  /** Block when deployed */
  deployedAtBlock: bigint;
}

// ============ Configuration ============

/**
 * Self-invoking contract system configuration
 */
export interface ContractConfig {
  /** Maximum invocations per contract per block */
  maxInvocationsPerBlock: number;

  /** Maximum gas per self-invocation */
  maxSelfInvocationGas: bigint;

  /** Minimum prepaid gas for deployment */
  minPrepaidGas: bigint;

  /** Gas price for self-invocations */
  selfInvocationGasPrice: bigint;

  /** Maximum concurrent invocations */
  maxConcurrentInvocations: number;

  /** Whether to allow recursive self-invocation */
  allowRecursiveSelfInvoke: boolean;

  /** Maximum recursion depth */
  maxRecursionDepth: number;
}

export const DEFAULT_CONTRACT_CONFIG: ContractConfig = {
  maxInvocationsPerBlock: 100,
  maxSelfInvocationGas: 5_000_000n,
  minPrepaidGas: 100_000n,
  selfInvocationGasPrice: 1n,
  maxConcurrentInvocations: 10,
  allowRecursiveSelfInvoke: false,
  maxRecursionDepth: 3,
};

// ============ Events ============

/**
 * Contract lifecycle events
 */
export type ContractLifecycleEvent =
  | { type: "deployed"; data: DeploymentResult }
  | {
      type: "invocation_scheduled";
      data: {
        contract: string;
        invocationId: string;
        scheduleProcessId: string;
      };
    }
  | {
      type: "self_invoked";
      data: {
        contract: string;
        invocationId: string;
        result: InvocationResult;
      };
    }
  | {
      type: "invocation_failed";
      data: { contract: string; invocationId: string; error: string };
    }
  | { type: "gas_depleted"; data: { contract: string; remainingGas: bigint } }
  | { type: "contract_paused"; data: { contract: string; reason: string } }
  | { type: "contract_resumed"; data: { contract: string } }
  | {
      type: "invocation_disabled";
      data: { contract: string; invocationId: string };
    };

// ============ Error Types ============

export class ContractError extends Error {
  constructor(
    message: string,
    public code: ContractErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "ContractError";
  }
}

export type ContractErrorCode =
  | "CONTRACT_NOT_FOUND"
  | "INVOCATION_NOT_FOUND"
  | "INSUFFICIENT_GAS"
  | "EXECUTION_FAILED"
  | "INVALID_SELECTOR"
  | "UNAUTHORIZED"
  | "CONTRACT_PAUSED"
  | "INVOCATION_DISABLED"
  | "MAX_INVOCATIONS_REACHED"
  | "RECURSION_LIMIT"
  | "RECURSION_NOT_ALLOWED"
  | "MAX_RECURSION_DEPTH"
  | "DEPLOYMENT_FAILED"
  | "INVALID_BYTECODE"
  | "SCHEDULE_FAILED";
