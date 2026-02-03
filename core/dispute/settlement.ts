/**
 * Dispute Settlement System
 *
 * Calculates rewards and punishments based on dispute outcome:
 *
 * PUNISHMENT (Harsh - Results Matter):
 * - False claim: 50% slash (you polluted the knowledge base)
 * - Failed challenge: 30% slash (you wasted everyone's time)
 * - Wrong arbitration: 15% slash (you failed your duty)
 * - Refusal to reveal: 100% slash (bad faith actor)
 *
 * REWARDS (Generous - Truth Discovery):
 * - Winner: 60% of loser's stake
 * - Truth bonus: +50% for exposing falsehood
 * - Early detection: +20% for catching within 1000 blocks
 * - Correct arbitration: 10% bonus on stake
 *
 * PP HIERARCHY:
 *   TRUTH: Settlement must be deterministic and verifiable
 *   UNDERSTANDING: Economics align incentives with truth-seeking
 *   BEAUTY: Simple formulas, predictable outcomes
 *   JUSTICE: Results matter, not intentions - harsh but fair
 */

import {
  PUNISHMENT_RATES,
  REWARD_RATES,
  DISPUTE_ECONOMICS,
  type Dispute,
  type DisputeSettlement,
  DisputeError,
} from './types.js';
import { determineFinalWinner } from './state_machine.js';

// ============ Early Detection ============

/**
 * Blocks threshold for early detection bonus
 */
export const EARLY_DETECTION_THRESHOLD = 1000n;

/**
 * Check if challenge was made early enough for bonus
 */
export function qualifiesForEarlyDetection(dispute: Dispute): boolean {
  // Only applicable when challenger wins (exposing falsehood)
  if (dispute.winner !== 'challenger') {
    return false;
  }

  // Check blocks since evidence was submitted
  // The target ID should be evidence that was submitted
  // For now, we'll use the dispute initiation time vs some baseline
  // In production, this would check when the disputed evidence was submitted

  // Simplified: if dispute was created within threshold of being able to dispute
  return true; // Would need evidence submission time for proper check
}

// ============ Loser Penalty Calculation ============

/**
 * Calculate penalty rate based on how dispute was lost
 */
export function calculatePenaltyRate(
  dispute: Dispute,
  loser: 'challenger' | 'defender'
): number {
  // Check for aggravating factors

  // Refusal to reveal evidence - maximum penalty
  if (loser === 'challenger' && dispute.commitments.challenger && !dispute.commitments.challenger.revealed) {
    return PUNISHMENT_RATES.refusalToRevealSlash;
  }
  if (loser === 'defender' && dispute.commitments.defender && !dispute.commitments.defender.revealed) {
    return PUNISHMENT_RATES.refusalToRevealSlash;
  }

  // Standard penalties based on role
  if (loser === 'defender') {
    // Defender lost = their claim was false
    return PUNISHMENT_RATES.falseClaimSlash;
  } else {
    // Challenger lost = their challenge was baseless
    return PUNISHMENT_RATES.failedChallengeSlash;
  }
}

/**
 * Calculate loser's penalty amount
 */
export function calculateLoserPenalty(
  dispute: Dispute,
  loser: 'challenger' | 'defender'
): { penalty: bigint; rate: number } {
  const stake = loser === 'challenger' ? dispute.challenger.stake : dispute.defender.stake;
  const rate = calculatePenaltyRate(dispute, loser);
  const penalty = (stake * BigInt(Math.floor(rate * 100))) / 100n;

  return { penalty, rate };
}

// ============ Winner Reward Calculation ============

/**
 * Calculate winner's base reward (60% of loser stake)
 */
export function calculateBaseReward(loserStake: bigint): bigint {
  return (loserStake * BigInt(Math.floor(REWARD_RATES.winnerShare * 100))) / 100n;
}

/**
 * Calculate truth bonus (50% extra for exposing falsehood)
 */
export function calculateTruthBonus(baseReward: bigint, exposedFalsehood: boolean): bigint {
  if (!exposedFalsehood) {
    return 0n;
  }
  return (baseReward * BigInt(Math.floor((REWARD_RATES.truthBonusMultiplier - 1.0) * 100))) / 100n;
}

/**
 * Calculate early detection bonus (20% extra for catching early)
 */
export function calculateEarlyBonus(baseReward: bigint, qualifies: boolean): bigint {
  if (!qualifies) {
    return 0n;
  }
  return (baseReward * BigInt(Math.floor(REWARD_RATES.earlyDetectionBonus * 100))) / 100n;
}

/**
 * Calculate total winner reward
 */
