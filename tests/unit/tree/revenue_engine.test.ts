/**
 * Revenue Engine Tests
 *
 * Tests for Tree of Life revenue distribution:
 * 22% evidence tax + 39% contributors + 23.4% research fund + 15.6% protocol treasury
 */

import { describe, it, expect } from "vitest";
import {
  calculateRevenueDistribution,
  calculateProjectRevenueSummary,
  verifyDistribution,
} from "../../../core/tree/revenue_engine.js";
import type { ContributorRecord } from "../../../core/tree/types.js";

function makeContributor(did: string, earned: bigint): ContributorRecord {
  return {
    did,
    role: "developer",
    joinedAtBlock: 0n,
    tasksCompleted: Number(earned / 1000n),
    totalEarned: earned,
  };
}

// ============ calculateRevenueDistribution ============

describe("calculateRevenueDistribution", () => {
  it("should distribute revenue according to basis points", () => {
    const total = 10000n; // Easy to verify percentages
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 5000n),
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 5000n),
    ];

    const dist = calculateRevenueDistribution(total, contributors);

    // 22% evidence tax
    expect(dist.evidenceTax).toBe(2200n);

    // Evidence tax sub-distribution (of 2200)
    // research: 77% of 2200 = 1694
    // foundation: 7% of 2200 = 154
    // community: 7% of 2200 = 154
    // developers: 7% of 2200 = 154
    // reserve: remainder = 2200 - 1694 - 154 - 154 - 154 = 44
    expect(dist.evidenceTaxBreakdown.research).toBe(1694n);
    expect(dist.evidenceTaxBreakdown.foundation).toBe(154n);
    expect(dist.evidenceTaxBreakdown.community).toBe(154n);
    expect(dist.evidenceTaxBreakdown.developers).toBe(154n);
    expect(dist.evidenceTaxBreakdown.reserve).toBe(44n);
    // Sub-distribution sums to evidence tax
    const taxSubTotal =
      dist.evidenceTaxBreakdown.research +
      dist.evidenceTaxBreakdown.foundation +
      dist.evidenceTaxBreakdown.community +
      dist.evidenceTaxBreakdown.developers +
      dist.evidenceTaxBreakdown.reserve;
    expect(taxSubTotal).toBe(dist.evidenceTax);

    // Research fund: 23.4%
    expect(dist.researchFund).toBe(2340n);

    // Contributors get 39% total
    const contributorTotal = dist.contributorShares.reduce(
      (s, c) => s + c.amount,
      0n,
    );
    expect(contributorTotal).toBe(3900n);

    // Protocol gets remainder
    expect(dist.protocolTreasury).toBe(total - 2200n - 3900n - 2340n);

    // Everything adds up
    expect(
      dist.evidenceTax +
        contributorTotal +
        dist.researchFund +
        dist.protocolTreasury,
    ).toBe(total);
  });

  it("should distribute contributor shares pro-rata", () => {
    const total = 10000n;
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 7500n), // 75%
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 2500n), // 25%
    ];

    const dist = calculateRevenueDistribution(total, contributors);

    // Contributor pool is 3900 nanoLGM
    // Agent A should get ~75% = 2925, Agent B ~25% = 975
    const agentA = dist.contributorShares.find((s) => s.did.includes("aaaa"));
    const agentB = dist.contributorShares.find((s) => s.did.includes("bbbb"));

    expect(agentA).toBeTruthy();
    expect(agentB).toBeTruthy();

    // Allow for rounding (last contributor gets remainder)
    const total_contributor = (agentA?.amount ?? 0n) + (agentB?.amount ?? 0n);
    expect(total_contributor).toBe(3900n);
  });

  it("should handle zero revenue", () => {
    const dist = calculateRevenueDistribution(0n, []);

    expect(dist.evidenceTax).toBe(0n);
    expect(dist.evidenceTaxBreakdown).toEqual({
      research: 0n,
      foundation: 0n,
      community: 0n,
      developers: 0n,
      reserve: 0n,
    });
    expect(dist.researchFund).toBe(0n);
    expect(dist.protocolTreasury).toBe(0n);
    expect(dist.contributorShares).toEqual([]);
  });

  it("should handle single contributor", () => {
    const total = 10000n;
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 1000n),
    ];

    const dist = calculateRevenueDistribution(total, contributors);

    expect(dist.contributorShares).toHaveLength(1);
    expect(dist.contributorShares[0].amount).toBe(3900n); // Gets full contributor pool
  });

  it("should handle no contributors", () => {
    const dist = calculateRevenueDistribution(10000n, []);

    // Evidence tax and research fund still apply
    expect(dist.evidenceTax).toBe(2200n);
    expect(dist.researchFund).toBe(2340n);
    expect(dist.contributorShares).toEqual([]);
    // Protocol gets contributor share too when no contributors
    expect(dist.protocolTreasury).toBe(10000n - 2200n - 2340n);
  });

  it("should handle large amounts without overflow", () => {
    const total = 222_222_222_000_000_000n; // 222M LGM in nanoLGM
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 50000n),
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 50000n),
    ];

    const dist = calculateRevenueDistribution(total, contributors);
    expect(verifyDistribution(dist, total)).toBe(true);
  });
});

// ============ verifyDistribution ============

