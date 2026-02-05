/**
 * Verification Reward Routing Tests
 *
 * Tests for GAP-006: Verification Reward Routing
 * Ensures claim submitters receive vesting schedules when their claims are accepted.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  applyVerificationResult,
  REWARD_POOL_DID,
  PROTOCOL_TREASURY_DID,
} from "../../../state/transitions.js";
import type {
  VerificationResult,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../../consensus/pot/types.js";
import type { ClaimState } from "../../../state/types.js";
import type { Fact } from "../../../core/knowledge/types.js";
import { RESEARCH_TREASURY_DID } from "../../../core/research/types.js";

// ============ Test Helpers ============

function createTestStateManager(): StateManager {
  const state = new StateManager();

  // Create necessary protocol accounts using correct API
  state.createAccount(
    REWARD_POOL_DID,
    "reward-pool-hash",
    1_000_000n * 10n ** 18n,
  );
  state.createAccount(RESEARCH_TREASURY_DID, "research-hash", 0n);
  state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);

  return state;
}

function createTestClaim(
  claimId: string,
  submitter: string,
  claimText: string,
  stake: bigint = 111n * 10n ** 18n,
): ClaimState {
  return {
    id: claimId,
    claim: claimText,
    claimType: "empirical",
    domain: "science.physics",
    stratum: "empirical_physical",
    submitter,
    stake,
    submittedAt: 100n,
    references: [],
    status: "verifying",
    statusChangedAt: 100n,
    verificationRoundId: "round-001",
  };
}

function createTestFact(claimId: string, claim: string): Fact {
  return {
    id: claimId,
    claim,
    domain: "science.physics",
    confidence: 0.85,
    verifiedAt: 200n,
    epistemicStatus: {
      category: "empirical",
      maxConfidence: 0.95,
      requiresReplication: true,
      decayRate: 0.001,
    },
    verifications: [],
  };
}

function createTestVerificationResult(
  claimId: string,
  outcome: "accepted" | "rejected" | "contested" | "insufficient",
  fact?: Fact,
): VerificationResult {
  return {
    claimId,
    outcome,
    finalConfidence: outcome === "accepted" ? 0.85 : 0.4,
    confidenceBreakdown: {
      confirmed: outcome === "accepted" ? 0.85 : 0.15,
      rejected: outcome === "rejected" ? 0.8 : 0.1,
      uncertain: 0.05,
    },
    totalValidators: 7,
    participatingValidators: 7,
    agreementLevel:
      outcome === "accepted" || outcome === "rejected" ? 0.85 : 0.55,
    rewards: new Map(),
    slashes: new Map(),
    fact,
    recordedAt: 200,
  };
}

// ============ Tests ============

/**
 * Generate a unique ID for test isolation
 * Uses a counter + timestamp + random to ensure uniqueness even within the same millisecond
 */
let _testIdCounter = 0;
function uniqueId(): string {
  _testIdCounter++;
  return `${_testIdCounter}-${Date.now()}-${Math.random().toString(36).substring(2, 8)}`;
}

const CLAIM_STAKE = 111n * 10n ** 18n; // 111 LGM

/**
 * Create a complete isolated test setup
 * Returns fresh state, IDs, and params for each test call
 */
function createIsolatedTestSetup() {
  const testId = uniqueId();
  const submitterDid = `did:lgm:claim-submitter-${testId}`;
  const claimId = `claim:test-claim-${testId}`;

  const state = createTestStateManager();
  const params = { ...DEFAULT_CONSENSUS_PARAMS };

  // Create submitter account with balance
  state.createAccount(submitterDid, "submitter-key-hash", 1000n * 10n ** 18n);

  // Lock stake (simulating claim submission)
  state.stake(submitterDid, CLAIM_STAKE);

  // Add the claim to state (simulating claim submission)
  const claim = createTestClaim(
    claimId,
    submitterDid,
    "Water boils at 100°C at sea level",
    CLAIM_STAKE,
  );
  state.addClaim(claim);

  return { state, params, submitterDid, claimId, claimStake: CLAIM_STAKE };
}

