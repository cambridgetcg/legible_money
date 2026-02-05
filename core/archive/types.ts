/**
 * Knowledge Archive Types
 *
 * Types for the archive node that serves the Ontological Tree of Knowledge
 * to AI agents with confidence-based queries, truth evaluation, and
 * relationship navigation.
 */

import type {
  FactState,
  FactStatus,
  ReferenceType,
} from "../../state/types.js";
import type {
  OntologicalStratum,
  OntologicalDomain,
  StratumDependency,
} from "../knowledge/ontology/types.js";

// ============================================================================
// Archive Entry Types
// ============================================================================

/**
 * Extended fact entry with archive metadata.
 */
export interface ArchiveFactEntry {
  /** The underlying fact state */
  fact: FactState;

  /** Resolved ontological domain */
  domain: OntologicalDomain;

  /** Stratum level (fundamental → technological) */
  stratum: OntologicalStratum;

  /** References to/from this fact */
  references: ArchiveReference[];

  /** Archive-specific metadata */
  archiveMetadata: ArchiveMetadata;
}

/**
 * Reference between facts in the archive.
 */
export interface ArchiveReference {
  /** Source fact ID */
  factId: string;

  /** Target fact ID */
  targetFactId: string;

  /** Type of reference */
  referenceType: ReferenceType;

  /** Strength of the reference (0-1) */
  strength: number;

  /** Whether the relationship is symmetric (e.g., contradicts) */
  bidirectional: boolean;
}

/**
 * Archive metadata for tracking and analytics.
 */
export interface ArchiveMetadata {
  /** Block number when indexed */
  indexedAt: bigint;

  /** Block number of last update */
  lastUpdated: bigint;

  /** Number of times queried */
  queryCount: number;

  /** How often this fact is cited by others */
  citationCount: number;
}

// ============================================================================
// Truth Evaluation Types
// ============================================================================

/**
 * Result of truth evaluation for a fact.
 */
export interface TruthEvaluation {
  /** The fact being evaluated */
  factId: string;

  /** The claim text */
  claim: string;

  /** Verdict based on confidence and supporting evidence */
  verdict: TruthVerdict;

  /** Effective confidence after all adjustments */
  confidence: number;

  /** Breakdown of supporting vs contradicting evidence */
  confidenceRatio: ConfidenceRatio;

  /** Facts that support this claim */
  supportingFacts: SupportingFact[];

  /** Facts that contradict this claim */
  contradictingFacts: ContradictingFact[];

  /** Ontological context affecting confidence */
  ontologicalContext: OntologicalContext;

  /** Reasoning for the verdict */
  reasoning: string;
}

/**
 * Truth verdict with thresholds.
 */
export type TruthVerdict = "true" | "false" | "uncertain";

/**
 * Thresholds for truth determination.
 */
export const TRUTH_THRESHOLDS = {
  /** Minimum confidence to consider 'true' */
  trueThreshold: 0.7,
  /** Maximum confidence to consider 'false' (when contradicted) */
  falseThreshold: 0.3,
  /** Below true, above false = uncertain */
} as const;

/**
 * Confidence ratio showing supporting vs contradicting weight.
 */
export interface ConfidenceRatio {
  /** Total weighted confidence of supporting facts */
  supporting: number;

  /** Total weighted confidence of contradicting facts */
  contradicting: number;

  /** Ratio: supporting / (supporting + contradicting) */
  ratio: number;

  /** Net confidence after weighing support/contradiction */
  netConfidence: number;
}

/**
 * A fact that supports the evaluated claim.
 */
export interface SupportingFact {
  /** Fact ID */
  factId: string;

  /** The supporting claim text */
  claim: string;

  /** Confidence of the supporting fact */
  confidence: number;

  /** How it supports (directly or extends) */
  referenceType: "supports" | "extends";

  /** Strength of the support relationship */
  strength: number;

  /** Domain of the supporting fact */
  domain: string;
}

/**
 * A fact that contradicts the evaluated claim.
 */
export interface ContradictingFact {
  /** Fact ID */
  factId: string;

  /** The contradicting claim text */
  claim: string;

  /** Confidence of the contradicting fact */
  confidence: number;

  /** Type of contradiction */
  contradictionType: "direct" | "implied";

  /** Strength of the contradiction */
  strength: number;

  /** Domain of the contradicting fact */
  domain: string;
}

/**
 * Ontological context affecting confidence evaluation.
 */
export interface OntologicalContext {
  /** Stratum of the fact's domain */
  stratum: OntologicalStratum;

  /** Maximum confidence allowed by stratum */
  stratumCeiling: number;

  /** Domain path from root */
  domainPath: string[];

  /** Cross-stratum dependencies */
  crossStratumDependencies: StratumDependency[];

  /** Cross-stratum discount applied */
  crossStratumDiscount: number;
}

