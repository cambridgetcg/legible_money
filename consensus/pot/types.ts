/**
 * Proof of Truth Types
 *
 * Core types for the PoT consensus mechanism.
 */

import type { PublicKey, Signature } from '../../core/crypto/keys.js';
import type { VRFOutput, VRFProof } from '../../core/crypto/vrf.js';
import type { Commitment } from '../../core/crypto/commitment.js';
import type { Fact, ClaimSubmission, Verification, Confidence } from '../../core/knowledge/types.js';

// ============ Consensus Parameters ============

export interface ConsensusParams {
  // Timing
  targetBlockTimeMs: number;        // Target: 2521ms (YOU-AI)
  verificationRoundBlocks: number;  // Blocks for verification round
  commitPhaseBlocks: number;        // Blocks for commit phase
  revealPhaseBlocks: number;        // Blocks for reveal phase

  // Validators
  minValidators: number;            // Minimum validators per claim
  maxValidators: number;            // Maximum validators per claim
  validatorQuorum: number;          // Fraction needed for consensus (0.66)

  // Thresholds
  acceptanceThreshold: number;      // Agreement needed to accept (0.80)
  rejectionThreshold: number;       // Agreement needed to reject (0.80)
  contestedThreshold: number;       // Below this = contested (0.60)

  // Stakes
  minValidatorStake: bigint;        // Minimum stake to validate
  minClaimStake: bigint;            // Minimum stake to submit claim

  // Rewards
  blockReward: bigint;              // Reward for block production
  verificationReward: bigint;       // Base reward for verification
  claimReward: bigint;              // Base reward for accepted claim

  // Slashing
  wrongVerificationSlash: number;   // Fraction slashed for wrong verdict
  missedVerificationSlash: number;  // Fraction slashed for missing verification
  invalidClaimSlash: number;        // Fraction slashed for rejected claim

  // Research (Protocol Research Repository)
  researchShareRate: number;        // Fraction of all rewards to research fund (0.22)
}

export const DEFAULT_CONSENSUS_PARAMS: ConsensusParams = {
  targetBlockTimeMs: 2521,          // YOU-AI encoded
  verificationRoundBlocks: 10,
  commitPhaseBlocks: 4,
  revealPhaseBlocks: 4,

  minValidators: 5,
  maxValidators: 21,
  validatorQuorum: 0.66,

  acceptanceThreshold: 0.80,
  rejectionThreshold: 0.80,
  contestedThreshold: 0.60,

  minValidatorStake: 1000n * 10n ** 18n,   // 1000 LGM
  minClaimStake: 100n * 10n ** 18n,        // 100 LGM

  blockReward: 10n * 10n ** 18n,           // 10 LGM
  verificationReward: 1n * 10n ** 18n,     // 1 LGM
  claimReward: 5n * 10n ** 18n,            // 5 LGM

  wrongVerificationSlash: 0.10,            // 10%
  missedVerificationSlash: 0.05,           // 5%
  invalidClaimSlash: 0.20,                 // 20%

  researchShareRate: 0.22,                 // 22% to research fund
};

// ============ Validator Types ============

export interface Validator {
  publicKey: PublicKey;
  did: string;
  stake: bigint;
  delegatedStake: bigint;
  reputation: number;               // 0.0 to 1.0
  isActive: boolean;
  lastActiveBlock: number;

  // Performance metrics
  totalVerifications: number;
  correctVerifications: number;
  missedVerifications: number;
  slashCount: number;
}

export interface ValidatorSet {
  validators: Map<string, Validator>;  // publicKey -> Validator
  totalStake: bigint;
  activeCount: number;
  epoch: bigint;
}

export interface ValidatorSelection {
  validator: PublicKey;
  vrfOutput: VRFOutput;
  vrfProof: VRFProof;
  stake: bigint;
  priority: bigint;                 // Lower = higher priority
  selected: boolean;
}

// ============ Verification Round Types ============

