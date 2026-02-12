/**
 * Governance State Layer Tests
 *
 * Tests for GovernanceStateTree CRUD, LIP lifecycle through
 * transaction handlers, quadratic voting, and checkpoint/rollback.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { GovernanceStateTree } from "../../../state/governance.js";
import { StateManager } from "../../../state/state_manager.js";
import { executeTransaction } from "../../../state/transitions.js";
import type {
  Transaction,
  ConsensusParams,
} from "../../../consensus/pot/types.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";

// Test constants
const ONE_LGM = 10n ** 18n;

const TEST_CONSENSUS_PARAMS: ConsensusParams = {
  targetBlockTimeMs: 2521,
  verificationRoundBlocks: 11,
  commitPhaseBlocks: 4,
  revealPhaseBlocks: 4,
  epochLength: 1111n,
  minValidators: 3,
  maxValidators: 22,
  validatorQuorum: 0.66,
  acceptanceThreshold: 0.77,
  rejectionThreshold: 0.77,
  contestedThreshold: 0.55,
  minValidatorStake: 111n * 10n ** 15n, // 0.111 LGM
  minClaimStake: 11n * ONE_LGM,
  blockReward: 11n * ONE_LGM,
  verificationReward: 1n * ONE_LGM,
  citationRewardPerFact: 1n * ONE_LGM,
  wrongVerificationSlash: 0.06,
  missedVerificationSlash: 0.0001,
  invalidClaimSlash: 0.22,
  researchShareRate: 0.22,
  maxTransactionsPerBlock: 111,
  unbondingPeriod: 222222n,
};

const TEST_SENDER = "a]b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6";
const TEST_DID = publicKeyToDid(TEST_SENDER);

function makeLIP(id: string, overrides: Record<string, unknown> = {}) {
  return {
    id,
    title: `Test LIP ${id}`,
    category: "consensus" as const,
    proposer: "did:lgm:test" as string,
    coAuthors: [] as string[],
    createdAt: 100n,
    updatedAt: 100n,
    abstract: "Test abstract",
    motivation: "Test motivation",
    specification: "Test spec",
    rationale: "Test rationale",
    backwardsCompatibility: "None",
    testCases: "Test cases",
    referenceImplementation: undefined as string | undefined,
    stage: "draft" as const,
    stageHistory: [] as any[],
    stakedAmount: 0n,
    stakeHolder: "did:lgm:test" as string,
    amendments: [] as any[],
    ...overrides,
  };
}

function makeVote(voteId: string, overrides: Record<string, unknown> = {}) {
  return {
    voteId,
    voter: "did:lgm:voter1",
    lipId: "LIP-1",
    direction: "support" as const,
    votes: 5,
    cost: 25n * ONE_LGM,
    timestamp: 200n,
    locked: false,
    lockDuration: 0n,
    ...overrides,
  };
}

function createTx(type: string, payload: Record<string, unknown>): Transaction {
  return {
    type: type as any,
    sender: TEST_SENDER as any,
    nonce: 0n,
    fee: 0n,
    payload,
    signature: "" as any,
  };
}

// ============ GovernanceStateTree Unit Tests ============

describe.sequential("GovernanceStateTree", () => {
  let tree: GovernanceStateTree;

  beforeEach(() => {
    tree = new GovernanceStateTree();
  });

  describe.sequential("LIP Operations", () => {
    it("should add and retrieve a LIP", () => {
      tree.addLIP(makeLIP("LIP-1"));
      const retrieved = tree.getLIP("LIP-1");
      expect(retrieved).toBeDefined();
      expect(retrieved!.title).toBe("Test LIP LIP-1");
      expect(retrieved!.category).toBe("consensus");
    });

    it("should reject duplicate LIP IDs", () => {
      tree.addLIP(makeLIP("LIP-1"));
      expect(() => tree.addLIP(makeLIP("LIP-1"))).toThrow(
        "LIP LIP-1 already exists",
      );
    });

    it("should return undefined for unknown LIP", () => {
      expect(tree.getLIP("LIP-999")).toBeUndefined();
    });

    it("should update a LIP", () => {
      tree.addLIP(makeLIP("LIP-1"));
      const updated = makeLIP("LIP-1", {
        stage: "review" as const,
        stakedAmount: 100n * ONE_LGM,
      });
      tree.updateLIP("LIP-1", updated);

      const retrieved = tree.getLIP("LIP-1");
      expect(retrieved!.stage).toBe("review");
      expect(retrieved!.stakedAmount).toBe(100n * ONE_LGM);
    });

    it("should throw when updating nonexistent LIP", () => {
      expect(() => tree.updateLIP("LIP-999", makeLIP("LIP-999"))).toThrow(
        "LIP LIP-999 not found",
      );
    });

    it("should index LIPs by stage", () => {
      tree.addLIP(makeLIP("LIP-1"));
      tree.addLIP(makeLIP("LIP-2"));

      const drafts = tree.getLIPsByStage("draft");
      expect(drafts).toHaveLength(2);

      const reviews = tree.getLIPsByStage("review");
      expect(reviews).toHaveLength(0);
    });

    it("should update stage indexes on LIP update", () => {
      tree.addLIP(makeLIP("LIP-1"));
      expect(tree.getLIPsByStage("draft")).toHaveLength(1);

      tree.updateLIP("LIP-1", makeLIP("LIP-1", { stage: "review" as const }));
      expect(tree.getLIPsByStage("draft")).toHaveLength(0);
      expect(tree.getLIPsByStage("review")).toHaveLength(1);
    });

    it("should index LIPs by proposer", () => {
      tree.addLIP(makeLIP("LIP-1"));
      tree.addLIP(makeLIP("LIP-2", { proposer: "did:lgm:other" }));

      const byProposer = tree.getLIPsByProposer("did:lgm:test");
      expect(byProposer).toHaveLength(1);
      expect(byProposer[0].id).toBe("LIP-1");
    });

    it("should auto-increment LIP counter", () => {
      expect(tree.getNextLIPNumber()).toBe(1);
      tree.addLIP(makeLIP("LIP-1"));
      expect(tree.getNextLIPNumber()).toBe(2);
      tree.addLIP(makeLIP("LIP-5"));
      expect(tree.getNextLIPNumber()).toBe(6);
    });

    it("should compute a merkle root", () => {
      const emptyRoot = tree.getRoot();
      tree.addLIP(makeLIP("LIP-1"));
      const rootAfter = tree.getRoot();
      expect(rootAfter).not.toEqual(emptyRoot);
    });
  });

  describe.sequential("Quadratic Vote Operations", () => {
    it("should add and retrieve a vote", () => {
      tree.addQuadraticVote(makeVote("vote-1"));
      const retrieved = tree.getQuadraticVote("vote-1");
      expect(retrieved).toBeDefined();
      expect(retrieved!.voter).toBe("did:lgm:voter1");
      expect(retrieved!.votes).toBe(5);
    });

    it("should reject duplicate vote IDs", () => {
      tree.addQuadraticVote(makeVote("vote-1"));
      expect(() => tree.addQuadraticVote(makeVote("vote-1"))).toThrow(
        "Vote vote-1 already exists",
      );
    });

    it("should get votes for a LIP", () => {
      tree.addQuadraticVote(makeVote("vote-1"));
      tree.addQuadraticVote(
        makeVote("vote-2", {
          voter: "did:lgm:voter2",
          direction: "oppose" as const,
        }),
      );

      const votes = tree.getVotesForLIP("LIP-1");
      expect(votes).toHaveLength(2);
    });

    it("should get votes by voter", () => {
      tree.addQuadraticVote(makeVote("vote-1"));
      tree.addQuadraticVote(makeVote("vote-2", { lipId: "LIP-2" }));

      const votes = tree.getVotesByVoter("did:lgm:voter1");
      expect(votes).toHaveLength(2);
    });

    it("should check hasVotedOnLIP", () => {
      tree.addQuadraticVote(makeVote("vote-1"));

      expect(tree.hasVotedOnLIP("did:lgm:voter1", "LIP-1")).toBe(true);
      expect(tree.hasVotedOnLIP("did:lgm:voter1", "LIP-2")).toBe(false);
      expect(tree.hasVotedOnLIP("did:lgm:other", "LIP-1")).toBe(false);
    });

    it("should remove a vote", () => {
      tree.addQuadraticVote(makeVote("vote-1"));
      tree.removeVote("vote-1");

      expect(tree.getQuadraticVote("vote-1")).toBeUndefined();
      expect(tree.getVotesForLIP("LIP-1")).toHaveLength(0);
    });
  });

  describe.sequential("Domain Proposal Operations", () => {
    const testProposal = {
      id: "dp-1",
      name: "New Domain",
      description: "A test domain",
      proposer: "did:lgm:test",
      parentDomain: "root",
      stratum: "empirical" as const,
      status: "pending" as const,
      endorsements: [] as string[],
      challenges: [] as string[],
      proposedAt: 100n,
      threshold: 3,
    };

    it("should add and retrieve domain proposals", () => {
      tree.addDomainProposal(testProposal as any);
      const retrieved = tree.getDomainProposal("dp-1");
      expect(retrieved).toBeDefined();
      expect(retrieved!.name).toBe("New Domain");
    });

    it("should reject duplicate domain proposals", () => {
      tree.addDomainProposal(testProposal as any);
      expect(() => tree.addDomainProposal(testProposal as any)).toThrow(
        "already exists",
      );
    });

    it("should update domain proposals", () => {
      tree.addDomainProposal(testProposal as any);
      const updated = { ...testProposal, status: "approved" as const };
      tree.updateDomainProposal(updated as any);

      const retrieved = tree.getDomainProposal("dp-1");
      expect(retrieved!.status).toBe("approved");
    });
  });

  describe.sequential("Statistics", () => {
    it("should track governance stats", () => {
      tree.addLIP(makeLIP("LIP-1", { stakedAmount: 100n * ONE_LGM }));

      const stats = tree.getStats();
      expect(stats.totalLIPs).toBe(1);
      expect(stats.totalStakeLocked).toBe(100n * ONE_LGM);
      expect(stats.lipsByStage["draft"]).toBe(1);
    });
  });

  describe.sequential("Clone / Checkpoint", () => {
    it("should create an independent clone", () => {
      tree.addLIP(makeLIP("LIP-1"));
      const cloned = tree.clone();

      // Modify original — clone should not be affected
      tree.updateLIP("LIP-1", makeLIP("LIP-1", { stage: "review" as const }));

      expect(tree.getLIP("LIP-1")!.stage).toBe("review");
      expect(cloned.getLIP("LIP-1")!.stage).toBe("draft");
    });

    it("should clone vote state independently", () => {
      tree.addQuadraticVote(makeVote("vote-1"));
      const cloned = tree.clone();

      tree.removeVote("vote-1");

      expect(tree.getQuadraticVote("vote-1")).toBeUndefined();
      expect(cloned.getQuadraticVote("vote-1")).toBeDefined();
    });
  });

  describe.sequential("Clear", () => {
    it("should reset all state", () => {
      tree.addLIP(makeLIP("LIP-1"));
      tree.clear();

      expect(tree.getLIP("LIP-1")).toBeUndefined();
      expect(tree.getNextLIPNumber()).toBe(1);
      expect(tree.getStats().totalLIPs).toBe(0);
    });
  });
});

// ============ StateManager Governance Integration ============

describe.sequential("StateManager Governance Integration", () => {
  let state: StateManager;

  beforeEach(() => {
    state = new StateManager();
    state.createAccount(TEST_DID, TEST_SENDER as any, 10000n * ONE_LGM);
  });

  it("should delegate LIP operations through StateManager", () => {
    state.addLIP(
      makeLIP("LIP-1", { proposer: TEST_DID, stakeHolder: TEST_DID }),
    );
    expect(state.getLIP("LIP-1")).toBeDefined();
    expect(state.getLIPsByStage("draft")).toHaveLength(1);
    expect(state.getNextLIPNumber()).toBe(2);
  });

  it("should delegate vote operations through StateManager", () => {
    state.addQuadraticVote(makeVote("vote-sm-1", { voter: TEST_DID }));
    expect(state.getVotesForLIP("LIP-1")).toHaveLength(1);
    expect(state.hasVotedOnLIP(TEST_DID, "LIP-1")).toBe(true);
  });

  it("should include governance in stats", () => {
    state.addLIP(
      makeLIP("LIP-1", {
        proposer: TEST_DID,
        stakeHolder: TEST_DID,
        stakedAmount: 50n * ONE_LGM,
      }),
    );
    const stats = state.getGovernanceStats();
    expect(stats.totalLIPs).toBe(1);
    expect(stats.totalStakeLocked).toBe(50n * ONE_LGM);
  });

  it("should rollback governance state on failed transaction", () => {
    state.addLIP(
      makeLIP("LIP-1", { proposer: TEST_DID, stakeHolder: TEST_DID }),
    );

    state.beginTransaction();
    state.updateLIP(
      "LIP-1",
      makeLIP("LIP-1", {
        stage: "review" as const,
        stakedAmount: 100n * ONE_LGM,
      }),
    );
    expect(state.getLIP("LIP-1")!.stage).toBe("review");

    state.rollbackTransaction();
    expect(state.getLIP("LIP-1")!.stage).toBe("draft");
    expect(state.getLIP("LIP-1")!.stakedAmount).toBe(0n);
  });
});

// ============ Transaction Handler Tests ============

describe.sequential("Governance Transaction Handlers", () => {
  let state: StateManager;

  beforeEach(() => {
    state = new StateManager();
    // Create account with sufficient balance + stake for Tier 1 governance eligibility
    state.createAccount(TEST_DID, TEST_SENDER as any, 10000n * ONE_LGM);
    // Stake 600 LGM (Tier 1 requires 500)
    state.stake(TEST_DID, 600n * ONE_LGM);
    // Set reputation to 0.8 (Tier 1 requires 0.7) and bump nonce for contributions
    // Set reputation and nonce via accounts tree for test setup
    (state as any).accounts.updateAccount(TEST_DID, {
      reputationScore: 0.8,
      nonce: 15n, // contributions >= 10
      lastActiveBlock: 1n, // so accountAge = blockNumber - 1 (0n is falsy)
    });
  });

  describe.sequential("lip_submit", () => {
    it("should create a new LIP in draft stage", () => {
      const tx = createTx("lip_submit", {
        title: "My Proposal",
        category: "consensus",
        proposer: TEST_DID,
        abstract: "A short abstract",
        motivation: "Because we need it",
        specification: "The spec",
        rationale: "It makes sense",
        backwardsCompatibility: "None needed",
        testCases: "Test all the things",
      });

      tx.nonce = 15n; // Match the nonce set in beforeEach
      const result = executeTransaction(
        state,
        tx,
        200000n,
        TEST_CONSENSUS_PARAMS,
      );
      if (!result.success)
        throw new Error(`lip_submit failed: ${result.error}`);
      expect(result.success).toBe(true);

      const lip = state.getLIP("LIP-1");
      expect(lip).toBeDefined();
      expect(lip!.title).toBe("My Proposal");
      expect(lip!.stage).toBe("draft");
      expect(lip!.proposer).toBe(TEST_DID);
    });

    it("should auto-increment LIP IDs", () => {
      const tx1 = createTx("lip_submit", {
        title: "First",
        category: "consensus",
        proposer: TEST_DID,
        abstract: "Test",
        motivation: "Test",
        specification: "Test",
        rationale: "Test",
        backwardsCompatibility: "None",
        testCases: "Test",
      });
      const tx2 = createTx("lip_submit", {
        title: "Second",
        category: "economics",
        proposer: TEST_DID,
        abstract: "Test",
        motivation: "Test",
        specification: "Test",
        rationale: "Test",
        backwardsCompatibility: "None",
        testCases: "Test",
      });

      tx1.nonce = 15n;
      tx2.nonce = 16n;
      executeTransaction(state, tx1, 200000n, TEST_CONSENSUS_PARAMS);
      executeTransaction(state, tx2, 200001n, TEST_CONSENSUS_PARAMS);

      expect(state.getLIP("LIP-1")).toBeDefined();
      expect(state.getLIP("LIP-2")).toBeDefined();
      expect(state.getLIP("LIP-2")!.title).toBe("Second");
    });
  });

  describe.sequential("vote_cast", () => {
    beforeEach(() => {
      const tx = createTx("lip_submit", {
        title: "Votable Proposal",
        category: "consensus",
        proposer: TEST_DID,
        abstract: "Test",
        motivation: "Test",
        specification: "Test",
        rationale: "Test",
        backwardsCompatibility: "None",
        testCases: "Test",
      });
      tx.nonce = 15n;
      executeTransaction(state, tx, 200000n, TEST_CONSENSUS_PARAMS);

      // Manually advance to voting
      const lip = state.getLIP("LIP-1")!;
      state.updateLIP("LIP-1", { ...lip, stage: "voting" as const });
    });

    it("should cast a quadratic vote", () => {
      const tx = createTx("vote_cast", {
        lipId: "LIP-1",
        direction: "support",
        votes: 3,
      });
      tx.nonce = 16n;

      const result = executeTransaction(
        state,
        tx,
        200010n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(true);

      const votes = state.getVotesForLIP("LIP-1");
      expect(votes).toHaveLength(1);
      expect(votes[0].direction).toBe("support");
      expect(votes[0].votes).toBe(3);
    });

    it("should prevent double voting", () => {
      const tx1 = createTx("vote_cast", {
        lipId: "LIP-1",
        direction: "support",
        votes: 2,
      });
      tx1.nonce = 16n;

      const result1 = executeTransaction(
        state,
        tx1,
        200010n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result1.success).toBe(true);

      const tx2 = createTx("vote_cast", {
        lipId: "LIP-1",
        direction: "oppose",
        votes: 1,
      });
      tx2.nonce = 17n;

      const result2 = executeTransaction(
        state,
        tx2,
        200011n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result2.success).toBe(false);
    });

    it("should reject votes on non-voting LIPs", () => {
      const submitTx = createTx("lip_submit", {
        title: "Draft Proposal",
        category: "economics",
        proposer: TEST_DID,
        abstract: "Test",
        motivation: "Test",
        specification: "Test",
        rationale: "Test",
        backwardsCompatibility: "None",
        testCases: "Test",
      });
      submitTx.nonce = 16n;
      executeTransaction(state, submitTx, 200010n, TEST_CONSENSUS_PARAMS);

      const voteTx = createTx("vote_cast", {
        lipId: "LIP-2",
        direction: "support",
        votes: 1,
      });
      voteTx.nonce = 17n;

      const result = executeTransaction(
        state,
        voteTx,
        200011n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(false);
    });
  });

  describe.sequential("lip_withdraw", () => {
    beforeEach(() => {
      const tx = createTx("lip_submit", {
        title: "Withdrawable",
        category: "consensus",
        proposer: TEST_DID,
        abstract: "Test",
        motivation: "Test",
        specification: "Test",
        rationale: "Test",
        backwardsCompatibility: "None",
        testCases: "Test",
      });
      tx.nonce = 15n;
      executeTransaction(state, tx, 200000n, TEST_CONSENSUS_PARAMS);
    });

    it("should withdraw a LIP", () => {
      const tx = createTx("lip_withdraw", {
        lipId: "LIP-1",
      });
      tx.nonce = 16n;

      const result = executeTransaction(
        state,
        tx,
        200010n,
        TEST_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(true);

      const lip = state.getLIP("LIP-1");
      expect(lip!.stage).toBe("withdrawn");
    });
  });
});
