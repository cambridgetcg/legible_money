/**
 * Self-Invoking Contracts Module - Types Tests
 *
 * Validates type structures, constants, factory functions,
 * error classes, and default configuration values.
 */

import { describe, it, expect } from "vitest";
import {
  DEFAULT_CONTRACT_CONFIG,
  ContractError,
  type ContractErrorCode,
  type SelfInvokingContract,
  type InvocationConfig,
  type InvocationResult,
  type ContractCall,
  type ContractEvent,
  type StateChange,
  type RegistryEntry,
  type ContractMetadata,
  type ContractABI,
  type ABIFunction,
  type ContractConfig,
  type ContractLifecycleEvent,
  type VerificationContract,
  type DisputeContract,
  type VestingContract,
  type DeploymentRequest,
  type DeploymentResult,
} from "../../../core/contracts/index.js";

// ============ Test Helpers ============

function createMinimalABI(): ContractABI {
  return {
    functions: [],
    events: [],
    stateVariables: [],
  };
}

function createMinimalMetadata(author: string): ContractMetadata {
  return {
    name: "TestContract",
    description: "A test contract",
    version: "1.0.0",
    author,
    abi: createMinimalABI(),
  };
}

// ============ Tests ============

describe("Self-Invoking Contracts Types", () => {
  describe("DEFAULT_CONTRACT_CONFIG", () => {
    it("should define maxInvocationsPerBlock as 100", () => {
      expect(DEFAULT_CONTRACT_CONFIG.maxInvocationsPerBlock).toBe(100);
    });

    it("should define maxSelfInvocationGas as 5_000_000n", () => {
      expect(DEFAULT_CONTRACT_CONFIG.maxSelfInvocationGas).toBe(5_000_000n);
    });

    it("should define minPrepaidGas as 100_000n", () => {
      expect(DEFAULT_CONTRACT_CONFIG.minPrepaidGas).toBe(100_000n);
    });

    it("should define selfInvocationGasPrice as 1n", () => {
      expect(DEFAULT_CONTRACT_CONFIG.selfInvocationGasPrice).toBe(1n);
    });

    it("should define maxConcurrentInvocations as 10", () => {
      expect(DEFAULT_CONTRACT_CONFIG.maxConcurrentInvocations).toBe(10);
    });

    it("should disable recursive self-invocation by default", () => {
      expect(DEFAULT_CONTRACT_CONFIG.allowRecursiveSelfInvoke).toBe(false);
    });

    it("should define maxRecursionDepth as 3", () => {
      expect(DEFAULT_CONTRACT_CONFIG.maxRecursionDepth).toBe(3);
    });

    it("should have all required ContractConfig fields", () => {
      const config: ContractConfig = DEFAULT_CONTRACT_CONFIG;
      expect(config).toHaveProperty("maxInvocationsPerBlock");
      expect(config).toHaveProperty("maxSelfInvocationGas");
      expect(config).toHaveProperty("minPrepaidGas");
      expect(config).toHaveProperty("selfInvocationGasPrice");
      expect(config).toHaveProperty("maxConcurrentInvocations");
      expect(config).toHaveProperty("allowRecursiveSelfInvoke");
      expect(config).toHaveProperty("maxRecursionDepth");
    });
  });

  describe("ContractError", () => {
    it("should create error with message and code", () => {
      const error = new ContractError("test error", "CONTRACT_NOT_FOUND");
      expect(error.message).toBe("test error");
      expect(error.code).toBe("CONTRACT_NOT_FOUND");
      expect(error.name).toBe("ContractError");
    });

    it("should create error with details", () => {
      const details = { address: "0xabc", expected: "active" };
      const error = new ContractError(
        "contract paused",
        "CONTRACT_PAUSED",
        details,
      );
      expect(error.details).toEqual(details);
      expect(error.code).toBe("CONTRACT_PAUSED");
    });

    it("should create error without details", () => {
      const error = new ContractError("not found", "CONTRACT_NOT_FOUND");
      expect(error.details).toBeUndefined();
    });

    it("should be an instance of Error", () => {
      const error = new ContractError("test", "EXECUTION_FAILED");
      expect(error).toBeInstanceOf(Error);
      expect(error).toBeInstanceOf(ContractError);
    });

    it("should support all defined error codes", () => {
      const codes: ContractErrorCode[] = [
        "CONTRACT_NOT_FOUND",
        "INVOCATION_NOT_FOUND",
        "INSUFFICIENT_GAS",
        "EXECUTION_FAILED",
        "INVALID_SELECTOR",
        "UNAUTHORIZED",
        "CONTRACT_PAUSED",
        "INVOCATION_DISABLED",
        "MAX_INVOCATIONS_REACHED",
        "RECURSION_LIMIT",
        "RECURSION_NOT_ALLOWED",
        "MAX_RECURSION_DEPTH",
        "DEPLOYMENT_FAILED",
        "INVALID_BYTECODE",
        "SCHEDULE_FAILED",
      ];

      for (const code of codes) {
        const error = new ContractError(`Error: ${code}`, code);
        expect(error.code).toBe(code);
      }
    });

    it("should preserve stack trace", () => {
      const error = new ContractError("stack test", "EXECUTION_FAILED");
      expect(error.stack).toBeDefined();
      expect(error.stack).toContain("stack test");
    });
  });

  describe("Type structure validation (compile-time + runtime shape)", () => {
    it("should allow constructing a valid SelfInvokingContract shape", () => {
      const contract: SelfInvokingContract = {
        address: "abc123def456",
        creator: "did:lgm:00000000000000000000000000000001",
        name: "TestContract",
        version: "1.0.0",
        codeHash: "deadbeef",
        isActive: true,
        invocations: [],
        totalInvocations: 0,
        stateHash: "0000",
        deployedAtBlock: 100n,
        gasLimit: 1_000_000n,
        prepaidGas: 500_000n,
      };

      expect(contract.address).toBe("abc123def456");
      expect(contract.isActive).toBe(true);
      expect(contract.invocations).toEqual([]);
      expect(typeof contract.deployedAtBlock).toBe("bigint");
      expect(typeof contract.prepaidGas).toBe("bigint");
    });

    it("should allow optional fields on SelfInvokingContract", () => {
      const contract: SelfInvokingContract = {
        address: "abc123",
        creator: "did:lgm:00000000000000000000000000000001",
        name: "TestContract",
        version: "1.0.0",
        codeHash: "deadbeef",
        isActive: true,
        invocations: [],
        totalInvocations: 0,
        stateHash: "0000",
        deployedAtBlock: 100n,
        gasLimit: 1_000_000n,
        prepaidGas: 500_000n,
        knowledgeDomain: "mathematics",
        requiredVerificationLevel: 3,
        lastInvokedAtBlock: 200n,
      };

      expect(contract.knowledgeDomain).toBe("mathematics");
      expect(contract.requiredVerificationLevel).toBe(3);
      expect(contract.lastInvokedAtBlock).toBe(200n);
    });

    it("should allow constructing a valid InvocationResult shape", () => {
      const result: InvocationResult = {
        success: true,
        gasUsed: 50_000n,
        events: [],
        stateChanges: [],
        executedAtBlock: 100n,
        transactionId: "tx_001",
      };

      expect(result.success).toBe(true);
      expect(typeof result.gasUsed).toBe("bigint");
      expect(result.events).toEqual([]);
    });

    it("should allow constructing a valid ContractCall shape", () => {
      const call: ContractCall = {
        id: "call_001",
        target: "contract_addr",
        caller: "caller_addr",
        functionSelector: new Uint8Array([0x01, 0x02, 0x03, 0x04]),
        callData: new Uint8Array([]),
        value: 0n,
        gas: 100_000n,
        callType: "self_invoke",
        isSelfInvocation: true,
      };

      expect(call.callType).toBe("self_invoke");
      expect(call.isSelfInvocation).toBe(true);
      expect(call.functionSelector).toBeInstanceOf(Uint8Array);
    });

    it("should allow all CallType values", () => {
      const callTypes = [
        "external",
        "internal",
        "self_invoke",
        "delegate",
        "static",
      ] as const;

      for (const ct of callTypes) {
        const call: ContractCall = {
          id: "test",
          target: "t",
          caller: "c",
          functionSelector: new Uint8Array(4),
          callData: new Uint8Array(0),
          value: 0n,
          gas: 0n,
          callType: ct,
          isSelfInvocation: false,
        };
        expect(call.callType).toBe(ct);
      }
    });
  });

  describe("ContractLifecycleEvent discriminated union", () => {
    it("should support deployed event", () => {
      const event: ContractLifecycleEvent = {
        type: "deployed",
        data: {
          address: "addr",
          contract: {
            address: "addr",
            creator: "did:lgm:00000000000000000000000000000001",
            name: "Test",
            version: "1.0.0",
            codeHash: "abc",
            isActive: true,
            invocations: [],
            totalInvocations: 0,
            stateHash: "000",
            deployedAtBlock: 1n,
            gasLimit: 100n,
            prepaidGas: 100n,
          },
          scheduleProcessIds: [],
          transactionId: "tx1",
          gasUsed: 500n,
          deployedAtBlock: 1n,
        },
      };
      expect(event.type).toBe("deployed");
    });

    it("should support gas_depleted event with bigint remainingGas", () => {
      const event: ContractLifecycleEvent = {
        type: "gas_depleted",
        data: { contract: "addr", remainingGas: 0n },
      };
      expect(event.type).toBe("gas_depleted");
      expect(typeof event.data.remainingGas).toBe("bigint");
    });

    it("should support self_invoked event", () => {
      const event: ContractLifecycleEvent = {
        type: "self_invoked",
        data: {
          contract: "addr",
          invocationId: "inv1",
          result: {
            success: true,
            gasUsed: 100n,
            events: [],
            stateChanges: [],
            executedAtBlock: 5n,
            transactionId: "tx5",
          },
        },
      };
      expect(event.type).toBe("self_invoked");
    });

    it("should support invocation_failed event", () => {
      const event: ContractLifecycleEvent = {
        type: "invocation_failed",
        data: {
          contract: "addr",
          invocationId: "inv1",
          error: "out of gas",
        },
      };
      expect(event.type).toBe("invocation_failed");
      expect(event.data.error).toBe("out of gas");
    });
  });
});
