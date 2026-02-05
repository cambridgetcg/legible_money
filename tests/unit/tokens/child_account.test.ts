/**
 * ChildAccountManager Unit Tests
 *
 * Tests for the per-period token balance management using the Dynamic Array method.
 * Implements the patent's approach of child accounts for each time period.
 *
 * PP HIERARCHY:
 *   TRUTH: Verify balance calculations are mathematically correct
 *   UNDERSTANDING: Test all balance operations (credit, debit, expire)
 *   BEAUTY: Clear test structure mirroring manager methods
 *   JUSTICE: Tests in proper tokens unit test location
 */

import { describe, it, expect, beforeEach } from "vitest";
import { ChildAccountManager } from "../../../core/tokens/child_account.js";
import {
  DEFAULT_EXPIRATION_CONFIG,
  DEFAULT_TRANSFER_POLICY,
  TokenError,
  type ExpirationConfig,
  type AccountType,
  type AccountTypeConfig,
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
    periodsUntilExpiration: 5, // 5 seconds until expiration
    tokenLifespanMs: 5000,
    ...overrides,
  };
}

// ============ Initialization Tests ============

describe.sequential("ChildAccountManager Initialization", () => {
  it("should initialize with default config", () => {
    const manager = new ChildAccountManager();
    expect(manager).toBeDefined();
  });

  it("should initialize with custom config", () => {
    const config = createTestConfig({ dustThreshold: 100n });
    const manager = new ChildAccountManager(config);
    expect(manager).toBeDefined();
  });

  it("should return zero balance for non-existent account", () => {
    const manager = new ChildAccountManager(createTestConfig());
    const balance = manager.getBalance(TEST_DID, Date.now());
    expect(balance).toBe(0n);
  });
});

// ============ Balance Credit Tests ============

describe.sequential("ChildAccountManager Credit Operations", () => {
  let manager: ChildAccountManager;
  let config: ExpirationConfig;
  let baseTime: number;

  beforeEach(() => {
    // Use low dust threshold to allow smaller test amounts
    config = createTestConfig({ dustThreshold: 1n });
    manager = new ChildAccountManager(config);
    baseTime = Date.now();
  });

  it("should credit tokens to account", () => {
    const result = manager.credit(TEST_DID, 1000n, baseTime);

    expect(result.period).toBeGreaterThan(0);
    expect(result.childAccount.balance).toBe(1000n);

    const balance = manager.getBalance(TEST_DID, baseTime);
    expect(balance).toBe(1000n);
  });

  it("should accumulate multiple credits in same period", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.credit(TEST_DID, 500n, baseTime);

    const balance = manager.getBalance(TEST_DID, baseTime);
    expect(balance).toBe(1500n);
  });

  it("should create separate child accounts for different periods", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    // Credit in next period
    manager.credit(TEST_DID, 500n, baseTime + config.periodDurationMs);

    const balance = manager.getBalance(
      TEST_DID,
      baseTime + config.periodDurationMs,
    );
    expect(balance).toBe(1500n);

    const breakdown = manager.getBalanceBreakdown(
      TEST_DID,
      baseTime + config.periodDurationMs,
    );
    expect(breakdown.active.size).toBe(2);
  });

  it("should track earliest and latest periods", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.credit(TEST_DID, 500n, baseTime + config.periodDurationMs * 2);

    const expBalance = manager.getOrCreateBalance(TEST_DID);
    expect(expBalance.latestPeriod).toBeGreaterThan(expBalance.earliestPeriod);
  });

  it("should reject dust amounts", () => {
    const dustConfig = createTestConfig({ dustThreshold: 1000n });
    const dustManager = new ChildAccountManager(dustConfig);

    expect(() => dustManager.credit(TEST_DID, 500n, baseTime)).toThrow(
      TokenError,
    );
  });

  it("should accept amounts at dust threshold", () => {
    const dustConfig = createTestConfig({ dustThreshold: 1000n });
    const dustManager = new ChildAccountManager(dustConfig);

    expect(() => dustManager.credit(TEST_DID, 1000n, baseTime)).not.toThrow();
    expect(dustManager.getBalance(TEST_DID, baseTime)).toBe(1000n);
  });
});

// ============ Balance Debit Tests ============

