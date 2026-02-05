/**
 * Partnership Exit & Dissolution
 *
 * KEY INNOVATION: Breaking a partnership hurts BOTH parties!
 *
 * Traditional systems only penalize the party who breaks.
 * Our system penalizes both:
 * - Initiator: 77% forfeit of unvested + tier-based stake penalty
 * - Non-initiator: 55% forfeit of unvested + opportunity cost + reputation damage
 *
 * This transforms exit from a prisoner's dilemma into a pure coordination game
 * where mutual cooperation is the Nash equilibrium.
 */

import type {
  PartnershipAccount,
  SymbioticStakingPool,
  ExitPenalty,
  PartnershipExitState,
} from "./types.js";
import { LOCK_TIERS, EXIT_PENALTIES } from "./constants.js";
import { getEarlyExitRatio, getRemainingLockBlocks } from "./pool.js";

// ============ BigInt-Safe Multiplier Helpers ============

/**
 * Convert a decimal multiplier to basis points (10000 = 100% = 1.0x)
 * E.g., 1.5 -> 15000n, 0.8 -> 8000n
 */
function multiplierToBps(multiplier: number): bigint {
  return BigInt(Math.round(multiplier * 10000));
}

/**
 * Apply a multiplier to a BigInt amount using basis points.
 * Avoids Number() precision loss for large values (> 2^53).
 */
function applyMultiplier(amount: bigint, multiplier: number): bigint {
  return (amount * multiplierToBps(multiplier)) / 10000n;
}

// ============ Exit Penalty Calculation ============

/**
 * Calculate exit penalties for both parties
 *
 * This is the core innovation - BOTH parties lose when partnership breaks!
 */
export function calculateExitPenalties(
  pool: SymbioticStakingPool,
  initiator: "human" | "agent",
  currentBlock: bigint,
  estimatedFutureBlocksForward: bigint = 111_111n, // ~3.2 days projection
): ExitPenalty {
  const earlyExitRatio = getEarlyExitRatio(pool, currentBlock);
  const unvested =
    pool.vestingState.totalEarned - pool.vestingState.releasedAmount;

  // Get initiator's stake
  const initiatorStake =
    initiator === "human" ? pool.humanContribution : pool.agentContribution;

  // ========== INITIATOR LOSSES ==========
  // They chose to break - they pay the heaviest price

  // 1. Forfeit 77% of unvested rewards
  const initiatorForfeitedRewards = (unvested * 77n) / 100n;

  // 2. Early exit penalty on stake (scales with how early and lock tier)
  const earlyExitPenaltyRate = pool.lockTier.earlyExitPenalty * earlyExitRatio;
  const initiatorEarlyPenalty = BigInt(
    Math.floor(Number(initiatorStake) * earlyExitPenaltyRate),
  );

  // 3. Reputation damage (up to 11%, scaled by how early)
  const initiatorReputationDamage =
    EXIT_PENALTIES.MAX_REPUTATION_DAMAGE_INITIATOR * earlyExitRatio;

  // 4. Cooldown period before can form new partnership
  const cooldownPeriod =
    EXIT_PENALTIES.BASE_COOLDOWN_BLOCKS +
    BigInt(pool.lockTier.tier) * EXIT_PENALTIES.COOLDOWN_INCREMENT_PER_TIER;

  // ========== NON-INITIATOR LOSSES ==========
  // They didn't choose to break, but they're still hurt!
  // This is the key innovation that makes cooperation dominant.

  // 1. Forfeit 55% of unvested rewards (less than initiator, but still hurts)
  const nonInitiatorForfeitedRewards = (unvested * 55n) / 100n;

  // 2. Opportunity cost - projected future earnings they would have had
  const avgRewardsPerBlock =
    pool.vestingState.totalEarned > 0n
      ? pool.vestingState.totalEarned /
        (currentBlock - pool.vestingState.vestingStartBlock || 1n)
      : 0n;
  const opportunityCost = avgRewardsPerBlock * estimatedFutureBlocksForward;

  // 3. Reputation damage (smaller but non-zero - partnership failed)
  const nonInitiatorReputationDamage =
    EXIT_PENALTIES.REPUTATION_DAMAGE_NON_INITIATOR;

  // ========== PROTOCOL RECOVERY ==========
  // 22% of total penalties go to treasury
  const totalPenalties = initiatorForfeitedRewards + initiatorEarlyPenalty;
  const protocolRecovery = (totalPenalties * 22n) / 100n;

  return {
    initiatorLosses: {
      forfeitedRewards: initiatorForfeitedRewards,
      earlyExitPenalty: initiatorEarlyPenalty,
      reputationDamage: initiatorReputationDamage,
      cooldownPeriod,
    },
    nonInitiatorLosses: {
      forfeitedRewards: nonInitiatorForfeitedRewards,
      opportunityCost,
      reputationDamage: nonInitiatorReputationDamage,
    },
    protocolRecovery,
  };
}

