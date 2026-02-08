/**
 * Custom Claiming Pot Module
 *
 * Enables user-created, fundable pools with configurable eligibility rules.
 *
 * Lifecycle: pot_create -> pot_fund (optional) -> pot_claim -> pot_close
 *
 * Key components:
 * - types.ts: All interfaces and error types
 * - pot_manager.ts: Pure logic functions (create, validate, evaluate eligibility)
 */

// ============ Types ============
export type {
  PotEligibility,
  PotConfig,
  PotStatus,
  ClaimingPot,
  PotCreatePayload,
  PotFundPayload,
  PotClaimPayload,
  PotClosePayload,
  EligibilityResult,
  PotErrorCode,
} from "./types.js";

export {
  DEFAULT_POT_ELIGIBILITY,
  DEFAULT_AMOUNT_PER_CLAIM,
  PotError,
} from "./types.js";

// ============ Manager ============
export type { AccountInfo } from "./pot_manager.js";

export {
  generatePotId,
  validatePotConfig,
  evaluateEligibility,
  createPot,
  clonePots,
} from "./pot_manager.js";