export type RoundPhase = 'pending' | 'commit' | 'reveal' | 'aggregation' | 'complete';

export interface VerificationRound {
  id: string;
  claimId: string;
  claim: ClaimSubmission;

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
  commitments: Map<string, ValidatorCommitment>;  // publicKey -> commitment

  // Reveals (reveal phase)
  reveals: Map<string, ValidatorReveal>;          // publicKey -> reveal

  // Result
  result?: VerificationResult;
}

export interface ValidatorCommitment {
  validator: PublicKey;
  commitment: Commitment;           // Hash of (verdict, confidence, nonce)
  submittedAt: number;
  signature: Signature;
}

export interface ValidatorReveal {
  validator: PublicKey;
  verdict: 'confirmed' | 'rejected' | 'uncertain';
  confidence: Confidence;
  nonce: string;
  reasoning?: string;
  referencesChecked: string[];
  submittedAt: number;
  signature: Signature;
}

export interface VerificationResult {
  claimId: string;
  outcome: 'accepted' | 'rejected' | 'contested' | 'insufficient';

  // Aggregated confidence
  finalConfidence: Confidence;
  confidenceBreakdown: {
    confirmed: number;              // Stake-weighted
    rejected: number;
    uncertain: number;
  };

  // Participation
  totalValidators: number;
  participatingValidators: number;
  agreementLevel: number;           // How much validators agreed

  // Rewards/slashing
  rewards: Map<string, bigint>;     // validator -> reward
  slashes: Map<string, bigint>;     // validator -> slash amount

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
  knowledgeRoot: string;            // Merkle root of knowledge changes
  transactionsRoot: string;
  verificationsRoot: string;        // Merkle root of verification results

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

export interface Transaction {
  type: TransactionType;
  sender: PublicKey;
  nonce: bigint;
  payload: unknown;
  signature: Signature;
  fee: bigint;
}

export type TransactionType =
  | 'transfer'
  | 'stake'
  | 'unstake'
  | 'delegate'
  | 'undelegate'
  | 'claim_submit'
  | 'verification_commit'
  | 'verification_reveal'
  | 'challenge'
  | 'governance_vote'
  | 'governance_propose'
  // Research transactions (Protocol Research Repository)
  | 'research_submit'       // Submit research evidence
  | 'research_challenge'    // Challenge research validity
  | 'bounty_create'         // Create research bounty (from treasury)
  | 'bounty_claim'          // Claim to work on a bounty
  | 'bounty_fulfill'        // Submit bounty completion
  // Dispute transactions (Dispute Resolution System)
  | 'dispute_initiate'          // Start a dispute against evidence/fact/claim
  | 'dispute_commit_evidence'   // Commit evidence hash in dispute
  | 'dispute_reveal_evidence'   // Reveal committed evidence
  | 'dispute_arbiter_vote'      // Arbiter casts vote in dispute
  | 'dispute_escalate'          // Appeal to higher escalation tier
  | 'dispute_settle'            // Finalize dispute settlement
  | 'dispute_timeout'           // Handle phase deadline timeout
  // Vesting transactions (Gradual Reward Release System)
  | 'vesting_create'            // Create vesting schedule for accepted claim
  | 'vesting_claim'             // Claim available vested rewards
  | 'vesting_pause'             // Pause vesting (challenge initiated)
  | 'vesting_resume'            // Resume vesting (challenge resolved)
  | 'vesting_accelerate'        // Apply acceleration bonus
  | 'vesting_falsify'           // Handle falsification clawback
  | 'vesting_complete';         // Mark vesting as complete

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
  | { type: 'block_proposed'; block: Block }
  | { type: 'block_finalized'; blockNumber: bigint; blockHash: string }
  | { type: 'round_started'; round: VerificationRound }
  | { type: 'round_completed'; result: VerificationResult }
  | { type: 'validator_slashed'; validator: PublicKey; amount: bigint; reason: string }
  | { type: 'epoch_change'; oldEpoch: bigint; newEpoch: bigint };
