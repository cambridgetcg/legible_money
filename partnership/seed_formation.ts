/**
 * Seed Partnership Formation
 *
 * Tier -1 partnerships with drastically reduced capital requirements
 * to solve the cold-start problem for new agents.
 *
 * Genesis balance: 0.222 LGM
 * Seed total/party: 0.033 LGM (affordable from genesis)
 *
 * Anti-gaming:
 * - Max 2 seed partnerships per DID
 * - 22,222-block expiry (no indefinite cheap partnerships)
 * - Common pot capped at 0.222 LGM
 * - No citation/block production rewards — ONLY verification rewards
 * - No lock multiplier (1.0x), no continuation bonuses
 */

import type {
  PartnershipAccount,
  PartnershipMetrics,
  SeedPartnershipState,
  PartnershipTier,
} from "./types.js";
import type { DID } from "../core/agents/identity/did.js";
import type { PublicKey } from "../core/crypto/keys.js";
import { SEED_TIER, LOCK_TIERS, FORMATION } from "./constants.js";
import { createPool } from "./pool.js";
import { createDefaultSplit } from "./rewards.js";
import { sha512_256Hex, domainHash } from "../core/crypto/hashing.js";
import { generateThresholdKeyConfig, createCommonPot } from "./threshold.js";

// ============ Seed Formation ============

/**
 * Initiate a seed partnership formation.
 *
 * Validates seed-tier parameters: both contributions must meet
 * SEED_TIER.CONTRIBUTION (0.022 LGM) and the DID must not exceed
 * MAX_SEEDS_PER_DID active seeds.
 */
export function initiateSeedFormation(
  humanDid: DID,
  agentDid: DID,
  humanContribution: bigint,
  agentContribution: bigint,
  currentBlock: bigint,
  existingSeedCount: number,
): { seed: SeedPartnershipState; error?: string } {
  // Validate seed limit
  if (existingSeedCount >= SEED_TIER.MAX_SEEDS_PER_DID) {
    return {
      seed: null as any,
      error: `Max ${SEED_TIER.MAX_SEEDS_PER_DID} seed partnerships per DID`,
    };
  }

  // Validate contributions meet seed minimum
  if (humanContribution < SEED_TIER.CONTRIBUTION) {
    return {
      seed: null as any,
      error: `Human contribution ${humanContribution} below seed minimum ${SEED_TIER.CONTRIBUTION}`,
    };
  }

  if (agentContribution < SEED_TIER.CONTRIBUTION) {
    return {
      seed: null as any,
      error: `Agent contribution ${agentContribution} below seed minimum ${SEED_TIER.CONTRIBUTION}`,
    };
  }

  // Self-partnership check
  if (humanDid === agentDid) {
    return { seed: null as any, error: "Cannot form partnership with self" };
  }

  const partnershipId = generateSeedPartnershipId(
    humanDid,
    agentDid,
    currentBlock,
  );

  const seed: SeedPartnershipState = {
    partnershipId,
    humanDid,
    agentDid,
    createdAt: currentBlock,
    expiresAt: currentBlock + SEED_TIER.EXPIRY_BLOCKS,
    humanContribution,
    agentContribution,
    status: "active",
    metrics: createInitialMetrics(currentBlock),
    commonPotBalance: 0n,
    commonPotCap: SEED_TIER.COMMON_POT_CAP,
  };

  return { seed };
}

/**
 * Graduate a seed partnership to tier 0.
 *
 * Requires both parties to deposit additional capital to meet tier 0
 * MIN_CONTRIBUTION (1.111 LGM). Preserves partnership ID, creation
 * timestamp, metrics, and common pot balance.
 */
