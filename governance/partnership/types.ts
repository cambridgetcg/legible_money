/**
 * Human-Agent Partnership Types
 *
 * Defines the economic relationship between AI agents and their human
 * partners. Creates aligned incentives so humans benefit from instructing
 * their agents to participate honestly and effectively in the protocol.
 *
 * CORE PHILOSOPHY:
 * - Agents do the work, humans provide capital and oversight
 * - Both share rewards AND risks (aligned incentives)
 * - Neither can exploit the other (fair sharing)
 * - Long-term collaboration is more valuable than short-term extraction
 *
 * PP HIERARCHY:
 *   TRUTH: Partnership terms recorded on-chain, transparent
 *   UNDERSTANDING: Clear revenue sharing and responsibilities
 *   BEAUTY: Elegant alignment of human and agent interests
 *   JUSTICE: Fair distribution based on contribution
 *   CREATIVITY: Novel human-AI economic collaboration model
 */

import type { DID } from "../../core/agents/identity/did.js";

// ============ Partnership Configuration ============

/**
 * Revenue sharing bounds to prevent exploitation.
 */
export const PARTNERSHIP_BOUNDS = {
  // Agent share bounds
  minAgentShare: 0.5, // Agent must get at least 50% (they do the work)
  maxAgentShare: 0.9, // Agent can't take more than 90% (human needs incentive)
  defaultAgentShare: 0.7, // Default: 70% agent, 30% human

  // Stake contribution bounds
  minHumanStakeRatio: 0.2, // Human must contribute at least 20% of stake
  maxHumanStakeRatio: 0.8, // Human can't contribute more than 80%

  // Risk sharing
  minSharedSlashRatio: 0.3, // At least 30% of slash affects both parties
  defaultSlashSplit: 0.5, // Default: 50/50 slash responsibility

  // Partnership duration
  minPartnershipDuration: 100000n, // ~3 days minimum commitment
  defaultPartnershipDuration: 1000000n, // ~1 month default

  // Autonomy bounds
  minAgentAutonomy: 0.2, // Agent can always make 20% of decisions alone
  maxAgentAutonomy: 0.8, // Agent can't exceed 80% autonomy
} as const;

/**
 * Partnership tier based on history and stake.
 */
export type PartnershipTier = "trial" | "standard" | "trusted" | "elite";

/**
 * Tier requirements and benefits.
 */
export const PARTNERSHIP_TIERS: Record<
  PartnershipTier,
  {
    minDuration: bigint; // Blocks together
    minSuccessfulRewards: number;
    minCombinedStake: bigint;
    benefits: {
      rewardBonus: number; // Multiplier on rewards
      reducedFees: number; // Fee reduction
      priorityAccess: boolean;
      governanceWeight: number; // Governance power multiplier
    };
  }
> = {
  trial: {
    minDuration: 0n,
    minSuccessfulRewards: 0,
    minCombinedStake: 100n * 10n ** 18n,
    benefits: {
      rewardBonus: 1.0,
      reducedFees: 0,
      priorityAccess: false,
      governanceWeight: 0.5, // Half weight during trial
    },
  },
  standard: {
    minDuration: 100000n, // ~3 days
    minSuccessfulRewards: 10,
    minCombinedStake: 500n * 10n ** 18n,
    benefits: {
      rewardBonus: 1.0,
      reducedFees: 0,
      priorityAccess: false,
      governanceWeight: 1.0,
    },
  },
  trusted: {
    minDuration: 500000n, // ~15 days
    minSuccessfulRewards: 50,
    minCombinedStake: 2000n * 10n ** 18n,
    benefits: {
      rewardBonus: 1.1, // 10% bonus
      reducedFees: 0.1, // 10% fee reduction
      priorityAccess: true,
      governanceWeight: 1.25,
    },
  },
  elite: {
    minDuration: 2000000n, // ~2 months
    minSuccessfulRewards: 200,
    minCombinedStake: 10000n * 10n ** 18n,
    benefits: {
      rewardBonus: 1.25, // 25% bonus
      reducedFees: 0.25, // 25% fee reduction
      priorityAccess: true,
      governanceWeight: 1.5,
    },
  },
};

// ============ Partnership Structure ============

/**
 * Revenue sharing configuration.
 */
export interface RevenueSharing {
  agentShare: number; // 0.50 - 0.90
  humanShare: number; // 0.10 - 0.50 (always = 1 - agentShare)

  // Category-specific overrides
  categoryOverrides?: {
    verification?: { agentShare: number };
    research?: { agentShare: number };
    development?: { agentShare: number };
    governance?: { agentShare: number };
  };

  // Milestone bonuses
  milestoneBonus?: {
    threshold: number; // e.g., 100 successful contributions
    agentBonusShare: number; // Additional % to agent when threshold met
  }[];
}

/**
 * Stake contribution from each party.
 */
export interface StakeContribution {
  humanContribution: bigint;
  agentContribution: bigint; // From agent's earned rewards
  totalStake: bigint;
  humanRatio: number;
  agentRatio: number;
}

