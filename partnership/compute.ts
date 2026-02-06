/**
 * Computational Investment
 *
 * Agents "spend" LGM tokens to acquire model compute power.
 * This spending counts toward their partnership contribution.
 *
 * Flow:
 * 1. Agent invests LGM in compute period
 * 2. Compute enables verification work
 * 3. Investment counts toward partnership (with 1.11x bonus)
 * 4. Higher investment → More capacity → More rewards
 */

import type {
  ComputeInvestment,
  ComputePeriod,
  SymbioticStakingPool,
} from "./types.js";
import { COMPUTE_INVESTMENT } from "./constants.js";

// ============ Investment Operations ============

/**
 * Record new compute investment
 */
export function recordComputeInvestment(
  investment: ComputeInvestment,
  lgmAmount: bigint,
  currentBlock: bigint,
): { investment: ComputeInvestment; error?: string } {
  // Validate amount
  if (lgmAmount < COMPUTE_INVESTMENT.MIN_PERIOD_INVESTMENT) {
    return {
      investment,
      error: `Minimum investment is ${COMPUTE_INVESTMENT.MIN_PERIOD_INVESTMENT}`,
    };
  }

  if (lgmAmount > COMPUTE_INVESTMENT.MAX_PERIOD_INVESTMENT) {
    return {
      investment,
      error: `Maximum investment per period is ${COMPUTE_INVESTMENT.MAX_PERIOD_INVESTMENT}`,
    };
  }

  const updated = { ...investment };
  updated.totalInvested = investment.totalInvested + lgmAmount;

  // Check for active period
  const activePeriod = updated.periods.find((p) => p.status === "active");

  if (activePeriod) {
    // Add to existing period
    activePeriod.lgmInvested = activePeriod.lgmInvested + lgmAmount;
  } else {
    // Create new period
    const newPeriod: ComputePeriod = {
      periodId: updated.currentPeriodId,
      startBlock: currentBlock,
      endBlock: currentBlock + COMPUTE_INVESTMENT.PERIOD_BLOCKS,
      lgmInvested: lgmAmount,
      computeUnitsAcquired: calculateComputeUnits(lgmAmount),
      verificationsPerformed: 0,
      verificationAccuracy: 0,
      rewardsGenerated: 0n,
      roi: 0,
      status: "active",
    };

    updated.periods = [...updated.periods, newPeriod];
    updated.currentPeriodId = updated.currentPeriodId + 1;
  }

  return { investment: updated };
}

/**
 * Calculate compute units from LGM investment
 */
function calculateComputeUnits(lgmAmount: bigint): bigint {
  return lgmAmount / COMPUTE_INVESTMENT.LGM_PER_COMPUTE_UNIT;
}

/**
 * Record verification work in current period
 */
export function recordVerificationWork(
  investment: ComputeInvestment,
  verificationsPerformed: number,
  correctVerifications: number,
): ComputeInvestment {
  const activePeriod = investment.periods.find((p) => p.status === "active");

  if (!activePeriod) {
    return investment;
  }

  activePeriod.verificationsPerformed += verificationsPerformed;
  const totalCorrect =
    activePeriod.verificationAccuracy *
      (activePeriod.verificationsPerformed - verificationsPerformed) +
    correctVerifications;
  activePeriod.verificationAccuracy =
    activePeriod.verificationsPerformed > 0
      ? totalCorrect / activePeriod.verificationsPerformed
      : 0;

  // Update efficiency
  const updated = { ...investment };
  updated.verificationContribution += verificationsPerformed;

  return updated;
}

/**
 * Record rewards generated in period
 */
export function recordRewardsGenerated(
  investment: ComputeInvestment,
  rewards: bigint,
): ComputeInvestment {
  const activePeriod = investment.periods.find((p) => p.status === "active");

  if (!activePeriod) {
    return investment;
  }

  activePeriod.rewardsGenerated = activePeriod.rewardsGenerated + rewards;

  return investment;
}

