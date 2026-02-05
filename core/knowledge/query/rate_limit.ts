/**
 * Query Rate Limiting Types
 *
 * GAP-010: Sliding window rate limiting at the query execution layer.
 * Protects against query spam with tier-based limits and cooldown periods.
 *
 * Three time windows:
 *   - Per-second: Burst protection
 *   - Per-minute: Sustained rate (sliding window)
 *   - Per-block: Consensus-aligned limiting (~2.5s)
 *
 * PP HIERARCHY:
 *   TRUTH: Accurate tracking with sliding windows
 *   UNDERSTANDING: Clear tier system aligned with SessionWallet presets
 *   BEAUTY: Minimal integration, single check point
 *   JUSTICE: Fair resource allocation by stake level
 *   CREATIVITY: Novel per-block limiting aligned with consensus rhythm
 */

// ============================================================================
// Rate Limit Tiers
// ============================================================================

/**
 * Rate limit tier - determines query quotas.
 * Aligns with existing tier system from RPC and SessionWallet.
 */
export type RateLimitTier =
  | "free"
  | "basic"
  | "premium"
  | "validator"
  | "unlimited";

/**
 * Rate limit configuration per tier.
 */
export interface TierRateLimits {
  /** Max queries per second (burst protection) */
  maxQueriesPerSecond: number;
  /** Max queries per minute (sustained rate) */
  maxQueriesPerMinute: number;
  /** Max queries per block (~2.5s, consensus-aligned) */
  maxQueriesPerBlock: number;
  /** Cooldown duration in ms after hitting limit */
  cooldownMs: number;
  /** Max concurrent queries (in-flight) */
  maxConcurrentQueries: number;
}

// ============================================================================
// Configuration
// ============================================================================

/**
 * Query rate limit configuration.
 */
export interface QueryRateLimitConfig {
  /** Enable rate limiting */
  enabled: boolean;
  /** Rate limits per tier */
  tiers: Record<RateLimitTier, TierRateLimits>;
  /** Sliding window size in ms (for per-minute tracking) */
  windowSizeMs: number;
  /** Grace period for first queries in new session (ms) */
  gracePeriodMs: number;
  /** Whether cached queries count against limit */
  countCachedQueries: boolean;
}

/**
 * Default rate limit configuration.
 *
 * Tier progression:
 *   free → basic → premium → validator → unlimited
 *
 * Limits scale ~5x between tiers for /sec and /min,
 * with proportional cooldown reduction.
 */
export const DEFAULT_QUERY_RATE_LIMIT_CONFIG: QueryRateLimitConfig = {
  enabled: true,
  tiers: {
    free: {
      maxQueriesPerSecond: 2,
      maxQueriesPerMinute: 20,
      maxQueriesPerBlock: 3,
      cooldownMs: 30000, // 30 seconds
      maxConcurrentQueries: 1,
    },
    basic: {
      maxQueriesPerSecond: 5,
      maxQueriesPerMinute: 100,
      maxQueriesPerBlock: 8,
      cooldownMs: 10000, // 10 seconds
      maxConcurrentQueries: 3,
    },
    premium: {
      maxQueriesPerSecond: 20,
      maxQueriesPerMinute: 500,
      maxQueriesPerBlock: 30,
      cooldownMs: 5000, // 5 seconds
      maxConcurrentQueries: 10,
    },
    validator: {
      maxQueriesPerSecond: 100,
      maxQueriesPerMinute: 2000,
      maxQueriesPerBlock: 150,
      cooldownMs: 1000, // 1 second
      maxConcurrentQueries: 50,
    },
    unlimited: {
      maxQueriesPerSecond: Number.MAX_SAFE_INTEGER,
      maxQueriesPerMinute: Number.MAX_SAFE_INTEGER,
      maxQueriesPerBlock: Number.MAX_SAFE_INTEGER,
      cooldownMs: 0,
      maxConcurrentQueries: Number.MAX_SAFE_INTEGER,
    },
  },
  windowSizeMs: 60000, // 1 minute sliding window
  gracePeriodMs: 5000, // 5 second grace for new sessions
  countCachedQueries: false, // Cached queries are free
};

