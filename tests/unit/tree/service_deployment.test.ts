/**
 * Service Deployment Tests
 *
 * Tests for service lifecycle: creation, activation, call cost, metrics, status transitions.
 */

import { describe, it, expect } from "vitest";
import {
  createServiceLeaf,
  activateService,
  calculateCallCost,
  isCallAffordable,
  recordServiceCall,
  updateUptime,
  pauseService,
  resumeService,
  markDegraded,
  retireService,
  isServiceCallable,
  generateServiceId,
  ServiceError,
} from "../../../core/tree/service_deployment.js";
import type { ServiceLeaf, PricingModel } from "../../../core/tree/types.js";
import type { ContractABI } from "../../../core/contracts/types.js";

// ============ Test Helpers ============

const MOCK_ABI: ContractABI = {
  version: "1.0.0",
  methods: [
    {
      name: "query",
      inputs: [{ name: "data", type: "bytes" }],
      outputs: [{ name: "result", type: "bytes" }],
    },
  ],
  events: [],
};

function makeService(overrides: Partial<ServiceLeaf> = {}): ServiceLeaf {
  return createServiceLeaf(
    "proj-1",
    "Test Service",
    "A test service",
    "api",
    { url: "https://example.com/api", protocol: "https" },
    MOCK_ABI,
    { type: "per_call", basePrice: 1000n, unit: "call" },
    "contract-addr-1",
    0n,
    100n,
    ...(Object.keys(overrides).length ? [] : []),
  );
}

function makeActiveService(): ServiceLeaf {
  return activateService(makeService());
}

// ============ createServiceLeaf ============

describe("createServiceLeaf", () => {
  it("should create a service in deploying status", () => {
    const service = createServiceLeaf(
      "proj-1",
      "API Service",
      "REST API",
      "api",
      { url: "https://api.example.com", protocol: "https" },
      MOCK_ABI,
      { type: "per_call", basePrice: 500n, unit: "call" },
      "contract-1",
      0n,
      100n,
    );

    expect(service.id).toBeTruthy();
    expect(service.projectId).toBe("proj-1");
    expect(service.name).toBe("API Service");
    expect(service.serviceType).toBe("api");
    expect(service.status).toBe("deploying");
    expect(service.totalCalls).toBe(0n);
    expect(service.totalRevenue).toBe(0n);
    expect(service.deployedAtBlock).toBe(100n);
  });

  it("should generate deterministic IDs", () => {
    const s1 = createServiceLeaf(
      "proj-1",
      "A",
      "",
      "api",
      { url: "", protocol: "" },
      MOCK_ABI,
      { type: "per_call", basePrice: 0n, unit: "call" },
      "",
      0n,
      0n,
    );
    const s2 = createServiceLeaf(
      "proj-1",
      "B",
      "",
      "api",
      { url: "", protocol: "" },
      MOCK_ABI,
      { type: "per_call", basePrice: 0n, unit: "call" },
      "",
      0n,
      0n,
    );
    const s3 = createServiceLeaf(
      "proj-1",
      "A",
      "",
      "api",
      { url: "", protocol: "" },
      MOCK_ABI,
      { type: "per_call", basePrice: 0n, unit: "call" },
      "",
      1n,
      0n,
    );

    expect(s1.id).toBe(s2.id); // Same nonce → same ID
    expect(s1.id).not.toBe(s3.id); // Different nonce → different ID
  });
});

// ============ activateService ============

describe("activateService", () => {
  it("should activate a deploying service", () => {
    const service = makeService();
    const active = activateService(service);

    expect(active.status).toBe("active");
  });

  it("should reject activating non-deploying service", () => {
    const active = makeActiveService();

    expect(() => activateService(active)).toThrow("deploying");
  });
});

// ============ calculateCallCost ============

describe("calculateCallCost", () => {
  it("should return base price for per_call pricing", () => {
    const service = makeActiveService();
    const cost = calculateCallCost(service, 1n);

    expect(cost).toBe(1000n); // basePrice from makeService
  });

  it("should return 0 for subscription pricing", () => {
    const service: ServiceLeaf = {
      ...makeActiveService(),
      pricing: {
        type: "subscription",
        basePrice: 5000n,
        unit: "block",
        subscriptionBlocks: 1000n,
      },
    };

    expect(calculateCallCost(service, 1n)).toBe(0n);
  });

  it("should calculate tiered pricing", () => {
    const service: ServiceLeaf = {
      ...makeActiveService(),
      totalCalls: 50n,
      pricing: {
        type: "tiered",
        basePrice: 100n,
        unit: "call",
        tiers: [
          { threshold: 0n, pricePerUnit: 100n },
          { threshold: 100n, pricePerUnit: 80n },
        ],
      },
    };

    // totalUsage = 50 + 5 = 55, still in tier 1 (threshold 0)
    expect(calculateCallCost(service, 5n)).toBe(500n);

    // totalUsage = 50 + 60 = 110, in tier 2 (threshold 100)
    expect(calculateCallCost(service, 60n)).toBe(80n * 60n);
  });

  it("should use base price for dynamic pricing", () => {
    const service: ServiceLeaf = {
      ...makeActiveService(),
      pricing: { type: "dynamic", basePrice: 200n, unit: "call" },
    };

    expect(calculateCallCost(service, 3n)).toBe(600n);
  });
});

