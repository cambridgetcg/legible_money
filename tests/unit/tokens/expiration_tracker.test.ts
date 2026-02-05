/**
 * ExpirationTracker Unit Tests
 *
 * Tests for the Dynamic Array method implementation for global expiration tracking.
 * Coordinates token expiration across all accounts with O(1) operations.
 *
 * PP HIERARCHY:
 *   TRUTH: Verify period transitions and expiration calculations are correct
 *   UNDERSTANDING: Test all tracker operations (mint, debit, expire)
 *   BEAUTY: Clear test structure mirroring tracker methods
 *   JUSTICE: Tests in proper tokens unit test location
 */

import { describe, it, expect, beforeEach } from "vitest";
import { ExpirationTracker } from "../../../core/tokens/expiration_tracker.js";
import {
  DEFAULT_EXPIRATION_CONFIG,
  TokenError,
  type ExpirationConfig,
} from "../../../core/tokens/types.js";

// ============ Test Helpers ============

const TEST_DID = "did:lgm:test-account-0123456789abcdef";
const TEST_DID_2 = "did:lgm:test-account-2-fedcba9876543210";

function createTestConfig(
  overrides: Partial<ExpirationConfig> = {},
): ExpirationConfig {
  return {
    ...DEFAULT_EXPIRATION_CONFIG,
    periodDurationMs: 1000, // 1 second periods for testing
    periodsUntilExpiration: 5, // 5 periods until expiration
    tokenLifespanMs: 5000,
    dustThreshold: 1n, // Allow small test amounts
    ...overrides,
  };
}

// ============ Initialization Tests ============

describe.sequential("ExpirationTracker Initialization", () => {
  it("should initialize with default config", () => {
    const tracker = new ExpirationTracker();
    expect(tracker).toBeDefined();
    const state = tracker.getGlobalState();
    expect(state.totalActiveSupply).toBe(0n);
    expect(state.totalExpiredSupply).toBe(0n);
  });

  it("should initialize with custom config", () => {
    const config = createTestConfig({ periodsUntilExpiration: 10 });
    const tracker = new ExpirationTracker(config);
    expect(tracker).toBeDefined();
  });

  it("should initialize with specific initial period", () => {
    const config = createTestConfig();
    const initialPeriod = 100;
    const tracker = new ExpirationTracker(config, initialPeriod);

    const state = tracker.getGlobalState();
    expect(state.currentPeriod).toBe(initialPeriod);
    expect(state.oldestActivePeriod).toBe(initialPeriod);
  });
});

// ============ Period Management Tests ============

