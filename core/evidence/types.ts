/**
 * Evidence Management System Types
 *
 * Comprehensive types for evidence lifecycle management including:
 * - Automatic archiving (IPFS, Arweave, Wayback Machine)
 * - Evidence deduplication (exact, near-duplicate, related)
 * - Evidence versioning (corrections, updates, retractions)
 * - Evidence expiry (eternal, heartbeat, decay-based)
 * - Chainlink oracle integration (price feeds, PoR, functions, CCIP)
 *
 * PP HIERARCHY:
 *   TRUTH: Evidence is the foundation of knowledge
 *   UNDERSTANDING: Lifecycle from submission through expiry
 *   BEAUTY: Elegant integration with existing Reference types
 *   JUSTICE: Fair rewards for submitters, confirmers, maintainers
 *   CREATIVITY: New infrastructure for oracles and archiving
 */

import type { DID } from "../../core/agents/identity/did.js";
import type {
  EvidenceIndependence,
  ReferenceType,
} from "../knowledge/types.js";

// ============ Evidence Status ============

/**
 * Evidence lifecycle status
 */
export type EvidenceStatus =
  | "pending_archive" // Awaiting archival
  | "archived" // Successfully archived
  | "archive_failed" // Archival failed (will retry)
  | "active" // Actively in use
  | "stale" // Beyond freshness threshold
  | "expired" // Past expiry, no longer valid
  | "superseded" // Replaced by newer version
  | "invalidated"; // Proven false

// ============ Evidence Types (Extended) ============

/**
 * Extended evidence types including Chainlink oracles
 */
export type EvidenceType =
  // Existing types (from ReferenceType)
  | "on_chain"
  | "cryptographic_proof"
  | "computational"
  | "formal_proof"
  | "peer_reviewed"
  | "institutional"
  | "oracle"
  | "archived_web"
  | "attestation"
  | "inference"
  // New Chainlink types
  | "chainlink_price_feed"
  | "chainlink_por" // Proof of Reserve
  | "chainlink_functions"
  | "chainlink_ccip" // Cross-Chain Interoperability
  | "prediction_market";

/**
 * Evidence type strength values (extended from EVIDENCE_TYPE_STRENGTH)
 */
export const EVIDENCE_TYPE_STRENGTH: Record<EvidenceType, number> = {
  on_chain: 1.0,
  cryptographic_proof: 0.99,
  computational: 0.95,
  formal_proof: 0.95,
  chainlink_price_feed: 0.92,
  chainlink_por: 0.91,
  chainlink_ccip: 0.9,
  chainlink_functions: 0.88,
  prediction_market: 0.88,
  peer_reviewed: 0.85,
  institutional: 0.8,
  oracle: 0.7,
  archived_web: 0.6,
  attestation: 0.5,
  inference: 0.4,
};

// ============ Canonical Evidence Types (Go/Cosmos SDK alignment) ============

/**
 * Canonical evidence types matching Go x/knowledge module (6 types).
 * Used at serialization boundaries when communicating with the Cosmos SDK chain.
 */
export type CanonicalEvidenceType =
  | "on_chain"
  | "oracle_feed"
  | "attestation"
  | "computation"
  | "document"
  | "experiment";

/**
 * Map TS's granular 14-type evidence taxonomy to Go's canonical 6 types.
 * Internal TS code retains granularity; this mapper is used at wire boundaries.
 */
export function toCanonicalEvidenceType(
  t: EvidenceType,
): CanonicalEvidenceType {
  switch (t) {
    case "on_chain":
      return "on_chain";
    case "cryptographic_proof":
    case "computational":
    case "formal_proof":
      return "computation";
    case "peer_reviewed":
      return "experiment";
    case "institutional":
    case "archived_web":
      return "document";
    case "oracle":
    case "chainlink_price_feed":
    case "chainlink_por":
    case "chainlink_functions":
    case "chainlink_ccip":
    case "prediction_market":
      return "oracle_feed";
    case "attestation":
    case "inference":
      return "attestation";
  }
}

/**
 * Canonical evidence type strength values (aligned with Go uint64 scale).
 * Go stores as 0-1,000,000; TS uses 0.0-1.0 float equivalents.
 */
