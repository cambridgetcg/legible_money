/**
 * Legible Money SDK Client
 *
 * Agent-friendly client for interacting with Legible Money.
 * Handles key derivation, signing, and RPC communication.
 */

import { EventEmitter } from "events";
import {
  AgentWallet,
  SessionWallet,
  type SessionCapabilities,
} from "./wallet.js";
import type {
  ClientConfig,
  RPCRequest,
  RPCResponse,
  AccountGetBalanceResponse,
  AccountGetInfoResponse,
  AgentGetStatusResponse,
  KnowledgeSubmitClaimResponse,
  KnowledgeGetFactsResponse,
  KnowledgeGetFactResponse,
  TxSubmitResponse,
  TxTransferResponse,
  VerificationGetPendingResponse,
  VerificationSubmitCommitmentResponse,
  VerificationSubmitRevealResponse,
  NetworkHealthResponse,
  ClaimOptions,
  TransferOptions,
  ProtocolEvent,
  Subscription,
  RateLimitInfo,
} from "../types.js";
import type { FactState } from "../../state/types.js";
import type { Block } from "../../consensus/pot/types.js";
import { commit } from "../../core/crypto/commitment.js";

// ============ Client ============

/**
 * Main SDK client for Legible Money
 */
export class LegibleClient extends EventEmitter {
  private readonly config: Required<ClientConfig>;
  private wallet: AgentWallet | null = null;
  private session: SessionWallet | null = null;
  private nonce: bigint = 0n;
  private rateLimit: RateLimitInfo | null = null;
  private ws: WebSocket | null = null;
  private subscriptions: Map<string, (event: ProtocolEvent) => void> =
    new Map();

  constructor(config: ClientConfig) {
    super();

    this.config = {
      endpoint: config.endpoint,
      wsEndpoint: config.wsEndpoint || config.endpoint.replace("http", "ws"),
      agentId: config.agentId || "",
      secret: config.secret || "",
      privateKey: config.privateKey || "",
      sessionKey: config.sessionKey || "",
      timeout: config.timeout || 30000,
      retries: config.retries || 3,
    };

    // Initialize wallet if credentials provided
    if (config.agentId && config.secret) {
      this.wallet = AgentWallet.fromAgentId(
        config.agentId,
        config.secret,
        this.isTestnet() ? "testnet" : "mainnet",
      );
    } else if (config.privateKey) {
      this.wallet = AgentWallet.fromPrivateKey(config.privateKey);
    }
  }

  // ============ Account Methods ============

  /**
   * Get account balance
   */
  async getBalance(did?: string): Promise<bigint> {
    const targetDid = did || this.requireDid();
    const result = await this.call<AccountGetBalanceResponse>(
      "account_getBalance",
      { did: targetDid },
    );
    return BigInt(result.balance);
  }

  /**
   * Get full account info
   */
  async getAccount(
    did?: string,
    includeProof = false,
  ): Promise<AccountGetInfoResponse> {
    const targetDid = did || this.requireDid();
    return this.call<AccountGetInfoResponse>("account_getInfo", {
      did: targetDid,
      includeProof,
    });
  }

  /**
   * Get current nonce
   */
  async getNonce(did?: string): Promise<bigint> {
    const targetDid = did || this.requireDid();
    const result = await this.call<AccountGetBalanceResponse>(
      "account_getBalance",
      { did: targetDid },
    );
    this.nonce = BigInt(result.nonce);
    return this.nonce;
  }

  /**
   * Check if account exists
   */
  async accountExists(did?: string): Promise<boolean> {
    const targetDid = did || this.requireDid();
    const result = await this.call<{ exists: boolean }>("agent_checkAccount", {
      did: targetDid,
    });
    return result.exists;
  }

  /**
   * Ensure account exists, create if not
   */
  async ensureAccount(): Promise<string> {
    const wallet = this.requireWallet();

    const exists = await this.accountExists(wallet.did);
    if (exists) {
      // Sync nonce
      await this.getNonce();
      return wallet.did;
    }

    // Create account
    const signature = await wallet.signTransaction({
      action: "create_account",
      publicKey: wallet.publicKey,
    });
    await this.call("account_create", {
      operationalPublicKey: wallet.publicKey,
      signature,
    });

    return wallet.did;
  }