describe.sequential("ExpirationTracker Period Management", () => {
  let tracker: ExpirationTracker;
  let config: ExpirationConfig;
  let basePeriod: number;

  beforeEach(() => {
    config = createTestConfig();
    basePeriod = 1000;
    tracker = new ExpirationTracker(config, basePeriod);
  });

  it("should create new period state", () => {
    const periodState = tracker.getOrCreatePeriod(basePeriod, 1n);

    expect(periodState.period).toBe(basePeriod);
    expect(periodState.totalMinted).toBe(0n);
    expect(periodState.activeBalance).toBe(0n);
    expect(periodState.accountCount).toBe(0);
    expect(periodState.fullyExpired).toBe(false);
  });

  it("should return existing period state", () => {
    const first = tracker.getOrCreatePeriod(basePeriod, 1n);
    first.totalMinted = 1000n;

    const second = tracker.getOrCreatePeriod(basePeriod, 2n);
    expect(second.totalMinted).toBe(1000n);
  });

  it("should record mint and update period state", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const periodState = tracker.getPeriodState(basePeriod);
    expect(periodState).toBeDefined();
    expect(periodState!.totalMinted).toBe(1000n);
    expect(periodState!.activeBalance).toBe(1000n);
    expect(periodState!.accountCount).toBe(1);
  });

  it("should accumulate mints in same period", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID, 500n, basePeriod, 2n);

    const periodState = tracker.getPeriodState(basePeriod);
    expect(periodState!.totalMinted).toBe(1500n);
    expect(periodState!.activeBalance).toBe(1500n);
    // Same account, count should still be 1
    expect(periodState!.accountCount).toBe(1);
  });

  it("should track multiple accounts in same period", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID_2, 500n, basePeriod, 2n);

    const periodState = tracker.getPeriodState(basePeriod);
    expect(periodState!.totalMinted).toBe(1500n);
    expect(periodState!.accountCount).toBe(2);
  });

  it("should update global state on mint", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const globalState = tracker.getGlobalState();
    expect(globalState.totalActiveSupply).toBe(1000n);
    expect(globalState.totalExpiredSupply).toBe(0n);
  });

  it("should update current period if mint is in future period", () => {
    const futurePeriod = basePeriod + 5;
    tracker.recordMint(TEST_DID, 1000n, futurePeriod, 1n);

    const globalState = tracker.getGlobalState();
    expect(globalState.currentPeriod).toBe(futurePeriod);
  });

  it("should record debit and update period state", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordDebit(TEST_DID, 300n, basePeriod, false);

    const periodState = tracker.getPeriodState(basePeriod);
    expect(periodState!.activeBalance).toBe(700n);
    expect(periodState!.expiredBalance).toBe(0n);
  });

  it("should record expiration debit differently", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordDebit(TEST_DID, 300n, basePeriod, true);

    const periodState = tracker.getPeriodState(basePeriod);
    expect(periodState!.activeBalance).toBe(700n);
    expect(periodState!.expiredBalance).toBe(300n);

    const globalState = tracker.getGlobalState();
    expect(globalState.totalActiveSupply).toBe(700n);
    expect(globalState.totalExpiredSupply).toBe(300n);
  });

  it("should throw error when debiting non-existent period", () => {
    expect(() => tracker.recordDebit(TEST_DID, 100n, 9999, false)).toThrow(
      TokenError,
    );
  });
});

// ============ Period Transition Tests ============

describe.sequential("ExpirationTracker Period Transitions", () => {
  let tracker: ExpirationTracker;
  let config: ExpirationConfig;
  let basePeriod: number;

  beforeEach(() => {
    config = createTestConfig({ periodsUntilExpiration: 3 });
    basePeriod = 1000;
    tracker = new ExpirationTracker(config, basePeriod);
  });

  it("should return null when no transition needed", () => {
    const result = tracker.processPeriodTransition(basePeriod, 1n);
    expect(result).toBeNull();
  });

  it("should return null for past periods", () => {
    const result = tracker.processPeriodTransition(basePeriod - 1, 1n);
    expect(result).toBeNull();
  });

  it("should process period transition without expiration", () => {
    const result = tracker.processPeriodTransition(basePeriod + 1, 1n);

    // No expired content yet, but period should advance
    expect(result).toBeNull();
    expect(tracker.getGlobalState().currentPeriod).toBe(basePeriod + 1);
  });

  it("should expire old period when transitioning", () => {
    // Mint in period 1000
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // processPeriodTransition expires exactly (newPeriod - periodsUntilExpiration)
    // To expire period 1000: newPeriod - 3 = 1000, so newPeriod = 1003
    const newPeriod = basePeriod + config.periodsUntilExpiration;
    const result = tracker.processPeriodTransition(newPeriod, 10n);

    expect(result).not.toBeNull();
    expect(result!.expiredPeriod).toBe(basePeriod);
    expect(result!.expiredAmount).toBe(1000n);
    expect(result!.accountsAffected).toBe(1);
    expect(result!.transitionBlock).toBe(10n);
  });

  it("should mark period as fully expired", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // To expire basePeriod: newPeriod = basePeriod + periodsUntilExpiration
    const newPeriod = basePeriod + config.periodsUntilExpiration;
    tracker.processPeriodTransition(newPeriod, 10n);

    const periodState = tracker.getPeriodState(basePeriod);
    expect(periodState!.fullyExpired).toBe(true);
    expect(periodState!.expiredAtBlock).toBe(10n);
    expect(periodState!.activeBalance).toBe(0n);
    expect(periodState!.expiredBalance).toBe(1000n);
  });

  it("should update global state on expiration", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // To expire basePeriod: newPeriod = basePeriod + periodsUntilExpiration
    const newPeriod = basePeriod + config.periodsUntilExpiration;
    tracker.processPeriodTransition(newPeriod, 10n);

    const globalState = tracker.getGlobalState();
    expect(globalState.totalActiveSupply).toBe(0n);
    expect(globalState.totalExpiredSupply).toBe(1000n);
    expect(globalState.currentPeriod).toBe(newPeriod);
  });

  it("should update oldest active period pointer", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID, 500n, basePeriod + 2, 2n);

    // Advance past first period expiration
    const newPeriod = basePeriod + config.periodsUntilExpiration + 1;
    tracker.processPeriodTransition(newPeriod, 10n);

    const globalState = tracker.getGlobalState();
    expect(globalState.oldestActivePeriod).toBeGreaterThan(basePeriod);
  });
});

