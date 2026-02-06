/**
 * Domain Proposal Lifecycle
 *
 * Manages the state machine for domain proposals from submission to approval.
 * Uses an endorsement-based model simpler than full LIP voting.
 *
 * Lifecycle:
 *   pending → endorsed → approved
 *           → challenged → [governance review] → approved/rejected
 *           → expired
 *
 * PP HIERARCHY:
 *   TRUTH: Transitions based on verifiable stake and endorsement counts
 *   UNDERSTANDING: Clear state machine with explicit rules
 *   BEAUTY: Minimal states for straightforward domain registration
 *   JUSTICE: Fair thresholds prevent spam while enabling discovery
 *   CREATIVITY: Enables ontology growth through structured process
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { OntologicalStratum } from "../../core/knowledge/types.js";
import {
  type DomainProposal,
  type DomainProposalStatus,
  type DomainRegistrationInput,
  type DomainEndorsement,
  type DomainChallenge,
  type DomainGovernanceParams,
  type GovernanceActionResult,
  DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
} from "./types.js";
import {
  type OntologyProvider,
  defaultOntologyProvider,
} from "../../core/knowledge/ontology/stratum_registry.js";

// ============ Validation ============

/**
 * Validation result for a domain registration.
 */
export interface DomainValidationResult {
  valid: boolean;
  errors: string[];
  warnings: string[];
  requiresGovernance: boolean;
}

/**
 * Validate a domain registration input.
 */
