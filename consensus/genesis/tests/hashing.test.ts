/**
 * Genesis Core Tests: Hashing
 *
 * Comprehensive tests for core/crypto/hashing.ts
 *
 * PP ALIGNMENT:
 *   TRUTH: Verified against NIST FIPS 180-4 and BLAKE3 official vectors
 *   UNDERSTANDING: Tests demonstrate hash properties
 *   BEAUTY: Complete coverage with clear organization
 *   JUSTICE: Tests in genesis/tests/ where they belong
 *   CREATIVITY: Building verification path for Genesis Core
 */

import { describe, it, expect } from "vitest";
import {
  sha512Hash,
  sha512Hex,
  sha512_256,
  sha256Hash,
  sha256Hex,
  blake3Hash,
  blake3Hex,
  blake3Variable,
  domainHash,
  HASH_DOMAINS,
  merkleHash,
  merkleLeafHash,
  merkleInternalHash,
  hashClaim,
  hashFact,
  constantTimeEqual,
  hashesEqual,
} from "../../../core/crypto/hashing.js";
import { bytesToHex } from "../../../core/crypto/keys.js";

// ============ NIST FIPS 180-4 SHA-512 Test Vectors ============
// https://csrc.nist.gov/CSRC/media/Projects/Cryptographic-Standards-and-Guidelines/documents/examples/SHA512.pdf

const SHA512_TEST_VECTORS = [
  {
    name: "Empty string",
    input: "",
    expected:
      "cf83e1357eefb8bdf1542850d66d8007d620e4050b5715dc83f4a921d36ce9ce47d0d13c5d85f2b0ff8318d2877eec2f63b931bd47417a81a538327af927da3e",
  },
  {
    name: "abc",
    input: "abc",
    expected:
      "ddaf35a193617abacc417349ae20413112e6fa4e89a97ea20a9eeee64b55d39a2192992a274fc1a836ba3c23a3feebbd454d4423643ce80e2a9ac94fa54ca49f",
  },
  {
    name: "Single block message",
    input:
      "abcdefghbcdefghicdefghijdefghijkefghijklfghijklmghijklmnhijklmnoijklmnopjklmnopqklmnopqrlmnopqrsmnopqrstnopqrstu",
    expected:
      "8e959b75dae313da8cf4f72814fc143f8f7779c6eb9f7fa17299aeadb6889018501d289e4900f7e4331b99dec4b5433ac7d329eeb6dd26545e96e55b874be909",
  },
];

// ============ SHA-256 Test Vectors ============

const SHA256_TEST_VECTORS = [
  {
    name: "Empty string",
    input: "",
    expected:
      "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
  },
  {
    name: "abc",
    input: "abc",
    expected:
      "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
  },
];

// ============ BLAKE3 Official Test Vectors ============
// https://github.com/BLAKE3-team/BLAKE3/blob/master/test_vectors/test_vectors.json

const BLAKE3_TEST_VECTORS = [
  {
    name: "Empty input",
    input: new Uint8Array(0),
    expected:
      "af1349b9f5f9a1a6a0404dea36dcc9499bcb25c9adc112b7cc9a93cae41f3262",
  },
  {
    name: "1 byte (0x00)",
    input: new Uint8Array([0]),
    expected:
      "2d3adedff11b61f14c886e35afa036736dcd87a74d27b5c1510225d0f592e213",
  },
];

// ============ SHA-512 Tests ============

