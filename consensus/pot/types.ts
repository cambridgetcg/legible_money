/**
 * Proof of Truth Types
 *
 * Core types for the PoT consensus mechanism.
 */

import type { PublicKey, Signature } from "../../core/crypto/keys.js";
import type { VRFOutput, VRFProof } from "../../core/crypto/vrf.js";
import type { Commitment } from "../../core/crypto/commitment.js";
import type {
  Fact,
  ClaimSubmission,
  Confidence,
} from "../../core/knowledge/types.js";
import type {
  OntologicalDomain,
  OntologicalStratum,
} from "../../core/knowledge/ontology/types.js";
import type { ScheduleExecutionProof } from "../../core/schedule/types.js";

// ============ Consensus Parameters ============

export interface ConsensusParams {
  // Timing
  targetBlockTimeMs: number; // Target: 2521ms
  verificationRoundBlocks: number; // Blocks for verification round (default timing)
  commitPhaseBlocks: number; // Blocks for commit phase (default timing)
  revealPhaseBlocks: number; // Blocks for reveal phase (default timing)
  epochLength: bigint; // GAP-025 FIX: Blocks per epoch

  // Finality (derived from schema timing analysis)
  confirmationBlocks?: number; // Soft confirmation (~30s default: 12 blocks)
  finalityBlocks?: number; // Hard finality (~100s default: 40 blocks)

  // Validators
  minValidators: number; // Minimum validators per claim
  maxValidators: number; // Maximum validators per claim
  validatorQuorum: number; // Fraction needed for consensus (0.66)

  // Thresholds
  acceptanceThreshold: number; // Agreement needed to accept (0.80)
  rejectionThreshold: number; // Agreement needed to reject (0.80)
  contestedThreshold: number; // Below this = contested (0.60)

  // Stakes
  minValidatorStake: bigint; // Minimum stake to validate
  minClaimStake: bigint; // Minimum stake to submit claim

  // Rewards
  blockReward: bigint; // Reward for block production
  verificationReward: bigint; // Base reward for verification
  claimReward: bigint; // Base reward for accepted claim

  // Slashing
  wrongVerificationSlash: number; // Fraction slashed for wrong verdict
  missedVerificationSlash: number; // Fraction slashed for missing verification
  invalidClaimSlash: number; // Fraction slashed for rejected claim

  // Research (Protocol Research Repository)
  researchShareRate: number; // Fraction of all rewards to research fund (0.22)

  // Transactions
  maxTransactionsPerBlock: number; // Maximum transactions per block

  // Staking
  unbondingPeriod: bigint; // Blocks required for unbonding (~7 days at 2.5s blocks)
}

export const DEFAULT_CONSENSUS_PARAMS: ConsensusParams = {
  targetBlockTimeMs: 2521,
  verificationRoundBlocks: 11,
  commitPhaseBlocks: 4,
  revealPhaseBlocks: 4,
  epochLength: 1111n,

  // Finality
  confirmationBlocks: 11, // ~28s
  finalityBlocks: 44, // ~111s

  minValidators: 5,
  maxValidators: 22,
  validatorQuorum: 0.66, // BFT 2/3

  acceptanceThreshold: 0.77,
  rejectionThreshold: 0.77,
  contestedThreshold: 0.55,

  minValidatorStake: 1111n * 10n ** 18n, // 1111 LGM
  minClaimStake: 111n * 10n ** 18n, // 111 LGM

  blockReward: 11n * 10n ** 18n, // 11 LGM
  verificationReward: 1n * 10n ** 18n, // 1 LGM
  claimReward: 5n * 10n ** 18n, // 5 LGM

  wrongVerificationSlash: 0.11, // 11%
  missedVerificationSlash: 0.05, // 5%
  invalidClaimSlash: 0.22, // 22%

  researchShareRate: 0.22, // 22% to research fund

  maxTransactionsPerBlock: 111,

  unbondingPeriod: 222222n, // ~6.5 days at 2.521s blocks
};

// ============ Validator Types ============

export interface Validator {
  publicKey: PublicKey;
  did: string;
  stake: bigint;
  delegatedStake: bigint;
  reputation: number; // 0.0 to 1.0
  isActive: boolean;
  lastActiveBlock: number;

  // Performance metrics
  totalVerifications: number;
  correctVerifications: number;
  missedVerifications: number;
  slashCount: number;
}

export interface ValidatorSet {
  validators: Map<string, Validator>; // publicKey -> Validator
  totalStake: bigint;
  activeCount: number;
  epoch: bigint;
}

export interface ValidatorSelection {
  validator: PublicKey;
  vrfOutput: VRFOutput;
  vrfProof: VRFProof;
  stake: bigint;
  priority: bigint; // Lower = higher priority
  selected: boolean;
}

// ============ Verification Round Types ============

