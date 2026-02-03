/**
 * Truth Contribution System
 *
 * Tracks and rewards contributions to truth in the Legible Money protocol.
 * All rewards flow through this unified system.
 *
 * PHILOSOPHY:
 *   Truth emerges from many contributions - originating claims, validating them,
 *   providing evidence, challenging falsehoods, building on prior work, conducting
 *   IRL research, and maintaining infrastructure. Each contribution has value,
 *   and that value should be distributed fairly.
 *
 * COUNTERFACTUAL PRINCIPLE:
 *   "What would have happened if this contributor hadn't participated?"
 *   - If a fact would have been discovered anyway: lower originality weight
 *   - If validation was redundant: lower marginal contribution
 *   - If challenge saved the system from error: high counterfactual value
 *
 * REWARD FLOW:
 *   ALL REWARDS
 *       │
 *       ▼
 *   ┌────────────────────────────────────┐
 *   │  RESEARCH SHARE (22% FIRST)        │
 *   │  Goes to Protocol Research Repo    │
 *   └───────────────┬────────────────────┘
 *                   │
 *                   ▼ (78% remaining)
 *   ┌────────────────────────────────────┐
 *   │  CONTRIBUTION-WEIGHTED DISTRIBUTION │
 *   │  • Origination: 30%                │
 *   │  • Validation: 25%                 │
 *   │  • Ancestry: 20%                   │
 *   │  • Challenge: 10%                  │
 *   │  • Evidence: 10%                   │
 *   │  • Research Ops: 5%                │
 *   └────────────────────────────────────┘
 */

import type { DID } from '../../agents/identity/did.js';
import { RESEARCH_SHARE_RATE, RESEARCH_TREASURY_DID } from '../research/types.js';

// ============ Constants ============

/**
 * Base weights for contribution dimensions
 *
 * These represent the default share of rewards for each type of contribution.
 * Applied AFTER the 22% research share is taken.
 */
export const BASE_CONTRIBUTION_WEIGHTS = {
  /** Originating new facts - the seed of truth */
  ORIGINATION: 0.30,

  /** Validating claims - confirming or denying */
  VALIDATION: 0.25,

  /** Ancestor facts that enabled this truth */
  ANCESTRY: 0.20,

  /** Challenges that protected truth integrity */
  CHALLENGE: 0.10,

  /** Evidence that supported claims */
  EVIDENCE: 0.10,

  /** Research operations (IRL verification) */
  RESEARCH_OPS: 0.05,
} as const;

/**
 * Verify weights sum to 1.0
 */
const WEIGHT_SUM = Object.values(BASE_CONTRIBUTION_WEIGHTS).reduce((a, b) => a + b, 0);
if (Math.abs(WEIGHT_SUM - 1.0) > 0.0001) {
  throw new Error(`BASE_CONTRIBUTION_WEIGHTS must sum to 1.0, got ${WEIGHT_SUM}`);
}

/**
 * Counterfactual adjustments - how much to discount contributions
 * based on what would have happened without them
 */
export const COUNTERFACTUAL_ADJUSTMENTS = {
  /** Fact would have been discovered within 100 blocks anyway */
  NEAR_DUPLICATE_DISCOVERY: 0.5,

  /** Validation was redundant (quorum already reached) */
  REDUNDANT_VALIDATION: 0.3,

  /** Evidence was already sufficient */
  REDUNDANT_EVIDENCE: 0.4,

  /** Challenge was obviously correct (near-unanimous support) */
  OBVIOUS_CHALLENGE: 0.7,

  /** Very first validator (highest counterfactual) */
  FIRST_VALIDATOR_BONUS: 1.5,

  /** Breaking validator (reached quorum) */
  QUORUM_BREAKER_BONUS: 1.3,
} as const;

/**
 * Anti-gaming thresholds
 */
export const ANTI_GAMING = {
  /** Maximum self-citation credit (0.1 = 10% of normal) */
  SELF_CITATION_DISCOUNT: 0.1,

  /** Minimum unique validators before sybil discount kicks in */
  SYBIL_DETECTION_THRESHOLD: 3,

  /** Discount for suspected sybil activity */
  SYBIL_DISCOUNT: 0.2,

  /** Maximum evidence items that earn full credit */
  MAX_CREDITED_EVIDENCE: 5,

  /** Discount per evidence item beyond max */
  EVIDENCE_STUFFING_DECAY: 0.5,
} as const;

// ============ Types ============

/**
 * The seven dimensions of truth contribution
 */