describe.sequential("ChildAccountManager Debit Operations", () => {
  let manager: ChildAccountManager;
  let config: ExpirationConfig;
  let baseTime: number;

  beforeEach(() => {
    // Use low dust threshold for smaller test amounts
    config = createTestConfig({ dustThreshold: 1n });
    manager = new ChildAccountManager(config);
    baseTime = Date.now();
  });

  it("should debit tokens from account", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    const result = manager.debit(TEST_DID, 300n, baseTime);

    // Sum periodBreakdown to get total debited
    let totalDebited = 0n;
    for (const amount of result.periodBreakdown.values()) {
      totalDebited += amount;
    }
    expect(totalDebited).toBe(300n);
    expect(manager.getBalance(TEST_DID, baseTime)).toBe(700n);
  });

  it("should return period breakdown for debit", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    const result = manager.debit(TEST_DID, 300n, baseTime);

    expect(result.periodBreakdown.size).toBeGreaterThan(0);
    let totalFromPeriods = 0n;
    for (const amount of result.periodBreakdown.values()) {
      totalFromPeriods += amount;
    }
    expect(totalFromPeriods).toBe(300n);
  });

  it("should use LIFO (newest first) by default", () => {
    // Credit in period 1
    manager.credit(TEST_DID, 1000n, baseTime);
    // Credit in period 3 (newer)
    manager.credit(TEST_DID, 500n, baseTime + config.periodDurationMs * 2);

    // Debit should take from period 3 first
    manager.debit(TEST_DID, 400n, baseTime + config.periodDurationMs * 2);

    // The newer period should have been debited
    const breakdown = manager.getBalanceBreakdown(
      TEST_DID,
      baseTime + config.periodDurationMs * 2,
    );
    let newestPeriodBalance = 0n;
    for (const [period, amount] of breakdown.active) {
      if (
        period ===
        Math.floor(
          (baseTime + config.periodDurationMs * 2) / config.periodDurationMs,
        )
      ) {
        newestPeriodBalance = amount;
      }
    }
    expect(newestPeriodBalance).toBe(100n); // 500 - 400
  });

  it("should debit across multiple periods if needed", () => {
    manager.credit(TEST_DID, 300n, baseTime);
    manager.credit(TEST_DID, 300n, baseTime + config.periodDurationMs);

    const result = manager.debit(
      TEST_DID,
      500n,
      baseTime + config.periodDurationMs,
    );

    // Sum periodBreakdown to verify total debited
    let totalDebited = 0n;
    for (const amount of result.periodBreakdown.values()) {
      totalDebited += amount;
    }
    expect(totalDebited).toBe(500n);
    expect(result.periodBreakdown.size).toBe(2);
    expect(
      manager.getBalance(TEST_DID, baseTime + config.periodDurationMs),
    ).toBe(100n);
  });

  it("should throw INSUFFICIENT_BALANCE when not enough", () => {
    manager.credit(TEST_DID, 1000n, baseTime);

    expect(() => manager.debit(TEST_DID, 2000n, baseTime)).toThrow(TokenError);
  });

  it("should throw ACCOUNT_NOT_FOUND for non-existent account", () => {
    expect(() => manager.debit(TEST_DID, 100n, baseTime)).toThrow(TokenError);
  });

  it("should handle exact balance debit", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    const result = manager.debit(TEST_DID, 1000n, baseTime);

    // Sum periodBreakdown to verify total debited
    let totalDebited = 0n;
    for (const amount of result.periodBreakdown.values()) {
      totalDebited += amount;
    }
    expect(totalDebited).toBe(1000n);
    expect(manager.getBalance(TEST_DID, baseTime)).toBe(0n);
  });
});

// ============ Expiration Tests ============

describe.sequential("ChildAccountManager Expiration", () => {
  let manager: ChildAccountManager;
  let config: ExpirationConfig;
  let baseTime: number;

  beforeEach(() => {
    config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n, // Allow smaller test amounts
    });
    manager = new ChildAccountManager(config);
    baseTime = Date.now();
  });

  it("should return full balance before expiration", () => {
    manager.credit(TEST_DID, 1000n, baseTime);

    // Check balance within lifespan
    const balance = manager.getBalance(TEST_DID, baseTime + 100);
    expect(balance).toBe(1000n);
  });

  it("should exclude expired balance", () => {
    manager.credit(TEST_DID, 1000n, baseTime);

    // Check balance after expiration
    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    const balance = manager.getBalance(TEST_DID, expiredTime);
    expect(balance).toBe(0n);
  });

  it("should process expiration and return expired amount", () => {
    manager.credit(TEST_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    const result = manager.processExpiration(TEST_DID, expiredTime);

    expect(result.expiredAmount).toBe(1000n);
    expect(result.expiredPeriods.length).toBeGreaterThan(0);
  });

  it("should track multiple periods with different expiration times", () => {
    // Credit at different times
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.credit(TEST_DID, 500n, baseTime + config.periodDurationMs);
    manager.credit(TEST_DID, 250n, baseTime + config.periodDurationMs * 2);

    // Check when only first period expires
    const partialExpireTime = baseTime + config.tokenLifespanMs + 50;
    const breakdown = manager.getBalanceBreakdown(TEST_DID, partialExpireTime);

    // First period should be expired, others active
    expect(breakdown.expired.size).toBeGreaterThanOrEqual(1);
  });

  it("should identify expiring-soon balance", () => {
    manager.credit(TEST_DID, 1000n, baseTime);

    // Time when balance is about to expire
    const nearExpiryTime =
      baseTime + config.tokenLifespanMs - config.periodDurationMs;
    const breakdown = manager.getBalanceBreakdown(TEST_DID, nearExpiryTime);

    expect(breakdown.expiringSoon.size).toBeGreaterThan(0);
  });
});

// ============ Balance Breakdown Tests ============

