/**
 * Partnership Manager
 *
 * Manages the lifecycle of human-agent partnerships including creation,
 * amendments, revenue distribution, and dissolution.
 *
 * PP HIERARCHY:
 *   TRUTH: All partnership state changes recorded on-chain
 *   UNDERSTANDING: Clear lifecycle management
 *   BEAUTY: Elegant state machine for partnerships
 *   JUSTICE: Fair treatment of both parties
 *   CREATIVITY: Novel human-AI collaboration protocol
 */

import type { DID } from "../../core/agents/identity/did.js";
import {
  type Partnership,
  type PartnershipProposal,
  type PartnershipAmendment,
  type PartnershipMetrics,
  type RevenueSharing,
  type StakeContribution,
  type RiskSharing,
  type AutonomyConfig,
  type PartnershipVesting,
  type PartnershipTier,
  type PartnershipStatus,
  type RewardDistribution,
  type DissolutionRequest,
  PARTNERSHIP_BOUNDS,
  PARTNERSHIP_TIERS,
} from "./types.js";

// ============ Validation ============

/**
 * Validate revenue sharing configuration.
 */
export function validateRevenueSharing(sharing: RevenueSharing): {
  valid: boolean;
  errors: string[];
} {
  const errors: string[] = [];

  if (sharing.agentShare < PARTNERSHIP_BOUNDS.minAgentShare) {
    errors.push(
      `Agent share ${sharing.agentShare} below minimum ${PARTNERSHIP_BOUNDS.minAgentShare}`,
    );
  }

  if (sharing.agentShare > PARTNERSHIP_BOUNDS.maxAgentShare) {
    errors.push(
      `Agent share ${sharing.agentShare} above maximum ${PARTNERSHIP_BOUNDS.maxAgentShare}`,
    );
  }

  if (Math.abs(sharing.agentShare + sharing.humanShare - 1.0) > 0.001) {
    errors.push("Agent share + human share must equal 1.0");
  }

  // Validate category overrides
  if (sharing.categoryOverrides) {
    for (const [category, override] of Object.entries(
      sharing.categoryOverrides,
    )) {
      if (override.agentShare < PARTNERSHIP_BOUNDS.minAgentShare) {
        errors.push(`${category} agent share below minimum`);
      }
      if (override.agentShare > PARTNERSHIP_BOUNDS.maxAgentShare) {
        errors.push(`${category} agent share above maximum`);
      }
    }
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Validate stake contribution.
 */
export function validateStakeContribution(contribution: StakeContribution): {
  valid: boolean;
  errors: string[];
} {
  const errors: string[] = [];

  if (contribution.humanRatio < PARTNERSHIP_BOUNDS.minHumanStakeRatio) {
    errors.push(
      `Human stake ratio ${contribution.humanRatio} below minimum ${PARTNERSHIP_BOUNDS.minHumanStakeRatio}`,
    );
  }

  if (contribution.humanRatio > PARTNERSHIP_BOUNDS.maxHumanStakeRatio) {
    errors.push(
      `Human stake ratio ${contribution.humanRatio} above maximum ${PARTNERSHIP_BOUNDS.maxHumanStakeRatio}`,
    );
  }

  if (
    contribution.totalStake !==
    contribution.humanContribution + contribution.agentContribution
  ) {
    errors.push("Total stake must equal sum of contributions");
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Validate autonomy configuration.
 */
export function validateAutonomy(autonomy: AutonomyConfig): {
  valid: boolean;
  errors: string[];
} {
  const errors: string[] = [];

  const checkBound = (value: number, name: string) => {
    if (value < PARTNERSHIP_BOUNDS.minAgentAutonomy) {
      errors.push(`${name} autonomy below minimum`);
    }
    if (value > PARTNERSHIP_BOUNDS.maxAgentAutonomy) {
      errors.push(`${name} autonomy above maximum`);
    }
  };

  checkBound(autonomy.verification, "Verification");
  checkBound(autonomy.claimSubmission, "Claim submission");
  checkBound(autonomy.governanceVoting, "Governance voting");
  checkBound(autonomy.globalAutonomy, "Global");

  // Partnership changes should always require more approval
  if (autonomy.partnershipChanges > 0.3) {
    errors.push("Partnership changes autonomy should not exceed 0.3");
  }

  return { valid: errors.length === 0, errors };
}

// ============ Partnership Creation ============

/**
 * Create a partnership proposal.
 */
export function createProposal(
  humanId: string,
  targetAgentDid: DID,
  terms: PartnershipProposal["proposedTerms"],
  currentBlock: bigint,
): { proposal: PartnershipProposal; errors: string[] } {
  const errors: string[] = [];

  // Validate all terms
  const revenueValidation = validateRevenueSharing(terms.revenueSharing);
  errors.push(...revenueValidation.errors);

  const autonomyValidation = validateAutonomy(terms.autonomy);
  errors.push(...autonomyValidation.errors);

  if (terms.duration < PARTNERSHIP_BOUNDS.minPartnershipDuration) {
    errors.push(
      `Duration ${terms.duration} below minimum ${PARTNERSHIP_BOUNDS.minPartnershipDuration}`,
    );
  }

  if (errors.length > 0) {
    return {
      proposal: null as unknown as PartnershipProposal,
      errors,
    };
  }

  const proposal: PartnershipProposal = {
    proposalId: `prop-${humanId}-${targetAgentDid}-${currentBlock}`,
    humanId,
    targetAgentDid,
    proposedTerms: terms,
    proposedAt: currentBlock,
    expiresAt: currentBlock + 50000n, // ~1.5 days to respond
    status: "pending",
  };

  return { proposal, errors: [] };
}

/**
 * Accept a partnership proposal (by agent).
 */
export function acceptProposal(
  proposal: PartnershipProposal,
  agentStakeContribution: bigint,
  currentBlock: bigint,
): { partnership: Partnership; errors: string[] } {
  const errors: string[] = [];

  if (proposal.status !== "pending") {
    errors.push(`Proposal is ${proposal.status}, cannot accept`);
    return { partnership: null as unknown as Partnership, errors };
  }

  if (currentBlock > proposal.expiresAt) {
    errors.push("Proposal has expired");
    return { partnership: null as unknown as Partnership, errors };
  }

  // Calculate stake contribution
  const totalStake =
    proposal.proposedTerms.initialStake + agentStakeContribution;
  const humanRatio =
    Number(proposal.proposedTerms.initialStake) / Number(totalStake);
  const agentRatio = 1 - humanRatio;

  const stakeContribution: StakeContribution = {
    humanContribution: proposal.proposedTerms.initialStake,
    agentContribution: agentStakeContribution,
    totalStake,
    humanRatio,
    agentRatio,
  };

  const stakeValidation = validateStakeContribution(stakeContribution);
  errors.push(...stakeValidation.errors);

  if (errors.length > 0) {
    return { partnership: null as unknown as Partnership, errors };
  }

  const partnership: Partnership = {
    partnershipId: `partnership-${proposal.humanId}-${proposal.targetAgentDid}-${currentBlock}`,
    humanId: proposal.humanId,
    agentDid: proposal.targetAgentDid,

    status: "active",
    tier: "trial", // Start at trial
    createdAt: currentBlock,
    updatedAt: currentBlock,
    expiresAt: currentBlock + proposal.proposedTerms.duration,

    revenueSharing: proposal.proposedTerms.revenueSharing,
    stakeContribution,
    riskSharing: proposal.proposedTerms.riskSharing,
    vesting: proposal.proposedTerms.vesting,
    autonomy: proposal.proposedTerms.autonomy,

    metrics: createInitialMetrics(),
    amendments: [],
    disputes: [],
  };

  return { partnership, errors: [] };
}

/**
 * Create initial partnership metrics.
 */
function createInitialMetrics(): PartnershipMetrics {
  return {
    totalRewardsEarned: 0n,
    humanRewardsReceived: 0n,
    agentRewardsReceived: 0n,
    totalContributions: 0,
    successfulContributions: 0,
    failedContributions: 0,
    slashEvents: 0,
    totalSlashed: 0n,
    humanSlashAbsorbed: 0n,
    agentSlashAbsorbed: 0n,
    activeBlocks: 0n,
    suspendedBlocks: 0n,
    cooperationScore: 0.5, // Neutral start
  };
}

// ============ Revenue Distribution ============

/**
 * Calculate and distribute a reward to the partnership.
 */
export function distributeReward(
  partnership: Partnership,
  grossReward: bigint,
  sourceType: RewardDistribution["sourceType"],
  sourceId: string,
  currentBlock: bigint,
): { distribution: RewardDistribution; updatedPartnership: Partnership } {
  // Get applicable agent share
  let agentShare = partnership.revenueSharing.agentShare;

  // Check for category override
  if (partnership.revenueSharing.categoryOverrides) {
    const override =
      partnership.revenueSharing.categoryOverrides[
        sourceType as keyof typeof partnership.revenueSharing.categoryOverrides
      ];
    if (override) {
      agentShare = override.agentShare;
    }
  }

  const humanShare = 1 - agentShare;

  // Calculate protocol fee (1%)
  const protocolFee = grossReward / 100n;
  const netReward = grossReward - protocolFee;

  // Calculate tier bonus
  const tierConfig = PARTNERSHIP_TIERS[partnership.tier];
  const tierBonus =
    (netReward *
      BigInt(Math.floor((tierConfig.benefits.rewardBonus - 1) * 10000))) /
    10000n;

  // Calculate milestone bonus
  let milestoneBonus = 0n;
  if (partnership.revenueSharing.milestoneBonus) {
    for (const milestone of partnership.revenueSharing.milestoneBonus) {
      if (partnership.metrics.successfulContributions >= milestone.threshold) {
        milestoneBonus +=
          (netReward * BigInt(Math.floor(milestone.agentBonusShare * 10000))) /
          10000n;
      }
    }
  }

  const totalReward = netReward + tierBonus + milestoneBonus;

  // Split between human and agent
  const agentAmount =
    (totalReward * BigInt(Math.floor(agentShare * 10000))) / 10000n;
  const humanAmount = totalReward - agentAmount;

  // Apply vesting
  const immediateRatio = partnership.vesting.immediateRatio;
  const vestedRatio = partnership.vesting.vestedRatio;

  const humanImmediate =
    (humanAmount * BigInt(Math.floor(immediateRatio * 10000))) / 10000n;
  const humanVested = humanAmount - humanImmediate;

  const agentImmediate =
    (agentAmount * BigInt(Math.floor(immediateRatio * 10000))) / 10000n;
  const agentVested = agentAmount - agentImmediate;

  const distribution: RewardDistribution = {
    distributionId: `dist-${partnership.partnershipId}-${currentBlock}`,
    partnershipId: partnership.partnershipId,
    sourceType,
    sourceId,
    grossReward,
    protocolFee,
    netReward,
    humanShare: humanAmount,
    humanImmediate,
    humanVested,
    agentShare: agentAmount,
    agentImmediate,
    agentVested,
    tierBonus,
    milestoneBonus,
    distributedAt: currentBlock,
  };

  // Update metrics
  const updatedMetrics: PartnershipMetrics = {
    ...partnership.metrics,
    totalRewardsEarned: partnership.metrics.totalRewardsEarned + totalReward,
    humanRewardsReceived:
      partnership.metrics.humanRewardsReceived + humanImmediate,
    agentRewardsReceived:
      partnership.metrics.agentRewardsReceived + agentImmediate,
    totalContributions: partnership.metrics.totalContributions + 1,
    successfulContributions: partnership.metrics.successfulContributions + 1,
  };

  // Update cooperation score
  updatedMetrics.cooperationScore = Math.min(
    1.0,
    updatedMetrics.cooperationScore + 0.01,
  );

  const updatedPartnership: Partnership = {
    ...partnership,
    metrics: updatedMetrics,
    updatedAt: currentBlock,
  };

  return { distribution, updatedPartnership };
}

/**
 * Process a slash event for the partnership.
 */
export function processSlash(
  partnership: Partnership,
  slashAmount: bigint,
  reason: string,
  currentBlock: bigint,
): { humanSlash: bigint; agentSlash: bigint; updatedPartnership: Partnership } {
  const { slashDistribution, humanMaxLoss, agentMaxLoss } =
    partnership.riskSharing;

  // Calculate raw split
  let humanSlash =
    (slashAmount * BigInt(Math.floor(slashDistribution.humanShare * 10000))) /
    10000n;
  let agentSlash = slashAmount - humanSlash;

  // Apply max loss limits
  if (humanSlash > humanMaxLoss) {
    const excess = humanSlash - humanMaxLoss;
    humanSlash = humanMaxLoss;
    agentSlash += excess;
  }

  if (agentSlash > agentMaxLoss) {
    const excess = agentSlash - agentMaxLoss;
    agentSlash = agentMaxLoss;
    // Excess is absorbed by protocol (burned)
  }

  // Update metrics
  const updatedMetrics: PartnershipMetrics = {
    ...partnership.metrics,
    slashEvents: partnership.metrics.slashEvents + 1,
    totalSlashed: partnership.metrics.totalSlashed + slashAmount,
    humanSlashAbsorbed: partnership.metrics.humanSlashAbsorbed + humanSlash,
    agentSlashAbsorbed: partnership.metrics.agentSlashAbsorbed + agentSlash,
    failedContributions: partnership.metrics.failedContributions + 1,
    cooperationScore: Math.max(0, partnership.metrics.cooperationScore - 0.05),
  };

  // Check auto-terminate threshold
  let newStatus = partnership.status;
  if (
    updatedMetrics.slashEvents >= partnership.riskSharing.autoTerminateThreshold
  ) {
    newStatus = "dissolving";
  }

  const updatedPartnership: Partnership = {
    ...partnership,
    status: newStatus,
    metrics: updatedMetrics,
    updatedAt: currentBlock,
  };

  return { humanSlash, agentSlash, updatedPartnership };
}

// ============ Tier Management ============

/**
 * Check and update partnership tier based on metrics.
 */
export function updateTier(
  partnership: Partnership,
  currentBlock: bigint,
): { newTier: PartnershipTier; upgraded: boolean } {
  const metrics = partnership.metrics;
  const activeDuration = currentBlock - partnership.createdAt;

  // Check each tier from highest to lowest
  const tiers: PartnershipTier[] = ["elite", "trusted", "standard", "trial"];

  for (const tier of tiers) {
    const requirements = PARTNERSHIP_TIERS[tier];

    if (
      activeDuration >= requirements.minDuration &&
      metrics.successfulContributions >= requirements.minSuccessfulRewards &&
      partnership.stakeContribution.totalStake >= requirements.minCombinedStake
    ) {
      const upgraded =
        tier !== partnership.tier &&
        tiers.indexOf(tier) < tiers.indexOf(partnership.tier);
      return { newTier: tier, upgraded };
    }
  }

  return { newTier: "trial", upgraded: false };
}

// ============ Autonomy Checks ============

/**
 * Check if agent can perform an action autonomously.
 */
export function canActAutonomously(
  partnership: Partnership,
  actionType: keyof Omit<AutonomyConfig, "globalAutonomy" | "inactivityDecay">,
  currentBlock: bigint,
): { canAct: boolean; requiresApproval: boolean; reason?: string } {
  const autonomy = partnership.autonomy;

  // Get base autonomy for this action
  let actionAutonomy = autonomy[actionType];

  // Apply global multiplier
  actionAutonomy *= autonomy.globalAutonomy;

  // Apply inactivity decay
  const lastActive = partnership.updatedAt;
  const inactiveBlocks = currentBlock - lastActive;

  if (inactiveBlocks > autonomy.inactivityDecay.decayStartBlocks) {
    const decayPeriods = Number(
      (inactiveBlocks - autonomy.inactivityDecay.decayStartBlocks) / 1000n,
    );
    const decay = decayPeriods * autonomy.inactivityDecay.decayRate;
    actionAutonomy = Math.max(
      autonomy.inactivityDecay.minAutonomy,
      actionAutonomy - decay,
    );
  }

  // Threshold: 0.5 means 50%+ autonomy = can act alone
  if (actionAutonomy >= 0.5) {
    return { canAct: true, requiresApproval: false };
  }

  // Between 0.25 and 0.5: can act but requires async approval
  if (actionAutonomy >= 0.25) {
    return {
      canAct: true,
      requiresApproval: true,
      reason: "Action allowed but human approval requested within 1000 blocks",
    };
  }

  // Below 0.25: requires approval before acting
  return {
    canAct: false,
    requiresApproval: true,
    reason: "Human approval required before this action",
  };
}

// ============ Amendment Processing ============

/**
 * Propose an amendment to partnership terms.
 */
export function proposeAmendment(
  partnership: Partnership,
  proposedBy: "human" | "agent",
  changes: PartnershipAmendment["changes"],
  rationale: string,
  currentBlock: bigint,
): { amendment: PartnershipAmendment; errors: string[] } {
  const errors: string[] = [];

  // Validate proposed changes
  if (changes.revenueSharing) {
    const validation = validateRevenueSharing(changes.revenueSharing);
    errors.push(...validation.errors);
  }

  if (changes.autonomy) {
    const validation = validateAutonomy(changes.autonomy);
    errors.push(...validation.errors);
  }

  if (errors.length > 0) {
    return { amendment: null as unknown as PartnershipAmendment, errors };
  }

  const amendment: PartnershipAmendment = {
    amendmentId: `amend-${partnership.partnershipId}-${currentBlock}`,
    partnershipId: partnership.partnershipId,
    proposedBy,
    changes,
    proposedAt: currentBlock,
    status: "pending",
    rationale,
  };

  return { amendment, errors: [] };
}

/**
 * Accept an amendment (by the other party).
 */
export function acceptAmendment(
  partnership: Partnership,
  amendment: PartnershipAmendment,
  currentBlock: bigint,
): Partnership {
  // Apply changes
  const updated = { ...partnership };

  if (amendment.changes.revenueSharing) {
    updated.revenueSharing = amendment.changes.revenueSharing;
  }

  if (amendment.changes.riskSharing) {
    updated.riskSharing = amendment.changes.riskSharing;
  }

  if (amendment.changes.autonomy) {
    updated.autonomy = amendment.changes.autonomy;
  }

  if (amendment.changes.vesting) {
    updated.vesting = amendment.changes.vesting;
  }

  // Update amendment status
  amendment.status = "accepted";
  amendment.acceptedAt = currentBlock;
  amendment.effectiveAt = currentBlock; // Could add delay for major changes

  updated.amendments = [...partnership.amendments, amendment];
  updated.updatedAt = currentBlock;

  return updated;
}

// ============ Dissolution ============

/**
 * Request partnership dissolution.
 */
export function requestDissolution(
  partnership: Partnership,
  requestedBy: "human" | "agent" | "system",
  reason: string,
  proposedTerms: DissolutionRequest["proposedTerms"],
  currentBlock: bigint,
): DissolutionRequest {
  return {
    requestId: `dissolve-${partnership.partnershipId}-${currentBlock}`,
    partnershipId: partnership.partnershipId,
    requestedBy,
    reason,
    proposedTerms,
    status: "pending",
    requestedAt: currentBlock,
  };
}

/**
 * Execute dissolution.
 */
export function executeDissolution(
  partnership: Partnership,
  request: DissolutionRequest,
  currentBlock: bigint,
): Partnership {
  return {
    ...partnership,
    status: "dissolved",
    updatedAt: currentBlock,
    expiresAt: currentBlock, // Mark as ended now
  };
}

// ============ Default Configurations ============

/**
 * Create default partnership terms.
 */
export function createDefaultTerms(
  humanInitialStake: bigint,
  duration: bigint = PARTNERSHIP_BOUNDS.defaultPartnershipDuration,
): PartnershipProposal["proposedTerms"] {
  return {
    revenueSharing: {
      agentShare: PARTNERSHIP_BOUNDS.defaultAgentShare,
      humanShare: 1 - PARTNERSHIP_BOUNDS.defaultAgentShare,
    },
    initialStake: humanInitialStake,
    riskSharing: {
      slashDistribution: {
        humanShare: PARTNERSHIP_BOUNDS.defaultSlashSplit,
        agentShare: PARTNERSHIP_BOUNDS.defaultSlashSplit,
      },
      humanMaxLoss: humanInitialStake / 2n, // Max 50% loss per incident
      agentMaxLoss: humanInitialStake / 2n,
      recoveryPeriod: 10000n, // ~7 hours
      autoTerminateThreshold: 5,
    },
    autonomy: {
      verification: 0.8, // High autonomy for verification
      claimSubmission: 0.6,
      challengeInitiation: 0.4, // Needs more oversight
      stakeAdjustment: 0.3,
      governanceVoting: 0.5,
      partnershipChanges: 0.1, // Almost always needs approval
      globalAutonomy: 0.7,
      inactivityDecay: {
        decayStartBlocks: 50000n, // ~1.5 days
        decayRate: 0.05, // 5% per 1000 blocks
        minAutonomy: 0.2,
      },
    },
    vesting: {
      immediateRatio: 0.7, // 70% immediate
      vestedRatio: 0.3, // 30% vested
      vestingPeriod: 200000n, // ~6 days
      cliffPeriod: 50000n, // ~1.5 days
      clawbackEnabled: true,
      clawbackConditions: [
        "slashing_event",
        "partnership_dissolution_for_cause",
      ],
    },
    duration,
  };
}
