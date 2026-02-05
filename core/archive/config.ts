/**
 * Knowledge Archive Configuration
 *
 * Configuration settings for the archive node including index parameters,
 * cache sizes, and performance tuning.
 */

// ============================================================================
// Archive Configuration
// ============================================================================

/**
 * Configuration for the knowledge archive.
 */
export interface ArchiveConfig {
  // === Index Configuration ===

  /** Maximum depth for domain hierarchy traversal */
  domainIndexDepth: number;

  /** Number of confidence buckets (100 = 0.01 granularity) */
  confidenceBuckets: number;

  /** Maximum depth for reference graph traversal */
  referenceGraphMaxDepth: number;

  // === Semantic Search ===

  /** Embedding dimensions (768 for BERT-style models) */
  embeddingDimensions: number;

  /** Minimum similarity score to include in results */
  similarityThreshold: number;

  /** Maximum semantic similarity results per query */
  maxSimilarityResults: number;

  // === Cache Configuration ===

  /** Maximum entries in fact cache */
  factCacheSize: number;

  /** Maximum entries in domain cache */
  domainCacheSize: number;

  /** Maximum entries in reference cache */
  referenceCacheSize: number;

  /** Maximum entries in embedding cache */
  embeddingCacheSize: number;

  /** Cache TTL in milliseconds */
  cacheTtlMs: number;

  // === Query Configuration ===

  /** Default batch size for queries */
  defaultBatchSize: number;

  /** Maximum batch size allowed */
  maxBatchSize: number;

  /** Streaming chunk size */
  streamChunkSize: number;

  /** Default query timeout (ms) */
  defaultQueryTimeout: number;

  /** Maximum query timeout (ms) */
  maxQueryTimeout: number;

  // === Sync Configuration ===

  /** Auto-sync interval in blocks */
  syncIntervalBlocks: number;

  /** Maximum acceptable sync lag in blocks */
  maxLagBlocks: number;

  /** Batch size for sync operations */
  syncBatchSize: number;

  // === Truth Evaluation ===

  /** Minimum confidence to consider verdict 'true' */
  trueThreshold: number;

  /** Maximum confidence to consider verdict 'false' */
  falseThreshold: number;

  /** Minimum support ratio for 'true' verdict */
  minSupportRatio: number;

  // === Reference Graph ===

  /** Minimum reference strength to include */
  minReferenceStrength: number;

  /** Weight for direct references */
  directReferenceWeight: number;

  /** Decay factor per traversal depth */
  depthDecayFactor: number;
}

/**
 * Default archive configuration.
 */
export const DEFAULT_ARCHIVE_CONFIG: ArchiveConfig = {
  // Index configuration
  domainIndexDepth: 10,
  confidenceBuckets: 100,
  referenceGraphMaxDepth: 5,

  // Semantic search
  embeddingDimensions: 768,
  similarityThreshold: 0.7,
  maxSimilarityResults: 50,

  // Cache configuration
  factCacheSize: 10000,
  domainCacheSize: 1000,
  referenceCacheSize: 5000,
  embeddingCacheSize: 1000,
  cacheTtlMs: 60000, // 1 minute

  // Query configuration
  defaultBatchSize: 100,
  maxBatchSize: 1000,
  streamChunkSize: 50,
  defaultQueryTimeout: 30000, // 30 seconds
  maxQueryTimeout: 300000, // 5 minutes

  // Sync configuration
  syncIntervalBlocks: 100,
  maxLagBlocks: 1000,
  syncBatchSize: 100,

  // Truth evaluation
  trueThreshold: 0.7,
  falseThreshold: 0.3,
  minSupportRatio: 0.6,

  // Reference graph
  minReferenceStrength: 0.1,
  directReferenceWeight: 1.0,
  depthDecayFactor: 0.8,
};

// ============================================================================
// Storage Namespaces
// ============================================================================

/**
 * LevelDB namespace prefixes for archive storage.
 */
export const ARCHIVE_NAMESPACES = {
  /** Facts indexed by ID */
  FACTS: 0x31,

  /** Domain hierarchy index */
  DOMAINS: 0x32,

  /** Reference graph */
  REFERENCES: 0x33,

  /** Confidence ranking index */
  CONFIDENCE: 0x34,

  /** Semantic embeddings */
  EMBEDDINGS: 0x35,

  /** Archive metadata */
  METADATA: 0x36,
} as const;

/**
 * Key prefixes for different index types.
 */
