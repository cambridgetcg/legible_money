/**
 * Verification Round
 *
 * A verification round is the process by which validators
 * collectively verify a claim and reach consensus on its truth.
 *
 * Phases:
 * 1. COMMIT: Validators submit commitment (hash of verdict)
 * 2. REVEAL: Validators reveal their verdicts
 * 3. AGGREGATION: System aggregates results
 * 4. COMPLETE: Final result recorded
 */

import {
  sha512_256Hex,
  domainHash,
  HASH_DOMAINS,
  hashesEqual,
} from "../../core/crypto/hashing.js";
import {
  verifyDomainSeparated,
  signDomainSeparated,
} from "../../core/crypto/signatures.js";
import { bytesToHex } from "../../core/crypto/keys.js";
import { randomBytes } from "@noble/hashes/utils";
// GAP-002 FIX: Import VRF verification for validator selection
import { verifyValidatorVRFSelection } from "./validator_selection.js";
// V1 FIX: Import multi-modal confidence (stake + reputation + agreement)
import { calculateFinalConfidence } from "./confidence.js";
import type { PublicKey, PrivateKey } from "../../core/crypto/keys.js";
import type {
  ClaimSubmission,
  Confidence,
} from "../../core/knowledge/types.js";
import type { SchemaTiming } from "../../core/knowledge/schemas/types.js";
import type { OntologicalStratum } from "../../core/knowledge/ontology/types.js";
import type {
  VerificationRound,
  ValidatorSelection,
  ValidatorCommitment,
  ValidatorReveal,
  VerificationResult,
  RoundPhase,
  ConsensusParams,
} from "./types.js";

// Cross-Stratum Validation imports
import type { CrossStratumValidationResult } from "../cross_stratum/types.js";
import {
  validateCrossStratumCoverage,
  calculateCrossStratumConfidence,
  CROSS_STRATUM_CONFIG,
} from "../cross_stratum/index.js";

/** Validator info for confidence aggregation (stake + reputation) */
export type ValidatorInfoMap = Map<
  string,
  { stake: bigint; reputation: number }
>;

// ============ Round Creation ============

/**
 * VULN-014 FIX: Generate unique claim ID that includes temporal context
 * This prevents claim ID collisions when the same claim content is submitted
 * at different times or by different submitters.
 */
export function generateClaimId(
  claim: ClaimSubmission,
  startBlock: number,
  submitter?: string,
): string {
  // Include claim content, block number, and optional submitter for uniqueness
  const claimData = {
    claim: JSON.stringify(claim, (_, v) =>
      typeof v === "bigint" ? v.toString() : v,
    ),
    startBlock,
    submitter: submitter || "unknown",
  };
  return sha512_256Hex(JSON.stringify(claimData));
}

/**
 * Round timing configuration
 *
 * Can come from schema (dynamic) or consensus params (default)
 */
export interface RoundTiming {
  commitPhaseBlocks: number;
  revealPhaseBlocks: number;
  totalRoundBlocks: number;
}

/**
 * Get timing for a verification round
 *
 * Uses schema timing if provided, otherwise falls back to consensus params
 */
export function getRoundTiming(
  params: ConsensusParams,
  schemaTiming?: SchemaTiming,
): RoundTiming {
  if (schemaTiming) {
    return {
      commitPhaseBlocks: schemaTiming.commitPhaseBlocks,
      revealPhaseBlocks: schemaTiming.revealPhaseBlocks,
      totalRoundBlocks: schemaTiming.totalRoundBlocks,
    };
  }

  return {
    commitPhaseBlocks: params.commitPhaseBlocks,
    revealPhaseBlocks: params.revealPhaseBlocks,
    totalRoundBlocks: params.verificationRoundBlocks,
  };
}

/**
 * Create a new verification round for a claim
 *
 * @param claim - The claim to verify
 * @param selectedValidators - Validators selected for this round
 * @param startBlock - Block number when round starts
 * @param params - Consensus parameters (default timing)
 * @param submitter - Optional submitter for unique ID generation
 * @param schemaTiming - Optional schema-specific timing (overrides params)
 */
