/**
 * Domain Capture Defense Integration
 *
 * Integrates all 4 domain capture defense systems:
 * 1. Domain Qualification - Validators must qualify per domain
 * 2. Cross-Stratum Validation - Higher strata need lower stratum validators
 * 3. Challenge Bounties - Economic incentive to expose capture
 * 4. Reputation Isolation - Domain-specific reputation (no transfer)
 *
 * This integration provides a unified interface for the consensus engine
 * to use all defense mechanisms together.
 */

import type { PublicKey } from "../../core/crypto/keys.js";
import type {
  OntologicalStratum,
  DomainTree,
} from "../../core/knowledge/ontology/types.js";
import type { ClaimSubmission } from "../../core/knowledge/types.js";
import type {
  ValidatorSet,
  ValidatorReveal,
  ConsensusParams,
  VerificationResult,
} from "./types.js";

// Domain Qualification
import type {
  DomainQualification,
  ValidatorQualificationState,
  QualificationRequest,
  QualificationResult,
} from "../domain_qualification/types.js";
import {
  qualifyByStakeCommitment,
  qualifyByTrackRecord,
  qualifyByCrossReference,
  qualifyByStratumInheritance,
  updateQualificationMetrics,
  checkQualificationExpiration,
} from "../domain_qualification/pathways.js";
import {
  DOMAIN_QUALIFICATION_CONFIG,
  createEmptyQualificationState,
} from "../domain_qualification/types.js";

// Cross-Stratum Validation
import type { CrossStratumValidationResult } from "../cross_stratum/types.js";
import {
  validateCrossStratumCoverage,
  calculateCrossStratumConfidence,
  CROSS_STRATUM_CONFIG,
} from "../cross_stratum/index.js";

// Challenge Bounties
import type {
  CaptureChallenge,
  DomainBountyPool,
  CaptureRiskAnalysis,
  ChallengeSubmissionInput,
  VerificationHistoryEntry as BountyVerificationHistoryEntry,
} from "../challenge_bounties/types.js";
import {
  submitCaptureChallenge,
  resolveCaptureChallenge,
  finalizeChallenge,
  getOrCreateBountyPool,
  CHALLENGE_BOUNTY_CONFIG,
} from "../challenge_bounties/index.js";
import { analyzeCaptureRisk } from "../challenge_bounties/detection.js";

// Reputation Isolation
import type {
  IsolatedReputation,
  EffectiveReputationResult,
} from "../reputation_isolation/types.js";
import {
  updateDomainReputation,
  recordChallengeForReputation,
  applyTimeDecay,
  REPUTATION_ISOLATION_CONFIG,
  createEmptyIsolatedReputation,
} from "../reputation_isolation/index.js";

// Existing selection and confidence
import { getEligibleValidatorsForDomain } from "./validator_selection.js";
import { calculateIsolatedReputationConfidence } from "./confidence.js";

// ============ Unified Defense State ============

/**
 * Combined state for all domain capture defense systems
 */
export interface DomainCaptureDefenseState {
  /** Validator qualifications by validator public key */
  qualifications: Map<string, ValidatorQualificationState>;

  /** Isolated reputations by validator public key */
  isolatedReputations: Map<string, IsolatedReputation>;

  /** Active capture challenges by challenge ID */
  activeChallenges: Map<string, CaptureChallenge>;

  /** Bounty pools by domain */
  bountyPools: Map<string, DomainBountyPool>;

  /** Verification history by domain (for capture detection) */
  verificationHistory: Map<string, BountyVerificationHistoryEntry[]>;
}

/**
 * Create empty defense state
 */
export function createEmptyDefenseState(): DomainCaptureDefenseState {
  return {
    qualifications: new Map(),
    isolatedReputations: new Map(),
    activeChallenges: new Map(),
    bountyPools: new Map(),
    verificationHistory: new Map(),
  };
}

// ============ Unified Validator Selection ============

/**
 * Select validators for verification with all defense mechanisms applied
 *
 * This is the main entry point for validator selection that combines:
 * 1. Domain qualification filtering
 * 2. Cross-stratum requirements
 * 3. Isolated reputation weighting
 */
