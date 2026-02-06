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

import type { Block, BlockHeader, ValidatorSet } from "./types.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";
import { verifyDomainSeparated } from "../../core/crypto/signatures.js";
import type { PublicKey } from "../../core/crypto/keys.js";

// ============ Finality Parameters ============

export const FINALITY_PARAMS = {
  // Block finality
  CONFIRMATION_DEPTH: 19, // Blocks needed for finality
  FAST_FINALITY_THRESHOLD: 0.67, // Stake needed for fast finality (BFT 2/3 - keep)

  // Checkpoint frequency
  CHECKPOINT_INTERVAL: 1111, // Blocks between checkpoints
  CHECKPOINT_SIGNATURES_REQUIRED: 0.67, // Stake needed to sign checkpoint (BFT 2/3)

  // Fork choice
  MAX_REORG_DEPTH: 111, // Maximum reorg depth allowed
  FORK_CHOICE_RULE: "heaviest_stake", // How to choose between forks
};

// ============ Block Finality ============

/**
 * Check if a block is finalized
 */
export function isBlockFinalized(
  blockNumber: bigint,
  currentBlock: bigint,
  checkpoints: Checkpoint[],
): boolean {
  // Check against checkpoints first
  const latestCheckpoint =
    checkpoints.length > 0 ? checkpoints[checkpoints.length - 1] : null;

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
  currentBlock: bigint,
): bigint {
  if (blockNumber > currentBlock) return 0n;
  return currentBlock - blockNumber;
}

/**
 * Calculate finality probability based on confirmations
 *
 * Assumes a simple model where each confirmation
 * exponentially decreases reorg probability.
 *
 * GAP-018 FIX: Now supports dynamic Byzantine fraction calculation
 */
export function calculateFinalityProbability(
  confirmations: bigint,
  byzantineStakeFraction: number = 0.33,
): number {
  // Probability of reorg decreases exponentially
  // P(reorg) ≈ byzantineStakeFraction ^ confirmations
  const reorgProb = Math.pow(byzantineStakeFraction, Number(confirmations));
  return 1 - reorgProb;
}

/**
 * GAP-018 FIX: Calculate Byzantine stake fraction from validator set
 *
 * Estimates the maximum fraction of stake that could be Byzantine based on:
 * - Known malicious validators (if any flagged)
 * - Offline/unresponsive validators
 * - Concentration risk (top N validators controlling too much)
 *
 * This provides a dynamic, conservative estimate rather than assuming constant 33%
 */
export function calculateByzantineFraction(
  validatorSet: ValidatorSet,
  offlineValidators: Set<string> = new Set(),
  flaggedValidators: Set<string> = new Set(),
): {
  byzantineFraction: number;
  breakdown: {
    offlineStake: bigint;
    flaggedStake: bigint;
    concentrationRisk: number;
    estimatedByzantine: number;
  };
} {
  let offlineStake = 0n;
  let flaggedStake = 0n;

  // Calculate stake from offline and flagged validators
  for (const [key, validator] of validatorSet.validators) {
    const stake = validator.stake + validator.delegatedStake;

    if (offlineValidators.has(key)) {
      offlineStake += stake;
    }

    if (flaggedValidators.has(key)) {
      flaggedStake += stake;
    }
  }

  // Calculate concentration risk (top 3 validators)
  const sortedValidators = Array.from(validatorSet.validators.values()).sort(
    (a, b) => {
      const stakeA = a.stake + a.delegatedStake;
      const stakeB = b.stake + b.delegatedStake;
      return stakeB > stakeA ? 1 : stakeB < stakeA ? -1 : 0;
    },
  );

  const top3Stake = sortedValidators
    .slice(0, 3)
    .reduce((sum, v) => sum + v.stake + v.delegatedStake, 0n);

  const concentrationRisk =
    validatorSet.totalStake > 0n
      ? Number(top3Stake) / Number(validatorSet.totalStake)
      : 0;

  // Calculate known Byzantine stake (offline or flagged)
  const knownByzantine = offlineStake + flaggedStake;
  const knownByzantineFraction =
    validatorSet.totalStake > 0n
      ? Number(knownByzantine) / Number(validatorSet.totalStake)
      : 0;

  // Conservative estimate: known Byzantine + potential from concentration
  // If top 3 control > 50%, add a risk premium
  const concentrationPremium =
    concentrationRisk > 0.5
      ? (concentrationRisk - 0.5) * 0.2 // 20% of excess concentration
      : 0;

  // Minimum Byzantine assumption of 10% even with no known issues
  const estimatedByzantine = Math.max(
    0.1,
    Math.min(0.49, knownByzantineFraction + concentrationPremium),
  );

  return {
    byzantineFraction: estimatedByzantine,
    breakdown: {
      offlineStake,
      flaggedStake,
      concentrationRisk,
      estimatedByzantine,
    },
  };
}

