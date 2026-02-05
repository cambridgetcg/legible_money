/**
 * Reputation Isolation System - Types
 *
 * Prevents reputation from one domain being used to gain influence
 * in another domain. Domain-specific reputation creates accountability
 * within each knowledge area.
 *
 * Configuration:
 * - 0.5 (50%) global reputation discount
 * - 0.7 (70%) stratum reputation discount
 * - 19 verifications for domain reputation
 * - 77777 blocks reputation half-life (~5.4 hours)
 * - 0 cross-domain transfer rate (NO transfer)
 */

import type { PublicKey } from "../../core/crypto/keys.js";
import type { OntologicalStratum } from "../../core/knowledge/ontology/types.js";

// ============================================================================
// Configuration
// ============================================================================

/**
 * Reputation isolation configuration with symbolic constants
 */
export const REPUTATION_ISOLATION_CONFIG = {
  // Discounts when using non-domain reputation
  globalReputationDiscount: 0.5, // 50% when only global
  stratumReputationDiscount: 0.7, // 70% when only stratum
  newDomainDiscount: 0.3, // 30% for first 19 verifications

  // Thresholds
  minVerificationsForDomainReputation: 19,
  minAccuracyForFullWeight: 0.77, // 77%

  // Score calculation
  baseScore: 500, // Start at neutral
  maxScore: 1000,
  minScore: 0,

  // Score components (must sum to 1.0)
  accuracyWeight: 0.5, // 50% of score
  consistencyWeight: 0.25, // 25% of score
  challengeDefenseWeight: 0.15, // 15% of score
  calibrationWeight: 0.1, // 10% of score

  // Decay
  reputationHalfLifeBlocks: 77777n, // ~5.4 hours
  minDecayedScore: 300, // Minimum after decay

  // Transfer
  crossDomainTransferRate: 0, // NO transfer between domains
  crossStratumTransferRate: 0.1, // 10% of stratum rep to new domain

  // Laplace smoothing for new validators
  laplacePrior: 1,
  laplaceTotal: 2,
} as const;

// ============================================================================
// Core Types
// ============================================================================

/**
 * Source of reputation being used
 */
export type ReputationSource = "domain" | "stratum" | "global";

/**
 * Global reputation (baseline, heavily discounted)
 */
export interface GlobalReputation {
  score: number; // 0-1000
  totalVerifications: number;
  correctVerifications: number;
  accuracy: number;

  // Cannot be used directly - must be discounted
  usageDiscount: number; // 0.5 (50%)
}

/**
 * Stratum-level reputation
 */
export interface StratumReputation {
  stratum: OntologicalStratum;
  score: number; // 0-1000

  totalVerifications: number;
  correctVerifications: number;
  accuracy: number;

  // Domains verified in this stratum
  domainsVerified: Set<string>;

  // Can be used in this stratum with discount
  usageDiscount: number; // 0.7 (70%)

  // Temporal
  firstVerificationBlock: bigint;
  lastActivityBlock: bigint;

  // Decay
  decayApplied: boolean;
  lastDecayBlock: bigint;
}

/**
 * Domain-specific reputation (primary measure)
 */
export interface DomainReputation {
  domain: string;
  stratum: OntologicalStratum;
  score: number; // 0-1000

  // Performance metrics
  totalVerifications: number;
  correctVerifications: number;
  accuracy: number;

  // Consistency metrics
  verdictConsistency: number; // 0-1, how consistent verdicts are
  confidenceCalibration: number; // 0-1, how calibrated confidence is

  // Challenge metrics
  challengesReceived: number;
  challengesLost: number;
  challengeDefenseRate: number;

  // Confidence in score (based on sample size)
  scoreConfidence: number; // 0-1, increases with verifications

  // No discount - this is the primary measure
  usageDiscount: number; // 1.0 (100%)

  // Temporal
  firstVerificationBlock: bigint;
  lastVerificationBlock: bigint;

  // Decay
  decayedScore: number; // Score after time decay
  lastDecayBlock: bigint;
}

/**
 * Isolated reputation system with domain-specific scores
 */
export interface IsolatedReputation {
  validatorId: PublicKey;

  // Global baseline (cannot use directly in domain)
  globalReputation: GlobalReputation;

  // Stratum-level reputation
  stratumReputations: Map<OntologicalStratum, StratumReputation>;

  // Domain-specific reputation (the primary measure)
  domainReputations: Map<string, DomainReputation>;

  // Metadata
  createdAt: bigint;
  lastUpdatedAt: bigint;
}

