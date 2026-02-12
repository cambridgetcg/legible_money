/**
 * Vesting Engine
 *
 * Main orchestration for the Gradual Reward Release System.
 * Handles the complete lifecycle of reward vesting:
 * - Creation of vesting schedules
 * - Release calculation and claiming
 * - Pause/resume for challenges
 * - Acceleration from successful defenses
 * - Falsification and clawback
 *
 * PP HIERARCHY:
 *   TRUTH: Rewards flow to truth-tellers, away from false claimers
 *   UNDERSTANDING: Clear lifecycle with predictable transitions
 *   BEAUTY: Single engine manages all vesting operations
 *   JUSTICE: Every operation is deterministic and verifiable
 */

import { sha512_256Hex } from "../crypto/hashing.js";
import {
  type RewardVesting,
  type VestingCategory,
  type VestingStatus,
  type CreateVestingPayload,
  type ClaimRewardsPayload,
  type VestingEvent,
  getVestingConfig,
  VestingError,
} from "./types.js";
import {
  calculateRelease,
  getReleaseMilestones,
  formatBlocksAsTime,
} from "./release_curve.js";
import {
  applyAcceleration,
  pauseVesting,
  resumeVesting,
} from "./acceleration.js";
import {
  calculateClawback,
  falsifyVesting as doFalsifyVesting,
  verifyClawback,
} from "./clawback.js";

// ============ Vesting Creation ============

/**
 * Create a new vesting schedule for an accepted claim
 */
export function createVesting(
  payload: CreateVestingPayload,
  currentBlock: bigint,
): RewardVesting {
  const config = getVestingConfig(payload.category);

  // Generate vesting ID
  const vestingId = sha512_256Hex(
    `vesting:${payload.claimId}:${payload.recipient}:${currentBlock}`,
  );

  // Calculate reserve amount
  // GAP-012 FIX: Use 10000 basis points for better precision (100 was losing precision for small amounts)
  const reserveAmount =
    (payload.amount * BigInt(Math.floor(config.reserveRate * 10000))) / 10000n;

  // Create vesting record
  const vesting: RewardVesting = {
    id: vestingId,
    claimId: payload.claimId,
    recipient: payload.recipient,
    totalAmount: payload.amount,
    releasedAmount: 0n,
    claimableAmount: 0n,
    reserveAmount,
    category: payload.category,
    acceptedAtBlock: currentBlock,
    cliffEndsAtBlock: currentBlock + config.cliffBlocks,
    accelerationBonus: 0,
    accelerationBreakdown: {
      fromDefenses: 0,
      fromReplications: 0,
      fromCorroboration: 0,
      fromCitations: 0,
    },
    decelerationFactor: 1.0,
    activeDecelerations: {
      challenge: 0,
      related_dispute: 0,
      domain_audit: 0,
      source_invalidated: 0,
    },
    isPaused: false,
    totalPausedBlocks: 0n,
    defenseCount: 0,
    failedChallengeCount: 0,
    replicationCount: 0,
    status: "vesting",
    statusChangedAtBlock: currentBlock,
    createdAt: currentBlock,
    updatedAt: currentBlock,
  };

  return vesting;
}

// ============ Release & Claiming ============

/**
 * Update vesting state and calculate claimable amount
 */
export function updateRelease(
  vesting: RewardVesting,
  currentBlock: bigint,
): RewardVesting {
  // Can't update if not vesting
  if (vesting.status !== "vesting") {
    return vesting;
  }

  // Calculate current release
  const release = calculateRelease(vesting, currentBlock);

  // Update claimable amount (newly released - already claimed)
  const newlyClaimable = release.totalReleased - vesting.releasedAmount;

  // Check if max release reached AND all rewards claimed
  // GAP-013 FIX: Original logic `maxReached && newlyClaimable === 0n` was contradictory
  // because when maxReached there's typically newly claimable rewards.
  // Fixed: Complete when max reached AND total claimable (old + new) is zero
  let status: VestingStatus = vesting.status;
  const totalClaimable = vesting.claimableAmount + newlyClaimable;
  if (release.maxReached && totalClaimable === 0n) {
    status = "completed";
  }

  return {
    ...vesting,
    claimableAmount: vesting.claimableAmount + newlyClaimable,
    releasedAmount: release.totalReleased,
    status,
    statusChangedAtBlock:
      status !== vesting.status ? currentBlock : vesting.statusChangedAtBlock,
    updatedAt: currentBlock,
  };
}

/**
 * Claim released rewards
 */
