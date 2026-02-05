/**
 * Domain Expertise Weighting
 *
 * Weights votes based on the voter's domain expertise. Voters who have
 * demonstrated accuracy in a domain get higher weight for proposals
 * in that domain. This ensures technical experts have appropriate
 * influence on technical decisions.
 *
 * PP HIERARCHY:
 *   TRUTH: Expertise based on historical accuracy
 *   UNDERSTANDING: Clear domain definitions
 *   BEAUTY: Elegant expertise → weight mapping
 *   JUSTICE: Expertise in domain = more influence in domain
 *   CREATIVITY: Novel reputation-weighted governance
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { LIPCategory } from "../proposals/types.js";

// ============ Domain Definitions ============

/**
 * Governance domains that expertise can be earned in.
 * Maps to LIP categories plus additional technical domains.
 */
export type GovernanceDomain =
  | "consensus" // Consensus mechanism expertise
  | "economics" // Token economics expertise
  | "cryptography" // Cryptographic primitives
  | "networking" // P2P and networking
  | "state" // State management
  | "security" // Security analysis
  | "governance" // Governance process itself
  | "interface"; // APIs and interfaces

/**
 * Map LIP categories to governance domains.
 */
export function lipCategoryToDomain(category: LIPCategory): GovernanceDomain {
  return category; // Direct mapping for now
}

// ============ Expertise Tracking ============

/**
 * Domain expertise record for an agent.
 */
export interface DomainExpertise {
  domain: GovernanceDomain;
  accuracy: number; // 0.0 - 1.0, historical accuracy
  totalParticipations: number;
  correctPositions: number; // Voted with final outcome
  weight: number; // Current weight multiplier
  lastUpdated: bigint;
}

/**
 * Complete expertise profile for an agent.
 */
export interface ExpertiseProfile {
  agent: DID;
  domains: Map<GovernanceDomain, DomainExpertise>;
  overallAccuracy: number;
  totalParticipations: number;
}

// ============ Weight Calculations ============

/**
 * Configuration for domain weight calculation.
 */
export interface DomainWeightConfig {
  minAccuracyForBonus: number; // Below this, weight = 1.0
  maxMultiplier: number; // Maximum weight multiplier
  minParticipations: number; // Minimum participations for weight
  decayHalfLife: bigint; // Blocks for expertise to decay by half
}

export const DEFAULT_DOMAIN_WEIGHT_CONFIG: DomainWeightConfig = {
  minAccuracyForBonus: 0.6, // 60% accuracy = 1.0x weight
  maxMultiplier: 3.0, // 100% accuracy = 3.0x weight
  minParticipations: 5, // Need 5 participations
  decayHalfLife: 200000n, // ~6 days
};

/**
 * Calculate weight multiplier from accuracy.
 *
 * Mapping:
 * - 60% accuracy = 1.0x
 * - 80% accuracy = 2.0x
 * - 100% accuracy = 3.0x
 *
 * Formula: weight = 1.0 + (accuracy - 0.6) * 5
 * (Maps [0.6, 1.0] to [1.0, 3.0])
 */
export function calculateDomainMultiplier(
  accuracy: number,
  config: DomainWeightConfig = DEFAULT_DOMAIN_WEIGHT_CONFIG,
): number {
  if (accuracy < config.minAccuracyForBonus) {
    return 1.0;
  }

  // Linear interpolation from 1.0 to maxMultiplier
  const range = 1.0 - config.minAccuracyForBonus; // 0.4
  const multiplierRange = config.maxMultiplier - 1.0; // 2.0
  const ratio = (accuracy - config.minAccuracyForBonus) / range;

  return Math.min(1.0 + ratio * multiplierRange, config.maxMultiplier);
}

/**
 * Calculate expertise decay based on time since last activity.
 */
export function calculateDecay(
  blocksSinceActivity: bigint,
  config: DomainWeightConfig = DEFAULT_DOMAIN_WEIGHT_CONFIG,
): number {
  // Exponential decay: decay = 0.5^(time / halfLife)
  const halfLives = Number(blocksSinceActivity) / Number(config.decayHalfLife);
  return Math.pow(0.5, halfLives);
}

/**
 * Apply decay to expertise.
 */
