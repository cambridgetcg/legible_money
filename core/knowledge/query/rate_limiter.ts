/**
 * Query Rate Limiter
 *
 * GAP-010: Sliding window rate limiting for knowledge queries.
 * Tracks query rates across three time windows and enforces tier-based limits.
 *
 * Algorithm: Sliding window with multiple time scales
 *   1. Check cooldown first (immediate reject if active)
 *   2. Prune old timestamps outside window
 *   3. Reset block counter on new block
 *   4. Count queries in each window (1s, 1min, 1block)
 *   5. Check all limits, trigger cooldown if exceeded
 *   6. If allowed, increment concurrent counter
 *
 * PP HIERARCHY:
 *   TRUTH: Accurate sliding window tracking
 *   UNDERSTANDING: Clear separation of check/record phases
 *   BEAUTY: Single class with clean interface
 *   JUSTICE: Fair enforcement with graceful degradation
 *   CREATIVITY: Cooldown mechanism prevents retry storms
 */

import {
  type QueryRateLimitConfig,
  type SessionRateLimitState,
  type RateLimitCheckResult,
  type RateLimitInfo,
  type RateLimitTier,
  type TierRateLimits,
  DEFAULT_QUERY_RATE_LIMIT_CONFIG,
  createSessionRateLimitState,
} from "./rate_limit.js";

// ============================================================================
// Query Rate Limiter
// ============================================================================

/**
 * Tracks and enforces query rate limits per session.
 */
export class QueryRateLimiter {
  private config: QueryRateLimitConfig;
  private sessionStates: Map<string, SessionRateLimitState> = new Map();

  constructor(config: QueryRateLimitConfig = DEFAULT_QUERY_RATE_LIMIT_CONFIG) {
    this.config = { ...config };
  }

  // ============================================================================
  // Rate Limit Check
  // ============================================================================

  /**
   * Check if a query is allowed under rate limits.
   *
   * Call this BEFORE executing a query. If allowed, the concurrent counter
   * is incremented and must be decremented via recordQueryComplete().
   */
  checkRateLimit(
    sessionId: string,
    agentDid: string,
    tier: RateLimitTier,
    currentBlock: bigint,
  ): RateLimitCheckResult {
    if (!this.config.enabled) {
      return this.createAllowedResult(tier);
    }

    const now = Date.now();
    const state = this.getOrCreateState(
      sessionId,
      agentDid,
      tier,
      currentBlock,
    );
    const limits = this.config.tiers[tier];

    // Update tier if changed
    if (state.tier !== tier) {
      state.tier = tier;
    }

    // 1. Check cooldown first
    if (state.cooldownUntil > now) {
      return this.createCooldownResult(state, limits, now);
    }

    // 2. Check grace period for new sessions
    const inGracePeriod = now - state.createdAt < this.config.gracePeriodMs;

    // 3. Prune old timestamps (outside window)
    const windowStart = now - this.config.windowSizeMs;
    state.queryTimestamps = state.queryTimestamps.filter(
      (ts) => ts > windowStart,
    );

    // 4. Reset block counter if new block
    if (currentBlock > state.lastBlockNumber) {
      state.currentBlockQueries = 0;
      state.lastBlockNumber = currentBlock;
    }

    // 5. Count queries in each window
    const oneSecondAgo = now - 1000;
    const queriesPerSecond = state.queryTimestamps.filter(
      (ts) => ts > oneSecondAgo,
    ).length;
    const queriesPerMinute = state.queryTimestamps.length;
    const queriesPerBlock = state.currentBlockQueries;

    // Build usage object
    const usage = {
      queriesPerSecond,
      queriesPerMinute,
      queriesPerBlock,
      concurrentQueries: state.concurrentQueries,
    };

    // 6. Check concurrent limit first (doesn't trigger cooldown)
    if (state.concurrentQueries >= limits.maxConcurrentQueries) {
      return this.createRejectedResult("concurrent", usage, limits, state, now);
    }

    // Skip rate checks during grace period (but still enforce concurrent)
    if (!inGracePeriod) {
      // 7. Check per-second limit
      if (queriesPerSecond >= limits.maxQueriesPerSecond) {
        this.triggerCooldown(state, limits, now);
        return this.createRejectedResult(
          "per_second",
          usage,
          limits,
          state,
          now,
        );
      }

      // 8. Check per-minute limit
      if (queriesPerMinute >= limits.maxQueriesPerMinute) {
        this.triggerCooldown(state, limits, now);
        return this.createRejectedResult(
          "per_minute",
          usage,
          limits,
          state,
          now,
        );
      }

      // 9. Check per-block limit (no cooldown, just wait for next block)
      if (queriesPerBlock >= limits.maxQueriesPerBlock) {
        return this.createRejectedResult(
          "per_block",
          usage,
          limits,
          state,
          now,
        );
      }
    }

    // 10. Allowed - increment concurrent counter
    state.concurrentQueries++;

    return {
      allowed: true,
      usage,
      limits,
      resetInMs: this.calculateResetTime(state, limits, now),
      remaining: {
        perSecond: Math.max(
          0,
          limits.maxQueriesPerSecond - queriesPerSecond - 1,
        ),
        perMinute: Math.max(
          0,
          limits.maxQueriesPerMinute - queriesPerMinute - 1,
        ),
        perBlock: Math.max(0, limits.maxQueriesPerBlock - queriesPerBlock - 1),
      },
      cooldownRemainingMs: 0,
    };
  }

