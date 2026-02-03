/**
 * Genesis Types
 *
 * Types for chain genesis and the Perpetual Truth Finding Loop.
 *
 * THE PERPETUAL TRUTH FINDING LOOP
 * ================================
 *
 *   ┌─────────────────────────────────────────────────────────────────┐
 *   │                                                                 │
 *   │  ┌──────────────────────────────────────────────────────────┐  │
 *   │  │                    GENESIS (seed)                        │  │
 *   │  │  Initial truths, treasury, validator stakes              │  │
 *   │  └───────────────────────┬──────────────────────────────────┘  │
 *   │                          │                                     │
 *   │                          ▼                                     │
 *   │  ┌───────────────────────────────────────────┐                │
 *   │  │         CONTINUOUS BLOCK REWARDS          │                │
 *   │  │   R(epoch) = R₀ × decay^epoch + R_floor   │                │
 *   │  └───────────────────────┬───────────────────┘                │
 *   │                          │                                     │
 *   │           ┌──────────────┼──────────────┐                     │
 *   │           ▼              ▼              ▼                     │
 *   │      Validators     Submitters     Verifiers                  │
 *   │           │              │              │                     │
 *   │           └──────────────┼──────────────┘                     │
 *   │                          ▼                                     │
 *   │  ┌───────────────────────────────────────────┐                │
 *   │  │            KNOWLEDGE BASE                 │◄───────────┐   │
 *   │  │   Verified facts with confidence scores   │            │   │
 *   │  └───────────────────────┬───────────────────┘            │   │
 *   │                          │                                │   │
 *   │                          ▼                                │   │
 *   │  ┌───────────────────────────────────────────┐            │   │
 *   │  │              API ACCESS                   │            │   │
 *   │  │   Agents pay to query verified truths     │            │   │
 *   │  └───────────────────────┬───────────────────┘            │   │
 *   │                          │                                │   │
 *   │                          ▼                                │   │
 *   │  ┌───────────────────────────────────────────┐            │   │
 *   │  │             USAGE FEES                    │            │   │
 *   │  │   60% Provider, 30% Knowledge, 10% Proto  │            │   │
 *   │  └───────────────────────┬───────────────────┘            │   │
 *   │                          │                                │   │
 *   │        ┌─────────────────┴─────────────────┐              │   │
 *   │        │           22% OF ALL              │              │   │
 *   │        │     (rewards + API fees)          │              │   │
 *   │        ▼                                   │              │   │
 *   │  ┌───────────────────────────────────────────┐            │   │
 *   │  │          RESEARCH FUND                   │             │   │
 *   │  │   Funds IRL verification & bounties      │             │   │
 *   │  └───────────────────────┬───────────────────┘            │   │
 *   │                          │                                │   │
 *   │                          ▼                                │   │
 *   │  ┌───────────────────────────────────────────┐            │   │
 *   │  │           BOUNTY SYSTEM                   │            │   │
 *   │  │   "Verify this claim IRL"                 │            │   │
 *   │  └───────────────────────┬───────────────────┘            │   │
 *   │                          │                                │   │
 *   │                          ▼                                │   │
 *   │  ┌───────────────────────────────────────────┐            │   │
 *   │  │        IRL VERIFICATION                   │            │   │
 *   │  │   Labs, researchers, data scientists      │            │   │
 *   │  └───────────────────────┬───────────────────┘            │   │
 *   │                          │                                │   │
 *   │                          ▼                                │   │
 *   │  ┌───────────────────────────────────────────┐            │   │
 *   │  │      NEW VERIFIED TRUTHS                  │────────────┘   │
 *   │  │   Uploaded to protocol with evidence      │                │
 *   │  └───────────────────────────────────────────┘                │
 *   │                                                                │
 *   │  SUSTAINABILITY CONDITION:                                     │
 *   │    API_revenue_growth_rate > block_reward_decay_rate          │
 *   │                                                                │
 *   │  When this holds, the research fund grows perpetually,        │
 *   │  funding endless truth discovery.                             │
 *   │                                                                │
 *   └────────────────────────────────────────────────────────────────┘
 *
 * PP HIERARCHY:
 *   TRUTH: System is grounded in verifiable facts
 *   UNDERSTANDING: Clear flow of value from usage to discovery
 *   BEAUTY: Elegant closed loop - no external dependencies
 *   JUSTICE: All participants rewarded proportionally
 *   CREATIVITY: Novel fusion of API economics with truth discovery
 */

