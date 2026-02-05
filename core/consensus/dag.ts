/**
 * Execution DAG - Directed Acyclic Graph for parallel task execution
 *
 * Why DAG instead of linear blockchain?
 * - Agents doing independent work shouldn't wait for sequential blocks
 * - Parallel execution matches how agents actually operate
 * - Higher throughput for independent tasks
 *
 * Philosophy: The structure should match the problem.
 * Independent tasks are independent. Let them run in parallel.
 */

import { sha512 } from "@noble/hashes/sha512";
import {
  DAGNode,
  DAGNodeType,
  Task,
  TaskState,
  ValidationResult,
  // WorkProof,
} from "./types.js";
import { bytesToHex } from "../crypto/keys.js";

// ============================================================================
// DAG NODE OPERATIONS
// ============================================================================

/**
 * Create a unique node ID from its contents
 * This ensures content-addressability: same content = same ID
 */
export function createNodeId(
  node: Omit<DAGNode, "nodeId" | "confirmations" | "isConfirmed" | "isFinal">,
): string {
  const encoder = new TextEncoder();
  const content = encoder.encode(
    JSON.stringify({
      nodeType: node.nodeType,
      parents: node.parents.sort(), // Sorted for determinism
      payload: node.payload,
      submitter: node.submitter,
      logicalTime: node.logicalTime.toString(),
    }),
  );

  const hash = sha512(content);
  return bytesToHex(hash.slice(0, 32));
}

/**
 * Validate that a node is well-formed
 */
export function validateNode(
  node: DAGNode,
  getNode: (id: string) => DAGNode | null,
): {
  valid: boolean;
  errors: string[];
} {
  const errors: string[] = [];

  // 1. Check node ID matches content
  const expectedId = createNodeId(node);
  if (node.nodeId !== expectedId) {
    errors.push(`Node ID mismatch: expected ${expectedId}, got ${node.nodeId}`);
  }

  // 2. Check all parents exist
  for (const parentId of node.parents) {
    const parent = getNode(parentId);
    if (!parent) {
      errors.push(`Parent node ${parentId} does not exist`);
    } else {
      // 3. Check logical time ordering
      if (parent.logicalTime >= node.logicalTime) {
        errors.push(
          `Logical time violation: parent ${parentId} has time ${parent.logicalTime}, ` +
            `but node has time ${node.logicalTime}`,
        );
      }
    }
  }

  // 4. Check signature (placeholder - would verify against submitter's public key)
  if (!node.signature || node.signature.length === 0) {
    errors.push("Missing signature");
  }

  // 5. Check work proof exists
  if (!node.workProof) {
    errors.push("Missing work proof");
  }

  return {
    valid: errors.length === 0,
    errors,
  };
}

// ============================================================================
// DAG STATE MANAGEMENT
// ============================================================================

/**
 * The Execution DAG - stores all nodes and manages consensus
 */
export class ExecutionDAG {
  private nodes: Map<string, DAGNode> = new Map();
  private tips: Set<string> = new Set(); // Nodes with no children
  private byTask: Map<string, string[]> = new Map(); // taskId -> nodeIds
  private bySubmitter: Map<string, string[]> = new Map(); // agentId -> nodeIds

  // Confirmation tracking
  private confirmations: Map<string, Set<string>> = new Map(); // nodeId -> validator set

  constructor(
    private readonly confirmationThreshold: number = 0.67,
    private readonly finalityBlocks: number = 4,
  ) {}

  /**
   * Add a new node to the DAG
   */
  addNode(node: DAGNode): { success: boolean; error?: string } {
    // Validate the node
    const validation = validateNode(node, (id) => this.nodes.get(id) ?? null);
    if (!validation.valid) {
      return { success: false, error: validation.errors.join("; ") };
    }

    // Check for duplicate
    if (this.nodes.has(node.nodeId)) {
      return { success: false, error: "Node already exists" };
    }

    // Add node
    this.nodes.set(node.nodeId, node);

    // Update tips (this node is a new tip, parents are no longer tips)
    this.tips.add(node.nodeId);
    for (const parentId of node.parents) {
      this.tips.delete(parentId);
    }

    // Index by task
    if ("taskId" in node.payload || (node.payload as any).task?.taskId) {
      const taskId =
        (node.payload as any).taskId ?? (node.payload as any).task.taskId;
      const existing = this.byTask.get(taskId) ?? [];
      existing.push(node.nodeId);
      this.byTask.set(taskId, existing);
    }

    // Index by submitter
    const submitterNodes = this.bySubmitter.get(node.submitter) ?? [];
    submitterNodes.push(node.nodeId);
    this.bySubmitter.set(node.submitter, submitterNodes);

    return { success: true };
  }

