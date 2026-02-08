/**
 * BVM Service — orchestration layer for the Blockchain Virtual Machine
 *
 * Bridges BVMStateStore, ExecutionEngine, and the knowledge layer into
 * a single service that can be wired into the node's state transitions.
 *
 * Key design: ExecutionEngine.execute() is synchronous, but BVMStateStore
 * is async. We solve this with an in-memory code cache populated at deploy
 * time or via ensureLoaded(). Storage writes are buffered in the store's
 * pending changes and flushed on commitChanges().
 */

import { BVMStateStore } from "../../storage/bvm_store.js";
import { ExecutionEngine } from "./execution.js";
import { createInitialState, createExecutionContext } from "./index.js";
import type {
  BVMKnowledgeService,
  ExecutionResult,
  BVMConfig,
  BVMState,
  StateChange as BVMStateChange,
} from "./types.js";
import { DEFAULT_BVM_CONFIG, NullKnowledgeService } from "./types.js";
import type {
  ContractCall,
  InvocationResult,
  ContractEvent,
  StateChange as ContractStateChange,
} from "../contracts/types.js";
import type { ContractExecutor } from "../contracts/invoker.js";

// ============ BVM Service ============

export class BVMService implements ContractExecutor {
  private codeCache: Map<string, Uint8Array> = new Map();
  private config: BVMConfig;

  constructor(
    private bvmStore: BVMStateStore,
    config?: Partial<BVMConfig>,
    private knowledgeService: BVMKnowledgeService = new NullKnowledgeService(),
  ) {
    this.config = { ...DEFAULT_BVM_CONFIG, ...config };
  }

  // ============ Contract Lifecycle ============

  /**
   * Deploy contract bytecode to storage and cache
   */
  async deployContract(address: string, bytecode: Uint8Array): Promise<void> {
    await this.bvmStore.saveContract(address, bytecode);
    this.codeCache.set(address, bytecode);
  }

  /**
   * Get contract bytecode (from cache or storage)
   */
  async getContractCode(address: string): Promise<Uint8Array | null> {
    const cached = this.codeCache.get(address);
    if (cached) return cached;

    const code = await this.bvmStore.loadContract(address);
    if (code) {
      this.codeCache.set(address, code);
    }
    return code;
  }

  /**
   * Ensure contract code is in cache (call before sync execution)
   */
  async ensureLoaded(address: string): Promise<boolean> {
    if (this.codeCache.has(address)) return true;
    const code = await this.bvmStore.loadContract(address);
    if (code) {
      this.codeCache.set(address, code);
      return true;
    }
    return false;
  }

  // ============ Synchronous Execution (for state transitions) ============

  /**
   * Execute a contract call synchronously using cached bytecode.
   *
   * The BVM ExecutionEngine.execute() is synchronous. Bytecode must be
   * pre-loaded into the code cache via deployContract() or ensureLoaded().
   */
  callContractSync(
    target: string,
    caller: string,
    callData: Uint8Array,
    opts: {
      blockNumber: bigint;
      callerDID?: string;
      value?: bigint;
      gasLimit?: bigint;
      txHash?: string;
    },
  ): ExecutionResult {
    const bytecode = this.codeCache.get(target);
    if (!bytecode) {
      return {
        success: false,
        returnData: new Uint8Array(0),
        gasUsed: 0n,
        logs: [],
        stateChanges: [],
        error: {
          code: "EXECUTION_FAILED",
          message: `Contract ${target} not loaded in cache`,
        },
      };
    }

    // Create fresh BVM state for this execution
    const bvmState: BVMState = createInitialState();
    bvmState.blockNumber = opts.blockNumber;
    bvmState.timestamp = Date.now();

    // Create execution engine with knowledge service
    const engine = new ExecutionEngine(
      bvmState,
      this.config,
      this.knowledgeService,
    );

    // Create execution context
    const context = createExecutionContext({
      caller,
      callerDID: opts.callerDID,
      currentContract: target,
      blockNumber: opts.blockNumber,
      callData,
      value: opts.value ?? 0n,
      gasLimit: opts.gasLimit ?? this.config.blockGasLimit,
      transactionHash: opts.txHash ?? "",
      mode: "call",
    });

    // Execute bytecode
    const result = engine.execute(bytecode, context);

    // Buffer storage changes for later commit
    this.bufferStorageChanges(target, result.stateChanges);

    return result;
  }

  // ============ Async Execution (for RPC / direct calls) ============