export const INDEX_PREFIXES = {
  // Domain index
  DOMAIN_ENTRY: "domain:",
  DOMAIN_FACTS: "domain_facts:",
  DOMAIN_CHILDREN: "domain_children:",

  // Confidence index
  CONFIDENCE_GLOBAL: "conf:",
  CONFIDENCE_DOMAIN: "conf_domain:",
  CONFIDENCE_STRATUM: "conf_stratum:",

  // Reference index
  REF_OUT: "ref_out:",
  REF_IN: "ref_in:",
  REF_STRENGTH: "ref_strength:",

  // Semantic index
  EMBEDDING: "emb:",
  EMBEDDING_DOMAIN: "emb_domain:",

  // Metadata
  SYNC_STATUS: "sync_status",
  STATISTICS: "statistics",
} as const;

// ============================================================================
// Query Defaults
// ============================================================================

/**
 * Default pagination settings.
 */
export const DEFAULT_PAGINATION = {
  limit: 100,
  sortBy: "confidence" as const,
  sortOrder: "desc" as const,
};

/**
 * Default query options.
 */
export const DEFAULT_QUERY_OPTIONS = {
  includeReferences: false,
  includeOntology: false,
  includeSimilarity: false,
  maxDepth: 3,
  maxSimilarityResults: 10,
};

/**
 * Default batch options.
 */
export const DEFAULT_BATCH_OPTIONS = {
  parallel: true,
  maxParallelism: 5,
  timeout: 30000,
  streaming: false,
};

// ============================================================================
// Confidence Bucket Utilities
// ============================================================================

/**
 * Convert confidence value to bucket index.
 */
export function confidenceToBucket(
  confidence: number,
  bucketCount: number = DEFAULT_ARCHIVE_CONFIG.confidenceBuckets,
): number {
  const clamped = Math.max(0, Math.min(1, confidence));
  const bucket = Math.floor(clamped * bucketCount);
  return Math.min(bucket, bucketCount - 1);
}

/**
 * Get bucket range for a confidence value.
 */
export function getBucketRange(
  bucket: number,
  bucketCount: number = DEFAULT_ARCHIVE_CONFIG.confidenceBuckets,
): { lower: number; upper: number } {
  const bucketSize = 1 / bucketCount;
  return {
    lower: bucket * bucketSize,
    upper: (bucket + 1) * bucketSize,
  };
}

/**
 * Get buckets for a confidence range.
 */
export function getBucketsInRange(
  minConfidence: number,
  maxConfidence: number,
  bucketCount: number = DEFAULT_ARCHIVE_CONFIG.confidenceBuckets,
): number[] {
  const minBucket = confidenceToBucket(minConfidence, bucketCount);
  const maxBucket = confidenceToBucket(maxConfidence, bucketCount);
  const buckets: number[] = [];
  for (let i = minBucket; i <= maxBucket; i++) {
    buckets.push(i);
  }
  return buckets;
}

// ============================================================================
// Key Generation Utilities
// ============================================================================

/**
 * Generate a domain index key.
 */
export function makeDomainKey(domainId: string): string {
  return `${INDEX_PREFIXES.DOMAIN_ENTRY}${domainId}`;
}

/**
 * Generate a domain facts key with confidence bucket.
 */
export function makeDomainFactsKey(
  domainId: string,
  bucket: number,
  factId: string,
): string {
  const paddedBucket = bucket.toString().padStart(3, "0");
  return `${INDEX_PREFIXES.DOMAIN_FACTS}${domainId}:${paddedBucket}:${factId}`;
}

/**
 * Generate a confidence index key.
 */
export function makeConfidenceKey(bucket: number, factId: string): string {
  const paddedBucket = bucket.toString().padStart(3, "0");
  return `${INDEX_PREFIXES.CONFIDENCE_GLOBAL}${paddedBucket}:${factId}`;
}

/**
 * Generate a reference key (outgoing).
 */
export function makeRefOutKey(
  factId: string,
  refType: string,
  targetId: string,
): string {
  return `${INDEX_PREFIXES.REF_OUT}${factId}:${refType}:${targetId}`;
}

/**
 * Generate a reference key (incoming).
 */
export function makeRefInKey(
  factId: string,
  refType: string,
  sourceId: string,
): string {
  return `${INDEX_PREFIXES.REF_IN}${factId}:${refType}:${sourceId}`;
}

/**
 * Generate an embedding key.
 */
export function makeEmbeddingKey(factId: string): string {
  return `${INDEX_PREFIXES.EMBEDDING}${factId}`;
}