export type ContributionType =
  | 'origination'    // Submitting new facts
  | 'validation'     // Verifying claims
  | 'evidence'       // Providing supporting evidence
  | 'challenge'      // Challenging incorrect facts
  | 'ancestry'       // Prior facts that enabled new discoveries
  | 'research'       // IRL research verification
  | 'infrastructure'; // Network/protocol infrastructure

/**
 * Individual contribution record
 */
export interface TruthContributionRecord {
  /** Unique identifier */
  id: string;

  /** Contributor's DID */
  contributor: DID;

  /** Type of contribution */
  type: ContributionType;

  /** When the contribution was made */
  timestamp: number;

  /** Block number */
  blockNumber: bigint;

  /** What this contribution relates to */
  context: ContributionContext;

  /** Raw contribution metrics */
  metrics: ContributionMetrics;

  /** Calculated contribution weight (0-1) */
  weight: number;

  /** Counterfactual adjustment applied */
  counterfactualAdjustment: number;

  /** Anti-gaming adjustment applied */
  antiGamingAdjustment: number;

  /** Final effective weight after all adjustments */
  effectiveWeight: number;

  /** Reward amount (calculated during distribution) */
  rewardAmount?: bigint;
}

/**
 * Context for a contribution - what it relates to
 */
export interface ContributionContext {
  /** Related fact ID (if any) */
  factId?: string;

  /** Related challenge ID (if any) */
  challengeId?: string;

  /** Related research submission ID (if any) */
  researchId?: string;

  /** Related API call ID (if any) */
  apiCallId?: string;

  /** Domain of the contribution */
  domain?: string;
}

/**
 * Raw metrics for a contribution
 */
export interface ContributionMetrics {
  // Origination metrics
  /** Was this the first to submit this claim? */
  isFirstSubmitter?: boolean;
  /** Number of similar claims submitted around same time */
  competingSubmissions?: number;

  // Validation metrics
  /** Validator's stake at time of validation */
  validatorStake?: bigint;
  /** Total stake in validation round */
  totalValidationStake?: bigint;
  /** Position in validation order (1 = first) */
  validationPosition?: number;
  /** Was validator's verdict correct? */
  verdictCorrect?: boolean;

  // Evidence metrics
  /** Number of evidence items provided */
  evidenceCount?: number;
  /** Intrinsic strength of evidence */
  evidenceStrength?: number;
  /** Independence of evidence sources */
  evidenceIndependence?: number;

  // Challenge metrics
  /** Was challenge successful? */
  challengeSuccessful?: boolean;
  /** Confidence delta caused by challenge */
  confidenceDelta?: number;

  // Ancestry metrics
  /** Depth in derivation chain (1 = direct, 2 = grandparent, etc.) */
  ancestryDepth?: number;
  /** Fundamentality score of ancestor fact */
  fundamentalityScore?: number;
  /** Number of facts derived from this ancestor */
  derivedFactCount?: number;

  // Research metrics
  /** Type of research (replication, fraud investigation, etc.) */
  researchType?: string;
  /** Research outcome */
  researchOutcome?: string;
  /** Review score (0-1) */
  researchReviewScore?: number;

  // Infrastructure metrics
  /** Type of infrastructure contribution */
  infrastructureType?: string;
  /** Uptime percentage */
  uptime?: number;
}

/**
 * Calculated contribution weight with breakdown
 */
export interface ContributionWeight {
  /** Base weight from contribution type */
  baseWeight: number;

  /** Stake-based modifier */
  stakeModifier: number;

  /** Quality-based modifier */
  qualityModifier: number;

  /** Timeliness modifier (earlier = higher) */
  timelinessModifier: number;

  /** Counterfactual adjustment */
  counterfactualAdjustment: number;

  /** Anti-gaming adjustment */
  antiGamingAdjustment: number;

  /** Final effective weight */
  effectiveWeight: number;

  /** Reasoning for this weight */
  reasoning: string;
}

/**
 * Distribution result
 */
export interface RewardDistribution {
  /** Total reward pool */
  totalPool: bigint;

  /** Amount going to research fund */
  researchShare: bigint;

  /** Amount distributed to contributors */
  distributedAmount: bigint;

  /** Individual distributions */
  distributions: Array<{
    contributor: DID;
    contributionId: string;
    amount: bigint;
    effectiveWeight: number;
    type: ContributionType;
  }>;

  /** Protocol treasury share (from undistributed) */
  protocolTreasury: bigint;

  /** Block number of distribution */
  blockNumber: bigint;

