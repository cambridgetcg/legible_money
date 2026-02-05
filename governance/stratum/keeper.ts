/**
 * Domain Governance Keeper
 *
 * Manages the storage and state of domain proposals, endorsements, and challenges.
 * Integrates with the stratum registry for approved domain registration.
 *
 * PP HIERARCHY:
 *   TRUTH: Single source of truth for domain governance state
 *   UNDERSTANDING: Clear interface for all governance operations
 *   BEAUTY: Clean separation between storage and business logic
 *   JUSTICE: Consistent state management across all operations
 *   CREATIVITY: Enables dynamic ontology evolution through governance
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
  type DomainProposalFilter,
  type DomainGovernanceStats,
  DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
} from "./types.js";
import {
  createDomainProposal,
  endorseProposal,
  challengeProposal,
  withdrawProposal,
  finalizeProposal,
  canFinalize,
  validateDomainRegistration,
} from "./lifecycle.js";
import {
  type OntologyProvider,
  type StratumDefinition,
  StratumRegistry,
  defaultOntologyProvider,
} from "../../core/knowledge/ontology/stratum_registry.js";

// ============ Keeper Interface ============

/**
 * Interface for domain governance state operations.
 */
export interface DomainGovernanceKeeper {
  // Proposals
  getProposal(id: string): DomainProposal | undefined;
  getProposals(filter?: DomainProposalFilter): DomainProposal[];
  getProposalCount(): number;

  // Endorsements
  getEndorsements(proposalId: string): DomainEndorsement[];
  getEndorsementsByEndorser(endorser: DID): DomainEndorsement[];

  // Challenges
  getChallenges(proposalId: string): DomainChallenge[];
  getChallengesByChallenger(challenger: DID): DomainChallenge[];

  // Actions
  submitProposal(
    input: DomainRegistrationInput,
    proposer: DID,
    stake: bigint,
    currentBlock: bigint,
  ): { proposal: DomainProposal } | { error: string };

  submitEndorsement(
    proposalId: string,
    endorser: DID,
    stake: bigint,
    currentBlock: bigint,
  ): { endorsement: DomainEndorsement } | { error: string };

  submitChallenge(
    proposalId: string,
    challenger: DID,
    stake: bigint,
    reason: string,
    currentBlock: bigint,
  ):
    | { challenge: DomainChallenge; escalateToLIP?: boolean }
    | { error: string };

  withdraw(
    proposalId: string,
    withdrawer: DID,
    currentBlock: bigint,
  ): { stakeReturn: bigint } | { error: string };

  finalize(
    proposalId: string,
    currentBlock: bigint,
    governanceApproved?: boolean,
  ):
    | {
        status: DomainProposalStatus;
        stakesToReturn: Array<{ did: DID; amount: bigint }>;
        stakesToSlash: Array<{ did: DID; amount: bigint }>;
      }
    | { error: string };

  // Stats
  getStats(): DomainGovernanceStats;

  // Parameters
  getParams(): DomainGovernanceParams;
  setParams(params: DomainGovernanceParams): void;
}

// ============ In-Memory Keeper Implementation ============

/**
 * In-memory implementation of the domain governance keeper.
 * For production, this would be backed by persistent state (SMT).
 */
export class InMemoryDomainGovernanceKeeper implements DomainGovernanceKeeper {
  private proposals: Map<string, DomainProposal> = new Map();
  private endorsements: Map<string, DomainEndorsement[]> = new Map(); // proposalId → endorsements
  private challenges: Map<string, DomainChallenge[]> = new Map(); // proposalId → challenges
  private proposalCounter: number = 0;
  private params: DomainGovernanceParams;
  private ontology: OntologyProvider;
  private existingDomains: Set<string>;
  private stratumRegistry?: StratumRegistry; // For registering approved domains

  constructor(
    params: DomainGovernanceParams = DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
    ontology: OntologyProvider = defaultOntologyProvider,
    existingDomains: Set<string> = new Set(),
    stratumRegistry?: StratumRegistry,
  ) {
    this.params = { ...params };
    this.ontology = ontology;
    this.existingDomains = existingDomains;
    this.stratumRegistry = stratumRegistry;
  }

  // ============ Queries ============

  getProposal(id: string): DomainProposal | undefined {
    return this.proposals.get(id);
  }

  getProposals(filter?: DomainProposalFilter): DomainProposal[] {
    let proposals = Array.from(this.proposals.values());

    if (filter) {
      if (filter.status) {
        proposals = proposals.filter((p) => p.status === filter.status);
      }
      if (filter.proposer) {
        proposals = proposals.filter((p) => p.proposer === filter.proposer);
      }
      if (filter.stratum) {
        proposals = proposals.filter(
          (p) => p.proposedDomain.stratum === filter.stratum,
        );
      }
      if (filter.parentDomain) {
        proposals = proposals.filter(
          (p) => p.proposedDomain.parentDomain === filter.parentDomain,
        );
      }
      if (filter.minStake !== undefined) {
        proposals = proposals.filter((p) => p.stake >= filter.minStake!);
      }
    }

    return proposals;
  }

