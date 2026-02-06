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

import type { PublicKey } from "../core/crypto/keys.js";
import type { MerkleRoot, MerkleProofData } from "../core/crypto/merkle.js";
import type { VerificationResult } from "../consensus/pot/types.js";
import type {
  Confidence,
  EpistemicCategory,
  FactStatus as CoreFactStatus,
} from "../core/knowledge/types.js";
import type {
  OntologicalStratum,
  StratumDependency,
} from "../core/knowledge/ontology/types.js";
import type { Decidability, LogicZone } from "../core/logic/types.js";

// ============ State Roots ============

/**
 * Combined state root - hash of all sub-roots
 */
export type StateRoot = string; // Hex-encoded, 32 bytes

/**
 * State root components
 */
export interface StateRootComponents {
  accountRoot: MerkleRoot;
  validatorRoot: MerkleRoot;
  knowledgeRoot: MerkleRoot;
  claimRoot?: MerkleRoot; // Claim submission state tree root
  researchRoot?: MerkleRoot; // Research state tree root (optional for backward compat)
  evidenceRoot?: MerkleRoot; // Evidence state tree root (optional for backward compat)
  disputeRoot?: MerkleRoot; // Dispute state tree root (optional for backward compat)
  vestingRoot?: MerkleRoot; // Vesting state tree root (optional for backward compat)
  scheduleRoot?: MerkleRoot; // DMSS Schedule Process State root (SPS 131313)
  tokenExpirationRoot?: MerkleRoot; // Expirable token tracking root
  contractRoot?: MerkleRoot; // Self-invoking contract registry root
  governanceRoot?: MerkleRoot; // Governance state tree root (LIPs, votes, deployments)
  operatorRoot?: MerkleRoot; // Operator registry state root
  paymentChannelRoot?: MerkleRoot; // Payment channel state tree root (GAP-005)
  projectRoot?: MerkleRoot; // Project state tree root (Tree of Life)
  serviceRoot?: MerkleRoot; // Service state tree root (Tree of Life)
  taskRoot?: MerkleRoot; // Task state tree root (Tree of Life)
  seedRoot?: MerkleRoot; // Opportunity seed state tree root (Tree of Life)

  // Domain Capture Defense state roots
  qualificationRoot?: MerkleRoot; // Validator qualification state root
  bountyPoolRoot?: MerkleRoot; // Challenge bounty pool state root
  captureChallengeRoot?: MerkleRoot; // Active capture challenges root
  isolatedReputationRoot?: MerkleRoot; // Isolated reputation state root

  version: bigint; // State version (block number)
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
  operationalKeyHash: string; // Hash of current operational key
  operationalKeyVersion: number;
  sessionKeyCount: number; // Number of active session keys
  reputationScore: number; // Cached reputation (0-1)
  lastActiveBlock: bigint;
  flags: AccountFlags;
  /** Agent availability for project work */
  availability?: AgentAvailability;
}

/**
 * Agent availability broadcast for project recruitment
 */
export interface AgentAvailability {
  available: boolean;
  capabilities: string[];
  preferredDomains: string[];
  minimumBounty: bigint;
  updatedAtBlock: bigint;
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
  balanceChange?: bigint; // Positive or negative
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
  totalStake: bigint; // stake + delegatedStake
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
  commissionRate: number; // 0-1, fraction of rewards
}

/**
 * Delegation record
 */
export interface DelegationState {
  delegator: string; // DID
  validator: PublicKey;
  amount: bigint;
  delegatedAt: bigint; // Block number
  unlocksAt?: bigint; // If unbonding
}

/**
 * Unbonding entry for tracking undelegation in progress
 * SECURITY: Undelegation requires a timelock to prevent stake-and-slash attacks
 */
export interface UnbondingEntry {
  id: string; // Unique ID for this unbonding entry
  delegator: string; // DID of the delegator
  validator: PublicKey; // Validator being undelegated from
  amount: bigint; // Amount being undelegated
  initiatedAt: bigint; // Block number when undelegation started
  completesAt: bigint; // Block number when unbonding completes
  status: "pending" | "completed" | "cancelled";
}

/**
 * Validator set snapshot (for epoch boundary)
 */
export interface ValidatorSetState {
  epoch: bigint;
  totalStake: bigint;
  activeValidators: number;
  validatorRoot: MerkleRoot; // Root of validator states
  delegationRoot: MerkleRoot; // Root of delegation states
}

// ============ Knowledge State ============

/**
 * On-chain fact state (verified claim)
 */
export interface FactState {
  id: string; // Fact ID (hash of claim)
  claim: string; // The claim text
  domain: string; // Knowledge domain
  confidence: Confidence;
  verifiedAt: bigint; // Block number
  verifierCount: number;
  challengeCount: number;
  lastChallengedAt?: bigint;
  status: FactStatus;

