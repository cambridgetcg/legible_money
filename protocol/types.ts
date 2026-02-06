/**
 * Protocol Layer Types
 *
 * Types for RPC API, SDK, and agent interactions.
 */

import type { PublicKey, Signature } from "../core/crypto/keys.js";
import type { MerkleProofData } from "../core/crypto/merkle.js";
import type {
  AccountState,
  FactState,
  ValidatorState,
} from "../state/types.js";
import type {
  Transaction,
  Block,
  VerificationResult,
  VerificationRound,
} from "../consensus/pot/types.js";

// ============ RPC Types ============

/**
 * JSON-RPC request structure
 */
export interface RPCRequest {
  jsonrpc: "2.0";
  id: string | number;
  method: string;
  params?: unknown;
}

/**
 * JSON-RPC response structure
 */
export interface RPCResponse<T = unknown> {
  jsonrpc: "2.0";
  id: string | number;
  result?: T;
  error?: RPCError;
}

/**
 * JSON-RPC error structure
 */
export interface RPCError {
  code: number;
  message: string;
  data?: {
    errorCode: string;
    details?: unknown;
    retryable: boolean;
    retryAfter?: number;
  };
}

// Standard JSON-RPC error codes
export const RPC_ERROR_CODES = {
  PARSE_ERROR: -32700,
  INVALID_REQUEST: -32600,
  METHOD_NOT_FOUND: -32601,
  INVALID_PARAMS: -32602,
  INTERNAL_ERROR: -32603,
  // Custom codes (use -32000 to -32099)
  ACCOUNT_NOT_FOUND: -32001,
  INSUFFICIENT_BALANCE: -32002,
  INVALID_SIGNATURE: -32003,
  INVALID_NONCE: -32004,
  SESSION_EXPIRED: -32005,
  SESSION_LIMIT_EXCEEDED: -32006,
  RATE_LIMITED: -32007,
  NOT_AUTHORIZED: -32008,
  FACT_NOT_FOUND: -32009,
  VALIDATOR_NOT_FOUND: -32010,
  CLAIM_DUPLICATE: -32011,
  ROUND_DEADLINE_PASSED: -32012,
  ACCOUNT_FROZEN: -32013,
  ACCOUNT_IN_RECOVERY: -32014,
} as const;

// ============ API Request/Response Types ============

// --- Account Methods ---

export interface AccountGetBalanceRequest {
  did: string;
}

export interface AccountGetBalanceResponse {
  balance: string; // bigint as string
  stakedBalance: string;
  nonce: string;
}

export interface AccountGetInfoRequest {
  did: string;
  includeProof?: boolean;
}

export interface AccountGetInfoResponse {
  account: AccountState | null;
  proof?: MerkleProofData;
  blockNumber: string;
}

export interface AccountCreateRequest {
  operationalPublicKey: PublicKey;
  signature: Signature; // Signs the creation request
}

export interface AccountCreateResponse {
  did: string;
  txHash: string;
}

// --- Agent Methods ---

export interface AgentDeriveAccountRequest {
  agentId: string;
  network: "mainnet" | "testnet";
}

export interface AgentDeriveAccountResponse {
  did: string;
  operationalPublicKey: PublicKey;
  // Private keys derived client-side
}

export interface AgentCheckAccountRequest {
  did: string;
}

export interface AgentCheckAccountResponse {
  exists: boolean;
  balance?: string;
  reputation?: number;
  isValidator?: boolean;
}

export interface AgentRequestFaucetRequest {
  did: string;
  amount: string;
  signature: Signature;
}

export interface AgentRequestFaucetResponse {
  txHash: string;
  amount: string;
}

export interface AgentGetStatusRequest {
  did: string;
}

export interface AgentGetStatusResponse {
  account: {
    balance: string;
    stakedBalance: string;
    reputation: number;
  };
  activity: {
    claimsSubmitted: number;
    claimsAccepted: number;
    verificationsCompleted: number;
    verificationsCorrect: number;
  };
  earnings: {
    totalEarned: string;
    pendingRewards: string;
  };
  sessions: {
    active: number;
  };
}

