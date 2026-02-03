/**
 * State Layer Types
 *
 * Core types for the persistent state layer.
 * State is organized as multiple Sparse Merkle Trees combined into a unified root.
 *
 * Architecture:
 *
 *                    StateRoot
 *                        │
 *           ┌────────────┼────────────┐
 *           │            │            │
 *      AccountRoot  ValidatorRoot  KnowledgeRoot
 *           │            │            │
 *         [SMT]        [SMT]        [SMT]
 *
 * Each sub-tree is a 256-bit keyed Sparse Merkle Tree.
 * The StateRoot is the hash of all sub-roots concatenated.
 */

import type { PublicKey } from '../core/crypto/keys.js';
import type { MerkleRoot, MerkleProofData } from '../core/crypto/merkle.js';
import type { VerificationResult } from '../consensus/pot/types.js';
import type { Confidence } from '../core/knowledge/types.js';

// ============ State Roots ============

/**
 * Combined state root - hash of all sub-roots
 */
export type StateRoot = string;  // Hex-encoded, 32 bytes

/**
 * State root components
 */
export interface StateRootComponents {
  accountRoot: MerkleRoot;
  validatorRoot: MerkleRoot;
  knowledgeRoot: MerkleRoot;
  researchRoot?: MerkleRoot;  // Research state tree root (optional for backward compat)
  evidenceRoot?: MerkleRoot;  // Evidence state tree root (optional for backward compat)
  disputeRoot?: MerkleRoot;   // Dispute state tree root (optional for backward compat)
  vestingRoot?: MerkleRoot;   // Vesting state tree root (optional for backward compat)
  version: bigint;          // State version (block number)
}

// ============ Account State ============

/**
 * On-chain account state (subset of AgentAccount stored in SMT)
 */
export interface AccountState {
  did: string;
  balance: bigint;
  stakedBalance: bigint;
  nonce: bigint;
  operationalKeyHash: string;     // Hash of current operational key
  operationalKeyVersion: number;
  sessionKeyCount: number;        // Number of active session keys
  reputationScore: number;        // Cached reputation (0-1)
  lastActiveBlock: bigint;
  flags: AccountFlags;
}

/**
 * Account flags packed into a single number
 */
export interface AccountFlags {
  isValidator: boolean;
  canSubmitClaims: boolean;
  canChallenge: boolean;
  inRecovery: boolean;
  frozen: boolean;
}

/**
 * Key for account lookup in SMT
 * Key = hash(did)
 */
export type AccountKey = Uint8Array;

/**
 * Account state change
 */
export interface AccountDelta {
  did: string;
  balanceChange?: bigint;         // Positive or negative
  stakedChange?: bigint;
  nonceIncrement?: boolean;
  reputationChange?: number;
  setFlags?: Partial<AccountFlags>;
}

// ============ Validator State ============

/**
 * On-chain validator state
 */
export interface ValidatorState {
  publicKey: PublicKey;
  did: string;
  stake: bigint;
  delegatedStake: bigint;
  totalStake: bigint;             // stake + delegatedStake
  reputation: number;
  isActive: boolean;
  activeSinceBlock: bigint;
  lastProposedBlock: bigint;

  // Performance
  blocksProposed: number;
  verificationsCompleted: number;
  verificationsCorrect: number;
  slashCount: number;

  // Commission for delegators
  commissionRate: number;         // 0-1, fraction of rewards
}

/**
 * Delegation record
 */
export interface DelegationState {
  delegator: string;              // DID
  validator: PublicKey;
  amount: bigint;
  delegatedAt: bigint;            // Block number
  unlocksAt?: bigint;             // If unbonding
}

/**
 * Validator set snapshot (for epoch boundary)
 */
export interface ValidatorSetState {
  epoch: bigint;
  totalStake: bigint;
  activeValidators: number;
  validatorRoot: MerkleRoot;      // Root of validator states
  delegationRoot: MerkleRoot;     // Root of delegation states
}

// ============ Knowledge State ============

/**
 * On-chain fact state (verified claim)
 */
export interface FactState {
  id: string;                     // Fact ID (hash of claim)
  claim: string;                  // The claim text
  domain: string;                 // Knowledge domain
  confidence: Confidence;
  verifiedAt: bigint;             // Block number
  verifierCount: number;
  challengeCount: number;
  lastChallengedAt?: bigint;
  status: FactStatus;

  // Merkle proof elements
  referenceRoot: MerkleRoot;      // Root of references to other facts
}

