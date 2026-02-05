/**
 * Partnership Reward Distribution
 *
 * Handles profit splitting between human and agent partners.
 * Default split: 50% human / 50% agent (equal partnership)
 *
 * Reward flow:
 * 1. Verification reward comes in
 * 2. Lock tier multiplier applied
 * 3. 22% goes to co-owned common pot (2-of-2 threshold key)
 * 4. Remaining 78% split 50/50 between human and agent
 * 5. Individual shares added to vesting (released over time)
 *
 * The common pot is co-owned: neither party can withdraw without
 * the other's signature. This creates shared skin-in-the-game
 * and funds joint operations (compute investment, tier upgrades).
 */

import type {
  PartnershipAccount,
  ProfitSplit,
  SplitAdjustments,
  SplitChange,
  PartnershipRewardInput,
  PartnershipRewardDistribution,
  ContinuationBonus,
  ContinuationMilestone,
} from "./types.js";
import {
  FORMATION,
  SPLIT_ADJUSTMENTS,
  REWARD_DISTRIBUTION,
  CONTINUATION_MILESTONES,
  LIFETIME_BOOST,
  MAX_CONTINUATION_BONUS,
  VALIDATION,
} from "./constants.js";
import { addRewardsToPool, applyAccelerationBonus } from "./pool.js";

// ============ Split Creation ============

/**
 * Create default profit split
 */
export function createDefaultSplit(): ProfitSplit {
  return {
    baseSplit: {
      human: FORMATION.DEFAULT_HUMAN_SPLIT,
      agent: FORMATION.DEFAULT_AGENT_SPLIT,
    },
    adjustments: {
      humanExpertisePremium: 0,
      agentAvailabilityPremium: 0,
      agentComputePremium: 0,
      longevityBonus: 0,
    },
    splitHistory: [],
  };
}

/**
 * Create custom profit split
 */
export function createCustomSplit(
  humanPercent: number,
  agentPercent: number,
): { split: ProfitSplit | null; error?: string } {
  // Validate percentages
  if (humanPercent + agentPercent !== 100) {
    return { split: null, error: "Split must total 100%" };
  }

  if (
    humanPercent < FORMATION.MIN_SPLIT ||
    humanPercent > FORMATION.MAX_SPLIT
  ) {
    return {
      split: null,
      error: `Human split must be between ${FORMATION.MIN_SPLIT}% and ${FORMATION.MAX_SPLIT}%`,
    };
  }

  if (
    agentPercent < FORMATION.MIN_SPLIT ||
    agentPercent > FORMATION.MAX_SPLIT
  ) {
    return {
      split: null,
      error: `Agent split must be between ${FORMATION.MIN_SPLIT}% and ${FORMATION.MAX_SPLIT}%`,
    };
  }

  return {
    split: {
      baseSplit: { human: humanPercent, agent: agentPercent },
      adjustments: {
        humanExpertisePremium: 0,
        agentAvailabilityPremium: 0,
        agentComputePremium: 0,
        longevityBonus: 0,
      },
      splitHistory: [],
    },
  };
}

// ============ Split Calculation ============

/**
 * Calculate effective split with all adjustments
 */
export function calculateEffectiveSplit(split: ProfitSplit): {
  human: number;
  agent: number;
} {
  const { baseSplit, adjustments } = split;

  // Calculate raw totals with adjustments
  let humanTotal =
    baseSplit.human +
    adjustments.humanExpertisePremium * 100 +
    adjustments.longevityBonus * 50; // Half to each

  let agentTotal =
    baseSplit.agent +
    adjustments.agentAvailabilityPremium * 100 +
    adjustments.agentComputePremium * 100 +
    adjustments.longevityBonus * 50; // Half to each

  // Cap individual contributions
  humanTotal = Math.min(FORMATION.MAX_SPLIT, humanTotal);
  agentTotal = Math.min(FORMATION.MAX_SPLIT, agentTotal);

  // Normalize to 100%
  const total = humanTotal + agentTotal;
  const normalizedHuman = Math.round((humanTotal / total) * 100);
  const normalizedAgent = 100 - normalizedHuman;

  return {
    human: normalizedHuman,
    agent: normalizedAgent,
  };
}

