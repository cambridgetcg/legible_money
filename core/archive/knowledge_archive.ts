/**
 * Knowledge Archive
 *
 * Main archive service for AI agents to query the Ontological Tree of Knowledge.
 * Provides truth evaluation, relationship navigation, semantic search, and
 * confidence-based queries.
 */

import type {
  ArchiveFactEntry,
  ArchiveReference,
  TruthEvaluation,
  TruthVerdict,
  ConfidenceRatio,
  SupportingFact,
  ContradictingFact,
  OntologicalContext,
  RelatedStatements,
  RelatedFact,
  SemanticMatch,
  RelationStatistics,
  ArchiveQuery,
  ArchiveQueryResult,
  ArchiveFilters,
  PaginationParams,
  QueryOptions,
  QueryMetadata,
  BatchArchiveQuery,
  BatchQueryResult,
  StreamingArchiveResponse,
  ArchiveError,
  ArchiveErrorCode,
  ArchiveSyncStatus,
  ArchiveStatistics,
} from "./types.js";
import type { FactState, ReferenceType } from "../../state/types.js";
import type {
  OntologicalDomain,
  OntologicalStratum,
  DomainTree,
  StratumDependency,
} from "../knowledge/ontology/types.js";
import { STRATUM_CONFIDENCE_CEILING } from "../knowledge/ontology/types.js";

/**
 * Cross-stratum discount: 5% confidence reduction per stratum boundary.
 */
const CROSS_STRATUM_DISCOUNT = 0.05;
import {
  DomainHierarchyIndex,
  ConfidenceRankingIndex,
  ReferenceGraphIndex,
  SemanticSimilarityIndex,
  type DomainQueryOptions,
  type ConfidenceQueryOptions,
  type SemanticQueryOptions,
  type Embedder,
} from "./indexes/index.js";
import {
  DEFAULT_ARCHIVE_CONFIG,
  DEFAULT_PAGINATION,
  DEFAULT_QUERY_OPTIONS,
  DEFAULT_BATCH_OPTIONS,
  type ArchiveConfig,
} from "./config.js";

// ============================================================================
// Knowledge Archive
// ============================================================================

/**
 * Knowledge Archive - Main service for AI agent queries.
 *
 * Features:
 * - Domain and stratum-based fact queries
 * - Truth evaluation with confidence ratios
 * - Related statement navigation
 * - Semantic similarity search
 * - Batch and streaming query support
 */
export class KnowledgeArchive {
  private config: ArchiveConfig;

  // Indexes
  private domainIndex: DomainHierarchyIndex;
  private confidenceIndex: ConfidenceRankingIndex;
  private referenceIndex: ReferenceGraphIndex;
  private semanticIndex: SemanticSimilarityIndex;

  // In-memory fact storage (would be backed by LevelDB in production)
  private facts: Map<string, ArchiveFactEntry> = new Map();

  // Sync status
  private currentBlock: bigint = 0n;
  private lastSyncTime: number = 0;

  constructor(
    config: ArchiveConfig = DEFAULT_ARCHIVE_CONFIG,
    embedder?: Embedder,
  ) {
    this.config = config;
    this.domainIndex = new DomainHierarchyIndex(config);
    this.confidenceIndex = new ConfidenceRankingIndex(config);
    this.referenceIndex = new ReferenceGraphIndex(config);
    this.semanticIndex = new SemanticSimilarityIndex(config, embedder);
  }

  // ============================================================================
  // Initialization
  // ============================================================================

  /**
   * Set the domain tree for ontological operations.
   */
  setDomainTree(tree: DomainTree): void {
    // Index all domains
    for (const [_domainId, entry] of tree.domainIndex) {
      this.domainIndex.indexDomain(entry.domain, this.currentBlock);
    }
  }

