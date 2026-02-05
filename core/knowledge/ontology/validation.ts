/**
 * Ontological Validation - Cross-Stratum Reference Rules
 *
 * Enforces ontological constraints on references between facts:
 * - supports/extends: Can only reference same or lower stratum
 * - specializes: Must stay in same stratum
 * - generalizes: Can only reference lower stratum
 * - contradicts: Any stratum (contradictions are universal)
 */

import {
  type OntologicalStratum,
  type OntologicalDomain,
  type OntologicalValidationResult,
  type DomainTree,
  type DomainRegistrationInput,
  STRATUM_CONFIDENCE_CEILING,
} from "./types.js";
import { isValidStratumDependency, getStratumLevel } from "./strata.js";
import { hasDomain, getDomain, isValidDomainId } from "./tree.js";

// ============================================================================
// Reference Types (from state/types.ts)
// ============================================================================

/**
 * Types of references between facts.
 */
export type ReferenceType =
  | "supports"
  | "contradicts"
  | "extends"
  | "specializes"
  | "generalizes";

// ============================================================================
// Reference Validation
// ============================================================================

/**
 * Validate that a reference respects ontological constraints.
 *
 * Rules:
 * - supports: Can only support facts in same or lower stratum
 * - extends: Can only extend facts in same or lower stratum
 * - specializes: Must be in the same stratum
 * - generalizes: Can only generalize to lower stratum
 * - contradicts: Any stratum (contradictions cross boundaries)
 */
export function validateOntologicalReference(
  fromDomain: OntologicalDomain,
  toDomain: OntologicalDomain,
  referenceType: ReferenceType,
): OntologicalValidationResult {
  const fromLevel = getStratumLevel(fromDomain.stratum);
  const toLevel = getStratumLevel(toDomain.stratum);

  // Contradictions can cross any stratum boundary
  if (referenceType === "contradicts") {
    return {
      valid: true,
      fromLevel,
      toLevel,
    };
  }

  // Supports and extends: can only reference same or lower stratum
  if (referenceType === "supports" || referenceType === "extends") {
    if (toLevel > fromLevel) {
      return {
        valid: false,
        reason: `Cannot ${referenceType} fact in higher stratum (${toDomain.stratum} level ${toLevel} > ${fromDomain.stratum} level ${fromLevel})`,
        fromLevel,
        toLevel,
      };
    }
    return { valid: true, fromLevel, toLevel };
  }

  // Specializes: must be in same stratum
  if (referenceType === "specializes") {
    if (toLevel !== fromLevel) {
      return {
        valid: false,
        reason: `Specialization must be within same stratum (${fromDomain.stratum} ≠ ${toDomain.stratum})`,
        fromLevel,
        toLevel,
      };
    }
    return { valid: true, fromLevel, toLevel };
  }

  // Generalizes: can only generalize to lower stratum
  if (referenceType === "generalizes") {
    if (toLevel > fromLevel) {
      return {
        valid: false,
        reason: `Cannot generalize to higher stratum (${toDomain.stratum} level ${toLevel} > ${fromDomain.stratum} level ${fromLevel})`,
        fromLevel,
        toLevel,
      };
    }
    return { valid: true, fromLevel, toLevel };
  }

  // Unknown reference type
  return {
    valid: false,
    reason: `Unknown reference type: ${referenceType}`,
    fromLevel,
    toLevel,
  };
}

/**
 * Validate a reference using domain IDs.
 */
export function validateReferenceByDomainId(
  fromDomainId: string,
  toDomainId: string,
  referenceType: ReferenceType,
  domainTree: DomainTree,
): OntologicalValidationResult {
  const fromDomain = getDomain(domainTree, fromDomainId);
  const toDomain = getDomain(domainTree, toDomainId);

  if (!fromDomain) {
    return {
      valid: false,
      reason: `Source domain not found: ${fromDomainId}`,
    };
  }

  if (!toDomain) {
    return {
      valid: false,
      reason: `Target domain not found: ${toDomainId}`,
    };
  }

  return validateOntologicalReference(fromDomain, toDomain, referenceType);
}

// ============================================================================
// Domain Registration Validation
// ============================================================================

/**
 * Validation result for domain registration.
 */
export interface DomainValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
}

/**
 * Validate a domain registration request.
 */
export function validateDomainRegistration(
  input: DomainRegistrationInput,
  domainTree: DomainTree,
): DomainValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];

  // 1. Validate ID format
  if (!isValidDomainId(input.id)) {
    errors.push(
      `Invalid domain ID format: "${input.id}". Must be lowercase letters, numbers, underscores, separated by dots.`,
    );
  }

  // 2. Check for duplicate
  if (hasDomain(domainTree, input.id)) {
    errors.push(`Domain already exists: ${input.id}`);
  }

  // 3. Validate parent exists
  const parentId = input.parentDomain ?? "root";
  if (!hasDomain(domainTree, parentId)) {
    errors.push(`Parent domain not found: ${parentId}`);
  }

  // 4. Validate ID follows parent path
  if (input.parentDomain) {
    const expectedPrefix = input.parentDomain + ".";
    if (!input.id.startsWith(expectedPrefix)) {
      errors.push(`Domain ID must start with parent path: ${expectedPrefix}`);
    }

    // Check ID has exactly one more segment than parent
    const parentSegments = input.parentDomain.split(".").length;
    const idSegments = input.id.split(".").length;
    if (idSegments !== parentSegments + 1) {
      errors.push(
        `Domain ID must be direct child of parent (expected ${parentSegments + 1} segments, got ${idSegments})`,
      );
    }
  }

  // 5. Validate stratum relative to parent
  const parentDomain = getDomain(domainTree, parentId);
  if (parentDomain) {
    const parentLevel = getStratumLevel(parentDomain.stratum);
    const newLevel = getStratumLevel(input.stratum);

    if (newLevel < parentLevel) {
      errors.push(
        `Child domain cannot be in lower stratum than parent (${input.stratum} < ${parentDomain.stratum})`,
      );
    }
  }

  // 6. Validate stratum dependencies
  for (const dep of input.dependsOnStrata) {
    if (!isValidStratumDependency(input.stratum, dep)) {
      errors.push(
        `Invalid stratum dependency: ${input.stratum} cannot depend on ${dep}`,
      );
    }
  }

  // 7. Warnings for edge cases
  if (input.dependsOnStrata.length === 0 && input.stratum !== "fundamental") {
    warnings.push(
      `Non-fundamental domain has no declared stratum dependencies`,
    );
  }

  if (input.examples.length === 0) {
    warnings.push(`No examples provided for domain`);
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
  };
}

