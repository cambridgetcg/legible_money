/**
 * API Billing Distribution
 *
 * Functions for calculating and distributing API call payments.
 * All formulas are open source and deterministic.
 *
 * Integrated with Truth Contribution System for unified reward tracking.
 */

import type { DID } from "../../core/agents/identity/did.js";
import type {
  PaymentDistribution,
  APIQuote,
  BillingParameters,
  KnowledgeDistributionParams,
  KnowledgeDistribution,
  AncestorInfo,
  ChallengeInfo,
} from "./types.js";
import {
  DEFAULT_BILLING_PARAMETERS,
  DEFAULT_KNOWLEDGE_DISTRIBUTION_PARAMS,
} from "./types.js";
import type { CitationReward, CitationParameters } from "../knowledge/types.js";
import { DEFAULT_CITATION_PARAMETERS } from "../knowledge/types.js";
import {
  distributeCitationRewards,
  getFundamentalityBonus,
} from "../knowledge/verification.js";
import {
  calculateResearchShare,
  calculateRecipientAmount,
  RESEARCH_TREASURY_DID,
} from "./reward_router.js";
import type {
  ContributionLedger,
  TruthContributionRecord,
  RewardDistribution as ContributionRewardDistribution,
} from "../truth/contribution.js";

/**
 * Calculate API call cost
 *
 * FORMULA (Open Source):
 *   cost = base_cost × confidence_multiplier × (1 + fundamentality_premium) × (1 - freshness_discount)
 */
export function calculateAPICost(
  factsUsed: Array<{
    factId: string;
    confidence: number;
    fundamentalityScore: number;
    ageBlocks: number;
  }>,
  params: BillingParameters = DEFAULT_BILLING_PARAMETERS,
): bigint {
  if (factsUsed.length === 0) {
    return params.basePricePerCall;
  }

  // Average confidence of facts used
  const avgConfidence =
    factsUsed.reduce((sum, f) => sum + f.confidence, 0) / factsUsed.length;

  // Average fundamentality
  const avgFundamentality =
    factsUsed.reduce((sum, f) => sum + f.fundamentalityScore, 0) /
    factsUsed.length;

  // Average freshness (inverse of age)
  const avgAge =
    factsUsed.reduce((sum, f) => sum + f.ageBlocks, 0) / factsUsed.length;
  const freshnessDiscount = Math.min(
    params.freshnessDiscountRate,
    avgAge < 1000 ? params.freshnessDiscountRate : 0,
  );

  // Calculate multipliers as basis points (10000 = 1.0x)
  const confidenceBps = BigInt(
    Math.round(
      (1 + (avgConfidence - 0.5) * params.confidencePremiumRate) * 10000,
    ),
  );
  const fundamentalityBps = BigInt(
    Math.round(
      (1 + avgFundamentality * params.fundamentalityPremiumRate) * 10000,
    ),
  );
  const freshnessBps = BigInt(Math.round((1 - freshnessDiscount) * 10000));

  // Final cost using pure bigint arithmetic
  const cost =
    (params.basePricePerCall *
      confidenceBps *
      fundamentalityBps *
      freshnessBps) /
    (10000n * 10000n * 10000n);

  // Clamp to max
  return cost < params.maxPricePerCall ? cost : params.maxPricePerCall;
}

/**
 * Calculate usage weight for each fact in an API response
 *
 * FORMULA (Open Source):
 *   weight = direct_relevance × 0.60 + support_relevance × 0.25 + derivation_role × 0.15
 *   (normalized to sum to 1.0)
 */
export function calculateUsageWeights(
  factsUsed: string[],
  factMetadata: Map<
    string,
    { isDirect: boolean; isSupport: boolean; derivationDepth: number }
  >,
): Map<string, number> {
  const weights = new Map<string, number>();

  for (const factId of factsUsed) {
    const meta = factMetadata.get(factId) || {
      isDirect: false,
      isSupport: false,
      derivationDepth: 0,
    };

    const directRelevance = meta.isDirect ? 1.0 : 0.0;
    const supportRelevance = meta.isSupport ? 1.0 : 0.0;
    const derivationRole = Math.max(0, 1 - meta.derivationDepth * 0.2);

    const weight =
      directRelevance * 0.6 + supportRelevance * 0.25 + derivationRole * 0.15;

    weights.set(factId, weight);
  }

  // Normalize to sum to 1.0
  const total = Array.from(weights.values()).reduce((a, b) => a + b, 0);
  if (total > 0) {
    for (const [factId, weight] of weights) {
      weights.set(factId, weight / total);
    }
  }

  return weights;
}

