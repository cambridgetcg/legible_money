/**
 * Emergency Ceremony Manager
 *
 * State machine managing the emergency kill switch lifecycle:
 *   NORMAL → HALTED → (optionally REVERTED) → RESUMED → NORMAL
 *
 * Enforces anti-abuse limits and coordinates BFT ceremony phases.
 */

import type { ValidatorSet } from "../pot/types.js";
import type {
  EmergencyCeremony,
  EmergencyHaltProposal,
  EmergencyRevertProposal,
  EmergencyResumeProposal,
  EmergencyStatus,
  CeremonyResult,
  FinalizedCeremonyResult,
} from "./types.js";
import { EMERGENCY_PARAMS } from "./constants.js";
import {
  isGuardian,
  getGuardianStake,
  createHaltCeremony,
  createRevertCeremony,
  createResumeCeremony,
  addPrevote,
  addPrecommit,
  checkCeremonyProgress,
  type CouncilContext,
} from "./ceremony.js";

// ============ Manager ============

export class EmergencyCeremonyManager {
  private activeCeremony: EmergencyCeremony | null = null;
  private completedCeremonies: EmergencyCeremony[] = [];
  private emergencyStatus: EmergencyStatus = "normal";

  // Anti-abuse tracking: guardian pubkey → { count, lastBlock }
  private proposalCounts: Map<string, { count: number; lastBlock: bigint }> =
    new Map();
  private epochProposalCount: number = 0;
  private lastProposalBlock: bigint = 0n;

  // Track the halt ceremony ID for revert/resume validation
  private activeHaltCeremonyId: string | null = null;

  // ============ Getters ============

  getActiveCeremony(): EmergencyCeremony | null {
    return this.activeCeremony;
  }

  getCompletedCeremonies(): EmergencyCeremony[] {
    return this.completedCeremonies;
  }

  getEmergencyStatus(): EmergencyStatus {
    return this.emergencyStatus;
  }

  getActiveHaltCeremonyId(): string | null {
    return this.activeHaltCeremonyId;
  }

  /**
   * Returns true when the network should reject non-emergency transactions
   * and stop block production.
   */
  isHalted(): boolean {
    return (
      this.emergencyStatus === "halted" ||
      this.emergencyStatus === "revert_voting" ||
      this.emergencyStatus === "reverting" ||
      this.emergencyStatus === "resume_voting"
    );
  }

  // ============ Status Management ============

  setEmergencyStatus(status: EmergencyStatus): void {
    this.emergencyStatus = status;
  }

  // ============ Halt Proposal ============

  proposeHalt(
    proposal: EmergencyHaltProposal,
    validatorSet: ValidatorSet,
    currentBlock: bigint,
    council?: CouncilContext,
  ): CeremonyResult {
    // Must be in normal status
    if (this.emergencyStatus !== "normal") {
      return {
        accepted: false,
        reason: `Cannot propose halt: network is in '${this.emergencyStatus}' state`,
      };
    }

    // No active ceremony
    if (this.activeCeremony !== null) {
      return {
        accepted: false,
        reason: "Another emergency ceremony is already active",
      };
    }

    // Proposer must be Guardian (H-5: council members accepted during bootstrap)
    if (!isGuardian(proposal.proposer, validatorSet, council)) {
      return { accepted: false, reason: "Not a Guardian-tier validator" };
    }

    // Minimum absolute Guardian stake floor — prevents early-network collusion
    // H-5: Skip when council is active (council provides bootstrap emergency capability)
    const totalGuardianStake = getGuardianStake(validatorSet, council);
    if (
      !council &&
      totalGuardianStake < EMERGENCY_PARAMS.MIN_GUARDIAN_STAKE_FOR_EMERGENCY
    ) {
      return {
        accepted: false,
        reason: `Insufficient Guardian stake: ${totalGuardianStake} < ${EMERGENCY_PARAMS.MIN_GUARDIAN_STAKE_FOR_EMERGENCY} minimum`,
      };
    }

    // Anti-abuse: per-guardian limit
    const guardianRecord = this.proposalCounts.get(proposal.proposer);
    if (
      guardianRecord &&
      guardianRecord.count >=
        EMERGENCY_PARAMS.MAX_HALT_PROPOSALS_PER_GUARDIAN_PER_EPOCH
    ) {
      return {
        accepted: false,
        reason: "Guardian has reached proposal limit for this epoch",
      };
    }

    // Anti-abuse: global limit
    if (
      this.epochProposalCount >= EMERGENCY_PARAMS.MAX_HALT_PROPOSALS_PER_EPOCH
    ) {
      return {
        accepted: false,
        reason: "Maximum halt proposals per epoch reached",
      };
    }

    // Anti-abuse: cooldown
    if (
      this.lastProposalBlock > 0n &&
      currentBlock - this.lastProposalBlock <
        EMERGENCY_PARAMS.HALT_PROPOSAL_COOLDOWN_BLOCKS
    ) {
      return {
        accepted: false,
        reason: "Proposal cooldown period not elapsed",
      };
    }

    // Create ceremony
    this.activeCeremony = createHaltCeremony(proposal, currentBlock);
    this.emergencyStatus = "halt_voting";

    // Track anti-abuse
    const existing = this.proposalCounts.get(proposal.proposer) || {
      count: 0,
      lastBlock: 0n,
    };
    this.proposalCounts.set(proposal.proposer, {
      count: existing.count + 1,
      lastBlock: currentBlock,
    });
    this.epochProposalCount++;
    this.lastProposalBlock = currentBlock;

    return { accepted: true };
  }

