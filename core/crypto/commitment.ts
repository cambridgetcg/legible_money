/**
 * Commitment Schemes
 *
 * Cryptographic commitment schemes for Legible Money.
 * Used for: commit-reveal voting, claim submission, validator coordination.
 *
 * Properties:
 * - Hiding: Commitment reveals nothing about the value
 * - Binding: Cannot open commitment to different value
 */

import { sha512Hash, domainHash, HASH_DOMAINS, blake3Hash } from './hashing.js';
import { bytesToHex, hexToBytes } from './keys.js';
import { randomBytes } from '@noble/hashes/utils';

// ============ Types ============

export type Commitment = string;    // Hex-encoded commitment
export type Nonce = string;         // Hex-encoded nonce (32 bytes)
export type RevealedValue = string; // The revealed value

export interface CommitmentData {
  commitment: Commitment;
  nonce: Nonce;
  value: RevealedValue;
}

export interface CommitRevealRound<T> {
  id: string;
  phase: 'commit' | 'reveal' | 'complete';
  deadline: number;
  commitments: Map<string, Commitment>;  // participant -> commitment
  reveals: Map<string, { value: T; nonce: Nonce }>;
  results?: T[];
}

// ============ Basic Commitment ============

/**
 * Create a commitment to a value
 * commitment = H(nonce || value)
 */
export function commit(value: string | Uint8Array): CommitmentData {
  const nonce = randomBytes(32);
  const valueBytes = typeof value === 'string'
    ? new TextEncoder().encode(value)
    : value;

  const combined = new Uint8Array(nonce.length + valueBytes.length);
  combined.set(nonce, 0);
  combined.set(valueBytes, nonce.length);

  const commitment = bytesToHex(domainHash(HASH_DOMAINS.COMMITMENT, combined));

  return {
    commitment,
    nonce: bytesToHex(nonce),
    value: typeof value === 'string' ? value : bytesToHex(value),
  };
}

/**
 * Verify a commitment opening
 */
export function verifyCommitment(
  commitment: Commitment,
  value: string | Uint8Array,
  nonce: Nonce
): boolean {
  const nonceBytes = hexToBytes(nonce);
  const valueBytes = typeof value === 'string'
    ? new TextEncoder().encode(value)
    : value;

  const combined = new Uint8Array(nonceBytes.length + valueBytes.length);
  combined.set(nonceBytes, 0);
  combined.set(valueBytes, nonceBytes.length);

  const expected = bytesToHex(domainHash(HASH_DOMAINS.COMMITMENT, combined));
  return commitment === expected;
}

// ============ Pedersen-like Commitment (for numeric values) ============

/**
 * Pedersen-like commitment for numeric values
 * Allows homomorphic addition of commitments
 *
 * Note: This is a simplified version. Production should use
 * proper elliptic curve Pedersen commitments.
 */
export interface NumericCommitment {
  commitment: Commitment;
  blinding: Nonce;
}

/**
 * Commit to a numeric value with blinding factor
 */
export function commitNumeric(value: bigint): NumericCommitment {
  const blinding = randomBytes(32);

  // commitment = H(value || blinding)
  const valueBytes = bigintToBytes(value);
  const combined = new Uint8Array(valueBytes.length + blinding.length);
  combined.set(valueBytes, 0);
  combined.set(blinding, valueBytes.length);

  const commitment = bytesToHex(sha512Hash(combined));

  return {
    commitment,
    blinding: bytesToHex(blinding),
  };
}

/**
 * Verify numeric commitment
 */
export function verifyNumericCommitment(
  commitment: Commitment,
  value: bigint,
  blinding: Nonce
): boolean {
  const blindingBytes = hexToBytes(blinding);
  const valueBytes = bigintToBytes(value);

  const combined = new Uint8Array(valueBytes.length + blindingBytes.length);
  combined.set(valueBytes, 0);
  combined.set(blindingBytes, valueBytes.length);

  const expected = bytesToHex(sha512Hash(combined));
  return commitment === expected;
}

// ============ Validator Commit-Reveal ============

/**
 * Commit to a verification verdict
 * Used in Proof of Truth consensus to prevent validators from copying
 */
export interface VerdictCommitment {
  claimId: string;
  commitment: Commitment;
  nonce: Nonce;
  verdict: 'confirmed' | 'rejected' | 'uncertain';
  confidence: number;
}

/**
 * Create verdict commitment
 */
export function commitVerdict(
  claimId: string,
  verdict: 'confirmed' | 'rejected' | 'uncertain',
  confidence: number,
  validatorId: string
): VerdictCommitment {
  const nonce = randomBytes(32);

  const data = JSON.stringify({
    claimId,
    verdict,
    confidence,
    validator: validatorId,
  });

  const combined = new Uint8Array(nonce.length + data.length);
  combined.set(nonce, 0);
  combined.set(new TextEncoder().encode(data), nonce.length);

  const commitment = bytesToHex(domainHash(HASH_DOMAINS.COMMITMENT, combined));

  return {
    claimId,
    commitment,
    nonce: bytesToHex(nonce),
    verdict,
    confidence,
  };
}

