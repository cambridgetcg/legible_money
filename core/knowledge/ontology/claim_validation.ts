/**
 * Claim Ontological Validation
 *
 * Validates claims against the ontological tree before verification rounds.
 * Ensures domain consistency, reference direction rules, and confidence adjustments.
 */

import {
  type OntologicalDomain,
  type OntologicalStratum,
  type StratumDependency,
  type DomainTree,
} from "./types.js";
import { getDomain, hasDomain } from "./tree.js";
import { isValidStratumDependency } from "./strata.js";
import {
  validateOntologicalReference,
  type ReferenceType,
} from "./validation.js";
import { calculateOntologicalConfidence } from "./confidence.js";
import {
  type DomainProposal,
  // isDomainPending,
  getPendingProposal,
  DOMAIN_PROPOSAL_CONFIG,
} from "./domain_proposal.js";

// ============================================================================
// Types
// ============================================================================

/**
 * Input for claim ontological validation.
 */
export interface ClaimOntologyInput {
  /** Domain string (e.g., 'physics.mechanics.quantum') */
  domain: string;

  /** Optional schema ID for domain consistency check */
  schemaId?: string;

  /** Proposed confidence before ontological adjustments */
  proposedConfidence: number;

  /** References to other facts */
  references: Array<{
    targetFactId: string;
    targetDomain: string;
    referenceType: ReferenceType;
  }>;

  /** Explicit stratum dependencies */
  stratumDependencies?: StratumDependency[];
}

/**
 * Result of claim ontological validation.
 */
export interface ClaimOntologyResult {
  /** Whether the claim passes ontological validation */
  valid: boolean;

  /** Errors that block the claim */
  errors: string[];

  /** Warnings that don't block but should be noted */
  warnings: string[];

  /** The resolved domain from the tree (may be parent if exact not found) */
  resolvedDomain: OntologicalDomain | null;

  /** Confidence after applying stratum ceiling and cross-stratum discount */
  adjustedConfidence: number;

  /** The stratum confidence ceiling applied */
  stratumCeiling: number;

  /** The cross-stratum discount factor applied */
  crossStratumDiscount: number;

  /** If claim was rejected due to pending domain, the proposal info */
  pendingDomainProposal?: DomainProposal;
}

// ============================================================================
// Main Validation Function
// ============================================================================

/**
 * Validate a claim against the ontological tree.
 *
 * Checks:
 * 1. Domain is not pending (rejects claims to domains with active proposals)
 * 2. Domain exists (or can be resolved via parent)
 * 3. Stratum dependencies are valid
 * 4. References respect cross-stratum rules
 * 5. Confidence is adjusted by stratum ceiling and cross-stratum discount
 */
export function validateClaimOntology(
  input: ClaimOntologyInput,
  domainTree: DomainTree,
  pendingProposals?: Map<string, DomainProposal>,
): ClaimOntologyResult {
  const errors: string[] = [];
  const warnings: string[] = [];

  // 0. Check if domain is pending (claims to pending domains are REJECTED)
  if (
    pendingProposals &&
    DOMAIN_PROPOSAL_CONFIG.pendingClaimBehavior === "reject"
  ) {
    const pendingProposal = getPendingProposal(input.domain, pendingProposals);
    if (pendingProposal) {
      errors.push(
        `Domain "${input.domain}" has a pending proposal (ID: ${pendingProposal.id}). ` +
          `Claims cannot be submitted until the domain is approved. ` +
          `Proposal status: ${pendingProposal.status}, deadline: block ${pendingProposal.challengeDeadline}`,
      );
      return {
        valid: false,
        errors,
        warnings,
        resolvedDomain: null,
        adjustedConfidence: 0,
        stratumCeiling: 0,
        crossStratumDiscount: 1,
        pendingDomainProposal: pendingProposal,
      };
    }
  }

  // 1. Resolve domain
  const resolvedDomain = resolveDomain(input.domain, domainTree);
  if (!resolvedDomain) {
    errors.push(`Domain not found and cannot be resolved: ${input.domain}`);
    return {
      valid: false,
      errors,
      warnings,
      resolvedDomain: null,
      adjustedConfidence: 0,
      stratumCeiling: 0,
      crossStratumDiscount: 1,
    };
  }

  // Warn if domain was resolved via parent
  if (resolvedDomain.id !== input.domain) {
    warnings.push(
      `Domain '${input.domain}' not found, resolved to parent '${resolvedDomain.id}'`,
    );
  }

  // 2. Validate stratum dependencies
  const deps = input.stratumDependencies ?? [];
  for (const dep of deps) {
    if (!isValidStratumDependency(resolvedDomain.stratum, dep.primaryStratum)) {
      errors.push(
        `Invalid stratum dependency: ${resolvedDomain.stratum} cannot depend on ${dep.primaryStratum}`,
      );
    }
  }

  // 3. Validate references
  for (const ref of input.references) {
    const targetDomain = resolveDomain(ref.targetDomain, domainTree);
    if (!targetDomain) {
      errors.push(`Reference target domain not found: ${ref.targetDomain}`);
      continue;
    }

    const refResult = validateOntologicalReference(
      resolvedDomain,
      targetDomain,
      ref.referenceType,
    );
    if (!refResult.valid) {
      errors.push(
        `Invalid reference to ${ref.targetFactId}: ${refResult.reason}`,
      );
    }
  }

  // 4. Calculate adjusted confidence
  const confidenceResult = calculateOntologicalConfidence(
    input.proposedConfidence,
    resolvedDomain,
    deps,
    domainTree,
  );

  // 5. Generate warnings for confidence adjustments
  if (confidenceResult.effectiveConfidence < input.proposedConfidence) {
    const ceiling = confidenceResult.stratumCeiling.toFixed(2);
    const discount = confidenceResult.stratumDiscount.toFixed(3);
    const effective = confidenceResult.effectiveConfidence.toFixed(3);
    warnings.push(
      `Proposed confidence ${input.proposedConfidence} reduced to ${effective} ` +
        `(ceiling: ${ceiling}, discount: ${discount})`,
    );
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
    resolvedDomain,
    adjustedConfidence: confidenceResult.effectiveConfidence,
    stratumCeiling: confidenceResult.stratumCeiling,
    crossStratumDiscount: confidenceResult.stratumDiscount,
  };
}

