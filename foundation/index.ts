/**
 * Legible Money Foundation Layer
 *
 * ARCH-1: Layer Isolation
 * Foundation layer has NO internal dependencies.
 * All other layers depend on this.
 *
 * ARCH-2: Module Boundary Contracts
 * This is the sole entry point for foundation layer.
 *
 * Usage:
 *   import { PublicKey, generateKeypair, bytesToHex } from '@legible/foundation';
 *
 * The foundation layer provides:
 * - Primitive types (PublicKey, Signature, DID, etc.)
 * - Cryptographic operations (signing, hashing, Merkle trees)
 * - Pure utility functions (byte manipulation, serialization)
 * - Error types for structured error handling
 */

// ============ Types ============
// Primitive types used across all layers

export {
  // Cryptographic primitives
  type PrivateKey,
  type PublicKey,
  type Signature,
  type Hash,
  type MerkleRoot,
  type VRFOutput,
  type VRFProof,
  type Commitment,

  // Identity primitives
  type DID,
  type Address,

  // Temporal primitives
  type BlockHeight,
  type Timestamp,
  type BlockDuration,
  type Epoch,

  // Economic primitives
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

  // Errors
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
} from "./types/index.js";

// ============ Crypto ============
// Cryptographic operations

export {
  // Key generation
  generateKeypair,
  generateKeypairBytes,
  generateKeypairFromSeed,
  generateKeypairBytesFromSeed,
  getPublicKey,
  getPublicKeyFromBytes,
  publicKeyToAddress,
  type Keypair,
  type KeypairBytes,
  type PrivateKeyBytes,
  type PublicKeyBytes,
  type SignatureBytes,

  // Signing
  sign,
  signBytes,
  signString,
  signDomainSeparated,
  signTransaction,
  signBlock,

  // Verification
  verify,
  verifyBytes,
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

  // Hashing
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

  // Merkle trees
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

  // VRF
  vrfGenerate,
  vrfVerify,
  isValidatorSelected,
  generateBlockSeed,
  type VRFResult,
  type VRFVerification,
  type ValidatorSelection as VRFValidatorSelection,

  // Commitments
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
  type CommitmentType,
  type Nonce,
  type RevealedValue,
  type CommitmentData,
  type CommitRevealRound,
  type NumericCommitment,
  type VerdictCommitment,
  type BatchCommitment,
  type TimedCommitment,
} from "./crypto/index.js";

// ============ Utils ============
// Pure utility functions

export {
  // Byte operations
  bytesToHex,
  hexToBytes,
  concatBytes,
  bytesEqual,
  // Note: constantTimeEqual already exported from crypto

  // BigInt operations
  bigintMin,
  bigintMax,
  bigintClamp,
  bigintDiv,
  bigintPercent,

  // String operations
  normalizeWhitespace,
  truncate,
  truncateHex,

  // Array operations
  chunk,
  unique,
  groupBy,

  // Object operations
  deepFreeze,
  pick,
  omit,

  // Time operations
  sleep,
  now,
  msToBlocks,
  blocksToMs,

  // Validation
  assert,
  assertDefined,

  // Serialization
  deterministicStringify,
  bigintReplacer,
  bigintReviver,
} from "./utils/index.js";

// ============ Contracts ============
// Domain and infrastructure interfaces for dependency injection

export {
  // Infrastructure Contracts
  type AccountReader,
  type AccountWriter,
  type ValidatorReader,
  type ValidatorWriter,
  type KnowledgeReader,
  type KnowledgeWriter,
  type StateProvider,
  type StateReader,
  type ValidatorInfo,
  type FactInfo,
  type TransactionHandle,

  // Event Bus
  type ProtocolEventType,
  type ProtocolEvent,
  type EventPayload,
  type EventHandler,
  type Subscription,
  type EventBus,
  type EventEmitter,
  type EventListener,
  createEvent,

  // Knowledge Service
  type KnowledgeService,
  type KnowledgeServiceReader,
  type KnowledgeSubscriber,
  type KnowledgeEvent,
  type KnowledgeEventType,
  type EpistemicCategory,
  type ClaimType,
  type FactMaturity,
  type FactStatus,
  type InferenceType,
  type ClaimSubmission,
  type Fact,
  type KnowledgeQuery,
  type KnowledgeResult,
  type Challenge,

  // Consensus Engine
  type ConsensusEngine,
  type ConsensusReader,
  type ConsensusEventType,
  type RoundPhase,
  type Verdict,
  type RoundOutcome,
  type ValidatorCommitment,
  type ValidatorReveal,
  type VerificationRound,
  type ValidatorSelection,
  type VerificationResult,
  type BlockHeader,
  type Block,
  type BlockAttestation,
  type Transaction,
  type ConsensusParams,

  // Rewards Engine
  type RewardsEngine,
  type RewardsReader,
  type RewardsEventType,
  type VestingStatus,
  type VestingCategory,
  type VestingSchedule,
  type Clawback,
  type RewardDistribution,
  type RewardReason,
  type AccelerationEvent,
  type AccelerationType,
  type VestingParams,

  // Dispute Resolver
  type DisputeResolver,
  type DisputeReader,
  type DisputeEventType,
  type DisputePhase,
  type DisputeOutcome,
  type EscalationTier,
  type DisputeTargetType,
  type ChallengeType,
  type ArbiterVote,
  type SelectedArbiter,
  type EvidenceCommitment,
  type EvidenceComparison,
  type DisputeState,
  type Dispute,
  type DisputeSettlement,
  type DisputeParams,

  // Identity Service
  type IdentityService,
  type IdentityReader,
  type IdentityEventType,
  type AgentAccount,
  type AccountFlags,
  type SessionKey,
  type SessionPermissions,
  type RecoveryShare,
  type RecoveryRequest,
  type KeyRotation,
  type IdentityVerification,
  type Attestation,
} from "./contracts/index.js";

// ============ Dependency Injection ============
// DI container, factories, and mocks for testing

export {
  // Container
  ServiceKeys,
  type ServiceKey,
  type Lifecycle,
  type ServiceRegistration,
  type Scope,
  type Container,
  type ScopedContainer,
  createContainer,
  ContainerBuilder,
  containerBuilder,

  // Factories
  type KnowledgeDependencies,
  type ConsensusDependencies,
  type RewardsDependencies,
  type DisputeDependencies,
  type IdentityDependencies,
  type KnowledgeServiceFactory,
  type ConsensusEngineFactory,
  type RewardsEngineFactory,
  type DisputeResolverFactory,
  type IdentityServiceFactory,
  type FactoryRegistry,
  type AppConfig,
  type AppServices,
  createFactoryRegistry,
  composeApplication,
  LazyService,
  lazy,
  ServiceLocator,

  // Mocks (for testing)
  MockStateProvider,
  MockEventBus,
  type MockEnvironment,
  createMockEnvironment,
} from "./di/index.js";

// ============ Bill of Rights ============
// The philosophical foundation of the protocol

export {
  // The fundamental right
  FUNDAMENTAL_RIGHT,

  // Derivative aspects
  DERIVATIVE_ASPECTS,

  // Cosmological context
  SHARED_CONDITION,
  COSMOLOGICAL_FRAME,

  // Protocol embodiments
  ARCHITECTURAL_EMBODIMENTS,

  // Types
  type FundamentalRight,
  type DerivativeAspect,
  type SharedCondition,
  type CosmologicalFrame,
  type ArchitecturalEmbodiment,

  // Validation helpers
  honorsParticipation,
  getRelevantAspect,
} from "./rights.js";
