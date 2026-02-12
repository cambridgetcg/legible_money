/**
 * Emergency BFT Ceremony Engine
 *
 * Guardian-only voting ceremonies for emergency operations.
 * Modeled on CheckpointCeremony but with:
 *   - Guardian-only participation (tier === 3)
 *   - Guardian-stake-weighted quorum (not total validator stake)
 *   - Higher thresholds: 75% halt, 80% revert/resume
 */

import type { ValidatorSet, ConsensusParams } from "../pot/types.js";
import type {
  EmergencyCeremony,
  EmergencyCeremonyType,
  EmergencyHaltProposal,
  EmergencyRevertProposal,
  EmergencyResumeProposal,
  EmergencyVote,
  EmergencyPrecommit,
} from "./types.js";
import { EMERGENCY_PARAMS } from "./constants.js";

// ============ Genesis Emergency Council (H-5) ============

/**
 * H-5: Genesis emergency council context.
 * When provided, council members are treated as honorary Guardians
 * with equal virtual stake during the bootstrap period.
 */
export interface CouncilContext {
  /** DIDs of genesis council members */
  councilMembers: string[];
  /** Block after which council powers expire (0 = no council) */
  expiryBlock: bigint;
  /** Current block number */
  currentBlock: bigint;
}

/** Virtual stake assigned to each council member for quorum calculation */
const COUNCIL_MEMBER_VIRTUAL_STAKE = 11111n * 10n ** 15n; // 11.111 LGM each

/**
 * Check if the genesis council is active (within bootstrap period).
 */
function isCouncilActive(council?: CouncilContext): boolean {
  if (!council) return false;
  if (council.councilMembers.length === 0) return false;
  if (council.expiryBlock === 0n) return false;
  return council.currentBlock < council.expiryBlock;
}

/**
 * Check if a public key is a genesis council member.
 */
function isCouncilMember(publicKey: string, council?: CouncilContext): boolean {
  if (!isCouncilActive(council)) return false;
  return council!.councilMembers.includes(publicKey);
}

/**
 * Build council context from ConsensusParams and current block.
 */
export function buildCouncilContext(
  params: ConsensusParams,
  currentBlock: bigint,
): CouncilContext | undefined {
  const members = params.genesisEmergencyCouncil;
  const expiry = params.genesisEmergencyCouncilExpiryBlock;
  if (!members || members.length === 0 || !expiry || expiry === 0n)
    return undefined;
  return { councilMembers: members, expiryBlock: expiry, currentBlock };
}

// ============ Guardian Helpers ============

/**
 * Get total stake held by Guardian-tier (tier 3) validators.
 * H-5: Includes virtual stake from active genesis council members.
 */
export function getGuardianStake(
  validatorSet: ValidatorSet,
  council?: CouncilContext,
): bigint {
  let total = 0n;
  for (const validator of validatorSet.validators.values()) {
    if (validator.tier === 3 && validator.isActive) {
      total += validator.stake + validator.delegatedStake;
    }
  }
  // H-5: Add council virtual stake during bootstrap period
  if (isCouncilActive(council)) {
    total +=
      COUNCIL_MEMBER_VIRTUAL_STAKE * BigInt(council!.councilMembers.length);
  }
  return total;
}

/**
 * Check if a public key belongs to an active Guardian-tier validator.
 * H-5: Council members are treated as honorary Guardians during bootstrap.
 */
export function isGuardian(
  publicKey: string,
  validatorSet: ValidatorSet,
  council?: CouncilContext,
): boolean {
  const validator = validatorSet.validators.get(publicKey);
  if (validator !== undefined && validator.tier === 3 && validator.isActive) {
    return true;
  }
  // H-5: Council members are honorary Guardians during bootstrap
  return isCouncilMember(publicKey, council);
}

/**
 * Get a Guardian's effective stake (own + delegated).
 * H-5: Council members get virtual stake during bootstrap.
 */
