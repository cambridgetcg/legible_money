/**
 * Rewards Engine Contract
 *
 * Domain interface for incentive mechanics.
 * Handles vesting, clawbacks, and reward distribution.
 *
 * MOD-1: Single Responsibility
 * Rewards module changes when incentive mechanics change.
 *
 * Core Philosophy: "Intention does not matter, result does."
 * - Harsh punishment for spreading falsehood (50% slash)
 * - Generous rewards for truth discovery (60% of loser's stake)
 */

import type { DID, Amount, BlockHeight, Confidence } from "../types/index.js";

// ============ Rewards Types ============

/**
 * Vesting schedule status
 */
export type VestingStatus =
  | "active" // Normal vesting
  | "paused" // Challenge pending
  | "completed" // Fully vested
  | "falsified"; // Clawed back due to falsification

/**
 * Vesting category (different rates/cliffs)
 */
export type VestingCategory =
  | "claim_reward" // Reward for verified claim
  | "verification" // Reward for correct verification
  | "dispute_win" // Reward for winning dispute
  | "citation" // Reward for cited facts
  | "research"; // Research bounty reward

/**
 * Vesting schedule
 */
export interface VestingSchedule {
  id: string;
  claimId: string;
  recipient: DID;
  totalAmount: Amount;
  vestedAmount: Amount;
  claimedAmount: Amount;
  category: VestingCategory;
  status: VestingStatus;
  createdAtBlock: BlockHeight;
  cliffEndBlock: BlockHeight;
  fullVestBlock: BlockHeight;
  pausedAtBlock?: BlockHeight;
  pauseReason?: string;
  relatedDisputeId?: string;
  lastClaimBlock: BlockHeight;
}

/**
 * Clawback record
 */
export interface Clawback {
  id: string;
  vestingId: string;
  recipient: DID;
  clawedAmount: Amount;
  reason: "falsification" | "dispute_loss";
  disputeId: string;
  executedAtBlock: BlockHeight;
  challengerDid: DID;
  challengerReward: Amount;
}

/**
 * Reward distribution record
 */
export interface RewardDistribution {
  id: string;
  recipient: DID;
  amount: Amount;
  reason: RewardReason;
  sourceId: string; // Claim ID, block hash, etc.
  distributed: boolean;
  distributedAtBlock?: BlockHeight;
  vestingId?: string; // If vested
}

export type RewardReason =
  | "block_production"
  | "verification_correct"
  | "verification_participation"
  | "claim_accepted"
  | "challenge_success"
  | "dispute_win"
  | "citation_reward"
  | "research_bounty"
  | "early_detection_bonus"
  | "truth_discovery_bonus";

/**
 * Acceleration event (speeds up vesting)
 */
export interface AccelerationEvent {
  vestingId: string;
  type: AccelerationType;
  multiplier: number;
  relatedEntityId?: string; // Citing fact ID, etc.
  appliedAtBlock: BlockHeight;
}

export type AccelerationType =
  | "citation" // Fact was cited
  | "cross_domain" // Cited across domains
  | "high_fundamentality" // High fundamentality score
  | "governance"; // Governance decision

/**
 * Vesting parameters
 */
export interface VestingParams {
  cliffPeriodBlocks: number;
  vestingPeriodBlocks: number;
  minCliffConfidence: Confidence;
  accelerationRates: Record<AccelerationType, number>;
  clawbackPenaltyRate: number;
  challengerRewardRate: number;
}

// ============ Rewards Engine Interface ============

/**
 * Rewards engine - main domain interface
 */
export interface RewardsEngine {
  // === Vesting Management ===

  /**
   * Create a vesting schedule for a claim reward
   */
  createVestingSchedule(
    claimId: string,
    recipient: DID,
    amount: Amount,
    category: VestingCategory,
  ): Promise<VestingSchedule>;

  /**
   * Get vesting schedule by ID
   */
  getVestingSchedule(vestingId: string): Promise<VestingSchedule | null>;

  /**
   * Get all vesting schedules for a recipient
   */
  getVestingSchedulesForRecipient(recipient: DID): Promise<VestingSchedule[]>;

  /**
   * Get vesting schedule for a claim
   */
  getVestingScheduleForClaim(claimId: string): Promise<VestingSchedule | null>;

  /**
   * Calculate currently vested amount
   */
  calculateVestedAmount(vestingId: string): Promise<Amount>;

  /**
   * Calculate claimable amount (vested - already claimed)
   */
  calculateClaimableAmount(vestingId: string): Promise<Amount>;

  /**
   * Claim vested tokens
   */
  claimVested(
    vestingId: string,
    claimer: DID,
    amount?: Amount, // Optional partial claim
  ): Promise<Amount>;

  // === Pause/Resume ===

  /**
   * Pause vesting (when challenge initiated)
   */
  pauseVesting(
    vestingId: string,
    reason: string,
    relatedDisputeId?: string,
  ): Promise<void>;

  /**
   * Resume vesting (challenge resolved in favor)
   */
  resumeVesting(vestingId: string, applyDefenseBonus: boolean): Promise<void>;

  // === Acceleration ===

  /**
   * Apply acceleration to vesting
   */
  applyAcceleration(
    vestingId: string,
    type: AccelerationType,
    relatedEntityId?: string,
  ): Promise<AccelerationEvent>;

  /**
   * Get acceleration events for a vesting
   */
  getAccelerationEvents(vestingId: string): Promise<AccelerationEvent[]>;

  // === Clawback ===

  /**
   * Execute clawback (falsification proven)
   */
  executeClawback(
    vestingId: string,
    challengerDid: DID,
    disputeId: string,
  ): Promise<Clawback>;

  /**
   * Get clawback record
   */
  getClawback(clawbackId: string): Promise<Clawback | null>;

  /**
   * Get clawbacks for a recipient
   */
  getClawbacksForRecipient(recipient: DID): Promise<Clawback[]>;

  // === Reward Distribution ===

  /**
   * Queue reward for distribution
   */
  queueReward(
    recipient: DID,
    amount: Amount,
    reason: RewardReason,
    sourceId: string,
    vest: boolean, // Whether to vest or distribute immediately
  ): Promise<RewardDistribution>;

  /**
   * Distribute immediate rewards
   */
  distributeRewards(): Promise<RewardDistribution[]>;

  /**
   * Get pending rewards for recipient
   */
  getPendingRewards(recipient: DID): Promise<RewardDistribution[]>;

  // === Parameters ===

  /**
   * Get vesting parameters
   */
  getParams(): VestingParams;

  /**
   * Update vesting parameters (via governance)
   */
  updateParams(updates: Partial<VestingParams>): Promise<void>;
}

// ============ Rewards Reader (Read-Only) ============

/**
 * Read-only rewards interface
 */
export interface RewardsReader {
  getVestingSchedule(vestingId: string): Promise<VestingSchedule | null>;
  getVestingSchedulesForRecipient(recipient: DID): Promise<VestingSchedule[]>;
  calculateVestedAmount(vestingId: string): Promise<Amount>;
  calculateClaimableAmount(vestingId: string): Promise<Amount>;
  getClawback(clawbackId: string): Promise<Clawback | null>;
  getPendingRewards(recipient: DID): Promise<RewardDistribution[]>;
  getParams(): VestingParams;
}

// ============ Rewards Events ============

/**
 * Events emitted by rewards engine
 */
export type RewardsEventType =
  | "vesting_created"
  | "vesting_claimed"
  | "vesting_paused"
  | "vesting_resumed"
  | "vesting_accelerated"
  | "vesting_completed"
  | "clawback_executed"
  | "reward_queued"
  | "reward_distributed";