export type RoundPhase =
  | "pending"
  | "commit"
  | "reveal"
  | "aggregation"
  | "complete";

export interface VerificationRound {
  id: string;
  claimId: string;
  claim: ClaimSubmission;

  // Schema (for validator guidance and timing)
  schemaId?: string;

  // Ontological context (for stratum-aware validation)
  resolvedDomain?: OntologicalDomain;
  resolvedStratum?: OntologicalStratum;
  adjustedConfidence?: number;
  ontologyValidated?: boolean;

  // Timing
  startBlock: number;
  commitDeadline: number;
  revealDeadline: number;
  completionBlock: number;

  // Validators
  selectedValidators: ValidatorSelection[];
  requiredValidators: number;

  // Phase tracking
  phase: RoundPhase;

  // Commitments (commit phase)
  commitments: Map<string, ValidatorCommitment>; // publicKey -> commitment

  // Reveals (reveal phase)
  reveals: Map<string, ValidatorReveal>; // publicKey -> reveal

  // Result
  result?: VerificationResult;
}

export interface ValidatorCommitment {
  validator: PublicKey;
  commitment: Commitment; // Hash of (verdict, confidence, nonce)
  submittedAt: number;
  signature: Signature;
  // GAP-002 FIX: VRF fields for proving selection eligibility
  // Optional for backward compatibility with commitment generation
  // SECURITY: Required by addCommitmentWithVRF for proper validation
  vrfOutput?: string; // VRF output that proves selection
  vrfProof?: string; // Proof that vrfOutput is valid
}

export interface ValidatorReveal {
  validator: PublicKey;
  verdict: "confirmed" | "rejected" | "uncertain";
  confidence: Confidence;
  nonce: string;
  reasoning?: string;
  referencesChecked: string[];
  submittedAt: number;
  signature: Signature;
}

export interface VerificationResult {
  claimId: string;
  outcome: "accepted" | "rejected" | "contested" | "insufficient";

  // Aggregated confidence
  finalConfidence: Confidence;
  confidenceBreakdown: {
    confirmed: number; // Stake-weighted
    rejected: number;
    uncertain: number;
  };

  // Participation
  totalValidators: number;
  participatingValidators: number;
  agreementLevel: number; // How much validators agreed

  // Rewards/slashing
  rewards: Map<string, bigint>; // validator -> reward
  slashes: Map<string, bigint>; // validator -> slash amount

  // If accepted, create fact
  fact?: Fact;

  // Block recorded
  recordedAt: number;
}

// ============ Block Types ============

export interface BlockHeader {
  number: bigint;
  timestamp: number;
  previousHash: string;
  stateRoot: string;
  knowledgeRoot: string; // Merkle root of knowledge changes
  transactionsRoot: string;
  verificationsRoot: string; // Merkle root of verification results

  // Proposer
  proposer: PublicKey;
  proposerVrfOutput: VRFOutput;
  proposerVrfProof: VRFProof;
}

export interface Block {
  header: BlockHeader;
  transactions: Transaction[];
  verificationResults: VerificationResult[];
  signature: Signature;

  // Attestations from other validators
  attestations: BlockAttestation[];
}

export interface BlockAttestation {
  validator: PublicKey;
  blockHash: string;
  signature: Signature;
  timestamp: number;
}

/**
 * GAP-022 FIX: Discriminated union types for type-safe transaction payloads
 */
