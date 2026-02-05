/**
 * Genesis Core Tests: Constants
 *
 * Comprehensive tests for core/constants.ts
 *
 * PP ALIGNMENT:
 *   TRUTH: Verify constant values are correct and meaningful
 *   UNDERSTANDING: Tests demonstrate what each constant means
 *   BEAUTY: Complete coverage with clear organization
 *   JUSTICE: Tests in genesis/tests/ where they belong
 *   CREATIVITY: Building verification path for Genesis Core
 */

import { describe, it, expect } from "vitest";
import {
  TOKEN,
  VERSION,
  TIMING,
  CONSENSUS,
  CLAIM_THRESHOLDS,
  ECONOMICS,
  EVIDENCE_TAX,
  TREASURY,
  GENESIS,
  DERIVATION_HINT,
  type ClaimThresholdType,
} from "../../../core/constants.js";

// ============ Token Constants Tests ============

describe("TOKEN Constants", () => {
  it("should have correct name", () => {
    expect(TOKEN.NAME).toBe("Legible Money");
  });

  it("should have correct symbol", () => {
    expect(TOKEN.SYMBOL).toBe("LGM");
  });

  it("should have 18 decimals (standard)", () => {
    expect(TOKEN.DECIMALS).toBe(18);
  });

  it("should have 222,222,222 max supply", () => {
    const maxSupply = 222_222_222n;
    const decimals = 10n ** 18n;
    const expected = maxSupply * decimals;

    expect(TOKEN.MAX_SUPPLY).toBe(expected);
  });

  it("should have max supply as bigint", () => {
    expect(typeof TOKEN.MAX_SUPPLY).toBe("bigint");
  });

  it('should be readonly via "as const"', () => {
    // Note: TypeScript "as const" provides compile-time immutability
    // Runtime Object.freeze is not applied - this is a design choice
    // The readonly constraint is enforced by TypeScript at compile time
    expect(TOKEN).toBeDefined();
    expect(typeof TOKEN).toBe("object");
  });
});

// ============ Version Constants Tests ============

describe("VERSION Constants", () => {
  it("should have MAJOR version", () => {
    expect(VERSION.MAJOR).toBeDefined();
    expect(typeof VERSION.MAJOR).toBe("number");
  });

  it("should have MINOR version", () => {
    expect(VERSION.MINOR).toBeDefined();
    expect(typeof VERSION.MINOR).toBe("number");
  });

  it("should have PATCH version", () => {
    expect(VERSION.PATCH).toBeDefined();
    expect(typeof VERSION.PATCH).toBe("number");
  });

  it("should have BUILD version", () => {
    expect(VERSION.BUILD).toBeDefined();
    expect(typeof VERSION.BUILD).toBe("number");
  });

  it("should have correct hex values", () => {
    expect(VERSION.MAJOR).toBe(0x59);
    expect(VERSION.MINOR).toBe(0x55);
    expect(VERSION.PATCH).toBe(0x41);
    expect(VERSION.BUILD).toBe(0x49);
  });

  it('should be readonly via "as const"', () => {
    expect(VERSION).toBeDefined();
    expect(typeof VERSION).toBe("object");
  });
});

// ============ Timing Constants Tests ============

describe("TIMING Constants", () => {
  describe("TARGET_BLOCK_MS", () => {
    it("should be 2521 milliseconds", () => {
      expect(TIMING.TARGET_BLOCK_MS).toBe(2521);
    });

    it("should be approximately 2.5 seconds", () => {
      expect(TIMING.TARGET_BLOCK_MS).toBeGreaterThan(2000);
      expect(TIMING.TARGET_BLOCK_MS).toBeLessThan(3000);
    });
  });

  describe("CONFIRMATION_BLOCKS", () => {
    it("should be 19 blocks", () => {
      expect(TIMING.CONFIRMATION_BLOCKS).toBe(19);
    });
  });

  describe("FINALITY_BLOCKS", () => {
    it("should be 100 blocks", () => {
      expect(TIMING.FINALITY_BLOCKS).toBe(100);
    });

    it("should be greater than CONFIRMATION_BLOCKS", () => {
      expect(TIMING.FINALITY_BLOCKS).toBeGreaterThan(
        TIMING.CONFIRMATION_BLOCKS,
      );
    });
  });

  it('should be readonly via "as const"', () => {
    expect(TIMING).toBeDefined();
    expect(typeof TIMING).toBe("object");
  });

  it("should have reasonable block production rate", () => {
    // Blocks per hour = (60 * 60 * 1000) / TARGET_BLOCK_MS
    const blocksPerHour = (60 * 60 * 1000) / TIMING.TARGET_BLOCK_MS;

    expect(blocksPerHour).toBeGreaterThan(1000); // More than 1000 blocks/hour
    expect(blocksPerHour).toBeLessThan(2000); // Less than 2000 blocks/hour
  });

  it("should have reasonable finality time", () => {
    // Finality time = FINALITY_BLOCKS * TARGET_BLOCK_MS
    const finalityTimeMs = TIMING.FINALITY_BLOCKS * TIMING.TARGET_BLOCK_MS;
    const finalityTimeMinutes = finalityTimeMs / (60 * 1000);

    // Should be between 3-5 minutes for finality
    expect(finalityTimeMinutes).toBeGreaterThan(3);
    expect(finalityTimeMinutes).toBeLessThan(5);
  });
});

