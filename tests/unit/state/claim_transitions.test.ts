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
    payload,
    signature: "" as any,
  };
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

    // Create a verified fact for reference testing
    state.addFact({
      id: "fact:reference-001",
      claim: "Referenced fact for testing",
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
      expect(claim?.status).toBe("verifying");
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
        expect(fact?.claim).toBe("Test claim for completion");
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
        // 15% of claimReward (5 LGM) = 0.75 LGM, all to one cited fact
        const expectedReward = (TEST_CONSENSUS_PARAMS.claimReward * 15n) / 100n;
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
        const expectedReward = (TEST_CONSENSUS_PARAMS.claimReward * 15n) / 100n;
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
      const verifyingClaims = state.getClaimsByStatus("verifying");

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
      expect(fact?.claim).toBe(claimText);
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
});
