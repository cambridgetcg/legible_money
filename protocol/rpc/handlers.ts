/**
 * RPC Handlers
 *
 * Handler implementations for all RPC methods.
 * Each handler takes params and returns result or throws.
 */

import type { StateManager } from "../../state/state_manager.js";
import type { ProofOfTruthEngine } from "../../consensus/pot/engine.js";
import type { StateIntegration } from "../../consensus/pot/state_integration.js";
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
  KnowledgeSessionStartRequest,
  KnowledgeSessionStartResponse,
  KnowledgeSessionEndRequest,
  KnowledgeSessionEndResponse,
  KnowledgeQueryRequest,
  KnowledgeQueryResponse,
  KnowledgeEstimateQueryCostRequest,
  KnowledgeEstimateQueryCostResponse,
  VerificationGetPendingRequest,
  VerificationGetPendingResponse,
  VerificationSubmitCommitmentRequest,
  VerificationSubmitCommitmentResponse,
  VerificationSubmitRevealRequest,
  VerificationSubmitRevealResponse,
  VerificationGetRoundRequest,
  VerificationGetRoundResponse,
  NetworkHealthResponse,
} from "../types.js";
import { RPC_ERROR_CODES } from "../types.js";
import { verify } from "../../core/crypto/signatures.js";
import type { PublicKey } from "../../core/crypto/keys.js";
import type { TruthGateway } from "../../core/billing/truth_gateway.js";
import {
  calculateQueryPrice,
  calculateDistribution,
  type FactResult as GatewayFactResult,
} from "../../core/billing/truth_gateway.js";
import type { LoopController } from "../../core/genesis/perpetual_loop.js";
import type { LoopEventBridge } from "../../core/genesis/loop_event_bridge.js";
import type { DID } from "../../core/agents/identity/did.js";
import type { FactState } from "../../state/types.js";
import {
  executePaymentDistribution,
  canAffordDistribution,
} from "../../core/billing/payment_executor.js";
import {
  validateLimit,
  validateCursor,
  validateDomain,
  validateSearch,
  validateConfidence,
} from "./validation.js";

// B3 FIX: Import new handlers
import { b3Handlers } from "./handlers/index.js";

// Gödelian logic layer
import { META_AXIOMS } from "../../core/logic/types.js";
import { GOEDELIAN_PROMISE } from "../../core/logic/goedel.js";
import { ZONE_PROPERTIES } from "../../core/logic/zones.js";

// ============ Handler Context ============

export interface HandlerContext {
  state: StateManager;
  consensus: ProofOfTruthEngine;
  stateIntegration: StateIntegration;
  nodeVersion: string;
  networkId: string;
  /** Network layer for peer info (optional for backward compat) */
  network?: { getPeerCount(): number };
  // Economic Sustainability Integration (optional for backward compat)
  truthGateway?: TruthGateway;
  loopController?: LoopController;
  loopEventBridge?: LoopEventBridge;
  providerDid?: DID;
}

// ============ Handler Type ============

export type Handler<TParams, TResult> = (
  params: TParams,
  ctx: HandlerContext,
) => Promise<TResult>;

// ============ Account Handlers ============

export const accountGetBalance: Handler<
  AccountGetBalanceRequest,
  AccountGetBalanceResponse
> = async (params, ctx) => {
  const account = ctx.state.getAccount(params.did);

  if (!account) {
    return {
      balance: "0",
      stakedBalance: "0",
      nonce: "0",
    };
  }

  return {
    balance: account.balance.toString(),
    stakedBalance: account.stakedBalance.toString(),
    nonce: account.nonce.toString(),
  };
};

export const accountGetInfo: Handler<
  AccountGetInfoRequest,
  AccountGetInfoResponse
