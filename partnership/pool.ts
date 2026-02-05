/**
 * Symbiotic Staking Pool (SSP) Operations
 *
 * The core financial mechanism that binds human-agent partners together.
 * Breaking the pool hurts BOTH parties - continuing enriches BOTH.
 */

import type {
  SymbioticStakingPool,
  PartnershipTier,
  LockTier,
  PoolVestingState,
  PoolHealth,
  ComputeInvestment,
} from "./types.js";
import { LOCK_TIERS, FORMATION, POOL_HEALTH, VESTING } from "./constants.js";

// ============ Pool Creation ============

/**
 * Create a new Symbiotic Staking Pool
 */
export function createPool(
  humanContribution: bigint,
  agentContribution: bigint,
  lockTier: PartnershipTier,
  currentBlock: bigint,
): SymbioticStakingPool {
  const tier = LOCK_TIERS[lockTier];

  return {
    humanContribution,
    agentContribution,
    sharedAppreciation: 0n,

    lockTier: tier,
    lockExpiry: currentBlock + tier.lockBlocks,
    renewalCount: 0,

    vestingState: createInitialVestingState(currentBlock),
    computeInvestment: createInitialComputeInvestment(),
    health: createInitialHealth(currentBlock),
  };
}

/**
 * Create initial vesting state
 */
function createInitialVestingState(currentBlock: bigint): PoolVestingState {
  return {
    totalEarned: 0n,
    releasedAmount: 0n,
    claimableAmount: 0n,
    vestingStartBlock: currentBlock,
    accelerationBonus: 0,
    fromVerification: 0n,
    fromComputeContribution: 0n,
    fromStakingYield: 0n,
    fromBonuses: 0n,
  };
}

/**
 * Create initial compute investment
 */
function createInitialComputeInvestment(): ComputeInvestment {
  return {
    totalInvested: 0n,
    periods: [],
    currentPeriodId: 0,
    computeEfficiency: 0,
    verificationContribution: 0,
  };
}

/**
 * Create initial health
 */
function createInitialHealth(currentBlock: bigint): PoolHealth {
  return {
    score: POOL_HEALTH.MAX_SCORE,
    lastActivityBlock: currentBlock,
    inactivityWarning: false,
    warningFlags: [],
  };
}

// ============ Pool Validation ============

/**
 * Validate pool contribution amounts
 */
export function validateContributions(
  humanContribution: bigint,
  agentContribution: bigint,
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  if (humanContribution < FORMATION.MIN_CONTRIBUTION) {
    errors.push(
      `Human contribution ${humanContribution} below minimum ${FORMATION.MIN_CONTRIBUTION}`,
    );
  }

  if (agentContribution < FORMATION.MIN_CONTRIBUTION) {
    errors.push(
      `Agent contribution ${agentContribution} below minimum ${FORMATION.MIN_CONTRIBUTION}`,
    );
  }

  // Contributions should be roughly balanced (within 3x of each other)
  const ratio = Number(humanContribution) / Number(agentContribution);
  if (ratio > 3 || ratio < 1 / 3) {
    errors.push("Contribution imbalance too large (max 3x ratio)");
  }

  return { valid: errors.length === 0, errors };
}

// ============ Pool Operations ============

/**
 * Add contribution to pool (after formation)
 */
export function addToPool(
  pool: SymbioticStakingPool,
  contributor: "human" | "agent",
  amount: bigint,
  currentBlock: bigint,
): SymbioticStakingPool {
  const updated = { ...pool };

  if (contributor === "human") {
    updated.humanContribution = pool.humanContribution + amount;
  } else {
    updated.agentContribution = pool.agentContribution + amount;
  }

  // Update health (activity)
  updated.health = updateHealthOnActivity(pool.health, currentBlock);

  return updated;
}

/**
 * Calculate total pool value
 */
export function getTotalPoolValue(pool: SymbioticStakingPool): bigint {
  return (
    pool.humanContribution +
    pool.agentContribution +
    pool.sharedAppreciation +
    pool.vestingState.claimableAmount
  );
}