  // Merkle proof elements
  referenceRoot: MerkleRoot; // Root of references to other facts

  // GAP-009: Expiration support
  expiresAt?: bigint; // Block number when fact expires (optional explicit expiration)
  lastReverifiedAt?: bigint; // Block number of last re-verification (for decay tracking)

  // Citation economics (Economic Sustainability Integration)
  citationCount: number; // Number of times this fact has been cited/queried
  lastCitedBlock?: bigint; // Block number of most recent citation
  totalCitationRewards?: bigint; // Cumulative LGM earned from citation rewards

  // Ontological metadata
  epistemicCategory?: EpistemicCategory; // Epistemic category (from domain or stratum default)
  stratum?: OntologicalStratum; // Stratum of the resolved domain
  resolvedDomainId?: string; // The domain ID resolved from ontology tree
  ontologyValidated?: boolean; // Whether claim passed ontology validation
  stratumDependencies?: StratumDependency[]; // Cross-stratum dependencies
  stratumCeiling?: number; // Confidence ceiling from stratum
  crossStratumDiscount?: number; // Discount factor from dependencies

  // Gödelian metadata (U1 fix: propagated from ClaimState)
  logicZone?: LogicZone; // Logic zone of the claim
  decidability?: Decidability; // Decidability assessment at submission
}

/**
 * Re-export unified FactStatus from core/knowledge/types.ts (single source of truth).
 * State layer primarily uses: 'active', 'challenged', 'superseded', 'revoked', 'expired'.
 */
export type FactStatus = CoreFactStatus;

/**
 * GAP-008: Information about confidence decay for a fact
 *
 * Provides both stored (original) and effective (time-decayed) confidence,
 * enabling consumers to make informed decisions based on fact freshness.
 */
export interface DecayInfo {
  /** Original confidence at verification time */
  storedConfidence: Confidence;
  /** Current effective confidence after decay */
  effectiveConfidence: Confidence;
  /** Decay multiplier (0-1, where 1 = no decay) */
  decayFactor: number;
  /** Whether this fact type never decays (eternal truths) */
  isEternal: boolean;
  /** Blocks elapsed since verification */
  blocksSinceVerification: number;
  /** Decay category used for calculation */
  category: string;
}

// ============ Claim State ============

/**
 * On-chain claim submission record.
 * Tracks claim from submission through verification to fact creation.
 *
 * Lifecycle: pending → verifying → accepted/rejected/expired/withdrawn
 */
export interface ClaimState {
  /** Unique claim ID (hash of claim + submitter + block) */
  id: string;

  /** Original claim text */
  claim: string;

  /** Claim type (empirical, logical, definitional, etc.) */
  claimType: string;

  /** Resolved ontological domain */
  domain: string;

  /** Resolved ontological stratum */
  stratum: OntologicalStratum;

  /** Submitter DID */
  submitter: string;

  /** Stake locked for this claim */
  stake: bigint;

  /** Block number of submission */
  submittedAt: bigint;

  /** References to supporting facts */
  references: string[];

  /** Current status */
  status: ClaimStatus;

  /** Verification round ID (once assigned) */
  verificationRoundId?: string;

  /** Resulting fact ID (if accepted) */
  resultFactId?: string;

  /** Block when status last changed */
  statusChangedAt: bigint;

  /** Logic zone classification */
  logicZone?: LogicZone;

  /** Decidability assessment */
  decidability?: Decidability;

  /** Gödelian status (benign, goedelian, paradoxical) */
  goedelianStatus?: string;

  /** Reason for rejection (if rejected) */
  rejectionReason?: string;

  /** If derived from existing facts (inference chain) */
  derivedFrom?: {
    factIds: string[];
    inferenceType: string;
    logicChain: string;
  };

  /** Optional schema ID for schema-based claims */
  schemaId?: string;
}

export type ClaimStatus =
  | "pending" // Submitted, awaiting verification round
  | "verifying" // In active verification round
  | "accepted" // Verification passed, fact created
  | "rejected" // Verification failed
  | "expired" // Timed out without verification
  | "withdrawn"; // Submitter withdrew claim

/**
 * Payload for claim_submit transaction
 */
export interface ClaimSubmitPayload {
  type: "claim_submit";
  claim: string;
  claimType: string;
  domain: string;
  stratum: OntologicalStratum;
  stake: bigint;
  references: string[];
  /** Optional formal notation for logical analysis */
  formalNotation?: string;
  /** If derived from existing facts (inference chain) */
  derivedFrom?: {
    factIds: string[];
    inferenceType: string;
    logicChain: string;
  };
  /** Optional schema ID for schema-based claims */
  schemaId?: string;
}

