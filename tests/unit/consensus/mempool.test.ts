/**
 * Mempool Tests
 *
 * Comprehensive unit tests for the transaction mempool:
 * addition, removal, retrieval, capacity, block selection,
 * eviction, stats, specialized pools, events, and clear.
 *
 * NOTE: All tests are self-contained (no shared mutable state via beforeEach)
 * because the vitest config sets sequence.concurrent: true, which runs tests
 * concurrently within describe blocks.
 */

import { describe, it, expect, vi } from "vitest";
import {
  Mempool,
  DEFAULT_MEMPOOL_CONFIG,
  createClaimPool,
  createVerificationPool,
} from "../../../consensus/mempool/pool.js";
import type { MempoolEntry } from "../../../consensus/mempool/pool.js";
import { generateKeypair } from "../../../core/crypto/keys.js";
import { sign } from "../../../core/crypto/signatures.js";
import type { Transaction } from "../../../consensus/pot/types.js";

// ============ Test Helpers ============

/** Generate a fresh keypair + mempool for each test. */
function setup(config?: Parameters<typeof Mempool>[0], clock?: () => number) {
  return {
    mempool: new Mempool(config ?? {}, clock),
    keyA: generateKeypair(),
    keyB: generateKeypair(),
  };
}

/**
 * Create a signed transaction whose signature matches the mempool's
 * validation convention: sign(JSON.stringify(tx.payload)).
 */
function createSignedTx(
  privateKey: string,
  publicKey: string,
  overrides?: Partial<{
    type: string;
    nonce: bigint;
    fee: bigint;
    payload: unknown;
  }>,
): Transaction {
  const payload = overrides?.payload ?? {
    type: "transfer",
    to: "recipient",
    amount: "100",
  };
  const payloadBytes = new TextEncoder().encode(JSON.stringify(payload));
  const signature = sign(payloadBytes, privateKey);

  return {
    type: (overrides?.type ?? "transfer") as Transaction["type"],
    sender: publicKey,
    nonce: overrides?.nonce ?? 0n,
    fee: overrides?.fee ?? 10000n,
    payload,
    signature,
  } as Transaction;
}

/**
 * Create a deliberately mis-signed transaction (signs with wrong data).
 */
function createBadSigTx(publicKey: string, privateKey: string): Transaction {
  const payload = { type: "transfer", to: "recipient", amount: "100" };
  const wrongData = new TextEncoder().encode("wrong-data");
  const badSignature = sign(wrongData, privateKey);

  return {
    type: "transfer" as Transaction["type"],
    sender: publicKey,
    nonce: 0n,
    fee: 10000n,
    payload,
    signature: badSignature,
  } as Transaction;
}

// ============ Tests ============

