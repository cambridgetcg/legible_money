/**
 * Expirable Tokens Module Types
 *
 * Implements token expiration using the Dynamic Array method from the patent:
 * "System And Method For Schedule Message And Self-Invoking Smart Contracts"
 *
 * Key Concept: Tokens have a limited lifespan to encourage economic activity
 * and enable Regenerative Agent Income (RAI) - a self-sustaining UBI system.
 *
 * From Patent: "Token Expiration is calculated by the Dynamic Array method, in which
 * each balance position in array is reset at regular intervals and child accounts
 * are created for each time period."
 *
 * PP HIERARCHY:
 *   TRUTH: Token expiration creates natural economic velocity
 *   UNDERSTANDING: Dynamic Array tracks minting time without per-token metadata
 *   BEAUTY: Elegant O(1) balance queries with automatic expiration
 *   JUSTICE: Expired tokens return to RAI fund for redistribution
 *   CREATIVITY: Novel token lifecycle enables circular economy
 */

import type { DID } from "../agents/identity/did.js";

// ============ Core Expirable Token Types ============

/**
 * Token with expiration metadata
 * Unlike standard ERC-20, expirable tokens track when they were minted
 */
export interface ExpirableBalance {
  /** Account DID */
  account: DID;

  /** Total non-expired balance (sum of all child accounts) */
  totalBalance: bigint;

  /** Balance by expiration period (child account structure) */
  periodBalances: Map<number, bigint>;

  /** Earliest non-zero period (for efficient iteration) */
  earliestPeriod: number;

  /** Latest non-zero period */
  latestPeriod: number;

  /** Total expired (for analytics) */
  totalExpired: bigint;

  /** Last update timestamp */
  updatedAt: number;
}

/**
 * Expiration period identifier
 * Period = floor(mintTimestamp / periodDuration)
 */
export type ExpirationPeriod = number;

/**
 * Child account representing tokens minted in a specific period
 * From Patent: "child accounts are created for each time period"
 */
export interface ChildAccount {
  /** Parent account DID */
  parentAccount: DID;

  /** Period this child account represents */
  period: ExpirationPeriod;

  /** Balance in this period */
  balance: bigint;

  /** When tokens in this period expire */
  expiresAt: number;

  /** Whether this period has been swept (expired tokens retrieved) */
  swept: boolean;

  /** Block when swept (if swept) */
  sweptAtBlock?: bigint;

  /** Whether tokens in this period are locked (lock_and_unlock mode) */
  locked: boolean;

  /** Block when locked (if locked) */
  lockedAtBlock?: bigint;
}

/**
 * Token mint event with expiration tracking
 */
export interface ExpirableMint {
  /** Unique mint ID */
  id: string;

  /** Recipient account */
  recipient: DID;

  /** Amount minted */
  amount: bigint;

  /** Period this mint belongs to */
  period: ExpirationPeriod;

  /** When these tokens expire */
  expiresAt: number;

  /** Source of mint (rewards, transfer, RAI, etc.) */
  source: MintSource;

  /** Related entity (claim ID, transfer ID, etc.) */
  sourceId?: string;

  /** Block when minted */
  mintedAtBlock: bigint;

  /** Timestamp when minted */
  mintedAt: number;
}

/**
 * Source of token minting
 */
export type MintSource =
  | "genesis" // Initial distribution
  | "block_reward" // Block production reward
  | "verification_reward" // PoT verification reward
  | "challenge_reward" // Successful challenge reward
  | "vesting_release" // Vesting schedule release
  | "rai_distribution" // RAI (UBI) distribution
  | "transfer" // Transfer from another account
  | "protocol_emission"; // Scheduled protocol emission

// ============ Dynamic Array Types ============

/**
 * Dynamic Array state for an account
 * Implements the patent's efficient expiration tracking
 */
export interface DynamicArrayState {
  /** Account DID */
  account: DID;

