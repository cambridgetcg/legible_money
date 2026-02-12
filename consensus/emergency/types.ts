/**
 * Emergency Kill Switch Types
 *
 * Types for the 3-phase BFT emergency protocol:
 *   NORMAL → [Guardian 75% vote] → HALTED → [Guardian 80% vote] → REVERTED → [Guardian 80% vote] → RESUMED → NORMAL
 *
 * Only Tier-3 Guardian validators can participate.
 */

// ============ Status ============

/**
 * Network emergency status.
 * Stored on StateManager to gate transaction processing.
 */
export type EmergencyStatus =
  | "normal"
  | "halt_voting"
  | "halted"
  | "revert_voting"
  | "reverting"
  | "resume_voting";

// ============ Proposals ============

/**
 * Halt proposal — initiated by any Guardian to stop the network.
 */
export interface EmergencyHaltProposal {
  id: string; // domainHash of proposal data
  proposer: string; // Guardian public key
  proposerSignature: string; // Domain-separated signature
  reason: string; // Human-readable exploit description
  evidenceHash: string; // Hash of supporting evidence
  proposedAtBlock: bigint;
  suggestedRevertBlock?: bigint; // Optional: suggested revert target
}

/**
 * Revert proposal — only valid while halted.
 * Proposes reverting state to a specific block height.
 *
 * NOTE: This is a GOVERNANCE SIGNAL, not an automated state rollback.
 * The target fields record the guardians' consensus on what to revert to,
 * but actual state reversion requires out-of-band validator coordination
 * (stop nodes → rollback → restart with new genesis).
 */
export interface EmergencyRevertProposal {
  id: string;
  proposer: string;
  proposerSignature: string;
  targetBlockNumber: bigint; // Block to revert to
  targetBlockHash: string; // Must match known block hash
  targetStateRoot: string; // Expected state root after revert
  targetSnapshotId?: string; // If reverting via snapshot
  justification: string;
  haltCeremonyId: string; // Must reference successful halt
}

/**
 * Resume proposal — only valid while halted (with or without prior revert).
 */
export interface EmergencyResumeProposal {
  id: string;
  proposer: string;
  proposerSignature: string;
  resumeFromBlock: bigint;
  resumeFromHash: string;
  resumeStateRoot: string;
  haltCeremonyId: string;
  revertCeremonyId?: string;
}

// ============ Ceremony ============

export type EmergencyCeremonyType = "halt" | "revert" | "resume";

export type EmergencyCeremonyPhase =
  | "propose"
  | "prevote"
  | "precommit"
  | "finalized"
  | "failed";

export interface EmergencyVote {
  voter: string; // Guardian public key
  vote: "yes" | "no";
}

export interface EmergencyPrecommit {
  voter: string;
}

/**
 * BFT ceremony for emergency operations.
 * Mirrors the CheckpointCeremony pattern with guardian-only participation.
 */
export interface EmergencyCeremony<
  P = EmergencyHaltProposal | EmergencyRevertProposal | EmergencyResumeProposal,
> {
  id: string;
  type: EmergencyCeremonyType;
  proposal: P;
  phase: EmergencyCeremonyPhase;
  startBlock: bigint;
  prevoteDeadline: bigint;
  precommitDeadline: bigint;
  timeoutDeadline: bigint;
  prevotes: Map<string, EmergencyVote>;
  precommits: Map<string, EmergencyPrecommit>;
  yesPrevoteStake: bigint;
  noPrevoteStake: bigint;
  precommitStake: bigint;
  failureReason?: string;
}

// ============ Results ============

export interface CeremonyResult {
  accepted: boolean;
  reason?: string;
}

export type FinalizedCeremonyResult = {
  type: EmergencyCeremonyType;
  ceremonyId: string;
  ceremony: EmergencyCeremony;
};

// ============ Audit ============

export type EmergencyAuditAction =
  | "halt_proposed"
  | "halt_prevote"
  | "halt_precommit"
  | "halt_finalized"
  | "halt_failed"
  | "halt_executed"
  | "revert_proposed"
  | "revert_prevote"
  | "revert_precommit"
  | "revert_finalized"
  | "revert_failed"
  | "revert_executed"
  | "resume_proposed"
  | "resume_prevote"
  | "resume_precommit"
  | "resume_finalized"
  | "resume_failed"
  | "resume_executed";

/**
 * Immutable audit log entry for emergency actions.
 * Stored on StateManager for transparency.
 */
export interface EmergencyAuditEntry {
  timestamp: number;
  blockNumber: bigint;
  action: EmergencyAuditAction;
  actor: string;
  ceremonyId: string;
  details: string;
}

// ============ Transaction Payloads ============

export interface EmergencyHaltProposePayload {
  type: "emergency_halt_propose";
  reason: string;
  evidenceHash: string;
  suggestedRevertBlock?: bigint;
}

export interface EmergencyHaltVotePayload {
  type: "emergency_halt_vote";
  ceremonyId: string;
  vote: "yes" | "no";
  phase: "prevote" | "precommit";
}

export interface EmergencyRevertProposePayload {
  type: "emergency_revert_propose";
  targetBlockNumber: bigint;
  targetBlockHash: string;
  targetStateRoot: string;
  targetSnapshotId?: string;
  justification: string;
  haltCeremonyId: string;
}

export interface EmergencyRevertVotePayload {
  type: "emergency_revert_vote";
  ceremonyId: string;
  vote: "yes" | "no";
  phase: "prevote" | "precommit";
}

export interface EmergencyResumeProposePayload {
  type: "emergency_resume_propose";
  resumeFromBlock: bigint;
  resumeFromHash: string;
  resumeStateRoot: string;
  haltCeremonyId: string;
  revertCeremonyId?: string;
}

export interface EmergencyResumeVotePayload {
  type: "emergency_resume_vote";
  ceremonyId: string;
  vote: "yes" | "no";
  phase: "prevote" | "precommit";
}

export type EmergencyTransactionPayload =
  | EmergencyHaltProposePayload
  | EmergencyHaltVotePayload
  | EmergencyRevertProposePayload
  | EmergencyRevertVotePayload
  | EmergencyResumeProposePayload
  | EmergencyResumeVotePayload;

// ============ Helpers ============

const EMERGENCY_TX_TYPES = new Set([
  "emergency_halt_propose",
  "emergency_halt_vote",
  "emergency_revert_propose",
  "emergency_revert_vote",
  "emergency_resume_propose",
  "emergency_resume_vote",
]);

/**
 * Check if a transaction type is an emergency transaction.
 * Emergency transactions are accepted even when the network is halted.
 */
export function isEmergencyTransaction(txType: string): boolean {
  return EMERGENCY_TX_TYPES.has(txType);
}
