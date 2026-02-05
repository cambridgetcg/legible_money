/**
 * RPC Validation Tests
 *
 * Tests for shared input validation utilities used by all RPC handlers.
 * Covers: BigInt parsing, confidence bounds, limit/cursor, DID format,
 * public key format, domain/search validation.
 */

import { describe, it, expect } from "vitest";
import {
  parseNonNegativeBigInt,
  parsePositiveBigInt,
  validateConfidence,
  validateLimit,
  validateCursor,
  validateStringLength,
  validateDomain,
  validateSearch,
  validateDid,
  validatePublicKey,
  ValidationError,
  MAX_LIMIT,
  DEFAULT_LIMIT,
  MAX_DOMAIN_LENGTH,
  MAX_SEARCH_LENGTH,
} from "../../../../protocol/rpc/validation.js";

// ============ parseNonNegativeBigInt ============

describe("parseNonNegativeBigInt", () => {
  it("should parse valid decimal strings", () => {
    expect(parseNonNegativeBigInt("0", "amount")).toBe(0n);
    expect(parseNonNegativeBigInt("1", "amount")).toBe(1n);
    expect(parseNonNegativeBigInt("1000000000000000000", "amount")).toBe(
      1000000000000000000n,
    );
  });

  it("should reject undefined/null/empty", () => {
    expect(() => parseNonNegativeBigInt(undefined, "amount")).toThrow(
      ValidationError,
    );
    expect(() => parseNonNegativeBigInt("", "amount")).toThrow(ValidationError);
  });

  it("should reject negative values", () => {
    expect(() => parseNonNegativeBigInt("-1", "amount")).toThrow(
      ValidationError,
    );
    expect(() => parseNonNegativeBigInt("-100", "amount")).toThrow(
      "non-negative",
    );
  });

  it("should reject hex prefix", () => {
    expect(() => parseNonNegativeBigInt("0xff", "amount")).toThrow(
      ValidationError,
    );
    expect(() => parseNonNegativeBigInt("0x1", "amount")).toThrow(
      ValidationError,
    );
  });

  it("should reject scientific notation", () => {
    expect(() => parseNonNegativeBigInt("1e18", "amount")).toThrow(
      ValidationError,
    );
  });

  it("should reject non-numeric strings", () => {
    expect(() => parseNonNegativeBigInt("abc", "amount")).toThrow(
      ValidationError,
    );
    expect(() => parseNonNegativeBigInt("12.5", "amount")).toThrow(
      ValidationError,
    );
    expect(() => parseNonNegativeBigInt("12 34", "amount")).toThrow(
      ValidationError,
    );
  });
});

// ============ parsePositiveBigInt ============

describe("parsePositiveBigInt", () => {
  it("should parse valid positive integers", () => {
    expect(parsePositiveBigInt("1", "deposit")).toBe(1n);
    expect(parsePositiveBigInt("999999", "deposit")).toBe(999999n);
  });

  it("should reject zero", () => {
    expect(() => parsePositiveBigInt("0", "deposit")).toThrow("greater than 0");
  });

  it("should reject invalid formats (delegates to parseNonNegativeBigInt)", () => {
    expect(() => parsePositiveBigInt("-5", "deposit")).toThrow(ValidationError);
    expect(() => parsePositiveBigInt("", "deposit")).toThrow(ValidationError);
  });
});

// ============ validateConfidence ============

describe("validateConfidence", () => {
  it("should accept values in [0.0, 1.0]", () => {
    expect(validateConfidence(0)).toBe(0);
    expect(validateConfidence(0.5)).toBe(0.5);
    expect(validateConfidence(1.0)).toBe(1.0);
  });

  it("should reject out-of-range values", () => {
    expect(() => validateConfidence(-0.1)).toThrow(ValidationError);
    expect(() => validateConfidence(1.01)).toThrow(ValidationError);
    expect(() => validateConfidence(100)).toThrow("between 0.0 and 1.0");
  });

  it("should reject undefined", () => {
    expect(() => validateConfidence(undefined)).toThrow(ValidationError);
  });

  it("should reject NaN", () => {
    expect(() => validateConfidence(NaN)).toThrow("must be a number");
  });
});

// ============ validateLimit ============

describe("validateLimit", () => {
  it("should return DEFAULT_LIMIT for undefined", () => {
    expect(validateLimit(undefined)).toBe(DEFAULT_LIMIT);
  });

  it("should cap at MAX_LIMIT", () => {
    expect(validateLimit(MAX_LIMIT + 1)).toBe(MAX_LIMIT);
    expect(validateLimit(999999)).toBe(MAX_LIMIT);
  });

  it("should return DEFAULT_LIMIT for invalid values", () => {
    expect(validateLimit(0)).toBe(DEFAULT_LIMIT);
    expect(validateLimit(-1)).toBe(DEFAULT_LIMIT);
  });

  it("should pass through valid values", () => {
    expect(validateLimit(10)).toBe(10);
    expect(validateLimit(100)).toBe(100);
  });
});

