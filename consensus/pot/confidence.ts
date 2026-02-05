/**
 * Confidence Aggregation
 *
 * How we combine individual validator confidence assessments
 * into a final confidence score for a fact.
 *
 * The formula is PUBLISHED and VERIFIABLE.
 * Any agent can compute the same result.
 *
 * Enhanced with:
 * - Bayesian evidence aggregation
 * - Evidence independence modeling
 * - Contradiction handling
 * - Epistemic category ceilings
 * - Multidimensional confidence output
 */

import type { Confidence } from "../../core/knowledge/types.js";
import type { ValidatorReveal } from "./types.js";
import type { FactResearchSummary } from "../../core/research/types.js";
import { RESEARCH_CONFIDENCE_MODIFIERS } from "../../core/research/types.js";

// Domain Capture Defense imports
import type {
  IsolatedReputation,
  EffectiveReputationResult,
} from "../reputation_isolation/types.js";
import type { DomainTree } from "../../core/knowledge/ontology/types.js";
import {
  getEffectiveReputation,
  REPUTATION_ISOLATION_CONFIG,
} from "../reputation_isolation/index.js";

// ============ Confidence Calculation ============

/**
 * The confidence aggregation formula
 *
 * final_confidence =
 *   (stake_weighted_confidence * STAKE_WEIGHT) +
 *   (reputation_weighted_confidence * REPUTATION_WEIGHT) +
 *   (agreement_bonus * AGREEMENT_WEIGHT)
 *
 * All weights are PUBLISHED:
 */
export const CONFIDENCE_WEIGHTS = {
  STAKE: 0.5, // 50% weight on stake-weighted confidence
  REPUTATION: 0.3, // 30% weight on reputation-weighted confidence
  AGREEMENT: 0.2, // 20% bonus for high agreement levels
} as const;

/**
 * Time decay parameters (legacy - use category-specific decay instead)
 */
export const TIME_DECAY = {
  HALF_LIFE_BLOCKS: 111111, // Confidence halves every ~111k blocks (~3.2 days)
  MIN_CONFIDENCE: 0.33, // Minimum confidence before fact expires
  REVERIFICATION_BONUS: 0.11, // Bonus for successful re-verification
} as const;

/**
 * Category-specific decay rates
 *
 * Key insight: Eternal truths don't decay. Empirical facts do.
 * Mathematical proofs remain valid forever.
 * Weather observations become stale quickly.
 *
 * NEW: 'replicated' category for facts with successful IRL replication.
 * Replicated facts have very slow decay due to real-world verification.
 */
export const CATEGORY_DECAY: Record<
  string,
  {
    halfLifeBlocks: number | null; // null = no decay
    minConfidence: number;
    reverificationBonus: number;
  }
> = {
  analytic: {
    halfLifeBlocks: null, // Eternal - true by definition
    minConfidence: 1.0,
    reverificationBonus: 0,
  },
  formal: {
    halfLifeBlocks: null, // Eternal - mathematically proven
    minConfidence: 1.0,
    reverificationBonus: 0,
  },
  protocol: {
    halfLifeBlocks: null, // Eternal - constructed by protocol
    minConfidence: 1.0,
    reverificationBonus: 0,
  },
  empirical: {
    halfLifeBlocks: 111111, // ~3.2 days
    minConfidence: 0.33,
    reverificationBonus: 0.11,
  },
  historical: {
    halfLifeBlocks: 555555, // ~16 days - slower decay
    minConfidence: 0.22,
    reverificationBonus: 0.11,
  },
  predictive: {
    halfLifeBlocks: 22222, // Fast decay until resolved
    minConfidence: 0.22,
    reverificationBonus: 0.11,
  },
  social: {
    halfLifeBlocks: 111111, // ~3.2 days
    minConfidence: 0.22,
    reverificationBonus: 0.11,
  },
  // Replicated category for IRL-verified facts
  replicated: {
    halfLifeBlocks: 1111111, // ~32 days - very slow decay
    minConfidence: 0.55, // Higher floor due to real-world verification
    reverificationBonus: 0.05, // Less bonus needed (already verified IRL)
  },
};

/**
 * Calculate final confidence from validator reveals
 */
