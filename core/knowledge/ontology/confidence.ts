/**
 * Ontological Confidence - Stratum-Aware Confidence Calculation
 *
 * Calculates effective confidence considering:
 * 1. Stratum confidence ceilings
 * 2. Cross-stratum dependency discounts
 * 3. Domain hierarchy inheritance
 */

import {
  type OntologicalDomain,
  type StratumDependency,
  type OntologicalConfidenceResult,
  type DomainTree,
  STRATUM_CONFIDENCE_CEILING,
} from "./types.js";
import {
  calculateCrossStratumDiscount,
  getStratumConfidenceCeiling,
} from "./strata.js";
import { getDomainPath } from "./tree.js";

// ============================================================================
// Configuration
// ============================================================================

/**
 * Configuration for ontological confidence calculation.
 */
export interface OntologicalConfidenceConfig {
  /** Discount per stratum boundary crossed (default: 0.95 = 5% reduction) */
  crossStratumDiscount: number;

  /** Whether to inherit ceilings from parent domains */
  inheritParentCeiling: boolean;

  /** Whether to apply cross-stratum discounts */
  applyCrossStratumDiscount: boolean;

  /** Minimum confidence floor (prevents confidence going to zero) */
  minimumConfidence: number;
}

export const DEFAULT_CONFIDENCE_CONFIG: OntologicalConfidenceConfig = {
  crossStratumDiscount: 0.95,
  inheritParentCeiling: true,
  applyCrossStratumDiscount: true,
  minimumConfidence: 0.01,
};

// ============================================================================
// Main Confidence Calculation
// ============================================================================

/**
 * Calculate ontological confidence for a fact.
 *
 * Applies:
 * 1. Stratum ceiling (facts can't exceed their stratum's max)
 * 2. Domain hierarchy ceiling (inherit strictest parent ceiling)
 * 3. Cross-stratum discount (5% per stratum boundary crossed)
 *
 * @param baseConfidence - The raw confidence before ontological adjustments
 * @param domain - The domain this fact belongs to
 * @param dependencies - Cross-stratum dependencies
 * @param domainTree - The domain tree (for hierarchy lookup)
 * @param config - Configuration options
 */
export function calculateOntologicalConfidence(
  baseConfidence: number,
  domain: OntologicalDomain,
  dependencies: StratumDependency[],
  domainTree?: DomainTree,
  config: OntologicalConfidenceConfig = DEFAULT_CONFIDENCE_CONFIG,
): OntologicalConfidenceResult {
  // 1. Get stratum ceiling
  const stratumCeiling = getStratumConfidenceCeiling(domain.stratum);

  // 2. Calculate inherited ceiling from domain hierarchy
  let inheritedCeiling = stratumCeiling;
  if (config.inheritParentCeiling && domainTree) {
    inheritedCeiling = calculateInheritedCeiling(domain, domainTree);
  }

  // 3. Calculate cross-stratum discount
  let stratumDiscount = 1.0;
  if (config.applyCrossStratumDiscount && dependencies.length > 0) {
    stratumDiscount = calculateCrossStratumDiscount(
      domain.stratum,
      dependencies,
    );
  }

  // 4. Calculate effective confidence
  let effectiveConfidence = baseConfidence * stratumDiscount;

  // Apply ceiling
  effectiveConfidence = Math.min(effectiveConfidence, inheritedCeiling);

  // Apply floor
  effectiveConfidence = Math.max(effectiveConfidence, config.minimumConfidence);

  return {
    baseConfidence,
    stratumCeiling,
    inheritedCeiling,
    stratumDiscount,
    effectiveConfidence,
    dependencies: dependencies.map((d) => ({
      stratum: d.primaryStratum,
      type: d.dependencyType,
      strength: d.dependencyStrength,
    })),
  };
}

/**
 * Calculate the inherited ceiling from domain hierarchy.
 *
 * Traverses from the domain up to root, taking the minimum ceiling
 * of all ancestors.
 */
function calculateInheritedCeiling(
  domain: OntologicalDomain,
  domainTree: DomainTree,
): number {
  const path = getDomainPath(domainTree, domain.id);

  let minCeiling = STRATUM_CONFIDENCE_CEILING[domain.stratum];

  for (const ancestor of path) {
    const ancestorCeiling = STRATUM_CONFIDENCE_CEILING[ancestor.stratum];
    minCeiling = Math.min(minCeiling, ancestorCeiling);
  }

  return minCeiling;
}

// ============================================================================
// Specialized Calculations
// ============================================================================

/**
 * Calculate confidence impact of adding a new dependency.
 */
export function calculateDependencyImpact(
  currentConfidence: number,
  domain: OntologicalDomain,
  newDependency: StratumDependency,
  existingDependencies: StratumDependency[],
): { newConfidence: number; impact: number; percentChange: number } {
  const allDependencies = [...existingDependencies, newDependency];

  const newResult = calculateOntologicalConfidence(
    currentConfidence,
    domain,
    allDependencies,
  );

  const oldResult = calculateOntologicalConfidence(
    currentConfidence,
    domain,
    existingDependencies,
  );

  const impact = newResult.effectiveConfidence - oldResult.effectiveConfidence;
  const percentChange =
    oldResult.effectiveConfidence > 0
      ? (impact / oldResult.effectiveConfidence) * 100
      : 0;

  return {
    newConfidence: newResult.effectiveConfidence,
    impact,
    percentChange,
  };
}

