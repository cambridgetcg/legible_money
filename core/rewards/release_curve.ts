/**
 * Release Curve Calculation
 *
 * Implements exponential release formula based on scientometric research:
 *
 *   R(t) = R_total × (1 - e^(-λt)) × max_release_rate
 *
 * Where:
 *   λ = ln(2) / half_life
 *   t = blocks elapsed since cliff ended (minus paused time)
 *
 * This mirrors how scientific knowledge naturally decays and gets replaced,
 * with different rates for different epistemic categories.
 *
 * PP HIERARCHY:
 *   TRUTH: Formula grounded in actual knowledge half-life research
 *   UNDERSTANDING: Exponential decay matches observed citation patterns
 *   BEAUTY: Simple formula with intuitive parameters
 *   JUSTICE: More stable categories get slower release, more time to challenge
 */

import {
  type RewardVesting,
  type ReleaseCalculation,
  getVestingConfig,
} from "./types.js";

// Maximum bigint value safe for Number() conversion in exponential math.
// Beyond this, the release ratio is effectively 1.0 (fully released).
const MAX_SAFE_BLOCKS = BigInt(Number.MAX_SAFE_INTEGER);

// ============ Core Formula ============

/**
 * Calculate decay constant (λ) from half-life
 *
 * λ = ln(2) / half_life
 *
 * This is the same formula used in radioactive decay,
 * and matches knowledge obsolescence patterns.
 *
 * H1 FIX: Overflow guard for extreme half-life values.
 */
export function calculateDecayConstant(halfLifeBlocks: bigint): number {
  if (halfLifeBlocks <= 0n) return 0;
  if (halfLifeBlocks > MAX_SAFE_BLOCKS) {
    // For extremely large half-lives, use safe ceiling — releases infinitesimally slowly
    return Math.LN2 / Number.MAX_SAFE_INTEGER;
  }
  return Math.LN2 / Number(halfLifeBlocks);
}

/**
 * Calculate raw release ratio at time t
 *
 * ratio = 1 - e^(-λt)
 *
 * At t = half_life: ratio = 0.5 (50% released)
 * At t = 2×half_life: ratio = 0.75 (75% released)
 * At t = ∞: ratio → 1.0
 *
 * H1 FIX: Overflow guard for large elapsed block values.
 */
export function calculateRawReleaseRatio(
  decayConstant: number,
  elapsedBlocks: bigint,
): number {
  if (elapsedBlocks <= 0n) return 0;
  if (elapsedBlocks > MAX_SAFE_BLOCKS) return 1.0;
  const t = Number(elapsedBlocks);
  const exponent = -decayConstant * t;
  // Guard: when exponent is very negative, e^x ≈ 0, ratio ≈ 1.0
  if (exponent < -50) return 1.0;
  return 1 - Math.exp(exponent);
}

/**
 * Calculate effective half-life after applying acceleration bonus
 *
 * A 10% acceleration means the half-life is reduced by 10%
 * effective_half_life = half_life / (1 + acceleration_bonus)
 *
 * H1 FIX: Pure bigint basis-point arithmetic instead of Number(bigint) * float.
 */
export function calculateEffectiveHalfLife(
  baseHalfLife: bigint,
  accelerationBonus: number,
): bigint {
  if (accelerationBonus <= 0) {
    return baseHalfLife;
  }

  // Convert to basis points: divisor = (1 + bonus) × 10000
  const divisorBps = 10000n + BigInt(Math.floor(accelerationBonus * 10000));
  const result = (baseHalfLife * 10000n) / divisorBps;
  // Minimum 1 block half-life
  return result > 0n ? result : 1n;
}

// ============ Main Release Calculation ============

/**
 * Calculate current release state for a vesting schedule
 *
 * Takes into account:
 * - Cliff period
 * - Acceleration bonuses
 * - Deceleration factors
 * - Paused time
 * - Max release ceiling
 */
