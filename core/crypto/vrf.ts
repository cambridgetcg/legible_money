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

import { ed25519 } from "@noble/curves/ed25519";
import { sha512 } from "@noble/hashes/sha512";
import {
  bytesToHex,
  hexToBytes,
  type PublicKey,
  type PrivateKey,
} from "./keys.js";
import { domainHash, HASH_DOMAINS, sha256Hash } from "./hashing.js";

// ============ Types ============

export type VRFOutput = string; // Hex-encoded VRF output (64 bytes)
export type VRFProof = string; // Hex-encoded VRF proof (96 bytes: 32 Gamma + 32 c + 32 s)

export interface VRFResult {
  output: VRFOutput; // The random output
  proof: VRFProof; // Proof of correct generation
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
  privateKey: PrivateKey,
): VRFResult {
  const inputBytes =
    typeof input === "string" ? new TextEncoder().encode(input) : input;

  // Domain-separate the input
  const domainSeparated = domainHash(HASH_DOMAINS.VRF, inputBytes);

  // Generate deterministic "random" point
  // H = hash_to_curve(input)
  const h = hashToCurve(domainSeparated);

  // Derive the actual scalar from the private key seed
  // Ed25519 keys: seed (32 bytes) -> SHA-512 -> first 32 bytes clamped = scalar
  const skBytes = hexToBytes(privateKey);
  const skScalar = deriveScalarFromSeed(skBytes);

  // Gamma = sk * H
  const gamma = ed25519.ExtendedPoint.fromHex(h).multiply(skScalar);

  // Generate proof using Schnorr-like protocol
  // k = hash(sk || H) - deterministic nonce
  const kHash = sha512(new Uint8Array([...skBytes, ...h]));
  const k = bytesToBigInt(kHash.slice(0, 32)) % ed25519.CURVE.n;

  // Ensure k is in valid range (1 <= k < n)
  const kClamped = k === 0n ? 1n : k;

  // U = k * G
  const U = ed25519.ExtendedPoint.BASE.multiply(kClamped);

  // V = k * H
  const V = ed25519.ExtendedPoint.fromHex(h).multiply(kClamped);

  // c = hash(G, H, pk, Gamma, U, V)
  const pk = ed25519.getPublicKey(skBytes);
  const c = computeChallenge(
    h,
    pk,
    gamma.toRawBytes(),
    U.toRawBytes(),
    V.toRawBytes(),
  );

  // s = k - c * sk (mod n)
  // Use proper modular arithmetic to handle potential negative values
  const n = ed25519.CURVE.n;
  const cTimessk = (c * skScalar) % n;
  const diff = kClamped - cTimessk;
  const s = ((diff % n) + n) % n;

  // Output = SHA-256(Gamma) — must match Go VRF (x/knowledge/crypto/vrf.go)
  const output = sha256Hash(gamma.toRawBytes());

  // Proof = (Gamma, c, s) - 96 bytes total
  const proof = new Uint8Array(96);
  proof.set(gamma.toRawBytes(), 0); // 32 bytes
  proof.set(bigIntToBytes(c, 32), 32); // 32 bytes (c is mod n, needs full 32 bytes)
  proof.set(bigIntToBytes(s, 32), 64); // 32 bytes (s is mod n, needs full 32 bytes)

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
  proof: VRFProof,
): VRFVerification {
  try {
    const inputBytes =
      typeof input === "string" ? new TextEncoder().encode(input) : input;

    // Domain-separate the input
    const domainSeparated = domainHash(HASH_DOMAINS.VRF, inputBytes);

    // Parse proof (96 bytes: 32 Gamma + 32 c + 32 s)
    const proofBytes = hexToBytes(proof);
    if (proofBytes.length !== 96) {
      return { valid: false, output: null };
    }

    const gamma = ed25519.ExtendedPoint.fromHex(proofBytes.slice(0, 32));
    const c = bytesToBigInt(proofBytes.slice(32, 64));
    const s = bytesToBigInt(proofBytes.slice(64, 96));

    // Validate c and s are in valid range
    if (c >= ed25519.CURVE.n || s >= ed25519.CURVE.n) {
      return { valid: false, output: null };
    }

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
    // Since Gamma = sk * H, we have c * Gamma = c * sk * H
    // So s * H + c * Gamma = (s + c*sk) * H = k * H
    const sH = H.multiply(s);
    const cGamma = gamma.multiply(c);
    const V = sH.add(cGamma);

    // c' = hash(G, H, pk, Gamma, U, V)
    const cPrime = computeChallenge(
      h,
      pkBytes,
      gamma.toRawBytes(),
      U.toRawBytes(),
      V.toRawBytes(),
    );

    // Verify c == c'
    if (c !== cPrime) {
      return { valid: false, output: null };
    }

    // Output = SHA-256(Gamma) — must match Go VRF (x/knowledge/crypto/vrf.go)
    const output = sha256Hash(gamma.toRawBytes());

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
  priority: bigint; // Lower = higher priority for block production
}

/**
 * Check if a validator is selected based on VRF output and stake
 */
export function isValidatorSelected(
  vrfOutput: VRFOutput,
  stake: bigint,
  totalStake: bigint,
  targetValidators: number,
): { selected: boolean; priority: bigint } {
  // Convert VRF output to a number in [0, 1)
  const outputBytes = hexToBytes(vrfOutput);
  const outputNum = bytesToBigInt(outputBytes.slice(0, 8));
  const maxNum = BigInt(2) ** BigInt(64);

  // Probability of selection = stake / totalStake * targetValidators
  // Selected if: outputNum / maxNum < (stake * targetValidators) / totalStake
  // Rearranged: outputNum * totalStake < stake * targetValidators * maxNum
  const left = outputNum * totalStake;
  const right = stake * BigInt(targetValidators) * maxNum;

  const selected = left < right;
  const priority = outputNum; // Lower output = higher priority

  return { selected, priority };
}

/**
 * Generate validator selection seed for a block
 *
 * VULN-015 FIX: Includes multiple entropy sources to prevent seed grinding attacks.
 * The seed combines:
 * - Previous block hash (committed before selection)
 * - Block number (deterministic ordering)
 * - Epoch (time-based partitioning)
 * - Optional accumulated randomness from past VRF outputs
 *
 * This makes it computationally infeasible to predict which validator will be
 * selected for a future block, even with partial control over block content.
 */
export function generateBlockSeed(
  previousBlockHash: string,
  blockNumber: bigint,
  epoch: bigint,
  accumulatedRandomness?: string,
): Uint8Array {
  // Combine all entropy sources
  const seedData = {
    previousBlockHash,
    blockNumber: blockNumber.toString(),
    epoch: epoch.toString(),
    // Include accumulated randomness if available (from previous VRF outputs)
    accumulatedRandomness: accumulatedRandomness || "genesis",
  };

  const data = new TextEncoder().encode(JSON.stringify(seedData));
  return domainHash(HASH_DOMAINS.VRF, data);
}

/**
 * Accumulate VRF randomness from block validators
 *
 * VULN-015 FIX: Combines VRF outputs from multiple blocks to create
 * unpredictable accumulated randomness. An attacker would need to control
 * multiple consecutive validators to influence the selection.
 *
 * @param currentAccumulator - Current accumulated randomness
 * @param vrfOutput - New VRF output to mix in
 * @param blockNumber - Block number for ordering
 * @returns New accumulated randomness
 */
export function accumulateRandomness(
  currentAccumulator: string,
  vrfOutput: string,
  blockNumber: bigint,
): string {
  const combined = new TextEncoder().encode(
    `${currentAccumulator}:${vrfOutput}:${blockNumber.toString()}`,
  );
  return bytesToHex(domainHash("LGM.vrf.accumulator.v1", combined));
}

// ============ Helper Functions ============

/**
 * Derive the scalar from an Ed25519 seed
 * Ed25519 key derivation: seed (32 bytes) -> SHA-512 -> first 32 bytes clamped
 * Clamping sets bits 0,1,2 to 0 and bit 254 to 1, bit 255 to 0
 */
function deriveScalarFromSeed(seed: Uint8Array): bigint {
  // Hash the seed with SHA-512
  const h = sha512(seed);
  const scalar = h.slice(0, 32);

  // Clamp the scalar (standard Ed25519 clamping)
  scalar[0] &= 248; // Clear bits 0, 1, 2
  scalar[31] &= 127; // Clear bit 255
  scalar[31] |= 64; // Set bit 254

  // Convert to little-endian BigInt (Ed25519 uses little-endian)
  let result = bytesToBigIntLE(scalar);

  // Reduce modulo curve order to ensure valid scalar
  // The clamped value can be > curve.n, so we must reduce
  result = result % ed25519.CURVE.n;

  // Ensure non-zero (though extremely unlikely)
  if (result === 0n) {
    result = 1n;
  }

  return result;
}

/**
 * Convert bytes to BigInt (little-endian, for Ed25519)
 */
function bytesToBigIntLE(bytes: Uint8Array): bigint {
  let result = BigInt(0);
  for (let i = bytes.length - 1; i >= 0; i--) {
    result = (result << BigInt(8)) + BigInt(bytes[i]);
  }
  return result;
}

/**
 * Hash to curve point (simplified Elligator2-like)
 *
 * IMPORTANT: We multiply by the cofactor (8) to ensure the point is in the
 * prime-order subgroup. This is necessary because Ed25519 has cofactor 8,
 * and points outside the prime-order subgroup can cause verification failures
 * due to the non-associativity of scalar multiplication with low-order components.
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
      // Multiply by cofactor (8) to clear low-order components
      // This ensures the point is in the prime-order subgroup
      const clearedPoint = point.multiply(8n);
      // Check the result is not the identity (unlikely but possible)
      if (clearedPoint.equals(ed25519.ExtendedPoint.ZERO)) {
        attempt++;
        continue;
      }
      return clearedPoint.toRawBytes();
    } catch {
      attempt++;
    }
  }
  throw new Error("Failed to hash to curve");
}

/**
 * Compute challenge value for proof
 */
function computeChallenge(
  h: Uint8Array,
  pk: Uint8Array,
  gamma: Uint8Array,
  u: Uint8Array,
  v: Uint8Array,
): bigint {
  const combined = new Uint8Array(
    h.length + pk.length + gamma.length + u.length + v.length,
  );
  let offset = 0;
  combined.set(h, offset);
  offset += h.length;
  combined.set(pk, offset);
  offset += pk.length;
  combined.set(gamma, offset);
  offset += gamma.length;
  combined.set(u, offset);
  offset += u.length;
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