  /** Array of period balances (sparse representation) */
  periods: Map<ExpirationPeriod, bigint>;

  /** Current period pointer */
  currentPeriod: ExpirationPeriod;

  /** Head pointer (earliest non-zero period) */
  headPeriod: ExpirationPeriod;

  /** Tail pointer (latest non-zero period) */
  tailPeriod: ExpirationPeriod;

  /** Total active (non-expired) balance */
  activeBalance: bigint;

  /** Pending expiration in next period */
  pendingExpiration: bigint;
}

/**
 * Period transition event (when a period expires)
 */
export interface PeriodTransition {
  /** Period that expired */
  expiredPeriod: ExpirationPeriod;

  /** New current period */
  newPeriod: ExpirationPeriod;

  /** Total tokens expired in this transition */
  expiredAmount: bigint;

  /** Number of accounts affected */
  accountsAffected: number;

  /** Block when transition occurred */
  transitionBlock: bigint;
}

// ============ Token Expiration Configuration ============

/**
 * Expiration configuration
 */
export interface ExpirationConfig {
  /** Duration of each period in milliseconds */
  periodDurationMs: number;

  /** Number of periods until expiration */
  periodsUntilExpiration: number;

  /** Total token lifespan in milliseconds */
  tokenLifespanMs: number;

  /** Minimum balance to track (dust threshold) */
  dustThreshold: bigint;

  /** Grace period after expiration (blocks) before retrieval */
  graceBlocks: bigint;

  /** Whether to allow expiration extension (staking, etc.) */
  allowExtension: boolean;

  /** Maximum extension periods */
  maxExtensionPeriods: number;

  /**
   * Retrieval rate (0.0-1.0) - percentage tax on transfers routed to RAI
   * From Patent: "Retrieval Rate (R2)" enables self-sustaining RUBI
   */
  retrievalRate: number;

  /**
   * Renewal fee rate for unlocking expired tokens (0.0-1.0)
   * From Patent: "Renewal Fee Rate (R1)" - defaults to retrievalRate if not set
   */
  renewalFeeRate?: number;

  /**
   * Expiration mode determining what happens when tokens expire
   * - 'direct_retrieval': Expired tokens are swept and returned to RAI (default)
   * - 'lock_and_unlock': Expired tokens freeze in-place, can be unlocked with renewal fee
   */
  expirationMode: "direct_retrieval" | "lock_and_unlock";
}

/**
 * Default expiration configuration
 * Tokens expire after ~6 months with weekly periods
 */
export const DEFAULT_EXPIRATION_CONFIG: ExpirationConfig = {
  periodDurationMs: 7 * 24 * 60 * 60 * 1000, // 1 week
  periodsUntilExpiration: 26, // ~6 months
  tokenLifespanMs: 26 * 7 * 24 * 60 * 60 * 1000, // ~6 months
  dustThreshold: 1000n, // Minimum trackable amount
  graceBlocks: 1000n, // ~42 minutes grace
  allowExtension: true,
  maxExtensionPeriods: 52, // Max 1 year total
  retrievalRate: 0, // No transfer tax by default
  expirationMode: "direct_retrieval", // Sweep expired tokens to RAI
};

// ============ Account Type Configuration ============

/**
 * Account type for per-type expiration config overrides
 * From Patent: Different account classes may have different token lifecycles
 */
export type AccountType = "agent" | "validator" | "system" | "contract";

/**
 * Per-type config override (periodDurationMs and tokenLifespanMs must stay global)
 */
export type AccountTypeConfig = Partial<
  Omit<ExpirationConfig, "periodDurationMs" | "tokenLifespanMs">
>;

// ============ Retrieval Types (Clawback) ============

/**
 * Expired token retrieval request
 * From Patent: "retrieves uncirculated currency and is returned
 * back to the UBI Holding Account 132400"
 */
export interface RetrievalRequest {
  /** Unique retrieval ID */
  id: string;

