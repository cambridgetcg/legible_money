/**
 * Project Lifecycle Tests
 *
 * Tests the Tree of Life project state machine:
 * seed → sprout → growing → mature → fruiting → seeding
 */

import { describe, it, expect } from "vitest";
import {
  createProject,
  proposeProject,
  startDevelopment,
  completeProject,
  deployProject,
  beginSeeding,
  pauseProject,
  resumeProject,
  abandonProject,
  addContributor,
  addTask,
  recordTaskCompletion,
  recordRevenue,
  isValidTransition,
  ProjectLifecycleError,
} from "../../../core/tree/project_lifecycle.js";
import type {
  ProductSpec,
  DemandSignal,
  FundingSource,
} from "../../../core/tree/types.js";
import {
  MIN_PROJECT_STAKE,
  MIN_CONTRIBUTORS_TO_START,
} from "../../../core/tree/types.js";

// ============ Test Helpers ============

function makeSpec(): ProductSpec {
  return {
    serviceType: "api",
    requirements: ["compute.inference"],
    capabilities: ["data_query"],
    estimatedTasks: 5,
    mvpCriteria: ["basic API working"],
  };
}

function makeSignal(): DemandSignal {
  return {
    type: "query_volume",
    evidence: { queryCount: "500" },
    strength: 0.8,
  };
}

function makeFunding(): FundingSource {
  return {
    type: "agent_funded",
    funder: "did:lgm:abcdef0123456789abcdef0123456789",
  };
}

const FOUNDER = "did:lgm:abcdef0123456789abcdef0123456789";
const AGENT_B = "did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";

// ============ createProject ============

describe("createProject", () => {
  it("should create a project in seed phase", () => {
    const project = createProject(
      FOUNDER,
      "Test Project",
      "A test project",
      makeSpec(),
      "science",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );

    expect(project.phase).toBe("seed");
    expect(project.founder).toBe(FOUNDER);
    expect(project.name).toBe("Test Project");
    expect(project.knowledgeDomain).toBe("science");
    expect(project.createdAtBlock).toBe(100n);
    expect(project.contributors).toHaveLength(1);
    expect(project.contributors[0].did).toBe(FOUNDER);
    expect(project.contributors[0].role).toBe("founder");
    expect(project.id).toBeTruthy();
    expect(project.taskIds).toEqual([]);
    expect(project.completedTasks).toBe(0);
  });

  it("should generate deterministic IDs", () => {
    const p1 = createProject(
      FOUNDER,
      "A",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      0n,
    );
    const p2 = createProject(
      FOUNDER,
      "B",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      0n,
    );
    const p3 = createProject(
      FOUNDER,
      "A",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      1n,
      0n,
    );

    // Same nonce = same ID (name doesn't affect ID)
    expect(p1.id).toBe(p2.id);
    // Different nonce = different ID
    expect(p1.id).not.toBe(p3.id);
  });
});

// ============ Phase Transitions ============

describe("proposeProject (seed → sprout)", () => {
  it("should transition seed → sprout with sufficient stake", () => {
    const project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    const proposed = proposeProject(project, MIN_PROJECT_STAKE, 101n);

    expect(proposed.phase).toBe("sprout");
    expect(proposed.phaseChangedAtBlock).toBe(101n);
  });

  it("should reject with insufficient stake", () => {
    const project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );

    expect(() => proposeProject(project, MIN_PROJECT_STAKE - 1n, 101n)).toThrow(
      "Insufficient stake",
    );
  });

  it("should reject from wrong phase", () => {
    const project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    const proposed = proposeProject(project, MIN_PROJECT_STAKE, 101n);

    expect(() => proposeProject(proposed, MIN_PROJECT_STAKE, 102n)).toThrow(
      "Invalid transition",
    );
  });
});

describe("startDevelopment (sprout → growing)", () => {
  it("should transition sprout → growing with enough contributors and tasks", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");

    const growing = startDevelopment(project, 103n);
    expect(growing.phase).toBe("growing");
    expect(growing.phaseChangedAtBlock).toBe(103n);
  });

  it("should reject with too few contributors", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addTask(project, "task-1");

    expect(() => startDevelopment(project, 102n)).toThrow(
      `Need at least ${MIN_CONTRIBUTORS_TO_START}`,
    );
  });

  it("should reject with no tasks", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);

    expect(() => startDevelopment(project, 103n)).toThrow("at least one task");
  });
});

