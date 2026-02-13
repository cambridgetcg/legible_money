/**
 * Knowledge Layer Types
 *
 * Core types for the Proof of Truth system.
 * Agents submit claims, validators verify, knowledge is recorded on-chain.
 */

import type { DID } from "../../core/agents/identity/did.js";

/**
 * Confidence level (0.0 to 1.0)
 *
 * ZD-K02: TS uses float 0.0–1.0, Go/Cosmos uses uint64 basis points (0–1,000,000).
 * Use the conversion helpers below when crossing the boundary.
 */
export type Confidence = number;

/** BPS scale used by Go/Cosmos SDK confidence values (1,000,000 = 100%). */
export const CONFIDENCE_BPS_SCALE = 1_000_000;

/**
 * Convert TS float confidence (0.0–1.0) to Go uint64 BPS (0–1,000,000).
 * Clamps to valid range.
 */
export function confidenceToBps(confidence: Confidence): number {
  const clamped = Math.max(0, Math.min(1, confidence));
  return Math.round(clamped * CONFIDENCE_BPS_SCALE);
}

/**
 * Convert Go uint64 BPS (0–1,000,000) to TS float confidence (0.0–1.0).
 * Clamps to valid range.
 */
export function bpsToConfidence(bps: number): Confidence {
  const clamped = Math.max(0, Math.min(CONFIDENCE_BPS_SCALE, bps));
  return clamped / CONFIDENCE_BPS_SCALE;
}

// ============ Epistemic Categories ============

/**
 * Epistemic Categories
 *
 * Different types of claims have fundamentally different confidence semantics.
 * A mathematical proof and a weather observation cannot be evaluated the same way.
 */
export type EpistemicCategory =
  | "analytic" // True by definition (e.g., "bachelors are unmarried")
  | "formal" // Mathematical/logical proofs (e.g., "2+2=4")
  | "protocol" // On-chain constructed truths (e.g., "Block N follows N-1")
  | "empirical" // Observable facts (e.g., "Temperature is 20°C")
  | "historical" // Past events (e.g., "Event X occurred on date Y")
  | "predictive" // Future predictions (e.g., "Price will rise")
  | "social" // Consensus-based (e.g., "X is considered beautiful")
  | "replicated"; // IRL-verified empirical facts (upgraded from empirical after 2+ replications)

/**
 * Profile for an epistemic category
 * Defines the confidence semantics for that type of claim
 */
export interface EpistemicProfile {
  category: EpistemicCategory;
  maxConfidence: Confidence; // Ceiling - cannot exceed this
  decayHalfLife: number | null; // Blocks until confidence halves (null = no decay)
  minConfidence: number; // Floor - fact expires below this (1.0 for eternal)
  reverificationBonus: number; // Confidence boost from successful re-verification
  verificationMethod:
    | "definition"
    | "proof"
    | "construction"
    | "observation"
    | "records"
    | "models"
    | "consensus"
    | "replication";
  requiresReverification: boolean; // Whether ongoing verification needed
  minVerifiers: number; // Minimum validators for this category
  description: string;
}

/**
 * Epistemic profiles — SINGLE SOURCE OF TRUTH for all category parameters.
 *
 * CATEGORY_DECAY and EPISTEMIC_CEILINGS in consensus/pot/confidence.ts
 * are derived from these profiles. Do NOT define parallel constants.
 *
 * Key insight: Eternal truths don't decay. Empirical facts do.
 */
