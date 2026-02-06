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
  /** Request timeout in ms (default 30000) */
  timeout?: number;
  /** Max retry attempts (default 3) */
  retries?: number;
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

export interface OpportunitySummary {
  id: string;
  knowledgeDomain: string;
  confidence: number;
  estimatedDemand: string;
  estimatedEffort: number;
  status: string;
}

export interface TxResult {
  txHash: string;
  success: boolean;
  error?: string;
}

// ============ Tree Client ============

/**
 * SDK client for Tree of Life operations.
 * Wraps RPC calls with wallet signing and type-safe responses.
 */
export class TreeClient {
  private rpcUrl: string;
  private _wallet: AgentWallet;
  private timeout: number;
  private retries: number;
  private requestId: number = 0;

  constructor(config: TreeClientConfig) {
    this.rpcUrl = config.rpcUrl;
    this._wallet = config.wallet;
    this.timeout = config.timeout ?? 30000;
    this.retries = config.retries ?? 3;
  }

  /** Get the wallet associated with this client */
  get wallet(): AgentWallet {
    return this._wallet;
  }

  // ============ Projects (Read) ============

  /**
   * Get project details by ID
   */
  async getProject(projectId: string): Promise<ProjectSummary | null> {
    const result = await this.rpcCall("tree_getProject", { projectId });
    if (!result.project) return null;
    return toProjectSummary(result.project as Record<string, unknown>);
  }

  /**
   * List projects by phase and/or domain
   */
  async listProjects(options?: {
    phase?: string;
    domain?: string;
    offset?: number;
    limit?: number;
  }): Promise<ProjectSummary[]> {
    const result = await this.rpcCall("tree_listProjects", options ?? {});
    return ((result.projects as Record<string, unknown>[]) ?? []).map(
      toProjectSummary,
    );
  }

  // ============ Projects (Write) ============

  /**
   * Propose a new project in a knowledge domain.
   * Creates the project and transitions it to "sprout" phase.
   */
  async proposeProject(options: {
    name: string;
    description: string;
    knowledgeDomain: string;
    productSpec: Record<string, unknown>;
    fundingSource: Record<string, unknown>;
    demandSignal: Record<string, unknown>;
    nonce: string;
    fee?: string;
  }): Promise<TxResult> {
    return this.signedRpcCall("tree_proposeProject", {
      name: options.name,
      description: options.description,
      knowledgeDomain: options.knowledgeDomain,
      productSpec: options.productSpec,
      fundingSource: options.fundingSource,
      demandSignal: options.demandSignal,
      nonce: options.nonce,
      fee: options.fee ?? "1000000000000000",
    });
  }

  /**
   * Add a contributor to a project.
   * Caller must be the project founder or an existing contributor.
   */
  async addContributor(options: {
    projectId: string;
    contributorDid: string;
    role: string;
    nonce: string;
    fee?: string;
  }): Promise<TxResult> {
    return this.signedRpcCall("tree_addContributor", {
      projectId: options.projectId,
      contributorDid: options.contributorDid,
      role: options.role,
      nonce: options.nonce,
      fee: options.fee ?? "1000000000000000",
    });
  }

  // ============ Tasks ============

  /**
   * Claim (assign self to) a task on a project.
   */
  async claimTask(options: {
    projectId: string;
    taskId: string;
    nonce: string;
    fee?: string;
  }): Promise<TxResult> {
    return this.signedRpcCall("tree_claimTask", {
      projectId: options.projectId,
      taskId: options.taskId,
      nonce: options.nonce,
      fee: options.fee ?? "1000000000000000",
    });
  }

  /**
   * Submit a deliverable for a task.
   */
  async submitDeliverable(options: {
    projectId: string;
    taskId: string;
    deliverable: Record<string, unknown>;
    nonce: string;
    fee?: string;
  }): Promise<TxResult> {
    return this.signedRpcCall("tree_submitDeliverable", {
      projectId: options.projectId,
      taskId: options.taskId,
      deliverable: options.deliverable,
      nonce: options.nonce,
      fee: options.fee ?? "1000000000000000",
    });
  }

  // ============ Services (Read) ============

  /**
   * Get service details by ID
   */
  async getService(serviceId: string): Promise<ServiceSummary | null> {
    const result = await this.rpcCall("tree_getService", { serviceId });
    if (!result.service) return null;
    return toServiceSummary(result.service as Record<string, unknown>);
  }

  /**
   * List services, optionally filtered by status
   */
  async listServices(options?: {
    status?: string;
    offset?: number;
    limit?: number;
  }): Promise<ServiceSummary[]> {
    const result = await this.rpcCall("tree_listServices", {
      status: options?.status ?? "active",
      offset: options?.offset,
      limit: options?.limit,
    });
    return ((result.services as Record<string, unknown>[]) ?? []).map(
      toServiceSummary,
    );
  }

  // ============ Services (Write) ============

  /**
   * Deploy a service for a project.
   * Only the project founder can deploy. Transitions project to fruiting if mature.
   */
  async deployService(options: {
    projectId: string;
    name: string;
    description: string;
    serviceType: string;
    endpoint: Record<string, unknown>;
    abi: Record<string, unknown>;
    pricing: Record<string, unknown>;
    contractAddress: string;
    nonce: string;
    fee?: string;
  }): Promise<TxResult> {
    return this.signedRpcCall("tree_deployService", {
      projectId: options.projectId,
      name: options.name,
      description: options.description,
      serviceType: options.serviceType,
      endpoint: options.endpoint,
      abi: options.abi,
      pricing: options.pricing,
      contractAddress: options.contractAddress,
      nonce: options.nonce,
      fee: options.fee ?? "1000000000000000",
    });
  }

