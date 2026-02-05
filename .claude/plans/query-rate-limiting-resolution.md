# GAP-010: Query Rate Limiting Resolution

## Summary

Design and implement **query-level rate limiting** at the knowledge query execution layer to protect against query spam and ensure fair resource allocation across agents.

**Core Mechanism**: Sliding window rate limiting with tier-based quotas, cooldown periods, and graceful degradation.

---

## Problem Statement

**Current Gap**: While RPC-level rate limiting exists (IP-based), there's no protection at the knowledge query execution layer. A malicious or misconfigured agent can:
1. Exhaust session budget rapidly without throttling
2. Overwhelm the query service with burst traffic
3. Starve other agents of query resources
4. Trigger excessive prefetching and cache thrashing

**Existing Protections** (insufficient alone):
- RPC rate limiting: IP-based, can be bypassed by distributed requests
- Session budget: Only limits total spend, not query rate
- Free tier: Limits free queries, not query frequency

---

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                 QUERY RATE LIMITING STACK                    │
│                                                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │                UnifiedQueryService                    │   │
│  │                                                       │   │
│  │  processIntent()                                      │   │
│  │    ├── 1. Session validation ✓ (exists)              │   │
│  │    ├── 2. RATE LIMIT CHECK ← NEW                     │   │
│  │    ├── 3. Cache check ✓ (exists)                     │   │
│  │    ├── 4. Budget check ✓ (exists)                    │   │
│  │    └── 5. Execute query                              │   │
│  └──────────────────────────────────────────────────────┘   │
│                             │                                │
│                             ▼                                │
│  ┌──────────────────────────────────────────────────────┐   │
│  │              QueryRateLimiter                         │   │
│  │                                                       │   │
│  │  ├── checkRateLimit(sessionId, agentDid, tier)       │   │
│  │  ├── recordQuery(sessionId)                          │   │
│  │  ├── getCooldownRemaining(sessionId)                 │   │
│  │  └── getRateLimitInfo(sessionId)                     │   │
│  └──────────────────────────────────────────────────────┘   │
│                             │                                │
│         ┌───────────────────┼───────────────────┐           │
│         ▼                   ▼                   ▼           │
│  ┌────────────┐    ┌──────────────┐    ┌──────────────┐    │
│  │ Per-Second │    │ Per-Minute   │    │ Per-Block    │    │
│  │ Window     │    │ Window       │    │ Window       │    │
│  └────────────┘    └──────────────┘    └──────────────┘    │
└─────────────────────────────────────────────────────────────┘
```

---

## Implementation Plan

### Phase 1: Core Types

**File**: `core/knowledge/query/rate_limit.ts`

```typescript
/**
 * Rate limit tier - aligns with existing tier system
 */
export type RateLimitTier = 'free' | 'basic' | 'premium' | 'validator' | 'unlimited';

/**
 * Rate limit configuration per tier
 */
export interface TierRateLimits {
  /** Max queries per second (burst protection) */
  maxQueriesPerSecond: number;
  /** Max queries per minute (sustained rate) */
  maxQueriesPerMinute: number;
  /** Max queries per block (~2.5s) */
  maxQueriesPerBlock: number;
  /** Cooldown duration in ms after hitting limit */
  cooldownMs: number;
  /** Max concurrent queries (in-flight) */
  maxConcurrentQueries: number;
}

/**
 * Rate limit configuration
 */
export interface QueryRateLimitConfig {
  /** Enable rate limiting */
  enabled: boolean;
  /** Rate limits per tier */
  tiers: Record<RateLimitTier, TierRateLimits>;
  /** Sliding window size in ms */
  windowSizeMs: number;
  /** Grace period for first queries (ms) */
  gracePeriodMs: number;
  /** Cached queries count against limit? */
  countCachedQueries: boolean;
}

/**
 * Session rate limit state
 */
export interface SessionRateLimitState {
  sessionId: string;
  agentDid: string;
  tier: RateLimitTier;
  /** Query timestamps for sliding window */
  queryTimestamps: number[];
  /** Current block query count */
  currentBlockQueries: number;
  /** Last block number tracked */
  lastBlockNumber: bigint;
  /** In-flight query count */
  concurrentQueries: number;
  /** Cooldown end time (0 if not in cooldown) */
  cooldownUntil: number;
  /** Total queries this session */
  totalQueries: number;
  /** Total rate limit hits */
  limitHits: number;
}

