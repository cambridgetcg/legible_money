/**
 * Domain Proposal System
 *
 * Handles proposals for new knowledge domains in the Ontological Tree.
 * Allows the tree to grow organically while maintaining quality through
 * stake requirements and validator endorsement.
 *
 * Symbolic Constants:
 * - 22/222/2222 LGM stakes (2 = "to", growth pattern)
 * - 77% stake return on rejection (7 = luck, fair return)
 * - 777 blocks challenge period (~32.7 minutes at 2521ms blocks)
 * - 111 max children (1 = unity, balanced growth)
 */

import type {
  OntologicalStratum,
  OntologicalDomain,
  DomainTree,
} from "./types.js";
import type { LogicZone as _LogicZone } from "../../logic/types.js";
import type { EpistemicCategory as _EpistemicCategory } from "../types.js";
import {
  STRATUM_HIERARCHY,
  STRATUM_LOGIC_ZONE,
  STRATUM_EPISTEMIC_CATEGORY,
} from "./types.js";
import { sha512_256Hex } from "../../crypto/hashing.js";

// ============================================================================
// Configuration
// ============================================================================

/**
 * Domain proposal configuration with symbolic constants
 */
export const DOMAIN_PROPOSAL_CONFIG = {
  // Stake requirements (22/222/2222 pattern)
  leafDomainStake: 22n * 10n ** 15n, // 0.022 LGM for leaf domains (depth > 2)
  midDomainStake: 222n * 10n ** 15n, // 0.222 LGM for mid-level (depth 2)
  topLevelDomainStake: 2222n * 10n ** 15n, // 2.222 LGM for top-level (depth 1)

  // Validator endorsement stake lock (proportional to domain level)
  // Validators lock this amount when endorsing, returned on finalization
  leafEndorsementStake: 2n * 10n ** 15n, // 0.002 LGM for leaf domains
  midEndorsementStake: 22n * 10n ** 15n, // 0.022 LGM for mid-level
  topLevelEndorsementStake: 222n * 10n ** 15n, // 0.222 LGM for top-level

  // Stake return on rejection (77%)
  stakeReturnOnRejection: 0.77,
  stakeReturnOnApproval: 1.0, // Full return on approval

  // Governance triggers
  governanceRequiredStratum: 2, // Fundamental, physical, chemical require governance
  governanceRequiredDepth: 1, // Direct children of root require governance

  // Auto-approval settings
  challengePeriodBlocks: 777n, // ~32.7 minutes at 2521ms blocks
  minimumSupporters: 2, // 2 validators must endorse
  parentVetoEnabled: false, // Parents cannot block children

  // Claim handling for pending domains
  pendingClaimBehavior: "reject" as const, // Reject claims to pending domains

  // Hierarchy constraints
  maxDepth: 10,
  maxChildrenPerDomain: 111, // Max 111 children per domain
  crossStratumDomains: false, // Must stay within stratum
} as const;

/**
 * LGM token unit (10^18 wei equivalent)
 */
export const LGM = 10n ** 18n;

// ============================================================================
// Domain Proposal Types
// ============================================================================

/**
 * Status of a domain proposal
 */
export type DomainProposalStatus =
  | "pending" // Waiting for challenge period or endorsement
  | "endorsed" // Has minimum supporters, waiting for challenge period
  | "governance" // Requires governance vote
  | "approved" // Approved and active
  | "rejected" // Rejected by governance or challenge
  | "expired"; // Challenge period ended without approval

/**
 * A proposal to create a new domain in the knowledge tree
 */
export interface DomainProposal {
  /** Unique proposal ID */
  id: string;

  /** Proposed domain ID (dot notation) */
  proposedDomainId: string;

  /** Human-readable name */
  name: string;

  /** Target stratum */
  stratum: OntologicalStratum;

  /** Parent domain ID */
  parentDomainId: string;

  /** Description of what this domain covers */
  description: string;

  /** Example claims that would belong in this domain */
  examples: string[];

  /** Proposer's DID */
  proposer: string;

  /** Stake locked by proposer */
  stake: bigint;

  /** Current status */
  status: DomainProposalStatus;

  /** Block when proposal was submitted */
  submittedAt: bigint;

