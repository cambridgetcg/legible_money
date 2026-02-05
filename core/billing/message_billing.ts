/**
 * Message Billing Types
 *
 * GAP-007: Per-message billing for agent chat sessions.
 * Aggregates query costs and applies complexity multipliers.
 *
 * Architecture:
 *   Message → [Query costs] + [Token cost] → Complexity Multiplier → Final Cost
 *
 * PP HIERARCHY:
 *   TRUTH: Costs reflect actual computational work
 *   UNDERSTANDING: Clear breakdown helps agents understand costs
 *   BEAUTY: Clean layering: queries → messages → sessions
 *   JUSTICE: Fair pricing based on actual resource usage
 *   CREATIVITY: Novel streaming checkpoint mechanism
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { QueryCostEstimate } from "../knowledge/query/types.js";

// ============================================================================
// Message Types
// ============================================================================

/**
 * Chat message for billing purposes
 */
export interface ChatMessage {
  /** Unique message ID */
  messageId: string;

  /** Session this message belongs to */
  sessionId: string;

  /** Agent DID */
  agentDid: DID;

  /** Message type (user input costs less than agent responses) */
  type: "user_input" | "agent_response" | "system";

  /** Token count (for complexity multiplier) */
  tokenCount?: number;

  /** Reasoning depth (1-5, affects complexity) */
  reasoningDepth?: number;

  /** Timestamp */
  timestamp: number;
}

/**
 * Message cost breakdown
 */
export interface MessageCost {
  /** Message ID */
  messageId: string;

  /** Session ID */
  sessionId: string;

  /** Base cost (sum of all queries + token cost) */
  baseCost: bigint;

  /** Complexity multiplier applied */
  complexityMultiplier: number;

  /** Total cost after multiplier */
  totalCost: bigint;

  /** Detailed breakdown */
  breakdown: {
    /** Cost from knowledge queries */
    queryCost: bigint;
    /** Number of queries executed */
    queryCount: number;
    /** Token-based cost component */
    tokenCost: bigint;
    /** Complexity premium */
    complexityPremium: bigint;
    /** Streaming checkpoint fees (if any) */
    streamingFees: bigint;
  };

  /** Individual query costs */
  queryCosts: QueryCostEstimate[];

  /** Research contribution (22%) */
  researchContribution: bigint;

  /** Knowledge rewards pool (30%) */
  knowledgeRewards: bigint;

  /** Was this message within free tier? */
  freeTier: boolean;

  /** Token count for this message */
  tokenCount: number;

  /** Reasoning depth for this message */
  reasoningDepth: number;

  /** Timestamp when completed */
  completedAt: number;
}

/**
 * Streaming checkpoint for real-time billing
 */
export interface StreamingCheckpoint {
  /** Checkpoint index */
  index: number;

  /** Tokens processed so far */
  tokensProcessed: number;

  /** Provisional cost at this checkpoint */
  provisionalCost: bigint;

  /** Timestamp */
  timestamp: number;

  /** Whether budget is still available */
  budgetAvailable: boolean;

  /** Remaining budget after this checkpoint */
  remainingBudget?: bigint;
}

/**
 * Active message being tracked
 */
export interface ActiveMessage {
  /** Message ID */
  messageId: string;

  /** Session ID */
  sessionId: string;

  /** Agent DID */
  agentDid: DID;

  /** Message type */
  type: "user_input" | "agent_response" | "system";

  /** Start timestamp */
  startedAt: number;

  /** Queries executed so far */
  queries: QueryCostEstimate[];

  /** Streaming checkpoints */
  checkpoints: StreamingCheckpoint[];

  /** Current token count */
  currentTokens: number;

  /** Current provisional cost */
  provisionalCost: bigint;

  /** Facts accessed during this message */
  factsAccessed: string[];

  /** Cross-stratum queries executed */
  crossStratumQueries: number;
}

// ============================================================================
// Configuration
// ============================================================================

/**
 * Message billing configuration
 */
export interface MessageBillingConfig {
  /** Base cost per message (even with no queries) */
  baseMessageCost: bigint;

  /** Cost per 1000 tokens */
  costPer1kTokens: bigint;

  /** Complexity multiplier thresholds */
  complexityThresholds: {
    /** Tokens for low complexity (multiplier: 1.0) */
    low: number;
    /** Tokens for medium complexity (multiplier: 1.5) */
    medium: number;
    /** Tokens for high complexity (multiplier: 2.0) */
    high: number;
    /** Tokens for very high complexity (multiplier: 3.0) */
    veryHigh: number;
  };

  /** Reasoning depth multipliers (depth 1-5) */
  depthMultipliers: [number, number, number, number, number];

  /** Streaming checkpoint interval (tokens) */
  streamingCheckpointInterval: number;

  /** Free tier: messages per session */
  freeTierMessagesPerSession: number;

  /** Free tier: max tokens per free message */
  freeTierMaxTokens: number;

  /** Research share rate (applied to all costs) */
  researchShareRate: number;

  /** Knowledge rewards rate */
  knowledgeRewardsRate: number;

  /** Streaming checkpoint fee */
  streamingCheckpointFee: bigint;
}

export const DEFAULT_MESSAGE_BILLING_CONFIG: MessageBillingConfig = {
  baseMessageCost: 50000000000000n, // 0.00005 LGM (18 decimals)
  costPer1kTokens: 10000000000000n, // 0.00001 LGM per 1k tokens

  complexityThresholds: {
    low: 500, // < 500 tokens: 1.0x
    medium: 2000, // 500-2000 tokens: 1.5x
    high: 5000, // 2000-5000 tokens: 2.0x
    veryHigh: 10000, // > 5000 tokens: 3.0x
  },

  // Depth 1 (surface) = 1.0, Depth 5 (expert) = 2.5
  depthMultipliers: [1.0, 1.25, 1.5, 2.0, 2.5],

  streamingCheckpointInterval: 500, // Checkpoint every 500 tokens

  freeTierMessagesPerSession: 5,
  freeTierMaxTokens: 1000,

  researchShareRate: 0.22, // 22% to research treasury
  knowledgeRewardsRate: 0.3, // 30% to knowledge rewards

  streamingCheckpointFee: 1000000000000n, // 0.000001 LGM per checkpoint
};

// ============================================================================
// Helper Types
// ============================================================================

/**
 * Message billing statistics for a session
 */
export interface SessionMessageStats {
  /** Total messages in session */
  messageCount: number;

  /** Total cost from messages */
  totalCost: bigint;

  /** Total tokens processed */
  totalTokens: number;

  /** Total queries executed */
  totalQueries: number;

  /** Messages within free tier */
  freeTierMessages: number;

  /** Average complexity multiplier */
  avgComplexityMultiplier: number;

  /** Total research contribution */
  totalResearchContribution: bigint;

  /** Total knowledge rewards */
  totalKnowledgeRewards: bigint;
}

/**
 * Estimate for a message before execution
 */
export interface MessageCostEstimate {
  /** Estimated base cost */
  estimatedBaseCost: bigint;

  /** Estimated total cost */
  estimatedTotalCost: bigint;

  /** Whether this would be free tier */
  wouldBeFreeTier: boolean;

  /** Estimated complexity multiplier */
  estimatedMultiplier: number;

  /** Whether session has budget for this */
  hasBudget: boolean;

  /** Remaining budget after this message */
  remainingBudget?: bigint;
}