/**
 * GAP-018 FIX: Calculate finality probability with dynamic Byzantine estimation
 */
export function calculateDynamicFinalityProbability(
  confirmations: bigint,
  validatorSet: ValidatorSet,
  offlineValidators: Set<string> = new Set(),
  flaggedValidators: Set<string> = new Set(),
): {
  probability: number;
  byzantineFraction: number;
  breakdown: ReturnType<typeof calculateByzantineFraction>["breakdown"];
} {
  const { byzantineFraction, breakdown } = calculateByzantineFraction(
    validatorSet,
    offlineValidators,
    flaggedValidators,
  );

  const probability = calculateFinalityProbability(
    confirmations,
    byzantineFraction,
  );

  return {
    probability,
    byzantineFraction,
    breakdown,
  };
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

  /**
   * GAP-026 FIX: Aggregated signature data for bandwidth efficiency
   */
  aggregation?: CheckpointAggregation;
}

/**
 * GAP-026 FIX: Aggregated signature representation
 *
 * While true BLS aggregation requires special cryptographic libraries,
 * this structure provides:
 * 1. A combined hash of all signatures for quick integrity checks
 * 2. A bitmap of which validators signed (compact representation)
 * 3. Optional BLS aggregate signature when available
 */
export interface CheckpointAggregation {
  /** Hash of all individual signatures combined (for integrity) */
  combinedHash: string;

  /** Bitmap of signing validators (indexed by sorted validator position) */
  signerBitmap: string; // Hex-encoded bitmap

  /** Number of signers */
  signerCount: number;

  /** BLS aggregate signature (if BLS is used) */
  blsAggregate?: string;
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
  signatures: CheckpointSignature[],
): Checkpoint {
  const totalStake = signatures.reduce((sum, s) => sum + s.stake, 0n);

  return {
    blockNumber: block.header.number,
    blockHash: hashBlockHeader(block.header),
    stateRoot: block.header.stateRoot,
    knowledgeRoot: block.header.knowledgeRoot,
    signatures,
    totalSigningStake: totalStake,
    createdAt: block.header.timestamp, // P1: Use deterministic block timestamp
  };
}

/**
 * Verify a checkpoint has sufficient stake
 * GAP-009 FIX: Now performs actual cryptographic signature verification
 */
