/**
 * Reasoning Session Cache
 *
 * Session-aware caching for AI agent reasoning.
 * Tracks working sets, query patterns, and provides LRU caching
 * for facts and domains accessed during reasoning sessions.
 *
 * PP HIERARCHY:
 *   TRUTH: Cache grounded in actual agent access patterns
 *   UNDERSTANDING: Tracks what agent has explored for context
 *   BEAUTY: Elegant LRU caching with pattern-based prefetch
 *   JUSTICE: Cache entries scoped to session, not polluting global state
 *   CREATIVITY: Novel query pattern tracking for predictive optimization
 */

import type { DID } from "../../../core/agents/identity/did.js";
import type { FactState } from "../../../state/types.js";
import type {
  OntologicalStratum,
  OntologicalDomain,
} from "../ontology/types.js";
import {
  type ReasoningSession,
  type QueryPatternTracker,
  type SessionStats,
  type CachedFact,
  type CachedDomain,
  type ReasoningPath,
  type QueryIntentType,
  type ConfidenceTrajectory,
  type ReasoningIndexConfig,
  DEFAULT_REASONING_INDEX_CONFIG,
} from "./types.js";

// ============================================================================
// Simple LRU Cache Implementation
// ============================================================================

/**
 * Simple LRU cache using Map (insertion order).
 */
export class LRUCache<K, V> {
  private cache: Map<K, V>;
  private readonly maxSize: number;

  constructor(maxSize: number) {
    this.cache = new Map();
    this.maxSize = maxSize;
  }

  get(key: K): V | undefined {
    const value = this.cache.get(key);
    if (value !== undefined) {
      // Move to end (most recently used)
      this.cache.delete(key);
      this.cache.set(key, value);
    }
    return value;
  }

  set(key: K, value: V): void {
    // Delete first if exists (to update position)
    if (this.cache.has(key)) {
      this.cache.delete(key);
    }
    // Evict oldest if at capacity
    while (this.cache.size >= this.maxSize) {
      const firstKey = this.cache.keys().next().value;
      if (firstKey !== undefined) {
        this.cache.delete(firstKey);
      } else {
        break;
      }
    }
    this.cache.set(key, value);
  }

  has(key: K): boolean {
    return this.cache.has(key);
  }

  delete(key: K): boolean {
    return this.cache.delete(key);
  }

  clear(): void {
    this.cache.clear();
  }

  get size(): number {
    return this.cache.size;
  }

  keys(): IterableIterator<K> {
    return this.cache.keys();
  }

  values(): IterableIterator<V> {
    return this.cache.values();
  }

  entries(): IterableIterator<[K, V]> {
    return this.cache.entries();
  }
}

// ============================================================================
// Session Factory Functions
// ============================================================================

/**
 * Create an empty query pattern tracker.
 */
export function createEmptyQueryPattern(): QueryPatternTracker {
  return {
    recentIntents: [],
    domainSequence: [],
    avgDepth: 2, // Default to 'working' depth
    directionBias: new Map(),
    stratumInterest: new Map(),
  };
}

/**
 * Create empty session statistics.
 */
export function createEmptySessionStats(): SessionStats {
  return {
    queryCount: 0,
    factAccessCount: 0,
    cacheHitRatio: 0,
    prefetchHitRatio: 0,
    avgQueryLatencyMs: 0,
    lastQueryTime: 0,
  };
}

/**
 * Create a new reasoning session.
 */
export function createReasoningSession(
  sessionId: string,
  agentDid: DID,
  currentBlock: bigint,
  _config: ReasoningIndexConfig = DEFAULT_REASONING_INDEX_CONFIG,
): ReasoningSession {
  return {
    sessionId,
    agentDid,
    workingDomains: new Set(),
    workingFacts: new Set(),
    prefetchedDomains: new Set(),
    queryPattern: createEmptyQueryPattern(),
    exploredPaths: new Map(),
    stats: createEmptySessionStats(),
    createdAtBlock: currentBlock,
    lastAccessTime: Date.now(),
  };
}

