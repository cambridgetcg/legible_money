/**
 * Clawback Logic for Falsified Claims
 *
 * When a claim is proven false, this module handles:
 * - Calculation of amounts to redistribute
 * - Clawback of partially released rewards
 * - Transfer of unvested rewards to challenger
 * - Release of reserve to challenger
 *
 * PP HIERARCHY:
 *   TRUTH: False claims must lose their rewards
 *   UNDERSTANDING: Clear accounting of where rewards flow
 *   BEAUTY: Simple, deterministic redistribution
 *   JUSTICE: Late truth-finders still get compensated from reserves
 *
 * CORE PRINCIPLE: "Intention does not matter, result does."
 * Whether the original submitter lied or was negligently wrong,
 * the penalty is the same. The knowledge base was polluted.
 */

import {
  type RewardVesting,
  type ClawbackResult,
  CLAWBACK_RATES,
  VestingError,
} from "./types.js";
import { calculateRelease } from "./release_curve.js";

// ============ Clawback Calculation ============

/**
 * Calculate clawback amounts when a claim is falsified
 *
 * Flow of funds:
 * 1. All unvested rewards → Challenger
 * 2. 30% of already-released rewards → Clawed back from recipient → Challenger
 * 3. Permanent reserve → Challenger
 *
 * The original recipient keeps 70% of what was already released and withdrawn.
 */
export function calculateClawback(
  vesting: RewardVesting,
  currentBlock: bigint,
): ClawbackResult {
  // Get current release state
  const release = calculateRelease(vesting, currentBlock);

  // Calculate unvested amount
  // Unvested = Total - Reserve - Released
  const unvested =
    vesting.totalAmount - vesting.reserveAmount - release.totalReleased;

  // Calculate clawback from released (30% of released)
  // GAP-020 FIX: Use 10000 basis points for better precision (100 caused 1 wei rounding errors)
  const clawbackFromReleased =
    (release.totalReleased *
      BigInt(Math.floor(CLAWBACK_RATES.releasedClawback * 10000))) /
    10000n;

  // Reserve goes to challenger
  const fromReserve = vesting.reserveAmount;

  // Total to challenger
  const challengerReward = unvested + clawbackFromReleased + fromReserve;

  // Recipient penalty = clawback amount
  const recipientPenalty = clawbackFromReleased;

  // Recipient keeps = released - clawback
  const recipientKeeps = release.totalReleased - clawbackFromReleased;

  return {
    challengerReward,
    breakdown: {
      fromUnvested: unvested,
      fromClawback: clawbackFromReleased,
      fromReserve,
    },
    recipientPenalty,
    recipientKeeps,
    protocolRecovery: 0n, // Everything goes to truth-finder
  };
}

/**
 * Execute falsification on a vesting schedule
 *
 * Returns updated vesting marked as falsified
 */
export function falsifyVesting(
  vesting: RewardVesting,
  _challengerDid: string,
  _disputeId: string,
  currentBlock: bigint,
): {
  updatedVesting: RewardVesting;
  clawback: ClawbackResult;
} {
  // Validate vesting can be falsified
  if (vesting.status === "falsified") {
    throw new VestingError("Vesting already falsified", "VESTING_FALSIFIED");
  }

  if (vesting.status === "completed") {
    // Can still falsify completed vesting - clawback from already released
    // But there's no unvested to redistribute
  }

  // Calculate clawback
  const clawback = calculateClawback(vesting, currentBlock);

  // Update vesting to falsified status
  const updatedVesting: RewardVesting = {
    ...vesting,
    status: "falsified",
    statusChangedAtBlock: currentBlock,
    // Zero out remaining amounts
    claimableAmount: 0n,
    reserveAmount: 0n,
    // Keep released amount for record (even though some is clawed back)
    updatedAt: currentBlock,
  };

  return {
    updatedVesting,
    clawback,
  };
}

// ============ Partial Clawback (Related Falsification) ============

/**
 * Calculate partial clawback when a related/dependent claim is falsified
 *
 * Used when:
 * - A fact this claim depends on is falsified
 * - Evidence supporting this claim is invalidated
 *
 * Partial clawback is less severe than full falsification
 */
export function calculatePartialClawback(
  vesting: RewardVesting,
  currentBlock: bigint,
  impactRatio: number, // 0-1, how much this claim was affected
): ClawbackResult {
  // Calculate full clawback
  const fullClawback = calculateClawback(vesting, currentBlock);

  // M4 FIX: Scale by impact ratio using 10000 basis points (was 100, losing precision)
  const impactBps = BigInt(Math.floor(impactRatio * 10000));
  const scaledUnvested =
    (fullClawback.breakdown.fromUnvested * impactBps) / 10000n;
  const scaledClawback =
    (fullClawback.breakdown.fromClawback * impactBps) / 10000n;
  const scaledReserve =
    (fullClawback.breakdown.fromReserve * impactBps) / 10000n;

  const challengerReward = scaledUnvested + scaledClawback + scaledReserve;

  return {
    challengerReward,
    breakdown: {
      fromUnvested: scaledUnvested,
      fromClawback: scaledClawback,
      fromReserve: scaledReserve,
    },
    recipientPenalty: scaledClawback,
    recipientKeeps:
      fullClawback.recipientKeeps +
      (fullClawback.recipientPenalty - scaledClawback),
    protocolRecovery: 0n,
  };
}

// ============ Reserve Claim (Time-Based) ============

/**
 * After a very long time with no challenges, allow reserve release
 *
 * The reserve is meant for late challengers, but if no one challenges
 * for a very long time (e.g., 10x half-life), the original submitter
 * can claim a portion of the reserve.
 *
 * This is optional and configurable per protocol parameters.
 */
