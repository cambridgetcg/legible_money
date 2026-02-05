/**
 * Genesis Core Tests: Signatures
 *
 * Comprehensive tests for core/crypto/signatures.ts
 *
 * PP ALIGNMENT:
 *   TRUTH: Verified against RFC 8032 Ed25519 test vectors
 *   UNDERSTANDING: Tests demonstrate signature properties
 *   BEAUTY: Complete coverage with clear organization
 *   JUSTICE: Tests in genesis/tests/ where they belong
 *   CREATIVITY: Building verification path for Genesis Core
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  sign,
  signString,
  signDomainSeparated,
  signTransaction,
  signBlock,
  verify,
  verifyString,
  verifyDomainSeparated,
  verifyTransaction,
  verifyBlock,
  batchVerify,
  createSignedMessage,
  verifySignedMessage,
  aggregateSignatures,
  verifyAggregatedSignature,
} from "../../../core/crypto/signatures.js";
import {
  generateKeypair,
  generateKeypairFromSeed,
  bytesToHex,
  hexToBytes,
  type Keypair,
} from "../../../core/crypto/keys.js";
import { HASH_DOMAINS } from "../../../core/crypto/hashing.js";

// ============ RFC 8032 Test Vectors ============
// https://www.rfc-editor.org/rfc/rfc8032#section-7.1

const RFC8032_SIGN_VECTORS = [
  {
    name: "Test Vector 1",
    privateKey:
      "9d61b19deffd5a60ba844af492ec2cc44449c5697b326919703bac031cae7f60",
    publicKey:
      "d75a980182b10ab7d54bfed3c964073a0ee172f3daa62325af021a68f707511a",
    message: "",
    signature:
      "e5564300c360ac729086e2cc806e828a84877f1eb8e5d974d873e065224901555fb8821590a33bacc61e39701cf9b46bd25bf5f0595bbe24655141438e7a100b",
  },
  {
    name: "Test Vector 2",
    privateKey:
      "4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb",
    publicKey:
      "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c",
    message: "72",
    signature:
      "92a009a9f0d4cab8720e820b5f642540a2b27b5416503f8fb3762223ebdb69da085ac1e43e15996e458f3613d0f11d8c387b2eaeb4302aeeb00d291612bb0c00",
  },
  {
    name: "Test Vector 3",
    privateKey:
      "c5aa8df43f9f837bedb7442f31dcb7b166d38535076f094b85ce3a2e0b4458f7",
    publicKey:
      "fc51cd8e6218a1a38da47ed00230f0580816ed13ba3303ac5deb911548908025",
    message: "af82",
    signature:
      "6291d657deec24024827e69c3abe01a30ce548a284743a445e3680d7db5ac3ac18ff9b538d16f290ae67f760984dc6594a7c15e9716ed28dc027beceea1ec40a",
  },
];

// ============ Basic Signing Tests ============

describe("Basic Signing", () => {
  let keypair: Keypair;

  beforeEach(() => {
    keypair = generateKeypair();
  });

  describe("sign", () => {
    it("should produce a 64-byte signature (128 hex chars)", () => {
      const message = new TextEncoder().encode("test message");
      const signature = sign(message, keypair.privateKey);

      expect(signature).toHaveLength(128);
      expect(signature).toMatch(/^[0-9a-f]{128}$/);
    });

    it("should be deterministic", () => {
      const message = new TextEncoder().encode("test message");

      const sig1 = sign(message, keypair.privateKey);
      const sig2 = sign(message, keypair.privateKey);

      expect(sig1).toBe(sig2);
    });

    it("should produce different signatures for different messages", () => {
      const msg1 = new TextEncoder().encode("message 1");
      const msg2 = new TextEncoder().encode("message 2");

      const sig1 = sign(msg1, keypair.privateKey);
      const sig2 = sign(msg2, keypair.privateKey);

      expect(sig1).not.toBe(sig2);
    });

    it("should produce different signatures for different keys", () => {
      const message = new TextEncoder().encode("same message");
      const keypair2 = generateKeypair();

      const sig1 = sign(message, keypair.privateKey);
      const sig2 = sign(message, keypair2.privateKey);

      expect(sig1).not.toBe(sig2);
    });
  });

  describe("signString", () => {
    it("should sign string messages", () => {
      const signature = signString("hello world", keypair.privateKey);

      expect(signature).toHaveLength(128);
    });

    it("should produce same signature as sign with encoded string", () => {
      const message = "test message";
      const messageBytes = new TextEncoder().encode(message);

      const sig1 = signString(message, keypair.privateKey);
      const sig2 = sign(messageBytes, keypair.privateKey);

      expect(sig1).toBe(sig2);
    });
  });
});

// ============ Basic Verification Tests ============

describe("Basic Verification", () => {
  let keypair: Keypair;

  beforeEach(() => {
    keypair = generateKeypair();
  });

  describe("verify", () => {
    it("should verify valid signature", () => {
      const message = new TextEncoder().encode("test message");
      const signature = sign(message, keypair.privateKey);

      const isValid = verify(message, signature, keypair.publicKey);

      expect(isValid).toBe(true);
    });

    it("should reject signature with wrong message", () => {
      const message = new TextEncoder().encode("original message");
      const signature = sign(message, keypair.privateKey);

      const wrongMessage = new TextEncoder().encode("wrong message");
      const isValid = verify(wrongMessage, signature, keypair.publicKey);

      expect(isValid).toBe(false);
    });

    it("should reject signature with wrong public key", () => {
      const message = new TextEncoder().encode("test message");
      const signature = sign(message, keypair.privateKey);

      const wrongKeypair = generateKeypair();
      const isValid = verify(message, signature, wrongKeypair.publicKey);

      expect(isValid).toBe(false);
    });

    it("should reject tampered signature", () => {
      const message = new TextEncoder().encode("test message");
      const signature = sign(message, keypair.privateKey);

      // Tamper with signature (flip a bit)
      const tamperedSig =
        signature.slice(0, -1) + (signature.slice(-1) === "0" ? "1" : "0");

      const isValid = verify(message, tamperedSig, keypair.publicKey);

      expect(isValid).toBe(false);
    });

    it("should reject invalid signature format", () => {
      const message = new TextEncoder().encode("test message");

      // Too short
      expect(verify(message, "abcd", keypair.publicKey)).toBe(false);

      // Invalid hex
      expect(verify(message, "zzzz".repeat(32), keypair.publicKey)).toBe(false);
    });
  });

  describe("verifyString", () => {
    it("should verify string messages", () => {
      const message = "hello world";
      const signature = signString(message, keypair.privateKey);

      const isValid = verifyString(message, signature, keypair.publicKey);

      expect(isValid).toBe(true);
    });

    it("should reject wrong string", () => {
      const signature = signString("original", keypair.privateKey);

      const isValid = verifyString("different", signature, keypair.publicKey);

      expect(isValid).toBe(false);
    });
  });
});

// ============ RFC 8032 Test Vector Verification ============

describe("RFC 8032 Test Vectors", () => {
  for (const vector of RFC8032_SIGN_VECTORS) {
    describe(vector.name, () => {
      it("should produce correct signature", () => {
        const message = hexToBytes(vector.message);
        const signature = sign(message, vector.privateKey);

        expect(signature.toLowerCase()).toBe(vector.signature.toLowerCase());
      });

      it("should verify correct signature", () => {
        const message = hexToBytes(vector.message);

        const isValid = verify(message, vector.signature, vector.publicKey);

        expect(isValid).toBe(true);
      });

      it("should derive correct public key", () => {
        const keypair = generateKeypairFromSeed(hexToBytes(vector.privateKey));

        expect(keypair.publicKey.toLowerCase()).toBe(
          vector.publicKey.toLowerCase(),
        );
      });
    });
  }
});

// ============ Domain-Separated Signing Tests ============

describe("Domain-Separated Signing", () => {
  let keypair: Keypair;

  beforeEach(() => {
    keypair = generateKeypair();
  });

  describe("signDomainSeparated", () => {
    it("should sign with domain separation", () => {
      const signature = signDomainSeparated(
        HASH_DOMAINS.TRANSACTION,
        "transaction data",
        keypair.privateKey,
      );

      expect(signature).toHaveLength(128);
    });

    it("should produce different signatures for different domains", () => {
      const data = "same data";

      const sig1 = signDomainSeparated(
        HASH_DOMAINS.TRANSACTION,
        data,
        keypair.privateKey,
      );
      const sig2 = signDomainSeparated(
        HASH_DOMAINS.BLOCK,
        data,
        keypair.privateKey,
      );

      expect(sig1).not.toBe(sig2);
    });

    it("should be verifiable with verifyDomainSeparated", () => {
      const domain = HASH_DOMAINS.TRANSACTION;
      const data = "test data";

      const signature = signDomainSeparated(domain, data, keypair.privateKey);
      const isValid = verifyDomainSeparated(
        domain,
        data,
        signature,
        keypair.publicKey,
      );

      expect(isValid).toBe(true);
    });

    it("should fail verification with wrong domain", () => {
      const data = "test data";

      const signature = signDomainSeparated(
        HASH_DOMAINS.TRANSACTION,
        data,
        keypair.privateKey,
      );
      const isValid = verifyDomainSeparated(
        HASH_DOMAINS.BLOCK,
        data,
        signature,
        keypair.publicKey,
      );

      expect(isValid).toBe(false);
    });
  });
});

// ============ Transaction Signing Tests ============

describe("Transaction Signing", () => {
  let keypair: Keypair;

  beforeEach(() => {
    keypair = generateKeypair();
  });

  describe("signTransaction", () => {
    it("should sign transaction object", () => {
      const transaction = {
        type: "transfer",
        from: keypair.publicKey,
        to: "recipient",
        amount: 1000,
        nonce: 1,
      };

      const signature = signTransaction(transaction, keypair.privateKey);

      expect(signature).toHaveLength(128);
    });

    it("should be deterministic for same transaction", () => {
      const transaction = {
        type: "transfer",
        amount: 100,
      };

      const sig1 = signTransaction(transaction, keypair.privateKey);
      const sig2 = signTransaction(transaction, keypair.privateKey);

      expect(sig1).toBe(sig2);
    });

    it("should be independent of key order in object", () => {
      const tx1 = { a: 1, b: 2, c: 3 };
      const tx2 = { c: 3, b: 2, a: 1 };

      const sig1 = signTransaction(tx1, keypair.privateKey);
      const sig2 = signTransaction(tx2, keypair.privateKey);

      expect(sig1).toBe(sig2);
    });
  });

  describe("verifyTransaction", () => {
    it("should verify valid transaction signature", () => {
      const transaction = {
        type: "transfer",
        amount: 100,
        timestamp: Date.now(),
      };

      const signature = signTransaction(transaction, keypair.privateKey);
      const isValid = verifyTransaction(
        transaction,
        signature,
        keypair.publicKey,
      );

      expect(isValid).toBe(true);
    });

    it("should reject modified transaction", () => {
      const transaction = {
        type: "transfer",
        amount: 100,
      };

      const signature = signTransaction(transaction, keypair.privateKey);

      // Modify transaction
      const modified = { ...transaction, amount: 200 };
      const isValid = verifyTransaction(modified, signature, keypair.publicKey);

      expect(isValid).toBe(false);
    });
  });
});

// ============ Block Signing Tests ============

describe("Block Signing", () => {
  let keypair: Keypair;

  beforeEach(() => {
    keypair = generateKeypair();
  });

  describe("signBlock", () => {
    it("should sign block header", () => {
      const blockHeader = {
        height: 100,
        previousHash: "0".repeat(64),
        merkleRoot: "a".repeat(64),
        timestamp: Date.now(),
        proposer: keypair.publicKey,
      };

      const signature = signBlock(blockHeader, keypair.privateKey);

      expect(signature).toHaveLength(128);
    });
  });

  describe("verifyBlock", () => {
    it("should verify valid block signature", () => {
      const blockHeader = {
        height: 1,
        timestamp: Date.now(),
      };

      const signature = signBlock(blockHeader, keypair.privateKey);
      const isValid = verifyBlock(blockHeader, signature, keypair.publicKey);

      expect(isValid).toBe(true);
    });

    it("should reject tampered block", () => {
      const blockHeader = {
        height: 1,
        timestamp: Date.now(),
      };

      const signature = signBlock(blockHeader, keypair.privateKey);

      const tampered = { ...blockHeader, height: 2 };
      const isValid = verifyBlock(tampered, signature, keypair.publicKey);

      expect(isValid).toBe(false);
    });
  });
});

// ============ Batch Verification Tests ============

describe("Batch Verification", () => {
  it("should verify multiple valid signatures", () => {
    const items = [];

    for (let i = 0; i < 10; i++) {
      const keypair = generateKeypair();
      const message = new TextEncoder().encode(`message ${i}`);
      const signature = sign(message, keypair.privateKey);

      items.push({ message, signature, publicKey: keypair.publicKey });
    }

    const allValid = batchVerify(items);

    expect(allValid).toBe(true);
  });

  it("should reject if any signature is invalid", () => {
    const keypair1 = generateKeypair();
    const keypair2 = generateKeypair();

    const msg1 = new TextEncoder().encode("message 1");
    const msg2 = new TextEncoder().encode("message 2");

    const sig1 = sign(msg1, keypair1.privateKey);
    const sig2 = sign(msg2, keypair2.privateKey);

    // Second item has wrong public key
    const items = [
      { message: msg1, signature: sig1, publicKey: keypair1.publicKey },
      { message: msg2, signature: sig2, publicKey: keypair1.publicKey }, // Wrong!
    ];

    const allValid = batchVerify(items);

    expect(allValid).toBe(false);
  });

  it("should handle empty batch", () => {
    const allValid = batchVerify([]);

    expect(allValid).toBe(true);
  });

  it("should handle single item batch", () => {
    const keypair = generateKeypair();
    const message = new TextEncoder().encode("single");
    const signature = sign(message, keypair.privateKey);

    const allValid = batchVerify([
      { message, signature, publicKey: keypair.publicKey },
    ]);

    expect(allValid).toBe(true);
  });
});

// ============ Signed Message Tests ============

describe("Signed Messages", () => {
  let keypair: Keypair;

  beforeEach(() => {
    keypair = generateKeypair();
  });

  describe("createSignedMessage", () => {
    it("should create signed message with all fields", () => {
      const message = { data: "test", timestamp: Date.now() };

      const signed = createSignedMessage(
        message,
        keypair.privateKey,
        keypair.publicKey,
      );

      expect(signed.message).toEqual(message);
      expect(signed.signature).toHaveLength(128);
      expect(signed.signer).toBe(keypair.publicKey);
    });
  });

  describe("verifySignedMessage", () => {
    it("should verify valid signed message", () => {
      const message = { data: "test" };
      const signed = createSignedMessage(
        message,
        keypair.privateKey,
        keypair.publicKey,
      );

      const verification = verifySignedMessage(signed);

      expect(verification.valid).toBe(true);
      expect(verification.signer).toBe(keypair.publicKey);
    });

    it("should reject tampered message", () => {
      const signed = createSignedMessage(
        { data: "original" },
        keypair.privateKey,
        keypair.publicKey,
      );

      // Tamper with message
      signed.message = { data: "tampered" };

      const verification = verifySignedMessage(signed);

      expect(verification.valid).toBe(false);
    });

    it("should reject wrong signer", () => {
      const signed = createSignedMessage(
        { data: "test" },
        keypair.privateKey,
        keypair.publicKey,
      );

      // Replace signer with different public key
      const otherKeypair = generateKeypair();
      signed.signer = otherKeypair.publicKey;

      const verification = verifySignedMessage(signed);

      expect(verification.valid).toBe(false);
    });
  });
});

// ============ Aggregated Signatures Tests ============

describe("Aggregated Signatures", () => {
  it("should aggregate multiple signatures", () => {
    const signers = [];

    for (let i = 0; i < 5; i++) {
      const keypair = generateKeypair();
      const message = new TextEncoder().encode("consensus message");
      const signature = sign(message, keypair.privateKey);

      signers.push({ signature, signer: keypair.publicKey });
    }

    const aggregated = aggregateSignatures(signers);

    expect(aggregated.signatures).toHaveLength(5);
    expect(aggregated.signers).toHaveLength(5);
    expect(aggregated.bitmap).toBeInstanceOf(Uint8Array);
  });

  it("should verify aggregated signature", () => {
    const message = new TextEncoder().encode("consensus message");
    const signers = [];

    for (let i = 0; i < 3; i++) {
      const keypair = generateKeypair();
      const signature = sign(message, keypair.privateKey);

      signers.push({ signature, signer: keypair.publicKey });
    }

    const aggregated = aggregateSignatures(signers);
    const isValid = verifyAggregatedSignature(message, aggregated);

    expect(isValid).toBe(true);
  });

  it("should reject if any signature is invalid", () => {
    const message = new TextEncoder().encode("consensus message");
    const keypair1 = generateKeypair();
    const keypair2 = generateKeypair();

    const sig1 = sign(message, keypair1.privateKey);
    const sig2 = sign(
      new TextEncoder().encode("different"),
      keypair2.privateKey,
    ); // Wrong message!

    const aggregated = aggregateSignatures([
      { signature: sig1, signer: keypair1.publicKey },
      { signature: sig2, signer: keypair2.publicKey },
    ]);

    const isValid = verifyAggregatedSignature(message, aggregated);

    expect(isValid).toBe(false);
  });
});

// ============ Security Properties Tests ============

describe("Security Properties", () => {
  it("should not allow signature malleability", () => {
    const keypair = generateKeypair();
    const message = new TextEncoder().encode("test");
    const signature = sign(message, keypair.privateKey);

    // Try to create "equivalent" signature by modifying bytes
    // Ed25519 signatures should be unique
    const sigBytes = hexToBytes(signature);
    sigBytes[0] ^= 0x01; // Flip a bit
    const malleatedSig = bytesToHex(sigBytes);

    const isValid = verify(message, malleatedSig, keypair.publicKey);

    expect(isValid).toBe(false);
  });

  it("should produce unpredictable signatures without private key", () => {
    // This is more of a sanity check - signatures should look random
    const keypair = generateKeypair();
    const signatures = new Set<string>();

    for (let i = 0; i < 100; i++) {
      const message = new TextEncoder().encode(`message ${i}`);
      const signature = sign(message, keypair.privateKey);
      signatures.add(signature);
    }

    // All signatures should be unique
    expect(signatures.size).toBe(100);
  });

  it("should handle concurrent signing safely", async () => {
    const keypair = generateKeypair();
    const promises = [];

    for (let i = 0; i < 100; i++) {
      promises.push(
        Promise.resolve().then(() => {
          const message = new TextEncoder().encode(`concurrent ${i}`);
          const signature = sign(message, keypair.privateKey);
          return { message, signature };
        }),
      );
    }

    const results = await Promise.all(promises);

    // Verify all signatures
    for (const { message, signature } of results) {
      expect(verify(message, signature, keypair.publicKey)).toBe(true);
    }
  });
});

// ============ Edge Cases ============

describe("Edge Cases", () => {
  let keypair: Keypair;

  beforeEach(() => {
    keypair = generateKeypair();
  });

  it("should sign empty message", () => {
    const message = new Uint8Array(0);
    const signature = sign(message, keypair.privateKey);

    expect(signature).toHaveLength(128);
    expect(verify(message, signature, keypair.publicKey)).toBe(true);
  });

  it("should sign very long message", () => {
    const message = new Uint8Array(1000000).fill(42); // 1MB
    const signature = sign(message, keypair.privateKey);

    expect(signature).toHaveLength(128);
    expect(verify(message, signature, keypair.publicKey)).toBe(true);
  });

  it("should sign message with null bytes", () => {
    const message = new Uint8Array([0, 0, 0, 1, 0, 0, 0]);
    const signature = sign(message, keypair.privateKey);

    expect(verify(message, signature, keypair.publicKey)).toBe(true);
  });

  it("should sign unicode string", () => {
    const signature = signString("你好世界 🌍", keypair.privateKey);

    expect(verifyString("你好世界 🌍", signature, keypair.publicKey)).toBe(
      true,
    );
  });

  it("should handle nested objects in transactions", () => {
    const tx = {
      type: "complex",
      data: {
        nested: {
          deep: {
            value: 42,
          },
        },
        array: [1, 2, 3],
      },
    };

    const signature = signTransaction(tx, keypair.privateKey);

    expect(verifyTransaction(tx, signature, keypair.publicKey)).toBe(true);
  });
});