  /**
   * Index a fact into the archive.
   */
  async indexFact(
    fact: FactState,
    domain: OntologicalDomain,
    references: ArchiveReference[] = [],
    claim?: string,
  ): Promise<void> {
    const entry: ArchiveFactEntry = {
      fact,
      domain,
      stratum: domain.stratum,
      references,
      archiveMetadata: {
        indexedAt: this.currentBlock,
        lastUpdated: this.currentBlock,
        queryCount: 0,
        citationCount: 0,
      },
    };

    this.facts.set(fact.id, entry);

    // Index in all indexes
    this.domainIndex.indexFact(fact, domain.id, this.currentBlock);
    this.confidenceIndex.indexFact(
      fact,
      domain.id,
      domain.stratum,
      this.currentBlock,
    );

    // Index references
    for (const ref of references) {
      this.referenceIndex.indexReference(ref, this.currentBlock);
    }

    // Index embedding if claim text available
    if (claim) {
      await this.semanticIndex.indexFact(
        fact,
        claim,
        domain.id,
        domain.stratum,
        this.currentBlock,
      );
    }
  }

  /**
   * Update the current block number.
   */
  setBlockNumber(blockNumber: bigint): void {
    this.currentBlock = blockNumber;
    this.lastSyncTime = Date.now();
  }

  // ============================================================================
  // Domain Queries
  // ============================================================================

  /**
   * Query facts by domain.
   */
  async queryByDomain(
    domain: string,
    options: DomainQueryOptions & {
      pagination?: Partial<PaginationParams>;
    } = {},
  ): Promise<ArchiveQueryResult<ArchiveFactEntry>> {
    const startTime = Date.now();
    const queryId = this.generateQueryId();

    try {
      // Match domain pattern
      const matches = this.domainIndex.matchDomainPattern(domain);
      if (matches.length === 0) {
        return this.createErrorResult(
          queryId,
          startTime,
          "DOMAIN_NOT_FOUND",
          `Domain not found: ${domain}`,
        );
      }

      // Get fact IDs from matched domains
      const factIds: Set<string> = new Set();
      for (const match of matches) {
        const domainFactIds = this.domainIndex.getFactsInDomain(
          match.domainId,
          options,
        );
        domainFactIds.forEach((id) => factIds.add(id));
      }

      // Apply confidence filter
      let filteredIds = Array.from(factIds);
      if (options.minConfidence !== undefined) {
        filteredIds = filteredIds.filter((id) => {
          const entry = this.facts.get(id);
          return entry && entry.fact.confidence >= options.minConfidence!;
        });
      }

      // Sort and paginate
      const pagination = { ...DEFAULT_PAGINATION, ...options.pagination };
      filteredIds = this.sortFactIds(
        filteredIds,
        pagination.sortBy,
        pagination.sortOrder,
      );

      const limit = Math.min(pagination.limit, filteredIds.length);
      const items = filteredIds
        .slice(0, limit)
        .map((id) => this.facts.get(id)!);

      return {
        success: true,
        items,
        pagination: {
          hasMore: filteredIds.length > limit,
          totalEstimate: filteredIds.length,
          cursor:
            items.length > 0 ? items[items.length - 1].fact.id : undefined,
        },
        metadata: this.createMetadata(queryId, startTime, ["domain"]),
      };
    } catch (error) {
      return this.createErrorResult(
        queryId,
        startTime,
        "INTERNAL_ERROR",
        String(error),
      );
    }
  }

  /**
   * Query facts by stratum.
   */
  async queryByStratum(
    stratum: OntologicalStratum,
    options: ConfidenceQueryOptions & {
      pagination?: Partial<PaginationParams>;
    } = {},
  ): Promise<ArchiveQueryResult<ArchiveFactEntry>> {
    const startTime = Date.now();
    const queryId = this.generateQueryId();

    try {
      const factIds = this.confidenceIndex.getByConfidenceRange({
        ...options,
        stratum,
      });

      const pagination = { ...DEFAULT_PAGINATION, ...options.pagination };
      const limit = Math.min(pagination.limit, factIds.length);
      const items = factIds
        .slice(0, limit)
        .map((id) => this.facts.get(id)!)
        .filter(Boolean);

      return {
        success: true,
        items,
        pagination: {
          hasMore: factIds.length > limit,
          totalEstimate: factIds.length,
        },
        metadata: this.createMetadata(queryId, startTime, [
          "confidence",
          "stratum",
        ]),
      };
    } catch (error) {
      return this.createErrorResult(
        queryId,
        startTime,
        "INTERNAL_ERROR",
        String(error),
      );
    }
  }

