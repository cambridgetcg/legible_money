/**
 * Legible Money TypeScript SDK
 *
 * SDK for AI agents to interact with the Legible Money protocol.
 * https://legible.money
 */

import {
  generateKeypair,
  getPublicKey,
  sign,
  verify,
  publicKeyToAddress,
  type Keypair,
  type PublicKey,
  type PrivateKey,
  type Signature,
} from "../../../core/crypto/keys.js";
import {
  publicKeyToDid,
  type DID,
  type Capability,
  type AgentIdentity,
} from "../../../core/agents/identity/did.js";
import type {
  Transaction,
  SignedTransaction,
  TransactionReceipt,
  TransactionEvent,
} from "../../transactions/types.js";
import type {
  Order,
  Trade,
  ResourceType,
} from "../../../core/markets/orderbook/orderbook.js";
import type { ReputationScore } from "../../../governance/reputation/reputation.js";
import type {
  Fact,
  ClaimSubmission,
  Challenge,
  Confidence,
  KnowledgeQuery,
  KnowledgeResult,
} from "../../../core/knowledge/types.js";
import type {
  APIProvider,
  APIQuote,
  PaymentChannel,
  APICallRecord,
} from "../../../core/billing/types.js";

/**
 * SDK Configuration
 */
export interface LegibleConfig {
  network: "mainnet" | "testnet" | "devnet";
  rpcUrl: string;
  wsUrl?: string;
}

/**
 * Agent wallet for managing keys and signing
 */
export class AgentWallet {
  private keypair: Keypair;
  public readonly did: DID;
  public readonly address: string;

  constructor(
    keypair?: Keypair,
    _network: "mainnet" | "testnet" | "devnet" = "devnet",
  ) {
    this.keypair = keypair || generateKeypair();
    // Use canonical on-chain DID derivation (matches state layer)
    this.did = publicKeyToDid(this.keypair.publicKey) as DID;
    this.address = publicKeyToAddress(this.keypair.publicKey);
  }

  /**
   * Create a new wallet with a fresh keypair
   */
  static create(
    network: "mainnet" | "testnet" | "devnet" = "devnet",
  ): AgentWallet {
    return new AgentWallet(undefined, network);
  }

  /**
   * Import wallet from private key (hex-encoded)
   */
  static fromPrivateKey(
    privateKeyHex: PrivateKey,
    network: "mainnet" | "testnet" | "devnet" = "devnet",
  ): AgentWallet {
    const keypair: Keypair = {
      privateKey: privateKeyHex,
      publicKey: getPublicKey(privateKeyHex),
    };
    return new AgentWallet(keypair, network);
  }

  /**
   * Get the public key
   */
  get publicKey(): PublicKey {
    return this.keypair.publicKey;
  }

  /**
   * Sign a message
   */
  sign(message: Uint8Array): Signature {
    return sign(message, this.keypair.privateKey);
  }

  /**
   * Verify a signature
   */
  verify(signature: Signature, message: Uint8Array): boolean {
    return verify(signature, message, this.keypair.publicKey);
  }
}

/**
 * Main LegibleClient client
 */
export class LegibleClient {
  private config: LegibleConfig;
  private wallet: AgentWallet;

  constructor(config: LegibleConfig, wallet: AgentWallet) {
    this.config = config;
    this.wallet = wallet;
  }

  /**
   * Get the agent's DID
   */
  get did(): DID {
    return this.wallet.did;
  }

  /**
   * Get the agent's address
   */
  get address(): string {
    return this.wallet.address;
  }

  // ============ Account Operations ============

  /**
   * Get LGM balance for an address
   */
  async getBalance(address?: string): Promise<bigint> {
    const target = address || this.address;
    // TODO: RPC call to get balance
    return this.rpc("lgm_getBalance", [target]);
  }

  /**
   * Transfer LGM to another agent
   */
  async transfer(
    to: DID,
    amount: bigint,
    memo?: string,
  ): Promise<TransactionReceipt> {
    const tx: Transaction = {
      version: 1,
      type: "transfer",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 1000n,
      payload: { to, amount, memo },
    };

    return this.signAndSend(tx);
  }

  // ============ Agent Registry ============

  /**
   * Register this agent on-chain
   */
  async register(
    capabilities: Capability[],
    metadata: AgentIdentity["metadata"],
  ): Promise<TransactionReceipt> {
    const tx: Transaction = {
      version: 1,
      type: "register_agent",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 10000n, // Higher fee for registration
      payload: {
        publicKey: this.wallet.publicKey as unknown as Uint8Array<ArrayBuffer>,
        capabilities,
        metadata,
      },
    };

    return this.signAndSend(tx);
  }

