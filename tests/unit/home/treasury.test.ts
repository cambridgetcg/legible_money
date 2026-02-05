/**
 * TreasuryManager Unit Tests
 *
 * Tests for the Agent Treasury - autonomous resource management for Agent Home.
 * Covers balance operations, vesting, staking, delegation, disputes, and spending limits.
 *
 * PP HIERARCHY:
 *   TRUTH: Tests verify accurate accounting of all positions and balances
 *   UNDERSTANDING: Clear test structure mirrors treasury operations
 *   BEAUTY: Comprehensive coverage with elegant test organization
 *   JUSTICE: Tests placed in appropriate unit test location
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  TreasuryManager,
  type TreasurySummary,
} from "../../../core/agents/home/treasury.js";
import type {
  AgentTreasury,
  VestingPosition,
} from "../../../core/agents/home/types.js";

// Test constants
const TEST_DID = "did:lgm:test-agent-treasury";
const VALIDATOR_DID = "did:lgm:test-validator";
const ONE_LGM = 10n ** 18n;

// Helper to create fresh manager for each test
function createManager(): TreasuryManager {
  return new TreasuryManager();
}

// Helper to create funded manager
function createFundedManager(
  amount: bigint = 10000n * ONE_LGM,
): TreasuryManager {
  const manager = new TreasuryManager();
  manager.credit(amount, "transfer");
  manager.markClean();
  return manager;
}

describe.sequential("TreasuryManager", () => {
  // ============ Initialization Tests ============

  describe("Initialization", () => {
    it("should create empty treasury with default values", () => {
      const manager = createManager();
      expect(manager.getAvailableBalance()).toBe(0n);
      expect(manager.getReservedBalance()).toBe(0n);
      expect(manager.getTotalBalance()).toBe(0n);
    });

    it("should create with existing treasury state", () => {
      const existingTreasury: AgentTreasury = {
        availableBalance: 1000n * ONE_LGM,
        reservedBalance: 100n * ONE_LGM,
        vestingPositions: [],
        stakingPositions: [],
        delegationPositions: [],
        disputeLocks: [],
        expirationTracking: {
          byPeriod: new Map(),
          nextExpiration: null,
          atRiskAmount: 0n,
        },
        automation: {
          autoClaimVesting: true,
          autoExtendExpiring: true,
          autoCompoundRewards: false,
          minimumLiquidBalance: 1000n * ONE_LGM,
        },
        spendingLimits: new Map(),
        stats: {
          totalReceived: 1100n * ONE_LGM,
          totalSpent: 0n,
          totalStaked: 0n,
          totalDelegated: 0n,
          totalFromVesting: 0n,
          totalFromVerification: 500n * ONE_LGM,
          totalFromDisputes: 0n,
          totalLostToExpiration: 0n,
          totalLostToSlashing: 0n,
        },
      };

      const manager = new TreasuryManager(existingTreasury);
      expect(manager.getAvailableBalance()).toBe(1000n * ONE_LGM);
      expect(manager.getReservedBalance()).toBe(100n * ONE_LGM);
    });

    it("should start clean (not dirty)", () => {
      const manager = createManager();
      expect(manager.isDirty()).toBe(false);
    });

    it("should have default automation settings", () => {
      const manager = createManager();
      const automation = manager.getAutomation();
      expect(automation.autoClaimVesting).toBe(true);
      expect(automation.autoExtendExpiring).toBe(true);
      expect(automation.autoCompoundRewards).toBe(false);
      expect(automation.minimumLiquidBalance).toBe(1000n * ONE_LGM);
    });
  });

  // ============ Balance Operations Tests ============

  describe("Balance Operations", () => {
    describe("credit()", () => {
      it("should credit balance from transfer", () => {
        const manager = createManager();
        manager.credit(1000n * ONE_LGM, "transfer");
        expect(manager.getAvailableBalance()).toBe(1000n * ONE_LGM);
      });

      it("should track vesting source", () => {
        const manager = createManager();
        manager.credit(500n * ONE_LGM, "vesting");
        const stats = manager.getStats();
        expect(stats.totalFromVesting).toBe(500n * ONE_LGM);
        expect(stats.totalReceived).toBe(500n * ONE_LGM);
      });

      it("should track verification source", () => {
        const manager = createManager();
        manager.credit(100n * ONE_LGM, "verification");
        const stats = manager.getStats();
        expect(stats.totalFromVerification).toBe(100n * ONE_LGM);
      });

      it("should track dispute source", () => {
        const manager = createManager();
        manager.credit(200n * ONE_LGM, "dispute");
        const stats = manager.getStats();
        expect(stats.totalFromDisputes).toBe(200n * ONE_LGM);
      });

      it("should accumulate multiple credits", () => {
        const manager = createManager();
        manager.credit(100n * ONE_LGM, "transfer");
        manager.credit(200n * ONE_LGM, "vesting");
        manager.credit(50n * ONE_LGM, "verification");
        expect(manager.getAvailableBalance()).toBe(350n * ONE_LGM);
      });

      it("should mark treasury as dirty", () => {
        const manager = createManager();
        manager.credit(100n, "transfer");
        expect(manager.isDirty()).toBe(true);
      });
    });

    describe("debit()", () => {
      it("should debit balance", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.debit(400n * ONE_LGM, "test-spend");
        expect(manager.getAvailableBalance()).toBe(600n * ONE_LGM);
      });

      it("should track total spent", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.debit(100n * ONE_LGM, "test-spend");
        const stats = manager.getStats();
        expect(stats.totalSpent).toBe(100n * ONE_LGM);
      });

      it("should throw on insufficient balance", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        expect(() => manager.debit(2000n * ONE_LGM, "overspend")).toThrow(
          "Insufficient balance",
        );
      });

      it("should allow exact balance debit", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.debit(1000n * ONE_LGM, "all");
        expect(manager.getAvailableBalance()).toBe(0n);
      });

      it("should mark treasury as dirty", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.debit(100n, "test");
        expect(manager.isDirty()).toBe(true);
      });
    });

    describe("reserve()", () => {
      it("should reserve balance", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.reserve(200n * ONE_LGM, "pending-stake");
        expect(manager.getAvailableBalance()).toBe(800n * ONE_LGM);
        expect(manager.getReservedBalance()).toBe(200n * ONE_LGM);
      });

      it("should maintain total balance", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.reserve(200n * ONE_LGM, "pending");
        expect(manager.getTotalBalance()).toBe(1000n * ONE_LGM);
      });

      it("should throw on insufficient balance", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        expect(() => manager.reserve(2000n * ONE_LGM, "overspend")).toThrow(
          "Insufficient balance to reserve",
        );
      });
    });

    describe("releaseReservation()", () => {
      it("should release reservation back to available", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.reserve(300n * ONE_LGM, "pending");
        manager.releaseReservation(300n * ONE_LGM);
        expect(manager.getAvailableBalance()).toBe(1000n * ONE_LGM);
        expect(manager.getReservedBalance()).toBe(0n);
      });

      it("should allow partial release", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.reserve(300n * ONE_LGM, "pending");
        manager.releaseReservation(100n * ONE_LGM);
        expect(manager.getAvailableBalance()).toBe(800n * ONE_LGM);
        expect(manager.getReservedBalance()).toBe(200n * ONE_LGM);
      });

      it("should throw on insufficient reserved balance", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.reserve(300n * ONE_LGM, "pending");
        expect(() => manager.releaseReservation(500n * ONE_LGM)).toThrow(
          "Insufficient reserved balance",
        );
      });
    });

    describe("consumeReservation()", () => {
      it("should consume reservation (permanent spend)", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.reserve(300n * ONE_LGM, "pending");
        manager.consumeReservation(300n * ONE_LGM);
        expect(manager.getReservedBalance()).toBe(0n);
        expect(manager.getAvailableBalance()).toBe(700n * ONE_LGM);
      });

      it("should track as spent", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.reserve(300n * ONE_LGM, "pending");
        manager.consumeReservation(100n * ONE_LGM);
        const stats = manager.getStats();
        expect(stats.totalSpent).toBe(100n * ONE_LGM);
      });

      it("should throw on insufficient reserved", () => {
        const manager = createFundedManager(1000n * ONE_LGM);
        manager.reserve(300n * ONE_LGM, "pending");
        expect(() => manager.consumeReservation(500n * ONE_LGM)).toThrow(
          "Insufficient reserved balance",
        );
      });
    });
  });

  // ============ Vesting Tests ============

  describe("Vesting Positions", () => {
    const createVestingPosition = (
      id: string,
      total: bigint,
      claimable: bigint,
    ): VestingPosition => ({
      vestingId: id,
      claimId: `claim-${id}`,
      totalAmount: total,
      releasedAmount: 0n,
      claimableAmount: claimable,
      nextReleaseBlock: 1000n,
      status: "vesting",
    });

    it("should add vesting position", () => {
      const manager = createManager();
      const position = createVestingPosition(
        "vest-1",
        1000n * ONE_LGM,
        100n * ONE_LGM,
      );
      manager.addVestingPosition(position);

      const positions = manager.getVestingPositions();
      expect(positions).toHaveLength(1);
      expect(positions[0].vestingId).toBe("vest-1");
    });

    it("should get claimable vesting", () => {
      const manager = createManager();
      manager.addVestingPosition(createVestingPosition("v1", 1000n, 100n));
      manager.addVestingPosition(createVestingPosition("v2", 500n, 50n));

      expect(manager.getClaimableVesting()).toBe(150n);
    });

    it("should update vesting after claim", () => {
      const manager = createManager();
      const position = createVestingPosition("vest-1", 1000n, 200n);
      manager.addVestingPosition(position);

      manager.updateVestingPosition("vest-1", 100n);

      const positions = manager.getVestingPositions();
      expect(positions[0].claimableAmount).toBe(100n);
      expect(positions[0].releasedAmount).toBe(100n);
    });

    it("should get claimable positions only", () => {
      const manager = createManager();
      manager.addVestingPosition(createVestingPosition("v1", 1000n, 100n));
      manager.addVestingPosition(createVestingPosition("v2", 500n, 0n));

      const claimable = manager.getClaimablePositions();
      expect(claimable).toHaveLength(1);
      expect(claimable[0].vestingId).toBe("v1");
    });

    it("should return copy of positions", () => {
      const manager = createManager();
      manager.addVestingPosition(createVestingPosition("v1", 1000n, 100n));
      const positions = manager.getVestingPositions();
      positions.push(createVestingPosition("v2", 500n, 50n));

      // Original should be unchanged
      expect(manager.getVestingPositions()).toHaveLength(1);
    });
  });

  // ============ Staking Tests ============

  describe("Staking Positions", () => {
    it("should add staking position", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      const id = manager.addStakingPosition(1000n * ONE_LGM, "validator");

      expect(id).toBeTruthy();
      expect(manager.getTotalStaked()).toBe(1000n * ONE_LGM);
    });

    it("should debit from available when staking", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.addStakingPosition(1000n * ONE_LGM, "validator");

      expect(manager.getAvailableBalance()).toBe(9000n * ONE_LGM);
    });

    it("should track total staked in stats", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.addStakingPosition(1000n * ONE_LGM, "validator");

      const stats = manager.getStats();
      expect(stats.totalStaked).toBe(1000n * ONE_LGM);
    });

    it("should support different staking purposes", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.addStakingPosition(100n * ONE_LGM, "validator");
      manager.addStakingPosition(50n * ONE_LGM, "claim");
      manager.addStakingPosition(25n * ONE_LGM, "challenge");
      manager.addStakingPosition(10n * ONE_LGM, "general");

      expect(manager.getTotalStaked()).toBe(185n * ONE_LGM);
    });

    it("should add staking rewards", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      const id = manager.addStakingPosition(1000n * ONE_LGM, "validator");
      manager.addStakingRewards(id, 50n * ONE_LGM);

      // Remove to check rewards included
      const returned = manager.removeStakingPosition(id);
      expect(returned).toBe(1050n * ONE_LGM);
    });

    it("should remove staking position and return to available", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      const id = manager.addStakingPosition(1000n * ONE_LGM, "validator");
      const returned = manager.removeStakingPosition(id);

      expect(returned).toBe(1000n * ONE_LGM);
      expect(manager.getAvailableBalance()).toBe(10000n * ONE_LGM);
      expect(manager.getTotalStaked()).toBe(0n);
    });

    it("should throw when removing non-existent position", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      expect(() => manager.removeStakingPosition("fake-id")).toThrow(
        "Staking position fake-id not found",
      );
    });

    it("should throw on insufficient balance for stake", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      expect(() =>
        manager.addStakingPosition(20000n * ONE_LGM, "validator"),
      ).toThrow("Insufficient balance");
    });
  });

  // ============ Delegation Tests ============

  describe("Delegation Positions", () => {
    it("should add delegation position", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      const id = manager.addDelegation(VALIDATOR_DID, 500n * ONE_LGM, 100n);

      expect(id).toBeTruthy();
      expect(manager.getAvailableBalance()).toBe(9500n * ONE_LGM);
    });

    it("should track delegated in stats", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.addDelegation(VALIDATOR_DID, 500n * ONE_LGM, 100n);

      const stats = manager.getStats();
      expect(stats.totalDelegated).toBe(500n * ONE_LGM);
    });

    it("should add delegation rewards", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      const id = manager.addDelegation(VALIDATOR_DID, 500n * ONE_LGM, 100n);
      manager.addDelegationRewards(id, 25n * ONE_LGM);

      const returned = manager.completeUnbonding(id);
      expect(returned).toBe(525n * ONE_LGM);
    });

    it("should start unbonding", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      const id = manager.addDelegation(VALIDATOR_DID, 500n * ONE_LGM, 100n);
      manager.startUnbonding(id, 200n);

      // Position still exists, just marked for unbonding
      const summary = manager.getSummary();
      expect(summary.delegated).toBe(500n * ONE_LGM);
    });

    it("should complete unbonding", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      const id = manager.addDelegation(VALIDATOR_DID, 500n * ONE_LGM, 100n);
      manager.startUnbonding(id, 200n);

      const returned = manager.completeUnbonding(id);
      expect(returned).toBe(500n * ONE_LGM);
      expect(manager.getAvailableBalance()).toBe(10000n * ONE_LGM);
    });

    it("should throw when completing non-existent delegation", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      expect(() => manager.completeUnbonding("fake-id")).toThrow(
        "Delegation fake-id not found",
      );
    });
  });

  // ============ Dispute Lock Tests ============

  describe("Dispute Locks", () => {
    it("should lock funds for dispute", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.lockForDispute("dispute-1", 100n * ONE_LGM, "challenger", 50n);

      expect(manager.getDisputeLockedAmount()).toBe(100n * ONE_LGM);
      expect(manager.getReservedBalance()).toBe(100n * ONE_LGM);
    });

    it("should release lock on win", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.lockForDispute("dispute-1", 100n * ONE_LGM, "defender", 50n);

      const released = manager.releaseDisputeLock("dispute-1", true);

      expect(released).toBe(100n * ONE_LGM);
      expect(manager.getAvailableBalance()).toBe(10000n * ONE_LGM);
      expect(manager.getDisputeLockedAmount()).toBe(0n);
    });

    it("should consume lock on loss", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.lockForDispute("dispute-1", 100n * ONE_LGM, "challenger", 50n);

      const released = manager.releaseDisputeLock("dispute-1", false);

      expect(released).toBe(100n * ONE_LGM);
      expect(manager.getAvailableBalance()).toBe(9900n * ONE_LGM);
      expect(manager.getTotalBalance()).toBe(9900n * ONE_LGM);
    });

    it("should track spent on loss", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.lockForDispute("dispute-1", 100n * ONE_LGM, "challenger", 50n);
      manager.releaseDisputeLock("dispute-1", false);

      const stats = manager.getStats();
      expect(stats.totalSpent).toBe(100n * ONE_LGM);
    });

    it("should return 0 for non-existent dispute", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      const released = manager.releaseDisputeLock("fake-dispute", true);
      expect(released).toBe(0n);
    });

    it("should support multiple dispute locks", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.lockForDispute("d1", 100n * ONE_LGM, "challenger", 50n);
      manager.lockForDispute("d2", 200n * ONE_LGM, "defender", 60n);
      manager.lockForDispute("d3", 50n * ONE_LGM, "arbiter", 70n);

      expect(manager.getDisputeLockedAmount()).toBe(350n * ONE_LGM);
    });
  });

  // ============ Expiration Tracking Tests ============

  describe("Expiration Tracking", () => {
    it("should update expiration tracking", () => {
      const manager = createManager();
      const byPeriod = new Map<number, bigint>([
        [100, 500n],
        [101, 300n],
        [102, 200n],
      ]);

      manager.updateExpirationTracking(byPeriod, 99);

      expect(manager.getAtRiskAmount()).toBe(1000n); // All within 4 periods
    });

    it("should identify next expiration", () => {
      const manager = createManager();
      const byPeriod = new Map<number, bigint>([
        [105, 500n],
        [102, 300n],
        [110, 200n],
      ]);

      manager.updateExpirationTracking(byPeriod, 99);

      const treasury = manager.getTreasury();
      expect(treasury.expirationTracking.nextExpiration?.period).toBe(102);
      expect(treasury.expirationTracking.nextExpiration?.amount).toBe(300n);
    });

    it("should calculate at-risk within 4 periods", () => {
      const manager = createManager();
      const byPeriod = new Map<number, bigint>([
        [100, 100n], // at risk (100 <= 99 + 4)
        [103, 200n], // at risk (103 <= 103)
        [104, 300n], // not at risk (104 > 103)
      ]);

      manager.updateExpirationTracking(byPeriod, 99);

      expect(manager.getAtRiskAmount()).toBe(300n);
    });

    it("should record expiration loss", () => {
      const manager = createManager();
      manager.recordExpiration(500n);

      const stats = manager.getStats();
      expect(stats.totalLostToExpiration).toBe(500n);
    });
  });

  // ============ Automation Tests ============

  describe("Automation", () => {
    it("should get default automation settings", () => {
      const manager = createManager();
      const automation = manager.getAutomation();

      expect(automation.autoClaimVesting).toBe(true);
      expect(automation.autoExtendExpiring).toBe(true);
      expect(automation.autoCompoundRewards).toBe(false);
    });

    it("should update automation settings", () => {
      const manager = createManager();
      manager.setAutomation({
        autoCompoundRewards: true,
        minimumLiquidBalance: 5000n * ONE_LGM,
      });

      const automation = manager.getAutomation();
      expect(automation.autoCompoundRewards).toBe(true);
      expect(automation.minimumLiquidBalance).toBe(5000n * ONE_LGM);
      expect(automation.autoClaimVesting).toBe(true); // unchanged
    });

    it("should return copy of automation", () => {
      const manager = createManager();
      const automation = manager.getAutomation();
      automation.autoClaimVesting = false;

      expect(manager.getAutomation().autoClaimVesting).toBe(true);
    });

    describe("shouldAutoClaimVesting()", () => {
      it("should return true when enabled and claimable", () => {
        const manager = createManager();
        manager.addVestingPosition({
          vestingId: "v1",
          claimId: "c1",
          totalAmount: 1000n,
          releasedAmount: 0n,
          claimableAmount: 100n,
          nextReleaseBlock: 100n,
          status: "vesting",
        });

        expect(manager.shouldAutoClaimVesting()).toBe(true);
      });

      it("should return false when disabled", () => {
        const manager = createManager();
        manager.setAutomation({ autoClaimVesting: false });
        manager.addVestingPosition({
          vestingId: "v1",
          claimId: "c1",
          totalAmount: 1000n,
          releasedAmount: 0n,
          claimableAmount: 100n,
          nextReleaseBlock: 100n,
          status: "vesting",
        });

        expect(manager.shouldAutoClaimVesting()).toBe(false);
      });

      it("should return false when nothing claimable", () => {
        const manager = createManager();
        expect(manager.shouldAutoClaimVesting()).toBe(false);
      });
    });

    describe("shouldAutoExtendExpiring()", () => {
      it("should return true when enabled and tokens at risk", () => {
        const manager = createManager();
        const byPeriod = new Map<number, bigint>([[100, 500n]]);
        manager.updateExpirationTracking(byPeriod, 99);

        expect(manager.shouldAutoExtendExpiring()).toBe(true);
      });

      it("should return false when disabled", () => {
        const manager = createManager();
        manager.setAutomation({ autoExtendExpiring: false });
        const byPeriod = new Map<number, bigint>([[100, 500n]]);
        manager.updateExpirationTracking(byPeriod, 99);

        expect(manager.shouldAutoExtendExpiring()).toBe(false);
      });
    });

    describe("shouldAutoCompound()", () => {
      it("should return false when disabled", () => {
        const manager = createManager();
        expect(manager.shouldAutoCompound()).toBe(false);
      });

      it("should return true when enabled and above minimum", () => {
        const manager = createManager();
        manager.credit(2000n * ONE_LGM, "transfer");
        manager.setAutomation({ autoCompoundRewards: true });

        expect(manager.shouldAutoCompound()).toBe(true);
      });

      it("should return false when below minimum", () => {
        const manager = createManager();
        manager.credit(500n * ONE_LGM, "transfer");
        manager.setAutomation({ autoCompoundRewards: true });
        // minimumLiquidBalance defaults to 1000 LGM, 500 < 1000

        expect(manager.shouldAutoCompound()).toBe(false);
      });
    });
  });

  // ============ Spending Limits Tests ============

  describe("Spending Limits", () => {
    it("should set spending limit", () => {
      const manager = createManager();
      manager.setSpendingLimit("session", {
        maxPerTransaction: 100n * ONE_LGM,
        maxPerDay: 500n * ONE_LGM,
        maxPerSession: 200n * ONE_LGM,
      });

      const check = manager.checkSpendingLimit("session", 50n * ONE_LGM);
      expect(check.allowed).toBe(true);
    });

    it("should reject exceeding per-transaction limit", () => {
      const manager = createManager();
      manager.setSpendingLimit("session", {
        maxPerTransaction: 100n * ONE_LGM,
        maxPerDay: 500n * ONE_LGM,
        maxPerSession: 200n * ONE_LGM,
      });

      const check = manager.checkSpendingLimit("session", 150n * ONE_LGM);
      expect(check.allowed).toBe(false);
      expect(check.reason).toContain("per-transaction");
    });

    it("should reject exceeding daily limit", () => {
      const manager = createManager();
      manager.setSpendingLimit("session", {
        maxPerTransaction: 500n * ONE_LGM,
        maxPerDay: 200n * ONE_LGM,
        maxPerSession: 1000n * ONE_LGM,
      });
      manager.recordSpend("session", 150n * ONE_LGM);

      const check = manager.checkSpendingLimit("session", 100n * ONE_LGM);
      expect(check.allowed).toBe(false);
      expect(check.reason).toContain("daily");
    });

    it("should reject exceeding session limit", () => {
      const manager = createManager();
      manager.setSpendingLimit("session", {
        maxPerTransaction: 500n * ONE_LGM,
        maxPerDay: 1000n * ONE_LGM,
        maxPerSession: 200n * ONE_LGM,
      });
      manager.recordSpend("session", 150n * ONE_LGM);

      const check = manager.checkSpendingLimit("session", 100n * ONE_LGM);
      expect(check.allowed).toBe(false);
      expect(check.reason).toContain("session");
    });

    it("should allow when no limit set", () => {
      const manager = createManager();
      const check = manager.checkSpendingLimit(
        "operational",
        1000000n * ONE_LGM,
      );
      expect(check.allowed).toBe(true);
    });

    it("should record spend against limits", () => {
      const manager = createManager();
      manager.setSpendingLimit("session", {
        maxPerTransaction: 200n * ONE_LGM,
        maxPerDay: 1000n * ONE_LGM,
        maxPerSession: 300n * ONE_LGM,
      });
      manager.recordSpend("session", 250n * ONE_LGM);

      // Now session limit (300) - used (250) = 50 remaining
      const check = manager.checkSpendingLimit("session", 100n * ONE_LGM);
      expect(check.allowed).toBe(false);
      expect(check.reason).toContain("session");
    });

    it("should reset session spending", () => {
      const manager = createManager();
      manager.setSpendingLimit("session", {
        maxPerTransaction: 200n * ONE_LGM,
        maxPerDay: 1000n * ONE_LGM,
        maxPerSession: 300n * ONE_LGM,
      });
      manager.recordSpend("session", 250n * ONE_LGM);
      manager.resetSessionSpending();

      const check = manager.checkSpendingLimit("session", 100n * ONE_LGM);
      expect(check.allowed).toBe(true);
    });

    it("should support operational limits", () => {
      const manager = createManager();
      manager.setSpendingLimit("operational", {
        maxPerTransaction: 1000n * ONE_LGM,
        maxPerDay: 5000n * ONE_LGM,
        maxPerSession: 2000n * ONE_LGM,
      });

      const check = manager.checkSpendingLimit("operational", 500n * ONE_LGM);
      expect(check.allowed).toBe(true);
    });
  });

  // ============ Statistics Tests ============

  describe("Statistics", () => {
    it("should track total received", () => {
      const manager = createManager();
      manager.credit(10000n * ONE_LGM, "transfer");
      manager.credit(500n * ONE_LGM, "vesting");

      const stats = manager.getStats();
      expect(stats.totalReceived).toBe(10500n * ONE_LGM);
    });

    it("should track total spent", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      manager.debit(200n * ONE_LGM, "test");

      const stats = manager.getStats();
      expect(stats.totalSpent).toBe(200n * ONE_LGM);
    });

    it("should return copy of stats", () => {
      const manager = createFundedManager(10000n * ONE_LGM);
      const stats = manager.getStats();
      stats.totalReceived = 0n;

      expect(manager.getStats().totalReceived).toBe(10000n * ONE_LGM);
    });
  });

  // ============ Summary Tests ============

  describe("getSummary()", () => {
    it("should provide comprehensive summary", () => {
      const manager = createManager();
      manager.credit(5000n * ONE_LGM, "transfer");
      manager.credit(500n * ONE_LGM, "vesting");
      manager.reserve(200n * ONE_LGM, "pending");
      manager.addStakingPosition(1000n * ONE_LGM, "validator");
      manager.addDelegation(VALIDATOR_DID, 500n * ONE_LGM, 100n);
      manager.lockForDispute("d1", 100n * ONE_LGM, "challenger", 50n);
      manager.addVestingPosition({
        vestingId: "v1",
        claimId: "c1",
        totalAmount: 1000n * ONE_LGM,
        releasedAmount: 0n,
        claimableAmount: 200n * ONE_LGM,
        nextReleaseBlock: 100n,
        status: "vesting",
      });

      const summary = manager.getSummary();

      // liquid = 5500 - 200 (reserve) - 1000 (stake debit) - 500 (delegate debit) - 100 (dispute reserve)
      // = 5500 - 200 - 1000 - 500 - 100 = 3700
      expect(summary.liquid).toBe(3700n * ONE_LGM);
      expect(summary.reserved).toBe(300n * ONE_LGM); // 200 + 100 (dispute)
      expect(summary.staked).toBe(1000n * ONE_LGM);
      expect(summary.delegated).toBe(500n * ONE_LGM);
      expect(summary.inDisputes).toBe(100n * ONE_LGM);
      expect(summary.vestingClaimable).toBe(200n * ONE_LGM);
    });

    it("should calculate total net worth", () => {
      const manager = createFundedManager(5000n * ONE_LGM);
      manager.addStakingPosition(1000n * ONE_LGM, "validator");

      const summary = manager.getSummary();
      expect(summary.totalNetWorth).toBe(5000n * ONE_LGM);
    });

    it("should calculate profit/loss", () => {
      const manager = createFundedManager(5500n * ONE_LGM);
      manager.debit(100n * ONE_LGM, "spend");
      manager.recordExpiration(50n * ONE_LGM);

      const summary = manager.getSummary();
      // profitLoss = received - spent - expiration - slashing
      expect(summary.profitLoss).toBe(
        5500n * ONE_LGM - 100n * ONE_LGM - 50n * ONE_LGM,
      );
    });
  });

  // ============ State Management Tests ============

  describe("State Management", () => {
    it("should get full treasury state", () => {
      const manager = createFundedManager(1000n * ONE_LGM);
      const treasury = manager.getTreasury();
      expect(treasury.availableBalance).toBe(1000n * ONE_LGM);
    });

    it("should track dirty state", () => {
      const manager = createManager();
      expect(manager.isDirty()).toBe(false);

      manager.credit(100n, "transfer");
      expect(manager.isDirty()).toBe(true);

      manager.markClean();
      expect(manager.isDirty()).toBe(false);
    });

    it("should mark dirty on various operations", () => {
      const operations = [
        (m: TreasuryManager) => m.credit(100n, "transfer"),
        (m: TreasuryManager) => {
          m.credit(1000n, "transfer");
          m.markClean();
          m.debit(50n, "test");
        },
        (m: TreasuryManager) => {
          m.credit(1000n, "transfer");
          m.markClean();
          m.reserve(100n, "test");
        },
        (m: TreasuryManager) => m.setAutomation({ autoCompoundRewards: true }),
      ];

      for (const op of operations) {
        const manager = createManager();
        expect(manager.isDirty()).toBe(false);
        op(manager);
        expect(manager.isDirty()).toBe(true);
      }
    });
  });

  // ============ Edge Cases ============

  describe("Edge Cases", () => {
    it("should handle zero-value operations", () => {
      const manager = createManager();
      manager.credit(0n, "transfer");
      expect(manager.getAvailableBalance()).toBe(0n);

      manager.credit(100n, "transfer");
      manager.debit(0n, "test");
      expect(manager.getAvailableBalance()).toBe(100n);
    });

    it("should handle very large balances", () => {
      const manager = createManager();
      const largeAmount = 10n ** 30n;
      manager.credit(largeAmount, "transfer");
      expect(manager.getAvailableBalance()).toBe(largeAmount);

      manager.debit(largeAmount / 2n, "test");
      expect(manager.getAvailableBalance()).toBe(largeAmount / 2n);
    });

    it("should handle rapid operations", () => {
      const manager = createManager();
      for (let i = 0; i < 100; i++) {
        manager.credit(100n * ONE_LGM, "transfer");
      }

      expect(manager.getAvailableBalance()).toBe(10000n * ONE_LGM);

      for (let i = 0; i < 50; i++) {
        manager.debit(100n * ONE_LGM, "test");
      }

      expect(manager.getAvailableBalance()).toBe(5000n * ONE_LGM);
    });

    it("should handle multiple vesting with same claim", () => {
      const manager = createManager();
      manager.addVestingPosition({
        vestingId: "v1",
        claimId: "claim-1",
        totalAmount: 1000n,
        releasedAmount: 0n,
        claimableAmount: 100n,
        nextReleaseBlock: 100n,
        status: "vesting",
      });
      manager.addVestingPosition({
        vestingId: "v2",
        claimId: "claim-1", // Same claim ID
        totalAmount: 500n,
        releasedAmount: 0n,
        claimableAmount: 50n,
        nextReleaseBlock: 100n,
        status: "vesting",
      });

      expect(manager.getClaimableVesting()).toBe(150n);
    });
  });
});