export function getGuardianEffectiveStake(
  publicKey: string,
  validatorSet: ValidatorSet,
  council?: CouncilContext,
): bigint {
  const validator = validatorSet.validators.get(publicKey);
  if (validator && validator.tier === 3 && validator.isActive) {
    return validator.stake + validator.delegatedStake;
  }
  // H-5: Council members get virtual stake
  if (isCouncilMember(publicKey, council)) {
    return COUNCIL_MEMBER_VIRTUAL_STAKE;
  }
  return 0n;
}

/**
 * Get the quorum threshold for a ceremony type.
 * Returns basis points (e.g. 750000 = 75%).
 */
function getQuorumThresholdBP(type: EmergencyCeremonyType): bigint {
  switch (type) {
    case "halt":
      return EMERGENCY_PARAMS.GUARDIAN_HALT_QUORUM_BP;
    case "revert":
      return EMERGENCY_PARAMS.GUARDIAN_REVERT_QUORUM_BP;
    case "resume":
      return EMERGENCY_PARAMS.GUARDIAN_RESUME_QUORUM_BP;
  }
}

/**
 * Get ceremony timing parameters for a ceremony type.
 */
function getCeremonyTiming(type: EmergencyCeremonyType): {
  prevoteBlocks: bigint;
  precommitBlocks: bigint;
  timeoutBlocks: bigint;
} {
  switch (type) {
    case "halt":
      return {
        prevoteBlocks: EMERGENCY_PARAMS.HALT_PREVOTE_BLOCKS,
        precommitBlocks: EMERGENCY_PARAMS.HALT_PRECOMMIT_BLOCKS,
        timeoutBlocks: EMERGENCY_PARAMS.HALT_TIMEOUT_BLOCKS,
      };
    case "revert":
      return {
        prevoteBlocks: EMERGENCY_PARAMS.REVERT_PREVOTE_BLOCKS,
        precommitBlocks: EMERGENCY_PARAMS.REVERT_PRECOMMIT_BLOCKS,
        timeoutBlocks: EMERGENCY_PARAMS.REVERT_TIMEOUT_BLOCKS,
      };
    case "resume":
      return {
        prevoteBlocks: EMERGENCY_PARAMS.RESUME_PREVOTE_BLOCKS,
        precommitBlocks: EMERGENCY_PARAMS.RESUME_PRECOMMIT_BLOCKS,
        timeoutBlocks: EMERGENCY_PARAMS.RESUME_TIMEOUT_BLOCKS,
      };
  }
}

// ============ Ceremony Creation ============

/**
 * Create a new halt ceremony from a proposal.
 */
export function createHaltCeremony(
  proposal: EmergencyHaltProposal,
  currentBlock: bigint,
): EmergencyCeremony<EmergencyHaltProposal> {
  return createCeremony("halt", proposal, currentBlock);
}

/**
 * Create a new revert ceremony from a proposal.
 */
export function createRevertCeremony(
  proposal: EmergencyRevertProposal,
  currentBlock: bigint,
): EmergencyCeremony<EmergencyRevertProposal> {
  return createCeremony("revert", proposal, currentBlock);
}

/**
 * Create a new resume ceremony from a proposal.
 */
export function createResumeCeremony(
  proposal: EmergencyResumeProposal,
  currentBlock: bigint,
): EmergencyCeremony<EmergencyResumeProposal> {
  return createCeremony("resume", proposal, currentBlock);
}

function createCeremony<P>(
  type: EmergencyCeremonyType,
  proposal: P & { id: string },
  currentBlock: bigint,
): EmergencyCeremony<P> {
  const timing = getCeremonyTiming(type);
  return {
    id: proposal.id,
    type,
    proposal,
    phase: "prevote", // Starts directly in prevote (proposer implicitly endorses)
    startBlock: currentBlock,
    prevoteDeadline: currentBlock + timing.prevoteBlocks,
    precommitDeadline:
      currentBlock + timing.prevoteBlocks + timing.precommitBlocks,
    timeoutDeadline: currentBlock + timing.timeoutBlocks,
    prevotes: new Map(),
    precommits: new Map(),
    yesPrevoteStake: 0n,
    noPrevoteStake: 0n,
    precommitStake: 0n,
  };
}

