/**
 * Arbitrator Selection System
 *
 * VRF-based selection of arbitrators for disputes with:
 * - Domain expertise weighting (experts get 1.0-2.0x multiplier)
 * - Stake weighting (sqrt to prevent plutocracy)
 * - Reputation weighting
 * - Conflict of interest exclusion
 * - Replacement prevention (no double-selection)
 *
 * PP HIERARCHY:
 *   TRUTH: Selection must be verifiably random and fair
 *   UNDERSTANDING: Expertise matters - domain experts weighted higher
 *   BEAUTY: Simple weights combine into fair selection
 *   JUSTICE: Sqrt stake prevents plutocracy, excludes conflicts
 */

import { blake3Hash } from "../crypto/hashing.js";
import type { ValidatorState } from "../../state/types.js";
import {
  EXPERTISE_DECAY,
  DISPUTE_ECONOMICS,
  type ArbiterSelectionConfig,
  type DomainExpertise,
  type SelectedArbiter,
} from "./types.js";

// ============ Weight Calculation ============

/**
 * Calculate selection weight for a validator
 *
 * Weight = baseReputation × stakeMultiplier × domainExpertise
 *
 * - stakeMultiplier uses sqrt to prevent plutocracy
 * - domainExpertise gives 1.0-2.0x multiplier
 */
export function calculateArbiterWeight(
  validator: ValidatorState,
  config: ArbiterSelectionConfig,
  domainExpertise: DomainExpertise | null,
  currentBlock: bigint,
): number {
  let weight = 1.0;

  // Base reputation (0-1)
  weight *= validator.reputation;

  // Stake weight (sqrt to prevent plutocracy)
  // P1-D4 FIX: Was Number(validator.stake) / Number(minStake) — both exceed
  // MAX_SAFE_INTEGER (77 LGM × 10^18 ≈ 7.7×10^19 >> 2^53 ≈ 9×10^15).
  // Now uses BigInt division with 1000x scaling to preserve 3 decimal places.
  if (config.stakeWeighting) {
    const minStake = DISPUTE_ECONOMICS.minArbiterStake;
    const stakeRatio = Number((validator.stake * 1000n) / minStake) / 1000;
    // sqrt caps influence of large stakes
    weight *= Math.sqrt(Math.max(1, stakeRatio));
  }

  // Domain expertise multiplier (1.0-2.0x)
  if (config.domainWeighting && domainExpertise) {
    const expertiseMultiplier = calculateExpertiseMultiplier(
      domainExpertise,
      currentBlock,
    );
    weight *= expertiseMultiplier;
  }

  return weight;
}

/**
 * Calculate expertise multiplier with decay
 */
export function calculateExpertiseMultiplier(
  expertise: DomainExpertise,
  currentBlock: bigint,
): number {
  // Must have minimum votes
  if (expertise.totalVotes < EXPERTISE_DECAY.minVotesForExpertise) {
    return 1.0; // No bonus
  }

  // Must have minimum accuracy
  if (expertise.accuracy < EXPERTISE_DECAY.minAccuracyForExpertise) {
    return 1.0; // No bonus
  }

  // Calculate decay based on last activity
  const blocksSinceActive = Number(currentBlock - expertise.lastActiveBlock);
  const decayFactor = Math.exp(
    (-Math.LN2 * blocksSinceActive) / Number(EXPERTISE_DECAY.halfLifeBlocks),
  );

  // Expertise score (0-1) based on accuracy above minimum
  const expertiseScore =
    (expertise.accuracy - EXPERTISE_DECAY.minAccuracyForExpertise) /
    (1.0 - EXPERTISE_DECAY.minAccuracyForExpertise);

  // Multiplier: 1.0 (no expertise) to 2.0 (perfect expertise)
  const rawMultiplier = 1.0 + expertiseScore;

  // Apply decay
  return 1.0 + (rawMultiplier - 1.0) * decayFactor;
}

// ============ Candidate Filtering ============

/**
 * Filter validators eligible to be arbiters
 */
export function filterEligibleArbiters(
  validators: ValidatorState[],
  config: ArbiterSelectionConfig,
): ValidatorState[] {
  return validators.filter((v) => {
    // Must be active
    if (!v.isActive) {
      return false;
    }

    // Must have minimum stake
    if (v.stake < DISPUTE_ECONOMICS.minArbiterStake) {
      return false;
    }

    // Must meet minimum reputation
    if (v.reputation < config.minReputation) {
      return false;
    }

    // Must not be excluded (conflict of interest)
    if (config.excludeValidators.has(v.did)) {
      return false;
    }

    return true;
  });
}

