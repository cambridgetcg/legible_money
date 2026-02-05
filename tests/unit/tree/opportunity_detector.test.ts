/**
 * Opportunity Detector Tests
 *
 * Tests for automated product opportunity detection from knowledge demand signals.
 */

import { describe, it, expect } from "vitest";
import {
  detectOpportunities,
  scoreOpportunity,
} from "../../../core/tree/opportunity_detector.js";
import type {
  DomainMetrics,
  MarketMetrics,
} from "../../../core/tree/opportunity_detector.js";
import {
  MAX_SEEDS_PER_EPOCH,
  MIN_SEED_CONFIDENCE,
} from "../../../core/tree/types.js";

// ============ Test Helpers ============

function makeDomainMetrics(
  overrides: Partial<DomainMetrics> = {},
): DomainMetrics {
  return {
    domain: "test-domain",
    queryCount: 1000n, // High enough for query volume signal
    citationCount: 500n, // High enough for citation cluster signal
    queryRevenue: 1_000_000_000n, // 1 LGM (exceeds AVERAGE_BOUNTY_COST for ROI)
    activeBounties: 5,
    fulfilledBounties: 50, // High enough for bounty pattern signal
    factCount: 50,
    averageConfidence: 0.95,
    queryGrowthRate: 2.0, // 100% growth
    ...overrides,
  };
}

function makeMarketMetrics(
  overrides: Partial<MarketMetrics> = {},
): MarketMetrics {
  return {
    resourceType: "compute.gpu.inference",
    unmatchedDemand: 50000n,
    unmatchedSupply: 10000n,
    demandSupplyRatio: 5.0,
    ...overrides,
  };
}

// ============ detectOpportunities ============

