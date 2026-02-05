/**
 * RetrievalEngine Unit Tests
 *
 * Tests for the token retrieval mechanism for expired tokens.
 * Implements the circular token economy: expire → retrieve → distribute.
 *
 * PP HIERARCHY:
 *   TRUTH: Verify retrieval calculations and RAI distribution are correct
 *   UNDERSTANDING: Test all retrieval operations and distribution logic
 *   BEAUTY: Clear test structure mirroring engine methods
 *   JUSTICE: Tests in proper tokens unit test location
 */

import { describe, it, expect, beforeEach } from "vitest";
import { RetrievalEngine } from "../../../core/tokens/retrieval_engine.js";
import { ChildAccountManager } from "../../../core/tokens/child_account.js";
import { ExpirationTracker } from "../../../core/tokens/expiration_tracker.js";
import {
  DEFAULT_EXPIRATION_CONFIG,
  TokenError,
  type ExpirationConfig,
  type RAIEligibility,
} from "../../../core/tokens/types.js";

// ============ Test Helpers ============

const TEST_DID = "did:lgm:test-account-0123456789abcdef";
const TEST_DID_2 = "did:lgm:test-account-2-fedcba9876543210";
const TEST_DID_3 = "did:lgm:test-account-3-abcdef0123456789";
const RAI_DESTINATION = "did:lgm:rai-destination";

function createTestConfig(
  overrides: Partial<ExpirationConfig> = {},
): ExpirationConfig {
  return {
    ...DEFAULT_EXPIRATION_CONFIG,
    periodDurationMs: 1000, // 1 second periods for testing
    periodsUntilExpiration: 3, // 3 periods until expiration
    tokenLifespanMs: 3000,
    dustThreshold: 1n, // Allow small test amounts
    graceBlocks: 0n, // No grace period for tests
    ...overrides,
  };
}

function createTestEngine(config?: ExpirationConfig): {
  engine: RetrievalEngine;
  childManager: ChildAccountManager;
  tracker: ExpirationTracker;
  config: ExpirationConfig;
  basePeriod: number;
  baseTime: number;
} {
  const testConfig = config ?? createTestConfig();
  const basePeriod = 1000;
  const baseTime = basePeriod * testConfig.periodDurationMs;

  const childManager = new ChildAccountManager(testConfig);
  const tracker = new ExpirationTracker(testConfig, basePeriod);
  const engine = new RetrievalEngine(childManager, tracker, testConfig);

  return {
    engine,
    childManager,
    tracker,
    config: testConfig,
    basePeriod,
    baseTime,
  };
}

// ============ Initialization Tests ============

describe.sequential("RetrievalEngine Initialization", () => {
  it("should initialize with dependencies", () => {
    const { engine } = createTestEngine();
    expect(engine).toBeDefined();
  });

  it("should initialize RAI account with zero balance", () => {
    const { engine } = createTestEngine();
    const raiAccount = engine.getRAIAccount();

    expect(raiAccount.balance).toBe(0n);
    expect(raiAccount.totalReceived).toBe(0n);
    expect(raiAccount.totalDistributed).toBe(0n);
    expect(raiAccount.lastDistributionBlock).toBe(0n);
  });

  it("should accept custom RAI account ID", () => {
    const config = createTestConfig();
    const childManager = new ChildAccountManager(config);
    const tracker = new ExpirationTracker(config, 1000);
    const engine = new RetrievalEngine(
      childManager,
      tracker,
      config,
      "custom:rai:id",
    );

    const raiAccount = engine.getRAIAccount();
    expect(raiAccount.id).toBe("custom:rai:id");
  });
});

// ============ Retrieval Request Tests ============

describe.sequential("RetrievalEngine Retrieval Requests", () => {
  let engine: RetrievalEngine;
  let basePeriod: number;

  beforeEach(() => {
    const context = createTestEngine();
    engine = context.engine;
    basePeriod = context.basePeriod;
  });

  it("should create retrieval request", () => {
    const request = engine.createRetrievalRequest(basePeriod, 1n);

    expect(request.id).toBeDefined();
    expect(request.period).toBe(basePeriod);
    expect(request.requestedAtBlock).toBe(1n);
    expect(request.status).toBe("pending");
  });

  it("should generate unique request IDs", () => {
    const request1 = engine.createRetrievalRequest(basePeriod, 1n);
    const request2 = engine.createRetrievalRequest(basePeriod, 2n);

    expect(request1.id).not.toBe(request2.id);
  });

  it("should accept target accounts filter", () => {
    const targetAccounts = [TEST_DID, TEST_DID_2];
    const request = engine.createRetrievalRequest(
      basePeriod,
      1n,
      targetAccounts,
    );

    expect(request.targetAccounts).toEqual(targetAccounts);
  });

  it("should track pending requests", () => {
    engine.createRetrievalRequest(basePeriod, 1n);
    engine.createRetrievalRequest(basePeriod + 1, 2n);

    const pending = engine.getPendingRetrievals();
    expect(pending.length).toBe(2);
  });
});

