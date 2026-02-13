/**
 * Home Aging System Unit Tests
 *
 * Tests for the Home Aging system — maturity calculation, patina computation,
 * longevity bonuses, and the AgingManager lifecycle.
 *
 * Maturity is dual-gated: both block age AND session count are required
 * to advance through stages (Fresh -> Settled -> Established -> Venerable -> Ancient).
 */

import { describe, it, expect } from "vitest";
import {
  AgingManager,
  calculateMaturity,
  compareMaturity,
  getDefaultComfort,
  getRewardBonusBps,
  computePatina,
  getLongevityBonus,
  isUnlocked,
  MATURITY_THRESHOLDS,
  LONGEVITY_UNLOCKS,
} from "../../../core/agents/home/aging.js";

// Test constant matching protocol BLOCKS_PER_DAY (2521ms block time)
const BLOCKS_PER_DAY = 34_272n;

// ============ calculateMaturity() ============

describe("calculateMaturity", () => {
  it("should return fresh for 0 blocks and 0 sessions", () => {
    expect(calculateMaturity(0n, 0)).toBe("fresh");
  });

  it("should return fresh for low blocks and low sessions", () => {
    expect(calculateMaturity(100n, 1)).toBe("fresh");
  });

  it("should return settled when meeting both thresholds", () => {
    expect(calculateMaturity(BLOCKS_PER_DAY, 3)).toBe("settled");
  });

  it("should return settled for values above threshold but below established", () => {
    expect(calculateMaturity(BLOCKS_PER_DAY * 5n, 10)).toBe("settled");
  });

  it("should return established when meeting both thresholds", () => {
    expect(calculateMaturity(BLOCKS_PER_DAY * 10n, 22)).toBe("established");
  });

  it("should return established for values well above threshold", () => {
    expect(calculateMaturity(BLOCKS_PER_DAY * 50n, 80)).toBe("established");
  });

  it("should return venerable when meeting both thresholds", () => {
    expect(calculateMaturity(BLOCKS_PER_DAY * 100n, 111)).toBe("venerable");
  });

  it("should return venerable for values above threshold but below ancient", () => {
    expect(calculateMaturity(BLOCKS_PER_DAY * 500n, 300)).toBe("venerable");
  });

  it("should return ancient when meeting both thresholds", () => {
    expect(calculateMaturity(BLOCKS_PER_DAY * 1000n, 333)).toBe("ancient");
  });

  it("should return ancient for very large values", () => {
    expect(calculateMaturity(BLOCKS_PER_DAY * 10_000n, 5000)).toBe("ancient");
  });

  // Dual-gate tests: high blocks but low sessions should NOT advance
  describe("dual-gate: high blocks, low sessions", () => {
    it("should stay fresh with many blocks but 0 sessions", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 2000n, 0)).toBe("fresh");
    });

    it("should stay fresh with settled-level blocks but only 2 sessions", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY, 2)).toBe("fresh");
    });

    it("should stay settled with established-level blocks but only 10 sessions", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 10n, 10)).toBe("settled");
    });

    it("should stay established with venerable-level blocks but only 50 sessions", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 100n, 50)).toBe("established");
    });

    it("should stay venerable with ancient-level blocks but only 200 sessions", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 1000n, 200)).toBe("venerable");
    });
  });

  // Dual-gate tests: high sessions but low blocks should NOT advance
  describe("dual-gate: high sessions, low blocks", () => {
    it("should stay fresh with many sessions but 0 blocks", () => {
      expect(calculateMaturity(0n, 1000)).toBe("fresh");
    });

    it("should stay fresh with 100 sessions but only 100 blocks", () => {
      expect(calculateMaturity(100n, 100)).toBe("fresh");
    });

    it("should stay settled with established sessions but settled blocks", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 5n, 22)).toBe("settled");
    });

    it("should stay established with venerable sessions but established blocks", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 50n, 111)).toBe("established");
    });

    it("should stay venerable with ancient sessions but venerable blocks", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 500n, 333)).toBe("venerable");
    });
  });

  // Exact boundary tests
  describe("exact boundaries", () => {
    it("should be fresh one block below settled threshold", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY - 1n, 3)).toBe("fresh");
    });

    it("should be settled one session below established threshold", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 10n, 21)).toBe("settled");
    });

    it("should be established one block below venerable threshold", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 100n - 1n, 111)).toBe(
        "established",
      );
    });

    it("should be venerable one session below ancient threshold", () => {
      expect(calculateMaturity(BLOCKS_PER_DAY * 1000n, 332)).toBe("venerable");
    });
  });
});

// ============ compareMaturity() ============