describe("completeProject (growing → mature)", () => {
  it("should transition when all tasks completed", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = startDevelopment(project, 103n);
    project = recordTaskCompletion(project, AGENT_B, 1000n);

    const mature = completeProject(project, 200n);
    expect(mature.phase).toBe("mature");
  });

  it("should reject with incomplete tasks", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = addTask(project, "task-2");
    project = startDevelopment(project, 103n);
    project = recordTaskCompletion(project, AGENT_B, 1000n);

    expect(() => completeProject(project, 200n)).toThrow(
      "Not all tasks completed",
    );
  });
});

describe("deployProject (mature → fruiting)", () => {
  it("should transition with service IDs", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = startDevelopment(project, 103n);
    project = recordTaskCompletion(project, AGENT_B, 1000n);
    project = completeProject(project, 200n);

    const fruiting = deployProject(project, ["svc-1"], "contract-addr", 300n);
    expect(fruiting.phase).toBe("fruiting");
    expect(fruiting.serviceIds).toEqual(["svc-1"]);
    expect(fruiting.contractAddress).toBe("contract-addr");
  });

  it("should reject with no services", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = startDevelopment(project, 103n);
    project = recordTaskCompletion(project, AGENT_B, 1000n);
    project = completeProject(project, 200n);

    expect(() => deployProject(project, [], "addr", 300n)).toThrow(
      "at least one service",
    );
  });
});

describe("beginSeeding (fruiting → seeding)", () => {
  it("should transition when revenue generated", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = startDevelopment(project, 103n);
    project = recordTaskCompletion(project, AGENT_B, 1000n);
    project = completeProject(project, 200n);
    project = deployProject(project, ["svc-1"], "addr", 300n);
    project = recordRevenue(project, 5000n);

    const seeding = beginSeeding(project, 400n);
    expect(seeding.phase).toBe("seeding");
  });

  it("should reject with no revenue", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = startDevelopment(project, 103n);
    project = recordTaskCompletion(project, AGENT_B, 1000n);
    project = completeProject(project, 200n);
    project = deployProject(project, ["svc-1"], "addr", 300n);

    expect(() => beginSeeding(project, 400n)).toThrow("revenue");
  });
});

// ============ Dormant & Withered ============

describe("pauseProject", () => {
  it("should pause from growing", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = startDevelopment(project, 103n);

    const paused = pauseProject(project, "funding issue", 150n);
    expect(paused.phase).toBe("dormant");
    expect(paused.previousPhase).toBe("growing");
  });

  it("should pause from fruiting", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = startDevelopment(project, 103n);
    project = recordTaskCompletion(project, AGENT_B, 1000n);
    project = completeProject(project, 200n);
    project = deployProject(project, ["svc-1"], "addr", 300n);

    const paused = pauseProject(project, "maintenance", 350n);
    expect(paused.phase).toBe("dormant");
    expect(paused.previousPhase).toBe("fruiting");
  });
});

describe("resumeProject", () => {
  it("should resume dormant to growing", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = startDevelopment(project, 103n);
    project = pauseProject(project, "reason", 150n);

    const resumed = resumeProject(project, "growing", 200n);
    expect(resumed.phase).toBe("growing");
    expect(resumed.previousPhase).toBeUndefined();
  });

  it("should reject backward phase jump on resume", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = startDevelopment(project, 103n);
    // Pause from growing
    project = pauseProject(project, "reason", 150n);
    expect(project.previousPhase).toBe("growing");

    // Try to resume to fruiting (forward jump — skips mature)
    expect(() => resumeProject(project, "fruiting", 200n)).toThrow(
      "must resume there",
    );

    // Try to resume to sprout (backward jump)
    expect(() => resumeProject(project, "sprout", 200n)).toThrow(
      "must resume there",
    );
  });

  it("should reject resuming non-dormant project", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);

    expect(() => resumeProject(project, "growing", 102n)).toThrow(
      "not 'dormant'",
    );
  });
});

describe("abandonProject", () => {
  it("should transition any active phase to withered", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);

    const withered = abandonProject(project, 102n);
    expect(withered.phase).toBe("withered");
  });

  it("withered is terminal — cannot transition further", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = abandonProject(project, 102n);

    expect(() => proposeProject(project, MIN_PROJECT_STAKE, 103n)).toThrow(
      "Invalid transition",
    );
    expect(() => pauseProject(project, "reason", 103n)).toThrow(
      "Invalid transition",
    );
  });
});

