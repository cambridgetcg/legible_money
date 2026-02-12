/**
 * Dispute State Machine Tests
 *
 * Comprehensive tests for the 8-phase dispute state machine.
 *
 * Phase flow:
 * initiation → evidence_commit → evidence_reveal → arbitration
 *                                                      ↓
 *           resolved ← settlement ← deliberation ←─────┘
 *              ↑                        ↓
 *              └────── escalated ←──────┘
 */

import { describe, it, expect } from "vitest";
import {
  isValidTransition,
  canEnterEvidenceCommit,
  canEnterEvidenceReveal,
  canEnterArbitration,
  canEnterDeliberation,
  canEnterSettlement,
  canEscalate,
  advancePhase,
  escalateDispute,
  hasPhaseTimedOut,
  handleTimeout,
  determineFinalWinner,
  isTerminal,
  VALID_TRANSITIONS,
} from "../../../core/dispute/state_machine.js";
import {
  PHASE_DURATIONS,
  ESCALATION_TIERS,
  DISPUTE_ECONOMICS,
  DisputeError,
  type Dispute,
  type DisputePhase,
  type DisputeState,
} from "../../../core/dispute/types.js";
import type { Evidence } from "../../../core/evidence/types.js";

// ============ Test Helpers ============

function createMinimalDispute(
  phase: DisputePhase = "initiation",
  overrides: Partial<Dispute> = {},
): Dispute {
  const baseState: DisputeState = {
    phase,
    phaseDeadline: 1000n,
    canEscalate: phase === "deliberation",
    escalationTier: 1,
    previousTierResults: [],
  };

  return {
    id: "dispute-test-123",
    targetId: "fact-test-456",
    targetType: "fact",
    domain: "test.domain",
    challenger: {
      did: "did:lgm:challenger",
      stake: DISPUTE_ECONOMICS.minChallengeStake,
      initialEvidence: ["ev1"],
    },
    defender: {
      did: "did:lgm:defender",
      stake: DISPUTE_ECONOMICS.minChallengeStake,
      responseEvidence: ["ev2"],
    },
    state: baseState,
    commitments: {},
    arbiters: [],
    votes: new Map(),
    outcome: undefined,
    winner: undefined,
    exposedFalsehood: false,
    initiatedAtBlock: 100n,
    resolvedAtBlock: undefined,
    createdAt: Date.now(),
    updatedAt: Date.now(),
    ...overrides,
  } as Dispute;
}

function createDisputeWithCommitments(): Dispute {
  return createMinimalDispute("evidence_commit", {
    commitments: {
      challenger: {
        committer: "did:lgm:challenger",
        evidenceHash: "hash-challenger",
        committedAtBlock: 200n,
        revealed: false,
      },
      defender: {
        committer: "did:lgm:defender",
        evidenceHash: "hash-defender",
        committedAtBlock: 210n,
        revealed: false,
      },
    },
  });
}

