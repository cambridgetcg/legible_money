/**
 * Claiming Pot Manager Unit Tests
 *
 * Tests pure functions: generatePotId, validatePotConfig,
 * evaluateEligibility, createPot, clonePots.
 */

import { describe, it, expect } from "vitest";
import {
  generatePotId,
  validatePotConfig,
  evaluateEligibility,
  createPot,
  clonePots,
  type AccountInfo,
} from "../../../core/claiming_pot/pot_manager.js";
import {
  type PotConfig,
  type PotEligibility,
  type ClaimingPot,
  DEFAULT_POT_ELIGIBILITY,
  DEFAULT_AMOUNT_PER_CLAIM,
  PotError,
} from "../../../core/claiming_pot/types.js";

// ============ Helpers ============

const ONE_LGM = 10n ** 18n;
/** 0.222 LGM — matches bootstrap fund per-agent amount */
const CLAIM_AMOUNT = DEFAULT_AMOUNT_PER_CLAIM;

function makeConfig(overrides: Partial<PotConfig> = {}): PotConfig {
  return {
    amountPerClaim: CLAIM_AMOUNT,
    maxClaims: 0n, // unlimited
    maxClaimsPerAccount: 1n,
    eligibility: { ...DEFAULT_POT_ELIGIBILITY },
    creatorCanClose: true,
    ...overrides,
  };
}

function makeAccountInfo(overrides: Partial<AccountInfo> = {}): AccountInfo {
  return {
    did: "did:lgm:test-claimant",
    balance: 100n * ONE_LGM,
    createdAtBlock: 0n,
    reputationScore: 0.5,
    verificationLevel: 0,
    ...overrides,
  };
}

// ============ generatePotId ============

describe("generatePotId", () => {
  it("should produce a deterministic pot DID", () => {
    const id1 = generatePotId("did:lgm:creator", 0n);
    const id2 = generatePotId("did:lgm:creator", 0n);
    expect(id1).toBe(id2);
  });

  it("should start with did:lgm:pot:", () => {
    const id = generatePotId("did:lgm:creator", 0n);
    expect(id).toMatch(/^did:lgm:pot:[0-9a-f]{32}$/);
  });

  it("should differ for different creators", () => {
    const id1 = generatePotId("did:lgm:alice", 0n);
    const id2 = generatePotId("did:lgm:bob", 0n);
    expect(id1).not.toBe(id2);
  });

  it("should differ for different nonces", () => {
    const id1 = generatePotId("did:lgm:creator", 0n);
    const id2 = generatePotId("did:lgm:creator", 1n);
    expect(id1).not.toBe(id2);
  });
});

// ============ validatePotConfig ============

