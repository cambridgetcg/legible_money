/**
 * Opportunity Detector
 *
 * The intelligence that makes the Tree of Life grow autonomously.
 * Analyzes knowledge graph demand signals to identify product opportunities:
 *
 * - High query volume in a knowledge domain → API/data service opportunity
 * - Many fulfilled bounties → mature domain ready for productization
 * - Orderbook demand without supply → market gap to fill
 * - Highly-cited facts → high-value content needing tools
 * - Cross-domain insights → novel product at domain intersection
 *
 * Conservative by design: max 3 seeds per epoch, minimum confidence threshold.
 */

import { domainHash } from "../crypto/hashing.js";
import type { OpportunitySeed, DemandSignal } from "./types.js";
import {
  MAX_SEEDS_PER_EPOCH,
  MIN_SEED_CONFIDENCE,
  SEED_EXPIRY_BLOCKS,
} from "./types.js";

// ============ Opportunity Detection ============

/**
 * Domain metrics used for opportunity detection.
 * Gathered from state before calling detection functions.
 */
export interface DomainMetrics {
  /** Knowledge domain identifier */
  domain: string;

  /** Total API queries in this domain (recent epoch) */
  queryCount: bigint;

  /** Total citation count across domain facts */
  citationCount: bigint;

  /** Revenue generated from domain queries */
  queryRevenue: bigint;

  /** Number of active bounties in this domain */
  activeBounties: number;

  /** Number of fulfilled bounties in this domain */
  fulfilledBounties: number;

  /** Number of verified facts in domain */
  factCount: number;

  /** Average confidence of domain facts */
  averageConfidence: number;

  /** Growth rate of queries (current/previous epoch ratio) */
  queryGrowthRate: number;
}

/**
 * Market metrics for gap detection.
 */
export interface MarketMetrics {
  /** Resource type with unmatched demand */
  resourceType: string;

  /** Total unmatched buy orders (demand) */
  unmatchedDemand: bigint;

  /** Total unmatched sell orders (supply) */
  unmatchedSupply: bigint;

  /** Demand/supply ratio (>1 = more demand than supply) */
  demandSupplyRatio: number;
}

/**
 * Detect product opportunities from domain and market metrics.
 * Returns at most MAX_SEEDS_PER_EPOCH seeds, above MIN_SEED_CONFIDENCE.
 */
export function detectOpportunities(
  domainMetrics: DomainMetrics[],
  marketMetrics: MarketMetrics[],
  existingProjectDomains: string[],
  currentBlock: bigint,
  nonce: bigint,
): OpportunitySeed[] {
  const candidates: OpportunitySeed[] = [];

  // 1. Detect query volume opportunities
  for (const dm of domainMetrics) {
    // Skip domains that already have active projects
    if (existingProjectDomains.includes(dm.domain)) continue;

    const querySignal = detectQueryVolumeSignal(dm);
    if (querySignal && querySignal.strength >= MIN_SEED_CONFIDENCE) {
      candidates.push(
        buildSeed(
          dm.domain,
          querySignal,
          dm.queryRevenue,
          estimateEffort(dm),
          currentBlock,
          nonce + BigInt(candidates.length),
        ),
      );
    }

    // 2. Detect citation cluster opportunities
    const citationSignal = detectCitationClusterSignal(dm);
    if (citationSignal && citationSignal.strength >= MIN_SEED_CONFIDENCE) {
      if (!candidates.some((c) => c.knowledgeDomain === dm.domain)) {
        candidates.push(
          buildSeed(
            dm.domain,
            citationSignal,
            dm.queryRevenue,
            estimateEffort(dm),
            currentBlock,
            nonce + BigInt(candidates.length),
          ),
        );
      }
    }

    // 3. Detect bounty pattern opportunities (mature domains)
    const bountySignal = detectBountyPatternSignal(dm);
    if (bountySignal && bountySignal.strength >= MIN_SEED_CONFIDENCE) {
      if (!candidates.some((c) => c.knowledgeDomain === dm.domain)) {
        candidates.push(
          buildSeed(
            dm.domain,
            bountySignal,
            dm.queryRevenue,
            estimateEffort(dm),
            currentBlock,
            nonce + BigInt(candidates.length),
          ),
        );
      }
    }
  }

  // 4. Detect market gap opportunities
  for (const mm of marketMetrics) {
    if (existingProjectDomains.includes(mm.resourceType)) continue;

    const marketSignal = detectMarketGapSignal(mm);
    if (marketSignal && marketSignal.strength >= MIN_SEED_CONFIDENCE) {
      candidates.push(
        buildSeed(
          mm.resourceType,
          marketSignal,
          mm.unmatchedDemand, // Potential revenue
          50, // Default effort estimate for market gap
          currentBlock,
          nonce + BigInt(candidates.length),
        ),
      );
    }
  }

  // Sort by estimated ROI descending, take top N
  return candidates
    .sort((a, b) => b.estimatedROI - a.estimatedROI)
    .slice(0, MAX_SEEDS_PER_EPOCH);
}

// ============ Signal Detection ============

/**
 * Detect high query volume signal.
 * Threshold: >100 queries/epoch with >20% growth.
 */