describe.sequential("ChildAccountManager Balance Breakdown", () => {
  let manager: ChildAccountManager;
  let config: ExpirationConfig;
  let baseTime: number;

  beforeEach(() => {
    config = createTestConfig({ dustThreshold: 1n });
    manager = new ChildAccountManager(config);
    baseTime = Date.now();
  });

  it("should return active balances by period", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.credit(TEST_DID, 500n, baseTime + config.periodDurationMs);

    const breakdown = manager.getBalanceBreakdown(
      TEST_DID,
      baseTime + config.periodDurationMs,
    );

    expect(breakdown.active.size).toBe(2);
    let total = 0n;
    for (const amount of breakdown.active.values()) {
      total += amount;
    }
    expect(total).toBe(1500n);
  });

  it("should return empty breakdown for non-existent account", () => {
    const breakdown = manager.getBalanceBreakdown(TEST_DID, baseTime);

    expect(breakdown.active.size).toBe(0);
    expect(breakdown.expired.size).toBe(0);
    expect(breakdown.expiringSoon.size).toBe(0);
  });
});

// ============ Extension Tests ============

describe.sequential("ChildAccountManager Extension", () => {
  let manager: ChildAccountManager;
  let config: ExpirationConfig;
  let baseTime: number;

  beforeEach(() => {
    config = createTestConfig({
      allowExtension: true,
      maxExtensionPeriods: 10,
      dustThreshold: 1n,
    });
    manager = new ChildAccountManager(config);
    baseTime = Date.now();
  });

  it("should extend expiration when allowed", () => {
    manager.credit(TEST_DID, 1000n, baseTime);

    const result = manager.extendExpiration(TEST_DID, 500n, 5, baseTime);

    // `extended` is the amount that was extended (bigint)
    expect(result.extended).toBe(500n);
    expect(result.newPeriod).toBeGreaterThan(0);
    expect(result.oldPeriods.length).toBeGreaterThan(0);
  });

  it("should reject extension when not allowed", () => {
    const noExtConfig = createTestConfig({
      allowExtension: false,
      dustThreshold: 1n,
    });
    const noExtManager = new ChildAccountManager(noExtConfig);
    noExtManager.credit(TEST_DID, 1000n, baseTime);

    expect(() =>
      noExtManager.extendExpiration(TEST_DID, 500n, 5, baseTime),
    ).toThrow(TokenError);
  });

  it("should reject extension beyond maximum", () => {
    manager.credit(TEST_DID, 1000n, baseTime);

    expect(() =>
      manager.extendExpiration(TEST_DID, 500n, 50, baseTime),
    ).toThrow(TokenError);
  });
});

// ============ Statistics Tests ============

describe.sequential("ChildAccountManager Statistics", () => {
  let manager: ChildAccountManager;
  let config: ExpirationConfig;
  let baseTime: number;

  beforeEach(() => {
    config = createTestConfig({ dustThreshold: 1n });
    manager = new ChildAccountManager(config);
    baseTime = Date.now();
  });

  it("should track total accounts", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.credit(TEST_DID_2, 500n, baseTime);

    const stats = manager.getStats(baseTime);
    expect(stats.totalAccounts).toBe(2);
  });

  it("should track total active balance", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.credit(TEST_DID_2, 500n, baseTime);

    const stats = manager.getStats(baseTime);
    expect(stats.totalActiveBalance).toBe(1500n);
  });

  it("should track child account count", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.credit(TEST_DID, 500n, baseTime + config.periodDurationMs);

    const stats = manager.getStats(baseTime + config.periodDurationMs);
    expect(stats.totalChildAccounts).toBeGreaterThanOrEqual(2);
  });
});

// ============ Export/Import Tests ============

describe.sequential("ChildAccountManager State Persistence", () => {
  let config: ExpirationConfig;
  let baseTime: number;

  beforeEach(() => {
    config = createTestConfig({ dustThreshold: 1n });
    baseTime = Date.now();
  });

  it("should export all balances", () => {
    const manager = new ChildAccountManager(config);
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.credit(TEST_DID_2, 500n, baseTime);

    const balances = manager.getAllBalances();
    // getAllBalances returns a Map, check size
    expect(balances.size).toBe(2);
  });

  it("should export child accounts", () => {
    const manager = new ChildAccountManager(config);
    manager.credit(TEST_DID, 1000n, baseTime);

    const childAccounts = manager.getAllChildAccounts();
    // getAllChildAccounts returns a Map, check size
    expect(childAccounts.size).toBeGreaterThan(0);
    // Get first entry to check parentAccount
    const firstChild = childAccounts.values().next().value;
    expect(firstChild.parentAccount).toBe(TEST_DID);
  });

  it("should get child accounts for specific account", () => {
    const manager = new ChildAccountManager(config);
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.credit(TEST_DID, 500n, baseTime + config.periodDurationMs);

    // getChildAccountsFor returns an array
    const childAccounts = manager.getChildAccountsFor(TEST_DID);
    expect(childAccounts.length).toBeGreaterThanOrEqual(2);
  });
});

// ============ Retrieval Rate Tests ============