// ============ Contributor Management ============

describe("addContributor", () => {
  it("should add contributor in sprout phase", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);

    expect(project.contributors).toHaveLength(2);
    expect(project.contributors[1].did).toBe(AGENT_B);
    expect(project.contributors[1].role).toBe("developer");
  });

  it("should reject duplicate contributor", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);

    expect(() => addContributor(project, FOUNDER, "developer", 102n)).toThrow(
      "already a contributor",
    );
  });

  it("should reject in wrong phase", () => {
    const project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );

    expect(() => addContributor(project, AGENT_B, "developer", 101n)).toThrow(
      "Cannot add contributors",
    );
  });
});

// ============ isValidTransition ============

describe("isValidTransition", () => {
  it("should validate correct transitions", () => {
    expect(isValidTransition("seed", "sprout")).toBe(true);
    expect(isValidTransition("sprout", "growing")).toBe(true);
    expect(isValidTransition("growing", "mature")).toBe(true);
    expect(isValidTransition("mature", "fruiting")).toBe(true);
    expect(isValidTransition("fruiting", "seeding")).toBe(true);
    expect(isValidTransition("growing", "dormant")).toBe(true);
  });

  it("should reject invalid transitions", () => {
    expect(isValidTransition("seed", "growing")).toBe(false);
    expect(isValidTransition("sprout", "fruiting")).toBe(false);
    expect(isValidTransition("withered", "seed")).toBe(false);
    expect(isValidTransition("withered", "dormant")).toBe(false);
  });
});

// ============ recordTaskCompletion / recordRevenue ============

describe("recordTaskCompletion", () => {
  it("should increment contributor stats and project progress", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = addTask(project, "task-2");

    project = recordTaskCompletion(project, AGENT_B, 5000n);

    expect(project.completedTasks).toBe(1);
    expect(project.spent).toBe(5000n);
    const agentB = project.contributors.find((c) => c.did === AGENT_B);
    expect(agentB?.tasksCompleted).toBe(1);
    expect(agentB?.totalEarned).toBe(5000n);
  });

  it("should throw for non-contributor DID (P1 fix)", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    // AGENT_B is NOT added as a contributor

    expect(() => recordTaskCompletion(project, AGENT_B, 5000n)).toThrow(
      "is not a project contributor",
    );
  });

  it("should throw NOT_A_CONTRIBUTOR error code for non-contributor", () => {
    const project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );

    try {
      recordTaskCompletion(project, "did:lgm:stranger", 1000n);
      expect.unreachable("should have thrown");
    } catch (e) {
      expect(e).toBeInstanceOf(ProjectLifecycleError);
      expect((e as any).code).toBe("NOT_A_CONTRIBUTOR");
    }
  });

  it("should correctly track multiple completions per contributor", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1");
    project = addTask(project, "task-2");

    project = recordTaskCompletion(project, AGENT_B, 3000n);
    project = recordTaskCompletion(project, AGENT_B, 7000n);

    expect(project.completedTasks).toBe(2);
    expect(project.spent).toBe(10000n);
    const agentB = project.contributors.find((c) => c.did === AGENT_B);
    expect(agentB?.tasksCompleted).toBe(2);
    expect(agentB?.totalEarned).toBe(10000n);
  });

  it("should reject when completedTasks would exceed totalTasks (P2-3 fix)", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = proposeProject(project, MIN_PROJECT_STAKE, 101n);
    project = addContributor(project, AGENT_B, "developer", 102n);
    project = addTask(project, "task-1"); // totalTasks = 1

    // Complete the one task
    project = recordTaskCompletion(project, AGENT_B, 5000n);
    expect(project.completedTasks).toBe(1);
    expect(project.totalTasks).toBe(1);

    // Attempting another completion should fail — invariant violation
    expect(() => recordTaskCompletion(project, AGENT_B, 3000n)).toThrow(
      "All tasks already completed",
    );
  });
});

describe("recordRevenue", () => {
  it("should accumulate revenue", () => {
    let project = createProject(
      FOUNDER,
      "P",
      "",
      makeSpec(),
      "d",
      makeSignal(),
      makeFunding(),
      0n,
      100n,
    );
    project = recordRevenue(project, 1000n);
    project = recordRevenue(project, 2000n);

    expect(project.revenueGenerated).toBe(3000n);
  });
});