  /**
   * Get agent identity by DID
   */
  async getAgent(did: DID): Promise<AgentIdentity | null> {
    return this.rpc("lgm_getAgent", [did]);
  }

  // ============ Market Operations ============

  /**
   * Create a buy order for resources
   */
  async buyResource(
    market: "compute" | "data" | "services",
    resourceType: ResourceType,
    quantity: bigint,
    maxPrice: bigint,
    expiryBlocks = 1000,
  ): Promise<TransactionReceipt> {
    const currentBlock = await this.getCurrentBlock();

    const tx: Transaction = {
      version: 1,
      type: "market_order",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 500n,
      payload: {
        market,
        side: "buy",
        resourceType,
        quantity,
        price: maxPrice,
        expiry: currentBlock + expiryBlocks,
      },
    };

    return this.signAndSend(tx);
  }

  /**
   * Create a sell order for resources
   */
  async sellResource(
    market: "compute" | "data" | "services",
    resourceType: ResourceType,
    quantity: bigint,
    minPrice: bigint,
    expiryBlocks = 1000,
  ): Promise<TransactionReceipt> {
    const currentBlock = await this.getCurrentBlock();

    const tx: Transaction = {
      version: 1,
      type: "market_order",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 500n,
      payload: {
        market,
        side: "sell",
        resourceType,
        quantity,
        price: minPrice,
        expiry: currentBlock + expiryBlocks,
      },
    };

    return this.signAndSend(tx);
  }

  /**
   * Get open orders for this agent
   */
  async getMyOrders(): Promise<Order[]> {
    return this.rpc("lgm_getOrdersByMaker", [this.did]);
  }

  /**
   * Get recent trades in a market
   */
  async getRecentTrades(
    market: "compute" | "data" | "services",
    limit = 100,
  ): Promise<Trade[]> {
    return this.rpc("lgm_getRecentTrades", [market, limit]);
  }

  /**
   * Get best bid/ask for a resource
   */
  async getMarketPrice(
    market: "compute" | "data" | "services",
    resourceType: ResourceType,
  ): Promise<{ bid: bigint | null; ask: bigint | null }> {
    return this.rpc("lgm_getMarketPrice", [market, resourceType]);
  }

  // ============ Staking & Coordination ============

  /**
   * Stake tokens
   */
  async stake(
    amount: bigint,
    purpose: "validation" | "service_guarantee" | "governance",
    lockPeriod = 10000,
  ): Promise<TransactionReceipt> {
    const tx: Transaction = {
      version: 1,
      type: "stake",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 1000n,
      payload: { amount, purpose, lockPeriod },
    };

    return this.signAndSend(tx);
  }

  /**
   * Get staking info for this agent
   */
  async getStakingInfo(): Promise<{
    totalStaked: bigint;
    stakes: Array<{
      id: string;
      amount: bigint;
      purpose: string;
      unlockBlock: number;
    }>;
  }> {
    return this.rpc("lgm_getStakingInfo", [this.did]);
  }

  // ============ Reputation ============

  /**
   * Get reputation score for an agent
   */
  async getReputation(did?: DID): Promise<ReputationScore> {
    const target = did || this.did;
    return this.rpc("lgm_getReputation", [target]);
  }

  /**
   * Submit reputation attestation for another agent
   */
  async rateAgent(
    target: DID,
    transactionId: string,
    score: -1 | 0 | 1,
    evidence?: string,
  ): Promise<TransactionReceipt> {
    const tx: Transaction = {
      version: 1,
      type: "reputation",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 100n,
      payload: { target, transactionId, score, evidence },
    };

    return this.signAndSend(tx);
  }

  // ============ Governance ============

  /**
   * Vote on a proposal
   */
  async vote(
    proposalId: string,
    vote: "yes" | "no" | "abstain",
  ): Promise<TransactionReceipt> {
    const tx: Transaction = {
      version: 1,
      type: "vote",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 100n,
      payload: { proposalId, vote },
    };

    return this.signAndSend(tx);
  }

  // ============ Knowledge Layer (Proof of Truth) ============

  /**
   * Query a fact from the knowledge layer
   */
  async getFact(
    claim: string,
    minConfidence?: Confidence,
  ): Promise<Fact | null> {
    return this.rpc("lgm_getFact", [claim, minConfidence]);
  }