export function createVerificationRound(
  claim: ClaimSubmission,
  selectedValidators: ValidatorSelection[],
  startBlock: number,
  params: ConsensusParams,
  submitter?: string, // VULN-014 FIX: Optional submitter for unique ID generation
  schemaTiming?: SchemaTiming, // Schema-based dynamic timing
): VerificationRound {
  // VULN-014 FIX: Generate unique claim ID including block context
  const claimId = generateClaimId(claim, startBlock, submitter);

  // Get timing (schema overrides params)
  const timing = getRoundTiming(params, schemaTiming);

  return {
    id: `round_${claimId}_${startBlock}`,
    claimId,
    claim,

    startBlock,
    commitDeadline: startBlock + timing.commitPhaseBlocks,
    revealDeadline:
      startBlock + timing.commitPhaseBlocks + timing.revealPhaseBlocks,
    completionBlock: startBlock + timing.totalRoundBlocks,

    selectedValidators,
    requiredValidators: Math.max(
      params.minValidators,
      Math.ceil(selectedValidators.length * params.validatorQuorum),
    ),

    phase: "pending",

    commitments: new Map(),
    reveals: new Map(),

    // Store schema info for validators
    schemaId: claim.schemaId,
  };
}

/**
 * Start a verification round (transition to commit phase)
 */
export function startRound(round: VerificationRound): VerificationRound {
  return {
    ...round,
    phase: "commit",
  };
}

// ============ Commit Phase ============

/**
 * Generate a validator's commitment for a claim
 */
export function generateCommitment(
  validatorPrivateKey: PrivateKey,
  validatorPublicKey: PublicKey,
  roundId: string,
  verdict: "confirmed" | "rejected" | "uncertain",
  confidence: Confidence,
  referencesChecked: string[],
): {
  commitment: ValidatorCommitment;
  salt: string;
} {
  // Generate random salt for commit-reveal
  const salt = bytesToHex(randomBytes(32));

  // Create commitment hash
  const commitData = JSON.stringify({
    roundId,
    verdict,
    confidence,
    referencesChecked: referencesChecked.sort(),
    salt,
  });

  const commitmentHash = sha512_256Hex(
    domainHash(HASH_DOMAINS.COMMITMENT, commitData).toString(),
  );

  // Sign the commitment
  const signature = signDomainSeparated(
    "LGM.verification.commit.v1",
    `${roundId}:${commitmentHash}`,
    validatorPrivateKey,
  );

  return {
    commitment: {
      validator: validatorPublicKey,
      commitment: commitmentHash,
      submittedAtHeight: Date.now(),
      signature,
    },
    salt,
  };
}

/**
 * GAP-002 FIX: Add a commitment to a round with proper VRF verification
 *
 * This is the secure version that verifies the validator's VRF proof
 * to ensure they are legitimately selected for this verification round.
 */
