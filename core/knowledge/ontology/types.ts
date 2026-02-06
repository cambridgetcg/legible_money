/**
 * Ontological Tree of Knowledge - Type Definitions
 *
 * Defines the hierarchical structure of knowledge from fundamental physics
 * through emergent phenomena (chemistry, biology, cognition, society, technology).
 *
 * Key principle: Higher strata DEPEND on lower strata. Chemistry depends on
 * physics. Biology depends on chemistry. This dependency is explicit and enforced.
 */

import type { LogicZone } from "../../logic/types.js";
import type { EpistemicCategory } from "../types.js";

// ============================================================================
// Ontological Stratum
// ============================================================================

/**
 * The seven ontological strata representing levels of emergence.
 *
 * Each stratum emerges from and depends on lower strata:
 * - fundamental: Physics constants, particles, mathematics (axiomatic)
 * - physical: Mechanics, thermodynamics, electromagnetism
 * - chemical: Elements, molecules, reactions
 * - biological: Cells, organisms, ecosystems
 * - cognitive: Perception, reasoning, consciousness
 * - social: Economics, politics, culture
 * - technological: Computation, networks, AI (designed systems)
 */
export type OntologicalStratum =
  | "fundamental"
  | "physical"
  | "chemical"
  | "biological"
  | "cognitive"
  | "social"
  | "technological";

/**
 * Numeric hierarchy of strata (lower = more fundamental)
 */
export const STRATUM_HIERARCHY: Record<OntologicalStratum, number> = {
  fundamental: 0,
  physical: 1,
  chemical: 2,
  biological: 3,
  cognitive: 4,
  social: 5,
  technological: 6,
};

/**
 * Maximum confidence achievable for facts in each stratum.
 *
 * Lower strata have higher ceilings because they're more foundational:
 * - fundamental/technological: 1.0 (mathematical certainty / protocol-defined)
 * - physical: 0.99 (measurement limits)
 * - chemical: 0.98 (experimental uncertainty)
 * - biological: 0.95 (biological variability)
 * - cognitive: 0.90 (subjective elements)
 * - social: 0.85 (social construction)
 */
export const STRATUM_CONFIDENCE_CEILING: Record<OntologicalStratum, number> = {
  fundamental: 1.0,
  physical: 0.99,
  chemical: 0.98,
  biological: 0.95,
  cognitive: 0.9,
  social: 0.85,
  technological: 1.0, // Protocol-defined truths can be certain
};

/**
 * Mapping from ontological stratum to decay category.
 *
 * Used as FALLBACK for calculating time-decayed confidence values when
 * a fact has no explicit epistemicCategory stored. New facts should have
 * epistemicCategory set from their domain's epistemicCategoryDefault.
 *
 * MUST be consistent with STRATUM_EPISTEMIC_CATEGORY below.
 *
 * Categories align with CATEGORY_DECAY (derived from EPISTEMIC_PROFILES):
 * - "analytic": Eternal — fundamental truths don't decay
 * - "protocol": Eternal — on-chain/protocol facts are permanent
 * - "empirical": ~2.9 days half-life — physical/chemical/biological facts need reverification
 * - "social": ~4.4 days half-life — cognitive/social consensus is stickier
 */
export const STRATUM_TO_DECAY_CATEGORY: Record<
  OntologicalStratum,
  EpistemicCategory
> = {
  fundamental: "formal", // Consistent with STRATUM_EPISTEMIC_CATEGORY (both eternal)
  physical: "empirical",
  chemical: "empirical",
  biological: "empirical",
  cognitive: "social",
  social: "social",
  technological: "protocol",
};

/**
 * Default logic zone for each stratum
 */
export const STRATUM_LOGIC_ZONE: Record<OntologicalStratum, LogicZone> = {
  fundamental: "peano",
  physical: "peano",
  chemical: "empirical",
  biological: "empirical",
  cognitive: "empirical",
  social: "empirical",
  technological: "propositional",
};

/**
 * Default epistemic category for each stratum.
 *
 * Used as the default when registering new domains. Domain-level overrides
 * are possible via OntologicalDomain.epistemicCategoryDefault.
 *
 * MUST be consistent with STRATUM_TO_DECAY_CATEGORY above — a domain's
 * epistemic category determines its decay behavior.
 */
export const STRATUM_EPISTEMIC_CATEGORY: Record<
  OntologicalStratum,
  EpistemicCategory
> = {
  fundamental: "formal", // Math axioms — eternal, proven
  physical: "empirical", // Physics observations — decay, need reverification
  chemical: "empirical", // Chemistry experiments — decay
  biological: "empirical", // Biology observations — decay
  cognitive: "social", // Cognitive science — consensus-mediated, stickier decay
  social: "social", // Social sciences — consensus-based
  technological: "protocol", // On-chain constructed — eternal
};

// ============================================================================
// Ontological Domain
// ============================================================================

/**
 * An ontological domain represents a branch in the knowledge tree.
 *
 * Domains are hierarchical (e.g., 'physics.mechanics.quantum') and inherit
 * properties from their parent domains. Each domain belongs to exactly one
 * stratum and can only depend on strata at or below its level.
 */
export interface OntologicalDomain {
  /** Unique identifier using dot notation: 'physics.mechanics.quantum' */
  id: string;

  /** Human-readable name: 'Quantum Mechanics' */
  name: string;

  /** Which ontological stratum this domain belongs to */
  stratum: OntologicalStratum;

