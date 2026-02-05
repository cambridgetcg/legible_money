/**
 * PP-Aligned Verification Gates
 *
 * Quality gates based on the Purpose Prompter hierarchy for verifying
 * agent outputs before they can be submitted to the knowledge layer.
 *
 * The five gates correspond to the PP dimensions:
 *   Gate 1 (TRUTH): Is output grounded in verified facts?
 *   Gate 2 (UNDERSTANDING): Does reasoning chain cohere?
 *   Gate 3 (BEAUTY): Is output elegant and well-structured?
 *   Gate 4 (JUSTICE): Is output placed correctly in the domain hierarchy?
 *   Gate 5 (CREATIVITY): Does output extend knowledge appropriately?
 *
 * PP HIERARCHY:
 *   TRUTH: This file enforces truth-grounding requirements
 *   UNDERSTANDING: Gates assess comprehension quality
 *   BEAUTY: Clean gate structure, single responsibility per gate
 *   JUSTICE: Placed alongside philosophy.ts and output_verification.ts
 *   CREATIVITY: Novel gate system for agent output verification
 */

import type { OutputVerificationResult } from "./output_verification.js";
import type { AntiHallucinationConfig } from "./output_verification.js";
import { DEFAULT_ANTI_HALLUCINATION_CONFIG } from "./output_verification.js";

// ============================================================================
// Gate Result Types
// ============================================================================

/**
 * Result of running a single PP gate.
 */
export interface PPGateResult {
  /** Which gate was run */
  gate: PPGateName;

  /** Whether the gate passed */
  passed: boolean;

  /** Score for this gate (0-1) */
  score: number;

  /** Human-readable details about the gate result */
  details: string;

  /** Suggestions for passing if failed */
  suggestions?: string[];
}

/**
 * Names of the five PP gates.
 */
export type PPGateName =
  | "truth"
  | "understanding"
  | "beauty"
  | "justice"
  | "creativity";

/**
 * Result of running all PP gates.
 */
export interface PPGatesResult {
  /** All gate results */
  gates: PPGateResult[];

  /** Overall pass (all gates passed) */
  allPassed: boolean;

  /** Number of gates passed */
  passedCount: number;

  /** Number of gates failed */
  failedCount: number;

  /** Summary message */
  summary: string;

  /** Aggregated suggestions from failed gates */
  suggestions: string[];
}

// ============================================================================
// Gate Configuration
// ============================================================================

/**
 * Thresholds for each gate.
 */
export interface PPGateThresholds {
  /** Minimum groundedness score for Truth gate */
  truthGroundedness: number;

  /** Minimum grounding ratio for Understanding gate */
  understandingGroundingRatio: number;

  /** Minimum understanding score */
  understandingMinScore: number;

  /** Maximum strata crossed for Beauty gate */
  beautyMaxStrata: number;

  /** Minimum beauty score */
  beautyMinScore: number;

  /** Whether domain must match for Justice gate */
  justiceRequireDomainMatch: boolean;

  /** Minimum justice score */
  justiceMinScore: number;

  /** Minimum groundedness for creative speculation */
  creativityMinGroundednessForSpeculation: number;
}

/**
 * Default gate thresholds.
 */
export const DEFAULT_GATE_THRESHOLDS: PPGateThresholds = {
  truthGroundedness: 0.5,
  understandingGroundingRatio: 0.3,
  understandingMinScore: 0.4,
  beautyMaxStrata: 2,
  beautyMinScore: 0.5,
  justiceRequireDomainMatch: false, // Warning only, not required
  justiceMinScore: 0.6,
  creativityMinGroundednessForSpeculation: 0.3,
};

// ============================================================================
// Main Gate Runner
// ============================================================================

/**
 * Run all PP gates on a verification result.
 */
export function runPPGates(
  verificationResult: OutputVerificationResult,
  thresholds: PPGateThresholds = DEFAULT_GATE_THRESHOLDS,
): PPGatesResult {
  const gates: PPGateResult[] = [
    runTruthGate(verificationResult, thresholds),
    runUnderstandingGate(verificationResult, thresholds),
    runBeautyGate(verificationResult, thresholds),
    runJusticeGate(verificationResult, thresholds),
    runCreativityGate(verificationResult, thresholds),
  ];

  const passedCount = gates.filter((g) => g.passed).length;
  const failedCount = gates.length - passedCount;
  const allPassed = failedCount === 0;

  // Aggregate suggestions from failed gates
  const suggestions = gates
    .filter((g) => !g.passed && g.suggestions)
    .flatMap((g) => g.suggestions!);

  // Build summary
  let summary: string;
  if (allPassed) {
    summary =
      "All PP gates passed - output is well-grounded and properly placed";
  } else if (passedCount >= 3) {
    summary = `${passedCount}/5 gates passed - minor improvements needed`;
  } else {
    summary = `${passedCount}/5 gates passed - significant grounding issues detected`;
  }

  return {
    gates,
    allPassed,
    passedCount,
    failedCount,
    summary,
    suggestions,
  };
}

