/**
 * Domain Reasoning Index Types
 *
 * Types for the domain reasoning index optimized for AI agent access.
 * Provides efficient queries for facts by domain/confidence, cross-stratum
 * dependencies, gateway domains, and session-aware caching.
 *
 * PP HIERARCHY:
 *   TRUTH: Types grounded in actual agent access patterns
 *   UNDERSTANDING: Clear interfaces for reasoning operations
 *   BEAUTY: Elegant type hierarchy matching domain model
 *   JUSTICE: Types placed in core/knowledge where they belong
 *   CREATIVITY: Novel gateway and trajectory concepts for cross-stratum reasoning
 */

import type { DID } from "../../../core/agents/identity/did.js";
import type { FactState } from "../../../state/types.js";
import type {
  OntologicalStratum,
  OntologicalDomain,
} from "../ontology/types.js";

// ============================================================================
// Confidence Bucket Index
// ============================================================================

/**
 * Configuration for confidence bucketing.
 * 100 buckets = 0.01 granularity (0-0.99, 1.00-1.00)
 */
export interface ConfidenceBucketConfig {
  /** Number of buckets (default: 100) */
  bucketCount: number;
  /** Whether to cache bucket results */
  enableCaching: boolean;
  /** Cache TTL in milliseconds */
  cacheTtlMs: number;
}

export const DEFAULT_CONFIDENCE_BUCKET_CONFIG: ConfidenceBucketConfig = {
  bucketCount: 100,
  enableCaching: true,
  cacheTtlMs: 60000, // 1 minute
};

/**
 * Convert confidence (0-1) to bucket index.
 */
export function confidenceToBucket(
  confidence: number,
  bucketCount: number = 100,
): number {
  if (confidence >= 1.0) return bucketCount - 1;
  if (confidence <= 0) return 0;
  return Math.floor(confidence * bucketCount);
}

/**
 * Get confidence range for a bucket.
 */
export function bucketToConfidenceRange(
  bucket: number,
  bucketCount: number = 100,
): [number, number] {
  const low = bucket / bucketCount;
  const high = (bucket + 1) / bucketCount;
  return [low, Math.min(high, 1.0)];
}

// ============================================================================
// Cross-Stratum Index
// ============================================================================

/**
 * A dependency between facts across strata.
 */
export interface CrossStratumDependency {
  /** Fact ID in higher stratum */
  fromFactId: string;
  /** Fact ID in lower stratum being depended upon */
  toFactId: string;
  /** Stratum of the dependent fact */
  fromStratum: OntologicalStratum;
  /** Stratum of the dependency */
  toStratum: OntologicalStratum;
  /** Strength of dependency (0-1) */
  strength: number;
  /** Type of dependency */
  dependencyType: "foundational" | "supporting" | "constraining";
}

/**
 * Result of a transitive dependency query.
 */
export interface TransitiveDependencyResult {
  /** All transitively depended-upon facts */
  dependencies: string[];
  /** Strata traversed in the dependency chain */
  strataTraversed: OntologicalStratum[];
  /** Whether max depth was reached */
  maxDepthReached: boolean;
  /** Paths through the dependency graph */
  paths: DependencyPath[];
}

/**
 * A path through the dependency graph.
 */
export interface DependencyPath {
  /** Fact IDs in order from source to deepest dependency */
  facts: string[];
  /** Cumulative strength along the path */
  cumulativeStrength: number;
  /** Strata crossed */
  strata: OntologicalStratum[];
}

// ============================================================================
// Gateway Domain Index
// ============================================================================

/**
 * A gateway domain bridges knowledge between strata.
 * High gateway score = important for cross-stratum reasoning.
 */
export interface GatewayDomainInfo {
  /** Domain ID */
  domainId: string;
  /** Domain data */
  domain: OntologicalDomain;
  /** Gateway score (0-1): how important for cross-stratum connections */
  gatewayScore: number;
  /** Domains in higher strata that depend on this domain */
  connectsTo: string[];
  /** Domains in lower strata this domain depends on */
  connectsFrom: string[];
  /** Number of cross-stratum facts passing through */
  crossStratumFactCount: number;
  /** Primary strata this gateway bridges */
  bridgesStrata: [OntologicalStratum, OntologicalStratum];
}