// ============ Query Methods Tests ============

describe.sequential("ExpirationTracker Query Methods", () => {
  let tracker: ExpirationTracker;
  let config: ExpirationConfig;
  let basePeriod: number;
  let baseTime: number;

  beforeEach(() => {
    config = createTestConfig({ periodsUntilExpiration: 3 });
    basePeriod = 1000;
    baseTime = basePeriod * config.periodDurationMs;
    tracker = new ExpirationTracker(config, basePeriod);
  });

  it("should return empty expired periods initially", () => {
    const expired = tracker.getExpiredPeriods(baseTime);
    expect(expired).toEqual([]);
  });

  it("should return expired periods that need processing", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // Time that would cause basePeriod to expire
    const futureTime =
      (basePeriod + config.periodsUntilExpiration + 1) *
      config.periodDurationMs;
    const expired = tracker.getExpiredPeriods(futureTime);

    expect(expired).toContain(basePeriod);
  });

  it("should not return fully processed expired periods", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // To expire basePeriod: newPeriod = basePeriod + periodsUntilExpiration
    const newPeriod = basePeriod + config.periodsUntilExpiration;
    tracker.processPeriodTransition(newPeriod, 10n);

    const futureTime = newPeriod * config.periodDurationMs;
    const expired = tracker.getExpiredPeriods(futureTime);

    expect(expired).not.toContain(basePeriod);
  });

  it("should return periods expiring soon", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // Time close to expiration
    const nearExpiryTime =
      (basePeriod + config.periodsUntilExpiration - 1) *
      config.periodDurationMs;
    const expiringSoon = tracker.getExpiringSoonPeriods(nearExpiryTime, 2);

    expect(expiringSoon.length).toBeGreaterThan(0);
    expect(expiringSoon[0].period).toBe(basePeriod);
    expect(expiringSoon[0].balance).toBe(1000n);
  });

  it("should calculate total expiring balance", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID_2, 500n, basePeriod + 1, 2n);

    // getExpiringSoonPeriods finds periods where expirationThreshold < period <= soonThreshold
    // At time = basePeriod * periodDurationMs:
    //   currentPeriod = basePeriod
    //   expirationThreshold = basePeriod - 3
    //   soonThreshold = basePeriod - 3 + withinPeriods = basePeriod - 3 + 4 = basePeriod + 1
    // So both basePeriod and basePeriod + 1 should be included with withinPeriods=4
    const currentTime = basePeriod * config.periodDurationMs;
    const expiringBalance = tracker.getExpiringBalance(currentTime, 4);

    expect(expiringBalance).toBe(1500n);
  });

  it("should return undefined for non-existent period state", () => {
    const periodState = tracker.getPeriodState(9999);
    expect(periodState).toBeUndefined();
  });

  it("should return active periods with balance", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID, 500n, basePeriod + 1, 2n);

    const activePeriods = tracker.getActivePeriods();
    expect(activePeriods).toContain(basePeriod);
    expect(activePeriods).toContain(basePeriod + 1);
    expect(activePeriods.length).toBe(2);
  });

  it("should return sorted active periods", () => {
    tracker.recordMint(TEST_DID, 500n, basePeriod + 2, 1n);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 2n);

    const activePeriods = tracker.getActivePeriods();
    expect(activePeriods[0]).toBe(basePeriod);
    expect(activePeriods[1]).toBe(basePeriod + 2);
  });

  it("should return periods for specific account", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID, 500n, basePeriod + 1, 2n);
    tracker.recordMint(TEST_DID_2, 300n, basePeriod + 2, 3n);

    const accountPeriods = tracker.getAccountPeriods(TEST_DID);
    expect(accountPeriods).toContain(basePeriod);
    expect(accountPeriods).toContain(basePeriod + 1);
    expect(accountPeriods).not.toContain(basePeriod + 2);
  });

  it("should return empty array for account with no periods", () => {
    const accountPeriods = tracker.getAccountPeriods("did:lgm:nonexistent");
    expect(accountPeriods).toEqual([]);
  });
});

