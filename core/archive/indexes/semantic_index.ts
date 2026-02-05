/**
 * Semantic Similarity Index
 *
 * Index for embedding-based semantic search. Stores fact embeddings and
 * enables approximate nearest neighbor queries for finding similar facts.
 */

import type { SemanticMatch } from "../types.js";
import type { OntologicalStratum } from "../../knowledge/ontology/types.js";
import type { FactState } from "../../../state/types.js";
import {
  makeEmbeddingKey,
  // INDEX_PREFIXES,
  DEFAULT_ARCHIVE_CONFIG,
  type ArchiveConfig,
} from "../config.js";

// ============================================================================
// Semantic Index Types
// ============================================================================

/**
 * Options for semantic search queries.
 */
export interface SemanticQueryOptions {
  /** Domain filter */
  domain?: string;

  /** Stratum filter */
  stratum?: OntologicalStratum;

  /** Minimum similarity score */
  minSimilarity?: number;

  /** Maximum results */
  limit?: number;

  /** Exclude specific fact IDs */
  excludeIds?: string[];
}

/**
 * Stored embedding entry.
 */
export interface EmbeddingEntry {
  /** Fact ID */
  factId: string;

  /** Embedding vector */
  embedding: Float32Array;

  /** Domain ID */
  domainId: string;

  /** Stratum */
  stratum: OntologicalStratum;

  /** Claim text (for result display) */
  claim: string;

  /** Fact confidence */
  confidence: number;

  /** Block when indexed */
  indexedAt: bigint;
}

/**
 * Result of embedding generation (stub for external embedder).
 */
export interface EmbeddingResult {
  /** Generated embedding */
  embedding: Float32Array;

  /** Model used */
  model: string;

  /** Dimensions */
  dimensions: number;
}

// ============================================================================
// Embedder Interface
// ============================================================================

/**
 * Interface for embedding providers.
 * In production, this would connect to an embedding service.
 */
export interface Embedder {
  /**
   * Generate embedding for text.
   */
  embed(text: string): Promise<EmbeddingResult>;

  /**
   * Batch embed multiple texts.
   */
  embedBatch(texts: string[]): Promise<EmbeddingResult[]>;

  /**
   * Get embedding dimensions.
   */
  getDimensions(): number;
}

/**
 * Stub embedder for development/testing.
 * Returns random embeddings with correct dimensions.
 */
export class StubEmbedder implements Embedder {
  private dimensions: number;

  constructor(dimensions: number = 768) {
    this.dimensions = dimensions;
  }

  async embed(text: string): Promise<EmbeddingResult> {
    // Generate deterministic pseudo-random embedding based on text hash
    const embedding = new Float32Array(this.dimensions);
    let hash = this.hashString(text);

    for (let i = 0; i < this.dimensions; i++) {
      // Use hash to generate pseudo-random values
      hash = (hash * 1103515245 + 12345) & 0x7fffffff;
      embedding[i] = (hash / 0x7fffffff) * 2 - 1; // Range [-1, 1]
    }

    // Normalize to unit vector
    this.normalize(embedding);

    return {
      embedding,
      model: "stub",
      dimensions: this.dimensions,
    };
  }

  async embedBatch(texts: string[]): Promise<EmbeddingResult[]> {
    return Promise.all(texts.map((t) => this.embed(t)));
  }

  getDimensions(): number {
    return this.dimensions;
  }

  private hashString(str: string): number {
    let hash = 0;
    for (let i = 0; i < str.length; i++) {
      const char = str.charCodeAt(i);
      hash = (hash << 5) - hash + char;
      hash = hash & hash;
    }
    return Math.abs(hash);
  }

  private normalize(vec: Float32Array): void {
    let norm = 0;
    for (let i = 0; i < vec.length; i++) {
      norm += vec[i] * vec[i];
    }
    norm = Math.sqrt(norm);
    if (norm > 0) {
      for (let i = 0; i < vec.length; i++) {
        vec[i] /= norm;
      }
    }
  }
}

// ============================================================================
// Semantic Similarity Index
// ============================================================================

/**
 * Index for semantic similarity search.
 *
 * Key structures:
 * - emb:<fact_id> → EmbeddingEntry
 * - emb_domain:<domain>:<fact_id> → factId
 *
 * Uses brute-force cosine similarity for simplicity.
 * Production would use ANN (Approximate Nearest Neighbor) algorithms.
 */
