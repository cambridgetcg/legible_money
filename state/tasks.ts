/**
 * Task State Tree
 *
 * Sparse Merkle Tree for Tree of Life project tasks.
 * Key: hash(taskId) - 32 bytes
 * Value: serialized ProjectTask
 *
 * Provides:
 * - Cryptographic proof of task existence
 * - Task lookup by project, assignee, and status
 * - On-chain task lifecycle tracking
 */

import {
  InMemorySMT,
  type MerkleRoot,
  type MerkleProofData,
} from "../core/crypto/merkle.js";
import { sha512_256 } from "../core/crypto/hashing.js";
import type { ProjectTask, TaskStatus, TaskType } from "../core/tree/types.js";
import { StateError } from "./types.js";

// ============ Serialization ============

/**
 * Serialize ProjectTask to bytes for SMT storage.
 * BigInt fields are converted to strings for JSON compatibility.
 */
function serializeTask(task: ProjectTask): Uint8Array {
  const json = JSON.stringify({
    id: task.id,
    projectId: task.projectId,
    title: task.title,
    description: task.description,
    taskType: task.taskType,
    requiredCapabilities: task.requiredCapabilities,
    assignee: task.assignee,
    reviewers: task.reviewers,
    status: task.status,
    createdAtBlock: task.createdAtBlock.toString(),
    deadlineBlock: task.deadlineBlock?.toString(),
    completedAtBlock: task.completedAtBlock?.toString(),
    deliverable: task.deliverable
      ? {
          type: task.deliverable.type,
          hash: task.deliverable.hash,
          metadata: task.deliverable.metadata,
          submittedAtBlock: task.deliverable.submittedAtBlock.toString(),
          verifiedBy: task.deliverable.verifiedBy,
        }
      : undefined,
    bountyAmount: task.bountyAmount.toString(),
    bountyId: task.bountyId,
    rejectionCount: task.rejectionCount,
  });
  return new TextEncoder().encode(json);
}

/**
 * Deserialize ProjectTask from bytes.
 */
function deserializeTask(bytes: Uint8Array): ProjectTask {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    id: obj.id,
    projectId: obj.projectId,
    title: obj.title,
    description: obj.description,
    taskType: obj.taskType as TaskType,
    requiredCapabilities: obj.requiredCapabilities,
    assignee: obj.assignee,
    reviewers: obj.reviewers,
    status: obj.status as TaskStatus,
    createdAtBlock: BigInt(obj.createdAtBlock),
    deadlineBlock: obj.deadlineBlock ? BigInt(obj.deadlineBlock) : undefined,
    completedAtBlock: obj.completedAtBlock
      ? BigInt(obj.completedAtBlock)
      : undefined,
    deliverable: obj.deliverable
      ? {
          type: obj.deliverable.type,
          hash: obj.deliverable.hash,
          metadata: obj.deliverable.metadata,
          submittedAtBlock: BigInt(obj.deliverable.submittedAtBlock),
          verifiedBy: obj.deliverable.verifiedBy,
        }
      : undefined,
    bountyAmount: BigInt(obj.bountyAmount),
    bountyId: obj.bountyId,
    rejectionCount: obj.rejectionCount,
  };
}

/**
 * Convert task ID to SMT key
 */
function taskIdToKey(taskId: string): Uint8Array {
  return sha512_256(taskId);
}

// ============ Task State Tree ============

export class TaskStateTree {
  private smt: InMemorySMT;
  private cache: Map<string, ProjectTask> = new Map();
  private taskCount: number = 0;

  constructor() {
    this.smt = new InMemorySMT();
  }

  /**
   * Get the current root hash
   */
  getRoot(): MerkleRoot {
    return this.smt.getRoot();
  }

  // ============ CRUD Operations ============

  /**
   * Get task by ID.
   * WARNING: Returns a cached reference. Do NOT mutate the returned object directly.
   * Use updateTask() for all modifications.
   */
  getTask(taskId: string): ProjectTask | null {
    const cached = this.cache.get(taskId);
    if (cached) return cached;

    const key = taskIdToKey(taskId);
    const value = this.smt.get(key);
    if (!value) return null;

    const task = deserializeTask(value);
    this.cache.set(taskId, task);
    return task;
  }

  /**
   * Create a new task
   */
  createTask(task: ProjectTask): MerkleRoot {
    if (this.getTask(task.id) !== null) {
      throw new StateError("Task already exists", "INVALID_TRANSITION", {
        taskId: task.id,
      });
    }

    const key = taskIdToKey(task.id);
    const value = serializeTask(task);
    const newRoot = this.smt.set(key, value);

    this.cache.set(task.id, task);
    this.taskCount++;

    return newRoot;
  }

  /**
   * Update an existing task
   */
  updateTask(taskId: string, updates: Partial<ProjectTask>): MerkleRoot {
    const existing = this.getTask(taskId);
    if (!existing) {
      throw new StateError("Task not found", "TASK_NOT_FOUND", { taskId });
    }

    const updated: ProjectTask = {
      ...existing,
      ...updates,
    };

    const key = taskIdToKey(taskId);
    const value = serializeTask(updated);
    const newRoot = this.smt.set(key, value);

    this.cache.set(taskId, updated);
    return newRoot;
  }

  // ============ Query Operations ============

  /**
   * Get all tasks for a project
   */
  getTasksByProject(projectId: string): ProjectTask[] {
    const results: ProjectTask[] = [];
    for (const task of this.cache.values()) {
      if (task.projectId === projectId) {
        results.push(task);
      }
    }
    return results;
  }

  /**
   * Get all tasks assigned to an agent
   */
  getTasksByAssignee(agentDid: string): ProjectTask[] {
    const results: ProjectTask[] = [];
    for (const task of this.cache.values()) {
      if (task.assignee === agentDid) {
        results.push(task);
      }
    }
    return results;
  }

  /**
   * Get all tasks with a given status
   */
  getTasksByStatus(status: TaskStatus): ProjectTask[] {
    const results: ProjectTask[] = [];
    for (const task of this.cache.values()) {
      if (task.status === status) {
        results.push(task);
      }
    }
    return results;
  }

  /**
   * Get total task count
   */
  getTaskCount(): number {
    return this.taskCount;
  }

  // ============ Proofs ============

  /**
   * Generate Merkle proof for a task.
   * Proves inclusion (or non-inclusion) in the task state tree.
   */
  proveTask(taskId: string): MerkleProofData {
    const key = taskIdToKey(taskId);
    return this.smt.prove(key);
  }

  // ============ Persistence Export/Restore ============

  /**
   * Export all tasks for persistence
   */
  exportTasks(): ProjectTask[] {
    return Array.from(this.cache.values());
  }

  // ============ Utilities ============

  /**
   * Clone for speculative execution
   */
  clone(): TaskStateTree {
    const cloned = new TaskStateTree();

    for (const [taskId, task] of this.cache) {
      const clonedTask: ProjectTask = {
        ...task,
        requiredCapabilities: [...task.requiredCapabilities],
        reviewers: [...task.reviewers],
        deliverable: task.deliverable
          ? {
              ...task.deliverable,
              metadata: { ...task.deliverable.metadata },
              verifiedBy: [...task.deliverable.verifiedBy],
            }
          : undefined,
      };

      cloned.cache.set(taskId, clonedTask);
      const key = taskIdToKey(taskId);
      cloned.smt.set(key, serializeTask(clonedTask));
    }

    cloned.taskCount = this.taskCount;
    return cloned;
  }
}
