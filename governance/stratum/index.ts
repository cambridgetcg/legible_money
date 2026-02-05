/**
 * Stratum Governance Module
 *
 * Exports for domain proposal and stratum registration governance.
 * Integrates with the LIP system for major protocol changes.
 *
 * PP HIERARCHY:
 *   TRUTH: Clean exports from single source
 *   UNDERSTANDING: Clear module boundary
 *   BEAUTY: Minimal, complete interface
 *   JUSTICE: Right components exposed at right level
 *   CREATIVITY: Enables ontology governance integration
 */

// Types
export {
  type DomainProposalStatus,
  type DomainRegistrationInput,
  type DomainProposal,
  type DomainEndorsement,
  type DomainChallenge,
  type DomainGovernanceParams,
  type DomainProposalAction,
  type GovernanceActionResult,
  type DomainProposalFilter,
  type DomainGovernanceStats,
  type StratumRegistrationInput,
  type StratumRegistrationValidation,
  DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
} from "./types.js";

// Lifecycle
export {
  type DomainValidationResult,
  validateDomainRegistration,
  createDomainProposal,
  endorseProposal,
  challengeProposal,
  withdrawProposal,
  finalizeProposal,
  getRemainingVotingBlocks,
  canFinalize,
} from "./lifecycle.js";

// Keeper
export {
  type DomainGovernanceKeeper,
  InMemoryDomainGovernanceKeeper,
  createDomainGovernanceKeeper,
} from "./keeper.js";