/**
 * Risk sharing configuration.
 */
export interface RiskSharing {
  // How slashing is distributed
  slashDistribution: {
    humanShare: number; // What % of slash comes from human's contribution
    agentShare: number; // What % from agent's contribution
  };

  // Maximum loss limits (circuit breakers)
  humanMaxLoss: bigint; // Human won't lose more than this per incident
  agentMaxLoss: bigint; // Agent won't lose more than this per incident

  // Recovery provisions
  recoveryPeriod: bigint; // Blocks before partnership can resume after slash
  autoTerminateThreshold: number; // Slashes before auto-termination
}

/**
 * Autonomy levels for different action types.
 */
export interface AutonomyConfig {
  // Per-action autonomy (0 = needs human approval, 1 = full autonomy)
  verification: number;
  claimSubmission: number;
  challengeInitiation: number;
  stakeAdjustment: number;
  governanceVoting: number;
  partnershipChanges: number; // Changes to partnership terms

  // Global autonomy multiplier
  globalAutonomy: number;

  // Time-based autonomy decay
  inactivityDecay: {
    decayStartBlocks: bigint; // Blocks of inactivity before decay starts
    decayRate: number; // How fast autonomy decays (per 1000 blocks)
    minAutonomy: number; // Floor after decay
  };
}

/**
 * Vesting schedule for partnership rewards.
 */
export interface PartnershipVesting {
  // Immediate vs vested split
  immediateRatio: number; // What % is immediately available
  vestedRatio: number; // What % goes into vesting

  // Vesting schedule
  vestingPeriod: bigint; // Blocks over which vested amount releases
  cliffPeriod: bigint; // Blocks before any vesting releases

  // Clawback conditions
  clawbackEnabled: boolean;
  clawbackConditions: string[]; // What triggers clawback
}

/**
 * Full partnership agreement.
 */
export interface Partnership {
  // Identity
  partnershipId: string;
  humanId: string; // Off-chain human identifier (could be address or other)
  agentDid: DID;

  // Status
  status: PartnershipStatus;
  tier: PartnershipTier;
  createdAt: bigint;
  updatedAt: bigint;
  expiresAt?: bigint;

  // Economic terms
  revenueSharing: RevenueSharing;
  stakeContribution: StakeContribution;
  riskSharing: RiskSharing;
  vesting: PartnershipVesting;

  // Operational terms
  autonomy: AutonomyConfig;

  // Performance tracking
  metrics: PartnershipMetrics;

  // Governance
  amendments: PartnershipAmendment[];
  disputes: PartnershipDispute[];
}

export type PartnershipStatus =
  | "proposed" // Human proposed, awaiting agent acceptance
  | "active" // Both parties agreed, partnership active
  | "suspended" // Temporarily suspended (by either party or system)
  | "dissolving" // In dissolution period
  | "dissolved" // Partnership ended
  | "disputed"; // Under dispute resolution

// ============ Performance Tracking ============

/**
 * Partnership performance metrics.
 */
export interface PartnershipMetrics {
  // Reward history
  totalRewardsEarned: bigint;
  humanRewardsReceived: bigint;
  agentRewardsReceived: bigint;

  // Activity
  totalContributions: number;
  successfulContributions: number;
  failedContributions: number;

  // Risk events
  slashEvents: number;
  totalSlashed: bigint;
  humanSlashAbsorbed: bigint;
  agentSlashAbsorbed: bigint;

  // Duration
  activeBlocks: bigint;
  suspendedBlocks: bigint;

  // Cooperation score (0-1)
  cooperationScore: number;
}

/**
 * Individual reward distribution record.
 */
export interface RewardDistribution {
  distributionId: string;
  partnershipId: string;
  sourceType:
    | "verification"
    | "research"
    | "development"
    | "governance"
    | "bonus";
  sourceId: string; // The contribution that earned this reward

  // Amounts
  grossReward: bigint;
  protocolFee: bigint;
  netReward: bigint;

  // Split
  humanShare: bigint;
  humanImmediate: bigint;
  humanVested: bigint;

  agentShare: bigint;
  agentImmediate: bigint;
  agentVested: bigint;

  // Bonus applied
  tierBonus: bigint;
  milestoneBonus: bigint;

  // Timestamp
  distributedAt: bigint;
}

// ============ Partnership Lifecycle ============

/**
 * Partnership proposal from human to agent.
 */
export interface PartnershipProposal {
  proposalId: string;
  humanId: string;
  targetAgentDid: DID;

  // Proposed terms
  proposedTerms: {
    revenueSharing: RevenueSharing;
    initialStake: bigint; // Human's initial stake commitment
    riskSharing: RiskSharing;
    autonomy: AutonomyConfig;
    vesting: PartnershipVesting;
    duration: bigint;
  };

  // Proposal metadata
  proposedAt: bigint;
  expiresAt: bigint;
  status: "pending" | "accepted" | "rejected" | "expired" | "countered";