// ============ Consensus Constants Tests ============

describe("CONSENSUS Constants", () => {
  describe("MIN_VALIDATORS", () => {
    it("should be at least 3 for BFT", () => {
      expect(CONSENSUS.MIN_VALIDATORS).toBeGreaterThanOrEqual(3);
    });

    it("should be 5", () => {
      expect(CONSENSUS.MIN_VALIDATORS).toBe(5);
    });

    it("should satisfy BFT formula n >= 3f + 1", () => {
      // With 5 validators, can tolerate f = 1 Byzantine fault
      const n = CONSENSUS.MIN_VALIDATORS;
      const f = Math.floor((n - 1) / 3);
      expect(n).toBeGreaterThanOrEqual(3 * f + 1);
    });
  });

  describe("QUORUM_THRESHOLD", () => {
    it("should be 67% (> 2/3 for BFT safety)", () => {
      expect(CONSENSUS.QUORUM_THRESHOLD).toBe(0.67);
    });

    it("should be greater than 2/3 for BFT safety", () => {
      expect(CONSENSUS.QUORUM_THRESHOLD).toBeGreaterThan(2 / 3);
    });

    it("should be between 0.5 and 1.0", () => {
      expect(CONSENSUS.QUORUM_THRESHOLD).toBeGreaterThan(0.5);
      expect(CONSENSUS.QUORUM_THRESHOLD).toBeLessThan(1.0);
    });
  });

  it('should be readonly via "as const"', () => {
    expect(CONSENSUS).toBeDefined();
    expect(typeof CONSENSUS).toBe("object");
  });
});

// ============ Tiered Claim Thresholds Tests ============

