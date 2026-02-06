/**
 * Service State Tree
 *
 * Sparse Merkle Tree for Tree of Life deployed services.
 * Key: hash(serviceId) - 32 bytes
 * Value: serialized ServiceLeaf
 *
 * Provides:
 * - Cryptographic proof of service existence
 * - Service lookup by project, type, and status
 * - Operational metric tracking (calls, revenue, uptime)
 * - Pricing model storage
 */

import {
  InMemorySMT,
  type MerkleRoot,
  type MerkleProofData,
} from "../core/crypto/merkle.js";
import { sha512_256 } from "../core/crypto/hashing.js";
import type {
  ServiceLeaf,
  ServiceType,
  ServiceStatus,
  ServiceEndpoint,
  PricingModel,
  PricingTier,
} from "../core/tree/types.js";
import type { ContractABI } from "../core/contracts/types.js";
import { StateError } from "./types.js";

// ============ Serialization ============

/**
 * Serialize ServiceLeaf to bytes for SMT storage.
 * BigInt fields are converted to strings for JSON compatibility.
 */
function serializeService(service: ServiceLeaf): Uint8Array {
  const json = JSON.stringify({
    id: service.id,
    projectId: service.projectId,
    name: service.name,
    description: service.description,
    serviceType: service.serviceType,
    endpoint: service.endpoint,
    subdomain: service.subdomain,
    abi: service.abi,
    pricing: serializePricing(service.pricing),
    contractAddress: service.contractAddress,
    healthCheckScheduleId: service.healthCheckScheduleId,
    totalCalls: service.totalCalls.toString(),
    totalRevenue: service.totalRevenue.toString(),
    uptimeBlocks: service.uptimeBlocks.toString(),
    lastCalledBlock: service.lastCalledBlock?.toString(),
    averageLatencyMs: service.averageLatencyMs,
    status: service.status,
    deployedAtBlock: service.deployedAtBlock.toString(),
  });
  return new TextEncoder().encode(json);
}

/**
 * Serialize PricingModel, converting bigint fields to strings.
 */
function serializePricing(pricing: PricingModel): Record<string, unknown> {
  return {
    type: pricing.type,
    basePrice: pricing.basePrice.toString(),
    unit: pricing.unit,
    tiers: pricing.tiers?.map((t) => ({
      threshold: t.threshold.toString(),
      pricePerUnit: t.pricePerUnit.toString(),
      label: t.label,
    })),
    subscriptionBlocks: pricing.subscriptionBlocks?.toString(),
    dynamicFactors: pricing.dynamicFactors,
  };
}

/**
 * Deserialize ServiceLeaf from bytes.
 * String fields are converted back to BigInt where needed.
 */
function deserializeService(bytes: Uint8Array): ServiceLeaf {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    id: obj.id,
    projectId: obj.projectId,
    name: obj.name,
    description: obj.description,
    serviceType: obj.serviceType as ServiceType,
    endpoint: obj.endpoint as ServiceEndpoint,
    subdomain: obj.subdomain ?? "",
    abi: obj.abi as ContractABI,
    pricing: deserializePricing(obj.pricing),
    contractAddress: obj.contractAddress,
    healthCheckScheduleId: obj.healthCheckScheduleId,
    totalCalls: BigInt(obj.totalCalls),
    totalRevenue: BigInt(obj.totalRevenue),
    uptimeBlocks: BigInt(obj.uptimeBlocks),
    lastCalledBlock: obj.lastCalledBlock
      ? BigInt(obj.lastCalledBlock)
      : undefined,
    averageLatencyMs: obj.averageLatencyMs,
    status: obj.status as ServiceStatus,
    deployedAtBlock: BigInt(obj.deployedAtBlock),
  };
}

/**
 * Deserialize PricingModel, converting string fields back to bigint.
 */
function deserializePricing(obj: Record<string, unknown>): PricingModel {
  const tiers = obj.tiers as Array<Record<string, unknown>> | undefined;
  return {
    type: obj.type as PricingModel["type"],
    basePrice: BigInt(obj.basePrice as string),
    unit: obj.unit as PricingModel["unit"],
    tiers: tiers?.map(
      (t): PricingTier => ({
        threshold: BigInt(t.threshold as string),
        pricePerUnit: BigInt(t.pricePerUnit as string),
        label: t.label as string,
      }),
    ),
    subscriptionBlocks: obj.subscriptionBlocks
      ? BigInt(obj.subscriptionBlocks as string)
      : undefined,
    dynamicFactors: obj.dynamicFactors as string[] | undefined,
  };
}

/**
 * Convert service ID to SMT key
 */
function serviceIdToKey(serviceId: string): Uint8Array {
  return sha512_256(serviceId);
}

// ============ Service State Tree ============

export class ServiceStateTree {
  private smt: InMemorySMT;
  private cache: Map<string, ServiceLeaf> = new Map();
  private subdomainIndex: Map<string, string> = new Map(); // subdomain → serviceId
  private serviceCount: number = 0;

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
   * Get service by ID.
   * WARNING: Returns a cached reference. Do NOT mutate the returned object directly.
   * Use updateService() for all modifications.
   */
  getService(serviceId: string): ServiceLeaf | null {
    const cached = this.cache.get(serviceId);
    if (cached) return cached;

    const key = serviceIdToKey(serviceId);
    const value = this.smt.get(key);
    if (!value) return null;

    const service = deserializeService(value);
    this.cache.set(serviceId, service);
    return service;
  }

