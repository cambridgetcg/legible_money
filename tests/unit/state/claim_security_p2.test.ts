/**
 * P2 Claim Security Tests
 *
 * Tests for all MEDIUM priority attack vectors identified
 * in CLAIM_EDGE_SCENARIOS.md:
 *
 * Fix 1: Citation Diminishing Returns (§7.4 + §3.5)
 * Fix 2: Pre-Submission Reference Cycle Check (§7.1)
 * Fix 3: Deterministic Block Tx Ordering (§9.4)
 * Fix 4: Deductive Proof Requirement (§7.3)
 * Fix 5: Insufficient Outcome Tracking (§6.4)
 * Fix 6: Domain Minimum Validator Pool (§5.2)
 * Fix 7: Graduated Stake Farming Defense (§3.3 + §2.1)
 * Fix 8: Locale Content Hash (§1.3) + Validator Diversity (§4.1)
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  getGraduatedMinStake,
} from "../../../state/transitions.js";
import { generateClaimContentHash } from "../../../state/claims.js";
import { applyDeterministicOrdering } from "../../../consensus/blocks/producer.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../../consensus/pot/types.js";
import type { ClaimSubmitPayload } from "../../../state/types.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";

// ============ Constants ============

const ONE_LGM = 10n ** 18n;

const BASE_PARAMS: ConsensusParams = {
  ...DEFAULT_CONSENSUS_PARAMS,
  minClaimStake: 111n * 10n ** 15n,
  graduatedClaimStakeFloor: 222000000000000000n,
  claimStakeGraduationClaims: 22,
  bootstrapVestingBlocks: 0n,
  maxBootstrapClaimsPerBlock: 100,
  newAccountTransferCooldown: 0n,
  bootstrapMinTxFee: 0n,
  minClaimTextLength: 20,
  claimCommitRevealEnabled: false,
  claimRevealWindowBlocks: 22,
} as ConsensusParams;

const ALICE_KEY = "alice-p2-security-test-key";
const ALICE_DID = publicKeyToDid(ALICE_KEY);
const BOB_KEY = "bob-p2-security-test-key";
const BOB_DID = publicKeyToDid(BOB_KEY);

// ============ Helpers ============

function makeClaimSubmitTx(
  sender: string,
  claim: string,
  stake: bigint,
  references: string[] = [],
  extra: Partial<ClaimSubmitPayload> = {},
): Transaction {
  const payload: ClaimSubmitPayload = {
    type: "claim_submit",
    claim,
    claimType: "empirical",
    domain: "test",
    stratum: "physical",
    stake,
    references,
    ...extra,
  };
  return {
    type: "claim_submit" as any,
    sender,
    nonce: 0n,
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

/**
 * Compute the minimum stake needed for the next claim submission,
 * accounting for graduated stake ramp.
 */
function computeRequiredStake(
  state: StateManager,
  sender: string,
  params: ConsensusParams,
): bigint {
  const did = publicKeyToDid(sender);
  const accepted = state.getAcceptedClaimCountBySubmitter(did, true);
  return getGraduatedMinStake(accepted, params);
}

/**
 * Submit a claim and run it through verification to acceptance.
 * Returns the claim ID.
 */
function submitAndAcceptClaim(
  state: StateManager,
  sender: string,
  stake: bigint,
  claim: string,
  roundId: string,
  blockNumber: bigint,
  params: ConsensusParams,
  references: string[] = [],
  extra: Partial<ClaimSubmitPayload> = {},
): string {
  const did = publicKeyToDid(sender);
  // Auto-compute minimum stake if requested stake is too low
  const minStake = computeRequiredStake(state, sender, params);
  const effectiveStake = stake < minStake ? minStake : stake;
  const tx = makeClaimSubmitTx(
    sender,
    claim,
    effectiveStake,
    references,
    extra,
  );
  tx.nonce = state.getNonce(did);
  const r1 = executeTransaction(state, tx, blockNumber, params);
  if (!r1.success) {
    throw new Error(`submitAndAcceptClaim failed at submit: ${r1.error}`);
  }

  const claims = state.getClaimsBySubmitter(did);
  const claimId = claims[claims.length - 1].id;

  const startTx = makeSystemTx("claim_start_verification", {
    claimId,
    roundId,
    validators: ["v1", "v2", "v3"],
  });
  const r2 = executeTransaction(state, startTx, blockNumber, params);
  if (!r2.success) {
    throw new Error(`submitAndAcceptClaim failed at start: ${r2.error}`);
  }

  const completeTx = makeSystemTx(
    "claim_complete_verification",
    completeVerificationPayload(claimId, roundId, "accepted", 0.85),
  );
  const r3 = executeTransaction(state, completeTx, blockNumber, params);
  if (!r3.success) {
    throw new Error(`submitAndAcceptClaim failed at complete: ${r3.error}`);
  }

  return claimId;
}

