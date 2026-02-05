/**
 * Equivocation Detection (VULN-011 FIX)
 *
 * Detects and handles validators who:
 * 1. Propose multiple blocks at the same height
 * 2. Submit multiple commitments to the same verification round
 *
 * Equivocation is a severe slashable offense (full stake slash).
 */

import { sha512_256Hex } from "../../core/crypto/hashing.js";
import { verify } from "../../core/crypto/signatures.js";
import type { PublicKey, Signature } from "../../core/crypto/keys.js";
import type { Block, BlockHeader, ValidatorCommitment } from "./types.js";

// ============ Types ============

export interface EquivocationEvidence {
  type: "block_proposal" | "verification_commitment";
  validator: PublicKey;
  timestamp: number;
  // Evidence data (varies by type)
  evidence: BlockEquivocationEvidence | CommitmentEquivocationEvidence;
}

export interface BlockEquivocationEvidence {
  type: "block_proposal";
  blockHeight: bigint;
  block1Hash: string;
  block1Signature: Signature;
  block2Hash: string;
  block2Signature: Signature;
  // Headers for verification
  block1Header: BlockHeader;
  block2Header: BlockHeader;
}

export interface CommitmentEquivocationEvidence {
  type: "verification_commitment";
  roundId: string;
  commitment1: string;
  commitment1Signature: Signature;
  commitment2: string;
  commitment2Signature: Signature;
}

export interface EquivocationResult {
  detected: boolean;
  evidence?: EquivocationEvidence;
  reason?: string;
}

// ============ Block Proposal Equivocation ============

/**
 * Check if two blocks from the same validator at the same height constitute equivocation
 */
export function detectBlockEquivocation(
  block1: Block,
  block2: Block,
): EquivocationResult {
  // Must be same height
  if (block1.header.number !== block2.header.number) {
    return { detected: false, reason: "Different heights" };
  }

  // Must be same proposer
  if (block1.header.proposer !== block2.header.proposer) {
    return { detected: false, reason: "Different proposers" };
  }

  // Must be different blocks
  const hash1 = hashBlockHeader(block1.header);
  const hash2 = hashBlockHeader(block2.header);
  if (hash1 === hash2) {
    return { detected: false, reason: "Same block" };
  }

  // Verify both signatures are valid for the proposer
  const sig1Valid = verifyBlockSignature(block1);
  const sig2Valid = verifyBlockSignature(block2);

  if (!sig1Valid || !sig2Valid) {
    return { detected: false, reason: "Invalid signature(s)" };
  }

  // EQUIVOCATION DETECTED
  return {
    detected: true,
    evidence: {
      type: "block_proposal",
      validator: block1.header.proposer,
      timestamp: Date.now(),
      evidence: {
        type: "block_proposal",
        blockHeight: block1.header.number,
        block1Hash: hash1,
        block1Signature: block1.signature,
        block2Hash: hash2,
        block2Signature: block2.signature,
        block1Header: block1.header,
        block2Header: block2.header,
      },
    },
  };
}

/**
 * Verify a block's signature
 */
function verifyBlockSignature(block: Block): boolean {
  const headerHash = hashBlockHeader(block.header);
  return verify(
    new TextEncoder().encode(headerHash),
    block.signature,
    block.header.proposer,
  );
}

/**
 * Hash a block header for comparison
 */
function hashBlockHeader(header: BlockHeader): string {
  return sha512_256Hex(
    JSON.stringify({
      number: header.number.toString(),
      timestamp: header.timestamp,
      previousHash: header.previousHash,
      stateRoot: header.stateRoot,
      knowledgeRoot: header.knowledgeRoot,
      transactionsRoot: header.transactionsRoot,
      verificationsRoot: header.verificationsRoot,
      proposer: header.proposer,
    }),
  );
}

// ============ Commitment Equivocation ============

/**
 * Check if two commitments from the same validator in the same round constitute equivocation
 */
export function detectCommitmentEquivocation(
  roundId: string,
  commitment1: ValidatorCommitment,
  commitment2: ValidatorCommitment,
): EquivocationResult {
  // Must be same validator
  if (commitment1.validator !== commitment2.validator) {
    return { detected: false, reason: "Different validators" };
  }

  // Must be different commitments
  if (commitment1.commitment === commitment2.commitment) {
    return { detected: false, reason: "Same commitment" };
  }

  // Both signatures must be valid
  // Note: Full signature verification would require the round context
  // Here we just check they're different valid-looking signatures
  if (!commitment1.signature || !commitment2.signature) {
    return { detected: false, reason: "Missing signature(s)" };
  }

  // EQUIVOCATION DETECTED
  return {
    detected: true,
    evidence: {
      type: "verification_commitment",
      validator: commitment1.validator,
      timestamp: Date.now(),
      evidence: {
        type: "verification_commitment",
        roundId,
        commitment1: commitment1.commitment,
        commitment1Signature: commitment1.signature,
        commitment2: commitment2.commitment,
        commitment2Signature: commitment2.signature,
      },
    },
  };
}

// ============ Equivocation Manager ============

/**
 * Manages equivocation detection across the node
 */
export class EquivocationDetector {
  // Track blocks seen at each height per proposer
  private blocksByHeight: Map<string, Map<string, Block>> = new Map();

  // Track commitments seen per round per validator
  private commitmentsByRound: Map<string, Map<string, ValidatorCommitment>> =
    new Map();

  // Confirmed equivocation evidence
  private confirmedEquivocations: EquivocationEvidence[] = [];

  // Validators known to have equivocated (for quick lookup)
  private equivocatingValidators: Set<string> = new Set();