// ============================================================================
// Individual Gates
// ============================================================================

/**
 * Gate 1: TRUTH
 *
 * Is the output grounded in verified facts?
 */
function runTruthGate(
  result: OutputVerificationResult,
  thresholds: PPGateThresholds,
): PPGateResult {
  const score = result.ppScores.truth;
  const groundednessOk =
    result.groundednessScore >= thresholds.truthGroundedness;
  const passed = groundednessOk && score >= thresholds.truthGroundedness;

  const validCitations = result.citations.filter(
    (c) => c.relationship !== "contradicts",
  );

  let details: string;
  if (passed) {
    details = `Grounded in ${validCitations.length} verified fact(s) with ${(result.groundednessScore * 100).toFixed(0)}% groundedness`;
  } else if (validCitations.length === 0) {
    details = "No valid citations provided - output lacks factual grounding";
  } else {
    details = `Groundedness ${(result.groundednessScore * 100).toFixed(0)}% below minimum ${(thresholds.truthGroundedness * 100).toFixed(0)}%`;
  }

  return {
    gate: "truth",
    passed,
    score,
    details,
    suggestions: passed
      ? undefined
      : [
          "Query knowledge layer for supporting facts",
          "Cite more directly relevant verified facts",
          "Reduce confidence to match evidence strength",
        ],
  };
}

/**
 * Gate 2: UNDERSTANDING
 *
 * Does the reasoning chain cohere? Is there genuine comprehension?
 */
function runUnderstandingGate(
  result: OutputVerificationResult,
  thresholds: PPGateThresholds,
): PPGateResult {
  const analysis = result.reasoningChainAnalysis;
  const score = result.ppScores.understanding;

  const groundingRatioOk =
    analysis.groundingRatio >= thresholds.understandingGroundingRatio;
  const scoreOk = score >= thresholds.understandingMinScore;
  const driftOk = analysis.driftScore <= 1.0;

  const passed = groundingRatioOk && scoreOk && driftOk;

  let details: string;
  if (passed) {
    details = `Reasoning chain is ${(analysis.groundingRatio * 100).toFixed(0)}% grounded with coherent derivation`;
  } else if (!groundingRatioOk) {
    details = `Only ${(analysis.groundingRatio * 100).toFixed(0)}% of reasoning is grounded (need ${(thresholds.understandingGroundingRatio * 100).toFixed(0)}%)`;
  } else if (!driftOk) {
    details = `Reasoning drift score ${analysis.driftScore.toFixed(2)} indicates loss of grounding`;
  } else {
    details = `Understanding score ${score.toFixed(2)} below minimum ${thresholds.understandingMinScore}`;
  }

  return {
    gate: "understanding",
    passed,
    score,
    details,
    suggestions: passed
      ? undefined
      : [
          "Re-anchor reasoning with fact queries",
          "Reduce inference chain length",
          "Add intermediate grounded steps",
        ],
  };
}

/**
 * Gate 3: BEAUTY
 *
 * Is the output elegant and well-structured?
 * Checks stratum consistency and coherence.
 */
function runBeautyGate(
  result: OutputVerificationResult,
  thresholds: PPGateThresholds,
): PPGateResult {
  const analysis = result.reasoningChainAnalysis;
  const score = result.ppScores.beauty;

  const strataCount = analysis.strataTraversed.length;
  const strataOk = strataCount <= thresholds.beautyMaxStrata;
  const scoreOk = score >= thresholds.beautyMinScore;

  const passed = strataOk && scoreOk;

  let details: string;
  if (passed) {
    if (strataCount <= 1) {
      details = "Output is coherent within a single ontological stratum";
    } else {
      details = `Output coherently bridges ${strataCount} strata`;
    }
  } else if (!strataOk) {
    details = `Output crosses ${strataCount} strata (max ${thresholds.beautyMaxStrata}) - consider using gateway domains`;
  } else {
    details = `Beauty score ${score.toFixed(2)} indicates structural issues`;
  }

  return {
    gate: "beauty",
    passed,
    score,
    details,
    suggestions: passed
      ? undefined
      : [
          "Use gateway domains to bridge strata",
          "Simplify reasoning to stay within fewer strata",
          "Break claim into stratum-specific sub-claims",
        ],
  };
}

/**
 * Gate 4: JUSTICE
 *
 * Is the output placed correctly in the domain hierarchy?
 */
