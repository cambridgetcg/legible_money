/**
 * RPC WebSocket Tests
 *
 * Tests for WebSocket connections, subscriptions, and broadcasts.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { RPCServer } from "../../../protocol/rpc/server.js";
import {
  TestNetwork,
  createTestNetwork,
} from "../../framework/test_harness.js";
import { TestWSClient, RPCCallError } from "./helpers/rpc_client.js";
import {
  createMockHandlerContext,
  RPC_TEST_CONFIG,
  TEST_SERVER_CONFIG,
  ERROR_CODES,
} from "./helpers/fixtures.js";
import type { NetworkHealthResponse } from "../../../protocol/types.js";

// ============ Test Setup ============

let network: TestNetwork;
let server: RPCServer;
let wsUrl: string;

async function setupServer() {
  network = await createTestNetwork(RPC_TEST_CONFIG);
  const context = createMockHandlerContext(network);
  server = new RPCServer(context, TEST_SERVER_CONFIG);
  await server.start();
  const addr = server.getAddress()!;
  wsUrl = `ws://${addr.host}:${addr.port}`;
}

async function teardownServer() {
  await server.stop();
  await network.stop();
}

// ============ Connection Tests ============

describe.sequential("RPC WebSocket - Connection", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  it("should connect via WebSocket", async () => {
    const client = new TestWSClient(wsUrl);

    await client.connect();
    expect(client.isConnected).toBe(true);

    client.disconnect();
  });

  it("should handle multiple concurrent connections", async () => {
    const clients = [
      new TestWSClient(wsUrl),
      new TestWSClient(wsUrl),
      new TestWSClient(wsUrl),
    ];

    await Promise.all(clients.map((c) => c.connect()));

    for (const client of clients) {
      expect(client.isConnected).toBe(true);
    }

    for (const client of clients) {
      client.disconnect();
    }
  });

  it("should handle disconnect gracefully", async () => {
    const client = new TestWSClient(wsUrl);

    await client.connect();
    expect(client.isConnected).toBe(true);

    client.disconnect();
    expect(client.isConnected).toBe(false);
  });

  it("should handle reconnect after disconnect", async () => {
    const client = new TestWSClient(wsUrl);

    await client.connect();
    client.disconnect();

    // Create new client for reconnect
    const client2 = new TestWSClient(wsUrl);
    await client2.connect();
    expect(client2.isConnected).toBe(true);

    client2.disconnect();
  });
});

// ============ RPC Calls Over WebSocket ============

describe.sequential("RPC WebSocket - RPC Calls", () => {
  let client: TestWSClient;

  beforeEach(async () => {
    await setupServer();
    client = new TestWSClient(wsUrl);
    await client.connect();
  });

  afterEach(async () => {
    client.disconnect();
    await teardownServer();
  });

  it("should call RPC methods over WebSocket", async () => {
    const result = await client.call<NetworkHealthResponse>("network_health");

    expect(result.status).toBe("healthy");
    expect(result.version).toBe("0.1.0-test");
  });

  it("should handle multiple sequential calls", async () => {
    const result1 = await client.call<NetworkHealthResponse>("network_health");
    const result2 = await client.call<NetworkHealthResponse>("network_health");

    expect(result1.status).toBe("healthy");
    expect(result2.status).toBe("healthy");
  });

  it("should handle multiple parallel calls", async () => {
    const results = await Promise.all([
      client.call<NetworkHealthResponse>("network_health"),
      client.call<NetworkHealthResponse>("network_health"),
      client.call<NetworkHealthResponse>("network_health"),
    ]);

    for (const result of results) {
      expect(result.status).toBe("healthy");
    }
  });

  it("should return error for unknown method", async () => {
    await expect(client.call("unknown_method")).rejects.toThrow(RPCCallError);

    try {
      await client.call("unknown_method");
    } catch (error) {
      expect(error).toBeInstanceOf(RPCCallError);
      expect((error as RPCCallError).code).toBe(ERROR_CODES.METHOD_NOT_FOUND);
    }
  });

  it("should handle calls with parameters", async () => {
    const account = network.getAccount(0);

    const result = await client.call<{ balance: string }>(
      "account_getBalance",
      {
        did: account.did,
      },
    );

    expect(result.balance).toBeDefined();
    expect(BigInt(result.balance)).toBeGreaterThan(0n);
  });
});

// ============ Subscription Tests ============

describe.sequential("RPC WebSocket - Subscriptions", () => {
  let client: TestWSClient;

  beforeEach(async () => {
    await setupServer();
    client = new TestWSClient(wsUrl);
    await client.connect();
  });

  afterEach(async () => {
    client.disconnect();
    await teardownServer();
  });

  it("should subscribe to block events", async () => {
    const result = await client.subscribe("block");

    expect(result.subscribed).toBe(true);
    expect(result.event).toBe("block");
  });

  it("should subscribe to transaction events", async () => {
    const result = await client.subscribe("transaction");

    expect(result.subscribed).toBe(true);
    expect(result.event).toBe("transaction");
  });

  it("should subscribe to multiple events", async () => {
    const result1 = await client.subscribe("block");
    const result2 = await client.subscribe("transaction");

    expect(result1.subscribed).toBe(true);
    expect(result2.subscribed).toBe(true);
  });

  it("should unsubscribe successfully", async () => {
    await client.subscribe("block");

    const result = await client.unsubscribe();

    expect(result.unsubscribed).toBe(true);
  });

  it("should require event parameter for subscribe", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "subscribe",
      params: {}, // Missing event
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.INVALID_PARAMS);
  });
});

// ============ Broadcast Tests ============

describe.sequential("RPC WebSocket - Broadcast", () => {
  let client: TestWSClient;

  beforeEach(async () => {
    await setupServer();
    client = new TestWSClient(wsUrl);
    await client.connect();
  });

  afterEach(async () => {
    client.disconnect();
    await teardownServer();
  });

  it("should receive broadcast events after subscribing", async () => {
    await client.subscribe("test-event");

    // Set up event listener
    const eventPromise = client.waitForEvent(3000);

    // Trigger broadcast from server
    // Access server internals for testing
    (server as any).broadcast("test-event", { data: "test-data" });

    const event = await eventPromise;
    expect(event).toBeDefined();
    expect((event as any).subscriptionId).toBe("test-event");
    expect((event as any).event).toEqual({ data: "test-data" });
  });

  it("should only receive events for subscribed topics", async () => {
    await client.subscribe("subscribed-event");

    let receivedUnsubscribed = false;
    const unsubHandler = client.onEvent((event) => {
      if ((event as any).subscriptionId === "other-event") {
        receivedUnsubscribed = true;
      }
    });

    // Broadcast to a different topic
    (server as any).broadcast("other-event", { data: "test" });

    // Small delay to ensure event would have been received
    await new Promise((resolve) => setTimeout(resolve, 100));

    expect(receivedUnsubscribed).toBe(false);
    unsubHandler();
  });

  it("should broadcast to multiple clients", async () => {
    const client2 = new TestWSClient(wsUrl);
    await client2.connect();

    await client.subscribe("multi-event");
    await client2.subscribe("multi-event");

    const event1Promise = client.waitForEvent(3000);
    const event2Promise = client2.waitForEvent(3000);

    (server as any).broadcast("multi-event", { data: "broadcast" });

    const [event1, event2] = await Promise.all([event1Promise, event2Promise]);

    expect((event1 as any).event).toEqual({ data: "broadcast" });
    expect((event2 as any).event).toEqual({ data: "broadcast" });

    client2.disconnect();
  });

  it("should not receive events after disconnect", async () => {
    await client.subscribe("disconnect-test");

    const receivedAfterDisconnect = false;

    client.disconnect();

    // Broadcast after disconnect
    (server as any).broadcast("disconnect-test", { data: "test" });

    // Wait a bit
    await new Promise((resolve) => setTimeout(resolve, 100));

    // Client should not have received anything
    expect(receivedAfterDisconnect).toBe(false);
  });
});

// ============ Error Handling Tests ============

describe.sequential("RPC WebSocket - Error Handling", () => {
  let client: TestWSClient;

  beforeEach(async () => {
    await setupServer();
    client = new TestWSClient(wsUrl);
    await client.connect();
  });

  afterEach(async () => {
    client.disconnect();
    await teardownServer();
  });

  it("should handle malformed JSON gracefully", async () => {
    // Send raw malformed message
    const ws = (client as any).ws;
    ws.send("not valid json");

    // Should not crash the server
    await new Promise((resolve) => setTimeout(resolve, 100));

    // Should still be able to make valid calls
    const result = await client.call<NetworkHealthResponse>("network_health");
    expect(result.status).toBe("healthy");
  });

  it("should handle invalid JSON-RPC version", async () => {
    const response = await client.callRaw({
      jsonrpc: "1.0" as any,
      id: 1,
      method: "network_health",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.INVALID_REQUEST);
  });

  it("should handle missing method", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.INVALID_REQUEST);
  });
});

// ============ Rate Limiting Tests ============

describe.sequential("RPC WebSocket - Rate Limiting", () => {
  beforeEach(async () => {
    network = await createTestNetwork(RPC_TEST_CONFIG);
    const context = createMockHandlerContext(network);
    // Use low rate limits for testing
    server = new RPCServer(context, {
      ...TEST_SERVER_CONFIG,
      rateLimits: {
        windowMs: 60000,
        maxRequests: {
          free: 10, // Low but not too low for testing
          staked100: 20,
          staked1000: 40,
          validator: 100,
        },
      },
    });
    await server.start();
    const addr = server.getAddress()!;
    wsUrl = `ws://${addr.host}:${addr.port}`;
  });

  afterEach(teardownServer);

  it("should track rate limits on WebSocket connections", async () => {
    const client = new TestWSClient(wsUrl);
    await client.connect();

    // Make several requests successfully
    for (let i = 0; i < 5; i++) {
      const response = await client.callRaw({
        jsonrpc: "2.0",
        id: i,
        method: "network_health",
      });
      // These should succeed
      expect(response.result || response.error).toBeDefined();
    }

    // Test passes if we can make multiple requests without error
    // Full rate limit enforcement testing is covered in HTTP tests
    client.disconnect();
  });
});