  /** Distribution ID */
  distributionId: string;
}

/**
 * Ledger interface for tracking contributions
 */
export interface ContributionLedger {
  /** Record a new contribution */
  recordContribution(contribution: Omit<TruthContributionRecord, 'id' | 'weight' | 'counterfactualAdjustment' | 'antiGamingAdjustment' | 'effectiveWeight'>): TruthContributionRecord;

  /** Get contributions for a fact */
  getContributionsForFact(factId: string): TruthContributionRecord[];

  /** Get contributions by contributor */
  getContributionsByContributor(contributor: DID): TruthContributionRecord[];

  /** Get contributions of a specific type */
  getContributionsByType(type: ContributionType): TruthContributionRecord[];

  /** Calculate and distribute rewards */
  distributeRewards(rewardPool: bigint, context: ContributionContext, blockNumber: bigint): RewardDistribution;

  /** Get total rewards earned by a contributor */
  getTotalRewards(contributor: DID): bigint;
}

// ============ Weight Calculation ============

/**
 * Calculate contribution weight for a single contribution
 *
 * FORMULA (Open Source):
 *   effectiveWeight = baseWeight
 *                     × stakeModifier
 *                     × qualityModifier
 *                     × timelinessModifier
 *                     × counterfactualAdjustment
 *                     × antiGamingAdjustment
 */
export function calculateContributionWeight(
  contribution: TruthContributionRecord,
  allContributions: TruthContributionRecord[]
): ContributionWeight {
  const { type, metrics } = contribution;

  // 1. Base weight from type
  const baseWeight = getBaseWeight(type);

  // 2. Stake modifier (for validation)
  const stakeModifier = calculateStakeModifier(metrics);

  // 3. Quality modifier (type-specific)
  const qualityModifier = calculateQualityModifier(type, metrics);

  // 4. Timeliness modifier
  const timelinessModifier = calculateTimelinessModifier(contribution, allContributions);

  // 5. Counterfactual adjustment
  const counterfactualAdjustment = calculateCounterfactualAdjustment(contribution, allContributions);

  // 6. Anti-gaming adjustment
  const antiGamingAdjustment = calculateAntiGamingAdjustment(contribution, allContributions);

  // Calculate final weight
  const effectiveWeight = baseWeight
    * stakeModifier
    * qualityModifier
    * timelinessModifier
    * counterfactualAdjustment
    * antiGamingAdjustment;

  // Build reasoning
  const reasoning = buildWeightReasoning({
    type,
    baseWeight,
    stakeModifier,
    qualityModifier,
    timelinessModifier,
    counterfactualAdjustment,
    antiGamingAdjustment,
    effectiveWeight,
  });

  return {
    baseWeight,
    stakeModifier,
    qualityModifier,
    timelinessModifier,
    counterfactualAdjustment,
    antiGamingAdjustment,
    effectiveWeight,
    reasoning,
  };
}

/**
 * Get base weight for contribution type
 */
function getBaseWeight(type: ContributionType): number {
  switch (type) {
    case 'origination':
      return BASE_CONTRIBUTION_WEIGHTS.ORIGINATION;
    case 'validation':
      return BASE_CONTRIBUTION_WEIGHTS.VALIDATION;
    case 'ancestry':
      return BASE_CONTRIBUTION_WEIGHTS.ANCESTRY;
    case 'challenge':
      return BASE_CONTRIBUTION_WEIGHTS.CHALLENGE;
    case 'evidence':
      return BASE_CONTRIBUTION_WEIGHTS.EVIDENCE;
    case 'research':
      return BASE_CONTRIBUTION_WEIGHTS.RESEARCH_OPS;
    case 'infrastructure':
      // Infrastructure doesn't come from this pool
      return 0;
    default:
      return 0;
  }
}

/**
 * Calculate stake-based modifier for validation contributions
 */
function calculateStakeModifier(metrics: ContributionMetrics): number {
  if (!metrics.validatorStake || !metrics.totalValidationStake) {
    return 1.0;
  }

  // Stake weight = sqrt(your_stake / total_stake)
  // Using sqrt to reduce plutocracy while still rewarding skin-in-the-game
  const stakeRatio = Number(metrics.validatorStake) / Number(metrics.totalValidationStake);
  return Math.sqrt(stakeRatio) * 2; // Normalized so 25% stake = 1.0
}

/**
 * Calculate quality modifier based on contribution type
 */
