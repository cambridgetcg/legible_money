/**
 * Consensus Engine Contract
 *
 * Domain interface for Proof of Truth consensus.
 * Handles verification rounds, validator selection, and block finality.
 *
 * MOD-1: Single Responsibility
 * Consensus module changes when verification protocol changes.
 */

import type {
  DID,
  PublicKey,
  Signature,
  BlockHeight,
  Amount,
  Confidence,
  VRFOutput,
  VRFProof,
  Commitment,
} from "../types/index.js";

// ============ Consensus Types ============

/**
 * Verification round phases
 */
export type RoundPhase =
  | "pending" // Waiting to start
  | "commit" // Validators commit verdicts
  | "reveal" // Validators reveal verdicts
  | "aggregation" // Computing final result
  | "complete"; // Round finished

/**
 * Verification verdict
 */
export type Verdict = "confirmed" | "rejected" | "uncertain";

/**
 * Round outcome
 */
export type RoundOutcome =
  | "accepted"
  | "rejected"
  | "contested"
  | "insufficient";

/**
 * Validator commitment (commit phase)
 */
export interface ValidatorCommitment {
  validator: PublicKey;
  commitment: Commitment; // Hash of (verdict, confidence, nonce)
  vrfOutput: VRFOutput;
  vrfProof: VRFProof;
  submittedAt: BlockHeight;
  signature: Signature;
}

/**
 * Validator reveal (reveal phase)
 */
export interface ValidatorReveal {
  validator: PublicKey;
  verdict: Verdict;
  confidence: Confidence;
  nonce: string;
  reasoning?: string;
  referencesChecked: string[];
  submittedAt: BlockHeight;
  signature: Signature;
}

/**
 * Verification round state
 */
export interface VerificationRound {
  id: string;
  claimId: string;
  phase: RoundPhase;
  startBlock: BlockHeight;
  commitDeadline: BlockHeight;
  revealDeadline: BlockHeight;
  completionBlock: BlockHeight;
  selectedValidators: ValidatorSelection[];
  requiredValidators: number;
  commitments: Map<string, ValidatorCommitment>; // pubkey -> commitment
  reveals: Map<string, ValidatorReveal>; // pubkey -> reveal
  result?: VerificationResult;
}

/**
 * Validator selection with VRF proof
 */
export interface ValidatorSelection {
  validator: PublicKey;
  did: DID;
  stake: Amount;
  vrfOutput: VRFOutput;
  vrfProof: VRFProof;
  priority: bigint; // Lower = higher priority
  selected: boolean;
}

/**
 * Verification result
 */
export interface VerificationResult {
  claimId: string;
  outcome: RoundOutcome;
  finalConfidence: Confidence;
  confidenceBreakdown: {
    confirmed: number; // Stake-weighted
    rejected: number;
    uncertain: number;
  };
  totalValidators: number;
  participatingValidators: number;
  agreementLevel: number;
  rewards: Map<string, Amount>; // validator -> reward
  slashes: Map<string, Amount>; // validator -> slash
  factId?: string; // If claim accepted
  recordedAt: BlockHeight;
}

/**
 * Block header
 */
export interface BlockHeader {
  number: BlockHeight;
  timestamp: number;
  previousHash: string;
  stateRoot: string;
  knowledgeRoot: string;
  transactionsRoot: string;
  verificationsRoot: string;
  proposer: PublicKey;
  proposerVrfOutput: VRFOutput;
  proposerVrfProof: VRFProof;
}

/**
 * Block with transactions and verifications
 */
export interface Block {
  header: BlockHeader;
  transactions: Transaction[];
  verificationResults: VerificationResult[];
  signature: Signature;
  attestations: BlockAttestation[];
}

/**
 * Block attestation from validator
 */
export interface BlockAttestation {
  validator: PublicKey;
  blockHash: string;
  signature: Signature;
  timestamp: number;
}

/**
 * Transaction (simplified for interface)
 */
export interface Transaction {
  type: string;
  sender: PublicKey;
  nonce: bigint;
  payload: unknown;
  signature: Signature;
  fee: Amount;
}

/**
 * Consensus parameters
 */