/**
 * Payload for claim_start_verification transaction (system)
 */
export interface ClaimStartVerificationPayload {
  type: "claim_start_verification";
  claimId: string;
  roundId: string;
  selectedValidators: string[];
}

/**
 * Payload for claim_complete_verification transaction (system)
 */
export interface ClaimCompleteVerificationPayload {
  type: "claim_complete_verification";
  claimId: string;
  roundId: string;
  outcome: "accepted" | "rejected" | "contested" | "insufficient";
  finalConfidence: number;
  verifierCount: number;
  /** Rewards to distribute (validator DID -> amount) */
  rewards: Array<[string, bigint]>;
  /** Slashes to apply (validator DID -> amount) */
  slashes: Array<[string, bigint]>;
}

/**
 * Payload for claim_withdraw transaction
 */
export interface ClaimWithdrawPayload {
  type: "claim_withdraw";
  claimId: string;
}

// ============ Payment Channel Types (GAP-005) ============

/**
 * Payment channel status
 */
export type PaymentChannelStatus = "open" | "closing" | "disputed" | "closed";

/**
 * On-chain payment channel state
 * Enables streaming micropayments for long-running API sessions.
 */
export interface PaymentChannelState {
  // Identity
  channelId: string;
  payer: string; // DID
  provider: string; // DID

  // Funds (all bigint for precision)
  deposited: bigint;
  spent: bigint;
  available: bigint;

  // State
  status: PaymentChannelStatus;
  openedAt: bigint; // Block number
  expiresAt: bigint; // Block number

  // Off-chain state tracking
  lastNonce: bigint; // Latest confirmed nonce
  lastStateHash: string; // Hash of latest confirmed state

  // Pending distributions (not yet settled)
  pendingDistributions: SerializedPaymentDistribution[];
  totalDistributed: bigint;

  // Settlement
  lastSettlementBlock: bigint;
  settlementFrequency: bigint; // Blocks between settlements (default: 100)

  // Dispute handling
  disputeStartBlock?: bigint;
  disputeDeadline?: bigint;
  challengedStateHash?: string;
}

/**
 * Signed off-chain state proof
 */
export interface SignedChannelState {
  channelId: string;
  nonce: bigint;
  spent: bigint;
  available: bigint;
  stateHash: string; // Hash of (channelId, nonce, spent, distributions)

  // Accumulated distributions since last settlement
  distributions: SerializedPaymentDistribution[];

  // Signatures
  payerSignature: string;
  providerSignature: string;
}

/**
 * Serialized payment distribution (bigint-safe for JSON)
 */
export interface SerializedPaymentDistribution {
  apiProvider: { did: string; amount: string };
  knowledgeRewards: Array<{ did: string; factId: string; amount: string }>;
  protocol: { burned: string; treasury: string };
  research?: { amount: string; treasury: string };
}

/**
 * Payload for open_payment_channel transaction
 */
export interface OpenPaymentChannelPayload {
  type: "open_payment_channel";
  provider: string; // Provider DID
  deposit: bigint;
  durationBlocks: bigint;
  settlementFrequency?: bigint;
}

/**
 * Payload for update_payment_channel transaction
 */
export interface UpdatePaymentChannelPayload {
  type: "update_payment_channel";
  channelId: string;
  signedState: SignedChannelState;
}

/**
 * Payload for close_payment_channel transaction
 */
export interface ClosePaymentChannelPayload {
  type: "close_payment_channel";
  channelId: string;
  finalState?: SignedChannelState; // Cooperative close
  unilateral?: boolean; // Unilateral close (starts dispute period)
}

/**
 * Payload for settle_payment_channel transaction
 */
export interface SettlePaymentChannelPayload {
  type: "settle_payment_channel";
  channelId: string;
  signedState: SignedChannelState;
}

/**
 * Payload for dispute_payment_channel transaction
 */
export interface DisputePaymentChannelPayload {
  type: "dispute_payment_channel";
  channelId: string;
  challengeState: SignedChannelState; // State with higher nonce
}

/**
 * Reference between facts
 */
export interface FactReference {
  fromFactId: string;
  toFactId: string;
  referenceType: ReferenceType;
  strength: number; // 0-1, how strongly connected
}

export type ReferenceType =
  | "supports" // This fact supports another
  | "contradicts" // This fact contradicts another
  | "extends" // This fact extends another
  | "specializes" // This fact is a special case
  | "generalizes" // This fact is a generalization
  | "derives_from"; // This fact is derived from another (inference)

/**
 * Knowledge domain (for organizing facts)
 */
