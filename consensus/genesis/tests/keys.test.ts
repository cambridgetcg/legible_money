/**
 * Genesis Core Tests: Keys
 *
 * Comprehensive tests for core/crypto/keys.ts
 *
 * PP ALIGNMENT:
 *   TRUTH: Verified against RFC 8032 Ed25519 test vectors
 *   UNDERSTANDING: Tests demonstrate what each function does
 *   BEAUTY: Complete coverage, clear organization
 *   JUSTICE: Tests belong here in genesis/tests/
 *   CREATIVITY: Building verification path for Genesis Core
 */

import { describe, it, expect } from "vitest";
import {
  generateKeypair,
  generateKeypairBytes,
  generateKeypairFromSeed,
  generateKeypairBytesFromSeed,
  getPublicKey,
  getPublicKeyFromBytes,
  publicKeyToAddress,
  bytesToHex,
  hexToBytes,
  type Keypair,
  type KeypairBytes,
} from "../../../core/crypto/keys.js";

// ============ RFC 8032 Test Vectors ============
// https://www.rfc-editor.org/rfc/rfc8032#section-7.1

const RFC8032_TEST_VECTORS = [
  {
    name: "Test Vector 1 (32-byte zero seed)",
    seed: "0000000000000000000000000000000000000000000000000000000000000000",
    publicKey:
      "3b6a27bcceb6a42d62a3a8d02a6f0d73653215771de243a63ac048a18b59da29",
  },
  {
    name: "Test Vector 2",
    seed: "4ccd089b28ff96da9db6c346ec114e0f5b8a319f35aba624da8cf6ed4fb8a6fb",
    publicKey:
      "3d4017c3e843895a92b70aa74d1b7ebc9c982ccf2ec4968cc0cd55f12af4660c",
  },
  {
    name: "Test Vector 3",
    seed: "c5aa8df43f9f837bedb7442f31dcb7b166d38535076f094b85ce3a2e0b4458f7",
    publicKey:
      "fc51cd8e6218a1a38da47ed00230f0580816ed13ba3303ac5deb911548908025",
  },
];

// ============ Key Generation Tests ============

describe("Key Generation", () => {
  describe("generateKeypair", () => {
    it("should generate a valid keypair", () => {
      const keypair = generateKeypair();

      expect(keypair).toHaveProperty("privateKey");
      expect(keypair).toHaveProperty("publicKey");
      expect(typeof keypair.privateKey).toBe("string");
      expect(typeof keypair.publicKey).toBe("string");
    });

    it("should generate 32-byte keys (64 hex chars)", () => {
      const keypair = generateKeypair();

      expect(keypair.privateKey).toHaveLength(64);
      expect(keypair.publicKey).toHaveLength(64);
    });

    it("should generate valid hex strings", () => {
      const keypair = generateKeypair();
      const hexRegex = /^[0-9a-f]{64}$/;

      expect(keypair.privateKey).toMatch(hexRegex);
      expect(keypair.publicKey).toMatch(hexRegex);
    });

    it("should generate different keypairs each time", () => {
      const keypair1 = generateKeypair();
      const keypair2 = generateKeypair();

      expect(keypair1.privateKey).not.toBe(keypair2.privateKey);
      expect(keypair1.publicKey).not.toBe(keypair2.publicKey);
    });

    it("should generate 1000 unique keypairs", () => {
      const privateKeys = new Set<string>();
      const publicKeys = new Set<string>();

      for (let i = 0; i < 1000; i++) {
        const keypair = generateKeypair();
        privateKeys.add(keypair.privateKey);
        publicKeys.add(keypair.publicKey);
      }

      expect(privateKeys.size).toBe(1000);
      expect(publicKeys.size).toBe(1000);
    });
  });

  describe("generateKeypairBytes", () => {
    it("should generate a valid keypair as Uint8Array", () => {
      const keypair = generateKeypairBytes();

      expect(keypair.privateKey).toBeInstanceOf(Uint8Array);
      expect(keypair.publicKey).toBeInstanceOf(Uint8Array);
      expect(keypair.privateKey.length).toBe(32);
      expect(keypair.publicKey.length).toBe(32);
    });
  });
});

// ============ Deterministic Generation Tests ============

