/**
 * Dimension 1: Cryptographic Integrity - Property-Based Tests
 *
 * Uses fast-check for randomized property-based testing of:
 * - VRF properties (uniqueness, unpredictability, soundness, fairness)
 * - Ed25519 domain-separated signing
 * - Sparse Merkle Tree algebraic properties
 */

import { describe, it, expect, test } from "vitest";
import fc from "fast-check";
import {
  generateKeypair,
  generateKeypairFromSeed,
  sign,
  verify,
  bytesToHex,
  hexToBytes,
} from "../../../core/crypto/keys.js";
import {
  signTransaction,
  signBlock,
  signDomainSeparated,
  verifyDomainSeparated,
} from "../../../core/crypto/signatures.js";
import { HASH_DOMAINS } from "../../../core/crypto/hashing.js";
import {
  vrfGenerate,
  vrfVerify,
  isValidatorSelected,
  generateBlockSeed,
} from "../../../core/crypto/vrf.js";
import { InMemorySMT } from "../../../core/crypto/merkle.js";

// ============================================================
// 1. VRF Properties (8 tests)
// ============================================================

describe("VRF Properties", () => {
  it("VRF-1: Uniqueness - same (key, input) always produces same output", () => {
    const { privateKey, publicKey } = generateKeypair();
    const input = "deterministic VRF input";

    const result1 = vrfGenerate(input, privateKey);
    const result2 = vrfGenerate(input, privateKey);

    expect(result1.output).toBe(result2.output);
    expect(result1.proof).toBe(result2.proof);

    // Both proofs should verify
    const v1 = vrfVerify(input, publicKey, result1.proof);
    const v2 = vrfVerify(input, publicKey, result2.proof);
    expect(v1.valid).toBe(true);
    expect(v2.valid).toBe(true);
    expect(v1.output).toBe(v2.output);
  });

  it("VRF-2: Unpredictability - VRF outputs from different inputs are uniformly distributed", () => {
    const { privateKey } = generateKeypair();
    const NUM_SAMPLES = 1000;
    const NUM_BUCKETS = 16;
    const buckets = new Array(NUM_BUCKETS).fill(0);

    for (let i = 0; i < NUM_SAMPLES; i++) {
      const input = `vrf-uniformity-test-input-${i}`;
      const result = vrfGenerate(input, privateKey);

      // Use first byte of the output to determine bucket
      const outputBytes = hexToBytes(result.output);
      const bucket = outputBytes[0] % NUM_BUCKETS;
      buckets[bucket]++;
    }

    // Chi-squared test for uniformity
    // Expected count per bucket = NUM_SAMPLES / NUM_BUCKETS
    const expected = NUM_SAMPLES / NUM_BUCKETS;
    let chiSquared = 0;
    for (const count of buckets) {
      chiSquared += (count - expected) ** 2 / expected;
    }

    // Critical value for chi-squared with 15 degrees of freedom at p=0.01 is 30.58
    // We use a generous threshold to avoid flaky tests
    expect(chiSquared).toBeLessThan(35);

    // Also verify no bucket is dramatically over/under represented
    for (const count of buckets) {
      expect(count).toBeGreaterThan(expected * 0.3); // At least 30% of expected
      expect(count).toBeLessThan(expected * 2.0); // At most 200% of expected
    }
  });

  it("VRF-3: Soundness - VRF proof from wrong key always fails verification", () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 100 }), (input) => {
        const keyA = generateKeypair();
        const keyB = generateKeypair();

        // Generate proof with keyA
        const result = vrfGenerate(input, keyA.privateKey);

        // Verify with keyB's public key -- should fail
        const verification = vrfVerify(input, keyB.publicKey, result.proof);
        return verification.valid === false;
      }),
      { numRuns: 30 },
    );
  });

  it("VRF-4: VRF output determinism - 100 proofs for same input all identical", () => {
    const { privateKey, publicKey } = generateKeypair();
    const input = "repeated-vrf-determinism-check";

    const firstResult = vrfGenerate(input, privateKey);
    const firstOutput = firstResult.output;

    for (let i = 0; i < 100; i++) {
      const result = vrfGenerate(input, privateKey);
      expect(result.output).toBe(firstOutput);

      const v = vrfVerify(input, publicKey, result.proof);
      expect(v.valid).toBe(true);
      expect(v.output).toBe(firstOutput);
    }
  });

  it("VRF-5: Seed grinding resistance - generateBlockSeed includes all entropy sources", () => {
    const blockHash = "aabbccdd";
    const blockNumber = 100n;
    const epoch = 5n;
    const randomness = "accumulated-randomness-hex";

    const seed1 = generateBlockSeed(blockHash, blockNumber, epoch, randomness);

    // Changing any single parameter changes the seed
    const seed2 = generateBlockSeed(
      "different",
      blockNumber,
      epoch,
      randomness,
    );
    const seed3 = generateBlockSeed(blockHash, 101n, epoch, randomness);
    const seed4 = generateBlockSeed(blockHash, blockNumber, 6n, randomness);
    const seed5 = generateBlockSeed(
      blockHash,
      blockNumber,
      epoch,
      "different-rand",
    );

    const seedHex1 = bytesToHex(seed1);
    expect(seedHex1).not.toBe(bytesToHex(seed2));
    expect(seedHex1).not.toBe(bytesToHex(seed3));
    expect(seedHex1).not.toBe(bytesToHex(seed4));
    expect(seedHex1).not.toBe(bytesToHex(seed5));

    // Without accumulated randomness, still works (uses "genesis" default)
    const seedNoRand = generateBlockSeed(blockHash, blockNumber, epoch);
    expect(seedNoRand).toBeDefined();
    expect(seedNoRand.length).toBe(32);

    // But result differs from when randomness is provided
    expect(bytesToHex(seedNoRand)).not.toBe(seedHex1);
  });

  it("VRF-6: Selection fairness - frequency proportional to stake within 15%", () => {
    const { privateKey, publicKey } = generateKeypair();
    const NUM_RUNS = 1000;
    const TOTAL_STAKE = 1000000n;
    const MY_STAKE = 200000n; // 20% of total
    const TARGET_VALIDATORS = 10;

    let selectedCount = 0;

    for (let i = 0; i < NUM_RUNS; i++) {
      const input = `selection-fairness-round-${i}`;
      const result = vrfGenerate(input, privateKey);
      const { selected } = isValidatorSelected(
        result.output,
        MY_STAKE,
        TOTAL_STAKE,
        TARGET_VALIDATORS,
      );
      if (selected) selectedCount++;
    }

    // Expected selection rate: (stake / totalStake) * targetValidators = 0.2 * 10 = 2.0
    // But since selection is P(selected) = min(1, stake * target / total),
    // with 20% stake and 10 targets, expected probability is min(1, 2.0) = 1.0
    // Actually, the formula is: outputNum * totalStake < stake * targetValidators * maxNum
    // So P(selected) = (stake * targetValidators) / totalStake = 2.0 => always selected
    // Let's use a smaller stake for a meaningful test
    let selectedCount2 = 0;
    const SMALL_STAKE = 50000n; // 5% of total
    // Expected P = 0.05 * 10 = 0.5

    for (let i = 0; i < NUM_RUNS; i++) {
      const input = `selection-fairness-small-${i}`;
      const result = vrfGenerate(input, privateKey);
      const { selected } = isValidatorSelected(
        result.output,
        SMALL_STAKE,
        TOTAL_STAKE,
        TARGET_VALIDATORS,
      );
      if (selected) selectedCount2++;
    }

    // Expected rate: 50%, tolerance: 15% means [35%, 65%]
    const rate = selectedCount2 / NUM_RUNS;
    expect(rate).toBeGreaterThan(0.35);
    expect(rate).toBeLessThan(0.65);
  });

  it("VRF-7: VRF proof size is always 192 hex characters (96 bytes)", () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 200 }), (input) => {
        const { privateKey } = generateKeypair();
        const result = vrfGenerate(input, privateKey);

        // Proof is hex-encoded 96 bytes = 192 hex characters
        return result.proof.length === 192 && /^[0-9a-f]+$/.test(result.proof);
      }),
      { numRuns: 50 },
    );
  });

  it("VRF-8: Invalid proof bytes reject cleanly (no throw/crash)", () => {
    const { publicKey } = generateKeypair();
    const input = "test-invalid-proof";

    // Completely wrong proof (wrong length)
    const shortProof = "aabbccdd";
    const v1 = vrfVerify(input, publicKey, shortProof);
    expect(v1.valid).toBe(false);
    expect(v1.output).toBeNull();

    // Correct length but random bytes (96 bytes = 192 hex)
    const randomProof = "a".repeat(192);
    const v2 = vrfVerify(input, publicKey, randomProof);
    expect(v2.valid).toBe(false);
    expect(v2.output).toBeNull();

    // Empty proof
    const v3 = vrfVerify(input, publicKey, "");
    expect(v3.valid).toBe(false);
    expect(v3.output).toBeNull();

    // Proof with all zeros (96 bytes)
    const zeroProof = "0".repeat(192);
    const v4 = vrfVerify(input, publicKey, zeroProof);
    expect(v4.valid).toBe(false);
    expect(v4.output).toBeNull();
  });
});