/**
 * Rate limit check result
 */
export interface RateLimitCheckResult {
  allowed: boolean;
  reason?: 'per_second' | 'per_minute' | 'per_block' | 'concurrent' | 'cooldown';
  /** Current usage */
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
  /** Remaining queries in current window */
  remaining: {
    perSecond: number;
    perMinute: number;
    perBlock: number;
  };
  /** If in cooldown, time remaining */
  cooldownRemainingMs: number;
}

/**
 * Rate limit info for response metadata
 */
export interface RateLimitInfo {
  tier: RateLimitTier;
  remaining: number;  // Minimum of all windows
  resetAt: number;    // Timestamp when limit resets
  inCooldown: boolean;
}
```

**Default Configuration**:
```typescript
export const DEFAULT_QUERY_RATE_LIMIT_CONFIG: QueryRateLimitConfig = {
  enabled: true,
  tiers: {
    free: {
      maxQueriesPerSecond: 2,
      maxQueriesPerMinute: 20,
      maxQueriesPerBlock: 3,
      cooldownMs: 30000,      // 30 seconds
      maxConcurrentQueries: 1,
    },
    basic: {
      maxQueriesPerSecond: 5,
      maxQueriesPerMinute: 100,
      maxQueriesPerBlock: 8,
      cooldownMs: 10000,      // 10 seconds
      maxConcurrentQueries: 3,
    },
    premium: {
      maxQueriesPerSecond: 20,
      maxQueriesPerMinute: 500,
      maxQueriesPerBlock: 30,
      cooldownMs: 5000,       // 5 seconds
      maxConcurrentQueries: 10,
    },
    validator: {
      maxQueriesPerSecond: 100,
      maxQueriesPerMinute: 2000,
      maxQueriesPerBlock: 150,
      cooldownMs: 1000,       // 1 second
      maxConcurrentQueries: 50,
    },
    unlimited: {
      maxQueriesPerSecond: Infinity,
      maxQueriesPerMinute: Infinity,
      maxQueriesPerBlock: Infinity,
      cooldownMs: 0,
      maxConcurrentQueries: Infinity,
    },
  },
  windowSizeMs: 60000,        // 1 minute sliding window
  gracePeriodMs: 5000,        // 5 second grace for new sessions
  countCachedQueries: false,  // Cached queries are free
};
```

### Phase 2: Rate Limiter Implementation

**File**: `core/knowledge/query/rate_limiter.ts`

**Core Class**: `QueryRateLimiter`

```typescript
export class QueryRateLimiter {
  private config: QueryRateLimitConfig;
  private sessionStates: Map<string, SessionRateLimitState> = new Map();

  constructor(config: QueryRateLimitConfig = DEFAULT_QUERY_RATE_LIMIT_CONFIG);

  /**
   * Check if a query is allowed under rate limits.
   */
  checkRateLimit(
    sessionId: string,
    agentDid: string,
    tier: RateLimitTier,
    currentBlock: bigint
  ): RateLimitCheckResult;

  /**
   * Record a query execution (call after successful query).
   */
  recordQuery(sessionId: string, cached: boolean): void;

  /**
   * Record query completion (for concurrent tracking).
   */
  recordQueryComplete(sessionId: string): void;

  /**
   * Get rate limit info for response metadata.
   */
  getRateLimitInfo(sessionId: string): RateLimitInfo | undefined;

  /**
   * Get session rate limit state.
   */
  getSessionState(sessionId: string): SessionRateLimitState | undefined;

  /**
   * Clear session state (on session end).
   */
  clearSession(sessionId: string): void;

