/**
 * Cross-Stratum Validation
 *
 * Validates that verification rounds include validators from required
 * strata and calculates cross-stratum confidence adjustments.
 *
 * Supports dependency injection via OntologyProvider for testability.
 */

import type { OntologicalStratum } from "../../core/knowledge/ontology/types.js";
import {
  type OntologyProvider,
  defaultOntologyProvider,
} from "../../core/knowledge/ontology/stratum_registry.js";
import type { ValidatorReveal } from "../pot/types.js";
import {
  type CrossStratumValidationResult,
  type CrossStratumConfidenceResult,
  type StratumSeparatedReveals,
  CROSS_STRATUM_CONFIG,
  createEmptyValidationResult,
} from "./types.js";

// ============================================================================
// Validation Functions
// ============================================================================

/**
 * Validate that a verification round meets cross-stratum requirements
 */
export function validateCrossStratumCoverage(
  claimStratum: OntologicalStratum,
  validatorStrata: Map<string, OntologicalStratum>,
): CrossStratumValidationResult {
  const requirement = CROSS_STRATUM_CONFIG.requirements[claimStratum];
  if (!requirement) {
    return {
      ...createEmptyValidationResult(),
      meetsRequirements: true, // No requirement = passes
    };
  }

  // Count validators per stratum
  const stratumCoverage = new Map<OntologicalStratum, number>();
  for (const stratum of validatorStrata.values()) {
    stratumCoverage.set(stratum, (stratumCoverage.get(stratum) || 0) + 1);
  }

  // Check if requirements are met
  const missingStrata: OntologicalStratum[] = [];
  let meetsRequirements = true;

  for (const requiredStratum of requirement.requiredStrata) {
    const count = stratumCoverage.get(requiredStratum) || 0;
    const minRequired =
      requirement.minValidatorsPerStratum[requiredStratum] || 1;

    if (count < minRequired) {
      missingStrata.push(requiredStratum);
      meetsRequirements = false;
    }
  }

  // Check for partial coverage (some but not all requirements met)
  const partialCoverage =
    !meetsRequirements &&
    missingStrata.length < requirement.requiredStrata.length;

  return {
    meetsRequirements,
    stratumCoverage,
    missingStrata,
    lowerStratumAgreement: 0, // Calculated separately with reveals
    sameStratumAgreement: 0,
    crossStratumConfidenceAdjustment: meetsRequirements
      ? 1.0
      : partialCoverage && CROSS_STRATUM_CONFIG.allowPartialCoverage
        ? 1.0 - CROSS_STRATUM_CONFIG.partialCoverageConfidencePenalty
        : 0.5, // Severe penalty for no coverage
    partialCoverage,
  };
}

/**
 * Separate reveals by stratum relationship to claim
 *
 * @param reveals - Validator reveals
 * @param validatorStrata - Map of validator IDs to their qualified stratum
 * @param claimStratum - The stratum of the claim being verified
 * @param ontology - OntologyProvider for stratum lookups (defaults to global)
 */
export function separateRevealsByStratum(
  reveals: Map<string, ValidatorReveal>,
  validatorStrata: Map<string, OntologicalStratum>,
  claimStratum: OntologicalStratum,
  ontology: OntologyProvider = defaultOntologyProvider,
): StratumSeparatedReveals {
  const claimLevel = ontology.getLevel(claimStratum);

  const lowerStratumReveals: ValidatorReveal[] = [];
  const sameStratumReveals: ValidatorReveal[] = [];
  const higherStratumReveals: ValidatorReveal[] = [];

  for (const [validatorId, reveal] of reveals) {
    const validatorStratum = validatorStrata.get(validatorId);
    if (!validatorStratum) continue;

    const validatorLevel = ontology.getLevel(validatorStratum);

    if (validatorLevel < claimLevel) {
      lowerStratumReveals.push(reveal);
    } else if (validatorLevel === claimLevel) {
      sameStratumReveals.push(reveal);
    } else {
      higherStratumReveals.push(reveal);
    }
  }

  return {
    lowerStratumReveals,
    sameStratumReveals,
    higherStratumReveals,
  };
}