/**
 * Calculate each party's current value
 */
export function getPartyValues(
  pool: SymbioticStakingPool,
  humanSplitPercent: number,
  agentSplitPercent: number,
): { humanValue: bigint; agentValue: bigint } {
  const sharedValue =
    pool.sharedAppreciation + pool.vestingState.claimableAmount;
  const humanShared = (sharedValue * BigInt(humanSplitPercent)) / 100n;
  const agentShared = sharedValue - humanShared;

  return {
    humanValue: pool.humanContribution + humanShared,
    agentValue: pool.agentContribution + agentShared,
  };
}

// ============ Lock Tier Management ============

/**
 * Upgrade to higher lock tier
 */
export function upgradeLockTier(
  pool: SymbioticStakingPool,
  newTier: PartnershipTier,
  currentBlock: bigint,
): { pool: SymbioticStakingPool; success: boolean; error?: string } {
  // Validate upgrade
  if (newTier <= pool.lockTier.tier) {
    return { pool, success: false, error: "Can only upgrade to higher tier" };
  }

  if (newTier > 5) {
    return { pool, success: false, error: "Maximum tier is 5" };
  }

  const newLockTier = LOCK_TIERS[newTier];

  // Calculate new expiry (extend from current block)
  const newExpiry = currentBlock + newLockTier.lockBlocks;

  const updated: SymbioticStakingPool = {
    ...pool,
    lockTier: newLockTier,
    lockExpiry: newExpiry,
    health: updateHealthOnActivity(pool.health, currentBlock),
  };

  return { pool: updated, success: true };
}

/**
 * Renew lock at current tier
 */
export function renewLock(
  pool: SymbioticStakingPool,
  currentBlock: bigint,
): { pool: SymbioticStakingPool; success: boolean; error?: string } {
  // Can only renew if within renewal window (last 10% of lock)
  const renewalWindowStart = pool.lockExpiry - pool.lockTier.lockBlocks / 10n;

  if (currentBlock < renewalWindowStart) {
    return {
      pool,
      success: false,
      error: "Can only renew in last 10% of lock period",
    };
  }

  const newExpiry = currentBlock + pool.lockTier.lockBlocks;

  const updated: SymbioticStakingPool = {
    ...pool,
    lockExpiry: newExpiry,
    renewalCount: pool.renewalCount + 1,
    health: updateHealthOnActivity(pool.health, currentBlock),
  };

  return { pool: updated, success: true };
}

/**
 * Check if pool lock has expired
 */
export function isLockExpired(
  pool: SymbioticStakingPool,
  currentBlock: bigint,
): boolean {
  return currentBlock >= pool.lockExpiry;
}

/**
 * Get remaining lock time in blocks
 */
export function getRemainingLockBlocks(
  pool: SymbioticStakingPool,
  currentBlock: bigint,
): bigint {
  if (currentBlock >= pool.lockExpiry) return 0n;
  return pool.lockExpiry - currentBlock;
}

/**
 * Calculate early exit ratio (how early is exit relative to lock)
 */
export function getEarlyExitRatio(
  pool: SymbioticStakingPool,
  currentBlock: bigint,
): number {
  const remaining = getRemainingLockBlocks(pool, currentBlock);
  return Number(remaining) / Number(pool.lockTier.lockBlocks);
}

// ============ Reward Operations ============

/**
 * Add rewards to pool vesting
 */
export function addRewardsToPool(
  pool: SymbioticStakingPool,
  amount: bigint,
  source: "verification" | "compute" | "staking" | "bonus",
  currentBlock: bigint,
): SymbioticStakingPool {
  const vesting = { ...pool.vestingState };

  vesting.totalEarned = vesting.totalEarned + amount;

  switch (source) {
    case "verification":
      vesting.fromVerification = vesting.fromVerification + amount;
      break;
    case "compute":
      vesting.fromComputeContribution =
        vesting.fromComputeContribution + amount;
      break;
    case "staking":
      vesting.fromStakingYield = vesting.fromStakingYield + amount;
      break;
    case "bonus":
      vesting.fromBonuses = vesting.fromBonuses + amount;
      break;
  }

  return {
    ...pool,
    sharedAppreciation: pool.sharedAppreciation + amount,
    vestingState: vesting,
    health: updateHealthOnActivity(pool.health, currentBlock),
  };
}

