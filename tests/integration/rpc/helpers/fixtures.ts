/**
 * RPC Test Fixtures
 *
 * Test data and helper functions for RPC server integration tests.
 */

import type { TestNetwork } from "../../../framework/test_harness.js";
import type { HandlerContext } from "../../../../protocol/rpc/handlers.js";
import type { StateIntegration } from "../../../../consensus/pot/state_integration.js";
import type { Transaction } from "../../../../consensus/pot/types.js";
import type { PublicKey } from "../../../../core/crypto/keys.js";

// ============ Context Factory ============

/**
 * Create a mock HandlerContext from a TestNetwork
 */
export function createMockHandlerContext(network: TestNetwork): HandlerContext {
  const stateManager = network.getState();
  const consensus = network.getConsensus();

  // Create a minimal state integration wrapper
  const stateIntegration: StateIntegration = {
    getStateManager: () => stateManager,
    computeStateRoot: () => "test-state-root",
    computeKnowledgeRoot: () => "test-knowledge-root",
    validateTransaction: async () => ({
      valid: true,
      error: undefined,
      fee: 1000000n,
    }),
    executeTransaction: async (tx: Transaction, blockNumber: bigint) => ({
      success: true,
      stateRoot: "test-root",
      error: undefined,
    }),
    applyVerificationResult: async () => {},
    collectTransactions: () => [],
    addToMempool: () => true,
    applyBlock: async () => "test-state-root",
    getBalance: (did: string) => stateManager.getAccount(did)?.balance ?? 0n,
    getNonce: (did: string) => stateManager.getAccount(did)?.nonce ?? 0n,
    isActiveValidator: (publicKey: PublicKey) =>
      stateManager.getValidator(publicKey)?.isActive ?? false,
    getValidatorStake: (publicKey: PublicKey) =>
      stateManager.getValidator(publicKey)?.stake ?? 0n,
    getTotalStake: () => {
      let total = 0n;
      // Sum up validator stakes from network
      for (const validator of network.validators) {
        total += validator.stake;
      }
      return total;
    },
    initialize: async () => true,
    hasPersistence: () => false,
  };

  return {
    state: stateManager,
    consensus,
    stateIntegration,
    nodeVersion: "0.1.0-test",
    networkId: "test-network",
  };
}

// ============ Test DIDs ============

/**
 * Well-known test DIDs for use in tests
 */
export const TEST_DIDS = [
  "did:lgm-test:account-0",
  "did:lgm-test:account-1",
  "did:lgm-test:account-2",
  "did:lgm-test:account-3",
  "did:lgm-test:account-4",
] as const;

export const TEST_VALIDATOR_DIDS = [
  "did:lgm-test:validator-0",
  "did:lgm-test:validator-1",
  "did:lgm-test:validator-2",
] as const;

// ============ Test Claims ============

/**
 * Sample claims for testing knowledge submission
 */
export const TEST_CLAIMS = [
  {
    claim: "Water boils at 100 degrees Celsius at sea level",
    domain: "science.chemistry",
  },
  {
    claim: "The speed of light in vacuum is approximately 299,792,458 m/s",
    domain: "science.physics",
  },
  {
    claim: "The Pythagorean theorem states that a^2 + b^2 = c^2",
    domain: "mathematics.geometry",
  },
  {
    claim: "Bitcoin was created by Satoshi Nakamoto in 2009",
    domain: "technology.cryptocurrency",
  },
] as const;

// ============ Test Network Config ============

/**
 * Standard test network configuration for RPC tests
 */
export const RPC_TEST_CONFIG = {
  validatorCount: 3,
  fullNodeCount: 0,
  testAccounts: 5,
  autoBlocks: false,
  blockInterval: 100,
  deterministicMode: true,
  totalSupply: 1_000_000_000n * 10n ** 18n,
  validatorStake: 10_000n * 10n ** 18n,
} as const;

// ============ Server Config ============

/**
 * RPC server configuration for tests
 * Uses port 0 to let OS assign available port
 */
export const TEST_SERVER_CONFIG = {
  port: 0, // Random available port
  host: "127.0.0.1",
  enableWebSocket: true,
  maxRequestSize: 1024 * 1024,
  rateLimits: {
    windowMs: 60000,
    maxRequests: {
      free: 1000, // High limit for tests
      staked100: 3000,
      staked1000: 10000,
      validator: 100000,
    },
  },
  cors: {
    enabled: false,
    origins: [],
  },
} as const;

// ============ JSON-RPC Helpers ============

/**
 * Create a valid JSON-RPC request
 */
export function createRPCRequest(
  method: string,
  params?: unknown,
  id: number | string = 1,
): { jsonrpc: "2.0"; id: number | string; method: string; params?: unknown } {
  const request: {
    jsonrpc: "2.0";
    id: number | string;
    method: string;
    params?: unknown;
  } = {
    jsonrpc: "2.0",
    id,
    method,
  };
  if (params !== undefined) {
    request.params = params;
  }
  return request;
}

/**
 * Standard error codes for reference in tests
 */
export const ERROR_CODES = {
  PARSE_ERROR: -32700,
  INVALID_REQUEST: -32600,
  METHOD_NOT_FOUND: -32601,
  INVALID_PARAMS: -32602,
  INTERNAL_ERROR: -32603,
  ACCOUNT_NOT_FOUND: -32001,
  INSUFFICIENT_BALANCE: -32002,
  INVALID_SIGNATURE: -32003,
  INVALID_NONCE: -32004,
  SESSION_EXPIRED: -32005,
  SESSION_LIMIT_EXCEEDED: -32006,
  RATE_LIMITED: -32007,
} as const;

// ============ Assertion Helpers ============

/**
 * Check if a response is a successful RPC response
 */
export function isSuccessResponse(response: unknown): boolean {
  return (
    typeof response === "object" &&
    response !== null &&
    "jsonrpc" in response &&
    response.jsonrpc === "2.0" &&
    "result" in response &&
    !("error" in response)
  );
}

/**
 * Check if a response is an error RPC response
 */
export function isErrorResponse(response: unknown): boolean {
  return (
    typeof response === "object" &&
    response !== null &&
    "jsonrpc" in response &&
    response.jsonrpc === "2.0" &&
    "error" in response
  );
}