  /**
   * Get a node by ID
   */
  getNode(nodeId: string): DAGNode | null {
    return this.nodes.get(nodeId) ?? null;
  }

  /**
   * Get current tips (nodes with no children)
   */
  getTips(): string[] {
    return Array.from(this.tips);
  }

  /**
   * Get all nodes for a task
   */
  getTaskNodes(taskId: string): DAGNode[] {
    const nodeIds = this.byTask.get(taskId) ?? [];
    return nodeIds.map((id) => this.nodes.get(id)!).filter(Boolean);
  }

  /**
   * Get task state by aggregating all nodes
   */
  getTaskState(taskId: string): Task | null {
    const nodes = this.getTaskNodes(taskId);
    if (nodes.length === 0) return null;

    // Find the task creation node
    const creationNode = nodes.find(
      (n) => n.nodeType === DAGNodeType.TASK_CREATED,
    );
    if (!creationNode || creationNode.payload.type !== "task_created") {
      return null;
    }

    const task = { ...creationNode.payload.task };

    // Apply subsequent state changes
    for (const node of nodes) {
      switch (node.nodeType) {
        case DAGNodeType.TASK_ACCEPTED:
          task.state = TaskState.ACCEPTED;
          task.acceptedAtBlock = node.logicalTime;
          break;

        case DAGNodeType.TASK_EXECUTED:
          if (node.payload.type === "task_executed") {
            task.state = TaskState.EXECUTED;
            task.outputHash = node.payload.outputHash;
            task.outputURI = node.payload.outputURI;
          }
          break;

        case DAGNodeType.VALIDATION:
          task.state = TaskState.VALIDATING;
          break;

        case DAGNodeType.SETTLEMENT:
          if (node.payload.type === "settlement") {
            task.state = TaskState.COMPLETED;
            task.verifiedAtBlock = node.logicalTime;
          }
          break;

        case DAGNodeType.DISPUTE:
          task.state = TaskState.DISPUTED;
          break;
      }
    }

    return task;
  }

  // ===========================================================================
  // CONFIRMATION & FINALITY
  // ===========================================================================

  /**
   * Record a validator's confirmation of a node
   */
  confirmNode(nodeId: string, validatorId: string): void {
    if (!this.confirmations.has(nodeId)) {
      this.confirmations.set(nodeId, new Set());
    }
    this.confirmations.get(nodeId)!.add(validatorId);

    // Check if threshold reached
    const node = this.nodes.get(nodeId);
    if (node) {
      node.confirmations = this.confirmations.get(nodeId)!.size;
      // Note: threshold check would use total validator count
    }
  }

  /**
   * Check if a node is confirmed (>= threshold of validators)
   */
  isConfirmed(nodeId: string, totalValidators: number): boolean {
    const confirmationCount = this.confirmations.get(nodeId)?.size ?? 0;
    return confirmationCount / totalValidators >= this.confirmationThreshold;
  }

  /**
   * Check if a node is final (confirmed + finality blocks passed)
   */
  isFinal(
    nodeId: string,
    currentBlock: bigint,
    totalValidators: number,
  ): boolean {
    const node = this.nodes.get(nodeId);
    if (!node) return false;

    if (!this.isConfirmed(nodeId, totalValidators)) {
      return false;
    }

    // Check if enough blocks have passed since confirmation
    // (In practice, would track confirmation block, not node's logical time)
    return currentBlock - node.logicalTime >= BigInt(this.finalityBlocks);
  }