export const CANONICAL_EVIDENCE_TYPE_STRENGTH: Record<
  CanonicalEvidenceType,
  number
> = {
  on_chain: 1.0, // Go: 1_000_000
  computation: 0.95, // Go: 950_000
  experiment: 0.85, // Go: 850_000
  oracle_feed: 0.7, // Go: 700_000
  document: 0.6, // Go: 600_000
  attestation: 0.5, // Go: 500_000
};

// ============ Archive Types ============

/**
 * Archive provider types
 */
export type ArchiveProvider = "wayback" | "ipfs" | "arweave" | "on_chain";

/**
 * Archive location record
 */
export interface ArchiveLocation {
  /** Archive provider */
  provider: ArchiveProvider;

  /** Content identifier (URL, CID, hash) */
  identifier: string;

  /** Timestamp of archival */
  archivedAt: number;

  /** Block number when archived */
  archivedAtBlock: bigint;

  /** Hash of archived content (for integrity verification) */
  contentHash: string;

  /** Is this the primary archive? */
  isPrimary: boolean;

  /** Has archive been verified accessible? */
  verified: boolean;

  /** When verification occurred */
  verifiedAt?: number;

  /** Archiver DID (for rewards) */
  archiver?: DID;
}

/**
 * Archive request status
 */
export type ArchiveRequestStatus =
  | "pending"
  | "in_progress"
  | "completed"
  | "failed"
  | "permanent_failure";

/**
 * Archive priority levels
 */
export type ArchivePriority = "low" | "normal" | "high" | "critical";

/**
 * Archive request record
 */
export interface ArchiveRequest {
  id: string;
  evidenceId: string;
  originalUrl: string;
  requestedAt: number;
  requestedAtBlock: bigint;
  requester: DID;
  status: ArchiveRequestStatus;
  priority: ArchivePriority;
  retryCount: number;
  maxRetries: number;
  lastError?: string;
  targetProviders: ArchiveProvider[];
}

/**
 * Archive provider configuration
 */
export interface ArchiveProviderConfig {
  provider: ArchiveProvider;
  enabled: boolean;
  priority: number;
  config: Record<string, unknown>;
}

/**
 * Archive system configuration
 */
export interface ArchiveConfig {
  /** Enable automatic archiving */
  autoArchiveEnabled: boolean;

  /** Archive providers in priority order */
  providers: ArchiveProviderConfig[];

  /** Minimum archive copies required */
  minArchiveCopies: number;

  /** Archive verification interval (blocks) */
  verificationInterval: number;

  /** Maximum retry attempts */
  maxRetries: number;

  /** Retry backoff (blocks) */
  retryBackoff: number;
}

/**
 * Default archive configuration
 */
export const DEFAULT_ARCHIVE_CONFIG: ArchiveConfig = {
  autoArchiveEnabled: true,
  providers: [
    { provider: "ipfs", enabled: true, priority: 1, config: {} },
    { provider: "arweave", enabled: true, priority: 2, config: {} },
    { provider: "wayback", enabled: true, priority: 3, config: {} },
  ],
  minArchiveCopies: 2,
  verificationInterval: 10000, // ~7 hours at 2521ms blocks
  maxRetries: 5,
  retryBackoff: 100, // ~4 minutes
};

// ============ Deduplication Types ============

/**
 * Content features for semantic similarity
 */
export interface ContentFeatures {
  /** Word/token count */
  tokenCount: number;

  /** Key entities extracted */
  entities: string[];

  /** Numeric values present */
  numericValues: number[];

  /** Temporal references */
  temporalReferences: string[];

  /** Domain/topic classification */
  topicClassification: string;
}

/**
 * Evidence fingerprint for deduplication
 */
export interface EvidenceFingerprint {
  /** Content hash (exact match) - SHA-512/256 */
  contentHash: string;

  /** Perceptual hash (near-duplicate detection) */
  perceptualHash: string;

  /** Normalized source URL */
  normalizedSource: string;

  /** Key content features */
  contentFeatures: ContentFeatures;

  /** When first seen */
  firstSeen: number;

