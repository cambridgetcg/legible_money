/**
 * Gradual Reward Release System
 *
 * Implements scientifically-grounded reward vesting for truth claims:
 * - Exponential release curves based on knowledge half-life research
 * - Category-specific parameters (formal proofs vs empirical claims)
 * - Cliff periods for initial challenge windows
 * - Permanent reserves for late challengers
 * - Acceleration from successful defenses and replications
 * - Clawback mechanisms for falsified claims
 *
 * Research Foundation:
 * - Knowledge half-life studies (3.9-45 years by domain)
 * - Replication crisis data (~50% general replication rate)
 * - Citation decay patterns
 *
 * Module Structure:
 * - types.ts: Type definitions, constants, configurations
 * - release_curve.ts: Exponential release formula
 * - acceleration.ts: Speed modification from events
 * - clawback.ts: Falsification handling
 * - vesting_engine.ts: Main orchestration
 */

// ============ Types ============

export {
  // Categories
  type VestingCategory,
  type VestingStatus,

  // Configuration
  RELEASE_HALF_LIVES,
  MAX_RELEASE_RATES,
  CLIFF_PERIODS,
  ACCELERATION_BONUSES,
  MAX_ACCELERATION,
  DECELERATION_FACTORS,
  CLAWBACK_RATES,

  // Main types
  type RewardVesting,
  type VestingConfig,
  type ReleaseCalculation,
  type ClawbackResult,

  // Payloads
  type CreateVestingPayload,
  type ClaimRewardsPayload,
  type PauseVestingPayload,
  type ResumeVestingPayload,
  type AccelerateVestingPayload,
  type FalsifyVestingPayload,

  // Events
  type VestingEventType,
  type VestingEvent,

  // Errors
  type VestingErrorCode,
  VestingError,

  // Utilities
  getVestingConfig,
} from './types.js';

// ============ Release Curve ============

export {
  // Core formula
  calculateDecayConstant,
  calculateRawReleaseRatio,
  calculateEffectiveHalfLife,

  // Main calculation
  calculateRelease,

  // Projections
  projectReleaseAtBlock,
  blocksUntilRelease,

  // Milestones
  getReleaseMilestones,

  // Comparison
  compareCategories,

  // Visualization
  generateReleaseCurve,
  formatReleasePercentage,
  formatBlocksAsTime,
} from './release_curve.js';

// ============ Acceleration ============

export {
  type AccelerationType,
  type DecelerationType,

  // Application
  applyAcceleration,
  applyDeceleration,
  removeDeceleration,

  // Pause/Resume
  pauseVesting,
  resumeVesting,

  // Analysis
  calculateSpeedMultiplier,
  describeSpeed,
  canAccelerate,
  getRemainingAccelerationCapacity,
} from './acceleration.js';

// ============ Clawback ============

export {
  // Calculation
  calculateClawback,
  calculatePartialClawback,
  calculateReserveClaim,

  // Execution
  falsifyVesting,

  // Verification
  verifyClawback,
  generateClawbackSummary,

  // Edge cases
  handleEarlyFalsification,
  handleLateFalsification,
} from './clawback.js';

// ============ Vesting Engine ============

export {
  // Creation
  createVesting,

  // Release & Claiming
  updateRelease,
  claimRewards,

  // Challenge Integration
  handleChallengeStart,
  handleChallengeResolution,

  // Acceleration Events
  handleReplication,
  handleCorroboration,
  handleCitation,

  // Queries
  getVestingSummary,
  canClaim,

  // Utilities
  determineVestingCategory,
  createVestingEvent,

  // Batch Operations
  batchUpdateReleases,
  findUpcomingMilestones,
} from './vesting_engine.js';
