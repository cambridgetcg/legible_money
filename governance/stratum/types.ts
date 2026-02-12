/**
 * Stratum Governance Types
 *
 * Defines types for domain proposals and stratum registration governance.
 * Domain proposals use an endorsement-based model (simpler than full LIP voting)
 * while stratum registration requires governance authority.
 *
 * PP HIERARCHY:
 *   TRUTH: Proposal status grounded in on-chain stake and endorsements
 *   UNDERSTANDING: Clear progression from proposal to approval
 *   BEAUTY: Elegant endorsement model fits domain registration needs
 *   JUSTICE: Stakes protect against spam while enabling discovery
 *   CREATIVITY: Enables ontology expansion through structured governance
 */

import type { DID } from "../../core/agents/identity/did.js";
import type {
  OntologicalStratum,
  LogicZone,
  EpistemicCategory,
} from "../../core/knowledge/types.js";

// ============ Domain Proposal Types ============

/**
 * Status of a domain proposal through its lifecycle.
 */
export type DomainProposalStatus =
  | "pending" // Awaiting endorsements
  | "endorsed" // Met endorsement threshold
  | "challenged" // Under challenge
  | "approved" // Approved and registered
  | "rejected" // Rejected via challenge or expiration
  | "expired"; // Voting period elapsed without resolution

/**
 * Input for proposing a new domain to the ontology.
 */
export interface DomainRegistrationInput {
  /** Unique identifier using dot notation: 'physics.mechanics.quantum' */
  id: string;
  /** Human-readable name */
  name: string;
  /** Which ontological stratum this domain belongs to */
  stratum: OntologicalStratum;
  /** Parent domain ID, empty for root domains */
  parentDomain: string;
  /** Which lower strata this domain's facts may depend on */
  dependsOnStrata: OntologicalStratum[];
  /** Default logic zone for claims in this domain */
  logicZoneDefault: LogicZone;
  /** Default epistemic category for claims */
  epistemicCategoryDefault: EpistemicCategory;
  /** Description of what this domain covers */
  description: string;
  /** Example claims that belong in this domain */
  examples: string[];
}

/**
 * A proposal to add a new domain to the ontology.
 */
export interface DomainProposal {
  /** Unique proposal ID */
  id: string;
  /** The proposed domain registration */
  proposedDomain: DomainRegistrationInput;
  /** Proposer's DID */
  proposer: DID;
  /** Stake locked by proposer (in base units) */
  stake: bigint;
  /** Current status */
  status: DomainProposalStatus;
  /** Block when proposed */
  proposedAt: bigint;
  /** Block when voting ends */
  votingEndsAt: bigint;
  /** Total stake from endorsements */
  endorsementStake: bigint;
  /** Total stake from challenges */
  challengeStake: bigint;
  /** Whether this requires governance vote (for high-level strata) */
  requiresGovernance: boolean;
  /** Associated LIP ID if governance required */
  lipId?: string;
}

/**
 * An endorsement of a domain proposal.
 */
export interface DomainEndorsement {
  /** Proposal being endorsed */
  proposalId: string;
  /** Endorser's DID */
  endorser: DID;
  /** Stake locked as endorsement */
  stake: bigint;
  /** Block when endorsed */
  endorsedAt: bigint;
}

/**
 * A challenge to a domain proposal.
 */
export interface DomainChallenge {
  /** Proposal being challenged */
  proposalId: string;
  /** Challenger's DID */
  challenger: DID;
  /** Stake locked as challenge */
  stake: bigint;
  /** Reason for challenge */
  reason: string;
  /** Block when challenged */
  challengedAt: bigint;
}

// ============ Governance Parameters ============

/**
 * Parameters controlling domain proposal governance.
 */
export interface DomainGovernanceParams {
  /** Minimum stake required to propose a domain */
  minProposalStake: bigint;
  /** Minimum stake required to endorse */
  minEndorsementStake: bigint;
  /** Minimum stake required to challenge */
  minChallengeStake: bigint;
  /** Number of blocks for voting period */
  votingPeriodBlocks: bigint;
  /** Minimum number of endorsements required */
  minEndorsements: number;
  /** Minimum total endorsement stake ratio (vs proposal stake) */
  minEndorsementStakeRatio: number;
  /** Strata that require full governance vote */
  governanceRequiredStrata: OntologicalStratum[];
  /** Challenge stake ratio needed to trigger governance review */
  challengeEscalationRatio: number;
}

/**
 * Default governance parameters.
 * Symbolic: Uses protocol constants where meaningful.
 */
export const DEFAULT_DOMAIN_GOVERNANCE_PARAMS: DomainGovernanceParams = {
  minProposalStake: 100n * 10n ** 15n, // 0.1 LGM (affordable with 0.222 LGM genesis balance)
  minEndorsementStake: 25n * 10n ** 15n, // 0.025 LGM
  minChallengeStake: 50n * 10n ** 15n, // 0.05 LGM
  votingPeriodBlocks: 100000n, // ~70 hours at 2521ms
  minEndorsements: 3, // Need 3 endorsers
  minEndorsementStakeRatio: 2.0, // Total endorsements must be 2x proposal stake
  governanceRequiredStrata: ["fundamental", "physical"], // Lower strata need governance
  challengeEscalationRatio: 0.5, // Challenge stake >= 50% of endorsement → escalate
};

// ============ Stratum Registration Types ============

/**
 * Input for registering a new stratum (governance-controlled).
 * This is a major protocol change requiring full LIP process.
 */
export interface StratumRegistrationInput {
  /** Unique stratum identifier */
  id: string;
  /** Human-readable name */
  name: string;
  /** Level in the hierarchy (0 = most fundamental) */
  level: number;
  /** Maximum confidence for claims in this stratum */
  confidenceCeiling: number;
  /** Default logic zone */
  defaultLogicZone: LogicZone;
  /** Default epistemic category */
  defaultEpistemicCategory: EpistemicCategory;
  /** Which strata this can validly reference */
  validDependencies: string[];
  /** Description */
  description: string;
  /** Example domains */
  exampleDomains: string[];
}

/**
 * Result of stratum registration proposal validation.
 */
export interface StratumRegistrationValidation {
  valid: boolean;
  errors: string[];
  warnings: string[];
  requiredLIPCategory: "consensus" | "process";
}

// ============ Governance Actions ============

/**
 * Actions that can be taken on a domain proposal.
 */
export type DomainProposalAction =
  | "endorse"
  | "challenge"
  | "withdraw"
  | "finalize";

/**
 * Result of a governance action.
 */
export interface GovernanceActionResult {
  success: boolean;
  error?: string;
  newStatus?: DomainProposalStatus;
  stakeChange?: bigint;
  escalatedToLIP?: string;
}

// ============ Query Types ============

/**
 * Filter for querying domain proposals.
 */
export interface DomainProposalFilter {
  status?: DomainProposalStatus;
  proposer?: DID;
  stratum?: OntologicalStratum;
  parentDomain?: string;
  minStake?: bigint;
}

/**
 * Summary statistics for domain governance.
 */
export interface DomainGovernanceStats {
  totalProposals: number;
  pendingProposals: number;
  approvedProposals: number;
  rejectedProposals: number;
  totalStakeLocked: bigint;
  averageEndorsements: number;
  challengeRate: number;
}