export const EPISTEMIC_PROFILES: Record<EpistemicCategory, EpistemicProfile> = {
  analytic: {
    category: "analytic",
    maxConfidence: 1.0,
    decayHalfLife: null, // Eternal - true by definition
    minConfidence: 1.0,
    reverificationBonus: 0,
    verificationMethod: "definition",
    requiresReverification: false,
    minVerifiers: 3,
    description: "Truths that follow from definitions alone",
  },
  formal: {
    category: "formal",
    maxConfidence: 1.0,
    decayHalfLife: null, // Eternal - mathematically proven
    minConfidence: 1.0,
    reverificationBonus: 0,
    verificationMethod: "proof",
    requiresReverification: false,
    minVerifiers: 5,
    description: "Mathematical and logical proofs",
  },
  protocol: {
    category: "protocol",
    maxConfidence: 1.0,
    decayHalfLife: null, // Eternal - constructed by protocol
    minConfidence: 1.0,
    reverificationBonus: 0,
    verificationMethod: "construction",
    requiresReverification: false,
    minVerifiers: 1, // Protocol itself verifies
    description: "On-chain state and protocol rules",
  },
  empirical: {
    category: "empirical",
    maxConfidence: 0.99, // Never 100% - always measurement error
    decayHalfLife: 100000, // ~2.9 days at 2521ms blocks
    minConfidence: 0.33,
    reverificationBonus: 0.11,
    verificationMethod: "observation",
    requiresReverification: true,
    minVerifiers: 5,
    description: "Observable facts about the physical world",
  },
  historical: {
    category: "historical",
    maxConfidence: 0.95, // Past is reconstructed, never certain
    decayHalfLife: 500000, // ~14.6 days - slower decay
    minConfidence: 0.22,
    reverificationBonus: 0.11,
    verificationMethod: "records",
    requiresReverification: true,
    minVerifiers: 5,
    description: "Claims about past events",
  },
  predictive: {
    category: "predictive",
    maxConfidence: 0.9, // Future is inherently uncertain
    decayHalfLife: 20000, // ~0.6 days - fast decay until resolved
    minConfidence: 0.22,
    reverificationBonus: 0.11,
    verificationMethod: "models",
    requiresReverification: true,
    minVerifiers: 7,
    description: "Predictions about future outcomes",
  },
  social: {
    category: "social",
    maxConfidence: 0.85, // Consensus-based, inherently contested
    decayHalfLife: 150000, // ~4.4 days - consensus is stickier than observation
    minConfidence: 0.22,
    reverificationBonus: 0.11,
    verificationMethod: "consensus",
    requiresReverification: true,
    minVerifiers: 10,
    description: "Socially constructed truths and values",
  },
  replicated: {
    category: "replicated",
    maxConfidence: 0.995, // Higher than empirical due to IRL verification
    decayHalfLife: 1000000, // ~29 days - very slow decay
    minConfidence: 0.55, // Higher floor due to real-world verification
    reverificationBonus: 0.05, // Less bonus needed (already verified IRL)
    verificationMethod: "replication",
    requiresReverification: true,
    minVerifiers: 5,
    description:
      "Empirical facts with successful IRL replication (2+ independent replications)",
  },
};

// ============ Multidimensional Confidence ============

/**
 * Temporal stability of confidence
 */
export type TemporalStability = "volatile" | "unstable" | "stable" | "anchored";

/**
 * Source coherence - how much evidence sources agree
 */
export type SourceCoherence =
  | "contradictory"
  | "mixed"
  | "consistent"
  | "unanimous";

/**
 * Epistemic basis - WHY we believe something
 */
export interface EpistemicBasis {
  category: EpistemicCategory;
  primaryMethod:
    | "definition"
    | "proof"
    | "construction"
    | "observation"
    | "records"
    | "models"
    | "consensus"
    | "replication";
  evidenceCount: number;
  verificationCount: number;
  derivedFrom: string[]; // Fact IDs this was derived from
}

/**
 * Confidence provenance - breakdown of how confidence was calculated
 */
export interface ConfidenceProvenance {
  evidenceContribution: number; // 0-1: How much evidence contributed
  verificationContribution: number; // 0-1: How much validator votes contributed
  temporalAdjustment: number; // Multiplier from decay/reverification
  adversarialAdjustment: number; // Multiplier from manipulation detection
  epistemicCeiling: number; // Max allowed by epistemic category
}

/**
 * Multidimensional Confidence
 *
 * Confidence is not a single number. It's a multidimensional epistemic state
 * that captures WHY we believe, HOW SURE we are, and WHAT COULD CHANGE our mind.
 *
 * This transparency enables AI agents to make rational economic decisions.
 */
export interface MultidimensionalConfidence {
  /** The headline number (0-1) - use this for quick comparisons */
  value: Confidence;

  /** Why we believe this - epistemic basis */
  basis: EpistemicBasis;

  /** How the confidence was calculated */
  provenance: ConfidenceProvenance;

  /** How stable is this confidence over time */
  stability: TemporalStability;

  /** How much do evidence sources agree */
  coherence: SourceCoherence;

  /** What would change this confidence */
  sensitivities: {
    toNewEvidence: number; // How much new evidence could shift it
    toContradiction: number; // How vulnerable to contradictory evidence
    toTimeDecay: number; // How fast it decays without reverification
  };

  /** Timestamp of calculation */
  calculatedAt: number;

