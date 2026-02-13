/**
 * Cryptographic Primitives
 *
 * Core cryptographic operations for Legible Money.
 * Uses @noble libraries for audited, pure-JS implementations.
 */

// Key types and generation from keys.ts
export {
  type PrivateKey,
  type PublicKey,
  type PrivateKeyBytes,
  type PublicKeyBytes,
  type Keypair,
  type KeypairBytes,
  generateKeypair,
  generateKeypairBytes,
  generateKeypairFromSeed,
  generateKeypairBytesFromSeed,
  getPublicKey,
  getPublicKeyFromBytes,
  publicKeyToAddress,
  bytesToHex,
  hexToBytes,
} from "./keys.js";

// Signature types and operations from signatures.ts (canonical)
export {
  type Signature,
  type SignatureBytes,
  type SignedMessage,
  type SignatureVerification,
  type AggregatedSignature,
  sign,
  signString,
  signDomainSeparated,
  signTransaction,
  signBlock,
  verify,
  verifyString,
  verifyDomainSeparated,
  verifyTransaction,
  verifyBlock,
  createSignedMessage,
  verifySignedMessage,
  batchVerify,
  aggregateSignatures,
  verifyAggregatedSignature,
} from "./signatures.js";

export * from "./hashing.js";
export * from "./vrf.js";
export * from "./merkle.js";
export * from "./commitment.js";
export * from "./cipher.js";
