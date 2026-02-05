/**
 * Foundation Crypto Module
 *
 * ARCH-2: Module Boundary Contracts
 * This is the public API for cryptographic operations.
 *
 * Re-exports from core/crypto with foundation-typed interfaces.
 * This allows gradual migration without breaking existing code.
 */

// Re-export from core/crypto (implementation)
export {
  // Key operations
  generateKeypair,
  generateKeypairBytes,
  generateKeypairFromSeed,
  generateKeypairBytesFromSeed,
  getPublicKey,
  getPublicKeyFromBytes,
  publicKeyToAddress,
  bytesToHex,
  hexToBytes,
  type Keypair,
  type KeypairBytes,
  type PrivateKeyBytes,
  type PublicKeyBytes,
  type SignatureBytes,
} from "../../core/crypto/keys.js";

// Signature operations
export {
  sign,
  signBytes,
  signString,
  verify,
  verifyBytes,
} from "../../core/crypto/keys.js";

// Domain-separated signatures (from signatures.ts)
export {
  signDomainSeparated,
  signTransaction,
  signBlock,
  verifyDomainSeparated,
  verifyTransaction,
  verifyBlock,
  createSignedMessage,
  verifySignedMessage,
  batchVerify,
  aggregateSignatures,
  verifyAggregatedSignature,
  type SignedMessage,
  type SignatureVerification,
  type AggregatedSignature,
} from "../../core/crypto/signatures.js";

// Hashing operations
export {
  sha512Hash,
  sha512Hex,
  sha512_256,
  sha512_256Hex,
  sha256Hash,
  sha256Hex,
  blake3Hash,
  blake3Hex,
  blake3Variable,
  domainHash,
  merkleHash,
  merkleLeafHash,
  merkleInternalHash,
  hashClaim,
  hashFact,
  constantTimeEqual,
  hashesEqual,
  HASH_DOMAINS,
  type HashBytes,
} from "../../core/crypto/hashing.js";

// Merkle tree operations
export {
  InMemorySMT,
  CompactMerkleTree,
  EMPTY_VALUE,
  TREE_DEPTH,
  EMPTY_HASHES,
  hashLeaf,
  hashNode,
  keyToBits,
  type SparseMerkleTree,
  type MerkleProofData,
  type MerkleProof,
  type SMTNode,
} from "../../core/crypto/merkle.js";

// VRF operations
export {
  vrfGenerate,
  vrfVerify,
  isValidatorSelected,
  generateBlockSeed,
  type VRFOutput as VRFOutputType,
  type VRFProof as VRFProofType,
  type VRFResult,
  type VRFVerification,
  type ValidatorSelection,
} from "../../core/crypto/vrf.js";

// Commitment operations
export {
  commit,
  verifyCommitment,
  commitNumeric,
  verifyNumericCommitment,
  commitVerdict,
  verifyVerdictCommitment,
  commitBatch,
  verifyBatchItem,
  commitTimed,
  canReveal,
  type Commitment as CommitmentType,
  type Nonce,
  type RevealedValue,
  type CommitmentData,
  type CommitRevealRound,
  type NumericCommitment,
  type VerdictCommitment,
  type BatchCommitment,
  type TimedCommitment,
} from "../../core/crypto/commitment.js";

// Re-export foundation types for convenience
// Note: VRFOutput, VRFProof, Commitment from types/ are branded versions
// The core/crypto versions are unbranded aliases (VRFOutputType, etc.)
export type {
  PrivateKey,
  PublicKey,
  Signature,
  Hash,
  MerkleRoot,
  VRFOutput,
  VRFProof,
  Commitment,
} from "../types/index.js";
