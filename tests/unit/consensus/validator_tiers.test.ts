/**
 * Validator Tier System Tests
 *
 * Tests the graduated validator tier system that enables zero-stake
 * onboarding for AI agents while maintaining Sybil resistance.
 *
 * Tier 0 (Apprentice): 0.111 LGM, any agent, protocol/computational/formal only
 * Tier 1 (Verified):   1.11 LGM, 22 correct verifications at 77% accuracy
 * Tier 2 (Bonded):     11.1 LGM staked (stake alone qualifies)
 * Tier 3 (Guardian):   111 LGM staked + 333 verifications (33 contested) + 77% accuracy
 *                       + 0 slashes in rolling 10-epoch window
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
        stake: 111n * 10n ** 16n, // 1.11 LGM (tier 1 minimum)
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

    it("should return Tier 2 for agent with 11.1+ LGM staked", () => {
      const state: ValidatorTierState = {
        stake: 111n * 10n ** 17n, // 11.1 LGM
        totalVerifications: 0,
        correctVerifications: 0,
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(2);
    });

    it("should return Tier 2 for staked agent even with no verifications (backward compat)", () => {
      const state: ValidatorTierState = {
        stake: 50n * ONE_LGM, // 50 LGM (well above 11.1 LGM tier 2 threshold)
        totalVerifications: 0,
        correctVerifications: 0,
        slashCount: 0,
      };
      expect(computeValidatorTier(state)).toBe(2);
    });

    it("should return Tier 3 for Guardian-level validator with contested verifications", () => {
      const state: ValidatorTierState = {
        stake: 111n * ONE_LGM, // 111 LGM
        totalVerifications: 400,
        correctVerifications: 350,
        slashCount: 0,
        contestedVerificationsCorrect: 33, // meets minContestedVerifications
      };
      // effectiveVerifications = (400 - 33) + 33 * 3 = 367 + 99 = 466 >= 333
      expect(computeValidatorTier(state)).toBe(3);
    });

    it("should not return Tier 3 without contested verifications", () => {
      const state: ValidatorTierState = {
        stake: 111n * ONE_LGM, // 111 LGM
        totalVerifications: 400,
        correctVerifications: 350,
        slashCount: 0,
        contestedVerificationsCorrect: 0, // no contested verifications
      };
      // Fails minContestedVerifications = 33
      expect(computeValidatorTier(state)).toBe(2);
    });

    it("should not return Tier 3 if slashCount > 0 within window", () => {
      const state: ValidatorTierState = {
        stake: 111n * ONE_LGM,
        totalVerifications: 400,
        correctVerifications: 350,
        slashCount: 1,
        contestedVerificationsCorrect: 33,
        lastSlashBlock: 9000n,
        currentBlock: 10000n, // within 10 epochs (11110 blocks)
        epochLength: 1111n,
      };
      // Falls back to Tier 2 (slash within window)
      expect(computeValidatorTier(state)).toBe(2);
    });

    it("should return Tier 3 if slash is outside the 10-epoch window", () => {
      const state: ValidatorTierState = {
        stake: 111n * ONE_LGM,
        totalVerifications: 400,
        correctVerifications: 350,
        slashCount: 1,
        contestedVerificationsCorrect: 33,
        lastSlashBlock: 1000n,
        currentBlock: 20000n, // 19000 blocks ago > 11110 window
        epochLength: 1111n,
      };
      // Slash is outside window, so windowed slash count = 0
      expect(computeValidatorTier(state)).toBe(3);
    });

    it("should not return Tier 3 if accuracy below 77%", () => {
      const state: ValidatorTierState = {
        stake: 111n * ONE_LGM,
        totalVerifications: 2000,
        correctVerifications: 1111, // 55.5% < 77%
        slashCount: 0,
        contestedVerificationsCorrect: 33,
      };
      expect(computeValidatorTier(state)).toBe(2);
    });

    it("Tier 0 always accepts (even high slash counts)", () => {
      const state: ValidatorTierState = {
        stake: 0n,
        totalVerifications: 0,
        correctVerifications: 0,
        slashCount: 100, // Tier 0 has maxSlashCount: Infinity
      };
      expect(computeValidatorTier(state)).toBe(0);
    });

    it("contested verification multiplier boosts effective count for Guardian", () => {
      // 100 total verifications, 50 contested correct
      // effective = (100 - 50) + 50 * 3 = 50 + 150 = 200 < 333
      const state: ValidatorTierState = {
        stake: 111n * ONE_LGM,
        totalVerifications: 100,
        correctVerifications: 90,
        slashCount: 0,
        contestedVerificationsCorrect: 50,
      };
      expect(computeValidatorTier(state)).toBe(2); // Not enough even with multiplier

      // 200 total, 80 contested
      // effective = (200 - 80) + 80 * 3 = 120 + 240 = 360 >= 333
      const state2: ValidatorTierState = {
        stake: 111n * ONE_LGM,
        totalVerifications: 200,
        correctVerifications: 170,
        slashCount: 0,
        contestedVerificationsCorrect: 80,
      };
      expect(computeValidatorTier(state2)).toBe(3);
    });
  });

  // ============ checkTierTransition ============

  describe("checkTierTransition", () => {
    it("should return new tier when promotion occurs", () => {
      const state: ValidatorTierState = {
        stake: 111n * 10n ** 16n, // 1.11 LGM (tier 1 minimum)
        totalVerifications: 28,
        correctVerifications: 22,
        slashCount: 0,
      };
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
      const state: ValidatorTierState = {
        stake: 5n * ONE_LGM, // Below 11.1 LGM (tier 2), above 1.11 LGM (tier 1)
        totalVerifications: 200,
        correctVerifications: 160,
        slashCount: 0,
      };
      expect(checkTierTransition(2, state)).toBe(1);
    });

    it("should detect promotion from Tier 2 to Tier 3", () => {
      const state: ValidatorTierState = {
        stake: 111n * ONE_LGM,
        totalVerifications: 400,
        correctVerifications: 350,
        slashCount: 0,
        contestedVerificationsCorrect: 33,
      };
      expect(checkTierTransition(2, state)).toBe(3);
    });
  });

  // ============ isTierEligibleForCategory ============

  describe("isTierEligibleForCategory", () => {
    it("Tier 0 should allow protocol claims", () => {
      expect(isTierEligibleForCategory(0, "protocol")).toBe(true);
    });

    it("Tier 0 should allow computational claims", () => {
      expect(isTierEligibleForCategory(0, "computational")).toBe(true);
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

    it("Tier 1 should NOT allow predictive claims", () => {
      expect(isTierEligibleForCategory(1, "predictive")).toBe(false);
    });

    it("Tier 2 should allow oracle and attestation but not contested", () => {
      expect(isTierEligibleForCategory(2, "oracle")).toBe(true);
      expect(isTierEligibleForCategory(2, "attestation")).toBe(true);
      expect(isTierEligibleForCategory(2, "contested")).toBe(false);
    });

    it("Tier 3 should allow contested and social claims", () => {
      expect(isTierEligibleForCategory(3, "social")).toBe(true);
      expect(isTierEligibleForCategory(3, "contested")).toBe(true);
      expect(isTierEligibleForCategory(3, "predictive")).toBe(true);
    });
  });

  // ============ getEffectiveSelectionStake ============

  describe("getEffectiveSelectionStake", () => {
    it("should return real stake for Tier 2 validators", () => {
      const result = getEffectiveSelectionStake(
        2,
        111n * 10n ** 17n, // 11.1 LGM
        5n * ONE_LGM,
        TIER_VIRTUAL_STAKE,
      );
      expect(result).toBe(111n * 10n ** 17n + 5n * ONE_LGM);
    });

    it("should return virtual stake * weight for Tier 0 with no real stake", () => {
      const result = getEffectiveSelectionStake(0, 0n, 0n, TIER_VIRTUAL_STAKE);
      const expected = (TIER_VIRTUAL_STAKE * 1000n) / 10000n;
      expect(result).toBe(expected);
    });

    it("should return virtual stake * 0.5 for Tier 1 with no real stake", () => {
      const result = getEffectiveSelectionStake(1, 0n, 0n, TIER_VIRTUAL_STAKE);
      const expected = (TIER_VIRTUAL_STAKE * 5000n) / 10000n;
      expect(result).toBe(expected);
    });

    it("should return real stake when Tier 0 has some stake", () => {
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
    const baseReward = 1n * ONE_LGM;

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
      const stake = 111n * 10n ** 17n;
      const { stakeSlash, reputationSlash } = calculateTierSlash(
        2,
        stake,
        0.05,
      );
      expect(stakeSlash).toBe((stake * 500n) / 10000n);
      expect(reputationSlash).toBe(0.1);
    });

    it("should always apply reputation slash regardless of tier", () => {
      const { reputationSlash: rep0 } = calculateTierSlash(0, 0n, 0.1);
      const { reputationSlash: rep3 } = calculateTierSlash(
        3,
        111n * ONE_LGM,
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

    it("TIER_VIRTUAL_STAKE should be 11 LGM", () => {
      expect(TIER_VIRTUAL_STAKE).toBe(11n * ONE_LGM);
    });

    it("MAX_APPRENTICE_VALIDATORS should be 111", () => {
      expect(MAX_APPRENTICE_VALIDATORS).toBe(111);
    });

    it("Tier 0 min stake should match MIN_VALIDATOR_STAKE", () => {
      expect(VALIDATOR_TIER_CONFIG[0].minStake).toBe(MIN_VALIDATOR_STAKE);
    });

    it("Tier 3 should be only tier with governance participation", () => {
      expect(VALIDATOR_TIER_CONFIG[0].canParticipateInGovernance).toBe(false);
      expect(VALIDATOR_TIER_CONFIG[1].canParticipateInGovernance).toBe(false);
      expect(VALIDATOR_TIER_CONFIG[2].canParticipateInGovernance).toBe(false);
      expect(VALIDATOR_TIER_CONFIG[3].canParticipateInGovernance).toBe(true);
    });

    it("Guardian should require contested verifications", () => {
      expect(VALIDATOR_TIER_CONFIG[3].minContestedVerifications).toBe(33);
      expect(VALIDATOR_TIER_CONFIG[3].contestedVerificationMultiplier).toBe(3);
    });

    it("Guardian should have 10-epoch slash window", () => {
      expect(VALIDATOR_TIER_CONFIG[3].slashWindowEpochs).toBe(10);
    });

    it("Guardian minVerifications should be 333 (reduced from 1111)", () => {
      expect(VALIDATOR_TIER_CONFIG[3].minVerifications).toBe(333);
    });

    it("Guardian minStake should be 111 LGM", () => {
      expect(VALIDATOR_TIER_CONFIG[3].minStake).toBe(111n * ONE_LGM);
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
      tree.registerValidator(
        publicKey,
        "did:lgm:staked",
        111n * 10n ** 17n,
        0n,
      );

      const validator = tree.getValidator(publicKey);
      expect(validator?.tier).toBe(2);
    });

    it("should promote tier when stake is added", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_promote" as any;
      tree.registerValidator(publicKey, "did:lgm:promote", 5n * ONE_LGM, 0n);

      expect(tree.getValidator(publicKey)?.tier).toBe(0);

      tree.addStake(publicKey, 7n * ONE_LGM); // Now 12 LGM > 11.1
      expect(tree.getValidator(publicKey)?.tier).toBe(2);
    });

    it("should demote tier when stake is removed", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_demote" as any;
      tree.registerValidator(publicKey, "did:lgm:demote", 12n * ONE_LGM, 0n);

      expect(tree.getValidator(publicKey)?.tier).toBe(2);

      tree.removeStake(publicKey, 2n * ONE_LGM); // Now 10 < 11.1
      expect(tree.getValidator(publicKey)?.tier).toBe(0);
    });

    it("should demote tier on slash", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_slash_demote" as any;
      tree.registerValidator(publicKey, "did:lgm:slash", 12n * ONE_LGM, 0n);

      expect(tree.getValidator(publicKey)?.tier).toBe(2);

      tree.slashValidator(publicKey, 2n * ONE_LGM, "test");
      expect(tree.getValidator(publicKey)?.tier).toBe(0);
    });

    it("should promote tier on verification milestone", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_verify_promote" as any;
      // Register with enough stake for Verified tier (1.11 LGM)
      tree.registerValidator(publicKey, "did:lgm:verify", 2n * ONE_LGM, 0n);

      expect(tree.getValidator(publicKey)?.tier).toBe(0);

      for (let i = 0; i < 22; i++) {
        tree.recordVerification(publicKey, true);
      }

      expect(tree.getValidator(publicKey)?.tier).toBe(1);
    });

    it("should track contested verifications", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_contested" as any;
      tree.registerValidator(publicKey, "did:lgm:contested", 0n, 0n);

      tree.recordVerification(publicKey, true, true); // contested
      tree.recordVerification(publicKey, true, false); // normal

      const v = tree.getValidator(publicKey);
      expect(v?.contestedVerificationsCorrect).toBe(1);
      expect(v?.verificationsCompleted).toBe(2);
      expect(v?.verificationsCorrect).toBe(2);
    });

    it("should record lastSlashBlock on slash", () => {
      const tree = new ValidatorStateTree();
      tree.setCurrentBlock(5000n);
      const publicKey = "pub_test_slash_block" as any;
      tree.registerValidator(publicKey, "did:lgm:slashb", 12n * ONE_LGM, 0n);

      tree.slashValidator(publicKey, 1n * ONE_LGM, "test");
      expect(tree.getValidator(publicKey)?.lastSlashBlock).toBe(5000n);
    });

    it("should enforce apprentice cap", () => {
      const tree = new ValidatorStateTree();

      for (let i = 0; i < MAX_APPRENTICE_VALIDATORS; i++) {
        const pk = `pub_apprentice_${i.toString().padStart(4, "0")}` as any;
        tree.registerValidator(pk, `did:lgm:a${i}`, 0n, 0n);
      }

      expect(tree.getApprenticeValidatorCount()).toBe(
        MAX_APPRENTICE_VALIDATORS,
      );

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

      tree.registerValidator(
        "pub_staked" as any,
        "did:lgm:s",
        111n * 10n ** 17n,
        0n,
      );
      expect(tree.getApprenticeValidatorCount()).toBe(0);

      tree.registerValidator("pub_free" as any, "did:lgm:f", 0n, 0n);
      expect(tree.getApprenticeValidatorCount()).toBe(1);
    });

    it("should serialize and deserialize tier correctly", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_serialize_tier" as any;
      tree.registerValidator(publicKey, "did:lgm:ser", 111n * 10n ** 17n, 0n);

      const cloned = tree.clone();
      const validator = cloned.getValidator(publicKey);
      expect(validator?.tier).toBe(2);
    });

    it("should decrement apprentice count on tier promotion", () => {
      const tree = new ValidatorStateTree();
      const publicKey = "pub_test_apprentice_dec" as any;
      tree.registerValidator(publicKey, "did:lgm:ad", 0n, 0n);
      expect(tree.getApprenticeValidatorCount()).toBe(1);

      tree.addStake(publicKey, 111n * 10n ** 17n);
      expect(tree.getValidator(publicKey)?.tier).toBe(2);
      expect(tree.getApprenticeValidatorCount()).toBe(0);
    });
  });
});