  /**
   * Cleanup stale sessions.
   */
  cleanupStaleSessions(maxAgeMs: number): number;
}
```

**Algorithm**: Sliding Window with Multiple Time Scales

```typescript
checkRateLimit(sessionId, agentDid, tier, currentBlock): RateLimitCheckResult {
  const now = Date.now();
  const state = this.getOrCreateState(sessionId, agentDid, tier);
  const limits = this.config.tiers[tier];

  // 1. Check cooldown first
  if (state.cooldownUntil > now) {
    return {
      allowed: false,
      reason: 'cooldown',
      cooldownRemainingMs: state.cooldownUntil - now,
      // ... other fields
    };
  }

  // 2. Prune old timestamps (outside window)
  state.queryTimestamps = state.queryTimestamps.filter(
    ts => now - ts < this.config.windowSizeMs
  );

  // 3. Reset block counter if new block
  if (currentBlock > state.lastBlockNumber) {
    state.currentBlockQueries = 0;
    state.lastBlockNumber = currentBlock;
  }

  // 4. Count queries in each window
  const oneSecondAgo = now - 1000;
  const queriesPerSecond = state.queryTimestamps.filter(ts => ts > oneSecondAgo).length;
  const queriesPerMinute = state.queryTimestamps.length;
  const queriesPerBlock = state.currentBlockQueries;

  // 5. Check all limits
  if (queriesPerSecond >= limits.maxQueriesPerSecond) {
    this.triggerCooldown(state, limits);
    return { allowed: false, reason: 'per_second', ... };
  }

  if (queriesPerMinute >= limits.maxQueriesPerMinute) {
    this.triggerCooldown(state, limits);
    return { allowed: false, reason: 'per_minute', ... };
  }

  if (queriesPerBlock >= limits.maxQueriesPerBlock) {
    return { allowed: false, reason: 'per_block', ... };  // No cooldown, just wait for next block
  }

  if (state.concurrentQueries >= limits.maxConcurrentQueries) {
    return { allowed: false, reason: 'concurrent', ... };
  }

  // 6. Allowed - increment concurrent counter
  state.concurrentQueries++;

  return {
    allowed: true,
    usage: { queriesPerSecond, queriesPerMinute, queriesPerBlock, concurrentQueries: state.concurrentQueries },
    limits,
    remaining: {
      perSecond: limits.maxQueriesPerSecond - queriesPerSecond - 1,
      perMinute: limits.maxQueriesPerMinute - queriesPerMinute - 1,
      perBlock: limits.maxQueriesPerBlock - queriesPerBlock - 1,
    },
    resetInMs: this.calculateResetTime(state, limits),
    cooldownRemainingMs: 0,
  };
}
```

### Phase 3: UnifiedQueryService Integration

**File to Modify**: `core/knowledge/query/unified_query_service.ts`

**Changes**:

1. Add rate limiter field and setter:
```typescript
private rateLimiter?: QueryRateLimiter;

setRateLimiter(limiter: QueryRateLimiter): void {
  this.rateLimiter = limiter;
}
```

2. Add rate limit config to `UnifiedQueryServiceConfig`:
```typescript
export interface UnifiedQueryServiceConfig {
  // ... existing fields ...
  /** Rate limit configuration */
  rateLimitConfig?: QueryRateLimitConfig;
}
```

3. Integrate into `processIntent()`:
```typescript
async processIntent(
  sessionId: string,
  intent: AgentIntent,
  messageId?: string
): Promise<IntentResponse> {
  const chatSession = this.chatSessions.get(sessionId);
  if (!chatSession) {
    throw new Error('Session not found');
  }

  // GAP-010: Check rate limit before processing
  if (this.rateLimiter) {
    const tier = this.determineRateLimitTier(chatSession);
    const rateLimitResult = this.rateLimiter.checkRateLimit(
      sessionId,
      chatSession.agentDid,
      tier,
      chatSession.currentBlock
    );

    if (!rateLimitResult.allowed) {
      throw new QueryRateLimitError(rateLimitResult);
    }
  }

  // ... existing logic ...

  // After successful execution, record the query
  if (this.rateLimiter) {
    const isCached = result.metadata.cached;
    if (!isCached || this.config.rateLimitConfig?.countCachedQueries) {
      this.rateLimiter.recordQuery(sessionId, isCached);
    }
  }

  // ... rest of method ...

  // Add rate limit info to response metadata
  if (this.rateLimiter) {
    result.metadata.rateLimit = this.rateLimiter.getRateLimitInfo(sessionId);
  }

  return result;
}
```

4. Add cleanup in `endSession()`:
```typescript
endSession(sessionId: string): void {
  // ... existing cleanup ...

  // GAP-010: Clear rate limit state
  if (this.rateLimiter) {
    this.rateLimiter.clearSession(sessionId);
  }
}
```

### Phase 4: Tier Determination

**Add to UnifiedQueryService**:
```typescript
/**
 * Determine rate limit tier for a session.
 * Based on agent stake level or explicit tier assignment.
 */