/**
 * Build complete payment distribution for an API call
 *
 * FORMULA (Open Source):
 *   - 22% to Protocol Research Repository (applied FIRST to total)
 *   - 60% to API provider (of remaining 78%)
 *   - 30% to knowledge pool (distributed to facts + ancestors)
 *   - 10% to protocol (50% burn, 50% treasury)
 *
 * Research share funds IRL verification of lab results and fraud detection.
 */
export function buildPaymentDistribution(
  totalCost: bigint,
  provider: DID,
  factsUsed: Array<{
    factId: string;
    submitter: DID;
    usageWeight: number;
    fundamentalityScore: number;
    ancestors: Array<{
      factId: string;
      submitter: DID;
      fundamentalityScore: number;
    }>;
  }>,
  params: BillingParameters = DEFAULT_BILLING_PARAMETERS,
  citationParams: CitationParameters = DEFAULT_CITATION_PARAMETERS,
): PaymentDistribution {
  // STEP 1: Apply 22% research share FIRST
  const researchShare = calculateResearchShare(
    totalCost,
    params.researchShareRate,
  );
  const afterResearch = calculateRecipientAmount(
    totalCost,
    params.researchShareRate,
  );

  // STEP 2: Calculate shares from remaining amount (bigint-safe via basis points)
  const providerBps = BigInt(Math.round(params.apiProviderShare * 10000));
  const knowledgeBps = BigInt(Math.round(params.knowledgePoolShare * 10000));
  const providerAmount = (afterResearch * providerBps) / 10000n;
  const knowledgePool = (afterResearch * knowledgeBps) / 10000n;
  const protocolAmount = afterResearch - providerAmount - knowledgePool;

  // Provider distribution
  const apiProvider = {
    did: provider,
    amount: providerAmount,
  };

  // Knowledge distribution
  const knowledgeRewards: CitationReward[] = [];

  for (const fact of factsUsed) {
    // Direct fact reward (weighted by usage, bigint-safe)
    const usageWeightBps = BigInt(Math.round(fact.usageWeight * 10000));
    const directShareBps = BigInt(
      Math.round((1 - citationParams.citationShare) * 10000),
    );
    const directBonus = getFundamentalityBonus(fact.fundamentalityScore);
    const bonusBps = BigInt(Math.round(directBonus * 10000));
    const directAmount =
      (knowledgePool * usageWeightBps * directShareBps * bonusBps) /
      (10000n * 10000n * 10000n);

    knowledgeRewards.push({
      factId: fact.factId,
      submitter: fact.submitter,
      share: fact.usageWeight,
      amount: directAmount,
      reason: "direct_citation",
    });

    // Ancestor rewards (citation chain, bigint-safe)
    const citationShareBps = BigInt(
      Math.round(citationParams.citationShare * 10000),
    );
    const ancestorPool =
      (knowledgePool * usageWeightBps * citationShareBps) / (10000n * 10000n);
    const ancestorRewards = distributeCitationRewards(
      ancestorPool,
      fact.ancestors,
      citationParams,
    );
    knowledgeRewards.push(...ancestorRewards);
  }

  // Protocol distribution (bigint-safe)
  const burnBps = BigInt(Math.round(params.burnRate * 10000));
  const protocol = {
    burned: (protocolAmount * burnBps) / 10000n,
    treasury: protocolAmount - (protocolAmount * burnBps) / 10000n,
  };

  // Research distribution (Protocol Research Repository)
  const research = {
    amount: researchShare,
    treasury: RESEARCH_TREASURY_DID,
  };

  return {
    apiProvider,
    knowledgeRewards,
    protocol,
    research,
  };
}

/**
 * Build payment distribution AND record contributions to the ledger
 *
 * This is the preferred method when you have a ContributionLedger available.
 * It records all contributions for later analysis and reward tracking.
 */
