/**
 * VRF (Verifiable Random Function) Tests
 *
 * Tests for VRF generation, verification, validator selection,
 * block seed generation, and randomness accumulation.
 */

import { describe, it, expect } from "vitest";
import {
  vrfGenerate,
  vrfVerify,
  isValidatorSelected,
  generateBlockSeed,
  accumulateRandomness,
} from "../../../core/crypto/vrf.js";
import { generateKeypair } from "../../../core/crypto/keys.js";
import { bytesToHex } from "../../../core/crypto/keys.js";

// ============ Test Helpers ============

function makeKeypair() {
  return generateKeypair();
}

// ============ VRF Generate ============

describe("vrfGenerate", () => {
  it("produces a non-empty output hex string", () => {
    const { privateKey } = makeKeypair();
    const { output } = vrfGenerate("test-input", privateKey);
    expect(output).toMatch(/^[0-9a-f]+$/);
    expect(output.length).toBeGreaterThan(0);
  });

  it("produces a non-empty proof hex string", () => {
    const { privateKey } = makeKeypair();
    const { proof } = vrfGenerate("test-input", privateKey);
    expect(proof).toMatch(/^[0-9a-f]+$/);
    // Proof is 96 bytes = 192 hex chars
    expect(proof.length).toBe(192);
  });

  it("output is 32 bytes (64 hex chars) from BLAKE3", () => {
    const { privateKey } = makeKeypair();
    const { output } = vrfGenerate("input", privateKey);
    expect(output.length).toBe(64);
  });

  it("is deterministic: same key + input produces same output", () => {
    const { privateKey } = makeKeypair();
    const a = vrfGenerate("deterministic-test", privateKey);
    const b = vrfGenerate("deterministic-test", privateKey);
    expect(a.output).toBe(b.output);
    expect(a.proof).toBe(b.proof);
  });

  it("different inputs produce different outputs", () => {
    const { privateKey } = makeKeypair();
    const a = vrfGenerate("input-a", privateKey);
    const b = vrfGenerate("input-b", privateKey);
    expect(a.output).not.toBe(b.output);
  });

  it("different keys produce different outputs for same input", () => {
    const kp1 = makeKeypair();
    const kp2 = makeKeypair();
    const a = vrfGenerate("same-input", kp1.privateKey);
    const b = vrfGenerate("same-input", kp2.privateKey);
    expect(a.output).not.toBe(b.output);
  });

  it("accepts Uint8Array input", () => {
    const { privateKey } = makeKeypair();
    const data = new Uint8Array([0xde, 0xad, 0xbe, 0xef]);
    const result = vrfGenerate(data, privateKey);
    expect(result.output.length).toBe(64);
    expect(result.proof.length).toBe(192);
  });
});

// ============ VRF Verify ============

describe("vrfVerify", () => {
  it("valid proof verifies successfully", () => {
    const { privateKey, publicKey } = makeKeypair();
    const { proof } = vrfGenerate("verify-me", privateKey);
    const result = vrfVerify("verify-me", publicKey, proof);
    expect(result.valid).toBe(true);
    expect(result.output).not.toBeNull();
  });

  it("output from verify matches output from generate", () => {
    const { privateKey, publicKey } = makeKeypair();
    const generated = vrfGenerate("match-test", privateKey);
    const verified = vrfVerify("match-test", publicKey, generated.proof);
    expect(verified.valid).toBe(true);
    expect(verified.output).toBe(generated.output);
  });

  it("wrong input fails verification", () => {
    const { privateKey, publicKey } = makeKeypair();
    const { proof } = vrfGenerate("correct-input", privateKey);
    const result = vrfVerify("wrong-input", publicKey, proof);
    expect(result.valid).toBe(false);
  });

  it("wrong public key fails verification", () => {
    const kp1 = makeKeypair();
    const kp2 = makeKeypair();
    const { proof } = vrfGenerate("key-mismatch", kp1.privateKey);
    const result = vrfVerify("key-mismatch", kp2.publicKey, proof);
    expect(result.valid).toBe(false);
  });

  it("tampered proof fails verification", () => {
    const { privateKey, publicKey } = makeKeypair();
    const { proof } = vrfGenerate("tamper-test", privateKey);
    // Flip a byte in the middle of the proof
    const proofChars = proof.split("");
    const midIndex = Math.floor(proofChars.length / 2);
    proofChars[midIndex] = proofChars[midIndex] === "0" ? "f" : "0";
    const tampered = proofChars.join("");
    const result = vrfVerify("tamper-test", publicKey, tampered);
    expect(result.valid).toBe(false);
  });

  it("invalid proof length returns invalid", () => {
    const { publicKey } = makeKeypair();
    // Proof should be 96 bytes (192 hex chars); provide only 64 hex chars
    const shortProof = "ab".repeat(32);
    const result = vrfVerify("test", publicKey, shortProof);
    expect(result.valid).toBe(false);
    expect(result.output).toBeNull();
  });

  it("completely random proof fails", () => {
    const { publicKey } = makeKeypair();
    // 96 bytes of random-looking data
    const fakeProof = "ff".repeat(96);
    const result = vrfVerify("test", publicKey, fakeProof);
    expect(result.valid).toBe(false);
  });

  it("works with Uint8Array input for verification", () => {
    const { privateKey, publicKey } = makeKeypair();
    const data = new Uint8Array([1, 2, 3, 4, 5]);
    const generated = vrfGenerate(data, privateKey);
    const verified = vrfVerify(data, publicKey, generated.proof);
    expect(verified.valid).toBe(true);
    expect(verified.output).toBe(generated.output);
  });
});