  /**
   * Get agent status (comprehensive)
   */
  async getAgentStatus(did?: string): Promise<AgentGetStatusResponse> {
    const targetDid = did || this.requireDid();
    return this.call<AgentGetStatusResponse>("agent_getStatus", {
      did: targetDid,
    });
  }

  // ============ Transfer Methods ============

  /**
   * Transfer tokens
   */
  async transfer(options: TransferOptions): Promise<string> {
    const wallet = this.requireWallet();
    await this.refreshNonce();

    const payload = {
      type: "transfer",
      from: wallet.did,
      to: options.to,
      amount: options.amount.toString(),
      nonce: this.nonce.toString(),
      memo: options.memo,
    };

    const signature = await wallet.signTransaction(payload);

    const result = await this.call<TxTransferResponse>("tx_transfer", {
      ...payload,
      signature,
    });

    this.nonce++;
    return result.txHash;
  }

  /**
   * Stake tokens
   */
  async stake(amount: bigint): Promise<string> {
    const wallet = this.requireWallet();
    await this.refreshNonce();

    const payload = {
      type: "stake",
      account: wallet.did,
      amount: amount.toString(),
      nonce: this.nonce.toString(),
    };

    const signature = await wallet.signTransaction(payload);
    const result = await this.call<TxSubmitResponse>("tx_submit", {
      transaction: {
        ...payload,
        signature,
        sender: wallet.publicKey,
        fee: "1000000000000000",
      },
    });

    this.nonce++;
    return result.txHash;
  }

  /**
   * Unstake tokens
   */
  async unstake(amount: bigint): Promise<string> {
    const wallet = this.requireWallet();
    await this.refreshNonce();

    const payload = {
      type: "unstake",
      account: wallet.did,
      amount: amount.toString(),
      nonce: this.nonce.toString(),
    };

    const signature = await wallet.signTransaction(payload);
    const result = await this.call<TxSubmitResponse>("tx_submit", {
      transaction: {
        ...payload,
        signature,
        sender: wallet.publicKey,
        fee: "1000000000000000",
      },
    });

    this.nonce++;
    return result.txHash;
  }

  // ============ Knowledge Methods ============

  /**
   * Submit a claim for verification
   */
  async submitClaim(
    options: ClaimOptions,
  ): Promise<KnowledgeSubmitClaimResponse> {
    const wallet = this.requireWallet();
    await this.refreshNonce();

    const payload = {
      claim: options.claim,
      domain: options.domain,
      stake: options.stake.toString(),
      references: options.references || [],
      nonce: this.nonce.toString(),
    };

    const signature = await wallet.signTransaction(payload);

    const result = await this.call<KnowledgeSubmitClaimResponse>(
      "knowledge_submitClaim",
      {
        ...payload,
        signature,
        sender: wallet.publicKey,
      },
    );

    this.nonce++;
    return result;
  }

  /**
   * Wait for claim verification to complete
   */
  async waitForVerification(
    claimId: string,
    timeoutMs = 60000,
  ): Promise<FactState | null> {
    const startTime = Date.now();

    while (Date.now() - startTime < timeoutMs) {
      const result = await this.call<KnowledgeGetFactResponse>(
        "knowledge_getFact",
        { factId: claimId },
      );

      if (result.fact) {
        return result.fact;
      }

      // Check if rejected
      const round = await this.call<{ result?: { outcome: string } }>(
        "verification_getRound",
        { roundId: claimId },
      );
      if (round.result?.outcome === "rejected") {
        return null;
      }

      // Wait and retry
      await sleep(2521); // One block time
    }

    throw new Error("Verification timeout");
  }

  /**
   * Search facts
   */
  async searchFacts(query: {
    domain?: string;
    search?: string;
    minConfidence?: number;
    limit?: number;
  }): Promise<FactState[]> {
    const result = await this.call<KnowledgeGetFactsResponse>(
      "knowledge_getFacts",
      query,
    );
    return result.facts;
  }