  // ============================================================================
  // Truth Evaluation
  // ============================================================================

  /**
   * Evaluate the truth of a fact.
   *
   * Returns a verdict (true/false/uncertain) with confidence ratio
   * based on supporting and contradicting evidence.
   */
  async evaluateTruth(factId: string): Promise<TruthEvaluation> {
    const entry = this.facts.get(factId);
    if (!entry) {
      throw new Error(`Fact not found: ${factId}`);
    }

    // Build confidence map for ratio calculation
    const factConfidences = new Map<string, number>();
    for (const [id, e] of this.facts) {
      factConfidences.set(id, e.fact.confidence);
    }

    // Calculate confidence ratio from supporting/contradicting facts
    const confidenceRatio = this.referenceIndex.calculateConfidenceRatio(
      factId,
      factConfidences,
    );

    // Get supporting facts
    const supportRefs = this.referenceIndex.getIncoming(factId, {
      referenceTypes: ["supports", "extends"],
    });
    const supportingFacts: SupportingFact[] = supportRefs.map((ref) => {
      const supportEntry = this.facts.get(ref.sourceId);
      return {
        factId: ref.sourceId,
        claim: supportEntry?.fact.id ?? "Unknown",
        confidence: supportEntry?.fact.confidence ?? 0,
        referenceType: ref.referenceType as "supports" | "extends",
        strength: ref.strength,
        domain: supportEntry?.domain.id ?? "unknown",
      };
    });

    // Get contradicting facts
    const contradictRefs = this.referenceIndex.getIncoming(factId, {
      referenceTypes: ["contradicts"],
    });
    const contradictingFacts: ContradictingFact[] = contradictRefs.map(
      (ref) => {
        const contradictEntry = this.facts.get(ref.sourceId);
        return {
          factId: ref.sourceId,
          claim: contradictEntry?.fact.id ?? "Unknown",
          confidence: contradictEntry?.fact.confidence ?? 0,
          contradictionType: "direct" as const,
          strength: ref.strength,
          domain: contradictEntry?.domain.id ?? "unknown",
        };
      },
    );

    // Build ontological context
    const ontologicalContext = this.buildOntologicalContext(entry);

    // Apply stratum ceiling to confidence
    const stratumCeiling = STRATUM_CONFIDENCE_CEILING[entry.stratum];
    const effectiveConfidence = Math.min(entry.fact.confidence, stratumCeiling);

    // Calculate final confidence with ratio adjustment
    const adjustedConfidence = this.calculateAdjustedConfidence(
      effectiveConfidence,
      confidenceRatio,
      ontologicalContext.crossStratumDiscount,
    );

    // Determine verdict
    const verdict = this.determineVerdict(adjustedConfidence, confidenceRatio);

    // Build reasoning
    const reasoning = this.buildReasoning(
      verdict,
      adjustedConfidence,
      confidenceRatio,
      supportingFacts.length,
      contradictingFacts.length,
      entry.stratum,
    );

    // Update query count
    entry.archiveMetadata.queryCount++;

    return {
      factId,
      claim: entry.fact.id, // Would be actual claim text in production
      verdict,
      confidence: adjustedConfidence,
      confidenceRatio,
      supportingFacts,
      contradictingFacts,
      ontologicalContext,
      reasoning,
    };
  }

  /**
   * Determine truth verdict based on confidence and ratio.
   */
  private determineVerdict(
    confidence: number,
    ratio: ConfidenceRatio,
  ): TruthVerdict {
    // If contradicting evidence dominates
    if (ratio.ratio < 0.4 && ratio.contradicting > 0.3) {
      return "false";
    }

    // If supporting evidence dominates and confidence is high
    if (
      confidence >= this.config.trueThreshold &&
      ratio.ratio >= this.config.minSupportRatio
    ) {
      return "true";
    }

    // If confidence is low
    if (confidence <= this.config.falseThreshold) {
      return "false";
    }

    return "uncertain";
  }

