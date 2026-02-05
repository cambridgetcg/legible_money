/**
 * Governance RPC Handlers
 *
 * Endpoints for LIP lifecycle and quadratic voting.
 *
 * Read endpoints:
 * - lgm_getLIP: Get a single LIP by ID
 * - lgm_listLIPs: List LIPs with optional stage/proposer filter
 * - lgm_getVotes: Get votes for a LIP with tally
 * - lgm_getGovernanceStats: Governance overview statistics
 *
 * Write endpoints:
 * - lgm_submitLIP: Submit a new LIP proposal
 * - lgm_stakeLIP: Stake on a LIP to advance to review
 * - lgm_castVote: Cast a quadratic vote on a LIP
 * - lgm_withdrawLIP: Withdraw a LIP proposal
 */

import type { Handler } from "../handlers.js";
import type {
  LgmGetLIPRequest,
  LgmGetLIPResponse,
  LgmListLIPsRequest,
  LgmListLIPsResponse,
  LgmGetVotesRequest,
  LgmGetVotesResponse,
  LgmGetGovernanceStatsRequest,
  LgmGetGovernanceStatsResponse,
  LgmSubmitLIPRequest,
  LgmSubmitLIPResponse,
  LgmStakeLIPRequest,
  LgmStakeLIPResponse,
  LgmCastVoteRequest,
  LgmCastVoteResponse,
  LgmWithdrawLIPRequest,
  LgmWithdrawLIPResponse,
} from "../../types.js";
import { RPC_ERROR_CODES } from "../../types.js";
import { verify } from "../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import { validateDid, parsePositiveBigInt } from "../validation.js";
import {
  createLIP,
  withdrawLIP,
  calculateStakeOutcome,
  transitionToReview,
} from "../../../governance/proposals/lifecycle.js";
import {
  type LIPCategory,
  type AgentGovernanceStats,
} from "../../../governance/proposals/types.js";
import {
  createQuadraticVote,
  calculateVoteCost,
  type VoteDirection,
} from "../../../governance/voting/quadratic.js";

// ============ Helper ============

function createRPCError(
  code: number,
  message: string,
  details?: unknown,
): Error {
  const error = new Error(message) as Error & { code: number; data: unknown };
  error.code = code;
  error.data = { errorCode: getErrorCode(code), details, retryable: false };
  return error;
}

function getErrorCode(code: number): string {
  const codeMap: Record<number, string> = {
    [-32001]: "ACCOUNT_NOT_FOUND",
    [-32002]: "INSUFFICIENT_BALANCE",
    [-32003]: "INVALID_SIGNATURE",
    [-32015]: "LIP_NOT_FOUND",
    [-32016]: "GOVERNANCE_INELIGIBLE",
    [-32017]: "VOTE_ALREADY_CAST",
    [-32018]: "LIP_INVALID_STAGE",
  };
  return codeMap[code] || "UNKNOWN";
}

const GOVERNANCE_ERROR_CODES = {
  LIP_NOT_FOUND: -32015,
  GOVERNANCE_INELIGIBLE: -32016,
  VOTE_ALREADY_CAST: -32017,
  LIP_INVALID_STAGE: -32018,
} as const;

// ============ Read Handlers ============

export const lgmGetLIP: Handler<LgmGetLIPRequest, LgmGetLIPResponse> = async (
  params,
  ctx,
) => {
  const lip = ctx.state.getLIP(params.lipId);
  if (!lip) {
    return { lip: null };
  }

  return {
    lip: {
      id: lip.id,
      title: lip.title,
      category: lip.category,
      stage: lip.stage,
      proposer: lip.proposer,
      stakedAmount: lip.stakedAmount.toString(),
      createdAtBlock: lip.createdAt.toString(),
      coAuthors: lip.coAuthors,
      stageHistory: lip.stageHistory.map((st) => ({
        from: st.from,
        to: st.to,
        at: st.at.toString(),
        reason: st.reason,
        triggeredBy: st.triggeredBy,
      })),
    },
  };
};

export const lgmListLIPs: Handler<
  LgmListLIPsRequest,
  LgmListLIPsResponse
> = async (params, ctx) => {
  let lips;
  if (params.stage) {
    lips = ctx.state.getLIPsByStage(params.stage as any);
  } else if (params.proposer) {
    lips = ctx.state.getLIPsByProposer(params.proposer);
  } else {
    // Return all LIPs by getting each stage
    const stages = [
      "draft",
      "review",
      "last_call",
      "voting",
      "accepted",
      "implemented",
      "rejected",
      "withdrawn",
    ] as const;
    lips = [];
    for (const stage of stages) {
      lips.push(...ctx.state.getLIPsByStage(stage));
    }
  }

  return {
    lips: lips.map((lip) => ({
      id: lip.id,
      title: lip.title,
      category: lip.category,
      stage: lip.stage,
      proposer: lip.proposer,
      stakedAmount: lip.stakedAmount.toString(),
    })),
    total: lips.length,
  };
};