/**
 * Components of a reputation score
 */
export interface ReputationComponents {
  accuracyScore: number;
  consistencyScore: number;
  challengeDefenseScore: number;
  calibrationScore: number;
}

/**
 * Result of reputation score calculation
 */
export interface ReputationScoreResult {
  score: number;
  decayedScore: number;
  scoreConfidence: number;
  components: ReputationComponents;
}

/**
 * Effective reputation lookup result
 */
export interface EffectiveReputationResult {
  score: number;
  source: ReputationSource;
  discount: number;
  confidence: number;
}

/**
 * Reputation update input
 */
export interface ReputationUpdateInput {
  validatorId: PublicKey;
  domain: string;
  stratum: OntologicalStratum;
  wasCorrect: boolean;
  verdictConfidence: number;
  finalConfidence: number;
  currentBlock: bigint;
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Create empty global reputation
 */
export function createEmptyGlobalReputation(): GlobalReputation {
  return {
    score: REPUTATION_ISOLATION_CONFIG.baseScore,
    totalVerifications: 0,
    correctVerifications: 0,
    accuracy: 0,
    usageDiscount: REPUTATION_ISOLATION_CONFIG.globalReputationDiscount,
  };
}

/**
 * Create empty stratum reputation
 */
export function createEmptyStratumReputation(
  stratum: OntologicalStratum,
  currentBlock: bigint,
): StratumReputation {
  return {
    stratum,
    score: REPUTATION_ISOLATION_CONFIG.baseScore,
    totalVerifications: 0,
    correctVerifications: 0,
    accuracy: 0,
    domainsVerified: new Set(),
    usageDiscount: REPUTATION_ISOLATION_CONFIG.stratumReputationDiscount,
    firstVerificationBlock: currentBlock,
    lastActivityBlock: currentBlock,
    decayApplied: false,
    lastDecayBlock: currentBlock,
  };
}

/**
 * Create empty domain reputation
 */
export function createEmptyDomainReputation(
  domain: string,
  stratum: OntologicalStratum,
  currentBlock: bigint,
): DomainReputation {
  return {
    domain,
    stratum,
    score: REPUTATION_ISOLATION_CONFIG.baseScore,
    totalVerifications: 0,
    correctVerifications: 0,
    accuracy: 0,
    verdictConsistency: 0,
    confidenceCalibration: 0,
    challengesReceived: 0,
    challengesLost: 0,
    challengeDefenseRate: 1.0, // Start assuming no challenges lost
    scoreConfidence: 0,
    usageDiscount: 1.0, // Full weight for domain-specific
    firstVerificationBlock: currentBlock,
    lastVerificationBlock: currentBlock,
    decayedScore: REPUTATION_ISOLATION_CONFIG.baseScore,
    lastDecayBlock: currentBlock,
  };
}

/**
 * Create empty isolated reputation for a validator
 */
export function createEmptyIsolatedReputation(
  validatorId: PublicKey,
  currentBlock: bigint,
): IsolatedReputation {
  return {
    validatorId,
    globalReputation: createEmptyGlobalReputation(),
    stratumReputations: new Map(),
    domainReputations: new Map(),
    createdAt: currentBlock,
    lastUpdatedAt: currentBlock,
  };
}

/**
 * Check if validator has domain-level reputation
 */
export function hasDomainReputation(
  isolatedRep: IsolatedReputation,
  domain: string,
): boolean {
  const domainRep = isolatedRep.domainReputations.get(domain);
  return (
    domainRep !== undefined &&
    domainRep.totalVerifications >=
      REPUTATION_ISOLATION_CONFIG.minVerificationsForDomainReputation
  );
}

/**
 * Check if validator has stratum-level reputation
 */
export function hasStratumReputation(
  isolatedRep: IsolatedReputation,
  stratum: OntologicalStratum,
): boolean {
  const stratumRep = isolatedRep.stratumReputations.get(stratum);
  return (
    stratumRep !== undefined &&
    stratumRep.totalVerifications >=
      REPUTATION_ISOLATION_CONFIG.minVerificationsForDomainReputation
  );
}

/**
 * Get reputation source for a domain
 */
export function getReputationSource(
  isolatedRep: IsolatedReputation,
  domain: string,
  stratum: OntologicalStratum,
): ReputationSource {
  if (hasDomainReputation(isolatedRep, domain)) {
    return "domain";
  }
  if (hasStratumReputation(isolatedRep, stratum)) {
    return "stratum";
  }
  return "global";
}