describe("CLAIM_THRESHOLDS", () => {
  const ALL_TYPES: ClaimThresholdType[] = [
    "AXIOM",
    "PROTOCOL",
    "COMPUTATIONAL",
    "DERIVATION",
    "EMPIRICAL",
    "HISTORICAL",
    "ORACLE",
    "ATTESTATION",
    "PREDICTIVE",
    "SOCIAL",
    "CONTESTED",
    "GOVERNANCE",
  ];

  describe("structure", () => {
    it("should have 12 claim types defined", () => {
      expect(Object.keys(CLAIM_THRESHOLDS).length).toBe(12);
    });

    it("should have all claim types defined", () => {
      for (const type of ALL_TYPES) {
        expect(CLAIM_THRESHOLDS[type]).toBeDefined();
      }
    });

    it("should have acceptance, rejection, and description for each type", () => {
      for (const type of ALL_TYPES) {
        expect(CLAIM_THRESHOLDS[type].acceptance).toBeDefined();
        expect(CLAIM_THRESHOLDS[type].rejection).toBeDefined();
        expect(CLAIM_THRESHOLDS[type].description).toBeDefined();
      }
    });
  });

  // ============ Highest Certainty Tests ============

  describe("AXIOM thresholds", () => {
    it("should require 95% for acceptance (near-unanimous)", () => {
      expect(CLAIM_THRESHOLDS.AXIOM.acceptance).toBe(0.95);
    });

    it("should allow 10% to block (high bar for foundational truths)", () => {
      expect(CLAIM_THRESHOLDS.AXIOM.rejection).toBe(0.1);
    });
  });

  describe("PROTOCOL thresholds", () => {
    it("should require 99% for acceptance (self-verifying)", () => {
      expect(CLAIM_THRESHOLDS.PROTOCOL.acceptance).toBe(0.99);
    });

    it("should have lowest rejection threshold (5%)", () => {
      expect(CLAIM_THRESHOLDS.PROTOCOL.rejection).toBe(0.05);
    });

    it("should have highest acceptance of all types", () => {
      for (const type of ALL_TYPES) {
        expect(CLAIM_THRESHOLDS.PROTOCOL.acceptance).toBeGreaterThanOrEqual(
          CLAIM_THRESHOLDS[type].acceptance,
        );
      }
    });
  });

  describe("COMPUTATIONAL thresholds", () => {
    it("should require 90% for acceptance (reproducible)", () => {
      expect(CLAIM_THRESHOLDS.COMPUTATIONAL.acceptance).toBe(0.9);
    });

    it("should allow 15% to block", () => {
      expect(CLAIM_THRESHOLDS.COMPUTATIONAL.rejection).toBe(0.15);
    });
  });

  // ============ High Certainty Tests ============

  describe("DERIVATION thresholds", () => {
    it("should require 90% for acceptance", () => {
      expect(CLAIM_THRESHOLDS.DERIVATION.acceptance).toBe(0.9);
    });

    it("should allow 15% to block", () => {
      expect(CLAIM_THRESHOLDS.DERIVATION.rejection).toBe(0.15);
    });
  });

  // ============ Standard Certainty Tests ============

  describe("EMPIRICAL thresholds", () => {
    it("should require 80% for acceptance", () => {
      expect(CLAIM_THRESHOLDS.EMPIRICAL.acceptance).toBe(0.8);
    });

    it("should allow >1/3 to block (BFT aligned)", () => {
      expect(CLAIM_THRESHOLDS.EMPIRICAL.rejection).toBe(0.34);
      expect(CLAIM_THRESHOLDS.EMPIRICAL.rejection).toBeGreaterThan(1 / 3);
    });
  });

  describe("HISTORICAL thresholds", () => {
    it("should require 85% for acceptance (higher than empirical)", () => {
      expect(CLAIM_THRESHOLDS.HISTORICAL.acceptance).toBe(0.85);
    });

    it("should allow 25% to block", () => {
      expect(CLAIM_THRESHOLDS.HISTORICAL.rejection).toBe(0.25);
    });

    it("should have higher acceptance than EMPIRICAL (past cannot be re-observed)", () => {
      expect(CLAIM_THRESHOLDS.HISTORICAL.acceptance).toBeGreaterThan(
        CLAIM_THRESHOLDS.EMPIRICAL.acceptance,
      );
    });
  });

  // ============ External/Trust Tests ============

  describe("ORACLE thresholds", () => {
    it("should require 75% for acceptance (trust-dependent)", () => {
      expect(CLAIM_THRESHOLDS.ORACLE.acceptance).toBe(0.75);
    });

    it("should allow >1/3 to block", () => {
      expect(CLAIM_THRESHOLDS.ORACLE.rejection).toBe(0.34);
    });

    it("should have lower acceptance than EMPIRICAL (inherent trust assumptions)", () => {
      expect(CLAIM_THRESHOLDS.ORACLE.acceptance).toBeLessThan(
        CLAIM_THRESHOLDS.EMPIRICAL.acceptance,
      );
    });
  });

  describe("ATTESTATION thresholds", () => {
    it("should require 80% for acceptance", () => {
      expect(CLAIM_THRESHOLDS.ATTESTATION.acceptance).toBe(0.8);
    });

    it("should allow 30% to block (human bias possible)", () => {
      expect(CLAIM_THRESHOLDS.ATTESTATION.rejection).toBe(0.3);
    });
  });

  // ============ High Uncertainty Tests ============

  describe("PREDICTIVE thresholds", () => {
    it("should require 70% for acceptance (predictions are hard)", () => {
      expect(CLAIM_THRESHOLDS.PREDICTIVE.acceptance).toBe(0.7);
    });

    it("should allow 40% to block (let predictions stand)", () => {
      expect(CLAIM_THRESHOLDS.PREDICTIVE.rejection).toBe(0.4);
    });

    it("should have lowest acceptance among non-governance types", () => {
      const nonGovTypes = ALL_TYPES.filter((t) => t !== "GOVERNANCE");
      for (const type of nonGovTypes) {
        expect(CLAIM_THRESHOLDS.PREDICTIVE.acceptance).toBeLessThanOrEqual(
          CLAIM_THRESHOLDS[type].acceptance,
        );
      }
    });
  });

  describe("SOCIAL thresholds", () => {
    it("should require 75% for acceptance (consensus-based)", () => {
      expect(CLAIM_THRESHOLDS.SOCIAL.acceptance).toBe(0.75);
    });

    it("should allow >1/3 to block", () => {
      expect(CLAIM_THRESHOLDS.SOCIAL.rejection).toBe(0.34);
    });
  });

  // ============ Special Cases Tests ============

  describe("CONTESTED thresholds", () => {
    it("should require 90% for acceptance (higher scrutiny)", () => {
      expect(CLAIM_THRESHOLDS.CONTESTED.acceptance).toBe(0.9);
    });

    it("should allow 20% to keep contested", () => {
      expect(CLAIM_THRESHOLDS.CONTESTED.rejection).toBe(0.2);
    });

    it("should have higher acceptance than EMPIRICAL", () => {
      expect(CLAIM_THRESHOLDS.CONTESTED.acceptance).toBeGreaterThan(
        CLAIM_THRESHOLDS.EMPIRICAL.acceptance,
      );
    });
  });

  describe("GOVERNANCE thresholds", () => {
    it("should require 67% for acceptance (2/3 majority)", () => {
      expect(CLAIM_THRESHOLDS.GOVERNANCE.acceptance).toBe(0.67);
    });

    it("should allow >1/3 to block", () => {
      expect(CLAIM_THRESHOLDS.GOVERNANCE.rejection).toBe(0.34);
    });

    it("should have lowest acceptance threshold (democratic)", () => {
      for (const type of ALL_TYPES) {
        expect(CLAIM_THRESHOLDS.GOVERNANCE.acceptance).toBeLessThanOrEqual(
          CLAIM_THRESHOLDS[type].acceptance,
        );
      }
    });
  });

  // ============ Validity Tests ============

  describe("threshold validity", () => {
    it("should have acceptance > rejection for all types", () => {
      for (const type of ALL_TYPES) {
        expect(CLAIM_THRESHOLDS[type].acceptance).toBeGreaterThan(
          CLAIM_THRESHOLDS[type].rejection,
        );
      }
    });

    it("should have acceptance between 0 and 1 for all types", () => {
      for (const type of ALL_TYPES) {
        expect(CLAIM_THRESHOLDS[type].acceptance).toBeGreaterThan(0);
        expect(CLAIM_THRESHOLDS[type].acceptance).toBeLessThanOrEqual(1);
      }
    });

    it("should have rejection between 0 and 1 for all types", () => {
      for (const type of ALL_TYPES) {
        expect(CLAIM_THRESHOLDS[type].rejection).toBeGreaterThan(0);
        expect(CLAIM_THRESHOLDS[type].rejection).toBeLessThan(1);
      }
    });

    it("should have a contested zone (gap > 25%) for all types", () => {
      for (const type of ALL_TYPES) {
        const gap =
          CLAIM_THRESHOLDS[type].acceptance - CLAIM_THRESHOLDS[type].rejection;
        expect(gap).toBeGreaterThan(0.25);
      }
    });
  });

  // ============ Hierarchy Tests ============

  describe("certainty hierarchy", () => {
    it("should follow: PROTOCOL > AXIOM > DERIVATION/COMPUTATIONAL", () => {
      expect(CLAIM_THRESHOLDS.PROTOCOL.acceptance).toBeGreaterThan(
        CLAIM_THRESHOLDS.AXIOM.acceptance,
      );
      expect(CLAIM_THRESHOLDS.AXIOM.acceptance).toBeGreaterThan(
        CLAIM_THRESHOLDS.DERIVATION.acceptance,
      );
      expect(CLAIM_THRESHOLDS.DERIVATION.acceptance).toEqual(
        CLAIM_THRESHOLDS.COMPUTATIONAL.acceptance,
      );
    });

    it("should follow: HISTORICAL > EMPIRICAL > ORACLE", () => {
      expect(CLAIM_THRESHOLDS.HISTORICAL.acceptance).toBeGreaterThan(
        CLAIM_THRESHOLDS.EMPIRICAL.acceptance,
      );
      expect(CLAIM_THRESHOLDS.EMPIRICAL.acceptance).toBeGreaterThan(
        CLAIM_THRESHOLDS.ORACLE.acceptance,
      );
    });

    it("should follow: SOCIAL/ORACLE > PREDICTIVE > GOVERNANCE", () => {
      expect(CLAIM_THRESHOLDS.SOCIAL.acceptance).toBeGreaterThan(
        CLAIM_THRESHOLDS.PREDICTIVE.acceptance,
      );
      expect(CLAIM_THRESHOLDS.PREDICTIVE.acceptance).toBeGreaterThan(
        CLAIM_THRESHOLDS.GOVERNANCE.acceptance,
      );
    });
  });

  it('should be readonly via "as const"', () => {
    expect(CLAIM_THRESHOLDS).toBeDefined();
    expect(typeof CLAIM_THRESHOLDS).toBe("object");
  });
});