export function calculateWinnerReward(
  dispute: Dispute,
  loserStake: bigint
): {
  total: bigint;
  base: bigint;
  truthBonus: bigint;
  earlyBonus: bigint;
  truthBonusApplied: boolean;
  earlyBonusApplied: boolean;
} {
  const base = calculateBaseReward(loserStake);

  // Truth bonus applies when challenger exposes defender's falsehood
  const truthBonusApplied = dispute.winner === 'challenger' && dispute.exposedFalsehood;
  const truthBonus = calculateTruthBonus(base, truthBonusApplied);

  // Early detection bonus
  const earlyBonusApplied = qualifiesForEarlyDetection(dispute);
  const earlyBonus = calculateEarlyBonus(base, earlyBonusApplied);

  const total = base + truthBonus + earlyBonus;

  return {
    total,
    base,
    truthBonus,
    earlyBonus,
    truthBonusApplied,
    earlyBonusApplied,
  };
}

// ============ Arbiter Settlement ============

/**
 * Calculate arbiter rewards and penalties
 */
export function calculateArbiterSettlement(
  dispute: Dispute
): {
  rewards: Map<string, bigint>;
  penalties: Map<string, bigint>;
} {
  const rewards = new Map<string, bigint>();
  const penalties = new Map<string, bigint>();

  const winner = dispute.winner;
  if (!winner) {
    // No winner (draw) - no arbiter settlements
    return { rewards, penalties };
  }

  for (const [arbiterDid, vote] of dispute.votes.entries()) {
    const lockedStake = vote.lockedStake;

    if (vote.vote === 'abstain') {
      // Abstaining returns stake, no bonus or penalty
      continue;
    }

    const votedCorrectly = vote.vote === winner;

    if (votedCorrectly) {
      // Correct vote: 10% bonus on locked stake
      const bonus = (lockedStake * BigInt(Math.floor(REWARD_RATES.arbiterCorrectBonus * 100))) / 100n;
      rewards.set(arbiterDid, bonus);
    } else {
      // Wrong vote: 15% slash on locked stake
      const slash = (lockedStake * BigInt(Math.floor(PUNISHMENT_RATES.wrongArbitrationSlash * 100))) / 100n;
      penalties.set(arbiterDid, slash);
    }
  }

  return { rewards, penalties };
}

// ============ Protocol Fees ============

/**
 * Calculate protocol fee from dispute
 */
export function calculateProtocolFee(totalPenalty: bigint): bigint {
  return (totalPenalty * BigInt(Math.floor(DISPUTE_ECONOMICS.protocolFee * 100))) / 100n;
}

/**
 * Calculate research fund share from dispute
 */
export function calculateResearchShare(totalPenalty: bigint): bigint {
  return (totalPenalty * BigInt(Math.floor(DISPUTE_ECONOMICS.researchShare * 100))) / 100n;
}

// ============ Main Settlement Calculation ============

/**
 * Calculate complete settlement for a dispute
 */
export function calculateSettlement(dispute: Dispute): DisputeSettlement {
  // Must have a winner
  const winner = dispute.winner || determineFinalWinner(dispute);
  if (winner === 'draw') {
    throw new DisputeError('Cannot settle dispute with draw outcome', 'INVALID_PHASE');
  }

  const loser = winner === 'challenger' ? 'defender' : 'challenger';
  const loserStake = loser === 'challenger' ? dispute.challenger.stake : dispute.defender.stake;

  // Calculate loser penalty
  const { penalty: loserPenalty, rate: penaltyRate } = calculateLoserPenalty(dispute, loser);

  // Calculate winner reward
  const winnerRewardCalc = calculateWinnerReward(dispute, loserStake);

  // Calculate arbiter settlements
  const arbiterSettlement = calculateArbiterSettlement(dispute);

  // Calculate protocol shares from penalties
  const protocolFee = calculateProtocolFee(loserPenalty);
  const researchFund = calculateResearchShare(loserPenalty);

  return {
    disputeId: dispute.id,
    winner,
    winnerReward: winnerRewardCalc.total,
    loserPenalty,
    truthBonusApplied: winnerRewardCalc.truthBonusApplied,
    earlyDetectionBonusApplied: winnerRewardCalc.earlyBonusApplied,
    arbiterRewards: arbiterSettlement.rewards,
    arbiterPenalties: arbiterSettlement.penalties,
    protocolFee,
    researchFund,
    breakdown: {
      loserStake,
      baseWinnerReward: winnerRewardCalc.base,
      truthBonus: winnerRewardCalc.truthBonus,
      earlyBonus: winnerRewardCalc.earlyBonus,
      penaltyRate,
    },
  };
}

