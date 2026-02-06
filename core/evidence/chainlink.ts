/**
 * Chainlink Oracle Integration
 *
 * Maps Chainlink's verification model to Legible Money's Proof of Truth confidence.
 *
 * SUPPORTED FEED TYPES:
 *   - Price Feeds: Aggregated asset prices (0.92 strength)
 *   - Proof of Reserve: Reserve verification (0.91 strength)
 *   - Functions: Custom computation (0.88 strength)
 *   - CCIP: Cross-chain data (0.90 strength)
 *
 * CONFIDENCE MAPPING:
 *   Chainlink: Multiple oracle nodes → aggregated answer
 *   PoT: Maps oracle count to equivalent verifications
 *   Freshness: Heartbeat-based with staleness penalties
 */

import type { DID } from "../../core/agents/identity/did.js";
import type {
  Evidence,
  EvidenceType,
  OracleMetadata,
  ChainlinkMetadata,
  ChainlinkRoundData,
  ChainlinkFeedRegistration,
  ChainlinkFeedType,
  HeartbeatConfig,
  L2SequencerStatus,
  ProofOfReserveMetadata,
} from "./types.js";
import { EVIDENCE_TYPE_STRENGTH } from "./types.js";
import { calculateHeartbeatFreshness, isStale } from "./expiry.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";

// ============ Constants ============

/**
 * Chainlink-specific confidence adjustments
 */
export const CHAINLINK_CONFIDENCE = {
  /** Base confidence for price feeds */
  PRICE_FEED_BASE: 0.92,

  /** Base confidence for Proof of Reserve */
  POR_BASE: 0.91,

  /** Base confidence for CCIP */
  CCIP_BASE: 0.9,

  /** Base confidence for Functions */
  FUNCTIONS_BASE: 0.88,

  /** Penalty for stale data (per % of staleness) */
  STALENESS_PENALTY_RATE: 0.005,

  /** Maximum staleness penalty */
  MAX_STALENESS_PENALTY: 0.5,

  /** Penalty when L2 sequencer is down */
  SEQUENCER_DOWN_PENALTY: 0.7,

  /** Penalty during L2 sequencer grace period */
  SEQUENCER_GRACE_PENALTY: 0.3,

  /** Bonus for high oracle count (per oracle above minimum) */
  ORACLE_COUNT_BONUS: 0.005,

  /** Maximum oracle count bonus */
  MAX_ORACLE_BONUS: 0.05,

  /** Minimum required oracles for full confidence */
  MIN_ORACLE_COUNT: 7,
} as const;

/**
 * Default heartbeat configurations by feed type
 */
export const DEFAULT_CHAINLINK_HEARTBEATS: Record<
  ChainlinkFeedType,
  HeartbeatConfig
> = {
  price_feed: {
    expectedInterval: 3600, // ~2.5 hours at 2521ms blocks
    maxStaleness: 7200, // ~5 hours
    deviationThreshold: 1.0, // 1%
    lastHeartbeat: 0,
    lastHeartbeatBlock: 0n,
    isStale: false,
  },
  por: {
    expectedInterval: 86400, // ~60 hours
    maxStaleness: 172800, // ~120 hours
    deviationThreshold: 5.0, // 5%
    lastHeartbeat: 0,
    lastHeartbeatBlock: 0n,
    isStale: false,
  },
  functions: {
    expectedInterval: 3600,
    maxStaleness: 14400, // ~10 hours
    deviationThreshold: 0, // N/A
    lastHeartbeat: 0,
    lastHeartbeatBlock: 0n,
    isStale: false,
  },
  ccip: {
    expectedInterval: 0, // Event-based
    maxStaleness: 0,
    deviationThreshold: 0,
    lastHeartbeat: 0,
    lastHeartbeatBlock: 0n,
    isStale: false,
  },
};

// ============ Feed Registration ============

/**
 * Create a Chainlink feed registration
 */