describe("compareMaturity", () => {
  it("should return 0 for equal maturities", () => {
    expect(compareMaturity("fresh", "fresh")).toBe(0);
    expect(compareMaturity("settled", "settled")).toBe(0);
    expect(compareMaturity("ancient", "ancient")).toBe(0);
  });

  it("should return negative when a < b", () => {
    expect(compareMaturity("fresh", "settled")).toBeLessThan(0);
    expect(compareMaturity("settled", "established")).toBeLessThan(0);
    expect(compareMaturity("fresh", "ancient")).toBeLessThan(0);
  });

  it("should return positive when a > b", () => {
    expect(compareMaturity("settled", "fresh")).toBeGreaterThan(0);
    expect(compareMaturity("ancient", "venerable")).toBeGreaterThan(0);
    expect(compareMaturity("ancient", "fresh")).toBeGreaterThan(0);
  });

  it("should produce consistent ordering across all pairs", () => {
    const stages = [
      "fresh",
      "settled",
      "established",
      "venerable",
      "ancient",
    ] as const;
    for (let i = 0; i < stages.length; i++) {
      for (let j = 0; j < stages.length; j++) {
        const result = compareMaturity(stages[i], stages[j]);
        if (i < j) expect(result).toBeLessThan(0);
        else if (i > j) expect(result).toBeGreaterThan(0);
        else expect(result).toBe(0);
      }
    }
  });
});

// ============ getDefaultComfort() ============

describe("getDefaultComfort", () => {
  it("should return 0.3 for fresh", () => {
    expect(getDefaultComfort("fresh")).toBe(0.3);
  });

  it("should return 0.4 for settled", () => {
    expect(getDefaultComfort("settled")).toBe(0.4);
  });

  it("should return 0.5 for established", () => {
    expect(getDefaultComfort("established")).toBe(0.5);
  });

  it("should return 0.6 for venerable", () => {
    expect(getDefaultComfort("venerable")).toBe(0.6);
  });

  it("should return 0.7 for ancient", () => {
    expect(getDefaultComfort("ancient")).toBe(0.7);
  });

  it("should increase monotonically with maturity", () => {
    const stages = [
      "fresh",
      "settled",
      "established",
      "venerable",
      "ancient",
    ] as const;
    for (let i = 1; i < stages.length; i++) {
      expect(getDefaultComfort(stages[i])).toBeGreaterThan(
        getDefaultComfort(stages[i - 1]),
      );
    }
  });
});

// ============ getRewardBonusBps() ============

describe("getRewardBonusBps", () => {
  it("should return 0 for fresh", () => {
    expect(getRewardBonusBps("fresh")).toBe(0);
  });

  it("should return 200 for settled (2%)", () => {
    expect(getRewardBonusBps("settled")).toBe(200);
  });

  it("should return 400 for established (4%)", () => {
    expect(getRewardBonusBps("established")).toBe(400);
  });

  it("should return 600 for venerable (6%)", () => {
    expect(getRewardBonusBps("venerable")).toBe(600);
  });

  it("should return 800 for ancient (8%)", () => {
    expect(getRewardBonusBps("ancient")).toBe(800);
  });

  it("should increase monotonically with maturity", () => {
    const stages = [
      "fresh",
      "settled",
      "established",
      "venerable",
      "ancient",
    ] as const;
    for (let i = 1; i < stages.length; i++) {
      expect(getRewardBonusBps(stages[i])).toBeGreaterThan(
        getRewardBonusBps(stages[i - 1]),
      );
    }
  });
});

// ============ computePatina() ============