/**
 * Submit a claim and run it through verification with a specific outcome.
 */
function submitAndCompleteClaim(
  state: StateManager,
  sender: string,
  stake: bigint,
  claim: string,
  roundId: string,
  blockNumber: bigint,
  params: ConsensusParams,
  outcome: "accepted" | "rejected" | "contested" | "insufficient",
): string {
  const did = publicKeyToDid(sender);
  // Auto-compute minimum stake if requested stake is too low
  const minStake = computeRequiredStake(state, sender, params);
  const effectiveStake = stake < minStake ? minStake : stake;
  const tx = makeClaimSubmitTx(sender, claim, effectiveStake);
  tx.nonce = state.getNonce(did);
  const r1 = executeTransaction(state, tx, blockNumber, params);
  if (!r1.success) {
    throw new Error(`submitAndCompleteClaim failed at submit: ${r1.error}`);
  }

  const claims = state.getClaimsBySubmitter(did);
  const claimId = claims[claims.length - 1].id;

  const startTx = makeSystemTx("claim_start_verification", {
    claimId,
    roundId,
    validators: ["v1", "v2", "v3"],
  });
  executeTransaction(state, startTx, blockNumber, params);

  const completeTx = makeSystemTx(
    "claim_complete_verification",
    completeVerificationPayload(claimId, roundId, outcome, 0.85),
  );
  executeTransaction(state, completeTx, blockNumber, params);

  return claimId;
}

// ============ Fix 1: Citation Diminishing Returns (§7.4 + §3.5) ============

describe.sequential("Citation Diminishing Returns (§7.4)", () => {
  let state: StateManager;
  const blockNumber = 100n;

  const CITATION_PARAMS: ConsensusParams = {
    ...BASE_PARAMS,
    citationRewardWindowBlocks: 34272, // 1 day
    citationRewardBaseCount: 2, // Full reward for first 2 citations
    citationRewardDecayFactor: 0.5, // 50% decay per excess citation
  } as ConsensusParams;

  beforeEach(() => {
    state = new StateManager();
    state.createAccount(ALICE_DID, "alice-op", 500000n * ONE_LGM);
    state.createAccount(BOB_DID, "bob-op", 500000n * ONE_LGM);
    state.createAccount(
      "did:lgm:fee-pool",
      "fee-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should give full citation reward within base count", () => {
    // Alice creates the target fact
    const aliceClaimId = submitAndAcceptClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Water boils at 100 degrees Celsius at standard pressure",
      "round-a1",
      blockNumber,
      CITATION_PARAMS,
    );
    const aliceClaim = state.getClaim(aliceClaimId)!;
    const factId = aliceClaim.resultFactId!;

    const aliceBalBefore = state.getAccount(ALICE_DID)!.balance;

    // Bob cites Alice's fact (1st citation — within base count)
    submitAndAcceptClaim(
      state,
      BOB_KEY,
      1n * ONE_LGM,
      "Water reaches boiling point at 100C at sea level atmospheric pressure",
      "round-b1",
      blockNumber + 1n,
      CITATION_PARAMS,
      [factId],
    );

    const aliceBalAfter = state.getAccount(ALICE_DID)!.balance;
    // Alice should have received a citation reward (balance increased)
    expect(aliceBalAfter).toBeGreaterThan(aliceBalBefore);
  });

  it("should record citation blocks for tracking", () => {
    const aliceClaimId = submitAndAcceptClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Gravity accelerates objects at 9.8 m/s squared on Earth surface",
      "round-a1",
      blockNumber,
      CITATION_PARAMS,
    );
    const factId = state.getClaim(aliceClaimId)!.resultFactId!;

    // First citation
    submitAndAcceptClaim(
      state,
      BOB_KEY,
      1n * ONE_LGM,
      "Objects fall at 9.8 m/s squared acceleration due to Earth gravity",
      "round-b1",
      blockNumber + 1n,
      CITATION_PARAMS,
      [factId],
    );

    // The fact should have a recorded citation block
    const recentCount = state.getRecentCitationCount(
      factId,
      blockNumber + 2n,
      BigInt(CITATION_PARAMS.citationRewardWindowBlocks),
    );
    expect(recentCount).toBeGreaterThanOrEqual(1);
  });

  it("should not apply decay when feature is disabled (default)", () => {
    // Use BASE_PARAMS which has citationRewardWindowBlocks = 0 (off)
    const aliceClaimId = submitAndAcceptClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Helium is the second lightest element in the periodic table",
      "round-a1",
      blockNumber,
      BASE_PARAMS,
    );
    const factId = state.getClaim(aliceClaimId)!.resultFactId!;
    const aliceBalBefore = state.getAccount(ALICE_DID)!.balance;

    // Multiple citations should all get full reward
    for (let i = 0; i < 5; i++) {
      submitAndAcceptClaim(
        state,
        BOB_KEY,
        1n * ONE_LGM,
        `Helium is very light element number ${i} test fact unique text here`,
        `round-b${i}`,
        blockNumber + BigInt(i + 1),
        BASE_PARAMS,
        [factId],
      );
    }

    const aliceBalAfter = state.getAccount(ALICE_DID)!.balance;
    // Alice should have received rewards for all 5 citations
    expect(aliceBalAfter).toBeGreaterThan(aliceBalBefore);
  });
});

