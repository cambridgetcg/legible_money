/**
 * Prediction Market Oracle Integration
 *
 * Maps resolved prediction market outcomes to Legible Money's Proof of Truth confidence.
 * Follows the Chainlink integration pattern exactly.
 *
 * SUPPORTED PLATFORMS:
 *   - Polymarket: Crypto-native, UMA oracle (base reputation 0.0)
 *   - Kalshi: CFTC-regulated, event contracts (+0.02 bonus)
 *   - Metaculus: Community forecasting, no real stakes (-0.02)
 *   - Custom: Unknown platforms (-0.05)
 *
 * CONFIDENCE MAPPING:
 *   Prediction markets: Participants put real money on outcomes
 *   PoT: Maps volume + trader count to equivalent verification strength
 *   Freshness: Event-based (resolved outcomes are permanent truth signals)
 */

import type { DID } from "../../core/agents/identity/did.js";
import type {
  Evidence,
  OracleMetadata,
  PredictionMarketMetadata,
  PredictionMarketRegistration,
  PredictionMarketPlatform,
  PredictionMarketType,
} from "./types.js";
import { EVIDENCE_TYPE_STRENGTH } from "./types.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";

// ============ Constants ============

/**
 * Prediction market confidence adjustments
 */
export const PREDICTION_MARKET_CONFIDENCE = {
  /** Base minimum confidence for resolved markets */
  BASE_MIN: 0.85,

  /** Base maximum confidence (hard cap) */
  BASE_MAX: 0.91,

  /** Volume scaling: log10(volumeUsd / 1000) * this factor */
  VOLUME_SCALING_FACTOR: 0.02,

  /** Maximum volume bonus */
  MAX_VOLUME_BONUS: 0.06,

  /** Trader count bonus: floor((traders - 50) / 100) * this factor */
  TRADER_COUNT_FACTOR: 0.003,

  /** Maximum trader count bonus */
  MAX_TRADER_BONUS: 0.03,

  /** Minimum traders for qualification */
  MIN_TRADERS: 50,

  /** Minimum volume in USD for qualification */
  MIN_VOLUME_USD: 10_000,

  /** Penalty when resolution is in dispute window (not yet finalized) */
  DISPUTE_PENDING_PENALTY: 0.15,

  /** Penalty when resolution has been disputed */
  DISPUTED_RESOLUTION_PENALTY: 0.4,

  /** Voided market confidence (effectively zero) */
  VOIDED_CONFIDENCE: 0,

  /** Market type bonuses */
  CATEGORICAL_BONUS: 0.01,
  SCALAR_BONUS: 0.015,
} as const;

/**
 * Platform reputation adjustments (additive to base confidence)
 */
export const PLATFORM_REPUTATION: Record<PredictionMarketPlatform, number> = {
  polymarket: 0.0, // Baseline — crypto-native, UMA oracle
  kalshi: 0.02, // CFTC-regulated, higher trust
  metaculus: -0.02, // Community forecasting, no real stakes
  custom: -0.05, // Unknown platform penalty
};

/**
 * Platform-specific dispute window lengths (in blocks)
 * Block time: 2521ms → 34,272 blocks/day
 */
export const PLATFORM_DISPUTE_WINDOWS: Record<
  PredictionMarketPlatform,
  number
> = {
  polymarket: 13_709, // ~9.6 hours (UMA DVM dispute window)
  kalshi: 6_854, // ~4.8 hours (shorter, regulated)
  metaculus: 34_272, // ~1 day (community resolution)
  custom: 34_272, // ~1 day (conservative default)
};

// ============ Market Registration ============

/**
 * Create a prediction market registration
 */
export function createMarketRegistration(
  platform: PredictionMarketPlatform,
  marketId: string,
  question: string,
  marketType: PredictionMarketType,
  marketUrl: string,
  currentTime?: number,
  minVolumeUsd?: number,
): PredictionMarketRegistration {
  const now = currentTime ?? Date.now();

  return {
    platform,
    marketId,
    question,
    marketType,
    marketUrl,
    registeredAt: now,
    isActive: true,
    minVolumeUsd: minVolumeUsd ?? PREDICTION_MARKET_CONFIDENCE.MIN_VOLUME_USD,
  };
}

// ============ Content Addressing ============

/**
 * Hash market resolution data for content addressing.
 * Parallels hashChainlinkRound().
 */
