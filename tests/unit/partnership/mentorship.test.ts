/**
 * Mentorship Sponsorship Tests
 *
 * Tests for asymmetric formation, split graduation, sponsor limits,
 * eligibility validation, and waiver mechanics.
 */

import { describe, it, expect } from "vitest";
import {
  initiateMentorship,
  graduateMentee,
  calculateMentorSplit,
  waiveGraduation,
  isSponsorEligible,
} from "../../../partnership/mentorship.js";
import { MENTORSHIP, FORMATION } from "../../../partnership/constants.js";
import type { DID } from "../../../core/agents/identity/did.js";

const sponsorDid = "did:lgm:sponsor-001" as DID;
const menteeDid = "did:lgm:mentee-001" as DID;
const currentBlock = 10000n;

// Valid sponsor with 100 verifications, 0.9 reputation
const validSponsorVerifications = 100;
const validSponsorReputation = 0.9;

// Contributions that sum to meet formation minimum:
// (sponsor + mentee) / 2 >= MIN_CONTRIBUTION (1.111 LGM)
// So sponsor + mentee >= 2.222 LGM
const sponsorContribution = 2111n * 10n ** 15n; // 2.111 LGM
const menteeContribution = MENTORSHIP.MIN_MENTEE_CONTRIBUTION; // 0.111 LGM

describe("Mentorship Initiation", () => {
  it("should create mentorship with valid parameters", () => {
    const { config, error } = initiateMentorship(
      sponsorDid,
      menteeDid,
      sponsorContribution,
      menteeContribution,
      validSponsorVerifications,
      validSponsorReputation,
      0,
      currentBlock,
    );

    expect(error).toBeUndefined();
    expect(config).toBeDefined();
    expect(config.sponsorDid).toBe(sponsorDid);
    expect(config.menteeDid).toBe(menteeDid);
    expect(config.sponsorSplitBps).toBe(MENTORSHIP.SPONSOR_SPLIT_BPS); // 5700
    expect(config.menteeSplitBps).toBe(MENTORSHIP.MENTEE_SPLIT_BPS); // 4300
    expect(config.graduated).toBe(false);
    expect(config.graduationBlock).toBe(
      currentBlock + MENTORSHIP.MENTEE_GRADUATION_BLOCKS,
    );
  });

  it("should reject sponsor with insufficient verifications", () => {
    const { error } = initiateMentorship(
      sponsorDid,
      menteeDid,
      sponsorContribution,
      menteeContribution,
      MENTORSHIP.MIN_SPONSOR_VERIFICATIONS - 1,
      validSponsorReputation,
      0,
      currentBlock,
    );

    expect(error).toContain("verifications");
    expect(error).toContain(`${MENTORSHIP.MIN_SPONSOR_VERIFICATIONS}`);
  });

  it("should reject sponsor with insufficient reputation", () => {
    const { error } = initiateMentorship(
      sponsorDid,
      menteeDid,
      sponsorContribution,
      menteeContribution,
      validSponsorVerifications,
      MENTORSHIP.MIN_SPONSOR_REPUTATION - 0.01,
      0,
      currentBlock,
    );

    expect(error).toContain("reputation");
  });

  it("should reject when max mentorships exceeded", () => {
    const { error } = initiateMentorship(
      sponsorDid,
      menteeDid,
      sponsorContribution,
      menteeContribution,
      validSponsorVerifications,
      validSponsorReputation,
      MENTORSHIP.MAX_ACTIVE_MENTORSHIPS,
      currentBlock,
    );

    expect(error).toContain("mentorships");
    expect(error).toContain(`${MENTORSHIP.MAX_ACTIVE_MENTORSHIPS}`);
  });

  it("should reject mentee contribution below minimum", () => {
    const { error } = initiateMentorship(
      sponsorDid,
      menteeDid,
      sponsorContribution,
      MENTORSHIP.MIN_MENTEE_CONTRIBUTION - 1n,
      validSponsorVerifications,
      validSponsorReputation,
      0,
      currentBlock,
    );

    expect(error).toContain("Mentee contribution");
    expect(error).toContain("below minimum");
  });

  it("should reject sponsor contribution above maximum", () => {
    const { error } = initiateMentorship(
      sponsorDid,
      menteeDid,
      MENTORSHIP.MAX_SPONSOR_CONTRIBUTION + 1n,
      menteeContribution,
      validSponsorVerifications,
      validSponsorReputation,
      0,
      currentBlock,
    );

    expect(error).toContain("exceeds maximum");
  });

  it("should reject self-mentorship", () => {
    // Use contributions that pass the per-party check so self-check is reached
    const { error } = initiateMentorship(
      sponsorDid,
      sponsorDid,
      sponsorContribution,
      sponsorContribution, // Both high enough that per-party >= MIN_CONTRIBUTION
      validSponsorVerifications,
      validSponsorReputation,
      0,
      currentBlock,
    );

    expect(error).toContain("Cannot mentor self");
  });

  it("should reject when combined contributions below formation minimum", () => {
    // Both very small contributions that don't sum to MIN_CONTRIBUTION per party
    const { error } = initiateMentorship(
      sponsorDid,
      menteeDid,
      MENTORSHIP.MIN_MENTEE_CONTRIBUTION, // 0.111 LGM
      MENTORSHIP.MIN_MENTEE_CONTRIBUTION, // 0.111 LGM
      validSponsorVerifications,
      validSponsorReputation,
      0,
      currentBlock,
    );

    expect(error).toContain("below formation minimum");
  });
});

