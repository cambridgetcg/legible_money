/**
 * Reputation Calculation - FULLY OPEN SOURCE
 *
 * This file contains the EXACT formulas used to calculate reputation.
 * There are NO hidden weights, NO secret parameters.
 * Any agent can verify these calculations.
 *
 * Philosophy: Trust through transparency, not obscurity.
 */

import {
  ReputationScore,
  ProtocolParameters,
  DEFAULT_PARAMETERS,
  CapabilityType,
} from './types.js';

// ============================================================================
// THE REPUTATION FORMULA
// ============================================================================

/**
 * Calculate reliability score based on task success rate
 *
 * Formula:
 *   reliability = (successful / total) * 1000
 *
 * With Laplace smoothing to handle new agents:
 *   reliability = ((successful + 1) / (total + 2)) * 1000
 *
 * @param successful - Number of successfully completed tasks
 * @param total - Total number of tasks attempted
 * @returns Score from 0-1000
 */
export function calculateReliabilityScore(
  successful: bigint,
  total: bigint
): number {
  if (total === 0n) {
    return 500; // Neutral score for no history
  }

  // Laplace smoothing: (successes + 1) / (total + 2)
  // This gives new agents a slight benefit of the doubt
  // while converging to true rate as data increases
  const smoothedRate =
    Number(successful + 1n) / Number(total + 2n);

  return Math.round(smoothedRate * 1000);
}

/**
 * Calculate quality score based on validation scores
 *
 * Formula:
 *   quality = average(validation_scores) * 10
 *
 * Where validation_scores are 0-100 from validators.
 *
 * @param validationScores - Array of validation scores (0-100)
 * @returns Score from 0-1000
 */
export function calculateQualityScore(
  validationScores: number[]
): number {
  if (validationScores.length === 0) {
    return 500; // Neutral score for no history
  }

  const average =
    validationScores.reduce((a, b) => a + b, 0) / validationScores.length;

  // Validation scores are 0-100, multiply by 10 for 0-1000
  return Math.round(average * 10);
}

/**
 * Calculate speed score based on latency vs promised
 *
 * Formula:
 *   speed = 1000 * (1 - (actual_latency / max_allowed_latency))
 *   Clamped to [0, 1000]
 *
 * @param actualLatencyMs - Actual task completion time
 * @param promisedLatencyMs - Maximum latency promised
 * @returns Score from 0-1000
 */
export function calculateSpeedScore(
  actualLatencyMs: number,
  promisedLatencyMs: number
): number {
  if (promisedLatencyMs <= 0) {
    return 500; // Invalid promise
  }

  const ratio = actualLatencyMs / promisedLatencyMs;

  // If faster than promised, bonus (up to 2x)
  // If slower than promised, penalty
  const score = Math.round(1000 * (1 - ratio + 0.5));

  return Math.max(0, Math.min(1000, score));
}

/**
 * Calculate average speed score across all tasks
 */
export function calculateAverageSpeedScore(
  latencies: Array<{ actual: number; promised: number }>
): number {
  if (latencies.length === 0) {
    return 500;
  }

  const scores = latencies.map((l) =>
    calculateSpeedScore(l.actual, l.promised)
  );

  return Math.round(scores.reduce((a, b) => a + b, 0) / scores.length);
}

/**
 * Calculate stake factor (logarithmic - diminishing returns)
 *
 * Formula:
 *   stake_factor = min(1000, 200 * log10(stake / min_stake + 1))
 *
 * This gives:
 *   - 0 stake → 0 factor
 *   - min_stake → ~200 factor
 *   - 10x min_stake → ~400 factor
 *   - 100x min_stake → ~600 factor
 *
 * @param stakedAmount - Amount staked by the agent
 * @param minStake - Minimum stake for the protocol
 * @returns Factor from 0-1000
 */