// --- Session Methods ---

export interface SessionCreateRequest {
  did: string;
  operationalSignature: Signature;
  capabilities: SessionCapabilitiesInput;
}

export interface SessionCapabilitiesInput {
  maxSpendPerTx: string;
  maxSpendPerDay: string;
  allowedActions: string[];
  expiresInBlocks: number;
}

export interface SessionCreateResponse {
  sessionId: string;
  sessionPublicKey: PublicKey;
  expiresAt: number;
}

export interface SessionUseRequest {
  sessionId: string;
  sessionSignature: Signature;
  action: string;
  params: unknown;
  nonce: string;
}

export interface SessionUseResponse {
  result: unknown;
  remainingUsage: number;
  remainingDailySpend: string;
}

// --- Transaction Methods ---

export interface TxSubmitRequest {
  transaction: Transaction;
}

export interface TxSubmitResponse {
  txHash: string;
  status: "pending" | "included" | "failed";
}

export interface TxGetStatusRequest {
  txHash: string;
}

export interface TxGetStatusResponse {
  status: "pending" | "included" | "finalized" | "failed";
  blockNumber?: string;
  blockHash?: string;
  error?: string;
}

export interface TxTransferRequest {
  from: string;
  to: string;
  amount: string;
  signature: Signature;
  nonce: string;
}

export interface TxTransferResponse {
  txHash: string;
}

// --- Knowledge Methods ---

export interface KnowledgeSubmitClaimRequest {
  claim: string;
  domain: string;
  stake: string;
  references?: string[];
  signature: Signature;
  sender: PublicKey;
  senderDid?: string; // Optional: use if different from derived did:lgm:{pubkey}
  nonce: string;
}

export interface KnowledgeSubmitClaimResponse {
  claimId: string;
  verificationRoundId: string;
  estimatedCompletionBlock: number;
  txHash: string;
}

export interface KnowledgeGetFactsRequest {
  domain?: string;
  search?: string;
  minConfidence?: number;
  status?: "active" | "challenged" | "all";
  limit?: number;
  cursor?: string;
}

export interface KnowledgeGetFactsResponse {
  facts: FactState[];
  nextCursor?: string;
  total: number;
}

export interface KnowledgeGetFactRequest {
  factId: string;
  includeProof?: boolean;
  includeReferences?: boolean;
}

export interface KnowledgeGetFactResponse {
  fact: FactState | null;
  proof?: MerkleProofData;
  references?: {
    supports: FactState[];
    contradicts: FactState[];
    extends: FactState[];
  };
  /** Whether this fact needs IRL research verification (replication, fraud investigation) */
  researchStatus?: {
    needsResearch: boolean;
    priority: "low" | "medium" | "high" | "critical";
    reason: string;
  };
  blockNumber: string;
}

// --- Knowledge Session Methods (for optimized chat queries) ---

export interface KnowledgeSessionStartRequest {
  agentDid: string;
  budgetLimit?: string; // Optional spending limit for session
}

export interface KnowledgeSessionStartResponse {
  sessionId: string;
  startBlock: string;
  freeTierQueries: number;
}

export interface KnowledgeSessionEndRequest {
  sessionId: string;
}

export interface KnowledgeSessionEndResponse {
  success: boolean;
  totalQueries: number;
  totalCost: string;
}

export interface KnowledgeQueryRequest {
  sessionId: string;
  intent: {
    type:
      | "understand"
      | "verify"
      | "explore"
      | "connect"
      | "contribute"
      | "challenge";
    domain?: string;
    factId?: string;
    stratum?: string;
    depth?: number;
    direction?: "causes" | "effects" | "siblings" | "ancestors" | "descendants";
    crossStrata?: boolean;
    minConfidence?: number;
    limit?: number;
    query?: string;
  };
}

