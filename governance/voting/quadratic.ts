/**
 * Quadratic Voting
 *
 * Implements quadratic voting where the cost of votes increases
 * quadratically. This prevents wealthy participants from dominating
 * while still allowing strong preferences to be expressed.
 *
 * PP HIERARCHY:
 *   TRUTH: Mathematical relationship between cost and votes
 *   UNDERSTANDING: Clear cost structure
 *   BEAUTY: Elegant quadratic function
 *   JUSTICE: Prevents plutocratic capture
 *   CREATIVITY: Adapted for stake-based governance
 */

import type { DID } from "../../core/agents/identity/did.js";

// ============ Configuration ============

/**
 * Quadratic voting configuration.
 */
export interface QuadraticVotingConfig {
  baseCost: bigint; // Cost of 1 vote (in LGM wei)
  maxVotesPerVoter: number; // Maximum votes any voter can cast
  refundOnCancel: number; // % refunded if vote is cancelled
}

export const DEFAULT_QV_CONFIG: QuadraticVotingConfig = {
  baseCost: 1n * 10n ** 18n, // 1 LGM per vote (squared for cost)
  maxVotesPerVoter: 100, // Max 100 votes = 10000 LGM cost
  refundOnCancel: 0.9, // 90% refund
};

// ============ Vote Types ============

export type VoteDirection = "support" | "oppose" | "abstain";

/**
 * A quadratic vote.
 */
export interface QuadraticVote {
  voteId: string;
  voter: DID;
  lipId: string;
  direction: VoteDirection;
  votes: number; // Number of votes (cost = votes^2 * baseCost)
  cost: bigint;
  timestamp: bigint;
  locked: boolean; // Whether vote is locked (conviction)
  lockDuration: bigint;
}

// ============ Cost Calculations ============

/**
 * Calculate the cost for a given number of votes.
 * Cost = votes^2 * baseCost
 *
 * Examples (with baseCost = 1 LGM):
 * - 1 vote = 1 LGM
 * - 2 votes = 4 LGM
 * - 3 votes = 9 LGM
 * - 10 votes = 100 LGM
 */
export function calculateVoteCost(
  votes: number,
  config: QuadraticVotingConfig = DEFAULT_QV_CONFIG,
): bigint {
  if (votes < 0) {
    throw new Error("Votes must be non-negative");
  }
  if (votes > config.maxVotesPerVoter) {
    throw new Error(
      `Votes exceed maximum: ${votes} > ${config.maxVotesPerVoter}`,
    );
  }

  return BigInt(votes * votes) * config.baseCost;
}

/**
 * Calculate the maximum votes possible with a given budget.
 * votes = floor(sqrt(budget / baseCost))
 */
export function calculateMaxVotes(
  budget: bigint,
  config: QuadraticVotingConfig = DEFAULT_QV_CONFIG,
): number {
  if (budget < 0n) {
    throw new Error("Budget must be non-negative");
  }

  // votes = sqrt(budget / baseCost)
  const quotient = budget / config.baseCost;

  // Integer square root using Newton's method
  const votes = Number(isqrt(quotient));

  // Cap at max votes
  return Math.min(votes, config.maxVotesPerVoter);
}

/**
 * Integer square root using Newton's method.
 */
function isqrt(n: bigint): bigint {
  if (n < 0n) throw new Error("Square root of negative number");
  if (n < 2n) return n;

  let x = n;
  let y = (x + 1n) / 2n;

  while (y < x) {
    x = y;
    y = (x + n / x) / 2n;
  }

  return x;
}

/**
 * Calculate the marginal cost of the next vote.
 * Marginal cost = (n+1)^2 - n^2 = 2n + 1
 */
export function calculateMarginalCost(
  currentVotes: number,
  config: QuadraticVotingConfig = DEFAULT_QV_CONFIG,
): bigint {
  // Cost of adding one more vote
  const currentCost = calculateVoteCost(currentVotes, config);
  const newCost = calculateVoteCost(currentVotes + 1, config);
  return newCost - currentCost;
}

/**
 * Calculate refund amount for cancelled votes.
 */
export function calculateRefund(
  originalCost: bigint,
  config: QuadraticVotingConfig = DEFAULT_QV_CONFIG,
): bigint {
  // Use basis points for precision
  const refundBasisPoints = BigInt(Math.floor(config.refundOnCancel * 10000));
  return (originalCost * refundBasisPoints) / 10000n;
}

// ============ Vote Management ============

/**
 * Create a quadratic vote.
 */
export function createQuadraticVote(
  voter: DID,
  lipId: string,
  direction: VoteDirection,
  votes: number,
  timestamp: bigint,
  config: QuadraticVotingConfig = DEFAULT_QV_CONFIG,
): QuadraticVote {
  const cost = calculateVoteCost(votes, config);

  return {
    voteId: `qv-${lipId}-${voter}-${timestamp}`,
    voter,
    lipId,
    direction,
    votes,
    cost,
    timestamp,
    locked: false,
    lockDuration: 0n,
  };
}

/**
 * Update a quadratic vote (increase or decrease votes).
 */
