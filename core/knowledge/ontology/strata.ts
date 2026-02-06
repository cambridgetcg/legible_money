/**
 * Ontological Strata - Hierarchy and Dependency Rules
 *
 * Defines the valid dependencies between ontological strata and provides
 * validation functions for cross-stratum references.
 */

import {
  type OntologicalStratum,
  type StratumDependency,
  type StratumDependencyType,
  STRATUM_HIERARCHY,
  STRATUM_CONFIDENCE_CEILING,
} from "./types.js";
import { defaultRegistry } from "./stratum_registry.js";

// ============================================================================
// Valid Stratum Dependencies
// ============================================================================

/**
 * Valid dependencies for each stratum.
 *
 * Higher strata can only depend on lower strata:
 * - fundamental: No dependencies (axiomatic)
 * - physical: Can depend on fundamental
 * - chemical: Can depend on fundamental, physical
 * - biological: Can depend on fundamental, physical, chemical
 * - cognitive: Can depend on biological (and implicitly all below)
 * - social: Can depend on cognitive (and implicitly all below)
 * - technological: Special - can depend on any stratum (designed systems)
 */
export const VALID_DEPENDENCIES: ReadonlyMap<
  OntologicalStratum,
  readonly OntologicalStratum[]
> = new Map([
  ["fundamental", []],
  ["physical", ["fundamental"]],
  ["chemical", ["fundamental", "physical"]],
  ["biological", ["fundamental", "physical", "chemical"]],
  ["cognitive", ["biological", "chemical", "physical", "fundamental"]],
  [
    "social",
    ["cognitive", "biological", "chemical", "physical", "fundamental"],
  ],
  [
    "technological",
    [
      "fundamental",
      "physical",
      "chemical",
      "biological",
      "cognitive",
      "social",
    ],
  ],
]);

/**
 * Get all strata that a given stratum can depend on.
 */
export function getValidDependencies(
  stratum: OntologicalStratum,
): readonly OntologicalStratum[] {
  return VALID_DEPENDENCIES.get(stratum) ?? [];
}

/**
 * Check if a stratum dependency is valid.
 *
 * Checks the static VALID_DEPENDENCIES map first (covers genesis strata),
 * then falls back to the registry for governance-registered strata that
 * may have canReferenceAll=true.
 *
 * @param fromStratum - The stratum of the dependent fact
 * @param toStratum - The stratum being depended upon
 * @returns true if the dependency is valid
 */
export function isValidStratumDependency(
  fromStratum: OntologicalStratum,
  toStratum: OntologicalStratum,
): boolean {
  // Same stratum is always valid
  if (fromStratum === toStratum) return true;

  // Check against static valid dependencies (covers all genesis strata)
  const validDeps = VALID_DEPENDENCIES.get(fromStratum);
  if (validDeps?.includes(toStratum)) return true;

  // Fallback: check registry for governance-registered strata with canReferenceAll
  const registryDef = defaultRegistry.get(fromStratum);
  if (registryDef?.canReferenceAll) return true;

  return false;
}

/**
 * Validate all dependencies for a fact.
 *
 * @param primaryStratum - The primary stratum of the fact
 * @param dependencies - The strata this fact depends on
 * @returns Array of invalid dependencies (empty if all valid)
 */
export function validateDependencies(
  primaryStratum: OntologicalStratum,
  dependencies: OntologicalStratum[],
): OntologicalStratum[] {
  const invalid: OntologicalStratum[] = [];

  for (const dep of dependencies) {
    if (!isValidStratumDependency(primaryStratum, dep)) {
      invalid.push(dep);
    }
  }

  return invalid;
}

// ============================================================================
// Stratum Level Comparisons
// ============================================================================

/**
 * Get the numeric level of a stratum.
 */
export function getStratumLevel(stratum: OntologicalStratum): number {
  return STRATUM_HIERARCHY[stratum];
}

/**
 * Check if one stratum is higher (more emergent) than another.
 */
export function isHigherStratum(
  a: OntologicalStratum,
  b: OntologicalStratum,
): boolean {
  return STRATUM_HIERARCHY[a] > STRATUM_HIERARCHY[b];
}

/**
 * Check if one stratum is lower (more fundamental) than another.
 */
export function isLowerStratum(
  a: OntologicalStratum,
  b: OntologicalStratum,
): boolean {
  return STRATUM_HIERARCHY[a] < STRATUM_HIERARCHY[b];
}

/**
 * Get the difference in levels between two strata.
 */
export function getStratumDistance(
  a: OntologicalStratum,
  b: OntologicalStratum,
): number {
  return Math.abs(STRATUM_HIERARCHY[a] - STRATUM_HIERARCHY[b]);
}

/**
 * Get all strata at or below a given stratum.
 */
export function getStrataBelowOrEqual(
  stratum: OntologicalStratum,
): OntologicalStratum[] {
  const level = STRATUM_HIERARCHY[stratum];
  return (Object.keys(STRATUM_HIERARCHY) as OntologicalStratum[]).filter(
    (s) => STRATUM_HIERARCHY[s] <= level,
  );
}

