/**
 * Foundation Primitive Types
 *
 * TYPE-1: Shared types used across modules MUST be defined in foundation layer.
 * These are the atomic building blocks used throughout Legible Money.
 *
 * Design principle: All primitive types are branded strings for:
 * - JSON serialization compatibility
 * - Type safety through nominal typing
 * - Human readability in logs/debugging
 */

// ============ Branding Utility ============

/**
 * Brand a type for nominal typing without runtime overhead
 */
declare const __brand: unique symbol;
type Brand<T, B> = T & { readonly [__brand]: B };

// ============ Cryptographic Primitives ============

/**
 * Hex-encoded private key (64 chars = 32 bytes)
 * NEVER expose in logs, serialization, or network transmission
 */
export type PrivateKey = Brand<string, "PrivateKey">;

/**
 * Hex-encoded public key (64 chars = 32 bytes)
 * Safe to share, used for identification and verification
 */
export type PublicKey = Brand<string, "PublicKey">;

/**
 * Hex-encoded Ed25519 signature (128 chars = 64 bytes)
 */
export type Signature = Brand<string, "Signature">;

/**
 * Hex-encoded hash (variable length, typically 64 chars = 32 bytes)
 */
export type Hash = Brand<string, "Hash">;

/**
 * Merkle tree root hash
 */
export type MerkleRoot = Brand<string, "MerkleRoot">;

/**
 * VRF output - deterministic randomness from validator's key
 */
export type VRFOutput = Brand<string, "VRFOutput">;

/**
 * VRF proof - proves VRF output was computed correctly
 */
export type VRFProof = Brand<string, "VRFProof">;

/**
 * Commitment hash for commit-reveal schemes
 */
export type Commitment = Brand<string, "Commitment">;

// ============ Identity Primitives ============

/**
 * Decentralized Identifier following W3C DID spec
 * Format: did:lgm:<method-specific-id>
 *
 * In Legible Money:
 * - did:lgm:pk:<publicKey> - Derived from public key
 * - did:lgm:claim:<claimHash> - References a verified claim
 */
export type DID = Brand<string, "DID">;

/**
 * Agent address (shorter form for on-chain use)
 * Format: lgm1<40-char-hex>
 */
export type Address = Brand<string, "Address">;

// ============ Temporal Primitives ============

/**
 * Block height (0-indexed, monotonically increasing)
 */
export type BlockHeight = bigint;

/**
 * Unix timestamp in milliseconds
 */
export type Timestamp = number;

/**
 * Duration in blocks
 */
export type BlockDuration = bigint;

/**
 * Epoch number (groups of blocks)
 */
export type Epoch = bigint;

// ============ Economic Primitives ============

/**
 * Token amount in smallest unit (1 LGM = 10^18 base units)
 * Always use bigint for economic values to prevent precision loss
 */
export type Amount = bigint;

/**
 * Transaction fee
 */
export type Fee = bigint;

/**
 * Staked amount
 */
export type Stake = bigint;

/**
 * LGM token decimals
 */
export const LGM_DECIMALS = 18;

/**
 * One LGM in base units
 */
export const ONE_LGM = 10n ** BigInt(LGM_DECIMALS);

// ============ Confidence Primitives ============

/**
 * Confidence level (0.0 to 1.0)
 * Represents epistemic certainty in a claim
 */
export type Confidence = number;

/**
 * Confidence validation
 */
export function isValidConfidence(value: number): value is Confidence {
  return value >= 0.0 && value <= 1.0;
}

// ============ Protocol Constants ============

/**
 * Protocol constants
 */
export const PROTOCOL_CONSTANTS = {
  /** Target block time in milliseconds */
  TARGET_BLOCK_TIME_MS: 2521,

  /** Finality depth in blocks */
  FINALITY_DEPTH: 19,

  /** Token symbol */
  TOKEN_SYMBOL: "LGM",

  /** Token name */
  TOKEN_NAME: "Legible Money",

  /** DID method */
  DID_METHOD: "lgm",
} as const;

// ============ Type Guards ============

/**
 * Check if string is valid hex
 */
export function isHex(value: string): boolean {
  return /^[0-9a-f]*$/i.test(value);
}

/**
 * Check if value is valid public key format
 */
export function isPublicKey(value: string): value is PublicKey {
  return value.length === 64 && isHex(value);
}

/**
 * Check if value is valid signature format
 */
export function isSignature(value: string): value is Signature {
  return value.length === 128 && isHex(value);
}

/**
 * Check if value is valid DID format
 */
export function isDID(value: string): value is DID {
  return value.startsWith("did:lgm:");
}

/**
 * Check if value is valid address format
 */
export function isAddress(value: string): value is Address {
  return (
    value.startsWith("lgm1") && value.length === 44 && isHex(value.slice(4))
  );
}

// ============ Conversion Utilities ============

/**
 * Convert public key to DID
 */
export function publicKeyToDID(publicKey: PublicKey): DID {
  return `did:lgm:pk:${publicKey}` as DID;
}

/**
 * Extract public key from DID (if pk-based)
 */
export function didToPublicKey(did: DID): PublicKey | null {
  const match = did.match(/^did:lgm:pk:([0-9a-f]{64})$/i);
  return match ? (match[1].toLowerCase() as PublicKey) : null;
}

/**
 * Format amount for display (with decimals)
 */
export function formatAmount(amount: Amount, decimals = 4): string {
  const divisor = ONE_LGM;
  const whole = amount / divisor;
  const remainder = amount % divisor;
  const fractional = remainder
    .toString()
    .padStart(LGM_DECIMALS, "0")
    .slice(0, decimals);
  return `${whole}.${fractional} LGM`;
}

/**
 * Parse amount from display format
 */
export function parseAmount(display: string): Amount {
  const cleaned = display.replace(/[^0-9.]/g, "");
  const [whole, fractional = ""] = cleaned.split(".");
  const paddedFractional = fractional
    .padEnd(LGM_DECIMALS, "0")
    .slice(0, LGM_DECIMALS);
  return BigInt(whole) * ONE_LGM + BigInt(paddedFractional);
}
