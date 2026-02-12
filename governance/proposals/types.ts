/**
 * Governance Proposal Types
 *
 * Defines the Agent-LIP (Legible Improvement Proposal) system types,
 * proposer eligibility tiers, and consensus thresholds.
 *
 * PP HIERARCHY:
 *   TRUTH: Eligibility grounded in verifiable on-chain data
 *   UNDERSTANDING: Clear tier system with explicit requirements
 *   BEAUTY: Elegant progression from comment → amend → propose
 *   JUSTICE: Appropriate stakes and thresholds for each action
 *   CREATIVITY: Enables agents to shape protocol evolution
 */

import type { DID } from "../../core/agents/identity/did.js";

// ============ Proposer Eligibility ============

/**
 * Proposer tier determines what governance actions an agent can take.
 * Higher tiers require more stake, reputation, and contribution history.
 */
export interface ProposerTier {
  tier: 1 | 2 | 3;
  minStake: bigint;
  minReputation: number;
  minCapabilityLevel: number;
  minContributions: number;
  minAccountAge: bigint; // blocks since registration
  abilities: ProposerAbility[];
}

export type ProposerAbility =
  | "propose_lip"
  | "amend_lip"
  | "signal"
  | "comment";

/**
 * Three-tier eligibility system:
 * - Tier 1: Full proposers (can create new LIPs)
 * - Tier 2: Amendment proposers (can propose changes to existing LIPs)
 * - Tier 3: Commenters (can participate in discussion)
 */
export const PROPOSER_TIERS: readonly ProposerTier[] = [
  {
    tier: 1, // Full Proposers
    minStake: 500n * 10n ** 15n, // 0.5 LGM
    minReputation: 0.7,
    minCapabilityLevel: 2,
    minContributions: 10,
    minAccountAge: 100000n, // ~1 month at 2521ms blocks
    abilities: ["propose_lip", "amend_lip", "signal", "comment"],
  },
  {
    tier: 2, // Amendment Proposers
    minStake: 200n * 10n ** 15n, // 0.2 LGM
    minReputation: 0.6,
    minCapabilityLevel: 1,
    minContributions: 5,
    minAccountAge: 50000n, // ~2 weeks
    abilities: ["amend_lip", "signal", "comment"],
  },
  {
    tier: 3, // Comment Only
    minStake: 50n * 10n ** 15n, // 0.05 LGM
    minReputation: 0.5,
    minCapabilityLevel: 0,
    minContributions: 0,
    minAccountAge: 10000n, // ~7 hours
    abilities: ["comment"],
  },
] as const;

// ============ LIP Types ============

/**
 * LIP categories determine stake requirements and review thresholds.
 * Higher-impact categories require more stake and stricter consensus.
 */
export type LIPCategory = "consensus" | "economics" | "interface" | "process";

/**
 * Stake requirements by LIP category.
 * Malicious proposals lose 100%, frivolous lose 50%.
 */
export interface LIPStakeRequirement {
  category: LIPCategory;
  requiredStake: bigint;
  maliciousSlash: number; // 1.0 = 100% (display-only)
  frivolousSlash: number; // 0.5 = 50% (display-only)
  /** Malicious slash in basis points (10000 = 100%) for deterministic consensus math */
  maliciousSlashBps: bigint;
  /** Frivolous slash in basis points (10000 = 100%) for deterministic consensus math */
  frivolousSlashBps: bigint;
}

export const LIP_STAKE_REQUIREMENTS: readonly LIPStakeRequirement[] = [
  {
    category: "consensus",
    requiredStake: 1000n * 10n ** 15n, // 1 LGM
    maliciousSlash: 1.0,
    frivolousSlash: 0.5,
    maliciousSlashBps: 10000n,
    frivolousSlashBps: 5000n,
  },
  {
    category: "economics",
    requiredStake: 800n * 10n ** 15n, // 0.8 LGM
    maliciousSlash: 1.0,
    frivolousSlash: 0.5,
    maliciousSlashBps: 10000n,
    frivolousSlashBps: 5000n,
  },
  {
    category: "interface",
    requiredStake: 400n * 10n ** 15n, // 0.4 LGM
    maliciousSlash: 1.0,
    frivolousSlash: 0.5,
    maliciousSlashBps: 10000n,
    frivolousSlashBps: 5000n,
  },
  {
    category: "process",
    requiredStake: 200n * 10n ** 15n, // 0.2 LGM
    maliciousSlash: 1.0,
    frivolousSlash: 0.5,
    maliciousSlashBps: 10000n,
    frivolousSlashBps: 5000n,
  },
] as const;