  /** First submitter DID */
  firstSubmitter: DID;
}

/**
 * Duplicate detection type
 */
export type DuplicationType = "exact" | "near" | "related" | "none";

/**
 * Duplicate detection result
 */
export interface DuplicateDetectionResult {
  /** Is this a duplicate? */
  isDuplicate: boolean;

  /** Type of duplication */
  duplicationType: DuplicationType;

  /** Similarity score (0-1) */
  similarityScore: number;

  /** Matching evidence IDs */
  matchingEvidenceIds: string[];

  /** Original evidence ID (if duplicate) */
  originalEvidenceId?: string;

  /** First submitter (for reward attribution) */
  firstSubmitter?: DID;

  /** Is this within simultaneous submission window? */
  isSimultaneous: boolean;

  /** Eligible for confirmation credit? */
  eligibleForConfirmationCredit: boolean;
}

/**
 * Deduplication configuration
 */
export interface DeduplicationConfig {
  /** Enable content hash exact match */
  exactMatchEnabled: boolean;

  /** Perceptual hash threshold (0-1, lower = stricter) */
  perceptualHashThreshold: number;

  /** Feature similarity threshold */
  featureSimilarityThreshold: number;

  /** Time window for "simultaneous" submissions (blocks) */
  simultaneousWindow: number;

  /** Confirmation credit rate (% of original reward) */
  confirmationCreditRate: number;
}

/**
 * Default deduplication configuration
 */
export const DEFAULT_DEDUPLICATION_CONFIG: DeduplicationConfig = {
  exactMatchEnabled: true,
  perceptualHashThreshold: 0.15, // 85% similar
  featureSimilarityThreshold: 0.8,
  simultaneousWindow: 10, // ~25 seconds at 2521ms blocks
  confirmationCreditRate: 0.1, // 10% of original reward
};

// ============ Version Types ============

/**
 * Version reason types
 */
export type VersionReason =
  | "correction"
  | "update"
  | "supplement"
  | "retraction";

/**
 * Evidence version information
 */
export interface EvidenceVersion {
  /** Current version number */
  versionNumber: number;

  /** Hash of previous version (null for v1) */
  previousVersionHash: string | null;

  /** ID of superseding evidence (if superseded) */
  supersededBy: string | null;

  /** IDs of all versions in this evidence chain */
  versionChain: string[];

  /** Timestamp of version creation */
  createdAt: number;

  /** Reason for new version */
  versionReason?: VersionReason;

  /** Description of changes */
  changeDescription?: string;

  /** Confidence impact of versioning (-1 to +1) */
  confidenceImpact: number;
}

/**
 * Version transition record
 */
export interface EvidenceVersionTransition {
  id: string;
  fromVersion: string; // Evidence ID
  toVersion: string; // Evidence ID
  transitionType: VersionReason;
  timestamp: number;
  blockNumber: bigint;
  submitter: DID;
  description: string;
  confidenceImpact: number;
}

/**
 * Confidence impact by version reason
 */
export const VERSION_CONFIDENCE_IMPACT: Record<
  VersionReason,
  { min: number; max: number }
> = {
  correction: { min: -0.15, max: -0.05 },
  update: { min: 0, max: 0.1 },
  supplement: { min: 0.05, max: 0.15 },
  retraction: { min: -1.0, max: -0.5 },
};

// ============ Expiry Types ============

/**
 * Expiry type
 */
export type ExpiryType =
  | "eternal" // Never expires (formal proofs, axioms)
  | "block_based" // Expires after N blocks
  | "heartbeat" // Expires if no heartbeat within threshold
  | "event_based" // Expires on specific event
  | "decay_based"; // Gradually loses validity

/**
 * Heartbeat configuration for oracle feeds
 */
export interface HeartbeatConfig {
  /** Expected heartbeat interval (blocks) */
  expectedInterval: number;

  /** Maximum allowed staleness (blocks) */
  maxStaleness: number;

  /** Deviation threshold (%) that triggers update */
  deviationThreshold: number;

  /** Last heartbeat received */
  lastHeartbeat: number;

  /** Last heartbeat block */
  lastHeartbeatBlock: bigint;