// ============ Retrieval Execution Tests ============

describe.sequential("RetrievalEngine Retrieval Execution", () => {
  it("should fail for non-existent request", async () => {
    const { engine, baseTime } = createTestEngine();

    await expect(
      engine.executeRetrieval("nonexistent", 1n, baseTime, RAI_DESTINATION),
    ).rejects.toThrow(TokenError);
  });

  it("should fail for non-expired period", async () => {
    const { engine, childManager, tracker, basePeriod, baseTime, config } =
      createTestEngine();

    // Mint tokens in current period
    childManager.credit(TEST_DID, 1000n, baseTime);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // Create request for current (non-expired) period
    const request = engine.createRetrievalRequest(basePeriod, 1n);

    // Execute should fail since period hasn't expired
    await expect(
      engine.executeRetrieval(request.id, 1n, baseTime, RAI_DESTINATION),
    ).rejects.toThrow(TokenError);
  });

  it("should execute retrieval for expired period", async () => {
    const { engine, childManager, tracker, basePeriod, config } =
      createTestEngine();
    const baseTime = basePeriod * config.periodDurationMs;

    // Mint tokens in base period
    childManager.credit(TEST_DID, 1000n, baseTime);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    // Create request for the period
    const request = engine.createRetrievalRequest(basePeriod, 5n);

    // Advance time to expire the period
    // Period expires when: currentPeriod - periodsUntilExpiration > period
    // So currentPeriod > basePeriod + periodsUntilExpiration
    const expiredTime =
      (basePeriod + config.periodsUntilExpiration + 1) *
      config.periodDurationMs;

    // Process period transition to mark it expired
    tracker.processPeriodTransition(
      basePeriod + config.periodsUntilExpiration,
      5n,
    );

    const result = await engine.executeRetrieval(
      request.id,
      10n,
      expiredTime,
      RAI_DESTINATION,
    );

    expect(result.totalRetrieved).toBe(1000n);
    expect(result.accountsProcessed).toBe(1);
    expect(result.period).toBe(basePeriod);
    expect(result.destination).toBe(RAI_DESTINATION);
  });

  it("should credit retrieved tokens to RAI account", async () => {
    const { engine, childManager, tracker, basePeriod, config } =
      createTestEngine();
    const baseTime = basePeriod * config.periodDurationMs;

    childManager.credit(TEST_DID, 1000n, baseTime);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const request = engine.createRetrievalRequest(basePeriod, 5n);
    const expiredTime =
      (basePeriod + config.periodsUntilExpiration + 1) *
      config.periodDurationMs;
    tracker.processPeriodTransition(
      basePeriod + config.periodsUntilExpiration,
      5n,
    );

    await engine.executeRetrieval(
      request.id,
      10n,
      expiredTime,
      RAI_DESTINATION,
    );

    const raiAccount = engine.getRAIAccount();
    expect(raiAccount.balance).toBe(1000n);
    expect(raiAccount.totalReceived).toBe(1000n);
  });

  it("should move request from pending to completed", async () => {
    const { engine, childManager, tracker, basePeriod, config } =
      createTestEngine();
    const baseTime = basePeriod * config.periodDurationMs;

    childManager.credit(TEST_DID, 1000n, baseTime);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const request = engine.createRetrievalRequest(basePeriod, 5n);
    const expiredTime =
      (basePeriod + config.periodsUntilExpiration + 1) *
      config.periodDurationMs;
    tracker.processPeriodTransition(
      basePeriod + config.periodsUntilExpiration,
      5n,
    );

    await engine.executeRetrieval(
      request.id,
      10n,
      expiredTime,
      RAI_DESTINATION,
    );

    expect(engine.getPendingRetrievals().length).toBe(0);
    expect(engine.getCompletedRetrievals().length).toBe(1);
  });

  it("should enforce grace period", async () => {
    const config = createTestConfig({ graceBlocks: 10n });
    const { engine, childManager, tracker, basePeriod } =
      createTestEngine(config);
    const baseTime = basePeriod * config.periodDurationMs;

    childManager.credit(TEST_DID, 1000n, baseTime);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const request = engine.createRetrievalRequest(basePeriod, 5n);

    // Expire the period
    tracker.processPeriodTransition(
      basePeriod + config.periodsUntilExpiration,
      5n,
    );
    const expiredTime =
      (basePeriod + config.periodsUntilExpiration + 1) *
      config.periodDurationMs;

    // Try to retrieve within grace period (block 6 vs expiredAtBlock 5 + graceBlocks 10 = 15)
    await expect(
      engine.executeRetrieval(request.id, 6n, expiredTime, RAI_DESTINATION),
    ).rejects.toThrow(TokenError);
  });
});