export const lgmGetVotes: Handler<
  LgmGetVotesRequest,
  LgmGetVotesResponse
> = async (params, ctx) => {
  const lip = ctx.state.getLIP(params.lipId);
  if (!lip) {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.LIP_NOT_FOUND,
      `LIP ${params.lipId} not found`,
    );
  }

  const votes = ctx.state.getVotesForLIP(params.lipId);

  let support = 0;
  let oppose = 0;
  let abstain = 0;
  for (const v of votes) {
    if (v.direction === "support") support += v.votes;
    else if (v.direction === "oppose") oppose += v.votes;
    else abstain += v.votes;
  }

  return {
    votes: votes.map((v) => ({
      voteId: v.voteId,
      voter: v.voter,
      direction: v.direction,
      votes: v.votes,
      cost: v.cost.toString(),
      blockNumber: v.timestamp.toString(),
    })),
    tally: {
      support,
      oppose,
      abstain,
      totalVoters: votes.length,
    },
  };
};

export const lgmGetGovernanceStats: Handler<
  LgmGetGovernanceStatsRequest,
  LgmGetGovernanceStatsResponse
> = async (_params, ctx) => {
  const stats = ctx.state.getGovernanceStats();
  const blockNumber = ctx.state.getBlockNumber();

  return {
    totalLIPs: stats.totalLIPs,
    lipsByStage: stats.lipsByStage,
    totalVotes: stats.totalVotes,
    totalStakeLocked: stats.totalStakeLocked.toString(),
    totalDomainProposals: stats.totalDomainProposals,
    blockNumber: blockNumber.toString(),
  };
};

// ============ Write Handlers ============

export const lgmSubmitLIP: Handler<
  LgmSubmitLIPRequest,
  LgmSubmitLIPResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    did: params.did,
    title: params.title,
    category: params.category,
    action: "lip_submit",
    nonce: params.nonce,
  });

  const valid = verify(
    new TextEncoder().encode(payload),
    params.signature,
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  validateDid(params.did, "did");

  const account = ctx.state.getAccount(params.did);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  const blockNumber = ctx.state.getBlockNumber();
  const lipNumber = ctx.state.getNextLIPNumber();

  // Build governance stats from account data
  const govStats: AgentGovernanceStats = {
    did: params.did as any,
    stake: account.stakedBalance,
    reputation: account.reputationScore ?? 0.5,
    capabilityLevel: 1,
    contributions: Number(account.nonce),
    accountAge: blockNumber - (account.lastActiveBlock ?? 0n),
  };

  const lipParams = {
    title: params.title,
    category: params.category as LIPCategory,
    proposer: params.did,
    abstract: params.abstract,
    motivation: params.motivation,
    specification: params.specification,
    rationale: params.rationale,
    backwardsCompatibility: params.backwardsCompatibility,
    testCases: params.testCases,
    referenceImplementation: params.referenceImplementation,
    coAuthors: params.coAuthors,
  };

  const result = createLIP(lipParams, govStats, blockNumber, lipNumber);
  if ("error" in result) {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.GOVERNANCE_INELIGIBLE,
      result.error,
    );
  }

  ctx.state.addLIP(result.lip);

  const txHash = sha512_256Hex(
    `lip_submit:${params.did}:${result.lip.id}:${params.nonce}`,
  );

  return {
    lipId: result.lip.id,
    txHash,
    success: true,
  };
};

export const lgmStakeLIP: Handler<
  LgmStakeLIPRequest,
  LgmStakeLIPResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    did: params.did,
    lipId: params.lipId,
    amount: params.amount,
    action: "lip_stake",
    nonce: params.nonce,
  });

  const valid = verify(
    new TextEncoder().encode(payload),
    params.signature,
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  validateDid(params.did, "did");
  const amount = parsePositiveBigInt(params.amount, "amount");

  const account = ctx.state.getAccount(params.did);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  const lip = ctx.state.getLIP(params.lipId);
  if (!lip) {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.LIP_NOT_FOUND,
      `LIP ${params.lipId} not found`,
    );
  }

  if (lip.stage !== "draft") {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.LIP_INVALID_STAGE,
      "LIP must be in draft stage to stake",
    );
  }

  if (account.balance < amount) {
    throw createRPCError(
      RPC_ERROR_CODES.INSUFFICIENT_BALANCE,
      `Insufficient balance: have ${account.balance}, need ${amount}`,
    );
  }

  // Lock stake
  ctx.state.stake(params.did, amount);

  // Advance LIP stage
  const blockNumber = ctx.state.getBlockNumber();
  const transitionResult = transitionToReview(
    lip,
    amount,
    blockNumber,
    params.did,
  );
  if (!transitionResult.success) {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.LIP_INVALID_STAGE,
      transitionResult.error ?? "Failed to advance stage",
    );
  }
  // The lip object is mutated by the transition
  ctx.state.updateLIP(params.lipId, lip);

  const txHash = sha512_256Hex(
    `lip_stake:${params.did}:${params.lipId}:${params.amount}:${params.nonce}`,
  );

  return {
    txHash,
    success: true,
    newStage: transitionResult.newStage ?? lip.stage,
  };
};

