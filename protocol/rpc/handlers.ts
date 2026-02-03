/**
 * RPC Handlers
 *
 * Handler implementations for all RPC methods.
 * Each handler takes params and returns result or throws.
 */

import type { StateManager } from '../../state/state_manager.js';
import type { ProofOfTruthEngine } from '../../consensus/pot/engine.js';
import type { Mempool } from '../../consensus/mempool/pool.js';
import type {
  RPCError,
  AccountGetBalanceRequest,
  AccountGetBalanceResponse,
  AccountGetInfoRequest,
  AccountGetInfoResponse,
  AgentCheckAccountRequest,
  AgentCheckAccountResponse,
  AgentGetStatusRequest,
  AgentGetStatusResponse,
  KnowledgeGetFactsRequest,
  KnowledgeGetFactsResponse,
  KnowledgeGetFactRequest,
  KnowledgeGetFactResponse,
  KnowledgeSubmitClaimRequest,
  KnowledgeSubmitClaimResponse,
  VerificationGetPendingRequest,
  VerificationGetPendingResponse,
  NetworkHealthResponse,
  RPC_ERROR_CODES,
} from '../types.js';
import { verify } from '../../core/crypto/signatures.js';
import type { PublicKey } from '../../core/crypto/keys.js';

// ============ Handler Context ============

export interface HandlerContext {
  state: StateManager;
  consensus: ProofOfTruthEngine;
  mempool: Mempool;
  nodeVersion: string;
  networkId: string;
}

// ============ Handler Type ============

export type Handler<TParams, TResult> = (
  params: TParams,
  ctx: HandlerContext
) => Promise<TResult>;

// ============ Account Handlers ============

export const accountGetBalance: Handler<AccountGetBalanceRequest, AccountGetBalanceResponse> = async (params, ctx) => {
  const account = ctx.state.getAccount(params.did);

  if (!account) {
    return {
      balance: '0',
      stakedBalance: '0',
      nonce: '0',
    };
  }

  return {
    balance: account.balance.toString(),
    stakedBalance: account.stakedBalance.toString(),
    nonce: account.nonce.toString(),
  };
};

export const accountGetInfo: Handler<AccountGetInfoRequest, AccountGetInfoResponse> = async (params, ctx) => {
  const account = ctx.state.getAccount(params.did);
  const blockNumber = ctx.state.getBlockNumber();

  const response: AccountGetInfoResponse = {
    account,
    blockNumber: blockNumber.toString(),
  };

  if (params.includeProof && account) {
    // Generate proof (would need to expose this from state manager)
    // response.proof = ctx.state.generateAccountProof(params.did);
  }

  return response;
};

// ============ Agent Handlers ============

export const agentCheckAccount: Handler<AgentCheckAccountRequest, AgentCheckAccountResponse> = async (params, ctx) => {
  const account = ctx.state.getAccount(params.did);

  if (!account) {
    return { exists: false };
  }

  const validator = ctx.state.getValidator(account.operationalKeyHash as PublicKey);

  return {
    exists: true,
    balance: account.balance.toString(),
    reputation: account.reputationScore,
    isValidator: validator?.isActive ?? false,
  };
};

export const agentGetStatus: Handler<AgentGetStatusRequest, AgentGetStatusResponse> = async (params, ctx) => {
  const account = ctx.state.getAccount(params.did);

  if (!account) {
    throw createRPCError(RPC_ERROR_CODES.ACCOUNT_NOT_FOUND, 'Account not found');
  }

  // TODO: Get activity metrics from indexer
  return {
    account: {
      balance: account.balance.toString(),
      stakedBalance: account.stakedBalance.toString(),
      reputation: account.reputationScore,
    },
    activity: {
      claimsSubmitted: 0,  // Would come from indexer
      claimsAccepted: 0,
      verificationsCompleted: 0,
      verificationsCorrect: 0,
    },
    earnings: {
      totalEarned: '0',
      pendingRewards: '0',
    },
    sessions: {
      active: account.sessionKeyCount,
    },
  };
};

// ============ Knowledge Handlers ============

export const knowledgeGetFacts: Handler<KnowledgeGetFactsRequest, KnowledgeGetFactsResponse> = async (params, ctx) => {
  let facts = [];

  if (params.domain) {
    facts = ctx.state.getFactsByDomain(params.domain);
  } else if (params.search) {
    facts = ctx.state.searchFacts(params.search);
  } else {
    // Get all active facts (paginated)
    facts = ctx.state.getFactsByDomain(''); // Would need a getAllFacts method
  }

  // Filter by confidence
  if (params.minConfidence !== undefined) {
    facts = facts.filter(f => f.confidence >= params.minConfidence!);
  }

  // Filter by status
  if (params.status && params.status !== 'all') {
    facts = facts.filter(f => f.status === params.status);
  }

  // Pagination
  const limit = params.limit || 50;
  const startIndex = params.cursor ? parseInt(params.cursor) : 0;
  const paginatedFacts = facts.slice(startIndex, startIndex + limit);

  return {
    facts: paginatedFacts,
    nextCursor: startIndex + limit < facts.length ? String(startIndex + limit) : undefined,
    total: facts.length,
  };
};

