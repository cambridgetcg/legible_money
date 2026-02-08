/**
 * Merkle Tree Tests
 *
 * Tests for InMemorySMT (Sparse Merkle Tree) and CompactMerkleTree,
 * including proof generation, verification, batch operations,
 * and low-level hashing primitives.
 */

import { describe, it, expect } from "vitest";
import {
  InMemorySMT,
  CompactMerkleTree,
  EMPTY_VALUE,
  TREE_DEPTH,
  EMPTY_HASHES,
  hashLeaf,
  hashNode,
  keyToBits,
} from "../../../core/crypto/merkle.js";
import { blake3Hash } from "../../../core/crypto/hashing.js";
import { bytesToHex } from "../../../core/crypto/keys.js";

// ============ Test Helpers ============

/** Create a deterministic key from a seed string. Returns 32 bytes. */
function makeKey(seed: string): Uint8Array {
  return blake3Hash(seed);
}

/** Create a deterministic value from a seed string. Returns 32 bytes. */
function makeValue(seed: string): Uint8Array {
  return blake3Hash(`value:${seed}`);
}

// Use a small depth (16) for InMemorySMT tests to keep them fast
const TEST_DEPTH = 16;

// ============ Constants ============

describe("EMPTY_VALUE", () => {
  it("is 32 zero bytes", () => {
    expect(EMPTY_VALUE).toBeInstanceOf(Uint8Array);
    expect(EMPTY_VALUE.length).toBe(32);
    for (let i = 0; i < 32; i++) {
      expect(EMPTY_VALUE[i]).toBe(0);
    }
  });
});

describe("TREE_DEPTH", () => {
  it("is 256 for 256-bit keys", () => {
    expect(TREE_DEPTH).toBe(256);
  });
});

describe("EMPTY_HASHES", () => {
  it("has TREE_DEPTH + 1 entries", () => {
    expect(EMPTY_HASHES.length).toBe(TREE_DEPTH + 1);
  });

  it("level 0 is the 32 zero bytes (EMPTY_VALUE)", () => {
    expect(EMPTY_HASHES[0]).toEqual(EMPTY_VALUE);
  });

  it("each level is the hash of two copies of the previous level", () => {
    // Verify the recursive construction for a few levels
    for (let i = 1; i <= 3; i++) {
      const expected = hashNode(EMPTY_HASHES[i - 1], EMPTY_HASHES[i - 1]);
      expect(EMPTY_HASHES[i]).toEqual(expected);
    }
  });

  it("all entries are 32 bytes", () => {
    for (let i = 0; i <= TREE_DEPTH; i++) {
      expect(EMPTY_HASHES[i].length).toBe(32);
    }
  });
});

// ============ hashLeaf / hashNode ============

describe("hashLeaf", () => {
  it("uses 0x00 prefix", () => {
    const key = makeKey("leaf-key");
    const value = makeValue("leaf-value");
    const result = hashLeaf(key, value);
    expect(result.length).toBe(32);

    // Manual verification
    const data = new Uint8Array(1 + key.length + value.length);
    data[0] = 0x00;
    data.set(key, 1);
    data.set(value, 1 + key.length);
    const expected = blake3Hash(data);
    expect(result).toEqual(expected);
  });

  it("different keys produce different leaf hashes", () => {
    const value = makeValue("same");
    const a = hashLeaf(makeKey("key-a"), value);
    const b = hashLeaf(makeKey("key-b"), value);
    expect(a).not.toEqual(b);
  });

  it("different values produce different leaf hashes", () => {
    const key = makeKey("same-key");
    const a = hashLeaf(key, makeValue("val-a"));
    const b = hashLeaf(key, makeValue("val-b"));
    expect(a).not.toEqual(b);
  });
});