  /** Block when challenge period ends */
  challengeDeadline: bigint;

  /** Validators who endorsed this proposal */
  endorsers: string[];

  /** Stake locked by each endorser (DID -> stake amount) */
  endorserStakes: Map<string, bigint>;

  /** Required endorsement stake for this proposal (based on depth) */
  requiredEndorsementStake: bigint;

  /** Validators who challenged this proposal */
  challengers: string[];

  /** Stake locked by each challenger (DID -> stake amount) */
  challengerStakes: Map<string, bigint>;

  /** Reason given by each challenger (DID -> reason) */
  challengerReasons: Map<string, string>;

  /** If rejected, reason for rejection */
  rejectionReason?: string;

  /** If approved, block when approved */
  approvedAt?: bigint;

  /** Governance proposal ID if requires governance */
  governanceProposalId?: string;

  /** Depth of the proposed domain (for stake calculations) */
  depth: number;
}

/**
 * Input for creating a domain proposal
 */
export interface DomainProposalInput {
  /** Proposed domain ID (dot notation) */
  domainId: string;

  /** Human-readable name */
  name: string;

  /** Target stratum */
  stratum: OntologicalStratum;

  /** Parent domain ID */
  parentDomainId: string;

  /** Description of what this domain covers */
  description: string;

  /** Example claims (minimum 2) */
  examples: string[];

  /** Proposer's DID */
  proposer: string;

  /** Stake amount (must meet minimum for domain type) */
  stake: bigint;
}

/**
 * Result of domain proposal validation
 */
export interface DomainProposalValidation {
  valid: boolean;
  errors: string[];
  requiredStake: bigint;
  requiresGovernance: boolean;
  depth: number;
}

/**
 * Endorsement for a domain proposal
 */
export interface DomainEndorsement {
  proposalId: string;
  endorser: string;
  endorsedAt: bigint;
  comment?: string;
}

/**
 * Challenge against a domain proposal
 */
export interface DomainChallenge {
  proposalId: string;
  challenger: string;
  challengedAt: bigint;
  reason: string;
  stake: bigint;
}

// ============================================================================
// Validation Functions
// ============================================================================

/**
 * Calculate required stake for a domain proposal
 */
export function calculateRequiredStake(
  parentDomainId: string,
  depth: number,
): bigint {
  // Top-level domains (direct children of root)
  if (parentDomainId === "root" || depth === 1) {
    return DOMAIN_PROPOSAL_CONFIG.topLevelDomainStake;
  }

  // Mid-level domains (depth 2)
  if (depth === 2) {
    return DOMAIN_PROPOSAL_CONFIG.midDomainStake;
  }

  // Leaf domains (depth > 2)
  return DOMAIN_PROPOSAL_CONFIG.leafDomainStake;
}

/**
 * Calculate required endorsement stake for a validator
 *
 * Validators must lock this stake when endorsing a domain proposal.
 * Stakes are returned on finalization:
 * - Approved: 100% returned
 * - Rejected (no challenge): 100% returned
 * - Rejected (challenged): 77% returned
 */
export function calculateEndorsementStake(
  parentDomainId: string,
  depth: number,
): bigint {
  // Top-level domains (direct children of root)
  if (parentDomainId === "root" || depth === 1) {
    return DOMAIN_PROPOSAL_CONFIG.topLevelEndorsementStake; // 222 LGM
  }

  // Mid-level domains (depth 2)
  if (depth === 2) {
    return DOMAIN_PROPOSAL_CONFIG.midEndorsementStake; // 22 LGM
  }

  // Leaf domains (depth > 2)
  return DOMAIN_PROPOSAL_CONFIG.leafEndorsementStake; // 2 LGM
}

/**
 * Determine if a domain proposal requires governance approval
 */
export function requiresGovernance(
  stratum: OntologicalStratum,
  parentDomainId: string,
  depth: number,
): boolean {
  // Fundamental strata always require governance
  const stratumLevel = STRATUM_HIERARCHY[stratum];
  if (stratumLevel <= DOMAIN_PROPOSAL_CONFIG.governanceRequiredStratum) {
    return true;
  }

  // Top-level domains require governance
  if (
    parentDomainId === "root" ||
    depth <= DOMAIN_PROPOSAL_CONFIG.governanceRequiredDepth
  ) {
    return true;
  }

  return false;
}