// ============================================================
// 2. Ed25519 Domain-Separated Signing (4 tests)
// ============================================================

describe("Ed25519 Domain-Separated Signing", () => {
  it("EDDS-1: signTransaction() signature differs from signBlock() for same data", () => {
    const { privateKey } = generateKeypair();
    const data = { type: "test", value: 42, nonce: 1 };

    const txSig = signTransaction(data, privateKey);
    const blockSig = signBlock(data, privateKey);

    // Different domains must produce different signatures
    expect(txSig).not.toBe(blockSig);
  });

  it("EDDS-2: verifyDomainSeparated() rejects signature made with wrong domain", () => {
    fc.assert(
      fc.property(fc.string({ minLength: 1, maxLength: 500 }), (message) => {
        const { privateKey, publicKey } = generateKeypair();

        // Sign with TRANSACTION domain
        const sig = signDomainSeparated(
          HASH_DOMAINS.TRANSACTION,
          message,
          privateKey,
        );

        // Verify with BLOCK domain -- must fail
        const valid = verifyDomainSeparated(
          HASH_DOMAINS.BLOCK,
          message,
          sig,
          publicKey,
        );
        return valid === false;
      }),
      { numRuns: 50 },
    );
  });

  it("EDDS-3: Domain-separated signatures are deterministic", () => {
    const { privateKey } = generateKeypair();
    const message = "deterministic domain-separated signature test";

    const sig1 = signDomainSeparated(HASH_DOMAINS.FACT, message, privateKey);
    const sig2 = signDomainSeparated(HASH_DOMAINS.FACT, message, privateKey);

    expect(sig1).toBe(sig2);
  });

  it("EDDS-4: Cross-domain signature reuse fails verification", () => {
    const { privateKey, publicKey } = generateKeypair();
    const message = "cross-domain reuse test";

    // Sign under each domain, verify under every other domain
    const domains = [
      HASH_DOMAINS.BLOCK,
      HASH_DOMAINS.TRANSACTION,
      HASH_DOMAINS.FACT,
      HASH_DOMAINS.VRF,
      HASH_DOMAINS.COMMITMENT,
      HASH_DOMAINS.SIGNATURE,
    ];

    for (const signDomain of domains) {
      const sig = signDomainSeparated(signDomain, message, privateKey);

      // Should verify under the correct domain
      expect(verifyDomainSeparated(signDomain, message, sig, publicKey)).toBe(
        true,
      );

      // Should fail under every other domain
      for (const verifyDomain of domains) {
        if (verifyDomain === signDomain) continue;
        expect(
          verifyDomainSeparated(verifyDomain, message, sig, publicKey),
        ).toBe(false);
      }
    }
  });
});