export function addCommitmentWithVRF(
  round: VerificationRound,
  commitment: ValidatorCommitment,
  currentBlock: number,
  previousBlockHash: string,
  validatorStake: bigint,
  totalStake: bigint,
  params: ConsensusParams,
): {
  round: VerificationRound;
  accepted: boolean;
  reason?: string;
  priority?: bigint;
} {
  // Check phase
  if (round.phase !== "commit") {
    return { round, accepted: false, reason: "Not in commit phase" };
  }

  // Check deadline
  if (currentBlock > round.commitDeadline) {
    return { round, accepted: false, reason: "Commit deadline passed" };
  }

  // Check not already committed
  if (round.commitments.has(commitment.validator)) {
    return { round, accepted: false, reason: "Already committed" };
  }

  // SECURITY: Require VRF proof for secure commitment verification
  if (!commitment.vrfOutput || !commitment.vrfProof) {
    return {
      round,
      accepted: false,
      reason: "VRF output and proof required for secure commitment",
    };
  }

  // GAP-002 FIX: Verify VRF-based selection
  // This proves the validator is legitimately selected based on their stake-weighted probability
  const vrfResult = verifyValidatorVRFSelection(
    commitment.validator,
    commitment.vrfOutput,
    commitment.vrfProof,
    round.claimId,
    BigInt(round.startBlock),
    previousBlockHash,
    validatorStake,
    totalStake,
    params,
  );

  if (!vrfResult.valid) {
    return {
      round,
      accepted: false,
      reason: `VRF verification failed: ${vrfResult.reason}`,
    };
  }

  // Verify signature
  const sigValid = verifyDomainSeparated(
    "LGM.verification.commit.v1",
    `${round.id}:${commitment.commitment}`,
    commitment.signature,
    commitment.validator,
  );
  if (!sigValid) {
    return { round, accepted: false, reason: "Invalid signature" };
  }

  // Add commitment
  const newCommitments = new Map(round.commitments);
  newCommitments.set(commitment.validator, commitment);

  return {
    round: { ...round, commitments: newCommitments },
    accepted: true,
    priority: vrfResult.priority,
  };
}

// SECURITY: The deprecated addCommitment function has been removed.
// It allowed commitments without VRF proof verification, enabling anyone
// with minimal stake to participate in ALL verification rounds.
// Use addCommitmentWithVRF instead which properly verifies VRF proofs.

/**
 * Transition from commit to reveal phase
 */
export function transitionToReveal(
  round: VerificationRound,
  currentBlock: number,
): VerificationRound {
  if (round.phase !== "commit") {
    return round;
  }

  if (currentBlock < round.commitDeadline) {
    return round;
  }

  return {
    ...round,
    phase: "reveal",
  };
}

// ============ Reveal Phase ============

/**
 * Generate a validator's reveal
 */
export function generateReveal(
  validatorPrivateKey: PrivateKey,
  validatorPublicKey: PublicKey,
  roundId: string,
  verdict: "confirmed" | "rejected" | "uncertain",
  confidence: Confidence,
  salt: string,
  referencesChecked: string[],
  reasoning?: string,
): ValidatorReveal {
  // Sign the reveal
  const revealData = JSON.stringify({
    roundId,
    verdict,
    confidence,
    salt,
    referencesChecked: referencesChecked.sort(),
    reasoning,
  });

  const signature = signDomainSeparated(
    "LGM.verification.reveal.v1",
    revealData,
    validatorPrivateKey,
  );

  return {
    validator: validatorPublicKey,
    verdict,
    confidence,
    salt,
    reasoning,
    referencesChecked,
    submittedAtHeight: Date.now(),
    signature,
  };
}

/**
 * Add a reveal to a round
 */
export function addReveal(
  round: VerificationRound,
  reveal: ValidatorReveal,
  currentBlock: number,
): { round: VerificationRound; accepted: boolean; reason?: string } {
  // Check phase
  if (round.phase !== "reveal") {
    return { round, accepted: false, reason: "Not in reveal phase" };
  }

  // Check deadline
  if (currentBlock > round.revealDeadline) {
    return { round, accepted: false, reason: "Reveal deadline passed" };
  }

  // Check validator committed
  const commitment = round.commitments.get(reveal.validator);
  if (!commitment) {
    return {
      round,
      accepted: false,
      reason: "No commitment from this validator",
    };
  }

  // Verify reveal matches commitment
  const commitData = JSON.stringify({
    roundId: round.id,
    verdict: reveal.verdict,
    confidence: reveal.confidence,
    referencesChecked: reveal.referencesChecked.sort(),
    salt: reveal.salt,
  });
  const expectedCommitment = sha512_256Hex(
    domainHash(HASH_DOMAINS.COMMITMENT, commitData).toString(),
  );

  // ZD-C02 FIX: Use constant-time comparison to prevent timing side-channels
  if (!hashesEqual(expectedCommitment, commitment.commitment)) {
    return {
      round,
      accepted: false,
      reason: "Reveal does not match commitment",
    };
  }

  // Verify signature
  const revealData = JSON.stringify({
    roundId: round.id,
    verdict: reveal.verdict,
    confidence: reveal.confidence,
    salt: reveal.salt,
    referencesChecked: reveal.referencesChecked.sort(),
    reasoning: reveal.reasoning,
  });

  const valid = verifyDomainSeparated(
    "LGM.verification.reveal.v1",
    revealData,
    reveal.signature,
    reveal.validator,
  );
  if (!valid) {
    return { round, accepted: false, reason: "Invalid signature" };
  }

  // Add reveal
  const newReveals = new Map(round.reveals);
  newReveals.set(reveal.validator, reveal);

  return {
    round: { ...round, reveals: newReveals },
    accepted: true,
  };
}