describe("hashNode", () => {
  it("uses 0x01 prefix", () => {
    const left = blake3Hash("left-child");
    const right = blake3Hash("right-child");
    const result = hashNode(left, right);
    expect(result.length).toBe(32);

    // Manual verification
    const data = new Uint8Array(1 + left.length + right.length);
    data[0] = 0x01;
    data.set(left, 1);
    data.set(right, 1 + left.length);
    const expected = blake3Hash(data);
    expect(result).toEqual(expected);
  });

  it("is not commutative (order matters)", () => {
    const a = blake3Hash("a");
    const b = blake3Hash("b");
    expect(hashNode(a, b)).not.toEqual(hashNode(b, a));
  });

  it("leaf and node prefixes produce different hashes for similar data", () => {
    const data = blake3Hash("data");
    // hashLeaf(data, data) uses 0x00 prefix
    // hashNode(data, data) uses 0x01 prefix
    const leaf = hashLeaf(data, data);
    const node = hashNode(data, data);
    expect(leaf).not.toEqual(node);
  });
});

// ============ keyToBits ============

describe("keyToBits", () => {
  it("extracts correct bits for a known byte", () => {
    // 0b10110100 = 0xB4 = 180
    const key = new Uint8Array([0xb4]);
    const bits = keyToBits(key, 8);
    // MSB first: 1,0,1,1,0,1,0,0
    expect(bits).toEqual([true, false, true, true, false, true, false, false]);
  });

  it("handles multi-byte keys", () => {
    const key = new Uint8Array([0xff, 0x00]);
    const bits = keyToBits(key, 16);
    // First byte 0xFF = all 1s, second byte 0x00 = all 0s
    const expected = [
      true,
      true,
      true,
      true,
      true,
      true,
      true,
      true,
      false,
      false,
      false,
      false,
      false,
      false,
      false,
      false,
    ];
    expect(bits).toEqual(expected);
  });

  it("respects depth limit", () => {
    const key = new Uint8Array([0xff, 0xff]);
    const bits = keyToBits(key, 4);
    expect(bits.length).toBe(4);
    expect(bits).toEqual([true, true, true, true]);
  });

  it("returns empty array for depth 0", () => {
    const key = new Uint8Array([0xff]);
    expect(keyToBits(key, 0)).toEqual([]);
  });

  it("handles missing bytes gracefully (treats as 0)", () => {
    // Key is shorter than depth would require
    const key = new Uint8Array([0xff]);
    const bits = keyToBits(key, 16);
    // First 8 bits: all true, next 8 bits: all false (byte index 1 is undefined -> 0)
    expect(bits.slice(0, 8).every((b) => b === true)).toBe(true);
    expect(bits.slice(8, 16).every((b) => b === false)).toBe(true);
  });
});

// ============ InMemorySMT ============