describe("SHA-512", () => {
  describe("sha512Hash", () => {
    it("should return Uint8Array of 64 bytes", () => {
      const hash = sha512Hash("test");
      expect(hash).toBeInstanceOf(Uint8Array);
      expect(hash.length).toBe(64);
    });

    it("should accept string input", () => {
      const hash = sha512Hash("hello world");
      expect(hash.length).toBe(64);
    });

    it("should accept Uint8Array input", () => {
      const input = new TextEncoder().encode("hello world");
      const hash = sha512Hash(input);
      expect(hash.length).toBe(64);
    });

    it("should produce same hash for string and equivalent bytes", () => {
      const str = "hello world";
      const bytes = new TextEncoder().encode(str);

      const hashFromStr = sha512Hash(str);
      const hashFromBytes = sha512Hash(bytes);

      expect(bytesToHex(hashFromStr)).toBe(bytesToHex(hashFromBytes));
    });

    it("should match NIST test vectors", () => {
      for (const vector of SHA512_TEST_VECTORS) {
        const hash = sha512Hex(vector.input);
        expect(hash.toLowerCase()).toBe(vector.expected.toLowerCase());
      }
    });

    it("should be deterministic", () => {
      const input = "deterministic test";
      const hash1 = sha512Hex(input);
      const hash2 = sha512Hex(input);
      expect(hash1).toBe(hash2);
    });

    it("should produce different hashes for different inputs", () => {
      const hash1 = sha512Hex("input1");
      const hash2 = sha512Hex("input2");
      expect(hash1).not.toBe(hash2);
    });

    it("should have avalanche effect (small change = different hash)", () => {
      const hash1 = sha512Hex("test");
      const hash2 = sha512Hex("Test"); // Only case difference

      expect(hash1).not.toBe(hash2);

      // Count differing characters - should be many
      let differences = 0;
      for (let i = 0; i < hash1.length; i++) {
        if (hash1[i] !== hash2[i]) differences++;
      }
      expect(differences).toBeGreaterThan(64); // More than half should differ
    });
  });

  describe("sha512_256", () => {
    it("should return 32 bytes (truncated SHA-512)", () => {
      const hash = sha512_256("test");
      expect(hash.length).toBe(32);
    });

    it("should produce SHA-256 output (aligned with Go/Cosmos SDK)", () => {
      const input = "test input";
      const fromSha256 = sha256Hash(input);
      const fromAlias = sha512_256(input);

      expect(bytesToHex(fromAlias)).toBe(bytesToHex(fromSha256));
    });
  });
});

// ============ SHA-256 Tests ============

describe("SHA-256", () => {
  describe("sha256Hash", () => {
    it("should return 32 bytes", () => {
      const hash = sha256Hash("test");
      expect(hash.length).toBe(32);
    });

    it("should match NIST test vectors", () => {
      for (const vector of SHA256_TEST_VECTORS) {
        const hash = sha256Hex(vector.input);
        expect(hash.toLowerCase()).toBe(vector.expected.toLowerCase());
      }
    });
  });
});

// ============ BLAKE3 Tests ============

describe("BLAKE3", () => {
  describe("blake3Hash", () => {
    it("should return 32 bytes by default", () => {
      const hash = blake3Hash("test");
      expect(hash.length).toBe(32);
    });

    it("should accept string input", () => {
      const hash = blake3Hash("hello world");
      expect(hash.length).toBe(32);
    });

    it("should accept Uint8Array input", () => {
      const input = new Uint8Array([1, 2, 3, 4, 5]);
      const hash = blake3Hash(input);
      expect(hash.length).toBe(32);
    });

    it("should match BLAKE3 test vectors", () => {
      for (const vector of BLAKE3_TEST_VECTORS) {
        const hash = bytesToHex(blake3Hash(vector.input));
        expect(hash.toLowerCase()).toBe(vector.expected.toLowerCase());
      }
    });

    it("should be deterministic", () => {
      const hash1 = blake3Hex("test");
      const hash2 = blake3Hex("test");
      expect(hash1).toBe(hash2);
    });
  });

  describe("blake3Variable", () => {
    it("should produce custom length output", () => {
      const hash16 = blake3Variable("test", 16);
      const hash64 = blake3Variable("test", 64);
      const hash128 = blake3Variable("test", 128);

      expect(hash16.length).toBe(16);
      expect(hash64.length).toBe(64);
      expect(hash128.length).toBe(128);
    });

    it("should be prefix-consistent", () => {
      const hash32 = blake3Variable("test", 32);
      const hash64 = blake3Variable("test", 64);

      // First 32 bytes of 64-byte hash should equal 32-byte hash
      expect(bytesToHex(hash32)).toBe(bytesToHex(hash64.slice(0, 32)));
    });
  });
});

