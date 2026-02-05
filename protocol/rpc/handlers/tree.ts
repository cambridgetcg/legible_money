/**
 * Tree of Life RPC Handlers
 *
 * Endpoints for product project lifecycle, service management,
 * and opportunity detection.
 *
 * Endpoints:
 * - tree_getProject: Get project details
 * - tree_listProjects: List projects by phase/domain
 * - tree_getService: Get service details
 * - tree_listServices: List services by type
 * - tree_getOpportunities: List detected opportunity seeds
 * - tree_getProjectRevenue: Revenue summary for a project
 */

import type { Handler } from "../handlers.js";

// ============ Request/Response Types ============

export interface TreeGetProjectRequest {
  projectId: string;
}

export interface TreeGetProjectResponse {
  project: Record<string, unknown> | null;
}

export interface TreeListProjectsRequest {
  phase?: string;
  domain?: string;
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

export interface TreeListServicesRequest {
  projectId?: string;
  status?: string;
  limit?: number;
}

export interface TreeListServicesResponse {
  services: Record<string, unknown>[];
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

// ============ Serialization Helpers ============

function serializeBigints(
  obj: Record<string, unknown>,
): Record<string, unknown> {
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

// ============ Handlers ============

export const treeGetProject: Handler<
  TreeGetProjectRequest,
  TreeGetProjectResponse
> = async (params, ctx) => {
  const project = ctx.state.getProject(params.projectId);

  return {
    project: project
      ? serializeBigints(project as unknown as Record<string, unknown>)
      : null,
  };
};

export const treeListProjects: Handler<
  TreeListProjectsRequest,
  TreeListProjectsResponse
> = async (params, ctx) => {
  let projects;

  if (params.phase) {
    projects = ctx.state.getProjectsByPhase(params.phase as any);
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
    ] as const;
    projects = [];
    for (const phase of phases) {
      projects.push(...ctx.state.getProjectsByPhase(phase));
    }
  }

  const limit = params.limit ?? 50;
  const limited = projects.slice(0, limit);

  return {
    projects: limited.map((p) =>
      serializeBigints(p as unknown as Record<string, unknown>),
    ),
    total: projects.length,
  };
};

export const treeGetService: Handler<
  TreeGetServiceRequest,
  TreeGetServiceResponse
> = async (params, ctx) => {
  const service = ctx.state.getService(params.serviceId);

  return {
    service: service
      ? serializeBigints(service as unknown as Record<string, unknown>)
      : null,
  };
};

export const treeListServices: Handler<
  TreeListServicesRequest,
  TreeListServicesResponse
> = async (params, ctx) => {
  let services;

  if (params.status === "active") {
    services = ctx.state.getActiveServices();
  } else {
    services = ctx.state.getActiveServices(); // Default to active
  }

  const limit = params.limit ?? 50;
  const limited = services.slice(0, limit);

  return {
    services: limited.map((s) =>
      serializeBigints(s as unknown as Record<string, unknown>),
    ),
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

// ============ Combined Handlers ============

export const treeHandlers: Record<string, Handler<any, any>> = {
  tree_getProject: treeGetProject,
  tree_listProjects: treeListProjects,
  tree_getService: treeGetService,
  tree_listServices: treeListServices,
  tree_getProjectRevenue: treeGetProjectRevenue,
};
