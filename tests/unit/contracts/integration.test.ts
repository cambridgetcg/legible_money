/**
 * DMSS-Contract Integration Tests
 *
 * Tests the bridge between DMSS Schedule module and Self-Invoking Contracts.
 * Covers deployment, schedule handling, invocation management, gas management,
 * and PoT-specific integration features.
 */

import { describe, it, expect } from "vitest";
import { ContractRegistry } from "../../../core/contracts/registry.js";
import {
  SelfInvoker,
  MockContractExecutor,
} from "../../../core/contracts/invoker.js";
import { ContractIntegration } from "../../../core/contracts/integration.js";
import {
  ContractError,
  DEFAULT_CONTRACT_CONFIG,
  type SelfInvokingContract,
  type InvocationConfig,
  type ContractMetadata,
  type ContractABI,
  type ContractConfig,
} from "../../../core/contracts/types.js";
import type {
  ScheduleMessage,
  ScheduleProcess,
  ScheduleCondition,
} from "../../../core/schedule/types.js";
import type { ScheduleProcessingUnit } from "../../../core/schedule/processing_unit.js";
import type { Signature } from "../../../core/crypto/keys.js";

// ============ Test Helpers ============

const CREATOR_DID = "did:lgm:00000000000000000000000000000001";
const OTHER_DID = "did:lgm:00000000000000000000000000000002";

function createMinimalABI(
  functions: ContractABI["functions"] = [],
): ContractABI {
  return { functions, events: [], stateVariables: [] };
}

function createMetadata(
  overrides: Partial<ContractMetadata> = {},
): ContractMetadata {
  return {
    name: "TestContract",
    description: "Integration test contract",
    version: "1.0.0",
    author: CREATOR_DID,
    abi: createMinimalABI(),
    ...overrides,
  };
}

/**
 * Creates a mock SPU that returns a ScheduleProcess from processScheduleMessage
 * and tracks calls to modifyScheduleProcess.
 */
function createMockSPU(): {
  spu: ScheduleProcessingUnit;
  processedMessages: ScheduleMessage[];
  modifications: any[];
} {
  const processedMessages: ScheduleMessage[] = [];
  const modifications: any[] = [];

  let processCounter = 0;

  const spu = {
    async processScheduleMessage(
      message: ScheduleMessage,
    ): Promise<ScheduleProcess> {
      processedMessages.push(message);
      processCounter++;
      return {
        id: `sched_proc_${processCounter}`,
        sourceMessageId: message.messageId,
        creator: message.senderAddress,
        condition: message.scheduleCondition,
        watchedStateKeys: [],
        status: "active" as const,
        executionCount: 0,
        maxExecutions: message.maxExecutions,
        remainingFee: message.prepaidFee,
        feePerExecution: message.networkFeeLimit,
        targetAddress: message.recipientAddress,
        callData: message.data,
        transferValue: message.transferValue,
        generatedTxNonce: 0n,
        createdAt: message.timestamp,
        modifiedAt: message.timestamp,
      };
    },
    async modifyScheduleProcess(modification: any): Promise<void> {
      modifications.push(modification);
    },
  } as unknown as ScheduleProcessingUnit;

  return { spu, processedMessages, modifications };
}

// ============ Fixture Factory ============

function freshFixtures() {
  const registry = new ContractRegistry();
  const invoker = new SelfInvoker(registry);
  const executor = new MockContractExecutor();
  const mockSPU = createMockSPU();
  const integration = new ContractIntegration(registry, invoker, mockSPU.spu);
  return { registry, invoker, executor, mockSPU, integration };
}

// ============ Tests ============

