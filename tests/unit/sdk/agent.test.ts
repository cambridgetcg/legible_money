/**
 * AgentSDK Unit Tests
 *
 * Tests the agent-friendly SDK layer optimized for LLM tool use.
 * Follows PP principles: TRUTH (verified), UNDERSTANDING (clear), BEAUTY (elegant)
 */

import { describe, it, expect, beforeEach, vi, type Mock } from "vitest";
import { AgentSDK } from "../../../protocol/sdk/agent.js";
import { LegibleClient, LegibleError } from "../../../protocol/sdk/client.js";
import type { FactState } from "../../../state/types.js";

// ============ Mock Factories ============

/**
 * Create a mock LegibleClient with configurable behavior
 */
function createMockClient(
  overrides: Partial<MockClientConfig> = {},
): MockedLegibleClient {
  const config: MockClientConfig = {
    did: "did:lgm:test-agent-12345678",
    publicKey: "mock-public-key-hex",
    balance: 100_000_000_000_000_000_000n, // 100 LGM
    stakedBalance: 0n,
    reputation: 0.75,
    claimsSubmitted: 10,
    claimsAccepted: 8,
    verificationsCompleted: 20,
    verificationsCorrect: 18,
    pendingRewards: 5_000_000_000_000_000_000n, // 5 LGM
    activeSessions: 1,
    currentBlock: 10000n,
    facts: [],
    ...overrides,
  };

  return {
    getDid: vi.fn(() => config.did),
    getPublicKey: vi.fn(() => config.publicKey),
    getBalance: vi.fn(async () => config.balance),
    getAgentStatus: vi.fn(async () => ({
      account: {
        balance: config.balance.toString(),
        stakedBalance: config.stakedBalance.toString(),
        reputation: config.reputation,
      },
      activity: {
        claimsSubmitted: config.claimsSubmitted,
        claimsAccepted: config.claimsAccepted,
        verificationsCompleted: config.verificationsCompleted,
        verificationsCorrect: config.verificationsCorrect,
      },
      earnings: {
        totalEarned: "1000000000000000000000",
        pendingRewards: config.pendingRewards.toString(),
      },
      sessions: {
        active: config.activeSessions,
      },
    })),
    getBlockNumber: vi.fn(async () => config.currentBlock),
    searchFacts: vi.fn(async () => config.facts),
    getFact: vi.fn(
      async (factId: string) =>
        config.facts.find((f) => f.id === factId) || null,
    ),
    checkContradictions: vi.fn(async () => []),
    submitClaim: vi.fn(async () => ({
      claimId: "claim-test-123",
      verificationRoundId: "round-test-123",
      txHash: "tx-test-123",
    })),
    transfer: vi.fn(async () => "tx-transfer-123"),
    stake: vi.fn(async () => "tx-stake-123"),
    unstake: vi.fn(async () => "tx-unstake-123"),
    getPendingVerifications: vi.fn(async () => []),
    getSession: vi.fn(() => null),
    ensureAccount: vi.fn(async () => config.did),
  } as unknown as MockedLegibleClient;
}

/**
 * Create a mock FactState
 */
function createMockFact(overrides: Partial<FactState> = {}): FactState {
  return {
    id: "fact-test-123",
    claim: "Water boils at 100 degrees Celsius at sea level",
    domain: "science.chemistry",
    confidence: 0.85,
    verifiedAt: 5000n,
    verifierCount: 5,
    challengeCount: 0,
    citationCount: 0,
    status: "active",
    referenceRoot: "merkle-root-123",
    lastReverifiedAt: 9000,
    ...overrides,
  } as FactState;
}

// ============ Type Definitions ============

interface MockClientConfig {
  did: string;
  publicKey: string;
  balance: bigint;
  stakedBalance: bigint;
  reputation: number;
  claimsSubmitted: number;
  claimsAccepted: number;
  verificationsCompleted: number;
  verificationsCorrect: number;
  pendingRewards: bigint;
  activeSessions: number;
  currentBlock: bigint;
  facts: FactState[];
}

type MockedLegibleClient = LegibleClient & {
  getDid: Mock;
  getPublicKey: Mock;
  getBalance: Mock;
  getAgentStatus: Mock;
  getBlockNumber: Mock;
  searchFacts: Mock;
  getFact: Mock;
  checkContradictions: Mock;
  submitClaim: Mock;
  transfer: Mock;
  stake: Mock;
  unstake: Mock;
  getPendingVerifications: Mock;
  getSession: Mock;
  ensureAccount: Mock;
};