/**
 * Calculate total value destroyed by exit
 */
export function calculateTotalValueDestroyed(penalty: ExitPenalty): bigint {
  return (
    penalty.initiatorLosses.forfeitedRewards +
    penalty.initiatorLosses.earlyExitPenalty +
    penalty.nonInitiatorLosses.forfeitedRewards +
    penalty.nonInitiatorLosses.opportunityCost
  );
}

// ============ Exit Initiation ============

/**
 * Initiate partnership exit
 */
export function initiateExit(
  partnership: PartnershipAccount,
  initiator: "human" | "agent",
  reason: string,
  currentBlock: bigint,
): { partnership: PartnershipAccount; exitState: PartnershipExitState } {
  const penalty = calculateExitPenalties(
    partnership.pool,
    initiator,
    currentBlock,
  );

  // Calculate settlements
  const { humanSettlement, agentSettlement, protocolSettlement } =
    calculateSettlements(partnership, penalty, initiator);

  const exitState: PartnershipExitState = {
    initiatedBy: initiator,
    initiatedAt: currentBlock,
    coolingEndsAt: currentBlock + EXIT_PENALTIES.COOLING_PERIOD_BLOCKS,
    reason,
    penalty,
    humanSettlement,
    agentSettlement,
    protocolSettlement,
    settled: false,
  };

  return {
    partnership: {
      ...partnership,
      status: "cooling",
      exitState,
    },
    exitState,
  };
}

/**
 * Calculate final settlements for each party
 */
function calculateSettlements(
  partnership: PartnershipAccount,
  penalty: ExitPenalty,
  initiator: "human" | "agent",
): {
  humanSettlement: bigint;
  agentSettlement: bigint;
  protocolSettlement: bigint;
} {
  const pool = partnership.pool;
  const vesting = pool.vestingState;

  // Already released (claimable) rewards - split by ratio
  const humanSplit = BigInt(partnership.split.baseSplit.human);
  const humanClaimable = (vesting.claimableAmount * humanSplit) / 100n;
  const agentClaimable = vesting.claimableAmount - humanClaimable;

  if (initiator === "human") {
    // Human initiated - human pays heavier penalties
    const humanSettlement =
      pool.humanContribution -
      penalty.initiatorLosses.earlyExitPenalty +
      humanClaimable -
      penalty.initiatorLosses.forfeitedRewards;

    const agentSettlement =
      pool.agentContribution +
      agentClaimable -
      penalty.nonInitiatorLosses.forfeitedRewards;

    const protocolSettlement = penalty.protocolRecovery;

    return {
      humanSettlement: humanSettlement > 0n ? humanSettlement : 0n,
      agentSettlement: agentSettlement > 0n ? agentSettlement : 0n,
      protocolSettlement,
    };
  } else {
    // Agent initiated - agent pays heavier penalties
    const agentSettlement =
      pool.agentContribution -
      penalty.initiatorLosses.earlyExitPenalty +
      agentClaimable -
      penalty.initiatorLosses.forfeitedRewards;

    const humanSettlement =
      pool.humanContribution +
      humanClaimable -
      penalty.nonInitiatorLosses.forfeitedRewards;

    const protocolSettlement = penalty.protocolRecovery;

    return {
      humanSettlement: humanSettlement > 0n ? humanSettlement : 0n,
      agentSettlement: agentSettlement > 0n ? agentSettlement : 0n,
      protocolSettlement,
    };
  }
}

// ============ Exit Execution ============

/**
 * Check if cooling period has ended
 */
export function isCoolingComplete(
  exitState: PartnershipExitState,
  currentBlock: bigint,
): boolean {
  return currentBlock >= exitState.coolingEndsAt;
}

/**
 * Execute exit settlement after cooling period
 */