/**
 * Calculate agreement level among reveals
 */
function calculateAgreement(reveals: ValidatorReveal[]): number {
  if (reveals.length === 0) return 0;

  const majority = getMajorityVerdict(reveals);
  const matching = reveals.filter((r) => r.verdict === majority).length;

  return matching / reveals.length;
}

/**
 * Get majority verdict from reveals
 */
function getMajorityVerdict(
  reveals: ValidatorReveal[],
): "confirmed" | "rejected" | "uncertain" {
  if (reveals.length === 0) return "uncertain";

  const counts = { confirmed: 0, rejected: 0, uncertain: 0 };
  for (const reveal of reveals) {
    counts[reveal.verdict]++;
  }

  if (
    counts.confirmed >= counts.rejected &&
    counts.confirmed >= counts.uncertain
  ) {
    return "confirmed";
  }
  if (counts.rejected >= counts.uncertain) {
    return "rejected";
  }
  return "uncertain";
}

/**
 * Calculate cross-stratum confidence adjustment
 *
 * Key logic:
 * - If lower stratum agrees with same stratum: apply bonus
 * - If lower stratum disagrees: apply veto penalty
 * - Lower stratum validators are closer to foundational truth
 *
 * @param reveals - Validator reveals
 * @param validatorStrata - Map of validator IDs to their qualified stratum
 * @param claimStratum - The stratum of the claim being verified
 * @param validators - Validator stakes and reputations
 * @param ontology - OntologyProvider for stratum lookups (defaults to global)
 */
export function calculateCrossStratumConfidence(
  reveals: Map<string, ValidatorReveal>,
  validatorStrata: Map<string, OntologicalStratum>,
  claimStratum: OntologicalStratum,
  validators: Map<string, { stake: bigint; reputation: number }>,
  ontology: OntologyProvider = defaultOntologyProvider,
): CrossStratumConfidenceResult {
  // Separate reveals by stratum
  const { lowerStratumReveals, sameStratumReveals } = separateRevealsByStratum(
    reveals,
    validatorStrata,
    claimStratum,
    ontology,
  );

  // Calculate agreement in each group
  const lowerStratumAgreement = calculateAgreement(lowerStratumReveals);
  const sameStratumAgreement = calculateAgreement(sameStratumReveals);

  // Get majority verdicts
  const lowerVerdict =
    lowerStratumReveals.length > 0
      ? getMajorityVerdict(lowerStratumReveals)
      : null;
  const sameVerdict =
    sameStratumReveals.length > 0
      ? getMajorityVerdict(sameStratumReveals)
      : null;

  // Check for cross-stratum disagreement
  const requirement = CROSS_STRATUM_CONFIG.requirements[claimStratum];
  const lowerBonus = requirement?.lowerStratumWeightBonus || 1.0;

  let crossStratumAdjustment = 1.0;
  let hasDisagreement = false;

  if (lowerVerdict !== null && sameVerdict !== null) {
    if (lowerVerdict !== sameVerdict) {
      // Cross-stratum disagreement - apply veto weight
      crossStratumAdjustment =
        1.0 - CROSS_STRATUM_CONFIG.lowerStratumVetoWeight;
      hasDisagreement = true;
    } else {
      // Cross-stratum agreement - apply bonus
      crossStratumAdjustment = lowerBonus;
    }
  }

  // Calculate base weighted confidence
  const baseConfidence = calculateWeightedConfidence(
    reveals,
    validators,
    validatorStrata,
    claimStratum,
    ontology,
  );

  return {
    confidence: baseConfidence * crossStratumAdjustment,
    lowerStratumAgreement,
    sameStratumAgreement,
    crossStratumAdjustment,
    lowerStratumVerdict: lowerVerdict,
    sameStratumVerdict: sameVerdict,
    hasDisagreement,
  };
}

