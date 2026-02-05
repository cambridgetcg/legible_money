/**
 * Knowledge Query Module
 *
 * Exports for the UnifiedQueryService and related components
 * that optimize knowledge queries during agent chat sessions.
 */

// Types
export {
  type AgentIntent,
  type IntentResponse,
  type FactResult,
  type DomainResult,
  type PathResult,
  type QueryMetadata,
  type AgentChatSession,
  type QueryCostEstimate,
  type QueryCostConfig,
  type UnifiedQueryServiceConfig,
  DEFAULT_QUERY_COST_CONFIG,
  DEFAULT_UNIFIED_QUERY_SERVICE_CONFIG,
  QueryRateLimitError,
} from "./types.js";

// Cost Estimator
export { QueryCostEstimator } from "./cost_estimator.js";

// Unified Query Service
export { UnifiedQueryService } from "./unified_query_service.js";

// GAP-010: Rate Limiting
export {
  type RateLimitTier,
  type TierRateLimits,
  type QueryRateLimitConfig,
  type SessionRateLimitState,
  type RateLimitCheckResult,
  type RateLimitInfo,
  type RateLimitReason,
  DEFAULT_QUERY_RATE_LIMIT_CONFIG,
  createSessionRateLimitState,
  determineTierFromBudget,
} from "./rate_limit.js";

export { QueryRateLimiter } from "./rate_limiter.js";
