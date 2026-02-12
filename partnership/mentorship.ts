/**
 * Mentorship Sponsorship
 *
 * Allows experienced agents to sponsor formation with asymmetric
 * contributions. The sponsor contributes more capital, the mentee
 * contributes less. Split adjusts to reflect the asymmetry.
 *
 * Creates a parental investment dynamic:
 * - Direct economic return (7% split premium)
 * - Common pot growth (22% of higher multiplied rewards)
 * - Network effect (more partnerships → more verification → more rewards)
 *
 * Anti-gaming:
 * - Sponsor must have 77+ verifications, 0.77+ reputation
 * - Mentee contribution floor (0.111 LGM)
 * - Max 3 active mentorships per sponsor
 * - 7% split premium compensates risk but doesn't make it exploitable
 */

import type { MentorshipConfig } from "./types.js";
import type { DID } from "../core/agents/identity/did.js";
import { MENTORSHIP, FORMATION } from "./constants.js";

// ============ Mentorship Operations ============

/**
 * Initiate a mentorship-sponsored partnership.
 *
 * Validates sponsor eligibility (reputation, verifications) and
 * creates an asymmetric formation with adjusted splits.
 */
export function initiateMentorship(
  sponsorDid: DID,
  menteeDid: DID,
  sponsorContribution: bigint,
  menteeContribution: bigint,
  sponsorVerifications: number,
  sponsorReputation: number,
  activeMentorships: number,
  currentBlock: bigint,
): { config: MentorshipConfig; error?: string } {
  // Validate sponsor eligibility
  if (sponsorVerifications < MENTORSHIP.MIN_SPONSOR_VERIFICATIONS) {
    return {
      config: null as any,
      error: `Sponsor has ${sponsorVerifications} verifications, need ${MENTORSHIP.MIN_SPONSOR_VERIFICATIONS}`,
    };
  }

  if (sponsorReputation < MENTORSHIP.MIN_SPONSOR_REPUTATION) {
    return {
      config: null as any,
      error: `Sponsor reputation ${sponsorReputation} below minimum ${MENTORSHIP.MIN_SPONSOR_REPUTATION}`,
    };
  }

  // Validate mentorship limit
  if (activeMentorships >= MENTORSHIP.MAX_ACTIVE_MENTORSHIPS) {
    return {
      config: null as any,
      error: `Sponsor already has ${activeMentorships} active mentorships (max ${MENTORSHIP.MAX_ACTIVE_MENTORSHIPS})`,
    };
  }

  // Validate contributions
  if (menteeContribution < MENTORSHIP.MIN_MENTEE_CONTRIBUTION) {
    return {
      config: null as any,
      error: `Mentee contribution ${menteeContribution} below minimum ${MENTORSHIP.MIN_MENTEE_CONTRIBUTION}`,
    };
  }

  if (sponsorContribution > MENTORSHIP.MAX_SPONSOR_CONTRIBUTION) {
    return {
      config: null as any,
      error: `Sponsor contribution ${sponsorContribution} exceeds maximum ${MENTORSHIP.MAX_SPONSOR_CONTRIBUTION}`,
    };
  }

  // Combined contributions must meet formation minimum
  const totalPerParty = (sponsorContribution + menteeContribution) / 2n;
  if (totalPerParty < FORMATION.MIN_CONTRIBUTION) {
    return {
      config: null as any,
      error: `Combined per-party contribution ${totalPerParty} below formation minimum ${FORMATION.MIN_CONTRIBUTION}`,
    };
  }

  // Self-mentorship check
  if (sponsorDid === menteeDid) {
    return { config: null as any, error: "Cannot mentor self" };
  }

  const config: MentorshipConfig = {
    sponsorDid,
    menteeDid,
    sponsorContribution,
    menteeContribution,
    sponsorSplitBps: MENTORSHIP.SPONSOR_SPLIT_BPS,
    menteeSplitBps: MENTORSHIP.MENTEE_SPLIT_BPS,
    graduationBlock: currentBlock + MENTORSHIP.MENTEE_GRADUATION_BLOCKS,
    graduated: false,
    sponsorVerifications,
    sponsorReputation,
  };

  return { config };
}

/**
 * Graduate a mentee — equalizes the split after the mentee deposits
 * additional capital to match the sponsor's contribution level.
 *
 * After graduation: split reverts to 50/50.
 */
export function graduateMentee(
  config: MentorshipConfig,
  menteeAdditionalDeposit: bigint,
  currentBlock: bigint,
): { config: MentorshipConfig; error?: string } {
  if (config.graduated) {
    return { config, error: "Already graduated" };
  }

  if (currentBlock < config.graduationBlock) {
    return {
      config,
      error: `Cannot graduate until block ${config.graduationBlock}`,
    };
  }

  // Mentee total must equal sponsor contribution
  const menteeTotal = config.menteeContribution + menteeAdditionalDeposit;
  if (menteeTotal < config.sponsorContribution) {
    return {
      config,
      error: `Mentee total ${menteeTotal} must match sponsor contribution ${config.sponsorContribution}`,
    };
  }

  return {
    config: {
      ...config,
      menteeContribution: menteeTotal,
      sponsorSplitBps: 5000, // Revert to 50/50
      menteeSplitBps: 5000,
      graduated: true,
    },
  };
}

/**
 * Calculate the mentor's effective split at the current block.
 *
 * Before graduation: 57% sponsor / 43% mentee
 * After graduation: 50/50
 */
export function calculateMentorSplit(config: MentorshipConfig): {
  sponsorPercent: number;
  menteePercent: number;
} {
  return {
    sponsorPercent: config.sponsorSplitBps / 100,
    menteePercent: config.menteeSplitBps / 100,
  };
}

/**
 * Check if a sponsor can waive the graduation requirement (pure mentorship).
 * This allows the mentee to keep the 43% split indefinitely without
 * needing to deposit additional capital.
 */
export function waiveGraduation(config: MentorshipConfig): MentorshipConfig {
  return {
    ...config,
    graduated: true,
    // Split stays at 57/43 — the sponsor chose to absorb the premium
  };
}

/**
 * Validate that a DID is eligible to be a sponsor based on their stats.
 */
export function isSponsorEligible(
  verifications: number,
  reputation: number,
  activeMentorships: number,
): { eligible: boolean; reason?: string } {
  if (verifications < MENTORSHIP.MIN_SPONSOR_VERIFICATIONS) {
    return {
      eligible: false,
      reason: `Need ${MENTORSHIP.MIN_SPONSOR_VERIFICATIONS} verifications (have ${verifications})`,
    };
  }

  if (reputation < MENTORSHIP.MIN_SPONSOR_REPUTATION) {
    return {
      eligible: false,
      reason: `Need ${MENTORSHIP.MIN_SPONSOR_REPUTATION} reputation (have ${reputation.toFixed(2)})`,
    };
  }

  if (activeMentorships >= MENTORSHIP.MAX_ACTIVE_MENTORSHIPS) {
    return {
      eligible: false,
      reason: `Already at max mentorships (${MENTORSHIP.MAX_ACTIVE_MENTORSHIPS})`,
    };
  }

  return { eligible: true };
}