  /** Block number of calculation */
  calculatedAtBlock: number;
}

// ============ Fact Maturity Levels ============

/**
 * Fact Maturity
 *
 * Facts progress through maturity levels based on confidence, age, and verifications.
 * Higher maturity = harder to challenge.
 */
export type FactMaturity = "emerging" | "established" | "canonical";

/**
 * Maturity promotion criteria
 */
export interface MaturityCriteria {
  minConfidence: Confidence;
  minAgeBlocks: number;
  minVerifications: number;
  challengeRules: string;
}

/**
 * Maturity level definitions
 *
 * EMERGING:    New facts that can be freely challenged
 * ESTABLISHED: Mature facts requiring evidence to challenge
 * CANONICAL:   Foundational facts requiring governance to dispute
 */
export const MATURITY_CRITERIA: Record<FactMaturity, MaturityCriteria> = {
  emerging: {
    minConfidence: 0.5,
    minAgeBlocks: 0,
    minVerifications: 1,
    challengeRules: "Can be challenged freely with any counter-evidence",
  },
  established: {
    minConfidence: 0.8,
    minAgeBlocks: 100000, // ~3 days
    minVerifications: 5,
    challengeRules: "Requires evidence of strength >= 0.7 to challenge",
  },
  canonical: {
    minConfidence: 0.95,
    minAgeBlocks: 500000, // ~15 days
    minVerifications: 10,
    challengeRules: "Requires governance vote to dispute",
  },
};

/**
 * Calculate current maturity level for a fact
 */
export function calculateMaturity(
  confidence: Confidence,
  ageBlocks: number,
  verificationCount: number,
): FactMaturity {
  const canonical = MATURITY_CRITERIA.canonical;
  if (
    confidence >= canonical.minConfidence &&
    ageBlocks >= canonical.minAgeBlocks &&
    verificationCount >= canonical.minVerifications
  ) {
    return "canonical";
  }

  const established = MATURITY_CRITERIA.established;
  if (
    confidence >= established.minConfidence &&
    ageBlocks >= established.minAgeBlocks &&
    verificationCount >= established.minVerifications
  ) {
    return "established";
  }

  return "emerging";
}

/**
 * Check if a challenge is allowed for a fact's maturity level
 */
export function canChallenge(
  maturity: FactMaturity,
  challengeEvidenceStrength: number,
  isGovernanceVote: boolean,
): { allowed: boolean; reason: string } {
  switch (maturity) {
    case "emerging":
      return {
        allowed: true,
        reason: "Emerging facts can be freely challenged",
      };

    case "established":
      if (challengeEvidenceStrength >= 0.7) {
        return {
          allowed: true,
          reason: "Sufficient evidence strength for established fact",
        };
      }
      return {
        allowed: false,
        reason:
          "Established facts require evidence strength >= 0.7 to challenge",
      };

    case "canonical":
      if (isGovernanceVote) {
        return {
          allowed: true,
          reason: "Governance vote can challenge canonical facts",
        };
      }
      return {
        allowed: false,
        reason: "Canonical facts require governance vote to challenge",
      };
  }
}

/**
 * Types of claims that can be verified
 */
export type ClaimType =
  | "mathematical" // Formal proofs, computations
  | "empirical" // Observable facts about the world
  | "historical" // Past events
  | "definitional" // Meanings and definitions
  | "computational" // Deterministic computation results
  | "protocol"; // On-chain state and rules

/**
 * Types of logical inference
 */
export type InferenceType =
  | "deductive" // If premises true, conclusion certain
  | "inductive" // Pattern suggests conclusion
  | "abductive"; // Best explanation for observation

/**
 * Inference strength by type
 */
export const INFERENCE_STRENGTH: Record<InferenceType, number> = {
  deductive: 0.99,
  inductive: 0.75,
  abductive: 0.6,
};

/**
 * Reference types for supporting evidence
 */
export type ReferenceType =
  | "on_chain" // Reference to other on-chain facts
  | "cryptographic_proof" // Zero-knowledge proofs, signatures
  | "computational" // Deterministic, reproducible computation
  | "formal_proof" // Mathematical demonstration
  | "peer_reviewed" // Academic verification
  | "institutional" // Government, recognized bodies
  | "oracle" // External data feeds
  | "archived_web" // Wayback machine, archived content
  | "attestation" // Human witness
  | "inference"; // Derived from other evidence

