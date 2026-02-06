/**
 * RPC Handlers Unit Tests
 *
 * Tests all 12 RPC handler functions with mocked dependencies.
 * Covers: account, agent, knowledge, verification, and network handlers.
 *
 * PP HIERARCHY:
 *   TRUTH: Verify handlers return correct responses for given inputs
 *   UNDERSTANDING: Test all handler paths including error cases
 *   BEAUTY: Clear test structure mirroring handler categories
 *   JUSTICE: Tests in proper location under protocol/rpc
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import {
  accountGetBalance,
  accountGetInfo,
  agentCheckAccount,
  agentGetStatus,
  knowledgeGetFacts,
  knowledgeGetFact,
  knowledgeSubmitClaim,
  verificationGetPending,
  verificationSubmitCommitment,
  verificationSubmitReveal,
  verificationGetRound,
  networkHealth,
  handlers,
  type HandlerContext,
} from "../../../../protocol/rpc/handlers.js";
import { RPC_ERROR_CODES } from "../../../../protocol/types.js";
import type { AccountState, FactState } from "../../../../state/types.js";

// ============ Mock Setup ============

function createMockAccount(
  overrides: Partial<AccountState> = {},
): AccountState {
  return {
    did: "did:lgm:test-account-12345678",
    balance: 10000n * 10n ** 18n,
    stakedBalance: 1000n * 10n ** 18n,
    nonce: 5n,
    operationalKeyHash:
      "abc123def456789012345678901234567890123456789012345678901234",
    operationalKeyVersion: 1,
    sessionKeyCount: 2,
    reputationScore: 0.85,
    lastActiveBlock: 100n,
    flags: {
      isValidator: false,
      canSubmitClaims: true,
      canChallenge: true,
      inRecovery: false,
      frozen: false,
    },
    ...overrides,
  };
}

function createMockFact(overrides: Partial<FactState> = {}): FactState {
  return {
    id: "fact-123456",
    claim: "The Earth orbits the Sun",
    domain: "astronomy",
    confidence: 0.95,
    verifiedAt: 100n,
    verifierCount: 10,
    challengeCount: 0,
    status: "active",
    ...overrides,
  };
}

function createMockContext(
  overrides: Partial<{
    accounts: Map<string, AccountState>;
    facts: Map<string, FactState>;
    validators: Map<string, any>;
    blockNumber: bigint;
    activeRounds: any[];
  }> = {},
): HandlerContext {
  const accounts = overrides.accounts ?? new Map();
  const facts = overrides.facts ?? new Map();
  const validators = overrides.validators ?? new Map();
  const blockNumber = overrides.blockNumber ?? 1000n;
  const activeRounds = overrides.activeRounds ?? [];

  return {
    state: {
      getAccount: vi.fn((did: string) => accounts.get(did) ?? null),
      getBlockNumber: vi.fn(() => blockNumber),
      getValidator: vi.fn((key: string) => validators.get(key) ?? null),
      getFact: vi.fn((factId: string) => facts.get(factId) ?? null),
      getFactsByDomain: vi.fn((domain: string) => {
        const allFacts = Array.from(facts.values());
        if (!domain) return allFacts;
        return allFacts.filter((f) => f.domain === domain);
      }),
      searchFacts: vi.fn((query: string) => {
        return Array.from(facts.values()).filter((f) =>
          f.claim.toLowerCase().includes(query.toLowerCase()),
        );
      }),
      getFactReferences: vi.fn((_factId: string) => ({
        supports: [],
        contradicts: [],
        extends: [],
      })),
      getFactResearchSummary: vi.fn((_factId: string) => ({
        factId: _factId,
        replicationAttempts: 0,
        successfulReplications: 0,
        failedReplications: 0,
        fraudInvestigations: 0,
        fraudDetected: false,
        netConfidenceModifier: 0,
      })),
    } as any,
    consensus: {
      getActiveRounds: vi.fn(() => activeRounds),
      submitClaim: vi.fn(async () => "claim-id-generated"),
      submitCommitment: vi.fn(async () => {}),
      submitCommitmentDeterministic: vi.fn(async () => {}),
      submitReveal: vi.fn(async () => {}),
    } as any,
    stateIntegration: {} as any,
    nodeVersion: "1.0.0-test",
    networkId: "legible-testnet",
  };
}

// ============ Account Handler Tests ============

describe.sequential("Account Handlers", () => {
  describe("accountGetBalance", () => {
    it("should return balance for existing account", async () => {
      const account = createMockAccount({
        balance: 5000n * 10n ** 18n,
        stakedBalance: 500n * 10n ** 18n,
        nonce: 10n,
      });
      const accounts = new Map([[account.did, account]]);
      const ctx = createMockContext({ accounts });

      const result = await accountGetBalance({ did: account.did }, ctx);

      expect(result.balance).toBe("5000000000000000000000");
      expect(result.stakedBalance).toBe("500000000000000000000");
      expect(result.nonce).toBe("10");
    });

    it("should return zeros for non-existent account", async () => {
      const ctx = createMockContext();

      const result = await accountGetBalance({ did: "did:lgm:unknown" }, ctx);

      expect(result.balance).toBe("0");
      expect(result.stakedBalance).toBe("0");
      expect(result.nonce).toBe("0");
    });

    it("should handle zero balance account", async () => {
      const account = createMockAccount({
        balance: 0n,
        stakedBalance: 0n,
        nonce: 0n,
      });
      const accounts = new Map([[account.did, account]]);
      const ctx = createMockContext({ accounts });

      const result = await accountGetBalance({ did: account.did }, ctx);

      expect(result.balance).toBe("0");
      expect(result.stakedBalance).toBe("0");
      expect(result.nonce).toBe("0");
    });

    it("should handle very large balances", async () => {
      const largeBalance = 10n ** 30n;
      const account = createMockAccount({ balance: largeBalance });
      const accounts = new Map([[account.did, account]]);
      const ctx = createMockContext({ accounts });

      const result = await accountGetBalance({ did: account.did }, ctx);

      expect(result.balance).toBe(largeBalance.toString());
    });
  });

  describe("accountGetInfo", () => {
    it("should return full account info with block number", async () => {
      const account = createMockAccount();
      const accounts = new Map([[account.did, account]]);
      const ctx = createMockContext({ accounts, blockNumber: 500n });

      const result = await accountGetInfo({ did: account.did }, ctx);

      expect(result.account).toEqual(account);
      expect(result.blockNumber).toBe("500");
    });

    it("should return null account for non-existent DID", async () => {
      const ctx = createMockContext({ blockNumber: 100n });

      const result = await accountGetInfo({ did: "did:lgm:nonexistent" }, ctx);

      expect(result.account).toBeNull();
      expect(result.blockNumber).toBe("100");
    });

    it("should not include proof when not requested", async () => {
      const account = createMockAccount();
      const accounts = new Map([[account.did, account]]);
      const ctx = createMockContext({ accounts });

      const result = await accountGetInfo(
        { did: account.did, includeProof: false },
        ctx,
      );

      expect(result.proof).toBeUndefined();
    });
  });
});

// ============ Agent Handler Tests ============

describe.sequential("Agent Handlers", () => {
  describe("agentCheckAccount", () => {
    it("should return exists=true for existing account", async () => {
      const account = createMockAccount({
        balance: 1000n * 10n ** 18n,
        reputationScore: 0.9,
      });
      const accounts = new Map([[account.did, account]]);
      const ctx = createMockContext({ accounts });

      const result = await agentCheckAccount({ did: account.did }, ctx);

      expect(result.exists).toBe(true);
      expect(result.balance).toBe("1000000000000000000000");
      expect(result.reputation).toBe(0.9);
      expect(result.isValidator).toBe(false);
    });

    it("should return exists=false for non-existent account", async () => {
      const ctx = createMockContext();

      const result = await agentCheckAccount({ did: "did:lgm:unknown" }, ctx);

      expect(result.exists).toBe(false);
      expect(result.balance).toBeUndefined();
      expect(result.reputation).toBeUndefined();
    });

    it("should detect validator status", async () => {
      const account = createMockAccount();
      const accounts = new Map([[account.did, account]]);
      const validators = new Map([
        [account.operationalKeyHash, { isActive: true }],
      ]);
      const ctx = createMockContext({ accounts, validators });

      const result = await agentCheckAccount({ did: account.did }, ctx);

      expect(result.isValidator).toBe(true);
    });

    it("should return isValidator=false when validator not found", async () => {
      const account = createMockAccount();
      const accounts = new Map([[account.did, account]]);
      const ctx = createMockContext({ accounts });

      const result = await agentCheckAccount({ did: account.did }, ctx);

      expect(result.isValidator).toBe(false);
    });
  });

  describe("agentGetStatus", () => {
    it("should return full status for existing account", async () => {
      const account = createMockAccount({
        balance: 5000n * 10n ** 18n,
        stakedBalance: 1000n * 10n ** 18n,
        reputationScore: 0.75,
        sessionKeyCount: 3,
      });
      const accounts = new Map([[account.did, account]]);
      const ctx = createMockContext({ accounts });

      const result = await agentGetStatus({ did: account.did }, ctx);

      expect(result.account.balance).toBe("5000000000000000000000");
      expect(result.account.stakedBalance).toBe("1000000000000000000000");
      expect(result.account.reputation).toBe(0.75);
      expect(result.sessions.active).toBe(3);
      // Activity metrics are TODOs
      expect(result.activity.claimsSubmitted).toBe(0);
    });

    it("should throw ACCOUNT_NOT_FOUND for non-existent account", async () => {
      const ctx = createMockContext();

      await expect(
        agentGetStatus({ did: "did:lgm:unknown" }, ctx),
      ).rejects.toMatchObject({
        code: RPC_ERROR_CODES.ACCOUNT_NOT_FOUND,
        message: "Account not found",
      });
    });
  });
});

// ============ Knowledge Handler Tests ============

describe.sequential("Knowledge Handlers", () => {
  describe("knowledgeGetFacts", () => {
    it("should return facts filtered by domain", async () => {
      const fact1 = createMockFact({ id: "fact-1", domain: "astronomy" });
      const fact2 = createMockFact({ id: "fact-2", domain: "biology" });
      const facts = new Map([
        ["fact-1", fact1],
        ["fact-2", fact2],
      ]);
      const ctx = createMockContext({ facts });

      const result = await knowledgeGetFacts({ domain: "astronomy" }, ctx);

      expect(result.facts).toHaveLength(1);
      expect(result.facts[0].domain).toBe("astronomy");
      expect(result.total).toBe(1);
    });

    it("should return facts filtered by search", async () => {
      const fact1 = createMockFact({ id: "fact-1", claim: "Earth orbits Sun" });
      const fact2 = createMockFact({
        id: "fact-2",
        claim: "Water boils at 100C",
      });
      const facts = new Map([
        ["fact-1", fact1],
        ["fact-2", fact2],
      ]);
      const ctx = createMockContext({ facts });

      const result = await knowledgeGetFacts({ search: "earth" }, ctx);

      expect(result.facts).toHaveLength(1);
      expect(result.facts[0].claim).toContain("Earth");
    });

    it("should filter by minimum confidence", async () => {
      const fact1 = createMockFact({ id: "fact-1", confidence: 0.9 });
      const fact2 = createMockFact({ id: "fact-2", confidence: 0.5 });
      const facts = new Map([
        ["fact-1", fact1],
        ["fact-2", fact2],
      ]);
      const ctx = createMockContext({ facts });

      const result = await knowledgeGetFacts({ minConfidence: 0.8 }, ctx);

      expect(result.facts).toHaveLength(1);
      expect(result.facts[0].confidence).toBeGreaterThanOrEqual(0.8);
    });

    it("should filter by status", async () => {
      const fact1 = createMockFact({ id: "fact-1", status: "active" });
      const fact2 = createMockFact({ id: "fact-2", status: "challenged" });
      const facts = new Map([
        ["fact-1", fact1],
        ["fact-2", fact2],
      ]);
      const ctx = createMockContext({ facts });

      const result = await knowledgeGetFacts({ status: "active" }, ctx);

      expect(result.facts).toHaveLength(1);
      expect(result.facts[0].status).toBe("active");
    });

    it("should paginate results with limit", async () => {
      const facts = new Map<string, FactState>();
      for (let i = 0; i < 100; i++) {
        const fact = createMockFact({ id: `fact-${i}` });
        facts.set(fact.id, fact);
      }
      const ctx = createMockContext({ facts });

      const result = await knowledgeGetFacts({ limit: 10 }, ctx);

      expect(result.facts).toHaveLength(10);
      expect(result.total).toBe(100);
      expect(result.nextCursor).toBe("10");
    });

    it("should handle cursor for pagination", async () => {
      const facts = new Map<string, FactState>();
      for (let i = 0; i < 30; i++) {
        const fact = createMockFact({ id: `fact-${i}` });
        facts.set(fact.id, fact);
      }
      const ctx = createMockContext({ facts });

      const result = await knowledgeGetFacts({ limit: 10, cursor: "10" }, ctx);

      expect(result.facts).toHaveLength(10);
      expect(result.nextCursor).toBe("20");
    });

    it("should return empty array when no facts match", async () => {
      const ctx = createMockContext();

      const result = await knowledgeGetFacts({ domain: "unknown" }, ctx);

      expect(result.facts).toHaveLength(0);
      expect(result.total).toBe(0);
      expect(result.nextCursor).toBeUndefined();
    });
  });

  describe("knowledgeGetFact", () => {
    it("should return fact by ID", async () => {
      const fact = createMockFact({ id: "fact-abc123" });
      const facts = new Map([["fact-abc123", fact]]);
      const ctx = createMockContext({ facts, blockNumber: 200n });

      const result = await knowledgeGetFact({ factId: "fact-abc123" }, ctx);

      expect(result.fact).toEqual(fact);
      expect(result.blockNumber).toBe("200");
    });

    it("should return null fact when not found", async () => {
      const ctx = createMockContext({ blockNumber: 100n });

      const result = await knowledgeGetFact({ factId: "unknown-fact" }, ctx);

      expect(result.fact).toBeNull();
      expect(result.blockNumber).toBe("100");
    });

    it("should include references when requested", async () => {
      const fact = createMockFact();
      const facts = new Map([[fact.id, fact]]);
      const ctx = createMockContext({ facts });

      const result = await knowledgeGetFact(
        { factId: fact.id, includeReferences: true },
        ctx,
      );

      expect(result.references).toEqual({
        supports: [],
        contradicts: [],
        extends: [],
      });
      // Verify getFactReferences was called with the correct fact ID
      expect(ctx.state.getFactReferences).toHaveBeenCalledWith(fact.id);
    });

    it("should not include references when not requested", async () => {
      const fact = createMockFact();
      const facts = new Map([[fact.id, fact]]);
      const ctx = createMockContext({ facts });

      const result = await knowledgeGetFact(
        { factId: fact.id, includeReferences: false },
        ctx,
      );

      expect(result.references).toBeUndefined();
    });
  });

  describe("knowledgeSubmitClaim", () => {
    it("should reject invalid signature", async () => {
      const account = createMockAccount({ balance: 1000n * 10n ** 18n });
      const accounts = new Map([[account.did, account]]);
      const ctx = createMockContext({ accounts });

      // Mock verify to return false
      vi.mock("../../../../core/crypto/signatures.js", () => ({
        verify: vi.fn(() => false),
      }));

      await expect(
        knowledgeSubmitClaim(
          {
            claim: "Test claim",
            domain: "test",
            stake: "100000000000000000000",
            sender:
              "abc123def456789012345678901234567890123456789012345678901234",
            signature: "invalid-signature",
            nonce: "0",
          },
          ctx,
        ),
      ).rejects.toMatchObject({
        code: RPC_ERROR_CODES.INVALID_SIGNATURE,
      });
    });
  });
});

// ============ Verification Handler Tests ============

describe.sequential("Verification Handlers", () => {
  describe("verificationGetPending", () => {
    it("should return assignments for validator", async () => {
      const validatorKey = "validator-key-12345678901234567890123456789012";
      const activeRounds = [
        {
          id: "round-1",
          claimId: "claim-1",
          claim: { claim: "Test claim", domain: "test" },
          commitDeadline: 1000,
          revealDeadline: 2000,
          selectedValidators: [
            { validator: validatorKey, selected: true },
            { validator: "other-validator", selected: true },
          ],
        },
      ];
      const ctx = createMockContext({ activeRounds });

      const result = await verificationGetPending({ validatorKey }, ctx);

      expect(result.assignments).toHaveLength(1);
      expect(result.assignments[0].roundId).toBe("round-1");
      expect(result.assignments[0].claim).toBe("Test claim");
    });

    it("should return empty when validator not assigned", async () => {
      const validatorKey = "unassigned-validator-key-00000000000000";
      const activeRounds = [
        {
          id: "round-1",
          claimId: "claim-1",
          claim: { claim: "Test", domain: "test" },
          commitDeadline: 1000,
          revealDeadline: 2000,
          selectedValidators: [
            { validator: "other-validator", selected: true },
          ],
        },
      ];
      const ctx = createMockContext({ activeRounds });

      const result = await verificationGetPending({ validatorKey }, ctx);

      expect(result.assignments).toHaveLength(0);
    });

    it("should handle multiple active rounds", async () => {
      const validatorKey = "validator-key-12345678901234567890123456789012";
      const activeRounds = [
        {
          id: "round-1",
          claimId: "claim-1",
          claim: { claim: "Claim 1", domain: "test" },
          commitDeadline: 1000,
          revealDeadline: 2000,
          selectedValidators: [{ validator: validatorKey, selected: true }],
        },
        {
          id: "round-2",
          claimId: "claim-2",
          claim: { claim: "Claim 2", domain: "test" },
          commitDeadline: 1500,
          revealDeadline: 2500,
          selectedValidators: [{ validator: validatorKey, selected: true }],
        },
      ];
      const ctx = createMockContext({ activeRounds });

      const result = await verificationGetPending({ validatorKey }, ctx);

      expect(result.assignments).toHaveLength(2);
    });
  });

  describe("verificationSubmitCommitment", () => {
    it("should accept valid commitment", async () => {
      const activeRounds = [
        {
          id: "round-1",
          commitDeadline: 1000,
        },
      ];
      const ctx = createMockContext({ activeRounds });

      const result = await verificationSubmitCommitment(
        {
          roundId: "round-1",
          commitment: "commitment-hash-123",
          validatorKey: "validator-key",
          signature: "signature",
        },
        ctx,
      );

      expect(result.accepted).toBe(true);
      expect(result.commitDeadline).toBe(1000);
      // Without vrfOutput/vrfProof, handler uses deterministic mode
      expect(ctx.consensus.submitCommitmentDeterministic).toHaveBeenCalledWith(
        "round-1",
        {
          validator: "validator-key",
          commitment: "commitment-hash-123",
          signature: "signature",
        },
      );
    });

    it("should propagate consensus errors", async () => {
      const ctx = createMockContext();
      (ctx.consensus.submitCommitmentDeterministic as any).mockRejectedValue(
        new Error("Round not in commit phase"),
      );

      await expect(
        verificationSubmitCommitment(
          {
            roundId: "round-1",
            commitment: "commitment",
            validatorKey: "validator",
            signature: "sig",
          },
          ctx,
        ),
      ).rejects.toMatchObject({
        code: RPC_ERROR_CODES.INVALID_PARAMS,
        message: "Round not in commit phase",
      });
    });
  });

  describe("verificationSubmitReveal", () => {
    it("should accept valid reveal", async () => {
      const activeRounds = [
        {
          id: "round-1",
          phase: "reveal",
        },
      ];
      const ctx = createMockContext({ activeRounds });

      const result = await verificationSubmitReveal(
        {
          roundId: "round-1",
          verdict: "confirmed",
          confidence: 0.9,
          nonce: "nonce-123",
          validatorKey: "validator-key",
          signature: "signature",
        },
        ctx,
      );

      expect(result.accepted).toBe(true);
      expect(result.roundStatus).toBe("reveal");
    });

    it("should propagate consensus errors", async () => {
      const ctx = createMockContext();
      (ctx.consensus.submitReveal as any).mockRejectedValue(
        new Error("Invalid commitment"),
      );

      await expect(
        verificationSubmitReveal(
          {
            roundId: "round-1",
            verdict: "confirmed",
            confidence: 0.9,
            nonce: "wrong-nonce",
            validatorKey: "validator",
            signature: "sig",
          },
          ctx,
        ),
      ).rejects.toMatchObject({
        code: RPC_ERROR_CODES.INVALID_PARAMS,
      });
    });
  });

  describe("verificationGetRound", () => {
    it("should return round by ID", async () => {
      const round = {
        id: "round-123",
        phase: "reveal",
        result: { outcome: "accepted", confidence: 0.85 },
      };
      const activeRounds = [round];
      const ctx = createMockContext({ activeRounds });

      const result = await verificationGetRound({ roundId: "round-123" }, ctx);

      expect(result.round).toEqual(round);
      expect(result.result).toEqual({ outcome: "accepted", confidence: 0.85 });
    });

    it("should return null for unknown round", async () => {
      const ctx = createMockContext({ activeRounds: [] });

      const result = await verificationGetRound({ roundId: "unknown" }, ctx);

      expect(result.round).toBeNull();
    });
  });
});

// ============ Network Handler Tests ============

describe.sequential("Network Handlers", () => {
  describe("networkHealth", () => {
    it("should return health status", async () => {
      const ctx = createMockContext({ blockNumber: 12345n });

      const result = await networkHealth({}, ctx);

      expect(result.status).toBe("healthy");
      expect(result.latestBlock).toBe("12345");
      expect(result.syncStatus).toBe("synced");
      expect(result.peerCount).toBe(0); // TODO in implementation
      expect(result.version).toBe("1.0.0-test");
    });

    it("should return correct version from context", async () => {
      const ctx = createMockContext();
      ctx.nodeVersion = "2.0.0-beta";

      const result = await networkHealth({}, ctx);

      expect(result.version).toBe("2.0.0-beta");
    });
  });
});

// ============ Handler Registry Tests ============

describe.sequential("Handler Registry", () => {
  it("should have all expected handlers registered", () => {
    expect(handlers["account_getBalance"]).toBe(accountGetBalance);
    expect(handlers["account_getInfo"]).toBe(accountGetInfo);
    expect(handlers["agent_checkAccount"]).toBe(agentCheckAccount);
    expect(handlers["agent_getStatus"]).toBe(agentGetStatus);
    expect(handlers["knowledge_getFacts"]).toBe(knowledgeGetFacts);
    expect(handlers["knowledge_getFact"]).toBe(knowledgeGetFact);
    expect(handlers["knowledge_submitClaim"]).toBe(knowledgeSubmitClaim);
    expect(handlers["verification_getPending"]).toBe(verificationGetPending);
    expect(handlers["verification_submitCommitment"]).toBe(
      verificationSubmitCommitment,
    );
    expect(handlers["verification_submitReveal"]).toBe(
      verificationSubmitReveal,
    );
    expect(handlers["verification_getRound"]).toBe(verificationGetRound);
    expect(handlers["network_health"]).toBe(networkHealth);
  });

  it("should have all registered handlers", () => {
    expect(Object.keys(handlers)).toHaveLength(71);
  });
});

// ============ Error Handling Tests ============

describe.sequential("Error Handling", () => {
  it("should create proper RPC error for ACCOUNT_NOT_FOUND", async () => {
    const ctx = createMockContext();

    try {
      await agentGetStatus({ did: "unknown" }, ctx);
      expect.fail("Should have thrown");
    } catch (error: any) {
      expect(error.code).toBe(RPC_ERROR_CODES.ACCOUNT_NOT_FOUND);
      expect(error.data.errorCode).toBe("ACCOUNT_NOT_FOUND");
      expect(error.data.retryable).toBe(false);
    }
  });
});
