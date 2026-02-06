/**
 * Domain Hierarchy Tests
 *
 * Tests for domain governance fixes:
 * - P0-D1: Authorization in executeDomainGovernanceApprove
 * - P0-D2: Duplicate challenge prevention + stake/reason storage
 * - P1-D1: Deep clone of DomainNodes in governance rollback
 * - P1-D3: Genesis domain deprecation authorization
 * - P1-D4: Proposal expiration for unendorsed proposals
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import { executeTransaction } from "../../../state/transitions.js";
import type { Transaction } from "../../../consensus/pot/types.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../../consensus/pot/types.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";
import {
  challengeProposal,
  finalizeProposal,
  calculateChallengerStakeReturns,
  calculateEndorserStakeReturns,
} from "../../../core/knowledge/ontology/domain_proposal.js";
import type { DomainProposal } from "../../../core/knowledge/ontology/domain_proposal.js";

// ============ Constants ============

const ONE_LGM = 10n ** 18n;

// Sender keys (32-char hex-like strings)
const PROPOSER_KEY = "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6";
const VALIDATOR_KEY = "b1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d7";
const VALIDATOR2_KEY = "c1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d8";
const CHALLENGER_KEY = "d1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d9";
const RANDOM_KEY = "e1b2c3d4e5f6a7b8c9d0e1f2a3b4c5da";

const PROPOSER_DID = publicKeyToDid(PROPOSER_KEY);
const VALIDATOR_DID = publicKeyToDid(VALIDATOR_KEY);
const VALIDATOR2_DID = publicKeyToDid(VALIDATOR2_KEY);
const CHALLENGER_DID = publicKeyToDid(CHALLENGER_KEY);
const RANDOM_DID = publicKeyToDid(RANDOM_KEY);

function createTx(
  type: string,
  sender: string,
  payload: Record<string, unknown>,
  nonce = 0n,
): Transaction {
  return {
    type: type as any,
    sender: sender as any,
    nonce,
    fee: 0n,
    payload,
    signature: "" as any,
  };
}

function makeProposal(overrides: Partial<DomainProposal> = {}): DomainProposal {
  return {
    id: "test-proposal-1",
    proposedDomainId: "physics.test",
    name: "Test Domain",
    stratum: "physical",
    parentDomainId: "physics",
    description: "A test domain",
    examples: ["example claim"],
    proposer: PROPOSER_DID,
    stake: 22n * ONE_LGM,
    status: "pending",
    submittedAt: 100n,
    challengeDeadline: 877n, // 100 + 777
    endorsers: [],
    endorserStakes: new Map(),
    requiredEndorsementStake: 2n * ONE_LGM,
    challengers: [],
    challengerStakes: new Map(),
    challengerReasons: new Map(),
    depth: 2,
    ...overrides,
  };
}

// ============ P0-D1: Governance Approve Authorization ============

describe.sequential(
  "P0-D1: executeDomainGovernanceApprove authorization",
  () => {
    let state: StateManager;

    beforeEach(() => {
      state = new StateManager();
      // Fund accounts
      state.creditBalance(PROPOSER_DID, 10000n * ONE_LGM);
      state.creditBalance(VALIDATOR_DID, 10000n * ONE_LGM);
      state.creditBalance(RANDOM_DID, 10000n * ONE_LGM);

      // Register validator
      state.registerValidator(
        VALIDATOR_KEY as any,
        VALIDATOR_DID,
        2000n * ONE_LGM,
      );

      // Set up a domain proposal in "governance" status
      // Proposer must have staked (the handler unstakes on approval)
      state.stake(PROPOSER_DID, 22n * ONE_LGM);
      const proposal = makeProposal({
        status: "governance",
      });
      state.addDomainProposal(proposal);

      // Set up an accepted LIP
      state.addLIP({
        id: "LIP-domain-1",
        title: "Add test domain",
        category: "consensus",
        proposer: PROPOSER_DID,
        coAuthors: [],
        createdAt: 100n,
        updatedAt: 100n,
        abstract: "Add domain",
        motivation: "Needed",
        specification: "spec",
        rationale: "reason",
        backwardsCompatibility: "None",
        testCases: "None",
        referenceImplementation: undefined,
        stage: "accepted",
        stageHistory: [],
        stakedAmount: 0n,
        stakeHolder: PROPOSER_DID,
        amendments: [],
      } as any);
    });

    it("should reject non-validator governance approval", () => {
      const tx = createTx("domain_governance_approve", RANDOM_KEY, {
        proposalId: "test-proposal-1",
        lipId: "LIP-domain-1",
      });

      const result = executeTransaction(
        state,
        tx,
        1000n,
        DEFAULT_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(false);
      expect(result.error).toContain("Only active validators");
    });

    it("should allow active validator to governance-approve", () => {
      const tx = createTx("domain_governance_approve", VALIDATOR_KEY, {
        proposalId: "test-proposal-1",
        lipId: "LIP-domain-1",
      });

      const result = executeTransaction(
        state,
        tx,
        1000n,
        DEFAULT_CONSENSUS_PARAMS,
      );
      expect(result.success).toBe(true);

      // Proposal should be approved
      const proposal = state.getDomainProposal("test-proposal-1");
      expect(proposal!.status).toBe("approved");
    });
  },
);

// ============ P0-D2: Duplicate Challenge Prevention ============

describe.sequential("P0-D2: challengeProposal duplicate prevention", () => {
  it("should reject duplicate challenge from same DID", () => {
    const proposal = makeProposal({
      challengers: [CHALLENGER_DID],
      challengerStakes: new Map([[CHALLENGER_DID, 5n * ONE_LGM]]),
      challengerReasons: new Map([[CHALLENGER_DID, "first reason"]]),
    });

    expect(() =>
      challengeProposal(
        proposal,
        CHALLENGER_DID,
        "second reason",
        5n * ONE_LGM,
        200n,
      ),
    ).toThrow("Challenger has already challenged this proposal");
  });

  it("should store challenger stake and reason", () => {
    const proposal = makeProposal();

    const updated = challengeProposal(
      proposal,
      CHALLENGER_DID,
      "This domain is too broad",
      10n * ONE_LGM,
      200n,
    );

    expect(updated.challengers).toContain(CHALLENGER_DID);
    expect(updated.challengerStakes.get(CHALLENGER_DID)).toBe(10n * ONE_LGM);
    expect(updated.challengerReasons.get(CHALLENGER_DID)).toBe(
      "This domain is too broad",
    );
  });

  it("should allow different DIDs to challenge", () => {
    const proposal = makeProposal();

    const after1 = challengeProposal(
      proposal,
      CHALLENGER_DID,
      "reason 1",
      5n * ONE_LGM,
      200n,
    );
    const after2 = challengeProposal(
      after1,
      RANDOM_DID,
      "reason 2",
      3n * ONE_LGM,
      200n,
    );

    expect(after2.challengers).toHaveLength(2);
    expect(after2.challengerStakes.size).toBe(2);
    expect(after2.challengerReasons.size).toBe(2);
  });
});

// ============ P0-D2 (cont): Challenger Stake Returns ============

describe.sequential("Challenger stake returns", () => {
  it("should return 100% to challengers on rejection", () => {
    const proposal = makeProposal({
      status: "rejected",
      challengers: [CHALLENGER_DID],
      challengerStakes: new Map([[CHALLENGER_DID, 10n * ONE_LGM]]),
      challengerReasons: new Map([[CHALLENGER_DID, "reason"]]),
    });

    const returns = calculateChallengerStakeReturns(proposal);
    expect(returns.get(CHALLENGER_DID)!.stakeReturn).toBe(10n * ONE_LGM);
    expect(returns.get(CHALLENGER_DID)!.stakeSlashed).toBe(0n);
  });

  it("should slash 23% from challengers on approval", () => {
    const proposal = makeProposal({
      status: "approved",
      challengers: [CHALLENGER_DID],
      challengerStakes: new Map([[CHALLENGER_DID, 100n * ONE_LGM]]),
      challengerReasons: new Map([[CHALLENGER_DID, "reason"]]),
    });

    const returns = calculateChallengerStakeReturns(proposal);
    const result = returns.get(CHALLENGER_DID)!;
    // 77% return = 77 LGM, 23% slashed = 23 LGM
    expect(result.stakeReturn).toBe(77n * ONE_LGM);
    expect(result.stakeSlashed).toBe(23n * ONE_LGM);
  });

  it("should return 100% to challengers on expiration", () => {
    const proposal = makeProposal({
      status: "expired",
      challengers: [CHALLENGER_DID],
      challengerStakes: new Map([[CHALLENGER_DID, 10n * ONE_LGM]]),
      challengerReasons: new Map([[CHALLENGER_DID, "reason"]]),
    });

    const returns = calculateChallengerStakeReturns(proposal);
    expect(returns.get(CHALLENGER_DID)!.stakeReturn).toBe(10n * ONE_LGM);
    expect(returns.get(CHALLENGER_DID)!.stakeSlashed).toBe(0n);
  });
});

// ============ P1-D1: Deep Clone Rollback Safety ============

describe.sequential("P1-D1: Deep clone prevents rollback corruption", () => {
  let state: StateManager;

  beforeEach(() => {
    state = new StateManager();
  });

  it("should not corrupt checkpoint when domain is deprecated after checkpoint", () => {
    // Get the domain tree from state and add a domain
    const tree = state.getKnowledgeDomainTree();
    expect(tree.domainCount).toBe(1); // root only

    // Add a test domain to the tree
    state.addDomainToTree({
      id: "physics",
      name: "Physics",
      stratum: "physical",
      parentDomain: "root",
      childDomains: [],
      dependsOnStrata: ["fundamental"],
      logicZoneDefault: "presburger",
      epistemicCategoryDefault: "empirical",
      description: "Physical sciences",
      examples: [],
      factCount: 0,
      axiomCount: 0,
      createdAt: 1,
    });

    // Checkpoint
    state.beginTransaction();

    // Deprecate domain in active state
    state.updateDomainInTree("physics", { deprecatedAt: 500 });

    // Verify active state is deprecated
    const activeTree = state.getKnowledgeDomainTree();
    const activeNode = activeTree.domainIndex.get("physics");
    expect(activeNode!.domain.deprecatedAt).toBe(500);

    // Rollback
    state.rollbackTransaction();

    // Verify rolled-back state is NOT deprecated
    const rolledBackTree = state.getKnowledgeDomainTree();
    const rolledBackNode = rolledBackTree.domainIndex.get("physics");
    expect(rolledBackNode!.domain.deprecatedAt).toBeUndefined();
  });

  it("should deep clone domain proposal Maps on checkpoint", () => {
    // Add a proposal with challengers
    const proposal = makeProposal({
      challengers: [CHALLENGER_DID],
      challengerStakes: new Map([[CHALLENGER_DID, 5n * ONE_LGM]]),
      challengerReasons: new Map([[CHALLENGER_DID, "original reason"]]),
    });
    state.addDomainProposal(proposal);

    // Checkpoint
    state.beginTransaction();

    // Modify the proposal's Maps in active state
    const activeProposal = state.getDomainProposal("test-proposal-1")!;
    activeProposal.challengerStakes.set(RANDOM_DID, 3n * ONE_LGM);

    // Rollback
    state.rollbackTransaction();

    // The rolled-back proposal should NOT have the new entry
    const rolledBackProposal = state.getDomainProposal("test-proposal-1")!;
    expect(rolledBackProposal.challengerStakes.has(RANDOM_DID)).toBe(false);
    expect(rolledBackProposal.challengerStakes.size).toBe(1);
  });
});

// ============ P1-D3: Genesis Domain Deprecation Guard ============

describe.sequential("P1-D3: Genesis domain deprecation authorization", () => {
  let state: StateManager;

  beforeEach(() => {
    state = new StateManager();
    state.creditBalance(PROPOSER_DID, 10000n * ONE_LGM);
    state.creditBalance(VALIDATOR_DID, 10000n * ONE_LGM);
    state.creditBalance(RANDOM_DID, 10000n * ONE_LGM);

    // Register validator
    state.registerValidator(
      VALIDATOR_KEY as any,
      VALIDATOR_DID,
      2000n * ONE_LGM,
    );

    // Add a genesis domain (no proposal exists for it)
    state.addDomainToTree({
      id: "physics",
      name: "Physics",
      stratum: "physical",
      parentDomain: "root",
      childDomains: [],
      dependsOnStrata: ["fundamental"],
      logicZoneDefault: "presburger",
      epistemicCategoryDefault: "empirical",
      description: "Physical sciences",
      examples: [],
      factCount: 0,
      axiomCount: 0,
      createdAt: 1,
    });
  });

  it("should reject non-validator deprecation of genesis domain", () => {
    const tx = createTx("domain_deprecate", RANDOM_KEY, {
      domainId: "physics",
      reason: "Not needed",
    });

    const result = executeTransaction(
      state,
      tx,
      100n,
      DEFAULT_CONSENSUS_PARAMS,
    );
    expect(result.success).toBe(false);
    expect(result.error).toContain(
      "Genesis domains can only be deprecated by active validators",
    );
  });

  it("should allow active validator to deprecate genesis domain", () => {
    const tx = createTx("domain_deprecate", VALIDATOR_KEY, {
      domainId: "physics",
      reason: "Restructuring taxonomy",
    });

    const result = executeTransaction(
      state,
      tx,
      100n,
      DEFAULT_CONSENSUS_PARAMS,
    );
    expect(result.success).toBe(true);

    // Verify deprecation
    const tree = state.getKnowledgeDomainTree();
    const node = tree.domainIndex.get("physics");
    expect(node!.domain.deprecatedAt).toBeDefined();
  });

  it("should reject non-proposer deprecation of proposed domain", () => {
    // Add an approved proposal for a domain
    const proposal = makeProposal({
      proposedDomainId: "physics",
      status: "approved",
      proposer: PROPOSER_DID,
    });
    state.addDomainProposal(proposal);

    // Random user tries to deprecate
    const tx = createTx("domain_deprecate", RANDOM_KEY, {
      domainId: "physics",
      reason: "Spam",
    });

    const result = executeTransaction(
      state,
      tx,
      100n,
      DEFAULT_CONSENSUS_PARAMS,
    );
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only the original domain proposer");
  });
});

// ============ P1-D4: Proposal Expiration ============

describe.sequential(
  "P1-D4: Proposal expiration for unendorsed proposals",
  () => {
    it("should expire unendorsed unchallenged proposals", () => {
      const proposal = makeProposal({
        status: "pending",
        challengeDeadline: 877n,
      });

      const finalized = finalizeProposal(proposal, 900n);
      expect(finalized.status).toBe("expired");
      expect(finalized.rejectionReason).toContain(
        "did not receive minimum required endorsements",
      );
    });

    it("should reject challenged proposals (not expire)", () => {
      const proposal = makeProposal({
        status: "pending",
        endorsers: [VALIDATOR_DID],
        endorserStakes: new Map([[VALIDATOR_DID, 2n * ONE_LGM]]),
        challengers: [CHALLENGER_DID],
        challengerStakes: new Map([[CHALLENGER_DID, 5n * ONE_LGM]]),
        challengerReasons: new Map([[CHALLENGER_DID, "bad domain"]]),
        challengeDeadline: 877n,
      });

      const finalized = finalizeProposal(proposal, 900n);
      expect(finalized.status).toBe("rejected");
      expect(finalized.rejectionReason).toContain("challenged");
    });

    it("should approve endorsed unchallenged proposals", () => {
      const proposal = makeProposal({
        status: "endorsed",
        endorsers: [VALIDATOR_DID, VALIDATOR2_DID],
        endorserStakes: new Map([
          [VALIDATOR_DID, 2n * ONE_LGM],
          [VALIDATOR2_DID, 2n * ONE_LGM],
        ]),
        challengeDeadline: 877n,
      });

      const finalized = finalizeProposal(proposal, 900n);
      expect(finalized.status).toBe("approved");
    });

    it("should return full endorser stake on expiration", () => {
      const proposal = makeProposal({
        status: "expired",
        endorsers: [VALIDATOR_DID],
        endorserStakes: new Map([[VALIDATOR_DID, 2n * ONE_LGM]]),
      });

      const returns = calculateEndorserStakeReturns(proposal);
      expect(returns.get(VALIDATOR_DID)!.stakeReturn).toBe(2n * ONE_LGM);
      expect(returns.get(VALIDATOR_DID)!.stakeSlashed).toBe(0n);
    });

    it("should not re-finalize already expired proposals", () => {
      const proposal = makeProposal({
        status: "expired",
        challengeDeadline: 877n,
      });

      const result = finalizeProposal(proposal, 1000n);
      expect(result.status).toBe("expired"); // unchanged
    });
  },
);