export function calculateFinalConfidence(
  reveals: Map<string, ValidatorReveal>,
  validators: Map<string, { stake: bigint; reputation: number }>,
  consensusVerdict: "confirmed" | "rejected",
): {
  confidence: Confidence;
  breakdown: ConfidenceBreakdown;
} {
  // Filter to consensus-matching reveals
  const matchingReveals = Array.from(reveals.values()).filter(
    (r) => r.verdict === consensusVerdict,
  );

  if (matchingReveals.length === 0) {
    return {
      confidence: consensusVerdict === "rejected" ? 0 : 0.5,
      breakdown: {
        stakeWeighted: 0,
        reputationWeighted: 0,
        agreementBonus: 0,
        participantCount: 0,
        totalStake: 0n,
      },
    };
  }

  // Calculate stake-weighted confidence using bigint basis-point arithmetic
  // to avoid Number() overflow on real stakes (1111 LGM = 1.111×10²¹)
  let stakeWeightedBps = 0n;
  let totalStake = 0n;

  for (const reveal of matchingReveals) {
    const validator = validators.get(reveal.validator);
    if (validator) {
      const confidenceBps = BigInt(Math.round(reveal.confidence * 10000));
      stakeWeightedBps += confidenceBps * validator.stake;
      totalStake += validator.stake;
    }
  }

  const stakeWeighted =
    totalStake > 0n ? Number(stakeWeightedBps / totalStake) / 10000 : 0;

  // Calculate reputation-weighted confidence
  let reputationWeightedSum = 0;
  let totalReputation = 0;

  for (const reveal of matchingReveals) {
    const validator = validators.get(reveal.validator);
    if (validator) {
      reputationWeightedSum += reveal.confidence * validator.reputation;
      totalReputation += validator.reputation;
    }
  }

  const reputationWeighted =
    totalReputation > 0 ? reputationWeightedSum / totalReputation : 0;

  // Calculate agreement bonus
  // GAP-011 FIX: Guard against division by zero
  const totalReveals = reveals.size;
  const matchingCount = matchingReveals.length;
  const agreementLevel = totalReveals > 0 ? matchingCount / totalReveals : 0;

  // Agreement bonus scales: 0 at 50% agreement, max at 100% agreement
  const agreementBonus = Math.max(0, (agreementLevel - 0.5) * 2);

  // Combine with weights
  const confidence =
    stakeWeighted * CONFIDENCE_WEIGHTS.STAKE +
    reputationWeighted * CONFIDENCE_WEIGHTS.REPUTATION +
    agreementBonus * CONFIDENCE_WEIGHTS.AGREEMENT;

  // Clamp to [0, 1]
  const finalConfidence = Math.max(0, Math.min(1, confidence));

  return {
    confidence: finalConfidence,
    breakdown: {
      stakeWeighted,
      reputationWeighted,
      agreementBonus,
      participantCount: matchingCount,
      totalStake,
    },
  };
}

export interface ConfidenceBreakdown {
  stakeWeighted: number;
  reputationWeighted: number;
  agreementBonus: number;
  participantCount: number;
  totalStake: bigint;
}

// ============ Time Decay ============

/**
 * Calculate decayed confidence based on blocks since verification (legacy)
 */
export function calculateDecayedConfidence(
  initialConfidence: Confidence,
  blocksSinceVerification: number,
): Confidence {
  // Exponential decay: C(t) = C_0 * 0.5^(t/half_life)
  const decayFactor = Math.pow(
    0.5,
    blocksSinceVerification / TIME_DECAY.HALF_LIFE_BLOCKS,
  );
  const decayed = initialConfidence * decayFactor;

  return Math.max(TIME_DECAY.MIN_CONFIDENCE, decayed);
}

/**
 * Calculate decayed confidence with epistemic category awareness
 *
 * Eternal truths (analytic, formal, protocol) never decay.
 * Empirical facts decay based on their category's half-life.
 */
export function calculateCategoryDecayedConfidence(
  initialConfidence: Confidence,
  blocksSinceVerification: number,
  category: string,
): {
  confidence: Confidence;
  decayFactor: number;
  isEternal: boolean;
} {
  const decayParams = CATEGORY_DECAY[category] || CATEGORY_DECAY.empirical;

  // Eternal truths don't decay
  if (decayParams.halfLifeBlocks === null) {
    return {
      confidence: initialConfidence,
      decayFactor: 1.0,
      isEternal: true,
    };
  }

  // Exponential decay: C(t) = C_0 * 0.5^(t/half_life)
  const decayFactor = Math.pow(
    0.5,
    blocksSinceVerification / decayParams.halfLifeBlocks,
  );
  const decayed = initialConfidence * decayFactor;

  return {
    confidence: Math.max(decayParams.minConfidence, decayed),
    decayFactor,
    isEternal: false,
  };
}

/**
 * Get decay rate description for a category
 */
export function getDecayDescription(category: string): string {
  const params = CATEGORY_DECAY[category];
  if (!params || params.halfLifeBlocks === null) {
    return "eternal (no decay)";
  }

  const days = (params.halfLifeBlocks * 2.521) / (24 * 60 * 60 * 1000);
  return `half-life: ~${days.toFixed(1)} days`;
}

/**
 * Check if a fact needs re-verification
 */
export function needsReverification(
  currentConfidence: Confidence,
  threshold: Confidence = 0.5,
): boolean {
  return currentConfidence < threshold;
}

/**
 * Calculate confidence after successful re-verification
 */
export function applyReverificationBonus(
  currentConfidence: Confidence,
  newVerificationConfidence: Confidence,
): Confidence {
  // Take the higher of current and new, plus a small bonus
  const base = Math.max(currentConfidence, newVerificationConfidence);
  const withBonus = base + TIME_DECAY.REVERIFICATION_BONUS;

  return Math.min(1, withBonus);
}

// ============ Evidence Strength ============

/**
 * Evidence type strength hierarchy (imported from types, mirrored here for runtime)
 */
const EVIDENCE_STRENGTH: Record<string, number> = {
  on_chain: 1.0,
  cryptographic_proof: 0.99,
  computational: 0.95,
  formal_proof: 0.95,
  peer_reviewed: 0.85,
  institutional: 0.8,
  oracle: 0.7,
  archived_web: 0.6,
  attestation: 0.5,
  inference: 0.4,
};

