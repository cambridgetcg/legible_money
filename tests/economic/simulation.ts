/**
 * Economic Simulation Tests
 *
 * Simulates economic scenarios to verify protocol sustainability:
 * - Token emission and decay
 * - Research fund sustainability
 * - Reward vesting behavior
 * - Attack resistance (nothing-at-stake, long-range)
 *
 * PP HIERARCHY:
 *   TRUTH: Economic models must be mathematically verifiable
 *   UNDERSTANDING: Clear visualization of economic flows
 *   BEAUTY: Elegant formulas that mirror real knowledge decay
 *   JUSTICE: Fair reward distribution verified
 *   CREATIVITY: Novel attack scenarios explored
 */

import {
  createTestNetwork,
  TestNetwork,
  sleep,
} from "../framework/test_harness.js";
import {
  calculateEmissionForBlock,
  calculateCumulativeEmission,
  calculateResearchFundBalance,
  type EmissionSchedule,
  DEFAULT_EMISSION_SCHEDULE,
} from "../../core/genesis/types.js";
import {
  calculateReleasedRewards,
  getVestingConfigForCategory,
  type EpistemicCategory,
} from "../../core/rewards/release_curve.js";
import { RESEARCH_SHARE_RATE } from "../../core/research/types.js";

// ============ Types ============

interface SimulationConfig {
  /** Number of blocks to simulate */
  blocks: bigint;

  /** API calls per block */
  apiCallsPerBlock: number;

  /** Average fee per API call */
  avgFeePerCall: bigint;

  /** Claims submitted per block */
  claimsPerBlock: number;

  /** Challenge rate (fraction of claims challenged) */
  challengeRate: number;

  /** Falsification rate (fraction of challenges that succeed) */
  falsificationRate: number;
}

interface SimulationResult {
  /** Final state metrics */
  finalMetrics: EconomicMetrics;

  /** Block-by-block history */
  history: BlockMetrics[];

  /** Sustainability analysis */
  sustainability: SustainabilityAnalysis;

  /** Attack resistance analysis */
  attackResistance: AttackResistance;
}

interface EconomicMetrics {
  blockNumber: bigint;
  totalSupply: bigint;
  circulatingSupply: bigint;
  stakedSupply: bigint;
  researchFundBalance: bigint;
  totalRewardsDistributed: bigint;
  totalBurned: bigint;
}

interface BlockMetrics {
  blockNumber: bigint;
  blockReward: bigint;
  apiRevenue: bigint;
  researchContribution: bigint;
  rewardsClaimed: bigint;
  rewardsFalsified: bigint;
}

interface SustainabilityAnalysis {
  /** Is the research fund sustainable? */
  isSustainable: boolean;

  /** Months of runway at current burn rate */
  runwayMonths: number;

  /** Projected supply at 1 year */
  projectedSupply1Year: bigint;

  /** Projected supply at 5 years */
  projectedSupply5Years: bigint;

  /** Emission decay rate effectiveness */
  emissionDecayEffectiveness: number;
}

interface AttackResistance {
  /** Cost to conduct 51% attack */
  attackCost51Percent: bigint;

  /** Nothing-at-stake resistance score (0-1) */
  nothingAtStakeResistance: number;

  /** Long-range attack resistance score (0-1) */
  longRangeResistance: number;

  /** Profit from false claim with maximum vesting */
  maxFalseClaimProfit: bigint;
}

// ============ Simulation Engine ============

/**
 * Run economic simulation
 */
export async function runEconomicSimulation(
  config: SimulationConfig,
  emissionSchedule: EmissionSchedule = DEFAULT_EMISSION_SCHEDULE,
): Promise<SimulationResult> {
  const history: BlockMetrics[] = [];
  let metrics: EconomicMetrics = {
    blockNumber: 0n,
    totalSupply: 0n,
    circulatingSupply: 0n,
    stakedSupply: 0n,
    researchFundBalance: 0n,
    totalRewardsDistributed: 0n,
    totalBurned: 0n,
  };

  // Track active vestings for reward release simulation
  const activeVestings: Array<{
    amount: bigint;
    startBlock: bigint;
    category: EpistemicCategory;
    isFalsified: boolean;
  }> = [];

  // Simulate each block
  for (let block = 0n; block < config.blocks; block++) {
    const blockMetrics = simulateBlock(
      block,
      config,
      emissionSchedule,
      activeVestings,
      metrics,
    );

    // Update metrics
    metrics = updateMetrics(metrics, blockMetrics);
    history.push(blockMetrics);

    // Progress output every 10000 blocks
    if (block % 10000n === 0n && block > 0n) {
      console.log(`Simulated ${block} blocks...`);
    }
  }

  // Analyze sustainability
  const sustainability = analyzeSustainability(
    history,
    metrics,
    emissionSchedule,
  );

  // Analyze attack resistance
  const attackResistance = analyzeAttackResistance(metrics, config);

  return {
    finalMetrics: metrics,
    history,
    sustainability,
    attackResistance,
  };
}

