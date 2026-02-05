/**
 * RPC Server Lifecycle Tests
 *
 * Tests for server start, stop, restart, and basic request handling.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { RPCServer } from "../../../protocol/rpc/server.js";
import {
  TestNetwork,
  createTestNetwork,
} from "../../framework/test_harness.js";
import { TestRPCClient } from "./helpers/rpc_client.js";
import {
  createMockHandlerContext,
  RPC_TEST_CONFIG,
  TEST_SERVER_CONFIG,
  ERROR_CODES,
} from "./helpers/fixtures.js";

// ============ Server Lifecycle Tests ============

describe.sequential("RPC Server - Lifecycle", () => {
  let network: TestNetwork;
  let server: RPCServer;

  beforeEach(async () => {
    network = await createTestNetwork(RPC_TEST_CONFIG);
  });

  afterEach(async () => {
    if (server) {
      await server.stop();
    }
    await network.stop();
  });

  it("should start on configured port", async () => {
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, TEST_SERVER_CONFIG);

    await server.start();

    const addr = server.getAddress();
    expect(addr).not.toBeNull();
    expect(addr!.port).toBeGreaterThan(0);
    expect(addr!.host).toBe("127.0.0.1");
  });

  it("should accept connections after start", async () => {
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, TEST_SERVER_CONFIG);

    await server.start();

    const addr = server.getAddress()!;
    const client = new TestRPCClient(`http://${addr.host}:${addr.port}`);

    const result = await client.call<{ status: string }>("network_health");
    expect(result.status).toBe("healthy");
  });

  it("should stop gracefully", async () => {
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, TEST_SERVER_CONFIG);

    await server.start();
    const addr = server.getAddress()!;

    await server.stop();

    // Server should no longer accept connections
    const client = new TestRPCClient(`http://${addr.host}:${addr.port}`);
    await expect(client.call("network_health")).rejects.toThrow();
  });

  it("should handle restart", async () => {
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, TEST_SERVER_CONFIG);

    // Start
    await server.start();
    const addr1 = server.getAddress()!;

    // Stop
    await server.stop();

    // Create new server instance for restart (simulates restart)
    server = new RPCServer(context, TEST_SERVER_CONFIG);
    await server.start();
    const addr2 = server.getAddress()!;

    // Should be able to make requests
    const client = new TestRPCClient(`http://${addr2.host}:${addr2.port}`);
    const result = await client.call<{ status: string }>("network_health");
    expect(result.status).toBe("healthy");
  });
});

// ============ HTTP Method Tests ============

describe.sequential("RPC Server - HTTP Methods", () => {
  let network: TestNetwork;
  let server: RPCServer;
  let baseUrl: string;

  beforeEach(async () => {
    network = await createTestNetwork(RPC_TEST_CONFIG);
    const context = createMockHandlerContext(network);
    server = new RPCServer(context, TEST_SERVER_CONFIG);
    await server.start();
    const addr = server.getAddress()!;
    baseUrl = `http://${addr.host}:${addr.port}`;
  });

  afterEach(async () => {
    await server.stop();
    await network.stop();
  });

  it("should reject GET requests with 405", async () => {
    const response = await fetch(baseUrl, { method: "GET" });
    expect(response.status).toBe(405);
  });

  it("should reject PUT requests with 405", async () => {
    const response = await fetch(baseUrl, { method: "PUT" });
    expect(response.status).toBe(405);
  });

  it("should reject DELETE requests with 405", async () => {
    const response = await fetch(baseUrl, { method: "DELETE" });
    expect(response.status).toBe(405);
  });

  it("should handle OPTIONS (preflight)", async () => {
    const response = await fetch(baseUrl, { method: "OPTIONS" });
    expect(response.status).toBe(204);
  });

  it("should accept POST requests", async () => {
    const response = await fetch(baseUrl, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        method: "network_health",
      }),
    });
    expect(response.status).toBe(200);
  });
});

// ============ JSON-RPC Validation Tests ============

describe.sequential("RPC Server - JSON-RPC Validation", () => {
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

  it("should reject invalid JSON with parse error (-32700)", async () => {
    const response = await client.rawRequest({
      body: "not valid json",
    });

    const data = await response.json();
    expect(data.error).toBeDefined();
    expect(data.error.code).toBe(ERROR_CODES.PARSE_ERROR);
  });

  it("should reject invalid JSON-RPC version with -32600", async () => {
    const response = await client.callRaw({
      jsonrpc: "1.0" as any,
      id: 1,
      method: "network_health",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.INVALID_REQUEST);
  });

  it("should reject missing method with -32600", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.INVALID_REQUEST);
  });

  it("should return method not found for unknown method (-32601)", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "unknown_method_that_does_not_exist",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.METHOD_NOT_FOUND);
  });

  it("should include request id in response", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 42,
      method: "network_health",
    });

    expect(response.id).toBe(42);
  });

  it("should include request id in error response", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 123,
      method: "unknown_method",
    });

    expect(response.id).toBe(123);
    expect(response.error).toBeDefined();
  });
});

// ============ Rate Limiting Tests ============

describe.sequential("RPC Server - Rate Limiting Headers", () => {
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

  it("should include rate limit headers in response", async () => {
    const info = await client.getRateLimitInfo();

    expect(info.limit).toBeGreaterThan(0);
    expect(info.remaining).toBeGreaterThanOrEqual(0);
    expect(info.reset).toBeGreaterThan(0);
    expect(info.tier).toBe("free");
  });

  it("should decrement remaining count", async () => {
    const info1 = await client.getRateLimitInfo();
    const info2 = await client.getRateLimitInfo();

    expect(info2.remaining).toBeLessThan(info1.remaining);
  });
});

// ============ Request Size Tests ============

describe.sequential("RPC Server - Request Size Limits", () => {
  let network: TestNetwork;
  let server: RPCServer;
  let client: TestRPCClient;

  beforeEach(async () => {
    network = await createTestNetwork(RPC_TEST_CONFIG);
    const context = createMockHandlerContext(network);
    // Use smaller max request size for testing
    server = new RPCServer(context, {
      ...TEST_SERVER_CONFIG,
      maxRequestSize: 1024, // 1KB
    });
    await server.start();
    const addr = server.getAddress()!;
    client = new TestRPCClient(`http://${addr.host}:${addr.port}`);
  });

  afterEach(async () => {
    await server.stop();
    await network.stop();
  });

  it("should reject oversized requests", async () => {
    const largePayload = "x".repeat(2048); // 2KB

    const response = await client.rawRequest({
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        method: "network_health",
        params: { data: largePayload },
      }),
    });

    // Server should reject or return error
    expect(response.status === 400 || response.status === 413).toBe(true);
  });
});
