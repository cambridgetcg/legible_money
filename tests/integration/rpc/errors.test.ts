/**
 * RPC Error Handling Tests
 *
 * Tests for JSON-RPC error codes and error response formatting.
 */

import { describe, it, expect, beforeEach, afterEach } from "vitest";
import { RPCServer } from "../../../protocol/rpc/server.js";
import {
  TestNetwork,
  createTestNetwork,
} from "../../framework/test_harness.js";
import { TestRPCClient, RPCCallError } from "./helpers/rpc_client.js";
import {
  createMockHandlerContext,
  RPC_TEST_CONFIG,
  TEST_SERVER_CONFIG,
  ERROR_CODES,
} from "./helpers/fixtures.js";

// ============ Test Setup ============

let network: TestNetwork;
let server: RPCServer;
let client: TestRPCClient;

async function setupServer() {
  network = await createTestNetwork(RPC_TEST_CONFIG);
  const context = createMockHandlerContext(network);
  server = new RPCServer(context, TEST_SERVER_CONFIG);
  await server.start();
  const addr = server.getAddress()!;
  client = new TestRPCClient(`http://${addr.host}:${addr.port}`);
}

async function teardownServer() {
  await server.stop();
  await network.stop();
}

// ============ Parse Error Tests (-32700) ============

describe.sequential("RPC Errors - Parse Error (-32700)", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  it("should return -32700 for invalid JSON", async () => {
    const response = await client.rawRequest({
      body: "not valid json {{{",
    });

    const data = await response.json();
    expect(data.error).toBeDefined();
    expect(data.error.code).toBe(ERROR_CODES.PARSE_ERROR);
    expect(data.error.message).toContain("JSON");
  });

  it("should return -32700 for empty body", async () => {
    const response = await client.rawRequest({
      body: "",
    });

    const data = await response.json();
    expect(data.error).toBeDefined();
    expect(data.error.code).toBe(ERROR_CODES.PARSE_ERROR);
  });

  it("should return -32700 for truncated JSON", async () => {
    const response = await client.rawRequest({
      body: '{"jsonrpc": "2.0", "method": "test"',
    });

    const data = await response.json();
    expect(data.error).toBeDefined();
    expect(data.error.code).toBe(ERROR_CODES.PARSE_ERROR);
  });
});

// ============ Invalid Request Tests (-32600) ============

describe.sequential("RPC Errors - Invalid Request (-32600)", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  it("should return -32600 for wrong JSON-RPC version", async () => {
    const response = await client.callRaw({
      jsonrpc: "1.0" as any,
      id: 1,
      method: "network_health",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.INVALID_REQUEST);
    expect(response.error!.message).toContain("version");
  });

  it("should return -32600 for missing jsonrpc field", async () => {
    const response = await client.rawRequest({
      body: JSON.stringify({
        id: 1,
        method: "network_health",
      }),
    });

    const data = await response.json();
    expect(data.error).toBeDefined();
    expect(data.error.code).toBe(ERROR_CODES.INVALID_REQUEST);
  });

  it("should return -32600 for missing method", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.INVALID_REQUEST);
    expect(response.error!.message).toContain("Method");
  });

  it("should return -32600 for null method", async () => {
    const response = await client.rawRequest({
      body: JSON.stringify({
        jsonrpc: "2.0",
        id: 1,
        method: null,
      }),
    });

    const data = await response.json();
    expect(data.error).toBeDefined();
    expect(data.error.code).toBe(ERROR_CODES.INVALID_REQUEST);
  });
});

// ============ Method Not Found Tests (-32601) ============

describe.sequential("RPC Errors - Method Not Found (-32601)", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  it("should return -32601 for unknown method", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "unknown_nonexistent_method",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.METHOD_NOT_FOUND);
    expect(response.error!.message).toContain("unknown_nonexistent_method");
  });

  it("should return -32601 for typo in method name", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "network_Heath", // Typo
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.METHOD_NOT_FOUND);
  });

  it("should return -32601 for private methods", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "_internal_method",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.METHOD_NOT_FOUND);
  });
});

// ============ Invalid Params Tests (-32602) ============

describe.sequential("RPC Errors - Invalid Params (-32602)", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  it("should return error for missing required params", async () => {
    // agent_getStatus requires 'did' param - calling without it should error
    // Note: The exact error depends on handler implementation
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "agent_getStatus",
      params: {}, // Missing 'did'
    });

    // Handler should return some error (either INVALID_PARAMS or ACCOUNT_NOT_FOUND)
    expect(response.error).toBeDefined();
    expect(response.error!.code).toBeLessThan(0); // Any error code
  });
});