export function createFeedRegistration(
  feedAddress: string,
  chainId: number,
  feedType: ChainlinkFeedType,
  feedName: string,
  decimals: number,
  aggregatorAddress: string,
  customHeartbeat?: Partial<HeartbeatConfig>,
  currentTime?: number,
): ChainlinkFeedRegistration {
  // P0-E4 FIX: Accept deterministic timestamp for consensus path
  const now = currentTime ?? Date.now();
  const defaultHeartbeat = DEFAULT_CHAINLINK_HEARTBEATS[feedType];

  return {
    feedAddress,
    chainId,
    feedType,
    decimals,
    description: feedName,
    version: 1,
    aggregatorAddress,
    heartbeatConfig: {
      ...defaultHeartbeat,
      ...customHeartbeat,
    },
    registeredAt: now,
    isActive: true,
  };
}

/**
 * Get evidence type for Chainlink feed type
 */
export function feedTypeToEvidenceType(
  feedType: ChainlinkFeedType,
): EvidenceType {
  switch (feedType) {
    case "price_feed":
      return "chainlink_price_feed";
    case "por":
      return "chainlink_por";
    case "functions":
      return "chainlink_functions";
    case "ccip":
      return "chainlink_ccip";
    default:
      return "oracle";
  }
}

// ============ Round Data Processing ============

/**
 * Hash Chainlink round data for content addressing
 */
export function hashChainlinkRound(roundData: ChainlinkRoundData): string {
  const data = `${roundData.roundId}:${roundData.answer}:${roundData.updatedAt}:${roundData.answeredInRound}`;
  return `chainlink:${sha512_256Hex(data)}`;
}

/**
 * Process Chainlink price feed update
 */
export function processChainlinkPriceUpdate(
  registration: ChainlinkFeedRegistration,
  roundData: ChainlinkRoundData,
  currentBlock: bigint,
  submitter: DID,
): {
  evidence: Partial<Evidence>;
  oracleMetadata: OracleMetadata;
  chainlinkMetadata: Partial<ChainlinkMetadata>;
} {
  const contentHash = hashChainlinkRound(roundData);

  const oracleMetadata: OracleMetadata = {
    provider: "chainlink",
    feedAddress: registration.feedAddress,
    chainId: registration.chainId,
    feedName: registration.description,
    decimals: registration.decimals,
    roundId: roundData.roundId,
    updatedAt: roundData.updatedAt,
    answeredInRound: roundData.answeredInRound,
    chainlinkMetadata: {
      aggregatorAddress: registration.aggregatorAddress,
      oracleCount: 0, // Will be updated from aggregator
      minSubmissions: 0,
      phaseId: 0,
    },
  };

  const evidence: Partial<Evidence> = {
    id: `${registration.feedAddress}:${roundData.roundId}`,
    type: feedTypeToEvidenceType(registration.feedType),
    originalSource: registration.feedAddress,
    contentHash,
    submittedAt: roundData.updatedAt,
    submittedAtBlock: currentBlock,
    firstSubmitter: submitter,
    confirmedBy: [],
    status: "active",
    direction: "supporting",
    intrinsicStrength:
      EVIDENCE_TYPE_STRENGTH[feedTypeToEvidenceType(registration.feedType)],
    oracleMetadata,
    relatedFactIds: [],
  };

  return {
    evidence,
    oracleMetadata,
    chainlinkMetadata: oracleMetadata.chainlinkMetadata!,
  };
}

/**
 * Process Proof of Reserve update
 */
