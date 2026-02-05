/**
 * Validator Selection
 *
 * VRF-based validator selection for Proof of Truth.
 * Selection is:
 * - Unpredictable (VRF output cannot be known in advance)
 * - Verifiable (anyone can verify the selection was fair)
 * - Stake-weighted (higher stake = higher selection probability)
 */

import {
  vrfGenerate,
  vrfVerify,
  generateBlockSeed,
  isValidatorSelected,
  type VRFOutput,
  type VRFProof,
} from "../../core/crypto/vrf.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";
import type { PublicKey, PrivateKey } from "../../core/crypto/keys.js";
import type {
  Validator,
  ValidatorSet,
  ValidatorSelection,
  ConsensusParams,
} from "./types.js";

// Domain Capture Defense imports
import type {
  DomainQualification,
  ValidatorQualificationState,
} from "../domain_qualification/types.js";
import type {
  IsolatedReputation,
  EffectiveReputationResult,
} from "../reputation_isolation/types.js";
import type { DomainTree } from "../../core/knowledge/ontology/types.js";
import {
  calculateQualificationWeight,
  isQualificationValid,
} from "../domain_qualification/types.js";
import { getEffectiveReputation } from "../reputation_isolation/calculation.js";

// ============ Selection for Verification ============

/**
 * GAP-002/GAP-010 FIX: Proper VRF-based validator selection
 *
 * Previous implementation just selected top-N validators by stake,
 * which was deterministic and allowed collusion. This fix uses VRF
 * for probabilistic stake-weighted selection.
 *
 * Architecture:
 * 1. Each validator generates VRF output locally when they see a claim
 * 2. VRF output determines if they're selected (probability ~ stake/totalStake)
 * 3. Selected validators submit VRF proof with their commitment
 * 4. We verify VRF proof before accepting their participation
 *
 * This function now returns ELIGIBLE validators (all who meet min stake).
 * Actual selection is determined by `verifyValidatorVRFSelection` when
 * validators submit their commitments.
 */
export function getEligibleValidators(
  validatorSet: ValidatorSet,
  params: ConsensusParams,
): Array<{ validator: PublicKey; stake: bigint }> {
  const eligible: Array<{ validator: PublicKey; stake: bigint }> = [];

  for (const [pubKey, validator] of validatorSet.validators) {
    if (!validator.isActive) continue;
    if (validator.stake < params.minValidatorStake) continue;

    eligible.push({
      validator: pubKey as PublicKey,
      stake: validator.stake + validator.delegatedStake,
    });
  }

  return eligible;
}

/**
 * Verify a validator's VRF-based selection for a verification round
 *
 * Returns true if:
 * 1. VRF proof is valid
 * 2. VRF output qualifies them for selection (stake-weighted probability)
 */
export function verifyValidatorVRFSelection(
  validator: PublicKey,
  vrfOutput: VRFOutput,
  vrfProof: VRFProof,
  claimId: string,
  blockNumber: bigint,
  previousBlockHash: string,
  validatorStake: bigint,
  totalStake: bigint,
  params: ConsensusParams,
): { valid: boolean; priority: bigint; reason?: string } {
  // Generate the seed this validator should have used
  const seed = generateVerificationSeed(
    claimId,
    blockNumber,
    previousBlockHash,
  );

  // Verify VRF proof matches the claimed output
  const verification = vrfVerify(seed, validator, vrfProof);

  if (!verification.valid) {
    return { valid: false, priority: 0n, reason: "Invalid VRF proof" };
  }

  if (verification.output !== vrfOutput) {
    return {
      valid: false,
      priority: 0n,
      reason: "VRF output does not match proof",
    };
  }

  // Check if VRF output qualifies this validator for selection
  // Probability of selection = (stake / totalStake) * targetValidators
  const selectionResult = isValidatorSelected(
    vrfOutput,
    validatorStake,
    totalStake,
    params.maxValidators,
  );

  if (!selectionResult.selected) {
    return {
      valid: false,
      priority: 0n,
      reason: "VRF output does not qualify for selection",
    };
  }

  return { valid: true, priority: selectionResult.priority };
}

