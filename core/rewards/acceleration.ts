/**
 * Acceleration and Deceleration Management
 *
 * Handles dynamic adjustment of release rates based on:
 * - Successful defenses against challenges (+acceleration)
 * - Independent replications (+acceleration)
 * - Cross-domain corroboration (+acceleration)
 * - Citations by other accepted facts (+acceleration)
 * - Active challenges (pause/decelerate)
 * - Related disputes (decelerate)
 *
 * PP HIERARCHY:
 *   TRUTH: Facts that survive scrutiny deserve faster reward release
 *   UNDERSTANDING: Each positive signal increases confidence
 *   BEAUTY: Simple additive bonuses with clear caps
 *   JUSTICE: Rewards flow faster to battle-tested truths
 */

import {
  type RewardVesting,
  ACCELERATION_BONUSES,
  MAX_ACCELERATION,
  DECELERATION_FACTORS,
  VestingError,
} from './types.js';

// ============ Acceleration Types ============

export type AccelerationType =
  | 'defense'
  | 'replication'
  | 'corroboration'
  | 'citation';

export type DecelerationType =
  | 'challenge'
  | 'related_dispute'
  | 'domain_audit'
  | 'source_invalidated';

// ============ Acceleration Application ============

/**
 * Apply acceleration bonus to a vesting schedule
 *
 * Returns updated vesting with new acceleration values
 */
export function applyAcceleration(
  vesting: RewardVesting,
  accelerationType: AccelerationType,
  _currentBlock: bigint
): RewardVesting {
  // Check if vesting can be accelerated
  if (vesting.status !== 'vesting') {
    throw new VestingError(
      `Cannot accelerate vesting in status: ${vesting.status}`,
      'VESTING_COMPLETED'
    );
  }

  if (vesting.isPaused) {
    throw new VestingError(
      'Cannot accelerate paused vesting',
      'VESTING_PAUSED'
    );
  }

  // Get bonus amount for this type
  const bonus = getAccelerationBonus(accelerationType);

  // Get current amount for this type
  const currentTypeAmount = getTypeAcceleration(vesting, accelerationType);

  // Check type-specific cap
  const typeCap = getTypeAccelerationCap(accelerationType);
  if (currentTypeAmount >= typeCap) {
    // Already at cap for this type, no change
    return vesting;
  }

  // Calculate new type-specific amount (capped)
  const newTypeAmount = Math.min(currentTypeAmount + bonus, typeCap);
  const actualBonus = newTypeAmount - currentTypeAmount;

  // Check total cap
  const currentTotal = vesting.accelerationBonus;
  if (currentTotal >= MAX_ACCELERATION.total) {
    // Already at total cap
    return vesting;
  }

  // Apply bonus (capped at total max)
  const newTotal = Math.min(currentTotal + actualBonus, MAX_ACCELERATION.total);
  const appliedBonus = newTotal - currentTotal;

  // Update breakdown
  const newBreakdown = {
    ...vesting.accelerationBreakdown,
  };

  switch (accelerationType) {
    case 'defense':
      newBreakdown.fromDefenses += appliedBonus;
      break;
    case 'replication':
      newBreakdown.fromReplications += appliedBonus;
      break;
    case 'corroboration':
      newBreakdown.fromCorroboration += appliedBonus;
      break;
    case 'citation':
      newBreakdown.fromCitations += appliedBonus;
      break;
  }

  // Update counts if applicable
  let defenseCount = vesting.defenseCount;
  let replicationCount = vesting.replicationCount;

  if (accelerationType === 'defense') {
    defenseCount++;
  } else if (accelerationType === 'replication') {
    replicationCount++;
  }

  return {
    ...vesting,
    accelerationBonus: newTotal,
    accelerationBreakdown: newBreakdown,
    defenseCount,
    replicationCount,
    updatedAt: Date.now(),
  };
}

/**
 * Get acceleration bonus amount for a type
 */
function getAccelerationBonus(type: AccelerationType): number {
  switch (type) {
    case 'defense':
      return ACCELERATION_BONUSES.successfulDefense;
    case 'replication':
      return ACCELERATION_BONUSES.independentReplication;
    case 'corroboration':
      return ACCELERATION_BONUSES.crossDomainCorroboration;
    case 'citation':
      return ACCELERATION_BONUSES.citationBonus;
  }
}