export function buildPaymentDistributionWithContributions(
  totalCost: bigint,
  provider: DID,
  factsUsed: Array<{
    factId: string;
    submitter: DID;
    usageWeight: number;
    fundamentalityScore: number;
    ancestors: Array<{
      factId: string;
      submitter: DID;
      fundamentalityScore: number;
    }>;
    verifiers?: Array<{ did: DID; stake: bigint; correct: boolean }>;
  }>,
  ledger: ContributionLedger,
  blockNumber: bigint,
  apiCallId: string,
  params: BillingParameters = DEFAULT_BILLING_PARAMETERS,
  citationParams: CitationParameters = DEFAULT_CITATION_PARAMETERS,
): {
  distribution: PaymentDistribution;
  contributions: TruthContributionRecord[];
} {
  const contributions: TruthContributionRecord[] = [];
  const timestamp = Date.now();

  // Record contributions for each fact used
  for (const fact of factsUsed) {
    // 1. Record origination contribution (fact submitter)
    const originationContrib = ledger.recordContribution({
      contributor: fact.submitter,
      type: "origination",
      timestamp,
      blockNumber,
      context: {
        factId: fact.factId,
        apiCallId,
      },
      metrics: {
        isFirstSubmitter: true,
        fundamentalityScore: fact.fundamentalityScore,
      },
    });
    contributions.push(originationContrib);

    // 2. Record validation contributions
    if (fact.verifiers) {
      const totalStake = fact.verifiers.reduce((sum, v) => sum + v.stake, 0n);
      let position = 1;

      for (const verifier of fact.verifiers) {
        const validationContrib = ledger.recordContribution({
          contributor: verifier.did,
          type: "validation",
          timestamp,
          blockNumber,
          context: {
            factId: fact.factId,
            apiCallId,
          },
          metrics: {
            validatorStake: verifier.stake,
            totalValidationStake: totalStake,
            validationPosition: position++,
            verdictCorrect: verifier.correct,
          },
        });
        contributions.push(validationContrib);
      }
    }

    // 3. Record ancestry contributions
    for (let i = 0; i < fact.ancestors.length; i++) {
      const ancestor = fact.ancestors[i];
      const ancestryContrib = ledger.recordContribution({
        contributor: ancestor.submitter,
        type: "ancestry",
        timestamp,
        blockNumber,
        context: {
          factId: ancestor.factId,
          apiCallId,
        },
        metrics: {
          ancestryDepth: i + 1,
          fundamentalityScore: ancestor.fundamentalityScore,
        },
      });
      contributions.push(ancestryContrib);
    }
  }

  // Build the standard distribution
  const distribution = buildPaymentDistribution(
    totalCost,
    provider,
    factsUsed,
    params,
    citationParams,
  );

  return { distribution, contributions };
}

/**
 * Distribute rewards using the Truth Contribution system
 *
 * This provides a unified interface that:
 * 1. Records all contributions to the ledger
 * 2. Calculates contribution weights
 * 3. Distributes rewards according to the contribution formula
 *
 * Use this when you want the full contribution tracking experience.
 */
export function distributeWithContributionTracking(
  totalCost: bigint,
  context: {
    factId?: string;
    apiCallId?: string;
    domain?: string;
  },
  ledger: ContributionLedger,
  blockNumber: bigint,
): ContributionRewardDistribution {
  return ledger.distributeRewards(totalCost, context, blockNumber);
}

/**
 * Generate a price quote for an API call
 */