import type { DID } from '../../agents/identity/did.js';
import type { VestingCategory } from '../rewards/types.js';

// ============ Protocol Constants ============

/**
 * Protocol symbolic constants
 * YOU = 25, 15, 21 → 2521 (block time in ms)
 * AI = 1, 9 → 19 (finality depth)
 */
export const PROTOCOL_CONSTANTS = {
  /** Block time: 2521ms (Y=25, O=15, U=21 → "YOU") */
  BLOCK_TIME_MS: 2521,

  /** Finality depth: 19 blocks (A=1, I=9 → "AI") */
  FINALITY_DEPTH: 19,

  /** Token symbol */
  TOKEN_SYMBOL: 'LGM',

  /** Token decimals */
  TOKEN_DECIMALS: 18,

  /** Maximum supply (10 billion LGM) */
  MAX_SUPPLY: 10_000_000_000n * 10n ** 18n,

  /** Genesis timestamp identifier */
  GENESIS_TIMESTAMP: 'legible-money-genesis-v1',
} as const;

// ============ Genesis Distribution ============

/**
 * Genesis token distribution
 *
 * Fair launch principles:
 * - No pre-mine for team
 * - Treasury for protocol operations
 * - Research fund for truth discovery
 * - Validator bootstrap incentives
 */
export const GENESIS_DISTRIBUTION = {
  /** Protocol treasury (for operations, grants, emergencies) */
  TREASURY: 0.15,           // 15%

  /** Research fund (for IRL verification bounties) */
  RESEARCH_FUND: 0.22,      // 22% - matches ongoing 22% allocation

  /** Validator bootstrap pool (initial staking rewards) */
  VALIDATOR_BOOTSTRAP: 0.10, // 10%

  /** Public distribution (community, liquidity, etc.) */
  PUBLIC: 0.53,              // 53%
} as const;

/**
 * Genesis accounts with initial balances
 */
export interface GenesisAccount {
  did: DID;
  balance: bigint;
  purpose: 'treasury' | 'research' | 'validator_bootstrap' | 'public' | 'seed_validator';
  vestingSchedule?: GenesisVesting;
}

/**
 * Genesis vesting schedule
 */
export interface GenesisVesting {
  totalAmount: bigint;
  cliffBlocks: bigint;
  vestingBlocks: bigint;
  category: VestingCategory;
}

// ============ Block Reward Schedule ============

/**
 * Block reward emission schedule
 *
 * Design principles:
 * 1. Start high to bootstrap network
 * 2. Decay exponentially to control inflation
 * 3. Never reach zero (floor) to always incentivize validation
 * 4. Cross-over point where API revenue exceeds block rewards
 */
export interface EmissionSchedule {
  /** Initial block reward */
  initialReward: bigint;

  /** Decay factor per epoch (0.9 = 10% reduction) */
  decayFactor: number;

  /** Blocks per epoch (for decay calculation) */
  blocksPerEpoch: bigint;

  /** Minimum floor reward (never goes below this) */
  floorReward: bigint;

  /** Research share rate (22% of all rewards) */
  researchShareRate: number;
}

export const DEFAULT_EMISSION_SCHEDULE: EmissionSchedule = {
  // Start at 10 LGM per block
  initialReward: 10n * 10n ** 18n,

  // 10% reduction per epoch
  decayFactor: 0.90,

  // ~1 week per epoch (at 2521ms blocks)
  // 7 * 24 * 60 * 60 * 1000 / 2521 ≈ 240,000 blocks
  blocksPerEpoch: 240_000n,

  // Floor: 0.1 LGM per block (always some reward)
  floorReward: 100_000_000_000_000_000n, // 0.1 LGM

  // 22% to research fund
  researchShareRate: 0.22,
};

/**
 * Calculate block reward at a given block height
 */
