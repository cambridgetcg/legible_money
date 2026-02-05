/**
 * Foundation Types
 *
 * ARCH-2: Module Boundary Contracts
 * Each module MUST expose only through its index.ts.
 *
 * This file is the public API for foundation/types.
 * Internal reorganization will not break consumers.
 */

// Primitive types (most commonly used)
export {
  // Cryptographic
  type PrivateKey,
  type PublicKey,
  type Signature,
  type Hash,
  type MerkleRoot,
  type VRFOutput,
  type VRFProof,
  type Commitment,

  // Identity
  type DID,
  type Address,

  // Temporal
  type BlockHeight,
  type Timestamp,
  type BlockDuration,
  type Epoch,

  // Economic
  type Amount,
  type Fee,
  type Stake,
  type Confidence,
  LGM_DECIMALS,
  ONE_LGM,

  // Protocol constants
  PROTOCOL_CONSTANTS,

  // Type guards
  isHex,
  isPublicKey,
  isSignature,
  isDID,
  isAddress,
  isValidConfidence,

  // Conversions
  publicKeyToDID,
  didToPublicKey,
  formatAmount,
  parseAmount,
} from "./primitives.js";

// Error types
export {
  LegibleError,
  CryptoError,
  ValidationError,
  SerializationError,
  type CryptoErrorCode,
  type ValidationErrorCode,
  type SerializationErrorCode,
  isLegibleError,
  isCryptoError,
  isValidationError,
  isSerializationError,
  wrapError,
  assertValid,
} from "./errors.js";