/**
 * Validate a domain proposal input
 */
export function validateDomainProposal(
  input: DomainProposalInput,
  domainTree: DomainTree,
): DomainProposalValidation {
  const errors: string[] = [];

  // 1. Validate domain ID format
  if (!isValidDomainId(input.domainId)) {
    errors.push(
      "Invalid domain ID format. Use lowercase alphanumeric with dots (e.g., physics.quantum.entanglement)",
    );
  }

  // 2. Check domain doesn't already exist
  if (domainTree.domainIndex.has(input.domainId)) {
    errors.push(`Domain "${input.domainId}" already exists`);
  }

  // 3. Check parent domain exists
  const parentNode = domainTree.domainIndex.get(input.parentDomainId);
  if (input.parentDomainId !== "root" && !parentNode) {
    errors.push(`Parent domain "${input.parentDomainId}" does not exist`);
  }

  // 4. Calculate depth
  let depth = 1;
  if (parentNode) {
    depth = parentNode.depth + 1;
  }

  // 5. Check max depth
  if (depth > DOMAIN_PROPOSAL_CONFIG.maxDepth) {
    errors.push(
      `Domain depth ${depth} exceeds maximum of ${DOMAIN_PROPOSAL_CONFIG.maxDepth}`,
    );
  }

  // 6. Check max children
  if (
    parentNode &&
    parentNode.children.length >= DOMAIN_PROPOSAL_CONFIG.maxChildrenPerDomain
  ) {
    errors.push(
      `Parent domain "${input.parentDomainId}" has reached maximum of ${DOMAIN_PROPOSAL_CONFIG.maxChildrenPerDomain} children`,
    );
  }

  // 7. Validate stratum consistency
  if (parentNode && !DOMAIN_PROPOSAL_CONFIG.crossStratumDomains) {
    if (parentNode.domain.stratum !== input.stratum) {
      errors.push(
        `Domain stratum "${input.stratum}" must match parent stratum "${parentNode.domain.stratum}"`,
      );
    }
  }

  // 8. Validate domain ID matches parent hierarchy
  if (input.parentDomainId !== "root") {
    if (!input.domainId.startsWith(input.parentDomainId + ".")) {
      errors.push(
        `Domain ID "${input.domainId}" must start with parent "${input.parentDomainId}."`,
      );
    }
  }

  // 9. Validate examples (minimum 2)
  if (input.examples.length < 2) {
    errors.push("At least 2 example claims are required");
  }

  // 10. Calculate required stake
  const requiredStake = calculateRequiredStake(input.parentDomainId, depth);
  if (input.stake < requiredStake) {
    errors.push(
      `Insufficient stake: ${input.stake} < required ${requiredStake}`,
    );
  }

  // 11. Determine if governance is required
  const needsGovernance = requiresGovernance(
    input.stratum,
    input.parentDomainId,
    depth,
  );

  return {
    valid: errors.length === 0,
    errors,
    requiredStake,
    requiresGovernance: needsGovernance,
    depth,
  };
}

/**
 * Validate domain ID format
 */
function isValidDomainId(domainId: string): boolean {
  // Must be lowercase alphanumeric with dots, no leading/trailing dots
  const pattern = /^[a-z][a-z0-9]*(\.[a-z][a-z0-9]*)*$/;
  return pattern.test(domainId);
}

// ============================================================================
// Proposal Creation
// ============================================================================

/**
 * Create a new domain proposal
 */
