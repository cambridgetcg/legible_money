/**
 * Truth & Evidence Message System - Core Types
 *
 * The TruthMessage is the atomic unit for all truth-related communication
 * in Legible Money. This module defines the message hierarchy:
 *
 *   TruthMessage (base)
 *   ├── ClaimMessage        - Assertions about reality
 *   ├── EvidenceMessage     - Supporting/contradicting data
 *   ├── AttestationMessage  - Third-party validation
 *   ├── ChallengeMessage    - Disputes against claims/evidence
 *   ├── VerificationCommitMessage  - Validator commits
 *   ├── VerificationRevealMessage  - Validator reveals
 *   ├── ResolutionMessage   - Final outcomes
 *   ├── EvidenceRequestMessage  - Ask for evidence
 *   ├── EvidenceResponseMessage - Respond with evidence
 *   └── AcknowledgmentMessage   - Delivery confirmation
 *
 * Design follows the ScheduleMessage pattern (Patent FIG.2 fields 201-214)
 * extended with truth-specific fields for provenance and verification.
 */

import type { DID } from "../agents/identity/did.js";
import type { Signature } from "../crypto/keys.js";
import type {
  Confidence,
  EpistemicCategory,
  ClaimType,
  InferenceType,
  EvidenceIndependence,
} from "../knowledge/types.js";
import type { EvidenceType } from "../evidence/types.js";

// ============ Message Type Discriminator ============

/**
 * All truth message types
 */
export type TruthMessageType =
  | "claim"
  | "evidence"
  | "attestation"
  | "challenge"
  | "verification_commit"
  | "verification_reveal"
  | "resolution"
  | "evidence_request"
  | "evidence_response"
  | "acknowledgment";

// ============ Base TruthMessage ============

/**
 * TruthMessage - Atomic unit for truth exchange
 *
 * All messages in the truth system inherit from this base.
 * Follows ScheduleMessage pattern with truth-specific extensions.
 */
export interface TruthMessage {
  // === Identity & Routing ===
  /** Message type discriminator */
  messageType: TruthMessageType;

  /** Unique identifier (content-addressed hash) */
  messageId: string;

  /** Protocol version */
  version: "1.0.0";

  // === Participants ===
  /** Who sent this message (DID) */
  sender: DID;

  /** Target recipient (null = broadcast to topic) */
  recipient?: DID | "broadcast";

  // === Content ===
  /** Type-specific payload */
  payload: TruthPayload;

  /** SHA-512/256 hash of payload */
  contentHash: string;

  /** Ed25519 signature from sender */
  signature: Signature;

  // === Provenance ===
  /** Previous message IDs this builds on (causal dependencies) */
  referencedMessages: string[];

  /** Message ID that triggered this (if reactive) */
  causedBy?: string;

  // === Network ===
  /** Gas price for processing (from ScheduleMessage field 207) */
  networkFeeUnit: bigint;

  /** Max gas for this message (from ScheduleMessage field 208) */
  networkFeeLimit: bigint;

  /** Unix timestamp in milliseconds */
  timestamp: number;

  /** Block height when created (if on-chain) */
  blockHeight?: bigint;

  // === Metadata ===
  /** Knowledge domain (e.g., 'mathematics', 'history') */
  domain?: string;

  /** Searchable tags */
  tags?: string[];

  /** Time-to-live in blocks (after which message expires) */
  ttlBlocks?: number;
}

/**
 * Union of all payload types
 */
export type TruthPayload =
  | ClaimPayload
  | EvidencePayload
  | AttestationPayload
  | ChallengePayload
  | VerificationCommitPayload
  | VerificationRevealPayload
  | ResolutionPayload
  | EvidenceRequestPayload
  | EvidenceResponsePayload
  | AcknowledgmentPayload;

// ============ ClaimMessage ============

/**
 * ClaimMessage - Submit a truth assertion
 *
 * Claims are statements about reality that can be verified.
 * They require stake and may include pre-attached evidence.
 */
export interface ClaimMessage extends TruthMessage {
  messageType: "claim";
  payload: ClaimPayload;
}

export interface ClaimPayload {
  /** The assertion text */
  claim: string;

  /** Type of claim (mathematical, empirical, historical, etc.) */
  claimType: ClaimType;

  /** Epistemic category determines max confidence ceiling */
  epistemicCategory: EpistemicCategory;

