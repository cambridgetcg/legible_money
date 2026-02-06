/**
 * Project Lifecycle State Machine
 *
 * Manages the lifecycle of product projects in the Tree of Life.
 *
 * State transitions:
 *   seed → sprout → growing → mature → fruiting → seeding
 *                     ↓                    ↓
 *                  dormant              dormant
 *                     ↓                    ↓
 *                  withered             withered
 */

import { domainHash } from "../crypto/hashing.js";
import type { DID } from "../agents/identity/did.js";
import {
  type ProductProject,
  type ProductSpec,
  type DemandSignal,
  type FundingSource,
  type ProjectPhase,
  type ContributorRecord,
  VALID_PHASE_TRANSITIONS,
  MIN_PROJECT_STAKE,
  MIN_CONTRIBUTORS_TO_START,
} from "./types.js";

// ============ Project Creation ============

/**
 * Create a new project from a detected opportunity.
 * Starts in 'seed' phase.
 */
export function createProject(
  founder: DID,
  name: string,
  description: string,
  spec: ProductSpec,
  domain: string,
  signal: DemandSignal,
  fundingSource: FundingSource,
  nonce: bigint,
  block: bigint,
): ProductProject {
  const id = generateProjectId(founder, nonce);

  return {
    id,
    name,
    description,
    phase: "seed",
    createdAtBlock: block,
    phaseChangedAtBlock: block,
    knowledgeDomain: domain,
    demandSignal: signal,
    founder,
    contributors: [
      {
        did: founder,
        role: "founder",
        joinedAtBlock: block,
        tasksCompleted: 0,
        totalEarned: 0n,
      },
    ],
    productSpec: spec,
    taskIds: [],
    completedTasks: 0,
    totalTasks: 0,
    fundingSource,
    budget:
      fundingSource.type === "research_fund"
        ? fundingSource.amount
        : fundingSource.type === "agent_funded"
          ? 0n
          : 0n,
    spent: 0n,
    revenueGenerated: 0n,
    serviceIds: [],
  };
}

// ============ Phase Transitions ============

/**
 * Propose a project (seed → sprout).
 * Founder must have minimum stake committed.
 */
export function proposeProject(
  project: ProductProject,
  founderBalance: bigint,
  block: bigint,
): ProductProject {
  assertTransition(project, "sprout");

  if (founderBalance < MIN_PROJECT_STAKE) {
    throw new ProjectLifecycleError(
      `Insufficient stake: need ${MIN_PROJECT_STAKE}, have ${founderBalance}`,
      "INSUFFICIENT_STAKE",
    );
  }

  return {
    ...project,
    phase: "sprout",
    phaseChangedAtBlock: block,
  };
}

/**
 * Start development (sprout → growing).
 * Requires minimum number of contributors.
 */
export function startDevelopment(
  project: ProductProject,
  block: bigint,
): ProductProject {
  assertTransition(project, "growing");

  if (project.contributors.length < MIN_CONTRIBUTORS_TO_START) {
    throw new ProjectLifecycleError(
      `Need at least ${MIN_CONTRIBUTORS_TO_START} contributors, have ${project.contributors.length}`,
      "INSUFFICIENT_CONTRIBUTORS",
    );
  }

  if (project.totalTasks === 0) {
    throw new ProjectLifecycleError(
      "Project must have at least one task defined before starting development",
      "NO_TASKS",
    );
  }

  return {
    ...project,
    phase: "growing",
    phaseChangedAtBlock: block,
  };
}

/**
 * Mark project as mature (growing → mature).
 * All tasks must be completed.
 */
export function completeProject(
  project: ProductProject,
  block: bigint,
): ProductProject {
  assertTransition(project, "mature");

  if (project.completedTasks < project.totalTasks) {
    throw new ProjectLifecycleError(
      `Not all tasks completed: ${project.completedTasks}/${project.totalTasks}`,
      "TASKS_INCOMPLETE",
    );
  }

  return {
    ...project,
    phase: "mature",
    phaseChangedAtBlock: block,
  };
}

/**
 * Deploy project as live service (mature → fruiting).
 * Creates service entries in state.
 */
export function deployProject(
  project: ProductProject,
  serviceIds: string[],
  contractAddress: string,
  block: bigint,
): ProductProject {
  assertTransition(project, "fruiting");

  if (serviceIds.length === 0) {
    throw new ProjectLifecycleError(
      "Must deploy at least one service",
      "NO_SERVICES",
    );
  }

  return {
    ...project,
    phase: "fruiting",
    phaseChangedAtBlock: block,
    contractAddress,
    serviceIds,
  };
}

/**
 * Begin spawning child projects from success (fruiting → seeding).
 * Project must have generated meaningful revenue.
 */