export function createDomainProposal(
  input: DomainProposalInput,
  currentBlock: bigint,
  domainTree: DomainTree,
): { proposal: DomainProposal; validation: DomainProposalValidation } {
  // Validate input
  const validation = validateDomainProposal(input, domainTree);

  if (!validation.valid) {
    throw new Error(`Invalid domain proposal: ${validation.errors.join(", ")}`);
  }

  // Generate proposal ID
  const proposalId = sha512_256Hex(
    `domain-proposal:${input.proposer}:${input.domainId}:${currentBlock}`,
  );

  // Calculate challenge deadline
  const challengeDeadline =
    currentBlock + DOMAIN_PROPOSAL_CONFIG.challengePeriodBlocks;

  // Determine initial status
  const status: DomainProposalStatus = validation.requiresGovernance
    ? "governance"
    : "pending";

  // Calculate required endorsement stake based on depth
  const requiredEndorsementStake = calculateEndorsementStake(
    input.parentDomainId,
    validation.depth,
  );

  const proposal: DomainProposal = {
    id: proposalId,
    proposedDomainId: input.domainId,
    name: input.name,
    stratum: input.stratum,
    parentDomainId: input.parentDomainId,
    description: input.description,
    examples: input.examples,
    proposer: input.proposer,
    stake: input.stake,
    status,
    submittedAt: currentBlock,
    challengeDeadline,
    endorsers: [],
    endorserStakes: new Map(),
    requiredEndorsementStake,
    challengers: [],
    challengerStakes: new Map(),
    challengerReasons: new Map(),
    depth: validation.depth,
  };

  return { proposal, validation };
}

// ============================================================================
// Proposal State Transitions
// ============================================================================

/**
 * Add endorsement to a proposal
 *
 * Validators must lock stake when endorsing:
 * - Top-level domains: 222 LGM
 * - Mid-level domains: 22 LGM
 * - Leaf domains: 2 LGM
 *
 * @param proposal - The proposal to endorse
 * @param endorser - DID of the endorsing validator
 * @param stake - Stake amount being locked (must meet minimum)
 * @param currentBlock - Current block number
 */
export function endorseProposal(
  proposal: DomainProposal,
  endorser: string,
  currentBlock: bigint,
  stake?: bigint,
): DomainProposal {
  // Cannot endorse already-decided proposals
  if (proposal.status !== "pending" && proposal.status !== "endorsed") {
    throw new Error(`Cannot endorse proposal with status "${proposal.status}"`);
  }

  // Cannot endorse after deadline
  if (currentBlock > proposal.challengeDeadline) {
    throw new Error("Challenge period has ended");
  }

  // Cannot endorse own proposal
  if (endorser === proposal.proposer) {
    throw new Error("Cannot endorse your own proposal");
  }

  // Cannot endorse twice
  if (proposal.endorsers.includes(endorser)) {
    throw new Error("Already endorsed this proposal");
  }

  // Validate stake amount (if provided)
  const endorsementStake = stake ?? proposal.requiredEndorsementStake;
  if (endorsementStake < proposal.requiredEndorsementStake) {
    throw new Error(
      `Insufficient endorsement stake: ${endorsementStake} < required ${proposal.requiredEndorsementStake}`,
    );
  }

  // Add endorsement
  const updatedEndorsers = [...proposal.endorsers, endorser];
  const updatedEndorserStakes = new Map(proposal.endorserStakes);
  updatedEndorserStakes.set(endorser, endorsementStake);

  // Check if we have minimum supporters
  const newStatus: DomainProposalStatus =
    updatedEndorsers.length >= DOMAIN_PROPOSAL_CONFIG.minimumSupporters
      ? "endorsed"
      : "pending";

  return {
    ...proposal,
    endorsers: updatedEndorsers,
    endorserStakes: updatedEndorserStakes,
    status: newStatus,
  };
}

/**
 * Challenge a proposal
 */
export function challengeProposal(
  proposal: DomainProposal,
  challenger: string,
  reason: string,
  stake: bigint,
  currentBlock: bigint,
): DomainProposal {
  // Cannot challenge already-decided proposals
  if (proposal.status === "approved" || proposal.status === "rejected") {
    throw new Error(
      `Cannot challenge proposal with status "${proposal.status}"`,
    );
  }

  // Cannot challenge after deadline
  if (currentBlock > proposal.challengeDeadline) {
    throw new Error("Challenge period has ended");
  }

  // Cannot challenge own proposal
  if (challenger === proposal.proposer) {
    throw new Error("Cannot challenge your own proposal");
  }

  // Cannot challenge twice
  if (proposal.challengers.includes(challenger)) {
    throw new Error("Challenger has already challenged this proposal");
  }

  // Add challenger with stake and reason
  const updatedChallengers = [...proposal.challengers, challenger];
  const updatedChallengerStakes = new Map(proposal.challengerStakes);
  updatedChallengerStakes.set(challenger, stake);
  const updatedChallengerReasons = new Map(proposal.challengerReasons);
  updatedChallengerReasons.set(challenger, reason);

  return {
    ...proposal,
    challengers: updatedChallengers,
    challengerStakes: updatedChallengerStakes,
    challengerReasons: updatedChallengerReasons,
  };
}