export function claimRewards(
  vesting: RewardVesting,
  payload: ClaimRewardsPayload,
  currentBlock: bigint,
): { updatedVesting: RewardVesting; claimedAmount: bigint } {
  // Validate claimer
  if (payload.claimer !== vesting.recipient) {
    throw new VestingError("Only recipient can claim rewards", "NOT_RECIPIENT");
  }

  // Update release first
  const updated = updateRelease(vesting, currentBlock);

  // Check if there's anything to claim
  if (updated.claimableAmount === 0n) {
    throw new VestingError("No rewards available to claim", "NOTHING_TO_CLAIM");
  }

  // Determine claim amount
  const claimAmount =
    payload.amount && payload.amount > 0n
      ? payload.amount > updated.claimableAmount
        ? updated.claimableAmount
        : payload.amount
      : updated.claimableAmount;

  // Update vesting
  const finalVesting: RewardVesting = {
    ...updated,
    claimableAmount: updated.claimableAmount - claimAmount,
    updatedAt: currentBlock,
  };

  return {
    updatedVesting: finalVesting,
    claimedAmount: claimAmount,
  };
}

// ============ Challenge Integration ============

/**
 * Pause vesting when a challenge is initiated
 */
export function handleChallengeStart(
  vesting: RewardVesting,
  _challengeId: string,
  currentBlock: bigint,
): RewardVesting {
  // Update release state first
  const updated = updateRelease(vesting, currentBlock);

  // Pause the vesting
  return pauseVesting(updated, currentBlock);
}

/**
 * Handle challenge resolution
 */
export function handleChallengeResolution(
  vesting: RewardVesting,
  challengerWon: boolean,
  challengerDid: string,
  disputeId: string,
  currentBlock: bigint,
): {
  updatedVesting: RewardVesting;
  clawback?: ReturnType<typeof calculateClawback>;
} {
  if (challengerWon) {
    // Falsification - calculate and execute clawback
    const result = doFalsifyVesting(
      vesting,
      challengerDid,
      disputeId,
      currentBlock,
    );

    // Verify clawback math
    const verification = verifyClawback(vesting, result.clawback);
    if (!verification.valid) {
      throw new VestingError(
        `Clawback verification failed: ${verification.errors.join(", ")}`,
        "INVALID_AMOUNT",
      );
    }

    return {
      updatedVesting: result.updatedVesting,
      clawback: result.clawback,
    };
  } else {
    // Defense successful - resume with bonus
    const resumed = resumeVesting(vesting, currentBlock, true);
    return {
      updatedVesting: resumed,
    };
  }
}

// ============ Acceleration Events ============

/**
 * Handle independent replication of the claim
 */
export function handleReplication(
  vesting: RewardVesting,
  _researchId: string,
  currentBlock: bigint,
): RewardVesting {
  // Update release first
  const updated = updateRelease(vesting, currentBlock);

  // Apply replication acceleration
  return applyAcceleration(updated, "replication", currentBlock);
}

/**
 * Handle corroboration from another domain
 */
export function handleCorroboration(
  vesting: RewardVesting,
  _corroboratingFactId: string,
  currentBlock: bigint,
): RewardVesting {
  const updated = updateRelease(vesting, currentBlock);
  return applyAcceleration(updated, "corroboration", currentBlock);
}

/**
 * Handle citation by another accepted fact
 */
export function handleCitation(
  vesting: RewardVesting,
  _citingFactId: string,
  currentBlock: bigint,
): RewardVesting {
  const updated = updateRelease(vesting, currentBlock);
  return applyAcceleration(updated, "citation", currentBlock);
}

// ============ Query Functions ============

/**
 * Get vesting summary for display
 */
export function getVestingSummary(
  vesting: RewardVesting,
  currentBlock: bigint,
): {
  status: VestingStatus;
  totalAmount: bigint;
  releasedAmount: bigint;
  claimableAmount: bigint;
  lockedAmount: bigint;
  reserveAmount: bigint;
  releasePercentage: number;
  maxReleasePercentage: number;
  speedMultiplier: number;
  isPaused: boolean;
  cliffEnded: boolean;
  blocksUntilNextMilestone: bigint;
  estimatedTimeToNextMilestone: string;
  milestones: ReturnType<typeof getReleaseMilestones>;
} {
  const updated = updateRelease(vesting, currentBlock);
  const release = calculateRelease(updated, currentBlock);
  const config = getVestingConfig(vesting.category);
  const milestones = getReleaseMilestones(updated);

  const speedMultiplier =
    (1 + updated.accelerationBonus) * updated.decelerationFactor;
  const lockedAmount =
    updated.totalAmount - updated.reserveAmount - release.totalReleased;

  return {
    status: updated.status,
    totalAmount: updated.totalAmount,
    releasedAmount: release.totalReleased,
    claimableAmount:
      updated.claimableAmount +
      (release.totalReleased - updated.releasedAmount),
    lockedAmount,
    reserveAmount: updated.reserveAmount,
    releasePercentage: release.releaseRatio * 100,
    maxReleasePercentage: config.maxReleaseRate * 100,
    speedMultiplier,
    isPaused: updated.isPaused,
    cliffEnded: release.cliffEnded,
    blocksUntilNextMilestone: release.blocksUntilNextMilestone,
    estimatedTimeToNextMilestone: formatBlocksAsTime(
      release.blocksUntilNextMilestone,
    ),
    milestones,
  };
}