describe.sequential("ChildAccountManager Retrieval Rate", () => {
  let baseTime: number;

  beforeEach(() => {
    baseTime = Date.now();
  });

  it("should return zero fee when retrieval rate is 0", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    expect(manager.calculateRetrievalFee(1000n)).toBe(0n);
  });

  it("should calculate correct fee at 5%", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0.05,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    expect(manager.calculateRetrievalFee(1000n)).toBe(50n);
  });

  it("should calculate correct fee at 1%", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0.01,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    expect(manager.calculateRetrievalFee(10000n)).toBe(100n);
  });

  it("should floor fractional fee amounts", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0.05,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    // 5% of 33 = 1.65, floor = 1
    expect(manager.calculateRetrievalFee(33n)).toBe(1n);
  });

  it("should return zero fee for dust amount with low rate", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0.05,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    // 5% of 1 = 0.05, floor = 0
    expect(manager.calculateRetrievalFee(1n)).toBe(0n);
  });

  it("should transferWithFee debiting gross and crediting net", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0.05,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);
    const result = manager.transferWithFee(
      TEST_DID,
      TEST_DID_2,
      1000n,
      baseTime,
    );

    expect(result.grossAmount).toBe(1000n);
    expect(result.fee).toBe(50n);
    expect(result.netAmount).toBe(950n);

    // Sender fully debited
    expect(manager.getBalance(TEST_DID, baseTime)).toBe(0n);
    // Recipient gets net
    expect(manager.getBalance(TEST_DID_2, baseTime)).toBe(950n);
  });

  it("should skip fee when applyRate is false", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0.05,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);
    const result = manager.transferWithFee(
      TEST_DID,
      TEST_DID_2,
      1000n,
      baseTime,
      false,
    );

    expect(result.fee).toBe(0n);
    expect(result.netAmount).toBe(1000n);
    expect(manager.getBalance(TEST_DID_2, baseTime)).toBe(1000n);
  });

  it("should transfer with fee across multiple periods (LIFO)", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0.1,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 500n, baseTime);
    manager.credit(TEST_DID, 500n, baseTime + config.periodDurationMs);

    const result = manager.transferWithFee(
      TEST_DID,
      TEST_DID_2,
      800n,
      baseTime + config.periodDurationMs,
    );

    expect(result.grossAmount).toBe(800n);
    expect(result.fee).toBe(80n); // 10% of 800
    expect(result.netAmount).toBe(720n);
    expect(result.periodBreakdown.size).toBeGreaterThanOrEqual(1);
    expect(
      manager.getBalance(TEST_DID_2, baseTime + config.periodDurationMs),
    ).toBe(720n);
  });

  it("should charge fee on self-transfer", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0.05,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);
    const result = manager.transferWithFee(TEST_DID, TEST_DID, 500n, baseTime);

    expect(result.fee).toBe(25n);
    // Sender debited 500, credited 475 back → net 975
    expect(manager.getBalance(TEST_DID, baseTime)).toBe(975n);
  });
});

// ============ Lock and Unlock Tests ============