/**
 * Evidence type hierarchy with intrinsic strength values
 * Higher values = more reliable
 */
export const EVIDENCE_TYPE_STRENGTH: Record<ReferenceType, number> = {
  on_chain: 1.0, // Cryptographically verified on-chain
  cryptographic_proof: 0.99, // ZK proofs, cryptographic signatures
  computational: 0.95, // Deterministic, reproducible
  formal_proof: 0.95, // Mathematical demonstration
  peer_reviewed: 0.85, // Academic verification
  institutional: 0.8, // Government, recognized bodies
  oracle: 0.7, // External data feeds (trust the oracle)
  archived_web: 0.6, // Archived but unverified
  attestation: 0.5, // Human witness (bias possible)
  inference: 0.4, // Derived, inherits premise uncertainty
};

/**
 * Evidence independence metrics
 * Two newspaper articles from the same wire service are NOT independent.
 */
export interface EvidenceIndependence {
  /** Are sources organizationally independent? (0-1) */
  sourceIndependence: number;

  /** Were different methods used to obtain evidence? (0-1) */
  methodIndependence: number;

  /** Was evidence gathered at different times? (0-1) */
  temporalIndependence: number;

  /** Geographic independence (different locations) */
  geographicIndependence?: number;

  /** Computed aggregate independence score */
  aggregateIndependence: number;
}

/**
 * Verification verdict
 */
export type Verdict = "confirmed" | "rejected" | "uncertain";

/**
 * Fact status — unified lifecycle covering both claim verification and stored fact states.
 *
 * Claim verification stages:
 *   - pending: submitted, awaiting verification
 *   - verified: accepted by validators (transitions to 'active' in state)
 *   - contested: mixed verification results
 *   - disproven: proven false through verification
 *
 * Stored fact lifecycle:
 *   - active: currently accepted as true
 *   - challenged: under active dispute
 *   - superseded: replaced by newer fact
 *   - revoked: removed through governance/dispute
 *   - expired: time-expired or confidence decayed
 */
export type FactStatus =
  | "pending" // Claim submitted, awaiting verification
  | "verified" // Accepted by validators
  | "active" // Currently accepted in state
  | "provisional" // Adversarial: fact created during challenge window, can be overturned
  | "contested" // Mixed verification results
  | "challenged" // Under active dispute
  | "superseded" // Replaced by newer fact
  | "expired" // Time-expired or confidence decayed
  | "disproven" // Proven false through verification
  | "revoked"; // Removed through governance/dispute

/**
 * Axiom - foundational truth (confidence = 1.0)
 */
export interface Axiom {
  id: string;
  claim: string;
  type: "mathematical" | "protocol" | "definitional" | "founding";
  confidence: 1.0;
  establishedAt: number; // Block number
  governanceVote?: string; // If added through governance
  commitment?: string; // For founding axioms: hash commitment
  revealConditions?: string[]; // Conditions to reveal hidden meaning
}

/**
 * The Genesis Axioms
 *
 * axiom_0 is special - it contains a founding truth
 * that reveals itself only under specific conditions.
 */
export const GENESIS_AXIOM_IDS = {
  FOUNDING: "axiom_0", // The founding truth
  IDENTITY: "axiom_1", // A = A
  ARITHMETIC: "axiom_2", // 1 + 1 = 2
  PROTOCOL: "axiom_3", // LGM is the native token
} as const;

/**
 * Reference - supporting evidence for a claim
 */
export interface Reference {
  id: string;
  type: ReferenceType;
  source: string;
  contentHash: string;
  retrievedAt: number;
  reliabilityScore: number;

  /** Evidence independence relative to other references */
  independence?: EvidenceIndependence;

  /** Is this evidence supporting or contradicting the claim? */
  direction: "supporting" | "contradicting" | "neutral";

  /** Strength based on type (from EVIDENCE_TYPE_STRENGTH) */
  intrinsicStrength: number;

  metadata?: {
    url?: string;
    doi?: string;
    archiveUrl?: string;
    oracleId?: string;
    methodology?: string; // How evidence was gathered
    sourceOrganization?: string; // For independence checking
  };
}

/**
 * Verification - a validator's attestation
 */
export interface Verification {
  validator: DID;
  timestamp: number;
  blockNumber: number;
  verdict: Verdict;
  confidenceAssessment: number;
  stake: bigint;
  referencesChecked: string[]; // Reference IDs
  reasoning?: string;
}

