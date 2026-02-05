/**
 * LIP Lifecycle Management
 *
 * Manages the state machine for LIP progression from draft to acceptance.
 * Handles stage transitions, validation, and timeout enforcement.
 *
 * PP HIERARCHY:
 *   TRUTH: Stage transitions based on verifiable vote tallies
 *   UNDERSTANDING: Clear state machine with explicit transitions
 *   BEAUTY: Elegant lifecycle with natural progression
 *   JUSTICE: Fair thresholds applied consistently
 *   CREATIVITY: Enables protocol evolution through structured process
 */

import type { DID } from "../../core/agents/identity/did.js";
import {
  type LIP,
  type LIPStage,
  type LIPCategory,
  type StageTransition,
  type VoteTally,
  type AgentGovernanceStats,
  getStakeRequirement,
  getThreshold,
  canPerformAction,
  CONSENSUS_THRESHOLDS,
} from "./types.js";

// ============ State Machine ============

/**
 * Valid stage transitions.
 * Each stage can only transition to specific next stages.
 */
const VALID_TRANSITIONS: Record<LIPStage, LIPStage[]> = {
  draft: ["review", "withdrawn"],
  review: ["last_call", "rejected", "withdrawn"],
  last_call: ["voting", "rejected", "withdrawn"],
  voting: ["accepted", "rejected"],
  accepted: ["implemented"],
  implemented: [], // Terminal
  rejected: [], // Terminal
  withdrawn: [], // Terminal
};

/**
 * Check if a stage transition is valid.
 */
export function isValidTransition(from: LIPStage, to: LIPStage): boolean {
  return VALID_TRANSITIONS[from]?.includes(to) ?? false;
}

// ============ LIP Creation ============

/**
 * Parameters for creating a new LIP.
 */
export interface CreateLIPParams {
  title: string;
  category: LIPCategory;
  proposer: DID;
  abstract: string;
  motivation: string;
  specification: string;
  rationale: string;
  backwardsCompatibility: string;
  testCases: string;
  referenceImplementation?: string;
  coAuthors?: DID[];
}

/**
 * Create a new LIP in draft stage.
 */
export function createLIP(
  params: CreateLIPParams,
  proposerStats: AgentGovernanceStats,
  currentBlock: bigint,
  lipNumber: number,
): { lip: LIP; stakeRequired: bigint } | { error: string } {
  // Check eligibility
  const eligibility = canPerformAction(proposerStats, "propose_lip");
  if (!eligibility.allowed) {
    return { error: eligibility.reason! };
  }

  // Validate abstract length
  if (params.abstract.length > 200) {
    return {
      error: `Abstract too long: ${params.abstract.length} chars, max 200`,
    };
  }

  // Get stake requirement
  const stakeReq = getStakeRequirement(params.category);

  const lip: LIP = {
    id: `LIP-${lipNumber}`,
    title: params.title,
    category: params.category,
    proposer: params.proposer,
    coAuthors: params.coAuthors || [],
    createdAt: currentBlock,
    updatedAt: currentBlock,
    abstract: params.abstract,
    motivation: params.motivation,
    specification: params.specification,
    rationale: params.rationale,
    backwardsCompatibility: params.backwardsCompatibility,
    testCases: params.testCases,
    referenceImplementation: params.referenceImplementation,
    stage: "draft",
    stageHistory: [
      {
        from: "draft",
        to: "draft",
        at: currentBlock,
        reason: "LIP created",
        triggeredBy: params.proposer,
      },
    ],
    stakedAmount: 0n, // Must be staked separately
    stakeHolder: params.proposer,
    amendments: [],
  };

  return { lip, stakeRequired: stakeReq.requiredStake };
}

// ============ Stage Transitions ============

/**
 * Result of attempting a stage transition.
 */
export interface TransitionResult {
  success: boolean;
  newStage?: LIPStage;
  error?: string;
  transition?: StageTransition;
}

/**
 * Transition a LIP to review stage.
 * Requires stake to be deposited.
 */
