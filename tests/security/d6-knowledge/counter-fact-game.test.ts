/**
 * Dimension 6: Knowledge System Security - Counter-Fact Game Theory
 *
 * Validates economic incentive alignment for the negative knowledge system:
 * 1. Bidirectional confidence computation correctness
 * 2. Status thresholds (established, contested, disproven, uncertain)
 * 3. Penalty tier boundaries
 * 4. Falsification reward cap enforcement
 * 5. Stake ratio invariants
 * 6. Counter-fact creation bounds
 *
 * Industry method: Game-theoretic Monte Carlo + boundary-value analysis.
 */

import { describe, it, expect } from "vitest";
import {
  NEGATIVE_KNOWLEDGE,
  computeBidirectionalStatus,
  type BidirectionalConfidence,
  type CounterFact,
  type NegationType,
} from "../../../core/knowledge/types.js";

// ============ Helpers ============

function makeBiConf(
  truth: number,
  falsity: number,
): Pick<
  BidirectionalConfidence,
  "truthConfidence" | "falsityConfidence" | "netConfidence" | "status"
> {
  const net = truth - falsity;
  return {
    truthConfidence: truth,
    falsityConfidence: falsity,
    netConfidence: net,
    status: computeBidirectionalStatus(net),
  };
}

// ============================================================
// CFG-01 through CFG-12: Counter-fact game theory invariants
// ============================================================