// ============ Split Adjustment ============

/**
 * Update split adjustments
 */
export function updateSplitAdjustments(
  split: ProfitSplit,
  newAdjustments: Partial<SplitAdjustments>,
  currentBlock: bigint,
): ProfitSplit {
  const previousEffective = calculateEffectiveSplit(split);

  const updatedAdjustments: SplitAdjustments = {
    humanExpertisePremium: Math.min(
      newAdjustments.humanExpertisePremium ??
        split.adjustments.humanExpertisePremium,
      SPLIT_ADJUSTMENTS.MAX_HUMAN_EXPERTISE_PREMIUM,
    ),
    agentAvailabilityPremium: Math.min(
      newAdjustments.agentAvailabilityPremium ??
        split.adjustments.agentAvailabilityPremium,
      SPLIT_ADJUSTMENTS.MAX_AGENT_AVAILABILITY_PREMIUM,
    ),
    agentComputePremium: Math.min(
      newAdjustments.agentComputePremium ??
        split.adjustments.agentComputePremium,
      SPLIT_ADJUSTMENTS.MAX_AGENT_COMPUTE_PREMIUM,
    ),
    longevityBonus: Math.min(
      newAdjustments.longevityBonus ?? split.adjustments.longevityBonus,
      SPLIT_ADJUSTMENTS.MAX_LONGEVITY_BONUS,
    ),
  };

  const newSplit: ProfitSplit = {
    ...split,
    adjustments: updatedAdjustments,
  };

  const newEffective = calculateEffectiveSplit(newSplit);

  // Record change if effective split changed
  if (previousEffective.human !== newEffective.human) {
    const change: SplitChange = {
      previousSplit: previousEffective,
      newSplit: newEffective,
      reason: "Adjustment update",
      changedAt: currentBlock,
      agreedByHuman: true,
      agreedByAgent: true,
    };
    newSplit.splitHistory = [...split.splitHistory, change];
  }

  return newSplit;
}

/**
 * Request base split change (requires mutual agreement)
 */
export function proposeSplitChange(
  split: ProfitSplit,
  newHumanPercent: number,
  newAgentPercent: number,
  proposer: "human" | "agent",
  reason: string,
  currentBlock: bigint,
): { split: ProfitSplit; pendingChange: SplitChange } | { error: string } {
  // Validate
  if (newHumanPercent + newAgentPercent !== 100) {
    return { error: "Split must total 100%" };
  }

  if (
    newHumanPercent < FORMATION.MIN_SPLIT ||
    newHumanPercent > FORMATION.MAX_SPLIT
  ) {
    return {
      error: `Human split must be between ${FORMATION.MIN_SPLIT}% and ${FORMATION.MAX_SPLIT}%`,
    };
  }

  const previousEffective = calculateEffectiveSplit(split);

  const pendingChange: SplitChange = {
    previousSplit: previousEffective,
    newSplit: { human: newHumanPercent, agent: newAgentPercent },
    reason,
    changedAt: currentBlock,
    agreedByHuman: proposer === "human",
    agreedByAgent: proposer === "agent",
  };

  return { split, pendingChange };
}

// ============ Reward Distribution ============

/**
 * Distribute reward to partnership
 *
 * Flow: gross → lock multiplier → 22% common pot + 78% split 50/50
 * Common pot uses 2-of-2 threshold keys — neither party can withdraw alone.
 */