/**
 * Check if vesting can be claimed
 */
export function canClaim(
  vesting: RewardVesting,
  currentBlock: bigint,
): { canClaim: boolean; reason?: string; amount?: bigint } {
  if (vesting.status === "falsified") {
    return { canClaim: false, reason: "Vesting was falsified" };
  }

  if (vesting.isPaused) {
    return { canClaim: false, reason: "Vesting is paused during challenge" };
  }

  if (currentBlock < vesting.cliffEndsAtBlock) {
    const blocksRemaining = vesting.cliffEndsAtBlock - currentBlock;
    return {
      canClaim: false,
      reason: `Cliff period: ${formatBlocksAsTime(blocksRemaining)} remaining`,
    };
  }

  const release = calculateRelease(vesting, currentBlock);
  const claimable =
    release.totalReleased - vesting.releasedAmount + vesting.claimableAmount;

  if (claimable === 0n) {
    return { canClaim: false, reason: "No rewards available yet" };
  }

  return { canClaim: true, amount: claimable };
}

// ============ Category Utilities ============

/**
 * Determine appropriate category for a claim based on its properties
 */
export function determineVestingCategory(
  _claimType: string,
  evidenceTypes: string[],
  isReplicated: boolean,
  isContested: boolean,
): VestingCategory {
  if (isContested) {
    return "contested";
  }

  if (isReplicated) {
    return "replicated";
  }

  // Check evidence types
  if (evidenceTypes.includes("formal_proof")) {
    return "formal_proof";
  }

  if (evidenceTypes.includes("on_chain")) {
    return "on_chain";
  }

  if (evidenceTypes.includes("cryptographic_proof")) {
    return "cryptographic";
  }

  if (
    evidenceTypes.includes("chainlink_price_feed") ||
    evidenceTypes.includes("chainlink_por")
  ) {
    return "oracle_feed";
  }

  if (evidenceTypes.includes("peer_reviewed")) {
    return "peer_reviewed";
  }

  if (evidenceTypes.includes("attestation")) {
    return "attestation";
  }

  if (evidenceTypes.includes("computational")) {
    return "computational";
  }

  // Default to peer_reviewed (middle of the road)
  return "peer_reviewed";
}

// ============ Event Generation ============

/**
 * Generate vesting event
 */
export function createVestingEvent(
  type: VestingEvent["type"],
  vesting: RewardVesting,
  currentBlock: bigint,
  data: Record<string, unknown> = {},
): VestingEvent {
  return {
    type,
    vestingId: vesting.id,
    blockNumber: currentBlock,
    timestamp: currentBlock,
    data: {
      claimId: vesting.claimId,
      recipient: vesting.recipient,
      status: vesting.status,
      ...data,
    },
  };
}

// ============ Batch Operations ============

/**
 * Process multiple vestings that need release updates
 */
export function batchUpdateReleases(
  vestings: RewardVesting[],
  currentBlock: bigint,
): RewardVesting[] {
  return vestings
    .filter((v) => v.status === "vesting" && !v.isPaused)
    .map((v) => updateRelease(v, currentBlock));
}

/**
 * Find vestings approaching milestones
 */
export function findUpcomingMilestones(
  vestings: RewardVesting[],
  currentBlock: bigint,
  withinBlocks: bigint,
): Array<{
  vesting: RewardVesting;
  milestone: number;
  blocksUntil: bigint;
}> {
  const results: Array<{
    vesting: RewardVesting;
    milestone: number;
    blocksUntil: bigint;
  }> = [];

  for (const vesting of vestings) {
    if (vesting.status !== "vesting" || vesting.isPaused) {
      continue;
    }

    const milestones = getReleaseMilestones(vesting);
    const release = calculateRelease(vesting, currentBlock);

    for (const milestone of milestones) {
      if (milestone.ratio > release.releaseRatio) {
        const blocksUntil = milestone.block - currentBlock;
        if (blocksUntil <= withinBlocks && blocksUntil > 0n) {
          results.push({
            vesting,
            milestone: milestone.ratio,
            blocksUntil,
          });
        }
        break; // Only get next milestone
      }
    }
  }

  return results.sort((a, b) => Number(a.blocksUntil - b.blocksUntil));
}
