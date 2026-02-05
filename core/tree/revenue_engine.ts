/**
 * Revenue Engine
 *
 * Processes service revenue and distributes it according to the Tree of Life
 * economics: evidence tax → contributors → research fund → protocol treasury.
 *
 * Revenue distribution follows the same economic model as knowledge queries:
 *   22% evidence tax (frozen forever, split 5 ways)
 *   78% remaining:
 *     50% to contributors (pro-rata by contribution weight)
 *     30% to research fund (feeds perpetual truth-finding loop)
 *     20% to protocol treasury (infrastructure)
 */

import type { DID } from "../agents/identity/did.js";
import type {
  RevenueDistribution,
  ContributorShare,
  ContributorRecord,
} from "./types.js";
import {
  REVENUE_EVIDENCE_TAX_BP,
  REVENUE_CONTRIBUTORS_BP,
  REVENUE_RESEARCH_FUND_BP,
} from "./types.js";
import { calculateAllContributorWeights } from "./coordination/skill_matching.js";

// ============ Revenue Distribution ============

/**
 * Calculate how revenue should be distributed for a service call.
 *
 * Uses basis points (out of 10000) for precise bigint math:
 *   2200 BP = 22% evidence tax
 *   3900 BP = 39% contributors
 *   2340 BP = 23.4% research fund
 *   1560 BP = 15.6% protocol treasury
 *   Total = 10000 BP = 100%
 */
export function calculateRevenueDistribution(
  totalRevenue: bigint,
  contributors: ContributorRecord[],
): RevenueDistribution {
  if (totalRevenue === 0n) {
    return {
      evidenceTax: 0n,
      contributorShares: [],
      researchFund: 0n,
      protocolTreasury: 0n,
    };
  }

  // Calculate base amounts using basis points
  const evidenceTax = (totalRevenue * REVENUE_EVIDENCE_TAX_BP) / 10000n;
  const contributorPool = (totalRevenue * REVENUE_CONTRIBUTORS_BP) / 10000n;
  const researchFund = (totalRevenue * REVENUE_RESEARCH_FUND_BP) / 10000n;
  // Protocol treasury gets the remainder (handles rounding)
  const protocolTreasury =
    totalRevenue - evidenceTax - contributorPool - researchFund;

  // Distribute contributor pool pro-rata
  const contributorShares = distributeContributorPool(
    contributorPool,
    contributors,
  );

  // When no contributors, redirect their pool to protocol treasury
  const actualContributorTotal = contributorShares.reduce(
    (s, c) => s + c.amount,
    0n,
  );
  const unclaimedPool = contributorPool - actualContributorTotal;

  return {
    evidenceTax,
    contributorShares,
    researchFund,
    protocolTreasury: protocolTreasury + unclaimedPool,
  };
}

/**
 * Distribute the contributor pool among project contributors.
 * Weights are based on total bounties earned (reflecting contribution value).
 */
function distributeContributorPool(
  pool: bigint,
  contributors: ContributorRecord[],
): ContributorShare[] {
  if (contributors.length === 0 || pool === 0n) return [];

  const weights = calculateAllContributorWeights(contributors);
  const shares: ContributorShare[] = [];
  let distributed = 0n;

  // Sort by DID for deterministic ordering
  const sorted = [...contributors].sort((a, b) => a.did.localeCompare(b.did));

  for (let i = 0; i < sorted.length; i++) {
    const contributor = sorted[i];
    const weight = weights.get(contributor.did) ?? 0;

    let amount: bigint;
    if (i === sorted.length - 1) {
      // Last contributor gets remainder (handles rounding)
      amount = pool - distributed;
    } else {
      amount = BigInt(Math.floor(Number(pool) * weight));
    }

    if (amount > 0n) {
      shares.push({
        did: contributor.did,
        amount,
        weight,
      });
      distributed += amount;
    }
  }

  return shares;
}

// ============ Revenue Summaries ============

/**
 * Summary of a project's revenue
 */
export interface ProjectRevenueSummary {
  projectId: string;
  totalRevenue: bigint;
  totalEvidenceTax: bigint;
  totalToContributors: bigint;
  totalToResearchFund: bigint;
  totalToProtocol: bigint;
  topContributors: ContributorShare[];
}

/**
 * Summary of an agent's revenue across all projects
 */
export interface AgentRevenueSummary {
  agentDid: DID;
  totalEarned: bigint;
  projectEarnings: Array<{
    projectId: string;
    earned: bigint;
    weight: number;
  }>;
}

/**
 * Calculate project-level revenue summary.
 */
export function calculateProjectRevenueSummary(
  projectId: string,
  totalRevenue: bigint,
  contributors: ContributorRecord[],
): ProjectRevenueSummary {
  const distribution = calculateRevenueDistribution(totalRevenue, contributors);

  return {
    projectId,
    totalRevenue,
    totalEvidenceTax: distribution.evidenceTax,
    totalToContributors: distribution.contributorShares.reduce(
      (sum, s) => sum + s.amount,
      0n,
    ),
    totalToResearchFund: distribution.researchFund,
    totalToProtocol: distribution.protocolTreasury,
    topContributors: distribution.contributorShares
      .sort((a, b) => Number(b.amount - a.amount))
      .slice(0, 10),
  };
}

/**
 * Verify that a revenue distribution sums to the total.
 * Used for integrity checking.
 */
export function verifyDistribution(
  distribution: RevenueDistribution,
  expectedTotal: bigint,
): boolean {
  const contributorTotal = distribution.contributorShares.reduce(
    (sum, s) => sum + s.amount,
    0n,
  );

  const actualTotal =
    distribution.evidenceTax +
    contributorTotal +
    distribution.researchFund +
    distribution.protocolTreasury;

  return actualTotal === expectedTotal;
}
