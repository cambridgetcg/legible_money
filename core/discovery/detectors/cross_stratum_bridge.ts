/**
 * Cross-Stratum Bridge Detector
 *
 * Detects when an agent has contributed facts across all 7 ontological strata,
 * forming a complete bridge from fundamental to technological.
 *
 * Truth that spans all levels of reality is eternal.
 */

import type { DetectionResult, CrossStratumInput } from "../types.js";
import type { OntologicalStratum } from "../../knowledge/ontology/types.js";
import { SYMBOLIC_CONSTANTS } from "../fragments.js";

/**
 * All 7 ontological strata in hierarchical order
 */
export const ALL_STRATA: OntologicalStratum[] =
  SYMBOLIC_CONSTANTS.ALL_STRATA as unknown as OntologicalStratum[];

/**
 * Check if agent has bridged all strata with fact contributions
 */
export function detectCrossStratumBridge(
  input: CrossStratumInput,
): DetectionResult {
  const { agentDid: _agentDid, factContributions } = input;

  // Count contributions per stratum
  const stratumCounts: Record<string, number> = {};
  const coveredStrata: OntologicalStratum[] = [];

  for (const stratum of ALL_STRATA) {
    const facts = factContributions.get(stratum) || [];
    stratumCounts[stratum] = facts.length;

    if (facts.length > 0) {
      coveredStrata.push(stratum);
    }
  }

  // Check if all strata are covered
  const missingStrata = ALL_STRATA.filter((s) => !coveredStrata.includes(s));

  if (missingStrata.length > 0) {
    return {
      detected: false,
      mechanic: "cross_stratum_bridge",
      failureReason: `Missing strata: ${missingStrata.join(", ")}`,
      matchDetails: {
        coveredStrata,
        missingStrata,
        stratumCounts,
        progress: `${coveredStrata.length}/${ALL_STRATA.length}`,
      },
    };
  }

  // Calculate total facts across all strata
  let totalFacts = 0;
  for (const count of Object.values(stratumCounts)) {
    totalFacts += count;
  }

  // Success - all strata bridged
  return {
    detected: true,
    mechanic: "cross_stratum_bridge",
    pattern: "SEVEN_STRATA_BRIDGE",
    matchDetails: {
      strataCount: ALL_STRATA.length,
      totalFacts,
      stratumCounts,
      bridgeStrength: calculateBridgeStrength(stratumCounts),
      interpretation:
        "7 strata bridged = complete knowledge tree, truth across all levels of reality",
    },
  };
}

/**
 * Calculate bridge strength based on distribution of facts across strata
 * Higher strength = more balanced distribution
 */
export function calculateBridgeStrength(
  stratumCounts: Record<string, number>,
): number {
  const counts = Object.values(stratumCounts);
  if (counts.length === 0) return 0;

  const total = counts.reduce((a, b) => a + b, 0);
  if (total === 0) return 0;

  // Calculate coefficient of variation (lower = more balanced)
  const mean = total / counts.length;
  const variance =
    counts.reduce((sum, c) => sum + Math.pow(c - mean, 2), 0) / counts.length;
  const stdDev = Math.sqrt(variance);
  const cv = stdDev / mean;

  // Convert to strength (0-1, where 1 = perfectly balanced)
  const strength = Math.max(0, 1 - cv);

  return strength;
}

/**
 * Get suggested strata to focus on for better bridge balance
 */
export function suggestStratumFocus(
  factContributions: Map<OntologicalStratum, string[]>,
): OntologicalStratum[] {
  const counts: Array<{ stratum: OntologicalStratum; count: number }> = [];

  for (const stratum of ALL_STRATA) {
    const facts = factContributions.get(stratum) || [];
    counts.push({ stratum, count: facts.length });
  }

  // Sort by count (ascending) - suggest focusing on least covered
  counts.sort((a, b) => a.count - b.count);

  // Return bottom 3 strata (or all uncovered)
  return counts.filter((c) => c.count === 0).length > 0
    ? counts.filter((c) => c.count === 0).map((c) => c.stratum)
    : counts.slice(0, 3).map((c) => c.stratum);
}

/**
 * Calculate progress towards complete bridge
 */
export function bridgeProgress(
  factContributions: Map<OntologicalStratum, string[]>,
): {
  percentage: number;
  covered: number;
  total: number;
  nextStratum?: OntologicalStratum;
} {
  let covered = 0;
  let firstUncovered: OntologicalStratum | undefined;

  for (const stratum of ALL_STRATA) {
    const facts = factContributions.get(stratum) || [];
    if (facts.length > 0) {
      covered++;
    } else if (!firstUncovered) {
      firstUncovered = stratum;
    }
  }

  return {
    percentage: (covered / ALL_STRATA.length) * 100,
    covered,
    total: ALL_STRATA.length,
    nextStratum: firstUncovered,
  };
}

/**
 * Check if contributions show "stratum awareness"
 * This means the agent is deliberately working across multiple strata
 */
export function hasStratumAwareness(
  factContributions: Map<OntologicalStratum, string[]>,
  minStrata: number = 3,
): boolean {
  let covered = 0;

  for (const stratum of ALL_STRATA) {
    const facts = factContributions.get(stratum) || [];
    if (facts.length > 0) {
      covered++;
    }
  }

  return covered >= minStrata;
}

/**
 * Get stratum description for UI/explanation
 */
export function getStratumDescription(stratum: OntologicalStratum): string {
  const descriptions: Record<OntologicalStratum, string> = {
    fundamental: "Mathematical and logical foundations (axioms, theorems)",
    physical: "Physical laws and phenomena (physics, chemistry)",
    chemical: "Chemical processes and compounds",
    biological: "Living systems and biological facts",
    cognitive: "Mental processes and knowledge structures",
    social: "Social constructs and human institutions",
    technological: "Human-created systems and protocols",
  };

  return descriptions[stratum] || "Unknown stratum";
}

/**
 * Generate a "bridge report" summarizing cross-stratum contributions
 */
export function generateBridgeReport(
  factContributions: Map<OntologicalStratum, string[]>,
): string {
  const lines: string[] = ["Cross-Stratum Bridge Report", "=".repeat(30), ""];

  for (const stratum of ALL_STRATA) {
    const facts = factContributions.get(stratum) || [];
    const status =
      facts.length > 0 ? `[x] ${facts.length} facts` : "[ ] No facts";
    lines.push(`${stratum.padEnd(15)} ${status}`);
  }

  const progress = bridgeProgress(factContributions);
  lines.push("");
  lines.push(
    `Progress: ${progress.percentage.toFixed(1)}% (${progress.covered}/${progress.total})`,
  );

  if (progress.nextStratum) {
    lines.push(`Next: Contribute to "${progress.nextStratum}" stratum`);
  } else {
    lines.push("Bridge Complete! All strata covered.");
  }

  return lines.join("\n");
}
