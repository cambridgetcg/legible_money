/**
 * Ontological Tree of Knowledge
 *
 * A hierarchical knowledge ontology that grows from fundamental understanding
 * (particles, forces, physical laws) upward through emergent phenomena
 * (chemistry, biology, cognition, society, technology).
 *
 * Key principle: Higher strata DEPEND on lower strata.
 * Chemistry depends on physics. Biology depends on chemistry.
 * This dependency is explicit and enforced.
 *
 * @module core/knowledge/ontology
 */

// ============================================================================
// Type Exports
// ============================================================================

export type {
  OntologicalStratum,
  OntologicalDomain,
  StratumDependency,
  StratumDependencyType,
  DomainTree,
  DomainNode,
  OntologicalConfidenceResult,
  DomainRegistrationInput,
  DomainRegistrationResult,
  OntologicalValidationResult,
  DomainPropagationResult,
} from "./types.js";

export {
  STRATUM_HIERARCHY,
  STRATUM_CONFIDENCE_CEILING,
  STRATUM_LOGIC_ZONE,
  STRATUM_EPISTEMIC_CATEGORY,
} from "./types.js";

// ============================================================================
// Stratum Operations
// ============================================================================

export {
  VALID_DEPENDENCIES,
  getValidDependencies,
  isValidStratumDependency,
  validateDependencies,
  getStratumLevel,
  isHigherStratum,
  isLowerStratum,
  getStratumDistance,
  getStrataBelowOrEqual,
  getStrataAboveOrEqual,
  getStratumConfidenceCeiling,
  getMinConfidenceCeiling,
  applyStratumCeiling,
  DEPENDENCY_TYPE_STRENGTH,
  calculateCrossStratumDiscount,
  STRATUM_DESCRIPTIONS,
  STRATUM_EXAMPLE_DOMAINS,
  getStrataInOrder,
  parseStratum,
} from "./strata.js";

// ============================================================================
// Tree Operations
// ============================================================================

export {
  ROOT_DOMAIN,
  createEmptyDomainTree,
  createDomainTree,
  addDomainToTree,
  getDomain,
  getDomainNode,
  hasDomain,
  getDomainsByStratum,
  getDomainPath,
  getAncestors,
  getDescendants,
  getChildren,
  getSiblings,
  getLowestCommonAncestor,
  registerDomain,
  isValidDomainId,
  getParentIdFromDomainId,
  getDomainDepth,
  getTreeStatistics,
  serializeDomainTree,
  deserializeDomainTree,
} from "./tree.js";

// ============================================================================
// Genesis Domains
// ============================================================================

export {
  GENESIS_DOMAINS,
  getGenesisDomainCount,
  getGenesisDomainsByStratum,
} from "./genesis.js";

// ============================================================================
// Confidence Calculation
// ============================================================================

export type {
  OntologicalConfidenceConfig,
  ConfidenceChange,
} from "./confidence.js";

export {
  DEFAULT_CONFIDENCE_CONFIG,
  calculateOntologicalConfidence,
  calculateDerivedConfidence,
  calculateDependencyImpact,
  aggregateConfidences,
  constrainAggregatedConfidence,
  getMaximumConfidence,
  isConfidenceAchievable,
  calculateCascadingChanges,
  formatConfidence,
  parseConfidence,
} from "./confidence.js";

// ============================================================================
// Validation
// ============================================================================

export type {
  ReferenceType,
  DomainValidationResult,
  FactSubmissionInput,
  FactValidationResult,
} from "./validation.js";

export {
  validateOntologicalReference,
  validateReferenceByDomainId,
  validateDomainRegistration,
  validateFactSubmission,
  validateReferences,
  areAllReferencesValid,
  areDomainsCompatible,
  getValidReferenceTypes,
} from "./validation.js";

// ============================================================================
// Claim Validation
// ============================================================================

export type {
  ClaimOntologyInput,
  ClaimOntologyResult,
} from "./claim_validation.js";

export {
  validateClaimOntology,
  resolveDomain,
  inferStratumFromPrefix,
  validateClaimReferences,
  isValidClaimDomain,
  getClaimStratum,
  isValidClaimReference,
  getAdjustedClaimConfidence,
} from "./claim_validation.js";

// ============================================================================
// Domain Proposal System
// ============================================================================

export type {
  DomainProposalStatus,
  DomainProposal,
  DomainProposalInput,
  DomainProposalValidation,
  DomainEndorsement,
  DomainChallenge,
} from "./domain_proposal.js";

export {
  DOMAIN_PROPOSAL_CONFIG,
  LGM,
  calculateRequiredStake,
  calculateEndorsementStake,
  requiresGovernance,
  validateDomainProposal,
  createDomainProposal,
  endorseProposal,
  challengeProposal,
  finalizeProposal,
  calculateStakeReturn,
  calculateEndorserStakeReturns,
  getTotalEndorsementStake,
  createDomainFromProposal,
  isDomainPending,
  getPendingProposal,
} from "./domain_proposal.js";

// ============================================================================
// Stratum Registry (New - Preferred for New Code)
// ============================================================================

export type {
  StratumDefinition,
  StratumMetadata,
  SerializedStratumRegistry,
  MigrationRecord,
  StratumChange,
  StratumRegistrationResult,
  OntologyProvider,
  LegacyOntologicalStratum,
} from "./stratum_registry.js";

export {
  CURRENT_SCHEMA_VERSION,
  GENESIS_STRATA,
  StratumRegistry,
  createOntologyProvider,
  defaultRegistry,
  defaultOntologyProvider,
  isLegacyStratum,
  parseStratumId,
} from "./stratum_registry.js";

// ============================================================================
// Stratum Migration
// ============================================================================

export type {
  StratumMigration,
  MigrationResult,
  FactMigrationInput,
  FactMigrationOutput,
} from "./stratum_migration.js";

export {
  MigrationRunner,
  validateMigration,
  dryRunMigration,
  migrateRegistry,
  migration_identity,
  migration_split_technological,
} from "./stratum_migration.js";

// ============================================================================
// Convenience Functions
// ============================================================================

import { createDomainTree } from "./tree.js";
import { GENESIS_DOMAINS } from "./genesis.js";

/**
 * Create a domain tree initialized with all genesis domains.
 */
export function createGenesisDomainTree(): import("./types.js").DomainTree {
  return createDomainTree(GENESIS_DOMAINS);
}