// ============ Settlement Validation ============

/**
 * Validate settlement is mathematically correct
 */
export function validateSettlement(
  settlement: DisputeSettlement,
  dispute: Dispute
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  // Winner reward should not exceed loser stake + bonuses
  const maxReward = settlement.breakdown.loserStake * 2n; // Generous upper bound
  if (settlement.winnerReward > maxReward) {
    errors.push('Winner reward exceeds maximum possible');
  }

  // Loser penalty should not exceed stake
  const loserStake =
    settlement.winner === 'challenger'
      ? dispute.defender.stake
      : dispute.challenger.stake;
  if (settlement.loserPenalty > loserStake) {
    errors.push('Loser penalty exceeds stake');
  }

  // Protocol + research shares should come from penalty
  const protocolShares = settlement.protocolFee + settlement.researchFund;
  if (protocolShares > settlement.loserPenalty) {
    errors.push('Protocol shares exceed penalty amount');
  }

  // Arbiter penalties should not exceed locked stakes
  for (const [arbiterDid, penalty] of settlement.arbiterPenalties) {
    const vote = dispute.votes.get(arbiterDid);
    if (vote && penalty > vote.lockedStake) {
      errors.push(`Arbiter ${arbiterDid} penalty exceeds locked stake`);
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

// ============ Draw Handling ============

/**
 * Calculate settlement for draw outcome (both parties get partial stake back)
 */
export function calculateDrawSettlement(dispute: Dispute): {
  challengerRefund: bigint;
  defenderRefund: bigint;
  protocolFee: bigint;
  researchFund: bigint;
} {
  // In a draw, both parties lose a small amount to protocol
  const drawPenaltyRate = 0.05; // 5% penalty for wasting everyone's time

  const challengerPenalty =
    (dispute.challenger.stake * BigInt(Math.floor(drawPenaltyRate * 100))) / 100n;
  const defenderPenalty =
    (dispute.defender.stake * BigInt(Math.floor(drawPenaltyRate * 100))) / 100n;

  const totalPenalty = challengerPenalty + defenderPenalty;
  const protocolFee = calculateProtocolFee(totalPenalty);
  const researchFund = calculateResearchShare(totalPenalty);

  return {
    challengerRefund: dispute.challenger.stake - challengerPenalty,
    defenderRefund: dispute.defender.stake - defenderPenalty,
    protocolFee,
    researchFund,
  };
}

// ============ Settlement Summary ============

/**
 * Generate human-readable settlement summary
 */
export function generateSettlementSummary(settlement: DisputeSettlement): string {
  const lines: string[] = [
    `Dispute ${settlement.disputeId} Settlement`,
    `========================================`,
    `Winner: ${settlement.winner}`,
    ``,
    `Winner Reward: ${formatAmount(settlement.winnerReward)}`,
  ];

  if (settlement.truthBonusApplied) {
    lines.push(`  - Truth Bonus Applied (+50%): ${formatAmount(settlement.breakdown.truthBonus)}`);
  }

  if (settlement.earlyDetectionBonusApplied) {
    lines.push(`  - Early Detection Bonus (+20%): ${formatAmount(settlement.breakdown.earlyBonus)}`);
  }

  lines.push(
    ``,
    `Loser Penalty: ${formatAmount(settlement.loserPenalty)} (${(settlement.breakdown.penaltyRate * 100).toFixed(0)}%)`,
    ``,
    `Protocol Fee: ${formatAmount(settlement.protocolFee)}`,
    `Research Fund: ${formatAmount(settlement.researchFund)}`,
    ``
  );

  if (settlement.arbiterRewards.size > 0) {
    lines.push(`Arbiter Rewards:`);
    for (const [did, amount] of settlement.arbiterRewards) {
      lines.push(`  ${did}: +${formatAmount(amount)}`);
    }
  }

  if (settlement.arbiterPenalties.size > 0) {
    lines.push(`Arbiter Penalties:`);
    for (const [did, amount] of settlement.arbiterPenalties) {
      lines.push(`  ${did}: -${formatAmount(amount)}`);
    }
  }

  return lines.join('\n');
}

/**
 * Format amount for display (assumes 18 decimals)
 */
function formatAmount(amount: bigint): string {
  const whole = amount / (10n ** 18n);
  const fraction = amount % (10n ** 18n);
  const fractionStr = fraction.toString().padStart(18, '0').slice(0, 4);
  return `${whole}.${fractionStr} LGM`;
}