export function processChainlinkPoRUpdate(
  registration: ChainlinkFeedRegistration,
  roundData: ChainlinkRoundData,
  porMetadata: ProofOfReserveMetadata,
  currentBlock: bigint,
  submitter: DID,
): {
  evidence: Partial<Evidence>;
  oracleMetadata: OracleMetadata;
} {
  const contentHash = hashChainlinkRound(roundData);

  const oracleMetadata: OracleMetadata = {
    provider: "chainlink",
    feedAddress: registration.feedAddress,
    chainId: registration.chainId,
    feedName: registration.description,
    decimals: registration.decimals,
    roundId: roundData.roundId,
    updatedAt: roundData.updatedAt,
    answeredInRound: roundData.answeredInRound,
    chainlinkMetadata: {
      aggregatorAddress: registration.aggregatorAddress,
      oracleCount: 0,
      minSubmissions: 0,
      phaseId: 0,
      porMetadata,
    },
  };

  const evidence: Partial<Evidence> = {
    id: `por:${registration.feedAddress}:${roundData.roundId}`,
    type: "chainlink_por",
    originalSource: registration.feedAddress,
    contentHash,
    submittedAt: roundData.updatedAt,
    submittedAtBlock: currentBlock,
    firstSubmitter: submitter,
    confirmedBy: [],
    status: "active",
    direction: "supporting",
    intrinsicStrength: EVIDENCE_TYPE_STRENGTH.chainlink_por,
    oracleMetadata,
    relatedFactIds: [],
  };

  return { evidence, oracleMetadata };
}

// ============ Confidence Calculation ============

/**
 * Calculate Chainlink confidence with all adjustments
 */
export function calculateChainlinkConfidence(
  oracleMetadata: OracleMetadata,
  heartbeat: HeartbeatConfig,
  currentBlock: bigint,
): {
  confidence: number;
  isStale: boolean;
  stalePenalty: number;
  sequencerPenalty: number;
  oracleBonus: number;
  breakdown: string;
} {
  // Start with base confidence for the feed type
  let confidence =
    EVIDENCE_TYPE_STRENGTH[
      oracleMetadata.provider === "chainlink"
        ? "chainlink_price_feed"
        : "oracle"
    ];

  let stalePenalty = 0;
  let sequencerPenalty = 0;
  let oracleBonus = 0;
  const breakdownParts: string[] = [`Base: ${(confidence * 100).toFixed(1)}%`];

  // 1. Check staleness
  const stale = isStale(heartbeat, currentBlock);
  if (stale) {
    const freshness = calculateHeartbeatFreshness(heartbeat, currentBlock);

    if (freshness === 0) {
      // Completely stale - maximum penalty
      stalePenalty = CHAINLINK_CONFIDENCE.MAX_STALENESS_PENALTY;
    } else {
      // Partial staleness penalty
      stalePenalty =
        (1 - freshness) * CHAINLINK_CONFIDENCE.MAX_STALENESS_PENALTY;
    }

    confidence *= 1 - stalePenalty;
    breakdownParts.push(`Staleness: -${(stalePenalty * 100).toFixed(1)}%`);
  }

  // 2. Check L2 sequencer (if applicable)
  if (oracleMetadata.chainlinkMetadata?.sequencerStatus) {
    const seqStatus = oracleMetadata.chainlinkMetadata.sequencerStatus;

    if (!seqStatus.isSequencerUp) {
      sequencerPenalty = CHAINLINK_CONFIDENCE.SEQUENCER_DOWN_PENALTY;
      confidence *= 1 - sequencerPenalty;
      breakdownParts.push(
        `Sequencer down: -${(sequencerPenalty * 100).toFixed(1)}%`,
      );
    } else if (seqStatus.inGracePeriod) {
      sequencerPenalty = CHAINLINK_CONFIDENCE.SEQUENCER_GRACE_PENALTY;
      confidence *= 1 - sequencerPenalty;
      breakdownParts.push(
        `Sequencer grace: -${(sequencerPenalty * 100).toFixed(1)}%`,
      );
    }
  }

  // 3. Oracle count bonus
  if (oracleMetadata.chainlinkMetadata) {
    const oracleCount = oracleMetadata.chainlinkMetadata.oracleCount;

    if (oracleCount > CHAINLINK_CONFIDENCE.MIN_ORACLE_COUNT) {
      const extraOracles = oracleCount - CHAINLINK_CONFIDENCE.MIN_ORACLE_COUNT;
      oracleBonus = Math.min(
        extraOracles * CHAINLINK_CONFIDENCE.ORACLE_COUNT_BONUS,
        CHAINLINK_CONFIDENCE.MAX_ORACLE_BONUS,
      );
      confidence = Math.min(1.0, confidence * (1 + oracleBonus));
      breakdownParts.push(`Oracle bonus: +${(oracleBonus * 100).toFixed(1)}%`);
    }
  }

  breakdownParts.push(`Final: ${(confidence * 100).toFixed(1)}%`);

  return {
    confidence,
    isStale: stale,
    stalePenalty,
    sequencerPenalty,
    oracleBonus,
    breakdown: breakdownParts.join(" → "),
  };
}