// ============ VRF-Based Selection ============

/**
 * Generate deterministic random number from VRF seed and index
 */
/**
 * P0-D3 FIX: Was using 8 bytes (64 bits) in JS Number which exceeds
 * MAX_SAFE_INTEGER (2^53), causing precision loss and potentially
 * non-deterministic results across different JS engines.
 * Now uses 6 bytes (48 bits) which fits safely in Number.
 */
function vrfRandom(seed: Uint8Array, index: number): number {
  // Combine seed with index for unique random per selection round
  const combined = new Uint8Array(seed.length + 4);
  combined.set(seed);
  combined[seed.length] = (index >> 24) & 0xff;
  combined[seed.length + 1] = (index >> 16) & 0xff;
  combined[seed.length + 2] = (index >> 8) & 0xff;
  combined[seed.length + 3] = index & 0xff;

  // Hash to get random bytes
  const hash = blake3Hash(combined);

  // Convert first 6 bytes (48 bits) to number (0-1)
  // 48 bits fits safely in JS Number (max safe = 2^53)
  let value = 0;
  for (let i = 0; i < 6; i++) {
    value = value * 256 + hash[i];
  }

  return value / 256 ** 6;
}

/**
 * Weighted random selection using VRF
 *
 * Uses weighted reservoir sampling for fair selection
 * without replacement.
 */
export function vrfWeightedSelection(
  candidates: Array<{ validator: ValidatorState; weight: number }>,
  count: number,
  seed: Uint8Array,
): ValidatorState[] {
  if (candidates.length === 0) {
    return [];
  }

  if (candidates.length <= count) {
    // Not enough candidates, return all
    return candidates.map((c) => c.validator);
  }

  // Calculate total weight
  const totalWeight = candidates.reduce((sum, c) => sum + c.weight, 0);

  // Normalize weights to probabilities
  const probabilities = candidates.map((c) => ({
    validator: c.validator,
    probability: c.weight / totalWeight,
  }));

  // Selection using modified weighted reservoir sampling
  const selected: ValidatorState[] = [];
  const selectedIndices = new Set<number>();

  for (
    let round = 0;
    round < count && selectedIndices.size < candidates.length;
    round++
  ) {
    // Generate random value for this round
    const random = vrfRandom(seed, round);

    // Adjust probabilities excluding already selected
    let cumulativeProbability = 0;
    let adjustedTotal = 0;

    for (let i = 0; i < probabilities.length; i++) {
      if (!selectedIndices.has(i)) {
        adjustedTotal += probabilities[i].probability;
      }
    }

    // Select based on cumulative probability
    const threshold = random * adjustedTotal;
    cumulativeProbability = 0;

    for (let i = 0; i < probabilities.length; i++) {
      if (selectedIndices.has(i)) {
        continue;
      }

      cumulativeProbability += probabilities[i].probability;

      if (cumulativeProbability >= threshold) {
        selected.push(probabilities[i].validator);
        selectedIndices.add(i);
        break;
      }
    }
  }

  return selected;
}

// ============ Main Selection Function ============

/**
 * Select arbiters for a dispute
 *
 * @param validators All validators in the network
 * @param config Selection configuration
 * @param expertiseMap Map of DID -> domain expertise
 * @param currentBlock Current block number
 * @returns Selected arbiters with weights
 */
export function selectArbiters(
  validators: ValidatorState[],
  config: ArbiterSelectionConfig,
  expertiseMap: Map<string, DomainExpertise[]>,
  currentBlock: bigint,
): SelectedArbiter[] {
  // Filter eligible candidates
  const eligible = filterEligibleArbiters(validators, config);

  if (eligible.length < config.targetCount) {
    // Not enough eligible arbiters - return all available
    console.warn(
      `Only ${eligible.length} eligible arbiters for ${config.targetCount} requested`,
    );
  }

  // Calculate weights for each candidate
  const weighted = eligible.map((validator) => {
    // Get domain expertise if applicable
    let domainExpertise: DomainExpertise | null = null;
    if (config.domain && config.domainWeighting) {
      const allExpertise = expertiseMap.get(validator.did) || [];
      domainExpertise =
        allExpertise.find((e) => e.domain === config.domain) || null;
    }

    const weight = calculateArbiterWeight(
      validator,
      config,
      domainExpertise,
      currentBlock,
    );

    return { validator, weight, domainExpertise };
  });

  // Select using VRF
  const selectedValidators = vrfWeightedSelection(
    weighted,
    config.targetCount,
    config.vrfSeed,
  );

  // Build result with full information
  return selectedValidators.map((validator) => {
    const info = weighted.find((w) => w.validator.did === validator.did)!;

    return {
      publicKey: validator.publicKey,
      did: validator.did,
      selectionWeight: info.weight,
      domainExpertise: info.domainExpertise?.accuracy || 0,
      stakedAmount: validator.stake,
    };
  });
}

