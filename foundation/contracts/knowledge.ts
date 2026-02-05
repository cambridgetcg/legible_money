/**
 * Knowledge Service Contract
 *
 * Domain interface for the knowledge layer.
 * Handles facts, claims, confidence, and epistemic operations.
 *
 * MOD-1: Single Responsibility
 * Knowledge module changes when fact/claim semantics change.
 */

import type { DID, Confidence, BlockHeight, Amount } from "../types/index.js";

// ============ Knowledge Types ============

/**
 * Epistemic categories (how knowledge is justified)
 */
export type EpistemicCategory =
  | "analytic" // True by definition
  | "formal" // Mathematical/logical proofs
  | "protocol" // On-chain constructed truths
  | "empirical" // Observable facts
  | "historical" // Past events
  | "predictive" // Future predictions
  | "social"; // Consensus-based

/**
 * Claim types
 */
export type ClaimType =
  | "mathematical"
  | "empirical"
  | "historical"
  | "definitional"
  | "computational"
  | "protocol";

/**
 * Fact maturity levels
 */
export type FactMaturity = "emerging" | "established" | "canonical";

/**
 * Fact status — re-exported from core for convenience.
 * See core/knowledge/types.ts for the canonical definition.
 */
import type { FactStatus } from "../../core/knowledge/types.js";
export type { FactStatus };

/**
 * Inference types for derived knowledge
 */
export type InferenceType = "deductive" | "inductive" | "abductive";

/**
 * Claim submission request
 */
export interface ClaimSubmission {
  claim: string;
  type: ClaimType;
  domain: string;
  epistemicCategory: EpistemicCategory;
  proposedConfidence: Confidence;
  stake: Amount;
  evidenceIds?: string[];
  derivedFrom?: {
    factIds: string[];
    inferenceType: InferenceType;
    logicChain: string;
  };
}

/**
 * Fact record (verified claim)
 */
export interface Fact {
  id: string;
  claim: string;
  claimHash: string;
  type: ClaimType;
  domain: string;
  epistemicCategory: EpistemicCategory;
  maturity: FactMaturity;
  confidence: Confidence;
  status: FactStatus;
  submittedBy: DID;
  submittedAt: BlockHeight;
  verifiedAt: BlockHeight;
  lastVerified: BlockHeight;
  verifierCount: number;
  challengeCount: number;
  derivedFrom?: string[]; // Parent fact IDs
  derivedFacts?: string[]; // Child fact IDs
}

/**
 * Knowledge query parameters
 */
export interface KnowledgeQuery {
  claim?: string;
  factId?: string;
  domain?: string;
  minConfidence?: Confidence;
  maxAge?: number; // Blocks since verification
  types?: ClaimType[];
  epistemicCategories?: EpistemicCategory[];
  status?: FactStatus[];
  limit?: number;
  offset?: number;
}

/**
 * Knowledge query result
 */
export interface KnowledgeResult {
  facts: Fact[];
  totalCount: number;
  queryConfidence: Confidence;
}

/**
 * Challenge to an existing fact
 */
export interface Challenge {
  id: string;
  factId: string;
  challenger: DID;
  reason: string;
  counterEvidenceIds: string[];
  stake: Amount;
  status: "pending" | "upheld" | "rejected";
  submittedAt: BlockHeight;
  resolvedAt?: BlockHeight;
}

// ============ Knowledge Service Interface ============

/**
 * Knowledge service - main domain interface
 */
export interface KnowledgeService {
  // === Queries ===

  /**
   * Get a fact by ID
   */
  getFact(factId: string): Promise<Fact | null>;

  /**
   * Query facts with filters
   */
  queryFacts(query: KnowledgeQuery): Promise<KnowledgeResult>;

  /**
   * Get confidence for a fact
   */
  getConfidence(factId: string): Promise<Confidence | null>;

  /**
   * Calculate maturity for a fact
   */
  getMaturity(factId: string): Promise<FactMaturity | null>;

  /**
   * Get facts that cite/derive from a given fact
   */
  getCitingFacts(factId: string): Promise<Fact[]>;

  /**
   * Get facts that a given fact derives from
   */
  getSourceFacts(factId: string): Promise<Fact[]>;

  /**
   * Check if a claim already exists (deduplication)
   */
  claimExists(claim: string): Promise<boolean>;

  // === Commands ===

  /**
   * Submit a claim for verification
   * Returns claim ID for tracking
   */
  submitClaim(
    submission: ClaimSubmission,
    submitter: DID,
  ): Promise<{ claimId: string; roundId: string }>;

  /**
   * Add evidence to a pending claim
   */
  addEvidence(claimId: string, evidenceId: string): Promise<void>;

  /**
   * Challenge an existing fact
   */
  challengeFact(
    factId: string,
    challenger: DID,
    reason: string,
    counterEvidenceIds: string[],
    stake: Amount,
  ): Promise<string>; // Returns challenge ID

  /**
   * Resolve a challenge (called by dispute system)
   */
  resolveChallenge(
    challengeId: string,
    upheld: boolean,
    newConfidence?: Confidence,
  ): Promise<void>;

  // === Confidence Operations ===

  /**
   * Update confidence based on new verification
   */
  updateConfidence(
    factId: string,
    newConfidence: Confidence,
    reason: string,
  ): Promise<void>;

  /**
   * Apply confidence decay (called periodically)
   */
  applyDecay(factId: string): Promise<Confidence>;

  /**
   * Recalculate confidence from all sources
   */
  recalculateConfidence(factId: string): Promise<Confidence>;

  // === Derivation ===

  /**
   * Create a derived fact from existing facts
   */
  createDerivedFact(
    sourceFacts: string[],
    derivedClaim: string,
    inferenceType: InferenceType,
    logicChain: string,
    submitter: DID,
  ): Promise<string>; // Returns new fact ID

  /**
   * Propagate confidence change to derived facts
   */
  propagateConfidenceChange(factId: string): Promise<void>;
}

// ============ Knowledge Reader (Read-Only) ============

/**
 * Read-only knowledge interface for queries
 */
export interface KnowledgeReader {
  getFact(factId: string): Promise<Fact | null>;
  queryFacts(query: KnowledgeQuery): Promise<KnowledgeResult>;
  getConfidence(factId: string): Promise<Confidence | null>;
  getMaturity(factId: string): Promise<FactMaturity | null>;
  claimExists(claim: string): Promise<boolean>;
}

// ============ Knowledge Events ============

/**
 * Events emitted by knowledge service
 */
export type KnowledgeEventType =
  | "claim_submitted"
  | "fact_verified"
  | "fact_challenged"
  | "fact_revoked"
  | "confidence_updated"
  | "derivation_created";

export interface KnowledgeEvent {
  type: KnowledgeEventType;
  factId?: string;
  claimId?: string;
  timestamp: number;
  data: Record<string, unknown>;
}

/**
 * Knowledge event subscriber
 */
export interface KnowledgeSubscriber {
  on(event: KnowledgeEventType, handler: (event: KnowledgeEvent) => void): void;
}