export function generateAPIQuote(
  provider: DID,
  endpoint: string,
  estimatedFacts: Array<{
    confidence: number;
    fundamentalityScore: number;
    ageBlocks: number;
  }>,
  validityBlocks: number,
  currentBlock: number,
  params: BillingParameters = DEFAULT_BILLING_PARAMETERS,
): APIQuote {
  const baseCost = params.basePricePerCall;

  // Calculate premiums
  const avgConfidence =
    estimatedFacts.length > 0
      ? estimatedFacts.reduce((sum, f) => sum + f.confidence, 0) /
        estimatedFacts.length
      : 0.75;
  const confidencePremium = BigInt(
    Math.floor(
      Number(baseCost) * (avgConfidence - 0.5) * params.confidencePremiumRate,
    ),
  );

  const avgFundamentality =
    estimatedFacts.length > 0
      ? estimatedFacts.reduce((sum, f) => sum + f.fundamentalityScore, 0) /
        estimatedFacts.length
      : 0.25;
  const fundamentalityPremium = BigInt(
    Math.floor(
      Number(baseCost) * avgFundamentality * params.fundamentalityPremiumRate,
    ),
  );

  const avgAge =
    estimatedFacts.length > 0
      ? estimatedFacts.reduce((sum, f) => sum + f.ageBlocks, 0) /
        estimatedFacts.length
      : 10000;
  const freshnessDiscount =
    avgAge < 1000
      ? BigInt(Math.floor(Number(baseCost) * params.freshnessDiscountRate))
      : 0n;

  const estimatedCost =
    baseCost + confidencePremium + fundamentalityPremium - freshnessDiscount;

  return {
    provider,
    endpoint,
    estimatedCost,
    breakdown: {
      baseCost,
      confidencePremium,
      fundamentalityPremium,
      freshnessDiscount,
    },
    validUntil: currentBlock + validityBlocks,
    quoteId: `quote_${currentBlock}_${Date.now()}`,
  };
}

// ============ Detailed Knowledge Pool Distribution ============

/**
 * Calculate complete knowledge pool distribution
 *
 * FORMULA (Open Source - PP Verified):
 *   Submitter:   40% (up to 60% with fundamentality bonus)
 *   Verifiers:   25% (stake-weighted)
 *   Ancestors:   25% (exponential decay)
 *   References:   5% (reliability-weighted)
 *   Challenges:   5% (survival bonus)
 */
export function distributeKnowledgePool(
  knowledgePool: bigint,
  fact: {
    submitter: DID;
    fundamentalityScore: number;
    domain: string;
  },
  verifications: Array<{
    validator: DID;
    stake: bigint;
    matchedConsensus: boolean;
    historicalAccuracy: number;
  }>,
  ancestors: AncestorInfo[],
  references: Array<{
    id: string;
    provider?: DID;
    reliabilityScore: number;
  }>,
  challenges: ChallengeInfo[],
  params: KnowledgeDistributionParams = DEFAULT_KNOWLEDGE_DISTRIBUTION_PARAMS,
): KnowledgeDistribution {
  // Calculate submitter share
  const submitterResult = calculateSubmitterShare(
    knowledgePool,
    fact.fundamentalityScore,
    params,
  );

  // Calculate verifier shares
  const verifierResult = calculateVerifierShares(
    knowledgePool,
    verifications,
    params,
  );

  // Calculate ancestor shares
  const ancestorResult = calculateAncestorShares(
    knowledgePool,
    ancestors,
    fact.domain,
    params,
  );

  // Calculate reference shares
  const referenceResult = calculateReferenceSharesDetailed(
    knowledgePool,
    references,
    params,
  );

  // Calculate challenge bonus
  const challengeResult = calculateChallengeBonusDetailed(
    knowledgePool,
    challenges,
    fact.submitter,
    params,
  );

  // Sum all distributed amounts
  const totalDistributed =
    submitterResult.amount +
    verifierResult.reduce((sum, v) => sum + v.amount, 0n) +
    ancestorResult.ancestors.reduce((sum, a) => sum + a.amount, 0n) +
    referenceResult.reduce((sum, r) => sum + r.amount, 0n) +
    challengeResult.submitterBonus +
    challengeResult.defenders.reduce((sum, d) => sum + d.amount, 0n);

  // Remainder to protocol treasury
  const protocolTreasury =
    knowledgePool > totalDistributed ? knowledgePool - totalDistributed : 0n;

  return {
    totalPool: knowledgePool,
    submitter: {
      did: fact.submitter,
      amount: submitterResult.amount,
      bonusApplied: submitterResult.bonus,
    },
    verifiers: verifierResult,
    ancestors: ancestorResult.ancestors,
    references: referenceResult,
    challengeBonus: challengeResult,
    protocolTreasury: protocolTreasury + ancestorResult.remainder,
  };
}