describe.sequential("ChildAccountManager Lock and Unlock", () => {
  let baseTime: number;

  beforeEach(() => {
    baseTime = Date.now();
  });

  it("should lock expired tokens instead of sweeping in lock_and_unlock mode", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0.05,
      expirationMode: "lock_and_unlock",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    const result = manager.processExpiration(TEST_DID, expiredTime);

    // Should lock, not sweep
    expect(result.lockedAmount).toBe(1000n);
    expect(result.lockedPeriods.length).toBeGreaterThan(0);
    expect(result.expiredAmount).toBe(0n);
    expect(result.expiredPeriods.length).toBe(0);
  });

  it("should not reduce totalBalance for locked tokens", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0,
      expirationMode: "lock_and_unlock",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    manager.processExpiration(TEST_DID, expiredTime);

    // totalBalance should still include locked tokens
    const balance = manager.getOrCreateBalance(TEST_DID);
    expect(balance.totalBalance).toBe(1000n);
  });

  it("should exclude locked tokens from getBalance()", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0,
      expirationMode: "lock_and_unlock",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    manager.processExpiration(TEST_DID, expiredTime);

    // getBalance dynamically computes active only, locked periods are past expiration threshold
    expect(manager.getBalance(TEST_DID, expiredTime)).toBe(0n);
  });

  it("should show locked in getBalanceBreakdown()", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0,
      expirationMode: "lock_and_unlock",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    manager.processExpiration(TEST_DID, expiredTime);

    const breakdown = manager.getBalanceBreakdown(TEST_DID, expiredTime);
    expect(breakdown.totalLocked).toBe(1000n);
    expect(breakdown.locked.size).toBeGreaterThan(0);
    expect(breakdown.totalExpired).toBe(0n);
  });

  it("should unlock expired tokens and move to fresh period", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0.1,
      expirationMode: "lock_and_unlock",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    manager.processExpiration(TEST_DID, expiredTime);

    const result = manager.unlockExpired(TEST_DID, expiredTime);

    expect(result.unlocked).toBe(1000n);
    expect(result.renewalFee).toBe(100n); // 10% of 1000
    expect(result.netUnlocked).toBe(900n);
    expect(result.unlockedPeriods.length).toBeGreaterThan(0);

    // Active balance should now be the net unlocked amount
    expect(manager.getBalance(TEST_DID, expiredTime)).toBe(900n);
  });

  it("should use renewalFeeRate when set", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0.1,
      renewalFeeRate: 0.02,
      expirationMode: "lock_and_unlock",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    manager.processExpiration(TEST_DID, expiredTime);

    const result = manager.unlockExpired(TEST_DID, expiredTime);

    expect(result.renewalFee).toBe(20n); // 2% of 1000 (uses renewalFeeRate, not retrievalRate)
    expect(result.netUnlocked).toBe(980n);
  });

  it("should throw in direct_retrieval mode when unlocking", () => {
    const config = createTestConfig({
      dustThreshold: 1n,
      retrievalRate: 0.1,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    expect(() => manager.unlockExpired(TEST_DID, baseTime)).toThrow(TokenError);
  });

  it("should throw when no locked balance exists", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0.1,
      expirationMode: "lock_and_unlock",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    // Not expired yet, so no locked balance
    expect(() => manager.unlockExpired(TEST_DID, baseTime)).toThrow(TokenError);
  });

  it("should unlock specific periods only", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0.1,
      expirationMode: "lock_and_unlock",
    });
    const manager = new ChildAccountManager(config);

    // Credit in two different periods
    manager.credit(TEST_DID, 500n, baseTime);
    manager.credit(TEST_DID, 500n, baseTime + config.periodDurationMs);

    // Expire both
    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs * 2;
    const expResult = manager.processExpiration(TEST_DID, expiredTime);

    // Unlock only the first locked period
    const firstLockedPeriod = expResult.lockedPeriods[0];
    const result = manager.unlockExpired(TEST_DID, expiredTime, [
      firstLockedPeriod,
    ]);

    expect(result.unlockedPeriods).toEqual([firstLockedPeriod]);
    // Some tokens should still be locked
    expect(manager.getLockedBalance(TEST_DID, expiredTime)).toBeGreaterThan(0n);
  });

  it("should report locked balance via getLockedBalance()", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0,
      expirationMode: "lock_and_unlock",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    manager.processExpiration(TEST_DID, expiredTime);

    expect(manager.getLockedBalance(TEST_DID, expiredTime)).toBe(1000n);
  });

  it("should preserve direct_retrieval behavior unchanged", () => {
    const config = createTestConfig({
      periodDurationMs: 100,
      periodsUntilExpiration: 3,
      tokenLifespanMs: 300,
      dustThreshold: 1n,
      retrievalRate: 0,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(config);

    manager.credit(TEST_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + config.tokenLifespanMs + config.periodDurationMs;
    const result = manager.processExpiration(TEST_DID, expiredTime);

    expect(result.expiredAmount).toBe(1000n);
    expect(result.lockedAmount).toBe(0n);
    expect(manager.getBalance(TEST_DID, expiredTime)).toBe(0n);
  });
});

// ============ Edge Cases ============

describe.sequential("ChildAccountManager Edge Cases", () => {
  let manager: ChildAccountManager;
  let config: ExpirationConfig;
  let baseTime: number;

  beforeEach(() => {
    config = createTestConfig({ dustThreshold: 1n });
    manager = new ChildAccountManager(config);
    baseTime = Date.now();
  });

  it("should handle very large balances", () => {
    const largeAmount = 10n ** 30n;
    manager.credit(TEST_DID, largeAmount, baseTime);

    expect(manager.getBalance(TEST_DID, baseTime)).toBe(largeAmount);
  });

  it("should handle rapid consecutive operations", () => {
    for (let i = 0; i < 100; i++) {
      manager.credit(TEST_DID, 100n, baseTime + i);
    }

    expect(manager.getBalance(TEST_DID, baseTime + 100)).toBe(10000n);
  });

  it("should handle zero-balance after full debit", () => {
    manager.credit(TEST_DID, 1000n, baseTime);
    manager.debit(TEST_DID, 1000n, baseTime);

    // Should be able to credit again
    manager.credit(TEST_DID, 500n, baseTime);
    expect(manager.getBalance(TEST_DID, baseTime)).toBe(500n);
  });

  it("should handle accounts at exact period boundaries", () => {
    const exactBoundary =
      Math.floor(baseTime / config.periodDurationMs) * config.periodDurationMs;
    manager.credit(TEST_DID, 1000n, exactBoundary);

    expect(manager.getBalance(TEST_DID, exactBoundary)).toBe(1000n);
  });
});

// ============ Account Type Configuration Tests ============

describe.sequential("ChildAccountManager Account Type Configuration", () => {
  const AGENT_DID = "did:lgm:agent-type-test-12345";
  const VALIDATOR_DID = "did:lgm:validator-type-test-67890";
  const SYSTEM_DID = "did:lgm:system-type-test-99999";

  let baseTime: number;
  let config: ExpirationConfig;

  beforeEach(() => {
    baseTime = Date.now();
    config = createTestConfig({
      dustThreshold: 1n,
      periodsUntilExpiration: 5,
      retrievalRate: 0.02,
      expirationMode: "direct_retrieval",
    });
  });

  it("should register and retrieve account type", () => {
    const manager = new ChildAccountManager(config);
    manager.setAccountType(AGENT_DID, "agent");
    expect(manager.getAccountType(AGENT_DID)).toBe("agent");
  });

  it("should return undefined for unregistered account type", () => {
    const manager = new ChildAccountManager(config);
    expect(manager.getAccountType(AGENT_DID)).toBeUndefined();
  });

  it("should set and get type config", () => {
    const manager = new ChildAccountManager(config);
    const typeConfig: AccountTypeConfig = { periodsUntilExpiration: 20 };
    manager.setTypeConfig("contract", typeConfig);
    expect(manager.getTypeConfig("contract")).toEqual(typeConfig);
  });

  it("should reject type config with periodDurationMs", () => {
    const manager = new ChildAccountManager(config);
    expect(() =>
      manager.setTypeConfig("agent", { periodDurationMs: 2000 } as any),
    ).toThrow(TokenError);
  });

  it("should use global config when no type set", () => {
    const manager = new ChildAccountManager(config);
    manager.credit(AGENT_DID, 1000n, baseTime);

    // Global: 5 periods until expiration → expired after 6 periods
    const expiredTime = baseTime + config.periodDurationMs * 6;
    expect(manager.getBalance(AGENT_DID, expiredTime)).toBe(0n);
  });

  it("should use type config override for validator with longer expiration", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      ["validator", { periodsUntilExpiration: 10 }],
    ]);
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
    );

    manager.setAccountType(AGENT_DID, "agent");
    manager.setAccountType(VALIDATOR_DID, "validator");

    manager.credit(AGENT_DID, 1000n, baseTime);
    manager.credit(VALIDATOR_DID, 1000n, baseTime);

    // After 6 periods: agent expired (5 limit), validator still active (10 limit)
    const checkTime = baseTime + config.periodDurationMs * 6;
    expect(manager.getBalance(AGENT_DID, checkTime)).toBe(0n);
    expect(manager.getBalance(VALIDATOR_DID, checkTime)).toBe(1000n);
  });

  it("should apply different retrieval rates per type", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      ["validator", { retrievalRate: 0.01 }],
    ]);
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
    );

    manager.setAccountType(AGENT_DID, "agent");
    manager.setAccountType(VALIDATOR_DID, "validator");

    // Agent: 2% (global)
    expect(manager.calculateRetrievalFee(10000n, AGENT_DID)).toBe(200n);
    // Validator: 1% (override)
    expect(manager.calculateRetrievalFee(10000n, VALIDATOR_DID)).toBe(100n);
  });

  it("should use different expiration mode per type", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      [
        "system",
        {
          periodsUntilExpiration: 3,
          expirationMode: "lock_and_unlock" as const,
        },
      ],
    ]);
    const localConfig = createTestConfig({
      dustThreshold: 1n,
      periodsUntilExpiration: 3,
      periodDurationMs: 100,
      tokenLifespanMs: 300,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(
      localConfig,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
    );

    manager.setAccountType(AGENT_DID, "agent");
    manager.setAccountType(SYSTEM_DID, "system");

    manager.credit(AGENT_DID, 1000n, baseTime);
    manager.credit(SYSTEM_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + localConfig.tokenLifespanMs + localConfig.periodDurationMs;

    // Agent: direct_retrieval → swept
    const agentResult = manager.processExpiration(AGENT_DID, expiredTime);
    expect(agentResult.expiredAmount).toBe(1000n);
    expect(agentResult.lockedAmount).toBe(0n);

    // System: lock_and_unlock → locked
    const systemResult = manager.processExpiration(SYSTEM_DID, expiredTime);
    expect(systemResult.lockedAmount).toBe(1000n);
    expect(systemResult.expiredAmount).toBe(0n);
  });

  it("should respect per-type config in getAccountsWithExpiredBalances", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      ["validator", { periodsUntilExpiration: 10 }],
    ]);
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
    );

    manager.setAccountType(AGENT_DID, "agent");
    manager.setAccountType(VALIDATOR_DID, "validator");

    manager.credit(AGENT_DID, 1000n, baseTime);
    manager.credit(VALIDATOR_DID, 1000n, baseTime);

    const checkTime = baseTime + config.periodDurationMs * 6;
    const expired = manager.getAccountsWithExpiredBalances(checkTime);

    expect(expired).toContain(AGENT_DID);
    expect(expired).not.toContain(VALIDATOR_DID);
  });

  it("should respect per-type config in getTotalExpiredForPeriod", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      ["validator", { periodsUntilExpiration: 10 }],
    ]);
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
    );

    manager.setAccountType(AGENT_DID, "agent");
    manager.setAccountType(VALIDATOR_DID, "validator");

    const period = Math.floor(baseTime / config.periodDurationMs);
    manager.credit(AGENT_DID, 1000n, baseTime);
    manager.credit(VALIDATOR_DID, 2000n, baseTime);

    // After 6 periods: only agent's tokens expired
    const checkTime = baseTime + config.periodDurationMs * 6;
    const totalExpired = manager.getTotalExpiredForPeriod(period, checkTime);
    expect(totalExpired).toBe(1000n);
  });

  it("should respect type config in extendExpiration", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      ["validator", { allowExtension: true, maxExtensionPeriods: 20 }],
    ]);
    const localConfig = createTestConfig({
      dustThreshold: 1n,
      allowExtension: false,
    });
    const manager = new ChildAccountManager(
      localConfig,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
    );

    manager.setAccountType(VALIDATOR_DID, "validator");
    manager.credit(VALIDATOR_DID, 1000n, baseTime);

    // Global config disallows extension, but validator override allows it
    const result = manager.extendExpiration(VALIDATOR_DID, 500n, 10, baseTime);
    expect(result.extended).toBe(500n);
  });

  it("should respect type config in unlockExpired", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      [
        "system",
        {
          periodsUntilExpiration: 3,
          expirationMode: "lock_and_unlock" as const,
          retrievalRate: 0.05,
        },
      ],
    ]);
    const localConfig = createTestConfig({
      dustThreshold: 1n,
      periodsUntilExpiration: 3,
      periodDurationMs: 100,
      tokenLifespanMs: 300,
      expirationMode: "direct_retrieval",
    });
    const manager = new ChildAccountManager(
      localConfig,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
    );

    manager.setAccountType(SYSTEM_DID, "system");
    manager.credit(SYSTEM_DID, 1000n, baseTime);

    const expiredTime =
      baseTime + localConfig.tokenLifespanMs + localConfig.periodDurationMs;
    manager.processExpiration(SYSTEM_DID, expiredTime);

    const result = manager.unlockExpired(SYSTEM_DID, expiredTime);
    expect(result.unlocked).toBe(1000n);
    expect(result.renewalFee).toBe(50n); // 5% of 1000
    expect(result.netUnlocked).toBe(950n);
  });

  it("should handle isPeriodExpired with account parameter", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      ["validator", { periodsUntilExpiration: 10 }],
    ]);
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
    );

    manager.setAccountType(AGENT_DID, "agent");
    manager.setAccountType(VALIDATOR_DID, "validator");

    const period = Math.floor(baseTime / config.periodDurationMs);
    const checkTime = baseTime + config.periodDurationMs * 6;

    expect(manager.isPeriodExpired(period, checkTime, AGENT_DID)).toBe(true);
    expect(manager.isPeriodExpired(period, checkTime, VALIDATOR_DID)).toBe(
      false,
    );
  });

  it("should handle periodToExpiration with account parameter", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      ["validator", { periodsUntilExpiration: 10 }],
    ]);
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
    );

    manager.setAccountType(AGENT_DID, "agent");
    manager.setAccountType(VALIDATOR_DID, "validator");

    const period = 100;
    // Agent: (100 + 5) * 1000 = 105000
    expect(manager.periodToExpiration(period, AGENT_DID)).toBe(
      105 * config.periodDurationMs,
    );
    // Validator: (100 + 10) * 1000 = 110000
    expect(manager.periodToExpiration(period, VALIDATOR_DID)).toBe(
      110 * config.periodDurationMs,
    );
  });

  it("should preserve backward compat with no type set for all operations", () => {
    const manager = new ChildAccountManager(config);
    manager.credit(AGENT_DID, 1000n, baseTime);

    expect(manager.getBalance(AGENT_DID, baseTime)).toBe(1000n);
    expect(manager.getBalanceBreakdown(AGENT_DID, baseTime).totalActive).toBe(
      1000n,
    );

    manager.debit(AGENT_DID, 200n, baseTime);
    expect(manager.getBalance(AGENT_DID, baseTime)).toBe(800n);

    // No type = global config
    expect(manager.calculateRetrievalFee(1000n)).toBe(20n); // 2% global rate
    expect(manager.isPeriodExpired(0, baseTime)).toBe(true);
  });
});

