/**
 * Task Manager Tests
 *
 * Tests for project task creation, assignment, submission, and review.
 */

import { describe, it, expect } from "vitest";
import {
  createTask,
  assignTask,
  startWork,
  submitDeliverable,
  approveDeliverable,
  rejectDeliverable,
  reopenTask,
  canAgentClaimTask,
  isTaskExpired,
} from "../../../core/tree/coordination/task_manager.js";
import type { TaskDeliverable } from "../../../core/tree/types.js";

const AGENT_A = "did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa";
const AGENT_B = "did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb";

function makeDeliverable(block: bigint = 200n): TaskDeliverable {
  return {
    type: "code",
    hash: "abc123",
    metadata: { files: 3 },
    submittedAtBlock: block,
    verifiedBy: [],
  };
}

// ============ createTask ============

describe("createTask", () => {
  it("should create a task in open status", () => {
    const task = createTask(
      "proj-1",
      "Implement API",
      "Build the REST API",
      "implement",
      ["development"],
      5000n,
      0n,
      100n,
    );

    expect(task.id).toBeTruthy();
    expect(task.projectId).toBe("proj-1");
    expect(task.title).toBe("Implement API");
    expect(task.taskType).toBe("implement");
    expect(task.status).toBe("open");
    expect(task.bountyAmount).toBe(5000n);
    expect(task.createdAtBlock).toBe(100n);
    expect(task.assignee).toBeUndefined();
  });

  it("should generate deterministic IDs", () => {
    const t1 = createTask("proj-1", "A", "", "test", [], 0n, 0n, 0n);
    const t2 = createTask("proj-1", "B", "", "test", [], 0n, 0n, 0n);
    const t3 = createTask("proj-1", "A", "", "test", [], 0n, 1n, 0n);

    expect(t1.id).toBe(t2.id); // Same nonce → same ID
    expect(t1.id).not.toBe(t3.id); // Different nonce → different ID
  });
});

// ============ Task Assignment Flow ============

describe("assignTask", () => {
  it("should assign agent to open task", () => {
    const task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    const assigned = assignTask(task, AGENT_A, 101n);

    expect(assigned.status).toBe("assigned");
    expect(assigned.assignee).toBe(AGENT_A);
  });

  it("should reject assigning to non-open task", () => {
    const task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    const assigned = assignTask(task, AGENT_A, 101n);

    expect(() => assignTask(assigned, AGENT_B, 102n)).toThrow("assigned");
  });

  it("should reject if deadline passed", () => {
    const task = createTask(
      "p",
      "T",
      "",
      "implement",
      [],
      1000n,
      0n,
      100n,
      150n,
    );

    expect(() => assignTask(task, AGENT_A, 200n)).toThrow("deadline");
  });
});

describe("startWork", () => {
  it("should transition assigned → in_progress", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);

    const working = startWork(task, AGENT_A);
    expect(working.status).toBe("in_progress");
  });

  it("should reject if not the assignee", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);

    expect(() => startWork(task, AGENT_B)).toThrow("Only the assignee");
  });
});

// ============ Deliverable Flow ============

describe("submitDeliverable", () => {
  it("should submit deliverable for review", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);
    task = startWork(task, AGENT_A);

    const submitted = submitDeliverable(task, AGENT_A, makeDeliverable(), 200n);
    expect(submitted.status).toBe("review");
    expect(submitted.deliverable).toBeTruthy();
    expect(submitted.deliverable?.hash).toBe("abc123");
  });

  it("should reject from non-assignee", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);

    expect(() =>
      submitDeliverable(task, AGENT_B, makeDeliverable(), 200n),
    ).toThrow("Only the assignee");
  });
});

describe("approveDeliverable", () => {
  it("should complete task on approval", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);
    task = submitDeliverable(task, AGENT_A, makeDeliverable(), 200n);

    const approved = approveDeliverable(task, AGENT_B, 201n);
    expect(approved.status).toBe("completed");
    expect(approved.completedAtBlock).toBe(201n);
    expect(approved.reviewers).toContain(AGENT_B);
    expect(approved.deliverable?.verifiedBy).toContain(AGENT_B);
  });

  it("should reject self-review", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);
    task = submitDeliverable(task, AGENT_A, makeDeliverable(), 200n);

    expect(() => approveDeliverable(task, AGENT_A, 201n)).toThrow(
      "cannot review their own",
    );
  });
});

describe("rejectDeliverable", () => {
  it("should reject and clear deliverable", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);
    task = submitDeliverable(task, AGENT_A, makeDeliverable(), 200n);

    const rejected = rejectDeliverable(task, AGENT_B, "needs more tests");
    expect(rejected.status).toBe("rejected");
    expect(rejected.deliverable).toBeUndefined();
    expect(rejected.reviewers).toContain(AGENT_B);
  });

  it("should reject self-review", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);
    task = submitDeliverable(task, AGENT_A, makeDeliverable(), 200n);

    expect(() => rejectDeliverable(task, AGENT_A, "reason")).toThrow(
      "cannot review their own",
    );
  });
});

describe("reopenTask", () => {
  it("should reopen rejected task", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);
    task = submitDeliverable(task, AGENT_A, makeDeliverable(), 200n);
    task = rejectDeliverable(task, AGENT_B, "reason");

    const reopened = reopenTask(task);
    expect(reopened.status).toBe("in_progress");
  });
});

// ============ Queries ============

describe("canAgentClaimTask", () => {
  it("should return true when agent has all required capabilities", () => {
    const task = createTask(
      "p",
      "T",
      "",
      "implement",
      ["development", "testing"],
      1000n,
      0n,
      100n,
    );
    expect(canAgentClaimTask(task, ["development", "testing", "review"])).toBe(
      true,
    );
  });

  it("should return false when missing capabilities", () => {
    const task = createTask(
      "p",
      "T",
      "",
      "implement",
      ["development", "testing"],
      1000n,
      0n,
      100n,
    );
    expect(canAgentClaimTask(task, ["development"])).toBe(false);
  });

  it("should return false for non-open tasks", () => {
    let task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    task = assignTask(task, AGENT_A, 101n);
    expect(canAgentClaimTask(task, [])).toBe(false);
  });

  it("should return true when no capabilities required", () => {
    const task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    expect(canAgentClaimTask(task, [])).toBe(true);
  });
});

describe("isTaskExpired", () => {
  it("should return false before deadline", () => {
    const task = createTask(
      "p",
      "T",
      "",
      "implement",
      [],
      1000n,
      0n,
      100n,
      200n,
    );
    expect(isTaskExpired(task, 150n)).toBe(false);
  });

  it("should return true at deadline", () => {
    const task = createTask(
      "p",
      "T",
      "",
      "implement",
      [],
      1000n,
      0n,
      100n,
      200n,
    );
    expect(isTaskExpired(task, 200n)).toBe(true);
  });

  it("should return false when no deadline", () => {
    const task = createTask("p", "T", "", "implement", [], 1000n, 0n, 100n);
    expect(isTaskExpired(task, 999999n)).toBe(false);
  });
});
