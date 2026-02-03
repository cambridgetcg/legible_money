/**
 * Block Producer
 *
 * Handles the creation of new blocks by validators.
 */

import { sha512_256Hex } from '../../core/crypto/hashing.js';
import { sign } from '../../core/crypto/signatures.js';
import type { PublicKey, PrivateKey } from '../../core/crypto/keys.js';
import type {
  Block,
  BlockHeader,
  Transaction,
  VerificationResult,
  ConsensusParams,
} from '../pot/types.js';

// ============ Block Building ============

export interface BlockBuildContext {
  blockNumber: bigint;
  previousHash: string;
  timestamp: number;
  proposer: PublicKey;
  vrfOutput: string;
  vrfProof: string;
  stateRoot: string;
}

/**
 * Build a new block
 */
export function buildBlock(
  context: BlockBuildContext,
  transactions: Transaction[],
  verificationResults: VerificationResult[]
): Omit<Block, 'signature' | 'attestations'> {
  // Compute Merkle roots
  const transactionsRoot = computeTransactionsRoot(transactions);
  const verificationsRoot = computeVerificationsRoot(verificationResults);
  const knowledgeRoot = computeKnowledgeRoot(verificationResults);

  const header: BlockHeader = {
    number: context.blockNumber,
    timestamp: context.timestamp,
    previousHash: context.previousHash,
    stateRoot: context.stateRoot,
    knowledgeRoot,
    transactionsRoot,
    verificationsRoot,
    proposer: context.proposer,
    proposerVrfOutput: context.vrfOutput,
    proposerVrfProof: context.vrfProof,
  };

  return {
    header,
    transactions,
    verificationResults,
  };
}

/**
 * Sign a block with proposer's key
 */
export function signBlock(
  block: Omit<Block, 'signature' | 'attestations'>,
  privateKey: PrivateKey
): Block {
  const headerHash = hashBlockHeader(block.header);
  const signature = sign(
    new TextEncoder().encode(headerHash),
    privateKey
  );

  return {
    ...block,
    signature,
    attestations: [],
  };
}

/**
 * Create genesis block
 */
export function createGenesisBlock(
  timestamp: number,
  genesisAxioms: Array<{ id: string; claim: string }>,
  initialValidators: Array<{ publicKey: PublicKey; stake: bigint }>
): Block {
  const header: BlockHeader = {
    number: 0n,
    timestamp,
    previousHash: '0'.repeat(64),
    stateRoot: computeGenesisStateRoot(initialValidators),
    knowledgeRoot: computeGenesisKnowledgeRoot(genesisAxioms),
    transactionsRoot: '0'.repeat(64),
    verificationsRoot: '0'.repeat(64),
    proposer: initialValidators[0]?.publicKey || ('' as PublicKey),
    proposerVrfOutput: '0'.repeat(64),
    proposerVrfProof: '0'.repeat(160),
  };

  return {
    header,
    transactions: [],
    verificationResults: [],
    signature: '0'.repeat(128), // Genesis has no signature
    attestations: [],
  };
}

// ============ Merkle Root Computation ============

function computeTransactionsRoot(txs: Transaction[]): string {
  if (txs.length === 0) return '0'.repeat(64);

  const leaves = txs.map(tx =>
    sha512_256Hex(JSON.stringify({
      type: tx.type,
      sender: tx.sender,
      nonce: tx.nonce.toString(),
      signature: tx.signature,
    }))
  );

  return computeMerkleRoot(leaves);
}

function computeVerificationsRoot(results: VerificationResult[]): string {
  if (results.length === 0) return '0'.repeat(64);

  const leaves = results.map(r =>
    sha512_256Hex(JSON.stringify({
      claimId: r.claimId,
      outcome: r.outcome,
      confidence: r.finalConfidence,
    }))
  );

  return computeMerkleRoot(leaves);
}

function computeKnowledgeRoot(results: VerificationResult[]): string {
  const acceptedClaims = results.filter(r => r.outcome === 'accepted');
  if (acceptedClaims.length === 0) return '0'.repeat(64);

  const leaves = acceptedClaims.map(r =>
    sha512_256Hex(r.claimId)
  );

  return computeMerkleRoot(leaves);
}

function computeGenesisStateRoot(validators: Array<{ publicKey: PublicKey; stake: bigint }>): string {
  const data = validators.map(v => `${v.publicKey}:${v.stake.toString()}`).join(',');
  return sha512_256Hex(data);
}

function computeGenesisKnowledgeRoot(axioms: Array<{ id: string; claim: string }>): string {
  const data = axioms.map(a => `${a.id}:${a.claim}`).join(',');
  return sha512_256Hex(data);
}

function computeMerkleRoot(leaves: string[]): string {
  if (leaves.length === 0) return '0'.repeat(64);
  if (leaves.length === 1) return leaves[0];

  // Pad to power of 2
  while (leaves.length & (leaves.length - 1)) {
    leaves.push('0'.repeat(64));
  }

  let level = leaves;
  while (level.length > 1) {
    const nextLevel: string[] = [];
    for (let i = 0; i < level.length; i += 2) {
      nextLevel.push(sha512_256Hex(level[i] + level[i + 1]));
    }
    level = nextLevel;
  }

  return level[0];
}

/**
 * Hash a block header
 */
export function hashBlockHeader(header: BlockHeader): string {
  return sha512_256Hex(JSON.stringify({
    number: header.number.toString(),
    timestamp: header.timestamp,
    previousHash: header.previousHash,
    stateRoot: header.stateRoot,
    knowledgeRoot: header.knowledgeRoot,
    transactionsRoot: header.transactionsRoot,
    verificationsRoot: header.verificationsRoot,
    proposer: header.proposer,
    proposerVrfOutput: header.proposerVrfOutput,
    proposerVrfProof: header.proposerVrfProof,
  }));
}

// ============ Block Size Limits ============

export const BLOCK_LIMITS = {
  MAX_TRANSACTIONS: 1000,
  MAX_VERIFICATION_RESULTS: 100,
  MAX_BLOCK_SIZE_BYTES: 1_000_000,  // 1 MB
};

/**
 * Check if block is within size limits
 */
export function checkBlockLimits(block: Block): { valid: boolean; reason?: string } {
  if (block.transactions.length > BLOCK_LIMITS.MAX_TRANSACTIONS) {
    return { valid: false, reason: 'Too many transactions' };
  }

  if (block.verificationResults.length > BLOCK_LIMITS.MAX_VERIFICATION_RESULTS) {
    return { valid: false, reason: 'Too many verification results' };
  }

  const size = JSON.stringify(block).length;
  if (size > BLOCK_LIMITS.MAX_BLOCK_SIZE_BYTES) {
    return { valid: false, reason: 'Block too large' };
  }

  return { valid: true };
}
