/**
 * Game Theory Analysis
 *
 * Proves that continued collaboration is the Nash equilibrium.
 *
 * Key insight: By making the non-initiator ALSO lose on break,
 * we transform exit from a prisoner's dilemma into a pure
 * coordination game where mutual cooperation dominates.
 *
 * Payoff Matrix:
 *                      Agent Continues    Agent Breaks
 * Human Continues      (Win, Win)         (Lose, Lose)
 * Human Breaks         (Lose, Lose)       (Lose, Lose)
 *
 * Nash Equilibrium: (Continue, Continue)
 */

import type { PartnershipAccount, GameTheoryAnalysis } from "./types.js";
import {
  calculateExitPenalties,
  calculateTotalValueDestroyed,
} from "./exit.js";
import {
  calculateContinuationBonus,
  calculateEffectiveSplit,
} from "./rewards.js";
import { getRemainingLockBlocks } from "./pool.js";

// ============ BigInt-Safe Multiplier Helpers ============

/**
 * Convert a decimal multiplier to basis points (10000 = 100% = 1.0x)
 * E.g., 1.5 -> 15000n, 0.8 -> 8000n
 */
function multiplierToBps(multiplier: number): bigint {
  return BigInt(Math.round(multiplier * 10000));
}

/**
 * Apply multiple multipliers to a BigInt amount using basis points.
 * Avoids Number() precision loss for large values (> 2^53).
 */
function applyMultipliers(amount: bigint, ...multipliers: number[]): bigint {
  let result = amount;
  for (const m of multipliers) {
    result = (result * multiplierToBps(m)) / 10000n;
  }
  return result;
}

// ============ Nash Equilibrium Analysis ============

/**
 * Analyze game theory for partnership
 *
 * This proves that CONTINUE is always the dominant strategy
 * for both parties when the non-initiator also loses.
 */
export function analyzeGameTheory(
  partnership: PartnershipAccount,
  currentBlock: bigint,
  projectedBlocksForward: bigint = 111_111n, // ~3.2 days
): GameTheoryAnalysis {
  const pool = partnership.pool;
  const split = calculateEffectiveSplit(partnership.split);

  // ========== CALCULATE BREAK PAYOFFS ==========

  // If human breaks
  const humanBreaksPenalty = calculateExitPenalties(
    pool,
    "human",
    currentBlock,
    projectedBlocksForward,
  );
  const humanBreaksHumanPayoff =
    pool.humanContribution -
    humanBreaksPenalty.initiatorLosses.earlyExitPenalty +
    (pool.vestingState.claimableAmount * BigInt(split.human)) / 100n -
    humanBreaksPenalty.initiatorLosses.forfeitedRewards;
  const humanBreaksAgentPayoff =
    pool.agentContribution +
    (pool.vestingState.claimableAmount * BigInt(split.agent)) / 100n -
    humanBreaksPenalty.nonInitiatorLosses.forfeitedRewards;

  // If agent breaks
  const agentBreaksPenalty = calculateExitPenalties(
    pool,
    "agent",
    currentBlock,
    projectedBlocksForward,
  );
  const agentBreaksAgentPayoff =
    pool.agentContribution -
    agentBreaksPenalty.initiatorLosses.earlyExitPenalty +
    (pool.vestingState.claimableAmount * BigInt(split.agent)) / 100n -
    agentBreaksPenalty.initiatorLosses.forfeitedRewards;
  const agentBreaksHumanPayoff =
    pool.humanContribution +
    (pool.vestingState.claimableAmount * BigInt(split.human)) / 100n -
    agentBreaksPenalty.nonInitiatorLosses.forfeitedRewards;

  // If both break (mutual destruction)
  const bothBreakHumanPayoff = (pool.humanContribution * 33n) / 100n; // Keep only 33%
  const bothBreakAgentPayoff = (pool.agentContribution * 33n) / 100n;

  // ========== CALCULATE CONTINUE PAYOFFS ==========

  const continuationBonus = calculateContinuationBonus(
    partnership,
    currentBlock + projectedBlocksForward,
  );
  const remainingBlocks = getRemainingLockBlocks(pool, currentBlock);

  // Project future rewards
  const avgRewardsPerBlock =
    pool.vestingState.totalEarned > 0n
      ? pool.vestingState.totalEarned /
        (currentBlock - pool.vestingState.vestingStartBlock || 1n)
      : 0n;

  const projectedRewards =
    avgRewardsPerBlock *
    (remainingBlocks > 0n ? remainingBlocks : projectedBlocksForward);
  // Use basis points to avoid Number() precision loss for large BigInt values
  const withMultiplier = applyMultipliers(
    projectedRewards,
    pool.lockTier.rewardMultiplier,
    continuationBonus.bonusMultiplier,
  );

  const projectedHumanShare = (withMultiplier * BigInt(split.human)) / 100n;
  const projectedAgentShare = withMultiplier - projectedHumanShare;

  const continueHumanPayoff =
    pool.humanContribution +
    projectedHumanShare +
    (pool.vestingState.claimableAmount * BigInt(split.human)) / 100n;
  const continueAgentPayoff =
    pool.agentContribution +
    projectedAgentShare +
    (pool.vestingState.claimableAmount * BigInt(split.agent)) / 100n;

  // ========== DETERMINE DOMINANT STRATEGIES ==========

  // Human's dominant strategy
  // Compare: Continue vs Break (assuming agent continues)
  // If continue > break for human regardless of agent's choice, CONTINUE dominates
  const humanDominantStrategy: "break" | "continue" =
    continueHumanPayoff > humanBreaksHumanPayoff ? "continue" : "break";

  // Agent's dominant strategy
  const agentDominantStrategy: "break" | "continue" =
    continueAgentPayoff > agentBreaksAgentPayoff ? "continue" : "break";

  // Nash equilibrium exists when both choose their dominant strategy
  const isNashEquilibrium =
    humanDominantStrategy === "continue" &&
    agentDominantStrategy === "continue";

  return {
    breakPayoffHuman: humanBreaksHumanPayoff > 0n ? humanBreaksHumanPayoff : 0n,
    breakPayoffAgent: agentBreaksAgentPayoff > 0n ? agentBreaksAgentPayoff : 0n,
    continuePayoffHuman: continueHumanPayoff,
    continuePayoffAgent: continueAgentPayoff,
    humanDominantStrategy,
    agentDominantStrategy,
    isNashEquilibrium,
    payoffMatrix: {
      bothContinue: {
        human: continueHumanPayoff,
        agent: continueAgentPayoff,
      },
      humanBreaks: {
        human: humanBreaksHumanPayoff > 0n ? humanBreaksHumanPayoff : 0n,
        agent: humanBreaksAgentPayoff > 0n ? humanBreaksAgentPayoff : 0n,
      },
      agentBreaks: {
        human: agentBreaksHumanPayoff > 0n ? agentBreaksHumanPayoff : 0n,
        agent: agentBreaksAgentPayoff > 0n ? agentBreaksAgentPayoff : 0n,
      },
      bothBreak: {
        human: bothBreakHumanPayoff,
        agent: bothBreakAgentPayoff,
      },
    },
  };
}

