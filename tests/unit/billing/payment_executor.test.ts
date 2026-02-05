/**
 * Payment Executor Tests
 *
 * Tests for atomic payment distribution execution.
 * Covers: successful transfers, insufficient balance rollback,
 * zero-cost distributions, canAffordDistribution, calculateTotalCost.
 */

import { describe, it, expect } from "vitest";
import {
  executePaymentDistribution,
  canAffordDistribution,
  calculateTotalCost,
} from "../../../core/billing/payment_executor.js";
import type { PaymentDistribution } from "../../../core/billing/types.js";
import { StateManager } from "../../../state/state_manager.js";

// ============ Test Helpers ============

function createTestDistribution(
  overrides: Partial<{
    apiAmount: bigint;
    knowledgeRewards: Array<{
      submitter: string;
      amount: bigint;
      reason: string;
    }>;
    burned: bigint;
    treasury: bigint;
    researchAmount: bigint;
  }>,
): PaymentDistribution {
  return {
    apiProvider: {
      did: "did:lgm:provider" as any,
      amount: overrides.apiAmount ?? 100n,
    },
    knowledgeRewards:
      overrides.knowledgeRewards ??
      ([
        {
          submitter: "did:lgm:contributor1",
          amount: 30n,
          reason: "direct_citation",
          factId: "fact1",
          confidence: 0.9,
        },
      ] as any[]),
    protocol: {
      burned: overrides.burned ?? 5n,
      treasury: overrides.treasury ?? 5n,
    },
    research: {
      amount: overrides.researchAmount ?? 22n,
      treasury: "did:lgm:research" as any,
    },
  };
}

function setupStateWithAccounts(
  state: StateManager,
  accounts: Record<string, bigint>,
): void {
  for (const [did, balance] of Object.entries(accounts)) {
    state.createAccount(did, "keyhash_" + did, balance);
  }
}

// ============ Tests ============

describe("calculateTotalCost", () => {
  it("should sum all distribution components", () => {
    const dist = createTestDistribution({
      apiAmount: 100n,
      knowledgeRewards: [
        { submitter: "did:lgm:c1", amount: 30n, reason: "citation" },
        { submitter: "did:lgm:c2", amount: 20n, reason: "citation" },
      ],
      burned: 5n,
      treasury: 5n,
      researchAmount: 22n,
    });

    expect(calculateTotalCost(dist)).toBe(182n);
  });

  it("should handle zero amounts", () => {
    const dist = createTestDistribution({
      apiAmount: 0n,
      knowledgeRewards: [],
      burned: 0n,
      treasury: 0n,
      researchAmount: 0n,
    });

    expect(calculateTotalCost(dist)).toBe(0n);
  });

  it("should handle distribution without research field", () => {
    const dist: PaymentDistribution = {
      apiProvider: { did: "did:lgm:p" as any, amount: 50n },
      knowledgeRewards: [],
      protocol: { burned: 3n, treasury: 2n },
    };

    expect(calculateTotalCost(dist)).toBe(55n);
  });
});

describe("canAffordDistribution", () => {
  it("should return true when balance covers total cost", () => {
    const dist = createTestDistribution({ apiAmount: 100n });
    const totalCost = calculateTotalCost(dist);
    expect(canAffordDistribution(totalCost + 1n, dist)).toBe(true);
    expect(canAffordDistribution(totalCost, dist)).toBe(true);
  });

  it("should return false when balance insufficient", () => {
    const dist = createTestDistribution({ apiAmount: 100n });
    expect(canAffordDistribution(0n, dist)).toBe(false);
    expect(canAffordDistribution(1n, dist)).toBe(false);
  });

  it("should apply safety margin", () => {
    const dist = createTestDistribution({
      apiAmount: 100n,
      knowledgeRewards: [],
      burned: 0n,
      treasury: 0n,
      researchAmount: 0n,
    });
    // Total = 100. With 1.5x margin, need 150
    expect(canAffordDistribution(150n, dist, 1.5)).toBe(true);
    expect(canAffordDistribution(149n, dist, 1.5)).toBe(false);
  });
});

