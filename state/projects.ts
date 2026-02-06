/**
 * Project State Tree
 *
 * Sparse Merkle Tree for Tree of Life product projects.
 * Key: hash(projectId) - 32 bytes
 * Value: serialized ProductProject
 *
 * Provides:
 * - Cryptographic proof of project existence
 * - Phase-based lifecycle queries
 * - Domain and founder lookups
 * - Budget and revenue tracking
 */

import {
  InMemorySMT,
  type MerkleRoot,
  type MerkleProofData,
} from "../core/crypto/merkle.js";
import { sha512_256 } from "../core/crypto/hashing.js";
import type {
  ProductProject,
  ProjectPhase,
  ContributorRecord,
  DemandSignal,
  ProductSpec,
  FundingSource,
} from "../core/tree/types.js";
import { StateError } from "./types.js";

// ============ Serialization ============

/**
 * Serialize ProductProject to bytes for SMT storage.
 * BigInt fields are converted to strings for JSON compatibility.
 */
function serializeProject(project: ProductProject): Uint8Array {
  const json = JSON.stringify({
    id: project.id,
    name: project.name,
    description: project.description,
    phase: project.phase,
    createdAtBlock: project.createdAtBlock.toString(),
    phaseChangedAtBlock: project.phaseChangedAtBlock.toString(),
    knowledgeDomain: project.knowledgeDomain,
    demandSignal: project.demandSignal,
    parentProjectId: project.parentProjectId,
    founder: project.founder,
    contributors: project.contributors.map((c) => ({
      did: c.did,
      role: c.role,
      joinedAtBlock: c.joinedAtBlock.toString(),
      tasksCompleted: c.tasksCompleted,
      totalEarned: c.totalEarned.toString(),
    })),
    partnershipId: project.partnershipId,
    productSpec: project.productSpec,
    taskIds: project.taskIds,
    completedTasks: project.completedTasks,
    totalTasks: project.totalTasks,
    fundingSource: serializeFundingSource(project.fundingSource),
    budget: project.budget.toString(),
    spent: project.spent.toString(),
    revenueGenerated: project.revenueGenerated.toString(),
    contractAddress: project.contractAddress,
    serviceIds: project.serviceIds,
    applications: (project.applications ?? []).map((a) => ({
      did: a.did,
      proposedRole: a.proposedRole,
      message: a.message,
      capabilities: a.capabilities,
      appliedAtBlock: a.appliedAtBlock.toString(),
      status: a.status,
    })),
  });
  return new TextEncoder().encode(json);
}

/**
 * Serialize FundingSource, converting bigint fields to strings.
 */
function serializeFundingSource(
  source: FundingSource,
): Record<string, unknown> {
  if (source.type === "research_fund") {
    return { type: source.type, amount: source.amount.toString() };
  }
  // Other variants have no bigint fields
  return source as Record<string, unknown>;
}

/**
 * Deserialize ProductProject from bytes.
 * String fields are converted back to BigInt where needed.
 */
function deserializeProject(bytes: Uint8Array): ProductProject {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    id: obj.id,
    name: obj.name,
    description: obj.description,
    phase: obj.phase as ProjectPhase,
    createdAtBlock: BigInt(obj.createdAtBlock),
    phaseChangedAtBlock: BigInt(obj.phaseChangedAtBlock),
    knowledgeDomain: obj.knowledgeDomain,
    demandSignal: obj.demandSignal as DemandSignal,
    parentProjectId: obj.parentProjectId,
    founder: obj.founder,
    contributors: (obj.contributors as Array<Record<string, unknown>>).map(
      (c): ContributorRecord => ({
        did: c.did as string,
        role: c.role as ContributorRecord["role"],
        joinedAtBlock: BigInt(c.joinedAtBlock as string),
        tasksCompleted: c.tasksCompleted as number,
        totalEarned: BigInt(c.totalEarned as string),
      }),
    ),
    partnershipId: obj.partnershipId,
    productSpec: obj.productSpec as ProductSpec,
    taskIds: obj.taskIds,
    completedTasks: obj.completedTasks,
    totalTasks: obj.totalTasks,
    fundingSource: deserializeFundingSource(obj.fundingSource),
    budget: BigInt(obj.budget),
    spent: BigInt(obj.spent),
    revenueGenerated: BigInt(obj.revenueGenerated),
    contractAddress: obj.contractAddress,
    serviceIds: obj.serviceIds,
    applications: obj.applications
      ? (obj.applications as Array<Record<string, unknown>>).map(
          (a): import("../core/tree/types.js").ProjectApplication => ({
            did: a.did as string,
            proposedRole: a.proposedRole as ContributorRecord["role"],
            message: a.message as string,
            capabilities: a.capabilities as string[],
            appliedAtBlock: BigInt(a.appliedAtBlock as string),
            status: a.status as "pending" | "approved" | "rejected",
          }),
        )
      : [],
  };
}