function createDisputeWithReveals(): Dispute {
  const evidence: Evidence[] = [
    {
      id: "ev1",
      type: "on_chain",
      content: "test content",
      contentHash: "hash",
      submitter: "did:lgm:submitter",
      submittedAt: Date.now(),
      submittedAtBlock: 100n,
      status: "active",
      reliability: 0.9,
      independence: "primary",
      archives: [],
      confirmations: [],
      challenges: [],
    },
  ];

  return createMinimalDispute("evidence_reveal", {
    commitments: {
      challenger: {
        committer: "did:lgm:challenger",
        evidenceHash: "hash-challenger",
        committedAtBlock: 200n,
        revealed: true,
        evidence,
      },
      defender: {
        committer: "did:lgm:defender",
        evidenceHash: "hash-defender",
        committedAtBlock: 210n,
        revealed: true,
        evidence,
      },
    },
    arbiters: [
      {
        publicKey: "pk1",
        did: "did:lgm:a1",
        selectionWeight: 1,
        domainExpertise: 0.8,
        stakedAmount: 100n * 10n ** 15n,
      },
      {
        publicKey: "pk2",
        did: "did:lgm:a2",
        selectionWeight: 1,
        domainExpertise: 0.8,
        stakedAmount: 100n * 10n ** 15n,
      },
      {
        publicKey: "pk3",
        did: "did:lgm:a3",
        selectionWeight: 1,
        domainExpertise: 0.8,
        stakedAmount: 100n * 10n ** 15n,
      },
      {
        publicKey: "pk4",
        did: "did:lgm:a4",
        selectionWeight: 1,
        domainExpertise: 0.8,
        stakedAmount: 100n * 10n ** 15n,
      },
      {
        publicKey: "pk5",
        did: "did:lgm:a5",
        selectionWeight: 1,
        domainExpertise: 0.8,
        stakedAmount: 100n * 10n ** 15n,
      },
      {
        publicKey: "pk6",
        did: "did:lgm:a6",
        selectionWeight: 1,
        domainExpertise: 0.8,
        stakedAmount: 100n * 10n ** 15n,
      },
      {
        publicKey: "pk7",
        did: "did:lgm:a7",
        selectionWeight: 1,
        domainExpertise: 0.8,
        stakedAmount: 100n * 10n ** 15n,
      },
    ],
  });
}

function createDisputeWithVotes(
  challengerVotes: number,
  defenderVotes: number,
  abstainVotes: number = 0,
): Dispute {
  const votes = new Map();
  let i = 0;

  for (let j = 0; j < challengerVotes; j++) {
    votes.set(`a${i}`, {
      arbiter: `a${i}`,
      vote: "challenger",
      justificationHash: `j${i}`,
      votedAtBlock: 100n,
      lockedStake: 77n * 10n ** 15n,
    });
    i++;
  }
  for (let j = 0; j < defenderVotes; j++) {
    votes.set(`a${i}`, {
      arbiter: `a${i}`,
      vote: "defender",
      justificationHash: `j${i}`,
      votedAtBlock: 100n,
      lockedStake: 77n * 10n ** 15n,
    });
    i++;
  }
  for (let j = 0; j < abstainVotes; j++) {
    votes.set(`a${i}`, {
      arbiter: `a${i}`,
      vote: "abstain",
      justificationHash: `j${i}`,
      votedAtBlock: 100n,
      lockedStake: 77n * 10n ** 15n,
    });
    i++;
  }

  return createMinimalDispute("arbitration", { votes });
}

// ============ Tests ============