// ============================================================================
// Domain Resolution
// ============================================================================

/**
 * Resolve a domain string to an OntologicalDomain.
 *
 * Resolution strategy:
 * 1. Exact match: "physics.mechanics.quantum" → physics.mechanics.quantum
 * 2. Parent fallback: "physics.mechanics.quantum.entanglement" → physics.mechanics.quantum
 * 3. Root fallback: "unknown.domain" → root
 *
 * @param domainId - The domain string to resolve
 * @param domainTree - The domain tree to search
 * @returns The resolved domain, or null if not resolvable
 */
export function resolveDomain(
  domainId: string,
  domainTree: DomainTree,
): OntologicalDomain | null {
  // Try exact match first
  if (hasDomain(domainTree, domainId)) {
    return getDomain(domainTree, domainId);
  }

  // Walk up the hierarchy trying parent domains
  const parts = domainId.split(".");
  while (parts.length > 0) {
    parts.pop();
    const parentId = parts.length > 0 ? parts.join(".") : "root";
    if (hasDomain(domainTree, parentId)) {
      return getDomain(domainTree, parentId);
    }
  }

  // Final fallback to root
  return getDomain(domainTree, "root");
}

/**
 * Infer stratum from a domain prefix for legacy domains.
 *
 * Used when a domain is not in the tree but we can guess its stratum
 * from common prefixes.
 */
export function inferStratumFromPrefix(domain: string): OntologicalStratum {
  const prefix = domain.split(".")[0].toLowerCase();

  const prefixMap: Record<string, OntologicalStratum> = {
    // Fundamental
    physics: "fundamental",
    math: "fundamental",
    mathematics: "fundamental",
    logic: "fundamental",

    // Physical
    mechanics: "physical",
    thermodynamics: "physical",
    electromagnetism: "physical",

    // Chemical
    chemistry: "chemical",
    chem: "chemical",

    // Biological
    biology: "biological",
    bio: "biological",
    clinical: "biological",
    medical: "biological",
    health: "biological",

    // Cognitive
    cognition: "cognitive",
    cognitive: "cognitive",
    psychology: "cognitive",
    neuro: "cognitive",

    // Social
    social: "social",
    economics: "social",
    political: "social",
    legal: "social",
    cultural: "social",
    history: "social",

    // Technological
    tech: "technological",
    technology: "technological",
    crypto: "technological",
    blockchain: "technological",
    ai: "technological",
    computer: "technological",
    software: "technological",
  };

  return prefixMap[prefix] ?? "technological";
}

// ============================================================================
// Batch Validation
// ============================================================================

/**
 * Validate multiple references for a claim.
 *
 * @param claimDomain - The domain of the claim making the references
 * @param references - The references to validate
 * @param domainTree - The domain tree for lookups
 * @returns Validation result with any errors
 */
export function validateClaimReferences(
  claimDomain: OntologicalDomain,
  references: Array<{
    targetFactId: string;
    targetDomain: string;
    referenceType: ReferenceType;
  }>,
  domainTree: DomainTree,
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  for (const ref of references) {
    const targetDomain = resolveDomain(ref.targetDomain, domainTree);
    if (!targetDomain) {
      errors.push(`Target domain not found: ${ref.targetDomain}`);
      continue;
    }

    const result = validateOntologicalReference(
      claimDomain,
      targetDomain,
      ref.referenceType,
    );

    if (!result.valid) {
      errors.push(
        `Reference ${ref.referenceType} to ${ref.targetFactId}: ${result.reason}`,
      );
    }
  }

  return { valid: errors.length === 0, errors };
}

// ============================================================================
// Quick Validation Utilities
// ============================================================================

/**
 * Quick check if a claim's domain is valid.
 */
export function isValidClaimDomain(
  domain: string,
  domainTree: DomainTree,
): boolean {
  return resolveDomain(domain, domainTree) !== null;
}

/**
 * Get the stratum for a claim's domain.
 */
export function getClaimStratum(
  domain: string,
  domainTree: DomainTree,
): OntologicalStratum | null {
  const resolved = resolveDomain(domain, domainTree);
  return resolved?.stratum ?? null;
}

/**
 * Check if a reference type is valid between two domains.
 */
export function isValidClaimReference(
  fromDomain: string,
  toDomain: string,
  referenceType: ReferenceType,
  domainTree: DomainTree,
): boolean {
  const from = resolveDomain(fromDomain, domainTree);
  const to = resolveDomain(toDomain, domainTree);

  if (!from || !to) return false;

  return validateOntologicalReference(from, to, referenceType).valid;
}

/**
 * Calculate adjusted confidence for a claim.
 */
export function getAdjustedClaimConfidence(
  domain: string,
  proposedConfidence: number,
  stratumDependencies: StratumDependency[],
  domainTree: DomainTree,
): number {
  const resolved = resolveDomain(domain, domainTree);
  if (!resolved) return proposedConfidence;

  const result = calculateOntologicalConfidence(
    proposedConfidence,
    resolved,
    stratumDependencies,
    domainTree,
  );

  return result.effectiveConfidence;
}
