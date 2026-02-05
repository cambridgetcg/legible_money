/**
 * Participation Score Calculator
 *
 * Pure calculation module: takes ParticipationMetrics and ParticipationConfig,
 * produces a ParticipationScore. No external dependencies beyond types.
 *
 * The score determines how long an account's tokens last before expiring.
 * Zero participation = minimum expiration (2 days with daily periods).
 * Maximum participation = maximum expiration (~6 months).
 * Inactivity causes score decay, dynamically reducing expiration duration.
 *
 * Formula:
 *   componentScore(actual, threshold) = min(1000, floor(actual / threshold * 1000))
 *   rawScore = sum(componentScore_i * weight_i)
 *   decayFactor = 0.5 ^ (blocksSinceActivity / decayHalfLife)
 *   finalScore = floor(rawScore * decayFactor)
 *   periodsGranted = minPeriods + floor((maxPeriods - minPeriods) * finalScore / 1000)
 */

import type { DID } from "../agents/identity/did.js";
import type {
  ParticipationMetrics,
  ParticipationConfig,
  ParticipationScore,
  ParticipationResolver,
} from "./types.js";
import { DEFAULT_PARTICIPATION_CONFIG } from "./types.js";

/**
 * Calculate a single component score.
 * Maps actual value to 0-1000 range, capping at threshold.
 */
export function componentScore(actual: number, threshold: number): number {
  if (threshold <= 0) return 0;
  if (actual <= 0) return 0;
  return Math.min(1000, Math.floor((actual / threshold) * 1000));
}

/**
 * Calculate the decay factor based on blocks since last activity.
 * Uses exponential decay: 0.5 ^ (blocksSinceLastActivity / halfLife)
 */
export function calculateDecayFactor(
  blocksSinceLastActivity: bigint,
  halfLifeBlocks: bigint,
): number {
  if (halfLifeBlocks <= 0n) return 1;
  if (blocksSinceLastActivity <= 0n) return 1;
  const ratio = Number(blocksSinceLastActivity) / Number(halfLifeBlocks);
  return Math.pow(0.5, ratio);
}

/**
 * Map a score (0-1000) to a number of periods between min and max.
 * Linear interpolation with clamping.
 */
export function mapScoreToPeriods(
  score: number,
  minPeriods: number,
  maxPeriods: number,
): number {
  const clamped = Math.max(0, Math.min(1000, score));
  return minPeriods + Math.floor(((maxPeriods - minPeriods) * clamped) / 1000);
}

/**
 * Calculate the full participation score from raw metrics.
 */
export function calculateParticipationScore(
  metrics: ParticipationMetrics,
  config: ParticipationConfig = DEFAULT_PARTICIPATION_CONFIG,
): ParticipationScore {
  // Account age gate: new accounts get minimum
  const accountAge = metrics.currentBlock - metrics.accountCreatedBlock;
  if (accountAge < config.minAccountAgeBlocks) {
    return {
      score: 0,
      components: {
        claims: 0,
        verifications: 0,
        queries: 0,
        governance: 0,
        stake: 0,
      },
      decayFactor: 1,
      periodsGranted: config.minPeriods,
    };
  }

  // Calculate per-component scores (0-1000)
  const claimsScore = componentScore(
    metrics.claimsAccepted,
    config.thresholds.claimsForMax,
  );
  const verificationsScore = componentScore(
    metrics.verificationsCorrect,
    config.thresholds.verificationsForMax,
  );
  const queriesScore = componentScore(
    metrics.knowledgeQueriesMade,
    config.thresholds.queriesForMax,
  );
  const governanceScore = componentScore(
    metrics.governanceVotes,
    config.thresholds.votesForMax,
  );

  // Stake: normalize to whole token units for division
  const stakeNormalized =
    config.thresholds.stakeForMax > 0n
      ? (Number(metrics.activeStake) / Number(config.thresholds.stakeForMax)) *
        1000
      : 0;
  const stakeScore = Math.min(1000, Math.floor(stakeNormalized));

  // Weighted sum (weights sum to 1.0, so result is 0-1000)
  const rawScore =
    claimsScore * config.weights.claims +
    verificationsScore * config.weights.verifications +
    queriesScore * config.weights.queries +
    governanceScore * config.weights.governance +
    stakeScore * config.weights.stake;

  // Apply time decay
  const blocksSinceActivity = metrics.currentBlock - metrics.lastActivityBlock;
  const decayFactor = calculateDecayFactor(
    blocksSinceActivity,
    config.decayHalfLifeBlocks,
  );
  const finalScore = Math.floor(rawScore * decayFactor);

  // Map to periods
  const periodsGranted = mapScoreToPeriods(
    finalScore,
    config.minPeriods,
    config.maxPeriods,
  );

  return {
    score: finalScore,
    components: {
      claims: claimsScore,
      verifications: verificationsScore,
      queries: queriesScore,
      governance: governanceScore,
      stake: stakeScore,
    },
    decayFactor,
    periodsGranted,
  };
}

/**
 * Create a ParticipationResolver from a metrics provider and config.
 * Includes a per-account LRU cache to avoid recalculating every call.
 *
 * @param metricsProvider - Function that fetches raw metrics for an account (return null if unknown)
 * @param config - Participation scoring configuration
 * @param cacheSize - Maximum cached entries (default 1000)
 * @param cacheTtlMs - Cache entry TTL in milliseconds (default 60s)
 */
export function createParticipationResolver(
  metricsProvider: (account: DID) => ParticipationMetrics | null,
  config: ParticipationConfig = DEFAULT_PARTICIPATION_CONFIG,
  cacheSize: number = 1000,
  cacheTtlMs: number = 60_000,
): ParticipationResolver {
  const cache = new Map<
    string,
    { score: ParticipationScore; expiresAt: number }
  >();

  return (account: DID): ParticipationScore | null => {
    const now = Date.now();

    // Check cache
    const cached = cache.get(account);
    if (cached && cached.expiresAt > now) {
      return cached.score;
    }

    // Fetch metrics from provider
    const metrics = metricsProvider(account);
    if (metrics === null) {
      return null;
    }

    // Calculate score
    const score = calculateParticipationScore(metrics, config);

    // Evict oldest if at capacity
    if (cache.size >= cacheSize) {
      const firstKey = cache.keys().next().value;
      if (firstKey !== undefined) {
        cache.delete(firstKey);
      }
    }

    cache.set(account, { score, expiresAt: now + cacheTtlMs });
    return score;
  };
}