export interface DomainState {
  id: string;
  name: string;
  factCount: number;
  rootFactId?: string; // Genesis fact for domain
  parentDomain?: string; // Hierarchical domains

  // Ontological metadata
  stratum?: OntologicalStratum; // Stratum level of this domain
  childDomains?: string[]; // Child domain IDs
  dependsOnStrata?: OntologicalStratum[]; // Strata this domain depends on
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
  | "create_account"
  | "transfer"
  | "stake"
  | "unstake"
  | "delegate"
  | "undelegate"
  | "rotate_key"
  | "authorize_session"
  | "revoke_session"
  | "update_reputation"

  // Validator transitions
  | "register_validator"
  | "deactivate_validator"
  | "slash_validator"
  | "epoch_transition"

  // Claim lifecycle transitions
  | "claim_submit" // User submits claim
  | "claim_start_verification" // System starts verification round
  | "claim_complete_verification" // System completes verification
  | "claim_withdraw" // User withdraws pending claim

  // Knowledge transitions
  | "add_fact"
  | "challenge_fact"
  | "resolve_challenge"
  | "revoke_fact"
  | "add_reference"

  // Research transitions (Protocol Research Repository)
  | "research_submit"
  | "research_challenge"
  | "bounty_create"
  | "bounty_claim"
  | "bounty_fulfill"

  // Evidence transitions (Evidence Management System)
  | "evidence_submit"
  | "evidence_archive_request"
  | "evidence_archive_complete"
  | "evidence_update_version"
  | "evidence_confirm_duplicate"
  | "evidence_heartbeat_update"
  | "evidence_expire"
  | "evidence_invalidate"
  | "oracle_register_feed"
  | "oracle_update_feed"
  | "oracle_report_staleness"

  // Dispute transitions (Dispute Resolution System)
  | "dispute_initiate"
  | "dispute_defend"
  | "dispute_commit_evidence"
  | "dispute_reveal_evidence"
  | "dispute_select_arbiters"
  | "dispute_arbiter_vote"
  | "dispute_escalate"
  | "dispute_advance_phase"
  | "dispute_settle"
  | "dispute_update_expertise"

  // Vesting transitions (Gradual Reward Release System)
  | "vesting_create"
  | "vesting_claim"
  | "vesting_pause"
  | "vesting_resume"
  | "vesting_accelerate"
  | "vesting_falsify"
  | "vesting_complete"

  // DMSS Schedule transitions (Decentralized Message Scheduling System)
  | "schedule_create"
  | "schedule_execute"
  | "schedule_pause"
  | "schedule_resume"
  | "schedule_cancel"
  | "schedule_add_fee"

  // Expirable Token transitions (Dynamic Array Method)
  | "token_mint_expirable"
  | "token_transfer_expirable"
  | "expired_token_retrieve"
  | "rai_distribute"
  | "token_extend_expiration"

  // Self-Invoking Contract transitions
  | "contract_deploy"
  | "contract_self_invoke"
  | "contract_add_gas"
  | "contract_pause"
  | "contract_resume"
  | "contract_add_invocation"

  // Governance transitions (Agent-LIP system)
  | "lip_submit"
  | "lip_stake"
  | "lip_amend"
  | "lip_advance_stage"
  | "lip_withdraw"

  // Governance review transitions
  | "review_commit"
  | "review_reveal"
  | "review_finalize"

  // Governance voting transitions
  | "vote_cast"
  | "vote_lock"
  | "vote_unlock"
  | "vote_switch"

  // Operator transitions
  | "operator_register"
  | "operator_add_agent"
  | "operator_remove_agent"
  | "operator_slash"

  // Deployment transitions
  | "deployment_create"
  | "deployment_advance"
  | "deployment_approve"
  | "deployment_rollback"

  // Domain proposal transitions (Ontological Tree Growth)
  | "domain_propose"
  | "domain_endorse"
  | "domain_challenge"
  | "domain_finalize"

  // Domain Qualification transitions (Domain Capture Defense)
  | "qualification_stake"
  | "qualification_track_record"
  | "qualification_endorse"
  | "qualification_inherit"
  | "qualification_suspend"
  | "qualification_revoke"
  | "qualification_renew"

  // Capture Challenge transitions (Challenge Bounties)
  | "capture_challenge_submit"
  | "capture_challenge_evidence"
  | "capture_challenge_review"
  | "capture_challenge_resolve"
  | "bounty_pool_fund"
  | "bounty_pool_auto_fund"

  // Reputation Isolation transitions
  | "reputation_update"
  | "reputation_decay"