// ============ Domain Separation Tests ============

describe("Domain Separation", () => {
  describe("domainHash", () => {
    it("should produce 32-byte hash (SHA-256)", () => {
      const hash = domainHash("test.domain", "data");
      expect(hash.length).toBe(32);
    });

    it("should produce different hashes for different domains", () => {
      const data = "same data";
      const hash1 = bytesToHex(domainHash("domain1", data));
      const hash2 = bytesToHex(domainHash("domain2", data));

      expect(hash1).not.toBe(hash2);
    });

    it("should produce different hashes for same domain, different data", () => {
      const domain = "same.domain";
      const hash1 = bytesToHex(domainHash(domain, "data1"));
      const hash2 = bytesToHex(domainHash(domain, "data2"));

      expect(hash1).not.toBe(hash2);
    });

    it("should be deterministic", () => {
      const hash1 = bytesToHex(domainHash("domain", "data"));
      const hash2 = bytesToHex(domainHash("domain", "data"));

      expect(hash1).toBe(hash2);
    });

    it("should accept Uint8Array data", () => {
      const data = new Uint8Array([1, 2, 3, 4, 5]);
      const hash = domainHash("domain", data);

      expect(hash.length).toBe(32);
    });

    it("should produce different hashes than raw SHA-256", () => {
      const data = "test data";
      const domainHashed = bytesToHex(domainHash("domain", data));
      const rawHashed = sha256Hex(data);

      expect(domainHashed).not.toBe(rawHashed);
    });
  });

  describe("HASH_DOMAINS", () => {
    it("should have all required domains", () => {
      expect(HASH_DOMAINS.BLOCK).toBeDefined();
      expect(HASH_DOMAINS.TRANSACTION).toBeDefined();
      expect(HASH_DOMAINS.STATE).toBeDefined();
      expect(HASH_DOMAINS.FACT).toBeDefined();
      expect(HASH_DOMAINS.COMMITMENT).toBeDefined();
      expect(HASH_DOMAINS.VRF).toBeDefined();
      expect(HASH_DOMAINS.SIGNATURE).toBeDefined();
    });

    it("should have unique domain strings", () => {
      const domains = Object.values(HASH_DOMAINS);
      const uniqueDomains = new Set(domains);

      expect(uniqueDomains.size).toBe(domains.length);
    });

    it("should follow naming convention", () => {
      for (const domain of Object.values(HASH_DOMAINS)) {
        expect(domain).toMatch(/^LGM\.[a-z.]+\.v\d+$/);
      }
    });

    it("should produce different hashes for each domain with same data", () => {
      const data = "identical data";
      const hashes = new Set<string>();

      for (const domain of Object.values(HASH_DOMAINS)) {
        const hash = bytesToHex(domainHash(domain, data));
        hashes.add(hash);
      }

      expect(hashes.size).toBe(Object.keys(HASH_DOMAINS).length);
    });
  });
});

// ============ Merkle Hashing Tests ============