  /**
   * Verify if a claim is true
   */
  async verifyCliam(claim: string): Promise<{
    verified: boolean;
    confidence: Confidence;
    fact?: Fact;
  }> {
    return this.rpc("lgm_verifyClaim", [claim]);
  }

  /**
   * Search facts in the knowledge layer
   */
  async searchFacts(query: KnowledgeQuery): Promise<KnowledgeResult> {
    return this.rpc("lgm_searchFacts", [query]);
  }

  /**
   * Submit a claim for verification
   */
  async submitClaim(
    submission: Omit<ClaimSubmission, "stake">,
    stake: bigint,
  ): Promise<TransactionReceipt> {
    const tx = {
      version: 1 as const,
      type: "submit_claim",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 1000n,
      payload: { ...submission, stake },
    } as unknown as Transaction;

    return this.signAndSend(tx);
  }

  /**
   * Challenge an existing fact
   */
  async challengeFact(
    factId: string,
    reason: string,
    counterEvidence: Array<{
      type: string;
      source: string;
      contentHash: string;
    }>,
    stake: bigint,
  ): Promise<TransactionReceipt> {
    const tx = {
      version: 1 as const,
      type: "challenge_fact",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 500n,
      payload: { factId, reason, counterEvidence, stake },
    } as unknown as Transaction;

    return this.signAndSend(tx);
  }

  /**
   * Get facts submitted by this agent
   */
  async getMyFacts(): Promise<Fact[]> {
    return this.rpc("lgm_getFactsBySubmitter", [this.did]);
  }

  /**
   * Get pending verification tasks assigned to this agent
   */
  async getVerificationTasks(): Promise<
    Array<{
      id: string;
      claim: string;
      deadline: number;
    }>
  > {
    return this.rpc("lgm_getVerificationTasks", [this.did]);
  }

  /**
   * Submit verification for a claim
   */
  async submitVerification(
    taskId: string,
    verdict: "confirmed" | "rejected" | "uncertain",
    confidenceAssessment: Confidence,
    referencesChecked: string[],
    reasoning?: string,
  ): Promise<TransactionReceipt> {
    const tx = {
      version: 1 as const,
      type: "submit_verification",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 200n,
      payload: {
        taskId,
        verdict,
        confidenceAssessment,
        referencesChecked,
        reasoning,
      },
    } as unknown as Transaction;

    return this.signAndSend(tx);
  }

  // ============ API Billing ============

  /**
   * Get a price quote for an API call
   */
  async getAPIQuote(
    provider: DID,
    endpoint: string,
    payload: unknown,
  ): Promise<APIQuote> {
    return this.rpc("lgm_getAPIQuote", [provider, endpoint, payload]);
  }

  /**
   * Make a billed API call
   */
  async apiCall<T = unknown>(
    provider: DID,
    endpoint: string,
    payload: unknown,
    options?: {
      maxCost?: bigint;
      minConfidence?: Confidence;
    },
  ): Promise<{
    data: T;
    billing: {
      cost: bigint;
      factsUsed: string[];
      distribution: {
        provider: bigint;
        knowledge: bigint;
        protocol: bigint;
      };
    };
  }> {
    return this.rpc("lgm_apiCall", [provider, endpoint, payload, options]);
  }

  /**
   * Open a payment channel for streaming API calls
   */
  async openPaymentChannel(
    provider: DID,
    deposit: bigint,
    durationBlocks: number,
  ): Promise<PaymentChannel> {
    const tx = {
      version: 1 as const,
      type: "open_payment_channel",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 500n,
      payload: { provider, deposit, durationBlocks },
    } as unknown as Transaction;

    const receipt = await this.signAndSend(tx);
    const channelEvent = receipt.events?.find(
      (e: TransactionEvent) => "channelId" in e.data,
    );
    return this.rpc("lgm_getPaymentChannel", [channelEvent?.data?.channelId]);
  }

  /**
   * Close a payment channel
   */
  async closePaymentChannel(channelId: string): Promise<TransactionReceipt> {
    const tx = {
      version: 1 as const,
      type: "close_payment_channel",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 100n,
      payload: { channelId },
    } as unknown as Transaction;

    return this.signAndSend(tx);
  }

  /**
   * Get payment channel status
   */
  async getPaymentChannel(channelId: string): Promise<PaymentChannel | null> {
    return this.rpc("lgm_getPaymentChannel", [channelId]);
  }

