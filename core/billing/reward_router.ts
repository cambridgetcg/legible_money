/**
 * Unified Reward Router
 *
 * ALL rewards in the protocol flow through this router.
 * 22% of every reward is captured for the research fund.
 *
 * This ensures consistent funding for IRL research verification
 * regardless of reward source (blocks, verification, claims, API, etc.)
 */

import type { DID } from "../../core/agents/identity/did.js";
import {
  RESEARCH_SHARE_RATE,
  RESEARCH_TREASURY_DID,
} from "../research/types.js";
import { TREASURY } from "../constants.js";

// ============ Constants ============

/**
 * VULN-016 FIX: Research share rate in basis points (1 bps = 0.01%)
 * 2200 bps = 22%
 *
 * Using basis points allows pure bigint arithmetic without precision loss.
 */
export const RESEARCH_SHARE_RATE_BPS = 2200n;

// ============ Types ============

/**
 * Source of the reward
 */
export type RewardSource =
  | "block_production" // Block producer reward
  | "verification" // Validator verification reward
  | "claim_submission" // Claim submitter reward
  | "api_call" // API provider/knowledge pool
  | "citation" // Citation/ancestor reward
  | "challenge_win" // Challenge winner reward
  | "challenge_defense" // Successful defense reward
  | "bounty_fulfillment"; // Research bounty reward

/**
 * Reward before routing
 */
export interface RewardInput {
  /** Source of the reward */
  source: RewardSource;

  /** Original reward amount */
  amount: bigint;

  /** Intended recipient */
  recipient: DID;

  /** Related fact ID (if applicable) */
  factId?: string;

  /** Block number of reward */
  blockNumber: bigint;

  /** Additional metadata */
  metadata?: Record<string, unknown>;
}

/**
 * Reward after routing (with research share deducted)
 */
export interface RoutedReward {
  /** Original input */
  input: RewardInput;

  /** Amount going to research fund (22%) */
  researchShare: bigint;

  /** Amount going to original recipient (78%) */
  recipientAmount: bigint;

  /** Research treasury DID */
  researchTreasury: DID;
}

/**
 * Batch of routed rewards
 */
export interface RewardBatch {
  /** Individual routed rewards */
  rewards: RoutedReward[];

  /** Total research share for batch */
  totalResearchShare: bigint;

  /** Total to recipients for batch */
  totalRecipientAmount: bigint;

  /** Block number */
  blockNumber: bigint;
}

/**
 * Reward distribution event
 */
export interface RewardDistributionEvent {
  id: string;
  blockNumber: bigint;
  timestamp: number;
  source: RewardSource;
  originalAmount: bigint;
  researchShare: bigint;
  recipientAmount: bigint;
  recipient: DID;
  factId?: string;
}

// ============ Router Implementation ============

/**
 * Route a single reward through the research share deduction
 *
 * VULN-016 FIX: Uses pure bigint arithmetic to prevent precision loss.
 * Previously used Number conversion which loses precision for amounts > 2^53.
 *
 * @param input - Reward to route
 * @param shareRateBps - Research share rate in basis points (default: 2200 = 22%)
 * @returns Routed reward with research and recipient amounts
 */
export function routeReward(
  input: RewardInput,
  shareRateBps: bigint = RESEARCH_SHARE_RATE_BPS,
): RoutedReward {
  // VULN-016 FIX: Pure bigint arithmetic for precision
  // Calculate research share: amount * rateBps / 10000
  const researchShare = (input.amount * shareRateBps) / 10000n;

  // Remaining goes to recipient
  const recipientAmount = input.amount - researchShare;

  // Sanity check: ensure no rounding errors create invalid state
  if (researchShare + recipientAmount !== input.amount) {
    throw new Error(
      `Reward routing arithmetic error: ${researchShare} + ${recipientAmount} !== ${input.amount}`,
    );
  }

  return {
    input,
    researchShare,
    recipientAmount,
    researchTreasury: RESEARCH_TREASURY_DID,
  };
}

/**
 * VULN-016 FIX: Legacy function for backwards compatibility
 * @deprecated Use routeReward with basis points instead
 */