/**
 * Calculate evidence strength based on references (legacy function)
 */
export function calculateEvidenceStrength(
  references: Array<{
    type: string;
    reliabilityScore: number;
  }>,
): number {
  if (references.length === 0) return 0;

  let weightedSum = 0;
  let totalWeight = 0;

  for (const ref of references) {
    const typeWeight = EVIDENCE_STRENGTH[ref.type] || 0.5;
    const combinedWeight = typeWeight * ref.reliabilityScore;
    weightedSum += combinedWeight;
    totalWeight += 1;
  }

  return weightedSum / totalWeight;
}

// ============ Bayesian Evidence Aggregation ============

/**
 * Prior probability used when no evidence exists
 * Represents maximum ignorance
 */
export const DEFAULT_PRIOR = 0.5;

/**
 * Evidence item for Bayesian aggregation
 */
export interface BayesianEvidence {
  id: string;
  type: string;
  strength: number; // Intrinsic strength (0-1)
  direction: "supporting" | "contradicting" | "neutral";
  independence: {
    sourceIndependence: number;
    methodIndependence: number;
    temporalIndependence: number;
    aggregateIndependence: number;
  };
  reliabilityScore: number;
}

/**
 * Calculate likelihood ratio for a piece of evidence
 *
 * L = P(E|H) / P(E|¬H)
 *
 * For supporting evidence: L > 1 (increases belief)
 * For contradicting evidence: L < 1 (decreases belief)
 */
export function calculateLikelihoodRatio(evidence: BayesianEvidence): number {
  const baseStrength = evidence.strength * evidence.reliabilityScore;

  if (evidence.direction === "neutral") {
    return 1.0; // No update
  }

  if (evidence.direction === "supporting") {
    // P(E|H) is high, P(E|¬H) is low
    // Stronger evidence = higher ratio
    const pEgivenH = 0.5 + baseStrength * 0.45; // 0.5 to 0.95
    const pEgivenNotH = 0.5 - baseStrength * 0.35; // 0.5 to 0.15
    return pEgivenH / pEgivenNotH;
  } else {
    // Contradicting: P(E|H) is low, P(E|¬H) is high
    const pEgivenH = 0.5 - baseStrength * 0.35;
    const pEgivenNotH = 0.5 + baseStrength * 0.45;
    return pEgivenH / pEgivenNotH;
  }
}

/**
 * Combine likelihood ratios based on independence
 *
 * Independent evidence: multiply ratios (stronger)
 * Correlated evidence: weighted geometric mean (weaker)
 */
export function combineLikelihoodRatios(
  evidenceItems: BayesianEvidence[],
): number {
  if (evidenceItems.length === 0) return 1.0;
  if (evidenceItems.length === 1) {
    return calculateLikelihoodRatio(evidenceItems[0]);
  }

  let combinedRatio = 1.0;

  for (let i = 0; i < evidenceItems.length; i++) {
    const evidence = evidenceItems[i];
    const ratio = calculateLikelihoodRatio(evidence);

    // Calculate independence from all previous evidence
    const avgIndependence = evidence.independence.aggregateIndependence;

    // For correlated evidence, dampen the effect
    // Independent: full multiplicative effect
    // Correlated: weighted geometric mean with existing
    const effectiveExponent = avgIndependence;
    const dampedRatio = Math.pow(ratio, effectiveExponent);

    combinedRatio *= dampedRatio;
  }

  return combinedRatio;
}

/**
 * Convert odds to probability
 * P = O / (1 + O)
 */
function oddsToProb(odds: number): number {
  return odds / (1 + odds);
}

/**
 * Convert probability to odds
 * O = P / (1 - P)
 */
function probToOdds(prob: number): number {
  return prob / (1 - prob);
}

/**
 * Apply Bayes' theorem to update belief
 *
 * posterior_odds = prior_odds × likelihood_ratio
 */
export function bayesianUpdate(prior: number, likelihoodRatio: number): number {
  const priorOdds = probToOdds(Math.max(0.001, Math.min(0.999, prior)));
  const posteriorOdds = priorOdds * likelihoodRatio;
  return oddsToProb(posteriorOdds);
}

/**
 * Handle contradictory evidence
 *
 * When evidence conflicts, the maximum achievable confidence is capped.
 * Genuine disagreement means uncertainty regardless of validator consensus.
 */
export function handleContradiction(
  supportingStrength: number,
  contradictingStrength: number,
): {
  netStrength: number;
  uncertaintyPenalty: number;
  maxAchievableConfidence: number;
} {
  const netStrength = supportingStrength - contradictingStrength;
  const conflictSeverity = Math.min(supportingStrength, contradictingStrength);
  const uncertaintyPenalty = conflictSeverity;

  // Cap max confidence based on contradiction level
  // Full contradiction (equal strength) caps at 0.7
  const maxAchievableConfidence = 1.0 - 0.3 * uncertaintyPenalty;

  return {
    netStrength,
    uncertaintyPenalty,
    maxAchievableConfidence,
  };
}