  // ============ Revert Proposal ============

  proposeRevert(
    proposal: EmergencyRevertProposal,
    validatorSet: ValidatorSet,
    currentBlock: bigint,
    council?: CouncilContext,
  ): CeremonyResult {
    // Must be halted
    if (this.emergencyStatus !== "halted") {
      return {
        accepted: false,
        reason: `Cannot propose revert: network is in '${this.emergencyStatus}' state (must be 'halted')`,
      };
    }

    // No active ceremony
    if (this.activeCeremony !== null) {
      return {
        accepted: false,
        reason: "Another emergency ceremony is already active",
      };
    }

    // Proposer must be Guardian (H-5: council accepted during bootstrap)
    if (!isGuardian(proposal.proposer, validatorSet, council)) {
      return { accepted: false, reason: "Not a Guardian-tier validator" };
    }

    // Minimum absolute Guardian stake floor (H-5: skip when council active)
    const totalGuardianStake = getGuardianStake(validatorSet, council);
    if (
      !council &&
      totalGuardianStake < EMERGENCY_PARAMS.MIN_GUARDIAN_STAKE_FOR_EMERGENCY
    ) {
      return {
        accepted: false,
        reason: `Insufficient Guardian stake: ${totalGuardianStake} < ${EMERGENCY_PARAMS.MIN_GUARDIAN_STAKE_FOR_EMERGENCY} minimum`,
      };
    }

    // Must reference the active halt ceremony
    if (proposal.haltCeremonyId !== this.activeHaltCeremonyId) {
      return {
        accepted: false,
        reason: "Revert must reference the active halt ceremony",
      };
    }

    // Create ceremony
    this.activeCeremony = createRevertCeremony(proposal, currentBlock);
    this.emergencyStatus = "revert_voting";

    return { accepted: true };
  }

  // ============ Resume Proposal ============

  proposeResume(
    proposal: EmergencyResumeProposal,
    validatorSet: ValidatorSet,
    currentBlock: bigint,
    council?: CouncilContext,
  ): CeremonyResult {
    // Must be halted
    if (this.emergencyStatus !== "halted") {
      return {
        accepted: false,
        reason: `Cannot propose resume: network is in '${this.emergencyStatus}' state (must be 'halted')`,
      };
    }

    // No active ceremony
    if (this.activeCeremony !== null) {
      return {
        accepted: false,
        reason: "Another emergency ceremony is already active",
      };
    }

    // Proposer must be Guardian (H-5: council accepted during bootstrap)
    if (!isGuardian(proposal.proposer, validatorSet, council)) {
      return { accepted: false, reason: "Not a Guardian-tier validator" };
    }

    // Minimum absolute Guardian stake floor (H-5: skip when council active)
    const totalGuardianStake = getGuardianStake(validatorSet, council);
    if (
      !council &&
      totalGuardianStake < EMERGENCY_PARAMS.MIN_GUARDIAN_STAKE_FOR_EMERGENCY
    ) {
      return {
        accepted: false,
        reason: `Insufficient Guardian stake: ${totalGuardianStake} < ${EMERGENCY_PARAMS.MIN_GUARDIAN_STAKE_FOR_EMERGENCY} minimum`,
      };
    }

    // Must reference the active halt ceremony
    if (proposal.haltCeremonyId !== this.activeHaltCeremonyId) {
      return {
        accepted: false,
        reason: "Resume must reference the active halt ceremony",
      };
    }

    // Create ceremony
    this.activeCeremony = createResumeCeremony(proposal, currentBlock);
    this.emergencyStatus = "resume_voting";

    return { accepted: true };
  }

