/**
 * Query Cost Estimator
 *
 * Estimates and tracks costs for knowledge queries.
 * Enables agents to budget their query sessions.
 *
 * Cost Model:
 * - Base cost per query type
 * - Cost per bucket scanned (O(buckets))
 * - Cross-stratum multiplier for complex queries
 * - Free tier for basic queries
 *
 * PP HIERARCHY:
 *   TRUTH: Costs reflect actual computational work
 *   UNDERSTANDING: Clear cost breakdown for agents
 *   BEAUTY: Simple formula with predictable results
 *   JUSTICE: Fair pricing based on resource usage
 *   CREATIVITY: Novel incentive for efficient queries
 */

import { confidenceToBucket } from "../indexes/types.js";
import {
  type AgentIntent,
  type QueryCostEstimate,
  type QueryCostConfig,
  type AgentChatSession,
  DEFAULT_QUERY_COST_CONFIG,
} from "./types.js";

// ============================================================================
// Query Cost Estimator
// ============================================================================

/**
 * Estimates costs for knowledge queries.
 */
export class QueryCostEstimator {
  private config: QueryCostConfig;

  constructor(config: QueryCostConfig = DEFAULT_QUERY_COST_CONFIG) {
    this.config = { ...config };
  }

  // ============================================================================
  // Cost Estimation
  // ============================================================================

  /**
   * Estimate cost for an intent before execution.
   */
  estimateCost(
    intent: AgentIntent,
    session: AgentChatSession,
    isCached: boolean,
  ): QueryCostEstimate {
    // Cached results are free
    if (isCached) {
      return this.createFreeCostEstimate(true);
    }

    // Check free tier
    if (this.isWithinFreeTier(session, intent)) {
      return this.createFreeCostEstimate(false);
    }

    // Calculate base cost
    const baseCost = this.getBaseCost(intent.type);

    // Calculate bucket cost
    const buckets = this.estimateBucketsToScan(intent);
    const bucketCost = BigInt(buckets) * this.config.costPerBucket;

    // Calculate cross-stratum cost
    const crossStratumMultiplier = intent.crossStrata
      ? this.config.crossStratumMultiplier
      : 1n;
    const crossStratumCost = intent.crossStrata
      ? baseCost * (crossStratumMultiplier - 1n)
      : 0n;

    // Calculate materialization cost
    const materializationCost = this.estimateMaterializationCost(intent);

    // Total cost
    const totalCost =
      (baseCost + bucketCost + crossStratumCost + materializationCost) *
      crossStratumMultiplier;

    return {
      cost: totalCost,
      cached: false,
      estimatedBuckets: buckets,
      crossStratumMultiplier,
      breakdown: {
        baseCost,
        bucketCost,
        crossStratumCost,
        materializationCost,
      },
    };
  }

  /**
   * Calculate actual cost after execution.
   */
  calculateActualCost(
    intent: AgentIntent,
    session: AgentChatSession,
    bucketsScanned: number,
    factsReturned: number,
    domainsReturned: number,
    wasCached: boolean,
  ): bigint {
    // Cached results are free
    if (wasCached) {
      return 0n;
    }

    // Check free tier
    if (this.isWithinFreeTier(session, intent)) {
      return 0n;
    }

    // Base cost
    const baseCost = this.getBaseCost(intent.type);

    // Bucket cost (actual)
    const bucketCost = BigInt(bucketsScanned) * this.config.costPerBucket;

    // Materialization cost (actual)
    const factCost =
      BigInt(factsReturned) * this.config.costPerFactMaterialized;
    const domainCost =
      BigInt(domainsReturned) * this.config.costPerDomainMaterialized;
    const materializationCost = factCost + domainCost;

    // Cross-stratum multiplier
    const multiplier = intent.crossStrata
      ? this.config.crossStratumMultiplier
      : 1n;

    return (baseCost + bucketCost + materializationCost) * multiplier;
  }

  // ============================================================================
  // Bucket Estimation
  // ============================================================================

  /**
   * Estimate how many buckets will be scanned for a query.
   */
  estimateBucketsToScan(intent: AgentIntent): number {
    const bucketCount = 100; // Default bucket count
    const minConfidence = intent.minConfidence ?? 0;
    const maxConfidence = 1.0;

    const minBucket = confidenceToBucket(minConfidence, bucketCount);
    const maxBucket = confidenceToBucket(maxConfidence, bucketCount);

    // Base: buckets in confidence range
    let buckets = maxBucket - minBucket + 1;

    // Adjust for query type
    switch (intent.type) {
      case "explore":
        // Exploration may scan multiple domains
        buckets *= intent.depth ?? 2;
        break;

      case "connect":
        // Cross-stratum connection scans multiple strata
        buckets *= 2;
        break;

      case "understand":
        // Deep understanding scans more thoroughly
        if (intent.depth && intent.depth >= 3) {
          buckets *= 1.5;
        }
        break;

      case "verify":
        // Verification scans references
        buckets *= 1.2;
        break;
    }

    // Cap at reasonable maximum
    return Math.min(Math.ceil(buckets), bucketCount);
  }