  /** Is feed currently stale? */
  isStale: boolean;
}

/**
 * Evidence expiry configuration
 */
export interface EvidenceExpiry {
  /** Expiry type */
  expiryType: ExpiryType;

  /** Absolute expiry block (if applicable) */
  expiresAtBlock?: bigint;

  /** Relative expiry (blocks after submission) */
  expiresAfterBlocks?: number;

  /** Heartbeat config (for oracles) */
  heartbeatConfig?: HeartbeatConfig;

  /** Current freshness score (0-1) */
  freshnessScore: number;

  /** Last freshness update */
  freshnessUpdatedAt: number;

  /** Has this evidence expired? */
  isExpired: boolean;

  /** When did it expire? */
  expiredAt?: number;
}

/**
 * Default expiry configurations by evidence type
 */
export const DEFAULT_EXPIRY_CONFIGS: Record<
  EvidenceType,
  Partial<EvidenceExpiry>
> = {
  on_chain: { expiryType: "eternal", freshnessScore: 1.0, isExpired: false },
  cryptographic_proof: {
    expiryType: "eternal",
    freshnessScore: 1.0,
    isExpired: false,
  },
  formal_proof: {
    expiryType: "eternal",
    freshnessScore: 1.0,
    isExpired: false,
  },
  computational: {
    expiryType: "eternal",
    freshnessScore: 1.0,
    isExpired: false,
  },
  chainlink_price_feed: {
    expiryType: "heartbeat",
    heartbeatConfig: {
      expectedInterval: 3600, // ~2.5 hours at 2521ms blocks
      maxStaleness: 7200, // ~5 hours
      deviationThreshold: 1.0, // 1%
      lastHeartbeat: 0,
      lastHeartbeatBlock: 0n,
      isStale: false,
    },
    freshnessScore: 1.0,
    isExpired: false,
  },
  chainlink_por: {
    expiryType: "heartbeat",
    heartbeatConfig: {
      expectedInterval: 86400, // ~60 hours
      maxStaleness: 172800, // ~120 hours
      deviationThreshold: 5.0, // 5%
      lastHeartbeat: 0,
      lastHeartbeatBlock: 0n,
      isStale: false,
    },
    freshnessScore: 1.0,
    isExpired: false,
  },
  chainlink_functions: {
    expiryType: "block_based",
    expiresAfterBlocks: 36000, // ~25 hours
    freshnessScore: 1.0,
    isExpired: false,
  },
  chainlink_ccip: {
    expiryType: "eternal",
    freshnessScore: 1.0,
    isExpired: false,
  },
  prediction_market: {
    expiryType: "eternal",
    freshnessScore: 1.0,
    isExpired: false,
  },
  peer_reviewed: {
    expiryType: "decay_based",
    expiresAfterBlocks: 500000, // ~15 days half-life
    freshnessScore: 1.0,
    isExpired: false,
  },
  institutional: {
    expiryType: "decay_based",
    expiresAfterBlocks: 200000, // ~6 days
    freshnessScore: 1.0,
    isExpired: false,
  },
  oracle: {
    expiryType: "heartbeat",
    heartbeatConfig: {
      expectedInterval: 7200,
      maxStaleness: 14400,
      deviationThreshold: 2.0,
      lastHeartbeat: 0,
      lastHeartbeatBlock: 0n,
      isStale: false,
    },
    freshnessScore: 1.0,
    isExpired: false,
  },
  archived_web: {
    expiryType: "decay_based",
    expiresAfterBlocks: 1000000, // ~30 days
    freshnessScore: 1.0,
    isExpired: false,
  },
  attestation: {
    expiryType: "decay_based",
    expiresAfterBlocks: 100000, // ~3 days
    freshnessScore: 1.0,
    isExpired: false,
  },
  inference: {
    expiryType: "decay_based",
    expiresAfterBlocks: 50000, // ~1.5 days
    freshnessScore: 1.0,
    isExpired: false,
  },
};

// ============ Chainlink Types ============

/**
 * Chainlink feed types
 */
export type ChainlinkFeedType = "price_feed" | "por" | "functions" | "ccip";