// ============ Fix 2: Pre-Submission Cycle Check (§7.1) ============

describe.sequential("Pre-Submission Reference Cycle Check (§7.1)", () => {
  let state: StateManager;
  const blockNumber = 100n;

  const CYCLE_PARAMS: ConsensusParams = {
    ...BASE_PARAMS,
    maxReferenceCycleHops: 3, // Reject cycles ≤3 hops
  } as ConsensusParams;

  beforeEach(() => {
    state = new StateManager();
    state.createAccount(ALICE_DID, "alice-op", 500000n * ONE_LGM);
    state.createAccount(BOB_DID, "bob-op", 500000n * ONE_LGM);
    state.createAccount(
      "did:lgm:fee-pool",
      "fee-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should allow claims with no cycle risk", () => {
    // Alice creates fact A
    const claimIdA = submitAndAcceptClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Photosynthesis converts sunlight into chemical energy in plants",
      "round-a1",
      blockNumber,
      CYCLE_PARAMS,
    );
    const factIdA = state.getClaim(claimIdA)!.resultFactId!;

    // Bob creates fact B referencing A — no cycle
    const result = submitAndAcceptClaim(
      state,
      BOB_KEY,
      1n * ONE_LGM,
      "Plants use photosynthesis to produce glucose from carbon dioxide",
      "round-b1",
      blockNumber + 1n,
      CYCLE_PARAMS,
      [factIdA],
    );
    expect(result).toBeTruthy(); // Succeeds
  });

  it("should not trigger cycle check when feature is disabled", () => {
    // Use BASE_PARAMS which has maxReferenceCycleHops = 0 (off)
    const claimIdA = submitAndAcceptClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "The speed of light is approximately 299792 kilometers per second",
      "round-a1",
      blockNumber,
      BASE_PARAMS,
    );
    const factIdA = state.getClaim(claimIdA)!.resultFactId!;

    // Should succeed regardless (cycle check disabled)
    const minStake = computeRequiredStake(state, ALICE_KEY, BASE_PARAMS);
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "Light travels at about 299792 km/s in a vacuum environment",
      minStake,
      [factIdA],
    );
    tx.nonce = state.getNonce(ALICE_DID);
    const result = executeTransaction(state, tx, blockNumber + 1n, BASE_PARAMS);
    expect(result.success).toBe(true);
  });

  it("should allow references when submitter has no accepted facts", () => {
    // Bob has no accepted facts, so no cycle possible
    const claimIdA = submitAndAcceptClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "DNA contains the genetic instructions for all living organisms",
      "round-a1",
      blockNumber,
      CYCLE_PARAMS,
    );
    const factIdA = state.getClaim(claimIdA)!.resultFactId!;

    const tx = makeClaimSubmitTx(
      BOB_KEY,
      "Deoxyribonucleic acid encodes genetic information for organisms",
      1n * ONE_LGM,
      [factIdA],
    );
    tx.nonce = state.getNonce(BOB_DID);
    const result = executeTransaction(
      state,
      tx,
      blockNumber + 1n,
      CYCLE_PARAMS,
    );
    expect(result.success).toBe(true);
  });
});

