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
  recoverService,
  retireService,
  isServiceCallable,
  generateServiceId,
  slugifyServiceName,
  generateSubdomain,
  ServiceError,
} from "../../../core/tree/service_deployment.js";
import { ServiceStateTree } from "../../../state/services.js";
import { SERVICE_BASE_DOMAIN } from "../../../core/constants.js";
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

const MOCK_ENDPOINT = {
  protocol: "rest" as const,
  contractAddress: "contract-addr-1",
  url: "https://example.com/api",
  methods: ["query"],
};

function makeService(overrides: Partial<ServiceLeaf> = {}): ServiceLeaf {
  const service = createServiceLeaf(
    "proj-1",
    "Test Service",
    "A test service",
    "api",
    MOCK_ENDPOINT,
    MOCK_ABI,
    { type: "per_call", basePrice: 1000n, unit: "call" },
    "contract-addr-1",
    0n,
    100n,
  );
  return { ...service, ...overrides };
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
      {
        protocol: "rest" as const,
        contractAddress: "contract-1",
        methods: ["query"],
      },
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
      { protocol: "rpc" as const, contractAddress: "", methods: [] },
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
      { protocol: "rpc" as const, contractAddress: "", methods: [] },
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
      { protocol: "rpc" as const, contractAddress: "", methods: [] },
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

    // Marginal pricing: 50 units fill tier 1 (100n each = 5000n),
    // remaining 10 units at tier 2 rate (80n each = 800n) = 5800n
    expect(calculateCallCost(service, 60n)).toBe(100n * 50n + 80n * 10n);
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

  it("should reject degrading non-active service", () => {
    const service = makeService(); // deploying
    expect(() => markDegraded(service)).toThrow("expected 'active'");
  });
});

