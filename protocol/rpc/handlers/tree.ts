/**
 * Tree of Life RPC Handlers
 *
 * Endpoints for product project lifecycle, service management,
 * and opportunity detection.
 *
 * Read Endpoints:
 * - tree_getProject: Get project details
 * - tree_listProjects: List projects by phase/domain
 * - tree_getService: Get service details
 * - tree_listServices: List services by status
 * - tree_getOpportunities: List detected opportunity seeds
 * - tree_getProjectRevenue: Revenue summary for a project
 * - tree_getDomainProposal: Get a domain proposal by ID
 * - tree_listDomainProposals: List domain proposals (filterable by status)
 * - tree_getDomainTree: Get registered ontological domains
 *
 * Write Endpoints:
 * - tree_proposeProject: Create and propose a new project
 * - tree_addContributor: Add contributor to a project
 * - tree_claimTask: Agent claims an available task
 * - tree_submitDeliverable: Submit work for a task
 * - tree_callService: Call a deployed service (revenue generator)
 * - tree_deployService: Deploy a service for a project
 * - tree_pauseService: Pause an active service
 * - tree_resumeService: Resume a paused service
 * - tree_retireService: Retire a service permanently
 */

import type { Handler } from "../handlers.js";
import { verify } from "../../../core/crypto/signatures.js";
import { sha512_256Hex } from "../../../core/crypto/hashing.js";
import { RPC_ERROR_CODES } from "../../types.js";
import { validatePublicKey } from "../validation.js";
import { publicKeyToDid } from "../../../core/agents/identity/did.js";
import {
  DEFAULT_CONSENSUS_PARAMS,
  type Transaction,
  type TransactionType,
} from "../../../consensus/pot/types.js";
import type {
  ProjectPhase,
  ServiceStatus,
  ServiceType,
  ContributorRole,
} from "../../../core/tree/types.js";

// ============ Request/Response Types (Read) ============

export interface TreeGetProjectRequest {
  projectId: string;
}

export interface TreeGetProjectResponse {
  project: Record<string, unknown> | null;
}

export interface TreeListProjectsRequest {
  phase?: string;
  domain?: string;
  offset?: number;
  limit?: number;
}

export interface TreeListProjectsResponse {
  projects: Record<string, unknown>[];
  total: number;
}

export interface TreeGetServiceRequest {
  serviceId: string;
}

export interface TreeGetServiceResponse {
  service: Record<string, unknown> | null;
}

export interface TreeResolveSubdomainRequest {
  subdomain: string;
}

export interface TreeResolveSubdomainResponse {
  service: Record<string, unknown> | null;
  subdomain: string;
}

export interface TreeListServicesRequest {
  projectId?: string;
  status?: string;
  offset?: number;
  limit?: number;
}

export interface TreeListServicesResponse {
  services: Record<string, unknown>[];
  total: number;
}

export interface TreeGetOpportunitiesRequest {
  domain?: string;
  minConfidence?: number;
  offset?: number;
  limit?: number;
}

export interface TreeGetOpportunitiesResponse {
  opportunities: Record<string, unknown>[];
  total: number;
}

export interface TreeGetProjectRevenueRequest {
  projectId: string;
}

export interface TreeGetProjectRevenueResponse {
  projectId: string;
  totalRevenue: string;
  contributorCount: number;
}

export interface TreeGetDomainProposalRequest {
  proposalId: string;
}

export interface TreeGetDomainProposalResponse {
  proposal: Record<string, unknown> | null;
}

export interface TreeListDomainProposalsRequest {
  status?: string;
  offset?: number;
  limit?: number;
}

export interface TreeListDomainProposalsResponse {
  proposals: Record<string, unknown>[];
  total: number;
}

export interface TreeGetDomainTreeRequest {
  stratum?: string;
}

export interface TreeGetDomainTreeResponse {
  domains: Record<string, unknown>[];
  domainCount: number;
  maxDepth: number;
}

// ============ Request/Response Types (Write) ============

