/**
 * Evidence Audit Review Reward System
 *
 * Lightweight pathway for proactive evidence quality checks.
 * Complements the high-stakes dispute system with lower-barrier auditing.
 *
 * Usage:
 *   Submit: evidence_audit_submit transaction (22 LGM stake, score 0-1)
 *   Resolve: evidence_audit_resolve system transaction (reward/slash)
 */

// Types
export type {
  AuditStatus,
  AuditResolutionReason,
  EvidenceAudit,
  AuditConsensus,
  EvidenceAuditSubmitPayload,
  EvidenceAuditResolvePayload,
} from "./types.js";

export { AUDIT_ANTI_GAMING } from "./types.js";

// Scoring engine
export {
  aggregateAuditScores,
  calculateAccuracy,
  calculateAuditReward,
  isEligibleAuditor,
  hasAuditCooldownElapsed,
} from "./scoring.js";