  /** Claimant's proposed confidence level */
  proposedConfidence: Confidence;

  /** LGM staked on this claim (slashed if proven false) */
  stake: bigint;

  /** If derived from other facts */
  derivation?: {
    /** Source fact IDs */
    factIds: string[];
    /** Type of inference used */
    inferenceType: InferenceType;
    /** Logic chain description */
    logicChain: string;
  };

  /** Pre-attached evidence IDs (evidence submitted separately) */
  evidenceIds?: string[];

  /** Optional external references (URLs, DOIs, etc.) */
  externalReferences?: string[];
}

// ============ EvidenceMessage ============

/**
 * EvidenceMessage - Provide supporting or contradicting evidence
 *
 * Evidence can target a specific claim, fact, or be submitted independently.
 */
export interface EvidenceMessage extends TruthMessage {
  messageType: "evidence";
  payload: EvidencePayload;
}

export interface EvidencePayload {
  /** Evidence type (on_chain, peer_reviewed, oracle, etc.) */
  evidenceType: EvidenceType;

  /** Content hash of the evidence */
  contentHash: string;

  /** Original source URL or identifier */
  originalSource: string;

  /** Target claim ID (if supporting/contradicting a claim) */
  targetClaimId?: string;

  /** Target fact ID (if challenging existing fact) */
  targetFactId?: string;

  /** Direction relative to target */
  direction: "supporting" | "contradicting" | "neutral";

  /** Intrinsic strength of this evidence (0-1) */
  intrinsicStrength: number;

  /** Independence metrics */
  independence: EvidenceIndependence;

  /** Request automatic archival (e.g., via Wayback Machine) */
  archiveRequested: boolean;

  /** Archive locations if already archived */
  archiveLocations?: string[];

  /** For oracle evidence: oracle metadata */
  oracleMetadata?: {
    oracleId: string;
    dataFeedId?: string;
    timestamp: number;
    signature?: string;
  };

  /** Additional metadata */
  metadata?: Record<string, unknown>;
}

// ============ AttestationMessage ============

/**
 * AttestationMessage - Third-party validation of claims/evidence/facts
 *
 * Attestations allow qualified parties to vouch for or qualify
 * existing claims, evidence, or facts.
 */
export interface AttestationMessage extends TruthMessage {
  messageType: "attestation";
  payload: AttestationPayload;
}

export interface AttestationPayload {
  /** What is being attested */
  targetType: "claim" | "evidence" | "fact";

  /** ID of the target */
  targetId: string;

  /** Type of attestation */
  attestationType: "confirm" | "deny" | "qualify";

  /** If qualified, what's the qualification */
  qualification?: string;

  /** Attester's confidence in their attestation */
  confidence: Confidence;

  /** Attester's relevant expertise areas */
  expertise?: string[];

  /** Stake backing this attestation */
  stake: bigint;

  /** Optional reasoning */
  reasoning?: string;
}

// ============ ChallengeMessage ============

/**
 * ChallengeMessage - Dispute a claim, evidence, or fact
 *
 * Challenges initiate the dispute resolution process.
 * "Intention does not matter, result does" - if wrong, challenger pays.
 */
export interface ChallengeMessage extends TruthMessage {
  messageType: "challenge";
  payload: ChallengePayload;
}

export interface ChallengePayload {
  /** What is being challenged */
  targetType: "claim" | "evidence" | "fact" | "verification";

  /** ID of the target */
  targetId: string;

  /** Reason for challenge */
  reason: string;

  /** Type of challenge */
  challengeType: ChallengeType;

  /** Counter-evidence IDs supporting the challenge */
  counterEvidenceIds: string[];

  /** Challenge stake (must meet maturity requirements) */
  stake: bigint;

  /** Requested outcome if challenge succeeds */
  requestedOutcome: "retraction" | "correction" | "confidence_reduction";

  /** Optional specific correction if requesting correction */
  proposedCorrection?: string;
}

export type ChallengeType =
  | "factual_error" // The claim is factually incorrect
  | "methodology_flaw" // Evidence gathering was flawed
  | "source_unreliable" // Source is not trustworthy
  | "outdated" // Information is no longer current
  | "misrepresentation" // Claim misrepresents the evidence
  | "logical_error" // Derivation has logical flaws
  | "incomplete"; // Missing crucial context

// ============ VerificationCommitMessage ============

