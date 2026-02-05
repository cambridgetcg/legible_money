/**
 * API Billing Types
 *
 * Types for usage-based billing with automatic knowledge distribution.
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { CitationReward } from "../knowledge/types.js";

/**
 * API call record
 */
export interface APICallRecord {
  callId: string;
  timestamp: number;
  blockNumber: number;

  // Caller
  caller: DID;
  provider: DID;
  endpoint: string;
  method: string;

  // Knowledge used
  factsUsed: Array<{
    factId: string;
    usageWeight: number;
    confidence: number;
  }>;

  // Billing
  totalCost: bigint;
  distribution: PaymentDistribution;

  // Response metadata
  responseConfidence: number;
  responseLatency: number;
}

/**
 * Payment distribution for an API call
 */
export interface PaymentDistribution {
  apiProvider: {
    did: DID;
    amount: bigint;
  };
  knowledgeRewards: CitationReward[];
  protocol: {
    burned: bigint;
    treasury: bigint;
  };
  // Protocol Research Repository share (22% of all rewards)
  research?: {
    amount: bigint;
    treasury: DID;
  };
}

/**
 * API Provider registration
 */
export interface APIProvider {
  did: DID;
  stake: bigint;
  endpoints: APIEndpoint[];

  // Pricing
  basePricePerCall: bigint;
  pricingModel: "per_call" | "per_token" | "per_fact";

  // SLA commitments
  maxLatency: number;
  minUptime: number;
  minConfidence: number;

  // Track record
  totalCalls: number;
  successfulCalls: number;
  averageLatency: number;
  reputationScore: number;

  // Earnings
  totalEarned: bigint;
  pendingPayout: bigint;

  // Status
  status: "active" | "suspended" | "deregistered";
  registeredAt: number;
}

/**
 * API endpoint definition
 */
export interface APIEndpoint {
  path: string;
  method: "GET" | "POST";
  description: string;
  knowledgeDomains: string[];
  priceMultiplier: number;
}

/**
 * Payment channel for streaming payments
 */
export interface PaymentChannel {
  channelId: string;
  payer: DID;
  provider: DID;

  // Funds
  deposited: bigint;
  spent: bigint;
  available: bigint;

  // State
  status: "open" | "closing" | "closed";
  openedAt: number;
  expiresAt: number;

  // Accumulated distributions
  pendingDistributions: PaymentDistribution[];

  // Settlement
  lastSettlementBlock: number;
  settlementFrequency: number;
}

/**
 * API call quote (estimated cost before call)
 */
export interface APIQuote {
  provider: DID;
  endpoint: string;
  estimatedCost: bigint;
  breakdown: {
    baseCost: bigint;
    confidencePremium: bigint;
    fundamentalityPremium: bigint;
    freshnessDiscount: bigint;
  };
  validUntil: number;
  quoteId: string;
}

/**
 * API settlement transaction
 */
export interface APISettlement {
  type: "api_settlement";
  callId: string;
  timestamp: number;
  blockNumber: number;

  // Payment
  totalPayment: bigint;
  payer: DID;

  // Distribution
  transfers: Array<{
    recipient: DID;
    amount: bigint;
    reason:
      | "api_service"
      | "fact_reward"
      | "citation_reward"
      | "protocol_burn"
      | "protocol_treasury";
    factId?: string;
  }>;

  // Verification
  factsUsed: string[];
  distributionHash: string;
}

/**
 * Billing parameters (protocol-governed)
 */
export interface BillingParameters {
  // Distribution splits
  apiProviderShare: number; // Default: 0.60
  knowledgePoolShare: number; // Default: 0.30
  protocolShare: number; // Default: 0.10

  // Protocol split
  burnRate: number; // Of protocol share: 0.50
  treasuryRate: number; // Of protocol share: 0.50

  // Research share (Protocol Research Repository)
  // Applied to ALL rewards before distribution
  researchShareRate: number; // Default: 0.22 (22%)

  // Pricing
  basePricePerCall: bigint; // Minimum fee
  confidencePremiumRate: number; // Multiplier for high confidence
  fundamentalityPremiumRate: number;
  freshnessDiscountRate: number;

  // Limits
  maxPricePerCall: bigint;
  minProviderStake: bigint;
}

export const DEFAULT_BILLING_PARAMETERS: BillingParameters = {
  apiProviderShare: 0.6,
  knowledgePoolShare: 0.3,
  protocolShare: 0.1,

  burnRate: 0.5,
  treasuryRate: 0.5,

  // 22% of ALL rewards go to Protocol Research Repository
  researchShareRate: 0.22,

  basePricePerCall: 1000000000000000n, // 0.001 LGM (18 decimals)
  confidencePremiumRate: 2.0,
  fundamentalityPremiumRate: 0.5,
  freshnessDiscountRate: 0.1,

  maxPricePerCall: 1000000000000000000n, // 1 LGM
  minProviderStake: 1000n * 10n ** 18n, // 1000 LGM
};

/**
 * Knowledge pool distribution parameters
 */
export interface KnowledgeDistributionParams {
  // Primary splits (must sum to 1.0)
  submitterBaseShare: number; // Default: 0.40
  verifierPoolShare: number; // Default: 0.25
  ancestorPoolShare: number; // Default: 0.25
  referencePoolShare: number; // Default: 0.05
  challengePoolShare: number; // Default: 0.05

  // Submitter modifiers
  maxSubmitterShare: number; // Default: 0.60 (cap after bonuses)

  // Ancestor decay
  ancestorDecayRate: number; // Default: 0.50
  maxAncestorDepth: number; // Default: 10

  // Bonuses
  fundamentalityBonusMax: number; // Default: 2.0
  crossDomainBonus: number; // Default: 1.2
  accuracyBonusMax: number; // Default: 1.5
}

export const DEFAULT_KNOWLEDGE_DISTRIBUTION_PARAMS: KnowledgeDistributionParams =
  {
    submitterBaseShare: 0.4,
    verifierPoolShare: 0.25,
    ancestorPoolShare: 0.25,
    referencePoolShare: 0.05,
    challengePoolShare: 0.05,

    maxSubmitterShare: 0.6,

    ancestorDecayRate: 0.5,
    maxAncestorDepth: 10,

    fundamentalityBonusMax: 2.0,
    crossDomainBonus: 1.2,
    accuracyBonusMax: 1.5,
  };

/**
 * Detailed knowledge distribution result
 */
export interface KnowledgeDistribution {
  totalPool: bigint;

  submitter: {
    did: DID;
    amount: bigint;
    bonusApplied: number;
  };

  verifiers: Array<{
    did: DID;
    amount: bigint;
    stakeWeight: number;
    accuracyBonus: number;
  }>;

  ancestors: Array<{
    did: DID;
    factId: string;
    amount: bigint;
    depth: number;
    decayFactor: number;
    fundamentalityBonus: number;
    crossDomainBonus: number;
  }>;

  references: Array<{
    did: DID;
    referenceId: string;
    amount: bigint;
    reliabilityWeight: number;
  }>;

  challengeBonus: {
    submitterBonus: bigint;
    defenders: Array<{
      did: DID;
      amount: bigint;
    }>;
  };

  protocolTreasury: bigint;
}

/**
 * Ancestor information for distribution
 */
export interface AncestorInfo {
  factId: string;
  submitter: DID;
  fundamentalityScore: number;
  domain: string;
  depth: number;
}

/**
 * Challenge information for distribution
 */
export interface ChallengeInfo {
  challengeId: string;
  status: "pending" | "upheld" | "rejected";
  defenders: DID[];
}