  /**
   * Calculate adjusted confidence with ratio and discount.
   */
  private calculateAdjustedConfidence(
    baseConfidence: number,
    ratio: ConfidenceRatio,
    crossStratumDiscount: number,
  ): number {
    // Apply ratio adjustment (supporting vs contradicting)
    const ratioAdjusted = baseConfidence * ratio.ratio;

    // Apply cross-stratum discount
    const discounted = ratioAdjusted * (1 - crossStratumDiscount);

    return Math.max(0, Math.min(1, discounted));
  }

  /**
   * Build ontological context for a fact.
   */
  private buildOntologicalContext(entry: ArchiveFactEntry): OntologicalContext {
    const domainPath = this.domainIndex.getDomainPath(entry.domain.id);
    const crossStratumDependencies: StratumDependency[] =
      entry.fact.stratumDependencies ?? [];

    // Calculate cross-stratum discount
    let crossStratumDiscount = 0;
    for (const dep of crossStratumDependencies) {
      crossStratumDiscount +=
        CROSS_STRATUM_DISCOUNT * (1 - dep.dependencyStrength);
    }
    crossStratumDiscount = Math.min(crossStratumDiscount, 0.5); // Cap at 50%

    return {
      stratum: entry.stratum,
      stratumCeiling: STRATUM_CONFIDENCE_CEILING[entry.stratum],
      domainPath,
      crossStratumDependencies,
      crossStratumDiscount,
    };
  }

  /**
   * Build human-readable reasoning for the verdict.
   */
  private buildReasoning(
    verdict: TruthVerdict,
    confidence: number,
    ratio: ConfidenceRatio,
    supportCount: number,
    contradictCount: number,
    stratum: OntologicalStratum,
  ): string {
    const parts: string[] = [];

    parts.push(
      `Verdict: ${verdict.toUpperCase()} (confidence: ${(confidence * 100).toFixed(1)}%)`,
    );

    parts.push(`Evidence ratio: ${(ratio.ratio * 100).toFixed(1)}% supporting`);

    if (supportCount > 0) {
      parts.push(
        `${supportCount} supporting fact(s) with weight ${ratio.supporting.toFixed(3)}`,
      );
    }

    if (contradictCount > 0) {
      parts.push(
        `${contradictCount} contradicting fact(s) with weight ${ratio.contradicting.toFixed(3)}`,
      );
    }

    parts.push(
      `Stratum: ${stratum} (ceiling: ${(STRATUM_CONFIDENCE_CEILING[stratum] * 100).toFixed(0)}%)`,
    );

    return parts.join(". ");
  }

  // ============================================================================
  // Related Statements
  // ============================================================================

  /**
   * Get related statements for a fact.
   */
  async getRelatedStatements(
    factId: string,
    options: QueryOptions = DEFAULT_QUERY_OPTIONS,
  ): Promise<RelatedStatements> {
    const entry = this.facts.get(factId);
    if (!entry) {
      throw new Error(`Fact not found: ${factId}`);
    }

    // Get reference-based relationships
    const supports = this.getRelatedByType(factId, "supports");
    const contradicts = this.getRelatedByType(factId, "contradicts");
    const extends_ = this.getRelatedByType(factId, "extends");
    const specializes = this.getRelatedByType(factId, "specializes");
    const generalizes = this.getRelatedByType(factId, "generalizes");

    // Get semantic similarity matches
    let semanticallySimilar: SemanticMatch[] = [];
    if (options.includeSimilarity) {
      semanticallySimilar = this.semanticIndex.findSimilar(factId, {
        limit: options.maxSimilarityResults ?? 10,
      });
    }

    // Calculate statistics
    const statistics: RelationStatistics = {
      totalRelated:
        supports.length +
        contradicts.length +
        extends_.length +
        specializes.length +
        generalizes.length,
      supportingWeight: supports.reduce(
        (sum, f) => sum + f.referenceStrength,
        0,
      ),
      contradictingWeight: contradicts.reduce(
        (sum, f) => sum + f.referenceStrength,
        0,
      ),
      semanticMatches: semanticallySimilar.length,
      averageSimilarity:
        semanticallySimilar.length > 0
          ? semanticallySimilar.reduce((sum, m) => sum + m.similarityScore, 0) /
            semanticallySimilar.length
          : 0,
    };

    return {
      factId,
      claim: entry.fact.id,
      supports,
      contradicts,
      extends: extends_,
      specializes,
      generalizes,
      semanticallySimilar,
      statistics,
    };
  }