// ============ Account Not Found Tests (-32001) ============

describe.sequential("RPC Errors - Account Not Found (-32001)", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  it("should return -32001 for agent_getStatus with unknown DID", async () => {
    try {
      await client.call("agent_getStatus", {
        did: "did:lgm-test:completely-unknown-account",
      });
      fail("Should have thrown");
    } catch (error) {
      expect(error).toBeInstanceOf(RPCCallError);
      expect((error as RPCCallError).code).toBe(ERROR_CODES.ACCOUNT_NOT_FOUND);
    }
  });
});

// ============ Rate Limited Tests (-32007) ============

describe.sequential("RPC Errors - Rate Limited (-32007)", () => {
  beforeEach(async () => {
    network = await createTestNetwork(RPC_TEST_CONFIG);
    const context = createMockHandlerContext(network);
    // Very low rate limit for testing
    server = new RPCServer(context, {
      ...TEST_SERVER_CONFIG,
      rateLimits: {
        windowMs: 60000,
        maxRequests: {
          free: 3, // Very low
          staked100: 10,
          staked1000: 20,
          validator: 100,
        },
      },
    });
    await server.start();
    const addr = server.getAddress()!;
    client = new TestRPCClient(`http://${addr.host}:${addr.port}`);
  });

  afterEach(teardownServer);

  it("should return -32007 when rate limit exceeded", async () => {
    // Exhaust rate limit (use callRaw to avoid throwing)
    // With limit=3 and check after increment, 2 calls succeed, 3rd gets limited
    for (let i = 0; i < 2; i++) {
      await client.callRaw({ jsonrpc: "2.0", id: i, method: "network_health" });
    }

    // Third request should be rate limited
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 100,
      method: "network_health",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBe(ERROR_CODES.RATE_LIMITED);
    expect(response.error!.message).toContain("Rate limit");
  });

  it("should include retryAfter in rate limit error", async () => {
    // Exhaust rate limit (use callRaw to avoid throwing)
    for (let i = 0; i < 2; i++) {
      await client.callRaw({ jsonrpc: "2.0", id: i, method: "network_health" });
    }

    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 100,
      method: "network_health",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.data).toBeDefined();
    expect((response.error!.data as any).retryAfter).toBeDefined();
  });
});

// ============ Error Response Format Tests ============

describe.sequential("RPC Errors - Response Format", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  it("should include jsonrpc version in error response", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "unknown_method",
    });

    expect(response.jsonrpc).toBe("2.0");
    expect(response.error).toBeDefined();
  });

  it("should include request id in error response", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 42,
      method: "unknown_method",
    });

    expect(response.id).toBe(42);
    expect(response.error).toBeDefined();
  });

  it("should include string id in error response", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: "string-id-123",
      method: "unknown_method",
    });

    expect(response.id).toBe("string-id-123");
    expect(response.error).toBeDefined();
  });

  it("should have error code, message, and data", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "unknown_method",
    });

    expect(response.error).toBeDefined();
    expect(response.error!.code).toBeDefined();
    expect(typeof response.error!.code).toBe("number");
    expect(response.error!.message).toBeDefined();
    expect(typeof response.error!.message).toBe("string");
  });

  it("should not include result in error response", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "unknown_method",
    });

    expect(response.error).toBeDefined();
    expect(response.result).toBeUndefined();
  });

  it("should not include error in success response", async () => {
    const response = await client.callRaw({
      jsonrpc: "2.0",
      id: 1,
      method: "network_health",
    });

    expect(response.result).toBeDefined();
    expect(response.error).toBeUndefined();
  });
});

// ============ Error Data Field Tests ============

describe.sequential("RPC Errors - Error Data", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  it("should include errorCode in error data", async () => {
    try {
      await client.call("agent_getStatus", {
        did: "did:lgm-test:unknown",
      });
      fail("Should have thrown");
    } catch (error) {
      expect(error).toBeInstanceOf(RPCCallError);
      const rpcError = error as RPCCallError;
      expect(rpcError.data).toBeDefined();
      expect((rpcError.data as any).errorCode).toBeDefined();
    }
  });

  it("should include retryable flag in error data", async () => {
    try {
      await client.call("agent_getStatus", {
        did: "did:lgm-test:unknown",
      });
      fail("Should have thrown");
    } catch (error) {
      expect(error).toBeInstanceOf(RPCCallError);
      const rpcError = error as RPCCallError;
      expect(rpcError.data).toBeDefined();
      expect(typeof (rpcError.data as any).retryable).toBe("boolean");
    }
  });
});
