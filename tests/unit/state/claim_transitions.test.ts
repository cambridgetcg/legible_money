/**
 * Claim Submission Workflow Tests
 *
 * Tests the complete claim lifecycle from submission to verification.
 * Validates GAP-002 resolution: claim_submit, claim_start_verification,
 * claim_complete_verification, and claim_withdraw transaction handlers.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  type TransitionResult,
} from "../../../state/transitions.js";
import { generateClaimId } from "../../../state/claims.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import type { ClaimSubmitPayload, ClaimState } from "../../../state/types.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";
import type { OntologicalDomain } from "../../../core/knowledge/ontology/types.js";

// Test constants
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
  minValidatorStake: 111n * 10n ** 15n, // 0.111 LGM
  minClaimStake: 11n * ONE_LGM,
  blockReward: 11n * ONE_LGM,
  verificationReward: 1n * ONE_LGM,
  citationRewardPerFact: 1n * ONE_LGM,
  wrongVerificationSlash: 0.06,
  missedVerificationSlash: 0.0001,
  invalidClaimSlash: 0.22,
  researchShareRate: 0.22,
  maxTransactionsPerBlock: 111,
  unbondingPeriod: 222222n,
};

const TEST_SENDER = "test-sender-public-key";
const TEST_DID = publicKeyToDid(TEST_SENDER);

function createSystemTransaction(
  type: string,
  payload: Record<string, unknown>,
): Transaction {
  return {
    type: type as any,
    sender: "system" as any,
    nonce: 0n,
    fee: 0n,
    _systemInternal: true,
    payload,
    signature: "" as any,
  } as any;
}

// Use describe.sequential to prevent parallel test execution issues
describe.sequential("Claim Submission Workflow", () => {
  let state: StateManager;
  let blockNumber: bigint;
  // Track nonces per sender - scoped to this describe block
  let nonceTracker: Map<string, bigint>;

  function createTransaction(
    type: string,
    payload: Record<string, unknown>,
    sender = TEST_SENDER,
  ): Transaction {
    const did = publicKeyToDid(sender);
    const nonce = nonceTracker.get(did) ?? 0n;
    nonceTracker.set(did, nonce + 1n);

    return {
      type: type as any,
      sender,
      nonce,
      fee: 0n,
      payload,
      signature: "" as any,
    };
  }

  function resetNonces(): void {
    nonceTracker = new Map<string, bigint>();
  }

  beforeEach(() => {
    // Reset nonces FIRST
    resetNonces();

    // Create fresh state
    state = new StateManager();
    blockNumber = 100n;

    // Create test account with balance
    state.createAccount(TEST_DID, "test-op-key-hash", 10000n * ONE_LGM);

    // Create fee pool account
    state.createAccount(
      "did:lgm:fee-pool",
      "fee-pool-hash",
      1000000n * ONE_LGM,
    );

    // Create reward pool account (citation rewards funded from here)
    state.createAccount(
      "did:lgm:reward-pool",
      "reward-pool-hash",
      1000000n * ONE_LGM,
    );

    // Create a verified fact for reference testing
    state.addFact({
      id: "fact:reference-001",
      claimText: "Referenced fact for testing",
      domain: "test.domain",
      confidence: 0.9,
      verifiedAt: 50n,
      verifierCount: 5,
      challengeCount: 0,
      citationCount: 0,
      status: "active",
    });
  });

  // ============================================================================
  // claim_submit Tests
  // ============================================================================

  describe("executeClaimSubmit", () => {
    it("should create ClaimState with pending status", () => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "The speed of light is constant",
        claimType: "empirical",
        domain: "physics.constants",
        stratum: "fundamental",
        stake: 200n * ONE_LGM,
        references: [],
      };

      const tx = createTransaction("claim_submit", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      // Verify claim was created
      const claims = state.getPendingClaims();
      expect(claims.length).toBe(1);
      expect(claims[0].claim).toBe("The speed of light is constant");
      expect(claims[0].status).toBe("pending");
      expect(claims[0].submitter).toBe(TEST_DID);
      expect(claims[0].stake).toBe(200n * ONE_LGM);
    });

    it("should lock stake from submitter", () => {
      const initialBalance = state.getAccount(TEST_DID)!.balance;
      const stakeAmount = 200n * ONE_LGM;

      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: stakeAmount,
        references: [],
      };

      executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const account = state.getAccount(TEST_DID)!;
      expect(account.balance).toBe(initialBalance - stakeAmount);
      expect(account.stakedBalance).toBe(stakeAmount);
    });

    it("should validate minimum stake requirement", () => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 10n * ONE_LGM, // Below minimum (100 LGM)
        references: [],
      };

      const tx = createTransaction("claim_submit", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Minimum claim stake");
    });

    it("should validate references exist", () => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim with reference",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: ["fact:reference-001"], // Valid reference
      };

      const tx = createTransaction("claim_submit", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      const claims = state.getPendingClaims();
      expect(claims[0].references).toContain("fact:reference-001");
    });

    it("should reject invalid references", () => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: ["fact:nonexistent-999"],
      };

      const tx = createTransaction("claim_submit", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("not found or inactive");
    });

    it("should generate deterministic claim ID", () => {
      const claim = "Deterministic claim test";
      const id1 = generateClaimId(claim, TEST_DID, blockNumber);
      const id2 = generateClaimId(claim, TEST_DID, blockNumber);

      expect(id1).toBe(id2);
      expect(id1).toMatch(/^claim:[a-f0-9]+$/);
    });

    it("should reject insufficient balance", () => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 50000n * ONE_LGM, // More than balance
        references: [],
      };

      const tx = createTransaction("claim_submit", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Insufficient balance");
    });

    it("should reject paradoxical claims (U1)", () => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "this statement is false",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      const tx = createTransaction("claim_submit", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("paradox");
    });

    it("should classify claim zone and decidability (U1)", () => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Water boils at 100 degrees Celsius at sea level",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      const tx = createTransaction("claim_submit", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(true);

      const claims = state.getPendingClaims();
      const claim = claims[claims.length - 1];

      // Claim should have logicZone and decidability set
      expect(claim.logicZone).toBeDefined();
      expect(claim.decidability).toBeDefined();
      // Benign empirical claims should not be paradoxical
      expect(claim.goedelianStatus).not.toBe("paradoxical");
    });

    it("should detect Gödelian self-reference without rejecting (U1)", () => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "This claim cannot be proven within this system",
        claimType: "formal",
        domain: "test",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      const tx = createTransaction("claim_submit", payload);
      const result = executeTransaction(
        state,
        tx,
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      // Gödelian self-reference is NOT paradoxical — it should succeed
      expect(result.success).toBe(true);

      const claims = state.getPendingClaims();
      const claim = claims[claims.length - 1];
      expect(claim.goedelianStatus).toBe("goedelian");
    });
  });

  // ============================================================================
  // claim_start_verification Tests
  // ============================================================================

  describe("executeClaimStartVerification", () => {
    let claimId: string;

    beforeEach(() => {
      // Create a pending claim first
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim for verification",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      const claims = state.getPendingClaims();
      claimId = claims[0].id;
    });

    it("should update claim status to verifying", () => {
      const tx = createSystemTransaction("claim_start_verification", {
        type: "claim_start_verification",
        claimId,
        roundId: "round-001",
        selectedValidators: ["validator1", "validator2", "validator3"],
      });

      const result = executeTransaction(
        state,
        tx,
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      const claim = state.getClaim(claimId);
      expect(claim?.status).toBe("in_verification");
      expect(claim?.verificationRoundId).toBe("round-001");
    });

    it("should reject non-pending claims", () => {
      // First start verification
      executeTransaction(
        state,
        createSystemTransaction("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-001",
          selectedValidators: ["v1", "v2", "v3"],
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      // Try to start again
      const tx = createSystemTransaction("claim_start_verification", {
        type: "claim_start_verification",
        claimId,
        roundId: "round-002",
        selectedValidators: ["v1", "v2", "v3"],
      });

      const result = executeTransaction(
        state,
        tx,
        blockNumber + 2n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("expected pending");
    });

    it("should reject non-system transactions", () => {
      const tx = createTransaction("claim_start_verification", {
        type: "claim_start_verification",
        claimId,
        roundId: "round-001",
        selectedValidators: ["v1", "v2", "v3"],
      });

      const result = executeTransaction(
        state,
        tx,
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("system-only");
    });
  });

  // ============================================================================
  // claim_complete_verification Tests
  // ============================================================================

  describe("executeClaimCompleteVerification", () => {
    let claimId: string;

    beforeEach(() => {
      // Create and start verification for a claim
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim for completion",
        claimType: "empirical",
        domain: "test.completion",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      claimId = state.getPendingClaims()[0].id;

      executeTransaction(
        state,
        createSystemTransaction("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-001",
          selectedValidators: ["v1", "v2", "v3"],
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );
    });

    describe("accepted outcome", () => {
      it("should create FactState with full metadata", () => {
        const tx = createSystemTransaction("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-001",
          outcome: "accepted",
          finalConfidence: 0.85,
          verifierCount: 5,
          rewards: [],
          slashes: [],
        });

        const result = executeTransaction(
          state,
          tx,
          blockNumber + 10n,
          TEST_CONSENSUS_PARAMS,
        );
        expect(result.success).toBe(true);

        // Check fact was created
        const claim = state.getClaim(claimId);
        expect(claim?.status).toBe("accepted");
        expect(claim?.resultFactId).toBeDefined();

        const fact = state.getFact(claim!.resultFactId!);
        expect(fact).not.toBeNull();
        expect(fact?.claimText).toBe("Test claim for completion");
        expect(fact?.confidence).toBe(0.85);
        expect(fact?.domain).toBe("test.completion");
        expect(fact?.status).toBe("active");
      });

      it("should return stake to submitter", () => {
        const accountBefore = state.getAccount(TEST_DID)!;
        const stakedBefore = accountBefore.stakedBalance;

        executeTransaction(
          state,
          createSystemTransaction("claim_complete_verification", {
            type: "claim_complete_verification",
            claimId,
            roundId: "round-001",
            outcome: "accepted",
            finalConfidence: 0.85,
            verifierCount: 5,
            rewards: [],
            slashes: [],
          }),
          blockNumber + 10n,
          TEST_CONSENSUS_PARAMS,
        );

        const accountAfter = state.getAccount(TEST_DID)!;
        expect(accountAfter.stakedBalance).toBe(stakedBefore - 200n * ONE_LGM);
        expect(accountAfter.balance).toBeGreaterThan(accountBefore.balance);
      });
    });

    describe("rejected outcome", () => {
      it("should update claim status to rejected", () => {
        const tx = createSystemTransaction("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-001",
          outcome: "rejected",
          finalConfidence: 0.2,
          verifierCount: 5,
          rewards: [],
          slashes: [],
        });

        const result = executeTransaction(
          state,
          tx,
          blockNumber + 10n,
          TEST_CONSENSUS_PARAMS,
        );
        expect(result.success).toBe(true);

        const claim = state.getClaim(claimId);
        expect(claim?.status).toBe("rejected");
        expect(claim?.rejectionReason).toBe("Verification failed");
        expect(claim?.resultFactId).toBeUndefined();
      });

      it("should return partial stake after slash", () => {
        const accountBefore = state.getAccount(TEST_DID)!;

        executeTransaction(
          state,
          createSystemTransaction("claim_complete_verification", {
            type: "claim_complete_verification",
            claimId,
            roundId: "round-001",
            outcome: "rejected",
            finalConfidence: 0.2,
            verifierCount: 5,
            rewards: [],
            slashes: [],
          }),
          blockNumber + 10n,
          TEST_CONSENSUS_PARAMS,
        );

        const accountAfter = state.getAccount(TEST_DID)!;
        // Should return 50% (100 LGM) of 200 LGM stake
        expect(accountAfter.balance).toBe(
          accountBefore.balance + 100n * ONE_LGM,
        );
      });
    });

    describe("contested outcome", () => {
      it("should return full stake", () => {
        const accountBefore = state.getAccount(TEST_DID)!;
        const stakeAmount = 200n * ONE_LGM;

        executeTransaction(
          state,
          createSystemTransaction("claim_complete_verification", {
            type: "claim_complete_verification",
            claimId,
            roundId: "round-001",
            outcome: "contested",
            finalConfidence: 0.5,
            verifierCount: 5,
            rewards: [],
            slashes: [],
          }),
          blockNumber + 10n,
          TEST_CONSENSUS_PARAMS,
        );

        const accountAfter = state.getAccount(TEST_DID)!;
        expect(accountAfter.balance).toBe(accountBefore.balance + stakeAmount);

        const claim = state.getClaim(claimId);
        expect(claim?.rejectionReason).toContain("contested");
      });

      it("should create a fact with status 'contested' and confidence 0.5 (U2)", () => {
        const result = executeTransaction(
          state,
          createSystemTransaction("claim_complete_verification", {
            type: "claim_complete_verification",
            claimId,
            roundId: "round-001",
            outcome: "contested",
            finalConfidence: 0.5,
            verifierCount: 5,
            rewards: [],
            slashes: [],
          }),
          blockNumber + 10n,
          TEST_CONSENSUS_PARAMS,
        );
        expect(result.success).toBe(true);

        // Claim should be accepted (not rejected) with a result fact
        const claim = state.getClaim(claimId)!;
        expect(claim.status).toBe("accepted");
        expect(claim.resultFactId).toBeDefined();

        // Fact should exist with contested status and 0.5 confidence
        const fact = state.getFact(claim.resultFactId!);
        expect(fact).not.toBeNull();
        expect(fact?.confidence).toBe(0.5);
        expect(fact?.status).toBe("contested");
        expect(fact?.claimText).toBe("Test claim for completion");
        expect(fact?.domain).toBe("test.completion");
      });

      it("should propagate Gödelian metadata from claim to contested fact (U2)", () => {
        const result = executeTransaction(
          state,
          createSystemTransaction("claim_complete_verification", {
            type: "claim_complete_verification",
            claimId,
            roundId: "round-001",
            outcome: "contested",
            finalConfidence: 0.5,
            verifierCount: 5,
            rewards: [],
            slashes: [],
          }),
          blockNumber + 10n,
          TEST_CONSENSUS_PARAMS,
        );
        expect(result.success).toBe(true);

        const claim = state.getClaim(claimId)!;
        const fact = state.getFact(claim.resultFactId!)!;

        // Gödelian metadata from claim should be propagated to fact
        // The claim "Test claim for completion" is empirical, so:
        expect(fact.logicZone).toBe(claim.logicZone);
        expect(fact.decidability).toBe(claim.decidability);
      });
    });

    describe("insufficient outcome", () => {
      it("should return full stake when insufficient validators", () => {
        const accountBefore = state.getAccount(TEST_DID)!;
        const stakeAmount = 200n * ONE_LGM;

        executeTransaction(
          state,
          createSystemTransaction("claim_complete_verification", {
            type: "claim_complete_verification",
            claimId,
            roundId: "round-001",
            outcome: "insufficient",
            finalConfidence: 0,
            verifierCount: 1,
            rewards: [],
            slashes: [],
          }),
          blockNumber + 10n,
          TEST_CONSENSUS_PARAMS,
        );

        const accountAfter = state.getAccount(TEST_DID)!;
        expect(accountAfter.balance).toBe(accountBefore.balance + stakeAmount);

        const claim = state.getClaim(claimId);
        expect(claim?.rejectionReason).toContain("Insufficient validators");
      });

      it("should NOT create a fact for insufficient outcome (U2 regression)", () => {
        executeTransaction(
          state,
          createSystemTransaction("claim_complete_verification", {
            type: "claim_complete_verification",
            claimId,
            roundId: "round-001",
            outcome: "insufficient",
            finalConfidence: 0,
            verifierCount: 1,
            rewards: [],
            slashes: [],
          }),
          blockNumber + 10n,
          TEST_CONSENSUS_PARAMS,
        );

        const claim = state.getClaim(claimId)!;
        expect(claim.status).toBe("rejected");
        expect(claim.resultFactId).toBeUndefined();

        // Fact should NOT exist
        const factId = `fact:${claimId.replace("claim:", "")}`;
        const fact = state.getFact(factId);
        expect(fact).toBeNull();
      });
    });

    describe("citation economics", () => {
      // Helper: submit a claim with references, start and complete verification
      function submitAndVerifyClaim(
        claimText: string,
        refs: string[],
        derivedFrom?: {
          factIds: string[];
          inferenceType: string;
          logicChain: string;
        },
        sender = TEST_SENDER,
      ): { claimId: string; factId: string } {
        const payload: ClaimSubmitPayload = {
          type: "claim_submit",
          claim: claimText,
          claimType: "empirical",
          domain: "test.citation",
          stratum: "physical",
          stake: 200n * ONE_LGM,
          references: refs,
          derivedFrom,
        };

        executeTransaction(
          state,
          createTransaction("claim_submit", payload, sender),
          blockNumber + 20n,
          TEST_CONSENSUS_PARAMS,
        );

        const pending = state.getPendingClaims();
        const newClaimId = pending[pending.length - 1].id;

        executeTransaction(
          state,
          createSystemTransaction("claim_start_verification", {
            type: "claim_start_verification",
            claimId: newClaimId,
            roundId: `round-cite-${newClaimId}`,
            selectedValidators: ["v1", "v2", "v3"],
          }),
          blockNumber + 21n,
          TEST_CONSENSUS_PARAMS,
        );

        executeTransaction(
          state,
          createSystemTransaction("claim_complete_verification", {
            type: "claim_complete_verification",
            claimId: newClaimId,
            roundId: `round-cite-${newClaimId}`,
            outcome: "accepted",
            finalConfidence: 0.85,
            verifierCount: 5,
            rewards: [],
            slashes: [],
          }),
          blockNumber + 30n,
          TEST_CONSENSUS_PARAMS,
        );

        const updatedClaim = state.getClaim(newClaimId)!;
        return {
          claimId: newClaimId,
          factId: updatedClaim.resultFactId!,
        };
      }

      it("should increment citation count on referenced facts", () => {
        const factBefore = state.getFact("fact:reference-001")!;
        expect(factBefore.citationCount).toBe(0);

        submitAndVerifyClaim("Claim citing reference-001", [
          "fact:reference-001",
        ]);

        const factAfter = state.getFact("fact:reference-001")!;
        expect(factAfter.citationCount).toBe(1);
        expect(factAfter.lastCitedBlock).toBe(blockNumber + 30n);
      });

      it("should increment citation count on derivedFrom facts", () => {
        const factBefore = state.getFact("fact:reference-001")!;
        expect(factBefore.citationCount).toBe(0);

        submitAndVerifyClaim("Derived from reference-001", [], {
          factIds: ["fact:reference-001"],
          inferenceType: "deductive",
          logicChain: "A -> B",
        });

        const factAfter = state.getFact("fact:reference-001")!;
        expect(factAfter.citationCount).toBe(1);
      });

      it("should record citation rewards on cited facts", () => {
        const factBefore = state.getFact("fact:reference-001")!;
        expect(factBefore.totalCitationRewards ?? 0n).toBe(0n);

        submitAndVerifyClaim("Claim for citation reward test", [
          "fact:reference-001",
        ]);

        const factAfter = state.getFact("fact:reference-001")!;
        // Each cited fact gets citationRewardPerFact (1 LGM)
        const expectedReward = TEST_CONSENSUS_PARAMS.citationRewardPerFact;
        expect(factAfter.totalCitationRewards).toBe(expectedReward);
      });

      it("should transfer citation reward to original fact submitter", () => {
        // Create a second account that will submit a fact
        const SECOND_SENDER = "second-sender-key";
        const SECOND_DID = publicKeyToDid(SECOND_SENDER);
        state.createAccount(SECOND_DID, "second-op-hash", 10000n * ONE_LGM);

        // Second account submits and gets a fact verified (no references)
        const { factId: secondFact } = submitAndVerifyClaim(
          "Fact by second sender",
          [],
          undefined,
          SECOND_SENDER,
        );

        // Record second account's balance before citation
        const balanceBefore = state.getAccount(SECOND_DID)!.balance;

        // Now TEST_DID submits a claim referencing secondFact
        submitAndVerifyClaim(`Claim citing ${secondFact}`, [secondFact]);

        const balanceAfter = state.getAccount(SECOND_DID)!.balance;
        // Citation reward routed through 22% evidence tax: recipient gets 78%
        const grossReward = TEST_CONSENSUS_PARAMS.citationRewardPerFact;
        const expectedReward = grossReward - (grossReward * 2200n) / 10000n;
        expect(balanceAfter - balanceBefore).toBe(expectedReward);
      });

      it("should not fail for genesis facts without claim submitter", () => {
        // fact:reference-001 was created directly (no associated claim)
        // Citation should still work — count increments, no transfer error
        const result = submitAndVerifyClaim("Citing a genesis-like fact", [
          "fact:reference-001",
        ]);
        expect(result.factId).toBeDefined();

        const fact = state.getFact("fact:reference-001")!;
        expect(fact.citationCount).toBe(1);
        expect(fact.totalCitationRewards).toBeGreaterThan(0n);
      });

      it("should deduplicate citation count for facts in both references and derivedFrom", () => {
        submitAndVerifyClaim(
          "Claim with overlapping refs",
          ["fact:reference-001"],
          {
            factIds: ["fact:reference-001"],
            inferenceType: "inductive",
            logicChain: "observation -> generalization",
          },
        );

        const fact = state.getFact("fact:reference-001")!;
        // Should count only once despite appearing in both references and derivedFrom
        expect(fact.citationCount).toBe(1);
      });
    });
  });

  // ============================================================================
  // claim_withdraw Tests
  // ============================================================================

  describe("executeClaimWithdraw", () => {
    let claimId: string;

    beforeEach(() => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim for withdrawal",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      claimId = state.getPendingClaims()[0].id;
    });

    it("should allow submitter to withdraw pending claim", () => {
      const tx = createTransaction("claim_withdraw", {
        type: "claim_withdraw",
        claimId,
      });

      const result = executeTransaction(
        state,
        tx,
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(true);

      const claim = state.getClaim(claimId);
      expect(claim?.status).toBe("withdrawn");
    });

    it("should return stake on withdrawal", () => {
      const accountBefore = state.getAccount(TEST_DID)!;
      const stakeAmount = 200n * ONE_LGM;

      executeTransaction(
        state,
        createTransaction("claim_withdraw", {
          type: "claim_withdraw",
          claimId,
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      const accountAfter = state.getAccount(TEST_DID)!;
      expect(accountAfter.balance).toBe(accountBefore.balance + stakeAmount);
      expect(accountAfter.stakedBalance).toBe(
        accountBefore.stakedBalance - stakeAmount,
      );
    });

    it("should reject withdrawal by non-submitter", () => {
      // Create another account
      state.createAccount("did:lgm:other-user", "other-hash", 1000n * ONE_LGM);

      const tx = createTransaction(
        "claim_withdraw",
        {
          type: "claim_withdraw",
          claimId,
        },
        "other-user",
      );

      const result = executeTransaction(
        state,
        tx,
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(false);
      expect(result.error).toContain("Only claim submitter");
    });

    it("should reject withdrawal of non-pending claims", () => {
      // Start verification first
      executeTransaction(
        state,
        createSystemTransaction("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-001",
          selectedValidators: ["v1", "v2", "v3"],
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      // Try to withdraw
      const tx = createTransaction("claim_withdraw", {
        type: "claim_withdraw",
        claimId,
      });

      const result = executeTransaction(
        state,
        tx,
        blockNumber + 2n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(false);
      expect(result.error).toContain("Cannot withdraw");
    });
  });

  // ============================================================================
  // ClaimStateTree Tests
  // ============================================================================

  describe("ClaimStateTree", () => {
    it("should track claim statistics", () => {
      // Submit multiple claims
      for (let i = 0; i < 3; i++) {
        const payload: ClaimSubmitPayload = {
          type: "claim_submit",
          claim: `Test claim ${i}`,
          claimType: "empirical",
          domain: "test",
          stratum: "physical",
          stake: 200n * ONE_LGM,
          references: [],
        };
        executeTransaction(
          state,
          createTransaction("claim_submit", payload),
          blockNumber + BigInt(i),
          TEST_CONSENSUS_PARAMS,
        );
      }

      const stats = state.getClaimStats();
      expect(stats.totalClaims).toBe(3);
      expect(stats.pendingClaims).toBe(3);
      expect(stats.totalStakeLocked).toBe(600n * ONE_LGM);
    });

    it("should index claims by submitter", () => {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const claims = state.getClaimsBySubmitter(TEST_DID);
      expect(claims.length).toBe(1);
      expect(claims[0].submitter).toBe(TEST_DID);
    });

    it("should index claims by status", () => {
      // Create and verify a claim
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Test claim",
        claimType: "empirical",
        domain: "test",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      const claimId = state.getPendingClaims()[0].id;

      executeTransaction(
        state,
        createSystemTransaction("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-001",
          selectedValidators: ["v1", "v2", "v3"],
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      const pendingClaims = state.getClaimsByStatus("pending");
      const verifyingClaims = state.getClaimsByStatus("in_verification");

      expect(pendingClaims.length).toBe(0);
      expect(verifyingClaims.length).toBe(1);
    });
  });

  // ============================================================================
  // Complete Flow Integration Tests
  // ============================================================================

  describe("Complete Claim Flow", () => {
    it("should flow from submission to verified fact", () => {
      // 1. Submit claim
      const claimText = "E = mc² describes mass-energy equivalence";
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: claimText,
        claimType: "empirical",
        domain: "physics.relativity",
        stratum: "fundamental",
        stake: 200n * ONE_LGM,
        references: [],
      };

      executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      // 2. Start verification
      const claimId = state.getPendingClaims()[0].id;
      executeTransaction(
        state,
        createSystemTransaction("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-physics-001",
          selectedValidators: ["v1", "v2", "v3", "v4", "v5"],
        }),
        blockNumber + 5n,
        TEST_CONSENSUS_PARAMS,
      );

      // 3. Complete verification (accepted)
      executeTransaction(
        state,
        createSystemTransaction("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-physics-001",
          outcome: "accepted",
          finalConfidence: 0.95,
          verifierCount: 5,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );

      // 4. Verify final state
      const claim = state.getClaim(claimId);
      expect(claim?.status).toBe("accepted");

      const fact = state.getFact(claim!.resultFactId!);
      expect(fact).not.toBeNull();
      expect(fact?.claimText).toBe(claimText);
      expect(fact?.confidence).toBe(0.95);
      expect(fact?.domain).toBe("physics.relativity");

      // Stake should be returned
      const account = state.getAccount(TEST_DID)!;
      expect(account.stakedBalance).toBe(0n);
    });

    it("should handle rejection with proper stake handling", () => {
      const initialBalance = state.getAccount(TEST_DID)!.balance;
      const stakeAmount = 200n * ONE_LGM;

      // Submit claim
      executeTransaction(
        state,
        createTransaction("claim_submit", {
          type: "claim_submit",
          claim: "False claim",
          claimType: "empirical",
          domain: "test",
          stratum: "physical",
          stake: stakeAmount,
          references: [],
        }),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const claimId = state.getPendingClaims()[0].id;

      // Start and complete verification (rejected)
      executeTransaction(
        state,
        createSystemTransaction("claim_start_verification", {
          type: "claim_start_verification",
          claimId,
          roundId: "round-001",
          selectedValidators: ["v1", "v2", "v3"],
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTransaction("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId,
          roundId: "round-001",
          outcome: "rejected",
          finalConfidence: 0.1,
          verifierCount: 5,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );

      // Check balances
      const account = state.getAccount(TEST_DID)!;
      // Should have lost 50% of stake (100 LGM)
      expect(account.balance).toBe(initialBalance - stakeAmount / 2n);
      expect(account.stakedBalance).toBe(0n);

      // No fact should be created
      const claim = state.getClaim(claimId);
      expect(claim?.resultFactId).toBeUndefined();
    });
  });

  // ============================================================================
  // Gödelian Logic Zone Integration Tests (G1/G2 Fix)
  // ============================================================================

  describe("Logic Zone Integration", () => {
    // Helper: set up domain tree with a domain that has a specific logicZoneDefault
    function addTestDomain(
      domainId: string,
      logicZoneDefault:
        | "propositional"
        | "presburger"
        | "peano"
        | "set_theory"
        | "empirical",
      stratum:
        | "fundamental"
        | "physical"
        | "chemical"
        | "biological"
        | "cognitive"
        | "social"
        | "technological" = "fundamental",
    ): void {
      const domain: OntologicalDomain = {
        id: domainId,
        name: domainId,
        stratum,
        parentDomain: "root",
        childDomains: [],
        dependsOnStrata: [],
        logicZoneDefault,
        epistemicCategoryDefault: "formal",
        description: `Test domain with ${logicZoneDefault} zone`,
        examples: [],
        factCount: 0,
        axiomCount: 0,
        createdAt: 0,
      };
      state.addDomainToTree(domain);
    }

    // Helper: submit, start verification, and complete a claim
    function submitAndVerify(
      claimText: string,
      domain: string,
      options: {
        stratum?: "fundamental" | "physical";
        finalConfidence?: number;
        references?: string[];
        derivedFrom?: {
          factIds: string[];
          inferenceType: string;
          logicChain: string;
        };
      } = {},
    ): { claimId: string; factId: string } {
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: claimText,
        claimType: "empirical",
        domain,
        stratum: options.stratum ?? "fundamental",
        stake: 200n * ONE_LGM,
        references: options.references ?? [],
        derivedFrom: options.derivedFrom,
      };

      const submitResult = executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      if (!submitResult.success)
        throw new Error(`Submit failed: ${submitResult.error}`);

      const pending = state.getPendingClaims();
      const cId = pending[pending.length - 1].id;

      executeTransaction(
        state,
        createSystemTransaction("claim_start_verification", {
          type: "claim_start_verification",
          claimId: cId,
          roundId: `round-zone-${cId}`,
          selectedValidators: ["v1", "v2", "v3"],
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTransaction("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: cId,
          roundId: `round-zone-${cId}`,
          outcome: "accepted",
          finalConfidence: options.finalConfidence ?? 0.85,
          verifierCount: 5,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );

      const claim = state.getClaim(cId)!;
      return { claimId: cId, factId: claim.resultFactId! };
    }

    it("G2: should reconcile text zone with domain logicZoneDefault", () => {
      // Set up a domain with peano zone default
      addTestDomain("math.number_theory", "peano");

      // Submit a simple claim (text classifies as "propositional")
      // but domain says "peano" — intersectZones should pick the higher zone
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Goldbach conjecture holds for all tested values",
        claimType: "empirical",
        domain: "math.number_theory",
        stratum: "fundamental",
        stake: 200n * ONE_LGM,
        references: [],
      };

      executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const claims = state.getPendingClaims();
      const claim = claims[claims.length - 1];

      // Domain default "peano" is higher than text-classified zone
      // intersectZones returns the higher zone
      expect(claim.logicZone).toBe("peano");
    });

    it("G1: should cap confidence at 0.99 for Gödel-applicable zones", () => {
      addTestDomain("math.set_theory", "set_theory");

      const { factId } = submitAndVerify(
        "ZFC is consistent",
        "math.set_theory",
        { finalConfidence: 1.0 },
      );

      const fact = state.getFact(factId)!;
      // set_theory zone has goedelApplies=true → capped at 0.99
      expect(fact.confidence).toBe(0.99);
    });

    it("G1: should not cap confidence for non-Gödel zones", () => {
      addTestDomain("logic.propositional", "propositional");

      const { factId } = submitAndVerify(
        "If A then B; A; therefore B",
        "logic.propositional",
        { finalConfidence: 1.0 },
      );

      const fact = state.getFact(factId)!;
      // propositional zone has goedelApplies=false → no cap
      expect(fact.confidence).toBe(1.0);
    });

    it("G1: should reject propositional claim deriving from peano fact (canReference)", () => {
      addTestDomain("logic.propositional", "propositional");
      addTestDomain("math.arithmetic", "peano");

      // Create a peano-zone fact first
      state.addFact({
        id: "fact:peano-parent",
        claimText: "Multiplication is commutative",
        domain: "math.arithmetic",
        confidence: 0.9,
        verifiedAt: 50n,
        verifierCount: 5,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
        logicZone: "peano",
      });

      // Try to submit a propositional claim that derives from the peano fact
      // propositional (zone 0) cannot reference peano (zone 2)
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Simple boolean statement",
        claimType: "empirical",
        domain: "logic.propositional",
        stratum: "fundamental",
        stake: 200n * ONE_LGM,
        references: [],
        derivedFrom: {
          factIds: ["fact:peano-parent"],
          inferenceType: "deductive",
          logicChain: "A -> B",
        },
      };

      const result = executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Zone violation");
      expect(result.error).toContain("propositional");
      expect(result.error).toContain("peano");
    });

    it("G1: should allow peano claim deriving from propositional fact (canReference)", () => {
      addTestDomain("math.arithmetic", "peano");

      // Create a propositional-zone fact
      state.addFact({
        id: "fact:prop-parent",
        claimText: "If P then Q",
        domain: "logic",
        confidence: 0.95,
        verifiedAt: 50n,
        verifierCount: 5,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
        logicZone: "propositional",
      });

      // peano (zone 2) can reference propositional (zone 0) — higher references lower
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Based on the logical premise, multiplication distributes",
        claimType: "empirical",
        domain: "math.arithmetic",
        stratum: "fundamental",
        stake: 200n * ONE_LGM,
        references: [],
        derivedFrom: {
          factIds: ["fact:prop-parent"],
          inferenceType: "deductive",
          logicChain: "modus ponens",
        },
      };

      const result = executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);
    });

    it("G1: should propagate logicZone and decidability from claim to fact", () => {
      addTestDomain("math.analysis", "peano");

      const { factId } = submitAndVerify(
        "Every bounded sequence has a convergent subsequence",
        "math.analysis",
        { finalConfidence: 0.9 },
      );

      const fact = state.getFact(factId)!;
      expect(fact.logicZone).toBeDefined();
      expect(fact.decidability).toBeDefined();
      // The claim text doesn't trigger multiplication keywords, but domain
      // default is peano, so intersectZones picks peano
      expect(fact.logicZone).toBe("peano");
    });

    it("should skip canReference check for parent facts without zone metadata (backward compat)", () => {
      // fact:reference-001 from beforeEach has no logicZone — legacy fact
      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Derived from a legacy fact without zone",
        claimType: "empirical",
        domain: "test.domain",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
        derivedFrom: {
          factIds: ["fact:reference-001"],
          inferenceType: "inductive",
          logicChain: "observation",
        },
      };

      const result = executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      // Should succeed — no zone check on legacy facts
      expect(result.success).toBe(true);
    });

    it("should persist logicZone and decidability on fact (serialization coverage)", () => {
      addTestDomain("math.sets", "set_theory");

      const { factId } = submitAndVerify(
        "The cardinality of the continuum",
        "math.sets",
        { finalConfidence: 0.8 },
      );

      const fact = state.getFact(factId)!;
      // Verify zone metadata was persisted on the fact
      expect(fact.logicZone).toBe("set_theory");
      expect(fact.decidability).toBeDefined();
      // set_theory has goedelApplies=true, so confidence capped at 0.99
      // but 0.8 < 0.99 so no cap effect here
      expect(fact.confidence).toBe(0.8);
      expect(fact.domain).toBe("math.sets");
    });

    // ============ U3: Stratum ceiling on contested facts ============

    it("U3: should apply stratum ceiling to contested facts", () => {
      // social stratum has ceiling 0.85
      addTestDomain("sociology.norms", "empirical", "social");

      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "Social norms vary by culture",
        claimType: "empirical",
        domain: "sociology.norms",
        stratum: "social",
        stake: 200n * ONE_LGM,
        references: [],
      };

      const submitResult = executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );
      expect(submitResult.success).toBe(true);

      const pending = state.getPendingClaims();
      const cId = pending[pending.length - 1].id;

      executeTransaction(
        state,
        createSystemTransaction("claim_start_verification", {
          type: "claim_start_verification",
          claimId: cId,
          roundId: `round-u3-${cId}`,
          selectedValidators: ["v1", "v2", "v3"],
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      // Complete with "contested" — confidence should be min(0.5, 0.85) = 0.5
      executeTransaction(
        state,
        createSystemTransaction("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: cId,
          roundId: `round-u3-${cId}`,
          outcome: "contested",
          finalConfidence: 0.5,
          verifierCount: 5,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );

      const claim = state.getClaim(cId)!;
      expect(claim.resultFactId).toBeDefined();
      const fact = state.getFact(claim.resultFactId!)!;
      expect(fact.status).toBe("contested");
      // 0.5 is below the social ceiling 0.85, so no clamping needed
      expect(fact.confidence).toBe(0.5);
      expect(fact.stratumCeiling).toBe(0.85);
    });

    it("U3: should populate stratumCeiling on accepted facts", () => {
      addTestDomain("physics.dynamics", "empirical", "physical");

      const { factId } = submitAndVerify(
        "Group dynamics follow predictable patterns",
        "physics.dynamics",
        { stratum: "physical", finalConfidence: 0.9 },
      );

      const fact = state.getFact(factId)!;
      expect(fact.status).toBe("active");
      // physical stratum ceiling is 0.99 — 0.9 is within bounds
      expect(fact.confidence).toBe(0.9);
      expect(fact.stratumCeiling).toBeDefined();
    });
  });

  // ============================================================================
  // resolve_undecidable Tests (U7)
  // ============================================================================

  describe("executeResolveUndecidable (U7)", () => {
    let contestedFactId: string;
    let claimIdU7: string;

    function addTestDomainU7(
      domainId: string,
      logicZoneDefault:
        | "propositional"
        | "presburger"
        | "peano"
        | "set_theory"
        | "empirical",
      stratum:
        | "fundamental"
        | "physical"
        | "chemical"
        | "biological"
        | "cognitive"
        | "social"
        | "technological" = "fundamental",
    ): void {
      const domain: OntologicalDomain = {
        id: domainId,
        name: domainId,
        stratum,
        parentDomain: "root",
        childDomains: [],
        dependsOnStrata: [],
        logicZoneDefault,
        epistemicCategoryDefault: "formal",
        description: `U7 test domain`,
        examples: [],
        factCount: 0,
        axiomCount: 0,
        createdAt: 0,
      };
      state.addDomainToTree(domain);
    }

    beforeEach(() => {
      // Submit a claim and complete it as "contested" to create a contested fact
      addTestDomainU7("test.undecidable", "empirical", "physical");

      const payload: ClaimSubmitPayload = {
        type: "claim_submit",
        claim: "A claim about an undecidable matter",
        claimType: "empirical",
        domain: "test.undecidable",
        stratum: "physical",
        stake: 200n * ONE_LGM,
        references: [],
      };

      executeTransaction(
        state,
        createTransaction("claim_submit", payload),
        blockNumber,
        TEST_CONSENSUS_PARAMS,
      );

      const pending = state.getPendingClaims();
      claimIdU7 = pending[pending.length - 1].id;

      executeTransaction(
        state,
        createSystemTransaction("claim_start_verification", {
          type: "claim_start_verification",
          claimId: claimIdU7,
          roundId: `round-u7-${claimIdU7}`,
          selectedValidators: ["v1", "v2", "v3"],
        }),
        blockNumber + 1n,
        TEST_CONSENSUS_PARAMS,
      );

      executeTransaction(
        state,
        createSystemTransaction("claim_complete_verification", {
          type: "claim_complete_verification",
          claimId: claimIdU7,
          roundId: `round-u7-${claimIdU7}`,
          outcome: "contested",
          finalConfidence: 0.5,
          verifierCount: 5,
          rewards: [],
          slashes: [],
        }),
        blockNumber + 10n,
        TEST_CONSENSUS_PARAMS,
      );

      const claim = state.getClaim(claimIdU7)!;
      contestedFactId = claim.resultFactId!;
      expect(state.getFact(contestedFactId)!.status).toBe("contested");
    });

    it("should resolve via oracle path (confidence → 0.8, status → active)", () => {
      const result = executeTransaction(
        state,
        createSystemTransaction("resolve_undecidable", {
          factId: contestedFactId,
          resolutionPath: "oracle",
          oracle: { oracleId: "oracle:physics-lab", verdict: true },
        }),
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      const fact = state.getFact(contestedFactId)!;
      expect(fact.status).toBe("active");
      expect(fact.confidence).toBe(0.8);
      expect(fact.undecidableResolution).toBeDefined();
      expect(fact.undecidableResolution!.resolutionPath).toBe("oracle");
      expect(fact.undecidableResolution!.previousConfidence).toBe(0.5);
      expect(fact.undecidableResolution!.oracle?.oracleId).toBe(
        "oracle:physics-lab",
      );
    });

    it("should resolve via consensus path (confidence → 0.7, status → active)", () => {
      const result = executeTransaction(
        state,
        createSystemTransaction("resolve_undecidable", {
          factId: contestedFactId,
          resolutionPath: "consensus",
          consensus: { proposalId: "prop-001", votesFor: 15, votesAgainst: 3 },
        }),
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      const fact = state.getFact(contestedFactId)!;
      expect(fact.status).toBe("active");
      expect(fact.confidence).toBe(0.7);
      expect(fact.undecidableResolution!.resolutionPath).toBe("consensus");
      expect(fact.undecidableResolution!.consensus?.votesFor).toBe(15);
    });

    it("should resolve via new_axiom path (confidence → 0.99 with physical ceiling)", () => {
      const result = executeTransaction(
        state,
        createSystemTransaction("resolve_undecidable", {
          factId: contestedFactId,
          resolutionPath: "new_axiom",
          newAxiom: { axiomId: "axiom:new-physical-law" },
        }),
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      const fact = state.getFact(contestedFactId)!;
      expect(fact.status).toBe("active");
      // new_axiom gives 1.0 but physical stratum ceiling is 0.99
      expect(fact.confidence).toBe(0.99);
      expect(fact.undecidableResolution!.resolutionPath).toBe("new_axiom");
      expect(fact.undecidableResolution!.newAxiom?.axiomId).toBe(
        "axiom:new-physical-law",
      );
    });

    it("should resolve via accept_undecidability (stays contested at 0.5)", () => {
      const result = executeTransaction(
        state,
        createSystemTransaction("resolve_undecidable", {
          factId: contestedFactId,
          resolutionPath: "accept_undecidability",
        }),
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(true);

      const fact = state.getFact(contestedFactId)!;
      expect(fact.status).toBe("contested"); // stays contested
      expect(fact.confidence).toBe(0.5);
      expect(fact.undecidableResolution!.resolutionPath).toBe(
        "accept_undecidability",
      );
    });

    it("should reject resolution of non-contested fact", () => {
      // Create an active fact directly
      state.addFact({
        id: "fact:active-001",
        claimText: "An active fact",
        domain: "test.undecidable",
        confidence: 0.9,
        verifiedAt: 50n,
        verifierCount: 5,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });

      const result = executeTransaction(
        state,
        createSystemTransaction("resolve_undecidable", {
          factId: "fact:active-001",
          resolutionPath: "oracle",
          oracle: { oracleId: "oracle:test", verdict: true },
        }),
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("contested");
    });

    it("should reject resolution of non-existent fact", () => {
      const result = executeTransaction(
        state,
        createSystemTransaction("resolve_undecidable", {
          factId: "fact:nonexistent",
          resolutionPath: "oracle",
          oracle: { oracleId: "oracle:test", verdict: true },
        }),
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("not found");
    });

    it("should reject oracle resolution without oracleId", () => {
      const result = executeTransaction(
        state,
        createSystemTransaction("resolve_undecidable", {
          factId: contestedFactId,
          resolutionPath: "oracle",
          // Missing oracle payload
        }),
        blockNumber + 20n,
        TEST_CONSENSUS_PARAMS,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("oracle");
    });

    it("should store resolution metadata with resolvedAt block number", () => {
      const resolveBlock = blockNumber + 25n;
      executeTransaction(
        state,
        createSystemTransaction("resolve_undecidable", {
          factId: contestedFactId,
          resolutionPath: "consensus",
          consensus: { proposalId: "prop-002", votesFor: 20, votesAgainst: 5 },
        }),
        resolveBlock,
        TEST_CONSENSUS_PARAMS,
      );

      const fact = state.getFact(contestedFactId)!;
      expect(fact.undecidableResolution!.resolvedAt).toBe(resolveBlock);
    });
  });
});
