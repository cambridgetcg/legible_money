/**
 * Truth Gateway
 *
 * API token utilization and retrieval system that connects:
 * - Token consumption (payment) to truth access (knowledge)
 * - Every query funds the discovery of new truth (22% to research)
 * - Citation rewards flow back to knowledge contributors
 *
 * This is the primary interface through which agents access verified knowledge.
 * The gateway meters usage, calculates fees, and distributes rewards automatically.
 *
 * THE VALUE EQUATION:
 * ==================
 * Query Value = Base Fee + Confidence Premium + Fundamentality Premium - Freshness Discount
 *
 * WHERE:
 * - Base Fee: Minimum cost per query
 * - Confidence Premium: Higher confidence = higher value = higher cost
 * - Fundamentality Premium: More fundamental facts (cited by many) cost more
 * - Freshness Discount: Recently verified facts may have lower confidence
 *
 * FLOW:
 * ====
 * Agent Query → Gateway → Fee Calculation → Payment → Knowledge Access
 *                  ↓
 *            22% to Research Fund
 *                  ↓
 *            Bounties for IRL Verification
 *                  ↓
 *            New Verified Truths
 *                  ↓
 *            Knowledge Base Grows
 *                  ↓
 *            More Query Value ←───────────────────────────────────┘
 *
 * REVENUE SPLIT (see calculateDistribution):
 *   60% to service provider (the agent or node answering the query)
 *   30% to knowledge fund (rewards fact submitters and verifiers)
 *   10% to protocol treasury
 *   The 22% research share is applied WITHIN the knowledge fund portion.
 *
 * NOTE: This is SEPARATE from the Tree of Life revenue model (core/tree/revenue_engine.ts).
 * Tree services use a 22/39/23.4/15.6 split (evidence tax/contributors/research/treasury)
 * because tree revenue rewards project contributors (developers, reviewers).
 * Gateway revenue rewards knowledge contributors (fact submitters, verifiers).
 * Both models fund research — they differ in who gets the operational remainder.
 *
 * PP HIERARCHY:
 *   TRUTH: Every query is grounded in verified facts
 *   UNDERSTANDING: Clear pricing based on knowledge value
 *   BEAUTY: Simple query interface, complex distribution underneath
 *   JUSTICE: Contributors rewarded proportionally to citations
 *   CREATIVITY: API economics that fund truth discovery
 */

import { sha512_256Hex } from "../crypto/hashing.js";
import type { DID } from "../../core/agents/identity/did.js";
import {
  type APICallRecord,
  type PaymentDistribution,
  type APIQuote,
  type BillingParameters,
  type KnowledgeDistributionParams,
  DEFAULT_BILLING_PARAMETERS,
  DEFAULT_KNOWLEDGE_DISTRIBUTION_PARAMS,
} from "./types.js";
import {
  RESEARCH_SHARE_RATE,
  RESEARCH_TREASURY_DID,
} from "../research/types.js";
import {
  calculateResearchShare,
  calculateRecipientAmount,
} from "./reward_router.js";

// ============ Query Types ============

/**
 * Knowledge query request
 */
export interface KnowledgeQuery {
  /** Unique query ID */
  queryId: string;

  /** Querying agent */
  caller: DID;

  /** Query type */
  queryType:
    | "fact_lookup"
    | "domain_search"
    | "citation_graph"
    | "confidence_check";

  /** Query parameters */
  params: QueryParams;

  /** Maximum price willing to pay */
  maxPrice?: bigint;

  /** Required minimum confidence */
  minConfidence?: number;
}

export type QueryParams =
  | FactLookupParams
  | DomainSearchParams
  | CitationGraphParams
  | ConfidenceCheckParams;

export interface FactLookupParams {
  type: "fact_lookup";
  factId: string;
}

export interface DomainSearchParams {
  type: "domain_search";
  domain: string;
  keywords?: string[];
  limit?: number;
}

export interface CitationGraphParams {
  type: "citation_graph";
  factId: string;
  depth?: number;
  direction?: "ancestors" | "descendants" | "both";
}

export interface ConfidenceCheckParams {
  type: "confidence_check";
  claim: string;
  domain?: string;
}

/**
 * Knowledge query result
 */
export interface KnowledgeResult {
  /** Query ID */
  queryId: string;

  /** Success status */
  success: boolean;

  /** Error message if failed */
  error?: string;

  /** Facts returned */
  facts: FactResult[];

  /** Total cost of query */
  cost: bigint;