> = async (params, ctx) => {
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

export const agentCheckAccount: Handler<
  AgentCheckAccountRequest,
  AgentCheckAccountResponse
> = async (params, ctx) => {
  const account = ctx.state.getAccount(params.did);

  if (!account) {
    return { exists: false };
  }

  const validator = ctx.state.getValidator(
    account.operationalKeyHash as PublicKey,
  );

  return {
    exists: true,
    balance: account.balance.toString(),
    reputation: account.reputationScore,
    isValidator: validator?.isActive ?? false,
  };
};

export const agentGetStatus: Handler<
  AgentGetStatusRequest,
  AgentGetStatusResponse
> = async (params, ctx) => {
  const account = ctx.state.getAccount(params.did);

  if (!account) {
    throw createRPCError(
      RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
      "Account not found",
    );
  }

  // TODO: Get activity metrics from indexer
  return {
    account: {
      balance: account.balance.toString(),
      stakedBalance: account.stakedBalance.toString(),
      reputation: account.reputationScore,
    },
    activity: {
      claimsSubmitted: 0, // Would come from indexer
      claimsAccepted: 0,
      verificationsCompleted: 0,
      verificationsCorrect: 0,
    },
    earnings: {
      totalEarned: "0",
      pendingRewards: "0",
    },
    sessions: {
      active: account.sessionKeyCount,
    },
  };
};

// ============ Knowledge Handlers ============

export const knowledgeGetFacts: Handler<
  KnowledgeGetFactsRequest,
  KnowledgeGetFactsResponse
> = async (params, ctx) => {
  // Input validation
  const domain = validateDomain(params.domain);
  const search = validateSearch(params.search);
  const limit = validateLimit(params.limit);
  const startIndex = validateCursor(params.cursor);

  let facts = [];

  if (domain) {
    facts = ctx.state.getFactsByDomain(domain);
  } else if (search) {
    facts = ctx.state.searchFacts(search);
  } else {
    // Get all active facts (paginated)
    facts = ctx.state.getFactsByDomain(""); // Would need a getAllFacts method
  }

  // Filter by confidence
  if (params.minConfidence !== undefined) {
    const minConf = validateConfidence(params.minConfidence, "minConfidence");
    facts = facts.filter((f) => f.confidence >= minConf);
  }

  // Filter by status
  if (params.status && params.status !== "all") {
    facts = facts.filter((f) => f.status === params.status);
  }

  // Pagination (limit already validated and capped)
  const paginatedFacts = facts.slice(startIndex, startIndex + limit);

  return {
    facts: paginatedFacts,
    nextCursor:
      startIndex + limit < facts.length
        ? String(startIndex + limit)
        : undefined,
    total: facts.length,
  };
};

export const knowledgeGetFact: Handler<
  KnowledgeGetFactRequest,
  KnowledgeGetFactResponse
> = async (params, ctx) => {
  const fact = ctx.state.getFact(params.factId);
  const blockNumber = ctx.state.getBlockNumber();

  const response: KnowledgeGetFactResponse = {
    fact,
    blockNumber: blockNumber.toString(),
  };

  // Include references if requested
  if (params.includeReferences && fact) {
    response.references = ctx.state.getFactReferences(params.factId);
  }

  return response;
};

export const knowledgeSubmitClaim: Handler<
  KnowledgeSubmitClaimRequest,
  KnowledgeSubmitClaimResponse
> = async (params, ctx) => {
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
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  // Check account balance for stake
  // Use provided DID or derive from public key
  const senderDid = params.senderDid || `did:lgm:${params.sender.slice(0, 32)}`;
  const account = ctx.state.getAccount(senderDid);

  if (!account || account.balance < BigInt(params.stake)) {
    throw createRPCError(
      RPC_ERROR_CODES.INSUFFICIENT_BALANCE,
      "Insufficient balance for stake",
    );
  }

  // Submit to consensus engine
  const claimId = await ctx.consensus.submitClaim(
    {
      claim: params.claim,
      domain: params.domain,
      type: "empirical" as any,
      stake: BigInt(params.stake),
      proposedConfidence: 0.5,
      references: (params.references || []) as any,
    } as any,
    params.sender,
  );
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

// ============ Knowledge Session Handlers ============
// Note: These require UnifiedQueryService to be wired into HandlerContext

export const knowledgeSessionStart: Handler<
  KnowledgeSessionStartRequest,
  KnowledgeSessionStartResponse
> = async (params, ctx) => {
  // This handler requires UnifiedQueryService to be added to HandlerContext
  // For now, return a placeholder that indicates the session was created
  const sessionId = `chat:${params.agentDid}:${Date.now()}`;
  const blockNumber = ctx.state.getBlockNumber();

  return {
    sessionId,
    startBlock: blockNumber.toString(),
    freeTierQueries: 10, // From DEFAULT_QUERY_COST_CONFIG.freeTierQueriesPerSession
  };
};

export const knowledgeSessionEnd: Handler<
  KnowledgeSessionEndRequest,
  KnowledgeSessionEndResponse
> = async (_params, _ctx) => {
  // This handler requires UnifiedQueryService to be added to HandlerContext
  // For now, return a placeholder response
  return {
    success: true,
    totalQueries: 0,
    totalCost: "0",
  };
};

export const knowledgeQuery: Handler<
  KnowledgeQueryRequest,
  KnowledgeQueryResponse
> = async (params, ctx) => {
  const startTime = Date.now();
  const blockNumber = ctx.state.getBlockNumber();

  // Input validation
  const queryLimit = validateLimit(params.intent.limit);
  const queryDomain = params.intent.domain
    ? validateDomain(params.intent.domain)
    : undefined;
  const minConf =
    params.intent.minConfidence !== undefined
      ? validateConfidence(params.intent.minConfidence, "minConfidence")
      : undefined;

  // Step 1: Query facts from state (existing logic)
  const factStates: FactState[] = [];

  if (queryDomain) {
    const domainFacts = ctx.state.getFactsByDomain(queryDomain);
    for (const fact of domainFacts.slice(0, queryLimit)) {
      if (minConf !== undefined && fact.confidence < minConf) {
        continue;
      }
      factStates.push(fact);
    }
  } else if (params.intent.factId) {
    const fact = ctx.state.getFact(params.intent.factId);
    if (fact) {
      factStates.push(fact);
    }
  }

  // Step 2: Build response facts
  const facts = factStates.map((fact) => ({
    factId: fact.id || "",
    fact,
    domain: fact.domain || "",
    stratum: fact.stratum || "empirical",
    relevance: fact.confidence,
    cached: false,
  }));

  // Step 3: Calculate billing if TruthGateway is available
  let estimatedCost = "0";
  let actualCost = "0";
  let paymentExecuted = false;
  let researchShare = "0";
  let providerShare = "0";
  let transferCount = 0;

  if (ctx.truthGateway && factStates.length > 0) {
    // Map FactState to TruthGateway FactResult for pricing
    const gatewayFacts: GatewayFactResult[] = factStates.map((fact) =>
      factStateToGatewayFactResult(
        fact,
        ctx.providerDid || ("did:lgm:node" as DID),
      ),
    );

    // Calculate cost
    const pricing = calculateQueryPrice(gatewayFacts, blockNumber);
    estimatedCost = pricing.totalCost.toString();

    // Calculate distribution breakdown
    const providerDid = ctx.providerDid || ("did:lgm:node" as DID);
    const distribution = calculateDistribution(
      pricing.totalCost,
      gatewayFacts,
      providerDid,
    );

    // Step 4: Execute payment if caller can afford it
    // Extract caller DID from session ID (format: "chat:${agentDid}:${timestamp}")
    const callerDid = extractCallerDid(params.sessionId);

    if (callerDid && distribution) {
      const callerBalance = ctx.state.getBalance(callerDid);

      if (canAffordDistribution(callerBalance, distribution)) {
        const result = executePaymentDistribution(
          distribution,
          callerDid as DID,
          ctx.state,
          blockNumber,
        );
        paymentExecuted = result.success;
        transferCount = result.transfers.length;
        actualCost = result.totalTransferred.toString();

        // Record citation rewards for each fact
        if (result.success) {
          for (const reward of distribution.knowledgeRewards) {
            if (reward.factId && reward.amount > 0n) {
              try {
                ctx.state.recordFactCitationReward(
                  reward.factId,
                  reward.amount,
                );
              } catch {
                // Citation reward tracking is non-critical
              }
            }
          }

          // Emit loop event for sustainability tracking
          if (ctx.loopEventBridge) {
            try {
              const factIds = factStates
                .map((f) => f.id)
                .filter(Boolean) as string[];
              ctx.loopEventBridge.onQueryExecuted(
                `query:${params.sessionId}:${Date.now()}`,
                distribution,
                result,
                factIds,
              );
            } catch {
              // Loop event tracking is non-critical
            }
          }
        }
      } else {
        // Graceful degradation: facts are returned but payment not executed
        actualCost = "0";
      }
    }

    researchShare = distribution.research?.amount?.toString() || "0";
    providerShare = distribution.apiProvider?.amount?.toString() || "0";

    // Step 5: Increment citation counts for each fact queried (regardless of payment)
    for (const fact of factStates) {
      if (fact.id && fact.status === "active") {
        try {
          ctx.state.incrementFactCitationCount(fact.id, blockNumber);
        } catch {
          // Citation tracking is non-critical; don't fail the query
        }
      }
    }
  }

  const latencyMs = Date.now() - startTime;

  return {
    intentType: params.intent.type,
    facts,
    domains: [],
    metadata: {
      cached: false,
      latencyMs,
      estimatedCost,
      actualCost,
      cacheHitRatio: 0,
      paymentExecuted,
      researchShare,
      providerShare,
      transferCount,
    },
  };
};

/**
 * Extract caller DID from session ID.
 * Session format: "chat:${agentDid}:${timestamp}"
 */
function extractCallerDid(sessionId: string): string | null {
  if (!sessionId.startsWith("chat:")) return null;
  const parts = sessionId.split(":");
  // Reconstruct DID from parts between first 'chat' and last timestamp
  // DID format: "did:lgm:name" or "did:lgm-devnet:name"
  if (parts.length < 3) return null;
  // Remove 'chat' prefix and timestamp suffix, rejoin the DID parts
  const didParts = parts.slice(1, -1);
  return didParts.join(":");
}

/**
 * Convert on-chain FactState to billing FactResult for pricing calculations.
 */
function factStateToGatewayFactResult(
  fact: FactState,
  defaultSubmitter: DID,
): GatewayFactResult {
  return {
    id: fact.id,
    claim: fact.claim,
    domain: fact.domain,
    confidence: fact.confidence,
    fundamentalityScore: 0, // Will be calculated from citationCount by pricing engine
    submitter: defaultSubmitter, // Would ideally come from ClaimState lookup
    verifiedAtBlock: fact.verifiedAt,
    citationCount: fact.citationCount,
    lastCitedBlock: fact.lastCitedBlock,
  };
}

export const knowledgeEstimateQueryCost: Handler<
  KnowledgeEstimateQueryCostRequest,
  KnowledgeEstimateQueryCostResponse
> = async (params, _ctx) => {
  // Simplified cost estimation without full index
  const buckets = Math.min(100, params.intent.limit ?? 50);
  const baseCost = 100n;
  const bucketCost = BigInt(buckets) * 10n;
  const crossStratumCost = params.intent.crossStrata ? baseCost : 0n;

  const totalCost = baseCost + bucketCost + crossStratumCost;

  return {
    cost: totalCost.toString(),
    cached: false,
    estimatedBuckets: buckets,
    breakdown: {
      baseCost: baseCost.toString(),
      bucketCost: bucketCost.toString(),
      crossStratumCost: crossStratumCost.toString(),
      materializationCost: "0",
    },
  };
};

// ============ Verification Handlers ============

export const verificationGetPending: Handler<
  VerificationGetPendingRequest,
  VerificationGetPendingResponse
> = async (params, ctx) => {
  // Get active rounds where this validator is assigned
  const rounds = ctx.consensus.getActiveRounds();
  const assignments: VerificationGetPendingResponse["assignments"] = [];

  for (const round of rounds) {
    const isAssigned = round.selectedValidators.some(
      (v) => v.validator === params.validatorKey && v.selected,
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

export const verificationSubmitCommitment: Handler<
  VerificationSubmitCommitmentRequest,
  VerificationSubmitCommitmentResponse
> = async (params, ctx) => {
  try {
    // Use deterministic mode when VRF params not provided (DevNet/testing)
    // Production clients should include vrfOutput + vrfProof
    if ((params as any).vrfOutput && (params as any).vrfProof) {
      await ctx.consensus.submitCommitment(params.roundId, {
        validator: params.validatorKey,
        commitment: params.commitment,
        signature: params.signature,
        vrfOutput: (params as any).vrfOutput,
        vrfProof: (params as any).vrfProof,
      });
    } else {
      await ctx.consensus.submitCommitmentDeterministic(params.roundId, {
        validator: params.validatorKey,
        commitment: params.commitment,
        signature: params.signature,
      });
    }

    // Get the round to return deadline
    const rounds = ctx.consensus.getActiveRounds();
    const round = rounds.find((r) => r.id === params.roundId);

    return {
      accepted: true,
      commitDeadline: round?.commitDeadline ?? 0,
    };
  } catch (error) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_PARAMS,
      error instanceof Error ? error.message : "Failed to submit commitment",
    );
  }
};

export const verificationSubmitReveal: Handler<
  VerificationSubmitRevealRequest,
  VerificationSubmitRevealResponse
> = async (params, ctx) => {
  // Validate confidence is within bounds
  const confidence = validateConfidence(params.confidence, "confidence");

  try {
    await ctx.consensus.submitReveal(params.roundId, {
      validator: params.validatorKey,
      verdict: params.verdict,
      confidence,
      nonce: params.nonce,
      referencesChecked: [],
      signature: params.signature,
    });

    // Get round status
    const rounds = ctx.consensus.getActiveRounds();
    const round = rounds.find((r) => r.id === params.roundId);

    return {
      accepted: true,
      roundStatus: round?.phase ?? "unknown",
    };
  } catch (error) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_PARAMS,
      error instanceof Error ? error.message : "Failed to submit reveal",
    );
  }
};

export const verificationGetRound: Handler<
  VerificationGetRoundRequest,
  VerificationGetRoundResponse
> = async (params, ctx) => {
  const rounds = ctx.consensus.getActiveRounds();
  const round = rounds.find((r) => r.id === params.roundId);

  if (!round) {
    return { round: null };
  }

  return {
    round,
    result: round.result,
  };
};

// ============ Network Handlers ============

export const networkHealth: Handler<
  Record<string, never>,
  NetworkHealthResponse
> = async (_, ctx) => {
  const blockNumber = ctx.state.getBlockNumber();

  return {
    status: "healthy",
    latestBlock: blockNumber.toString(),
    syncStatus: "synced",
    peerCount: ctx.network?.getPeerCount() ?? 0,
    version: ctx.nodeVersion,
  };
};

// ============ Logic Layer Handlers ============

/**
 * Get the protocol's Gödelian meta-axioms.
 * These are the foundational acknowledgments of system limitations.
 */
export const logicGetMetaAxioms: Handler<
  Record<string, never>,
  { axioms: typeof META_AXIOMS; goedelianPromise: string }
> = async () => {
  return {
    axioms: META_AXIOMS,
    goedelianPromise: GOEDELIAN_PROMISE,
  };
};

/**
 * Get logic zone properties.
 * Returns decidability, completeness, and Gödel-applicability for each zone.
 */
export const logicGetZones: Handler<
  Record<string, never>,
  { zones: typeof ZONE_PROPERTIES }
> = async () => {
  return { zones: ZONE_PROPERTIES };
};

// ============ Handler Registry ============

export const handlers: Record<string, Handler<any, any>> = {
  // Account
  account_getBalance: accountGetBalance,
  account_getInfo: accountGetInfo,

  // Agent
  agent_checkAccount: agentCheckAccount,
  agent_getStatus: agentGetStatus,

  // Knowledge
  knowledge_getFacts: knowledgeGetFacts,
  knowledge_getFact: knowledgeGetFact,
  knowledge_submitClaim: knowledgeSubmitClaim,
  knowledge_sessionStart: knowledgeSessionStart,
  knowledge_sessionEnd: knowledgeSessionEnd,
  knowledge_query: knowledgeQuery,
  knowledge_estimateQueryCost: knowledgeEstimateQueryCost,

  // Verification
  verification_getPending: verificationGetPending,
  verification_submitCommitment: verificationSubmitCommitment,
  verification_submitReveal: verificationSubmitReveal,
  verification_getRound: verificationGetRound,

  // Logic (Gödelian)
  logic_getMetaAxioms: logicGetMetaAxioms,
  logic_getZones: logicGetZones,

  // Network
  network_health: networkHealth,

  // B3 FIX: Staking, Transactions, Disputes, Channels
  ...b3Handlers,
};

// ============ Helpers ============

function createRPCError(
  code: number,
  message: string,
  details?: unknown,
): RPCError {
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
    [-32001]: "ACCOUNT_NOT_FOUND",
    [-32002]: "INSUFFICIENT_BALANCE",
    [-32003]: "INVALID_SIGNATURE",
    [-32004]: "INVALID_NONCE",
    [-32005]: "SESSION_EXPIRED",
    [-32006]: "SESSION_LIMIT_EXCEEDED",
    [-32007]: "RATE_LIMITED",
  };
  return codeMap[code] || "UNKNOWN";
}

function isRetryable(code: number): boolean {
  return code === -32007; // Only rate limiting is retryable
}
