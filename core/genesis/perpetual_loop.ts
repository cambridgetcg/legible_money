/**
 * Perpetual Truth Finding Loop Engine
 *
 * Orchestrates the closed economic loop that makes truth discovery self-sustaining:
 *
 * API Usage → Fees → Research Fund → Bounties → IRL Verification → New Truths → API Value ↑
 *
 * THE CORE INSIGHT:
 * ================
 * Every API call that retrieves truth funds the discovery of new truth.
 * As block rewards decay (inflation control), API revenue grows (utility value).
 * The crossover point marks protocol maturity - when utility sustains the truth engine.
 *
 * PP HIERARCHY:
 *   TRUTH: Grounded in economic reality - truth discovery costs resources
 *   UNDERSTANDING: Clear value flow through each loop stage
 *   BEAUTY: Elegant closed loop that mirrors natural knowledge ecosystems
 *   JUSTICE: All participants compensated proportionally to truth contribution
 *   CREATIVITY: Novel fusion of API economics with epistemological infrastructure
 *
 * CORE PRINCIPLE: "Intention does not matter, result does."
 * Whether participants seek profit or truth, the system produces verified knowledge.
 */

import {
  type EmissionSchedule,
  type APIRevenueMetrics,
  type SustainabilityMetrics,
  type LoopHealthStatus,
  type GenesisConfig,
  calculateSustainabilityMetrics,
  assessLoopHealth,
} from './types.js';

// ============ Loop State ============

/**
 * Complete state of the perpetual loop
 */
export interface LoopState {
  /** Current block */
  currentBlock: bigint;

  /** Current epoch */
  currentEpoch: bigint;

  /** Emission schedule */
  emissionSchedule: EmissionSchedule;

  /** Cumulative block rewards emitted */
  totalBlockRewardsEmitted: bigint;

  /** API revenue metrics (rolling window) */
  apiRevenueMetrics: APIRevenueMetrics;

  /** Research fund state */
  researchFund: ResearchFundState;

  /** Bounty pool state */
  bountyPool: BountyPoolState;

  /** Knowledge base state */
  knowledgeBase: KnowledgeBaseState;

  /** Sustainability metrics */
  sustainability: SustainabilityMetrics;

  /** Loop health */
  health: LoopHealthStatus;
}

/**
 * Research fund state
 */
export interface ResearchFundState {
  /** Total balance available */
  balance: bigint;

  /** Cumulative inflow (all time) */
  totalInflow: bigint;

  /** Cumulative outflow to bounties */
  totalBountyFunding: bigint;

  /** Cumulative outflow to infrastructure */
  totalInfrastructureFunding: bigint;

  /** Reserve for emergencies */
  emergencyReserve: bigint;

  /** Inflow per block (rolling average) */
  inflowPerBlock: bigint;
}

/**
 * Bounty pool state
 */
export interface BountyPoolState {
  /** Total value of active bounties */
  activeBountiesValue: bigint;

  /** Number of active bounties */
  activeBountiesCount: number;

  /** Bounties created this epoch */
  bountiesCreatedThisEpoch: number;

  /** Bounties fulfilled this epoch */
  bountiesFulfilledThisEpoch: number;

  /** Average bounty value */
  averageBountyValue: bigint;

  /** Average time to fulfillment (blocks) */
  averageFulfillmentTime: bigint;
}

/**
 * Knowledge base state
 */
export interface KnowledgeBaseState {
  /** Total verified facts */
  totalFacts: number;

  /** Facts added this epoch */
  factsAddedThisEpoch: number;

  /** Facts challenged this epoch */
  factsChallengedThisEpoch: number;

  /** Facts falsified this epoch */
  factsFalsifiedThisEpoch: number;

  /** Average fact confidence */
  averageConfidence: number;

  /** Knowledge domains active */
  activeDomains: number;

  /** Most valuable facts (by citation revenue) */
  topRevenueGenerators: string[];
}

// ============ Loop Metrics ============

/**
 * Calculate loop efficiency metrics
 */
export interface LoopEfficiency {
  /** Research fund conversion rate (bounties fulfilled / funded) */
  bountyConversionRate: number;