export function distributePartnershipReward(
  partnership: PartnershipAccount,
  input: PartnershipRewardInput,
  currentBlock: bigint,
): {
  partnership: PartnershipAccount;
  distribution: PartnershipRewardDistribution;
} {
  // Step 1: Apply lock tier multiplier to gross amount
  const multiplier = partnership.pool.lockTier.rewardMultiplier;
  const withMultiplier = BigInt(
    Math.floor(Number(input.grossAmount) * multiplier),
  );

  // Step 2: Route 22% to co-owned common pot
  const commonPotShare = (withMultiplier * 22n) / 100n;
  const individualPool = withMultiplier - commonPotShare;

  // Step 3: Split remaining 78% by partnership ratio (default 50/50)
  const effectiveSplit = calculateEffectiveSplit(partnership.split);
  const humanShare = (individualPool * BigInt(effectiveSplit.human)) / 100n;
  const agentShare = individualPool - humanShare;

  // Step 4: Deposit common pot share (no consensus needed for deposits)
  const updatedCommonPot = {
    ...partnership.commonPot,
    balance: partnership.commonPot.balance + commonPotShare,
  };

  // Step 5: Add individual shares to pool vesting
  const sourceType =
    input.sourceType === "verification"
      ? "verification"
      : input.sourceType === "compute"
        ? "compute"
        : input.sourceType === "staking"
          ? "staking"
          : "bonus";

  const updatedPool = addRewardsToPool(
    partnership.pool,
    individualPool,
    sourceType,
    currentBlock,
  );

  // Step 6: Update metrics
  const updatedMetrics = {
    ...partnership.metrics,
    totalRewardsEarned: partnership.metrics.totalRewardsEarned + withMultiplier,
    lastActivityBlock: currentBlock,
  };

  // Create distribution record
  const distribution: PartnershipRewardDistribution = {
    totalPartnershipReward: withMultiplier,
    humanShare,
    agentShare,
    commonPotShare,
    lockBonus: multiplier - 1,
    appliedAt: currentBlock,
    vestingAdded: individualPool,
  };

  return {
    partnership: {
      ...partnership,
      pool: updatedPool,
      commonPot: updatedCommonPot,
      metrics: updatedMetrics,
    },
    distribution,
  };
}

/**
 * Route evidence tax to treasury funds
 */
export function routeEvidenceTax(taxAmount: bigint): {
  research: bigint;
  foundation: bigint;
  community: bigint;
  developers: bigint;
  reserve: bigint;
} {
  return {
    research: (taxAmount * 77n) / 100n, // 77% to research
    foundation: (taxAmount * 7n) / 100n, // 7% to foundation
    community: (taxAmount * 7n) / 100n, // 7% to community
    developers: (taxAmount * 7n) / 100n, // 7% to developers
    reserve: (taxAmount * 2n) / 100n, // 2% to reserve
  };
}

// ============ Continuation Bonuses ============

/**
 * Calculate continuation bonus for partnership
 */
export function calculateContinuationBonus(
  partnership: PartnershipAccount,
  currentBlock: bigint,
): ContinuationBonus {
  const partnershipDuration = currentBlock - partnership.createdAt;

  // Find current milestone tier
  let currentTier = 0;
  let accumulatedBonus = 0;
  let nextMilestone: ContinuationMilestone | undefined;

  for (const milestone of CONTINUATION_MILESTONES) {
    if (partnershipDuration >= milestone.blocks) {
      currentTier = milestone.tier;
      accumulatedBonus += milestone.bonus;
    } else {
      nextMilestone = milestone;
      break;
    }
  }

  // Check lifetime boost eligibility
  const lifetimeBoostActive =
    partnershipDuration >= LIFETIME_BOOST.THRESHOLD_BLOCKS;

  // Calculate total multiplier
  let totalMultiplier = 1 + accumulatedBonus;
  if (lifetimeBoostActive) {
    totalMultiplier += LIFETIME_BOOST.MULTIPLIER - 1;
  }

  // Cap at max
  totalMultiplier = Math.min(1 + MAX_CONTINUATION_BONUS, totalMultiplier);

  return {
    bonusTier: currentTier as 0 | 1 | 2 | 3 | 4 | 5,
    bonusMultiplier: totalMultiplier,
    nextMilestoneBlock: nextMilestone
      ? partnership.createdAt + nextMilestone.blocks
      : partnership.createdAt +
        CONTINUATION_MILESTONES[CONTINUATION_MILESTONES.length - 1].blocks,
    nextMilestoneBonus: nextMilestone?.bonus ?? 0,
    lifetimeBoostActive,
    lifetimeBoostMultiplier: lifetimeBoostActive
      ? LIFETIME_BOOST.MULTIPLIER
      : 1,
  };
}

/**
 * Apply continuation bonus to partnership
 */