/**
 * VerificationCommitMessage - Validator commits to a verdict
 *
 * Part of the commit-reveal mechanism for verification.
 * Commitment hides the actual verdict until reveal phase.
 */
export interface VerificationCommitMessage extends TruthMessage {
  messageType: "verification_commit";
  payload: VerificationCommitPayload;
}

export interface VerificationCommitPayload {
  /** Verification round ID */
  roundId: string;

  /** Claim ID being verified */
  claimId: string;

  /** Hash of (verdict, confidence, nonce, referencesChecked) */
  commitment: string;

  /** VRF output proving selection eligibility */
  vrfOutput: string;

  /** VRF proof */
  vrfProof: string;
}

// ============ VerificationRevealMessage ============

/**
 * VerificationRevealMessage - Validator reveals their verdict
 *
 * Must match the previous commitment. Reveals the actual
 * verdict, confidence, and reasoning.
 */
export interface VerificationRevealMessage extends TruthMessage {
  messageType: "verification_reveal";
  payload: VerificationRevealPayload;
}

export interface VerificationRevealPayload {
  /** Verification round ID */
  roundId: string;

  /** Claim ID being verified */
  claimId: string;

  /** The verdict */
  verdict: "confirmed" | "rejected" | "uncertain";

  /** Confidence in the verdict (0-1) */
  confidence: Confidence;

  /** Nonce used in commitment */
  nonce: string;

  /** Evidence IDs that were examined */
  referencesChecked: string[];

  /** Optional reasoning for the verdict */
  reasoning?: string;

  /** Methodology used for verification */
  methodologyUsed?: string;
}

// ============ ResolutionMessage ============

/**
 * ResolutionMessage - Final outcome of verification or dispute
 *
 * Announces the final result including reward/slash distribution.
 */
export interface ResolutionMessage extends TruthMessage {
  messageType: "resolution";
  payload: ResolutionPayload;
}

export interface ResolutionPayload {
  /** What was resolved */
  targetType: "claim" | "challenge" | "dispute";

  /** ID of the resolved item */
  targetId: string;

  /** Final outcome */
  outcome: "accepted" | "rejected" | "contested" | "withdrawn";

  /** Final confidence level (for accepted claims) */
  finalConfidence: Confidence;

  /** Resulting fact ID (if claim accepted) */
  resultingFactId?: string;

  /** Block number when resolved */
  resolvedAtBlock: bigint;

  /** Reward distribution */
  rewards: Array<{
    recipient: DID;
    amount: bigint;
    reason: string;
  }>;

  /** Slash distribution */
  slashes: Array<{
    target: DID;
    amount: bigint;
    reason: string;
  }>;

  /** Summary of participation */
  participationSummary?: {
    totalValidators: number;
    confirmVotes: number;
    rejectVotes: number;
    uncertainVotes: number;
    abstentions: number;
  };
}

// ============ EvidenceRequestMessage ============

/**
 * EvidenceRequestMessage - Request evidence on a topic or claim
 *
 * Allows participants to solicit evidence from the network.
 * Can include a bounty for quality responses.
 */
export interface EvidenceRequestMessage extends TruthMessage {
  messageType: "evidence_request";
  payload: EvidenceRequestPayload;
}

export interface EvidenceRequestPayload {
  /** Type of request */
  requestType: EvidenceRequestType;

  /** For topic-based requests */
  topic?: string;

  /** Knowledge domain filter */
  domain?: string;

  /** Minimum evidence strength required */
  minStrength?: number;

  /** Maximum age in blocks */
  maxAgeBlocks?: number;

  /** Evidence types to include */
  evidenceTypes?: EvidenceType[];

  /** For specific evidence requests */
  evidenceId?: string;

  /** For claim-related requests */
  claimId?: string;

  /** For fact-related requests */
  factId?: string;

  /** Pagination limit */
  limit?: number;

  /** Pagination cursor */
  cursor?: string;

  /** Optional bounty for quality responses (in LGM) */
  bounty?: bigint;

  /** Block height deadline for responses */
  deadline?: bigint;
}

export type EvidenceRequestType =
  | "by_topic" // Evidence related to a topic
  | "by_claim" // Evidence for/against a specific claim
  | "by_fact" // Evidence supporting a fact
  | "by_id" // Specific evidence by ID
  | "contradiction" // Evidence that contradicts something
  | "corroboration"; // Evidence that corroborates something

// ============ EvidenceResponseMessage ============

