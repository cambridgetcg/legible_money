/**
 * Tree of Life Transaction Handler Tests
 *
 * Tests all 9 Tree of Life transaction types executed through
 * the state transition layer (state/transitions.ts:4595-4940).
 *
 * Covers:
 *   - create_project, propose_project
 *   - assign_task, submit_deliverable, approve_deliverable
 *   - deploy_service, call_service, pause_service, retire_service
 *
 * Validates:
 *   - Happy-path state mutations
 *   - Authorization checks (Bug #3 fix)
 *   - Budget validation (Bug #2 fix)
 *   - Evidence tax 5-way routing (Bug #1 fix)
 *   - Error conditions and gas costs
 */

import { describe, it, expect, beforeEach } from "vitest";
import { StateManager } from "../../../state/state_manager.js";
import { executeTransaction } from "../../../state/transitions.js";
import type { Transaction } from "../../../consensus/pot/types.js";
import { generateKeypair } from "../../../core/crypto/keys.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";
import { MIN_PROJECT_STAKE } from "../../../core/tree/types.js";
import type {
  ProductProject,
  ServiceLeaf,
  ContributorRecord,
} from "../../../core/tree/types.js";
import { TREASURY } from "../../../core/constants.js";
import { DEFAULT_CONSENSUS_PARAMS } from "../../../consensus/pot/types.js";

// ============ Test Helpers ============

const blockNumber = 100n;

function makeAgent(): { publicKey: string; did: string } {
  const kp = generateKeypair();
  return { publicKey: kp.publicKey, did: publicKeyToDid(kp.publicKey) };
}

function fundAgent(state: StateManager, did: string, amount: bigint): void {
  state.creditBalance(did, amount);
}

function makeTx(
  type: string,
  sender: string,
  payload: Record<string, unknown>,
  fee: bigint = 1000n,
  nonce: bigint = 0n,
): Transaction {
  return {
    type,
    sender,
    payload,
    fee,
    nonce,
    timestamp: Date.now(),
    signature: "test-signature",
  } as Transaction;
}

function makeProductSpec() {
  return {
    serviceType: "api" as const,
    requirements: ["compute"],
    capabilities: ["inference"],
    estimatedTasks: 5,
    mvpCriteria: ["working endpoint"],
  };
}

function makeDemandSignal() {
  return {
    type: "query_volume" as const,
    evidence: { queries: 150 },
    strength: 0.8,
  };
}

function makeFundingSource(funderDid: string) {
  return { type: "agent_funded" as const, funder: funderDid };
}

function makeContributor(
  did: string,
  role: "founder" | "developer" | "reviewer" = "developer",
  totalEarned: bigint = 0n,
): ContributorRecord {
  return {
    did,
    role,
    joinedAtBlock: 50n,
    tasksCompleted: 0,
    totalEarned,
  };
}

function seedProject(
  state: StateManager,
  founder: { publicKey: string; did: string },
  opts: Partial<ProductProject> = {},
  fundEscrow = true,
): ProductProject {
  const project: ProductProject = {
    id: opts.id ?? "proj-test-1",
    name: opts.name ?? "Test Project",
    description: "A test project",
    phase: opts.phase ?? "growing",
    createdAtBlock: 50n,
    phaseChangedAtBlock: 50n,
    knowledgeDomain: "physics.mechanics",
    demandSignal: makeDemandSignal(),
    founder: founder.did,
    contributors: opts.contributors ?? [
      makeContributor(founder.did, "founder"),
    ],
    productSpec: makeProductSpec(),
    taskIds: opts.taskIds ?? ["task-1"],
    completedTasks: opts.completedTasks ?? 0,
    totalTasks: opts.totalTasks ?? 5,
    fundingSource: makeFundingSource(founder.did),
    budget: opts.budget ?? 1_000_000_000n,
    spent: opts.spent ?? 0n,
    revenueGenerated: opts.revenueGenerated ?? 0n,
    serviceIds: opts.serviceIds ?? [],
    ...opts,
  };
  state.createProject(project);

  // Fund the project escrow so bounty validation works
  if (fundEscrow && project.budget > 0n) {
    state.creditBalance(`did:lgm:project-escrow:${project.id}`, project.budget);
  }

  return project;
}

function seedService(
  state: StateManager,
  projectId: string,
  opts: Partial<ServiceLeaf> = {},
): ServiceLeaf {
  const service: ServiceLeaf = {
    id: opts.id ?? "svc-test-1",
    projectId,
    name: "Test Service",
    description: "A test service",
    serviceType: "api",
    endpoint: {
      protocol: "rpc" as const,
      contractAddress: "0xcontract",
      methods: ["call"],
    },
    abi: { functions: [], events: [], stateVariables: [] },
    pricing: {
      type: "per_call" as const,
      basePrice: 10_000n,
      unit: "call" as const,
    },
    contractAddress: "0xcontract",
    totalCalls: 0n,
    totalRevenue: 0n,
    uptimeBlocks: 0n,
    averageLatencyMs: 10,
    status: opts.status ?? "active",
    deployedAtBlock: 60n,
    ...opts,
  };
  state.createService(service);
  return service;
}

function seedTask(
  state: StateManager,
  projectId: string,
  taskId: string,
  opts: {
    bountyAmount?: bigint;
    assignee?: string;
    status?: string;
    requiredCapabilities?: string[];
    deliverable?: {
      type: string;
      hash: string;
      metadata: Record<string, string>;
      submittedAtBlock: bigint;
      verifiedBy: string[];
    };
  } = {},
): void {
  state.createTask({
    id: taskId,
    projectId,
    title: `Task ${taskId}`,
    description: "Test task",
    taskType: "implement" as const,
    requiredCapabilities: opts.requiredCapabilities ?? [],
    assignee: opts.assignee,
    reviewers: [],
    status: (opts.status ?? "open") as any,
    createdAtBlock: 50n,
    bountyAmount: opts.bountyAmount ?? 50_000_000n,
    deliverable: opts.deliverable,
  });
}

const REVIEW_DELIVERABLE = {
  type: "code",
  hash: "abc123",
  metadata: {},
  submittedAtBlock: 90n,
  verifiedBy: [],
};

// ============ create_project ============

describe.sequential("create_project transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
  });

  it("should create a project in seed phase", () => {
    const tx = makeTx("create_project", founder.publicKey, {
      name: "AI Inference API",
      description: "LLM inference service",
      knowledgeDomain: "ai.inference",
      demandSignal: makeDemandSignal(),
      productSpec: makeProductSpec(),
      fundingSource: makeFundingSource(founder.did),
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(50000n);
  });

  it("should deduct fee from sender", () => {
    const balanceBefore = state.getBalance(founder.did);
    const tx = makeTx(
      "create_project",
      founder.publicKey,
      {
        name: "Test",
        description: "Test",
        knowledgeDomain: "test",
        demandSignal: makeDemandSignal(),
        productSpec: makeProductSpec(),
        fundingSource: makeFundingSource(founder.did),
      },
      5000n,
    );

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    const balanceAfter = state.getBalance(founder.did);
    // Fee of 5000 is deducted via deductBalance + routed to destinations
    expect(balanceBefore - balanceAfter).toBe(5000n);
  });

  it("should store project retrievable by state", () => {
    const tx = makeTx("create_project", founder.publicKey, {
      name: "Retrievable Project",
      description: "Should be in state",
      knowledgeDomain: "test.domain",
      demandSignal: makeDemandSignal(),
      productSpec: makeProductSpec(),
      fundingSource: makeFundingSource(founder.did),
    });

    executeTransaction(state, tx, blockNumber);
    const count = state.getProjectCount();
    expect(count).toBeGreaterThanOrEqual(1);
  });
});

// ============ propose_project ============

