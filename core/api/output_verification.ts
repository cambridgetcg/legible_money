/**
 * Output Verification Types for Anti-Hallucination System
 *
 * Type definitions for verifying AI agent outputs against on-chain knowledge.
 * Ensures agent claims are grounded in verified facts before submission.
 *
 * PP HIERARCHY:
 *   TRUTH: Types enable verification that claims cite verified facts
 *   UNDERSTANDING: ReasoningChainAnalysis tracks comprehension quality
 *   BEAUTY: Clean type separation, coherent interfaces
 *   JUSTICE: Placed in core/api alongside agentic.ts and philosophy.ts
 *   CREATIVITY: Novel types for output verification (new territory)
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { OntologicalStratum } from "../knowledge/ontology/types.js";
import type { PPScore, PPSignal } from "./philosophy.js";

// ============================================================================
// Core Types
// ============================================================================

/**
 * A claim that an agent wants to submit or output.
 *
 * Every agent output MUST include cited facts to demonstrate grounding
 * in verified on-chain knowledge.
 */
export interface AgentOutputClaim {
  /** Unique ID for tracking this output */
  outputId: string;

  /** The claim text being asserted */
  claim: string;

  /** Agent's self-assessed confidence (0-1) */
  selfConfidence: number;

  /** Domain the claim belongs to */
  domain: string;

  /** Agent's reasoning for this claim */
  reasoning: string;

  /** Fact IDs the agent claims support this - REQUIRED for grounding */
  citedFactIds: string[];

  /** Output type classification */
  outputType: OutputType;

  /** Timestamp of creation */
  createdAt: number;
}

/**
 * Types of agent outputs with different grounding requirements:
 * - assertion: Direct factual claim (highest grounding required)
 * - derivation: Logical conclusion from cited facts
 * - speculation: Exploratory claim (lowest grounding, must be marked)
 */
export type OutputType = "assertion" | "derivation" | "speculation";

// ============================================================================
// Citation and Grounding Types
// ============================================================================

/**
 * A citation that grounds an agent claim in verified knowledge.
 */
export interface GroundingCitation {
  /** The fact ID being cited */
  factId: string;

  /** The fact's claim text */
  factClaim: string;

  /** The fact's verified confidence from consensus */
  factConfidence: number;

  /** The fact's domain */
  factDomain: string;

  /** The fact's ontological stratum */
  factStratum: OntologicalStratum;

  /** How this fact relates to the agent's claim */
  relationship: CitationRelationship;

  /** Strength of the grounding (0-1) */
  groundingStrength: number;

  /** Semantic similarity score (0-1) */
  semanticSimilarity: number;
}

/**
 * Types of relationships between cited facts and agent claims.
 */
export type CitationRelationship =
  | "directly_supports" // Fact directly states the claim
  | "partially_supports" // Fact supports part of the claim
  | "implies" // Fact logically implies the claim
  | "analogous" // Fact is analogous (weaker grounding)
  | "contradicts"; // Fact contradicts the claim (no grounding)

// ============================================================================
// Verification Result Types
// ============================================================================

/**
 * Result of verifying an agent's output against on-chain knowledge.
 */
export interface OutputVerificationResult {
  /** The output being verified */
  outputId: string;

  /** Whether the output passed verification */
  passed: boolean;

  /** Overall groundedness score (0-1) */
  groundednessScore: number;

  /** Effective confidence after verification adjustments */
  effectiveConfidence: number;

  /** Citations that ground this output */
  citations: GroundingCitation[];

  /** Confidence ceiling from ontological stratum */
  ontologyCeiling: number;

  /** Cross-stratum discount applied (0-1, 1 = no discount) */
  crossStratumDiscount: number;

  /** PP scores for this output */
  ppScores: PPScore;

  /** PP signals detected */
  signals: PPSignal[];

  /** Reasoning chain analysis */
  reasoningChainAnalysis: ReasoningChainAnalysis;