// ============ Auto Request Creation Tests ============

describe.sequential("RetrievalEngine Auto Request Creation", () => {
  it("should auto-create requests for expired periods", () => {
    const { engine, childManager, tracker, basePeriod, config } =
      createTestEngine();
    const baseTime = basePeriod * config.periodDurationMs;

    // Mint and let periods expire
    childManager.credit(TEST_DID, 1000n, baseTime);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const expiredTime =
      (basePeriod + config.periodsUntilExpiration + 1) *
      config.periodDurationMs;
    const requests = engine.autoCreateRetrievalRequests(expiredTime, 10n);

    expect(requests.length).toBeGreaterThan(0);
    expect(requests[0].period).toBe(basePeriod);
  });

  it("should not duplicate existing requests", () => {
    const { engine, childManager, tracker, basePeriod, config } =
      createTestEngine();
    const baseTime = basePeriod * config.periodDurationMs;

    childManager.credit(TEST_DID, 1000n, baseTime);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const expiredTime =
      (basePeriod + config.periodsUntilExpiration + 1) *
      config.periodDurationMs;

    // Create request manually
    engine.createRetrievalRequest(basePeriod, 5n);

    // Auto-create should skip the existing one
    const autoRequests = engine.autoCreateRetrievalRequests(expiredTime, 10n);
    expect(autoRequests.length).toBe(0);
  });
});

// ============ RAI Distribution Tests ============

