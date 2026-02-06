/**
 * Cross-Stratum Validation System - Types
 *
 * Requires claims in higher ontological strata to include validators
 * qualified in lower strata, providing foundational cross-checking.
 *
 * Rationale: Physics knowledge is relevant to chemistry claims.
 * Biology claims should include validators who understand chemistry.
 *
 * Symbolic Constants:
 * - 2 lower-stratum validators minimum (pair for verification)
 * - 0.7 (70%) cross-stratum agreement threshold
 * - 0.3 (30%) lower stratum veto weight
 */

import {
  type OntologicalStratum,
  STRATUM_HIERARCHY,
} from "../../core/knowledge/ontology/types.js";
import type { ValidatorReveal } from "../pot/types.js";

// ============================================================================
// Configuration
// ============================================================================

/**
 * Cross-stratum validation requirement for a claim
 */
export interface CrossStratumRequirement {
  claimStratum: OntologicalStratum;
  requiredStrata: OntologicalStratum[];
  minValidatorsPerStratum: Partial<Record<OntologicalStratum, number>>;
  lowerStratumWeightBonus: number;
}

/**
 * Configuration for cross-stratum validation
 */
export const CROSS_STRATUM_CONFIG = {
  // Requirements by claim stratum
  requirements: {
    technological: {
      claimStratum: "technological" as OntologicalStratum,
      requiredStrata: ["technological"] as OntologicalStratum[],
      minValidatorsPerStratum: { technological: 3 },
      lowerStratumWeightBonus: 1.0,
    },
    social: {
      claimStratum: "social" as OntologicalStratum,
      requiredStrata: ["social", "cognitive"] as OntologicalStratum[],
      minValidatorsPerStratum: { social: 2, cognitive: 1 },
      lowerStratumWeightBonus: 1.1,
    },
    cognitive: {
      claimStratum: "cognitive" as OntologicalStratum,
      requiredStrata: ["cognitive", "biological"] as OntologicalStratum[],
      minValidatorsPerStratum: { cognitive: 2, biological: 1 },
      lowerStratumWeightBonus: 1.1,
    },
    biological: {
      claimStratum: "biological" as OntologicalStratum,
      requiredStrata: ["biological", "chemical"] as OntologicalStratum[],
      minValidatorsPerStratum: { biological: 2, chemical: 2 },
      lowerStratumWeightBonus: 1.15,
    },
    chemical: {
      claimStratum: "chemical" as OntologicalStratum,
      requiredStrata: ["chemical", "physical"] as OntologicalStratum[],
      minValidatorsPerStratum: { chemical: 2, physical: 2 },
      lowerStratumWeightBonus: 1.15,
    },
    physical: {
      claimStratum: "physical" as OntologicalStratum,
      requiredStrata: ["physical", "fundamental"] as OntologicalStratum[],
      minValidatorsPerStratum: { physical: 2, fundamental: 1 },
      lowerStratumWeightBonus: 1.2,
    },
    fundamental: {
      claimStratum: "fundamental" as OntologicalStratum,
      requiredStrata: ["fundamental"] as OntologicalStratum[],
      minValidatorsPerStratum: { fundamental: 3 },
      lowerStratumWeightBonus: 1.0,
    },
  } as Record<OntologicalStratum, CrossStratumRequirement>,

  // Agreement thresholds
  crossStratumAgreementThreshold: 0.7, // 70% cross-stratum agreement

  // Disagreement handling
  lowerStratumVetoWeight: 0.3, // Lower stratum disagreement penalty

  // Fallback when requirements not met
  allowPartialCoverage: true,
  partialCoverageConfidencePenalty: 0.2, // 20% confidence reduction

  // Maximum stratum distance for inheritance
  maxStratumDistance: 2,
} as const;

// ============================================================================
// Core Types
// ============================================================================

/**
 * Cross-stratum validation result
 */
export interface CrossStratumValidationResult {
  meetsRequirements: boolean;
  stratumCoverage: Map<OntologicalStratum, number>; // stratum -> validator count
  missingStrata: OntologicalStratum[];
  lowerStratumAgreement: number;
  sameStratumAgreement: number;
  crossStratumConfidenceAdjustment: number;
  partialCoverage: boolean;
}

/**
 * Stratum-separated reveals for analysis
 */
export interface StratumSeparatedReveals {
  lowerStratumReveals: ValidatorReveal[];
  sameStratumReveals: ValidatorReveal[];
  higherStratumReveals: ValidatorReveal[];
}

/**
 * Validator with stratum qualification info
 */
export interface ValidatorStratumInfo {
  validatorId: string;
  qualifiedStratum: OntologicalStratum;
  qualificationWeight: number;
}

/**
 * Cross-stratum confidence calculation input
 */
export interface CrossStratumConfidenceInput {
  reveals: Map<string, ValidatorReveal>;
  validatorStrata: Map<string, OntologicalStratum>;
  claimStratum: OntologicalStratum;
  validators: Map<string, { stake: bigint; reputation: number }>;
}

/**
 * Cross-stratum confidence calculation result
 */
export interface CrossStratumConfidenceResult {
  confidence: number;
  lowerStratumAgreement: number;
  sameStratumAgreement: number;
  crossStratumAdjustment: number;
  lowerStratumVerdict: "confirmed" | "rejected" | "uncertain" | null;
  sameStratumVerdict: "confirmed" | "rejected" | "uncertain" | null;
  hasDisagreement: boolean;
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Get requirement for a stratum
 */
export function getStratumRequirement(
  stratum: OntologicalStratum,
): CrossStratumRequirement {
  return CROSS_STRATUM_CONFIG.requirements[stratum];
}

/**
 * Check if stratum is lower than another.
 * Delegates to the single source of truth: STRATUM_HIERARCHY.
 */
export function isLowerStratum(
  candidateStratum: OntologicalStratum,
  referenceStratum: OntologicalStratum,
): boolean {
  return (
    STRATUM_HIERARCHY[candidateStratum] < STRATUM_HIERARCHY[referenceStratum]
  );
}

/**
 * Get stratum level (0-6).
 * Delegates to the single source of truth: STRATUM_HIERARCHY.
 */
export function getStratumLevel(stratum: OntologicalStratum): number {
  return STRATUM_HIERARCHY[stratum];
}

/**
 * Create empty validation result
 */
export function createEmptyValidationResult(): CrossStratumValidationResult {
  return {
    meetsRequirements: false,
    stratumCoverage: new Map(),
    missingStrata: [],
    lowerStratumAgreement: 0,
    sameStratumAgreement: 0,
    crossStratumConfidenceAdjustment: 1.0,
    partialCoverage: false,
  };
}
