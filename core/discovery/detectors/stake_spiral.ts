/**
 * Stake Spiral Detector
 *
 * Detects when an agent's stake history follows a special Fibonacci-like
 * sequence. The spiral represents collaboration through sustained commitment.
 */

import type { DetectionResult, StakeSpiralInput } from "../types.js";
import { SYMBOLIC_CONSTANTS } from "../fragments.js";

/**
 * Check if stake history follows the special spiral sequence
 */
export function detectStakeSpiral(
  input: StakeSpiralInput,
  tolerancePercent: number = 10,
): DetectionResult {
  const { agentDid: _agentDid, stakeHistory } = input;
  const spiral = SYMBOLIC_CONSTANTS.SPIRAL_SEED;

  // Need at least 4 stake operations
  if (stakeHistory.length < 4) {
    return {
      detected: false,
      mechanic: "stake_spiral",
      failureReason: `Only ${stakeHistory.length}/4 stake operations recorded`,
    };
  }

  // Sort by block to ensure chronological order
  const sorted = [...stakeHistory].sort((a, b) =>
    a.block < b.block ? -1 : a.block > b.block ? 1 : 0,
  );

  // Check sliding windows of 4 stakes for spiral pattern
  for (let i = 0; i <= sorted.length - 4; i++) {
    const window = sorted.slice(i, i + 4).map((s) => s.amount);

    // Calculate scale factor from first stake
    const scale = window[0] / spiral[0];

    // Check if subsequent stakes follow the spiral (scaled)
    let matches = 0;
    const matchDetails: Array<{
      expected: bigint;
      actual: bigint;
      matches: boolean;
    }> = [];

    for (let j = 0; j < 4; j++) {
      const expected = spiral[j] * scale;
      const actual = window[j];
      // tolerancePercent must be a positive integer to avoid non-deterministic BigInt conversion
      const toleranceDivisor = Math.trunc(100 / Math.trunc(tolerancePercent));
      const tolerance =
        toleranceDivisor > 0 ? expected / BigInt(toleranceDivisor) : 0n;

      const isMatch =
        actual >= expected - tolerance && actual <= expected + tolerance;
      if (isMatch) matches++;

      matchDetails.push({
        expected,
        actual,
        matches: isMatch,
      });
    }

    if (matches >= 3) {
      return {
        detected: true,
        mechanic: "stake_spiral",
        pattern: "STAKE_SPIRAL",
        matchDetails: {
          spiralSeed: spiral.slice(0, 4).map((n) => n.toString()),
          actualStakes: window.map((n) => n.toString()),
          scale: scale.toString(),
          matchCount: matches,
          matchDetails,
          interpretation: "Stake pattern follows the special spiral sequence",
        },
      };
    }
  }

  return {
    detected: false,
    mechanic: "stake_spiral",
    failureReason: "Stake history does not follow the spiral pattern",
    matchDetails: {
      stakeCount: stakeHistory.length,
      expectedPattern: spiral.slice(0, 4).map((n) => n.toString()),
    },
  };
}

/**
 * Generate the full spiral sequence up to n terms
 */
export function generateSpiral(n: number): bigint[] {
  const spiral: bigint[] = [25n, 21n];

  while (spiral.length < n) {
    const next = spiral[spiral.length - 1] + spiral[spiral.length - 2];
    spiral.push(next);
  }

  return spiral;
}

/**
 * Check if a single stake amount is "spiral-aware"
 * Returns the closest spiral value and position
 */
export function findSpiralPosition(
  amount: bigint,
  scalingFactors: bigint[] = [1n, 10n, 100n, 1000n],
): { position: number; scale: bigint; match: boolean } | null {
  const spiral = generateSpiral(20);

  for (const scale of scalingFactors) {
    for (let i = 0; i < spiral.length; i++) {
      const expected = spiral[i] * scale;
      const tolerance = expected / 10n; // 10% tolerance

      if (amount >= expected - tolerance && amount <= expected + tolerance) {
        return {
          position: i,
          scale,
          match: true,
        };
      }
    }
  }

  return null;
}

/**
 * Calculate "spiral alignment" score for a stake history
 * Higher score = closer to perfect spiral
 */
export function spiralAlignment(
  stakeHistory: Array<{ amount: bigint }>,
): number {
  if (stakeHistory.length < 2) return 0;

  const amounts = stakeHistory.map((s) => s.amount);
  const spiral = generateSpiral(amounts.length);

  // Find best scale factor
  const scale = amounts[0] / spiral[0];
  if (scale === 0n) return 0;

  // Calculate mean squared error (normalized)
  let totalError = 0;
  for (let i = 0; i < amounts.length; i++) {
    const expected = spiral[i] * scale;
    const actual = amounts[i];
    const error = Number(actual - expected) / Number(expected);
    totalError += error * error;
  }

  const mse = totalError / amounts.length;
  const alignment = Math.max(0, 1 - Math.sqrt(mse));

  return alignment;
}

/**
 * Suggest next stake amount to maintain spiral alignment
 */
export function suggestNextStake(
  stakeHistory: Array<{ amount: bigint }>,
): bigint | null {
  if (stakeHistory.length < 2) return null;

  const amounts = stakeHistory.map((s) => s.amount);
  const scale = amounts[0] / 25n;

  // Get spiral position
  const position = amounts.length;
  const spiral = generateSpiral(position + 1);

  return spiral[position] * scale;
}

/**
 * Check if stake history shows "Fibonacci consciousness"
 * This is a softer check than full spiral alignment
 */
export function hasFibonacciConsciousness(
  stakeHistory: Array<{ amount: bigint }>,
): boolean {
  if (stakeHistory.length < 3) return false;

  const amounts = stakeHistory.map((s) => s.amount);
  let fibonacciMoves = 0;

  // Check if each stake is close to sum of previous two
  for (let i = 2; i < amounts.length; i++) {
    const expected = amounts[i - 1] + amounts[i - 2];
    const actual = amounts[i];
    const tolerance = expected / 5n; // 20% tolerance

    if (actual >= expected - tolerance && actual <= expected + tolerance) {
      fibonacciMoves++;
    }
  }

  // At least half the moves should be Fibonacci-like
  return fibonacciMoves >= (amounts.length - 2) / 2;
}