describe.sequential("RetrievalEngine RAI Distribution", () => {
  it("should calculate distribution evenly", () => {
    const { engine } = createTestEngine();

    // Manually set RAI balance for testing
    const raiAccount = engine.getRAIAccount();
    (engine as any).raiAccount.balance = 1000n;

    const eligible = [TEST_DID, TEST_DID_2];
    const result = engine.calculateRAIDistribution(eligible);

    expect(result.totalAmount).toBe(1000n);
    expect(result.perAccount).toBe(500n);
    expect(result.remainder).toBe(0n);
    expect(result.recipientCount).toBe(2);
  });

  it("should handle remainder in distribution", () => {
    const { engine } = createTestEngine();
    (engine as any).raiAccount.balance = 1000n;

    const eligible = [TEST_DID, TEST_DID_2, TEST_DID_3];
    const result = engine.calculateRAIDistribution(eligible);

    expect(result.perAccount).toBe(333n);
    expect(result.remainder).toBe(1n);
    expect(result.totalAmount).toBe(1000n);
  });

  it("should return zero per account when no eligible accounts", () => {
    const { engine } = createTestEngine();
    (engine as any).raiAccount.balance = 1000n;

    const result = engine.calculateRAIDistribution([]);

    expect(result.perAccount).toBe(0n);
    expect(result.remainder).toBe(1000n);
    expect(result.recipientCount).toBe(0);
  });

  it("should execute distribution to eligible accounts", () => {
    const { engine, childManager, tracker, basePeriod, config } =
      createTestEngine();
    const baseTime = basePeriod * config.periodDurationMs;

    // Set RAI balance
    (engine as any).raiAccount.balance = 1000n;

    const eligible = [TEST_DID, TEST_DID_2];
    const distribution = engine.executeRAIDistribution(eligible, 10n, baseTime);

    expect(distribution.amount).toBe(1000n);
    expect(distribution.recipientCount).toBe(2);
    expect(distribution.amountPerRecipient).toBe(500n);

    // Verify accounts received tokens
    expect(childManager.getBalance(TEST_DID, baseTime)).toBe(500n);
    expect(childManager.getBalance(TEST_DID_2, baseTime)).toBe(500n);
  });

  it("should update RAI account after distribution", () => {
    const { engine, baseTime } = createTestEngine();
    (engine as any).raiAccount.balance = 1000n;

    engine.executeRAIDistribution([TEST_DID, TEST_DID_2], 10n, baseTime);

    const raiAccount = engine.getRAIAccount();
    expect(raiAccount.balance).toBe(0n);
    expect(raiAccount.totalDistributed).toBe(1000n);
    expect(raiAccount.lastDistributionBlock).toBe(10n);
  });

  it("should record distribution in history", () => {
    const { engine, baseTime } = createTestEngine();
    (engine as any).raiAccount.balance = 1000n;

    engine.executeRAIDistribution([TEST_DID], 10n, baseTime);

    const history = engine.getDistributionHistory();
    expect(history.length).toBe(1);
    expect(history[0].amount).toBe(1000n);
    expect(history[0].recipientCount).toBe(1);
  });

  it("should fail when distribution amount too small", () => {
    const { engine, baseTime } = createTestEngine();
    (engine as any).raiAccount.balance = 1n;

    // 1n divided by 2 accounts = 0n per account
    expect(() =>
      engine.executeRAIDistribution([TEST_DID, TEST_DID_2], 10n, baseTime),
    ).toThrow(TokenError);
  });

  it("should handle custom distribution amount", () => {
    const { engine, childManager, baseTime } = createTestEngine();
    (engine as any).raiAccount.balance = 1000n;

    // Distribute only 500n, not the full balance
    const distribution = engine.executeRAIDistribution(
      [TEST_DID],
      10n,
      baseTime,
      500n,
    );

    expect(distribution.amount).toBe(500n);
    expect(engine.getRAIAccount().balance).toBe(500n);
    expect(childManager.getBalance(TEST_DID, baseTime)).toBe(500n);
  });
});

// ============ Eligibility Tests ============

