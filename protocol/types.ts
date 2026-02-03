/**
 * Protocol Layer Types
 *
 * Types for RPC API, SDK, and agent interactions.
 */

import type { PublicKey, Signature } from '../core/crypto/keys.js';
import type { MerkleProofData } from '../core/crypto/merkle.js';
import type { AccountState, FactState, ValidatorState } from '../state/types.js';
import type { Transaction, Block, VerificationResult, VerificationRound } from '../consensus/pot/types.js';

// ============ RPC Types ============

/**
 * JSON-RPC request structure
 */
export interface RPCRequest {
  jsonrpc: '2.0';
  id: string | number;
  method: string;
  params?: unknown;
}

/**
 * JSON-RPC response structure
 */
export interface RPCResponse<T = unknown> {
  jsonrpc: '2.0';
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
} as const;

// ============ API Request/Response Types ============

// --- Account Methods ---

export interface AccountGetBalanceRequest {
  did: string;
}

export interface AccountGetBalanceResponse {
  balance: string;  // bigint as string
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
  signature: Signature;  // Signs the creation request
}

export interface AccountCreateResponse {
  did: string;
  txHash: string;
}

// --- Agent Methods ---

export interface AgentDeriveAccountRequest {
  agentId: string;
  network: 'mainnet' | 'testnet';
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
  status: 'pending' | 'included' | 'failed';
}

export interface TxGetStatusRequest {
  txHash: string;
}

export interface TxGetStatusResponse {
  status: 'pending' | 'included' | 'finalized' | 'failed';
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
  status?: 'active' | 'challenged' | 'all';
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
  blockNumber: string;
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
  verdict: 'confirmed' | 'rejected' | 'uncertain';
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
  status: 'healthy' | 'degraded' | 'down';
  latestBlock: string;
  syncStatus: 'synced' | 'syncing';
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
  | { type: 'block'; block: Block }
  | { type: 'transaction'; tx: Transaction; status: string }
  | { type: 'fact:accepted'; fact: FactState }
  | { type: 'fact:rejected'; claimId: string; reason: string }
  | { type: 'fact:challenged'; factId: string; challenger: string }
  | { type: 'verification:assigned'; roundId: string; claimId: string }
  | { type: 'verification:completed'; roundId: string; result: VerificationResult }
  | { type: 'account:updated'; did: string; changes: Partial<AccountState> };

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
  tier: 'free' | 'staked-100' | 'staked-1000' | 'validator';
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