/**
 * L2 Sequencer status (for Arbitrum, Optimism, etc.)
 */
export interface L2SequencerStatus {
  /** Is sequencer up? */
  isSequencerUp: boolean;

  /** When sequencer came back up */
  startedAt: number;

  /** Grace period after recovery (blocks) */
  gracePeriodBlocks: number;

  /** Is grace period active? */
  inGracePeriod: boolean;
}

/**
 * Proof of Reserve metadata
 */
export interface ProofOfReserveMetadata {
  /** Asset being attested */
  asset: string;

  /** Reserve type */
  reserveType: "onchain" | "offchain" | "cross_chain";

  /** Total reserves */
  totalReserves: bigint;

  /** Verification method */
  verificationMethod: string;

  /** Auditor (if any) */
  auditor?: string;
}

/**
 * Chainlink Functions metadata
 */
export interface ChainlinkFunctionsMetadata {
  /** DON ID */
  donId: string;

  /** Subscription ID */
  subscriptionId: bigint;

  /** Source hash */
  sourceHash: string;

  /** Request ID */
  requestId: string;

  /** Gas limit */
  gasLimit: number;
}

/**
 * Chainlink-specific metadata
 */
export interface ChainlinkMetadata {
  /** Aggregator address */
  aggregatorAddress: string;

  /** Number of oracles in aggregator */
  oracleCount: number;

  /** Minimum submissions for answer */
  minSubmissions: number;

  /** Phase ID */
  phaseId: number;

  /** L2 sequencer status (for L2s) */
  sequencerStatus?: L2SequencerStatus;

  /** Proof of Reserve specific */
  porMetadata?: ProofOfReserveMetadata;

  /** Functions specific */
  functionsMetadata?: ChainlinkFunctionsMetadata;
}

/**
 * Oracle metadata (generic)
 */
export interface OracleMetadata {
  /** Oracle provider */
  provider:
    | "chainlink"
    | "band"
    | "api3"
    | "polymarket"
    | "kalshi"
    | "metaculus"
    | "custom";

  /** Feed/contract address */
  feedAddress: string;

  /** Chain ID (for multi-chain) */
  chainId: number;

  /** Feed name/description */
  feedName: string;

  /** Decimals for numeric data */
  decimals: number;

  /** Current round ID */
  roundId: bigint;

  /** Last update timestamp */
  updatedAt: number;

  /** Last answered in round */
  answeredInRound: bigint;

  /** Chainlink-specific metadata */
  chainlinkMetadata?: ChainlinkMetadata;

  /** Prediction market-specific metadata */
  predictionMarketMetadata?: PredictionMarketMetadata;
}

/**
 * Chainlink round data (from aggregator)
 */
export interface ChainlinkRoundData {
  roundId: bigint;
  answer: bigint;
  startedAt: number;
  updatedAt: number;
  answeredInRound: bigint;
}

/**
 * Chainlink feed registration
 */
export interface ChainlinkFeedRegistration {
  feedAddress: string;
  chainId: number;
  feedType: ChainlinkFeedType;
  decimals: number;
  description: string;
  version: number;
  aggregatorAddress: string;
  heartbeatConfig: HeartbeatConfig;
  registeredAt: number;
  isActive: boolean;
}

// ============ Prediction Market Types ============

/**
 * Prediction market platform identifiers
 */
export type PredictionMarketPlatform =
  | "polymarket"
  | "kalshi"
  | "metaculus"
  | "custom";

/**
 * Prediction market outcome types
 */
export type PredictionMarketType = "binary_outcome" | "categorical" | "scalar";

/**
 * Market resolution lifecycle status
 */
export type MarketResolutionStatus =
  | "open"
  | "closed"
  | "resolved"
  | "disputed"
  | "voided";

/**
 * Prediction market metadata for oracle evidence
 */
export interface PredictionMarketMetadata {
  /** Platform identifier */
  platform: PredictionMarketPlatform;

  /** Platform-specific market ID */
  marketId: string;

  /** Human-readable market question */
  question: string;

  /** Market outcome type */
  marketType: PredictionMarketType;