/**
 * Finalize a proposal after challenge period
 */
export function finalizeProposal(
  proposal: DomainProposal,
  currentBlock: bigint,
): DomainProposal {
  // Must be past deadline
  if (currentBlock <= proposal.challengeDeadline) {
    throw new Error("Challenge period has not ended");
  }

  // Already finalized
  if (
    proposal.status === "approved" ||
    proposal.status === "rejected" ||
    proposal.status === "expired"
  ) {
    return proposal;
  }

  // Governance proposals handled separately
  if (proposal.status === "governance") {
    return proposal;
  }

  // Check approval conditions
  const hasMinEndorsers =
    proposal.endorsers.length >= DOMAIN_PROPOSAL_CONFIG.minimumSupporters;
  const hasChallenges = proposal.challengers.length > 0;

  // Approval requires:
  // 1. Minimum endorsers (2)
  // 2. No challenges OR more endorsers than challengers
  const approved =
    hasMinEndorsers &&
    (!hasChallenges || proposal.endorsers.length > proposal.challengers.length);

  if (approved) {
    return {
      ...proposal,
      status: "approved",
      approvedAt: currentBlock,
    };
  } else if (hasChallenges) {
    // Actively rejected by challengers
    return {
      ...proposal,
      status: "rejected",
      rejectionReason:
        "Proposal was challenged and did not receive sufficient endorsement",
    };
  } else {
    // No challenges, just insufficient endorsements — expired
    return {
      ...proposal,
      status: "expired",
      rejectionReason:
        "Proposal did not receive minimum required endorsements before deadline",
    };
  }
}

/**
 * Calculate stake return for a rejected proposal (proposer)
 */
export function calculateStakeReturn(proposal: DomainProposal): bigint {
  if (proposal.status !== "rejected") {
    return 0n;
  }

  // Return 77% of stake
  // Use BPS to avoid float*100 rounding fragility
  const returnRateBps = BigInt(
    Math.round(DOMAIN_PROPOSAL_CONFIG.stakeReturnOnRejection * 10000),
  );
  return (proposal.stake * returnRateBps) / 10000n;
}

/**
 * Calculate stake returns for all endorsers
 *
 * Return rules:
 * - Approved: 100% returned to all endorsers
 * - Rejected (challenged): 77% returned to all endorsers
 * - Expired (no challenges): 100% returned to all endorsers
 */
export function calculateEndorserStakeReturns(
  proposal: DomainProposal,
): Map<string, { stakeReturn: bigint; stakeSlashed: bigint }> {
  const returns = new Map<
    string,
    { stakeReturn: bigint; stakeSlashed: bigint }
  >();

  for (const [endorser, stake] of proposal.endorserStakes) {
    if (proposal.status === "approved" || proposal.status === "expired") {
      // Full return on approval or expiration (no fault)
      returns.set(endorser, { stakeReturn: stake, stakeSlashed: 0n });
    } else if (proposal.status === "rejected") {
      // 77% return when rejected after challenge
      // Use BPS to avoid float*100 rounding fragility
      const returnRateBps = BigInt(
        Math.round(DOMAIN_PROPOSAL_CONFIG.stakeReturnOnRejection * 10000),
      );
      const stakeReturn = (stake * returnRateBps) / 10000n;
      const stakeSlashed = stake - stakeReturn;
      returns.set(endorser, { stakeReturn, stakeSlashed });
    }
  }

  return returns;
}

/**
 * Calculate stake returns for all challengers
 *
 * Return rules:
 * - Approved (challengers lost): 77% returned to challengers (slashed for incorrect challenge)
 * - Rejected (challengers won): 100% returned to challengers
 * - Expired: 100% returned to challengers
 */