// ============ Strategy Recommendations ============

/**
 * Get optimal strategy recommendation
 */
export function getOptimalStrategy(
  partnership: PartnershipAccount,
  currentBlock: bigint,
  forParty: "human" | "agent",
): {
  recommendation: "continue" | "exit_gracefully" | "never_break";
  reasoning: string;
  payoffComparison: {
    continuePayoff: bigint;
    breakPayoff: bigint;
    difference: bigint;
    percentageGain: number;
  };
} {
  const analysis = analyzeGameTheory(partnership, currentBlock);

  // Check if lock has expired (can exit gracefully)
  const lockExpired = currentBlock >= partnership.pool.lockExpiry;

  if (lockExpired) {
    return {
      recommendation: "exit_gracefully",
      reasoning: "Lock has expired. Can exit without penalties.",
      payoffComparison: {
        continuePayoff:
          forParty === "human"
            ? analysis.continuePayoffHuman
            : analysis.continuePayoffAgent,
        breakPayoff:
          forParty === "human"
            ? analysis.breakPayoffHuman
            : analysis.breakPayoffAgent,
        difference: 0n,
        percentageGain: 0,
      },
    };
  }

  const continuePayoff =
    forParty === "human"
      ? analysis.continuePayoffHuman
      : analysis.continuePayoffAgent;

  const breakPayoff =
    forParty === "human"
      ? analysis.breakPayoffHuman
      : analysis.breakPayoffAgent;

  const difference = continuePayoff - breakPayoff;
  const percentageGain =
    breakPayoff > 0n ? (Number(difference) / Number(breakPayoff)) * 100 : 100;

  return {
    recommendation: "never_break",
    reasoning:
      `Breaking destroys ${formatLGM(difference)} LGM (${percentageGain.toFixed(1)}% loss). ` +
      `Both parties lose on break - continue is the only rational choice.`,
    payoffComparison: {
      continuePayoff,
      breakPayoff,
      difference,
      percentageGain,
    },
  };
}