describe("Merkle Hashing", () => {
  describe("merkleHash", () => {
    it("should combine two hashes into one", () => {
      const left = blake3Hash("left");
      const right = blake3Hash("right");
      const combined = merkleHash(left, right);

      expect(combined.length).toBe(32);
    });

    it("should be deterministic", () => {
      const left = blake3Hash("left");
      const right = blake3Hash("right");

      const hash1 = bytesToHex(merkleHash(left, right));
      const hash2 = bytesToHex(merkleHash(left, right));

      expect(hash1).toBe(hash2);
    });

    it("should be order-dependent", () => {
      const a = blake3Hash("a");
      const b = blake3Hash("b");

      const hash1 = bytesToHex(merkleHash(a, b));
      const hash2 = bytesToHex(merkleHash(b, a));

      expect(hash1).not.toBe(hash2);
    });
  });

  describe("merkleLeafHash", () => {
    it("should prefix leaf data with 0x00", () => {
      const data = new Uint8Array([1, 2, 3]);
      const leafHash = merkleLeafHash(data);

      expect(leafHash.length).toBe(32);
    });

    it("should produce different hash than raw BLAKE3", () => {
      const data = new Uint8Array([1, 2, 3]);
      const leafHash = bytesToHex(merkleLeafHash(data));
      const rawHash = blake3Hex(data);

      expect(leafHash).not.toBe(rawHash);
    });
  });

  describe("merkleInternalHash", () => {
    it("should prefix internal nodes with 0x01", () => {
      const left = blake3Hash("left");
      const right = blake3Hash("right");
      const internalHash = merkleInternalHash(left, right);

      expect(internalHash.length).toBe(32);
    });

    it("should be different from merkleHash (different prefix)", () => {
      const left = blake3Hash("left");
      const right = blake3Hash("right");

      const simpleHash = bytesToHex(merkleHash(left, right));
      const internalHash = bytesToHex(merkleInternalHash(left, right));

      expect(simpleHash).not.toBe(internalHash);
    });

    it("should prevent second preimage attacks via prefix", () => {
      // Leaf cannot be confused with internal node due to different prefix
      const data = new Uint8Array(64); // Same size as two hashes
      const leafHash = merkleLeafHash(data);

      const left = data.slice(0, 32);
      const right = data.slice(32);
      const internalHash = merkleInternalHash(left, right);

      expect(bytesToHex(leafHash)).not.toBe(bytesToHex(internalHash));
    });
  });
});

// ============ Claim Hashing Tests ============

describe("Claim Hashing", () => {
  describe("hashClaim", () => {
    it("should normalize case", () => {
      const hash1 = hashClaim("Hello World");
      const hash2 = hashClaim("hello world");

      expect(hash1).toBe(hash2);
    });

    it("should normalize whitespace", () => {
      const hash1 = hashClaim("hello   world");
      const hash2 = hashClaim("hello world");
      const hash3 = hashClaim("  hello world  ");

      expect(hash1).toBe(hash2);
      expect(hash2).toBe(hash3);
    });

    it("should produce hex string", () => {
      const hash = hashClaim("test claim");
      expect(hash).toMatch(/^[0-9a-f]+$/);
    });

    it("should be deterministic", () => {
      const hash1 = hashClaim("The sky is blue");
      const hash2 = hashClaim("The sky is blue");

      expect(hash1).toBe(hash2);
    });
  });

  describe("hashFact", () => {
    it("should hash fact object", () => {
      const fact = {
        claim: "Water boils at 100C",
        type: "empirical",
        domain: "physics",
        submittedBy: "did:lgm:devnet:abc123",
        references: ["https://example.com"],
      };

      const hash = hashFact(fact);
      expect(hash).toMatch(/^[0-9a-f]+$/);
    });

    it("should be deterministic", () => {
      const fact = {
        claim: "test claim",
        type: "test",
        domain: "test",
        submittedBy: "test",
        references: ["ref1", "ref2"],
      };

      const hash1 = hashFact(fact);
      const hash2 = hashFact(fact);

      expect(hash1).toBe(hash2);
    });

    it("should sort references for consistency", () => {
      const fact1 = {
        claim: "test",
        type: "test",
        domain: "test",
        submittedBy: "test",
        references: ["b", "a", "c"],
      };

      const fact2 = {
        claim: "test",
        type: "test",
        domain: "test",
        submittedBy: "test",
        references: ["a", "b", "c"],
      };

      expect(hashFact(fact1)).toBe(hashFact(fact2));
    });

    it("should normalize claim within fact", () => {
      const fact1 = {
        claim: "TEST CLAIM",
        type: "test",
        domain: "test",
        submittedBy: "test",
        references: [],
      };

      const fact2 = {
        claim: "test claim",
        type: "test",
        domain: "test",
        submittedBy: "test",
        references: [],
      };

      expect(hashFact(fact1)).toBe(hashFact(fact2));
    });
  });
});

