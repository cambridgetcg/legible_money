/**
 * Knowledge Verification
 *
 * Logic for verifying claims, calculating confidence, and managing the knowledge layer.
 * All formulas are open source and deterministic.
 */

import type { DID } from '../../agents/identity/did.js';
import type {
  Fact,
  Verification,
  Reference,
  Derivation,
  ClaimSubmission,
  Challenge,
  VerificationTask,
  TruthParameters,
  Confidence,
  InferenceType,
  Citation,
  FundamentalityMetrics,
  CitationReward,
  CitationParameters,
} from './types.js';
import {
  INFERENCE_STRENGTH,
  DEFAULT_TRUTH_PARAMETERS,
  DEFAULT_CITATION_PARAMETERS,
  FUNDAMENTALITY_THRESHOLDS,
  FUNDAMENTALITY_BONUSES,
} from './types.js';

/**
 * Calculate confidence from verifications
 *
 * FORMULA (Open Source):
 *   confidence = weighted_agreement × evidence_strength × time_factor
 *
 * WHERE:
 *   weighted_agreement = sum(stake_i × verdict_i) / sum(stake_i)
 *   evidence_strength = avg(reference_reliability)
 *   time_factor = 1.0 - (blocks_since_verification × decay_rate)
 */
export function calculateConfidence(
  verifications: Verification[],
  references: Reference[],
  blocksSinceVerification: number,
  params: TruthParameters = DEFAULT_TRUTH_PARAMETERS
): Confidence {
  if (verifications.length === 0) return 0;

  // Weighted agreement (stake-weighted)
  let totalStake = 0n;
  let confirmedStake = 0n;

  for (const v of verifications) {
    totalStake += v.stake;
    if (v.verdict === 'confirmed') {
      confirmedStake += v.stake;
    }
  }

  const weightedAgreement = totalStake > 0n
    ? Number(confirmedStake) / Number(totalStake)
    : 0;

  // Evidence strength (average reference reliability)
  const evidenceStrength = references.length > 0
    ? references.reduce((sum, r) => sum + r.reliabilityScore, 0) / references.length
    : 0.5;  // Default if no references

  // Time decay
  const timeFactor = Math.max(
    0,
    1.0 - (blocksSinceVerification * params.confidenceDecayRate)
  );

  // Combine factors
  const confidence = weightedAgreement * evidenceStrength * timeFactor;

  // Clamp to [0, 1]
  return Math.max(0, Math.min(1, confidence));
}

/**
 * Calculate derived confidence from premises
 *
 * FORMULA (Open Source):
 *   derived_confidence = min(premise_confidences) × inference_strength
 *
 * Derivations are only as strong as their weakest premise.
 */
export function calculateDerivedConfidence(
  premiseConfidences: Confidence[],
  inferenceType: InferenceType
): Confidence {
  if (premiseConfidences.length === 0) return 0;

  const minPremiseConfidence = Math.min(...premiseConfidences);
  const inferenceStrength = INFERENCE_STRENGTH[inferenceType];

  return minPremiseConfidence * inferenceStrength;
}

/**
 * Calculate reference reliability score
 *
 * FORMULA (Open Source):
 *   reliability = type_base × freshness × verification_count_factor
 */
export function calculateReferenceReliability(
  reference: Reference,
  currentBlock: number,
  verificationCount: number
): number {
  // Base reliability by type
  const typeBase: Record<string, number> = {
    on_chain: 1.0,       // On-chain data is definitive
    computational: 0.95, // Deterministic computation
    institutional: 0.85, // Published papers, official data
    oracle: 0.80,        // Trusted oracles
    archived_web: 0.70,  // Web archives
  };

  const base = typeBase[reference.type] || 0.5;

  // Freshness (newer = more reliable for empirical claims)
  const ageBlocks = currentBlock - reference.retrievedAt;
  const freshness = Math.max(0.5, 1.0 - (ageBlocks * 0.00001));

  // Verification count factor (more verifications = more reliable)
  const verificationFactor = Math.min(1.0, 0.5 + (verificationCount * 0.1));

  return base * freshness * verificationFactor;
}

/**
 * Check if verification quorum is reached
 */