describe("Dispute State Machine", () => {
  describe("Valid Transitions", () => {
    it("should define all phases", () => {
      const phases: DisputePhase[] = [
        "initiation",
        "evidence_commit",
        "evidence_reveal",
        "arbitration",
        "deliberation",
        "settlement",
        "escalated",
        "resolved",
      ];

      for (const phase of phases) {
        expect(VALID_TRANSITIONS[phase]).toBeDefined();
      }
    });

    it("should allow initiation → evidence_commit", () => {
      expect(isValidTransition("initiation", "evidence_commit")).toBe(true);
    });

    it("should allow evidence_commit → evidence_reveal", () => {
      expect(isValidTransition("evidence_commit", "evidence_reveal")).toBe(
        true,
      );
    });

    it("should allow evidence_reveal → arbitration", () => {
      expect(isValidTransition("evidence_reveal", "arbitration")).toBe(true);
    });

    it("should allow evidence_reveal → settlement (auto-resolve)", () => {
      expect(isValidTransition("evidence_reveal", "settlement")).toBe(true);
    });

    it("should allow arbitration → deliberation", () => {
      expect(isValidTransition("arbitration", "deliberation")).toBe(true);
    });

    it("should allow deliberation → settlement", () => {
      expect(isValidTransition("deliberation", "settlement")).toBe(true);
    });

    it("should allow deliberation → escalated", () => {
      expect(isValidTransition("deliberation", "escalated")).toBe(true);
    });

    it("should allow settlement → resolved", () => {
      expect(isValidTransition("settlement", "resolved")).toBe(true);
    });

    it("should allow escalated → evidence_commit (restart)", () => {
      expect(isValidTransition("escalated", "evidence_commit")).toBe(true);
    });

    it("should not allow resolved to transition anywhere", () => {
      expect(VALID_TRANSITIONS["resolved"]).toHaveLength(0);
      expect(isValidTransition("resolved", "initiation")).toBe(false);
      expect(isValidTransition("resolved", "settlement")).toBe(false);
    });

    it("should not allow backwards transitions", () => {
      expect(isValidTransition("evidence_reveal", "evidence_commit")).toBe(
        false,
      );
      expect(isValidTransition("arbitration", "evidence_reveal")).toBe(false);
      expect(isValidTransition("settlement", "deliberation")).toBe(false);
    });

    it("should not allow skipping phases", () => {
      expect(isValidTransition("initiation", "arbitration")).toBe(false);
      expect(isValidTransition("evidence_commit", "deliberation")).toBe(false);
      expect(isValidTransition("initiation", "resolved")).toBe(false);
    });
  });

  describe("canEnterEvidenceCommit", () => {
    it("should require initiation or escalated phase", () => {
      const dispute = createMinimalDispute("arbitration");
      const result = canEnterEvidenceCommit(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/initiation or escalated/);
    });

    it("should require defender to have matched stake", () => {
      const dispute = createMinimalDispute("initiation", {
        defender: { did: "", stake: 0n, responseEvidence: undefined },
      });
      const result = canEnterEvidenceCommit(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/stake/i);
    });

    it("should allow entry when conditions met", () => {
      const dispute = createMinimalDispute("initiation");
      const result = canEnterEvidenceCommit(dispute);

      expect(result.canEnter).toBe(true);
    });

    it("should allow entry from escalated phase", () => {
      const dispute = createMinimalDispute("escalated");
      const result = canEnterEvidenceCommit(dispute);

      expect(result.canEnter).toBe(true);
    });
  });

  describe("canEnterEvidenceReveal", () => {
    it("should require evidence_commit phase", () => {
      const dispute = createMinimalDispute("initiation");
      const result = canEnterEvidenceReveal(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/evidence_commit/);
    });

    it("should require challenger commitment", () => {
      const dispute = createMinimalDispute("evidence_commit", {
        commitments: {
          defender: {
            committer: "d",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: false,
          },
        },
      });
      const result = canEnterEvidenceReveal(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/Challenger/);
    });

    it("should require defender commitment", () => {
      const dispute = createMinimalDispute("evidence_commit", {
        commitments: {
          challenger: {
            committer: "c",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: false,
          },
        },
      });
      const result = canEnterEvidenceReveal(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/Defender/);
    });

    it("should allow entry when both committed", () => {
      const dispute = createDisputeWithCommitments();
      const result = canEnterEvidenceReveal(dispute);

      expect(result.canEnter).toBe(true);
    });
  });

  describe("canEnterArbitration", () => {
    it("should require evidence_reveal phase", () => {
      const dispute = createMinimalDispute("evidence_commit");
      const result = canEnterArbitration(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/evidence_reveal/);
    });

    it("should require challenger reveal", () => {
      const dispute = createMinimalDispute("evidence_reveal", {
        commitments: {
          challenger: {
            committer: "c",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: false,
          },
          defender: {
            committer: "d",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: true,
          },
        },
        arbiters: [
          {
            publicKey: "pk",
            did: "d",
            selectionWeight: 1,
            domainExpertise: 0.8,
            stakedAmount: 100n * 10n ** 15n,
          },
        ],
      });
      const result = canEnterArbitration(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/Challenger/);
    });

    it("should require defender reveal", () => {
      const dispute = createMinimalDispute("evidence_reveal", {
        commitments: {
          challenger: {
            committer: "c",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: true,
          },
          defender: {
            committer: "d",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: false,
          },
        },
        arbiters: [
          {
            publicKey: "pk",
            did: "d",
            selectionWeight: 1,
            domainExpertise: 0.8,
            stakedAmount: 100n * 10n ** 15n,
          },
        ],
      });
      const result = canEnterArbitration(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/Defender/);
    });

    it("should require arbiters to be selected", () => {
      const dispute = createMinimalDispute("evidence_reveal", {
        commitments: {
          challenger: {
            committer: "c",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: true,
          },
          defender: {
            committer: "d",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: true,
          },
        },
        arbiters: [],
      });
      const result = canEnterArbitration(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/Arbiters/);
    });

    it("should allow entry when all conditions met", () => {
      const dispute = createDisputeWithReveals();
      const result = canEnterArbitration(dispute);

      expect(result.canEnter).toBe(true);
    });
  });

  describe("canEnterDeliberation", () => {
    it("should require arbitration phase", () => {
      const dispute = createMinimalDispute("evidence_reveal");
      const result = canEnterDeliberation(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/arbitration/);
    });

    it("should require quorum votes (tier 1 = 55% of 7 = 4)", () => {
      const dispute = createDisputeWithVotes(2, 1); // Only 3 votes
      const result = canEnterDeliberation(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/Quorum/);
    });

    it("should allow entry with quorum met", () => {
      const dispute = createDisputeWithVotes(3, 2); // 5 votes >= 4 required
      // Need to also add arbiters
      const disputeWithArbiters = {
        ...dispute,
        arbiters: [
          {
            publicKey: "pk1",
            did: "a0",
            selectionWeight: 1,
            domainExpertise: 0.8,
            stakedAmount: 100n * 10n ** 15n,
          },
          {
            publicKey: "pk2",
            did: "a1",
            selectionWeight: 1,
            domainExpertise: 0.8,
            stakedAmount: 100n * 10n ** 15n,
          },
          {
            publicKey: "pk3",
            did: "a2",
            selectionWeight: 1,
            domainExpertise: 0.8,
            stakedAmount: 100n * 10n ** 15n,
          },
          {
            publicKey: "pk4",
            did: "a3",
            selectionWeight: 1,
            domainExpertise: 0.8,
            stakedAmount: 100n * 10n ** 15n,
          },
          {
            publicKey: "pk5",
            did: "a4",
            selectionWeight: 1,
            domainExpertise: 0.8,
            stakedAmount: 100n * 10n ** 15n,
          },
          {
            publicKey: "pk6",
            did: "a5",
            selectionWeight: 1,
            domainExpertise: 0.8,
            stakedAmount: 100n * 10n ** 15n,
          },
          {
            publicKey: "pk7",
            did: "a6",
            selectionWeight: 1,
            domainExpertise: 0.8,
            stakedAmount: 100n * 10n ** 15n,
          },
        ],
      };
      const result = canEnterDeliberation(disputeWithArbiters);

      expect(result.canEnter).toBe(true);
    });
  });

  describe("canEnterSettlement", () => {
    it("should require evidence_reveal or deliberation phase", () => {
      const dispute = createMinimalDispute("arbitration");
      const result = canEnterSettlement(dispute);

      expect(result.canEnter).toBe(false);
      expect(result.reason).toMatch(/evidence_reveal or deliberation/);
    });

    it("should allow entry from evidence_reveal (auto-settle)", () => {
      const dispute = createMinimalDispute("evidence_reveal");
      const result = canEnterSettlement(dispute);

      expect(result.canEnter).toBe(true);
    });
  });

  describe("canEscalate", () => {
    it("should require deliberation phase", () => {
      const dispute = createMinimalDispute("arbitration");
      const result = canEscalate(dispute);

      expect(result.canEscalate).toBe(false);
      expect(result.reason).toMatch(/deliberation/);
    });

    it("should not allow escalation at max tier", () => {
      const dispute = createMinimalDispute("deliberation", {
        state: {
          phase: "deliberation",
          phaseDeadline: 1000n,
          canEscalate: true,
          escalationTier: 3,
          previousTierResults: [],
        },
      });
      const result = canEscalate(dispute);

      expect(result.canEscalate).toBe(false);
      expect(result.reason).toMatch(/maximum/);
    });

    it("should allow escalation from deliberation at tier 1", () => {
      const dispute = createMinimalDispute("deliberation");
      const result = canEscalate(dispute);

      expect(result.canEscalate).toBe(true);
    });

    it("should allow escalation from deliberation at tier 2", () => {
      const dispute = createMinimalDispute("deliberation", {
        state: {
          phase: "deliberation",
          phaseDeadline: 1000n,
          canEscalate: true,
          escalationTier: 2,
          previousTierResults: [],
        },
      });
      const result = canEscalate(dispute);

      expect(result.canEscalate).toBe(true);
    });
  });

  describe("advancePhase", () => {
    it("should update phase and set deadline", () => {
      const dispute = createMinimalDispute("initiation");
      const newState = advancePhase(dispute, "evidence_commit", 500n);

      expect(newState.phase).toBe("evidence_commit");
      expect(newState.phaseDeadline).toBe(
        500n + BigInt(PHASE_DURATIONS.evidence_commit),
      );
    });

    it("should reject invalid transitions", () => {
      const dispute = createMinimalDispute("initiation");

      expect(() => advancePhase(dispute, "arbitration", 500n)).toThrow(
        DisputeError,
      );
    });

    it("should preserve escalation tier", () => {
      const dispute = createMinimalDispute("deliberation", {
        state: {
          phase: "deliberation",
          phaseDeadline: 1000n,
          canEscalate: true,
          escalationTier: 2,
          previousTierResults: [],
        },
      });
      const newState = advancePhase(dispute, "settlement", 500n);

      expect(newState.escalationTier).toBe(2);
    });

    it("should enable escalation only in deliberation", () => {
      const dispute = createMinimalDispute("arbitration");
      const newState = advancePhase(dispute, "deliberation", 500n);

      expect(newState.canEscalate).toBe(true);
    });
  });

  describe("escalateDispute", () => {
    it("should increment escalation tier", () => {
      const dispute = createMinimalDispute("deliberation");
      const newState = escalateDispute(dispute, 500n);

      expect(newState.escalationTier).toBe(2);
    });

    it("should reset phase to escalated", () => {
      const dispute = createMinimalDispute("deliberation");
      const newState = escalateDispute(dispute, 500n);

      expect(newState.phase).toBe("escalated");
    });

    it("should reject escalation from wrong phase", () => {
      const dispute = createMinimalDispute("arbitration");

      expect(() => escalateDispute(dispute, 500n)).toThrow(DisputeError);
    });

    it("should reject escalation at max tier", () => {
      const dispute = createMinimalDispute("deliberation", {
        state: {
          phase: "deliberation",
          phaseDeadline: 1000n,
          canEscalate: true,
          escalationTier: 3,
          previousTierResults: [],
        },
      });

      expect(() => escalateDispute(dispute, 500n)).toThrow(DisputeError);
    });
  });

  describe("hasPhaseTimedOut", () => {
    it("should return false before deadline", () => {
      const dispute = createMinimalDispute("initiation", {
        state: {
          phase: "initiation",
          phaseDeadline: 1000n,
          canEscalate: false,
          escalationTier: 1,
          previousTierResults: [],
        },
      });

      expect(hasPhaseTimedOut(dispute, 500n)).toBe(false);
      expect(hasPhaseTimedOut(dispute, 999n)).toBe(false);
    });

    it("should return false at deadline", () => {
      const dispute = createMinimalDispute("initiation", {
        state: {
          phase: "initiation",
          phaseDeadline: 1000n,
          canEscalate: false,
          escalationTier: 1,
          previousTierResults: [],
        },
      });

      expect(hasPhaseTimedOut(dispute, 1000n)).toBe(false);
    });

    it("should return true after deadline", () => {
      const dispute = createMinimalDispute("initiation", {
        state: {
          phase: "initiation",
          phaseDeadline: 1000n,
          canEscalate: false,
          escalationTier: 1,
          previousTierResults: [],
        },
      });

      expect(hasPhaseTimedOut(dispute, 1001n)).toBe(true);
    });
  });

  describe("handleTimeout", () => {
    it("should reject when not timed out", () => {
      const dispute = createMinimalDispute("initiation", {
        state: {
          phase: "initiation",
          phaseDeadline: 1000n,
          canEscalate: false,
          escalationTier: 1,
          previousTierResults: [],
        },
      });

      expect(() => handleTimeout(dispute, 500n)).toThrow(DisputeError);
    });

    it("should default to challenger win on initiation timeout", () => {
      const dispute = createMinimalDispute("initiation", {
        state: {
          phase: "initiation",
          phaseDeadline: 1000n,
          canEscalate: false,
          escalationTier: 1,
          previousTierResults: [],
        },
      });

      const result = handleTimeout(dispute, 1001n);

      expect(result.action).toBe("default_judgment");
      expect(result.winner).toBe("challenger");
    });

    it("should default to defender win if challenger fails to commit", () => {
      const dispute = createMinimalDispute("evidence_commit", {
        state: {
          phase: "evidence_commit",
          phaseDeadline: 1000n,
          canEscalate: false,
          escalationTier: 1,
          previousTierResults: [],
        },
        commitments: {
          defender: {
            committer: "d",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: false,
          },
        },
      });

      const result = handleTimeout(dispute, 1001n);

      expect(result.action).toBe("default_judgment");
      expect(result.winner).toBe("defender");
    });

    it("should default to challenger win if defender fails to commit", () => {
      const dispute = createMinimalDispute("evidence_commit", {
        state: {
          phase: "evidence_commit",
          phaseDeadline: 1000n,
          canEscalate: false,
          escalationTier: 1,
          previousTierResults: [],
        },
        commitments: {
          challenger: {
            committer: "c",
            evidenceHash: "h",
            committedAtBlock: 100n,
            revealed: false,
          },
        },
      });

      const result = handleTimeout(dispute, 1001n);

      expect(result.action).toBe("default_judgment");
      expect(result.winner).toBe("challenger");
    });

    it("should forfeit if both fail to commit", () => {
      const dispute = createMinimalDispute("evidence_commit", {
        state: {
          phase: "evidence_commit",
          phaseDeadline: 1000n,
          canEscalate: false,
          escalationTier: 1,
          previousTierResults: [],
        },
        commitments: {},
      });

      const result = handleTimeout(dispute, 1001n);

      expect(result.action).toBe("forfeit");
    });

    it("should advance if both committed on evidence_commit timeout", () => {
      const dispute = createDisputeWithCommitments();
      dispute.state.phaseDeadline = 1000n;

      const result = handleTimeout(dispute, 1001n);

      expect(result.action).toBe("advance");
      expect(result.nextPhase).toBe("evidence_reveal");
    });
  });

  describe("determineFinalWinner", () => {
    it("should return challenger with clear majority", () => {
      const dispute = createDisputeWithVotes(5, 2);
      expect(determineFinalWinner(dispute)).toBe("challenger");
    });

    it("should return defender with clear majority", () => {
      const dispute = createDisputeWithVotes(2, 5);
      expect(determineFinalWinner(dispute)).toBe("defender");
    });

    it("should return draw on exact tie", () => {
      const dispute = createDisputeWithVotes(3, 3);
      expect(determineFinalWinner(dispute)).toBe("draw");
    });

    it("should return draw with no votes", () => {
      const dispute = createMinimalDispute("arbitration", { votes: new Map() });
      expect(determineFinalWinner(dispute)).toBe("draw");
    });

    it("should return draw with all abstains", () => {
      const dispute = createDisputeWithVotes(0, 0, 5);
      expect(determineFinalWinner(dispute)).toBe("draw");
    });

    it("should handle challenger lead with abstains", () => {
      const dispute = createDisputeWithVotes(3, 2, 2);
      expect(determineFinalWinner(dispute)).toBe("challenger");
    });

    it("should handle defender lead with abstains", () => {
      const dispute = createDisputeWithVotes(2, 3, 2);
      expect(determineFinalWinner(dispute)).toBe("defender");
    });
  });

  describe("isTerminal", () => {
    it("should return true for resolved phase", () => {
      const dispute = createMinimalDispute("resolved");
      expect(isTerminal(dispute)).toBe(true);
    });

    it("should return false for non-resolved phases", () => {
      const phases: DisputePhase[] = [
        "initiation",
        "evidence_commit",
        "evidence_reveal",
        "arbitration",
        "deliberation",
        "settlement",
        "escalated",
      ];

      for (const phase of phases) {
        const dispute = createMinimalDispute(phase);
        expect(isTerminal(dispute)).toBe(false);
      }
    });
  });

  describe("Escalation Tier Configurations", () => {
    it("should have correct tier 1 configuration", () => {
      expect(ESCALATION_TIERS[1]).toEqual({
        tier: 1,
        arbiterCount: 7,
        quorumRequirement: 0.55,
        stakeMultiplier: 1.1,
        appealWindow: 555,
      });
    });

    it("should have correct tier 2 configuration", () => {
      expect(ESCALATION_TIERS[2]).toEqual({
        tier: 2,
        arbiterCount: 11,
        quorumRequirement: 0.66,
        stakeMultiplier: 2.2,
        appealWindow: 1111,
      });
    });

    it("should have correct tier 3 configuration", () => {
      expect(ESCALATION_TIERS[3]).toEqual({
        tier: 3,
        arbiterCount: 22,
        quorumRequirement: 0.77,
        stakeMultiplier: 5.5,
        appealWindow: 0, // Final - no appeal
      });
    });

    it("should increase arbiter count with each tier", () => {
      expect(ESCALATION_TIERS[2].arbiterCount).toBeGreaterThan(
        ESCALATION_TIERS[1].arbiterCount,
      );
      expect(ESCALATION_TIERS[3].arbiterCount).toBeGreaterThan(
        ESCALATION_TIERS[2].arbiterCount,
      );
    });

    it("should increase quorum requirement with each tier", () => {
      expect(ESCALATION_TIERS[2].quorumRequirement).toBeGreaterThan(
        ESCALATION_TIERS[1].quorumRequirement,
      );
      expect(ESCALATION_TIERS[3].quorumRequirement).toBeGreaterThan(
        ESCALATION_TIERS[2].quorumRequirement,
      );
    });

    it("should increase stake multiplier with each tier", () => {
      expect(ESCALATION_TIERS[2].stakeMultiplier).toBeGreaterThan(
        ESCALATION_TIERS[1].stakeMultiplier,
      );
      expect(ESCALATION_TIERS[3].stakeMultiplier).toBeGreaterThan(
        ESCALATION_TIERS[2].stakeMultiplier,
      );
    });
  });

  describe("Phase Durations", () => {
    it("should have all required phase durations", () => {
      expect(PHASE_DURATIONS.initiation).toBeDefined();
      expect(PHASE_DURATIONS.evidence_commit).toBeDefined();
      expect(PHASE_DURATIONS.evidence_reveal).toBeDefined();
      expect(PHASE_DURATIONS.arbitration).toBeDefined();
      expect(PHASE_DURATIONS.deliberation).toBeDefined();
      expect(PHASE_DURATIONS.settlement).toBeDefined();
    });

    it("should have positive durations", () => {
      expect(PHASE_DURATIONS.initiation).toBeGreaterThan(0);
      expect(PHASE_DURATIONS.evidence_commit).toBeGreaterThan(0);
      expect(PHASE_DURATIONS.evidence_reveal).toBeGreaterThan(0);
      expect(PHASE_DURATIONS.arbitration).toBeGreaterThan(0);
      expect(PHASE_DURATIONS.deliberation).toBeGreaterThan(0);
      expect(PHASE_DURATIONS.settlement).toBeGreaterThan(0);
    });

    it("should have arbitration as longest phase", () => {
      const durations = Object.values(PHASE_DURATIONS);
      const maxDuration = Math.max(...durations);
      expect(PHASE_DURATIONS.arbitration).toBe(maxDuration);
    });
  });
});
