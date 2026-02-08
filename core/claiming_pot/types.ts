/**
 * Custom Claiming Pot Types
 *
 * A claiming pot is a user-created, fundable pool with configurable
 * eligibility rules that eligible accounts can claim from.
 *
 * Use cases: targeted airdrops, research incentive pools, community
 * reward pots, private distribution lists, third-party bounty funds.
 */

// ============ Eligibility ============

/**
 * Eligibility rules for claiming from a pot.
 * Modeled on RAIEligibility (core/tokens/types.ts:399-417).
 */
export interface PotEligibility {
  /** Minimum account age in blocks (0n = no restriction) */
  minAccountAge: bigint;

  /** Minimum activity score, 0-1 range (0 = no restriction) */
  minActivityScore: number;

  /** Maximum balance to qualify (undefined = no cap) */
  maxBalance?: bigint;

  /** Required verification level (0 = no restriction) */
  requiredVerificationLevel: number;

  /** Excluded account types (e.g., ["system", "contract"]) */
  excludedTypes: string[];

  /** Whitelist DIDs — if non-empty, ONLY these DIDs can claim */
  allowlist: string[];
}

/**
 * Default eligibility: public pot, no restrictions except system accounts
 */
export const DEFAULT_POT_ELIGIBILITY: PotEligibility = {
  minAccountAge: 0n,
  minActivityScore: 0,
  requiredVerificationLevel: 0,
  excludedTypes: ["system"],
  allowlist: [],
};

/**
 * Recommended amount per claim: 0.222 LGM (matches bootstrap fund per-agent amount).
 * Pot creators can override this — it's a sensible default, not a protocol requirement.
 */
export const DEFAULT_AMOUNT_PER_CLAIM = BigInt("222000000000000000"); // 0.222 × 10^18

// ============ Configuration ============

/**
 * Pot configuration — immutable after creation to prevent rug-pulls.
 */
export interface PotConfig {
  /** Amount of LGM per claim (18 decimals) */
  amountPerClaim: bigint;

  /** Maximum total claims allowed (0n = unlimited, until funds exhausted) */
  maxClaims: bigint;

  /** Maximum claims per account (typically 1n) */
  maxClaimsPerAccount: bigint;

  /** Eligibility rules */
  eligibility: PotEligibility;

  /** Expiration block (undefined = no expiration) */
  expiresAtBlock?: bigint;

  /** Whether creator can close the pot early and reclaim remaining funds */
  creatorCanClose: boolean;
}

// ============ Pot State ============

/**
 * Pot status lifecycle:
 *   active -> exhausted (funds depleted or maxClaims reached)
 *   active -> expired (past expiresAtBlock)
 *   active -> closed (creator closed it)
 *   exhausted -> active (re-funded via pot_fund)
 *   exhausted -> closed
 *   expired -> closed
 */
export type PotStatus = "active" | "exhausted" | "expired" | "closed";

/**
 * Claiming pot state.
 * Follows BootstrapFundState pattern (core/genesis/bootstrap_fund.ts:42-48).
 */
export interface ClaimingPot {
  /** Pot DID: did:lgm:pot:{32-hex} */
  potId: string;

  /** Creator DID */
  creator: string;

  /** Configuration (immutable after creation) */
  config: PotConfig;

  /** Total deposited (cumulative across all funding events) */
  totalDeposited: bigint;

  /** Total claimed so far */
  totalClaimed: bigint;

  /** Current available balance (deposited - claimed) */
  balance: bigint;

  /** Number of claims made */
  claimCount: bigint;

  /** DID -> claim count for per-account limit enforcement */
  claimedAccounts: Map<string, bigint>;

  /** Block when created */
  createdAtBlock: bigint;

  /** Block when closed (if closed) */
  closedAtBlock?: bigint;

  /** Current status */
  status: PotStatus;
}

// ============ Transaction Payloads ============

export interface PotCreatePayload {
  type: "pot_create";
  config: PotConfig;
  initialDeposit: bigint;
}

export interface PotFundPayload {
  type: "pot_fund";
  potId: string;
  amount: bigint;
}

export interface PotClaimPayload {
  type: "pot_claim";
  potId: string;
}

export interface PotClosePayload {
  type: "pot_close";
  potId: string;
}

// ============ Result Types ============

export interface EligibilityResult {
  eligible: boolean;
  reason?: string;
}

// ============ Errors ============

export type PotErrorCode =
  | "POT_NOT_FOUND"
  | "POT_NOT_ACTIVE"
  | "POT_EXPIRED"
  | "POT_EXHAUSTED"
  | "ALREADY_CLAIMED"
  | "MAX_CLAIMS_REACHED"
  | "INELIGIBLE"
  | "INSUFFICIENT_FUNDS"
  | "INSUFFICIENT_DEPOSIT"
  | "NOT_POT_CREATOR"
  | "INVALID_CONFIG";

export class PotError extends Error {
  constructor(
    message: string,
    public code: PotErrorCode,
  ) {
    super(message);
    this.name = "PotError";
  }
}
