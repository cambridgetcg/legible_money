/**
 * Property-Based Tests for Cryptographic Signatures
 *
 * These tests verify the core invariants of Ed25519 signature operations
 * using fast-check for property-based testing.
 *
 * Invariants tested:
 * - INV-SIGN-1: Signature Validity (valid signatures verify)
 * - INV-SIGN-2: Signature Uniqueness (different messages, different sigs)
 * - INV-SIGN-3: Tamper Detection (modified messages fail)
 * - INV-SIGN-4: Wrong Key Rejection (wrong public key fails)
 */

import { describe, test, expect } from "vitest";
import fc from "fast-check";
import {
  generateKeypair,
  sign,
  verify,
  hexToBytes,
} from "../../../core/crypto/keys.js";

// Helper: Compare Uint8Arrays
function arraysEqual(a: Uint8Array, b: Uint8Array): boolean {
  if (a.length !== b.length) return false;
  for (let i = 0; i < a.length; i++) {
    if (a[i] !== b[i]) return false;
  }
  return true;
}

describe("Signature Invariants", () => {
  // ============ INV-SIGN-1: Signature Validity ============

  test("INV-SIGN-1: Valid signatures always verify (random messages)", () => {
    fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 10000 }),
        (message) => {
          const { privateKey, publicKey } = generateKeypair();
          const signature = sign(message, privateKey);
          return verify(signature, message, publicKey) === true;
        },
      ),
      { numRuns: 100 },
    );
  });

  test("INV-SIGN-1: Valid signatures verify (empty message)", () => {
    const { privateKey, publicKey } = generateKeypair();
    const message = new Uint8Array(0);
    const signature = sign(message, privateKey);
    expect(verify(signature, message, publicKey)).toBe(true);
  });

  test("INV-SIGN-1: Valid signatures verify (single byte)", () => {
    fc.assert(
      fc.property(fc.integer({ min: 0, max: 255 }), (byte) => {
        const { privateKey, publicKey } = generateKeypair();
        const message = new Uint8Array([byte]);
        const signature = sign(message, privateKey);
        return verify(signature, message, publicKey) === true;
      }),
      { numRuns: 50 },
    );
  });

  // ============ INV-SIGN-2: Signature Uniqueness ============

  test("INV-SIGN-2: Different messages produce different signatures", () => {
    fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 500 }),
        fc.uint8Array({ minLength: 1, maxLength: 500 }),
        (msg1, msg2) => {
          // Skip if messages are equal
          if (arraysEqual(msg1, msg2)) return true;

          const { privateKey } = generateKeypair();
          const sig1 = sign(msg1, privateKey);
          const sig2 = sign(msg2, privateKey);

          return sig1 !== sig2;
        },
      ),
      { numRuns: 100 },
    );
  });

  test("INV-SIGN-2: Single-bit difference produces different signature", () => {
    const { privateKey } = generateKeypair();
    const msg1 = new Uint8Array([0x00]);
    const msg2 = new Uint8Array([0x01]);

    const sig1 = sign(msg1, privateKey);
    const sig2 = sign(msg2, privateKey);

    expect(sig1).not.toBe(sig2);
  });

  // ============ INV-SIGN-3: Tamper Detection ============

  test("INV-SIGN-3: Tampered messages fail verification", () => {
    fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 1000 }),
        fc.nat(),
        (message, flipSeed) => {
          const { privateKey, publicKey } = generateKeypair();
          const signature = sign(message, privateKey);

          // Create tampered message
          const tampered = new Uint8Array(message);
          const idx = flipSeed % message.length;
          tampered[idx] = tampered[idx] ^ 0xff;

          // Tampered message should fail
          return verify(signature, tampered, publicKey) === false;
        },
      ),
      { numRuns: 100 },
    );
  });

  test("INV-SIGN-3: Appended byte fails verification", () => {
    const { privateKey, publicKey } = generateKeypair();
    const message = new Uint8Array([1, 2, 3]);
    const signature = sign(message, privateKey);

    const extended = new Uint8Array([1, 2, 3, 4]);
    expect(verify(signature, extended, publicKey)).toBe(false);
  });

  test("INV-SIGN-3: Truncated message fails verification", () => {
    const { privateKey, publicKey } = generateKeypair();
    const message = new Uint8Array([1, 2, 3, 4]);
    const signature = sign(message, privateKey);

    const truncated = new Uint8Array([1, 2, 3]);
    expect(verify(signature, truncated, publicKey)).toBe(false);
  });

  // ============ INV-SIGN-4: Wrong Key Rejection ============

  test("INV-SIGN-4: Wrong public key fails verification", () => {
    fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 1000 }),
        (message) => {
          const keypair1 = generateKeypair();
          const keypair2 = generateKeypair();

          const signature = sign(message, keypair1.privateKey);
          return verify(signature, message, keypair2.publicKey) === false;
        },
      ),
      { numRuns: 100 },
    );
  });

  test("INV-SIGN-4: Swapped keys fail verification", () => {
    const keypair1 = generateKeypair();
    const keypair2 = generateKeypair();
    const message = new Uint8Array([1, 2, 3]);

    // Sign with keypair1, verify with keypair2's public key
    const signature = sign(message, keypair1.privateKey);
    expect(verify(signature, message, keypair2.publicKey)).toBe(false);

    // And vice versa
    const signature2 = sign(message, keypair2.privateKey);
    expect(verify(signature2, message, keypair1.publicKey)).toBe(false);
  });

  // ============ Edge Cases ============

  test("Edge: All-zero message signs correctly", () => {
    const { privateKey, publicKey } = generateKeypair();
    const message = new Uint8Array(100).fill(0);
    const signature = sign(message, privateKey);
    expect(verify(signature, message, publicKey)).toBe(true);
  });

  test("Edge: All-ones message signs correctly", () => {
    const { privateKey, publicKey } = generateKeypair();
    const message = new Uint8Array(100).fill(0xff);
    const signature = sign(message, privateKey);
    expect(verify(signature, message, publicKey)).toBe(true);
  });

  test("Edge: Repeated signing produces same signature (deterministic)", () => {
    const { privateKey, publicKey } = generateKeypair();
    const message = new Uint8Array([1, 2, 3, 4, 5]);

    const sig1 = sign(message, privateKey);
    const sig2 = sign(message, privateKey);

    // Ed25519 is deterministic - same key + message = same signature
    expect(sig1).toBe(sig2);
    expect(verify(sig1, message, publicKey)).toBe(true);
  });

  // ============ Signature Format ============

  test("Signature is 128-character hex string (64 bytes)", () => {
    fc.assert(
      fc.property(
        fc.uint8Array({ minLength: 1, maxLength: 100 }),
        (message) => {
          const { privateKey } = generateKeypair();
          const signature = sign(message, privateKey);

          // Hex string: 64 bytes = 128 hex characters
          return signature.length === 128 && /^[0-9a-f]+$/.test(signature);
        },
      ),
      { numRuns: 50 },
    );
  });

  test("Public key is 64-character hex string (32 bytes)", () => {
    fc.assert(
      fc.property(fc.constant(null), () => {
        const { publicKey } = generateKeypair();
        return publicKey.length === 64 && /^[0-9a-f]+$/.test(publicKey);
      }),
      { numRuns: 50 },
    );
  });

  test("Private key is 64-character hex string (32 bytes)", () => {
    fc.assert(
      fc.property(fc.constant(null), () => {
        const { privateKey } = generateKeypair();
        return privateKey.length === 64 && /^[0-9a-f]+$/.test(privateKey);
      }),
      { numRuns: 50 },
    );
  });
});
