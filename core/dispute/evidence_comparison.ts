/**
 * Evidence Comparison System
 *
 * Calculates and compares evidence strength for disputes.
 * Uses weighted scoring across 5 dimensions:
 * - Type weight (25%): Evidence type reliability
 * - Reliability (30%): Source reliability score
 * - Recency (15%): How fresh the evidence is
 * - Independence (20%): Number of independent sources
 * - Corroboration (10%): Cross-references to verified facts
 *
 * PP HIERARCHY:
 *   TRUTH: Evidence strength must be objectively measurable
 *   UNDERSTANDING: Multi-dimensional scoring captures nuance
 *   BEAUTY: Simple formula, clear weights, predictable outcomes
 *   JUSTICE: Fair comparison - both parties scored equally
 */

import type { Evidence, EvidenceType } from "../evidence/types.js";
import { calculateFreshness } from "../evidence/expiry.js";
import {
  DISPUTE_EVIDENCE_WEIGHTS,
  EVIDENCE_SCORE_WEIGHTS,
  type EvidenceScore,
  type EvidenceComparisonResult,
} from "./types.js";

// ============ Constants ============

/**
 * Threshold for clear winner determination
 * If strength difference > threshold, auto-resolution is possible
 */
export const CLEAR_WINNER_THRESHOLD = 0.11;

/**
 * Maximum evidence items per side (anti-spam)
 */
export const MAX_EVIDENCE_PER_SIDE = 22;

/**
 * Recency scoring parameters
 */
export const RECENCY_PARAMS = {
  /** Evidence submitted within this many blocks gets full score */
  fullScoreBlocks: 1111,

  /** Evidence older than this many blocks gets minimum score */
  minScoreBlocks: 111111,

  /** Minimum recency score */
  minScore: 0.22,
} as const;

/**
 * Independence scoring parameters
 */
export const INDEPENDENCE_PARAMS = {
  /** Maximum sources to count (diminishing returns) */
  maxSources: 11,

  /** Weight per independent source (logarithmic) */
  perSourceWeight: 0.11,
} as const;

// ============ Individual Score Calculations ============

/**
 * Calculate type weight for evidence
 */
export function calculateTypeWeight(evidenceType: EvidenceType): number {
  return DISPUTE_EVIDENCE_WEIGHTS[evidenceType] ?? 0.5;
}

/**
 * Calculate recency score for evidence
 * Fresher evidence scores higher
 */
export function calculateRecencyScore(
  evidence: Evidence,
  currentBlock: bigint,
): number {
  // First, use the expiry system's freshness if available
  const freshness = calculateFreshness(evidence, currentBlock);
  if (freshness < 1.0) {
    // Evidence is decaying or stale - use freshness directly
    return Math.max(RECENCY_PARAMS.minScore, freshness);
  }

  // For eternal evidence, calculate based on submission time
  const blocksSinceSubmission = Number(
    currentBlock - evidence.submittedAtBlock,
  );

  if (blocksSinceSubmission <= RECENCY_PARAMS.fullScoreBlocks) {
    return 1.0;
  }

  if (blocksSinceSubmission >= RECENCY_PARAMS.minScoreBlocks) {
    return RECENCY_PARAMS.minScore;
  }

  // Linear decay between full and min
  const decayRange =
    RECENCY_PARAMS.minScoreBlocks - RECENCY_PARAMS.fullScoreBlocks;
  const decayProgress = blocksSinceSubmission - RECENCY_PARAMS.fullScoreBlocks;
  const decayRatio = decayProgress / decayRange;

  return 1.0 - (1.0 - RECENCY_PARAMS.minScore) * decayRatio;
}

/**
 * Calculate independence score
 * More independent sources = higher score
 */
export function calculateIndependenceScore(evidence: Evidence): number {
  // Use evidence's independence property which has multiple dimensions
  const independence = evidence.independence;

  if (!independence) {
    return 0.5; // Default neutral score
  }

  // Combine independence dimensions with weights
  const score =
    (independence.sourceIndependence ?? 0.5) * 0.4 +
    (independence.methodIndependence ?? 0.5) * 0.3 +
    (independence.temporalIndependence ?? 0.5) * 0.2 +
    (independence.geographicIndependence ?? 0.5) * 0.1;

  return Math.min(1.0, Math.max(0, score));
}

/**
 * Calculate corroboration score
 * How well this evidence connects to other verified facts
 */
export function calculateCorroborationScore(
  evidence: Evidence,
  _verifiedFactIds: Set<string>,
): number {
  // Check how many of the evidence's related facts are verified
  const relatedFacts = evidence.relatedFactIds || [];

  if (relatedFacts.length === 0) {
    return 0.5; // Neutral - no connections
  }

  let verifiedCount = 0;
  for (const factId of relatedFacts) {
    if (_verifiedFactIds.has(factId)) {
      verifiedCount++;
    }
  }

  const verificationRatio = verifiedCount / relatedFacts.length;

  // Scale from 0.3 (no corroboration) to 1.0 (full corroboration)
  return 0.3 + 0.7 * verificationRatio;
}