// ============================================================================
// Reasoning Session Manager
// ============================================================================

/**
 * Manages reasoning sessions for agents.
 */
export class ReasoningSessionManager {
  private sessions: Map<string, ReasoningSession> = new Map();
  private agentSessions: Map<DID, string> = new Map(); // Agent → active session
  private config: ReasoningIndexConfig;

  // Session-level caches (separate from session object for memory management)
  private factCaches: Map<string, LRUCache<string, CachedFact>> = new Map();
  private domainCaches: Map<string, LRUCache<string, CachedDomain>> = new Map();

  private cleanupTimer?: ReturnType<typeof setInterval>;

  constructor(config: ReasoningIndexConfig = DEFAULT_REASONING_INDEX_CONFIG) {
    this.config = { ...config };

    // Start cleanup timer
    this.cleanupTimer = setInterval(() => {
      this.cleanupStaleSessions();
    }, 60000); // Every minute
  }

  /**
   * Get or create session for an agent.
   */
  getOrCreateSession(agentDid: DID, currentBlock: bigint): ReasoningSession {
    // Check for existing session
    const existingSessionId = this.agentSessions.get(agentDid);
    if (existingSessionId) {
      const session = this.sessions.get(existingSessionId);
      if (session) {
        session.lastAccessTime = Date.now();
        return session;
      }
    }

    // Create new session
    const sessionId = `reasoning:${agentDid}:${Date.now()}`;
    const session = createReasoningSession(
      sessionId,
      agentDid,
      currentBlock,
      this.config,
    );

    this.sessions.set(sessionId, session);
    this.agentSessions.set(agentDid, sessionId);

    // Create caches for session
    this.factCaches.set(
      sessionId,
      new LRUCache(this.config.sessionFactCacheSize),
    );
    this.domainCaches.set(
      sessionId,
      new LRUCache(this.config.sessionDomainCacheSize),
    );

    return session;
  }

  /**
   * Get session by ID.
   */
  getSession(sessionId: string): ReasoningSession | undefined {
    const session = this.sessions.get(sessionId);
    if (session) {
      session.lastAccessTime = Date.now();
    }
    return session;
  }

  /**
   * Get fact cache for session.
   */
  getFactCache(sessionId: string): LRUCache<string, CachedFact> | undefined {
    return this.factCaches.get(sessionId);
  }

  /**
   * Get domain cache for session.
   */
  getDomainCache(
    sessionId: string,
  ): LRUCache<string, CachedDomain> | undefined {
    return this.domainCaches.get(sessionId);
  }

  /**
   * Record a domain access in session.
   */
  recordDomainAccess(session: ReasoningSession, domainId: string): void {
    session.workingDomains.add(domainId);
    session.prefetchedDomains.delete(domainId);
    session.lastAccessTime = Date.now();

    // Update domain sequence for pattern tracking
    const pattern = session.queryPattern;
    pattern.domainSequence.push(domainId);
    if (pattern.domainSequence.length > 50) {
      pattern.domainSequence.shift();
    }
  }

  /**
   * Record a fact access in session.
   */
  recordFactAccess(session: ReasoningSession, factId: string): void {
    session.workingFacts.add(factId);
    session.stats.factAccessCount++;
    session.lastAccessTime = Date.now();

    // Update cache hit stats
    const factCache = this.factCaches.get(session.sessionId);
    if (factCache) {
      const cached = factCache.get(factId);
      if (cached) {
        cached.accessCount++;
        // Update cache hit ratio
        session.stats.cacheHitRatio =
          (session.stats.cacheHitRatio * (session.stats.factAccessCount - 1) +
            1) /
          session.stats.factAccessCount;
      } else {
        // Cache miss
        session.stats.cacheHitRatio =
          (session.stats.cacheHitRatio * (session.stats.factAccessCount - 1)) /
          session.stats.factAccessCount;
      }
    }
  }