  /** Truth discovery cost (research spent / new facts) */
  truthDiscoveryCost: bigint;

  /** API value per fact */
  apiValuePerFact: bigint;

  /** Knowledge monetization rate (API revenue / fact count) */
  knowledgeMonetizationRate: bigint;

  /** Loop velocity (how fast value circulates) */
  loopVelocity: number;
}

export function calculateLoopEfficiency(state: LoopState): LoopEfficiency {
  const bountyConversionRate = state.bountyPool.activeBountiesCount > 0
    ? state.bountyPool.bountiesFulfilledThisEpoch / state.bountyPool.bountiesCreatedThisEpoch
    : 0;

  const truthDiscoveryCost = state.knowledgeBase.factsAddedThisEpoch > 0
    ? state.researchFund.totalBountyFunding / BigInt(state.knowledgeBase.factsAddedThisEpoch)
    : 0n;

  const apiValuePerFact = state.knowledgeBase.totalFacts > 0
    ? state.apiRevenueMetrics.totalRevenue / BigInt(state.knowledgeBase.totalFacts)
    : 0n;

  const knowledgeMonetizationRate = state.knowledgeBase.totalFacts > 0
    ? state.apiRevenueMetrics.totalRevenue / BigInt(state.knowledgeBase.totalFacts)
    : 0n;

  // Loop velocity: how many times value circulates per epoch
  // API revenue → research → bounty → fact → API (one loop)
  const loopVelocity = state.researchFund.totalInflow > 0n
    ? Number(state.apiRevenueMetrics.totalRevenue) / Number(state.researchFund.totalInflow)
    : 0;

  return {
    bountyConversionRate,
    truthDiscoveryCost,
    apiValuePerFact,
    knowledgeMonetizationRate,
    loopVelocity,
  };
}

// ============ Loop Events ============

/**
 * Events that affect the loop
 */
export type LoopEvent =
  | BlockProducedEvent
  | APICallEvent
  | ClaimSubmittedEvent
  | ClaimVerifiedEvent
  | BountyCreatedEvent
  | BountyFulfilledEvent
  | ResearchSubmittedEvent
  | FactFalsifiedEvent;

export interface BlockProducedEvent {
  type: 'block_produced';
  blockNumber: bigint;
  blockReward: bigint;
  researchShare: bigint;
  validatorRewards: Map<string, bigint>;
}

export interface APICallEvent {
  type: 'api_call';
  callId: string;
  revenue: bigint;
  factsUsed: string[];
  researchShare: bigint;
  knowledgeDistribution: Map<string, bigint>;
}

export interface ClaimSubmittedEvent {
  type: 'claim_submitted';
  claimId: string;
  submitter: string;
  stake: bigint;
  domain: string;
}

export interface ClaimVerifiedEvent {
  type: 'claim_verified';
  claimId: string;
  outcome: 'accepted' | 'rejected' | 'contested';
  factId?: string;
  rewards: Map<string, bigint>;
  researchShare: bigint;
}

export interface BountyCreatedEvent {
  type: 'bounty_created';
  bountyId: string;
  value: bigint;
  targetClaim: string;
  domain: string;
}

export interface BountyFulfilledEvent {
  type: 'bounty_fulfilled';
  bountyId: string;
  fulfiller: string;
  researchId: string;
  outcome: 'confirms' | 'refutes' | 'inconclusive' | 'fraud_detected';
}

export interface ResearchSubmittedEvent {
  type: 'research_submitted';
  researchId: string;
  researcher: string;
  targetClaim: string;
  outcome: 'confirms' | 'refutes' | 'inconclusive' | 'fraud_detected';
}

export interface FactFalsifiedEvent {
  type: 'fact_falsified';
  factId: string;
  challenger: string;
  clawbackAmount: bigint;
}

// ============ Loop Controller ============

/**
 * Perpetual Loop Controller
 *
 * Monitors and maintains the health of the truth-finding loop.
 */
export class LoopController {
  private state: LoopState;
  private eventHistory: LoopEvent[] = [];

  constructor(config: GenesisConfig) {
    this.state = this.initializeState(config);
  }

