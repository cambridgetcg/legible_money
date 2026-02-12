/**
 * Emergency Kill Switch Tests
 *
 * Tests the 3-phase BFT emergency protocol:
 *   NORMAL → HALTED → (optionally REVERTED) → RESUMED → NORMAL
 *
 * Covers: happy paths, quorum failures, non-guardian rejection,
 * anti-abuse limits, ceremony timeouts, halt guard, audit logging.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  getGuardianStake,
  isGuardian,
  getGuardianEffectiveStake,
  createHaltCeremony,
  createRevertCeremony,
  createResumeCeremony,
  addPrevote,
  addPrecommit,
  checkCeremonyProgress,
} from "../../../consensus/emergency/ceremony.js";
import { EmergencyCeremonyManager } from "../../../consensus/emergency/manager.js";
import { EMERGENCY_PARAMS } from "../../../consensus/emergency/constants.js";
import { executeHalt } from "../../../consensus/emergency/halt_executor.js";
import { executeRevert } from "../../../consensus/emergency/revert_executor.js";
import { executeResume } from "../../../consensus/emergency/resume_executor.js";
import { isEmergencyTransaction } from "../../../consensus/emergency/types.js";
import type {
  EmergencyHaltProposal,
  EmergencyRevertProposal,
  EmergencyResumeProposal,
  EmergencyCeremony,
} from "../../../consensus/emergency/types.js";
import type { Validator, ValidatorSet } from "../../../consensus/pot/types.js";
import { StateManager } from "../../../state/state_manager.js";
import { domainHash } from "../../../core/crypto/hashing.js";
import { bytesToHex } from "../../../core/crypto/keys.js";

// ============ Test Helpers ============

function createGuardianValidator(
  publicKey: string,
  stake: bigint = 25000n * 10n ** 15n,
  isActive: boolean = true,
): Validator {
  return {
    publicKey,
    did: `did:lgm:${publicKey.slice(0, 32)}`,
    stake,
    delegatedStake: 0n,
    reputation: 1.0,
    isActive,
    lastActiveBlock: 0,
    totalVerifications: 2000,
    correctVerifications: 1800,
    missedVerifications: 0,
    slashCount: 0,
    tier: 3, // Guardian
  };
}

function createTier2Validator(
  publicKey: string,
  stake: bigint = 5000n * 10n ** 15n,
): Validator {
  return {
    publicKey,
    did: `did:lgm:${publicKey.slice(0, 32)}`,
    stake,
    delegatedStake: 0n,
    reputation: 0.9,
    isActive: true,
    lastActiveBlock: 0,
    totalVerifications: 100,
    correctVerifications: 90,
    missedVerifications: 0,
    slashCount: 0,
    tier: 2, // Bonded (not Guardian)
  };
}

function createValidatorSet(validators: Validator[]): ValidatorSet {
  const map = new Map<string, Validator>();
  let totalStake = 0n;
  let activeCount = 0;
  for (const v of validators) {
    map.set(v.publicKey, v);
    totalStake += v.stake + v.delegatedStake;
    if (v.isActive) activeCount++;
  }
  return { validators: map, totalStake, activeCount, epoch: 1n };
}

function makeHaltProposal(
  proposer: string,
  block: bigint = 100n,
): EmergencyHaltProposal {
  const id = bytesToHex(
    domainHash(
      EMERGENCY_PARAMS.DOMAIN_HALT_PROPOSAL,
      `halt-${proposer}-${block}`,
    ),
  );
  return {
    id,
    proposer,
    proposerSignature: "sig_" + proposer,
    reason: "Critical exploit detected: unauthorized token minting",
    evidenceHash: "evidence_hash_abc123",
    proposedAtBlock: block,
  };
}

function makeRevertProposal(
  proposer: string,
  haltCeremonyId: string,
  targetBlock: bigint = 50n,
): EmergencyRevertProposal {
  const id = bytesToHex(
    domainHash(
      EMERGENCY_PARAMS.DOMAIN_REVERT_PROPOSAL,
      `revert-${proposer}-${targetBlock}`,
    ),
  );
  return {
    id,
    proposer,
    proposerSignature: "sig_" + proposer,
    targetBlockNumber: targetBlock,
    targetBlockHash: "target_hash_abc",
    targetStateRoot: "target_state_root_abc",
    justification: "Revert to pre-exploit state",
    haltCeremonyId,
  };
}

function makeResumeProposal(
  proposer: string,
  haltCeremonyId: string,
  stateRoot: string,
  resumeBlock: bigint = 50n,
): EmergencyResumeProposal {
  const id = bytesToHex(
    domainHash(
      EMERGENCY_PARAMS.DOMAIN_RESUME_PROPOSAL,
      `resume-${proposer}-${resumeBlock}`,
    ),
  );
  return {
    id,
    proposer,
    proposerSignature: "sig_" + proposer,
    resumeFromBlock: resumeBlock,
    resumeFromHash: "resume_hash_abc",
    resumeStateRoot: stateRoot,
    haltCeremonyId,
  };
}

/**
 * Drive a ceremony to finalization by having guardians vote and checking progress.
 */