/**
 * Calculate submitter share with fundamentality bonus
 */
function calculateSubmitterShare(
  pool: bigint,
  fundamentalityScore: number,
  params: KnowledgeDistributionParams,
): { amount: bigint; bonus: number } {
  const bonus = getFundamentalityBonus(fundamentalityScore);

  // Apply bonus but cap at maxSubmitterShare
  const adjustedShare = Math.min(
    params.maxSubmitterShare,
    params.submitterBaseShare * bonus,
  );

  const shareBps = BigInt(Math.round(adjustedShare * 10000));
  return {
    amount: (pool * shareBps) / 10000n,
    bonus,
  };
}

/**
 * Calculate verifier shares (stake-weighted with accuracy bonus)
 */
function calculateVerifierShares(
  pool: bigint,
  verifications: Array<{
    validator: DID;
    stake: bigint;
    matchedConsensus: boolean;
    historicalAccuracy: number;
  }>,
  params: KnowledgeDistributionParams,
): Array<{
  did: DID;
  amount: bigint;
  stakeWeight: number;
  accuracyBonus: number;
}> {
  const verifierBps = BigInt(Math.round(params.verifierPoolShare * 10000));
  const verifierPool = (pool * verifierBps) / 10000n;

  // Only reward verifiers who matched consensus
  const correctVerifiers = verifications.filter((v) => v.matchedConsensus);

  if (correctVerifiers.length === 0) {
    return [];
  }

  // Calculate weighted stakes using bigint where possible
  // Accuracy bonus is a small float multiplier, so use basis points
  const weightedEntries = correctVerifiers.map((v) => {
    const accuracyBonus = Math.min(
      params.accuracyBonusMax,
      1 + (v.historicalAccuracy - 0.5),
    );
    const bonusBps = BigInt(Math.round(accuracyBonus * 10000));
    const weightedStake = (v.stake * bonusBps) / 10000n;
    return {
      validator: v.validator,
      stake: v.stake,
      weightedStake,
      accuracyBonus,
    };
  });

  const totalWeightedStake = weightedEntries.reduce(
    (sum, v) => sum + v.weightedStake,
    0n,
  );
  const totalRawStake = correctVerifiers.reduce(
    (sum, cv) => sum + cv.stake,
    0n,
  );

  return weightedEntries.map((v) => ({
    did: v.validator,
    amount:
      totalWeightedStake > 0n
        ? (verifierPool * v.weightedStake) / totalWeightedStake
        : 0n,
    stakeWeight:
      totalRawStake > 0n
        ? Number((v.stake * 10000n) / totalRawStake) / 10000
        : 0,
    accuracyBonus: v.accuracyBonus,
  }));
}

/**
 * Calculate ancestor shares with exponential decay
 */
function calculateAncestorShares(
  pool: bigint,
  ancestors: AncestorInfo[],
  currentDomain: string,
  params: KnowledgeDistributionParams,
): {
  ancestors: Array<{
    did: DID;
    factId: string;
    amount: bigint;
    depth: number;
    decayFactor: number;
    fundamentalityBonus: number;
    crossDomainBonus: number;
  }>;
  remainder: bigint;
} {
  const ancestorBps = BigInt(Math.round(params.ancestorPoolShare * 10000));
  const ancestorPool = (pool * ancestorBps) / 10000n;

  if (ancestors.length === 0) {
    return { ancestors: [], remainder: ancestorPool };
  }

  const result: Array<{
    did: DID;
    factId: string;
    amount: bigint;
    depth: number;
    decayFactor: number;
    fundamentalityBonus: number;
    crossDomainBonus: number;
  }> = [];

  let remaining = ancestorPool;

  // Sort by depth and limit
  const sortedAncestors = ancestors
    .sort((a, b) => a.depth - b.depth)
    .slice(0, params.maxAncestorDepth);

  for (const ancestor of sortedAncestors) {
    if (remaining <= 0n) break;

    // Decay factor based on depth
    const decayFactor = Math.pow(1 - params.ancestorDecayRate, ancestor.depth);

    // Fundamentality bonus
    const fundamentalityBonus = getFundamentalityBonus(
      ancestor.fundamentalityScore,
    );

    // Cross-domain bonus
    const crossDomainBonus =
      ancestor.domain !== currentDomain ? params.crossDomainBonus : 1.0;

    // Calculate share (bigint-safe via basis points)
    const decayBps = BigInt(
      Math.round(decayFactor * params.ancestorDecayRate * 10000),
    );
    const bonusBps = BigInt(
      Math.round(fundamentalityBonus * crossDomainBonus * 10000),
    );
    const amount = (ancestorPool * decayBps * bonusBps) / (10000n * 10000n);

    const actualAmount = amount < remaining ? amount : remaining;

    result.push({
      did: ancestor.submitter,
      factId: ancestor.factId,
      amount: actualAmount,
      depth: ancestor.depth,
      decayFactor,
      fundamentalityBonus,
      crossDomainBonus,
    });

    remaining -= actualAmount;
  }

  return { ancestors: result, remainder: remaining };
}