export function isQuorumReached(
  verifications: Verification[],
  params: TruthParameters = DEFAULT_TRUTH_PARAMETERS
): { reached: boolean; consensus: 'confirmed' | 'rejected' | 'none' } {
  if (verifications.length < params.minVerifiers) {
    return { reached: false, consensus: 'none' };
  }

  let totalStake = 0n;
  let confirmedStake = 0n;
  let rejectedStake = 0n;

  for (const v of verifications) {
    totalStake += v.stake;
    if (v.verdict === 'confirmed') confirmedStake += v.stake;
    if (v.verdict === 'rejected') rejectedStake += v.stake;
  }

  const confirmRatio = Number(confirmedStake) / Number(totalStake);
  const rejectRatio = Number(rejectedStake) / Number(totalStake);

  if (confirmRatio >= params.verificationQuorum) {
    return { reached: true, consensus: 'confirmed' };
  }
  if (rejectRatio >= params.verificationQuorum) {
    return { reached: true, consensus: 'rejected' };
  }

  return { reached: false, consensus: 'none' };
}

/**
 * Calculate submission reward
 *
 * FORMULA (Open Source):
 *   reward = base_reward × novelty × utility × confidence
 *
 * WHERE:
 *   novelty = 1 / (1 + similar_facts_count)
 *   utility = query_potential (estimated)
 *   confidence = final verified confidence
 */
export function calculateSubmissionReward(
  baseReward: bigint,
  similarFactsCount: number,
  confidence: Confidence,
  utilityScore: number = 1.0
): bigint {
  const novelty = 1 / (1 + similarFactsCount);
  const multiplier = novelty * utilityScore * confidence;

  return BigInt(Math.floor(Number(baseReward) * multiplier));
}

/**
 * Calculate verification reward
 *
 * FORMULA (Open Source):
 *   reward = base_reward × (your_stake / total_stake) × correctness
 *
 * WHERE:
 *   correctness = 1.0 if matched consensus, 0.0 if not
 */
export function calculateVerificationReward(
  baseReward: bigint,
  validatorStake: bigint,
  totalStake: bigint,
  matchedConsensus: boolean
): bigint {
  if (!matchedConsensus) return 0n;

  const stakeWeight = Number(validatorStake) / Number(totalStake);
  return BigInt(Math.floor(Number(baseReward) * stakeWeight));
}

/**
 * Calculate challenge reward
 *
 * FORMULA (Open Source):
 *   reward = loser_stake × challenge_reward_share
 */
export function calculateChallengeReward(
  loserStake: bigint,
  params: TruthParameters = DEFAULT_TRUTH_PARAMETERS
): bigint {
  return BigInt(Math.floor(Number(loserStake) * params.challengeRewardShare));
}

/**
 * Calculate slashing amount
 */
export function calculateSlashing(
  stake: bigint,
  slashRate: number
): bigint {
  return BigInt(Math.floor(Number(stake) * slashRate));
}

/**
 * Select validators for verification task
 *
 * Stake-weighted random selection
 */
export function selectValidators(
  availableValidators: Array<{ did: DID; stake: bigint }>,
  count: number,
  seed: string
): DID[] {
  if (availableValidators.length <= count) {
    return availableValidators.map(v => v.did);
  }

  // Calculate total stake
  const totalStake = availableValidators.reduce((sum, v) => sum + v.stake, 0n);

  // Stake-weighted selection (simplified - production would use VRF)
  const selected: DID[] = [];
  const remaining = [...availableValidators];

  for (let i = 0; i < count && remaining.length > 0; i++) {
    // Simple deterministic selection based on seed
    const hash = simpleHash(seed + i.toString());
    const target = BigInt(hash) % totalStake;

    let cumulative = 0n;
    for (let j = 0; j < remaining.length; j++) {
      cumulative += remaining[j].stake;
      if (cumulative >= target) {
        selected.push(remaining[j].did);
        remaining.splice(j, 1);
        break;
      }
    }
  }

  return selected;
}

/**
 * Simple hash function for deterministic selection
 * (Production would use proper cryptographic hash)
 */
function simpleHash(input: string): number {
  let hash = 0;
  for (let i = 0; i < input.length; i++) {
    const char = input.charCodeAt(i);
    hash = ((hash << 5) - hash) + char;
    hash = hash & hash;
  }
  return Math.abs(hash);
}

/**
 * Normalize a claim for comparison and hashing
 */
export function normalizeClaim(claim: string): string {
  return claim
    .toLowerCase()
    .trim()
    .replace(/\s+/g, ' ')
    .replace(/[^\w\s]/g, '');
}

/**
 * Check if two claims are semantically similar
 * (Simplified - production would use embeddings)
 */