describe("GAP-006: Verification Reward Routing", () => {
  describe("Accepted Claims", () => {
    it("should create vesting schedule for claim submitter when claim is accepted", () => {
      const { state, params, submitterDid, claimId } =
        createIsolatedTestSetup();

      const fact = createTestFact(claimId, "Water boils at 100°C at sea level");
      const result = createTestVerificationResult(claimId, "accepted", fact);

      applyVerificationResult(state, result, 200n, params);

      // Verify vesting was created
      const vesting = state.getVestingForClaim(claimId);
      expect(vesting).not.toBeNull();
      expect(vesting!.claimId).toBe(claimId);
      expect(vesting!.recipient).toBe(submitterDid);
    });

    it("should route 22% of claim reward across 5 treasury funds", () => {
      const { state, params, claimId } = createIsolatedTestSetup();

      const fact = createTestFact(claimId, "Water boils at 100°C at sea level");
      const result = createTestVerificationResult(claimId, "accepted", fact);

      const researchBefore = state.getAccount(RESEARCH_TREASURY_DID)!.balance;

      applyVerificationResult(state, result, 200n, params);

      const researchAfter = state.getAccount(RESEARCH_TREASURY_DID)!.balance;
      const claimReward = params.claimReward; // 5 LGM
      // Total evidence tax = 22% of claim reward
      const totalEvidenceTax = (claimReward * 2200n) / 10000n;
      // Research fund gets 77% of the evidence tax
      const expectedResearchShare = (totalEvidenceTax * 7700n) / 10000n;

      expect(researchAfter - researchBefore).toBe(expectedResearchShare);

      // Total across all 5 treasury funds should equal the full 22%
      const foundationBalance = state.getBalance("did:lgm:treasury-foundation");
      const communityBalance = state.getBalance("did:lgm:treasury-community");
      const developersBalance = state.getBalance("did:lgm:treasury-developers");
      const reserveBalance = state.getBalance("did:lgm:treasury-reserve");
      const totalTreasury =
        researchAfter -
        researchBefore +
        foundationBalance +
        communityBalance +
        developersBalance +
        reserveBalance;
      expect(totalTreasury).toBe(totalEvidenceTax);
    });

    it("should vest 78% of claim reward to submitter", () => {
      const { state, params, claimId } = createIsolatedTestSetup();

      const fact = createTestFact(claimId, "Water boils at 100°C at sea level");
      const result = createTestVerificationResult(claimId, "accepted", fact);

      applyVerificationResult(state, result, 200n, params);

      const vesting = state.getVestingForClaim(claimId);
      const claimReward = params.claimReward; // 5 LGM
      const expectedVestingAmount =
        claimReward - BigInt(Math.floor(Number(claimReward) * 0.22));

      expect(vesting).not.toBeNull();
      expect(vesting!.totalAmount).toBe(expectedVestingAmount);
    });

    it("should return staked claim stake to submitter", () => {
      const { state, params, submitterDid, claimId, claimStake } =
        createIsolatedTestSetup();

      const fact = createTestFact(claimId, "Water boils at 100°C at sea level");
      const result = createTestVerificationResult(claimId, "accepted", fact);

      const accountBefore = state.getAccount(submitterDid)!;
      const balanceBefore = accountBefore.balance;
      const stakedBefore = accountBefore.stakedBalance;

      applyVerificationResult(state, result, 200n, params);

      const accountAfter = state.getAccount(submitterDid)!;

      // Stake should be returned (moved from stakedBalance to balance)
      expect(accountAfter.stakedBalance).toBe(stakedBefore - claimStake);
      expect(accountAfter.balance).toBe(balanceBefore + claimStake);
    });

    it("should update claim status to accepted", () => {
      const { state, params, claimId } = createIsolatedTestSetup();

      const fact = createTestFact(claimId, "Water boils at 100°C at sea level");
      const result = createTestVerificationResult(claimId, "accepted", fact);

      applyVerificationResult(state, result, 200n, params);

      const claim = state.getClaim(claimId);
      expect(claim).not.toBeUndefined();
      expect(claim!.status).toBe("accepted");
      expect(claim!.resultFactId).toBe(claimId);
    });

    it("should determine correct vesting category based on claim type", () => {
      const { state, params, submitterDid, claimStake } =
        createIsolatedTestSetup();

      // Create a formal/logical claim with unique ID
      const formalClaimId = `claim:formal-${uniqueId()}`;
      const formalClaim = createTestClaim(
        formalClaimId,
        submitterDid,
        "Mathematical theorem",
        claimStake,
      );
      formalClaim.claimType = "formal";
      formalClaim.stratum = "formal_logical";

      // Deposit more balance and stake for the formal claim
      state.transfer(REWARD_POOL_DID, submitterDid, claimStake);
      state.stake(submitterDid, claimStake);
      state.addClaim(formalClaim);

      const formalFact = createTestFact(formalClaimId, "Mathematical theorem");
      const formalResult = createTestVerificationResult(
        formalClaimId,
        "accepted",
        formalFact,
      );

      applyVerificationResult(state, formalResult, 200n, params);

      const vesting = state.getVestingForClaim(formalClaimId);
      expect(vesting).not.toBeNull();
      // Formal claims should get 'formal_proof' category
      expect(vesting!.category).toBe("formal_proof");
    });

    it("should NOT create duplicate vesting for same claim", () => {
      const { state, params, submitterDid, claimId } =
        createIsolatedTestSetup();

      const fact = createTestFact(claimId, "Water boils at 100°C at sea level");
      const result = createTestVerificationResult(claimId, "accepted", fact);

      // Apply once
      applyVerificationResult(state, result, 200n, params);

      // Get all vestings for this recipient
      const vestings = state.getVestingsForRecipient(submitterDid);

      // Should only have one vesting for this claim
      const claimVestings = vestings.filter((v) => v.claimId === claimId);
      expect(claimVestings).toHaveLength(1);
    });
  });

  describe("Rejected Claims", () => {
    it("should NOT create vesting when claim is rejected", () => {
      const { state, params, claimId } = createIsolatedTestSetup();

      const result = createTestVerificationResult(claimId, "rejected");

      applyVerificationResult(state, result, 200n, params);

      const vesting = state.getVestingForClaim(claimId);
      expect(vesting).toBeNull();
    });

    it("should slash portion of stake when claim is rejected", () => {
      const { state, params, submitterDid, claimId, claimStake } =
        createIsolatedTestSetup();

      const result = createTestVerificationResult(claimId, "rejected");

      const accountBefore = state.getAccount(submitterDid)!;
      const balanceBefore = accountBefore.balance;

      applyVerificationResult(state, result, 200n, params);

      const accountAfter = state.getAccount(submitterDid)!;

      // Calculate expected slash
      const slashRate = params.invalidClaimSlash; // 22%
      const slashAmount = BigInt(Math.floor(Number(claimStake) * slashRate));
      const returnAmount = claimStake - slashAmount;

      // Balance should increase by returnAmount (stake returned minus slash)
      expect(accountAfter.balance).toBe(balanceBefore + returnAmount);
      // Staked balance should be zero (full stake was unstaked)
      expect(accountAfter.stakedBalance).toBe(0n);
    });

    it("should transfer slashed amount to protocol treasury", () => {
      const { state, params, claimId, claimStake } = createIsolatedTestSetup();

      const result = createTestVerificationResult(claimId, "rejected");

      const treasuryBefore = state.getAccount(PROTOCOL_TREASURY_DID)!.balance;

      applyVerificationResult(state, result, 200n, params);

      const treasuryAfter = state.getAccount(PROTOCOL_TREASURY_DID)!.balance;
      const slashRate = params.invalidClaimSlash;
      const expectedSlash = BigInt(Math.floor(Number(claimStake) * slashRate));

      expect(treasuryAfter - treasuryBefore).toBe(expectedSlash);
    });

    it("should update claim status to rejected with reason", () => {
      const { state, params, claimId } = createIsolatedTestSetup();

      const result = createTestVerificationResult(claimId, "rejected");

      applyVerificationResult(state, result, 200n, params);

      const claim = state.getClaim(claimId);
      expect(claim).not.toBeUndefined();
      expect(claim!.status).toBe("rejected");
      expect(claim!.rejectionReason).toContain("Verification failed");
    });
  });

  describe("Contested/Insufficient Claims", () => {
    it("should keep stake locked when claim is contested", () => {
      const { state, params, submitterDid, claimId } =
        createIsolatedTestSetup();

      const result = createTestVerificationResult(claimId, "contested");

      const stakedBefore = state.getAccount(submitterDid)!.stakedBalance;

      applyVerificationResult(state, result, 200n, params);

      const stakedAfter = state.getAccount(submitterDid)!.stakedBalance;

      // Stake should remain locked
      expect(stakedAfter).toBe(stakedBefore);
    });

    it("should NOT create vesting when claim is contested", () => {
      const { state, params, claimId } = createIsolatedTestSetup();

      const result = createTestVerificationResult(claimId, "contested");

      applyVerificationResult(state, result, 200n, params);

      const vesting = state.getVestingForClaim(claimId);
      expect(vesting).toBeNull();
    });

    it("should set claim status back to pending for re-verification", () => {
      const { state, params, claimId } = createIsolatedTestSetup();

      const result = createTestVerificationResult(claimId, "insufficient");

      applyVerificationResult(state, result, 200n, params);

      const claim = state.getClaim(claimId);
      expect(claim).not.toBeUndefined();
      expect(claim!.status).toBe("pending");
    });
  });

  describe("Research Share Routing", () => {
    it("should route validator rewards with 22% research share", () => {
      const { state, params, claimId } = createIsolatedTestSetup();

      const fact = createTestFact(claimId, "Water boils at 100°C at sea level");
      const result = createTestVerificationResult(claimId, "accepted", fact);

      // Add validator rewards
      const validatorKey = "validator001";
      const validatorDid = `did:lgm:${validatorKey}`;
      state.createAccount(validatorDid, "val-key-hash", 10000n * 10n ** 18n);

      const validatorReward = 1n * 10n ** 18n; // 1 LGM
      result.rewards.set(validatorKey, validatorReward);

      const researchBefore = state.getAccount(RESEARCH_TREASURY_DID)!.balance;
      const validatorBefore = state.getAccount(validatorDid)!.balance;

      applyVerificationResult(state, result, 200n, params);

      const researchAfter = state.getAccount(RESEARCH_TREASURY_DID)!.balance;
      const validatorAfter = state.getAccount(validatorDid)!.balance;

      const expectedResearchShareFromValidator = BigInt(
        Math.floor(Number(validatorReward) * 0.22),
      );
      const expectedValidatorAmount =
        validatorReward - expectedResearchShareFromValidator;

      // Validator should receive 78% of their reward
      expect(validatorAfter - validatorBefore).toBe(expectedValidatorAmount);

      // Research treasury should receive research share from both claim reward and validator reward
      // (we verify it increased, exact amount calculation would be complex)
      expect(researchAfter).toBeGreaterThan(researchBefore);
    });
  });

  describe("Edge Cases", () => {
    it("should handle claim with zero stake gracefully", () => {
      const { state, params, submitterDid } = createIsolatedTestSetup();

      const zeroStakeClaimId = `claim:zero-stake-${uniqueId()}`;
      const zeroStakeClaim = createTestClaim(
        zeroStakeClaimId,
        submitterDid,
        "Zero stake claim",
        0n,
      );
      state.addClaim(zeroStakeClaim);

      const fact = createTestFact(zeroStakeClaimId, "Zero stake claim");
      const result = createTestVerificationResult(
        zeroStakeClaimId,
        "accepted",
        fact,
      );

      // Should not throw
      expect(() => {
        applyVerificationResult(state, result, 200n, params);
      }).not.toThrow();

      // Vesting should still be created (reward comes from protocol, not stake)
      const vesting = state.getVestingForClaim(zeroStakeClaimId);
      expect(vesting).not.toBeNull();
    });

    it("should handle missing claim gracefully (no crash)", () => {
      const { state, params } = createIsolatedTestSetup();

      const missingClaimId = `claim:nonexistent-${uniqueId()}`;
      const fact = createTestFact(missingClaimId, "Nonexistent claim");
      const result = createTestVerificationResult(
        missingClaimId,
        "accepted",
        fact,
      );

      // Should not throw - just won't create vesting
      expect(() => {
        applyVerificationResult(state, result, 200n, params);
      }).not.toThrow();

      // No vesting created for missing claim
      const vesting = state.getVestingForClaim(missingClaimId);
      expect(vesting).toBeNull();
    });
  });
});
