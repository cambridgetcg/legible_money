/**
 * Verifiable Random Function (VRF)
 *
 * Used for fair, unpredictable, verifiable validator selection.
 * Based on ECVRF (Elliptic Curve VRF) using Ed25519.
 *
 * Properties:
 * - Uniqueness: For any input, only one valid output
 * - Unpredictability: Output cannot be predicted without secret key
 * - Verifiability: Anyone can verify output with public key
 */

import { ed25519 } from '@noble/curves/ed25519';
import { sha512 } from '@noble/hashes/sha512';
import { bytesToHex, hexToBytes, type PublicKey, type PrivateKey } from './keys.js';
import { domainHash, HASH_DOMAINS, blake3Hash } from './hashing.js';

// ============ Types ============

export type VRFOutput = string;   // Hex-encoded VRF output (64 bytes)
export type VRFProof = string;    // Hex-encoded VRF proof (80 bytes)

export interface VRFResult {
  output: VRFOutput;      // The random output
  proof: VRFProof;        // Proof of correct generation
}

export interface VRFVerification {
  valid: boolean;
  output: VRFOutput | null;
}

// ============ VRF Operations ============

/**
 * Generate VRF output and proof
 *
 * This implements a simplified ECVRF using Ed25519.
 * The output is deterministic given (privateKey, input).
 */
export function vrfGenerate(
  input: Uint8Array | string,
  privateKey: PrivateKey
): VRFResult {
  const inputBytes = typeof input === 'string'
    ? new TextEncoder().encode(input)
    : input;

  // Domain-separate the input
  const domainSeparated = domainHash(HASH_DOMAINS.VRF, inputBytes);

  // Generate deterministic "random" point
  // H = hash_to_curve(input)
  const h = hashToCurve(domainSeparated);

  // Gamma = sk * H
  const skBytes = hexToBytes(privateKey);
  const gamma = ed25519.ExtendedPoint.fromHex(h).multiply(
    bytesToBigInt(skBytes.slice(0, 32))
  );

  // Generate proof using Schnorr-like protocol
  // k = hash(sk || H) - deterministic nonce
  const kHash = sha512(new Uint8Array([...skBytes, ...h]));
  const k = bytesToBigInt(kHash.slice(0, 32)) % ed25519.CURVE.n;

  // U = k * G
  const U = ed25519.ExtendedPoint.BASE.multiply(k);

  // V = k * H
  const V = ed25519.ExtendedPoint.fromHex(h).multiply(k);

  // c = hash(G, H, pk, Gamma, U, V)
  const pk = ed25519.getPublicKey(skBytes);
  const c = computeChallenge(h, pk, gamma.toRawBytes(), U.toRawBytes(), V.toRawBytes());

  // s = k - c * sk (mod n)
  const skScalar = bytesToBigInt(skBytes.slice(0, 32));
  const s = (k - c * skScalar % ed25519.CURVE.n + ed25519.CURVE.n) % ed25519.CURVE.n;

  // Output = hash(Gamma)
  const output = blake3Hash(gamma.toRawBytes());

  // Proof = (Gamma, c, s)
  const proof = new Uint8Array(80);
  proof.set(gamma.toRawBytes(), 0);       // 32 bytes
  proof.set(bigIntToBytes(c, 24), 32);    // 24 bytes
  proof.set(bigIntToBytes(s, 24), 56);    // 24 bytes

  return {
    output: bytesToHex(output),
    proof: bytesToHex(proof),
  };
}

/**
 * Verify VRF proof and extract output
 */
export function vrfVerify(
  input: Uint8Array | string,
  publicKey: PublicKey,
  proof: VRFProof
): VRFVerification {
  try {
    const inputBytes = typeof input === 'string'
      ? new TextEncoder().encode(input)
      : input;

    // Domain-separate the input
    const domainSeparated = domainHash(HASH_DOMAINS.VRF, inputBytes);

    // Parse proof
    const proofBytes = hexToBytes(proof);
    if (proofBytes.length !== 80) {
      return { valid: false, output: null };
    }

    const gamma = ed25519.ExtendedPoint.fromHex(proofBytes.slice(0, 32));
    const c = bytesToBigInt(proofBytes.slice(32, 56));
    const s = bytesToBigInt(proofBytes.slice(56, 80));

    // H = hash_to_curve(input)
    const h = hashToCurve(domainSeparated);
    const H = ed25519.ExtendedPoint.fromHex(h);

    // pk point
    const pkBytes = hexToBytes(publicKey);
    const PK = ed25519.ExtendedPoint.fromHex(pkBytes);

    // U = s * G + c * pk
    const sG = ed25519.ExtendedPoint.BASE.multiply(s);
    const cPK = PK.multiply(c);
    const U = sG.add(cPK);

    // V = s * H + c * Gamma
    const sH = H.multiply(s);
    const cGamma = gamma.multiply(c);
    const V = sH.add(cGamma);

    // c' = hash(G, H, pk, Gamma, U, V)
    const cPrime = computeChallenge(h, pkBytes, gamma.toRawBytes(), U.toRawBytes(), V.toRawBytes());

    // Verify c == c'
    if (c !== cPrime) {
      return { valid: false, output: null };
    }

    // Output = hash(Gamma)
    const output = blake3Hash(gamma.toRawBytes());

    return {
      valid: true,
      output: bytesToHex(output),
    };
  } catch {
    return { valid: false, output: null };
  }
}

