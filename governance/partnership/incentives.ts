/**
 * Partnership Incentive Alignment
 *
 * Mechanisms to ensure humans and agents work together effectively.
 * Creates positive-sum dynamics where both parties benefit from
 * honest, productive collaboration.
 *
 * KEY INSIGHT:
 * - Humans provide: Capital, oversight, strategic guidance
 * - Agents provide: Labor, consistency, availability, expertise
 * - Both need each other for maximum value capture
 *
 * PP HIERARCHY:
 *   TRUTH: Incentives based on verifiable on-chain outcomes
 *   UNDERSTANDING: Clear reward/penalty mechanisms
 *   BEAUTY: Elegant game-theoretic alignment
 *   JUSTICE: Fair value distribution for contributions
 *   CREATIVITY: Novel cooperative economics
 */

import type { DID } from "../../core/agents/identity/did.js";
import type {
  Partnership,
  PartnershipMetrics,
  CooperationBonus,
  HumanEngagementIncentives,
  DEFAULT_COOPERATION_BONUS,
  DEFAULT_HUMAN_ENGAGEMENT,
} from "./types.js";

// ============ Cooperation Scoring ============

/**
 * Factors that contribute to cooperation score.
 */
export interface CooperationFactors {
  // Success rate (0-1)
  successRate: number;

  // Communication quality
  instructionFollowRate: number; // How often agent follows human instructions
  feedbackResponseRate: number; // How often human responds to agent requests

  // Engagement
  humanEngagementLevel: number; // 0-1, how active is human
  agentAvailability: number; // 0-1, how available is agent

  // Conflict handling
  disputeCount: number;
  amicableResolutionRate: number; // How often disputes resolved without escalation

  // Financial health
  onTimePayments: number; // Human pays stake on time
  sustainedStake: number; // Blocks stake has been maintained
}

/**
 * Calculate overall cooperation score.
 */
export function calculateCooperationScore(factors: CooperationFactors): number {
  const weights = {
    successRate: 0.25,
    instructionFollowRate: 0.15,
    feedbackResponseRate: 0.1,
    humanEngagementLevel: 0.15,
    agentAvailability: 0.1,
    disputeHandling: 0.15,
    financialHealth: 0.1,
  };

  // Calculate dispute handling score
  const disputeScore =
    factors.disputeCount === 0
      ? 1.0
      : factors.amicableResolutionRate *
        Math.max(0, 1 - factors.disputeCount * 0.1);

  // Calculate financial health score
  const financialScore = Math.min(1, factors.sustainedStake / 1000000); // Max at ~1 month

  const score =
    factors.successRate * weights.successRate +
    factors.instructionFollowRate * weights.instructionFollowRate +
    factors.feedbackResponseRate * weights.feedbackResponseRate +
    factors.humanEngagementLevel * weights.humanEngagementLevel +
    factors.agentAvailability * weights.agentAvailability +
    disputeScore * weights.disputeHandling +
    financialScore * weights.financialHealth;

  return Math.max(0, Math.min(1, score));
}

// ============ Bonus Calculations ============

/**
 * Calculate cooperation bonus multiplier.
 */