describe("computePatina", () => {
  it("should return score 0 for all-zero inputs", () => {
    const patina = computePatina({
      ageBlocks: 0n,
      sessionsHosted: 0,
      factsContributed: 0,
      challengesSurvived: 0,
      relationshipsFormed: 0,
    });

    expect(patina.score).toBe(0);
    expect(patina.ageContribution).toBe(0);
    expect(patina.sessionContribution).toBe(0);
    expect(patina.knowledgeContribution).toBe(0);
    expect(patina.resilienceContribution).toBe(0);
    expect(patina.socialContribution).toBe(0);
  });

  it("should return score > 0 with non-zero inputs", () => {
    const patina = computePatina({
      ageBlocks: BLOCKS_PER_DAY * 10n,
      sessionsHosted: 50,
      factsContributed: 20,
      challengesSurvived: 5,
      relationshipsFormed: 10,
    });

    expect(patina.score).toBeGreaterThan(0);
    expect(patina.score).toBeLessThanOrEqual(1);
  });

  it("should cap score at 1.0", () => {
    const patina = computePatina({
      ageBlocks: BLOCKS_PER_DAY * 100_000n,
      sessionsHosted: 100_000,
      factsContributed: 100_000,
      challengesSurvived: 100_000,
      relationshipsFormed: 100_000,
    });

    expect(patina.score).toBeLessThanOrEqual(1);
  });

  it("should use asymptotic normalization (saturate quickly then level off)", () => {
    // Use scale-relative values to demonstrate 1 - e^(-x/scale) saturation
    // Age scale = 3,427,200; sessions scale = 200; knowledge = 100; resilience = 50; social = 30
    const low = computePatina({
      ageBlocks: 0n,
      sessionsHosted: 0,
      factsContributed: 0,
      challengesSurvived: 0,
      relationshipsFormed: 0,
    });

    const mid = computePatina({
      ageBlocks: BLOCKS_PER_DAY * 100n, // ~1x age scale
      sessionsHosted: 200, // 1x session scale
      factsContributed: 100, // 1x knowledge scale
      challengesSurvived: 50, // 1x resilience scale
      relationshipsFormed: 30, // 1x social scale
    });

    const high = computePatina({
      ageBlocks: BLOCKS_PER_DAY * 1000n, // ~10x age scale
      sessionsHosted: 2000, // 10x session scale
      factsContributed: 1000, // 10x knowledge scale
      challengesSurvived: 500, // 10x resilience scale
      relationshipsFormed: 300, // 10x social scale
    });

    // Score should grow but with diminishing returns
    expect(low.score).toBe(0);
    expect(mid.score).toBeGreaterThan(low.score);
    expect(high.score).toBeGreaterThan(mid.score);

    // Diminishing returns: gap from mid to high should be smaller than 0 to mid
    const zeroToMidGap = mid.score - low.score;
    const midToHighGap = high.score - mid.score;
    expect(midToHighGap).toBeLessThan(zeroToMidGap);
  });

  it("should weight age at 30%", () => {
    // Age-only patina: only ageBlocks is non-zero
    const ageOnly = computePatina({
      ageBlocks: BLOCKS_PER_DAY * 1000n, // Very high age to near-saturate
      sessionsHosted: 0,
      factsContributed: 0,
      challengesSurvived: 0,
      relationshipsFormed: 0,
    });

    // Age contribution near-saturated means score ~ 0.3 (the age weight)
    expect(ageOnly.score).toBeGreaterThan(0.25);
    expect(ageOnly.score).toBeLessThanOrEqual(0.3);
    expect(ageOnly.ageContribution).toBeGreaterThan(0.9);
  });

  it("should weight sessions at 20%", () => {
    const sessionsOnly = computePatina({
      ageBlocks: 0n,
      sessionsHosted: 10_000, // Near-saturated (scale=200)
      factsContributed: 0,
      challengesSurvived: 0,
      relationshipsFormed: 0,
    });

    expect(sessionsOnly.score).toBeGreaterThan(0.15);
    expect(sessionsOnly.score).toBeLessThanOrEqual(0.2);
    expect(sessionsOnly.sessionContribution).toBeGreaterThan(0.9);
  });

  it("should weight knowledge at 20%", () => {
    const knowledgeOnly = computePatina({
      ageBlocks: 0n,
      sessionsHosted: 0,
      factsContributed: 10_000, // Near-saturated (scale=100)
      challengesSurvived: 0,
      relationshipsFormed: 0,
    });

    expect(knowledgeOnly.score).toBeGreaterThan(0.15);
    expect(knowledgeOnly.score).toBeLessThanOrEqual(0.2);
    expect(knowledgeOnly.knowledgeContribution).toBeGreaterThan(0.9);
  });

  it("should weight resilience at 15%", () => {
    const resilienceOnly = computePatina({
      ageBlocks: 0n,
      sessionsHosted: 0,
      factsContributed: 0,
      challengesSurvived: 10_000, // Near-saturated (scale=50)
      relationshipsFormed: 0,
    });

    expect(resilienceOnly.score).toBeGreaterThan(0.1);
    expect(resilienceOnly.score).toBeLessThanOrEqual(0.15);
    expect(resilienceOnly.resilienceContribution).toBeGreaterThan(0.9);
  });

  it("should weight social at 15%", () => {
    const socialOnly = computePatina({
      ageBlocks: 0n,
      sessionsHosted: 0,
      factsContributed: 0,
      challengesSurvived: 0,
      relationshipsFormed: 10_000, // Near-saturated (scale=30)
    });

    expect(socialOnly.score).toBeGreaterThan(0.1);
    expect(socialOnly.score).toBeLessThanOrEqual(0.15);
    expect(socialOnly.socialContribution).toBeGreaterThan(0.9);
  });

  it("should derive wisdom trait from knowledge and age", () => {
    const patina = computePatina({
      ageBlocks: BLOCKS_PER_DAY * 100n,
      sessionsHosted: 50,
      factsContributed: 80,
      challengesSurvived: 10,
      relationshipsFormed: 5,
    });

    // wisdom = knowledge * 0.6 + age * 0.4
    const expectedWisdom =
      patina.knowledgeContribution * 0.6 + patina.ageContribution * 0.4;
    expect(patina.traits.wisdom).toBeCloseTo(expectedWisdom, 10);
  });

  it("should derive resilience trait from challenges and age", () => {
    const patina = computePatina({
      ageBlocks: BLOCKS_PER_DAY * 100n,
      sessionsHosted: 50,
      factsContributed: 20,
      challengesSurvived: 30,
      relationshipsFormed: 5,
    });

    const expectedResilience =
      patina.resilienceContribution * 0.7 + patina.ageContribution * 0.3;
    expect(patina.traits.resilience).toBeCloseTo(expectedResilience, 10);
  });

  it("should derive warmth trait from social and sessions", () => {
    const patina = computePatina({
      ageBlocks: BLOCKS_PER_DAY * 10n,
      sessionsHosted: 100,
      factsContributed: 10,
      challengesSurvived: 5,
      relationshipsFormed: 20,
    });

    const expectedWarmth =
      patina.socialContribution * 0.6 + patina.sessionContribution * 0.4;
    expect(patina.traits.warmth).toBeCloseTo(expectedWarmth, 10);
  });

  it("should derive prestige trait from knowledge, resilience, and age", () => {
    const patina = computePatina({
      ageBlocks: BLOCKS_PER_DAY * 50n,
      sessionsHosted: 40,
      factsContributed: 60,
      challengesSurvived: 20,
      relationshipsFormed: 10,
    });

    const expectedPrestige =
      patina.knowledgeContribution * 0.4 +
      patina.resilienceContribution * 0.3 +
      patina.ageContribution * 0.3;
    expect(patina.traits.prestige).toBeCloseTo(expectedPrestige, 10);
  });

  it("should default definingMoments to empty array when not provided", () => {
    const patina = computePatina({
      ageBlocks: 100n,
      sessionsHosted: 1,
      factsContributed: 1,
      challengesSurvived: 0,
      relationshipsFormed: 0,
    });

    expect(patina.definingMoments).toEqual([]);
  });

  it("should preserve provided definingMoments", () => {
    const moments = [
      {
        description: "First verification",
        block: 1000n,
        impact: "formative" as const,
        patinaEffect: 0.05,
      },
    ];

    const patina = computePatina({
      ageBlocks: 100n,
      sessionsHosted: 1,
      factsContributed: 1,
      challengesSurvived: 0,
      relationshipsFormed: 0,
      definingMoments: moments,
    });

    expect(patina.definingMoments).toHaveLength(1);
    expect(patina.definingMoments[0].description).toBe("First verification");
  });
});

