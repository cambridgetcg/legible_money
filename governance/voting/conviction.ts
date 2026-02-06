/**
 * Conviction Voting
 *
 * Implements conviction voting where vote weight increases over time
 * as tokens remain locked. This rewards sustained support and
 * discourages vote switching/manipulation.
 *
 * PP HIERARCHY:
 *   TRUTH: Lock duration mathematically determines multiplier
 *   UNDERSTANDING: Clear lock tiers and multipliers
 *   BEAUTY: Continuous conviction accumulation
 *   JUSTICE: Rewards long-term commitment
 *   CREATIVITY: Time-based vote strengthening
 */

import type { DID } from "../../core/agents/identity/did.js";

// ============ Lock Duration Tiers ============

/**
 * Lock duration tier with multiplier.
 */
export interface LockTier {
  name: string;
  minBlocks: bigint;
  multiplier: number;
}

/**
 * Lock duration tiers.
 * At 2521ms per block:
 * - 50000 blocks ≈ 1.5 days
 * - 200000 blocks ≈ 6 days
 * - 600000 blocks ≈ 17 days
 * - 1000000 blocks ≈ 29 days
 */
export const LOCK_TIERS: readonly LockTier[] = [
  { name: "unlocked", minBlocks: 0n, multiplier: 1.0 },
  { name: "short", minBlocks: 50000n, multiplier: 1.25 },
  { name: "medium", minBlocks: 200000n, multiplier: 1.5 },
  { name: "long", minBlocks: 600000n, multiplier: 1.75 },
  { name: "maximum", minBlocks: 1000000n, multiplier: 2.0 },
] as const;

/**
 * Maximum conviction multiplier.
 */
export const MAX_CONVICTION_MULTIPLIER = 2.0;

// ============ Vote Switching Penalties ============

/**
 * Penalties for switching votes.
 */
export interface VoteSwitchPenalty {
  convictionReset: boolean; // Reset conviction to 0
  stakeBurnPercent: number; // % of stake burned
  cooldownBlocks: bigint; // Blocks before can vote again
}

export const VOTE_SWITCH_PENALTY: VoteSwitchPenalty = {
  convictionReset: true,
  stakeBurnPercent: 0.1, // 10%
  cooldownBlocks: 17200n, // ~12 hours
};

// ============ Conviction State ============

/**
 * Conviction state for a vote.
 */
export interface ConvictionState {
  voter: DID;
  lipId: string;
  lockedAmount: bigint;
  lockStartBlock: bigint;
  lockDuration: bigint; // Committed lock duration
  unlockBlock: bigint; // When tokens unlock
  currentTier: string;
  currentMultiplier: number;
  accumulatedConviction: number; // Grows over time
}

/**
 * Conviction vote record.
 */
export interface ConvictionVote {
  voteId: string;
  voter: DID;
  lipId: string;
  direction: "support" | "oppose" | "abstain";
  baseVotes: number;
  lockedAmount: bigint;
  lockDuration: bigint;
  createdAt: bigint;
  convictionState: ConvictionState;
}

// ============ Multiplier Calculations ============

/**
 * Get multiplier for a lock duration.
 */
export function getMultiplierForDuration(lockBlocks: bigint): number {
  let multiplier = LOCK_TIERS[0].multiplier;

  for (const tier of LOCK_TIERS) {
    if (lockBlocks >= tier.minBlocks) {
      multiplier = tier.multiplier;
    } else {
      break;
    }
  }

  return multiplier;
}

/**
 * Get tier name for a lock duration.
 */
export function getTierForDuration(lockBlocks: bigint): string {
  let tierName = LOCK_TIERS[0].name;

  for (const tier of LOCK_TIERS) {
    if (lockBlocks >= tier.minBlocks) {
      tierName = tier.name;
    } else {
      break;
    }
  }

  return tierName;
}

/**
 * Calculate continuous conviction multiplier based on elapsed time.
 * Conviction grows linearly from 1.0 to the tier maximum over the lock period.
 * Uses basis-point integer math for determinism.
 */
export function calculateConvictionMultiplier(
  lockDuration: bigint,
  elapsedBlocks: bigint,
): number {
  if (lockDuration <= 0n) {
    return 1.0; // No lock = base multiplier
  }

  // Get the target multiplier for this lock duration
  const targetMultiplier = getMultiplierForDuration(lockDuration);

  // If lock period complete, return full multiplier
  if (elapsedBlocks >= lockDuration) {
    return targetMultiplier;
  }

  // Linear interpolation using basis points for determinism
  // progressBps = (elapsedBlocks * 10000) / lockDuration
  const progressBps = Number((elapsedBlocks * 10000n) / lockDuration);
  const rangeBps = Math.round((targetMultiplier - 1.0) * 10000);
  // result = 1.0 + (range * progress) / 10000
  return 1.0 + (rangeBps * progressBps) / (10000 * 10000);
}

