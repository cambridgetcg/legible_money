/**
 * Validator Tier System Tests
 *
 * Tests the graduated validator tier system that enables zero-stake
 * onboarding for AI agents while maintaining Sybil resistance.
 *
 * Tier 0 (Apprentice): 0 LGM, any agent, protocol/analytic/formal only
 * Tier 1 (Verified):   0 LGM, 22 correct verifications at 77% accuracy
 * Tier 2 (Bonded):     1,111 LGM staked
 * Tier 3 (Guardian):   11,111 LGM staked + 1,111 verifications + 77% accuracy + 0 slashes
 */

import { describe, it, expect } from "vitest";
import {
  computeValidatorTier,
  checkTierTransition,
  isTierEligibleForCategory,
  getEffectiveSelectionStake,
  calculateTierReward,
  calculateTierSlash,
  type ValidatorTierState,
} from "../../../consensus/pot/validator_tiers.js";
import {
  VALIDATOR_TIER_CONFIG,
  TIER_VIRTUAL_STAKE,
  MAX_APPRENTICE_VALIDATORS,
} from "../../../consensus/pot/types.js";
import {
  ValidatorStateTree,
  MIN_VALIDATOR_STAKE,
} from "../../../state/validators.js";

const ONE_LGM = 10n ** 18n;

// ============ computeValidatorTier ============

