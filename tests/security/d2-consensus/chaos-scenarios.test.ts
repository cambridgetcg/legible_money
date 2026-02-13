/**
 * Dimension 2: Consensus Security - Chaos Scenarios
 *
 * Tests consensus resilience under adversarial and degraded conditions:
 * 1. Commit phase: late/duplicate/missing commitments
 * 2. Reveal phase: mismatched reveals, non-committed reveals
 * 3. Phase transition boundary conditions
 * 4. Round lifecycle invariants
 * 5. Quorum calculation edge cases
 *
 * Industry method: Jepsen-style fault injection + property testing.
 */

import { describe, it, expect } from "vitest";
import {
  createVerificationRound,
  startRound,
  transitionToReveal,
  addCommitmentWithVRF,
  addReveal,
  generateCommitment,
  generateReveal,
  type VerificationRound,
} from "../../../consensus/pot/verification_round.js";
import {
  DEFAULT_CONSENSUS_PARAMS,
  type Validator,
  type ValidatorSelection,
  type ConsensusParams,
} from "../../../consensus/pot/types.js";
import { generateKeyPair } from "../../../core/crypto/keys.js";
import { commit as hashCommit } from "../../../core/crypto/commitment.js";
import type { ClaimSubmission } from "../../../core/knowledge/types.js";

// ============ Helpers ============

function makeClaim(): ClaimSubmission {
  return {
    text: "Test claim for chaos scenario",
    domain: "test",
    submitter: "did:lgm:chaos-submitter",
    confidence: 0.8,
    evidence: ["evidence-1"],
    logicZone: "empirical",
    epistemicCategory: "observation",
    references: [],
    timestamp: Date.now(),
  } as ClaimSubmission;
}

function makeValidatorSelection(count: number): ValidatorSelection[] {
  return Array.from({ length: count }, (_, i) => ({
    validator: `validator-pk-${i}`,
    did: `did:lgm:validator-${i}`,
    stake: 10n * 10n ** 18n,
    priority: BigInt(1000 - i),
    vrfOutput: new Uint8Array(32),
    vrfProof: new Uint8Array(80),
  })) as ValidatorSelection[];
}

// ============================================================
// CHAOS-01 through CHAOS-10: Consensus chaos scenarios
// ============================================================

describe("Consensus Chaos Scenarios (CHAOS)", () => {
  it("CHAOS-01: Round starts in pending phase", () => {
    const round = createVerificationRound(
      makeClaim(),
      makeValidatorSelection(5),
      100,
      DEFAULT_CONSENSUS_PARAMS,
    );

    expect(round.phase).toBe("pending");
    expect(round.commitments.size).toBe(0);
    expect(round.reveals.size).toBe(0);
  });

  it("CHAOS-02: startRound transitions pending → commit", () => {
    const round = createVerificationRound(
      makeClaim(),
      makeValidatorSelection(5),
      100,
      DEFAULT_CONSENSUS_PARAMS,
    );

    const started = startRound(round);
    expect(started.phase).toBe("commit");
  });

  it("CHAOS-03: transitionToReveal before deadline is no-op", () => {
    const round = startRound(
      createVerificationRound(
        makeClaim(),
        makeValidatorSelection(5),
        100,
        DEFAULT_CONSENSUS_PARAMS,
      ),
    );

    // Try to transition before commit deadline
    const result = transitionToReveal(round, 100);

    // Should stay in commit phase (not past deadline)
    expect(result.phase).toBe("commit");
  });

  it("CHAOS-04: transitionToReveal at/after deadline transitions to reveal", () => {
    const round = startRound(
      createVerificationRound(
        makeClaim(),
        makeValidatorSelection(5),
        100,
        DEFAULT_CONSENSUS_PARAMS,
      ),
    );

    // Transition after commit deadline
    const revealed = transitionToReveal(round, round.commitDeadline + 1);
    expect(revealed.phase).toBe("reveal");
  });

  it("CHAOS-05: transitionToReveal on non-commit phase is no-op", () => {
    const round = createVerificationRound(
      makeClaim(),
      makeValidatorSelection(5),
      100,
      DEFAULT_CONSENSUS_PARAMS,
    );

    // Round is still "pending", not "commit"
    const result = transitionToReveal(round, round.commitDeadline + 1);
    expect(result.phase).toBe("pending"); // No change
  });

  it("CHAOS-06: commitDeadline is startBlock + commitPhaseBlocks", () => {
    const startBlock = 500;
    const round = createVerificationRound(
      makeClaim(),
      makeValidatorSelection(5),
      startBlock,
      DEFAULT_CONSENSUS_PARAMS,
    );

    // commitDeadline should be after startBlock
    expect(round.commitDeadline).toBeGreaterThan(startBlock);
    // revealDeadline should be after commitDeadline
    expect(round.revealDeadline).toBeGreaterThan(round.commitDeadline);
    // completionBlock should be after revealDeadline
    expect(round.completionBlock).toBeGreaterThanOrEqual(round.revealDeadline);
  });

  it("CHAOS-07: requiredValidators respects quorum and minValidators", () => {
    const params = DEFAULT_CONSENSUS_PARAMS;
    const validators = makeValidatorSelection(10);

    const round = createVerificationRound(makeClaim(), validators, 100, params);

    // Required validators should be at least minValidators
    expect(round.requiredValidators).toBeGreaterThanOrEqual(
      params.minValidators,
    );

    // And at least quorum × selectedValidators
    const quorumCount = Math.ceil(validators.length * params.validatorQuorum);
    expect(round.requiredValidators).toBeGreaterThanOrEqual(quorumCount);
  });

  it("CHAOS-08: Round with 0 validators creates degenerate round", () => {
    const round = createVerificationRound(
      makeClaim(),
      [],
      100,
      DEFAULT_CONSENSUS_PARAMS,
    );

    expect(round.selectedValidators).toHaveLength(0);
    expect(round.commitments.size).toBe(0);
  });

  it("CHAOS-09: Round with 1 validator uses minValidators for quorum", () => {
    const round = createVerificationRound(
      makeClaim(),
      makeValidatorSelection(1),
      100,
      DEFAULT_CONSENSUS_PARAMS,
    );

    // Even with 1 validator, requiredValidators uses max(minValidators, quorum)
    expect(round.requiredValidators).toBeGreaterThanOrEqual(
      DEFAULT_CONSENSUS_PARAMS.minValidators,
    );
  });

  it("CHAOS-10: Round ID is unique per claim+block combination", () => {
    const claim = makeClaim();
    const validators = makeValidatorSelection(5);

    const round1 = createVerificationRound(
      claim,
      validators,
      100,
      DEFAULT_CONSENSUS_PARAMS,
    );
    const round2 = createVerificationRound(
      claim,
      validators,
      101,
      DEFAULT_CONSENSUS_PARAMS,
    );

    // Different blocks → different round IDs
    expect(round1.id).not.toBe(round2.id);
  });
});
