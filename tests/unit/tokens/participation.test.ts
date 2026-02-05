/**
 * Participation Score Calculator Tests
 *
 * Tests for the pure calculation module that maps protocol participation
 * to token expiration periods.
 */

import { describe, it, expect } from "vitest";
import {
  componentScore,
  calculateDecayFactor,
  mapScoreToPeriods,
  calculateParticipationScore,
  createParticipationResolver,
} from "../../../core/tokens/participation.js";
import {
  DEFAULT_PARTICIPATION_CONFIG,
  type ParticipationMetrics,
  type ParticipationConfig,
} from "../../../core/tokens/types.js";

// ============ Test Helpers ============

const TEST_DID = "did:lgm:test0123456789abcdef0123456789ab";
const TEST_DID_2 = "did:lgm:test-account-2-fedcba9876543210";

function createMetrics(
  overrides: Partial<ParticipationMetrics> = {},
): ParticipationMetrics {
  return {
    claimsSubmitted: 0,
    claimsAccepted: 0,
    verificationsCompleted: 0,
    verificationsCorrect: 0,
    knowledgeQueriesMade: 0,
    governanceVotes: 0,
    activeStake: 0n,
    lastActivityBlock: 1000n,
    currentBlock: 1000n,
    accountCreatedBlock: 0n,
    ...overrides,
  };
}

// ============ componentScore Tests ============

describe.sequential("componentScore", () => {
  it("should return 0 for zero actual", () => {
    expect(componentScore(0, 100)).toBe(0);
  });

  it("should return 500 for half of threshold", () => {
    expect(componentScore(50, 100)).toBe(500);
  });

  it("should return 1000 for full threshold", () => {
    expect(componentScore(100, 100)).toBe(1000);
  });

  it("should cap at 1000 for values above threshold", () => {
    expect(componentScore(200, 100)).toBe(1000);
  });

  it("should return 0 for zero threshold (division guard)", () => {
    expect(componentScore(50, 0)).toBe(0);
  });

  it("should return 0 for negative actual", () => {
    expect(componentScore(-10, 100)).toBe(0);
  });

  it("should floor fractional scores", () => {
    // 33 / 100 * 1000 = 330.0 → 330
    expect(componentScore(33, 100)).toBe(330);
  });
});

// ============ calculateDecayFactor Tests ============

describe.sequential("calculateDecayFactor", () => {
  it("should return 1 for zero blocks since activity", () => {
    expect(calculateDecayFactor(0n, 100_000n)).toBe(1);
  });

  it("should return 0.5 after one half-life", () => {
    expect(calculateDecayFactor(100_000n, 100_000n)).toBeCloseTo(0.5, 5);
  });

  it("should return 0.25 after two half-lives", () => {
    expect(calculateDecayFactor(200_000n, 100_000n)).toBeCloseTo(0.25, 5);
  });

  it("should return ~0.707 after half a half-life", () => {
    expect(calculateDecayFactor(50_000n, 100_000n)).toBeCloseTo(0.7071, 3);
  });

  it("should return 1 for zero half-life (no decay)", () => {
    expect(calculateDecayFactor(1000n, 0n)).toBe(1);
  });

  it("should return 1 for negative blocks since activity", () => {
    expect(calculateDecayFactor(-100n, 100_000n)).toBe(1);
  });
});

// ============ mapScoreToPeriods Tests ============

describe.sequential("mapScoreToPeriods", () => {
  it("should return minPeriods for score 0", () => {
    expect(mapScoreToPeriods(0, 2, 182)).toBe(2);
  });

  it("should return maxPeriods for score 1000", () => {
    expect(mapScoreToPeriods(1000, 2, 182)).toBe(182);
  });

  it("should return midpoint for score 500", () => {
    // 2 + floor((182 - 2) * 500 / 1000) = 2 + 90 = 92
    expect(mapScoreToPeriods(500, 2, 182)).toBe(92);
  });

  it("should clamp negative scores to minPeriods", () => {
    expect(mapScoreToPeriods(-100, 2, 182)).toBe(2);
  });

  it("should clamp scores above 1000 to maxPeriods", () => {
    expect(mapScoreToPeriods(2000, 2, 182)).toBe(182);
  });

  it("should handle equal min and max", () => {
    expect(mapScoreToPeriods(500, 10, 10)).toBe(10);
  });
});

