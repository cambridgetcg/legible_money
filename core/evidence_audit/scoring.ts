/**
 * Evidence Audit Scoring Engine
 *
 * Handles score aggregation, accuracy calculation, and reward determination.
 * Uses basis-point arithmetic for bigint precision (VULN-016 pattern from reward_router.ts).
 */

import type { DID } from "../agents/identity/did.js";
import type { EvidenceAudit, AuditConsensus } from "./types.js";
import { EVIDENCE_AUDIT } from "../constants.js";

// ============ Score Aggregation ============

/**
 * Aggregate audit scores into stake-weighted consensus
 *
 * Each auditor's score is weighted by their stake. This means
 * auditors who stake more have proportionally more influence
 * on the consensus score.
 */
export function aggregateAuditScores(audits: EvidenceAudit[]): AuditConsensus {
  if (audits.length === 0) {
    return {
      evidenceId: "",
      weightedScore: 0,
      totalStake: 0n,
      auditCount: 0,
      auditIds: [],
      scoreDeviation: 0,
      lastAuditBlock: 0n,
    };
  }

  const evidenceId = audits[0].evidenceId;
  let totalStake = 0n;
  let weightedSum = 0;
  let lastBlock = 0n;

  // First pass: compute stake-weighted average
  for (const audit of audits) {
    totalStake += audit.stake;
    if (audit.submittedAtBlock > lastBlock) {
      lastBlock = audit.submittedAtBlock;
    }
  }

  if (totalStake === 0n) {
    return {
      evidenceId,
      weightedScore: 0,
      totalStake: 0n,
      auditCount: audits.length,
      auditIds: audits.map((a) => a.id),
      scoreDeviation: 0,
      lastAuditBlock: lastBlock,
    };
  }

  // Stake-weighted average: sum(score_i * stake_i) / totalStake
  for (const audit of audits) {
    const weight = Number(audit.stake) / Number(totalStake);
    weightedSum += audit.score * weight;
  }

  // Second pass: compute weighted standard deviation
  let varianceSum = 0;
  for (const audit of audits) {
    const weight = Number(audit.stake) / Number(totalStake);
    const diff = audit.score - weightedSum;
    varianceSum += weight * diff * diff;
  }
  const scoreDeviation = Math.sqrt(varianceSum);

  return {
    evidenceId,
    weightedScore: weightedSum,
    totalStake,
    auditCount: audits.length,
    auditIds: audits.map((a) => a.id),
    scoreDeviation,
    lastAuditBlock: lastBlock,
  };
}

// ============ Accuracy Calculation ============

/**
 * Calculate accuracy delta between auditor score and ground truth
 */
export function calculateAccuracy(
  auditScore: number,
  groundTruthScore: number,
): { delta: number; isAccurate: boolean } {
  const delta = Math.abs(auditScore - groundTruthScore);
  // Use small epsilon for floating-point comparison at tolerance boundary
  const epsilon = 1e-10;
  return {
    delta,
    isAccurate: delta <= EVIDENCE_AUDIT.ACCURACY_TOLERANCE + epsilon,
  };
}

// ============ Reward Calculation ============

/**
 * Calculate reward for an accurate audit
 *
 * Reward = BASE_AUDIT_REWARD * accuracyMultiplier * earlyMultiplier
 *   - accuracyMultiplier: 1.0 at exact match, 0.5 at tolerance edge
 *   - earlyMultiplier: up to 1.5x for audits within EARLY_AUDIT_WINDOW
 *   - capped at MAX_ACCURACY_MULTIPLIER (2.0x)
 *
 * Uses basis-point arithmetic for bigint precision.
 */
export function calculateAuditReward(
  audit: EvidenceAudit,
  groundTruthScore: number,
  evidenceSubmittedAtBlock: bigint,
): bigint {
  const { delta, isAccurate } = calculateAccuracy(
    audit.score,
    groundTruthScore,
  );

  if (!isAccurate) {
    return 0n;
  }

  const base = EVIDENCE_AUDIT.BASE_AUDIT_REWARD;

  // Accuracy multiplier: perfect = 1.0, tolerance edge = 0.5
  const accuracyMultiplier =
    1.0 - (delta / EVIDENCE_AUDIT.ACCURACY_TOLERANCE) * 0.5;

  // Early bonus: audits within EARLY_AUDIT_WINDOW get up to 1.5x
  const blocksAfterEvidence = Number(
    audit.submittedAtBlock - evidenceSubmittedAtBlock,
  );
  let earlyMultiplier = 1.0;
  if (
    blocksAfterEvidence >= 0 &&
    blocksAfterEvidence < EVIDENCE_AUDIT.EARLY_AUDIT_WINDOW
  ) {
    const earlyFraction =
      1.0 - blocksAfterEvidence / EVIDENCE_AUDIT.EARLY_AUDIT_WINDOW;
    earlyMultiplier =
      1.0 + (EVIDENCE_AUDIT.EARLY_AUDIT_MULTIPLIER - 1.0) * earlyFraction;
  }

  // Combined multiplier, capped
  const totalMultiplier = Math.min(
    accuracyMultiplier * earlyMultiplier,
    EVIDENCE_AUDIT.MAX_ACCURACY_MULTIPLIER,
  );

  // Basis-point math for bigint precision (10000 bps = 1.0x)
  const multiplierBps = BigInt(Math.round(totalMultiplier * 10000));
  return (base * multiplierBps) / 10000n;
}

// ============ Eligibility ============

/**
 * Check if an auditor is eligible to audit a given evidence item
 */
export function isEligibleAuditor(
  auditorDid: DID,
  evidenceSubmitter: DID,
  auditorAccountAge: bigint,
  isValidator: boolean,
  existingAuditsByAuditor: EvidenceAudit[],
  currentBlock: bigint,
): { eligible: boolean; reason?: string } {
  // Self-audit check
  if (auditorDid === evidenceSubmitter) {
    return { eligible: false, reason: "Cannot audit own evidence" };
  }

  // Account age OR validator check
  if (
    !isValidator &&
    auditorAccountAge < EVIDENCE_AUDIT.MIN_AUDITOR_ACCOUNT_AGE
  ) {
    return {
      eligible: false,
      reason: "Account too young and not a validator",
    };
  }

  // Cooldown check
  const pendingAudits = existingAuditsByAuditor
    .filter((a) => a.status === "pending")
    .sort((a, b) => Number(b.submittedAtBlock - a.submittedAtBlock));

  if (pendingAudits.length > 0) {
    const blocksSinceLastAudit = Number(
      currentBlock - pendingAudits[0].submittedAtBlock,
    );
    if (blocksSinceLastAudit < EVIDENCE_AUDIT.AUDIT_COOLDOWN_BLOCKS) {
      return { eligible: false, reason: "Audit cooldown not elapsed" };
    }
  }

  return { eligible: true };
}

/**
 * Check if cooldown has elapsed since last audit
 */
export function hasAuditCooldownElapsed(
  lastAuditBlock: bigint,
  currentBlock: bigint,
): boolean {
  return (
    Number(currentBlock - lastAuditBlock) >=
    EVIDENCE_AUDIT.AUDIT_COOLDOWN_BLOCKS
  );
}
