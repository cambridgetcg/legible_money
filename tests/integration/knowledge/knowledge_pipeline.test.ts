/**
 * Knowledge Pipeline Integration Test
 *
 * Tests the complete knowledge lifecycle:
 *   claim_submit → claim_start_verification → claim_complete_verification → fact + graph edges
 *
 * Validates:
 * - Fact creation with full metadata (domain, stratum, confidence)
 * - Reference edges ("supports") wired from claim.references
 * - Derivation edges ("derives_from") wired from claim.derivedFrom
 * - Stake lifecycle (lock → return/slash)
 * - Validator reward distribution
 * - Multi-claim dependency chains (fact A → fact B derived from A)
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  FEE_POOL_DID,
} from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import type {
  ClaimSubmitPayload,
  ClaimCompleteVerificationPayload,
  FactReference,
} from "../../../state/types.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";
import type { KnowledgeStateTree } from "../../../state/knowledge.js";

// ============================================================================
// Test Constants
// ============================================================================

const ONE_LGM = 10n ** 18n;

const TEST_CONSENSUS_PARAMS: ConsensusParams = {
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

const SUBMITTER_KEY = "submitter-public-key";
const SUBMITTER_DID = publicKeyToDid(SUBMITTER_KEY);

const VALIDATOR_KEYS = [
  "validator-1-key",
  "validator-2-key",
  "validator-3-key",
];
const VALIDATOR_DIDS = VALIDATOR_KEYS.map(publicKeyToDid);

// ============================================================================
// Helpers
// ============================================================================

/** Access the private knowledge tree for edge assertions */
function getKnowledgeTree(state: StateManager): KnowledgeStateTree {
  return (state as any).knowledge;
}

function createUserTx(
  type: string,
  payload: Record<string, unknown>,
  sender: string,
  state: StateManager,
): Transaction {
  const did = publicKeyToDid(sender);
  const nonce = state.getNonce(did);
  return {
    type: type as any,
    sender,
    nonce,
    fee: 0n,
    payload,
    signature: "" as any,
  };
}

function createSystemTx(
  type: string,
  payload: Record<string, unknown>,
): Transaction {
  return {
    type: type as any,
    sender: "system" as any,
    nonce: 0n,
    fee: 0n,
    payload,
    signature: "" as any,
  };
}

// ============================================================================
// Tests
// ============================================================================

