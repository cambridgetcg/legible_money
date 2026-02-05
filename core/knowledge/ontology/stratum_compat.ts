/**
 * Stratum Compatibility Layer
 *
 * Provides backward-compatible exports that bridge the old hardcoded constants
 * to the new registry-based approach. Existing code continues to work unchanged.
 *
 * MIGRATION GUIDE:
 *
 * Old way (still works):
 * ```typescript
 * import { STRATUM_HIERARCHY, OntologicalStratum } from './types.js';
 * const level = STRATUM_HIERARCHY['physical']; // 1
 * ```
 *
 * New way (preferred):
 * ```typescript
 * import { defaultRegistry } from './stratum_registry.js';
 * const level = defaultRegistry.getLevel('physical'); // 1
 * ```
 *
 * With dependency injection (best for testability):
 * ```typescript
 * import { OntologyProvider } from './stratum_registry.js';
 *
 * class MyService {
 *   constructor(private ontology: OntologyProvider) {}
 *
 *   doSomething(stratum: string) {
 *     const level = this.ontology.getLevel(stratum);
 *   }
 * }
 * ```
 *
 * @module core/knowledge/ontology/stratum_compat
 */

import type { LogicZone } from "../../logic/types.js";
import type { EpistemicCategory } from "../types.js";
import {
  defaultRegistry,
  type LegacyOntologicalStratum,
} from "./stratum_registry.js";

// ============================================================================
// Legacy Type Export
// ============================================================================

/**
 * @deprecated Use string IDs with registry.has() validation instead.
 *
 * This union type is maintained for backward compatibility.
 * New code should use string IDs and validate via the registry.
 */
export type OntologicalStratum = LegacyOntologicalStratum;

// ============================================================================
// Legacy Constant Exports (Generated from Registry)
// ============================================================================

/**
 * @deprecated Use defaultRegistry.getLevel() instead.
 *
 * Numeric hierarchy of strata (lower = more fundamental).
 * This is now generated from the registry for consistency.
 */
export const STRATUM_HIERARCHY: Record<OntologicalStratum, number> =
  defaultRegistry.toLegacyHierarchy() as Record<OntologicalStratum, number>;

/**
 * @deprecated Use defaultRegistry.getConfidenceCeiling() instead.
 *
 * Maximum confidence achievable for facts in each stratum.
 */
export const STRATUM_CONFIDENCE_CEILING: Record<OntologicalStratum, number> =
  defaultRegistry.toLegacyConfidenceCeiling() as Record<
    OntologicalStratum,
    number
  >;

/**
 * @deprecated Use defaultRegistry.get(id)?.defaultLogicZone instead.
 *
 * Default logic zone for each stratum.
 */
export const STRATUM_LOGIC_ZONE: Record<OntologicalStratum, LogicZone> =
  defaultRegistry.toLegacyLogicZone() as Record<OntologicalStratum, LogicZone>;

/**
 * @deprecated Use defaultRegistry.get(id)?.defaultEpistemicCategory instead.
 *
 * Default epistemic category for each stratum.
 */
export const STRATUM_EPISTEMIC_CATEGORY: Record<
  OntologicalStratum,
  EpistemicCategory
> = defaultRegistry.toLegacyEpistemicCategory() as Record<
  OntologicalStratum,
  EpistemicCategory
>;

// ============================================================================
// Legacy Function Wrappers
// ============================================================================

/**
 * @deprecated Use defaultRegistry.getLevel() instead.
 */
export function getStratumLevel(stratum: OntologicalStratum): number {
  return defaultRegistry.getLevel(stratum);
}

/**
 * @deprecated Use defaultRegistry.isHigher() instead.
 */
export function isHigherStratum(
  a: OntologicalStratum,
  b: OntologicalStratum,
): boolean {
  return defaultRegistry.isHigher(a, b);
}

/**
 * @deprecated Use defaultRegistry.isLower() instead.
 */
export function isLowerStratum(
  a: OntologicalStratum,
  b: OntologicalStratum,
): boolean {
  return defaultRegistry.isLower(a, b);
}

/**
 * @deprecated Use defaultRegistry.getDistance() instead.
 */
export function getStratumDistance(
  a: OntologicalStratum,
  b: OntologicalStratum,
): number {
  return defaultRegistry.getDistance(a, b);
}

/**
 * @deprecated Use defaultRegistry.getConfidenceCeiling() instead.
 */
export function getStratumConfidenceCeiling(
  stratum: OntologicalStratum,
): number {
  return defaultRegistry.getConfidenceCeiling(stratum);
}

/**
 * @deprecated Use defaultRegistry.getValidDependencies() instead.
 */
export function getValidDependencies(
  stratum: OntologicalStratum,
): readonly OntologicalStratum[] {
  return defaultRegistry.getValidDependencies(stratum) as OntologicalStratum[];
}

/**
 * @deprecated Use defaultRegistry.isValidDependency() instead.
 */