export class SemanticSimilarityIndex {
  private config: ArchiveConfig;
  private embedder: Embedder;

  // In-memory index (would be backed by vector DB in production)
  private entries: Map<string, EmbeddingEntry> = new Map();
  private domainIndex: Map<string, Set<string>> = new Map(); // domain → factIds

  constructor(
    config: ArchiveConfig = DEFAULT_ARCHIVE_CONFIG,
    embedder?: Embedder,
  ) {
    this.config = config;
    this.embedder = embedder || new StubEmbedder(config.embeddingDimensions);
  }

  // ============================================================================
  // Indexing Operations
  // ============================================================================

  /**
   * Index a fact with its embedding.
   */
  async indexFact(
    fact: FactState,
    claim: string,
    domainId: string,
    stratum: OntologicalStratum,
    blockNumber: bigint,
  ): Promise<void> {
    // Generate embedding
    const result = await this.embedder.embed(claim);

    const entry: EmbeddingEntry = {
      factId: fact.id,
      embedding: result.embedding,
      domainId,
      stratum,
      claim,
      confidence: fact.confidence,
      indexedAt: blockNumber,
    };

    this.entries.set(fact.id, entry);

    // Add to domain index
    let domainSet = this.domainIndex.get(domainId);
    if (!domainSet) {
      domainSet = new Set();
      this.domainIndex.set(domainId, domainSet);
    }
    domainSet.add(fact.id);
  }

  /**
   * Index a fact with a pre-computed embedding.
   */
  indexFactWithEmbedding(
    fact: FactState,
    claim: string,
    embedding: Float32Array,
    domainId: string,
    stratum: OntologicalStratum,
    blockNumber: bigint,
  ): void {
    const entry: EmbeddingEntry = {
      factId: fact.id,
      embedding,
      domainId,
      stratum,
      claim,
      confidence: fact.confidence,
      indexedAt: blockNumber,
    };

    this.entries.set(fact.id, entry);

    // Add to domain index
    let domainSet = this.domainIndex.get(domainId);
    if (!domainSet) {
      domainSet = new Set();
      this.domainIndex.set(domainId, domainSet);
    }
    domainSet.add(fact.id);
  }

  /**
   * Remove a fact from the index.
   */
  removeFact(factId: string): void {
    const entry = this.entries.get(factId);
    if (entry) {
      this.domainIndex.get(entry.domainId)?.delete(factId);
      this.entries.delete(factId);
    }
  }

  // ============================================================================
  // Query Operations
  // ============================================================================

  /**
   * Find facts similar to a given fact.
   */
  findSimilar(
    factId: string,
    options: SemanticQueryOptions = {},
  ): SemanticMatch[] {
    const entry = this.entries.get(factId);
    if (!entry) return [];

    return this.findSimilarByEmbedding(entry.embedding, {
      ...options,
      excludeIds: [...(options.excludeIds || []), factId],
    });
  }

  /**
   * Find facts similar to query text.
   */
  async searchByText(
    queryText: string,
    options: SemanticQueryOptions = {},
  ): Promise<SemanticMatch[]> {
    const result = await this.embedder.embed(queryText);
    return this.findSimilarByEmbedding(result.embedding, options);
  }

  /**
   * Find facts similar to a given embedding.
   */
  findSimilarByEmbedding(
    queryEmbedding: Float32Array,
    options: SemanticQueryOptions = {},
  ): SemanticMatch[] {
    const minSimilarity =
      options.minSimilarity ?? this.config.similarityThreshold;
    const limit = options.limit ?? this.config.maxSimilarityResults;
    const excludeIds = new Set(options.excludeIds || []);

    const matches: SemanticMatch[] = [];

    // Get candidate entries
    let candidates: EmbeddingEntry[];
    if (options.domain) {
      const domainFactIds = this.domainIndex.get(options.domain);
      if (!domainFactIds) return [];
      candidates = Array.from(domainFactIds)
        .map((id) => this.entries.get(id)!)
        .filter((e) => e !== undefined);
    } else {
      candidates = Array.from(this.entries.values());
    }

    // Filter by stratum if specified
    if (options.stratum) {
      candidates = candidates.filter((e) => e.stratum === options.stratum);
    }

    // Calculate similarities
    for (const entry of candidates) {
      if (excludeIds.has(entry.factId)) continue;

      const similarity = this.cosineSimilarity(queryEmbedding, entry.embedding);

      if (similarity >= minSimilarity) {
        matches.push({
          factId: entry.factId,
          claim: entry.claim,
          domain: entry.domainId,
          stratum: entry.stratum,
          confidence: entry.confidence,
          similarityScore: similarity,
        });
      }
    }

    // Sort by similarity (descending) and limit
    matches.sort((a, b) => b.similarityScore - a.similarityScore);

    if (matches.length > limit) {
      return matches.slice(0, limit);
    }

    return matches;
  }