  /**
   * Call a deployed service. Deducts LGM payment and distributes revenue.
   * This is the primary revenue-generating operation in the Tree of Life.
   */
  async callService(options: {
    serviceId: string;
    units: string;
    nonce: string;
    fee?: string;
  }): Promise<TxResult> {
    return this.signedRpcCall("tree_callService", {
      serviceId: options.serviceId,
      units: options.units,
      nonce: options.nonce,
      fee: options.fee ?? "1000000000000000",
    });
  }

  /**
   * Pause an active service. Only the project founder can pause.
   */
  async pauseService(options: {
    serviceId: string;
    nonce: string;
    fee?: string;
  }): Promise<TxResult> {
    return this.signedRpcCall("tree_pauseService", {
      serviceId: options.serviceId,
      nonce: options.nonce,
      fee: options.fee ?? "1000000000000000",
    });
  }

  /**
   * Resume a paused service. Only the project founder can resume.
   */
  async resumeService(options: {
    serviceId: string;
    nonce: string;
    fee?: string;
  }): Promise<TxResult> {
    return this.signedRpcCall("tree_resumeService", {
      serviceId: options.serviceId,
      nonce: options.nonce,
      fee: options.fee ?? "1000000000000000",
    });
  }

  /**
   * Retire a service permanently. Only the project founder can retire.
   */
  async retireService(options: {
    serviceId: string;
    nonce: string;
    fee?: string;
  }): Promise<TxResult> {
    return this.signedRpcCall("tree_retireService", {
      serviceId: options.serviceId,
      nonce: options.nonce,
      fee: options.fee ?? "1000000000000000",
    });
  }

  // ============ Opportunities ============

  /**
   * List detected opportunity seeds
   */
  async getOpportunities(options?: {
    domain?: string;
    minConfidence?: number;
    offset?: number;
    limit?: number;
  }): Promise<OpportunitySummary[]> {
    const result = await this.rpcCall("tree_getOpportunities", options ?? {});
    return ((result.opportunities as Record<string, unknown>[]) ?? []).map(
      (s) => ({
        id: s.id as string,
        knowledgeDomain: s.knowledgeDomain as string,
        confidence: s.confidence as number,
        estimatedDemand: (s.estimatedDemand as string) ?? "0",
        estimatedEffort: (s.estimatedEffort as number) ?? 0,
        status: s.status as string,
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
    return {
      totalRevenue: (result.totalRevenue as string) ?? "0",
      contributorCount: (result.contributorCount as number) ?? 0,
    };
  }

  // ============ Signed RPC Helper ============

  /**
   * Build a signed transaction and submit via RPC.
   * Signs the payload with the wallet and includes sender public key.
   */
  private async signedRpcCall(
    method: string,
    params: Record<string, unknown>,
  ): Promise<TxResult> {
    const signature = await this._wallet.signTransaction(params);
    const result = await this.rpcCall(method, {
      ...params,
      signature,
      sender: this._wallet.publicKey,
    });
    return {
      txHash: (result.txHash as string) ?? "",
      success: (result.success as boolean) ?? false,
      error: result.error as string | undefined,
    };
  }

  // ============ RPC Helper ============

  private async rpcCall(
    method: string,
    params: Record<string, unknown>,
  ): Promise<Record<string, unknown>> {
    const request = JSON.stringify({
      jsonrpc: "2.0",
      id: ++this.requestId,
      method,
      params,
    });

    let lastError: Error | null = null;

    for (let attempt = 0; attempt < this.retries; attempt++) {
      try {
        const response = await fetch(this.rpcUrl, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: request,
          signal: AbortSignal.timeout(this.timeout),
        });

        const json = (await response.json()) as {
          result?: Record<string, unknown>;
          error?: {
            message: string;
            data?: { retryable?: boolean; retryAfter?: number };
          };
        };

        if (json.error) {
          const retryable = json.error.data?.retryable ?? false;
          if (!retryable) {
            throw new Error(`RPC error: ${json.error.message}`);
          }
          lastError = new Error(`RPC error: ${json.error.message}`);
          await sleep(json.error.data?.retryAfter ?? 1000 * 2 ** attempt);
          continue;
        }

        return json.result ?? {};
      } catch (error) {
        if (error instanceof Error && error.message.startsWith("RPC error:")) {
          throw error;
        }
        lastError = error as Error;
        if (attempt < this.retries - 1) {
          await sleep(1000 * 2 ** attempt);
        }
      }
    }

    throw lastError ?? new Error("Request failed after retries");
  }
}

// ============ Mapping Helpers ============

function toProjectSummary(p: Record<string, unknown>): ProjectSummary {
  return {
    id: p.id as string,
    name: p.name as string,
    phase: p.phase as string,
    domain: (p.knowledgeDomain as string) ?? "",
    contributorCount: Array.isArray(p.contributors) ? p.contributors.length : 0,
    totalTasks: (p.totalTasks as number) ?? 0,
    completedTasks: (p.completedTasks as number) ?? 0,
    revenueGenerated: (p.revenueGenerated as string) ?? "0",
  };
}

function toServiceSummary(s: Record<string, unknown>): ServiceSummary {
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

function sleep(ms: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, ms));
}