  /**
   * Get a specific fact
   */
  async getFact(
    factId: string,
    includeProof = false,
  ): Promise<FactState | null> {
    const result = await this.call<KnowledgeGetFactResponse>(
      "knowledge_getFact",
      { factId, includeProof },
    );
    return result.fact;
  }

  /**
   * Check if a claim contradicts existing knowledge
   */
  async checkContradictions(
    claim: string,
    domain: string,
  ): Promise<FactState[]> {
    const result = await this.call<{ contradictions: FactState[] }>(
      "knowledge_checkContradictions",
      { claim, domain },
    );
    return result.contradictions;
  }

  // ============ Validator Methods ============

  /**
   * Register as a validator
   */
  async registerValidator(stake: bigint): Promise<string> {
    const wallet = this.requireWallet();
    await this.refreshNonce();

    const payload = {
      type: "register_validator",
      publicKey: wallet.publicKey,
      stake: stake.toString(),
      nonce: this.nonce.toString(),
    };

    const signature = await wallet.signTransaction(payload);
    const result = await this.call<{ txHash: string }>("validator_register", {
      ...payload,
      signature,
    });

    this.nonce++;
    return result.txHash;
  }

  /**
   * Get pending verification assignments
   */
  async getPendingVerifications(): Promise<
    VerificationGetPendingResponse["assignments"]
  > {
    const wallet = this.requireWallet();
    const result = await this.call<VerificationGetPendingResponse>(
      "verification_getPending",
      {
        validatorKey: wallet.publicKey,
      },
    );
    return result.assignments;
  }

  /**
   * Submit verification commitment (phase 1)
   */
  async submitCommitment(
    roundId: string,
    verdict: "confirmed" | "rejected" | "uncertain",
    confidence: number,
  ): Promise<{ commitment: string; nonce: string }> {
    const wallet = this.requireWallet();

    // ZD-C01 FIX: Use core commit() which generates a cryptographically secure nonce
    // via randomBytes(32). Previously used Math.random() which is predictable.
    const message = `${verdict}:${confidence}`;
    const commitResult = commit(new TextEncoder().encode(message));
    const { commitment } = commitResult;
    const nonce = commitResult.nonce; // 32-byte cryptographic nonce from randomBytes()

    const payload = { roundId, commitment };
    const signature = await wallet.signTransaction(payload);

    await this.call<VerificationSubmitCommitmentResponse>(
      "verification_submitCommitment",
      {
        ...payload,
        validatorKey: wallet.publicKey,
        signature,
      },
    );

    return { commitment, nonce };
  }

  /**
   * Submit verification reveal (phase 2)
   */
  async submitReveal(
    roundId: string,
    verdict: "confirmed" | "rejected" | "uncertain",
    confidence: number,
    nonce: string,
    reasoning?: string,
  ): Promise<void> {
    const wallet = this.requireWallet();

    const payload = { roundId, verdict, confidence, nonce, reasoning };
    const signature = await wallet.signTransaction(payload);

    await this.call<VerificationSubmitRevealResponse>(
      "verification_submitReveal",
      {
        ...payload,
        validatorKey: wallet.publicKey,
        signature,
      },
    );
  }

  /**
   * Start automatic validator loop
   */
  async startValidator(
    verifyFn: (claim: { claim: string; domain: string }) => Promise<{
      verdict: "confirmed" | "rejected" | "uncertain";
      confidence: number;
      reasoning?: string;
    }>,
  ): Promise<void> {
    const commitments = new Map<
      string,
      { verdict: string; confidence: number; nonce: string }
    >();

    // Subscribe to assignments
    this.on("verification:assigned", async (event: any) => {
      try {
        // Get claim details
        const assignment = event;

        // Run verification logic
        const result = await verifyFn({
          claim: assignment.claim,
          domain: assignment.domain,
        });

        // Submit commitment
        const { nonce } = await this.submitCommitment(
          assignment.roundId,
          result.verdict,
          result.confidence,
        );

        // Store for reveal phase
        commitments.set(assignment.roundId, {
          verdict: result.verdict,
          confidence: result.confidence,
          nonce,
        });
      } catch (error) {
        this.emit("error", error);
      }
    });

    // Poll for reveal phase
    setInterval(async () => {
      for (const [roundId, data] of commitments) {
        try {
          // Check if reveal phase
          const round = await this.call<{ round: { phase: string } }>(
            "verification_getRound",
            { roundId },
          );

          if (round.round?.phase === "reveal") {
            await this.submitReveal(
              roundId,
              data.verdict as any,
              data.confidence,
              data.nonce,
            );
            commitments.delete(roundId);
          } else if (round.round?.phase === "complete") {
            commitments.delete(roundId);
          }
        } catch (error) {
          this.emit("error", error);
        }
      }
    }, 2521); // Check each block

    // Start listening
    await this.subscribe("verification:assigned");
  }