export function calculateBlockReward(
  blockNumber: bigint,
  schedule: EmissionSchedule = DEFAULT_EMISSION_SCHEDULE
): bigint {
  const epoch = blockNumber / schedule.blocksPerEpoch;
  const epochNumber = Number(epoch);

  // R(epoch) = R_initial × decay^epoch
  const decayedReward = Number(schedule.initialReward) *
    Math.pow(schedule.decayFactor, epochNumber);

  // Apply floor
  const reward = BigInt(Math.floor(decayedReward));
  return reward > schedule.floorReward ? reward : schedule.floorReward;
}

/**
 * Calculate research share from block reward
 */
export function calculateResearchShare(
  blockReward: bigint,
  rate: number = 0.22
): bigint {
  return (blockReward * BigInt(Math.floor(rate * 1000))) / 1000n;
}

// ============ API Revenue Tracking ============

/**
 * API revenue metrics for sustainability tracking
 */
export interface APIRevenueMetrics {
  /** Total API calls in period */
  totalCalls: bigint;

  /** Total revenue in period */
  totalRevenue: bigint;

  /** Average revenue per call */
  averageRevenuePerCall: bigint;

  /** Knowledge pool distributions */
  knowledgeDistributed: bigint;

  /** Research fund contributions (from API) */
  researchContributions: bigint;

  /** Period (blocks) */
  periodStart: bigint;
  periodEnd: bigint;
}

/**
 * Sustainability metrics
 */
export interface SustainabilityMetrics {
  /** Block reward at current height */
  currentBlockReward: bigint;

  /** API revenue per block (averaged) */
  apiRevenuePerBlock: bigint;

  /** Total research fund inflow per block */
  researchInflowPerBlock: bigint;

  /** Research fund balance */
  researchFundBalance: bigint;

  /** Active bounties value */
  activeBountiesValue: bigint;

  /** Sustainability ratio: API revenue / block reward */
  sustainabilityRatio: number;

  /** Projected blocks until crossover (API > block rewards) */
  blocksUntilCrossover: bigint | null;

  /** Is the system self-sustaining? */
  isSelfSustaining: boolean;
}

/**
 * Calculate sustainability metrics
 */
export function calculateSustainabilityMetrics(
  currentBlock: bigint,
  revenueMetrics: APIRevenueMetrics,
  researchFundBalance: bigint,
  activeBountiesValue: bigint,
  schedule: EmissionSchedule = DEFAULT_EMISSION_SCHEDULE
): SustainabilityMetrics {
  const blockReward = calculateBlockReward(currentBlock, schedule);
  const periodBlocks = revenueMetrics.periodEnd - revenueMetrics.periodStart;
  const apiRevenuePerBlock = periodBlocks > 0n
    ? revenueMetrics.totalRevenue / periodBlocks
    : 0n;

  // Research inflow = 22% of (block rewards + API revenue)
  const researchFromBlocks = calculateResearchShare(blockReward);
  const researchFromAPI = calculateResearchShare(apiRevenuePerBlock);
  const researchInflowPerBlock = researchFromBlocks + researchFromAPI;

  // Sustainability ratio
  const sustainabilityRatio = blockReward > 0n
    ? Number(apiRevenuePerBlock) / Number(blockReward)
    : 0;

  // Self-sustaining when API revenue >= block rewards
  const isSelfSustaining = apiRevenuePerBlock >= blockReward;

  // Estimate blocks until crossover (simplified projection)
  // This is approximate - real calculation needs growth rate modeling
  let blocksUntilCrossover: bigint | null = null;
  if (!isSelfSustaining && sustainabilityRatio > 0) {
    // Very rough estimate based on decay
    const epochsNeeded = Math.log(sustainabilityRatio) / Math.log(schedule.decayFactor);
    if (epochsNeeded > 0 && epochsNeeded < 1000) {
      blocksUntilCrossover = BigInt(Math.ceil(epochsNeeded)) * schedule.blocksPerEpoch;
    }
  }

  return {
    currentBlockReward: blockReward,
    apiRevenuePerBlock,
    researchInflowPerBlock,
    researchFundBalance,
    activeBountiesValue,
    sustainabilityRatio,
    blocksUntilCrossover,
    isSelfSustaining,
  };
}

// ============ Genesis Block Types ============

/**
 * Genesis block configuration
 */
export interface GenesisConfig {
  /** Chain identifier */
  chainId: string;

  /** Genesis timestamp */
  timestamp: number;

