/**
 * Unit Tests for Truth Message Integrity
 *
 * Tests message signing, hashing, verification, and integrity checks.
 */

import { describe, it, expect } from "vitest";
import type { DID } from "../../../core/agents/identity/did.js";
import { generateKeypair } from "../../../core/crypto/keys.js";
import { signString, verifyString } from "../../../core/crypto/signatures.js";
import {
  generateContentHash,
  generateMessageId,
  generateNonce,
  generateVerificationCommitment,
  verifyContentHash,
  verifyMessageId,
  verifyMessageIntegrity,
  signTruthMessage,
  type TruthMessage,
  type ClaimPayload,
} from "../../../core/messaging/integrity.js";

// ============ Test Fixtures ============

const TEST_DID = "did:lgm:test-sender" as DID;

function createTestClaimPayload(): ClaimPayload {
  return {
    claim: "The speed of light is approximately 299,792,458 m/s",
    claimType: "empirical",
    epistemicCategory: "empirical",
    proposedConfidence: 0.999,
    stake: 100n * 10n ** 15n,
    evidenceIds: ["evidence-1"],
  };
}

function createUnsignedMessage(
  payload: ClaimPayload,
): Omit<TruthMessage, "messageId" | "signature" | "contentHash"> {
  return {
    messageType: "claim",
    version: "1.0.0",
    sender: TEST_DID,
    recipient: "broadcast",
    payload,
    referencedMessages: [],
    networkFeeUnit: 1n,
    networkFeeLimit: 100000n,
    timestamp: 1700000000000,
  };
}

// ============ generateContentHash Tests ============

describe("generateContentHash", () => {
  it("should generate consistent hash for same payload", () => {
    const payload = createTestClaimPayload();

    const hash1 = generateContentHash(payload);
    const hash2 = generateContentHash(payload);

    expect(hash1).toBe(hash2);
  });

  it("should generate different hash for different payloads", () => {
    const payload1 = createTestClaimPayload();
    const payload2 = { ...createTestClaimPayload(), claim: "Different claim" };

    const hash1 = generateContentHash(payload1);
    const hash2 = generateContentHash(payload2);

    expect(hash1).not.toBe(hash2);
  });

  it("should generate hash with proper prefix", () => {
    const payload = createTestClaimPayload();

    const hash = generateContentHash(payload);

    expect(hash).toMatch(/^[a-f0-9]+$/);
    expect(hash.length).toBeGreaterThan(32);
  });

  it("should handle payload with bigint values", () => {
    const payload = {
      ...createTestClaimPayload(),
      stake: 1000000000000000000000n, // Large bigint
    };

    // Should not throw
    const hash = generateContentHash(payload);

    expect(typeof hash).toBe("string");
    expect(hash.length).toBeGreaterThan(0);
  });
});

// ============ generateMessageId Tests ============

describe("generateMessageId", () => {
  it("should generate consistent ID for same message", () => {
    const payload = createTestClaimPayload();
    const message = createUnsignedMessage(payload);

    const id1 = generateMessageId(message);
    const id2 = generateMessageId(message);

    expect(id1).toBe(id2);
  });

  it("should generate different ID for different messages", () => {
    const message1 = createUnsignedMessage(createTestClaimPayload());
    const message2 = {
      ...message1,
      timestamp: message1.timestamp + 1,
    };

    const id1 = generateMessageId(message1);
    const id2 = generateMessageId(message2);

    expect(id1).not.toBe(id2);
  });

  it("should generate ID with msg_ prefix", () => {
    const message = createUnsignedMessage(createTestClaimPayload());

    const id = generateMessageId(message);

    expect(id).toMatch(/^msg_[a-f0-9]+$/);
  });

  it("should generate different IDs for different senders", () => {
    const message1 = createUnsignedMessage(createTestClaimPayload());
    const message2 = {
      ...message1,
      sender: "did:lgm:different-sender" as DID,
    };

    const id1 = generateMessageId(message1);
    const id2 = generateMessageId(message2);

    expect(id1).not.toBe(id2);
  });
});

// ============ generateNonce Tests ============

describe("generateNonce", () => {
  it("should generate unique nonces", () => {
    const nonces = new Set<string>();

    for (let i = 0; i < 100; i++) {
      nonces.add(generateNonce());
    }

    expect(nonces.size).toBe(100);
  });

  it("should generate nonce of appropriate length", () => {
    const nonce = generateNonce();

    expect(nonce.length).toBeGreaterThan(16);
    expect(nonce).toMatch(/^[a-f0-9]+$/);
  });
});