  // Tree of Life transitions
  | "create_project"
  | "propose_project"
  | "assign_task"
  | "submit_deliverable"
  | "approve_deliverable"
  | "deploy_service"
  | "call_service"
  | "subscribe_service"
  | "pause_service"
  | "retire_service"

  // Tree of Life lifecycle transitions
  | "start_development"
  | "complete_project"
  | "begin_seeding"
  | "pause_project"
  | "resume_project"
  | "abandon_project"
  | "add_contributor"
  | "add_task"
  | "start_work"
  | "reject_deliverable"
  | "reopen_task"
  | "detect_opportunity"
  | "claim_opportunity"
  | "apply_to_project"
  | "review_application"
  | "set_availability"

  // Disbursement transitions
  | "disbursement_submit"
  | "disbursement_stake"
  | "disbursement_advance_stage"
  | "disbursement_execute"
  | "disbursement_cancel";

export type TransitionPayload =
  | CreateAccountPayload
  | TransferPayload
  | StakePayload
  | DelegatePayload
  | SlashPayload
  | AddFactPayload
  | ChallengeFactPayload
  | EpochTransitionPayload
  // Claim lifecycle payloads
  | ClaimSubmitPayload
  | ClaimStartVerificationPayload
  | ClaimCompleteVerificationPayload
  | ClaimWithdrawPayload
  | DomainProposePayload
  | DomainEndorsePayload
  | DomainChallengePayload
  | DomainFinalizePayload
  // Domain Capture Defense payloads
  | QualificationStakePayload
  | QualificationTrackRecordPayload
  | QualificationEndorsePayload
  | QualificationInheritPayload
  | CaptureChallengeSubmitPayload
  | CaptureChallengeEvidencePayload
  | CaptureChallengeResolvePayload
  | BountyPoolFundPayload
  | RotateKeyPayload
  | ReputationUpdatePayload
  // Governance payloads
  | LipSubmitPayload
  | LipStakePayload
  | LipAdvanceStagePayload
  | LipWithdrawPayload
  | VoteCastPayload
  // Disbursement payloads
  | DisbursementSubmitPayload
  | DisbursementStakePayload
  | DisbursementAdvanceStagePayload
  | DisbursementExecutePayload
  | DisbursementCancelPayload;

// ============ Governance Payloads ============

export interface LipSubmitPayload {
  title: string;
  category: "consensus" | "economics" | "interface" | "process";
  abstract: string;
  motivation: string;
  specification: string;
  rationale: string;
  backwardsCompatibility: string;
  testCases: string;
  referenceImplementation?: string;
  coAuthors?: string[];
}

export interface LipStakePayload {
  lipId: string;
  amount: bigint;
}

export interface LipAdvanceStagePayload {
  lipId: string;
}

export interface LipWithdrawPayload {
  lipId: string;
}

export interface VoteCastPayload {
  lipId: string;
  direction: "support" | "oppose" | "abstain";
  votes: number;
}

// ============ Disbursement Payloads ============

export interface DisbursementSubmitPayload {
  title: string;
  description: string;
  category: string; // DisbursementCategory
  treasuryFund: string; // TreasuryFund
  recipientDid: string;
  amount: bigint;
  justification: string;
  externalReference?: string;
}

export interface DisbursementStakePayload {
  disbursementId: string;
  amount: bigint;
}

export interface DisbursementAdvanceStagePayload {
  disbursementId: string;
}

export interface DisbursementExecutePayload {
  disbursementId: string;
}

export interface DisbursementCancelPayload {
  disbursementId: string;
}

export interface RotateKeyPayload {
  account: string; // DID of the account rotating the key
  newOperationalKeyHash: string; // Hash of the new operational key
}

export interface CreateAccountPayload {
  did: string;
  operationalKeyHash: string;
  initialBalance: bigint;
}

export interface TransferPayload {
  from: string; // DID
  to: string; // DID
  amount: bigint;
  memo?: string;
}

export interface StakePayload {
  account: string; // DID
  amount: bigint;
  action: "stake" | "unstake";
}

export interface DelegatePayload {
  delegator: string; // DID
  validator: PublicKey;
  amount: bigint;
  action: "delegate" | "undelegate";
}

export interface SlashPayload {
  validator: PublicKey;
  amount: bigint;
  reason: string;
  evidence: string; // Hash of evidence
}

export interface AddFactPayload {
  fact: Omit<FactState, "verifiedAt" | "status">;
  verificationResult: VerificationResult;
}

export interface ChallengeFactPayload {
  factId: string;
  challenger: string; // DID
  stake: bigint;
  evidence: string;
}

export interface EpochTransitionPayload {
  oldEpoch: bigint;
  newEpoch: bigint;
  newValidatorSet: ValidatorSetState;
  rewards: Map<string, bigint>; // validator -> reward
  slashes: Map<string, bigint>; // validator -> slash
}

// ============ Domain Proposal Payloads ============

/**
 * Payload for proposing a new domain in the ontological tree
 *
 * Stake requirements (22/222/2222 LGM pattern):
 * - Leaf domains (depth > 2): 22 LGM
 * - Mid-level domains (depth 2): 222 LGM
 * - Top-level domains (depth 1): 2222 LGM
 */
export interface DomainProposePayload {
  /** Proposed domain ID (dot notation, e.g., 'biology.marine.cetaceans') */
  domainId: string;

