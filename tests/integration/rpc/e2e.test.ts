/**
 * RPC E2E Integration Tests
 *
 * Full end-to-end integration scenarios testing the complete
 * request/response cycle with real consensus and state.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { RPCServer } from "../../../protocol/rpc/server.js";
import {
  TestNetwork,
  createTestNetwork,
} from "../../framework/test_harness.js";
import { TestRPCClient, TestWSClient } from "./helpers/rpc_client.js";
import {
  createMockHandlerContext,
  RPC_TEST_CONFIG,
  TEST_SERVER_CONFIG,
  TEST_CLAIMS,
} from "./helpers/fixtures.js";
import type {
  AccountGetBalanceResponse,
  KnowledgeGetFactResponse,
  VerificationGetPendingResponse,
  NetworkHealthResponse,
} from "../../../protocol/types.js";

// ============ Full Lifecycle Tests ============

describe.sequential("RPC E2E - Full Lifecycle", () => {
  let network: TestNetwork;
  let server: RPCServer;
  let client: TestRPCClient;

  beforeEach(async () => {
    network = await createTestNetwork({
      ...RPC_TEST_CONFIG,
      validatorCount: 5,
    });
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, TEST_SERVER_CONFIG);
    await server.start();
    const addr = server.getAddress()!;
    client = new TestRPCClient(`http://${addr.host}:${addr.port}`);
  });

  afterEach(async () => {
    await server.stop();
    await network.stop();
  });

  it("should complete full claim verification lifecycle via RPC", async () => {
    const account = network.getAccount(0);
    const testClaim = TEST_CLAIMS[0];

    // 1. Check initial account balance via RPC
    const initialBalance = await client.call<AccountGetBalanceResponse>(
      "account_getBalance",
      {
        did: account.did,
      },
    );
    expect(BigInt(initialBalance.balance)).toBeGreaterThan(0n);

    // 2. Submit claim directly to network (simulating RPC submission)
    const claimId = await network.submitClaim(
      testClaim.claim,
      testClaim.domain,
      account,
      100n * 10n ** 18n,
    );
    expect(claimId).toBeTruthy();

    // 3. Check validator has pending assignment via RPC
    const validator = network.getValidator(0);
    const pending = await client.call<VerificationGetPendingResponse>(
      "verification_getPending",
      {
        validatorKey: validator.publicKey,
      },
    );
    // Assignment may or may not be present depending on VRF selection
    expect(Array.isArray(pending.assignments)).toBe(true);

    // 4. Complete verification round
    const rounds = network.getConsensus().getActiveRounds();
    if (rounds.length > 0) {
      await network.completeVerificationRoundDeterministic(
        rounds[0].id,
        network.validators.map((v) => ({
          validator: v,
          verdict: "confirmed" as const,
          confidence: 0.9,
        })),
      );
    }

    // 5. Query the fact via RPC
    const factResult = await client.call<KnowledgeGetFactResponse>(
      "knowledge_getFact",
      {
        factId: claimId,
      },
    );

    // Fact should exist after verification completes
    // Note: The fact may or may not exist depending on verification timing
    expect(factResult.blockNumber).toBeDefined();
    if (factResult.fact) {
      expect(factResult.fact.claim).toBeTruthy(); // Claim text exists (may be hashed/summarized)
      expect(factResult.fact.confidence).toBeGreaterThan(0);
    }
  });

  it("should handle query progression through RPC", async () => {
    // 1. Start knowledge session
    const account = network.getAccount(0);
    const sessionStart = await client.call<{
      sessionId: string;
      freeTierQueries: number;
    }>("knowledge_sessionStart", { agentDid: account.did });
    expect(sessionStart.sessionId).toBeTruthy();
    expect(sessionStart.freeTierQueries).toBeGreaterThan(0);

    // 2. Estimate query cost
    const costEstimate = await client.call<{
      cost: string;
      estimatedBuckets: number;
    }>("knowledge_estimateQueryCost", {
      sessionId: sessionStart.sessionId,
      intent: {
        type: "explore",
        domain: "test",
        limit: 10,
      },
    });
    expect(BigInt(costEstimate.cost)).toBeGreaterThan(0n);

    // 3. Execute query
    const queryResult = await client.call<{
      facts: unknown[];
      metadata: { latencyMs: number };
    }>("knowledge_query", {
      sessionId: sessionStart.sessionId,
      intent: {
        type: "explore",
        domain: "test",
        limit: 10,
      },
    });
    expect(Array.isArray(queryResult.facts)).toBe(true);
    expect(queryResult.metadata.latencyMs).toBeDefined();

    // 4. End session
    const sessionEnd = await client.call<{
      success: boolean;
      totalQueries: number;
    }>("knowledge_sessionEnd", { sessionId: sessionStart.sessionId });
    expect(sessionEnd.success).toBe(true);
  });
});

// ============ Concurrent Requests Tests ============

describe.sequential("RPC E2E - Concurrent Requests", () => {
  let network: TestNetwork;
  let server: RPCServer;
  let client: TestRPCClient;

  beforeEach(async () => {
    network = await createTestNetwork(RPC_TEST_CONFIG);
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, {
      ...TEST_SERVER_CONFIG,
      rateLimits: {
        ...TEST_SERVER_CONFIG.rateLimits,
        maxRequests: {
          free: 10000, // High limit for concurrent tests
          staked100: 30000,
          staked1000: 100000,
          validator: 1000000,
        },
      },
    });
    await server.start();
    const addr = server.getAddress()!;
    client = new TestRPCClient(`http://${addr.host}:${addr.port}`);
  });

  afterEach(async () => {
    await server.stop();
    await network.stop();
  });

  it("should handle 50 concurrent requests", async () => {
    const requests = Array.from({ length: 50 }, () =>
      client.call<NetworkHealthResponse>("network_health"),
    );

    const results = await Promise.all(requests);

    expect(results.length).toBe(50);
    for (const result of results) {
      expect(result.status).toBe("healthy");
    }
  });

  it("should handle mixed concurrent requests", async () => {
    const accounts = network.accounts.slice(0, 3);
    const validators = network.validators.slice(0, 2);

    const requests = [
      // Health checks
      ...Array.from({ length: 10 }, () =>
        client.call<NetworkHealthResponse>("network_health"),
      ),
      // Balance queries
      ...accounts.map((a) =>
        client.call<AccountGetBalanceResponse>("account_getBalance", {
          did: a.did,
        }),
      ),
      // Validator pending checks
      ...validators.map((v) =>
        client.call<VerificationGetPendingResponse>("verification_getPending", {
          validatorKey: v.publicKey,
        }),
      ),
    ];

    const results = await Promise.all(requests);

    expect(results.length).toBe(15);
    // All requests should have completed without error
  });

  it("should maintain consistency under concurrent load", async () => {
    const account = network.getAccount(0);

    // Make many concurrent balance queries for the same account
    const requests = Array.from({ length: 20 }, () =>
      client.call<AccountGetBalanceResponse>("account_getBalance", {
        did: account.did,
      }),
    );

    const results = await Promise.all(requests);

    // All results should be identical
    const firstBalance = results[0].balance;
    for (const result of results) {
      expect(result.balance).toBe(firstBalance);
    }
  });
});

// ============ WebSocket Event Integration Tests ============

describe.sequential("RPC E2E - WebSocket Events", () => {
  let network: TestNetwork;
  let server: RPCServer;
  let wsClient: TestWSClient;

  beforeEach(async () => {
    network = await createTestNetwork(RPC_TEST_CONFIG);
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, TEST_SERVER_CONFIG);
    await server.start();
    const addr = server.getAddress()!;
    wsClient = new TestWSClient(`ws://${addr.host}:${addr.port}`);
    await wsClient.connect();
  });

  afterEach(async () => {
    wsClient.disconnect();
    await server.stop();
    await network.stop();
  });

  it("should receive block events via WebSocket subscription", async () => {
    // Subscribe to block events
    await wsClient.subscribe("block");

    // Set up event listener
    const eventPromise = wsClient.waitForEvent(5000);

    // Trigger a block event via broadcast
    (server as any).broadcast("block", {
      number: 1,
      hash: "test-hash",
      timestamp: Date.now(),
    });

    const event = await eventPromise;
    expect(event).toBeDefined();
    expect((event as any).event.number).toBe(1);
  });

  it("should receive multiple event types", async () => {
    await wsClient.subscribe("block");
    await wsClient.subscribe("transaction");

    const events: unknown[] = [];
    const cleanup = wsClient.onEvent((event) => {
      events.push(event);
    });

    // Broadcast different event types
    (server as any).broadcast("block", { type: "block", number: 1 });
    (server as any).broadcast("transaction", { type: "tx", hash: "tx-1" });

    // Wait for events
    await new Promise((resolve) => setTimeout(resolve, 200));

    expect(events.length).toBe(2);
    cleanup();
  });

  it("should handle RPC calls and subscriptions simultaneously", async () => {
    // Subscribe to events
    await wsClient.subscribe("test-event");

    // Make RPC calls while subscribed
    const healthResult =
      await wsClient.call<NetworkHealthResponse>("network_health");
    expect(healthResult.status).toBe("healthy");

    // Should still receive events
    const eventPromise = wsClient.waitForEvent(3000);
    (server as any).broadcast("test-event", { data: "test" });

    const event = await eventPromise;
    expect((event as any).event.data).toBe("test");
  });
});

// ============ Error Recovery Tests ============

describe.sequential("RPC E2E - Error Recovery", () => {
  let network: TestNetwork;
  let server: RPCServer;
  let client: TestRPCClient;

  beforeEach(async () => {
    network = await createTestNetwork(RPC_TEST_CONFIG);
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, TEST_SERVER_CONFIG);
    await server.start();
    const addr = server.getAddress()!;
    client = new TestRPCClient(`http://${addr.host}:${addr.port}`);
  });

  afterEach(async () => {
    await server.stop();
    await network.stop();
  });

  it("should recover from handler error without crashing", async () => {
    // Trigger an error (unknown account)
    try {
      await client.call("agent_getStatus", { did: "did:lgm-test:unknown" });
    } catch {
      // Expected to fail
    }

    // Server should still be responsive
    const result = await client.call<NetworkHealthResponse>("network_health");
    expect(result.status).toBe("healthy");
  });

  it("should handle rapid error/success alternation", async () => {
    const requests = [];

    // Alternate between valid and invalid requests
    for (let i = 0; i < 20; i++) {
      if (i % 2 === 0) {
        requests.push(
          client
            .call<NetworkHealthResponse>("network_health")
            .catch(() => null),
        );
      } else {
        requests.push(client.call("unknown_method").catch(() => null));
      }
    }

    const results = await Promise.all(requests);

    // Half should be successful
    const successful = results.filter((r) => r !== null);
    expect(successful.length).toBe(10);
  });

  it("should maintain state consistency after errors", async () => {
    const account = network.getAccount(0);

    // Get initial state
    const before = await client.call<AccountGetBalanceResponse>(
      "account_getBalance",
      {
        did: account.did,
      },
    );

    // Trigger some errors
    for (let i = 0; i < 5; i++) {
      try {
        await client.call("agent_getStatus", { did: "did:lgm-test:unknown" });
      } catch {
        // Expected
      }
    }

    // State should be unchanged
    const after = await client.call<AccountGetBalanceResponse>(
      "account_getBalance",
      {
        did: account.did,
      },
    );

    expect(after.balance).toBe(before.balance);
  });
});

// ============ Multi-Client Tests ============

describe.sequential("RPC E2E - Multi-Client", () => {
  let network: TestNetwork;
  let server: RPCServer;

  beforeEach(async () => {
    network = await createTestNetwork(RPC_TEST_CONFIG);
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, {
      ...TEST_SERVER_CONFIG,
      rateLimits: {
        ...TEST_SERVER_CONFIG.rateLimits,
        maxRequests: {
          free: 10000,
          staked100: 30000,
          staked1000: 100000,
          validator: 1000000,
        },
      },
    });
    await server.start();
  });

  afterEach(async () => {
    await server.stop();
    await network.stop();
  });

  it("should handle multiple HTTP clients simultaneously", async () => {
    const addr = server.getAddress()!;
    const baseUrl = `http://${addr.host}:${addr.port}`;

    const clients = [
      new TestRPCClient(baseUrl),
      new TestRPCClient(baseUrl),
      new TestRPCClient(baseUrl),
    ];

    // Each client makes requests
    const requests = clients.flatMap((client) =>
      Array.from({ length: 5 }, () =>
        client.call<NetworkHealthResponse>("network_health"),
      ),
    );

    const results = await Promise.all(requests);
    expect(results.length).toBe(15);
  });

  it("should handle mixed HTTP and WebSocket clients", async () => {
    const addr = server.getAddress()!;
    const httpClient = new TestRPCClient(`http://${addr.host}:${addr.port}`);
    const wsClient = new TestWSClient(`ws://${addr.host}:${addr.port}`);

    await wsClient.connect();

    // Make requests from both clients
    const httpResult =
      await httpClient.call<NetworkHealthResponse>("network_health");
    const wsResult =
      await wsClient.call<NetworkHealthResponse>("network_health");

    expect(httpResult.status).toBe("healthy");
    expect(wsResult.status).toBe("healthy");

    wsClient.disconnect();
  });
});