  // ============ Voting ============

  handlePrevote(
    voter: string,
    vote: "yes" | "no",
    validatorSet: ValidatorSet,
    currentBlock: bigint,
    council?: CouncilContext,
  ): CeremonyResult {
    if (!this.activeCeremony) {
      return { accepted: false, reason: "No active emergency ceremony" };
    }

    const result = addPrevote(
      this.activeCeremony,
      { voter, vote },
      validatorSet,
      currentBlock,
      council,
    );

    this.activeCeremony = result.ceremony;
    return { accepted: result.accepted, reason: result.reason };
  }

  handlePrecommit(
    voter: string,
    validatorSet: ValidatorSet,
    currentBlock: bigint,
    council?: CouncilContext,
  ): CeremonyResult {
    if (!this.activeCeremony) {
      return { accepted: false, reason: "No active emergency ceremony" };
    }

    const result = addPrecommit(
      this.activeCeremony,
      { voter },
      validatorSet,
      currentBlock,
      council,
    );

    this.activeCeremony = result.ceremony;
    return { accepted: result.accepted, reason: result.reason };
  }

  // ============ Block Processing ============

  /**
   * Process block: check ceremony timeouts and phase transitions.
   * Returns a finalized ceremony result if a ceremony just completed.
   */
  processBlock(
    validatorSet: ValidatorSet,
    currentBlock: bigint,
    council?: CouncilContext,
  ): FinalizedCeremonyResult | null {
    if (!this.activeCeremony) return null;

    // Check progress and advance phases
    this.activeCeremony = checkCeremonyProgress(
      this.activeCeremony,
      validatorSet,
      currentBlock,
      council,
    );

    // Handle finalization
    if (this.activeCeremony.phase === "finalized") {
      const result: FinalizedCeremonyResult = {
        type: this.activeCeremony.type,
        ceremonyId: this.activeCeremony.id,
        ceremony: this.activeCeremony,
      };

      this.completedCeremonies.push(this.activeCeremony);

      // Update status based on ceremony type
      switch (this.activeCeremony.type) {
        case "halt":
          this.emergencyStatus = "halted";
          this.activeHaltCeremonyId = this.activeCeremony.id;
          break;
        case "revert":
          // Stays halted after revert — resume ceremony required.
          // NOTE: This is a governance signal, not automated state rollback.
          // Actual state reversion requires out-of-band validator coordination.
          this.emergencyStatus = "halted";
          break;
        case "resume":
          this.emergencyStatus = "normal";
          this.activeHaltCeremonyId = null;
          break;
      }

      this.activeCeremony = null;
      return result;
    }

    // Handle failure
    if (this.activeCeremony.phase === "failed") {
      this.completedCeremonies.push(this.activeCeremony);

      // Revert status to pre-ceremony state
      switch (this.activeCeremony.type) {
        case "halt":
          this.emergencyStatus = "normal";
          break;
        case "revert":
        case "resume":
          this.emergencyStatus = "halted";
          break;
      }

      this.activeCeremony = null;
      return null;
    }

    return null;
  }

  // ============ Epoch Reset ============

  /**
   * Reset per-epoch anti-abuse counters.
   * Call on epoch boundaries.
   */
  resetEpochCounters(): void {
    this.proposalCounts.clear();
    this.epochProposalCount = 0;
  }
}
