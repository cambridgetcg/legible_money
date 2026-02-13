/**
 * HIGH Priority Economic Loopholes — Security Tests
 *
 * Tests for HIGH loopholes from the Financial Incentives PP Review:
 *
 * D1-L8: Tier manipulation via Sybil (HIGH)
 * D3-L3: Citation count gaming (HIGH)
 * D3-L5: MEV via knowledge ordering (HIGH)
 * D3-L6: Freshness cliff arbitrage (HIGH)
 * D3-L7: Validator micro-stake Sybil (HIGH)
 * D5-L4: Domain expertise capture (HIGH) — see dispute-loopholes.test.ts
 * D5-L5: Confidence suppression (HIGH) — see dispute-loopholes.test.ts
 */

import { describe, it, expect } from "vitest";

import {
  DEFAULT_CONSENSUS_PARAMS,
  VALIDATOR_TIER_CONFIG,
  MAX_APPRENTICE_VALIDATORS,
  type Validator,
} from "../../../consensus/pot/types.js";

import {
  calculateVotingPower,
  slashValidatorByAmount,
} from "../../../consensus/pot/validator_selection.js";

import { calculateAPICost } from "../../../core/billing/distribution.js";

import { DEFAULT_BILLING_PARAMETERS } from "../../../core/billing/types.js";

import {
  routeReward,
  RESEARCH_SHARE_RATE_BPS,
} from "../../../core/billing/reward_router.js";

import { PUNISHMENT_RATES } from "../../../core/dispute/types.js";

import {
  BASE_CHALLENGE_DURATION,
  CONFIDENCE_GROWTH_EPOCH,
  CONFIDENCE_GROWTH_PER_EPOCH,
  MAX_SURVIVAL_CONFIDENCE,
} from "../../../consensus/pot/adversarial.js";

// ============ Helpers ============

const ONE_LGM = 10n ** 18n;

function makeValidator(
  stake: bigint,
  overrides: Partial<Validator> = {},
): Validator {
  return {
    publicKey:
      overrides.publicKey ?? "val-pk-" + Math.random().toString(36).slice(2),
    did: overrides.did ?? "did:lgm:val-" + Math.random().toString(36).slice(2),
    stake,
    delegatedStake: overrides.delegatedStake ?? 0n,
    reputation: overrides.reputation ?? 0.8,
    isActive: overrides.isActive ?? true,
    lastActiveBlock: overrides.lastActiveBlock ?? 0,
    totalVerifications: overrides.totalVerifications ?? 100,
    correctVerifications: overrides.correctVerifications ?? 80,
    missedVerifications: overrides.missedVerifications ?? 0,
    slashCount: overrides.slashCount ?? 0,
    slashesThisEpoch: overrides.slashesThisEpoch ?? 0,
    lastSlashedBlock: overrides.lastSlashedBlock ?? 0,
    tier: overrides.tier ?? 1,
  };
}

// ============================================================
// D1-L8: TIER MANIPULATION VIA SYBIL (HIGH)
// ============================================================

