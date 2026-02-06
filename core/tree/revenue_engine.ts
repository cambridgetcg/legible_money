/**
 * Revenue Engine — Tree of Life Project Revenue Distribution
 *
 * Distributes revenue from Tree of Life service calls (deploy_service/call_service)
 * among project contributors, research fund, and protocol treasury.
 *
 * REVENUE SPLIT (basis points, sum to 10000):
 *   2200 BP = 22% evidence tax (frozen forever, split 5 ways via TREASURY.ALLOCATION)
 *   3900 BP = 39% to project contributors (pro-rata by contribution weight)
 *   2340 BP = 23.4% to research fund (feeds perpetual truth-finding loop)
 *   1560 BP = 15.6% to protocol treasury (infrastructure)
 *
 * NOTE: This is SEPARATE from the Truth Gateway billing model (core/billing/).
 * The Truth Gateway uses a 60/30/10 split (provider/knowledge/protocol) because
 * gateway queries reward knowledge contributors (fact submitters, verifiers),
 * while tree services reward project contributors (developers, reviewers).
 * Both models apply the 22% research share — they differ in who gets the remainder.
 */

import type { DID } from "../agents/identity/did.js";
import type {
  RevenueDistribution,
  EvidenceTaxBreakdown,
  ContributorShare,
  ContributorRecord,
} from "./types.js";
import {
  REVENUE_EVIDENCE_TAX_BP,
  REVENUE_CONTRIBUTORS_BP,
  REVENUE_RESEARCH_FUND_BP,
  REVENUE_PROTOCOL_TREASURY_BP,
} from "./types.js";
import { splitEvidenceTax } from "../billing/reward_router.js";
import { calculateAllContributorWeights } from "./coordination/skill_matching.js";

// Static assertion: all 4 BP constants must sum to 10000 (100%)
const _BP_SUM =
  REVENUE_EVIDENCE_TAX_BP +
  REVENUE_CONTRIBUTORS_BP +
  REVENUE_RESEARCH_FUND_BP +
  REVENUE_PROTOCOL_TREASURY_BP;
if (_BP_SUM !== 10000n) {
  throw new Error(`Revenue basis points must sum to 10000, got ${_BP_SUM}`);
}

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
      evidenceTaxBreakdown: {
        research: 0n,
        foundation: 0n,
        community: 0n,
        developers: 0n,
        reserve: 0n,
      },
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
    evidenceTaxBreakdown: distributeEvidenceTax(evidenceTax),
    contributorShares,
    researchFund,
    protocolTreasury: protocolTreasury + unclaimedPool,
  };
}

/**
 * Distribute the contributor pool among project contributors.
 * Weights are based on total bounties earned (reflecting contribution value).
 *
 * Amount calculation uses pure bigint arithmetic: (pool * earned) / totalEarned.
 * This avoids Number() precision loss that would misallocate for pools > 2^53.
 * Float weights are still computed for the ContributorShare.weight display field.
 */
function distributeContributorPool(
  pool: bigint,
  contributors: ContributorRecord[],
): ContributorShare[] {
  if (contributors.length === 0 || pool === 0n) return [];

  const totalEarned = contributors.reduce((sum, c) => sum + c.totalEarned, 0n);
  const weights = calculateAllContributorWeights(contributors);
  const shares: ContributorShare[] = [];
  let distributed = 0n;

  // Sort by DID for deterministic ordering
  const sorted = [...contributors].sort((a, b) => a.did.localeCompare(b.did));
  const n = BigInt(sorted.length);

  for (let i = 0; i < sorted.length; i++) {
    const contributor = sorted[i];
    const weight = weights.get(contributor.did) ?? 0;

    let amount: bigint;
    if (i === sorted.length - 1) {
      // Last contributor gets remainder (handles rounding)
      amount = pool - distributed;
    } else if (totalEarned === 0n) {
      // Equal split when no earnings yet
      amount = pool / n;
    } else {
      // Pure bigint: avoids Number() precision loss for large pools
      amount = (pool * contributor.totalEarned) / totalEarned;
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

/**
 * Sub-distribute the evidence tax across 5 treasury funds.
 * Delegates to splitEvidenceTax() in reward_router.ts (single source of truth)
 * and converts from TreasurySplitEntry[] to EvidenceTaxBreakdown.
 */
function distributeEvidenceTax(evidenceTax: bigint): EvidenceTaxBreakdown {
  if (evidenceTax === 0n) {
    return {
      research: 0n,
      foundation: 0n,
      community: 0n,
      developers: 0n,
      reserve: 0n,
    };
  }

  const entries = splitEvidenceTax(evidenceTax);
  const breakdown: EvidenceTaxBreakdown = {
    research: 0n,
    foundation: 0n,
    community: 0n,
    developers: 0n,
    reserve: 0n,
  };
  for (const entry of entries) {
    breakdown[entry.fund] = entry.amount;
  }
  return breakdown;
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

  if (actualTotal !== expectedTotal) return false;

  // Verify evidence tax sub-distribution sums to evidence tax
  const { research, foundation, community, developers, reserve } =
    distribution.evidenceTaxBreakdown;
  const taxSubTotal = research + foundation + community + developers + reserve;

  return taxSubTotal === distribution.evidenceTax;
}