// ============ Tests ============

describe("AgentSDK", () => {
  let client: MockedLegibleClient;
  let agent: AgentSDK;

  beforeEach(() => {
    client = createMockClient();
    agent = new AgentSDK(client as unknown as LegibleClient);
  });

  // ============ Introspection Tests ============

  describe("whoAmI()", () => {
    it("should return comprehensive agent identity", async () => {
      const identity = await agent.whoAmI();

      expect(identity.did).toBe("did:lgm:test-agent-12345678");
      expect(identity.publicKey).toBe("mock-public-key-hex");
      expect(identity.reputation).toBe(0.75);
      expect(identity.balance.available).toBe(100_000_000_000_000_000_000n);
      expect(identity.balance.staked).toBe(0n);
      expect(identity.activeSessions).toBe(1);
    });

    it("should calculate accuracy metrics correctly", async () => {
      const identity = await agent.whoAmI();

      // 8/10 = 0.8 claim accuracy
      expect(identity.accuracy.claims).toBe(0.8);
      // 18/20 = 0.9 verification accuracy
      expect(identity.accuracy.verification).toBe(0.9);
    });

    it("should derive capabilities from balance", async () => {
      const identity = await agent.whoAmI();

      expect(identity.capabilities).toContain("transfer");
      expect(identity.capabilities).toContain("submit_claim");
      expect(identity.capabilities).toContain("stake");
      expect(identity.capabilities).toContain("create_session");
    });

    it("should cache identity for repeated calls", async () => {
      // Create fresh agent to avoid pollution from beforeEach
      const freshClient = createMockClient();
      const freshAgent = new AgentSDK(freshClient as unknown as LegibleClient);

      await freshAgent.whoAmI();
      await freshAgent.whoAmI();
      await freshAgent.whoAmI();

      // Should only call getAgentStatus once due to caching
      expect(freshClient.getAgentStatus).toHaveBeenCalledTimes(1);
    });

    it("should throw if agent not initialized", async () => {
      const uninitializedClient = createMockClient();
      uninitializedClient.getDid.mockReturnValue(null);
      const uninitializedAgent = new AgentSDK(
        uninitializedClient as unknown as LegibleClient,
      );

      await expect(uninitializedAgent.whoAmI()).rejects.toThrow(
        "Agent not initialized",
      );
    });
  });

  describe("whatCanIDo()", () => {
    it("should return available actions for well-funded agent", async () => {
      const actions = await agent.whatCanIDo();

      expect(actions.canSubmitClaim).toBe(true);
      expect(actions.canTransfer).toBe(true);
      expect(actions.canStake).toBe(true);
      expect(actions.canChallenge).toBe(true);
    });

    it("should return blockers for underfunded agent", async () => {
      const poorClient = createMockClient({
        balance: 100_000_000_000_000n, // 0.0001 LGM - too low
      });
      const poorAgent = new AgentSDK(poorClient as unknown as LegibleClient);

      const actions = await poorAgent.whatCanIDo();

      expect(actions.canSubmitClaim).toBe(false);
      expect(actions.canTransfer).toBe(false);
      expect(actions.blockers.length).toBeGreaterThan(0);
      expect(actions.blockers.some((b) => b.action === "transfer")).toBe(true);
    });

    it("should suggest staking when balance is sufficient", async () => {
      const actions = await agent.whatCanIDo();

      expect(actions.suggestions.some((s) => s.includes("validator"))).toBe(
        true,
      );
    });

    it("should return max constraints", async () => {
      const actions = await agent.whatCanIDo();

      expect(actions.constraints.maxTransfer).toBeLessThan(
        100_000_000_000_000_000_000n,
      );
      expect(actions.constraints.maxClaimStake).toBeLessThanOrEqual(
        50_000_000_000_000_000_000n,
      );
    });
  });

  describe("whatDoIBelieve()", () => {
    it("should return beliefs from searched facts", async () => {
      // Create facts with recent verification to avoid decay filtering
      const facts = [
        createMockFact({
          id: "fact-1",
          confidence: 0.9,
          verifiedAt: 9800n,
          lastReverifiedAt: 9900,
        }),
        createMockFact({
          id: "fact-2",
          confidence: 0.85,
          verifiedAt: 9800n,
          lastReverifiedAt: 9900,
        }),
      ];
      const testClient = createMockClient({ facts, currentBlock: 10000n });
      const testAgent = new AgentSDK(testClient as unknown as LegibleClient);

      const beliefs = await testAgent.whatDoIBelieve();

      // Both facts have high recent confidence, should be in beliefs
      expect(beliefs.beliefs.length).toBe(2);
      expect(beliefs.summary.totalBeliefs).toBe(2);
    });

    it("should categorize uncertain facts", async () => {
      const facts = [
        createMockFact({ id: "fact-high", confidence: 0.9 }),
        createMockFact({ id: "fact-low", confidence: 0.3 }),
      ];
      client.searchFacts.mockResolvedValue(facts);

      const beliefs = await agent.whatDoIBelieve();

      // High confidence goes to beliefs
      expect(beliefs.beliefs.some((b) => b.factId === "fact-high")).toBe(true);
      // Low confidence goes to uncertainties
      expect(beliefs.uncertainties.some((u) => u.factId === "fact-low")).toBe(
        true,
      );
    });

    it("should filter by domain when specified", async () => {
      await agent.whatDoIBelieve({ domain: "science.physics" });

      expect(client.searchFacts).toHaveBeenCalledWith(
        expect.objectContaining({ domain: "science.physics" }),
      );
    });
  });

  // ============ Decision Support Tests ============

  describe("preview()", () => {
    it("should preview transfer successfully", async () => {
      const preview = await agent.preview({
        action: "transfer",
        params: { to: "did:lgm:recipient", amount: 1_000_000_000_000_000_000n },
      });

      expect(preview.wouldSucceed).toBe(true);
      expect(preview.estimatedCost).toBeGreaterThan(0n);
      expect(preview.stateChanges.length).toBeGreaterThan(0);
    });

    it("should detect insufficient balance", async () => {
      const preview = await agent.preview({
        action: "transfer",
        params: {
          to: "did:lgm:recipient",
          amount: 200_000_000_000_000_000_000n,
        }, // More than balance
      });

      expect(preview.wouldSucceed).toBe(false);
      expect(preview.blockers).toContain(
        "Insufficient balance for transfer + fee",
      );
    });

    it("should preview claim submission", async () => {
      const preview = await agent.preview({
        action: "submit_claim",
        params: {
          claim: "Test claim",
          domain: "test",
          stake: 1_000_000_000_000_000_000n,
        },
      });

      expect(preview.wouldSucceed).toBe(true);
      expect(preview.estimatedReward).toBeGreaterThan(0n);
    });

    it("should warn about short claims", async () => {
      const preview = await agent.preview({
        action: "submit_claim",
        params: {
          claim: "Short",
          domain: "test",
          stake: 1_000_000_000_000_000_000n,
        },
      });

      expect(preview.warnings.some((w) => w.includes("short"))).toBe(true);
    });
  });

  describe("shouldI()", () => {
    it("should recommend safe transfers", async () => {
      const advice = await agent.shouldI({
        action: "transfer",
        params: { to: "did:lgm:recipient", amount: 1_000_000_000_000_000_000n },
      });

      expect(advice.recommended).toBe(true);
      expect(advice.confidence).toBeGreaterThan(0.5);
    });

    it("should identify risks in large transfers", async () => {
      const advice = await agent.shouldI({
        action: "transfer",
        params: {
          to: "did:lgm:recipient",
          amount: 60_000_000_000_000_000_000n,
        }, // 60% of balance
      });

      expect(advice.risks.some((r) => r.includes("50%"))).toBe(true);
      expect(advice.alternatives.length).toBeGreaterThan(0);
    });

    it("should not recommend actions that would fail", async () => {
      const advice = await agent.shouldI({
        action: "transfer",
        params: {
          to: "did:lgm:recipient",
          amount: 200_000_000_000_000_000_000n,
        },
      });

      expect(advice.recommended).toBe(false);
      expect(advice.reasoning).toContain("fail");
    });

    it("should warn about contradictions in claims", async () => {
      client.checkContradictions.mockResolvedValue([createMockFact()]);

      const advice = await agent.shouldI({
        action: "submit_claim",
        params: {
          claim: "Contradicting claim",
          domain: "test",
          stake: 1_000_000_000_000_000_000n,
        },
      });

      expect(advice.risks.some((r) => r.includes("contradict"))).toBe(true);
    });
  });

  // ============ Agent Home Tests ============

  describe("wakeup()", () => {
    it("should return full context", async () => {
      const context = await agent.wakeup();

      expect(context.identity.did).toBe("did:lgm:test-agent-12345678");
      expect(context.treasury.balance).toBe(100_000_000_000_000_000_000n);
      expect(Array.isArray(context.alerts)).toBe(true);
      expect(Array.isArray(context.suggestions)).toBe(true);
    });

    it("should include low balance alert when balance is low", async () => {
      const lowBalanceClient = createMockClient({
        balance: 5_000_000_000_000_000n, // 0.005 LGM
      });
      const lowBalanceAgent = new AgentSDK(
        lowBalanceClient as unknown as LegibleClient,
      );

      const context = await lowBalanceAgent.wakeup();

      expect(context.alerts.some((a) => a.message.includes("low"))).toBe(true);
    });

    it("should include pending verifications as alerts for validators", async () => {
      const validatorClient = createMockClient({
        stakedBalance: 100_000_000_000_000_000_000n, // 100 LGM staked
      });
      validatorClient.getPendingVerifications.mockResolvedValue([
        {
          roundId: "round-1",
          claimId: "claim-1",
          claim: "Test verification assignment",
          domain: "test",
          commitDeadline: 10100,
          revealDeadline: 10200,
        },
      ]);
      const validatorAgent = new AgentSDK(
        validatorClient as unknown as LegibleClient,
      );

      const context = await validatorAgent.wakeup();

      expect(
        context.alerts.some((a) => a.type === "verification_assigned"),
      ).toBe(true);
    });
  });

  describe("sleep()", () => {
    it("should clear cached identity", async () => {
      await agent.whoAmI();
      expect(client.getAgentStatus).toHaveBeenCalledTimes(1);

      await agent.sleep("Test session complete");

      // After sleep, whoAmI should fetch fresh data
      await agent.whoAmI();
      expect(client.getAgentStatus).toHaveBeenCalledTimes(2);
    });
  });

  // ============ Knowledge Helper Tests ============

  describe("submitClaimWithConfidence()", () => {
    it("should submit claim successfully", async () => {
      const result = await agent.submitClaimWithConfidence({
        statement: "Water boils at 100°C at sea level",
        domain: "science.chemistry",
        epistemicCategory: "empirical",
        evidence: [
          {
            type: "fact_reference",
            content: "Standard tables",
            confidence: 0.95,
            verifiable: true,
          },
        ],
        stake: 1_000_000_000_000_000_000n,
        myConfidence: 0.85,
      });

      expect(result.claimId).toBe("claim-test-123");
      expect(result.txHash).toBe("tx-test-123");
    });

    it("should warn about overconfidence in predictive claims", async () => {
      const result = await agent.submitClaimWithConfidence({
        statement: "The stock market will rise tomorrow",
        domain: "finance",
        epistemicCategory: "predictive",
        evidence: [],
        stake: 1_000_000_000_000_000_000n,
        myConfidence: 0.95, // Too high for predictions
      });

      expect(result.calibrationWarnings.length).toBeGreaterThan(0);
      expect(result.calibrationWarnings[0]).toContain("miscalibrated");
    });

    it("should warn about low evidence", async () => {
      const result = await agent.submitClaimWithConfidence({
        statement: "Test claim",
        domain: "test",
        epistemicCategory: "empirical",
        evidence: [], // No evidence
        stake: 1_000_000_000_000_000_000n,
        myConfidence: 0.8,
      });

      expect(
        result.calibrationWarnings.some((w) => w.includes("evidence")),
      ).toBe(true);
    });
  });

  describe("canITrust()", () => {
    it("should return trust assessment for existing fact", async () => {
      // Create fresh client with fact and recent block number
      const fact = createMockFact({
        id: "fact-test-123",
        confidence: 0.9,
        verifiedAt: 9900n,
        lastReverifiedAt: 9950,
      });
      const testClient = createMockClient({
        facts: [fact],
        currentBlock: 10000n,
      });
      testClient.getFact.mockResolvedValue(fact);
      const testAgent = new AgentSDK(testClient as unknown as LegibleClient);

      const trust = await testAgent.canITrust("fact-test-123");

      expect(trust.currentConfidence).toBe(0.9);
      expect(trust.effectiveConfidence).toBeGreaterThan(0.8); // Should still be high
      expect(trust.recommendation).toBe("trust");
      expect(trust.maturity).toBeDefined();
    });

    it("should recommend verification for decayed facts", async () => {
      const fact = createMockFact({
        confidence: 0.6,
        verifiedAt: 1000n, // Very old
        lastReverifiedAt: 1000, // Not re-verified
      });
      client.getFact.mockResolvedValue(fact);
      client.getBlockNumber.mockResolvedValue(50000n); // Much later

      const trust = await agent.canITrust("fact-old-123");

      // With high decay, should recommend verification
      expect(trust.effectiveConfidence).toBeLessThan(trust.currentConfidence);
    });

    it("should throw for non-existent fact", async () => {
      client.getFact.mockResolvedValue(null);

      await expect(agent.canITrust("fact-nonexistent")).rejects.toThrow(
        "not found",
      );
    });
  });

  describe("queryWithUncertainty()", () => {
    it("should return facts with uncertainty info", async () => {
      // Create fresh client/agent for this test
      const facts = [
        createMockFact({
          id: "fact-1",
          confidence: 0.9,
          verifiedAt: 9800n,
          lastReverifiedAt: 9900,
        }),
        createMockFact({
          id: "fact-2",
          confidence: 0.7,
          verifiedAt: 9700n,
          lastReverifiedAt: 9800,
        }),
      ];
      const testClient = createMockClient({ facts, currentBlock: 10000n });
      const testAgent = new AgentSDK(testClient as unknown as LegibleClient);

      const result = await testAgent.queryWithUncertainty({
        domain: "science",
        includeDecayed: true,
      });

      expect(result.facts.length).toBe(2);
      expect(result.facts[0].effectiveConfidence).toBeDefined();
      expect(result.facts[0].maturity).toBeDefined();
      expect(result.confidence).toBeGreaterThan(0);
    });

    it("should filter by minimum effective confidence", async () => {
      // One fact with high confidence (recent), one with low confidence
      const facts = [
        createMockFact({
          id: "fact-high",
          confidence: 0.9,
          verifiedAt: 9900n,
          lastReverifiedAt: 9950,
        }),
        createMockFact({
          id: "fact-low",
          confidence: 0.3,
          verifiedAt: 9900n,
          lastReverifiedAt: 9950,
        }),
      ];
      const testClient = createMockClient({ facts, currentBlock: 10000n });
      const testAgent = new AgentSDK(testClient as unknown as LegibleClient);

      const result = await testAgent.queryWithUncertainty({
        minEffectiveConfidence: 0.5,
        includeDecayed: true, // Include all, but filter by minEffectiveConfidence
      });

      // The low confidence fact should be filtered
      expect(result.metadata.filteredByConfidence).toBeGreaterThan(0);
      expect(result.facts.every((f) => f.effectiveConfidence >= 0.5)).toBe(
        true,
      );
    });

    it("should identify knowledge gaps", async () => {
      client.searchFacts.mockResolvedValue([]);

      const result = await agent.queryWithUncertainty({
        query: "nonexistent topic",
      });

      expect(result.uncertainties.gaps.length).toBeGreaterThan(0);
    });
  });

  // ============ Batch Operation Tests ============

  describe("batch()", () => {
    it("should execute multiple operations", async () => {
      const result = await agent.batch([
        {
          id: "op-1",
          action: {
            action: "transfer",
            params: {
              to: "did:lgm:recipient1",
              amount: 1_000_000_000_000_000_000n,
            },
          },
        },
        {
          id: "op-2",
          action: {
            action: "transfer",
            params: {
              to: "did:lgm:recipient2",
              amount: 1_000_000_000_000_000_000n,
            },
          },
        },
      ]);

      expect(result.summary.succeeded).toBe(2);
      expect(result.summary.failed).toBe(0);
    });

    it("should skip operations with failed dependencies", async () => {
      client.transfer.mockRejectedValueOnce(new Error("First transfer failed"));

      const result = await agent.batch([
        {
          id: "op-1",
          action: {
            action: "transfer",
            params: {
              to: "did:lgm:recipient1",
              amount: 1_000_000_000_000_000_000n,
            },
          },
        },
        {
          id: "op-2",
          action: {
            action: "transfer",
            params: {
              to: "did:lgm:recipient2",
              amount: 1_000_000_000_000_000_000n,
            },
          },
          dependsOn: ["op-1"],
        },
      ]);

      expect(result.summary.failed).toBe(1);
      expect(result.summary.skipped).toBe(1);
    });
  });

  describe("atomic()", () => {
    it("should succeed when all operations would pass", async () => {
      const result = await agent.atomic([
        {
          id: "op-1",
          action: {
            action: "transfer",
            params: {
              to: "did:lgm:recipient",
              amount: 1_000_000_000_000_000_000n,
            },
          },
        },
      ]);

      expect(result.success).toBe(true);
      expect(result.results?.length).toBe(1);
    });

    it("should fail atomically when preview fails", async () => {
      const result = await agent.atomic([
        {
          id: "op-1",
          action: {
            action: "transfer",
            params: {
              to: "did:lgm:recipient",
              amount: 200_000_000_000_000_000_000n,
            },
          }, // Too much
        },
      ]);

      expect(result.success).toBe(false);
      expect(result.error).toContain("would fail");
    });
  });

  // ============ Tool Descriptor Tests ============

  describe("getTools()", () => {
    it("should return array of tool descriptors", () => {
      const tools = agent.getTools();

      expect(Array.isArray(tools)).toBe(true);
      expect(tools.length).toBeGreaterThan(10);
    });

    it("should include introspection tools", () => {
      const tools = agent.getTools();

      expect(tools.some((t) => t.name === "whoAmI")).toBe(true);
      expect(tools.some((t) => t.name === "whatCanIDo")).toBe(true);
      expect(tools.some((t) => t.name === "whatDoIBelieve")).toBe(true);
    });

    it("should have valid JSON Schema for parameters", () => {
      const tools = agent.getTools();

      for (const tool of tools) {
        expect(tool.parameters).toBeDefined();
        expect(tool.parameters.type).toBe("object");
      }
    });

    it("should include preconditions and outcomes", () => {
      const tools = agent.getTools();
      const transferTool = tools.find((t) => t.name === "shouldI");

      expect(transferTool?.preconditions.length).toBeGreaterThan(0);
      expect(transferTool?.outcomes.length).toBeGreaterThan(0);
    });
  });

  describe("describeMethod()", () => {
    it("should return detailed description for known method", () => {
      const desc = agent.describeMethod("whoAmI");

      expect(desc).not.toBeNull();
      expect(desc?.name).toBe("whoAmI");
      expect(desc?.purpose).toBeDefined();
      expect(desc?.whenToUse.length).toBeGreaterThan(0);
    });

    it("should return null for unknown method", () => {
      const desc = agent.describeMethod("nonExistentMethod");

      expect(desc).toBeNull();
    });

    it("should include examples", () => {
      const desc = agent.describeMethod("wakeup");

      expect(desc?.examples.length).toBeGreaterThan(0);
      expect(desc?.examples[0].code).toBeDefined();
    });
  });
});