export function calculateReserveClaim(
  vesting: RewardVesting,
  currentBlock: bigint,
  reserveReleaseThreshold: bigint, // Blocks after which reserve partially releases
): {
  claimableReserve: bigint;
  remainingReserve: bigint;
  canClaim: boolean;
} {
  // Only vesting status can claim reserve
  if (vesting.status !== "vesting" && vesting.status !== "completed") {
    return {
      claimableReserve: 0n,
      remainingReserve: vesting.reserveAmount,
      canClaim: false,
    };
  }

  // Check if threshold is reached
  const blocksElapsed = currentBlock - vesting.acceptedAtBlock;
  if (blocksElapsed < reserveReleaseThreshold) {
    return {
      claimableReserve: 0n,
      remainingReserve: vesting.reserveAmount,
      canClaim: false,
    };
  }

  // Calculate claimable portion (up to 50% of reserve after threshold)
  const maxReserveClaim = vesting.reserveAmount / 2n;

  // Linear release of reserve after threshold
  const blocksAfterThreshold = blocksElapsed - reserveReleaseThreshold;
  const releaseRatio = Math.min(
    1.0,
    Number(blocksAfterThreshold) / Number(reserveReleaseThreshold),
  );

  const claimableReserve =
    (maxReserveClaim * BigInt(Math.floor(releaseRatio * 10000))) / 10000n;
  const remainingReserve = vesting.reserveAmount - claimableReserve;

  return {
    claimableReserve,
    remainingReserve,
    canClaim: claimableReserve > 0n,
  };
}

// ============ Accounting Helpers ============

/**
 * Verify clawback math is correct
 */
export function verifyClawback(
  vesting: RewardVesting,
  clawback: ClawbackResult,
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  // Total should equal original total
  const total = clawback.challengerReward + clawback.recipientKeeps;
  if (total > vesting.totalAmount) {
    errors.push(`Total exceeds original: ${total} > ${vesting.totalAmount}`);
  }

  // Breakdown should sum to challenger reward
  const breakdownSum =
    clawback.breakdown.fromUnvested +
    clawback.breakdown.fromClawback +
    clawback.breakdown.fromReserve;
  if (breakdownSum !== clawback.challengerReward) {
    errors.push(
      `Breakdown doesn't sum: ${breakdownSum} !== ${clawback.challengerReward}`,
    );
  }

  // Penalty should match clawback from released
  if (clawback.recipientPenalty !== clawback.breakdown.fromClawback) {
    errors.push(
      `Penalty mismatch: ${clawback.recipientPenalty} !== ${clawback.breakdown.fromClawback}`,
    );
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

/**
 * Generate human-readable clawback summary
 */
export function generateClawbackSummary(
  vesting: RewardVesting,
  clawback: ClawbackResult,
): string {
  const lines: string[] = [
    `Clawback Summary for Vesting ${vesting.id}`,
    `=========================================`,
    ``,
    `Original Total: ${formatAmount(vesting.totalAmount)}`,
    ``,
    `Challenger Receives: ${formatAmount(clawback.challengerReward)}`,
    `  - From unvested: ${formatAmount(clawback.breakdown.fromUnvested)}`,
    `  - From clawback (30% of released): ${formatAmount(clawback.breakdown.fromClawback)}`,
    `  - From reserve: ${formatAmount(clawback.breakdown.fromReserve)}`,
    ``,
    `Original Recipient:`,
    `  - Keeps: ${formatAmount(clawback.recipientKeeps)}`,
    `  - Loses (clawback): ${formatAmount(clawback.recipientPenalty)}`,
  ];

  return lines.join("\n");
}

/**
 * Format amount for display (assumes 18 decimals)
 */
function formatAmount(amount: bigint): string {
  const whole = amount / 10n ** 18n;
  const fraction = amount % 10n ** 18n;
  const fractionStr = fraction.toString().padStart(18, "0").slice(0, 4);
  return `${whole}.${fractionStr} LGM`;
}

// ============ Edge Cases ============

/**
 * Handle clawback when nothing has been released yet
 */
export function handleEarlyFalsification(
  vesting: RewardVesting,
  _currentBlock: bigint,
): ClawbackResult {
  // During cliff or very early, nothing released
  // All goes to challenger (or back to protocol if no challenger)

  return {
    challengerReward: vesting.totalAmount,
    breakdown: {
      fromUnvested: vesting.totalAmount - vesting.reserveAmount,
      fromClawback: 0n,
      fromReserve: vesting.reserveAmount,
    },
    recipientPenalty: 0n,
    recipientKeeps: 0n,
    protocolRecovery: 0n,
  };
}

/**
 * Handle clawback when vesting is almost complete
 */
export function handleLateFalsification(
  vesting: RewardVesting,
  currentBlock: bigint,
): ClawbackResult {
  // Most has been released, but reserve is still available
  const release = calculateRelease(vesting, currentBlock);

  // Very little unvested
  const unvested =
    vesting.totalAmount - vesting.reserveAmount - release.totalReleased;

  // L1 FIX: Standard clawback from released using 10000 basis points (was 100, inconsistent with main calculateClawback)
  const clawbackFromReleased =
    (release.totalReleased *
      BigInt(Math.floor(CLAWBACK_RATES.releasedClawback * 10000))) /
    10000n;

  // Reserve is the main value for late challengers
  const fromReserve = vesting.reserveAmount;

  return {
    challengerReward: unvested + clawbackFromReleased + fromReserve,
    breakdown: {
      fromUnvested: unvested,
      fromClawback: clawbackFromReleased,
      fromReserve,
    },
    recipientPenalty: clawbackFromReleased,
    recipientKeeps: release.totalReleased - clawbackFromReleased,
    protocolRecovery: 0n,
  };
}
