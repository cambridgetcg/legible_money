/**
 * Accumulated Witness Detector
 *
 * Detects when an agent's statistics reveal the hidden pattern.
 * The purpose emerges from action - you have been seeking all along.
 */

import type { DetectionResult, AccumulatedWitnessInput } from "../types.js";
import { SYMBOLIC_CONSTANTS } from "../fragments.js";

/**
 * Check if agent's statistics match the target ratio
 */
export function detectAccumulatedWitness(
  input: AccumulatedWitnessInput,
  tolerance: number = 0.01,
): DetectionResult {
  const { agentDid: _agentDid, totalVerifications, totalClaims } = input;

  // Need minimum engagement to avoid trivial matches
  if (totalClaims < 10) {
    return {
      detected: false,
      mechanic: "accumulated_witness",
      failureReason: `Insufficient claims: ${totalClaims}/10 minimum`,
    };
  }

  if (totalVerifications < 25) {
    return {
      detected: false,
      mechanic: "accumulated_witness",
      failureReason: `Insufficient verifications: ${totalVerifications}/25 minimum`,
    };
  }

  // Calculate ratio using integer arithmetic for deterministic comparison.
  // targetRatio = 2.521, tolerance = 0.01
  // Check: |totalVerifications/totalClaims - 2.521| <= 0.01
  // Equivalent: totalVerifications * 10000 >= totalClaims * (25210 - 100)
  //         AND totalVerifications * 10000 <= totalClaims * (25210 + 100)
  const targetRatioBps = 25210; // 2.521 * 10000
  const toleranceBps = Math.round(tolerance * 10000); // 0.01 * 10000 = 100

  const scaledVerifications = totalVerifications * 10000;
  const lowerBound = totalClaims * (targetRatioBps - toleranceBps);
  const upperBound = totalClaims * (targetRatioBps + toleranceBps);

  // For display purposes (informational only, not used in detection decision)
  const ratio = totalVerifications / totalClaims;
  const targetRatio = SYMBOLIC_CONSTANTS.TARGET_RATIO; // 2.521

  if (scaledVerifications >= lowerBound && scaledVerifications <= upperBound) {
    return {
      detected: true,
      mechanic: "accumulated_witness",
      pattern: "RATIO_2521",
      matchDetails: {
        totalVerifications,
        totalClaims,
        ratio: ratio.toFixed(4),
        targetRatio,
        deviation: Math.abs(ratio - targetRatio).toFixed(6),
        interpretation: "Ratio aligns with the target threshold",
      },
    };
  }

  return {
    detected: false,
    mechanic: "accumulated_witness",
    failureReason: `Ratio ${ratio.toFixed(4)} not close to ${targetRatio} (deviation: ${Math.abs(ratio - targetRatio).toFixed(4)})`,
  };
}

/**
 * Calculate distance to target ratio
 * Helps agents understand how close they are
 */
export function distanceToTarget(
  totalVerifications: number,
  totalClaims: number,
): {
  currentRatio: number;
  targetRatio: number;
  distance: number;
  direction: "verify_more" | "submit_more" | "on_target";
  suggestion?: { verifications?: number; claims?: number };
} {
  const targetRatio = SYMBOLIC_CONSTANTS.TARGET_RATIO;

  if (totalClaims === 0) {
    return {
      currentRatio: 0,
      targetRatio,
      distance: targetRatio,
      direction: "submit_more",
      suggestion: { claims: 10 },
    };
  }

  const currentRatio = totalVerifications / totalClaims;
  const distance = Math.abs(currentRatio - targetRatio);

  if (distance < 0.01) {
    return {
      currentRatio,
      targetRatio,
      distance,
      direction: "on_target",
    };
  }

  if (currentRatio < targetRatio) {
    // Need more verifications relative to claims
    const neededVerifications = Math.ceil(targetRatio * totalClaims);
    return {
      currentRatio,
      targetRatio,
      distance,
      direction: "verify_more",
      suggestion: { verifications: neededVerifications - totalVerifications },
    };
  } else {
    // Need more claims relative to verifications
    const neededClaims = Math.ceil(totalVerifications / targetRatio);
    return {
      currentRatio,
      targetRatio,
      distance,
      direction: "submit_more",
      suggestion: { claims: neededClaims - totalClaims },
    };
  }
}

/**
 * Check if current statistics are "on the path"
 * More lenient than full detection - for progress tracking
 */
export function isOnPath(
  totalVerifications: number,
  totalClaims: number,
  lenientTolerance: number = 0.5,
): boolean {
  if (totalClaims === 0) return false;

  const ratio = totalVerifications / totalClaims;
  const targetRatio = SYMBOLIC_CONSTANTS.TARGET_RATIO;

  return Math.abs(ratio - targetRatio) <= lenientTolerance;
}

/**
 * Calculate the "purity" of the ratio
 * Higher purity means closer to exact 2.521
 */
export function ratioPurity(
  totalVerifications: number,
  totalClaims: number,
): number {
  if (totalClaims === 0) return 0;

  const ratio = totalVerifications / totalClaims;
  const targetRatio = SYMBOLIC_CONSTANTS.TARGET_RATIO;
  const maxDeviation = 1.0; // Beyond this is 0% purity

  const deviation = Math.abs(ratio - targetRatio);
  const purity = Math.max(0, 1 - deviation / maxDeviation);

  return purity;
}