private determineRateLimitTier(session: AgentChatSession): RateLimitTier {
  // If explicitly set on session, use that
  if (session.rateLimitTier) {
    return session.rateLimitTier;
  }

  // Otherwise, derive from budget/stake (integration point)
  // This would typically come from account state
  if (!session.budgetLimit) {
    return 'free';
  }

  const budget = session.budgetLimit;
  if (budget >= 10000n * 10n ** 18n) return 'validator';   // 10,000+ LGM
  if (budget >= 1000n * 10n ** 18n) return 'premium';      // 1,000+ LGM
  if (budget >= 100n * 10n ** 18n) return 'basic';         // 100+ LGM
  return 'free';
}
```

### Phase 5: Error Type

**Add to `core/knowledge/query/types.ts`**:
```typescript
/**
 * Error thrown when rate limit is exceeded
 */
export class QueryRateLimitError extends Error {
  readonly code = 'RATE_LIMIT_EXCEEDED';
  readonly result: RateLimitCheckResult;

  constructor(result: RateLimitCheckResult) {
    const reason = result.reason || 'unknown';
    const resetIn = Math.ceil(result.resetInMs / 1000);
    const cooldown = Math.ceil(result.cooldownRemainingMs / 1000);

    let message = `Query rate limit exceeded (${reason})`;
    if (result.cooldownRemainingMs > 0) {
      message += `. In cooldown for ${cooldown}s`;
    } else {
      message += `. Resets in ${resetIn}s`;
    }

    super(message);
    this.name = 'QueryRateLimitError';
    this.result = result;
  }
}
```

### Phase 6: Response Metadata Extension

**Extend `QueryMetadata` in types.ts**:
```typescript
export interface QueryMetadata {
  // ... existing fields ...

  /** GAP-010: Rate limit info */
  rateLimit?: RateLimitInfo;
}
```

---

## Files to Create

| File | Purpose | Lines (est.) |
|------|---------|--------------|
| `core/knowledge/query/rate_limit.ts` | Types and configuration | ~150 |
| `core/knowledge/query/rate_limiter.ts` | QueryRateLimiter class | ~250 |

## Files to Modify

| File | Change |
|------|--------|
| `core/knowledge/query/unified_query_service.ts` | Add rate limiter integration |
| `core/knowledge/query/types.ts` | Add RateLimitInfo to QueryMetadata, add QueryRateLimitError |
| `core/knowledge/query/index.ts` | Export new modules |

---

## Default Tier Limits

| Tier | /sec | /min | /block | Cooldown | Concurrent |
|------|------|------|--------|----------|------------|
| **free** | 2 | 20 | 3 | 30s | 1 |
| **basic** | 5 | 100 | 8 | 10s | 3 |
| **premium** | 20 | 500 | 30 | 5s | 10 |
| **validator** | 100 | 2000 | 150 | 1s | 50 |
| **unlimited** | ∞ | ∞ | ∞ | 0 | ∞ |

---

## PP Alignment

| Dimension | Implementation |
|-----------|----------------|
| **TRUTH** | Accurate tracking with sliding windows and block awareness |
| **UNDERSTANDING** | Clear tier system aligned with existing SessionWallet presets |
| **BEAUTY** | Minimal integration, single check point in processIntent() |
| **JUSTICE** | Fair resource allocation, graduated limits by stake |
| **CREATIVITY** | Novel per-block limiting aligned with consensus rhythm |

---

## Verification

### Unit Tests
```bash
# Test rate limiter
npm test -- core/knowledge/query/rate_limiter.test.ts
```

### Test Cases
1. Free tier hits per-second limit → cooldown triggered
2. Premium tier sustained load → per-minute limit reached
3. Burst within block → per-block limit enforced
4. Cooldown expires → queries allowed again
5. Cached queries don't count (when configured)
6. Concurrent limit prevents parallel abuse
7. Tier upgrade increases limits immediately
8. Session cleanup removes state

### Integration Test
1. Create session with free tier
2. Send 3 queries rapidly → should succeed
3. Send 4th query → should fail with per-second limit
4. Wait 30s → cooldown expires
5. Send query → should succeed

---

## Implementation Order

1. `core/knowledge/query/rate_limit.ts` - Types and config
2. `core/knowledge/query/rate_limiter.ts` - Core logic
3. `core/knowledge/query/types.ts` - Add QueryRateLimitError
4. `core/knowledge/query/unified_query_service.ts` - Integration
5. `core/knowledge/query/index.ts` - Exports
6. Tests and refinement

---

## Estimated Effort

~3-4 hours implementation (foundation is strong, clear integration point)