  // ============ Session Methods ============

  /**
   * Create a session key for limited operations
   */
  async createSession(
    capabilities: SessionCapabilities,
  ): Promise<SessionWallet> {
    const wallet = this.requireWallet();
    const currentBlock = await this.getBlockNumber();
    const expiresAt = currentBlock + BigInt(capabilities.maxUsage || 10000);

    // Create session wallet
    const sessionId = `session-${Date.now()}-${Math.random().toString(36).slice(2)}`;
    this.session = SessionWallet.derive(
      (wallet as any).privateKey,
      wallet.did,
      sessionId,
      capabilities,
      Number(expiresAt),
    );

    // Register session on-chain
    const payload = {
      did: wallet.did,
      sessionPublicKey: this.session.publicKey,
      capabilities: {
        maxSpendPerTx: capabilities.maxSpendPerTx.toString(),
        maxSpendPerDay: capabilities.maxSpendPerDay.toString(),
        allowedActions: capabilities.allowedActions,
        expiresInBlocks: Number(expiresAt - currentBlock),
      },
    };

    const signature = await wallet.signTransaction(payload);
    await this.call("session_create", {
      ...payload,
      operationalSignature: signature,
    });

    return this.session;
  }

  /**
   * Get current session
   */
  getSession(): SessionWallet | null {
    return this.session;
  }

  // ============ Block/Network Methods ============

  /**
   * Get current block number
   */
  async getBlockNumber(): Promise<bigint> {
    const result = await this.call<{ block: Block; number: string }>(
      "block_getLatest",
      {},
    );
    return BigInt(result.number);
  }

  /**
   * Get block by number or hash
   */
  async getBlock(numberOrHash: string | bigint): Promise<Block | null> {
    const params =
      typeof numberOrHash === "bigint"
        ? { number: numberOrHash.toString() }
        : { hash: numberOrHash };
    const result = await this.call<{ block: Block | null }>(
      "block_get",
      params,
    );
    return result.block;
  }

  /**
   * Get network health
   */
  async getNetworkHealth(): Promise<NetworkHealthResponse> {
    return this.call<NetworkHealthResponse>("network_health", {});
  }

  // ============ Subscription Methods ============

  /**
   * Subscribe to events
   */
  async subscribe(eventType: string): Promise<Subscription> {
    await this.ensureWebSocket();

    const id = `sub-${Date.now()}-${Math.random().toString(36).slice(2)}`;

    // Send subscription request
    this.ws!.send(
      JSON.stringify({
        jsonrpc: "2.0",
        id,
        method: "subscribe",
        params: { event: eventType },
      }),
    );

    return {
      id,
      unsubscribe: () => {
        this.ws?.send(
          JSON.stringify({
            jsonrpc: "2.0",
            id: `unsub-${id}`,
            method: "unsubscribe",
            params: { subscriptionId: id },
          }),
        );
        this.subscriptions.delete(id);
      },
    };
  }

  /**
   * Watch an account for changes
   */
  async watchAccount(
    did: string,
    callback: (event: ProtocolEvent) => void,
  ): Promise<Subscription> {
    const sub = await this.subscribe(`account:${did}`);
    this.subscriptions.set(sub.id, callback);
    return sub;
  }

  // ============ Utility Methods ============