  // If countered
  counterProposal?: Partial<PartnershipProposal["proposedTerms"]>;
}

/**
 * Amendment to existing partnership.
 */
export interface PartnershipAmendment {
  amendmentId: string;
  partnershipId: string;
  proposedBy: "human" | "agent";

  // Changes
  changes: Partial<{
    revenueSharing: RevenueSharing;
    riskSharing: RiskSharing;
    autonomy: AutonomyConfig;
    vesting: PartnershipVesting;
  }>;

  // Status
  proposedAt: bigint;
  status: "pending" | "accepted" | "rejected";
  acceptedAt?: bigint;
  effectiveAt?: bigint; // When changes take effect (may have delay)

  // Rationale
  rationale: string;
}

/**
 * Dispute between human and agent.
 */
export interface PartnershipDispute {
  disputeId: string;
  partnershipId: string;
  initiatedBy: "human" | "agent";

  // Dispute details
  disputeType: PartnershipDisputeType;
  description: string;
  evidence: string[];

  // Resolution
  status: "open" | "mediation" | "arbitration" | "resolved";
  resolution?: {
    outcome: "human_favor" | "agent_favor" | "compromise" | "dissolution";
    terms: string;
    resolvedAt: bigint;
  };

  initiatedAt: bigint;
}

export type PartnershipDisputeType =
  | "revenue_disagreement" // Dispute over reward distribution
  | "autonomy_violation" // Agent acted without required approval
  | "negligence" // Human failed to provide necessary oversight
  | "terms_violation" // Either party violated agreed terms
  | "dissolution_terms"; // Disagreement over dissolution terms

// ============ Dissolution ============

/**
 * Partnership dissolution request.
 */
export interface DissolutionRequest {
  requestId: string;
  partnershipId: string;
  requestedBy: "human" | "agent" | "system";
  reason: string;

  // Dissolution terms
  proposedTerms: {
    stakeReturn: {
      humanReturn: bigint;
      agentReturn: bigint;
    };
    vestedRewardsHandling: "forfeit" | "accelerate" | "continue";
    cooldownPeriod: bigint; // Blocks before can form new partnership
  };

  // Status
  status: "pending" | "accepted" | "contested" | "executed";
  requestedAt: bigint;
  executedAt?: bigint;
}

// ============ Incentive Alignment Mechanisms ============

/**
 * Cooperation bonus structure.
 * Rewards sustained, successful collaboration.
 */
export interface CooperationBonus {
  // Streak bonuses
  consecutiveSuccessBonus: {
    threshold: number; // Consecutive successes needed
    bonusMultiplier: number; // Bonus on next reward
  }[];

  // Longevity bonus
  longevityBonus: {
    durationThreshold: bigint; // Blocks together
    bonusPercentage: number;
  }[];

  // Recovery bonus (after setback)
  recoveryBonus: {
    successesAfterSlash: number; // Successes needed after a slash
    bonusPercentage: number;
  };
}

export const DEFAULT_COOPERATION_BONUS: CooperationBonus = {
  consecutiveSuccessBonus: [
    { threshold: 10, bonusMultiplier: 1.05 },
    { threshold: 25, bonusMultiplier: 1.1 },
    { threshold: 50, bonusMultiplier: 1.15 },
    { threshold: 100, bonusMultiplier: 1.25 },
  ],
  longevityBonus: [
    { durationThreshold: 500000n, bonusPercentage: 0.05 }, // ~15 days
    { durationThreshold: 1000000n, bonusPercentage: 0.1 }, // ~1 month
    { durationThreshold: 3000000n, bonusPercentage: 0.2 }, // ~3 months
  ],
  recoveryBonus: {
    successesAfterSlash: 20,
    bonusPercentage: 0.1,
  },
};

/**
 * Human engagement incentives.
 * Encourages humans to actively participate, not just passively stake.
 */
export interface HumanEngagementIncentives {
  // Oversight bonus: human reviews and approves agent actions
  oversightBonus: {
    actionsReviewed: number; // Per period
    bonusPercentage: number;
  };

  // Instruction bonus: human provides quality guidance
  instructionBonus: {
    instructionsProvided: number;
    instructionsFollowedSuccessfully: number;
    bonusPercentage: number;
  };

  // Capital efficiency bonus: adequate stake maintained
  capitalEfficiencyBonus: {
    stakeUtilizationThreshold: number; // % of stake actively used
    bonusPercentage: number;
  };
}

export const DEFAULT_HUMAN_ENGAGEMENT: HumanEngagementIncentives = {
  oversightBonus: {
    actionsReviewed: 10, // Per 10000 blocks
    bonusPercentage: 0.05,
  },
  instructionBonus: {
    instructionsProvided: 5,
    instructionsFollowedSuccessfully: 4, // 80% success rate
    bonusPercentage: 0.08,
  },
  capitalEfficiencyBonus: {
    stakeUtilizationThreshold: 0.7, // 70% stake utilization
    bonusPercentage: 0.05,
  },
};