export function hashMarketResolution(
  metadata: PredictionMarketMetadata,
): string {
  const data = [
    metadata.platform,
    metadata.marketId,
    metadata.resolutionStatus,
    metadata.resolvedOutcome ?? "null",
    metadata.resolvedAt?.toString() ?? "0",
    metadata.volumeUsd.toString(),
  ].join(":");
  return `prediction_market:${sha512_256Hex(data)}`;
}

// ============ Validation ============

/**
 * Validate that a market is suitable for use as evidence.
 * Returns null if valid, or an error string if invalid.
 */
export function validateMarketForEvidence(
  metadata: PredictionMarketMetadata,
): string | null {
  if (metadata.resolutionStatus === "voided") {
    return "Market has been voided";
  }

  if (
    metadata.resolutionStatus !== "resolved" &&
    metadata.resolutionStatus !== "disputed"
  ) {
    return `Market not yet resolved (status: ${metadata.resolutionStatus})`;
  }

  if (metadata.volumeUsd < PREDICTION_MARKET_CONFIDENCE.MIN_VOLUME_USD) {
    return `Insufficient volume: $${metadata.volumeUsd} < $${PREDICTION_MARKET_CONFIDENCE.MIN_VOLUME_USD} minimum`;
  }

  if (metadata.traderCount < PREDICTION_MARKET_CONFIDENCE.MIN_TRADERS) {
    return `Insufficient traders: ${metadata.traderCount} < ${PREDICTION_MARKET_CONFIDENCE.MIN_TRADERS} minimum`;
  }

  return null;
}

// ============ Evidence Processing ============

/**
 * Process a resolved prediction market into Evidence + OracleMetadata.
 * Parallels processChainlinkPriceUpdate().
 */
export function processPredictionMarketResolution(
  registration: PredictionMarketRegistration,
  metadata: PredictionMarketMetadata,
  currentBlock: bigint,
  submitter: DID,
): {
  evidence: Partial<Evidence>;
  oracleMetadata: OracleMetadata;
} {
  const validationError = validateMarketForEvidence(metadata);
  if (validationError) {
    throw new Error(`Invalid market for evidence: ${validationError}`);
  }

  const contentHash = hashMarketResolution(metadata);

  const oracleMetadata: OracleMetadata = {
    provider: metadata.platform as OracleMetadata["provider"],
    feedAddress: `${metadata.platform}:${metadata.marketId}`,
    chainId: 0, // Not chain-specific
    feedName: metadata.question,
    decimals: 6, // Probability precision
    roundId: 0n, // Not applicable
    updatedAt: metadata.resolvedAt ?? Date.now(),
    answeredInRound: 0n,
    predictionMarketMetadata: metadata,
  };

  const evidence: Partial<Evidence> = {
    id: `pm:${metadata.platform}:${metadata.marketId}`,
    type: "prediction_market",
    originalSource: registration.marketUrl,
    contentHash,
    submittedAt: metadata.resolvedAt ?? Date.now(),
    submittedAtBlock: currentBlock,
    firstSubmitter: submitter,
    confirmedBy: [],
    status: "active",
    direction: "supporting",
    intrinsicStrength: EVIDENCE_TYPE_STRENGTH.prediction_market,
    oracleMetadata,
    relatedFactIds: [],
  };

  return { evidence, oracleMetadata };
}

// ============ Confidence Calculation ============

/**
 * Calculate prediction market confidence with all adjustments.
 * Parallels calculateChainlinkConfidence().
 *
 * Formula:
 *   base = 0.85
 *   + volume bonus: min(0.06, log10(volumeUsd / 1000) * 0.02)
 *   + trader bonus: min(0.03, floor((traders - 50) / 100) * 0.003)
 *   + platform reputation: -0.05 to +0.02
 *   + market type bonus: categorical +0.01, scalar +0.015
 *   - dispute pending: -0.15
 *   - disputed resolution: -0.40
 *   - voided: confidence = 0
 *   = clamped to [0, BASE_MAX]
 */