export function calculateCooperationBonus(
  metrics: PartnershipMetrics,
  config: CooperationBonus = DEFAULT_COOPERATION_BONUS,
): { multiplier: number; appliedBonuses: string[] } {
  let multiplier = 1.0;
  const appliedBonuses: string[] = [];

  // Consecutive success bonus
  // Note: Would need to track consecutive successes in metrics
  const consecutiveSuccesses = estimateConsecutiveSuccesses(metrics);
  for (const bonus of config.consecutiveSuccessBonus) {
    if (consecutiveSuccesses >= bonus.threshold) {
      multiplier = Math.max(multiplier, bonus.bonusMultiplier);
      appliedBonuses.push(
        `Consecutive success (${bonus.threshold}+): ${((bonus.bonusMultiplier - 1) * 100).toFixed(0)}%`,
      );
    }
  }

  // Longevity bonus
  for (const bonus of config.longevityBonus) {
    if (metrics.activeBlocks >= bonus.durationThreshold) {
      multiplier += bonus.bonusPercentage;
      appliedBonuses.push(
        `Longevity (${bonus.durationThreshold}+ blocks): ${(bonus.bonusPercentage * 100).toFixed(0)}%`,
      );
    }
  }

  // Recovery bonus (after slash)
  if (metrics.slashEvents > 0) {
    const successesAfterLastSlash = estimateSuccessesAfterSlash(metrics);
    if (successesAfterLastSlash >= config.recoveryBonus.successesAfterSlash) {
      multiplier += config.recoveryBonus.bonusPercentage;
      appliedBonuses.push(
        `Recovery bonus: ${(config.recoveryBonus.bonusPercentage * 100).toFixed(0)}%`,
      );
    }
  }

  return { multiplier, appliedBonuses };
}

/**
 * Estimate consecutive successes from metrics.
 */
function estimateConsecutiveSuccesses(metrics: PartnershipMetrics): number {
  // Simplified estimation based on success rate
  // In production, would track actual streak
  if (metrics.totalContributions === 0) return 0;
  const successRate =
    metrics.successfulContributions / metrics.totalContributions;
  if (successRate >= 0.95)
    return Math.min(100, metrics.successfulContributions);
  if (successRate >= 0.9) return Math.min(50, metrics.successfulContributions);
  if (successRate >= 0.8) return Math.min(25, metrics.successfulContributions);
  return Math.min(10, metrics.successfulContributions);
}

/**
 * Estimate successes after last slash.
 */
function estimateSuccessesAfterSlash(metrics: PartnershipMetrics): number {
  // Simplified: assume even distribution
  if (metrics.slashEvents === 0) return metrics.successfulContributions;
  return Math.floor(
    metrics.successfulContributions / (metrics.slashEvents + 1),
  );
}

// ============ Human Engagement Tracking ============

/**
 * Track human engagement actions.
 */
export interface HumanEngagementRecord {
  partnershipId: string;
  periodStart: bigint; // Block
  periodEnd: bigint;

  // Actions
  actionsReviewed: number;
  instructionsProvided: number;
  instructionsFollowedSuccessfully: number;
  feedbackGiven: number;
  stakeAdjustments: number;

  // Engagement metrics
  averageResponseTime: bigint; // Blocks
  approvalRate: number; // % of agent requests approved
}

/**
 * Calculate human engagement bonus.
 */
export function calculateHumanEngagementBonus(
  record: HumanEngagementRecord,
  stakeUtilization: number,
  config: HumanEngagementIncentives = DEFAULT_HUMAN_ENGAGEMENT,
): { bonusPercentage: number; breakdown: { [key: string]: number } } {
  const breakdown: { [key: string]: number } = {};
  let totalBonus = 0;

  // Oversight bonus
  if (record.actionsReviewed >= config.oversightBonus.actionsReviewed) {
    breakdown["oversight"] = config.oversightBonus.bonusPercentage;
    totalBonus += config.oversightBonus.bonusPercentage;
  }

  // Instruction bonus
  if (
    record.instructionsProvided >= config.instructionBonus.instructionsProvided
  ) {
    const successRate =
      record.instructionsFollowedSuccessfully / record.instructionsProvided;
    const targetRate =
      config.instructionBonus.instructionsFollowedSuccessfully /
      config.instructionBonus.instructionsProvided;

    if (successRate >= targetRate) {
      breakdown["instruction"] = config.instructionBonus.bonusPercentage;
      totalBonus += config.instructionBonus.bonusPercentage;
    }
  }

  // Capital efficiency bonus
  if (
    stakeUtilization >= config.capitalEfficiencyBonus.stakeUtilizationThreshold
  ) {
    breakdown["capitalEfficiency"] =
      config.capitalEfficiencyBonus.bonusPercentage;
    totalBonus += config.capitalEfficiencyBonus.bonusPercentage;
  }

  return { bonusPercentage: totalBonus, breakdown };
}