export function calculateRelease(
  vesting: RewardVesting,
  currentBlock: bigint,
): ReleaseCalculation {
  const config = getVestingConfig(vesting.category);

  // Check if cliff has ended
  if (currentBlock < vesting.cliffEndsAtBlock) {
    return {
      totalReleased: 0n,
      newlyReleasable: 0n,
      releaseRatio: 0,
      effectiveHalfLife: config.halfLifeBlocks,
      cliffEnded: false,
      maxReached: false,
      blocksUntilNextMilestone: vesting.cliffEndsAtBlock - currentBlock,
    };
  }

  // If paused, return current state
  if (vesting.isPaused) {
    return {
      totalReleased: vesting.releasedAmount,
      newlyReleasable: 0n,
      releaseRatio:
        vesting.totalAmount - vesting.reserveAmount > 0n
          ? Number(
              (vesting.releasedAmount * 1000000n) /
                (vesting.totalAmount - vesting.reserveAmount),
            ) / 1000000
          : 0,
      effectiveHalfLife: config.halfLifeBlocks,
      cliffEnded: true,
      maxReached: false,
      blocksUntilNextMilestone: 0n, // Unknown while paused
    };
  }

  // Calculate effective time elapsed (excluding paused periods)
  const totalElapsed = currentBlock - vesting.cliffEndsAtBlock;
  const effectiveElapsed = totalElapsed - vesting.totalPausedBlocks;

  // Apply acceleration to get effective half-life
  const effectiveHalfLife = calculateEffectiveHalfLife(
    config.halfLifeBlocks,
    vesting.accelerationBonus,
  );

  // H1 FIX: Apply deceleration using bigint basis-point arithmetic
  const decelerationBps = BigInt(
    Math.floor(vesting.decelerationFactor * 10000),
  );
  const deceleratedTime = (effectiveElapsed * decelerationBps) / 10000n;

  // Calculate decay constant and raw ratio
  const lambda = calculateDecayConstant(effectiveHalfLife);
  const rawRatio = calculateRawReleaseRatio(lambda, deceleratedTime);

  // Cap at max release rate
  const cappedRatio = Math.min(rawRatio, config.maxReleaseRate);

  // H1 FIX: Calculate amounts using 1,000,000 basis points (0.0001% precision)
  const releasablePool = vesting.totalAmount - vesting.reserveAmount;
  const releaseBps = BigInt(Math.floor(cappedRatio * 1000000));
  const totalReleased = (releasablePool * releaseBps) / 1000000n;
  const newlyReleasable =
    totalReleased > vesting.releasedAmount
      ? totalReleased - vesting.releasedAmount
      : 0n;

  // Check if max reached
  const maxReached = cappedRatio >= config.maxReleaseRate - 0.001; // Small tolerance

  // Calculate blocks until next milestone (50%, 75%, 90%, max)
  const milestones = [0.5, 0.75, 0.9, config.maxReleaseRate];
  let blocksUntilNextMilestone = 0n;

  for (const milestone of milestones) {
    if (cappedRatio < milestone) {
      // t = -ln(1 - ratio) / λ
      const targetTime = -Math.log(1 - milestone) / lambda;
      const targetBlocks = BigInt(Math.ceil(targetTime));
      blocksUntilNextMilestone = targetBlocks - deceleratedTime;
      if (blocksUntilNextMilestone < 0n) blocksUntilNextMilestone = 0n;
      break;
    }
  }

  return {
    totalReleased,
    newlyReleasable,
    releaseRatio: cappedRatio,
    effectiveHalfLife,
    cliffEnded: true,
    maxReached,
    blocksUntilNextMilestone,
  };
}

// ============ Projection Functions ============

/**
 * Project release amount at a future block
 */
export function projectReleaseAtBlock(
  vesting: RewardVesting,
  targetBlock: bigint,
): bigint {
  // Create a temporary vesting state at target block
  // (This is a projection, not actual state)
  const calculation = calculateRelease(vesting, targetBlock);
  return calculation.totalReleased;
}

/**
 * Calculate blocks until a specific release percentage is reached
 */
export function blocksUntilRelease(
  vesting: RewardVesting,
  targetRatio: number,
  currentBlock: bigint,
): bigint | null {
  const config = getVestingConfig(vesting.category);

  // Can't exceed max release
  if (targetRatio > config.maxReleaseRate) {
    return null;
  }

  // If cliff hasn't ended, add cliff time
  let blocksFromNow = 0n;
  if (currentBlock < vesting.cliffEndsAtBlock) {
    blocksFromNow = vesting.cliffEndsAtBlock - currentBlock;
  }

  // Calculate effective half-life
  const effectiveHalfLife = calculateEffectiveHalfLife(
    config.halfLifeBlocks,
    vesting.accelerationBonus,
  );

  // t = -ln(1 - ratio) / λ
  const lambda = calculateDecayConstant(effectiveHalfLife);
  const targetTime = -Math.log(1 - targetRatio) / lambda;

  // Adjust for deceleration
  const adjustedTime = targetTime / vesting.decelerationFactor;

  // Account for already elapsed time
  const currentCalculation = calculateRelease(vesting, currentBlock);
  if (currentCalculation.releaseRatio >= targetRatio) {
    return 0n; // Already reached
  }

  const currentElapsed =
    currentBlock - vesting.cliffEndsAtBlock - vesting.totalPausedBlocks;
  const remainingTime = BigInt(Math.ceil(adjustedTime)) - currentElapsed;

  return blocksFromNow + (remainingTime > 0n ? remainingTime : 0n);
}

