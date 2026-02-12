/**
 * Evidence Audit Scoring Engine Tests
 *
 * Tests for score aggregation, accuracy calculation, reward determination,
 * and auditor eligibility checks.
 */

import { describe, it, expect } from "vitest";
import {
  aggregateAuditScores,
  calculateAccuracy,
  calculateAuditReward,
  isEligibleAuditor,
  hasAuditCooldownElapsed,
} from "../../../core/evidence_audit/scoring.js";
import { EVIDENCE_AUDIT } from "../../../core/constants.js";
import type { EvidenceAudit } from "../../../core/evidence_audit/types.js";
import type { DID } from "../../../core/agents/identity/did.js";

// ============ Test Helpers ============

const TEST_AUDITOR = "did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1" as DID;
const TEST_AUDITOR_2 = "did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa2" as DID;
const TEST_SUBMITTER = "did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" as DID;
const TEST_EVIDENCE_ID = "evidence:test-001";

function makeAudit(overrides: Partial<EvidenceAudit> = {}): EvidenceAudit {
  return {
    id: "audit:test-001",
    evidenceId: TEST_EVIDENCE_ID,
    auditor: TEST_AUDITOR,
    stake: EVIDENCE_AUDIT.MIN_AUDIT_STAKE,
    score: 0.8,
    rationale:
      "This evidence appears well-sourced and independently verifiable.",
    submittedAtBlock: 1000n,
    status: "pending",
    ...overrides,
  };
}

// ============ aggregateAuditScores ============

describe.sequential("aggregateAuditScores", () => {
  it("should return empty consensus for no audits", () => {
    const result = aggregateAuditScores([]);
    expect(result.auditCount).toBe(0);
    expect(result.weightedScore).toBe(0);
    expect(result.totalStake).toBe(0n);
  });

  it("should return raw score for single audit", () => {
    const audit = makeAudit({ score: 0.75 });
    const result = aggregateAuditScores([audit]);

    expect(result.auditCount).toBe(1);
    expect(result.weightedScore).toBeCloseTo(0.75, 5);
    expect(result.totalStake).toBe(EVIDENCE_AUDIT.MIN_AUDIT_STAKE);
    expect(result.scoreDeviation).toBe(0);
  });

  it("should compute stake-weighted average for multiple audits", () => {
    const audit1 = makeAudit({
      id: "audit:1",
      score: 0.9,
      stake: 100n * 10n ** 15n, // 100 LGM
    });
    const audit2 = makeAudit({
      id: "audit:2",
      auditor: TEST_AUDITOR_2,
      score: 0.5,
      stake: 100n * 10n ** 15n, // 100 LGM
    });

    const result = aggregateAuditScores([audit1, audit2]);

    // Equal stake → simple average: (0.9 + 0.5) / 2 = 0.7
    expect(result.weightedScore).toBeCloseTo(0.7, 5);
    expect(result.auditCount).toBe(2);
    expect(result.totalStake).toBe(200n * 10n ** 15n);
  });

  it("should weight by stake when stakes differ", () => {
    const audit1 = makeAudit({
      id: "audit:1",
      score: 1.0,
      stake: 300n * 10n ** 15n, // 3x stake
    });
    const audit2 = makeAudit({
      id: "audit:2",
      auditor: TEST_AUDITOR_2,
      score: 0.0,
      stake: 100n * 10n ** 15n,
    });

    const result = aggregateAuditScores([audit1, audit2]);

    // Weighted: (1.0 * 300 + 0.0 * 100) / 400 = 0.75
    expect(result.weightedScore).toBeCloseTo(0.75, 5);
  });

  it("should track the last audit block", () => {
    const audit1 = makeAudit({ id: "a1", submittedAtBlock: 100n });
    const audit2 = makeAudit({
      id: "a2",
      auditor: TEST_AUDITOR_2,
      submittedAtBlock: 500n,
    });

    const result = aggregateAuditScores([audit1, audit2]);
    expect(result.lastAuditBlock).toBe(500n);
  });

  it("should compute non-zero standard deviation for diverging scores", () => {
    const audit1 = makeAudit({ id: "a1", score: 0.2 });
    const audit2 = makeAudit({
      id: "a2",
      auditor: TEST_AUDITOR_2,
      score: 0.8,
    });

    const result = aggregateAuditScores([audit1, audit2]);
    expect(result.scoreDeviation).toBeGreaterThan(0);
  });
});

// ============ calculateAccuracy ============

describe.sequential("calculateAccuracy", () => {
  it("should mark exact match as accurate", () => {
    const result = calculateAccuracy(0.85, 0.85);
    expect(result.delta).toBe(0);
    expect(result.isAccurate).toBe(true);
  });

  it("should mark within tolerance as accurate", () => {
    const result = calculateAccuracy(0.8, 0.9);
    expect(result.delta).toBeCloseTo(0.1, 5);
    expect(result.isAccurate).toBe(true); // 0.1 < 0.15 tolerance
  });

  it("should mark at tolerance edge as accurate", () => {
    const result = calculateAccuracy(0.7, 0.85);
    expect(result.delta).toBeCloseTo(EVIDENCE_AUDIT.ACCURACY_TOLERANCE, 10);
    expect(result.isAccurate).toBe(true);
  });

  it("should mark beyond tolerance as inaccurate", () => {
    const result = calculateAccuracy(0.5, 0.9);
    expect(result.delta).toBeCloseTo(0.4, 5);
    expect(result.isAccurate).toBe(false);
  });

  it("should handle boundary scores (0 and 1)", () => {
    expect(calculateAccuracy(0, 0).isAccurate).toBe(true);
    expect(calculateAccuracy(1, 1).isAccurate).toBe(true);
    expect(calculateAccuracy(0, 1).isAccurate).toBe(false);
  });
});

