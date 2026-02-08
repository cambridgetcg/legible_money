/**
 * Knowledge Gateway Node Role
 *
 * LGM's differentiator — optimized knowledge and evidence query serving.
 * Knowledge Gateways maintain indexed copies of the knowledge tree and
 * serve structured queries for AI agent access.
 *
 * Capabilities:
 * - Topics: CLAIMS, VERIFICATIONS, EVIDENCE, EVIDENCE_REQUESTS, ATTESTATIONS
 * - Storage: knowledge + evidence + references + claims + research namespaces
 * - Sync: state sync (knowledge subtree only)
 * - DHT: server
 * - Protocols: knowledge query, evidence query
 */

import { EventEmitter } from "events";
import { ROLE_PROTOCOL_IDS } from "./types.js";
import type { DID } from "../../core/agents/identity/did.js";

// ============ Constants ============

/** Maximum results per query */
const MAX_QUERY_RESULTS = 100;

/** Default query timeout (ms) */
const DEFAULT_QUERY_TIMEOUT = 5_000;

/** Maximum batch query size */
const MAX_BATCH_SIZE = 50;

/** Cache TTL for hot facts (ms) */
const CACHE_TTL = 60_000; // 1 minute

/** Maximum cache entries */
const MAX_CACHE_SIZE = 10_000;

/** Rate limit per DID (queries/minute) */
const RATE_LIMIT_PER_DID = 60;

// ============ Types ============

export type KnowledgeQueryType =
  | "fact_by_id"
  | "facts_by_category"
  | "facts_by_confidence"
  | "evidence_by_id"
  | "evidence_by_source"
  | "evidence_by_type"
  | "confidence_history"
  | "fact_references";

export interface KnowledgeQuery {
  /** Query type */
  type: KnowledgeQueryType;

  /** Query parameters */
  params: Record<string, unknown>;

  /** Requesting agent's DID (for rate limiting) */
  requesterDid?: DID;

  /** Maximum results */
  limit?: number;

  /** Offset for pagination */
  offset?: number;
}

export interface KnowledgeBatchQuery {
  /** Multiple queries to execute */
  queries: KnowledgeQuery[];

  /** Requesting agent's DID */
  requesterDid?: DID;
}

export interface KnowledgeQueryResult {
  /** Whether the query succeeded */
  success: boolean;

  /** Query type that was executed */
  type: KnowledgeQueryType;

  /** Result data */
  data: unknown[];

  /** Total results available (for pagination) */
  totalCount: number;

  /** Execution time in ms */
  executionTimeMs: number;

  /** Error message if failed */
  error?: string;
}

export interface KnowledgeBatchResult {
  /** Results for each query in order */
  results: KnowledgeQueryResult[];

  /** Total execution time */
  totalExecutionTimeMs: number;
}

export interface ConfidenceHistoryEntry {
  /** Block number when confidence changed */
  blockNumber: bigint;

  /** Confidence value at this point */
  confidence: number;

  /** What caused the change */
  reason: string;
}

export interface KnowledgeGatewayConfig {
  /** Maximum results per query */
  maxResults: number;

  /** Query timeout (ms) */
  queryTimeout: number;

  /** Maximum batch size */
  maxBatchSize: number;

  /** Cache TTL (ms) */
  cacheTtl: number;

  /** Maximum cache entries */
  maxCacheSize: number;

  /** Rate limit per DID (queries/minute) */
  rateLimitPerDid: number;
}

export interface KnowledgeGatewayStats {
  /** Total queries served */
  queriesServed: number;

  /** Total batch queries served */
  batchQueriesServed: number;

  /** Cache hit rate (0-1) */
  cacheHitRate: number;

  /** Queries rate-limited */
  queriesRateLimited: number;

  /** Average query time (ms) */
  avgQueryTimeMs: number;

  /** Current cache size */
  cacheSize: number;

  /** Uptime in ms */
  uptime: number;
}

// ============ Query Cache ============

interface CacheEntry {
  key: string;
  data: unknown[];
  totalCount: number;
  cachedAt: number;
}

class QueryCache {
  private entries: Map<string, CacheEntry> = new Map();
  private ttl: number;
  private maxSize: number;

  constructor(ttl: number, maxSize: number) {
    this.ttl = ttl;
    this.maxSize = maxSize;
  }