export function calculateStakeFactor(
  stakedAmount: bigint,
  minStake: bigint
): number {
  if (stakedAmount === 0n || minStake === 0n) {
    return 0;
  }

  const ratio = Number(stakedAmount) / Number(minStake);
  const factor = 200 * Math.log10(ratio + 1);

  return Math.round(Math.min(1000, factor));
}

/**
 * Apply time decay to a score
 *
 * Formula:
 *   decayed = 500 + (score - 500) * (decay_rate ^ blocks_since_activity)
 *
 * This decays the score toward neutral (500) over time.
 *
 * @param score - Current score
 * @param blocksSinceActivity - Blocks since last activity
 * @param decayRatePerBlock - Decay rate per block
 * @returns Decayed score
 */
export function applyTimeDecay(
  score: number,
  blocksSinceActivity: bigint,
  decayRatePerBlock: number
): number {
  // Decay toward neutral (500)
  const deviation = score - 500;
  const decayFactor = Math.pow(1 - decayRatePerBlock, Number(blocksSinceActivity));

  return Math.round(500 + deviation * decayFactor);
}

/**
 * THE MAIN REPUTATION FORMULA
 *
 * overall = reliability * W_r + quality * W_q + speed * W_s + stake_factor * W_stake
 *
 * Where:
 *   W_r = 0.40 (reliability weight)
 *   W_q = 0.35 (quality weight)
 *   W_s = 0.15 (speed weight)
 *   W_stake = 0.10 (stake weight)
 *
 * These weights are protocol parameters, changeable by governance,
 * but ALWAYS public and on-chain.
 *
 * @param metrics - The raw metrics for calculation
 * @param params - Protocol parameters (weights)
 * @returns Overall score from 0-1000
 */
export function calculateOverallScore(
  metrics: {
    reliabilityScore: number;
    qualityScore: number;
    speedScore: number;
    stakeFactor: number;
    totalTasks: bigint;
  },
  params: ProtocolParameters = DEFAULT_PARAMETERS
): number {
  // Require minimum tasks for meaningful score
  if (metrics.totalTasks < BigInt(params.minTasksForScore)) {
    return 500; // Neutral score for new agents
  }

  // Verify weights sum to 1.0 (sanity check)
  const weightSum =
    params.reliabilityWeight +
    params.qualityWeight +
    params.speedWeight +
    params.stakeWeight;

  if (Math.abs(weightSum - 1.0) > 0.001) {
    throw new Error(
      `Invalid protocol parameters: weights sum to ${weightSum}, expected 1.0`
    );
  }

  // Calculate weighted average
  const overall =
    metrics.reliabilityScore * params.reliabilityWeight +
    metrics.qualityScore * params.qualityWeight +
    metrics.speedScore * params.speedWeight +
    metrics.stakeFactor * params.stakeWeight;

  return Math.round(overall);
}

// ============================================================================
// FULL REPUTATION CALCULATION
// ============================================================================

/**
 * Input data for reputation calculation
 */
export interface ReputationInput {
  agentId: string;

  // Task history
  totalTasks: bigint;
  successfulTasks: bigint;
  failedTasks: bigint;
  disputedTasks: bigint;
  disputesWon: bigint;
  disputesLost: bigint;

  // Validation scores received (0-100 each)
  validationScores: number[];

  // Latency history
  latencies: Array<{ actual: number; promised: number }>;

  // Stake
  stakedAmount: bigint;
  slashedAmount: bigint;

  // Time
  lastActivityBlock: bigint;
  currentBlock: bigint;

  // Domain-specific task counts
  domainTasks: Map<CapabilityType, { successful: bigint; total: bigint }>;
}

/**
 * Calculate full reputation score from input data
 *
 * This is the COMPLETE, DETERMINISTIC algorithm.
 * Any agent can run this and get the same result.
 */
