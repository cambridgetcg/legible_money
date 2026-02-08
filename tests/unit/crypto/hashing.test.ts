/**
 * Hashing Module Tests
 *
 * Comprehensive tests for all cryptographic hash functions in Legible Money:
 * SHA-512, SHA-512/256, SHA-256, BLAKE3, domain-separated hashing,
 * Merkle hashing, claim/fact hashing, and constant-time comparison.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  sha512Hash,
  sha512Hex,
  sha512_256,
  sha512_256Hex,
  sha256Hash,
  sha256Hex,
  blake3Hash,
  blake3Hex,
  blake3Variable,
  domainHash,
  HASH_DOMAINS,
  CHAIN_IDS,
  ACTIVE_CHAIN_ID,
  setActiveChainId,
  _resetChainIdForTesting,
  merkleHash,
  merkleLeafHash,
  merkleInternalHash,
  hashClaim,
  hashFact,
  constantTimeEqual,
  hashesEqual,
} from "../../../core/crypto/hashing.js";

// ============ SHA-512 ============

describe("sha512Hash / sha512Hex", () => {
  it("produces deterministic output for the same string input", () => {
    const a = sha512Hex("hello");
    const b = sha512Hex("hello");
    expect(a).toBe(b);
  });

  it("produces deterministic output for the same Uint8Array input", () => {
    const data = new TextEncoder().encode("legible money");
    const a = sha512Hash(data);
    const b = sha512Hash(data);
    expect(a).toEqual(b);
  });

  it("produces different hashes for different inputs", () => {
    const a = sha512Hex("hello");
    const b = sha512Hex("world");
    expect(a).not.toBe(b);
  });

  it("returns 64 bytes (512 bits) as HashBytes", () => {
    const hash = sha512Hash("test");
    expect(hash).toBeInstanceOf(Uint8Array);
    expect(hash.length).toBe(64);
  });

  it("returns 128 hex chars as hex string", () => {
    const hex = sha512Hex("test");
    expect(hex).toMatch(/^[0-9a-f]{128}$/);
  });

  it("produces identical output for string and equivalent Uint8Array", () => {
    const str = "legible money protocol";
    const bytes = new TextEncoder().encode(str);
    expect(sha512Hex(str)).toBe(sha512Hex(bytes));
  });
});

// ============ SHA-512/256 ============

describe("sha512_256 / sha512_256Hex", () => {
  it("returns exactly 32 bytes (256 bits)", () => {
    const hash = sha512_256("test");
    expect(hash).toBeInstanceOf(Uint8Array);
    expect(hash.length).toBe(32);
  });

  it("returns exactly 64 hex chars", () => {
    const hex = sha512_256Hex("test");
    expect(hex).toMatch(/^[0-9a-f]{64}$/);
  });

  it("is the first 32 bytes of the full SHA-512", () => {
    const full = sha512Hash("test");
    const truncated = sha512_256("test");
    expect(truncated).toEqual(full.slice(0, 32));
  });

  it("is deterministic", () => {
    expect(sha512_256Hex("abc")).toBe(sha512_256Hex("abc"));
  });

  it("produces different hashes for different inputs", () => {
    expect(sha512_256Hex("a")).not.toBe(sha512_256Hex("b"));
  });
});

// ============ SHA-256 ============

describe("sha256Hash / sha256Hex", () => {
  it("returns 32 bytes", () => {
    const hash = sha256Hash("test");
    expect(hash.length).toBe(32);
  });

  it("returns 64 hex chars", () => {
    const hex = sha256Hex("test");
    expect(hex).toMatch(/^[0-9a-f]{64}$/);
  });

  it("is deterministic", () => {
    expect(sha256Hex("proof of truth")).toBe(sha256Hex("proof of truth"));
  });

  it("differs from SHA-512/256 for the same input", () => {
    // SHA-256 and SHA-512/256 are distinct algorithms
    expect(sha256Hex("test")).not.toBe(sha512_256Hex("test"));
  });
});

// ============ BLAKE3 ============

describe("blake3Hash / blake3Hex / blake3Variable", () => {
  it("returns 32 bytes by default", () => {
    const hash = blake3Hash("test");
    expect(hash.length).toBe(32);
  });

  it("returns 64 hex chars by default", () => {
    const hex = blake3Hex("test");
    expect(hex).toMatch(/^[0-9a-f]{64}$/);
  });

  it("is deterministic", () => {
    expect(blake3Hex("lgm")).toBe(blake3Hex("lgm"));
  });

  it("produces different hashes for different inputs", () => {
    expect(blake3Hex("a")).not.toBe(blake3Hex("b"));
  });

  it("accepts both string and Uint8Array", () => {
    const str = "blake3 test";
    const bytes = new TextEncoder().encode(str);
    expect(blake3Hex(str)).toBe(blake3Hex(bytes));
  });

  it("blake3Variable produces correct output lengths", () => {
    expect(blake3Variable("test", 16).length).toBe(16);
    expect(blake3Variable("test", 48).length).toBe(48);
    expect(blake3Variable("test", 64).length).toBe(64);
    expect(blake3Variable("test", 1).length).toBe(1);
  });

  it("blake3Variable is deterministic for same length", () => {
    const a = blake3Variable("seed", 24);
    const b = blake3Variable("seed", 24);
    expect(a).toEqual(b);
  });

  it("blake3Variable with 32 bytes matches default blake3Hash", () => {
    const variable = blake3Variable("test", 32);
    const standard = blake3Hash("test");
    expect(variable).toEqual(standard);
  });
});

// ============ Domain-Separated Hashing ============

describe("domainHash", () => {
  it("returns a Uint8Array (HashBytes)", () => {
    const result = domainHash("LGM.test.v1", "data");
    expect(result).toBeInstanceOf(Uint8Array);
    expect(result.length).toBe(64); // SHA-512 output
  });

  it("is deterministic", () => {
    const a = domainHash("LGM.test.v1", "data");
    const b = domainHash("LGM.test.v1", "data");
    expect(a).toEqual(b);
  });

  it("different domains produce different hashes for same data", () => {
    const a = domainHash("LGM.block.v1", "same-data");
    const b = domainHash("LGM.tx.v1", "same-data");
    expect(a).not.toEqual(b);
  });

  it("same domain with different data produces different hashes", () => {
    const a = domainHash("LGM.test.v1", "data-a");
    const b = domainHash("LGM.test.v1", "data-b");
    expect(a).not.toEqual(b);
  });

  it("accepts Uint8Array data", () => {
    const data = new Uint8Array([1, 2, 3, 4]);
    const result = domainHash("LGM.test.v1", data);
    expect(result).toBeInstanceOf(Uint8Array);
    expect(result.length).toBe(64);
  });
});

// ============ HASH_DOMAINS ============

describe("HASH_DOMAINS", () => {
  it("has all expected domain keys", () => {
    expect(HASH_DOMAINS.BLOCK).toBe("LGM.block.v1");
    expect(HASH_DOMAINS.TRANSACTION).toBe("LGM.tx.v1");
    expect(HASH_DOMAINS.STATE).toBe("LGM.state.v1");
    expect(HASH_DOMAINS.FACT).toBe("LGM.fact.v1");
    expect(HASH_DOMAINS.AXIOM).toBe("LGM.axiom.v1");
    expect(HASH_DOMAINS.DERIVATION).toBe("LGM.derivation.v1");
    expect(HASH_DOMAINS.COMMITMENT).toBe("LGM.commitment.v1");
    expect(HASH_DOMAINS.VRF).toBe("LGM.vrf.v1");
    expect(HASH_DOMAINS.SIGNATURE).toBe("LGM.sig.v1");
    expect(HASH_DOMAINS.SCHEDULE_EXECUTION).toBe("LGM.schedule.exec.v1");
    expect(HASH_DOMAINS.SCHEDULE_TX).toBe("LGM.schedule.tx.v1");
  });

  it("all domain values follow LGM.*.v1 convention", () => {
    for (const [, value] of Object.entries(HASH_DOMAINS)) {
      expect(value).toMatch(/^LGM\..+\.v1$/);
    }
  });
});

// ============ CHAIN_IDS and setActiveChainId ============

describe("CHAIN_IDS", () => {
  it("has mainnet, testnet, devnet, and local", () => {
    expect(CHAIN_IDS.MAINNET).toBe("lgm-mainnet-1");
    expect(CHAIN_IDS.TESTNET).toBe("lgm-testnet-1");
    expect(CHAIN_IDS.DEVNET).toBe("lgm-devnet-1");
    expect(CHAIN_IDS.LOCAL).toBe("lgm-local-0");
  });
});

describe.sequential("setActiveChainId / _resetChainIdForTesting", () => {
  beforeEach(() => {
    _resetChainIdForTesting();
  });

  it("defaults to DEVNET after reset", () => {
    expect(ACTIVE_CHAIN_ID).toBe(CHAIN_IDS.DEVNET);
  });

  it("sets and locks the chain ID on first call", () => {
    setActiveChainId(CHAIN_IDS.MAINNET);
    expect(ACTIVE_CHAIN_ID).toBe(CHAIN_IDS.MAINNET);
  });

  it("allows setting the same value twice (no-op)", () => {
    setActiveChainId(CHAIN_IDS.TESTNET);
    expect(() => setActiveChainId(CHAIN_IDS.TESTNET)).not.toThrow();
  });

  it("throws when changing to a different value after lock", () => {
    setActiveChainId(CHAIN_IDS.MAINNET);
    expect(() => setActiveChainId(CHAIN_IDS.TESTNET)).toThrow(
      /Cannot change chain ID/,
    );
  });

  it("_resetChainIdForTesting unlocks, allowing a new chain ID", () => {
    setActiveChainId(CHAIN_IDS.MAINNET);
    _resetChainIdForTesting();
    expect(() => setActiveChainId(CHAIN_IDS.LOCAL)).not.toThrow();
    expect(ACTIVE_CHAIN_ID).toBe(CHAIN_IDS.LOCAL);
  });
});

// ============ Merkle Hashing ============

describe("merkleHash", () => {
  it("produces a 32-byte hash from two inputs", () => {
    const left = blake3Hash("left");
    const right = blake3Hash("right");
    const result = merkleHash(left, right);
    expect(result.length).toBe(32);
  });

  it("is deterministic", () => {
    const left = blake3Hash("left");
    const right = blake3Hash("right");
    expect(merkleHash(left, right)).toEqual(merkleHash(left, right));
  });

  it("is order-dependent (not commutative)", () => {
    const a = blake3Hash("a");
    const b = blake3Hash("b");
    expect(merkleHash(a, b)).not.toEqual(merkleHash(b, a));
  });
});

describe("merkleLeafHash", () => {
  it("prefixes data with 0x00", () => {
    const data = new Uint8Array([10, 20, 30]);
    const result = merkleLeafHash(data);
    expect(result.length).toBe(32);

    // Manually verify prefix behavior: hash(0x00 || data) via blake3
    const prefixed = new Uint8Array(1 + data.length);
    prefixed[0] = 0x00;
    prefixed.set(data, 1);
    const expected = blake3Hash(prefixed);
    expect(result).toEqual(expected);
  });

  it("differs from merkleInternalHash for the same pair of inputs", () => {
    const data = blake3Hash("node");
    // Use leaf hash vs internal hash on same data, treated as left+right
    const leaf = merkleLeafHash(data);
    const internal = merkleInternalHash(data, data);
    expect(leaf).not.toEqual(internal);
  });
});

describe("merkleInternalHash", () => {
  it("prefixes data with 0x01", () => {
    const left = blake3Hash("left");
    const right = blake3Hash("right");
    const result = merkleInternalHash(left, right);
    expect(result.length).toBe(32);

    // Manually verify prefix behavior
    const combined = new Uint8Array(1 + left.length + right.length);
    combined[0] = 0x01;
    combined.set(left, 1);
    combined.set(right, 1 + left.length);
    const expected = blake3Hash(combined);
    expect(result).toEqual(expected);
  });
});

// ============ Claim Hashing ============

describe("hashClaim", () => {
  it("returns a hex string", () => {
    const result = hashClaim("The sky is blue");
    expect(result).toMatch(/^[0-9a-f]+$/);
  });

  it("normalizes to lowercase", () => {
    expect(hashClaim("The Sky Is Blue")).toBe(hashClaim("the sky is blue"));
  });

  it("trims leading and trailing whitespace", () => {
    expect(hashClaim("  hello  ")).toBe(hashClaim("hello"));
  });

  it("collapses internal whitespace", () => {
    expect(hashClaim("hello    world")).toBe(hashClaim("hello world"));
  });

  it("full normalization: trim + lowercase + collapse", () => {
    const a = hashClaim("  The   SKY   is    BLUE  ");
    const b = hashClaim("the sky is blue");
    expect(a).toBe(b);
  });

  it("different claims produce different hashes", () => {
    expect(hashClaim("water is wet")).not.toBe(hashClaim("fire is hot"));
  });
});

// ============ Fact Hashing ============

describe("hashFact", () => {
  const baseFact = {
    claim: "Water boils at 100C",
    type: "empirical",
    domain: "chemistry",
    submittedBy: "did:lgm:abc123",
    references: ["ref-1", "ref-2"],
  };

  it("returns a hex string", () => {
    const result = hashFact(baseFact);
    expect(result).toMatch(/^[0-9a-f]+$/);
  });

  it("is deterministic", () => {
    expect(hashFact(baseFact)).toBe(hashFact({ ...baseFact }));
  });

  it("sorts references for deterministic output", () => {
    const a = hashFact({ ...baseFact, references: ["ref-2", "ref-1"] });
    const b = hashFact({ ...baseFact, references: ["ref-1", "ref-2"] });
    expect(a).toBe(b);
  });

  it("different claims produce different hashes", () => {
    const other = { ...baseFact, claim: "Water freezes at 0C" };
    expect(hashFact(baseFact)).not.toBe(hashFact(other));
  });

  it("different submitters produce different hashes", () => {
    const other = { ...baseFact, submittedBy: "did:lgm:xyz789" };
    expect(hashFact(baseFact)).not.toBe(hashFact(other));
  });
});

// ============ Constant-Time Comparison ============

describe("constantTimeEqual", () => {
  it("returns true for identical arrays", () => {
    const a = new Uint8Array([1, 2, 3, 4]);
    const b = new Uint8Array([1, 2, 3, 4]);
    expect(constantTimeEqual(a, b)).toBe(true);
  });

  it("returns false for different arrays", () => {
    const a = new Uint8Array([1, 2, 3, 4]);
    const b = new Uint8Array([1, 2, 3, 5]);
    expect(constantTimeEqual(a, b)).toBe(false);
  });

  it("returns false for different lengths", () => {
    const a = new Uint8Array([1, 2, 3]);
    const b = new Uint8Array([1, 2, 3, 4]);
    expect(constantTimeEqual(a, b)).toBe(false);
  });

  it("returns true for empty arrays", () => {
    expect(constantTimeEqual(new Uint8Array(0), new Uint8Array(0))).toBe(true);
  });

  it("works with hash outputs", () => {
    const hash1 = sha512Hash("same");
    const hash2 = sha512Hash("same");
    const hash3 = sha512Hash("different");
    expect(constantTimeEqual(hash1, hash2)).toBe(true);
    expect(constantTimeEqual(hash1, hash3)).toBe(false);
  });
});

describe("hashesEqual", () => {
  it("returns true for identical hex hashes", () => {
    const h = sha512Hex("test");
    expect(hashesEqual(h, h)).toBe(true);
  });

  it("returns false for different hex hashes", () => {
    const a = sha512Hex("a");
    const b = sha512Hex("b");
    expect(hashesEqual(a, b)).toBe(false);
  });

  it("compares hex strings from different hash functions correctly", () => {
    const a = blake3Hex("test");
    const b = blake3Hex("test");
    expect(hashesEqual(a, b)).toBe(true);
  });
});