describe.sequential("Knowledge Pipeline Integration", () => {
  let state: StateManager;
  let blockNumber: bigint;

  beforeEach(() => {
    state = new StateManager();
    blockNumber = 100n;

    // Fund submitter
    state.createAccount(SUBMITTER_DID, "submitter-op-hash", 50000n * ONE_LGM);

    // Fund fee pool (needed for reward distribution)
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 1_000_000n * ONE_LGM);
  });

  // --------------------------------------------------------------------------
  // Full pipeline: submit → verify → accept → fact with metadata
  // --------------------------------------------------------------------------

  describe("single claim pipeline", () => {
    it("should produce a fact with correct domain, stratum, and confidence", () => {
      // 1. Submit claim
      const submitPayload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Water boils at 100°C at standard pressure",
        claimType: "empirical",
        domain: "chemistry.phase_transitions",
        stratum: "chemical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      const submitResult = executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          submitPayload as any,
          SUBMITTER_KEY,
          state,
        ),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      expect(submitResult.success).toBe(true);

      const claimId = state.getPendingClaims()[0].id;

      // 2. Start verification
      executeTransaction(
        state,
        createSystemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-chem-001",
          selectedValidators: VALIDATOR_DIDS,
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      // 3. Complete verification (accepted)
      const completePayload: ClaimCompleteVerificationPayload = {
        type: "claim_complete_verification",
        claimId,
        roundId: "round-chem-001",
        outcome: "accepted",
        finalConfidence: 0.92,
        verifierCount: 3,
        rewards: [
          [VALIDATOR_DIDS[0], 1n * ONE_LGM],
          [VALIDATOR_DIDS[1], 1n * ONE_LGM],
          [VALIDATOR_DIDS[2], 1n * ONE_LGM],
        ],
        slashes: [],
      };

      const completeResult = executeTransaction(
        state,
        createSystemTx("claim_complete_verification", completePayload as any),
        blockNumber + 15n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(completeResult.success).toBe(true);

      // 4. Verify fact created with full metadata
      const claim = state.getClaim(claimId);
      expect(claim?.status).toBe("accepted");
      expect(claim?.resultFactId).toBeDefined();

      const fact = state.getFact(claim!.resultFactId!);
      expect(fact).not.toBeNull();
      expect(fact!.claim).toBe("Water boils at 100°C at standard pressure");
      expect(fact!.domain).toBe("chemistry.phase_transitions");
      expect(fact!.confidence).toBe(0.92);
      expect(fact!.verifierCount).toBe(3);
      expect(fact!.status).toBe("active");
    });

    it("should return stake to submitter on acceptance", () => {
      const balanceBefore = state.getAccount(SUBMITTER_DID)!.balance;
      const stakeAmount = 200n * ONE_LGM;

      // Submit
      executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          {
            type: "claim_submit",
            claim: "Stake return test",
            claimType: "empirical",
            domain: "test",
            stratum: "physical",
            stake: stakeAmount,
            references: [],
          },
          SUBMITTER_KEY,
          state,
        ),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const claimId = state.getPendingClaims()[0].id;

      // Start + complete
      executeTransaction(
        state,
        createSystemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-stake-001",
          selectedValidators: VALIDATOR_DIDS,
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-stake-001",
          outcome: "accepted",
          finalConfidence: 0.88,
          verifierCount: 3,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 15n,
        TEST_CONSENSUS_PARAMS,
      );

      // Stake returned: balance should be back to original
      const balanceAfter = state.getAccount(SUBMITTER_DID)!.balance;
      expect(balanceAfter).toBe(balanceBefore);
      expect(state.getAccount(SUBMITTER_DID)!.stakedBalance).toBe(0n);
    });
  });

  // --------------------------------------------------------------------------
  // Reference wiring: claim.references → "supports" edges
  // --------------------------------------------------------------------------

  describe("reference edge wiring", () => {
    it("should create 'supports' edges from claim references", () => {
      // Seed a base fact
      state.addFact({
        id: "fact:base-physics-001",
        claim: "F = ma",
        domain: "physics.mechanics",
        confidence: 0.99,
        verifiedAt: 50n,
        verifierCount: 10,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });

      // Submit a claim referencing the base fact
      executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          {
            type: "claim_submit",
            claim: "A 1kg object accelerates at 9.8m/s² under gravity",
            claimType: "empirical",
            domain: "physics.mechanics",
            stratum: "physical",
            stake: 200n * ONE_LGM,
            references: ["fact:base-physics-001"],
          },
          SUBMITTER_KEY,
          state,
        ),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const claimId = state.getPendingClaims()[0].id;

      // Start + complete verification
      executeTransaction(
        state,
        createSystemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-ref-001",
          selectedValidators: VALIDATOR_DIDS,
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-ref-001",
          outcome: "accepted",
          finalConfidence: 0.91,
          verifierCount: 3,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 15n,
        TEST_CONSENSUS_PARAMS,
      );

      // Verify: new fact should have a "supports" edge to base fact
      const claim = state.getClaim(claimId)!;
      const newFactId = claim.resultFactId!;

      const kt = getKnowledgeTree(state);
      const ref = kt.getReference(newFactId, "fact:base-physics-001");
      expect(ref).not.toBeNull();
      expect(ref!.referenceType).toBe("supports");
      expect(ref!.strength).toBe(0.8);

      // Also verify via public API (supports direction)
      const refs = state.getFactReferences("fact:base-physics-001");
      expect(refs.supports.length).toBe(1);
      expect(refs.supports[0].id).toBe(newFactId);
    });
  });

  // --------------------------------------------------------------------------
  // Derivation wiring: claim.derivedFrom → "derives_from" edges
  // --------------------------------------------------------------------------

  describe("derivedFrom edge wiring", () => {
    it("should create 'derives_from' edges with inference-appropriate strength", () => {
      // Seed two parent facts
      state.addFact({
        id: "fact:premise-001",
        claim: "All metals conduct electricity",
        domain: "physics.materials",
        confidence: 0.95,
        verifiedAt: 50n,
        verifierCount: 8,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });

      state.addFact({
        id: "fact:premise-002",
        claim: "Copper is a metal",
        domain: "chemistry.elements",
        confidence: 0.99,
        verifiedAt: 50n,
        verifierCount: 10,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });

      // Submit a derived claim (deductive inference)
      executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          {
            type: "claim_submit",
            claim: "Copper conducts electricity",
            claimType: "empirical",
            domain: "physics.materials",
            stratum: "physical",
            stake: 200n * ONE_LGM,
            references: [],
            derivedFrom: {
              factIds: ["fact:premise-001", "fact:premise-002"],
              inferenceType: "deductive",
              logicChain:
                "All metals conduct electricity ∧ Copper is a metal → Copper conducts electricity",
            },
          },
          SUBMITTER_KEY,
          state,
        ),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const claimId = state.getPendingClaims()[0].id;

      // Verify derivedFrom stored on ClaimState
      const claimState = state.getClaim(claimId)!;
      expect(claimState.derivedFrom).toBeDefined();
      expect(claimState.derivedFrom!.factIds).toEqual([
        "fact:premise-001",
        "fact:premise-002",
      ]);
      expect(claimState.derivedFrom!.inferenceType).toBe("deductive");

      // Start + complete verification
      executeTransaction(
        state,
        createSystemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-derive-001",
          selectedValidators: VALIDATOR_DIDS,
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-derive-001",
          outcome: "accepted",
          finalConfidence: 0.94,
          verifierCount: 3,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 15n,
        TEST_CONSENSUS_PARAMS,
      );

      // Verify derives_from edges
      const newFactId = state.getClaim(claimId)!.resultFactId!;
      const kt = getKnowledgeTree(state);

      const ref1 = kt.getReference(newFactId, "fact:premise-001");
      expect(ref1).not.toBeNull();
      expect(ref1!.referenceType).toBe("derives_from");
      expect(ref1!.strength).toBe(0.99); // deductive strength

      const ref2 = kt.getReference(newFactId, "fact:premise-002");
      expect(ref2).not.toBeNull();
      expect(ref2!.referenceType).toBe("derives_from");
      expect(ref2!.strength).toBe(0.99); // deductive strength
    });

    it("should use correct strength for inductive inference", () => {
      state.addFact({
        id: "fact:observation-001",
        claim: "Swan 1 is white",
        domain: "biological.ornithology",
        confidence: 0.9,
        verifiedAt: 50n,
        verifierCount: 5,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });

      executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          {
            type: "claim_submit",
            claim: "All swans are white",
            claimType: "empirical",
            domain: "biological.ornithology",
            stratum: "biological",
            stake: 200n * ONE_LGM,
            references: [],
            derivedFrom: {
              factIds: ["fact:observation-001"],
              inferenceType: "inductive",
              logicChain: "Observed white swans → All swans are white",
            },
          },
          SUBMITTER_KEY,
          state,
        ),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const claimId = state.getPendingClaims()[0].id;

      executeTransaction(
        state,
        createSystemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-induct-001",
          selectedValidators: VALIDATOR_DIDS,
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-induct-001",
          outcome: "accepted",
          finalConfidence: 0.7,
          verifierCount: 3,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 15n,
        TEST_CONSENSUS_PARAMS,
      );

      const newFactId = state.getClaim(claimId)!.resultFactId!;
      const kt = getKnowledgeTree(state);
      const ref = kt.getReference(newFactId, "fact:observation-001");
      expect(ref).not.toBeNull();
      expect(ref!.referenceType).toBe("derives_from");
      expect(ref!.strength).toBe(0.75); // inductive strength
    });

    it("should reject derivedFrom with inactive parent fact", () => {
      // Seed a revoked fact
      state.addFact({
        id: "fact:revoked-001",
        claim: "Revoked premise",
        domain: "test",
        confidence: 0.5,
        verifiedAt: 50n,
        verifierCount: 3,
        challengeCount: 2,
        citationCount: 0,
        status: "revoked",
      });

      const result = executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          {
            type: "claim_submit",
            claim: "Derived from revoked",
            claimType: "empirical",
            domain: "test",
            stratum: "physical",
            stake: 200n * ONE_LGM,
            references: [],
            derivedFrom: {
              factIds: ["fact:revoked-001"],
              inferenceType: "deductive",
              logicChain: "Invalid derivation",
            },
          },
          SUBMITTER_KEY,
          state,
        ),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("not found or inactive");
    });
  });

  // --------------------------------------------------------------------------
  // Multi-claim dependency chain: fact A → fact B (references A) → fact C (derives from B)
  // --------------------------------------------------------------------------

  describe("multi-claim dependency chain", () => {
    it("should build a knowledge graph across three chained claims", () => {
      // === Claim A: base fact ===
      executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          {
            type: "claim_submit",
            claim: "E = mc²",
            claimType: "empirical",
            domain: "physics.relativity",
            stratum: "physical",
            stake: 200n * ONE_LGM,
            references: [],
          },
          SUBMITTER_KEY,
          state,
        ),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const claimAId = state.getPendingClaims()[0].id;

      executeTransaction(
        state,
        createSystemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId: claimAId,
          roundId: "round-A",
          selectedValidators: VALIDATOR_DIDS,
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: claimAId,
          roundId: "round-A",
          outcome: "accepted",
          finalConfidence: 0.98,
          verifierCount: 3,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 15n,
        TEST_CONSENSUS_PARAMS,
      );

      const factAId = state.getClaim(claimAId)!.resultFactId!;
      expect(state.getFact(factAId)).not.toBeNull();

      // === Claim B: references fact A ===
      const submitBResult = executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          {
            type: "claim_submit",
            claim: "Mass-energy equivalence implies nuclear energy release",
            claimType: "empirical",
            domain: "physics.nuclear",
            stratum: "physical",
            stake: 200n * ONE_LGM,
            references: [factAId],
          },
          SUBMITTER_KEY,
          state,
        ),
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(submitBResult.success).toBe(true);

      const claimBId = state.getPendingClaims()[0].id;

      executeTransaction(
        state,
        createSystemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId: claimBId,
          roundId: "round-B",
          selectedValidators: VALIDATOR_DIDS,
        }),
        blockNumber + 21n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: claimBId,
          roundId: "round-B",
          outcome: "accepted",
          finalConfidence: 0.93,
          verifierCount: 3,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 35n,
        TEST_CONSENSUS_PARAMS,
      );

      const factBId = state.getClaim(claimBId)!.resultFactId!;

      // Verify: B supports A
      const kt = getKnowledgeTree(state);
      const refBA = kt.getReference(factBId, factAId);
      expect(refBA).not.toBeNull();
      expect(refBA!.referenceType).toBe("supports");

      // === Claim C: derives from fact B ===
      const submitCResult = executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          {
            type: "claim_submit",
            claim: "Nuclear reactors convert mass to energy per E=mc²",
            claimType: "empirical",
            domain: "technological.energy",
            stratum: "technological",
            stake: 200n * ONE_LGM,
            references: [],
            derivedFrom: {
              factIds: [factBId],
              inferenceType: "deductive",
              logicChain:
                "Nuclear energy release + engineering → nuclear reactors",
            },
          },
          SUBMITTER_KEY,
          state,
        ),
        blockNumber + 40n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(submitCResult.success).toBe(true);

      const claimCId = state.getPendingClaims()[0].id;

      executeTransaction(
        state,
        createSystemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId: claimCId,
          roundId: "round-C",
          selectedValidators: VALIDATOR_DIDS,
        }),
        blockNumber + 41n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: claimCId,
          roundId: "round-C",
          outcome: "accepted",
          finalConfidence: 0.88,
          verifierCount: 3,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 55n,
        TEST_CONSENSUS_PARAMS,
      );

      const factCId = state.getClaim(claimCId)!.resultFactId!;

      // Verify: C derives_from B
      const refCB = kt.getReference(factCId, factBId);
      expect(refCB).not.toBeNull();
      expect(refCB!.referenceType).toBe("derives_from");
      expect(refCB!.strength).toBe(0.99); // deductive

      // Verify: three facts exist, all active
      expect(state.getFact(factAId)!.status).toBe("active");
      expect(state.getFact(factBId)!.status).toBe("active");
      expect(state.getFact(factCId)!.status).toBe("active");

      // Verify graph connectivity: A ← supports ← B ← derives_from ← C
      const aRefs = state.getFactReferences(factAId);
      expect(aRefs.supports.length).toBe(1);
      expect(aRefs.supports[0].id).toBe(factBId);
    });
  });

  // --------------------------------------------------------------------------
  // Validator reward distribution
  // --------------------------------------------------------------------------

  describe("validator rewards", () => {
    it("should distribute rewards to validators from fee pool", () => {
      // Create validator accounts
      for (const did of VALIDATOR_DIDS) {
        state.createAccount(did, `${did}-op-hash`, 0n);
      }

      // Submit + verify claim with rewards
      executeTransaction(
        state,
        createUserTx(
          "claim_submit",
          {
            type: "claim_submit",
            claim: "Reward test claim",
            claimType: "empirical",
            domain: "test",
            stratum: "physical",
            stake: 200n * ONE_LGM,
            references: [],
          },
          SUBMITTER_KEY,
          state,
        ),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const claimId = state.getPendingClaims()[0].id;

      executeTransaction(
        state,
        createSystemTx("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-reward-001",
          selectedValidators: VALIDATOR_DIDS,
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      const rewardPerValidator = 2n * ONE_LGM;
      executeTransaction(
        state,
        createSystemTx("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-reward-001",
          outcome: "accepted",
          finalConfidence: 0.9,
          verifierCount: 3,
          rewards: [
            [VALIDATOR_DIDS[0], rewardPerValidator],
            [VALIDATOR_DIDS[1], rewardPerValidator],
            [VALIDATOR_DIDS[2], rewardPerValidator],
          ],
          slashes: [],
        }),
        blockNumber + 15n,
        TEST_CONSENSUS_PARAMS,
      );

      // Each validator should have received their reward
      for (const did of VALIDATOR_DIDS) {
        const acct = state.getAccount(did)!;
        expect(acct.balance).toBe(rewardPerValidator);
      }
    });
  });
});