// ============ generateVerificationCommitment Tests ============

describe("generateVerificationCommitment", () => {
  it("should generate consistent commitment for same inputs", () => {
    const verdict = "confirmed" as const;
    const confidence = 0.95;
    const nonce = "test-nonce-123";
    const references = ["ref-1", "ref-2"];

    const commitment1 = generateVerificationCommitment(
      verdict,
      confidence,
      nonce,
      references,
    );
    const commitment2 = generateVerificationCommitment(
      verdict,
      confidence,
      nonce,
      references,
    );

    expect(commitment1).toBe(commitment2);
  });

  it("should generate different commitment for different verdicts", () => {
    const confidence = 0.95;
    const nonce = "test-nonce-123";
    const references = ["ref-1"];

    const commitment1 = generateVerificationCommitment(
      "confirmed",
      confidence,
      nonce,
      references,
    );
    const commitment2 = generateVerificationCommitment(
      "rejected",
      confidence,
      nonce,
      references,
    );

    expect(commitment1).not.toBe(commitment2);
  });

  it("should generate different commitment for different nonces", () => {
    const verdict = "confirmed" as const;
    const confidence = 0.95;
    const references = ["ref-1"];

    const commitment1 = generateVerificationCommitment(
      verdict,
      confidence,
      "nonce-1",
      references,
    );
    const commitment2 = generateVerificationCommitment(
      verdict,
      confidence,
      "nonce-2",
      references,
    );

    expect(commitment1).not.toBe(commitment2);
  });

  it("should generate different commitment for different confidence", () => {
    const verdict = "confirmed" as const;
    const nonce = "test-nonce-123";
    const references = ["ref-1"];

    const commitment1 = generateVerificationCommitment(
      verdict,
      0.9,
      nonce,
      references,
    );
    const commitment2 = generateVerificationCommitment(
      verdict,
      0.8,
      nonce,
      references,
    );

    expect(commitment1).not.toBe(commitment2);
  });
});

// ============ verifyContentHash Tests ============

describe("verifyContentHash", () => {
  it("should verify correct content hash", () => {
    const payload = createTestClaimPayload();
    const contentHash = generateContentHash(payload);
    const message = {
      ...createUnsignedMessage(payload),
      messageId: "msg_test",
      contentHash,
      signature: "test-sig",
    } as TruthMessage;

    expect(verifyContentHash(message)).toBe(true);
  });

  it("should reject incorrect content hash", () => {
    const payload = createTestClaimPayload();
    const message = {
      ...createUnsignedMessage(payload),
      messageId: "msg_test",
      contentHash: "incorrect_hash",
      signature: "test-sig",
    } as TruthMessage;

    expect(verifyContentHash(message)).toBe(false);
  });

  it("should reject if payload was modified", () => {
    const payload = createTestClaimPayload();
    const contentHash = generateContentHash(payload);
    const message = {
      ...createUnsignedMessage(payload),
      messageId: "msg_test",
      contentHash,
      signature: "test-sig",
    } as TruthMessage;

    // Modify payload after hash generation
    (message.payload as ClaimPayload).claim = "Modified claim";

    expect(verifyContentHash(message)).toBe(false);
  });
});

// ============ verifyMessageId Tests ============

describe("verifyMessageId", () => {
  it("should verify correct message ID", () => {
    const payload = createTestClaimPayload();
    const unsignedMessage = createUnsignedMessage(payload);
    const messageId = generateMessageId(unsignedMessage);
    const message = {
      ...unsignedMessage,
      messageId,
      contentHash: generateContentHash(payload),
      signature: "test-sig",
    } as TruthMessage;

    expect(verifyMessageId(message)).toBe(true);
  });

  it("should reject incorrect message ID", () => {
    const payload = createTestClaimPayload();
    const message = {
      ...createUnsignedMessage(payload),
      messageId: "msg_incorrect",
      contentHash: generateContentHash(payload),
      signature: "test-sig",
    } as TruthMessage;

    expect(verifyMessageId(message)).toBe(false);
  });
});

// ============ signTruthMessage Tests ============

