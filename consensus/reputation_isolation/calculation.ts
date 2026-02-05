/**
 * Reputation Isolation - Score Calculation
 *
 * Domain-specific reputation calculation with:
 * - Laplace smoothing for new validators
 * - Time decay toward neutral
 * - Multi-component scoring (accuracy, consistency, defense, calibration)
 */

import type { PublicKey } from "../../core/crypto/keys.js";
import type {
  OntologicalStratum,
  DomainTree,
} from "../../core/knowledge/ontology/types.js";
import {
  type IsolatedReputation,
  type DomainReputation,
  type StratumReputation,
  type EffectiveReputationResult,
  type ReputationScoreResult,
  REPUTATION_ISOLATION_CONFIG,
  createEmptyDomainReputation,
  createEmptyStratumReputation,
  hasDomainReputation,
  hasStratumReputation,
} from "./types.js";

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Get domain stratum from domain tree
 */
function getDomainStratum(
  domain: string,
  domainTree: DomainTree,
): OntologicalStratum {
  const node = domainTree.domainIndex.get(domain);
  return node?.domain.stratum ?? ("empirical" as unknown as OntologicalStratum);
}

// ============================================================================
// Score Calculation
// ============================================================================

/**
 * Calculate domain-specific reputation score
 *
 * Score components (must sum to 1.0):
 * - Accuracy: 50% - correctVerifications / totalVerifications
 * - Consistency: 25% - how consistent verdicts are
 * - Challenge Defense: 15% - defending against challenges
 * - Calibration: 10% - how calibrated confidence is
 */
export function calculateDomainReputationScore(
  metrics: {
    totalVerifications: number;
    correctVerifications: number;
    verdictConsistency: number;
    challengesReceived: number;
    challengesLost: number;
    confidenceCalibration: number;
  },
  currentBlock: bigint,
  lastActivityBlock: bigint,
): ReputationScoreResult {
  const { laplacePrior, laplaceTotal } = REPUTATION_ISOLATION_CONFIG;

  // 1. Accuracy score with Laplace smoothing
  // (correct + prior) / (total + prior_total) prevents 0/0 and rewards activity
  const accuracyRaw =
    (metrics.correctVerifications + laplacePrior) /
    (metrics.totalVerifications + laplaceTotal);
  const accuracyScore = accuracyRaw * REPUTATION_ISOLATION_CONFIG.maxScore;

  // 2. Consistency score (0-1 input, scale to 0-1000)
  const consistencyScore =
    metrics.verdictConsistency * REPUTATION_ISOLATION_CONFIG.maxScore;

  // 3. Challenge defense score
  let challengeDefenseScore = REPUTATION_ISOLATION_CONFIG.baseScore; // Neutral if no challenges
  if (metrics.challengesReceived > 0) {
    const defenseRate = 1 - metrics.challengesLost / metrics.challengesReceived;
    challengeDefenseScore = defenseRate * REPUTATION_ISOLATION_CONFIG.maxScore;
  }

  // 4. Calibration score (0-1 input, scale to 0-1000)
  const calibrationScore =
    metrics.confidenceCalibration * REPUTATION_ISOLATION_CONFIG.maxScore;

  // Weighted combination
  const rawScore =
    accuracyScore * REPUTATION_ISOLATION_CONFIG.accuracyWeight +
    consistencyScore * REPUTATION_ISOLATION_CONFIG.consistencyWeight +
    challengeDefenseScore * REPUTATION_ISOLATION_CONFIG.challengeDefenseWeight +
    calibrationScore * REPUTATION_ISOLATION_CONFIG.calibrationWeight;

  const score = Math.max(
    REPUTATION_ISOLATION_CONFIG.minScore,
    Math.min(REPUTATION_ISOLATION_CONFIG.maxScore, rawScore),
  );

  // Apply time decay
  const blocksSinceActivity = Number(currentBlock - lastActivityBlock);
  const halfLife = Number(REPUTATION_ISOLATION_CONFIG.reputationHalfLifeBlocks);
  const decayFactor = Math.pow(0.5, blocksSinceActivity / halfLife);

  // Decay toward neutral (baseScore), not toward 0
  const baseScore = REPUTATION_ISOLATION_CONFIG.baseScore;
  const decayedScore = baseScore + (score - baseScore) * decayFactor;
  const finalDecayedScore = Math.max(
    REPUTATION_ISOLATION_CONFIG.minDecayedScore,
    decayedScore,
  );

  // Score confidence based on sample size (more verifications = more confident)
  const scoreConfidence = Math.min(1, metrics.totalVerifications / 100);

  return {
    score,
    decayedScore: finalDecayedScore,
    scoreConfidence,
    components: {
      accuracyScore,
      consistencyScore,
      challengeDefenseScore,
      calibrationScore,
    },
  };
}