// ============ Aggregation Phase ============

/**
 * Transition to aggregation and compute result
 */
export function aggregateResults(
  round: VerificationRound,
  validatorStakes: ValidatorInfoMap,
  currentBlock: number,
  params: ConsensusParams,
): VerificationRound {
  if (round.phase !== "reveal" || currentBlock < round.revealDeadline) {
    return round;
  }

  // Calculate stake-weighted votes
  let confirmedStake = 0n;
  let rejectedStake = 0n;
  let uncertainStake = 0n;
  let totalParticipatingStake = 0n;

  const rewards = new Map<string, bigint>();
  const slashes = new Map<string, bigint>();

  // Process reveals
  for (const [validator, reveal] of round.reveals) {
    const stake = validatorStakes.get(validator)?.stake || 0n;
    totalParticipatingStake += stake;

    switch (reveal.verdict) {
      case "confirmed":
        confirmedStake += stake;
        break;
      case "rejected":
        rejectedStake += stake;
        break;
      case "uncertain":
        uncertainStake += stake;
        break;
    }
  }

  // Check for validators who committed but didn't reveal (slash them)
  for (const [validator, _commitment] of round.commitments) {
    if (!round.reveals.has(validator)) {
      const stake = validatorStakes.get(validator)?.stake || 0n;
      const slashAmount =
        (stake * BigInt(Math.floor(params.missedVerificationSlash * 10000))) /
        10000n;
      slashes.set(validator, slashAmount);
    }
  }

  // Determine outcome
  let outcome: "accepted" | "rejected" | "contested" | "insufficient";
  let finalConfidence: Confidence;

  const totalSelectedStake = round.selectedValidators.reduce(
    (sum, v) => sum + (validatorStakes.get(v.validator)?.stake || 0n),
    0n,
  );

  // Use bigint basis-point arithmetic to avoid Number() overflow on real stakes
  // (1111 LGM = 1.111×10²¹ > Number.MAX_SAFE_INTEGER)
  const participationBps =
    totalSelectedStake > 0n
      ? (totalParticipatingStake * 10000n) / totalSelectedStake
      : 0n;
  const quorumBps = BigInt(Math.floor(params.validatorQuorum * 10000));

  if (participationBps < quorumBps) {
    outcome = "insufficient";
    finalConfidence = 0;
  } else {
    const confirmedBps =
      totalParticipatingStake > 0n
        ? (confirmedStake * 10000n) / totalParticipatingStake
        : 0n;
    const rejectedBps =
      totalParticipatingStake > 0n
        ? (rejectedStake * 10000n) / totalParticipatingStake
        : 0n;
    const acceptanceBps = BigInt(
      Math.floor(params.acceptanceThreshold * 10000),
    );
    const rejectionBps = BigInt(Math.floor(params.rejectionThreshold * 10000));

    if (confirmedBps >= acceptanceBps) {
      outcome = "accepted";
      // P0-M3 FIX: Use reputation-aware confidence (stake + reputation + agreement)
      // instead of raw stake-only aggregation
      finalConfidence = calculateFinalConfidence(
        round.reveals,
        validatorStakes,
        "confirmed",
      ).confidence;
    } else if (rejectedBps >= rejectionBps) {
      outcome = "rejected";
      finalConfidence = 0;
    } else {
      outcome = "contested";
      finalConfidence = 0.5; // Maximum uncertainty
    }
  }

  // Calculate rewards and slashes based on outcome
  if (outcome === "accepted" || outcome === "rejected") {
    const consensusVerdict = outcome === "accepted" ? "confirmed" : "rejected";

    for (const [validator, reveal] of round.reveals) {
      if (reveal.verdict === consensusVerdict) {
        // Reward for correct verification
        rewards.set(validator, params.verificationReward);
      } else if (reveal.verdict !== "uncertain") {
        // Slash for wrong verification
        const stake = validatorStakes.get(validator)?.stake || 0n;
        const slashAmount =
          (stake * BigInt(Math.floor(params.wrongVerificationSlash * 10000))) /
          10000n;
        slashes.set(validator, (slashes.get(validator) || 0n) + slashAmount);
      }
      // Uncertain verdicts get no reward but no slash
    }
  }

  const result: VerificationResult = {
    claimId: round.claimId,
    outcome,
    finalConfidence,
    confidenceBreakdown: {
      confirmed:
        totalParticipatingStake > 0n
          ? Number((confirmedStake * 10000n) / totalParticipatingStake) / 10000
          : 0,
      rejected:
        totalParticipatingStake > 0n
          ? Number((rejectedStake * 10000n) / totalParticipatingStake) / 10000
          : 0,
      uncertain:
        totalParticipatingStake > 0n
          ? Number((uncertainStake * 10000n) / totalParticipatingStake) / 10000
          : 0,
    },
    totalValidators: round.selectedValidators.length,
    participatingValidators: round.reveals.size,
    agreementLevel:
      totalParticipatingStake > 0n
        ? Number(
            ((confirmedStake > rejectedStake ? confirmedStake : rejectedStake) *
              10000n) /
              totalParticipatingStake,
          ) / 10000
        : 0,
    rewards,
    slashes,
    recordedAt: currentBlock,
  };

  return {
    ...round,
    phase: "complete",
    result,
  };
}