/**
 * Aggregate evidence using Bayesian framework
 */
export function aggregateEvidenceBayesian(
  evidenceItems: BayesianEvidence[],
  prior: number = DEFAULT_PRIOR,
): {
  posterior: number;
  supportingStrength: number;
  contradictingStrength: number;
  contradictionPenalty: number;
  effectiveConfidence: number;
} {
  // Separate supporting and contradicting evidence
  const supporting = evidenceItems.filter((e) => e.direction === "supporting");
  const contradicting = evidenceItems.filter(
    (e) => e.direction === "contradicting",
  );

  // Calculate aggregate strength for each
  const supportingStrength =
    supporting.reduce((sum, e) => sum + e.strength * e.reliabilityScore, 0) /
    Math.max(1, supporting.length);

  const contradictingStrength =
    contradicting.reduce((sum, e) => sum + e.strength * e.reliabilityScore, 0) /
    Math.max(1, contradicting.length);

  // Handle contradiction
  const { maxAchievableConfidence, uncertaintyPenalty } = handleContradiction(
    supporting.length > 0 ? supportingStrength : 0,
    contradicting.length > 0 ? contradictingStrength : 0,
  );

  // Calculate likelihood ratio from all evidence
  const combinedRatio = combineLikelihoodRatios(evidenceItems);

  // Apply Bayesian update
  const posterior = bayesianUpdate(prior, combinedRatio);

  // Apply contradiction cap
  const effectiveConfidence = Math.min(posterior, maxAchievableConfidence);

  return {
    posterior,
    supportingStrength,
    contradictingStrength,
    contradictionPenalty: uncertaintyPenalty,
    effectiveConfidence,
  };
}

// ============ Confidence for Derivations ============

/**
 * Calculate confidence for a derived fact
 *
 * Derived confidence = min(premise_confidences) * inference_strength
 */
export function calculateDerivedConfidence(
  premiseConfidences: Confidence[],
  inferenceStrength: number,
): Confidence {
  if (premiseConfidences.length === 0) return 0;

  // Take minimum of all premises (weakest link)
  const minPremise = Math.min(...premiseConfidences);

  // Multiply by inference strength
  return minPremise * inferenceStrength;
}

/**
 * Inference strength by type
 */
export const INFERENCE_STRENGTHS: Record<string, number> = {
  deductive: 0.99, // If premises true, conclusion certain
  modus_ponens: 0.99,
  modus_tollens: 0.99,
  inductive: 0.77, // Pattern suggests conclusion
  abductive: 0.55, // Best explanation
  analogy: 0.55, // Similar cases
};

// ============ Confidence Queries ============

/**
 * Get confidence category
 */
export function getConfidenceCategory(confidence: Confidence): string {
  if (confidence >= 0.99) return "near_certain";
  if (confidence >= 0.88) return "high";
  if (confidence >= 0.77) return "moderate";
  if (confidence >= 0.55) return "low";
  return "unreliable";
}

/**
 * Check if confidence meets threshold
 */
export function meetsConfidenceThreshold(
  confidence: Confidence,
  threshold: Confidence,
): boolean {
  return confidence >= threshold;
}

/**
 * Format confidence for display
 */
export function formatConfidence(confidence: Confidence): string {
  const percent = (confidence * 100).toFixed(1);
  const category = getConfidenceCategory(confidence);
  return `${percent}% (${category})`;
}

// ============ Confidence History ============

export interface ConfidenceHistoryEntry {
  timestamp: number;
  blockNumber: number;
  confidence: Confidence;
  reason: string;
  verificationRoundId?: string;
}

/**
 * Create confidence history entry
 */
export function createHistoryEntry(
  confidence: Confidence,
  blockNumber: number,
  reason: string,
  roundId?: string,
): ConfidenceHistoryEntry {
  return {
    timestamp: Date.now(),
    blockNumber,
    confidence,
    reason,
    verificationRoundId: roundId,
  };
}

/**
 * Get confidence trend from history
 */
export function getConfidenceTrend(
  history: ConfidenceHistoryEntry[],
): "increasing" | "decreasing" | "stable" {
  if (history.length < 2) return "stable";

  const recent = history.slice(-5);
  const first = recent[0].confidence;
  const last = recent[recent.length - 1].confidence;

  const diff = last - first;
  if (diff > 0.05) return "increasing";
  if (diff < -0.05) return "decreasing";
  return "stable";
}

// ============ Master Confidence Formula ============

/**
 * Epistemic category profiles (mirrored from types for runtime)
 */
const EPISTEMIC_CEILINGS: Record<string, number> = {
  analytic: 1.0,
  formal: 1.0,
  protocol: 1.0,
  empirical: 0.99,
  historical: 0.95,
  predictive: 0.9,
  social: 0.85,
  // NEW: Replicated empirical facts can reach higher confidence
  replicated: 0.995,
};

/**
 * Input for master confidence calculation
 */
export interface ConfidenceCalculationInput {
  // Evidence
  evidence: BayesianEvidence[];

  // Verification (from commit-reveal round)
  verificationReveals: Map<string, ValidatorReveal>;
  validators: Map<string, { stake: bigint; reputation: number }>;
  consensusVerdict: "confirmed" | "rejected";