// ============ isCallAffordable ============

describe("isCallAffordable", () => {
  it("should return true when balance exceeds cost", () => {
    const service = makeActiveService();
    expect(isCallAffordable(service, 1n, 5000n)).toBe(true);
  });

  it("should return false when balance insufficient", () => {
    const service = makeActiveService();
    expect(isCallAffordable(service, 1n, 500n)).toBe(false);
  });

  it("should return true when balance equals cost", () => {
    const service = makeActiveService();
    expect(isCallAffordable(service, 1n, 1000n)).toBe(true);
  });
});

// ============ recordServiceCall ============

describe("recordServiceCall", () => {
  it("should increment call count and revenue", () => {
    const service = makeActiveService();
    const updated = recordServiceCall(service, 1000n, 50, 200n);

    expect(updated.totalCalls).toBe(1n);
    expect(updated.totalRevenue).toBe(1000n);
    expect(updated.lastCalledBlock).toBe(200n);
    expect(updated.averageLatencyMs).toBe(50);
  });

  it("should calculate rolling average latency", () => {
    let service = makeActiveService();
    service = recordServiceCall(service, 100n, 100, 200n); // avg: 100
    service = recordServiceCall(service, 100n, 200, 201n); // avg: (100*1 + 200) / 2 = 150

    expect(service.totalCalls).toBe(2n);
    expect(service.averageLatencyMs).toBe(150);
  });
});

// ============ updateUptime ============

describe("updateUptime", () => {
  it("should increment uptime for active service", () => {
    const service = makeActiveService();
    const updated = updateUptime(service);

    expect(updated.uptimeBlocks).toBe(1n);
  });

  it("should not increment uptime for non-active service", () => {
    const service = makeService(); // deploying
    const updated = updateUptime(service);

    expect(updated.uptimeBlocks).toBe(0n);
  });
});

// ============ Status Transitions ============

describe("pauseService", () => {
  it("should pause an active service", () => {
    const service = makeActiveService();
    const paused = pauseService(service);

    expect(paused.status).toBe("paused");
  });

  it("should reject pausing a deploying service", () => {
    const service = makeService();
    expect(() => pauseService(service)).toThrow("Cannot pause");
  });
});

describe("resumeService", () => {
  it("should resume a paused service", () => {
    const service = pauseService(makeActiveService());
    const resumed = resumeService(service);

    expect(resumed.status).toBe("active");
  });

  it("should reject resuming non-paused service", () => {
    const service = makeActiveService();
    expect(() => resumeService(service)).toThrow("paused");
  });
});

describe("markDegraded", () => {
  it("should mark active service as degraded", () => {
    const service = makeActiveService();
    const degraded = markDegraded(service);

    expect(degraded.status).toBe("degraded");
  });

  it("should not degrade non-active service", () => {
    const service = makeService(); // deploying
    const result = markDegraded(service);

    expect(result.status).toBe("deploying"); // Unchanged
  });
});

describe("retireService", () => {
  it("should retire any non-retired service", () => {
    const service = makeActiveService();
    const retired = retireService(service);

    expect(retired.status).toBe("retired");
  });

  it("should reject retiring already retired service", () => {
    const retired = retireService(makeActiveService());
    expect(() => retireService(retired)).toThrow("already retired");
  });
});

describe("isServiceCallable", () => {
  it("should return true for active service", () => {
    expect(isServiceCallable(makeActiveService())).toBe(true);
  });

  it("should return false for non-active service", () => {
    expect(isServiceCallable(makeService())).toBe(false);
    expect(isServiceCallable(pauseService(makeActiveService()))).toBe(false);
  });
});

// ============ generateServiceId ============

describe("generateServiceId", () => {
  it("should be deterministic", () => {
    const id1 = generateServiceId("proj-1", 0n);
    const id2 = generateServiceId("proj-1", 0n);

    expect(id1).toBe(id2);
  });

  it("should differ by nonce", () => {
    const id1 = generateServiceId("proj-1", 0n);
    const id2 = generateServiceId("proj-1", 1n);

    expect(id1).not.toBe(id2);
  });
});