/**
 * Get effective reputation for a validator in a domain
 *
 * Priority:
 * 1. Domain-specific reputation (full weight)
 * 2. Stratum reputation (70% weight)
 * 3. Global reputation (50% weight)
 */
export function getEffectiveReputation(
  _validatorId: PublicKey,
  domain: string,
  isolatedRep: IsolatedReputation,
  domainTree: DomainTree,
): EffectiveReputationResult {
  // Check domain-specific reputation
  if (hasDomainReputation(isolatedRep, domain)) {
    const domainRep = isolatedRep.domainReputations.get(domain)!;
    return {
      score: domainRep.decayedScore,
      source: "domain",
      discount: 1.0,
      confidence: domainRep.scoreConfidence,
    };
  }

  // Fall back to stratum reputation
  const stratum = getDomainStratum(domain, domainTree);
  if (hasStratumReputation(isolatedRep, stratum)) {
    const stratumRep = isolatedRep.stratumReputations.get(stratum)!;
    const discountedScore =
      stratumRep.score * REPUTATION_ISOLATION_CONFIG.stratumReputationDiscount;
    const confidence = Math.min(1, stratumRep.totalVerifications / 100) * 0.7;

    return {
      score: discountedScore,
      source: "stratum",
      discount: REPUTATION_ISOLATION_CONFIG.stratumReputationDiscount,
      confidence,
    };
  }

  // Fall back to global reputation (heavily discounted)
  const globalRep = isolatedRep.globalReputation;
  const discountedScore =
    globalRep.score * REPUTATION_ISOLATION_CONFIG.globalReputationDiscount;
  const confidence = Math.min(1, globalRep.totalVerifications / 100) * 0.5;

  return {
    score: discountedScore,
    source: "global",
    discount: REPUTATION_ISOLATION_CONFIG.globalReputationDiscount,
    confidence,
  };
}

// ============================================================================
// Reputation Updates
// ============================================================================

/**
 * Update reputation after verification
 */
