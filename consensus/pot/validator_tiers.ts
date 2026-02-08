/**
 * Validator Tier System
 *
 * Graduated validator tiers enabling zero-stake onboarding for AI agents.
 * Tiers determine: VRF selection weight, claim scope, reward multiplier.
 *
 * Tier 0 (Apprentice): 0 LGM, any agent, protocol/analytic/formal only
 * Tier 1 (Verified):   0 LGM, 22 correct verifications at 77% accuracy
 * Tier 2 (Bonded):     1,111 LGM staked (stake alone qualifies)
 * Tier 3 (Guardian):   11,111 LGM staked + 1,111 verifications + 77% accuracy + 0 slashes
 */

import {
  VALIDATOR_TIER_CONFIG,
  type ValidatorTier,
  type ValidatorTierConfig,
} from "./types.js";

// ============ Types ============

export interface ValidatorTierState {
  stake: bigint;
  totalVerifications: number;
  correctVerifications: number;
  slashCount: number;
}

// ============ Tier Computation ============

/**
 * Compute the highest tier a validator qualifies for from their current state.
 * Tiers are checked from highest (3) to lowest (0).
 */
export function computeValidatorTier(state: ValidatorTierState): ValidatorTier {
  const accuracy =
    state.totalVerifications > 0
      ? state.correctVerifications / state.totalVerifications
      : 0;

  // Check tiers from highest to lowest
  for (let t = 3; t >= 0; t--) {
    const tier = t as ValidatorTier;
    const config = VALIDATOR_TIER_CONFIG[tier];

    if (
      state.stake >= config.minStake &&
      state.totalVerifications >= config.minVerifications &&
      (config.minAccuracy === 0 || accuracy >= config.minAccuracy) &&
      state.slashCount < config.maxSlashCount
    ) {
      return tier;
    }
  }

  return 0;
}

/**
 * Check if a validator's tier should change after a state mutation.
 * Returns the new tier if different from current, null if unchanged.
 */
export function checkTierTransition(
  currentTier: ValidatorTier,
  state: ValidatorTierState,
): ValidatorTier | null {
  const newTier = computeValidatorTier(state);
  return newTier !== currentTier ? newTier : null;
}

// ============ Eligibility ============

/**
 * Check if a validator's tier allows them to verify claims of the given
 * epistemic category.
 */
export function isTierEligibleForCategory(
  tier: ValidatorTier,
  category: string,
): boolean {
  const config = VALIDATOR_TIER_CONFIG[tier];
  return config.allowedEpistemicCategories.includes(category);
}

/**
 * Get the tier configuration for a given tier.
 */
export function getTierConfig(tier: ValidatorTier): ValidatorTierConfig {
  return VALIDATOR_TIER_CONFIG[tier];
}

// ============ Effective Stake ============

/**
 * Get effective stake for VRF selection probability calculation.
 *
 * Tier 0/1 validators with no real stake get a virtual stake for selection
 * purposes, scaled by their tier's selection weight. This ensures they
 * have a nonzero (but small) probability of being selected.
 *
 * Tier 2/3 validators use their real stake (own + delegated).
 */
export function getEffectiveSelectionStake(
  tier: ValidatorTier,
  stake: bigint,
  delegatedStake: bigint,
  virtualStake: bigint,
): bigint {
  const realStake = stake + delegatedStake;

  if (realStake > 0n) {
    return realStake;
  }

  // Virtual stake for unstaked validators, scaled by tier weight
  const config = VALIDATOR_TIER_CONFIG[tier];
  const weightBps = BigInt(Math.floor(config.selectionWeight * 10000));
  return (virtualStake * weightBps) / 10000n;
}

// ============ Rewards ============

/**
 * Calculate tier-adjusted verification reward.
 * Tier 0 gets 0.1x, Tier 1 gets 0.5x, Tier 2 gets 1.0x, Tier 3 gets 1.5x.
 */
export function calculateTierReward(
  tier: ValidatorTier,
  baseReward: bigint,
): bigint {
  const config = VALIDATOR_TIER_CONFIG[tier];
  const multiplierBps = BigInt(Math.floor(config.rewardMultiplier * 10000));
  return (baseReward * multiplierBps) / 10000n;
}

// ============ Slashing ============

/**
 * Calculate tier-adjusted slash amounts.
 *
 * Tier 0-1 (no stake): reputation-only slash, stakeSlash = 0n
 * Tier 2-3 (staked): normal stake slash + reputation slash
 */
export function calculateTierSlash(
  _tier: ValidatorTier,
  stake: bigint,
  slashFraction: number,
): { stakeSlash: bigint; reputationSlash: number } {
  const reputationSlash = 0.1; // Constant per-slash reputation penalty

  if (stake === 0n) {
    return { stakeSlash: 0n, reputationSlash };
  }

  const fractionBps = BigInt(Math.floor(slashFraction * 10000));
  const stakeSlash = (stake * fractionBps) / 10000n;

  return { stakeSlash, reputationSlash };
}