// ============ Validator Selection ============

describe("isValidatorSelected", () => {
  it("returns selected and priority fields", () => {
    const { privateKey } = makeKeypair();
    const { output } = vrfGenerate("selection-test", privateKey);
    const result = isValidatorSelected(output, 1000n, 10000n, 10);
    expect(result).toHaveProperty("selected");
    expect(result).toHaveProperty("priority");
    expect(typeof result.selected).toBe("boolean");
    expect(typeof result.priority).toBe("bigint");
  });

  it("zero stake is never selected", () => {
    const { privateKey } = makeKeypair();
    const { output } = vrfGenerate("zero-stake", privateKey);
    const result = isValidatorSelected(output, 0n, 10000n, 10);
    expect(result.selected).toBe(false);
  });

  it("high stake relative to total is more likely selected", () => {
    // Run a statistical test: with stake = totalStake, targetValidators = 10,
    // the probability should be very high (essentially 1).
    const { privateKey } = makeKeypair();
    let selectedCount = 0;
    for (let i = 0; i < 20; i++) {
      const { output } = vrfGenerate(`stat-test-${i}`, privateKey);
      const result = isValidatorSelected(output, 10000n, 10000n, 10);
      if (result.selected) selectedCount++;
    }
    // With stake == totalStake and targetValidators = 10, should always be selected
    expect(selectedCount).toBe(20);
  });

  it("low stake relative to total is less likely selected", () => {
    // stake/totalStake * targetValidators = 1/1000000 * 1 = very low probability
    const { privateKey } = makeKeypair();
    let selectedCount = 0;
    for (let i = 0; i < 50; i++) {
      const { output } = vrfGenerate(`low-stake-${i}`, privateKey);
      const result = isValidatorSelected(output, 1n, 1000000n, 1);
      if (result.selected) selectedCount++;
    }
    // With such low probability, most should not be selected
    expect(selectedCount).toBeLessThan(10);
  });

  it("priority is a bigint derived from VRF output", () => {
    const { privateKey } = makeKeypair();
    const { output } = vrfGenerate("priority-test", privateKey);
    const result = isValidatorSelected(output, 100n, 1000n, 5);
    expect(result.priority).toBeGreaterThanOrEqual(0n);
  });
});

// ============ Block Seed Generation ============

describe("generateBlockSeed", () => {
  it("returns a Uint8Array", () => {
    const seed = generateBlockSeed("abc123", 1n, 0n);
    expect(seed).toBeInstanceOf(Uint8Array);
    expect(seed.length).toBe(64); // SHA-512 output via domainHash
  });

  it("is deterministic for same inputs", () => {
    const a = generateBlockSeed("blockhash", 42n, 1n);
    const b = generateBlockSeed("blockhash", 42n, 1n);
    expect(a).toEqual(b);
  });

  it("different block numbers produce different seeds", () => {
    const a = generateBlockSeed("same-hash", 1n, 0n);
    const b = generateBlockSeed("same-hash", 2n, 0n);
    expect(a).not.toEqual(b);
  });

  it("different previous block hashes produce different seeds", () => {
    const a = generateBlockSeed("hash-a", 1n, 0n);
    const b = generateBlockSeed("hash-b", 1n, 0n);
    expect(a).not.toEqual(b);
  });

  it("different epochs produce different seeds", () => {
    const a = generateBlockSeed("hash", 1n, 0n);
    const b = generateBlockSeed("hash", 1n, 1n);
    expect(a).not.toEqual(b);
  });

  it("includes accumulated randomness when provided", () => {
    const withoutAR = generateBlockSeed("hash", 1n, 0n);
    const withAR = generateBlockSeed("hash", 1n, 0n, "some-randomness");
    expect(withoutAR).not.toEqual(withAR);
  });

  it("defaults accumulated randomness to 'genesis'", () => {
    // Calling without vs with "genesis" should produce same result
    const a = generateBlockSeed("hash", 1n, 0n);
    const b = generateBlockSeed("hash", 1n, 0n, "genesis");
    expect(a).toEqual(b);
  });
});

// ============ Randomness Accumulation ============

describe("accumulateRandomness", () => {
  it("returns a hex string", () => {
    const result = accumulateRandomness("initial", "vrf-output", 1n);
    expect(result).toMatch(/^[0-9a-f]+$/);
  });

  it("is deterministic", () => {
    const a = accumulateRandomness("acc", "out", 5n);
    const b = accumulateRandomness("acc", "out", 5n);
    expect(a).toBe(b);
  });

  it("different accumulator inputs produce different results", () => {
    const a = accumulateRandomness("acc-a", "out", 1n);
    const b = accumulateRandomness("acc-b", "out", 1n);
    expect(a).not.toBe(b);
  });

  it("different VRF outputs produce different results", () => {
    const a = accumulateRandomness("acc", "out-a", 1n);
    const b = accumulateRandomness("acc", "out-b", 1n);
    expect(a).not.toBe(b);
  });

  it("different block numbers produce different results", () => {
    const a = accumulateRandomness("acc", "out", 1n);
    const b = accumulateRandomness("acc", "out", 2n);
    expect(a).not.toBe(b);
  });
});