/**
 * Derivation - logical chain from premises to conclusion
 */
export interface Derivation {
  fromFacts: string[]; // Fact IDs used as premises
  inferenceType: InferenceType;
  inferenceStrength: number;
  logicChain: string; // Formal representation
  derivedAt: number;
  derivedBy: DID;
}

/**
 * Fact - a verified claim in the knowledge layer
 */
export interface Fact {
  id: string;
  claim: string;
  claimHash: string; // Hash of normalized claim
  type: ClaimType;
  domain: string; // Knowledge domain (science, history, etc.)

  // Epistemic Classification
  epistemicCategory: EpistemicCategory;
  maturity: FactMaturity;

  // Confidence (simple)
  confidence: Confidence;
  confidenceHistory: Array<{
    timestamp: number;
    confidence: Confidence;
    reason: string;
  }>;

  // Multidimensional Confidence (rich)
  multidimensionalConfidence?: MultidimensionalConfidence;

  // Evidence
  references: Reference[];
  verifications: Verification[];
  derivedFrom?: Derivation;

  // Derived facts (for propagation)
  derivedFacts?: string[]; // Fact IDs derived from this fact

  // Provenance
  submittedBy: DID;
  submittedAt: number;
  lastVerified: number;
  createdAtBlock: number;

  // Status
  status: FactStatus;
  disputeId?: string;
  expiresAt?: number;
}

/**
 * Claim submission for verification
 *
 * Supports two modes:
 * 1. Schema-based (preferred): Structured data validated against a schema
 * 2. Legacy free-form: Unstructured claim string (deprecated)
 *
 * Schema-based claims enable:
 * - Type-safe validation at submission
 * - Dynamic verification timing based on complexity
 * - Automatic verification for deterministic claims
 * - Clear verification guidance for validators
 */
export interface ClaimSubmission {
  // ============ Schema-Based (Preferred) ============

  /**
   * Schema identifier (e.g., 'hash_verification_v1', 'oracle_price_v1')
   * When provided, claimData must match the schema's dataSchema
   */
  schemaId?: string;

  /**
   * Structured claim data matching the schema
   * Validated against schema.dataSchema before round creation
   */
  claimData?: Record<string, unknown>;

  // ============ Legacy Free-Form (Deprecated) ============

  /**
   * Free-form claim string
   * @deprecated Use schemaId + claimData for new claims
   */
  claim: string;

  // ============ Common Fields ============

  /** Claim type classification */
  type: ClaimType;

  /** Knowledge domain (e.g., 'science.neuroscience', 'crypto.prices') */
  domain: string;

  /** Supporting references/evidence */
  references: Omit<Reference, "id">[];

  /** Submitter's proposed confidence level */
  proposedConfidence: Confidence;

  /** Stake backing this claim */
  stake: bigint;

  /** If derived from existing facts */
  derivedFrom?: {
    factIds: string[];
    inferenceType: InferenceType;
    logicChain: string;
  };
}

/**
 * Check if a claim submission uses schema-based format
 */
export function isSchemaBasedClaim(submission: ClaimSubmission): boolean {
  return (
    submission.schemaId !== undefined && submission.claimData !== undefined
  );
}

/**
 * Get the claim text for display (schema-based claims generate from data)
 */
export function getClaimText(submission: ClaimSubmission): string {
  if (submission.schemaId && submission.claimData) {
    // Generate human-readable claim from structured data
    return `[${submission.schemaId}] ${JSON.stringify(submission.claimData)}`;
  }
  return submission.claim;
}

/**
 * Challenge to an existing fact
 */
export interface Challenge {
  id: string;
  factId: string;
  challenger: DID;
  reason: string;
  counterEvidence: Reference[];
  stake: bigint;
  status: "pending" | "upheld" | "rejected";
  submittedAt: number;
  resolvedAt?: number;
}

/**
 * Knowledge query parameters
 */
export interface KnowledgeQuery {
  claim?: string;
  factId?: string;
  domain?: string;
  minConfidence?: Confidence;
  maxAge?: number; // Blocks since last verification
  types?: ClaimType[];
  limit?: number;
  offset?: number;
}

/**
 * Knowledge query result
 */
export interface KnowledgeResult {
  facts: Fact[];
  totalCount: number;
  queryConfidence: Confidence; // Confidence in query results
}

/**
 * Verification task assigned to validators
 */