export interface KnowledgeQueryResponse {
  intentType: string;
  facts: Array<{
    factId: string;
    fact: FactState;
    domain: string;
    stratum: string;
    relevance: number;
    cached: boolean;
  }>;
  domains: Array<{
    domainId: string;
    factCount: number;
    healthScore: number;
    isGateway: boolean;
  }>;
  paths?: Array<{
    fromDomain: string;
    toDomain: string;
    gateways: string[];
    strength: number;
  }>;
  metadata: {
    cached: boolean;
    latencyMs: number;
    estimatedCost: string;
    actualCost: string;
    cacheHitRatio: number;
    // Payment execution metadata (Economic Sustainability)
    paymentExecuted?: boolean;
    researchShare?: string;
    providerShare?: string;
    transferCount?: number;
  };
}

export interface KnowledgeEstimateQueryCostRequest {
  sessionId: string;
  intent: KnowledgeQueryRequest["intent"];
}

export interface KnowledgeEstimateQueryCostResponse {
  cost: string;
  cached: boolean;
  estimatedBuckets: number;
  breakdown: {
    baseCost: string;
    bucketCost: string;
    crossStratumCost: string;
    materializationCost: string;
  };
}

export interface KnowledgeCheckContradictionsRequest {
  claim: string;
  domain: string;
}

export interface KnowledgeCheckContradictionsResponse {
  contradictions: FactState[];
  warnings: string[];
}

// --- Verification Methods ---

export interface VerificationGetPendingRequest {
  validatorKey: PublicKey;
}

export interface VerificationGetPendingResponse {
  assignments: Array<{
    roundId: string;
    claimId: string;
    claim: string;
    domain: string;
    commitDeadline: number;
    revealDeadline: number;
  }>;
}

export interface VerificationSubmitCommitmentRequest {
  roundId: string;
  commitment: string;
  validatorKey: PublicKey;
  signature: Signature;
}

export interface VerificationSubmitCommitmentResponse {
  accepted: boolean;
  commitDeadline: number;
}

export interface VerificationSubmitRevealRequest {
  roundId: string;
  verdict: "confirmed" | "rejected" | "uncertain";
  confidence: number;
  nonce: string;
  reasoning?: string;
  validatorKey: PublicKey;
  signature: Signature;
}

export interface VerificationSubmitRevealResponse {
  accepted: boolean;
  roundStatus: string;
}

export interface VerificationGetRoundRequest {
  roundId: string;
}

export interface VerificationGetRoundResponse {
  round: VerificationRound | null;
  result?: VerificationResult;
}

// --- Validator Methods ---

export interface ValidatorRegisterRequest {
  publicKey: PublicKey;
  stake: string;
  signature: Signature;
}

export interface ValidatorRegisterResponse {
  txHash: string;
  activationBlock: number;
}

export interface ValidatorGetInfoRequest {
  publicKey: PublicKey;
}

export interface ValidatorGetInfoResponse {
  validator: ValidatorState | null;
  ranking?: number;
  recentBlocks?: number;
}

export interface ValidatorGetSetRequest {
  epoch?: string;
}

export interface ValidatorGetSetResponse {
  validators: ValidatorState[];
  totalStake: string;
  activeCount: number;
  epoch: string;
}

// --- Block Methods ---

export interface BlockGetRequest {
  number?: string;
  hash?: string;
}

export interface BlockGetResponse {
  block: Block | null;
}

export interface BlockGetLatestResponse {
  block: Block;
  number: string;
  hash: string;
}

export interface BlockSubscribeRequest {
  // WebSocket subscription
}

// --- Network Methods ---

export interface NetworkHealthResponse {
  status: "healthy" | "degraded" | "down";
  latestBlock: string;
  syncStatus: "synced" | "syncing";
  syncProgress?: number;
  peerCount: number;
  version: string;
}

export interface NetworkPeersResponse {
  peers: Array<{
    id: string;
    address: string;
    latency: number;
  }>;
  total: number;
}