export function selectDefendedValidators(
  claim: ClaimSubmission,
  claimStratum: OntologicalStratum,
  validatorSet: ValidatorSet,
  defenseState: DomainCaptureDefenseState,
  domainTree: DomainTree,
  currentBlock: bigint,
  params: ConsensusParams,
): {
  selectedValidators: Array<{
    validator: PublicKey;
    stake: bigint;
    votingPower: bigint;
    qualification: DomainQualification;
    effectiveReputation: EffectiveReputationResult;
    stratum: OntologicalStratum;
  }>;
  crossStratumValidation: CrossStratumValidationResult;
  warnings: string[];
} {
  const domain = claim.domain;
  const warnings: string[] = [];

  // Get eligible validators with qualifications
  const eligible = getEligibleValidatorsForDomain(
    validatorSet,
    domain,
    defenseState.qualifications,
    defenseState.isolatedReputations,
    domainTree,
    currentBlock,
    params,
  );

  // Separate by stratum for cross-stratum validation
  const validatorStrata = new Map<string, OntologicalStratum>();
  for (const v of eligible) {
    // Get validator's qualification stratum
    validatorStrata.set(v.validator, v.qualification.stratum);
  }

  // Validate cross-stratum requirements
  // validateCrossStratumCoverage expects Map<validatorId, stratum>
  const crossStratumValidation = validateCrossStratumCoverage(
    claimStratum,
    validatorStrata,
  );

  if (!crossStratumValidation.meetsRequirements) {
    warnings.push(
      `Missing validators from strata: ${crossStratumValidation.missingStrata.join(", ")}`,
    );
  }

  // Map to output format
  const selectedValidators = eligible.map((e) => ({
    validator: e.validator,
    stake: e.stake,
    votingPower: e.votingPower,
    qualification: e.qualification,
    effectiveReputation: e.effectiveReputation,
    stratum: e.qualification.stratum,
  }));

  return {
    selectedValidators,
    crossStratumValidation,
    warnings,
  };
}

// ============ Unified Confidence Calculation ============

/**
 * Calculate confidence with all defense mechanisms applied
 *
 * Combines:
 * 1. Isolated reputation weighting
 * 2. Cross-stratum agreement/disagreement adjustment
 */
export function calculateDefendedConfidence(
  reveals: Map<string, ValidatorReveal>,
  validators: Map<string, { stake: bigint; reputation: number }>,
  defenseState: DomainCaptureDefenseState,
  validatorStrata: Map<string, OntologicalStratum>,
  domain: string,
  claimStratum: OntologicalStratum,
  domainTree: DomainTree,
  consensusVerdict: "confirmed" | "rejected",
): {
  confidence: number;
  isolatedBreakdown: {
    stakeWeighted: number;
    reputationWeighted: number;
    agreementBonus: number;
    reputationSources: { domain: number; stratum: number; global: number };
  };
  crossStratumAdjustment: {
    adjustment: number;
    lowerStratumAgreement: number;
    sameStratumAgreement: number;
  };
  finalConfidence: number;
} {
  // Calculate base confidence with isolated reputation
  const isolatedResult = calculateIsolatedReputationConfidence(
    reveals,
    validators,
    defenseState.isolatedReputations,
    domain,
    domainTree,
    consensusVerdict,
  );

  // Calculate cross-stratum adjustment
  const crossStratumResult = calculateCrossStratumConfidence(
    reveals,
    validatorStrata,
    claimStratum,
    validators,
  );

  // Apply cross-stratum adjustment
  const finalConfidence =
    isolatedResult.confidence * crossStratumResult.crossStratumAdjustment;

  return {
    confidence: isolatedResult.confidence,
    isolatedBreakdown: {
      stakeWeighted: isolatedResult.breakdown.stakeWeighted,
      reputationWeighted: isolatedResult.breakdown.reputationWeighted,
      agreementBonus: isolatedResult.breakdown.agreementBonus,
      reputationSources: isolatedResult.breakdown.reputationSources,
    },
    crossStratumAdjustment: {
      adjustment: crossStratumResult.crossStratumAdjustment,
      lowerStratumAgreement: crossStratumResult.lowerStratumAgreement,
      sameStratumAgreement: crossStratumResult.sameStratumAgreement,
    },
    finalConfidence,
  };
}