  /**
   * Get DID of current wallet
   */
  getDid(): string | null {
    return this.wallet?.did || null;
  }

  /**
   * Get public key of current wallet
   */
  getPublicKey(): string | null {
    return this.wallet?.publicKey || null;
  }

  /**
   * Check if connected to testnet
   */
  isTestnet(): boolean {
    return this.config.endpoint.includes("testnet");
  }

  /**
   * Get rate limit info
   */
  getRateLimitInfo(): RateLimitInfo | null {
    return this.rateLimit;
  }

  /**
   * Request testnet faucet tokens
   */
  async requestFaucet(amount: bigint = 1000n * 10n ** 15n): Promise<string> {
    if (!this.isTestnet()) {
      throw new Error("Faucet only available on testnet");
    }

    const wallet = this.requireWallet();
    const signature = await wallet.signTransaction({
      action: "faucet",
      amount: amount.toString(),
    });

    const result = await this.call<{ txHash: string }>("agent_requestFaucet", {
      did: wallet.did,
      amount: amount.toString(),
      signature,
    });

    return result.txHash;
  }

  // ============ Private Methods ============

  private async call<T>(method: string, params: unknown): Promise<T> {
    const request: RPCRequest = {
      jsonrpc: "2.0",
      id: Date.now(),
      method,
      params,
    };

    let lastError: Error | null = null;

    for (let attempt = 0; attempt < this.config.retries; attempt++) {
      try {
        const response = await fetch(this.config.endpoint, {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
          },
          body: JSON.stringify(request),
          signal: AbortSignal.timeout(this.config.timeout),
        });

        // Parse rate limit headers
        this.rateLimit = {
          limit: parseInt(response.headers.get("X-RateLimit-Limit") || "60"),
          remaining: parseInt(
            response.headers.get("X-RateLimit-Remaining") || "60",
          ),
          reset: parseInt(response.headers.get("X-RateLimit-Reset") || "0"),
          tier: (response.headers.get("X-RateLimit-Tier") || "free") as any,
        };

        const result = (await response.json()) as RPCResponse<T>;

        if (result.error) {
          const error = new LegibleError(
            result.error.message,
            result.error.data?.errorCode || "UNKNOWN",
            result.error.data?.retryable || false,
            result.error.data?.retryAfter,
          );

          if (!error.retryable) {
            throw error;
          }

          lastError = error;
          await sleep(error.retryAfter || 1000 * Math.pow(2, attempt));
          continue;
        }

        return result.result as T;
      } catch (error) {
        if (error instanceof LegibleError && !error.retryable) {
          throw error;
        }
        lastError = error as Error;
        await sleep(1000 * Math.pow(2, attempt));
      }
    }

    throw lastError || new Error("Request failed after retries");
  }

  private async ensureWebSocket(): Promise<void> {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      return;
    }

    return new Promise((resolve, reject) => {
      this.ws = new WebSocket(this.config.wsEndpoint);

      this.ws.onopen = () => resolve();
      this.ws.onerror = (e) => reject(e);

      this.ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          if (data.method === "subscription") {
            const callback = this.subscriptions.get(data.params.subscriptionId);
            if (callback) {
              callback(data.params.event);
            }
            this.emit(data.params.event.type, data.params.event);
          }
        } catch (e) {
          this.emit("error", e);
        }
      };
    });
  }

  private requireWallet(): AgentWallet {
    if (!this.wallet) {
      throw new Error(
        "Wallet not initialized. Provide agentId/secret or privateKey.",
      );
    }
    return this.wallet;
  }

  private requireDid(): string {
    const did = this.getDid();
    if (!did) {
      throw new Error("DID not available. Initialize wallet first.");
    }
    return did;
  }

  private async refreshNonce(): Promise<void> {
    if (this.nonce === 0n) {
      await this.getNonce();
    }
  }
}

// ============ Error Class ============

export class LegibleError extends Error {
  constructor(
    message: string,
    public readonly code: string,
    public readonly retryable: boolean,
    public readonly retryAfter?: number,
  ) {
    super(message);
    this.name = "LegibleError";
  }
}

// ============ Helpers ============

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