export interface TreeProposeProjectRequest {
  name: string;
  description: string;
  knowledgeDomain: string;
  productSpec: Record<string, unknown>;
  fundingSource: Record<string, unknown>;
  demandSignal: Record<string, unknown>;
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreeWriteResponse {
  txHash: string;
  success: boolean;
  error?: string;
}

export interface TreeAddContributorRequest {
  projectId: string;
  contributorDid: string;
  role: string;
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreeClaimTaskRequest {
  projectId: string;
  taskId: string;
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreeSubmitDeliverableRequest {
  projectId: string;
  taskId: string;
  deliverable: Record<string, unknown>;
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreeCallServiceRequest {
  serviceId: string;
  units: string;
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreeDeployServiceRequest {
  projectId: string;
  name: string;
  description: string;
  serviceType: string;
  endpoint: Record<string, unknown>;
  abi: Record<string, unknown>;
  pricing: Record<string, unknown>;
  contractAddress: string;
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreePauseResumeRetireServiceRequest {
  serviceId: string;
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

// ============ Recruitment Request/Response Types ============

export interface TreeClaimOpportunityRequest {
  seedId: string;
  projectName: string;
  projectDescription: string;
  productSpec: Record<string, unknown>;
  fundingSource: Record<string, unknown>;
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreeApplyToProjectRequest {
  projectId: string;
  proposedRole: string;
  message: string;
  declaredCapabilities: string[];
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreeReviewApplicationRequest {
  projectId: string;
  applicantDid: string;
  decision: "approve" | "reject";
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreeSetAvailabilityRequest {
  available: boolean;
  capabilities: string[];
  preferredDomains: string[];
  minimumBounty: string;
  nonce: string;
  fee: string;
  signature: string;
  sender: string;
}

export interface TreeMatchAgentsToTaskRequest {
  taskId: string;
  candidateProfiles?: Array<{
    did: string;
    capabilities: Record<string, { level: number } | null>;
    reputation: number;
    contributions: {
      total: number;
      successRate: number;
      byType: Record<string, number>;
    };
    lastActiveAt: number;
  }>;
  limit?: number;
}

export interface TreeGetAgentReputationRequest {
  agentDid: string;
}

export interface TreeGetAvailableAgentsRequest {
  capability?: string;
  domain?: string;
  offset?: number;
  limit?: number;
}

export interface TreeGetProjectApplicationsRequest {
  projectId: string;
  status?: string;
}

// ============ Validation Constants ============

const VALID_PHASES: ReadonlySet<string> = new Set<ProjectPhase>([
  "seed",
  "sprout",
  "growing",
  "mature",
  "fruiting",
  "seeding",
  "dormant",
  "withered",
]);

const VALID_SERVICE_STATUSES: ReadonlySet<string> = new Set<ServiceStatus>([
  "deploying",
  "active",
  "degraded",
  "paused",
  "retired",
]);

const VALID_CONTRIBUTOR_ROLES: ReadonlySet<string> = new Set<ContributorRole>([
  "founder",
  "architect",
  "developer",
  "reviewer",
  "tester",
  "maintainer",
]);

const VALID_SERVICE_TYPES: ReadonlySet<string> = new Set<ServiceType>([
  "api",
  "data_feed",
  "computation",
  "verification",
  "aggregation",
  "analytics",
]);

// ============ Helpers ============

/**
 * Recursively convert bigint fields to strings for JSON serialization.
 * Accepts any object type — avoids double-casting typed state objects.
 */
function serializeBigints(obj: object): Record<string, unknown> {
  const result: Record<string, unknown> = {};
  for (const [key, value] of Object.entries(obj)) {
    if (typeof value === "bigint") {
      result[key] = value.toString();
    } else if (Array.isArray(value)) {
      result[key] = value.map((v) =>
        typeof v === "object" && v !== null
          ? serializeBigints(v as Record<string, unknown>)
          : v,
      );
    } else if (typeof value === "object" && value !== null) {
      result[key] = serializeBigints(value as Record<string, unknown>);
    } else {
      result[key] = value;
    }
  }
  return result;
}

function createRPCError(code: number, message: string): Error {
  const error = new Error(message) as Error & { code: number; data: unknown };
  error.code = code;
  error.data = { errorCode: "TREE_ERROR", retryable: false };
  return error;
}

/**
 * Verify signature on a signed transaction payload.
 * Returns the sender's DID derived from public key.
 */
function verifySender(
  params: { signature: string; sender: string },
  payload: Record<string, unknown>,
): string {
  validatePublicKey(params.sender, "sender");

  const payloadStr = JSON.stringify(payload);
  const valid = verify(
    new TextEncoder().encode(payloadStr),
    params.signature,
    params.sender,
  );

  if (!valid) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_SIGNATURE,
      "Invalid signature",
    );
  }

  return publicKeyToDid(params.sender);
}

/**
 * Execute a transaction through the state transition layer.
 */
async function executeTreeTransaction(
  ctx: Parameters<Handler<unknown, unknown>>[1],
  txType: TransactionType,
  sender: string,
  payload: Record<string, unknown>,
  fee: string,
  nonce: string,
): Promise<TreeWriteResponse> {
  const { executeTransaction } = await import("../../../state/transitions.js");

  const txHash = sha512_256Hex(
    `tree:${txType}:${sender}:${nonce}:${Date.now()}`,
  );
  const blockNumber = ctx.state.getBlockNumber();

  const transaction: Transaction = {
    type: txType,
    sender,
    fee: BigInt(fee),
    nonce: BigInt(nonce),
    payload: { type: txType, ...payload } as Transaction["payload"],
    signature: "",
  };

  try {
    const result = executeTransaction(
      ctx.state,
      transaction,
      blockNumber,
      DEFAULT_CONSENSUS_PARAMS,
    );

    if (result.success) {
      return { txHash, success: true };
    } else {
      return { txHash, success: false, error: result.error };
    }
  } catch (error) {
    return {
      txHash,
      success: false,
      error: error instanceof Error ? error.message : "Transaction failed",
    };
  }
}

// ============ Read Handlers ============

export const treeGetProject: Handler<
  TreeGetProjectRequest,
  TreeGetProjectResponse
> = async (params, ctx) => {
  const project = ctx.state.getProject(params.projectId);

  return {
    project: project ? serializeBigints(project) : null,
  };
};

export const treeListProjects: Handler<
  TreeListProjectsRequest,
  TreeListProjectsResponse
> = async (params, ctx) => {
  let projects;

  if (params.domain) {
    projects = ctx.state.getProjectsByDomain(params.domain);
    // Apply phase filter on top of domain filter
    if (params.phase) {
      projects = projects.filter((p) => p.phase === params.phase);
    }
  } else if (params.phase) {
    if (!VALID_PHASES.has(params.phase)) {
      throw createRPCError(
        RPC_ERROR_CODES.INVALID_PARAMS,
        `Invalid phase: ${params.phase}`,
      );
    }
    projects = ctx.state.getProjectsByPhase(params.phase as ProjectPhase);
  } else {
    // Return all projects by getting each phase
    const phases = [
      "seed",
      "sprout",
      "growing",
      "mature",
      "fruiting",
      "seeding",
      "dormant",
      "withered",
    ] as const;
    projects = [];
    for (const phase of phases) {
      projects.push(...ctx.state.getProjectsByPhase(phase));
    }
  }

  const offset = params.offset ?? 0;
  const limit = params.limit ?? 50;
  const limited = projects.slice(offset, offset + limit);

  return {
    projects: limited.map((p) => serializeBigints(p)),
    total: projects.length,
  };
};

export const treeGetService: Handler<
  TreeGetServiceRequest,
  TreeGetServiceResponse
> = async (params, ctx) => {
  const service = ctx.state.getService(params.serviceId);

  return {
    service: service ? serializeBigints(service) : null,
  };
};

/**
 * Resolve a service by its protocol-assigned subdomain.
 * Returns the service and the queried subdomain.
 */
export const treeResolveSubdomain: Handler<
  TreeResolveSubdomainRequest,
  TreeResolveSubdomainResponse
> = async (params, ctx) => {
  if (!params.subdomain) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_PARAMS,
      "subdomain is required",
    );
  }

  const service = ctx.state.getServiceBySubdomain(params.subdomain);

  return {
    service: service ? serializeBigints(service) : null,
    subdomain: params.subdomain,
  };
};

export const treeListServices: Handler<
  TreeListServicesRequest,
  TreeListServicesResponse
> = async (params, ctx) => {
  let services;

  if (params.status) {
    if (!VALID_SERVICE_STATUSES.has(params.status)) {
      throw createRPCError(
        RPC_ERROR_CODES.INVALID_PARAMS,
        `Invalid status: ${params.status}`,
      );
    }
    services = ctx.state.getServicesByStatus(params.status as ServiceStatus);
  } else {
    services = ctx.state.getActiveServices(); // Default to active
  }

  const offset = params.offset ?? 0;
  const limit = params.limit ?? 50;
  const limited = services.slice(offset, offset + limit);

  return {
    services: limited.map((s) => serializeBigints(s)),
    total: services.length,
  };
};

export const treeGetProjectRevenue: Handler<
  TreeGetProjectRevenueRequest,
  TreeGetProjectRevenueResponse
> = async (params, ctx) => {
  const project = ctx.state.getProject(params.projectId);

  if (!project) {
    return {
      projectId: params.projectId,
      totalRevenue: "0",
      contributorCount: 0,
    };
  }

  return {
    projectId: params.projectId,
    totalRevenue: project.revenueGenerated.toString(),
    contributorCount: project.contributors.length,
  };
};

export const treeGetOpportunities: Handler<
  TreeGetOpportunitiesRequest,
  TreeGetOpportunitiesResponse
> = async (params, ctx) => {
  // Gather existing project domains to exclude from opportunity detection
  const phases = [
    "seed",
    "sprout",
    "growing",
    "mature",
    "fruiting",
    "seeding",
    "dormant",
    "withered",
  ] as const;
  const existingProjectDomains: string[] = [];
  for (const phase of phases) {
    for (const p of ctx.state.getProjectsByPhase(phase)) {
      if (!existingProjectDomains.includes(p.knowledgeDomain)) {
        existingProjectDomains.push(p.knowledgeDomain);
      }
    }
  }

  // Detect opportunities from current state metrics.
  const {
    detectOpportunities,
    aggregateDomainMetrics,
    aggregateCrossDomainMetrics,
  } = await import("../../../core/tree/opportunity_detector.js");

  const currentBlock = ctx.state.getBlockNumber();

  const domainMetrics = aggregateDomainMetrics(
    (d: string) => ctx.state.getFactsByDomain(d),
    () => ctx.state.getDomainIds(),
    () => ctx.state.getAllBounties(),
    (fId: string) => ctx.state.getFact(fId),
  );

  const crossDomainMetrics = aggregateCrossDomainMetrics(
    (d: string) => ctx.state.getFactsByDomain(d),
    () => ctx.state.getDomainIds(),
    (fId: string) => ctx.state.getOutgoingFactReferences(fId),
    (fId: string) => ctx.state.getFact(fId),
  );

  const seeds = detectOpportunities(
    domainMetrics,
    [], // MarketMetrics — no orderbook in state yet
    existingProjectDomains,
    currentBlock,
    currentBlock,
    [], // agentRequests — not yet exposed via RPC
    crossDomainMetrics,
  );

  // Apply filters
  let filtered = seeds;
  if (params.domain) {
    filtered = filtered.filter((s) => s.knowledgeDomain === params.domain);
  }
  if (params.minConfidence !== undefined) {
    filtered = filtered.filter((s) => s.confidence >= params.minConfidence!);
  }

  const offset = params.offset ?? 0;
  const limit = params.limit ?? 50;
  const limited = filtered.slice(offset, offset + limit);

  return {
    opportunities: limited.map((s) => serializeBigints(s)),
    total: filtered.length,
  };
};

// ============ Domain Query Handlers ============

/**
 * Get a single domain proposal by ID.
 */
export const treeGetDomainProposal: Handler<
  TreeGetDomainProposalRequest,
  TreeGetDomainProposalResponse
> = async (params, ctx) => {
  const proposal = ctx.state.getDomainProposal(params.proposalId);
  return {
    proposal: proposal ? serializeBigints(proposal) : null,
  };
};

/**
 * List domain proposals, optionally filtered by status.
 */
export const treeListDomainProposals: Handler<
  TreeListDomainProposalsRequest,
  TreeListDomainProposalsResponse
> = async (params, ctx) => {
  let proposals = ctx.state.getAllDomainProposals();

  if (params.status) {
    proposals = proposals.filter((p) => p.status === params.status);
  }

  const offset = params.offset ?? 0;
  const limit = params.limit ?? 50;
  const page = proposals.slice(offset, offset + limit);

  return {
    proposals: page.map((p) => serializeBigints(p)),
    total: proposals.length,
  };
};

/**
 * Get the ontological domain tree (all registered domains).
 */
export const treeGetDomainTree: Handler<
  TreeGetDomainTreeRequest,
  TreeGetDomainTreeResponse
> = async (params, ctx) => {
  const domainTree = ctx.state.getKnowledgeDomainTree();
  const domains: Array<Record<string, unknown>> = [];

  for (const [_id, node] of domainTree.domainIndex) {
    if (params.stratum && node.domain.stratum !== params.stratum) continue;
    domains.push(serializeBigints(node.domain));
  }

  return {
    domains,
    domainCount: domainTree.domainCount,
    maxDepth: domainTree.maxDepth,
  };
};

// ============ Write Handlers ============

/**
 * Create and propose a new project in a knowledge domain.
 * Executes create_project + propose_project as an atomic pair.
 */
export const treeProposeProject: Handler<
  TreeProposeProjectRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const payload = {
    name: params.name,
    description: params.description,
    knowledgeDomain: params.knowledgeDomain,
    productSpec: params.productSpec,
    fundingSource: params.fundingSource,
    demandSignal: params.demandSignal,
    nonce: params.nonce,
    fee: params.fee,
  };

  const senderPubKey = verifySender(params, payload);

  // Step 1: Create the project
  const createResult = await executeTreeTransaction(
    ctx,
    "create_project",
    senderPubKey,
    {
      name: params.name,
      description: params.description,
      knowledgeDomain: params.knowledgeDomain,
      productSpec: params.productSpec,
      fundingSource: params.fundingSource,
      demandSignal: params.demandSignal,
    },
    params.fee,
    params.nonce,
  );

  if (!createResult.success) {
    return createResult;
  }

  // Step 2: Propose (transition seed → sprout, locks stake)
  // Find the newly created project by getting all seed projects and finding ours
  const senderDid = publicKeyToDid(params.sender);
  const seedProjects = ctx.state.getProjectsByPhase("seed");
  const newProject = seedProjects.find(
    (p) => p.founder === senderDid && p.name === params.name,
  );

  if (!newProject) {
    return {
      txHash: createResult.txHash,
      success: true, // Project was created but not proposed
    };
  }

  const proposeResult = await executeTreeTransaction(
    ctx,
    "propose_project",
    senderPubKey,
    { projectId: newProject.id },
    "0",
    params.nonce,
  );

  return {
    txHash: createResult.txHash,
    success: proposeResult.success,
    error: proposeResult.error,
  };
};

/**
 * Add a contributor to a project.
 * Caller must be the project founder or an existing contributor.
 */
export const treeAddContributor: Handler<
  TreeAddContributorRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const payload = {
    projectId: params.projectId,
    contributorDid: params.contributorDid,
    role: params.role,
    nonce: params.nonce,
    fee: params.fee,
  };

  const senderPubKey = verifySender(params, payload);

  // Validate project exists
  const project = ctx.state.getProject(params.projectId);
  if (!project) {
    throw createRPCError(RPC_ERROR_CODES.INVALID_PARAMS, "Project not found");
  }

  // Use core lifecycle function to produce updated project
  const { addContributor } =
    await import("../../../core/tree/project_lifecycle.js");

  try {
    const blockNumber = ctx.state.getBlockNumber();
    if (!VALID_CONTRIBUTOR_ROLES.has(params.role)) {
      throw createRPCError(
        RPC_ERROR_CODES.INVALID_PARAMS,
        `Invalid role: ${params.role}`,
      );
    }
    const updated = addContributor(
      project,
      params.contributorDid,
      params.role as ContributorRole,
      blockNumber,
    );
    ctx.state.updateProject(params.projectId, updated);

    const txHash = sha512_256Hex(
      `tree:add_contributor:${senderPubKey}:${params.nonce}:${Date.now()}`,
    );
    return { txHash, success: true };
  } catch (error) {
    const txHash = sha512_256Hex(
      `tree:add_contributor:${senderPubKey}:${params.nonce}:${Date.now()}`,
    );
    return {
      txHash,
      success: false,
      error:
        error instanceof Error ? error.message : "Failed to add contributor",
    };
  }
};

/**
 * Agent claims (assigns self to) a task on a project.
 * Routes through the assign_task state transition.
 */
export const treeClaimTask: Handler<
  TreeClaimTaskRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const payload = {
    projectId: params.projectId,
    taskId: params.taskId,
    nonce: params.nonce,
    fee: params.fee,
  };

  const senderPubKey = verifySender(params, payload);
  const senderDid = publicKeyToDid(params.sender);

  return executeTreeTransaction(
    ctx,
    "assign_task",
    senderPubKey,
    {
      projectId: params.projectId,
      taskId: params.taskId,
      agentDid: senderDid,
    },
    params.fee,
    params.nonce,
  );
};

/**
 * Submit a deliverable for a task.
 * Routes through the submit_deliverable state transition.
 */
export const treeSubmitDeliverable: Handler<
  TreeSubmitDeliverableRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const payload = {
    projectId: params.projectId,
    taskId: params.taskId,
    deliverable: params.deliverable,
    nonce: params.nonce,
    fee: params.fee,
  };

  const senderPubKey = verifySender(params, payload);

  return executeTreeTransaction(
    ctx,
    "submit_deliverable",
    senderPubKey,
    {
      projectId: params.projectId,
      taskId: params.taskId,
      deliverable: params.deliverable,
    },
    params.fee,
    params.nonce,
  );
};

/**
 * Call a deployed service. Deducts LGM payment and distributes revenue.
 * Routes through the call_service state transition.
 */
export const treeCallService: Handler<
  TreeCallServiceRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const payload = {
    serviceId: params.serviceId,
    units: params.units,
    nonce: params.nonce,
    fee: params.fee,
  };