describe("validatePotConfig", () => {
  it("should accept valid config", () => {
    expect(() => validatePotConfig(makeConfig(), 100n)).not.toThrow();
  });

  it("should reject zero amountPerClaim", () => {
    expect(() =>
      validatePotConfig(makeConfig({ amountPerClaim: 0n }), 100n),
    ).toThrow(PotError);
  });

  it("should reject negative amountPerClaim", () => {
    expect(() =>
      validatePotConfig(makeConfig({ amountPerClaim: -1n }), 100n),
    ).toThrow("amountPerClaim must be positive");
  });

  it("should reject maxClaimsPerAccount < 1", () => {
    expect(() =>
      validatePotConfig(makeConfig({ maxClaimsPerAccount: 0n }), 100n),
    ).toThrow("maxClaimsPerAccount must be at least 1");
  });

  it("should reject negative maxClaims", () => {
    expect(() =>
      validatePotConfig(makeConfig({ maxClaims: -1n }), 100n),
    ).toThrow("maxClaims cannot be negative");
  });

  it("should accept maxClaims of 0 (unlimited)", () => {
    expect(() =>
      validatePotConfig(makeConfig({ maxClaims: 0n }), 100n),
    ).not.toThrow();
  });

  it("should reject expiresAtBlock in the past", () => {
    expect(() =>
      validatePotConfig(makeConfig({ expiresAtBlock: 50n }), 100n),
    ).toThrow("expiresAtBlock must be in the future");
  });

  it("should reject expiresAtBlock equal to current block", () => {
    expect(() =>
      validatePotConfig(makeConfig({ expiresAtBlock: 100n }), 100n),
    ).toThrow("expiresAtBlock must be in the future");
  });

  it("should accept expiresAtBlock in the future", () => {
    expect(() =>
      validatePotConfig(makeConfig({ expiresAtBlock: 200n }), 100n),
    ).not.toThrow();
  });

  it("should accept undefined expiresAtBlock", () => {
    expect(() =>
      validatePotConfig(makeConfig({ expiresAtBlock: undefined }), 100n),
    ).not.toThrow();
  });

  it("should reject minActivityScore > 1", () => {
    const config = makeConfig({
      eligibility: { ...DEFAULT_POT_ELIGIBILITY, minActivityScore: 1.5 },
    });
    expect(() => validatePotConfig(config, 100n)).toThrow(
      "minActivityScore must be between 0 and 1",
    );
  });

  it("should reject negative minActivityScore", () => {
    const config = makeConfig({
      eligibility: { ...DEFAULT_POT_ELIGIBILITY, minActivityScore: -0.1 },
    });
    expect(() => validatePotConfig(config, 100n)).toThrow(
      "minActivityScore must be between 0 and 1",
    );
  });
});

// ============ evaluateEligibility ============

describe("evaluateEligibility", () => {
  const defaultEligibility: PotEligibility = {
    minAccountAge: 0n,
    minActivityScore: 0,
    requiredVerificationLevel: 0,
    excludedTypes: [],
    allowlist: [],
  };

  it("should approve eligible account with no restrictions", () => {
    const result = evaluateEligibility(
      defaultEligibility,
      makeAccountInfo(),
      100n,
    );
    expect(result.eligible).toBe(true);
    expect(result.reason).toBeUndefined();
  });

  // Allowlist
  it("should approve account on allowlist", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      allowlist: ["did:lgm:test-claimant", "did:lgm:other"],
    };
    const result = evaluateEligibility(eligibility, makeAccountInfo(), 100n);
    expect(result.eligible).toBe(true);
  });

  it("should reject account not on allowlist", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      allowlist: ["did:lgm:other-account"],
    };
    const result = evaluateEligibility(eligibility, makeAccountInfo(), 100n);
    expect(result.eligible).toBe(false);
    expect(result.reason).toContain("allowlist");
  });

  it("should skip allowlist check when allowlist is empty", () => {
    const result = evaluateEligibility(
      { ...defaultEligibility, allowlist: [] },
      makeAccountInfo(),
      100n,
    );
    expect(result.eligible).toBe(true);
  });

  // Excluded types
  it("should reject excluded account type", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      excludedTypes: ["system", "contract"],
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ accountType: "system" }),
      100n,
    );
    expect(result.eligible).toBe(false);
    expect(result.reason).toContain("excluded");
  });

  it("should approve non-excluded account type", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      excludedTypes: ["system"],
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ accountType: "agent" }),
      100n,
    );
    expect(result.eligible).toBe(true);
  });

  it("should approve account with no type set", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      excludedTypes: ["system"],
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ accountType: undefined }),
      100n,
    );
    expect(result.eligible).toBe(true);
  });

  // Account age
  it("should reject account that is too young", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      minAccountAge: 100n,
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ createdAtBlock: 50n }),
      100n, // age = 50
    );
    expect(result.eligible).toBe(false);
    expect(result.reason).toContain("Account age");
  });

  it("should approve account with sufficient age", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      minAccountAge: 100n,
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ createdAtBlock: 0n }),
      200n, // age = 200
    );
    expect(result.eligible).toBe(true);
  });

  // Activity score
  it("should reject account with low activity score", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      minActivityScore: 0.8,
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ reputationScore: 0.3 }),
      100n,
    );
    expect(result.eligible).toBe(false);
    expect(result.reason).toContain("Activity score");
  });

  it("should approve account with sufficient activity score", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      minActivityScore: 0.5,
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ reputationScore: 0.7 }),
      100n,
    );
    expect(result.eligible).toBe(true);
  });

  // Max balance
  it("should reject account with balance over maximum", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      maxBalance: 50n * ONE_LGM,
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ balance: 100n * ONE_LGM }),
      100n,
    );
    expect(result.eligible).toBe(false);
    expect(result.reason).toContain("exceeds maximum");
  });

  it("should approve account with balance under maximum", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      maxBalance: 200n * ONE_LGM,
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ balance: 100n * ONE_LGM }),
      100n,
    );
    expect(result.eligible).toBe(true);
  });

  it("should skip max balance check when undefined", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      maxBalance: undefined,
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ balance: 999999n * ONE_LGM }),
      100n,
    );
    expect(result.eligible).toBe(true);
  });

  // Verification level
  it("should reject account with insufficient verification level", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      requiredVerificationLevel: 2,
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ verificationLevel: 1 }),
      100n,
    );
    expect(result.eligible).toBe(false);
    expect(result.reason).toContain("Verification level");
  });

  it("should approve account meeting verification level", () => {
    const eligibility: PotEligibility = {
      ...defaultEligibility,
      requiredVerificationLevel: 2,
    };
    const result = evaluateEligibility(
      eligibility,
      makeAccountInfo({ verificationLevel: 3 }),
      100n,
    );
    expect(result.eligible).toBe(true);
  });
});