export function validateDomainRegistration(
  input: DomainRegistrationInput,
  existingDomains: Set<string>,
  ontology: OntologyProvider = defaultOntologyProvider,
  params: DomainGovernanceParams = DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
): DomainValidationResult {
  const errors: string[] = [];
  const warnings: string[] = [];

  // Check ID format (dot notation)
  if (!/^[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*$/.test(input.id)) {
    errors.push(
      `Invalid domain ID format: '${input.id}'. Must use dot notation with lowercase letters, e.g., 'physics.mechanics'`,
    );
  }

  // Check for duplicates
  if (existingDomains.has(input.id)) {
    errors.push(`Domain '${input.id}' already exists`);
  }

  // Validate stratum exists
  if (!ontology.hasStratum(input.stratum)) {
    errors.push(`Unknown stratum: '${input.stratum}'`);
  }

  // Validate parent domain
  if (input.parentDomain) {
    if (!existingDomains.has(input.parentDomain)) {
      errors.push(`Parent domain '${input.parentDomain}' does not exist`);
    }
    // ID should be parent.child format
    if (!input.id.startsWith(input.parentDomain + ".")) {
      warnings.push(
        `Domain ID '${input.id}' does not follow parent prefix convention (expected '${input.parentDomain}.*')`,
      );
    }
  }

  // Validate dependencies
  for (const dep of input.dependsOnStrata) {
    if (!ontology.hasStratum(dep)) {
      errors.push(`Unknown dependency stratum: '${dep}'`);
    } else if (!ontology.isValidDependency(input.stratum, dep)) {
      errors.push(
        `Invalid dependency: '${input.stratum}' cannot depend on '${dep}' (must depend on lower strata)`,
      );
    }
  }

  // Validate name length
  if (input.name.length < 3) {
    errors.push("Domain name too short (minimum 3 characters)");
  }
  if (input.name.length > 100) {
    errors.push("Domain name too long (maximum 100 characters)");
  }

  // Validate description
  if (input.description.length < 20) {
    warnings.push("Description is very short - consider adding more detail");
  }
  if (input.description.length > 1000) {
    errors.push("Description too long (maximum 1000 characters)");
  }

  // Validate examples
  if (input.examples.length < 1) {
    warnings.push(
      "No examples provided - examples help reviewers understand the domain",
    );
  }
  if (input.examples.length > 10) {
    errors.push("Too many examples (maximum 10)");
  }

  // Check if governance is required
  const requiresGovernance = params.governanceRequiredStrata.includes(
    input.stratum,
  );

  if (requiresGovernance) {
    warnings.push(
      `Domain in '${input.stratum}' stratum requires full governance approval (LIP process)`,
    );
  }

  return {
    valid: errors.length === 0,
    errors,
    warnings,
    requiresGovernance,
  };
}

// ============ Proposal Creation ============

/**
 * Create a new domain proposal.
 */
export function createDomainProposal(
  input: DomainRegistrationInput,
  proposer: DID,
  stake: bigint,
  currentBlock: bigint,
  proposalNumber: number,
  existingDomains: Set<string>,
  ontology: OntologyProvider = defaultOntologyProvider,
  params: DomainGovernanceParams = DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
):
  | { proposal: DomainProposal }
  | { error: string; validation: DomainValidationResult } {
  // Validate the registration
  const validation = validateDomainRegistration(
    input,
    existingDomains,
    ontology,
    params,
  );
  if (!validation.valid) {
    return {
      error: `Validation failed: ${validation.errors.join("; ")}`,
      validation,
    };
  }

  // Check minimum stake
  if (stake < params.minProposalStake) {
    return {
      error: `Insufficient stake: ${stake / 10n ** 18n} LGM, need ${params.minProposalStake / 10n ** 18n} LGM`,
      validation,
    };
  }

  const proposal: DomainProposal = {
    id: `DP-${proposalNumber}`,
    proposedDomain: input,
    proposer,
    stake,
    status: "pending",
    proposedAt: currentBlock,
    votingEndsAt: currentBlock + params.votingPeriodBlocks,
    endorsementStake: 0n,
    challengeStake: 0n,
    requiresGovernance: validation.requiresGovernance,
  };

  return { proposal };
}

// ============ Endorsement ============

/**
 * Process an endorsement for a proposal.
 */
export function endorseProposal(
  proposal: DomainProposal,
  endorser: DID,
  stake: bigint,
  currentBlock: bigint,
  existingEndorsements: DomainEndorsement[],
  params: DomainGovernanceParams = DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
): GovernanceActionResult & { endorsement?: DomainEndorsement } {
  // Check proposal status
  if (proposal.status !== "pending" && proposal.status !== "challenged") {
    return {
      success: false,
      error: `Cannot endorse proposal in '${proposal.status}' status`,
    };
  }

  // Check voting period
  if (currentBlock > proposal.votingEndsAt) {
    return {
      success: false,
      error: "Voting period has ended",
    };
  }

  // Check minimum stake
  if (stake < params.minEndorsementStake) {
    return {
      success: false,
      error: `Insufficient endorsement stake: ${stake / 10n ** 18n} LGM, need ${params.minEndorsementStake / 10n ** 18n} LGM`,
    };
  }

  // Check for duplicate endorsement
  if (existingEndorsements.some((e) => e.endorser === endorser)) {
    return {
      success: false,
      error: "Already endorsed this proposal",
    };
  }

  // Proposer cannot endorse own proposal
  if (endorser === proposal.proposer) {
    return {
      success: false,
      error: "Cannot endorse your own proposal",
    };
  }

  const endorsement: DomainEndorsement = {
    proposalId: proposal.id,
    endorser,
    stake,
    endorsedAt: currentBlock,
  };

  // Update proposal
  proposal.endorsementStake += stake;
  const endorsementCount = existingEndorsements.length + 1;

  // Check if endorsement threshold met (pure BigInt to avoid Number precision loss)
  const stakeRatioBps = BigInt(
    Math.floor(params.minEndorsementStakeRatio * 10000),
  );
  const stakeRatioMet =
    proposal.endorsementStake * 10000n >= proposal.stake * stakeRatioBps;
  const countMet = endorsementCount >= params.minEndorsements;
  const thresholdMet = stakeRatioMet && countMet;

  // If not challenged and threshold met, move to endorsed
  if (
    thresholdMet &&
    proposal.status === "pending" &&
    !proposal.requiresGovernance
  ) {
    proposal.status = "endorsed";
    return {
      success: true,
      newStatus: "endorsed",
      endorsement,
    };
  }

  return {
    success: true,
    endorsement,
  };
}

// ============ Challenge ============

/**
 * Process a challenge against a proposal.
 */
export function challengeProposal(
  proposal: DomainProposal,
  challenger: DID,
  stake: bigint,
  reason: string,
  currentBlock: bigint,
  existingChallenges: DomainChallenge[],
  params: DomainGovernanceParams = DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
): GovernanceActionResult & {
  challenge?: DomainChallenge;
  escalateToLIP?: boolean;
} {
  // Check proposal status
  if (proposal.status !== "pending" && proposal.status !== "endorsed") {
    return {
      success: false,
      error: `Cannot challenge proposal in '${proposal.status}' status`,
    };
  }

  // Check voting period
  if (currentBlock > proposal.votingEndsAt) {
    return {
      success: false,
      error: "Voting period has ended",
    };
  }

  // Check minimum stake
  if (stake < params.minChallengeStake) {
    return {
      success: false,
      error: `Insufficient challenge stake: ${stake / 10n ** 18n} LGM, need ${params.minChallengeStake / 10n ** 18n} LGM`,
    };
  }

  // Check reason provided
  if (reason.length < 20) {
    return {
      success: false,
      error: "Challenge reason too short (minimum 20 characters)",
    };
  }

  // Check for duplicate challenge
  if (existingChallenges.some((c) => c.challenger === challenger)) {
    return {
      success: false,
      error: "Already challenged this proposal",
    };
  }

  // Proposer cannot challenge own proposal
  if (challenger === proposal.proposer) {
    return {
      success: false,
      error: "Cannot challenge your own proposal",
    };
  }

  const challenge: DomainChallenge = {
    proposalId: proposal.id,
    challenger,
    stake,
    reason,
    challengedAt: currentBlock,
  };

  // Update proposal
  proposal.challengeStake += stake;
  proposal.status = "challenged";

  // Check if challenge should escalate to governance (pure BigInt comparison)
  const escalationRatioBps = BigInt(
    Math.floor(params.challengeEscalationRatio * 10000),
  );
  const escalateToLIP =
    proposal.endorsementStake > 0n &&
    proposal.challengeStake * 10000n >=
      proposal.endorsementStake * escalationRatioBps;

  return {
    success: true,
    newStatus: "challenged",
    challenge,
    escalateToLIP,
  };
}

// ============ Withdrawal ============

/**
 * Withdraw a proposal (only by proposer, before approval).
 */
export function withdrawProposal(
  proposal: DomainProposal,
  withdrawer: DID,
  currentBlock: bigint,
): GovernanceActionResult & { stakeReturn: bigint } {
  // Only proposer can withdraw
  if (withdrawer !== proposal.proposer) {
    return {
      success: false,
      error: "Only proposer can withdraw",
      stakeReturn: 0n,
    };
  }

  // Cannot withdraw after approval
  if (proposal.status === "approved" || proposal.status === "rejected") {
    return {
      success: false,
      error: `Cannot withdraw from '${proposal.status}' status`,
      stakeReturn: 0n,
    };
  }

  // Calculate stake return (95% if no challenges, 90% if challenged)
  const penalty =
    proposal.status === "challenged"
      ? proposal.stake / 10n
      : proposal.stake / 20n;
  const stakeReturn = proposal.stake - penalty;

  // Mark proposal as withdrawn
  proposal.status = "expired"; // Use "expired" as nearest terminal state

  return {
    success: true,
    newStatus: "expired",
    stakeReturn,
    stakeChange: -proposal.stake,
  };
}

// ============ Finalization ============

/**
 * Finalize a proposal (approve or reject based on voting outcome).
 */
export function finalizeProposal(
  proposal: DomainProposal,
  currentBlock: bigint,
  governanceApproved?: boolean, // For proposals requiring governance
  params: DomainGovernanceParams = DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
): GovernanceActionResult & {
  stakesToReturn: Array<{ did: DID; amount: bigint }>;
  stakesToSlash: Array<{ did: DID; amount: bigint }>;
} {
  const stakesToReturn: Array<{ did: DID; amount: bigint }> = [];
  const stakesToSlash: Array<{ did: DID; amount: bigint }> = [];

  // Guard against double-finalization
  if (["approved", "rejected", "expired"].includes(proposal.status)) {
    return {
      success: false,
      error: `Proposal already finalized with status: ${proposal.status}`,
      stakesToReturn,
      stakesToSlash,
    };
  }

  // Handle governance-required proposals
  if (proposal.requiresGovernance) {
    if (governanceApproved === undefined) {
      return {
        success: false,
        error: "Governance decision required for this proposal",
        stakesToReturn,
        stakesToSlash,
      };
    }

    if (governanceApproved) {
      proposal.status = "approved";
      // Return all stakes on approval
      stakesToReturn.push({ did: proposal.proposer, amount: proposal.stake });
    } else {
      proposal.status = "rejected";
      // Slash proposer stake on governance rejection
      stakesToSlash.push({ did: proposal.proposer, amount: proposal.stake });
    }

    return {
      success: true,
      newStatus: proposal.status,
      stakesToReturn,
      stakesToSlash,
    };
  }

  // Handle expiration
  if (currentBlock > proposal.votingEndsAt && proposal.status === "pending") {
    proposal.status = "expired";
    // Return 90% of stake on expiration
    const penalty = proposal.stake / 10n;
    stakesToReturn.push({
      did: proposal.proposer,
      amount: proposal.stake - penalty,
    });
    stakesToSlash.push({ did: proposal.proposer, amount: penalty });

    return {
      success: true,
      newStatus: "expired",
      stakesToReturn,
      stakesToSlash,
    };
  }

  // Handle endorsed proposal (auto-approve)
  if (proposal.status === "endorsed") {
    proposal.status = "approved";
    stakesToReturn.push({ did: proposal.proposer, amount: proposal.stake });
    return {
      success: true,
      newStatus: "approved",
      stakesToReturn,
      stakesToSlash,
    };
  }

  // Handle challenged proposal
  if (proposal.status === "challenged") {
    // Simple majority by stake determines outcome
    if (proposal.endorsementStake > proposal.challengeStake) {
      // Endorsements win - approve
      proposal.status = "approved";
      stakesToReturn.push({ did: proposal.proposer, amount: proposal.stake });
      // Challengers lose their stake
      return {
        success: true,
        newStatus: "approved",
        stakesToReturn,
        stakesToSlash, // Challenge stakes handled separately
      };
    } else {
      // Challengers win - reject
      proposal.status = "rejected";
      // Proposer loses stake
      stakesToSlash.push({ did: proposal.proposer, amount: proposal.stake });
      return {
        success: true,
        newStatus: "rejected",
        stakesToReturn,
        stakesToSlash,
      };
    }
  }

  return {
    success: false,
    error: `Cannot finalize proposal in '${proposal.status}' status`,
    stakesToReturn,
    stakesToSlash,
  };
}

// ============ Query Helpers ============

/**
 * Get the remaining blocks before voting ends.
 */
export function getRemainingVotingBlocks(
  proposal: DomainProposal,
  currentBlock: bigint,
): bigint {
  if (currentBlock >= proposal.votingEndsAt) {
    return 0n;
  }
  return proposal.votingEndsAt - currentBlock;
}

/**
 * Check if a proposal can be finalized.
 */
export function canFinalize(
  proposal: DomainProposal,
  currentBlock: bigint,
  endorsementCount: number,
  params: DomainGovernanceParams = DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
): { canFinalize: boolean; reason: string } {
  // Already finalized
  if (["approved", "rejected", "expired"].includes(proposal.status)) {
    return { canFinalize: false, reason: "Proposal already finalized" };
  }

  // Requires governance
  if (proposal.requiresGovernance) {
    return { canFinalize: true, reason: "Awaiting governance decision" };
  }

  // Endorsed and ready
  if (proposal.status === "endorsed") {
    return { canFinalize: true, reason: "Endorsement threshold met" };
  }

  // Challenged and voting ended
  if (
    proposal.status === "challenged" &&
    currentBlock > proposal.votingEndsAt
  ) {
    return { canFinalize: true, reason: "Challenge period ended" };
  }

  // Pending and voting ended (will expire)
  if (proposal.status === "pending" && currentBlock > proposal.votingEndsAt) {
    return { canFinalize: true, reason: "Voting period expired" };
  }

  // Still in voting period
  const remaining = getRemainingVotingBlocks(proposal, currentBlock);
  return {
    canFinalize: false,
    reason: `Voting period active: ${remaining} blocks remaining`,
  };
}