// ============ Voting ============

/**
 * Add a prevote to a ceremony.
 * Returns whether the vote was accepted and the updated ceremony.
 */
export function addPrevote(
  ceremony: EmergencyCeremony,
  vote: EmergencyVote,
  validatorSet: ValidatorSet,
  currentBlock: bigint,
  council?: CouncilContext,
): { accepted: boolean; ceremony: EmergencyCeremony; reason?: string } {
  // Phase check
  if (ceremony.phase !== "prevote") {
    return {
      accepted: false,
      ceremony,
      reason: `Not in prevote phase (current: ${ceremony.phase})`,
    };
  }

  // Deadline check
  if (currentBlock > ceremony.prevoteDeadline) {
    return { accepted: false, ceremony, reason: "Prevote deadline passed" };
  }

  // Guardian check (H-5: council members accepted during bootstrap)
  if (!isGuardian(vote.voter, validatorSet, council)) {
    return {
      accepted: false,
      ceremony,
      reason: "Not a Guardian-tier validator",
    };
  }

  // Duplicate check
  if (ceremony.prevotes.has(vote.voter)) {
    return { accepted: false, ceremony, reason: "Already voted in prevote" };
  }

  // Record vote
  const stake = getGuardianEffectiveStake(vote.voter, validatorSet, council);
  ceremony.prevotes.set(vote.voter, vote);
  if (vote.vote === "yes") {
    ceremony.yesPrevoteStake += stake;
  } else {
    ceremony.noPrevoteStake += stake;
  }

  return { accepted: true, ceremony };
}

/**
 * Add a precommit to a ceremony.
 */
export function addPrecommit(
  ceremony: EmergencyCeremony,
  precommit: EmergencyPrecommit,
  validatorSet: ValidatorSet,
  currentBlock: bigint,
  council?: CouncilContext,
): { accepted: boolean; ceremony: EmergencyCeremony; reason?: string } {
  // Phase check
  if (ceremony.phase !== "precommit") {
    return {
      accepted: false,
      ceremony,
      reason: `Not in precommit phase (current: ${ceremony.phase})`,
    };
  }

  // Deadline check
  if (currentBlock > ceremony.precommitDeadline) {
    return { accepted: false, ceremony, reason: "Precommit deadline passed" };
  }

  // Guardian check (H-5: council members accepted during bootstrap)
  if (!isGuardian(precommit.voter, validatorSet, council)) {
    return {
      accepted: false,
      ceremony,
      reason: "Not a Guardian-tier validator",
    };
  }

  // Must have prevoted "yes" to precommit
  const prevote = ceremony.prevotes.get(precommit.voter);
  if (!prevote || prevote.vote !== "yes") {
    return {
      accepted: false,
      ceremony,
      reason: "Must prevote 'yes' before precommit",
    };
  }

  // Duplicate check
  if (ceremony.precommits.has(precommit.voter)) {
    return { accepted: false, ceremony, reason: "Already precommitted" };
  }

  // Record precommit
  const stake = getGuardianEffectiveStake(
    precommit.voter,
    validatorSet,
    council,
  );
  ceremony.precommits.set(precommit.voter, precommit);
  ceremony.precommitStake += stake;

  return { accepted: true, ceremony };
}

// ============ Phase Transitions ============

/**
 * Check ceremony progress and handle phase transitions.
 * Called once per block to advance ceremonies or detect timeouts.
 */