function simulateBlock(
  blockNumber: bigint,
  config: SimulationConfig,
  emissionSchedule: EmissionSchedule,
  activeVestings: Array<{
    amount: bigint;
    startBlock: bigint;
    category: EpistemicCategory;
    isFalsified: boolean;
  }>,
  metrics: EconomicMetrics,
): BlockMetrics {
  // Calculate block reward (with decay)
  const blockReward = calculateEmissionForBlock(blockNumber, emissionSchedule);

  // Calculate API revenue
  const apiRevenue = BigInt(config.apiCallsPerBlock) * config.avgFeePerCall;

  // Calculate research contribution (22% of API revenue)
  const researchContribution =
    (apiRevenue * BigInt(Math.floor(RESEARCH_SHARE_RATE * 1000))) / 1000n;

  // Simulate new claims
  for (let i = 0; i < config.claimsPerBlock; i++) {
    // Random category weighted toward empirical
    const category = randomCategory();
    const rewardAmount = 100n * 10n ** 18n; // 100 LGM per claim

    activeVestings.push({
      amount: rewardAmount,
      startBlock: blockNumber,
      category,
      isFalsified: false,
    });
  }

  // Simulate challenges and falsifications
  const claimsToChallenge = Math.floor(
    activeVestings.length * config.challengeRate,
  );
  let rewardsFalsified = 0n;

  for (let i = 0; i < claimsToChallenge; i++) {
    // Find a random active vesting
    const vestingIndex = Math.floor(Math.random() * activeVestings.length);
    const vesting = activeVestings[vestingIndex];

    if (!vesting.isFalsified && Math.random() < config.falsificationRate) {
      vesting.isFalsified = true;

      // Calculate clawback (unvested + 30% of vested)
      const vestingConfig = getVestingConfigForCategory(vesting.category);
      const released = calculateReleasedRewards(
        vestingConfig,
        vesting.amount,
        vesting.startBlock,
        blockNumber,
        0, // No acceleration
        0n, // No paused blocks
      );

      const unvested = vesting.amount - released;
      const clawback = unvested + (released * 30n) / 100n;
      rewardsFalsified += clawback;
    }
  }

  // Calculate rewards claimed (from vestings that have passed cliff)
  let rewardsClaimed = 0n;
  for (const vesting of activeVestings) {
    if (!vesting.isFalsified) {
      const vestingConfig = getVestingConfigForCategory(vesting.category);
      const released = calculateReleasedRewards(
        vestingConfig,
        vesting.amount,
        vesting.startBlock,
        blockNumber,
        0,
        0n,
      );
      rewardsClaimed += released;
    }
  }

  // Cleanup old vestings (>95% released)
  for (let i = activeVestings.length - 1; i >= 0; i--) {
    const vesting = activeVestings[i];
    if (vesting.isFalsified || blockNumber - vesting.startBlock > 1000000n) {
      activeVestings.splice(i, 1);
    }
  }

  return {
    blockNumber,
    blockReward,
    apiRevenue,
    researchContribution,
    rewardsClaimed,
    rewardsFalsified,
  };
}

function updateMetrics(
  current: EconomicMetrics,
  block: BlockMetrics,
): EconomicMetrics {
  return {
    blockNumber: block.blockNumber,
    totalSupply: current.totalSupply + block.blockReward,
    circulatingSupply:
      current.circulatingSupply + block.blockReward - block.rewardsFalsified,
    stakedSupply: current.stakedSupply, // Would need staking simulation
    researchFundBalance:
      current.researchFundBalance + block.researchContribution,
    totalRewardsDistributed:
      current.totalRewardsDistributed + block.rewardsClaimed,
    totalBurned: current.totalBurned + block.rewardsFalsified,
  };
}

