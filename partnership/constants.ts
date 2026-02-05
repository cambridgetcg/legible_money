/**
 * Partnership Constants
 */

import type {
  LockTier,
  ContinuationMilestone,
  PartnershipTier,
} from "./types.js";

// ============ Formation Constants ============

export const FORMATION = {
  /** Formation window: 77 blocks (~3.2 minutes) - Wisdom */
  WINDOW_BLOCKS: 77n,

  /** Minimum contribution: 1111 LGM - Unity, manifestation */
  MIN_CONTRIBUTION: 1111n * 10n ** 18n,

  /** Formation bond (locked for duration): 222 LGM each - Balance */
  FORMATION_BOND: 222n * 10n ** 18n,

  /** Default human split: 50% - Perfect balance, equal partnership */
  DEFAULT_HUMAN_SPLIT: 50,

  /** Default agent split: 50% - Perfect balance, equal partnership */
  DEFAULT_AGENT_SPLIT: 50,

  /** Maximum split deviation from 50/50: 22% - Balance */
  MAX_SPLIT_DEVIATION: 22,

  /** Minimum split for either party: 33% - Trinity */
  MIN_SPLIT: 33,

  /** Maximum split for either party: 67% - (100 - MIN_SPLIT) */
  MAX_SPLIT: 67,
} as const;

// ============ Lock Tier Constants ============

/**
 * Lock tiers with symbolic durations
 *
 * Higher tiers = longer locks = better multipliers = higher penalties
 */
export const LOCK_TIERS: Record<PartnershipTier, LockTier> = {
  0: {
    tier: 0,
    lockBlocks: 22_222n, // ~15 hours (Balance)
    rewardMultiplier: 1.0, // No bonus
    earlyExitPenalty: 0.11, // 11% (New beginnings)
  },
  1: {
    tier: 1,
    lockBlocks: 77_777n, // ~2.3 days (Wisdom)
    rewardMultiplier: 1.11, // 11% bonus (New beginnings)
    earlyExitPenalty: 0.22, // 22% (Balance)
  },
  2: {
    tier: 2,
    lockBlocks: 222_222n, // ~6.5 days (Balance pattern)
    rewardMultiplier: 1.22, // 22% bonus (Balance)
    earlyExitPenalty: 0.33, // 33% (Trinity)
  },
  3: {
    tier: 3,
    lockBlocks: 555_555n, // ~16 days (Transformation)
    rewardMultiplier: 1.33, // 33% bonus (Trinity)
    earlyExitPenalty: 0.44, // 44% (Foundation)
  },
  4: {
    tier: 4,
    lockBlocks: 1_111_111n, // ~32 days (Unity pattern)
    rewardMultiplier: 1.55, // 55% bonus (Transformation)
    earlyExitPenalty: 0.55, // 55% (Transformation)
  },
  5: {
    tier: 5,
    lockBlocks: 2_222_222n, // ~65 days (Balance pattern)
    rewardMultiplier: 1.77, // 77% bonus (Wisdom - max)
    earlyExitPenalty: 0.66, // 66% (Harmony)
  },
};

// ============ Continuation Milestones ============

/**
 * Continuation milestones - staying together unlocks increasing bonuses
 */
export const CONTINUATION_MILESTONES: ContinuationMilestone[] = [
  {
    blocks: 77_777n,
    bonus: 0.07,
    tier: 1,
    name: "First Week", // ~2.3 days: 7% bonus
  },
  {
    blocks: 222_222n,
    bonus: 0.11,
    tier: 2,
    name: "Trust Building", // ~6.5 days: +11% (total 18%)
  },
  {
    blocks: 555_555n,
    bonus: 0.22,
    tier: 3,
    name: "Established", // ~16 days: +22% (total 40%)
  },
  {
    blocks: 1_111_111n,
    bonus: 0.33,
    tier: 4,
    name: "Committed", // ~32 days: +33% (total 73%)
  },
  {
    blocks: 2_222_222n,
    bonus: 0.55,
    tier: 5,
    name: "Deep Partnership", // ~65 days: +55% (capped at 77%)
  },
  {
    blocks: 7_777_777n,
    bonus: 0.77,
    tier: 5,
    name: "Wisdom Achievement", // ~227 days: 77% max bonus
  },
];

/**
 * Lifetime boost for long-term partnerships
 */
