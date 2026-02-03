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

import { sha512_256Hex, domainHash, HASH_DOMAINS } from '../../core/crypto/hashing.js';
import { commit, verifyCommitment, commitVerdict, verifyVerdictCommitment } from '../../core/crypto/commitment.js';
import { sign, verify, verifyDomainSeparated, signDomainSeparated } from '../../core/crypto/signatures.js';
import { bytesToHex } from '../../core/crypto/keys.js';
import { randomBytes } from '@noble/hashes/utils';
import type { PublicKey, PrivateKey } from '../../core/crypto/keys.js';
import type { ClaimSubmission, Confidence, Reference } from '../../core/knowledge/types.js';
import type {
  VerificationRound,
  ValidatorSelection,
  ValidatorCommitment,
  ValidatorReveal,
  VerificationResult,
  RoundPhase,
  ConsensusParams,
} from './types.js';

// ============ Round Creation ============

/**
 * Create a new verification round for a claim
 */
export function createVerificationRound(
  claim: ClaimSubmission,
  selectedValidators: ValidatorSelection[],
  startBlock: number,
  params: ConsensusParams
): VerificationRound {
  const claimId = sha512_256Hex(JSON.stringify(claim));

  return {
    id: `round_${claimId}_${startBlock}`,
    claimId,
    claim,

    startBlock,
    commitDeadline: startBlock + params.commitPhaseBlocks,
    revealDeadline: startBlock + params.commitPhaseBlocks + params.revealPhaseBlocks,
    completionBlock: startBlock + params.verificationRoundBlocks,

    selectedValidators,
    requiredValidators: Math.max(
      params.minValidators,
      Math.ceil(selectedValidators.length * params.validatorQuorum)
    ),

    phase: 'pending',

    commitments: new Map(),
    reveals: new Map(),
  };
}

/**
 * Start a verification round (transition to commit phase)
 */