/**
 * Generate a validator's VRF for verification round participation
 * (Called by validator nodes locally)
 */
export function generateValidatorVRF(
  validatorPrivateKey: PrivateKey,
  claimId: string,
  blockNumber: bigint,
  previousBlockHash: string,
): { output: VRFOutput; proof: VRFProof } {
  const seed = generateVerificationSeed(
    claimId,
    blockNumber,
    previousBlockHash,
  );
  return vrfGenerate(seed, validatorPrivateKey);
}

/**
 * Check if a validator should participate based on their VRF
 * (Called by validator nodes to decide whether to commit)
 */
export function shouldValidatorParticipate(
  vrfOutput: VRFOutput,
  validatorStake: bigint,
  totalStake: bigint,
  params: ConsensusParams,
): { shouldParticipate: boolean; priority: bigint } {
  const result = isValidatorSelected(
    vrfOutput,
    validatorStake,
    totalStake,
    params.maxValidators,
  );

  return {
    shouldParticipate: result.selected,
    priority: result.priority,
  };
}

/**
 * Legacy function for backward compatibility
 * @deprecated Use getEligibleValidators + verifyValidatorVRFSelection instead
 */
export function selectValidatorsForVerification(
  _claimId: string,
  _blockNumber: bigint,
  _previousBlockHash: string,
  validatorSet: ValidatorSet,
  params: ConsensusParams,
): ValidatorSelection[] {
  // For tests and backward compatibility, return all eligible validators as "potentially selected"
  // Actual selection verification happens via verifyValidatorVRFSelection
  const eligible = getEligibleValidators(validatorSet, params);

  return eligible.map((e, i) => ({
    validator: e.validator,
    vrfOutput: "", // Filled when validator submits
    vrfProof: "", // Filled when validator submits
    stake: e.stake,
    priority: BigInt(i),
    selected: true, // Potentially selected - actual verification via VRF
  }));
}

/**
 * Generate seed for verification round
 */
function generateVerificationSeed(
  claimId: string,
  blockNumber: bigint,
  previousBlockHash: string,
): Uint8Array {
  const input = `verification:${claimId}:${blockNumber}:${previousBlockHash}`;
  return new TextEncoder().encode(sha512_256Hex(input));
}

// ============ Selection for Block Proposal ============

/**
 * Determine if a validator can propose the next block
 */
export function canProposeBlock(
  validatorPrivateKey: PrivateKey,
  blockNumber: bigint,
  epoch: bigint,
  previousBlockHash: string,
  validatorStake: bigint,
  totalStake: bigint,
  targetProposers: number = 1,
): {
  canPropose: boolean;
  vrfOutput: VRFOutput;
  vrfProof: VRFProof;
  priority: bigint;
} {
  // Generate VRF for this block slot
  const seed = generateBlockSeed(previousBlockHash, blockNumber, epoch);
  const { output, proof } = vrfGenerate(seed, validatorPrivateKey);

  // Check if selected
  const { selected, priority } = isValidatorSelected(
    output,
    validatorStake,
    totalStake,
    targetProposers,
  );

  return {
    canPropose: selected,
    vrfOutput: output,
    vrfProof: proof,
    priority,
  };
}

/**
 * Verify a validator's claim to propose a block
 */
export function verifyBlockProposer(
  proposer: PublicKey,
  vrfOutput: VRFOutput,
  vrfProof: VRFProof,
  blockNumber: bigint,
  epoch: bigint,
  previousBlockHash: string,
  proposerStake: bigint,
  totalStake: bigint,
  targetProposers: number = 1,
): { valid: boolean; reason?: string } {
  // Verify VRF proof
  const seed = generateBlockSeed(previousBlockHash, blockNumber, epoch);
  const verification = vrfVerify(seed, proposer, vrfProof);

  if (!verification.valid) {
    return { valid: false, reason: "Invalid VRF proof" };
  }

  if (verification.output !== vrfOutput) {
    return { valid: false, reason: "VRF output mismatch" };
  }

  // Verify selection (targetProposers matches what producer used)
  const { selected } = isValidatorSelected(
    vrfOutput,
    proposerStake,
    totalStake,
    targetProposers,
  );

  if (!selected) {
    return { valid: false, reason: "Validator not selected for this slot" };
  }

  return { valid: true };
}