  get(key: string): CacheEntry | null {
    const entry = this.entries.get(key);
    if (!entry) return null;

    if (Date.now() - entry.cachedAt > this.ttl) {
      this.entries.delete(key);
      return null;
    }

    return entry;
  }

  set(key: string, data: unknown[], totalCount: number): void {
    // Evict oldest if at capacity
    if (this.entries.size >= this.maxSize) {
      const oldest = this.findOldest();
      if (oldest) this.entries.delete(oldest);
    }

    this.entries.set(key, {
      key,
      data,
      totalCount,
      cachedAt: Date.now(),
    });
  }

  get size(): number {
    return this.entries.size;
  }

  clear(): void {
    this.entries.clear();
  }

  private findOldest(): string | null {
    let oldestKey: string | null = null;
    let oldestTime = Infinity;

    for (const [key, entry] of this.entries) {
      if (entry.cachedAt < oldestTime) {
        oldestTime = entry.cachedAt;
        oldestKey = key;
      }
    }

    return oldestKey;
  }
}

// ============ DID Rate Limiter ============

class DIDRateLimiter {
  private windows: Map<string, number[]> = new Map();
  private limit: number; // per minute

  constructor(limit: number) {
    this.limit = limit;
  }

  allow(did: string): boolean {
    const now = Date.now();
    const window = this.windows.get(did) ?? [];
    const recent = window.filter((t) => now - t < 60_000);
    recent.push(now);
    this.windows.set(did, recent);
    return recent.length <= this.limit;
  }

  cleanup(): void {
    const now = Date.now();
    for (const [did, window] of this.windows) {
      const recent = window.filter((t) => now - t < 60_000);
      if (recent.length === 0) {
        this.windows.delete(did);
      } else {
        this.windows.set(did, recent);
      }
    }
  }
}

// ============ Knowledge Gateway Service ============

/**
 * Knowledge Gateway service for indexed knowledge queries.
 *
 * This is LGM's killer feature — providing structured access to the
 * knowledge tree for AI agents. The gateway maintains indexes and
 * caches for fast query resolution.
 */
export class KnowledgeGatewayService extends EventEmitter {
  private config: KnowledgeGatewayConfig;
  private cache: QueryCache;
  private rateLimiter: DIDRateLimiter;
  private cleanupTimer: ReturnType<typeof setInterval> | null = null;
  private startedAt: number = 0;

  // Stats tracking
  private totalQueries: number = 0;
  private totalBatchQueries: number = 0;
  private cacheHits: number = 0;
  private cacheMisses: number = 0;
  private rateLimited: number = 0;
  private queryTimes: number[] = [];

  /**
   * Query handler — set by the node orchestrator to connect
   * the gateway to the actual state layer.
   */
  private queryHandler:
    | ((query: KnowledgeQuery) => Promise<KnowledgeQueryResult>)
    | null = null;

  constructor(config?: Partial<KnowledgeGatewayConfig>) {
    super();
    this.config = {
      maxResults: MAX_QUERY_RESULTS,
      queryTimeout: DEFAULT_QUERY_TIMEOUT,
      maxBatchSize: MAX_BATCH_SIZE,
      cacheTtl: CACHE_TTL,
      maxCacheSize: MAX_CACHE_SIZE,
      rateLimitPerDid: RATE_LIMIT_PER_DID,
      ...config,
    };
    this.cache = new QueryCache(this.config.cacheTtl, this.config.maxCacheSize);
    this.rateLimiter = new DIDRateLimiter(this.config.rateLimitPerDid);
  }

  /**
   * Set the query handler that connects to the state layer.
   */
  setQueryHandler(
    handler: (query: KnowledgeQuery) => Promise<KnowledgeQueryResult>,
  ): void {
    this.queryHandler = handler;
  }

  /**
   * Start the gateway service.
   */
  start(): void {
    this.startedAt = Date.now();

    // Periodic cleanup of rate limiter windows
    this.cleanupTimer = setInterval(() => this.rateLimiter.cleanup(), 30_000);

    this.emit("started");
  }

  /**
   * Stop the gateway service.
   */
  stop(): void {
    if (this.cleanupTimer) {
      clearInterval(this.cleanupTimer);
      this.cleanupTimer = null;
    }
    this.cache.clear();
    this.emit("stopped");
  }

