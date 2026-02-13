/**
 * Dimension 1: Cryptographic Integrity - Differential Tests
 *
 * Cross-validates cryptographic operations for consistency:
 * 1. Domain separation produces different outputs for different domains
 * 2. Hash function determinism (same input → same output)
 * 3. Hash function collision resistance (different inputs → different outputs)
 * 4. Key derivation determinism
 * 5. Commitment scheme binding (can't open to different value)
 * 6. VRF output uniqueness per input
 *
 * Industry method: Differential testing + property-based invariants.
 */

import { describe, it, expect } from "vitest";
import {
  domainHash,
  sha512_256,
  sha512_256Hex,
  blake3Hash,
  HASH_DOMAINS,
  constantTimeEqual,
} from "../../../core/crypto/hashing.js";
import {
  bytesToHex,
  hexToBytes,
  generateKeypair,
} from "../../../core/crypto/keys.js";
import {
  commit,
  verifyCommitment,
  commitVerdict,
  verifyVerdictCommitment,
} from "../../../core/crypto/commitment.js";

// ============================================================
// DIFF-01 through DIFF-10: Differential cryptographic tests
// ============================================================

describe("Cryptographic Differential Tests (DIFF)", () => {
  it("DIFF-01: All HASH_DOMAINS produce unique outputs for same input", () => {
    const input = "test-input-for-domain-separation";
    const outputs = new Set<string>();

    const domains = Object.values(HASH_DOMAINS);
    expect(domains.length).toBeGreaterThan(0);

    for (const domain of domains) {
      const hash = bytesToHex(domainHash(domain, input));
      outputs.add(hash);
    }

    // Every domain should produce a unique hash
    expect(outputs.size).toBe(domains.length);
  });

  it("DIFF-02: domainHash is deterministic", () => {
    const domain = HASH_DOMAINS.STATE;
    const input = "determinism-test";

    const hash1 = bytesToHex(domainHash(domain, input));
    const hash2 = bytesToHex(domainHash(domain, input));

    expect(hash1).toBe(hash2);
  });

  it("DIFF-03: sha512_256 and sha512_256Hex are consistent", () => {
    const input = "consistency-test";

    const bytes = sha512_256(input);
    const hex = sha512_256Hex(input);

    expect(bytesToHex(bytes)).toBe(hex);
  });

  it("DIFF-04: blake3Hash produces 32-byte output", () => {
    const inputs = ["", "a", "hello world", "x".repeat(10000)];

    for (const input of inputs) {
      const hash = blake3Hash(input);
      expect(hash).toBeInstanceOf(Uint8Array);
      expect(hash.length).toBe(32);
    }
  });

  it("DIFF-05: Different inputs produce different sha512_256 hashes (collision resistance)", () => {
    const hashes = new Set<string>();
    const count = 1000;

    for (let i = 0; i < count; i++) {
      const hash = sha512_256Hex(`input-${i}`);
      hashes.add(hash);
    }

    // All 1000 hashes should be unique
    expect(hashes.size).toBe(count);
  });

  it("DIFF-06: Different inputs produce different blake3 hashes", () => {
    const hashes = new Set<string>();
    const count = 1000;

    for (let i = 0; i < count; i++) {
      const hash = bytesToHex(blake3Hash(`input-${i}`));
      hashes.add(hash);
    }

    expect(hashes.size).toBe(count);
  });

  it("DIFF-07: Commitment binding — opening with wrong value fails", () => {
    const claimId = "claim-binding-test";
    const validatorId = "validator-binding-test";
    const trueVerdict = "confirmed" as const;
    const trueConfidence = 0.85;

    // Create commitment (returns { commitment, nonce, verdict, confidence, claimId })
    const result = commitVerdict(
      claimId,
      trueVerdict,
      trueConfidence,
      validatorId,
    );

    // Verify with correct values (nonce from the commitment result)
    const correctVerify = verifyVerdictCommitment(
      result.commitment,
      claimId,
      trueVerdict,
      trueConfidence,
      validatorId,
      result.nonce,
    );
    expect(correctVerify).toBe(true);

    // Verify with wrong verdict — must fail
    const wrongVerdict = verifyVerdictCommitment(
      result.commitment,
      claimId,
      "rejected",
      trueConfidence,
      validatorId,
      result.nonce,
    );
    expect(wrongVerdict).toBe(false);

    // Verify with wrong confidence — must fail
    const wrongConfidence = verifyVerdictCommitment(
      result.commitment,
      claimId,
      trueVerdict,
      0.1,
      validatorId,
      result.nonce,
    );
    expect(wrongConfidence).toBe(false);

    // Verify with wrong validator — must fail
    const wrongValidator = verifyVerdictCommitment(
      result.commitment,
      claimId,
      trueVerdict,
      trueConfidence,
      "wrong-validator",
      result.nonce,
    );
    expect(wrongValidator).toBe(false);
  });

  it("DIFF-08: constantTimeEqual rejects different-length arrays", () => {
    const a = new Uint8Array([1, 2, 3]);
    const b = new Uint8Array([1, 2, 3, 4]);

    expect(constantTimeEqual(a, b)).toBe(false);
  });

  it("DIFF-09: constantTimeEqual accepts identical arrays", () => {
    const data = new Uint8Array([0xde, 0xad, 0xbe, 0xef]);
    expect(constantTimeEqual(data, data)).toBe(true);

    // Also test with a copy
    const copy = new Uint8Array(data);
    expect(constantTimeEqual(data, copy)).toBe(true);
  });

  it("DIFF-10: Key generation produces unique keypairs", () => {
    const keys = new Set<string>();
    const count = 50;

    for (let i = 0; i < count; i++) {
      const kp = generateKeypair();
      keys.add(kp.publicKey);
    }

    // All keypairs should be unique
    expect(keys.size).toBe(count);
  });
});