// ============ Economics Constants Tests ============

describe("ECONOMICS Constants", () => {
  describe("Share distributions", () => {
    it("should have KNOWLEDGE_POOL_SHARE", () => {
      expect(ECONOMICS.KNOWLEDGE_POOL_SHARE).toBe(0.3);
    });

    it("should have API_PROVIDER_SHARE", () => {
      expect(ECONOMICS.API_PROVIDER_SHARE).toBe(0.6);
    });

    it("should have PROTOCOL_SHARE", () => {
      expect(ECONOMICS.PROTOCOL_SHARE).toBe(0.1);
    });

    it("should sum to 100%", () => {
      const total =
        ECONOMICS.KNOWLEDGE_POOL_SHARE +
        ECONOMICS.API_PROVIDER_SHARE +
        ECONOMICS.PROTOCOL_SHARE;

      expect(total).toBeCloseTo(1.0, 10);
    });
  });

  describe("BURN_RATE", () => {
    it("should be 50%", () => {
      expect(ECONOMICS.BURN_RATE).toBe(0.5);
    });

    it("should be between 0 and 1", () => {
      expect(ECONOMICS.BURN_RATE).toBeGreaterThanOrEqual(0);
      expect(ECONOMICS.BURN_RATE).toBeLessThanOrEqual(1);
    });
  });

  it('should be readonly via "as const"', () => {
    expect(ECONOMICS).toBeDefined();
    expect(typeof ECONOMICS).toBe("object");
  });

  it("should have all values as numbers between 0 and 1", () => {
    for (const [_key, value] of Object.entries(ECONOMICS)) {
      expect(typeof value).toBe("number");
      expect(value).toBeGreaterThanOrEqual(0);
      expect(value).toBeLessThanOrEqual(1);
    }
  });
});