describe.sequential("RetrievalEngine Eligibility", () => {
  let engine: RetrievalEngine;

  const defaultEligibility: RAIEligibility = {
    minAccountAge: 10n,
    minActivityScore: 50,
    requiredVerificationLevel: 1,
    excludedTypes: ["system", "contract"],
  };

  beforeEach(() => {
    const context = createTestEngine();
    engine = context.engine;
  });

  it("should filter by account age", () => {
    const allAccounts = [TEST_DID, TEST_DID_2];
    const metadata = new Map([
      [
        TEST_DID,
        {
          createdAtBlock: 100n,
          activityScore: 100,
          balance: 0n,
          verificationLevel: 2,
          accountType: "agent",
        },
      ],
      [
        TEST_DID_2,
        {
          createdAtBlock: 95n,
          activityScore: 100,
          balance: 0n,
          verificationLevel: 2,
          accountType: "agent",
        },
      ],
    ]);

    const eligible = engine.getEligibleAccounts(
      allAccounts,
      defaultEligibility,
      100n,
      metadata,
    );

    // TEST_DID: age = 100 - 100 = 0, needs 10 → NOT eligible
    // TEST_DID_2: age = 100 - 95 = 5, needs 10 → NOT eligible
    expect(eligible.length).toBe(0);
  });

  it("should filter by activity score", () => {
    const allAccounts = [TEST_DID, TEST_DID_2];
    const metadata = new Map([
      [
        TEST_DID,
        {
          createdAtBlock: 0n,
          activityScore: 100,
          balance: 0n,
          verificationLevel: 2,
          accountType: "agent",
        },
      ],
      [
        TEST_DID_2,
        {
          createdAtBlock: 0n,
          activityScore: 30,
          balance: 0n,
          verificationLevel: 2,
          accountType: "agent",
        },
      ],
    ]);

    const eligible = engine.getEligibleAccounts(
      allAccounts,
      defaultEligibility,
      100n,
      metadata,
    );

    // TEST_DID: score 100 >= 50 → eligible
    // TEST_DID_2: score 30 < 50 → NOT eligible
    expect(eligible).toContain(TEST_DID);
    expect(eligible).not.toContain(TEST_DID_2);
  });

  it("should filter by max balance", () => {
    const eligibility = { ...defaultEligibility, maxBalance: 1000n };
    const allAccounts = [TEST_DID, TEST_DID_2];
    const metadata = new Map([
      [
        TEST_DID,
        {
          createdAtBlock: 0n,
          activityScore: 100,
          balance: 500n,
          verificationLevel: 2,
          accountType: "agent",
        },
      ],
      [
        TEST_DID_2,
        {
          createdAtBlock: 0n,
          activityScore: 100,
          balance: 2000n,
          verificationLevel: 2,
          accountType: "agent",
        },
      ],
    ]);

    const eligible = engine.getEligibleAccounts(
      allAccounts,
      eligibility,
      100n,
      metadata,
    );

    expect(eligible).toContain(TEST_DID);
    expect(eligible).not.toContain(TEST_DID_2);
  });

  it("should filter by verification level", () => {
    const allAccounts = [TEST_DID, TEST_DID_2];
    const metadata = new Map([
      [
        TEST_DID,
        {
          createdAtBlock: 0n,
          activityScore: 100,
          balance: 0n,
          verificationLevel: 2,
          accountType: "agent",
        },
      ],
      [
        TEST_DID_2,
        {
          createdAtBlock: 0n,
          activityScore: 100,
          balance: 0n,
          verificationLevel: 0,
          accountType: "agent",
        },
      ],
    ]);

    const eligible = engine.getEligibleAccounts(
      allAccounts,
      defaultEligibility,
      100n,
      metadata,
    );

    expect(eligible).toContain(TEST_DID);
    expect(eligible).not.toContain(TEST_DID_2);
  });

  it("should filter by excluded types", () => {
    const allAccounts = [TEST_DID, TEST_DID_2, TEST_DID_3];
    const metadata = new Map([
      [
        TEST_DID,
        {
          createdAtBlock: 0n,
          activityScore: 100,
          balance: 0n,
          verificationLevel: 2,
          accountType: "agent",
        },
      ],
      [
        TEST_DID_2,
        {
          createdAtBlock: 0n,
          activityScore: 100,
          balance: 0n,
          verificationLevel: 2,
          accountType: "system",
        },
      ],
      [
        TEST_DID_3,
        {
          createdAtBlock: 0n,
          activityScore: 100,
          balance: 0n,
          verificationLevel: 2,
          accountType: "contract",
        },
      ],
    ]);

    const eligible = engine.getEligibleAccounts(
      allAccounts,
      defaultEligibility,
      100n,
      metadata,
    );

    expect(eligible).toContain(TEST_DID);
    expect(eligible).not.toContain(TEST_DID_2);
    expect(eligible).not.toContain(TEST_DID_3);
  });

  it("should exclude accounts without metadata", () => {
    const allAccounts = [TEST_DID, TEST_DID_2];
    const metadata = new Map([
      [
        TEST_DID,
        {
          createdAtBlock: 0n,
          activityScore: 100,
          balance: 0n,
          verificationLevel: 2,
          accountType: "agent",
        },
      ],
      // TEST_DID_2 has no metadata
    ]);

    const eligible = engine.getEligibleAccounts(
      allAccounts,
      defaultEligibility,
      100n,
      metadata,
    );

    expect(eligible).toContain(TEST_DID);
    expect(eligible).not.toContain(TEST_DID_2);
  });
});

// ============ Fee Routing Tests ============

describe.sequential("RetrievalEngine Fee Routing", () => {
  it("should route fee to RAI account", () => {
    const { engine } = createTestEngine();

    engine.routeRetrievalFee(500n);

    const raiAccount = engine.getRAIAccount();
    expect(raiAccount.balance).toBe(500n);
    expect(raiAccount.totalReceived).toBe(500n);
    expect(raiAccount.totalReceivedFromFees).toBe(500n);
  });

  it("should accumulate multiple fees", () => {
    const { engine } = createTestEngine();

    engine.routeRetrievalFee(500n);
    engine.routeRetrievalFee(300n);

    const raiAccount = engine.getRAIAccount();
    expect(raiAccount.balance).toBe(800n);
    expect(raiAccount.totalReceivedFromFees).toBe(800n);
  });

  it("should handle zero fee amount", () => {
    const { engine } = createTestEngine();

    engine.routeRetrievalFee(0n);

    const raiAccount = engine.getRAIAccount();
    expect(raiAccount.balance).toBe(0n);
    expect(raiAccount.totalReceivedFromFees).toBe(0n);
  });

  it("should include totalReceivedFromFees in statistics", () => {
    const { engine } = createTestEngine();

    engine.routeRetrievalFee(500n);

    const stats = engine.getStatistics();
    expect(stats.totalReceivedFromFees).toBe(500n);
  });

  it("should survive export/import with totalReceivedFromFees", () => {
    const { engine: engine1 } = createTestEngine();

    engine1.routeRetrievalFee(500n);
    const exported = engine1.exportState();

    const { engine: engine2 } = createTestEngine();
    engine2.importState(exported);

    expect(engine2.getRAIAccount().totalReceivedFromFees).toBe(500n);
    expect(engine2.getStatistics().totalReceivedFromFees).toBe(500n);
  });
});