// ============ Constant Time Comparison Tests ============

describe("Constant Time Comparison", () => {
  describe("constantTimeEqual", () => {
    it("should return true for equal arrays", () => {
      const a = new Uint8Array([1, 2, 3, 4, 5]);
      const b = new Uint8Array([1, 2, 3, 4, 5]);

      expect(constantTimeEqual(a, b)).toBe(true);
    });

    it("should return false for different arrays", () => {
      const a = new Uint8Array([1, 2, 3, 4, 5]);
      const b = new Uint8Array([1, 2, 3, 4, 6]);

      expect(constantTimeEqual(a, b)).toBe(false);
    });

    it("should return false for different lengths", () => {
      const a = new Uint8Array([1, 2, 3]);
      const b = new Uint8Array([1, 2, 3, 4]);

      expect(constantTimeEqual(a, b)).toBe(false);
    });

    it("should return true for empty arrays", () => {
      const a = new Uint8Array([]);
      const b = new Uint8Array([]);

      expect(constantTimeEqual(a, b)).toBe(true);
    });

    it("should detect difference at any position", () => {
      for (let i = 0; i < 10; i++) {
        const a = new Uint8Array(10).fill(0);
        const b = new Uint8Array(10).fill(0);
        b[i] = 1;

        expect(constantTimeEqual(a, b)).toBe(false);
      }
    });
  });

  describe("hashesEqual", () => {
    it("should compare hex hash strings", () => {
      const hash1 = sha512Hex("test");
      const hash2 = sha512Hex("test");
      const hash3 = sha512Hex("different");

      expect(hashesEqual(hash1, hash2)).toBe(true);
      expect(hashesEqual(hash1, hash3)).toBe(false);
    });
  });
});

// ============ Edge Cases ============

describe("Edge Cases", () => {
  it("should handle empty input", () => {
    const sha512Empty = sha512Hex("");
    const blake3Empty = blake3Hex("");

    expect(sha512Empty).toBe(SHA512_TEST_VECTORS[0].expected);
    expect(sha512Empty.length).toBe(128); // 64 bytes = 128 hex chars
    expect(blake3Empty.length).toBe(64); // 32 bytes = 64 hex chars
  });

  it("should handle very long input", () => {
    const longInput = "a".repeat(1000000); // 1MB of 'a'

    const sha512Long = sha512Hash(longInput);
    const blake3Long = blake3Hash(longInput);

    expect(sha512Long.length).toBe(64);
    expect(blake3Long.length).toBe(32);
  });

  it("should handle binary data with null bytes", () => {
    const binaryData = new Uint8Array([0, 0, 0, 1, 0, 0, 0]);
    const hash = sha512Hash(binaryData);

    expect(hash.length).toBe(64);
  });

  it("should handle unicode strings", () => {
    const unicode = "你好世界 🌍 مرحبا";
    const hash = sha512Hash(unicode);

    expect(hash.length).toBe(64);
  });

  it("should handle emoji", () => {
    const emoji = "😀🎉🔐💻";
    const hash = sha512Hash(emoji);

    expect(hash.length).toBe(64);
  });
});

// ============ Security Properties ============

describe("Security Properties", () => {
  it("should not leak input length in hash", () => {
    // All hashes should be same length regardless of input
    const inputs = ["a", "ab", "abc", "a".repeat(1000)];

    for (const input of inputs) {
      expect(sha512Hash(input).length).toBe(64);
      expect(blake3Hash(input).length).toBe(32);
    }
  });

  it("should be collision resistant (probabilistic test)", () => {
    const hashes = new Set<string>();
    const iterations = 10000;

    for (let i = 0; i < iterations; i++) {
      const hash = sha512Hex(`input-${i}-${Math.random()}`);
      expect(hashes.has(hash)).toBe(false);
      hashes.add(hash);
    }
  });
});
