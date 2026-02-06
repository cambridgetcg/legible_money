/**
 * Opportunity Detector Tests
 *
 * Tests for automated product opportunity detection from knowledge demand signals.
 */

import { describe, it, expect } from "vitest";
import {
  detectOpportunities,
  scoreOpportunity,
  aggregateDomainMetrics,
  aggregateCrossDomainMetrics,
} from "../../../core/tree/opportunity_detector.js";
import type {
  DomainMetrics,
  MarketMetrics,
  AgentRequest,
  CrossDomainMetrics,
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
        fulfilledBounties: 2, // Low bounties so query_volume is strongest signal
        citationCount: 10n, // Low citations so no citation signal
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

// ============ agent_request & cross_domain signals ============

describe("detectOpportunities — agent_request signal", () => {
  it("should detect agent_request opportunities", () => {
    const agentRequests: AgentRequest[] = [
      {
        requesterDid: "did:lgm:agent-001",
        domain: "quantum-computing",
        description: "Need verified quantum error correction data",
        estimatedDemand: 2_000_000_000n, // 2 LGM
      },
    ];

    const seeds = detectOpportunities([], [], [], 100n, 0n, agentRequests);

    expect(seeds.length).toBeGreaterThan(0);
    const agentSeed = seeds.find((s) => s.signal.type === "agent_request");
    expect(agentSeed).toBeTruthy();
    expect(agentSeed!.knowledgeDomain).toBe("quantum-computing");
    expect(agentSeed!.confidence).toBeGreaterThanOrEqual(MIN_SEED_CONFIDENCE);
  });

  it("should skip agent_request for existing project domains", () => {
    const agentRequests: AgentRequest[] = [
      {
        requesterDid: "did:lgm:agent-001",
        domain: "existing-domain",
        description: "Requested domain",
        estimatedDemand: 2_000_000_000n,
      },
    ];

    const seeds = detectOpportunities(
      [],
      [],
      ["existing-domain"],
      100n,
      0n,
      agentRequests,
    );
    const agentSeed = seeds.find((s) => s.signal.type === "agent_request");
    expect(agentSeed).toBeUndefined();
  });
});

describe("detectOpportunities — cross_domain signal", () => {
  it("should detect cross_domain opportunities", () => {
    const crossDomain: CrossDomainMetrics[] = [
      {
        domainA: "neuroscience",
        domainB: "ai-safety",
        crossReferences: 10, // Well above threshold of 5
        combinedCitationCount: 200n,
        combinedFactCount: 40,
      },
    ];

    const seeds = detectOpportunities([], [], [], 100n, 0n, [], crossDomain);

    expect(seeds.length).toBeGreaterThan(0);
    const crossSeed = seeds.find((s) => s.signal.type === "cross_domain");
    expect(crossSeed).toBeTruthy();
    expect(crossSeed!.confidence).toBeGreaterThanOrEqual(MIN_SEED_CONFIDENCE);
  });

  it("should skip cross_domain below threshold", () => {
    const crossDomain: CrossDomainMetrics[] = [
      {
        domainA: "physics",
        domainB: "chemistry",
        crossReferences: 2, // Below threshold of 5
        combinedCitationCount: 10n,
        combinedFactCount: 5,
      },
    ];

    const seeds = detectOpportunities([], [], [], 100n, 0n, [], crossDomain);

    const crossSeed = seeds.find((s) => s.signal.type === "cross_domain");
    expect(crossSeed).toBeUndefined();
  });
});

// ============ Metrics Aggregation ============

describe("aggregateDomainMetrics", () => {
  it("should aggregate domain metrics from state callbacks", () => {
    const factsDB: Record<
      string,
      {
        id: string;
        citationCount: number;
        confidence: number;
        domain: string;
      }[]
    > = {
      "ai-safety": [
        {
          id: "fact:1",
          citationCount: 10,
          confidence: 0.9,
          domain: "ai-safety",
        },
        {
          id: "fact:2",
          citationCount: 5,
          confidence: 0.8,
          domain: "ai-safety",
        },
      ],
      climate: [
        {
          id: "fact:3",
          citationCount: 20,
          confidence: 0.95,
          domain: "climate",
        },
      ],
    };

    const bounties = [
      { targetFactId: "fact:1", status: "open" },
      { targetFactId: "fact:1", status: "fulfilled" },
      { targetFactId: "fact:3", status: "fulfilled" },
    ];

    const metrics = aggregateDomainMetrics(
      (d) => factsDB[d] ?? [],
      () => ["ai-safety", "climate", "empty-domain"],
      () => bounties,
      (fId) => {
        for (const facts of Object.values(factsDB)) {
          const f = facts.find((x) => x.id === fId);
          if (f) return { domain: f.domain };
        }
        return null;
      },
    );

    // "empty-domain" should be skipped (no facts)
    expect(metrics).toHaveLength(2);

    const aiSafety = metrics.find((m) => m.domain === "ai-safety")!;
    expect(aiSafety).toBeTruthy();
    expect(aiSafety.factCount).toBe(2);
    expect(aiSafety.citationCount).toBe(15n); // 10 + 5
    expect(aiSafety.averageConfidence).toBeCloseTo(0.85); // (0.9+0.8)/2
    expect(aiSafety.activeBounties).toBe(1); // 1 open bounty for fact:1
    expect(aiSafety.fulfilledBounties).toBe(1); // 1 fulfilled for fact:1

    const climate = metrics.find((m) => m.domain === "climate")!;
    expect(climate.factCount).toBe(1);
    expect(climate.citationCount).toBe(20n);
    expect(climate.fulfilledBounties).toBe(1);
  });
});

describe("aggregateCrossDomainMetrics", () => {
  it("should detect cross-domain metrics from reference graph", () => {
    const factsDB: Record<
      string,
      { id: string; citationCount: number; domain: string }[]
    > = {
      physics: [
        { id: "fact:p1", citationCount: 10, domain: "physics" },
        { id: "fact:p2", citationCount: 5, domain: "physics" },
      ],
      chemistry: [{ id: "fact:c1", citationCount: 8, domain: "chemistry" }],
      biology: [{ id: "fact:b1", citationCount: 3, domain: "biology" }],
    };

    // fact:p1 -> fact:c1 (physics cites chemistry)
    // fact:p2 -> fact:c1 (physics cites chemistry again)
    // fact:c1 -> fact:b1 (chemistry cites biology)
    const refs: Record<string, string[]> = {
      "fact:p1": ["fact:c1"],
      "fact:p2": ["fact:c1"],
      "fact:c1": ["fact:b1"],
    };

    const allFacts = Object.values(factsDB).flat();

    const crossMetrics = aggregateCrossDomainMetrics(
      (d) => factsDB[d] ?? [],
      () => ["physics", "chemistry", "biology"],
      (fId) => refs[fId] ?? [],
      (fId) => {
        const f = allFacts.find((x) => x.id === fId);
        return f ? { domain: f.domain, citationCount: f.citationCount } : null;
      },
    );

    // physics↔chemistry: 2 cross-refs (>= 2 threshold), should appear
    const physChem = crossMetrics.find(
      (m) =>
        (m.domainA === "chemistry" && m.domainB === "physics") ||
        (m.domainA === "physics" && m.domainB === "chemistry"),
    );
    expect(physChem).toBeTruthy();
    expect(physChem!.crossReferences).toBe(2);

    // chemistry↔biology: only 1 cross-ref (below threshold of 2), should NOT appear
    const chemBio = crossMetrics.find(
      (m) =>
        (m.domainA === "chemistry" && m.domainB === "biology") ||
        (m.domainA === "biology" && m.domainB === "chemistry"),
    );
    expect(chemBio).toBeUndefined();
  });
});
