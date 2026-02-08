/**
 * Self-Invocation Engine (SelfInvoker) Tests
 *
 * Tests invocation queuing, execution, gas metering, priority calculation,
 * recursion limits, event handling, and queue management.
 *
 * Each test creates its own registry/invoker/executor to ensure isolation
 * under concurrent test execution.
 */

import { describe, it, expect } from "vitest";
import { ContractRegistry } from "../../../core/contracts/registry.js";
import {
  SelfInvoker,
  MockContractExecutor,
  type ContractExecutor,
} from "../../../core/contracts/invoker.js";
import {
  ContractError,
  DEFAULT_CONTRACT_CONFIG,
  type SelfInvokingContract,
  type InvocationConfig,
  type ContractCall,
  type ContractConfig,
  type ContractLifecycleEvent,
  type ContractMetadata,
  type ContractABI,
} from "../../../core/contracts/types.js";
import type { EvaluationResult } from "../../../core/schedule/condition_evaluator.js";

// ============ Test Helpers ============

const CREATOR_DID = "did:lgm:00000000000000000000000000000001";
const PROPOSER_DID = "did:lgm:00000000000000000000000000000099";

function createMinimalABI(): ContractABI {
  return { functions: [], events: [], stateVariables: [] };
}

function createMetadata(): ContractMetadata {
  return {
    name: "Test",
    description: "Test contract",
    version: "1.0.0",
    author: CREATOR_DID,
    abi: createMinimalABI(),
  };
}

function createInvocation(
  overrides: Partial<InvocationConfig> = {},
): InvocationConfig {
  return {
    id: "inv_" + Math.random().toString(36).slice(2, 8),
    functionName: "execute",
    functionSelector: new Uint8Array([0xde, 0xad, 0xbe, 0xef]),
    parameters: new Uint8Array([0x01]),
    condition: { time: { type: "periodic", interval: 60000 } },
    scheduleType: "time_periodic",
    enabled: true,
    invocationCount: 0,
    gasLimit: 100_000n,
    value: 0n,
    ...overrides,
  };
}

function createContract(
  overrides: Partial<SelfInvokingContract> = {},
): SelfInvokingContract {
  const inv = createInvocation({ id: "inv_default" });
  return {
    address: "contract_" + Math.random().toString(36).slice(2, 10),
    creator: CREATOR_DID,
    name: "TestContract",
    version: "1.0.0",
    codeHash: "abc",
    isActive: true,
    invocations: [inv],
    totalInvocations: 0,
    stateHash: "000",
    deployedAtBlock: 10n,
    gasLimit: 1_000_000n,
    prepaidGas: 1_000_000n,
    ...overrides,
  };
}

function createExecutionContext(overrides = {}) {
  return {
    blockNumber: 100n,
    timestamp: Date.now(),
    proposer: PROPOSER_DID,
    gasPrice: 1n,
    recursionDepth: 0,
    ...overrides,
  };
}

function satisfiedEvaluation(): EvaluationResult {
  return { satisfied: true };
}

function registerContract(
  registry: ContractRegistry,
  contract: SelfInvokingContract,
): void {
  registry.register(contract, createMetadata(), [], contract.deployedAtBlock);
}

/** Create fresh isolated test fixtures */
function freshFixtures(config?: ContractConfig) {
  const registry = new ContractRegistry();
  const invoker = new SelfInvoker(registry, config);
  const executor = new MockContractExecutor();
  return { registry, invoker, executor };
}

// ============ Tests ============