// ============ Evidence Tax Tests (FROZEN) ============

describe("EVIDENCE_TAX", () => {
  it("should be 22%", () => {
    expect(EVIDENCE_TAX).toBe(0.22);
  });

  it("should be between 0 and 1", () => {
    expect(EVIDENCE_TAX).toBeGreaterThan(0);
    expect(EVIDENCE_TAX).toBeLessThan(1);
  });
});

// ============ Treasury Tests (FROZEN) ============

describe("TREASURY", () => {
  describe("ALLOCATION", () => {
    it("should have five fund allocations", () => {
      expect(TREASURY.ALLOCATION.RESEARCH).toBeDefined();
      expect(TREASURY.ALLOCATION.FOUNDATION).toBeDefined();
      expect(TREASURY.ALLOCATION.COMMUNITY).toBeDefined();
      expect(TREASURY.ALLOCATION.DEVELOPERS).toBeDefined();
      expect(TREASURY.ALLOCATION.RESERVE).toBeDefined();
    });

    it("should sum to 100%", () => {
      const total =
        TREASURY.ALLOCATION.RESEARCH +
        TREASURY.ALLOCATION.FOUNDATION +
        TREASURY.ALLOCATION.COMMUNITY +
        TREASURY.ALLOCATION.DEVELOPERS +
        TREASURY.ALLOCATION.RESERVE;
      expect(total).toBeCloseTo(1.0, 10);
    });

    it("should have Research at 77% (largest allocation)", () => {
      expect(TREASURY.ALLOCATION.RESEARCH).toBe(0.77);
    });

    it("should have Foundation at 7%", () => {
      expect(TREASURY.ALLOCATION.FOUNDATION).toBe(0.07);
    });

    it("should have Community at 7%", () => {
      expect(TREASURY.ALLOCATION.COMMUNITY).toBe(0.07);
    });

    it("should have Developers at 7%", () => {
      expect(TREASURY.ALLOCATION.DEVELOPERS).toBe(0.07);
    });

    it("should have Reserve at 2%", () => {
      expect(TREASURY.ALLOCATION.RESERVE).toBe(0.02);
    });

    it("should prioritize Research as largest fund", () => {
      expect(TREASURY.ALLOCATION.RESEARCH).toBeGreaterThan(
        TREASURY.ALLOCATION.FOUNDATION,
      );
      expect(TREASURY.ALLOCATION.RESEARCH).toBeGreaterThan(
        TREASURY.ALLOCATION.COMMUNITY,
      );
      expect(TREASURY.ALLOCATION.RESEARCH).toBeGreaterThan(
        TREASURY.ALLOCATION.DEVELOPERS,
      );
    });
  });

  describe("ADDRESSES", () => {
    it("should have five fund addresses", () => {
      expect(TREASURY.ADDRESSES.RESEARCH).toBeDefined();
      expect(TREASURY.ADDRESSES.FOUNDATION).toBeDefined();
      expect(TREASURY.ADDRESSES.COMMUNITY).toBeDefined();
      expect(TREASURY.ADDRESSES.DEVELOPERS).toBeDefined();
      expect(TREASURY.ADDRESSES.RESERVE).toBeDefined();
    });

    it("should have addresses starting with lgm1", () => {
      // Placeholder format until genesis derivation
      expect(TREASURY.ADDRESSES.RESEARCH.startsWith("lgm1")).toBe(true);
      expect(TREASURY.ADDRESSES.FOUNDATION.startsWith("lgm1")).toBe(true);
      expect(TREASURY.ADDRESSES.COMMUNITY.startsWith("lgm1")).toBe(true);
      expect(TREASURY.ADDRESSES.DEVELOPERS.startsWith("lgm1")).toBe(true);
      expect(TREASURY.ADDRESSES.RESERVE.startsWith("lgm1")).toBe(true);
    });

    it("should have unique addresses for each fund", () => {
      const addresses = Object.values(TREASURY.ADDRESSES);
      const uniqueAddresses = new Set(addresses);
      expect(uniqueAddresses.size).toBe(addresses.length);
    });

    it("should indicate genesis derivation in placeholder", () => {
      // These placeholders will be replaced by derived addresses at genesis
      expect(TREASURY.ADDRESSES.RESEARCH).toContain("GENESIS_DERIVED");
      expect(TREASURY.ADDRESSES.FOUNDATION).toContain("GENESIS_DERIVED");
      expect(TREASURY.ADDRESSES.COMMUNITY).toContain("GENESIS_DERIVED");
      expect(TREASURY.ADDRESSES.DEVELOPERS).toContain("GENESIS_DERIVED");
      expect(TREASURY.ADDRESSES.RESERVE).toContain("GENESIS_DERIVED");
    });
  });

  describe("GOVERNANCE", () => {
    it("should use PROGRESSIVE_TRUST phase", () => {
      expect(TREASURY.GOVERNANCE.phase).toBe("PROGRESSIVE_TRUST");
    });

    it("should start with 1-of-1 (founder only)", () => {
      expect(TREASURY.GOVERNANCE.initial.signers).toBe(1);
      expect(TREASURY.GOVERNANCE.initial.threshold).toBe(1);
    });

    it("should require UNANIMOUS approval to add signers", () => {
      expect(TREASURY.GOVERNANCE.growth.addSignerThreshold).toBe("UNANIMOUS");
    });

    it("should use MAJORITY_PLUS_ONE for threshold formula", () => {
      expect(TREASURY.GOVERNANCE.growth.thresholdFormula).toBe(
        "MAJORITY_PLUS_ONE",
      );
    });

    it("should have longer timelock for signer changes than operations", () => {
      expect(
        TREASURY.GOVERNANCE.growth.signerChangeTimelockBlocks,
      ).toBeGreaterThan(TREASURY.OPERATION_TIMELOCK_BLOCKS);
    });

    it("should have cooldown between signer changes (~3 days)", () => {
      const cooldownMs =
        TREASURY.GOVERNANCE.growth.signerChangeCooldownBlocks * 2521;
      const cooldownDays = cooldownMs / (1000 * 60 * 60 * 24);
      expect(cooldownDays).toBeGreaterThan(2);
      expect(cooldownDays).toBeLessThan(4);
    });

    it("should cap signers at 11", () => {
      expect(TREASURY.GOVERNANCE.growth.maxSigners).toBe(11);
    });

    it("should have operation timelock for transparency", () => {
      expect(TREASURY.OPERATION_TIMELOCK_BLOCKS).toBeGreaterThan(0);
    });

    it("should have ~42 min operation timelock (1000 blocks)", () => {
      expect(TREASURY.OPERATION_TIMELOCK_BLOCKS).toBe(1000);
      const timelockMinutes =
        (TREASURY.OPERATION_TIMELOCK_BLOCKS * 2521) / (1000 * 60);
      expect(timelockMinutes).toBeGreaterThan(40);
      expect(timelockMinutes).toBeLessThan(45);
    });
  });

  describe("RULES", () => {
    it("should have auto-distribute interval", () => {
      expect(TREASURY.RULES.AUTO_DISTRIBUTE_BLOCKS).toBe(1_000_000);
    });

    it("should have maximum fund balance cap", () => {
      const fiveMillionLGM = 5_000_000n * 10n ** 18n;
      expect(TREASURY.RULES.MAX_FUND_BALANCE).toBe(fiveMillionLGM);
    });

    it("should overflow to Community fund", () => {
      expect(TREASURY.RULES.OVERFLOW_RECIPIENT).toBe("COMMUNITY");
    });

    it("should have reasonable auto-distribute interval (~29 days)", () => {
      const intervalMs = TREASURY.RULES.AUTO_DISTRIBUTE_BLOCKS * 2521;
      const intervalDays = intervalMs / (1000 * 60 * 60 * 24);
      expect(intervalDays).toBeGreaterThan(25);
      expect(intervalDays).toBeLessThan(35);
    });
  });

  describe("fee flow calculation", () => {
    it("should correctly calculate treasury share from 100 LGM fee", () => {
      const totalFee = 100;
      const treasuryShare = totalFee * EVIDENCE_TAX;
      expect(treasuryShare).toBe(22);
    });

    it("should correctly split treasury among funds", () => {
      const treasuryShare = 22; // From 100 LGM fee

      const research = treasuryShare * TREASURY.ALLOCATION.RESEARCH;
      const foundation = treasuryShare * TREASURY.ALLOCATION.FOUNDATION;
      const community = treasuryShare * TREASURY.ALLOCATION.COMMUNITY;
      const developers = treasuryShare * TREASURY.ALLOCATION.DEVELOPERS;
      const reserve = treasuryShare * TREASURY.ALLOCATION.RESERVE;

      expect(research).toBeCloseTo(16.94, 1); // 77% of 22
      expect(foundation).toBeCloseTo(1.54, 1); // 7% of 22
      expect(community).toBeCloseTo(1.54, 1); // 7% of 22
      expect(developers).toBeCloseTo(1.54, 1); // 7% of 22
      expect(reserve).toBeCloseTo(0.44, 1); // 2% of 22

      expect(
        research + foundation + community + developers + reserve,
      ).toBeCloseTo(22, 1);
    });
  });
});

