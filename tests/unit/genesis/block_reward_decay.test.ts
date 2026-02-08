/**
 * Block Reward Decay Tests
 *
 * Tests the dual-decay formula:
 *   R(block) = max(R_initial × timeDecay × apiSuppressor × fundHealth, R_floor)
 *
 * Covers:
 * - Pure time decay (backward compatibility)
 * - API suppression (quadratic)
 * - Fund health tapering
 * - Genesis allocation correctness
 */
import { describe, it, expect } from "vitest";
import {
  calculateBlockReward,
  type EmissionSchedule,
  DEFAULT_EMISSION_SCHEDULE,
} from "../../../core/genesis/types.js";
import {
  calculateGenesisAllocations,
  verifyGenesisAllocations,
  GENESIS_ALLOCATION_BPS,
} from "../../../core/genesis/genesis_allocation.js";
import {
  MAX_BOOTSTRAP_CLAIMS,
  BOOTSTRAP_AMOUNT,
  TOTAL_BOOTSTRAP_FUND,
} from "../../../core/genesis/bootstrap_fund.js";
import { TOKEN } from "../../../core/constants.js";

const ONE_LGM = 10n ** 18n;

// Simplified schedule for predictable test math
const TEST_SCHEDULE: EmissionSchedule = {
  initialReward: 10n * ONE_LGM,
  decayFactor: 0.9,
  blocksPerEpoch: 100n,
  floorReward: ONE_LGM / 10n, // 0.1 LGM
  researchShareRate: 0.22,
  initialFundBalance: 1000n * ONE_LGM,
  fundHealthThreshold: 0.1,
  apiSuppressionExponent: 2,
};