export function claimsSimilar(claim1: string, claim2: string): boolean {
  const norm1 = normalizeClaim(claim1);
  const norm2 = normalizeClaim(claim2);

  // Exact match after normalization
  if (norm1 === norm2) return true;

  // Check word overlap (simplified similarity)
  const words1 = new Set(norm1.split(' '));
  const words2 = new Set(norm2.split(' '));

  const intersection = new Set([...words1].filter(w => words2.has(w)));
  const union = new Set([...words1, ...words2]);

  const jaccard = intersection.size / union.size;
  return jaccard > 0.8;  // 80% word overlap
}

/**
 * Validate a derivation chain
 */
export function validateDerivation(
  derivation: Derivation,
  premises: Fact[]
): { valid: boolean; reason?: string } {
  // Check all premises exist and are verified
  for (const factId of derivation.fromFacts) {
    const premise = premises.find(p => p.id === factId);
    if (!premise) {
      return { valid: false, reason: `Premise ${factId} not found` };
    }
    if (premise.status !== 'verified') {
      return { valid: false, reason: `Premise ${factId} not verified` };
    }
    if (premise.confidence < 0.5) {
      return { valid: false, reason: `Premise ${factId} confidence too low` };
    }
  }

  // Check inference type is valid
  if (!INFERENCE_STRENGTH[derivation.inferenceType]) {
    return { valid: false, reason: `Invalid inference type` };
  }

  return { valid: true };
}

// ============ Fundamentality & Citation Rewards ============

/**
 * Calculate fundamentality score for a fact
 *
 * FORMULA (Open Source):
 *   fundamentality = citation_score × 0.30 +
 *                    depth_score × 0.25 +
 *                    cross_domain_score × 0.20 +
 *                    velocity_score × 0.15 +
 *                    longevity_score × 0.10
 */
export function calculateFundamentality(
  citationCount: number,
  maxDerivationDepth: number,
  citingDomains: string[],
  citationVelocity: number,  // Citations per 1000 blocks
  ageBlocks: number
): FundamentalityMetrics {
  // Citation count (logarithmic - diminishing returns)
  const citationScore = Math.min(1, Math.log10(citationCount + 1) / Math.log10(1000));

  // Derivation depth (how many levels build on this)
  const depthScore = Math.min(1, maxDerivationDepth / 10);

  // Cross-domain usage
  const uniqueDomains = new Set(citingDomains);
  const crossDomainScore = Math.min(1, uniqueDomains.size / 5);

  // Citation velocity (new citations rate)
  const velocityScore = Math.min(1, citationVelocity / 10);

  // Longevity (age in blocks, capped at 1M blocks ~= 1.5 years)
  const longevityScore = Math.min(1, ageBlocks / 1000000);

  // Weighted combination
  const fundamentalityScore =
    citationScore * 0.30 +
    depthScore * 0.25 +
    crossDomainScore * 0.20 +
    velocityScore * 0.15 +
    longevityScore * 0.10;

  // Cross-domain multiplier
  const crossDomainMultiplier = Math.min(2.0, 1.0 + (uniqueDomains.size - 1) * 0.2);

  return {
    citationCount,
    maxDerivationDepth,
    citingDomains: Array.from(uniqueDomains),
    citationVelocity,
    ageBlocks,
    fundamentalityScore,
    crossDomainMultiplier,
  };
}

/**
 * Get fundamentality bonus multiplier based on score
 */
export function getFundamentalityBonus(score: number): number {
  if (score >= FUNDAMENTALITY_THRESHOLDS.AXIOMATIC) return FUNDAMENTALITY_BONUSES.AXIOMATIC;
  if (score >= FUNDAMENTALITY_THRESHOLDS.FUNDAMENTAL) return FUNDAMENTALITY_BONUSES.FUNDAMENTAL;
  if (score >= FUNDAMENTALITY_THRESHOLDS.FOUNDATIONAL) return FUNDAMENTALITY_BONUSES.FOUNDATIONAL;
  if (score >= FUNDAMENTALITY_THRESHOLDS.NOTABLE) return FUNDAMENTALITY_BONUSES.NOTABLE;
  return 1.0;  // No bonus below NOTABLE threshold
}

/**
 * Distribute citation rewards up the derivation chain
 *
 * FORMULA (Open Source):
 *   Each ancestor gets: base_share × (1 - decay_rate)^depth
 *   Remainder goes to protocol treasury
 */
