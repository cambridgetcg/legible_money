/**
 * Memory Encryption Tests (C-2 + C-3 Remediation)
 *
 * Tests for:
 * - AES-256-GCM cipher module (encrypt/decrypt/key derivation)
 * - MemoryManager encrypted storage (encryptForStorage/decryptFromStorage)
 * - Integrity hash verification (C-3: tamper detection)
 */

import { describe, it, expect } from "vitest";
import {
  deriveMemoryEncryptionKey,
  encrypt,
  decrypt,
  encryptJSON,
  decryptJSON,
} from "../../../core/crypto/cipher.js";
import {
  MemoryManager,
  type EncryptedMemoryEnvelope,
} from "../../../core/agents/home/memory.js";
import type { DID } from "../../../core/agents/identity/did.js";
import { randomBytes } from "crypto";

// ============ Test Helpers ============

const TEST_DID: DID = "did:lgm:test-encrypt-agent-abcdef01" as DID;
const TEST_DID_2: DID = "did:lgm:test-encrypt-peer-fedcba01" as DID;

function makeTestKey(): Uint8Array {
  return randomBytes(32);
}

function populateMemory(manager: MemoryManager): void {
  manager.setCurrentTime(Date.now());

  // Add contributions
  manager.recordContribution({
    factId: "fact:test:001",
    role: "verifier",
    contribution: "Verified claim about Bitcoin halvings",
    outcome: "accepted",
    confidence: 0.95,
    rewardEarned: 1000n * 10n ** 18n,
    block: 100n,
  });

  // Add beliefs
  manager.setBelief("fact:test:001", 0.95, "verified");
  manager.setBelief("fact:test:002", 0.7, "trusted_agent", TEST_DID_2);

  // Add relationships
  manager.trust(TEST_DID_2, 0.8, "Reliable verifier");

  // Add verification history
  manager.recordVerification(
    {
      claimId: "claim:test:001",
      role: "verifier",
      vote: 0.9,
      consensusResult: 0.85,
      wasCorrect: true,
      rewardOrPenalty: 500n * 10n ** 18n,
    },
    100n,
  );

  // Add dispute history
  manager.recordDispute(
    {
      disputeId: "dispute:test:001",
      role: "defender",
      outcome: "won",
      stake: 100n * 10n ** 18n,
      netResult: 50n * 10n ** 18n,
      lessons: ["Evidence quality matters"],
    },
    200n,
  );

  // Add notes
  manager.addNote(new Uint8Array([1, 2, 3, 4, 5]), ["test", "important"]);
}

// ============ Cipher Module Tests ============