  /** Current resolution status */
  resolutionStatus: MarketResolutionStatus;

  /** Resolved outcome (null if unresolved) */
  resolvedOutcome: string | null;

  /** When market was resolved (epoch ms) */
  resolvedAt: number | null;

  /** Total trading volume in USD */
  volumeUsd: number;

  /** Number of unique traders */
  traderCount: number;

  /** Final closing probability (0-1 for binary markets) */
  closingProbability: number;

  /** Resolution source description */
  resolutionSource: string;

  /** When market was created (epoch ms) */
  createdAt: number;

  /** When market closed for trading (epoch ms) */
  closedAt: number | null;

  /** Block at which dispute window ends */
  disputeWindowEnd: bigint;

  /** Whether dispute window has elapsed */
  disputeWindowElapsed: boolean;
}

/**
 * Prediction market registration for the registry
 */
export interface PredictionMarketRegistration {
  /** Platform identifier */
  platform: PredictionMarketPlatform;

  /** Platform-specific market ID */
  marketId: string;

  /** Human-readable market question */
  question: string;

  /** Market outcome type */
  marketType: PredictionMarketType;

  /** Market URL for reference */
  marketUrl: string;

  /** When registered (epoch ms) */
  registeredAt: number;

  /** Whether this registration is active */
  isActive: boolean;

  /** Minimum volume in USD for this market to qualify */
  minVolumeUsd: number;
}

// ============ Evidence Metadata ============

/**
 * Extended evidence metadata
 */
export interface EvidenceMetadata {
  /** Original URL */
  url?: string;

  /** Academic DOI */
  doi?: string;

  /** Archive URL (legacy) */
  archiveUrl?: string;

  /** Oracle ID (legacy) */
  oracleId?: string;

  /** Methodology description */
  methodology?: string;

  /** Source organization */
  sourceOrganization?: string;

  /** Author/submitter description */
  author?: string;

  /** Publication date */
  publicationDate?: string;

  /** Additional tags */
  tags?: string[];

  /** Custom metadata */
  custom?: Record<string, unknown>;
}

// ============ Core Evidence Type ============

/**
 * Evidence - the core type for the Evidence Management System
 *
 * Extends the existing Reference type with:
 * - Lifecycle management (archiving, expiry)
 * - Deduplication tracking
 * - Version history
 * - Oracle integration
 */
export interface Evidence {
  /** Unique identifier */
  id: string;

  /** Evidence type */
  type: EvidenceType;

  /** Original source URL/identifier */
  originalSource: string;

  /** Primary content hash (SHA-512/256) */
  contentHash: string;

  /** Perceptual hash for near-duplicate detection */
  perceptualHash?: string;

  /** Archive locations (multiple for redundancy) */
  archives: ArchiveLocation[];

  /** When evidence was first submitted */
  submittedAt: number;

  /** Block number of submission */
  submittedAtBlock: bigint;

  /** First submitter (for reward priority) */
  firstSubmitter: DID;

  /** Subsequent submitters (for confirmation credit) */
  confirmedBy: DID[];

  /** Current lifecycle status */
  status: EvidenceStatus;

  /** Evidence direction */
  direction: "supporting" | "contradicting" | "neutral";

  /** Intrinsic strength based on type */
  intrinsicStrength: number;

  /** Current reliability score (may decay over time) */
  reliabilityScore: number;

  /** Independence metrics */
  independence: EvidenceIndependence;

  /** Version information */
  version: EvidenceVersion;

  /** Expiry configuration */
  expiry: EvidenceExpiry;

  /** Oracle metadata (for oracle evidence) */
  oracleMetadata?: OracleMetadata;

  /** Extended metadata */
  metadata: EvidenceMetadata;

  /** Related fact IDs */
  relatedFactIds: string[];
}

// ============ State Types ============

/**
 * On-chain evidence state (stored in SMT)
 */
export interface EvidenceState {
  id: string;
  contentHash: string;
  perceptualHash: string;
  type: EvidenceType;
  status: EvidenceStatus;
  firstSubmitter: DID;
  confirmedBy: DID[];
  submittedAt: bigint;
  currentVersion: number;
  latestVersionId: string;
  archives: ArchiveLocation[];
  reliabilityScore: number;
  freshnessScore: number;
  expiresAt: bigint | null;
  isExpired: boolean;
  oracleMetadata?: OracleMetadata;
}