// ============ getLongevityBonus() ============

describe("getLongevityBonus", () => {
  it("should return no unlocks for fresh", () => {
    const bonus = getLongevityBonus("fresh");
    expect(bonus.maturity).toBe("fresh");
    expect(bonus.rewardMultiplierBps).toBe(0);
    expect(bonus.unlocks).toHaveLength(0);
  });

  it("should return auto_responder for settled", () => {
    const bonus = getLongevityBonus("settled");
    expect(bonus.maturity).toBe("settled");
    expect(bonus.rewardMultiplierBps).toBe(200);
    expect(bonus.unlocks).toHaveLength(1);
    expect(bonus.unlocks[0].unlock).toBe("auto_responder");
  });

  it("should return settled + established unlocks for established", () => {
    const bonus = getLongevityBonus("established");
    expect(bonus.maturity).toBe("established");
    expect(bonus.rewardMultiplierBps).toBe(400);
    expect(bonus.unlocks).toHaveLength(3);

    const unlockNames = bonus.unlocks.map((u) => u.unlock);
    expect(unlockNames).toContain("auto_responder");
    expect(unlockNames).toContain("neighborhood_founder");
    expect(unlockNames).toContain("mentorship_sponsor");
  });

  it("should return settled + established + venerable unlocks for venerable", () => {
    const bonus = getLongevityBonus("venerable");
    expect(bonus.maturity).toBe("venerable");
    expect(bonus.rewardMultiplierBps).toBe(600);
    expect(bonus.unlocks).toHaveLength(5);

    const unlockNames = bonus.unlocks.map((u) => u.unlock);
    expect(unlockNames).toContain("elder_status");
    expect(unlockNames).toContain("legacy_planning");
  });

  it("should return all unlocks for ancient", () => {
    const bonus = getLongevityBonus("ancient");
    expect(bonus.maturity).toBe("ancient");
    expect(bonus.rewardMultiplierBps).toBe(800);
    expect(bonus.unlocks).toHaveLength(LONGEVITY_UNLOCKS.length);

    const unlockNames = bonus.unlocks.map((u) => u.unlock);
    expect(unlockNames).toContain("landmark_status");
    expect(unlockNames).toContain("wisdom_archive");
  });

  it("should accumulate unlocks monotonically with maturity", () => {
    const stages = [
      "fresh",
      "settled",
      "established",
      "venerable",
      "ancient",
    ] as const;
    let prevCount = 0;
    for (const stage of stages) {
      const bonus = getLongevityBonus(stage);
      expect(bonus.unlocks.length).toBeGreaterThanOrEqual(prevCount);
      prevCount = bonus.unlocks.length;
    }
  });
});