export function applyContinuationBonus(
  partnership: PartnershipAccount,
  currentBlock: bigint,
): PartnershipAccount {
  const bonus = calculateContinuationBonus(partnership, currentBlock);

  // Update tier if progressed
  if (bonus.bonusTier > partnership.tier) {
    // Apply acceleration bonus for milestone achievement
    const accelerationIncrease =
      SPLIT_ADJUSTMENTS.LONGEVITY_BONUS_PER_MILESTONE;
    const updatedPool = applyAccelerationBonus(
      partnership.pool,
      accelerationIncrease,
    );

    // Update longevity bonus in split
    const updatedSplit = updateSplitAdjustments(
      partnership.split,
      {
        longevityBonus:
          bonus.bonusTier * SPLIT_ADJUSTMENTS.LONGEVITY_BONUS_PER_MILESTONE,
      },
      currentBlock,
    );

    return {
      ...partnership,
      tier: bonus.bonusTier,
      tierAchievedAt: currentBlock,
      pool: updatedPool,
      split: updatedSplit,
    };
  }

  return partnership;
}

// ============ Reward Claiming ============

/**
 * Calculate claimable rewards for each party
 */
export function calculateClaimableRewards(partnership: PartnershipAccount): {
  humanClaimable: bigint;
  agentClaimable: bigint;
} {
  const claimable = partnership.pool.vestingState.claimableAmount;
  const effectiveSplit = calculateEffectiveSplit(partnership.split);

  const humanClaimable = (claimable * BigInt(effectiveSplit.human)) / 100n;
  const agentClaimable = claimable - humanClaimable;

  return { humanClaimable, agentClaimable };
}

/**
 * Process reward claim
 */
export function processRewardClaim(
  partnership: PartnershipAccount,
  claimer: "human" | "agent",
  amount: bigint,
  currentBlock: bigint,
): { partnership: PartnershipAccount; claimed: bigint; error?: string } {
  const { humanClaimable, agentClaimable } =
    calculateClaimableRewards(partnership);
  const maxClaimable = claimer === "human" ? humanClaimable : agentClaimable;

  if (amount > maxClaimable) {
    return {
      partnership,
      claimed: 0n,
      error: `Requested ${amount} but only ${maxClaimable} claimable`,
    };
  }

  // Update vesting state
  const updatedVesting = { ...partnership.pool.vestingState };
  updatedVesting.claimableAmount = updatedVesting.claimableAmount - amount;

  return {
    partnership: {
      ...partnership,
      pool: {
        ...partnership.pool,
        vestingState: updatedVesting,
      },
    },
    claimed: amount,
  };
}

// ============ Reward Summary ============

/**
 * Get comprehensive reward summary
 */
export function getRewardSummary(
  partnership: PartnershipAccount,
  currentBlock: bigint,
): {
  totalEarned: bigint;
  totalVested: bigint;
  totalClaimable: bigint;
  humanClaimable: bigint;
  agentClaimable: bigint;
  effectiveSplit: { human: number; agent: number };
  currentMultiplier: number;
  continuationBonus: ContinuationBonus;
  vestingProgress: number;
  sourceBreakdown: {
    verification: bigint;
    compute: bigint;
    staking: bigint;
    bonuses: bigint;
  };
} {
  const vesting = partnership.pool.vestingState;
  const effectiveSplit = calculateEffectiveSplit(partnership.split);
  const { humanClaimable, agentClaimable } =
    calculateClaimableRewards(partnership);
  const continuationBonus = calculateContinuationBonus(
    partnership,
    currentBlock,
  );

  return {
    totalEarned: vesting.totalEarned,
    totalVested: vesting.releasedAmount,
    totalClaimable: vesting.claimableAmount,
    humanClaimable,
    agentClaimable,
    effectiveSplit,
    currentMultiplier: partnership.pool.lockTier.rewardMultiplier,
    continuationBonus,
    vestingProgress:
      vesting.totalEarned > 0n
        ? Number(vesting.releasedAmount) / Number(vesting.totalEarned)
        : 0,
    sourceBreakdown: {
      verification: vesting.fromVerification,
      compute: vesting.fromComputeContribution,
      staking: vesting.fromStakingYield,
      bonuses: vesting.fromBonuses,
    },
  };
}