export function updateQuadraticVote(
  vote: QuadraticVote,
  newVotes: number,
  config: QuadraticVotingConfig = DEFAULT_QV_CONFIG,
): { updatedVote: QuadraticVote; costDelta: bigint } {
  if (vote.locked) {
    throw new Error("Cannot update locked vote");
  }

  const newCost = calculateVoteCost(newVotes, config);
  const costDelta = newCost - vote.cost;

  const updatedVote: QuadraticVote = {
    ...vote,
    votes: newVotes,
    cost: newCost,
  };

  return { updatedVote, costDelta };
}

/**
 * Cancel a quadratic vote.
 */
export function cancelQuadraticVote(
  vote: QuadraticVote,
  config: QuadraticVotingConfig = DEFAULT_QV_CONFIG,
): { cancelledVote: QuadraticVote; refund: bigint } {
  if (vote.locked) {
    throw new Error("Cannot cancel locked vote");
  }

  const refund = calculateRefund(vote.cost, config);

  const cancelledVote: QuadraticVote = {
    ...vote,
    votes: 0,
    cost: 0n,
  };

  return { cancelledVote, refund };
}

// ============ Vote Aggregation ============

/**
 * Aggregated quadratic votes for a LIP.
 */
export interface QuadraticVoteTally {
  lipId: string;
  support: {
    totalVotes: number;
    totalCost: bigint;
    voterCount: number;
  };
  oppose: {
    totalVotes: number;
    totalCost: bigint;
    voterCount: number;
  };
  abstain: {
    totalVotes: number;
    totalCost: bigint;
    voterCount: number;
  };
  netSupport: number; // support - oppose
  participationCost: bigint; // Total LGM committed
}

/**
 * Aggregate quadratic votes.
 */
export function aggregateQuadraticVotes(
  votes: QuadraticVote[],
): QuadraticVoteTally {
  if (votes.length === 0 || !votes[0]) {
    throw new Error("No votes to aggregate");
  }

  const lipId = votes[0].lipId;

  const support = { totalVotes: 0, totalCost: 0n, voterCount: 0 };
  const oppose = { totalVotes: 0, totalCost: 0n, voterCount: 0 };
  const abstain = { totalVotes: 0, totalCost: 0n, voterCount: 0 };

  for (const vote of votes) {
    if (vote.lipId !== lipId) {
      throw new Error("Votes are for different LIPs");
    }

    switch (vote.direction) {
      case "support":
        support.totalVotes += vote.votes;
        support.totalCost += vote.cost;
        support.voterCount++;
        break;
      case "oppose":
        oppose.totalVotes += vote.votes;
        oppose.totalCost += vote.cost;
        oppose.voterCount++;
        break;
      case "abstain":
        abstain.totalVotes += vote.votes;
        abstain.totalCost += vote.cost;
        abstain.voterCount++;
        break;
    }
  }

  return {
    lipId,
    support,
    oppose,
    abstain,
    netSupport: support.totalVotes - oppose.totalVotes,
    participationCost: support.totalCost + oppose.totalCost + abstain.totalCost,
  };
}

// ============ Vote Validation ============

/**
 * Validate a quadratic vote.
 */
export function validateQuadraticVote(
  vote: QuadraticVote,
  voterBalance: bigint,
  config: QuadraticVotingConfig = DEFAULT_QV_CONFIG,
): { valid: boolean; error?: string } {
  // Check vote count
  if (vote.votes < 0) {
    return { valid: false, error: "Negative votes" };
  }

  if (vote.votes > config.maxVotesPerVoter) {
    return {
      valid: false,
      error: `Exceeds max votes: ${config.maxVotesPerVoter}`,
    };
  }

  // Verify cost calculation
  const expectedCost = calculateVoteCost(vote.votes, config);
  if (vote.cost !== expectedCost) {
    return { valid: false, error: "Cost mismatch" };
  }

  // Check balance
  if (voterBalance < vote.cost) {
    return {
      valid: false,
      error: `Insufficient balance: ${voterBalance} < ${vote.cost}`,
    };
  }

  return { valid: true };
}

// ============ Utility Functions ============

/**
 * Calculate effective vote power considering all factors.
 * This is the base for conviction and domain weighting.
 */
export function calculateEffectiveVotePower(
  votes: number,
  convictionMultiplier: number,
  domainMultiplier: number,
): number {
  return votes * convictionMultiplier * domainMultiplier;
}

/**
 * Get vote cost table for display.
 */
export function getVoteCostTable(
  maxVotes: number = 10,
  config: QuadraticVotingConfig = DEFAULT_QV_CONFIG,
): Array<{ votes: number; cost: string; marginal: string }> {
  const table: Array<{ votes: number; cost: string; marginal: string }> = [];

  for (let votes = 1; votes <= maxVotes; votes++) {
    const cost = calculateVoteCost(votes, config);
    const marginal = calculateMarginalCost(votes - 1, config);

    table.push({
      votes,
      cost: `${cost / 10n ** 18n} LGM`,
      marginal: `${marginal / 10n ** 18n} LGM`,
    });
  }

  return table;
}

/**
 * Format vote cost for display.
 */
export function formatVoteCost(cost: bigint): string {
  const lgm = Number(cost / 10n ** 18n);
  const remainder = Number(cost % 10n ** 18n) / 10 ** 18;

  if (remainder === 0) {
    return `${lgm} LGM`;
  }

  return `${(lgm + remainder).toFixed(2)} LGM`;
}