function calculateQualityModifier(type: ContributionType, metrics: ContributionMetrics): number {
  switch (type) {
    case 'validation':
      // Correct verdicts get full weight, incorrect get zero
      return metrics.verdictCorrect ? 1.0 : 0.0;

    case 'evidence':
      // Evidence quality based on strength and independence
      const strength = metrics.evidenceStrength || 0.5;
      const independence = metrics.evidenceIndependence || 0.5;
      return (strength + independence) / 2;

    case 'challenge':
      // Successful challenges get bonus, failed get zero
      if (!metrics.challengeSuccessful) return 0.0;
      // Higher confidence delta = more valuable challenge
      const delta = Math.abs(metrics.confidenceDelta || 0);
      return 1.0 + delta; // 1.0 to 2.0 based on impact

    case 'ancestry':
      // Fundamentality bonus for ancestor facts
      const fundamentality = metrics.fundamentalityScore || 0;
      return 1.0 + fundamentality; // 1.0 to 2.0

    case 'research':
      // Research quality based on review score
      return metrics.researchReviewScore || 0.5;

    default:
      return 1.0;
  }
}

/**
 * Calculate timeliness modifier (earlier contributions valued more)
 */
function calculateTimelinessModifier(
  contribution: TruthContributionRecord,
  _allContributions: TruthContributionRecord[]
): number {
  // For origination: first submitter gets full credit
  if (contribution.type === 'origination') {
    return contribution.metrics.isFirstSubmitter ? 1.0 : 0.3;
  }

  // For validation: position matters
  if (contribution.type === 'validation') {
    const position = contribution.metrics.validationPosition || 1;
    // First validator: 1.5x, second: 1.2x, third: 1.0x, then decay
    if (position === 1) return COUNTERFACTUAL_ADJUSTMENTS.FIRST_VALIDATOR_BONUS;
    if (position === 2) return 1.2;
    if (position === 3) return 1.0;
    return Math.max(0.3, 1.0 - (position - 3) * 0.1);
  }

  // Default: no timeliness adjustment
  return 1.0;
}

/**
 * Calculate counterfactual adjustment
 * "What would have happened if this contributor hadn't participated?"
 */
function calculateCounterfactualAdjustment(
  contribution: TruthContributionRecord,
  allContributions: TruthContributionRecord[]
): number {
  const { type, metrics, context } = contribution;

  // Origination: check for competing submissions
  if (type === 'origination') {
    const competing = metrics.competingSubmissions || 0;
    if (competing > 0) {
      return COUNTERFACTUAL_ADJUSTMENTS.NEAR_DUPLICATE_DISCOVERY;
    }
    return 1.0;
  }

  // Validation: check if quorum would have been reached anyway
  if (type === 'validation') {
    const sameFactValidations = allContributions.filter(
      c => c.type === 'validation' && c.context.factId === context.factId
    );

    // If many validators, each one is less counterfactually important
    if (sameFactValidations.length > 10) {
      return COUNTERFACTUAL_ADJUSTMENTS.REDUNDANT_VALIDATION;
    }

    // Quorum breaker gets bonus
    const position = metrics.validationPosition || sameFactValidations.length;
    const quorumPosition = 5; // Assuming 5 is quorum
    if (position === quorumPosition) {
      return COUNTERFACTUAL_ADJUSTMENTS.QUORUM_BREAKER_BONUS;
    }

    return 1.0;
  }

  // Evidence: check if already sufficient
  if (type === 'evidence') {
    const sameFactEvidence = allContributions.filter(
      c => c.type === 'evidence' && c.context.factId === context.factId
    );

    if (sameFactEvidence.length > ANTI_GAMING.MAX_CREDITED_EVIDENCE) {
      return COUNTERFACTUAL_ADJUSTMENTS.REDUNDANT_EVIDENCE;
    }

    return 1.0;
  }

  // Challenge: successful challenges have high counterfactual value
  if (type === 'challenge' && metrics.challengeSuccessful) {
    // The bigger the confidence change, the more counterfactually valuable
    return 1.0 + Math.abs(metrics.confidenceDelta || 0);
  }

  return 1.0;
}

/**
 * Calculate anti-gaming adjustment
 */