  // Temporal
  category: string;
  blocksSinceLastVerification: number;
  currentBlockNumber: number;

  // Prior state (if updating existing fact)
  priorConfidence?: number;

  // Adversarial adjustments (from manipulation detection)
  sybilAdjustment?: number; // 0-1, 1 = no sybil detected
  manipulationAdjustment?: number; // 0-1, 1 = no manipulation detected
}

/**
 * Determine temporal stability from history
 */
function calculateStability(
  history: ConfidenceHistoryEntry[],
): "volatile" | "unstable" | "stable" | "anchored" {
  if (history.length < 3) return "unstable";

  const recent = history.slice(-10);
  const values = recent.map((h) => h.confidence);
  const variance = calculateVariance(values);

  if (variance < 0.01) return "anchored";
  if (variance < 0.05) return "stable";
  if (variance < 0.15) return "unstable";
  return "volatile";
}

function calculateVariance(values: number[]): number {
  if (values.length === 0) return 0;
  const mean = values.reduce((a, b) => a + b, 0) / values.length;
  const squaredDiffs = values.map((v) => Math.pow(v - mean, 2));
  return squaredDiffs.reduce((a, b) => a + b, 0) / values.length;
}

/**
 * Determine source coherence from evidence
 */
function calculateCoherence(
  evidence: BayesianEvidence[],
): "contradictory" | "mixed" | "consistent" | "unanimous" {
  const supporting = evidence.filter(
    (e) => e.direction === "supporting",
  ).length;
  const contradicting = evidence.filter(
    (e) => e.direction === "contradicting",
  ).length;
  const total = supporting + contradicting;

  if (total === 0) return "consistent";

  const agreementRatio = Math.max(supporting, contradicting) / total;

  if (agreementRatio >= 1.0) return "unanimous";
  if (agreementRatio >= 0.9) return "consistent";
  if (agreementRatio >= 0.7) return "mixed";
  return "contradictory";
}

/**
 * THE MASTER CONFIDENCE FORMULA
 *
 * C_final = min(
 *   C_max,                                    // Epistemic ceiling
 *   C_base × M_temporal × M_adversarial       // Computed confidence
 * )
 *
 * Where:
 *   C_max = epistemic_category.maxConfidence
 *   C_base = combine(bayesian_evidence, multi_modal_verification)
 *   M_temporal = decay_factor × reverification_bonus
 *   M_adversarial = sybil_adjustment × manipulation_adjustment
 */
export function calculateMasterConfidence(
  input: ConfidenceCalculationInput,
  history: ConfidenceHistoryEntry[] = [],
): {
  value: number;
  basis: {
    category: string;
    primaryMethod: string;
    evidenceCount: number;
    verificationCount: number;
    derivedFrom: string[];
  };
  provenance: {
    evidenceContribution: number;
    verificationContribution: number;
    temporalAdjustment: number;
    adversarialAdjustment: number;
    epistemicCeiling: number;
  };
  stability: "volatile" | "unstable" | "stable" | "anchored";
  coherence: "contradictory" | "mixed" | "consistent" | "unanimous";
  sensitivities: {
    toNewEvidence: number;
    toContradiction: number;
    toTimeDecay: number;
  };
  calculatedAt: number;
  calculatedAtBlock: number;
} {
  // 1. Get epistemic ceiling
  const epistemicCeiling =
    EPISTEMIC_CEILINGS[input.category] || EPISTEMIC_CEILINGS.empirical;

  // 2. Calculate evidence-based confidence (Bayesian)
  const prior = input.priorConfidence ?? DEFAULT_PRIOR;
  const evidenceResult = aggregateEvidenceBayesian(input.evidence, prior);
  const evidenceConfidence = evidenceResult.effectiveConfidence;

  // 3. Calculate verification-based confidence
  const verificationResult = calculateFinalConfidence(
    input.verificationReveals,
    input.validators,
    input.consensusVerdict,
  );
  const verificationConfidence = verificationResult.confidence;

  // 4. Combine evidence and verification
  // Weight: 40% evidence, 60% verification (verification is more robust to manipulation)
  const EVIDENCE_WEIGHT = 0.4;
  const VERIFICATION_WEIGHT = 0.6;
  const baseConfidence =
    evidenceConfidence * EVIDENCE_WEIGHT +
    verificationConfidence * VERIFICATION_WEIGHT;

  // 5. Apply temporal adjustment
  const { confidence: decayedConfidence, decayFactor } =
    calculateCategoryDecayedConfidence(
      baseConfidence,
      input.blocksSinceLastVerification,
      input.category,
    );
  const temporalAdjustment = decayFactor;

  // 6. Apply adversarial adjustments
  const sybilAdj = input.sybilAdjustment ?? 1.0;
  const manipulationAdj = input.manipulationAdjustment ?? 1.0;
  const adversarialAdjustment = sybilAdj * manipulationAdj;

  // GAP-017 FIX: Apply epistemic ceiling BEFORE adversarial adjustment
  // This ensures adversarial factors can reduce confidence below the ceiling
  // (e.g., sybil detection should reduce even ceiling-level confidence)

  // 7. Apply epistemic ceiling first
  const ceiledConfidence = Math.min(epistemicCeiling, decayedConfidence);

  // 8. Then apply adversarial adjustment
  const finalValue = ceiledConfidence * adversarialAdjustment;

  // 9. Calculate metadata
  const stability = calculateStability(history);
  const coherence = calculateCoherence(input.evidence);

  // 10. Calculate sensitivities
  const toNewEvidence = 1.0 - input.evidence.length / 10; // More evidence = less sensitive
  const toContradiction =
    coherence === "unanimous" ? 0.9 : coherence === "consistent" ? 0.6 : 0.3;
  const decayParams =
    CATEGORY_DECAY[input.category] || CATEGORY_DECAY.empirical;
  const toTimeDecay = decayParams.halfLifeBlocks === null ? 0 : 0.5;

  return {
    value: finalValue,
    basis: {
      category: input.category,
      primaryMethod: getCategoryMethod(input.category),
      evidenceCount: input.evidence.length,
      verificationCount: input.verificationReveals.size,
      derivedFrom: [],
    },
    provenance: {
      evidenceContribution: evidenceConfidence * EVIDENCE_WEIGHT,
      verificationContribution: verificationConfidence * VERIFICATION_WEIGHT,
      temporalAdjustment,
      adversarialAdjustment,
      epistemicCeiling,
    },
    stability,
    coherence,
    sensitivities: {
      toNewEvidence: Math.max(0, Math.min(1, toNewEvidence)),
      toContradiction,
      toTimeDecay,
    },
    calculatedAt: Date.now(),
    calculatedAtBlock: input.currentBlockNumber,
  };
}

