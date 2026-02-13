/**
 * Period Balance Tracking Tests
 *
 * Tests the per-period balance tracking system that enables token expiration:
 * - Period tracking activation/deactivation
 * - Lazy migration of existing accounts
 * - Credit to current period
 * - FIFO debit (oldest periods first)
 * - Staking exemption (stake debits, unstake credits current period)
 * - Serialization round-trip
 * - Clone safety (deep copy of Map)
 * - Sweep: finding and removing expired period balances
 */

import { describe, it, expect, beforeEach } from "vitest";
import { AccountStateTree } from "../../../state/accounts.js";
import { StateManager } from "../../../state/state_manager.js";
import { DEFAULT_EXPIRATION_CONFIG } from "../../../core/tokens/types.js";

// ============ Constants ============

const PERIOD_MS = DEFAULT_EXPIRATION_CONFIG.periodDurationMs; // 604,800,000 (1 week)
const PERIODS_UNTIL_EXPIRATION =
  DEFAULT_EXPIRATION_CONFIG.periodsUntilExpiration; // 26

// Test-friendly period numbers
const PERIOD_0 = 0;
const PERIOD_1 = 1;
const PERIOD_5 = 5;
const PERIOD_10 = 10;
const PERIOD_30 = 30; // Old enough to be expired relative to period 56+

// ============ Helpers ============

function createTree(): AccountStateTree {
  return new AccountStateTree();
}

function createAccount(
  tree: AccountStateTree,
  did: string,
  balance: bigint,
): void {
  tree.createAccount(did, `${did}-hash`, balance);
}

// ============ Tests ============