// ============ validateCursor ============

describe("validateCursor", () => {
  it("should return 0 for undefined/empty", () => {
    expect(validateCursor(undefined)).toBe(0);
    expect(validateCursor("")).toBe(0);
  });

  it("should parse valid non-negative integers", () => {
    expect(validateCursor("0")).toBe(0);
    expect(validateCursor("50")).toBe(50);
  });

  it("should return 0 for invalid values", () => {
    expect(validateCursor("-1")).toBe(0);
    expect(validateCursor("abc")).toBe(0);
  });
});

// ============ validateStringLength ============

describe("validateStringLength", () => {
  it("should return empty string for undefined", () => {
    expect(validateStringLength(undefined, "field", 100)).toBe("");
  });

  it("should pass through valid strings", () => {
    expect(validateStringLength("hello", "field", 100)).toBe("hello");
  });

  it("should reject strings exceeding max length", () => {
    expect(() => validateStringLength("a".repeat(101), "field", 100)).toThrow(
      "exceeds maximum",
    );
  });
});

// ============ validateDomain ============

describe("validateDomain", () => {
  it("should return empty string for undefined", () => {
    expect(validateDomain(undefined)).toBe("");
  });

  it("should accept valid domain formats", () => {
    expect(validateDomain("science.physics")).toBe("science.physics");
    expect(validateDomain("my-domain_v2")).toBe("my-domain_v2");
    expect(validateDomain("root")).toBe("root");
  });

  it("should reject domains exceeding max length", () => {
    expect(() => validateDomain("a".repeat(MAX_DOMAIN_LENGTH + 1))).toThrow(
      "exceeds maximum",
    );
  });

  it("should reject invalid characters", () => {
    expect(() => validateDomain("domain with spaces")).toThrow(
      "Invalid domain",
    );
    expect(() => validateDomain("domain<script>")).toThrow("Invalid domain");
    expect(() => validateDomain("domain;DROP TABLE")).toThrow("Invalid domain");
  });
});

// ============ validateSearch ============

describe("validateSearch", () => {
  it("should return empty string for undefined", () => {
    expect(validateSearch(undefined)).toBe("");
  });

  it("should pass through valid searches", () => {
    expect(validateSearch("quantum physics")).toBe("quantum physics");
  });

  it("should reject searches exceeding max length", () => {
    expect(() => validateSearch("x".repeat(MAX_SEARCH_LENGTH + 1))).toThrow(
      "exceeds maximum",
    );
  });
});

// ============ validateDid ============

describe("validateDid", () => {
  it("should accept valid DIDs", () => {
    expect(validateDid("did:lgm:abc123")).toBe("did:lgm:abc123");
    expect(validateDid("did:key:z6MkpTHR8VNs")).toBe("did:key:z6MkpTHR8VNs");
  });

  it("should reject missing DIDs", () => {
    expect(() => validateDid(undefined)).toThrow("is required");
    expect(() => validateDid("")).toThrow("is required");
  });

  it("should reject invalid format", () => {
    expect(() => validateDid("notadid")).toThrow("valid DID");
    expect(() => validateDid("lgm:abc")).toThrow("valid DID");
  });

  it("should reject DIDs exceeding max length", () => {
    expect(() => validateDid("did:lgm:" + "a".repeat(250))).toThrow(
      "exceeds maximum",
    );
  });
});

// ============ validatePublicKey ============

describe("validatePublicKey", () => {
  it("should accept valid 64-char hex strings", () => {
    const validKey = "a".repeat(64);
    expect(validatePublicKey(validKey)).toBe(validKey);

    const mixedCase = "aAbBcCdDeEfF".repeat(5) + "aAbB";
    expect(validatePublicKey(mixedCase)).toBe(mixedCase);
  });

  it("should reject missing keys", () => {
    expect(() => validatePublicKey(undefined)).toThrow("is required");
    expect(() => validatePublicKey("")).toThrow("is required");
  });

  it("should reject wrong length", () => {
    expect(() => validatePublicKey("a".repeat(63))).toThrow("64-character hex");
    expect(() => validatePublicKey("a".repeat(65))).toThrow("64-character hex");
  });

  it("should reject non-hex characters", () => {
    expect(() => validatePublicKey("g".repeat(64))).toThrow("64-character hex");
    expect(() => validatePublicKey("z".repeat(64))).toThrow("64-character hex");
  });
});