function detectQueryVolumeSignal(dm: DomainMetrics): DemandSignal | null {
  const MIN_QUERIES = 100n;
  const MIN_GROWTH = 1.2; // 20% growth

  if (dm.queryCount < MIN_QUERIES) return null;
  if (dm.queryGrowthRate < MIN_GROWTH) return null;

  // Strength: combination of volume and growth
  const volumeScore = Math.min(1, Number(dm.queryCount) / 1000);
  const growthScore = Math.min(1, (dm.queryGrowthRate - 1) / 2);
  const strength = volumeScore * 0.6 + growthScore * 0.4;

  return {
    type: "query_volume",
    evidence: {
      queryCount: dm.queryCount.toString(),
      growthRate: dm.queryGrowthRate,
      revenue: dm.queryRevenue.toString(),
    },
    strength,
  };
}

/**
 * Detect citation cluster signal.
 * Threshold: >50 total citations with high average confidence.
 */
function detectCitationClusterSignal(dm: DomainMetrics): DemandSignal | null {
  const MIN_CITATIONS = 50n;
  const MIN_CONFIDENCE = 0.7;

  if (dm.citationCount < MIN_CITATIONS) return null;
  if (dm.averageConfidence < MIN_CONFIDENCE) return null;

  const citationScore = Math.min(1, Number(dm.citationCount) / 500);
  const confidenceScore = (dm.averageConfidence - 0.7) / 0.3; // 0.7-1.0 → 0-1
  const strength = citationScore * 0.5 + confidenceScore * 0.5;

  return {
    type: "citation_cluster",
    evidence: {
      citationCount: dm.citationCount.toString(),
      averageConfidence: dm.averageConfidence,
      factCount: dm.factCount,
    },
    strength,
  };
}

/**
 * Detect bounty pattern signal (mature domain with fulfilled bounties).
 * Threshold: >10 fulfilled bounties with >50% fulfillment rate.
 */
function detectBountyPatternSignal(dm: DomainMetrics): DemandSignal | null {
  const MIN_FULFILLED = 10;
  const MIN_RATE = 0.5;

  if (dm.fulfilledBounties < MIN_FULFILLED) return null;

  const totalBounties = dm.activeBounties + dm.fulfilledBounties;
  if (totalBounties === 0) return null;

  const fulfillmentRate = dm.fulfilledBounties / totalBounties;
  if (fulfillmentRate < MIN_RATE) return null;

  const maturityScore = Math.min(1, dm.fulfilledBounties / 50);
  const rateScore = (fulfillmentRate - 0.5) / 0.5; // 0.5-1.0 → 0-1
  const strength = maturityScore * 0.6 + rateScore * 0.4;

  return {
    type: "bounty_pattern",
    evidence: {
      fulfilledBounties: dm.fulfilledBounties,
      activeBounties: dm.activeBounties,
      fulfillmentRate,
    },
    strength,
  };
}

/**
 * Detect market gap signal (demand exceeds supply).
 * Threshold: demand/supply ratio > 2.0 with meaningful volume.
 */
function detectMarketGapSignal(mm: MarketMetrics): DemandSignal | null {
  const MIN_RATIO = 2.0;
  const MIN_DEMAND = 1000n; // Minimum meaningful demand

  if (mm.demandSupplyRatio < MIN_RATIO) return null;
  if (mm.unmatchedDemand < MIN_DEMAND) return null;

  const ratioScore = Math.min(1, (mm.demandSupplyRatio - 2) / 8); // 2-10 → 0-1
  const volumeScore = Math.min(1, Number(mm.unmatchedDemand) / 100000);
  const strength = ratioScore * 0.5 + volumeScore * 0.5;

  return {
    type: "market_gap",
    evidence: {
      resourceType: mm.resourceType,
      unmatchedDemand: mm.unmatchedDemand.toString(),
      unmatchedSupply: mm.unmatchedSupply.toString(),
      ratio: mm.demandSupplyRatio,
    },
    strength,
  };
}

// ============ Scoring ============

/**
 * Score an opportunity seed for ROI estimation.
 */
export function scoreOpportunity(
  estimatedDemand: bigint,
  estimatedEffort: number,
): number {
  if (estimatedEffort === 0) return 0;

  // ROI = demand revenue / (effort × average bounty cost)
  const AVERAGE_BOUNTY_COST = 10_000_000n; // 0.01 LGM in nanoLGM
  const estimatedCost = BigInt(estimatedEffort) * AVERAGE_BOUNTY_COST;

  if (estimatedCost === 0n) return 0;

  return Number((estimatedDemand * 100n) / estimatedCost) / 100;
}

// ============ Helpers ============

function buildSeed(
  domain: string,
  signal: DemandSignal,
  estimatedDemand: bigint,
  estimatedEffort: number,
  currentBlock: bigint,
  nonce: bigint,
): OpportunitySeed {
  const id = generateSeedId(domain, nonce);

  return {
    id,
    detectedAtBlock: currentBlock,
    knowledgeDomain: domain,
    signal,
    confidence: signal.strength,
    estimatedDemand,
    estimatedEffort,
    estimatedROI: scoreOpportunity(estimatedDemand, estimatedEffort),
    status: "detected",
    expiresAtBlock: currentBlock + SEED_EXPIRY_BLOCKS,
  };
}

function estimateEffort(dm: DomainMetrics): number {
  // Rough heuristic: more facts = more complex domain = more tasks
  const baseEffort = 10;
  const complexityFactor = Math.min(5, dm.factCount / 100);
  return Math.ceil(baseEffort + complexityFactor * 10);
}

function generateSeedId(domain: string, nonce: bigint): string {
  const input = `${domain}:seed:${nonce}`;
  return Buffer.from(
    domainHash("LGM.tree.seed.id.v1", input).slice(0, 16),
  ).toString("hex");
}