export function isValidStratumDependency(
  fromStratum: OntologicalStratum,
  toStratum: OntologicalStratum,
): boolean {
  return defaultRegistry.isValidDependency(fromStratum, toStratum);
}

/**
 * @deprecated Use defaultRegistry.getAtOrBelow() instead.
 */
export function getStrataBelowOrEqual(
  stratum: OntologicalStratum,
): OntologicalStratum[] {
  return defaultRegistry
    .getAtOrBelow(stratum)
    .map((s) => s.id) as OntologicalStratum[];
}

/**
 * @deprecated Use defaultRegistry.getAtOrAbove() instead.
 */
export function getStrataAboveOrEqual(
  stratum: OntologicalStratum,
): OntologicalStratum[] {
  return defaultRegistry
    .getAtOrAbove(stratum)
    .map((s) => s.id) as OntologicalStratum[];
}

/**
 * @deprecated Use defaultRegistry.getAllIds() instead.
 */
export function getStrataInOrder(): OntologicalStratum[] {
  return defaultRegistry.getAllIds() as OntologicalStratum[];
}

/**
 * @deprecated Use defaultRegistry.has() and get() instead.
 */
export function parseStratum(str: string): OntologicalStratum | null {
  const normalized = str.toLowerCase().trim();
  if (defaultRegistry.has(normalized)) {
    return normalized as OntologicalStratum;
  }
  return null;
}

// ============================================================================
// Legacy Constants (Generated from Registry)
// ============================================================================

/**
 * @deprecated Use defaultRegistry.toLegacyValidDependencies() or getValidDependencies() instead.
 */
export const VALID_DEPENDENCIES: ReadonlyMap<
  OntologicalStratum,
  readonly OntologicalStratum[]
> = defaultRegistry.toLegacyValidDependencies() as ReadonlyMap<
  OntologicalStratum,
  readonly OntologicalStratum[]
>;

/**
 * @deprecated Use defaultRegistry.get(id)?.description instead.
 */
export const STRATUM_DESCRIPTIONS: Record<OntologicalStratum, string> = (() => {
  const result: Record<string, string> = {};
  for (const stratum of defaultRegistry.getAll()) {
    result[stratum.id] = stratum.description;
  }
  return result as Record<OntologicalStratum, string>;
})();

/**
 * @deprecated Use defaultRegistry.get(id)?.exampleDomains instead.
 */
export const STRATUM_EXAMPLE_DOMAINS: Record<OntologicalStratum, string[]> =
  (() => {
    const result: Record<string, string[]> = {};
    for (const stratum of defaultRegistry.getAll()) {
      result[stratum.id] = [...stratum.exampleDomains];
    }
    return result as Record<OntologicalStratum, string[]>;
  })();

// ============================================================================
// Type Guards
// ============================================================================

/**
 * Check if a value is a valid stratum ID.
 */
export function isOntologicalStratum(
  value: unknown,
): value is OntologicalStratum {
  return typeof value === "string" && defaultRegistry.has(value);
}

/**
 * Assert that a value is a valid stratum ID.
 * @throws Error if invalid
 */
export function assertOntologicalStratum(
  value: unknown,
): asserts value is OntologicalStratum {
  if (!isOntologicalStratum(value)) {
    throw new Error(
      `Invalid stratum: ${value}. Valid strata: ${defaultRegistry.getAllIds().join(", ")}`,
    );
  }
}

// ============================================================================
// Utility Functions
// ============================================================================

/**
 * Get the minimum confidence ceiling across multiple strata.
 *
 * @deprecated Use registry methods directly.
 */
export function getMinConfidenceCeiling(strata: OntologicalStratum[]): number {
  if (strata.length === 0) return 1.0;
  return Math.min(
    ...strata.map((s) => defaultRegistry.getConfidenceCeiling(s)),
  );
}

/**
 * Apply stratum confidence ceiling to a value.
 *
 * @deprecated Use registry methods directly.
 */
export function applyStratumCeiling(
  confidence: number,
  stratum: OntologicalStratum,
): number {
  const ceiling = defaultRegistry.getConfidenceCeiling(stratum);
  return Math.min(confidence, ceiling);
}

/**
 * Validate all dependencies for a fact.
 *
 * @deprecated Use registry.isValidDependency() in a loop instead.
 */
export function validateDependencies(
  primaryStratum: OntologicalStratum,
  dependencies: OntologicalStratum[],
): OntologicalStratum[] {
  const invalid: OntologicalStratum[] = [];

  for (const dep of dependencies) {
    if (!defaultRegistry.isValidDependency(primaryStratum, dep)) {
      invalid.push(dep);
    }
  }

  return invalid;
}

// ============================================================================
// Re-export Registry for Gradual Migration
// ============================================================================

export {
  defaultRegistry,
  defaultOntologyProvider,
  StratumRegistry,
  createOntologyProvider,
  type StratumDefinition,
  type OntologyProvider,
  type SerializedStratumRegistry,
} from "./stratum_registry.js";