/**
 * Verify verdict commitment opening
 */
export function verifyVerdictCommitment(
  commitment: Commitment,
  claimId: string,
  verdict: 'confirmed' | 'rejected' | 'uncertain',
  confidence: number,
  validatorId: string,
  nonce: Nonce
): boolean {
  const nonceBytes = hexToBytes(nonce);

  const data = JSON.stringify({
    claimId,
    verdict,
    confidence,
    validator: validatorId,
  });

  const combined = new Uint8Array(nonceBytes.length + data.length);
  combined.set(nonceBytes, 0);
  combined.set(new TextEncoder().encode(data), nonceBytes.length);

  const expected = bytesToHex(domainHash(HASH_DOMAINS.COMMITMENT, combined));
  return commitment === expected;
}

// ============ Batch Commitment ============

/**
 * Commit to multiple values with single commitment
 * Uses Merkle tree for efficiency
 */
export interface BatchCommitment {
  commitment: Commitment;  // Merkle root
  nonces: Nonce[];
  values: string[];
}

/**
 * Create batch commitment
 */
export function commitBatch(values: string[]): BatchCommitment {
  const nonces: Uint8Array[] = [];
  const leaves: Uint8Array[] = [];

  // Create individual commitments
  for (const value of values) {
    const nonce = randomBytes(32);
    nonces.push(nonce);

    const valueBytes = new TextEncoder().encode(value);
    const combined = new Uint8Array(nonce.length + valueBytes.length);
    combined.set(nonce, 0);
    combined.set(valueBytes, nonce.length);

    leaves.push(blake3Hash(combined));
  }

  // Build Merkle tree
  const root = buildMerkleRoot(leaves);

  return {
    commitment: bytesToHex(root),
    nonces: nonces.map(bytesToHex),
    values,
  };
}

/**
 * Verify single value in batch commitment
 */
export function verifyBatchItem(
  commitment: Commitment,
  index: number,
  value: string,
  nonce: Nonce,
  proof: string[]  // Merkle proof
): boolean {
  const nonceBytes = hexToBytes(nonce);
  const valueBytes = new TextEncoder().encode(value);

  const combined = new Uint8Array(nonceBytes.length + valueBytes.length);
  combined.set(nonceBytes, 0);
  combined.set(valueBytes, nonceBytes.length);

  const leaf = blake3Hash(combined);

  // Verify Merkle proof
  let current = leaf;
  for (let i = 0; i < proof.length; i++) {
    const sibling = hexToBytes(proof[i]);
    const isLeft = (index >> i) & 1;

    if (isLeft) {
      current = blake3Hash(new Uint8Array([...sibling, ...current]));
    } else {
      current = blake3Hash(new Uint8Array([...current, ...sibling]));
    }
  }

  return bytesToHex(current) === commitment;
}

// ============ Timed Commitment ============

/**
 * Commitment with time-lock
 * Value can only be revealed after deadline
 */
export interface TimedCommitment {
  commitment: Commitment;
  nonce: Nonce;
  value: string;
  revealAfter: number;  // Unix timestamp
}

/**
 * Create timed commitment
 */
export function commitTimed(value: string, revealAfter: number): TimedCommitment {
  const { commitment, nonce } = commit(value);

  return {
    commitment,
    nonce,
    value,
    revealAfter,
  };
}

/**
 * Check if timed commitment can be revealed
 */
export function canReveal(timedCommitment: TimedCommitment): boolean {
  return Date.now() >= timedCommitment.revealAfter;
}

// ============ Helper Functions ============

/**
 * Build Merkle root from leaves
 */
function buildMerkleRoot(leaves: Uint8Array[]): Uint8Array {
  if (leaves.length === 0) {
    return new Uint8Array(32);
  }
  if (leaves.length === 1) {
    return leaves[0];
  }

  // Pad to power of 2
  const paddedLeaves = [...leaves];
  while (paddedLeaves.length & (paddedLeaves.length - 1)) {
    paddedLeaves.push(new Uint8Array(32));
  }

  let level = paddedLeaves;
  while (level.length > 1) {
    const nextLevel: Uint8Array[] = [];
    for (let i = 0; i < level.length; i += 2) {
      const combined = new Uint8Array([...level[i], ...level[i + 1]]);
      nextLevel.push(blake3Hash(combined));
    }
    level = nextLevel;
  }

  return level[0];
}

/**
 * Convert bigint to bytes
 */
function bigintToBytes(num: bigint): Uint8Array {
  const hex = num.toString(16).padStart(64, '0');
  return hexToBytes(hex);
}