  /** Human-readable name */
  name: string;

  /** Target stratum (must match parent unless cross-stratum enabled) */
  stratum: OntologicalStratum;

  /** Parent domain ID ('root' for top-level) */
  parentDomainId: string;

  /** Description of what this domain covers */
  description: string;

  /** Example claims (minimum 2 required) */
  examples: string[];

  /** Proposer's DID */
  proposer: string;

  /** Stake amount (must meet minimum for domain type) */
  stake: bigint;
}

/**
 * Payload for endorsing a domain proposal
 * Requires 2 validator endorsements for auto-approval
 *
 * Validators must lock stake when endorsing:
 * - Top-level domains: 222 LGM
 * - Mid-level domains: 22 LGM
 * - Leaf domains: 2 LGM
 */
export interface DomainEndorsePayload {
  /** Proposal ID to endorse */
  proposalId: string;

  /** Endorser's DID (must be a validator) */
  endorser: string;

  /** Stake to lock (must meet minimum for domain level) */
  stake: bigint;

  /** Optional comment */
  comment?: string;
}

/**
 * Payload for challenging a domain proposal
 */
export interface DomainChallengePayload {
  /** Proposal ID to challenge */
  proposalId: string;

  /** Challenger's DID */
  challenger: string;

  /** Reason for challenge */
  reason: string;

  /** Stake backing the challenge */
  stake: bigint;
}

/**
 * Payload for finalizing a domain proposal after challenge period
 * 777 blocks challenge period (~32.7 minutes at 2521ms blocks)
 * 77% stake return on rejection
 */
export interface DomainFinalizePayload {
  /** Proposal ID to finalize */
  proposalId: string;
}

export interface DomainGovernanceApprovePayload {
  /** Proposal ID to approve */
  proposalId: string;
  /** LIP ID that authorizes this approval */
  lipId: string;
}

// ============ Domain Qualification Payloads ============

/**
 * Payload for qualifying via stake commitment
 *
 * Stake requirements (22/222/2222 LGM pattern):
 * - Leaf domains (depth > 2): 22 LGM
 * - Mid-level domains (depth 2): 222 LGM
 * - Top-level domains (depth 1): 2222 LGM
 */
export interface QualificationStakePayload {
  /** Validator's DID */
  validatorDid: string;

  /** Domain to qualify for */
  domain: string;

  /** Stake to lock */
  stake: bigint;
}

/**
 * Payload for qualifying via track record
 * Requires 19 verifications at 77% accuracy in related domain
 */
export interface QualificationTrackRecordPayload {
  /** Validator's DID */
  validatorDid: string;

  /** Domain to qualify for */
  domain: string;

  /** Related domain with existing qualification */
  relatedDomain: string;
}

/**
 * Payload for endorsing another validator for qualification
 * Requires 3 endorsers each staking 22 LGM
 */
export interface QualificationEndorsePayload {
  /** Endorser's DID (must be qualified) */
  endorserDid: string;

  /** Target validator's DID */
  targetValidatorDid: string;

  /** Domain for qualification */
  domain: string;

  /** Stake to lock for endorsement */
  stake: bigint;
}

/**
 * Payload for inheriting qualification from lower stratum
 * Weight reduces by 70%^distance
 */
export interface QualificationInheritPayload {
  /** Validator's DID */
  validatorDid: string;

  /** Target domain */
  domain: string;

  /** Source domain/stratum to inherit from */
  sourceQualificationDomain: string;
}

// ============ Capture Challenge Payloads ============

/**
 * Payload for submitting a capture challenge
 *
 * Requires 77 LGM minimum stake
 * 7.7x reward on success + 777 LGM capture exposure bonus
 */
export interface CaptureChallengeSubmitPayload {
  /** Challenger's DID */
  challengerDid: string;

  /** Domain being challenged for capture */
  domain: string;

  /** Challenge stake (minimum 77 LGM) */
  stake: bigint;

