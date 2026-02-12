/**
 * Domain Qualification System - Types
 *
 * Prevents unqualified validators from dominating specialized domains
 * by requiring demonstrated competence or stake commitment.
 *
 * Configuration:
 * - 22/222/2222 LGM stakes (tiered by domain depth)
 * - 19 verifications for track record
 * - 77% accuracy threshold
 * - 7777 blocks stake lock period (~5.4 hours)
 */

import type { PublicKey } from "../../core/crypto/keys.js";
import type { OntologicalStratum } from "../../core/knowledge/ontology/types.js";

// ============================================================================
// Configuration
// ============================================================================

/**
 * Domain qualification configuration with symbolic constants
 */
export const DOMAIN_QUALIFICATION_CONFIG = {
  // Stake commitment pathway (22/222/2222 pattern)
  stakeRequirements: {
    leaf: 22n * 10n ** 15n, // 0.022 LGM for leaf domains (depth > 2)
    mid: 222n * 10n ** 15n, // 0.222 LGM for mid-level (depth 2)
    top: 2222n * 10n ** 15n, // 2.222 LGM for top-level (depth 1)
  },
  stakeLockPeriod: 7777n, // ~5.4 hours at 2521ms blocks

  // Track record pathway
  minVerificationsForQualification: 19,
  minAccuracyForQualification: 0.77, // 77%
  trackRecordDomain: "parent_or_adjacent" as const,

  // Cross-reference pathway
  minEndorsersForQualification: 3,
  endorserStakeRequirement: 22n * 10n ** 15n, // Each endorser stakes 0.022 LGM
  endorsementDecayBlocks: 77777n, // Endorsements expire (~5.4 hours)

  // Stratum inheritance
  stratumInheritanceDiscount: 0.7, // 70% weight when using inheritance
  requireLowerStratumForHigher: true,

  // Maintenance
  inactivityExpirationBlocks: 777777n, // ~27 days
  probationaryPeriodBlocks: 777n, // ~32 minutes
  probationaryWeight: 0.5, // 50% weight during probation

  // Suspension/Revocation
  accuracyThresholdForSuspension: 0.5, // Below 50% accuracy
  consecutiveIncorrectForSuspension: 5,
  challengesLostForRevocation: 3,

  // Domain depth
  maxQualificationDepth: 7, // Max domain tree depth for qualification
} as const;

/**
 * LGM token unit (10^18 wei equivalent)
 */
export const LGM = 10n ** 18n;

// ============================================================================
// Core Types
// ============================================================================

/**
 * How a validator earned qualification in a domain
 */
export type QualificationPathway =
  | "stake_commitment" // Locked stake in domain
  | "track_record" // Proven verification history
  | "cross_reference" // Endorsed by qualified validators
  | "stratum_inheritance" // Qualified in lower stratum
  | "genesis"; // Protocol-granted at genesis

/**
 * Qualification status
 */
export type QualificationStatus =
  | "active" // Currently qualified
  | "probationary" // New qualification, limited weight
  | "suspended" // Temporarily suspended (poor performance)
  | "revoked" // Permanently revoked (misconduct)
  | "expired"; // Lapsed due to inactivity

/**
 * Performance metrics for qualification
 */
export interface QualificationMetrics {
  totalVerifications: number;
  correctVerifications: number;
  accuracy: number; // correctVerifications / totalVerifications

  // Temporal metrics
  lastVerificationBlock: bigint;
  consecutiveCorrect: number;
  consecutiveIncorrect: number;

  // Challenge metrics
  challengesReceived: number;
  challengesLost: number;

  // Weight factors
  effectiveWeight: number; // 0-1, applied to votes
}

/**
 * A validator's qualification in a specific domain
 */
export interface DomainQualification {
  // Identity
  validatorId: PublicKey;
  domain: string;
  stratum: OntologicalStratum;

  // How earned
  pathway: QualificationPathway;
  earnedAt: bigint; // Block number
  expiresAt: bigint | null; // null = no expiration

  // Current status
  status: QualificationStatus;

  // Performance in this domain
  metrics: QualificationMetrics;

  // Stake (if stake_commitment pathway)
  lockedStake?: bigint;
  stakeUnlocksAt?: bigint;