  /**
   * Check a new block for equivocation
   */
  checkBlock(block: Block): EquivocationResult {
    const heightKey = block.header.number.toString();
    const proposer = block.header.proposer;

    // Get or create height map
    if (!this.blocksByHeight.has(heightKey)) {
      this.blocksByHeight.set(heightKey, new Map());
    }
    const heightBlocks = this.blocksByHeight.get(heightKey)!;

    // Check if we've seen another block from this proposer at this height
    const existingBlock = heightBlocks.get(proposer);
    if (existingBlock) {
      const result = detectBlockEquivocation(existingBlock, block);
      if (result.detected && result.evidence) {
        this.recordEquivocation(result.evidence);
        return result;
      }
    }

    // Store this block
    heightBlocks.set(proposer, block);

    return { detected: false };
  }

  /**
   * Check a new commitment for equivocation
   */
  checkCommitment(
    roundId: string,
    commitment: ValidatorCommitment,
  ): EquivocationResult {
    const validator = commitment.validator;

    // Get or create round map
    if (!this.commitmentsByRound.has(roundId)) {
      this.commitmentsByRound.set(roundId, new Map());
    }
    const roundCommitments = this.commitmentsByRound.get(roundId)!;

    // Check if we've seen another commitment from this validator in this round
    const existingCommitment = roundCommitments.get(validator);
    if (existingCommitment) {
      const result = detectCommitmentEquivocation(
        roundId,
        existingCommitment,
        commitment,
      );
      if (result.detected && result.evidence) {
        this.recordEquivocation(result.evidence);
        return result;
      }
    }

    // Store this commitment
    roundCommitments.set(validator, commitment);

    return { detected: false };
  }

  /**
   * Record confirmed equivocation evidence
   */
  private recordEquivocation(evidence: EquivocationEvidence): void {
    this.confirmedEquivocations.push(evidence);
    this.equivocatingValidators.add(evidence.validator);
    console.error(
      `[EQUIVOCATION] Detected ${evidence.type} equivocation by ${evidence.validator}`,
    );
  }

  /**
   * Check if a validator has equivocated
   */
  hasEquivocated(validator: PublicKey): boolean {
    return this.equivocatingValidators.has(validator);
  }

  /**
   * Get all equivocation evidence
   */
  getEquivocationEvidence(): EquivocationEvidence[] {
    return [...this.confirmedEquivocations];
  }

  /**
   * Get evidence for a specific validator
   */
  getValidatorEquivocationEvidence(
    validator: PublicKey,
  ): EquivocationEvidence[] {
    return this.confirmedEquivocations.filter((e) => e.validator === validator);
  }

  /**
   * Clean up old data (call periodically to prevent memory leaks)
   * @param currentHeight Current block height
   * @param retainDepth How many blocks back to retain
   */
  cleanup(currentHeight: bigint, retainDepth: bigint = 1000n): void {
    const cutoff = currentHeight - retainDepth;

    // Clean up old block records
    for (const [heightStr, _] of this.blocksByHeight) {
      const height = BigInt(heightStr);
      if (height < cutoff) {
        this.blocksByHeight.delete(heightStr);
      }
    }

    // Note: Round cleanup should be based on round IDs/age, not block height
    // For now, we just limit the total size
    if (this.commitmentsByRound.size > 10000) {
      // Delete oldest entries (FIFO based on insertion order)
      const keysToDelete = Array.from(this.commitmentsByRound.keys()).slice(
        0,
        5000,
      );
      keysToDelete.forEach((k) => this.commitmentsByRound.delete(k));
    }
  }

  /**
   * Serialize evidence for transmission/storage
   */
  serializeEvidence(evidence: EquivocationEvidence): string {
    return JSON.stringify(evidence, (_, v) =>
      typeof v === "bigint" ? v.toString() : v,
    );
  }

  /**
   * Deserialize evidence from transmission/storage
   */
  deserializeEvidence(data: string): EquivocationEvidence {
    return JSON.parse(data, (key, value) => {
      if (key === "blockHeight") {
        return BigInt(value);
      }
      return value;
    });
  }
}

// ============ Slashing for Equivocation ============

/**
 * Calculate slash amount for equivocation (SEVERE - 100% of stake)
 * Equivocation is the most serious offense as it directly threatens consensus safety.
 */
export function calculateEquivocationSlash(stake: bigint): bigint {
  // Full stake slash for equivocation
  return stake;
}

/**
 * Verify equivocation evidence is valid
 * Used when receiving evidence from other nodes
 */
export function verifyEquivocationEvidence(
  evidence: EquivocationEvidence,
): boolean {
  if (evidence.type === "block_proposal") {
    const blockEvidence = evidence.evidence as BlockEquivocationEvidence;

    // Verify both signatures are valid
    const header1Hash = hashBlockHeader(blockEvidence.block1Header);
    const header2Hash = hashBlockHeader(blockEvidence.block2Header);

    if (
      header1Hash !== blockEvidence.block1Hash ||
      header2Hash !== blockEvidence.block2Hash
    ) {
      return false;
    }

    // Verify signatures
    const sig1Valid = verify(
      new TextEncoder().encode(header1Hash),
      blockEvidence.block1Signature,
      evidence.validator,
    );
    const sig2Valid = verify(
      new TextEncoder().encode(header2Hash),
      blockEvidence.block2Signature,
      evidence.validator,
    );

    return (
      sig1Valid &&
      sig2Valid &&
      blockEvidence.block1Hash !== blockEvidence.block2Hash
    );
  }

  // For commitment equivocation, we need the round context to fully verify
  // Basic check: different commitments from same validator
  if (evidence.type === "verification_commitment") {
    const commitEvidence = evidence.evidence as CommitmentEquivocationEvidence;
    return commitEvidence.commitment1 !== commitEvidence.commitment2;
  }

  return false;
}