  // ===========================================================================
  // PARALLEL EXECUTION SUPPORT
  // ===========================================================================

  /**
   * Find all nodes that can be executed in parallel
   * (their dependencies are all satisfied)
   */
  getExecutableNodes(satisfiedDeps: Set<string>): DAGNode[] {
    const executable: DAGNode[] = [];

    for (const node of this.nodes.values()) {
      // Skip already confirmed
      if (node.isConfirmed) continue;

      // Check if all parents are satisfied
      const allParentsSatisfied = node.parents.every((p) =>
        satisfiedDeps.has(p),
      );

      if (allParentsSatisfied) {
        executable.push(node);
      }
    }

    return executable;
  }

  /**
   * Get the "frontier" - the set of nodes needed to understand current state
   * This is what a new validator would need to sync
   */
  getFrontier(): DAGNode[] {
    // Get all tips and their ancestors up to last final node
    const frontier: DAGNode[] = [];
    const visited = new Set<string>();

    const visit = (nodeId: string) => {
      if (visited.has(nodeId)) return;
      visited.add(nodeId);

      const node = this.nodes.get(nodeId);
      if (!node) return;

      frontier.push(node);

      // Only traverse unfinal ancestors
      if (!node.isFinal) {
        for (const parentId of node.parents) {
          visit(parentId);
        }
      }
    };

    for (const tipId of this.tips) {
      visit(tipId);
    }

    return frontier;
  }

  // ===========================================================================
  // METRICS
  // ===========================================================================

  /**
   * Get DAG statistics
   */
  getStats(): {
    totalNodes: number;
    tipCount: number;
    confirmedCount: number;
    finalCount: number;
    taskCount: number;
  } {
    let confirmedCount = 0;
    let finalCount = 0;

    for (const node of this.nodes.values()) {
      if (node.isConfirmed) confirmedCount++;
      if (node.isFinal) finalCount++;
    }

    return {
      totalNodes: this.nodes.size,
      tipCount: this.tips.size,
      confirmedCount,
      finalCount,
      taskCount: this.byTask.size,
    };
  }
}

// ============================================================================
// NODE CREATION HELPERS
// ============================================================================

/**
 * Create a task creation node
 */
export function createTaskNode(
  task: Task,
  submitter: string,
  parents: string[],
  logicalTime: bigint,
  signature: Uint8Array,
): DAGNode {
  const node: Omit<
    DAGNode,
    "nodeId" | "confirmations" | "isConfirmed" | "isFinal"
  > = {
    nodeType: DAGNodeType.TASK_CREATED,
    parents,
    payload: { type: "task_created", task },
    workProof: {
      proofType: "execution",
      data: new Uint8Array(),
      metadata: { escrowTx: task.payment.escrowId },
    },
    submitter,
    signature,
    logicalTime,
  };

  return {
    ...node,
    nodeId: createNodeId(node),
    confirmations: 0,
    isConfirmed: false,
    isFinal: false,
  };
}

/**
 * Create a validation node
 */
export function createValidationNode(
  validation: ValidationResult,
  submitter: string,
  parents: string[],
  logicalTime: bigint,
  signature: Uint8Array,
): DAGNode {
  const node: Omit<
    DAGNode,
    "nodeId" | "confirmations" | "isConfirmed" | "isFinal"
  > = {
    nodeType: DAGNodeType.VALIDATION,
    parents,
    payload: {
      type: "validation",
      taskId: validation.taskId,
      validator: validation.validator,
      score: validation.score,
      passed: validation.passed,
      evidenceType: validation.evidenceType,
      evidenceURI: validation.evidenceURI,
      evidenceHash: validation.evidenceHash,
      commitment: validation.commitmentHash,
      reveal: validation.reveal,
    },
    workProof: {
      proofType: "validation",
      data: new Uint8Array(), // Would contain actual proof
      metadata: { validationType: validation.evidenceType },
    },
    submitter,
    signature,
    logicalTime,
  };

  return {
    ...node,
    nodeId: createNodeId(node),
    confirmations: 0,
    isConfirmed: false,
    isFinal: false,
  };
}