export function updateDomainReputation(
  isolatedRep: IsolatedReputation,
  domain: string,
  stratum: OntologicalStratum,
  wasCorrect: boolean,
  verdictConfidence: number,
  finalConfidence: number,
  currentBlock: bigint,
): IsolatedReputation {
  // Get or create domain reputation
  let domainRep = isolatedRep.domainReputations.get(domain);
  if (!domainRep) {
    domainRep = createEmptyDomainReputation(domain, stratum, currentBlock);
  }

  // Update domain metrics
  domainRep = { ...domainRep };
  domainRep.totalVerifications++;
  if (wasCorrect) {
    domainRep.correctVerifications++;
  }
  domainRep.accuracy =
    domainRep.correctVerifications / domainRep.totalVerifications;
  domainRep.lastVerificationBlock = currentBlock;

  // Update calibration (how close was verdict confidence to final?)
  const calibrationError = Math.abs(verdictConfidence - finalConfidence);
  const newCalibration = 1 - calibrationError;
  // Running average of calibration
  domainRep.confidenceCalibration =
    (domainRep.confidenceCalibration * (domainRep.totalVerifications - 1) +
      newCalibration) /
    domainRep.totalVerifications;

  // Recalculate score
  const scoreResult = calculateDomainReputationScore(
    {
      totalVerifications: domainRep.totalVerifications,
      correctVerifications: domainRep.correctVerifications,
      verdictConsistency: domainRep.verdictConsistency,
      challengesReceived: domainRep.challengesReceived,
      challengesLost: domainRep.challengesLost,
      confidenceCalibration: domainRep.confidenceCalibration,
    },
    currentBlock,
    domainRep.lastVerificationBlock,
  );

  domainRep.score = scoreResult.score;
  domainRep.decayedScore = scoreResult.decayedScore;
  domainRep.scoreConfidence = scoreResult.scoreConfidence;
  domainRep.lastDecayBlock = currentBlock;

  // Update stratum reputation
  let stratumRep = isolatedRep.stratumReputations.get(stratum);
  if (!stratumRep) {
    stratumRep = createEmptyStratumReputation(stratum, currentBlock);
  }
  stratumRep = { ...stratumRep };
  stratumRep.totalVerifications++;
  if (wasCorrect) {
    stratumRep.correctVerifications++;
  }
  stratumRep.accuracy =
    stratumRep.correctVerifications / stratumRep.totalVerifications;
  stratumRep.domainsVerified = new Set(stratumRep.domainsVerified);
  stratumRep.domainsVerified.add(domain);
  stratumRep.lastActivityBlock = currentBlock;

  // Recalculate stratum score (simplified)
  stratumRep.score =
    REPUTATION_ISOLATION_CONFIG.baseScore +
    (stratumRep.accuracy - 0.5) * REPUTATION_ISOLATION_CONFIG.maxScore;

  // Update global reputation
  const globalRep = { ...isolatedRep.globalReputation };
  globalRep.totalVerifications++;
  if (wasCorrect) {
    globalRep.correctVerifications++;
  }
  globalRep.accuracy =
    globalRep.correctVerifications / globalRep.totalVerifications;
  globalRep.score =
    REPUTATION_ISOLATION_CONFIG.baseScore +
    (globalRep.accuracy - 0.5) * REPUTATION_ISOLATION_CONFIG.maxScore;

  // Return updated isolated reputation
  const newDomainReps = new Map(isolatedRep.domainReputations);
  newDomainReps.set(domain, domainRep);

  const newStratumReps = new Map(isolatedRep.stratumReputations);
  newStratumReps.set(stratum, stratumRep);

  return {
    ...isolatedRep,
    globalReputation: globalRep,
    stratumReputations: newStratumReps,
    domainReputations: newDomainReps,
    lastUpdatedAt: currentBlock,
  };
}

/**
 * Update verdict consistency metric
 */
export function updateVerdictConsistency(
  domainRep: DomainReputation,
  previousVerdict: "confirmed" | "rejected" | "uncertain" | null,
  currentVerdict: "confirmed" | "rejected" | "uncertain",
): DomainReputation {
  if (previousVerdict === null) {
    return domainRep;
  }

  // Simple consistency: did verdict match previous?
  const consistent = previousVerdict === currentVerdict ? 1 : 0;

  // Running average
  const newConsistency =
    (domainRep.verdictConsistency * (domainRep.totalVerifications - 1) +
      consistent) /
    domainRep.totalVerifications;

  return {
    ...domainRep,
    verdictConsistency: newConsistency,
  };
}

/**
 * Record challenge result for domain reputation
 */