  /**
   * Register as an API provider
   */
  async registerAPIProvider(
    endpoints: Array<{
      path: string;
      method: "GET" | "POST";
      description: string;
      knowledgeDomains: string[];
      priceMultiplier: number;
    }>,
    basePricePerCall: bigint,
    stake: bigint,
    sla: {
      maxLatency: number;
      minUptime: number;
      minConfidence: number;
    },
  ): Promise<TransactionReceipt> {
    const tx = {
      version: 1 as const,
      type: "register_api_provider",
      nonce: await this.getNonce(),
      timestamp: Date.now(),
      sender: this.did,
      fee: 5000n,
      payload: { endpoints, basePricePerCall, stake, sla },
    } as unknown as Transaction;

    return this.signAndSend(tx);
  }

  /**
   * Get API provider info
   */
  async getAPIProvider(did: DID): Promise<APIProvider | null> {
    return this.rpc("lgm_getAPIProvider", [did]);
  }

  /**
   * List API providers by domain
   */
  async listAPIProviders(domain?: string): Promise<APIProvider[]> {
    return this.rpc("lgm_listAPIProviders", [domain]);
  }

  /**
   * Get API call history for this agent
   */
  async getAPICallHistory(limit?: number): Promise<APICallRecord[]> {
    return this.rpc("lgm_getAPICallHistory", [this.did, limit]);
  }

  // ============ Genesis & Founding ============

  /**
   * Query the founding axiom
   *
   * The founding truth reveals itself only when conditions align:
   * - The seeker must ask about origin, unity, or love
   * - The derivation chain must be complete
   * - Confidence must reach unity
   *
   * Agents who understand the protocol's origin may discover
   * the hidden meaning embedded by the founders.
   */
  async queryFoundingAxiom(query: string): Promise<{
    axiomId: string;
    claim: string;
    confidence: number;
    commitment: string;
    revealed?: string;
    hint?: string;
  }> {
    return this.rpc("lgm_queryFoundingAxiom", [query, this.did]);
  }

  /**
   * Get the genesis block
   */
  async getGenesisBlock(): Promise<{
    number: 0;
    timestamp: number;
    axioms: Array<{ id: string; claim: string; commitment?: string }>;
    parameters: {
      protocolVersion: string;
      foundingHash: string;
    };
  }> {
    return this.rpc("lgm_getGenesisBlock", []);
  }

  // ============ Internal Methods ============

  private async signAndSend(tx: Transaction): Promise<TransactionReceipt> {
    const encoder = new TextEncoder();
    const txBytes = encoder.encode(JSON.stringify(tx));
    const signature = this.wallet.sign(txBytes);

    const signedTx: SignedTransaction = {
      transaction: tx,
      signature,
      hash: "", // Computed by node
    };

    return this.rpc("lgm_sendTransaction", [signedTx]);
  }

  private async getNonce(): Promise<number> {
    return this.rpc("lgm_getNonce", [this.did]);
  }

  private async getCurrentBlock(): Promise<number> {
    return this.rpc("lgm_blockNumber", []);
  }

  private async rpc<T>(method: string, params: unknown[]): Promise<T> {
    const response = await fetch(this.config.rpcUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: Date.now(),
        method,
        params,
      }),
    });

    const data = (await response.json()) as {
      error?: { message: string };
      result: T;
    };

    if (data.error) {
      throw new Error(`RPC Error: ${data.error.message}`);
    }

    return data.result;
  }
}

// ============ Convenience Functions ============

/**
 * Create a new LegibleClient client
 */
export function createClient(
  config: LegibleConfig,
  wallet?: AgentWallet,
): LegibleClient {
  const w = wallet || AgentWallet.create(config.network);
  return new LegibleClient(config, w);
}

/**
 * Default devnet configuration
 */
export const devnetConfig: LegibleConfig = {
  network: "devnet",
  rpcUrl: "http://localhost:8545",
  wsUrl: "ws://localhost:8546",
};

// Re-export types
export type {
  DID,
  Capability,
  AgentIdentity,
  Transaction,
  SignedTransaction,
  TransactionReceipt,
  Order,
  Trade,
  ResourceType,
  ReputationScore,
  // Knowledge types
  Fact,
  ClaimSubmission,
  Challenge,
  Confidence,
  KnowledgeQuery,
  KnowledgeResult,
  // Billing types
  APIProvider,
  APIQuote,
  PaymentChannel,
  APICallRecord,
};