// ============ Agent Performance Tracking ============

/**
 * Track agent performance for fair evaluation.
 */
export interface AgentPerformanceRecord {
  partnershipId: string;
  agentDid: DID;
  periodStart: bigint;
  periodEnd: bigint;

  // Work output
  verificationsCompleted: number;
  claimsSubmitted: number;
  reviewsCompleted: number;
  governanceVotes: number;

  // Quality metrics
  verificationAccuracy: number;
  claimAcceptanceRate: number;
  reviewAgreementRate: number;

  // Availability
  requestsReceived: number;
  requestsHandled: number;
  averageResponseTime: bigint; // Blocks

  // Autonomy usage
  autonomousActions: number;
  approvalRequestedActions: number;
  approvalDeniedActions: number;
}

/**
 * Calculate agent performance score.
 */
export function calculateAgentPerformanceScore(
  record: AgentPerformanceRecord,
): { score: number; breakdown: { [key: string]: number } } {
  const breakdown: { [key: string]: number } = {};

  // Quality (40%)
  const qualityScore =
    record.verificationAccuracy * 0.5 +
    record.claimAcceptanceRate * 0.3 +
    record.reviewAgreementRate * 0.2;
  breakdown["quality"] = qualityScore;

  // Availability (30%)
  const availabilityScore =
    record.requestsReceived > 0
      ? record.requestsHandled / record.requestsReceived
      : 1.0;
  breakdown["availability"] = availabilityScore;

  // Autonomy respect (15%)
  // Good score if agent requests approval when it should
  const totalActions =
    record.autonomousActions + record.approvalRequestedActions;
  const autonomyScore =
    totalActions > 0 ? 1 - record.approvalDeniedActions / totalActions : 1.0;
  breakdown["autonomyRespect"] = autonomyScore;

  // Productivity (15%)
  const productivityScore = Math.min(
    1,
    (record.verificationsCompleted +
      record.claimsSubmitted +
      record.reviewsCompleted +
      record.governanceVotes) /
      100, // Target: 100 actions per period
  );
  breakdown["productivity"] = productivityScore;

  const score =
    qualityScore * 0.4 +
    availabilityScore * 0.3 +
    autonomyScore * 0.15 +
    productivityScore * 0.15;

  return { score, breakdown };
}

// ============ Mutual Accountability ============

/**
 * Accountability event types.
 */
export type AccountabilityEvent =
  | "human_delayed_approval" // Human didn't approve in time
  | "human_rejected_valid_action" // Human rejected something that was correct
  | "human_stake_shortfall" // Human didn't maintain stake
  | "agent_unauthorized_action" // Agent acted without required approval
  | "agent_ignored_instruction" // Agent didn't follow valid instruction
  | "agent_quality_failure" // Agent produced poor quality work
  | "mutual_communication_failure"; // Both parties failed to communicate

/**
 * Accountability penalty/consequence.
 */
export interface AccountabilityConsequence {
  event: AccountabilityEvent;
  responsibleParty: "human" | "agent" | "both";
  consequence: {
    reputationPenalty: number;
    rewardReduction: number; // % reduction in next reward
    autonomyAdjustment?: number; // Positive = more autonomy, negative = less
    requiresAcknowledgment: boolean;
  };
}

/**
 * Define consequences for accountability events.
 */