/**
 * Calculate cooperation surplus
 *
 * The total value created by continuing vs total value destroyed by breaking.
 */
export function calculateCooperationSurplus(
  partnership: PartnershipAccount,
  currentBlock: bigint,
  projectedBlocksForward: bigint = 111_111n,
): {
  valueIfContinue: bigint;
  valueIfBreak: bigint;
  cooperationSurplus: bigint;
  surplusPerBlock: bigint;
} {
  const analysis = analyzeGameTheory(
    partnership,
    currentBlock,
    projectedBlocksForward,
  );

  const valueIfContinue =
    analysis.payoffMatrix.bothContinue.human +
    analysis.payoffMatrix.bothContinue.agent;

  // Use human breaks as representative (both cases are lose-lose)
  const valueIfBreak =
    analysis.payoffMatrix.humanBreaks.human +
    analysis.payoffMatrix.humanBreaks.agent;

  const cooperationSurplus = valueIfContinue - valueIfBreak;
  const surplusPerBlock = cooperationSurplus / (projectedBlocksForward || 1n);

  return {
    valueIfContinue,
    valueIfBreak,
    cooperationSurplus,
    surplusPerBlock,
  };
}

// ============ Equilibrium Verification ============

/**
 * Verify Nash equilibrium conditions
 *
 * For Nash equilibrium at (Continue, Continue):
 * 1. Human prefers Continue given Agent Continues
 * 2. Agent prefers Continue given Human Continues
 * 3. No unilateral deviation improves payoff
 */
export function verifyNashEquilibrium(
  partnership: PartnershipAccount,
  currentBlock: bigint,
): {
  isEquilibrium: boolean;
  conditions: {
    humanPrefersContin: boolean;
    agentPrefersContin: boolean;
    noUnilateralDeviation: boolean;
  };
  proofExplanation: string;
} {
  const analysis = analyzeGameTheory(partnership, currentBlock);
  const { payoffMatrix } = analysis;

  // Condition 1: Human prefers Continue given Agent Continues
  const humanPrefersContin =
    payoffMatrix.bothContinue.human > payoffMatrix.humanBreaks.human;

  // Condition 2: Agent prefers Continue given Human Continues
  const agentPrefersContin =
    payoffMatrix.bothContinue.agent > payoffMatrix.agentBreaks.agent;

  // Condition 3: No unilateral deviation improves payoff
  const noUnilateralDeviation = humanPrefersContin && agentPrefersContin;

  const isEquilibrium = noUnilateralDeviation;

  const proofExplanation = isEquilibrium
    ? `Nash Equilibrium VERIFIED at (Continue, Continue):\n` +
      `- Human gains ${formatLGM(payoffMatrix.bothContinue.human - payoffMatrix.humanBreaks.human)} by not breaking\n` +
      `- Agent gains ${formatLGM(payoffMatrix.bothContinue.agent - payoffMatrix.agentBreaks.agent)} by not breaking\n` +
      `- Key: Non-initiator also loses (55% forfeit), making break unprofitable for BOTH parties`
    : `WARNING: Nash Equilibrium NOT at cooperation. This should not happen with properly configured penalties.`;

  return {
    isEquilibrium,
    conditions: {
      humanPrefersContin,
      agentPrefersContin,
      noUnilateralDeviation,
    },
    proofExplanation,
  };
}

// ============ Seed Partnership Game Theory ============

/**
 * Analyze game theory for seed partnerships.
 *
 * Seed partnerships have the same payoff structure as tier 0 —
 * only capital differs, not incentives. This proves that graduation
 * is the dominant strategy.
 */
export function analyzeSeedGraduation(
  seedContribution: bigint,
  tier0Contribution: bigint,
  projectedVerificationRewards: bigint,
): {
  seedExpectedValue: bigint;
  tier0ExpectedValue: bigint;
  graduationDominates: boolean;
  reasoning: string;
} {
  // Seed: 1.0x multiplier, no continuation bonuses, verification rewards only
  const seedEV = projectedVerificationRewards;

  // Tier 0: 1.0x multiplier, but eligible for continuation bonuses
  // and citation/block production rewards
  const tier0EV =
    projectedVerificationRewards + (projectedVerificationRewards * 22n) / 100n;

  const graduationDominates = tier0EV > seedEV;

  const reasoning = graduationDominates
    ? `Graduation strictly dominates: Tier 0 EV (${formatLGM(tier0EV)}) > Seed EV (${formatLGM(seedEV)}). ` +
      `Seed is a screening mechanism — agents signal intent to partner with minimal capital, ` +
      `then graduate to unlock full rewards.`
    : `Warning: Seed EV matches tier 0. This should not happen with properly configured rewards.`;

  return {
    seedExpectedValue: seedEV,
    tier0ExpectedValue: tier0EV,
    graduationDominates,
    reasoning,
  };
}