describe("Mempool", () => {
  // ============ Transaction Addition ============

  describe("Transaction Addition", () => {
    it("should add a valid transaction", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      const result = mempool.add(tx);

      expect(result.added).toBe(true);
      expect(mempool.size()).toBe(1);
    });

    it("should return txHash on successful add", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      const result = mempool.add(tx);

      expect(result.added).toBe(true);
      expect(result.txHash).toBeDefined();
      expect(typeof result.txHash).toBe("string");
      expect(result.txHash!.length).toBeGreaterThan(0);
    });

    it("should reject duplicate transactions", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      mempool.add(tx);

      const result = mempool.add(tx);
      expect(result.added).toBe(false);
      expect(result.reason).toMatch(/already in pool/i);
      expect(mempool.size()).toBe(1);
    });

    it("should reject transaction with fee below minimum", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey, {
        fee: 0n,
      });
      const result = mempool.add(tx);

      expect(result.added).toBe(false);
      expect(result.reason).toMatch(/fee too low/i);
    });

    it("should reject transaction with invalid signature", () => {
      const { mempool, keyA } = setup();
      const tx = createBadSigTx(keyA.publicKey, keyA.privateKey);
      const result = mempool.add(tx);

      expect(result.added).toBe(false);
      expect(result.reason).toMatch(/invalid signature/i);
    });

    it("should reject transaction exceeding max size", () => {
      const { keyA } = setup();
      const smallPool = new Mempool({ maxTxSize: 10 });
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      const result = smallPool.add(tx);

      expect(result.added).toBe(false);
      expect(result.reason).toMatch(/too large/i);
    });

    it("should accept well-formed transactions with non-zero serialized size", () => {
      // The zero-size guard (pool.ts:105-107) is impossible to trigger with
      // real Transaction objects because JSON.stringify always produces > 0
      // bytes. Verify that a minimal valid tx passes and the stats reflect it.
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      const result = mempool.add(tx);
      expect(result.added).toBe(true);

      const stats = mempool.getStats();
      expect(stats.size).toBe(1);
    });

    it("should track transaction by sender", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      mempool.add(tx);

      const senderTxs = mempool.getBySender(keyA.publicKey);
      expect(senderTxs).toHaveLength(1);
      expect(senderTxs[0].sender).toBe(keyA.publicKey);
    });
  });

  // ============ Transaction Removal ============

  describe("Transaction Removal", () => {
    it("should remove existing transaction", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      const { txHash } = mempool.add(tx);

      const removed = mempool.remove(txHash!);
      expect(removed).toBe(true);
      expect(mempool.size()).toBe(0);
      expect(mempool.get(txHash!)).toBeUndefined();
    });

    it("should return false for non-existent transaction", () => {
      const { mempool } = setup();
      const removed = mempool.remove("nonexistent-hash");
      expect(removed).toBe(false);
    });

    it("should clean up sender tracking on removal", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      const { txHash } = mempool.add(tx);

      mempool.remove(txHash!);

      const senderTxs = mempool.getBySender(keyA.publicKey);
      expect(senderTxs).toHaveLength(0);
    });
  });

  // ============ Transaction Retrieval ============

  describe("Transaction Retrieval", () => {
    it("should get transaction by hash", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      const { txHash } = mempool.add(tx);

      const retrieved = mempool.get(txHash!);
      expect(retrieved).toBeDefined();
      expect(retrieved!.sender).toBe(keyA.publicKey);
      expect(retrieved!.fee).toBe(10000n);
    });

    it("should return undefined for unknown hash", () => {
      const { mempool } = setup();
      const retrieved = mempool.get("unknown-hash");
      expect(retrieved).toBeUndefined();
    });

    it("should check existence with has()", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      const { txHash } = mempool.add(tx);

      expect(mempool.has(txHash!)).toBe(true);
      expect(mempool.has("nonexistent")).toBe(false);
    });

    it("should get all transactions for a sender", () => {
      const { mempool, keyA, keyB } = setup();
      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
      });
      const tx2 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 1n,
      });
      const tx3 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 0n,
      });

      mempool.add(tx1);
      mempool.add(tx2);
      mempool.add(tx3);

      const senderATxs = mempool.getBySender(keyA.publicKey);
      expect(senderATxs).toHaveLength(2);
      expect(senderATxs.every((t) => t.sender === keyA.publicKey)).toBe(true);

      const senderBTxs = mempool.getBySender(keyB.publicKey);
      expect(senderBTxs).toHaveLength(1);
    });

    it("should return empty array for unknown sender", () => {
      const { mempool } = setup();
      const txs = mempool.getBySender("unknown-sender-key");
      expect(txs).toEqual([]);
    });

    it("should get transactions by type", () => {
      const { mempool, keyA, keyB } = setup();
      const transferTx = createSignedTx(keyA.privateKey, keyA.publicKey, {
        type: "transfer",
        payload: { type: "transfer", to: "bob", amount: "50" },
      });
      const stakeTx = createSignedTx(keyB.privateKey, keyB.publicKey, {
        type: "stake",
        payload: { type: "stake", amount: "1000" },
      });

      mempool.add(transferTx);
      mempool.add(stakeTx);

      const transfers = mempool.getByType("transfer");
      expect(transfers).toHaveLength(1);
      expect(transfers[0].type).toBe("transfer");

      const stakes = mempool.getByType("stake");
      expect(stakes).toHaveLength(1);
      expect(stakes[0].type).toBe("stake");

      const delegates = mempool.getByType("delegate");
      expect(delegates).toHaveLength(0);
    });
  });

  // ============ Pool Capacity ============

  describe("Pool Capacity", () => {
    it("should reject when pool is full and fee too low", () => {
      const { keyA, keyB } = setup();
      const tinyPool = new Mempool({ maxSize: 2 });

      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
        fee: 5000n,
      });
      const tx2 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 1n,
        fee: 5000n,
      });

      tinyPool.add(tx1);
      tinyPool.add(tx2);
      expect(tinyPool.size()).toBe(2);

      // New tx with lower fee should be rejected
      const tx3 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 0n,
        fee: 1000n,
      });
      const result = tinyPool.add(tx3);

      expect(result.added).toBe(false);
      expect(result.reason).toMatch(/pool full/i);
    });

    it("should evict lowest feePerByte when pool is full and new tx has higher fee", () => {
      const { keyA, keyB } = setup();
      const tinyPool = new Mempool({ maxSize: 2 });

      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
        fee: 2000n,
      });
      const tx2 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 1n,
        fee: 3000n,
      });

      tinyPool.add(tx1);
      tinyPool.add(tx2);
      expect(tinyPool.size()).toBe(2);

      // New tx with much higher fee should evict the lowest feePerByte entry
      const tx3 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 0n,
        fee: 999999n,
      });
      const result = tinyPool.add(tx3);

      expect(result.added).toBe(true);
      expect(tinyPool.size()).toBe(2);
    });

    it("should enforce per-sender limit (maxPerSender)", () => {
      const { keyA } = setup();
      const pool = new Mempool({ maxPerSender: 2 });

      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
      });
      const tx2 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 1n,
      });
      const tx3 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 2n,
      });

      expect(pool.add(tx1).added).toBe(true);
      expect(pool.add(tx2).added).toBe(true);

      const result = pool.add(tx3);
      expect(result.added).toBe(false);
      expect(result.reason).toMatch(/too many transactions/i);
    });
  });

  // ============ Block Selection - getForBlock ============

  describe("Block Selection - getForBlock", () => {
    it("should return transactions sorted by feePerByte", () => {
      const { mempool, keyA, keyB } = setup();
      // Use different senders so round-robin picks highest-fee-lead sender first
      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
        fee: 5000n,
      });
      const tx2 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 0n,
        fee: 50000n,
      });

      mempool.add(tx1);
      mempool.add(tx2);

      const block = mempool.getForBlock(10, 1_000_000);
      expect(block).toHaveLength(2);
      // Higher feePerByte sender's tx should come first
      expect(block[0].sender).toBe(keyB.publicKey);
      expect(block[1].sender).toBe(keyA.publicKey);
    });

    it("should respect maxCount limit", () => {
      const { mempool, keyA, keyB } = setup();
      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
      });
      const tx2 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 0n,
      });

      mempool.add(tx1);
      mempool.add(tx2);

      const block = mempool.getForBlock(1, 1_000_000);
      expect(block).toHaveLength(1);
    });

    it("should respect maxSize limit", () => {
      const { mempool, keyA } = setup();
      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
      });
      mempool.add(tx1);

      // Use maxSize = 1 byte, which is smaller than any transaction
      const block = mempool.getForBlock(100, 1);
      expect(block).toHaveLength(0);
    });

    it("should enforce nonce continuity per sender", () => {
      const { mempool, keyA } = setup();
      // Add nonce 0 and nonce 2 (gap at nonce 1)
      const tx0 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
      });
      const tx2 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 2n,
      });

      mempool.add(tx0);
      mempool.add(tx2);

      // With senderBaseNonces, the expected starting nonce is 0.
      // Sorted entries: [nonce 0, nonce 2].
      // First entry nonce 0 matches expected 0, so contiguous = [0].
      // Next expected = 1, but next entry is nonce 2 => break.
      const baseNonces = new Map<string, bigint>();
      baseNonces.set(keyA.publicKey, 0n);

      const block = mempool.getForBlock(10, 1_000_000, baseNonces);
      expect(block).toHaveLength(1);
      expect(block[0].nonce).toBe(0n);
    });

    it("should skip nonce-gapped transactions when senderBaseNonces provided", () => {
      const { mempool, keyA } = setup();
      // State expects nonce 5, but pool has nonces 3, 5, 6.
      // Sorted: [3, 5, 6]. expectedNonce = 5, first entry = 3 => gap => break.
      // Result: 0 contiguous transactions.
      const txStale = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 3n,
      });
      const tx5 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 5n,
      });
      const tx6 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 6n,
      });

      mempool.add(txStale);
      mempool.add(tx5);
      mempool.add(tx6);

      const baseNonces = new Map<string, bigint>();
      baseNonces.set(keyA.publicKey, 5n);

      const block = mempool.getForBlock(10, 1_000_000, baseNonces);
      expect(block).toHaveLength(0);
    });

    it("should select contiguous nonce sequence starting from base", () => {
      const { mempool, keyA } = setup();
      // Pool has nonces 5, 6, 7. Base nonce = 5.
      // Sorted: [5, 6, 7]. All contiguous from 5.
      const tx5 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 5n,
      });
      const tx6 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 6n,
      });
      const tx7 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 7n,
      });

      mempool.add(tx5);
      mempool.add(tx6);
      mempool.add(tx7);

      const baseNonces = new Map<string, bigint>();
      baseNonces.set(keyA.publicKey, 5n);

      const block = mempool.getForBlock(10, 1_000_000, baseNonces);
      expect(block).toHaveLength(3);
      expect(block[0].nonce).toBe(5n);
      expect(block[1].nonce).toBe(6n);
      expect(block[2].nonce).toBe(7n);
    });

    it("should round-robin across senders", () => {
      const { mempool, keyA, keyB } = setup();
      // Two senders with 2 txs each, similar fees
      const txA0 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
        fee: 10000n,
      });
      const txA1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 1n,
        fee: 10000n,
      });
      const txB0 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 0n,
        fee: 10000n,
      });
      const txB1 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 1n,
        fee: 10000n,
      });

      mempool.add(txA0);
      mempool.add(txA1);
      mempool.add(txB0);
      mempool.add(txB1);

      const block = mempool.getForBlock(10, 1_000_000);
      expect(block).toHaveLength(4);

      // Round-robin: first pick one from each sender, then second from each.
      // Both senders have the same feePerByte so order depends on Map iteration,
      // but the interleaving pattern should be A,B,A,B or B,A,B,A.
      const senderOrder = block.map((tx) => tx.sender);
      // First two should be from different senders
      expect(senderOrder[0]).not.toBe(senderOrder[1]);
    });

    it("should handle empty pool", () => {
      const { mempool } = setup();
      const block = mempool.getForBlock(10, 1_000_000);
      expect(block).toEqual([]);
    });
  });

  // ============ Eviction ============

  describe("Eviction", () => {
    it("should evict expired transactions (using injectable clock)", () => {
      let now = 1000;
      const { keyA } = setup();
      const pool = new Mempool(
        { maxTxAge: 500, evictionInterval: 10 },
        () => now,
      );

      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      pool.add(tx);
      expect(pool.size()).toBe(1);

      // Advance clock past maxTxAge
      now = 2000;

      pool.start();
      return new Promise<void>((resolve) => {
        setTimeout(() => {
          expect(pool.size()).toBe(0);
          pool.stop();
          resolve();
        }, 50);
      });
    });

    it("should emit evicted event", () => {
      let now = 1000;
      const { keyA } = setup();
      const pool = new Mempool(
        { maxTxAge: 500, evictionInterval: 10 },
        () => now,
      );

      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      pool.add(tx);

      const evictedHandler = vi.fn();
      pool.on("evicted", evictedHandler);

      now = 2000;
      pool.start();

      return new Promise<void>((resolve) => {
        setTimeout(() => {
          expect(evictedHandler).toHaveBeenCalledWith(1);
          pool.stop();
          resolve();
        }, 50);
      });
    });

    it("should not evict fresh transactions", () => {
      let now = 1000;
      const { keyA } = setup();
      const pool = new Mempool(
        { maxTxAge: 5000, evictionInterval: 10 },
        () => now,
      );

      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      pool.add(tx);

      // Advance only slightly -- well within maxTxAge
      now = 1100;
      pool.start();

      return new Promise<void>((resolve) => {
        setTimeout(() => {
          expect(pool.size()).toBe(1);
          pool.stop();
          resolve();
        }, 50);
      });
    });
  });

  // ============ Stats ============

  describe("Stats", () => {
    it("should return correct pool statistics", () => {
      const { mempool, keyA, keyB } = setup();
      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
        fee: 5000n,
      });
      const tx2 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 1n,
        fee: 15000n,
      });
      const tx3 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 0n,
        fee: 10000n,
      });

      mempool.add(tx1);
      mempool.add(tx2);
      mempool.add(tx3);

      const stats = mempool.getStats();
      expect(stats.size).toBe(3);
      expect(stats.senders).toBe(2);
      expect(stats.totalFees).toBe(30000n);
      expect(stats.minFee).toBe(5000n);
      expect(stats.maxFee).toBe(15000n);
      expect(stats.avgFee).toBe(10000n); // 30000 / 3
    });

    it("should handle empty pool stats", () => {
      const { mempool } = setup();
      const stats = mempool.getStats();
      expect(stats.size).toBe(0);
      expect(stats.senders).toBe(0);
      expect(stats.totalFees).toBe(0n);
      expect(stats.minFee).toBe(0n);
      expect(stats.maxFee).toBe(0n);
      expect(stats.avgFee).toBe(0n);
    });

    it("should track totalFees, minFee, maxFee, avgFee", () => {
      const { mempool, keyA, keyB } = setup();
      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
        fee: 2000n,
      });
      const tx2 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 0n,
        fee: 8000n,
      });

      mempool.add(tx1);
      mempool.add(tx2);

      const stats = mempool.getStats();
      expect(stats.totalFees).toBe(10000n);
      expect(stats.minFee).toBe(2000n);
      expect(stats.maxFee).toBe(8000n);
      expect(stats.avgFee).toBe(5000n); // 10000 / 2
    });
  });

  // ============ Specialized Pools ============

  describe("Specialized Pools", () => {
    it("createClaimPool should have reduced maxSize and maxPerSender", () => {
      const { keyA } = setup();
      const pool = createClaimPool();

      // Fill up to the per-sender limit (10 for claim pool)
      for (let i = 0; i < 10; i++) {
        const tx = createSignedTx(keyA.privateKey, keyA.publicKey, {
          nonce: BigInt(i),
          type: "claim_submit",
          payload: {
            type: "claim_submit",
            claim: `claim-${i}`,
            domain: "test",
            stake: "100",
          },
        });
        const result = pool.add(tx);
        expect(result.added).toBe(true);
      }

      // 11th from same sender should fail
      const overflow = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 10n,
        type: "claim_submit",
        payload: {
          type: "claim_submit",
          claim: "overflow",
          domain: "test",
          stake: "100",
        },
      });
      const result = pool.add(overflow);
      expect(result.added).toBe(false);
      expect(result.reason).toMatch(/too many transactions/i);
    });

    it("createVerificationPool should allow zero fee", () => {
      const { keyA } = setup();
      const pool = createVerificationPool();

      const tx = createSignedTx(keyA.privateKey, keyA.publicKey, {
        fee: 0n,
        type: "verification_commit",
        payload: {
          type: "verification_commit",
          roundId: "round-1",
          commitment: "abc",
          vrfOutput: "def",
          vrfProof: "ghi",
        },
      });
      const result = pool.add(tx);

      expect(result.added).toBe(true);
    });
  });

  // ============ Events ============

  describe("Events", () => {
    it("should emit tx:added on successful add", () => {
      const { mempool, keyA } = setup();
      const handler = vi.fn();
      mempool.on("tx:added", handler);

      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      mempool.add(tx);

      expect(handler).toHaveBeenCalledTimes(1);
      const entry: MempoolEntry = handler.mock.calls[0][0];
      expect(entry.tx.sender).toBe(keyA.publicKey);
      expect(entry.txHash).toBeDefined();
      expect(entry.fee).toBe(10000n);
      expect(entry.size).toBeGreaterThan(0);
    });

    it("should emit tx:removed on removal", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      const { txHash } = mempool.add(tx);

      const handler = vi.fn();
      mempool.on("tx:removed", handler);

      mempool.remove(txHash!);

      expect(handler).toHaveBeenCalledTimes(1);
      const entry: MempoolEntry = handler.mock.calls[0][0];
      expect(entry.txHash).toBe(txHash);
    });

    it("should emit cleared on clear", () => {
      const { mempool, keyA } = setup();
      const handler = vi.fn();
      mempool.on("cleared", handler);

      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      mempool.add(tx);

      mempool.clear();

      expect(handler).toHaveBeenCalledTimes(1);
    });
  });

  // ============ Clear ============

  describe("Clear", () => {
    it("should remove all transactions", () => {
      const { mempool, keyA, keyB } = setup();
      const tx1 = createSignedTx(keyA.privateKey, keyA.publicKey, {
        nonce: 0n,
      });
      const tx2 = createSignedTx(keyB.privateKey, keyB.publicKey, {
        nonce: 0n,
      });

      mempool.add(tx1);
      mempool.add(tx2);
      expect(mempool.size()).toBe(2);

      mempool.clear();
      expect(mempool.size()).toBe(0);
    });

    it("should clean up sender tracking", () => {
      const { mempool, keyA } = setup();
      const tx = createSignedTx(keyA.privateKey, keyA.publicKey);
      mempool.add(tx);

      mempool.clear();

      const senderTxs = mempool.getBySender(keyA.publicKey);
      expect(senderTxs).toHaveLength(0);

      const stats = mempool.getStats();
      expect(stats.senders).toBe(0);
    });
  });

  // ============ Start / Stop ============

  describe("Start / Stop", () => {
    it("should start and stop without error", () => {
      const { mempool } = setup();
      expect(() => mempool.start()).not.toThrow();
      expect(() => mempool.stop()).not.toThrow();
    });

    it("should stop gracefully even when not started", () => {
      const { mempool } = setup();
      expect(() => mempool.stop()).not.toThrow();
    });
  });

  // ============ Default Config ============

  describe("Default Config", () => {
    it("should export DEFAULT_MEMPOOL_CONFIG with expected values", () => {
      expect(DEFAULT_MEMPOOL_CONFIG.maxSize).toBe(10000);
      expect(DEFAULT_MEMPOOL_CONFIG.maxPerSender).toBe(100);
      expect(DEFAULT_MEMPOOL_CONFIG.minFee).toBe(1000n);
      expect(DEFAULT_MEMPOOL_CONFIG.maxTxSize).toBe(100000);
      expect(DEFAULT_MEMPOOL_CONFIG.evictionInterval).toBe(60000);
      expect(DEFAULT_MEMPOOL_CONFIG.maxTxAge).toBe(3600000);
    });
  });
});