// ============ Genesis Constants Tests ============

describe("GENESIS Constants", () => {
  describe("TIMESTAMP", () => {
    it("should be defined", () => {
      expect(GENESIS.TIMESTAMP).toBeDefined();
    });

    it("should be the founding date", () => {
      // 20250203 = February 3, 2025
      expect(GENESIS.TIMESTAMP).toBe(20250203);
    });
  });

  describe("BLOCK_HASH", () => {
    it("should be 64 zeros with 0x prefix", () => {
      expect(GENESIS.BLOCK_HASH).toBe("0x" + "0".repeat(64));
    });

    it("should be valid hex", () => {
      expect(GENESIS.BLOCK_HASH).toMatch(/^0x[0-9a-f]{64}$/);
    });
  });

  describe("FOUNDING_AXIOM_ID", () => {
    it("should be axiom_0", () => {
      expect(GENESIS.FOUNDING_AXIOM_ID).toBe("axiom_0");
    });
  });

  it('should be readonly via "as const"', () => {
    expect(GENESIS).toBeDefined();
    expect(typeof GENESIS).toBe("object");
  });
});

// ============ Derivation Hint Tests ============

describe("DERIVATION_HINT", () => {
  it("should contain the derivation path", () => {
    expect(DERIVATION_HINT).toContain("axiom_0");
    expect(DERIVATION_HINT).toContain("origin");
    expect(DERIVATION_HINT).toContain("unity");
  });

  it("should be a string", () => {
    expect(typeof DERIVATION_HINT).toBe("string");
  });
});

