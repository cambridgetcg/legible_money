/**
 * Evidence Management System
 *
 * Comprehensive evidence lifecycle management including:
 * - Automatic archiving (IPFS, Arweave, Wayback Machine)
 * - Evidence deduplication (exact, near-duplicate, related)
 * - Evidence versioning (corrections, updates, retractions)
 * - Evidence expiry (eternal, heartbeat, decay-based)
 * - Chainlink oracle integration (price feeds, PoR, functions, CCIP)
 */

// Core types
export * from "./types.js";

// Archiving system
export {
  ARCHIVE_CONFIDENCE,
  ARCHIVE_RETRY,
  createArchiveRequest,
  calculateRetryDelay,
  shouldRetryArchive,
  updateArchiveRequestStatus,
  createArchiveLocation,
  verifyArchive,
  addArchiveToEvidence,
  calculateArchiveBonus,
  applyArchiveBonus,
  needsArchiving,
  getMissingProviders,
  ArchiveQueue,
  verifyArchiveAccessibility,
  verifyEvidenceArchives,
  applyVerificationResults,
  IPFSArchiveProvider,
  ArweaveArchiveProvider,
  WaybackArchiveProvider,
} from "./archiving.js";
export type {
  ArchiveVerificationResult,
  ArchiveProviderInterface,
} from "./archiving.js";

// Deduplication system
export {
  generateContentHash,
  generatePerceptualHash,
  normalizeUrl,
  extractContentFeatures,
  generateFingerprint,
  calculatePerceptualSimilarity,
  calculateFeatureSimilarity,
  detectDuplicate,
  checkEvidenceDuplicate,
  calculateConfirmationCredit,
  DeduplicationIndex,
} from "./deduplication.js";

// Expiry system
export {
  calculateFreshness,
  calculateHeartbeatFreshness,
  calculateBlockBasedFreshness,
  calculateDecayFreshness,
  isExpired,
  isStale,
  blocksUntilExpiry,
  updateExpiryState,
  updateHeartbeat,
  markExpiredByEvent,
  processExpirations,
  findStaleHeartbeats,
  findExpiringEvidence,
  createDefaultExpiry,
  applyFreshnessToConfidence,
  calculateEffectiveReliability,
} from "./expiry.js";
export type { ExpiredEvidenceResult } from "./expiry.js";

// Chainlink integration
export {
  CHAINLINK_CONFIDENCE,
  DEFAULT_CHAINLINK_HEARTBEATS,
  createFeedRegistration,
  feedTypeToEvidenceType,
  hashChainlinkRound,
  processChainlinkPriceUpdate,
  processChainlinkPoRUpdate,
  calculateChainlinkConfidence,
  mapChainlinkToPoTConfidence,
  checkSequencerStatus,
  updateSequencerStatus,
  ChainlinkFeedRegistry,
} from "./chainlink.js";

// Prediction market integration
export {
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
} from "./prediction_market.js";