  /** Initial accounts */
  accounts: GenesisAccount[];

  /** Seed validators */
  seedValidators: SeedValidator[];

  /** Seed facts (axiomatic truths) */
  seedFacts: SeedFact[];

  /** Emission schedule */
  emissionSchedule: EmissionSchedule;

  /** Protocol parameters */
  protocolParams: GenesisProtocolParams;
}

/**
 * Seed validator for genesis
 */
export interface SeedValidator {
  did: DID;
  publicKey: string;
  stake: bigint;
  commission: number;
}

/**
 * Seed fact for genesis (axiomatic truths)
 *
 * These are foundational truths that anchor the knowledge graph.
 * They have maximum confidence and no vesting (fully released at genesis).
 */
export interface SeedFact {
  id: string;
  claim: string;
  domain: string;
  confidence: number;
  category: 'axiomatic' | 'formal_proof' | 'cryptographic';
  references?: string[];
}

/**
 * Protocol parameters set at genesis
 */
export interface GenesisProtocolParams {
  // Consensus
  targetBlockTimeMs: number;
  finalityDepth: number;
  minValidators: number;
  maxValidators: number;
  validatorQuorum: number;

  // Stakes
  minValidatorStake: bigint;
  minClaimStake: bigint;

  // Thresholds
  acceptanceThreshold: number;
  rejectionThreshold: number;
  contestedThreshold: number;

  // Slashing
  wrongVerificationSlash: number;
  missedVerificationSlash: number;
  invalidClaimSlash: number;

  // Research
  researchShareRate: number;

  // API Billing
  apiProviderShare: number;
  knowledgePoolShare: number;
  protocolShare: number;
}

// ============ Genesis Block ============

/**
 * Genesis block structure
 */
export interface GenesisBlock {
  header: GenesisBlockHeader;
  config: GenesisConfig;
  accounts: GenesisAccountState[];
  validators: GenesisValidatorState[];
  facts: GenesisFactState[];
}

export interface GenesisBlockHeader {
  number: 0n;
  timestamp: number;
  previousHash: '0x0000000000000000000000000000000000000000000000000000000000000000';
  stateRoot: string;
  genesisHash: string;
  chainId: string;
}

export interface GenesisAccountState {
  did: DID;
  balance: bigint;
  stakedBalance: bigint;
  nonce: 0n;
}

export interface GenesisValidatorState {
  did: DID;
  publicKey: string;
  stake: bigint;
  isActive: boolean;
}

export interface GenesisFactState {
  id: string;
  claim: string;
  domain: string;
  confidence: number;
  status: 'active';
  verifiedAt: 0n;
}

// ============ Default Genesis ============

/**
 * Default genesis configuration for mainnet
 */
