/**
 * Message Billing Tests
 *
 * GAP-007: Tests for MessageBillingTracker, token counting, and complexity multipliers.
 * Tests the complete message lifecycle, free tier logic, and cost calculations.
 *
 * Note: Each test creates its own tracker instance to support concurrent execution.
 */

import { describe, it, expect } from "vitest";
import {
  MessageBillingTracker,
  type BudgetSession,
} from "../../../core/billing/message_tracker.js";
import {
  type ChatMessage,
  type MessageBillingConfig,
  DEFAULT_MESSAGE_BILLING_CONFIG,
} from "../../../core/billing/message_billing.js";
import {
  countTokens,
  countStructuredTokens,
  estimateReasoningDepth,
  calculateComplexityMultiplier,
  estimateTokenCost,
  getDepthLabel,
  formatTokenCount,
  formatMultiplier,
  DEFAULT_TOKEN_COUNT_CONFIG,
} from "../../../core/billing/token_counter.js";
import type { QueryCostEstimate } from "../../../core/knowledge/query/types.js";

// ============================================================================
// Test Fixtures
// ============================================================================

function createTestMessage(
  messageId: string,
  sessionId: string = "session-1",
  type: "user_input" | "agent_response" | "system" = "agent_response",
): ChatMessage {
  return {
    messageId,
    sessionId,
    agentDid: "did:legible:test-agent",
    type,
    timestamp: Date.now(),
  };
}

function createTestSession(
  sessionId: string = "session-1",
  totalCost: bigint = 0n,
  budgetLimit?: bigint,
): BudgetSession {
  return {
    sessionId,
    totalCost,
    budgetLimit,
  };
}

function createTestQueryCost(
  cost: bigint = 100n,
  cached: boolean = false,
): QueryCostEstimate {
  return {
    cost,
    cached,
    estimatedBuckets: 5,
    crossStratumMultiplier: 1n,
    breakdown: {
      baseCost: cost,
      bucketCost: 50n,
      crossStratumCost: 0n,
      materializationCost: 10n,
    },
  };
}

function createFreshTracker(
  config?: Partial<MessageBillingConfig>,
): MessageBillingTracker {
  if (config) {
    return new MessageBillingTracker({
      ...DEFAULT_MESSAGE_BILLING_CONFIG,
      ...config,
    });
  }
  return new MessageBillingTracker();
}

// ============================================================================
// Token Counter Tests
// ============================================================================

