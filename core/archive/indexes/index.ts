/**
 * Archive Indexes
 *
 * Exports all specialized indexes for the Knowledge Archive.
 */

export {
  DomainHierarchyIndex,
  type DomainQueryOptions,
  type DomainPatternMatch,
} from "./domain_index.js";

export {
  ConfidenceRankingIndex,
  type ConfidenceQueryOptions,
  type ConfidenceIndexEntry,
} from "./confidence_index.js";

export {
  ReferenceGraphIndex,
  type ReferenceQueryOptions,
  type ReferenceTraversalResult,
  type StoredReference,
} from "./reference_graph.js";

export {
  SemanticSimilarityIndex,
  StubEmbedder,
  type SemanticQueryOptions,
  type EmbeddingEntry,
  type EmbeddingResult,
  type Embedder,
} from "./semantic_index.js";
