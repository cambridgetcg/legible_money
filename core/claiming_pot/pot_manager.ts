/**
 * Claiming Pot Manager
 *
 * Pure functions for pot lifecycle: create, fund, claim, close.
 * No direct state access — receives data as parameters for testability.
 */

import { domainHash, sha512_256Hex } from "../crypto/hashing.js";
import type {
  ClaimingPot,
  PotConfig,
  PotEligibility,
  EligibilityResult,
} from "./types.js";
import { PotError } from "./types.js";

// ============ Pot ID Generation ============

/**
 * Generate a deterministic pot DID from creator + nonce.
 * Format: did:lgm:pot:{32-hex}
 */
export function generatePotId(creatorDid: string, nonce: bigint): string {
  const hashInput = `${creatorDid}:${nonce.toString()}`;
  const hash = sha512_256Hex(domainHash("LGM.pot.id.v1", hashInput));
  return `did:lgm:pot:${hash.slice(0, 32)}`;
}

// ============ Config Validation ============

/**
 * Validate pot configuration. Throws PotError on invalid config.
 */
export function validatePotConfig(
  config: PotConfig,
  currentBlock: bigint,
): void {
  if (config.amountPerClaim <= 0n) {
    throw new PotError("amountPerClaim must be positive", "INVALID_CONFIG");
  }

  if (config.maxClaimsPerAccount < 1n) {
    throw new PotError(
      "maxClaimsPerAccount must be at least 1",
      "INVALID_CONFIG",
    );
  }

  if (config.maxClaims < 0n) {
    throw new PotError("maxClaims cannot be negative", "INVALID_CONFIG");
  }

  if (
    config.expiresAtBlock !== undefined &&
    config.expiresAtBlock <= currentBlock
  ) {
    throw new PotError(
      "expiresAtBlock must be in the future",
      "INVALID_CONFIG",
    );
  }

  if (
    config.eligibility.minActivityScore < 0 ||
    config.eligibility.minActivityScore > 1
  ) {
    throw new PotError(
      "minActivityScore must be between 0 and 1",
      "INVALID_CONFIG",
    );
  }
}

// ============ Eligibility Evaluation ============

/**
 * Account state subset needed for eligibility checks.
 * Avoids importing the full AccountState type.
 */
export interface AccountInfo {
  did: string;
  balance: bigint;
  createdAtBlock: bigint;
  reputationScore: number;
  verificationLevel: number;
  accountType?: string;
}

/**
 * Evaluate whether an account is eligible to claim from a pot.
 */
export function evaluateEligibility(
  eligibility: PotEligibility,
  claimant: AccountInfo,
  currentBlock: bigint,
): EligibilityResult {
  // 1. Allowlist check (fast path)
  if (
    eligibility.allowlist.length > 0 &&
    !eligibility.allowlist.includes(claimant.did)
  ) {
    return { eligible: false, reason: "Not in allowlist" };
  }

  // 2. Excluded types
  if (
    claimant.accountType &&
    eligibility.excludedTypes.includes(claimant.accountType)
  ) {
    return {
      eligible: false,
      reason: `Account type "${claimant.accountType}" is excluded`,
    };
  }

  // 3. Account age
  const age = currentBlock - claimant.createdAtBlock;
  if (age < eligibility.minAccountAge) {
    return {
      eligible: false,
      reason: `Account age ${age} < minimum ${eligibility.minAccountAge}`,
    };
  }

  // 4. Activity score
  if (claimant.reputationScore < eligibility.minActivityScore) {
    return {
      eligible: false,
      reason: `Activity score ${claimant.reputationScore} < minimum ${eligibility.minActivityScore}`,
    };
  }

  // 5. Max balance
  if (
    eligibility.maxBalance !== undefined &&
    claimant.balance > eligibility.maxBalance
  ) {
    return {
      eligible: false,
      reason: `Balance ${claimant.balance} exceeds maximum ${eligibility.maxBalance}`,
    };
  }

  // 6. Verification level
  if (claimant.verificationLevel < eligibility.requiredVerificationLevel) {
    return {
      eligible: false,
      reason: `Verification level ${claimant.verificationLevel} < required ${eligibility.requiredVerificationLevel}`,
    };
  }

  return { eligible: true };
}

// ============ Pot Creation ============

/**
 * Create a new ClaimingPot object.
 */
export function createPot(
  potId: string,
  config: PotConfig,
  creatorDid: string,
  initialDeposit: bigint,
  blockNumber: bigint,
): ClaimingPot {
  return {
    potId,
    creator: creatorDid,
    config,
    totalDeposited: initialDeposit,
    totalClaimed: 0n,
    balance: initialDeposit,
    claimCount: 0n,
    claimedAccounts: new Map(),
    createdAtBlock: blockNumber,
    status: "active",
  };
}

// ============ Checkpoint Support ============

/**
 * Deep clone claiming pots map for checkpoint/rollback support.
 * Follows the cloneSubscriptions pattern in state_manager.ts.
 */
export function clonePots(
  pots: Map<string, ClaimingPot>,
): Map<string, ClaimingPot> {
  const cloned = new Map<string, ClaimingPot>();
  for (const [id, pot] of pots) {
    cloned.set(id, {
      ...pot,
      config: {
        ...pot.config,
        eligibility: {
          ...pot.config.eligibility,
          excludedTypes: [...pot.config.eligibility.excludedTypes],
          allowlist: [...pot.config.eligibility.allowlist],
        },
      },
      claimedAccounts: new Map(pot.claimedAccounts),
    });
  }
  return cloned;
}