  /** Detailed breakdown of verification components */
  breakdown: VerificationBreakdown;

  /** Failure reasons (if not passed) */
  failureReasons?: string[];

  /** Suggestions for improving groundedness */
  suggestions?: string[];
}

/**
 * Detailed breakdown of verification scoring components.
 */
export interface VerificationBreakdown {
  /** What percentage of the claim is grounded in citations */
  citationCoverage: number;

  /** Average quality of citations */
  averageCitationQuality: number;

  /** Strongest citation strength */
  maxCitationStrength: number;

  /** Weakest citation strength */
  minCitationStrength: number;

  /** How well self-confidence matches evidence (1 = perfect calibration) */
  confidenceCalibration: number;

  /** Ontological consistency score */
  ontologicalConsistency: number;

  /** Individual PP dimension scores */
  ppDimensions: {
    truth: number;
    understanding: number;
    beauty: number;
    justice: number;
    creativity: number;
  };
}

// ============================================================================
// Reasoning Chain Types
// ============================================================================

/**
 * A single node in the reasoning chain.
 */
export interface ReasoningNode {
  /** Step number in the chain */
  stepNumber: number;

  /** The claim or derivation at this step */
  claim: string;

  /** Type of reasoning step */
  stepType: ReasoningStepType;

  /** If grounded, the fact ID providing grounding */
  groundedFactId?: string;

  /** If inference, what type of inference */
  inferenceType?: InferenceType;

  /** Confidence at this step */
  confidence: number;

  /** Distance from nearest grounded knowledge (0 = directly grounded) */
  groundingDistance: number;

  /** Ontological stratum at this step */
  stratum?: OntologicalStratum;
}

/**
 * Types of reasoning steps:
 * - grounded_fact: Directly citing a verified fact (distance 0)
 * - inference: Logical derivation from grounded facts
 * - assumption: Unstated assumption (should be minimized)
 * - speculation: Exploratory claim beyond evidence
 */
export type ReasoningStepType =
  | "grounded_fact"
  | "inference"
  | "assumption"
  | "speculation";

/**
 * Types of logical inference:
 * - deductive: Certain conclusion from premises
 * - inductive: Probable conclusion from patterns
 * - abductive: Best explanation for observations
 */
export type InferenceType = "deductive" | "inductive" | "abductive";

/**
 * Analysis of a complete reasoning chain.
 */
export interface ReasoningChainAnalysis {
  /** The reasoning chain nodes */
  chain: ReasoningNode[];

  /** Total chain length */
  chainLength: number;

  /** Number of grounded nodes (directly citing facts) */
  groundedNodeCount: number;

  /** Number of inference nodes */
  inferenceNodeCount: number;

  /** Number of assumption nodes */
  assumptionNodeCount: number;

  /** Number of speculation nodes */
  speculationNodeCount: number;

  /** Grounding ratio: groundedNodes / totalNodes */
  groundingRatio: number;

  /** Maximum distance from any node to grounded knowledge */
  maxGroundingDistance: number;

  /** Whether chain crosses ontological stratum boundaries */
  crossesStratumBoundaries: boolean;

  /** Strata traversed in the chain */
  strataTraversed: OntologicalStratum[];

  /** Confidence decay through the chain (final / initial) */
  confidenceDecay: number;

  /** Drift score: how far reasoning has drifted from grounded knowledge (0-1) */
  driftScore: number;
}

// ============================================================================
// Session-Level Types
// ============================================================================

/**
 * Session state for tracking reasoning chains and drift.
 */
export interface ReasoningSessionState {
  /** Session ID */
  sessionId: string;

  /** Agent DID */
  agentDid: DID;

  /** Current reasoning chain being built */
  currentChain: ReasoningNode[];

  /** Facts accessed during this session (factId -> citation) */
  accessedFacts: Map<string, GroundingCitation>;

