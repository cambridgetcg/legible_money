/**
 * Commitment Scheme Tests
 *
 * Tests for basic commitments, numeric (Pedersen-like) commitments,
 * verdict commitments, batch commitments, and timed commitments.
 */

import { describe, it, expect, vi } from "vitest";
import {
  commit,
  verifyCommitment,
  commitNumeric,
  verifyNumericCommitment,
  commitVerdict,
  verifyVerdictCommitment,
  commitBatch,
  verifyBatchItem,
  commitTimed,
  canReveal,
} from "../../../core/crypto/commitment.js";

// ============ Basic Commitment ============

describe("commit", () => {
  it("returns a commitment, nonce, and value", () => {
    const result = commit("secret");
    expect(result).toHaveProperty("commitment");
    expect(result).toHaveProperty("nonce");
    expect(result).toHaveProperty("value");
  });

  it("commitment is a non-empty hex string", () => {
    const { commitment } = commit("data");
    expect(commitment).toMatch(/^[0-9a-f]+$/);
    expect(commitment.length).toBeGreaterThan(0);
  });

  it("nonce is 64 hex chars (32 bytes)", () => {
    const { nonce } = commit("data");
    expect(nonce).toMatch(/^[0-9a-f]{64}$/);
  });

  it("preserves the original string value", () => {
    const { value } = commit("my secret value");
    expect(value).toBe("my secret value");
  });

  it("accepts Uint8Array value and returns hex value", () => {
    const bytes = new Uint8Array([0xaa, 0xbb, 0xcc]);
    const result = commit(bytes);
    expect(result.value).toBe("aabbcc");
  });

  it("same value produces different commitments (random nonce)", () => {
    const a = commit("same-value");
    const b = commit("same-value");
    expect(a.commitment).not.toBe(b.commitment);
    expect(a.nonce).not.toBe(b.nonce);
  });

  it("different values produce different commitments", () => {
    const a = commit("value-a");
    const b = commit("value-b");
    expect(a.commitment).not.toBe(b.commitment);
  });
});

// ============ Verify Commitment ============

describe("verifyCommitment", () => {
  it("returns true for correct value and nonce", () => {
    const { commitment, nonce, value } = commit("verifiable");
    expect(verifyCommitment(commitment, value, nonce)).toBe(true);
  });

  it("returns false for wrong value", () => {
    const { commitment, nonce } = commit("correct");
    expect(verifyCommitment(commitment, "wrong", nonce)).toBe(false);
  });

  it("returns false for wrong nonce", () => {
    const { commitment, value } = commit("test");
    const fakeNonce = "00".repeat(32);
    expect(verifyCommitment(commitment, value, fakeNonce)).toBe(false);
  });

  it("works with Uint8Array value for verification", () => {
    const bytes = new Uint8Array([1, 2, 3, 4]);
    const result = commit(bytes);
    // To verify with Uint8Array, pass the original bytes
    expect(verifyCommitment(result.commitment, bytes, result.nonce)).toBe(true);
  });

  it("fails when both value and nonce are wrong", () => {
    const { commitment } = commit("original");
    expect(verifyCommitment(commitment, "tampered", "ff".repeat(32))).toBe(
      false,
    );
  });
});

// ============ Numeric Commitment ============

describe("commitNumeric", () => {
  it("returns a commitment and blinding factor", () => {
    const result = commitNumeric(1000n);
    expect(result).toHaveProperty("commitment");
    expect(result).toHaveProperty("blinding");
  });

  it("commitment is a non-empty hex string", () => {
    const { commitment } = commitNumeric(42n);
    expect(commitment).toMatch(/^[0-9a-f]+$/);
    expect(commitment.length).toBeGreaterThan(0);
  });

  it("blinding is 64 hex chars (32 bytes)", () => {
    const { blinding } = commitNumeric(100n);
    expect(blinding).toMatch(/^[0-9a-f]{64}$/);
  });

  it("same value produces different commitments (random blinding)", () => {
    const a = commitNumeric(500n);
    const b = commitNumeric(500n);
    expect(a.commitment).not.toBe(b.commitment);
  });

  it("works with zero", () => {
    const result = commitNumeric(0n);
    expect(result.commitment).toMatch(/^[0-9a-f]+$/);
  });

  it("works with very large values", () => {
    const bigValue = BigInt("999999999999999999999999999999");
    const result = commitNumeric(bigValue);
    expect(result.commitment).toMatch(/^[0-9a-f]+$/);
  });
});