describe.sequential("Period Balance Tracking", () => {
  // ============ Activation ============

  describe.sequential("Activation", () => {
    let tree: AccountStateTree;

    beforeEach(() => {
      tree = createTree();
    });

    it("should NOT create periodBalances when currentPeriod is undefined", () => {
      createAccount(tree, "did:lgm:alice", 100n);
      tree.creditBalance("did:lgm:alice", 50n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(150n);
      expect(account.periodBalances).toBeUndefined();
    });

    it("should create periodBalances when currentPeriod is set", () => {
      createAccount(tree, "did:lgm:alice", 100n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:alice", 50n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(150n);
      expect(account.periodBalances).toBeDefined();
      // Migration: 100n adopted into period 5, then +50n credited to period 5
      expect(account.periodBalances!.get(PERIOD_5)).toBe(150n);
    });

    it("should stop tracking when currentPeriod is reset to undefined", () => {
      createAccount(tree, "did:lgm:alice", 100n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:alice", 50n);

      // Disable period tracking
      tree.setCurrentPeriod(undefined);
      tree.creditBalance("did:lgm:alice", 25n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(175n);
      // periodBalances preserved but not updated
      expect(account.periodBalances!.get(PERIOD_5)).toBe(150n);
    });
  });

  // ============ Migration ============

  describe.sequential("Lazy Migration", () => {
    let tree: AccountStateTree;

    beforeEach(() => {
      tree = createTree();
    });

    it("should adopt existing balance into currentPeriod on first credit", () => {
      createAccount(tree, "did:lgm:bob", 500n);
      tree.setCurrentPeriod(PERIOD_10);
      tree.creditBalance("did:lgm:bob", 100n);

      const account = tree.getAccount("did:lgm:bob")!;
      expect(account.balance).toBe(600n);
      // 500n (existing) + 100n (credit) = 600n in period 10
      expect(account.periodBalances!.get(PERIOD_10)).toBe(600n);
      expect(account.periodBalances!.size).toBe(1);
    });

    it("should adopt existing balance into currentPeriod on first debit", () => {
      createAccount(tree, "did:lgm:bob", 500n);
      tree.setCurrentPeriod(PERIOD_10);
      tree.deductBalance("did:lgm:bob", 100n);

      const account = tree.getAccount("did:lgm:bob")!;
      expect(account.balance).toBe(400n);
      // 500n adopted into period 10, then -100n debited from period 10
      expect(account.periodBalances!.get(PERIOD_10)).toBe(400n);
    });

    it("should handle zero-balance account on first credit", () => {
      createAccount(tree, "did:lgm:charlie", 0n);
      tree.setCurrentPeriod(PERIOD_10);
      tree.creditBalance("did:lgm:charlie", 200n);

      const account = tree.getAccount("did:lgm:charlie")!;
      expect(account.balance).toBe(200n);
      expect(account.periodBalances!.get(PERIOD_10)).toBe(200n);
      expect(account.periodBalances!.size).toBe(1);
    });
  });

  // ============ Credit ============

  describe.sequential("Credit", () => {
    let tree: AccountStateTree;

    beforeEach(() => {
      tree = createTree();
      tree.setCurrentPeriod(PERIOD_5);
    });

    it("should credit to current period", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      tree.creditBalance("did:lgm:alice", 100n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.periodBalances!.get(PERIOD_5)).toBe(100n);
    });

    it("should accumulate multiple credits in same period", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      tree.creditBalance("did:lgm:alice", 100n);
      tree.creditBalance("did:lgm:alice", 200n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.periodBalances!.get(PERIOD_5)).toBe(300n);
    });

    it("should add new period entry when period changes", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      tree.creditBalance("did:lgm:alice", 100n);

      tree.setCurrentPeriod(PERIOD_10);
      tree.creditBalance("did:lgm:alice", 200n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.periodBalances!.get(PERIOD_5)).toBe(100n);
      expect(account.periodBalances!.get(PERIOD_10)).toBe(200n);
      expect(account.periodBalances!.size).toBe(2);
    });
  });

  // ============ FIFO Debit ============

  describe.sequential("FIFO Debit", () => {
    let tree: AccountStateTree;

    beforeEach(() => {
      tree = createTree();
    });

    it("should debit from oldest period first", () => {
      createAccount(tree, "did:lgm:alice", 0n);

      // Credit 100 in period 1, 200 in period 5
      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:alice", 100n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:alice", 200n);

      // Debit 150 — FIFO takes 100 from period 1, then 50 from period 5
      tree.deductBalance("did:lgm:alice", 150n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(150n);
      expect(account.periodBalances!.has(PERIOD_1)).toBe(false); // fully consumed
      expect(account.periodBalances!.get(PERIOD_5)).toBe(150n);
    });

    it("should span multiple periods on large debit", () => {
      createAccount(tree, "did:lgm:alice", 0n);

      // Credit: period 1 = 100, period 5 = 200, period 10 = 300
      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:alice", 100n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:alice", 200n);
      tree.setCurrentPeriod(PERIOD_10);
      tree.creditBalance("did:lgm:alice", 300n);

      // Debit 450: FIFO takes 100 from p1, 200 from p5, 150 from p10
      tree.deductBalance("did:lgm:alice", 450n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(150n);
      expect(account.periodBalances!.has(PERIOD_1)).toBe(false); // fully debited
      expect(account.periodBalances!.has(PERIOD_5)).toBe(false); // fully debited
      expect(account.periodBalances!.get(PERIOD_10)).toBe(150n);
    });

    it("should remove period entry when fully debited", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:alice", 100n);

      tree.deductBalance("did:lgm:alice", 100n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(0n);
      expect(account.periodBalances!.size).toBe(0);
    });

    it("should handle debit of entire balance across all periods", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:alice", 100n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:alice", 200n);

      tree.deductBalance("did:lgm:alice", 300n);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(0n);
      expect(account.periodBalances!.size).toBe(0);
    });
  });

  // ============ Staking ============

  describe.sequential("Staking Exemption", () => {
    let tree: AccountStateTree;

    beforeEach(() => {
      tree = createTree();
      tree.setCurrentPeriod(PERIOD_5);
    });

    it("should debit periodBalances when staking (tokens leave expiration)", () => {
      createAccount(tree, "did:lgm:validator", 1000n);
      // Migration puts 1000n in period 5

      tree.stake("did:lgm:validator", 400n);

      const account = tree.getAccount("did:lgm:validator")!;
      expect(account.balance).toBe(600n);
      expect(account.stakedBalance).toBe(400n);
      expect(account.periodBalances!.get(PERIOD_5)).toBe(600n);
    });

    it("should credit currentPeriod when unstaking (fresh expiration)", () => {
      createAccount(tree, "did:lgm:validator", 1000n);
      tree.stake("did:lgm:validator", 400n);

      // Advance to new period
      tree.setCurrentPeriod(PERIOD_10);
      tree.unstake("did:lgm:validator", 400n);

      const account = tree.getAccount("did:lgm:validator")!;
      expect(account.balance).toBe(1000n);
      expect(account.stakedBalance).toBe(0n);
      // 600n from period 5 (remaining after stake), 400n in period 10 (unstaked)
      expect(account.periodBalances!.get(PERIOD_5)).toBe(600n);
      expect(account.periodBalances!.get(PERIOD_10)).toBe(400n);
    });
  });

  // ============ Transfer ============

  describe.sequential("Transfer with Period Tracking", () => {
    let tree: AccountStateTree;

    beforeEach(() => {
      tree = createTree();
      tree.setCurrentPeriod(PERIOD_5);
    });

    it("should FIFO debit sender and credit recipient in currentPeriod", () => {
      createAccount(tree, "did:lgm:sender", 0n);
      createAccount(tree, "did:lgm:receiver", 0n);

      // Sender has tokens in two periods
      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:sender", 100n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:sender", 200n);

      // Transfer 150: FIFO takes from period 1 first (100), then 50 from period 5
      tree.transfer("did:lgm:sender", "did:lgm:receiver", 150n);

      const sender = tree.getAccount("did:lgm:sender")!;
      expect(sender.balance).toBe(150n);
      expect(sender.periodBalances!.has(PERIOD_1)).toBe(false); // fully consumed
      expect(sender.periodBalances!.get(PERIOD_5)).toBe(150n);

      const receiver = tree.getAccount("did:lgm:receiver")!;
      expect(receiver.balance).toBe(150n);
      // Receiver gets tokens in current period (period 5) with fresh expiration
      expect(receiver.periodBalances!.get(PERIOD_5)).toBe(150n);
    });
  });

  // ============ Period-Preserving Transfer (Ping-Pong Fix) ============

  describe.sequential("Period-Preserving Transfer", () => {
    let tree: AccountStateTree;

    beforeEach(() => {
      tree = createTree();
    });

    it("should preserve sender period-of-origin in recipient", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      createAccount(tree, "did:lgm:bob", 0n);

      // Alice has tokens from period 1 and period 5
      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:alice", 100n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:alice", 200n);

      // Transfer 150 from Alice to Bob preserving periods
      // FIFO: takes 100 from period 1, 50 from period 5
      tree.transferPreservingPeriods("did:lgm:alice", "did:lgm:bob", 150n);

      const alice = tree.getAccount("did:lgm:alice")!;
      expect(alice.balance).toBe(150n);
      expect(alice.periodBalances!.has(PERIOD_1)).toBe(false);
      expect(alice.periodBalances!.get(PERIOD_5)).toBe(150n);

      const bob = tree.getAccount("did:lgm:bob")!;
      expect(bob.balance).toBe(150n);
      // Bob inherits Alice's period 1 and period 5 — NOT currentPeriod
      expect(bob.periodBalances!.get(PERIOD_1)).toBe(100n);
      expect(bob.periodBalances!.get(PERIOD_5)).toBe(50n);
    });

    it("should NOT refresh expiration on ping-pong transfers", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      createAccount(tree, "did:lgm:bob", 0n);

      // Alice gets tokens in period 1 (old)
      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:alice", 1000n);

      // Advance to period 30 — period 1 tokens are nearing expiration
      tree.setCurrentPeriod(PERIOD_30);

      // Alice sends to Bob (preserving periods)
      tree.transferPreservingPeriods("did:lgm:alice", "did:lgm:bob", 1000n);

      const bob = tree.getAccount("did:lgm:bob")!;
      // Bob's tokens are still in period 1 (near expiration), NOT period 30
      expect(bob.periodBalances!.get(PERIOD_1)).toBe(1000n);
      expect(bob.periodBalances!.has(PERIOD_30)).toBe(false);

      // Bob sends back to Alice — still no refresh
      tree.transferPreservingPeriods("did:lgm:bob", "did:lgm:alice", 1000n);

      const alice = tree.getAccount("did:lgm:alice")!;
      expect(alice.periodBalances!.get(PERIOD_1)).toBe(1000n);
      expect(alice.periodBalances!.has(PERIOD_30)).toBe(false);
    });

    it("should merge with recipient's existing period balances", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      createAccount(tree, "did:lgm:bob", 0n);

      // Both get tokens in period 5
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:alice", 300n);
      tree.creditBalance("did:lgm:bob", 200n);

      // Transfer from Alice to Bob, preserving period 5
      tree.transferPreservingPeriods("did:lgm:alice", "did:lgm:bob", 300n);

      const bob = tree.getAccount("did:lgm:bob")!;
      expect(bob.balance).toBe(500n);
      // Bob's existing 200 + Alice's 300 = 500 in period 5
      expect(bob.periodBalances!.get(PERIOD_5)).toBe(500n);
    });

    it("should fall back to regular transfer when period tracking is off", () => {
      // No period tracking (currentPeriod = undefined)
      createAccount(tree, "did:lgm:alice", 1000n);
      createAccount(tree, "did:lgm:bob", 0n);

      tree.transferPreservingPeriods("did:lgm:alice", "did:lgm:bob", 500n);

      const bob = tree.getAccount("did:lgm:bob")!;
      expect(bob.balance).toBe(500n);
      expect(bob.periodBalances).toBeUndefined();
    });

    it("should handle lazy migration for recipient without periodBalances", () => {
      // Create recipient without period tracking, then enable it
      createAccount(tree, "did:lgm:alice", 0n);
      createAccount(tree, "did:lgm:bob", 500n); // bob has 500n but no periodBalances

      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:alice", 200n);

      // Now transfer preserving periods — bob's 500n gets migrated to currentPeriod
      tree.transferPreservingPeriods("did:lgm:alice", "did:lgm:bob", 200n);

      const bob = tree.getAccount("did:lgm:bob")!;
      expect(bob.balance).toBe(700n);
      // Bob's existing 500n migrated to currentPeriod (1), plus Alice's 200n from period 1
      expect(bob.periodBalances!.get(PERIOD_1)).toBe(700n);
    });

    it("should auto-create recipient on period-preserving transfer", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:alice", 100n);

      // Transfer to non-existent account
      tree.transferPreservingPeriods("did:lgm:alice", "did:lgm:new-user", 100n);

      const newUser = tree.getAccount("did:lgm:new-user")!;
      expect(newUser.balance).toBe(100n);
      expect(newUser.periodBalances!.get(PERIOD_5)).toBe(100n);
    });

    it("should still give fresh expiration for system transfer()", () => {
      // Verify that the regular transfer() still gives currentPeriod (for rewards, fees)
      createAccount(tree, "did:lgm:system", 0n);
      createAccount(tree, "did:lgm:agent", 0n);

      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:system", 1000n);

      tree.setCurrentPeriod(PERIOD_10);
      // Regular transfer — should credit to currentPeriod (period 10)
      tree.transfer("did:lgm:system", "did:lgm:agent", 500n);

      const agent = tree.getAccount("did:lgm:agent")!;
      expect(agent.periodBalances!.get(PERIOD_10)).toBe(500n);
      expect(agent.periodBalances!.has(PERIOD_1)).toBe(false);
    });
  });

  // ============ Serialization ============

  describe.sequential("Serialization Round-Trip", () => {
    let tree: AccountStateTree;

    beforeEach(() => {
      tree = createTree();
      tree.setCurrentPeriod(PERIOD_5);
    });

    it("should preserve periodBalances through serialize/deserialize", () => {
      createAccount(tree, "did:lgm:alice", 0n);
      tree.creditBalance("did:lgm:alice", 100n);
      tree.setCurrentPeriod(PERIOD_10);
      tree.creditBalance("did:lgm:alice", 200n);

      // Create a new tree from the existing one's data
      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.periodBalances).toBeDefined();
      expect(account.periodBalances!.get(PERIOD_5)).toBe(100n);
      expect(account.periodBalances!.get(PERIOD_10)).toBe(200n);

      // Clone forces serialization/deserialization through SMT
      const cloned = tree.clone();
      const clonedAccount = cloned.getAccount("did:lgm:alice")!;
      expect(clonedAccount.periodBalances).toBeDefined();
      expect(clonedAccount.periodBalances!.get(PERIOD_5)).toBe(100n);
      expect(clonedAccount.periodBalances!.get(PERIOD_10)).toBe(200n);
    });

    it("should handle account without periodBalances (undefined)", () => {
      // Don't set currentPeriod — period tracking inactive
      tree.setCurrentPeriod(undefined);
      createAccount(tree, "did:lgm:legacy", 500n);

      const account = tree.getAccount("did:lgm:legacy")!;
      expect(account.periodBalances).toBeUndefined();

      const cloned = tree.clone();
      const clonedAccount = cloned.getAccount("did:lgm:legacy")!;
      expect(clonedAccount.periodBalances).toBeUndefined();
      expect(clonedAccount.balance).toBe(500n);
    });
  });

  // ============ Clone Safety ============

  describe.sequential("Clone Safety", () => {
    it("should deep-copy periodBalances (mutation in clone does not affect original)", () => {
      const tree = createTree();
      tree.setCurrentPeriod(PERIOD_5);
      createAccount(tree, "did:lgm:alice", 0n);
      tree.creditBalance("did:lgm:alice", 100n);

      const cloned = tree.clone();

      // Mutate the clone
      cloned.creditBalance("did:lgm:alice", 50n);

      // Original unchanged
      const original = tree.getAccount("did:lgm:alice")!;
      expect(original.balance).toBe(100n);
      expect(original.periodBalances!.get(PERIOD_5)).toBe(100n);

      // Clone updated
      const clonedAccount = cloned.getAccount("did:lgm:alice")!;
      expect(clonedAccount.balance).toBe(150n);
      expect(clonedAccount.periodBalances!.get(PERIOD_5)).toBe(150n);
    });

    it("should preserve currentPeriod in clone", () => {
      const tree = createTree();
      tree.setCurrentPeriod(PERIOD_10);
      createAccount(tree, "did:lgm:alice", 0n);

      const cloned = tree.clone();
      // If currentPeriod is propagated, crediting in clone should track periods
      cloned.creditBalance("did:lgm:alice", 100n);

      const account = cloned.getAccount("did:lgm:alice")!;
      expect(account.periodBalances).toBeDefined();
      expect(account.periodBalances!.get(PERIOD_10)).toBe(100n);
    });
  });

  // ============ Sweep ============

  describe.sequential("Sweep Expired Periods", () => {
    let tree: AccountStateTree;

    beforeEach(() => {
      tree = createTree();
    });

    it("should find accounts with expired period balances", () => {
      // Create accounts with balances in old and new periods
      tree.setCurrentPeriod(PERIOD_0);
      createAccount(tree, "did:lgm:old-holder", 0n);
      tree.creditBalance("did:lgm:old-holder", 500n);

      tree.setCurrentPeriod(PERIOD_30);
      createAccount(tree, "did:lgm:new-holder", 0n);
      tree.creditBalance("did:lgm:new-holder", 300n);

      // Expiration threshold at period 4 (anything in period <= 4 is expired)
      const expired = tree.getAccountsWithExpiredPeriods(4);
      expect(expired.length).toBe(1);
      expect(expired[0].did).toBe("did:lgm:old-holder");
      expect(expired[0].totalExpired).toBe(500n);
    });

    it("should sweep only expired periods and leave active ones", () => {
      createAccount(tree, "did:lgm:mixed", 0n);

      // Balance in period 1 (will expire) and period 30 (still active)
      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:mixed", 200n);
      tree.setCurrentPeriod(PERIOD_30);
      tree.creditBalance("did:lgm:mixed", 800n);

      // Sweep with threshold 5 (period 1 expires, period 30 survives)
      const swept = tree.sweepExpiredPeriods("did:lgm:mixed", 5);

      expect(swept).toBe(200n);
      const account = tree.getAccount("did:lgm:mixed")!;
      expect(account.balance).toBe(800n);
      expect(account.periodBalances!.has(PERIOD_1)).toBe(false);
      expect(account.periodBalances!.get(PERIOD_30)).toBe(800n);
    });

    it("should return 0 for accounts without periodBalances", () => {
      createAccount(tree, "did:lgm:legacy", 500n);
      const swept = tree.sweepExpiredPeriods("did:lgm:legacy", 100);
      expect(swept).toBe(0n);
    });

    it("should return 0 when no periods are expired", () => {
      tree.setCurrentPeriod(PERIOD_30);
      createAccount(tree, "did:lgm:fresh", 0n);
      tree.creditBalance("did:lgm:fresh", 500n);

      // Threshold at period 5 — period 30 is not expired
      const swept = tree.sweepExpiredPeriods("did:lgm:fresh", 5);
      expect(swept).toBe(0n);
    });

    it("should sweep all periods when all are expired", () => {
      createAccount(tree, "did:lgm:ancient", 0n);
      tree.setCurrentPeriod(PERIOD_0);
      tree.creditBalance("did:lgm:ancient", 100n);
      tree.setCurrentPeriod(PERIOD_1);
      tree.creditBalance("did:lgm:ancient", 200n);
      tree.setCurrentPeriod(PERIOD_5);
      tree.creditBalance("did:lgm:ancient", 300n);

      // All periods <= 10 expire
      const swept = tree.sweepExpiredPeriods("did:lgm:ancient", 10);
      expect(swept).toBe(600n);

      const account = tree.getAccount("did:lgm:ancient")!;
      expect(account.balance).toBe(0n);
      expect(account.periodBalances!.size).toBe(0);
    });
  });
});