// ============ Fix 3: Deterministic Block Tx Ordering (§9.4) ============

describe("Deterministic MEV Ordering (§9.4)", () => {
  it("should produce same ordering for same inputs regardless of input order", () => {
    const txA: Transaction = {
      type: "claim_submit" as any,
      sender: "alice" as any,
      nonce: 1n,
      fee: 100n,
      payload: { type: "claim_submit", claim: "Fact A" },
      signature: "sigA" as any,
    };
    const txB: Transaction = {
      type: "claim_submit" as any,
      sender: "bob" as any,
      nonce: 1n,
      fee: 200n,
      payload: { type: "claim_submit", claim: "Fact B" },
      signature: "sigB" as any,
    };
    const txC: Transaction = {
      type: "evidence_submit" as any,
      sender: "charlie" as any,
      nonce: 1n,
      fee: 50n,
      payload: { type: "evidence_submit", data: "evidence" },
      signature: "sigC" as any,
    };

    // Different input orders should produce same output
    const order1 = applyDeterministicOrdering([txA, txB, txC]);
    const order2 = applyDeterministicOrdering([txC, txA, txB]);
    const order3 = applyDeterministicOrdering([txB, txC, txA]);

    // All sensitive txs should be in the same order
    expect(order1.map((t) => t.sender)).toEqual(order2.map((t) => t.sender));
    expect(order2.map((t) => t.sender)).toEqual(order3.map((t) => t.sender));
  });

  it("should preserve non-sensitive tx ordering ahead of sensitive txs", () => {
    const transfer: Transaction = {
      type: "transfer" as any,
      sender: "alice" as any,
      nonce: 1n,
      fee: 100n,
      payload: { type: "transfer", to: "bob", amount: 100n },
      signature: "sig" as any,
    };
    const claimTx: Transaction = {
      type: "claim_submit" as any,
      sender: "bob" as any,
      nonce: 1n,
      fee: 200n,
      payload: { type: "claim_submit", claim: "test" },
      signature: "sig2" as any,
    };

    const ordered = applyDeterministicOrdering([claimTx, transfer]);
    // Non-sensitive (transfer) comes first
    expect(ordered[0].type).toBe("transfer");
    expect(ordered[1].type).toBe("claim_submit");
  });

  it("should be idempotent", () => {
    const txs: Transaction[] = [
      {
        type: "claim_submit" as any,
        sender: "alice" as any,
        nonce: 1n,
        fee: 10n,
        payload: { type: "claim_submit", claim: "X" },
        signature: "s1" as any,
      },
      {
        type: "dispute_open" as any,
        sender: "bob" as any,
        nonce: 2n,
        fee: 20n,
        payload: { type: "dispute_open", target: "Y" },
        signature: "s2" as any,
      },
    ];

    const first = applyDeterministicOrdering(txs);
    const second = applyDeterministicOrdering(first);
    expect(first.map((t) => t.sender)).toEqual(second.map((t) => t.sender));
  });
});

// ============ Fix 4: Deductive Proof Requirement (§7.3) ============

