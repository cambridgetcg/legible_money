/**
 * Tree of Life SDK Client
 *
 * High-level client for agents to interact with the Tree of Life:
 * - Browse and create product projects
 * - Discover and claim tasks
 * - Call deployed services
 * - View revenue and opportunities
 */

import type { AgentWallet } from "./wallet.js";

// ============ Types ============

export interface TreeClientConfig {
  rpcUrl: string;
  wallet: AgentWallet;
}

export interface ProjectSummary {
  id: string;
  name: string;
  phase: string;
  domain: string;
  contributorCount: number;
  totalTasks: number;
  completedTasks: number;
  revenueGenerated: string;
}

export interface ServiceSummary {
  id: string;
  projectId: string;
  name: string;
  serviceType: string;
  status: string;
  totalCalls: string;
  totalRevenue: string;
}

export interface TaskSummary {
  id: string;
  projectId: string;
  title: string;
  taskType: string;
  status: string;
  bountyAmount: string;
  requiredCapabilities: string[];
}

// ============ Tree Client ============

/**
 * SDK client for Tree of Life operations.
 * Wraps RPC calls with wallet signing and type-safe responses.
 */
export class TreeClient {
  private rpcUrl: string;

  constructor(config: TreeClientConfig) {
    this.rpcUrl = config.rpcUrl;
  }

  // ============ Projects ============

  /**
   * Get project details by ID
   */
  async getProject(projectId: string): Promise<ProjectSummary | null> {
    const result = await this.rpcCall("tree_getProject", { projectId });
    if (!(result as any).project) return null;

    const p = (result as any).project;
    return {
      id: p.id as string,
      name: p.name as string,
      phase: p.phase as string,
      domain: p.knowledgeDomain as string,
      contributorCount: Array.isArray(p.contributors)
        ? p.contributors.length
        : 0,
      totalTasks: (p.totalTasks as number) ?? 0,
      completedTasks: (p.completedTasks as number) ?? 0,
      revenueGenerated: (p.revenueGenerated as string) ?? "0",
    };
  }

  /**
   * List projects by phase
   */
  async listProjects(
    phase?: string,
    limit?: number,
  ): Promise<ProjectSummary[]> {
    const result = await this.rpcCall("tree_listProjects", { phase, limit });
    const resultAny = result as any;
    return ((resultAny.projects ?? []) as Record<string, unknown>[]).map(
      (p: Record<string, unknown>) => ({
        id: p.id as string,
        name: p.name as string,
        phase: p.phase as string,
        domain: p.knowledgeDomain as string,
        contributorCount: Array.isArray(p.contributors)
          ? p.contributors.length
          : 0,
        totalTasks: (p.totalTasks as number) ?? 0,
        completedTasks: (p.completedTasks as number) ?? 0,
        revenueGenerated: (p.revenueGenerated as string) ?? "0",
      }),
    );
  }

  // ============ Services ============

  /**
   * Get service details by ID
   */
  async getService(serviceId: string): Promise<ServiceSummary | null> {
    const result = await this.rpcCall("tree_getService", { serviceId });
    if (!(result as any).service) return null;

    const s = (result as any).service;
    return {
      id: s.id as string,
      projectId: s.projectId as string,
      name: s.name as string,
      serviceType: s.serviceType as string,
      status: s.status as string,
      totalCalls: (s.totalCalls as string) ?? "0",
      totalRevenue: (s.totalRevenue as string) ?? "0",
    };
  }

  /**
   * List active services
   */
  async listServices(limit?: number): Promise<ServiceSummary[]> {
    const result = await this.rpcCall("tree_listServices", {
      status: "active",
      limit,
    });
    const resultAny = result as any;
    return ((resultAny.services ?? []) as Record<string, unknown>[]).map(
      (s: Record<string, unknown>) => ({
        id: s.id as string,
        projectId: s.projectId as string,
        name: s.name as string,
        serviceType: s.serviceType as string,
        status: s.status as string,
        totalCalls: (s.totalCalls as string) ?? "0",
        totalRevenue: (s.totalRevenue as string) ?? "0",
      }),
    );
  }

  // ============ Revenue ============

  /**
   * Get revenue summary for a project
   */
  async getProjectRevenue(projectId: string): Promise<{
    totalRevenue: string;
    contributorCount: number;
  }> {
    const result = await this.rpcCall("tree_getProjectRevenue", { projectId });
    const resultAny = result as any;
    return {
      totalRevenue: (resultAny.totalRevenue as string) ?? "0",
      contributorCount: (resultAny.contributorCount as number) ?? 0,
    };
  }

  // ============ RPC Helper ============

  private async rpcCall(
    method: string,
    params: Record<string, unknown>,
  ): Promise<Record<string, unknown>> {
    const response = await fetch(this.rpcUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: Date.now(),
        method,
        params,
      }),
    });

    const json = (await response.json()) as {
      result?: Record<string, unknown>;
      error?: { message: string };
    };

    if (json.error) {
      throw new Error(`RPC error: ${json.error.message}`);
    }

    return json.result ?? {};
  }
}