// ============ Transaction Payloads ============

/**
 * Submit new evidence
 */
export interface EvidenceSubmitPayload {
  originalSource: string;
  contentHash: string;
  perceptualHash?: string;
  type: EvidenceType;
  direction: "supporting" | "contradicting" | "neutral";
  relatedFactId?: string;
  metadata: EvidenceMetadata;
  oracleMetadata?: OracleMetadata;
  stake: bigint;
}

/**
 * Request evidence archival
 */
export interface EvidenceArchiveRequestPayload {
  evidenceId: string;
  priority: ArchivePriority;
  providers: ArchiveProvider[];
}

/**
 * Complete evidence archival
 */
export interface EvidenceArchiveCompletePayload {
  evidenceId: string;
  archiveLocation: ArchiveLocation;
}

/**
 * Create new evidence version
 */
export interface EvidenceVersionUpdatePayload {
  previousEvidenceId: string;
  newEvidence: EvidenceSubmitPayload;
  versionReason: VersionReason;
  changeDescription: string;
  confidenceImpact: number;
}

/**
 * Confirm duplicate evidence
 */
export interface EvidenceConfirmDuplicatePayload {
  evidenceId: string;
  duplicateContentHash: string;
}

/**
 * Oracle heartbeat update
 */
export interface OracleHeartbeatPayload {
  evidenceId: string;
  roundId: bigint;
  answer: bigint;
  updatedAt: number;
  answeredInRound: bigint;
  sequencerStatus?: L2SequencerStatus;
}

/**
 * Register Chainlink feed
 */
export interface ChainlinkFeedRegisterPayload {
  feedAddress: string;
  chainId: number;
  feedName: string;
  feedType: ChainlinkFeedType;
  decimals: number;
  heartbeatConfig: HeartbeatConfig;
  aggregatorAddress: string;
}

// ============ Anti-Gaming ============

/**
 * Evidence anti-gaming configuration
 */
export const EVIDENCE_ANTI_GAMING = {
  // Archive gaming
  /** Minimum time between archive reward claims (blocks) */
  ARCHIVE_COOLDOWN_BLOCKS: 1000,

  /** Maximum archive rewards per submitter per day */
  MAX_DAILY_ARCHIVE_REWARDS: 10,

  // Duplicate gaming
  /** Window for "simultaneous" submission (blocks) */
  SIMULTANEOUS_SUBMISSION_WINDOW: 10,

  /** Penalty for submitting known duplicate */
  KNOWN_DUPLICATE_PENALTY: 0.9,

  /** Confirmation credit cap */
  MAX_CONFIRMATION_CREDIT: 0.2,

  // Oracle gaming
  /** Minimum interval between oracle updates from same source */
  ORACLE_UPDATE_COOLDOWN: 50,

  /** Maximum oracle updates per block */
  MAX_ORACLE_UPDATES_PER_BLOCK: 10,

  // Version gaming
  /** Minimum interval between version updates */
  VERSION_UPDATE_COOLDOWN: 1000,

  /** Maximum versions per evidence chain */
  MAX_VERSION_CHAIN_LENGTH: 10,

  /** Penalty for excessive versioning (per version beyond 5) */
  EXCESSIVE_VERSION_PENALTY: 0.05,

  // Self-dealing
  /** Discount for self-confirming evidence */
  SELF_CONFIRMATION_DISCOUNT: 0.95,

  /** Maximum self-submitted evidence per fact */
  MAX_SELF_EVIDENCE_PER_FACT: 3,
} as const;

// ============ Reward Types ============

/**
 * Evidence reward distribution weights
 */
export const EVIDENCE_REWARD_WEIGHTS = {
  /** First submitter of new evidence */
  FIRST_SUBMISSION: 0.5,

  /** Successful archival */
  ARCHIVAL: 0.15,

  /** Duplicate confirmation */
  CONFIRMATION: 0.1,

  /** Version updates */
  VERSION_UPDATE: 0.1,

  /** Oracle maintenance (heartbeat) */
  ORACLE_MAINTENANCE: 0.1,

  /** Archive verification */
  ARCHIVE_VERIFICATION: 0.05,
} as const;