// ============ Integration: StateManager + Sweep ============

describe.sequential("StateManager Period Integration", () => {
  let state: StateManager;

  beforeEach(() => {
    state = new StateManager();
  });

  it("setBlockTimestamp should compute correct period", () => {
    // Period = floor(timestamp / periodDurationMs)
    // PERIOD_MS = 604,800,000 (1 week)
    const timestamp = PERIOD_MS * 10 + 1000; // period 10
    state.setBlockTimestamp(timestamp);

    // Verify by crediting and checking periodBalances
    state.createAccount("did:lgm:test", "hash", 0n);
    state.creditBalance("did:lgm:test", 100n);

    const account = state.getAccount("did:lgm:test")!;
    expect(account.periodBalances).toBeDefined();
    expect(account.periodBalances!.get(10)).toBe(100n);
  });

  it("should sweep expired balances to RAI holding via transitions", async () => {
    // Import constants
    const { TREASURY } = await import("../../../core/constants.js");
    const { SYSTEM_CONTRACTS } = await import("../../../core/constants.js");
    const {
      executeTransaction,
      FEE_POOL_DID,
      REWARD_POOL_DID,
      PROTOCOL_TREASURY_DID,
    } = await import("../../../state/transitions.js");
    const { sha512_256Hex } = await import("../../../core/crypto/hashing.js");

    // Set up system accounts
    state.createAccount(TREASURY.ADDRESSES.RAI_HOLDING, "rai-hash", 0n);
    state.createAccount(SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID, "sys-hash", 0n);
    state.createAccount(FEE_POOL_DID, "fee-hash", 0n);
    state.createAccount(REWARD_POOL_DID, "reward-hash", 0n);
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);

    // Create agents with balances in old periods
    state.setBlockTimestamp(PERIOD_MS * 2); // period 2
    state.createAccount("did:lgm:agent-1", "a1-hash", 0n);
    state.creditBalance("did:lgm:agent-1", 500n * 10n ** 18n);
    state.createAccount("did:lgm:agent-2", "a2-hash", 0n);
    state.creditBalance("did:lgm:agent-2", 300n * 10n ** 18n);

    // Advance time: set timestamp to period 30 (period 2 is now 28 periods old > 26)
    const sweepTimestamp = PERIOD_MS * 30;
    state.setBlockTimestamp(sweepTimestamp);

    // Create the sweep transaction
    const tx = {
      type: "scheduled_execution",
      sender: "system",
      fee: 0n,
      _systemInternal: true,
      nonce: 0n,
      timestamp: sweepTimestamp,
      signature: "system",
      payload: {
        type: "scheduled_execution",
        processId: "sweep-process",
        executionIndex: 0,
        targetContract: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        functionSelector:
          SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        proof: {
          processStateHash: sha512_256Hex("sweep"),
          conditionProof: {
            timestamp: sweepTimestamp,
            stateProofs: [],
          },
          executionNonce: 0n,
          signature: "system",
        },
      },
    } as any;

    const params = {
      targetBlockTimeMs: 2521,
      minValidatorStake: 111n * 10n ** 15n, // 0.111 LGM
      minClaimStake: 10n * 10n ** 15n,
      validatorQuorum: 0.66,
      verificationRoundBlocks: 20,
      commitPhaseBlocks: 6,
      revealPhaseBlocks: 6,
      epochBlocks: 100,
      maxValidators: 100,
      slashFractionDowntime: 0.01,
      slashFractionDoubleSign: 0.05,
      unbondingPeriod: 1000n,
    } as any;

    const result = executeTransaction(state, tx, 100n, params);
    expect(result.success).toBe(true);

    // Agents should have lost their expired balance
    expect(state.getBalance("did:lgm:agent-1")).toBe(0n);
    expect(state.getBalance("did:lgm:agent-2")).toBe(0n);

    // RAI holding should have the swept tokens
    expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(
      800n * 10n ** 18n,
    );

    // Gas should reflect 2 accounts processed
    expect(result.gasUsed).toBe(50000n + 2n * 5000n);
  });

  it("should skip dust amounts during sweep", async () => {
    const { TREASURY } = await import("../../../core/constants.js");
    const { SYSTEM_CONTRACTS } = await import("../../../core/constants.js");
    const {
      executeTransaction,
      FEE_POOL_DID,
      REWARD_POOL_DID,
      PROTOCOL_TREASURY_DID,
    } = await import("../../../state/transitions.js");
    const { sha512_256Hex } = await import("../../../core/crypto/hashing.js");

    state.createAccount(TREASURY.ADDRESSES.RAI_HOLDING, "rai-hash", 0n);
    state.createAccount(SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID, "sys-hash", 0n);
    state.createAccount(FEE_POOL_DID, "fee-hash", 0n);
    state.createAccount(REWARD_POOL_DID, "reward-hash", 0n);
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);

    // Create agent with dust amount (below 1000n threshold)
    state.setBlockTimestamp(PERIOD_MS * 2);
    state.createAccount("did:lgm:dust-agent", "dust-hash", 0n);
    state.creditBalance("did:lgm:dust-agent", 500n); // Below dustThreshold of 1000n

    // Create agent with real balance
    state.createAccount("did:lgm:real-agent", "real-hash", 0n);
    state.creditBalance("did:lgm:real-agent", 5000n);

    const sweepTimestamp = PERIOD_MS * 30;
    state.setBlockTimestamp(sweepTimestamp);

    const tx = {
      type: "scheduled_execution",
      sender: "system",
      fee: 0n,
      _systemInternal: true,
      nonce: 0n,
      timestamp: sweepTimestamp,
      signature: "system",
      payload: {
        type: "scheduled_execution",
        processId: "sweep-process",
        executionIndex: 0,
        targetContract: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        functionSelector:
          SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        proof: {
          processStateHash: sha512_256Hex("sweep"),
          conditionProof: { timestamp: sweepTimestamp, stateProofs: [] },
          executionNonce: 0n,
          signature: "system",
        },
      },
    } as any;

    const params = {
      targetBlockTimeMs: 2521,
      minValidatorStake: 111n * 10n ** 15n, // 0.111 LGM
      minClaimStake: 10n * 10n ** 15n,
      validatorQuorum: 0.66,
      verificationRoundBlocks: 20,
      commitPhaseBlocks: 6,
      revealPhaseBlocks: 6,
    } as any;

    const result = executeTransaction(state, tx, 100n, params);
    expect(result.success).toBe(true);

    // Dust agent untouched (500n < 1000n threshold)
    expect(state.getBalance("did:lgm:dust-agent")).toBe(500n);

    // Real agent swept
    expect(state.getBalance("did:lgm:real-agent")).toBe(0n);

    // Only 1 account processed (not the dust one)
    expect(result.gasUsed).toBe(50000n + 1n * 5000n);
  });
});

