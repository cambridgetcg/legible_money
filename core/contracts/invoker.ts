/**
 * Self-Invocation Engine
 *
 * Executes self-invocations for contracts based on DMSS schedule conditions.
 * This is the bridge between the Schedule Processing Unit (SPU) and contract execution.
 *
 * From Patent: "SPU 131322 regularly checks that all processes set up are served properly.
 * In one embodiment, a process can be woken up by SPU 131322 when the scheduled time
 * is reached."
 *
 * For self-invoking contracts, the SPU triggers the Invoker which then:
 * 1. Validates the invocation conditions
 * 2. Prepares the contract call
 * 3. Executes via the BVM
 * 4. Records results and updates state
 *
 * PP HIERARCHY:
 *   TRUTH: Invocations are deterministic and verifiable
 *   UNDERSTANDING: Clear execution flow from schedule to result
 *   BEAUTY: Single engine handles all self-invocations
 *   JUSTICE: Gas costs are fairly charged from prepaid balance
 *   CREATIVITY: Enables autonomous on-chain agents
 */

import { sha512_256Hex } from "../crypto/hashing.js";
import type { DID } from "../agents/identity/did.js";
import type { EvaluationResult } from "../schedule/condition_evaluator.js";
import {
  type SelfInvokingContract,
  type InvocationConfig,
  type InvocationResult,
  type ContractCall,
  type ContractConfig,
  type ContractLifecycleEvent,
  DEFAULT_CONTRACT_CONFIG,
  ContractError,
} from "./types.js";
import { ContractRegistry } from "./registry.js";

/**
 * Execution context for self-invocation
 */
interface ExecutionContext {
  /** Current block number */
  blockNumber: bigint;

  /** Current timestamp */
  timestamp: number;

  /** Block proposer */
  proposer: DID;

  /** Gas price for this block */
  gasPrice: bigint;

  /** Parent transaction ID (if part of a batch) */
  parentTxId?: string;

  /** Recursion depth */
  recursionDepth: number;
}

/**
 * Pending invocation in the queue
 */
interface PendingInvocation {
  /** Contract address */
  contract: string;

  /** Invocation config ID */
  invocationId: string;

  /** Schedule process ID */
  scheduleProcessId: string;

  /** Condition evaluation result */
  evaluation: EvaluationResult;

  /** Priority (higher = execute first) */
  priority: number;

  /** Added at block */
  addedAtBlock: bigint;
}

/**
 * Self-Invocation Engine
 * Coordinates contract self-invocations based on DMSS schedules
 */
export class SelfInvoker {
  private pendingQueue: PendingInvocation[] = [];
  private executionHistory: Map<string, InvocationResult[]> = new Map();
  private eventEmitters: Array<(event: ContractLifecycleEvent) => void> = [];

  constructor(
    private registry: ContractRegistry,
    private config: ContractConfig = DEFAULT_CONTRACT_CONFIG,
  ) {}

  // ============ Invocation Queue ============

  /**
   * Queue an invocation for execution
   * Called by SPU when schedule condition is satisfied
   */
  queueInvocation(
    contract: SelfInvokingContract,
    invocationId: string,
    scheduleProcessId: string,
    evaluation: EvaluationResult,
    currentBlock: bigint,
  ): void {
    // Validate contract is active
    if (!contract.isActive) {
      throw new ContractError(
        `Contract ${contract.address} is not active`,
        "CONTRACT_PAUSED",
      );
    }

    // Find invocation config
    const invocation = contract.invocations.find((i) => i.id === invocationId);
    if (!invocation) {
      throw new ContractError(
        `Invocation ${invocationId} not found in contract ${contract.address}`,
        "INVOCATION_NOT_FOUND",
      );
    }

    // Validate invocation is enabled
    if (!invocation.enabled) {
      throw new ContractError(
        `Invocation ${invocationId} is disabled`,
        "INVOCATION_DISABLED",
      );
    }

    // Check max invocations
    if (
      invocation.maxInvocations !== undefined &&
      invocation.invocationCount >= invocation.maxInvocations
    ) {
      throw new ContractError(
        `Invocation ${invocationId} has reached max invocations`,
        "MAX_INVOCATIONS_REACHED",
      );
    }

    // Calculate priority based on gas and time waiting
    const priority = this.calculatePriority(contract, invocation, currentBlock);

    // Add to queue
    const pending: PendingInvocation = {
      contract: contract.address,
      invocationId,
      scheduleProcessId,
      evaluation,
      priority,
      addedAtBlock: currentBlock,
    };

    this.pendingQueue.push(pending);
    // P0-C2 FIX: Add deterministic tiebreakers to prevent validator divergence.
    this.pendingQueue.sort(
      (a, b) =>
        b.priority - a.priority ||
        a.contract.localeCompare(b.contract) ||
        a.invocationId.localeCompare(b.invocationId),
    );

    // Emit event
    this.emit({
      type: "invocation_scheduled",
      data: {
        contract: contract.address,
        invocationId,
        scheduleProcessId,
      },
    });
  }