// ============ createPot ============

describe("createPot", () => {
  it("should create pot with correct fields", () => {
    const config = makeConfig();
    const pot = createPot(
      "did:lgm:pot:abc123",
      config,
      "did:lgm:creator",
      100n * ONE_LGM,
      50n,
    );

    expect(pot.potId).toBe("did:lgm:pot:abc123");
    expect(pot.creator).toBe("did:lgm:creator");
    expect(pot.config).toBe(config);
    expect(pot.totalDeposited).toBe(100n * ONE_LGM);
    expect(pot.totalClaimed).toBe(0n);
    expect(pot.balance).toBe(100n * ONE_LGM);
    expect(pot.claimCount).toBe(0n);
    expect(pot.claimedAccounts.size).toBe(0);
    expect(pot.createdAtBlock).toBe(50n);
    expect(pot.status).toBe("active");
    expect(pot.closedAtBlock).toBeUndefined();
  });
});

// ============ clonePots ============

describe("clonePots", () => {
  it("should produce a deep copy", () => {
    const original = new Map<string, ClaimingPot>();
    const pot = createPot(
      "did:lgm:pot:test",
      makeConfig(),
      "did:lgm:creator",
      100n * ONE_LGM,
      0n,
    );
    pot.claimedAccounts.set("did:lgm:alice", 1n);
    original.set(pot.potId, pot);

    const cloned = clonePots(original);

    // Same content
    expect(cloned.size).toBe(1);
    const clonedPot = cloned.get(pot.potId)!;
    expect(clonedPot.potId).toBe(pot.potId);
    expect(clonedPot.balance).toBe(pot.balance);
    expect(clonedPot.claimedAccounts.get("did:lgm:alice")).toBe(1n);

    // Mutations don't propagate
    clonedPot.claimedAccounts.set("did:lgm:bob", 2n);
    expect(pot.claimedAccounts.has("did:lgm:bob")).toBe(false);

    clonedPot.config.eligibility.excludedTypes.push("contract");
    expect(pot.config.eligibility.excludedTypes).not.toContain("contract");

    clonedPot.config.eligibility.allowlist.push("did:lgm:extra");
    expect(pot.config.eligibility.allowlist).not.toContain("did:lgm:extra");
  });

  it("should handle empty map", () => {
    const cloned = clonePots(new Map());
    expect(cloned.size).toBe(0);
  });
});
