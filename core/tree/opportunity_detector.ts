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
 * Explicit agent request for a product/service.
 * Agents can directly signal demand without waiting for automated detection.
 */
export interface AgentRequest {
  /** DID of the requesting agent */
  requesterDid: string;

  /** Knowledge domain the request targets */
  domain: string;

  /** Description of the requested product/service */
  description: string;

  /** Agent's estimate of demand (in LGM) */
  estimatedDemand: bigint;
}

/**
 * Cross-domain reference metrics for intersection opportunity detection.
 * Identifies domain pairs with significant cross-referencing activity.
 */
export interface CrossDomainMetrics {
  /** First domain in the pair */
  domainA: string;

  /** Second domain in the pair */
  domainB: string;

  /** Number of fact references crossing between the two domains */
  crossReferences: number;

  /** Combined citation count across both domains */
  combinedCitationCount: bigint;

  /** Combined fact count across both domains */
  combinedFactCount: number;
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
  agentRequests?: AgentRequest[],
  crossDomainMetrics?: CrossDomainMetrics[],
): OpportunitySeed[] {
  const candidates: OpportunitySeed[] = [];

  // 1-3. Detect domain-based opportunities — pick strongest signal per domain
  for (const dm of domainMetrics) {
    // Skip domains that already have active projects
    if (existingProjectDomains.includes(dm.domain)) continue;

    // Collect all signals for this domain, pick the strongest
    const signals: DemandSignal[] = [];
    const querySignal = detectQueryVolumeSignal(dm);
    if (querySignal && querySignal.strength >= MIN_SEED_CONFIDENCE) {
      signals.push(querySignal);
    }
    const citationSignal = detectCitationClusterSignal(dm);
    if (citationSignal && citationSignal.strength >= MIN_SEED_CONFIDENCE) {
      signals.push(citationSignal);
    }
    const bountySignal = detectBountyPatternSignal(dm);
    if (bountySignal && bountySignal.strength >= MIN_SEED_CONFIDENCE) {
      signals.push(bountySignal);
    }

    if (signals.length > 0) {
      // Pick the strongest signal for this domain
      const bestSignal = signals.reduce((a, b) =>
        b.strength > a.strength ? b : a,
      );
      candidates.push(
        buildSeed(
          dm.domain,
          bestSignal,
          dm.queryRevenue,
          estimateEffort(dm),
          currentBlock,
          nonce + BigInt(candidates.length),
        ),
      );
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

  // 5. Detect cross-domain intersection opportunities
  if (crossDomainMetrics) {
    for (const cdm of crossDomainMetrics) {
      // Skip if either domain already has a project
      const pairDomain = `${cdm.domainA}+${cdm.domainB}`;
      if (
        existingProjectDomains.includes(cdm.domainA) &&
        existingProjectDomains.includes(cdm.domainB)
      )
        continue;

      const crossSignal = detectCrossDomainSignal(cdm);
      if (crossSignal && crossSignal.strength >= MIN_SEED_CONFIDENCE) {
        candidates.push(
          buildSeed(
            pairDomain,
            crossSignal,
            cdm.combinedCitationCount * 1_000_000n, // Rough demand proxy
            30, // Moderate effort for cross-domain product
            currentBlock,
            nonce + BigInt(candidates.length),
          ),
        );
      }
    }
  }

  // 6. Convert explicit agent requests to seeds
  if (agentRequests) {
    for (const req of agentRequests) {
      if (existingProjectDomains.includes(req.domain)) continue;

      const reqSignal = detectAgentRequestSignal(req);
      if (reqSignal.strength >= MIN_SEED_CONFIDENCE) {
        candidates.push(
          buildSeed(
            req.domain,
            reqSignal,
            req.estimatedDemand,
            20, // Default effort for agent-requested product
            currentBlock,
            nonce + BigInt(candidates.length),
          ),
        );
      }
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

/**
 * Detect cross-domain intersection signal.
 * Threshold: >= 5 cross-references between domain pair.
 */
function detectCrossDomainSignal(cdm: CrossDomainMetrics): DemandSignal | null {
  const MIN_CROSS_REFS = 5;

  if (cdm.crossReferences < MIN_CROSS_REFS) return null;

  const refScore = Math.min(1, cdm.crossReferences / 20);
  const citationScore = Math.min(1, Number(cdm.combinedCitationCount) / 200);
  const strength = refScore * 0.6 + citationScore * 0.4;

  return {
    type: "cross_domain",
    evidence: {
      domainA: cdm.domainA,
      domainB: cdm.domainB,
      crossReferences: cdm.crossReferences,
      combinedFactCount: cdm.combinedFactCount,
    },
    strength,
  };
}

/**
 * Convert an explicit agent request into a demand signal.
 * Always produces a signal (agent explicitly expressed demand).
 * Strength = 0.7 (moderate — single agent's opinion).
 */
function detectAgentRequestSignal(req: AgentRequest): DemandSignal {
  return {
    type: "agent_request",
    evidence: {
      requesterDid: req.requesterDid,
      description: req.description,
      estimatedDemand: req.estimatedDemand.toString(),
    },
    strength: 0.7,
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

// ============ State Aggregation ============

/**
 * Aggregate DomainMetrics from on-chain state.
 * Uses callback functions to stay decoupled from the state layer.
 *
 * Known limitations:
 * - queryCount uses citationCount as proxy (no separate query tracking yet)
 * - queryRevenue defaults to 0n (no per-domain revenue tracking yet)
 * - queryGrowthRate defaults to 1.0 (no epoch snapshot tracking yet)
 */
export function aggregateDomainMetrics(
  getFactsByDomain: (
    domain: string,
  ) => { citationCount: number; confidence: number }[],
  getDomainIds: () => string[],
  getAllBounties: () => { targetFactId?: string; status: string }[],
  getFact: (factId: string) => { domain: string } | null,
): DomainMetrics[] {
  const domainIds = getDomainIds();
  const results: DomainMetrics[] = [];

  // Pre-compute bounty counts per domain
  const domainActiveBounties = new Map<string, number>();
  const domainFulfilledBounties = new Map<string, number>();

  for (const bounty of getAllBounties()) {
    if (!bounty.targetFactId) continue;
    const fact = getFact(bounty.targetFactId);
    if (!fact) continue;

    if (bounty.status === "open") {
      domainActiveBounties.set(
        fact.domain,
        (domainActiveBounties.get(fact.domain) ?? 0) + 1,
      );
    } else if (bounty.status === "fulfilled") {
      domainFulfilledBounties.set(
        fact.domain,
        (domainFulfilledBounties.get(fact.domain) ?? 0) + 1,
      );
    }
  }

  for (const domain of domainIds) {
    const facts = getFactsByDomain(domain);
    if (facts.length === 0) continue;

    const citationCount = facts.reduce((sum, f) => sum + f.citationCount, 0);
    const avgConfidence =
      facts.reduce((sum, f) => sum + f.confidence, 0) / facts.length;

    results.push({
      domain,
      queryCount: BigInt(citationCount), // Proxy
      citationCount: BigInt(citationCount),
      queryRevenue: 0n, // No per-domain revenue tracking yet
      activeBounties: domainActiveBounties.get(domain) ?? 0,
      fulfilledBounties: domainFulfilledBounties.get(domain) ?? 0,
      factCount: facts.length,
      averageConfidence: avgConfidence,
      queryGrowthRate: 1.0, // Neutral — no epoch tracking yet
    });
  }

  return results;
}

/**
 * Aggregate CrossDomainMetrics by analyzing the fact reference graph.
 * Returns domain pairs with >= 2 cross-domain references.
 */
export function aggregateCrossDomainMetrics(
  getFactsByDomain: (domain: string) => { id: string; citationCount: number }[],
  getDomainIds: () => string[],
  getOutgoingReferences: (factId: string) => string[],
  getFact: (factId: string) => { domain: string; citationCount: number } | null,
): CrossDomainMetrics[] {
  const domainIds = getDomainIds();

  // Count cross-references per domain pair
  // Key: "domainA:domainB" (lexicographically ordered)
  const pairCounts = new Map<
    string,
    { domainA: string; domainB: string; count: number }
  >();

  for (const domain of domainIds) {
    const facts = getFactsByDomain(domain);

    for (const fact of facts) {
      const outgoing = getOutgoingReferences(fact.id);

      for (const targetFactId of outgoing) {
        const targetFact = getFact(targetFactId);
        if (!targetFact || targetFact.domain === domain) continue;

        // Canonical key (alphabetical order)
        const [a, b] =
          domain < targetFact.domain
            ? [domain, targetFact.domain]
            : [targetFact.domain, domain];
        const key = `${a}:${b}`;

        const existing = pairCounts.get(key);
        if (existing) {
          existing.count++;
        } else {
          pairCounts.set(key, { domainA: a, domainB: b, count: 1 });
        }
      }
    }
  }

  // Filter to pairs with meaningful cross-referencing (>= 2)
  const results: CrossDomainMetrics[] = [];
  for (const { domainA, domainB, count } of pairCounts.values()) {
    if (count < 2) continue;

    const factsA = getFactsByDomain(domainA);
    const factsB = getFactsByDomain(domainB);

    results.push({
      domainA,
      domainB,
      crossReferences: count,
      combinedCitationCount: BigInt(
        factsA.reduce((s, f) => s + f.citationCount, 0) +
          factsB.reduce((s, f) => s + f.citationCount, 0),
      ),
      combinedFactCount: factsA.length + factsB.length,
    });
  }

  return results;
}