export function verifyCheckpoint(
  checkpoint: Checkpoint,
  validatorSet: ValidatorSet,
): { valid: boolean; reason?: string } {
  // Check stake threshold
  const requiredStake =
    (validatorSet.totalStake *
      BigInt(
        Math.floor(FINALITY_PARAMS.CHECKPOINT_SIGNATURES_REQUIRED * 10000),
      )) /
    10000n;

  if (checkpoint.totalSigningStake < requiredStake) {
    return {
      valid: false,
      reason: `Insufficient stake: ${checkpoint.totalSigningStake} < ${requiredStake}`,
    };
  }

  // Prepare checkpoint data for signature verification
  const checkpointData = {
    blockNumber: checkpoint.blockNumber.toString(),
    blockHash: checkpoint.blockHash,
    stateRoot: checkpoint.stateRoot,
    knowledgeRoot: checkpoint.knowledgeRoot,
  };
  const serializedData = JSON.stringify(checkpointData);

  // GAP-009 FIX: Verify each signature cryptographically
  let verifiedStake = 0n;
  for (const sig of checkpoint.signatures) {
    const validator = validatorSet.validators.get(sig.validator);
    if (!validator) {
      return { valid: false, reason: `Unknown validator: ${sig.validator}` };
    }

    if (!validator.isActive) {
      return { valid: false, reason: `Validator not active: ${sig.validator}` };
    }

    // Verify the signature using domain-separated verification
    const signatureValid = verifyDomainSeparated(
      "LGM.checkpoint.commit.v1",
      serializedData,
      sig.signature,
      sig.validator as PublicKey,
    );

    if (!signatureValid) {
      return {
        valid: false,
        reason: `Invalid signature from validator: ${sig.validator}`,
      };
    }

    // Verify stake matches current validator stake
    const actualStake = validator.stake + validator.delegatedStake;
    if (sig.stake !== actualStake) {
      return {
        valid: false,
        reason: `Stake mismatch for ${sig.validator}: claimed ${sig.stake}, actual ${actualStake}`,
      };
    }

    verifiedStake += actualStake;
  }

  // Final stake verification after all signatures verified
  if (verifiedStake < requiredStake) {
    return {
      valid: false,
      reason: `Verified stake insufficient: ${verifiedStake} < ${requiredStake}`,
    };
  }

  return { valid: true };
}

/**
 * Check if a new checkpoint should be created
 */