  /**
   * Execute a contract call asynchronously (loads bytecode if needed)
   */
  async callContract(
    target: string,
    caller: string,
    callData: Uint8Array,
    opts: {
      blockNumber: bigint;
      callerDID?: string;
      value?: bigint;
      gasLimit?: bigint;
      txHash?: string;
    },
  ): Promise<ExecutionResult> {
    // Ensure bytecode is cached
    const loaded = await this.ensureLoaded(target);
    if (!loaded) {
      return {
        success: false,
        returnData: new Uint8Array(0),
        gasUsed: 0n,
        logs: [],
        stateChanges: [],
        error: {
          code: "EXECUTION_FAILED",
          message: `Contract ${target} not found`,
        },
      };
    }

    return this.callContractSync(target, caller, callData, opts);
  }

  // ============ ContractExecutor Interface ============

  /**
   * Execute a ContractCall (used by self-invoking contracts)
   */
  async execute(
    call: ContractCall,
    _context: {
      blockNumber: bigint;
      timestamp: number;
      proposer: string;
      gasPrice: bigint;
    },
  ): Promise<InvocationResult> {
    const result = await this.callContract(
      call.target,
      call.caller,
      call.callData,
      {
        blockNumber: _context.blockNumber,
        value: call.value,
        gasLimit: call.gas,
      },
    );

    // Map BVM logs to ContractEvent format
    const events: ContractEvent[] = result.logs.map((log, i) => ({
      contract: log.address,
      signature:
        log.topics.length > 0 ? Buffer.from(log.topics[0]).toString("hex") : "",
      topics: log.topics,
      data: log.data,
      logIndex: i,
      blockNumber: _context.blockNumber,
      transactionId: call.id,
    }));

    // Map BVM state changes to contracts/types StateChange format
    const stateChanges: ContractStateChange[] = result.stateChanges
      .filter((sc) => sc.type === "storage")
      .map((sc) => ({
        contract: sc.address,
        key: sc.key ? new TextEncoder().encode(sc.key) : new Uint8Array(0),
        previousValue:
          sc.oldValue instanceof Uint8Array ? sc.oldValue : undefined,
        newValue:
          sc.newValue instanceof Uint8Array
            ? (sc.newValue as Uint8Array)
            : new Uint8Array(0),
      }));

    return {
      success: result.success,
      returnData: result.returnData,
      gasUsed: result.gasUsed,
      error: result.error?.message,
      events,
      stateChanges,
      executedAtBlock: _context.blockNumber,
      transactionId: call.id,
    };
  }

  /**
   * Estimate gas for a contract call
   */
  async estimateGas(call: ContractCall): Promise<bigint> {
    const result = await this.callContract(
      call.target,
      call.caller,
      call.callData,
      {
        blockNumber: 0n,
        value: call.value,
        gasLimit: call.gas,
      },
    );
    // Add 20% buffer for estimation
    return result.gasUsed + result.gasUsed / 5n;
  }

  /**
   * Dry run a contract call (no state persistence)
   */
  async dryRun(
    call: ContractCall,
    context: {
      blockNumber: bigint;
      timestamp: number;
      proposer: string;
      gasPrice: bigint;
    },
  ): Promise<InvocationResult> {
    // Execute but discard any buffered storage changes
    const pendingBefore = this.bvmStore.getPendingOperationCount();
    const result = await this.execute(call, context);
    // Rollback if we added changes
    if (
      this.bvmStore.hasActiveTransaction() &&
      this.bvmStore.getPendingOperationCount() > pendingBefore
    ) {
      await this.bvmStore.rollback();
    }
    return result;
  }

  // ============ Persistence ============

  /**
   * Commit all buffered storage changes
   */
  async commitChanges(): Promise<void> {
    if (this.bvmStore.hasActiveTransaction()) {
      await this.bvmStore.commit();
    }
  }

  /**
   * Rollback all buffered storage changes
   */
  async rollbackChanges(): Promise<void> {
    if (this.bvmStore.hasActiveTransaction()) {
      await this.bvmStore.rollback();
    }
  }

  // ============ Internal ============

  /**
   * Buffer storage changes from execution result into the BVM store
   */
  private bufferStorageChanges(
    contractAddress: string,
    stateChanges: BVMStateChange[],
  ): void {
    for (const change of stateChanges) {
      if (change.type === "storage" && change.key && change.newValue) {
        const slotKey = new TextEncoder().encode(change.key);
        const value =
          change.newValue instanceof Uint8Array
            ? change.newValue
            : new TextEncoder().encode(String(change.newValue));
        // Fire-and-forget — the store buffers these as pending ops
        void this.bvmStore.saveStorage(contractAddress, slotKey, value);
      }
    }
  }
}
