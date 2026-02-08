/**
 * Anti-Sybil Defense Tests
 *
 * Tests the 4-layer bootstrap fund protection:
 *   Layer 1: Bootstrap vesting (linear unlock over vestingBlocks)
 *   Layer 2: Per-block claim cap (max N claims per block)
 *   Layer 3: Transfer cooldown for new accounts
 *   Layer 4: Minimum tx fee to trigger bootstrap
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  resetBootstrapFund,
  resetBlockBootstrapCounter,
  setBootstrapFund,
  BOOTSTRAP_FUND_DID,
} from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import {
  BootstrapFund,
  BOOTSTRAP_AMOUNT,
} from "../../../core/genesis/bootstrap_fund.js";

// ============ Test Helpers ============

function createAntiSybilParams(
  overrides: Partial<ConsensusParams> = {},
): ConsensusParams {
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
    // Anti-sybil defaults
    bootstrapVestingBlocks: 34272n,
    maxBootstrapClaimsPerBlock: 10,
    newAccountTransferCooldown: 4800n,
    bootstrapMinTxFee: 1000n,
    ...overrides,
  };
}

function makeTransferTx(
  senderDid: string,
  to: string,
  amount: bigint,
  fee: bigint = 1000n,
  nonce: bigint = 0n,
): Transaction {
  return {
    type: "transfer",
    sender: senderDid,
    senderDid,
    payload: { to, amount },
    fee,
    nonce,
    timestamp: Date.now(),
    signature: `sig-${senderDid}-${nonce}`,
  } as Transaction;
}

/** Access the private AccountStateTree for test setup */
function accountTree(state: StateManager) {
  return (state as any)
    .accounts as import("../../../state/accounts.js").AccountStateTree;
}

function setupBootstrapFundBalance(
  state: StateManager,
  amount: bigint = 10n ** 24n,
): void {
  if (!state.accountExists(BOOTSTRAP_FUND_DID)) {
    accountTree(state).createAccount(BOOTSTRAP_FUND_DID, "", amount);
  } else {
    accountTree(state).creditBalance(BOOTSTRAP_FUND_DID, amount);
  }
}

function createFundedAccount(
  state: StateManager,
  did: string,
  balance: bigint,
  createdAtBlock: bigint = 0n,
): void {
  accountTree(state).createAccount(did, "", balance, createdAtBlock);
}

// ============ Tests ============