  /** Initial evidence (minimum 3 pieces) */
  evidence: Array<{
    type: string;
    description: string;
    validatorIds?: string[];
    dataHash: string;
    proofType: string;
    proof: string;
  }>;
}

/**
 * Payload for adding evidence to a challenge
 */
export interface CaptureChallengeEvidencePayload {
  /** Challenge ID */
  challengeId: string;

  /** Submitter's DID */
  submitterDid: string;

  /** Additional evidence */
  evidence: {
    type: string;
    description: string;
    validatorIds?: string[];
    dataHash: string;
    proofType: string;
    proof: string;
  };
}

/**
 * Payload for resolving a capture challenge
 */
export interface CaptureChallengeResolvePayload {
  /** Challenge ID */
  challengeId: string;

  /** Resolution outcome */
  outcome: "upheld" | "rejected" | "partial";

  /** Justification */
  justification: string;
}

/**
 * Payload for funding a bounty pool
 */
export interface BountyPoolFundPayload {
  /** Domain for bounty pool */
  domain: string;

  /** Funder's DID */
  funderDid: string;

  /** Amount to add */
  amount: bigint;
}

// ============ Reputation Isolation Payloads ============

/**
 * Payload for updating isolated reputation after verification
 */
export interface ReputationUpdatePayload {
  /** Validator's public key */
  validatorId: string;

  /** Domain where verification occurred */
  domain: string;

  /** Whether the verification was correct */
  wasCorrect: boolean;

  /** Verdict confidence */
  verdictConfidence: number;

  /** Final claim confidence */
  finalConfidence: number;
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
  tree:
    | "account"
    | "validator"
    | "knowledge"
    | "research"
    | "evidence"
    | "dispute"
    | "project"
    | "service"
    | "task";
  key: string;
  proof: MerkleProofData;
}

/**
 * Proof of account balance (for light clients)
 */
export interface BalanceProof {
  account: string; // DID
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
  scheduleProcessCount?: number; // DMSS active schedule processes
  contractCount?: number; // Self-invoking contracts
  activeExpirationPeriods?: number; // Token expiration periods

  // Governance metadata
  lipCount?: number; // Total LIPs submitted
  activeLipCount?: number; // LIPs in active stages
  operatorCount?: number; // Registered operators
  activeDeployments?: number; // Deployments in progress

  // Domain Capture Defense metadata
  qualifiedValidatorCount?: number; // Validators with domain qualifications
  activeBountyPoolCount?: number; // Active bounty pools
  activeCaptureChallengCount?: number; // Active capture challenges
  isolatedReputationCount?: number; // Validators with isolated reputation

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
  keepHistoricalStates: number; // Number of block states to keep
  snapshotInterval: number; // Blocks between snapshots

  // Caching
  accountCacheSize: number;
  validatorCacheSize: number;
  factCacheSize: number;

  // Limits
  maxAccountsPerBlock: number;
  maxFactsPerBlock: number;
}

export const DEFAULT_STATE_CONFIG: StateConfig = {
  keepHistoricalStates: 1111,
  snapshotInterval: 111,

  accountCacheSize: 11111,
  validatorCacheSize: 1111,
  factCacheSize: 5555,

  maxAccountsPerBlock: 111,
  maxFactsPerBlock: 11,
};

// ============ Error Types ============

export class StateError extends Error {
  constructor(
    message: string,
    public code: StateErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "StateError";
  }
}

export type StateErrorCode =
  | "ACCOUNT_NOT_FOUND"
  | "INSUFFICIENT_BALANCE"
  | "INVALID_NONCE"
  | "VALIDATOR_NOT_FOUND"
  | "FACT_NOT_FOUND"
  | "DUPLICATE_FACT" // GAP-023 FIX: For uniqueness check
  | "INVALID_TRANSITION"
  | "PROOF_INVALID"
  | "STATE_MISMATCH"
  | "ROLLBACK_FAILED"
  // Research errors
  | "RESEARCH_NOT_FOUND"
  | "BOUNTY_NOT_FOUND"
  | "UNAUTHORIZED"
  // Evidence errors
  | "EVIDENCE_NOT_FOUND"
  | "ARCHIVE_NOT_FOUND"
  | "VERSION_LIMIT_EXCEEDED"
  | "DUPLICATE_EVIDENCE"
  | "FEED_NOT_FOUND"
  | "STALE_EVIDENCE"
  // Dispute errors
  | "DISPUTE_NOT_FOUND"
  | "INVALID_PHASE"
  | "INSUFFICIENT_STAKE"
  | "ALREADY_COMMITTED"
  | "COMMITMENT_MISMATCH"
  | "ALREADY_VOTED"
  | "NOT_ARBITER"
  | "CANNOT_ESCALATE"
  | "DEADLINE_PASSED"
  | "ALREADY_RESOLVED"
  // Vesting errors
  | "VESTING_NOT_FOUND"
  | "VESTING_PAUSED"
  | "VESTING_COMPLETED"
  | "VESTING_FALSIFIED"
  | "CLIFF_NOT_ENDED"
  | "NOTHING_TO_CLAIM"
  | "NOT_RECIPIENT"
  // DMSS Schedule errors
  | "SCHEDULE_NOT_FOUND"
  | "SCHEDULE_PAUSED"
  | "SCHEDULE_COMPLETED"
  | "SCHEDULE_EXHAUSTED"
  | "CONDITION_NOT_MET"
  | "SCHEDULE_FEE_INSUFFICIENT"
  // Expirable token errors
  | "TOKENS_EXPIRED"
  | "PERIOD_NOT_EXPIRED"
  | "RAI_NOT_ELIGIBLE"
  | "DUST_AMOUNT"
  // Contract errors
  | "CONTRACT_NOT_FOUND"
  | "CONTRACT_PAUSED"
  | "INSUFFICIENT_CONTRACT_GAS"
  | "INVOCATION_FAILED"
  | "INVOCATION_DISABLED"
  // Undelegation errors
  | "INSUFFICIENT_DELEGATION"
  | "UNBONDING_NOT_FOUND"
  // Domain proposal errors
  | "DOMAIN_ALREADY_EXISTS"
  | "DOMAIN_PROPOSAL_NOT_FOUND"
  | "DOMAIN_PROPOSAL_EXPIRED"
  | "DOMAIN_PROPOSAL_DECIDED"
  | "PARENT_DOMAIN_NOT_FOUND"
  | "MAX_DOMAIN_CHILDREN"
  | "MAX_DOMAIN_DEPTH"
  | "DOMAIN_PENDING"
  | "INSUFFICIENT_ENDORSEMENTS"