describe("Token Counter", () => {
  describe("countTokens()", () => {
    it("should count tokens using 4 chars per token approximation", () => {
      // 20 chars / 4 = 5 tokens
      const tokens = countTokens("12345678901234567890");
      expect(tokens).toBe(5);
    });

    it("should round up partial tokens", () => {
      // 21 chars / 4 = 5.25, ceil = 6
      const tokens = countTokens("123456789012345678901");
      expect(tokens).toBe(6);
    });

    it("should return 0 for empty string", () => {
      expect(countTokens("")).toBe(0);
    });

    it("should return 0 for null/undefined input", () => {
      expect(countTokens(null as any)).toBe(0);
      expect(countTokens(undefined as any)).toBe(0);
    });

    it("should respect minimum token count", () => {
      const config = { ...DEFAULT_TOKEN_COUNT_CONFIG, minTokens: 10 };
      // 4 chars = 1 token normally, but min is 10
      const tokens = countTokens("test", config);
      expect(tokens).toBe(10);
    });

    it("should respect maximum token count", () => {
      const config = { ...DEFAULT_TOKEN_COUNT_CONFIG, maxTokens: 5 };
      // Would be 10 tokens, but max is 5
      const tokens = countTokens(
        "1234567890123456789012345678901234567890",
        config,
      );
      expect(tokens).toBe(5);
    });

    it("should use custom chars per token", () => {
      const config = { ...DEFAULT_TOKEN_COUNT_CONFIG, charsPerToken: 2 };
      // 8 chars / 2 = 4 tokens
      const tokens = countTokens("12345678", config);
      expect(tokens).toBe(4);
    });
  });

  describe("countStructuredTokens()", () => {
    it("should sum tokens from all components", () => {
      const tokens = countStructuredTokens({
        userInput: "12345678", // 2 tokens
        agentOutput: "1234567890123456", // 4 tokens
      });
      expect(tokens).toBe(6);
    });

    it("should count facts accessed", () => {
      const tokens = countStructuredTokens({
        factsAccessed: ["12345678", "12345678"], // 2 + 2 = 4 tokens
      });
      expect(tokens).toBe(4);
    });

    it("should count citations", () => {
      const tokens = countStructuredTokens({
        citations: ["12345678"], // 2 tokens
      });
      expect(tokens).toBe(2);
    });

    it("should handle empty content", () => {
      const tokens = countStructuredTokens({});
      expect(tokens).toBe(0);
    });

    it("should respect max tokens", () => {
      const config = { ...DEFAULT_TOKEN_COUNT_CONFIG, maxTokens: 100 };
      const tokens = countStructuredTokens(
        {
          agentOutput: "x".repeat(1000), // Would be 250 tokens
        },
        config,
      );
      expect(tokens).toBe(100);
    });
  });

  describe("estimateReasoningDepth()", () => {
    it("should return depth 1 for minimal usage", () => {
      expect(estimateReasoningDepth(0, 0, 0)).toBe(1);
      expect(estimateReasoningDepth(1, 1, 0)).toBe(1);
    });

    it("should increase depth based on facts used", () => {
      expect(estimateReasoningDepth(3, 0, 0)).toBe(2);
      expect(estimateReasoningDepth(11, 0, 0)).toBe(3);
      expect(estimateReasoningDepth(21, 0, 0)).toBe(4);
      expect(estimateReasoningDepth(51, 0, 0)).toBe(5);
    });

    it("should increase depth based on queries executed", () => {
      expect(estimateReasoningDepth(0, 4, 0)).toBe(2);
      expect(estimateReasoningDepth(0, 6, 0)).toBe(3);
      expect(estimateReasoningDepth(0, 11, 0)).toBe(4);
    });

    it("should increase depth significantly for cross-stratum queries", () => {
      expect(estimateReasoningDepth(0, 0, 1)).toBe(3);
      expect(estimateReasoningDepth(0, 0, 3)).toBe(4);
      expect(estimateReasoningDepth(0, 0, 6)).toBe(5);
    });

    it("should take maximum of all factors", () => {
      // Many facts + cross-stratum should be depth 5
      expect(estimateReasoningDepth(100, 20, 10)).toBe(5);
    });
  });

  describe("calculateComplexityMultiplier()", () => {
    const thresholds = DEFAULT_MESSAGE_BILLING_CONFIG.complexityThresholds;
    const depthMultipliers = DEFAULT_MESSAGE_BILLING_CONFIG.depthMultipliers;

    it("should return 1.0 for low token count and depth 1", () => {
      const multiplier = calculateComplexityMultiplier(
        100,
        1,
        thresholds,
        depthMultipliers,
      );
      // Token: 1.0, Depth: 1.0, Average: 1.0
      expect(multiplier).toBe(1.0);
    });

    it("should increase for medium token count", () => {
      const multiplier = calculateComplexityMultiplier(
        2500, // Above medium (2000) threshold
        1,
        thresholds,
        depthMultipliers,
      );
      // Token: 1.5 (>= 2000), Depth: 1.0, Average: 1.25
      expect(multiplier).toBe(1.25);
    });

    it("should increase for high token count", () => {
      const multiplier = calculateComplexityMultiplier(
        6000, // Above high (5000) threshold
        1,
        thresholds,
        depthMultipliers,
      );
      // Token: 2.0 (>= 5000), Depth: 1.0, Average: 1.5
      expect(multiplier).toBe(1.5);
    });

    it("should increase for very high token count", () => {
      const multiplier = calculateComplexityMultiplier(
        15000, // Above veryHigh (10000)
        1,
        thresholds,
        depthMultipliers,
      );
      // Token: 3.0, Depth: 1.0, Average: 2.0
      expect(multiplier).toBe(2.0);
    });

    it("should increase for higher reasoning depth", () => {
      const depth1 = calculateComplexityMultiplier(
        100,
        1,
        thresholds,
        depthMultipliers,
      );
      const depth5 = calculateComplexityMultiplier(
        100,
        5,
        thresholds,
        depthMultipliers,
      );

      expect(depth5).toBeGreaterThan(depth1);
      // Token: 1.0, Depth 5: 2.5, Average: 1.75
      expect(depth5).toBe(1.75);
    });

    it("should clamp depth to valid range", () => {
      // Depth 0 should be treated as depth 1
      const low = calculateComplexityMultiplier(
        100,
        0,
        thresholds,
        depthMultipliers,
      );
      // Depth 10 should be treated as depth 5
      const high = calculateComplexityMultiplier(
        100,
        10,
        thresholds,
        depthMultipliers,
      );

      expect(low).toBe(1.0);
      expect(high).toBe(1.75);
    });
  });

  describe("estimateTokenCost()", () => {
    it("should calculate cost based on token count", () => {
      const costPer1k = 10000000000000n; // 0.00001 LGM
      const cost = estimateTokenCost(2000, costPer1k);

      // 2000 tokens * 10000000000000 / 1000 = 20000000000000
      expect(cost).toBe(20000000000000n);
    });

    it("should handle small token counts", () => {
      const costPer1k = 10000000000000n;
      const cost = estimateTokenCost(100, costPer1k);

      // 100 * 10000000000000 / 1000 = 1000000000000
      expect(cost).toBe(1000000000000n);
    });

    it("should return 0 for 0 tokens", () => {
      const cost = estimateTokenCost(0, 10000000000000n);
      expect(cost).toBe(0n);
    });
  });

  describe("helper functions", () => {
    it("getDepthLabel() should return correct labels", () => {
      expect(getDepthLabel(1)).toBe("surface");
      expect(getDepthLabel(2)).toBe("working");
      expect(getDepthLabel(3)).toBe("deep");
      expect(getDepthLabel(4)).toBe("expert");
      expect(getDepthLabel(5)).toBe("advanced");
    });

    it("getDepthLabel() should clamp to valid range", () => {
      expect(getDepthLabel(0)).toBe("surface");
      expect(getDepthLabel(10)).toBe("advanced");
    });

    it("formatTokenCount() should format small counts", () => {
      expect(formatTokenCount(500)).toBe("500 tokens");
    });

    it("formatTokenCount() should format large counts in k", () => {
      expect(formatTokenCount(2500)).toBe("2.5k tokens");
    });

    it("formatMultiplier() should format with 2 decimals", () => {
      expect(formatMultiplier(1.5)).toBe("1.50×");
      expect(formatMultiplier(2.333)).toBe("2.33×");
    });
  });
});