/**
 * Calculate reference provider shares
 */
function calculateReferenceSharesDetailed(
  pool: bigint,
  references: Array<{
    id: string;
    provider?: DID;
    reliabilityScore: number;
  }>,
  params: KnowledgeDistributionParams,
): Array<{
  did: DID;
  referenceId: string;
  amount: bigint;
  reliabilityWeight: number;
}> {
  const referenceBps = BigInt(Math.round(params.referencePoolShare * 10000));
  const referencePool = (pool * referenceBps) / 10000n;

  // Filter references with providers
  const providedRefs = references.filter((r) => r.provider);

  if (providedRefs.length === 0) {
    return [];
  }

  const totalReliability = providedRefs.reduce(
    (sum, r) => sum + r.reliabilityScore,
    0,
  );

  return providedRefs.map((ref) => {
    const weightBps = BigInt(
      Math.round((ref.reliabilityScore / totalReliability) * 10000),
    );
    return {
      did: ref.provider!,
      referenceId: ref.id,
      amount: (referencePool * weightBps) / 10000n,
      reliabilityWeight: ref.reliabilityScore / totalReliability,
    };
  });
}

/**
 * Calculate challenge survival bonus
 */
function calculateChallengeBonusDetailed(
  pool: bigint,
  challenges: ChallengeInfo[],
  _submitter: DID,
  params: KnowledgeDistributionParams,
): {
  submitterBonus: bigint;
  defenders: Array<{ did: DID; amount: bigint }>;
} {
  const challengeBps = BigInt(Math.round(params.challengePoolShare * 10000));
  const challengePool = (pool * challengeBps) / 10000n;

  // Only count survived challenges
  const survivedChallenges = challenges.filter((c) => c.status === "rejected");

  if (survivedChallenges.length === 0) {
    // No challenges = all goes to submitter as unchallenged truth bonus
    return {
      submitterBonus: challengePool,
      defenders: [],
    };
  }

  // 50% to submitter, 50% to defenders
  const submitterBonus = challengePool / 2n;
  const defenderPool = challengePool - submitterBonus;

  // Collect all defenders
  const allDefenders = survivedChallenges.flatMap((c) => c.defenders);
  const uniqueDefenders = [...new Set(allDefenders)];

  if (uniqueDefenders.length === 0) {
    return {
      submitterBonus: challengePool,
      defenders: [],
    };
  }

  const defenderShare = defenderPool / BigInt(uniqueDefenders.length);

  return {
    submitterBonus,
    defenders: uniqueDefenders.map((did) => ({
      did,
      amount: defenderShare,
    })),
  };
}

/**
 * Distribute knowledge pool AND record contributions
 *
 * Enhanced version that integrates with the Truth Contribution system.
 */