// ============ Seed Generation ============

/**
 * Generate VRF seed for dispute arbiter selection
 *
 * Combines block hash and dispute ID for unique, unpredictable seed
 */
export function generateDisputeSeed(
  blockHash: string,
  disputeId: string,
): Uint8Array {
  const combined = `${blockHash}:${disputeId}`;
  return blake3Hash(new TextEncoder().encode(combined));
}

// ============ Verification ============

/**
 * Verify arbiter selection was done correctly
 * (For dispute auditing/challenges)
 */
export function verifyArbiterSelection(
  validators: ValidatorState[],
  config: ArbiterSelectionConfig,
  expertiseMap: Map<string, DomainExpertise[]>,
  currentBlock: bigint,
  claimedSelection: SelectedArbiter[],
): { valid: boolean; errors: string[] } {
  const errors: string[] = [];

  // Recompute selection
  const expected = selectArbiters(
    validators,
    config,
    expertiseMap,
    currentBlock,
  );

  // Check count matches
  if (expected.length !== claimedSelection.length) {
    errors.push(
      `Count mismatch: expected ${expected.length}, got ${claimedSelection.length}`,
    );
  }

  // Check each arbiter
  const expectedDids = new Set(expected.map((a) => a.did));
  const claimedDids = new Set(claimedSelection.map((a) => a.did));

  for (const did of expectedDids) {
    if (!claimedDids.has(did)) {
      errors.push(`Missing expected arbiter: ${did}`);
    }
  }

  for (const did of claimedDids) {
    if (!expectedDids.has(did)) {
      errors.push(`Unexpected arbiter included: ${did}`);
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

// ============ Expertise Management ============

/**
 * Update domain expertise after dispute resolution
 */
export function updateExpertise(
  existing: DomainExpertise | null,
  domain: string,
  wasCorrect: boolean,
  currentBlock: bigint,
): DomainExpertise {
  if (!existing) {
    return {
      domain,
      correctVotes: wasCorrect ? 1 : 0,
      totalVotes: 1,
      accuracy: wasCorrect ? 1.0 : 0.0,
      lastActiveBlock: currentBlock,
    };
  }

  const newCorrect = existing.correctVotes + (wasCorrect ? 1 : 0);
  const newTotal = existing.totalVotes + 1;

  return {
    domain,
    correctVotes: newCorrect,
    totalVotes: newTotal,
    accuracy: newCorrect / newTotal,
    lastActiveBlock: currentBlock,
  };
}

/**
 * Get all domains where a validator has expertise
 */
export function getExpertiseDomains(
  expertiseMap: Map<string, DomainExpertise[]>,
  validatorDid: string,
): string[] {
  const expertise = expertiseMap.get(validatorDid) || [];
  return expertise
    .filter(
      (e) =>
        e.totalVotes >= EXPERTISE_DECAY.minVotesForExpertise &&
        e.accuracy >= EXPERTISE_DECAY.minAccuracyForExpertise,
    )
    .map((e) => e.domain);
}

/**
 * Check if validator is expert in domain
 */
export function isExpertInDomain(
  expertiseMap: Map<string, DomainExpertise[]>,
  validatorDid: string,
  domain: string,
): boolean {
  const expertise = expertiseMap.get(validatorDid) || [];
  const domainExpertise = expertise.find((e) => e.domain === domain);

  if (!domainExpertise) {
    return false;
  }

  return (
    domainExpertise.totalVotes >= EXPERTISE_DECAY.minVotesForExpertise &&
    domainExpertise.accuracy >= EXPERTISE_DECAY.minAccuracyForExpertise
  );
}