describe("ContractIntegration", () => {
  // ============ Contract Deployment ============

  describe("deployContract", () => {
    it("should deploy a contract and register it in the registry", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const result = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01, 0x02, 0x03]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 500_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      expect(result.address).toBeDefined();
      expect(result.address).toHaveLength(42);
      expect(result.contract.isActive).toBe(true);
      expect(result.contract.creator).toBe(CREATOR_DID);
      expect(result.deployedAtBlock).toBe(100n);
      expect(typeof result.gasUsed).toBe("bigint");
      expect(result.gasUsed).toBeGreaterThan(0n);

      // Verify contract was registered
      const retrieved = registry.getContract(result.address);
      expect(retrieved).toBeDefined();
      expect(retrieved!.name).toBe("TestContract");
    });

    it("should register schedule processes for each invocation", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const result = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [
            {
              functionName: "autoExecute",
              functionSelector: new Uint8Array([0xaa, 0xbb, 0xcc, 0xdd]),
              parameters: new Uint8Array([0x01]),
              condition: { time: { type: "periodic", interval: 60000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 100_000n,
              value: 0n,
            },
            {
              functionName: "autoCheck",
              functionSelector: new Uint8Array([0x11, 0x22, 0x33, 0x44]),
              parameters: new Uint8Array([]),
              condition: {
                time: { type: "absolute", executeAt: Date.now() + 3600000 },
              },
              scheduleType: "time_absolute",
              enabled: true,
              gasLimit: 50_000n,
              value: 0n,
            },
          ],
          prepaidGas: 1_000_000n,
          signature: "" as Signature,
        },
        200n,
        Date.now(),
      );

      // Two invocations should produce two schedule processes
      expect(result.scheduleProcessIds).toHaveLength(2);
      expect(mockSPU.processedMessages).toHaveLength(2);

      // Contract should have two invocations with schedule process IDs
      const contract = registry.getContract(result.address);
      expect(contract!.invocations).toHaveLength(2);
      expect(contract!.invocations[0].scheduleProcessId).toBeDefined();
      expect(contract!.invocations[1].scheduleProcessId).toBeDefined();
    });

    it("should use deterministic nonce from block number", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const result1 = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 100_000n,
          signature: "" as Signature,
        },
        42n,
        1000,
      );

      // Deploy again with same block number should get same address
      // (since nonce = currentBlock and same deployer)
      // But need separate registry since first is already registered
      const registry2 = new ContractRegistry();
      const invoker2 = new SelfInvoker(registry2);
      const mockSPU2 = createMockSPU();
      const integration2 = new ContractIntegration(
        registry2,
        invoker2,
        mockSPU2.spu,
      );

      const result2 = await integration2.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 100_000n,
          signature: "" as Signature,
        },
        42n,
        2000, // Different time, same block
      );

      expect(result1.address).toBe(result2.address);
    });

    it("should calculate gas based on bytecode size and invocation count", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      // Deploy with 64 bytes of bytecode and 2 invocations
      const result = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array(64), // 2 words of 32 bytes
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [
            {
              functionName: "fn1",
              functionSelector: new Uint8Array(4),
              parameters: new Uint8Array(0),
              condition: { time: { type: "periodic", interval: 1000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 50_000n,
              value: 0n,
            },
            {
              functionName: "fn2",
              functionSelector: new Uint8Array(4),
              parameters: new Uint8Array(0),
              condition: { time: { type: "periodic", interval: 2000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 50_000n,
              value: 0n,
            },
          ],
          prepaidGas: 1_000_000n,
          signature: "" as Signature,
        },
        1n,
        Date.now(),
      );

      // Gas = 500000 (base) + 2 * 50000 (invocations) + 2 * 20000 (storage for 2 words)
      const expectedGas = 500000n + 2n * 50000n + 2n * 20000n;
      expect(result.gasUsed).toBe(expectedGas);
    });
  });

  // ============ Schedule Execution Handling ============

  describe("handleScheduleExecution", () => {
    it("should execute contract invocation when schedule triggers", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      // Deploy a contract first
      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [
            {
              functionName: "autoFn",
              functionSelector: new Uint8Array([0xab, 0xcd, 0xef, 0x01]),
              parameters: new Uint8Array([0x42]),
              condition: { time: { type: "periodic", interval: 60000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 100_000n,
              value: 0n,
            },
          ],
          prepaidGas: 1_000_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      // Simulate SPU calling back with a schedule process
      const scheduleProcessId = deployResult.scheduleProcessIds[0];
      const process: ScheduleProcess = {
        id: scheduleProcessId,
        sourceMessageId: "msg_1",
        creator: CREATOR_DID,
        condition: { time: { type: "periodic", interval: 60000 } },
        watchedStateKeys: [],
        status: "active",
        executionCount: 0,
        remainingFee: 1_000_000n,
        feePerExecution: 100_000n,
        targetAddress: deployResult.address,
        callData: new Uint8Array([]),
        transferValue: 0n,
        generatedTxNonce: 0n,
        createdAt: Date.now(),
        modifiedAt: Date.now(),
      };

      const result = await integration.handleScheduleExecution(
        process,
        200n,
        Date.now(),
        executor,
      );

      expect(result).not.toBeNull();
      expect(result!.success).toBe(true);
    });

    it("should return null when schedule does not match any contract", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const process: ScheduleProcess = {
        id: "unrelated_schedule",
        sourceMessageId: "msg_x",
        creator: OTHER_DID,
        condition: { time: { type: "absolute", executeAt: Date.now() } },
        watchedStateKeys: [],
        status: "active",
        executionCount: 0,
        remainingFee: 100n,
        feePerExecution: 10n,
        targetAddress: "unknown_address",
        callData: new Uint8Array([]),
        transferValue: 0n,
        generatedTxNonce: 0n,
        createdAt: Date.now(),
        modifiedAt: Date.now(),
      };

      const result = await integration.handleScheduleExecution(
        process,
        300n,
        Date.now(),
        executor,
      );

      expect(result).toBeNull();
    });
  });

  // ============ Dynamic Invocation Management ============

  describe("addInvocation", () => {
    it("should add a new invocation to an existing contract", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 500_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      const invocation = await integration.addInvocation(
        deployResult.address,
        {
          functionName: "newFunc",
          functionSelector: new Uint8Array([0x01, 0x02, 0x03, 0x04]),
          parameters: new Uint8Array([]),
          condition: { time: { type: "periodic", interval: 30000 } },
          scheduleType: "time_periodic",
          enabled: true,
          gasLimit: 50_000n,
          value: 0n,
        },
        CREATOR_DID,
        200n,
        Date.now(),
      );

      expect(invocation.id).toBeDefined();
      expect(invocation.functionName).toBe("newFunc");
      expect(invocation.invocationCount).toBe(0);
      expect(invocation.scheduleProcessId).toBeDefined();

      // Verify contract now has the invocation
      const contract = registry.getContract(deployResult.address);
      expect(contract!.invocations).toHaveLength(1);
    });

    it("should reject adding invocation for non-existent contract", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      await expect(
        integration.addInvocation(
          "nonexistent_addr",
          {
            functionName: "fn",
            functionSelector: new Uint8Array(4),
            parameters: new Uint8Array(0),
            condition: { time: { type: "periodic", interval: 1000 } },
            scheduleType: "time_periodic",
            enabled: true,
            gasLimit: 50_000n,
            value: 0n,
          },
          CREATOR_DID,
          100n,
          Date.now(),
        ),
      ).rejects.toThrowError(ContractError);
    });

    it("should reject adding invocation from non-creator", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 500_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      await expect(
        integration.addInvocation(
          deployResult.address,
          {
            functionName: "fn",
            functionSelector: new Uint8Array(4),
            parameters: new Uint8Array(0),
            condition: { time: { type: "periodic", interval: 1000 } },
            scheduleType: "time_periodic",
            enabled: true,
            gasLimit: 50_000n,
            value: 0n,
          },
          OTHER_DID, // Not the creator
          200n,
          Date.now(),
        ),
      ).rejects.toThrowError(ContractError);
    });
  });

  describe("disableInvocation", () => {
    it("should disable an invocation and cancel schedule", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [
            {
              functionName: "autoFn",
              functionSelector: new Uint8Array(4),
              parameters: new Uint8Array(0),
              condition: { time: { type: "periodic", interval: 60000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 100_000n,
              value: 0n,
            },
          ],
          prepaidGas: 1_000_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      const contract = registry.getContract(deployResult.address)!;
      const invId = contract.invocations[0].id;

      await integration.disableInvocation(
        deployResult.address,
        invId,
        CREATOR_DID,
      );

      // Verify invocation is disabled
      const updated = registry.getContract(deployResult.address)!;
      const inv = updated.invocations.find((i) => i.id === invId);
      expect(inv!.enabled).toBe(false);

      // Verify SPU modify was called with cancel action
      expect(mockSPU.modifications).toHaveLength(1);
      expect(mockSPU.modifications[0].action).toBe("cancel");
    });

    it("should reject disabling from non-creator", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [
            {
              functionName: "fn",
              functionSelector: new Uint8Array(4),
              parameters: new Uint8Array(0),
              condition: { time: { type: "periodic", interval: 60000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 100_000n,
              value: 0n,
            },
          ],
          prepaidGas: 500_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      const contract = registry.getContract(deployResult.address)!;
      const invId = contract.invocations[0].id;

      await expect(
        integration.disableInvocation(deployResult.address, invId, OTHER_DID),
      ).rejects.toThrowError(ContractError);
    });
  });

  describe("updateInvocationCondition", () => {
    it("should update condition and notify SPU", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [
            {
              functionName: "fn",
              functionSelector: new Uint8Array(4),
              parameters: new Uint8Array(0),
              condition: { time: { type: "periodic", interval: 60000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 100_000n,
              value: 0n,
            },
          ],
          prepaidGas: 500_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      const contract = registry.getContract(deployResult.address)!;
      const invId = contract.invocations[0].id;

      const newCondition: ScheduleCondition = {
        time: { type: "periodic", interval: 120000 },
      };

      await integration.updateInvocationCondition(
        deployResult.address,
        invId,
        newCondition,
        CREATOR_DID,
        200n,
        Date.now(),
      );

      // Verify condition was updated on contract
      const updated = registry.getContract(deployResult.address)!;
      const inv = updated.invocations.find((i) => i.id === invId);
      expect(inv!.condition.time!.interval).toBe(120000);

      // Verify SPU was notified
      const updateMods = mockSPU.modifications.filter(
        (m) => m.action === "update_condition",
      );
      expect(updateMods).toHaveLength(1);
    });

    it("should reject update from non-creator", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [
            {
              functionName: "fn",
              functionSelector: new Uint8Array(4),
              parameters: new Uint8Array(0),
              condition: { time: { type: "periodic", interval: 60000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 100_000n,
              value: 0n,
            },
          ],
          prepaidGas: 500_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      const contract = registry.getContract(deployResult.address)!;
      const invId = contract.invocations[0].id;

      await expect(
        integration.updateInvocationCondition(
          deployResult.address,
          invId,
          { time: { type: "periodic", interval: 999 } },
          OTHER_DID,
          200n,
          Date.now(),
        ),
      ).rejects.toThrowError(ContractError);
    });
  });

  // ============ Gas Management ============

  describe("addPrepaidGas / getGasBalance", () => {
    it("should add prepaid gas to a contract", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 100_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      expect(integration.getGasBalance(deployResult.address)).toBe(100_000n);

      integration.addPrepaidGas(deployResult.address, 50_000n, CREATOR_DID);

      expect(integration.getGasBalance(deployResult.address)).toBe(150_000n);
    });

    it("should reject adding gas from non-creator", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 100_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      expect(() =>
        integration.addPrepaidGas(deployResult.address, 50_000n, OTHER_DID),
      ).toThrowError(ContractError);
    });

    it("should throw for non-existent contract gas query", () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      expect(() => integration.getGasBalance("nonexistent")).toThrowError(
        ContractError,
      );
    });
  });

  // ============ Query Methods ============

  describe("getSchedulesForContract", () => {
    it("should return schedule process IDs for a contract", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [
            {
              functionName: "fn1",
              functionSelector: new Uint8Array(4),
              parameters: new Uint8Array(0),
              condition: { time: { type: "periodic", interval: 60000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 50_000n,
              value: 0n,
            },
          ],
          prepaidGas: 500_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      const schedules = integration.getSchedulesForContract(
        deployResult.address,
      );
      expect(schedules).toHaveLength(1);
      expect(schedules[0]).toMatch(/^sched_proc_/);
    });

    it("should return empty array for non-existent contract", () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      expect(integration.getSchedulesForContract("nonexistent")).toEqual([]);
    });
  });

  describe("getContractsWithPendingSchedules", () => {
    it("should return contracts that have enabled invocations with schedule IDs", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      // Deploy a contract with invocations
      await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [
            {
              functionName: "fn",
              functionSelector: new Uint8Array(4),
              parameters: new Uint8Array(0),
              condition: { time: { type: "periodic", interval: 60000 } },
              scheduleType: "time_periodic",
              enabled: true,
              gasLimit: 50_000n,
              value: 0n,
            },
          ],
          prepaidGas: 500_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      // Deploy a contract without invocations
      await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x02]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 100_000n,
          signature: "" as Signature,
          salt: new Uint8Array([0xff]), // Different salt for unique address
        },
        101n,
        Date.now(),
      );

      const pending = integration.getContractsWithPendingSchedules();
      expect(pending).toHaveLength(1);
    });
  });

  // ============ PoT-Specific Integration ============

  describe("createVerificationDeadline", () => {
    it("should create a one-time verification deadline invocation", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const abi = createMinimalABI([
        {
          name: "onVerificationTimeout",
          selector: "aabb0011",
          inputs: [],
          outputs: [],
          stateMutability: "nonpayable",
          selfInvokable: true,
        },
      ]);

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata({ abi }),
          invocations: [],
          prepaidGas: 1_000_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      const invocationId = await integration.createVerificationDeadline(
        deployResult.address,
        "claim_001",
        30000, // 30 second deadline
        200n,
        Date.now(),
      );

      expect(invocationId).toBeDefined();
      expect(typeof invocationId).toBe("string");

      // Contract should now have 1 invocation
      const contract = registry.getContract(deployResult.address);
      expect(contract!.invocations).toHaveLength(1);
      expect(contract!.invocations[0].functionName).toBe(
        "onVerificationTimeout",
      );
      expect(contract!.invocations[0].maxInvocations).toBe(1);
    });

    it("should throw if contract lacks verification timeout function", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(), // No verification function in ABI
          invocations: [],
          prepaidGas: 500_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      await expect(
        integration.createVerificationDeadline(
          deployResult.address,
          "claim_001",
          30000,
          200n,
          Date.now(),
        ),
      ).rejects.toThrowError(ContractError);
    });
  });

  describe("createConfidenceDecaySchedule", () => {
    it("should create a perpetual confidence decay schedule", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 1_000_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      const invocationId = await integration.createConfidenceDecaySchedule(
        deployResult.address,
        "claim_002",
        3600000, // 1 hour interval
        200n,
        Date.now(),
      );

      expect(invocationId).toBeDefined();

      const contract = registry.getContract(deployResult.address);
      const inv = contract!.invocations.find((i) => i.id === invocationId);
      expect(inv).toBeDefined();
      expect(inv!.functionName).toBe("updateConfidenceDecay");
      // No maxInvocations = perpetual
      expect(inv!.maxInvocations).toBeUndefined();
    });
  });

  describe("createChallengeDeadline", () => {
    it("should create a one-time challenge window close schedule", async () => {
      const { registry, invoker, executor, mockSPU, integration } =
        freshFixtures();

      const deployResult = await integration.deployContract(
        {
          deployer: CREATOR_DID,
          bytecode: new Uint8Array([0x01]),
          constructorArgs: new Uint8Array([]),
          metadata: createMetadata(),
          invocations: [],
          prepaidGas: 1_000_000n,
          signature: "" as Signature,
        },
        100n,
        Date.now(),
      );

      const invocationId = await integration.createChallengeDeadline(
        deployResult.address,
        "dispute_001",
        86400000, // 24-hour deadline
        200n,
        Date.now(),
      );

      expect(invocationId).toBeDefined();

      const contract = registry.getContract(deployResult.address);
      const inv = contract!.invocations.find((i) => i.id === invocationId);
      expect(inv).toBeDefined();
      expect(inv!.functionName).toBe("onChallengeWindowClose");
      expect(inv!.maxInvocations).toBe(1);
    });
  });
});