  /** Payment distribution breakdown */
  distribution: PaymentDistribution;

  /** Aggregate confidence of results */
  aggregateConfidence: number;

  /** Response latency (ms) */
  latencyMs: number;
}

/**
 * Individual fact result
 */
export interface FactResult {
  id: string;
  claim: string;
  domain: string;
  confidence: number;
  fundamentalityScore: number;
  submitter: DID;
  verifiedAtBlock: bigint;
  citationCount: number;
  lastCitedBlock?: bigint;
  references?: string[];
}

// ============ Pricing Engine ============

/**
 * Calculate query price
 */
export interface PriceCalculation {
  baseCost: bigint;
  confidencePremium: bigint;
  fundamentalityPremium: bigint;
  freshnessDiscount: bigint;
  totalCost: bigint;
  breakdown: PriceBreakdown;
}

export interface PriceBreakdown {
  perFactCosts: Array<{
    factId: string;
    cost: bigint;
    confidenceMultiplier: number;
    fundamentalityMultiplier: number;
    freshnessDiscount: number;
  }>;
}

/**
 * Calculate price for a set of facts
 */
export function calculateQueryPrice(
  facts: FactResult[],
  currentBlock: bigint,
  params: BillingParameters = DEFAULT_BILLING_PARAMETERS,
): PriceCalculation {
  let totalBase = 0n;
  let totalConfidencePremium = 0n;
  let totalFundamentalityPremium = 0n;
  let totalFreshnessDiscount = 0n;
  const perFactCosts: PriceBreakdown["perFactCosts"] = [];

  for (const fact of facts) {
    // Base cost
    const baseCost = params.basePricePerCall;

    // Confidence premium (higher confidence = more valuable)
    // Premium = base × (confidence - 0.5) × rate
    // Only apply premium above 0.5 confidence
    const confidenceMultiplier =
      fact.confidence > 0.5
        ? 1 + (fact.confidence - 0.5) * params.confidencePremiumRate
        : 1;
    const confidencePremium =
      (baseCost * BigInt(Math.floor((confidenceMultiplier - 1) * 1000))) /
      1000n;

    // Fundamentality premium (more cited = more fundamental = more valuable)
    // Premium based on citation count (log scale)
    const fundamentalityMultiplier =
      1 +
      (Math.log(fact.citationCount + 1) * params.fundamentalityPremiumRate) /
        10;
    const fundamentalityPremium =
      (baseCost * BigInt(Math.floor((fundamentalityMultiplier - 1) * 1000))) /
      1000n;

    // Freshness discount (recently verified = less certain)
    // Discount for facts verified in last 1000 blocks
    const blocksSinceVerification = currentBlock - fact.verifiedAtBlock;
    const freshnessDiscount =
      blocksSinceVerification < 1000n
        ? (baseCost * BigInt(Math.floor(params.freshnessDiscountRate * 1000))) /
          1000n
        : 0n;

    // Total for this fact
    const factCost =
      baseCost + confidencePremium + fundamentalityPremium - freshnessDiscount;

    // Accumulate
    totalBase += baseCost;
    totalConfidencePremium += confidencePremium;
    totalFundamentalityPremium += fundamentalityPremium;
    totalFreshnessDiscount += freshnessDiscount;

    perFactCosts.push({
      factId: fact.id,
      cost: factCost,
      confidenceMultiplier,
      fundamentalityMultiplier,
      freshnessDiscount: Number(freshnessDiscount) / Number(baseCost),
    });
  }

  const totalCost =
    totalBase +
    totalConfidencePremium +
    totalFundamentalityPremium -
    totalFreshnessDiscount;

  return {
    baseCost: totalBase,
    confidencePremium: totalConfidencePremium,
    fundamentalityPremium: totalFundamentalityPremium,
    freshnessDiscount: totalFreshnessDiscount,
    totalCost: totalCost > 0n ? totalCost : params.basePricePerCall, // Minimum floor
    breakdown: { perFactCosts },
  };
}

/**
 * Generate price quote for a query
 */