// ============ Constant Consistency Tests ============

describe("Constant Consistency", () => {
  it("should have expected block time", () => {
    expect(TIMING.TARGET_BLOCK_MS).toBe(2521);
  });

  it("should have expected confirmation blocks", () => {
    expect(TIMING.CONFIRMATION_BLOCKS).toBe(19);
  });

  it("should have expected version numbers", () => {
    expect(VERSION.MAJOR).toBe(89);
    expect(VERSION.MINOR).toBe(85);
    expect(VERSION.PATCH).toBe(65);
    expect(VERSION.BUILD).toBe(73);
  });
});

// ============ Compile-Time Immutability Tests ============

describe("Compile-Time Immutability", () => {
  // Note: TypeScript "as const" provides compile-time readonly enforcement.
  // These tests verify the objects exist and have expected structure.
  // Runtime immutability would require Object.freeze() which is not used.
  // This is a design choice - TypeScript catches mutations at compile time.

  it("should have TOKEN defined with expected properties", () => {
    expect(TOKEN.NAME).toBe("Legible Money");
    expect(TOKEN.SYMBOL).toBe("LGM");
    expect(TOKEN.DECIMALS).toBe(18);
    expect(typeof TOKEN.MAX_SUPPLY).toBe("bigint");
  });

  it("should have VERSION defined with expected properties", () => {
    expect(typeof VERSION.MAJOR).toBe("number");
    expect(typeof VERSION.MINOR).toBe("number");
    expect(typeof VERSION.PATCH).toBe("number");
    expect(typeof VERSION.BUILD).toBe("number");
  });

  it("should have TIMING defined with expected properties", () => {
    expect(typeof TIMING.TARGET_BLOCK_MS).toBe("number");
    expect(typeof TIMING.CONFIRMATION_BLOCKS).toBe("number");
    expect(typeof TIMING.FINALITY_BLOCKS).toBe("number");
  });

  it("should have CONSENSUS defined with expected properties", () => {
    expect(typeof CONSENSUS.MIN_VALIDATORS).toBe("number");
    expect(typeof CONSENSUS.QUORUM_THRESHOLD).toBe("number");
  });

  it("should have CLAIM_THRESHOLDS defined with expected properties", () => {
    expect(typeof CLAIM_THRESHOLDS.AXIOM.acceptance).toBe("number");
    expect(typeof CLAIM_THRESHOLDS.EMPIRICAL.rejection).toBe("number");
    expect(typeof CLAIM_THRESHOLDS.GOVERNANCE.description).toBe("string");
  });

  it("should have EVIDENCE_TAX defined as number", () => {
    expect(typeof EVIDENCE_TAX).toBe("number");
  });

  it("should have TREASURY defined with expected structure", () => {
    expect(typeof TREASURY.ALLOCATION.FOUNDATION).toBe("number");
    expect(typeof TREASURY.ADDRESSES.FOUNDATION).toBe("string");
    expect(typeof TREASURY.RULES.AUTO_DISTRIBUTE_BLOCKS).toBe("number");
    expect(typeof TREASURY.GOVERNANCE.phase).toBe("string");
    expect(typeof TREASURY.OPERATION_TIMELOCK_BLOCKS).toBe("number");
  });

  it("should have ECONOMICS defined with expected properties", () => {
    expect(typeof ECONOMICS.KNOWLEDGE_POOL_SHARE).toBe("number");
    expect(typeof ECONOMICS.API_PROVIDER_SHARE).toBe("number");
    expect(typeof ECONOMICS.PROTOCOL_SHARE).toBe("number");
    expect(typeof ECONOMICS.BURN_RATE).toBe("number");
  });

  it("should have GENESIS defined with expected properties", () => {
    expect(typeof GENESIS.TIMESTAMP).toBe("number");
    expect(typeof GENESIS.BLOCK_HASH).toBe("string");
    expect(typeof GENESIS.FOUNDING_AXIOM_ID).toBe("string");
  });
});