/**
 * EvidenceResponseMessage - Response to an evidence request
 *
 * Contains matching evidence and quality metrics.
 */
export interface EvidenceResponseMessage extends TruthMessage {
  messageType: "evidence_response";
  payload: EvidenceResponsePayload;
}

export interface EvidenceResponsePayload {
  /** Original request message ID */
  requestId: string;

  /** Matching evidence */
  evidence: EvidenceSummary[];

  /** Total matching (for pagination) */
  total: number;

  /** Next cursor for pagination */
  nextCursor?: string;

  /** Per-evidence relevance scores (0-1) */
  relevanceScores: number[];

  /** Estimated coverage (what % of available evidence is included) */
  coverageEstimate: number;
}

/**
 * Summary of evidence for responses (not full Evidence object)
 */
export interface EvidenceSummary {
  id: string;
  type: EvidenceType;
  contentHash: string;
  direction: "supporting" | "contradicting" | "neutral";
  strength: number;
  submittedAt: number;
  source: string;
}

// ============ AcknowledgmentMessage ============

/**
 * AcknowledgmentMessage - Delivery confirmation
 *
 * Used to confirm receipt and processing status of messages.
 */
export interface AcknowledgmentMessage extends TruthMessage {
  messageType: "acknowledgment";
  payload: AcknowledgmentPayload;
}

export interface AcknowledgmentPayload {
  /** ID of the acknowledged message */
  acknowledgedMessageId: string;

  /** Processing status */
  status: "received" | "processing" | "accepted" | "rejected";

  /** Error details if rejected */
  error?: {
    code: AcknowledgmentErrorCode;
    message: string;
    details?: Record<string, unknown>;
  };

  /** Processing ID for tracking async operations */
  processingId?: string;

  /** Estimated completion time (for 'processing' status) */
  estimatedCompletionMs?: number;
}

export type AcknowledgmentErrorCode =
  | "INVALID_SIGNATURE"
  | "INVALID_FORMAT"
  | "DUPLICATE_MESSAGE"
  | "INSUFFICIENT_FEE"
  | "INSUFFICIENT_STAKE"
  | "TARGET_NOT_FOUND"
  | "UNAUTHORIZED"
  | "EXPIRED"
  | "RATE_LIMITED"
  | "INTERNAL_ERROR";

// ============ Type Guards ============

/**
 * Check if a message is a ClaimMessage
 */
export function isClaimMessage(msg: TruthMessage): msg is ClaimMessage {
  return msg.messageType === "claim";
}

/**
 * Check if a message is an EvidenceMessage
 */
export function isEvidenceMessage(msg: TruthMessage): msg is EvidenceMessage {
  return msg.messageType === "evidence";
}

/**
 * Check if a message is an AttestationMessage
 */
export function isAttestationMessage(
  msg: TruthMessage,
): msg is AttestationMessage {
  return msg.messageType === "attestation";
}

/**
 * Check if a message is a ChallengeMessage
 */
export function isChallengeMessage(msg: TruthMessage): msg is ChallengeMessage {
  return msg.messageType === "challenge";
}

/**
 * Check if a message is a VerificationCommitMessage
 */
export function isVerificationCommitMessage(
  msg: TruthMessage,
): msg is VerificationCommitMessage {
  return msg.messageType === "verification_commit";
}

/**
 * Check if a message is a VerificationRevealMessage
 */
export function isVerificationRevealMessage(
  msg: TruthMessage,
): msg is VerificationRevealMessage {
  return msg.messageType === "verification_reveal";
}

/**
 * Check if a message is a ResolutionMessage
 */
export function isResolutionMessage(
  msg: TruthMessage,
): msg is ResolutionMessage {
  return msg.messageType === "resolution";
}

/**
 * Check if a message is an EvidenceRequestMessage
 */
export function isEvidenceRequestMessage(
  msg: TruthMessage,
): msg is EvidenceRequestMessage {
  return msg.messageType === "evidence_request";
}

/**
 * Check if a message is an EvidenceResponseMessage
 */
export function isEvidenceResponseMessage(
  msg: TruthMessage,
): msg is EvidenceResponseMessage {
  return msg.messageType === "evidence_response";
}

/**
 * Check if a message is an AcknowledgmentMessage
 */
export function isAcknowledgmentMessage(
  msg: TruthMessage,
): msg is AcknowledgmentMessage {
  return msg.messageType === "acknowledgment";
}