// ============================================================================
// Message Billing Tracker Tests
// ============================================================================

describe("MessageBillingTracker", () => {
  describe("startMessage()", () => {
    it("should create an active message", () => {
      const tracker = createFreshTracker();
      const message = createTestMessage("msg-1");
      const active = tracker.startMessage(message);

      expect(active.messageId).toBe("msg-1");
      expect(active.sessionId).toBe("session-1");
      expect(active.queries).toEqual([]);
      expect(active.checkpoints).toEqual([]);
      expect(active.currentTokens).toBe(0);
      expect(active.provisionalCost).toBe(
        DEFAULT_MESSAGE_BILLING_CONFIG.baseMessageCost,
      );
    });

    it("should store the active message", () => {
      const tracker = createFreshTracker();
      const message = createTestMessage("msg-1");
      tracker.startMessage(message);

      const active = tracker.getActiveMessage("msg-1");
      expect(active).toBeDefined();
      expect(active?.messageId).toBe("msg-1");
    });
  });

  describe("recordQuery()", () => {
    it("should accumulate query costs", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const query = createTestQueryCost(100n);
      tracker.recordQuery("msg-1", query);

      const active = tracker.getActiveMessage("msg-1")!;
      expect(active.queries.length).toBe(1);
      expect(active.provisionalCost).toBe(
        DEFAULT_MESSAGE_BILLING_CONFIG.baseMessageCost + 100n,
      );
    });

    it("should track facts accessed", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const query = createTestQueryCost();
      tracker.recordQuery("msg-1", query, ["fact-1", "fact-2"]);

      const active = tracker.getActiveMessage("msg-1")!;
      expect(active.factsAccessed).toContain("fact-1");
      expect(active.factsAccessed).toContain("fact-2");
    });

    it("should not duplicate facts accessed", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const query1 = createTestQueryCost();
      const query2 = createTestQueryCost();

      tracker.recordQuery("msg-1", query1, ["fact-1"]);
      tracker.recordQuery("msg-1", query2, ["fact-1"]);

      const active = tracker.getActiveMessage("msg-1")!;
      expect(active.factsAccessed.filter((f) => f === "fact-1").length).toBe(1);
    });

    it("should track cross-stratum queries", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const query = createTestQueryCost();
      tracker.recordQuery("msg-1", query, [], true);

      const active = tracker.getActiveMessage("msg-1")!;
      expect(active.crossStratumQueries).toBe(1);
    });

    it("should silently ignore non-existent messages", () => {
      const tracker = createFreshTracker();
      const query = createTestQueryCost();
      expect(() => tracker.recordQuery("non-existent", query)).not.toThrow();
    });
  });

  describe("updateTokens()", () => {
    it("should update current token count", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      tracker.updateTokens("msg-1", 250, session);

      const active = tracker.getActiveMessage("msg-1")!;
      expect(active.currentTokens).toBe(250);
    });

    it("should not create checkpoint before threshold", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      // Default checkpoint interval is 500
      const checkpoint = tracker.updateTokens("msg-1", 250, session);

      expect(checkpoint).toBeNull();
    });

    it("should create checkpoint at threshold", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      // Update to 500+ to trigger checkpoint
      const checkpoint = tracker.updateTokens("msg-1", 500, session);

      expect(checkpoint).not.toBeNull();
      expect(checkpoint?.index).toBe(0);
      expect(checkpoint?.tokensProcessed).toBe(500);
    });

    it("should track budget availability in checkpoint", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession("session-1", 0n, 1000000000000000n);
      const checkpoint = tracker.updateTokens("msg-1", 500, session);

      expect(checkpoint?.budgetAvailable).toBe(true);
      expect(checkpoint?.remainingBudget).toBeDefined();
    });

    it("should indicate budget exhausted when over limit", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      // Session already at limit
      const session = createTestSession(
        "session-1",
        1000000000000000n,
        1000000000000000n,
      );
      const checkpoint = tracker.updateTokens("msg-1", 500, session);

      expect(checkpoint?.budgetAvailable).toBe(false);
    });

    it("should return null for non-existent message", () => {
      const tracker = createFreshTracker();
      const session = createTestSession();
      const checkpoint = tracker.updateTokens("non-existent", 500, session);

      expect(checkpoint).toBeNull();
    });
  });

  describe("completeMessage()", () => {
    it("should calculate final message cost", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));
      tracker.recordQuery("msg-1", createTestQueryCost(100n));

      const session = createTestSession();
      const cost = tracker.completeMessage("msg-1", 100, 1, session);

      expect(cost.messageId).toBe("msg-1");
      expect(cost.sessionId).toBe("session-1");
      expect(cost.tokenCount).toBe(100);
      expect(cost.reasoningDepth).toBe(1);
    });

    it("should include breakdown of costs", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));
      tracker.recordQuery("msg-1", createTestQueryCost(100n));

      const session = createTestSession();
      const cost = tracker.completeMessage("msg-1", 100, 1, session);

      expect(cost.breakdown.queryCost).toBe(100n);
      expect(cost.breakdown.queryCount).toBe(1);
      expect(cost.breakdown.tokenCost).toBeGreaterThan(0n);
    });

    it("should apply complexity multiplier", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      const cost = tracker.completeMessage("msg-1", 3000, 3, session);

      expect(cost.complexityMultiplier).toBeGreaterThan(1.0);
      expect(cost.breakdown.complexityPremium).toBeGreaterThan(0n);
    });

    it("should calculate research contribution (22%)", () => {
      const tracker = createFreshTracker({ freeTierMessagesPerSession: 0 });
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      const cost = tracker.completeMessage("msg-1", 100, 1, session);

      // 22% of total cost
      const expectedResearch = BigInt(
        Math.floor(Number(cost.totalCost) * 0.22),
      );
      expect(cost.researchContribution).toBe(expectedResearch);
    });

    it("should calculate knowledge rewards (30%)", () => {
      const tracker = createFreshTracker({ freeTierMessagesPerSession: 0 });
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      const cost = tracker.completeMessage("msg-1", 100, 1, session);

      const expectedRewards = BigInt(Math.floor(Number(cost.totalCost) * 0.3));
      expect(cost.knowledgeRewards).toBe(expectedRewards);
    });

    it("should store cost in message history", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      tracker.completeMessage("msg-1", 100, 1, session);

      const history = tracker.getSessionHistory("session-1");
      expect(history.length).toBe(1);
      expect(history[0].messageId).toBe("msg-1");
    });

    it("should increment session message count", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      tracker.completeMessage("msg-1", 100, 1, session);

      tracker.startMessage(createTestMessage("msg-2"));
      const remaining = tracker.getRemainingFreeTierMessages("session-1");

      expect(remaining).toBe(4); // 5 - 1 = 4
    });

    it("should auto-estimate reasoning depth if not provided", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));
      tracker.recordQuery("msg-1", createTestQueryCost(), [
        "fact-1",
        "fact-2",
        "fact-3",
      ]);

      const session = createTestSession();
      const cost = tracker.completeMessage("msg-1", 100, undefined, session);

      expect(cost.reasoningDepth).toBeGreaterThanOrEqual(1);
    });

    it("should throw for non-existent message", () => {
      const tracker = createFreshTracker();
      const session = createTestSession();

      expect(() =>
        tracker.completeMessage("non-existent", 100, 1, session),
      ).toThrow("Message non-existent not found");
    });

    it("should remove message from active messages", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      tracker.completeMessage("msg-1", 100, 1, session);

      expect(tracker.getActiveMessage("msg-1")).toBeUndefined();
    });
  });

  describe("abortMessage()", () => {
    it("should return provisional cost", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));
      tracker.recordQuery("msg-1", createTestQueryCost(100n));

      const partialCost = tracker.abortMessage("msg-1");

      expect(partialCost).toBeGreaterThan(0n);
    });

    it("should remove message from active messages", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));
      tracker.abortMessage("msg-1");

      expect(tracker.getActiveMessage("msg-1")).toBeUndefined();
    });

    it("should return 0 for non-existent message", () => {
      const tracker = createFreshTracker();
      const partialCost = tracker.abortMessage("non-existent");
      expect(partialCost).toBe(0n);
    });
  });

  describe("Free Tier", () => {
    it("should mark first 5 messages as free tier", () => {
      const tracker = createFreshTracker();
      const session = createTestSession();

      for (let i = 1; i <= 5; i++) {
        tracker.startMessage(createTestMessage(`msg-${i}`));
        const cost = tracker.completeMessage(`msg-${i}`, 500, 1, session);
        expect(cost.freeTier).toBe(true);
        expect(cost.totalCost).toBe(0n);
      }
    });

    it("should charge for messages after free tier", () => {
      const tracker = createFreshTracker();
      const session = createTestSession();

      // Use up free tier
      for (let i = 1; i <= 5; i++) {
        tracker.startMessage(createTestMessage(`msg-${i}`));
        tracker.completeMessage(`msg-${i}`, 500, 1, session);
      }

      // 6th message should be charged
      tracker.startMessage(createTestMessage("msg-6"));
      const cost = tracker.completeMessage("msg-6", 500, 1, session);

      expect(cost.freeTier).toBe(false);
      expect(cost.totalCost).toBeGreaterThan(0n);
    });

    it("should charge if tokens exceed free tier max", () => {
      const tracker = createFreshTracker();
      const session = createTestSession();

      tracker.startMessage(createTestMessage("msg-1"));
      // Exceed free tier max tokens (1000)
      const cost = tracker.completeMessage("msg-1", 1500, 1, session);

      expect(cost.freeTier).toBe(false);
      expect(cost.totalCost).toBeGreaterThan(0n);
    });

    it("should track remaining free tier messages", () => {
      const tracker = createFreshTracker();
      const session = createTestSession();

      expect(tracker.getRemainingFreeTierMessages("session-1")).toBe(5);

      tracker.startMessage(createTestMessage("msg-1"));
      tracker.completeMessage("msg-1", 500, 1, session);

      expect(tracker.getRemainingFreeTierMessages("session-1")).toBe(4);
    });
  });

  describe("estimateMessageCost()", () => {
    it("should estimate cost before execution", () => {
      const tracker = createFreshTracker();
      const session = createTestSession();

      const estimate = tracker.estimateMessageCost(
        "session-1",
        500,
        2,
        session,
      );

      expect(estimate.estimatedBaseCost).toBeGreaterThan(0n);
      expect(estimate.estimatedTotalCost).toBeDefined();
      expect(estimate.estimatedMultiplier).toBeGreaterThanOrEqual(1.0);
    });

    it("should predict free tier eligibility", () => {
      const tracker = createFreshTracker();
      const session = createTestSession();

      const estimate = tracker.estimateMessageCost(
        "session-1",
        500,
        2,
        session,
      );

      expect(estimate.wouldBeFreeTier).toBe(true);
      expect(estimate.estimatedTotalCost).toBe(0n);
    });

    it("should check budget availability", () => {
      const tracker = createFreshTracker();
      const session = createTestSession("session-1", 0n, 1000n);

      const estimate = tracker.estimateMessageCost(
        "session-1",
        500,
        2,
        session,
      );

      expect(estimate.hasBudget).toBeDefined();
      expect(estimate.remainingBudget).toBeDefined();
    });
  });

  describe("canAffordMessage()", () => {
    it("should return true when within budget", () => {
      const tracker = createFreshTracker();
      const session = createTestSession("session-1", 0n, 1000000000000000000n);

      const canAfford = tracker.canAffordMessage(session, 500, 2);

      expect(canAfford).toBe(true);
    });

    it("should return true when no budget limit", () => {
      const tracker = createFreshTracker();
      const session = createTestSession("session-1", 0n, undefined);

      const canAfford = tracker.canAffordMessage(session, 10000, 10);

      expect(canAfford).toBe(true);
    });

    it("should return false when over budget", () => {
      const tracker = createFreshTracker();
      // Session with very small budget
      const session = createTestSession("session-1", 0n, 1n);

      // Use up free tier
      for (let i = 0; i < 5; i++) {
        tracker.startMessage(createTestMessage(`msg-${i}`, "session-1"));
        tracker.completeMessage(`msg-${i}`, 100, 1, session);
      }

      const canAfford = tracker.canAffordMessage(session, 1000, 5);

      expect(canAfford).toBe(false);
    });
  });

  describe("Session Management", () => {
    describe("getSessionHistory()", () => {
      it("should return empty array for new session", () => {
        const tracker = createFreshTracker();
        const history = tracker.getSessionHistory("new-session");
        expect(history).toEqual([]);
      });

      it("should return completed messages", () => {
        const tracker = createFreshTracker();
        const session = createTestSession();

        tracker.startMessage(createTestMessage("msg-1"));
        tracker.completeMessage("msg-1", 100, 1, session);
        tracker.startMessage(createTestMessage("msg-2"));
        tracker.completeMessage("msg-2", 200, 2, session);

        const history = tracker.getSessionHistory("session-1");
        expect(history.length).toBe(2);
      });
    });

    describe("getSessionMessageCost()", () => {
      it("should return total cost of all messages", () => {
        const tracker = createFreshTracker();
        const session = createTestSession();

        // Exhaust free tier first
        for (let i = 1; i <= 5; i++) {
          tracker.startMessage(createTestMessage(`free-${i}`));
          tracker.completeMessage(`free-${i}`, 100, 1, session);
        }

        // Now messages cost money
        tracker.startMessage(createTestMessage("paid-1"));
        const cost1 = tracker.completeMessage("paid-1", 500, 1, session);
        tracker.startMessage(createTestMessage("paid-2"));
        const cost2 = tracker.completeMessage("paid-2", 500, 1, session);

        const totalCost = tracker.getSessionMessageCost("session-1");
        expect(totalCost).toBe(cost1.totalCost + cost2.totalCost);
      });
    });

    describe("getSessionStats()", () => {
      it("should return zero stats for empty session", () => {
        const tracker = createFreshTracker();
        const stats = tracker.getSessionStats("empty-session");

        expect(stats.messageCount).toBe(0);
        expect(stats.totalCost).toBe(0n);
        expect(stats.totalTokens).toBe(0);
        expect(stats.avgComplexityMultiplier).toBe(1.0);
      });

      it("should calculate correct stats", () => {
        const tracker = createFreshTracker();
        const session = createTestSession();

        tracker.startMessage(createTestMessage("msg-1"));
        tracker.recordQuery("msg-1", createTestQueryCost());
        tracker.completeMessage("msg-1", 500, 1, session);

        tracker.startMessage(createTestMessage("msg-2"));
        tracker.recordQuery("msg-2", createTestQueryCost());
        tracker.recordQuery("msg-2", createTestQueryCost());
        tracker.completeMessage("msg-2", 1000, 2, session);

        const stats = tracker.getSessionStats("session-1");

        expect(stats.messageCount).toBe(2);
        expect(stats.totalTokens).toBe(1500);
        expect(stats.totalQueries).toBe(3);
        expect(stats.freeTierMessages).toBe(2);
      });
    });

    describe("clearSession()", () => {
      it("should remove all session data", () => {
        const tracker = createFreshTracker();
        const session = createTestSession();

        tracker.startMessage(createTestMessage("msg-1"));
        tracker.completeMessage("msg-1", 100, 1, session);
        tracker.startMessage(createTestMessage("msg-2")); // Still active

        tracker.clearSession("session-1");

        expect(tracker.getSessionHistory("session-1")).toEqual([]);
        expect(tracker.getRemainingFreeTierMessages("session-1")).toBe(5);
        expect(tracker.getActiveMessage("msg-2")).toBeUndefined();
      });
    });

    describe("getActiveMessageIds()", () => {
      it("should return all active message IDs", () => {
        const tracker = createFreshTracker();
        tracker.startMessage(createTestMessage("msg-1"));
        tracker.startMessage(createTestMessage("msg-2"));
        tracker.startMessage(createTestMessage("msg-3"));

        const ids = tracker.getActiveMessageIds();

        expect(ids).toContain("msg-1");
        expect(ids).toContain("msg-2");
        expect(ids).toContain("msg-3");
        expect(ids.length).toBe(3);
      });
    });
  });

  describe("Configuration", () => {
    describe("getConfig()", () => {
      it("should return current configuration", () => {
        const tracker = createFreshTracker();
        const config = tracker.getConfig();

        expect(config.baseMessageCost).toBe(
          DEFAULT_MESSAGE_BILLING_CONFIG.baseMessageCost,
        );
        expect(config.freeTierMessagesPerSession).toBe(5);
      });

      it("should return a copy (not reference)", () => {
        const tracker = createFreshTracker();
        const config = tracker.getConfig();
        config.freeTierMessagesPerSession = 100;

        expect(tracker.getConfig().freeTierMessagesPerSession).toBe(5);
      });
    });

    describe("updateConfig()", () => {
      it("should update configuration", () => {
        const tracker = createFreshTracker();
        tracker.updateConfig({ freeTierMessagesPerSession: 10 });

        expect(tracker.getConfig().freeTierMessagesPerSession).toBe(10);
      });

      it("should preserve other config values", () => {
        const tracker = createFreshTracker();
        const originalBase = tracker.getConfig().baseMessageCost;
        tracker.updateConfig({ freeTierMessagesPerSession: 10 });

        expect(tracker.getConfig().baseMessageCost).toBe(originalBase);
      });
    });
  });

  describe("Edge Cases", () => {
    it("should handle zero token count", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      const cost = tracker.completeMessage("msg-1", 0, 1, session);

      expect(cost.tokenCount).toBe(0);
      expect(cost.breakdown.tokenCost).toBe(0n);
    });

    it("should handle very large token counts", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();
      const cost = tracker.completeMessage("msg-1", 100000, 5, session);

      // Should not throw and should have high multiplier
      expect(cost.complexityMultiplier).toBeGreaterThan(2.0);
    });

    it("should handle multiple sessions independently", () => {
      const tracker = createFreshTracker();
      const session1 = createTestSession("session-1");
      const session2 = createTestSession("session-2");

      tracker.startMessage(createTestMessage("msg-1", "session-1"));
      tracker.completeMessage("msg-1", 100, 1, session1);

      tracker.startMessage(createTestMessage("msg-2", "session-2"));
      tracker.completeMessage("msg-2", 200, 2, session2);

      expect(tracker.getSessionHistory("session-1").length).toBe(1);
      expect(tracker.getSessionHistory("session-2").length).toBe(1);
      expect(tracker.getRemainingFreeTierMessages("session-1")).toBe(4);
      expect(tracker.getRemainingFreeTierMessages("session-2")).toBe(4);
    });

    it("should handle BigInt arithmetic correctly", () => {
      const tracker = createFreshTracker();
      tracker.startMessage(createTestMessage("msg-1"));

      // Use large query costs
      tracker.recordQuery("msg-1", createTestQueryCost(1000000000000000n));
      tracker.recordQuery("msg-1", createTestQueryCost(2000000000000000n));

      const session = createTestSession();
      // Should not overflow
      expect(() =>
        tracker.completeMessage("msg-1", 1000, 1, session),
      ).not.toThrow();
    });

    it("should include streaming fees in total cost", () => {
      const tracker = createFreshTracker({ freeTierMessagesPerSession: 0 });
      tracker.startMessage(createTestMessage("msg-1"));

      const session = createTestSession();

      // Trigger multiple checkpoints
      for (let tokens = 500; tokens <= 1500; tokens += 500) {
        tracker.updateTokens("msg-1", tokens, session);
      }

      const cost = tracker.completeMessage("msg-1", 1500, 1, session);

      expect(cost.breakdown.streamingFees).toBeGreaterThan(0n);
    });
  });
});