export function beginSeeding(
  project: ProductProject,
  block: bigint,
): ProductProject {
  assertTransition(project, "seeding");

  if (project.revenueGenerated === 0n) {
    throw new ProjectLifecycleError(
      "Project must have generated revenue before seeding",
      "NO_REVENUE",
    );
  }

  return {
    ...project,
    phase: "seeding",
    phaseChangedAtBlock: block,
  };
}

/**
 * Pause a project (any active phase → dormant).
 */
export function pauseProject(
  project: ProductProject,
  reason: string,
  block: bigint,
): ProductProject {
  assertTransition(project, "dormant");

  return {
    ...project,
    phase: "dormant",
    previousPhase: project.phase,
    pauseReason: reason || undefined,
    phaseChangedAtBlock: block,
  };
}

/**
 * Resume a dormant project to its previous phase.
 * Only allows returning to the phase the project was in before pausing.
 * If targetPhase is specified, it must match previousPhase.
 */
export function resumeProject(
  project: ProductProject,
  targetPhase: ProjectPhase,
  block: bigint,
): ProductProject {
  if (project.phase !== "dormant") {
    throw new ProjectLifecycleError(
      `Cannot resume: project is in '${project.phase}', not 'dormant'`,
      "INVALID_TRANSITION",
    );
  }

  // Must have a recorded previous phase (set by pauseProject)
  if (!project.previousPhase) {
    throw new ProjectLifecycleError(
      "Cannot resume: no previous phase recorded (project may have been created dormant)",
      "INVALID_TRANSITION",
    );
  }

  // Only allow resuming to the phase the project was in before pausing
  if (targetPhase !== project.previousPhase) {
    throw new ProjectLifecycleError(
      `Cannot resume to '${targetPhase}': project was paused from '${project.previousPhase}', must resume there`,
      "INVALID_TRANSITION",
    );
  }

  return {
    ...project,
    phase: targetPhase,
    previousPhase: undefined,
    pauseReason: undefined,
    phaseChangedAtBlock: block,
  };
}

/**
 * Abandon a project (any phase → withered).
 * Terminal state — budget remainder should be refunded.
 */
export function abandonProject(
  project: ProductProject,
  block: bigint,
): ProductProject {
  assertTransition(project, "withered");

  return {
    ...project,
    phase: "withered",
    phaseChangedAtBlock: block,
  };
}

// ============ Contributor Management ============

/**
 * Add a contributor to the project.
 * Project must be in sprout or growing phase.
 */
export function addContributor(
  project: ProductProject,
  contributor: DID,
  role: ContributorRecord["role"],
  block: bigint,
): ProductProject {
  if (project.phase !== "sprout" && project.phase !== "growing") {
    throw new ProjectLifecycleError(
      `Cannot add contributors in '${project.phase}' phase`,
      "INVALID_PHASE",
    );
  }

  if (project.contributors.some((c) => c.did === contributor)) {
    throw new ProjectLifecycleError(
      `Agent ${contributor} is already a contributor`,
      "DUPLICATE_CONTRIBUTOR",
    );
  }

  return {
    ...project,
    contributors: [
      ...project.contributors,
      {
        did: contributor,
        role,
        joinedAtBlock: block,
        tasksCompleted: 0,
        totalEarned: 0n,
      },
    ],
  };
}

/**
 * Record a task completion for a contributor.
 * Throws if the contributor is not in the project's contributor list,
 * preventing silent weight drift where project.spent diverges from
 * sum(contributors[].totalEarned).
 */
export function recordTaskCompletion(
  project: ProductProject,
  contributor: DID,
  bountyEarned: bigint,
): ProductProject {
  if (!project.contributors.some((c) => c.did === contributor)) {
    throw new ProjectLifecycleError(
      `Agent ${contributor} is not a project contributor`,
      "NOT_A_CONTRIBUTOR",
    );
  }

  if (project.completedTasks >= project.totalTasks) {
    throw new ProjectLifecycleError(
      "All tasks already completed",
      "INVARIANT_VIOLATION",
    );
  }

  const updatedContributors = project.contributors.map((c) =>
    c.did === contributor
      ? {
          ...c,
          tasksCompleted: c.tasksCompleted + 1,
          totalEarned: c.totalEarned + bountyEarned,
        }
      : c,
  );

  return {
    ...project,
    contributors: updatedContributors,
    completedTasks: project.completedTasks + 1,
    spent: project.spent + bountyEarned,
  };
}

/**
 * Record revenue for the project.
 */
export function recordRevenue(
  project: ProductProject,
  amount: bigint,
): ProductProject {
  return {
    ...project,
    revenueGenerated: project.revenueGenerated + amount,
  };
}

// ============ Task Management ============

/**
 * Add a task to the project.
 */