  // Cross-reference (if cross_reference pathway)
  endorsers?: PublicKey[];
  endorserStakes?: Map<string, bigint>;

  // Inheritance (if stratum_inheritance pathway)
  inheritedFrom?: string; // Parent domain/stratum
  inheritanceDiscount: number; // Weight reduction (0.7 default)
}

/**
 * Input for requesting qualification
 */
export interface QualificationRequest {
  validatorId: PublicKey;
  domain: string;
  pathway: QualificationPathway;
  stake?: bigint; // For stake_commitment
  endorsers?: PublicKey[]; // For cross_reference
  endorserStakes?: Map<string, bigint>; // Stake each endorser is locking
  inheritFrom?: string; // For stratum_inheritance
}

/**
 * Result of qualification request
 */
export interface QualificationResult {
  granted: boolean;
  qualification?: DomainQualification;
  errors: string[];
  requiredStake?: bigint;
  requiredVerifications?: number;
  requiredEndorsers?: number;
}

/**
 * Validator qualification state (stored per validator)
 */
export interface ValidatorQualificationState {
  validatorId: PublicKey;

  // Map: domain -> qualification
  qualifications: Map<string, DomainQualification>;

  // Total stake locked across all domain qualifications
  totalLockedStake: bigint;

  // Qualification count by pathway
  pathwayCounts: {
    stake_commitment: number;
    track_record: number;
    cross_reference: number;
    stratum_inheritance: number;
    genesis: number;
  };

  // Last updated block
  lastUpdatedAt: bigint;
}

/**
 * Endorsement for another validator's qualification
 */
export interface QualificationEndorsement {
  endorserId: PublicKey;
  targetValidatorId: PublicKey;
  domain: string;
  stake: bigint;
  endorsedAt: bigint;
  expiresAt: bigint;
  comment?: string;
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Create empty qualification metrics
 */
export function createEmptyMetrics(): QualificationMetrics {
  return {
    totalVerifications: 0,
    correctVerifications: 0,
    accuracy: 0,
    lastVerificationBlock: 0n,
    consecutiveCorrect: 0,
    consecutiveIncorrect: 0,
    challengesReceived: 0,
    challengesLost: 0,
    effectiveWeight: 1.0,
  };
}

/**
 * Create empty validator qualification state
 */
export function createEmptyQualificationState(
  validatorId: PublicKey,
  currentBlock: bigint,
): ValidatorQualificationState {
  return {
    validatorId,
    qualifications: new Map(),
    totalLockedStake: 0n,
    pathwayCounts: {
      stake_commitment: 0,
      track_record: 0,
      cross_reference: 0,
      stratum_inheritance: 0,
      genesis: 0,
    },
    lastUpdatedAt: currentBlock,
  };
}

/**
 * Calculate required stake based on domain depth
 */
export function calculateQualificationStake(depth: number): bigint {
  if (depth <= 1) return DOMAIN_QUALIFICATION_CONFIG.stakeRequirements.top;
  if (depth === 2) return DOMAIN_QUALIFICATION_CONFIG.stakeRequirements.mid;
  return DOMAIN_QUALIFICATION_CONFIG.stakeRequirements.leaf;
}

/**
 * Check if qualification is still valid (not expired)
 */
export function isQualificationValid(
  qualification: DomainQualification,
  currentBlock: bigint,
): boolean {
  if (
    qualification.status !== "active" &&
    qualification.status !== "probationary"
  ) {
    return false;
  }

  if (
    qualification.expiresAt !== null &&
    currentBlock > qualification.expiresAt
  ) {
    return false;
  }

  return true;
}

/**
 * Calculate effective weight for a qualification
 */
export function calculateQualificationWeight(
  qualification: DomainQualification,
): number {
  let weight = 1.0;

  // Apply inheritance discount
  weight *= qualification.inheritanceDiscount;

  // Apply probationary discount
  if (qualification.status === "probationary") {
    weight *= DOMAIN_QUALIFICATION_CONFIG.probationaryWeight;
  }

  // Apply accuracy bonus (1.0 to 1.2 for accuracy above 77%)
  const accuracyBonus =
    1.0 + Math.max(0, qualification.metrics.accuracy - 0.77) * 0.5;
  weight *= Math.min(1.2, accuracyBonus);

  return weight;
}