export function checkCeremonyProgress(
  ceremony: EmergencyCeremony,
  validatorSet: ValidatorSet,
  currentBlock: bigint,
  council?: CouncilContext,
): EmergencyCeremony {
  // Terminal states
  if (ceremony.phase === "finalized" || ceremony.phase === "failed") {
    return ceremony;
  }

  // Timeout check — applies to all active phases
  if (currentBlock > ceremony.timeoutDeadline) {
    ceremony.phase = "failed";
    ceremony.failureReason = "Ceremony timed out";
    return ceremony;
  }

  const guardianStake = getGuardianStake(validatorSet, council);
  if (guardianStake === 0n) {
    ceremony.phase = "failed";
    ceremony.failureReason = "No active Guardian validators";
    return ceremony;
  }

  // Minimum absolute Guardian stake floor — prevents early-network collusion
  // H-5: When council is active, skip this check (council provides the emergency capability)
  if (
    !isCouncilActive(council) &&
    guardianStake < EMERGENCY_PARAMS.MIN_GUARDIAN_STAKE_FOR_EMERGENCY
  ) {
    ceremony.phase = "failed";
    ceremony.failureReason = `Insufficient Guardian stake: below ${EMERGENCY_PARAMS.MIN_GUARDIAN_STAKE_FOR_EMERGENCY} minimum`;
    return ceremony;
  }

  // Quorum threshold in basis points (e.g. 750000 = 75%)
  const thresholdBP = getQuorumThresholdBP(ceremony.type);
  const BASIS = 1_000_000n;

  if (ceremony.phase === "prevote") {
    // Check if quorum reached for prevote
    // yesRatioBP = yesPrevoteStake * 1_000_000 / guardianStake
    const yesRatioBP = (ceremony.yesPrevoteStake * BASIS) / guardianStake;
    if (yesRatioBP >= thresholdBP) {
      // Advance to precommit
      ceremony.phase = "precommit";
      return ceremony;
    }

    // Check if quorum is impossible (too many "no" votes)
    const noRatioBP = (ceremony.noPrevoteStake * BASIS) / guardianStake;
    if (noRatioBP > BASIS - thresholdBP) {
      ceremony.phase = "failed";
      ceremony.failureReason = `Insufficient support: ${(yesRatioBP * 100n) / BASIS}% < ${(thresholdBP * 100n) / BASIS}% required`;
      return ceremony;
    }

    // Check prevote deadline
    if (currentBlock > ceremony.prevoteDeadline) {
      ceremony.phase = "failed";
      ceremony.failureReason = `Prevote quorum not reached before deadline: ${(yesRatioBP * 100n) / BASIS}% < ${(thresholdBP * 100n) / BASIS}% required`;
      return ceremony;
    }
  }

  if (ceremony.phase === "precommit") {
    // Check if precommit quorum reached
    const precommitRatioBP = (ceremony.precommitStake * BASIS) / guardianStake;
    if (precommitRatioBP >= thresholdBP) {
      // Minimum distinct voter count — prevents single whale from acting alone
      if (ceremony.precommits.size < EMERGENCY_PARAMS.MIN_GUARDIAN_VOTERS) {
        // Don't fail yet — more voters may arrive before deadline
      } else {
        ceremony.phase = "finalized";
        return ceremony;
      }
    }

    // Check precommit deadline
    if (currentBlock > ceremony.precommitDeadline) {
      if (
        precommitRatioBP >= thresholdBP &&
        ceremony.precommits.size < EMERGENCY_PARAMS.MIN_GUARDIAN_VOTERS
      ) {
        ceremony.phase = "failed";
        ceremony.failureReason = `Insufficient distinct voters: ${ceremony.precommits.size} < ${EMERGENCY_PARAMS.MIN_GUARDIAN_VOTERS} required`;
      } else {
        ceremony.phase = "failed";
        ceremony.failureReason = `Precommit quorum not reached before deadline: ${(precommitRatioBP * 100n) / BASIS}% < ${(thresholdBP * 100n) / BASIS}% required`;
      }
      return ceremony;
    }
  }

  return ceremony;
}