  getProposalCount(): number {
    return this.proposals.size;
  }

  getEndorsements(proposalId: string): DomainEndorsement[] {
    return this.endorsements.get(proposalId) || [];
  }

  getEndorsementsByEndorser(endorser: DID): DomainEndorsement[] {
    const result: DomainEndorsement[] = [];
    for (const endorsements of this.endorsements.values()) {
      result.push(...endorsements.filter((e) => e.endorser === endorser));
    }
    return result;
  }

  getChallenges(proposalId: string): DomainChallenge[] {
    return this.challenges.get(proposalId) || [];
  }

  getChallengesByChallenger(challenger: DID): DomainChallenge[] {
    const result: DomainChallenge[] = [];
    for (const challenges of this.challenges.values()) {
      result.push(...challenges.filter((c) => c.challenger === challenger));
    }
    return result;
  }

  // ============ Actions ============

  submitProposal(
    input: DomainRegistrationInput,
    proposer: DID,
    stake: bigint,
    currentBlock: bigint,
  ): { proposal: DomainProposal } | { error: string } {
    this.proposalCounter++;

    const result = createDomainProposal(
      input,
      proposer,
      stake,
      currentBlock,
      this.proposalCounter,
      this.existingDomains,
      this.ontology,
      this.params,
    );

    if ("error" in result) {
      this.proposalCounter--; // Rollback counter
      return { error: result.error };
    }

    this.proposals.set(result.proposal.id, result.proposal);
    this.endorsements.set(result.proposal.id, []);
    this.challenges.set(result.proposal.id, []);

    return { proposal: result.proposal };
  }

  submitEndorsement(
    proposalId: string,
    endorser: DID,
    stake: bigint,
    currentBlock: bigint,
  ): { endorsement: DomainEndorsement } | { error: string } {
    const proposal = this.proposals.get(proposalId);
    if (!proposal) {
      return { error: `Proposal '${proposalId}' not found` };
    }

    const existingEndorsements = this.endorsements.get(proposalId) || [];

    const result = endorseProposal(
      proposal,
      endorser,
      stake,
      currentBlock,
      existingEndorsements,
      this.params,
    );

    if (!result.success) {
      return { error: result.error! };
    }

    if (result.endorsement) {
      existingEndorsements.push(result.endorsement);
      this.endorsements.set(proposalId, existingEndorsements);
    }

    return { endorsement: result.endorsement! };
  }

  submitChallenge(
    proposalId: string,
    challenger: DID,
    stake: bigint,
    reason: string,
    currentBlock: bigint,
  ):
    | { challenge: DomainChallenge; escalateToLIP?: boolean }
    | { error: string } {
    const proposal = this.proposals.get(proposalId);
    if (!proposal) {
      return { error: `Proposal '${proposalId}' not found` };
    }

    const existingChallenges = this.challenges.get(proposalId) || [];

    const result = challengeProposal(
      proposal,
      challenger,
      stake,
      reason,
      currentBlock,
      existingChallenges,
      this.params,
    );

    if (!result.success) {
      return { error: result.error! };
    }

    if (result.challenge) {
      existingChallenges.push(result.challenge);
      this.challenges.set(proposalId, existingChallenges);
    }

    return {
      challenge: result.challenge!,
      escalateToLIP: result.escalateToLIP,
    };
  }

  withdraw(
    proposalId: string,
    withdrawer: DID,
    currentBlock: bigint,
  ): { stakeReturn: bigint } | { error: string } {
    const proposal = this.proposals.get(proposalId);
    if (!proposal) {
      return { error: `Proposal '${proposalId}' not found` };
    }

    const result = withdrawProposal(proposal, withdrawer, currentBlock);

    if (!result.success) {
      return { error: result.error! };
    }

    // Mark proposal as withdrawn by setting status manually after withdrawal
    // The lifecycle function already handles this, but we keep proposal in store for history
    return { stakeReturn: result.stakeReturn };
  }

  finalize(
    proposalId: string,
    currentBlock: bigint,
    governanceApproved?: boolean,
  ):
    | {
        status: DomainProposalStatus;
        stakesToReturn: Array<{ did: DID; amount: bigint }>;
        stakesToSlash: Array<{ did: DID; amount: bigint }>;
      }
    | { error: string } {
    const proposal = this.proposals.get(proposalId);
    if (!proposal) {
      return { error: `Proposal '${proposalId}' not found` };
    }

    const result = finalizeProposal(
      proposal,
      currentBlock,
      governanceApproved,
      this.params,
    );

    if (!result.success) {
      return { error: result.error! };
    }

    // If approved, register the domain
    if (proposal.status === "approved") {
      this.existingDomains.add(proposal.proposedDomain.id);
    }

    return {
      status: proposal.status,
      stakesToReturn: result.stakesToReturn,
      stakesToSlash: result.stakesToSlash,
    };
  }

