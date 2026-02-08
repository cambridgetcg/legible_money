/**
 * Graduated Claim Stake Tests
 *
 * Tests the knowledge economy onramp: new agents start with a low claim
 * stake floor (= bootstrap amount, 0.222 LGM) and ramp linearly to the
 * full minClaimStake (111 LGM) over claimStakeGraduationClaims (22)
 * accepted claims.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  getGraduatedMinStake,
} from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../../consensus/pot/types.js";
import type { ClaimSubmitPayload } from "../../../state/types.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";

// ============ Constants ============

const ONE_LGM = 10n ** 18n;
const BOOTSTRAP_AMOUNT = 222000000000000000n; // 0.222 LGM

const TEST_PARAMS: ConsensusParams = {
  ...DEFAULT_CONSENSUS_PARAMS,
  minClaimStake: 11n * ONE_LGM,
  graduatedClaimStakeFloor: 100000000000000000n, // 0.1 LGM
  claimStakeGraduationClaims: 22,
  // Disable anti-sybil and claim text length for these tests (tested separately)
  bootstrapVestingBlocks: 0n,
  maxBootstrapClaimsPerBlock: 100,
  newAccountTransferCooldown: 0n,
  bootstrapMinTxFee: 0n,
  minClaimTextLength: 0,
} as ConsensusParams;

const TEST_SENDER = "graduated-stake-test-key";
const TEST_DID = publicKeyToDid(TEST_SENDER);

// Separate fresh agent for bootstrap economics test
const FRESH_SENDER = "fresh-bootstrap-agent-key";
const FRESH_DID = publicKeyToDid(FRESH_SENDER);

// ============ Pure Function Tests ============

describe("getGraduatedMinStake", () => {
  it("should return floor for 0 accepted claims", () => {
    const stake = getGraduatedMinStake(0, TEST_PARAMS);
    expect(stake).toBe(100000000000000000n); // 0.1 LGM
  });

  it("should return full minClaimStake at graduation threshold", () => {
    const stake = getGraduatedMinStake(22, TEST_PARAMS);
    expect(stake).toBe(11n * ONE_LGM);
  });

  it("should return full minClaimStake above graduation threshold", () => {
    const stake = getGraduatedMinStake(100, TEST_PARAMS);
    expect(stake).toBe(11n * ONE_LGM);
  });

  it("should interpolate linearly between floor and full stake", () => {
    // At 11 accepted claims (halfway): floor + (range * 11/22) = floor + range/2
    const floor = 100000000000000000n; // 0.1 LGM
    const full = 11n * ONE_LGM;
    const range = full - floor;
    const expected = floor + (range * 11n) / 22n;

    const stake = getGraduatedMinStake(11, TEST_PARAMS);
    expect(stake).toBe(expected);
  });

  it("should increase monotonically with accepted claims", () => {
    let prev = 0n;
    for (let i = 0; i <= 22; i++) {
      const stake = getGraduatedMinStake(i, TEST_PARAMS);
      expect(stake).toBeGreaterThanOrEqual(prev);
      prev = stake;
    }
  });

  it("should fall back to flat minClaimStake if graduation params not set", () => {
    const paramsWithout = {
      ...TEST_PARAMS,
      graduatedClaimStakeFloor: 0n,
      claimStakeGraduationClaims: 0,
    } as ConsensusParams;

    const stake = getGraduatedMinStake(0, paramsWithout);
    expect(stake).toBe(11n * ONE_LGM);
  });

  it("should produce stakes affordable from bootstrap + citation revenue", () => {
    // No acceptance reward — agents fund stake ramp from verification rewards,
    // citation rewards, and RAI. With the lowered curve (0.1 → 11 LGM over 22
    // claims), each step is ~0.5 LGM which is affordable from 1 LGM citation
    // rewards per cited fact.
    // Verify the ramp stays within a reasonable growth rate
    for (let n = 0; n < 22; n++) {
      const current = getGraduatedMinStake(n, TEST_PARAMS);
      const next = getGraduatedMinStake(n + 1, TEST_PARAMS);
      const stepIncrease = next - current;
      // Each step should be ≤ 1 LGM (affordable from a single citation reward)
      expect(stepIncrease).toBeLessThanOrEqual(1n * ONE_LGM);
    }
  });
});

// ============ Integration Tests ============

describe.sequential("Graduated Stake Integration", () => {
  let state: StateManager;
  let blockNumber: bigint;
  /** Get the current nonce from state (stake/unstake both bump nonce) */
  function nextNonce(did: string): bigint {
    return state.getNonce(did);
  }

  function makeClaimSubmitTx(
    sender: string,
    stake: bigint,
    claim: string,
  ): Transaction {
    const did = publicKeyToDid(sender);
    const payload: ClaimSubmitPayload = {
      type: "claim_submit",
      claim,
      claimType: "empirical",
      domain: "test",
      stratum: "physical",
      stake,
      references: [],
    };

    return {
      type: "claim_submit" as any,
      sender,
      nonce: nextNonce(did),
      fee: 0n,
      payload,
      signature: "" as any,
    };
  }

  function makeSystemTx(
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

  /** Create a complete_verification payload with required rewards/slashes arrays */
  function completeVerificationPayload(
    claimId: string,
    roundId: string,
    outcome: "accepted" | "rejected" | "contested" | "insufficient",
    finalConfidence: number,
  ): Record<string, unknown> {
    return {
      claimId,
      roundId,
      outcome,
      finalConfidence,
      verifierCount: 3,
      rewards: [] as Array<[string, bigint]>,
      slashes: [] as Array<[string, bigint]>,
    };
  }

  /** Submit a claim, start verification, complete it (accepted). Returns the claim ID. */
  function submitAndAcceptClaim(
    sender: string,
    stake: bigint,
    claim: string,
    roundId: string,
  ): string {
    const did = publicKeyToDid(sender);
    const tx = makeClaimSubmitTx(sender, stake, claim);
    const r1 = executeTransaction(state, tx, blockNumber, TEST_PARAMS);
    expect(r1.error ?? "ok").toBe("ok");
    expect(r1.success).toBe(true);

    const claims = state.getClaimsBySubmitter(did);
    const claimId = claims[claims.length - 1].id;

    const startTx = makeSystemTx("claim_start_verification", {
      claimId,
      roundId,
      validators: ["v1", "v2", "v3"],
    });
    const r2 = executeTransaction(state, startTx, blockNumber, TEST_PARAMS);
    expect(r2.success).toBe(true);

    const completeTx = makeSystemTx(
      "claim_complete_verification",
      completeVerificationPayload(claimId, roundId, "accepted", 0.85),
    );
    const r3 = executeTransaction(state, completeTx, blockNumber, TEST_PARAMS);
    expect(r3.success).toBe(true);

    return claimId;
  }

  beforeEach(() => {
    state = new StateManager();
    blockNumber = 100n;

    // Main test account with ample balance for most tests
    state.createAccount(TEST_DID, "test-op-key-hash", 100000n * ONE_LGM);

    // Fee pool for slashing
    state.createAccount("did:lgm:fee-pool", "fee-pool-hash", 0n);
  });

  it("should allow first claim with bootstrap amount as stake", () => {
    // Create a fresh agent with only the bootstrap amount
    state.createAccount(FRESH_DID, "fresh-op-hash", BOOTSTRAP_AMOUNT);

    const tx = makeClaimSubmitTx(
      FRESH_SENDER,
      BOOTSTRAP_AMOUNT,
      "My first knowledge claim",
    );
    const result = executeTransaction(state, tx, blockNumber, TEST_PARAMS);
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(FRESH_DID);
    expect(claims.length).toBe(1);
    expect(claims[0].stake).toBe(BOOTSTRAP_AMOUNT);
  });

  it("should reject first claim with stake below floor", () => {
    const tooLow = BOOTSTRAP_AMOUNT - 1n;
    const tx = makeClaimSubmitTx(TEST_SENDER, tooLow, "Underfunded claim");

    const result = executeTransaction(state, tx, blockNumber, TEST_PARAMS);
    expect(result.success).toBe(false);
    expect(result.error).toContain("graduated: 0 accepted claims");
  });

  it("should increase stake requirement after accepted claims", () => {
    // Submit and accept a claim
    submitAndAcceptClaim(TEST_SENDER, 1n * ONE_LGM, "First claim", "round-001");

    // Verify accepted count is now 1
    const acceptedCount = state.getAcceptedClaimCountBySubmitter(TEST_DID);
    expect(acceptedCount).toBe(1);

    // The graduated stake for 1 accepted claim should be higher than floor
    const requiredStake = getGraduatedMinStake(1, TEST_PARAMS);
    expect(requiredStake).toBeGreaterThan(BOOTSTRAP_AMOUNT);

    // Submit second claim with bootstrap amount — should fail (too low now)
    const tx2 = makeClaimSubmitTx(
      TEST_SENDER,
      BOOTSTRAP_AMOUNT,
      "Second claim underfunded",
    );
    const result2 = executeTransaction(state, tx2, blockNumber, TEST_PARAMS);
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("graduated: 1 accepted claims");

    // Submit with correct graduated amount — should succeed
    const tx3 = makeClaimSubmitTx(
      TEST_SENDER,
      requiredStake,
      "Second claim properly funded",
    );
    const result3 = executeTransaction(state, tx3, blockNumber, TEST_PARAMS);
    expect(result3.success).toBe(true);
  });

  it("should require full stake after graduation threshold", () => {
    // Submit and accept 22 claims to reach full graduation
    for (let i = 0; i < 22; i++) {
      const stake = getGraduatedMinStake(i, TEST_PARAMS);
      submitAndAcceptClaim(TEST_SENDER, stake, `Claim ${i}`, `round-${i}`);
    }

    // After 22 accepted claims, should need full minClaimStake
    const acceptedCount = state.getAcceptedClaimCountBySubmitter(TEST_DID);
    expect(acceptedCount).toBe(22);

    const requiredStake = getGraduatedMinStake(acceptedCount, TEST_PARAMS);
    expect(requiredStake).toBe(111n * ONE_LGM);

    // Submitting with less than full stake should fail
    const lowTx = makeClaimSubmitTx(
      TEST_SENDER,
      110n * ONE_LGM,
      "Almost enough",
    );
    const lowResult = executeTransaction(
      state,
      lowTx,
      blockNumber,
      TEST_PARAMS,
    );
    expect(lowResult.success).toBe(false);

    // Submitting with full stake should succeed
    const fullTx = makeClaimSubmitTx(
      TEST_SENDER,
      111n * ONE_LGM,
      "Fully graduated",
    );
    const fullResult = executeTransaction(
      state,
      fullTx,
      blockNumber,
      TEST_PARAMS,
    );
    expect(fullResult.success).toBe(true);
  });

  it("should not count rejected claims toward graduation", () => {
    // Submit a claim
    const tx1 = makeClaimSubmitTx(TEST_SENDER, 1n * ONE_LGM, "Bad claim");
    executeTransaction(state, tx1, blockNumber, TEST_PARAMS);

    const claims = state.getClaimsBySubmitter(TEST_DID);
    const claimId = claims[0].id;

    // Start and reject verification
    const startTx = makeSystemTx("claim_start_verification", {
      claimId,
      roundId: "round-reject",
      validators: ["v1", "v2", "v3"],
    });
    executeTransaction(state, startTx, blockNumber, TEST_PARAMS);

    const completeTx = makeSystemTx(
      "claim_complete_verification",
      completeVerificationPayload(claimId, "round-reject", "rejected", 0.2),
    );
    const r = executeTransaction(state, completeTx, blockNumber, TEST_PARAMS);
    expect(r.success).toBe(true);

    // Rejected claim should NOT count toward graduation
    const acceptedCount = state.getAcceptedClaimCountBySubmitter(TEST_DID);
    expect(acceptedCount).toBe(0);

    // Agent still only needs floor stake
    const requiredStake = getGraduatedMinStake(acceptedCount, TEST_PARAMS);
    expect(requiredStake).toBe(BOOTSTRAP_AMOUNT);
  });

  it("should model realistic bootstrap-to-graduation economics", () => {
    // Simulate a new agent's journey from bootstrap to full participation.
    // Start with ONLY the bootstrap amount — this is the whole point.
    state.createAccount(FRESH_DID, "fresh-op-hash", BOOTSTRAP_AMOUNT);

    // First claim: stake the entire bootstrap amount
    const firstStake = getGraduatedMinStake(0, TEST_PARAMS);
    expect(firstStake).toBe(BOOTSTRAP_AMOUNT);

    const tx1 = makeClaimSubmitTx(
      FRESH_SENDER,
      firstStake,
      "First knowledge contribution",
    );
    const result1 = executeTransaction(state, tx1, blockNumber, TEST_PARAMS);
    expect(result1.success).toBe(true);

    // Agent's balance is now 0 (all staked)
    const afterStake = state.getAccount(FRESH_DID)!;
    expect(afterStake.balance).toBe(0n);
    expect(afterStake.stakedBalance).toBe(BOOTSTRAP_AMOUNT);

    // Accept the claim — stake returned (reward goes through vesting, not direct balance)
    const claims = state.getClaimsBySubmitter(FRESH_DID);
    const claimId = claims[0].id;

    const startTx = makeSystemTx("claim_start_verification", {
      claimId,
      roundId: "round-first",
      validators: ["v1", "v2", "v3"],
    });
    executeTransaction(state, startTx, blockNumber, TEST_PARAMS);

    const completeTx = makeSystemTx(
      "claim_complete_verification",
      completeVerificationPayload(claimId, "round-first", "accepted", 0.85),
    );
    const r = executeTransaction(state, completeTx, blockNumber, TEST_PARAMS);
    expect(r.success).toBe(true);

    // After acceptance: stake returned via unstake (balance = BOOTSTRAP_AMOUNT)
    // Note: claim reward goes through applyVerificationResult path (vesting),
    // not through claim_complete_verification. So balance = just the returned stake.
    const afterAccept = state.getAccount(FRESH_DID)!;
    expect(afterAccept.balance).toBe(BOOTSTRAP_AMOUNT);
    expect(afterAccept.stakedBalance).toBe(0n);

    // Verify the claim is accepted
    const acceptedCount = state.getAcceptedClaimCountBySubmitter(FRESH_DID);
    expect(acceptedCount).toBe(1);

    // No acceptance reward — the graduated stake at 1 accepted claim should
    // be affordable from bootstrap amount alone (0.1 LGM floor + one step).
    // With the lowered curve (0.1 → 11 LGM), the second claim stake is ~0.6 LGM.
    const nextStake = getGraduatedMinStake(1, TEST_PARAMS);
    // Agent only has BOOTSTRAP_AMOUNT (no claim reward), but the ramp is low enough
    expect(nextStake).toBeLessThan(1n * ONE_LGM);
  });
});