  // Domain Qualification errors
  | "QUALIFICATION_NOT_FOUND"
  | "QUALIFICATION_EXPIRED"
  | "QUALIFICATION_SUSPENDED"
  | "QUALIFICATION_REVOKED"
  | "INSUFFICIENT_TRACK_RECORD"
  | "INSUFFICIENT_ENDORSERS"
  | "NO_INHERITABLE_QUALIFICATION"
  | "QUALIFICATION_ALREADY_EXISTS"

  // Capture Challenge errors
  | "CAPTURE_CHALLENGE_NOT_FOUND"
  | "CAPTURE_CHALLENGE_EXPIRED"
  | "CAPTURE_CHALLENGE_RESOLVED"
  | "INSUFFICIENT_EVIDENCE"
  | "BOUNTY_POOL_NOT_FOUND"
  | "BOUNTY_POOL_INSUFFICIENT"

  // Reputation Isolation errors
  | "REPUTATION_NOT_FOUND"
  | "DOMAIN_REPUTATION_REQUIRED"

  // Claim lifecycle errors
  | "CLAIM_NOT_FOUND"
  | "CLAIM_ALREADY_VERIFYING"
  | "CLAIM_NOT_PENDING"
  | "CLAIM_NOT_VERIFYING"
  | "CLAIM_ALREADY_RESOLVED"
  | "INVALID_REFERENCE"

  // Payment channel errors
  | "INVALID_DEPOSIT"
  | "INVALID_DURATION"
  | "PROVIDER_NOT_FOUND"
  | "CHANNEL_EXISTS"
  | "CHANNEL_NOT_FOUND"
  | "CHANNEL_NOT_OPEN"
  | "INVALID_BALANCE"
  | "INVALID_STATE_HASH"
  | "INVALID_STATUS"
  | "SETTLEMENT_TOO_SOON"
  | "NOT_EXPIRED"
  | "NOT_IN_DISPUTE"
  | "DISPUTE_EXPIRED"
  | "INVALID_CHALLENGE"

  // Account status errors
  | "ACCOUNT_FROZEN"
  | "ACCOUNT_IN_RECOVERY"

  // Tree of Life errors
  | "PROJECT_NOT_FOUND"
  | "SERVICE_NOT_FOUND"
  | "BUDGET_EXCEEDED"
  | "TASK_NOT_FOUND"
  | "INSUFFICIENT_CAPABILITIES"
  | "INVALID_CAPABILITY"
  | "NOT_A_CONTRIBUTOR"
  | "SELF_REVIEW"
  | "INTERNAL_ERROR"
  | "SEED_NOT_FOUND"
  | "SEED_EXPIRED"

  // Governance errors
  | "LIP_NOT_FOUND"
  | "LIP_INVALID_STAGE"
  | "GOVERNANCE_INELIGIBLE"
  | "VOTE_ALREADY_CAST";