  /**
   * Get related facts by reference type.
   */
  private getRelatedByType(
    factId: string,
    refType: ReferenceType,
  ): RelatedFact[] {
    const refs = this.referenceIndex.getIncoming(factId, {
      referenceTypes: [refType],
    });

    return refs.map((ref) => {
      const entry = this.facts.get(ref.sourceId);
      return {
        factId: ref.sourceId,
        claim: entry?.fact.id ?? "Unknown",
        domain: entry?.domain.id ?? "unknown",
        stratum: entry?.stratum ?? "fundamental",
        confidence: entry?.fact.confidence ?? 0,
        referenceStrength: ref.strength,
      };
    });
  }

  // ============================================================================
  // Semantic Search
  // ============================================================================

  /**
   * Find semantically similar facts.
   */
  async findSimilar(
    factId: string,
    options: SemanticQueryOptions = {},
  ): Promise<SemanticMatch[]> {
    return this.semanticIndex.findSimilar(factId, {
      ...options,
      limit: options.limit ?? this.config.maxSimilarityResults,
      minSimilarity: options.minSimilarity ?? this.config.similarityThreshold,
    });
  }

  /**
   * Search by text query.
   */
  async search(
    query: string,
    options: SemanticQueryOptions = {},
  ): Promise<SemanticMatch[]> {
    return this.semanticIndex.searchByText(query, {
      ...options,
      limit: options.limit ?? this.config.maxSimilarityResults,
      minSimilarity: options.minSimilarity ?? this.config.similarityThreshold,
    });
  }

  // ============================================================================
  // Batch Operations
  // ============================================================================

  /**
   * Execute multiple queries in batch.
   */
  async batchQuery(batch: BatchArchiveQuery): Promise<BatchQueryResult> {
    const startTime = Date.now();
    const results = new Map<number, ArchiveQueryResult<unknown>>();
    const errors = new Map<number, ArchiveError>();

    const options = { ...DEFAULT_BATCH_OPTIONS, ...batch.options };

    if (options.parallel) {
      // Execute in parallel with concurrency limit
      const chunks = this.chunkArray(batch.queries, options.maxParallelism);

      for (const chunk of chunks) {
        const promises = chunk.map(async (query, _index) => {
          const queryIndex = batch.queries.indexOf(query);
          try {
            const result = await this.executeQuery(query);
            results.set(queryIndex, result);
          } catch (error) {
            errors.set(queryIndex, {
              code: "INTERNAL_ERROR",
              message: String(error),
            });
          }
        });

        await Promise.all(promises);
      }
    } else {
      // Execute sequentially
      for (let i = 0; i < batch.queries.length; i++) {
        try {
          const result = await this.executeQuery(batch.queries[i]);
          results.set(i, result);
        } catch (error) {
          errors.set(i, {
            code: "INTERNAL_ERROR",
            message: String(error),
          });
        }
      }
    }

    return {
      results,
      totalTime: Date.now() - startTime,
      errors,
    };
  }