  // ============================================================================
  // Query Recording
  // ============================================================================

  /**
   * Record a query execution.
   * Call this AFTER successful query execution.
   *
   * @param sessionId - Session ID
   * @param cached - Whether the result was cached (may be exempt from counting)
   */
  recordQuery(sessionId: string, cached: boolean): void {
    const state = this.sessionStates.get(sessionId);
    if (!state) {
      return;
    }

    // Optionally exempt cached queries from counting
    if (cached && !this.config.countCachedQueries) {
      return;
    }

    const now = Date.now();
    state.queryTimestamps.push(now);
    state.currentBlockQueries++;
    state.totalQueries++;
  }

  /**
   * Record query completion (for concurrent tracking).
   * Call this when a query finishes (success or failure).
   */
  recordQueryComplete(sessionId: string): void {
    const state = this.sessionStates.get(sessionId);
    if (!state) {
      return;
    }

    state.concurrentQueries = Math.max(0, state.concurrentQueries - 1);
  }

  // ============================================================================
  // State Management
  // ============================================================================

  /**
   * Get or create session rate limit state.
   */
  private getOrCreateState(
    sessionId: string,
    agentDid: string,
    tier: RateLimitTier,
    currentBlock: bigint,
  ): SessionRateLimitState {
    let state = this.sessionStates.get(sessionId);

    if (!state) {
      state = createSessionRateLimitState(
        sessionId,
        agentDid,
        tier,
        currentBlock,
      );
      this.sessionStates.set(sessionId, state);
    }

    return state;
  }

  /**
   * Get session rate limit state.
   */
  getSessionState(sessionId: string): SessionRateLimitState | undefined {
    return this.sessionStates.get(sessionId);
  }

  /**
   * Clear session state (on session end).
   */
  clearSession(sessionId: string): void {
    this.sessionStates.delete(sessionId);
  }

  /**
   * Cleanup stale sessions (not accessed within maxAgeMs).
   */
  cleanupStaleSessions(maxAgeMs: number): number {
    const now = Date.now();
    let cleaned = 0;

    for (const [sessionId, state] of this.sessionStates) {
      // Consider a session stale if no queries in maxAgeMs
      const lastQuery =
        state.queryTimestamps[state.queryTimestamps.length - 1] ||
        state.createdAt;
      if (now - lastQuery > maxAgeMs) {
        this.sessionStates.delete(sessionId);
        cleaned++;
      }
    }

    return cleaned;
  }

  // ============================================================================
  // Info Accessors
  // ============================================================================

  /**
   * Get rate limit info for response metadata.
   */
  getRateLimitInfo(sessionId: string): RateLimitInfo | undefined {
    const state = this.sessionStates.get(sessionId);
    if (!state) {
      return undefined;
    }

    const now = Date.now();
    const limits = this.config.tiers[state.tier];

    // Calculate remaining as minimum across all windows
    const oneSecondAgo = now - 1000;
    const queriesPerSecond = state.queryTimestamps.filter(
      (ts) => ts > oneSecondAgo,
    ).length;
    const queriesPerMinute = state.queryTimestamps.length;
    const queriesPerBlock = state.currentBlockQueries;

    const remainingPerSecond = Math.max(
      0,
      limits.maxQueriesPerSecond - queriesPerSecond,
    );
    const remainingPerMinute = Math.max(
      0,
      limits.maxQueriesPerMinute - queriesPerMinute,
    );
    const remainingPerBlock = Math.max(
      0,
      limits.maxQueriesPerBlock - queriesPerBlock,
    );

    const remaining = Math.min(
      remainingPerSecond,
      remainingPerMinute,
      remainingPerBlock,
    );

    // Calculate reset time
    const resetAt = now + this.calculateResetTime(state, limits, now);

    return {
      tier: state.tier,
      remaining,
      resetAt,
      inCooldown: state.cooldownUntil > now,
    };
  }