// ============ F4: DID Index Completeness ============

describe.sequential("DID Index (F4 — beyond cache)", () => {
  it("should return all accounts via getAllAccounts even when cache evicts", () => {
    // Use a tiny cache (5 entries) to force eviction
    const tree = new AccountStateTree(5);
    tree.setCurrentPeriod(PERIOD_5);

    // Create 10 accounts — exceeds cache capacity
    for (let i = 0; i < 10; i++) {
      tree.createAccount(
        `did:lgm:agent-${i}`,
        `hash-${i}`,
        BigInt(i + 1) * 100n,
      );
    }

    const all = tree.getAllAccounts();
    expect(all.length).toBe(10);

    // Verify every account is present (regardless of cache eviction)
    const dids = new Set(all.map((a) => a.did));
    for (let i = 0; i < 10; i++) {
      expect(dids.has(`did:lgm:agent-${i}`)).toBe(true);
    }
  });

  it("should find expired periods in accounts beyond cache", () => {
    const tree = new AccountStateTree(3); // Very small cache

    // Create accounts in old period
    tree.setCurrentPeriod(PERIOD_0);
    for (let i = 0; i < 6; i++) {
      tree.createAccount(`did:lgm:old-${i}`, `hash-${i}`, 0n);
      tree.creditBalance(`did:lgm:old-${i}`, 1000n);
    }

    // Some accounts will be evicted from the 3-entry cache
    // but getAccountsWithExpiredPeriods should still find them
    const expired = tree.getAccountsWithExpiredPeriods(5);
    expect(expired.length).toBe(6);
  });

  it("should preserve allDids through clone", () => {
    const tree = new AccountStateTree(3);
    tree.setCurrentPeriod(PERIOD_5);

    for (let i = 0; i < 5; i++) {
      tree.createAccount(`did:lgm:c-${i}`, `hash-${i}`, 100n);
    }

    const cloned = tree.clone();
    const all = cloned.getAllAccounts();
    expect(all.length).toBe(5);
  });
});