// ============================================================================
// Related Statements Types
// ============================================================================

/**
 * Collection of related statements for a fact.
 */
export interface RelatedStatements {
  /** The fact for which relations are found */
  factId: string;

  /** The claim text */
  claim: string;

  /** Facts that support this one */
  supports: RelatedFact[];

  /** Facts that contradict this one */
  contradicts: RelatedFact[];

  /** Facts that extend this one */
  extends: RelatedFact[];

  /** Facts that specialize this one */
  specializes: RelatedFact[];

  /** Facts that generalize from this one */
  generalizes: RelatedFact[];

  /** Semantically similar facts (by embedding) */
  semanticallySimilar: SemanticMatch[];

  /** Summary statistics */
  statistics: RelationStatistics;
}

/**
 * A related fact with relationship metadata.
 */
export interface RelatedFact {
  /** Fact ID */
  factId: string;

  /** The claim text */
  claim: string;

  /** Domain of the related fact */
  domain: string;

  /** Stratum of the related fact */
  stratum: OntologicalStratum;

  /** Confidence of the related fact */
  confidence: number;

  /** Strength of the relationship */
  referenceStrength: number;
}

/**
 * A semantically similar fact found via embedding similarity.
 */
export interface SemanticMatch {
  /** Fact ID */
  factId: string;

  /** The claim text */
  claim: string;

  /** Domain of the similar fact */
  domain: string;

  /** Stratum of the similar fact */
  stratum: OntologicalStratum;

  /** Confidence of the similar fact */
  confidence: number;

  /** Cosine similarity score (0-1) */
  similarityScore: number;
}

/**
 * Statistics about related statements.
 */
export interface RelationStatistics {
  /** Total number of related facts */
  totalRelated: number;

  /** Total supporting weight */
  supportingWeight: number;

  /** Total contradicting weight */
  contradictingWeight: number;

  /** Number of semantic matches */
  semanticMatches: number;

  /** Average similarity score */
  averageSimilarity: number;
}

// ============================================================================
// Query Types
// ============================================================================

/**
 * Query for the knowledge archive.
 */
export interface ArchiveQuery {
  /** Type of query */
  type: ArchiveQueryType;

  /** Filters to apply */
  filters: ArchiveFilters;

  /** Pagination settings */
  pagination: PaginationParams;

  /** Additional options */
  options: QueryOptions;
}

/**
 * Types of archive queries.
 */
export type ArchiveQueryType =
  | "domain" // Query by domain
  | "stratum" // Query by stratum level
  | "confidence" // Query by confidence range
  | "truth" // Truth evaluation
  | "related" // Related statements
  | "semantic"; // Semantic similarity search

/**
 * Filters for archive queries.
 */
export interface ArchiveFilters {
  /** Domain ID or pattern (e.g., 'physics.*') */
  domain?: string;

  /** Stratum level */
  stratum?: OntologicalStratum;

  /** Minimum confidence */
  minConfidence?: number;

  /** Maximum confidence */
  maxConfidence?: number;

  /** Fact status filter */
  status?: FactStatus;

  /** Reference types to include */
  referenceTypes?: ReferenceType[];

  /** Only facts verified after this block */
  verifiedAfter?: bigint;

  /** Only facts verified before this block */
  verifiedBefore?: bigint;

  /** Minimum verifier count */
  minVerifiers?: number;

  /** Search text (for semantic queries) */
  searchText?: string;

  /** Specific fact ID (for truth/related queries) */
  factId?: string;
}

/**
 * Pagination parameters.
 */
export interface PaginationParams {
  /** Maximum results to return */
  limit: number;

  /** Cursor for continuation */
  cursor?: string;

  /** Sort field */
  sortBy: "confidence" | "verifiedAt" | "citationCount" | "similarity";

  /** Sort direction */
  sortOrder: "asc" | "desc";
}

/**
 * Additional query options.
 */
export interface QueryOptions {
  /** Include reference details */
  includeReferences: boolean;

  /** Include full ontology context */
  includeOntology: boolean;

  /** Include semantic similarity matches */
  includeSimilarity: boolean;

  /** Max depth for reference traversal */
  maxDepth?: number;

  /** Max similarity results */
  maxSimilarityResults?: number;
}

// ============================================================================
// Batch Query Types
// ============================================================================

/**
 * Batch query for multiple archive queries at once.
 */
export interface BatchArchiveQuery {
  /** Individual queries */
  queries: ArchiveQuery[];

  /** Batch options */
  options: BatchOptions;
}

/**
 * Options for batch query execution.
 */
export interface BatchOptions {
  /** Execute queries in parallel */
  parallel: boolean;

  /** Maximum parallelism */
  maxParallelism: number;

  /** Timeout per query (ms) */
  timeout: number;

  /** Stream results as they complete */
  streaming: boolean;
}

/**
 * Result of batch query.
 */