  /**
   * Create a new service
   */
  createService(service: ServiceLeaf): MerkleRoot {
    if (this.getService(service.id) !== null) {
      throw new StateError("Service already exists", "INVALID_TRANSITION", {
        serviceId: service.id,
      });
    }

    const key = serviceIdToKey(service.id);
    const value = serializeService(service);
    const newRoot = this.smt.set(key, value);

    this.cache.set(service.id, service);
    this.serviceCount++;

    // Index subdomain for reverse lookup
    if (service.subdomain) {
      this.subdomainIndex.set(service.subdomain, service.id);
    }

    return newRoot;
  }

  /**
   * Update an existing service
   */
  updateService(serviceId: string, updates: Partial<ServiceLeaf>): MerkleRoot {
    const existing = this.getService(serviceId);
    if (!existing) {
      throw new StateError("Service not found", "SERVICE_NOT_FOUND", {
        serviceId,
      });
    }

    const updated: ServiceLeaf = {
      ...existing,
      ...updates,
    };

    const key = serviceIdToKey(serviceId);
    const value = serializeService(updated);
    const newRoot = this.smt.set(key, value);

    this.cache.set(serviceId, updated);
    return newRoot;
  }

  // ============ Query Operations ============

  /**
   * Get all services belonging to a project
   */
  getServicesByProject(projectId: string): ServiceLeaf[] {
    const results: ServiceLeaf[] = [];
    for (const service of this.cache.values()) {
      if (service.projectId === projectId) {
        results.push(service);
      }
    }
    return results;
  }

  /**
   * Get all services of a given type
   */
  getServicesByType(type: ServiceType): ServiceLeaf[] {
    const results: ServiceLeaf[] = [];
    for (const service of this.cache.values()) {
      if (service.serviceType === type) {
        results.push(service);
      }
    }
    return results;
  }

  /**
   * Get all active services (status === 'active')
   */
  getActiveServices(): ServiceLeaf[] {
    return this.getServicesByStatus("active");
  }

  /**
   * Get all services with a given status
   */
  getServicesByStatus(status: ServiceStatus): ServiceLeaf[] {
    const results: ServiceLeaf[] = [];
    for (const service of this.cache.values()) {
      if (service.status === status) {
        results.push(service);
      }
    }
    return results;
  }

  /**
   * Look up a service by its protocol-assigned subdomain.
   */
  getServiceBySubdomain(subdomain: string): ServiceLeaf | null {
    const serviceId = this.subdomainIndex.get(subdomain);
    if (!serviceId) return null;
    return this.getService(serviceId);
  }

  /**
   * Get total service count
   */
  getServiceCount(): number {
    return this.serviceCount;
  }

  // ============ Metrics ============

  /**
   * Increment service call metrics.
   * Updates totalCalls, totalRevenue, lastCalledBlock, and uptimeBlocks.
   */
  incrementServiceCalls(
    serviceId: string,
    revenue: bigint,
    block: bigint,
  ): MerkleRoot {
    const existing = this.getService(serviceId);
    if (!existing) {
      throw new StateError("Service not found", "SERVICE_NOT_FOUND", {
        serviceId,
      });
    }

    // Calculate uptime: blocks since deployment (or last known call) up to this block
    const uptimeIncrease =
      existing.lastCalledBlock !== undefined
        ? block - existing.lastCalledBlock
        : 0n;

    return this.updateService(serviceId, {
      totalCalls: existing.totalCalls + 1n,
      totalRevenue: existing.totalRevenue + revenue,
      lastCalledBlock: block,
      uptimeBlocks: existing.uptimeBlocks + uptimeIncrease,
    });
  }

  // ============ Proofs ============

  /**
   * Generate Merkle proof for a service.
   * Proves inclusion (or non-inclusion) in the service state tree.
   */
  proveService(serviceId: string): MerkleProofData {
    const key = serviceIdToKey(serviceId);
    return this.smt.prove(key);
  }

  // ============ Persistence Export/Restore ============

  /**
   * Export all services for persistence
   */
  exportServices(): ServiceLeaf[] {
    return Array.from(this.cache.values());
  }

  // ============ Utilities ============

  /**
   * Clone for speculative execution
   */
  clone(): ServiceStateTree {
    const cloned = new ServiceStateTree();

    for (const [serviceId, service] of this.cache) {
      // Deep clone the service — spread top-level, clone nested objects/arrays
      const clonedService: ServiceLeaf = {
        ...service,
        endpoint: {
          ...service.endpoint,
          methods: [...service.endpoint.methods],
        },
        abi: JSON.parse(JSON.stringify(service.abi)),
        pricing: {
          ...service.pricing,
          tiers: service.pricing.tiers?.map((t) => ({ ...t })),
          dynamicFactors: service.pricing.dynamicFactors
            ? [...service.pricing.dynamicFactors]
            : undefined,
        },
      };

      cloned.cache.set(serviceId, clonedService);
      const key = serviceIdToKey(serviceId);
      cloned.smt.set(key, serializeService(clonedService));
    }

    cloned.serviceCount = this.serviceCount;
    cloned.subdomainIndex = new Map(this.subdomainIndex);
    return cloned;
  }
}
