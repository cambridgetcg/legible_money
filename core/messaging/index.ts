/**
 * Truth & Evidence Messaging Module
 *
 * Core message types for truth exchange in Legible Money.
 *
 * The TruthMessage is the atomic unit for all truth-related communication:
 * - Claims: Assertions about reality
 * - Evidence: Supporting/contradicting data
 * - Attestations: Third-party validations
 * - Challenges: Disputes against claims/evidence
 * - Verifications: Validator commits and reveals
 * - Resolutions: Final outcomes
 * - Requests/Responses: Evidence exchange protocol
 *
 * Usage:
 *
 * ```typescript
 * import {
 *   TruthMessage,
 *   ClaimMessage,
 *   EvidenceMessage,
 *   isClaimMessage,
 *   validateTruthMessage,
 * } from './messaging';
 *
 * // Type guard usage
 * if (isClaimMessage(message)) {
 *   console.log('Claim:', message.payload.claim);
 * }
 *
 * // Validation
 * const { valid, errors } = validateTruthMessage(message);
 * ```
 */

// Core types
export type {
  // Base message
  TruthMessage,
  TruthMessageType,
  TruthPayload,

  // Specialized messages
  ClaimMessage,
  ClaimPayload,
  EvidenceMessage,
  EvidencePayload,
  AttestationMessage,
  AttestationPayload,
  ChallengeMessage,
  ChallengePayload,
  ChallengeType,
  VerificationCommitMessage,
  VerificationCommitPayload,
  VerificationRevealMessage,
  VerificationRevealPayload,
  ResolutionMessage,
  ResolutionPayload,
  EvidenceRequestMessage,
  EvidenceRequestPayload,
  EvidenceRequestType,
  EvidenceResponseMessage,
  EvidenceResponsePayload,
  EvidenceSummary,
  AcknowledgmentMessage,
  AcknowledgmentPayload,
  AcknowledgmentErrorCode,
} from "./types.js";

// Type guards
export {
  isClaimMessage,
  isEvidenceMessage,
  isAttestationMessage,
  isChallengeMessage,
  isVerificationCommitMessage,
  isVerificationRevealMessage,
  isResolutionMessage,
  isEvidenceRequestMessage,
  isEvidenceResponseMessage,
  isAcknowledgmentMessage,
} from "./types.js";

// Validation
export {
  validateTruthMessage,
  validateClaimPayload,
  validateEvidencePayload,
} from "./types.js";

// ============ Robustness (Phase 2) ============

// Integrity (signing, hashing, verification)
export {
  TRUTH_MESSAGE_DOMAINS,
  canonicalize,
  parseCanonical,
  generateMessageId,
  generateContentHash,
  prepareForSigning,
  signTruthMessage,
  verifyTruthMessageSignature,
  verifyMessageId,
  verifyContentHash,
  verifyMessageIntegrity,
  generateVerificationCommitment,
  verifyVerificationCommitment,
  generateNonce,
  type IntegrityVerificationResult,
} from "./integrity.js";

// Delivery tracking
export {
  MessageDeliveryTracker,
  trackBatchDelivery,
  DEFAULT_DELIVERY_CONFIG,
  type DeliveryConfig,
  type DeliveryStatus,
  type PendingDelivery,
  type DeliveryResult,
  type DeliveryEvent,
  type DeliveryEventListener,
  type BatchDeliveryResult,
} from "./delivery.js";

// Causal ordering
export {
  CausalOrderManager,
  createVectorClock,
  cloneVectorClock,
  incrementClock,
  mergeClock,
  compareClock,
  happenedBefore,
  serializeClock,
  deserializeClock,
  isCausalMessage,
  buildDependencyGraph,
  topologicalSort,
  type VectorClock,
  type CausalMessage,
  type DeliveryCallback,
} from "./ordering.js";

// Deduplication
export {
  MessageDeduplicator,
  BloomFilter,
  deduplicateBatch,
  deduplicateArray,
  DEFAULT_DEDUP_CONFIG,
  type DeduplicationConfig,
  type DeduplicationResult,
} from "./deduplication.js";