function getCategoryMethod(category: string): string {
  const methods: Record<string, string> = {
    analytic: "definition",
    formal: "proof",
    protocol: "construction",
    empirical: "observation",
    historical: "records",
    predictive: "models",
    social: "consensus",
  };
  return methods[category] || "observation";
}

// ============ Research-Enhanced Confidence ============

/**
 * Apply research evidence modifiers to fact confidence
 *
 * This integrates findings from the Protocol Research Repository
 * into the confidence calculation. Research evidence can:
 * - BOOST confidence (successful replications)
 * - INVALIDATE facts (fraud detection)
 * - UPGRADE epistemic category (empirical -> replicated)
 *
 * @param baseConfidence - Confidence from validator consensus
 * @param researchSummary - Summary of research evidence for this fact
 * @returns Modified confidence and category upgrade info
 */
export function applyResearchEnhancement(
  baseConfidence: Confidence,
  researchSummary: FactResearchSummary,
): {
  enhancedConfidence: Confidence;
  modifier: number;
  shouldUpgradeCategory: boolean;
  newCategory?: string;
  fraudDetected: boolean;
  reasoning: string;
} {
  // CRITICAL: Fraud detection immediately invalidates
  if (researchSummary.fraudDetected) {
    return {
      enhancedConfidence: 0,
      modifier: RESEARCH_CONFIDENCE_MODIFIERS.FRAUD_DETECTED,
      shouldUpgradeCategory: false,
      fraudDetected: true,
      reasoning: "Fraud detected in research investigation - fact invalidated",
    };
  }

  let modifier = 0;
  const reasons: string[] = [];

  // Apply replication bonuses
  if (researchSummary.successfulReplications > 0) {
    // First successful replication
    modifier += RESEARCH_CONFIDENCE_MODIFIERS.REPLICATION_SUCCESS;
    reasons.push(
      `+${RESEARCH_CONFIDENCE_MODIFIERS.REPLICATION_SUCCESS} (first replication)`,
    );

    // Additional replications (capped at MAX_REPLICATION_BONUS)
    const additionalReplications = Math.min(
      researchSummary.successfulReplications - 1,
      5,
    );
    const additionalBonus =
      additionalReplications *
      RESEARCH_CONFIDENCE_MODIFIERS.ADDITIONAL_REPLICATION;
    modifier += additionalBonus;

    if (additionalReplications > 0) {
      reasons.push(
        `+${additionalBonus.toFixed(2)} (${additionalReplications} additional replications)`,
      );
    }
  }

  // Penalize failed replications
  if (researchSummary.failedReplications > 0) {
    const penalty = researchSummary.failedReplications * 0.1;
    modifier -= penalty;
    reasons.push(
      `-${penalty.toFixed(2)} (${researchSummary.failedReplications} failed replications)`,
    );
  }

  // Bonus for passing fraud investigation
  if (
    researchSummary.fraudInvestigations > 0 &&
    !researchSummary.fraudDetected
  ) {
    modifier += RESEARCH_CONFIDENCE_MODIFIERS.FRAUD_INVESTIGATION_CLEAN;
    reasons.push(
      `+${RESEARCH_CONFIDENCE_MODIFIERS.FRAUD_INVESTIGATION_CLEAN} (passed fraud investigation)`,
    );
  }

  // Cap modifier at MAX_REPLICATION_BONUS
  modifier = Math.max(
    -1,
    Math.min(RESEARCH_CONFIDENCE_MODIFIERS.MAX_REPLICATION_BONUS, modifier),
  );

  // Apply modifier to base confidence
  const enhancedConfidence = Math.max(
    0,
    Math.min(1, baseConfidence + modifier),
  );

  // Determine if fact should upgrade to 'replicated' category
  const shouldUpgradeCategory =
    researchSummary.successfulReplications >= 2 && modifier > 0;

  return {
    enhancedConfidence,
    modifier,
    shouldUpgradeCategory,
    newCategory: shouldUpgradeCategory ? "replicated" : undefined,
    fraudDetected: false,
    reasoning:
      reasons.length > 0 ? reasons.join("; ") : "No research evidence applied",
  };
}