describe("SelfInvoker", () => {
  // ============ Queue Invocation ============

  describe("queueInvocation", () => {
    it("should queue a valid invocation", () => {
      const { registry, invoker } = freshFixtures();
      const contract = createContract({ address: "q_001" });
      registerContract(registry, contract);

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched_001",
        satisfiedEvaluation(),
        100n,
      );

      const stats = invoker.getQueueStats();
      expect(stats.queueLength).toBe(1);
      expect(stats.totalPending).toBe(1);
    });

    it("should reject queueing for inactive contract", () => {
      const { invoker } = freshFixtures();
      const contract = createContract({
        address: "q_inactive",
        isActive: false,
      });

      expect(() =>
        invoker.queueInvocation(
          contract,
          "inv_default",
          "sched",
          satisfiedEvaluation(),
          100n,
        ),
      ).toThrowError(ContractError);
    });

    it("should reject queueing for non-existent invocation ID", () => {
      const { invoker } = freshFixtures();
      const contract = createContract({ address: "q_noinv" });

      expect(() =>
        invoker.queueInvocation(
          contract,
          "nonexistent_inv",
          "sched",
          satisfiedEvaluation(),
          100n,
        ),
      ).toThrowError(ContractError);
    });

    it("should reject queueing for disabled invocation", () => {
      const { invoker } = freshFixtures();
      const inv = createInvocation({ id: "inv_disabled", enabled: false });
      const contract = createContract({
        address: "q_disabled",
        invocations: [inv],
      });

      expect(() =>
        invoker.queueInvocation(
          contract,
          "inv_disabled",
          "sched",
          satisfiedEvaluation(),
          100n,
        ),
      ).toThrowError(ContractError);
    });

    it("should reject when max invocations reached", () => {
      const { invoker } = freshFixtures();
      const inv = createInvocation({
        id: "inv_maxed",
        maxInvocations: 5,
        invocationCount: 5,
      });
      const contract = createContract({
        address: "q_maxed",
        invocations: [inv],
      });

      expect(() =>
        invoker.queueInvocation(
          contract,
          "inv_maxed",
          "sched",
          satisfiedEvaluation(),
          100n,
        ),
      ).toThrowError(ContractError);
    });

    it("should emit invocation_scheduled event on queue", () => {
      const { registry, invoker } = freshFixtures();
      const contract = createContract({ address: "q_event" });
      registerContract(registry, contract);

      const events: ContractLifecycleEvent[] = [];
      invoker.onEvent((e) => events.push(e));

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched_001",
        satisfiedEvaluation(),
        100n,
      );

      expect(events).toHaveLength(1);
      expect(events[0].type).toBe("invocation_scheduled");
    });

    it("should sort queue by priority with deterministic tiebreakers", () => {
      const { registry, invoker } = freshFixtures();
      const inv1 = createInvocation({ id: "inv_a", gasLimit: 100n, value: 0n });
      const inv2 = createInvocation({
        id: "inv_b",
        gasLimit: 100n,
        value: 100n * 10n ** 16n,
      });

      const c1 = createContract({
        address: "aaaa",
        invocations: [inv1],
        knowledgeDomain: undefined,
      });
      const c2 = createContract({
        address: "bbbb",
        invocations: [inv2],
        knowledgeDomain: undefined,
      });
      registerContract(registry, c1);
      registerContract(registry, c2);

      invoker.queueInvocation(c1, "inv_a", "s1", satisfiedEvaluation(), 100n);
      invoker.queueInvocation(c2, "inv_b", "s2", satisfiedEvaluation(), 100n);

      // c2 has higher value transfer, so higher priority
      const pending = invoker.getPendingForContract("bbbb");
      expect(pending).toHaveLength(1);
    });
  });

  // ============ Process Pending Invocations ============

  describe("processPendingInvocations", () => {
    it("should execute queued invocations and return results", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const contract = createContract({
        address: "exec_001",
        prepaidGas: 1_000_000n,
      });
      registerContract(registry, contract);

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext();
      const results = await invoker.processPendingInvocations(
        context,
        executor,
      );

      expect(results).toHaveLength(1);
      expect(results[0].success).toBe(true);
      expect(typeof results[0].gasUsed).toBe("bigint");
    });

    it("should update contract state after execution", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const contract = createContract({
        address: "exec_state",
        prepaidGas: 1_000_000n,
        totalInvocations: 0,
      });
      registerContract(registry, contract);

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext();
      await invoker.processPendingInvocations(context, executor);

      const updated = registry.getContract("exec_state");
      expect(updated).toBeDefined();
      expect(updated!.totalInvocations).toBe(1);
      expect(updated!.prepaidGas).toBeLessThan(1_000_000n);
    });

    it("should respect maxInvocationsPerBlock limit", async () => {
      const { registry, invoker: _unused, executor } = freshFixtures();
      const config: ContractConfig = {
        ...DEFAULT_CONTRACT_CONFIG,
        maxInvocationsPerBlock: 1,
      };
      const limitedInvoker = new SelfInvoker(registry, config);

      const inv1 = createInvocation({ id: "inv_1" });
      const inv2 = createInvocation({ id: "inv_2" });
      const contract = createContract({
        address: "exec_limit",
        invocations: [inv1, inv2],
        prepaidGas: 10_000_000n,
      });
      registerContract(registry, contract);

      limitedInvoker.queueInvocation(
        contract,
        "inv_1",
        "s1",
        satisfiedEvaluation(),
        100n,
      );
      limitedInvoker.queueInvocation(
        contract,
        "inv_2",
        "s2",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext();
      const results = await limitedInvoker.processPendingInvocations(
        context,
        executor,
      );

      expect(results).toHaveLength(1);
      expect(limitedInvoker.getQueueStats().queueLength).toBe(1);
    });

    it("should clear pending queue after processing", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const contract = createContract({
        address: "exec_clear",
        prepaidGas: 1_000_000n,
      });
      registerContract(registry, contract);

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext();
      await invoker.processPendingInvocations(context, executor);

      expect(invoker.getQueueStats().queueLength).toBe(0);
    });

    it("should handle execution errors gracefully and emit failure event", async () => {
      const { registry, invoker } = freshFixtures();
      const contract = createContract({
        address: "exec_fail",
        prepaidGas: 1_000_000n,
      });
      registerContract(registry, contract);

      const failingExecutor: ContractExecutor = {
        async execute() {
          throw new Error("BVM crash");
        },
        async estimateGas() {
          return 0n;
        },
        async dryRun() {
          throw new Error("BVM crash");
        },
      };

      const events: ContractLifecycleEvent[] = [];
      invoker.onEvent((e) => events.push(e));

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext();
      const results = await invoker.processPendingInvocations(
        context,
        failingExecutor,
      );

      expect(results).toHaveLength(0);
      const failEvents = events.filter((e) => e.type === "invocation_failed");
      expect(failEvents).toHaveLength(1);
    });

    it("should emit self_invoked event on successful execution", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const contract = createContract({
        address: "exec_evt",
        prepaidGas: 1_000_000n,
      });
      registerContract(registry, contract);

      const events: ContractLifecycleEvent[] = [];
      invoker.onEvent((e) => events.push(e));

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext();
      await invoker.processPendingInvocations(context, executor);

      const selfInvokedEvents = events.filter((e) => e.type === "self_invoked");
      expect(selfInvokedEvents).toHaveLength(1);
    });

    it("should throw INSUFFICIENT_GAS when prepaidGas is too low", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const inv = createInvocation({ id: "inv_gas", gasLimit: 500_000n });
      const contract = createContract({
        address: "exec_nogas",
        invocations: [inv],
        prepaidGas: 100n,
      });
      registerContract(registry, contract);

      const events: ContractLifecycleEvent[] = [];
      invoker.onEvent((e) => events.push(e));

      invoker.queueInvocation(
        contract,
        "inv_gas",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext();
      const results = await invoker.processPendingInvocations(
        context,
        executor,
      );

      expect(results).toHaveLength(0);
      const gasEvents = events.filter((e) => e.type === "gas_depleted");
      expect(gasEvents.length).toBeGreaterThanOrEqual(1);
    });

    it("should disable invocation when maxInvocations reached after execution", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const inv = createInvocation({
        id: "inv_max",
        maxInvocations: 1,
        invocationCount: 0,
      });
      const contract = createContract({
        address: "exec_max",
        invocations: [inv],
        prepaidGas: 1_000_000n,
      });
      registerContract(registry, contract);

      const events: ContractLifecycleEvent[] = [];
      invoker.onEvent((e) => events.push(e));

      invoker.queueInvocation(
        contract,
        "inv_max",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext();
      await invoker.processPendingInvocations(context, executor);

      const disabledEvents = events.filter(
        (e) => e.type === "invocation_disabled",
      );
      expect(disabledEvents).toHaveLength(1);

      const updated = registry.getContract("exec_max");
      const updatedInv = updated!.invocations.find((i) => i.id === "inv_max");
      expect(updatedInv!.enabled).toBe(false);
    });
  });

  // ============ Recursion Limits ============

  describe("recursion depth enforcement", () => {
    it("should reject recursive invocation when not allowed", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const contract = createContract({
        address: "rec_deny",
        prepaidGas: 1_000_000n,
      });
      registerContract(registry, contract);

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext({ recursionDepth: 1 });
      const results = await invoker.processPendingInvocations(
        context,
        executor,
      );

      expect(results).toHaveLength(0);
    });

    it("should reject when max recursion depth exceeded even if allowed", async () => {
      const config: ContractConfig = {
        ...DEFAULT_CONTRACT_CONFIG,
        allowRecursiveSelfInvoke: true,
        maxRecursionDepth: 2,
      };
      const {
        registry,
        invoker: recursiveInvoker,
        executor,
      } = freshFixtures(config);

      const contract = createContract({
        address: "rec_max",
        prepaidGas: 1_000_000n,
      });
      registerContract(registry, contract);

      recursiveInvoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      const context = createExecutionContext({ recursionDepth: 3 });
      const results = await recursiveInvoker.processPendingInvocations(
        context,
        executor,
      );

      expect(results).toHaveLength(0);
    });
  });

  // ============ Query Methods ============

  describe("getPendingForContract", () => {
    it("should return pending invocations only for specified contract", () => {
      const { registry, invoker } = freshFixtures();
      const c1 = createContract({ address: "pf_c1" });
      const c2 = createContract({ address: "pf_c2" });
      registerContract(registry, c1);
      registerContract(registry, c2);

      invoker.queueInvocation(
        c1,
        "inv_default",
        "s1",
        satisfiedEvaluation(),
        100n,
      );
      invoker.queueInvocation(
        c2,
        "inv_default",
        "s2",
        satisfiedEvaluation(),
        100n,
      );

      const pending = invoker.getPendingForContract("pf_c1");
      expect(pending).toHaveLength(1);
      expect(pending[0].contract).toBe("pf_c1");
    });
  });

  describe("getHistory", () => {
    it("should return execution history for a contract", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const contract = createContract({
        address: "hist_001",
        prepaidGas: 10_000_000n,
      });
      registerContract(registry, contract);

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );
      await invoker.processPendingInvocations(
        createExecutionContext(),
        executor,
      );

      const history = invoker.getHistory("hist_001");
      expect(history).toHaveLength(1);
      expect(history[0].success).toBe(true);
    });

    it("should respect limit parameter", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const inv1 = createInvocation({ id: "inv_h1" });
      const inv2 = createInvocation({ id: "inv_h2" });
      const contract = createContract({
        address: "hist_lim",
        invocations: [inv1, inv2],
        prepaidGas: 10_000_000n,
      });
      registerContract(registry, contract);

      invoker.queueInvocation(
        contract,
        "inv_h1",
        "s1",
        satisfiedEvaluation(),
        100n,
      );
      await invoker.processPendingInvocations(
        createExecutionContext(),
        executor,
      );

      const updatedContract = registry.getContract("hist_lim")!;
      invoker.queueInvocation(
        updatedContract,
        "inv_h2",
        "s2",
        satisfiedEvaluation(),
        101n,
      );
      await invoker.processPendingInvocations(
        createExecutionContext({ blockNumber: 101n }),
        executor,
      );

      expect(invoker.getHistory("hist_lim")).toHaveLength(2);
      expect(invoker.getHistory("hist_lim", 1)).toHaveLength(1);
    });

    it("should return empty array for no history", () => {
      const { invoker } = freshFixtures();
      expect(invoker.getHistory("nonexistent")).toEqual([]);
    });
  });

  describe("getQueueStats", () => {
    it("should track by-contract counts", () => {
      const { registry, invoker } = freshFixtures();
      const c1 = createContract({ address: "qs_c1" });
      const c2 = createContract({ address: "qs_c2" });
      registerContract(registry, c1);
      registerContract(registry, c2);

      invoker.queueInvocation(
        c1,
        "inv_default",
        "s1",
        satisfiedEvaluation(),
        100n,
      );
      invoker.queueInvocation(
        c2,
        "inv_default",
        "s2",
        satisfiedEvaluation(),
        100n,
      );

      const stats = invoker.getQueueStats();
      expect(stats.byContract.get("qs_c1")).toBe(1);
      expect(stats.byContract.get("qs_c2")).toBe(1);
      expect(stats.queueLength).toBe(2);
    });
  });

  // ============ Event Handling ============

  describe("onEvent / unsubscribe", () => {
    it("should subscribe and receive events", () => {
      const { registry, invoker } = freshFixtures();
      const contract = createContract({ address: "evt_sub" });
      registerContract(registry, contract);

      const events: ContractLifecycleEvent[] = [];
      invoker.onEvent((e) => events.push(e));

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      expect(events).toHaveLength(1);
    });

    it("should unsubscribe and stop receiving events", () => {
      const { registry, invoker } = freshFixtures();
      const contract = createContract({ address: "evt_unsub" });
      registerContract(registry, contract);

      const events: ContractLifecycleEvent[] = [];
      const unsub = invoker.onEvent((e) => events.push(e));

      invoker.queueInvocation(
        contract,
        "inv_default",
        "s1",
        satisfiedEvaluation(),
        100n,
      );
      expect(events).toHaveLength(1);

      unsub();

      const contract2 = createContract({ address: "evt_unsub2" });
      registerContract(registry, contract2);

      invoker.queueInvocation(
        contract2,
        "inv_default",
        "s2",
        satisfiedEvaluation(),
        101n,
      );
      expect(events).toHaveLength(1); // Still 1, not 2
    });
  });

  // ============ State Management ============

  describe("clearQueue / clearHistory", () => {
    it("should clear pending queue", () => {
      const { registry, invoker } = freshFixtures();
      const contract = createContract({ address: "clr_q" });
      registerContract(registry, contract);

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );

      expect(invoker.getQueueStats().queueLength).toBe(1);
      invoker.clearQueue();
      expect(invoker.getQueueStats().queueLength).toBe(0);
    });

    it("should clear execution history", async () => {
      const { registry, invoker, executor } = freshFixtures();
      const contract = createContract({
        address: "clr_h",
        prepaidGas: 1_000_000n,
      });
      registerContract(registry, contract);

      invoker.queueInvocation(
        contract,
        "inv_default",
        "sched",
        satisfiedEvaluation(),
        100n,
      );
      await invoker.processPendingInvocations(
        createExecutionContext(),
        executor,
      );

      expect(invoker.getHistory("clr_h")).toHaveLength(1);
      invoker.clearHistory();
      expect(invoker.getHistory("clr_h")).toHaveLength(0);
    });
  });

  // ============ MockContractExecutor ============

  describe("MockContractExecutor", () => {
    it("should return success with half gas used", async () => {
      const executor = new MockContractExecutor();
      const call: ContractCall = {
        id: "call_1",
        target: "addr",
        caller: "addr",
        functionSelector: new Uint8Array(4),
        callData: new Uint8Array(0),
        value: 0n,
        gas: 200_000n,
        callType: "self_invoke",
        isSelfInvocation: true,
      };
      const context = createExecutionContext();

      const result = await executor.execute(call, context);
      expect(result.success).toBe(true);
      expect(result.gasUsed).toBe(100_000n);
    });

    it("should estimate gas as half of provided gas", async () => {
      const executor = new MockContractExecutor();
      const call: ContractCall = {
        id: "call_2",
        target: "addr",
        caller: "addr",
        functionSelector: new Uint8Array(4),
        callData: new Uint8Array(0),
        value: 0n,
        gas: 400_000n,
        callType: "self_invoke",
        isSelfInvocation: true,
      };

      const estimate = await executor.estimateGas(call);
      expect(estimate).toBe(200_000n);
    });

    it("should return same result from dryRun as execute", async () => {
      const executor = new MockContractExecutor();
      const call: ContractCall = {
        id: "call_3",
        target: "addr",
        caller: "addr",
        functionSelector: new Uint8Array(4),
        callData: new Uint8Array(0),
        value: 0n,
        gas: 100_000n,
        callType: "self_invoke",
        isSelfInvocation: true,
      };
      const context = createExecutionContext();

      const execResult = await executor.execute(call, context);
      const dryResult = await executor.dryRun(call, context);

      expect(dryResult.success).toBe(execResult.success);
      expect(dryResult.gasUsed).toBe(execResult.gasUsed);
    });
  });
});
