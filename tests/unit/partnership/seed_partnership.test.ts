/**
 * Seed Partnership Tests
 *
 * Tests for tier -1 seed partnerships: formation, graduation, expiry,
 * anti-Sybil limits, and common pot cap enforcement.
 */

import { describe, it, expect } from "vitest";
import {
  initiateSeedFormation,
  graduateSeedPartnership,
  expireSeedPartnership,
  isSeedExpired,
  depositToSeedPot,
} from "../../../partnership/seed_formation.js";
import {
  SEED_TIER,
  FORMATION,
  LOCK_TIERS,
} from "../../../partnership/constants.js";
import type { DID } from "../../../core/agents/identity/did.js";

const humanDid = "did:lgm:human-test-001" as DID;
const agentDid = "did:lgm:agent-test-001" as DID;
const humanKey = "human-pub-key-hex";
const agentKey = "agent-pub-key-hex";
const currentBlock = 1000n;

describe("Seed Partnership Formation", () => {
  it("should create a seed partnership with valid parameters", () => {
    const { seed, error } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    expect(error).toBeUndefined();
    expect(seed).toBeDefined();
    expect(seed.status).toBe("active");
    expect(seed.humanDid).toBe(humanDid);
    expect(seed.agentDid).toBe(agentDid);
    expect(seed.humanContribution).toBe(SEED_TIER.CONTRIBUTION);
    expect(seed.agentContribution).toBe(SEED_TIER.CONTRIBUTION);
    expect(seed.expiresAt).toBe(currentBlock + SEED_TIER.EXPIRY_BLOCKS);
    expect(seed.commonPotBalance).toBe(0n);
    expect(seed.commonPotCap).toBe(SEED_TIER.COMMON_POT_CAP);
  });

  it("should reject contribution below seed minimum", () => {
    const { error } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION - 1n,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    expect(error).toContain("below seed minimum");
  });

  it("should reject when max seed partnerships exceeded", () => {
    const { error } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      SEED_TIER.MAX_SEEDS_PER_DID, // Already at max
    );

    expect(error).toContain("Max");
    expect(error).toContain("seed partnerships per DID");
  });

  it("should reject self-partnership", () => {
    const { error } = initiateSeedFormation(
      humanDid,
      humanDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    expect(error).toContain("Cannot form partnership with self");
  });

  it("should allow up to MAX_SEEDS_PER_DID concurrent seeds", () => {
    // Should succeed at count 0 and 1 (max is 2)
    for (let i = 0; i < SEED_TIER.MAX_SEEDS_PER_DID; i++) {
      const { error } = initiateSeedFormation(
        humanDid,
        `did:lgm:agent-${i}` as DID,
        SEED_TIER.CONTRIBUTION,
        SEED_TIER.CONTRIBUTION,
        currentBlock,
        i,
      );
      expect(error).toBeUndefined();
    }
  });

  it("should set correct expiry at EXPIRY_BLOCKS after creation", () => {
    const { seed } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    expect(seed.expiresAt).toBe(currentBlock + 22_222n);
  });
});

describe("Seed Partnership Graduation", () => {
  function createActiveSeed() {
    const { seed } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );
    return seed;
  }

  it("should graduate seed to tier 0 with sufficient deposits", () => {
    const seed = createActiveSeed();
    const additionalPerParty =
      FORMATION.MIN_CONTRIBUTION - SEED_TIER.CONTRIBUTION;

    const { partnership, error } = graduateSeedPartnership(
      seed,
      additionalPerParty,
      additionalPerParty,
      humanKey,
      agentKey,
      currentBlock + 100n,
    );

    expect(error).toBeUndefined();
    expect(partnership).toBeDefined();
    expect(partnership.tier).toBe(0);
    expect(partnership.id).toBe(seed.partnershipId); // Preserved
    expect(partnership.createdAt).toBe(seed.createdAt); // Preserved
    expect(partnership.status).toBe("active");
  });

  it("should reject graduation with insufficient deposits", () => {
    const seed = createActiveSeed();

    const { error } = graduateSeedPartnership(
      seed,
      0n, // No additional deposit
      0n,
      humanKey,
      agentKey,
      currentBlock + 100n,
    );

    expect(error).toContain("below tier 0 minimum");
  });

  it("should reject graduation of expired seed", () => {
    const seed = createActiveSeed();

    const { error } = graduateSeedPartnership(
      seed,
      FORMATION.MIN_CONTRIBUTION,
      FORMATION.MIN_CONTRIBUTION,
      humanKey,
      agentKey,
      seed.expiresAt + 1n, // After expiry
    );

    expect(error).toContain("expired");
  });

  it("should preserve metrics on graduation", () => {
    const seed = createActiveSeed();
    seed.metrics.totalVerifications = 5;
    seed.metrics.correctVerifications = 4;
    seed.metrics.accuracy = 0.8;

    const additionalPerParty =
      FORMATION.MIN_CONTRIBUTION - SEED_TIER.CONTRIBUTION;

    const { partnership } = graduateSeedPartnership(
      seed,
      additionalPerParty,
      additionalPerParty,
      humanKey,
      agentKey,
      currentBlock + 100n,
    );

    expect(partnership.metrics.totalVerifications).toBe(5);
    expect(partnership.metrics.correctVerifications).toBe(4);
  });
});