describe("Block Reward Decay", () => {
  describe("Pure time decay (backward compatibility)", () => {
    it("returns initial reward at block 0 with no API or fund params", () => {
      const reward = calculateBlockReward(0n, TEST_SCHEDULE);
      expect(reward).toBe(10n * ONE_LGM);
    });

    it("decays by 10% per epoch", () => {
      // Epoch 1 (block 100): 10 × 0.9 = 9 LGM
      const reward = calculateBlockReward(100n, TEST_SCHEDULE);
      expect(reward).toBe(9n * ONE_LGM);
    });

    it("applies floor reward when decay goes below it", () => {
      // After many epochs, should hit floor
      const reward = calculateBlockReward(10000n, TEST_SCHEDULE);
      expect(reward).toBe(TEST_SCHEDULE.floorReward);
    });

    it("matches original behavior when optional params are undefined", () => {
      const withoutParams = calculateBlockReward(50n, TEST_SCHEDULE);
      const withUndefined = calculateBlockReward(
        50n,
        TEST_SCHEDULE,
        undefined,
        undefined,
      );
      expect(withoutParams).toBe(withUndefined);
    });
  });

  describe("API suppression (quadratic)", () => {
    it("has no effect when apiRevenuePerBlock is 0", () => {
      const withoutApi = calculateBlockReward(0n, TEST_SCHEDULE);
      const withZeroApi = calculateBlockReward(0n, TEST_SCHEDULE, 0n);
      expect(withoutApi).toBe(withZeroApi);
    });

    it("suppresses by 25% when s=0.5 (API covers half)", () => {
      // s = 0.5, suppressor = 1 - 0.5² = 0.75
      // reward = 10 LGM × 0.75 = 7.5 LGM
      const baseReward = 10n * ONE_LGM;
      const apiRevenue = baseReward / 2n; // s = 0.5
      const reward = calculateBlockReward(0n, TEST_SCHEDULE, apiRevenue);
      expect(reward).toBe((baseReward * 7500n) / 10000n);
    });

    it("zeroes out when s=1.0 (API fully sustains)", () => {
      const baseReward = 10n * ONE_LGM;
      const reward = calculateBlockReward(0n, TEST_SCHEDULE, baseReward);
      // s = 1.0, suppressor = 1 - 1.0² = 0
      // But floor applies when fund balance is not tracked
      expect(reward).toBe(TEST_SCHEDULE.floorReward);
    });

    it("caps s at 1.0 when API exceeds block reward", () => {
      const baseReward = 10n * ONE_LGM;
      const doubleApi = baseReward * 2n;
      const reward = calculateBlockReward(0n, TEST_SCHEDULE, doubleApi);
      // s > 1.0 → capped to 1.0 → suppressor = 0 → floor
      expect(reward).toBe(TEST_SCHEDULE.floorReward);
    });

    it("gives gentle early taper (quadratic curve)", () => {
      const baseReward = 10n * ONE_LGM;
      // s=0.1 → suppressor = 1 - 0.01 = 0.99
      const apiLow = baseReward / 10n;
      const rewardLow = calculateBlockReward(0n, TEST_SCHEDULE, apiLow);
      // s=0.3 → suppressor = 1 - 0.09 = 0.91
      const apiMid = (baseReward * 3n) / 10n;
      const rewardMid = calculateBlockReward(0n, TEST_SCHEDULE, apiMid);

      // Low API → minimal suppression
      expect(rewardLow).toBeGreaterThan((baseReward * 98n) / 100n);
      // Mid API → moderate suppression
      expect(rewardMid).toBeGreaterThan((baseReward * 90n) / 100n);
      expect(rewardMid).toBeLessThan(baseReward);
    });

    it("supports linear suppression (exponent=1)", () => {
      const linearSchedule = { ...TEST_SCHEDULE, apiSuppressionExponent: 1 };
      const baseReward = 10n * ONE_LGM;
      // s=0.5 → suppressor = 1 - 0.5 = 0.5
      const apiRevenue = baseReward / 2n;
      const reward = calculateBlockReward(0n, linearSchedule, apiRevenue);
      expect(reward).toBe((baseReward * 5000n) / 10000n);
    });
  });

  describe("Fund health tapering", () => {
    it("has no effect when fund is above threshold", () => {
      // threshold = 10% of 1000 = 100 LGM
      // fund at 500 LGM → above threshold → no tapering
      const reward = calculateBlockReward(
        0n,
        TEST_SCHEDULE,
        undefined,
        500n * ONE_LGM,
      );
      expect(reward).toBe(10n * ONE_LGM);
    });

    it("tapers linearly when fund drops below threshold", () => {
      // threshold = 10% of 1000 = 100 LGM
      // fund at 50 LGM → factor = 50/100 = 0.5
      const reward = calculateBlockReward(
        0n,
        TEST_SCHEDULE,
        undefined,
        50n * ONE_LGM,
      );
      // reward = 10 × 0.5 = 5 LGM
      expect(reward).toBe(5n * ONE_LGM);
    });

    it("returns zero when fund is empty", () => {
      const reward = calculateBlockReward(0n, TEST_SCHEDULE, undefined, 0n);
      expect(reward).toBe(0n);
    });

    it("returns zero when fund is negative (should not happen)", () => {
      const reward = calculateBlockReward(0n, TEST_SCHEDULE, undefined, -1n);
      expect(reward).toBe(0n);
    });

    it("does not apply floor when fund is empty", () => {
      const reward = calculateBlockReward(0n, TEST_SCHEDULE, undefined, 0n);
      // Floor should NOT apply when fund is exhausted
      expect(reward).toBe(0n);
      expect(reward).not.toBe(TEST_SCHEDULE.floorReward);
    });

    it("applies floor when fund has balance but reward is low", () => {
      // Fund has balance but is below threshold → tapering applies
      // Small enough fund that tapered reward < floor, but fund > 0
      const reward = calculateBlockReward(
        0n,
        TEST_SCHEDULE,
        undefined,
        ONE_LGM / 10n,
      );
      // Fund = 0.1 LGM, threshold = 100 LGM, factor = 0.001
      // Tapered = 10 × 0.001 = 0.01 LGM → floor = 0.1 LGM wins
      expect(reward).toBe(TEST_SCHEDULE.floorReward);
    });
  });

  describe("Combined decay factors", () => {
    it("applies time decay + API suppression together", () => {
      // Epoch 1: time decay = 0.9, API s=0.5: suppressor = 0.75
      // Combined: 10 × 0.9 × 0.75 = 6.75 LGM
      const baseAfterTimeDecay = 9n * ONE_LGM; // 10 × 0.9
      const apiRevenue = baseAfterTimeDecay / 2n; // s = 0.5 relative to decayed reward
      const reward = calculateBlockReward(100n, TEST_SCHEDULE, apiRevenue);
      const expected = (baseAfterTimeDecay * 7500n) / 10000n;
      expect(reward).toBe(expected);
    });

    it("applies all three factors: time + API + fund health", () => {
      // Epoch 1: time decay = 0.9 → 9 LGM
      // Fund at 5% of initial (50 LGM) → factor = 50/100 = 0.5
      // No API → suppressor = 1.0
      // Combined: 9 × 0.5 = 4.5 LGM
      const reward = calculateBlockReward(
        100n,
        TEST_SCHEDULE,
        undefined,
        50n * ONE_LGM,
      );
      expect(reward).toBe((9n * ONE_LGM) / 2n);
    });
  });
});