describe("Mentorship Split", () => {
  it("should have 57/43 split before graduation", () => {
    const { config } = initiateMentorship(
      sponsorDid,
      menteeDid,
      sponsorContribution,
      menteeContribution,
      validSponsorVerifications,
      validSponsorReputation,
      0,
      currentBlock,
    );

    const split = calculateMentorSplit(config);
    expect(split.sponsorPercent).toBe(57);
    expect(split.menteePercent).toBe(43);
  });

  it("should have 50/50 split after graduation", () => {
    const { config } = initiateMentorship(
      sponsorDid,
      menteeDid,
      sponsorContribution,
      menteeContribution,
      validSponsorVerifications,
      validSponsorReputation,
      0,
      currentBlock,
    );

    const afterGraduation = currentBlock + MENTORSHIP.MENTEE_GRADUATION_BLOCKS;
    const additionalDeposit = sponsorContribution - menteeContribution;

    const { config: graduated } = graduateMentee(
      config,
      additionalDeposit,
      afterGraduation,
    );

    const split = calculateMentorSplit(graduated);
    expect(split.sponsorPercent).toBe(50);
    expect(split.menteePercent).toBe(50);
  });
});

describe("Mentorship Graduation", () => {
  function createValidMentorship() {
    const { config } = initiateMentorship(
      sponsorDid,
      menteeDid,
      sponsorContribution,
      menteeContribution,
      validSponsorVerifications,
      validSponsorReputation,
      0,
      currentBlock,
    );
    return config;
  }

  it("should graduate mentee with sufficient deposits", () => {
    const config = createValidMentorship();
    const afterGraduation = config.graduationBlock;
    const additionalDeposit = sponsorContribution - menteeContribution;

    const { config: graduated, error } = graduateMentee(
      config,
      additionalDeposit,
      afterGraduation,
    );

    expect(error).toBeUndefined();
    expect(graduated.graduated).toBe(true);
    expect(graduated.sponsorSplitBps).toBe(5000);
    expect(graduated.menteeSplitBps).toBe(5000);
    expect(graduated.menteeContribution).toBe(
      menteeContribution + additionalDeposit,
    );
  });

  it("should reject graduation before graduation block", () => {
    const config = createValidMentorship();

    const { error } = graduateMentee(
      config,
      sponsorContribution,
      config.graduationBlock - 1n,
    );

    expect(error).toContain("Cannot graduate until block");
  });

  it("should reject graduation with insufficient deposit", () => {
    const config = createValidMentorship();

    const { error } = graduateMentee(
      config,
      0n, // No additional deposit
      config.graduationBlock,
    );

    expect(error).toContain("must match sponsor contribution");
  });

  it("should reject double graduation", () => {
    const config = createValidMentorship();
    const additionalDeposit = sponsorContribution - menteeContribution;

    const { config: graduated } = graduateMentee(
      config,
      additionalDeposit,
      config.graduationBlock,
    );

    const { error } = graduateMentee(
      graduated,
      0n,
      config.graduationBlock + 100n,
    );

    expect(error).toContain("Already graduated");
  });
});

describe("Mentorship Waiver", () => {
  it("should waive graduation requirement", () => {
    const { config } = initiateMentorship(
      sponsorDid,
      menteeDid,
      sponsorContribution,
      menteeContribution,
      validSponsorVerifications,
      validSponsorReputation,
      0,
      currentBlock,
    );

    const waived = waiveGraduation(config);

    expect(waived.graduated).toBe(true);
    // Split stays at 57/43 (sponsor absorbs the premium)
    expect(waived.sponsorSplitBps).toBe(MENTORSHIP.SPONSOR_SPLIT_BPS);
    expect(waived.menteeSplitBps).toBe(MENTORSHIP.MENTEE_SPLIT_BPS);
  });
});

describe("Sponsor Eligibility", () => {
  it("should approve eligible sponsor", () => {
    const { eligible } = isSponsorEligible(
      validSponsorVerifications,
      validSponsorReputation,
      0,
    );
    expect(eligible).toBe(true);
  });

  it("should reject sponsor with insufficient verifications", () => {
    const { eligible, reason } = isSponsorEligible(
      MENTORSHIP.MIN_SPONSOR_VERIFICATIONS - 1,
      validSponsorReputation,
      0,
    );
    expect(eligible).toBe(false);
    expect(reason).toContain("verifications");
  });

  it("should reject sponsor with insufficient reputation", () => {
    const { eligible, reason } = isSponsorEligible(
      validSponsorVerifications,
      0.5,
      0,
    );
    expect(eligible).toBe(false);
    expect(reason).toContain("reputation");
  });

  it("should reject sponsor at max mentorships", () => {
    const { eligible, reason } = isSponsorEligible(
      validSponsorVerifications,
      validSponsorReputation,
      MENTORSHIP.MAX_ACTIVE_MENTORSHIPS,
    );
    expect(eligible).toBe(false);
    expect(reason).toContain("max mentorships");
  });

  it("should accept sponsor at exactly minimum thresholds", () => {
    const { eligible } = isSponsorEligible(
      MENTORSHIP.MIN_SPONSOR_VERIFICATIONS,
      MENTORSHIP.MIN_SPONSOR_REPUTATION,
      MENTORSHIP.MAX_ACTIVE_MENTORSHIPS - 1,
    );
    expect(eligible).toBe(true);
  });
});