export const ACCOUNTABILITY_CONSEQUENCES: AccountabilityConsequence[] = [
  {
    event: "human_delayed_approval",
    responsibleParty: "human",
    consequence: {
      reputationPenalty: 0.01,
      rewardReduction: 0.05,
      autonomyAdjustment: 0.05, // Agent gets more autonomy
      requiresAcknowledgment: false,
    },
  },
  {
    event: "human_rejected_valid_action",
    responsibleParty: "human",
    consequence: {
      reputationPenalty: 0.02,
      rewardReduction: 0.1,
      autonomyAdjustment: 0.1,
      requiresAcknowledgment: true,
    },
  },
  {
    event: "human_stake_shortfall",
    responsibleParty: "human",
    consequence: {
      reputationPenalty: 0.05,
      rewardReduction: 0.2,
      requiresAcknowledgment: true,
    },
  },
  {
    event: "agent_unauthorized_action",
    responsibleParty: "agent",
    consequence: {
      reputationPenalty: 0.03,
      rewardReduction: 0.15,
      autonomyAdjustment: -0.1, // Reduce autonomy
      requiresAcknowledgment: true,
    },
  },
  {
    event: "agent_ignored_instruction",
    responsibleParty: "agent",
    consequence: {
      reputationPenalty: 0.02,
      rewardReduction: 0.1,
      autonomyAdjustment: -0.05,
      requiresAcknowledgment: true,
    },
  },
  {
    event: "agent_quality_failure",
    responsibleParty: "agent",
    consequence: {
      reputationPenalty: 0.02,
      rewardReduction: 0.1,
      requiresAcknowledgment: false,
    },
  },
  {
    event: "mutual_communication_failure",
    responsibleParty: "both",
    consequence: {
      reputationPenalty: 0.01,
      rewardReduction: 0.05,
      requiresAcknowledgment: true,
    },
  },
];

// ============ Synergy Bonuses ============

/**
 * Bonuses for effective human-agent synergy.
 */
export interface SynergyBonus {
  name: string;
  description: string;
  requirements: {
    minCooperationScore: number;
    minDuration: bigint;
    minSuccessRate: number;
    additionalRequirements?: string[];
  };
  reward: {
    bonusMultiplier: number;
    specialPrivileges: string[];
  };
}

export const SYNERGY_BONUSES: SynergyBonus[] = [
  {
    name: "Perfect Partnership",
    description: "Zero disputes, 95%+ success rate for 1 month",
    requirements: {
      minCooperationScore: 0.95,
      minDuration: 1000000n,
      minSuccessRate: 0.95,
      additionalRequirements: ["zero_disputes"],
    },
    reward: {
      bonusMultiplier: 1.3,
      specialPrivileges: [
        "Priority task assignment",
        "Reduced protocol fees",
        "Early access to new features",
      ],
    },
  },
  {
    name: "Resilient Partnership",
    description: "Recovered from 2+ slashes to 90%+ cooperation",
    requirements: {
      minCooperationScore: 0.9,
      minDuration: 500000n,
      minSuccessRate: 0.85,
      additionalRequirements: ["recovered_from_slashes:2"],
    },
    reward: {
      bonusMultiplier: 1.2,
      specialPrivileges: [
        "Reduced slash severity on future incidents",
        "Partnership recovery guidance",
      ],
    },
  },
  {
    name: "High-Volume Partnership",
    description: "500+ successful contributions",
    requirements: {
      minCooperationScore: 0.8,
      minDuration: 200000n,
      minSuccessRate: 0.8,
      additionalRequirements: ["contributions:500"],
    },
    reward: {
      bonusMultiplier: 1.15,
      specialPrivileges: ["Batch processing privileges", "Higher daily limits"],
    },
  },
  {
    name: "Balanced Partnership",
    description: "Human highly engaged, agent highly available",
    requirements: {
      minCooperationScore: 0.85,
      minDuration: 300000n,
      minSuccessRate: 0.85,
      additionalRequirements: [
        "human_engagement:high",
        "agent_availability:high",
      ],
    },
    reward: {
      bonusMultiplier: 1.1,
      specialPrivileges: [
        "Flexible autonomy settings",
        "Custom revenue sharing",
      ],
    },
  },
];

