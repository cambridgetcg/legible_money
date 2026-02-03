/**
 * Unified Reward Router
 *
 * ALL rewards in the protocol flow through this router.
 * 22% of every reward is captured for the research fund.
 *
 * This ensures consistent funding for IRL research verification
 * regardless of reward source (blocks, verification, claims, API, etc.)
 */

import type { DID } from '../../agents/identity/did.js';
import {
  RESEARCH_SHARE_RATE,
  RESEARCH_TREASURY_DID,
} from '../research/types.js';

// ============ Types ============

/**
 * Source of the reward
 */
export type RewardSource =
  | 'block_production'      // Block producer reward
  | 'verification'          // Validator verification reward
  | 'claim_submission'      // Claim submitter reward
  | 'api_call'              // API provider/knowledge pool
  | 'citation'              // Citation/ancestor reward
  | 'challenge_win'         // Challenge winner reward
  | 'challenge_defense'     // Successful defense reward
  | 'bounty_fulfillment';   // Research bounty reward

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
 * @param input - Reward to route
 * @param shareRate - Research share rate (default: 0.22)
 * @returns Routed reward with research and recipient amounts
 */
export function routeReward(
  input: RewardInput,
  shareRate: number = RESEARCH_SHARE_RATE
): RoutedReward {
  // Calculate research share (22%)
  const researchShare = BigInt(
    Math.floor(Number(input.amount) * shareRate)
  );

  // Remaining goes to recipient (78%)
  const recipientAmount = input.amount - researchShare;

  return {
    input,
    researchShare,
    recipientAmount,
    researchTreasury: RESEARCH_TREASURY_DID,
  };
}

/**
 * Route multiple rewards as a batch
 *
 * @param inputs - Rewards to route
 * @param shareRate - Research share rate (default: 0.22)
 * @returns Batch of routed rewards
 */
export function routeRewardBatch(
  inputs: RewardInput[],
  shareRate: number = RESEARCH_SHARE_RATE
): RewardBatch {
  const rewards = inputs.map(input => routeReward(input, shareRate));

  const totalResearchShare = rewards.reduce(
    (sum, r) => sum + r.researchShare,
    0n
  );

  const totalRecipientAmount = rewards.reduce(
    (sum, r) => sum + r.recipientAmount,
    0n
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
  blockNumber: bigint
): RoutedReward {
  return routeReward({
    source: 'block_production',
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
  blockNumber: bigint
): RoutedReward {
  return routeReward({
    source: 'verification',
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
  blockNumber: bigint
): RoutedReward {
  return routeReward({
    source: 'claim_submission',
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
  blockNumber: bigint
): RoutedReward {
  return routeReward({
    source: 'citation',
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
  blockNumber: bigint
): RoutedReward {
  return routeReward({
    source: 'challenge_win',
    amount,
    recipient: winner,
    factId,
    blockNumber,
  });
}

// ============ Utilities ============

/**
 * Calculate research share for an amount
 */
export function calculateResearchShare(
  amount: bigint,
  shareRate: number = RESEARCH_SHARE_RATE
): bigint {
  return BigInt(Math.floor(Number(amount) * shareRate));
}

/**
 * Calculate recipient amount after research share
 */
export function calculateRecipientAmount(
  amount: bigint,
  shareRate: number = RESEARCH_SHARE_RATE
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
  const totalInput = batch.rewards.reduce(
    (sum, r) => sum + r.input.amount,
    0n
  );
  const totalOutput = batch.totalResearchShare + batch.totalRecipientAmount;
  return totalInput === totalOutput;
}

// ============ Event Generation ============

/**
 * Create reward distribution event
 */
export function createDistributionEvent(
  routed: RoutedReward,
  timestamp: number
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
  bySource: Record<RewardSource, {
    count: number;
    total: bigint;
    researchShare: bigint;
  }>;
}

/**
 * Create empty reward stats
 */
export function createEmptyStats(): RewardStats {
  const bySource: RewardStats['bySource'] = {} as RewardStats['bySource'];
  const sources: RewardSource[] = [
    'block_production',
    'verification',
    'claim_submission',
    'api_call',
    'citation',
    'challenge_win',
    'challenge_defense',
    'bounty_fulfillment',
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
  routed: RoutedReward
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
        researchShare: stats.bySource[source].researchShare + routed.researchShare,
      },
    },
  };
}

// ============ Exports ============

export {
  RESEARCH_SHARE_RATE,
  RESEARCH_TREASURY_DID,
};
