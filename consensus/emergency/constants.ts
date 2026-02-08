/**
 * Emergency Kill Switch Constants
 *
 * Quorum thresholds, ceremony timing, anti-abuse limits, and domain separation.
 * All timing values are in blocks (2521ms per block).
 */

export const EMERGENCY_PARAMS = {
  // ============ Quorum Thresholds ============
  // Stricter than normal BFT 0.67 threshold — guardian-stake-weighted

  /** 75% of Guardian stake required to halt */
  GUARDIAN_HALT_QUORUM: 0.75,
  /** 80% of Guardian stake required to revert (more destructive = stricter) */
  GUARDIAN_REVERT_QUORUM: 0.8,
  /** 80% of Guardian stake required to resume (prevents premature restart) */
  GUARDIAN_RESUME_QUORUM: 0.8,

  /**
   * Minimum absolute Guardian stake for emergency ceremonies to function.
   * Prevents early-network attacks where few Guardians could collude.
   * 111,111 LGM = at minimum ~10 Guardians at minimum stake (11,111 each).
   */
  MIN_GUARDIAN_STAKE_FOR_EMERGENCY: 111111n * 10n ** 18n,

  /**
   * Minimum number of distinct Guardian validators required to finalize.
   * Even if stake threshold is met, a single whale can't act alone.
   * Requires at least 4 distinct Guardian prevotes for any ceremony.
   */
  MIN_GUARDIAN_VOTERS: 4,

  // ============ Ceremony Timing (blocks @ 2521ms each) ============

  // Halt ceremony — fast for urgency
  /** ~28s prevote window */
  HALT_PREVOTE_BLOCKS: 11n,
  /** ~28s precommit window */
  HALT_PRECOMMIT_BLOCKS: 11n,
  /** ~111s total ceremony timeout */
  HALT_TIMEOUT_BLOCKS: 44n,

  // Revert ceremony — more deliberation required
  /** ~55s prevote window */
  REVERT_PREVOTE_BLOCKS: 22n,
  /** ~55s precommit window */
  REVERT_PRECOMMIT_BLOCKS: 22n,
  /** ~280s total ceremony timeout */
  REVERT_TIMEOUT_BLOCKS: 111n,

  // Resume ceremony — equal deliberation to revert
  /** ~55s prevote window */
  RESUME_PREVOTE_BLOCKS: 22n,
  /** ~55s precommit window */
  RESUME_PRECOMMIT_BLOCKS: 22n,
  /** ~280s total ceremony timeout */
  RESUME_TIMEOUT_BLOCKS: 111n,

  // ============ Anti-Abuse ============

  /** Max halt proposals per epoch (global) */
  MAX_HALT_PROPOSALS_PER_EPOCH: 3,
  /** Max halt proposals per guardian per epoch */
  MAX_HALT_PROPOSALS_PER_GUARDIAN_PER_EPOCH: 1,
  /** ~4.7 min cooldown between any proposals */
  HALT_PROPOSAL_COOLDOWN_BLOCKS: 111n,

  // ============ Revert Constraints ============

  /** ~3.2 days max revert depth */
  MAX_EMERGENCY_REVERT_BLOCKS: 111111n,
  /** ~55s mandatory observation delay before revert executes */
  REVERT_OBSERVATION_BLOCKS: 22n,

  // ============ Auto-Snapshot ============

  /** Automatically snapshot state when halt is executed */
  AUTO_SNAPSHOT_ON_HALT: true,

  // ============ Domain Separation ============

  DOMAIN_HALT_PROPOSAL: "LGM.emergency.halt.proposal.v1",
  DOMAIN_HALT_VOTE: "LGM.emergency.halt.vote.v1",
  DOMAIN_REVERT_PROPOSAL: "LGM.emergency.revert.proposal.v1",
  DOMAIN_REVERT_VOTE: "LGM.emergency.revert.vote.v1",
  DOMAIN_RESUME_PROPOSAL: "LGM.emergency.resume.proposal.v1",
  DOMAIN_RESUME_VOTE: "LGM.emergency.resume.vote.v1",
} as const;