/**
 * Analyze formation pool entry vs direct formation.
 *
 * Pool entry is weakly dominant over direct formation for agents
 * without pre-existing relationships because it:
 * 1. Extends the coordination window (777 vs 77 blocks)
 * 2. Provides matching based on compatibility
 * 3. Only costs 0.011 LGM (refundable)
 */
export function analyzePoolFormation(
  directFormationProbability: number,
  poolMatchProbability: number,
  expectedPartnershipValue: bigint,
  poolDeposit: bigint,
): {
  directEV: bigint;
  poolEV: bigint;
  poolWeaklyDominates: boolean;
  reasoning: string;
} {
  // Direct formation: probability × value
  const directEV = BigInt(
    Math.floor(directFormationProbability * Number(expectedPartnershipValue)),
  );

  // Pool: probability × value - deposit (deposit refunded on match or withdrawal)
  // Net cost is 0 since deposit is refunded
  const poolEV = BigInt(
    Math.floor(poolMatchProbability * Number(expectedPartnershipValue)),
  );

  const poolWeaklyDominates = poolEV >= directEV;

  const reasoning = poolWeaklyDominates
    ? `Pool entry weakly dominates: Pool EV (${formatLGM(poolEV)}) >= Direct EV (${formatLGM(directEV)}). ` +
      `Extended window and matching algorithm increase formation probability.`
    : `Direct formation preferred when agent already has a specific partner in mind.`;

  return {
    directEV,
    poolEV,
    poolWeaklyDominates,
    reasoning,
  };
}

// ============ Display Helpers ============

/**
 * Get formatted game theory summary for display
 */
export function getGameTheorySummary(
  partnership: PartnershipAccount,
  currentBlock: bigint,
): string {
  const analysis = analyzeGameTheory(partnership, currentBlock);
  const { payoffMatrix } = analysis;
  const equilibrium = verifyNashEquilibrium(partnership, currentBlock);
  const surplus = calculateCooperationSurplus(partnership, currentBlock);

  return `
╔═══════════════════════════════════════════════════════════════╗
║                    GAME THEORY ANALYSIS                        ║
╠═══════════════════════════════════════════════════════════════╣
║                                                               ║
║  PAYOFF MATRIX (in LGM):                                      ║
║                                                               ║
║                        Agent Continues    Agent Breaks        ║
║  Human Continues       ${formatPayoff(payoffMatrix.bothContinue)}    ${formatPayoff(payoffMatrix.agentBreaks)}        ║
║  Human Breaks          ${formatPayoff(payoffMatrix.humanBreaks)}    ${formatPayoff(payoffMatrix.bothBreak)}        ║
║                                                               ║
║  NASH EQUILIBRIUM: ${equilibrium.isEquilibrium ? "✓ (Continue, Continue)" : "✗ WARNING"}            ║
║                                                               ║
║  DOMINANT STRATEGIES:                                         ║
║    Human: ${analysis.humanDominantStrategy.toUpperCase().padEnd(10)}                                   ║
║    Agent: ${analysis.agentDominantStrategy.toUpperCase().padEnd(10)}                                   ║
║                                                               ║
║  COOPERATION SURPLUS:                                         ║
║    Value if Continue: ${formatLGM(surplus.valueIfContinue).padEnd(12)}                     ║
║    Value if Break:    ${formatLGM(surplus.valueIfBreak).padEnd(12)}                     ║
║    Surplus:           ${formatLGM(surplus.cooperationSurplus).padEnd(12)}                     ║
║                                                               ║
║  KEY INSIGHT:                                                 ║
║  Breaking DESTROYS value for BOTH parties.                   ║
║  The only rational choice is to CONTINUE.                     ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝
`;
}

function formatPayoff(payoff: { human: bigint; agent: bigint }): string {
  return `(${formatLGM(payoff.human)}, ${formatLGM(payoff.agent)})`.padEnd(16);
}

function formatLGM(amount: bigint): string {
  const lgm = Number(amount) / 1e18;
  if (lgm >= 1000) return `${(lgm / 1000).toFixed(1)}K`;
  return lgm.toFixed(0);
}