describe("InMemorySMT", () => {
  it("empty tree has deterministic root", () => {
    const tree1 = new InMemorySMT(TEST_DEPTH);
    const tree2 = new InMemorySMT(TEST_DEPTH);
    expect(tree1.getRoot()).toBe(tree2.getRoot());
  });

  it("empty tree root matches EMPTY_HASHES at depth", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const expectedRoot = bytesToHex(EMPTY_HASHES[TEST_DEPTH]);
    expect(tree.getRoot()).toBe(expectedRoot);
  });

  it("set and get a value", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const key = makeKey("my-key");
    const value = makeValue("my-value");
    tree.set(key, value);
    const retrieved = tree.get(key);
    expect(retrieved).not.toBeNull();
    expect(retrieved).toEqual(value);
  });

  it("get returns null for non-existent key", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const key = makeKey("missing");
    expect(tree.get(key)).toBeNull();
  });

  it("set changes the root", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const rootBefore = tree.getRoot();
    tree.set(makeKey("k"), makeValue("v"));
    const rootAfter = tree.getRoot();
    expect(rootBefore).not.toBe(rootAfter);
  });

  it("different keys produce different roots", () => {
    const tree1 = new InMemorySMT(TEST_DEPTH);
    const tree2 = new InMemorySMT(TEST_DEPTH);
    tree1.set(makeKey("key-a"), makeValue("val"));
    tree2.set(makeKey("key-b"), makeValue("val"));
    expect(tree1.getRoot()).not.toBe(tree2.getRoot());
  });

  it("delete returns to empty root when only entry is removed", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const emptyRoot = tree.getRoot();
    const key = makeKey("ephemeral");
    tree.set(key, makeValue("temp"));
    expect(tree.getRoot()).not.toBe(emptyRoot);
    tree.delete(key);
    expect(tree.getRoot()).toBe(emptyRoot);
  });

  it("multiple set/get operations work correctly", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const entries = [
      { key: makeKey("k1"), value: makeValue("v1") },
      { key: makeKey("k2"), value: makeValue("v2") },
      { key: makeKey("k3"), value: makeValue("v3") },
    ];

    for (const { key, value } of entries) {
      tree.set(key, value);
    }

    for (const { key, value } of entries) {
      expect(tree.get(key)).toEqual(value);
    }
  });

  it("overwriting a key updates the value", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const key = makeKey("overwrite-me");
    tree.set(key, makeValue("original"));
    tree.set(key, makeValue("updated"));
    expect(tree.get(key)).toEqual(makeValue("updated"));
  });

  it("batchSet updates multiple entries", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const entries = [
      { key: makeKey("batch-1"), value: makeValue("b1") },
      { key: makeKey("batch-2"), value: makeValue("b2") },
      { key: makeKey("batch-3"), value: makeValue("b3") },
    ];

    const root = tree.batchSet(entries);
    expect(root).toMatch(/^[0-9a-f]+$/);

    for (const { key, value } of entries) {
      expect(tree.get(key)).toEqual(value);
    }
  });

  it("batchSet produces same root as sequential sets", () => {
    const entries = [
      { key: makeKey("seq-1"), value: makeValue("s1") },
      { key: makeKey("seq-2"), value: makeValue("s2") },
    ];

    const tree1 = new InMemorySMT(TEST_DEPTH);
    tree1.batchSet(entries);

    const tree2 = new InMemorySMT(TEST_DEPTH);
    for (const { key, value } of entries) {
      tree2.set(key, value);
    }

    expect(tree1.getRoot()).toBe(tree2.getRoot());
  });
});

// ============ InMemorySMT Proofs ============

describe("InMemorySMT proofs", () => {
  it("prove generates a valid proof for an existing key", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const key = makeKey("provable");
    const value = makeValue("provable-val");
    tree.set(key, value);

    const proofData = tree.prove(key);
    expect(proofData.key).toBe(bytesToHex(key));
    expect(proofData.value).toEqual(value);
    expect(proofData.proof.length).toBe(TEST_DEPTH);
    expect(proofData.root).toBe(tree.getRoot());
  });

  it("verify validates a correct membership proof", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const key = makeKey("member");
    tree.set(key, makeValue("member-val"));

    const proofData = tree.prove(key);
    expect(InMemorySMT.verify(proofData)).toBe(true);
  });

  it("verify rejects a tampered proof", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const key = makeKey("tamper-target");
    tree.set(key, makeValue("val"));

    const proofData = tree.prove(key);
    // Tamper with a sibling hash in the proof
    const tamperedProof = [...proofData.proof];
    tamperedProof[0] = "ff".repeat(32);

    expect(InMemorySMT.verify({ ...proofData, proof: tamperedProof })).toBe(
      false,
    );
  });

  it("non-membership proof for absent key verifies", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    tree.set(makeKey("present"), makeValue("yes"));

    const absentKey = makeKey("absent");
    const proofData = tree.prove(absentKey);
    expect(proofData.value).toBeNull();
    expect(InMemorySMT.verify(proofData)).toBe(true);
  });

  it("proof is invalid with wrong root", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const key = makeKey("root-check");
    tree.set(key, makeValue("val"));

    const proofData = tree.prove(key);
    const wrongRoot = "aa".repeat(32);

    expect(InMemorySMT.verify({ ...proofData, root: wrongRoot })).toBe(false);
  });

  it("proof is invalid with wrong value", () => {
    const tree = new InMemorySMT(TEST_DEPTH);
    const key = makeKey("val-check");
    tree.set(key, makeValue("correct"));

    const proofData = tree.prove(key);
    const wrongValue = makeValue("wrong");

    expect(InMemorySMT.verify({ ...proofData, value: wrongValue })).toBe(false);
  });
});