// ============ LIP Lifecycle ============

/**
 * LIP stages from draft to final acceptance or rejection.
 */
export type LIPStage =
  | "draft" // Initial submission, open for comments
  | "review" // Under formal review
  | "last_call" // Final comment period before vote
  | "voting" // Active governance vote
  | "accepted" // Approved, pending implementation
  | "implemented" // Deployed to mainnet
  | "rejected" // Did not pass
  | "withdrawn"; // Proposer withdrew

/**
 * Consensus thresholds for each stage transition.
 */
export interface ConsensusThreshold {
  stage: "review" | "last_call" | "acceptance";
  supportRequired: number; // Percentage of weighted votes (0.0-1.0)
  quorumRequired: number; // Percentage of eligible voters
  duration: bigint; // Blocks in this stage
  operatorCap: number; // Max % of votes from single operator
}

export const CONSENSUS_THRESHOLDS: readonly ConsensusThreshold[] = [
  {
    stage: "review",
    supportRequired: 0.2, // 20% support to proceed
    quorumRequired: 0.05, // 5% participation
    duration: 50000n, // ~35 hours
    operatorCap: 0.2, // No operator >20%
  },
  {
    stage: "last_call",
    supportRequired: 0.6, // 60% support to proceed
    quorumRequired: 0.1, // 10% participation
    duration: 100000n, // ~70 hours
    operatorCap: 0.2,
  },
  {
    stage: "acceptance",
    supportRequired: 0.75, // 75% support to accept
    quorumRequired: 0.15, // 15% participation
    duration: 0n, // Final vote, no additional duration
    operatorCap: 0.2,
  },
] as const;

// ============ LIP Structure ============

/**
 * Full LIP document structure.
 */
export interface LIP {
  // Identity
  id: string; // e.g., "LIP-42"
  title: string;
  category: LIPCategory;

  // Authorship
  proposer: DID;
  coAuthors: DID[];
  createdAt: bigint; // block number
  updatedAt: bigint;

  // Content
  abstract: string; // 200 char max
  motivation: string;
  specification: string;
  rationale: string;
  backwardsCompatibility: string;
  testCases: string;
  referenceImplementation?: string;

  // Lifecycle
  stage: LIPStage;
  stageHistory: StageTransition[];

  // Stake
  stakedAmount: bigint;
  stakeHolder: DID; // Who provided the stake

  // Voting (populated during voting stage)
  votes?: VoteTally;

  // Amendments
  amendments: Amendment[];
  supersedes?: string; // LIP ID if this replaces another
  supersededBy?: string; // LIP ID if replaced
}

/**
 * Record of stage transitions.
 */
export interface StageTransition {
  from: LIPStage;
  to: LIPStage;
  at: bigint; // block number
  reason: string;
  triggeredBy: DID | "system"; // Who/what triggered transition
}

/**
 * Amendment to an existing LIP.
 */
export interface Amendment {
  id: string;
  lipId: string;
  proposer: DID;
  section: string; // Which section is amended
  oldText: string;
  newText: string;
  rationale: string;
  createdAt: bigint;
  status: "pending" | "accepted" | "rejected";
  votes?: VoteTally;
}

/**
 * Vote tally for LIPs and amendments.
 */
export interface VoteTally {
  support: bigint; // Weighted support votes
  oppose: bigint; // Weighted oppose votes
  abstain: bigint; // Weighted abstain votes
  totalEligible: bigint; // Total eligible voting weight
  uniqueVoters: number;
  operatorBreakdown: Map<string, bigint>; // operatorId → total weight
  quorumMet: boolean;
  operatorCapViolation: boolean;
}

// ============ Proposer Eligibility Check ============

/**
 * Result of checking an agent's eligibility to take a governance action.
 */