/**
 * Calculate weighted confidence with stratum bonuses
 */
function calculateWeightedConfidence(
  reveals: Map<string, ValidatorReveal>,
  validators: Map<string, { stake: bigint; reputation: number }>,
  validatorStrata: Map<string, OntologicalStratum>,
  claimStratum: OntologicalStratum,
  ontology: OntologyProvider = defaultOntologyProvider,
): number {
  const claimLevel = ontology.getLevel(claimStratum);
  const requirement = CROSS_STRATUM_CONFIG.requirements[claimStratum];
  const lowerBonus = requirement?.lowerStratumWeightBonus || 1.0;

  // ZD-S01 FIX: Use BigInt-scaled weights to prevent Number() overflow on large stakes.
  // Scale factor: 1M BPS for fractional multipliers applied to BigInt stakes.
  const SCALE = 1_000_000n;
  let weightedSumScaled = 0n;
  let totalWeightScaled = 0n;

  for (const [validatorId, reveal] of reveals) {
    const validator = validators.get(validatorId);
    if (!validator) continue;

    const validatorStratum = validatorStrata.get(validatorId);
    if (!validatorStratum) continue;

    const validatorLevel = ontology.getLevel(validatorStratum);

    // Base weight from stake (BigInt)
    let weightScaled = validator.stake * SCALE;

    // Apply stratum bonus for lower stratum validators
    if (validatorLevel < claimLevel) {
      const bonusBps = BigInt(Math.round(lowerBonus * 1_000_000));
      weightScaled = (weightScaled * bonusBps) / SCALE;
    }

    // Weight by reputation: 0.5 + reputation * 0.5
    const repBps = BigInt(
      Math.round((0.5 + validator.reputation * 0.5) * 1_000_000),
    );
    weightScaled = (weightScaled * repBps) / SCALE;

    // Confidence is a float 0.0-1.0, apply to BigInt weight
    const confBps = BigInt(Math.round(reveal.confidence * 1_000_000));
    weightedSumScaled += (weightScaled * confBps) / SCALE;
    totalWeightScaled += weightScaled;
  }

  return totalWeightScaled > 0n
    ? Number((weightedSumScaled * SCALE) / totalWeightScaled) / 1_000_000
    : 0;
}

/**
 * Get required validators for a stratum
 */
export function getRequiredValidatorsForStratum(
  claimStratum: OntologicalStratum,
): { stratum: OntologicalStratum; count: number }[] {
  const requirement = CROSS_STRATUM_CONFIG.requirements[claimStratum];
  if (!requirement) return [];

  return requirement.requiredStrata.map((stratum) => ({
    stratum,
    count: requirement.minValidatorsPerStratum[stratum] || 1,
  }));
}

/**
 * Check if a validator's stratum qualifies for a claim stratum
 */
export function isValidatorStratumEligible(
  validatorStratum: OntologicalStratum,
  claimStratum: OntologicalStratum,
): boolean {
  const requirement = CROSS_STRATUM_CONFIG.requirements[claimStratum];
  if (!requirement) return true;

  return requirement.requiredStrata.includes(validatorStratum);
}

/**
 * Calculate stratum weight multiplier for a validator
 *
 * @param validatorStratum - The validator's qualified stratum
 * @param claimStratum - The claim's stratum
 * @param ontology - OntologyProvider for stratum lookups (defaults to global)
 */
export function getStratumWeightMultiplier(
  validatorStratum: OntologicalStratum,
  claimStratum: OntologicalStratum,
  ontology: OntologyProvider = defaultOntologyProvider,
): number {
  const validatorLevel = ontology.getLevel(validatorStratum);
  const claimLevel = ontology.getLevel(claimStratum);

  if (validatorLevel < claimLevel) {
    // Lower stratum gets bonus
    const requirement = CROSS_STRATUM_CONFIG.requirements[claimStratum];
    return requirement?.lowerStratumWeightBonus || 1.0;
  }

  return 1.0;
}