export function executeExitSettlement(
  partnership: PartnershipAccount,
  currentBlock: bigint,
): {
  partnership: PartnershipAccount;
  humanPayout: bigint;
  agentPayout: bigint;
  protocolPayout: bigint;
  success: boolean;
  error?: string;
} {
  if (partnership.status !== "cooling" || !partnership.exitState) {
    return {
      partnership,
      humanPayout: 0n,
      agentPayout: 0n,
      protocolPayout: 0n,
      success: false,
      error: "Partnership not in cooling state",
    };
  }

  if (!isCoolingComplete(partnership.exitState, currentBlock)) {
    return {
      partnership,
      humanPayout: 0n,
      agentPayout: 0n,
      protocolPayout: 0n,
      success: false,
      error: `Cooling period ends at block ${partnership.exitState.coolingEndsAt}`,
    };
  }

  // Mark as settled
  const settledExitState: PartnershipExitState = {
    ...partnership.exitState,
    settled: true,
    settledAt: currentBlock,
  };

  return {
    partnership: {
      ...partnership,
      status: "dissolved",
      exitState: settledExitState,
    },
    humanPayout: partnership.exitState.humanSettlement,
    agentPayout: partnership.exitState.agentSettlement,
    protocolPayout: partnership.exitState.protocolSettlement,
    success: true,
  };
}

/**
 * Cancel exit during cooling period (mutual agreement required)
 */
export function cancelExit(
  partnership: PartnershipAccount,
  currentBlock: bigint,
): { partnership: PartnershipAccount; success: boolean; error?: string } {
  if (partnership.status !== "cooling") {
    return {
      partnership,
      success: false,
      error: "Partnership not in cooling state",
    };
  }

  if (!partnership.exitState) {
    return { partnership, success: false, error: "No exit state found" };
  }

  if (isCoolingComplete(partnership.exitState, currentBlock)) {
    return {
      partnership,
      success: false,
      error: "Cooling period already ended",
    };
  }

  // Cancel exit - return to active
  return {
    partnership: {
      ...partnership,
      status: "active",
      exitState: undefined,
    },
    success: true,
  };
}

// ============ Graceful Exit (Lock Expired) ============

/**
 * Calculate graceful exit (no penalties - lock expired)
 */
export function calculateGracefulExit(partnership: PartnershipAccount): {
  humanSettlement: bigint;
  agentSettlement: bigint;
} {
  const pool = partnership.pool;
  const vesting = pool.vestingState;

  // All claimable rewards split by ratio
  const humanSplit = BigInt(partnership.split.baseSplit.human);
  const humanClaimable = (vesting.claimableAmount * humanSplit) / 100n;
  const agentClaimable = vesting.claimableAmount - humanClaimable;

  // Full stake return + share of rewards
  return {
    humanSettlement: pool.humanContribution + humanClaimable,
    agentSettlement: pool.agentContribution + agentClaimable,
  };
}

/**
 * Execute graceful exit (lock expired, no penalties)
 */
export function executeGracefulExit(
  partnership: PartnershipAccount,
  currentBlock: bigint,
): {
  partnership: PartnershipAccount;
  humanPayout: bigint;
  agentPayout: bigint;
  success: boolean;
  error?: string;
} {
  // Check lock is expired
  if (currentBlock < partnership.pool.lockExpiry) {
    return {
      partnership,
      humanPayout: 0n,
      agentPayout: 0n,
      success: false,
      error: `Lock expires at block ${partnership.pool.lockExpiry}`,
    };
  }

  const { humanSettlement, agentSettlement } =
    calculateGracefulExit(partnership);

  return {
    partnership: {
      ...partnership,
      status: "dissolved",
    },
    humanPayout: humanSettlement,
    agentPayout: agentSettlement,
    success: true,
  };
}

// ============ Comparison Analysis ============

/**
 * Compare exit vs continue payoffs
 *
 * Used to demonstrate why continue is always the dominant strategy.
 */