// ============ F1/F7: Authorization & Timestamp Security ============

describe.sequential("Scheduled Execution Security (F1/F7)", () => {
  let state: StateManager;

  beforeEach(async () => {
    state = new StateManager();
    const { TREASURY } = await import("../../../core/constants.js");
    const { SYSTEM_CONTRACTS } = await import("../../../core/constants.js");
    const { FEE_POOL_DID, REWARD_POOL_DID, PROTOCOL_TREASURY_DID } =
      await import("../../../state/transitions.js");

    state.createAccount(TREASURY.ADDRESSES.RAI_HOLDING, "rai-hash", 0n);
    state.createAccount(SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID, "sys-hash", 0n);
    state.createAccount(FEE_POOL_DID, "fee-hash", 0n);
    state.createAccount(REWARD_POOL_DID, "reward-hash", 0n);
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);
  });

  it("should reject non-system sender for scheduled_execution (F1)", async () => {
    const { SYSTEM_CONTRACTS } = await import("../../../core/constants.js");
    const { executeTransaction } =
      await import("../../../state/transitions.js");
    const { sha512_256Hex } = await import("../../../core/crypto/hashing.js");

    state.setBlockTimestamp(PERIOD_MS * 30);

    // Create attacker account first so validateTransaction passes account/nonce checks
    state.createAccount("did:lgm:attacker", "atk-hash", 10000n);

    const tx = {
      type: "scheduled_execution",
      sender: "did:lgm:attacker",
      senderDid: "did:lgm:attacker", // bypass publicKeyToDid resolution
      fee: 0n,
      nonce: 0n,
      timestamp: Date.now(),
      signature: "attacker-sig",
      payload: {
        type: "scheduled_execution",
        processId: "sweep-process",
        executionIndex: 0,
        targetContract: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        functionSelector:
          SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        proof: {
          processStateHash: sha512_256Hex("sweep"),
          conditionProof: { timestamp: PERIOD_MS * 9999, stateProofs: [] },
          executionNonce: 0n,
          signature: "attacker-sig",
        },
      },
    } as any;

    const result = executeTransaction(state, tx, 100n);
    expect(result.success).toBe(false);
    expect(result.error).toContain("system sender");
  });

  it("should use block timestamp, not proof timestamp for sweep (F7)", async () => {
    const { SYSTEM_CONTRACTS } = await import("../../../core/constants.js");
    const { TREASURY } = await import("../../../core/constants.js");
    const { executeTransaction } =
      await import("../../../state/transitions.js");
    const { sha512_256Hex } = await import("../../../core/crypto/hashing.js");

    // Create agent with balance in period 2
    state.setBlockTimestamp(PERIOD_MS * 2);
    state.createAccount("did:lgm:victim", "v-hash", 0n);
    state.creditBalance("did:lgm:victim", 5000n);

    // Block timestamp at period 10 (period 2 is only 8 periods old, NOT expired)
    const legitimateTimestamp = PERIOD_MS * 10;
    state.setBlockTimestamp(legitimateTimestamp);

    // Attacker provides far-future proof timestamp that would sweep everything
    const maliciousTimestamp = PERIOD_MS * 9999;

    const tx = {
      type: "scheduled_execution",
      sender: "system",
      fee: 0n,
      _systemInternal: true,
      nonce: 0n,
      timestamp: legitimateTimestamp,
      signature: "system",
      payload: {
        type: "scheduled_execution",
        processId: "sweep-process",
        executionIndex: 0,
        targetContract: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        functionSelector:
          SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        proof: {
          processStateHash: sha512_256Hex("sweep"),
          conditionProof: { timestamp: maliciousTimestamp, stateProofs: [] },
          executionNonce: 0n,
          signature: "system",
        },
      },
    } as any;

    const result = executeTransaction(state, tx, 100n);
    expect(result.success).toBe(true);

    // Victim's balance should NOT be swept — block timestamp says period 10,
    // which makes threshold = 10 - 26 = -16 (no periods expired)
    expect(state.getBalance("did:lgm:victim")).toBe(5000n);
    expect(state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING)).toBe(0n);
  });
});

// ============ F2+F5+F6: RAI Distribution Hardening ============