/**
 * Get all strata at or above a given stratum.
 */
export function getStrataAboveOrEqual(
  stratum: OntologicalStratum,
): OntologicalStratum[] {
  const level = STRATUM_HIERARCHY[stratum];
  return (Object.keys(STRATUM_HIERARCHY) as OntologicalStratum[]).filter(
    (s) => STRATUM_HIERARCHY[s] >= level,
  );
}

// ============================================================================
// Confidence Ceilings
// ============================================================================

/**
 * Get the confidence ceiling for a stratum.
 */
export function getStratumConfidenceCeiling(
  stratum: OntologicalStratum,
): number {
  return STRATUM_CONFIDENCE_CEILING[stratum];
}

/**
 * Get the minimum confidence ceiling across multiple strata.
 */
export function getMinConfidenceCeiling(strata: OntologicalStratum[]): number {
  if (strata.length === 0) return 1.0;

  return Math.min(...strata.map((s) => STRATUM_CONFIDENCE_CEILING[s]));
}

/**
 * Apply confidence ceiling based on stratum.
 */
export function applyStratumCeiling(
  confidence: number,
  stratum: OntologicalStratum,
): number {
  const ceiling = STRATUM_CONFIDENCE_CEILING[stratum];
  return Math.min(confidence, ceiling);
}

// ============================================================================
// Dependency Strength
// ============================================================================

/**
 * Default strength values for different dependency types.
 */
export const DEPENDENCY_TYPE_STRENGTH: Record<StratumDependencyType, number> = {
  emergence: 0.9, // Emergent behavior - strong dependency
  composition: 0.95, // Made of lower-level parts - very strong
  constraint: 0.8, // Limited by lower laws - moderate
  application: 0.7, // Uses as tool - weaker dependency
};

/**
 * Calculate the cross-stratum discount for a set of dependencies.
 *
 * Each stratum boundary crossed reduces confidence by 5%.
 *
 * @param primaryStratum - The stratum of the fact
 * @param dependencies - The dependencies to consider
 * @returns Discount multiplier (0-1)
 */
export function calculateCrossStratumDiscount(
  primaryStratum: OntologicalStratum,
  dependencies: StratumDependency[],
): number {
  if (dependencies.length === 0) return 1.0;

  let totalDiscount = 1.0;
  const discountPerBoundary = 0.95; // 5% reduction per stratum crossed

  for (const dep of dependencies) {
    const distance = getStratumDistance(primaryStratum, dep.primaryStratum);
    if (distance > 0) {
      const boundaryDiscount = Math.pow(discountPerBoundary, distance);
      // Weight by dependency strength
      const weightedDiscount =
        1 - (1 - boundaryDiscount) * dep.dependencyStrength;
      totalDiscount *= weightedDiscount;
    }
  }

  return totalDiscount;
}

// ============================================================================
// Stratum Metadata
// ============================================================================

/**
 * Human-readable descriptions of each stratum.
 */
export const STRATUM_DESCRIPTIONS: Record<OntologicalStratum, string> = {
  fundamental:
    "Physical constants, elementary particles, fundamental forces, and mathematical axioms",
  physical:
    "Classical and quantum mechanics, thermodynamics, electromagnetism, relativity",
  chemical:
    "Atomic structure, molecular bonds, chemical reactions, materials science",
  biological: "Cells, organisms, genetics, evolution, ecology, biochemistry",
  cognitive: "Perception, learning, reasoning, language, consciousness",
  social: "Economics, politics, culture, law, history",
  technological: "Computation, networks, cryptography, AI, blockchain",
};

/**
 * Example domains for each stratum.
 */
export const STRATUM_EXAMPLE_DOMAINS: Record<OntologicalStratum, string[]> = {
  fundamental: ["physics.particles", "physics.forces", "mathematics.logic"],
  physical: ["physics.mechanics", "physics.thermodynamics", "physics.quantum"],
  chemical: ["chemistry.atomic", "chemistry.molecular", "chemistry.organic"],
  biological: ["biology.cellular", "biology.genetics", "biology.ecology"],
  cognitive: [
    "cognition.perception",
    "cognition.reasoning",
    "cognition.language",
  ],
  social: ["social.economics", "social.political", "social.legal"],
  technological: [
    "technology.computation",
    "technology.cryptography",
    "technology.blockchain",
  ],
};

/**
 * Get all strata in order from most fundamental to most emergent.
 */
export function getStrataInOrder(): OntologicalStratum[] {
  return [
    "fundamental",
    "physical",
    "chemical",
    "biological",
    "cognitive",
    "social",
    "technological",
  ];
}

/**
 * Parse a stratum from a string.
 */
export function parseStratum(str: string): OntologicalStratum | null {
  const normalized = str.toLowerCase().trim();
  if (normalized in STRATUM_HIERARCHY) {
    return normalized as OntologicalStratum;
  }
  return null;
}