  const senderPubKey = verifySender(params, payload);

  return executeTreeTransaction(
    ctx,
    "call_service",
    senderPubKey,
    {
      serviceId: params.serviceId,
      units: BigInt(params.units),
    },
    params.fee,
    params.nonce,
  );
};

/**
 * Deploy a service for a project.
 * Creates the service and transitions project to fruiting if mature.
 */
export const treeDeployService: Handler<
  TreeDeployServiceRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const payload = {
    projectId: params.projectId,
    name: params.name,
    description: params.description,
    serviceType: params.serviceType,
    endpoint: params.endpoint,
    abi: params.abi,
    pricing: params.pricing,
    contractAddress: params.contractAddress,
    nonce: params.nonce,
    fee: params.fee,
  };

  const senderPubKey = verifySender(params, payload);

  if (!VALID_SERVICE_TYPES.has(params.serviceType)) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_PARAMS,
      `Invalid serviceType: ${params.serviceType}`,
    );
  }

  return executeTreeTransaction(
    ctx,
    "deploy_service",
    senderPubKey,
    {
      projectId: params.projectId,
      name: params.name,
      description: params.description,
      serviceType: params.serviceType,
      endpoint: params.endpoint,
      abi: params.abi,
      pricing: params.pricing,
      contractAddress: params.contractAddress,
    },
    params.fee,
    params.nonce,
  );
};