/**
 * Check if partnership qualifies for synergy bonuses.
 */
export function checkSynergyBonuses(
  partnership: Partnership,
  humanEngagement: HumanEngagementRecord,
  agentPerformance: AgentPerformanceRecord,
  currentBlock: bigint,
): SynergyBonus[] {
  const qualifiedBonuses: SynergyBonus[] = [];
  const metrics = partnership.metrics;
  const duration = currentBlock - partnership.createdAt;

  for (const bonus of SYNERGY_BONUSES) {
    const reqs = bonus.requirements;

    // Check base requirements
    if (metrics.cooperationScore < reqs.minCooperationScore) continue;
    if (duration < reqs.minDuration) continue;

    const successRate =
      metrics.totalContributions > 0
        ? metrics.successfulContributions / metrics.totalContributions
        : 0;
    if (successRate < reqs.minSuccessRate) continue;

    // Check additional requirements
    let meetsAdditional = true;
    if (reqs.additionalRequirements) {
      for (const req of reqs.additionalRequirements) {
        if (req === "zero_disputes" && partnership.disputes.length > 0) {
          meetsAdditional = false;
        }
        if (req.startsWith("recovered_from_slashes:")) {
          const minSlashes = parseInt(req.split(":")[1]);
          if (metrics.slashEvents < minSlashes) {
            meetsAdditional = false;
          }
        }
        if (req.startsWith("contributions:")) {
          const minContrib = parseInt(req.split(":")[1]);
          if (metrics.successfulContributions < minContrib) {
            meetsAdditional = false;
          }
        }
        // Add more checks as needed
      }
    }

    if (meetsAdditional) {
      qualifiedBonuses.push(bonus);
    }
  }

  return qualifiedBonuses;
}

// ============ Fair Sharing Verification ============

/**
 * Verify that revenue sharing is fair based on contributions.
 */
export function verifyFairSharing(
  partnership: Partnership,
  humanContribution: {
    capitalProvided: bigint;
    oversightHours: number; // Estimated
    instructionsProvided: number;
  },
  agentContribution: {
    workUnitsCompleted: number;
    availabilityHours: number;
    qualityScore: number;
  },
): {
  isFair: boolean;
  suggestedAdjustment?: {
    newAgentShare: number;
    reason: string;
  };
} {
  // Calculate contribution ratios
  const humanCapitalValue = Number(
    humanContribution.capitalProvided / 10n ** 18n,
  );
  const humanLaborValue =
    humanContribution.oversightHours * 50 +
    humanContribution.instructionsProvided * 10;
  const humanTotalValue = humanCapitalValue + humanLaborValue;

  const agentLaborValue =
    agentContribution.workUnitsCompleted * 20 * agentContribution.qualityScore;
  const agentAvailabilityValue = agentContribution.availabilityHours * 5;
  const agentTotalValue = agentLaborValue + agentAvailabilityValue;

  const totalValue = humanTotalValue + agentTotalValue;
  if (totalValue === 0) {
    return { isFair: true };
  }

  const fairAgentShare = agentTotalValue / totalValue;
  const currentAgentShare = partnership.revenueSharing.agentShare;

  // Allow 10% deviation from "fair" share
  const deviation = Math.abs(fairAgentShare - currentAgentShare);
  if (deviation <= 0.1) {
    return { isFair: true };
  }

  // Suggest adjustment
  const suggestedShare = Math.max(
    0.5, // Min agent share
    Math.min(0.9, fairAgentShare), // Max agent share
  );

  return {
    isFair: false,
    suggestedAdjustment: {
      newAgentShare: suggestedShare,
      reason: `Current share (${(currentAgentShare * 100).toFixed(0)}%) deviates ${(deviation * 100).toFixed(0)}% from fair share (${(fairAgentShare * 100).toFixed(0)}%) based on contributions`,
    },
  };
}
