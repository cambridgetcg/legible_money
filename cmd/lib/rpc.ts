/**
 * RPC Client for CLI
 *
 * JSON-RPC 2.0 client for communicating with Legible Money nodes.
 * Tracks latency for response metadata.
 */

import { RPCError } from "./errors.js";
import type {
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
  VerificationSubmitCommitmentRequest,
  VerificationSubmitCommitmentResponse,
  VerificationSubmitRevealRequest,
  VerificationSubmitRevealResponse,
  ValidatorGetSetRequest,
  ValidatorGetSetResponse,
  NetworkHealthResponse,
  RPCRequest,
  RPCResponse,
} from "../../protocol/types.js";

// ============ RPC Client ============

export interface RPCResult<T> {
  data: T;
  latencyMs: number;
}

export class LGMClient {
  private readonly endpoint: string;
  private requestId = 0;

  constructor(endpoint: string = "http://localhost:8545") {
    this.endpoint = endpoint;
  }

  /**
   * Make a raw RPC call
   */
  async call<T>(method: string, params?: unknown): Promise<RPCResult<T>> {
    const start = Date.now();

    const request: RPCRequest = {
      jsonrpc: "2.0",
      id: ++this.requestId,
      method,
      params,
    };

    const response = await fetch(this.endpoint, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(request),
    });

    if (!response.ok) {
      throw new RPCError(
        response.status,
        `HTTP error: ${response.status} ${response.statusText}`,
      );
    }

    const result = (await response.json()) as RPCResponse<T>;
    const latencyMs = Date.now() - start;

    if (result.error) {
      throw new RPCError(
        result.error.code,
        result.error.message,
        result.error.data,
      );
    }