export type FactStatus =
  | 'active'                      // Currently accepted as true
  | 'challenged'                  // Under challenge
  | 'superseded'                  // Replaced by newer fact
  | 'revoked';                    // Proven false, removed

/**
 * Reference between facts
 */
export interface FactReference {
  fromFactId: string;
  toFactId: string;
  referenceType: ReferenceType;
  strength: number;               // 0-1, how strongly connected
}

export type ReferenceType =
  | 'supports'                    // This fact supports another
  | 'contradicts'                 // This fact contradicts another
  | 'extends'                     // This fact extends another
  | 'specializes'                 // This fact is a special case
  | 'generalizes';                // This fact is a generalization

/**
 * Knowledge domain (for organizing facts)
 */
export interface DomainState {
  id: string;
  name: string;
  factCount: number;
  rootFactId?: string;            // Genesis fact for domain
  parentDomain?: string;          // Hierarchical domains
}

// ============ State Transitions ============

/**
 * A state transition that can be applied atomically
 */
export interface StateTransition {
  type: TransitionType;
  blockNumber: bigint;
  transactionIndex: number;
  payload: TransitionPayload;
}

export type TransitionType =
  // Account transitions
  | 'create_account'
  | 'transfer'
  | 'stake'
  | 'unstake'
  | 'delegate'
  | 'undelegate'
  | 'rotate_key'
  | 'authorize_session'
  | 'revoke_session'
  | 'update_reputation'

  // Validator transitions
  | 'register_validator'
  | 'deactivate_validator'
  | 'slash_validator'
  | 'epoch_transition'

  // Knowledge transitions
  | 'add_fact'
  | 'challenge_fact'
  | 'resolve_challenge'
  | 'revoke_fact'
  | 'add_reference'

  // Research transitions (Protocol Research Repository)
  | 'research_submit'
  | 'research_challenge'
  | 'bounty_create'
  | 'bounty_claim'
  | 'bounty_fulfill'

  // Evidence transitions (Evidence Management System)
  | 'evidence_submit'
  | 'evidence_archive_request'
  | 'evidence_archive_complete'
  | 'evidence_update_version'
  | 'evidence_confirm_duplicate'
  | 'evidence_heartbeat_update'
  | 'evidence_expire'
  | 'evidence_invalidate'
  | 'oracle_register_feed'
  | 'oracle_update_feed'
  | 'oracle_report_staleness'

  // Dispute transitions (Dispute Resolution System)
  | 'dispute_initiate'
  | 'dispute_defend'
  | 'dispute_commit_evidence'
  | 'dispute_reveal_evidence'
  | 'dispute_select_arbiters'
  | 'dispute_arbiter_vote'
  | 'dispute_escalate'
  | 'dispute_advance_phase'
  | 'dispute_settle'
  | 'dispute_update_expertise'

  // Vesting transitions (Gradual Reward Release System)
  | 'vesting_create'
  | 'vesting_claim'
  | 'vesting_pause'
  | 'vesting_resume'
  | 'vesting_accelerate'
  | 'vesting_falsify'
  | 'vesting_complete';

export type TransitionPayload =
  | CreateAccountPayload
  | TransferPayload
  | StakePayload
  | DelegatePayload
  | SlashPayload
  | AddFactPayload
  | ChallengeFactPayload
  | EpochTransitionPayload;

export interface CreateAccountPayload {
  did: string;
  operationalKeyHash: string;
  initialBalance: bigint;
}

export interface TransferPayload {
  from: string;                   // DID
  to: string;                     // DID
  amount: bigint;
  memo?: string;
}

export interface StakePayload {
  account: string;                // DID
  amount: bigint;
  action: 'stake' | 'unstake';
}

export interface DelegatePayload {
  delegator: string;              // DID
  validator: PublicKey;
  amount: bigint;
  action: 'delegate' | 'undelegate';
}

export interface SlashPayload {
  validator: PublicKey;
  amount: bigint;
  reason: string;
  evidence: string;               // Hash of evidence
}

export interface AddFactPayload {
  fact: Omit<FactState, 'verifiedAt' | 'status'>;
  verificationResult: VerificationResult;
}

export interface ChallengeFactPayload {
  factId: string;
  challenger: string;             // DID
  stake: bigint;
  evidence: string;
}

export interface EpochTransitionPayload {
  oldEpoch: bigint;
  newEpoch: bigint;
  newValidatorSet: ValidatorSetState;
  rewards: Map<string, bigint>;   // validator -> reward
  slashes: Map<string, bigint>;   // validator -> slash
}