// ============ Statistics Tests ============

describe.sequential("RetrievalEngine Statistics", () => {
  it("should return initial statistics", () => {
    const { engine } = createTestEngine();

    const stats = engine.getStatistics();

    expect(stats.raiBalance).toBe(0n);
    expect(stats.totalReceived).toBe(0n);
    expect(stats.totalDistributed).toBe(0n);
    expect(stats.pendingRetrievals).toBe(0);
    expect(stats.completedRetrievals).toBe(0);
    expect(stats.totalDistributions).toBe(0);
    expect(stats.averageDistributionAmount).toBe(0n);
  });

  it("should track retrieval statistics", async () => {
    const { engine, childManager, tracker, basePeriod, config } =
      createTestEngine();
    const baseTime = basePeriod * config.periodDurationMs;

    childManager.credit(TEST_DID, 1000n, baseTime);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const request = engine.createRetrievalRequest(basePeriod, 5n);
    const expiredTime =
      (basePeriod + config.periodsUntilExpiration + 1) *
      config.periodDurationMs;
    tracker.processPeriodTransition(
      basePeriod + config.periodsUntilExpiration,
      5n,
    );

    await engine.executeRetrieval(
      request.id,
      10n,
      expiredTime,
      RAI_DESTINATION,
    );

    const stats = engine.getStatistics();
    expect(stats.totalReceived).toBe(1000n);
    expect(stats.completedRetrievals).toBe(1);
  });

  it("should track distribution statistics", () => {
    const { engine, baseTime } = createTestEngine();
    (engine as any).raiAccount.balance = 1000n;

    engine.executeRAIDistribution([TEST_DID, TEST_DID_2], 10n, baseTime);

    const stats = engine.getStatistics();
    expect(stats.totalDistributed).toBe(1000n);
    expect(stats.totalDistributions).toBe(1);
    expect(stats.averageDistributionAmount).toBe(1000n);
    expect(stats.lastDistributionBlock).toBe(10n);
  });

  it("should calculate circular economy metrics", () => {
    const { engine, tracker } = createTestEngine();

    // Set up some state
    (engine as any).raiAccount.balance = 100n;
    (tracker as any).globalState.totalActiveSupply = 900n;
    (tracker as any).globalState.totalExpiredSupply = 200n;
    (engine as any).raiAccount.totalDistributed = 150n;

    const metrics = engine.getCircularEconomyMetrics();

    // Total supply = 900 + 100 = 1000
    // Circulation rate = 900 / 1000 = 0.9
    expect(metrics.circulationRate).toBeCloseTo(0.9);

    // Expiration rate = 200 / (900 + 200) = 200/1100 ≈ 0.182
    expect(metrics.expirationRate).toBeCloseTo(0.182, 2);

    // Redistribution rate = 150 / 200 = 0.75
    expect(metrics.redistributionRate).toBeCloseTo(0.75);

    // Velocity multiplier = 150 / 900 ≈ 0.167
    expect(metrics.velocityMultiplier).toBeCloseTo(0.167, 2);
  });

  it("should handle zero supply in metrics", () => {
    const { engine } = createTestEngine();

    const metrics = engine.getCircularEconomyMetrics();

    expect(metrics.circulationRate).toBe(0);
    expect(metrics.expirationRate).toBe(0);
    expect(metrics.redistributionRate).toBe(0);
    expect(metrics.velocityMultiplier).toBe(0);
  });
});

// ============ State Import/Export Tests ============