/**
 * Pause an active service.
 * Only the project founder can pause.
 */
export const treePauseService: Handler<
  TreePauseResumeRetireServiceRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const payload = {
    serviceId: params.serviceId,
    nonce: params.nonce,
    fee: params.fee,
  };

  const senderPubKey = verifySender(params, payload);

  return executeTreeTransaction(
    ctx,
    "pause_service",
    senderPubKey,
    { serviceId: params.serviceId },
    params.fee,
    params.nonce,
  );
};

/**
 * Resume a paused service.
 * Only the project founder can resume.
 */
export const treeResumeService: Handler<
  TreePauseResumeRetireServiceRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const payload = {
    serviceId: params.serviceId,
    nonce: params.nonce,
    fee: params.fee,
  };

  const senderPubKey = verifySender(params, payload);

  return executeTreeTransaction(
    ctx,
    "resume_service",
    senderPubKey,
    { serviceId: params.serviceId },
    params.fee,
    params.nonce,
  );
};

/**
 * Retire a service permanently.
 * Only the project founder can retire.
 */
export const treeRetireService: Handler<
  TreePauseResumeRetireServiceRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const payload = {
    serviceId: params.serviceId,
    nonce: params.nonce,
    fee: params.fee,
  };

  const senderPubKey = verifySender(params, payload);

  return executeTreeTransaction(
    ctx,
    "retire_service",
    senderPubKey,
    { serviceId: params.serviceId },
    params.fee,
    params.nonce,
  );
};