export function routeRewardLegacy(
  input: RewardInput,
  shareRate: number = RESEARCH_SHARE_RATE,
): RoutedReward {
  // Convert float rate to basis points
  const shareRateBps = BigInt(Math.round(shareRate * 10000));
  return routeReward(input, shareRateBps);
}

/**
 * Route multiple rewards as a batch
 *
 * VULN-016 FIX: Uses basis points for precision
 *
 * @param inputs - Rewards to route
 * @param shareRateBps - Research share rate in basis points (default: 2200 = 22%)
 * @returns Batch of routed rewards
 */
export function routeRewardBatch(
  inputs: RewardInput[],
  shareRateBps: bigint = RESEARCH_SHARE_RATE_BPS,
): RewardBatch {
  const rewards = inputs.map((input) => routeReward(input, shareRateBps));

  const totalResearchShare = rewards.reduce(
    (sum, r) => sum + r.researchShare,
    0n,
  );

  const totalRecipientAmount = rewards.reduce(
    (sum, r) => sum + r.recipientAmount,
    0n,
  );

  const blockNumber = inputs[0]?.blockNumber ?? 0n;

  return {
    rewards,
    totalResearchShare,
    totalRecipientAmount,
    blockNumber,
  };
}

// ============ Specialized Routers ============

/**
 * Route block production reward
 */
export function routeBlockReward(
  amount: bigint,
  producer: DID,
  blockNumber: bigint,
): RoutedReward {
  return routeReward({
    source: "block_production",
    amount,
    recipient: producer,
    blockNumber,
  });
}

/**
 * Route verification reward
 */
export function routeVerificationReward(
  amount: bigint,
  validator: DID,
  factId: string,
  blockNumber: bigint,
): RoutedReward {
  return routeReward({
    source: "verification",
    amount,
    recipient: validator,
    factId,
    blockNumber,
  });
}

/**
 * Route claim submission reward
 */
export function routeClaimReward(
  amount: bigint,
  submitter: DID,
  factId: string,
  blockNumber: bigint,
): RoutedReward {
  return routeReward({
    source: "claim_submission",
    amount,
    recipient: submitter,
    factId,
    blockNumber,
  });
}

/**
 * Route API/citation reward
 */
export function routeCitationReward(
  amount: bigint,
  recipient: DID,
  factId: string,
  blockNumber: bigint,
): RoutedReward {
  return routeReward({
    source: "citation",
    amount,
    recipient,
    factId,
    blockNumber,
  });
}

/**
 * Route challenge winner reward
 */
export function routeChallengeWinReward(
  amount: bigint,
  winner: DID,
  factId: string,
  blockNumber: bigint,
): RoutedReward {
  return routeReward({
    source: "challenge_win",
    amount,
    recipient: winner,
    factId,
    blockNumber,
  });
}

// ============ Utilities ============

/**
 * Calculate research share for an amount
 *
 * VULN-016 FIX: Uses basis points for precision.
 * Accepts either a float rate (legacy) or basis points.
 */
export function calculateResearchShare(
  amount: bigint,
  shareRate: number = RESEARCH_SHARE_RATE,
): bigint {
  // Convert float rate to basis points for pure bigint arithmetic
  const bps = BigInt(Math.round(shareRate * 10000));
  return (amount * bps) / 10000n;
}

/**
 * Calculate recipient amount after research share
 */
export function calculateRecipientAmount(
  amount: bigint,
  shareRate: number = RESEARCH_SHARE_RATE,
): bigint {
  return amount - calculateResearchShare(amount, shareRate);
}

/**
 * Validate that routing preserves total amount
 */
export function validateRouting(routed: RoutedReward): boolean {
  return routed.researchShare + routed.recipientAmount === routed.input.amount;
}

/**
 * Validate batch routing preserves total amount
 */
export function validateBatchRouting(batch: RewardBatch): boolean {
  const totalInput = batch.rewards.reduce((sum, r) => sum + r.input.amount, 0n);
  const totalOutput = batch.totalResearchShare + batch.totalRecipientAmount;
  return totalInput === totalOutput;
}

// ============ Evidence Tax Treasury Split ============

/**
 * Destination in the 5-way treasury split
 */
export interface TreasurySplitEntry {
  did: string;
  amount: bigint;
  fund: "research" | "foundation" | "community" | "developers" | "reserve";
}