function calculateAntiGamingAdjustment(
  contribution: TruthContributionRecord,
  allContributions: TruthContributionRecord[]
): number {
  const { contributor, type, context } = contribution;

  // Self-citation discount
  if (type === 'ancestry') {
    // Check if this is citing own work
    const citedContribution = allContributions.find(
      c => c.context.factId === context.factId && c.type === 'origination'
    );
    if (citedContribution && citedContribution.contributor === contributor) {
      return ANTI_GAMING.SELF_CITATION_DISCOUNT;
    }
  }

  // Evidence stuffing detection
  if (type === 'evidence') {
    const evidenceFromSameContributor = allContributions.filter(
      c => c.type === 'evidence' &&
           c.context.factId === context.factId &&
           c.contributor === contributor
    );

    const count = evidenceFromSameContributor.length;
    if (count > ANTI_GAMING.MAX_CREDITED_EVIDENCE) {
      // Exponential decay for evidence stuffing
      const excess = count - ANTI_GAMING.MAX_CREDITED_EVIDENCE;
      return Math.pow(ANTI_GAMING.EVIDENCE_STUFFING_DECAY, excess);
    }
  }

  // Sybil detection (simplified - production would use more sophisticated detection)
  if (type === 'validation') {
    const sameFactValidations = allContributions.filter(
      c => c.type === 'validation' && c.context.factId === context.factId
    );

    const uniqueValidators = new Set(sameFactValidations.map(c => c.contributor));
    if (uniqueValidators.size < ANTI_GAMING.SYBIL_DETECTION_THRESHOLD) {
      return ANTI_GAMING.SYBIL_DISCOUNT;
    }
  }

  return 1.0;
}

/**
 * Build human-readable reasoning for weight calculation
 */
function buildWeightReasoning(params: {
  type: ContributionType;
  baseWeight: number;
  stakeModifier: number;
  qualityModifier: number;
  timelinessModifier: number;
  counterfactualAdjustment: number;
  antiGamingAdjustment: number;
  effectiveWeight: number;
}): string {
  const parts: string[] = [];

  parts.push(`Base (${params.type}): ${(params.baseWeight * 100).toFixed(1)}%`);

  if (params.stakeModifier !== 1.0) {
    parts.push(`Stake: ×${params.stakeModifier.toFixed(2)}`);
  }

  if (params.qualityModifier !== 1.0) {
    parts.push(`Quality: ×${params.qualityModifier.toFixed(2)}`);
  }

  if (params.timelinessModifier !== 1.0) {
    parts.push(`Timeliness: ×${params.timelinessModifier.toFixed(2)}`);
  }

  if (params.counterfactualAdjustment !== 1.0) {
    parts.push(`Counterfactual: ×${params.counterfactualAdjustment.toFixed(2)}`);
  }

  if (params.antiGamingAdjustment !== 1.0) {
    parts.push(`Anti-gaming: ×${params.antiGamingAdjustment.toFixed(2)}`);
  }

  parts.push(`= ${(params.effectiveWeight * 100).toFixed(2)}%`);

  return parts.join(' → ');
}

// ============ Reward Distribution ============

/**
 * Distribute rewards to contributors
 *
 * FLOW:
 *   1. Take 22% for research fund
 *   2. Calculate effective weights for all contributors
 *   3. Distribute remaining 78% proportionally to weights
 */
export function distributeRewards(
  rewardPool: bigint,
  contributions: TruthContributionRecord[],
  blockNumber: bigint
): RewardDistribution {
  // 1. Research share FIRST (22%)
  const researchShare = BigInt(Math.floor(Number(rewardPool) * RESEARCH_SHARE_RATE));
  const afterResearch = rewardPool - researchShare;

  // 2. Calculate all weights
  const weightedContributions = contributions.map(c => {
    const weight = calculateContributionWeight(c, contributions);
    return {
      ...c,
      weight: weight.baseWeight,
      counterfactualAdjustment: weight.counterfactualAdjustment,
      antiGamingAdjustment: weight.antiGamingAdjustment,
      effectiveWeight: weight.effectiveWeight,
    };
  });

  // 3. Sum of all effective weights
  const totalWeight = weightedContributions.reduce((sum, c) => sum + c.effectiveWeight, 0);

  // 4. Distribute proportionally
  const distributions: RewardDistribution['distributions'] = [];
  let distributed = 0n;

  if (totalWeight > 0) {
    for (const contribution of weightedContributions) {
      const share = contribution.effectiveWeight / totalWeight;
      const amount = BigInt(Math.floor(Number(afterResearch) * share));

      distributions.push({
        contributor: contribution.contributor,
        contributionId: contribution.id,
        amount,
        effectiveWeight: contribution.effectiveWeight,
        type: contribution.type,
      });

      distributed += amount;
    }
  }

  // 5. Remainder goes to protocol treasury
  const protocolTreasury = afterResearch - distributed;

  // 6. Generate distribution ID
  const distributionId = `dist_${blockNumber}_${Date.now()}`;

  return {
    totalPool: rewardPool,
    researchShare,
    distributedAmount: distributed,
    distributions,
    protocolTreasury,
    blockNumber,
    distributionId,
  };
}