/**
 * Result of finding gateways between strata.
 */
export interface GatewayQueryResult {
  /** Gateway domains ordered by score */
  gateways: GatewayDomainInfo[];
  /** Whether result was from cache */
  cached: boolean;
  /** From stratum */
  fromStratum: OntologicalStratum;
  /** To stratum */
  toStratum: OntologicalStratum;
}

// ============================================================================
// Confidence Trajectory
// ============================================================================

/**
 * Tracks confidence changes in a domain over time.
 * Used to identify contested or stabilizing domains.
 */
export interface ConfidenceTrajectory {
  /** Domain ID */
  domainId: string;
  /** Average confidence of facts in domain */
  avgConfidence: number;
  /** Standard deviation of confidence values */
  confidenceStdDev: number;
  /** Recent trend: positive = gaining confidence, negative = losing */
  trend: number;
  /** Number of active challenges in this domain */
  activeChallenges: number;
  /** Historical samples: block → average confidence */
  history: Array<{ block: bigint; avgConfidence: number }>;
  /** Overall domain health score (0-1) */
  healthScore: number;
  /** When trajectory was last computed */
  computedAt: number;
}

/**
 * Compute domain health score from trajectory metrics.
 */
export function computeDomainHealthScore(
  avgConfidence: number,
  stdDev: number,
  trend: number,
  activeChallenges: number,
): number {
  // High confidence is good
  const confidenceComponent = avgConfidence * 0.4;

  // Low variance is good (stable domain)
  const stabilityComponent = Math.max(0, 1 - stdDev * 2) * 0.25;

  // Positive or neutral trend is good
  const trendComponent = Math.max(0, Math.min(1, 0.5 + trend)) * 0.2;

  // Few challenges is good
  const challengeComponent = Math.max(0, 1 - activeChallenges * 0.1) * 0.15;

  return Math.min(
    1,
    confidenceComponent +
      stabilityComponent +
      trendComponent +
      challengeComponent,
  );
}

// ============================================================================
// Reasoning Session Types
// ============================================================================

/**
 * A reasoning session tracks agent context during knowledge exploration.
 */
export interface ReasoningSession {
  /** Unique session ID */
  sessionId: string;
  /** Agent DID */
  agentDid: DID;
  /** Domains actively being reasoned about */
  workingDomains: Set<string>;
  /** Facts accessed in this session */
  workingFacts: Set<string>;
  /** Domains prefetched but not yet accessed */
  prefetchedDomains: Set<string>;
  /** Query pattern tracker for predictive prefetching */
  queryPattern: QueryPatternTracker;
  /** Cross-domain reasoning paths explored */
  exploredPaths: Map<string, ReasoningPath[]>;
  /** Session statistics */
  stats: SessionStats;
  /** Block when session started */
  createdAtBlock: bigint;
  /** Last access timestamp */
  lastAccessTime: number;
}

/**
 * Tracks agent query patterns for prefetch optimization.
 */
export interface QueryPatternTracker {
  /** Recent intent types */
  recentIntents: QueryIntentType[];
  /** Domain access sequence */
  domainSequence: string[];
  /** Average query depth preference */
  avgDepth: number;
  /** Direction bias: 'causes' vs 'effects' etc. */
  directionBias: Map<string, number>;
  /** Interest level per stratum */
  stratumInterest: Map<OntologicalStratum, number>;
}

export type QueryIntentType =
  | "understand"
  | "verify"
  | "explore"
  | "connect"
  | "contribute"
  | "challenge"
  | "search"; // GAP-004: Semantic search intent

/**
 * A reasoning path between two domains.
 */
export interface ReasoningPath {
  /** Source domain */
  fromDomain: string;
  /** Target domain */
  toDomain: string;
  /** Intermediate domains traversed */
  intermediates: string[];
  /** Facts forming this reasoning chain */
  facts: string[];
  /** Path strength (product of reference strengths) */
  strength: number;
  /** Whether path crosses strata */
  crossesStrata: boolean;
  /** Strata traversed */
  strataTraversed: OntologicalStratum[];
}

/**
 * Session statistics for monitoring and optimization.
 */