// ============ calculateAuditReward ============

describe.sequential("calculateAuditReward", () => {
  it("should return base reward for exact match, no early bonus", () => {
    const audit = makeAudit({
      score: 0.8,
      submittedAtBlock: 10000n, // Way past early window
    });

    const reward = calculateAuditReward(audit, 0.8, 1000n);

    // Exact match: accuracyMultiplier = 1.0, no early bonus
    // Reward = BASE (3 LGM) * 1.0 = 3 LGM
    expect(reward).toBe(EVIDENCE_AUDIT.BASE_AUDIT_REWARD);
  });

  it("should reduce reward near tolerance edge", () => {
    const audit = makeAudit({
      score: 0.67,
      submittedAtBlock: 10000n,
    });

    const reward = calculateAuditReward(audit, 0.8, 1000n);

    // Delta = 0.13, accuracy multiplier = 1.0 - (0.13/0.15)*0.5 ≈ 0.567
    // Reward should be between 0 and base (reduced but non-zero)
    expect(reward).toBeGreaterThan(0n);
    expect(reward).toBeLessThan(EVIDENCE_AUDIT.BASE_AUDIT_REWARD);
  });

  it("should return 0 for inaccurate audit", () => {
    const audit = makeAudit({ score: 0.1 });
    const reward = calculateAuditReward(audit, 0.9, 0n);
    expect(reward).toBe(0n);
  });

  it("should apply early audit bonus", () => {
    // Audit submitted 0 blocks after evidence → max early bonus (1.5x)
    const audit = makeAudit({
      score: 0.8,
      submittedAtBlock: 1000n,
    });

    const reward = calculateAuditReward(audit, 0.8, 1000n);

    // Exact match + immediate early: 1.0 * 1.5 = 1.5x
    // Reward = 3 LGM * 1.5 = 4.5 LGM
    const expectedBps = BigInt(Math.round(1.5 * 10000));
    const expected = (EVIDENCE_AUDIT.BASE_AUDIT_REWARD * expectedBps) / 10000n;
    expect(reward).toBe(expected);
  });

  it("should cap multiplier at MAX_ACCURACY_MULTIPLIER", () => {
    const audit = makeAudit({
      score: 0.8,
      submittedAtBlock: 1000n, // Immediate audit
    });

    const reward = calculateAuditReward(audit, 0.8, 1000n);

    // Max = 2.0x, actual = 1.5x, so no cap hit here
    const maxReward =
      (EVIDENCE_AUDIT.BASE_AUDIT_REWARD *
        BigInt(Math.round(EVIDENCE_AUDIT.MAX_ACCURACY_MULTIPLIER * 10000))) /
      10000n;
    expect(reward).toBeLessThanOrEqual(maxReward);
  });
});

// ============ isEligibleAuditor ============

describe.sequential("isEligibleAuditor", () => {
  it("should block self-audit", () => {
    const result = isEligibleAuditor(
      TEST_SUBMITTER,
      TEST_SUBMITTER, // same as submitter
      5000n,
      false,
      [],
      1000n,
    );

    expect(result.eligible).toBe(false);
    expect(result.reason).toContain("own evidence");
  });

  it("should block young non-validator accounts", () => {
    const result = isEligibleAuditor(
      TEST_AUDITOR,
      TEST_SUBMITTER,
      100n, // too young
      false,
      [],
      1000n,
    );

    expect(result.eligible).toBe(false);
    expect(result.reason).toContain("young");
  });

  it("should allow validators regardless of age", () => {
    const result = isEligibleAuditor(
      TEST_AUDITOR,
      TEST_SUBMITTER,
      1n, // very young but is validator
      true,
      [],
      1000n,
    );

    expect(result.eligible).toBe(true);
  });

  it("should block during cooldown", () => {
    const recentAudit = makeAudit({ submittedAtBlock: 990n });
    const result = isEligibleAuditor(
      TEST_AUDITOR,
      TEST_SUBMITTER,
      5000n,
      false,
      [recentAudit], // audit at block 990
      1000n, // current block 1000, delta = 10 < 222
    );

    expect(result.eligible).toBe(false);
    expect(result.reason).toContain("cooldown");
  });

  it("should allow after cooldown", () => {
    const oldAudit = makeAudit({ submittedAtBlock: 500n });
    const result = isEligibleAuditor(
      TEST_AUDITOR,
      TEST_SUBMITTER,
      5000n,
      false,
      [oldAudit], // audit at block 500
      1000n, // current block 1000, delta = 500 > 222
    );

    expect(result.eligible).toBe(true);
  });
});

// ============ hasAuditCooldownElapsed ============

describe.sequential("hasAuditCooldownElapsed", () => {
  it("should return false during cooldown", () => {
    expect(hasAuditCooldownElapsed(900n, 1000n)).toBe(false); // 100 < 222
  });

  it("should return true after cooldown", () => {
    expect(hasAuditCooldownElapsed(700n, 1000n)).toBe(true); // 300 > 222
  });

  it("should return true at exact cooldown boundary", () => {
    expect(hasAuditCooldownElapsed(778n, 1000n)).toBe(true); // 222 >= 222
  });
});