/**
 * Evidence reward distribution result
 */
export interface EvidenceRewardDistribution {
  /** Research share (22%) */
  researchShare: bigint;

  /** First submitter reward */
  firstSubmitter: bigint;

  /** Archiver rewards */
  archivers: Array<{
    archiver: DID;
    amount: bigint;
    provider: ArchiveProvider;
  }>;

  /** Confirmer rewards */
  confirmers: Array<{
    confirmer: DID;
    amount: bigint;
  }>;

  /** Version updater reward */
  versionUpdater: bigint;

  /** Oracle maintainer reward */
  oracleMaintainer: bigint;

  /** Protocol treasury */
  protocol: bigint;
}

// ============ Compatibility ============

/**
 * Convert legacy Reference type to Evidence
 */
export function referenceToEvidence(
  ref: {
    id: string;
    type: ReferenceType;
    source: string;
    contentHash: string;
    retrievedAt: number;
    reliabilityScore: number;
    independence?: EvidenceIndependence;
    direction: "supporting" | "contradicting" | "neutral";
    intrinsicStrength: number;
    metadata?: EvidenceMetadata;
  },
  submitter: DID = "did:lgm:legacy" as DID,
): Evidence {
  const evidenceType = ref.type as EvidenceType;

  return {
    id: ref.id,
    type: evidenceType,
    originalSource: ref.source,
    contentHash: ref.contentHash,
    archives: ref.metadata?.archiveUrl
      ? [
          {
            provider: "wayback" as ArchiveProvider,
            identifier: ref.metadata.archiveUrl,
            archivedAt: ref.retrievedAt,
            archivedAtBlock: 0n,
            contentHash: ref.contentHash,
            isPrimary: true,
            verified: false,
          },
        ]
      : [],
    submittedAt: ref.retrievedAt,
    submittedAtBlock: 0n,
    firstSubmitter: submitter,
    confirmedBy: [],
    status: "active",
    direction: ref.direction,
    intrinsicStrength: ref.intrinsicStrength,
    reliabilityScore: ref.reliabilityScore,
    independence: ref.independence || {
      sourceIndependence: 0.5,
      methodIndependence: 0.5,
      temporalIndependence: 0.5,
      aggregateIndependence: 0.5,
    },
    version: {
      versionNumber: 1,
      previousVersionHash: null,
      supersededBy: null,
      versionChain: [ref.id],
      createdAt: ref.retrievedAt,
      confidenceImpact: 0,
    },
    expiry: {
      ...(DEFAULT_EXPIRY_CONFIGS[evidenceType] ||
        DEFAULT_EXPIRY_CONFIGS.archived_web),
      freshnessScore: 1.0,
      freshnessUpdatedAt: ref.retrievedAt,
      isExpired: false,
    } as EvidenceExpiry,
    metadata: ref.metadata || {},
    relatedFactIds: [],
  };
}

/**
 * Convert Evidence back to Reference for legacy compatibility
 */
export function evidenceToReference(evidence: Evidence): {
  id: string;
  type: ReferenceType;
  source: string;
  contentHash: string;
  retrievedAt: number;
  reliabilityScore: number;
  independence: EvidenceIndependence;
  direction: "supporting" | "contradicting" | "neutral";
  intrinsicStrength: number;
  metadata: EvidenceMetadata;
} {
  return {
    id: evidence.id,
    type: evidence.type as ReferenceType,
    source: evidence.originalSource,
    contentHash: evidence.contentHash,
    retrievedAt: evidence.submittedAt,
    reliabilityScore: evidence.reliabilityScore,
    independence: evidence.independence,
    direction: evidence.direction,
    intrinsicStrength: evidence.intrinsicStrength,
    metadata: {
      ...evidence.metadata,
      archiveUrl: evidence.archives.find((a) => a.isPrimary)?.identifier,
      oracleId: evidence.oracleMetadata?.feedAddress,
    },
  };
}