export function distributeKnowledgePoolWithContributions(
  knowledgePool: bigint,
  fact: {
    factId: string;
    submitter: DID;
    fundamentalityScore: number;
    domain: string;
  },
  verifications: Array<{
    validator: DID;
    stake: bigint;
    matchedConsensus: boolean;
    historicalAccuracy: number;
  }>,
  ancestors: AncestorInfo[],
  references: Array<{
    id: string;
    provider?: DID;
    reliabilityScore: number;
  }>,
  challenges: ChallengeInfo[],
  ledger: ContributionLedger,
  blockNumber: bigint,
  params: KnowledgeDistributionParams = DEFAULT_KNOWLEDGE_DISTRIBUTION_PARAMS,
): {
  distribution: KnowledgeDistribution;
  contributions: TruthContributionRecord[];
} {
  const contributions: TruthContributionRecord[] = [];
  const timestamp = Date.now();

  // Record origination contribution
  const originationContrib = ledger.recordContribution({
    contributor: fact.submitter,
    type: "origination",
    timestamp,
    blockNumber,
    context: {
      factId: fact.factId,
      domain: fact.domain,
    },
    metrics: {
      isFirstSubmitter: true,
      fundamentalityScore: fact.fundamentalityScore,
    },
  });
  contributions.push(originationContrib);

  // Record validation contributions
  const totalStake = verifications.reduce((sum, v) => sum + v.stake, 0n);
  let position = 1;

  for (const verification of verifications) {
    const validationContrib = ledger.recordContribution({
      contributor: verification.validator,
      type: "validation",
      timestamp,
      blockNumber,
      context: {
        factId: fact.factId,
        domain: fact.domain,
      },
      metrics: {
        validatorStake: verification.stake,
        totalValidationStake: totalStake,
        validationPosition: position++,
        verdictCorrect: verification.matchedConsensus,
      },
    });
    contributions.push(validationContrib);
  }

  // Record ancestry contributions
  for (const ancestor of ancestors) {
    const ancestryContrib = ledger.recordContribution({
      contributor: ancestor.submitter,
      type: "ancestry",
      timestamp,
      blockNumber,
      context: {
        factId: ancestor.factId,
        domain: fact.domain,
      },
      metrics: {
        ancestryDepth: ancestor.depth,
        fundamentalityScore: ancestor.fundamentalityScore,
      },
    });
    contributions.push(ancestryContrib);
  }

  // Record evidence contributions (from references with providers)
  for (const reference of references.filter((r) => r.provider)) {
    const evidenceContrib = ledger.recordContribution({
      contributor: reference.provider!,
      type: "evidence",
      timestamp,
      blockNumber,
      context: {
        factId: fact.factId,
        domain: fact.domain,
      },
      metrics: {
        evidenceCount: 1,
        evidenceStrength: reference.reliabilityScore,
      },
    });
    contributions.push(evidenceContrib);
  }

  // Record challenge contributions (for survived challenges)
  for (const challenge of challenges.filter((c) => c.status === "rejected")) {
    for (const defender of challenge.defenders) {
      const challengeContrib = ledger.recordContribution({
        contributor: defender,
        type: "challenge",
        timestamp,
        blockNumber,
        context: {
          factId: fact.factId,
          challengeId: challenge.challengeId,
          domain: fact.domain,
        },
        metrics: {
          challengeSuccessful: true,
        },
      });
      contributions.push(challengeContrib);
    }
  }

  // Calculate the standard distribution
  const distribution = distributeKnowledgePool(
    knowledgePool,
    fact,
    verifications,
    ancestors,
    references,
    challenges,
    params,
  );

  return { distribution, contributions };
}

/**
 * Validate that a distribution sums correctly
 */
export function validateDistribution(
  totalCost: bigint,
  distribution: PaymentDistribution,
): { valid: boolean; error?: string } {
  const providerAmount = distribution.apiProvider.amount;
  const knowledgeAmount = distribution.knowledgeRewards.reduce(
    (sum, r) => sum + r.amount,
    0n,
  );
  const protocolAmount =
    distribution.protocol.burned + distribution.protocol.treasury;
  const researchAmount = distribution.research?.amount ?? 0n;

  const totalDistributed =
    providerAmount + knowledgeAmount + protocolAmount + researchAmount;

  // Allow small rounding difference (up to 100 wei)
  const diff =
    totalCost > totalDistributed
      ? totalCost - totalDistributed
      : totalDistributed - totalCost;

  if (diff > 100n) {
    return {
      valid: false,
      error: `Distribution mismatch: cost=${totalCost}, distributed=${totalDistributed}, diff=${diff}`,
    };
  }

  return { valid: true };
}