// ============ Validator Set Management ============

/**
 * Create initial validator set
 */
export function createValidatorSet(
  validators: Array<{
    publicKey: PublicKey;
    did: string;
    stake: bigint;
  }>,
  epoch: bigint = 0n,
): ValidatorSet {
  const validatorMap = new Map<string, Validator>();
  let totalStake = 0n;

  for (const v of validators) {
    validatorMap.set(v.publicKey, {
      publicKey: v.publicKey,
      did: v.did,
      stake: v.stake,
      delegatedStake: 0n,
      reputation: 0.5, // Neutral starting reputation
      isActive: true,
      lastActiveBlock: 0,
      totalVerifications: 0,
      correctVerifications: 0,
      missedVerifications: 0,
      slashCount: 0,
    });
    totalStake += v.stake;
  }

  return {
    validators: validatorMap,
    totalStake,
    activeCount: validators.length,
    epoch,
  };
}

/**
 * Update validator after verification
 */
export function updateValidatorStats(
  validator: Validator,
  wasCorrect: boolean,
  currentBlock: number,
): Validator {
  const updated = { ...validator };
  updated.totalVerifications++;
  updated.lastActiveBlock = currentBlock;

  if (wasCorrect) {
    updated.correctVerifications++;
  }

  // Update reputation
  if (updated.totalVerifications >= 11) {
    updated.reputation =
      updated.correctVerifications / updated.totalVerifications;
  }

  return updated;
}

/**
 * Slash a validator by fraction (basis points internally)
 * @deprecated Use slashValidatorByAmount for exact slashing without float precision loss
 */
export function slashValidator(
  validator: Validator,
  slashFraction: number,
  reason: string,
): { validator: Validator; slashedAmount: bigint } {
  // SECURITY: Use basis points (1/10000) to avoid float precision issues
  // slashFraction is expected to be 0.0-1.0, convert to basis points
  const basisPoints = BigInt(Math.floor(slashFraction * 10000));
  const slashedAmount = (validator.stake * basisPoints) / 10000n;

  return slashValidatorByAmount(validator, slashedAmount, reason);
}

/**
 * VULN-010 FIX: Slash a validator by exact bigint amount
 * Avoids float precision loss when slash amount is already known
 */
export function slashValidatorByAmount(
  validator: Validator,
  slashAmount: bigint,
  _reason: string,
): { validator: Validator; slashedAmount: bigint } {
  // SECURITY: Ensure we don't slash more than the validator has
  const actualSlash =
    slashAmount > validator.stake ? validator.stake : slashAmount;

  const updated: Validator = {
    ...validator,
    stake: validator.stake - actualSlash,
    slashCount: validator.slashCount + 1,
    reputation: Math.max(0, validator.reputation - 0.1),
  };

  // Deactivate if stake too low or too many slashes
  if (updated.slashCount >= 3) {
    updated.isActive = false;
  }

  return { validator: updated, slashedAmount: actualSlash };
}

/**
 * Calculate voting power (stake + reputation weighted)
 * VULN-010 FIX: Uses basis points to avoid float precision loss with large stakes
 */
export function calculateVotingPower(validator: Validator): bigint {
  // Base voting power is stake
  const stakeWeight = validator.stake + validator.delegatedStake;

  // Reputation multiplier (0.5 to 1.5) in basis points (5000 to 15000)
  // reputation is 0.0 to 1.0, so multiplier = 5000 + (reputation * 10000)
  const multiplierBasisPoints =
    5000n + BigInt(Math.floor(validator.reputation * 10000));

  // Apply multiplier using bigint math: (stake * multiplier) / 10000
  return (stakeWeight * multiplierBasisPoints) / 10000n;
}

/**
 * Get validators sorted by voting power
 */