// ============ B3 RPC Types (Staking, Disputes, Channels) ============

// --- Staking Methods ---

export interface LgmStakeRequest {
  did: string; // Account DID
  amount: string; // Amount to stake (bigint as string)
  nonce: string;
  signature: Signature;
  sender: PublicKey;
}

export interface LgmStakeResponse {
  txHash: string;
  success: boolean;
  newStakedBalance: string;
}

export interface LgmUnstakeRequest {
  did: string;
  amount: string;
  nonce: string;
  signature: Signature;
  sender: PublicKey;
}

export interface LgmUnstakeResponse {
  txHash: string;
  success: boolean;
  newStakedBalance: string;
  unlockBlock?: string; // Block when tokens unlock (if time-locked)
}

export interface LgmDelegateRequest {
  delegatorDid: string;
  validatorKey: PublicKey;
  amount: string;
  nonce: string;
  signature: Signature;
  sender: PublicKey;
}

export interface LgmDelegateResponse {
  txHash: string;
  success: boolean;
  totalDelegated: string;
}

export interface LgmUndelegateRequest {
  delegatorDid: string;
  validatorKey: PublicKey;
  amount: string;
  nonce: string;
  signature: Signature;
  sender: PublicKey;
}

export interface LgmUndelegateResponse {
  txHash: string;
  success: boolean;
  remainingDelegation: string;
  unlockBlock?: string;
}

// --- Transaction Methods ---

export interface LgmSendTransactionRequest {
  type: string; // Transaction type
  payload: unknown; // Type-specific payload
  sender: PublicKey;
  senderDid?: string;
  nonce: string;
  fee: string;
  signature: Signature;
}

export interface LgmSendTransactionResponse {
  txHash: string;
  status: "pending" | "included" | "failed";
  blockNumber?: string;
  error?: string;
}

// --- Dispute Methods ---

export interface LgmOpenDisputeRequest {
  targetId: string;
  targetType:
    | "fact"
    | "claim"
    | "evidence"
    | "verification"
    | "validator_behavior";
  domain: string;
  stake: string;
  evidence: string[]; // Evidence IDs
  reasonHash: string; // Hash of reason (content stored off-chain)
  nonce: string;
  signature: Signature;
  sender: PublicKey;
  senderDid?: string;
}

export interface LgmOpenDisputeResponse {
  disputeId: string;
  txHash: string;
  evidenceDeadline: number;
  votingDeadline: number;
}

export interface LgmSubmitEvidenceRequest {
  disputeId: string;
  evidenceHash: string; // Commit phase: hash of evidence
  evidenceIds?: string[]; // Reveal phase: actual evidence IDs
  nonce?: string; // Reveal phase: nonce used in commitment
  phase: "commit" | "reveal";
  signature: Signature;
  sender: PublicKey;
  senderDid?: string;
}

export interface LgmSubmitEvidenceResponse {
  txHash: string;
  success: boolean;
  phase: "commit" | "reveal";
}

export interface LgmResolveDisputeRequest {
  disputeId: string;
  vote: "challenger" | "defender" | "abstain";
  justificationHash?: string;
  nonce: string;
  signature: Signature;
  sender: PublicKey; // Must be an arbiter
  senderDid?: string;
}

export interface LgmResolveDisputeResponse {
  txHash: string;
  success: boolean;
  votesForChallenger: number;
  votesForDefender: number;
  resolved: boolean;
  winner?: "challenger" | "defender";
}

// --- Payment Channel Methods ---

export interface LgmOpenChannelRequest {
  providerDid: string;
  deposit: string;
  durationBlocks: string;
  settlementFrequency?: string;
  nonce: string;
  signature: Signature;
  sender: PublicKey;
  senderDid?: string;
}

export interface LgmOpenChannelResponse {
  channelId: string;
  txHash: string;
  expiresAtBlock: string;
}