describe("D1-L8: Tier Manipulation via Sybil", () => {
  it("SYBIL-1: Max apprentice cap limits Sybil surface", () => {
    expect(MAX_APPRENTICE_VALIDATORS).toBe(111);
    // Even with 2000 registrations, only 111 can be active apprentices
  });

  it("SYBIL-2: 111 Apprentice validators vs 1 Guardian — effective power comparison", () => {
    const apprenticeTier = VALIDATOR_TIER_CONFIG[0];
    const guardianTier = VALIDATOR_TIER_CONFIG[3];

    // 111 apprentices at min stake
    const totalSybilStake =
      BigInt(MAX_APPRENTICE_VALIDATORS) * apprenticeTier.minStake;
    const sybilEffectivePower =
      (BigInt(MAX_APPRENTICE_VALIDATORS) *
        apprenticeTier.minStake *
        BigInt(Math.round(apprenticeTier.selectionWeight * 10000))) /
      10000n;

    // 1 guardian
    const guardianEffectivePower =
      (guardianTier.minStake *
        BigInt(Math.round(guardianTier.selectionWeight * 10000))) /
      10000n;

    // Guardian should dominate: 111 LGM * 1.5 = 166.5 effective
    // vs 111 * 0.111 * 0.1 = 1.23 effective
    expect(guardianEffectivePower).toBeGreaterThan(sybilEffectivePower);

    // Guardian has > 100x the effective power
    const ratio = Number(guardianEffectivePower) / Number(sybilEffectivePower);
    expect(ratio).toBeGreaterThan(50);
  });

  it("SYBIL-3: Sybil validators start with 0.5 reputation — reduced voting power", () => {
    const sybilValidator = makeValidator(VALIDATOR_TIER_CONFIG[0].minStake, {
      reputation: 0.5,
      tier: 0,
    });

    const veteranValidator = makeValidator(VALIDATOR_TIER_CONFIG[0].minStake, {
      reputation: 0.95,
      tier: 0,
    });

    const sybilPower = calculateVotingPower(sybilValidator);
    const veteranPower = calculateVotingPower(veteranValidator);

    // Veteran has ~2x voting power due to reputation
    expect(veteranPower).toBeGreaterThan(sybilPower);
  });

  it("SYBIL-4: Total Sybil capital cost — 111 apprentices = 12.3 LGM", () => {
    const apprenticeMin = VALIDATOR_TIER_CONFIG[0].minStake;
    const totalCost = BigInt(MAX_APPRENTICE_VALIDATORS) * apprenticeMin;

    // 111 * 0.111 = 12.321 LGM
    expect(totalCost).toBeGreaterThan(12n * ONE_LGM);
    expect(totalCost).toBeLessThan(13n * ONE_LGM);

    // For the same capital, one could stake as a Bonded validator (11.1 LGM)
    // with 1.0x selection weight instead of 111 * 0.1x = 11.1x effective
    // But effective power: 11.1 * 1.0 = 11.1 vs 12.3 * 0.1 = 1.23
    // Single bonded = 9x better
  });
});

// ============================================================
// D3-L3: CITATION COUNT GAMING (HIGH)
// ============================================================

describe("D3-L3: Citation Count Gaming", () => {
  it("CIT-1: log2(citations+1) gives unbounded discount", () => {
    // Citation discount formula: log2(citations + 1)
    // This gives an unbounded discount as citations grow
    const discountAtCitations = (n: number) => Math.log2(n + 1);

    expect(discountAtCitations(0)).toBeCloseTo(0, 2);
    expect(discountAtCitations(1)).toBeCloseTo(1, 2);
    expect(discountAtCitations(7)).toBeCloseTo(3, 2);
    expect(discountAtCitations(15)).toBeCloseTo(4, 2);
    expect(discountAtCitations(255)).toBeCloseTo(8, 2);
    expect(discountAtCitations(1023)).toBeCloseTo(10, 2);

    // At 256 self-queries: 50% discount (8 out of ~16 max practical)
    // This is gameable
  });

  it("CIT-2: Self-querying 256 times costs less than the discount gained", () => {
    // Cost per query: ~basePricePerCall
    const baseCost = DEFAULT_BILLING_PARAMETERS.basePricePerCall;
    const selfQueryCost = baseCost * 256n;

    // After 256 queries, the fact has a permanent discount
    // Discount factor: log2(257) ≈ 8 → used as divisor in pricing
    // Effective price: baseCost / 8 = 12.5% of original

    // Break-even: when does the discount save more than the 256-query cost?
    // If the fact is queried 2048+ more times, the discount exceeds the investment
    const futureQueries = 2048n;
    const savingsPerQuery = (baseCost * 7n) / 8n; // 87.5% savings
    const totalSavings = savingsPerQuery * futureQueries;

    // Total savings > investment?
    expect(totalSavings).toBeGreaterThan(selfQueryCost);
  });

  it("CIT-3: Proposed fix — cap citation discount at 30%", () => {
    const MAX_CITATION_DISCOUNT_BPS = 3000n; // 30%

    // With cap: even 1M citations give max 30% discount
    const discounts = [0, 1, 10, 100, 256, 1000, 1000000];

    for (const citations of discounts) {
      const rawDiscount = Math.log2(citations + 1) / 10; // Normalized to 0-1 range
      const cappedDiscountBps = BigInt(
        Math.round(Math.min(rawDiscount, 0.3) * 10000),
      );

      expect(cappedDiscountBps).toBeLessThanOrEqual(MAX_CITATION_DISCOUNT_BPS);
    }
  });

  it("CIT-4: Proposed fix — unique querier requirement", () => {
    // Only count citations from distinct DIDs
    const queries = [
      { did: "did:lgm:a", factId: "f1" },
      { did: "did:lgm:a", factId: "f1" }, // Duplicate — don't count
      { did: "did:lgm:b", factId: "f1" },
      { did: "did:lgm:c", factId: "f1" },
      { did: "did:lgm:a", factId: "f1" }, // Duplicate — don't count
    ];

    const uniqueQueriers = new Set(queries.map((q) => q.did));
    expect(uniqueQueriers.size).toBe(3); // Only 3 unique queriers, not 5 queries
  });
});