  /** Parent domain ID, or null for root domains */
  parentDomain: string | null;

  /** IDs of child domains */
  childDomains: string[];

  /** Which lower strata this domain's facts may depend on */
  dependsOnStrata: OntologicalStratum[];

  /** Default logic zone for claims in this domain */
  logicZoneDefault: LogicZone;

  /** Default epistemic category for claims in this domain */
  epistemicCategoryDefault: EpistemicCategory;

  /** Description of what this domain covers */
  description: string;

  /** Example claims that belong in this domain */
  examples: string[];

  /** Number of facts currently in this domain */
  factCount: number;

  /** Number of axioms (confidence 1.0 facts) in this domain */
  axiomCount: number;

  /** Genesis fact ID that established this domain (if any) */
  rootFactId?: string;

  /** Block number when domain was created */
  createdAt: number;

  /** Governance proposal that created this domain (if not genesis) */
  governanceProposal?: string;

  /** Block number when domain was deprecated (undefined = active) */
  deprecatedAt?: number;
}

// ============================================================================
// Stratum Dependencies
// ============================================================================

/**
 * How a fact depends on knowledge from other strata.
 *
 * For example, a fact about protein folding (biological) depends on
 * facts about molecular bonds (chemical) and electron interactions (physical).
 */
export interface StratumDependency {
  /** The fact that has this dependency */
  factId: string;

  /** The primary stratum of this fact */
  primaryStratum: OntologicalStratum;

  /** Lower strata this fact depends on */
  dependentStrata: OntologicalStratum[];

  /** Type of dependency */
  dependencyType: StratumDependencyType;

  /** Strength of the dependency (0-1), affects confidence propagation */
  dependencyStrength: number;
}

/**
 * Types of cross-stratum dependencies:
 * - emergence: Higher-level behavior arising from lower-level rules
 * - composition: Made up of lower-level components
 * - constraint: Limited by lower-level laws
 * - application: Uses lower-level knowledge as tool
 */
export type StratumDependencyType =
  | "emergence"
  | "composition"
  | "constraint"
  | "application";

// ============================================================================
// Domain Tree Structure
// ============================================================================

/**
 * The complete domain tree with indexes for efficient lookup.
 */
export interface DomainTree {
  /** Root node of the tree */
  root: DomainNode;

  /** Index: stratum -> domain IDs in that stratum */
  strataIndex: Map<OntologicalStratum, string[]>;

  /** Index: domain ID -> domain node */
  domainIndex: Map<string, DomainNode>;

  /** Total number of domains in the tree */
  domainCount: number;

  /** Maximum depth of the tree */
  maxDepth: number;
}

/**
 * A node in the domain tree.
 */
export interface DomainNode {
  /** The domain data */
  domain: OntologicalDomain;

  /** Child nodes */
  children: DomainNode[];

  /** Depth from root (root = 0) */
  depth: number;

  /** Path from root: ['physics', 'mechanics', 'quantum'] */
  path: string[];
}

// ============================================================================
// Confidence Calculation Results
// ============================================================================

/**
 * Result of ontological confidence calculation.
 */
export interface OntologicalConfidenceResult {
  /** Original confidence before ontological adjustments */
  baseConfidence: number;

  /** Maximum confidence allowed by this stratum */
  stratumCeiling: number;

  /** Maximum confidence inherited from parent domains */
  inheritedCeiling: number;

  /** Discount factor from cross-stratum dependencies */
  stratumDiscount: number;

  /** Final effective confidence */
  effectiveConfidence: number;

  /** Details of dependencies that affected the calculation */
  dependencies: Array<{
    stratum: OntologicalStratum;
    type: StratumDependencyType;
    strength: number;
  }>;
}

// ============================================================================
// Domain Registration
// ============================================================================

/**
 * Input for registering a new domain.
 */
export interface DomainRegistrationInput {
  id: string;
  name: string;
  stratum: OntologicalStratum;
  parentDomain: string | null;
  dependsOnStrata: OntologicalStratum[];
  logicZoneDefault?: LogicZone;
  epistemicCategoryDefault?: EpistemicCategory;
  description: string;
  examples: string[];
}

/**
 * Result of domain registration.
 */
export interface DomainRegistrationResult {
  /** The created domain */
  domain: OntologicalDomain;

  /** Whether governance approval is required */
  requiresGovernance: boolean;

  /** Minimum stake required for registration */
  minimumStake: bigint;

  /** Validation errors (if any) */
  errors: string[];
}

// ============================================================================
// Validation Results
// ============================================================================

/**
 * Result of validating an ontological reference.
 */
export interface OntologicalValidationResult {
  /** Whether the reference is valid */
  valid: boolean;

  /** Reason for invalidity (if invalid) */
  reason?: string;

  /** The from domain's stratum level */
  fromLevel?: number;

  /** The to domain's stratum level */
  toLevel?: number;
}

// ============================================================================
// Domain Propagation
// ============================================================================

/**
 * Result of propagating confidence changes through the domain hierarchy.
 */
export interface DomainPropagationResult {
  /** Facts affected by the change */
  affectedFacts: Array<{
    factId: string;
    domain: string;
    oldConfidence: number;
    newConfidence: number;
  }>;

  /** Domains that were processed */
  affectedDomains: string[];

  /** Cross-domain updates (between different domain branches) */
  crossDomainUpdates: Array<{
    factId: string;
    fromDomain: string;
    toDomain: string;
    confidenceChange: number;
  }>;
}
