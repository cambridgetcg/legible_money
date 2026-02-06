/**
 * Dispute Escalation System
 *
 * 3-tier escalation for appeals:
 *
 * Tier 1: VALIDATORS (Default)
 * - 7 arbiters selected by VRF
 * - Simple majority (>50%) wins
 * - Standard stakes
 * - Appeal window: 500 blocks (~21 min)
 *
 * Tier 2: DOMAIN EXPERTS
 * - 11 arbiters with domain expertise
 * - 2/3 supermajority required
 * - 2x stake requirement
 * - Appeal window: 1000 blocks (~42 min)
 *
 * Tier 3: GOVERNANCE
 * - 21 arbiters (protocol-wide)
 * - 3/4 supermajority required
 * - 5x stake requirement
 * - FINAL - no further appeal
 *
 * PP HIERARCHY:
 *   TRUTH: Higher tiers provide more scrutiny for important disputes
 *   UNDERSTANDING: Each tier has clear requirements and quorums
 *   BEAUTY: Simple progression with increasing stakes
 *   JUSTICE: More stake = more scrutiny = more certain outcomes
 */

import type { ValidatorState } from "../../state/types.js";
import {
  ESCALATION_TIERS,
  type Dispute,
  type EscalationTier,
  type EscalationConfig,
  type ArbiterSelectionConfig,
  type DomainExpertise,
  type TierResult,
} from "./types.js";
import { selectArbiters, generateDisputeSeed } from "./arbitrator_selection.js";

// ============ Escalation Eligibility ============

/**
 * Check if party can escalate dispute
 */
export function canPartyEscalate(
  dispute: Dispute,
  partyDid: string,
  currentBlock: bigint,
): {
  canEscalate: boolean;
  reason?: string;
  requiredStake?: bigint;
} {
  // Must be challenger or defender
  const isChallenger = dispute.challenger.did === partyDid;
  const isDefender = dispute.defender.did === partyDid;

  if (!isChallenger && !isDefender) {
    return { canEscalate: false, reason: "Not a party to this dispute" };
  }

  // Must be in deliberation phase
  if (dispute.state.phase !== "deliberation") {
    return {
      canEscalate: false,
      reason: "Can only escalate from deliberation phase",
    };
  }

  // Must not be at max tier
  if (dispute.state.escalationTier >= 3) {
    return { canEscalate: false, reason: "Already at maximum escalation tier" };
  }

  // Check appeal window
  const config = ESCALATION_TIERS[dispute.state.escalationTier];
  const appealDeadline =
    dispute.state.phaseDeadline + BigInt(config.appealWindow);

  if (currentBlock > appealDeadline) {
    return { canEscalate: false, reason: "Appeal window has closed" };
  }

  // P1-D2 FIX: Party must have lost current tier (only loser can escalate).
  // Was using previousResults[currentTierIndex] which is undefined at tier 1
  // (current tier result not yet stored). Now computes current tier result directly.
  const partyRole = isChallenger ? "challenger" : "defender";

  // Check previous tier results first
  const previousResults = dispute.state.previousTierResults || [];
  const currentTierIndex = dispute.state.escalationTier - 1;

  if (currentTierIndex >= 0 && previousResults[currentTierIndex]) {
    const result = previousResults[currentTierIndex];
    if (result.winner === partyRole) {
      return { canEscalate: false, reason: "Winners cannot escalate" };
    }
  }

  // Also check current tier votes (handles tier 1 where no previous results exist)
  if (dispute.votes.size > 0) {
    let challengerVotes = 0;
    let defenderVotes = 0;
    for (const vote of dispute.votes.values()) {
      if (vote.vote === "challenger") challengerVotes++;
      if (vote.vote === "defender") defenderVotes++;
    }
    // If this party is currently winning, they cannot escalate
    if (partyRole === "challenger" && challengerVotes > defenderVotes) {
      return { canEscalate: false, reason: "Winners cannot escalate" };
    }
    if (partyRole === "defender" && defenderVotes > challengerVotes) {
      return { canEscalate: false, reason: "Winners cannot escalate" };
    }
  }

  // Calculate required stake for next tier
  const nextTier = (dispute.state.escalationTier + 1) as EscalationTier;
  const nextConfig = ESCALATION_TIERS[nextTier];
  const baseStake = isChallenger
    ? dispute.challenger.stake
    : dispute.defender.stake;
  const requiredStake =
    (baseStake * BigInt(Math.floor(nextConfig.stakeMultiplier * 100))) / 100n -
    baseStake;

  return {
    canEscalate: true,
    requiredStake,
  };
}