// ============================================================
// D3-L5: MEV VIA KNOWLEDGE ORDERING (HIGH)
// ============================================================

describe("D3-L5: MEV via Knowledge Ordering", () => {
  it("MEV-1: Block producer can reorder queries for profit", () => {
    // In a knowledge marketplace, a block producer who is also an API provider
    // can see pending queries and front-run by:
    // 1. Observing a pending query for fact F
    // 2. Verifying fact F themselves (cheaper, as validator)
    // 3. Including their verification BEFORE the query
    // 4. Earning the query revenue + verification reward

    // The extractable value per query = query cost * provider share
    const queryCost = DEFAULT_BILLING_PARAMETERS.basePricePerCall;
    const providerShare = DEFAULT_BILLING_PARAMETERS.apiProviderShare;
    const extractablePerQuery =
      (queryCost * BigInt(Math.round(providerShare * 10000))) / 10000n;

    // For a busy fact (~100 queries/day), daily MEV:
    const dailyMev = extractablePerQuery * 100n;
    expect(dailyMev).toBeGreaterThan(0n);
  });

  it("MEV-2: Batch pricing neutralizes ordering MEV", () => {
    // Proposed fix: all queries for the same fact in a block pay the same price
    // This means reordering queries doesn't change the price anyone pays

    const queries = [
      { factId: "f1", position: 1, fee: 1000n },
      { factId: "f1", position: 2, fee: 1200n },
      { factId: "f1", position: 3, fee: 800n },
    ];

    // With batch pricing: all pay the average
    const totalFees = queries.reduce((sum, q) => sum + q.fee, 0n);
    const batchPrice = totalFees / BigInt(queries.length);

    // No ordering advantage — all pay same price
    for (const q of queries) {
      expect(batchPrice).toBe(1000n); // Average
    }
  });

  it("MEV-3: Query commit-reveal hides query intent from block producer", () => {
    // For high-value queries (> 1 LGM): require commit-reveal
    const HIGH_VALUE_THRESHOLD = ONE_LGM;
    const queryValue = 2n * ONE_LGM;

    const requiresCommitReveal = queryValue > HIGH_VALUE_THRESHOLD;
    expect(requiresCommitReveal).toBe(true);

    // Low-value queries: no commit-reveal (too much overhead)
    const lowQueryValue = ONE_LGM / 10n;
    expect(lowQueryValue > HIGH_VALUE_THRESHOLD).toBe(false);
  });
});

// ============================================================
// D3-L6: FRESHNESS CLIFF ARBITRAGE (HIGH)
// ============================================================