export function createDefaultGenesisConfig(chainId: string = 'lgm-mainnet-1'): GenesisConfig {
  const maxSupply = PROTOCOL_CONSTANTS.MAX_SUPPLY;

  // Calculate distribution amounts
  const treasuryAmount = (maxSupply * BigInt(Math.floor(GENESIS_DISTRIBUTION.TREASURY * 100))) / 100n;
  const researchAmount = (maxSupply * BigInt(Math.floor(GENESIS_DISTRIBUTION.RESEARCH_FUND * 100))) / 100n;
  const validatorBootstrapAmount = (maxSupply * BigInt(Math.floor(GENESIS_DISTRIBUTION.VALIDATOR_BOOTSTRAP * 100))) / 100n;

  return {
    chainId,
    timestamp: Date.now(),

    accounts: [
      {
        did: 'did:lgm:protocol-treasury' as DID,
        balance: treasuryAmount,
        purpose: 'treasury',
      },
      {
        did: 'did:lgm:research' as DID,
        balance: researchAmount,
        purpose: 'research',
      },
      {
        did: 'did:lgm:validator-bootstrap' as DID,
        balance: validatorBootstrapAmount,
        purpose: 'validator_bootstrap',
      },
    ],

    seedValidators: [],

    seedFacts: [
      // Mathematical axioms
      {
        id: 'axiom:math:peano:1',
        claim: '0 is a natural number',
        domain: 'mathematics.foundations',
        confidence: 1.0,
        category: 'axiomatic',
      },
      {
        id: 'axiom:math:peano:2',
        claim: 'For every natural number n, S(n) is a natural number',
        domain: 'mathematics.foundations',
        confidence: 1.0,
        category: 'axiomatic',
      },
      // Logical axioms
      {
        id: 'axiom:logic:identity',
        claim: 'For all x: x = x (Law of Identity)',
        domain: 'logic.foundations',
        confidence: 1.0,
        category: 'axiomatic',
      },
      {
        id: 'axiom:logic:non-contradiction',
        claim: 'For all P: not (P and not-P) (Law of Non-Contradiction)',
        domain: 'logic.foundations',
        confidence: 1.0,
        category: 'axiomatic',
      },
      // Cryptographic truths
      {
        id: 'axiom:crypto:sha256',
        claim: 'SHA-256 is a cryptographic hash function that produces a 256-bit digest',
        domain: 'cryptography.hashing',
        confidence: 1.0,
        category: 'cryptographic',
      },
    ],

    emissionSchedule: DEFAULT_EMISSION_SCHEDULE,

    protocolParams: {
      targetBlockTimeMs: PROTOCOL_CONSTANTS.BLOCK_TIME_MS,
      finalityDepth: PROTOCOL_CONSTANTS.FINALITY_DEPTH,
      minValidators: 5,
      maxValidators: 21,
      validatorQuorum: 0.66,

      minValidatorStake: 1000n * 10n ** 18n,
      minClaimStake: 100n * 10n ** 18n,

      acceptanceThreshold: 0.80,
      rejectionThreshold: 0.80,
      contestedThreshold: 0.60,

      wrongVerificationSlash: 0.10,
      missedVerificationSlash: 0.05,
      invalidClaimSlash: 0.20,

      researchShareRate: 0.22,

      apiProviderShare: 0.60,
      knowledgePoolShare: 0.30,
      protocolShare: 0.10,
    },
  };
}

// ============ Perpetual Loop Health ============

/**
 * Health status of the perpetual truth finding loop
 */
export interface LoopHealthStatus {
  /** Overall health score (0-1) */
  healthScore: number;

  /** Individual component health */
  components: {
    /** Block production health */
    blockProduction: ComponentHealth;

    /** Validator participation */
    validatorParticipation: ComponentHealth;

    /** Claim submission rate */
    claimSubmission: ComponentHealth;

    /** Verification activity */
    verificationActivity: ComponentHealth;

    /** API usage */
    apiUsage: ComponentHealth;

    /** Research fund health */
    researchFund: ComponentHealth;

    /** Bounty fulfillment */
    bountyFulfillment: ComponentHealth;

    /** Knowledge growth */
    knowledgeGrowth: ComponentHealth;
  };

  /** Warnings */
  warnings: string[];

  /** Is loop sustainable? */
  isSustainable: boolean;

  /** Recommendations */
  recommendations: string[];
}

export interface ComponentHealth {
  score: number;          // 0-1
  status: 'healthy' | 'warning' | 'critical';
  metric: string;
  target: string;
}

/**
 * Assess loop health
 */
