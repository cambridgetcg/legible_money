/**
 * Adversarial Verification — Survive-to-Accept PoT Engine
 *
 * TypeScript prototype of the adversarial verification system.
 * Claims get provisional acceptance via structural evaluation,
 * then enter a challenge window where agents can stake counter-evidence.
 * Truth emerges from adversarial exposure, not unanimous agreement.
 *
 * Go production implementation: x/knowledge/keeper/{provisional,challenge,growth}.go
 */

import type { StateManager } from "../../state/state_manager.js";
import type { ClaimState } from "../../state/types.js";
import type { ConsensusParams } from "./types.js";
import type { EpistemicCategory } from "../../core/knowledge/types.js";

// ============ Constants ============

/** Blocks per confidence growth epoch (~47 min at 2521ms) */
export const CONFIDENCE_GROWTH_EPOCH = 1111;

/** Default score threshold below which claims are rejected */
export const REJECT_THRESHOLD = 300_000;

/** Default score threshold for provisional fact creation */
export const PROVISIONAL_THRESHOLD = 500_000;

/** Base challenge window duration (~1 day) */
export const BASE_CHALLENGE_DURATION = 34_272;

/** Max concurrent challenges per claim */
export const MAX_CONCURRENT_CHALLENGES = 3;

/** Challenge window multipliers by category profile */
export const CHALLENGE_WINDOW_PROFILE: Record<string, number> = {
  analytic: 3, // 3 days — eternal truths
  formal: 3,
  protocol: 3,
  computational: 3,
  historical: 2, // 2 days — slow decay
  replicated: 2,
  empirical: 1, // 1 day — fast decay
  social: 1,
  predictive: 1,
};

/** Categories resolved deterministically (Path A) */
const DETERMINISTIC_CATEGORIES = new Set<string>([
  "analytic",
  "formal",
  "protocol",
  "computational",
]);

/** Confidence caps */
export const MAX_SURVIVAL_CONFIDENCE = 770_000; // 77%
export const SURVIVED_CHALLENGE_CONFIDENCE_CAP = 880_000; // 88%

/** Default growth per epoch */
export const CONFIDENCE_GROWTH_PER_EPOCH = 11_000; // 1.1%

/** Challenge cooldown (exponential) */
export const BASE_COOLDOWN_BLOCKS = 17_136; // ~12 hours
export const MAX_COOLDOWN_BLOCKS = 171_360; // ~5 days

// ============ Maturity Constants ============

/** Emerging → Established */
export const MATURITY_ESTABLISHED_MIN_CONF = 800_000; // 80%
export const MATURITY_ESTABLISHED_MIN_AGE = 34_272n; // ~1 day
export const MATURITY_ESTABLISHED_MIN_VERIFICATIONS = 3;

/** Established → Canonical */
export const MATURITY_CANONICAL_MIN_CONF = 900_000; // 90%
export const MATURITY_CANONICAL_MIN_AGE = 342_720n; // ~10 days
export const MATURITY_CANONICAL_MIN_VERIFICATIONS = 10;

// ============ Structural Evaluation ============

/**
 * Evaluate a claim structurally and transition to the appropriate status.
 * This is the TS equivalent of x/knowledge/keeper/provisional.go:EvaluateAndProvisionClaim.
 *
 * @returns The structural score (0-1_000_000)
 */
export function evaluateAndProvisionClaim(
  state: StateManager,
  claim: ClaimState,
  blockNumber: bigint,
  params: ConsensusParams,
): number {
  // Compute structural score from category and references
  const score = computeStructuralScore(state, claim);

  const provisionalThreshold =
    params.provisionalThreshold ?? PROVISIONAL_THRESHOLD / 1_000_000;
  const rejectThreshold = REJECT_THRESHOLD / 1_000_000;

  // Normalize to 0-1 range for comparison
  const normalizedScore = score / 1_000_000;

  if (normalizedScore < rejectThreshold) {
    // REJECTED — structural score too low
    state.updateClaimStatus(claim.id, "rejected", blockNumber, {
      rejectionReason: `Structural score ${score} below threshold ${REJECT_THRESHOLD}`,
    });
    // Slash 22% of claim stake: unstake everything, effectively burning the slashed portion
    const slashAmount = (claim.stake * 22n) / 100n;
    const returnAmount = claim.stake - slashAmount;
    state.unstake(claim.submitter, returnAmount);
    return score;
  }

  // Calculate challenge window
  const windowDuration = calculateChallengeWindow(
    claim.claimType,
    claim.logicZone,
    score,
    params,
  );

  if (normalizedScore >= provisionalThreshold) {
    // PROVISIONAL — create provisional fact
    const factId = `provisional:${claim.id}:${blockNumber}`;
    state.addFact({
      id: factId,
      claimText: claim.claim,
      domain: claim.domain,
      confidence: normalizedScore,
      verifiedAt: 0n, // not yet verified
      verifierCount: 0,
      challengeCount: 0,
      citationCount: 0,
      status: "provisional",
      epistemicCategory: claim.claimType as EpistemicCategory,
      logicZone: claim.logicZone,
      isProvisional: true,
      challengeWindowEnd: blockNumber + windowDuration,
      originalClaimId: claim.id,
      survivedChallenges: 0,
    });

    state.updateClaimStatus(claim.id, "provisional", blockNumber);
  } else {
    // EVALUATED — in challenge window but no provisional fact
    state.updateClaimStatus(claim.id, "evaluated", blockNumber);
  }

  return score;
}