describe("Anti-Sybil Defenses", () => {
  let state: StateManager;
  let params: ConsensusParams;

  beforeEach(() => {
    state = new StateManager();
    params = createAntiSybilParams();
    resetBootstrapFund();
    resetBlockBootstrapCounter();
    setupBootstrapFundBalance(state);
  });

  // -------- Layer 1: Bootstrap Vesting --------

  describe("Layer 1: Bootstrap Vesting", () => {
    it("should credit bootstrap as vested, not freely transferable", () => {
      const did = "did:lgm:new-agent-001";
      const recipient = "did:lgm:recipient-001";
      createFundedAccount(state, recipient, 0n);
      createFundedAccount(state, did, 5000n, 0n);

      // Disable cooldown so the transfer itself can succeed
      const localParams = createAntiSybilParams({
        newAccountTransferCooldown: 0n,
      });

      resetBlockBootstrapCounter();
      const tx = makeTransferTx(did, recipient, 1n, 1000n, 0n);
      const result = executeTransaction(state, tx, 0n, localParams);
      expect(result.success).toBe(true);

      // Verify bootstrap was credited as vested
      const account = state.getAccount(did);
      expect(account).toBeDefined();
      expect(account!.bootstrapVested).toBe(BOOTSTRAP_AMOUNT);
      expect(account!.bootstrapVestedAtBlock).toBe(0n);
    });

    it("should lock full bootstrap at block 0 (just claimed)", () => {
      const did = "did:lgm:vesting-lock-test";
      const vestingBlocks = params.bootstrapVestingBlocks!;
      const regularBalance = 10000n;

      // Create account with regular balance, then credit bootstrap on top
      createFundedAccount(state, did, regularBalance, 0n);
      state.creditBootstrapVested(did, BOOTSTRAP_AMOUNT, 0n);
      // Total balance is now regularBalance + BOOTSTRAP_AMOUNT

      // At block 0, entire bootstrap is locked
      const locked = state.getLockedBootstrap(did, 0n, vestingBlocks);
      expect(locked).toBe(BOOTSTRAP_AMOUNT);

      const transferable = state.getTransferableBalance(did, 0n, vestingBlocks);
      // Only the non-bootstrap balance is transferable
      expect(transferable).toBe(regularBalance);
    });

    it("should unlock linearly over vesting period", () => {
      const did = "did:lgm:vesting-linear-test";
      const vestingBlocks = 34272n;

      // Create with 0 balance, then credit bootstrap
      createFundedAccount(state, did, 0n, 0n);
      state.creditBootstrapVested(did, BOOTSTRAP_AMOUNT, 0n);

      // At halfway point
      const halfBlock = vestingBlocks / 2n;
      const lockedAtHalf = state.getLockedBootstrap(
        did,
        halfBlock,
        vestingBlocks,
      );
      const expectedUnlocked = (BOOTSTRAP_AMOUNT * halfBlock) / vestingBlocks;
      const expectedLocked = BOOTSTRAP_AMOUNT - expectedUnlocked;
      expect(lockedAtHalf).toBe(expectedLocked);

      // At quarter
      const quarterBlock = vestingBlocks / 4n;
      const lockedAtQuarter = state.getLockedBootstrap(
        did,
        quarterBlock,
        vestingBlocks,
      );
      const expectedUnlockedQ =
        (BOOTSTRAP_AMOUNT * quarterBlock) / vestingBlocks;
      expect(lockedAtQuarter).toBe(BOOTSTRAP_AMOUNT - expectedUnlockedQ);

      // Locked at quarter > locked at half
      expect(lockedAtQuarter).toBeGreaterThan(lockedAtHalf);
    });

    it("should fully unlock after vesting period", () => {
      const did = "did:lgm:vesting-full-test";
      const vestingBlocks = 34272n;

      // Create with 0 balance, then credit bootstrap
      createFundedAccount(state, did, 0n, 0n);
      state.creditBootstrapVested(did, BOOTSTRAP_AMOUNT, 0n);
      // Total balance = BOOTSTRAP_AMOUNT

      // After vesting completes
      const locked = state.getLockedBootstrap(
        did,
        vestingBlocks,
        vestingBlocks,
      );
      expect(locked).toBe(0n);

      // Way past vesting
      const lockedLater = state.getLockedBootstrap(
        did,
        vestingBlocks * 2n,
        vestingBlocks,
      );
      expect(lockedLater).toBe(0n);

      // Full balance transferable (balance = BOOTSTRAP_AMOUNT, locked = 0)
      const transferable = state.getTransferableBalance(
        did,
        vestingBlocks,
        vestingBlocks,
      );
      expect(transferable).toBe(BOOTSTRAP_AMOUNT);
    });

    it("should block transfer of locked bootstrap funds", () => {
      const sender = "did:lgm:vesting-sender";
      const recipient = "did:lgm:vesting-recipient";
      const vestedAtBlock = 10n;

      // Create sender with enough regular balance for the tx, plus bootstrap vested
      createFundedAccount(state, sender, 100000n, 0n);
      state.creditBootstrapVested(sender, BOOTSTRAP_AMOUNT, vestedAtBlock);
      createFundedAccount(state, recipient, 0n);

      // Set nonce > 0 to skip bootstrap trigger
      state.incrementNonce(sender);

      // Try to transfer the full bootstrap amount at block 11 (almost nothing vested)
      // Disable cooldown so we only test vesting
      const paramsNoCooldown = createAntiSybilParams({
        newAccountTransferCooldown: 0n,
      });
      const tx = makeTransferTx(
        sender,
        recipient,
        BOOTSTRAP_AMOUNT + 50000n,
        1000n,
        1n,
      );
      const result = executeTransaction(
        state,
        tx,
        vestedAtBlock + 1n,
        paramsNoCooldown,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("bootstrap vesting");
    });
  });

  // -------- Layer 2: Per-Block Bootstrap Claim Cap --------

  describe("Layer 2: Per-Block Claim Cap", () => {
    it("should allow claims up to the cap", () => {
      const maxPerBlock = 3;
      const localParams = createAntiSybilParams({
        maxBootstrapClaimsPerBlock: maxPerBlock,
        bootstrapMinTxFee: 0n, // Disable fee gate for this test
        bootstrapVestingBlocks: 0n, // Disable vesting for clarity
        newAccountTransferCooldown: 0n,
      });

      resetBlockBootstrapCounter();

      let claimCount = 0;
      for (let i = 0; i < maxPerBlock; i++) {
        const did = `did:lgm:cap-test-${i}`;
        createFundedAccount(state, did, 100000n, 0n);
        const recipient = `did:lgm:cap-recipient-${i}`;
        createFundedAccount(state, recipient, 0n);

        const tx = makeTransferTx(did, recipient, 1n, 0n, 0n);
        const result = executeTransaction(state, tx, 1n, localParams);
        expect(result.success).toBe(true);

        // Check if bootstrap was credited
        const account = state.getAccount(did);
        if (account && account.balance > 100000n - 1n) {
          claimCount++;
        }
      }

      expect(claimCount).toBe(maxPerBlock);
    });

    it("should deny bootstrap claims beyond the cap (tx still succeeds)", () => {
      const maxPerBlock = 2;
      const localParams = createAntiSybilParams({
        maxBootstrapClaimsPerBlock: maxPerBlock,
        bootstrapMinTxFee: 0n,
        bootstrapVestingBlocks: 0n,
        newAccountTransferCooldown: 0n,
      });

      resetBlockBootstrapCounter();

      // Process maxPerBlock claims
      for (let i = 0; i < maxPerBlock; i++) {
        const did = `did:lgm:cap-fill-${i}`;
        createFundedAccount(state, did, 100000n, 0n);
        const recipient = `did:lgm:cap-fill-recipient-${i}`;
        createFundedAccount(state, recipient, 0n);

        const tx = makeTransferTx(did, recipient, 1n, 0n, 0n);
        executeTransaction(state, tx, 1n, localParams);
      }

      // Next claim should NOT get bootstrap, but tx should still succeed
      const overCapDid = "did:lgm:over-cap";
      createFundedAccount(state, overCapDid, 100000n, 0n);
      const overCapRecipient = "did:lgm:over-cap-recipient";
      createFundedAccount(state, overCapRecipient, 0n);

      const balanceBefore = state.getBalance(overCapDid);
      const tx = makeTransferTx(overCapDid, overCapRecipient, 1n, 0n, 0n);
      const result = executeTransaction(state, tx, 1n, localParams);

      expect(result.success).toBe(true);
      // Bootstrap should NOT have been credited (balance shouldn't include bootstrap amount)
      const balanceAfter = state.getBalance(overCapDid);
      // Balance should be original - transfer amount (no bootstrap added)
      expect(balanceAfter).toBe(balanceBefore - 1n);
    });

    it("should reset counter between blocks", () => {
      const maxPerBlock = 1;
      const localParams = createAntiSybilParams({
        maxBootstrapClaimsPerBlock: maxPerBlock,
        bootstrapMinTxFee: 0n,
        bootstrapVestingBlocks: 0n,
        newAccountTransferCooldown: 0n,
      });

      // First block - use up the cap
      resetBlockBootstrapCounter();
      const did1 = "did:lgm:reset-test-1";
      createFundedAccount(state, did1, 100000n, 0n);
      const r1 = "did:lgm:reset-r-1";
      createFundedAccount(state, r1, 0n);
      executeTransaction(
        state,
        makeTransferTx(did1, r1, 1n, 0n, 0n),
        1n,
        localParams,
      );

      // Second claim in same block should NOT get bootstrap
      const did2 = "did:lgm:reset-test-2";
      createFundedAccount(state, did2, 100000n, 0n);
      const r2 = "did:lgm:reset-r-2";
      createFundedAccount(state, r2, 0n);
      const balBefore = state.getBalance(did2);
      executeTransaction(
        state,
        makeTransferTx(did2, r2, 1n, 0n, 0n),
        1n,
        localParams,
      );
      const balAfterSameBlock = state.getBalance(did2);

      // Reset for "new block"
      resetBlockBootstrapCounter();

      // Third claim in new block SHOULD get bootstrap
      const did3 = "did:lgm:reset-test-3";
      createFundedAccount(state, did3, 100000n, 0n);
      const r3 = "did:lgm:reset-r-3";
      createFundedAccount(state, r3, 0n);
      const balBefore3 = state.getBalance(did3);
      executeTransaction(
        state,
        makeTransferTx(did3, r3, 1n, 0n, 0n),
        2n,
        localParams,
      );
      const balAfterNewBlock = state.getBalance(did3);

      // did2 should not have gotten bootstrap (same block, over cap)
      expect(balAfterSameBlock).toBe(balBefore - 1n);
      // did3 should have gotten bootstrap (new block, cap reset)
      expect(balAfterNewBlock).toBeGreaterThan(balBefore3 - 1n);
    });
  });

  // -------- Layer 3: Transfer Cooldown --------

  describe("Layer 3: Transfer Cooldown", () => {
    it("should block outbound transfers from new accounts", () => {
      const sender = "did:lgm:cooldown-sender";
      const recipient = "did:lgm:cooldown-recipient";
      const createdAtBlock = 100n;
      const currentBlock = 200n; // Only 100 blocks old, cooldown is 4800

      createFundedAccount(state, sender, 1000000n, createdAtBlock);
      createFundedAccount(state, recipient, 0n);

      // Skip bootstrap trigger
      state.incrementNonce(sender);

      const tx = makeTransferTx(sender, recipient, 1000n, 1000n, 1n);
      const localParams = createAntiSybilParams({ bootstrapVestingBlocks: 0n });
      const result = executeTransaction(state, tx, currentBlock, localParams);

      expect(result.success).toBe(false);
      expect(result.error).toContain("Account too new");
    });

    it("should allow transfers after cooldown period", () => {
      const sender = "did:lgm:cooldown-pass-sender";
      const recipient = "did:lgm:cooldown-pass-recipient";
      const createdAtBlock = 100n;
      const cooldown = 4800n;
      const currentBlock = createdAtBlock + cooldown + 1n; // Past cooldown

      createFundedAccount(state, sender, 1000000n, createdAtBlock);
      createFundedAccount(state, recipient, 0n);

      // Skip bootstrap trigger
      state.incrementNonce(sender);

      const tx = makeTransferTx(sender, recipient, 1000n, 1000n, 1n);
      const localParams = createAntiSybilParams({ bootstrapVestingBlocks: 0n });
      const result = executeTransaction(state, tx, currentBlock, localParams);

      expect(result.success).toBe(true);
    });

    it("should allow transfers at exactly the cooldown boundary", () => {
      const sender = "did:lgm:cooldown-exact";
      const recipient = "did:lgm:cooldown-exact-r";
      const createdAtBlock = 0n;
      const cooldown = 4800n;
      const currentBlock = cooldown; // Exactly at boundary

      createFundedAccount(state, sender, 1000000n, createdAtBlock);
      createFundedAccount(state, recipient, 0n);
      state.incrementNonce(sender);

      const tx = makeTransferTx(sender, recipient, 1000n, 1000n, 1n);
      const localParams = createAntiSybilParams({ bootstrapVestingBlocks: 0n });
      const result = executeTransaction(state, tx, currentBlock, localParams);

      expect(result.success).toBe(true);
    });

    it("should not apply cooldown when disabled (cooldown = 0)", () => {
      const sender = "did:lgm:no-cooldown";
      const recipient = "did:lgm:no-cooldown-r";

      createFundedAccount(state, sender, 1000000n, 100n);
      createFundedAccount(state, recipient, 0n);
      state.incrementNonce(sender);

      const tx = makeTransferTx(sender, recipient, 1000n, 1000n, 1n);
      const localParams = createAntiSybilParams({
        newAccountTransferCooldown: 0n,
        bootstrapVestingBlocks: 0n,
      });
      const result = executeTransaction(state, tx, 101n, localParams);

      expect(result.success).toBe(true);
    });
  });

  // -------- Layer 4: Minimum Fee Gate --------

  describe("Layer 4: Minimum Fee Gate", () => {
    it("should not trigger bootstrap when fee is below minimum", () => {
      const did = "did:lgm:fee-gate-low";
      const recipient = "did:lgm:fee-gate-low-r";
      const minFee = 1000n;

      createFundedAccount(state, did, 100000n, 0n);
      createFundedAccount(state, recipient, 0n);

      const localParams = createAntiSybilParams({
        bootstrapMinTxFee: minFee,
        bootstrapVestingBlocks: 0n,
        newAccountTransferCooldown: 0n,
      });

      resetBlockBootstrapCounter();

      // Send with fee below minimum
      const tx = makeTransferTx(did, recipient, 1n, minFee - 1n, 0n);
      const balBefore = state.getBalance(did);
      executeTransaction(state, tx, 1n, localParams);
      const balAfter = state.getBalance(did);

      // Balance should have decreased by transfer + fee, but NO bootstrap added
      // If bootstrap was NOT credited, balance = balBefore - amount - fee
      // Note: fee deduction happens in the handler
      const account = state.getAccount(did);
      expect(account!.bootstrapVested).toBeUndefined();
    });

    it("should trigger bootstrap when fee meets minimum", () => {
      const did = "did:lgm:fee-gate-ok";
      const recipient = "did:lgm:fee-gate-ok-r";
      const minFee = 1000n;

      createFundedAccount(state, did, 100000n, 0n);
      createFundedAccount(state, recipient, 0n);

      const localParams = createAntiSybilParams({
        bootstrapMinTxFee: minFee,
        newAccountTransferCooldown: 0n,
      });

      resetBlockBootstrapCounter();

      const tx = makeTransferTx(did, recipient, 1n, minFee, 0n);
      executeTransaction(state, tx, 1n, localParams);

      // Bootstrap should have been credited as vested
      const account = state.getAccount(did);
      expect(account!.bootstrapVested).toBe(BOOTSTRAP_AMOUNT);
    });

    it("should allow zero-fee bootstrap when min fee is 0", () => {
      const did = "did:lgm:fee-gate-zero";
      const recipient = "did:lgm:fee-gate-zero-r";

      createFundedAccount(state, did, 100000n, 0n);
      createFundedAccount(state, recipient, 0n);

      const localParams = createAntiSybilParams({
        bootstrapMinTxFee: 0n,
        newAccountTransferCooldown: 0n,
      });

      resetBlockBootstrapCounter();

      const tx = makeTransferTx(did, recipient, 1n, 0n, 0n);
      executeTransaction(state, tx, 1n, localParams);

      const account = state.getAccount(did);
      expect(account!.bootstrapVested).toBe(BOOTSTRAP_AMOUNT);
    });
  });

  // -------- StateManager Vesting Helpers --------

  describe("StateManager Vesting Helpers", () => {
    it("should return 0 locked for accounts without vesting", () => {
      const did = "did:lgm:no-vesting";
      createFundedAccount(state, did, 1000000n, 0n);

      const locked = state.getLockedBootstrap(did, 100n, 34272n);
      expect(locked).toBe(0n);
    });

    it("should return 0 locked for non-existent accounts", () => {
      const locked = state.getLockedBootstrap(
        "did:lgm:nonexistent",
        100n,
        34272n,
      );
      expect(locked).toBe(0n);
    });

    it("should credit bootstrap vested correctly", () => {
      const did = "did:lgm:credit-test";
      createFundedAccount(state, did, 0n, 0n);

      state.creditBootstrapVested(did, BOOTSTRAP_AMOUNT, 50n);

      const account = state.getAccount(did);
      expect(account!.balance).toBe(BOOTSTRAP_AMOUNT);
      expect(account!.bootstrapVested).toBe(BOOTSTRAP_AMOUNT);
      expect(account!.bootstrapVestedAtBlock).toBe(50n);
    });

    it("should calculate transferable balance correctly", () => {
      const did = "did:lgm:transferable-test";
      const vestingBlocks = 1000n;
      const regularBalance = 5000n;

      createFundedAccount(state, did, regularBalance, 0n);
      state.creditBootstrapVested(did, BOOTSTRAP_AMOUNT, 0n);

      // At block 0: only regular balance is transferable
      const t0 = state.getTransferableBalance(did, 0n, vestingBlocks);
      expect(t0).toBe(regularBalance);

      // At half vesting: regular + half bootstrap
      const tHalf = state.getTransferableBalance(did, 500n, vestingBlocks);
      const expectedUnlocked = (BOOTSTRAP_AMOUNT * 500n) / 1000n;
      expect(tHalf).toBe(regularBalance + expectedUnlocked);

      // After full vesting: everything
      const tFull = state.getTransferableBalance(did, 1000n, vestingBlocks);
      expect(tFull).toBe(regularBalance + BOOTSTRAP_AMOUNT);
    });
  });

  // -------- Combined Layers --------

  describe("Combined Defense Layers", () => {
    it("should apply all layers simultaneously", () => {
      const attacker = "did:lgm:sybil-attacker";
      const consolidator = "did:lgm:consolidator";

      // Created at block 100. Cooldown=4800, so must wait until block 4900+
      createFundedAccount(state, attacker, 100000n, 100n);
      createFundedAccount(state, consolidator, 0n);

      resetBlockBootstrapCounter();

      // First tx with sufficient fee triggers vested bootstrap
      // Disable cooldown for the first tx (it's a transfer, will hit cooldown)
      // Actually cooldown only affects outbound transfers — but this IS a transfer.
      // The account is brand new (age=0 at block 100). Let's disable cooldown for the initial claim tx.
      const claimParams = createAntiSybilParams({
        newAccountTransferCooldown: 0n,
      });
      const claimTx = makeTransferTx(attacker, consolidator, 1n, 1000n, 0n);
      const result1 = executeTransaction(state, claimTx, 100n, claimParams);
      expect(result1.success).toBe(true);

      // Bootstrap credited as vested
      const account = state.getAccount(attacker);
      expect(account!.bootstrapVested).toBe(BOOTSTRAP_AMOUNT);

      // Now try to transfer the bootstrap out immediately (block 101) WITH cooldown re-enabled
      const nonce = state.getNonce(attacker);
      const transferTx = makeTransferTx(
        attacker,
        consolidator,
        BOOTSTRAP_AMOUNT,
        1000n,
        nonce,
      );
      const result2 = executeTransaction(state, transferTx, 101n, params);
      expect(result2.success).toBe(false);
      // Should fail on cooldown (checked first)
      expect(result2.error).toContain("Account too new");
    });

    it("should allow legitimate agent flow after waiting", () => {
      const agent = "did:lgm:legitimate-agent";
      const peer = "did:lgm:peer";
      const createdAtBlock = 0n;

      createFundedAccount(state, agent, 100000n, createdAtBlock);
      createFundedAccount(state, peer, 0n);

      resetBlockBootstrapCounter();

      // First tx at block 0: triggers bootstrap (disable cooldown for claim tx)
      const claimParams = createAntiSybilParams({
        newAccountTransferCooldown: 0n,
      });
      const tx1 = makeTransferTx(agent, peer, 1n, 1000n, 0n);
      executeTransaction(state, tx1, 0n, claimParams);

      const account = state.getAccount(agent);
      expect(account!.bootstrapVested).toBe(BOOTSTRAP_AMOUNT);

      // After cooldown AND full vesting: should work
      const safeBlock =
        params.newAccountTransferCooldown! +
        params.bootstrapVestingBlocks! +
        1n;
      const nonce = state.getNonce(agent);
      const tx2 = makeTransferTx(
        agent,
        peer,
        BOOTSTRAP_AMOUNT / 2n,
        1000n,
        nonce,
      );
      const result = executeTransaction(state, tx2, safeBlock, params);

      expect(result.success).toBe(true);
    });

    it("sybil farming scenario: attacker farms N DIDs", () => {
      const maxPerBlock = 3;
      const localParams = createAntiSybilParams({
        maxBootstrapClaimsPerBlock: maxPerBlock,
        bootstrapMinTxFee: 500n,
        bootstrapVestingBlocks: 100n,
        newAccountTransferCooldown: 50n,
      });
      // Also need a version with no cooldown for initial claim txs
      const claimParams = createAntiSybilParams({
        maxBootstrapClaimsPerBlock: maxPerBlock,
        bootstrapMinTxFee: 500n,
        bootstrapVestingBlocks: 100n,
        newAccountTransferCooldown: 0n, // Disable cooldown for initial claims
      });

      const consolidator = "did:lgm:farm-consolidator";
      createFundedAccount(state, consolidator, 0n, 0n);

      resetBlockBootstrapCounter();

      // Attacker farms 5 DIDs in one block (disable cooldown so claim txs succeed)
      let claimsGranted = 0;
      for (let i = 0; i < 5; i++) {
        const did = `did:lgm:farm-${i}`;
        createFundedAccount(state, did, 10000n, 10n);

        const tx = makeTransferTx(did, consolidator, 1n, 500n, 0n);
        const result = executeTransaction(state, tx, 10n, claimParams);
        expect(result.success).toBe(true);

        const acct = state.getAccount(did);
        if (acct?.bootstrapVested === BOOTSTRAP_AMOUNT) {
          claimsGranted++;
        }
      }

      // Only maxPerBlock should have received bootstrap
      expect(claimsGranted).toBe(maxPerBlock);

      // None can transfer out yet (cooldown re-enabled)
      for (let i = 0; i < maxPerBlock; i++) {
        const did = `did:lgm:farm-${i}`;
        const nonce = state.getNonce(did);
        const tx = makeTransferTx(
          did,
          consolidator,
          BOOTSTRAP_AMOUNT,
          500n,
          nonce,
        );
        // Block 20, account created at 10, age=10 < 50 cooldown
        const result = executeTransaction(state, tx, 20n, localParams);
        expect(result.success).toBe(false);
      }

      // After cooldown but before vesting completes, transfer is limited
      for (let i = 0; i < maxPerBlock; i++) {
        const did = `did:lgm:farm-${i}`;
        const nonce = state.getNonce(did);
        // Block 80: age=70 > cooldown(50), vesting started at block 10, elapsed=70 < 100
        const tx = makeTransferTx(
          did,
          consolidator,
          BOOTSTRAP_AMOUNT,
          500n,
          nonce,
        );
        const result = executeTransaction(state, tx, 80n, localParams);
        expect(result.success).toBe(false);
        expect(result.error).toContain("bootstrap vesting");
      }
    });
  });

  // -------- Edge Cases --------

  describe("Edge Cases", () => {
    it("should not apply bootstrap to system transactions", () => {
      const systemDid = "system";
      const recipient = "did:lgm:system-recipient";
      createFundedAccount(state, recipient, 100000n);

      const tx: Transaction = {
        type: "add_fact",
        sender: systemDid,
        payload: {},
        fee: 0n,
        nonce: 0n,
        timestamp: Date.now(),
        signature: "system-sig",
      } as Transaction;

      resetBlockBootstrapCounter();
      // System tx should bypass bootstrap entirely
      // (it won't try to access bootstrap fund for "system" sender)
      const fundBalBefore = state.getBalance(BOOTSTRAP_FUND_DID);
      executeTransaction(state, tx, 1n, params);
      const fundBalAfter = state.getBalance(BOOTSTRAP_FUND_DID);

      expect(fundBalAfter).toBe(fundBalBefore);
    });

    it("should not trigger bootstrap on second transaction (nonce > 0)", () => {
      const did = "did:lgm:second-tx";
      const recipient = "did:lgm:second-tx-r";

      createFundedAccount(state, did, 1000000n, 0n);
      createFundedAccount(state, recipient, 0n);

      // Increment nonce to simulate having already done first tx
      state.incrementNonce(did);

      resetBlockBootstrapCounter();

      const tx = makeTransferTx(did, recipient, 1000n, 1000n, 1n);
      const localParams = createAntiSybilParams({
        newAccountTransferCooldown: 0n,
        bootstrapVestingBlocks: 0n,
      });
      const fundBalBefore = state.getBalance(BOOTSTRAP_FUND_DID);
      executeTransaction(state, tx, 100n, localParams);
      const fundBalAfter = state.getBalance(BOOTSTRAP_FUND_DID);

      // No bootstrap should have been deducted from fund
      expect(fundBalAfter).toBe(fundBalBefore);
    });

    it("should handle bootstrap fund with insufficient balance gracefully", () => {
      const did = "did:lgm:empty-fund";
      const recipient = "did:lgm:empty-fund-r";

      // Drain the bootstrap fund
      const fundBal = state.getBalance(BOOTSTRAP_FUND_DID);
      if (fundBal > 0n) {
        state.deductBalance(BOOTSTRAP_FUND_DID, fundBal);
      }

      createFundedAccount(state, did, 100000n, 0n);
      createFundedAccount(state, recipient, 0n);

      resetBlockBootstrapCounter();

      const tx = makeTransferTx(did, recipient, 1n, 1000n, 0n);
      const localParams = createAntiSybilParams({
        newAccountTransferCooldown: 0n,
        bootstrapVestingBlocks: 0n,
      });
      const result = executeTransaction(state, tx, 1n, localParams);

      // Transaction should succeed, but no bootstrap credited
      expect(result.success).toBe(true);
      const account = state.getAccount(did);
      expect(account!.bootstrapVested).toBeUndefined();
    });
  });
});