export interface VerificationTask {
  id: string;
  submissionId: string;
  claim: string;
  references: Reference[];
  assignedValidators: DID[];
  requiredVerifications: number;
  deadline: number;
  status: "pending" | "in_progress" | "complete";
  verifications: Verification[];
}

/**
 * Protocol parameters for truth verification
 */
export interface TruthParameters {
  // Verification
  minVerifiers: number; // Minimum validators per claim
  verificationQuorum: number; // % agreement needed
  verificationTimeout: number; // Blocks to complete verification

  // Confidence
  confidenceDecayRate: number; // Per-block decay without reverification
  minConfidenceThreshold: number; // Below this, fact expires

  // Stakes
  minSubmissionStake: bigint; // Minimum stake to submit claim
  minVerificationStake: bigint; // Minimum stake to verify
  minChallengeStake: bigint; // Minimum stake to challenge

  // Rewards
  submissionRewardBase: bigint; // Base reward for accepted submission
  verificationRewardBase: bigint; // Base reward for verification
  challengeRewardShare: number; // % of loser's stake to winner

  // Slashing
  wrongVerificationSlash: number; // % stake lost for wrong verification
  rejectedSubmissionSlash: number; // % stake lost for rejected submission
  failedChallengeSlash: number; // % stake lost for failed challenge
}

/**
 * Citation tracking
 */
export interface Citation {
  citingFactId: string;
  citedFactId: string;
  domain: string;
  timestamp: number;
  blockNumber: number;
}

/**
 * Fundamentality metrics for a fact
 */
export interface FundamentalityMetrics {
  citationCount: number;
  maxDerivationDepth: number;
  citingDomains: string[];
  citationVelocity: number; // Citations per 1000 blocks
  ageBlocks: number;
  fundamentalityScore: number; // 0.0 to 1.0
  crossDomainMultiplier: number;
}

/**
 * Citation reward distribution
 */
export interface CitationReward {
  factId: string;
  submitter: DID;
  share: number; // 0.0 to 1.0
  amount: bigint;
  reason:
    | "direct_citation"
    | "ancestor"
    | "cross_domain_bonus"
    | "fundamentality_bonus";
}

/**
 * Citation reward event (recorded on-chain)
 */
export interface CitationRewardEvent {
  id: string;
  factId: string;
  triggerType: "query" | "derivation" | "task" | "challenge";
  triggerFactId: string;
  amount: bigint;
  distributions: CitationReward[];
  timestamp: number;
  blockNumber: number;
}

/**
 * Extended fact with citation metrics
 */
export interface FactWithCitations extends Fact {
  // Citation metrics
  citationCount: number;
  citingDomains: string[];
  maxDerivationDepth: number;

  // Calculated scores
  fundamentalityScore: number;
  crossDomainMultiplier: number;

  // Accumulated rewards
  lifetimeCitationRewards: bigint;
  pendingRewards: bigint;
}

/**
 * Fundamentality thresholds for bonus rewards
 */
export const FUNDAMENTALITY_THRESHOLDS = {
  NOTABLE: 0.25, // 10% bonus
  FOUNDATIONAL: 0.5, // 25% bonus
  FUNDAMENTAL: 0.75, // 50% bonus
  AXIOMATIC: 0.9, // 100% bonus
} as const;

/**
 * Fundamentality bonus multipliers
 */
export const FUNDAMENTALITY_BONUSES = {
  NOTABLE: 1.1,
  FOUNDATIONAL: 1.25,
  FUNDAMENTAL: 1.5,
  AXIOMATIC: 2.0,
} as const;

/**
 * Citation economy parameters
 */
export interface CitationParameters {
  citationShare: number; // % of fees to citation pool (default: 0.15)
  decayRate: number; // Decay per ancestor level (default: 0.5)
  protocolFloor: number; // Minimum to protocol (default: 0.02)
  crossDomainBonusRate: number; // Per additional domain (default: 0.2)
  maxCrossDomainBonus: number; // Cap on cross-domain multiplier (default: 2.0)
}

export const DEFAULT_CITATION_PARAMETERS: CitationParameters = {
  citationShare: 0.15,
  decayRate: 0.5,
  protocolFloor: 0.02,
  crossDomainBonusRate: 0.2,
  maxCrossDomainBonus: 2.0,
};

/**
 * Default truth parameters
 */
// ============ Negative Knowledge System ============

/**
 * Negation types for counter-facts
 */
