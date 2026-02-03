/**
 * Hashing Functions
 *
 * Cryptographic hash functions for Legible Money.
 * Primary: SHA-512 (aligned with Ed25519)
 * Secondary: BLAKE3 (for performance-critical paths)
 */

import { sha512 } from '@noble/hashes/sha512';
import { sha256 } from '@noble/hashes/sha256';
import { blake3 } from '@noble/hashes/blake3';
import { bytesToHex, hexToBytes } from './keys.js';

// ============ Types ============

export type Hash = string;  // Hex-encoded hash
export type HashBytes = Uint8Array;

// ============ SHA-512 (Primary) ============

/**
 * SHA-512 hash (primary hash function)
 * Used for: signatures, commitments, fact hashes
 */
export function sha512Hash(data: Uint8Array | string): HashBytes {
  const input = typeof data === 'string' ? new TextEncoder().encode(data) : data;
  return sha512(input);
}

/**
 * SHA-512 hash as hex string
 */
export function sha512Hex(data: Uint8Array | string): Hash {
  return bytesToHex(sha512Hash(data));
}

/**
 * SHA-512/256 (truncated to 256 bits)
 * Used for: addresses, short identifiers
 */
export function sha512_256(data: Uint8Array | string): HashBytes {
  return sha512Hash(data).slice(0, 32);
}

export function sha512_256Hex(data: Uint8Array | string): Hash {
  return bytesToHex(sha512_256(data));
}

// ============ SHA-256 (Compatibility) ============

/**
 * SHA-256 hash (for compatibility)
 */
export function sha256Hash(data: Uint8Array | string): HashBytes {
  const input = typeof data === 'string' ? new TextEncoder().encode(data) : data;
  return sha256(input);
}

export function sha256Hex(data: Uint8Array | string): Hash {
  return bytesToHex(sha256Hash(data));
}

// ============ BLAKE3 (Performance) ============

/**
 * BLAKE3 hash (high performance)
 * Used for: Merkle trees, state hashing
 */
export function blake3Hash(data: Uint8Array | string): HashBytes {
  const input = typeof data === 'string' ? new TextEncoder().encode(data) : data;
  return blake3(input);
}

export function blake3Hex(data: Uint8Array | string): Hash {
  return bytesToHex(blake3Hash(data));
}

/**
 * BLAKE3 with custom output length
 */
export function blake3Variable(data: Uint8Array | string, outputLength: number): HashBytes {
  const input = typeof data === 'string' ? new TextEncoder().encode(data) : data;
  return blake3(input, { dkLen: outputLength });
}

// ============ Domain-Separated Hashing ============

/**
 * Domain-separated hash to prevent cross-protocol attacks
 * Format: SHA-512(domain || length || data)
 */
export function domainHash(domain: string, data: Uint8Array | string): HashBytes {
  const domainBytes = new TextEncoder().encode(domain);
  const dataBytes = typeof data === 'string' ? new TextEncoder().encode(data) : data;

  // Prefix with domain and length to prevent ambiguity
  const lengthBytes = new Uint8Array(4);
  new DataView(lengthBytes.buffer).setUint32(0, domainBytes.length, false);

  const combined = new Uint8Array(domainBytes.length + 4 + dataBytes.length);
  combined.set(domainBytes, 0);
  combined.set(lengthBytes, domainBytes.length);
  combined.set(dataBytes, domainBytes.length + 4);

  return sha512Hash(combined);
}

/**
 * Predefined domains for Legible Money
 */
export const HASH_DOMAINS = {
  BLOCK: 'LGM.block.v1',
  TRANSACTION: 'LGM.tx.v1',
  STATE: 'LGM.state.v1',
  FACT: 'LGM.fact.v1',
  AXIOM: 'LGM.axiom.v1',
  DERIVATION: 'LGM.derivation.v1',
  COMMITMENT: 'LGM.commitment.v1',
  VRF: 'LGM.vrf.v1',
  SIGNATURE: 'LGM.sig.v1',
} as const;

// ============ Merkle Hashing ============

/**
 * Hash two nodes together for Merkle tree
 * Uses BLAKE3 for performance
 */
export function merkleHash(left: HashBytes, right: HashBytes): HashBytes {
  const combined = new Uint8Array(left.length + right.length);
  combined.set(left, 0);
  combined.set(right, left.length);
  return blake3Hash(combined);
}

/**
 * Hash a leaf node (with domain separation)
 */
export function merkleLeafHash(data: Uint8Array): HashBytes {
  // Prefix with 0x00 to distinguish leaves from internal nodes
  const prefixed = new Uint8Array(1 + data.length);
  prefixed[0] = 0x00;
  prefixed.set(data, 1);
  return blake3Hash(prefixed);
}

/**
 * Hash an internal node (with domain separation)
 */
export function merkleInternalHash(left: HashBytes, right: HashBytes): HashBytes {
  // Prefix with 0x01 to distinguish from leaves
  const combined = new Uint8Array(1 + left.length + right.length);
  combined[0] = 0x01;
  combined.set(left, 1);
  combined.set(right, 1 + left.length);
  return blake3Hash(combined);
}

// ============ Claim Hashing ============

/**
 * Normalize and hash a claim for consistent identification
 */
export function hashClaim(claim: string): Hash {
  // Normalize: lowercase, trim, collapse whitespace
  const normalized = claim.toLowerCase().trim().replace(/\s+/g, ' ');
  return bytesToHex(domainHash(HASH_DOMAINS.FACT, normalized));
}

/**
 * Hash a fact for on-chain storage
 */
export function hashFact(fact: {
  claim: string;
  type: string;
  domain: string;
  submittedBy: string;
  references: string[];
}): Hash {
  const data = JSON.stringify({
    claim: hashClaim(fact.claim),
    type: fact.type,
    domain: fact.domain,
    submittedBy: fact.submittedBy,
    references: fact.references.sort(),
  });
  return bytesToHex(domainHash(HASH_DOMAINS.FACT, data));
}

// ============ Utilities ============

/**
 * Constant-time comparison to prevent timing attacks
 */
export function constantTimeEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;

  let result = 0;
  for (let i = 0; i < a.length; i++) {
    result |= a[i] ^ b[i];
  }
  return result === 0;
}

/**
 * Compare two hex hashes
 */
export function hashesEqual(a: Hash, b: Hash): boolean {
  return constantTimeEqual(hexToBytes(a), hexToBytes(b));
}