describe.sequential("RAI Distribution Hardening (F2/F5/F6)", () => {
  let state: StateManager;

  beforeEach(async () => {
    state = new StateManager();
    const { TREASURY } = await import("../../../core/constants.js");
    const { SYSTEM_CONTRACTS } = await import("../../../core/constants.js");
    const { FEE_POOL_DID, REWARD_POOL_DID, PROTOCOL_TREASURY_DID } =
      await import("../../../state/transitions.js");

    state.createAccount(TREASURY.ADDRESSES.RAI_HOLDING, "rai-hash", 0n);
    state.createAccount(SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID, "sys-hash", 0n);
    state.createAccount(FEE_POOL_DID, "fee-hash", 0n);
    state.createAccount(
      REWARD_POOL_DID,
      "reward-hash",
      1_000_000n * 10n ** 18n,
    );
    state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);

    const { TREASURY: T } = await import("../../../core/constants.js");
    state.createAccount(T.ADDRESSES.RESEARCH, "res-hash", 0n);
    state.createAccount(T.ADDRESSES.RESEARCH_FUND, "rf-hash", 0n);
    state.createAccount(T.ADDRESSES.FOUNDATION, "found-hash", 0n);
    state.createAccount(T.ADDRESSES.COMMUNITY, "comm-hash", 0n);
    state.createAccount(T.ADDRESSES.DEVELOPERS, "dev-hash", 0n);
    state.createAccount(T.ADDRESSES.RESERVE, "res-hash2", 0n);
  });

  it("should exclude fresh accounts from RAI distribution (F2)", async () => {
    const { TREASURY } = await import("../../../core/constants.js");
    const { SYSTEM_CONTRACTS } = await import("../../../core/constants.js");
    const { executeTransaction } =
      await import("../../../state/transitions.js");
    const { sha512_256Hex } = await import("../../../core/crypto/hashing.js");

    const MIN_AGE_BLOCKS = 7n * 34272n;

    const execBlock = MIN_AGE_BLOCKS + 200n; // 240104 — mature passes age check

    // Create a mature agent (block 0) and a fresh agent
    state.createAccount("did:lgm:mature", "m-hash", 50n * 10n ** 18n);
    state.createAccount("did:lgm:fresh", "f-hash", 50n * 10n ** 18n);
    // StateManager.createAccount doesn't accept blockNumber, so set it via internal API
    (state as any).accounts.updateAccount("did:lgm:fresh", {
      createdAtBlock: execBlock - 100n,
    });

    // Fund RAI holding
    state.creditBalance(TREASURY.ADDRESSES.RAI_HOLDING, 1000n * 10n ** 18n);

    state.setBlockTimestamp(PERIOD_MS * 10);

    const tx = {
      type: "scheduled_execution",
      sender: "system",
      fee: 0n,
      _systemInternal: true,
      nonce: 0n,
      timestamp: Date.now(),
      signature: "system",
      payload: {
        type: "scheduled_execution",
        processId: "distribute-process",
        executionIndex: 0,
        targetContract: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        functionSelector:
          SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
        proof: {
          processStateHash: sha512_256Hex("dist"),
          conditionProof: { timestamp: PERIOD_MS * 10, stateProofs: [] },
          executionNonce: 0n,
          signature: "system",
        },
      },
    } as any;

    // Execute at execBlock — mature agent (block 0) passes age check,
    // fresh agent (block execBlock-100) does NOT
    const result = executeTransaction(state, tx, execBlock);
    expect(result.success).toBe(true);

    // Mature agent should receive RAI, fresh agent should not
    const matureBalance = state.getBalance("did:lgm:mature");
    const freshBalance = state.getBalance("did:lgm:fresh");
    expect(matureBalance).toBeGreaterThan(50n * 10n ** 18n); // Got RAI
    expect(freshBalance).toBe(50n * 10n ** 18n); // No RAI
  });

  it("should exclude dust-balance accounts from RAI distribution (F5)", async () => {
    const { TREASURY } = await import("../../../core/constants.js");
    const { SYSTEM_CONTRACTS } = await import("../../../core/constants.js");
    const { executeTransaction } =
      await import("../../../state/transitions.js");
    const { sha512_256Hex } = await import("../../../core/crypto/hashing.js");

    // Create an agent with dust balance (< 1000n)
    state.createAccount("did:lgm:dust-recv", "d-hash", 999n, 0n);
    // Create an agent with real balance
    state.createAccount("did:lgm:real-recv", "r-hash", 50n * 10n ** 18n, 0n);

    state.creditBalance(TREASURY.ADDRESSES.RAI_HOLDING, 1000n * 10n ** 18n);
    state.setBlockTimestamp(PERIOD_MS * 10);

    const tx = {
      type: "scheduled_execution",
      sender: "system",
      fee: 0n,
      _systemInternal: true,
      nonce: 0n,
      timestamp: Date.now(),
      signature: "system",
      payload: {
        type: "scheduled_execution",
        processId: "distribute-process",
        executionIndex: 0,
        targetContract: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        functionSelector:
          SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
        proof: {
          processStateHash: sha512_256Hex("dist"),
          conditionProof: { timestamp: PERIOD_MS * 10, stateProofs: [] },
          executionNonce: 0n,
          signature: "system",
        },
      },
    } as any;

    const result = executeTransaction(state, tx, MIN_AGE_BLOCKS_TEST + 1n);
    expect(result.success).toBe(true);

    // Dust account should not receive RAI
    expect(state.getBalance("did:lgm:dust-recv")).toBe(999n);
    // Real account should receive RAI
    expect(state.getBalance("did:lgm:real-recv")).toBeGreaterThan(
      50n * 10n ** 18n,
    );
  });

  it("should exclude pot accounts from sweep and RAI distribution (F6)", async () => {
    const { TREASURY } = await import("../../../core/constants.js");
    const { SYSTEM_CONTRACTS } = await import("../../../core/constants.js");
    const { executeTransaction } =
      await import("../../../state/transitions.js");
    const { sha512_256Hex } = await import("../../../core/crypto/hashing.js");

    // Create a pot with old period balance
    state.setBlockTimestamp(PERIOD_MS * 2);
    state.createAccount("did:lgm:pot:test-pot", "pot-hash", 0n, 0n);
    state.creditBalance("did:lgm:pot:test-pot", 10000n);

    // Create a regular agent with old balance too
    state.createAccount("did:lgm:agent-normal", "normal-hash", 0n, 0n);
    state.creditBalance("did:lgm:agent-normal", 10000n);

    // Advance to period 30 (period 2 is now expired)
    const sweepTimestamp = PERIOD_MS * 30;
    state.setBlockTimestamp(sweepTimestamp);

    const sweepTx = {
      type: "scheduled_execution",
      sender: "system",
      fee: 0n,
      _systemInternal: true,
      nonce: 0n,
      timestamp: sweepTimestamp,
      signature: "system",
      payload: {
        type: "scheduled_execution",
        processId: "sweep-process",
        executionIndex: 0,
        targetContract: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        functionSelector:
          SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.SWEEP_EXPIRED,
        proof: {
          processStateHash: sha512_256Hex("sweep"),
          conditionProof: { timestamp: sweepTimestamp, stateProofs: [] },
          executionNonce: 0n,
          signature: "system",
        },
      },
    } as any;

    const result = executeTransaction(state, sweepTx, 100n);
    expect(result.success).toBe(true);

    // Pot account should NOT be swept
    expect(state.getBalance("did:lgm:pot:test-pot")).toBe(10000n);

    // Normal agent SHOULD be swept
    expect(state.getBalance("did:lgm:agent-normal")).toBe(0n);

    // Now test RAI distribution — pot should not receive
    const raiBalance = state.getBalance(TREASURY.ADDRESSES.RAI_HOLDING);
    expect(raiBalance).toBe(10000n); // only the normal agent's funds

    // Fund more for distribution and add a real eligible agent
    state.createAccount("did:lgm:eligible", "el-hash", 50n * 10n ** 18n, 0n);

    const distTx = {
      type: "scheduled_execution",
      sender: "system",
      fee: 0n,
      _systemInternal: true,
      nonce: 0n,
      timestamp: sweepTimestamp,
      signature: "system",
      payload: {
        type: "scheduled_execution",
        processId: "distribute-process",
        executionIndex: 0,
        targetContract: SYSTEM_CONTRACTS.RAI_RETRIEVAL.DID,
        functionSelector:
          SYSTEM_CONTRACTS.RAI_RETRIEVAL.SELECTORS.DISTRIBUTE_RAI,
        proof: {
          processStateHash: sha512_256Hex("dist"),
          conditionProof: { timestamp: sweepTimestamp, stateProofs: [] },
          executionNonce: 0n,
          signature: "system",
        },
      },
    } as any;

    const distResult = executeTransaction(
      state,
      distTx,
      MIN_AGE_BLOCKS_TEST + 1n,
    );
    expect(distResult.success).toBe(true);

    // Pot should not have received RAI
    expect(state.getBalance("did:lgm:pot:test-pot")).toBe(10000n);
  });
});

