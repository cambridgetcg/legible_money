/**
 * State Transitions Unit Tests
 *
 * Tests deterministic state transition logic for transaction types.
 * Focuses on core transactions: transfer, stake, system transactions.
 *
 * PP HIERARCHY:
 *   TRUTH: Verify balance changes are mathematically correct
 *   UNDERSTANDING: Test all transition paths and edge cases
 *   BEAUTY: Clear test structure matching transaction types
 *   JUSTICE: Tests in proper unit test location
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  executeBlock,
  applyVerificationResult,
  FEE_POOL_DID,
  REWARD_POOL_DID,
  PROTOCOL_TREASURY_DID,
  type TransitionResult,
} from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
  VerificationResult,
} from "../../../consensus/pot/types.js";
import { generateKeypair } from "../../../core/crypto/keys.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";

// ============ Test Helpers ============

function createTestState(): StateManager {
  return new StateManager();
}

function createTestParams(): ConsensusParams {
  return {
    targetBlockTimeMs: 2521,
    minValidatorStake: 1000n * 10n ** 18n,
    minClaimStake: 10n * 10n ** 18n,
    validatorQuorum: 0.66,
    verificationRoundBlocks: 20,
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 6,
    epochBlocks: 100,
    maxValidators: 100,
    slashFractionDowntime: 0.01,
    slashFractionDoubleSign: 0.05,
    unbondingPeriod: 1000n,
  };
}

function createTransferTx(
  sender: string,
  to: string,
  amount: bigint,
  fee: bigint = 1000n,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "transfer",
    sender,
    payload: { to, amount },
    fee,
    nonce,
    timestamp: Date.now(),
    signature: "test-signature",
  };
}

function createStakeTx(
  sender: string,
  amount: bigint,
  fee: bigint = 1000n,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "stake",
    sender,
    payload: { amount },
    fee,
    nonce,
    timestamp: Date.now(),
    signature: "test-signature",
  };
}

function createUnstakeTx(
  sender: string,
  amount: bigint,
  fee: bigint = 1000n,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "unstake",
    sender,
    payload: { amount },
    fee,
    nonce,
    timestamp: Date.now(),
    signature: "test-signature",
  };
}

// ============ Transfer Transaction Tests ============

describe.sequential("Transfer Transactions", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let senderKey: { publicKey: string; privateKey: string };
  let recipientKey: { publicKey: string; privateKey: string };
  let senderDid: string;
  let recipientDid: string;

  beforeEach(() => {
    state = createTestState();
    params = createTestParams();
    senderKey = generateKeypair();
    recipientKey = generateKeypair();
    senderDid = publicKeyToDid(senderKey.publicKey);
    recipientDid = publicKeyToDid(recipientKey.publicKey);

    // Setup accounts with initial balance
    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      10000n * 10n ** 18n,
    );
    state.createAccount(
      recipientDid,
      sha512_256Hex(recipientKey.publicKey),
      0n,
    );

    // Setup fee pool and reward pool
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * 10n ** 18n,
    );
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);

    // Setup research treasury for fee routing
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should execute valid transfer", () => {
    const tx = createTransferTx(
      senderKey.publicKey,
      recipientDid,
      100n * 10n ** 18n,
      1000n,
    );

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(21000n);
  });

  it("should deduct correct amount from sender", () => {
    const amount = 100n * 10n ** 18n;
    const fee = 1000n;
    const initialBalance = state.getBalance(senderDid);

    const tx = createTransferTx(senderKey.publicKey, recipientDid, amount, fee);
    executeTransaction(state, tx, 1n, params);

    const finalBalance = state.getBalance(senderDid);
    // Sender loses: amount + fee
    expect(initialBalance - finalBalance).toBe(amount + fee);
  });

  it("should credit correct amount to recipient", () => {
    const amount = 100n * 10n ** 18n;
    const initialBalance = state.getBalance(recipientDid);

    const tx = createTransferTx(senderKey.publicKey, recipientDid, amount);
    executeTransaction(state, tx, 1n, params);

    const finalBalance = state.getBalance(recipientDid);
    expect(finalBalance - initialBalance).toBe(amount);
  });

  it("should reject transfer with insufficient balance", () => {
    const amount = 20000n * 10n ** 18n; // More than sender has
    const tx = createTransferTx(senderKey.publicKey, recipientDid, amount);

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("Insufficient");
  });

  it("should reject transfer with zero amount", () => {
    const tx = createTransferTx(senderKey.publicKey, recipientDid, 0n);

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid transfer amount");
  });

  it("should reject transfer with negative amount", () => {
    const tx = createTransferTx(senderKey.publicKey, recipientDid, -100n);

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(false);
  });

  it("should reject transfer with invalid nonce", () => {
    const tx = createTransferTx(
      senderKey.publicKey,
      recipientDid,
      100n * 10n ** 18n,
      1000n,
      5n,
    );

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("nonce");
  });

  it("should increment nonce after successful transfer", () => {
    const nonceBefore = state.getNonce(senderDid);
    const tx = createTransferTx(
      senderKey.publicKey,
      recipientDid,
      100n * 10n ** 18n,
      1000n,
      nonceBefore,
    );

    executeTransaction(state, tx, 1n, params);

    const nonceAfter = state.getNonce(senderDid);
    // Nonce increments once per transaction (fee routing uses deductBalance/creditBalance)
    expect(nonceAfter).toBe(nonceBefore + 1n);
  });

  it("should reject transfer from non-existent account", () => {
    const unknownKey = generateKeypair();
    const tx = createTransferTx(
      unknownKey.publicKey,
      recipientDid,
      100n * 10n ** 18n,
    );

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("does not exist");
  });
});

// ============ Stake Transaction Tests ============

describe.sequential("Stake Transactions", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let validatorKey: { publicKey: string; privateKey: string };
  let validatorDid: string;

  beforeEach(() => {
    state = createTestState();
    params = createTestParams();
    validatorKey = generateKeypair();
    validatorDid = publicKeyToDid(validatorKey.publicKey);

    // Setup validator account with balance
    state.createAccount(
      validatorDid,
      sha512_256Hex(validatorKey.publicKey),
      50000n * 10n ** 18n,
    );

    // Setup protocol accounts
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * 10n ** 18n,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should execute valid stake transaction", () => {
    const stakeAmount = params.minValidatorStake;
    const tx = createStakeTx(validatorKey.publicKey, stakeAmount);

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(50000n);
  });

  it("should move balance from available to staked", () => {
    const stakeAmount = params.minValidatorStake;
    const initialBalance = state.getBalance(validatorDid);

    const tx = createStakeTx(validatorKey.publicKey, stakeAmount);
    executeTransaction(state, tx, 1n, params);

    const account = state.getAccount(validatorDid);
    expect(account?.stakedBalance).toBe(stakeAmount);
    expect(state.getBalance(validatorDid)).toBeLessThan(initialBalance);
  });

  it("should reject stake below minimum", () => {
    const stakeAmount = params.minValidatorStake - 1n;
    const tx = createStakeTx(validatorKey.publicKey, stakeAmount);

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("Minimum stake");
  });

  it("should reject stake with insufficient balance", () => {
    const stakeAmount = 100000n * 10n ** 18n; // More than balance
    const tx = createStakeTx(validatorKey.publicKey, stakeAmount);

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(false);
  });

  it("should register as validator on first stake", () => {
    const stakeAmount = params.minValidatorStake;
    const tx = createStakeTx(validatorKey.publicKey, stakeAmount);

    executeTransaction(state, tx, 1n, params);

    expect(state.isActiveValidator(validatorKey.publicKey)).toBe(true);
  });
});

// ============ Unstake Transaction Tests ============

describe.sequential("Unstake Transactions", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let validatorKey: { publicKey: string; privateKey: string };
  let validatorDid: string;

  beforeEach(() => {
    state = createTestState();
    params = createTestParams();
    validatorKey = generateKeypair();
    validatorDid = publicKeyToDid(validatorKey.publicKey);

    // Setup validator with stake
    state.createAccount(
      validatorDid,
      sha512_256Hex(validatorKey.publicKey),
      50000n * 10n ** 18n,
    );
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);

    // First stake
    const stakeTx = createStakeTx(validatorKey.publicKey, 10000n * 10n ** 18n);
    executeTransaction(state, stakeTx, 1n, params);
  });

  it("should execute valid unstake transaction", () => {
    const unstakeAmount = 5000n * 10n ** 18n;
    const currentNonce = state.getNonce(validatorDid);
    const tx = createUnstakeTx(
      validatorKey.publicKey,
      unstakeAmount,
      1000n,
      currentNonce,
    );

    const result = executeTransaction(state, tx, 2n, params);

    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(50000n);
  });

  it("should reduce staked balance", () => {
    const unstakeAmount = 5000n * 10n ** 18n;
    const stakedBefore = state.getAccount(validatorDid)?.stakedBalance ?? 0n;
    const currentNonce = state.getNonce(validatorDid);

    const tx = createUnstakeTx(
      validatorKey.publicKey,
      unstakeAmount,
      1000n,
      currentNonce,
    );
    executeTransaction(state, tx, 2n, params);

    const stakedAfter = state.getAccount(validatorDid)?.stakedBalance ?? 0n;
    expect(stakedBefore - stakedAfter).toBe(unstakeAmount);
  });

  it("should reject unstake more than staked", () => {
    const unstakeAmount = 50000n * 10n ** 18n; // More than staked
    const currentNonce = state.getNonce(validatorDid);
    const tx = createUnstakeTx(
      validatorKey.publicKey,
      unstakeAmount,
      1000n,
      currentNonce,
    );

    const result = executeTransaction(state, tx, 2n, params);

    expect(result.success).toBe(false);
  });
});

// ============ System Transaction Tests ============

describe.sequential("System Transactions", () => {
  let state: StateManager;
  let params: ConsensusParams;

  beforeEach(() => {
    state = createTestState();
    params = createTestParams();

    // Setup protocol accounts
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * 10n ** 18n,
    );
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  describe.sequential("Add Fact", () => {
    it("should add fact to knowledge layer", () => {
      const factId = sha512_256Hex("test-fact-1");

      const tx: Transaction = {
        type: "add_fact",
        sender: "system",
        payload: {
          type: "add_fact",
          factId,
          confidence: 0.85,
          verifierCount: 5,
          verifiedAtBlock: 100,
          claim: "Test claim",
          domain: "science",
        },
        fee: 0n,
        nonce: 0n,
        timestamp: Date.now(),
        signature: "system",
      };

      const result = executeTransaction(state, tx, 100n, params);

      expect(result.success).toBe(true);
      expect(result.gasUsed).toBe(0n); // System tx
    });

    it("should not charge fee for system transaction", () => {
      const factId = sha512_256Hex("test-fact-2");

      const tx: Transaction = {
        type: "add_fact",
        sender: "system",
        payload: {
          type: "add_fact",
          factId,
          confidence: 0.9,
          verifierCount: 7,
          verifiedAtBlock: 100,
        },
        fee: 0n,
        nonce: 0n,
        timestamp: Date.now(),
        signature: "system",
      };

      const result = executeTransaction(state, tx, 100n, params);

      expect(result.success).toBe(true);
      expect(result.gasUsed).toBe(0n);
    });
  });

  describe.sequential("Verification Reward", () => {
    let validatorKey: { publicKey: string; privateKey: string };
    let validatorDid: string;

    beforeEach(() => {
      validatorKey = generateKeypair();
      validatorDid = publicKeyToDid(validatorKey.publicKey);
      state.createAccount(
        validatorDid,
        sha512_256Hex(validatorKey.publicKey),
        0n,
      );
    });

    it("should reward validator for verification", () => {
      const rewardAmount = 100n * 10n ** 18n;

      const tx: Transaction = {
        type: "verification_reward",
        sender: "system",
        payload: {
          type: "verification_reward",
          recipient: validatorDid,
          amount: rewardAmount,
          roundId: "round-1",
        },
        fee: 0n,
        nonce: 0n,
        timestamp: Date.now(),
        signature: "system",
      };

      const balanceBefore = state.getBalance(validatorDid);
      executeTransaction(state, tx, 100n, params);
      const balanceAfter = state.getBalance(validatorDid);

      expect(balanceAfter).toBeGreaterThan(balanceBefore);
    });
  });

  describe.sequential("Slash Validator", () => {
    let validatorKey: { publicKey: string; privateKey: string };
    let validatorDid: string;

    beforeEach(() => {
      validatorKey = generateKeypair();
      validatorDid = publicKeyToDid(validatorKey.publicKey);

      // Setup validator with stake
      state.createAccount(
        validatorDid,
        sha512_256Hex(validatorKey.publicKey),
        50000n * 10n ** 18n,
      );
      // Note: FEE_POOL_DID and research-treasury need to exist for fee routing
      // Only create if not already created by parent beforeEach
      if (!state.accountExists(FEE_POOL_DID)) {
        state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
      }
      // research-treasury already created by parent beforeEach

      // Stake to become validator
      const stakeTx = createStakeTx(
        validatorKey.publicKey,
        10000n * 10n ** 18n,
      );
      executeTransaction(state, stakeTx, 1n, params);
    });

    it("should slash validator stake", () => {
      const slashAmount = 1000n * 10n ** 18n;

      const tx: Transaction = {
        type: "slash_validator",
        sender: "system",
        payload: {
          type: "slash_validator",
          validator: validatorKey.publicKey,
          amount: slashAmount,
          reason: "Double signing",
        },
        fee: 0n,
        nonce: 0n,
        timestamp: Date.now(),
        signature: "system",
      };

      const stakeBefore =
        state.getValidator(validatorKey.publicKey)?.stake ?? 0n;
      executeTransaction(state, tx, 100n, params);
      const stakeAfter =
        state.getValidator(validatorKey.publicKey)?.stake ?? 0n;

      expect(stakeBefore - stakeAfter).toBe(slashAmount);
    });

    it("should not charge fee for slash transaction", () => {
      const tx: Transaction = {
        type: "slash_validator",
        sender: "system",
        payload: {
          type: "slash_validator",
          validator: validatorKey.publicKey,
          amount: 500n * 10n ** 18n,
          reason: "Missed verification",
        },
        fee: 0n,
        nonce: 0n,
        timestamp: Date.now(),
        signature: "system",
      };

      const result = executeTransaction(state, tx, 100n, params);

      expect(result.success).toBe(true);
      expect(result.gasUsed).toBe(0n);
    });
  });
});

// ============ Transaction Validation Tests ============

describe.sequential("Transaction Validation", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let senderKey: { publicKey: string; privateKey: string };
  let senderDid: string;

  beforeEach(() => {
    state = createTestState();
    params = createTestParams();
    senderKey = generateKeypair();
    senderDid = publicKeyToDid(senderKey.publicKey);

    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      10000n * 10n ** 18n,
    );
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should reject transaction with insufficient fee", () => {
    const tx = createTransferTx(
      senderKey.publicKey,
      "did:lgm:recipient",
      9999n * 10n ** 18n, // Almost all balance
      2n * 10n ** 18n, // Fee that would exceed remaining
    );

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(false);
  });

  it("should reject unknown transaction type", () => {
    const tx: Transaction = {
      type: "unknown_type" as any,
      sender: senderKey.publicKey,
      payload: {},
      fee: 1000n,
      nonce: 0n,
      timestamp: Date.now(),
      signature: "test",
    };

    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("Unknown transaction type");
  });
});

// ============ Block Execution Tests ============

describe.sequential("Block Execution", () => {
  let state: StateManager;
  let params: ConsensusParams;

  beforeEach(() => {
    state = createTestState();
    params = createTestParams();

    // Setup accounts
    const senderKey = generateKeypair();
    const senderDid = publicKeyToDid(senderKey.publicKey);

    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      100000n * 10n ** 18n,
    );
    state.createAccount("did:lgm:recipient-1", "recipient-1-hash", 0n);
    state.createAccount("did:lgm:recipient-2", "recipient-2-hash", 0n);
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-pool-hash",
      1000000n * 10n ** 18n,
    );
    state.createAccount("did:lgm:research", "research-hash", 0n);
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
  });

  it("should execute multiple transactions in block", () => {
    const senderKey = generateKeypair();
    const senderDid = publicKeyToDid(senderKey.publicKey);
    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      100000n * 10n ** 18n,
    );

    const transactions: Transaction[] = [
      createTransferTx(
        senderKey.publicKey,
        "did:lgm:recipient-1",
        100n * 10n ** 18n,
        1000n,
        0n,
      ),
      createTransferTx(
        senderKey.publicKey,
        "did:lgm:recipient-2",
        200n * 10n ** 18n,
        1000n,
        1n,
      ),
    ];

    const result = executeBlock(state, transactions, [], 1n, params);

    expect(result.successCount).toBe(2);
    expect(result.failureCount).toBe(0);
    expect(result.totalGasUsed).toBe(42000n); // 2 * 21000
  });

  it("should continue after failed transaction", () => {
    const senderKey = generateKeypair();
    const senderDid = publicKeyToDid(senderKey.publicKey);
    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      100000n * 10n ** 18n,
    );

    // tx1: nonce 0, succeeds, nonce becomes 1
    // tx2: nonce 1, fails (invalid amount 0n), nonce stays 1 (rollback)
    // tx3: nonce 1, succeeds
    const transactions: Transaction[] = [
      createTransferTx(
        senderKey.publicKey,
        "did:lgm:recipient-1",
        100n * 10n ** 18n,
        1000n,
        0n,
      ),
      createTransferTx(
        senderKey.publicKey,
        "did:lgm:recipient-2",
        0n,
        1000n,
        1n,
      ), // Invalid amount
      createTransferTx(
        senderKey.publicKey,
        "did:lgm:recipient-1",
        50n * 10n ** 18n,
        1000n,
        1n,
      ),
    ];

    const result = executeBlock(state, transactions, [], 1n, params);

    expect(result.successCount).toBe(2);
    expect(result.failureCount).toBe(1);
  });

  it("should return new state root", () => {
    const senderKey = generateKeypair();
    const senderDid = publicKeyToDid(senderKey.publicKey);
    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      100000n * 10n ** 18n,
    );

    const transactions: Transaction[] = [
      createTransferTx(
        senderKey.publicKey,
        "did:lgm:recipient-1",
        100n * 10n ** 18n,
        1000n,
        0n,
      ),
    ];

    const result = executeBlock(state, transactions, [], 1n, params);

    expect(result.stateRoot).toBeTruthy();
    expect(result.stateRoot.length).toBe(64); // 32 bytes hex (SHA-256)
  });
});

// ============ Fee Routing Tests ============

describe.sequential("Fee Routing", () => {
  let state: StateManager;
  let params: ConsensusParams;
  let senderKey: { publicKey: string; privateKey: string };
  let senderDid: string;

  beforeEach(() => {
    state = createTestState();
    params = createTestParams();
    senderKey = generateKeypair();
    senderDid = publicKeyToDid(senderKey.publicKey);

    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      10000n * 10n ** 18n,
    );
    state.createAccount("did:lgm:recipient", "recipient-hash", 0n);
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should route fee to fee pool and 5-way treasury split", () => {
    const fee = 1000000n; // 1M units

    const tx = createTransferTx(
      senderKey.publicKey,
      "did:lgm:recipient",
      100n * 10n ** 18n,
      fee,
    );
    executeTransaction(state, tx, 1n, params);

    const feePoolBalance = state.getBalance(FEE_POOL_DID);
    // Research gets 77% of the 22% evidence tax
    const researchBalance = state.getBalance("did:lgm:research");

    // 78% to fee pool
    expect(feePoolBalance).toBeGreaterThan(0n);
    // Research gets 77% of 22% = ~16.94%
    expect(researchBalance).toBeGreaterThan(0n);

    // Sum all 5 treasury funds + fee pool = total fee
    const foundationBalance = state.getBalance("did:lgm:treasury-foundation");
    const communityBalance = state.getBalance("did:lgm:treasury-community");
    const developersBalance = state.getBalance("did:lgm:treasury-developers");
    const reserveBalance = state.getBalance("did:lgm:treasury-reserve");
    const totalDistributed =
      feePoolBalance +
      researchBalance +
      foundationBalance +
      communityBalance +
      developersBalance +
      reserveBalance;
    expect(totalDistributed).toBe(fee);
  });
});

// ============ Edge Cases ============

describe.sequential("Transition Edge Cases", () => {
  let state: StateManager;
  let params: ConsensusParams;

  beforeEach(() => {
    state = createTestState();
    params = createTestParams();
    state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
    state.createAccount("did:lgm:research", "research-hash", 0n);
  });

  it("should handle transfer to self", () => {
    const senderKey = generateKeypair();
    const senderDid = publicKeyToDid(senderKey.publicKey);
    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      10000n * 10n ** 18n,
    );

    const tx = createTransferTx(
      senderKey.publicKey,
      senderDid,
      100n * 10n ** 18n,
    );
    const result = executeTransaction(state, tx, 1n, params);

    // Self-transfer should succeed (useful for nonce management)
    expect(result.success).toBe(true);
  });

  it("should handle very small amounts", () => {
    const senderKey = generateKeypair();
    const senderDid = publicKeyToDid(senderKey.publicKey);
    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      10000n * 10n ** 18n,
    );
    state.createAccount("did:lgm:recipient", "recipient-hash", 0n);

    const tx = createTransferTx(
      senderKey.publicKey,
      "did:lgm:recipient",
      1n,
      100n,
    );
    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(true);
    expect(state.getBalance("did:lgm:recipient")).toBe(1n);
  });

  it("should handle very large amounts", () => {
    const senderKey = generateKeypair();
    const senderDid = publicKeyToDid(senderKey.publicKey);
    const largeAmount = 10n ** 30n;
    state.createAccount(
      senderDid,
      sha512_256Hex(senderKey.publicKey),
      largeAmount,
    );
    state.createAccount("did:lgm:recipient", "recipient-hash", 0n);

    const tx = createTransferTx(
      senderKey.publicKey,
      "did:lgm:recipient",
      largeAmount - 10000n,
      1000n,
    );
    const result = executeTransaction(state, tx, 1n, params);

    expect(result.success).toBe(true);
    expect(state.getBalance("did:lgm:recipient")).toBe(largeAmount - 10000n);
  });
});
