/**
 * Query Service Types
 *
 * Types for the UnifiedQueryService that orchestrates
 * knowledge queries during agent chat sessions.
 *
 * PP HIERARCHY:
 *   TRUTH: Types grounded in actual query patterns
 *   UNDERSTANDING: Clear interfaces for query operations
 *   BEAUTY: Elegant type hierarchy matching domain model
 *   JUSTICE: Placed alongside existing knowledge types
 *   CREATIVITY: Novel cost estimation for query budgeting
 */

import type { DID } from "../../../core/agents/identity/did.js";
import type { FactState, DecayInfo } from "../../../state/types.js";
import type {
  OntologicalStratum,
  OntologicalDomain,
} from "../ontology/types.js";
import type {
  ConfidenceTrajectory,
  QueryIntentType,
} from "../indexes/types.js";
// GAP-010: Rate limiting types
import type {
  RateLimitInfo,
  RateLimitTier,
  RateLimitCheckResult,
} from "./rate_limit.js";

// ============================================================================
// Agent Intent Types
// ============================================================================

/**
 * An agent's query intent during a chat session.
 */
export interface AgentIntent {
  /** Intent type (understand, verify, explore, connect, contribute, challenge) */
  type: QueryIntentType;
  /** Target domain ID (optional) */
  domain?: string;
  /** Target fact ID (optional) */
  factId?: string;
  /** Target stratum (optional) */
  stratum?: OntologicalStratum;
  /** Query depth preference (1=quick, 2=working, 3=deep) */
  depth?: number;
  /** Direction for exploration (causes, effects, siblings, ancestors) */
  direction?: "causes" | "effects" | "siblings" | "ancestors" | "descendants";
  /** Cross-stratum query flag */
  crossStrata?: boolean;
  /** Minimum confidence threshold */
  minConfidence?: number;
  /** Maximum results */
  limit?: number;
  /** Free-form query text (for semantic search) */
  query?: string;
}

/**
 * Response to an agent intent query.
 */
export interface IntentResponse {
  /** Query type that was executed */
  intentType: QueryIntentType;
  /** Facts returned */
  facts: FactResult[];
  /** Domains returned (for explore intents) */
  domains: DomainResult[];
  /** Cross-stratum paths (for connect intents) */
  paths?: PathResult[];
  /** Domain trajectory (for understand intents) */
  trajectory?: ConfidenceTrajectory;
  /** Query metadata */
  metadata: QueryMetadata;
}

/**
 * A fact result with session-specific context.
 */
export interface FactResult {
  /** Fact ID */
  factId: string;
  /** Fact state */
  fact: FactState;
  /** Domain the fact belongs to */
  domain: OntologicalDomain;
  /** Stratum of the fact */
  stratum: OntologicalStratum;
  /** Relevance score (0-1) for this query */
  relevance: number;
  /** Whether this was from cache */
  cached: boolean;
  /** Whether this was prefetched */
  prefetched: boolean;
  /** GAP-008: Effective confidence after time decay */
  effectiveConfidence: number;
  /** GAP-008: Detailed decay information (optional) */
  decayInfo?: DecayInfo;
}

/**
 * A domain result with health metrics.
 */
export interface DomainResult {
  /** Domain ID */
  domainId: string;
  /** Domain data */
  domain: OntologicalDomain;
  /** Fact count in domain */
  factCount: number;
  /** Domain health trajectory */
  trajectory: ConfidenceTrajectory;
  /** Whether this is a gateway domain */
  isGateway: boolean;
  /** Gateway score if applicable */
  gatewayScore?: number;
}

/**
 * A cross-stratum reasoning path.
 */
export interface PathResult {
  /** Source domain */
  fromDomain: string;
  /** Target domain */
  toDomain: string;
  /** Intermediate gateway domains */
  gateways: string[];
  /** Facts forming this path */
  facts: string[];
  /** Path strength (product of dependency strengths) */
  strength: number;
  /** Strata traversed */
  strataTraversed: OntologicalStratum[];
}

/**
 * Query execution metadata.
 */
export interface QueryMetadata {
  /** Whether result came from cache */
  cached: boolean;
  /** Number of buckets scanned */
  bucketsScanned: number;
  /** Domains searched */
  domainsSearched: string[];
  /** Query latency in ms */
  latencyMs: number;
  /** Estimated cost (if applicable) */
  estimatedCost?: bigint;
  /** Actual cost charged */
  actualCost?: bigint;
  /** Session cache hit ratio */
  cacheHitRatio: number;
  /** Prefetch hit ratio */
  prefetchHitRatio: number;
  /** GAP-010: Rate limit info */
  rateLimit?: RateLimitInfo;
}

// ============================================================================
// Session Types
// ============================================================================

/**
 * An agent's chat session with knowledge queries.
 */