// ============================================================================
// Fact Submission Validation
// ============================================================================

/**
 * Input for fact submission validation.
 */
export interface FactSubmissionInput {
  domainId: string;
  proposedConfidence: number;
  stratumDependencies: Array<{ stratum: OntologicalStratum; strength: number }>;
  references: Array<{
    targetFactId: string;
    targetDomainId: string;
    referenceType: ReferenceType;
  }>;
}

/**
 * Validation result for fact submission.
 */
export interface FactValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
  adjustedConfidence?: number;
}

/**
 * Validate a fact submission for ontological consistency.
 */
export function validateFactSubmission(
  input: FactSubmissionInput,
  domainTree: DomainTree,
): FactValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];
  let adjustedConfidence = input.proposedConfidence;

  // 1. Validate domain exists
  const domain = getDomain(domainTree, input.domainId);
  if (!domain) {
    errors.push(`Domain not found: ${input.domainId}`);
    return { valid: false, errors, warnings };
  }

  // 2. Validate stratum dependencies
  for (const dep of input.stratumDependencies) {
    if (!isValidStratumDependency(domain.stratum, dep.stratum)) {
      errors.push(
        `Invalid stratum dependency: ${domain.stratum} cannot depend on ${dep.stratum}`,
      );
    }
  }

  // 3. Validate references
  for (const ref of input.references) {
    const result = validateReferenceByDomainId(
      input.domainId,
      ref.targetDomainId,
      ref.referenceType,
      domainTree,
    );

    if (!result.valid) {
      errors.push(`Invalid reference to ${ref.targetFactId}: ${result.reason}`);
    }
  }

  // 4. Check confidence against stratum ceiling
  const ceiling = STRATUM_CONFIDENCE_CEILING[domain.stratum];

  if (input.proposedConfidence > ceiling) {
    warnings.push(
      `Proposed confidence ${input.proposedConfidence} exceeds stratum ceiling ${ceiling}, will be capped`,
    );
    adjustedConfidence = ceiling;
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
    adjustedConfidence,
  };
}

// ============================================================================
// Batch Validation
// ============================================================================

/**
 * Validate multiple references at once.
 */
export function validateReferences(
  references: Array<{
    fromDomainId: string;
    toDomainId: string;
    referenceType: ReferenceType;
  }>,
  domainTree: DomainTree,
): Map<number, OntologicalValidationResult> {
  const results = new Map<number, OntologicalValidationResult>();

  for (let i = 0; i < references.length; i++) {
    const ref = references[i];
    results.set(
      i,
      validateReferenceByDomainId(
        ref.fromDomainId,
        ref.toDomainId,
        ref.referenceType,
        domainTree,
      ),
    );
  }

  return results;
}

/**
 * Check if all references in a batch are valid.
 */
export function areAllReferencesValid(
  references: Array<{
    fromDomainId: string;
    toDomainId: string;
    referenceType: ReferenceType;
  }>,
  domainTree: DomainTree,
): boolean {
  for (const ref of references) {
    const result = validateReferenceByDomainId(
      ref.fromDomainId,
      ref.toDomainId,
      ref.referenceType,
      domainTree,
    );
    if (!result.valid) return false;
  }
  return true;
}

// ============================================================================
// Cross-Domain Validation
// ============================================================================

/**
 * Check if two domains are in compatible strata for a given reference type.
 */
export function areDomainsCompatible(
  fromStratum: OntologicalStratum,
  toStratum: OntologicalStratum,
  referenceType: ReferenceType,
): boolean {
  const fromLevel = getStratumLevel(fromStratum);
  const toLevel = getStratumLevel(toStratum);

  switch (referenceType) {
    case "contradicts":
      return true; // Always compatible

    case "supports":
    case "extends":
      return toLevel <= fromLevel;

    case "specializes":
      return toLevel === fromLevel;

    case "generalizes":
      return toLevel <= fromLevel;

    default:
      return false;
  }
}

/**
 * Get valid reference types between two strata.
 */
export function getValidReferenceTypes(
  fromStratum: OntologicalStratum,
  toStratum: OntologicalStratum,
): ReferenceType[] {
  const valid: ReferenceType[] = ["contradicts"]; // Always valid
  const fromLevel = getStratumLevel(fromStratum);
  const toLevel = getStratumLevel(toStratum);

  if (toLevel <= fromLevel) {
    valid.push("supports", "extends", "generalizes");
  }

  if (toLevel === fromLevel) {
    valid.push("specializes");
  }

  return valid;
}