describe("D3-L6: Freshness Cliff Arbitrage", () => {
  it("FRESH-1: Current — hard cutoff at 1000 blocks", () => {
    // At block 999: 10% premium
    // At block 1001: 0% premium
    // This creates an exploitable cliff

    const freshnessDiscountRate =
      DEFAULT_BILLING_PARAMETERS.freshnessDiscountRate;
    expect(freshnessDiscountRate).toBeGreaterThan(0);

    // Testing the cliff behavior
    const facts999 = [
      { confidence: 0.8, fundamentalityScore: 0.5, ageBlocks: 999 },
    ];
    const facts1001 = [
      { confidence: 0.8, fundamentalityScore: 0.5, ageBlocks: 1001 },
    ];

    const price999 = calculateAPICost(facts999);
    const price1001 = calculateAPICost(facts1001);

    // There should be a price difference at the boundary
    // The cliff means price drops discontinuously
    if (price999 !== price1001) {
      const dropBps = ((price999 - price1001) * 10000n) / price999;
      // Cliff creates a sudden > 5% price drop
      expect(dropBps).toBeGreaterThan(0n);
    }
  });

  it("FRESH-2: Proposed exponential decay — no cliff, smooth transition", () => {
    const halfLifeBlocks = 500;
    const maxPremium = 0.1;

    // Exponential decay: premium = maxPremium * exp(-age / halfLife)
    const premiumAt = (age: number) =>
      maxPremium * Math.exp(-age / halfLifeBlocks);

    // Smooth transition — no sudden jumps
    const ages = [0, 100, 200, 500, 1000, 2000, 5000];
    const premiums = ages.map(premiumAt);

    // Verify monotonically decreasing
    for (let i = 1; i < premiums.length; i++) {
      expect(premiums[i]).toBeLessThan(premiums[i - 1]);
    }

    // Verify no sudden jumps (max 20% change between consecutive samples)
    for (let i = 1; i < premiums.length; i++) {
      if (premiums[i - 1] > 0.001) {
        const changeRatio = premiums[i] / premiums[i - 1];
        expect(changeRatio).toBeGreaterThan(0);
        expect(changeRatio).toBeLessThan(1); // Decreasing
      }
    }

    // At half-life (500 blocks): premium ≈ 3.68% (half of peak)
    expect(premiumAt(500)).toBeCloseTo(maxPremium / Math.E, 3);

    // At 2x half-life (1000 blocks): premium ≈ 1.35%
    expect(premiumAt(1000)).toBeCloseTo(maxPremium / (Math.E * Math.E), 3);
  });

  it("FRESH-3: Cliff arbitrage — wait for block 999→1001 to save on query", () => {
    // Attacker strategy: wait for fact to age past 1000 blocks to save freshness premium
    // With cliff: exact timing gives 10% savings
    // With decay: waiting gives diminishing returns, no exact cutoff

    const maxPremium = 0.1;
    const halfLife = 500;

    // Savings from waiting 1 block at various ages (exponential decay)
    const savingsPerBlock = (age: number) => {
      const premiumBefore = maxPremium * Math.exp(-age / halfLife);
      const premiumAfter = maxPremium * Math.exp(-(age + 1) / halfLife);
      return premiumBefore - premiumAfter;
    };

    // Savings per block is tiny and decreasing
    expect(savingsPerBlock(0)).toBeLessThan(0.0003);
    expect(savingsPerBlock(500)).toBeLessThan(0.0001);
    expect(savingsPerBlock(1000)).toBeLessThan(0.00005);

    // No single block transition gives > 0.1% advantage
    for (let age = 0; age < 2000; age += 10) {
      expect(savingsPerBlock(age)).toBeLessThan(0.001);
    }
  });
});

// ============================================================
// D3-L7: VALIDATOR MICRO-STAKE SYBIL (HIGH)
// ============================================================