// ============================================================================
// Session State
// ============================================================================

/**
 * Rate limit state for a session.
 * Tracks query history for sliding window calculations.
 */
export interface SessionRateLimitState {
  /** Session ID */
  sessionId: string;
  /** Agent DID */
  agentDid: string;
  /** Current rate limit tier */
  tier: RateLimitTier;
  /** Query timestamps for sliding window (within windowSizeMs) */
  queryTimestamps: number[];
  /** Query count in current block */
  currentBlockQueries: number;
  /** Last block number tracked */
  lastBlockNumber: bigint;
  /** Currently in-flight query count */
  concurrentQueries: number;
  /** Cooldown end timestamp (0 if not in cooldown) */
  cooldownUntil: number;
  /** Session creation time (for grace period) */
  createdAt: number;
  /** Total queries this session (for stats) */
  totalQueries: number;
  /** Total rate limit hits (for stats) */
  limitHits: number;
}

// ============================================================================
// Check Results
// ============================================================================

/**
 * Reason for rate limit rejection.
 */
export type RateLimitReason =
  | "per_second"
  | "per_minute"
  | "per_block"
  | "concurrent"
  | "cooldown";

/**
 * Result of a rate limit check.
 */
export interface RateLimitCheckResult {
  /** Whether the query is allowed */
  allowed: boolean;
  /** Reason for rejection (if not allowed) */
  reason?: RateLimitReason;
  /** Current usage metrics */
  usage: {
    queriesPerSecond: number;
    queriesPerMinute: number;
    queriesPerBlock: number;
    concurrentQueries: number;
  };
  /** Limits for this tier */
  limits: TierRateLimits;
  /** Time until rate limit resets (ms) */
  resetInMs: number;
  /** Remaining queries in each window */
  remaining: {
    perSecond: number;
    perMinute: number;
    perBlock: number;
  };
  /** Cooldown time remaining (ms), 0 if not in cooldown */
  cooldownRemainingMs: number;
}

/**
 * Rate limit info for response metadata.
 * Minimal subset for client awareness.
 */
export interface RateLimitInfo {
  /** Current tier */
  tier: RateLimitTier;
  /** Minimum remaining across all windows */
  remaining: number;
  /** Timestamp when limit resets */
  resetAt: number;
  /** Whether currently in cooldown */
  inCooldown: boolean;
}

// ============================================================================
// Helpers
// ============================================================================

/**
 * Create initial session rate limit state.
 */
export function createSessionRateLimitState(
  sessionId: string,
  agentDid: string,
  tier: RateLimitTier,
  currentBlock: bigint,
): SessionRateLimitState {
  return {
    sessionId,
    agentDid,
    tier,
    queryTimestamps: [],
    currentBlockQueries: 0,
    lastBlockNumber: currentBlock,
    concurrentQueries: 0,
    cooldownUntil: 0,
    createdAt: Date.now(),
    totalQueries: 0,
    limitHits: 0,
  };
}

/**
 * Determine rate limit tier from session budget.
 * Higher budget = higher tier.
 */
export function determineTierFromBudget(budgetLimit?: bigint): RateLimitTier {
  if (!budgetLimit) {
    return "free";
  }

  const LGM = 10n ** 18n;

  // Validator tier: 10,000+ LGM budget
  if (budgetLimit >= 10000n * LGM) {
    return "validator";
  }

  // Premium tier: 1,000+ LGM budget
  if (budgetLimit >= 1000n * LGM) {
    return "premium";
  }

  // Basic tier: 100+ LGM budget
  if (budgetLimit >= 100n * LGM) {
    return "basic";
  }

  return "free";
}
