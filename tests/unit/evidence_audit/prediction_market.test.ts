/**
 * Prediction Market Evidence Integration Tests
 *
 * Tests for type integration, hashing, validation, confidence calculation,
 * evidence processing, PoT mapping, registry, and dispute windows.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  EVIDENCE_TYPE_STRENGTH,
  toCanonicalEvidenceType,
  DEFAULT_EXPIRY_CONFIGS,
} from "../../../core/evidence/types.js";
import type {
  PredictionMarketMetadata,
  PredictionMarketPlatform,
  PredictionMarketRegistration,
} from "../../../core/evidence/types.js";
import {
  PREDICTION_MARKET_CONFIDENCE,
  PLATFORM_REPUTATION,
  PLATFORM_DISPUTE_WINDOWS,
  createMarketRegistration,
  hashMarketResolution,
  validateMarketForEvidence,
  processPredictionMarketResolution,
  calculatePredictionMarketConfidence,
  mapPredictionMarketToPoTConfidence,
  isDisputeWindowElapsed,
  PredictionMarketRegistry,
} from "../../../core/evidence/prediction_market.js";
import type { DID } from "../../../core/agents/identity/did.js";

// ============ Test Helpers ============

const TEST_SUBMITTER = "did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa1" as DID;

function makeMarketMetadata(
  overrides: Partial<PredictionMarketMetadata> = {},
): PredictionMarketMetadata {
  return {
    platform: "polymarket",
    marketId: "market-001",
    question: "Will BTC exceed $100k by end of 2026?",
    marketType: "binary_outcome",
    resolutionStatus: "resolved",
    resolvedOutcome: "Yes",
    resolvedAt: 1707600000000,
    volumeUsd: 500_000,
    traderCount: 1200,
    closingProbability: 0.72,
    resolutionSource: "UMA Oracle",
    createdAt: 1704067200000,
    closedAt: 1707500000000,
    disputeWindowEnd: 50000n,
    disputeWindowElapsed: true,
    ...overrides,
  };
}

function makeRegistration(
  overrides: Partial<PredictionMarketRegistration> = {},
): PredictionMarketRegistration {
  return {
    platform: "polymarket",
    marketId: "market-001",
    question: "Will BTC exceed $100k by end of 2026?",
    marketType: "binary_outcome",
    marketUrl: "https://polymarket.com/market/market-001",
    registeredAt: Date.now(),
    isActive: true,
    minVolumeUsd: 10_000,
    ...overrides,
  };
}

// ============ 1. Type Integration ============

describe.sequential("Type integration", () => {
  it("should have prediction_market in EVIDENCE_TYPE_STRENGTH", () => {
    expect(EVIDENCE_TYPE_STRENGTH.prediction_market).toBeDefined();
    expect(EVIDENCE_TYPE_STRENGTH.prediction_market).toBe(0.88);
  });

  it("should map prediction_market to oracle_feed canonical type", () => {
    expect(toCanonicalEvidenceType("prediction_market")).toBe("oracle_feed");
  });

  it("should have prediction_market in DEFAULT_EXPIRY_CONFIGS", () => {
    const config = DEFAULT_EXPIRY_CONFIGS.prediction_market;
    expect(config).toBeDefined();
    expect(config.expiryType).toBe("eternal");
    expect(config.freshnessScore).toBe(1.0);
    expect(config.isExpired).toBe(false);
  });

  it("should have prediction_market strength between chainlink_functions and chainlink_por", () => {
    expect(EVIDENCE_TYPE_STRENGTH.prediction_market).toBeGreaterThanOrEqual(
      EVIDENCE_TYPE_STRENGTH.chainlink_functions,
    );
    expect(EVIDENCE_TYPE_STRENGTH.prediction_market).toBeLessThanOrEqual(
      EVIDENCE_TYPE_STRENGTH.chainlink_por,
    );
  });

  it("should have platform reputation values for all platforms", () => {
    const platforms: PredictionMarketPlatform[] = [
      "polymarket",
      "kalshi",
      "metaculus",
      "custom",
    ];
    for (const platform of platforms) {
      expect(PLATFORM_REPUTATION[platform]).toBeDefined();
      expect(typeof PLATFORM_REPUTATION[platform]).toBe("number");
    }
  });
});

// ============ 2. hashMarketResolution ============

describe.sequential("hashMarketResolution", () => {
  it("should produce deterministic hashes", () => {
    const metadata = makeMarketMetadata();
    const hash1 = hashMarketResolution(metadata);
    const hash2 = hashMarketResolution(metadata);
    expect(hash1).toBe(hash2);
  });

  it("should produce different hashes for different markets", () => {
    const meta1 = makeMarketMetadata({ marketId: "market-001" });
    const meta2 = makeMarketMetadata({ marketId: "market-002" });
    expect(hashMarketResolution(meta1)).not.toBe(hashMarketResolution(meta2));
  });

  it("should have prediction_market prefix", () => {
    const hash = hashMarketResolution(makeMarketMetadata());
    expect(hash.startsWith("prediction_market:")).toBe(true);
  });
});

// ============ 3. validateMarketForEvidence ============

describe.sequential("validateMarketForEvidence", () => {
  it("should accept valid resolved market", () => {
    const result = validateMarketForEvidence(makeMarketMetadata());
    expect(result).toBeNull();
  });

  it("should reject unresolved (open) market", () => {
    const result = validateMarketForEvidence(
      makeMarketMetadata({ resolutionStatus: "open" }),
    );
    expect(result).toContain("not yet resolved");
  });

  it("should reject unresolved (closed) market", () => {
    const result = validateMarketForEvidence(
      makeMarketMetadata({ resolutionStatus: "closed" }),
    );
    expect(result).toContain("not yet resolved");
  });

  it("should reject voided market", () => {
    const result = validateMarketForEvidence(
      makeMarketMetadata({ resolutionStatus: "voided" }),
    );
    expect(result).toContain("voided");
  });

  it("should reject low-volume market", () => {
    const result = validateMarketForEvidence(
      makeMarketMetadata({ volumeUsd: 5_000 }),
    );
    expect(result).toContain("Insufficient volume");
  });

  it("should reject market with too few traders", () => {
    const result = validateMarketForEvidence(
      makeMarketMetadata({ traderCount: 20 }),
    );
    expect(result).toContain("Insufficient traders");
  });

  it("should accept disputed market (with penalties applied elsewhere)", () => {
    const result = validateMarketForEvidence(
      makeMarketMetadata({ resolutionStatus: "disputed" }),
    );
    expect(result).toBeNull();
  });
});

// ============ 4. calculatePredictionMarketConfidence ============

describe.sequential("calculatePredictionMarketConfidence", () => {
  it("should return base confidence for minimal valid market", () => {
    const meta = makeMarketMetadata({
      volumeUsd: 10_000,
      traderCount: 50,
      platform: "polymarket",
      marketType: "binary_outcome",
      disputeWindowElapsed: true,
    });
    const result = calculatePredictionMarketConfidence(meta);
    // Base 0.85 + volume bonus for $10k + platform 0.0 + no trader bonus (exactly 50)
    expect(result.confidence).toBeGreaterThanOrEqual(0.85);
    expect(result.confidence).toBeLessThanOrEqual(
      PREDICTION_MARKET_CONFIDENCE.BASE_MAX,
    );
  });

  it("should scale confidence with volume", () => {
    const lowVolume = calculatePredictionMarketConfidence(
      makeMarketMetadata({ volumeUsd: 50_000, traderCount: 50 }),
    );
    const highVolume = calculatePredictionMarketConfidence(
      makeMarketMetadata({ volumeUsd: 5_000_000, traderCount: 50 }),
    );
    expect(highVolume.confidence).toBeGreaterThan(lowVolume.confidence);
    expect(highVolume.volumeBonus).toBeGreaterThan(lowVolume.volumeBonus);
  });

  it("should scale confidence with trader count", () => {
    const fewTraders = calculatePredictionMarketConfidence(
      makeMarketMetadata({ traderCount: 100 }),
    );
    const manyTraders = calculatePredictionMarketConfidence(
      makeMarketMetadata({ traderCount: 1000 }),
    );
    expect(manyTraders.traderBonus).toBeGreaterThan(fewTraders.traderBonus);
  });

  it("should apply positive platform adjustment for Kalshi", () => {
    // Use low volume/traders so the cap doesn't mask the difference
    const base = { volumeUsd: 11_000, traderCount: 55 };
    const polymarket = calculatePredictionMarketConfidence(
      makeMarketMetadata({ platform: "polymarket", ...base }),
    );
    const kalshi = calculatePredictionMarketConfidence(
      makeMarketMetadata({ platform: "kalshi", ...base }),
    );
    expect(kalshi.platformAdjustment).toBe(0.02);
    expect(kalshi.confidence).toBeGreaterThan(polymarket.confidence);
  });

  it("should apply negative platform adjustment for custom", () => {
    const result = calculatePredictionMarketConfidence(
      makeMarketMetadata({ platform: "custom" }),
    );
    expect(result.platformAdjustment).toBe(-0.05);
  });

  it("should apply market type bonus for categorical", () => {
    // Use low volume/traders so the cap doesn't mask the difference
    const base = { volumeUsd: 11_000, traderCount: 55 };
    const binary = calculatePredictionMarketConfidence(
      makeMarketMetadata({ marketType: "binary_outcome", ...base }),
    );
    const categorical = calculatePredictionMarketConfidence(
      makeMarketMetadata({ marketType: "categorical", ...base }),
    );
    expect(categorical.marketTypeBonus).toBe(
      PREDICTION_MARKET_CONFIDENCE.CATEGORICAL_BONUS,
    );
    expect(categorical.confidence).toBeGreaterThan(binary.confidence);
  });

  it("should apply market type bonus for scalar", () => {
    const result = calculatePredictionMarketConfidence(
      makeMarketMetadata({ marketType: "scalar" }),
    );
    expect(result.marketTypeBonus).toBe(
      PREDICTION_MARKET_CONFIDENCE.SCALAR_BONUS,
    );
  });

  it("should apply dispute pending penalty", () => {
    const finalized = calculatePredictionMarketConfidence(
      makeMarketMetadata({ disputeWindowElapsed: true }),
    );
    const pending = calculatePredictionMarketConfidence(
      makeMarketMetadata({ disputeWindowElapsed: false }),
    );
    expect(pending.disputePenalty).toBe(
      PREDICTION_MARKET_CONFIDENCE.DISPUTE_PENDING_PENALTY,
    );
    expect(pending.confidence).toBeLessThan(finalized.confidence);
  });

  it("should apply disputed resolution penalty", () => {
    const result = calculatePredictionMarketConfidence(
      makeMarketMetadata({ resolutionStatus: "disputed" }),
    );
    expect(result.disputePenalty).toBe(
      PREDICTION_MARKET_CONFIDENCE.DISPUTED_RESOLUTION_PENALTY,
    );
  });

  it("should return zero confidence for voided market", () => {
    const result = calculatePredictionMarketConfidence(
      makeMarketMetadata({ resolutionStatus: "voided" }),
    );
    expect(result.confidence).toBe(0);
  });
});

// ============ 5. processPredictionMarketResolution ============

describe.sequential("processPredictionMarketResolution", () => {
  it("should throw for invalid market", () => {
    const reg = makeRegistration();
    const meta = makeMarketMetadata({ resolutionStatus: "open" });
    expect(() =>
      processPredictionMarketResolution(reg, meta, 100n, TEST_SUBMITTER),
    ).toThrow("Invalid market for evidence");
  });

  it("should throw for voided market", () => {
    const reg = makeRegistration();
    const meta = makeMarketMetadata({ resolutionStatus: "voided" });
    expect(() =>
      processPredictionMarketResolution(reg, meta, 100n, TEST_SUBMITTER),
    ).toThrow("voided");
  });

  it("should create evidence with correct type", () => {
    const reg = makeRegistration();
    const meta = makeMarketMetadata();
    const { evidence } = processPredictionMarketResolution(
      reg,
      meta,
      100n,
      TEST_SUBMITTER,
    );
    expect(evidence.type).toBe("prediction_market");
  });

  it("should create correct oracle metadata", () => {
    const reg = makeRegistration();
    const meta = makeMarketMetadata();
    const { oracleMetadata } = processPredictionMarketResolution(
      reg,
      meta,
      100n,
      TEST_SUBMITTER,
    );
    expect(oracleMetadata.provider).toBe("polymarket");
    expect(oracleMetadata.predictionMarketMetadata).toBeDefined();
    expect(oracleMetadata.predictionMarketMetadata!.marketId).toBe(
      "market-001",
    );
  });

  it("should set correct evidence fields", () => {
    const reg = makeRegistration();
    const meta = makeMarketMetadata();
    const { evidence } = processPredictionMarketResolution(
      reg,
      meta,
      100n,
      TEST_SUBMITTER,
    );
    expect(evidence.id).toBe("pm:polymarket:market-001");
    expect(evidence.originalSource).toBe(
      "https://polymarket.com/market/market-001",
    );
    expect(evidence.firstSubmitter).toBe(TEST_SUBMITTER);
    expect(evidence.intrinsicStrength).toBe(
      EVIDENCE_TYPE_STRENGTH.prediction_market,
    );
    expect(evidence.submittedAtBlock).toBe(100n);
    expect(evidence.contentHash).toContain("prediction_market:");
  });
});

// ============ 6. mapPredictionMarketToPoTConfidence ============

describe.sequential("mapPredictionMarketToPoTConfidence", () => {
  it("should calculate equivalent verifications from trader count", () => {
    const result = mapPredictionMarketToPoTConfidence(
      makeMarketMetadata({ traderCount: 500 }),
    );
    expect(result.equivalentVerifications).toBe(10); // 500 / 50
  });

  it("should scale volume score logarithmically", () => {
    const lowVol = mapPredictionMarketToPoTConfidence(
      makeMarketMetadata({ volumeUsd: 10_000 }),
    );
    const highVol = mapPredictionMarketToPoTConfidence(
      makeMarketMetadata({ volumeUsd: 1_000_000 }),
    );
    expect(highVol.volumeScore).toBeGreaterThan(lowVol.volumeScore);
    expect(highVol.volumeScore).toBeLessThanOrEqual(1.0);
  });

  it("should give full freshness for resolved markets", () => {
    const result = mapPredictionMarketToPoTConfidence(makeMarketMetadata());
    expect(result.freshnessScore).toBe(1.0);
  });

  it("should give zero freshness for voided markets", () => {
    const result = mapPredictionMarketToPoTConfidence(
      makeMarketMetadata({ resolutionStatus: "voided" }),
    );
    expect(result.freshnessScore).toBe(0);
    expect(result.adjustedConfidence).toBe(0);
  });
});

// ============ 7. PredictionMarketRegistry ============

describe.sequential("PredictionMarketRegistry", () => {
  let registry: PredictionMarketRegistry;

  beforeEach(() => {
    registry = new PredictionMarketRegistry();
  });

  it("should register and retrieve a market", () => {
    const reg = makeRegistration();
    registry.registerMarket(reg);
    const found = registry.getMarket("polymarket", "market-001");
    expect(found).toBeDefined();
    expect(found!.question).toBe(reg.question);
  });

  it("should return undefined for unknown market", () => {
    const found = registry.getMarket("polymarket", "nonexistent");
    expect(found).toBeUndefined();
  });

  it("should index markets by platform", () => {
    registry.registerMarket(
      makeRegistration({ platform: "polymarket", marketId: "pm-1" }),
    );
    registry.registerMarket(
      makeRegistration({ platform: "polymarket", marketId: "pm-2" }),
    );
    registry.registerMarket(
      makeRegistration({ platform: "kalshi", marketId: "k-1" }),
    );

    const polymarketMarkets = registry.getMarketsByPlatform("polymarket");
    expect(polymarketMarkets).toHaveLength(2);

    const kalshiMarkets = registry.getMarketsByPlatform("kalshi");
    expect(kalshiMarkets).toHaveLength(1);
  });

  it("should deactivate a market", () => {
    registry.registerMarket(makeRegistration());
    const result = registry.deactivateMarket("polymarket", "market-001");
    expect(result).toBe(true);

    const market = registry.getMarket("polymarket", "market-001");
    expect(market!.isActive).toBe(false);
  });

  it("should return false when deactivating nonexistent market", () => {
    const result = registry.deactivateMarket("polymarket", "nonexistent");
    expect(result).toBe(false);
  });

  it("should report correct stats", () => {
    registry.registerMarket(
      makeRegistration({ platform: "polymarket", marketId: "pm-1" }),
    );
    registry.registerMarket(
      makeRegistration({ platform: "kalshi", marketId: "k-1" }),
    );
    registry.registerMarket(
      makeRegistration({
        platform: "kalshi",
        marketId: "k-2",
        isActive: false,
      }),
    );

    // Manually deactivate k-2 since makeRegistration isActive override is in the registration object
    registry.deactivateMarket("kalshi", "k-2");

    const stats = registry.getStats();
    expect(stats.totalMarkets).toBe(3);
    expect(stats.activeMarkets).toBe(2);
    expect(stats.marketsByPlatform["polymarket"]).toBe(1);
    expect(stats.marketsByPlatform["kalshi"]).toBe(2);
  });
});

// ============ 8. isDisputeWindowElapsed ============

describe.sequential("isDisputeWindowElapsed", () => {
  it("should return false when current block is before dispute window end", () => {
    const meta = makeMarketMetadata({
      disputeWindowEnd: 50000n,
      resolutionStatus: "resolved",
    });
    expect(isDisputeWindowElapsed(meta, 40000n)).toBe(false);
  });

  it("should return true when current block is at dispute window end", () => {
    const meta = makeMarketMetadata({
      disputeWindowEnd: 50000n,
      resolutionStatus: "resolved",
    });
    expect(isDisputeWindowElapsed(meta, 50000n)).toBe(true);
  });

  it("should return true when current block is past dispute window end", () => {
    const meta = makeMarketMetadata({
      disputeWindowEnd: 50000n,
      resolutionStatus: "resolved",
    });
    expect(isDisputeWindowElapsed(meta, 60000n)).toBe(true);
  });

  it("should return false for non-resolved markets regardless of block", () => {
    const meta = makeMarketMetadata({
      disputeWindowEnd: 50000n,
      resolutionStatus: "open",
    });
    expect(isDisputeWindowElapsed(meta, 100000n)).toBe(false);
  });
});

// ============ 9. createMarketRegistration ============

describe.sequential("createMarketRegistration", () => {
  it("should create registration with provided values", () => {
    const reg = createMarketRegistration(
      "kalshi",
      "kalshi-001",
      "Will it rain tomorrow?",
      "binary_outcome",
      "https://kalshi.com/events/kalshi-001",
      1700000000000,
    );
    expect(reg.platform).toBe("kalshi");
    expect(reg.marketId).toBe("kalshi-001");
    expect(reg.isActive).toBe(true);
    expect(reg.registeredAt).toBe(1700000000000);
    expect(reg.minVolumeUsd).toBe(PREDICTION_MARKET_CONFIDENCE.MIN_VOLUME_USD);
  });

  it("should accept custom min volume", () => {
    const reg = createMarketRegistration(
      "polymarket",
      "pm-001",
      "Test",
      "binary_outcome",
      "https://polymarket.com/pm-001",
      undefined,
      50_000,
    );
    expect(reg.minVolumeUsd).toBe(50_000);
  });
});

// ============ 10. Platform dispute windows ============

describe.sequential("Platform dispute windows", () => {
  it("should have dispute windows for all platforms", () => {
    const platforms: PredictionMarketPlatform[] = [
      "polymarket",
      "kalshi",
      "metaculus",
      "custom",
    ];
    for (const platform of platforms) {
      expect(PLATFORM_DISPUTE_WINDOWS[platform]).toBeGreaterThan(0);
    }
  });

  it("should have Kalshi with shorter window than Polymarket (regulated)", () => {
    expect(PLATFORM_DISPUTE_WINDOWS.kalshi).toBeLessThan(
      PLATFORM_DISPUTE_WINDOWS.polymarket,
    );
  });
});