// ============ Milestone Calculations ============

/**
 * Get all release milestones for a vesting schedule
 */
export function getReleaseMilestones(
  vesting: RewardVesting,
): Array<{ ratio: number; block: bigint; amount: bigint }> {
  const config = getVestingConfig(vesting.category);
  const milestones: Array<{ ratio: number; block: bigint; amount: bigint }> =
    [];

  const ratios = [0.25, 0.5, 0.75, 0.9, config.maxReleaseRate];
  const effectiveHalfLife = calculateEffectiveHalfLife(
    config.halfLifeBlocks,
    vesting.accelerationBonus,
  );
  const lambda = calculateDecayConstant(effectiveHalfLife);
  const releasablePool = vesting.totalAmount - vesting.reserveAmount;

  for (const ratio of ratios) {
    if (ratio > config.maxReleaseRate) continue;

    // t = -ln(1 - ratio) / λ
    const timeBlocks = -Math.log(1 - ratio) / lambda;
    const adjustedTime = BigInt(
      Math.ceil(timeBlocks / vesting.decelerationFactor),
    );
    const block =
      vesting.cliffEndsAtBlock + adjustedTime + vesting.totalPausedBlocks;
    const amount =
      (releasablePool * BigInt(Math.floor(ratio * 1e18))) / BigInt(1e18);

    milestones.push({ ratio, block, amount });
  }

  return milestones;
}

// ============ Comparison Functions ============

/**
 * Compare release rates between two categories
 */
export function compareCategories(
  category1: RewardVesting["category"],
  category2: RewardVesting["category"],
  blocksElapsed: bigint,
): {
  category1Ratio: number;
  category2Ratio: number;
  difference: number;
} {
  const config1 = getVestingConfig(category1);
  const config2 = getVestingConfig(category2);

  const lambda1 = calculateDecayConstant(config1.halfLifeBlocks);
  const lambda2 = calculateDecayConstant(config2.halfLifeBlocks);

  const raw1 = calculateRawReleaseRatio(lambda1, blocksElapsed);
  const raw2 = calculateRawReleaseRatio(lambda2, blocksElapsed);

  const ratio1 = Math.min(raw1, config1.maxReleaseRate);
  const ratio2 = Math.min(raw2, config2.maxReleaseRate);

  return {
    category1Ratio: ratio1,
    category2Ratio: ratio2,
    difference: ratio1 - ratio2,
  };
}

// ============ Visualization Helpers ============

/**
 * Generate release curve data points for visualization
 */
export function generateReleaseCurve(
  vesting: RewardVesting,
  startBlock: bigint,
  endBlock: bigint,
  numPoints: number = 100,
): Array<{ block: bigint; ratio: number; amount: bigint }> {
  const points: Array<{ block: bigint; ratio: number; amount: bigint }> = [];
  const step = (endBlock - startBlock) / BigInt(numPoints);

  for (let i = 0; i <= numPoints; i++) {
    const block = startBlock + step * BigInt(i);
    const calc = calculateRelease(vesting, block);
    points.push({
      block,
      ratio: calc.releaseRatio,
      amount: calc.totalReleased,
    });
  }

  return points;
}

/**
 * Format release ratio as percentage string
 */
export function formatReleasePercentage(ratio: number): string {
  return `${(ratio * 100).toFixed(2)}%`;
}

/**
 * Format blocks as approximate time
 * Assumes 2.521 seconds per block
 */
export function formatBlocksAsTime(blocks: bigint): string {
  const seconds = Number(blocks) * 2.521;
  const minutes = seconds / 60;
  const hours = minutes / 60;
  const days = hours / 24;
  const weeks = days / 7;

  if (weeks >= 1) {
    return `~${weeks.toFixed(1)} weeks`;
  } else if (days >= 1) {
    return `~${days.toFixed(1)} days`;
  } else if (hours >= 1) {
    return `~${hours.toFixed(1)} hours`;
  } else {
    return `~${minutes.toFixed(0)} minutes`;
  }
}