/**
 * Map Chainlink verification model to PoT confidence
 *
 * Chainlink: Multiple independent oracles submit observations, aggregator takes median
 * PoT: Maps oracle count to "equivalent verifications" for confidence calculation
 */
export function mapChainlinkToPoTConfidence(
  chainlinkMetadata: ChainlinkMetadata,
  heartbeat: HeartbeatConfig,
  currentBlock: bigint,
): {
  equivalentVerifications: number;
  quorumRatio: number;
  freshnessScore: number;
  baseConfidence: number;
  adjustedConfidence: number;
  reasoning: string;
} {
  // Oracle count as "verification count"
  const oracleCount = chainlinkMetadata.oracleCount;
  const minSubmissions = chainlinkMetadata.minSubmissions;

  // Map to equivalent PoT verifications (70% of oracles = verified)
  const equivalentVerifications = Math.floor(oracleCount * 0.7);

  // Quorum ratio
  const quorumRatio = oracleCount > 0 ? minSubmissions / oracleCount : 0;

  // Freshness from heartbeat
  const freshnessScore = calculateHeartbeatFreshness(heartbeat, currentBlock);

  // Base confidence from type
  const baseConfidence = EVIDENCE_TYPE_STRENGTH.chainlink_price_feed;

  // Apply freshness adjustment
  const adjustedConfidence = baseConfidence * freshnessScore;

  const reasoning = [
    `${oracleCount} oracles`,
    `${minSubmissions} min submissions`,
    `${equivalentVerifications} equiv. verifications`,
    `${(quorumRatio * 100).toFixed(0)}% quorum`,
    `${(freshnessScore * 100).toFixed(1)}% fresh`,
  ].join(", ");

  return {
    equivalentVerifications,
    quorumRatio,
    freshnessScore,
    baseConfidence,
    adjustedConfidence,
    reasoning,
  };
}

// ============ L2 Sequencer Handling ============

/**
 * Check L2 sequencer uptime status
 */
export function checkSequencerStatus(
  status: L2SequencerStatus,
  currentBlock: bigint,
  gracePeriodBlocks: number,
): {
  isHealthy: boolean;
  inGracePeriod: boolean;
  canUseFeeds: boolean;
  reason: string;
} {
  if (!status.isSequencerUp) {
    return {
      isHealthy: false,
      inGracePeriod: false,
      canUseFeeds: false,
      reason: "Sequencer is down",
    };
  }

  // Check grace period
  const blocksSinceRecovery = Number(currentBlock) - status.startedAt;
  const inGracePeriod = blocksSinceRecovery < gracePeriodBlocks;

  if (inGracePeriod) {
    return {
      isHealthy: true,
      inGracePeriod: true,
      canUseFeeds: true, // Can use but with penalty
      reason: `Sequencer recovered ${blocksSinceRecovery} blocks ago, in grace period`,
    };
  }

  return {
    isHealthy: true,
    inGracePeriod: false,
    canUseFeeds: true,
    reason: "Sequencer healthy",
  };
}

/**
 * Update sequencer status in oracle metadata
 */