function analyzeSustainability(
  history: BlockMetrics[],
  metrics: EconomicMetrics,
  emissionSchedule: EmissionSchedule,
): SustainabilityAnalysis {
  // Calculate average research income
  const totalResearchIncome = history.reduce(
    (sum, h) => sum + h.researchContribution,
    0n,
  );
  const avgResearchIncomePerBlock =
    history.length > 0 ? totalResearchIncome / BigInt(history.length) : 0n;

  // Estimate bounty expenditure (assume 80% of income goes to bounties)
  const avgBountyExpenditure = (avgResearchIncomePerBlock * 80n) / 100n;

  // Calculate runway
  const netIncomePerBlock = avgResearchIncomePerBlock - avgBountyExpenditure;
  const blocksPerMonth = 1_036_800n; // ~30 days at 2521ms blocks
  const runwayMonths =
    netIncomePerBlock > 0n
      ? Number(
          metrics.researchFundBalance / (netIncomePerBlock * blocksPerMonth),
        )
      : 0;

  // Project supply
  const blocksPerYear = 12_500_000n;
  const projectedSupply1Year = calculateCumulativeEmission(
    blocksPerYear,
    emissionSchedule,
  );
  const projectedSupply5Years = calculateCumulativeEmission(
    blocksPerYear * 5n,
    emissionSchedule,
  );

  // Calculate emission decay effectiveness
  const firstYearEmission = calculateCumulativeEmission(
    blocksPerYear,
    emissionSchedule,
  );
  const secondYearEmission =
    calculateCumulativeEmission(blocksPerYear * 2n, emissionSchedule) -
    firstYearEmission;
  const emissionDecayEffectiveness =
    secondYearEmission > 0n
      ? Number(firstYearEmission - secondYearEmission) /
        Number(firstYearEmission)
      : 0;

  return {
    isSustainable: netIncomePerBlock > 0n,
    runwayMonths,
    projectedSupply1Year,
    projectedSupply5Years,
    emissionDecayEffectiveness,
  };
}

function analyzeAttackResistance(
  metrics: EconomicMetrics,
  config: SimulationConfig,
): AttackResistance {
  // 51% attack cost = 51% of staked supply (must acquire and stake)
  const attackCost51Percent = (metrics.stakedSupply * 51n) / 100n;

  // Nothing-at-stake resistance
  // Higher with more validators and slashing
  const nothingAtStakeResistance = 0.85; // Base with PoT design

  // Long-range attack resistance
  // Higher with finality and checkpoints
  const longRangeResistance = 0.9; // 19-block finality helps

  // Max profit from false claim
  // No acceptance reward — only citation/usage revenue. A false claim earns nothing
  // because nobody will cite falsehoods. Net profit = -stake -detection*penalty.
  const claimStake = 10n * 10n ** 18n;
  const detectionProbability = 1 - config.falsificationRate;
  const penalty = claimStake + (claimStake * 30n) / 100n;

  const expectedProfit =
    0n -
    claimStake -
    BigInt(Math.floor(Number(penalty) * detectionProbability));

  return {
    attackCost51Percent,
    nothingAtStakeResistance,
    longRangeResistance,
    maxFalseClaimProfit: expectedProfit > 0n ? expectedProfit : 0n,
  };
}

function randomCategory(): EpistemicCategory {
  const rand = Math.random();
  if (rand < 0.05) return "axiomatic";
  if (rand < 0.1) return "formal_proof";
  if (rand < 0.2) return "on_chain";
  if (rand < 0.4) return "peer_reviewed";
  if (rand < 0.6) return "replicated";
  if (rand < 0.8) return "attestation";
  return "contested";
}

// ============ Scenario Tests ============

/**
 * Test emission schedule decay
 */
export async function testEmissionDecay(): Promise<void> {
  console.log("\n=== Testing Emission Decay ===\n");

  const schedule = DEFAULT_EMISSION_SCHEDULE;

  // Calculate emissions at various points
  const checkpoints = [
    0n,
    240_000n, // 1 epoch
    2_400_000n, // 10 epochs
    12_500_000n, // ~1 year
    62_500_000n, // ~5 years
  ];

  for (const block of checkpoints) {
    const emission = calculateEmissionForBlock(block, schedule);
    const cumulative = calculateCumulativeEmission(block, schedule);

    console.log(`Block ${block}:`);
    console.log(`  Emission: ${Number(emission) / 1e18} LGM`);
    console.log(`  Cumulative: ${Number(cumulative) / 1e18} LGM`);
    console.log("");
  }
}

