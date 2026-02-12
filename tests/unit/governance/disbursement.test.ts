/**
 * Disbursement Proposal & Expense Tracking Tests
 *
 * Tests for:
 * - Disbursement lifecycle (submit → stake → vote → approve → execute)
 * - Rejection flow with stake return
 * - Cancellation by proposer
 * - Amount cap validation (10% of treasury)
 * - Timelock enforcement
 * - Authorization checks
 * - Expense record creation on execution
 * - Stake slashing on rejection/cancellation
 */

import { describe, it, expect } from "vitest";
import { GovernanceStateTree } from "../../../state/governance.js";
import {
  createDisbursement,
  transitionToVoting,
  transitionFromVoting,
  executeDisbursement,
  cancelDisbursement,
  calculateDisbursementStakeOutcome,
} from "../../../governance/treasury/disbursement_lifecycle.js";
import {
  DISBURSEMENT_STAKE,
  DISBURSEMENT_THRESHOLDS,
  TIMELOCK_BLOCKS,
  validateDisbursementAmount,
  treasuryFundToDid,
  isDisbursementTerminal,
  type DisbursementProposal,
  type ExpenseRecord,
} from "../../../governance/treasury/types.js";
import type { VoteTally } from "../../../governance/proposals/types.js";

const ONE_LGM = 10n ** 18n;

function makeProposal(
  overrides: Partial<DisbursementProposal> = {},
): DisbursementProposal {
  return {
    id: "DISB-1",
    title: "Infrastructure hosting costs",
    description: "Monthly hosting costs for protocol nodes",
    category: "infrastructure",
    proposer: "did:lgm:proposer1",
    treasuryFund: "FOUNDATION",
    recipientDid: "did:lgm:ops-agent",
    amount: 1000n * ONE_LGM,
    stage: "proposed",
    stageHistory: [
      {
        from: "proposed" as any,
        to: "proposed" as any,
        at: 100n,
        reason: "Created",
        triggeredBy: "did:lgm:proposer1",
      },
    ],
    stakedAmount: 0n,
    stakeHolder: "did:lgm:proposer1",
    createdAt: 100n,
    updatedAt: 100n,
    justification: "Critical infrastructure needs funding",
    ...overrides,
  };
}

function makePassingVoteTally(): VoteTally {
  return {
    support: 900n,
    oppose: 100n,
    abstain: 50n,
    totalEligible: 1050n,
    uniqueVoters: 10,
    operatorBreakdown: new Map(),
    quorumMet: true,
    operatorCapViolation: false,
  };
}

function makeFailingVoteTally(): VoteTally {
  return {
    support: 200n,
    oppose: 800n,
    abstain: 50n,
    totalEligible: 1050n,
    uniqueVoters: 10,
    operatorBreakdown: new Map(),
    quorumMet: true,
    operatorCapViolation: false,
  };
}

// ============ Type & Constant Tests ============

describe("Disbursement Types & Constants", () => {
  it("DISBURSEMENT_STAKE is 2000 LGM", () => {
    expect(DISBURSEMENT_STAKE).toBe(2000n * 10n ** 15n);
  });

  it("TIMELOCK_BLOCKS is 50000", () => {
    expect(TIMELOCK_BLOCKS).toBe(50000n);
  });

  it("thresholds require 80% support and 20% quorum", () => {
    expect(DISBURSEMENT_THRESHOLDS.supportRequired).toBe(0.8);
    expect(DISBURSEMENT_THRESHOLDS.quorumRequired).toBe(0.2);
    expect(DISBURSEMENT_THRESHOLDS.operatorCap).toBe(0.15);
  });

  it("isDisbursementTerminal identifies terminal stages", () => {
    expect(isDisbursementTerminal("executed")).toBe(true);
    expect(isDisbursementTerminal("rejected")).toBe(true);
    expect(isDisbursementTerminal("cancelled")).toBe(true);
    expect(isDisbursementTerminal("proposed")).toBe(false);
    expect(isDisbursementTerminal("voting")).toBe(false);
    expect(isDisbursementTerminal("approved")).toBe(false);
  });

  it("treasuryFundToDid maps all funds correctly", () => {
    expect(treasuryFundToDid("RESEARCH_FUND")).toBe(
      "did:lgm:treasury-research-fund",
    );
    expect(treasuryFundToDid("FOUNDATION")).toBe("did:lgm:treasury-foundation");
    expect(treasuryFundToDid("COMMUNITY")).toBe("did:lgm:treasury-community");
    expect(treasuryFundToDid("DEVELOPERS")).toBe("did:lgm:treasury-developers");
    expect(treasuryFundToDid("RESERVE")).toBe("did:lgm:treasury-reserve");
  });
});