export interface SessionStats {
  /** Total queries in session */
  queryCount: number;
  /** Total facts accessed */
  factAccessCount: number;
  /** Cache hit ratio */
  cacheHitRatio: number;
  /** Prefetch hit ratio */
  prefetchHitRatio: number;
  /** Average query latency (ms) */
  avgQueryLatencyMs: number;
  /** Last query timestamp */
  lastQueryTime: number;
}

// ============================================================================
// Cached Entry Types
// ============================================================================

/**
 * Cached fact with session-specific metadata.
 */
export interface CachedFact {
  /** The fact state */
  fact: FactState;
  /** Domain the fact belongs to */
  domain: OntologicalDomain;
  /** Stratum of the fact */
  stratum: OntologicalStratum;
  /** Access count in this session */
  accessCount: number;
  /** When loaded into cache */
  loadedAt: number;
  /** Related facts that were prefetched */
  prefetchedRelated: string[];
}

/**
 * Cached domain with session-specific metadata.
 */
export interface CachedDomain {
  /** The domain */
  domain: OntologicalDomain;
  /** Fact count in domain */
  factCount: number;
  /** Confidence trajectory */
  trajectory: ConfidenceTrajectory;
  /** When loaded into cache */
  loadedAt: number;
}

// ============================================================================
// Query Types
// ============================================================================

/**
 * Options for facts-by-domain queries.
 */
export interface FactsByDomainOptions {
  /** Minimum confidence threshold */
  minConfidence?: number;
  /** Maximum confidence threshold */
  maxConfidence?: number;
  /** Include descendant domains */
  includeDescendants?: boolean;
  /** Maximum results */
  limit?: number;
  /** Offset for pagination */
  offset?: number;
}

/**
 * Result of a facts-by-domain query.
 */
export interface FactsByDomainResult {
  /** Matching fact IDs */
  factIds: string[];
  /** Total count (for pagination) */
  totalCount: number;
  /** Domains included */
  domainsIncluded: string[];
  /** Buckets scanned */
  bucketsScanned: number;
}

/**
 * Options for dependent facts query.
 */
export interface DependentFactsOptions {
  /** Maximum traversal depth */
  maxDepth?: number;
  /** Filter by stratum */
  strataFilter?: OntologicalStratum[];
  /** Minimum dependency strength */
  minStrength?: number;
  /** Maximum results */
  limit?: number;
}

/**
 * Result of a dependent facts query.
 */
export interface DependentFactsResult {
  /** Facts that depend on the queried domain/fact */
  facts: Array<{
    factId: string;
    depth: number;
    path: string[];
    stratum: OntologicalStratum;
  }>;
  /** Total count */
  totalCount: number;
}

/**
 * Domain prediction for prefetching.
 */
export interface DomainPrediction {
  /** Domain ID */
  domainId: string;
  /** Confidence in prediction (0-1) */
  confidence: number;
  /** Reason for prediction */
  reason: "sibling" | "child" | "ancestor" | "gateway" | "pattern" | "related";
}

// ============================================================================
// Index Configuration
// ============================================================================

/**
 * Configuration for the domain reasoning index.
 */
export interface ReasoningIndexConfig {
  /** Confidence bucket configuration */
  confidenceBuckets: ConfidenceBucketConfig;
  /** Maximum depth for dependency traversal */
  maxDependencyDepth: number;
  /** Gateway score threshold (domains above this are gateways) */
  gatewayThreshold: number;
  /** Session cache sizes */
  sessionFactCacheSize: number;
  sessionDomainCacheSize: number;
  /** Trajectory history sample interval (blocks) */
  trajectorySampleInterval: bigint;
  /** Maximum trajectory history entries */
  maxTrajectoryHistory: number;
  /** Prefetch count per query */
  prefetchCount: number;
  /** Session timeout (ms) */
  sessionTimeoutMs: number;
}

export const DEFAULT_REASONING_INDEX_CONFIG: ReasoningIndexConfig = {
  confidenceBuckets: DEFAULT_CONFIDENCE_BUCKET_CONFIG,
  maxDependencyDepth: 10,
  gatewayThreshold: 0.3,
  sessionFactCacheSize: 500,
  sessionDomainCacheSize: 100,
  trajectorySampleInterval: 1000n, // ~42 minutes at 2521ms blocks
  maxTrajectoryHistory: 100,
  prefetchCount: 5,
  sessionTimeoutMs: 3600000, // 1 hour
};
