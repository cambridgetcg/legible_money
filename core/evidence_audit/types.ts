/**
 * Evidence Audit Review Types
 *
 * Lightweight evidence quality auditing system.
 * Auditors stake a small amount, score evidence reliability (0-1),
 * and receive rewards or slashing based on eventual accuracy.
 *
 * Fills the gap between passive acceptance and adversarial disputes:
 *   - Dispute system: 111 LGM stake, 8-phase, high-stakes
 *   - Audit system:    22 LGM stake, single tx, low-barrier
 */

import type { DID } from "../agents/identity/did.js";

// ============ Audit Status ============

export type AuditStatus =
  | "pending" // Audit submitted, awaiting resolution
  | "accurate" // Resolved: auditor score matched ground truth
  | "inaccurate" // Resolved: auditor score diverged from ground truth
  | "slashed" // Resolved: auditor lost portion of stake
  | "rewarded" // Resolved: auditor received reward + stake back
  | "expired"; // Evidence removed before resolution

// ============ Resolution Reason ============

export type AuditResolutionReason =
  | "fact_survived" // Fact survived past vesting cliff
  | "fact_challenged" // Fact challenged via dispute
  | "fact_overturned"; // Fact falsified/overturned

// ============ Core Types ============

export interface EvidenceAudit {
  /** Deterministic ID: sha512_256("audit:" + auditor + ":" + evidenceId + ":" + block) */
  id: string;

  /** Evidence being audited */
  evidenceId: string;

  /** Auditor DID */
  auditor: DID;

  /** Stake locked for this audit */
  stake: bigint;

  /** Auditor's reliability score (0.0-1.0) */
  score: number;

  /** Brief rationale for the score */
  rationale: string;

  /** Block when audit was submitted */
  submittedAtBlock: bigint;

  /** Current audit status */
  status: AuditStatus;

  /** Block when resolved (if resolved) */
  resolvedAtBlock?: bigint;

  /** Reward amount (if rewarded) */
  rewardAmount?: bigint;

  /** Slash amount (if slashed) */
  slashAmount?: bigint;

  /** Accuracy delta from ground truth (set at resolution) */
  accuracyDelta?: number;
}

// ============ Aggregated Score ============

export interface AuditConsensus {
  /** Evidence ID */
  evidenceId: string;

  /** Stake-weighted average score */
  weightedScore: number;

  /** Total stake across all audits */
  totalStake: bigint;

  /** Number of audits */
  auditCount: number;

  /** Individual audit IDs */
  auditIds: string[];

  /** Standard deviation of scores (0 = perfect agreement) */
  scoreDeviation: number;

  /** Block of most recent audit */
  lastAuditBlock: bigint;
}

// ============ Transaction Payloads ============

export interface EvidenceAuditSubmitPayload {
  type: "evidence_audit_submit";

  /** Evidence to audit */
  evidenceId: string;

  /** Stake amount (>= MIN_AUDIT_STAKE) */
  stake: bigint;

  /** Reliability score (0.0 - 1.0) */
  score: number;

  /** Rationale for score (min MIN_RATIONALE_LENGTH chars) */
  rationale: string;
}

export interface EvidenceAuditResolvePayload {
  type: "evidence_audit_resolve";

  /** Evidence ID to resolve audits for */
  evidenceId: string;

  /** Ground truth score (from fact survival/falsification) */
  groundTruthScore: number;

  /** Resolution reason */
  reason: AuditResolutionReason;
}

// ============ Anti-Gaming ============

export const AUDIT_ANTI_GAMING = {
  /** Cannot audit evidence you submitted */
  SELF_AUDIT_BLOCKED: true,

  /** Cannot audit same evidence twice */
  DUPLICATE_AUDIT_BLOCKED: true,

  /** Minimum score variance required (prevent all-0.5 spam) */
  MIN_SCORE_VARIANCE: 0.01,

  /** Maximum audits per auditor per day (BLOCKS_PER_DAY = 34272) */
  MAX_DAILY_AUDITS: 22,

  /** Minimum time between audits for same auditor (blocks) */
  COOLDOWN_BLOCKS: 222,
} as const;