/**
 * Calculate accumulated conviction over time.
 * Conviction = integral of multiplier over time = area under curve.
 * Uses basis-point arithmetic for determinism.
 */
export function calculateAccumulatedConviction(
  baseVotes: number,
  lockDuration: bigint,
  elapsedBlocks: bigint,
): number {
  if (lockDuration <= 0n) {
    // No lock: linear accumulation at 1x
    return baseVotes * Number(elapsedBlocks);
  }

  const targetMultiplier = getMultiplierForDuration(lockDuration);
  // Average multiplier over full lock period = (1.0 + target) / 2
  // In basis points: ((10000 + targetBps) / 2)
  const targetBps = Math.round(targetMultiplier * 10000);
  const avgLockBps = ((10000 + targetBps) / 2) | 0;

  if (elapsedBlocks >= lockDuration) {
    // Lock complete: trapezoidal area during lock + rectangular area after
    const lockBlocks = Number(lockDuration);
    const lockConviction = (baseVotes * lockBlocks * avgLockBps) / 10000;
    const postLockBlocks = Number(elapsedBlocks - lockDuration);
    const postLockConviction = (baseVotes * postLockBlocks * targetBps) / 10000;
    return lockConviction + postLockConviction;
  }

  // During lock: triangular growth
  const currentMultiplier = calculateConvictionMultiplier(
    lockDuration,
    elapsedBlocks,
  );
  const currentBps = Math.round(currentMultiplier * 10000);
  const avgBps = ((10000 + currentBps) / 2) | 0;
  return (baseVotes * Number(elapsedBlocks) * avgBps) / 10000;
}

// ============ Conviction Vote Management ============

/**
 * Create a conviction vote.
 */
export function createConvictionVote(
  voter: DID,
  lipId: string,
  direction: "support" | "oppose" | "abstain",
  baseVotes: number,
  lockedAmount: bigint,
  lockDuration: bigint,
  currentBlock: bigint,
): ConvictionVote {
  const tier = getTierForDuration(lockDuration);
  const multiplier = getMultiplierForDuration(lockDuration);

  const convictionState: ConvictionState = {
    voter,
    lipId,
    lockedAmount,
    lockStartBlock: currentBlock,
    lockDuration,
    unlockBlock: currentBlock + lockDuration,
    currentTier: tier,
    currentMultiplier: 1.0, // Starts at 1.0, grows over time
    accumulatedConviction: 0,
  };

  return {
    voteId: `cv-${lipId}-${voter}-${currentBlock}`,
    voter,
    lipId,
    direction,
    baseVotes,
    lockedAmount,
    lockDuration,
    createdAt: currentBlock,
    convictionState,
  };
}

/**
 * Update conviction state at current block.
 */
export function updateConvictionState(
  vote: ConvictionVote,
  currentBlock: bigint,
): ConvictionVote {
  const elapsedBlocks = currentBlock - vote.createdAt;

  const newMultiplier = calculateConvictionMultiplier(
    vote.lockDuration,
    elapsedBlocks,
  );
  const newConviction = calculateAccumulatedConviction(
    vote.baseVotes,
    vote.lockDuration,
    elapsedBlocks,
  );

  return {
    ...vote,
    convictionState: {
      ...vote.convictionState,
      currentMultiplier: newMultiplier,
      accumulatedConviction: newConviction,
    },
  };
}

/**
 * Get effective vote power at current block.
 */
export function getEffectiveVotePower(
  vote: ConvictionVote,
  currentBlock: bigint,
): number {
  const elapsedBlocks = currentBlock - vote.createdAt;
  const multiplier = calculateConvictionMultiplier(
    vote.lockDuration,
    elapsedBlocks,
  );
  return vote.baseVotes * multiplier;
}

/**
 * Check if tokens can be unlocked.
 */
export function canUnlock(vote: ConvictionVote, currentBlock: bigint): boolean {
  return currentBlock >= vote.convictionState.unlockBlock;
}

/**
 * Calculate penalty for switching vote.
 */