// ============ calculateParticipationScore Tests ============

describe.sequential("calculateParticipationScore", () => {
  it("should give minPeriods for zero participation", () => {
    const result = calculateParticipationScore(createMetrics());
    expect(result.score).toBe(0);
    expect(result.periodsGranted).toBe(DEFAULT_PARTICIPATION_CONFIG.minPeriods);
    expect(result.decayFactor).toBe(1);
  });

  it("should give maxPeriods for max participation (all at threshold, active)", () => {
    const metrics = createMetrics({
      claimsAccepted: DEFAULT_PARTICIPATION_CONFIG.thresholds.claimsForMax,
      verificationsCorrect:
        DEFAULT_PARTICIPATION_CONFIG.thresholds.verificationsForMax,
      knowledgeQueriesMade:
        DEFAULT_PARTICIPATION_CONFIG.thresholds.queriesForMax,
      governanceVotes: DEFAULT_PARTICIPATION_CONFIG.thresholds.votesForMax,
      activeStake: DEFAULT_PARTICIPATION_CONFIG.thresholds.stakeForMax,
    });
    const result = calculateParticipationScore(metrics);
    expect(result.score).toBe(1000);
    expect(result.periodsGranted).toBe(DEFAULT_PARTICIPATION_CONFIG.maxPeriods);
  });

  it("should apply decay for inactive accounts", () => {
    const metrics = createMetrics({
      claimsAccepted: DEFAULT_PARTICIPATION_CONFIG.thresholds.claimsForMax,
      verificationsCorrect:
        DEFAULT_PARTICIPATION_CONFIG.thresholds.verificationsForMax,
      knowledgeQueriesMade:
        DEFAULT_PARTICIPATION_CONFIG.thresholds.queriesForMax,
      governanceVotes: DEFAULT_PARTICIPATION_CONFIG.thresholds.votesForMax,
      activeStake: DEFAULT_PARTICIPATION_CONFIG.thresholds.stakeForMax,
      lastActivityBlock: 0n,
      currentBlock: 100_000n, // One half-life elapsed
    });
    const result = calculateParticipationScore(metrics);
    expect(result.decayFactor).toBeCloseTo(0.5, 3);
    expect(result.score).toBe(500);
    // 2 + floor(180 * 500 / 1000) = 2 + 90 = 92
    expect(result.periodsGranted).toBe(92);
  });

  it("should give minPeriods for accounts below age gate", () => {
    const config: ParticipationConfig = {
      ...DEFAULT_PARTICIPATION_CONFIG,
      minAccountAgeBlocks: 10_000n,
    };
    const metrics = createMetrics({
      claimsAccepted: 50,
      currentBlock: 5_000n,
      accountCreatedBlock: 0n,
    });
    const result = calculateParticipationScore(metrics, config);
    expect(result.score).toBe(0);
    expect(result.periodsGranted).toBe(config.minPeriods);
  });

  it("should weight verifications highest (0.30)", () => {
    const verificationsOnly = createMetrics({
      verificationsCorrect:
        DEFAULT_PARTICIPATION_CONFIG.thresholds.verificationsForMax,
    });
    const claimsOnly = createMetrics({
      claimsAccepted: DEFAULT_PARTICIPATION_CONFIG.thresholds.claimsForMax,
    });
    const vResult = calculateParticipationScore(verificationsOnly);
    const cResult = calculateParticipationScore(claimsOnly);
    expect(vResult.score).toBeGreaterThan(cResult.score);
    expect(vResult.components.verifications).toBe(1000);
    expect(vResult.components.claims).toBe(0);
  });

  it("should handle stake-only participation", () => {
    const metrics = createMetrics({
      activeStake: DEFAULT_PARTICIPATION_CONFIG.thresholds.stakeForMax,
    });
    const result = calculateParticipationScore(metrics);
    // Weight is 0.20, so raw score = 1000 * 0.20 = 200
    expect(result.score).toBe(200);
    expect(result.components.stake).toBe(1000);
  });

  it("should handle partial participation", () => {
    const metrics = createMetrics({
      claimsAccepted: 25, // Half of threshold 50
      verificationsCorrect: 100, // Half of threshold 200
    });
    const result = calculateParticipationScore(metrics);
    // Claims: 500 * 0.25 = 125, Verifications: 500 * 0.30 = 150
    // Total: 275
    expect(result.score).toBe(275);
    expect(result.components.claims).toBe(500);
    expect(result.components.verifications).toBe(500);
  });

  it("should degrade significantly after two half-lives of inactivity", () => {
    const metrics = createMetrics({
      claimsAccepted: DEFAULT_PARTICIPATION_CONFIG.thresholds.claimsForMax,
      verificationsCorrect:
        DEFAULT_PARTICIPATION_CONFIG.thresholds.verificationsForMax,
      knowledgeQueriesMade:
        DEFAULT_PARTICIPATION_CONFIG.thresholds.queriesForMax,
      governanceVotes: DEFAULT_PARTICIPATION_CONFIG.thresholds.votesForMax,
      activeStake: DEFAULT_PARTICIPATION_CONFIG.thresholds.stakeForMax,
      lastActivityBlock: 0n,
      currentBlock: 200_000n, // Two half-lives
    });
    const result = calculateParticipationScore(metrics);
    expect(result.decayFactor).toBeCloseTo(0.25, 3);
    expect(result.score).toBe(250);
  });
});