  /**
   * Process pending invocations for a block
   */
  async processPendingInvocations(
    context: ExecutionContext,
    executor: ContractExecutor,
  ): Promise<InvocationResult[]> {
    const results: InvocationResult[] = [];
    let invocationsThisBlock = 0;
    let concurrentCount = 0;

    // Process queue in priority order
    const toProcess = [...this.pendingQueue];
    this.pendingQueue = [];

    for (const pending of toProcess) {
      // Check block limits
      if (invocationsThisBlock >= this.config.maxInvocationsPerBlock) {
        // Re-queue for next block
        this.pendingQueue.push(pending);
        continue;
      }

      // Check concurrent limit
      if (concurrentCount >= this.config.maxConcurrentInvocations) {
        this.pendingQueue.push(pending);
        continue;
      }

      try {
        concurrentCount++;
        const result = await this.executeInvocation(pending, context, executor);
        results.push(result);
        invocationsThisBlock++;

        // Record in history
        const history = this.executionHistory.get(pending.contract) ?? [];
        history.push(result);
        this.executionHistory.set(pending.contract, history);

        // Emit event
        this.emit({
          type: "self_invoked",
          data: {
            contract: pending.contract,
            invocationId: pending.invocationId,
            result,
          },
        });
      } catch (error) {
        // Log error but continue processing
        this.emit({
          type: "invocation_failed",
          data: {
            contract: pending.contract,
            invocationId: pending.invocationId,
            error: error instanceof Error ? error.message : String(error),
          },
        });
      } finally {
        concurrentCount--;
      }
    }

    return results;
  }

  /**
   * Execute a single invocation
   */
  private async executeInvocation(
    pending: PendingInvocation,
    context: ExecutionContext,
    executor: ContractExecutor,
  ): Promise<InvocationResult> {
    // P0-3 FIX: Enforce recursion depth limits
    if (context.recursionDepth > 0 && !this.config.allowRecursiveSelfInvoke) {
      throw new ContractError(
        `Recursive self-invocation not allowed (depth: ${context.recursionDepth})`,
        "RECURSION_NOT_ALLOWED",
      );
    }
    if (context.recursionDepth >= this.config.maxRecursionDepth) {
      throw new ContractError(
        `Max recursion depth exceeded: ${context.recursionDepth} >= ${this.config.maxRecursionDepth}`,
        "MAX_RECURSION_DEPTH",
      );
    }

    // Get contract
    const contract = this.registry.getContract(pending.contract);
    if (!contract) {
      throw new ContractError(
        `Contract ${pending.contract} not found`,
        "CONTRACT_NOT_FOUND",
      );
    }

    // Get invocation config
    const invocation = contract.invocations.find(
      (i) => i.id === pending.invocationId,
    );
    if (!invocation) {
      throw new ContractError(
        `Invocation ${pending.invocationId} not found`,
        "INVOCATION_NOT_FOUND",
      );
    }

    // Check gas
    const gasNeeded = invocation.gasLimit;
    if (contract.prepaidGas < gasNeeded) {
      this.emit({
        type: "gas_depleted",
        data: { contract: contract.address, remainingGas: contract.prepaidGas },
      });
      throw new ContractError(
        `Insufficient prepaid gas: ${contract.prepaidGas} < ${gasNeeded}`,
        "INSUFFICIENT_GAS",
      );
    }

    // Build contract call
    const call: ContractCall = {
      id: sha512_256Hex(
        `call:${contract.address}:${pending.invocationId}:${context.blockNumber}`,
      ),
      target: contract.address,
      caller: contract.address, // Self-invocation
      functionSelector: invocation.functionSelector,
      callData: invocation.parameters,
      value: invocation.value,
      gas: invocation.gasLimit,
      callType: "self_invoke",
      isSelfInvocation: true,
      scheduleProcessId: pending.scheduleProcessId,
    };

    // Execute via executor
    const result = await executor.execute(call, context);

    // Update contract state
    const updatedContract: SelfInvokingContract = {
      ...contract,
      totalInvocations: contract.totalInvocations + 1,
      prepaidGas: contract.prepaidGas - result.gasUsed,
      lastInvokedAtBlock: context.blockNumber,
    };

    // Update invocation config
    const updatedInvocation: InvocationConfig = {
      ...invocation,
      invocationCount: invocation.invocationCount + 1,
      lastInvokedBlock: context.blockNumber,
      lastResult: result,
    };

    // Check if max invocations reached
    if (
      updatedInvocation.maxInvocations !== undefined &&
      updatedInvocation.invocationCount >= updatedInvocation.maxInvocations
    ) {
      updatedInvocation.enabled = false;
      this.emit({
        type: "invocation_disabled",
        data: {
          contract: contract.address,
          invocationId: invocation.id,
        },
      });
    }

    // Update contract in registry
    updatedContract.invocations = contract.invocations.map((i) =>
      i.id === invocation.id ? updatedInvocation : i,
    );
    this.registry.updateContract(contract.address, updatedContract);

    return result;
  }

