/**
 * Fact Expiration Tests
 *
 * Tests for GAP-009: Fact Expiration
 * Ensures facts properly expire based on:
 * 1. Explicit expiration (expiresAt < currentBlock)
 * 2. Confidence decay (confidence < minConfidenceThreshold)
 * 3. Reverification staleness (age > 2 * decayHalfLife without reverification)
 */

import { describe, it, expect } from "vitest";
import {
  KnowledgeStateTree,
  DEFAULT_EXPIRATION_PARAMS,
  type ExpirationParams,
} from "../../../state/knowledge.js";
import type { FactState, FactStatus } from "../../../state/types.js";
import type { Confidence } from "../../../core/knowledge/types.js";

// ============ Test Helpers ============

let _testIdCounter = 0;
function uniqueId(): string {
  _testIdCounter++;
  return `${_testIdCounter}-${Date.now()}-${Math.random().toString(36).substring(2, 8)}`;
}

function createTestKnowledgeTree(): KnowledgeStateTree {
  return new KnowledgeStateTree();
}

function createTestFact(
  id: string,
  options: {
    confidence?: number;
    status?: FactStatus;
    verifiedAt?: bigint;
    expiresAt?: bigint;
    lastReverifiedAt?: bigint;
    stratum?:
      | "fundamental"
      | "physical"
      | "chemical"
      | "biological"
      | "cognitive"
      | "social"
      | "technological";
  } = {},
): Omit<FactState, "referenceRoot"> {
  return {
    id,
    claim: `Test claim for ${id}`,
    domain: "test.domain",
    confidence: (options.confidence ?? 0.85) as Confidence,
    verifiedAt: options.verifiedAt ?? 100n,
    verifierCount: 5,
    challengeCount: 0,
    citationCount: 0,
    status: options.status ?? "active",
    expiresAt: options.expiresAt,
    lastReverifiedAt: options.lastReverifiedAt,
    stratum: options.stratum,
  };
}

// ============ Tests ============

