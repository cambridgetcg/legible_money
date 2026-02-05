/**
 * Cross-Stratum Validation System
 *
 * Requires claims in higher ontological strata to include validators
 * qualified in lower strata, providing foundational cross-checking.
 *
 * Key features:
 * - Biology claims need chemical validators
 * - Chemistry claims need physical validators
 * - Lower stratum disagreement applies 30% confidence penalty
 * - Lower stratum agreement applies 10-20% confidence bonus
 *
 * @module consensus/cross_stratum
 */

// Types
export type {
  CrossStratumRequirement,
  CrossStratumValidationResult,
  StratumSeparatedReveals,
  ValidatorStratumInfo,
  CrossStratumConfidenceInput,
  CrossStratumConfidenceResult,
} from "./types.js";

export {
  CROSS_STRATUM_CONFIG,
  getStratumRequirement,
  isLowerStratum,
  getStratumLevel,
  createEmptyValidationResult,
} from "./types.js";

// Validation
export {
  validateCrossStratumCoverage,
  separateRevealsByStratum,
  calculateCrossStratumConfidence,
  getRequiredValidatorsForStratum,
  isValidatorStratumEligible,
  getStratumWeightMultiplier,
} from "./validation.js";