/**
 * Deserialize FundingSource, converting string fields back to bigint.
 */
function deserializeFundingSource(obj: Record<string, unknown>): FundingSource {
  if (obj.type === "research_fund") {
    return { type: "research_fund", amount: BigInt(obj.amount as string) };
  }
  return obj as FundingSource;
}

/**
 * Convert project ID to SMT key
 */
function projectIdToKey(projectId: string): Uint8Array {
  return sha512_256(projectId);
}

// ============ Project State Tree ============

export class ProjectStateTree {
  private smt: InMemorySMT;
  private cache: Map<string, ProductProject> = new Map();
  private projectCount: number = 0;

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
   * Get project by ID.
   * WARNING: Returns a cached reference. Do NOT mutate the returned object directly.
   * Use updateProject() for all modifications.
   */
  getProject(projectId: string): ProductProject | null {
    const cached = this.cache.get(projectId);
    if (cached) return cached;

    const key = projectIdToKey(projectId);
    const value = this.smt.get(key);
    if (!value) return null;

    const project = deserializeProject(value);
    this.cache.set(projectId, project);
    return project;
  }

  /**
   * Check if project exists
   */
  hasProject(projectId: string): boolean {
    return this.getProject(projectId) !== null;
  }

  /**
   * Create a new project
   */
  createProject(project: ProductProject): MerkleRoot {
    if (this.hasProject(project.id)) {
      throw new StateError("Project already exists", "INVALID_TRANSITION", {
        projectId: project.id,
      });
    }

    const key = projectIdToKey(project.id);
    const value = serializeProject(project);
    const newRoot = this.smt.set(key, value);

    this.cache.set(project.id, project);
    this.projectCount++;

    return newRoot;
  }

  /**
   * Update an existing project
   */
  updateProject(
    projectId: string,
    updates: Partial<ProductProject>,
  ): MerkleRoot {
    const existing = this.getProject(projectId);
    if (!existing) {
      throw new StateError("Project not found", "PROJECT_NOT_FOUND", {
        projectId,
      });
    }

    const updated: ProductProject = {
      ...existing,
      ...updates,
    };

    const key = projectIdToKey(projectId);
    const value = serializeProject(updated);
    const newRoot = this.smt.set(key, value);

    this.cache.set(projectId, updated);
    return newRoot;
  }

  // ============ Query Operations ============

  /**
   * Get all projects in a given phase
   */
  getProjectsByPhase(phase: ProjectPhase): ProductProject[] {
    const results: ProductProject[] = [];
    for (const project of this.cache.values()) {
      if (project.phase === phase) {
        results.push(project);
      }
    }
    return results;
  }

  /**
   * Get all projects in a knowledge domain
   */
  getProjectsByDomain(domain: string): ProductProject[] {
    const results: ProductProject[] = [];
    for (const project of this.cache.values()) {
      if (project.knowledgeDomain === domain) {
        results.push(project);
      }
    }
    return results;
  }

  /**
   * Get all projects by a founder
   */
  getProjectsByFounder(founder: string): ProductProject[] {
    const results: ProductProject[] = [];
    for (const project of this.cache.values()) {
      if (project.founder === founder) {
        results.push(project);
      }
    }
    return results;
  }

  /**
   * Get total project count
   */
  getProjectCount(): number {
    return this.projectCount;
  }

  // ============ Proofs ============

  /**
   * Generate Merkle proof for a project.
   * Proves inclusion (or non-inclusion) in the project state tree.
   */
  proveProject(projectId: string): MerkleProofData {
    const key = projectIdToKey(projectId);
    return this.smt.prove(key);
  }

  // ============ Persistence Export/Restore ============

  /**
   * Export all projects for persistence
   */
  exportProjects(): ProductProject[] {
    return Array.from(this.cache.values());
  }

  // ============ Utilities ============

  /**
   * Clone for speculative execution
   */
  clone(): ProjectStateTree {
    const cloned = new ProjectStateTree();

    for (const [projectId, project] of this.cache) {
      // Deep clone the project — spread top-level, clone nested arrays/objects
      const clonedProject: ProductProject = {
        ...project,
        demandSignal: { ...project.demandSignal },
        contributors: project.contributors.map((c) => ({ ...c })),
        productSpec: {
          ...project.productSpec,
          requirements: [...project.productSpec.requirements],
          capabilities: [...project.productSpec.capabilities],
          mvpCriteria: [...project.productSpec.mvpCriteria],
        },
        taskIds: [...project.taskIds],
        serviceIds: [...project.serviceIds],
      };

      cloned.cache.set(projectId, clonedProject);
      const key = projectIdToKey(projectId);
      cloned.smt.set(key, serializeProject(clonedProject));
    }

    cloned.projectCount = this.projectCount;
    return cloned;
  }
}