// ============ CompactMerkleTree ============

describe("CompactMerkleTree", () => {
  it("empty tree has a root (EMPTY_VALUE hex)", () => {
    const tree = new CompactMerkleTree();
    const root = tree.getRoot();
    expect(root).toBe(bytesToHex(EMPTY_VALUE));
  });

  it("append changes the root", () => {
    const tree = new CompactMerkleTree();
    const emptyRoot = tree.getRoot();
    tree.append(new Uint8Array([1, 2, 3]));
    expect(tree.getRoot()).not.toBe(emptyRoot);
  });

  it("append returns sequential indices", () => {
    const tree = new CompactMerkleTree();
    expect(tree.append(new Uint8Array([1]))).toBe(0);
    expect(tree.append(new Uint8Array([2]))).toBe(1);
    expect(tree.append(new Uint8Array([3]))).toBe(2);
  });

  it("different data produces different roots", () => {
    const tree1 = new CompactMerkleTree();
    const tree2 = new CompactMerkleTree();
    tree1.append(new Uint8Array([1, 2, 3]));
    tree2.append(new Uint8Array([4, 5, 6]));
    expect(tree1.getRoot()).not.toBe(tree2.getRoot());
  });

  it("prove and verify round-trip for single leaf", () => {
    const tree = new CompactMerkleTree();
    const data = new Uint8Array([10, 20, 30]);
    tree.append(data);
    const proof = tree.prove(0);
    const root = tree.getRoot();
    expect(CompactMerkleTree.verify(data, 0, proof, root)).toBe(true);
  });

  it("prove and verify round-trip for multiple leaves", () => {
    const tree = new CompactMerkleTree();
    const items = [
      new Uint8Array([1]),
      new Uint8Array([2]),
      new Uint8Array([3]),
      new Uint8Array([4]),
    ];
    for (const item of items) {
      tree.append(item);
    }

    const root = tree.getRoot();
    for (let i = 0; i < items.length; i++) {
      const proof = tree.prove(i);
      expect(CompactMerkleTree.verify(items[i], i, proof, root)).toBe(true);
    }
  });

  it("verify rejects wrong data", () => {
    const tree = new CompactMerkleTree();
    const data = new Uint8Array([1, 2, 3]);
    tree.append(data);
    const proof = tree.prove(0);
    const root = tree.getRoot();
    const wrongData = new Uint8Array([9, 9, 9]);
    expect(CompactMerkleTree.verify(wrongData, 0, proof, root)).toBe(false);
  });

  it("verify rejects wrong index", () => {
    const tree = new CompactMerkleTree();
    tree.append(new Uint8Array([1]));
    tree.append(new Uint8Array([2]));
    const proof = tree.prove(0);
    const root = tree.getRoot();
    // Verify leaf 0's data at index 1 should fail
    expect(CompactMerkleTree.verify(new Uint8Array([1]), 1, proof, root)).toBe(
      false,
    );
  });

  it("prove throws for out-of-bounds index", () => {
    const tree = new CompactMerkleTree();
    tree.append(new Uint8Array([1]));
    expect(() => tree.prove(5)).toThrow(/Index out of bounds/);
  });

  it("non-power-of-two leaf count works correctly", () => {
    const tree = new CompactMerkleTree();
    // 3 leaves (not a power of 2)
    const items = [
      new Uint8Array([10]),
      new Uint8Array([20]),
      new Uint8Array([30]),
    ];
    for (const item of items) {
      tree.append(item);
    }

    const root = tree.getRoot();
    for (let i = 0; i < items.length; i++) {
      const proof = tree.prove(i);
      expect(CompactMerkleTree.verify(items[i], i, proof, root)).toBe(true);
    }
  });
});