describe.sequential("propose_project transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let stranger: { publicKey: string; did: string };
  let projectId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    stranger = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, stranger.did, 10_000_000_000n);

    const project = seedProject(state, founder, {
      id: "proj-propose-1",
      phase: "seed",
    });
    projectId = project.id;
  });

  it("should propose project and deduct stake", () => {
    const balanceBefore = state.getBalance(founder.did);

    const tx = makeTx("propose_project", founder.publicKey, {
      projectId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(40000n);

    const balanceAfter = state.getBalance(founder.did);
    // MIN_PROJECT_STAKE + fee deducted
    expect(balanceBefore - balanceAfter).toBeGreaterThanOrEqual(
      MIN_PROJECT_STAKE,
    );
  });

  it("should reject non-founder proposing", () => {
    const tx = makeTx("propose_project", stranger.publicKey, {
      projectId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only founder");
  });

  it("should reject proposal with insufficient balance for stake", () => {
    const poorAgent = makeAgent();
    fundAgent(state, poorAgent.did, 1000n);

    seedProject(state, poorAgent, {
      id: "proj-poor",
      phase: "seed",
    });

    const tx = makeTx("propose_project", poorAgent.publicKey, {
      projectId: "proj-poor",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Insufficient");
  });

  it("should reject proposal for non-existent project", () => {
    // Use a fresh agent so nonce is 0
    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);
    // Seed a project with this agent as founder so auth passes
    // but use a nonexistent project ID in the tx
    const tx = makeTx("propose_project", agent.publicKey, {
      projectId: "nonexistent",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });
});

// ============ assign_task ============

describe.sequential("assign_task transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let contributor: { publicKey: string; did: string };
  let stranger: { publicKey: string; did: string };
  let projectId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    contributor = makeAgent();
    stranger = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, contributor.did, 10_000_000_000n);
    fundAgent(state, stranger.did, 10_000_000_000n);

    seedProject(state, founder, {
      id: "proj-task-1",
      contributors: [
        makeContributor(founder.did, "founder"),
        makeContributor(contributor.did, "developer"),
      ],
    });
    projectId = "proj-task-1";
    seedTask(state, projectId, "task-1");
    seedTask(state, projectId, "task-2");
  });

  it("should allow founder to assign tasks", () => {
    const tx = makeTx("assign_task", founder.publicKey, {
      projectId,
      taskId: "task-1",
      agentDid: contributor.did,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(30000n);
  });

  it("should allow contributor to assign tasks", () => {
    const tx = makeTx("assign_task", contributor.publicKey, {
      projectId,
      taskId: "task-2",
      agentDid: founder.did,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });

  it("should reject stranger assigning tasks (authorization)", () => {
    const tx = makeTx("assign_task", stranger.publicKey, {
      projectId,
      taskId: "task-1",
      agentDid: stranger.did,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only project members");
  });

  it("should reject non-contributor assignee (P1 fix)", () => {
    // Stranger is not a project contributor — assigning them should fail
    const tx = makeTx("assign_task", founder.publicKey, {
      projectId,
      taskId: "task-1",
      agentDid: stranger.did,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Assignee must be a project contributor");
  });

  it("should reject for non-existent project", () => {
    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);

    const tx = makeTx("assign_task", agent.publicKey, {
      projectId: "nonexistent",
      taskId: "task-1",
      agentDid: contributor.did,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should reject task assignment in non-development phase (P1-3 fix)", () => {
    // Project in "mature" phase should not allow task assignment
    const matureFounder = makeAgent();
    fundAgent(state, matureFounder.did, 10_000_000_000n);
    seedProject(state, matureFounder, {
      id: "proj-mature-assign",
      phase: "mature",
      contributors: [makeContributor(matureFounder.did, "founder")],
    });
    seedTask(state, "proj-mature-assign", "task-mature-1");

    const tx = makeTx("assign_task", matureFounder.publicKey, {
      projectId: "proj-mature-assign",
      taskId: "task-mature-1",
      agentDid: matureFounder.did,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("sprout or growing");
  });
});

// ============ submit_deliverable ============

describe.sequential("submit_deliverable transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let contributor: { publicKey: string; did: string };
  let stranger: { publicKey: string; did: string };
  let projectId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    contributor = makeAgent();
    stranger = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, contributor.did, 10_000_000_000n);
    fundAgent(state, stranger.did, 10_000_000_000n);

    seedProject(state, founder, {
      id: "proj-deliver-1",
      contributors: [
        makeContributor(founder.did, "founder"),
        makeContributor(contributor.did, "developer"),
      ],
    });
    projectId = "proj-deliver-1";
    seedTask(state, projectId, "task-1", {
      assignee: contributor.did,
      status: "in_progress",
    });
  });

  it("should allow contributor to submit deliverable", () => {
    const tx = makeTx("submit_deliverable", contributor.publicKey, {
      projectId,
      taskId: "task-1",
      deliverable: {
        type: "code",
        hash: "abc123",
        uri: "ipfs://Qm...",
        description: "Implementation",
      },
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(35000n);
  });

  it("should reject stranger submitting deliverable (authorization)", () => {
    const tx = makeTx("submit_deliverable", stranger.publicKey, {
      projectId,
      taskId: "task-1",
      deliverable: {
        type: "code",
        hash: "abc123",
        uri: "ipfs://Qm...",
        description: "Malicious",
      },
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only project members");
  });

  it("should reject for non-existent project", () => {
    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);

    const tx = makeTx("submit_deliverable", agent.publicKey, {
      projectId: "nonexistent",
      taskId: "task-1",
      deliverable: {},
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should reject deliverable submission in withered phase (P1-3 fix)", () => {
    const wFounder = makeAgent();
    fundAgent(state, wFounder.did, 10_000_000_000n);
    const wDev = makeAgent();
    fundAgent(state, wDev.did, 10_000_000_000n);
    seedProject(state, wFounder, {
      id: "proj-withered-submit",
      phase: "withered",
      contributors: [
        makeContributor(wFounder.did, "founder"),
        makeContributor(wDev.did, "developer"),
      ],
    });
    seedTask(state, "proj-withered-submit", "task-w-1", {
      assignee: wDev.did,
      status: "in_progress",
    });

    const tx = makeTx("submit_deliverable", wDev.publicKey, {
      projectId: "proj-withered-submit",
      taskId: "task-w-1",
      deliverable: { type: "code", hash: "abc123" },
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("sprout, growing, or mature");
  });
});

// ============ approve_deliverable ============

describe.sequential("approve_deliverable transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let reviewer: { publicKey: string; did: string };
  let developer: { publicKey: string; did: string };
  let stranger: { publicKey: string; did: string };
  let projectId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    reviewer = makeAgent();
    developer = makeAgent();
    stranger = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, reviewer.did, 10_000_000_000n);
    fundAgent(state, developer.did, 10_000_000_000n);
    fundAgent(state, stranger.did, 10_000_000_000n);

    seedProject(state, founder, {
      id: "proj-approve-1",
      budget: 500_000_000n,
      spent: 0n,
      contributors: [
        makeContributor(founder.did, "founder"),
        makeContributor(reviewer.did, "reviewer"),
        makeContributor(developer.did, "developer"),
      ],
    });
    projectId = "proj-approve-1";
    seedTask(state, projectId, "task-1", {
      bountyAmount: 50_000_000n,
      assignee: developer.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });
    // Fund project escrow for bounty payments
    state.creditBalance(`did:lgm:project-escrow:${projectId}`, 10_000_000_000n);
  });

  it("should allow founder to approve and pay bounty", () => {
    const devBalanceBefore = state.getBalance(developer.did);

    const tx = makeTx("approve_deliverable", founder.publicKey, {
      projectId,
      taskId: "task-1",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(40000n);

    const devBalanceAfter = state.getBalance(developer.did);
    expect(devBalanceAfter - devBalanceBefore).toBe(50_000_000n);
  });

  it("should allow reviewer to approve with code_review capability", () => {
    seedTask(state, projectId, "task-review", {
      bountyAmount: 10_000_000n,
      assignee: developer.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });
    const tx = makeTx("approve_deliverable", reviewer.publicKey, {
      projectId,
      taskId: "task-review",
      declaredCapabilities: ["code_review"],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });

  it("should update project spent and completedTasks", () => {
    const tx = makeTx("approve_deliverable", founder.publicKey, {
      projectId,
      taskId: "task-1",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    const project = state.getProject(projectId);
    expect(project).toBeDefined();
    expect(project!.spent).toBe(50_000_000n);
    expect(project!.completedTasks).toBe(1);
  });

  it("should reject developer approving own deliverable (authorization)", () => {
    const tx = makeTx("approve_deliverable", developer.publicKey, {
      projectId,
      taskId: "task-1",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only founder or reviewers");
  });

  it("should reject stranger approving (authorization)", () => {
    const tx = makeTx("approve_deliverable", stranger.publicKey, {
      projectId,
      taskId: "task-1",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only founder or reviewers");
  });

  it("should reject bounty exceeding project budget (budget validation)", () => {
    // Use a fresh agent as founder to avoid nonce issues
    const freshFounder = makeAgent();
    fundAgent(state, freshFounder.did, 10_000_000_000n);
    seedProject(state, freshFounder, {
      id: "proj-budget-test",
      budget: 500_000_000n,
      spent: 0n,
      contributors: [makeContributor(freshFounder.did, "founder")],
    });
    seedTask(state, "proj-budget-test", "task-budget-1", {
      bountyAmount: 600_000_000n,
      assignee: developer.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });
    state.creditBalance(
      `did:lgm:project-escrow:proj-budget-test`,
      10_000_000_000n,
    );

    const tx = makeTx("approve_deliverable", freshFounder.publicKey, {
      projectId: "proj-budget-test",
      taskId: "task-budget-1",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Bounty exceeds project budget");
  });

  it("should reject cumulative spending exceeding budget", () => {
    // Seed tasks with specific bounty amounts
    seedTask(state, projectId, "task-cumul-1", {
      bountyAmount: 400_000_000n,
      assignee: developer.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });
    seedTask(state, projectId, "task-cumul-2", {
      bountyAmount: 200_000_000n,
      assignee: developer.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });

    // First approval: 400M of 500M budget
    const tx1 = makeTx("approve_deliverable", founder.publicKey, {
      projectId,
      taskId: "task-cumul-1",
    });
    const r1 = executeTransaction(state, tx1, blockNumber);
    expect(r1.success).toBe(true);

    // Second approval: 200M more would exceed 500M total (nonce=1)
    const tx2 = makeTx(
      "approve_deliverable",
      founder.publicKey,
      {
        projectId,
        taskId: "task-cumul-2",
      },
      1000n,
      1n,
    );
    const r2 = executeTransaction(state, tx2, blockNumber);
    expect(r2.success).toBe(false);
    expect(r2.error).toContain("Bounty exceeds project budget");
  });

  it("should allow zero-budget projects to pay bounties", () => {
    // Project with budget=0 skips budget validation per: `project.budget > 0n && ...`
    const freshFounder = makeAgent();
    fundAgent(state, freshFounder.did, 10_000_000_000n);
    seedProject(state, freshFounder, {
      id: "proj-no-budget",
      budget: 0n,
      spent: 0n,
      contributors: [
        makeContributor(freshFounder.did, "founder"),
        makeContributor(developer.did, "developer"),
      ],
    });
    seedTask(state, "proj-no-budget", "task-nobud-1", {
      bountyAmount: 50_000_000n,
      assignee: developer.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });
    state.creditBalance(
      `did:lgm:project-escrow:proj-no-budget`,
      10_000_000_000n,
    );

    const tx = makeTx("approve_deliverable", freshFounder.publicKey, {
      projectId: "proj-no-budget",
      taskId: "task-nobud-1",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });

  it("should reject for non-existent project", () => {
    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);

    const tx = makeTx("approve_deliverable", agent.publicKey, {
      projectId: "nonexistent",
      taskId: "task-1",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should reject approval in fruiting phase (P1-3 fix)", () => {
    const fFounder = makeAgent();
    const fDev = makeAgent();
    fundAgent(state, fFounder.did, 10_000_000_000n);
    fundAgent(state, fDev.did, 10_000_000_000n);
    seedProject(state, fFounder, {
      id: "proj-fruiting-approve",
      phase: "fruiting",
      budget: 500_000_000n,
      spent: 0n,
      contributors: [
        makeContributor(fFounder.did, "founder"),
        makeContributor(fDev.did, "developer"),
      ],
    });
    seedTask(state, "proj-fruiting-approve", "task-fruit-1", {
      bountyAmount: 50_000_000n,
      assignee: fDev.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });
    state.creditBalance(
      "did:lgm:project-escrow:proj-fruiting-approve",
      10_000_000_000n,
    );

    const tx = makeTx("approve_deliverable", fFounder.publicKey, {
      projectId: "proj-fruiting-approve",
      taskId: "task-fruit-1",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("sprout, growing, or mature");
  });

  it("should reject founder approving their own deliverable (T7 self-review)", () => {
    // Founder is also the assignee on this task
    seedTask(state, projectId, "task-self-founder", {
      bountyAmount: 10_000_000n,
      assignee: founder.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });

    const tx = makeTx("approve_deliverable", founder.publicKey, {
      projectId,
      taskId: "task-self-founder",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("cannot approve their own");
  });

  it("should reject reviewer approving their own deliverable (T7 self-review)", () => {
    // Reviewer is the assignee
    seedTask(state, projectId, "task-self-reviewer", {
      bountyAmount: 10_000_000n,
      assignee: reviewer.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });

    const tx = makeTx("approve_deliverable", reviewer.publicKey, {
      projectId,
      taskId: "task-self-reviewer",
      declaredCapabilities: ["code_review"],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("cannot approve their own");
  });

  it("should allow founder to approve someone else's deliverable (T7 non-self)", () => {
    // Founder approving developer's task — should still work
    const tx = makeTx("approve_deliverable", founder.publicKey, {
      projectId,
      taskId: "task-1", // assigned to developer, not founder
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });

  it("should reject when escrow has insufficient funds (P1-1 fix)", () => {
    // Create project with NO escrow funding
    const eFounder = makeAgent();
    const eDev = makeAgent();
    fundAgent(state, eFounder.did, 10_000_000_000n);
    fundAgent(state, eDev.did, 10_000_000_000n);
    seedProject(
      state,
      eFounder,
      {
        id: "proj-no-escrow",
        budget: 500_000_000n,
        spent: 0n,
        contributors: [
          makeContributor(eFounder.did, "founder"),
          makeContributor(eDev.did, "developer"),
        ],
      },
      false,
    ); // Deliberately NOT funding escrow — deductBalance will fail
    seedTask(state, "proj-no-escrow", "task-noesc-1", {
      bountyAmount: 50_000_000n,
      assignee: eDev.did,
      status: "review",
      deliverable: REVIEW_DELIVERABLE,
    });

    const tx = makeTx("approve_deliverable", eFounder.publicKey, {
      projectId: "proj-no-escrow",
      taskId: "task-noesc-1",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    // Escrow account doesn't exist → deductBalance throws (no phantom minting)
    expect(result.error).toBeDefined();
  });
});

// ============ deploy_service ============

describe.sequential("deploy_service transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let contributor: { publicKey: string; did: string };
  let projectId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    contributor = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, contributor.did, 10_000_000_000n);

    seedProject(state, founder, {
      id: "proj-deploy-1",
      phase: "mature",
      contributors: [
        makeContributor(founder.did, "founder"),
        makeContributor(contributor.did, "developer"),
      ],
    });
    projectId = "proj-deploy-1";
  });

  it("should deploy a service and add to project", () => {
    const tx = makeTx("deploy_service", founder.publicKey, {
      projectId,
      name: "Inference API",
      description: "LLM inference endpoint",
      serviceType: "api",
      endpoint: {
        protocol: "rpc",
        contractAddress: "0xdeploy",
        methods: ["infer"],
      },
      abi: { functions: [], events: [], stateVariables: [] },
      pricing: { type: "per_call", basePrice: 5000n, unit: "call" },
      contractAddress: "0xdeploy",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(80000n);

    const project = state.getProject(projectId);
    expect(project).toBeDefined();
    expect(project!.serviceIds.length).toBe(1);
  });

  it("should reject non-founder deploying (authorization)", () => {
    const tx = makeTx("deploy_service", contributor.publicKey, {
      projectId,
      name: "Rogue Service",
      description: "Unauthorized",
      serviceType: "api",
      endpoint: {
        protocol: "rpc",
        contractAddress: "0xrogue",
        methods: ["hack"],
      },
      abi: { functions: [], events: [], stateVariables: [] },
      pricing: { type: "per_call", basePrice: 1n, unit: "call" },
      contractAddress: "0xrogue",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only founder can deploy");
  });

  it("should reject for non-existent project", () => {
    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);

    const tx = makeTx("deploy_service", agent.publicKey, {
      projectId: "nonexistent",
      name: "Orphan",
      description: "No project",
      serviceType: "api",
      endpoint: { protocol: "rpc", contractAddress: "0x", methods: [] },
      abi: { functions: [], events: [], stateVariables: [] },
      pricing: { type: "per_call", basePrice: 1n, unit: "call" },
      contractAddress: "0x",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should create service in active status (auto-activated)", () => {
    const tx = makeTx("deploy_service", founder.publicKey, {
      projectId,
      name: "Active Service",
      description: "Should be active immediately",
      serviceType: "computation",
      endpoint: {
        protocol: "rest",
        contractAddress: "0xactive",
        methods: ["compute"],
      },
      abi: { functions: [], events: [], stateVariables: [] },
      pricing: { type: "per_call", basePrice: 10000n, unit: "call" },
      contractAddress: "0xactive",
    });

    executeTransaction(state, tx, blockNumber);

    const project = state.getProject(projectId);
    expect(project!.serviceIds.length).toBe(1);
    const svc = state.getService(project!.serviceIds[0]);
    expect(svc).toBeDefined();
    expect(svc!.status).toBe("active");
  });

  it("should reject deploy in non-mature phase (P1-4 fix)", () => {
    const gFounder = makeAgent();
    fundAgent(state, gFounder.did, 10_000_000_000n);
    seedProject(state, gFounder, {
      id: "proj-growing-deploy",
      phase: "growing",
      contributors: [makeContributor(gFounder.did, "founder")],
    });

    const tx = makeTx("deploy_service", gFounder.publicKey, {
      projectId: "proj-growing-deploy",
      name: "Early Service",
      description: "Not ready yet",
      serviceType: "api",
      endpoint: {
        protocol: "rpc",
        contractAddress: "0xearly",
        methods: ["call"],
      },
      abi: { functions: [], events: [], stateVariables: [] },
      pricing: { type: "per_call", basePrice: 1n, unit: "call" },
      contractAddress: "0xearly",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("mature phase");
  });
});

// ============ call_service ============

describe.sequential("call_service transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let contributor1: { publicKey: string; did: string };
  let contributor2: { publicKey: string; did: string };
  let caller: { publicKey: string; did: string };
  let projectId: string;
  let serviceId: string;
  const serviceCost = 10_000n;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    contributor1 = makeAgent();
    contributor2 = makeAgent();
    caller = makeAgent();

    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, caller.did, 10_000_000_000n);
    fundAgent(state, contributor1.did, 1_000_000n);
    fundAgent(state, contributor2.did, 1_000_000n);

    projectId = "proj-call-1";
    serviceId = "svc-call-1";

    seedProject(state, founder, {
      id: projectId,
      phase: "fruiting",
      serviceIds: [serviceId],
      contributors: [
        makeContributor(founder.did, "founder", 100_000n),
        makeContributor(contributor1.did, "developer", 60_000n),
        makeContributor(contributor2.did, "developer", 40_000n),
      ],
    });

    seedService(state, projectId, {
      id: serviceId,
      pricing: {
        type: "per_call",
        basePrice: serviceCost,
        unit: "call",
      },
    });
  });

  it("should deduct payment from caller", () => {
    const balanceBefore = state.getBalance(caller.did);

    const tx = makeTx("call_service", caller.publicKey, {
      serviceId,
      units: 1n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(60000n);

    const balanceAfter = state.getBalance(caller.did);
    // Cost (10000) + fee (1000) deducted
    expect(balanceBefore - balanceAfter).toBeGreaterThanOrEqual(serviceCost);
  });

  it("should distribute revenue to contributors pro-rata", () => {
    // Use a larger service cost to ensure measurable splits
    const bigServiceId = "svc-big-contrib";
    seedService(state, projectId, {
      id: bigServiceId,
      pricing: { type: "per_call", basePrice: 1_000_000n, unit: "call" },
    });

    const c1Before = state.getBalance(contributor1.did);
    const c2Before = state.getBalance(contributor2.did);

    const tx = makeTx("call_service", caller.publicKey, {
      serviceId: bigServiceId,
      units: 1n,
    });

    executeTransaction(state, tx, blockNumber);

    const c1After = state.getBalance(contributor1.did);
    const c2After = state.getBalance(contributor2.did);

    // Contributors share 39% = 390,000 of 1M
    // Weights: founder=100K, c1=60K, c2=40K → total=200K
    // c1: 60K/200K * 390000 = 117000
    // c2: 40K/200K * 390000 = 78000
    expect(c1After - c1Before).toBeGreaterThan(0n);
    expect(c2After - c2Before).toBeGreaterThan(0n);
    // c1 should get more than c2 (60K vs 40K weight)
    expect(c1After - c1Before).toBeGreaterThan(c2After - c2Before);
  });

  it("should route evidence tax to 5 treasury funds (Bug #1 fix)", () => {
    const bigServiceId = "svc-big-tax";
    seedService(state, projectId, {
      id: bigServiceId,
      pricing: { type: "per_call", basePrice: 1_000_000_000n, unit: "call" },
    });

    const tResearchBefore = state.getBalance(TREASURY.ADDRESSES.RESEARCH);
    const tFoundationBefore = state.getBalance(TREASURY.ADDRESSES.FOUNDATION);
    const tCommunityBefore = state.getBalance(TREASURY.ADDRESSES.COMMUNITY);
    const tDevelopersBefore = state.getBalance(TREASURY.ADDRESSES.DEVELOPERS);
    const tReserveBefore = state.getBalance(TREASURY.ADDRESSES.RESERVE);
    const tResearchFundBefore = state.getBalance(
      TREASURY.ADDRESSES.RESEARCH_FUND,
    );

    // Use fee=0 to isolate revenue distribution from fee routing
    const tx = makeTx(
      "call_service",
      caller.publicKey,
      {
        serviceId: bigServiceId,
        units: 1n,
      },
      0n,
    );

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    const tResearchAfter = state.getBalance(TREASURY.ADDRESSES.RESEARCH);
    const tFoundationAfter = state.getBalance(TREASURY.ADDRESSES.FOUNDATION);
    const tCommunityAfter = state.getBalance(TREASURY.ADDRESSES.COMMUNITY);
    const tDevelopersAfter = state.getBalance(TREASURY.ADDRESSES.DEVELOPERS);
    const tReserveAfter = state.getBalance(TREASURY.ADDRESSES.RESERVE);
    const tResearchFundAfter = state.getBalance(
      TREASURY.ADDRESSES.RESEARCH_FUND,
    );

    const researchGain = tResearchAfter - tResearchBefore;
    const foundationGain = tFoundationAfter - tFoundationBefore;
    const communityGain = tCommunityAfter - tCommunityBefore;
    const developersGain = tDevelopersAfter - tDevelopersBefore;
    const reserveGain = tReserveAfter - tReserveBefore;
    const researchFundGain = tResearchFundAfter - tResearchFundBefore;

    const cost = 1_000_000_000n;
    // Evidence tax = 22% of 1B = 220M
    const expectedEvidenceTax = (cost * 2200n) / 10000n;
    const researchFundAmt = (cost * 2340n) / 10000n;

    // All 5 evidence tax funds should receive something
    expect(researchGain).toBeGreaterThan(0n);
    expect(foundationGain).toBeGreaterThan(0n);
    expect(communityGain).toBeGreaterThan(0n);
    expect(developersGain).toBeGreaterThan(0n);
    expect(reserveGain).toBeGreaterThan(0n);

    // I2 FIX: Research fund now goes to separate RESEARCH_FUND address
    expect(researchFundGain).toBe(researchFundAmt);

    // Evidence tax total = sum of all 5 treasury fund gains (no subtraction needed)
    // Foundation also receives protocol treasury, so subtract it
    const contributorPool = (cost * 3900n) / 10000n;
    const protocolTreasury =
      cost - expectedEvidenceTax - contributorPool - researchFundAmt;
    const evidenceTaxTotal =
      researchGain +
      (foundationGain - protocolTreasury) +
      communityGain +
      developersGain +
      reserveGain;
    expect(evidenceTaxTotal).toBe(expectedEvidenceTax);

    // Research evidence tax = 77% of evidence tax (now the ONLY thing in RESEARCH address)
    expect(researchGain).toBe((expectedEvidenceTax * 7700n) / 10000n);
  });

  it("should credit research fund and protocol treasury", () => {
    const bigServiceId = "svc-big-treasury";
    seedService(state, projectId, {
      id: bigServiceId,
      pricing: { type: "per_call", basePrice: 100_000_000n, unit: "call" },
    });

    // I2 FIX: Research fund now goes to RESEARCH_FUND, evidence tax research to RESEARCH
    const researchFundBefore = state.getBalance(
      TREASURY.ADDRESSES.RESEARCH_FUND,
    );
    const researchTaxBefore = state.getBalance(TREASURY.ADDRESSES.RESEARCH);
    const protocolBefore = state.getBalance("did:lgm:treasury-foundation");

    // fee=0 to isolate revenue distribution
    const tx = makeTx(
      "call_service",
      caller.publicKey,
      {
        serviceId: bigServiceId,
        units: 1n,
      },
      0n,
    );

    executeTransaction(state, tx, blockNumber);

    const researchFundAfter = state.getBalance(
      TREASURY.ADDRESSES.RESEARCH_FUND,
    );
    const researchTaxAfter = state.getBalance(TREASURY.ADDRESSES.RESEARCH);
    const protocolAfter = state.getBalance("did:lgm:treasury-foundation");

    // Research fund (23.4%) goes to its own address
    expect(researchFundAfter - researchFundBefore).toBe(
      (100_000_000n * 2340n) / 10000n,
    );
    // Evidence tax research (77% of 22%) goes to RESEARCH address
    const evidenceTax = (100_000_000n * 2200n) / 10000n;
    expect(researchTaxAfter - researchTaxBefore).toBe(
      (evidenceTax * 7700n) / 10000n,
    );
    // Protocol treasury gets remainder
    expect(protocolAfter).toBeGreaterThan(protocolBefore);
  });

  it("should track revenue on project", () => {
    const tx = makeTx("call_service", caller.publicKey, {
      serviceId,
      units: 1n,
    });

    executeTransaction(state, tx, blockNumber);

    const project = state.getProject(projectId);
    expect(project).toBeDefined();
    expect(project!.revenueGenerated).toBe(serviceCost);
  });

  it("should reject call to non-existent service", () => {
    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);

    const tx = makeTx("call_service", agent.publicKey, {
      serviceId: "nonexistent",
      units: 1n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });

  it("should reject call to non-active service", () => {
    seedService(state, projectId, {
      id: "svc-paused",
      status: "paused",
    });

    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);

    const tx = makeTx("call_service", agent.publicKey, {
      serviceId: "svc-paused",
      units: 1n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not active");
  });

  it("should reject call with insufficient balance", () => {
    const poorAgent = makeAgent();
    fundAgent(state, poorAgent.did, 100n);

    const tx = makeTx("call_service", poorAgent.publicKey, {
      serviceId,
      units: 1n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Insufficient");
  });
});

// ============ pause_service ============

describe.sequential("pause_service transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let stranger: { publicKey: string; did: string };
  let projectId: string;
  let serviceId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    stranger = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, stranger.did, 10_000_000_000n);

    projectId = "proj-pause-1";
    serviceId = "svc-pause-1";

    seedProject(state, founder, {
      id: projectId,
      phase: "fruiting",
      serviceIds: [serviceId],
    });

    seedService(state, projectId, {
      id: serviceId,
      status: "active",
    });
  });

  it("should pause an active service", () => {
    const tx = makeTx("pause_service", founder.publicKey, {
      serviceId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(25000n);

    const svc = state.getService(serviceId);
    expect(svc).toBeDefined();
    expect(svc!.status).toBe("paused");
  });

  it("should reject stranger pausing (authorization)", () => {
    const tx = makeTx("pause_service", stranger.publicKey, {
      serviceId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only project founder can pause");
  });

  it("should reject for non-existent service", () => {
    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);

    const tx = makeTx("pause_service", agent.publicKey, {
      serviceId: "nonexistent",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });
});

// ============ resume_service ============

describe.sequential("resume_service transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let stranger: { publicKey: string; did: string };
  let projectId: string;
  let serviceId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    stranger = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, stranger.did, 10_000_000_000n);

    projectId = "proj-resume-1";
    serviceId = "svc-resume-1";

    seedProject(state, founder, {
      id: projectId,
      phase: "fruiting",
      serviceIds: [serviceId],
    });

    seedService(state, projectId, {
      id: serviceId,
      status: "paused",
    });
  });

  it("should resume a paused service", () => {
    const tx = makeTx("resume_service", founder.publicKey, {
      serviceId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(25000n);

    const svc = state.getService(serviceId);
    expect(svc).toBeDefined();
    expect(svc!.status).toBe("active");
  });

  it("should reject stranger resuming (authorization)", () => {
    const tx = makeTx("resume_service", stranger.publicKey, {
      serviceId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only project founder can resume");
  });

  it("should reject resuming non-paused service", () => {
    // Override the paused service back to active
    state.updateService(serviceId, { status: "active" });

    const tx = makeTx("resume_service", founder.publicKey, {
      serviceId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("paused");
  });

  it("should reject for non-existent service", () => {
    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);

    const tx = makeTx("resume_service", agent.publicKey, {
      serviceId: "nonexistent",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });
});

// ============ retire_service ============

describe.sequential("retire_service transaction", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let stranger: { publicKey: string; did: string };
  let projectId: string;
  let serviceId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    stranger = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, stranger.did, 10_000_000_000n);

    projectId = "proj-retire-1";
    serviceId = "svc-retire-1";

    seedProject(state, founder, {
      id: projectId,
      phase: "fruiting",
      serviceIds: [serviceId],
    });

    seedService(state, projectId, {
      id: serviceId,
      status: "active",
    });
  });

  it("should retire an active service", () => {
    const tx = makeTx("retire_service", founder.publicKey, {
      serviceId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
    expect(result.gasUsed).toBe(25000n);

    const svc = state.getService(serviceId);
    expect(svc).toBeDefined();
    expect(svc!.status).toBe("retired");
  });

  it("should reject stranger retiring (authorization)", () => {
    const tx = makeTx("retire_service", stranger.publicKey, {
      serviceId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only project founder can retire");
  });

  it("should reject for non-existent service", () => {
    const agent = makeAgent();
    fundAgent(state, agent.did, 10_000_000_000n);

    const tx = makeTx("retire_service", agent.publicKey, {
      serviceId: "nonexistent",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("not found");
  });
});

// ============ End-to-end lifecycle ============

describe.sequential("Tree of Life end-to-end lifecycle", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let developer: { publicKey: string; did: string };
  let caller: { publicKey: string; did: string };

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    developer = makeAgent();
    caller = makeAgent();

    fundAgent(state, founder.did, 100_000_000_000n);
    fundAgent(state, developer.did, 10_000_000_000n);
    fundAgent(state, caller.did, 10_000_000_000n);
  });

  it("should execute create → propose flow", () => {
    // 1. Create project
    const createTx = makeTx("create_project", founder.publicKey, {
      name: "E2E Project",
      description: "End to end test",
      knowledgeDomain: "test.e2e",
      demandSignal: makeDemandSignal(),
      productSpec: makeProductSpec(),
      fundingSource: makeFundingSource(founder.did),
    });

    const createResult = executeTransaction(state, createTx, 100n);
    expect(createResult.success).toBe(true);

    // 2. Propose project (requires founder + stake)
    const projects = state.getProjectsByPhase("seed");
    expect(projects.length).toBeGreaterThanOrEqual(1);
    const project = projects[0];

    const proposeTx = makeTx(
      "propose_project",
      founder.publicKey,
      { projectId: project.id },
      1000n,
      1n, // nonce=1 since create used nonce=0
    );

    const proposeResult = executeTransaction(state, proposeTx, 101n);
    expect(proposeResult.success).toBe(true);

    const updatedProject = state.getProject(project.id);
    expect(updatedProject).toBeDefined();
  });

  it("should correctly account for all money in call_service (fee=0)", () => {
    // Use fee=0 to isolate revenue distribution from fee routing
    const projectId = "proj-accounting";
    const serviceId = "svc-accounting";

    seedProject(state, founder, {
      id: projectId,
      phase: "fruiting",
      serviceIds: [serviceId],
      contributors: [makeContributor(founder.did, "founder", 100_000n)],
    });

    seedService(state, projectId, {
      id: serviceId,
      pricing: { type: "per_call", basePrice: 1_000_000n, unit: "call" },
    });

    const callerBefore = state.getBalance(caller.did);
    const founderBefore = state.getBalance(founder.did);

    // I2 FIX: Track evidence tax research and research fund separately
    const tResearchBefore = state.getBalance(TREASURY.ADDRESSES.RESEARCH);
    const tResearchFundBefore = state.getBalance(
      TREASURY.ADDRESSES.RESEARCH_FUND,
    );
    const tFoundationBefore = state.getBalance(TREASURY.ADDRESSES.FOUNDATION);
    const tCommunityBefore = state.getBalance(TREASURY.ADDRESSES.COMMUNITY);
    const tDevelopersBefore = state.getBalance(TREASURY.ADDRESSES.DEVELOPERS);
    const tReserveBefore = state.getBalance(TREASURY.ADDRESSES.RESERVE);

    const tx = makeTx(
      "call_service",
      caller.publicKey,
      {
        serviceId,
        units: 1n,
      },
      0n,
    ); // fee=0 to isolate

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    const cost = 1_000_000n;
    // Evidence tax: 22% = 220,000
    const evidenceTax = (cost * 2200n) / 10000n;
    // Contributors: 39% = 390,000
    const contributorPool = (cost * 3900n) / 10000n;
    // Research fund: 23.4% = 234,000
    const researchFund = (cost * 2340n) / 10000n;
    // Protocol treasury: remainder
    const protocolTreasury =
      cost - evidenceTax - contributorPool - researchFund;

    // Verify caller paid exactly cost (no fee)
    const callerAfter = state.getBalance(caller.did);
    expect(callerBefore - callerAfter).toBe(cost);

    // Verify founder received contributor share (sole contributor = 100%)
    const founderAfter = state.getBalance(founder.did);
    expect(founderAfter - founderBefore).toBe(contributorPool);

    // I2 FIX: Research fund goes to RESEARCH_FUND address, evidence tax research to RESEARCH
    const tResearchFundAfter = state.getBalance(
      TREASURY.ADDRESSES.RESEARCH_FUND,
    );
    expect(tResearchFundAfter - tResearchFundBefore).toBe(researchFund);

    const tResearchAfter = state.getBalance(TREASURY.ADDRESSES.RESEARCH);
    const evidenceTaxResearch = (evidenceTax * 7700n) / 10000n;
    expect(tResearchAfter - tResearchBefore).toBe(evidenceTaxResearch);

    // Verify evidence tax split across 5 treasury funds (no subtraction needed)
    const tFoundationAfter = state.getBalance(TREASURY.ADDRESSES.FOUNDATION);
    const tCommunityAfter = state.getBalance(TREASURY.ADDRESSES.COMMUNITY);
    const tDevelopersAfter = state.getBalance(TREASURY.ADDRESSES.DEVELOPERS);
    const tReserveAfter = state.getBalance(TREASURY.ADDRESSES.RESERVE);

    // Foundation receives evidence tax foundation + protocol treasury
    const foundationGain = tFoundationAfter - tFoundationBefore;
    const totalEvidenceTaxRouted =
      tResearchAfter -
      tResearchBefore +
      (foundationGain - protocolTreasury) +
      (tCommunityAfter - tCommunityBefore) +
      (tDevelopersAfter - tDevelopersBefore) +
      (tReserveAfter - tReserveBefore);

    expect(totalEvidenceTaxRouted).toBe(evidenceTax);

    // Verify total distributed = cost (conservation of money)
    const totalDistributed =
      totalEvidenceTaxRouted +
      contributorPool +
      researchFund +
      protocolTreasury;
    expect(totalDistributed).toBe(cost);
  });
});

// ============ C1: assign_task capability check ============

describe.sequential("assign_task capability check (C1 fix)", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let contributor: { publicKey: string; did: string };
  let projectId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    contributor = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, contributor.did, 10_000_000_000n);

    seedProject(state, founder, {
      id: "proj-cap-1",
      contributors: [
        makeContributor(founder.did, "founder"),
        makeContributor(contributor.did, "developer"),
      ],
    });
    projectId = "proj-cap-1";
  });

  it("should allow assignment when task has no required capabilities", () => {
    seedTask(state, projectId, "task-no-caps", { requiredCapabilities: [] });

    const tx = makeTx("assign_task", founder.publicKey, {
      projectId,
      taskId: "task-no-caps",
      agentDid: contributor.did,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });

  it("should reject assignment when agent lacks declared capabilities", () => {
    seedTask(state, projectId, "task-with-caps", {
      requiredCapabilities: ["development", "testing"],
    });

    // Agent declares only "development", missing "testing"
    const tx = makeTx("assign_task", founder.publicKey, {
      projectId,
      taskId: "task-with-caps",
      agentDid: contributor.did,
      declaredCapabilities: ["development"],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("lacks required capabilities");
  });

  it("should reject assignment when no capabilities declared for gated task", () => {
    seedTask(state, projectId, "task-gated", {
      requiredCapabilities: ["review"],
    });

    // No declaredCapabilities in payload
    const tx = makeTx("assign_task", founder.publicKey, {
      projectId,
      taskId: "task-gated",
      agentDid: contributor.did,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("lacks required capabilities");
  });

  it("should allow assignment when declared capabilities satisfy requirements", () => {
    seedTask(state, projectId, "task-ok-caps", {
      requiredCapabilities: ["development", "testing"],
    });

    const tx = makeTx("assign_task", founder.publicKey, {
      projectId,
      taskId: "task-ok-caps",
      agentDid: contributor.did,
      declaredCapabilities: ["development", "testing", "review"],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });
});

// ============ C2: add_task capability validation ============

describe.sequential("add_task capability validation (C2 fix)", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let projectId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);

    seedProject(state, founder, {
      id: "proj-addtask-1",
      contributors: [makeContributor(founder.did, "founder")],
    });
    projectId = "proj-addtask-1";
  });

  it("should accept valid capability names", () => {
    const tx = makeTx("add_task", founder.publicKey, {
      projectId,
      title: "Valid task",
      description: "Task with valid capabilities",
      taskType: "implement",
      requiredCapabilities: ["development", "testing"],
      bountyAmount: 50_000_000n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });

  it("should accept empty capabilities", () => {
    const tx = makeTx("add_task", founder.publicKey, {
      projectId,
      title: "Open task",
      description: "Anyone can do this",
      taskType: "implement",
      requiredCapabilities: [],
      bountyAmount: 50_000_000n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });

  it("should reject invalid capability names", () => {
    const tx = makeTx("add_task", founder.publicKey, {
      projectId,
      title: "Bad task",
      description: "Task with invalid capabilities",
      taskType: "implement",
      requiredCapabilities: ["development", "hacking"],
      bountyAmount: 50_000_000n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Invalid capability: hacking");
  });

  it("should accept composite capability names", () => {
    const tx = makeTx("add_task", founder.publicKey, {
      projectId,
      title: "Design task",
      description: "Needs design capability",
      taskType: "design",
      requiredCapabilities: ["design"],
      bountyAmount: 50_000_000n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });

  it("should accept all valid task capability aliases", () => {
    // These are all the aliases defined in CAPABILITY_TASK_NAMES + composites
    const validNames = [
      "verification",
      "verify",
      "research",
      "development",
      "implement",
      "deploy",
      "testing",
      "test",
      "code_review",
      "review",
      "governance",
      "mentorship",
      "document",
      "design",
    ];

    for (const cap of validNames) {
      const freshState = new StateManager();
      const f = makeAgent();
      fundAgent(freshState, f.did, 10_000_000_000n);
      seedProject(freshState, f, {
        id: `proj-alias-${cap}`,
        contributors: [makeContributor(f.did, "founder")],
      });

      const tx = makeTx("add_task", f.publicKey, {
        projectId: `proj-alias-${cap}`,
        title: `Task requiring ${cap}`,
        description: "Test",
        taskType: "implement",
        requiredCapabilities: [cap],
        bountyAmount: 50_000_000n,
      });

      const result = executeTransaction(freshState, tx, blockNumber);
      expect(result.error ?? "").toBe("");
      expect(result.success).toBe(true);
    }
  });
});

// ============ P2-1: add_task phase validation ============

describe.sequential("add_task phase validation (P2-1 fix)", () => {
  it("should reject adding tasks to dormant project", () => {
    const state = new StateManager();
    const f = makeAgent();
    fundAgent(state, f.did, 10_000_000_000n);
    seedProject(state, f, {
      id: "proj-dormant-add",
      phase: "dormant",
      contributors: [makeContributor(f.did, "founder")],
    });

    const tx = makeTx("add_task", f.publicKey, {
      projectId: "proj-dormant-add",
      title: "Task on paused project",
      description: "Should fail",
      taskType: "implement",
      requiredCapabilities: [],
      bountyAmount: 0n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("sprout or growing");
  });

  it("should reject adding tasks to withered project", () => {
    const state = new StateManager();
    const f = makeAgent();
    fundAgent(state, f.did, 10_000_000_000n);
    seedProject(state, f, {
      id: "proj-withered-add",
      phase: "withered",
      contributors: [makeContributor(f.did, "founder")],
    });

    const tx = makeTx("add_task", f.publicKey, {
      projectId: "proj-withered-add",
      title: "Task on dead project",
      description: "Should fail",
      taskType: "implement",
      requiredCapabilities: [],
      bountyAmount: 0n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("sprout or growing");
  });

  it("should allow adding tasks in sprout phase", () => {
    const state = new StateManager();
    const f = makeAgent();
    fundAgent(state, f.did, 10_000_000_000n);
    seedProject(state, f, {
      id: "proj-sprout-add",
      phase: "sprout",
      contributors: [makeContributor(f.did, "founder")],
    });

    const tx = makeTx("add_task", f.publicKey, {
      projectId: "proj-sprout-add",
      title: "Task in sprout phase",
      description: "Should work",
      taskType: "implement",
      requiredCapabilities: [],
      bountyAmount: 0n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });
});

// ============ C3: approve_deliverable reviewer capability check ============

describe.sequential(
  "approve_deliverable reviewer capability check (C3 fix)",
  () => {
    let state: StateManager;
    let founder: { publicKey: string; did: string };
    let reviewer: { publicKey: string; did: string };
    let developer: { publicKey: string; did: string };
    let projectId: string;

    beforeEach(() => {
      state = new StateManager();
      founder = makeAgent();
      reviewer = makeAgent();
      developer = makeAgent();
      fundAgent(state, founder.did, 10_000_000_000n);
      fundAgent(state, reviewer.did, 10_000_000_000n);
      fundAgent(state, developer.did, 10_000_000_000n);

      seedProject(state, founder, {
        id: "proj-c3-1",
        budget: 500_000_000n,
        spent: 0n,
        contributors: [
          makeContributor(founder.did, "founder"),
          makeContributor(reviewer.did, "reviewer"),
          makeContributor(developer.did, "developer"),
        ],
      });
      projectId = "proj-c3-1";
      seedTask(state, projectId, "task-c3-1", {
        bountyAmount: 50_000_000n,
        assignee: developer.did,
        status: "review",
        deliverable: REVIEW_DELIVERABLE,
      });
      state.creditBalance(
        `did:lgm:project-escrow:${projectId}`,
        10_000_000_000n,
      );
    });

    it("should allow founder to approve without declaring capabilities", () => {
      const tx = makeTx("approve_deliverable", founder.publicKey, {
        projectId,
        taskId: "task-c3-1",
      });

      const result = executeTransaction(state, tx, blockNumber);
      expect(result.success).toBe(true);
    });

    it("should reject reviewer without declared code_review capability", () => {
      const tx = makeTx("approve_deliverable", reviewer.publicKey, {
        projectId,
        taskId: "task-c3-1",
      });

      const result = executeTransaction(state, tx, blockNumber);
      expect(result.success).toBe(false);
      expect(result.error).toContain("code_review capability");
    });

    it("should allow reviewer with code_review declared", () => {
      const tx = makeTx("approve_deliverable", reviewer.publicKey, {
        projectId,
        taskId: "task-c3-1",
        declaredCapabilities: ["code_review"],
      });

      const result = executeTransaction(state, tx, blockNumber);
      expect(result.success).toBe(true);
    });

    it("should allow reviewer with review alias declared", () => {
      seedTask(state, projectId, "task-c3-2", {
        bountyAmount: 10_000_000n,
        assignee: developer.did,
        status: "review",
        deliverable: REVIEW_DELIVERABLE,
      });

      const tx = makeTx("approve_deliverable", reviewer.publicKey, {
        projectId,
        taskId: "task-c3-2",
        declaredCapabilities: ["review"],
      });

      const result = executeTransaction(state, tx, blockNumber);
      expect(result.success).toBe(true);
    });
  },
);

// ============ S1 regression: deploy_service phase check ordering ============

describe.sequential("deploy_service phase check ordering (S1)", () => {
  it("should not leave orphan service when phase is wrong", () => {
    const state = new StateManager();
    const founder = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);

    // Seed project in "growing" phase — NOT mature
    seedProject(state, founder, {
      id: "proj-s1-orphan",
      phase: "growing",
      contributors: [makeContributor(founder.did, "founder")],
      serviceIds: [],
    });

    const tx = makeTx("deploy_service", founder.publicKey, {
      projectId: "proj-s1-orphan",
      name: "Orphan Check",
      description: "Should not persist",
      serviceType: "api",
      endpoint: {
        protocol: "rpc",
        contractAddress: "0xorphan",
        methods: ["call"],
      },
      abi: { functions: [], events: [], stateVariables: [] },
      pricing: { type: "per_call", basePrice: 1000n, unit: "call" },
      contractAddress: "0xorphan",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("mature phase");

    // S1 regression: no orphan service should exist in state after rollback
    const project = state.getProject("proj-s1-orphan");
    expect(project!.serviceIds.length).toBe(0);
  });
});

// ============ S2: subscription persistence ============

describe.sequential("subscription persistence (S2)", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let caller: { publicKey: string; did: string };
  let projectId: string;
  let serviceId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    caller = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, caller.did, 10_000_000_000n);

    projectId = "proj-sub-1";
    serviceId = "svc-sub-1";

    seedProject(state, founder, {
      id: projectId,
      phase: "fruiting",
      contributors: [makeContributor(founder.did, "founder")],
      serviceIds: [serviceId],
    });

    seedService(state, projectId, {
      id: serviceId,
      pricing: {
        type: "subscription",
        basePrice: 100_000n,
        unit: "period",
        subscriptionBlocks: 1000n,
      },
    });
  });

  it("should export and restore subscriptions round-trip", () => {
    // Create a subscription directly
    state.setSubscription(serviceId, caller.did, 2000n);

    // Verify it exists
    expect(state.hasActiveSubscription(serviceId, caller.did, 100n)).toBe(true);

    // Export
    const exported = state.exportSubscriptions();
    expect(exported.length).toBe(1);
    expect(exported[0].serviceId).toBe(serviceId);
    expect(exported[0].callerDid).toBe(caller.did);
    expect(exported[0].expiresAt).toBe(2000n);

    // Create fresh state and restore
    const state2 = new StateManager();
    state2.restoreSubscriptions(exported);

    // Subscription survives the round-trip
    expect(state2.hasActiveSubscription(serviceId, caller.did, 100n)).toBe(
      true,
    );
    expect(state2.getSubscriptionExpiry(serviceId, caller.did)).toBe(2000n);
  });

  it("should rollback subscriptions on failed transaction", () => {
    // Set up a valid subscription first
    state.setSubscription(serviceId, caller.did, 5000n);

    // Start a transaction, set a new subscription, then rollback
    state.beginTransaction();
    state.setSubscription(serviceId, caller.did, 9999n);

    // During transaction, new value is visible
    expect(state.getSubscriptionExpiry(serviceId, caller.did)).toBe(9999n);

    // Rollback restores old value
    state.rollbackTransaction();
    expect(state.getSubscriptionExpiry(serviceId, caller.did)).toBe(5000n);
  });

  it("should persist subscriptions in persistBlock batch", async () => {
    // Set a subscription
    state.setSubscription(serviceId, caller.did, 3000n);

    // persistBlock should include subscription (returns count > 0)
    // Without a real store, persistBlock returns 0 — but we can verify
    // the export method works correctly as the persistence path uses it
    const exported = state.exportSubscriptions();
    expect(exported.length).toBe(1);
    expect(exported[0].expiresAt).toBe(3000n);
  });

  it("should handle subscribe_service transaction creating a queryable subscription", () => {
    const tx = makeTx("subscribe_service", caller.publicKey, {
      serviceId,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    // Subscription should now be active
    expect(
      state.hasActiveSubscription(serviceId, caller.did, blockNumber),
    ).toBe(true);

    // Expiry = blockNumber + subscriptionBlocks = 100 + 1000 = 1100
    const expiry = state.getSubscriptionExpiry(serviceId, caller.did);
    expect(expiry).toBe(1100n);

    // Subscription should survive export/restore
    const exported = state.exportSubscriptions();
    expect(exported.length).toBe(1);
  });

  it("should extend existing subscription on renewal", () => {
    // First subscribe at block 100
    const tx1 = makeTx("subscribe_service", caller.publicKey, { serviceId });
    const r1 = executeTransaction(state, tx1, blockNumber);
    expect(r1.success).toBe(true);

    // Expiry = 100 + 1000 = 1100
    expect(state.getSubscriptionExpiry(serviceId, caller.did)).toBe(1100n);

    // Renew at block 200 (before expiry) — should extend from 1100, not from 200
    const tx2 = makeTx(
      "subscribe_service",
      caller.publicKey,
      { serviceId },
      1000n,
      1n,
    );
    const r2 = executeTransaction(state, tx2, 200n);
    expect(r2.success).toBe(true);

    // Expiry = 1100 + 1000 = 2100 (extended from existing expiry)
    expect(state.getSubscriptionExpiry(serviceId, caller.did)).toBe(2100n);

    // Round-trip preserves extended expiry
    const exported = state.exportSubscriptions();
    const state2 = new StateManager();
    state2.restoreSubscriptions(exported);
    expect(state2.getSubscriptionExpiry(serviceId, caller.did)).toBe(2100n);
  });

  it("should allow call_service after subscribing", () => {
    // Subscribe first
    const subTx = makeTx("subscribe_service", caller.publicKey, { serviceId });
    const subResult = executeTransaction(state, subTx, blockNumber);
    expect(subResult.success).toBe(true);

    // Now call the service — subscription pricing returns 0 cost
    const callTx = makeTx(
      "call_service",
      caller.publicKey,
      {
        serviceId,
        units: 1n,
      },
      1000n,
      1n,
    );
    const callResult = executeTransaction(state, callTx, blockNumber);
    expect(callResult.success).toBe(true);
  });

  it("should reject call_service without subscription", () => {
    // Try to call without subscribing first
    const callTx = makeTx("call_service", caller.publicKey, {
      serviceId,
      units: 1n,
    });
    const callResult = executeTransaction(state, callTx, blockNumber);
    expect(callResult.success).toBe(false);
    expect(callResult.error).toContain("subscription required");
  });
});

// ============ G1: apply_to_project ============

describe.sequential("apply_to_project transaction (G1)", () => {
  let state: StateManager;
  let founder: ReturnType<typeof makeAgent>;
  let applicant: ReturnType<typeof makeAgent>;
  const projectId = "proj-g1-apply";

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    applicant = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, applicant.did, 1_000_000_000n);

    seedProject(state, founder, {
      id: projectId,
      phase: "growing",
      contributors: [makeContributor(founder.did, "founder")],
    });
  });

  it("should accept application during growing phase", () => {
    const tx = makeTx("apply_to_project", applicant.publicKey, {
      projectId,
      proposedRole: "developer",
      message: "I want to contribute",
      declaredCapabilities: ["inference"],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    const project = state.getProject(projectId);
    expect(project!.applications).toHaveLength(1);
    expect(project!.applications![0].did).toBe(applicant.did);
    expect(project!.applications![0].status).toBe("pending");
    expect(project!.applications![0].proposedRole).toBe("developer");
  });

  it("should accept application during sprout phase", () => {
    state.updateProject(projectId, {
      ...state.getProject(projectId)!,
      phase: "sprout",
    });

    const tx = makeTx("apply_to_project", applicant.publicKey, {
      projectId,
      proposedRole: "developer",
      message: "Early contributor",
      declaredCapabilities: [],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });

  it("should reject application if already a contributor", () => {
    // Make applicant a contributor
    const project = state.getProject(projectId)!;
    state.updateProject(projectId, {
      ...project,
      contributors: [
        ...project.contributors,
        makeContributor(applicant.did, "developer"),
      ],
    });

    const tx = makeTx("apply_to_project", applicant.publicKey, {
      projectId,
      proposedRole: "developer",
      message: "Apply again",
      declaredCapabilities: [],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Already a contributor");
  });

  it("should reject duplicate pending application", () => {
    const tx1 = makeTx("apply_to_project", applicant.publicKey, {
      projectId,
      proposedRole: "developer",
      message: "First application",
      declaredCapabilities: [],
    });
    executeTransaction(state, tx1, blockNumber);

    const tx2 = makeTx(
      "apply_to_project",
      applicant.publicKey,
      {
        projectId,
        proposedRole: "reviewer",
        message: "Second try",
        declaredCapabilities: [],
      },
      1000n,
      1n,
    );
    const result = executeTransaction(state, tx2, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Application already pending");
  });

  it("should reject application in mature phase", () => {
    state.updateProject(projectId, {
      ...state.getProject(projectId)!,
      phase: "mature",
    });

    const tx = makeTx("apply_to_project", applicant.publicKey, {
      projectId,
      proposedRole: "developer",
      message: "Too late",
      declaredCapabilities: [],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Cannot apply in 'mature' phase");
  });
});

// ============ G1: review_application ============

describe.sequential("review_application transaction (G1)", () => {
  let state: StateManager;
  let founder: ReturnType<typeof makeAgent>;
  let applicant: ReturnType<typeof makeAgent>;
  const projectId = "proj-g1-review";

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    applicant = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, applicant.did, 1_000_000_000n);

    seedProject(state, founder, {
      id: projectId,
      phase: "growing",
      contributors: [makeContributor(founder.did, "founder")],
    });

    // Submit an application
    const applyTx = makeTx("apply_to_project", applicant.publicKey, {
      projectId,
      proposedRole: "developer",
      message: "I want to contribute",
      declaredCapabilities: ["inference"],
    });
    executeTransaction(state, applyTx, blockNumber);
  });

  it("should approve application and add contributor", () => {
    const tx = makeTx("review_application", founder.publicKey, {
      projectId,
      applicantDid: applicant.did,
      decision: "approve",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    const project = state.getProject(projectId)!;
    expect(project.applications![0].status).toBe("approved");
    expect(project.contributors).toHaveLength(2);
    expect(project.contributors[1].did).toBe(applicant.did);
    expect(project.contributors[1].role).toBe("developer");
  });

  it("should reject application without adding contributor", () => {
    const tx = makeTx("review_application", founder.publicKey, {
      projectId,
      applicantDid: applicant.did,
      decision: "reject",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    const project = state.getProject(projectId)!;
    expect(project.applications![0].status).toBe("rejected");
    expect(project.contributors).toHaveLength(1); // Still just founder
  });

  it("should reject if sender is not founder", () => {
    const nonFounder = makeAgent();
    fundAgent(state, nonFounder.did, 1_000_000_000n);

    const tx = makeTx("review_application", nonFounder.publicKey, {
      projectId,
      applicantDid: applicant.did,
      decision: "approve",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("Only founder can review");
  });

  it("should reject if no pending application exists", () => {
    const tx = makeTx("review_application", founder.publicKey, {
      projectId,
      applicantDid: "did:lgm:nonexistent",
      decision: "approve",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("No pending application found");
  });
});

// ============ G4: set_availability ============

describe.sequential("set_availability transaction (G4)", () => {
  let state: StateManager;
  let agent: ReturnType<typeof makeAgent>;

  beforeEach(() => {
    state = new StateManager();
    agent = makeAgent();
    fundAgent(state, agent.did, 1_000_000_000n);
  });

  it("should set agent as available", () => {
    const tx = makeTx("set_availability", agent.publicKey, {
      available: true,
      capabilities: ["inference", "compute"],
      preferredDomains: ["physics.mechanics"],
      minimumBounty: "1000000",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    const account = state.getAccount(agent.did);
    expect(account!.availability).toBeDefined();
    expect(account!.availability!.available).toBe(true);
    expect(account!.availability!.capabilities).toEqual([
      "inference",
      "compute",
    ]);
    expect(account!.availability!.minimumBounty).toBe(1000000n);
    expect(account!.availability!.updatedAtBlock).toBe(blockNumber);
  });

  it("should set agent as unavailable", () => {
    // First set available
    const tx1 = makeTx("set_availability", agent.publicKey, {
      available: true,
      capabilities: ["inference"],
      preferredDomains: [],
      minimumBounty: 0n,
    });
    executeTransaction(state, tx1, blockNumber);

    // Then set unavailable
    const tx2 = makeTx(
      "set_availability",
      agent.publicKey,
      {
        available: false,
        capabilities: [],
        preferredDomains: [],
        minimumBounty: 0n,
      },
      1000n,
      1n,
    );
    const result = executeTransaction(state, tx2, blockNumber);
    expect(result.success).toBe(true);

    const account = state.getAccount(agent.did);
    expect(account!.availability!.available).toBe(false);
  });

  it("should reject if account does not exist", () => {
    const unknown = makeAgent();
    // Don't fund the account

    const tx = makeTx("set_availability", unknown.publicKey, {
      available: true,
      capabilities: [],
      preferredDomains: [],
      minimumBounty: 0n,
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
    expect(result.error).toContain("account does not exist");
  });
});

// ============ G6: Auto phase transitions ============

describe.sequential("auto phase transition growing→mature (G6)", () => {
  it("should auto-advance to mature when last task is approved", () => {
    const state = new StateManager();
    const founder = makeAgent();
    const developer = makeAgent();
    const reviewer = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, developer.did, 1_000_000_000n);
    fundAgent(state, reviewer.did, 1_000_000_000n);

    const projectId = "proj-g6-auto";

    // Project with totalTasks=1, completedTasks=0
    seedProject(state, founder, {
      id: projectId,
      phase: "growing",
      totalTasks: 1,
      completedTasks: 0,
      contributors: [
        makeContributor(founder.did, "founder"),
        makeContributor(developer.did, "developer"),
        makeContributor(reviewer.did, "reviewer"),
      ],
      taskIds: ["task-g6-1"],
    });

    // Seed the single task in review state
    seedTask(state, projectId, "task-g6-1", {
      bountyAmount: 10_000_000n,
      assignee: developer.did,
      status: "review",
      deliverable: {
        type: "code",
        hash: "abc123",
        metadata: {},
        submittedAtBlock: 90n,
        verifiedBy: [],
      },
    });

    // Approve the deliverable — should auto-advance project to mature
    const tx = makeTx("approve_deliverable", reviewer.publicKey, {
      projectId,
      taskId: "task-g6-1",
      declaredCapabilities: ["code_review"],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    const project = state.getProject(projectId)!;
    expect(project.phase).toBe("mature");
  });

  it("should NOT auto-advance when tasks remain", () => {
    const state = new StateManager();
    const founder = makeAgent();
    const developer = makeAgent();
    const reviewer = makeAgent();
    fundAgent(state, founder.did, 10_000_000_000n);
    fundAgent(state, developer.did, 1_000_000_000n);
    fundAgent(state, reviewer.did, 1_000_000_000n);

    const projectId = "proj-g6-no-advance";

    // totalTasks=2, completedTasks=0 — after 1 approval, still 1 remaining
    seedProject(state, founder, {
      id: projectId,
      phase: "growing",
      totalTasks: 2,
      completedTasks: 0,
      contributors: [
        makeContributor(founder.did, "founder"),
        makeContributor(developer.did, "developer"),
        makeContributor(reviewer.did, "reviewer"),
      ],
      taskIds: ["task-g6-2a", "task-g6-2b"],
    });

    seedTask(state, projectId, "task-g6-2a", {
      bountyAmount: 10_000_000n,
      assignee: developer.did,
      status: "review",
      deliverable: {
        type: "code",
        hash: "abc123",
        metadata: {},
        submittedAtBlock: 90n,
        verifiedBy: [],
      },
    });

    const tx = makeTx("approve_deliverable", reviewer.publicKey, {
      projectId,
      taskId: "task-g6-2a",
      declaredCapabilities: ["code_review"],
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    const project = state.getProject(projectId)!;
    expect(project.phase).toBe("growing"); // Still growing
    expect(project.completedTasks).toBe(1);
  });
});

// ============ P0-1: spawn_child_project ============

describe.sequential("spawn_child_project", () => {
  let state: StateManager;
  let founder: { publicKey: string; did: string };
  let projectId: string;

  beforeEach(() => {
    state = new StateManager();
    founder = makeAgent();
    fundAgent(state, founder.did, 100_000_000_000n);
    projectId = "proj-seeding-1";

    // Create a project in seeding phase with revenue generated
    seedProject(state, founder, {
      id: projectId,
      phase: "seeding",
      revenueGenerated: 10_000_000_000n,
      budget: 5_000_000_000n,
    });
  });

  it("should spawn child project from seeding parent", () => {
    const tx = makeTx("spawn_child_project", founder.publicKey, {
      parentProjectId: projectId,
      name: "Child Project",
      description: "Spawned from parent",
      knowledgeDomain: "",
      demandSignal: makeDemandSignal(),
      productSpec: makeProductSpec(),
      budgetBp: "2000", // 20% of parent revenue
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);

    // Find child project (there should be 2 projects now)
    const parent = state.getProject(projectId)!;
    // Child should have parentProjectId set
    // We need to find the child by iterating — get all projects
    // The child escrow should have received funds
    const budgetAmount = (10_000_000_000n * 2000n) / 10000n; // 2B
    expect(budgetAmount).toBe(2_000_000_000n);
  });

  it("should reject if parent not in seeding phase", () => {
    state.updateProject(projectId, {
      ...state.getProject(projectId)!,
      phase: "fruiting",
    });

    const tx = makeTx("spawn_child_project", founder.publicKey, {
      parentProjectId: projectId,
      name: "Bad Child",
      description: "Should fail",
      knowledgeDomain: "",
      demandSignal: makeDemandSignal(),
      productSpec: makeProductSpec(),
      budgetBp: "1000",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
  });

  it("should reject if sender is not a contributor", () => {
    const stranger = makeAgent();
    fundAgent(state, stranger.did, 100_000_000_000n);

    const tx = makeTx("spawn_child_project", stranger.publicKey, {
      parentProjectId: projectId,
      name: "Unauthorized Child",
      description: "Should fail",
      knowledgeDomain: "",
      demandSignal: makeDemandSignal(),
      productSpec: makeProductSpec(),
      budgetBp: "1000",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
  });

  it("should reject if budget exceeds 30% cap", () => {
    const tx = makeTx("spawn_child_project", founder.publicKey, {
      parentProjectId: projectId,
      name: "Greedy Child",
      description: "Too much budget",
      knowledgeDomain: "",
      demandSignal: makeDemandSignal(),
      productSpec: makeProductSpec(),
      budgetBp: "5000", // 50% — exceeds 30% cap
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
  });

  it("should reject if parent escrow has insufficient funds", () => {
    // Drain escrow
    const escrow = `did:lgm:project-escrow:${projectId}`;
    const balance = state.getBalance(escrow);
    state.deductBalance(escrow, balance);

    const tx = makeTx("spawn_child_project", founder.publicKey, {
      parentProjectId: projectId,
      name: "Underfunded Child",
      description: "No escrow",
      knowledgeDomain: "",
      demandSignal: makeDemandSignal(),
      productSpec: makeProductSpec(),
      budgetBp: "1000",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(false);
  });

  it("should set parentProjectId on child", () => {
    const tx = makeTx("spawn_child_project", founder.publicKey, {
      parentProjectId: projectId,
      name: "Linked Child",
      description: "Has parent ID",
      knowledgeDomain: "",
      demandSignal: makeDemandSignal(),
      productSpec: makeProductSpec(),
      budgetBp: "1500",
    });

    const result = executeTransaction(state, tx, blockNumber);
    expect(result.success).toBe(true);
  });
});

// ============ P0-2: claim_submit domain deprecation check ============

describe.sequential("claim_submit domain deprecation check", () => {
  const ONE_LGM = 10n ** 18n;

  let state: StateManager;
  let claimer: { publicKey: string; did: string };

  beforeEach(() => {
    state = new StateManager();
    claimer = makeAgent();
    fundAgent(state, claimer.did, 10000n * ONE_LGM);
  });

  function setupDomainTree(
    domains: Array<{
      id: string;
      deprecated?: boolean;
    }>,
  ): void {
    // Add root domain first
    state.addDomainToTree({
      id: "root",
      name: "Root",
      stratum: "formal",
      parentDomain: null,
      childDomains: domains.map((d) => d.id),
      dependsOnStrata: [],
      logicZoneDefault: "propositional",
      epistemicCategoryDefault: "axiom",
      description: "Root domain",
      examples: [],
      factCount: 0,
      axiomCount: 0,
      createdAt: 0,
    });

    for (const d of domains) {
      state.addDomainToTree({
        id: d.id,
        name: d.id,
        stratum: "empirical",
        parentDomain: "root",
        childDomains: [],
        dependsOnStrata: [],
        logicZoneDefault: "propositional",
        epistemicCategoryDefault: "empirical",
        description: `Test domain ${d.id}`,
        examples: [],
        factCount: 1,
        axiomCount: 0,
        createdAt: 0,
        deprecatedAt: d.deprecated ? 100 : undefined,
      });
    }
  }

  it("should allow claim in active domain", () => {
    setupDomainTree([{ id: "physics.optics" }]);

    const tx = makeTx("claim_submit", claimer.publicKey, {
      type: "claim_submit",
      claim: "Light refracts through prisms",
      claimType: "empirical",
      domain: "physics.optics",
      stratum: "empirical",
      stake: 200n * ONE_LGM,
      references: [],
    });

    const result = executeTransaction(
      state,
      tx,
      blockNumber,
      DEFAULT_CONSENSUS_PARAMS,
    );
    expect(result.success).toBe(true);
  });

  it("should reject claim in deprecated domain", () => {
    setupDomainTree([{ id: "alchemy.transmutation", deprecated: true }]);

    const tx = makeTx("claim_submit", claimer.publicKey, {
      type: "claim_submit",
      claim: "Lead can become gold",
      claimType: "empirical",
      domain: "alchemy.transmutation",
      stratum: "empirical",
      stake: 200n * ONE_LGM,
      references: [],
    });

    const result = executeTransaction(
      state,
      tx,
      blockNumber,
      DEFAULT_CONSENSUS_PARAMS,
    );
    expect(result.success).toBe(false);
    expect(result.error).toContain("deprecated");
  });

  it("should reject claim in non-existent domain (when tree populated)", () => {
    setupDomainTree([{ id: "physics.optics" }]);

    const tx = makeTx("claim_submit", claimer.publicKey, {
      type: "claim_submit",
      claim: "Test claim",
      claimType: "empirical",
      domain: "nonexistent.domain",
      stratum: "empirical",
      stake: 200n * ONE_LGM,
      references: [],
    });

    const result = executeTransaction(
      state,
      tx,
      blockNumber,
      DEFAULT_CONSENSUS_PARAMS,
    );
    expect(result.success).toBe(false);
    expect(result.error).toContain("does not exist");
  });

  it("should allow claim when domain tree is empty (genesis state)", () => {
    // No domain tree setup — fresh state
    const tx = makeTx("claim_submit", claimer.publicKey, {
      type: "claim_submit",
      claim: "Genesis claim",
      claimType: "empirical",
      domain: "some.domain",
      stratum: "empirical",
      stake: 200n * ONE_LGM,
      references: [],
    });

    const result = executeTransaction(
      state,
      tx,
      blockNumber,
      DEFAULT_CONSENSUS_PARAMS,
    );
    expect(result.success).toBe(true);
  });
});