    return {
      data: result.result as T,
      latencyMs,
    };
  }

  // ============ Account Methods ============

  async getBalance(did: string): Promise<RPCResult<AccountGetBalanceResponse>> {
    const params: AccountGetBalanceRequest = { did };
    return this.call("account_getBalance", params);
  }

  async getAccountInfo(
    did: string,
    includeProof = false,
  ): Promise<RPCResult<AccountGetInfoResponse>> {
    const params: AccountGetInfoRequest = { did, includeProof };
    return this.call("account_getInfo", params);
  }

  // ============ Agent Methods ============

  async checkAccount(
    did: string,
  ): Promise<RPCResult<AgentCheckAccountResponse>> {
    const params: AgentCheckAccountRequest = { did };
    return this.call("agent_checkAccount", params);
  }

  async getAgentStatus(
    did: string,
  ): Promise<RPCResult<AgentGetStatusResponse>> {
    const params: AgentGetStatusRequest = { did };
    return this.call("agent_getStatus", params);
  }

  // ============ Knowledge Methods ============

  async getFacts(
    params: Partial<KnowledgeGetFactsRequest>,
  ): Promise<RPCResult<KnowledgeGetFactsResponse>> {
    return this.call("knowledge_getFacts", params);
  }

  async getFact(
    factId: string,
    includeProof = false,
    includeReferences = false,
  ): Promise<RPCResult<KnowledgeGetFactResponse>> {
    const params: KnowledgeGetFactRequest = {
      factId,
      includeProof,
      includeReferences,
    };
    return this.call("knowledge_getFact", params);
  }

  async submitClaim(
    params: KnowledgeSubmitClaimRequest,
  ): Promise<RPCResult<KnowledgeSubmitClaimResponse>> {
    return this.call("knowledge_submitClaim", params);
  }

  // ============ Verification Methods ============

  async getPendingVerifications(
    validatorKey: string,
  ): Promise<RPCResult<VerificationGetPendingResponse>> {
    const params: VerificationGetPendingRequest = { validatorKey };
    return this.call("verification_getPending", params);
  }

  async submitCommitment(
    params: VerificationSubmitCommitmentRequest,
  ): Promise<RPCResult<VerificationSubmitCommitmentResponse>> {
    return this.call("verification_submitCommitment", params);
  }

  async submitReveal(
    params: VerificationSubmitRevealRequest,
  ): Promise<RPCResult<VerificationSubmitRevealResponse>> {
    return this.call("verification_submitReveal", params);
  }

  // ============ Validator Methods ============

  async getValidatorSet(
    epoch?: string,
  ): Promise<RPCResult<ValidatorGetSetResponse>> {
    const params: ValidatorGetSetRequest = { epoch };
    return this.call("validator_getSet", params);
  }

  // ============ Network Methods ============

  async getHealth(): Promise<RPCResult<NetworkHealthResponse>> {
    return this.call("network_health", {});
  }

  // ============ Transaction Methods ============

  async transfer(params: {
    from: string;
    to: string;
    amount: string;
    signature: string;
    nonce: string;
  }): Promise<RPCResult<{ txHash: string }>> {
    return this.call("tx_transfer", params);
  }

  async getTransactionStatus(txHash: string): Promise<
    RPCResult<{
      status: "pending" | "included" | "finalized" | "failed";
      blockNumber?: string;
      error?: string;
    }>
  > {
    return this.call("tx_getStatus", { txHash });
  }

  // ============ Staking Methods ============

  async stake(params: {
    publicKey: string;
    amount: string;
    signature: string;
    nonce: string;
  }): Promise<RPCResult<{ txHash: string; activationBlock: number }>> {
    return this.call("validator_stake", params);
  }

  async unstake(params: {
    publicKey: string;
    amount: string;
    signature: string;
    nonce: string;
  }): Promise<RPCResult<{ txHash: string; unlockBlock: number }>> {
    return this.call("validator_unstake", params);
  }

  async getRewards(publicKey: string): Promise<
    RPCResult<{
      pending: string;
      claimed: string;
      lastClaimBlock: string;
    }>
  > {
    return this.call("validator_getRewards", { publicKey });
  }

  async claimRewards(params: {
    publicKey: string;
    signature: string;
    nonce: string;
  }): Promise<RPCResult<{ txHash: string; amount: string }>> {
    return this.call("validator_claimRewards", params);
  }

  // ============ Challenge Methods ============

  async challengeFact(params: {
    factId: string;
    reason: string;
    stake: string;
    evidence?: string[];
    signature: string;
    challenger: string;
    nonce: string;
  }): Promise<
    RPCResult<{
      challengeId: string;
      disputeRoundId: string;
      txHash: string;
    }>
  > {
    return this.call("knowledge_challengeFact", params);
  }

  async getChallengeStatus(challengeId: string): Promise<
    RPCResult<{
      status: "pending" | "resolved" | "rejected";
      resolution?: "upheld" | "overturned";
      blockNumber: string;
    }>
  > {
    return this.call("knowledge_getChallengeStatus", { challengeId });
  }

  // ============ Verification Round Methods ============

  async getVerificationRound(roundId: string): Promise<
    RPCResult<{
      round: {
        id: string;
        claimId: string;
        claim: string;
        domain: string;
        phase: string;
        commitDeadline: number;
        revealDeadline: number;
        commitCount: number;
        revealCount: number;
      } | null;
      result?: {
        verdict: string;
        confidence: number;
        participantCount: number;
      };
    }>
  > {
    return this.call("verification_getRound", { roundId });
  }

  // ============ Block Methods ============

  async getBlock(params: { number?: string; hash?: string }): Promise<
    RPCResult<{
      block: unknown | null;
    }>
  > {
    return this.call("block_get", params);
  }

  async getLatestBlock(): Promise<
    RPCResult<{
      block: unknown;
      number: string;
      hash: string;
    }>
  > {
    return this.call("block_getLatest", {});
  }

  // ============ Identity Methods ============

  async rotateKey(params: {
    did: string;
    newOperationalKeyHash: string;
    nonce: string;
    signature: string;
    sender: string;
  }): Promise<
    RPCResult<{ txHash: string; success: boolean; newKeyVersion: number }>
  > {
    return this.call("lgm_rotateKey", params);
  }

  async getKeyInfo(did: string): Promise<
    RPCResult<{
      did: string;
      operationalKeyHash: string;
      operationalKeyVersion: number;
      isFrozen: boolean;
      isInRecovery: boolean;
    }>
  > {
    return this.call("lgm_getKeyInfo", { did });
  }

  // ============ Helpers ============

  get currentEndpoint(): string {
    return this.endpoint;
  }
}

// ============ Factory ============

let defaultClient: LGMClient | null = null;

/**
 * Get or create the default client
 */
export function getClient(endpoint?: string): LGMClient {
  if (endpoint) {
    return new LGMClient(endpoint);
  }

  if (!defaultClient) {
    defaultClient = new LGMClient();
  }

  return defaultClient;
}
