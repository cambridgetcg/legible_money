/**
 * Finality
 *
 * When is a block considered final (irreversible)?
 * When is a fact considered permanently recorded?
 *
 * Legible Money uses probabilistic finality with a configurable
 * confirmation depth, similar to how Bitcoin gains finality
 * through block confirmations.
 */

import type { Block, BlockHeader, ConsensusState, ValidatorSet } from './types.js';
import { sha512_256Hex } from '../../core/crypto/hashing.js';

// ============ Finality Parameters ============

export const FINALITY_PARAMS = {
  // Block finality
  CONFIRMATION_DEPTH: 19,           // Blocks needed for finality (1-9 = AI encoded)
  FAST_FINALITY_THRESHOLD: 0.67,    // Stake needed for fast finality

  // Checkpoint frequency
  CHECKPOINT_INTERVAL: 1000,        // Blocks between checkpoints
  CHECKPOINT_SIGNATURES_REQUIRED: 0.67, // Stake needed to sign checkpoint

  // Fork choice
  MAX_REORG_DEPTH: 100,             // Maximum reorg depth allowed
  FORK_CHOICE_RULE: 'heaviest_stake', // How to choose between forks
};

// ============ Block Finality ============

/**
 * Check if a block is finalized
 */
export function isBlockFinalized(
  blockNumber: bigint,
  currentBlock: bigint,
  checkpoints: Checkpoint[]
): boolean {
  // Check against checkpoints first
  const latestCheckpoint = checkpoints.length > 0
    ? checkpoints[checkpoints.length - 1]
    : null;

  if (latestCheckpoint && blockNumber <= latestCheckpoint.blockNumber) {
    return true;
  }

  // Check confirmation depth
  const confirmations = currentBlock - blockNumber;
  return confirmations >= BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH);
}

/**
 * Get number of confirmations for a block
 */
export function getConfirmations(
  blockNumber: bigint,
  currentBlock: bigint
): bigint {
  if (blockNumber > currentBlock) return 0n;
  return currentBlock - blockNumber;
}

/**
 * Calculate finality probability based on confirmations
 *
 * Assumes a simple model where each confirmation
 * exponentially decreases reorg probability.
 */
export function calculateFinalityProbability(
  confirmations: bigint,
  byzantineStakeFraction: number = 0.33
): number {
  // Probability of reorg decreases exponentially
  // P(reorg) ≈ byzantineStakeFraction ^ confirmations
  const reorgProb = Math.pow(byzantineStakeFraction, Number(confirmations));
  return 1 - reorgProb;
}

// ============ Checkpoints ============

export interface Checkpoint {
  blockNumber: bigint;
  blockHash: string;
  stateRoot: string;
  knowledgeRoot: string;
  signatures: CheckpointSignature[];
  totalSigningStake: bigint;
  createdAt: number;
}

export interface CheckpointSignature {
  validator: string;
  signature: string;
  stake: bigint;
}

/**
 * Create a new checkpoint
 */
export function createCheckpoint(
  block: Block,
  signatures: CheckpointSignature[]
): Checkpoint {
  const totalStake = signatures.reduce((sum, s) => sum + s.stake, 0n);

  return {
    blockNumber: block.header.number,
    blockHash: hashBlockHeader(block.header),
    stateRoot: block.header.stateRoot,
    knowledgeRoot: block.header.knowledgeRoot,
    signatures,
    totalSigningStake: totalStake,
    createdAt: Date.now(),
  };
}

/**
 * Verify a checkpoint has sufficient stake
 */
export function verifyCheckpoint(
  checkpoint: Checkpoint,
  validatorSet: ValidatorSet
): { valid: boolean; reason?: string } {
  // Check stake threshold
  const requiredStake = (validatorSet.totalStake *
    BigInt(Math.floor(FINALITY_PARAMS.CHECKPOINT_SIGNATURES_REQUIRED * 10000))) / 10000n;

  if (checkpoint.totalSigningStake < requiredStake) {
    return {
      valid: false,
      reason: `Insufficient stake: ${checkpoint.totalSigningStake} < ${requiredStake}`,
    };
  }

  // Verify each signature
  for (const sig of checkpoint.signatures) {
    const validator = validatorSet.validators.get(sig.validator);
    if (!validator) {
      return { valid: false, reason: `Unknown validator: ${sig.validator}` };
    }
    // In production, verify actual signature
  }

  return { valid: true };
}

/**
 * Check if a new checkpoint should be created
 */
export function shouldCreateCheckpoint(
  currentBlock: bigint,
  lastCheckpointBlock: bigint
): boolean {
  return currentBlock - lastCheckpointBlock >= BigInt(FINALITY_PARAMS.CHECKPOINT_INTERVAL);
}

// ============ Fork Choice ============

export interface ForkChoice {
  bestBlock: Block;
  bestBlockHash: string;
  totalStakeWeight: bigint;
  chainLength: bigint;
}

/**
 * Choose between competing chain tips
 *
 * Uses "heaviest stake" rule: prefer chain with most
 * cumulative validator stake in attestations.
 */