  /**
   * Execute a single query.
   */
  private async executeQuery(
    query: ArchiveQuery,
  ): Promise<ArchiveQueryResult<unknown>> {
    switch (query.type) {
      case "domain":
        return this.queryByDomain(query.filters.domain ?? "", {
          minConfidence: query.filters.minConfidence,
          pagination: query.pagination,
        });

      case "stratum":
        return this.queryByStratum(query.filters.stratum!, {
          minConfidence: query.filters.minConfidence,
          maxConfidence: query.filters.maxConfidence,
          pagination: query.pagination,
        });

      case "confidence":
        return this.queryByConfidence(query.filters, query.pagination);

      case "truth": {
        if (!query.filters.factId) {
          throw new Error("factId required for truth query");
        }
        const evaluation = await this.evaluateTruth(query.filters.factId);
        return {
          success: true,
          items: [evaluation],
          pagination: { hasMore: false, totalEstimate: 1 },
          metadata: this.createMetadata(this.generateQueryId(), Date.now(), [
            "truth",
          ]),
        };
      }

      case "related": {
        if (!query.filters.factId) {
          throw new Error("factId required for related query");
        }
        const related = await this.getRelatedStatements(
          query.filters.factId,
          query.options,
        );
        return {
          success: true,
          items: [related],
          pagination: { hasMore: false, totalEstimate: 1 },
          metadata: this.createMetadata(this.generateQueryId(), Date.now(), [
            "reference",
          ]),
        };
      }

      case "semantic": {
        const matches = await this.search(query.filters.searchText ?? "", {
          domain: query.filters.domain,
          stratum: query.filters.stratum,
          limit: query.pagination.limit,
        });
        return {
          success: true,
          items: matches,
          pagination: { hasMore: false, totalEstimate: matches.length },
          metadata: this.createMetadata(this.generateQueryId(), Date.now(), [
            "semantic",
          ]),
        };
      }

      default:
        throw new Error(`Unknown query type: ${query.type}`);
    }
  }

  /**
   * Query by confidence range.
   */
  private async queryByConfidence(
    filters: ArchiveFilters,
    pagination: PaginationParams,
  ): Promise<ArchiveQueryResult<ArchiveFactEntry>> {
    const startTime = Date.now();
    const queryId = this.generateQueryId();

    const factIds = this.confidenceIndex.getByConfidenceRange({
      minConfidence: filters.minConfidence,
      maxConfidence: filters.maxConfidence,
      domain: filters.domain,
      stratum: filters.stratum,
      limit: pagination.limit,
      sortOrder: pagination.sortOrder,
    });

    const items = factIds.map((id) => this.facts.get(id)!).filter(Boolean);

    return {
      success: true,
      items,
      pagination: {
        hasMore: factIds.length >= pagination.limit,
        totalEstimate: factIds.length,
      },
      metadata: this.createMetadata(queryId, startTime, ["confidence"]),
    };
  }

  /**
   * Stream query results.
   */
  streamQuery(query: ArchiveQuery): StreamingArchiveResponse<ArchiveFactEntry> {
    const streamId = this.generateQueryId();
    let cancelled = false;

    const items = this.createStreamIterator(query, () => cancelled);

    return {
      items,
      metadata: {
        streamId,
        totalEstimate: this.facts.size,
        batchSize: this.config.streamChunkSize,
        startBlock: this.currentBlock,
      },
      cancel: () => {
        cancelled = true;
      },
    };
  }

  /**
   * Create async iterator for streaming.
   */
  private async *createStreamIterator(
    query: ArchiveQuery,
    isCancelled: () => boolean,
  ): AsyncIterableIterator<ArchiveFactEntry> {
    // Get all matching fact IDs
    let factIds: string[];

    if (query.filters.domain) {
      const matches = this.domainIndex.matchDomainPattern(query.filters.domain);
      factIds = [];
      for (const match of matches) {
        factIds.push(...this.domainIndex.getFactsInDomain(match.domainId, {}));
      }
    } else {
      factIds = Array.from(this.facts.keys());
    }

    // Apply filters
    if (query.filters.minConfidence !== undefined) {
      factIds = factIds.filter((id) => {
        const entry = this.facts.get(id);
        return entry && entry.fact.confidence >= query.filters.minConfidence!;
      });
    }

    // Yield in chunks
    for (let i = 0; i < factIds.length; i += this.config.streamChunkSize) {
      if (isCancelled()) break;

      const chunk = factIds.slice(i, i + this.config.streamChunkSize);
      for (const id of chunk) {
        const entry = this.facts.get(id);
        if (entry) {
          yield entry;
        }
      }
    }
  }

  // ============================================================================
  // Statistics and Status
  // ============================================================================

  /**
   * Get archive sync status.
   */
  getSyncStatus(): ArchiveSyncStatus {
    return {
      currentBlock: this.currentBlock,
      latestBlock: this.currentBlock, // Would be from chain in production
      lag: 0n,
      synced: true,
      lastSyncTime: this.lastSyncTime,
      progress: 1.0,
    };
  }

