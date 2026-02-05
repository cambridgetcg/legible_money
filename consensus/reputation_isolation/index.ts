/**
 * Reputation Isolation System
 *
 * Prevents reputation from one domain being used to gain influence
 * in another domain. Domain-specific reputation creates accountability.
 *
 * Key features:
 * - Domain-specific reputation (100% weight)
 * - Stratum-level fallback (70% weight)
 * - Global fallback (50% weight)
 * - 19 verifications for domain reputation
 * - Time decay toward neutral (half-life: 77777 blocks)
 * - NO cross-domain transfer
 *
 * Score components:
 * - Accuracy: 50%
 * - Consistency: 25%
 * - Challenge Defense: 15%
 * - Calibration: 10%
 *
 * @module consensus/reputation_isolation
 */

// Types
export type {
  ReputationSource,
  GlobalReputation,
  StratumReputation,
  DomainReputation,
  IsolatedReputation,
  ReputationComponents,
  ReputationScoreResult,
  EffectiveReputationResult,
  ReputationUpdateInput,
} from "./types.js";

export {
  REPUTATION_ISOLATION_CONFIG,
  createEmptyGlobalReputation,
  createEmptyStratumReputation,
  createEmptyDomainReputation,
  createEmptyIsolatedReputation,
  hasDomainReputation,
  hasStratumReputation,
  getReputationSource,
} from "./types.js";

// Calculation
export {
  calculateDomainReputationScore,
  getEffectiveReputation,
  updateDomainReputation,
  updateVerdictConsistency,
  recordChallengeForReputation,
  applyTimeDecay,
  getReputationSummary,
  getScoreDescription,
  normalizeScore,
} from "./calculation.js";