export function calculatePredictionMarketConfidence(
  metadata: PredictionMarketMetadata,
): {
  confidence: number;
  volumeBonus: number;
  traderBonus: number;
  platformAdjustment: number;
  marketTypeBonus: number;
  disputePenalty: number;
  breakdown: string;
} {
  // Voided markets get zero confidence
  if (metadata.resolutionStatus === "voided") {
    return {
      confidence: PREDICTION_MARKET_CONFIDENCE.VOIDED_CONFIDENCE,
      volumeBonus: 0,
      traderBonus: 0,
      platformAdjustment: 0,
      marketTypeBonus: 0,
      disputePenalty: 0,
      breakdown: "Voided market: 0%",
    };
  }

  let confidence: number = PREDICTION_MARKET_CONFIDENCE.BASE_MIN;
  const breakdownParts: string[] = [`Base: ${(confidence * 100).toFixed(1)}%`];

  // 1. Volume bonus: log-scaled
  let volumeBonus = 0;
  if (metadata.volumeUsd > 1000) {
    volumeBonus = Math.min(
      PREDICTION_MARKET_CONFIDENCE.MAX_VOLUME_BONUS,
      Math.log10(metadata.volumeUsd / 1000) *
        PREDICTION_MARKET_CONFIDENCE.VOLUME_SCALING_FACTOR,
    );
    confidence += volumeBonus;
    breakdownParts.push(`Volume: +${(volumeBonus * 100).toFixed(2)}%`);
  }

  // 2. Trader count bonus
  let traderBonus = 0;
  if (metadata.traderCount > PREDICTION_MARKET_CONFIDENCE.MIN_TRADERS) {
    const extraTraders =
      metadata.traderCount - PREDICTION_MARKET_CONFIDENCE.MIN_TRADERS;
    traderBonus = Math.min(
      PREDICTION_MARKET_CONFIDENCE.MAX_TRADER_BONUS,
      Math.floor(extraTraders / 100) *
        PREDICTION_MARKET_CONFIDENCE.TRADER_COUNT_FACTOR,
    );
    confidence += traderBonus;
    if (traderBonus > 0) {
      breakdownParts.push(`Traders: +${(traderBonus * 100).toFixed(2)}%`);
    }
  }

  // 3. Platform reputation
  const platformAdjustment = PLATFORM_REPUTATION[metadata.platform];
  confidence += platformAdjustment;
  if (platformAdjustment !== 0) {
    const sign = platformAdjustment > 0 ? "+" : "";
    breakdownParts.push(
      `Platform (${metadata.platform}): ${sign}${(platformAdjustment * 100).toFixed(1)}%`,
    );
  }

  // 4. Market type bonus
  let marketTypeBonus = 0;
  if (metadata.marketType === "categorical") {
    marketTypeBonus = PREDICTION_MARKET_CONFIDENCE.CATEGORICAL_BONUS;
  } else if (metadata.marketType === "scalar") {
    marketTypeBonus = PREDICTION_MARKET_CONFIDENCE.SCALAR_BONUS;
  }
  if (marketTypeBonus > 0) {
    confidence += marketTypeBonus;
    breakdownParts.push(
      `Market type (${metadata.marketType}): +${(marketTypeBonus * 100).toFixed(1)}%`,
    );
  }

  // 5. Dispute penalties
  let disputePenalty = 0;
  if (
    metadata.resolutionStatus === "resolved" &&
    !metadata.disputeWindowElapsed
  ) {
    disputePenalty = PREDICTION_MARKET_CONFIDENCE.DISPUTE_PENDING_PENALTY;
    confidence -= disputePenalty;
    breakdownParts.push(
      `Dispute window pending: -${(disputePenalty * 100).toFixed(1)}%`,
    );
  } else if (metadata.resolutionStatus === "disputed") {
    disputePenalty = PREDICTION_MARKET_CONFIDENCE.DISPUTED_RESOLUTION_PENALTY;
    confidence -= disputePenalty;
    breakdownParts.push(
      `Disputed resolution: -${(disputePenalty * 100).toFixed(1)}%`,
    );
  }

  // Clamp to [0, BASE_MAX]
  confidence = Math.max(
    0,
    Math.min(PREDICTION_MARKET_CONFIDENCE.BASE_MAX, confidence),
  );

  breakdownParts.push(`Final: ${(confidence * 100).toFixed(1)}%`);

  return {
    confidence,
    volumeBonus,
    traderBonus,
    platformAdjustment,
    marketTypeBonus,
    disputePenalty,
    breakdown: breakdownParts.join(" → "),
  };
}

// ============ PoT Confidence Mapping ============

/**
 * Map prediction market consensus to PoT confidence equivalence.
 * Parallels mapChainlinkToPoTConfidence().
 */