  /**
   * Estimate materialization cost based on expected results.
   */
  private estimateMaterializationCost(intent: AgentIntent): bigint {
    const limit = intent.limit ?? 50;

    // Estimate facts and domains to materialize
    let estimatedFacts = Math.min(limit, 50);
    let estimatedDomains = 1;

    switch (intent.type) {
      case "explore":
        estimatedDomains = 5;
        break;
      case "connect":
        estimatedDomains = 3;
        estimatedFacts = Math.min(limit, 20);
        break;
      case "understand":
        estimatedDomains = 1;
        break;
    }

    const factCost =
      BigInt(estimatedFacts) * this.config.costPerFactMaterialized;
    const domainCost =
      BigInt(estimatedDomains) * this.config.costPerDomainMaterialized;

    return factCost + domainCost;
  }

  // ============================================================================
  // Free Tier
  // ============================================================================

  /**
   * Check if query is within free tier.
   */
  isWithinFreeTier(session: AgentChatSession, intent: AgentIntent): boolean {
    // Free queries per session
    if (session.queryCount < this.config.freeTierQueriesPerSession) {
      return true;
    }

    // Simple queries (few buckets) are free
    const buckets = this.estimateBucketsToScan(intent);
    if (buckets <= this.config.freeTierBucketsPerQuery && !intent.crossStrata) {
      return true;
    }

    return false;
  }

  /**
   * Get remaining free tier queries for session.
   */
  getRemainingFreeTierQueries(session: AgentChatSession): number {
    return Math.max(
      0,
      this.config.freeTierQueriesPerSession - session.queryCount,
    );
  }

  // ============================================================================
  // Budget Management
  // ============================================================================

  /**
   * Check if session has budget for a query.
   */
  hasBudget(session: AgentChatSession, estimatedCost: bigint): boolean {
    if (session.budgetLimit === undefined) {
      return true; // No limit set
    }

    return session.totalCost + estimatedCost <= session.budgetLimit;
  }

  /**
   * Get remaining budget for session.
   */
  getRemainingBudget(session: AgentChatSession): bigint | undefined {
    if (session.budgetLimit === undefined) {
      return undefined;
    }

    return session.budgetLimit > session.totalCost
      ? session.budgetLimit - session.totalCost
      : 0n;
  }

  // ============================================================================
  // Helpers
  // ============================================================================

  /**
   * Get base cost for query type.
   */
  private getBaseCost(intentType: string): bigint {
    const baseCosts: Record<string, bigint> = {
      understand: 100n,
      verify: 150n,
      explore: 200n,
      connect: 300n,
      contribute: 500n,
      challenge: 1000n,
    };

    return baseCosts[intentType] ?? this.config.baseQueryCost;
  }

  /**
   * Create a free cost estimate.
   */
  private createFreeCostEstimate(cached: boolean): QueryCostEstimate {
    return {
      cost: 0n,
      cached,
      estimatedBuckets: 0,
      crossStratumMultiplier: 1n,
      breakdown: {
        baseCost: 0n,
        bucketCost: 0n,
        crossStratumCost: 0n,
        materializationCost: 0n,
      },
    };
  }

  /**
   * Format cost for display.
   */
  formatCost(cost: bigint): string {
    if (cost === 0n) {
      return "Free";
    }

    // Convert to decimal (assuming 18 decimals like ETH)
    const decimals = 18n;
    const whole = cost / 10n ** decimals;
    const fraction = cost % 10n ** decimals;

    if (whole > 0n) {
      return `${whole}.${fraction.toString().padStart(4, "0").slice(0, 4)} LGM`;
    }

    // Sub-unit display
    return `${cost} wei`;
  }

  /**
   * Get cost breakdown as human-readable object.
   */
  getCostBreakdown(estimate: QueryCostEstimate): Record<string, string> {
    return {
      total: this.formatCost(estimate.cost),
      base: this.formatCost(estimate.breakdown.baseCost),
      buckets: this.formatCost(estimate.breakdown.bucketCost),
      crossStratum: this.formatCost(estimate.breakdown.crossStratumCost),
      materialization: this.formatCost(estimate.breakdown.materializationCost),
      cached: estimate.cached ? "Yes" : "No",
      bucketsToScan: String(estimate.estimatedBuckets),
    };
  }
}