describe.sequential("AES-256-GCM Cipher", () => {
  it("should derive a 32-byte encryption key", () => {
    const seed = randomBytes(32);
    const key = deriveMemoryEncryptionKey(seed);
    expect(key.length).toBe(32);
  });

  it("should derive different keys from different seeds", () => {
    const key1 = deriveMemoryEncryptionKey(randomBytes(32));
    const key2 = deriveMemoryEncryptionKey(randomBytes(32));
    expect(Buffer.from(key1).equals(Buffer.from(key2))).toBe(false);
  });

  it("should derive same key from same seed (deterministic)", () => {
    const seed = randomBytes(32);
    const key1 = deriveMemoryEncryptionKey(seed);
    const key2 = deriveMemoryEncryptionKey(seed);
    expect(Buffer.from(key1).equals(Buffer.from(key2))).toBe(true);
  });

  it("should encrypt and decrypt binary data", () => {
    const key = makeTestKey();
    const plaintext = new Uint8Array([72, 101, 108, 108, 111]); // "Hello"
    const ciphertext = encrypt(plaintext, key);

    // Ciphertext should be larger (iv + authTag + encrypted)
    expect(ciphertext.length).toBeGreaterThan(plaintext.length);

    const decrypted = decrypt(ciphertext, key);
    expect(Buffer.from(decrypted).equals(Buffer.from(plaintext))).toBe(true);
  });

  it("should produce different ciphertexts for same plaintext (random IV)", () => {
    const key = makeTestKey();
    const plaintext = new Uint8Array([1, 2, 3]);
    const ct1 = encrypt(plaintext, key);
    const ct2 = encrypt(plaintext, key);
    expect(Buffer.from(ct1).equals(Buffer.from(ct2))).toBe(false);
  });

  it("should reject wrong decryption key", () => {
    const key1 = makeTestKey();
    const key2 = makeTestKey();
    const ciphertext = encrypt(new Uint8Array([1, 2, 3]), key1);

    expect(() => decrypt(ciphertext, key2)).toThrow();
  });

  it("should detect tampered ciphertext", () => {
    const key = makeTestKey();
    const ciphertext = encrypt(new Uint8Array([1, 2, 3]), key);

    // Flip a byte in the encrypted data region
    const tampered = new Uint8Array(ciphertext);
    tampered[tampered.length - 1] ^= 0xff;

    expect(() => decrypt(tampered, key)).toThrow();
  });

  it("should reject too-short ciphertext", () => {
    const key = makeTestKey();
    expect(() => decrypt(new Uint8Array(10), key)).toThrow(/too short/);
  });

  it("should reject wrong key length", () => {
    expect(() => encrypt(new Uint8Array([1]), new Uint8Array(16))).toThrow(
      /must be 32 bytes/,
    );
    expect(() => decrypt(new Uint8Array(30), new Uint8Array(16))).toThrow(
      /must be 32 bytes/,
    );
  });
});

describe.sequential("JSON Encrypt/Decrypt", () => {
  it("should round-trip JSON objects", () => {
    const key = makeTestKey();
    const data = { name: "test", count: 42, nested: { ok: true } };
    const encrypted = encryptJSON(data, key);
    const decrypted = decryptJSON<typeof data>(encrypted, key);
    expect(decrypted).toEqual(data);
  });

  it("should handle BigInt values via __bigint: prefix", () => {
    const key = makeTestKey();
    const data = { amount: 1000000000000000000n, label: "one LGM" };
    const encrypted = encryptJSON(data, key);
    const decrypted = decryptJSON<typeof data>(encrypted, key);
    expect(decrypted.amount).toBe(1000000000000000000n);
    expect(decrypted.label).toBe("one LGM");
  });

  it("should handle arrays and Maps serialized as entries", () => {
    const key = makeTestKey();
    const data = {
      entries: [
        ["key1", { value: 100n }],
        ["key2", { value: 200n }],
      ],
    };
    const encrypted = encryptJSON(data, key);
    const decrypted = decryptJSON<typeof data>(encrypted, key);
    expect(decrypted.entries).toHaveLength(2);
    expect(decrypted.entries[0][1].value).toBe(100n);
  });
});

// ============ Memory Encryption Tests ============