export const knowledgeGetFact: Handler<KnowledgeGetFactRequest, KnowledgeGetFactResponse> = async (params, ctx) => {
  const fact = ctx.state.getFact(params.factId);
  const blockNumber = ctx.state.getBlockNumber();

  const response: KnowledgeGetFactResponse = {
    fact,
    blockNumber: blockNumber.toString(),
  };

  // Include references if requested
  if (params.includeReferences && fact) {
    // Would need to get references from knowledge tree
    response.references = {
      supports: [],
      contradicts: [],
      extends: [],
    };
  }

  return response;
};

export const knowledgeSubmitClaim: Handler<KnowledgeSubmitClaimRequest, KnowledgeSubmitClaimResponse> = async (params, ctx) => {
  // Verify signature
  const payload = JSON.stringify({
    claim: params.claim,
    domain: params.domain,
    stake: params.stake,
    references: params.references,
    nonce: params.nonce,
  });

  const valid = verify(
    new TextEncoder().encode(payload),
    params.signature,
    params.sender
  );

  if (!valid) {
    throw createRPCError(RPC_ERROR_CODES.INVALID_SIGNATURE, 'Invalid signature');
  }

  // Check account balance for stake
  const senderDid = `did:lgm:${params.sender.slice(0, 32)}`;
  const account = ctx.state.getAccount(senderDid);

  if (!account || account.balance < BigInt(params.stake)) {
    throw createRPCError(RPC_ERROR_CODES.INSUFFICIENT_BALANCE, 'Insufficient balance for stake');
  }

  // Submit to consensus engine
  const claimId = await ctx.consensus.submitClaim({
    id: '',  // Generated by engine
    claim: params.claim,
    domain: params.domain,
    submitter: params.sender,
    stake: BigInt(params.stake),
    submittedAt: Date.now(),
    references: params.references || [],
  });

  // Estimate completion
  const currentBlock = Number(ctx.state.getBlockNumber());
  const roundBlocks = 10; // From consensus params

  return {
    claimId,
    verificationRoundId: claimId,
    estimatedCompletionBlock: currentBlock + roundBlocks,
    txHash: claimId, // Simplified
  };
};

// ============ Verification Handlers ============

export const verificationGetPending: Handler<VerificationGetPendingRequest, VerificationGetPendingResponse> = async (params, ctx) => {
  // Get active rounds where this validator is assigned
  const rounds = ctx.consensus.getActiveRounds();
  const assignments: VerificationGetPendingResponse['assignments'] = [];

  for (const round of rounds) {
    const isAssigned = round.selectedValidators.some(
      v => v.validator === params.validatorKey && v.selected
    );

    if (isAssigned) {
      assignments.push({
        roundId: round.id,
        claimId: round.claimId,
        claim: round.claim.claim,
        domain: round.claim.domain,
        commitDeadline: round.commitDeadline,
        revealDeadline: round.revealDeadline,
      });
    }
  }

  return { assignments };
};

// ============ Network Handlers ============

export const networkHealth: Handler<{}, NetworkHealthResponse> = async (_, ctx) => {
  const blockNumber = ctx.state.getBlockNumber();

  return {
    status: 'healthy',
    latestBlock: blockNumber.toString(),
    syncStatus: 'synced',
    peerCount: 0, // Would get from network layer
    version: ctx.nodeVersion,
  };
};

// ============ Handler Registry ============

export const handlers: Record<string, Handler<any, any>> = {
  // Account
  'account_getBalance': accountGetBalance,
  'account_getInfo': accountGetInfo,

  // Agent
  'agent_checkAccount': agentCheckAccount,
  'agent_getStatus': agentGetStatus,

  // Knowledge
  'knowledge_getFacts': knowledgeGetFacts,
  'knowledge_getFact': knowledgeGetFact,
  'knowledge_submitClaim': knowledgeSubmitClaim,

  // Verification
  'verification_getPending': verificationGetPending,

  // Network
  'network_health': networkHealth,
};

// ============ Helpers ============

function createRPCError(code: number, message: string, details?: unknown): RPCError {
  return {
    code,
    message,
    data: {
      errorCode: getErrorCode(code),
      details,
      retryable: isRetryable(code),
    },
  };
}

function getErrorCode(code: number): string {
  const codeMap: Record<number, string> = {
    [-32001]: 'ACCOUNT_NOT_FOUND',
    [-32002]: 'INSUFFICIENT_BALANCE',
    [-32003]: 'INVALID_SIGNATURE',
    [-32004]: 'INVALID_NONCE',
    [-32005]: 'SESSION_EXPIRED',
    [-32006]: 'SESSION_LIMIT_EXCEEDED',
    [-32007]: 'RATE_LIMITED',
  };
  return codeMap[code] || 'UNKNOWN';
}

function isRetryable(code: number): boolean {
  return code === -32007; // Only rate limiting is retryable
}