export function recordChallengeForReputation(
  isolatedRep: IsolatedReputation,
  domain: string,
  challengeLost: boolean,
  currentBlock: bigint,
): IsolatedReputation {
  const domainRep = isolatedRep.domainReputations.get(domain);
  if (!domainRep) {
    return isolatedRep;
  }

  const updatedDomainRep = { ...domainRep };
  updatedDomainRep.challengesReceived++;
  if (challengeLost) {
    updatedDomainRep.challengesLost++;
  }
  updatedDomainRep.challengeDefenseRate =
    1 - updatedDomainRep.challengesLost / updatedDomainRep.challengesReceived;

  // Recalculate score
  const scoreResult = calculateDomainReputationScore(
    {
      totalVerifications: updatedDomainRep.totalVerifications,
      correctVerifications: updatedDomainRep.correctVerifications,
      verdictConsistency: updatedDomainRep.verdictConsistency,
      challengesReceived: updatedDomainRep.challengesReceived,
      challengesLost: updatedDomainRep.challengesLost,
      confidenceCalibration: updatedDomainRep.confidenceCalibration,
    },
    currentBlock,
    updatedDomainRep.lastVerificationBlock,
  );

  updatedDomainRep.score = scoreResult.score;
  updatedDomainRep.decayedScore = scoreResult.decayedScore;
  updatedDomainRep.scoreConfidence = scoreResult.scoreConfidence;

  const newDomainReps = new Map(isolatedRep.domainReputations);
  newDomainReps.set(domain, updatedDomainRep);

  return {
    ...isolatedRep,
    domainReputations: newDomainReps,
    lastUpdatedAt: currentBlock,
  };
}

/**
 * Apply time decay to all reputations
 */
export function applyTimeDecay(
  isolatedRep: IsolatedReputation,
  currentBlock: bigint,
): IsolatedReputation {
  const halfLife = Number(REPUTATION_ISOLATION_CONFIG.reputationHalfLifeBlocks);
  const baseScore = REPUTATION_ISOLATION_CONFIG.baseScore;
  const minScore = REPUTATION_ISOLATION_CONFIG.minDecayedScore;

  // Decay domain reputations
  const newDomainReps = new Map<string, DomainReputation>();
  for (const [domain, rep] of isolatedRep.domainReputations) {
    const blocksSince = Number(currentBlock - rep.lastVerificationBlock);
    const decayFactor = Math.pow(0.5, blocksSince / halfLife);
    const decayedScore = Math.max(
      minScore,
      baseScore + (rep.score - baseScore) * decayFactor,
    );

    newDomainReps.set(domain, {
      ...rep,
      decayedScore,
      lastDecayBlock: currentBlock,
    });
  }

  // Decay stratum reputations
  const newStratumReps = new Map<OntologicalStratum, StratumReputation>();
  for (const [stratum, rep] of isolatedRep.stratumReputations) {
    const blocksSince = Number(currentBlock - rep.lastActivityBlock);
    const decayFactor = Math.pow(0.5, blocksSince / halfLife);
    const decayedScore = Math.max(
      minScore,
      baseScore + (rep.score - baseScore) * decayFactor,
    );

    newStratumReps.set(stratum, {
      ...rep,
      score: decayedScore,
      decayApplied: true,
      lastDecayBlock: currentBlock,
    });
  }

  return {
    ...isolatedRep,
    domainReputations: newDomainReps,
    stratumReputations: newStratumReps,
    lastUpdatedAt: currentBlock,
  };
}

// ============================================================================
// Reputation Queries
// ============================================================================

/**
 * Get reputation summary for display
 */
export function getReputationSummary(isolatedRep: IsolatedReputation): {
  globalScore: number;
  globalAccuracy: number;
  domainCount: number;
  stratumCount: number;
  totalVerifications: number;
} {
  return {
    globalScore: isolatedRep.globalReputation.score,
    globalAccuracy: isolatedRep.globalReputation.accuracy,
    domainCount: isolatedRep.domainReputations.size,
    stratumCount: isolatedRep.stratumReputations.size,
    totalVerifications: isolatedRep.globalReputation.totalVerifications,
  };
}

/**
 * Get score description
 */
export function getScoreDescription(score: number): string {
  if (score >= 900) return "Excellent";
  if (score >= 700) return "Good";
  if (score >= 500) return "Neutral";
  if (score >= 300) return "Poor";
  return "Very Poor";
}

/**
 * Normalize score to 0-1 range
 */
export function normalizeScore(score: number): number {
  return score / REPUTATION_ISOLATION_CONFIG.maxScore;
}