// ============ Post-Verification Updates ============

/**
 * Update defense state after a verification round completes
 */
export function updateDefenseStateAfterVerification(
  defenseState: DomainCaptureDefenseState,
  verificationResult: VerificationResult,
  reveals: Map<string, ValidatorReveal>,
  domain: string,
  stratum: OntologicalStratum,
  currentBlock: bigint,
): DomainCaptureDefenseState {
  const updatedState = { ...defenseState };

  // Update isolated reputations for all participating validators
  const newIsolatedReps = new Map(defenseState.isolatedReputations);

  for (const [validatorId, reveal] of reveals) {
    // Determine if validator was correct
    const wasCorrect =
      (verificationResult.outcome === "accepted" &&
        reveal.verdict === "confirmed") ||
      (verificationResult.outcome === "rejected" &&
        reveal.verdict === "rejected");

    // Get or create isolated reputation
    let isolatedRep = newIsolatedReps.get(validatorId);
    if (!isolatedRep) {
      isolatedRep = createEmptyIsolatedReputation(validatorId, currentBlock);
    }

    // Update domain reputation
    isolatedRep = updateDomainReputation(
      isolatedRep,
      domain,
      stratum,
      wasCorrect,
      reveal.confidence,
      verificationResult.finalConfidence,
      currentBlock,
    );

    newIsolatedReps.set(validatorId, isolatedRep);

    // Update qualification metrics
    const qualState = defenseState.qualifications.get(validatorId);
    if (qualState) {
      const qual = qualState.qualifications.get(domain);
      if (qual) {
        const updatedQual = updateQualificationMetrics(
          qual,
          wasCorrect,
          currentBlock,
        );
        const newQualState = { ...qualState };
        const newQuals = new Map(newQualState.qualifications);
        newQuals.set(domain, updatedQual);
        newQualState.qualifications = newQuals;
        updatedState.qualifications.set(validatorId, newQualState);
      }
    }
  }

  updatedState.isolatedReputations = newIsolatedReps;

  // Add to verification history for capture detection
  const history = defenseState.verificationHistory.get(domain) || [];

  // Convert outcome to the format expected by bounty types
  const finalOutcome: "accepted" | "rejected" | "contested" =
    verificationResult.outcome === "insufficient"
      ? "contested"
      : verificationResult.outcome;

  const historyEntry: BountyVerificationHistoryEntry = {
    claimId: verificationResult.claimId,
    blockNumber: BigInt(verificationResult.recordedAt),
    validators: Array.from(reveals.keys()),
    verdicts: Array.from(reveals.values()).map((r) => r.verdict),
    finalOutcome,
    submissionTimestamps: Array.from(reveals.values()).map((r) =>
      BigInt(r.submittedAtHeight),
    ),
  };

  // Keep last 100 entries per domain
  const newHistory = [...history, historyEntry].slice(-100);
  updatedState.verificationHistory = new Map(defenseState.verificationHistory);
  updatedState.verificationHistory.set(domain, newHistory);

  return updatedState;
}

// ============ Capture Detection & Challenge ============

/**
 * Analyze a domain for capture risk
 */
export function analyzeDomainCaptureRisk(
  defenseState: DomainCaptureDefenseState,
  domain: string,
  currentBlock: bigint,
): CaptureRiskAnalysis {
  const history = defenseState.verificationHistory.get(domain) || [];

  return analyzeCaptureRisk(domain, history, currentBlock);
}

/**
 * Submit a capture challenge
 */
export function submitCaptureChallengeToState(
  defenseState: DomainCaptureDefenseState,
  input: ChallengeSubmissionInput,
  currentBlock: bigint,
  domainTree: DomainTree,
): {
  defenseState: DomainCaptureDefenseState;
  challenge: CaptureChallenge | null;
  errors: string[];
} {
  const result = submitCaptureChallenge(
    input.challenger,
    input.domain,
    input.evidence,
    input.stake,
    currentBlock,
    domainTree,
  );

  if (!result.validation.valid || !result.challenge) {
    return {
      defenseState,
      challenge: null,
      errors: result.validation.errors,
    };
  }

  // Add challenge to state
  const newChallenges = new Map(defenseState.activeChallenges);
  newChallenges.set(result.challenge.id, result.challenge);

  return {
    defenseState: {
      ...defenseState,
      activeChallenges: newChallenges,
    },
    challenge: result.challenge,
    errors: [],
  };
}