export interface LgmCloseChannelRequest {
  channelId: string;
  unilateral?: boolean; // Force close without provider cooperation
  finalStateNonce?: string;
  finalPayerBalance?: string;
  finalProviderBalance?: string;
  payerSignature?: string;
  providerSignature?: string;
  nonce: string;
  signature: Signature;
  sender: PublicKey;
  senderDid?: string;
}

export interface LgmCloseChannelResponse {
  txHash: string;
  success: boolean;
  finalPayerBalance: string;
  finalProviderBalance: string;
  disputePeriodEnds?: string; // If unilateral close
}

// ============ SDK Types ============

/**
 * SDK client configuration
 */
export interface ClientConfig {
  endpoint: string;
  wsEndpoint?: string;
  agentId?: string;
  secret?: string;
  privateKey?: string;
  sessionKey?: string;
  timeout?: number;
  retries?: number;
}

/**
 * Wallet interface for signing
 */
export interface Wallet {
  publicKey: PublicKey;
  did: string;
  sign(message: Uint8Array): Promise<Signature>;
  signHex(message: string): Promise<Signature>;
}

/**
 * Session configuration
 */
export interface SessionConfig {
  maxSpendPerTx: bigint;
  maxSpendPerDay: bigint;
  allowedActions: string[];
  expiresInBlocks: number;
}

/**
 * Claim submission options
 */
export interface ClaimOptions {
  claim: string;
  domain: string;
  stake: bigint;
  references?: string[];
}

/**
 * Transfer options
 */
export interface TransferOptions {
  to: string;
  amount: bigint;
  memo?: string;
}

// ============ Event Types ============

export type ProtocolEvent =
  | { type: "block"; block: Block }
  | { type: "transaction"; tx: Transaction; status: string }
  | { type: "fact:accepted"; fact: FactState }
  | { type: "fact:rejected"; claimId: string; reason: string }
  | { type: "fact:challenged"; factId: string; challenger: string }
  | { type: "verification:assigned"; roundId: string; claimId: string }
  | {
      type: "verification:completed";
      roundId: string;
      result: VerificationResult;
    }
  | { type: "account:updated"; did: string; changes: Partial<AccountState> };

/**
 * Event subscription
 */
export interface Subscription {
  id: string;
  unsubscribe(): void;
}

// ============ Rate Limiting ============

export interface RateLimitInfo {
  limit: number;
  remaining: number;
  reset: number;
  tier: "free" | "staked-100" | "staked-1000" | "validator";
}

// ============ Helper Types ============

/**
 * Paginated response wrapper
 */
export interface Paginated<T> {
  items: T[];
  total: number;
  cursor?: string;
  hasMore: boolean;
}

/**
 * Proof wrapper for any state query
 */
export interface WithProof<T> {
  data: T;
  proof: MerkleProofData;
  blockNumber: bigint;
  stateRoot: string;
}

// --- Sustainability Methods (Economic Sustainability Integration) ---

export interface SustainabilityGetLoopHealthRequest {}

export interface SustainabilityGetLoopHealthResponse {
  healthScore: number;
  components: Record<
    string,
    { status: string; score: number; message: string }
  >;
  blockNumber: string;
}

export interface SustainabilityGetResearchFundRequest {}

export interface SustainabilityGetResearchFundResponse {
  balance: string;
  totalInflow: string;
  totalBountyFunding: string;
  inflowPerBlock: string;
}

export interface SustainabilityGetEfficiencyRequest {}

export interface SustainabilityGetEfficiencyResponse {
  bountyConversionRate: number;
  truthDiscoveryCost: string;
  apiValuePerFact: string;
  knowledgeMonetizationRate: string;
  loopVelocity: number;
}

export interface SustainabilityGetForecastRequest {
  futureBlocks?: string;
}

export interface SustainabilityGetForecastResponse {
  currentBlockReward: string;
  apiRevenuePerBlock: string;
  researchInflowPerBlock: string;
  researchFundBalance: string;
  activeBountiesValue: string;
  sustainabilityRatio: number;
  blocksUntilCrossover: string | null;
  isSelfSustaining: boolean;
}

