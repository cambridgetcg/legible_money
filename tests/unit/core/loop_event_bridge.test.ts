/**
 * Loop Event Bridge Tests
 *
 * Tests for the LoopEventBridge that converts payment/state events
 * into LoopController events for sustainability tracking.
 */

import { describe, it, expect, vi } from "vitest";
import { LoopEventBridge } from "../../../core/genesis/loop_event_bridge.js";
import type { PaymentExecutionResult } from "../../../core/billing/payment_executor.js";
import type { PaymentDistribution } from "../../../core/billing/types.js";

// ============ Helpers ============

function createMockDistribution(): PaymentDistribution {
  return {
    apiProvider: { did: "did:lgm:provider" as any, amount: 100n },
    knowledgeRewards: [],
    protocol: { burned: 5n, treasury: 5n },
    research: { amount: 22n, treasury: "did:lgm:research" as any },
  };
}

function createBridgeWithMock() {
  const processEvent = vi.fn();
  const mockController = {
    processEvent,
    getState: vi.fn(() => ({ health: { score: 1 } })),
    getEfficiency: vi.fn(),
    getSustainabilityForecast: vi.fn(),
  };
  const bridge = new LoopEventBridge(mockController as any);
  return { bridge, processEvent };
}

// ============ Tests ============

describe("LoopEventBridge", () => {
  describe("onQueryExecuted", () => {
    it("should emit APICallEvent on successful payment", () => {
      const { bridge, processEvent } = createBridgeWithMock();
      const result: PaymentExecutionResult = {
        success: true,
        transfers: [
          {
            from: "caller",
            to: "did:lgm:c1",
            amount: 30n,
            reason: "direct_citation",
          },
          { from: "caller", to: "did:lgm:c2", amount: 15n, reason: "ancestor" },
          {
            from: "caller",
            to: "provider",
            amount: 100n,
            reason: "api_service",
          },
        ],
        totalTransferred: 172n,
        researchFunded: 22n,
        errors: [],
      };

      bridge.onQueryExecuted("query-1", createMockDistribution(), result, [
        "fact-1",
        "fact-2",
      ]);

      expect(processEvent).toHaveBeenCalledOnce();
      const event = processEvent.mock.calls[0][0];
      expect(event.type).toBe("api_call");
      expect(event.callId).toBe("query-1");
      expect(event.revenue).toBe(172n);
      expect(event.factsUsed).toEqual(["fact-1", "fact-2"]);
      expect(event.researchShare).toBe(22n);
      // Only citation rewards in knowledge distribution
      expect(event.knowledgeDistribution.get("did:lgm:c1")).toBe(30n);
      expect(event.knowledgeDistribution.get("did:lgm:c2")).toBe(15n);
      expect(event.knowledgeDistribution.has("provider")).toBe(false);
    });

    it("should not emit event on failed payment", () => {
      const { bridge, processEvent } = createBridgeWithMock();
      const result: PaymentExecutionResult = {
        success: false,
        transfers: [],
        totalTransferred: 0n,
        researchFunded: 0n,
        errors: ["Insufficient balance"],
      };

      bridge.onQueryExecuted("query-1", createMockDistribution(), result, [
        "fact-1",
      ]);

      expect(processEvent).not.toHaveBeenCalled();
    });
  });

  describe("onClaimVerified", () => {
    it("should emit ClaimVerifiedEvent with accepted outcome", () => {
      const { bridge, processEvent } = createBridgeWithMock();
      const rewards = new Map([
        ["v1", 100n],
        ["v2", 80n],
      ]);

      bridge.onClaimVerified("claim-1", "accepted", "fact-1", rewards, 22n);

      expect(processEvent).toHaveBeenCalledOnce();
      const event = processEvent.mock.calls[0][0];
      expect(event.type).toBe("claim_verified");
      expect(event.claimId).toBe("claim-1");
      expect(event.outcome).toBe("accepted");
      expect(event.factId).toBe("fact-1");
      expect(event.researchShare).toBe(22n);
    });

    it("should emit ClaimVerifiedEvent with rejected outcome", () => {
      const { bridge, processEvent } = createBridgeWithMock();

      bridge.onClaimVerified("claim-2", "rejected", undefined, new Map(), 0n);

      expect(processEvent).toHaveBeenCalledOnce();
      const event = processEvent.mock.calls[0][0];
      expect(event.type).toBe("claim_verified");
      expect(event.outcome).toBe("rejected");
      expect(event.factId).toBeUndefined();
    });
  });

  describe("onBlockProduced", () => {
    it("should emit BlockProducedEvent", () => {
      const { bridge, processEvent } = createBridgeWithMock();
      const validatorRewards = new Map([["v1", 50n]]);

      bridge.onBlockProduced(100n, 1000n, 220n, validatorRewards);

      expect(processEvent).toHaveBeenCalledOnce();
      const event = processEvent.mock.calls[0][0];
      expect(event.type).toBe("block_produced");
      expect(event.blockNumber).toBe(100n);
      expect(event.blockReward).toBe(1000n);
      expect(event.researchShare).toBe(220n);
    });
  });

  describe("onBountyCreated", () => {
    it("should emit BountyCreatedEvent", () => {
      const { bridge, processEvent } = createBridgeWithMock();

      bridge.onBountyCreated("bounty-1", 500n, "claim-1", "science.physics");

      expect(processEvent).toHaveBeenCalledOnce();
      const event = processEvent.mock.calls[0][0];
      expect(event.type).toBe("bounty_created");
      expect(event.bountyId).toBe("bounty-1");
      expect(event.value).toBe(500n);
      expect(event.targetClaim).toBe("claim-1");
      expect(event.domain).toBe("science.physics");
    });
  });

  describe("onBountyFulfilled", () => {
    it("should emit BountyFulfilledEvent with confirms outcome", () => {
      const { bridge, processEvent } = createBridgeWithMock();

      bridge.onBountyFulfilled(
        "bounty-1",
        "did:lgm:researcher",
        "research-1",
        "confirms",
      );

      expect(processEvent).toHaveBeenCalledOnce();
      const event = processEvent.mock.calls[0][0];
      expect(event.type).toBe("bounty_fulfilled");
      expect(event.bountyId).toBe("bounty-1");
      expect(event.fulfiller).toBe("did:lgm:researcher");
      expect(event.outcome).toBe("confirms");
    });

    it("should emit BountyFulfilledEvent with fraud_detected outcome", () => {
      const { bridge, processEvent } = createBridgeWithMock();

      bridge.onBountyFulfilled(
        "bounty-2",
        "did:lgm:auditor",
        "research-2",
        "fraud_detected",
      );

      expect(processEvent).toHaveBeenCalledOnce();
      const event = processEvent.mock.calls[0][0];
      expect(event.type).toBe("bounty_fulfilled");
      expect(event.outcome).toBe("fraud_detected");
    });
  });
});