export interface BatchQueryResult {
  /** Results keyed by query index */
  results: Map<number, ArchiveQueryResult<unknown>>;

  /** Overall execution time (ms) */
  totalTime: number;

  /** Any errors encountered */
  errors: Map<number, ArchiveError>;
}

// ============================================================================
// Response Types
// ============================================================================

/**
 * Generic archive query result.
 */
export interface ArchiveQueryResult<T> {
  /** Whether the query succeeded */
  success: boolean;

  /** Error if failed */
  error?: ArchiveError;

  /** Query results */
  items: T[];

  /** Pagination info */
  pagination: {
    /** Cursor for next page */
    cursor?: string;
    /** Whether more results exist */
    hasMore: boolean;
    /** Total estimated results */
    totalEstimate: number;
  };

  /** Query metadata */
  metadata: QueryMetadata;
}

/**
 * Metadata about query execution.
 */
export interface QueryMetadata {
  /** Unique query ID */
  queryId: string;

  /** Execution time (ms) */
  executionTime: number;

  /** Archive's current block number */
  archiveBlockNumber: bigint;

  /** Whether result came from cache */
  cacheHit: boolean;

  /** Index(es) used */
  indexesUsed: string[];
}

/**
 * Archive error.
 */
export interface ArchiveError {
  /** Error code */
  code: ArchiveErrorCode;

  /** Error message */
  message: string;

  /** Additional details */
  details?: Record<string, unknown>;
}

/**
 * Error codes for archive operations.
 */
export type ArchiveErrorCode =
  | "FACT_NOT_FOUND"
  | "DOMAIN_NOT_FOUND"
  | "INVALID_QUERY"
  | "TIMEOUT"
  | "INDEX_ERROR"
  | "EMBEDDING_ERROR"
  | "SYNC_LAG"
  | "INTERNAL_ERROR";

// ============================================================================
// Streaming Types
// ============================================================================

/**
 * Streaming response for large queries.
 */
export interface StreamingArchiveResponse<T> {
  /** Async iterator of results */
  items: AsyncIterableIterator<T>;

  /** Stream metadata */
  metadata: StreamMetadata;

  /** Cancel the stream */
  cancel: () => void;
}

/**
 * Metadata for streaming responses.
 */
export interface StreamMetadata {
  /** Unique stream ID */
  streamId: string;

  /** Estimated total items */
  totalEstimate: number;

  /** Batch size for streaming */
  batchSize: number;

  /** Archive block when stream started */
  startBlock: bigint;
}

// ============================================================================
// Index Types
// ============================================================================

/**
 * Domain index entry.
 */
export interface DomainIndexEntry {
  /** Domain ID */
  domainId: string;

  /** Domain data */
  domain: OntologicalDomain;

  /** Fact count in this domain */
  factCount: number;

  /** Average confidence */
  avgConfidence: number;

  /** Child domain IDs */
  childDomains: string[];

  /** Last indexed block */
  lastIndexed: bigint;
}

/**
 * Reference graph node.
 */
export interface ReferenceGraphNode {
  /** Fact ID */
  factId: string;

  /** Outgoing references */
  outgoing: Map<ReferenceType, string[]>;

  /** Incoming references */
  incoming: Map<ReferenceType, string[]>;

  /** Total in-degree */
  inDegree: number;

  /** Total out-degree */
  outDegree: number;
}

/**
 * Confidence bucket entry.
 */
export interface ConfidenceBucketEntry {
  /** Bucket index (0-99 for 0.01 granularity) */
  bucket: number;

  /** Fact IDs in this bucket */
  factIds: string[];

  /** Lower bound of bucket */
  lowerBound: number;

  /** Upper bound of bucket */
  upperBound: number;
}

// ============================================================================
// Archive State Types
// ============================================================================

/**
 * Archive sync status.
 */
export interface ArchiveSyncStatus {
  /** Current archive block */
  currentBlock: bigint;

  /** Latest chain block */
  latestBlock: bigint;

  /** Sync lag in blocks */
  lag: bigint;

  /** Whether sync is complete */
  synced: boolean;

  /** Last sync timestamp */
  lastSyncTime: number;

  /** Sync progress (0-1) */
  progress: number;
}

/**
 * Archive statistics.
 */
export interface ArchiveStatistics {
  /** Total facts indexed */
  totalFacts: number;

  /** Facts by stratum */
  factsByStratum: Record<OntologicalStratum, number>;

  /** Facts by status */
  factsByStatus: Record<FactStatus, number>;

  /** Total domains */
  totalDomains: number;

  /** Total references indexed */
  totalReferences: number;

  /** References by type */
  referencesByType: Record<ReferenceType, number>;

  /** Index sizes */
  indexSizes: {
    domain: number;
    confidence: number;
    reference: number;
    semantic: number;
  };

  /** Last updated */
  updatedAt: number;
}