describe("Genesis Allocation", () => {
  it("all allocations sum to exactly MAX_SUPPLY", () => {
    const allocations = calculateGenesisAllocations();
    const verification = verifyGenesisAllocations(allocations);
    expect(verification.valid).toBe(true);
    expect(verification.difference).toBe(0n);
    expect(verification.total).toBe(TOKEN.MAX_SUPPLY);
  });

  it("produces 5 fund allocations", () => {
    const allocations = calculateGenesisAllocations();
    expect(allocations).toHaveLength(5);
  });

  it("block reward fund gets 60% of supply", () => {
    const allocations = calculateGenesisAllocations();
    const blockRewardFund = allocations.find(
      (a) => a.label === "Block Reward Fund",
    )!;
    const expected = (TOKEN.MAX_SUPPLY * 6000n) / 10000n;
    expect(blockRewardFund.amount).toBe(expected);
  });

  it("bootstrap fund gets 20% of supply", () => {
    const allocations = calculateGenesisAllocations();
    const bootstrapFund = allocations.find(
      (a) => a.label === "Bootstrap Fund",
    )!;
    const expected = (TOKEN.MAX_SUPPLY * 2000n) / 10000n;
    expect(bootstrapFund.amount).toBe(expected);
  });

  it("allocation shares sum to 10000 bps", () => {
    const totalBps =
      GENESIS_ALLOCATION_BPS.BLOCK_REWARD_FUND +
      GENESIS_ALLOCATION_BPS.BOOTSTRAP_FUND +
      GENESIS_ALLOCATION_BPS.TREASURY +
      GENESIS_ALLOCATION_BPS.VALIDATOR_GENESIS +
      GENESIS_ALLOCATION_BPS.COMMUNITY_POOL;
    expect(totalBps).toBe(10000n);
  });

  it("works with custom supply amounts", () => {
    const customSupply = 1_000_000n * ONE_LGM;
    const allocations = calculateGenesisAllocations(customSupply);
    const verification = verifyGenesisAllocations(allocations, customSupply);
    expect(verification.valid).toBe(true);
  });
});

describe("Bootstrap Fund Reduction", () => {
  it("MAX_BOOTSTRAP_CLAIMS is 200,200,200", () => {
    expect(MAX_BOOTSTRAP_CLAIMS).toBe(200_200_200n);
  });

  it("TOTAL_BOOTSTRAP_FUND is ~44.4M LGM", () => {
    const expectedTotal = BOOTSTRAP_AMOUNT * MAX_BOOTSTRAP_CLAIMS;
    expect(TOTAL_BOOTSTRAP_FUND).toBe(expectedTotal);
    // ~44,444,444.4 LGM
    expect(TOTAL_BOOTSTRAP_FUND / ONE_LGM).toBeLessThan(45_000_000n);
    expect(TOTAL_BOOTSTRAP_FUND / ONE_LGM).toBeGreaterThan(44_000_000n);
  });

  it("TOTAL_BOOTSTRAP_FUND fits within 20% of MAX_SUPPLY", () => {
    const twentyPercent = (TOKEN.MAX_SUPPLY * 2000n) / 10000n;
    expect(TOTAL_BOOTSTRAP_FUND).toBeLessThanOrEqual(twentyPercent);
  });
});

describe("DEFAULT_EMISSION_SCHEDULE", () => {
  it("has initialFundBalance set", () => {
    expect(DEFAULT_EMISSION_SCHEDULE.initialFundBalance).toBeDefined();
    expect(DEFAULT_EMISSION_SCHEDULE.initialFundBalance).toBeGreaterThan(0n);
  });

  it("initialFundBalance matches 60% of MAX_SUPPLY", () => {
    const sixtyPercent = (TOKEN.MAX_SUPPLY * 6000n) / 10000n;
    expect(DEFAULT_EMISSION_SCHEDULE.initialFundBalance).toBe(sixtyPercent);
  });

  it("has quadratic API suppression", () => {
    expect(DEFAULT_EMISSION_SCHEDULE.apiSuppressionExponent).toBe(2);
  });

  it("has 10% fund health threshold", () => {
    expect(DEFAULT_EMISSION_SCHEDULE.fundHealthThreshold).toBe(0.1);
  });
});