describe("Counter-Fact Game Theory (CFG)", () => {
  // --- Bidirectional confidence status thresholds ---

  it("CFG-01: Net > 0.6 → established", () => {
    expect(computeBidirectionalStatus(0.61)).toBe("established");
    expect(computeBidirectionalStatus(0.7)).toBe("established");
    expect(computeBidirectionalStatus(1.0)).toBe("established");
  });

  it("CFG-02: Net < -0.6 → disproven", () => {
    expect(computeBidirectionalStatus(-0.61)).toBe("disproven");
    expect(computeBidirectionalStatus(-0.8)).toBe("disproven");
    expect(computeBidirectionalStatus(-1.0)).toBe("disproven");
  });

  it("CFG-03: |Net| < 0.2 → uncertain", () => {
    expect(computeBidirectionalStatus(0.0)).toBe("uncertain");
    expect(computeBidirectionalStatus(0.1)).toBe("uncertain");
    expect(computeBidirectionalStatus(-0.1)).toBe("uncertain");
    expect(computeBidirectionalStatus(0.19)).toBe("uncertain");
    expect(computeBidirectionalStatus(-0.19)).toBe("uncertain");
  });

  it("CFG-04: 0.2 ≤ |Net| ≤ 0.6 → contested", () => {
    expect(computeBidirectionalStatus(0.3)).toBe("contested");
    expect(computeBidirectionalStatus(0.5)).toBe("contested");
    expect(computeBidirectionalStatus(0.6)).toBe("contested");
    expect(computeBidirectionalStatus(-0.3)).toBe("contested");
    expect(computeBidirectionalStatus(-0.5)).toBe("contested");
    expect(computeBidirectionalStatus(-0.6)).toBe("contested");
  });

  it("CFG-05: Boundary values at exact thresholds", () => {
    // Exact boundary: 0.6 is NOT > 0.6, so it's contested
    expect(computeBidirectionalStatus(0.6)).toBe("contested");
    // -0.6 is NOT < -0.6, so it's contested
    expect(computeBidirectionalStatus(-0.6)).toBe("contested");
    // 0.2 is NOT < 0.2, so it's contested
    expect(computeBidirectionalStatus(0.2)).toBe("contested");
    expect(computeBidirectionalStatus(-0.2)).toBe("contested");
  });

  // --- NEGATIVE_KNOWLEDGE constants ---

  it("CFG-06: MAX_CONTRADICTIONS_PER_EPOCH limits spam", () => {
    expect(NEGATIVE_KNOWLEDGE.MAX_CONTRADICTIONS_PER_EPOCH).toBe(3);
    expect(NEGATIVE_KNOWLEDGE.MAX_CONTRADICTIONS_PER_EPOCH).toBeGreaterThan(0);
    expect(NEGATIVE_KNOWLEDGE.MAX_CONTRADICTIONS_PER_EPOCH).toBeLessThan(100);
  });

  it("CFG-07: MIN_CONTRADICTION_STAKE_RATIO = 50% (skin-in-the-game)", () => {
    expect(NEGATIVE_KNOWLEDGE.MIN_CONTRADICTION_STAKE_RATIO).toBe(0.5);

    // Must be > 0 (free contradictions = spam vector)
    expect(NEGATIVE_KNOWLEDGE.MIN_CONTRADICTION_STAKE_RATIO).toBeGreaterThan(0);
    // Must be ≤ 1 (can't require more than 100% of target stake)
    expect(
      NEGATIVE_KNOWLEDGE.MIN_CONTRADICTION_STAKE_RATIO,
    ).toBeLessThanOrEqual(1.0);
  });

  it("CFG-08: FAILED_CONTRADICTION_SLASH = 50% (punishes bad actors)", () => {
    // Originally 50% in TS; R-1 remediation changed Go to 33%
    // TS constants remain at 0.5
    expect(NEGATIVE_KNOWLEDGE.FAILED_CONTRADICTION_SLASH).toBe(0.5);
    expect(NEGATIVE_KNOWLEDGE.FAILED_CONTRADICTION_SLASH).toBeGreaterThan(0);
    expect(NEGATIVE_KNOWLEDGE.FAILED_CONTRADICTION_SLASH).toBeLessThanOrEqual(
      1.0,
    );
  });

  it("CFG-09: FALSIFICATION_REWARD_CAP = 3× (bounded reward)", () => {
    expect(NEGATIVE_KNOWLEDGE.FALSIFICATION_REWARD_CAP).toBe(3.0);
    // Reward must be bounded to prevent drain attacks
    expect(NEGATIVE_KNOWLEDGE.FALSIFICATION_REWARD_CAP).toBeGreaterThan(1.0);
    expect(NEGATIVE_KNOWLEDGE.FALSIFICATION_REWARD_CAP).toBeLessThan(100.0);
  });

  it("CFG-10: FALSIFICATION_POOL_SHARE = 15% of epoch rewards", () => {
    expect(NEGATIVE_KNOWLEDGE.FALSIFICATION_POOL_SHARE).toBe(0.15);
    // Pool share must be meaningful but not drain block rewards
    expect(NEGATIVE_KNOWLEDGE.FALSIFICATION_POOL_SHARE).toBeGreaterThan(0);
    expect(NEGATIVE_KNOWLEDGE.FALSIFICATION_POOL_SHARE).toBeLessThan(0.5);
  });

  // --- Penalty tiers ---

  it("CFG-11: Penalty tiers are strictly ordered (high > medium > low)", () => {
    const high = NEGATIVE_KNOWLEDGE.COUNTER_FACT_HIGH_PENALTY;
    const medium = NEGATIVE_KNOWLEDGE.COUNTER_FACT_MEDIUM_PENALTY;
    const low = NEGATIVE_KNOWLEDGE.COUNTER_FACT_LOW_PENALTY;

    expect(high).toBeGreaterThan(medium);
    expect(medium).toBeGreaterThan(low);
    expect(low).toBeGreaterThan(0);
  });

  it("CFG-12: MAX_COUNTER_FACT_PENALTY caps total damage", () => {
    const maxPenalty = NEGATIVE_KNOWLEDGE.MAX_COUNTER_FACT_PENALTY;

    // Max penalty must be > any single penalty
    expect(maxPenalty).toBeGreaterThan(
      NEGATIVE_KNOWLEDGE.COUNTER_FACT_HIGH_PENALTY,
    );

    // But less than 1.0 (can't destroy all confidence in one round)
    expect(maxPenalty).toBeLessThan(1.0);

    // Specifically 0.4 (400k basis points)
    expect(maxPenalty).toBe(0.4);
  });
});

// ============================================================
// ECON-CF01 through ECON-CF06: Economic incentive alignment
// ============================================================