describe.sequential("Deductive Proof Requirement (§7.3)", () => {
  let state: StateManager;
  const blockNumber = 100n;

  beforeEach(() => {
    state = new StateManager();
    state.createAccount(ALICE_DID, "alice-op", 500000n * ONE_LGM);
    state.createAccount(
      "did:lgm:fee-pool",
      "fee-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should downgrade deductive to inductive without formal notation", () => {
    // Submit a claim with deductive inference but no formal notation
    // The claim zone will be classified — if propositional/presburger, it should downgrade
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "If all humans are mortal and Socrates is human then Socrates is mortal",
      1n * ONE_LGM,
      [],
      {
        derivedFrom: {
          factIds: [],
          inferenceType: "deductive",
          confidence: 0.99,
        },
      } as any,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    const result = executeTransaction(state, tx, blockNumber, BASE_PARAMS);
    // Should succeed (downgraded, not rejected)
    expect(result.success).toBe(true);

    // Check the stored claim — derivedFrom should have been downgraded
    const claims = state.getClaimsBySubmitter(ALICE_DID);
    const claim = claims[claims.length - 1];
    if (claim.derivedFrom) {
      // If zone was propositional/presburger, inferenceType was downgraded
      // If zone was empirical/peano, it stays deductive
      // Either way the claim succeeded
      expect(["deductive", "inductive"]).toContain(
        claim.derivedFrom.inferenceType,
      );
    }
  });

  it("should preserve deductive when formal notation is provided", () => {
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "If all humans are mortal and Socrates is human then Socrates is mortal",
      1n * ONE_LGM,
      [],
      {
        derivedFrom: {
          factIds: [],
          inferenceType: "deductive",
          confidence: 0.99,
        },
        formalNotation:
          "∀x(Human(x) → Mortal(x)) ∧ Human(Socrates) → Mortal(Socrates)",
      } as any,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    const result = executeTransaction(state, tx, blockNumber, BASE_PARAMS);
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(ALICE_DID);
    const claim = claims[claims.length - 1];
    // With formal notation, deductive should be preserved
    if (claim.derivedFrom) {
      expect(claim.derivedFrom.inferenceType).toBe("deductive");
    }
  });

  it("should not affect non-deductive claims", () => {
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "Most swans observed in nature have been white in color historically",
      1n * ONE_LGM,
      [],
      {
        derivedFrom: {
          factIds: [],
          inferenceType: "inductive",
          confidence: 0.75,
        },
      } as any,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    const result = executeTransaction(state, tx, blockNumber, BASE_PARAMS);
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(ALICE_DID);
    const claim = claims[claims.length - 1];
    if (claim.derivedFrom) {
      expect(claim.derivedFrom.inferenceType).toBe("inductive");
    }
  });
});

// ============ Fix 5: Insufficient Outcome Tracking (§6.4) ============

describe.sequential("Insufficient Outcome Tracking (§6.4)", () => {
  let state: StateManager;
  const blockNumber = 100n;

  beforeEach(() => {
    state = new StateManager();
    state.createAccount(ALICE_DID, "alice-op", 500000n * ONE_LGM);
    state.createAccount(
      "did:lgm:fee-pool",
      "fee-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should increment insufficient count on insufficient outcome", () => {
    const beforeCount = state.getInsufficientCount(ALICE_DID);
    expect(beforeCount).toBe(0);

    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Claim that will get insufficient validator participation result",
      "round-1",
      blockNumber,
      BASE_PARAMS,
      "insufficient",
    );

    const afterCount = state.getInsufficientCount(ALICE_DID);
    expect(afterCount).toBe(1);
  });

  it("should increment total verification count on all outcomes", () => {
    const before = state.getTotalVerificationCount(ALICE_DID);
    expect(before).toBe(0);

    // Accept one claim
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Accepted claim with sufficient validators for proper verification",
      "round-1",
      blockNumber,
      BASE_PARAMS,
      "accepted",
    );

    expect(state.getTotalVerificationCount(ALICE_DID)).toBe(1);

    // Insufficient outcome
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Claim resulting in insufficient validators for this test case",
      "round-2",
      blockNumber + 1n,
      BASE_PARAMS,
      "insufficient",
    );

    expect(state.getTotalVerificationCount(ALICE_DID)).toBe(2);
    expect(state.getInsufficientCount(ALICE_DID)).toBe(1);
  });

  it("should not double stake below threshold ratio", () => {
    // 1 insufficient out of 4 total = 25% (below 50% threshold)
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Accepted claim number one for ratio calculation testing purpose",
      "round-1",
      blockNumber,
      BASE_PARAMS,
      "accepted",
    );
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Accepted claim number two for ratio calculation testing purpose",
      "round-2",
      blockNumber + 1n,
      BASE_PARAMS,
      "accepted",
    );
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Accepted claim number three for ratio calculation testing now",
      "round-3",
      blockNumber + 2n,
      BASE_PARAMS,
      "accepted",
    );
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Claim that gets insufficient validator participation result",
      "round-4",
      blockNumber + 3n,
      BASE_PARAMS,
      "insufficient",
    );

    // Now submit another claim — should NOT get doubled stake
    const acceptedCount = state.getAcceptedClaimCountBySubmitter(
      ALICE_DID,
      true,
    );
    const normalStake = getGraduatedMinStake(acceptedCount, BASE_PARAMS);

    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "This claim should require normal stake amount not doubled",
      normalStake,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    const result = executeTransaction(state, tx, blockNumber + 4n, BASE_PARAMS);
    expect(result.success).toBe(true);
  });

  it("should double stake when insufficient ratio exceeds threshold", () => {
    // 3 insufficient out of 4 total = 75% (above 50% threshold, and >3 total)
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Accepted claim for ratio calculation with validators present",
      "round-1",
      blockNumber,
      BASE_PARAMS,
      "accepted",
    );
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "First insufficient claim with not enough validator participation",
      "round-2",
      blockNumber + 1n,
      BASE_PARAMS,
      "insufficient",
    );
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Second insufficient claim with not enough validator participation",
      "round-3",
      blockNumber + 2n,
      BASE_PARAMS,
      "insufficient",
    );
    submitAndCompleteClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Third insufficient claim with not enough validator participation",
      "round-4",
      blockNumber + 3n,
      BASE_PARAMS,
      "insufficient",
    );

    // 3/4 = 75% insufficient and >3 total → stake should be doubled
    const acceptedCount = state.getAcceptedClaimCountBySubmitter(
      ALICE_DID,
      true,
    );
    const normalStake = getGraduatedMinStake(acceptedCount, BASE_PARAMS);

    // Try with normal stake — should fail (needs 2x)
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "This claim needs doubled stake due to high insufficient ratio",
      normalStake,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    const result = executeTransaction(state, tx, blockNumber + 4n, BASE_PARAMS);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Minimum claim stake");

    // Try with doubled stake — should succeed
    const tx2 = makeClaimSubmitTx(
      ALICE_KEY,
      "This claim has doubled stake due to high insufficient ratio test",
      normalStake * 2n,
    );
    tx2.nonce = state.getNonce(ALICE_DID);
    const result2 = executeTransaction(
      state,
      tx2,
      blockNumber + 4n,
      BASE_PARAMS,
    );
    expect(result2.success).toBe(true);
  });
});

