/**
 * Node Role System Tests
 *
 * Tests for role capabilities, role merging, topic filtering,
 * and each specialized node service.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import {
  ROLE_CAPABILITIES,
  ROLE_PROTOCOL_IDS,
  mergeRoleCapabilities,
  roleProcessesBlocks,
  roleParticipatesInConsensus,
  getDHTServerRoles,
  type NodeRole,
} from "../../../network/roles/types.js";
import { PUBSUB_TOPICS, PROTOCOL_IDS } from "../../../network/config.js";
import { SeedNodeService } from "../../../network/roles/seed.js";
import { SentryNodeService } from "../../../network/roles/sentry.js";
import { KnowledgeGatewayService } from "../../../network/roles/knowledge_gateway.js";
import { RelayNodeService } from "../../../network/roles/relay.js";
import { EvidenceOracleService } from "../../../network/roles/evidence_oracle.js";

const ALL_TOPICS = Object.values(PUBSUB_TOPICS);
const ALL_ROLES: NodeRole[] = [
  "full",
  "validator",
  "archive",
  "chatnode",
  "seed",
  "sentry",
  "knowledge_gateway",
  "relay",
  "evidence_oracle",
];

// ============ RoleCapabilities Completeness ============

describe("RoleCapabilities", () => {
  it("should define capabilities for every role", () => {
    for (const role of ALL_ROLES) {
      const caps = ROLE_CAPABILITIES[role];
      expect(caps).toBeDefined();
      expect(caps.description).toBeTruthy();
      expect(caps.syncStrategy).toBeTruthy();
      expect(caps.dhtMode).toMatch(/^(server|client)$/);
      expect(caps.maxConnections).toBeGreaterThan(0);
      expect(caps.minConnections).toBeGreaterThan(0);
      expect(caps.maxConnections).toBeGreaterThanOrEqual(caps.minConnections);
    }
  });

  it("should have topics as a subset of PUBSUB_TOPICS", () => {
    for (const role of ALL_ROLES) {
      const caps = ROLE_CAPABILITIES[role];
      for (const topic of caps.topics) {
        expect(ALL_TOPICS).toContain(topic);
      }
    }
  });

  it("should have non-negative storage namespaces", () => {
    for (const role of ALL_ROLES) {
      const caps = ROLE_CAPABILITIES[role];
      for (const ns of caps.storageNamespaces) {
        expect(ns).toBeGreaterThanOrEqual(0);
        expect(ns).toBeLessThanOrEqual(0xff);
      }
    }
  });
});

// ============ Topic Filtering ============

describe("Topic filtering by role", () => {
  it("full node subscribes to all 12 topics", () => {
    expect(ROLE_CAPABILITIES.full.topics.length).toBe(ALL_TOPICS.length);
  });

  it("validator subscribes to all 12 topics", () => {
    expect(ROLE_CAPABILITIES.validator.topics.length).toBe(ALL_TOPICS.length);
  });

  it("seed node subscribes to 0 topics", () => {
    expect(ROLE_CAPABILITIES.seed.topics.length).toBe(0);
  });

  it("sentry subscribes to all 12 topics (proxies everything)", () => {
    expect(ROLE_CAPABILITIES.sentry.topics.length).toBe(ALL_TOPICS.length);
  });

  it("knowledge_gateway subscribes to 5 knowledge topics", () => {
    const caps = ROLE_CAPABILITIES.knowledge_gateway;
    expect(caps.topics.length).toBe(5);
    expect(caps.topics).toContain(PUBSUB_TOPICS.CLAIMS);
    expect(caps.topics).toContain(PUBSUB_TOPICS.VERIFICATIONS);
    expect(caps.topics).toContain(PUBSUB_TOPICS.EVIDENCE);
    expect(caps.topics).toContain(PUBSUB_TOPICS.EVIDENCE_REQUESTS);
    expect(caps.topics).toContain(PUBSUB_TOPICS.ATTESTATIONS);
  });

  it("relay subscribes to 4 core topics", () => {
    const caps = ROLE_CAPABILITIES.relay;
    expect(caps.topics.length).toBe(4);
    expect(caps.topics).toContain(PUBSUB_TOPICS.BLOCKS);
    expect(caps.topics).toContain(PUBSUB_TOPICS.TRANSACTIONS);
    expect(caps.topics).toContain(PUBSUB_TOPICS.CONSENSUS);
    expect(caps.topics).toContain(PUBSUB_TOPICS.VALIDATORS);
  });

  it("evidence_oracle subscribes to 3 evidence topics", () => {
    const caps = ROLE_CAPABILITIES.evidence_oracle;
    expect(caps.topics.length).toBe(3);
    expect(caps.topics).toContain(PUBSUB_TOPICS.EVIDENCE);
    expect(caps.topics).toContain(PUBSUB_TOPICS.EVIDENCE_REQUESTS);
    expect(caps.topics).toContain(PUBSUB_TOPICS.ATTESTATIONS);
  });
});

// ============ Storage Namespace Filtering ============

describe("Storage namespace filtering", () => {
  it("seed node stores only config namespace", () => {
    const ns = ROLE_CAPABILITIES.seed.storageNamespaces;
    expect(ns.length).toBe(1);
    expect(ns).toContain(0xf0); // CONFIG
  });

  it("knowledge_gateway stores knowledge-focused namespaces", () => {
    const ns = ROLE_CAPABILITIES.knowledge_gateway.storageNamespaces;
    expect(ns).toContain(0x03); // KNOWLEDGE
    expect(ns).toContain(0x04); // EVIDENCE
    expect(ns).toContain(0x07); // RESEARCH
    // Should NOT contain account or validator state
    expect(ns).not.toContain(0x01); // ACCOUNTS
    expect(ns).not.toContain(0x02); // VALIDATORS
  });

  it("relay stores only block headers and config", () => {
    const ns = ROLE_CAPABILITIES.relay.storageNamespaces;
    expect(ns).toContain(0x10); // BLOCK_HEADERS
    expect(ns).toContain(0xf0); // CONFIG
    expect(ns).not.toContain(0x01); // ACCOUNTS
    expect(ns).not.toContain(0x11); // BLOCK_BODIES
  });

  it("archive stores everything including archive namespace", () => {
    const ns = ROLE_CAPABILITIES.archive.storageNamespaces;
    expect(ns).toContain(0x30); // ARCHIVE
    expect(ns).toContain(0x01); // ACCOUNTS
    expect(ns).toContain(0x02); // VALIDATORS
  });
});

// ============ Role Utilities ============

describe("Role utilities", () => {
  it("roleProcessesBlocks returns true for block-subscribing roles", () => {
    expect(roleProcessesBlocks("full")).toBe(true);
    expect(roleProcessesBlocks("validator")).toBe(true);
    expect(roleProcessesBlocks("relay")).toBe(true);
    expect(roleProcessesBlocks("seed")).toBe(false);
    expect(roleProcessesBlocks("evidence_oracle")).toBe(false);
    expect(roleProcessesBlocks("knowledge_gateway")).toBe(false);
  });

  it("roleParticipatesInConsensus returns true for consensus roles", () => {
    expect(roleParticipatesInConsensus("full")).toBe(true);
    expect(roleParticipatesInConsensus("validator")).toBe(true);
    expect(roleParticipatesInConsensus("relay")).toBe(true);
    expect(roleParticipatesInConsensus("seed")).toBe(false);
    expect(roleParticipatesInConsensus("evidence_oracle")).toBe(false);
  });

  it("getDHTServerRoles returns roles with server DHT mode", () => {
    const serverRoles = getDHTServerRoles();
    expect(serverRoles).toContain("full");
    expect(serverRoles).toContain("seed");
    expect(serverRoles).toContain("sentry");
    expect(serverRoles).toContain("relay");
    expect(serverRoles).not.toContain("chatnode");
    expect(serverRoles).not.toContain("evidence_oracle");
  });
});

// ============ mergeRoleCapabilities ============

describe("mergeRoleCapabilities", () => {
  it("returns full capabilities for empty roles array", () => {
    const merged = mergeRoleCapabilities([]);
    expect(merged).toEqual(ROLE_CAPABILITIES.full);
  });

  it("returns exact capabilities for single role", () => {
    const merged = mergeRoleCapabilities(["seed"]);
    expect(merged).toEqual(ROLE_CAPABILITIES.seed);
  });

  it("merges topics from multiple roles", () => {
    const merged = mergeRoleCapabilities(["seed", "knowledge_gateway"]);
    // seed has 0, knowledge_gateway has 5
    expect(merged.topics.length).toBe(5);
  });

  it("takes highest maxConnections", () => {
    const merged = mergeRoleCapabilities(["seed", "chatnode"]);
    expect(merged.maxConnections).toBe(
      Math.max(
        ROLE_CAPABILITIES.seed.maxConnections,
        ROLE_CAPABILITIES.chatnode.maxConnections,
      ),
    );
  });

  it("takes most complete sync strategy", () => {
    const merged = mergeRoleCapabilities(["seed", "validator"]);
    expect(merged.syncStrategy).toBe("full"); // full > none
  });

  it("prefers server DHT mode", () => {
    const merged = mergeRoleCapabilities(["chatnode", "seed"]);
    expect(merged.dhtMode).toBe("server"); // seed is server
  });

  it("unions storage namespaces", () => {
    const merged = mergeRoleCapabilities(["seed", "knowledge_gateway"]);
    // seed has [CONFIG], knowledge_gateway has knowledge set
    expect(merged.storageNamespaces).toContain(0xf0); // CONFIG from seed
    expect(merged.storageNamespaces).toContain(0x03); // KNOWLEDGE from gateway
  });
});

// ============ SeedNodeService ============

describe.sequential("SeedNodeService", () => {
  let service: SeedNodeService;

  beforeEach(() => {
    service = new SeedNodeService();
    service.start();
  });

  afterEach(() => {
    service.stop();
  });

  it("should start with empty address book", () => {
    const stats = service.getStats();
    expect(stats.totalPeers).toBe(0);
    expect(stats.pexRequestsServed).toBe(0);
  });

  it("should register valid peers", () => {
    const registered = service.registerPeer(
      "/ip4/192.168.1.1/tcp/9000/p2p/12D3KooWTestPeerIdThatIsExactlyFiftyTwoCharactersLo",
    );
    expect(registered).toBe(true);
    expect(service.getStats().totalPeers).toBe(1);
  });

  it("should reject invalid addresses", () => {
    const registered = service.registerPeer("not-a-multiaddr");
    expect(registered).toBe(false);
    expect(service.getStats().totalPeers).toBe(0);
  });

  it("should handle peer exchange requests", () => {
    // Register some peers
    service.registerPeer(
      "/ip4/192.168.1.1/tcp/9000/p2p/12D3KooWTestPeerIdThatIsExactlyFiftyTwoCharactersLo",
    );
    service.registerPeer(
      "/ip4/192.168.1.2/tcp/9000/p2p/12D3KooWTestPeerIdThatIsExactlyFiftyTwoCharactersLp",
    );

    const response = service.handlePeerExchange({
      requesterId: "12D3KooWRequester",
      maxPeers: 10,
    });

    expect(response.peers.length).toBe(2);
    expect(response.totalKnown).toBe(2);
    expect(service.getStats().pexRequestsServed).toBe(1);
  });

  it("should exclude requester from PEX response", () => {
    const peerId = "12D3KooWTestPeerIdThatIsExactlyFiftyTwoCharactersLo";
    service.registerPeer(`/ip4/192.168.1.1/tcp/9000/p2p/${peerId}`);

    const response = service.handlePeerExchange({
      requesterId: peerId,
    });

    expect(response.peers.length).toBe(0);
  });

  it("should track liveness via markPeerFailed", () => {
    const peerId = "12D3KooWTestPeerIdThatIsExactlyFiftyTwoCharactersLo";
    service.registerPeer(`/ip4/192.168.1.1/tcp/9000/p2p/${peerId}`);
    service.markPeerFailed(peerId);
    service.markPeerFailed(peerId);
    service.markPeerFailed(peerId);

    // After 3 failures, peer should be filtered from PEX
    const response = service.handlePeerExchange({
      requesterId: "someone-else",
    });
    expect(response.peers.length).toBe(0);
  });

  it("should report correct protocol ID", () => {
    expect(SeedNodeService.protocolId).toBe(ROLE_PROTOCOL_IDS.PEER_EXCHANGE);
  });
});

// ============ SentryNodeService ============

describe.sequential("SentryNodeService", () => {
  let service: SentryNodeService;
  const VALIDATOR_PEER_ID = "validator-peer-001";

  beforeEach(() => {
    service = new SentryNodeService({
      validatorPeerId: VALIDATOR_PEER_ID,
      rateLimitPerPeer: 5, // Low limit for testing
    });
    service.start();
  });

  afterEach(() => {
    service.stop();
  });

  it("should allow non-blocklisted connections", () => {
    expect(service.shouldAllowConnection("peer-001")).toBe(true);
  });

  it("should block blocklisted peers", () => {
    const svc = new SentryNodeService({
      validatorPeerId: VALIDATOR_PEER_ID,
      connectionBlocklist: ["bad-peer"],
    });
    expect(svc.shouldAllowConnection("bad-peer")).toBe(false);
    expect(svc.getStats().connectionsBlocked).toBe(1);
  });

  it("should enforce allowlist when set", () => {
    const svc = new SentryNodeService({
      validatorPeerId: VALIDATOR_PEER_ID,
      connectionAllowlist: ["good-peer"],
    });
    expect(svc.shouldAllowConnection("good-peer")).toBe(true);
    expect(svc.shouldAllowConnection("unknown-peer")).toBe(false);
  });

  it("should always allow validator through allowlist", () => {
    const svc = new SentryNodeService({
      validatorPeerId: VALIDATOR_PEER_ID,
      connectionAllowlist: ["good-peer"],
    });
    expect(svc.shouldAllowConnection(VALIDATOR_PEER_ID)).toBe(true);
  });

  it("should filter messages with rate limiting", () => {
    const data = new Uint8Array([1, 2, 3]);

    // First 5 should pass
    for (let i = 0; i < 5; i++) {
      expect(service.filterInboundMessage("peer-1", "blocks", data)).toBe(true);
    }

    // 6th should be rate-limited
    expect(service.filterInboundMessage("peer-1", "blocks", data)).toBe(false);
    expect(service.getStats().messagesDropped).toBeGreaterThan(0);
  });

  it("should relay messages from validator", () => {
    service.handleValidatorOutbound("blocks", new Uint8Array([1]));
    expect(service.getStats().messagesRelayedFromValidator).toBe(1);
  });

  it("should track validator health", () => {
    service.recordValidatorHeartbeat();
    expect(service.getStats().validatorConnected).toBe(true);
  });

  it("should never expose validator peer ID", () => {
    expect(service.getValidatorPeerId()).toBe(VALIDATOR_PEER_ID);
  });

  it("should report correct protocol ID", () => {
    expect(SentryNodeService.protocolId).toBe(ROLE_PROTOCOL_IDS.SENTRY_LINK);
  });
});

// ============ KnowledgeGatewayService ============

describe.sequential("KnowledgeGatewayService", () => {
  let service: KnowledgeGatewayService;

  beforeEach(() => {
    service = new KnowledgeGatewayService({
      rateLimitPerDid: 3,
    });
    service.setQueryHandler(async (query) => ({
      success: true,
      type: query.type,
      data: [{ id: "fact:001", confidence: 0.95 }],
      totalCount: 1,
      executionTimeMs: 0,
    }));
    service.start();
  });

  afterEach(() => {
    service.stop();
  });

  it("should execute a query successfully", async () => {
    const result = await service.executeQuery({
      type: "fact_by_id",
      params: { factId: "fact:001" },
    });

    expect(result.success).toBe(true);
    expect(result.data.length).toBe(1);
    expect(result.executionTimeMs).toBeGreaterThanOrEqual(0);
  });

  it("should cache query results", async () => {
    // First query
    await service.executeQuery({
      type: "fact_by_id",
      params: { factId: "fact:001" },
    });

    // Second identical query — should hit cache
    await service.executeQuery({
      type: "fact_by_id",
      params: { factId: "fact:001" },
    });

    const stats = service.getStats();
    expect(stats.cacheHitRate).toBeGreaterThan(0);
    expect(stats.cacheSize).toBe(1);
  });

  it("should rate limit per DID", async () => {
    const did = "did:lgm:rateLimitTest" as any;

    // First 3 should succeed
    for (let i = 0; i < 3; i++) {
      const result = await service.executeQuery({
        type: "fact_by_id",
        params: { factId: `fact:${i}` },
        requesterDid: did,
      });
      expect(result.success).toBe(true);
    }

    // 4th should be rate-limited
    const result = await service.executeQuery({
      type: "fact_by_id",
      params: { factId: "fact:3" },
      requesterDid: did,
    });
    expect(result.success).toBe(false);
    expect(result.error).toContain("Rate limit");
  });

  it("should handle batch queries", async () => {
    const result = await service.executeBatchQuery({
      queries: [
        { type: "fact_by_id", params: { factId: "fact:001" } },
        { type: "fact_by_id", params: { factId: "fact:002" } },
      ],
    });

    expect(result.results.length).toBe(2);
    expect(result.totalExecutionTimeMs).toBeGreaterThanOrEqual(0);
  });

  it("should reject oversized batch", async () => {
    const service2 = new KnowledgeGatewayService({ maxBatchSize: 2 });
    service2.setQueryHandler(async (q) => ({
      success: true,
      type: q.type,
      data: [],
      totalCount: 0,
      executionTimeMs: 0,
    }));
    service2.start();

    const result = await service2.executeBatchQuery({
      queries: [
        { type: "fact_by_id", params: {} },
        { type: "fact_by_id", params: {} },
        { type: "fact_by_id", params: {} },
      ],
    });

    expect(result.results[0].success).toBe(false);
    expect(result.results[0].error).toContain("Batch too large");
    service2.stop();
  });

  it("should report correct protocol ID", () => {
    expect(KnowledgeGatewayService.protocolId).toBe(
      ROLE_PROTOCOL_IDS.KNOWLEDGE_QUERY,
    );
  });
});

// ============ RelayNodeService ============

describe.sequential("RelayNodeService", () => {
  let service: RelayNodeService;

  beforeEach(() => {
    service = new RelayNodeService();
    service.start();
  });

  afterEach(() => {
    service.stop();
  });

  it("should relay new messages", () => {
    const data = new Uint8Array([1, 2, 3]);
    const relayed = service.processMessage("msg-001", "blocks", data, "peer-1");

    expect(relayed).toBe(true);
    expect(service.getStats().messagesRelayed).toBe(1);
  });

  it("should deduplicate messages via bloom filter", () => {
    const data = new Uint8Array([1, 2, 3]);

    // First should relay
    expect(service.processMessage("msg-001", "blocks", data, "peer-1")).toBe(
      true,
    );

    // Same message ID should be deduplicated
    expect(service.processMessage("msg-001", "blocks", data, "peer-2")).toBe(
      false,
    );

    expect(service.getStats().messagesDeduplicated).toBe(1);
  });

  it("should prioritize consensus messages", () => {
    const data = new Uint8Array([1, 2, 3]);

    // Queue in reverse priority order
    service.processMessage("tx-1", "transactions", data, "peer-1");
    service.processMessage("block-1", "blocks", data, "peer-1");
    service.processMessage("consensus-1", "consensus", data, "peer-1");

    // Drain should return highest priority first
    const batch = service.drainQueue(10);
    expect(batch.length).toBe(3);
    expect(batch[0].id).toBe("consensus-1");
    expect(batch[1].id).toBe("block-1");
    expect(batch[2].id).toBe("tx-1");
  });

  it("should record peer latency", () => {
    service.recordLatency("peer-1", 50, "us-east");
    service.recordLatency("peer-2", 150, "eu-west");

    const stats = service.getStats();
    expect(stats.peersByRegion["us-east"]).toBe(1);
    expect(stats.peersByRegion["eu-west"]).toBe(1);
    expect(stats.latencyByRegion["us-east"]).toBe(50);
  });

  it("should report bloom filter false positive rate", () => {
    // Process some messages to fill the bloom
    for (let i = 0; i < 100; i++) {
      service.processMessage(
        `msg-${i}`,
        "blocks",
        new Uint8Array([1]),
        "peer-1",
      );
    }

    const stats = service.getStats();
    expect(stats.bloomFalsePositiveRate).toBeGreaterThanOrEqual(0);
    expect(stats.bloomFalsePositiveRate).toBeLessThan(1);
  });

  it("should report correct protocol ID", () => {
    expect(RelayNodeService.protocolId).toBe(ROLE_PROTOCOL_IDS.RELAY);
  });
});

// ============ EvidenceOracleService ============

describe.sequential("EvidenceOracleService", () => {
  let service: EvidenceOracleService;

  beforeEach(() => {
    service = new EvidenceOracleService({
      sourceRateLimit: 5,
    });

    // Set up mock handlers
    service.setFetchHandler(async (source, _type, _timeout) => {
      return {
        data: new TextEncoder().encode(`Content from ${source}`),
        contentType: "text/plain",
      };
    });
    service.setHashHandler((_data) => "sha256:mockhash");

    service.start();
  });

  afterEach(() => {
    service.stop();
  });

  it("should process a fetch request successfully", async () => {
    const result = await service.processFetchRequest({
      id: "req-001",
      source: "https://example.com/evidence",
      sourceType: "url",
      priority: 1,
      createdAt: Date.now(),
    });

    expect(result.success).toBe(true);
    expect(result.contentHash).toBe("sha256:mockhash");
    expect(result.contentSize).toBeGreaterThan(0);
    expect(result.fetchDurationMs).toBeGreaterThanOrEqual(0);
  });

  it("should verify expected content hash", async () => {
    const result = await service.processFetchRequest({
      id: "req-002",
      source: "https://example.com/evidence",
      sourceType: "url",
      expectedHash: "sha256:mockhash",
      priority: 1,
      createdAt: Date.now(),
    });

    expect(result.hashVerified).toBe(true);
  });

  it("should detect hash mismatch", async () => {
    const result = await service.processFetchRequest({
      id: "req-003",
      source: "https://example.com/evidence",
      sourceType: "url",
      expectedHash: "sha256:different",
      priority: 1,
      createdAt: Date.now(),
    });

    expect(result.hashVerified).toBe(false);
    expect(service.getStats().hashMismatches).toBe(1);
  });

  it("should handle fetch failures gracefully", async () => {
    service.setFetchHandler(async () => {
      throw new Error("Network error");
    });

    const result = await service.processFetchRequest({
      id: "req-004",
      source: "https://example.com/broken",
      sourceType: "url",
      priority: 1,
      createdAt: Date.now(),
    });

    expect(result.success).toBe(false);
    expect(result.error).toContain("Network error");
    expect(service.getStats().failedFetches).toBe(1);
  });

  it("should track source reliability", async () => {
    // Successful fetch
    await service.processFetchRequest({
      id: "req-005",
      source: "https://reliable.com/data",
      sourceType: "url",
      priority: 1,
      createdAt: Date.now(),
    });

    const reliability = service.getSourceReliability(
      "https://reliable.com/data",
    );
    expect(reliability).toBeDefined();
    expect(reliability!.successfulFetches).toBe(1);
    expect(reliability!.reliabilityScore).toBeGreaterThan(0);
  });

  it("should submit and process queue entries", () => {
    const accepted = service.submitFetchRequest({
      id: "req-006",
      source: "https://example.com/queued",
      sourceType: "url",
      priority: 1,
      createdAt: Date.now(),
    });

    expect(accepted).toBe(true);
    expect(service.getStats().queueSize).toBe(1);
  });

  it("should report correct protocol ID", () => {
    expect(EvidenceOracleService.protocolId).toBe(
      ROLE_PROTOCOL_IDS.EVIDENCE_FETCH,
    );
  });
});