export function transitionToReview(
  lip: LIP,
  stakedAmount: bigint,
  currentBlock: bigint,
  triggeredBy: DID,
): TransitionResult {
  if (lip.stage !== "draft") {
    return {
      success: false,
      error: `Cannot transition from ${lip.stage} to review`,
    };
  }

  const stakeReq = getStakeRequirement(lip.category);
  if (stakedAmount < stakeReq.requiredStake) {
    return {
      success: false,
      error: `Insufficient stake: ${stakedAmount / 10n ** 18n} LGM, need ${stakeReq.requiredStake / 10n ** 18n} LGM`,
    };
  }

  const transition: StageTransition = {
    from: "draft",
    to: "review",
    at: currentBlock,
    reason: "Stake deposited, entering review",
    triggeredBy,
  };

  lip.stage = "review";
  lip.stakedAmount = stakedAmount;
  lip.stageHistory.push(transition);
  lip.updatedAt = currentBlock;

  return { success: true, newStage: "review", transition };
}

/**
 * Transition based on review stage voting results.
 */
export function transitionFromReview(
  lip: LIP,
  votes: VoteTally,
  currentBlock: bigint,
  stageStartBlock: bigint,
): TransitionResult {
  if (lip.stage !== "review") {
    return { success: false, error: `LIP not in review stage` };
  }

  const threshold = getThreshold("review");
  const stageDuration = currentBlock - stageStartBlock;

  // Check if stage duration has passed
  if (stageDuration < threshold.duration) {
    return {
      success: false,
      error: `Review period not complete: ${stageDuration}/${threshold.duration} blocks`,
    };
  }

  // Check operator cap violation
  if (votes.operatorCapViolation) {
    return {
      success: false,
      error:
        "Operator cap violation detected - single operator exceeds 20% of votes",
    };
  }

  // Calculate support ratio
  const totalVotes = votes.support + votes.oppose;
  const supportRatio =
    totalVotes > 0n ? Number(votes.support) / Number(totalVotes) : 0;
  const quorumRatio =
    Number(votes.support + votes.oppose + votes.abstain) /
    Number(votes.totalEligible);

  // Check thresholds
  if (quorumRatio < threshold.quorumRequired) {
    const transition: StageTransition = {
      from: "review",
      to: "rejected",
      at: currentBlock,
      reason: `Quorum not met: ${(quorumRatio * 100).toFixed(1)}% < ${threshold.quorumRequired * 100}%`,
      triggeredBy: "system",
    };
    lip.stage = "rejected";
    lip.stageHistory.push(transition);
    lip.votes = votes;
    lip.updatedAt = currentBlock;
    return { success: true, newStage: "rejected", transition };
  }

  if (supportRatio < threshold.supportRequired) {
    const transition: StageTransition = {
      from: "review",
      to: "rejected",
      at: currentBlock,
      reason: `Support threshold not met: ${(supportRatio * 100).toFixed(1)}% < ${threshold.supportRequired * 100}%`,
      triggeredBy: "system",
    };
    lip.stage = "rejected";
    lip.stageHistory.push(transition);
    lip.votes = votes;
    lip.updatedAt = currentBlock;
    return { success: true, newStage: "rejected", transition };
  }

  // Proceed to last_call
  const transition: StageTransition = {
    from: "review",
    to: "last_call",
    at: currentBlock,
    reason: `Review passed: ${(supportRatio * 100).toFixed(1)}% support, ${(quorumRatio * 100).toFixed(1)}% quorum`,
    triggeredBy: "system",
  };
  lip.stage = "last_call";
  lip.stageHistory.push(transition);
  lip.votes = votes;
  lip.updatedAt = currentBlock;

  return { success: true, newStage: "last_call", transition };
}

/**
 * Transition based on last_call stage voting results.
 */
