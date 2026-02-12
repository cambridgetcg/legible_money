/**
 * Unit Tests for Verification Round
 *
 * Tests the core verification round mechanics:
 *   - Round creation and lifecycle
 *   - Commit-reveal cryptographic protocol
 *   - Result aggregation
 *   - Phase transitions
 *
 * These tests are fast and don't require network setup.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  createVerificationRound,
  startRound,
  addCommitmentWithVRF,
  transitionToReveal,
  addReveal,
  aggregateResults,
  getCurrentPhase,
  hasQuorum,
  getRoundSummary,
  generateCommitment,
  generateReveal,
  generateClaimId,
} from "../../../consensus/pot/verification_round.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../../consensus/pot/types.js";
import type {
  ConsensusParams,
  ValidatorSelection,
  VerificationRound,
} from "../../../consensus/pot/types.js";
import type {
  ClaimSubmission,
  Reference,
} from "../../../core/knowledge/types.js";
import {
  generateKeypairFromSeed,
  type PublicKey,
  type PrivateKey,
} from "../../../core/crypto/keys.js";
import { domainHash } from "../../../core/crypto/hashing.js";
import { vrfGenerate } from "../../../core/crypto/vrf.js";

// ============ Test Fixtures ============

interface TestValidator {
  publicKey: PublicKey;
  privateKey: PrivateKey;
  stake: bigint;
}

function createTestValidators(count: number): TestValidator[] {
  const validators: TestValidator[] = [];
  for (let i = 0; i < count; i++) {
    const seed = domainHash("LGM.test.validator.v1", `test-validator-${i}`);
    const { publicKey, privateKey } = generateKeypairFromSeed(
      seed.slice(0, 32),
    );
    validators.push({
      publicKey,
      privateKey,
      stake: 10000n * 10n ** 15n,
    });
  }
  return validators;
}

function createTestClaim(): ClaimSubmission {
  return {
    id: "",
    claim: "Test claim for unit testing",
    domain: "test.unit",
    submitter: "test-submitter",
    stake: 100n * 10n ** 15n,
    submittedAt: Date.now(),
    references: [
      {
        id: "ref-1",
        type: "supports",
        source: "Test source",
        verified: false,
      } as Reference,
    ],
  };
}

function createTestValidatorSelections(
  validators: TestValidator[],
): ValidatorSelection[] {
  return validators.map((v, i) => ({
    validator: v.publicKey,
    vrfOutput: `vrf-output-${i}`,
    vrfProof: `vrf-proof-${i}`,
    stake: v.stake,
    priority: BigInt(i),
    selected: true,
  }));
}

const TEST_PARAMS: ConsensusParams = {
  ...DEFAULT_CONSENSUS_PARAMS,
  minValidators: 3,
  maxValidators: 5,
  validatorQuorum: 0.66,
  acceptanceThreshold: 0.66,
  rejectionThreshold: 0.66,
  verificationRoundBlocks: 10,
  commitPhaseBlocks: 3,
  revealPhaseBlocks: 3,
  verificationReward: 1n * 10n ** 18n,
  wrongVerificationSlash: 0.06,
  missedVerificationSlash: 0.0001,
};

// ============ Round Creation Tests ============

describe("Verification Round - Creation", () => {
  it("should create round with correct timing", () => {
    const claim = createTestClaim();
    const validators = createTestValidators(5);
    const selections = createTestValidatorSelections(validators);
    const startBlock = 100;

    const round = createVerificationRound(
      claim,
      selections,
      startBlock,
      TEST_PARAMS,
    );

    expect(round.startBlock).toBe(startBlock);
    expect(round.commitDeadline).toBe(
      startBlock + TEST_PARAMS.commitPhaseBlocks,
    );
    expect(round.revealDeadline).toBe(
      startBlock +
        TEST_PARAMS.commitPhaseBlocks +
        TEST_PARAMS.revealPhaseBlocks,
    );
    expect(round.completionBlock).toBe(
      startBlock + TEST_PARAMS.verificationRoundBlocks,
    );
  });

  it("should create round in pending phase", () => {
    const claim = createTestClaim();
    const validators = createTestValidators(3);
    const selections = createTestValidatorSelections(validators);

    const round = createVerificationRound(claim, selections, 100, TEST_PARAMS);

    expect(round.phase).toBe("pending");
  });

  it("should calculate required validators correctly", () => {
    const claim = createTestClaim();
    const validators = createTestValidators(5);
    const selections = createTestValidatorSelections(validators);

    const round = createVerificationRound(claim, selections, 100, TEST_PARAMS);

    // 5 validators * 0.66 quorum = 3.3, ceil = 4, but min is 3
    expect(round.requiredValidators).toBe(
      Math.max(
        TEST_PARAMS.minValidators,
        Math.ceil(selections.length * TEST_PARAMS.validatorQuorum),
      ),
    );
  });

  it("should initialize empty commitments and reveals", () => {
    const claim = createTestClaim();
    const validators = createTestValidators(3);
    const selections = createTestValidatorSelections(validators);

    const round = createVerificationRound(claim, selections, 100, TEST_PARAMS);

    expect(round.commitments.size).toBe(0);
    expect(round.reveals.size).toBe(0);
  });

  it("should generate unique claim IDs for same content at different blocks", () => {
    const claim = createTestClaim();
    const validators = createTestValidators(3);
    const selections = createTestValidatorSelections(validators);

    const round1 = createVerificationRound(
      claim,
      selections,
      100,
      TEST_PARAMS,
      "submitter-a",
    );
    const round2 = createVerificationRound(
      claim,
      selections,
      200,
      TEST_PARAMS,
      "submitter-a",
    );
    const round3 = createVerificationRound(
      claim,
      selections,
      100,
      TEST_PARAMS,
      "submitter-b",
    );

    // Different blocks = different claim IDs
    expect(round1.claimId).not.toBe(round2.claimId);
    // Different submitters = different claim IDs
    expect(round1.claimId).not.toBe(round3.claimId);
  });
});

// ============ Phase Transition Tests ============

describe("Verification Round - Phase Transitions", () => {
  it("should transition from pending to commit via startRound", () => {
    const claim = createTestClaim();
    const validators = createTestValidators(3);
    const selections = createTestValidatorSelections(validators);

    let round = createVerificationRound(claim, selections, 100, TEST_PARAMS);
    expect(round.phase).toBe("pending");

    round = startRound(round);
    expect(round.phase).toBe("commit");
  });

  it("should determine phase based on block number", () => {
    const claim = createTestClaim();
    const validators = createTestValidators(3);
    const selections = createTestValidatorSelections(validators);

    const round = startRound(
      createVerificationRound(claim, selections, 100, TEST_PARAMS),
    );

    // Before start
    expect(getCurrentPhase(round, 99)).toBe("pending");

    // In commit phase
    expect(getCurrentPhase(round, 100)).toBe("commit");
    expect(getCurrentPhase(round, 102)).toBe("commit");
    expect(getCurrentPhase(round, 103)).toBe("commit");

    // In reveal phase
    expect(getCurrentPhase(round, 104)).toBe("reveal");
    expect(getCurrentPhase(round, 106)).toBe("reveal");

    // In aggregation
    expect(getCurrentPhase(round, 107)).toBe("aggregation");

    // Complete
    expect(getCurrentPhase(round, 111)).toBe("complete");
  });

  it("should transition to reveal phase", () => {
    const claim = createTestClaim();
    const validators = createTestValidators(3);
    const selections = createTestValidatorSelections(validators);

    let round = startRound(
      createVerificationRound(claim, selections, 100, TEST_PARAMS),
    );

    // Before deadline - no transition
    round = transitionToReveal(round, 100);
    expect(round.phase).toBe("commit");

    // After deadline - transition
    round = transitionToReveal(round, 104);
    expect(round.phase).toBe("reveal");
  });
});

// ============ Commitment Tests ============

describe("Verification Round - Commitments", () => {
  let validators: TestValidator[];
  let selections: ValidatorSelection[];
  let round: VerificationRound;

  beforeEach(() => {
    validators = createTestValidators(5);
    selections = createTestValidatorSelections(validators);
    round = startRound(
      createVerificationRound(createTestClaim(), selections, 100, TEST_PARAMS),
    );
  });

  it("should generate valid commitment", () => {
    const validator = validators[0];
    const { commitment, salt } = generateCommitment(
      validator.privateKey,
      validator.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    expect(commitment.validator).toBe(validator.publicKey);
    expect(commitment.commitment).toBeTruthy();
    expect(commitment.signature).toBeTruthy();
    expect(salt).toBeTruthy();
    expect(salt.length).toBe(64); // 32 bytes hex
  });

  it("should require VRF proof for commitment", () => {
    const validator = validators[0];
    const { commitment } = generateCommitment(
      validator.privateKey,
      validator.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    // Without VRF proof, should fail
    const result = addCommitmentWithVRF(
      round,
      {
        ...commitment,
        vrfOutput: undefined,
        vrfProof: undefined,
      },
      100,
      "previous-block-hash",
      validator.stake,
      50000n * 10n ** 18n,
      TEST_PARAMS,
    );

    expect(result.accepted).toBe(false);
    expect(result.reason).toMatch(/VRF/i);
  });

  it("should reject commitment after deadline", () => {
    const validator = validators[0];
    const { commitment } = generateCommitment(
      validator.privateKey,
      validator.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    const vrfResult = vrfGenerate(
      `${round.claimId}:100:prev-hash`,
      validator.privateKey,
    );

    // Commitment after deadline (block 104+)
    const result = addCommitmentWithVRF(
      round,
      {
        ...commitment,
        vrfOutput: vrfResult.output,
        vrfProof: vrfResult.proof,
      },
      105, // After commit deadline
      "previous-block-hash",
      validator.stake,
      50000n * 10n ** 18n,
      TEST_PARAMS,
    );

    expect(result.accepted).toBe(false);
    expect(result.reason).toMatch(/deadline/i);
  });

  it("should reject commitment in wrong phase", () => {
    // Transition to reveal phase
    let revealRound = transitionToReveal(round, 104);
    revealRound = { ...revealRound, phase: "reveal" };

    const validator = validators[0];
    const { commitment } = generateCommitment(
      validator.privateKey,
      validator.publicKey,
      revealRound.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    const vrfResult = vrfGenerate(
      `${revealRound.claimId}:100:prev-hash`,
      validator.privateKey,
    );

    const result = addCommitmentWithVRF(
      revealRound,
      {
        ...commitment,
        vrfOutput: vrfResult.output,
        vrfProof: vrfResult.proof,
      },
      104,
      "previous-block-hash",
      validator.stake,
      50000n * 10n ** 18n,
      TEST_PARAMS,
    );

    expect(result.accepted).toBe(false);
    expect(result.reason).toMatch(/phase/i);
  });
});

// ============ Reveal Tests ============

describe("Verification Round - Reveals", () => {
  let validators: TestValidator[];
  let selections: ValidatorSelection[];
  let round: VerificationRound;

  beforeEach(() => {
    validators = createTestValidators(5);
    selections = createTestValidatorSelections(validators);
    round = startRound(
      createVerificationRound(createTestClaim(), selections, 100, TEST_PARAMS),
    );
  });

  it("should generate valid reveal", () => {
    const validator = validators[0];
    const reveal = generateReveal(
      validator.privateKey,
      validator.publicKey,
      round.id,
      "confirmed",
      0.9,
      "test-nonce",
      ["ref-1"],
      "Test reasoning",
    );

    expect(reveal.validator).toBe(validator.publicKey);
    expect(reveal.verdict).toBe("confirmed");
    expect(reveal.confidence).toBe(0.9);
    expect(reveal.salt).toBe("test-nonce");
    expect(reveal.reasoning).toBe("Test reasoning");
    expect(reveal.signature).toBeTruthy();
  });

  it("should reject reveal without commitment", () => {
    // Transition to reveal phase without any commitments
    const revealRound = transitionToReveal(round, 104);

    const validator = validators[0];
    const reveal = generateReveal(
      validator.privateKey,
      validator.publicKey,
      revealRound.id,
      "confirmed",
      0.9,
      "test-nonce",
      ["ref-1"],
    );

    const result = addReveal(revealRound, reveal, 104);

    expect(result.accepted).toBe(false);
    expect(result.reason).toMatch(/commitment/i);
  });

  it("should reject reveal that does not match commitment", () => {
    const validator = validators[0];

    // Create commitment with 'confirmed' verdict
    const { commitment, salt } = generateCommitment(
      validator.privateKey,
      validator.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    // Add commitment to round manually (bypassing VRF for unit test)
    round.commitments.set(validator.publicKey, commitment);

    // Transition to reveal
    const revealRound = transitionToReveal(round, 104);

    // Create reveal with 'rejected' verdict - mismatch!
    const reveal = generateReveal(
      validator.privateKey,
      validator.publicKey,
      revealRound.id,
      "rejected", // Different from commitment
      0.9,
      salt,
      ["ref-1"],
    );

    const result = addReveal(revealRound, reveal, 104);

    expect(result.accepted).toBe(false);
    expect(result.reason).toMatch(/match/i);
  });

  it("should accept valid reveal that matches commitment", () => {
    const validator = validators[0];

    // Create commitment
    const { commitment, salt } = generateCommitment(
      validator.privateKey,
      validator.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );

    // Add commitment to round
    round.commitments.set(validator.publicKey, commitment);

    // Transition to reveal
    const revealRound = transitionToReveal(round, 104);

    // Create matching reveal
    const reveal = generateReveal(
      validator.privateKey,
      validator.publicKey,
      revealRound.id,
      "confirmed", // Same as commitment
      0.9,
      salt,
      ["ref-1"],
    );

    const result = addReveal(revealRound, reveal, 104);

    expect(result.accepted).toBe(true);
    expect(result.round.reveals.size).toBe(1);
  });

  it("should reject reveal after deadline", () => {
    const validator = validators[0];

    // Create and add commitment
    const { commitment, salt } = generateCommitment(
      validator.privateKey,
      validator.publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );
    round.commitments.set(validator.publicKey, commitment);

    // Transition to reveal
    const revealRound = transitionToReveal(round, 104);

    // Create reveal
    const reveal = generateReveal(
      validator.privateKey,
      validator.publicKey,
      revealRound.id,
      "confirmed",
      0.9,
      salt,
      ["ref-1"],
    );

    // Submit after deadline
    const result = addReveal(revealRound, reveal, 108); // After reveal deadline

    expect(result.accepted).toBe(false);
    expect(result.reason).toMatch(/deadline/i);
  });
});

// ============ Aggregation Tests ============

describe("Verification Round - Aggregation", () => {
  let validators: TestValidator[];
  let validatorStakes: Map<string, { stake: bigint; reputation: number }>;

  beforeEach(() => {
    validators = createTestValidators(5);
    validatorStakes = new Map(
      validators.map((v) => [v.publicKey, { stake: v.stake, reputation: 0.5 }]),
    );
  });

  function setupRoundWithReveals(
    verdicts: Array<"confirmed" | "rejected" | "uncertain">,
    confidences: number[],
  ): VerificationRound {
    const selections = createTestValidatorSelections(
      validators.slice(0, verdicts.length),
    );
    let round = startRound(
      createVerificationRound(createTestClaim(), selections, 100, TEST_PARAMS),
    );

    // Add commitments and reveals for each validator
    for (let i = 0; i < verdicts.length; i++) {
      const validator = validators[i];
      const { commitment, salt } = generateCommitment(
        validator.privateKey,
        validator.publicKey,
        round.id,
        verdicts[i],
        confidences[i],
        ["ref-1"],
      );

      round.commitments.set(validator.publicKey, commitment);

      const reveal = generateReveal(
        validator.privateKey,
        validator.publicKey,
        round.id,
        verdicts[i],
        confidences[i],
        salt,
        ["ref-1"],
      );

      round.reveals.set(validator.publicKey, reveal);
    }

    // Transition to reveal phase
    round = transitionToReveal(round, 104);

    return round;
  }

  it("should accept when validators confirm with quorum", () => {
    const round = setupRoundWithReveals(
      ["confirmed", "confirmed", "confirmed", "confirmed", "rejected"],
      [0.9, 0.85, 0.95, 0.8, 0.1],
    );

    const result = aggregateResults(round, validatorStakes, 107, TEST_PARAMS);

    expect(result.result).toBeDefined();
    expect(result.result!.outcome).toBe("accepted");
    expect(result.result!.finalConfidence).toBeGreaterThan(0.5);
  });

  it("should reject when validators reject with quorum", () => {
    const round = setupRoundWithReveals(
      ["rejected", "rejected", "rejected", "rejected", "confirmed"],
      [0.1, 0.15, 0.05, 0.2, 0.9],
    );

    const result = aggregateResults(round, validatorStakes, 107, TEST_PARAMS);

    expect(result.result).toBeDefined();
    expect(result.result!.outcome).toBe("rejected");
    expect(result.result!.finalConfidence).toBe(0);
  });

  it("should mark contested when no clear consensus", () => {
    const round = setupRoundWithReveals(
      ["confirmed", "rejected", "uncertain", "confirmed", "rejected"],
      [0.9, 0.8, 0.5, 0.85, 0.75],
    );

    const result = aggregateResults(round, validatorStakes, 107, TEST_PARAMS);

    expect(result.result).toBeDefined();
    expect(result.result!.outcome).toBe("contested");
  });

  it("should mark insufficient when participation is too low", () => {
    // Only 2 validators participate out of 5
    const selections = createTestValidatorSelections(validators);
    let round = startRound(
      createVerificationRound(createTestClaim(), selections, 100, TEST_PARAMS),
    );

    // Add only 2 reveals (below 66% quorum)
    for (let i = 0; i < 2; i++) {
      const validator = validators[i];
      const { commitment, salt } = generateCommitment(
        validator.privateKey,
        validator.publicKey,
        round.id,
        "confirmed",
        0.9,
        ["ref-1"],
      );
      round.commitments.set(validator.publicKey, commitment);

      const reveal = generateReveal(
        validator.privateKey,
        validator.publicKey,
        round.id,
        "confirmed",
        0.9,
        salt,
        ["ref-1"],
      );
      round.reveals.set(validator.publicKey, reveal);
    }

    round = transitionToReveal(round, 104);

    const result = aggregateResults(round, validatorStakes, 107, TEST_PARAMS);

    expect(result.result).toBeDefined();
    expect(result.result!.outcome).toBe("insufficient");
  });

  it("should calculate confidence breakdown correctly", () => {
    const round = setupRoundWithReveals(
      ["confirmed", "confirmed", "rejected", "uncertain", "confirmed"],
      [0.9, 0.85, 0.1, 0.5, 0.8],
    );

    const result = aggregateResults(round, validatorStakes, 107, TEST_PARAMS);

    expect(result.result).toBeDefined();
    expect(result.result!.confidenceBreakdown.confirmed).toBeCloseTo(0.6, 1);
    expect(result.result!.confidenceBreakdown.rejected).toBeCloseTo(0.2, 1);
    expect(result.result!.confidenceBreakdown.uncertain).toBeCloseTo(0.2, 1);
  });

  it("should track participating validators", () => {
    const round = setupRoundWithReveals(
      ["confirmed", "confirmed", "confirmed"],
      [0.9, 0.85, 0.8],
    );

    const result = aggregateResults(round, validatorStakes, 107, TEST_PARAMS);

    expect(result.result!.totalValidators).toBe(3);
    expect(result.result!.participatingValidators).toBe(3);
  });

  it("should slash validators who committed but did not reveal", () => {
    const selections = createTestValidatorSelections(validators.slice(0, 3));
    let round = startRound(
      createVerificationRound(createTestClaim(), selections, 100, TEST_PARAMS),
    );

    // Add 3 commitments
    for (let i = 0; i < 3; i++) {
      const validator = validators[i];
      const { commitment, salt } = generateCommitment(
        validator.privateKey,
        validator.publicKey,
        round.id,
        "confirmed",
        0.9,
        ["ref-1"],
      );
      round.commitments.set(validator.publicKey, commitment);

      // Only add reveal for first 2
      if (i < 2) {
        const reveal = generateReveal(
          validator.privateKey,
          validator.publicKey,
          round.id,
          "confirmed",
          0.9,
          salt,
          ["ref-1"],
        );
        round.reveals.set(validator.publicKey, reveal);
      }
    }

    round = transitionToReveal(round, 104);

    const result = aggregateResults(round, validatorStakes, 107, TEST_PARAMS);

    // Third validator should be slashed
    expect(result.result!.slashes.size).toBe(1);
    expect(result.result!.slashes.has(validators[2].publicKey)).toBe(true);
  });
});

// ============ Utility Tests ============

describe("Verification Round - Utilities", () => {
  it("should check quorum correctly", () => {
    const validators = createTestValidators(5);
    const selections = createTestValidatorSelections(validators);
    const round = startRound(
      createVerificationRound(createTestClaim(), selections, 100, TEST_PARAMS),
    );

    // No commitments yet
    expect(hasQuorum(round, TEST_PARAMS)).toBe(false);

    // Add commitments until quorum
    for (let i = 0; i < 4; i++) {
      const validator = validators[i];
      const { commitment } = generateCommitment(
        validator.privateKey,
        validator.publicKey,
        round.id,
        "confirmed",
        0.9,
        ["ref-1"],
      );
      round.commitments.set(validator.publicKey, commitment);
    }

    // Now has quorum
    expect(hasQuorum(round, TEST_PARAMS)).toBe(true);
  });

  it("should generate round summary", () => {
    const validators = createTestValidators(3);
    const selections = createTestValidatorSelections(validators);
    const round = startRound(
      createVerificationRound(createTestClaim(), selections, 100, TEST_PARAMS),
    );

    // Add some commitments
    const { commitment } = generateCommitment(
      validators[0].privateKey,
      validators[0].publicKey,
      round.id,
      "confirmed",
      0.9,
      ["ref-1"],
    );
    round.commitments.set(validators[0].publicKey, commitment);

    const summary = getRoundSummary(round);

    expect(summary.id).toBe(round.id);
    expect(summary.claimId).toBe(round.claimId);
    expect(summary.phase).toBe("commit");
    expect(summary.commitments).toBe(1);
    expect(summary.reveals).toBe(0);
    expect(summary.required).toBe(round.requiredValidators);
    expect(summary.hasResult).toBe(false);
  });

  it("should generate unique claim IDs", () => {
    const claim = createTestClaim();

    const id1 = generateClaimId(claim, 100, "submitter-1");
    const id2 = generateClaimId(claim, 100, "submitter-2");
    const id3 = generateClaimId(claim, 101, "submitter-1");

    // All IDs should be unique
    expect(id1).not.toBe(id2);
    expect(id1).not.toBe(id3);
    expect(id2).not.toBe(id3);

    // Same inputs should give same ID
    const id1Repeat = generateClaimId(claim, 100, "submitter-1");
    expect(id1).toBe(id1Repeat);
  });
});