describe("recoverService", () => {
  it("should recover a degraded service to active", () => {
    const degraded = markDegraded(makeActiveService());
    const recovered = recoverService(degraded);

    expect(recovered.status).toBe("active");
  });

  it("should reject recovering non-degraded service", () => {
    const service = makeActiveService();
    expect(() => recoverService(service)).toThrow("expected 'degraded'");
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

  it("should reject retiring a deploying service", () => {
    const service = makeService(); // still in "deploying" status
    expect(() => retireService(service)).toThrow("never activated");
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

// ============ slugifyServiceName ============

describe("slugifyServiceName", () => {
  it("should lowercase and hyphenate standard names", () => {
    expect(slugifyServiceName("Inference API")).toBe("inference-api");
  });

  it("should strip special characters", () => {
    expect(slugifyServiceName("My $ervice (v2.0)")).toBe("my-ervice-v2-0");
  });

  it("should collapse consecutive hyphens", () => {
    expect(slugifyServiceName("a---b")).toBe("a-b");
  });

  it("should strip leading and trailing hyphens", () => {
    expect(slugifyServiceName("--hello--")).toBe("hello");
  });

  it("should return empty string for all-special-char input", () => {
    expect(slugifyServiceName("$$$%%%")).toBe("");
  });

  it("should truncate to maxLength", () => {
    const long = "a".repeat(100);
    const result = slugifyServiceName(long, 10);
    expect(result.length).toBeLessThanOrEqual(10);
    expect(result).toBe("aaaaaaaaaa");
  });

  it("should not end with trailing hyphen after truncation", () => {
    // "ab-cd-ef" truncated to 5 chars = "ab-cd" → fine
    // "ab-cd-ef" truncated to 3 chars = "ab-" → strip trailing → "ab"
    expect(slugifyServiceName("ab cd ef", 3)).toBe("ab");
  });

  it("should return empty for empty input", () => {
    expect(slugifyServiceName("")).toBe("");
  });
});

// ============ generateSubdomain ============

describe("generateSubdomain", () => {
  it("should produce deterministic subdomains", () => {
    const id = generateServiceId("proj-1", 0n);
    const sub1 = generateSubdomain(id, "Inference API");
    const sub2 = generateSubdomain(id, "Inference API");

    expect(sub1).toBe(sub2);
  });

  it("should include slug and short ID in label", () => {
    const id = generateServiceId("proj-1", 0n);
    const sub = generateSubdomain(id, "Inference API");
    const shortId = id.slice(0, 8);

    expect(sub).toBe(`inference-api-${shortId}.${SERVICE_BASE_DOMAIN}`);
  });

  it("should fall back to short ID when name slugifies to empty", () => {
    const id = generateServiceId("proj-1", 0n);
    const sub = generateSubdomain(id, "$$$");
    const shortId = id.slice(0, 8);

    expect(sub).toBe(`${shortId}.${SERVICE_BASE_DOMAIN}`);
  });

  it("should use custom base domain when provided", () => {
    const id = generateServiceId("proj-1", 0n);
    const sub = generateSubdomain(id, "Test", "custom.domain");

    expect(sub).toContain(".custom.domain");
  });

  it("should produce labels ≤ 63 chars (DNS RFC 1035)", () => {
    const id = generateServiceId("proj-1", 0n);
    const longName = "a".repeat(200);
    const sub = generateSubdomain(id, longName);
    const label = sub.split(".")[0];

    expect(label.length).toBeLessThanOrEqual(63);
  });
});

// ============ createServiceLeaf subdomain integration ============

describe("createServiceLeaf subdomain", () => {
  it("should populate subdomain field", () => {
    const service = createServiceLeaf(
      "proj-1",
      "My Analytics API",
      "Analytics",
      "analytics",
      MOCK_ENDPOINT,
      MOCK_ABI,
      { type: "per_call", basePrice: 100n, unit: "call" },
      "contract-1",
      0n,
      100n,
    );

    expect(service.subdomain).toBeTruthy();
    expect(service.subdomain).toContain(SERVICE_BASE_DOMAIN);
    expect(service.subdomain).toContain("my-analytics-api");
  });

  it("should have deterministic subdomain from same inputs", () => {
    const args = [
      "proj-1",
      "Service A",
      "Desc",
      "api" as const,
      MOCK_ENDPOINT,
      MOCK_ABI,
      { type: "per_call" as const, basePrice: 100n, unit: "call" as const },
      "contract-1",
      5n,
      200n,
    ] as const;

    const s1 = createServiceLeaf(...args);
    const s2 = createServiceLeaf(...args);

    expect(s1.subdomain).toBe(s2.subdomain);
  });
});

// ============ ServiceStateTree subdomain index ============

describe("ServiceStateTree subdomain index", () => {
  function makeTestService(
    id: string,
    subdomain: string,
  ): import("../../../core/tree/types.js").ServiceLeaf {
    return {
      id,
      projectId: "proj-1",
      name: "Test",
      description: "Test",
      serviceType: "api",
      endpoint: { protocol: "rest", contractAddress: "c1", methods: ["q"] },
      subdomain,
      abi: MOCK_ABI,
      pricing: { type: "per_call", basePrice: 100n, unit: "call" },
      contractAddress: "c1",
      totalCalls: 0n,
      totalRevenue: 0n,
      uptimeBlocks: 0n,
      averageLatencyMs: 0,
      status: "active",
      deployedAtBlock: 100n,
    };
  }

  it("should index subdomain on create and look up by subdomain", () => {
    const tree = new ServiceStateTree();
    const service = makeTestService(
      "svc-1",
      "my-api-12345678.services.legible.money",
    );
    tree.createService(service);

    const found = tree.getServiceBySubdomain(
      "my-api-12345678.services.legible.money",
    );
    expect(found).not.toBeNull();
    expect(found!.id).toBe("svc-1");
  });

  it("should return null for unknown subdomain", () => {
    const tree = new ServiceStateTree();
    expect(
      tree.getServiceBySubdomain("unknown.services.legible.money"),
    ).toBeNull();
  });

  it("should not index empty subdomain", () => {
    const tree = new ServiceStateTree();
    const service = makeTestService("svc-1", "");
    tree.createService(service);

    expect(tree.getServiceBySubdomain("")).toBeNull();
  });

  it("should preserve subdomain index after clone", () => {
    const tree = new ServiceStateTree();
    const service = makeTestService(
      "svc-1",
      "cloned-api-abcdef12.services.legible.money",
    );
    tree.createService(service);

    const cloned = tree.clone();
    const found = cloned.getServiceBySubdomain(
      "cloned-api-abcdef12.services.legible.money",
    );
    expect(found).not.toBeNull();
    expect(found!.id).toBe("svc-1");
  });

  it("should preserve subdomain through serialization round-trip", () => {
    const tree = new ServiceStateTree();
    const subdomain = "round-trip-12345678.services.legible.money";
    const service = makeTestService("svc-rt", subdomain);
    tree.createService(service);

    // Export and restore into a new tree
    const exported = tree.exportServices();
    const tree2 = new ServiceStateTree();
    for (const svc of exported) {
      tree2.createService(svc);
    }

    const found = tree2.getServiceBySubdomain(subdomain);
    expect(found).not.toBeNull();
    expect(found!.subdomain).toBe(subdomain);
  });
});