export function updateSequencerStatus(
  oracleMetadata: OracleMetadata,
  newStatus: L2SequencerStatus,
): OracleMetadata {
  if (!oracleMetadata.chainlinkMetadata) {
    return oracleMetadata;
  }

  return {
    ...oracleMetadata,
    chainlinkMetadata: {
      ...oracleMetadata.chainlinkMetadata,
      sequencerStatus: newStatus,
    },
  };
}

// ============ Feed Registry ============

/**
 * In-memory Chainlink feed registry
 */
export class ChainlinkFeedRegistry {
  private feeds: Map<string, ChainlinkFeedRegistration> = new Map();
  private feedsByChain: Map<number, string[]> = new Map();
  private feedsByType: Map<ChainlinkFeedType, string[]> = new Map();

  /**
   * Register a new feed
   */
  registerFeed(registration: ChainlinkFeedRegistration): void {
    const key = `${registration.chainId}:${registration.feedAddress}`;

    this.feeds.set(key, registration);

    // Index by chain
    const chainFeeds = this.feedsByChain.get(registration.chainId) || [];
    chainFeeds.push(key);
    this.feedsByChain.set(registration.chainId, chainFeeds);

    // Index by type
    const typeFeeds = this.feedsByType.get(registration.feedType) || [];
    typeFeeds.push(key);
    this.feedsByType.set(registration.feedType, typeFeeds);
  }

  /**
   * Get feed by address and chain
   */
  getFeed(
    chainId: number,
    feedAddress: string,
  ): ChainlinkFeedRegistration | undefined {
    return this.feeds.get(`${chainId}:${feedAddress}`);
  }

  /**
   * Get all feeds for a chain
   */
  getFeedsByChain(chainId: number): ChainlinkFeedRegistration[] {
    const keys = this.feedsByChain.get(chainId) || [];
    return keys.map((key) => this.feeds.get(key)!).filter(Boolean);
  }

  /**
   * Get all feeds of a type
   */
  getFeedsByType(feedType: ChainlinkFeedType): ChainlinkFeedRegistration[] {
    const keys = this.feedsByType.get(feedType) || [];
    return keys.map((key) => this.feeds.get(key)!).filter(Boolean);
  }

  /**
   * Deactivate a feed
   */
  deactivateFeed(chainId: number, feedAddress: string): boolean {
    const feed = this.getFeed(chainId, feedAddress);
    if (!feed) return false;

    feed.isActive = false;
    return true;
  }

  /**
   * Update feed heartbeat
   */
  updateHeartbeat(
    chainId: number,
    feedAddress: string,
    heartbeatBlock: bigint,
    heartbeatTime: number,
  ): boolean {
    const feed = this.getFeed(chainId, feedAddress);
    if (!feed) return false;

    feed.heartbeatConfig.lastHeartbeat = heartbeatTime;
    feed.heartbeatConfig.lastHeartbeatBlock = heartbeatBlock;
    feed.heartbeatConfig.isStale = false;

    return true;
  }

  /**
   * Get stale feeds
   */
  getStaleFeeds(currentBlock: bigint): ChainlinkFeedRegistration[] {
    return Array.from(this.feeds.values()).filter((feed) => {
      if (!feed.isActive) return false;
      return isStale(feed.heartbeatConfig, currentBlock);
    });
  }

  /**
   * Get registry statistics
   */
  getStats(): {
    totalFeeds: number;
    activeFeeds: number;
    feedsByChain: Record<number, number>;
    feedsByType: Record<string, number>;
  } {
    const activeFeeds = Array.from(this.feeds.values()).filter(
      (f) => f.isActive,
    ).length;

    const feedsByChain: Record<number, number> = {};
    for (const [chainId, feeds] of this.feedsByChain) {
      feedsByChain[chainId] = feeds.length;
    }

    const feedsByType: Record<string, number> = {};
    for (const [type, feeds] of this.feedsByType) {
      feedsByType[type] = feeds.length;
    }

    return {
      totalFeeds: this.feeds.size,
      activeFeeds,
      feedsByChain,
      feedsByType,
    };
  }
}