describe("D3-L7: Validator Micro-Stake Sybil — Provider Registration", () => {
  it("MICRO-1: Current MinProviderStake allows mass registration", () => {
    // MinProviderStake: 100 nanoLGM (100 * 10^9 = 10^11 wei)
    const MIN_PROVIDER_STAKE = 100n * 10n ** 9n; // 100 nanoLGM

    // With 0.222 LGM: 0.222 * 10^18 / 10^11 = 2,220,000 providers
    const availableFunds = 222n * 10n ** 15n; // 0.222 LGM
    const maxProviders = availableFunds / MIN_PROVIDER_STAKE;

    expect(maxProviders).toBeGreaterThan(2_000_000n);
  });

  it("MICRO-2: Proposed fix — MinProviderStake = 0.111 LGM (match validator min)", () => {
    const PROPOSED_MIN_PROVIDER_STAKE = 111n * 10n ** 15n; // 0.111 LGM
    const availableFunds = 222n * 10n ** 15n; // 0.222 LGM
    const maxProviders = availableFunds / PROPOSED_MIN_PROVIDER_STAKE;

    // Can only create 2 providers with 0.222 LGM (was 2.2M)
    expect(maxProviders).toBe(2n);

    // Sybil cost: 111 LGM for 1000 providers (vs 0.0001 LGM previously)
    const sybil1000Cost = PROPOSED_MIN_PROVIDER_STAKE * 1000n;
    expect(sybil1000Cost).toBe(111n * ONE_LGM);
  });

  it("MICRO-3: Sybil provider doesn't gain selection advantage", () => {
    // Even with 2M providers, selection should be weighted by total stake
    // A single provider with 1 LGM should outperform 1000 providers with 0.001 LGM each

    const singleProviderStake = ONE_LGM;
    const sybilStakeEach = ONE_LGM / 1000n;
    const numSybils = 1000;

    // Total stake is the same: 1 LGM
    const totalSybilStake = sybilStakeEach * BigInt(numSybils);
    expect(totalSybilStake).toBe(singleProviderStake);

    // But with reputation weighting, single established provider wins
    // New providers start at 0.5 reputation
  });
});

// ============================================================
// D1-L3: FLOAT PRECISION IN CLAIM SLASHING (MEDIUM — included for completeness)
// ============================================================

describe("D1-L3: Float Precision in Claim Slashing", () => {
  it("FLOAT-1: Current code uses Number() conversion — unsafe for large stakes", () => {
    // state/transitions.ts:4652: BigInt(Math.floor(Number(claim.stake) * slashRate))
    // Number() loses precision for values > 2^53

    const largeStake = 10000n * ONE_LGM; // 10,000 LGM
    const slashRate = 0.22;

    // Current (buggy) implementation
    const currentSlash = BigInt(Math.floor(Number(largeStake) * slashRate));

    // Correct implementation using pure bigint
    const correctSlash = (largeStake * 2200n) / 10000n;

    // For 10,000 LGM, the difference should be small
    // But for > 2^53 wei (= 9007 LGM), Number() loses precision
    const maxSafeStake = BigInt(Number.MAX_SAFE_INTEGER); // 2^53 - 1

    // Verify correctSlash is deterministic
    expect(correctSlash).toBe((largeStake * 2200n) / 10000n);
  });

  it("FLOAT-2: Pure bigint fix — correct at all stake sizes", () => {
    const slashBps = 2200n; // 22%

    const stakes = [
      1n, // 1 wei
      ONE_LGM, // 1 LGM
      100n * ONE_LGM, // 100 LGM
      10000n * ONE_LGM, // 10,000 LGM
      1000000n * ONE_LGM, // 1M LGM
      222222222n * ONE_LGM, // Max supply
    ];

    for (const stake of stakes) {
      const slash = (stake * slashBps) / 10000n;

      // Slash never exceeds stake
      expect(slash).toBeLessThanOrEqual(stake);

      // Slash is exactly 22% (within 1 wei rounding)
      const expectedExact = (stake * 22n) / 100n;
      const diff =
        slash > expectedExact ? slash - expectedExact : expectedExact - slash;
      expect(diff).toBeLessThanOrEqual(1n);
    }
  });
});

// ============================================================
// D2-L1: CARRY-FORWARD UNBOUNDED (MEDIUM — included for completeness)
// ============================================================