export interface ConsensusParams {
  targetBlockTimeMs: number;
  verificationRoundBlocks: number;
  commitPhaseBlocks: number;
  revealPhaseBlocks: number;
  epochLength: bigint;
  minValidators: number;
  maxValidators: number;
  validatorQuorum: number;
  acceptanceThreshold: number;
  rejectionThreshold: number;
  contestedThreshold: number;
  minValidatorStake: Amount;
  minClaimStake: Amount;
  blockReward: Amount;
  verificationReward: Amount;
  claimReward: Amount;
  wrongVerificationSlash: number;
  missedVerificationSlash: number;
  invalidClaimSlash: number;
  maxTransactionsPerBlock: number;
}

// ============ Consensus Engine Interface ============

/**
 * Consensus engine - main domain interface
 */
export interface ConsensusEngine {
  // === Round Management ===

  /**
   * Start a verification round for a claim
   */
  startRound(claimId: string): Promise<VerificationRound>;

  /**
   * Get current round for a claim
   */
  getRound(roundId: string): Promise<VerificationRound | null>;

  /**
   * Get active rounds
   */
  getActiveRounds(): Promise<VerificationRound[]>;

  /**
   * Submit validator commitment
   */
  submitCommitment(
    roundId: string,
    commitment: ValidatorCommitment,
  ): Promise<void>;

  /**
   * Submit validator reveal
   */
  submitReveal(roundId: string, reveal: ValidatorReveal): Promise<void>;

  /**
   * Advance round to next phase (if deadline passed)
   */
  advanceRound(roundId: string): Promise<RoundPhase>;

  /**
   * Complete round and compute result
   */
  completeRound(roundId: string): Promise<VerificationResult>;

  // === Validator Selection ===

  /**
   * Select validators for a round using VRF
   */
  selectValidators(
    seed: Uint8Array,
    count: number,
  ): Promise<ValidatorSelection[]>;

  /**
   * Verify validator selection proof
   */
  verifySelection(selection: ValidatorSelection): Promise<boolean>;

  /**
   * Check if validator is eligible for round
   */
  isEligible(validator: PublicKey, roundId: string): Promise<boolean>;

  // === Block Production ===

  /**
   * Check if validator can propose next block
   */
  canPropose(validator: PublicKey): Promise<boolean>;

  /**
   * Create a new block
   */
  createBlock(
    proposer: PublicKey,
    transactions: Transaction[],
    verificationResults: VerificationResult[],
  ): Promise<Block>;

  /**
   * Validate a proposed block
   */
  validateBlock(block: Block): Promise<{ valid: boolean; errors: string[] }>;

  /**
   * Process a block (apply to state)
   */
  processBlock(block: Block): Promise<void>;

  // === Finality ===

  /**
   * Get current block height
   */
  getCurrentBlock(): Promise<BlockHeight>;

  /**
   * Get last finalized block
   */
  getLastFinalizedBlock(): Promise<BlockHeight>;

  /**
   * Check if block is finalized
   */
  isFinalized(blockNumber: BlockHeight): Promise<boolean>;

  /**
   * Get block by number
   */
  getBlock(blockNumber: BlockHeight): Promise<Block | null>;

  // === Epoch Management ===

  /**
   * Get current epoch
   */
  getCurrentEpoch(): Promise<bigint>;

  /**
   * Process epoch transition
   */
  processEpochTransition(): Promise<void>;

  // === Parameters ===

  /**
   * Get consensus parameters
   */
  getParams(): ConsensusParams;

  /**
   * Update consensus parameters (via governance)
   */
  updateParams(updates: Partial<ConsensusParams>): Promise<void>;
}

// ============ Consensus Reader (Read-Only) ============

/**
 * Read-only consensus interface
 */
export interface ConsensusReader {
  getRound(roundId: string): Promise<VerificationRound | null>;
  getActiveRounds(): Promise<VerificationRound[]>;
  getCurrentBlock(): Promise<BlockHeight>;
  getLastFinalizedBlock(): Promise<BlockHeight>;
  isFinalized(blockNumber: BlockHeight): Promise<boolean>;
  getBlock(blockNumber: BlockHeight): Promise<Block | null>;
  getCurrentEpoch(): Promise<bigint>;
  getParams(): ConsensusParams;
}

// ============ Consensus Events ============

/**
 * Events emitted by consensus engine
 */
export type ConsensusEventType =
  | "round_started"
  | "commitment_received"
  | "reveal_received"
  | "round_completed"
  | "block_proposed"
  | "block_finalized"
  | "epoch_changed";