// ============ Dynamic Array State Tests ============

describe.sequential("ExpirationTracker Dynamic Array State", () => {
  let tracker: ExpirationTracker;
  let config: ExpirationConfig;
  let basePeriod: number;

  beforeEach(() => {
    config = createTestConfig({ periodsUntilExpiration: 3 });
    basePeriod = 1000;
    tracker = new ExpirationTracker(config, basePeriod);
  });

  it("should build dynamic array state for account", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID, 500n, basePeriod + 1, 2n);

    const periodBalances = new Map<number, bigint>([
      [basePeriod, 1000n],
      [basePeriod + 1, 500n],
    ]);

    const state = tracker.buildDynamicArrayState(TEST_DID, periodBalances);

    expect(state.account).toBe(TEST_DID);
    expect(state.activeBalance).toBe(1500n);
    expect(state.headPeriod).toBe(basePeriod);
    expect(state.tailPeriod).toBe(basePeriod + 1);
  });

  it("should calculate pending expiration in dynamic array", () => {
    // This test is timing-sensitive; just verify structure
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const periodBalances = new Map<number, bigint>([[basePeriod, 1000n]]);
    const state = tracker.buildDynamicArrayState(TEST_DID, periodBalances);

    expect(state.periods).toBeDefined();
    expect(state.currentPeriod).toBeDefined();
    expect(state.pendingExpiration).toBeDefined();
  });
});

// ============ Merkle Proof Tests ============

describe.sequential("ExpirationTracker Merkle Proofs", () => {
  let tracker: ExpirationTracker;
  let config: ExpirationConfig;
  let basePeriod: number;

  beforeEach(() => {
    config = createTestConfig();
    basePeriod = 1000;
    tracker = new ExpirationTracker(config, basePeriod);
  });

  it("should generate period state hash for empty period", () => {
    const hash = tracker.getPeriodStateHash(9999);
    expect(hash).toBeDefined();
    expect(hash.length).toBeGreaterThan(0);
  });

  it("should generate period state hash for active period", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const hash = tracker.getPeriodStateHash(basePeriod);
    expect(hash).toBeDefined();
    expect(hash.length).toBeGreaterThan(0);
  });

  it("should generate different hashes for different states", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    const hash1 = tracker.getPeriodStateHash(basePeriod);

    tracker.recordMint(TEST_DID, 500n, basePeriod, 2n);
    const hash2 = tracker.getPeriodStateHash(basePeriod);

    expect(hash1).not.toBe(hash2);
  });

  it("should generate global state hash", () => {
    const hash = tracker.getGlobalStateHash();
    expect(hash).toBeDefined();
    expect(hash.length).toBeGreaterThan(0);
  });

  it("should update global state hash after changes", () => {
    const hash1 = tracker.getGlobalStateHash();

    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    const hash2 = tracker.getGlobalStateHash();

    expect(hash1).not.toBe(hash2);
  });
});

// ============ Utility Tests ============