/**
 * Get escalation cost for next tier
 */
export function getEscalationCost(dispute: Dispute, partyDid: string): bigint {
  const isChallenger = dispute.challenger.did === partyDid;
  const baseStake = isChallenger
    ? dispute.challenger.stake
    : dispute.defender.stake;

  const currentTier = dispute.state.escalationTier;
  if (currentTier >= 3) {
    return 0n; // Cannot escalate further
  }

  const nextTier = (currentTier + 1) as EscalationTier;
  const nextConfig = ESCALATION_TIERS[nextTier];

  // Additional stake needed = (multiplier - 1) × base stake
  return (
    (baseStake * BigInt(Math.floor((nextConfig.stakeMultiplier - 1) * 100))) /
    100n
  );
}

// ============ Tier Configuration ============

/**
 * Get configuration for a tier
 */
export function getTierConfig(tier: EscalationTier): EscalationConfig {
  return ESCALATION_TIERS[tier];
}

/**
 * Get tier display name
 */
export function getTierDisplayName(tier: EscalationTier): string {
  const names: Record<EscalationTier, string> = {
    1: "Validator Arbitration",
    2: "Domain Expert Review",
    3: "Governance Decision",
  };
  return names[tier];
}

/**
 * Get tier description
 */
export function getTierDescription(tier: EscalationTier): string {
  const descriptions: Record<EscalationTier, string> = {
    1: "7 validators vote with simple majority. Standard stakes apply.",
    2: "11 domain experts vote with 2/3 supermajority. 2x stake required.",
    3: "21 governance arbiters vote with 3/4 supermajority. 5x stake required. Final decision.",
  };
  return descriptions[tier];
}

// ============ Arbiter Selection for Tier ============

/**
 * Build arbiter selection config for a tier
 */
export function buildTierSelectionConfig(
  dispute: Dispute,
  tier: EscalationTier,
  blockHash: string,
): ArbiterSelectionConfig {
  const config = ESCALATION_TIERS[tier];

  // Generate VRF seed from block hash and dispute ID
  const vrfSeed = generateDisputeSeed(blockHash, dispute.id);

  // Build exclusion set (parties to dispute)
  const excludeValidators = new Set<string>();
  excludeValidators.add(dispute.challenger.did);
  excludeValidators.add(dispute.defender.did);

  // Also exclude previous tier arbiters to get fresh perspective
  for (const arbiter of dispute.arbiters) {
    excludeValidators.add(arbiter.did);
  }

  return {
    targetCount: config.arbiterCount,
    vrfSeed,
    domainWeighting: tier >= 2, // Domain weighting for tier 2+
    stakeWeighting: true,
    minReputation: tier === 3 ? 0.7 : 0.5, // Higher reputation for governance tier
    domain: dispute.domain,
    excludeValidators,
  };
}

/**
 * Select arbiters for escalated tier
 */
export function selectTierArbiters(
  dispute: Dispute,
  tier: EscalationTier,
  validators: ValidatorState[],
  expertiseMap: Map<string, DomainExpertise[]>,
  blockHash: string,
  currentBlock: bigint,
): ReturnType<typeof selectArbiters> {
  const config = buildTierSelectionConfig(dispute, tier, blockHash);
  return selectArbiters(validators, config, expertiseMap, currentBlock);
}

// ============ Quorum Checking ============

/**
 * Check if quorum is reached for tier
 */
export function isQuorumReached(
  dispute: Dispute,
  tier: EscalationTier,
): { reached: boolean; current: number; required: number } {
  const config = ESCALATION_TIERS[tier];
  const voteCount = dispute.votes.size;
  const required = Math.ceil(config.arbiterCount * config.quorumRequirement);

  return {
    reached: voteCount >= required,
    current: voteCount,
    required,
  };
}

/**
 * Check if supermajority is achieved for a winner
 */