// ============================================================================
// Confidence Aggregation
// ============================================================================

/**
 * Aggregate confidences from multiple sources using stake-weighted average.
 *
 * Uses BigInt basis-point arithmetic to avoid Number() overflow on real
 * LGM stakes (1111 LGM = 1.111×10²¹ exceeds Number.MAX_SAFE_INTEGER).
 */
export function aggregateConfidences(
  contributions: Array<{ confidence: number; stake: bigint }>,
): number {
  if (contributions.length === 0) return 0;

  let weightedBps = 0n;
  let totalStake = 0n;

  for (const { confidence, stake } of contributions) {
    const confidenceBps = BigInt(Math.round(confidence * 10000));
    weightedBps += confidenceBps * stake;
    totalStake += stake;
  }

  if (totalStake === 0n) return 0;

  return Number(weightedBps / totalStake) / 10000;
}

/**
 * Apply ontological constraints to an aggregated confidence.
 */
export function constrainAggregatedConfidence(
  aggregatedConfidence: number,
  domain: OntologicalDomain,
  dependencies: StratumDependency[],
  domainTree?: DomainTree,
): OntologicalConfidenceResult {
  return calculateOntologicalConfidence(
    aggregatedConfidence,
    domain,
    dependencies,
    domainTree,
  );
}

// ============================================================================
// Confidence Bounds
// ============================================================================

/**
 * Get the theoretical maximum confidence for a domain.
 */
export function getMaximumConfidence(
  domain: OntologicalDomain,
  domainTree?: DomainTree,
): number {
  if (!domainTree) {
    return STRATUM_CONFIDENCE_CEILING[domain.stratum];
  }

  return calculateInheritedCeiling(domain, domainTree);
}

/**
 * Check if a confidence value is achievable in a domain.
 */
export function isConfidenceAchievable(
  targetConfidence: number,
  domain: OntologicalDomain,
  domainTree?: DomainTree,
): { achievable: boolean; maxAchievable: number; reason?: string } {
  const maxConfidence = getMaximumConfidence(domain, domainTree);

  if (targetConfidence <= maxConfidence) {
    return { achievable: true, maxAchievable: maxConfidence };
  }

  return {
    achievable: false,
    maxAchievable: maxConfidence,
    reason: `Stratum ceiling for ${domain.stratum} is ${maxConfidence}`,
  };
}

// ============================================================================
// Confidence Change Propagation
// ============================================================================

/**
 * Represents a confidence change that needs to be propagated.
 */
export interface ConfidenceChange {
  factId: string;
  domainId: string;
  oldConfidence: number;
  newConfidence: number;
  reason: "direct_update" | "dependency_update" | "stratum_change";
}

/**
 * Calculate cascading confidence changes from a single update.
 *
 * When a fact's confidence changes, dependent facts may also need updates.
 */
export function calculateCascadingChanges(
  initialChange: ConfidenceChange,
  getDependentFacts: (
    factId: string,
  ) => Array<{ factId: string; domainId: string; currentConfidence: number }>,
  getFactDependencies: (factId: string) => StratumDependency[],
  getDomain: (domainId: string) => OntologicalDomain | null,
  maxDepth: number = 10,
): ConfidenceChange[] {
  const changes: ConfidenceChange[] = [initialChange];
  const visited = new Set<string>([initialChange.factId]);
  const queue: Array<{ factId: string; depth: number }> = [
    { factId: initialChange.factId, depth: 0 },
  ];

  while (queue.length > 0) {
    const { factId, depth } = queue.shift()!;

    if (depth >= maxDepth) continue;

    const dependents = getDependentFacts(factId);

    for (const dependent of dependents) {
      if (visited.has(dependent.factId)) continue;
      visited.add(dependent.factId);

      const domain = getDomain(dependent.domainId);
      if (!domain) continue;

      const dependencies = getFactDependencies(dependent.factId);

      // Recalculate confidence
      const result = calculateOntologicalConfidence(
        dependent.currentConfidence,
        domain,
        dependencies,
      );

      // Check if confidence changed significantly (>0.1%)
      if (
        Math.abs(result.effectiveConfidence - dependent.currentConfidence) >
        0.001
      ) {
        changes.push({
          factId: dependent.factId,
          domainId: dependent.domainId,
          oldConfidence: dependent.currentConfidence,
          newConfidence: result.effectiveConfidence,
          reason: "dependency_update",
        });

        queue.push({ factId: dependent.factId, depth: depth + 1 });
      }
    }
  }

  return changes;
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Format a confidence value for display.
 */
export function formatConfidence(
  confidence: number,
  precision: number = 2,
): string {
  return (confidence * 100).toFixed(precision) + "%";
}

/**
 * Parse a confidence string (e.g., "95%") to a number.
 */
export function parseConfidence(str: string): number | null {
  const match = str.trim().match(/^(\d+(?:\.\d+)?)\s*%?$/);
  if (!match) return null;

  let value = parseFloat(match[1]);

  // If looks like a percentage (>1), convert to decimal
  if (value > 1) {
    value = value / 100;
  }

  // Clamp to valid range
  return Math.max(0, Math.min(1, value));
}