describe("Deterministic Key Generation", () => {
  describe("generateKeypairFromSeed", () => {
    it("should generate consistent keypair from same seed", () => {
      const seed = hexToBytes(
        "0000000000000000000000000000000000000000000000000000000000000000",
      );

      const keypair1 = generateKeypairFromSeed(seed);
      const keypair2 = generateKeypairFromSeed(seed);

      expect(keypair1.privateKey).toBe(keypair2.privateKey);
      expect(keypair1.publicKey).toBe(keypair2.publicKey);
    });

    it("should generate different keypairs from different seeds", () => {
      const seed1 = hexToBytes(
        "0000000000000000000000000000000000000000000000000000000000000000",
      );
      const seed2 = hexToBytes(
        "0000000000000000000000000000000000000000000000000000000000000001",
      );

      const keypair1 = generateKeypairFromSeed(seed1);
      const keypair2 = generateKeypairFromSeed(seed2);

      expect(keypair1.privateKey).not.toBe(keypair2.privateKey);
      expect(keypair1.publicKey).not.toBe(keypair2.publicKey);
    });

    it("should reject seeds that are not 32 bytes", () => {
      const shortSeed = new Uint8Array(16);
      const longSeed = new Uint8Array(64);

      expect(() => generateKeypairFromSeed(shortSeed)).toThrow(
        "Seed must be exactly 32 bytes",
      );
      expect(() => generateKeypairFromSeed(longSeed)).toThrow(
        "Seed must be exactly 32 bytes",
      );
    });

    it("should match RFC 8032 test vectors", () => {
      for (const vector of RFC8032_TEST_VECTORS) {
        const seed = hexToBytes(vector.seed);
        const keypair = generateKeypairFromSeed(seed);

        expect(keypair.publicKey.toLowerCase()).toBe(
          vector.publicKey.toLowerCase(),
        );
      }
    });
  });

  describe("generateKeypairBytesFromSeed", () => {
    it("should generate consistent keypair bytes from same seed", () => {
      const seed = new Uint8Array(32).fill(0);

      const keypair1 = generateKeypairBytesFromSeed(seed);
      const keypair2 = generateKeypairBytesFromSeed(seed);

      expect(bytesToHex(keypair1.privateKey)).toBe(
        bytesToHex(keypair2.privateKey),
      );
      expect(bytesToHex(keypair1.publicKey)).toBe(
        bytesToHex(keypair2.publicKey),
      );
    });

    it("should reject non-32-byte seeds", () => {
      expect(() => generateKeypairBytesFromSeed(new Uint8Array(31))).toThrow();
      expect(() => generateKeypairBytesFromSeed(new Uint8Array(33))).toThrow();
    });
  });
});

// ============ Public Key Derivation Tests ============

describe("Public Key Derivation", () => {
  describe("getPublicKey", () => {
    it("should derive correct public key from private key", () => {
      const keypair = generateKeypair();
      const derivedPublicKey = getPublicKey(keypair.privateKey);

      expect(derivedPublicKey).toBe(keypair.publicKey);
    });

    it("should match RFC 8032 test vectors", () => {
      for (const vector of RFC8032_TEST_VECTORS) {
        const derivedPublicKey = getPublicKey(vector.seed);

        expect(derivedPublicKey.toLowerCase()).toBe(
          vector.publicKey.toLowerCase(),
        );
      }
    });

    it("should be deterministic", () => {
      const privateKey =
        "0000000000000000000000000000000000000000000000000000000000000001";

      const publicKey1 = getPublicKey(privateKey);
      const publicKey2 = getPublicKey(privateKey);

      expect(publicKey1).toBe(publicKey2);
    });
  });

  describe("getPublicKeyFromBytes", () => {
    it("should derive correct public key bytes", () => {
      const keypair = generateKeypairBytes();
      const derivedPublicKey = getPublicKeyFromBytes(keypair.privateKey);

      expect(bytesToHex(derivedPublicKey)).toBe(bytesToHex(keypair.publicKey));
    });
  });
});

// ============ Address Derivation Tests ============

describe("Address Derivation", () => {
  describe("publicKeyToAddress", () => {
    it("should generate address with lgm1 prefix", () => {
      const keypair = generateKeypair();
      const address = publicKeyToAddress(keypair.publicKey);

      expect(address.startsWith("lgm1")).toBe(true);
    });

    it("should generate consistent address from same public key", () => {
      const keypair = generateKeypair();

      const address1 = publicKeyToAddress(keypair.publicKey);
      const address2 = publicKeyToAddress(keypair.publicKey);

      expect(address1).toBe(address2);
    });

    it("should generate address of correct length", () => {
      const keypair = generateKeypair();
      const address = publicKeyToAddress(keypair.publicKey);

      // lgm1 (4 chars) + 40 hex chars (20 bytes) = 44 chars
      expect(address.length).toBe(44);
    });

    it("should generate different addresses for different public keys", () => {
      const keypair1 = generateKeypair();
      const keypair2 = generateKeypair();

      const address1 = publicKeyToAddress(keypair1.publicKey);
      const address2 = publicKeyToAddress(keypair2.publicKey);

      expect(address1).not.toBe(address2);
    });

    it("should generate valid hex after prefix", () => {
      const keypair = generateKeypair();
      const address = publicKeyToAddress(keypair.publicKey);
      const hexPart = address.slice(4);

      expect(hexPart).toMatch(/^[0-9a-f]{40}$/);
    });
  });
});

// ============ Hex Encoding Tests ============

