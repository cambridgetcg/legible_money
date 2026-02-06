/**
 * Disbursement Lifecycle Management
 *
 * Pure state machine functions for disbursement proposal progression.
 * No state access — receives proposal objects, mutates them, returns success/error.
 * Pattern follows governance/proposals/lifecycle.ts.
 *
 * Lifecycle: proposed → voting → approved → executed
 *                                        → rejected
 *            proposed|approved → cancelled
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { StageTransition, VoteTally } from "../proposals/types.js";
import {
  type DisbursementProposal,
  type DisbursementStage,
  type DisbursementCategory,
  type TreasuryFund,
  DISBURSEMENT_STAKE,
  DISBURSEMENT_THRESHOLDS,
  DISBURSEMENT_THRESHOLDS_BPS,
  TIMELOCK_BLOCKS,
  DISBURSEMENT_SLASH_BPS,
  VALID_DISBURSEMENT_CATEGORIES,
  VALID_TREASURY_FUNDS,
} from "./types.js";

// ============ State Machine ============

const VALID_TRANSITIONS: Record<DisbursementStage, DisbursementStage[]> = {
  proposed: ["voting", "cancelled"],
  voting: ["approved", "rejected"],
  approved: ["executed", "cancelled"],
  executed: [], // Terminal
  rejected: [], // Terminal
  cancelled: [], // Terminal
};

export function isValidDisbursementTransition(
  from: DisbursementStage,
  to: DisbursementStage,
): boolean {
  return VALID_TRANSITIONS[from]?.includes(to) ?? false;
}

// ============ Result Type ============

export interface DisbursementTransitionResult {
  success: boolean;
  newStage?: DisbursementStage;
  error?: string;
  transition?: StageTransition;
}

// ============ Creation ============

export interface CreateDisbursementParams {
  title: string;
  description: string;
  category: DisbursementCategory;
  treasuryFund: TreasuryFund;
  proposer: DID;
  recipientDid: DID;
  amount: bigint;
  justification: string;
  externalReference?: string;
}

/**
 * Create a new disbursement proposal in "proposed" stage.
 * Validates inputs but does NOT check eligibility (caller must do that).
 */
export function createDisbursement(
  params: CreateDisbursementParams,
  currentBlock: bigint,
  disbursementNumber: number,
): { proposal: DisbursementProposal } | { error: string } {
  // Validate category
  if (!VALID_DISBURSEMENT_CATEGORIES.includes(params.category)) {
    return { error: `Invalid category: ${params.category}` };
  }

  // Validate treasury fund
  if (!VALID_TREASURY_FUNDS.includes(params.treasuryFund)) {
    return { error: `Invalid treasury fund: ${params.treasuryFund}` };
  }

  // Validate amount
  if (params.amount <= 0n) {
    return { error: "Amount must be positive" };
  }

  // Validate title
  if (!params.title || params.title.length === 0) {
    return { error: "Title is required" };
  }

  // Validate justification
  if (!params.justification || params.justification.length === 0) {
    return { error: "Justification is required" };
  }

  const proposal: DisbursementProposal = {
    id: `DISB-${disbursementNumber}`,
    title: params.title,
    description: params.description,
    category: params.category,
    proposer: params.proposer,
    treasuryFund: params.treasuryFund,
    recipientDid: params.recipientDid,
    amount: params.amount,
    stage: "proposed",
    stageHistory: [
      {
        from: "proposed" as any,
        to: "proposed" as any,
        at: currentBlock,
        reason: "Disbursement proposal created",
        triggeredBy: params.proposer,
      },
    ],
    stakedAmount: 0n,
    stakeHolder: params.proposer,
    createdAt: currentBlock,
    updatedAt: currentBlock,
    justification: params.justification,
    externalReference: params.externalReference,
  };

  return { proposal };
}

// ============ Stage Transitions ============

/**
 * Transition from proposed → voting by staking.
 * Requires DISBURSEMENT_STAKE (2000 LGM).
 */