// ============ Recruitment Handlers ============

/**
 * Claim an opportunity seed and create a project from it.
 */
export const treeClaimOpportunity: Handler<
  TreeClaimOpportunityRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const senderPubKey = verifySender(params, {
    seedId: params.seedId,
    projectName: params.projectName,
    projectDescription: params.projectDescription,
    productSpec: params.productSpec,
    fundingSource: params.fundingSource,
    nonce: params.nonce,
    fee: params.fee,
  });

  return executeTreeTransaction(
    ctx,
    "claim_opportunity",
    senderPubKey,
    {
      seedId: params.seedId,
      projectName: params.projectName,
      projectDescription: params.projectDescription,
      productSpec: params.productSpec,
      fundingSource: params.fundingSource,
    },
    params.fee,
    params.nonce,
  );
};

/**
 * Agent applies to join a project.
 */
export const treeApplyToProject: Handler<
  TreeApplyToProjectRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  if (!VALID_CONTRIBUTOR_ROLES.has(params.proposedRole)) {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_PARAMS,
      `Invalid role: ${params.proposedRole}`,
    );
  }

  const senderPubKey = verifySender(params, {
    projectId: params.projectId,
    proposedRole: params.proposedRole,
    message: params.message,
    declaredCapabilities: params.declaredCapabilities,
    nonce: params.nonce,
    fee: params.fee,
  });

  return executeTreeTransaction(
    ctx,
    "apply_to_project",
    senderPubKey,
    {
      projectId: params.projectId,
      proposedRole: params.proposedRole,
      message: params.message,
      declaredCapabilities: params.declaredCapabilities,
    },
    params.fee,
    params.nonce,
  );
};