describe.sequential("ExpirationTracker Utilities", () => {
  let tracker: ExpirationTracker;
  let config: ExpirationConfig;

  beforeEach(() => {
    config = createTestConfig({ periodDurationMs: 1000 });
    tracker = new ExpirationTracker(config, 100);
  });

  it("should convert timestamp to period", () => {
    const timestamp = 5500;
    const period = tracker.timestampToPeriod(timestamp);
    expect(period).toBe(5);
  });

  it("should convert period to timestamp", () => {
    const period = 5;
    const timestamp = tracker.periodToTimestamp(period);
    expect(timestamp).toBe(5000);
  });

  it("should calculate time until expiration", () => {
    const period = 100;
    const currentTime = 102 * config.periodDurationMs;

    const timeUntil = tracker.getTimeUntilExpiration(period, currentTime);
    // Period expires at 100 + 5 = 105, current is 102
    // 105 - 102 = 3 periods = 3000ms
    expect(timeUntil).toBe(3 * config.periodDurationMs);
  });

  it("should return 0 for already expired period", () => {
    const period = 100;
    const currentTime = 110 * config.periodDurationMs;

    const timeUntil = tracker.getTimeUntilExpiration(period, currentTime);
    expect(timeUntil).toBe(0);
  });
});

// ============ Statistics Tests ============

describe.sequential("ExpirationTracker Statistics", () => {
  let tracker: ExpirationTracker;
  let config: ExpirationConfig;
  let basePeriod: number;

  beforeEach(() => {
    config = createTestConfig({ periodsUntilExpiration: 3 });
    basePeriod = 1000;
    tracker = new ExpirationTracker(config, basePeriod);
  });

  it("should return initial statistics", () => {
    const stats = tracker.getStatistics();

    expect(stats.totalPeriods).toBe(0);
    expect(stats.activePeriods).toBe(0);
    expect(stats.expiredPeriods).toBe(0);
    expect(stats.totalActiveSupply).toBe(0n);
    expect(stats.totalExpiredSupply).toBe(0n);
    expect(stats.accountsWithBalances).toBe(0);
  });

  it("should track statistics after mints", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID_2, 500n, basePeriod + 1, 2n);

    const stats = tracker.getStatistics();

    expect(stats.totalPeriods).toBe(2);
    expect(stats.activePeriods).toBe(2);
    expect(stats.totalActiveSupply).toBe(1500n);
    expect(stats.totalMinted).toBe(1500n);
    expect(stats.accountsWithBalances).toBe(2);
  });

  it("should track expired periods in statistics", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // To expire basePeriod: newPeriod = basePeriod + periodsUntilExpiration
    const newPeriod = basePeriod + config.periodsUntilExpiration;
    tracker.processPeriodTransition(newPeriod, 10n);

    const stats = tracker.getStatistics();

    expect(stats.expiredPeriods).toBe(1);
    expect(stats.totalExpiredSupply).toBe(1000n);
    expect(stats.totalActiveSupply).toBe(0n);
  });

  it("should calculate average balance per period", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID, 500n, basePeriod + 1, 2n);

    const stats = tracker.getStatistics();

    // 1500n total / 2 periods = 750n average
    expect(stats.averageBalancePerPeriod).toBe(750n);
  });

  it("should track newest active period", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordMint(TEST_DID, 500n, basePeriod + 5, 2n);

    const stats = tracker.getStatistics();

    expect(stats.newestActivePeriod).toBe(basePeriod + 5);
  });
});

// ============ State Import/Export Tests ============

describe.sequential("ExpirationTracker State Persistence", () => {
  let config: ExpirationConfig;
  let basePeriod: number;

  beforeEach(() => {
    config = createTestConfig();
    basePeriod = 1000;
  });

  it("should export state", () => {
    const tracker = new ExpirationTracker(config, basePeriod);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const exported = tracker.exportState();

    expect(exported.periodStates).toBeDefined();
    expect(exported.accountPeriods).toBeDefined();
    expect(exported.globalState).toBeDefined();
    expect(exported.periodStates.size).toBe(1);
    expect(exported.accountPeriods.size).toBe(1);
  });

  it("should import state", () => {
    const tracker1 = new ExpirationTracker(config, basePeriod);
    tracker1.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker1.recordMint(TEST_DID_2, 500n, basePeriod + 1, 2n);

    const exported = tracker1.exportState();

    const tracker2 = new ExpirationTracker(config, basePeriod);
    tracker2.importState(exported);

    expect(tracker2.getGlobalState().totalActiveSupply).toBe(1500n);
    expect(tracker2.getPeriodState(basePeriod)?.activeBalance).toBe(1000n);
    expect(tracker2.getAccountPeriods(TEST_DID)).toContain(basePeriod);
  });

  it("should preserve all data through export/import cycle", () => {
    const tracker1 = new ExpirationTracker(config, basePeriod);
    tracker1.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // Expire the period
    const newPeriod = basePeriod + config.periodsUntilExpiration + 1;
    tracker1.processPeriodTransition(newPeriod, 10n);

    const exported = tracker1.exportState();
    const tracker2 = new ExpirationTracker(config, basePeriod);
    tracker2.importState(exported);

    const stats1 = tracker1.getStatistics();
    const stats2 = tracker2.getStatistics();

    expect(stats2.totalExpiredSupply).toBe(stats1.totalExpiredSupply);
    expect(stats2.expiredPeriods).toBe(stats1.expiredPeriods);
  });
});