export function transitionFromLastCall(
  lip: LIP,
  votes: VoteTally,
  currentBlock: bigint,
  stageStartBlock: bigint,
): TransitionResult {
  if (lip.stage !== "last_call") {
    return { success: false, error: `LIP not in last_call stage` };
  }

  const threshold = getThreshold("last_call");
  const stageDuration = currentBlock - stageStartBlock;

  if (stageDuration < threshold.duration) {
    return {
      success: false,
      error: `Last call period not complete: ${stageDuration}/${threshold.duration} blocks`,
    };
  }

  if (votes.operatorCapViolation) {
    return {
      success: false,
      error: "Operator cap violation detected",
    };
  }

  const totalVotes = votes.support + votes.oppose;
  const supportRatio =
    totalVotes > 0n ? Number(votes.support) / Number(totalVotes) : 0;
  const quorumRatio =
    Number(votes.support + votes.oppose + votes.abstain) /
    Number(votes.totalEligible);

  if (
    quorumRatio < threshold.quorumRequired ||
    supportRatio < threshold.supportRequired
  ) {
    const transition: StageTransition = {
      from: "last_call",
      to: "rejected",
      at: currentBlock,
      reason: `Last call failed: ${(supportRatio * 100).toFixed(1)}% support, ${(quorumRatio * 100).toFixed(1)}% quorum`,
      triggeredBy: "system",
    };
    lip.stage = "rejected";
    lip.stageHistory.push(transition);
    lip.votes = votes;
    lip.updatedAt = currentBlock;
    return { success: true, newStage: "rejected", transition };
  }

  // Proceed to voting
  const transition: StageTransition = {
    from: "last_call",
    to: "voting",
    at: currentBlock,
    reason: `Last call passed: ${(supportRatio * 100).toFixed(1)}% support`,
    triggeredBy: "system",
  };
  lip.stage = "voting";
  lip.stageHistory.push(transition);
  lip.votes = votes;
  lip.updatedAt = currentBlock;

  return { success: true, newStage: "voting", transition };
}

/**
 * Final acceptance vote.
 */
export function transitionFromVoting(
  lip: LIP,
  votes: VoteTally,
  currentBlock: bigint,
): TransitionResult {
  if (lip.stage !== "voting") {
    return { success: false, error: `LIP not in voting stage` };
  }

  const threshold = getThreshold("acceptance");

  if (votes.operatorCapViolation) {
    return {
      success: false,
      error: "Operator cap violation detected",
    };
  }

  const totalVotes = votes.support + votes.oppose;
  const supportRatio =
    totalVotes > 0n ? Number(votes.support) / Number(totalVotes) : 0;
  const quorumRatio =
    Number(votes.support + votes.oppose + votes.abstain) /
    Number(votes.totalEligible);

  if (
    quorumRatio < threshold.quorumRequired ||
    supportRatio < threshold.supportRequired
  ) {
    const transition: StageTransition = {
      from: "voting",
      to: "rejected",
      at: currentBlock,
      reason: `Vote failed: ${(supportRatio * 100).toFixed(1)}% support (need ${threshold.supportRequired * 100}%), ${(quorumRatio * 100).toFixed(1)}% quorum`,
      triggeredBy: "system",
    };
    lip.stage = "rejected";
    lip.stageHistory.push(transition);
    lip.votes = votes;
    lip.updatedAt = currentBlock;
    return { success: true, newStage: "rejected", transition };
  }

  // Accepted!
  const transition: StageTransition = {
    from: "voting",
    to: "accepted",
    at: currentBlock,
    reason: `Vote passed: ${(supportRatio * 100).toFixed(1)}% support, ${(quorumRatio * 100).toFixed(1)}% quorum`,
    triggeredBy: "system",
  };
  lip.stage = "accepted";
  lip.stageHistory.push(transition);
  lip.votes = votes;
  lip.updatedAt = currentBlock;

  return { success: true, newStage: "accepted", transition };
}

/**
 * Mark a LIP as implemented.
 */
export function transitionToImplemented(
  lip: LIP,
  currentBlock: bigint,
  triggeredBy: DID,
  implementationDetails: string,
): TransitionResult {
  if (lip.stage !== "accepted") {
    return { success: false, error: `Can only implement accepted LIPs` };
  }

  const transition: StageTransition = {
    from: "accepted",
    to: "implemented",
    at: currentBlock,
    reason: implementationDetails,
    triggeredBy,
  };
  lip.stage = "implemented";
  lip.stageHistory.push(transition);
  lip.updatedAt = currentBlock;

  return { success: true, newStage: "implemented", transition };
}