/**
 * Update vesting release based on time
 */
export function updateVestingRelease(
  pool: SymbioticStakingPool,
  currentBlock: bigint,
): SymbioticStakingPool {
  const vesting = { ...pool.vestingState };

  // Calculate blocks since vesting start
  const blocksSinceStart = currentBlock - vesting.vestingStartBlock;

  // Skip if still in cliff period
  if (blocksSinceStart < VESTING.BASE_CLIFF_BLOCKS) {
    return pool;
  }

  // Calculate release using exponential decay
  // R(t) = R_total × (1 - e^(-λt)) where λ = ln(2) / half_life
  const effectiveHalfLife =
    Number(VESTING.BASE_VESTING_BLOCKS) / (1 + vesting.accelerationBonus);
  const lambda = Math.log(2) / effectiveHalfLife;
  const t = Number(blocksSinceStart - VESTING.BASE_CLIFF_BLOCKS);

  const releaseRatio = 1 - Math.exp(-lambda * t);
  const shouldBeReleased = BigInt(
    Math.floor(Number(vesting.totalEarned) * releaseRatio),
  );

  // Calculate newly claimable
  const newlyReleased = shouldBeReleased - vesting.releasedAmount;

  if (newlyReleased > 0n) {
    vesting.releasedAmount = shouldBeReleased;
    vesting.claimableAmount = vesting.claimableAmount + newlyReleased;
  }

  return { ...pool, vestingState: vesting };
}

/**
 * Claim vested rewards
 */
export function claimVestedRewards(
  pool: SymbioticStakingPool,
  amount: bigint,
  currentBlock: bigint,
): { pool: SymbioticStakingPool; claimed: bigint; error?: string } {
  const vesting = { ...pool.vestingState };

  if (amount > vesting.claimableAmount) {
    return {
      pool,
      claimed: 0n,
      error: `Requested ${amount} but only ${vesting.claimableAmount} claimable`,
    };
  }

  // Minimum claim check
  if (
    amount < VESTING.MIN_RELEASE_PER_CLAIM &&
    amount !== vesting.claimableAmount
  ) {
    return {
      pool,
      claimed: 0n,
      error: `Minimum claim is ${VESTING.MIN_RELEASE_PER_CLAIM}`,
    };
  }

  vesting.claimableAmount = vesting.claimableAmount - amount;

  return {
    pool: {
      ...pool,
      vestingState: vesting,
      health: updateHealthOnActivity(pool.health, currentBlock),
    },
    claimed: amount,
  };
}

/**
 * Apply acceleration bonus to vesting
 */
export function applyAccelerationBonus(
  pool: SymbioticStakingPool,
  bonusIncrease: number,
): SymbioticStakingPool {
  const newBonus = Math.min(
    VESTING.MAX_ACCELERATION,
    pool.vestingState.accelerationBonus + bonusIncrease,
  );

  return {
    ...pool,
    vestingState: {
      ...pool.vestingState,
      accelerationBonus: newBonus,
    },
  };
}

// ============ Health Management ============

/**
 * Update health on activity
 */
function updateHealthOnActivity(
  health: PoolHealth,
  currentBlock: bigint,
): PoolHealth {
  const newScore = Math.min(
    POOL_HEALTH.MAX_SCORE,
    health.score + POOL_HEALTH.ACTIVITY_GAIN,
  );

  return {
    score: newScore,
    lastActivityBlock: currentBlock,
    inactivityWarning: false,
    warningFlags: health.warningFlags.filter((f) => f !== "inactivity"),
  };
}