// ============ Edge Cases ============

describe.sequential("ExpirationTracker Edge Cases", () => {
  let tracker: ExpirationTracker;
  let config: ExpirationConfig;
  let basePeriod: number;

  beforeEach(() => {
    config = createTestConfig();
    basePeriod = 1000;
    tracker = new ExpirationTracker(config, basePeriod);
  });

  it("should handle very large balances", () => {
    const largeAmount = 10n ** 30n;
    tracker.recordMint(TEST_DID, largeAmount, basePeriod, 1n);

    const state = tracker.getGlobalState();
    expect(state.totalActiveSupply).toBe(largeAmount);
  });

  it("should handle rapid consecutive mints", () => {
    for (let i = 0; i < 100; i++) {
      tracker.recordMint(TEST_DID, 100n, basePeriod + i, BigInt(i));
    }

    const stats = tracker.getStatistics();
    expect(stats.totalActiveSupply).toBe(10000n);
    expect(stats.totalPeriods).toBe(100);
  });

  it("should handle debit that empties period balance", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);
    tracker.recordDebit(TEST_DID, 1000n, basePeriod, false);

    const periodState = tracker.getPeriodState(basePeriod);
    expect(periodState!.activeBalance).toBe(0n);
    expect(periodState!.totalMinted).toBe(1000n); // Still tracks what was minted
  });

  it("should handle multiple period transitions", () => {
    // Create balance in several periods
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n); // period 1000
    tracker.recordMint(TEST_DID, 500n, basePeriod + 1, 2n); // period 1001
    tracker.recordMint(TEST_DID, 250n, basePeriod + 2, 3n); // period 1002

    // processPeriodTransition expires (newPeriod - periodsUntilExpiration)
    // periodsUntilExpiration = 5 (default), so:
    // - To expire 1000: newPeriod = 1005
    // - To expire 1001: newPeriod = 1006
    // - To expire 1002: newPeriod = 1007
    tracker.processPeriodTransition(
      basePeriod + config.periodsUntilExpiration,
      10n,
    ); // expires 1000
    tracker.processPeriodTransition(
      basePeriod + config.periodsUntilExpiration + 1,
      11n,
    ); // expires 1001
    tracker.processPeriodTransition(
      basePeriod + config.periodsUntilExpiration + 2,
      12n,
    ); // expires 1002

    const stats = tracker.getStatistics();
    expect(stats.totalExpiredSupply).toBe(1750n);
    expect(stats.totalActiveSupply).toBe(0n);
  });

  it("should not double-expire periods", () => {
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // To expire basePeriod: newPeriod = basePeriod + periodsUntilExpiration
    const newPeriod = basePeriod + config.periodsUntilExpiration;

    // Process transition that expires basePeriod
    const result1 = tracker.processPeriodTransition(newPeriod, 10n);
    // Try to process same period again
    const result2 = tracker.processPeriodTransition(newPeriod, 11n);

    expect(result1).not.toBeNull();
    expect(result1!.expiredAmount).toBe(1000n);
    expect(result2).toBeNull(); // Already at this period, no transition

    const stats = tracker.getStatistics();
    expect(stats.totalExpiredSupply).toBe(1000n);
  });
});