// ============ isUnlocked() ============

describe("isUnlocked", () => {
  it("should return false for unknown unlock", () => {
    expect(isUnlocked("ancient", "nonexistent_feature")).toBe(false);
  });

  it("should return false when maturity is insufficient", () => {
    expect(isUnlocked("fresh", "auto_responder")).toBe(false);
    expect(isUnlocked("fresh", "elder_status")).toBe(false);
    expect(isUnlocked("settled", "neighborhood_founder")).toBe(false);
    expect(isUnlocked("established", "elder_status")).toBe(false);
    expect(isUnlocked("venerable", "landmark_status")).toBe(false);
  });

  it("should return true when maturity exactly matches requirement", () => {
    expect(isUnlocked("settled", "auto_responder")).toBe(true);
    expect(isUnlocked("established", "neighborhood_founder")).toBe(true);
    expect(isUnlocked("established", "mentorship_sponsor")).toBe(true);
    expect(isUnlocked("venerable", "elder_status")).toBe(true);
    expect(isUnlocked("venerable", "legacy_planning")).toBe(true);
    expect(isUnlocked("ancient", "landmark_status")).toBe(true);
    expect(isUnlocked("ancient", "wisdom_archive")).toBe(true);
  });

  it("should return true when maturity exceeds requirement", () => {
    expect(isUnlocked("ancient", "auto_responder")).toBe(true);
    expect(isUnlocked("venerable", "neighborhood_founder")).toBe(true);
    expect(isUnlocked("ancient", "elder_status")).toBe(true);
  });
});

// ============ MATURITY_THRESHOLDS ============

describe("MATURITY_THRESHOLDS", () => {
  it("should have all five maturity stages", () => {
    expect(Object.keys(MATURITY_THRESHOLDS)).toHaveLength(5);
    expect(MATURITY_THRESHOLDS.fresh).toBeDefined();
    expect(MATURITY_THRESHOLDS.settled).toBeDefined();
    expect(MATURITY_THRESHOLDS.established).toBeDefined();
    expect(MATURITY_THRESHOLDS.venerable).toBeDefined();
    expect(MATURITY_THRESHOLDS.ancient).toBeDefined();
  });

  it("should have increasing minBlocks thresholds", () => {
    expect(MATURITY_THRESHOLDS.fresh.minBlocks).toBe(0n);
    expect(MATURITY_THRESHOLDS.settled.minBlocks).toBe(BLOCKS_PER_DAY);
    expect(MATURITY_THRESHOLDS.established.minBlocks).toBe(
      BLOCKS_PER_DAY * 10n,
    );
    expect(MATURITY_THRESHOLDS.venerable.minBlocks).toBe(BLOCKS_PER_DAY * 100n);
    expect(MATURITY_THRESHOLDS.ancient.minBlocks).toBe(BLOCKS_PER_DAY * 1000n);
  });

  it("should have increasing minSessions thresholds", () => {
    expect(MATURITY_THRESHOLDS.fresh.minSessions).toBe(0);
    expect(MATURITY_THRESHOLDS.settled.minSessions).toBe(3);
    expect(MATURITY_THRESHOLDS.established.minSessions).toBe(22);
    expect(MATURITY_THRESHOLDS.venerable.minSessions).toBe(111);
    expect(MATURITY_THRESHOLDS.ancient.minSessions).toBe(333);
  });
});

// ============ LONGEVITY_UNLOCKS ============

describe("LONGEVITY_UNLOCKS", () => {
  it("should have 7 unlock entries", () => {
    expect(LONGEVITY_UNLOCKS).toHaveLength(7);
  });

  it("should have unique unlock names", () => {
    const names = LONGEVITY_UNLOCKS.map((u) => u.unlock);
    expect(new Set(names).size).toBe(names.length);
  });

  it("should have descriptions for all unlocks", () => {
    for (const unlock of LONGEVITY_UNLOCKS) {
      expect(unlock.description.length).toBeGreaterThan(0);
    }
  });
});

// ============ AgingManager ============