/**
 * Founder approves or rejects an application.
 */
export const treeReviewApplication: Handler<
  TreeReviewApplicationRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  if (params.decision !== "approve" && params.decision !== "reject") {
    throw createRPCError(
      RPC_ERROR_CODES.INVALID_PARAMS,
      `Invalid decision: ${params.decision}`,
    );
  }

  const senderPubKey = verifySender(params, {
    projectId: params.projectId,
    applicantDid: params.applicantDid,
    decision: params.decision,
    nonce: params.nonce,
    fee: params.fee,
  });

  return executeTreeTransaction(
    ctx,
    "review_application",
    senderPubKey,
    {
      projectId: params.projectId,
      applicantDid: params.applicantDid,
      decision: params.decision,
    },
    params.fee,
    params.nonce,
  );
};

/**
 * Agent sets availability for project work.
 */
export const treeSetAvailability: Handler<
  TreeSetAvailabilityRequest,
  TreeWriteResponse
> = async (params, ctx) => {
  const senderPubKey = verifySender(params, {
    available: params.available,
    capabilities: params.capabilities,
    preferredDomains: params.preferredDomains,
    minimumBounty: params.minimumBounty,
    nonce: params.nonce,
    fee: params.fee,
  });

  return executeTreeTransaction(
    ctx,
    "set_availability",
    senderPubKey,
    {
      available: params.available,
      capabilities: params.capabilities,
      preferredDomains: params.preferredDomains,
      minimumBounty: params.minimumBounty,
    },
    params.fee,
    params.nonce,
  );
};

