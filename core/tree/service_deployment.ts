/**
 * Service Deployment
 *
 * Manages the deployment, operation, and lifecycle of services (Leaves)
 * in the Tree of Life. Services are SelfInvokingContracts with pricing
 * and operational metrics.
 */

import { domainHash } from "../crypto/hashing.js";
import type { ContractABI } from "../contracts/types.js";
import type {
  ServiceLeaf,
  ServiceType,
  ServiceEndpoint,
  PricingModel,
} from "./types.js";

// ============ Service Creation ============

/**
 * Create a new service leaf for deployment.
 */
export function createServiceLeaf(
  projectId: string,
  name: string,
  description: string,
  serviceType: ServiceType,
  endpoint: ServiceEndpoint,
  abi: ContractABI,
  pricing: PricingModel,
  contractAddress: string,
  nonce: bigint,
  block: bigint,
): ServiceLeaf {
  const id = generateServiceId(projectId, nonce);

  return {
    id,
    projectId,
    name,
    description,
    serviceType,
    endpoint,
    abi,
    pricing,
    contractAddress,
    totalCalls: 0n,
    totalRevenue: 0n,
    uptimeBlocks: 0n,
    averageLatencyMs: 0,
    status: "deploying",
    deployedAtBlock: block,
  };
}

/**
 * Activate a service after successful deployment.
 */
export function activateService(
  service: ServiceLeaf,
  healthCheckScheduleId?: string,
): ServiceLeaf {
  if (service.status !== "deploying") {
    throw new ServiceError(
      `Cannot activate: service is '${service.status}', expected 'deploying'`,
      "INVALID_STATUS",
    );
  }

  return {
    ...service,
    status: "active",
    healthCheckScheduleId,
  };
}

// ============ Service Operations ============

/**
 * Calculate the cost of a service call based on pricing model.
 */
export function calculateCallCost(service: ServiceLeaf, units: bigint): bigint {
  const { pricing } = service;

  switch (pricing.type) {
    case "per_call":
      return pricing.basePrice;

    case "subscription":
      // Subscription is pre-paid, individual calls are free
      return 0n;

    case "tiered": {
      if (!pricing.tiers || pricing.tiers.length === 0) {
        return pricing.basePrice * units;
      }

      // Find applicable tier based on cumulative usage
      const totalUsage = service.totalCalls + units;
      let applicableTier = pricing.tiers[0];
      for (const tier of pricing.tiers) {
        if (totalUsage >= tier.threshold) {
          applicableTier = tier;
        }
      }
      return applicableTier.pricePerUnit * units;
    }

    case "dynamic":
      // Dynamic pricing uses base price as starting point
      // Actual price determined by state factors (handled by caller)
      return pricing.basePrice * units;

    default:
      return pricing.basePrice;
  }
}

/**
 * Check if a caller can afford a service call.
 */
export function isCallAffordable(
  service: ServiceLeaf,
  units: bigint,
  callerBalance: bigint,
): boolean {
  const cost = calculateCallCost(service, units);
  return callerBalance >= cost;
}

/**
 * Record a successful service call (update metrics).
 */
export function recordServiceCall(
  service: ServiceLeaf,
  revenue: bigint,
  latencyMs: number,
  block: bigint,
): ServiceLeaf {
  const newTotalCalls = service.totalCalls + 1n;
  // Rolling average latency
  const avgLatency =
    service.totalCalls === 0n
      ? latencyMs
      : (service.averageLatencyMs * Number(service.totalCalls) + latencyMs) /
        Number(newTotalCalls);

  return {
    ...service,
    totalCalls: newTotalCalls,
    totalRevenue: service.totalRevenue + revenue,
    lastCalledBlock: block,
    averageLatencyMs: avgLatency,
  };
}

/**
 * Update service uptime (called each block the service is active).
 */
export function updateUptime(service: ServiceLeaf): ServiceLeaf {
  if (service.status !== "active") return service;

  return {
    ...service,
    uptimeBlocks: service.uptimeBlocks + 1n,
  };
}

// ============ Status Management ============

/**
 * Pause a service.
 */
export function pauseService(service: ServiceLeaf): ServiceLeaf {
  if (service.status !== "active" && service.status !== "degraded") {
    throw new ServiceError(
      `Cannot pause: service is '${service.status}'`,
      "INVALID_STATUS",
    );
  }

  return {
    ...service,
    status: "paused",
  };
}

/**
 * Resume a paused service.
 */
export function resumeService(service: ServiceLeaf): ServiceLeaf {
  if (service.status !== "paused") {
    throw new ServiceError(
      `Cannot resume: service is '${service.status}', expected 'paused'`,
      "INVALID_STATUS",
    );
  }

  return {
    ...service,
    status: "active",
  };
}

/**
 * Mark a service as degraded (health check failed).
 */
export function markDegraded(service: ServiceLeaf): ServiceLeaf {
  if (service.status !== "active") return service;

  return {
    ...service,
    status: "degraded",
  };
}

/**
 * Retire a service permanently.
 */
export function retireService(service: ServiceLeaf): ServiceLeaf {
  if (service.status === "retired") {
    throw new ServiceError("Service is already retired", "INVALID_STATUS");
  }

  return {
    ...service,
    status: "retired",
  };
}

/**
 * Check if a service can accept calls.
 */
export function isServiceCallable(service: ServiceLeaf): boolean {
  return service.status === "active";
}

// ============ Helpers ============

/**
 * Generate deterministic service ID
 */
export function generateServiceId(projectId: string, nonce: bigint): string {
  const input = `${projectId}:service:${nonce}`;
  return Buffer.from(
    domainHash("LGM.tree.service.id.v1", input).slice(0, 16),
  ).toString("hex");
}

// ============ Error Types ============

export type ServiceErrorCode =
  | "INVALID_STATUS"
  | "NOT_CALLABLE"
  | "INSUFFICIENT_PAYMENT";

export class ServiceError extends Error {
  constructor(
    message: string,
    public code: ServiceErrorCode,
  ) {
    super(message);
    this.name = "ServiceError";
  }
}
