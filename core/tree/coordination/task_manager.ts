/**
 * Task Manager
 *
 * Manages task creation, assignment, submission, and approval within
 * Tree of Life projects. Each task maps to a bounty for incentive alignment.
 */

import { domainHash } from "../../crypto/hashing.js";
import type { DID } from "../../agents/identity/did.js";
import type {
  ProjectTask,
  TaskType,
  TaskStatus,
  TaskDeliverable,
} from "../types.js";

// ============ Task Creation ============

/**
 * Create a new task within a project.
 */
export function createTask(
  projectId: string,
  title: string,
  description: string,
  taskType: TaskType,
  requiredCapabilities: string[],
  bountyAmount: bigint,
  nonce: bigint,
  block: bigint,
  deadlineBlock?: bigint,
): ProjectTask {
  const id = generateTaskId(projectId, nonce);

  return {
    id,
    projectId,
    title,
    description,
    taskType,
    requiredCapabilities,
    reviewers: [],
    status: "open",
    createdAtBlock: block,
    deadlineBlock,
    bountyAmount,
  };
}

// ============ Task Assignment ============

/**
 * Assign an agent to a task.
 */
export function assignTask(
  task: ProjectTask,
  agentDid: DID,
  block: bigint,
): ProjectTask {
  assertStatus(task, ["open"]);

  if (task.deadlineBlock !== undefined && block >= task.deadlineBlock) {
    throw new TaskError("Task deadline has passed", "DEADLINE_PASSED");
  }

  return {
    ...task,
    assignee: agentDid,
    status: "assigned",
  };
}

/**
 * Mark task as in progress (assignee starts working).
 */
export function startWork(task: ProjectTask, agentDid: DID): ProjectTask {
  assertStatus(task, ["assigned"]);

  if (task.assignee !== agentDid) {
    throw new TaskError("Only the assignee can start work", "NOT_ASSIGNEE");
  }

  return {
    ...task,
    status: "in_progress",
  };
}

// ============ Deliverable Submission ============

/**
 * Submit a deliverable for review.
 */
export function submitDeliverable(
  task: ProjectTask,
  agentDid: DID,
  deliverable: TaskDeliverable,
  block: bigint,
): ProjectTask {
  assertStatus(task, ["assigned", "in_progress"]);

  if (task.assignee !== agentDid) {
    throw new TaskError(
      "Only the assignee can submit deliverables",
      "NOT_ASSIGNEE",
    );
  }

  return {
    ...task,
    status: "review",
    deliverable: {
      ...deliverable,
      submittedAtBlock: block,
    },
  };
}

// ============ Review ============

/**
 * Approve a deliverable. Task is completed and bounty should be paid.
 */
export function approveDeliverable(
  task: ProjectTask,
  reviewerDid: DID,
  block: bigint,
): ProjectTask {
  assertStatus(task, ["review"]);

  if (!task.deliverable) {
    throw new TaskError("No deliverable to approve", "NO_DELIVERABLE");
  }

  if (task.assignee === reviewerDid) {
    throw new TaskError(
      "Assignee cannot review their own deliverable",
      "SELF_REVIEW",
    );
  }

  return {
    ...task,
    status: "completed",
    completedAtBlock: block,
    reviewers: [...task.reviewers, reviewerDid],
    deliverable: {
      ...task.deliverable,
      verifiedBy: [...task.deliverable.verifiedBy, reviewerDid],
    },
  };
}

/**
 * Reject a deliverable. Task returns to in_progress.
 */
export function rejectDeliverable(
  task: ProjectTask,
  reviewerDid: DID,
  _reason: string,
): ProjectTask {
  assertStatus(task, ["review"]);

  if (!task.deliverable) {
    throw new TaskError("No deliverable to reject", "NO_DELIVERABLE");
  }

  if (task.assignee === reviewerDid) {
    throw new TaskError(
      "Assignee cannot review their own deliverable",
      "SELF_REVIEW",
    );
  }

  return {
    ...task,
    status: "rejected",
    reviewers: [...task.reviewers, reviewerDid],
    deliverable: undefined, // Clear rejected deliverable
  };
}

/**
 * Reopen a rejected task for another attempt.
 */
export function reopenTask(task: ProjectTask): ProjectTask {
  assertStatus(task, ["rejected"]);

  return {
    ...task,
    status: "in_progress",
  };
}

// ============ Task Queries ============

/**
 * Check if a task can be assigned to an agent based on capabilities.
 */
export function canAgentClaimTask(
  task: ProjectTask,
  agentCapabilities: string[],
): boolean {
  if (task.status !== "open") return false;

  // All required capabilities must be present
  return task.requiredCapabilities.every((cap) =>
    agentCapabilities.includes(cap),
  );
}

/**
 * Check if a task is past its deadline.
 */
export function isTaskExpired(
  task: ProjectTask,
  currentBlock: bigint,
): boolean {
  if (task.deadlineBlock === undefined) return false;
  return currentBlock >= task.deadlineBlock;
}

// ============ Helpers ============

/**
 * Generate deterministic task ID
 */
export function generateTaskId(projectId: string, nonce: bigint): string {
  const input = `${projectId}:task:${nonce}`;
  return Buffer.from(
    domainHash("LGM.tree.task.id.v1", input).slice(0, 16),
  ).toString("hex");
}

function assertStatus(task: ProjectTask, allowed: TaskStatus[]): void {
  if (!allowed.includes(task.status)) {
    throw new TaskError(
      `Task is '${task.status}', expected one of: ${allowed.join(", ")}`,
      "INVALID_STATUS",
    );
  }
}

// ============ Error Types ============

export type TaskErrorCode =
  | "INVALID_STATUS"
  | "NOT_ASSIGNEE"
  | "NO_DELIVERABLE"
  | "SELF_REVIEW"
  | "DEADLINE_PASSED";

export class TaskError extends Error {
  constructor(
    message: string,
    public code: TaskErrorCode,
  ) {
    super(message);
    this.name = "TaskError";
  }
}