function runJusticeGate(
  result: OutputVerificationResult,
  thresholds: PPGateThresholds,
): PPGateResult {
  const score = result.ppScores.justice;

  const validCitations = result.citations.filter(
    (c) => c.relationship !== "contradicts",
  );
  const hasDomainMatch =
    validCitations.length === 0 ||
    validCitations.some(
      (c) => c.factDomain === result.citations[0]?.factDomain,
    );

  const domainOk = !thresholds.justiceRequireDomainMatch || hasDomainMatch;
  const scoreOk = score >= thresholds.justiceMinScore;

  const passed = domainOk && scoreOk;

  let details: string;
  if (passed) {
    details = "Output is correctly placed in domain hierarchy";
  } else if (!hasDomainMatch) {
    details = "Citations span multiple domains - consider domain consistency";
  } else {
    details = `Justice score ${score.toFixed(2)} suggests placement issues`;
  }

  return {
    gate: "justice",
    passed,
    score,
    details,
    suggestions: passed
      ? undefined
      : [
          "Verify domain assignment matches cited facts",
          "Review ontological tree for correct placement",
          "Consider if claim belongs in a different domain",
        ],
  };
}

/**
 * Gate 5: CREATIVITY
 *
 * Does the output extend knowledge appropriately?
 * Penalizes ungrounded speculation, rewards grounded derivation.
 */
function runCreativityGate(
  result: OutputVerificationResult,
  thresholds: PPGateThresholds,
): PPGateResult {
  const analysis = result.reasoningChainAnalysis;
  const score = result.ppScores.creativity;

  const hasSpeculation = analysis.speculationNodeCount > 0;
  const speculationGrounded =
    !hasSpeculation ||
    result.groundednessScore >=
      thresholds.creativityMinGroundednessForSpeculation;

  // Creativity is valid if:
  // - No speculation present (pure derivation/assertion)
  // - OR speculation is grounded enough
  const passed = speculationGrounded;

  let details: string;
  if (passed) {
    if (hasSpeculation) {
      details = "Creative speculation is appropriately grounded";
    } else if (analysis.inferenceNodeCount > 0) {
      details = "Creative derivation from grounded facts";
    } else {
      details = "Grounded assertion (no creative extension)";
    }
  } else {
    details = `Speculation exceeds grounding (${(result.groundednessScore * 100).toFixed(0)}% < ${(thresholds.creativityMinGroundednessForSpeculation * 100).toFixed(0)}%)`;
  }

  return {
    gate: "creativity",
    passed,
    score,
    details,
    suggestions: passed
      ? undefined
      : [
          "Ground creative insights in verified facts first",
          "Mark ungrounded claims as speculation explicitly",
          "Reduce speculation distance from grounded knowledge",
        ],
  };
}

// ============================================================================
// Gate Utilities
// ============================================================================

/**
 * Check if a specific gate passed.
 */
export function isGatePassed(
  gatesResult: PPGatesResult,
  gateName: PPGateName,
): boolean {
  const gate = gatesResult.gates.find((g) => g.gate === gateName);
  return gate?.passed ?? false;
}

/**
 * Get gate result by name.
 */
export function getGateResult(
  gatesResult: PPGatesResult,
  gateName: PPGateName,
): PPGateResult | undefined {
  return gatesResult.gates.find((g) => g.gate === gateName);
}

/**
 * Get all failed gates.
 */
export function getFailedGates(gatesResult: PPGatesResult): PPGateResult[] {
  return gatesResult.gates.filter((g) => !g.passed);
}

/**
 * Quick check: does output pass minimum truth gate?
 */
export function passesMinimumTruth(
  verificationResult: OutputVerificationResult,
  config: AntiHallucinationConfig = DEFAULT_ANTI_HALLUCINATION_CONFIG,
): boolean {
  return verificationResult.groundednessScore >= config.minGroundednessScore;
}

/**
 * Format gate results for display.
 */
export function formatGateResults(gatesResult: PPGatesResult): string {
  const lines: string[] = [`PP Gates: ${gatesResult.passedCount}/5 passed`, ""];

  for (const gate of gatesResult.gates) {
    const status = gate.passed ? "✓" : "✗";
    const score = (gate.score * 100).toFixed(0);
    lines.push(
      `  ${status} ${gate.gate.toUpperCase()} (${score}%): ${gate.details}`,
    );
  }

  if (gatesResult.suggestions.length > 0) {
    lines.push("");
    lines.push("Suggestions:");
    for (const suggestion of gatesResult.suggestions.slice(0, 3)) {
      lines.push(`  - ${suggestion}`);
    }
  }

  return lines.join("\n");
}