// ============ Validation ============

/**
 * Validate TruthMessage base structure
 */
export function validateTruthMessage(msg: unknown): {
  valid: boolean;
  errors: string[];
} {
  const errors: string[] = [];

  if (!msg || typeof msg !== "object") {
    return { valid: false, errors: ["Message must be an object"] };
  }

  const m = msg as Record<string, unknown>;

  // Valid message types
  const validMessageTypes = [
    "claim",
    "evidence",
    "attestation",
    "challenge",
    "verification_commit",
    "verification_reveal",
    "resolution",
    "evidence_request",
    "evidence_response",
    "acknowledgment",
  ];

  // Required fields
  if (!m.messageType || typeof m.messageType !== "string") {
    errors.push("messageType is required and must be a string");
  } else if (!validMessageTypes.includes(m.messageType)) {
    errors.push(`messageType must be one of: ${validMessageTypes.join(", ")}`);
  }
  if (!m.messageId || typeof m.messageId !== "string") {
    errors.push("messageId is required and must be a string");
  }
  if (!m.sender || typeof m.sender !== "string") {
    errors.push("sender is required and must be a string (DID)");
  }
  if (!m.payload || typeof m.payload !== "object") {
    errors.push("payload is required and must be an object");
  }
  if (!m.contentHash || typeof m.contentHash !== "string") {
    errors.push("contentHash is required and must be a string");
  }
  if (!m.signature) {
    errors.push("signature is required");
  }
  if (!Array.isArray(m.referencedMessages)) {
    errors.push("referencedMessages must be an array");
  }
  if (typeof m.networkFeeUnit !== "bigint") {
    errors.push("networkFeeUnit must be a bigint");
  }
  if (typeof m.networkFeeLimit !== "bigint") {
    errors.push("networkFeeLimit must be a bigint");
  }
  if (typeof m.timestamp !== "number") {
    errors.push("timestamp must be a number");
  }

  // Version check
  if (m.version !== "1.0.0") {
    errors.push('version must be "1.0.0"');
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Validate ClaimPayload
 */
export function validateClaimPayload(payload: unknown): {
  valid: boolean;
  errors: string[];
} {
  const errors: string[] = [];

  if (!payload || typeof payload !== "object") {
    return { valid: false, errors: ["Payload must be an object"] };
  }

  const p = payload as Record<string, unknown>;

  if (!p.claim || typeof p.claim !== "string" || p.claim.length === 0) {
    errors.push("claim is required and must be a non-empty string");
  }
  if (!p.claimType || typeof p.claimType !== "string") {
    errors.push("claimType is required");
  }
  if (!p.epistemicCategory || typeof p.epistemicCategory !== "string") {
    errors.push("epistemicCategory is required");
  }
  if (
    typeof p.proposedConfidence !== "number" ||
    p.proposedConfidence < 0 ||
    p.proposedConfidence > 1
  ) {
    errors.push("proposedConfidence must be a number between 0 and 1");
  }
  if (typeof p.stake !== "bigint" || p.stake <= 0n) {
    errors.push("stake must be a positive bigint");
  }

  return { valid: errors.length === 0, errors };
}

/**
 * Validate EvidencePayload
 */
export function validateEvidencePayload(payload: unknown): {
  valid: boolean;
  errors: string[];
} {
  const errors: string[] = [];

  if (!payload || typeof payload !== "object") {
    return { valid: false, errors: ["Payload must be an object"] };
  }

  const p = payload as Record<string, unknown>;

  if (!p.evidenceType || typeof p.evidenceType !== "string") {
    errors.push("evidenceType is required");
  }
  if (
    !p.contentHash ||
    typeof p.contentHash !== "string" ||
    p.contentHash.length < 16
  ) {
    errors.push("contentHash is required and must be at least 16 characters");
  }
  if (!p.originalSource || typeof p.originalSource !== "string") {
    errors.push("originalSource is required");
  }
  if (
    !["supporting", "contradicting", "neutral"].includes(p.direction as string)
  ) {
    errors.push("direction must be supporting, contradicting, or neutral");
  }
  if (
    typeof p.intrinsicStrength !== "number" ||
    p.intrinsicStrength < 0 ||
    p.intrinsicStrength > 1
  ) {
    errors.push("intrinsicStrength must be a number between 0 and 1");
  }

  return { valid: errors.length === 0, errors };
}