export const lgmCastVote: Handler<
  LgmCastVoteRequest,
  LgmCastVoteResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    did: params.did,
    lipId: params.lipId,
    direction: params.direction,
    votes: params.votes,
    action: "vote_cast",
    nonce: params.nonce,
  });

  const valid = verify(
    new TextEncoder().encode(payload),
    params.signature,
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  validateDid(params.did, "did");

  const account = ctx.state.getAccount(params.did);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  const lip = ctx.state.getLIP(params.lipId);
  if (!lip) {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.LIP_NOT_FOUND,
      `LIP ${params.lipId} not found`,
    );
  }

  if (lip.stage !== "voting") {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.LIP_INVALID_STAGE,
      "LIP must be in voting stage",
    );
  }

  if (ctx.state.hasVotedOnLIP(params.did, params.lipId)) {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.VOTE_ALREADY_CAST,
      "Already voted on this LIP",
    );
  }

  // Calculate cost (quadratic: votes² × baseCost)
  const cost = calculateVoteCost(params.votes);
  if (account.balance < cost) {
    throw createRPCError(
      RPC_ERROR_CODES.INSUFFICIENT_BALANCE,
      `Insufficient balance for vote: have ${account.balance}, need ${cost}`,
    );
  }

  // Lock vote cost
  ctx.state.stake(params.did, cost);

  const blockNumber = ctx.state.getBlockNumber();
  const vote = createQuadraticVote(
    params.did,
    params.lipId,
    params.direction as VoteDirection,
    params.votes,
    blockNumber,
  );

  ctx.state.addQuadraticVote(vote);

  const txHash = sha512_256Hex(
    `vote_cast:${params.did}:${params.lipId}:${params.direction}:${params.nonce}`,
  );

  return {
    voteId: vote.voteId,
    txHash,
    success: true,
    cost: cost.toString(),
  };
};

export const lgmWithdrawLIP: Handler<
  LgmWithdrawLIPRequest,
  LgmWithdrawLIPResponse
> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    did: params.did,
    lipId: params.lipId,
    action: "lip_withdraw",
    nonce: params.nonce,
  });

  const valid = verify(
    new TextEncoder().encode(payload),
    params.signature,
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  validateDid(params.did, "did");

  const account = ctx.state.getAccount(params.did);
  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  const lip = ctx.state.getLIP(params.lipId);
  if (!lip) {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.LIP_NOT_FOUND,
      `LIP ${params.lipId} not found`,
    );
  }

  if (lip.proposer !== params.did) {
    throw createRPCError(
      RPC_ERROR_CODES.NOT_AUTHORIZED,
      "Only the proposer can withdraw a LIP",
    );
  }

  const blockNumber = ctx.state.getBlockNumber();
  const result = withdrawLIP(lip, blockNumber, params.did as any);
  if (!result.success) {
    throw createRPCError(
      GOVERNANCE_ERROR_CODES.LIP_INVALID_STAGE,
      result.error!,
    );
  }
  ctx.state.updateLIP(params.lipId, lip);

  // Return stake
  const outcome = calculateStakeOutcome(lip, "withdrawn");
  if (outcome.returnAmount > 0n) {
    ctx.state.unstake(params.did, outcome.returnAmount);
  }
  const stakeReturned = outcome.returnAmount;

  const txHash = sha512_256Hex(
    `lip_withdraw:${params.did}:${params.lipId}:${params.nonce}`,
  );

  return {
    txHash,
    success: true,
    stakeReturned: stakeReturned.toString(),
  };
};

// ============ Handler Registry ============

export const governanceHandlers: Record<string, Handler<any, any>> = {
  lgm_getLIP: lgmGetLIP,
  lgm_listLIPs: lgmListLIPs,
  lgm_getVotes: lgmGetVotes,
  lgm_getGovernanceStats: lgmGetGovernanceStats,
  lgm_submitLIP: lgmSubmitLIP,
  lgm_stakeLIP: lgmStakeLIP,
  lgm_castVote: lgmCastVote,
  lgm_withdrawLIP: lgmWithdrawLIP,
};