// ============================================================
// 3. Sparse Merkle Tree Properties (4 tests)
// ============================================================

describe("Sparse Merkle Tree Properties", () => {
  const TEST_DEPTH = 16;

  function makeKey(id: number, depth: number): Uint8Array {
    const keyLen = Math.ceil(depth / 8);
    const key = new Uint8Array(keyLen);
    const bytesForId = Math.min(4, keyLen);
    for (let i = 0; i < bytesForId; i++) {
      key[bytesForId - 1 - i] = (id >> (8 * i)) & 0xff;
    }
    return key;
  }

  function makeValue(content: string): Uint8Array {
    return new TextEncoder().encode(content);
  }

  it("SMT-1: For N random inserts, root is deterministic regardless of insertion order", () => {
    fc.assert(
      fc.property(
        // Generate a list of unique key-value pairs
        fc.array(
          fc.record({
            keyId: fc.integer({ min: 0, max: 65535 }),
            value: fc.string({ minLength: 1, maxLength: 50 }),
          }),
          { minLength: 2, maxLength: 20 },
        ),
        (entries) => {
          // Deduplicate by keyId
          const uniqueMap = new Map<number, string>();
          for (const e of entries) {
            uniqueMap.set(e.keyId, e.value);
          }
          const uniqueEntries = Array.from(uniqueMap.entries()).map(
            ([keyId, value]) => ({ keyId, value }),
          );

          if (uniqueEntries.length < 2) return true; // Need at least 2 to test ordering

          // Insert in original order
          const tree1 = new InMemorySMT(TEST_DEPTH);
          for (const e of uniqueEntries) {
            tree1.set(makeKey(e.keyId, TEST_DEPTH), makeValue(e.value));
          }

          // Insert in reverse order
          const tree2 = new InMemorySMT(TEST_DEPTH);
          const reversed = [...uniqueEntries].reverse();
          for (const e of reversed) {
            tree2.set(makeKey(e.keyId, TEST_DEPTH), makeValue(e.value));
          }

          return tree1.getRoot() === tree2.getRoot();
        },
      ),
      { numRuns: 30 },
    );
  });

  it("SMT-2: Deleting all keys returns tree to empty root state", () => {
    fc.assert(
      fc.property(
        fc.array(fc.integer({ min: 0, max: 65535 }), {
          minLength: 1,
          maxLength: 15,
        }),
        (keyIds) => {
          const tree = new InMemorySMT(TEST_DEPTH);
          const emptyRoot = tree.getRoot();

          // Deduplicate
          const uniqueKeys = [...new Set(keyIds)];

          // Insert all keys
          for (const id of uniqueKeys) {
            tree.set(makeKey(id, TEST_DEPTH), makeValue(`val-${id}`));
          }

          // Root should have changed (assuming at least one key)
          if (uniqueKeys.length > 0) {
            if (tree.getRoot() === emptyRoot) return false; // Unexpected
          }

          // Delete all keys
          for (const id of uniqueKeys) {
            tree.delete(makeKey(id, TEST_DEPTH));
          }

          // Root should be back to empty
          return tree.getRoot() === emptyRoot;
        },
      ),
      { numRuns: 30 },
    );
  });

  it("SMT-3: For any key, exactly one of inclusion or exclusion proof succeeds", () => {
    fc.assert(
      fc.property(
        fc.array(fc.integer({ min: 0, max: 65535 }), {
          minLength: 0,
          maxLength: 20,
        }),
        fc.integer({ min: 0, max: 65535 }),
        (insertedIds, queryId) => {
          const tree = new InMemorySMT(TEST_DEPTH);
          const insertedSet = new Set(insertedIds);

          for (const id of insertedSet) {
            tree.set(makeKey(id, TEST_DEPTH), makeValue(`val-${id}`));
          }

          const queryKey = makeKey(queryId, TEST_DEPTH);
          const proofData = tree.prove(queryKey);

          // Proof must verify
          const verifies = InMemorySMT.verify(proofData);
          if (!verifies) return false;

          if (insertedSet.has(queryId)) {
            // Key exists: inclusion proof (value is not null)
            return proofData.value !== null;
          } else {
            // Key does not exist: exclusion proof (value is null)
            return proofData.value === null;
          }
        },
      ),
      { numRuns: 50 },
    );
  });

  it("SMT-4: Batch insert of K items produces same root as K individual inserts", () => {
    fc.assert(
      fc.property(
        fc.array(
          fc.record({
            keyId: fc.integer({ min: 0, max: 65535 }),
            value: fc.string({ minLength: 1, maxLength: 30 }),
          }),
          { minLength: 1, maxLength: 15 },
        ),
        (entries) => {
          // Deduplicate
          const uniqueMap = new Map<number, string>();
          for (const e of entries) {
            uniqueMap.set(e.keyId, e.value);
          }

          // Individual inserts
          const tree1 = new InMemorySMT(TEST_DEPTH);
          for (const [keyId, value] of uniqueMap) {
            tree1.set(makeKey(keyId, TEST_DEPTH), makeValue(value));
          }

          // Batch insert
          const tree2 = new InMemorySMT(TEST_DEPTH);
          const batchEntries = Array.from(uniqueMap).map(([keyId, value]) => ({
            key: makeKey(keyId, TEST_DEPTH),
            value: makeValue(value),
          }));
          tree2.batchSet(batchEntries);

          return tree1.getRoot() === tree2.getRoot();
        },
      ),
      { numRuns: 30 },
    );
  });
});