export function applyDecay(
  expertise: DomainExpertise,
  currentBlock: bigint,
  config: DomainWeightConfig = DEFAULT_DOMAIN_WEIGHT_CONFIG,
): DomainExpertise {
  const blocksSinceUpdate = currentBlock - expertise.lastUpdated;
  const decay = calculateDecay(blocksSinceUpdate, config);

  // Decay moves accuracy toward 0.5 (neutral)
  const decayedAccuracy = 0.5 + (expertise.accuracy - 0.5) * decay;

  return {
    ...expertise,
    accuracy: decayedAccuracy,
    weight: calculateDomainMultiplier(decayedAccuracy, config),
    lastUpdated: currentBlock,
  };
}

// ============ Expertise Management ============

/**
 * Create a new expertise profile.
 */
export function createExpertiseProfile(agent: DID): ExpertiseProfile {
  return {
    agent,
    domains: new Map(),
    overallAccuracy: 0.5, // Neutral
    totalParticipations: 0,
  };
}

/**
 * Initialize expertise in a domain.
 */
export function initializeDomainExpertise(
  domain: GovernanceDomain,
  currentBlock: bigint,
): DomainExpertise {
  return {
    domain,
    accuracy: 0.5, // Neutral starting accuracy
    totalParticipations: 0,
    correctPositions: 0,
    weight: 1.0, // Base weight
    lastUpdated: currentBlock,
  };
}

/**
 * Update expertise after a proposal outcome.
 */
export function updateExpertiseAfterOutcome(
  expertise: DomainExpertise,
  wasCorrect: boolean,
  currentBlock: bigint,
  config: DomainWeightConfig = DEFAULT_DOMAIN_WEIGHT_CONFIG,
): DomainExpertise {
  const newParticipations = expertise.totalParticipations + 1;
  const newCorrect = expertise.correctPositions + (wasCorrect ? 1 : 0);
  const newAccuracy = newCorrect / newParticipations;

  const weight =
    newParticipations >= config.minParticipations
      ? calculateDomainMultiplier(newAccuracy, config)
      : 1.0;

  return {
    domain: expertise.domain,
    accuracy: newAccuracy,
    totalParticipations: newParticipations,
    correctPositions: newCorrect,
    weight,
    lastUpdated: currentBlock,
  };
}

/**
 * Get domain weight for a voter.
 */
export function getDomainWeight(
  profile: ExpertiseProfile,
  domain: GovernanceDomain,
  currentBlock: bigint,
  config: DomainWeightConfig = DEFAULT_DOMAIN_WEIGHT_CONFIG,
): number {
  const expertise = profile.domains.get(domain);

  if (!expertise) {
    return 1.0; // No expertise = base weight
  }

  if (expertise.totalParticipations < config.minParticipations) {
    return 1.0; // Not enough participations
  }

  // Apply decay and return weight
  const decayed = applyDecay(expertise, currentBlock, config);
  return decayed.weight;
}

/**
 * Record a vote participation.
 */
export function recordParticipation(
  profile: ExpertiseProfile,
  domain: GovernanceDomain,
  currentBlock: bigint,
): ExpertiseProfile {
  const existing = profile.domains.get(domain);
  const expertise = existing || initializeDomainExpertise(domain, currentBlock);

  // Just update the timestamp for now
  // Accuracy will be updated when outcome is known
  const updated = {
    ...expertise,
    lastUpdated: currentBlock,
  };

  const newDomains = new Map(profile.domains);
  newDomains.set(domain, updated);

  return {
    ...profile,
    domains: newDomains,
    totalParticipations: profile.totalParticipations + 1,
  };
}

/**
 * Record outcome for a voter.
 */
export function recordOutcome(
  profile: ExpertiseProfile,
  domain: GovernanceDomain,
  wasCorrect: boolean,
  currentBlock: bigint,
  config: DomainWeightConfig = DEFAULT_DOMAIN_WEIGHT_CONFIG,
): ExpertiseProfile {
  const existing = profile.domains.get(domain);

  if (!existing) {
    // First participation in this domain
    const expertise = initializeDomainExpertise(domain, currentBlock);
    const updated = updateExpertiseAfterOutcome(
      expertise,
      wasCorrect,
      currentBlock,
      config,
    );

    const newDomains = new Map(profile.domains);
    newDomains.set(domain, updated);

    return {
      ...profile,
      domains: newDomains,
    };
  }

  const updated = updateExpertiseAfterOutcome(
    existing,
    wasCorrect,
    currentBlock,
    config,
  );

  const newDomains = new Map(profile.domains);
  newDomains.set(domain, updated);

  // Recalculate overall accuracy
  let totalCorrect = 0;
  let totalParticipations = 0;

  for (const exp of newDomains.values()) {
    totalCorrect += exp.correctPositions;
    totalParticipations += exp.totalParticipations;
  }

  const overallAccuracy =
    totalParticipations > 0 ? totalCorrect / totalParticipations : 0.5;

  return {
    ...profile,
    domains: newDomains,
    overallAccuracy,
    totalParticipations,
  };
}