describe("verifyNumericCommitment", () => {
  it("returns true for correct value and blinding", () => {
    const value = 12345n;
    const { commitment, blinding } = commitNumeric(value);
    expect(verifyNumericCommitment(commitment, value, blinding)).toBe(true);
  });

  it("returns false for wrong value", () => {
    const { commitment, blinding } = commitNumeric(100n);
    expect(verifyNumericCommitment(commitment, 999n, blinding)).toBe(false);
  });

  it("returns false for wrong blinding", () => {
    const { commitment } = commitNumeric(100n);
    expect(verifyNumericCommitment(commitment, 100n, "aa".repeat(32))).toBe(
      false,
    );
  });

  it("verifies zero correctly", () => {
    const { commitment, blinding } = commitNumeric(0n);
    expect(verifyNumericCommitment(commitment, 0n, blinding)).toBe(true);
    expect(verifyNumericCommitment(commitment, 1n, blinding)).toBe(false);
  });
});

// ============ Verdict Commitment ============

describe("commitVerdict", () => {
  it("returns claimId, commitment, nonce, verdict, and confidence", () => {
    const result = commitVerdict("claim-001", "confirmed", 0.95, "validator-1");
    expect(result.claimId).toBe("claim-001");
    expect(result.verdict).toBe("confirmed");
    expect(result.confidence).toBe(0.95);
    expect(result.commitment).toMatch(/^[0-9a-f]+$/);
    expect(result.nonce).toMatch(/^[0-9a-f]{64}$/);
  });

  it("preserves all verdict types", () => {
    for (const verdict of ["confirmed", "rejected", "uncertain"] as const) {
      const result = commitVerdict("c", verdict, 0.5, "v");
      expect(result.verdict).toBe(verdict);
    }
  });

  it("same inputs produce different commitments (random nonce)", () => {
    const a = commitVerdict("claim", "confirmed", 0.9, "validator");
    const b = commitVerdict("claim", "confirmed", 0.9, "validator");
    expect(a.commitment).not.toBe(b.commitment);
  });
});

describe("verifyVerdictCommitment", () => {
  it("returns true for correct parameters", () => {
    const result = commitVerdict("claim-x", "confirmed", 0.85, "val-a");
    expect(
      verifyVerdictCommitment(
        result.commitment,
        "claim-x",
        "confirmed",
        0.85,
        "val-a",
        result.nonce,
      ),
    ).toBe(true);
  });

  it("returns false for wrong verdict", () => {
    const result = commitVerdict("claim-x", "confirmed", 0.85, "val-a");
    expect(
      verifyVerdictCommitment(
        result.commitment,
        "claim-x",
        "rejected",
        0.85,
        "val-a",
        result.nonce,
      ),
    ).toBe(false);
  });

  it("returns false for wrong confidence", () => {
    const result = commitVerdict("claim-x", "confirmed", 0.85, "val-a");
    expect(
      verifyVerdictCommitment(
        result.commitment,
        "claim-x",
        "confirmed",
        0.5,
        "val-a",
        result.nonce,
      ),
    ).toBe(false);
  });

  it("returns false for wrong claimId", () => {
    const result = commitVerdict("claim-x", "confirmed", 0.85, "val-a");
    expect(
      verifyVerdictCommitment(
        result.commitment,
        "claim-y",
        "confirmed",
        0.85,
        "val-a",
        result.nonce,
      ),
    ).toBe(false);
  });

  it("returns false for wrong validatorId", () => {
    const result = commitVerdict("claim-x", "confirmed", 0.85, "val-a");
    expect(
      verifyVerdictCommitment(
        result.commitment,
        "claim-x",
        "confirmed",
        0.85,
        "val-b",
        result.nonce,
      ),
    ).toBe(false);
  });

  it("returns false for wrong nonce", () => {
    const result = commitVerdict("claim-x", "confirmed", 0.85, "val-a");
    expect(
      verifyVerdictCommitment(
        result.commitment,
        "claim-x",
        "confirmed",
        0.85,
        "val-a",
        "00".repeat(32),
      ),
    ).toBe(false);
  });
});