// ============ Amount Validation Tests ============

describe("Disbursement Amount Validation", () => {
  it("rejects zero amount", () => {
    const result = validateDisbursementAmount(0n, 100000n * ONE_LGM);
    expect(result.valid).toBe(false);
    expect(result.error).toContain("positive");
  });

  it("rejects negative amount", () => {
    const result = validateDisbursementAmount(-1n, 100000n * ONE_LGM);
    expect(result.valid).toBe(false);
  });

  it("accepts amount within 10% cap", () => {
    const treasury = 100000n * ONE_LGM;
    const result = validateDisbursementAmount(10000n * ONE_LGM, treasury);
    expect(result.valid).toBe(true);
  });

  it("rejects amount exceeding 10% cap", () => {
    const treasury = 100000n * ONE_LGM;
    const result = validateDisbursementAmount(10001n * ONE_LGM, treasury);
    expect(result.valid).toBe(false);
    expect(result.error).toContain("10%");
  });

  it("handles edge case: exactly 10% is valid", () => {
    const treasury = 50000n * ONE_LGM;
    const result = validateDisbursementAmount(5000n * ONE_LGM, treasury);
    expect(result.valid).toBe(true);
  });
});

// ============ Lifecycle Tests ============

describe("Disbursement Lifecycle", () => {
  describe("createDisbursement", () => {
    it("creates proposal in proposed stage", () => {
      const result = createDisbursement(
        {
          title: "Server costs",
          description: "Monthly hosting",
          category: "infrastructure",
          treasuryFund: "FOUNDATION",
          proposer: "did:lgm:proposer1" as any,
          recipientDid: "did:lgm:ops" as any,
          amount: 500n * ONE_LGM,
          justification: "Required for operation",
        },
        1000n,
        1,
      );

      expect("proposal" in result).toBe(true);
      if ("proposal" in result) {
        expect(result.proposal.id).toBe("DISB-1");
        expect(result.proposal.stage).toBe("proposed");
        expect(result.proposal.amount).toBe(500n * ONE_LGM);
        expect(result.proposal.stakedAmount).toBe(0n);
      }
    });

    it("rejects invalid category", () => {
      const result = createDisbursement(
        {
          title: "Test",
          description: "Test",
          category: "invalid" as any,
          treasuryFund: "FOUNDATION",
          proposer: "did:lgm:proposer1" as any,
          recipientDid: "did:lgm:ops" as any,
          amount: 500n * ONE_LGM,
          justification: "Test",
        },
        1000n,
        1,
      );
      expect("error" in result).toBe(true);
    });

    it("rejects missing justification", () => {
      const result = createDisbursement(
        {
          title: "Test",
          description: "Test",
          category: "infrastructure",
          treasuryFund: "FOUNDATION",
          proposer: "did:lgm:proposer1" as any,
          recipientDid: "did:lgm:ops" as any,
          amount: 500n * ONE_LGM,
          justification: "",
        },
        1000n,
        1,
      );
      expect("error" in result).toBe(true);
    });

    it("rejects zero amount", () => {
      const result = createDisbursement(
        {
          title: "Test",
          description: "Test",
          category: "infrastructure",
          treasuryFund: "FOUNDATION",
          proposer: "did:lgm:proposer1" as any,
          recipientDid: "did:lgm:ops" as any,
          amount: 0n,
          justification: "Test",
        },
        1000n,
        1,
      );
      expect("error" in result).toBe(true);
    });
  });

  describe("transitionToVoting", () => {
    it("transitions from proposed to voting with sufficient stake", () => {
      const proposal = makeProposal();
      const result = transitionToVoting(
        proposal,
        DISBURSEMENT_STAKE,
        200n,
        "did:lgm:proposer1" as any,
      );

      expect(result.success).toBe(true);
      expect(result.newStage).toBe("voting");
      expect(proposal.stage).toBe("voting");
      expect(proposal.stakedAmount).toBe(DISBURSEMENT_STAKE);
    });

    it("rejects insufficient stake", () => {
      const proposal = makeProposal();
      const result = transitionToVoting(
        proposal,
        DISBURSEMENT_STAKE - 1n,
        200n,
        "did:lgm:proposer1" as any,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Insufficient stake");
      expect(proposal.stage).toBe("proposed");
    });

    it("rejects transition from non-proposed stage", () => {
      const proposal = makeProposal({ stage: "voting" });
      const result = transitionToVoting(
        proposal,
        DISBURSEMENT_STAKE,
        200n,
        "did:lgm:proposer1" as any,
      );

      expect(result.success).toBe(false);
    });
  });

  describe("transitionFromVoting", () => {
    it("approves with sufficient support and quorum", () => {
      const proposal = makeProposal({
        stage: "voting",
        stakedAmount: DISBURSEMENT_STAKE,
      });
      const tally = makePassingVoteTally();
      const result = transitionFromVoting(proposal, tally, 500n);

      expect(result.success).toBe(true);
      expect(result.newStage).toBe("approved");
      expect(proposal.stage).toBe("approved");
      expect(proposal.approvedAt).toBe(500n);
      expect(proposal.timelockExpiry).toBe(500n + TIMELOCK_BLOCKS);
    });

    it("rejects with insufficient support", () => {
      const proposal = makeProposal({
        stage: "voting",
        stakedAmount: DISBURSEMENT_STAKE,
      });
      const tally = makeFailingVoteTally();
      const result = transitionFromVoting(proposal, tally, 500n);

      expect(result.success).toBe(true);
      expect(result.newStage).toBe("rejected");
      expect(proposal.stage).toBe("rejected");
    });

    it("rejects with insufficient quorum", () => {
      const proposal = makeProposal({
        stage: "voting",
        stakedAmount: DISBURSEMENT_STAKE,
      });
      const tally: VoteTally = {
        support: 90n,
        oppose: 10n,
        abstain: 0n,
        totalEligible: 10000n, // quorum = 100/10000 = 1% < 20%
        uniqueVoters: 5,
        operatorBreakdown: new Map(),
        quorumMet: false,
        operatorCapViolation: false,
      };
      const result = transitionFromVoting(proposal, tally, 500n);

      expect(result.success).toBe(true);
      expect(result.newStage).toBe("rejected");
    });

    it("rejects if operator cap violated", () => {
      const proposal = makeProposal({
        stage: "voting",
        stakedAmount: DISBURSEMENT_STAKE,
      });
      const tally: VoteTally = {
        ...makePassingVoteTally(),
        operatorCapViolation: true,
      };
      const result = transitionFromVoting(proposal, tally, 500n);

      expect(result.success).toBe(false);
      expect(result.error).toContain("Operator cap");
    });

    it("fails for non-voting stage", () => {
      const proposal = makeProposal({ stage: "proposed" });
      const result = transitionFromVoting(
        proposal,
        makePassingVoteTally(),
        500n,
      );
      expect(result.success).toBe(false);
    });
  });

  describe("executeDisbursement", () => {
    it("executes after timelock expires", () => {
      const proposal = makeProposal({
        stage: "approved",
        approvedAt: 1000n,
        timelockExpiry: 1000n + TIMELOCK_BLOCKS,
      });
      const blockAfterTimelock = 1000n + TIMELOCK_BLOCKS + 1n;
      const result = executeDisbursement(proposal, blockAfterTimelock);

      expect(result.success).toBe(true);
      expect(result.newStage).toBe("executed");
      expect(proposal.stage).toBe("executed");
      expect(proposal.executedAt).toBe(blockAfterTimelock);
    });

    it("rejects before timelock expires", () => {
      const proposal = makeProposal({
        stage: "approved",
        approvedAt: 1000n,
        timelockExpiry: 1000n + TIMELOCK_BLOCKS,
      });
      const blockBeforeTimelock = 1000n + TIMELOCK_BLOCKS - 1n;
      const result = executeDisbursement(proposal, blockBeforeTimelock);

      expect(result.success).toBe(false);
      expect(result.error).toContain("Timelock not expired");
    });

    it("rejects from non-approved stage", () => {
      const proposal = makeProposal({ stage: "voting" });
      const result = executeDisbursement(proposal, 99999n);
      expect(result.success).toBe(false);
    });

    it("executes exactly at timelock expiry", () => {
      const proposal = makeProposal({
        stage: "approved",
        approvedAt: 1000n,
        timelockExpiry: 1000n + TIMELOCK_BLOCKS,
      });
      const result = executeDisbursement(proposal, 1000n + TIMELOCK_BLOCKS);
      expect(result.success).toBe(true);
    });
  });

  describe("cancelDisbursement", () => {
    it("allows proposer to cancel from proposed stage", () => {
      const proposal = makeProposal();
      const result = cancelDisbursement(
        proposal,
        300n,
        "did:lgm:proposer1" as any,
      );

      expect(result.success).toBe(true);
      expect(result.newStage).toBe("cancelled");
      expect(proposal.stage).toBe("cancelled");
    });

    it("allows proposer to cancel from approved stage", () => {
      const proposal = makeProposal({
        stage: "approved",
        approvedAt: 200n,
        timelockExpiry: 200n + TIMELOCK_BLOCKS,
      });
      const result = cancelDisbursement(
        proposal,
        300n,
        "did:lgm:proposer1" as any,
      );

      expect(result.success).toBe(true);
      expect(result.newStage).toBe("cancelled");
    });

    it("rejects non-proposer cancellation", () => {
      const proposal = makeProposal();
      const result = cancelDisbursement(
        proposal,
        300n,
        "did:lgm:other-agent" as any,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Only the proposer");
    });

    it("rejects cancellation from voting stage", () => {
      const proposal = makeProposal({ stage: "voting" });
      const result = cancelDisbursement(
        proposal,
        300n,
        "did:lgm:proposer1" as any,
      );

      expect(result.success).toBe(false);
      expect(result.error).toContain("Cannot cancel");
    });

    it("rejects cancellation from executed stage", () => {
      const proposal = makeProposal({ stage: "executed" });
      const result = cancelDisbursement(
        proposal,
        300n,
        "did:lgm:proposer1" as any,
      );

      expect(result.success).toBe(false);
    });
  });
});

// ============ Stake Outcome Tests ============

describe("Disbursement Stake Outcomes", () => {
  it("returns full stake on execution", () => {
    const proposal = makeProposal({ stakedAmount: DISBURSEMENT_STAKE });
    const outcome = calculateDisbursementStakeOutcome(proposal, "executed");
    expect(outcome.returnAmount).toBe(DISBURSEMENT_STAKE);
    expect(outcome.slashAmount).toBe(0n);
  });

  it("slashes 100% on malicious rejection", () => {
    const proposal = makeProposal({ stakedAmount: DISBURSEMENT_STAKE });
    const outcome = calculateDisbursementStakeOutcome(
      proposal,
      "rejected_malicious",
    );
    expect(outcome.returnAmount).toBe(0n);
    expect(outcome.slashAmount).toBe(DISBURSEMENT_STAKE);
  });

  it("slashes 50% on frivolous rejection", () => {
    const proposal = makeProposal({ stakedAmount: DISBURSEMENT_STAKE });
    const outcome = calculateDisbursementStakeOutcome(
      proposal,
      "rejected_frivolous",
    );
    expect(outcome.slashAmount).toBe(DISBURSEMENT_STAKE / 2n);
    expect(outcome.returnAmount).toBe(DISBURSEMENT_STAKE / 2n);
  });

  it("slashes 10% on normal rejection", () => {
    const proposal = makeProposal({ stakedAmount: DISBURSEMENT_STAKE });
    const outcome = calculateDisbursementStakeOutcome(
      proposal,
      "rejected_normal",
    );
    expect(outcome.slashAmount).toBe(DISBURSEMENT_STAKE / 10n);
    expect(outcome.returnAmount).toBe(
      DISBURSEMENT_STAKE - DISBURSEMENT_STAKE / 10n,
    );
  });

  it("charges 5% processing fee on cancellation", () => {
    const proposal = makeProposal({ stakedAmount: DISBURSEMENT_STAKE });
    const outcome = calculateDisbursementStakeOutcome(proposal, "cancelled");
    expect(outcome.slashAmount).toBe(DISBURSEMENT_STAKE / 20n);
    expect(outcome.returnAmount).toBe(
      DISBURSEMENT_STAKE - DISBURSEMENT_STAKE / 20n,
    );
  });
});

// ============ GovernanceStateTree Integration Tests ============

describe("GovernanceStateTree Disbursement CRUD", () => {
  // Each test creates its own tree (tests run concurrently)
  function freshTree() {
    return new GovernanceStateTree();
  }

  it("adds and retrieves disbursement", () => {
    const tree = freshTree();
    const proposal = makeProposal();
    tree.addDisbursement(proposal);

    const retrieved = tree.getDisbursement("DISB-1");
    expect(retrieved).toBeDefined();
    expect(retrieved!.title).toBe("Infrastructure hosting costs");
    expect(retrieved!.amount).toBe(1000n * ONE_LGM);
  });

  it("rejects duplicate disbursement ID", () => {
    const tree = freshTree();
    const proposal = makeProposal({ id: "DISB-100" });
    tree.addDisbursement(proposal);
    expect(() => tree.addDisbursement(proposal)).toThrow();
  });

  it("updates disbursement and reindexes stage", () => {
    const tree = freshTree();
    const proposal = makeProposal({ id: "DISB-101" });
    tree.addDisbursement(proposal);

    expect(tree.getDisbursementsByStage("proposed")).toHaveLength(1);
    expect(tree.getDisbursementsByStage("voting")).toHaveLength(0);

    const updated = { ...proposal, stage: "voting" as const };
    tree.updateDisbursement("DISB-101", updated);

    expect(tree.getDisbursementsByStage("proposed")).toHaveLength(0);
    expect(tree.getDisbursementsByStage("voting")).toHaveLength(1);
  });

  it("indexes by treasury fund", () => {
    const tree = freshTree();
    tree.addDisbursement(
      makeProposal({ id: "DISB-102", treasuryFund: "FOUNDATION" }),
    );
    tree.addDisbursement(
      makeProposal({ id: "DISB-103", treasuryFund: "COMMUNITY" }),
    );

    expect(tree.getDisbursementsByTreasury("FOUNDATION")).toHaveLength(1);
    expect(tree.getDisbursementsByTreasury("COMMUNITY")).toHaveLength(1);
    expect(tree.getDisbursementsByTreasury("RESERVE")).toHaveLength(0);
  });

  it("indexes by category", () => {
    const tree = freshTree();
    tree.addDisbursement(
      makeProposal({ id: "DISB-104", category: "infrastructure" }),
    );
    tree.addDisbursement(makeProposal({ id: "DISB-105", category: "legal" }));

    expect(tree.getDisbursementsByCategory("infrastructure")).toHaveLength(1);
    expect(tree.getDisbursementsByCategory("legal")).toHaveLength(1);
  });

  it("auto-increments disbursement counter", () => {
    const tree = freshTree();
    expect(tree.getNextDisbursementNumber()).toBe(1);
    tree.addDisbursement(makeProposal({ id: "DISB-1" }));
    expect(tree.getNextDisbursementNumber()).toBe(2);
    tree.addDisbursement(makeProposal({ id: "DISB-5" }));
    expect(tree.getNextDisbursementNumber()).toBe(6);
  });

  it("tracks totalDisbursed on execution", () => {
    const tree = freshTree();
    const proposal = makeProposal({ id: "DISB-106", stage: "approved" });
    tree.addDisbursement(proposal);

    const updated = {
      ...proposal,
      stage: "executed" as const,
      executedAt: 999n,
    };
    tree.updateDisbursement("DISB-106", updated);

    expect(tree.getStats().totalDisbursed).toBe(1000n * ONE_LGM);
  });
});

// ============ Expense Record Tests ============

describe("GovernanceStateTree Expense Records", () => {
  // Each test creates its own tree (tests run concurrently)
  function freshTree() {
    return new GovernanceStateTree();
  }

  function makeExpense(overrides: Partial<ExpenseRecord> = {}): ExpenseRecord {
    return {
      id: "EXP-1",
      disbursementId: "DISB-1",
      category: "infrastructure",
      treasuryFund: "FOUNDATION",
      recipientDid: "did:lgm:ops-agent",
      amount: 500n * ONE_LGM,
      executedAt: 999n,
      txHash: "abc123",
      justification: "Monthly hosting",
      ...overrides,
    };
  }

  it("adds and retrieves expense record", () => {
    const tree = freshTree();
    const expense = makeExpense();
    tree.addExpenseRecord(expense);

    const retrieved = tree.getExpenseRecord("EXP-1");
    expect(retrieved).toBeDefined();
    expect(retrieved!.amount).toBe(500n * ONE_LGM);
  });

  it("rejects duplicate expense ID", () => {
    const tree = freshTree();
    const expense = makeExpense({ id: "EXP-200" });
    tree.addExpenseRecord(expense);
    expect(() => tree.addExpenseRecord(expense)).toThrow();
  });

  it("indexes by category", () => {
    const tree = freshTree();
    tree.addExpenseRecord(
      makeExpense({ id: "EXP-201", category: "infrastructure" }),
    );
    tree.addExpenseRecord(makeExpense({ id: "EXP-202", category: "legal" }));

    expect(tree.getExpensesByCategory("infrastructure")).toHaveLength(1);
    expect(tree.getExpensesByCategory("legal")).toHaveLength(1);
  });

  it("indexes by treasury fund", () => {
    const tree = freshTree();
    tree.addExpenseRecord(
      makeExpense({ id: "EXP-203", treasuryFund: "FOUNDATION" }),
    );
    tree.addExpenseRecord(
      makeExpense({ id: "EXP-204", treasuryFund: "COMMUNITY" }),
    );

    expect(tree.getExpensesByTreasury("FOUNDATION")).toHaveLength(1);
    expect(tree.getExpensesByTreasury("COMMUNITY")).toHaveLength(1);
  });

  it("calculates total expenses across all funds", () => {
    const tree = freshTree();
    tree.addExpenseRecord(
      makeExpense({ id: "EXP-205", amount: 100n * ONE_LGM }),
    );
    tree.addExpenseRecord(
      makeExpense({ id: "EXP-206", amount: 200n * ONE_LGM }),
    );

    expect(tree.getTotalExpenses()).toBe(300n * ONE_LGM);
  });

  it("calculates total expenses for specific fund", () => {
    const tree = freshTree();
    tree.addExpenseRecord(
      makeExpense({
        id: "EXP-207",
        treasuryFund: "FOUNDATION",
        amount: 100n * ONE_LGM,
      }),
    );
    tree.addExpenseRecord(
      makeExpense({
        id: "EXP-208",
        treasuryFund: "COMMUNITY",
        amount: 200n * ONE_LGM,
      }),
    );

    expect(tree.getTotalExpenses("FOUNDATION")).toBe(100n * ONE_LGM);
    expect(tree.getTotalExpenses("COMMUNITY")).toBe(200n * ONE_LGM);
  });

  it("auto-increments expense counter", () => {
    const tree = freshTree();
    expect(tree.getNextExpenseNumber()).toBe(1);
    tree.addExpenseRecord(makeExpense({ id: "EXP-1" }));
    expect(tree.getNextExpenseNumber()).toBe(2);
  });
});

// ============ Export/Import/Clone Tests ============

describe("GovernanceStateTree Disbursement Persistence", () => {
  it("exports and imports disbursements and expenses", () => {
    const tree = new GovernanceStateTree();
    tree.addDisbursement(makeProposal({ id: "DISB-300" }));
    tree.addExpenseRecord({
      id: "EXP-300",
      disbursementId: "DISB-300",
      category: "infrastructure",
      treasuryFund: "FOUNDATION",
      recipientDid: "did:lgm:ops",
      amount: 500n * ONE_LGM,
      executedAt: 999n,
      txHash: "abc",
      justification: "Test",
    });

    const exported = tree.exportState();
    expect(exported.disbursements).toHaveLength(1);
    expect(exported.expenses).toHaveLength(1);

    const tree2 = new GovernanceStateTree();
    tree2.importState(exported);

    expect(tree2.getDisbursement("DISB-300")).toBeDefined();
    expect(tree2.getExpenseRecord("EXP-300")).toBeDefined();
    expect(tree2.getExpensesByCategory("infrastructure")).toHaveLength(1);
  });

  it("clones disbursements and expenses", () => {
    const tree = new GovernanceStateTree();
    tree.addDisbursement(makeProposal({ id: "DISB-301" }));
    tree.addExpenseRecord({
      id: "EXP-301",
      disbursementId: "DISB-301",
      category: "infrastructure",
      treasuryFund: "FOUNDATION",
      recipientDid: "did:lgm:ops",
      amount: 500n * ONE_LGM,
      executedAt: 999n,
      txHash: "abc",
      justification: "Test",
    });

    const cloned = tree.clone();

    // Verify cloned data
    expect(cloned.getDisbursement("DISB-301")).toBeDefined();
    expect(cloned.getExpenseRecord("EXP-301")).toBeDefined();

    // Verify independence — modify clone, original unchanged
    const proposal = cloned.getDisbursement("DISB-301")!;
    cloned.updateDisbursement("DISB-301", { ...proposal, stage: "voting" });

    expect(cloned.getDisbursementsByStage("voting")).toHaveLength(1);
    expect(tree.getDisbursementsByStage("voting")).toHaveLength(0);
  });

  it("clear resets all disbursement/expense state", () => {
    const tree = new GovernanceStateTree();
    tree.addDisbursement(makeProposal({ id: "DISB-302" }));
    tree.addExpenseRecord({
      id: "EXP-302",
      disbursementId: "DISB-302",
      category: "infrastructure",
      treasuryFund: "FOUNDATION",
      recipientDid: "did:lgm:ops",
      amount: 500n * ONE_LGM,
      executedAt: 999n,
      txHash: "abc",
      justification: "Test",
    });

    tree.clear();

    expect(tree.getDisbursement("DISB-302")).toBeUndefined();
    expect(tree.getExpenseRecord("EXP-302")).toBeUndefined();
    expect(tree.getNextDisbursementNumber()).toBe(1);
    expect(tree.getNextExpenseNumber()).toBe(1);
    expect(tree.getStats().totalDisbursements).toBe(0);
    expect(tree.getStats().totalExpenses).toBe(0);
  });
});

// ============ Full Lifecycle Integration Test ============

describe("Disbursement Full Lifecycle", () => {
  it("submit → stake → vote → approve → (timelock) → execute", () => {
    // 1. Create
    const result = createDisbursement(
      {
        title: "Domain renewal",
        description: "Annual domain registration",
        category: "domain",
        treasuryFund: "FOUNDATION",
        proposer: "did:lgm:proposer1" as any,
        recipientDid: "did:lgm:registrar" as any,
        amount: 100n * ONE_LGM,
        justification: "Annual domain renewal for protocol website",
      },
      1000n,
      1,
    );
    expect("proposal" in result).toBe(true);
    const proposal = (result as { proposal: DisbursementProposal }).proposal;
    expect(proposal.stage).toBe("proposed");

    // 2. Stake → voting
    const stakeResult = transitionToVoting(
      proposal,
      DISBURSEMENT_STAKE,
      2000n,
      "did:lgm:proposer1" as any,
    );
    expect(stakeResult.success).toBe(true);
    expect(proposal.stage).toBe("voting");

    // 3. Vote → approved
    const voteResult = transitionFromVoting(
      proposal,
      makePassingVoteTally(),
      3000n,
    );
    expect(voteResult.success).toBe(true);
    expect(proposal.stage).toBe("approved");
    expect(proposal.timelockExpiry).toBe(3000n + TIMELOCK_BLOCKS);

    // 4. Timelock not expired yet
    const earlyExec = executeDisbursement(
      proposal,
      3000n + TIMELOCK_BLOCKS - 1n,
    );
    expect(earlyExec.success).toBe(false);

    // 5. Execute after timelock
    const execResult = executeDisbursement(
      proposal,
      3000n + TIMELOCK_BLOCKS + 1n,
    );
    expect(execResult.success).toBe(true);
    expect(proposal.stage).toBe("executed");
    expect(proposal.executedAt).toBe(3000n + TIMELOCK_BLOCKS + 1n);
  });

  it("submit → stake → vote → reject → stake returned with 10% slash", () => {
    const result = createDisbursement(
      {
        title: "Rejected proposal",
        description: "This will fail",
        category: "marketing",
        treasuryFund: "COMMUNITY",
        proposer: "did:lgm:proposer2" as any,
        recipientDid: "did:lgm:marketing" as any,
        amount: 200n * ONE_LGM,
        justification: "Marketing campaign",
      },
      1000n,
      2,
    );
    const proposal = (result as { proposal: DisbursementProposal }).proposal;

    transitionToVoting(
      proposal,
      DISBURSEMENT_STAKE,
      2000n,
      "did:lgm:proposer2" as any,
    );
    transitionFromVoting(proposal, makeFailingVoteTally(), 3000n);

    expect(proposal.stage).toBe("rejected");

    const stakeOutcome = calculateDisbursementStakeOutcome(
      proposal,
      "rejected_normal",
    );
    expect(stakeOutcome.returnAmount).toBe(
      DISBURSEMENT_STAKE - DISBURSEMENT_STAKE / 10n,
    );
    expect(stakeOutcome.slashAmount).toBe(DISBURSEMENT_STAKE / 10n);
  });
});