/**
 * Compute structural score for a claim based on category base score,
 * reference count, and evidence quality.
 *
 * Simplified version of the 6-stage Go evaluation pipeline.
 */
function computeStructuralScore(
  state: StateManager,
  claim: ClaimState,
): number {
  // Stage 1: Category base score
  const categoryScores: Record<string, number> = {
    protocol: 700_000,
    computational: 650_000,
    formal: 600_000,
    analytic: 600_000,
    empirical: 400_000,
    historical: 350_000,
    replicated: 500_000,
    social: 300_000,
    predictive: 250_000,
  };
  let score = categoryScores[claim.claimType] ?? 400_000;

  // Stage 2: Reference quality bonus
  for (const refId of claim.references) {
    const refFact = state.getFact(refId);
    if (
      refFact &&
      (refFact.status === "active" || refFact.status === "verified")
    ) {
      // Each supporting reference adds up to 30k based on its confidence
      score += Math.floor(refFact.confidence * 30_000);
    }
  }

  // Stage 3: Claim text length penalty (very short claims are suspect)
  if (claim.claim.length < 20) {
    score = Math.floor(score * 0.7);
  }

  // Stage 4: Zone ceiling
  if (claim.logicZone) {
    const zoneCeilings: Record<string, number> = {
      propositional: 1_000_000,
      presburger: 950_000,
      peano: 850_000,
      empirical: 990_000,
    };
    const ceiling = zoneCeilings[claim.logicZone] ?? 1_000_000;
    if (score > ceiling) score = ceiling;
  }

  // Cap at 1M
  if (score > 1_000_000) score = 1_000_000;

  return score;
}

// ============ Challenge Window ============

/**
 * Calculate challenge window duration based on epistemic category.
 */
export function calculateChallengeWindow(
  category: string | undefined,
  logicZone: string | undefined,
  score: number,
  params: ConsensusParams,
): bigint {
  const baseDuration = BigInt(
    params.baseChallengeDurationBlocks ?? BASE_CHALLENGE_DURATION,
  );
  const multiplier = BigInt(
    CHALLENGE_WINDOW_PROFILE[category ?? "empirical"] ?? 1,
  );

  let duration = baseDuration * multiplier;

  // Propositional tautologies (score at zone ceiling): halved window
  if (logicZone === "propositional" && score >= 950_000) {
    duration /= 2n;
  }

  // Protocol axioms: quarter window
  if (category === "protocol" && logicZone === "propositional") {
    duration /= 4n;
  }

  return duration;
}

/**
 * Check if a category uses deterministic resolution (Path A).
 */
export function isDeterministicCategory(category: string): boolean {
  return DETERMINISTIC_CATEGORIES.has(category);
}

// ============ Challenge Window Expiry ============

/**
 * Process claims whose challenge window has closed.
 * Transitions them to accepted and creates/upgrades facts.
 */
export function processChallengeWindowExpiry(
  state: StateManager,
  blockNumber: bigint,
): void {
  // Get claims in provisional and evaluated status
  const provisionalClaims = state.getClaimsByStatus("provisional");
  const evaluatedClaims = state.getClaimsByStatus("evaluated");
  const allClaims = [...provisionalClaims, ...evaluatedClaims];

  for (const claim of allClaims) {
    if (!claim.challengeWindowEnd || blockNumber < claim.challengeWindowEnd) {
      continue;
    }
    if (claim.activeChallengeCount && claim.activeChallengeCount > 0) {
      continue;
    }

    transitionToVerified(state, claim, blockNumber);
  }
}

/**
 * Transition a claim from provisional/evaluated to accepted after
 * its challenge window closes without successful challenges.
 */