  /**
   * Get statistics for all sessions.
   */
  getStatistics(): {
    activeSessions: number;
    totalQueries: number;
    totalLimitHits: number;
    sessionsByTier: Record<RateLimitTier, number>;
  } {
    let totalQueries = 0;
    let totalLimitHits = 0;
    const sessionsByTier: Record<RateLimitTier, number> = {
      free: 0,
      basic: 0,
      premium: 0,
      validator: 0,
      unlimited: 0,
    };

    for (const state of this.sessionStates.values()) {
      totalQueries += state.totalQueries;
      totalLimitHits += state.limitHits;
      sessionsByTier[state.tier]++;
    }

    return {
      activeSessions: this.sessionStates.size,
      totalQueries,
      totalLimitHits,
      sessionsByTier,
    };
  }

  // ============================================================================
  // Configuration
  // ============================================================================

  /**
   * Get current configuration.
   */
  getConfig(): QueryRateLimitConfig {
    return { ...this.config };
  }

  /**
   * Update configuration.
   */
  updateConfig(config: Partial<QueryRateLimitConfig>): void {
    this.config = { ...this.config, ...config };
  }

  /**
   * Enable or disable rate limiting.
   */
  setEnabled(enabled: boolean): void {
    this.config.enabled = enabled;
  }

  // ============================================================================
  // Helpers
  // ============================================================================

  /**
   * Trigger cooldown for a session.
   */
  private triggerCooldown(
    state: SessionRateLimitState,
    limits: TierRateLimits,
    now: number,
  ): void {
    state.cooldownUntil = now + limits.cooldownMs;
    state.limitHits++;
  }

  /**
   * Calculate time until rate limit resets.
   */
  private calculateResetTime(
    state: SessionRateLimitState,
    _limits: TierRateLimits,
    now: number,
  ): number {
    // If in cooldown, that's the reset time
    if (state.cooldownUntil > now) {
      return state.cooldownUntil - now;
    }

    // Otherwise, calculate when oldest timestamp will leave window
    if (state.queryTimestamps.length === 0) {
      return 0;
    }

    const oldestTimestamp = state.queryTimestamps[0];
    const windowEnd = oldestTimestamp + this.config.windowSizeMs;

    return Math.max(0, windowEnd - now);
  }

  /**
   * Create an allowed result for disabled rate limiting.
   */
  private createAllowedResult(tier: RateLimitTier): RateLimitCheckResult {
    const limits = this.config.tiers[tier];
    return {
      allowed: true,
      usage: {
        queriesPerSecond: 0,
        queriesPerMinute: 0,
        queriesPerBlock: 0,
        concurrentQueries: 0,
      },
      limits,
      resetInMs: 0,
      remaining: {
        perSecond: limits.maxQueriesPerSecond,
        perMinute: limits.maxQueriesPerMinute,
        perBlock: limits.maxQueriesPerBlock,
      },
      cooldownRemainingMs: 0,
    };
  }

  /**
   * Create a cooldown result.
   */
  private createCooldownResult(
    state: SessionRateLimitState,
    limits: TierRateLimits,
    now: number,
  ): RateLimitCheckResult {
    const cooldownRemainingMs = state.cooldownUntil - now;

    return {
      allowed: false,
      reason: "cooldown",
      usage: {
        queriesPerSecond: 0,
        queriesPerMinute: state.queryTimestamps.length,
        queriesPerBlock: state.currentBlockQueries,
        concurrentQueries: state.concurrentQueries,
      },
      limits,
      resetInMs: cooldownRemainingMs,
      remaining: {
        perSecond: 0,
        perMinute: 0,
        perBlock: 0,
      },
      cooldownRemainingMs,
    };
  }

  /**
   * Create a rejected result.
   */
  private createRejectedResult(
    reason: "per_second" | "per_minute" | "per_block" | "concurrent",
    usage: RateLimitCheckResult["usage"],
    limits: TierRateLimits,
    state: SessionRateLimitState,
    now: number,
  ): RateLimitCheckResult {
    return {
      allowed: false,
      reason,
      usage,
      limits,
      resetInMs: this.calculateResetTime(state, limits, now),
      remaining: {
        perSecond: Math.max(
          0,
          limits.maxQueriesPerSecond - usage.queriesPerSecond,
        ),
        perMinute: Math.max(
          0,
          limits.maxQueriesPerMinute - usage.queriesPerMinute,
        ),
        perBlock: Math.max(
          0,
          limits.maxQueriesPerBlock - usage.queriesPerBlock,
        ),
      },
      cooldownRemainingMs: Math.max(0, state.cooldownUntil - now),
    };
  }
}