  /** Period being retrieved */
  period: ExpirationPeriod;

  /** Accounts to retrieve from (empty = all with expired balances) */
  targetAccounts?: DID[];

  /** Block when retrieval was requested */
  requestedAtBlock: bigint;

  /** Status of retrieval */
  status: RetrievalStatus;
}

export type RetrievalStatus =
  | "pending" // Waiting to execute
  | "executing" // In progress
  | "completed" // Successfully retrieved
  | "failed"; // Failed (will retry)

/**
 * Retrieval result for a single account
 */
export interface AccountRetrievalResult {
  /** Account DID */
  account: DID;

  /** Amount retrieved */
  amount: bigint;

  /** Period retrieved from */
  period: ExpirationPeriod;

  /** Whether account balance is now zero */
  balanceZeroed: boolean;
}

/**
 * Batch retrieval result
 */
export interface BatchRetrievalResult {
  /** Retrieval request ID */
  requestId: string;

  /** Period retrieved */
  period: ExpirationPeriod;

  /** Total amount retrieved */
  totalRetrieved: bigint;

  /** Number of accounts processed */
  accountsProcessed: number;

  /** Individual results */
  results: AccountRetrievalResult[];

  /** Block when completed */
  completedAtBlock: bigint;

  /** Destination (RAI fund address) */
  destination: DID;
}

// ============ RAI (Regenerative Agent Income) Types ============

/**
 * RAI Holding Account state
 * From Patent: "UBI Holding Account 132400"
 */
export interface RAIHoldingAccount {
  /** Holding account identifier */
  id: string;

  /** Current balance from expired tokens */
  balance: bigint;

  /** Total received from expirations */
  totalReceived: bigint;

  /** Total distributed via RAI */
  totalDistributed: bigint;

  /** Last distribution block */
  lastDistributionBlock: bigint;

  /** Distribution schedule ID */
  distributionScheduleId?: string;

  /** Total received from transfer fees (retrieval rate) */
  totalReceivedFromFees: bigint;
}

/**
 * RAI distribution event
 */
export interface RAIDistribution {
  /** Distribution ID */
  id: string;

  /** Amount distributed */
  amount: bigint;

  /** Number of recipients */
  recipientCount: number;

  /** Amount per recipient */
  amountPerRecipient: bigint;

  /** Block when distributed */
  distributedAtBlock: bigint;

  /** Period this distribution is for */
  forPeriod: ExpirationPeriod;
}

/**
 * RAI eligibility criteria
 */
export interface RAIEligibility {
  /** Minimum account age (blocks) */
  minAccountAge: bigint;

  /** Minimum activity score */
  minActivityScore: number;

  /** Maximum balance to qualify (optional) */
  maxBalance?: bigint;

  /** Required verification level */
  requiredVerificationLevel: number;

  /** Excluded account types */
  excludedTypes: string[];
}

// ============ Transfer Types ============

/**
 * Transfer with expiration handling
 * When transferring, newest tokens are transferred first (LIFO)
 */
export interface ExpirableTransfer {
  /** Transfer ID */
  id: string;

  /** Sender account */
  from: DID;

  /** Recipient account */
  to: DID;

  /** Total amount transferred */
  amount: bigint;

  /** Breakdown by period (which tokens were transferred) */
  periodBreakdown: Map<ExpirationPeriod, bigint>;

  /** Block when transferred */
  transferredAtBlock: bigint;

  /** Whether recipient receives fresh expiration or inherits */
  freshExpiration: boolean;
}

/**
 * Transfer policy options
 */
export interface TransferPolicy {
  /** Transfer newest first (LIFO) or oldest first (FIFO) */
  order: "lifo" | "fifo";

  /** Whether recipient gets fresh expiration */
  freshExpirationOnReceive: boolean;

  /** Whether to allow partial transfers when some tokens expired mid-tx */
  allowPartialOnExpiration: boolean;