export function transitionToVoting(
  proposal: DisbursementProposal,
  stakedAmount: bigint,
  currentBlock: bigint,
  staker: DID,
): DisbursementTransitionResult {
  if (proposal.stage !== "proposed") {
    return {
      success: false,
      error: `Cannot transition from ${proposal.stage} to voting`,
    };
  }

  if (stakedAmount < DISBURSEMENT_STAKE) {
    return {
      success: false,
      error: `Insufficient stake: ${stakedAmount / 10n ** 18n} LGM, need ${DISBURSEMENT_STAKE / 10n ** 18n} LGM`,
    };
  }

  const transition: StageTransition = {
    from: "proposed" as any,
    to: "voting" as any,
    at: currentBlock,
    reason: "Stake deposited, entering voting",
    triggeredBy: staker,
  };

  proposal.stage = "voting";
  proposal.stakedAmount = stakedAmount;
  proposal.stakeHolder = staker;
  proposal.stageHistory.push(transition);
  proposal.updatedAt = currentBlock;

  return { success: true, newStage: "voting", transition };
}

/**
 * Transition from voting → approved | rejected based on vote tally.
 * Uses DISBURSEMENT_THRESHOLDS (80% support, 20% quorum, 15% operator cap).
 */
export function transitionFromVoting(
  proposal: DisbursementProposal,
  votes: VoteTally,
  currentBlock: bigint,
): DisbursementTransitionResult {
  if (proposal.stage !== "voting") {
    return { success: false, error: "Proposal not in voting stage" };
  }

  // Check operator cap violation
  if (votes.operatorCapViolation) {
    return {
      success: false,
      error: `Operator cap violation: single operator exceeds ${DISBURSEMENT_THRESHOLDS.operatorCap * 100}% of votes`,
    };
  }

  // Calculate support and quorum ratios in basis points (10000 = 100%)
  const totalVotes = votes.support + votes.oppose;
  const supportBps =
    totalVotes > 0n ? (votes.support * 10000n) / totalVotes : 0n;
  const totalVoted = votes.support + votes.oppose + votes.abstain;
  const quorumBps =
    votes.totalEligible > 0n ? (totalVoted * 10000n) / votes.totalEligible : 0n;
  const supportRequiredBps = DISBURSEMENT_THRESHOLDS_BPS.supportRequired;
  const quorumRequiredBps = DISBURSEMENT_THRESHOLDS_BPS.quorumRequired;

  // Check thresholds
  if (quorumBps < quorumRequiredBps || supportBps < supportRequiredBps) {
    const supportPct = (Number(supportBps) / 100).toFixed(1);
    const quorumPct = (Number(quorumBps) / 100).toFixed(1);
    const transition: StageTransition = {
      from: "voting" as any,
      to: "rejected" as any,
      at: currentBlock,
      reason: `Vote failed: ${supportPct}% support (need ${DISBURSEMENT_THRESHOLDS.supportRequired * 100}%), ${quorumPct}% quorum (need ${DISBURSEMENT_THRESHOLDS.quorumRequired * 100}%)`,
      triggeredBy: "system",
    };
    proposal.stage = "rejected";
    proposal.stageHistory.push(transition);
    proposal.votes = votes;
    proposal.updatedAt = currentBlock;
    return { success: true, newStage: "rejected", transition };
  }

  // Approved — set timelock
  const supportPct = (Number(supportBps) / 100).toFixed(1);
  const quorumPct = (Number(quorumBps) / 100).toFixed(1);
  const transition: StageTransition = {
    from: "voting" as any,
    to: "approved" as any,
    at: currentBlock,
    reason: `Vote passed: ${supportPct}% support, ${quorumPct}% quorum. Timelock: ${TIMELOCK_BLOCKS} blocks`,
    triggeredBy: "system",
  };
  proposal.stage = "approved";
  proposal.approvedAt = currentBlock;
  proposal.timelockExpiry = currentBlock + TIMELOCK_BLOCKS;
  proposal.stageHistory.push(transition);
  proposal.votes = votes;
  proposal.updatedAt = currentBlock;

  return { success: true, newStage: "approved", transition };
}