  private initializeState(config: GenesisConfig): LoopState {
    return {
      currentBlock: 0n,
      currentEpoch: 0n,
      emissionSchedule: config.emissionSchedule,
      totalBlockRewardsEmitted: 0n,
      apiRevenueMetrics: {
        totalCalls: 0n,
        totalRevenue: 0n,
        averageRevenuePerCall: 0n,
        knowledgeDistributed: 0n,
        researchContributions: 0n,
        periodStart: 0n,
        periodEnd: 0n,
      },
      researchFund: {
        balance: 0n,
        totalInflow: 0n,
        totalBountyFunding: 0n,
        totalInfrastructureFunding: 0n,
        emergencyReserve: 0n,
        inflowPerBlock: 0n,
      },
      bountyPool: {
        activeBountiesValue: 0n,
        activeBountiesCount: 0,
        bountiesCreatedThisEpoch: 0,
        bountiesFulfilledThisEpoch: 0,
        averageBountyValue: 0n,
        averageFulfillmentTime: 0n,
      },
      knowledgeBase: {
        totalFacts: config.seedFacts.length,
        factsAddedThisEpoch: 0,
        factsChallengedThisEpoch: 0,
        factsFalsifiedThisEpoch: 0,
        averageConfidence: 1.0, // Seed facts are axiomatic
        activeDomains: new Set(config.seedFacts.map(f => f.domain)).size,
        topRevenueGenerators: [],
      },
      sustainability: this.calculateInitialSustainability(config),
      health: this.calculateInitialHealth(config),
    };
  }

  private calculateInitialSustainability(config: GenesisConfig): SustainabilityMetrics {
    return calculateSustainabilityMetrics(
      0n,
      {
        totalCalls: 0n,
        totalRevenue: 0n,
        averageRevenuePerCall: 0n,
        knowledgeDistributed: 0n,
        researchContributions: 0n,
        periodStart: 0n,
        periodEnd: 0n,
      },
      0n,
      0n,
      config.emissionSchedule
    );
  }

  private calculateInitialHealth(config: GenesisConfig): LoopHealthStatus {
    return assessLoopHealth(
      this.calculateInitialSustainability(config),
      0, // validators
      0, // claims
      0, // verifications
      0, // api calls
      0, // pending bounties
      0, // fulfilled bounties
      config.seedFacts.length // seed facts
    );
  }

  /**
   * Process a loop event
   */
  processEvent(event: LoopEvent): void {
    this.eventHistory.push(event);

    switch (event.type) {
      case 'block_produced':
        this.handleBlockProduced(event);
        break;
      case 'api_call':
        this.handleAPICall(event);
        break;
      case 'claim_verified':
        this.handleClaimVerified(event);
        break;
      case 'bounty_created':
        this.handleBountyCreated(event);
        break;
      case 'bounty_fulfilled':
        this.handleBountyFulfilled(event);
        break;
      case 'fact_falsified':
        this.handleFactFalsified(event);
        break;
    }

    this.updateMetrics();
  }

  private handleBlockProduced(event: BlockProducedEvent): void {
    this.state.currentBlock = event.blockNumber;
    this.state.currentEpoch = event.blockNumber / this.state.emissionSchedule.blocksPerEpoch;
    this.state.totalBlockRewardsEmitted += event.blockReward;
    this.state.researchFund.balance += event.researchShare;
    this.state.researchFund.totalInflow += event.researchShare;
  }

  private handleAPICall(event: APICallEvent): void {
    this.state.apiRevenueMetrics.totalCalls += 1n;
    this.state.apiRevenueMetrics.totalRevenue += event.revenue;
    this.state.apiRevenueMetrics.researchContributions += event.researchShare;
    this.state.researchFund.balance += event.researchShare;
    this.state.researchFund.totalInflow += event.researchShare;

    // Update knowledge distribution
    for (const [_factId, amount] of event.knowledgeDistribution) {
      this.state.apiRevenueMetrics.knowledgeDistributed += amount;
    }
  }