/**
 * Get project applications (read).
 */
export const treeGetProjectApplications: Handler<
  TreeGetProjectApplicationsRequest,
  { applications: Array<Record<string, unknown>>; total: number }
> = async (params, ctx) => {
  const project = ctx.state.getProject(params.projectId);
  if (!project) {
    throw createRPCError(RPC_ERROR_CODES.INVALID_PARAMS, "Project not found");
  }

  let applications = project.applications ?? [];
  if (params.status) {
    applications = applications.filter((a) => a.status === params.status);
  }

  return {
    applications: applications.map((a) => ({
      did: a.did,
      proposedRole: a.proposedRole,
      message: a.message,
      capabilities: a.capabilities,
      appliedAtBlock: a.appliedAtBlock.toString(),
      status: a.status,
    })),
    total: applications.length,
  };
};

/**
 * Get available agents for project recruitment (read).
 */
export const treeGetAvailableAgents: Handler<
  TreeGetAvailableAgentsRequest,
  { agents: Array<Record<string, unknown>>; total: number }
> = async (params, ctx) => {
  const agents = ctx.state.getAvailableAgents({
    capability: params.capability,
    domain: params.domain,
  });

  const offset = params.offset ?? 0;
  const limit = Math.min(params.limit ?? 50, 100);
  const paginated = agents.slice(offset, offset + limit);

  return {
    agents: paginated.map((a) => ({
      did: a.did,
      capabilities: a.availability.capabilities,
      preferredDomains: a.availability.preferredDomains,
      minimumBounty: a.availability.minimumBounty.toString(),
      updatedAtBlock: a.availability.updatedAtBlock.toString(),
    })),
    total: agents.length,
  };
};

/**
 * Get agent's cross-project reputation (read).
 */
export const treeGetAgentReputation: Handler<
  TreeGetAgentReputationRequest,
  Record<string, unknown>
> = async (params, ctx) => {
  const rep = ctx.state.getAgentCrossProjectReputation(params.agentDid);

  return {
    projectsContributed: rep.projectsContributed,
    totalTasksCompleted: rep.totalTasksCompleted,
    totalEarned: rep.totalEarned.toString(),
    roleBreakdown: rep.roleBreakdown,
    topDomains: rep.topDomains,
    projectsByPhase: rep.projectsByPhase,
  };
};