export interface AgentChatSession {
  /** Unique session ID */
  sessionId: string;
  /** Agent DID */
  agentDid: DID;
  /** Session start block */
  startBlock: bigint;
  /** Session start timestamp */
  startedAt: number;
  /** Last activity timestamp */
  lastActivityAt: number;
  /** Current block number */
  currentBlock: bigint;
  /** Total queries in session */
  queryCount: number;
  /** Total cost incurred */
  totalCost: bigint;
  /** Budget limit (if set) */
  budgetLimit?: bigint;
  /** GAP-010: Explicit rate limit tier (if set, overrides budget-based) */
  rateLimitTier?: RateLimitTier;
}

// ============================================================================
// Cost Estimation Types
// ============================================================================

/**
 * Cost estimation for a query.
 */
export interface QueryCostEstimate {
  /** Estimated cost in base units */
  cost: bigint;
  /** Whether result is cached (free) */
  cached: boolean;
  /** Estimated buckets to scan */
  estimatedBuckets: number;
  /** Cross-stratum multiplier applied */
  crossStratumMultiplier: bigint;
  /** Breakdown by component */
  breakdown: {
    /** Base cost for query type */
    baseCost: bigint;
    /** Cost for bucket scanning */
    bucketCost: bigint;
    /** Cost for cross-stratum traversal */
    crossStratumCost: bigint;
    /** Cost for result materialization */
    materializationCost: bigint;
  };
}

/**
 * Cost configuration for queries.
 */
export interface QueryCostConfig {
  /** Base cost per query */
  baseQueryCost: bigint;
  /** Cost per bucket scanned */
  costPerBucket: bigint;
  /** Multiplier for cross-stratum queries */
  crossStratumMultiplier: bigint;
  /** Cost per fact materialized */
  costPerFactMaterialized: bigint;
  /** Cost per domain materialized */
  costPerDomainMaterialized: bigint;
  /** Free tier: queries per session */
  freeTierQueriesPerSession: number;
  /** Free tier: buckets per query */
  freeTierBucketsPerQuery: number;
}

export const DEFAULT_QUERY_COST_CONFIG: QueryCostConfig = {
  baseQueryCost: 100n,
  costPerBucket: 10n,
  crossStratumMultiplier: 2n,
  costPerFactMaterialized: 5n,
  costPerDomainMaterialized: 20n,
  freeTierQueriesPerSession: 10,
  freeTierBucketsPerQuery: 10,
};

// ============================================================================
// Service Configuration
// ============================================================================

/**
 * Configuration for the UnifiedQueryService.
 */
export interface UnifiedQueryServiceConfig {
  /** Cost configuration */
  costConfig: QueryCostConfig;
  /** Maximum results per query */
  maxResultsPerQuery: number;
  /** Session timeout in ms */
  sessionTimeoutMs: number;
  /** Enable prefetching */
  enablePrefetch: boolean;
  /** Prefetch concurrency */
  prefetchConcurrency: number;
  /** Cache TTL in ms */
  cacheTtlMs: number;
}

export const DEFAULT_UNIFIED_QUERY_SERVICE_CONFIG: UnifiedQueryServiceConfig = {
  costConfig: DEFAULT_QUERY_COST_CONFIG,
  maxResultsPerQuery: 100,
  sessionTimeoutMs: 3600000, // 1 hour
  enablePrefetch: true,
  prefetchConcurrency: 3,
  cacheTtlMs: 60000, // 1 minute
};

// ============================================================================
// GAP-010: Rate Limit Error
// ============================================================================

/**
 * Error thrown when query rate limit is exceeded.
 *
 * Includes full rate limit check result for client handling.
 */
export class QueryRateLimitError extends Error {
  readonly code = "RATE_LIMIT_EXCEEDED";
  readonly result: RateLimitCheckResult;

  constructor(result: RateLimitCheckResult) {
    const reason = result.reason || "unknown";
    const resetIn = Math.ceil(result.resetInMs / 1000);
    const cooldown = Math.ceil(result.cooldownRemainingMs / 1000);

    let message = `Query rate limit exceeded (${reason})`;
    if (result.cooldownRemainingMs > 0) {
      message += `. In cooldown for ${cooldown}s`;
    } else if (result.resetInMs > 0) {
      message += `. Resets in ${resetIn}s`;
    }

    super(message);
    this.name = "QueryRateLimitError";
    this.result = result;
  }

  /**
   * Get rate limit info for error response.
   */
  getRateLimitInfo(): RateLimitInfo {
    return {
      tier: "free", // Will be overwritten by actual tier
      remaining: 0,
      resetAt: Date.now() + this.result.resetInMs,
      inCooldown: this.result.cooldownRemainingMs > 0,
    };
  }
}