  // ============ Priority Calculation ============

  /**
   * Calculate execution priority for an invocation
   */
  private calculatePriority(
    contract: SelfInvokingContract,
    invocation: InvocationConfig,
    currentBlock: bigint,
  ): number {
    // P0-C1 FIX: Use integer-only arithmetic for deterministic priority.
    // All operations scaled to integer basis points to avoid floating-point divergence.
    let priorityBps = 0;

    // Higher gas limit = lower priority (conserve resources)
    // Scale: 1M gas = -100 bps
    priorityBps -= Number(invocation.gasLimit / 10000n);

    // Higher value transfer = higher priority (more important)
    // Scale: 1 LGM (10^18) = +100 bps
    priorityBps += Number(invocation.value / 10n ** 16n);

    // Longer wait = higher priority (fairness)
    // Scale: 1 block = +1 bps
    if (invocation.lastInvokedBlock) {
      const blocksWaiting = Number(currentBlock - invocation.lastInvokedBlock);
      priorityBps += blocksWaiting;
    }

    // PoT-specific: Knowledge domain contracts get boost
    if (contract.knowledgeDomain) {
      priorityBps += 10000;
    }

    return priorityBps;
  }

  // ============ Query Methods ============

  /**
   * Get pending invocations for a contract
   */
  getPendingForContract(address: string): PendingInvocation[] {
    return this.pendingQueue.filter((p) => p.contract === address);
  }

  /**
   * Get execution history for a contract
   */
  getHistory(address: string, limit?: number): InvocationResult[] {
    const history = this.executionHistory.get(address) ?? [];
    return limit ? history.slice(-limit) : history;
  }

  /**
   * Get queue statistics
   */
  getQueueStats(): {
    queueLength: number;
    byContract: Map<string, number>;
    totalPending: number;
  } {
    const byContract = new Map<string, number>();

    for (const pending of this.pendingQueue) {
      const count = byContract.get(pending.contract) ?? 0;
      byContract.set(pending.contract, count + 1);
    }

    return {
      queueLength: this.pendingQueue.length,
      byContract,
      totalPending: this.pendingQueue.length,
    };
  }

  // ============ Event Handling ============

  /**
   * Subscribe to lifecycle events
   */
  onEvent(handler: (event: ContractLifecycleEvent) => void): () => void {
    this.eventEmitters.push(handler);
    return () => {
      const index = this.eventEmitters.indexOf(handler);
      if (index >= 0) {
        this.eventEmitters.splice(index, 1);
      }
    };
  }

  /**
   * Emit a lifecycle event
   */
  private emit(event: ContractLifecycleEvent): void {
    for (const handler of this.eventEmitters) {
      try {
        handler(event);
      } catch {
        // Ignore handler errors
      }
    }
  }

  // ============ State Management ============

  /**
   * Clear pending queue (for testing or reset)
   */
  clearQueue(): void {
    this.pendingQueue = [];
  }

  /**
   * Clear execution history (for testing or reset)
   */
  clearHistory(): void {
    this.executionHistory.clear();
  }
}

/**
 * Contract Executor interface
 * Implemented by BVM to actually execute contract calls
 */
export interface ContractExecutor {
  /**
   * Execute a contract call
   */
  execute(
    call: ContractCall,
    context: ExecutionContext,
  ): Promise<InvocationResult>;

  /**
   * Estimate gas for a call
   */
  estimateGas(call: ContractCall): Promise<bigint>;

  /**
   * Check if call would succeed (dry run)
   */
  dryRun(
    call: ContractCall,
    context: ExecutionContext,
  ): Promise<InvocationResult>;
}

/**
 * Mock executor for testing
 */
export class MockContractExecutor implements ContractExecutor {
  async execute(
    call: ContractCall,
    context: ExecutionContext,
  ): Promise<InvocationResult> {
    // Simulate execution
    const gasUsed = call.gas / 2n; // Use half the gas limit

    return {
      success: true,
      gasUsed,
      events: [],
      stateChanges: [],
      executedAtBlock: context.blockNumber,
      transactionId: sha512_256Hex(`tx:${call.id}:${context.blockNumber}`),
    };
  }

  async estimateGas(call: ContractCall): Promise<bigint> {
    return call.gas / 2n;
  }

  async dryRun(
    call: ContractCall,
    context: ExecutionContext,
  ): Promise<InvocationResult> {
    return this.execute(call, context);
  }
}