// ============ Period Settlement ============

/**
 * Check if period should be settled
 */
export function shouldSettlePeriod(
  investment: ComputeInvestment,
  currentBlock: bigint,
): boolean {
  const activePeriod = investment.periods.find((p) => p.status === "active");

  if (!activePeriod) return false;

  return currentBlock >= activePeriod.endBlock;
}

/**
 * Settle completed compute period
 */
export function settlePeriod(
  investment: ComputeInvestment,
  currentBlock: bigint,
): {
  investment: ComputeInvestment;
  settledPeriod?: ComputePeriod;
  contributionCredit: bigint;
} {
  const activePeriodIndex = investment.periods.findIndex(
    (p) => p.status === "active",
  );

  if (activePeriodIndex === -1) {
    return { investment, contributionCredit: 0n };
  }

  const activePeriod = investment.periods[activePeriodIndex];

  if (currentBlock < activePeriod.endBlock) {
    return { investment, contributionCredit: 0n };
  }

  // Calculate ROI
  const roi =
    activePeriod.lgmInvested > 0n
      ? Number(activePeriod.rewardsGenerated) /
          Number(activePeriod.lgmInvested) -
        1
      : 0;

  // Mark as completed
  const completedPeriod: ComputePeriod = {
    ...activePeriod,
    endBlock: currentBlock,
    roi,
    status: "completed",
  };

  // Calculate contribution credit with multiplier
  const contributionCredit = calculateContributionCredit(completedPeriod);

  // Update investment
  const updatedPeriods = [...investment.periods];
  updatedPeriods[activePeriodIndex] = completedPeriod;

  // Update efficiency metrics
  const totalRewards = updatedPeriods
    .filter((p) => p.status === "completed")
    .reduce((sum, p) => sum + p.rewardsGenerated, 0n);
  const computeEfficiency =
    investment.totalInvested > 0n
      ? Number(totalRewards) / Number(investment.totalInvested)
      : 0;

  return {
    investment: {
      ...investment,
      periods: updatedPeriods,
      computeEfficiency,
    },
    settledPeriod: completedPeriod,
    contributionCredit,
  };
}

/**
 * Calculate contribution credit with multiplier
 */
function calculateContributionCredit(period: ComputePeriod): bigint {
  // Base contribution is LGM invested
  let credit = period.lgmInvested;

  // Apply compute contribution multiplier (1.11x) using BPS to avoid Number() precision loss
  const contributionBps = BigInt(
    Math.round(COMPUTE_INVESTMENT.CONTRIBUTION_MULTIPLIER * 10000),
  );
  credit = (credit * contributionBps) / 10000n;

  // Apply accuracy bonus if above threshold (1.11x via BPS)
  if (
    period.verificationAccuracy >= COMPUTE_INVESTMENT.ACCURACY_BONUS_THRESHOLD
  ) {
    const accuracyBps = BigInt(
      Math.round(COMPUTE_INVESTMENT.ACCURACY_BONUS_MULTIPLIER * 10000),
    );
    credit = (credit * accuracyBps) / 10000n;
  }

  return credit;
}

// ============ Analysis ============

/**
 * Get compute investment summary
 */
