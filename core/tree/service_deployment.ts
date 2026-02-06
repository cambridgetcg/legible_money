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
import {
  SERVICE_BASE_DOMAIN,
  SERVICE_SUBDOMAIN_SHORT_ID_LEN,
} from "../constants.js";

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
  const subdomain = generateSubdomain(id, name);

  return {
    id,
    projectId,
    name,
    description,
    serviceType,
    endpoint,
    subdomain,
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
 * Optional state resolver for dynamic pricing factors.
 * Returns a multiplier (basis points, 10000 = 1.0×) for each factor key.
 */
export type DynamicPriceResolver = (factorKey: string) => bigint;

/**
 * Calculate the cost of a service call based on pricing model.
 *
 * For tiered pricing, uses marginal pricing: units that cross tier boundaries
 * are split across tiers at each tier's rate.
 *
 * For dynamic pricing, applies state-based factor multipliers via an optional
 * resolver callback. Without a resolver, falls back to basePrice.
 *
 * @param service - The service being called
 * @param units  - Number of units consumed (interpretation depends on pricing.unit)
 * @param dynamicResolver - Optional resolver for dynamic pricing state factors
 */
export function calculateCallCost(
  service: ServiceLeaf,
  units: bigint,
  dynamicResolver?: DynamicPriceResolver,
): bigint {
  const { pricing } = service;

  if (units <= 0n) return 0n;

  switch (pricing.type) {
    case "per_call":
      return pricing.basePrice * units;

    case "subscription":
      // Subscription is pre-paid; individual calls are free for active subscribers.
      // Subscription validity is enforced by the transaction handler, not here.
      return 0n;

    case "tiered":
      return calculateTieredCost(pricing, service.totalCalls, units);

    case "dynamic":
      return calculateDynamicCost(pricing, units, dynamicResolver);

    default:
      return pricing.basePrice * units;
  }
}

/**
 * Marginal tiered pricing: splits units across tier boundaries.
 *
 * Example: tiers at [0→100/unit, 100→80/unit], prior usage=50, units=60
 *   First 50 units at 100/unit (filling tier 1 up to threshold 100): 5000
 *   Remaining 10 units at 80/unit (tier 2): 800
 *   Total: 5800
 *
 * Tiers must be sorted ascending by threshold (enforced by validatePricingModel).
 */
function calculateTieredCost(
  pricing: PricingModel,
  priorUsage: bigint,
  units: bigint,
): bigint {
  if (!pricing.tiers || pricing.tiers.length === 0) {
    return pricing.basePrice * units;
  }

  const tiers = pricing.tiers;
  let remaining = units;
  let totalCost = 0n;
  let currentUsage = priorUsage;

  for (let i = 0; i < tiers.length && remaining > 0n; i++) {
    const tierRate = tiers[i].pricePerUnit;
    const nextThreshold = i + 1 < tiers.length ? tiers[i + 1].threshold : null;

    // How many units can be billed at this tier's rate?
    let unitsAtTier: bigint;
    if (nextThreshold !== null && currentUsage < nextThreshold) {
      // Units until next tier boundary
      const untilNext = nextThreshold - currentUsage;
      unitsAtTier = remaining < untilNext ? remaining : untilNext;
    } else if (nextThreshold !== null && currentUsage >= nextThreshold) {
      // Already past this tier — skip to the next
      continue;
    } else {
      // Last tier — all remaining units
      unitsAtTier = remaining;
    }

    totalCost += tierRate * unitsAtTier;
    currentUsage += unitsAtTier;
    remaining -= unitsAtTier;
  }

  return totalCost;
}

/**
 * Dynamic pricing: applies state-based multipliers from dynamicFactors.
 *
 * Each factor key is resolved to a multiplier in basis points (10000 = 1.0×).
 * All factor multipliers are combined multiplicatively.
 * Without a resolver or factors, falls back to basePrice × units.
 */
function calculateDynamicCost(
  pricing: PricingModel,
  units: bigint,
  resolver?: DynamicPriceResolver,
): bigint {
  let cost = pricing.basePrice * units;

  if (
    !resolver ||
    !pricing.dynamicFactors ||
    pricing.dynamicFactors.length === 0
  ) {
    return cost;
  }

  // Apply each factor's multiplier (in basis points)
  for (const factorKey of pricing.dynamicFactors) {
    const multiplierBP = resolver(factorKey);
    cost = (cost * multiplierBP) / 10000n;
  }

  return cost;
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

  // Rolling average latency using exponential moving average (EMA) for
  // high-call-count services to avoid Number(bigint) precision loss above 2^53.
  // Alpha = 1/min(n, 1000) — window caps at 1000 for numerical stability.
  let avgLatency: number;
  if (service.totalCalls === 0n) {
    avgLatency = latencyMs;
  } else {
    const window =
      service.totalCalls < 1000n ? Number(service.totalCalls) : 1000;
    const alpha = 1 / (window + 1);
    avgLatency = service.averageLatencyMs * (1 - alpha) + latencyMs * alpha;
  }

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
  if (service.status !== "active") {
    throw new ServiceError(
      `Cannot mark degraded: service is '${service.status}', expected 'active'`,
      "INVALID_STATUS",
    );
  }

  return {
    ...service,
    status: "degraded",
  };
}

/**
 * Recover a degraded service back to active (health check passed).
 */
export function recoverService(service: ServiceLeaf): ServiceLeaf {
  if (service.status !== "degraded") {
    throw new ServiceError(
      `Cannot recover: service is '${service.status}', expected 'degraded'`,
      "INVALID_STATUS",
    );
  }

  return {
    ...service,
    status: "active",
  };
}

/**
 * Retire a service permanently.
 */
export function retireService(service: ServiceLeaf): ServiceLeaf {
  if (service.status === "retired") {
    throw new ServiceError("Service is already retired", "INVALID_STATUS");
  }
  if (service.status === "deploying") {
    throw new ServiceError(
      "Cannot retire a service that was never activated",
      "INVALID_STATUS",
    );
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

// ============ Pricing Validation ============

/**
 * Validate a PricingModel before service deployment.
 * Throws ServiceError if the model is malformed.
 */
export function validatePricingModel(pricing: PricingModel): void {
  if (pricing.basePrice < 0n) {
    throw new ServiceError("Base price cannot be negative", "INVALID_PRICING");
  }

  switch (pricing.type) {
    case "subscription":
      if (!pricing.subscriptionBlocks || pricing.subscriptionBlocks <= 0n) {
        throw new ServiceError(
          "Subscription pricing requires subscriptionBlocks > 0",
          "INVALID_PRICING",
        );
      }
      if (pricing.basePrice === 0n) {
        throw new ServiceError(
          "Subscription pricing requires basePrice > 0 (subscription fee)",
          "INVALID_PRICING",
        );
      }
      break;

    case "tiered":
      if (!pricing.tiers || pricing.tiers.length === 0) {
        throw new ServiceError(
          "Tiered pricing requires at least one tier",
          "INVALID_PRICING",
        );
      }
      // Validate tiers are sorted ascending by threshold
      for (let i = 1; i < pricing.tiers.length; i++) {
        if (pricing.tiers[i].threshold <= pricing.tiers[i - 1].threshold) {
          throw new ServiceError(
            "Pricing tiers must be sorted ascending by threshold",
            "INVALID_PRICING",
          );
        }
      }
      // First tier must start at 0
      if (pricing.tiers[0].threshold !== 0n) {
        throw new ServiceError(
          "First pricing tier must have threshold 0",
          "INVALID_PRICING",
        );
      }
      // No negative tier prices
      for (const tier of pricing.tiers) {
        if (tier.pricePerUnit < 0n) {
          throw new ServiceError(
            `Tier '${tier.label}' has negative pricePerUnit`,
            "INVALID_PRICING",
          );
        }
      }
      break;

    case "dynamic":
      if (!pricing.dynamicFactors || pricing.dynamicFactors.length === 0) {
        throw new ServiceError(
          "Dynamic pricing requires at least one dynamic factor key",
          "INVALID_PRICING",
        );
      }
      break;

    case "per_call":
      // per_call is valid with any non-negative basePrice
      break;
  }
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

// ============ Subdomain Allocation ============

/**
 * Slugify a service name for DNS-safe subdomain use.
 * Lowercases, replaces non-alphanumeric with hyphens, collapses, strips edges.
 */
export function slugifyServiceName(
  name: string,
  maxLength: number = 54,
): string {
  return name
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "")
    .slice(0, maxLength)
    .replace(/-+$/, "");
}

/**
 * Generate a deterministic subdomain for a service.
 *
 * Format: {name-slug}-{short-id}.{baseDomain}
 * Total label ≤ 63 chars (DNS RFC 1035).
 */
export function generateSubdomain(
  serviceId: string,
  serviceName: string,
  baseDomain?: string,
): string {
  const domain = baseDomain ?? SERVICE_BASE_DOMAIN;
  const shortId = serviceId.slice(0, SERVICE_SUBDOMAIN_SHORT_ID_LEN);
  const slug = slugifyServiceName(serviceName);
  const label = slug ? `${slug}-${shortId}` : shortId;
  return `${label}.${domain}`;
}

// ============ Health Check Schedule ============

/** Default health check interval: ~60s at 2521ms block time (24 blocks) */
export const HEALTH_CHECK_INTERVAL_BLOCKS = 24;

/**
 * Create a ScheduleProcess for periodic service health checks.
 * The schedule monitors the service and can trigger degradation/pause
 * if the service becomes unhealthy.
 */
export function createHealthCheckSchedule(
  service: ServiceLeaf,
  founderDid: string,
  blockTimestampMs: number,
): import("../schedule/types.js").ScheduleProcess {
  const intervalMs = HEALTH_CHECK_INTERVAL_BLOCKS * 2521;
  const scheduleId = Buffer.from(
    domainHash("LGM.tree.service.healthcheck.v1", service.id).slice(0, 16),
  ).toString("hex");

  return {
    id: scheduleId,
    sourceMessageId: `healthcheck:${service.id}`,
    creator: founderDid,
    condition: {
      time: {
        type: "periodic",
        interval: intervalMs,
        startTime: blockTimestampMs + intervalMs,
      },
    },
    nextCheckTime: blockTimestampMs + intervalMs,
    watchedStateKeys: [`service:${service.id}:status`],
    status: "active",
    executionCount: 0,
    remainingFee: 0n, // Health checks are fee-exempt (protocol operation)
    feePerExecution: 0n,
    targetAddress: service.contractAddress,
    callData: new Uint8Array(0), // Health check uses service endpoint directly
    transferValue: 0n,
    generatedTxNonce: 0n,
    knowledgeDomain: undefined,
    linkedEntityType: "service",
    linkedEntityId: service.id,
    createdAt: blockTimestampMs,
    modifiedAt: blockTimestampMs,
  };
}

// ============ Error Types ============

export type ServiceErrorCode =
  | "INVALID_STATUS"
  | "INVALID_PRICING"
  | "NOT_CALLABLE"
  | "INSUFFICIENT_PAYMENT"
  | "SUBSCRIPTION_REQUIRED";

export class ServiceError extends Error {
  constructor(
    message: string,
    public code: ServiceErrorCode,
  ) {
    super(message);
    this.name = "ServiceError";
  }
}