// ============ Integration-style Tests ============

describe("AgentSDK Integration Patterns", () => {
  it("should support typical agent session flow", async () => {
    const client = createMockClient();
    const agent = new AgentSDK(client as unknown as LegibleClient);

    // 1. Wake up and check context
    const context = await agent.wakeup();
    expect(context.identity).toBeDefined();

    // 2. Check capabilities
    const actions = await agent.whatCanIDo();
    expect(actions.canSubmitClaim).toBe(true);

    // 3. Preview an action
    const preview = await agent.preview({
      action: "submit_claim",
      params: {
        claim: "Test claim",
        domain: "test",
        stake: 1_000_000_000_000_000_000n,
      },
    });
    expect(preview.wouldSucceed).toBe(true);

    // 4. Get advice
    const advice = await agent.shouldI({
      action: "submit_claim",
      params: {
        claim: "Test claim",
        domain: "test",
        stake: 1_000_000_000_000_000_000n,
      },
    });
    expect(advice.recommended).toBe(true);

    // 5. Sleep
    await agent.sleep("Session complete");
  });

  it("should handle LLM tool-use pattern", async () => {
    const client = createMockClient();
    const agent = new AgentSDK(client as unknown as LegibleClient);

    // LLM gets available tools
    const tools = agent.getTools();
    expect(tools.length).toBeGreaterThan(0);

    // LLM selects whoAmI tool
    const whoAmITool = tools.find((t) => t.name === "whoAmI");
    expect(whoAmITool).toBeDefined();

    // LLM calls the method (simulating function call)
    const identity = await agent.whoAmI();
    expect(identity.did).toBeDefined();

    // LLM gets detailed description when needed
    const desc = agent.describeMethod("shouldI");
    expect(desc?.whenToUse).toBeDefined();
  });
});