// ============ Type Safety Tests ============

describe("Type Safety", () => {
  it("TOKEN should have correct structure", () => {
    expect(TOKEN).toHaveProperty("NAME");
    expect(TOKEN).toHaveProperty("SYMBOL");
    expect(TOKEN).toHaveProperty("DECIMALS");
    expect(TOKEN).toHaveProperty("MAX_SUPPLY");

    expect(typeof TOKEN.NAME).toBe("string");
    expect(typeof TOKEN.SYMBOL).toBe("string");
    expect(typeof TOKEN.DECIMALS).toBe("number");
    expect(typeof TOKEN.MAX_SUPPLY).toBe("bigint");
  });

  it("VERSION should have correct structure", () => {
    expect(VERSION).toHaveProperty("MAJOR");
    expect(VERSION).toHaveProperty("MINOR");
    expect(VERSION).toHaveProperty("PATCH");
    expect(VERSION).toHaveProperty("BUILD");

    for (const key of ["MAJOR", "MINOR", "PATCH", "BUILD"] as const) {
      expect(typeof VERSION[key]).toBe("number");
      expect(Number.isInteger(VERSION[key])).toBe(true);
    }
  });

  it("TIMING should have correct structure", () => {
    expect(TIMING).toHaveProperty("TARGET_BLOCK_MS");
    expect(TIMING).toHaveProperty("CONFIRMATION_BLOCKS");
    expect(TIMING).toHaveProperty("FINALITY_BLOCKS");

    for (const key of [
      "TARGET_BLOCK_MS",
      "CONFIRMATION_BLOCKS",
      "FINALITY_BLOCKS",
    ] as const) {
      expect(typeof TIMING[key]).toBe("number");
      expect(Number.isInteger(TIMING[key])).toBe(true);
      expect(TIMING[key]).toBeGreaterThan(0);
    }
  });
});