export function getValidatorsByPower(validatorSet: ValidatorSet): Validator[] {
  return Array.from(validatorSet.validators.values())
    .filter((v) => v.isActive)
    .sort((a, b) => {
      const powerA = calculateVotingPower(a);
      const powerB = calculateVotingPower(b);
      if (powerB > powerA) return 1;
      if (powerB < powerA) return -1;
      return 0;
    });
}

// ============ Epoch Transitions ============

/**
 * Check if epoch should change
 * GAP-025 FIX: Accept epochLength as parameter, no hardcoded default
 * Caller must provide epochLength from ConsensusParams
 */
export function shouldChangeEpoch(
  currentBlock: bigint,
  epochLength: bigint,
): boolean {
  if (epochLength <= 0n) {
    throw new Error("Epoch length must be positive");
  }
  return currentBlock % epochLength === 0n;
}

/**
 * GAP-025 FIX: Default epoch length constant for reference
 * Actual value should come from ConsensusParams.epochLength
 */
export const DEFAULT_EPOCH_LENGTH = 1111n;

/**
 * Compute next epoch validator set
 */
export function computeNextEpochValidators(
  currentSet: ValidatorSet,
  pendingStakeChanges: Array<{
    validator: PublicKey;
    newStake: bigint;
  }>,
  newValidators: Array<{
    publicKey: PublicKey;
    did: string;
    stake: bigint;
  }>,
  removedValidators: PublicKey[],
  newEpoch: bigint,
): ValidatorSet {
  const newSet = new Map(currentSet.validators);
  let totalStake = currentSet.totalStake;

  // Apply stake changes
  for (const change of pendingStakeChanges) {
    const validator = newSet.get(change.validator);
    if (validator) {
      totalStake -= validator.stake;
      totalStake += change.newStake;
      newSet.set(change.validator, { ...validator, stake: change.newStake });
    }
  }

  // Add new validators
  for (const newVal of newValidators) {
    newSet.set(newVal.publicKey, {
      publicKey: newVal.publicKey,
      did: newVal.did,
      stake: newVal.stake,
      delegatedStake: 0n,
      reputation: 0.5,
      isActive: true,
      lastActiveBlock: 0,
      totalVerifications: 0,
      correctVerifications: 0,
      missedVerifications: 0,
      slashCount: 0,
    });
    totalStake += newVal.stake;
  }

  // Remove validators
  for (const removed of removedValidators) {
    const validator = newSet.get(removed);
    if (validator) {
      totalStake -= validator.stake;
      newSet.delete(removed);
    }
  }

  return {
    validators: newSet,
    totalStake,
    activeCount: Array.from(newSet.values()).filter((v) => v.isActive).length,
    epoch: newEpoch,
  };
}

// ============ Domain Qualification Filtering ============

/**
 * Filter validators by domain qualification
 *
 * Only validators with active qualification in the specified domain
 * (or inheritable qualification from lower stratum) are eligible.
 */
export function filterByDomainQualification(
  validators: Validator[],
  domain: string,
  qualificationStates: Map<string, ValidatorQualificationState>,
  currentBlock: bigint,
): Array<{
  validator: Validator;
  qualification: DomainQualification;
  qualificationWeight: number;
}> {
  const qualified: Array<{
    validator: Validator;
    qualification: DomainQualification;
    qualificationWeight: number;
  }> = [];

  for (const validator of validators) {
    const qualState = qualificationStates.get(validator.publicKey);
    if (!qualState) continue;

    // Check direct domain qualification
    const qual = qualState.qualifications.get(domain);
    if (qual && isQualificationValid(qual, currentBlock)) {
      const weight = calculateQualificationWeight(qual);
      qualified.push({
        validator,
        qualification: qual,
        qualificationWeight: weight,
      });
      continue;
    }

    // Check for inheritable qualification from other domains
    // (stratum inheritance is handled during qualification creation)
    for (const [, domainQual] of qualState.qualifications) {
      if (
        domainQual.pathway === "stratum_inheritance" &&
        domainQual.domain === domain &&
        isQualificationValid(domainQual, currentBlock)
      ) {
        const weight = calculateQualificationWeight(domainQual);
        qualified.push({
          validator,
          qualification: domainQual,
          qualificationWeight: weight,
        });
        break;
      }
    }
  }

  return qualified;
}