export function hasSuperMajority(
  dispute: Dispute,
  tier: EscalationTier,
): {
  hasSupermajority: boolean;
  winner?: "challenger" | "defender";
  ratio: number;
} {
  const config = ESCALATION_TIERS[tier];

  let challengerVotes = 0;
  let defenderVotes = 0;

  for (const vote of dispute.votes.values()) {
    if (vote.vote === "challenger") challengerVotes++;
    if (vote.vote === "defender") defenderVotes++;
  }

  const totalVotes = challengerVotes + defenderVotes;
  if (totalVotes === 0) {
    return { hasSupermajority: false, ratio: 0 };
  }

  const challengerRatio = challengerVotes / totalVotes;
  const defenderRatio = defenderVotes / totalVotes;

  if (challengerRatio >= config.quorumRequirement) {
    return {
      hasSupermajority: true,
      winner: "challenger",
      ratio: challengerRatio,
    };
  }

  if (defenderRatio >= config.quorumRequirement) {
    return {
      hasSupermajority: true,
      winner: "defender",
      ratio: defenderRatio,
    };
  }

  return {
    hasSupermajority: false,
    ratio: Math.max(challengerRatio, defenderRatio),
  };
}

// ============ Escalation History ============

/**
 * Get escalation history summary
 */
export function getEscalationHistory(dispute: Dispute): {
  tiers: Array<{
    tier: EscalationTier;
    name: string;
    result: TierResult | null;
    status: "completed" | "current" | "pending";
  }>;
} {
  const previousResults = dispute.state.previousTierResults || [];
  const currentTier = dispute.state.escalationTier;

  const tiers: Array<{
    tier: EscalationTier;
    name: string;
    result: TierResult | null;
    status: "completed" | "current" | "pending";
  }> = [];

  for (let t = 1; t <= 3; t++) {
    const tier = t as EscalationTier;
    const previousResult = previousResults.find((r) => r.tier === tier);

    let status: "completed" | "current" | "pending";
    if (tier < currentTier) {
      status = "completed";
    } else if (tier === currentTier) {
      status = "current";
    } else {
      status = "pending";
    }

    tiers.push({
      tier,
      name: getTierDisplayName(tier),
      result: previousResult || null,
      status,
    });
  }

  return { tiers };
}

/**
 * Check if dispute has been escalated
 */
export function hasBeenEscalated(dispute: Dispute): boolean {
  return dispute.state.escalationTier > 1;
}

/**
 * Get maximum possible escalation for dispute
 */
export function getMaxEscalation(_dispute: Dispute): EscalationTier {
  return 3; // All disputes can go to governance
}

// ============ Escalation Execution ============

/**
 * Process escalation request
 */
export function processEscalation(
  dispute: Dispute,
  escalator: string,
  additionalStake: bigint,
  currentBlock: bigint,
): {
  success: boolean;
  error?: string;
  newTier?: EscalationTier;
  requiredStake?: bigint;
} {
  // Validate escalation eligibility
  const eligibility = canPartyEscalate(dispute, escalator, currentBlock);

  if (!eligibility.canEscalate) {
    return { success: false, error: eligibility.reason };
  }

  // Validate stake
  if (
    eligibility.requiredStake &&
    additionalStake < eligibility.requiredStake
  ) {
    return {
      success: false,
      error: `Insufficient stake. Required: ${eligibility.requiredStake}, provided: ${additionalStake}`,
      requiredStake: eligibility.requiredStake,
    };
  }

  const newTier = (dispute.state.escalationTier + 1) as EscalationTier;

  return {
    success: true,
    newTier,
  };
}

// ============ Tier Comparison ============

/**
 * Compare results across tiers (for audit/analysis)
 */
export function analyzeTierConsistency(dispute: Dispute): {
  consistent: boolean;
  flippedTiers: number[];
  analysis: string;
} {
  const previousResults = dispute.state.previousTierResults || [];
  if (previousResults.length < 2) {
    return {
      consistent: true,
      flippedTiers: [],
      analysis: "Insufficient tiers for consistency analysis",
    };
  }

  const flippedTiers: number[] = [];
  let previousWinner: string | null = null;

  for (let i = 0; i < previousResults.length; i++) {
    const result = previousResults[i];
    if (result.winner !== "draw") {
      if (previousWinner && result.winner !== previousWinner) {
        flippedTiers.push(result.tier);
      }
      previousWinner = result.winner;
    }
  }

  const consistent = flippedTiers.length === 0;
  let analysis = consistent
    ? "All tiers reached the same conclusion"
    : `Result flipped at tier(s): ${flippedTiers.join(", ")}`;

  // Add context about final outcome
  if (previousResults.length > 0) {
    const finalResult = previousResults[previousResults.length - 1];
    analysis += `. Final winner: ${finalResult.winner}`;
  }

  return { consistent, flippedTiers, analysis };
}