// ============ Weighted Voting ============

/**
 * Calculate weighted vote power combining all factors.
 */
export function calculateWeightedVotePower(
  baseVotes: number,
  convictionMultiplier: number,
  domainMultiplier: number,
): number {
  return baseVotes * convictionMultiplier * domainMultiplier;
}

/**
 * Vote with full weighting applied.
 */
export interface WeightedVote {
  voter: DID;
  lipId: string;
  direction: "support" | "oppose" | "abstain";
  baseVotes: number;
  quadraticCost: bigint;
  convictionMultiplier: number;
  domainMultiplier: number;
  effectiveVotePower: number;
}

/**
 * Create a weighted vote with all multipliers.
 */
export function createWeightedVote(
  voter: DID,
  lipId: string,
  direction: "support" | "oppose" | "abstain",
  baseVotes: number,
  quadraticCost: bigint,
  convictionMultiplier: number,
  domainMultiplier: number,
): WeightedVote {
  return {
    voter,
    lipId,
    direction,
    baseVotes,
    quadraticCost,
    convictionMultiplier,
    domainMultiplier,
    effectiveVotePower: calculateWeightedVotePower(
      baseVotes,
      convictionMultiplier,
      domainMultiplier,
    ),
  };
}

// ============ Expertise Leaderboard ============

/**
 * Entry in domain expertise leaderboard.
 */
export interface ExpertiseLeaderboardEntry {
  agent: DID;
  domain: GovernanceDomain;
  accuracy: number;
  weight: number;
  participations: number;
}

/**
 * Get top experts in a domain.
 */
export function getTopExperts(
  profiles: ExpertiseProfile[],
  domain: GovernanceDomain,
  currentBlock: bigint,
  limit: number = 10,
  config: DomainWeightConfig = DEFAULT_DOMAIN_WEIGHT_CONFIG,
): ExpertiseLeaderboardEntry[] {
  const entries: ExpertiseLeaderboardEntry[] = [];

  for (const profile of profiles) {
    const expertise = profile.domains.get(domain);
    if (
      !expertise ||
      expertise.totalParticipations < config.minParticipations
    ) {
      continue;
    }

    const decayed = applyDecay(expertise, currentBlock, config);

    entries.push({
      agent: profile.agent,
      domain,
      accuracy: decayed.accuracy,
      weight: decayed.weight,
      participations: expertise.totalParticipations,
    });
  }

  // Sort by weight descending
  entries.sort((a, b) => b.weight - a.weight);

  return entries.slice(0, limit);
}

/**
 * Get expertise summary for display.
 */
export function getExpertiseSummary(
  profile: ExpertiseProfile,
  currentBlock: bigint,
  config: DomainWeightConfig = DEFAULT_DOMAIN_WEIGHT_CONFIG,
): Array<{
  domain: GovernanceDomain;
  accuracy: string;
  weight: string;
  participations: number;
  status: "active" | "building" | "decaying";
}> {
  const summary: Array<{
    domain: GovernanceDomain;
    accuracy: string;
    weight: string;
    participations: number;
    status: "active" | "building" | "decaying";
  }> = [];

  for (const [domain, expertise] of profile.domains) {
    const decayed = applyDecay(expertise, currentBlock, config);
    const blocksSinceUpdate = currentBlock - expertise.lastUpdated;

    let status: "active" | "building" | "decaying";
    if (blocksSinceUpdate > config.decayHalfLife) {
      status = "decaying";
    } else if (expertise.totalParticipations < config.minParticipations) {
      status = "building";
    } else {
      status = "active";
    }

    summary.push({
      domain,
      accuracy: `${(decayed.accuracy * 100).toFixed(1)}%`,
      weight: `${decayed.weight.toFixed(2)}x`,
      participations: expertise.totalParticipations,
      status,
    });
  }

  // Sort by weight descending
  summary.sort((a, b) => parseFloat(b.weight) - parseFloat(a.weight));

  return summary;
}