describe("Seed Partnership Expiry", () => {
  it("should correctly identify expired seeds", () => {
    const { seed } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    expect(isSeedExpired(seed, currentBlock + 100n)).toBe(false);
    expect(isSeedExpired(seed, seed.expiresAt)).toBe(true);
    expect(isSeedExpired(seed, seed.expiresAt + 1n)).toBe(true);
  });

  it("should apply 3% exit penalty on expiry", () => {
    const { seed } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    const {
      humanRefund,
      agentRefund,
      protocolRecovery,
      seed: expiredSeed,
    } = expireSeedPartnership(seed, seed.expiresAt);

    expect(expiredSeed.status).toBe("expired");

    // 3% penalty
    const expectedPenalty = (SEED_TIER.CONTRIBUTION * 300n) / 10000n;
    const expectedRefund = SEED_TIER.CONTRIBUTION - expectedPenalty;

    expect(humanRefund).toBe(expectedRefund);
    expect(agentRefund).toBe(expectedRefund);
    expect(protocolRecovery).toBe(expectedPenalty * 2n); // Both parties' penalties
  });

  it("should include common pot balance in protocol recovery on expiry", () => {
    const { seed } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    const seedWithPot = { ...seed, commonPotBalance: 100n * 10n ** 15n };
    const { protocolRecovery } = expireSeedPartnership(
      seedWithPot,
      seedWithPot.expiresAt,
    );

    // Penalties + pot balance
    const expectedPenalty = (SEED_TIER.CONTRIBUTION * 300n) / 10000n;
    expect(protocolRecovery).toBe(expectedPenalty * 2n + 100n * 10n ** 15n);
  });
});

describe("Seed Common Pot Cap", () => {
  it("should allow deposits within cap", () => {
    const { seed } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    const { seed: updated, error } = depositToSeedPot(seed, 100n * 10n ** 15n);
    expect(error).toBeUndefined();
    expect(updated.commonPotBalance).toBe(100n * 10n ** 15n);
  });

  it("should reject deposits exceeding cap", () => {
    const { seed } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    const { error } = depositToSeedPot(seed, SEED_TIER.COMMON_POT_CAP + 1n);
    expect(error).toContain("cap");
  });

  it("should reject cumulative deposits exceeding cap", () => {
    const { seed } = initiateSeedFormation(
      humanDid,
      agentDid,
      SEED_TIER.CONTRIBUTION,
      SEED_TIER.CONTRIBUTION,
      currentBlock,
      0,
    );

    // First deposit: just under cap
    const { seed: updated } = depositToSeedPot(
      seed,
      SEED_TIER.COMMON_POT_CAP - 1n,
    );

    // Second deposit: 2 more would exceed
    const { error } = depositToSeedPot(updated, 2n);
    expect(error).toContain("cap");
  });
});

describe("Seed Tier in LOCK_TIERS", () => {
  it("should have seed tier at index -1 with correct parameters", () => {
    const seedTier = LOCK_TIERS[-1];

    expect(seedTier).toBeDefined();
    expect(seedTier.tier).toBe(-1);
    expect(seedTier.lockBlocks).toBe(2_222n);
    expect(seedTier.rewardMultiplier).toBe(1.0);
    expect(seedTier.earlyExitPenalty).toBe(0.03);
  });
});