// ============ createParticipationResolver Tests ============

describe.sequential("createParticipationResolver", () => {
  it("should return score from provider", () => {
    let callCount = 0;
    const resolver = createParticipationResolver(
      (_account) => {
        callCount++;
        return createMetrics({ claimsAccepted: 50 });
      },
      DEFAULT_PARTICIPATION_CONFIG,
      100,
      60_000,
    );
    const result = resolver(TEST_DID);
    expect(result).not.toBeNull();
    expect(result!.components.claims).toBe(1000);
    expect(callCount).toBe(1);
  });

  it("should cache results within TTL", () => {
    let callCount = 0;
    const resolver = createParticipationResolver(
      (_account) => {
        callCount++;
        return createMetrics({ claimsAccepted: 25 });
      },
      DEFAULT_PARTICIPATION_CONFIG,
      100,
      60_000,
    );
    resolver(TEST_DID);
    resolver(TEST_DID);
    resolver(TEST_DID);
    expect(callCount).toBe(1);
  });

  it("should return null when provider returns null", () => {
    const resolver = createParticipationResolver(
      (_account) => null,
      DEFAULT_PARTICIPATION_CONFIG,
    );
    expect(resolver(TEST_DID)).toBeNull();
  });

  it("should handle multiple accounts independently", () => {
    const resolver = createParticipationResolver(
      (account) => {
        if (account === TEST_DID) return createMetrics({ claimsAccepted: 50 });
        if (account === TEST_DID_2)
          return createMetrics({ verificationsCorrect: 200 });
        return null;
      },
      DEFAULT_PARTICIPATION_CONFIG,
      100,
      60_000,
    );
    const r1 = resolver(TEST_DID);
    const r2 = resolver(TEST_DID_2);
    expect(r1!.components.claims).toBe(1000);
    expect(r1!.components.verifications).toBe(0);
    expect(r2!.components.claims).toBe(0);
    expect(r2!.components.verifications).toBe(1000);
  });

  it("should evict oldest entry when cache is full", () => {
    let callCount = 0;
    const resolver = createParticipationResolver(
      (_account) => {
        callCount++;
        return createMetrics({ claimsAccepted: 10 });
      },
      DEFAULT_PARTICIPATION_CONFIG,
      2, // Cache size of 2
      60_000,
    );
    resolver("did:lgm:account-1");
    resolver("did:lgm:account-2");
    resolver("did:lgm:account-3"); // Evicts account-1
    expect(callCount).toBe(3);

    // account-1 was evicted, so this should call provider again
    resolver("did:lgm:account-1");
    expect(callCount).toBe(4);

    // account-3 should still be cached
    resolver("did:lgm:account-3");
    expect(callCount).toBe(4);
  });
});