export function chooseFork(
  candidates: Array<{
    block: Block;
    attestingStake: bigint;
    chainLength: bigint;
  }>
): ForkChoice | null {
  if (candidates.length === 0) return null;

  // Sort by attesting stake, then by chain length
  const sorted = [...candidates].sort((a, b) => {
    // Primary: more attesting stake
    if (b.attestingStake > a.attestingStake) return 1;
    if (b.attestingStake < a.attestingStake) return -1;

    // Secondary: longer chain
    if (b.chainLength > a.chainLength) return 1;
    if (b.chainLength < a.chainLength) return -1;

    // Tertiary: lower block hash (deterministic tiebreaker)
    const hashA = hashBlockHeader(a.block.header);
    const hashB = hashBlockHeader(b.block.header);
    return hashA.localeCompare(hashB);
  });

  const best = sorted[0];
  return {
    bestBlock: best.block,
    bestBlockHash: hashBlockHeader(best.block.header),
    totalStakeWeight: best.attestingStake,
    chainLength: best.chainLength,
  };
}

/**
 * Check if a reorg is allowed
 */
export function isReorgAllowed(
  currentBlock: bigint,
  reorgToBlock: bigint,
  lastCheckpoint: bigint
): { allowed: boolean; reason?: string } {
  // Cannot reorg past checkpoint
  if (reorgToBlock < lastCheckpoint) {
    return {
      allowed: false,
      reason: 'Cannot reorg past checkpoint',
    };
  }

  // Cannot reorg more than MAX_REORG_DEPTH
  const depth = currentBlock - reorgToBlock;
  if (depth > BigInt(FINALITY_PARAMS.MAX_REORG_DEPTH)) {
    return {
      allowed: false,
      reason: `Reorg depth ${depth} exceeds maximum ${FINALITY_PARAMS.MAX_REORG_DEPTH}`,
    };
  }

  return { allowed: true };
}

// ============ Fact Finality ============

/**
 * Check if a fact is finalized
 *
 * A fact is final when:
 * 1. The block containing it is finalized
 * 2. No successful challenge exists
 */
export function isFactFinalized(
  factRecordedBlock: bigint,
  currentBlock: bigint,
  hasPendingChallenge: boolean,
  checkpoints: Checkpoint[]
): { finalized: boolean; reason: string } {
  // Check for pending challenges
  if (hasPendingChallenge) {
    return {
      finalized: false,
      reason: 'Fact has pending challenge',
    };
  }

  // Check block finality
  if (!isBlockFinalized(factRecordedBlock, currentBlock, checkpoints)) {
    const confirmations = getConfirmations(factRecordedBlock, currentBlock);
    return {
      finalized: false,
      reason: `Block not finalized (${confirmations}/${FINALITY_PARAMS.CONFIRMATION_DEPTH} confirmations)`,
    };
  }

  return {
    finalized: true,
    reason: 'Block finalized, no pending challenges',
  };
}

/**
 * Get fact finality status
 */
export interface FactFinalityStatus {
  isFinalized: boolean;
  confirmations: bigint;
  confirmationsRequired: number;
  finalityProbability: number;
  pendingChallenges: number;
  estimatedFinalityBlock: bigint;
}

export function getFactFinalityStatus(
  factRecordedBlock: bigint,
  currentBlock: bigint,
  pendingChallenges: number
): FactFinalityStatus {
  const confirmations = getConfirmations(factRecordedBlock, currentBlock);
  const isFinalized = confirmations >= BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH)
    && pendingChallenges === 0;

  return {
    isFinalized,
    confirmations,
    confirmationsRequired: FINALITY_PARAMS.CONFIRMATION_DEPTH,
    finalityProbability: calculateFinalityProbability(confirmations),
    pendingChallenges,
    estimatedFinalityBlock: factRecordedBlock + BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH),
  };
}

// ============ Utility Functions ============

/**
 * Hash a block header
 */
function hashBlockHeader(header: BlockHeader): string {
  const data = JSON.stringify({
    number: header.number.toString(),
    timestamp: header.timestamp,
    previousHash: header.previousHash,
    stateRoot: header.stateRoot,
    knowledgeRoot: header.knowledgeRoot,
    transactionsRoot: header.transactionsRoot,
    verificationsRoot: header.verificationsRoot,
    proposer: header.proposer,
  });

  return sha512_256Hex(data);
}

/**
 * Calculate block weight for fork choice
 */
export function calculateBlockWeight(
  block: Block,
  validatorSet: ValidatorSet
): bigint {
  // Weight is sum of attesting validator stakes
  let weight = 0n;

  for (const attestation of block.attestations) {
    const validator = validatorSet.validators.get(attestation.validator);
    if (validator) {
      weight += validator.stake + validator.delegatedStake;
    }
  }

  return weight;
}

/**
 * Get chain weight from genesis to tip
 */
export function calculateChainWeight(
  blocks: Block[],
  validatorSet: ValidatorSet
): bigint {
  return blocks.reduce(
    (sum, block) => sum + calculateBlockWeight(block, validatorSet),
    0n
  );
}
