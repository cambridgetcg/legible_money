/**
 * Knowledge Indexes Module
 *
 * Exports for the domain reasoning index system optimized for
 * AI agent access during reasoning.
 *
 * Components:
 * - DomainReasoningIndex: Confidence-bucketed fact lookups, cross-stratum dependencies
 * - GatewayDomainIndex: Identifies domains that bridge strata
 * - ReasoningSessionManager: Session-aware caching with LRU
 * - PrefetchEngine: Predictive prefetching based on query patterns
 *
 * PP HIERARCHY:
 *   TRUTH: Clean exports from single source
 *   UNDERSTANDING: Clear module boundary
 *   BEAUTY: Minimal, complete interface
 *   JUSTICE: Right components exposed at right level
 *   CREATIVITY: Enables novel agent reasoning optimizations
 */

// Types
export {
  // Configuration
  type ConfidenceBucketConfig,
  DEFAULT_CONFIDENCE_BUCKET_CONFIG,
  type ReasoningIndexConfig,
  DEFAULT_REASONING_INDEX_CONFIG,

  // Utility functions
  confidenceToBucket,
  bucketToConfidenceRange,
  computeDomainHealthScore,

  // Cross-stratum types
  type CrossStratumDependency,
  type TransitiveDependencyResult,
  type DependencyPath,

  // Gateway types
  type GatewayDomainInfo,
  type GatewayQueryResult,

  // Confidence trajectory
  type ConfidenceTrajectory,

  // Session types
  type ReasoningSession,
  type QueryPatternTracker,
  type QueryIntentType,
  type ReasoningPath,
  type SessionStats,
  type CachedFact,
  type CachedDomain,

  // Query types
  type FactsByDomainOptions,
  type FactsByDomainResult,
  type DependentFactsOptions,
  type DependentFactsResult,
  type DomainPrediction,
} from "./types.js";

// Core reasoning index
export { DomainReasoningIndex } from "./reasoning_index.js";

// Gateway domain index
export { GatewayDomainIndex } from "./gateway_index.js";

// Session management
export {
  LRUCache,
  createEmptyQueryPattern,
  createEmptySessionStats,
  createReasoningSession,
  ReasoningSessionManager,
} from "./session_cache.js";

// Prefetch engine
export { PrefetchEngine } from "./prefetch_engine.js";