/**
 * Get current acceleration amount for a specific type
 */
function getTypeAcceleration(vesting: RewardVesting, type: AccelerationType): number {
  switch (type) {
    case 'defense':
      return vesting.accelerationBreakdown.fromDefenses;
    case 'replication':
      return vesting.accelerationBreakdown.fromReplications;
    case 'corroboration':
      return vesting.accelerationBreakdown.fromCorroboration;
    case 'citation':
      return vesting.accelerationBreakdown.fromCitations;
  }
}

/**
 * Get max cap for a specific acceleration type
 */
function getTypeAccelerationCap(type: AccelerationType): number {
  switch (type) {
    case 'defense':
      return MAX_ACCELERATION.successfulDefense;
    case 'replication':
      return MAX_ACCELERATION.independentReplication;
    case 'corroboration':
      return MAX_ACCELERATION.crossDomainCorroboration;
    case 'citation':
      return MAX_ACCELERATION.citationBonus;
  }
}

// ============ Deceleration Application ============

/**
 * Apply deceleration to a vesting schedule
 */
export function applyDeceleration(
  vesting: RewardVesting,
  decelerationType: DecelerationType,
  _currentBlock: bigint
): RewardVesting {
  // Check if vesting can be decelerated
  if (vesting.status !== 'vesting') {
    throw new VestingError(
      `Cannot decelerate vesting in status: ${vesting.status}`,
      'VESTING_COMPLETED'
    );
  }

  const factor = getDecelerationFactor(decelerationType);

  // If factor is 0, this is a pause, not deceleration
  if (factor === 0) {
    return pauseVesting(vesting, _currentBlock);
  }

  // Apply deceleration (multiply existing factor)
  const newFactor = vesting.decelerationFactor * factor;

  return {
    ...vesting,
    decelerationFactor: newFactor,
    updatedAt: Date.now(),
  };
}

/**
 * Get deceleration factor for a type
 */
function getDecelerationFactor(type: DecelerationType): number {
  switch (type) {
    case 'challenge':
      return DECELERATION_FACTORS.activeChallenge;
    case 'related_dispute':
      return DECELERATION_FACTORS.relatedFactDisputed;
    case 'domain_audit':
      return DECELERATION_FACTORS.domainAudit;
    case 'source_invalidated':
      return DECELERATION_FACTORS.sourceInvalidated;
  }
}

/**
 * Remove deceleration (restore to normal speed)
 */
export function removeDeceleration(
  vesting: RewardVesting,
  decelerationType: DecelerationType
): RewardVesting {
  if (vesting.status !== 'vesting') {
    return vesting;
  }

  const factor = getDecelerationFactor(decelerationType);

  // If it was a pause (factor = 0), this should use resumeVesting instead
  if (factor === 0) {
    throw new VestingError(
      'Use resumeVesting to unpause',
      'VESTING_PAUSED'
    );
  }

  // Reverse the deceleration
  const newFactor = Math.min(1.0, vesting.decelerationFactor / factor);

  return {
    ...vesting,
    decelerationFactor: newFactor,
    updatedAt: Date.now(),
  };
}

// ============ Pause/Resume ============

/**
 * Pause vesting (complete halt of release)
 */
export function pauseVesting(
  vesting: RewardVesting,
  currentBlock: bigint
): RewardVesting {
  if (vesting.isPaused) {
    throw new VestingError(
      'Vesting is already paused',
      'ALREADY_PAUSED'
    );
  }

  if (vesting.status !== 'vesting') {
    throw new VestingError(
      `Cannot pause vesting in status: ${vesting.status}`,
      'VESTING_COMPLETED'
    );
  }

  return {
    ...vesting,
    isPaused: true,
    pausedAtBlock: currentBlock,
    decelerationFactor: 0,
    updatedAt: Date.now(),
  };
}

/**
 * Resume vesting after pause
 *
 * Time spent paused is added to totalPausedBlocks
 * so it doesn't count toward release calculation
 */