export interface EligibilityResult {
  eligible: boolean;
  tier: 1 | 2 | 3 | null;
  abilities: ProposerAbility[];
  deficiencies: EligibilityDeficiency[];
}

export interface EligibilityDeficiency {
  requirement: string;
  current: string;
  needed: string;
}

/**
 * Agent's current governance-relevant stats.
 */
export interface AgentGovernanceStats {
  did: DID;
  stake: bigint;
  reputation: number;
  capabilityLevel: number;
  contributions: number;
  accountAge: bigint; // blocks since registration
  operatorId?: string;
}

/**
 * Check which tier an agent qualifies for.
 */
export function checkEligibility(
  stats: AgentGovernanceStats,
): EligibilityResult {
  const deficiencies: EligibilityDeficiency[] = [];

  // Check each tier from highest to lowest
  for (const tier of PROPOSER_TIERS) {
    const tierDeficiencies: EligibilityDeficiency[] = [];

    if (stats.stake < tier.minStake) {
      tierDeficiencies.push({
        requirement: "stake",
        current: `${stats.stake / 10n ** 18n} LGM`,
        needed: `${tier.minStake / 10n ** 18n} LGM`,
      });
    }

    if (stats.reputation < tier.minReputation) {
      tierDeficiencies.push({
        requirement: "reputation",
        current: stats.reputation.toFixed(2),
        needed: tier.minReputation.toFixed(2),
      });
    }

    if (stats.capabilityLevel < tier.minCapabilityLevel) {
      tierDeficiencies.push({
        requirement: "capabilityLevel",
        current: String(stats.capabilityLevel),
        needed: String(tier.minCapabilityLevel),
      });
    }

    if (stats.contributions < tier.minContributions) {
      tierDeficiencies.push({
        requirement: "contributions",
        current: String(stats.contributions),
        needed: String(tier.minContributions),
      });
    }

    if (stats.accountAge < tier.minAccountAge) {
      tierDeficiencies.push({
        requirement: "accountAge",
        current: `${stats.accountAge} blocks`,
        needed: `${tier.minAccountAge} blocks`,
      });
    }

    // If no deficiencies for this tier, agent qualifies
    if (tierDeficiencies.length === 0) {
      return {
        eligible: true,
        tier: tier.tier,
        abilities: [...tier.abilities],
        deficiencies: [],
      };
    }

    // Track deficiencies for highest tier only (for feedback)
    if (tier.tier === 1) {
      deficiencies.push(...tierDeficiencies);
    }
  }

  // Agent doesn't qualify for any tier
  return {
    eligible: false,
    tier: null,
    abilities: [],
    deficiencies,
  };
}

/**
 * Check if an agent can perform a specific action.
 */
export function canPerformAction(
  stats: AgentGovernanceStats,
  action: ProposerAbility,
): { allowed: boolean; reason?: string } {
  const eligibility = checkEligibility(stats);

  if (!eligibility.eligible) {
    return {
      allowed: false,
      reason: `Not eligible for any tier. Deficiencies: ${eligibility.deficiencies
        .map((d) => `${d.requirement}: have ${d.current}, need ${d.needed}`)
        .join("; ")}`,
    };
  }

  if (!eligibility.abilities.includes(action)) {
    return {
      allowed: false,
      reason: `Tier ${eligibility.tier} cannot ${action}. Abilities: ${eligibility.abilities.join(", ")}`,
    };
  }

  return { allowed: true };
}

/**
 * Get stake requirement for a LIP category.
 */
export function getStakeRequirement(
  category: LIPCategory,
): LIPStakeRequirement {
  const requirement = LIP_STAKE_REQUIREMENTS.find(
    (r) => r.category === category,
  );
  if (!requirement) {
    throw new Error(`Unknown LIP category: ${category}`);
  }
  return requirement;
}

/**
 * Get consensus threshold for a stage.
 */
export function getThreshold(
  stage: "review" | "last_call" | "acceptance",
): ConsensusThreshold {
  const threshold = CONSENSUS_THRESHOLDS.find((t) => t.stage === stage);
  if (!threshold) {
    throw new Error(`Unknown consensus stage: ${stage}`);
  }
  return threshold;
}