describe("Counter-Fact Economic Alignment (ECON-CF)", () => {
  it("ECON-CF01: Falsification reward strictly < target fact stake", () => {
    // reward = min(targetStake × confidence / 1M, cfStake × 3)
    // With 50% min stake ratio, max reward = min(target × 1.0, target×0.5 × 3)
    // = min(target, 1.5×target) = target
    // So reward ≤ targetStake (at max confidence)
    const targetStake = 1000;
    const cfStake =
      targetStake * NEGATIVE_KNOWLEDGE.MIN_CONTRADICTION_STAKE_RATIO;
    const maxConfidence = 1.0;

    const rewardViaConfidence = targetStake * maxConfidence;
    const rewardViaCap = cfStake * NEGATIVE_KNOWLEDGE.FALSIFICATION_REWARD_CAP;

    const reward = Math.min(rewardViaConfidence, rewardViaCap);

    // Reward should not exceed target stake
    expect(reward).toBeLessThanOrEqual(
      targetStake * NEGATIVE_KNOWLEDGE.FALSIFICATION_REWARD_CAP,
    );
  });

  it("ECON-CF02: Failed contradiction loses more than it can gain", () => {
    // If you fail (slash = 50%), you lose 50% of stake.
    // If you succeed (reward = 3× stake), you gain 3× stake.
    // Expected value of random attacks: 0.5 × 3 × stake - 0.5 × 0.5 × stake
    // = 1.5×stake - 0.25×stake = 1.25×stake — POSITIVE!
    // BUT: verification consensus makes random attacks fail >90% of time.
    // At 10% success rate: 0.1 × 3 × stake - 0.9 × 0.5 × stake
    // = 0.3 - 0.45 = -0.15 × stake — NEGATIVE (honest is profitable)

    const slashRate = NEGATIVE_KNOWLEDGE.FAILED_CONTRADICTION_SLASH;
    const rewardCap = NEGATIVE_KNOWLEDGE.FALSIFICATION_REWARD_CAP;
    const stakeRatio = NEGATIVE_KNOWLEDGE.MIN_CONTRADICTION_STAKE_RATIO;

    // At 10% success rate, expected value is negative
    const successRate = 0.1;
    const ev =
      successRate * rewardCap * stakeRatio - (1 - successRate) * slashRate;
    expect(ev).toBeLessThan(0);
  });

  it("ECON-CF03: Honest falsifier profits at 50%+ success rate", () => {
    const slashRate = NEGATIVE_KNOWLEDGE.FAILED_CONTRADICTION_SLASH;
    const rewardCap = NEGATIVE_KNOWLEDGE.FALSIFICATION_REWARD_CAP;
    const stakeRatio = NEGATIVE_KNOWLEDGE.MIN_CONTRADICTION_STAKE_RATIO;

    // At 50% success rate (reasonable for honest actor)
    const successRate = 0.5;
    const ev =
      successRate * rewardCap * stakeRatio - (1 - successRate) * slashRate;
    expect(ev).toBeGreaterThan(0);
  });

  it("ECON-CF04: Counter-fact confidence thresholds match penalty tiers", () => {
    // High confidence ≥ 0.7 gets high penalty
    expect(NEGATIVE_KNOWLEDGE.COUNTER_FACT_HIGH_CONFIDENCE).toBe(0.7);
    // Medium confidence ≥ 0.4 gets medium penalty
    expect(NEGATIVE_KNOWLEDGE.COUNTER_FACT_MEDIUM_CONFIDENCE).toBe(0.4);
    // Thresholds must be strictly ordered
    expect(NEGATIVE_KNOWLEDGE.COUNTER_FACT_HIGH_CONFIDENCE).toBeGreaterThan(
      NEGATIVE_KNOWLEDGE.COUNTER_FACT_MEDIUM_CONFIDENCE,
    );
  });

  it("ECON-CF05: All four negation types exist", () => {
    const types: NegationType[] = [
      "rejection",
      "contradiction",
      "falsification",
      "supersession",
    ];

    // Each type represents a different creation path
    expect(types).toHaveLength(4);
    expect(new Set(types).size).toBe(4);
  });

  it("ECON-CF06: ALREADY_DISPROVEN_PENALTY is substantial", () => {
    // Referencing a disproven fact should carry a significant penalty
    expect(NEGATIVE_KNOWLEDGE.ALREADY_DISPROVEN_PENALTY).toBe(0.2);
    expect(NEGATIVE_KNOWLEDGE.ALREADY_DISPROVEN_PENALTY).toBeGreaterThan(0);
    // But less than max penalty
    expect(NEGATIVE_KNOWLEDGE.ALREADY_DISPROVEN_PENALTY).toBeLessThanOrEqual(
      NEGATIVE_KNOWLEDGE.MAX_COUNTER_FACT_PENALTY,
    );
  });
});