// ============ State Proofs ============

/**
 * Proof that a state exists at a given root
 */
export interface StateProof {
  root: StateRoot;
  blockNumber: bigint;
  proofs: StateSubProof[];
}

export interface StateSubProof {
  tree: 'account' | 'validator' | 'knowledge' | 'research' | 'evidence' | 'dispute';
  key: string;
  proof: MerkleProofData;
}

/**
 * Proof of account balance (for light clients)
 */
export interface BalanceProof {
  account: string;                // DID
  balance: bigint;
  stateRoot: StateRoot;
  blockNumber: bigint;
  proof: MerkleProofData;
}

/**
 * Proof of fact existence
 */
export interface FactProof {
  factId: string;
  fact: FactState;
  stateRoot: StateRoot;
  blockNumber: bigint;
  proof: MerkleProofData;
}

// ============ State Snapshots ============

/**
 * Complete state snapshot (for sync/recovery)
 */
export interface StateSnapshot {
  version: bigint;
  blockNumber: bigint;
  blockHash: string;

  roots: StateRootComponents;

  // Metadata
  accountCount: number;
  validatorCount: number;
  factCount: number;
  researchCount?: number;
  bountyCount?: number;
  evidenceCount?: number;
  disputeCount?: number;

  // Checksum for integrity
  checksum: string;
}

/**
 * Incremental state update (for efficient sync)
 */
export interface StateDiff {
  fromBlock: bigint;
  toBlock: bigint;
  fromRoot: StateRoot;
  toRoot: StateRoot;

  // Changes
  accountChanges: AccountDelta[];
  validatorChanges: ValidatorState[];
  factChanges: FactState[];

  // Transitions applied
  transitions: StateTransition[];
}

// ============ Configuration ============

/**
 * State layer configuration
 */
export interface StateConfig {
  // Pruning
  keepHistoricalStates: number;   // Number of block states to keep
  snapshotInterval: number;       // Blocks between snapshots

  // Caching
  accountCacheSize: number;
  validatorCacheSize: number;
  factCacheSize: number;

  // Limits
  maxAccountsPerBlock: number;
  maxFactsPerBlock: number;
}

export const DEFAULT_STATE_CONFIG: StateConfig = {
  keepHistoricalStates: 1000,
  snapshotInterval: 100,

  accountCacheSize: 10000,
  validatorCacheSize: 1000,
  factCacheSize: 5000,

  maxAccountsPerBlock: 100,
  maxFactsPerBlock: 10,
};

// ============ Error Types ============

export class StateError extends Error {
  constructor(
    message: string,
    public code: StateErrorCode,
    public details?: Record<string, unknown>
  ) {
    super(message);
    this.name = 'StateError';
  }
}

export type StateErrorCode =
  | 'ACCOUNT_NOT_FOUND'
  | 'INSUFFICIENT_BALANCE'
  | 'INVALID_NONCE'
  | 'VALIDATOR_NOT_FOUND'
  | 'FACT_NOT_FOUND'
  | 'INVALID_TRANSITION'
  | 'PROOF_INVALID'
  | 'STATE_MISMATCH'
  | 'ROLLBACK_FAILED'
  // Research errors
  | 'RESEARCH_NOT_FOUND'
  | 'BOUNTY_NOT_FOUND'
  | 'UNAUTHORIZED'
  // Evidence errors
  | 'EVIDENCE_NOT_FOUND'
  | 'ARCHIVE_NOT_FOUND'
  | 'VERSION_LIMIT_EXCEEDED'
  | 'DUPLICATE_EVIDENCE'
  | 'FEED_NOT_FOUND'
  | 'STALE_EVIDENCE'
  // Dispute errors
  | 'DISPUTE_NOT_FOUND'
  | 'INVALID_PHASE'
  | 'INSUFFICIENT_STAKE'
  | 'ALREADY_COMMITTED'
  | 'COMMITMENT_MISMATCH'
  | 'ALREADY_VOTED'
  | 'NOT_ARBITER'
  | 'CANNOT_ESCALATE'
  | 'DEADLINE_PASSED'
  | 'ALREADY_RESOLVED'
  // Vesting errors
  | 'VESTING_NOT_FOUND'
  | 'VESTING_PAUSED'
  | 'VESTING_COMPLETED'
  | 'VESTING_FALSIFIED'
  | 'CLIFF_NOT_ENDED'
  | 'NOTHING_TO_CLAIM'
  | 'NOT_RECIPIENT';