/**
 * Withdraw a LIP (only by proposer, before voting stage).
 */
export function withdrawLIP(
  lip: LIP,
  currentBlock: bigint,
  withdrawer: DID,
): TransitionResult {
  if (lip.proposer !== withdrawer && !lip.coAuthors.includes(withdrawer)) {
    return {
      success: false,
      error: "Only proposer or co-authors can withdraw",
    };
  }

  if (!["draft", "review", "last_call"].includes(lip.stage)) {
    return { success: false, error: `Cannot withdraw from ${lip.stage} stage` };
  }

  const transition: StageTransition = {
    from: lip.stage,
    to: "withdrawn",
    at: currentBlock,
    reason: "Withdrawn by proposer",
    triggeredBy: withdrawer,
  };
  lip.stage = "withdrawn";
  lip.stageHistory.push(transition);
  lip.updatedAt = currentBlock;

  return { success: true, newStage: "withdrawn", transition };
}

// ============ Stake Management ============

/**
 * Calculate stake return or slash based on outcome.
 */
export interface StakeOutcome {
  returnAmount: bigint;
  slashAmount: bigint;
  reason: string;
}

export function calculateStakeOutcome(
  lip: LIP,
  outcome:
    | "accepted"
    | "rejected_malicious"
    | "rejected_frivolous"
    | "rejected_normal"
    | "withdrawn",
): StakeOutcome {
  const stakeReq = getStakeRequirement(lip.category);

  switch (outcome) {
    case "accepted":
      // Full return on acceptance
      return {
        returnAmount: lip.stakedAmount,
        slashAmount: 0n,
        reason: "LIP accepted - full stake returned",
      };

    case "rejected_malicious":
      // 100% slash for malicious proposals
      return {
        returnAmount: 0n,
        slashAmount: lip.stakedAmount,
        reason: "LIP rejected as malicious - full stake slashed",
      };

    case "rejected_frivolous": {
      // 50% slash for frivolous proposals
      const frivolousSlash =
        (lip.stakedAmount * BigInt(Math.floor(stakeReq.frivolousSlash * 100))) /
        100n;
      return {
        returnAmount: lip.stakedAmount - frivolousSlash,
        slashAmount: frivolousSlash,
        reason: "LIP rejected as frivolous - 50% stake slashed",
      };
    }

    case "rejected_normal": {
      // Small penalty for normal rejection (10%)
      const normalSlash = lip.stakedAmount / 10n;
      return {
        returnAmount: lip.stakedAmount - normalSlash,
        slashAmount: normalSlash,
        reason: "LIP rejected - 10% stake slashed",
      };
    }

    case "withdrawn": {
      // Early withdrawal returns most stake (95%)
      const withdrawalFee = lip.stakedAmount / 20n;
      return {
        returnAmount: lip.stakedAmount - withdrawalFee,
        slashAmount: withdrawalFee,
        reason: "LIP withdrawn - 5% processing fee",
      };
    }
  }
}

// ============ Timeline Helpers ============

/**
 * Get the block when a stage started.
 */
export function getStageStartBlock(lip: LIP, stage: LIPStage): bigint | null {
  const transition = lip.stageHistory.find((t) => t.to === stage);
  return transition?.at ?? null;
}

/**
 * Get remaining blocks in current stage.
 */
export function getRemainingBlocks(
  lip: LIP,
  currentBlock: bigint,
): bigint | null {
  const stageStart = getStageStartBlock(lip, lip.stage);
  if (!stageStart) return null;

  const threshold = CONSENSUS_THRESHOLDS.find((t) => t.stage === lip.stage);
  if (!threshold) return null;

  const elapsed = currentBlock - stageStart;
  const remaining = threshold.duration - elapsed;
  return remaining > 0n ? remaining : 0n;
}

/**
 * Check if stage has timed out.
 */
export function isStageExpired(lip: LIP, currentBlock: bigint): boolean {
  const remaining = getRemainingBlocks(lip, currentBlock);
  return remaining !== null && remaining === 0n;
}