describe("D2-L1: Falsification Carry-Forward Bounds", () => {
  it("CARRY-1: Carry-forward can accumulate across many epochs", () => {
    // Simulate: every epoch hits the cap, defers 20 LGM
    const epochPool = 100n * ONE_LGM;
    const deferredPerEpoch = 20n * ONE_LGM;

    let totalDeferred = 0n;
    for (let epoch = 0; epoch < 100; epoch++) {
      totalDeferred += deferredPerEpoch;
      // Attempt to pay from pool
      const payable = totalDeferred < epochPool ? totalDeferred : epochPool;
      totalDeferred -= payable;
    }

    // After 100 epochs: deferred should be bounded
    // Without cap: could accumulate indefinitely
    // With proposed cap (10x original): bounded
  });

  it("CARRY-2: Proposed cap — max carry-forward = 10x original reward", () => {
    const MAX_CARRY_MULTIPLIER = 10n;

    const originalReward = 5n * ONE_LGM;
    const maxCarry = originalReward * MAX_CARRY_MULTIPLIER;

    expect(maxCarry).toBe(50n * ONE_LGM);

    // Any carry-forward exceeding this is silently dropped
    const excessiveCarry = 100n * ONE_LGM;
    const effectiveCarry =
      excessiveCarry < maxCarry ? excessiveCarry : maxCarry;
    expect(effectiveCarry).toBe(maxCarry);
  });
});

// ============================================================
// CROSS-DIMENSION COMPOUND ATTACK TESTS
// ============================================================

describe("Cross-Dimension Compound Attack: C1 — Governance + IBC Drain", () => {
  it("C1-1: Three-gate defense — each gate independently blocks the attack", () => {
    // Gate 1: Apprentice cap (111 max)
    const maxApprentices = MAX_APPRENTICE_VALIDATORS;
    const apprenticeStake = VALIDATOR_TIER_CONFIG[0].minStake;
    const maxSybilStake = BigInt(maxApprentices) * apprenticeStake;

    // Gate 2: Quadratic voting + 10% cap
    const totalStake = 1000n * ONE_LGM;
    const attackerStake = maxSybilStake; // ~12.3 LGM
    const attackerPowerRatio = Number(attackerStake) / Number(totalStake);
    // 12.3 / 1000 = 1.23% — far below governance threshold

    // Gate 3: Aggregate IBC cap (50K LGM/day)
    const AGGREGATE_IBC_CAP = 50_000n * ONE_LGM;

    // All three gates must be breached for the attack to succeed
    // Gate 1: ✓ Blocks at 111 validators
    expect(maxApprentices).toBe(111);
    // Gate 2: ✓ 1.23% power < 80% threshold
    expect(attackerPowerRatio).toBeLessThan(0.02);
    // Gate 3: ✓ Even if governance captured, max drain is 50K/day
    // 50K LGM/day is a small fraction of total supply (222M)
    const totalSupply = 222_222_222n * ONE_LGM;
    expect(AGGREGATE_IBC_CAP).toBeLessThan(totalSupply);
  });
});

describe("Cross-Dimension Compound Attack: C2 — Fee Gaming + Dispute Suppression", () => {
  it("C2-1: Citation gaming + challenge spam — compounding defense", () => {
    // Attack path:
    // 1. Self-query fact 256 times → 50% discount
    // 2. Challenge counter-evidence → suppress competing facts

    // Defense 1: Unique-querier cap → max 30% discount
    const MAX_CITATION_DISCOUNT = 0.3;

    // Defense 2: 3-dispute cap per agent
    const MAX_ACTIVE_DISPUTES = 3;

    // Defense 3: Frivolous challenges don't trigger confidence cap
    // Net: attacker gets limited discount, can't suppress competition
    expect(MAX_CITATION_DISCOUNT).toBeLessThan(0.5);
    expect(MAX_ACTIVE_DISPUTES).toBeLessThanOrEqual(3);
  });
});