describe("GAP-009: Fact Expiration", () => {
  describe("Expiration Detection", () => {
    it("should detect explicitly expired facts", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-explicit-${uniqueId()}`;

      // Add fact that expires at block 500
      tree.addFact(
        createTestFact(factId, {
          expiresAt: 500n,
          verifiedAt: 100n,
        }),
      );

      const fact = tree.getFact(factId)!;

      // Before expiration
      const resultBefore = tree.isFactExpired(fact, 499n);
      expect(resultBefore.expired).toBe(false);

      // At expiration
      const resultAt = tree.isFactExpired(fact, 500n);
      expect(resultAt.expired).toBe(true);
      expect(resultAt.reason).toBe("explicit_expiration");

      // After expiration
      const resultAfter = tree.isFactExpired(fact, 600n);
      expect(resultAfter.expired).toBe(true);
      expect(resultAfter.reason).toBe("explicit_expiration");
      expect(resultAfter.details?.blocksFromExpiration).toBe(100n);
    });

    it("should detect confidence-decayed facts", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-decay-${uniqueId()}`;

      // Add fact with confidence below threshold (0.3)
      tree.addFact(
        createTestFact(factId, {
          confidence: 0.25, // Below default threshold of 0.3
        }),
      );

      const fact = tree.getFact(factId)!;
      const result = tree.isFactExpired(fact, 1000n);

      expect(result.expired).toBe(true);
      expect(result.reason).toBe("confidence_decay");
      expect(result.details?.currentConfidence).toBe(0.25);
      expect(result.details?.threshold).toBe(0.3);
    });

    it("should detect stale reverification-required facts", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-stale-${uniqueId()}`;

      // Add empirical fact (requires reverification, decayHalfLife = 100000)
      // Staleness threshold = 2 * 100000 = 200000 blocks
      tree.addFact(
        createTestFact(factId, {
          stratum: "physical",
          verifiedAt: 0n,
          confidence: 0.85, // Above threshold
        }),
      );

      const fact = tree.getFact(factId)!;

      // Just under staleness threshold
      const resultBefore = tree.isFactExpired(fact, 199999n);
      expect(resultBefore.expired).toBe(false);

      // At staleness threshold
      const resultAt = tree.isFactExpired(fact, 200001n);
      expect(resultAt.expired).toBe(true);
      expect(resultAt.reason).toBe("reverification_required");
    });

    it("should NOT expire eternal category facts (formal, analytic)", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-eternal-${uniqueId()}`;

      // Add formal/logical fact (no decay, no reverification required)
      tree.addFact(
        createTestFact(factId, {
          stratum: "fundamental",
          verifiedAt: 0n,
          confidence: 0.95,
        }),
      );

      const fact = tree.getFact(factId)!;

      // Even after very long time, should not expire
      const result = tree.isFactExpired(fact, 10_000_000n);
      expect(result.expired).toBe(false);
    });

    it("should not consider non-active facts as expired", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-revoked-${uniqueId()}`;

      // Add and then revoke a fact
      tree.addFact(
        createTestFact(factId, {
          confidence: 0.1, // Would be below threshold if active
        }),
      );
      tree.revokeFact(factId);

      const fact = tree.getFact(factId)!;
      expect(fact.status).toBe("revoked");

      // Revoked facts shouldn't be "expired" - they're already not active
      const result = tree.isFactExpired(fact, 1000n);
      expect(result.expired).toBe(false);
    });

    it("should use lastReverifiedAt when available for staleness check", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-reverified-${uniqueId()}`;

      // Add empirical fact verified at block 0, reverified at block 150000
      tree.addFact(
        createTestFact(factId, {
          stratum: "physical",
          verifiedAt: 0n,
          lastReverifiedAt: 150000n,
          confidence: 0.85,
        }),
      );

      const fact = tree.getFact(factId)!;

      // Without reverification, would be stale at 200001
      // With reverification at 150000, stale at 150000 + 200000 = 350001
      const resultBefore = tree.isFactExpired(fact, 349999n);
      expect(resultBefore.expired).toBe(false);

      const resultAfter = tree.isFactExpired(fact, 350001n);
      expect(resultAfter.expired).toBe(true);
      expect(resultAfter.reason).toBe("reverification_required");
    });
  });

  describe("Query Filtering", () => {
    it("should filter expired facts using getExpiredFacts", () => {
      const tree = createTestKnowledgeTree();
      const expiredId = `fact-expired-${uniqueId()}`;
      const activeId = `fact-active-${uniqueId()}`;

      // Add one expired fact (low confidence)
      tree.addFact(createTestFact(expiredId, { confidence: 0.1 }));

      // Add one active fact
      tree.addFact(createTestFact(activeId, { confidence: 0.9 }));

      const expired = tree.getExpiredFacts(1000n);

      expect(expired.length).toBe(1);
      expect(expired[0].fact.id).toBe(expiredId);
      expect(expired[0].result.reason).toBe("confidence_decay");
    });

    it("should include already-expired facts in getExpiredFacts", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-already-expired-${uniqueId()}`;

      tree.addFact(createTestFact(factId, { status: "expired" }));

      const expired = tree.getExpiredFacts(1000n);

      expect(expired.length).toBe(1);
      expect(expired[0].fact.status).toBe("expired");
    });

    it("should mark expired facts with reason in results", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-explicit-exp-${uniqueId()}`;

      tree.addFact(createTestFact(factId, { expiresAt: 500n }));

      const expired = tree.getExpiredFacts(600n);

      expect(expired.length).toBe(1);
      expect(expired[0].result.expired).toBe(true);
      expect(expired[0].result.reason).toBe("explicit_expiration");
    });
  });

  describe("Lazy Expiration", () => {
    it("should update fact status to expired on query with updateIfExpired", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-lazy-${uniqueId()}`;

      tree.addFact(createTestFact(factId, { expiresAt: 500n }));

      // Initial status should be active
      const factBefore = tree.getFact(factId)!;
      expect(factBefore.status).toBe("active");

      // Query with expiration check and update
      const factAfter = tree.getFactWithExpiration(factId, {
        checkExpiration: true,
        currentBlock: 600n,
        updateIfExpired: true,
      });

      expect(factAfter?.status).toBe("expired");

      // Verify the underlying fact was updated
      const factFinal = tree.getFact(factId)!;
      expect(factFinal.status).toBe("expired");
    });

    it("should not modify fact if not expired", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-not-exp-${uniqueId()}`;

      tree.addFact(createTestFact(factId, { confidence: 0.9 }));

      const fact = tree.getFactWithExpiration(factId, {
        checkExpiration: true,
        currentBlock: 1000n,
        updateIfExpired: true,
      });

      expect(fact?.status).toBe("active");
    });

    it("should not update when updateIfExpired is false", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-no-update-${uniqueId()}`;

      tree.addFact(createTestFact(factId, { expiresAt: 500n }));

      const fact = tree.getFactWithExpiration(factId, {
        checkExpiration: true,
        currentBlock: 600n,
        updateIfExpired: false,
      });

      // Should still return active (not updated)
      expect(fact?.status).toBe("active");

      // Underlying fact should still be active
      const factUnderlying = tree.getFact(factId)!;
      expect(factUnderlying.status).toBe("active");
    });
  });

  describe("Fact Expiration", () => {
    it("should mark fact as expired with expireFact", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-to-expire-${uniqueId()}`;

      tree.addFact(createTestFact(factId));

      tree.expireFact(factId, 1000n, "confidence_decay");

      const fact = tree.getFact(factId)!;
      expect(fact.status).toBe("expired");
    });
  });

  describe("Reverification", () => {
    it("should reset staleness clock with reverifyFact", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-reverify-${uniqueId()}`;

      tree.addFact(
        createTestFact(factId, {
          stratum: "physical",
          verifiedAt: 0n,
          confidence: 0.85,
        }),
      );

      // Reverify at block 100000
      tree.reverifyFact(factId, 100000n);

      const fact = tree.getFact(factId)!;
      expect(fact.lastReverifiedAt).toBe(100000n);
      expect(fact.verifierCount).toBe(6); // Initial 5 + 1

      // Should not be stale until 100000 + 200000 = 300000
      const result = tree.isFactExpired(fact, 299999n);
      expect(result.expired).toBe(false);
    });

    it("should reactivate expired facts on reverification", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-reactivate-${uniqueId()}`;

      tree.addFact(createTestFact(factId, { status: "expired" }));

      tree.reverifyFact(factId, 1000n, 0.9 as Confidence);

      const fact = tree.getFact(factId)!;
      expect(fact.status).toBe("active");
      expect(fact.confidence).toBe(0.9);
      expect(fact.lastReverifiedAt).toBe(1000n);
    });
  });

  describe("Custom Expiration Parameters", () => {
    it("should respect custom minConfidenceThreshold", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-custom-threshold-${uniqueId()}`;

      tree.addFact(createTestFact(factId, { confidence: 0.4 }));

      const fact = tree.getFact(factId)!;

      // Default threshold (0.3) - should not be expired
      const resultDefault = tree.isFactExpired(fact, 1000n);
      expect(resultDefault.expired).toBe(false);

      // Custom higher threshold (0.5) - should be expired
      const customParams: ExpirationParams = {
        ...DEFAULT_EXPIRATION_PARAMS,
        minConfidenceThreshold: 0.5,
      };
      const resultCustom = tree.isFactExpired(fact, 1000n, customParams);
      expect(resultCustom.expired).toBe(true);
      expect(resultCustom.reason).toBe("confidence_decay");
    });

    it("should respect custom stalenessMultiplier", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-custom-staleness-${uniqueId()}`;

      tree.addFact(
        createTestFact(factId, {
          stratum: "physical",
          verifiedAt: 0n,
          confidence: 0.85,
        }),
      );

      const fact = tree.getFact(factId)!;

      // Default multiplier (2.0): stale at 200000
      // Custom multiplier (1.0): stale at 100000
      const customParams: ExpirationParams = {
        ...DEFAULT_EXPIRATION_PARAMS,
        stalenessMultiplier: 1.0,
      };

      const resultAt100k = tree.isFactExpired(fact, 100001n, customParams);
      expect(resultAt100k.expired).toBe(true);
      expect(resultAt100k.reason).toBe("reverification_required");
    });

    it("should skip stratum policy when useStratumPolicy is false", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-no-stratum-${uniqueId()}`;

      tree.addFact(
        createTestFact(factId, {
          stratum: "physical",
          verifiedAt: 0n,
          confidence: 0.85,
        }),
      );

      const fact = tree.getFact(factId)!;

      const customParams: ExpirationParams = {
        ...DEFAULT_EXPIRATION_PARAMS,
        useStratumPolicy: false,
      };

      // Even at very old age, should not expire without stratum check
      const result = tree.isFactExpired(fact, 500000n, customParams);
      expect(result.expired).toBe(false);
    });
  });

  describe("Edge Cases", () => {
    it("should handle facts without stratum", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-no-stratum-${uniqueId()}`;

      tree.addFact(
        createTestFact(factId, {
          confidence: 0.85,
          // No stratum specified
        }),
      );

      const fact = tree.getFact(factId)!;

      // Should not trigger reverification_required without stratum
      const result = tree.isFactExpired(fact, 1_000_000n);
      expect(result.expired).toBe(false);
    });

    it("should handle fact with all expiration conditions set", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-all-conditions-${uniqueId()}`;

      // Fact with explicit expiration, low confidence, and old age
      tree.addFact(
        createTestFact(factId, {
          expiresAt: 100n,
          confidence: 0.1,
          stratum: "physical",
          verifiedAt: 0n,
        }),
      );

      const fact = tree.getFact(factId)!;

      // First matching condition should be returned
      // (explicit_expiration is checked first)
      const result = tree.isFactExpired(fact, 500n);
      expect(result.expired).toBe(true);
      expect(result.reason).toBe("explicit_expiration");
    });

    it("should not crash on very large block numbers", () => {
      const tree = createTestKnowledgeTree();
      const factId = `fact-large-block-${uniqueId()}`;

      tree.addFact(
        createTestFact(factId, {
          stratum: "physical",
          verifiedAt: 0n,
        }),
      );

      const fact = tree.getFact(factId)!;

      // Very large block number
      const result = tree.isFactExpired(fact, 2n ** 62n);
      expect(result.expired).toBe(true);
      expect(result.reason).toBe("reverification_required");
    });
  });
});