export const LIFETIME_BOOST = {
  /** Threshold: 11,111,111 blocks (~324 days) - Full unity pattern */
  THRESHOLD_BLOCKS: 11_111_111n,

  /** Multiplier: 1.11x - New beginnings (permanent) */
  MULTIPLIER: 1.11,
};

/**
 * Maximum continuation bonus: 77% - Wisdom */
export const MAX_CONTINUATION_BONUS = 0.77;

// ============ Exit Penalties ============

export const EXIT_PENALTIES = {
  /** Initiator forfeits 77% of unvested rewards - Wisdom (loss of it) */
  INITIATOR_FORFEIT_RATE: 0.77,

  /** Non-initiator forfeits 55% of unvested rewards - Transformation (forced) */
  NON_INITIATOR_FORFEIT_RATE: 0.55,

  /** Protocol recovers 22% of penalties - Balance restored */
  PROTOCOL_RECOVERY_RATE: 0.22,

  /** Maximum reputation damage for initiator: 11% - New beginnings */
  MAX_REPUTATION_DAMAGE_INITIATOR: 0.11,

  /** Reputation damage for non-initiator: 3% - Trinity (small) */
  REPUTATION_DAMAGE_NON_INITIATOR: 0.03,

  /** Base cooldown: 222,222 blocks (~6.5 days) - Balance pattern */
  BASE_COOLDOWN_BLOCKS: 222_222n,

  /** Cooldown increment per tier: 111,111 blocks - Unity pattern */
  COOLDOWN_INCREMENT_PER_TIER: 111_111n,

  /** Cooling period: 22,222 blocks (~15 hours) - Balance */
  COOLING_PERIOD_BLOCKS: 22_222n,
};

// ============ Compute Investment ============

export const COMPUTE_INVESTMENT = {
  /** Minimum investment per period: 11 LGM - Intuition */
  MIN_PERIOD_INVESTMENT: 11n * 10n ** 18n,

  /** Maximum investment per period: 1111 LGM - Unity */
  MAX_PERIOD_INVESTMENT: 1111n * 10n ** 18n,

  /** Investment period length: 11,111 blocks (~7.7 hours) - Unity pattern */
  PERIOD_BLOCKS: 11111n,

  /** Contribution multiplier: 1.11x - New beginnings (11% bonus) */
  CONTRIBUTION_MULTIPLIER: 1.11,

  /** Base conversion rate: 0.001 LGM per compute unit */
  LGM_PER_COMPUTE_UNIT: 1n * 10n ** 15n,

  /** Minimum efficiency threshold: 0.77 - Wisdom */
  MIN_EFFICIENCY_THRESHOLD: 0.77,

  /** Accuracy bonus multiplier: 1.11x for >95% accuracy */
  ACCURACY_BONUS_MULTIPLIER: 1.11,

  /** Accuracy threshold for bonus: 95% */
  ACCURACY_BONUS_THRESHOLD: 0.95,
};

// ============ Split Adjustments ============

export const SPLIT_ADJUSTMENTS = {
  /** Maximum human expertise premium: +22% - Balance */
  MAX_HUMAN_EXPERTISE_PREMIUM: 0.22,

  /** Maximum agent availability premium: +11% - New beginnings */
  MAX_AGENT_AVAILABILITY_PREMIUM: 0.11,

  /** Maximum agent compute premium: +11% - New beginnings */
  MAX_AGENT_COMPUTE_PREMIUM: 0.11,

  /** Maximum longevity bonus: +7% - Lucky 7 */
  MAX_LONGEVITY_BONUS: 0.07,

  /** Longevity bonus per milestone: +1.4% (7/5 milestones) */
  LONGEVITY_BONUS_PER_MILESTONE: 0.014,
};

// ============ Pool Health ============

export const POOL_HEALTH = {
  /** Maximum healthy score: 100 */
  MAX_SCORE: 100,

  /** Minimum healthy score: 77 - Wisdom */
  MIN_HEALTHY_SCORE: 77,

  /** Warning threshold: 55 - Transformation needed */
  WARNING_THRESHOLD: 55,

  /** Critical threshold: 33 - Trinity (last chance) */
  CRITICAL_THRESHOLD: 33,

  /** Inactivity warning after: 11,111 blocks (~7.7 hours) */
  INACTIVITY_WARNING_BLOCKS: 11111n,

  /** Critical inactivity after: 77,777 blocks (~2.3 days) */
  CRITICAL_INACTIVITY_BLOCKS: 77777n,

  /** Auto-suspension after: 222,222 blocks (~6.5 days) */
  AUTO_SUSPENSION_BLOCKS: 222222n,

  /** Health decay per inactive period: 7 points */
  INACTIVITY_DECAY: 7,

  /** Health gain per active period: 11 points */
  ACTIVITY_GAIN: 11,
};