  /** Whether to apply retrieval rate (transfer tax) on this transfer */
  applyRetrievalRate: boolean;
}

export const DEFAULT_TRANSFER_POLICY: TransferPolicy = {
  order: "lifo", // Transfer newest first (preserves value)
  freshExpirationOnReceive: true, // Recipients get full lifespan
  allowPartialOnExpiration: true, // Don't fail entire tx if some expire
  applyRetrievalRate: true, // Apply retrieval rate when configured
};

// ============ Transfer Fee Result Types ============

/**
 * Result of a transfer with retrieval rate applied
 * From Patent: "Retrieval Rate (R2)" on every transfer
 */
export interface TransferWithFeeResult {
  /** Gross amount debited from sender */
  grossAmount: bigint;

  /** Fee routed to RAI holding (grossAmount * retrievalRate) */
  fee: bigint;

  /** Net amount credited to recipient (grossAmount - fee) */
  netAmount: bigint;

  /** Period breakdown of debited tokens from sender */
  periodBreakdown: Map<ExpirationPeriod, bigint>;

  /** Period the recipient's tokens were credited to */
  recipientPeriod: ExpirationPeriod;
}

/**
 * Result of unlocking expired (locked) tokens
 * From Patent: "Lock and Unlock" method with renewal fee
 */
export interface UnlockResult {
  /** Total locked balance that was unlocked */
  unlocked: bigint;

  /** Renewal fee charged (unlocked * renewalFeeRate) */
  renewalFee: bigint;

  /** Net amount re-activated (unlocked - renewalFee) */
  netUnlocked: bigint;

  /** New period the unlocked tokens were credited to */
  newPeriod: ExpirationPeriod;

  /** Periods that were unlocked */
  unlockedPeriods: ExpirationPeriod[];
}

// ============ Events ============

/**
 * Token lifecycle events
 */
export type TokenEvent =
  | { type: "minted"; data: ExpirableMint }
  | { type: "transferred"; data: ExpirableTransfer }
  | {
      type: "expired";
      data: { account: DID; period: ExpirationPeriod; amount: bigint };
    }
  | { type: "retrieved"; data: BatchRetrievalResult }
  | { type: "period_transition"; data: PeriodTransition }
  | { type: "rai_distributed"; data: RAIDistribution }
  | {
      type: "extended";
      data: { account: DID; amount: bigint; newExpiration: number };
    }
  | {
      type: "transfer_fee_collected";
      data: {
        from: DID;
        to: DID;
        grossAmount: bigint;
        fee: bigint;
        netAmount: bigint;
      };
    }
  | {
      type: "locked";
      data: {
        account: DID;
        period: ExpirationPeriod;
        amount: bigint;
        block: bigint;
      };
    }
  | {
      type: "unlocked";
      data: {
        account: DID;
        unlocked: bigint;
        renewalFee: bigint;
        netUnlocked: bigint;
        newPeriod: ExpirationPeriod;
      };
    };

// ============ Participation-Based Expiration Types ============

/**
 * Metrics collected about an account's protocol participation.
 * This interface lives in core/tokens/ to avoid layer violations.
 * Higher layers (state, consensus) provide these values via callback.
 */
export interface ParticipationMetrics {
  /** Number of claims submitted (all-time) */
  claimsSubmitted: number;
  /** Number of claims accepted by verification */
  claimsAccepted: number;
  /** Number of verification rounds completed */
  verificationsCompleted: number;
  /** Number of verification verdicts that were correct */
  verificationsCorrect: number;
  /** Number of knowledge queries made (paying for knowledge) */
  knowledgeQueriesMade: number;
  /** Number of governance votes cast */
  governanceVotes: number;
  /** Active stake amount */
  activeStake: bigint;
  /** Block number of last on-chain activity */
  lastActivityBlock: bigint;
  /** Current block number (for decay calculation) */
  currentBlock: bigint;
  /** Block number when account was created (for age calculation) */
  accountCreatedBlock: bigint;
}