function transitionToVerified(
  state: StateManager,
  claim: ClaimState,
  blockNumber: bigint,
): void {
  if (claim.status === "provisional" && claim.provisionalFactId) {
    // Upgrade provisional fact to active
    const fact = state.getFact(claim.provisionalFactId);
    if (fact) {
      state.updateFactStatus(claim.provisionalFactId, "active");
    }
  } else if (claim.status === "evaluated") {
    // Create new verified fact from the evaluated claim
    const factId = `verified:${claim.id}:${blockNumber}`;
    state.addFact({
      id: factId,
      claimText: claim.claim,
      domain: claim.domain,
      confidence: (claim.structuralScore ?? 400_000) / 1_000_000,
      verifiedAt: blockNumber,
      verifierCount: 0,
      challengeCount: 0,
      citationCount: 0,
      status: "active",
      epistemicCategory: claim.claimType as EpistemicCategory,
      logicZone: claim.logicZone,
      originalClaimId: claim.id,
    });
  }

  // Return stake to submitter
  state.unstake(claim.submitter, claim.stake);

  state.updateClaimStatus(claim.id, "accepted", blockNumber);
}

// ============ Confidence Growth ============

/**
 * Apply survival-based confidence growth to verified facts.
 * Called every CONFIDENCE_GROWTH_EPOCH blocks.
 *
 * Growth formula:
 *   effective_growth = growth_per_epoch * survival_multiplier
 *   survival_multiplier = min(3.0, 1.0 + survived_challenges * 0.5)
 */
export function applyConfidenceGrowth(
  state: StateManager,
  params: ConsensusParams,
): void {
  const growthPerEpoch =
    params.confidenceGrowthPerEpoch ?? CONFIDENCE_GROWTH_PER_EPOCH / 1_000_000;
  const maxSurvival =
    params.maxSurvivalConfidence ?? MAX_SURVIVAL_CONFIDENCE / 1_000_000;

  // Get all active and verified facts
  const activeFacts = state.getActiveFacts();

  for (const fact of activeFacts) {
    if (fact.isProvisional) continue;

    const survived = fact.survivedChallenges ?? 0;
    const survivalMultiplier = Math.min(3.0, 1.0 + survived * 0.5);
    const effectiveGrowth = growthPerEpoch * survivalMultiplier;

    // Determine cap
    let cap = maxSurvival;
    if (survived > 0) {
      cap = SURVIVED_CHALLENGE_CONFIDENCE_CAP / 1_000_000; // 0.88
    }

    // Category ceiling always applies
    const categoryMaxConf = getCategoryCeiling(fact.epistemicCategory);
    if (categoryMaxConf < cap) {
      cap = categoryMaxConf;
    }

    const newConfidence = Math.min(fact.confidence + effectiveGrowth, cap);
    if (newConfidence !== fact.confidence) {
      state.updateFactConfidence(fact.id, newConfidence);
    }
  }
}

/**
 * Get the maximum confidence for an epistemic category.
 */
function getCategoryCeiling(category?: string): number {
  const ceilings: Record<string, number> = {
    analytic: 1.0,
    formal: 1.0,
    protocol: 1.0,
    computational: 1.0,
    empirical: 0.99,
    historical: 0.95,
    replicated: 0.995,
    social: 0.85,
    predictive: 0.9,
  };
  return ceilings[category ?? "empirical"] ?? 1.0;
}

// ============ Maturity Advancement ============

/**
 * Check facts for maturity advancement.
 * Called every CONFIDENCE_GROWTH_EPOCH blocks.
 *
 * Note: FactState in the TS prototype doesn't have a maturity field.
 * Maturity is tracked in the Go production chain (x/knowledge/keeper/growth.go).
 * This function is a no-op placeholder for future extension.
 */
export function advanceMaturityLevels(
  _state: StateManager,
  _blockNumber: bigint,
): void {
  // Maturity tracking is handled in the Go production chain.
  // The TS prototype uses confidence thresholds directly.
}

// ============ BeginBlocker Orchestrator ============

/**
 * Adversarial verification BeginBlocker.
 * When adversarialVerificationEnabled is true, this replaces the legacy flow.
 */
export function beginBlockerAdversarial(
  state: StateManager,
  blockNumber: bigint,
  params: ConsensusParams,
): void {
  if (!params.adversarialVerificationEnabled) {
    return; // Legacy mode — handled by existing engine
  }

  // 1. Evaluate pending claims
  const pendingClaims = state.getClaimsByStatus("pending_evaluation");
  for (const claim of pendingClaims) {
    evaluateAndProvisionClaim(state, claim, blockNumber, params);
  }

  // 2. Process challenge window expiry → transition to verified
  processChallengeWindowExpiry(state, blockNumber);

  // 3. At epoch boundaries: confidence growth + maturity advancement
  if (blockNumber % BigInt(CONFIDENCE_GROWTH_EPOCH) === 0n) {
    applyConfidenceGrowth(state, params);
    advanceMaturityLevels(state, blockNumber);
  }
}