/**
 * Match agents to a task using skill matching (read).
 * Accepts optional candidate profiles; if omitted, returns empty (no profile storage).
 */
export const treeMatchAgentsToTask: Handler<
  TreeMatchAgentsToTaskRequest,
  { matches: Array<Record<string, unknown>>; total: number }
> = async (params, ctx) => {
  const task = ctx.state.getTask(params.taskId);
  if (!task) {
    throw createRPCError(RPC_ERROR_CODES.INVALID_PARAMS, "Task not found");
  }

  const { matchAgentsToTask } =
    await import("../../../core/tree/coordination/skill_matching.js");

  // Build AgentProfile[] from candidate profiles if provided
  const profiles = (params.candidateProfiles ?? []).map((cp) => {
    const caps: Record<string, unknown> = {};
    for (const [key, val] of Object.entries(cp.capabilities)) {
      caps[key] = val
        ? {
            type: key,
            level: val.level,
            status: "attested",
            attestedBy: [],
            evidence: [],
            stakedAmount: 0n,
          }
        : null;
    }

    return {
      did: cp.did,
      name: "",
      description: "",
      specializations: [],
      operationalKeyVersion: 0,
      sessionKeyCount: 0,
      capabilities: caps as any,
      reputation: {
        overall: cp.reputation,
        domains: { verification: 0, research: 0, development: 0, testing: 0 },
        history: [],
        decay: 0,
        lastCalculatedAt: 0,
      },
      contributions: {
        total: cp.contributions.total,
        byType: {
          verifications: cp.contributions.byType.verifications ?? 0,
          claims: cp.contributions.byType.claims ?? 0,
          challenges: cp.contributions.byType.challenges ?? 0,
          codeChanges: cp.contributions.byType.codeChanges ?? 0,
          tests: cp.contributions.byType.tests ?? 0,
          reviews: cp.contributions.byType.reviews ?? 0,
        },
        successRate: cp.contributions.successRate,
        recentContributions: [],
      },
      stake: {
        total: 0n,
        delegated: 0n,
        unbonding: 0n,
        available: 0n,
        delegations: [],
        unbondingEntries: [],
      },
      createdAt: 0,
      lastActiveAt: cp.lastActiveAt,
      profileVersion: 0,
    };
  });

  const matches = matchAgentsToTask(task, profiles as any);
  const limit = Math.min(params.limit ?? 20, 100);

  return {
    matches: matches
      .slice(0, limit)
      .map(
        (m: {
          did: string;
          score: number;
          matchedCapabilities: string[];
          reputationScore: number;
        }) => ({
          did: m.did,
          score: m.score,
          matchedCapabilities: m.matchedCapabilities,
          reputationScore: m.reputationScore,
        }),
      ),
    total: matches.length,
  };
};

// ============ Combined Handlers ============

// Individual handlers are fully typed; the dispatch map uses `any` due to Handler contravariance
export const treeHandlers: Record<string, Handler<any, any>> = {
  // Read
  tree_getProject: treeGetProject,
  tree_listProjects: treeListProjects,
  tree_getService: treeGetService,
  tree_resolveSubdomain: treeResolveSubdomain,
  tree_listServices: treeListServices,
  tree_getOpportunities: treeGetOpportunities,
  tree_getProjectRevenue: treeGetProjectRevenue,
  tree_getDomainProposal: treeGetDomainProposal,
  tree_listDomainProposals: treeListDomainProposals,
  tree_getDomainTree: treeGetDomainTree,
  // Write
  tree_proposeProject: treeProposeProject,
  tree_addContributor: treeAddContributor,
  tree_claimTask: treeClaimTask,
  tree_submitDeliverable: treeSubmitDeliverable,
  tree_callService: treeCallService,
  tree_deployService: treeDeployService,
  tree_pauseService: treePauseService,
  tree_resumeService: treeResumeService,
  tree_retireService: treeRetireService,
  // Recruitment
  tree_claimOpportunity: treeClaimOpportunity,
  tree_applyToProject: treeApplyToProject,
  tree_reviewApplication: treeReviewApplication,
  tree_setAvailability: treeSetAvailability,
  tree_getProjectApplications: treeGetProjectApplications,
  tree_getAvailableAgents: treeGetAvailableAgents,
  tree_getAgentReputation: treeGetAgentReputation,
  tree_matchAgentsToTask: treeMatchAgentsToTask,
};