export function mapPredictionMarketToPoTConfidence(
  metadata: PredictionMarketMetadata,
): {
  equivalentVerifications: number;
  volumeScore: number;
  freshnessScore: number;
  baseConfidence: number;
  adjustedConfidence: number;
  reasoning: string;
} {
  // Volume-based "verification count": log10(volume) equivalent verifiers
  const volumeScore =
    metadata.volumeUsd > 0
      ? Math.min(1.0, Math.log10(metadata.volumeUsd) / 7) // $10M = ~1.0
      : 0;

  // Trader count as equivalent verifications (1 per 50 traders)
  const equivalentVerifications = Math.floor(metadata.traderCount / 50);

  // Resolved outcomes are permanently fresh
  const freshnessScore = metadata.resolutionStatus === "voided" ? 0 : 1.0;

  const baseConfidence = EVIDENCE_TYPE_STRENGTH.prediction_market;

  // Adjust by volume score and freshness
  const adjustedConfidence = baseConfidence * volumeScore * freshnessScore;

  const reasoning = [
    `${metadata.traderCount} traders`,
    `$${metadata.volumeUsd.toLocaleString()} volume`,
    `${equivalentVerifications} equiv. verifications`,
    `${(volumeScore * 100).toFixed(1)}% volume score`,
    `${(freshnessScore * 100).toFixed(0)}% fresh`,
  ].join(", ");

  return {
    equivalentVerifications,
    volumeScore,
    freshnessScore,
    baseConfidence,
    adjustedConfidence,
    reasoning,
  };
}

// ============ Dispute Window ============

/**
 * Check if a market's dispute window has elapsed.
 * Parallels checkSequencerStatus() for Chainlink L2.
 */
export function isDisputeWindowElapsed(
  metadata: PredictionMarketMetadata,
  currentBlock: bigint,
): boolean {
  if (metadata.resolutionStatus !== "resolved") {
    return false;
  }
  return currentBlock >= metadata.disputeWindowEnd;
}

// ============ Market Registry ============

/**
 * In-memory prediction market registry.
 * Parallels ChainlinkFeedRegistry.
 */
export class PredictionMarketRegistry {
  private markets: Map<string, PredictionMarketRegistration> = new Map();
  private marketsByPlatform: Map<PredictionMarketPlatform, string[]> =
    new Map();

  /**
   * Register a new market
   */
  registerMarket(registration: PredictionMarketRegistration): void {
    const key = `${registration.platform}:${registration.marketId}`;
    this.markets.set(key, registration);

    const platformMarkets =
      this.marketsByPlatform.get(registration.platform) || [];
    if (!platformMarkets.includes(key)) {
      platformMarkets.push(key);
      this.marketsByPlatform.set(registration.platform, platformMarkets);
    }
  }

  /**
   * Get market by platform and ID
   */
  getMarket(
    platform: PredictionMarketPlatform,
    marketId: string,
  ): PredictionMarketRegistration | undefined {
    return this.markets.get(`${platform}:${marketId}`);
  }

  /**
   * Get all markets for a platform
   */
  getMarketsByPlatform(
    platform: PredictionMarketPlatform,
  ): PredictionMarketRegistration[] {
    const keys = this.marketsByPlatform.get(platform) || [];
    return keys.map((key) => this.markets.get(key)!).filter(Boolean);
  }

  /**
   * Deactivate a market
   */
  deactivateMarket(
    platform: PredictionMarketPlatform,
    marketId: string,
  ): boolean {
    const market = this.getMarket(platform, marketId);
    if (!market) return false;
    market.isActive = false;
    return true;
  }

  /**
   * Get all active markets across all platforms
   */
  getActiveMarkets(): PredictionMarketRegistration[] {
    return Array.from(this.markets.values()).filter((m) => m.isActive);
  }

  /**
   * Get registry statistics
   */
  getStats(): {
    totalMarkets: number;
    activeMarkets: number;
    marketsByPlatform: Record<string, number>;
  } {
    const activeMarkets = this.getActiveMarkets().length;

    const marketsByPlatform: Record<string, number> = {};
    for (const [platform, keys] of this.marketsByPlatform) {
      marketsByPlatform[platform] = keys.length;
    }

    return {
      totalMarkets: this.markets.size,
      activeMarkets,
      marketsByPlatform,
    };
  }
}