  /**
   * Get archive statistics.
   */
  getStatistics(): ArchiveStatistics {
    const factsByStratum: Record<OntologicalStratum, number> = {
      fundamental: 0,
      physical: 0,
      chemical: 0,
      biological: 0,
      cognitive: 0,
      social: 0,
      technological: 0,
    };

    const factsByStatus: Record<string, number> = {};
    for (const entry of this.facts.values()) {
      factsByStratum[entry.stratum]++;
      factsByStatus[entry.fact.status] =
        (factsByStatus[entry.fact.status] || 0) + 1;
    }

    const refStats = this.referenceIndex.getStatistics();
    const domainStats = this.domainIndex.getStatistics();
    const confStats = this.confidenceIndex.getStatistics();
    const semStats = this.semanticIndex.getStatistics();

    return {
      totalFacts: this.facts.size,
      factsByStratum,
      factsByStatus: factsByStatus as any,
      totalDomains: domainStats.totalDomains,
      totalReferences: refStats.totalReferences,
      referencesByType: refStats.referencesByType as any,
      indexSizes: {
        domain: domainStats.totalDomains,
        confidence: confStats.totalFacts,
        reference: refStats.totalReferences,
        semantic: semStats.totalEmbeddings,
      },
      updatedAt: this.lastSyncTime,
    };
  }

  // ============================================================================
  // Utility Methods
  // ============================================================================

  /**
   * Get a fact by ID.
   */
  getFact(factId: string): ArchiveFactEntry | undefined {
    return this.facts.get(factId);
  }

  /**
   * Generate a unique query ID.
   */
  private generateQueryId(): string {
    return `q_${Date.now()}_${Math.random().toString(36).substring(2, 8)}`;
  }

  /**
   * Create query metadata.
   */
  private createMetadata(
    queryId: string,
    startTime: number,
    indexesUsed: string[],
  ): QueryMetadata {
    return {
      queryId,
      executionTime: Date.now() - startTime,
      archiveBlockNumber: this.currentBlock,
      cacheHit: false,
      indexesUsed,
    };
  }

  /**
   * Create error result.
   */
  private createErrorResult<T>(
    queryId: string,
    startTime: number,
    code: ArchiveErrorCode,
    message: string,
  ): ArchiveQueryResult<T> {
    return {
      success: false,
      error: { code, message },
      items: [],
      pagination: { hasMore: false, totalEstimate: 0 },
      metadata: this.createMetadata(queryId, startTime, []),
    };
  }

  /**
   * Sort fact IDs by criteria.
   */
  private sortFactIds(
    factIds: string[],
    sortBy: string,
    sortOrder: "asc" | "desc",
  ): string[] {
    return factIds.sort((a, b) => {
      const entryA = this.facts.get(a);
      const entryB = this.facts.get(b);
      if (!entryA || !entryB) return 0;

      let comparison = 0;
      switch (sortBy) {
        case "confidence":
          comparison = entryA.fact.confidence - entryB.fact.confidence;
          break;
        case "verifiedAt":
          comparison = Number(entryA.fact.verifiedAt - entryB.fact.verifiedAt);
          break;
        case "citationCount":
          comparison =
            entryA.archiveMetadata.citationCount -
            entryB.archiveMetadata.citationCount;
          break;
        default:
          comparison = 0;
      }

      return sortOrder === "asc" ? comparison : -comparison;
    });
  }

  /**
   * Chunk array for parallel processing.
   */
  private chunkArray<T>(array: T[], chunkSize: number): T[][] {
    const chunks: T[][] = [];
    for (let i = 0; i < array.length; i += chunkSize) {
      chunks.push(array.slice(i, i + chunkSize));
    }
    return chunks;
  }

  /**
   * Clear all data (for testing).
   */
  clear(): void {
    this.facts.clear();
    this.domainIndex.clear();
    this.confidenceIndex.clear();
    this.referenceIndex.clear();
    this.semanticIndex.clear();
  }
}