// calculateAggregatedConfidence replaced by calculateFinalConfidence from
// confidence.ts — adds reputation weighting + agreement bonus (P0-M3 fix)

// ============ Round Status ============

/**
 * Get current phase based on block number
 */
export function getCurrentPhase(
  round: VerificationRound,
  currentBlock: number,
): RoundPhase {
  if (round.phase === "complete") return "complete";

  if (currentBlock < round.startBlock) return "pending";
  if (currentBlock <= round.commitDeadline) return "commit";
  if (currentBlock <= round.revealDeadline) return "reveal";
  if (currentBlock <= round.completionBlock) return "aggregation";

  return "complete";
}

/**
 * Check if round has enough participation
 */
export function hasQuorum(
  round: VerificationRound,
  _params: ConsensusParams,
): boolean {
  // GAP-001 FIX: Changed `commits` to `commitments` to match type definition
  return round.commitments.size >= round.requiredValidators;
}

/**
 * Get round summary
 */
export function getRoundSummary(round: VerificationRound): {
  id: string;
  claimId: string;
  phase: RoundPhase;
  commitments: number;
  reveals: number;
  required: number;
  hasResult: boolean;
  outcome?: string;
} {
  return {
    id: round.id,
    claimId: round.claimId,
    phase: round.phase,
    commitments: round.commitments.size,
    reveals: round.reveals.size,
    required: round.requiredValidators,
    hasResult: !!round.result,
    outcome: round.result?.outcome,
  };
}

// ============ Cross-Stratum Validation ============

/**
 * Extended verification round with cross-stratum requirements
 */
export interface CrossStratumVerificationRound extends VerificationRound {
  /** Stratum of the claim being verified */
  claimStratum: OntologicalStratum;

  /** Cross-stratum validation status */
  crossStratumValidation?: CrossStratumValidationResult;

  /** Validators by stratum */
  validatorStrata: Map<string, OntologicalStratum>;
}