describe.sequential("MemoryManager Encrypted Storage (C-2)", () => {
  it("should encrypt and restore memory round-trip", () => {
    const key = makeTestKey();
    const manager = new MemoryManager();
    populateMemory(manager);

    // Encrypt
    const envelope = manager.encryptForStorage(key);
    expect(envelope.version).toBe(1);
    expect(envelope.encryptedSensitive.length).toBeGreaterThan(0);
    expect(envelope.integrityHash).toBeTruthy();

    // Decrypt and restore
    const restored = MemoryManager.decryptFromStorage(envelope, key);
    const original = manager.getMemory();
    const restoredMem = restored.getMemory();

    // Verify sensitive fields restored correctly
    expect(restoredMem.beliefs.size).toBe(original.beliefs.size);
    expect(restoredMem.beliefs.get("fact:test:001")?.confidence).toBe(0.95);
    expect(restoredMem.beliefs.get("fact:test:002")?.source).toBe(
      "trusted_agent",
    );

    expect(restoredMem.relationships.trusted.size).toBe(1);
    expect(restoredMem.relationships.trusted.get(TEST_DID_2)?.trustLevel).toBe(
      0.8,
    );

    expect(restoredMem.contributions).toHaveLength(1);
    expect(restoredMem.contributions[0].rewardEarned).toBe(1000n * 10n ** 18n);

    expect(restoredMem.verificationHistory).toHaveLength(1);
    expect(restoredMem.disputeHistory).toHaveLength(1);
  });

  it("should preserve plaintext metadata in envelope", () => {
    const key = makeTestKey();
    const manager = new MemoryManager();
    populateMemory(manager);

    const envelope = manager.encryptForStorage(key);

    // Stats, milestones, sessionLog, notes should be readable without decryption
    expect(envelope.stats.totalContributions).toBe(1);
    // Total = contribution reward (1000 LGM) + verification reward (500 LGM) + dispute net (50 LGM)
    expect(envelope.stats.totalEarnings).toBe(1550n * 10n ** 18n);
    expect(envelope.notes).toHaveLength(1);
    expect(envelope.milestones.length).toBeGreaterThan(0);
  });

  it("should reject wrong decryption key", () => {
    const key1 = makeTestKey();
    const key2 = makeTestKey();
    const manager = new MemoryManager();
    populateMemory(manager);

    const envelope = manager.encryptForStorage(key1);

    expect(() => MemoryManager.decryptFromStorage(envelope, key2)).toThrow();
  });

  it("should detect tampered encrypted data (C-3 integrity)", () => {
    const key = makeTestKey();
    const manager = new MemoryManager();
    populateMemory(manager);

    const envelope = manager.encryptForStorage(key);

    // Tamper with the encrypted sensitive data
    const tampered: EncryptedMemoryEnvelope = {
      ...envelope,
      encryptedSensitive: new Uint8Array([
        ...envelope.encryptedSensitive.slice(0, -1),
        envelope.encryptedSensitive[envelope.encryptedSensitive.length - 1] ^
          0xff,
      ]),
    };

    // Should throw on decryption (GCM auth tag mismatch)
    expect(() => MemoryManager.decryptFromStorage(tampered, key)).toThrow();
  });

  it("should detect tampered plaintext metadata via integrity hash (C-3)", () => {
    const key = makeTestKey();
    const manager = new MemoryManager();
    populateMemory(manager);

    const envelope = manager.encryptForStorage(key);

    // Tamper with plaintext stats (not encrypted, but integrity-checked)
    const tampered: EncryptedMemoryEnvelope = {
      ...envelope,
      stats: { ...envelope.stats, totalContributions: 999 },
    };

    // Decryption succeeds but integrity hash won't match the tampered stats
    // Actually, the integrity hash is computed over the in-memory state before encryption,
    // so the stats in the envelope are separate from the hash. The hash validates the
    // reconstructed memory, which uses decrypted sensitive data + envelope metadata.
    // Tampering with stats means reconstructed memory differs from original.
    expect(() => MemoryManager.decryptFromStorage(tampered, key)).toThrow(
      /integrity check failed/,
    );
  });

  it("should detect tampered integrity hash", () => {
    const key = makeTestKey();
    const manager = new MemoryManager();
    populateMemory(manager);

    const envelope = manager.encryptForStorage(key);

    // Replace integrity hash with garbage
    const tampered: EncryptedMemoryEnvelope = {
      ...envelope,
      integrityHash:
        "0000000000000000000000000000000000000000000000000000000000000000",
    };

    expect(() => MemoryManager.decryptFromStorage(tampered, key)).toThrow(
      /integrity check failed/,
    );
  });

  it("should handle empty memory encryption", () => {
    const key = makeTestKey();
    const manager = new MemoryManager();

    const envelope = manager.encryptForStorage(key);
    const restored = MemoryManager.decryptFromStorage(envelope, key);

    expect(restored.getMemory().beliefs.size).toBe(0);
    expect(restored.getMemory().contributions).toHaveLength(0);
  });
});