// ============ Aggregate Scoring ============

/**
 * Calculate complete evidence score for a single piece of evidence
 */
export function calculateEvidenceScore(
  evidence: Evidence,
  currentBlock: bigint,
  verifiedFactIds: Set<string>,
): EvidenceScore {
  return {
    typeWeight: calculateTypeWeight(evidence.type),
    reliabilityScore: evidence.reliabilityScore,
    recencyScore: calculateRecencyScore(evidence, currentBlock),
    independenceScore: calculateIndependenceScore(evidence),
    corroborationScore: calculateCorroborationScore(evidence, verifiedFactIds),
  };
}

/**
 * Calculate weighted combined score from score components
 */
export function calculateWeightedScore(score: EvidenceScore): number {
  return (
    EVIDENCE_SCORE_WEIGHTS.type * score.typeWeight +
    EVIDENCE_SCORE_WEIGHTS.reliability * score.reliabilityScore +
    EVIDENCE_SCORE_WEIGHTS.recency * score.recencyScore +
    EVIDENCE_SCORE_WEIGHTS.independence * score.independenceScore +
    EVIDENCE_SCORE_WEIGHTS.corroboration * score.corroborationScore
  );
}

/**
 * Aggregate scores from multiple pieces of evidence
 * Uses weighted average with diminishing returns for additional evidence
 */
export function aggregateEvidenceStrength(
  evidenceList: Evidence[],
  currentBlock: bigint,
  verifiedFactIds: Set<string>,
): { strength: number; scores: EvidenceScore[] } {
  if (evidenceList.length === 0) {
    return { strength: 0, scores: [] };
  }

  // Limit evidence count (anti-spam)
  const limitedEvidence = evidenceList.slice(0, MAX_EVIDENCE_PER_SIDE);

  // Calculate individual scores
  const scores: EvidenceScore[] = limitedEvidence.map((e) =>
    calculateEvidenceScore(e, currentBlock, verifiedFactIds),
  );

  // Calculate weighted scores
  const weightedScores = scores.map(calculateWeightedScore);

  // Sort by strength (best first)
  const sortedScores = [...weightedScores].sort((a, b) => b - a);

  // Aggregate with diminishing returns
  // First piece of evidence: 100% weight
  // Second: 77% weight (wisdom)
  // Third: 55% weight (change)
  // Fourth+: 33% weight each (trinity)
  const diminishingWeights = [
    1.0, 0.77, 0.55, 0.33, 0.33, 0.33, 0.33, 0.33, 0.33, 0.33, 0.33,
  ];

  let totalWeight = 0;
  let weightedSum = 0;

  for (
    let i = 0;
    i < sortedScores.length && i < diminishingWeights.length;
    i++
  ) {
    const weight = diminishingWeights[i];
    weightedSum += sortedScores[i] * weight;
    totalWeight += weight;
  }

  // Handle more than 11 pieces of evidence
  for (let i = diminishingWeights.length; i < sortedScores.length; i++) {
    weightedSum += sortedScores[i] * 0.22; // 22% weight for extras
    totalWeight += 0.22;
  }

  const strength = totalWeight > 0 ? weightedSum / totalWeight : 0;

  return { strength, scores };
}

// ============ Comparison ============

/**
 * Compare evidence between challenger and defender
 */
export function compareEvidence(
  challengerEvidence: Evidence[],
  defenderEvidence: Evidence[],
  currentBlock: bigint,
  verifiedFactIds: Set<string>,
): EvidenceComparisonResult {
  const challengerResult = aggregateEvidenceStrength(
    challengerEvidence,
    currentBlock,
    verifiedFactIds,
  );

  const defenderResult = aggregateEvidenceStrength(
    defenderEvidence,
    currentBlock,
    verifiedFactIds,
  );

  const strengthDifference =
    challengerResult.strength - defenderResult.strength;
  const absoluteDifference = Math.abs(strengthDifference);

  // Determine if there's a clear winner
  const isClearWinner = absoluteDifference >= CLEAR_WINNER_THRESHOLD;

  let suggestedWinner: "challenger" | "defender" | undefined;
  if (isClearWinner) {
    suggestedWinner = strengthDifference > 0 ? "challenger" : "defender";
  }

  return {
    challengerStrength: challengerResult.strength,
    defenderStrength: defenderResult.strength,
    strengthDifference,
    isClearWinner,
    suggestedWinner,
    challengerScores: challengerResult.scores,
    defenderScores: defenderResult.scores,
  };
}

// ============ Auto-Resolution ============

/**
 * Determine if dispute can be auto-resolved based on evidence
 * Auto-resolution saves time and gas when outcome is clear
 */