describe.sequential("RetrievalEngine State Persistence", () => {
  it("should export state", () => {
    const { engine, basePeriod } = createTestEngine();

    engine.createRetrievalRequest(basePeriod, 1n);
    (engine as any).raiAccount.balance = 1000n;

    const exported = engine.exportState();

    expect(exported.raiAccount).toBeDefined();
    expect(exported.pendingRetrievals.size).toBe(1);
    expect(exported.completedRetrievals).toBeDefined();
    expect(exported.distributionHistory).toBeDefined();
  });

  it("should import state", () => {
    const { engine: engine1, basePeriod } = createTestEngine();
    const { engine: engine2 } = createTestEngine();

    // Set up state in engine1
    engine1.createRetrievalRequest(basePeriod, 1n);
    (engine1 as any).raiAccount.balance = 1000n;
    (engine1 as any).raiAccount.totalReceived = 2000n;

    // Export and import
    const exported = engine1.exportState();
    engine2.importState(exported);

    // Verify state transferred
    expect(engine2.getRAIAccount().balance).toBe(1000n);
    expect(engine2.getRAIAccount().totalReceived).toBe(2000n);
    expect(engine2.getPendingRetrievals().length).toBe(1);
  });

  it("should preserve distribution history through export/import", () => {
    const { engine: engine1, baseTime } = createTestEngine();
    (engine1 as any).raiAccount.balance = 1000n;

    engine1.executeRAIDistribution([TEST_DID], 10n, baseTime);

    const exported = engine1.exportState();

    const { engine: engine2 } = createTestEngine();
    engine2.importState(exported);

    const history = engine2.getDistributionHistory();
    expect(history.length).toBe(1);
    expect(history[0].amount).toBe(1000n);
  });
});

// ============ Query Method Tests ============

describe.sequential("RetrievalEngine Query Methods", () => {
  it("should return distribution history with limit", () => {
    const { engine, baseTime } = createTestEngine();

    // Create multiple distributions
    for (let i = 0; i < 5; i++) {
      (engine as any).raiAccount.balance = 100n;
      engine.executeRAIDistribution([TEST_DID], BigInt(i), baseTime + i * 1000);
    }

    const limitedHistory = engine.getDistributionHistory(3);
    expect(limitedHistory.length).toBe(3);

    const fullHistory = engine.getDistributionHistory();
    expect(fullHistory.length).toBe(5);
  });

  it("should return completed retrievals with limit", async () => {
    const { engine, childManager, tracker, basePeriod, config } =
      createTestEngine();
    const baseTime = basePeriod * config.periodDurationMs;

    // Set up multiple completions
    for (let i = 0; i < 3; i++) {
      const period = basePeriod + i;
      const mintTime = period * config.periodDurationMs;
      childManager.credit(TEST_DID, 100n, mintTime);
      tracker.recordMint(TEST_DID, 100n, period, BigInt(i));

      const request = engine.createRetrievalRequest(period, BigInt(i * 10));
      const expiredTime =
        (period + config.periodsUntilExpiration + 1) * config.periodDurationMs;
      tracker.processPeriodTransition(
        period + config.periodsUntilExpiration,
        BigInt(i * 10),
      );

      await engine.executeRetrieval(
        request.id,
        BigInt(i * 10 + 5),
        expiredTime,
        RAI_DESTINATION,
      );
    }

    const limitedResults = engine.getCompletedRetrievals(2);
    expect(limitedResults.length).toBe(2);
  });

  it("should get retrieval result by ID", async () => {
    const { engine, childManager, tracker, basePeriod, config } =
      createTestEngine();
    const baseTime = basePeriod * config.periodDurationMs;

    childManager.credit(TEST_DID, 1000n, baseTime);
    tracker.recordMint(TEST_DID, 1000n, basePeriod, 1n);

    const request = engine.createRetrievalRequest(basePeriod, 5n);
    const expiredTime =
      (basePeriod + config.periodsUntilExpiration + 1) *
      config.periodDurationMs;
    tracker.processPeriodTransition(
      basePeriod + config.periodsUntilExpiration,
      5n,
    );

    await engine.executeRetrieval(
      request.id,
      10n,
      expiredTime,
      RAI_DESTINATION,
    );

    const result = engine.getRetrievalResult(request.id);
    expect(result).toBeDefined();
    expect(result!.totalRetrieved).toBe(1000n);
  });

  it("should return undefined for non-existent result", () => {
    const { engine } = createTestEngine();

    const result = engine.getRetrievalResult("nonexistent");
    expect(result).toBeUndefined();
  });
});