describe("detectOpportunities", () => {
  it("should detect query volume opportunities", () => {
    const domains = [
      makeDomainMetrics({
        domain: "ai-safety",
        queryCount: 1000n,
        queryGrowthRate: 2.5,
      }),
    ];

    const seeds = detectOpportunities(domains, [], [], 100n, 0n);

    expect(seeds.length).toBeGreaterThan(0);
    expect(seeds[0].knowledgeDomain).toBe("ai-safety");
    expect(seeds[0].signal.type).toBe("query_volume");
    expect(seeds[0].confidence).toBeGreaterThanOrEqual(MIN_SEED_CONFIDENCE);
  });

  it("should detect citation cluster opportunities", () => {
    const domains = [
      makeDomainMetrics({
        domain: "climate-science",
        queryCount: 10n, // Low queries (no query volume signal)
        queryGrowthRate: 0.5, // No growth
        citationCount: 500n, // High citations
        averageConfidence: 0.95,
        fulfilledBounties: 2, // Low bounties (no bounty signal)
      }),
    ];

    const seeds = detectOpportunities(domains, [], [], 100n, 0n);

    expect(seeds.length).toBeGreaterThan(0);
    const citationSeed = seeds.find(
      (s) => s.signal.type === "citation_cluster",
    );
    expect(citationSeed).toBeTruthy();
  });

  it("should detect bounty pattern opportunities", () => {
    const domains = [
      makeDomainMetrics({
        domain: "materials-science",
        queryCount: 10n, // Low queries (no query volume signal)
        queryGrowthRate: 0.5, // No growth
        citationCount: 10n, // Low citations (no citation signal)
        averageConfidence: 0.5, // Low confidence
        fulfilledBounties: 50, // Many fulfilled bounties
        activeBounties: 5,
      }),
    ];

    const seeds = detectOpportunities(domains, [], [], 100n, 0n);

    expect(seeds.length).toBeGreaterThan(0);
    const bountySeed = seeds.find((s) => s.signal.type === "bounty_pattern");
    expect(bountySeed).toBeTruthy();
  });

  it("should detect market gap opportunities", () => {
    const markets = [
      makeMarketMetrics({
        resourceType: "compute.gpu.training",
        unmatchedDemand: 100000n,
        demandSupplyRatio: 5.0,
      }),
    ];

    const seeds = detectOpportunities([], markets, [], 100n, 0n);

    expect(seeds.length).toBeGreaterThan(0);
    expect(seeds[0].signal.type).toBe("market_gap");
  });

  it("should skip domains with existing projects", () => {
    const domains = [makeDomainMetrics({ domain: "existing-domain" })];

    const seeds = detectOpportunities(
      domains,
      [],
      ["existing-domain"],
      100n,
      0n,
    );
    expect(seeds).toHaveLength(0);
  });

  it("should limit to MAX_SEEDS_PER_EPOCH", () => {
    const domains = Array.from({ length: 10 }, (_, i) =>
      makeDomainMetrics({
        domain: `domain-${i}`,
        queryCount: 1000n + BigInt(i * 100),
        queryGrowthRate: 2.0 + i * 0.1,
      }),
    );

    const seeds = detectOpportunities(domains, [], [], 100n, 0n);
    expect(seeds.length).toBeLessThanOrEqual(MAX_SEEDS_PER_EPOCH);
  });

  it("should sort by estimated ROI", () => {
    const domains = [
      makeDomainMetrics({
        domain: "low-roi",
        queryCount: 1000n,
        queryGrowthRate: 2.0,
        queryRevenue: 100_000_000n,
      }),
      makeDomainMetrics({
        domain: "high-roi",
        queryCount: 1000n,
        queryGrowthRate: 3.0,
        queryRevenue: 5_000_000_000n,
      }),
    ];

    const seeds = detectOpportunities(domains, [], [], 100n, 0n);

    if (seeds.length >= 2) {
      expect(seeds[0].estimatedROI).toBeGreaterThanOrEqual(
        seeds[1].estimatedROI,
      );
    }
  });

  it("should filter below minimum confidence", () => {
    const domains = [
      makeDomainMetrics({
        domain: "weak-signal",
        queryCount: 50n, // Below threshold
        queryGrowthRate: 1.0, // No growth
        citationCount: 5n, // Low citations
        fulfilledBounties: 2, // Few bounties
      }),
    ];

    const seeds = detectOpportunities(domains, [], [], 100n, 0n);
    expect(seeds).toHaveLength(0);
  });

  it("should set correct expiry block", () => {
    const domains = [makeDomainMetrics()];
    const currentBlock = 1000n;

    const seeds = detectOpportunities(domains, [], [], currentBlock, 0n);

    if (seeds.length > 0) {
      expect(seeds[0].expiresAtBlock).toBeGreaterThan(currentBlock);
      expect(seeds[0].detectedAtBlock).toBe(currentBlock);
      expect(seeds[0].status).toBe("detected");
    }
  });
});

// ============ scoreOpportunity ============

describe("scoreOpportunity", () => {
  // Note: AVERAGE_BOUNTY_COST = 10_000_000n (0.01 LGM)
  // ROI = demand * 100 / (effort * 10M) / 100

  it("should return positive ROI for viable opportunities", () => {
    const roi = scoreOpportunity(1_000_000_000n, 5); // 1 LGM demand
    expect(roi).toBeGreaterThan(0);
  });

  it("should return 0 for zero effort", () => {
    const roi = scoreOpportunity(1_000_000_000n, 0);
    expect(roi).toBe(0);
  });

  it("should return 0 for zero demand", () => {
    const roi = scoreOpportunity(0n, 10);
    expect(roi).toBe(0);
  });

  it("should scale with demand", () => {
    const lowROI = scoreOpportunity(100_000_000n, 10); // 0.1 LGM
    const highROI = scoreOpportunity(10_000_000_000n, 10); // 10 LGM

    expect(highROI).toBeGreaterThan(lowROI);
  });

  it("should inversely scale with effort", () => {
    const easyROI = scoreOpportunity(5_000_000_000n, 5); // 5 LGM, 5 tasks
    const hardROI = scoreOpportunity(5_000_000_000n, 50); // 5 LGM, 50 tasks

    expect(easyROI).toBeGreaterThan(hardROI);
  });
});