export function distributeCitationRewards(
  totalReward: bigint,
  ancestorChain: Array<{ factId: string; submitter: DID; fundamentalityScore: number }>,
  params: CitationParameters = DEFAULT_CITATION_PARAMETERS
): CitationReward[] {
  const rewards: CitationReward[] = [];

  // Calculate citation pool
  const citationPool = BigInt(Math.floor(Number(totalReward) * params.citationShare));

  if (citationPool === 0n || ancestorChain.length === 0) {
    return rewards;
  }

  let remaining = citationPool;

  // Distribute up the chain with decay
  for (let i = 0; i < ancestorChain.length; i++) {
    const ancestor = ancestorChain[i];
    const decayMultiplier = Math.pow(1 - params.decayRate, i);
    const baseShare = Number(citationPool) * decayMultiplier * (1 - params.decayRate);

    // Apply fundamentality bonus
    const bonus = getFundamentalityBonus(ancestor.fundamentalityScore);
    const shareWithBonus = baseShare * bonus;
    const amount = BigInt(Math.floor(shareWithBonus));

    if (amount > 0n && amount <= remaining) {
      rewards.push({
        factId: ancestor.factId,
        submitter: ancestor.submitter,
        share: shareWithBonus / Number(citationPool),
        amount,
        reason: i === 0 ? 'direct_citation' : 'ancestor',
      });
      remaining -= amount;
    }
  }

  // Protocol treasury gets remainder (minimum floor)
  const protocolMinimum = BigInt(Math.floor(Number(citationPool) * params.protocolFloor));
  if (remaining > 0n || remaining < protocolMinimum) {
    const protocolAmount = remaining > protocolMinimum ? remaining : protocolMinimum;
    rewards.push({
      factId: 'protocol_treasury',
      submitter: 'protocol' as DID,
      share: Number(protocolAmount) / Number(citationPool),
      amount: protocolAmount,
      reason: 'ancestor',
    });
  }

  return rewards;
}

/**
 * Calculate cross-domain bonus for a reward
 */
export function applyCrossDomainBonus(
  baseReward: bigint,
  citingDomains: string[],
  params: CitationParameters = DEFAULT_CITATION_PARAMETERS
): bigint {
  const uniqueDomains = new Set(citingDomains);
  const multiplier = Math.min(
    params.maxCrossDomainBonus,
    1.0 + (uniqueDomains.size - 1) * params.crossDomainBonusRate
  );

  return BigInt(Math.floor(Number(baseReward) * multiplier));
}

/**
 * Calculate maximum derivation depth from a fact
 */
export function calculateMaxDerivationDepth(
  factId: string,
  citedBy: Map<string, string[]>,
  visited: Set<string> = new Set()
): number {
  if (visited.has(factId)) return 0;  // Prevent cycles
  visited.add(factId);

  const children = citedBy.get(factId) || [];
  if (children.length === 0) return 0;

  let maxDepth = 0;
  for (const childId of children) {
    const childDepth = calculateMaxDerivationDepth(childId, citedBy, visited);
    maxDepth = Math.max(maxDepth, childDepth + 1);
  }

  return maxDepth;
}

/**
 * Create a new fact from verified submission
 */
export function createFact(
  submission: ClaimSubmission,
  verifications: Verification[],
  submitter: DID,
  currentBlock: number
): Fact {
  const confidence = calculateConfidence(verifications, submission.references as Reference[], 0);

  return {
    id: `fact_${currentBlock}_${simpleHash(submission.claim)}`,
    claim: submission.claim,
    claimHash: normalizeClaim(submission.claim),
    type: submission.type,
    domain: submission.domain,

    confidence,
    confidenceHistory: [{
      timestamp: currentBlock,
      confidence,
      reason: 'initial_verification',
    }],

    references: submission.references as Reference[],
    verifications,
    derivedFrom: submission.derivedFrom ? {
      fromFacts: submission.derivedFrom.factIds,
      inferenceType: submission.derivedFrom.inferenceType,
      inferenceStrength: INFERENCE_STRENGTH[submission.derivedFrom.inferenceType],
      logicChain: submission.derivedFrom.logicChain,
      derivedAt: currentBlock,
      derivedBy: submitter,
    } : undefined,

    submittedBy: submitter,
    submittedAt: currentBlock,
    lastVerified: currentBlock,

    status: 'verified',
  };
}