export function canAutoResolve(comparison: EvidenceComparisonResult): boolean {
  // Must have clear winner
  if (!comparison.isClearWinner) {
    return false;
  }

  // Winner must have minimum strength (prevent weak vs weaker)
  const winnerStrength =
    comparison.suggestedWinner === "challenger"
      ? comparison.challengerStrength
      : comparison.defenderStrength;

  return winnerStrength >= 0.5;
}

/**
 * Get auto-resolution result if applicable
 */
export function getAutoResolutionResult(comparison: EvidenceComparisonResult): {
  canResolve: boolean;
  winner?: "challenger" | "defender";
  confidence: number;
} {
  if (!canAutoResolve(comparison)) {
    return { canResolve: false, confidence: 0 };
  }

  const confidence = Math.min(
    0.99,
    0.7 + Math.abs(comparison.strengthDifference) * 2,
  );

  return {
    canResolve: true,
    winner: comparison.suggestedWinner,
    confidence,
  };
}

// ============ Evidence Validation ============

/**
 * Valid evidence types for disputes
 * GAP-015 FIX: Explicit type whitelist
 */
const VALID_DISPUTE_EVIDENCE_TYPES: Set<EvidenceType> = new Set([
  "on_chain",
  "cryptographic_proof",
  "formal_proof",
  "computational",
  "peer_reviewed",
  "institutional",
  "oracle",
  "archived_web",
  "attestation",
  "inference",
]);

/**
 * Maximum age of evidence in blocks (roughly 30 days)
 */
const MAX_EVIDENCE_AGE_BLOCKS = 1_000_000n;

/**
 * Validate evidence is suitable for dispute
 * GAP-015 FIX: Now validates types, timestamps, and other properties
 */
export function validateDisputeEvidence(
  evidence: Evidence[],
  currentBlock?: bigint,
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  if (evidence.length === 0) {
    errors.push("At least one piece of evidence is required");
  }

  if (evidence.length > MAX_EVIDENCE_PER_SIDE) {
    errors.push(`Maximum ${MAX_EVIDENCE_PER_SIDE} pieces of evidence allowed`);
  }

  const now = Date.now();

  for (const e of evidence) {
    // GAP-015 FIX: Validate evidence type
    if (!VALID_DISPUTE_EVIDENCE_TYPES.has(e.type)) {
      errors.push(`Evidence ${e.id} has invalid type: ${e.type}`);
    }

    // Check evidence is active
    if (e.status === "expired" || e.status === "invalidated") {
      errors.push(`Evidence ${e.id} is ${e.status}`);
    }

    // Check evidence has minimum reliability
    if (e.reliabilityScore < 0.1) {
      errors.push(
        `Evidence ${e.id} has insufficient reliability (${e.reliabilityScore})`,
      );
    }

    // GAP-015 FIX: Validate reliability is in valid range
    if (e.reliabilityScore > 1.0) {
      errors.push(
        `Evidence ${e.id} has invalid reliability score: ${e.reliabilityScore} (max 1.0)`,
      );
    }

    // GAP-015 FIX: Validate timestamp is not in the future
    if (e.submittedAt > now) {
      errors.push(`Evidence ${e.id} has future timestamp`);
    }

    // GAP-015 FIX: Validate evidence is not too old (if current block provided)
    if (currentBlock !== undefined && e.submittedAtBlock !== undefined) {
      const age = currentBlock - e.submittedAtBlock;
      if (age > MAX_EVIDENCE_AGE_BLOCKS) {
        errors.push(`Evidence ${e.id} is too old (${age} blocks)`);
      }

      // Validate block number is not in the future
      if (e.submittedAtBlock > currentBlock) {
        errors.push(`Evidence ${e.id} claims future block number`);
      }
    }

    // GAP-015 FIX: Validate source is provided
    if (!e.originalSource || e.originalSource.trim() === "") {
      errors.push(`Evidence ${e.id} is missing source attribution`);
    }

    // GAP-015 FIX: Validate content hash exists
    if (!e.contentHash || e.contentHash.length < 16) {
      errors.push(`Evidence ${e.id} has invalid or missing content hash`);
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

// ============ Commitment Verification ============

/**
 * Generate evidence commitment hash
 * Used in commit-reveal scheme to prevent evidence copying
 */
export function generateEvidenceCommitment(
  evidenceIds: string[],
  nonce: string,
): string {
  // Sort for determinism
  const sorted = [...evidenceIds].sort();
  const data = sorted.join(",") + ":" + nonce;

  // Simple hash (would use proper crypto in production)
  let hash = 0;
  for (let i = 0; i < data.length; i++) {
    const char = data.charCodeAt(i);
    hash = (hash << 5) - hash + char;
    hash = hash & hash; // Convert to 32bit integer
  }

  return "commit_" + Math.abs(hash).toString(16).padStart(16, "0");
}

/**
 * Verify revealed evidence matches commitment
 */
export function verifyEvidenceCommitment(
  evidenceIds: string[],
  nonce: string,
  commitment: string,
): boolean {
  const computed = generateEvidenceCommitment(evidenceIds, nonce);
  return computed === commitment;
}