/**
 * Calculate research-adjusted confidence for a fact
 *
 * This is the primary entry point for agents querying fact confidence.
 * It combines validator consensus with research evidence.
 */
export function calculateResearchAdjustedConfidence(
  validatorConfidence: Confidence,
  category: string,
  researchSummary: FactResearchSummary | null,
  blocksSinceVerification: number,
): {
  confidence: Confidence;
  effectiveCategory: string;
  researchApplied: boolean;
  fraudDetected: boolean;
  components: {
    validatorConfidence: Confidence;
    researchModifier: number;
    temporalDecay: number;
    finalConfidence: Confidence;
  };
} {
  // Start with base confidence
  let confidence = validatorConfidence;
  let effectiveCategory = category;
  let researchModifier = 0;
  let fraudDetected = false;
  let researchApplied = false;

  // Apply research enhancement if available
  if (
    researchSummary &&
    (researchSummary.successfulReplications > 0 ||
      researchSummary.failedReplications > 0 ||
      researchSummary.fraudInvestigations > 0)
  ) {
    const enhancement = applyResearchEnhancement(confidence, researchSummary);
    confidence = enhancement.enhancedConfidence;
    researchModifier = enhancement.modifier;
    fraudDetected = enhancement.fraudDetected;
    researchApplied = true;

    if (enhancement.shouldUpgradeCategory && enhancement.newCategory) {
      effectiveCategory = enhancement.newCategory;
    }
  }

  // Apply temporal decay with effective category
  const { confidence: decayedConfidence, decayFactor } =
    calculateCategoryDecayedConfidence(
      confidence,
      blocksSinceVerification,
      effectiveCategory,
    );

  return {
    confidence: decayedConfidence,
    effectiveCategory,
    researchApplied,
    fraudDetected,
    components: {
      validatorConfidence,
      researchModifier,
      temporalDecay: decayFactor,
      finalConfidence: decayedConfidence,
    },
  };
}

/**
 * Determine if a fact needs IRL research verification
 *
 * High-stakes empirical facts should be flagged for research bounties.
 */
export function needsResearchVerification(
  confidence: Confidence,
  category: string,
  challengeCount: number,
  verifierCount: number,
): {
  needsResearch: boolean;
  priority: "low" | "medium" | "high" | "critical";
  reason: string;
} {
  // Analytic/formal/protocol don't need IRL verification
  if (["analytic", "formal", "protocol"].includes(category)) {
    return {
      needsResearch: false,
      priority: "low",
      reason: "Category does not require IRL verification",
    };
  }

  // High challenge count = needs investigation
  if (challengeCount >= 3) {
    return {
      needsResearch: true,
      priority: "critical",
      reason: "Multiple challenges indicate disputed fact",
    };
  }

  // Moderate confidence with challenges = medium priority
  if (confidence < 0.8 && challengeCount >= 1) {
    return {
      needsResearch: true,
      priority: "high",
      reason: "Challenged fact with moderate confidence",
    };
  }

  // Low verifier count = may need additional verification
  if (verifierCount < 5 && category === "empirical") {
    return {
      needsResearch: true,
      priority: "medium",
      reason: "Empirical fact with limited validator coverage",
    };
  }

  // High confidence empirical facts benefit from replication
  if (confidence > 0.9 && category === "empirical") {
    return {
      needsResearch: true,
      priority: "low",
      reason: "High-confidence empirical fact - replication recommended",
    };
  }

  return {
    needsResearch: false,
    priority: "low",
    reason: "No immediate research needed",
  };
}

/**
 * Convert master calculation result to MultidimensionalConfidence type
 */
export function toMultidimensionalConfidence(
  result: ReturnType<typeof calculateMasterConfidence>,
): {
  value: number;
  basis: {
    category: string;
    primaryMethod: string;
    evidenceCount: number;
    verificationCount: number;
    derivedFrom: string[];
  };
  provenance: {
    evidenceContribution: number;
    verificationContribution: number;
    temporalAdjustment: number;
    adversarialAdjustment: number;
    epistemicCeiling: number;
  };
  stability: "volatile" | "unstable" | "stable" | "anchored";
  coherence: "contradictory" | "mixed" | "consistent" | "unanimous";
  sensitivities: {
    toNewEvidence: number;
    toContradiction: number;
    toTimeDecay: number;
  };
  calculatedAt: number;
  calculatedAtBlock: number;
} {
  return result;
}

// ============ Isolated Reputation-Aware Confidence ============