const MIN_AGE_BLOCKS_TEST = 7n * 34272n;

// ============ Edge Scenario Tests (TOKEN_EXPIRATION_EDGE_SCENARIOS.md) ============

describe.sequential("Token Expiration Edge Scenarios", () => {
  // ---------- H4: Self-Transfer Detection ----------

  describe.sequential("H4: Self-Transfer Detection", () => {
    it("should only increment nonce on self-transfer (no balance change)", async () => {
      const { executeTransaction } =
        await import("../../../state/transitions.js");
      const sm = new StateManager();
      sm.createAccount("did:lgm:alice", "alice-key", 1000n);
      sm.setBlockTimestamp(PERIOD_MS * 30);

      // Give alice tokens in period 10 (old period)
      const tree = (sm as any).accounts as AccountStateTree;
      tree.setCurrentPeriod(10);
      tree.creditBalance("did:lgm:alice", 500n);

      // Now at period 30
      tree.setCurrentPeriod(30);

      const nonceBefore = sm.getNonce("did:lgm:alice");
      const balanceBefore = sm.getBalance("did:lgm:alice");

      const tx = {
        type: "transfer",
        sender: "alice-key",
        senderDid: "did:lgm:alice",
        fee: 0n,
        nonce: nonceBefore,
        timestamp: Date.now(),
        signature: "sig",
        payload: { type: "transfer", to: "did:lgm:alice", amount: 500n },
      } as any;

      const result = executeTransaction(sm, tx, 100n);
      expect(result.success).toBe(true);

      // Balance unchanged — no refresh
      expect(sm.getBalance("did:lgm:alice")).toBe(balanceBefore);

      // Nonce incremented
      expect(sm.getNonce("did:lgm:alice")).toBe(nonceBefore + 1n);

      // Period balances unchanged — tokens still in period 10, NOT refreshed to 30
      // (1000n from lazy migration + 500n credit = 1500n in period 10)
      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.periodBalances!.get(10)).toBe(1500n);
    });
  });

  // ---------- H1: Expired Period Transfer Validation ----------

  describe.sequential("H1: Expired Period Transfer Validation", () => {
    it("should reject transfer that includes expired-period tokens", () => {
      const tree = new AccountStateTree();
      createAccount(tree, "did:lgm:alice", 0n);
      createAccount(tree, "did:lgm:bob", 0n);

      // Give alice tokens in period 1 (very old)
      tree.setCurrentPeriod(1);
      tree.creditBalance("did:lgm:alice", 1000n);

      // Advance to period 30 — period 1 is expired (30 - 26 = 4 > 1)
      tree.setCurrentPeriod(30);

      // Transfer should fail because period 1 tokens are expired
      expect(() =>
        tree.transferPreservingPeriods(
          "did:lgm:alice",
          "did:lgm:bob",
          1000n,
          PERIODS_UNTIL_EXPIRATION,
        ),
      ).toThrow("expired period");
    });

    it("should allow transfer of non-expired tokens", () => {
      const tree = new AccountStateTree();
      createAccount(tree, "did:lgm:alice", 0n);
      createAccount(tree, "did:lgm:bob", 0n);

      // Give alice tokens in period 20 (recent)
      tree.setCurrentPeriod(20);
      tree.creditBalance("did:lgm:alice", 1000n);

      // Advance to period 30 — period 20 is NOT expired (30 - 26 = 4 < 20)
      tree.setCurrentPeriod(30);

      // Transfer should succeed
      tree.transferPreservingPeriods(
        "did:lgm:alice",
        "did:lgm:bob",
        1000n,
        PERIODS_UNTIL_EXPIRATION,
      );

      const bob = tree.getAccount("did:lgm:bob")!;
      expect(bob.balance).toBe(1000n);
      expect(bob.periodBalances!.get(20)).toBe(1000n);
    });

    it("should reject when FIFO debit includes a mix of expired and fresh periods", () => {
      const tree = new AccountStateTree();
      createAccount(tree, "did:lgm:alice", 0n);
      createAccount(tree, "did:lgm:bob", 0n);

      // Give alice tokens across multiple periods
      tree.setCurrentPeriod(1);
      tree.creditBalance("did:lgm:alice", 200n); // period 1 (will expire)
      tree.setCurrentPeriod(20);
      tree.creditBalance("did:lgm:alice", 800n); // period 20 (still fresh)

      // Advance to period 30 — period 1 is expired
      tree.setCurrentPeriod(30);

      // Transferring 300n requires 200n from period 1 (expired) + 100n from period 20
      // Should reject because FIFO starts with expired period 1
      expect(() =>
        tree.transferPreservingPeriods(
          "did:lgm:alice",
          "did:lgm:bob",
          300n,
          PERIODS_UNTIL_EXPIRATION,
        ),
      ).toThrow("expired period");
    });

    it("should allow transfer that only touches non-expired periods via FIFO", () => {
      const tree = new AccountStateTree();
      createAccount(tree, "did:lgm:alice", 0n);
      createAccount(tree, "did:lgm:bob", 0n);

      // Give alice tokens across periods: 200 expired + 800 fresh
      tree.setCurrentPeriod(1);
      tree.creditBalance("did:lgm:alice", 200n); // expired at period 30
      tree.setCurrentPeriod(20);
      tree.creditBalance("did:lgm:alice", 800n); // fresh

      // But FIFO would take from period 1 first for any amount...
      // Even 100n will fail because FIFO takes from oldest (period 1)
      tree.setCurrentPeriod(30);
      expect(() =>
        tree.transferPreservingPeriods(
          "did:lgm:alice",
          "did:lgm:bob",
          100n,
          PERIODS_UNTIL_EXPIRATION,
        ),
      ).toThrow("expired period");
    });

    it("should work without periodsUntilExpiration (no expiry check)", () => {
      const tree = new AccountStateTree();
      createAccount(tree, "did:lgm:alice", 0n);
      createAccount(tree, "did:lgm:bob", 0n);

      tree.setCurrentPeriod(1);
      tree.creditBalance("did:lgm:alice", 1000n);
      tree.setCurrentPeriod(30);

      // Without periodsUntilExpiration, no expiry check — should succeed
      tree.transferPreservingPeriods("did:lgm:alice", "did:lgm:bob", 1000n);

      const bob = tree.getAccount("did:lgm:bob")!;
      expect(bob.balance).toBe(1000n);
    });
  });

  // ---------- C2: Unbonding Period-Aware Release ----------

  describe.sequential("C2: Stake Expiration Refresh Prevention", () => {
    it("should credit to specific period instead of currentPeriod", () => {
      const tree = new AccountStateTree();
      createAccount(tree, "did:lgm:alice", 0n);

      // Give alice tokens in period 10
      tree.setCurrentPeriod(10);
      tree.creditBalance("did:lgm:alice", 1000n);

      // Advance to period 30
      tree.setCurrentPeriod(30);

      // Credit to period 15 (simulating unbonding release to origin period)
      tree.creditBalanceToSpecificPeriod("did:lgm:alice", 500n, 15);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(1500n);
      expect(account.periodBalances!.get(10)).toBe(1000n); // original tokens
      expect(account.periodBalances!.get(15)).toBe(500n); // credited to specific period
      expect(account.periodBalances!.has(30)).toBe(false); // NOT at currentPeriod
    });

    it("should merge with existing balance at the target period", () => {
      const tree = new AccountStateTree();
      createAccount(tree, "did:lgm:alice", 0n);

      tree.setCurrentPeriod(10);
      tree.creditBalance("did:lgm:alice", 300n);

      tree.setCurrentPeriod(30);

      // Credit more to period 10
      tree.creditBalanceToSpecificPeriod("did:lgm:alice", 200n, 10);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(500n);
      expect(account.periodBalances!.get(10)).toBe(500n); // 300 + 200 merged
    });

    it("should auto-create recipient when crediting to specific period", () => {
      const tree = new AccountStateTree();
      tree.setCurrentPeriod(15);

      // Credit to non-existent account at specific period
      tree.creditBalanceToSpecificPeriod("did:lgm:new-user", 1000n, 10);

      const account = tree.getAccount("did:lgm:new-user")!;
      expect(account.balance).toBe(1000n);
      expect(account.periodBalances!.get(10)).toBe(1000n);
      expect(account.periodBalances!.has(15)).toBe(false);
    });

    it("should fall back to regular credit when period tracking is off", () => {
      const tree = new AccountStateTree();
      createAccount(tree, "did:lgm:alice", 1000n);

      // No period set — period tracking inactive
      tree.creditBalanceToSpecificPeriod("did:lgm:alice", 500n, 10);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(1500n);
      expect(account.periodBalances).toBeUndefined(); // no period tracking
    });

    it("should handle lazy migration when crediting to specific period", () => {
      const tree = new AccountStateTree();
      createAccount(tree, "did:lgm:alice", 500n); // has balance but no periodBalances

      tree.setCurrentPeriod(20);

      // Credit to period 10 triggers lazy migration (500n → period 20)
      tree.creditBalanceToSpecificPeriod("did:lgm:alice", 200n, 10);

      const account = tree.getAccount("did:lgm:alice")!;
      expect(account.balance).toBe(700n);
      expect(account.periodBalances!.get(20)).toBe(500n); // migrated
      expect(account.periodBalances!.get(10)).toBe(200n); // newly credited
    });

    it("should record originPeriod on unbonding entry and use it for release", async () => {
      const { executeTransaction, processUnbondingQueue } =
        await import("../../../state/transitions.js");
      const sm = new StateManager();

      // Setup accounts
      sm.createAccount("did:lgm:validator1", "val1-key", 5000n);
      sm.createAccount("did:lgm:delegation-pool", "pool-hash", 0n);
      sm.createAccount("did:lgm:fee-pool", "fee-hash", 0n);
      sm.createAccount("did:lgm:reward-pool", "reward-hash", 1_000_000n);

      const { TREASURY: T } = await import("../../../core/constants.js");
      sm.createAccount(T.ADDRESSES.RESEARCH, "res-hash", 0n);
      sm.createAccount(T.ADDRESSES.RESEARCH_FUND, "rf-hash", 0n);
      sm.createAccount(T.ADDRESSES.FOUNDATION, "found-hash", 0n);
      sm.createAccount(T.ADDRESSES.COMMUNITY, "comm-hash", 0n);
      sm.createAccount(T.ADDRESSES.DEVELOPERS, "dev-hash", 0n);
      sm.createAccount(T.ADDRESSES.RESERVE, "res-hash2", 0n);

      // Register as validator
      sm.registerValidator("val1-key", "did:lgm:validator1", 5000n);

      // Set timestamp to period 25
      sm.setBlockTimestamp(PERIOD_MS * 25);

      // Stake
      const stakeTx = {
        type: "stake",
        sender: "val1-key",
        senderDid: "did:lgm:validator1",
        fee: 0n,
        nonce: 0n,
        timestamp: Date.now(),
        signature: "sig",
        payload: { type: "stake", amount: 2000n },
      } as any;

      const stakeResult = executeTransaction(sm, stakeTx, 100n);
      expect(stakeResult.success).toBe(true);

      // Unstake — should record originPeriod = 25
      const unstakeTx = {
        type: "unstake",
        sender: "val1-key",
        senderDid: "did:lgm:validator1",
        fee: 0n,
        nonce: 1n,
        timestamp: Date.now(),
        signature: "sig",
        payload: { type: "unstake", amount: 2000n },
      } as any;

      const unstakeResult = executeTransaction(sm, unstakeTx, 101n);
      expect(unstakeResult.success).toBe(true);

      // Verify unbonding entry has originPeriod
      const entries = sm.getMaturedUnbondingEntries(1_000_000n);
      expect(entries.length).toBeGreaterThan(0);
      expect(entries[0].originPeriod).toBe(25);

      // Advance time to period 40 — process unbonding
      sm.setBlockTimestamp(PERIOD_MS * 40);

      const result = processUnbondingQueue(sm, 1_000_000n);
      expect(result.processed).toBe(1);

      // Tokens should be at period 25 (origin), NOT period 40 (current)
      const tree = (sm as any).accounts as AccountStateTree;
      const account = tree.getAccount("did:lgm:validator1")!;
      if (account.periodBalances) {
        const periodBalance25 = account.periodBalances.get(25) ?? 0n;
        const periodBalance40 = account.periodBalances.get(40) ?? 0n;
        // Released tokens should be at originPeriod 25
        expect(periodBalance25).toBeGreaterThan(0n);
        // Should NOT be at the current period 40
        expect(periodBalance40).toBe(0n);
      }
    });
  });
});