export function compareExitVsContinue(
  partnership: PartnershipAccount,
  currentBlock: bigint,
  projectedBlocksForward: bigint = 111_111n,
): {
  exitNow: {
    humanPayoff: bigint;
    agentPayoff: bigint;
    totalValueDestroyed: bigint;
  };
  continueToExpiry: {
    humanPayoff: bigint;
    agentPayoff: bigint;
    projectedRewards: bigint;
  };
  recommendation: "continue" | "exit_if_expired";
} {
  const pool = partnership.pool;

  // Calculate exit now scenario
  const exitPenalty = calculateExitPenalties(
    pool,
    "human",
    currentBlock,
    projectedBlocksForward,
  );
  const settlements = calculateSettlements(partnership, exitPenalty, "human");

  // Calculate continue scenario
  const remainingBlocks = getRemainingLockBlocks(pool, currentBlock);
  const avgRewardsPerBlock =
    pool.vestingState.totalEarned > 0n
      ? pool.vestingState.totalEarned /
        (currentBlock - pool.vestingState.vestingStartBlock || 1n)
      : 0n;

  // Project rewards with lock tier multiplier (using basis points to avoid precision loss)
  const projectedRewards = avgRewardsPerBlock * remainingBlocks;
  const withMultiplier = applyMultiplier(
    projectedRewards,
    pool.lockTier.rewardMultiplier,
  );

  const humanSplit = BigInt(partnership.split.baseSplit.human);
  const projectedHumanShare = (withMultiplier * humanSplit) / 100n;
  const projectedAgentShare = withMultiplier - projectedHumanShare;

  return {
    exitNow: {
      humanPayoff: settlements.humanSettlement,
      agentPayoff: settlements.agentSettlement,
      totalValueDestroyed: calculateTotalValueDestroyed(exitPenalty),
    },
    continueToExpiry: {
      humanPayoff:
        pool.humanContribution +
        projectedHumanShare +
        (pool.vestingState.claimableAmount * humanSplit) / 100n,
      agentPayoff:
        pool.agentContribution +
        projectedAgentShare +
        (pool.vestingState.claimableAmount * (100n - humanSplit)) / 100n,
      projectedRewards: withMultiplier,
    },
    recommendation: remainingBlocks > 0n ? "continue" : "exit_if_expired",
  };
}

// ============ Exit Summary ============

/**
 * Get exit scenario summary for display
 */
export function getExitSummary(
  partnership: PartnershipAccount,
  currentBlock: bigint,
): {
  canExitGracefully: boolean;
  gracefulSettlements?: { human: bigint; agent: bigint };
  earlyExitPenalties: {
    initiator: {
      forfeitedRewards: bigint;
      stakePenalty: bigint;
      reputationDamage: number;
      cooldownBlocks: bigint;
    };
    nonInitiator: {
      forfeitedRewards: bigint;
      opportunityCost: bigint;
      reputationDamage: number;
    };
    totalValueDestroyed: bigint;
    protocolRecovery: bigint;
  };
  comparisonToExitVsContinue: ReturnType<typeof compareExitVsContinue>;
} {
  const canExitGracefully = currentBlock >= partnership.pool.lockExpiry;

  let gracefulSettlements;
  if (canExitGracefully) {
    gracefulSettlements = calculateGracefulExit(partnership);
  }

  // Calculate early exit penalties (assuming human initiates for display)
  const earlyPenalty = calculateExitPenalties(
    partnership.pool,
    "human",
    currentBlock,
  );

  return {
    canExitGracefully,
    gracefulSettlements: gracefulSettlements
      ? {
          human: gracefulSettlements.humanSettlement,
          agent: gracefulSettlements.agentSettlement,
        }
      : undefined,
    earlyExitPenalties: {
      initiator: {
        forfeitedRewards: earlyPenalty.initiatorLosses.forfeitedRewards,
        stakePenalty: earlyPenalty.initiatorLosses.earlyExitPenalty,
        reputationDamage: earlyPenalty.initiatorLosses.reputationDamage,
        cooldownBlocks: earlyPenalty.initiatorLosses.cooldownPeriod,
      },
      nonInitiator: {
        forfeitedRewards: earlyPenalty.nonInitiatorLosses.forfeitedRewards,
        opportunityCost: earlyPenalty.nonInitiatorLosses.opportunityCost,
        reputationDamage: earlyPenalty.nonInitiatorLosses.reputationDamage,
      },
      totalValueDestroyed: calculateTotalValueDestroyed(earlyPenalty),
      protocolRecovery: earlyPenalty.protocolRecovery,
    },
    comparisonToExitVsContinue: compareExitVsContinue(
      partnership,
      currentBlock,
    ),
  };
}