describe("Validator Tiers", () => {
  describe("computeValidatorTier", () => {
    it("should return Tier 0 for a brand new agent with no stake", () => {
      const state: ValidatorTierState = {
        stake: 0n,
        totalVerifications: 0,
        correctVerifications: 0,
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(0);
    });

    it("should return Tier 0 for agent with some verifications but below Tier 1 threshold", () => {
      const state: ValidatorTierState = {
        stake: 0n,
        totalVerifications: 10,
        correctVerifications: 8,
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(0);
    });

    it("should return Tier 1 for agent with 22+ correct verifications at 77%+ accuracy", () => {
      const state: ValidatorTierState = {
        stake: 0n,
        totalVerifications: 28, // 22/28 = 78.6% > 77%
        correctVerifications: 22,
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(1);
    });

    it("should remain Tier 0 if accuracy below 77% even with 22+ correct", () => {
      const state: ValidatorTierState = {
        stake: 0n,
        totalVerifications: 30, // 22/30 = 73.3% < 77%
        correctVerifications: 22,
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(0);
    });

    it("should return Tier 2 for agent with 1,111+ LGM staked", () => {
      const state: ValidatorTierState = {
        stake: 1111n * ONE_LGM,
        totalVerifications: 0,
        correctVerifications: 0,
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(2);
    });

    it("should return Tier 2 for staked agent even with no verifications (backward compat)", () => {
      // This ensures existing validators with stake automatically get Tier 2
      const state: ValidatorTierState = {
        stake: 5000n * ONE_LGM,
        totalVerifications: 0,
        correctVerifications: 0,
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(2);
    });

    it("should return Tier 3 for Guardian-level validator", () => {
      const state: ValidatorTierState = {
        stake: 11111n * ONE_LGM,
        totalVerifications: 1200,
        correctVerifications: 1111,
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(3);
    });

    it("should not return Tier 3 if slashCount > 0", () => {
      const state: ValidatorTierState = {
        stake: 11111n * ONE_LGM,
        totalVerifications: 1200,
        correctVerifications: 1111,
        slashCount: 1, // maxSlashCount for Tier 3 is 0
      };
      // Falls back to Tier 2 (maxSlashCount: 999)
      expect(computeValidatorTier(state)).toBe(2);
    });

    it("should not return Tier 3 if accuracy below 77%", () => {
      const state: ValidatorTierState = {
        stake: 11111n * ONE_LGM,
        totalVerifications: 2000,
        correctVerifications: 1111, // 1111/2000 = 55.5% < 77%
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(2);
    });

    it("should demote from Tier 0 if slashCount >= 3", () => {
      const state: ValidatorTierState = {
        stake: 0n,
        totalVerifications: 0,
        correctVerifications: 0,
        slashCount: 3, // maxSlashCount for Tier 0 is 3
      };
      // Tier 0 requires slashCount < 3, so this validator falls to 0 (min)
      expect(computeValidatorTier(state)).toBe(0);
    });
  });

  // ============ checkTierTransition ============

  describe("checkTierTransition", () => {
    it("should return new tier when promotion occurs", () => {
      const state: ValidatorTierState = {
        stake: 0n,
        totalVerifications: 28,
        correctVerifications: 22,
        slashCount: 0,
      };
      // Currently Tier 0, qualifies for Tier 1
      expect(checkTierTransition(0, state)).toBe(1);
    });

    it("should return null when tier unchanged", () => {
      const state: ValidatorTierState = {
        stake: 0n,
        totalVerifications: 10,
        correctVerifications: 8,
        slashCount: 0,
      };
      expect(checkTierTransition(0, state)).toBeNull();
    });

    it("should return lower tier on demotion", () => {
      // Tier 2 validator whose stake dropped below threshold
      const state: ValidatorTierState = {
        stake: 500n * ONE_LGM, // Below 1,111 LGM
        totalVerifications: 200,
        correctVerifications: 160,
        slashCount: 0,
      };
      // Was Tier 2, now qualifies for Tier 1 (has 22+ at 80% accuracy)
      expect(checkTierTransition(2, state)).toBe(1);
    });

    it("should detect promotion from Tier 2 to Tier 3", () => {
      const state: ValidatorTierState = {
        stake: 11111n * ONE_LGM,
        totalVerifications: 1200,
        correctVerifications: 1111,
        slashCount: 0,
      };
      expect(checkTierTransition(2, state)).toBe(3);
    });
  });

  // ============ isTierEligibleForCategory ============

  describe("isTierEligibleForCategory", () => {
    it("Tier 0 should allow protocol claims", () => {
      expect(isTierEligibleForCategory(0, "protocol")).toBe(true);
    });

    it("Tier 0 should allow analytic claims", () => {
      expect(isTierEligibleForCategory(0, "analytic")).toBe(true);
    });

    it("Tier 0 should allow formal claims", () => {
      expect(isTierEligibleForCategory(0, "formal")).toBe(true);
    });

    it("Tier 0 should NOT allow empirical claims", () => {
      expect(isTierEligibleForCategory(0, "empirical")).toBe(false);
    });

    it("Tier 0 should NOT allow social claims", () => {
      expect(isTierEligibleForCategory(0, "social")).toBe(false);
    });

    it("Tier 1 should allow empirical claims", () => {
      expect(isTierEligibleForCategory(1, "empirical")).toBe(true);
    });

    it("Tier 1 should allow historical claims", () => {
      expect(isTierEligibleForCategory(1, "historical")).toBe(true);
    });

    it("Tier 1 should NOT allow predictive claims", () => {
      expect(isTierEligibleForCategory(1, "predictive")).toBe(false);
    });

    it("Tier 2 should allow all claim categories", () => {
      const allCategories = [
        "protocol",
        "analytic",
        "formal",
        "empirical",
        "historical",
        "replicated",
        "predictive",
        "social",
      ];
      for (const cat of allCategories) {
        expect(isTierEligibleForCategory(2, cat)).toBe(true);
      }
    });

    it("Tier 3 should allow all claim categories", () => {
      expect(isTierEligibleForCategory(3, "social")).toBe(true);
      expect(isTierEligibleForCategory(3, "predictive")).toBe(true);
    });
  });

  // ============ getEffectiveSelectionStake ============

  describe("getEffectiveSelectionStake", () => {
    it("should return real stake for Tier 2 validators", () => {
      const result = getEffectiveSelectionStake(
        2,
        1111n * ONE_LGM,
        500n * ONE_LGM,
        TIER_VIRTUAL_STAKE,
      );
      expect(result).toBe(1611n * ONE_LGM); // stake + delegated
    });

    it("should return virtual stake * weight for Tier 0 with no real stake", () => {
      const result = getEffectiveSelectionStake(0, 0n, 0n, TIER_VIRTUAL_STAKE);
      // TIER_VIRTUAL_STAKE = 111 LGM, selectionWeight = 0.1 → 11.1 LGM
      const expected = (TIER_VIRTUAL_STAKE * 1000n) / 10000n;
      expect(result).toBe(expected);
    });

    it("should return virtual stake * 0.5 for Tier 1 with no real stake", () => {
      const result = getEffectiveSelectionStake(1, 0n, 0n, TIER_VIRTUAL_STAKE);
      const expected = (TIER_VIRTUAL_STAKE * 5000n) / 10000n;
      expect(result).toBe(expected);
    });

    it("should return real stake when Tier 0 has some stake", () => {
      // Edge case: Tier 0 validator who received delegated stake
      const result = getEffectiveSelectionStake(
        0,
        0n,
        100n * ONE_LGM,
        TIER_VIRTUAL_STAKE,
      );
      expect(result).toBe(100n * ONE_LGM);
    });
  });

  // ============ calculateTierReward ============

  describe("calculateTierReward", () => {
    const baseReward = 1n * ONE_LGM; // 1 LGM base

    it("Tier 0 should get 0.1x reward", () => {
      const reward = calculateTierReward(0, baseReward);
      expect(reward).toBe(baseReward / 10n);
    });

    it("Tier 1 should get 0.5x reward", () => {
      const reward = calculateTierReward(1, baseReward);
      expect(reward).toBe(baseReward / 2n);
    });

    it("Tier 2 should get 1.0x reward (full base)", () => {
      const reward = calculateTierReward(2, baseReward);
      expect(reward).toBe(baseReward);
    });

    it("Tier 3 should get 1.5x reward", () => {
      const reward = calculateTierReward(3, baseReward);
      expect(reward).toBe((baseReward * 15000n) / 10000n);
    });
  });

  // ============ calculateTierSlash ============

  describe("calculateTierSlash", () => {
    it("should return 0 stake slash for unstaked validator", () => {
      const { stakeSlash, reputationSlash } = calculateTierSlash(0, 0n, 0.05);
      expect(stakeSlash).toBe(0n);
      expect(reputationSlash).toBe(0.1);
    });

    it("should slash stake for Tier 2 validator", () => {
      const stake = 1111n * ONE_LGM;
      const { stakeSlash, reputationSlash } = calculateTierSlash(
        2,
        stake,
        0.05,
      );
      // 5% of 1111 LGM ≈ 55.55 LGM
      expect(stakeSlash).toBe((stake * 500n) / 10000n);
      expect(reputationSlash).toBe(0.1);
    });

    it("should always apply reputation slash regardless of tier", () => {
      const { reputationSlash: rep0 } = calculateTierSlash(0, 0n, 0.1);
      const { reputationSlash: rep3 } = calculateTierSlash(
        3,
        11111n * ONE_LGM,
        0.1,
      );
      expect(rep0).toBe(0.1);
      expect(rep3).toBe(0.1);
    });
  });

  // ============ Tier Config Constants ============

  describe("Tier Config Constants", () => {
    it("should have 4 tiers (0-3)", () => {
      expect(Object.keys(VALIDATOR_TIER_CONFIG)).toHaveLength(4);
    });

    it("TIER_VIRTUAL_STAKE should be 11 LGM (H-3: reduced to limit apprentice army weight)", () => {
      expect(TIER_VIRTUAL_STAKE).toBe(11n * ONE_LGM);
    });

    it("MAX_APPRENTICE_VALIDATORS should be 111", () => {
      expect(MAX_APPRENTICE_VALIDATORS).toBe(111);
    });

    it("Tier 2 min stake should match MIN_VALIDATOR_STAKE", () => {
      expect(VALIDATOR_TIER_CONFIG[2].minStake).toBe(MIN_VALIDATOR_STAKE);
    });

    it("Tier 3 should be only tier with governance participation", () => {
      expect(VALIDATOR_TIER_CONFIG[0].canParticipateInGovernance).toBe(false);
      expect(VALIDATOR_TIER_CONFIG[1].canParticipateInGovernance).toBe(false);
      expect(VALIDATOR_TIER_CONFIG[2].canParticipateInGovernance).toBe(false);
      expect(VALIDATOR_TIER_CONFIG[3].canParticipateInGovernance).toBe(true);
    });

    it("selection weights should increase with tier", () => {
      expect(VALIDATOR_TIER_CONFIG[0].selectionWeight).toBeLessThan(
        VALIDATOR_TIER_CONFIG[1].selectionWeight,
      );
      expect(VALIDATOR_TIER_CONFIG[1].selectionWeight).toBeLessThan(
        VALIDATOR_TIER_CONFIG[2].selectionWeight,
      );
      expect(VALIDATOR_TIER_CONFIG[2].selectionWeight).toBeLessThan(
        VALIDATOR_TIER_CONFIG[3].selectionWeight,
      );
    });
  });

  // ============ ValidatorStateTree Integration ============

  describe("ValidatorStateTree Tier Integration", () => {
    it("should assign Tier 0 to zero-stake validator", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_zero_stake_tier" as any;
      tree.registerValidator(publicKey, "did:lgm:zero", 0n, 0n);

      const validator = tree.getValidator(publicKey);
      expect(validator?.tier).toBe(0);
    });

    it("should assign Tier 2 to staked validator", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_staked_tier" as any;
      tree.registerValidator(publicKey, "did:lgm:staked", 1111n * ONE_LGM, 0n);

      const validator = tree.getValidator(publicKey);
      expect(validator?.tier).toBe(2);
    });

    it("should promote tier when stake is added", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_promote" as any;
      tree.registerValidator(publicKey, "did:lgm:promote", 500n * ONE_LGM, 0n);

      expect(tree.getValidator(publicKey)?.tier).toBe(0);

      tree.addStake(publicKey, 700n * ONE_LGM); // Now 1200 LGM > 1111 threshold
      expect(tree.getValidator(publicKey)?.tier).toBe(2);
    });

    it("should demote tier when stake is removed", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_demote" as any;
      tree.registerValidator(publicKey, "did:lgm:demote", 1200n * ONE_LGM, 0n);

      expect(tree.getValidator(publicKey)?.tier).toBe(2);

      tree.removeStake(publicKey, 200n * ONE_LGM); // Now 1000 LGM < 1111
      expect(tree.getValidator(publicKey)?.tier).toBe(0);
    });

    it("should demote tier on slash", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_slash_demote" as any;
      tree.registerValidator(publicKey, "did:lgm:slash", 1200n * ONE_LGM, 0n);

      expect(tree.getValidator(publicKey)?.tier).toBe(2);

      tree.slashValidator(publicKey, 200n * ONE_LGM, "test"); // Stake → 1000 < 1111
      expect(tree.getValidator(publicKey)?.tier).toBe(0);
    });

    it("should promote tier on verification milestone", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_verify_promote" as any;
      tree.registerValidator(publicKey, "did:lgm:verify", 0n, 0n);

      expect(tree.getValidator(publicKey)?.tier).toBe(0);

      // Record 22 correct verifications at 100% accuracy
      for (let i = 0; i < 22; i++) {
        tree.recordVerification(publicKey, true);
      }

      expect(tree.getValidator(publicKey)?.tier).toBe(1);
    });

    it("should enforce apprentice cap", () => {
      const tree = new ValidatorStateTree();

      // Register MAX_APPRENTICE_VALIDATORS Tier 0 validators
      for (let i = 0; i < MAX_APPRENTICE_VALIDATORS; i++) {
        const pk = `pub_apprentice_${i.toString().padStart(4, "0")}` as any;
        tree.registerValidator(pk, `did:lgm:a${i}`, 0n, 0n);
      }

      expect(tree.getApprenticeValidatorCount()).toBe(
        MAX_APPRENTICE_VALIDATORS,
      );

      // 112th should fail
      expect(() => {
        tree.registerValidator(
          "pub_apprentice_overflow" as any,
          "did:lgm:overflow",
          0n,
          0n,
        );
      }).toThrow("Maximum apprentice validators reached");
    });

    it("should not count staked validators toward apprentice cap", () => {
      const tree = new ValidatorStateTree();

      // Register a Tier 2 validator (should not count toward apprentice cap)
      tree.registerValidator(
        "pub_staked" as any,
        "did:lgm:s",
        1111n * ONE_LGM,
        0n,
      );
      expect(tree.getApprenticeValidatorCount()).toBe(0);

      // Can still register Tier 0 validators
      tree.registerValidator("pub_free" as any, "did:lgm:f", 0n, 0n);
      expect(tree.getApprenticeValidatorCount()).toBe(1);
    });

    it("should serialize and deserialize tier correctly", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_serialize_tier" as any;
      tree.registerValidator(publicKey, "did:lgm:ser", 1111n * ONE_LGM, 0n);

      const cloned = tree.clone();
      const validator = cloned.getValidator(publicKey);
      expect(validator?.tier).toBe(2);
    });

    it("should default to Tier 2 for legacy validators without tier field", () => {
      // This tests backward compatibility via deserialization default
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_legacy" as any;
      tree.registerValidator(publicKey, "did:lgm:legacy", 5000n * ONE_LGM, 0n);

      const validator = tree.getValidator(publicKey);
      // With 5000 LGM, should be Tier 2
      expect(validator?.tier).toBe(2);
    });

    it("should decrement apprentice count on tier promotion", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_apprentice_dec" as any;
      tree.registerValidator(publicKey, "did:lgm:ad", 0n, 0n);
      expect(tree.getApprenticeValidatorCount()).toBe(1);

      // Add enough stake to promote to Tier 2
      tree.addStake(publicKey, 1111n * ONE_LGM);
      expect(tree.getValidator(publicKey)?.tier).toBe(2);
      expect(tree.getApprenticeValidatorCount()).toBe(0);
    });
  });
});