/**
 * Resolve a capture challenge
 */
export function resolveCaptureChallengInState(
  defenseState: DomainCaptureDefenseState,
  challengeId: string,
  outcome: "upheld" | "rejected" | "partial",
  justification: string,
  validatorStakes: Map<string, bigint>,
  currentBlock: bigint,
): {
  defenseState: DomainCaptureDefenseState;
  challenge: CaptureChallenge | null;
  challengerReward: bigint;
  validatorSlashes: Map<string, bigint>;
  errors: string[];
} {
  const challenge = defenseState.activeChallenges.get(challengeId);
  if (!challenge) {
    return {
      defenseState,
      challenge: null,
      challengerReward: 0n,
      validatorSlashes: new Map(),
      errors: ["Challenge not found"],
    };
  }

  // Get or create bounty pool
  const bountyPool = getOrCreateBountyPool(
    challenge.domain,
    defenseState.bountyPools,
    currentBlock,
  );

  // Resolve the challenge
  const resolution = resolveCaptureChallenge(
    challenge,
    outcome,
    bountyPool,
    validatorStakes,
    currentBlock,
    justification,
  );

  // Finalize the challenge with resolution
  const resolvedChallenge = finalizeChallenge(challenge, resolution.resolution);

  // Calculate rewards/slashes
  const challengerReward = resolution.resolution.challengerReward;
  const validatorSlashes = resolution.resolution.validatorSlashes;

  // Update challenge reputation for validators
  const newIsolatedReps = new Map(defenseState.isolatedReputations);
  if (outcome === "upheld" || outcome === "partial") {
    for (const validatorId of challenge.accusedValidators) {
      let isolatedRep = newIsolatedReps.get(validatorId);
      if (isolatedRep) {
        isolatedRep = recordChallengeForReputation(
          isolatedRep,
          challenge.domain,
          true, // Challenge was lost (validator was wrong)
          currentBlock,
        );
        newIsolatedReps.set(validatorId, isolatedRep);
      }
    }
  }

  // Remove from active challenges
  const newChallenges = new Map(defenseState.activeChallenges);
  newChallenges.delete(challengeId);

  // Update bounty pool
  const newBountyPools = new Map(defenseState.bountyPools);
  newBountyPools.set(challenge.domain, resolution.updatedBountyPool);

  return {
    defenseState: {
      ...defenseState,
      activeChallenges: newChallenges,
      bountyPools: newBountyPools,
      isolatedReputations: newIsolatedReps,
    },
    challenge: resolvedChallenge,
    challengerReward,
    validatorSlashes,
    errors: [],
  };
}

// ============ Qualification Management ============

/**
 * Process a qualification request
 */