// ============ Participation Resolver Integration Tests ============

describe.sequential("ChildAccountManager Participation Resolver", () => {
  const ACTIVE_DID = "did:lgm:active-participant-0123456789abcdef";
  const INACTIVE_DID = "did:lgm:inactive-user-fedcba9876543210";
  const UNKNOWN_DID = "did:lgm:unknown-user-00112233445566778899";

  const config = createTestConfig({
    dustThreshold: 1n,
    periodsUntilExpiration: 26, // Global default
  });
  const baseTime = config.periodDurationMs * 100;

  function makeResolver(periodsMap: Map<string, number>) {
    return (account: string) => {
      const periods = periodsMap.get(account);
      if (periods === undefined) return null;
      return {
        score: periods * 5, // Approximate
        components: {
          claims: 0,
          verifications: 0,
          queries: 0,
          governance: 0,
          stake: 0,
        },
        decayFactor: 1,
        periodsGranted: periods,
      };
    };
  }

  it("should behave identically without resolver (backward compat)", () => {
    const manager = new ChildAccountManager(config);
    manager.credit(ACTIVE_DID, 1000n, baseTime);

    // Global: 26 periods until expiration
    const stillActive = baseTime + config.periodDurationMs * 25;
    expect(manager.getBalance(ACTIVE_DID, stillActive)).toBe(1000n);

    const justExpired = baseTime + config.periodDurationMs * 27;
    expect(manager.getBalance(ACTIVE_DID, justExpired)).toBe(0n);
  });

  it("should override periodsUntilExpiration via resolver", () => {
    const resolver = makeResolver(new Map([[ACTIVE_DID, 3]]));
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      undefined,
      resolver,
    );

    manager.credit(ACTIVE_DID, 1000n, baseTime);

    // Resolver says 3 periods, not 26
    // At exactly 3 periods: period 100 <= 103 - 3 = 100 → boundary (expired)
    // At 2 periods: still active
    const stillActive = baseTime + config.periodDurationMs * 2;
    expect(manager.getBalance(ACTIVE_DID, stillActive)).toBe(1000n);

    const expired = baseTime + config.periodDurationMs * 4;
    expect(manager.getBalance(ACTIVE_DID, expired)).toBe(0n);
  });

  it("should fall back to global config when resolver returns null", () => {
    const resolver = makeResolver(new Map([[ACTIVE_DID, 3]]));
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      undefined,
      resolver,
    );

    // UNKNOWN_DID not in resolver map → returns null → global config (26 periods)
    manager.credit(UNKNOWN_DID, 1000n, baseTime);

    const after4 = baseTime + config.periodDurationMs * 4;
    expect(manager.getBalance(UNKNOWN_DID, after4)).toBe(1000n); // Still active (global = 26)
  });

  it("should prioritize resolver over type config", () => {
    const typeConfigs = new Map<AccountType, AccountTypeConfig>([
      ["agent", { periodsUntilExpiration: 20 }],
    ]);
    const resolver = makeResolver(new Map([[ACTIVE_DID, 5]]));
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      typeConfigs,
      resolver,
    );

    manager.setAccountType(ACTIVE_DID, "agent");
    manager.credit(ACTIVE_DID, 1000n, baseTime);

    // Type config says 20, resolver says 5 → resolver wins
    const after6 = baseTime + config.periodDurationMs * 6;
    expect(manager.getBalance(ACTIVE_DID, after6)).toBe(0n);
  });

  it("should work with setParticipationResolver after construction", () => {
    const manager = new ChildAccountManager(config);
    manager.credit(ACTIVE_DID, 1000n, baseTime);

    // Without resolver: global config (26 periods)
    const after4 = baseTime + config.periodDurationMs * 4;
    expect(manager.getBalance(ACTIVE_DID, after4)).toBe(1000n);

    // Now set resolver with 3 periods
    const resolver = makeResolver(new Map([[ACTIVE_DID, 3]]));
    manager.setParticipationResolver(resolver);

    // Same time: now expired (3 < 4)
    expect(manager.getBalance(ACTIVE_DID, after4)).toBe(0n);
  });

  it("should respect resolver in processExpiration", () => {
    const resolver = makeResolver(new Map([[ACTIVE_DID, 3]]));
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      undefined,
      resolver,
    );

    manager.credit(ACTIVE_DID, 1000n, baseTime);

    const after4 = baseTime + config.periodDurationMs * 4;
    const result = manager.processExpiration(ACTIVE_DID, after4);
    expect(result.expiredAmount).toBe(1000n);
  });

  it("should respect resolver in getAccountsWithExpiredBalances", () => {
    const resolver = makeResolver(
      new Map([
        [ACTIVE_DID, 3],
        [INACTIVE_DID, 10],
      ]),
    );
    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      undefined,
      resolver,
    );

    manager.credit(ACTIVE_DID, 1000n, baseTime);
    manager.credit(INACTIVE_DID, 1000n, baseTime);

    // After 4 periods: ACTIVE (3 periods) expired, INACTIVE (10 periods) not
    const after4 = baseTime + config.periodDurationMs * 4;
    const expired = manager.getAccountsWithExpiredBalances(after4);
    expect(expired).toContain(ACTIVE_DID);
    expect(expired).not.toContain(INACTIVE_DID);
  });

  it("should dynamically downgrade when score drops", () => {
    let activePeriods = 20;
    const resolver = (account: string) => {
      if (account !== ACTIVE_DID) return null;
      return {
        score: activePeriods * 50,
        components: {
          claims: 0,
          verifications: 0,
          queries: 0,
          governance: 0,
          stake: 0,
        },
        decayFactor: 1,
        periodsGranted: activePeriods,
      };
    };

    const manager = new ChildAccountManager(
      config,
      DEFAULT_TRANSFER_POLICY,
      undefined,
      resolver,
    );
    manager.credit(ACTIVE_DID, 1000n, baseTime);

    // With 20 periods: tokens still active after 15 periods
    const after15 = baseTime + config.periodDurationMs * 15;
    expect(manager.getBalance(ACTIVE_DID, after15)).toBe(1000n);

    // Score drops: now only 3 periods
    activePeriods = 3;

    // Same tokens, same time: now expired because resolver says 3 periods
    expect(manager.getBalance(ACTIVE_DID, after15)).toBe(0n);
  });
});
