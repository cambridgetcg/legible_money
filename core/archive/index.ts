/**
 * Knowledge Archive Module
 *
 * Archive node for the Ontological Tree of Knowledge.
 * Enables AI agents to query facts by domain/stratum with confidence ratings,
 * perform truth evaluations, navigate reference graphs, and find semantically
 * related statements.
 */

// Types
export type {
  // Archive entry types
  ArchiveFactEntry,
  ArchiveReference,
  ArchiveMetadata,

  // Truth evaluation types
  TruthEvaluation,
  TruthVerdict,
  ConfidenceRatio,
  SupportingFact,
  ContradictingFact,
  OntologicalContext,

  // Related statements types
  RelatedStatements,
  RelatedFact,
  SemanticMatch,
  RelationStatistics,

  // Query types
  ArchiveQuery,
  ArchiveQueryType,
  ArchiveFilters,
  PaginationParams,
  QueryOptions,
  BatchArchiveQuery,
  BatchOptions,
  BatchQueryResult,

  // Response types
  ArchiveQueryResult,
  QueryMetadata,
  ArchiveError,
  ArchiveErrorCode,

  // Streaming types
  StreamingArchiveResponse,
  StreamMetadata,

  // Index types
  DomainIndexEntry,
  ReferenceGraphNode,
  ConfidenceBucketEntry,

  // State types
  ArchiveSyncStatus,
  ArchiveStatistics,
} from "./types.js";

// Configuration
export {
  DEFAULT_ARCHIVE_CONFIG,
  ARCHIVE_NAMESPACES,
  INDEX_PREFIXES,
  DEFAULT_PAGINATION,
  DEFAULT_QUERY_OPTIONS,
  DEFAULT_BATCH_OPTIONS,
  confidenceToBucket,
  getBucketRange,
  getBucketsInRange,
  makeDomainKey,
  makeDomainFactsKey,
  makeConfidenceKey,
  makeRefOutKey,
  makeRefInKey,
  makeEmbeddingKey,
  type ArchiveConfig,
} from "./config.js";

// Indexes
export {
  DomainHierarchyIndex,
  ConfidenceRankingIndex,
  ReferenceGraphIndex,
  SemanticSimilarityIndex,
  StubEmbedder,
  type DomainQueryOptions,
  type DomainPatternMatch,
  type ConfidenceQueryOptions,
  type ConfidenceIndexEntry,
  type ReferenceQueryOptions,
  type ReferenceTraversalResult,
  type StoredReference,
  type SemanticQueryOptions,
  type EmbeddingEntry,
  type EmbeddingResult,
  type Embedder,
} from "./indexes/index.js";

// Main archive class
export { KnowledgeArchive } from "./knowledge_archive.js";

// API
export {
  KnowledgeArchiveAPI,
  type ArchiveSession,
  type ArchiveQueryRecord,
  type ArchiveResponse,
  type ArchiveIntent,
  type EvaluateTruthIntent,
  type GetRelatedIntent,
  type FindSimilarIntent,
  type QueryDomainIntent,
  type QueryStratumIntent,
  type SearchIntent,
} from "./archive_api.js";

// RPC handlers
export {
  ArchiveRPCHandler,
  ARCHIVE_RPC_METHODS,
  RPC_ERRORS,
  type RPCRequest,
  type RPCResponse,
  type RPCError,
  type CreateSessionParams,
  type EvaluateTruthParams,
  type GetRelatedParams,
  type FindSimilarParams,
  type QueryDomainParams,
  type QueryStratumParams,
  type SearchParams,
  type BatchQueryParams,
  type GetFactParams,
} from "./rpc_handlers.js";