export function calculateChallengerStakeReturns(
  proposal: DomainProposal,
): Map<string, { stakeReturn: bigint; stakeSlashed: bigint }> {
  const returns = new Map<
    string,
    { stakeReturn: bigint; stakeSlashed: bigint }
  >();

  for (const [challenger, stake] of proposal.challengerStakes) {
    if (proposal.status === "rejected" || proposal.status === "expired") {
      // Challengers won or proposal expired — full return
      returns.set(challenger, { stakeReturn: stake, stakeSlashed: 0n });
    } else if (proposal.status === "approved") {
      // Challengers lost — 77% return (same slashing rate)
      // Use BPS to avoid float*100 rounding fragility
      const returnRateBps = BigInt(
        Math.round(DOMAIN_PROPOSAL_CONFIG.stakeReturnOnRejection * 10000),
      );
      const stakeReturn = (stake * returnRateBps) / 10000n;
      const stakeSlashed = stake - stakeReturn;
      returns.set(challenger, { stakeReturn, stakeSlashed });
    }
  }

  return returns;
}

/**
 * Get total endorsement stake locked in a proposal
 */
export function getTotalEndorsementStake(proposal: DomainProposal): bigint {
  let total = 0n;
  for (const stake of proposal.endorserStakes.values()) {
    total += stake;
  }
  return total;
}

// ============================================================================
// Domain Creation from Approved Proposal
// ============================================================================

/**
 * Create domain from an approved proposal
 */
export function createDomainFromProposal(
  proposal: DomainProposal,
  parentDomain: OntologicalDomain | null,
  createdAtBlock: number,
): OntologicalDomain {
  if (proposal.status !== "approved") {
    throw new Error("Can only create domain from approved proposal");
  }

  // Determine dependencies from parent
  const dependsOnStrata: OntologicalStratum[] = [];
  if (parentDomain) {
    dependsOnStrata.push(...parentDomain.dependsOnStrata);
    // Add parent's stratum if not already included and lower
    const parentLevel = STRATUM_HIERARCHY[parentDomain.stratum];
    const currentLevel = STRATUM_HIERARCHY[proposal.stratum];
    if (
      parentLevel < currentLevel &&
      !dependsOnStrata.includes(parentDomain.stratum)
    ) {
      dependsOnStrata.push(parentDomain.stratum);
    }
  }

  return {
    id: proposal.proposedDomainId,
    name: proposal.name,
    stratum: proposal.stratum,
    parentDomain:
      proposal.parentDomainId === "root" ? null : proposal.parentDomainId,
    childDomains: [],
    dependsOnStrata,
    logicZoneDefault: STRATUM_LOGIC_ZONE[proposal.stratum],
    epistemicCategoryDefault: STRATUM_EPISTEMIC_CATEGORY[proposal.stratum],
    description: proposal.description,
    examples: proposal.examples,
    factCount: 0,
    axiomCount: 0,
    createdAt: createdAtBlock,
    governanceProposal: proposal.governanceProposalId,
  };
}

// ============================================================================
// Pending Domain Claim Rejection
// ============================================================================

/**
 * Check if a domain is pending (has an active proposal)
 */
export function isDomainPending(
  domainId: string,
  pendingProposals: Map<string, DomainProposal>,
): boolean {
  for (const proposal of pendingProposals.values()) {
    if (
      proposal.proposedDomainId === domainId &&
      (proposal.status === "pending" ||
        proposal.status === "endorsed" ||
        proposal.status === "governance")
    ) {
      return true;
    }
  }
  return false;
}

/**
 * Get the pending proposal for a domain (if any)
 */
export function getPendingProposal(
  domainId: string,
  pendingProposals: Map<string, DomainProposal>,
): DomainProposal | undefined {
  for (const proposal of pendingProposals.values()) {
    if (
      proposal.proposedDomainId === domainId &&
      (proposal.status === "pending" ||
        proposal.status === "endorsed" ||
        proposal.status === "governance")
    ) {
      return proposal;
    }
  }
  return undefined;
}

// ============================================================================
// Exports
// ============================================================================