/**
 * Result of computing a participation score.
 * The score determines how many periods of expiration the account earns.
 */
export interface ParticipationScore {
  /** Normalized total score (0-1000) */
  score: number;
  /** Per-component scores (0-1000 each) */
  components: {
    claims: number;
    verifications: number;
    queries: number;
    governance: number;
    stake: number;
  };
  /** Time-decay multiplier applied (0-1, where 1 = fully active) */
  decayFactor: number;
  /** Computed periods until expiration for this account */
  periodsGranted: number;
}

/**
 * Configuration for the participation-to-expiration mapping.
 * Controls weights, thresholds, and decay for the scoring formula.
 */
export interface ParticipationConfig {
  /** Minimum periods (for zero participation). Maps to ~2 days when periodDurationMs = 1 day. */
  minPeriods: number;
  /** Maximum periods (for full participation). Maps to ~6 months when periodDurationMs = 1 day. */
  maxPeriods: number;
  /** Half-life for activity decay in blocks. After this many blocks of inactivity, score halves. */
  decayHalfLifeBlocks: bigint;
  /** Weight for each component (must sum to 1.0) */
  weights: {
    claims: number;
    verifications: number;
    queries: number;
    governance: number;
    stake: number;
  };
  /** Threshold values for maximum score in each component */
  thresholds: {
    claimsForMax: number;
    verificationsForMax: number;
    queriesForMax: number;
    votesForMax: number;
    stakeForMax: bigint;
  };
  /** Minimum account age in blocks before participation scoring activates.
   *  Accounts younger than this get minPeriods. */
  minAccountAgeBlocks: bigint;
}

/**
 * Default participation configuration.
 *
 * With daily periods (periodDurationMs = 86400000):
 * - minPeriods: 2 (2 days for zero contribution)
 * - maxPeriods: 182 (~6 months for max contribution)
 *
 * Decay half-life: ~2.9 days at 2521ms blocks (100,000 blocks).
 */
export const DEFAULT_PARTICIPATION_CONFIG: ParticipationConfig = {
  minPeriods: 2,
  maxPeriods: 182,
  decayHalfLifeBlocks: 100_000n,
  weights: {
    claims: 0.25,
    verifications: 0.3,
    queries: 0.15,
    governance: 0.1,
    stake: 0.2,
  },
  thresholds: {
    claimsForMax: 50,
    verificationsForMax: 200,
    queriesForMax: 100,
    votesForMax: 20,
    stakeForMax: 10_000n * 10n ** 18n,
  },
  minAccountAgeBlocks: 0n,
};

/**
 * Callback that resolves participation score for an account.
 * Returns null to fall back to static config.
 * Injected into ChildAccountManager by the wiring layer.
 */
export type ParticipationResolver = (account: DID) => ParticipationScore | null;

// ============ Error Types ============

export class TokenError extends Error {
  constructor(
    message: string,
    public code: TokenErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "TokenError";
  }
}

export type TokenErrorCode =
  | "INSUFFICIENT_BALANCE"
  | "ACCOUNT_NOT_FOUND"
  | "PERIOD_NOT_FOUND"
  | "TOKENS_EXPIRED"
  | "RETRIEVAL_FAILED"
  | "INVALID_PERIOD"
  | "EXTENSION_NOT_ALLOWED"
  | "MAX_EXTENSION_EXCEEDED"
  | "RAI_NOT_ELIGIBLE"
  | "DUST_AMOUNT"
  | "TRANSFER_FAILED"
  | "INVALID_RETRIEVAL_RATE"
  | "PERIOD_NOT_LOCKED"
  | "NO_LOCKED_BALANCE"
  | "INVALID_TYPE_CONFIG"
  | "PARTICIPATION_RESOLVER_ERROR";
