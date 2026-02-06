/**
 * Gödelian Integration Tests
 *
 * Tests that the logic layer is properly wired into claim submission.
 * Verifies: zone classification, decidability assessment, paradox rejection.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  resetBootstrapFund,
} from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";

const ONE_LGM = 10n ** 18n;

const TEST_PARAMS: ConsensusParams = {
  targetBlockTimeMs: 2521,
  verificationRoundBlocks: 11,
  commitPhaseBlocks: 4,
  revealPhaseBlocks: 4,
  epochLength: 1111n,
  minValidators: 3,
  maxValidators: 22,
  validatorQuorum: 0.66,
  acceptanceThreshold: 0.77,
  rejectionThreshold: 0.77,
  contestedThreshold: 0.55,
  minValidatorStake: 1111n * ONE_LGM,
  minClaimStake: 100n * ONE_LGM,
  blockReward: 11n * ONE_LGM,
  verificationReward: 1n * ONE_LGM,
  claimReward: 5n * ONE_LGM,
  wrongVerificationSlash: 0.11,
  missedVerificationSlash: 0.05,
  invalidClaimSlash: 0.22,
  researchShareRate: 0.22,
  maxTransactionsPerBlock: 111,
  unbondingPeriod: 222222n,
};

const TEST_SENDER = "a".repeat(64);
const TEST_DID = publicKeyToDid(TEST_SENDER);

let currentNonce = 0n;

function createClaimTx(
  claim: string,
  overrides?: Record<string, unknown>,
): Transaction {
  const nonce = currentNonce;
  currentNonce += 1n;
  return {
    type: "claim_submit",
    sender: TEST_SENDER,
    nonce,
    fee: 0n,
    timestamp: 1000,
    signature: "test_sig",
    payload: {
      type: "claim_submit",
      claim,
      claimType: "empirical",
      domain: "physics.classical",
      stratum: "physical",
      stake: 100n * ONE_LGM,
      references: [],
      ...overrides,
    },
  } as Transaction;
}

describe.sequential("Gödelian Claim Integration", () => {
  let state: StateManager;

  beforeEach(() => {
    currentNonce = 0n;
    resetBootstrapFund();
    state = new StateManager();
    state.createAccount(TEST_DID, "test-op-key-hash", 10000n * ONE_LGM);
  });

  it("should accept benign claims and populate logicZone + decidability", () => {
    const tx = createClaimTx("Water boils at 100 degrees Celsius at sea level");
    const result = executeTransaction(state, tx, 1n, TEST_PARAMS);
    expect(result.success).toBe(true);

    // Retrieve the claim and check Gödelian metadata
    const claims = state.getClaimsBySubmitter(TEST_DID);
    expect(claims.length).toBe(1);
    const claim = claims[0];
    expect(claim.logicZone).toBeDefined();
    expect(claim.decidability).toBeDefined();
    expect(claim.goedelianStatus).toBe("benign");
  });

  it("should reject paradoxical claims (Liar paradox)", () => {
    const tx = createClaimTx("This statement is false");
    const result = executeTransaction(state, tx, 1n, TEST_PARAMS);
    expect(result.success).toBe(false);
    expect(result.error).toContain("paradox");
  });

  it("should reject Russell-type paradox", () => {
    const tx = createClaimTx(
      "The set of all sets that do not contain themselves is a valid set",
    );
    const result = executeTransaction(state, tx, 1n, TEST_PARAMS);
    expect(result.success).toBe(false);
    expect(result.error).toContain("paradox");
  });

  it("should accept Gödelian claims (self-referential but not paradoxical)", () => {
    // Use text that matches the Gödelian pattern: "cannot be proven in this system"
    const tx = createClaimTx(
      "This proposition cannot be proved within this system",
    );
    const result = executeTransaction(state, tx, 1n, TEST_PARAMS);
    // Gödelian claims are accepted — they're valid claims about system limits
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(TEST_DID);
    expect(claims.length).toBe(1);
    expect(claims[0].goedelianStatus).toBe("goedelian");
  });

  it("should classify empirical claims correctly", () => {
    const tx = createClaimTx(
      "The experiment observed a temperature increase of 5 degrees",
    );
    const result = executeTransaction(state, tx, 1n, TEST_PARAMS);
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(TEST_DID);
    expect(claims[0].logicZone).toBe("empirical");
    // Benign claims in empirical zone return "decidable" because
    // goedelApplies("empirical") is false and the claim is not self-referential.
    // Only self-referential empirical claims reach the "empirical" decidability path.
    expect(claims[0].decidability).toBe("decidable");
  });

  it("should classify multiplication claims in peano zone", () => {
    const tx = createClaimTx("The product of 7 and 13 equals 91", {
      formalNotation: "7 × 13 = 91",
    });
    const result = executeTransaction(state, tx, 1n, TEST_PARAMS);
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(TEST_DID);
    expect(claims[0].logicZone).toBe("peano");
  });

  it("should classify addition claims in presburger zone", () => {
    const tx = createClaimTx("The successor of 5 is 6", {
      formalNotation: "S(5) = 6",
    });
    const result = executeTransaction(state, tx, 1n, TEST_PARAMS);
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(TEST_DID);
    expect(claims[0].logicZone).toBe("presburger");
  });

  it("should assess Gödelian claims in peano zone as undecidable_self_ref", () => {
    // To get undecidable_self_ref, the claim must:
    // 1. Be self-referential (goedelian=true)
    // 2. Be in a zone where Gödel applies (peano or set_theory)
    // Use formal notation with multiplication to force peano classification
    const tx = createClaimTx(
      "This proposition cannot be proved within this system",
      {
        formalNotation: "∃G. ¬Provable(G) ∧ G × S(G) is not derivable",
        claimType: "logical",
        domain: "mathematics.logic",
        stratum: "formal",
      },
    );
    const result = executeTransaction(state, tx, 1n, TEST_PARAMS);
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(TEST_DID);
    expect(claims[0].goedelianStatus).toBe("goedelian");
    expect(claims[0].logicZone).toBe("peano");
    expect(claims[0].decidability).toBe("undecidable_self_ref");
  });
});