  // ============ Stats ============

  getStats(): DomainGovernanceStats {
    const proposals = Array.from(this.proposals.values());

    const pendingProposals = proposals.filter(
      (p) =>
        p.status === "pending" ||
        p.status === "challenged" ||
        p.status === "endorsed",
    ).length;

    const approvedProposals = proposals.filter(
      (p) => p.status === "approved",
    ).length;

    const rejectedProposals = proposals.filter(
      (p) => p.status === "rejected" || p.status === "expired",
    ).length;

    const totalStakeLocked = proposals
      .filter(
        (p) =>
          p.status === "pending" ||
          p.status === "challenged" ||
          p.status === "endorsed",
      )
      .reduce(
        (sum, p) => sum + p.stake + p.endorsementStake + p.challengeStake,
        0n,
      );

    const endorsementCounts = proposals.map(
      (p) => this.endorsements.get(p.id)?.length || 0,
    );
    const averageEndorsements =
      endorsementCounts.length > 0
        ? endorsementCounts.reduce((a, b) => a + b, 0) /
          endorsementCounts.length
        : 0;

    const challengedCount = proposals.filter(
      (p) => p.challengeStake > 0n,
    ).length;
    const challengeRate =
      proposals.length > 0 ? challengedCount / proposals.length : 0;

    return {
      totalProposals: proposals.length,
      pendingProposals,
      approvedProposals,
      rejectedProposals,
      totalStakeLocked,
      averageEndorsements,
      challengeRate,
    };
  }

  // ============ Parameters ============

  getParams(): DomainGovernanceParams {
    return { ...this.params };
  }

  setParams(params: DomainGovernanceParams): void {
    this.params = { ...params };
  }

  // ============ Integration Helpers ============

  /**
   * Check if a proposal can be finalized.
   */
  canFinalizeProposal(
    proposalId: string,
    currentBlock: bigint,
  ): { canFinalize: boolean; reason: string } {
    const proposal = this.proposals.get(proposalId);
    if (!proposal) {
      return { canFinalize: false, reason: "Proposal not found" };
    }

    const endorsements = this.endorsements.get(proposalId) || [];
    return canFinalize(
      proposal,
      currentBlock,
      endorsements.length,
      this.params,
    );
  }

  /**
   * Validate a domain registration without submitting.
   */
  validateRegistration(input: DomainRegistrationInput): {
    valid: boolean;
    errors: string[];
    warnings: string[];
    requiresGovernance: boolean;
  } {
    return validateDomainRegistration(
      input,
      this.existingDomains,
      this.ontology,
      this.params,
    );
  }

  /**
   * Get proposals that need attention (can be finalized).
   */
  getActionableProposals(currentBlock: bigint): DomainProposal[] {
    return Array.from(this.proposals.values()).filter((p) => {
      const result = this.canFinalizeProposal(p.id, currentBlock);
      return result.canFinalize;
    });
  }

  /**
   * Process all finalizable proposals.
   */
  processFinalizableProposals(currentBlock: bigint): Array<{
    proposalId: string;
    status: DomainProposalStatus;
    stakesToReturn: Array<{ did: DID; amount: bigint }>;
    stakesToSlash: Array<{ did: DID; amount: bigint }>;
  }> {
    const results: Array<{
      proposalId: string;
      status: DomainProposalStatus;
      stakesToReturn: Array<{ did: DID; amount: bigint }>;
      stakesToSlash: Array<{ did: DID; amount: bigint }>;
    }> = [];

    for (const proposal of this.getActionableProposals(currentBlock)) {
      if (!proposal.requiresGovernance) {
        const result = this.finalize(proposal.id, currentBlock);
        if ("status" in result) {
          results.push({
            proposalId: proposal.id,
            ...result,
          });
        }
      }
    }

    return results;
  }
}

// ============ Factory ============

/**
 * Create a domain governance keeper with default configuration.
 */
export function createDomainGovernanceKeeper(
  params?: Partial<DomainGovernanceParams>,
  ontology?: OntologyProvider,
  existingDomains?: Set<string>,
): DomainGovernanceKeeper {
  const mergedParams: DomainGovernanceParams = {
    ...DEFAULT_DOMAIN_GOVERNANCE_PARAMS,
    ...params,
  };

  return new InMemoryDomainGovernanceKeeper(
    mergedParams,
    ontology || defaultOntologyProvider,
    existingDomains || new Set(),
  );
}
