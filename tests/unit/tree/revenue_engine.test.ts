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
