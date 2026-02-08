/**
 * Evidence Audit Transaction Tests
 *
 * Tests for evidence_audit_submit and evidence_audit_resolve transaction handlers.
 * Validates the complete lifecycle: submit audit → lock stake → resolve → reward/slash.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import {
  executeTransaction,
  FEE_POOL_DID,
  REWARD_POOL_DID,
  PROTOCOL_TREASURY_DID,
} from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import { EVIDENCE_AUDIT, TREASURY } from "../../../core/constants.js";
import type { Evidence } from "../../../core/evidence/types.js";
import type { DID } from "../../../core/agents/identity/did.js";

// ============ Test Helpers ============

const AUDITOR_DID = "did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1" as DID;
const SUBMITTER_DID = "did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb" as DID;
const EVIDENCE_ID = "evidence:test-audit-001";
const BLOCK = 10000n;

function createTestParams(): ConsensusParams {
  return {
    targetBlockTimeMs: 2521,
    minValidatorStake: 1000n * 10n ** 18n,
    minClaimStake: 10n * 10n ** 18n,
    validatorQuorum: 0.66,
    verificationRoundBlocks: 20,
    commitPhaseBlocks: 6,
    revealPhaseBlocks: 6,
    epochBlocks: 100,
    maxValidators: 100,
    slashFractionDowntime: 0.01,
    slashFractionDoubleSign: 0.05,
    unbondingPeriod: 1000n,
  };
}

function createState(): StateManager {
  const state = new StateManager();

  // Create accounts (must use createAccount for validateTransaction to pass)
  state.createAccount(
    AUDITOR_DID,
    "auditor-key-hash",
    1000n * 10n ** 18n, // 1000 LGM
  );
  state.createAccount(SUBMITTER_DID, "submitter-key-hash", 100n * 10n ** 18n);

  // Protocol accounts
  state.createAccount(FEE_POOL_DID, "fee-pool-hash", 0n);
  state.createAccount(
    REWARD_POOL_DID,
    "reward-pool-hash",
    1_000_000n * 10n ** 18n, // 1M LGM
  );
  state.createAccount(PROTOCOL_TREASURY_DID, "treasury-hash", 0n);

  // Treasury DIDs for splitEvidenceTax routing
  state.createAccount(TREASURY.ADDRESSES.RESEARCH, "research-hash", 0n);
  state.createAccount(TREASURY.ADDRESSES.FOUNDATION, "foundation-hash", 0n);
  state.createAccount(TREASURY.ADDRESSES.COMMUNITY, "community-hash", 0n);
  state.createAccount(TREASURY.ADDRESSES.DEVELOPERS, "developers-hash", 0n);
  state.createAccount(TREASURY.ADDRESSES.RESERVE, "reserve-hash", 0n);

  // Create evidence to audit
  const evidence: Evidence = {
    id: EVIDENCE_ID,
    type: "peer_reviewed",
    originalSource: "https://arxiv.org/abs/test",
    contentHash: "abc123",
    archives: [],
    submittedAt: Date.now(),
    submittedAtBlock: 5000n,
    firstSubmitter: SUBMITTER_DID,
    confirmedBy: [],
    status: "archived",
    direction: "supporting",
    intrinsicStrength: 0.8,
    reliabilityScore: 0.75,
    independence: { score: 0.7, sources: 2, independentConfirmations: 1 },
    version: {
      versionNumber: 1,
      previousVersionHash: "",
      versionChain: [],
      createdAt: Date.now(),
      confidenceImpact: 0,
    },
    expiry: {
      isExpired: false,
      expiryType: "none",
      freshnessScore: 1.0,
    },
    metadata: {
      title: "Test evidence",
      description: "Test evidence for audit tests",
      tags: [],
    },
    relatedFactIds: ["fact:test-001"],
  } as Evidence;

  state.submitEvidence(evidence, "test evidence content");

  return state;
}

function makeAuditSubmitTx(
  overrides: Record<string, unknown> = {},
  senderDid: DID = AUDITOR_DID,
): Transaction {
  return {
    type: "evidence_audit_submit",
    sender: senderDid,
    senderDid,
    payload: {
      evidenceId: EVIDENCE_ID,
      stake: EVIDENCE_AUDIT.MIN_AUDIT_STAKE,
      score: 0.8,
      rationale:
        "This evidence is from a reputable peer-reviewed source with strong methodology and clear documentation.",
      ...overrides,
    },
    fee: 10000n,
    nonce: 0n,
    timestamp: Date.now(),
    signature: "test-signature",
  } as unknown as Transaction;
}

function makeAuditResolveTx(
  groundTruthScore: number,
  reason: string = "fact_survived",
): Transaction {
  return {
    type: "evidence_audit_resolve",
    sender: "system",
    payload: {
      evidenceId: EVIDENCE_ID,
      groundTruthScore,
      reason,
    },
    fee: 0n,
    nonce: 0n,
    timestamp: Date.now(),
    signature: "system",
    systemTx: true,
  } as unknown as Transaction;
}

// ============ evidence_audit_submit ============

describe.sequential("evidence_audit_submit", () => {
  let state: StateManager;
  let params: ConsensusParams;

  beforeEach(() => {
    state = createState();
    params = createTestParams();
  });

  it("should accept a valid audit submission", () => {
    const tx = makeAuditSubmitTx();
    const result = executeTransaction(state, tx, BLOCK, params);

    expect(result.success).toBe(true);
  });

  it("should lock auditor stake on submission", () => {
    const balanceBefore = state.getBalance(AUDITOR_DID);
    const tx = makeAuditSubmitTx();
    executeTransaction(state, tx, BLOCK, params);

    // Stake should be locked (moved from balance to staked)
    const balanceAfter = state.getBalance(AUDITOR_DID);
    expect(balanceBefore - balanceAfter).toBeGreaterThanOrEqual(
      EVIDENCE_AUDIT.MIN_AUDIT_STAKE,
    );
  });

  it("should store the audit record", () => {
    const tx = makeAuditSubmitTx();
    executeTransaction(state, tx, BLOCK, params);

    const audits = state.getEvidenceAudits(EVIDENCE_ID);
    expect(audits.length).toBe(1);
    expect(audits[0].score).toBe(0.8);
    expect(audits[0].status).toBe("pending");
    expect(audits[0].auditor).toBe(AUDITOR_DID);
  });

  it("should reject stake below minimum", () => {
    const tx = makeAuditSubmitTx({ stake: 1n * 10n ** 18n }); // 1 LGM < 22 LGM
    const result = executeTransaction(state, tx, BLOCK, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("Minimum audit stake");
  });

  it("should reject score out of range (negative)", () => {
    const tx = makeAuditSubmitTx({ score: -0.1 });
    const result = executeTransaction(state, tx, BLOCK, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("score");
  });

  it("should reject score out of range (above 1)", () => {
    const tx = makeAuditSubmitTx({ score: 1.5 });
    const result = executeTransaction(state, tx, BLOCK, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("score");
  });

  it("should reject short rationale", () => {
    const tx = makeAuditSubmitTx({ rationale: "too short" });
    const result = executeTransaction(state, tx, BLOCK, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("Rationale");
  });

  it("should reject audit of non-existent evidence", () => {
    const tx = makeAuditSubmitTx({ evidenceId: "evidence:does-not-exist" });
    const result = executeTransaction(state, tx, BLOCK, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should reject self-audit", () => {
    // Create tx where sender is the evidence submitter
    const tx = makeAuditSubmitTx({}, SUBMITTER_DID);
    const result = executeTransaction(state, tx, BLOCK, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("own evidence");
  });

  it("should reject duplicate audit of same evidence", () => {
    // First audit succeeds
    const tx1 = makeAuditSubmitTx();
    const result1 = executeTransaction(state, tx1, BLOCK, params);
    expect(result1.success).toBe(true);

    // Second audit from same auditor should fail
    const tx2 = makeAuditSubmitTx();
    tx2.nonce = 1n;
    const result2 = executeTransaction(state, tx2, BLOCK + 300n, params);
    expect(result2.success).toBe(false);
    expect(result2.error).toContain("Already audited");
  });

  it("should reject when insufficient balance", () => {
    // Drain auditor balance first
    state.deductBalance(AUDITOR_DID, 999n * 10n ** 18n);

    const tx = makeAuditSubmitTx();
    const result = executeTransaction(state, tx, BLOCK, params);

    expect(result.success).toBe(false);
  });
});

// ============ evidence_audit_resolve ============

describe.sequential("evidence_audit_resolve", () => {
  let state: StateManager;
  let params: ConsensusParams;

  beforeEach(() => {
    state = createState();
    params = createTestParams();
    // Submit an audit first
    const submitTx = makeAuditSubmitTx();
    const result = executeTransaction(state, submitTx, BLOCK, params);
    expect(result.success).toBe(true);
  });

  it("should reward accurate auditors (exact match)", () => {
    const balanceBefore = state.getBalance(AUDITOR_DID);
    const resolveTx = makeAuditResolveTx(0.8); // Ground truth matches audit score
    const result = executeTransaction(state, resolveTx, BLOCK + 1000n, params);

    expect(result.success).toBe(true);

    // Auditor should have more than before (stake returned + reward)
    const balanceAfter = state.getBalance(AUDITOR_DID);
    expect(balanceAfter).toBeGreaterThan(balanceBefore);

    // Audit should be marked as rewarded
    const audits = state.getEvidenceAudits(EVIDENCE_ID);
    expect(audits[0].status).toBe("rewarded");
    expect(audits[0].rewardAmount).toBeGreaterThan(0n);
  });

  it("should slash inaccurate auditors", () => {
    // Balance before resolve (post-submit: stake + fee already deducted)
    const balancePostSubmit = state.getBalance(AUDITOR_DID);

    const resolveTx = makeAuditResolveTx(0.2); // Ground truth far from 0.8 audit score
    const result = executeTransaction(state, resolveTx, BLOCK + 1000n, params);

    expect(result.success).toBe(true);

    // After resolve: stake returned minus 33% slash
    // Net = balancePostSubmit + stake - slashAmount
    // slashAmount = 33% of stake, so auditor loses ~7.26 LGM from original 1000 LGM
    const balanceAfter = state.getBalance(AUDITOR_DID);

    // Auditor gets stake back minus slash, so balanceAfter > balancePostSubmit
    // but total lost (fee + slash) means less than original 1000 LGM
    expect(balanceAfter).toBeLessThan(1000n * 10n ** 18n);

    // Audit should be marked as slashed
    const audits = state.getEvidenceAudits(EVIDENCE_ID);
    expect(audits[0].status).toBe("slashed");
    expect(audits[0].slashAmount).toBeGreaterThan(0n);
  });

  it("should reject non-system transaction", () => {
    const resolveTx = {
      ...makeAuditResolveTx(0.8),
      sender: AUDITOR_DID,
      senderDid: AUDITOR_DID,
      fee: 10000n,
      nonce: 1n, // After submit, auditor nonce is 1
      systemTx: undefined,
    } as unknown as Transaction;
    const result = executeTransaction(state, resolveTx, BLOCK + 1000n, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("system-only");
  });

  it("should reject when no pending audits exist", () => {
    // Resolve once (success)
    const resolve1 = makeAuditResolveTx(0.8);
    executeTransaction(state, resolve1, BLOCK + 1000n, params);

    // Try to resolve again — no more pending audits
    const resolve2 = makeAuditResolveTx(0.8);
    const result = executeTransaction(state, resolve2, BLOCK + 2000n, params);

    expect(result.success).toBe(false);
    expect(result.error).toContain("No pending audits");
  });

  it("should record accuracy delta on resolution", () => {
    const resolveTx = makeAuditResolveTx(0.75); // delta = 0.05 from 0.8
    executeTransaction(state, resolveTx, BLOCK + 1000n, params);

    const audits = state.getEvidenceAudits(EVIDENCE_ID);
    expect(audits[0].accuracyDelta).toBeCloseTo(0.05, 5);
  });
});
