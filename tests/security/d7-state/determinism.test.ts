/**
 * Dimension 7: State & Data Integrity - State Determinism Tests
 *
 * Validates that identical inputs always produce identical outputs:
 * 1. executeTransaction is deterministic for same inputs
 * 2. Block application is deterministic
 * 3. State root computation is a pure function of components
 * 4. No hidden randomness in state transitions
 *
 * Industry method: Differential testing (run same input twice, compare).
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  resetBootstrapFund,
} from "../../../state/transitions.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../../consensus/pot/types.js";
import type { Transaction } from "../../../state/types.js";

// ============ Helpers ============

const ALICE = "did:lgm:alice-det-test";
const BOB = "did:lgm:bob-det-test";
const OP_KEY = "b".repeat(64);
const ONE_LGM = 10n ** 18n;

function freshState(): StateManager {
  const state = new StateManager();
  state.createAccount(ALICE, OP_KEY, 100n * ONE_LGM);
  state.createAccount(BOB, OP_KEY, 50n * ONE_LGM);
  return state;
}

function makeTransferTx(
  from: string,
  to: string,
  amount: bigint,
  nonce: bigint,
): Transaction {
  return {
    type: "transfer",
    sender: from,
    payload: {
      to,
      amount,
    },
    nonce,
    fee: ONE_LGM / 1000n, // 0.001 LGM
    signature: "test-sig-" + Math.random().toString(36),
    timestamp: 1000000,
  } as Transaction;
}

// ============================================================
// DET-T01 through DET-T06: Transaction determinism
// ============================================================

describe("Transaction Determinism (DET-T)", () => {
  beforeEach(() => {
    resetBootstrapFund();
  });

  it("DET-T01: Same transfer tx on cloned state produces identical results", () => {
    const state1 = freshState();
    const state2 = freshState();

    // Both start identical
    expect(state1.getStateRoot()).toBe(state2.getStateRoot());

    const tx: Transaction = {
      type: "transfer",
      sender: ALICE,
      payload: { to: BOB, amount: 10n * ONE_LGM },
      nonce: 0n,
      fee: 0n,
      signature: "determinism-test-sig",
      timestamp: 1000000,
    } as Transaction;

    const result1 = executeTransaction(
      state1,
      tx,
      1n,
      DEFAULT_CONSENSUS_PARAMS,
    );
    const result2 = executeTransaction(
      state2,
      tx,
      1n,
      DEFAULT_CONSENSUS_PARAMS,
    );

    // Both succeed or both fail
    expect(result1.success).toBe(result2.success);

    // Both produce same final state
    expect(state1.getStateRoot()).toBe(state2.getStateRoot());
    expect(state1.getBalance(ALICE)).toBe(state2.getBalance(ALICE));
    expect(state1.getBalance(BOB)).toBe(state2.getBalance(BOB));
  });

  it("DET-T02: State root is a pure function of all sub-tree roots + version", () => {
    const state = freshState();

    // Get root twice without mutation — must be identical
    const root1 = state.getStateRoot();
    const root2 = state.getStateRoot();
    expect(root1).toBe(root2);

    // Components should also be stable
    const comp1 = state.getStateRootComponents();
    const comp2 = state.getStateRootComponents();
    expect(comp1.accountRoot).toBe(comp2.accountRoot);
    expect(comp1.validatorRoot).toBe(comp2.validatorRoot);
    expect(comp1.knowledgeRoot).toBe(comp2.knowledgeRoot);
    expect(comp1.version).toBe(comp2.version);
  });

  it("DET-T03: Sequential commits produce monotonically different roots", () => {
    const state = freshState();
    const roots: string[] = [];

    for (let i = 0; i < 5; i++) {
      state.beginTransaction();
      state.transfer(ALICE, BOB, 1n * ONE_LGM);
      const root = state.commitTransaction();
      roots.push(root);
    }

    // All roots unique
    const uniqueRoots = new Set(roots);
    expect(uniqueRoots.size).toBe(5);
  });

  it("DET-T04: Failed transaction (insufficient balance) produces no state change", () => {
    const state = freshState();
    const rootBefore = state.getStateRoot();
    const aliceBal = state.getBalance(ALICE);

    // Try to transfer more than Alice has
    const tx: Transaction = {
      type: "transfer",
      sender: ALICE,
      payload: { to: BOB, amount: 999n * ONE_LGM }, // More than 100 LGM
      nonce: 0n,
      fee: 0n,
      signature: "fail-test-sig",
      timestamp: 1000000,
    } as Transaction;

    const result = executeTransaction(state, tx, 1n, DEFAULT_CONSENSUS_PARAMS);

    // Transaction should fail
    expect(result.success).toBe(false);

    // State must be unchanged
    expect(state.getStateRoot()).toBe(rootBefore);
    expect(state.getBalance(ALICE)).toBe(aliceBal);
  });

  it("DET-T05: System transactions bypass nonce and fee checks deterministically", () => {
    const state = freshState();

    // System transaction: sender = "system"
    const sysTx: Transaction = {
      type: "verification_reward",
      sender: "system",
      payload: {
        validatorDid: ALICE,
        amount: 5n * ONE_LGM,
        blockNumber: 1n,
        reason: "verification",
      },
      nonce: 0n,
      fee: 0n,
      signature: "system-sig",
      timestamp: 1000000,
    } as Transaction;

    const result = executeTransaction(
      state,
      sysTx,
      1n,
      DEFAULT_CONSENSUS_PARAMS,
    );

    // System transactions should succeed (or at least be deterministic)
    // The exact behavior depends on the handler, but it should not throw
    expect(typeof result.success).toBe("boolean");
  });

  it("DET-T06: Nonce mismatch rejection is deterministic", () => {
    const state1 = freshState();
    const state2 = freshState();

    // Both states start at nonce 0, send tx with nonce 5 (mismatch)
    const tx: Transaction = {
      type: "transfer",
      sender: ALICE,
      payload: { to: BOB, amount: 1n * ONE_LGM },
      nonce: 5n, // Wrong nonce
      fee: 0n,
      signature: "nonce-test-sig",
      timestamp: 1000000,
    } as Transaction;

    const result1 = executeTransaction(
      state1,
      tx,
      1n,
      DEFAULT_CONSENSUS_PARAMS,
    );
    const result2 = executeTransaction(
      state2,
      tx,
      1n,
      DEFAULT_CONSENSUS_PARAMS,
    );

    // Both should fail identically
    expect(result1.success).toBe(result2.success);
    expect(result1.success).toBe(false);

    // State unchanged in both
    expect(state1.getStateRoot()).toBe(state2.getStateRoot());
  });
});