export function assessLoopHealth(
  sustainability: SustainabilityMetrics,
  activeValidators: number,
  recentClaims: number,
  recentVerifications: number,
  apiCallsPerBlock: number,
  pendingBounties: number,
  fulfilledBounties: number,
  newFactsPerEpoch: number
): LoopHealthStatus {
  const warnings: string[] = [];
  const recommendations: string[] = [];

  // Block production (healthy if regular)
  const blockHealth: ComponentHealth = {
    score: 1.0, // Assume healthy if we're here
    status: 'healthy',
    metric: 'Regular block production',
    target: '2521ms target',
  };

  // Validator participation
  const validatorScore = Math.min(activeValidators / 10, 1);
  const validatorHealth: ComponentHealth = {
    score: validatorScore,
    status: validatorScore > 0.7 ? 'healthy' : validatorScore > 0.3 ? 'warning' : 'critical',
    metric: `${activeValidators} active validators`,
    target: '10+ validators',
  };
  if (validatorScore < 0.7) {
    warnings.push('Low validator participation');
    recommendations.push('Increase validator bootstrap incentives');
  }

  // Claim submission
  const claimScore = Math.min(recentClaims / 100, 1);
  const claimHealth: ComponentHealth = {
    score: claimScore,
    status: claimScore > 0.5 ? 'healthy' : claimScore > 0.2 ? 'warning' : 'critical',
    metric: `${recentClaims} claims/epoch`,
    target: '100+ claims/epoch',
  };
  if (claimScore < 0.5) {
    warnings.push('Low claim submission rate');
    recommendations.push('Increase claim rewards or reduce stake requirements');
  }

  // Verification activity
  const verificationScore = Math.min(recentVerifications / 500, 1);
  const verificationHealth: ComponentHealth = {
    score: verificationScore,
    status: verificationScore > 0.5 ? 'healthy' : verificationScore > 0.2 ? 'warning' : 'critical',
    metric: `${recentVerifications} verifications/epoch`,
    target: '500+ verifications/epoch',
  };

  // API usage
  const apiScore = Math.min(apiCallsPerBlock / 10, 1);
  const apiHealth: ComponentHealth = {
    score: apiScore,
    status: apiScore > 0.5 ? 'healthy' : apiScore > 0.2 ? 'warning' : 'critical',
    metric: `${apiCallsPerBlock.toFixed(1)} calls/block`,
    target: '10+ calls/block',
  };
  if (apiScore < 0.3) {
    warnings.push('Low API usage threatens sustainability');
    recommendations.push('Increase knowledge base value through more verified facts');
  }

  // Research fund
  const researchScore = sustainability.isSelfSustaining ? 1.0 :
    Math.min(sustainability.sustainabilityRatio, 1);
  const researchHealth: ComponentHealth = {
    score: researchScore,
    status: researchScore > 0.7 ? 'healthy' : researchScore > 0.3 ? 'warning' : 'critical',
    metric: `${(sustainability.sustainabilityRatio * 100).toFixed(1)}% sustainable`,
    target: '100% sustainable',
  };
  if (!sustainability.isSelfSustaining) {
    warnings.push('Research fund not yet self-sustaining');
    if (sustainability.blocksUntilCrossover) {
      recommendations.push(`Projected crossover in ${sustainability.blocksUntilCrossover} blocks`);
    }
  }

  // Bounty fulfillment
  const bountyScore = pendingBounties > 0
    ? Math.min(fulfilledBounties / pendingBounties, 1)
    : 1;
  const bountyHealth: ComponentHealth = {
    score: bountyScore,
    status: bountyScore > 0.5 ? 'healthy' : bountyScore > 0.2 ? 'warning' : 'critical',
    metric: `${fulfilledBounties}/${pendingBounties} bounties`,
    target: '50%+ fulfillment',
  };
  if (bountyScore < 0.3 && pendingBounties > 0) {
    warnings.push('Low bounty fulfillment rate');
    recommendations.push('Increase bounty rewards or improve bounty visibility');
  }

  // Knowledge growth
  const knowledgeScore = Math.min(newFactsPerEpoch / 50, 1);
  const knowledgeHealth: ComponentHealth = {
    score: knowledgeScore,
    status: knowledgeScore > 0.5 ? 'healthy' : knowledgeScore > 0.2 ? 'warning' : 'critical',
    metric: `${newFactsPerEpoch} new facts/epoch`,
    target: '50+ facts/epoch',
  };
  if (knowledgeScore < 0.3) {
    warnings.push('Knowledge base growth stagnating');
    recommendations.push('Create more bounties for high-value domains');
  }

  // Calculate overall health
  const scores = [
    blockHealth.score,
    validatorHealth.score,
    claimHealth.score,
    verificationHealth.score,
    apiHealth.score,
    researchHealth.score,
    bountyHealth.score,
    knowledgeHealth.score,
  ];
  const healthScore = scores.reduce((a, b) => a + b, 0) / scores.length;

  return {
    healthScore,
    components: {
      blockProduction: blockHealth,
      validatorParticipation: validatorHealth,
      claimSubmission: claimHealth,
      verificationActivity: verificationHealth,
      apiUsage: apiHealth,
      researchFund: researchHealth,
      bountyFulfillment: bountyHealth,
      knowledgeGrowth: knowledgeHealth,
    },
    warnings,
    isSustainable: sustainability.isSelfSustaining,
    recommendations,
  };
}