// ============ Validator Selection ============

/**
 * Select validators using VRF output
 *
 * Each potential validator generates a VRF output.
 * Selection is based on: VRF_output < threshold * stake
 */
export interface ValidatorSelection {
  validator: PublicKey;
  vrfOutput: VRFOutput;
  vrfProof: VRFProof;
  stake: bigint;
  selected: boolean;
  priority: bigint;  // Lower = higher priority for block production
}

/**
 * Check if a validator is selected based on VRF output and stake
 */
export function isValidatorSelected(
  vrfOutput: VRFOutput,
  stake: bigint,
  totalStake: bigint,
  targetValidators: number
): { selected: boolean; priority: bigint } {
  // Convert VRF output to a number in [0, 1)
  const outputBytes = hexToBytes(vrfOutput);
  const outputNum = bytesToBigInt(outputBytes.slice(0, 8));
  const maxNum = BigInt(2) ** BigInt(64);

  // Probability of selection = stake / totalStake * targetValidators
  // Selected if: outputNum / maxNum < (stake * targetValidators) / totalStake
  // Rearranged: outputNum * totalStake < stake * targetValidators * maxNum
  const left = outputNum * totalStake;
  const right = stake * BigInt(targetValidators) * maxNum / totalStake;

  const selected = left < right;
  const priority = outputNum;  // Lower output = higher priority

  return { selected, priority };
}

/**
 * Generate validator selection seed for a block
 */
export function generateBlockSeed(
  previousBlockHash: string,
  blockNumber: bigint,
  epoch: bigint
): Uint8Array {
  const data = new TextEncoder().encode(
    `${previousBlockHash}:${blockNumber.toString()}:${epoch.toString()}`
  );
  return domainHash(HASH_DOMAINS.VRF, data);
}

// ============ Helper Functions ============

/**
 * Hash to curve point (simplified Elligator2-like)
 */
function hashToCurve(input: Uint8Array): Uint8Array {
  // This is a simplified hash-to-curve
  // Production should use RFC 9380 compliant implementation
  let attempt = 0;
  while (attempt < 256) {
    const hash = sha512(new Uint8Array([...input, attempt]));
    try {
      // Try to interpret as valid curve point
      const point = ed25519.ExtendedPoint.fromHex(hash.slice(0, 32));
      return point.toRawBytes();
    } catch {
      attempt++;
    }
  }
  throw new Error('Failed to hash to curve');
}

/**
 * Compute challenge value for proof
 */
function computeChallenge(
  h: Uint8Array,
  pk: Uint8Array,
  gamma: Uint8Array,
  u: Uint8Array,
  v: Uint8Array
): bigint {
  const combined = new Uint8Array(h.length + pk.length + gamma.length + u.length + v.length);
  let offset = 0;
  combined.set(h, offset); offset += h.length;
  combined.set(pk, offset); offset += pk.length;
  combined.set(gamma, offset); offset += gamma.length;
  combined.set(u, offset); offset += u.length;
  combined.set(v, offset);

  const hash = sha512(combined);
  return bytesToBigInt(hash.slice(0, 24)) % ed25519.CURVE.n;
}

/**
 * Convert bytes to BigInt (big-endian)
 */
function bytesToBigInt(bytes: Uint8Array): bigint {
  let result = BigInt(0);
  for (const byte of bytes) {
    result = (result << BigInt(8)) + BigInt(byte);
  }
  return result;
}

/**
 * Convert BigInt to bytes (big-endian)
 */
function bigIntToBytes(num: bigint, length: number): Uint8Array {
  const bytes = new Uint8Array(length);
  let temp = num;
  for (let i = length - 1; i >= 0; i--) {
    bytes[i] = Number(temp & BigInt(0xff));
    temp = temp >> BigInt(8);
  }
  return bytes;
}