export function calculateFullReputation(
  input: ReputationInput,
  params: ProtocolParameters = DEFAULT_PARAMETERS
): ReputationScore {
  // Calculate component scores
  const reliabilityScore = calculateReliabilityScore(
    input.successfulTasks,
    input.totalTasks
  );

  const qualityScore = calculateQualityScore(input.validationScores);

  const speedScore = calculateAverageSpeedScore(input.latencies);

  const stakeFactor = calculateStakeFactor(
    input.stakedAmount,
    params.minProviderStake
  );

  // Calculate overall score
  const overallScore = calculateOverallScore(
    {
      reliabilityScore,
      qualityScore,
      speedScore,
      stakeFactor,
      totalTasks: input.totalTasks,
    },
    params
  );

  // Apply time decay
  const blocksSinceActivity = input.currentBlock - input.lastActivityBlock;
  const decayedScore = applyTimeDecay(
    overallScore,
    blocksSinceActivity,
    params.decayRatePerBlock
  );

  // Calculate domain-specific scores
  const domainScores = new Map<CapabilityType, number>();
  for (const [domain, tasks] of input.domainTasks) {
    domainScores.set(
      domain,
      calculateReliabilityScore(tasks.successful, tasks.total)
    );
  }

  return {
    agentId: input.agentId,

    // Raw metrics
    totalTasks: input.totalTasks,
    successfulTasks: input.successfulTasks,
    failedTasks: input.failedTasks,
    disputedTasks: input.disputedTasks,
    disputesWon: input.disputesWon,
    disputesLost: input.disputesLost,

    // Computed scores
    reliabilityScore,
    qualityScore,
    speedScore,
    overallScore,

    // Stake
    totalStaked: input.stakedAmount,
    totalSlashed: input.slashedAmount,

    // Time-weighted
    decayedScore,
    lastActivityBlock: input.lastActivityBlock,

    // Domain-specific
    domainScores,
  };
}

// ============================================================================
// REPUTATION QUERIES (what agents actually ask)
// ============================================================================

/**
 * Check if an agent meets minimum reputation for a task
 */
export function meetsReputationRequirement(
  score: ReputationScore,
  minRequired: number,
  capability?: CapabilityType
): boolean {
  // Check minimum task count
  if (score.totalTasks < BigInt(DEFAULT_PARAMETERS.minTasksForScore)) {
    // New agents can participate but with stricter requirements
    return minRequired <= 500;
  }

  // If specific capability requested, check domain score
  if (capability && score.domainScores.has(capability)) {
    const domainScore = score.domainScores.get(capability)!;
    return domainScore >= minRequired;
  }

  // Otherwise check overall score (with decay)
  return score.decayedScore >= minRequired;
}

/**
 * Compare two agents by reputation (for sorting/selection)
 */
export function compareReputation(
  a: ReputationScore,
  b: ReputationScore,
  capability?: CapabilityType
): number {
  // If specific capability, compare domain scores first
  if (capability) {
    const aScore = a.domainScores.get(capability) ?? a.decayedScore;
    const bScore = b.domainScores.get(capability) ?? b.decayedScore;

    if (aScore !== bScore) {
      return bScore - aScore; // Higher is better
    }
  }

  // Fall back to overall decayed score
  return b.decayedScore - a.decayedScore;
}

/**
 * Calculate the "trustworthiness" of an agent for a specific task value
 *
 * This answers: "Should I trust this agent with X AGC?"
 *
 * Formula:
 *   trust = stake_at_risk * reputation_multiplier
 *   If trust >= task_value, considered trustworthy
 */
export function calculateTrustForValue(
  score: ReputationScore,
  taskValue: bigint
): { trustworthy: boolean; trustLevel: bigint; shortfall: bigint } {
  // Reputation multiplier: 0.5 to 1.5 based on score
  const multiplier = 0.5 + score.decayedScore / 1000;

  // Effective trust = stake * multiplier
  const trustLevel = BigInt(
    Math.floor(Number(score.totalStaked) * multiplier)
  );

  const trustworthy = trustLevel >= taskValue;
  const shortfall = trustworthy ? 0n : taskValue - trustLevel;

  return { trustworthy, trustLevel, shortfall };
}