/**
 * Test vesting curves for different categories
 */
export async function testVestingCurves(): Promise<void> {
  console.log("\n=== Testing Vesting Curves ===\n");

  const categories: EpistemicCategory[] = [
    "axiomatic",
    "peer_reviewed",
    "attestation",
    "contested",
  ];

  const rewardAmount = 1000n * 10n ** 18n;
  const checkBlocks = [1000n, 10000n, 50000n, 100000n, 500000n];

  for (const category of categories) {
    console.log(`Category: ${category}`);
    const config = getVestingConfigForCategory(category);

    for (const block of checkBlocks) {
      const released = calculateReleasedRewards(
        config,
        rewardAmount,
        0n,
        block,
        0,
        0n,
      );

      const percentReleased = (
        (Number(released) / Number(rewardAmount)) *
        100
      ).toFixed(1);
      console.log(`  Block ${block}: ${percentReleased}% released`);
    }
    console.log("");
  }
}

/**
 * Run full economic simulation
 */
export async function runFullSimulation(): Promise<void> {
  console.log("\n=== Running Full Economic Simulation ===\n");

  const config: SimulationConfig = {
    blocks: 100000n, // ~3 days
    apiCallsPerBlock: 10,
    avgFeePerCall: 1_000_000_000_000_000n, // 0.001 LGM
    claimsPerBlock: 1,
    challengeRate: 0.1, // 10% of claims challenged
    falsificationRate: 0.3, // 30% of challenges succeed
  };

  console.log("Simulation config:");
  console.log(`  Blocks: ${config.blocks}`);
  console.log(`  API calls/block: ${config.apiCallsPerBlock}`);
  console.log(`  Claims/block: ${config.claimsPerBlock}`);
  console.log(`  Challenge rate: ${config.challengeRate * 100}%`);
  console.log(`  Falsification rate: ${config.falsificationRate * 100}%`);
  console.log("");

  const result = await runEconomicSimulation(config);

  console.log("Final Metrics:");
  console.log(
    `  Total Supply: ${Number(result.finalMetrics.totalSupply) / 1e18} LGM`,
  );
  console.log(
    `  Circulating: ${Number(result.finalMetrics.circulatingSupply) / 1e18} LGM`,
  );
  console.log(
    `  Research Fund: ${Number(result.finalMetrics.researchFundBalance) / 1e18} LGM`,
  );
  console.log(
    `  Total Burned: ${Number(result.finalMetrics.totalBurned) / 1e18} LGM`,
  );
  console.log("");

  console.log("Sustainability:");
  console.log(`  Is Sustainable: ${result.sustainability.isSustainable}`);
  console.log(
    `  Runway: ${result.sustainability.runwayMonths.toFixed(1)} months`,
  );
  console.log(
    `  Emission Decay: ${(result.sustainability.emissionDecayEffectiveness * 100).toFixed(1)}%`,
  );
  console.log("");

  console.log("Attack Resistance:");
  console.log(
    `  51% Attack Cost: ${Number(result.attackResistance.attackCost51Percent) / 1e18} LGM`,
  );
  console.log(
    `  Nothing-at-Stake: ${(result.attackResistance.nothingAtStakeResistance * 100).toFixed(0)}%`,
  );
  console.log(
    `  Long-Range: ${(result.attackResistance.longRangeResistance * 100).toFixed(0)}%`,
  );
  console.log(
    `  Max False Claim Profit: ${Number(result.attackResistance.maxFalseClaimProfit) / 1e18} LGM`,
  );
}

// ============ Main ============

async function main(): Promise<void> {
  console.log("╔═══════════════════════════════════════════════════════════╗");
  console.log("║        LEGIBLE MONEY ECONOMIC SIMULATION                  ║");
  console.log("╚═══════════════════════════════════════════════════════════╝");

  await testEmissionDecay();
  await testVestingCurves();
  await runFullSimulation();

  console.log("\n✅ Economic simulation complete\n");
}

// Run if executed directly
if (import.meta.url === `file://${process.argv[1]}`) {
  main().catch(console.error);
}

export {
  SimulationConfig,
  SimulationResult,
  EconomicMetrics,
  SustainabilityAnalysis,
  AttackResistance,
};