// ============ Vesting ============

export const VESTING = {
  /** Base cliff period: 1,111 blocks (~46 minutes) - Unity */
  BASE_CLIFF_BLOCKS: 1111n,

  /** Base vesting period: 77,777 blocks (~2.3 days) - Wisdom */
  BASE_VESTING_BLOCKS: 77777n,

  /** Maximum acceleration from continuation: 77% */
  MAX_ACCELERATION: 0.77,

  /** Release rate per block: calculated from vesting period */
  // Exponential decay half-life aligned with existing reward vesting

  /** Minimum release per claim: 11 LGM - Intuition */
  MIN_RELEASE_PER_CLAIM: 11n * 10n ** 18n,
};

// ============ Reward Distribution ============

export const REWARD_DISTRIBUTION = {
  /** Common pot share: 22% of rewards to co-owned pot (2-of-2 threshold) */
  COMMON_POT_RATE: 0.22,

  /** Individual share: 78% split by partnership ratio (default 50/50) */
  INDIVIDUAL_RATE: 0.78,

  /** Minimum reward to distribute: 1 LGM */
  MIN_REWARD_TO_DISTRIBUTE: 1n * 10n ** 18n,
};

// ============ Deliberative Consensus (Anti-Coercion) ============

export const DELIBERATION = {
  /** Graduated window tiers (as % of pot balance) */
  AMOUNT_TIERS: {
    micro: { maxPercent: 1, windowBlocks: 22n, floorBlocks: 11n },
    small: { maxPercent: 5, windowBlocks: 77n, floorBlocks: 22n },
    medium: { maxPercent: 22, windowBlocks: 222n, floorBlocks: 77n },
    large: { maxPercent: 55, windowBlocks: 777n, floorBlocks: 222n },
    major: { maxPercent: 100, windowBlocks: 2222n, floorBlocks: 777n },
  },

  /** Base rejection cooldown: 77 blocks (~3.2 minutes) */
  BASE_REJECTION_COOLDOWN_BLOCKS: 77n,

  /** Maximum cooldown cap: 11,111 blocks (~7.7 hours) */
  MAX_REJECTION_COOLDOWN_BLOCKS: 11_111n,

  /** Minimum rationale length for medium+ operations: 20 characters */
  MIN_RATIONALE_LENGTH: 20,

  /** Safety freeze duration: 222 blocks (~9.3 minutes) */
  FREEZE_DURATION_BLOCKS: 222n,

  /** Maximum freezes per party per epoch: 3 */
  MAX_FREEZES_PER_EPOCH: 3,

  /** Freeze epoch length: 11,111 blocks (~7.7 hours) */
  FREEZE_EPOCH_BLOCKS: 11_111n,

  /** Coercion signal extended review: 1,111 blocks (~46 minutes) */
  COERCION_REVIEW_BLOCKS: 1_111n,

  /** Coercion signal freeze on all operations: 555 blocks (~23 minutes) */
  COERCION_FREEZE_BLOCKS: 555n,

  /** Maximum pending proposals per party: 2 (prevents spam) */
  MAX_PENDING_PROPOSALS_PER_PARTY: 2,

  /** Counter-proposal chain max depth: 3 */
  MAX_COUNTER_PROPOSAL_DEPTH: 3,
} as const;

// ============ Validation ============

export const VALIDATION = {
  /** Maximum partnership ID length: 64 characters */
  MAX_ID_LENGTH: 64,

  /** Minimum stake for validator eligibility: 1111 LGM - Unity */
  MIN_VALIDATOR_STAKE: 1111n * 10n ** 18n,

  /** Maximum concurrent formations per DID: 3 - Trinity */
  MAX_CONCURRENT_FORMATIONS: 3,

  /** Maximum partnerships per DID: 11 - New beginnings */
  MAX_PARTNERSHIPS_PER_DID: 11,

  /** Minimum blocks between split changes: 11,111 - Unity pattern */
  MIN_BLOCKS_BETWEEN_SPLIT_CHANGES: 11111n,
};