  /** Cumulative drift score across all outputs this session */
  cumulativeDrift: number;

  /** Number of ungrounded assertions in this session */
  ungroundedAssertionCount: number;

  /** Number of times grounding was reset this session */
  groundingResetCount: number;

  /** Most recent grounding anchor (last verified fact accessed) */
  lastGroundingAnchor?: GroundingCitation;

  /** Threshold for drift warning */
  driftWarningThreshold: number;

  /** Threshold for forced grounding reset */
  driftResetThreshold: number;

  /** Session start time */
  startedAt: number;

  /** Last activity time */
  lastActivityAt: number;
}

// ============================================================================
// Configuration Types
// ============================================================================

/**
 * Configuration for the anti-hallucination system.
 */
export interface AntiHallucinationConfig {
  /** Minimum groundedness score to pass verification (0-1) */
  minGroundednessScore: number;

  /** Minimum citation coverage required (0-1) */
  minCitationCoverage: number;

  /** Maximum allowed grounding distance (inference steps from facts) */
  maxGroundingDistance: number;

  /** Confidence decay multiplier per inference step */
  confidenceDecayPerStep: number;

  /** Drift score threshold for warning */
  driftWarningThreshold: number;

  /** Drift score threshold for forced reset */
  driftResetThreshold: number;

  /** Cross-stratum penalty multiplier per boundary (e.g., 0.95 = 5% penalty) */
  crossStratumPenalty: number;

  /** Whether to enforce ontological confidence ceilings */
  enforceOntologyCeilings: boolean;

  /** Minimum PP truth dimension score */
  minTruthScore: number;

  /** Whether speculation outputs are allowed */
  allowSpeculation: boolean;

  /** Maximum grounding distance for speculation */
  maxSpeculationDistance: number;
}

/**
 * Default configuration values for anti-hallucination system.
 */
export const DEFAULT_ANTI_HALLUCINATION_CONFIG: AntiHallucinationConfig = {
  minGroundednessScore: 0.6, // 60% minimum groundedness
  minCitationCoverage: 0.5, // 50% of claim must be grounded
  maxGroundingDistance: 5, // Max 5 inference steps from facts
  confidenceDecayPerStep: 0.9, // 10% decay per inference step
  driftWarningThreshold: 3, // Warn at drift score 3
  driftResetThreshold: 5, // Force reset at drift score 5
  crossStratumPenalty: 0.95, // 5% penalty per stratum boundary
  enforceOntologyCeilings: true, // Enforce stratum confidence ceilings
  minTruthScore: 0.5, // Minimum PP truth score
  allowSpeculation: true, // Allow marked speculation
  maxSpeculationDistance: 3, // Speculation max 3 steps from grounded
};

// ============================================================================
// Events
// ============================================================================

/**
 * Event emitted when verification fails.
 */
export interface VerificationFailureEvent {
  /** The output that failed */
  outputId: string;

  /** Agent DID */
  agentDid: DID;

  /** Failure reasons */
  reasons: string[];

  /** Groundedness score achieved */
  groundednessScore: number;

  /** Suggestions for improvement */
  suggestions: string[];

  /** Timestamp */
  timestamp: number;
}

/**
 * Event emitted when drift warning threshold is exceeded.
 */
export interface DriftWarningEvent {
  /** Session ID */
  sessionId: string;

  /** Agent DID */
  agentDid: DID;

  /** Current drift score */
  driftScore: number;

  /** Warning threshold */
  threshold: number;

  /** Suggestion */
  suggestion: string;

  /** Timestamp */
  timestamp: number;
}

/**
 * Event emitted when grounding is forcefully reset.
 */
export interface GroundingResetEvent {
  /** Session ID */
  sessionId: string;

  /** Agent DID */
  agentDid: DID;

  /** Drift score that triggered reset */
  driftScore: number;

  /** Number of resets this session */
  resetCount: number;

  /** Timestamp */
  timestamp: number;
}