/**
 * Check and update pool health for inactivity
 */
export function checkPoolHealth(
  pool: SymbioticStakingPool,
  currentBlock: bigint,
): SymbioticStakingPool {
  const blocksSinceActivity = currentBlock - pool.health.lastActivityBlock;
  const health = { ...pool.health };

  // Check inactivity levels
  if (blocksSinceActivity >= POOL_HEALTH.CRITICAL_INACTIVITY_BLOCKS) {
    health.score = Math.max(
      POOL_HEALTH.CRITICAL_THRESHOLD,
      health.score - POOL_HEALTH.INACTIVITY_DECAY * 3,
    );
    health.inactivityWarning = true;
    if (!health.warningFlags.includes("critical_inactivity")) {
      health.warningFlags.push("critical_inactivity");
    }
  } else if (blocksSinceActivity >= POOL_HEALTH.INACTIVITY_WARNING_BLOCKS) {
    health.score = Math.max(
      POOL_HEALTH.WARNING_THRESHOLD,
      health.score - POOL_HEALTH.INACTIVITY_DECAY,
    );
    health.inactivityWarning = true;
    if (!health.warningFlags.includes("inactivity")) {
      health.warningFlags.push("inactivity");
    }
  }

  return { ...pool, health };
}

/**
 * Get pool health status
 */
export function getPoolHealthStatus(
  pool: SymbioticStakingPool,
): "healthy" | "warning" | "critical" | "suspended" {
  if (pool.health.score >= POOL_HEALTH.MIN_HEALTHY_SCORE) return "healthy";
  if (pool.health.score >= POOL_HEALTH.WARNING_THRESHOLD) return "warning";
  if (pool.health.score >= POOL_HEALTH.CRITICAL_THRESHOLD) return "critical";
  return "suspended";
}

// ============ Compute Investment ============

/**
 * Record compute investment in pool
 */
export function recordComputeInvestment(
  pool: SymbioticStakingPool,
  lgmInvested: bigint,
  currentBlock: bigint,
): SymbioticStakingPool {
  const compute = { ...pool.computeInvestment };

  compute.totalInvested = compute.totalInvested + lgmInvested;

  // Add or update current period
  const currentPeriod = compute.periods.find((p) => p.status === "active");

  if (currentPeriod) {
    currentPeriod.lgmInvested = currentPeriod.lgmInvested + lgmInvested;
  } else {
    compute.periods.push({
      periodId: compute.currentPeriodId++,
      startBlock: currentBlock,
      endBlock: 0n, // Set on settlement
      lgmInvested,
      computeUnitsAcquired: 0n,
      verificationsPerformed: 0,
      verificationAccuracy: 0,
      rewardsGenerated: 0n,
      roi: 0,
      status: "active",
    });
  }

  return {
    ...pool,
    computeInvestment: compute,
    health: updateHealthOnActivity(pool.health, currentBlock),
  };
}

// ============ Pool Summary ============

/**
 * Get pool summary for display
 */
export function getPoolSummary(
  pool: SymbioticStakingPool,
  currentBlock: bigint,
): {
  totalValue: bigint;
  lockedUntil: bigint;
  remainingLockBlocks: bigint;
  lockTier: number;
  rewardMultiplier: number;
  vestingProgress: number;
  healthStatus: string;
  earlyExitRatio: number;
} {
  return {
    totalValue: getTotalPoolValue(pool),
    lockedUntil: pool.lockExpiry,
    remainingLockBlocks: getRemainingLockBlocks(pool, currentBlock),
    lockTier: pool.lockTier.tier,
    rewardMultiplier: pool.lockTier.rewardMultiplier,
    vestingProgress:
      Number(pool.vestingState.releasedAmount) /
      Number(pool.vestingState.totalEarned || 1n),
    healthStatus: getPoolHealthStatus(pool),
    earlyExitRatio: getEarlyExitRatio(pool, currentBlock),
  };
}