export function generateQuote(
  query: KnowledgeQuery,
  estimatedFacts: FactResult[],
  currentBlock: bigint,
  provider: DID,
  params: BillingParameters = DEFAULT_BILLING_PARAMETERS,
): APIQuote {
  const pricing = calculateQueryPrice(estimatedFacts, currentBlock, params);

  const quoteId = sha512_256Hex(
    `quote:${query.queryId}:${provider}:${currentBlock}`,
  );

  return {
    provider,
    endpoint: `/${query.queryType}`,
    estimatedCost: pricing.totalCost,
    breakdown: {
      baseCost: pricing.baseCost,
      confidencePremium: pricing.confidencePremium,
      fundamentalityPremium: pricing.fundamentalityPremium,
      freshnessDiscount: pricing.freshnessDiscount,
    },
    validUntil: Number(currentBlock) + 24, // ~1 minute at 2521ms/block
    quoteId,
  };
}

// ============ Distribution Engine ============

/**
 * Calculate payment distribution for a query
 *
 * Simplified distribution for the gateway use case (no ancestor/citation chains).
 * For full distribution with ancestor tracking, use buildPaymentDistribution() from distribution.ts.
 *
 * @see buildPaymentDistribution in distribution.ts for the full implementation
 */
export function calculateDistribution(
  totalPayment: bigint,
  facts: FactResult[],
  provider: DID,
  billingParams: BillingParameters = DEFAULT_BILLING_PARAMETERS,
  _knowledgeParams: KnowledgeDistributionParams = DEFAULT_KNOWLEDGE_DISTRIBUTION_PARAMS,
): PaymentDistribution {
  // Delegate research split to shared implementation (reward_router.ts)
  const researchShare = calculateResearchShare(
    totalPayment,
    billingParams.researchShareRate,
  );
  const afterResearch = calculateRecipientAmount(
    totalPayment,
    billingParams.researchShareRate,
  );

  // Split remaining: 60% provider, 30% knowledge, 10% protocol (bigint-safe)
  const providerBps = BigInt(
    Math.round(billingParams.apiProviderShare * 10000),
  );
  const knowledgeBps = BigInt(
    Math.round(billingParams.knowledgePoolShare * 10000),
  );
  const providerAmount = (afterResearch * providerBps) / 10000n;
  const knowledgePoolAmount = (afterResearch * knowledgeBps) / 10000n;
  const protocolAmount = afterResearch - providerAmount - knowledgePoolAmount;

  // Protocol split: 50% burn, 50% treasury (bigint-safe)
  const burnBps = BigInt(Math.round(billingParams.burnRate * 10000));
  const burnAmount = (protocolAmount * burnBps) / 10000n;
  const treasuryAmount = protocolAmount - burnAmount;

  // Distribute knowledge pool to fact contributors
  const knowledgeRewards = distributeToContributors(knowledgePoolAmount, facts);

  return {
    apiProvider: {
      did: provider,
      amount: providerAmount,
    },
    knowledgeRewards,
    protocol: {
      burned: burnAmount,
      treasury: treasuryAmount,
    },
    research: {
      amount: researchShare,
      treasury: RESEARCH_TREASURY_DID,
    },
  };
}

/**
 * Distribute knowledge pool to contributors based on usage weight
 */
function distributeToContributors(
  pool: bigint,
  facts: FactResult[],
): PaymentDistribution["knowledgeRewards"] {
  if (facts.length === 0) {
    return [];
  }

  // Weight by confidence × fundamentality
  const weights = facts.map((f) => ({
    factId: f.id,
    submitter: f.submitter,
    weight: f.confidence * (1 + Math.log(f.citationCount + 1) / 10),
  }));

  const totalWeight = weights.reduce((sum, w) => sum + w.weight, 0);

  return weights.map((w) => {
    const share = totalWeight > 0 ? w.weight / totalWeight : 1 / facts.length;
    const shareBps = BigInt(Math.round(share * 10000));
    return {
      factId: w.factId,
      submitter: w.submitter,
      share,
      amount: (pool * shareBps) / 10000n,
      reason: "direct_citation" as const,
    };
  });
}

// ============ Gateway Class ============

/**
 * Truth Gateway - main interface for knowledge access
 */
export class TruthGateway {
  private billingParams: BillingParameters;
  private knowledgeParams: KnowledgeDistributionParams;
  private provider: DID;

  // Analytics
  private totalQueries = 0n;
  private totalRevenue = 0n;
  private totalResearchContributions = 0n;

  constructor(
    provider: DID,
    billingParams: BillingParameters = DEFAULT_BILLING_PARAMETERS,
    knowledgeParams: KnowledgeDistributionParams = DEFAULT_KNOWLEDGE_DISTRIBUTION_PARAMS,
  ) {
    this.provider = provider;
    this.billingParams = billingParams;
    this.knowledgeParams = knowledgeParams;
  }