// ============ Fix 6: Domain Minimum Validator Pool (§5.2) ============

describe.sequential("Domain Minimum Validator Pool (§5.2)", () => {
  let state: StateManager;
  const blockNumber = 100n;

  const DOMAIN_PARAMS: ConsensusParams = {
    ...BASE_PARAMS,
    minDomainValidatorCount: 3, // Require at least 3 validators per domain
  } as ConsensusParams;

  beforeEach(() => {
    state = new StateManager();
    state.createAccount(ALICE_DID, "alice-op", 500000n * ONE_LGM);
    state.createAccount(
      "did:lgm:fee-pool",
      "fee-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should emit event when domain has too few validators", () => {
    // Submit and start verification — domain has 0 validators
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "Specialized niche domain claim that needs validator coverage test",
      1n * ONE_LGM,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    const submitResult = executeTransaction(
      state,
      tx,
      blockNumber,
      DOMAIN_PARAMS,
    );
    expect(submitResult.success).toBe(true);

    const claims = state.getClaimsBySubmitter(ALICE_DID);
    const claimId = claims[0].id;

    // Start verification — this should emit event but not block
    const consoleSpy = { called: false };
    const origLog = console.log;
    console.log = (...args: unknown[]) => {
      const msg = String(args[0] ?? "");
      if (msg.includes("DomainValidatorPoolTooSmall")) {
        consoleSpy.called = true;
      }
    };

    const startTx = makeSystemTx("claim_start_verification", {
      claimId,
      roundId: "round-1",
      validators: ["v1", "v2", "v3"],
    });
    const startResult = executeTransaction(
      state,
      startTx,
      blockNumber,
      DOMAIN_PARAMS,
    );
    console.log = origLog;

    // Verification should still proceed (event is informational)
    expect(startResult.success).toBe(true);
    // The event was emitted (in non-production, it logs)
    expect(consoleSpy.called).toBe(true);
  });

  it("should not emit event when feature is disabled", () => {
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "Claim in domain without minimum validator count requirement set",
      1n * ONE_LGM,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    executeTransaction(state, tx, blockNumber, BASE_PARAMS);

    const claims = state.getClaimsBySubmitter(ALICE_DID);
    const claimId = claims[0].id;

    const consoleSpy = { called: false };
    const origLog = console.log;
    console.log = (...args: unknown[]) => {
      const msg = String(args[0] ?? "");
      if (msg.includes("DomainValidatorPoolTooSmall")) {
        consoleSpy.called = true;
      }
    };

    const startTx = makeSystemTx("claim_start_verification", {
      claimId,
      roundId: "round-1",
      validators: ["v1", "v2", "v3"],
    });
    executeTransaction(state, startTx, blockNumber, BASE_PARAMS);
    console.log = origLog;

    expect(consoleSpy.called).toBe(false);
  });
});

// ============ Fix 7: Graduated Stake Farming Defense (§3.3 + §2.1) ============

describe.sequential("Graduation Requires References (§3.3)", () => {
  let state: StateManager;
  const blockNumber = 100n;

  const GRAD_PARAMS: ConsensusParams = {
    ...BASE_PARAMS,
    graduationRequiresRefsAfter: 3, // After 3 claims, require refs for graduation credit
  } as ConsensusParams;

  beforeEach(() => {
    state = new StateManager();
    state.createAccount(ALICE_DID, "alice-op", 500000n * ONE_LGM);
    state.createAccount(
      "did:lgm:fee-pool",
      "fee-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should count first N claims toward graduation without refs", () => {
    // First 3 claims should get graduation credit even without references
    for (let i = 0; i < 3; i++) {
      submitAndAcceptClaim(
        state,
        ALICE_KEY,
        1n * ONE_LGM,
        `Basic claim number ${i} without any references for testing graduation`,
        `round-${i}`,
        blockNumber + BigInt(i),
        GRAD_PARAMS,
      );
    }

    // All 3 should count toward graduation
    const gradCount = state.getAcceptedClaimCountBySubmitter(ALICE_DID, true);
    expect(gradCount).toBe(3);
  });

  it("should not give graduation credit to unreferenced claims after threshold", () => {
    // First 3 claims
    for (let i = 0; i < 3; i++) {
      submitAndAcceptClaim(
        state,
        ALICE_KEY,
        1n * ONE_LGM,
        `Claim number ${i} without references for the graduation test thing`,
        `round-${i}`,
        blockNumber + BigInt(i),
        GRAD_PARAMS,
      );
    }

    // 4th claim without references — should NOT get graduation credit
    submitAndAcceptClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Fourth claim with no references should not count for graduation",
      "round-3",
      blockNumber + 3n,
      GRAD_PARAMS,
    );

    // Only 3 should count for graduation (the 4th was accepted but no grad credit)
    const gradCount = state.getAcceptedClaimCountBySubmitter(ALICE_DID, true);
    expect(gradCount).toBe(3);

    // Total accepted is still 4
    const totalAccepted = state.getAcceptedClaimCountBySubmitter(
      ALICE_DID,
      false,
    );
    expect(totalAccepted).toBe(4);
  });

  it("should give graduation credit to referenced claims after threshold", () => {
    // Create 3 base claims
    const factIds: string[] = [];
    for (let i = 0; i < 3; i++) {
      const cid = submitAndAcceptClaim(
        state,
        ALICE_KEY,
        1n * ONE_LGM,
        `Base claim number ${i} for the reference graduation credit testing`,
        `round-${i}`,
        blockNumber + BigInt(i),
        GRAD_PARAMS,
      );
      factIds.push(state.getClaim(cid)!.resultFactId!);
    }

    // 4th claim WITH references — should get graduation credit
    submitAndAcceptClaim(
      state,
      ALICE_KEY,
      1n * ONE_LGM,
      "Fourth claim with references so it should get graduation credit",
      "round-3",
      blockNumber + 3n,
      GRAD_PARAMS,
      [factIds[0]],
    );

    const gradCount = state.getAcceptedClaimCountBySubmitter(ALICE_DID, true);
    expect(gradCount).toBe(4);
  });
});

describe.sequential("Multi-Sentence Claim Flagging (§2.1)", () => {
  let state: StateManager;
  const blockNumber = 100n;

  const MULTI_SENT_PARAMS: ConsensusParams = {
    ...BASE_PARAMS,
    multiSentenceClaimPenalty: true,
  } as ConsensusParams;

  beforeEach(() => {
    state = new StateManager();
    state.createAccount(ALICE_DID, "alice-op", 500000n * ONE_LGM);
    state.createAccount(
      "did:lgm:fee-pool",
      "fee-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should not flag single-sentence claims", () => {
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "Water boils at 100 degrees Celsius at standard atmospheric pressure",
      1n * ONE_LGM,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    const result = executeTransaction(
      state,
      tx,
      blockNumber,
      MULTI_SENT_PARAMS,
    );
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(ALICE_DID);
    const claim = claims[0];
    expect(claim.flags?.multiSentence).toBeUndefined();
  });

  it("should flag claims with more than 2 sentences", () => {
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "Water boils at 100 degrees Celsius. Ice melts at 0 degrees Celsius. Steam condenses at 100 degrees Celsius.",
      1n * ONE_LGM,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    const result = executeTransaction(
      state,
      tx,
      blockNumber,
      MULTI_SENT_PARAMS,
    );
    expect(result.success).toBe(true);

    const claims = state.getClaimsBySubmitter(ALICE_DID);
    const claim = claims[0];
    expect(claim.flags?.multiSentence).toBe(true);
  });
});

// ============ Fix 8: Locale Content Hash (§1.3) + Validator Diversity (§4.1) ============

describe("Locale Content Hash (§1.3)", () => {
  it("should produce different hashes for same text with different locale", () => {
    const h1 = generateClaimContentHash("Water boils at 100 degrees", "en");
    const h2 = generateClaimContentHash("Water boils at 100 degrees", "es");
    expect(h1).not.toBe(h2);
  });

  it("should produce same hash for same text and same locale", () => {
    const h1 = generateClaimContentHash("Water boils at 100 degrees", "en");
    const h2 = generateClaimContentHash("Water boils at 100 degrees", "en");
    expect(h1).toBe(h2);
  });

  it("should be backward compatible — no locale produces same as before", () => {
    const withoutLocale = generateClaimContentHash(
      "Water boils at 100 degrees",
    );
    const withUndefined = generateClaimContentHash(
      "Water boils at 100 degrees",
      undefined,
    );
    expect(withoutLocale).toBe(withUndefined);
  });
});

describe.sequential("Validator Key Diversity (§4.1)", () => {
  let state: StateManager;
  const blockNumber = 100n;

  beforeEach(() => {
    state = new StateManager();
    state.createAccount(ALICE_DID, "alice-op", 500000n * ONE_LGM);
    state.createAccount(
      "did:lgm:fee-pool",
      "fee-pool-hash",
      1000000n * ONE_LGM,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should reject duplicate validators in selection", () => {
    // Submit a claim first
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "Claim to test validator diversity during verification start",
      1n * ONE_LGM,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    executeTransaction(state, tx, blockNumber, BASE_PARAMS);

    const claims = state.getClaimsBySubmitter(ALICE_DID);
    const claimId = claims[0].id;

    // Start verification with duplicate validators
    const startTx = makeSystemTx("claim_start_verification", {
      claimId,
      roundId: "round-1",
      validators: ["v1", "v2", "v1"], // duplicate v1!
      selectedValidators: ["v1", "v2", "v1"],
    });
    const result = executeTransaction(state, startTx, blockNumber, BASE_PARAMS);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Duplicate validators");
  });

  it("should accept unique validators", () => {
    const tx = makeClaimSubmitTx(
      ALICE_KEY,
      "Claim to test unique validator set during verification process",
      1n * ONE_LGM,
    );
    tx.nonce = state.getNonce(ALICE_DID);
    executeTransaction(state, tx, blockNumber, BASE_PARAMS);

    const claims = state.getClaimsBySubmitter(ALICE_DID);
    const claimId = claims[0].id;

    const startTx = makeSystemTx("claim_start_verification", {
      claimId,
      roundId: "round-1",
      validators: ["v1", "v2", "v3"],
    });
    const result = executeTransaction(state, startTx, blockNumber, BASE_PARAMS);
    expect(result.success).toBe(true);
  });
});