describe("executePaymentDistribution", () => {
  function freshState(accounts: Record<string, bigint>): StateManager {
    const state = new StateManager();
    for (const [did, balance] of Object.entries(accounts)) {
      state.createAccount(did, "keyhash_" + did, balance);
    }
    return state;
  }

  it("should return success with zero-cost distribution", () => {
    const state = new StateManager();
    const dist = createTestDistribution({
      apiAmount: 0n,
      knowledgeRewards: [],
      burned: 0n,
      treasury: 0n,
      researchAmount: 0n,
    });

    const result = executePaymentDistribution(
      dist,
      "did:lgm:caller" as any,
      state,
      100n,
    );
    expect(result.success).toBe(true);
    expect(result.transfers).toHaveLength(0);
    expect(result.totalTransferred).toBe(0n);
  });

  it("should fail with insufficient balance", () => {
    const state = freshState({
      "did:lgm:caller": 10n, // Not enough for 162n total
      "did:lgm:provider": 0n,
      "did:lgm:contributor1": 0n,
      "did:lgm:research": 0n,
      "did:lgm:burn": 0n,
      "did:lgm:protocol-treasury": 0n,
    });

    const dist = createTestDistribution({});
    const result = executePaymentDistribution(
      dist,
      "did:lgm:caller" as any,
      state,
      100n,
    );

    expect(result.success).toBe(false);
    expect(result.errors).toHaveLength(1);
    expect(result.errors[0]).toContain("Insufficient balance");
  });

  it("should execute all transfers atomically on success", () => {
    const totalCost = 162n; // 100 + 30 + 5 + 5 + 22
    const state = freshState({
      "did:lgm:caller": totalCost + 100n,
      "did:lgm:provider": 0n,
      "did:lgm:contributor1": 0n,
      "did:lgm:research": 0n,
      "did:lgm:burn": 0n,
      "did:lgm:protocol-treasury": 0n,
    });

    const dist = createTestDistribution({});
    const result = executePaymentDistribution(
      dist,
      "did:lgm:caller" as any,
      state,
      100n,
    );

    expect(result.success).toBe(true);
    expect(result.totalTransferred).toBe(totalCost);
    expect(result.researchFunded).toBe(22n);

    // Verify individual transfer records
    // Research 22n is split 5 ways: research(16n), foundation(1n), community(1n), developers(1n), reserve(3n)
    expect(
      result.transfers.find((t) => t.reason === "api_service")?.amount,
    ).toBe(100n);
    expect(
      result.transfers.find((t) => t.reason === "protocol_burn")?.amount,
    ).toBe(5n);
    expect(
      result.transfers.find((t) => t.reason === "protocol_treasury")?.amount,
    ).toBe(5n);

    // Treasury split: 22n total across 5 funds
    const treasuryTransfers = result.transfers.filter((t) =>
      t.reason.startsWith("treasury_"),
    );
    const treasuryTotal = treasuryTransfers.reduce(
      (sum, t) => sum + t.amount,
      0n,
    );
    expect(treasuryTotal).toBe(22n);

    // Verify state balances changed
    expect(state.getBalance("did:lgm:caller")).toBe(100n); // started with 262, spent 162
    expect(state.getBalance("did:lgm:provider")).toBe(100n);
    expect(state.getBalance("did:lgm:contributor1")).toBe(30n);
    // Research gets 77% of 22n = 16n
    expect(state.getBalance("did:lgm:research")).toBe(16n);
    expect(state.getBalance("did:lgm:burn")).toBe(5n);
    expect(state.getBalance("did:lgm:protocol-treasury")).toBe(5n);
  });

  it("should handle multiple knowledge rewards", () => {
    const state = freshState({
      "did:lgm:caller": 1000n,
      "did:lgm:provider": 0n,
      "did:lgm:c1": 0n,
      "did:lgm:c2": 0n,
      "did:lgm:c3": 0n,
      "did:lgm:research": 0n,
      "did:lgm:burn": 0n,
      "did:lgm:protocol-treasury": 0n,
    });

    const dist = createTestDistribution({
      apiAmount: 50n,
      knowledgeRewards: [
        { submitter: "did:lgm:c1", amount: 20n, reason: "direct_citation" },
        { submitter: "did:lgm:c2", amount: 15n, reason: "ancestor" },
        { submitter: "did:lgm:c3", amount: 10n, reason: "cross_domain_bonus" },
      ],
      burned: 3n,
      treasury: 2n,
      researchAmount: 11n,
    });

    const result = executePaymentDistribution(
      dist,
      "did:lgm:caller" as any,
      state,
      100n,
    );
    expect(result.success).toBe(true);
    // api(1) + knowledge(3) + treasury_research(1) + treasury_reserve(1) + burn(1) + protocol_treasury(1) = 8
    // (11n research: 77%=8n research, 7%=0n each for found/comm/dev, remainder=3n reserve → 2 treasury transfers)
    expect(result.transfers).toHaveLength(8);

    expect(state.getBalance("did:lgm:c1")).toBe(20n);
    expect(state.getBalance("did:lgm:c2")).toBe(15n);
    expect(state.getBalance("did:lgm:c3")).toBe(10n);
  });
});