/**
 * Create a verification round with cross-stratum requirements
 *
 * For claims in higher strata (e.g., biology), this ensures
 * validators from lower strata (e.g., chemistry) are included.
 */
export function createCrossStratumVerificationRound(
  claim: ClaimSubmission,
  claimStratum: OntologicalStratum,
  selectedValidators: ValidatorSelection[],
  validatorStrata: Map<string, OntologicalStratum>,
  startBlock: number,
  params: ConsensusParams,
  submitter?: string,
  schemaTiming?: SchemaTiming,
): CrossStratumVerificationRound {
  // Create base round
  const baseRound = createVerificationRound(
    claim,
    selectedValidators,
    startBlock,
    params,
    submitter,
    schemaTiming,
  );

  // Validate cross-stratum coverage
  // validateCrossStratumCoverage expects Map<validatorId, stratum>
  const crossStratumValidation = validateCrossStratumCoverage(
    claimStratum,
    validatorStrata,
  );

  return {
    ...baseRound,
    claimStratum,
    crossStratumValidation,
    validatorStrata,
  };
}

/**
 * Aggregate results with cross-stratum confidence adjustment
 *
 * When lower stratum validators disagree with higher stratum validators,
 * a confidence penalty is applied. When they agree, a bonus is applied.
 */
export function aggregateCrossStratumResults(
  round: CrossStratumVerificationRound,
  validatorStakes: ValidatorInfoMap,
  currentBlock: number,
  params: ConsensusParams,
): CrossStratumVerificationRound {
  // First do base aggregation
  const aggregatedRound = aggregateResults(
    round,
    validatorStakes,
    currentBlock,
    params,
  );

  if (aggregatedRound.phase !== "complete" || !aggregatedRound.result) {
    return {
      ...aggregatedRound,
      claimStratum: round.claimStratum,
      validatorStrata: round.validatorStrata,
    };
  }

  // Calculate cross-stratum confidence adjustment
  // P0-M3 FIX: validatorStakes now carries real reputation (no more hardcoded 0.5)
  const crossStratumResult = calculateCrossStratumConfidence(
    aggregatedRound.reveals,
    round.validatorStrata,
    round.claimStratum,
    validatorStakes,
  );

  // Apply cross-stratum adjustment to final confidence
  const adjustedConfidence =
    aggregatedRound.result.finalConfidence *
    crossStratumResult.crossStratumAdjustment;

  // Update result
  const updatedResult: VerificationResult = {
    ...aggregatedRound.result,
    finalConfidence: adjustedConfidence,
    // Add cross-stratum info to breakdown if needed
    confidenceBreakdown: {
      ...aggregatedRound.result.confidenceBreakdown,
    },
  };

  return {
    ...aggregatedRound,
    claimStratum: round.claimStratum,
    validatorStrata: round.validatorStrata,
    crossStratumValidation: {
      ...round.crossStratumValidation!,
      lowerStratumAgreement: crossStratumResult.lowerStratumAgreement,
      sameStratumAgreement: crossStratumResult.sameStratumAgreement,
      crossStratumConfidenceAdjustment:
        crossStratumResult.crossStratumAdjustment,
    },
    result: updatedResult,
  };
}

/**
 * Check if round meets cross-stratum requirements
 *
 * Returns false if the claim stratum requires lower stratum validators
 * but they weren't included.
 */
export function meetsCrossStratumRequirements(
  round: CrossStratumVerificationRound,
): {
  meets: boolean;
  missingStrata: OntologicalStratum[];
  confidencePenalty: number;
} {
  if (!round.crossStratumValidation) {
    return { meets: true, missingStrata: [], confidencePenalty: 0 };
  }

  return {
    meets: round.crossStratumValidation.meetsRequirements,
    missingStrata: round.crossStratumValidation.missingStrata,
    confidencePenalty: round.crossStratumValidation.meetsRequirements
      ? 0
      : CROSS_STRATUM_CONFIG.partialCoverageConfidencePenalty,
  };
}