  private handleClaimVerified(event: ClaimVerifiedEvent): void {
    if (event.outcome === 'accepted' && event.factId) {
      this.state.knowledgeBase.totalFacts++;
      this.state.knowledgeBase.factsAddedThisEpoch++;
    }
    this.state.researchFund.balance += event.researchShare;
    this.state.researchFund.totalInflow += event.researchShare;
  }

  private handleBountyCreated(event: BountyCreatedEvent): void {
    this.state.bountyPool.activeBountiesValue += event.value;
    this.state.bountyPool.activeBountiesCount++;
    this.state.bountyPool.bountiesCreatedThisEpoch++;
    this.state.researchFund.balance -= event.value;
    this.state.researchFund.totalBountyFunding += event.value;
  }

  private handleBountyFulfilled(event: BountyFulfilledEvent): void {
    this.state.bountyPool.activeBountiesCount--;
    this.state.bountyPool.bountiesFulfilledThisEpoch++;

    if (event.outcome === 'confirms') {
      this.state.knowledgeBase.factsAddedThisEpoch++;
    } else if (event.outcome === 'fraud_detected') {
      this.state.knowledgeBase.factsFalsifiedThisEpoch++;
    }
  }

  private handleFactFalsified(_event: FactFalsifiedEvent): void {
    this.state.knowledgeBase.totalFacts--;
    this.state.knowledgeBase.factsFalsifiedThisEpoch++;
  }

  private updateMetrics(): void {
    // Update rolling averages
    const blocks = this.state.currentBlock - this.state.apiRevenueMetrics.periodStart;
    if (blocks > 0n) {
      this.state.apiRevenueMetrics.averageRevenuePerCall =
        this.state.apiRevenueMetrics.totalCalls > 0n
          ? this.state.apiRevenueMetrics.totalRevenue / this.state.apiRevenueMetrics.totalCalls
          : 0n;

      this.state.researchFund.inflowPerBlock =
        this.state.researchFund.totalInflow / blocks;
    }

    this.state.apiRevenueMetrics.periodEnd = this.state.currentBlock;

    // Recalculate sustainability
    this.state.sustainability = calculateSustainabilityMetrics(
      this.state.currentBlock,
      this.state.apiRevenueMetrics,
      this.state.researchFund.balance,
      this.state.bountyPool.activeBountiesValue,
      this.state.emissionSchedule
    );

    // Recalculate health
    const periodBlocks = this.state.apiRevenueMetrics.periodEnd -
      this.state.apiRevenueMetrics.periodStart;
    const apiCallsPerBlock = periodBlocks > 0n
      ? Number(this.state.apiRevenueMetrics.totalCalls) / Number(periodBlocks)
      : 0;

    this.state.health = assessLoopHealth(
      this.state.sustainability,
      0, // TODO: get from validator state
      this.state.knowledgeBase.factsAddedThisEpoch,
      0, // TODO: get from verification state
      apiCallsPerBlock,
      this.state.bountyPool.activeBountiesCount,
      this.state.bountyPool.bountiesFulfilledThisEpoch,
      this.state.knowledgeBase.factsAddedThisEpoch
    );
  }

  /**
   * Get current loop state
   */
  getState(): LoopState {
    return { ...this.state };
  }

  /**
   * Get loop efficiency
   */
  getEfficiency(): LoopEfficiency {
    return calculateLoopEfficiency(this.state);
  }

  /**
   * Get sustainability forecast
   */
  getSustainabilityForecast(futureBlocks: bigint): SustainabilityMetrics {
    const futureBlock = this.state.currentBlock + futureBlocks;

    // Project API revenue growth (simplified linear projection)
    const currentRevenue = this.state.apiRevenueMetrics.totalRevenue;
    const blocks = this.state.apiRevenueMetrics.periodEnd -
      this.state.apiRevenueMetrics.periodStart;
    const revenueGrowthRate = blocks > 0n ? currentRevenue / blocks : 0n;

    const projectedRevenue = currentRevenue + (revenueGrowthRate * futureBlocks);

    const projectedMetrics: APIRevenueMetrics = {
      ...this.state.apiRevenueMetrics,
      totalRevenue: projectedRevenue,
      periodEnd: futureBlock,
    };

    return calculateSustainabilityMetrics(
      futureBlock,
      projectedMetrics,
      this.state.researchFund.balance,
      this.state.bountyPool.activeBountiesValue,
      this.state.emissionSchedule
    );
  }

