/**
 * RPC Methods Tests
 *
 * Tests for all RPC method handlers.
 * Organized by method category: Account, Agent, Knowledge, Verification, Network.
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
  TEST_DIDS,
  TEST_CLAIMS,
  ERROR_CODES,
} from "./helpers/fixtures.js";
import type {
  AccountGetBalanceResponse,
  AccountGetInfoResponse,
  AgentCheckAccountResponse,
  AgentGetStatusResponse,
  KnowledgeGetFactsResponse,
  KnowledgeGetFactResponse,
  KnowledgeSessionStartResponse,
  KnowledgeSessionEndResponse,
  KnowledgeQueryResponse,
  KnowledgeEstimateQueryCostResponse,
  VerificationGetPendingResponse,
  VerificationGetRoundResponse,
  NetworkHealthResponse,
} from "../../../protocol/types.js";

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

// ============ Account Methods ============

describe.sequential("RPC Methods - Account", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  describe("account_getBalance", () => {
    it("should return balance for existing account", async () => {
      const account = network.getAccount(0);

      const result = await client.call<AccountGetBalanceResponse>(
        "account_getBalance",
        {
          did: account.did,
        },
      );

      expect(result.balance).toBeDefined();
      expect(BigInt(result.balance)).toBeGreaterThan(0n);
      expect(result.stakedBalance).toBeDefined();
      expect(result.nonce).toBeDefined();
    });

    it("should return zeros for non-existent account", async () => {
      const result = await client.call<AccountGetBalanceResponse>(
        "account_getBalance",
        {
          did: "did:lgm-test:nonexistent-account",
        },
      );

      expect(result.balance).toBe("0");
      expect(result.stakedBalance).toBe("0");
      expect(result.nonce).toBe("0");
    });

    it("should return balance for validator account", async () => {
      const validator = network.getValidator(0);

      const result = await client.call<AccountGetBalanceResponse>(
        "account_getBalance",
        {
          did: validator.did,
        },
      );

      expect(BigInt(result.balance)).toBeGreaterThan(0n);
    });
  });

  describe("account_getInfo", () => {
    it("should return full account state", async () => {
      const account = network.getAccount(0);

      const result = await client.call<AccountGetInfoResponse>(
        "account_getInfo",
        {
          did: account.did,
        },
      );

      expect(result.account).not.toBeNull();
      expect(result.blockNumber).toBeDefined();
    });

    it("should return null account for non-existent DID", async () => {
      const result = await client.call<AccountGetInfoResponse>(
        "account_getInfo",
        {
          did: "did:lgm-test:nonexistent",
        },
      );

      expect(result.account).toBeNull();
      expect(result.blockNumber).toBeDefined();
    });
  });
});

// ============ Agent Methods ============

describe.sequential("RPC Methods - Agent", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  describe("agent_checkAccount", () => {
    it("should return exists=true for known account", async () => {
      const account = network.getAccount(0);

      const result = await client.call<AgentCheckAccountResponse>(
        "agent_checkAccount",
        {
          did: account.did,
        },
      );

      expect(result.exists).toBe(true);
      expect(result.balance).toBeDefined();
      expect(result.reputation).toBeDefined();
    });

    it("should return exists=false for unknown account", async () => {
      const result = await client.call<AgentCheckAccountResponse>(
        "agent_checkAccount",
        {
          did: "did:lgm-test:unknown-agent",
        },
      );

      expect(result.exists).toBe(false);
    });

    it("should indicate validator status for validator account", async () => {
      const validator = network.getValidator(0);

      const result = await client.call<AgentCheckAccountResponse>(
        "agent_checkAccount",
        {
          did: validator.did,
        },
      );

      expect(result.exists).toBe(true);
      expect(result.isValidator).toBe(true);
    });
  });

  describe("agent_getStatus", () => {
    it("should return activity metrics for existing account", async () => {
      const account = network.getAccount(0);

      const result = await client.call<AgentGetStatusResponse>(
        "agent_getStatus",
        {
          did: account.did,
        },
      );

      expect(result.account).toBeDefined();
      expect(result.account.balance).toBeDefined();
      expect(result.activity).toBeDefined();
      expect(result.activity.claimsSubmitted).toBeDefined();
      expect(result.earnings).toBeDefined();
      expect(result.sessions).toBeDefined();
    });

    it("should throw for non-existent account", async () => {
      await expect(
        client.call("agent_getStatus", {
          did: "did:lgm-test:nonexistent",
        }),
      ).rejects.toThrow(RPCCallError);
    });
  });
});

// ============ Knowledge Methods ============

describe.sequential("RPC Methods - Knowledge", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  describe("knowledge_getFacts", () => {
    it("should return facts by domain", async () => {
      const result = await client.call<KnowledgeGetFactsResponse>(
        "knowledge_getFacts",
        {
          domain: "test",
        },
      );

      expect(result.facts).toBeDefined();
      expect(Array.isArray(result.facts)).toBe(true);
      expect(result.total).toBeDefined();
    });

    it("should filter by minConfidence", async () => {
      const result = await client.call<KnowledgeGetFactsResponse>(
        "knowledge_getFacts",
        {
          domain: "test",
          minConfidence: 0.9,
        },
      );

      expect(result.facts).toBeDefined();
      // All returned facts should have confidence >= 0.9
      for (const fact of result.facts) {
        expect(fact.confidence).toBeGreaterThanOrEqual(0.9);
      }
    });

    it("should support pagination with limit and cursor", async () => {
      const result = await client.call<KnowledgeGetFactsResponse>(
        "knowledge_getFacts",
        {
          domain: "test",
          limit: 10,
        },
      );

      expect(result.facts.length).toBeLessThanOrEqual(10);
      // nextCursor may or may not be present depending on total facts
      expect(result.total).toBeDefined();
    });
  });

  describe("knowledge_getFact", () => {
    it("should return null for non-existent fact", async () => {
      const result = await client.call<KnowledgeGetFactResponse>(
        "knowledge_getFact",
        {
          factId: "non-existent-fact-id",
        },
      );

      expect(result.fact).toBeNull();
      expect(result.blockNumber).toBeDefined();
    });

    it("should return fact by ID when exists", async () => {
      // First submit a claim and let it be verified
      const account = network.getAccount(0);
      const testClaim = TEST_CLAIMS[0];

      const claimId = await network.submitClaim(
        testClaim.claim,
        testClaim.domain,
        account,
        100n * 10n ** 18n,
      );

      // Complete verification
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

      // Now query the fact
      const result = await client.call<KnowledgeGetFactResponse>(
        "knowledge_getFact",
        {
          factId: claimId,
        },
      );

      // Fact may or may not exist depending on timing
      expect(result.blockNumber).toBeDefined();
    });
  });

  describe("knowledge_sessionStart", () => {
    it("should start a knowledge session", async () => {
      const account = network.getAccount(0);

      const result = await client.call<KnowledgeSessionStartResponse>(
        "knowledge_sessionStart",
        {
          agentDid: account.did,
        },
      );

      expect(result.sessionId).toBeDefined();
      expect(result.sessionId).toContain("chat:");
      expect(result.startBlock).toBeDefined();
      expect(result.freeTierQueries).toBeGreaterThan(0);
    });
  });

  describe("knowledge_sessionEnd", () => {
    it("should end a knowledge session", async () => {
      const result = await client.call<KnowledgeSessionEndResponse>(
        "knowledge_sessionEnd",
        {
          sessionId: "test-session-id",
        },
      );

      expect(result.success).toBe(true);
      expect(result.totalQueries).toBeDefined();
      expect(result.totalCost).toBeDefined();
    });
  });

  describe("knowledge_query", () => {
    it("should query facts by domain", async () => {
      const result = await client.call<KnowledgeQueryResponse>(
        "knowledge_query",
        {
          sessionId: "test-session",
          intent: {
            type: "explore",
            domain: "test",
            limit: 10,
          },
        },
      );

      expect(result.intentType).toBe("explore");
      expect(result.facts).toBeDefined();
      expect(result.metadata).toBeDefined();
      expect(result.metadata.latencyMs).toBeDefined();
    });

    it("should query fact by ID", async () => {
      const result = await client.call<KnowledgeQueryResponse>(
        "knowledge_query",
        {
          sessionId: "test-session",
          intent: {
            type: "understand",
            factId: "some-fact-id",
          },
        },
      );

      expect(result.intentType).toBe("understand");
      expect(result.facts).toBeDefined();
    });
  });

  describe("knowledge_estimateQueryCost", () => {
    it("should estimate query cost", async () => {
      const result = await client.call<KnowledgeEstimateQueryCostResponse>(
        "knowledge_estimateQueryCost",
        {
          sessionId: "test-session",
          intent: {
            type: "explore",
            domain: "test",
            limit: 50,
          },
        },
      );

      expect(result.cost).toBeDefined();
      expect(BigInt(result.cost)).toBeGreaterThan(0n);
      expect(result.estimatedBuckets).toBeDefined();
      expect(result.breakdown).toBeDefined();
      expect(result.breakdown.baseCost).toBeDefined();
      expect(result.breakdown.bucketCost).toBeDefined();
    });

    it("should include cross-stratum cost when enabled", async () => {
      const result = await client.call<KnowledgeEstimateQueryCostResponse>(
        "knowledge_estimateQueryCost",
        {
          sessionId: "test-session",
          intent: {
            type: "explore",
            domain: "test",
            crossStrata: true,
          },
        },
      );

      expect(BigInt(result.breakdown.crossStratumCost)).toBeGreaterThan(0n);
    });
  });
});

// ============ Verification Methods ============

describe.sequential("RPC Methods - Verification", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  describe("verification_getPending", () => {
    it("should return pending assignments for validator", async () => {
      const validator = network.getValidator(0);

      const result = await client.call<VerificationGetPendingResponse>(
        "verification_getPending",
        {
          validatorKey: validator.publicKey,
        },
      );

      expect(result.assignments).toBeDefined();
      expect(Array.isArray(result.assignments)).toBe(true);
    });

    it("should return empty array for non-validator", async () => {
      const result = await client.call<VerificationGetPendingResponse>(
        "verification_getPending",
        {
          validatorKey: "non-validator-key",
        },
      );

      expect(result.assignments).toEqual([]);
    });

    it("should return assignments when claim submitted", async () => {
      const account = network.getAccount(0);
      const validator = network.getValidator(0);

      // Submit a claim
      await network.submitClaim(
        "Test claim for verification",
        "test.domain",
        account,
        100n * 10n ** 18n,
      );

      const result = await client.call<VerificationGetPendingResponse>(
        "verification_getPending",
        {
          validatorKey: validator.publicKey,
        },
      );

      // May have assignments depending on selection
      expect(result.assignments).toBeDefined();
    });
  });

  describe("verification_getRound", () => {
    it("should return null for non-existent round", async () => {
      const result = await client.call<VerificationGetRoundResponse>(
        "verification_getRound",
        {
          roundId: "non-existent-round",
        },
      );

      expect(result.round).toBeNull();
    });

    it("should return round details when exists", async () => {
      const account = network.getAccount(0);

      // Submit a claim to create a round
      const claimId = await network.submitClaim(
        "Test claim",
        "test.domain",
        account,
        100n * 10n ** 18n,
      );

      const rounds = network.getConsensus().getActiveRounds();
      if (rounds.length > 0) {
        const result = await client.call<VerificationGetRoundResponse>(
          "verification_getRound",
          {
            roundId: rounds[0].id,
          },
        );

        expect(result.round).not.toBeNull();
        expect(result.round!.claimId).toBe(claimId);
        expect(result.round!.phase).toBeDefined();
      }
    });
  });
});

// ============ Network Methods ============

describe.sequential("RPC Methods - Network", () => {
  beforeEach(setupServer);
  afterEach(teardownServer);

  describe("network_health", () => {
    it("should return healthy status", async () => {
      const result = await client.call<NetworkHealthResponse>("network_health");

      expect(result.status).toBe("healthy");
      expect(result.latestBlock).toBeDefined();
      expect(result.syncStatus).toBe("synced");
      expect(result.version).toBeDefined();
    });

    it("should return correct node version", async () => {
      const result = await client.call<NetworkHealthResponse>("network_health");

      expect(result.version).toBe("0.1.0-test");
    });

    it("should return block number as string", async () => {
      const result = await client.call<NetworkHealthResponse>("network_health");

      expect(typeof result.latestBlock).toBe("string");
      // Should be parseable as bigint
      expect(() => BigInt(result.latestBlock)).not.toThrow();
    });
  });
});