export function addTask(
  project: ProductProject,
  taskId: string,
): ProductProject {
  if (project.phase !== "sprout" && project.phase !== "growing") {
    throw new ProjectLifecycleError(
      `Cannot add tasks in '${project.phase}' phase`,
      "INVALID_PHASE",
    );
  }

  return {
    ...project,
    taskIds: [...project.taskIds, taskId],
    totalTasks: project.totalTasks + 1,
  };
}

// ============ Helpers ============

/**
 * Generate deterministic project ID
 */
export function generateProjectId(founder: DID, nonce: bigint): string {
  const input = `${founder}:${nonce}`;
  return Buffer.from(
    domainHash("LGM.tree.project.id.v1", input).slice(0, 16),
  ).toString("hex");
}

/**
 * Check if a phase transition is valid
 */
export function isValidTransition(
  from: ProjectPhase,
  to: ProjectPhase,
): boolean {
  return VALID_PHASE_TRANSITIONS[from].includes(to);
}

/**
 * Assert a transition is valid, throw if not
 */
function assertTransition(project: ProductProject, target: ProjectPhase): void {
  if (!isValidTransition(project.phase, target)) {
    throw new ProjectLifecycleError(
      `Invalid transition: '${project.phase}' → '${target}'`,
      "INVALID_TRANSITION",
    );
  }
}

// ============ Child Project Spawning (Seeding Phase) ============

/** Maximum fraction of parent revenue that can be allocated to a child (30%) */
export const MAX_CHILD_BUDGET_BP = 3000n; // basis points out of 10000

/**
 * Spawn a child project from a seeding parent.
 * The parent must be in 'seeding' phase and have generated revenue.
 * Child inherits the parent's knowledge domain (or a specified one).
 */
export function spawnChildProject(
  parent: ProductProject,
  founderDid: DID,
  childName: string,
  childDescription: string,
  childSpec: ProductSpec,
  childDomain: string,
  childSignal: DemandSignal,
  budgetBp: bigint,
  nonce: bigint,
  block: bigint,
): { child: ProductProject; budgetAmount: bigint } {
  if (parent.phase !== "seeding") {
    throw new ProjectLifecycleError(
      `Parent must be in 'seeding' phase, is '${parent.phase}'`,
      "INVALID_PHASE",
    );
  }

  if (parent.revenueGenerated === 0n) {
    throw new ProjectLifecycleError(
      "Parent has no revenue to allocate",
      "NO_REVENUE",
    );
  }

  if (budgetBp > MAX_CHILD_BUDGET_BP) {
    throw new ProjectLifecycleError(
      `Budget cannot exceed ${MAX_CHILD_BUDGET_BP} BP (30%) of parent revenue, got ${budgetBp}`,
      "INSUFFICIENT_STAKE",
    );
  }

  if (budgetBp <= 0n) {
    throw new ProjectLifecycleError(
      "Budget must be positive",
      "INSUFFICIENT_STAKE",
    );
  }

  const budgetAmount = (parent.revenueGenerated * budgetBp) / 10000n;

  const childId = generateChildProjectId(parent.id, nonce);

  const child: ProductProject = {
    id: childId,
    name: childName,
    description: childDescription,
    phase: "seed",
    createdAtBlock: block,
    phaseChangedAtBlock: block,
    knowledgeDomain: childDomain,
    demandSignal: childSignal,
    parentProjectId: parent.id,
    founder: founderDid,
    contributors: [
      {
        did: founderDid,
        role: "founder",
        joinedAtBlock: block,
        tasksCompleted: 0,
        totalEarned: 0n,
      },
    ],
    productSpec: childSpec,
    taskIds: [],
    completedTasks: 0,
    totalTasks: 0,
    fundingSource: { type: "research_fund", amount: budgetAmount },
    budget: budgetAmount,
    spent: 0n,
    revenueGenerated: 0n,
    serviceIds: [],
  };

  return { child, budgetAmount };
}

/**
 * Generate deterministic child project ID from parent ID + nonce.
 */
export function generateChildProjectId(
  parentId: string,
  nonce: bigint,
): string {
  const input = `child:${parentId}:${nonce}`;
  return Buffer.from(
    domainHash("LGM.tree.child_project.id.v1", input).slice(0, 16),
  ).toString("hex");
}

// ============ Error Types ============

export type ProjectLifecycleErrorCode =
  | "INVALID_TRANSITION"
  | "INVALID_PHASE"
  | "INSUFFICIENT_STAKE"
  | "INSUFFICIENT_CONTRIBUTORS"
  | "NO_TASKS"
  | "TASKS_INCOMPLETE"
  | "NO_SERVICES"
  | "NO_REVENUE"
  | "DUPLICATE_CONTRIBUTOR"
  | "NOT_A_CONTRIBUTOR"
  | "INVARIANT_VIOLATION";

export class ProjectLifecycleError extends Error {
  constructor(
    message: string,
    public code: ProjectLifecycleErrorCode,
  ) {
    super(message);
    this.name = "ProjectLifecycleError";
  }
}
