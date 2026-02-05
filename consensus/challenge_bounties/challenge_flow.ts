/**
 * Challenge Bounty Flow
 *
 * Lifecycle management for domain capture challenges:
 * submit → evidence_period → under_review → resolution
 */

import type {
  OntologicalStratum,
  DomainTree,
} from "../../core/knowledge/ontology/types.js";
import {
  type CaptureChallenge,
  type CaptureEvidence,
  type CaptureResolution,
  type DomainBountyPool,
  type DomainRecoveryAction,
  type ChallengeValidation,
  CHALLENGE_BOUNTY_CONFIG,
  generateChallengeId,
  createEmptyBountyPool,
} from "./types.js";

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Get domain stratum from domain tree
 */
function getDomainStratum(
  domain: string,
  domainTree: DomainTree,
): OntologicalStratum {
  const node = domainTree.domainIndex.get(domain);
  return node?.domain.stratum ?? ("empirical" as unknown as OntologicalStratum);
}

// ============================================================================
// Challenge Submission
// ============================================================================

/**
 * Validate a capture challenge submission
 */
export function validateChallengeSubmission(
  _challenger: string,
  domain: string,
  evidence: CaptureEvidence[],
  stake: bigint,
  domainTree: DomainTree,
): ChallengeValidation {
  const errors: string[] = [];

  // Validate stake
  if (stake < CHALLENGE_BOUNTY_CONFIG.minChallengeStake) {
    errors.push(
      `Insufficient stake: ${stake} < ${CHALLENGE_BOUNTY_CONFIG.minChallengeStake}`,
    );
  }

  // Validate evidence count
  if (evidence.length < CHALLENGE_BOUNTY_CONFIG.minEvidencePieces) {
    errors.push(
      `Insufficient evidence: ${evidence.length} < ${CHALLENGE_BOUNTY_CONFIG.minEvidencePieces}`,
    );
  }

  // Validate domain exists
  if (!domainTree.domainIndex.has(domain)) {
    errors.push(`Domain "${domain}" does not exist`);
  }

  // Validate each evidence piece
  for (let i = 0; i < evidence.length; i++) {
    const e = evidence[i];
    if (!e.type || !e.description || !e.dataHash) {
      errors.push(`Evidence piece ${i + 1} is incomplete`);
    }
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

/**
 * Submit a capture challenge
 */
export function submitCaptureChallenge(
  challenger: string,
  domain: string,
  evidence: CaptureEvidence[],
  stake: bigint,
  currentBlock: bigint,
  domainTree: DomainTree,
): { challenge: CaptureChallenge | null; validation: ChallengeValidation } {
  // Validate submission
  const validation = validateChallengeSubmission(
    challenger,
    domain,
    evidence,
    stake,
    domainTree,
  );

  if (!validation.valid) {
    return { challenge: null, validation };
  }

  // Extract accused validators from evidence
  const accusedValidators = new Set<string>();
  for (const e of evidence) {
    for (const v of e.validatorIds || []) {
      accusedValidators.add(v);
    }
  }

  const challenge: CaptureChallenge = {
    id: generateChallengeId(challenger, domain, currentBlock),
    domain,
    stratum: getDomainStratum(domain, domainTree),
    challenger,
    challengeStake: stake,
    evidence,
    minEvidenceRequired: CHALLENGE_BOUNTY_CONFIG.minEvidencePieces,
    accusedValidators: Array.from(accusedValidators),
    submittedAt: currentBlock,
    evidenceDeadline:
      currentBlock + CHALLENGE_BOUNTY_CONFIG.evidencePeriodBlocks,
    reviewDeadline:
      currentBlock +
      CHALLENGE_BOUNTY_CONFIG.evidencePeriodBlocks +
      CHALLENGE_BOUNTY_CONFIG.reviewPeriodBlocks,
    status: "submitted",
  };

  return { challenge, validation };
}

// ============================================================================
// Challenge Progression
// ============================================================================

/**
 * Add additional evidence to a challenge
 */
export function addEvidence(
  challenge: CaptureChallenge,
  evidence: CaptureEvidence,
  currentBlock: bigint,
): CaptureChallenge {
  // Can only add evidence during evidence period
  if (
    challenge.status !== "submitted" &&
    challenge.status !== "evidence_period"
  ) {
    return challenge;
  }

  // Check deadline
  if (currentBlock > challenge.evidenceDeadline) {
    return challenge;
  }

  // Add new validators to accused list
  const accusedValidators = new Set(challenge.accusedValidators);
  for (const v of evidence.validatorIds || []) {
    accusedValidators.add(v);
  }

  return {
    ...challenge,
    evidence: [...challenge.evidence, evidence],
    accusedValidators: Array.from(accusedValidators),
    status: "evidence_period",
  };
}

/**
 * Advance challenge to review phase
 */
export function advanceToReview(
  challenge: CaptureChallenge,
  currentBlock: bigint,
): CaptureChallenge {
  // Must be past evidence deadline
  if (currentBlock <= challenge.evidenceDeadline) {
    return challenge;
  }

  // Must have minimum evidence
  if (challenge.evidence.length < challenge.minEvidenceRequired) {
    return {
      ...challenge,
      status: "expired",
    };
  }

  return {
    ...challenge,
    status: "under_review",
  };
}

/**
 * Check for challenge expiration
 */
export function checkChallengeExpiration(
  challenge: CaptureChallenge,
  currentBlock: bigint,
): CaptureChallenge {
  if (challenge.status === "upheld" || challenge.status === "rejected") {
    return challenge;
  }

  if (currentBlock > challenge.reviewDeadline) {
    return {
      ...challenge,
      status: "expired",
    };
  }

  return challenge;
}

// ============================================================================
// Challenge Resolution
// ============================================================================

/**
 * Calculate challenger reward for upheld challenge
 */
export function calculateChallengerReward(
  challenge: CaptureChallenge,
  bountyPool: DomainBountyPool,
): bigint {
  // Base: stake * multiplier
  const stakeReturn = BigInt(
    Math.floor(
      Number(challenge.challengeStake) *
        CHALLENGE_BOUNTY_CONFIG.challengeStakeMultiplierOnSuccess,
    ),
  );

  // Capture exposure bonus
  const exposureBonus = CHALLENGE_BOUNTY_CONFIG.captureExposureBonus;

  // Share of bounty pool (proportional to evidence quality)
  const evidenceQuality =
    challenge.evidence.length / Math.max(challenge.evidence.length, 10);
  const poolShare = BigInt(
    Math.floor(Number(bountyPool.balance) * evidenceQuality * 0.5),
  );

  return stakeReturn + exposureBonus + poolShare;
}

/**
 * Calculate validator slashes for upheld challenge
 */
export function calculateValidatorSlashes(
  challenge: CaptureChallenge,
  validatorStakes: Map<string, bigint>,
): Map<string, bigint> {
  const slashes = new Map<string, bigint>();

  for (const validatorId of challenge.accusedValidators) {
    const stake = validatorStakes.get(validatorId);
    if (!stake) continue;

    const slashAmount = BigInt(
      Math.floor(
        Number(stake) * CHALLENGE_BOUNTY_CONFIG.validatorSlashOnUpheldChallenge,
      ),
    );
    slashes.set(validatorId, slashAmount);
  }

  return slashes;
}

/**
 * Calculate failed challenge penalty
 */
export function calculateFailedChallengePenalty(
  challenge: CaptureChallenge,
): bigint {
  return BigInt(
    Math.floor(
      Number(challenge.challengeStake) *
        CHALLENGE_BOUNTY_CONFIG.failedChallengePenalty,
    ),
  );
}

/**
 * Resolve capture challenge
 */
export function resolveCaptureChallenge(
  challenge: CaptureChallenge,
  outcome: "upheld" | "rejected" | "partial",
  bountyPool: DomainBountyPool,
  validatorStakes: Map<string, bigint>,
  currentBlock: bigint,
  justification: string,
): {
  resolution: CaptureResolution;
  stakeReturns: Map<string, bigint>;
  updatedBountyPool: DomainBountyPool;
} {
  const stakeReturns = new Map<string, bigint>();
  let updatedBountyPool = { ...bountyPool };

  if (outcome === "upheld" || outcome === "partial") {
    // Challenger wins
    const challengerReward = calculateChallengerReward(challenge, bountyPool);
    const validatorSlashes = calculateValidatorSlashes(
      challenge,
      validatorStakes,
    );

    // Determine recovery actions
    const recoveryActions: DomainRecoveryAction[] = [];

    if (outcome === "upheld") {
      // Full capture confirmed - revoke all accused qualifications
      recoveryActions.push({
        type: "revoke_qualifications",
        validators: challenge.accusedValidators,
      });

      // Pause domain for new claims
      if (CHALLENGE_BOUNTY_CONFIG.domainPauseOnUpheldChallenge) {
        recoveryActions.push({
          type: "pause_new_claims",
          durationBlocks: CHALLENGE_BOUNTY_CONFIG.domainPauseDurationBlocks,
        });
      }
    }

    // Extract fact IDs from evidence
    const factsToRequeue: string[] = [];
    for (const e of challenge.evidence) {
      for (const factId of e.factIds || []) {
        factsToRequeue.push(factId);
      }
    }

    const resolution: CaptureResolution = {
      outcome,
      challengerReward,
      validatorSlashes,
      factsInvalidated: [],
      factsRequeuedForVerification: factsToRequeue,
      domainRecoveryActions: recoveryActions,
      resolvedAt: currentBlock,
      resolvedBy: "governance",
      justification,
    };

    stakeReturns.set(challenge.challenger, challengerReward);

    // Update bounty pool
    updatedBountyPool = {
      ...bountyPool,
      challengesUpheld: bountyPool.challengesUpheld + 1,
      totalRewardsPaid: bountyPool.totalRewardsPaid + challengerReward,
      balance:
        bountyPool.balance > challengerReward
          ? bountyPool.balance - challengerReward
          : 0n,
      lastUpdatedAt: currentBlock,
    };

    return { resolution, stakeReturns, updatedBountyPool };
  } else {
    // Challenge rejected - challenger loses stake
    const penalty = calculateFailedChallengePenalty(challenge);

    const resolution: CaptureResolution = {
      outcome: "rejected",
      challengerReward: 0n,
      validatorSlashes: new Map(),
      factsInvalidated: [],
      factsRequeuedForVerification: [],
      domainRecoveryActions: [],
      resolvedAt: currentBlock,
      resolvedBy: "governance",
      justification,
    };

    // Penalty goes to bounty pool
    updatedBountyPool = {
      ...bountyPool,
      challengesRejected: bountyPool.challengesRejected + 1,
      balance: bountyPool.balance + penalty,
      lastUpdatedAt: currentBlock,
    };

    // Return remaining stake to challenger
    const returnAmount = challenge.challengeStake - penalty;
    if (returnAmount > 0n) {
      stakeReturns.set(challenge.challenger, returnAmount);
    }

    return { resolution, stakeReturns, updatedBountyPool };
  }
}

/**
 * Finalize challenge with resolution
 */
export function finalizeChallenge(
  challenge: CaptureChallenge,
  resolution: CaptureResolution,
): CaptureChallenge {
  return {
    ...challenge,
    status: resolution.outcome === "rejected" ? "rejected" : "upheld",
    resolution,
  };
}

// ============================================================================
// Bounty Pool Management
// ============================================================================

/**
 * Create or get bounty pool for domain
 */
export function getOrCreateBountyPool(
  domain: string,
  existingPools: Map<string, DomainBountyPool>,
  currentBlock: bigint,
): DomainBountyPool {
  const existing = existingPools.get(domain);
  if (existing) return existing;

  return createEmptyBountyPool(domain, currentBlock);
}

/**
 * Fund bounty pool
 */
export function fundBountyPool(
  bountyPool: DomainBountyPool,
  amount: bigint,
  currentBlock: bigint,
): DomainBountyPool {
  return {
    ...bountyPool,
    balance: bountyPool.balance + amount,
    lastUpdatedAt: currentBlock,
  };
}

/**
 * Update bounty pool from verification rewards
 */
export function replenishBountyPool(
  bountyPool: DomainBountyPool,
  verificationReward: bigint,
  currentBlock: bigint,
): DomainBountyPool {
  const replenishment = BigInt(
    Math.floor(
      Number(verificationReward) *
        CHALLENGE_BOUNTY_CONFIG.bountyReplenishmentRate,
    ),
  );

  return {
    ...bountyPool,
    balance: bountyPool.balance + replenishment,
    lastUpdatedAt: currentBlock,
  };
}

/**
 * Auto-fund bounty pool based on risk score
 */
export function autoFundBountyPool(
  bountyPool: DomainBountyPool,
  riskScore: number,
  currentBlock: bigint,
): DomainBountyPool {
  if (riskScore <= CHALLENGE_BOUNTY_CONFIG.herfindahlTriggerThreshold) {
    return bountyPool;
  }

  // Calculate auto-funding amount based on risk
  const basePool = CHALLENGE_BOUNTY_CONFIG.baseBountyPool;
  const autoFundAmount = BigInt(Math.floor(Number(basePool) * riskScore));

  return {
    ...bountyPool,
    balance: bountyPool.balance + autoFundAmount,
    captureRiskScore: riskScore,
    autoGeneratedBounty: true,
    lastUpdatedAt: currentBlock,
  };
}