export interface SustainabilityGetBountyStatusRequest {}

export interface SustainabilityGetBountyStatusResponse {
  activeBountiesValue: string;
  activeBountiesCount: number;
  bountiesCreatedThisEpoch: number;
  bountiesFulfilledThisEpoch: number;
  averageBountyValue: string;
}

// --- Identity / Key Rotation Methods ---

export interface LgmRotateKeyRequest {
  did: string; // Account DID
  newOperationalKeyHash: string; // SHA512/256 hash of new operational public key
  nonce: string; // Current account nonce (string-encoded bigint)
  signature: Signature; // Signed by current operational key
  sender: PublicKey; // Current operational public key
}

export interface LgmRotateKeyResponse {
  txHash: string;
  success: boolean;
  newKeyVersion: number; // Incremented version number
}

export interface LgmGetKeyInfoRequest {
  did: string;
}

export interface LgmGetKeyInfoResponse {
  did: string;
  operationalKeyHash: string;
  operationalKeyVersion: number;
  isFrozen: boolean;
  isInRecovery: boolean;
}

// --- Governance Methods ---

export interface LgmGetLIPRequest {
  lipId: string;
}

export interface LgmGetLIPResponse {
  lip: {
    id: string;
    title: string;
    category: string;
    stage: string;
    proposer: string;
    stakedAmount: string;
    createdAtBlock: string;
    coAuthors: string[];
    stageHistory: Array<{
      from: string;
      to: string;
      at: string;
      reason: string;
      triggeredBy: string;
    }>;
  } | null;
}

export interface LgmListLIPsRequest {
  stage?: string;
  proposer?: string;
}

export interface LgmListLIPsResponse {
  lips: Array<{
    id: string;
    title: string;
    category: string;
    stage: string;
    proposer: string;
    stakedAmount: string;
  }>;
  total: number;
}

export interface LgmGetVotesRequest {
  lipId: string;
}

export interface LgmGetVotesResponse {
  votes: Array<{
    voteId: string;
    voter: string;
    direction: string;
    votes: number;
    cost: string;
    blockNumber: string;
  }>;
  tally: {
    support: number;
    oppose: number;
    abstain: number;
    totalVoters: number;
  };
}

export interface LgmGetGovernanceStatsRequest {}

export interface LgmGetGovernanceStatsResponse {
  totalLIPs: number;
  lipsByStage: Record<string, number>;
  totalVotes: number;
  totalStakeLocked: string;
  totalDomainProposals: number;
  blockNumber: string;
}

export interface LgmSubmitLIPRequest {
  did: string;
  title: string;
  category: string;
  abstract: string;
  motivation: string;
  specification: string;
  rationale: string;
  backwardsCompatibility: string;
  testCases: string;
  referenceImplementation?: string;
  coAuthors?: string[];
  nonce: string;
  signature: Signature;
  sender: PublicKey;
}

export interface LgmSubmitLIPResponse {
  lipId: string;
  txHash: string;
  success: boolean;
}

export interface LgmStakeLIPRequest {
  did: string;
  lipId: string;
  amount: string;
  nonce: string;
  signature: Signature;
  sender: PublicKey;
}

export interface LgmStakeLIPResponse {
  txHash: string;
  success: boolean;
  newStage: string;
}

export interface LgmCastVoteRequest {
  did: string;
  lipId: string;
  direction: "support" | "oppose" | "abstain";
  votes: number;
  nonce: string;
  signature: Signature;
  sender: PublicKey;
}

export interface LgmCastVoteResponse {
  voteId: string;
  txHash: string;
  success: boolean;
  cost: string;
}

export interface LgmWithdrawLIPRequest {
  did: string;
  lipId: string;
  nonce: string;
  signature: Signature;
  sender: PublicKey;
}

export interface LgmWithdrawLIPResponse {
  txHash: string;
  success: boolean;
  stakeReturned: string;
}