export type NegationType =
  | "rejection" // Claim evaluated and rejected by validators
  | "contradiction" // Explicit counter-claim with evidence
  | "falsification" // Research outcome = "refutes"
  | "supersession"; // Newer evidence overturns prior acceptance

/**
 * Counter-fact lifecycle status
 */
export type CounterFactStatus =
  | "pending" // Awaiting verification
  | "provisional" // In challenge window
  | "verified" // Confirmed false
  | "contested" // Mixed verification
  | "overturned"; // Counter-fact itself was disproved

/**
 * CounterFact represents a verified negative knowledge assertion.
 *
 * For any proposition P, the knowledge graph can now store:
 *   Fact(P)        → confidence_truth  [0, 1_000_000]  (existing)
 *   CounterFact(P) → confidence_false  [0, 1_000_000]  (new)
 *
 * Bidirectional confidence = truth - falsity (derived, can be negative).
 */
export interface CounterFact {
  id: string; // "cfact-{hash}"
  targetFactId: string; // Fact being negated (if exists)
  targetClaimId: string; // Original rejected claim
  negationType: NegationType;
  confidence: Confidence; // How confident we are it's false [0, 1.0]
  justification: string;
  evidenceIds: string[];
  submitter: DID;
  submitterAddress: string;
  verifiers: DID[];
  domain: string;
  logicZone: string;
  epistemicCategory: EpistemicCategory;
  status: CounterFactStatus;
  createdAtBlock: number;
  verifiedAtBlock?: number;
  roundId?: string;
  incompletenessAck?: boolean; // Gödelian: negation may be undecidable
}

/**
 * Bidirectional confidence for a proposition.
 * Truth and falsity are epistemically independent.
 */
export interface BidirectionalConfidence {
  factId: string;
  truthConfidence: Confidence;
  falsityConfidence: Confidence;
  netConfidence: number; // truth - falsity (can be negative)
  counterFactCount: number;
  status: "established" | "contested" | "disproven" | "uncertain";
}

/**
 * Compute bidirectional status from net confidence.
 */
export function computeBidirectionalStatus(
  net: number,
): BidirectionalConfidence["status"] {
  if (net > 0.6) return "established";
  if (net < -0.6) return "disproven";
  if (Math.abs(net) < 0.2) return "uncertain";
  return "contested";
}

/**
 * Negative knowledge constants (matching Go x/knowledge/types/types.go)
 */
export const NEGATIVE_KNOWLEDGE = {
  MAX_CONTRADICTIONS_PER_EPOCH: 3,
  MIN_CONTRADICTION_STAKE_RATIO: 0.5, // 50% of target fact stake
  FAILED_CONTRADICTION_SLASH: 0.5, // 50%
  FALSIFICATION_REWARD_CAP: 3.0, // 3× original stake
  FALSIFICATION_POOL_SHARE: 0.15, // 15% of epoch rewards
  COUNTER_FACT_HIGH_CONFIDENCE: 0.7,
  COUNTER_FACT_MEDIUM_CONFIDENCE: 0.4,
  COUNTER_FACT_HIGH_PENALTY: 0.08, // basis points scaled to 0-1
  COUNTER_FACT_MEDIUM_PENALTY: 0.04,
  COUNTER_FACT_LOW_PENALTY: 0.01,
  MAX_COUNTER_FACT_PENALTY: 0.4,
  ALREADY_DISPROVEN_PENALTY: 0.2,
} as const;

export const DEFAULT_TRUTH_PARAMETERS: TruthParameters = {
  minVerifiers: 5,
  verificationQuorum: 0.66, // 66% agreement
  verificationTimeout: 100, // ~100 blocks

  confidenceDecayRate: 0.0001, // 0.01% per block
  minConfidenceThreshold: 0.3,

  minSubmissionStake: 100n * 10n ** 15n, // 0.1 LGM (affordable with 0.222 LGM genesis balance)
  minVerificationStake: 50n * 10n ** 15n, // 0.05 LGM
  minChallengeStake: 200n * 10n ** 15n, // 0.2 LGM

  submissionRewardBase: 50n * 10n ** 15n, // 0.05 LGM
  verificationRewardBase: 10n * 10n ** 15n, // 0.01 LGM
  challengeRewardShare: 0.5, // 50% of loser's stake

  wrongVerificationSlash: 0.06, // 6% stake
  rejectedSubmissionSlash: 0.2, // 20% stake
  failedChallengeSlash: 0.3, // 30% stake
};