/**
 * Distribute rewards for a specific fact
 * Convenience wrapper that gathers contributions and distributes
 */
export function distributeFactRewards(
  factId: string,
  rewardPool: bigint,
  ledger: ContributionLedger,
  blockNumber: bigint
): RewardDistribution {
  const contributions = ledger.getContributionsForFact(factId);
  return distributeRewards(rewardPool, contributions, blockNumber);
}

// ============ Ledger Implementation ============

/**
 * In-memory contribution ledger implementation
 */
export class InMemoryContributionLedger implements ContributionLedger {
  private contributions: Map<string, TruthContributionRecord> = new Map();
  private byFact: Map<string, string[]> = new Map();
  private byContributor: Map<DID, string[]> = new Map();
  private byType: Map<ContributionType, string[]> = new Map();
  private rewardsByContributor: Map<DID, bigint> = new Map();
  private nextId = 1;

  recordContribution(
    input: Omit<TruthContributionRecord, 'id' | 'weight' | 'counterfactualAdjustment' | 'antiGamingAdjustment' | 'effectiveWeight'>
  ): TruthContributionRecord {
    const id = `contrib_${this.nextId++}`;

    // Get all contributions for context
    const allContributions = Array.from(this.contributions.values());

    // Calculate weight
    const tempRecord: TruthContributionRecord = {
      ...input,
      id,
      weight: 0,
      counterfactualAdjustment: 1,
      antiGamingAdjustment: 1,
      effectiveWeight: 0,
    };

    const weight = calculateContributionWeight(tempRecord, allContributions);

    const record: TruthContributionRecord = {
      ...input,
      id,
      weight: weight.baseWeight,
      counterfactualAdjustment: weight.counterfactualAdjustment,
      antiGamingAdjustment: weight.antiGamingAdjustment,
      effectiveWeight: weight.effectiveWeight,
    };

    // Store
    this.contributions.set(id, record);

    // Index by fact
    if (record.context.factId) {
      const factContribs = this.byFact.get(record.context.factId) || [];
      factContribs.push(id);
      this.byFact.set(record.context.factId, factContribs);
    }

    // Index by contributor
    const contributorContribs = this.byContributor.get(record.contributor) || [];
    contributorContribs.push(id);
    this.byContributor.set(record.contributor, contributorContribs);

    // Index by type
    const typeContribs = this.byType.get(record.type) || [];
    typeContribs.push(id);
    this.byType.set(record.type, typeContribs);

    return record;
  }

  getContributionsForFact(factId: string): TruthContributionRecord[] {
    const ids = this.byFact.get(factId) || [];
    return ids.map(id => this.contributions.get(id)!).filter(Boolean);
  }

  getContributionsByContributor(contributor: DID): TruthContributionRecord[] {
    const ids = this.byContributor.get(contributor) || [];
    return ids.map(id => this.contributions.get(id)!).filter(Boolean);
  }

  getContributionsByType(type: ContributionType): TruthContributionRecord[] {
    const ids = this.byType.get(type) || [];
    return ids.map(id => this.contributions.get(id)!).filter(Boolean);
  }

  distributeRewards(
    rewardPool: bigint,
    context: ContributionContext,
    blockNumber: bigint
  ): RewardDistribution {
    // Get relevant contributions
    let contributions: TruthContributionRecord[];

    if (context.factId) {
      contributions = this.getContributionsForFact(context.factId);
    } else {
      contributions = Array.from(this.contributions.values());
    }

    // Distribute
    const distribution = distributeRewards(rewardPool, contributions, blockNumber);

    // Record rewards for each contributor
    for (const dist of distribution.distributions) {
      const current = this.rewardsByContributor.get(dist.contributor) || 0n;
      this.rewardsByContributor.set(dist.contributor, current + dist.amount);

      // Update the contribution record
      const contribution = this.contributions.get(dist.contributionId);
      if (contribution) {
        contribution.rewardAmount = dist.amount;
      }
    }

    return distribution;
  }

  getTotalRewards(contributor: DID): bigint {
    return this.rewardsByContributor.get(contributor) || 0n;
  }
}

// ============ Exports ============

export {
  RESEARCH_SHARE_RATE,
  RESEARCH_TREASURY_DID,
};