export function calculateSwitchPenalty(
  lockedAmount: bigint,
  penalty: VoteSwitchPenalty = VOTE_SWITCH_PENALTY,
): { burnAmount: bigint; cooldownBlocks: bigint } {
  const burnAmount =
    (lockedAmount * BigInt(Math.floor(penalty.stakeBurnPercent * 10000))) /
    10000n;

  return {
    burnAmount,
    cooldownBlocks: penalty.cooldownBlocks,
  };
}

/**
 * Switch vote direction (with penalty).
 */
export function switchVote(
  vote: ConvictionVote,
  newDirection: "support" | "oppose" | "abstain",
  currentBlock: bigint,
  penalty: VoteSwitchPenalty = VOTE_SWITCH_PENALTY,
): { newVote: ConvictionVote; burnAmount: bigint; cooldownEndsAt: bigint } {
  const { burnAmount, cooldownBlocks } = calculateSwitchPenalty(
    vote.lockedAmount,
    penalty,
  );

  // Create new vote with reset conviction
  const newVote = createConvictionVote(
    vote.voter,
    vote.lipId,
    newDirection,
    vote.baseVotes,
    vote.lockedAmount - burnAmount, // Reduced by burn
    vote.lockDuration,
    currentBlock,
  );

  return {
    newVote,
    burnAmount,
    cooldownEndsAt: currentBlock + cooldownBlocks,
  };
}

// ============ Aggregation ============

/**
 * Aggregated conviction votes.
 */
export interface ConvictionVoteTally {
  lipId: string;
  support: {
    totalEffectiveVotes: number;
    totalConviction: number;
    voterCount: number;
    totalLocked: bigint;
  };
  oppose: {
    totalEffectiveVotes: number;
    totalConviction: number;
    voterCount: number;
    totalLocked: bigint;
  };
  abstain: {
    totalEffectiveVotes: number;
    totalConviction: number;
    voterCount: number;
    totalLocked: bigint;
  };
  netEffectiveVotes: number;
  netConviction: number;
}

/**
 * Aggregate conviction votes at a specific block.
 */
export function aggregateConvictionVotes(
  votes: ConvictionVote[],
  currentBlock: bigint,
): ConvictionVoteTally {
  if (votes.length === 0 || !votes[0]) {
    throw new Error("No votes to aggregate");
  }

  const lipId = votes[0].lipId;

  const support = {
    totalEffectiveVotes: 0,
    totalConviction: 0,
    voterCount: 0,
    totalLocked: 0n,
  };
  const oppose = {
    totalEffectiveVotes: 0,
    totalConviction: 0,
    voterCount: 0,
    totalLocked: 0n,
  };
  const abstain = {
    totalEffectiveVotes: 0,
    totalConviction: 0,
    voterCount: 0,
    totalLocked: 0n,
  };

  for (const vote of votes) {
    const updated = updateConvictionState(vote, currentBlock);
    const effectiveVotes = getEffectiveVotePower(vote, currentBlock);

    const target =
      vote.direction === "support"
        ? support
        : vote.direction === "oppose"
          ? oppose
          : abstain;

    target.totalEffectiveVotes += effectiveVotes;
    target.totalConviction += updated.convictionState.accumulatedConviction;
    target.voterCount++;
    target.totalLocked += vote.lockedAmount;
  }

  return {
    lipId,
    support,
    oppose,
    abstain,
    netEffectiveVotes: support.totalEffectiveVotes - oppose.totalEffectiveVotes,
    netConviction: support.totalConviction - oppose.totalConviction,
  };
}

// ============ Utilities ============

/**
 * Get lock tier table for display.
 */
export function getLockTierTable(): Array<{
  tier: string;
  minBlocks: string;
  approximateDays: string;
  multiplier: string;
}> {
  return LOCK_TIERS.map((tier) => {
    const days = (Number(tier.minBlocks) * 2521) / (1000 * 60 * 60 * 24);
    return {
      tier: tier.name,
      minBlocks: tier.minBlocks.toString(),
      approximateDays: days.toFixed(1),
      multiplier: `${tier.multiplier}x`,
    };
  });
}

/**
 * Estimate conviction growth over time.
 */
export function estimateConvictionGrowth(
  baseVotes: number,
  lockDuration: bigint,
  checkpoints: bigint[],
): Array<{ block: bigint; multiplier: number; conviction: number }> {
  return checkpoints.map((block) => ({
    block,
    multiplier: calculateConvictionMultiplier(lockDuration, block),
    conviction: calculateAccumulatedConviction(baseVotes, lockDuration, block),
  }));
}