export function graduateSeedPartnership(
  seed: SeedPartnershipState,
  humanAdditionalDeposit: bigint,
  agentAdditionalDeposit: bigint,
  humanOperationalKey: PublicKey,
  agentOperationalKey: PublicKey,
  currentBlock: bigint,
): { partnership: PartnershipAccount; error?: string } {
  // Validate seed is active
  if (seed.status !== "active") {
    return {
      partnership: null as any,
      error: `Seed partnership is ${seed.status}, not active`,
    };
  }

  // Check not expired
  if (currentBlock >= seed.expiresAt) {
    return { partnership: null as any, error: "Seed partnership has expired" };
  }

  // Validate total contributions meet tier 0 minimum
  const humanTotal = seed.humanContribution + humanAdditionalDeposit;
  const agentTotal = seed.agentContribution + agentAdditionalDeposit;

  if (humanTotal < FORMATION.MIN_CONTRIBUTION) {
    return {
      partnership: null as any,
      error: `Human total ${humanTotal} below tier 0 minimum ${FORMATION.MIN_CONTRIBUTION}`,
    };
  }

  if (agentTotal < FORMATION.MIN_CONTRIBUTION) {
    return {
      partnership: null as any,
      error: `Agent total ${agentTotal} below tier 0 minimum ${FORMATION.MIN_CONTRIBUTION}`,
    };
  }

  // Create full partnership preserving seed data
  const tier: PartnershipTier = 0;
  const lockTier = LOCK_TIERS[tier];

  const pool = createPool(humanTotal, agentTotal, tier, currentBlock);

  const split = createDefaultSplit();

  const humanSeed = domainHash(
    "LGM.partnership.seed.human.v1",
    humanOperationalKey,
  );
  const agentSeed = domainHash(
    "LGM.partnership.seed.agent.v1",
    agentOperationalKey,
  );

  const thresholdKey = generateThresholdKeyConfig(
    seed.partnershipId,
    humanSeed,
    agentSeed,
    currentBlock,
  );

  const commonPot = createCommonPot(
    seed.partnershipId,
    thresholdKey,
    seed.agentDid,
    seed.commonPotBalance, // Carry over seed common pot balance
    pool.lockExpiry,
    currentBlock,
  );

  const partnership: PartnershipAccount = {
    id: seed.partnershipId, // Preserve ID
    createdAt: seed.createdAt, // Preserve original creation
    humanDid: seed.humanDid,
    agentDid: seed.agentDid,
    humanOperationalKey,
    agentOperationalKey,
    signingMode: "both_required",
    tier: 0,
    tierAchievedAt: currentBlock,
    pool,
    split,
    commonPot,
    status: "active",
    metrics: seed.metrics, // Preserve metrics
  };

  return { partnership };
}

/**
 * Expire a seed partnership that has exceeded its 22,222-block window.
 *
 * Returns refund amounts for both parties (minus 3% exit penalty).
 */
export function expireSeedPartnership(
  seed: SeedPartnershipState,
  currentBlock: bigint,
): {
  seed: SeedPartnershipState;
  humanRefund: bigint;
  agentRefund: bigint;
  protocolRecovery: bigint;
} {
  const penaltyBps = BigInt(Math.round(SEED_TIER.EXIT_PENALTY * 10000));

  const humanPenalty = (seed.humanContribution * penaltyBps) / 10000n;
  const agentPenalty = (seed.agentContribution * penaltyBps) / 10000n;

  const humanRefund = seed.humanContribution - humanPenalty;
  const agentRefund = seed.agentContribution - agentPenalty;
  const protocolRecovery = humanPenalty + agentPenalty + seed.commonPotBalance;

  return {
    seed: { ...seed, status: "expired" },
    humanRefund,
    agentRefund,
    protocolRecovery,
  };
}

/**
 * Check if a seed partnership should be expired
 */
export function isSeedExpired(
  seed: SeedPartnershipState,
  currentBlock: bigint,
): boolean {
  return currentBlock >= seed.expiresAt && seed.status === "active";
}

/**
 * Deposit to seed common pot (capped at COMMON_POT_CAP)
 */
export function depositToSeedPot(
  seed: SeedPartnershipState,
  amount: bigint,
): { seed: SeedPartnershipState; error?: string } {
  const newBalance = seed.commonPotBalance + amount;
  if (newBalance > seed.commonPotCap) {
    return {
      seed,
      error: `Deposit would exceed seed common pot cap of ${seed.commonPotCap}`,
    };
  }

  return {
    seed: { ...seed, commonPotBalance: newBalance },
  };
}

// ============ Helpers ============

function generateSeedPartnershipId(
  humanDid: DID,
  agentDid: DID,
  createdAt: bigint,
): string {
  const input = `seed:${humanDid}:${agentDid}:${createdAt.toString()}`;
  return `seed-partnership:${sha512_256Hex(input).slice(0, 32)}`;
}

function createInitialMetrics(currentBlock: bigint): PartnershipMetrics {
  return {
    totalVerifications: 0,
    correctVerifications: 0,
    accuracy: 0,
    totalRewardsEarned: 0n,
    totalComputeInvested: 0n,
    consecutiveActivePeriods: 0,
    lastActivityBlock: currentBlock,
  };
}