  /**
   * Get recommendations for improving loop health
   */
  getRecommendations(): string[] {
    const recommendations: string[] = [];
    const efficiency = this.getEfficiency();

    if (efficiency.bountyConversionRate < 0.3) {
      recommendations.push('Increase bounty rewards to attract more researchers');
    }

    if (efficiency.truthDiscoveryCost > 100n * 10n ** 18n) {
      recommendations.push('Research is expensive - consider batch verification');
    }

    if (efficiency.knowledgeMonetizationRate < 1n * 10n ** 18n) {
      recommendations.push('Knowledge is undermonetized - increase API adoption');
    }

    if (!this.state.sustainability.isSelfSustaining) {
      recommendations.push('Focus on API adoption to achieve sustainability');
    }

    if (this.state.knowledgeBase.factsFalsifiedThisEpoch >
        this.state.knowledgeBase.factsAddedThisEpoch * 0.1) {
      recommendations.push('High falsification rate - tighten verification standards');
    }

    return [...recommendations, ...this.state.health.recommendations];
  }

  /**
   * Reset epoch counters (called at epoch boundary)
   */
  resetEpochCounters(): void {
    this.state.bountyPool.bountiesCreatedThisEpoch = 0;
    this.state.bountyPool.bountiesFulfilledThisEpoch = 0;
    this.state.knowledgeBase.factsAddedThisEpoch = 0;
    this.state.knowledgeBase.factsChallengedThisEpoch = 0;
    this.state.knowledgeBase.factsFalsifiedThisEpoch = 0;
  }
}

// ============ Utility Functions ============

/**
 * Calculate optimal bounty value for a claim
 *
 * Based on:
 * - Expected verification cost
 * - Potential API revenue from fact
 * - Research fund balance
 */
export function calculateOptimalBountyValue(
  verificationCostEstimate: bigint,
  _expectedFactValue: bigint,
  researchFundBalance: bigint,
  urgency: 'low' | 'medium' | 'high' = 'medium'
): bigint {
  // Base: cover verification cost
  let bounty = verificationCostEstimate;

  // Add profit margin for researcher
  const profitMargin = 0.20; // 20%
  bounty += (bounty * BigInt(Math.floor(profitMargin * 100))) / 100n;

  // Adjust for urgency
  const urgencyMultiplier = urgency === 'high' ? 1.5 : urgency === 'medium' ? 1.0 : 0.75;
  bounty = (bounty * BigInt(Math.floor(urgencyMultiplier * 100))) / 100n;

  // Cap at 1% of research fund
  const maxBounty = researchFundBalance / 100n;
  if (bounty > maxBounty) {
    bounty = maxBounty;
  }

  // Minimum viable bounty
  const minBounty = 10n * 10n ** 18n; // 10 LGM
  if (bounty < minBounty) {
    bounty = minBounty;
  }

  return bounty;
}

/**
 * Prioritize claims for bounty creation
 *
 * Higher priority for:
 * - High-value domains (frequently queried)
 * - Low confidence facts (need verification)
 * - Long time since verification
 */
export function prioritizeClaimsForBounties(
  claims: Array<{
    claimId: string;
    domain: string;
    confidence: number;
    lastVerifiedBlock: bigint;
    queryCount: number;
  }>,
  currentBlock: bigint
): string[] {
  const scored = claims.map(claim => {
    // Query frequency score (normalized)
    const queryScore = Math.log(claim.queryCount + 1) / Math.log(100);

    // Confidence score (lower = higher priority)
    const confidenceScore = 1 - claim.confidence;

    // Staleness score (older = higher priority)
    const blocksAgo = Number(currentBlock - claim.lastVerifiedBlock);
    const stalenessScore = Math.min(blocksAgo / 1000000, 1);

    const totalScore = (queryScore * 0.4) + (confidenceScore * 0.3) + (stalenessScore * 0.3);

    return { claimId: claim.claimId, score: totalScore };
  });

  scored.sort((a, b) => b.score - a.score);
  return scored.map(s => s.claimId);
}