export function startRound(round: VerificationRound): VerificationRound {
  return {
    ...round,
    phase: 'commit',
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
  verdict: 'confirmed' | 'rejected' | 'uncertain',
  confidence: Confidence,
  referencesChecked: string[]
): {
  commitment: ValidatorCommitment;
  nonce: string;
} {
  // Generate random nonce
  const nonce = bytesToHex(randomBytes(32));

  // Create commitment hash
  const commitData = JSON.stringify({
    roundId,
    verdict,
    confidence,
    referencesChecked: referencesChecked.sort(),
    nonce,
  });

  const commitmentHash = sha512_256Hex(domainHash(HASH_DOMAINS.COMMITMENT, commitData).toString());

  // Sign the commitment
  const signature = signDomainSeparated(
    'LGM.verification.commit.v1',
    `${roundId}:${commitmentHash}`,
    validatorPrivateKey
  );

  return {
    commitment: {
      validator: validatorPublicKey,
      commitment: commitmentHash,
      submittedAt: Date.now(),
      signature,
    },
    nonce,
  };
}

/**
 * Add a commitment to a round
 */
export function addCommitment(
  round: VerificationRound,
  commitment: ValidatorCommitment,
  currentBlock: number
): { round: VerificationRound; accepted: boolean; reason?: string } {
  // Check phase
  if (round.phase !== 'commit') {
    return { round, accepted: false, reason: 'Not in commit phase' };
  }

  // Check deadline
  if (currentBlock > round.commitDeadline) {
    return { round, accepted: false, reason: 'Commit deadline passed' };
  }

  // Check validator is selected
  const isSelected = round.selectedValidators.some(
    v => v.validator === commitment.validator && v.selected
  );
  if (!isSelected) {
    return { round, accepted: false, reason: 'Validator not selected' };
  }

  // Check not already committed
  if (round.commitments.has(commitment.validator)) {
    return { round, accepted: false, reason: 'Already committed' };
  }

  // Verify signature
  const valid = verifyDomainSeparated(
    'LGM.verification.commit.v1',
    `${round.id}:${commitment.commitment}`,
    commitment.signature,
    commitment.validator
  );
  if (!valid) {
    return { round, accepted: false, reason: 'Invalid signature' };
  }

  // Add commitment
  const newCommitments = new Map(round.commitments);
  newCommitments.set(commitment.validator, commitment);

  return {
    round: { ...round, commitments: newCommitments },
    accepted: true,
  };
}

/**
 * Transition from commit to reveal phase
 */
export function transitionToReveal(
  round: VerificationRound,
  currentBlock: number
): VerificationRound {
  if (round.phase !== 'commit') {
    return round;
  }

  if (currentBlock < round.commitDeadline) {
    return round;
  }

  return {
    ...round,
    phase: 'reveal',
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
  verdict: 'confirmed' | 'rejected' | 'uncertain',
  confidence: Confidence,
  nonce: string,
  referencesChecked: string[],
  reasoning?: string
): ValidatorReveal {
  // Sign the reveal
  const revealData = JSON.stringify({
    roundId,
    verdict,
    confidence,
    nonce,
    referencesChecked: referencesChecked.sort(),
    reasoning,
  });

  const signature = signDomainSeparated(
    'LGM.verification.reveal.v1',
    revealData,
    validatorPrivateKey
  );

  return {
    validator: validatorPublicKey,
    verdict,
    confidence,
    nonce,
    reasoning,
    referencesChecked,
    submittedAt: Date.now(),
    signature,
  };
}

/**
 * Add a reveal to a round
 */
export function addReveal(
  round: VerificationRound,
  reveal: ValidatorReveal,
  currentBlock: number
): { round: VerificationRound; accepted: boolean; reason?: string } {
  // Check phase
  if (round.phase !== 'reveal') {
    return { round, accepted: false, reason: 'Not in reveal phase' };
  }

  // Check deadline
  if (currentBlock > round.revealDeadline) {
    return { round, accepted: false, reason: 'Reveal deadline passed' };
  }

  // Check validator committed
  const commitment = round.commitments.get(reveal.validator);
  if (!commitment) {
    return { round, accepted: false, reason: 'No commitment from this validator' };
  }

  // Verify reveal matches commitment
  const commitData = JSON.stringify({
    roundId: round.id,
    verdict: reveal.verdict,
    confidence: reveal.confidence,
    referencesChecked: reveal.referencesChecked.sort(),
    nonce: reveal.nonce,
  });
  const expectedCommitment = sha512_256Hex(domainHash(HASH_DOMAINS.COMMITMENT, commitData).toString());

  if (expectedCommitment !== commitment.commitment) {
    return { round, accepted: false, reason: 'Reveal does not match commitment' };
  }

  // Verify signature
  const revealData = JSON.stringify({
    roundId: round.id,
    verdict: reveal.verdict,
    confidence: reveal.confidence,
    nonce: reveal.nonce,
    referencesChecked: reveal.referencesChecked.sort(),
    reasoning: reveal.reasoning,
  });

  const valid = verifyDomainSeparated(
    'LGM.verification.reveal.v1',
    revealData,
    reveal.signature,
    reveal.validator
  );
  if (!valid) {
    return { round, accepted: false, reason: 'Invalid signature' };
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
  validatorStakes: Map<string, bigint>,
  currentBlock: number,
  params: ConsensusParams
): VerificationRound {
  if (round.phase !== 'reveal' || currentBlock < round.revealDeadline) {
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
    const stake = validatorStakes.get(validator) || 0n;
    totalParticipatingStake += stake;

    switch (reveal.verdict) {
      case 'confirmed':
        confirmedStake += stake;
        break;
      case 'rejected':
        rejectedStake += stake;
        break;
      case 'uncertain':
        uncertainStake += stake;
        break;
    }
  }

  // Check for validators who committed but didn't reveal (slash them)
  for (const [validator, commitment] of round.commitments) {
    if (!round.reveals.has(validator)) {
      const stake = validatorStakes.get(validator) || 0n;
      const slashAmount = (stake * BigInt(Math.floor(params.missedVerificationSlash * 10000))) / 10000n;
      slashes.set(validator, slashAmount);
    }
  }

  // Determine outcome
  let outcome: 'accepted' | 'rejected' | 'contested' | 'insufficient';
  let finalConfidence: Confidence;

  const totalSelectedStake = round.selectedValidators.reduce(
    (sum, v) => sum + (validatorStakes.get(v.validator) || 0n),
    0n
  );

  const participationRate = Number(totalParticipatingStake) / Number(totalSelectedStake);

  if (participationRate < params.validatorQuorum) {
    outcome = 'insufficient';
    finalConfidence = 0;
  } else {
    const confirmedFraction = Number(confirmedStake) / Number(totalParticipatingStake);
    const rejectedFraction = Number(rejectedStake) / Number(totalParticipatingStake);

    if (confirmedFraction >= params.acceptanceThreshold) {
      outcome = 'accepted';
      finalConfidence = calculateAggregatedConfidence(round.reveals, validatorStakes, 'confirmed');
    } else if (rejectedFraction >= params.rejectionThreshold) {
      outcome = 'rejected';
      finalConfidence = 0;
    } else {
      outcome = 'contested';
      finalConfidence = 0.5; // Maximum uncertainty
    }
  }

  // Calculate rewards and slashes based on outcome
  if (outcome === 'accepted' || outcome === 'rejected') {
    const consensusVerdict = outcome === 'accepted' ? 'confirmed' : 'rejected';

    for (const [validator, reveal] of round.reveals) {
      if (reveal.verdict === consensusVerdict) {
        // Reward for correct verification
        rewards.set(validator, params.verificationReward);
      } else if (reveal.verdict !== 'uncertain') {
        // Slash for wrong verification
        const stake = validatorStakes.get(validator) || 0n;
        const slashAmount = (stake * BigInt(Math.floor(params.wrongVerificationSlash * 10000))) / 10000n;
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
      confirmed: Number(confirmedStake) / Number(totalParticipatingStake || 1n),
      rejected: Number(rejectedStake) / Number(totalParticipatingStake || 1n),
      uncertain: Number(uncertainStake) / Number(totalParticipatingStake || 1n),
    },
    totalValidators: round.selectedValidators.length,
    participatingValidators: round.reveals.size,
    agreementLevel: Math.max(
      Number(confirmedStake),
      Number(rejectedStake)
    ) / Number(totalParticipatingStake || 1n),
    rewards,
    slashes,
    recordedAt: currentBlock,
  };

  return {
    ...round,
    phase: 'complete',
    result,
  };
}

/**
 * Calculate aggregated confidence from reveals
 */
function calculateAggregatedConfidence(
  reveals: Map<string, ValidatorReveal>,
  stakes: Map<string, bigint>,
  consensusVerdict: 'confirmed' | 'rejected'
): Confidence {
  let weightedConfidence = 0;
  let totalWeight = 0;

  for (const [validator, reveal] of reveals) {
    if (reveal.verdict === consensusVerdict) {
      const stake = Number(stakes.get(validator) || 0n);
      weightedConfidence += reveal.confidence * stake;
      totalWeight += stake;
    }
  }

  return totalWeight > 0 ? weightedConfidence / totalWeight : 0;
}

// ============ Round Status ============

/**
 * Get current phase based on block number
 */
export function getCurrentPhase(
  round: VerificationRound,
  currentBlock: number
): RoundPhase {
  if (round.phase === 'complete') return 'complete';

  if (currentBlock < round.startBlock) return 'pending';
  if (currentBlock <= round.commitDeadline) return 'commit';
  if (currentBlock <= round.revealDeadline) return 'reveal';
  if (currentBlock <= round.completionBlock) return 'aggregation';

  return 'complete';
}

/**
 * Check if round has enough participation
 */
export function hasQuorum(
  round: VerificationRound,
  params: ConsensusParams
): boolean {
  return round.commits.size >= round.requiredValidators;
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