export function processQualificationRequest(
  defenseState: DomainCaptureDefenseState,
  request: QualificationRequest,
  currentBlock: bigint,
  domainTree: DomainTree,
): {
  defenseState: DomainCaptureDefenseState;
  result: QualificationResult;
} {
  let result: QualificationResult;

  // Get existing qualification state
  let qualState = defenseState.qualifications.get(request.validatorId);
  if (!qualState) {
    qualState = createEmptyQualificationState(
      request.validatorId,
      currentBlock,
    );
  }

  switch (request.pathway) {
    case "stake_commitment":
      result = qualifyByStakeCommitment(
        request.validatorId,
        request.domain,
        request.stake || 0n,
        currentBlock,
        domainTree,
      );
      break;

    case "track_record":
      result = qualifyByTrackRecord(
        request.validatorId,
        request.domain,
        qualState,
        currentBlock,
        domainTree,
      );
      break;

    case "cross_reference":
      // Need endorser qualifications - simplified for now
      result = qualifyByCrossReference(
        request.validatorId,
        request.domain,
        request.endorsers || [],
        new Map(), // Would need endorser qualifications
        new Map(), // Would need endorser stakes
        currentBlock,
        domainTree,
      );
      break;

    case "stratum_inheritance":
      result = qualifyByStratumInheritance(
        request.validatorId,
        request.domain,
        qualState,
        currentBlock,
        domainTree,
      );
      break;

    default:
      result = {
        granted: false,
        errors: ["Unknown qualification pathway"],
      };
  }

  if (result.granted && result.qualification) {
    // Add qualification to state
    const newQualState = { ...qualState };
    const newQuals = new Map(newQualState.qualifications);
    newQuals.set(request.domain, result.qualification);
    newQualState.qualifications = newQuals;

    // Update locked stake if applicable
    if (result.qualification.lockedStake) {
      newQualState.totalLockedStake += result.qualification.lockedStake;
    }

    // Update pathway count
    newQualState.pathwayCounts[request.pathway]++;

    const newQualifications = new Map(defenseState.qualifications);
    newQualifications.set(request.validatorId, newQualState);

    return {
      defenseState: {
        ...defenseState,
        qualifications: newQualifications,
      },
      result,
    };
  }

  return { defenseState, result };
}

// ============ Periodic Maintenance ============

/**
 * Apply time decay to all isolated reputations
 */
export function applyPeriodicDecay(
  defenseState: DomainCaptureDefenseState,
  currentBlock: bigint,
): DomainCaptureDefenseState {
  const newIsolatedReps = new Map<string, IsolatedReputation>();

  for (const [validatorId, isolatedRep] of defenseState.isolatedReputations) {
    const decayed = applyTimeDecay(isolatedRep, currentBlock);
    newIsolatedReps.set(validatorId, decayed);
  }

  return {
    ...defenseState,
    isolatedReputations: newIsolatedReps,
  };
}

/**
 * Check and expire qualifications
 */
export function expireQualifications(
  defenseState: DomainCaptureDefenseState,
  currentBlock: bigint,
): DomainCaptureDefenseState {
  const newQualifications = new Map<string, ValidatorQualificationState>();

  for (const [validatorId, qualState] of defenseState.qualifications) {
    const newQualState = { ...qualState };
    const newQuals = new Map<string, DomainQualification>();

    for (const [domain, qual] of qualState.qualifications) {
      const checked = checkQualificationExpiration(qual, currentBlock);
      if (checked.status !== "expired") {
        newQuals.set(domain, checked);
      }
    }

    newQualState.qualifications = newQuals;
    newQualifications.set(validatorId, newQualState);
  }

  return {
    ...defenseState,
    qualifications: newQualifications,
  };
}

/**
 * Check for domains that need auto-funded bounties
 */
export function checkAutoFundBounties(
  defenseState: DomainCaptureDefenseState,
  currentBlock: bigint,
): {
  defenseState: DomainCaptureDefenseState;
  autoFundedDomains: string[];
} {
  const autoFundedDomains: string[] = [];
  const newBountyPools = new Map(defenseState.bountyPools);

  for (const [domain] of defenseState.verificationHistory) {
    const riskAnalysis = analyzeDomainCaptureRisk(
      defenseState,
      domain,
      currentBlock,
    );

    if (riskAnalysis.shouldCreateBounty) {
      const pool = getOrCreateBountyPool(domain, newBountyPools, currentBlock);
      pool.autoGeneratedBounty = true;
      pool.balance += riskAnalysis.suggestedBountySize;
      pool.captureRiskScore = riskAnalysis.riskScore;
      pool.lastAnalyzedBlock = currentBlock;

      newBountyPools.set(domain, pool);
      autoFundedDomains.push(domain);
    }
  }

  return {
    defenseState: {
      ...defenseState,
      bountyPools: newBountyPools,
    },
    autoFundedDomains,
  };
}

// ============ Exports ============

export {
  // Re-export for convenience
  DOMAIN_QUALIFICATION_CONFIG,
  CROSS_STRATUM_CONFIG,
  CHALLENGE_BOUNTY_CONFIG,
  REPUTATION_ISOLATION_CONFIG,
};