  /**
   * Record a query intent for pattern tracking.
   */
  recordQueryIntent(
    session: ReasoningSession,
    intentType: QueryIntentType,
    metadata?: {
      domain?: string;
      depth?: number;
      direction?: string;
      stratum?: OntologicalStratum;
    },
  ): void {
    const pattern = session.queryPattern;

    // Track intent
    pattern.recentIntents.push(intentType);
    if (pattern.recentIntents.length > 20) {
      pattern.recentIntents.shift();
    }

    // Update depth preference
    if (metadata?.depth !== undefined) {
      pattern.avgDepth = pattern.avgDepth * 0.9 + metadata.depth * 0.1;
    }

    // Update direction bias
    if (metadata?.direction) {
      const current = pattern.directionBias.get(metadata.direction) ?? 0;
      pattern.directionBias.set(metadata.direction, current + 1);
    }

    // Update stratum interest
    if (metadata?.stratum) {
      const current = pattern.stratumInterest.get(metadata.stratum) ?? 0;
      pattern.stratumInterest.set(metadata.stratum, current + 1);
    }

    // Update stats
    session.stats.queryCount++;
    session.stats.lastQueryTime = Date.now();
    session.lastAccessTime = Date.now();
  }

  /**
   * Cache a fact in session.
   */
  cacheFact(
    session: ReasoningSession,
    factId: string,
    fact: FactState,
    domain: OntologicalDomain,
    stratum: OntologicalStratum,
  ): void {
    const factCache = this.factCaches.get(session.sessionId);
    if (factCache) {
      factCache.set(factId, {
        fact,
        domain,
        stratum,
        accessCount: 1,
        loadedAt: Date.now(),
        prefetchedRelated: [],
      });
    }
  }

  /**
   * Cache a domain in session.
   */
  cacheDomain(
    session: ReasoningSession,
    domainId: string,
    domain: OntologicalDomain,
    factCount: number,
    trajectory: ConfidenceTrajectory,
  ): void {
    const domainCache = this.domainCaches.get(session.sessionId);
    if (domainCache) {
      domainCache.set(domainId, {
        domain,
        factCount,
        trajectory,
        loadedAt: Date.now(),
      });
    }
  }

  /**
   * Mark domains as prefetched.
   */
  markPrefetched(session: ReasoningSession, domainIds: string[]): void {
    for (const domainId of domainIds) {
      if (!session.workingDomains.has(domainId)) {
        session.prefetchedDomains.add(domainId);
      }
    }
  }

  /**
   * Record an explored reasoning path.
   */
  recordPath(session: ReasoningSession, path: ReasoningPath): void {
    const key = `${path.fromDomain}:${path.toDomain}`;
    let paths = session.exploredPaths.get(key);
    if (!paths) {
      paths = [];
      session.exploredPaths.set(key, paths);
    }
    paths.push(path);
  }

  /**
   * End a session.
   */
  endSession(sessionId: string): void {
    const session = this.sessions.get(sessionId);
    if (session) {
      this.agentSessions.delete(session.agentDid);
    }
    this.sessions.delete(sessionId);
    this.factCaches.delete(sessionId);
    this.domainCaches.delete(sessionId);
  }

  /**
   * Clean up stale sessions.
   */
  cleanupStaleSessions(): number {
    const now = Date.now();
    const timeout = this.config.sessionTimeoutMs;
    let cleaned = 0;

    for (const [sessionId, session] of this.sessions) {
      if (now - session.lastAccessTime > timeout) {
        this.endSession(sessionId);
        cleaned++;
      }
    }

    return cleaned;
  }

  /**
   * Get active session count.
   */
  getActiveSessionCount(): number {
    return this.sessions.size;
  }

  /**
   * Get session statistics.
   */
  getSessionStats(sessionId: string): SessionStats | undefined {
    return this.sessions.get(sessionId)?.stats;
  }

  /**
   * Shutdown the manager.
   */
  shutdown(): void {
    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
    }
    this.sessions.clear();
    this.agentSessions.clear();
    this.factCaches.clear();
    this.domainCaches.clear();
  }
}