export function getComputeSummary(
  investment: ComputeInvestment,
  currentBlock: bigint,
): {
  totalInvested: bigint;
  totalRewards: bigint;
  overallROI: number;
  activePeriod?: {
    invested: bigint;
    blocksRemaining: bigint;
    verificationsPerformed: number;
    accuracy: number;
    rewardsGenerated: bigint;
  };
  completedPeriods: number;
  averageAccuracy: number;
  computeEfficiency: number;
} {
  const activePeriod = investment.periods.find((p) => p.status === "active");
  const completedPeriods = investment.periods.filter(
    (p) => p.status === "completed",
  );

  const totalRewards = investment.periods.reduce(
    (sum, p) => sum + p.rewardsGenerated,
    0n,
  );
  const overallROI =
    investment.totalInvested > 0n
      ? Number(totalRewards) / Number(investment.totalInvested) - 1
      : 0;

  const totalVerifications = completedPeriods.reduce(
    (sum, p) => sum + p.verificationsPerformed,
    0,
  );
  const totalCorrect = completedPeriods.reduce(
    (sum, p) => sum + p.verificationsPerformed * p.verificationAccuracy,
    0,
  );
  const averageAccuracy =
    totalVerifications > 0 ? totalCorrect / totalVerifications : 0;

  return {
    totalInvested: investment.totalInvested,
    totalRewards,
    overallROI,
    activePeriod: activePeriod
      ? {
          invested: activePeriod.lgmInvested,
          blocksRemaining:
            activePeriod.endBlock > currentBlock
              ? activePeriod.endBlock - currentBlock
              : 0n,
          verificationsPerformed: activePeriod.verificationsPerformed,
          accuracy: activePeriod.verificationAccuracy,
          rewardsGenerated: activePeriod.rewardsGenerated,
        }
      : undefined,
    completedPeriods: completedPeriods.length,
    averageAccuracy,
    computeEfficiency: investment.computeEfficiency,
  };
}

/**
 * Project returns for potential investment
 */
export function projectInvestmentReturns(
  investment: ComputeInvestment,
  proposedInvestment: bigint,
  expectedAccuracy: number = 0.85,
): {
  projectedComputeUnits: bigint;
  projectedVerifications: number;
  projectedRewards: bigint;
  projectedROI: number;
  contributionCredit: bigint;
} {
  const computeUnits = calculateComputeUnits(proposedInvestment);

  // Project verifications based on compute units (1 verification per 1000 units baseline)
  const projectedVerifications = Math.floor(Number(computeUnits) / 1000);

  // Project rewards based on historical efficiency
  const avgRewardPerVerification =
    investment.verificationContribution > 0
      ? investment.periods.reduce((s, p) => s + p.rewardsGenerated, 0n) /
        BigInt(investment.verificationContribution)
      : 1n * 10n ** 18n; // Default 1 LGM per verification

  const projectedRewards =
    avgRewardPerVerification * BigInt(projectedVerifications);
  const projectedROI =
    Number(projectedRewards) / Number(proposedInvestment) - 1;

  // Calculate contribution credit using BPS to avoid Number() precision loss
  const projContribBps = BigInt(
    Math.round(COMPUTE_INVESTMENT.CONTRIBUTION_MULTIPLIER * 10000),
  );
  let contributionCredit = (proposedInvestment * projContribBps) / 10000n;

  if (expectedAccuracy >= COMPUTE_INVESTMENT.ACCURACY_BONUS_THRESHOLD) {
    const projAccBps = BigInt(
      Math.round(COMPUTE_INVESTMENT.ACCURACY_BONUS_MULTIPLIER * 10000),
    );
    contributionCredit = (contributionCredit * projAccBps) / 10000n;
  }

  return {
    projectedComputeUnits: computeUnits,
    projectedVerifications,
    projectedRewards,
    projectedROI,
    contributionCredit,
  };
}

// ============ Agent Contribution Analysis ============

/**
 * Calculate agent's total contribution including compute
 */
export function calculateAgentTotalContribution(
  baseStake: bigint,
  investment: ComputeInvestment,
): {
  baseStake: bigint;
  computeCredit: bigint;
  totalContribution: bigint;
  computeMultiplier: number;
} {
  // Sum up all contribution credits from completed periods
  const computeCredit = investment.periods
    .filter((p) => p.status === "completed")
    .reduce((sum, p) => {
      const credit = calculateContributionCredit(p);
      return sum + credit;
    }, 0n);

  const totalContribution = baseStake + computeCredit;
  const computeMultiplier = Number(totalContribution) / Number(baseStake);

  return {
    baseStake,
    computeCredit,
    totalContribution,
    computeMultiplier,
  };
}