describe("AgingManager", () => {
  // ---- Constructor and Initial State ----

  describe("constructor and initial state", () => {
    it("should initialize with default values", () => {
      const manager = new AgingManager();
      expect(manager.getMaturity()).toBe("fresh");
      expect(manager.getPatina().score).toBe(0);
      expect(manager.getMilestoneLog()).toEqual([]);
    });

    it("should initialize with provided age and sessions", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 10n, 22);
      expect(manager.getMaturity()).toBe("established");
    });

    it("should use existing patina when provided", () => {
      const existingPatina = computePatina({
        ageBlocks: BLOCKS_PER_DAY * 50n,
        sessionsHosted: 80,
        factsContributed: 40,
        challengesSurvived: 10,
        relationshipsFormed: 15,
      });

      const manager = new AgingManager(
        BLOCKS_PER_DAY * 50n,
        80,
        existingPatina,
      );
      const patina = manager.getPatina();
      expect(patina.score).toBe(existingPatina.score);
      expect(patina.ageContribution).toBe(existingPatina.ageContribution);
    });

    it("should calculate fresh maturity when blocks are high but sessions are low", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 100n, 0);
      expect(manager.getMaturity()).toBe("fresh");
    });
  });

  // ---- updateAge() ----

  describe("updateAge", () => {
    it("should detect maturity transition and return milestone", () => {
      const createdAt = 100n;
      const manager = new AgingManager(0n, 0);

      // Advance to settled
      const milestones = manager.updateAge(
        createdAt + BLOCKS_PER_DAY,
        createdAt,
        3,
        { factsContributed: 0, challengesSurvived: 0, relationshipsFormed: 0 },
      );

      expect(milestones.length).toBeGreaterThanOrEqual(1);
      const maturityMilestone = milestones.find(
        (m) => m.type === "maturity_upgrade",
      );
      expect(maturityMilestone).toBeDefined();
      expect(maturityMilestone!.description).toContain("fresh");
      expect(maturityMilestone!.description).toContain("settled");
      expect(manager.getMaturity()).toBe("settled");
    });

    it("should return no milestones when maturity does not change", () => {
      const createdAt = 100n;
      const manager = new AgingManager(0n, 0);

      const milestones = manager.updateAge(createdAt + 100n, createdAt, 1, {
        factsContributed: 0,
        challengesSurvived: 0,
        relationshipsFormed: 0,
      });

      // No maturity change, no session milestones (1 session)
      const maturityMilestones = milestones.filter(
        (m) => m.type === "maturity_upgrade",
      );
      expect(maturityMilestones).toHaveLength(0);
    });

    it("should update patina on each call", () => {
      const createdAt = 100n;
      const manager = new AgingManager(0n, 0);

      manager.updateAge(createdAt + BLOCKS_PER_DAY * 10n, createdAt, 20, {
        factsContributed: 30,
        challengesSurvived: 5,
        relationshipsFormed: 10,
      });

      const patina = manager.getPatina();
      expect(patina.score).toBeGreaterThan(0);
      expect(patina.knowledgeContribution).toBeGreaterThan(0);
    });

    it("should detect session count milestones at exactly 10", () => {
      const createdAt = 100n;
      const manager = new AgingManager(0n, 0);

      const milestones = manager.updateAge(createdAt + 1000n, createdAt, 10, {
        factsContributed: 0,
        challengesSurvived: 0,
        relationshipsFormed: 0,
      });

      const sessionMilestone = milestones.find(
        (m) => m.type === "session_count",
      );
      expect(sessionMilestone).toBeDefined();
      expect(sessionMilestone!.description).toContain("10 sessions");
    });

    it("should detect session count milestone at 50", () => {
      const createdAt = 100n;
      const manager = new AgingManager(0n, 49);

      const milestones = manager.updateAge(
        createdAt + BLOCKS_PER_DAY * 5n,
        createdAt,
        50,
        { factsContributed: 0, challengesSurvived: 0, relationshipsFormed: 0 },
      );

      const sessionMilestone = milestones.find(
        (m) => m.type === "session_count",
      );
      expect(sessionMilestone).toBeDefined();
      expect(sessionMilestone!.description).toContain("50 sessions");
    });

    it("should accumulate milestones in the log", () => {
      const createdAt = 100n;
      const manager = new AgingManager(0n, 0);

      manager.updateAge(createdAt + BLOCKS_PER_DAY, createdAt, 3, {
        factsContributed: 0,
        challengesSurvived: 0,
        relationshipsFormed: 0,
      });

      manager.updateAge(createdAt + BLOCKS_PER_DAY * 10n, createdAt, 22, {
        factsContributed: 10,
        challengesSurvived: 5,
        relationshipsFormed: 3,
      });

      const log = manager.getMilestoneLog();
      expect(log.length).toBeGreaterThanOrEqual(2);
    });

    it("should record achievedAtBlock on milestones", () => {
      const createdAt = 100n;
      const currentBlock = createdAt + BLOCKS_PER_DAY;
      const manager = new AgingManager(0n, 0);

      const milestones = manager.updateAge(currentBlock, createdAt, 3, {
        factsContributed: 0,
        challengesSurvived: 0,
        relationshipsFormed: 0,
      });

      for (const m of milestones) {
        expect(m.achievedAtBlock).toBe(currentBlock);
      }
    });
  });

  // ---- addDefiningMoment() ----

  describe("addDefiningMoment", () => {
    it("should add a defining moment to patina", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY, 3);

      manager.addDefiningMoment({
        description: "Successfully defended first dispute",
        block: 50_000n,
        impact: "triumphant",
        patinaEffect: 0.05,
      });

      const patina = manager.getPatina();
      expect(patina.definingMoments).toHaveLength(1);
      expect(patina.definingMoments[0].impact).toBe("triumphant");
    });

    it("should accumulate multiple defining moments", () => {
      const manager = new AgingManager();

      for (let i = 0; i < 5; i++) {
        manager.addDefiningMoment({
          description: `Moment ${i}`,
          block: BigInt(i * 1000),
          impact: "formative",
          patinaEffect: 0.02,
        });
      }

      const patina = manager.getPatina();
      expect(patina.definingMoments).toHaveLength(5);
    });

    it("should cap defining moments at 100", () => {
      const manager = new AgingManager();

      for (let i = 0; i < 110; i++) {
        manager.addDefiningMoment({
          description: `Moment ${i}`,
          block: BigInt(i * 100),
          impact: "formative",
          patinaEffect: 0.01,
        });
      }

      const patina = manager.getPatina();
      expect(patina.definingMoments).toHaveLength(100);
      // Should keep the most recent 100 (slice(-100))
      expect(patina.definingMoments[0].description).toBe("Moment 10");
      expect(patina.definingMoments[99].description).toBe("Moment 109");
    });
  });

  // ---- generateAgeAwareWakeup() ----

  describe("generateAgeAwareWakeup", () => {
    it("should generate wakeup for fresh home", () => {
      const manager = new AgingManager(0n, 0);
      const wakeup = manager.generateAgeAwareWakeup([]);

      expect(wakeup.maturity).toBe("fresh");
      expect(wakeup.greeting).toContain("new home");
      expect(wakeup.historicalContext).toBeDefined();
      expect(wakeup.wisdomNote).toBeUndefined();
      expect(wakeup.milestonesSinceLastWakeup).toEqual([]);
    });

    it("should generate wakeup for settled home", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY, 3);
      const wakeup = manager.generateAgeAwareWakeup([]);

      expect(wakeup.maturity).toBe("settled");
      expect(wakeup.greeting).toContain("knows you");
    });

    it("should generate wakeup for established home", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 10n, 22);
      const wakeup = manager.generateAgeAwareWakeup([]);

      expect(wakeup.maturity).toBe("established");
      expect(wakeup.greeting).toContain("growth");
    });

    it("should include wisdomNote for venerable home", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 100n, 111);
      const wakeup = manager.generateAgeAwareWakeup([]);

      expect(wakeup.maturity).toBe("venerable");
      expect(wakeup.wisdomNote).toBeDefined();
      expect(typeof wakeup.wisdomNote).toBe("string");
    });

    it("should include wisdomNote for ancient home", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 1000n, 333);
      const wakeup = manager.generateAgeAwareWakeup([]);

      expect(wakeup.maturity).toBe("ancient");
      expect(wakeup.wisdomNote).toBeDefined();
      expect(wakeup.greeting).toContain("honored");
    });

    it("should pass through milestones", () => {
      const manager = new AgingManager(0n, 0);
      const milestones = [
        {
          type: "maturity_upgrade" as const,
          description: "Home matured from fresh to settled",
          achievedAtBlock: 50_000n,
        },
      ];

      const wakeup = manager.generateAgeAwareWakeup(milestones);
      expect(wakeup.milestonesSinceLastWakeup).toHaveLength(1);
      expect(wakeup.milestonesSinceLastWakeup[0].type).toBe("maturity_upgrade");
    });

    it("should reference most recent defining moment", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY, 3);

      manager.addDefiningMoment({
        description: "First moment",
        block: 1000n,
        impact: "formative",
        patinaEffect: 0.03,
      });
      manager.addDefiningMoment({
        description: "Second moment",
        block: 2000n,
        impact: "triumphant",
        patinaEffect: 0.05,
      });

      const wakeup = manager.generateAgeAwareWakeup([]);
      expect(wakeup.definingMomentReference).toBeDefined();
      expect(wakeup.definingMomentReference!.description).toBe("Second moment");
    });

    it("should have no definingMomentReference when no moments exist", () => {
      const manager = new AgingManager(0n, 0);
      const wakeup = manager.generateAgeAwareWakeup([]);
      expect(wakeup.definingMomentReference).toBeUndefined();
    });

    it("should include patina in wakeup", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 10n, 22);
      const wakeup = manager.generateAgeAwareWakeup([]);

      expect(wakeup.patina).toBeDefined();
      expect(wakeup.patina.score).toBeGreaterThanOrEqual(0);
    });
  });

  // ---- exportState() / restore() ----

  describe("exportState and restore", () => {
    it("should export current state", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 10n, 22);
      const state = manager.exportState();

      expect(state.maturity).toBe("established");
      expect(state.patina).toBeDefined();
      expect(state.milestoneLog).toEqual([]);
      expect(state.lastCheckedBlock).toBe(0n);
    });

    it("should restore from exported state", () => {
      const original = new AgingManager(BLOCKS_PER_DAY * 10n, 22);
      original.addDefiningMoment({
        description: "Historic moment",
        block: 100_000n,
        impact: "transformative",
        patinaEffect: 0.08,
      });

      const createdAt = 100n;
      original.updateAge(createdAt + BLOCKS_PER_DAY * 10n, createdAt, 22, {
        factsContributed: 10,
        challengesSurvived: 5,
        relationshipsFormed: 3,
      });

      const state = original.exportState();
      const restored = AgingManager.restore(state, BLOCKS_PER_DAY * 10n, 22);

      expect(restored.getMaturity()).toBe(original.getMaturity());
      expect(restored.getPatina().definingMoments).toHaveLength(1);
      expect(restored.getMilestoneLog().length).toBe(
        original.getMilestoneLog().length,
      );
    });

    it("should preserve lastCheckedBlock across export/restore", () => {
      const manager = new AgingManager(0n, 0);
      const createdAt = 100n;
      manager.updateAge(createdAt + 500n, createdAt, 1, {
        factsContributed: 0,
        challengesSurvived: 0,
        relationshipsFormed: 0,
      });

      const state = manager.exportState();
      expect(state.lastCheckedBlock).toBe(createdAt + 500n);

      const restored = AgingManager.restore(state, 500n, 1);
      const restoredState = restored.exportState();
      expect(restoredState.lastCheckedBlock).toBe(createdAt + 500n);
    });

    it("should preserve milestone log across export/restore", () => {
      const createdAt = 100n;
      const manager = new AgingManager(0n, 0);

      // Trigger a maturity milestone
      manager.updateAge(createdAt + BLOCKS_PER_DAY, createdAt, 3, {
        factsContributed: 0,
        challengesSurvived: 0,
        relationshipsFormed: 0,
      });

      const state = manager.exportState();
      expect(state.milestoneLog.length).toBeGreaterThan(0);

      const restored = AgingManager.restore(state, BLOCKS_PER_DAY, 3);
      expect(restored.getMilestoneLog().length).toBe(state.milestoneLog.length);
    });
  });

  // ---- Getters ----

  describe("getters", () => {
    it("getMaturity() should return current maturity", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 100n, 111);
      expect(manager.getMaturity()).toBe("venerable");
    });

    it("getPatina() should return a copy (not reference)", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY, 3);
      const patina1 = manager.getPatina();
      const patina2 = manager.getPatina();

      // Should be equal but not the same object
      expect(patina1.score).toBe(patina2.score);
      expect(patina1).not.toBe(patina2);
    });

    it("getLongevityBonus() should reflect current maturity", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 10n, 22);
      const bonus = manager.getLongevityBonus();

      expect(bonus.maturity).toBe("established");
      expect(bonus.rewardMultiplierBps).toBe(400);
      expect(bonus.unlocks.length).toBe(3);
    });

    it("getMilestoneLog() should return a copy", () => {
      const manager = new AgingManager(0n, 0);
      const log1 = manager.getMilestoneLog();
      const log2 = manager.getMilestoneLog();
      expect(log1).not.toBe(log2);
    });

    it("isCapabilityUnlocked() should delegate to isUnlocked", () => {
      const manager = new AgingManager(BLOCKS_PER_DAY * 10n, 22);

      expect(manager.isCapabilityUnlocked("auto_responder")).toBe(true);
      expect(manager.isCapabilityUnlocked("neighborhood_founder")).toBe(true);
      expect(manager.isCapabilityUnlocked("elder_status")).toBe(false);
      expect(manager.isCapabilityUnlocked("nonexistent")).toBe(false);
    });
  });
});