  /**
   * Execute a single knowledge query.
   */
  async executeQuery(query: KnowledgeQuery): Promise<KnowledgeQueryResult> {
    const startTime = Date.now();

    // Rate limit check
    if (query.requesterDid && !this.rateLimiter.allow(query.requesterDid)) {
      this.rateLimited++;
      return {
        success: false,
        type: query.type,
        data: [],
        totalCount: 0,
        executionTimeMs: Date.now() - startTime,
        error: "Rate limit exceeded",
      };
    }

    // Enforce result limit
    const limit = Math.min(
      query.limit ?? this.config.maxResults,
      this.config.maxResults,
    );
    query = { ...query, limit };

    // Check cache
    const cacheKey = this.buildCacheKey(query);
    const cached = this.cache.get(cacheKey);
    if (cached) {
      this.cacheHits++;
      this.totalQueries++;
      const execTime = Date.now() - startTime;
      this.queryTimes.push(execTime);
      return {
        success: true,
        type: query.type,
        data: cached.data,
        totalCount: cached.totalCount,
        executionTimeMs: execTime,
      };
    }
    this.cacheMisses++;

    // Execute query
    if (!this.queryHandler) {
      return {
        success: false,
        type: query.type,
        data: [],
        totalCount: 0,
        executionTimeMs: Date.now() - startTime,
        error: "Query handler not configured",
      };
    }

    try {
      const result = await Promise.race([
        this.queryHandler(query),
        this.createTimeout(this.config.queryTimeout),
      ]);

      // Cache successful results
      if (result.success) {
        this.cache.set(cacheKey, result.data, result.totalCount);
      }

      this.totalQueries++;
      const execTime = Date.now() - startTime;
      this.queryTimes.push(execTime);
      result.executionTimeMs = execTime;

      return result;
    } catch (error) {
      this.totalQueries++;
      return {
        success: false,
        type: query.type,
        data: [],
        totalCount: 0,
        executionTimeMs: Date.now() - startTime,
        error: error instanceof Error ? error.message : "Query failed",
      };
    }
  }

  /**
   * Execute a batch of knowledge queries.
   */
  async executeBatchQuery(
    batch: KnowledgeBatchQuery,
  ): Promise<KnowledgeBatchResult> {
    const startTime = Date.now();

    // Enforce batch size limit
    if (batch.queries.length > this.config.maxBatchSize) {
      return {
        results: [
          {
            success: false,
            type: "fact_by_id",
            data: [],
            totalCount: 0,
            executionTimeMs: 0,
            error: `Batch too large: ${batch.queries.length} > ${this.config.maxBatchSize}`,
          },
        ],
        totalExecutionTimeMs: Date.now() - startTime,
      };
    }

    // Execute all queries
    const results = await Promise.all(
      batch.queries.map((query) =>
        this.executeQuery({
          ...query,
          requesterDid: query.requesterDid ?? batch.requesterDid,
        }),
      ),
    );

    this.totalBatchQueries++;

    return {
      results,
      totalExecutionTimeMs: Date.now() - startTime,
    };
  }

  /**
   * Get gateway statistics.
   */
  getStats(): KnowledgeGatewayStats {
    const totalCacheAttempts = this.cacheHits + this.cacheMisses;
    const avgTime =
      this.queryTimes.length > 0
        ? this.queryTimes.reduce((a, b) => a + b, 0) / this.queryTimes.length
        : 0;

    return {
      queriesServed: this.totalQueries,
      batchQueriesServed: this.totalBatchQueries,
      cacheHitRate:
        totalCacheAttempts > 0 ? this.cacheHits / totalCacheAttempts : 0,
      queriesRateLimited: this.rateLimited,
      avgQueryTimeMs: Math.round(avgTime * 100) / 100,
      cacheSize: this.cache.size,
      uptime: this.startedAt > 0 ? Date.now() - this.startedAt : 0,
    };
  }

  /**
   * Get the protocol ID for knowledge queries.
   */
  static get protocolId(): string {
    return ROLE_PROTOCOL_IDS.KNOWLEDGE_QUERY;
  }

  // ============ Internal ============

  private buildCacheKey(query: KnowledgeQuery): string {
    return `${query.type}:${JSON.stringify(query.params)}:${query.limit ?? ""}:${query.offset ?? ""}`;
  }

  private createTimeout(ms: number): Promise<never> {
    return new Promise((_, reject) => {
      setTimeout(() => reject(new Error("Query timeout")), ms);
    });
  }
}