describe("Hex Encoding", () => {
  describe("bytesToHex", () => {
    it("should encode empty array", () => {
      expect(bytesToHex(new Uint8Array([]))).toBe("");
    });

    it("should encode single byte", () => {
      expect(bytesToHex(new Uint8Array([0]))).toBe("00");
      expect(bytesToHex(new Uint8Array([255]))).toBe("ff");
      expect(bytesToHex(new Uint8Array([16]))).toBe("10");
    });

    it("should encode multiple bytes", () => {
      expect(bytesToHex(new Uint8Array([0, 1, 2, 3]))).toBe("00010203");
      expect(bytesToHex(new Uint8Array([255, 254, 253]))).toBe("fffefd");
    });

    it("should pad single-digit hex values", () => {
      expect(bytesToHex(new Uint8Array([0, 1, 2, 3, 4, 5, 6, 7, 8, 9]))).toBe(
        "00010203040506070809",
      );
    });

    it("should produce lowercase hex", () => {
      const result = bytesToHex(new Uint8Array([170, 187, 204, 221, 238, 255]));
      expect(result).toBe("aabbccddeeff");
      expect(result).toBe(result.toLowerCase());
    });
  });

  describe("hexToBytes", () => {
    it("should decode empty string", () => {
      const result = hexToBytes("");
      expect(result).toBeInstanceOf(Uint8Array);
      expect(result.length).toBe(0);
    });

    it("should decode single byte", () => {
      expect(Array.from(hexToBytes("00"))).toEqual([0]);
      expect(Array.from(hexToBytes("ff"))).toEqual([255]);
      expect(Array.from(hexToBytes("10"))).toEqual([16]);
    });

    it("should decode multiple bytes", () => {
      expect(Array.from(hexToBytes("00010203"))).toEqual([0, 1, 2, 3]);
      expect(Array.from(hexToBytes("fffefd"))).toEqual([255, 254, 253]);
    });

    it("should handle uppercase hex", () => {
      expect(Array.from(hexToBytes("AABBCCDD"))).toEqual([170, 187, 204, 221]);
    });

    it("should handle mixed case hex", () => {
      expect(Array.from(hexToBytes("AaBbCcDd"))).toEqual([170, 187, 204, 221]);
    });
  });

  describe("roundtrip", () => {
    it("should roundtrip bytes -> hex -> bytes", () => {
      const original = new Uint8Array([0, 127, 255, 1, 254, 128]);
      const hex = bytesToHex(original);
      const recovered = hexToBytes(hex);

      expect(Array.from(recovered)).toEqual(Array.from(original));
    });

    it("should roundtrip hex -> bytes -> hex", () => {
      // Note: hex string must have even length (each byte = 2 hex chars)
      const original = "deadbeef01234567890abcde";
      const bytes = hexToBytes(original);
      const recovered = bytesToHex(bytes);

      expect(recovered).toBe(original.toLowerCase());
    });

    it("should roundtrip keypair encoding", () => {
      const keypair = generateKeypairBytes();

      const privateHex = bytesToHex(keypair.privateKey);
      const publicHex = bytesToHex(keypair.publicKey);

      const privateRecovered = hexToBytes(privateHex);
      const publicRecovered = hexToBytes(publicHex);

      expect(Array.from(privateRecovered)).toEqual(
        Array.from(keypair.privateKey),
      );
      expect(Array.from(publicRecovered)).toEqual(
        Array.from(keypair.publicKey),
      );
    });
  });
});

// ============ Edge Cases ============

describe("Edge Cases", () => {
  it("should handle all-zero seed", () => {
    const seed = new Uint8Array(32).fill(0);
    const keypair = generateKeypairFromSeed(seed);

    expect(keypair.privateKey).toHaveLength(64);
    expect(keypair.publicKey).toHaveLength(64);
  });

  it("should handle all-ones seed", () => {
    const seed = new Uint8Array(32).fill(255);
    const keypair = generateKeypairFromSeed(seed);

    expect(keypair.privateKey).toHaveLength(64);
    expect(keypair.publicKey).toHaveLength(64);
  });

  it("should handle sequential byte seed", () => {
    const seed = new Uint8Array(32);
    for (let i = 0; i < 32; i++) {
      seed[i] = i;
    }
    const keypair = generateKeypairFromSeed(seed);

    expect(keypair.privateKey).toHaveLength(64);
    expect(keypair.publicKey).toHaveLength(64);
  });
});

// ============ Type Safety Tests ============

describe("Type Safety", () => {
  it("should return correct types for generateKeypair", () => {
    const keypair: Keypair = generateKeypair();

    // TypeScript compilation proves types are correct
    expect(keypair.privateKey).toBeDefined();
    expect(keypair.publicKey).toBeDefined();

    expect(typeof keypair.privateKey).toBe("string");
    expect(typeof keypair.publicKey).toBe("string");
  });

  it("should return correct types for generateKeypairBytes", () => {
    const keypair: KeypairBytes = generateKeypairBytes();

    expect(keypair.privateKey).toBeInstanceOf(Uint8Array);
    expect(keypair.publicKey).toBeInstanceOf(Uint8Array);
  });
});