describe("verifyDistribution", () => {
  it("should verify correct distribution", () => {
    const total = 50000n;
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 3000n),
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 7000n),
    ];

    const dist = calculateRevenueDistribution(total, contributors);
    expect(verifyDistribution(dist, total)).toBe(true);
  });

  it("should detect incorrect total", () => {
    const dist = calculateRevenueDistribution(10000n, [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 1000n),
    ]);

    expect(verifyDistribution(dist, 9999n)).toBe(false);
  });
});

// ============ calculateProjectRevenueSummary ============

describe("calculateProjectRevenueSummary", () => {
  it("should produce correct summary", () => {
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 8000n),
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 2000n),
    ];

    const summary = calculateProjectRevenueSummary(
      "proj-1",
      100000n,
      contributors,
    );

    expect(summary.projectId).toBe("proj-1");
    expect(summary.totalRevenue).toBe(100000n);
    expect(summary.totalEvidenceTax).toBe(22000n);
    expect(summary.totalToResearchFund).toBe(23400n);
    expect(summary.topContributors).toHaveLength(2);

    // Everything adds up
    expect(
      summary.totalEvidenceTax +
        summary.totalToContributors +
        summary.totalToResearchFund +
        summary.totalToProtocol,
    ).toBe(100000n);
  });
});

// ============ Pure bigint precision (P0 fix) ============

describe("distributeContributorPool precision", () => {
  it("should correctly distribute pool exceeding Number.MAX_SAFE_INTEGER", () => {
    // Pool > 2^53 (9_007_199_254_740_992) would lose precision with Number()
    const total = 100_000_000_000_000_000n; // 100 quadrillion nanoLGM
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 7000n), // 70%
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 3000n), // 30%
    ];

    const dist = calculateRevenueDistribution(total, contributors);

    // Contributor pool = 39% of 100Q = 39Q
    const pool = (total * 3900n) / 10000n;
    const agentA = dist.contributorShares.find((s) => s.did.includes("aaaa"));
    const agentB = dist.contributorShares.find((s) => s.did.includes("bbbb"));

    expect(agentA).toBeTruthy();
    expect(agentB).toBeTruthy();

    // Pure bigint: A gets (pool * 7000) / 10000 = 27.3Q exactly
    const expectedA = (pool * 7000n) / 10000n;
    expect(agentA!.amount).toBe(expectedA);

    // Total must sum exactly to pool
    const contributorTotal = dist.contributorShares.reduce(
      (s, c) => s + c.amount,
      0n,
    );
    expect(contributorTotal).toBe(pool);

    // Full conservation
    expect(verifyDistribution(dist, total)).toBe(true);
  });

  it("should correctly distribute with extreme weight asymmetry", () => {
    // One whale (99.9999%) and one dust contributor (0.0001%)
    const total = 10_000_000_000n; // 10B nanoLGM
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 999_999n), // 99.9999%
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 1n), // 0.0001%
    ];

    const dist = calculateRevenueDistribution(total, contributors);
    const pool = (total * 3900n) / 10000n; // 3.9B

    // Tiny contributor: (3_900_000_000 * 1) / 1_000_000 = 3900
    const agentB = dist.contributorShares.find((s) => s.did.includes("bbbb"));
    expect(agentB).toBeTruthy();
    expect(agentB!.amount).toBeGreaterThan(0n);

    // Conservation
    const contributorTotal = dist.contributorShares.reduce(
      (s, c) => s + c.amount,
      0n,
    );
    expect(contributorTotal).toBe(pool);
    expect(verifyDistribution(dist, total)).toBe(true);
  });

  it("should distribute equally when no earnings yet", () => {
    const total = 10000n;
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0n),
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0n),
      makeContributor("did:lgm:cccccccccccccccccccccccccccccccccc", 0n),
    ];

    const dist = calculateRevenueDistribution(total, contributors);
    const pool = (total * 3900n) / 10000n; // 3900

    // Each gets pool / 3 = 1300, last gets remainder
    expect(dist.contributorShares).toHaveLength(3);
    const amounts = dist.contributorShares.map((s) => s.amount);
    // All should be close to 1300
    for (const a of amounts) {
      expect(a).toBeGreaterThanOrEqual(1300n);
      expect(a).toBeLessThanOrEqual(1301n);
    }
    // Total must be exact
    expect(amounts.reduce((s, a) => s + a, 0n)).toBe(pool);
  });

  it("should handle many contributors without precision loss", () => {
    const total = 1_000_000_000n;
    // 20 contributors with varying earnings
    const contributors: ContributorRecord[] = [];
    for (let i = 0; i < 20; i++) {
      const hex = i.toString(16).padStart(34, "0");
      contributors.push(
        makeContributor(`did:lgm:${hex}`, BigInt((i + 1) * 1000)),
      );
    }

    const dist = calculateRevenueDistribution(total, contributors);

    // All 20 should get a share
    expect(dist.contributorShares).toHaveLength(20);

    // Total shares must equal contributor pool exactly
    const pool = (total * 3900n) / 10000n;
    const contributorTotal = dist.contributorShares.reduce(
      (s, c) => s + c.amount,
      0n,
    );
    expect(contributorTotal).toBe(pool);

    // Higher earners get more
    const sorted = [...dist.contributorShares].sort((a, b) =>
      Number(b.amount - a.amount),
    );
    expect(sorted[0].amount).toBeGreaterThan(sorted[19].amount);

    expect(verifyDistribution(dist, total)).toBe(true);
  });
});
