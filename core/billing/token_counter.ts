/**
 * Token Counter
 *
 * GAP-007: Estimates token count for billing purposes.
 * Uses a simple approximation (4 chars ≈ 1 token) for efficiency.
 *
 * PP HIERARCHY:
 *   TRUTH: Approximation is transparent and predictable
 *   UNDERSTANDING: Clear, simple formula
 *   BEAUTY: Minimal implementation, no external dependencies
 *   JUSTICE: Consistent counting for fair billing
 *   CREATIVITY: Heuristic reasoning depth estimation
 */

// ============================================================================
// Configuration
// ============================================================================

/**
 * Token counting configuration
 */
export interface TokenCountConfig {
  /** Average characters per token (default: 4) */
  charsPerToken: number;
  /** Minimum token count */
  minTokens: number;
  /** Maximum token count (for safety) */
  maxTokens: number;
}

export const DEFAULT_TOKEN_COUNT_CONFIG: TokenCountConfig = {
  charsPerToken: 4,
  minTokens: 1,
  maxTokens: 100000,
};

// ============================================================================
// Token Counting
// ============================================================================

/**
 * Count tokens in text using simple approximation.
 *
 * Formula: ceil(text.length / charsPerToken)
 *
 * This is intentionally simple and predictable for billing purposes.
 * Agents can estimate costs before making requests.
 */
export function countTokens(
  text: string,
  config: TokenCountConfig = DEFAULT_TOKEN_COUNT_CONFIG,
): number {
  if (!text || text.length === 0) return 0;

  // Simple approximation: ~4 characters per token (GPT-like tokenizers)
  const estimate = Math.ceil(text.length / config.charsPerToken);

  // Clamp to bounds
  return Math.max(config.minTokens, Math.min(config.maxTokens, estimate));
}

/**
 * Count tokens in structured content (multiple components)
 */
export function countStructuredTokens(
  content: {
    userInput?: string;
    agentOutput?: string;
    factsAccessed?: string[];
    reasoning?: string;
    citations?: string[];
  },
  config: TokenCountConfig = DEFAULT_TOKEN_COUNT_CONFIG,
): number {
  let total = 0;

  if (content.userInput) {
    total += countTokens(content.userInput, config);
  }

  if (content.agentOutput) {
    total += countTokens(content.agentOutput, config);
  }

  if (content.factsAccessed) {
    for (const fact of content.factsAccessed) {
      total += countTokens(fact, config);
    }
  }

  if (content.reasoning) {
    total += countTokens(content.reasoning, config);
  }

  if (content.citations) {
    for (const citation of content.citations) {
      total += countTokens(citation, config);
    }
  }

  return Math.min(total, config.maxTokens);
}

// ============================================================================
// Reasoning Depth Estimation
// ============================================================================

/**
 * Estimate reasoning depth from content analysis.
 *
 * Depth levels:
 *   1 (surface):  Simple queries, few facts
 *   2 (working):  Multiple facts, some exploration
 *   3 (deep):     Cross-domain, dependency chains
 *   4 (expert):   Complex reasoning, cross-stratum
 *   5 (advanced): Heavy cross-stratum, extensive exploration
 *
 * Heuristic based on:
 *   - Number of facts accessed
 *   - Number of queries executed
 *   - Cross-stratum queries (most significant)
 */
export function estimateReasoningDepth(
  factsUsed: number,
  queriesExecuted: number,
  crossStratumQueries: number,
): number {
  // Start at depth 1
  let depth = 1;

  // Facts accessed increase depth
  if (factsUsed > 2) depth = 2;
  if (factsUsed > 5) depth = Math.max(depth, 2);
  if (factsUsed > 10) depth = Math.max(depth, 3);
  if (factsUsed > 20) depth = Math.max(depth, 4);
  if (factsUsed > 50) depth = Math.max(depth, 5);

  // Queries executed increase depth
  if (queriesExecuted > 3) depth = Math.max(depth, 2);
  if (queriesExecuted > 5) depth = Math.max(depth, 3);
  if (queriesExecuted > 10) depth = Math.max(depth, 4);

  // Cross-stratum queries are the strongest signal
  if (crossStratumQueries > 0) depth = Math.max(depth, 3);
  if (crossStratumQueries > 2) depth = Math.max(depth, 4);
  if (crossStratumQueries > 5) depth = 5;

  return depth;
}

/**
 * Get human-readable depth label
 */
export function getDepthLabel(depth: number): string {
  const labels = ["surface", "working", "deep", "expert", "advanced"];
  const index = Math.max(0, Math.min(4, depth - 1));
  return labels[index];
}

// ============================================================================
// Cost Estimation Helpers
// ============================================================================

/**
 * Estimate token cost before execution
 */
export function estimateTokenCost(
  estimatedTokens: number,
  costPer1kTokens: bigint,
): bigint {
  return (BigInt(estimatedTokens) * costPer1kTokens) / 1000n;
}

/**
 * Calculate complexity multiplier from tokens and depth
 */
export function calculateComplexityMultiplier(
  tokenCount: number,
  reasoningDepth: number,
  thresholds: {
    low: number;
    medium: number;
    high: number;
    veryHigh: number;
  },
  depthMultipliers: [number, number, number, number, number],
): number {
  // Token-based multiplier
  let tokenMultiplier = 1.0;

  if (tokenCount >= thresholds.veryHigh) {
    tokenMultiplier = 3.0;
  } else if (tokenCount >= thresholds.high) {
    tokenMultiplier = 2.0;
  } else if (tokenCount >= thresholds.medium) {
    tokenMultiplier = 1.5;
  } else if (tokenCount >= thresholds.low) {
    tokenMultiplier = 1.0;
  }

  // Depth-based multiplier (clamp to 1-5, then to array index 0-4)
  const depthIndex = Math.max(0, Math.min(4, reasoningDepth - 1));
  const depthMultiplier = depthMultipliers[depthIndex];

  // Combined: average of token and depth multipliers
  return (tokenMultiplier + depthMultiplier) / 2;
}

// ============================================================================
// Format Helpers
// ============================================================================

/**
 * Format token count for display
 */
export function formatTokenCount(tokens: number): string {
  if (tokens < 1000) {
    return `${tokens} tokens`;
  }
  return `${(tokens / 1000).toFixed(1)}k tokens`;
}

/**
 * Format complexity multiplier for display
 */
export function formatMultiplier(multiplier: number): string {
  return `${multiplier.toFixed(2)}×`;
}