  /**
   * Get a price quote for a query
   */
  getQuote(
    query: KnowledgeQuery,
    estimatedFacts: FactResult[],
    currentBlock: bigint,
  ): APIQuote {
    return generateQuote(
      query,
      estimatedFacts,
      currentBlock,
      this.provider,
      this.billingParams,
    );
  }

  /**
   * Execute a query (called after payment confirmed)
   */
  executeQuery(
    query: KnowledgeQuery,
    facts: FactResult[],
    payment: bigint,
    currentBlock: bigint,
  ): KnowledgeResult {
    const startTime = Date.now();

    // Apply minimum confidence filter if specified
    const filteredFacts = query.minConfidence
      ? facts.filter((f) => f.confidence >= query.minConfidence!)
      : facts;

    // Calculate actual cost
    const pricing = calculateQueryPrice(
      filteredFacts,
      currentBlock,
      this.billingParams,
    );

    // Calculate distribution
    const distribution = calculateDistribution(
      payment,
      filteredFacts,
      this.provider,
      this.billingParams,
      this.knowledgeParams,
    );

    // Update analytics
    this.totalQueries += 1n;
    this.totalRevenue += payment;
    this.totalResearchContributions += distribution.research?.amount || 0n;

    const latencyMs = Date.now() - startTime;

    // Calculate aggregate confidence
    const aggregateConfidence =
      filteredFacts.length > 0
        ? filteredFacts.reduce((sum, f) => sum + f.confidence, 0) /
          filteredFacts.length
        : 0;

    return {
      queryId: query.queryId,
      success: true,
      facts: filteredFacts,
      cost: pricing.totalCost,
      distribution,
      aggregateConfidence,
      latencyMs,
    };
  }

  /**
   * Create an API call record for on-chain settlement
   */
  createCallRecord(
    query: KnowledgeQuery,
    result: KnowledgeResult,
    blockNumber: number,
  ): APICallRecord {
    return {
      callId: sha512_256Hex(`call:${query.queryId}:${Date.now()}`),
      timestamp: Date.now(),
      blockNumber,
      caller: query.caller,
      provider: this.provider,
      endpoint: `/${query.queryType}`,
      method: "GET",
      factsUsed: result.facts.map((f) => ({
        factId: f.id,
        usageWeight: 1 / result.facts.length,
        confidence: f.confidence,
      })),
      totalCost: result.cost,
      distribution: result.distribution,
      responseConfidence: result.aggregateConfidence,
      responseLatency: result.latencyMs,
    };
  }

  /**
   * Get gateway analytics
   */
  getAnalytics(): GatewayAnalytics {
    return {
      totalQueries: this.totalQueries,
      totalRevenue: this.totalRevenue,
      totalResearchContributions: this.totalResearchContributions,
      averageRevenuePerQuery:
        this.totalQueries > 0n ? this.totalRevenue / this.totalQueries : 0n,
      researchContributionRate:
        this.totalRevenue > 0n
          ? Number(this.totalResearchContributions) / Number(this.totalRevenue)
          : 0,
    };
  }
}

export interface GatewayAnalytics {
  totalQueries: bigint;
  totalRevenue: bigint;
  totalResearchContributions: bigint;
  averageRevenuePerQuery: bigint;
  researchContributionRate: number;
}

// ============ Utility Functions ============

/**
 * Check if a query is affordable
 */
export function isQueryAffordable(
  balance: bigint,
  estimatedCost: bigint,
  safetyMargin: number = 1.1,
): boolean {
  const requiredBalance =
    (estimatedCost * BigInt(Math.floor(safetyMargin * 100))) / 100n;
  return balance >= requiredBalance;
}

/**
 * Calculate research contribution from a query
 */
export function calculateResearchContribution(payment: bigint): bigint {
  const bps = BigInt(Math.round(RESEARCH_SHARE_RATE * 10000));
  return (payment * bps) / 10000n;
}

/**
 * Estimate annual research funding from query volume
 */
export function estimateAnnualResearchFunding(
  queriesPerBlock: number,
  averageFeePerQuery: bigint,
  blocksPerYear: bigint = 12_500_000n, // ~1 year at 2521ms blocks
): bigint {
  const totalQueries = BigInt(Math.floor(queriesPerBlock)) * blocksPerYear;
  const totalRevenue = totalQueries * averageFeePerQuery;
  return calculateResearchContribution(totalRevenue);
}