/**
 * Calculate final confidence using isolated (domain-specific) reputation
 *
 * This replaces the global reputation with domain-specific reputation,
 * preventing reputation from one domain being used to gain influence
 * in another domain.
 *
 * Key changes from calculateFinalConfidence:
 * - Uses domain-specific reputation instead of global
 * - Applies reputation discount based on source (domain=100%, stratum=70%, global=50%)
 * - Factors in reputation confidence (sample size)
 */
export function calculateIsolatedReputationConfidence(
  reveals: Map<string, ValidatorReveal>,
  validators: Map<string, { stake: bigint; reputation: number }>,
  isolatedReputations: Map<string, IsolatedReputation>,
  domain: string,
  domainTree: DomainTree,
  consensusVerdict: "confirmed" | "rejected",
): {
  confidence: Confidence;
  breakdown: IsolatedConfidenceBreakdown;
} {
  // Filter to consensus-matching reveals
  const matchingReveals = Array.from(reveals.values()).filter(
    (r) => r.verdict === consensusVerdict,
  );

  if (matchingReveals.length === 0) {
    return {
      confidence: consensusVerdict === "rejected" ? 0 : 0.5,
      breakdown: {
        stakeWeighted: 0,
        reputationWeighted: 0,
        agreementBonus: 0,
        participantCount: 0,
        totalStake: 0n,
        reputationSources: { domain: 0, stratum: 0, global: 0 },
        averageReputationConfidence: 0,
      },
    };
  }

  // Calculate stake-weighted confidence using bigint basis-point arithmetic
  // to avoid Number() overflow on real stakes (1111 LGM = 1.111×10²¹)
  let stakeWeightedBps = 0n;
  let totalStake = 0n;

  for (const reveal of matchingReveals) {
    const validator = validators.get(reveal.validator);
    if (validator) {
      const confidenceBps = BigInt(Math.round(reveal.confidence * 10000));
      stakeWeightedBps += confidenceBps * validator.stake;
      totalStake += validator.stake;
    }
  }

  const stakeWeighted =
    totalStake > 0n ? Number(stakeWeightedBps / totalStake) / 10000 : 0;

  // Calculate isolated reputation-weighted confidence
  let reputationWeightedSum = 0;
  let totalReputationWeight = 0;
  let totalReputationConfidence = 0;

  // Track reputation sources
  const reputationSources = { domain: 0, stratum: 0, global: 0 };

  for (const reveal of matchingReveals) {
    const isolatedRep = isolatedReputations.get(reveal.validator);

    // Get effective reputation for this domain
    let effectiveRep: EffectiveReputationResult;
    if (isolatedRep) {
      effectiveRep = getEffectiveReputation(
        reveal.validator,
        domain,
        isolatedRep,
        domainTree,
      );
      reputationSources[effectiveRep.source]++;
    } else {
      // No isolated reputation - use global with maximum discount
      effectiveRep = {
        score: REPUTATION_ISOLATION_CONFIG.baseScore,
        source: "global",
        discount: REPUTATION_ISOLATION_CONFIG.globalReputationDiscount,
        confidence: 0,
      };
      reputationSources.global++;
    }

    // Normalize score (0-1000 -> 0-1)
    const normalizedScore =
      effectiveRep.score / REPUTATION_ISOLATION_CONFIG.maxScore;

    // Apply discount based on source
    const effectiveScore = normalizedScore * effectiveRep.discount;

    // Weight by reputation confidence (sample size)
    const weight = 0.5 + effectiveRep.confidence * 0.5; // 0.5 to 1.0

    reputationWeightedSum += reveal.confidence * effectiveScore * weight;
    totalReputationWeight += weight;
    totalReputationConfidence += effectiveRep.confidence;
  }

  const reputationWeighted =
    totalReputationWeight > 0
      ? reputationWeightedSum / totalReputationWeight
      : 0;

  const averageReputationConfidence =
    matchingReveals.length > 0
      ? totalReputationConfidence / matchingReveals.length
      : 0;

  // Calculate agreement bonus
  const totalReveals = reveals.size;
  const matchingCount = matchingReveals.length;
  const agreementLevel = totalReveals > 0 ? matchingCount / totalReveals : 0;
  const agreementBonus = Math.max(0, (agreementLevel - 0.5) * 2);

  // Combine with weights
  const confidence =
    stakeWeighted * CONFIDENCE_WEIGHTS.STAKE +
    reputationWeighted * CONFIDENCE_WEIGHTS.REPUTATION +
    agreementBonus * CONFIDENCE_WEIGHTS.AGREEMENT;

  // Clamp to [0, 1]
  const finalConfidence = Math.max(0, Math.min(1, confidence));

  return {
    confidence: finalConfidence,
    breakdown: {
      stakeWeighted,
      reputationWeighted,
      agreementBonus,
      participantCount: matchingCount,
      totalStake,
      reputationSources,
      averageReputationConfidence,
    },
  };
}

/**
 * Extended confidence breakdown including isolated reputation details
 */
export interface IsolatedConfidenceBreakdown extends ConfidenceBreakdown {
  /** Count of validators using each reputation source */
  reputationSources: {
    domain: number;
    stratum: number;
    global: number;
  };
  /** Average reputation confidence (sample size confidence) */
  averageReputationConfidence: number;
}