function driveToFinalization(
  manager: EmergencyCeremonyManager,
  guardianKeys: string[],
  validatorSet: ValidatorSet,
  startBlock: bigint,
): boolean {
  // Prevote: all guardians vote yes
  for (const key of guardianKeys) {
    manager.handlePrevote(
      key,
      "yes",
      `sig_${key}`,
      validatorSet,
      startBlock + 1n,
    );
  }

  // Check progress to advance to precommit
  let result = manager.processBlock(validatorSet, startBlock + 2n);
  if (result) return true; // Finalized in prevote? Shouldn't happen but handle it.

  // Precommit: all guardians precommit
  for (const key of guardianKeys) {
    manager.handlePrecommit(
      key,
      `precommit_${key}`,
      validatorSet,
      startBlock + 3n,
    );
  }

  // Check progress to finalize
  result = manager.processBlock(validatorSet, startBlock + 4n);
  return result !== null;
}

// ============ Tests ============

describe("Emergency Kill Switch", () => {
  const G1 = "guardian_1_pubkey_aaaa";
  const G2 = "guardian_2_pubkey_bbbb";
  const G3 = "guardian_3_pubkey_cccc";
  const G4 = "guardian_4_pubkey_dddd";
  const G5 = "guardian_5_pubkey_eeee";
  const T2 = "tier2_validator_pubkey";

  let guardians: Validator[];
  let validatorSet: ValidatorSet;
  let manager: EmergencyCeremonyManager;

  beforeEach(() => {
    guardians = [
      createGuardianValidator(G1, 25000n * 10n ** 18n),
      createGuardianValidator(G2, 25000n * 10n ** 18n),
      createGuardianValidator(G3, 25000n * 10n ** 18n),
      createGuardianValidator(G4, 25000n * 10n ** 18n),
      createGuardianValidator(G5, 25000n * 10n ** 18n),
    ];
    validatorSet = createValidatorSet([...guardians, createTier2Validator(T2)]);
    manager = new EmergencyCeremonyManager();
  });

  // ============ Ceremony Helpers ============

  describe("Guardian helpers", () => {
    it("should compute total guardian stake", () => {
      const stake = getGuardianStake(validatorSet);
      expect(stake).toBe(125000n * 10n ** 15n); // 5 guardians × 25000
    });

    it("should identify guardians", () => {
      expect(isGuardian(G1, validatorSet)).toBe(true);
      expect(isGuardian(T2, validatorSet)).toBe(false);
      expect(isGuardian("nonexistent", validatorSet)).toBe(false);
    });

    it("should get guardian effective stake", () => {
      expect(getGuardianEffectiveStake(G1, validatorSet)).toBe(
        25000n * 10n ** 18n,
      );
      expect(getGuardianEffectiveStake(T2, validatorSet)).toBe(0n);
    });

    it("should exclude inactive guardians", () => {
      const inactiveGuardian = createGuardianValidator(
        "inactive_g",
        30000n * 10n ** 18n,
        false,
      );
      const vs = createValidatorSet([...guardians, inactiveGuardian]);
      expect(getGuardianStake(vs)).toBe(125000n * 10n ** 15n); // inactive excluded
    });
  });

  // ============ isEmergencyTransaction ============

  describe("isEmergencyTransaction", () => {
    it("should identify emergency transaction types", () => {
      expect(isEmergencyTransaction("emergency_halt_propose")).toBe(true);
      expect(isEmergencyTransaction("emergency_halt_vote")).toBe(true);
      expect(isEmergencyTransaction("emergency_revert_propose")).toBe(true);
      expect(isEmergencyTransaction("emergency_revert_vote")).toBe(true);
      expect(isEmergencyTransaction("emergency_resume_propose")).toBe(true);
      expect(isEmergencyTransaction("emergency_resume_vote")).toBe(true);
    });

    it("should reject non-emergency types", () => {
      expect(isEmergencyTransaction("transfer")).toBe(false);
      expect(isEmergencyTransaction("stake")).toBe(false);
      expect(isEmergencyTransaction("claim_submit")).toBe(false);
    });
  });

  // ============ Happy Path: Halt ============

  describe("Happy path: halt", () => {
    it("should halt with 80% guardian vote (4/5)", () => {
      const proposal = makeHaltProposal(G1, 100n);
      const result = manager.proposeHalt(proposal, validatorSet, 100n);
      expect(result.accepted).toBe(true);
      expect(manager.getEmergencyStatus()).toBe("halt_voting");

      // 4 of 5 guardians prevote yes (80% > 75% threshold)
      for (const key of [G1, G2, G3, G4]) {
        const r = manager.handlePrevote(
          key,
          "yes",
          `sig_${key}`,
          validatorSet,
          101n,
        );
        expect(r.accepted).toBe(true);
      }

      // Advance to precommit
      manager.processBlock(validatorSet, 102n);
      expect(manager.getActiveCeremony()?.phase).toBe("precommit");

      // 4 of 5 guardians precommit
      for (const key of [G1, G2, G3, G4]) {
        const r = manager.handlePrecommit(
          key,
          `precommit_${key}`,
          validatorSet,
          103n,
        );
        expect(r.accepted).toBe(true);
      }

      // Finalize
      const finalized = manager.processBlock(validatorSet, 104n);
      expect(finalized).not.toBeNull();
      expect(finalized!.type).toBe("halt");
      expect(manager.getEmergencyStatus()).toBe("halted");
      expect(manager.isHalted()).toBe(true);
    });

    it("should record the halt ceremony ID", () => {
      const proposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal, validatorSet, 100n);
      driveToFinalization(manager, [G1, G2, G3, G4], validatorSet, 100n);
      expect(manager.getActiveHaltCeremonyId()).toBe(proposal.id);
    });
  });

  // ============ Happy Path: Revert ============

  describe("Happy path: revert", () => {
    it("should revert after halt with 80% guardian vote", () => {
      // First: halt
      const haltProposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(haltProposal, validatorSet, 100n);
      driveToFinalization(manager, [G1, G2, G3, G4], validatorSet, 100n);
      expect(manager.getEmergencyStatus()).toBe("halted");

      // Now: propose revert
      const revertProposal = makeRevertProposal(G2, haltProposal.id, 50n);
      const result = manager.proposeRevert(revertProposal, validatorSet, 200n);
      expect(result.accepted).toBe(true);
      expect(manager.getEmergencyStatus()).toBe("revert_voting");

      // Vote and finalize
      driveToFinalization(manager, [G1, G2, G3, G4, G5], validatorSet, 200n);
      // After revert finalization, status returns to "halted" (resume needed)
      expect(manager.getEmergencyStatus()).toBe("halted");
    });
  });

  // ============ Happy Path: Resume ============

  describe("Happy path: resume", () => {
    it("should resume after halt with 80% guardian vote", () => {
      // Halt
      const haltProposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(haltProposal, validatorSet, 100n);
      driveToFinalization(manager, [G1, G2, G3, G4], validatorSet, 100n);

      // Resume (skip revert)
      const state = new StateManager();
      const stateRoot = state.getStateRoot();
      const resumeProposal = makeResumeProposal(
        G3,
        haltProposal.id,
        stateRoot,
        100n,
      );
      const result = manager.proposeResume(resumeProposal, validatorSet, 200n);
      expect(result.accepted).toBe(true);
      expect(manager.getEmergencyStatus()).toBe("resume_voting");

      // Vote and finalize
      driveToFinalization(manager, [G1, G2, G3, G4, G5], validatorSet, 200n);
      expect(manager.getEmergencyStatus()).toBe("normal");
      expect(manager.isHalted()).toBe(false);
      expect(manager.getActiveHaltCeremonyId()).toBeNull();
    });
  });

  // ============ Full Cycle ============

  describe("Full cycle: halt → revert → resume", () => {
    it("should complete a full emergency cycle", () => {
      // Phase 1: Halt
      const haltProposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(haltProposal, validatorSet, 100n);
      driveToFinalization(manager, [G1, G2, G3, G4], validatorSet, 100n);
      expect(manager.getEmergencyStatus()).toBe("halted");

      // Phase 2: Revert
      const revertProposal = makeRevertProposal(G2, haltProposal.id, 50n);
      manager.proposeRevert(revertProposal, validatorSet, 200n);
      driveToFinalization(manager, [G1, G2, G3, G4, G5], validatorSet, 200n);
      expect(manager.getEmergencyStatus()).toBe("halted"); // Still halted after revert

      // Phase 3: Resume
      const state = new StateManager();
      const stateRoot = state.getStateRoot();
      const resumeProposal = makeResumeProposal(
        G3,
        haltProposal.id,
        stateRoot,
        50n,
      );
      manager.proposeResume(resumeProposal, validatorSet, 300n);
      driveToFinalization(manager, [G1, G2, G3, G4, G5], validatorSet, 300n);
      expect(manager.getEmergencyStatus()).toBe("normal");

      // Verify completed ceremonies tracked
      expect(manager.getCompletedCeremonies()).toHaveLength(3);
    });
  });

  // ============ Insufficient Quorum ============

  describe("Insufficient quorum", () => {
    it("should fail halt with 40% guardian vote (2/5)", () => {
      const proposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal, validatorSet, 100n);

      // Only 2 of 5 vote yes
      manager.handlePrevote(G1, "yes", "sig_g1", validatorSet, 101n);
      manager.handlePrevote(G2, "yes", "sig_g2", validatorSet, 101n);
      // 3 vote no
      manager.handlePrevote(G3, "no", "sig_g3", validatorSet, 101n);
      manager.handlePrevote(G4, "no", "sig_g4", validatorSet, 101n);
      manager.handlePrevote(G5, "no", "sig_g5", validatorSet, 101n);

      // Process: 60% no > 25% (1 - 0.75), should fail immediately
      const result = manager.processBlock(validatorSet, 102n);
      expect(result).toBeNull(); // No finalization
      expect(manager.getEmergencyStatus()).toBe("normal"); // Reverted to normal
      expect(manager.getActiveCeremony()).toBeNull();
    });
  });

  // ============ Non-Guardian Rejected ============

  describe("Non-guardian rejection", () => {
    it("should reject halt proposal from tier-2 validator", () => {
      const proposal = makeHaltProposal(T2, 100n);
      const result = manager.proposeHalt(proposal, validatorSet, 100n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Not a Guardian-tier validator");
    });

    it("should reject prevote from tier-2 validator", () => {
      const proposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal, validatorSet, 100n);

      const result = manager.handlePrevote(
        T2,
        "yes",
        "sig_t2",
        validatorSet,
        101n,
      );
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Not a Guardian-tier validator");
    });

    it("should reject precommit from tier-2 validator", () => {
      const proposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal, validatorSet, 100n);

      // Get to precommit phase first
      for (const key of [G1, G2, G3, G4]) {
        manager.handlePrevote(key, "yes", `sig_${key}`, validatorSet, 101n);
      }
      manager.processBlock(validatorSet, 102n);

      const result = manager.handlePrecommit(
        T2,
        "precommit_t2",
        validatorSet,
        103n,
      );
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Not a Guardian-tier validator");
    });
  });

  // ============ Anti-Abuse: Per-Guardian Limit ============

  describe("Anti-abuse: per-guardian limit", () => {
    it("should reject second halt proposal from same guardian in same epoch", () => {
      // First proposal: propose and let it fail via timeout
      const proposal1 = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal1, validatorSet, 100n);

      // Timeout the ceremony
      manager.processBlock(
        validatorSet,
        100n + EMERGENCY_PARAMS.HALT_TIMEOUT_BLOCKS + 1n,
      );
      expect(manager.getActiveCeremony()).toBeNull();

      // Second proposal from same guardian
      const proposal2 = makeHaltProposal(G1, 300n);
      const result = manager.proposeHalt(proposal2, validatorSet, 300n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("proposal limit");
    });
  });

  // ============ Anti-Abuse: Cooldown ============

  describe("Anti-abuse: cooldown", () => {
    it("should reject proposal within cooldown period", () => {
      // First proposal
      const proposal1 = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal1, validatorSet, 100n);

      // Timeout
      manager.processBlock(
        validatorSet,
        100n + EMERGENCY_PARAMS.HALT_TIMEOUT_BLOCKS + 1n,
      );

      // Second proposal too soon (from different guardian, but cooldown is global)
      const proposal2 = makeHaltProposal(G2, 110n); // Only 10 blocks later
      const result = manager.proposeHalt(proposal2, validatorSet, 110n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("cooldown");
    });

    it("should accept proposal after cooldown elapses", () => {
      const proposal1 = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal1, validatorSet, 100n);

      // Timeout
      manager.processBlock(
        validatorSet,
        100n + EMERGENCY_PARAMS.HALT_TIMEOUT_BLOCKS + 1n,
      );

      // After cooldown (from different guardian since G1 used their limit)
      const proposal2 = makeHaltProposal(
        G2,
        100n + EMERGENCY_PARAMS.HALT_PROPOSAL_COOLDOWN_BLOCKS + 1n,
      );
      const result = manager.proposeHalt(
        proposal2,
        validatorSet,
        100n + EMERGENCY_PARAMS.HALT_PROPOSAL_COOLDOWN_BLOCKS + 1n,
      );
      expect(result.accepted).toBe(true);
    });
  });

  // ============ Ceremony Timeout ============

  describe("Ceremony timeout", () => {
    it("should fail halt ceremony on timeout", () => {
      const proposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal, validatorSet, 100n);

      // Only 1 prevote (not enough)
      manager.handlePrevote(G1, "yes", "sig_g1", validatorSet, 101n);

      // Advance past timeout
      const result = manager.processBlock(
        validatorSet,
        100n + EMERGENCY_PARAMS.HALT_TIMEOUT_BLOCKS + 1n,
      );
      expect(result).toBeNull();
      expect(manager.getActiveCeremony()).toBeNull();
      expect(manager.getEmergencyStatus()).toBe("normal");
    });

    it("should fail prevote phase on deadline", () => {
      const proposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal, validatorSet, 100n);

      // Only 2 prevotes (not enough for 75%)
      manager.handlePrevote(G1, "yes", "sig_g1", validatorSet, 101n);
      manager.handlePrevote(G2, "yes", "sig_g2", validatorSet, 101n);

      // Past prevote deadline but before total timeout
      const result = manager.processBlock(
        validatorSet,
        100n + EMERGENCY_PARAMS.HALT_PREVOTE_BLOCKS + 1n,
      );
      expect(result).toBeNull();
      expect(manager.getActiveCeremony()).toBeNull();
    });
  });

  // ============ Resume Requires Halt ============

  describe("Resume requires halt", () => {
    it("should reject resume proposal when network is normal", () => {
      const state = new StateManager();
      const proposal = makeResumeProposal(
        G1,
        "fake_halt_id",
        state.getStateRoot(),
      );
      const result = manager.proposeResume(proposal, validatorSet, 100n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("must be 'halted'");
    });
  });

  // ============ Revert Requires Halt ============

  describe("Revert requires halt", () => {
    it("should reject revert proposal when network is normal", () => {
      const proposal = makeRevertProposal(G1, "fake_halt_id", 50n);
      const result = manager.proposeRevert(proposal, validatorSet, 100n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("must be 'halted'");
    });
  });

  // ============ Halted Blocks Non-Emergency Txs ============

  describe("Halt guard on StateManager", () => {
    it("should report halted state correctly", () => {
      const state = new StateManager();
      expect(state.isHalted()).toBe(false);

      state.setEmergencyStatus("halted");
      expect(state.isHalted()).toBe(true);

      state.setEmergencyStatus("revert_voting");
      expect(state.isHalted()).toBe(true);

      state.setEmergencyStatus("reverting");
      expect(state.isHalted()).toBe(true);

      state.setEmergencyStatus("resume_voting");
      expect(state.isHalted()).toBe(true);

      state.setEmergencyStatus("normal");
      expect(state.isHalted()).toBe(false);

      state.setEmergencyStatus("halt_voting");
      expect(state.isHalted()).toBe(false); // Voting phase, not yet halted
    });
  });

  // ============ Halt Executor ============

  describe("Halt executor", () => {
    it("should execute halt and set state", () => {
      const state = new StateManager();
      const proposal = makeHaltProposal(G1, 100n);
      const ceremony = createHaltCeremony(proposal, 100n);
      ceremony.phase = "finalized";

      const result = executeHalt(state, ceremony, 100n);
      expect(result.success).toBe(true);
      expect(state.getEmergencyStatus()).toBe("halted");
      expect(state.getLastHaltBlock()).toBe(100n);

      // Audit log
      const log = state.getEmergencyAuditLog();
      expect(log).toHaveLength(1);
      expect(log[0].action).toBe("halt_executed");
      expect(log[0].ceremonyId).toBe(proposal.id);
    });
  });

  // ============ Revert Executor ============

  describe("Revert executor", () => {
    it("should execute revert when halted", () => {
      const state = new StateManager();
      state.setEmergencyStatus("halted");
      state.setLastHaltBlock(100n);

      const proposal = makeRevertProposal(G2, "halt_ceremony_id", 50n);
      const ceremony = createRevertCeremony(proposal, 200n);
      ceremony.phase = "finalized";

      const result = executeRevert(state, ceremony, 200n);
      expect(result.success).toBe(true);
      expect(result.targetBlock).toBe(50n);
      // After revert completes, status returns to halted
      expect(state.getEmergencyStatus()).toBe("halted");

      const log = state.getEmergencyAuditLog();
      expect(log).toHaveLength(1);
      expect(log[0].action).toBe("revert_executed");
    });

    it("should reject revert when not halted", () => {
      const state = new StateManager();
      // Status is "normal"

      const proposal = makeRevertProposal(G2, "halt_ceremony_id", 50n);
      const ceremony = createRevertCeremony(proposal, 200n);
      ceremony.phase = "finalized";

      const result = executeRevert(state, ceremony, 200n);
      expect(result.success).toBe(false);
      expect(result.error).toContain("Cannot revert");
    });

    it("should reject revert exceeding max depth", () => {
      const state = new StateManager();
      state.setEmergencyStatus("halted");
      state.setLastHaltBlock(200000n);

      // Try to revert way too far back
      const proposal = makeRevertProposal(G2, "halt_ceremony_id", 1n);
      const ceremony = createRevertCeremony(proposal, 200000n);
      ceremony.phase = "finalized";

      const result = executeRevert(state, ceremony, 200000n);
      expect(result.success).toBe(false);
      expect(result.error).toContain("exceeds maximum");
    });

    it("should allow deep revert within limit (50K blocks)", () => {
      const state = new StateManager();
      state.setEmergencyStatus("halted");
      state.setLastHaltBlock(100000n);

      const proposal = makeRevertProposal(G2, "halt_ceremony_id", 50000n);
      const ceremony = createRevertCeremony(proposal, 100000n);
      ceremony.phase = "finalized";

      const result = executeRevert(state, ceremony, 100000n);
      expect(result.success).toBe(true); // 50K depth within 111,111 limit
    });
  });

  // ============ Resume Executor ============

  describe("Resume executor", () => {
    it("should execute resume when halted with matching state root", () => {
      const state = new StateManager();
      state.setEmergencyStatus("halted");
      state.setHaltSnapshotId("snapshot_123");

      const stateRoot = state.getStateRoot();
      const proposal = makeResumeProposal(
        G3,
        "halt_ceremony_id",
        stateRoot,
        100n,
      );
      const ceremony = createResumeCeremony(proposal, 200n);
      ceremony.phase = "finalized";

      const result = executeResume(state, ceremony, 200n);
      expect(result.success).toBe(true);
      expect(state.getEmergencyStatus()).toBe("normal");
      expect(state.getHaltSnapshotId()).toBeNull();

      const log = state.getEmergencyAuditLog();
      expect(log).toHaveLength(1);
      expect(log[0].action).toBe("resume_executed");
    });

    it("should reject resume when not halted", () => {
      const state = new StateManager();
      const proposal = makeResumeProposal(G3, "halt_id", "fake_root");
      const ceremony = createResumeCeremony(proposal, 200n);
      ceremony.phase = "finalized";

      const result = executeResume(state, ceremony, 200n);
      expect(result.success).toBe(false);
      expect(result.error).toContain("Cannot resume");
    });

    it("should reject resume with mismatched state root", () => {
      const state = new StateManager();
      state.setEmergencyStatus("halted");

      const proposal = makeResumeProposal(G3, "halt_id", "wrong_state_root");
      const ceremony = createResumeCeremony(proposal, 200n);
      ceremony.phase = "finalized";

      const result = executeResume(state, ceremony, 200n);
      expect(result.success).toBe(false);
      expect(result.error).toContain("State root mismatch");
    });
  });

  // ============ Ceremony Phase Mechanics ============

  describe("Ceremony phase mechanics", () => {
    it("should reject duplicate prevotes", () => {
      const proposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal, validatorSet, 100n);

      manager.handlePrevote(G1, "yes", "sig_g1", validatorSet, 101n);
      const dup = manager.handlePrevote(
        G1,
        "yes",
        "sig_g1_again",
        validatorSet,
        101n,
      );
      expect(dup.accepted).toBe(false);
      expect(dup.reason).toContain("Already voted");
    });

    it("should reject precommit without prior yes prevote", () => {
      const proposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal, validatorSet, 100n);

      // Get to precommit phase
      for (const key of [G1, G2, G3, G4]) {
        manager.handlePrevote(key, "yes", `sig_${key}`, validatorSet, 101n);
      }
      manager.processBlock(validatorSet, 102n);

      // G5 didn't prevote yes, try to precommit
      const result = manager.handlePrecommit(
        G5,
        "precommit_g5",
        validatorSet,
        103n,
      );
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Must prevote 'yes' before precommit");
    });

    it("should reject prevote in wrong phase", () => {
      const proposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal, validatorSet, 100n);

      // Advance to precommit
      for (const key of [G1, G2, G3, G4]) {
        manager.handlePrevote(key, "yes", `sig_${key}`, validatorSet, 101n);
      }
      manager.processBlock(validatorSet, 102n);

      // Try prevote in precommit phase
      const result = manager.handlePrevote(
        G5,
        "yes",
        "sig_g5",
        validatorSet,
        103n,
      );
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Not in prevote phase");
    });

    it("should handle no active guardians", () => {
      // Create a validator set with no guardians
      const emptySet = createValidatorSet([createTier2Validator("t2_only")]);
      const ceremony = createHaltCeremony(makeHaltProposal(G1, 100n), 100n);

      const updated = checkCeremonyProgress(ceremony, emptySet, 101n);
      expect(updated.phase).toBe("failed");
      expect(updated.failureReason).toContain("No active Guardian");
    });
  });

  // ============ Active Ceremony Guard ============

  describe("Active ceremony guard", () => {
    it("should reject second halt proposal while ceremony active", () => {
      const proposal1 = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal1, validatorSet, 100n);

      const proposal2 = makeHaltProposal(G2, 200n);
      const result = manager.proposeHalt(proposal2, validatorSet, 200n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("halt_voting");
    });
  });

  // ============ Revert Must Reference Halt ============

  describe("Revert must reference correct halt ceremony", () => {
    it("should reject revert with wrong halt ceremony ID", () => {
      const haltProposal = makeHaltProposal(G1, 100n);
      manager.proposeHalt(haltProposal, validatorSet, 100n);
      driveToFinalization(manager, [G1, G2, G3, G4], validatorSet, 100n);

      const revertProposal = makeRevertProposal(G2, "wrong_halt_id", 50n);
      const result = manager.proposeRevert(revertProposal, validatorSet, 200n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("reference the active halt ceremony");
    });
  });

  // ============ Epoch Reset ============

  describe("Epoch reset", () => {
    it("should reset anti-abuse counters on epoch change", () => {
      const proposal1 = makeHaltProposal(G1, 100n);
      manager.proposeHalt(proposal1, validatorSet, 100n);
      manager.processBlock(
        validatorSet,
        100n + EMERGENCY_PARAMS.HALT_TIMEOUT_BLOCKS + 1n,
      );

      // G1 used their per-guardian limit
      const proposal2 = makeHaltProposal(G1, 300n);
      const result1 = manager.proposeHalt(proposal2, validatorSet, 300n);
      expect(result1.accepted).toBe(false);

      // Reset epoch counters
      manager.resetEpochCounters();

      // G1 can propose again
      const proposal3 = makeHaltProposal(G1, 500n);
      const result2 = manager.proposeHalt(proposal3, validatorSet, 500n);
      expect(result2.accepted).toBe(true);
    });
  });

  // ============ Audit Log Completeness ============

  describe("Audit log completeness", () => {
    it("should record all audit entries across full cycle", () => {
      const state = new StateManager();

      // Halt
      const haltProposal = makeHaltProposal(G1, 100n);
      const haltCeremony = createHaltCeremony(haltProposal, 100n);
      haltCeremony.phase = "finalized";
      executeHalt(state, haltCeremony, 100n);

      // Revert
      state.setLastHaltBlock(100n);
      const revertProposal = makeRevertProposal(G2, haltProposal.id, 50n);
      const revertCeremony = createRevertCeremony(revertProposal, 200n);
      revertCeremony.phase = "finalized";
      executeRevert(state, revertCeremony, 200n);

      // Resume
      const stateRoot = state.getStateRoot();
      const resumeProposal = makeResumeProposal(
        G3,
        haltProposal.id,
        stateRoot,
        50n,
      );
      const resumeCeremony = createResumeCeremony(resumeProposal, 300n);
      resumeCeremony.phase = "finalized";
      executeResume(state, resumeCeremony, 300n);

      const log = state.getEmergencyAuditLog();
      expect(log).toHaveLength(3);
      expect(log[0].action).toBe("halt_executed");
      expect(log[1].action).toBe("revert_executed");
      expect(log[2].action).toBe("resume_executed");

      // Verify all entries have required fields
      for (const entry of log) {
        expect(entry.timestamp).toBeGreaterThan(0);
        expect(entry.blockNumber).toBeGreaterThanOrEqual(0n);
        expect(entry.actor).toBeTruthy();
        expect(entry.ceremonyId).toBeTruthy();
        expect(entry.details).toBeTruthy();
      }
    });
  });

  // ============ Stake-Weighted Voting ============

  describe("Stake-weighted voting", () => {
    it("should reach quorum with fewer guardians if they have more stake", () => {
      // 2 heavy guardians + 3 light guardians
      const heavy1 = createGuardianValidator("heavy_1", 40000n * 10n ** 18n);
      const heavy2 = createGuardianValidator("heavy_2", 40000n * 10n ** 18n);
      const light1 = createGuardianValidator("light_1", 11111n * 10n ** 18n);
      const light2 = createGuardianValidator("light_2", 11111n * 10n ** 18n);
      const light3 = createGuardianValidator("light_3", 11111n * 10n ** 18n);
      const unequalSet = createValidatorSet([
        heavy1,
        heavy2,
        light1,
        light2,
        light3,
      ]);

      // Total guardian stake: 80000 + 33333 = 113333
      // 75% quorum = ~85000
      // Just the 2 heavy guardians = 80000/113333 = ~70.6% — not enough
      // 2 heavy + 1 light = 91111/113333 = ~80.4% — enough

      const mgr = new EmergencyCeremonyManager();
      const proposal = makeHaltProposal("heavy_1", 100n);
      mgr.proposeHalt(proposal, unequalSet, 100n);

      // 2 heavy vote yes
      mgr.handlePrevote("heavy_1", "yes", "sig_h1", unequalSet, 101n);
      mgr.handlePrevote("heavy_2", "yes", "sig_h2", unequalSet, 101n);

      // Not enough yet
      mgr.processBlock(unequalSet, 102n);
      expect(mgr.getActiveCeremony()?.phase).toBe("prevote"); // Still in prevote

      // 1 more light guardian tips it over
      mgr.handlePrevote("light_1", "yes", "sig_l1", unequalSet, 102n);
      mgr.processBlock(unequalSet, 103n);
      expect(mgr.getActiveCeremony()?.phase).toBe("precommit"); // Advanced!
    });
  });

  // ============ Constants Sanity Checks ============

  describe("Constants sanity", () => {
    it("should have reasonable quorum thresholds", () => {
      expect(EMERGENCY_PARAMS.GUARDIAN_HALT_QUORUM).toBeGreaterThanOrEqual(
        0.67,
      );
      expect(EMERGENCY_PARAMS.GUARDIAN_HALT_QUORUM).toBeLessThanOrEqual(1.0);
      expect(EMERGENCY_PARAMS.GUARDIAN_REVERT_QUORUM).toBeGreaterThan(
        EMERGENCY_PARAMS.GUARDIAN_HALT_QUORUM,
      );
      expect(EMERGENCY_PARAMS.GUARDIAN_RESUME_QUORUM).toBe(
        EMERGENCY_PARAMS.GUARDIAN_REVERT_QUORUM,
      );
    });

    it("should have halt ceremony be faster than revert/resume", () => {
      expect(EMERGENCY_PARAMS.HALT_TIMEOUT_BLOCKS).toBeLessThan(
        EMERGENCY_PARAMS.REVERT_TIMEOUT_BLOCKS,
      );
      expect(EMERGENCY_PARAMS.HALT_PREVOTE_BLOCKS).toBeLessThan(
        EMERGENCY_PARAMS.REVERT_PREVOTE_BLOCKS,
      );
    });

    it("should have a meaningful revert depth limit", () => {
      expect(EMERGENCY_PARAMS.MAX_EMERGENCY_REVERT_BLOCKS).toBeGreaterThan(
        10000n,
      );
    });

    it("should require meaningful minimum guardian stake", () => {
      // At least ~10 minimum-stake Guardians (11,111 each = 111,111)
      expect(
        EMERGENCY_PARAMS.MIN_GUARDIAN_STAKE_FOR_EMERGENCY,
      ).toBeGreaterThanOrEqual(111111n * 10n ** 18n);
    });

    it("should require multiple distinct voters", () => {
      expect(EMERGENCY_PARAMS.MIN_GUARDIAN_VOTERS).toBeGreaterThanOrEqual(4);
    });
  });

  // ============ Minimum Guardian Stake Floor ============

  describe("Minimum guardian stake floor", () => {
    it("should reject halt proposal when total guardian stake is below minimum", () => {
      // 3 guardians at minimum stake = 33,333 LGM — well below 111,111 threshold
      const smallGuardians = [
        createGuardianValidator("sg1", 11111n * 10n ** 18n),
        createGuardianValidator("sg2", 11111n * 10n ** 18n),
        createGuardianValidator("sg3", 11111n * 10n ** 18n),
      ];
      const smallSet = createValidatorSet(smallGuardians);

      const mgr = new EmergencyCeremonyManager();
      const proposal = makeHaltProposal("sg1", 100n);
      const result = mgr.proposeHalt(proposal, smallSet, 100n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Insufficient Guardian stake");
    });

    it("should reject revert proposal when guardian stake is below minimum", () => {
      // Use a full-size set to halt, then check revert with small set
      const mgr = new EmergencyCeremonyManager();
      // Manually set halted state
      mgr.setEmergencyStatus("halted");
      (mgr as any).activeHaltCeremonyId = "test_halt_id";

      const smallGuardians = [
        createGuardianValidator("sg1", 11111n * 10n ** 18n),
        createGuardianValidator("sg2", 11111n * 10n ** 18n),
      ];
      const smallSet = createValidatorSet(smallGuardians);

      const proposal = makeRevertProposal("sg1", "test_halt_id", 50n);
      const result = mgr.proposeRevert(proposal, smallSet, 200n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Insufficient Guardian stake");
    });

    it("should reject resume proposal when guardian stake is below minimum", () => {
      const mgr = new EmergencyCeremonyManager();
      mgr.setEmergencyStatus("halted");
      (mgr as any).activeHaltCeremonyId = "test_halt_id";

      const smallGuardians = [
        createGuardianValidator("sg1", 11111n * 10n ** 18n),
      ];
      const smallSet = createValidatorSet(smallGuardians);

      const state = new StateManager();
      const proposal = makeResumeProposal(
        "sg1",
        "test_halt_id",
        state.getStateRoot(),
      );
      const result = mgr.proposeResume(proposal, smallSet, 200n);
      expect(result.accepted).toBe(false);
      expect(result.reason).toContain("Insufficient Guardian stake");
    });

    it("should accept halt when guardian stake meets minimum", () => {
      // 10 guardians at minimum = 111,110 — just below threshold
      // 11 guardians at minimum = 122,221 — above threshold
      const enoughGuardians = [];
      for (let i = 0; i < 11; i++) {
        enoughGuardians.push(
          createGuardianValidator(`g_${i}`, 11111n * 10n ** 18n),
        );
      }
      const enoughSet = createValidatorSet(enoughGuardians);

      const mgr = new EmergencyCeremonyManager();
      const proposal = makeHaltProposal("g_0", 100n);
      const result = mgr.proposeHalt(proposal, enoughSet, 100n);
      expect(result.accepted).toBe(true);
    });

    it("should fail ceremony at progress check if guardians unstake below minimum", () => {
      const proposal = makeHaltProposal(G1, 100n);
      const ceremony = createHaltCeremony(proposal, 100n);

      // Small guardian set — below threshold
      const tinySet = createValidatorSet([
        createGuardianValidator("tiny1", 11111n * 10n ** 18n),
      ]);

      const updated = checkCeremonyProgress(ceremony, tinySet, 101n);
      expect(updated.phase).toBe("failed");
      expect(updated.failureReason).toContain("Insufficient Guardian stake");
    });
  });

  // ============ Minimum Distinct Voter Count ============

  describe("Minimum distinct voter count", () => {
    it("should not finalize with fewer than MIN_GUARDIAN_VOTERS even if stake quorum met", () => {
      // 1 whale guardian with >75% of stake + 3 small guardians
      const whale = createGuardianValidator("whale", 300000n * 10n ** 18n);
      const small1 = createGuardianValidator("small1", 25000n * 10n ** 18n);
      const small2 = createGuardianValidator("small2", 25000n * 10n ** 18n);
      const small3 = createGuardianValidator("small3", 25000n * 10n ** 18n);
      const whaleSet = createValidatorSet([whale, small1, small2, small3]);
      // Total: 375,000 LGM. Whale alone = 300K/375K = 80% > 75% threshold
      // But only 1 voter < MIN_GUARDIAN_VOTERS (4)

      const mgr = new EmergencyCeremonyManager();
      const proposal = makeHaltProposal("whale", 100n);
      mgr.proposeHalt(proposal, whaleSet, 100n);

      // Only whale prevotes
      mgr.handlePrevote("whale", "yes", "sig_whale", whaleSet, 101n);
      mgr.processBlock(whaleSet, 102n);
      // Should advance to precommit (stake quorum met)
      expect(mgr.getActiveCeremony()?.phase).toBe("precommit");

      // Only whale precommits — stake is 80% > 75% but only 1 voter
      mgr.handlePrecommit("whale", "precommit_whale", whaleSet, 103n);
      mgr.processBlock(whaleSet, 104n);
      // Should NOT finalize — still in precommit waiting for more voters
      expect(mgr.getActiveCeremony()?.phase).toBe("precommit");
    });

    it("should finalize when both stake quorum AND voter count are met", () => {
      // 4 equal guardians — each has 25% = 100% total
      const g = [
        createGuardianValidator("eq1", 30000n * 10n ** 18n),
        createGuardianValidator("eq2", 30000n * 10n ** 18n),
        createGuardianValidator("eq3", 30000n * 10n ** 18n),
        createGuardianValidator("eq4", 30000n * 10n ** 18n),
      ];
      const eqSet = createValidatorSet(g);

      const mgr = new EmergencyCeremonyManager();
      const proposal = makeHaltProposal("eq1", 100n);
      mgr.proposeHalt(proposal, eqSet, 100n);

      // All 4 prevote yes
      for (const v of g) {
        mgr.handlePrevote(
          v.publicKey,
          "yes",
          `sig_${v.publicKey}`,
          eqSet,
          101n,
        );
      }
      mgr.processBlock(eqSet, 102n);
      expect(mgr.getActiveCeremony()?.phase).toBe("precommit");

      // All 4 precommit
      for (const v of g) {
        mgr.handlePrecommit(v.publicKey, `pc_${v.publicKey}`, eqSet, 103n);
      }
      const result = mgr.processBlock(eqSet, 104n);
      expect(result).not.toBeNull();
      expect(result!.type).toBe("halt");
      expect(mgr.getEmergencyStatus()).toBe("halted");
    });

    it("should fail at deadline when stake met but voter count not", () => {
      const whale = createGuardianValidator("whale2", 300000n * 10n ** 18n);
      const small1 = createGuardianValidator("sm1", 25000n * 10n ** 18n);
      const small2 = createGuardianValidator("sm2", 25000n * 10n ** 18n);
      const small3 = createGuardianValidator("sm3", 25000n * 10n ** 18n);
      const whaleSet = createValidatorSet([whale, small1, small2, small3]);

      const mgr = new EmergencyCeremonyManager();
      const proposal = makeHaltProposal("whale2", 100n);
      mgr.proposeHalt(proposal, whaleSet, 100n);

      // Whale prevotes + precommits alone
      mgr.handlePrevote("whale2", "yes", "sig", whaleSet, 101n);
      mgr.processBlock(whaleSet, 102n); // -> precommit

      mgr.handlePrecommit("whale2", "pc", whaleSet, 103n);

      // Advance past precommit deadline
      const deadline =
        100n +
        EMERGENCY_PARAMS.HALT_PREVOTE_BLOCKS +
        EMERGENCY_PARAMS.HALT_PRECOMMIT_BLOCKS +
        1n;
      mgr.processBlock(whaleSet, deadline);

      // Should have failed
      expect(mgr.getActiveCeremony()).toBeNull();
      expect(mgr.getEmergencyStatus()).toBe("normal");

      // Check the completed ceremony has the right failure reason
      const completed = mgr.getCompletedCeremonies();
      expect(completed).toHaveLength(1);
      expect(completed[0].failureReason).toContain(
        "Insufficient distinct voters",
      );
    });
  });
});