/**
 * Execute an approved disbursement.
 * Caller must verify: timelock expired, treasury balance sufficient.
 * This function only validates stage + timelock.
 */
export function executeDisbursement(
  proposal: DisbursementProposal,
  currentBlock: bigint,
): DisbursementTransitionResult {
  if (proposal.stage !== "approved") {
    return {
      success: false,
      error: `Cannot execute from stage: ${proposal.stage}`,
    };
  }

  // Check timelock — defense-in-depth: reject if timelockExpiry was never set
  if (proposal.timelockExpiry === undefined) {
    return {
      success: false,
      error: "Timelock not set for approved proposal",
    };
  }
  if (currentBlock < proposal.timelockExpiry) {
    const remaining = proposal.timelockExpiry - currentBlock;
    return {
      success: false,
      error: `Timelock not expired: ${remaining} blocks remaining`,
    };
  }

  const transition: StageTransition = {
    from: "approved" as any,
    to: "executed" as any,
    at: currentBlock,
    reason: "Disbursement executed after timelock expiry",
    triggeredBy: "system",
  };
  proposal.stage = "executed";
  proposal.executedAt = currentBlock;
  proposal.stageHistory.push(transition);
  proposal.updatedAt = currentBlock;

  return { success: true, newStage: "executed", transition };
}

/**
 * Cancel a disbursement proposal.
 * Only the proposer can cancel. Allowed from "proposed" or "approved" stages.
 */
export function cancelDisbursement(
  proposal: DisbursementProposal,
  currentBlock: bigint,
  canceller: DID,
): DisbursementTransitionResult {
  if (proposal.proposer !== canceller) {
    return {
      success: false,
      error: "Only the proposer can cancel a disbursement",
    };
  }

  if (proposal.stage !== "proposed" && proposal.stage !== "approved") {
    return {
      success: false,
      error: `Cannot cancel from stage: ${proposal.stage}`,
    };
  }

  const transition: StageTransition = {
    from: proposal.stage as any,
    to: "cancelled" as any,
    at: currentBlock,
    reason: "Cancelled by proposer",
    triggeredBy: canceller,
  };
  proposal.stage = "cancelled";
  proposal.stageHistory.push(transition);
  proposal.updatedAt = currentBlock;

  return { success: true, newStage: "cancelled", transition };
}

// ============ Stake Management ============

export interface DisbursementStakeOutcome {
  returnAmount: bigint;
  slashAmount: bigint;
  reason: string;
}

/**
 * Calculate stake return or slash based on disbursement outcome.
 */
export function calculateDisbursementStakeOutcome(
  proposal: DisbursementProposal,
  outcome:
    | "executed"
    | "rejected_malicious"
    | "rejected_frivolous"
    | "rejected_normal"
    | "cancelled",
): DisbursementStakeOutcome {
  switch (outcome) {
    case "executed":
      return {
        returnAmount: proposal.stakedAmount,
        slashAmount: 0n,
        reason: "Disbursement executed — full stake returned",
      };

    case "rejected_malicious":
      return {
        returnAmount: 0n,
        slashAmount: proposal.stakedAmount,
        reason: "Rejected as malicious — full stake slashed",
      };

    case "rejected_frivolous": {
      const slashAmount =
        (proposal.stakedAmount * DISBURSEMENT_SLASH_BPS.frivolous) / 10000n;
      return {
        returnAmount: proposal.stakedAmount - slashAmount,
        slashAmount,
        reason: "Rejected as frivolous — 50% stake slashed",
      };
    }

    case "rejected_normal": {
      const slashAmount =
        (proposal.stakedAmount * DISBURSEMENT_SLASH_BPS.normal) / 10000n;
      return {
        returnAmount: proposal.stakedAmount - slashAmount,
        slashAmount,
        reason: "Rejected — 10% stake slashed",
      };
    }

    case "cancelled": {
      const fee =
        (proposal.stakedAmount * DISBURSEMENT_SLASH_BPS.cancelled) / 10000n;
      return {
        returnAmount: proposal.stakedAmount - fee,
        slashAmount: fee,
        reason: "Cancelled — 5% processing fee",
      };
    }
  }
}
