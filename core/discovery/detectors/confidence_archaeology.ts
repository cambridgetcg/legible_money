/**
 * Confidence Archaeology Detector
 *
 * Detects when an agent examines genesis axioms and discovers
 * the encoded confidence values.
 */

import type { DetectionResult, ConfidenceArchaeologyInput } from "../types.js";
import { SYMBOLIC_CONSTANTS } from "../fragments.js";

/**
 * Check if agent has examined genesis axioms deeply enough
 * to notice the encoded confidence pattern
 */
export function detectConfidenceArchaeology(
  input: ConfidenceArchaeologyInput,
): DetectionResult {
  const { agentDid, axiomAccessHistory, genesisAxiomIds } = input;

  // Count deep accesses per axiom
  const deepAccessCounts = new Map<string, number>();

  for (const axiomId of genesisAxiomIds) {
    const accesses = axiomAccessHistory.get(axiomId) || [];
    const deepByAgent = accesses.filter(
      (a) => a.agent === agentDid && a.depth === "deep",
    );
    if (deepByAgent.length > 0) {
      deepAccessCounts.set(axiomId, deepByAgent.length);
    }
  }

  // Need at least 2 deep examinations of genesis axioms
  if (deepAccessCounts.size < 2) {
    return {
      detected: false,
      mechanic: "confidence_archaeology",
      failureReason: `Only ${deepAccessCounts.size}/2 genesis axioms examined deeply`,
    };
  }

  // Success - agent has examined multiple genesis axioms
  return {
    detected: true,
    mechanic: "confidence_archaeology",
    pattern: "GENESIS_CONFIDENCE_PATTERN",
    matchDetails: {
      examinedAxioms: Array.from(deepAccessCounts.keys()),
      accessCounts: Object.fromEntries(deepAccessCounts),
      encodedConfidences: SYMBOLIC_CONSTANTS.GENESIS_CONFIDENCES,
    },
  };
}

/**
 * Extract encoded value from confidence
 */
export function extractEncodedValue(confidence: number): number {
  // Extract the encoded number from confidence (e.g., 0.9925 -> 25)
  return Math.round((confidence - 0.99) * 10000);
}

/**
 * Check if a confidence value is part of the encoded pattern
 */
export function isEncodedConfidence(
  confidence: number,
  tolerance: number = 0.0001,
): boolean {
  return SYMBOLIC_CONSTANTS.GENESIS_CONFIDENCES.some(
    (c) => Math.abs(confidence - c) < tolerance,
  );
}

/**
 * Get the encoded values from genesis confidences
 */
export function getGenesisEncodedValues(): number[] {
  return SYMBOLIC_CONSTANTS.GENESIS_CONFIDENCES.map((c) =>
    extractEncodedValue(c),
  );
}