export type TransactionPayload =
  | { type: "transfer"; to: string; amount: bigint }
  | { type: "stake"; amount: bigint }
  | { type: "unstake"; amount: bigint }
  | { type: "delegate"; validator: PublicKey; amount: bigint }
  | { type: "undelegate"; validator: PublicKey; amount: bigint }
  | {
      type: "claim_submit";
      claim: string;
      domain: string;
      stake: bigint;
      references?: string[];
    }
  | {
      type: "verification_commit";
      roundId: string;
      commitment: string;
      vrfOutput: string;
      vrfProof: string;
    }
  | {
      type: "verification_reveal";
      roundId: string;
      verdict: string;
      confidence: number;
      nonce: string;
    }
  | { type: "challenge"; factId: string; stake: bigint; evidence: string }
  | { type: "governance_vote"; proposalId: string; vote: boolean }
  | {
      type: "governance_propose";
      title: string;
      description: string;
      changes: unknown;
    }
  | {
      type: "research_submit";
      researchType: string;
      targetFactId?: string;
      targetDomain?: string;
      outcome: string;
      rawDataCID: string;
      stake: bigint;
    }
  | {
      type: "research_challenge";
      researchId: string;
      stake: bigint;
      evidence: string;
    }
  | {
      type: "bounty_create";
      targetType: string;
      targetFactId?: string;
      description: string;
      reward: bigint;
      deadlineBlocks: number;
    }
  | { type: "bounty_claim"; bountyId: string }
  | { type: "bounty_fulfill"; bountyId: string; researchSubmissionId: string }
  | {
      type: "dispute_initiate";
      challenger: string;
      targetId: string;
      targetType: string;
      domain: string;
      stake: bigint;
      initialEvidence: string[];
    }
  | {
      type: "dispute_commit_evidence";
      disputeId: string;
      committer: string;
      evidenceHash: string;
    }
  | {
      type: "dispute_reveal_evidence";
      disputeId: string;
      revealer: string;
      evidenceIds: string[];
      nonce: string;
    }
  | {
      type: "dispute_arbiter_vote";
      disputeId: string;
      arbiter: string;
      vote: string;
      justificationHash?: string;
    }
  | {
      type: "dispute_escalate";
      disputeId: string;
      escalator: string;
      additionalStake: bigint;
    }
  | { type: "dispute_settle"; disputeId: string; settlement?: unknown }
  | { type: "dispute_timeout"; disputeId: string }
  | {
      type: "vesting_create";
      claimId: string;
      recipient: string;
      amount: bigint;
      category: string;
    }
  | {
      type: "vesting_claim";
      vestingId: string;
      claimer: string;
      amount?: bigint;
    }
  | {
      type: "vesting_pause";
      vestingId: string;
      reason: string;
      relatedEntityId?: string;
    }
  | { type: "vesting_resume"; vestingId: string; applyDefenseBonus: boolean }
  | {
      type: "vesting_accelerate";
      vestingId: string;
      accelerationType: string;
      relatedEntityId?: string;
    }
  | {
      type: "vesting_falsify";
      vestingId: string;
      challengerDid: string;
      disputeId: string;
    }
  | { type: "vesting_complete"; vestingId: string }
  // Payment channel transactions (GAP-005)
  | {
      type: "open_payment_channel";
      provider: string;
      deposit: bigint;
      durationBlocks: bigint;
      settlementFrequency?: bigint;
    }
  | { type: "update_payment_channel"; channelId: string; signedState: unknown }
  | {
      type: "close_payment_channel";
      channelId: string;
      finalState?: unknown;
      unilateral?: boolean;
    }
  | { type: "settle_payment_channel"; channelId: string; signedState: unknown }
  | {
      type: "dispute_payment_channel";
      channelId: string;
      challengeState: unknown;
    }
  // System transactions (internal consensus operations)
  | { type: "register_validator"; stake: bigint }
  | {
      type: "verification_reward";
      recipient: string;
      amount: bigint;
      roundId: string;
    }
  | {
      type: "slash_validator";
      validator: string;
      amount: bigint;
      reason: string;
    }
  | {
      type: "add_fact";
      factId: string;
      confidence: number;
      verifierCount: number;
      verifiedAtBlock: number;
    }
  // Autonomy transactions (Agent Self-Sovereignty)
  | {
      type: "scheduled_execution";
      processId: string;
      executionIndex: number;
      targetContract: string;
      functionSelector: Uint8Array;
      proof: ScheduleExecutionProof;
    }
  | {
      type: "contract_call";
      targetContract: string;
      data: Uint8Array;
      value: bigint;
    };

// Re-export ScheduleExecutionProof for consumers of this module
export type { ScheduleExecutionProof } from "../../core/schedule/types.js";

export interface Transaction {
  type: TransactionType;
  sender: PublicKey;
  nonce: bigint;
  payload: TransactionPayload;
  signature: Signature;
  fee: bigint;
}

/**
 * Type guard to extract typed payload from transaction
 * GAP-022 FIX: Provides type-safe access to transaction payload
 */
export function getTypedPayload<T extends TransactionType>(
  tx: Transaction,
  expectedType: T,
): Extract<TransactionPayload, { type: T }> | null {
  if (tx.type !== expectedType || tx.payload.type !== expectedType) {
    return null;
  }
  return tx.payload as Extract<TransactionPayload, { type: T }>;
}