export function shouldCreateCheckpoint(
  currentBlock: bigint,
  lastCheckpointBlock: bigint,
): boolean {
  return (
    currentBlock - lastCheckpointBlock >=
    BigInt(FINALITY_PARAMS.CHECKPOINT_INTERVAL)
  );
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
  }>,
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
  lastCheckpoint: bigint,
): { allowed: boolean; reason?: string } {
  // Cannot reorg past checkpoint
  if (reorgToBlock < lastCheckpoint) {
    return {
      allowed: false,
      reason: "Cannot reorg past checkpoint",
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
  checkpoints: Checkpoint[],
): { finalized: boolean; reason: string } {
  // Check for pending challenges
  if (hasPendingChallenge) {
    return {
      finalized: false,
      reason: "Fact has pending challenge",
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
    reason: "Block finalized, no pending challenges",
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
  pendingChallenges: number,
): FactFinalityStatus {
  const confirmations = getConfirmations(factRecordedBlock, currentBlock);
  const isFinalized =
    confirmations >= BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH) &&
    pendingChallenges === 0;

  return {
    isFinalized,
    confirmations,
    confirmationsRequired: FINALITY_PARAMS.CONFIRMATION_DEPTH,
    finalityProbability: calculateFinalityProbability(confirmations),
    pendingChallenges,
    estimatedFinalityBlock:
      factRecordedBlock + BigInt(FINALITY_PARAMS.CONFIRMATION_DEPTH),
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
  validatorSet: ValidatorSet,
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
  validatorSet: ValidatorSet,
): bigint {
  return blocks.reduce(
    (sum, block) => sum + calculateBlockWeight(block, validatorSet),
    0n,
  );
}

// ============ GAP-026 FIX: Signature Aggregation ============

/**
 * GAP-026 FIX: Aggregate checkpoint signatures
 *
 * Creates a compact representation of all signatures:
 * - Combined hash for integrity verification
 * - Bitmap for efficient signer lookup
 *
 * @param signatures - Individual validator signatures
 * @param validatorSet - For determining bitmap positions
 * @returns Aggregation data
 */
export function aggregateCheckpointSignatures(
  signatures: CheckpointSignature[],
  validatorSet: ValidatorSet,
): CheckpointAggregation {
  // Sort validator keys for deterministic bitmap ordering
  const sortedValidatorKeys = Array.from(validatorSet.validators.keys()).sort();
  const validatorIndexMap = new Map(sortedValidatorKeys.map((k, i) => [k, i]));

  // Create bitmap of signers
  const signerSet = new Set(signatures.map((s) => s.validator));
  const bitmapSize = Math.ceil(sortedValidatorKeys.length / 8);
  const bitmap = new Uint8Array(bitmapSize);

  for (const [validator, index] of validatorIndexMap) {
    if (signerSet.has(validator)) {
      const byteIndex = Math.floor(index / 8);
      const bitIndex = index % 8;
      bitmap[byteIndex] |= 1 << bitIndex;
    }
  }

  // Create combined hash of all signatures (sorted for determinism)
  const sortedSignatures = [...signatures].sort((a, b) =>
    a.validator.localeCompare(b.validator),
  );
  const combinedData = sortedSignatures
    .map((s) => `${s.validator}:${s.signature}`)
    .join("|");
  const combinedHash = sha512_256Hex(combinedData);

  // Convert bitmap to hex without Buffer
  const signerBitmap = Array.from(bitmap)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");

  return {
    combinedHash,
    signerBitmap,
    signerCount: signatures.length,
  };
}

/**
 * GAP-026 FIX: Verify aggregated signatures match individual signatures
 */
export function verifySignatureAggregation(
  checkpoint: Checkpoint,
  validatorSet: ValidatorSet,
): { valid: boolean; reason?: string } {
  if (!checkpoint.aggregation) {
    return { valid: true }; // No aggregation to verify
  }

  // Recompute aggregation from individual signatures
  const recomputed = aggregateCheckpointSignatures(
    checkpoint.signatures,
    validatorSet,
  );

  // Verify combined hash matches
  if (recomputed.combinedHash !== checkpoint.aggregation.combinedHash) {
    return {
      valid: false,
      reason: `Combined hash mismatch: ${recomputed.combinedHash} !== ${checkpoint.aggregation.combinedHash}`,
    };
  }

  // Verify signer count matches
  if (recomputed.signerCount !== checkpoint.aggregation.signerCount) {
    return {
      valid: false,
      reason: `Signer count mismatch: ${recomputed.signerCount} !== ${checkpoint.aggregation.signerCount}`,
    };
  }

  // Verify bitmap matches
  if (recomputed.signerBitmap !== checkpoint.aggregation.signerBitmap) {
    return {
      valid: false,
      reason: `Signer bitmap mismatch`,
    };
  }

  return { valid: true };
}

/**
 * GAP-026 FIX: Get signers from aggregated bitmap
 *
 * Allows reconstructing signer list from bitmap without storing full signatures
 */
export function getSignersFromBitmap(
  aggregation: CheckpointAggregation,
  validatorSet: ValidatorSet,
): string[] {
  const sortedValidatorKeys = Array.from(validatorSet.validators.keys()).sort();

  // Convert hex to Uint8Array without Buffer
  const hex = aggregation.signerBitmap;
  const bitmap = new Uint8Array(hex.length / 2);
  for (let i = 0; i < bitmap.length; i++) {
    bitmap[i] = parseInt(hex.slice(i * 2, i * 2 + 2), 16);
  }

  const signers: string[] = [];

  for (let i = 0; i < sortedValidatorKeys.length; i++) {
    const byteIndex = Math.floor(i / 8);
    const bitIndex = i % 8;

    if (
      byteIndex < bitmap.length &&
      (bitmap[byteIndex] & (1 << bitIndex)) !== 0
    ) {
      signers.push(sortedValidatorKeys[i]);
    }
  }

  return signers;
}

/**
 * GAP-026 FIX: Create checkpoint with aggregation
 */
export function createCheckpointWithAggregation(
  block: Block,
  signatures: CheckpointSignature[],
  validatorSet: ValidatorSet,
): Checkpoint {
  const checkpoint = createCheckpoint(block, signatures);

  // Add aggregation data
  checkpoint.aggregation = aggregateCheckpointSignatures(
    signatures,
    validatorSet,
  );

  return checkpoint;
}