export function resumeVesting(
  vesting: RewardVesting,
  currentBlock: bigint,
  applyDefenseBonus: boolean = false
): RewardVesting {
  if (!vesting.isPaused) {
    throw new VestingError(
      'Vesting is not paused',
      'NOT_PAUSED'
    );
  }

  if (vesting.status !== 'vesting') {
    throw new VestingError(
      `Cannot resume vesting in status: ${vesting.status}`,
      'VESTING_COMPLETED'
    );
  }

  // Calculate pause duration
  const pauseDuration = currentBlock - (vesting.pausedAtBlock || currentBlock);

  let updated: RewardVesting = {
    ...vesting,
    isPaused: false,
    pausedAtBlock: undefined,
    totalPausedBlocks: vesting.totalPausedBlocks + pauseDuration,
    decelerationFactor: 1.0, // Restore to normal speed
    updatedAt: Date.now(),
  };

  // Apply defense bonus if requested (successful defense)
  if (applyDefenseBonus) {
    updated.failedChallengeCount++;
    updated = applyAcceleration(updated, 'defense', currentBlock);
  }

  return updated;
}

// ============ Analysis Functions ============

/**
 * Calculate effective release speed multiplier
 *
 * Combined effect of acceleration and deceleration
 */
export function calculateSpeedMultiplier(vesting: RewardVesting): number {
  // Acceleration reduces half-life (speeds up)
  const accelerationMultiplier = 1 + vesting.accelerationBonus;

  // Deceleration slows down (can be 0 for pause)
  const decelerationMultiplier = vesting.decelerationFactor;

  // Combined effect
  return accelerationMultiplier * decelerationMultiplier;
}

/**
 * Get human-readable description of current speed
 */
export function describeSpeed(vesting: RewardVesting): string {
  if (vesting.isPaused) {
    return 'PAUSED';
  }

  const multiplier = calculateSpeedMultiplier(vesting);

  if (multiplier < 0.5) {
    return `Very slow (${(multiplier * 100).toFixed(0)}% speed)`;
  } else if (multiplier < 0.9) {
    return `Slowed (${(multiplier * 100).toFixed(0)}% speed)`;
  } else if (multiplier < 1.1) {
    return 'Normal speed';
  } else if (multiplier < 1.5) {
    return `Accelerated (${(multiplier * 100).toFixed(0)}% speed)`;
  } else {
    return `Fast (${(multiplier * 100).toFixed(0)}% speed)`;
  }
}

/**
 * Check if vesting can receive more acceleration
 */
export function canAccelerate(
  vesting: RewardVesting,
  accelerationType: AccelerationType
): { canAccelerate: boolean; reason?: string } {
  if (vesting.status !== 'vesting') {
    return { canAccelerate: false, reason: `Vesting is ${vesting.status}` };
  }

  if (vesting.isPaused) {
    return { canAccelerate: false, reason: 'Vesting is paused' };
  }

  // Check type cap
  const currentType = getTypeAcceleration(vesting, accelerationType);
  const typeCap = getTypeAccelerationCap(accelerationType);
  if (currentType >= typeCap) {
    return { canAccelerate: false, reason: `At ${accelerationType} cap` };
  }

  // Check total cap
  if (vesting.accelerationBonus >= MAX_ACCELERATION.total) {
    return { canAccelerate: false, reason: 'At total acceleration cap' };
  }

  return { canAccelerate: true };
}

/**
 * Get remaining acceleration capacity
 */
export function getRemainingAccelerationCapacity(
  vesting: RewardVesting
): {
  total: number;
  byType: Record<AccelerationType, number>;
} {
  const totalRemaining = MAX_ACCELERATION.total - vesting.accelerationBonus;

  const byType: Record<AccelerationType, number> = {
    defense: MAX_ACCELERATION.successfulDefense - vesting.accelerationBreakdown.fromDefenses,
    replication: MAX_ACCELERATION.independentReplication - vesting.accelerationBreakdown.fromReplications,
    corroboration: MAX_ACCELERATION.crossDomainCorroboration - vesting.accelerationBreakdown.fromCorroboration,
    citation: MAX_ACCELERATION.citationBonus - vesting.accelerationBreakdown.fromCitations,
  };

  return { total: totalRemaining, byType };
}