// ============ Batch Commitment ============

describe("commitBatch", () => {
  it("returns commitment, nonces array, and values array", () => {
    const result = commitBatch(["alpha", "beta", "gamma"]);
    expect(result).toHaveProperty("commitment");
    expect(result).toHaveProperty("nonces");
    expect(result).toHaveProperty("values");
  });

  it("commitment is a hex string (Merkle root)", () => {
    const { commitment } = commitBatch(["a", "b"]);
    expect(commitment).toMatch(/^[0-9a-f]+$/);
  });

  it("nonces length matches values length", () => {
    const values = ["one", "two", "three", "four"];
    const result = commitBatch(values);
    expect(result.nonces.length).toBe(values.length);
    expect(result.values.length).toBe(values.length);
  });

  it("preserves values", () => {
    const values = ["hello", "world"];
    const result = commitBatch(values);
    expect(result.values).toEqual(values);
  });

  it("each nonce is 64 hex chars (32 bytes)", () => {
    const result = commitBatch(["a", "b", "c"]);
    for (const nonce of result.nonces) {
      expect(nonce).toMatch(/^[0-9a-f]{64}$/);
    }
  });

  it("different batches produce different commitments", () => {
    const a = commitBatch(["a", "b"]);
    const b = commitBatch(["c", "d"]);
    expect(a.commitment).not.toBe(b.commitment);
  });

  it("same values produce different commitments (random nonces)", () => {
    const a = commitBatch(["x", "y"]);
    const b = commitBatch(["x", "y"]);
    expect(a.commitment).not.toBe(b.commitment);
  });

  it("handles single value", () => {
    const result = commitBatch(["only"]);
    expect(result.commitment).toMatch(/^[0-9a-f]+$/);
    expect(result.nonces.length).toBe(1);
  });
});

// ============ Timed Commitment ============

describe("commitTimed", () => {
  it("stores the revealAfter timestamp", () => {
    const deadline = Date.now() + 60000;
    const result = commitTimed("secret", deadline);
    expect(result.revealAfter).toBe(deadline);
  });

  it("stores the value", () => {
    const result = commitTimed("my-secret", Date.now());
    expect(result.value).toBe("my-secret");
  });

  it("creates a valid commitment and nonce", () => {
    const result = commitTimed("timed-value", Date.now());
    expect(result.commitment).toMatch(/^[0-9a-f]+$/);
    expect(result.nonce).toMatch(/^[0-9a-f]{64}$/);
  });

  it("commitment is verifiable via verifyCommitment", () => {
    const result = commitTimed("verify-me", Date.now());
    expect(
      verifyCommitment(result.commitment, result.value, result.nonce),
    ).toBe(true);
  });
});

describe("canReveal", () => {
  it("returns true when current time is past the deadline", () => {
    const pastDeadline = Date.now() - 10000; // 10 seconds ago
    const tc = commitTimed("past", pastDeadline);
    expect(canReveal(tc)).toBe(true);
  });

  it("returns false when current time is before the deadline", () => {
    const futureDeadline = Date.now() + 60000; // 60 seconds from now
    const tc = commitTimed("future", futureDeadline);
    expect(canReveal(tc)).toBe(false);
  });

  it("returns true when current time equals the deadline", () => {
    // Use vi.spyOn to fix Date.now for this test
    const fixedTime = 1700000000000;
    vi.spyOn(Date, "now").mockReturnValue(fixedTime);
    const tc = commitTimed("exact", fixedTime);
    expect(canReveal(tc)).toBe(true);
    vi.restoreAllMocks();
  });
});