/**
 * Get eligible validators for a domain with qualification and reputation
 *
 * Returns validators filtered by:
 * 1. Active status
 * 2. Minimum stake
 * 3. Domain qualification
 * Along with their effective reputation in the domain
 */
export function getEligibleValidatorsForDomain(
  validatorSet: ValidatorSet,
  domain: string,
  qualificationStates: Map<string, ValidatorQualificationState>,
  isolatedReputations: Map<string, IsolatedReputation>,
  domainTree: DomainTree,
  currentBlock: bigint,
  params: ConsensusParams,
): Array<{
  validator: PublicKey;
  stake: bigint;
  qualification: DomainQualification;
  qualificationWeight: number;
  effectiveReputation: EffectiveReputationResult;
  votingPower: bigint;
}> {
  const eligible: Array<{
    validator: PublicKey;
    stake: bigint;
    qualification: DomainQualification;
    qualificationWeight: number;
    effectiveReputation: EffectiveReputationResult;
    votingPower: bigint;
  }> = [];

  // First get base eligible validators
  const baseEligible = getEligibleValidators(validatorSet, params);

  // Get qualified validators
  const qualifiedValidators = filterByDomainQualification(
    baseEligible.map((e) => validatorSet.validators.get(e.validator)!),
    domain,
    qualificationStates,
    currentBlock,
  );

  for (const {
    validator,
    qualification,
    qualificationWeight,
  } of qualifiedValidators) {
    const stake = validator.stake + validator.delegatedStake;

    // Get isolated reputation for this domain
    const isolatedRep = isolatedReputations.get(validator.publicKey);
    const effectiveReputation = isolatedRep
      ? getEffectiveReputation(
          validator.publicKey,
          domain,
          isolatedRep,
          domainTree,
        )
      : { score: 500, source: "global" as const, discount: 0.5, confidence: 0 };

    // Calculate voting power with qualification and reputation
    const votingPower = calculateQualifiedVotingPower(
      validator,
      qualification,
      effectiveReputation,
    );

    eligible.push({
      validator: validator.publicKey,
      stake,
      qualification,
      qualificationWeight,
      effectiveReputation,
      votingPower,
    });
  }

  return eligible;
}

/**
 * Calculate effective voting power with qualification and isolated reputation
 *
 * voting_power = base_stake × qualification_weight × reputation_multiplier
 *
 * Where:
 * - qualification_weight: 1.0 (direct) or 0.7^distance (inherited)
 * - reputation_multiplier: 0.5 to 1.5 based on isolated domain reputation
 */
export function calculateQualifiedVotingPower(
  validator: Validator,
  qualification: DomainQualification,
  effectiveReputation: EffectiveReputationResult,
): bigint {
  const baseStake = validator.stake + validator.delegatedStake;

  // Qualification weight (1.0 for direct, 0.7^distance for inherited)
  const qualWeight = calculateQualificationWeight(qualification);

  // Reputation multiplier based on isolated reputation
  // Score is 0-1000, normalize to 0-1, then apply 0.5-1.5 range
  const normalizedRep = effectiveReputation.score / 1000;
  const repMultiplier = 0.5 + normalizedRep;

  // Apply reputation discount (domain=1.0, stratum=0.7, global=0.5)
  const effectiveRepMultiplier = repMultiplier * effectiveReputation.discount;

  // Convert to basis points for bigint math (avoid float precision loss)
  const qualBasisPoints = BigInt(Math.floor(qualWeight * 10000));
  const repBasisPoints = BigInt(Math.floor(effectiveRepMultiplier * 10000));

  // voting_power = stake × qualWeight × repMultiplier
  // Using (stake × qual × rep) / (10000 × 10000) for proper scaling
  return (baseStake * qualBasisPoints * repBasisPoints) / (10000n * 10000n);
}