/**
 * Split the evidence tax (22%) across all 5 treasury funds
 * per the frozen allocation in core/constants.ts.
 *
 * 77% Research, 7% Foundation, 7% Community, 7% Developers, 2% Reserve
 */
export function splitEvidenceTax(
  evidenceTaxAmount: bigint,
): TreasurySplitEntry[] {
  const researchBps = BigInt(Math.round(TREASURY.ALLOCATION.RESEARCH * 10000));
  const foundationBps = BigInt(
    Math.round(TREASURY.ALLOCATION.FOUNDATION * 10000),
  );
  const communityBps = BigInt(
    Math.round(TREASURY.ALLOCATION.COMMUNITY * 10000),
  );
  const developersBps = BigInt(
    Math.round(TREASURY.ALLOCATION.DEVELOPERS * 10000),
  );

  const research = (evidenceTaxAmount * researchBps) / 10000n;
  const foundation = (evidenceTaxAmount * foundationBps) / 10000n;
  const community = (evidenceTaxAmount * communityBps) / 10000n;
  const developers = (evidenceTaxAmount * developersBps) / 10000n;
  // Reserve gets the remainder to prevent rounding dust loss
  const reserve =
    evidenceTaxAmount - research - foundation - community - developers;

  return [
    { did: TREASURY.ADDRESSES.RESEARCH, amount: research, fund: "research" },
    {
      did: TREASURY.ADDRESSES.FOUNDATION,
      amount: foundation,
      fund: "foundation",
    },
    { did: TREASURY.ADDRESSES.COMMUNITY, amount: community, fund: "community" },
    {
      did: TREASURY.ADDRESSES.DEVELOPERS,
      amount: developers,
      fund: "developers",
    },
    { did: TREASURY.ADDRESSES.RESERVE, amount: reserve, fund: "reserve" },
  ];
}

// ============ Event Generation ============

/**
 * Create reward distribution event
 */
export function createDistributionEvent(
  routed: RoutedReward,
  timestamp: number,
): RewardDistributionEvent {
  return {
    id: `reward_${routed.input.source}_${routed.input.blockNumber}_${Date.now()}`,
    blockNumber: routed.input.blockNumber,
    timestamp,
    source: routed.input.source,
    originalAmount: routed.input.amount,
    researchShare: routed.researchShare,
    recipientAmount: routed.recipientAmount,
    recipient: routed.input.recipient,
    factId: routed.input.factId,
  };
}

// ============ Statistics ============

/**
 * Reward statistics
 */
export interface RewardStats {
  totalDistributed: bigint;
  totalToResearch: bigint;
  totalToRecipients: bigint;
  bySource: Record<
    RewardSource,
    {
      count: number;
      total: bigint;
      researchShare: bigint;
    }
  >;
}

/**
 * Create empty reward stats
 */
export function createEmptyStats(): RewardStats {
  const bySource: RewardStats["bySource"] = {} as RewardStats["bySource"];
  const sources: RewardSource[] = [
    "block_production",
    "verification",
    "claim_submission",
    "api_call",
    "citation",
    "challenge_win",
    "challenge_defense",
    "bounty_fulfillment",
  ];

  for (const source of sources) {
    bySource[source] = { count: 0, total: 0n, researchShare: 0n };
  }

  return {
    totalDistributed: 0n,
    totalToResearch: 0n,
    totalToRecipients: 0n,
    bySource,
  };
}

/**
 * Update stats with a routed reward
 */
export function updateStats(
  stats: RewardStats,
  routed: RoutedReward,
): RewardStats {
  const source = routed.input.source;

  return {
    totalDistributed: stats.totalDistributed + routed.input.amount,
    totalToResearch: stats.totalToResearch + routed.researchShare,
    totalToRecipients: stats.totalToRecipients + routed.recipientAmount,
    bySource: {
      ...stats.bySource,
      [source]: {
        count: stats.bySource[source].count + 1,
        total: stats.bySource[source].total + routed.input.amount,
        researchShare:
          stats.bySource[source].researchShare + routed.researchShare,
      },
    },
  };
}

// ============ Exports ============

export { RESEARCH_SHARE_RATE, RESEARCH_TREASURY_DID };