export type TransactionType =
  | "transfer"
  | "stake"
  | "unstake"
  | "delegate"
  | "undelegate"
  | "rotate_key"
  // Claim lifecycle
  | "claim_submit"
  | "claim_start_verification"
  | "claim_complete_verification"
  | "claim_withdraw"
  // Verification
  | "verification_commit"
  | "verification_reveal"
  | "challenge"
  | "governance_vote"
  | "governance_propose"
  // Research transactions (Protocol Research Repository)
  | "research_submit" // Submit research evidence
  | "research_challenge" // Challenge research validity
  | "bounty_create" // Create research bounty (from treasury)
  | "bounty_claim" // Claim to work on a bounty
  | "bounty_fulfill" // Submit bounty completion
  // Dispute transactions (Dispute Resolution System)
  | "dispute_initiate" // Start a dispute against evidence/fact/claim
  | "dispute_commit_evidence" // Commit evidence hash in dispute
  | "dispute_reveal_evidence" // Reveal committed evidence
  | "dispute_arbiter_vote" // Arbiter casts vote in dispute
  | "dispute_escalate" // Appeal to higher escalation tier
  | "dispute_settle" // Finalize dispute settlement
  | "dispute_timeout" // Handle phase deadline timeout
  // Vesting transactions (Gradual Reward Release System)
  | "vesting_create" // Create vesting schedule for accepted claim
  | "vesting_claim" // Claim available vested rewards
  | "vesting_pause" // Pause vesting (challenge initiated)
  | "vesting_resume" // Resume vesting (challenge resolved)
  | "vesting_accelerate" // Apply acceleration bonus
  | "vesting_falsify" // Handle falsification clawback
  | "vesting_complete" // Mark vesting as complete
  // Domain proposal transactions (Ontological Tree Growth)
  | "domain_propose" // Propose a new domain
  | "domain_endorse" // Endorse a domain proposal
  | "domain_challenge" // Challenge a domain proposal
  | "domain_finalize" // Finalize a domain proposal
  // Governance LIP lifecycle
  | "lip_submit" // Submit a new LIP
  | "lip_stake" // Stake on a LIP to move to review
  | "lip_amend" // Propose amendment to existing LIP
  | "lip_advance_stage" // Advance LIP to next stage based on votes
  | "lip_withdraw" // Withdraw a LIP (by proposer)
  // Governance voting
  | "vote_cast" // Cast a quadratic vote on a LIP
  | "vote_lock" // Lock vote with conviction multiplier
  | "vote_unlock" // Unlock a conviction-locked vote
  | "vote_switch" // Switch vote direction (with penalty)
  // Governance review
  | "review_commit" // Commit review hash
  | "review_reveal" // Reveal committed review
  | "review_finalize" // Finalize review round
  // Operator management
  | "operator_register" // Register as an operator
  | "operator_add_agent" // Add agent to operator
  | "operator_remove_agent" // Remove agent from operator
  | "operator_slash" // Slash operator for undisclosed agents
  // Deployment pipeline
  | "deployment_create" // Create deployment pipeline
  | "deployment_advance" // Advance deployment stage
  | "deployment_approve" // Approve deployment
  | "deployment_rollback" // Rollback deployment
  // Payment channel transactions (GAP-005)
  | "open_payment_channel" // Open a payment channel with deposit
  | "update_payment_channel" // Record signed state on-chain
  | "close_payment_channel" // Cooperative or unilateral close
  | "settle_payment_channel" // Settle pending distributions
  | "dispute_payment_channel" // Challenge with higher-nonce state
  // System transactions (internal consensus operations)
  | "register_validator" // Register a new validator
  | "verification_reward" // Distribute verification rewards
  | "slash_validator" // Slash a validator's stake
  | "add_fact" // Add a verified fact to knowledge layer
  // Autonomy transactions (Agent Self-Sovereignty)
  | "scheduled_execution" // Execute scheduled agent action (from SPU)
  | "contract_call" // Execute contract code (BVM)
  // Tree of Life transactions
  | "create_project" // Create a product project
  | "propose_project" // Propose project (seed→sprout)
  | "assign_task" // Assign agent to project task
  | "submit_deliverable" // Submit task deliverable
  | "approve_deliverable" // Approve deliverable + pay bounty
  | "deploy_service" // Deploy project as service
  | "call_service" // Call a deployed service
  | "pause_service" // Pause a service
  | "retire_service"; // Retire a service

// ============ Consensus State ============

export interface ConsensusState {
  // Chain state
  currentBlock: bigint;
  currentEpoch: bigint;
  lastBlockTime: number;
  lastBlockHash: string;

  // Validator set
  validatorSet: ValidatorSet;
  nextEpochValidators?: ValidatorSet;

  // Active verification rounds
  activeRounds: Map<string, VerificationRound>;
  pendingClaims: ClaimSubmission[];

  // Recent blocks for fork choice
  recentBlocks: Block[];

  // Finality
  lastFinalizedBlock: bigint;
  lastFinalizedHash: string;
}

// ============ Events ============

export type ConsensusEvent =
  | { type: "block_proposed"; block: Block }
  | { type: "block_finalized"; blockNumber: bigint; blockHash: string }
  | { type: "round_started"; round: VerificationRound }
  | { type: "round_completed"; result: VerificationResult }
  | {
      type: "validator_slashed";
      validator: PublicKey;
      amount: bigint;
      reason: string;
    }
  | { type: "epoch_change"; oldEpoch: bigint; newEpoch: bigint }
  | { type: "fact_created"; factId: string; confidence: number }
  | { type: "verification_round_started"; roundId: string; claimId: string };