describe("signTruthMessage", () => {
  it("should sign and verify a message", () => {
    const { publicKey, privateKey } = generateKeypair();
    const payload = createTestClaimPayload();
    const unsignedMessage = {
      ...createUnsignedMessage(payload),
      sender: publicKey as unknown as DID, // Use pubkey as DID for test
    };

    const signedMessage = signTruthMessage(
      unsignedMessage,
      privateKey,
      signString,
    );

    expect(signedMessage.signature).toBeTruthy();
    expect(signedMessage.messageId).toMatch(/^msg_/);
    expect(signedMessage.contentHash).toBeTruthy();
  });

  it("should produce verifiable signature", () => {
    const { publicKey, privateKey } = generateKeypair();
    const payload = createTestClaimPayload();
    const unsignedMessage = {
      ...createUnsignedMessage(payload),
      sender: publicKey as unknown as DID,
    };

    const signedMessage = signTruthMessage(
      unsignedMessage,
      privateKey,
      signString,
    );

    // Verify the message integrity
    const integrityResult = verifyMessageIntegrity(
      signedMessage,
      publicKey,
      verifyString,
    );

    expect(integrityResult.valid).toBe(true);
    expect(integrityResult.errors).toHaveLength(0);
  });

  it("should generate unique signatures for different messages", () => {
    const { privateKey } = generateKeypair();
    const payload1 = createTestClaimPayload();
    const payload2 = { ...createTestClaimPayload(), claim: "Different claim" };

    const message1 = signTruthMessage(
      createUnsignedMessage(payload1),
      privateKey,
      signString,
    );
    const message2 = signTruthMessage(
      createUnsignedMessage(payload2),
      privateKey,
      signString,
    );

    expect(message1.signature).not.toBe(message2.signature);
  });
});

// ============ verifyMessageIntegrity Tests ============

describe("verifyMessageIntegrity", () => {
  it("should pass for valid signed message", () => {
    const { publicKey, privateKey } = generateKeypair();
    const payload = createTestClaimPayload();
    const unsignedMessage = createUnsignedMessage(payload);

    const signedMessage = signTruthMessage(
      unsignedMessage,
      privateKey,
      signString,
    );

    const result = verifyMessageIntegrity(
      signedMessage,
      publicKey,
      verifyString,
    );

    expect(result.valid).toBe(true);
    expect(result.errors).toHaveLength(0);
  });

  it("should fail for tampered content hash", () => {
    const { publicKey, privateKey } = generateKeypair();
    const payload = createTestClaimPayload();
    const unsignedMessage = createUnsignedMessage(payload);

    const signedMessage = signTruthMessage(
      unsignedMessage,
      privateKey,
      signString,
    );

    // Tamper with content hash
    const tamperedMessage = {
      ...signedMessage,
      contentHash: "tampered_hash",
    };

    const result = verifyMessageIntegrity(
      tamperedMessage,
      publicKey,
      verifyString,
    );

    expect(result.valid).toBe(false);
    expect(result.errors.some((e) => e.toLowerCase().includes("content"))).toBe(
      true,
    );
  });

  it("should fail for tampered message ID", () => {
    const { publicKey, privateKey } = generateKeypair();
    const payload = createTestClaimPayload();
    const unsignedMessage = createUnsignedMessage(payload);

    const signedMessage = signTruthMessage(
      unsignedMessage,
      privateKey,
      signString,
    );

    // Tamper with message ID
    const tamperedMessage = {
      ...signedMessage,
      messageId: "msg_tampered",
    };

    const result = verifyMessageIntegrity(
      tamperedMessage,
      publicKey,
      verifyString,
    );

    expect(result.valid).toBe(false);
    expect(result.errors.some((e) => e.toLowerCase().includes("id"))).toBe(
      true,
    );
  });

  it("should fail for wrong public key", () => {
    const { privateKey } = generateKeypair();
    const { publicKey: wrongPublicKey } = generateKeypair();
    const payload = createTestClaimPayload();
    const unsignedMessage = createUnsignedMessage(payload);

    const signedMessage = signTruthMessage(
      unsignedMessage,
      privateKey,
      signString,
    );

    const result = verifyMessageIntegrity(
      signedMessage,
      wrongPublicKey,
      verifyString,
    );

    expect(result.valid).toBe(false);
    expect(
      result.errors.some((e) => e.toLowerCase().includes("signature")),
    ).toBe(true);
  });

  it("should fail for modified payload", () => {
    const { publicKey, privateKey } = generateKeypair();
    const payload = createTestClaimPayload();
    const unsignedMessage = createUnsignedMessage(payload);

    const signedMessage = signTruthMessage(
      unsignedMessage,
      privateKey,
      signString,
    );

    // Tamper with payload
    const tamperedMessage = {
      ...signedMessage,
      payload: {
        ...signedMessage.payload,
        claim: "Tampered claim",
      },
    } as TruthMessage;

    const result = verifyMessageIntegrity(
      tamperedMessage,
      publicKey,
      verifyString,
    );

    expect(result.valid).toBe(false);
  });
});