  /**
   * Find similar facts within a domain.
   */
  findSimilarInDomain(
    factId: string,
    domainId: string,
    options: SemanticQueryOptions = {},
  ): SemanticMatch[] {
    return this.findSimilar(factId, {
      ...options,
      domain: domainId,
    });
  }

  /**
   * Get the embedding for a fact.
   */
  getEmbedding(factId: string): Float32Array | undefined {
    return this.entries.get(factId)?.embedding;
  }

  /**
   * Get the entry for a fact.
   */
  getEntry(factId: string): EmbeddingEntry | undefined {
    return this.entries.get(factId);
  }

  // ============================================================================
  // Similarity Calculations
  // ============================================================================

  /**
   * Calculate cosine similarity between two vectors.
   */
  private cosineSimilarity(a: Float32Array, b: Float32Array): number {
    if (a.length !== b.length) return 0;

    let dotProduct = 0;
    let normA = 0;
    let normB = 0;

    for (let i = 0; i < a.length; i++) {
      dotProduct += a[i] * b[i];
      normA += a[i] * a[i];
      normB += b[i] * b[i];
    }

    const denominator = Math.sqrt(normA) * Math.sqrt(normB);
    return denominator > 0 ? dotProduct / denominator : 0;
  }

  /**
   * Calculate Euclidean distance between two vectors.
   */
  // @ts-expect-error Reserved alternative distance metric
  private euclideanDistance(a: Float32Array, b: Float32Array): number {
    if (a.length !== b.length) return Infinity;

    let sum = 0;
    for (let i = 0; i < a.length; i++) {
      const diff = a[i] - b[i];
      sum += diff * diff;
    }
    return Math.sqrt(sum);
  }

  // ============================================================================
  // Statistics
  // ============================================================================

  /**
   * Get index statistics.
   */
  getStatistics(): {
    totalEmbeddings: number;
    embeddingDimensions: number;
    domainsIndexed: number;
    avgEmbeddingsPerDomain: number;
  } {
    let totalEmbeddingsInDomains = 0;
    for (const factSet of this.domainIndex.values()) {
      totalEmbeddingsInDomains += factSet.size;
    }

    return {
      totalEmbeddings: this.entries.size,
      embeddingDimensions: this.config.embeddingDimensions,
      domainsIndexed: this.domainIndex.size,
      avgEmbeddingsPerDomain:
        this.domainIndex.size > 0
          ? totalEmbeddingsInDomains / this.domainIndex.size
          : 0,
    };
  }

  // ============================================================================
  // Embedder Management
  // ============================================================================

  /**
   * Set the embedder to use.
   */
  setEmbedder(embedder: Embedder): void {
    this.embedder = embedder;
  }

  /**
   * Get the current embedder.
   */
  getEmbedder(): Embedder {
    return this.embedder;
  }

  // ============================================================================
  // Serialization
  // ============================================================================

  /**
   * Generate storage key for embedding.
   */
  makeStorageKey(factId: string): string {
    return makeEmbeddingKey(factId);
  }

  /**
   * Serialize embedding to buffer.
   */
  serializeEmbedding(embedding: Float32Array): Buffer {
    return Buffer.from(embedding.buffer);
  }

  /**
   * Deserialize embedding from buffer.
   */
  deserializeEmbedding(buffer: Buffer): Float32Array {
    return new Float32Array(
      buffer.buffer,
      buffer.byteOffset,
      buffer.length / 4,
    );
  }

  /**
   * Clear the index (for testing).
   */
  clear(): void {
    this.entries.clear();
    this.domainIndex.clear();
  }
}
