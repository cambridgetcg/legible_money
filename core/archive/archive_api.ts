/**
 * Knowledge Archive API
 *
 * API layer for AI agents to query the Knowledge Archive.
 * Provides intent-based queries aligned with the Agentic API pattern.
 */

import type { DID } from "../agents/identity/did.js";
import type { PPMetadata } from "../api/philosophy.js";
import { createPPMetadata } from "../api/philosophy.js";
import type {
  ArchiveFactEntry,
  TruthEvaluation,
  RelatedStatements,
  SemanticMatch,
  BatchArchiveQuery,
  BatchQueryResult,
  ArchiveSyncStatus,
  ArchiveStatistics,
} from "./types.js";
import type { OntologicalStratum } from "../knowledge/ontology/types.js";
import { KnowledgeArchive } from "./knowledge_archive.js";
import type { DomainQueryOptions } from "./indexes/domain_index.js";
import type { ConfidenceQueryOptions } from "./indexes/confidence_index.js";

// ============================================================================
// Archive API Types
// ============================================================================

/**
 * Archive session for tracking agent queries.
 */
export interface ArchiveSession {
  sessionId: string;
  agentDid: DID;
  queryHistory: ArchiveQueryRecord[];
  startedAt: number;
  lastActivityAt: number;
  totalQueries: number;
  totalFacts: number;
}

/**
 * Record of an archive query.
 */
export interface ArchiveQueryRecord {
  queryId: string;
  type: string;
  timestamp: number;
  executionTime: number;
  resultCount: number;
}

/**
 * Archive API response wrapper.
 */
export interface ArchiveResponse<T> {
  success: boolean;
  error?: {
    code: string;
    message: string;
  };
  result?: T;
  pagination?: {
    cursor?: string;
    hasMore: boolean;
    totalEstimate: number;
  };
  metadata: {
    queryId: string;
    executionTime: number;
    archiveBlockNumber: bigint;
  };
  pp: PPMetadata;
}

// ============================================================================
// Intent Types for Archive Queries
// ============================================================================

/**
 * Intent types for archive queries.
 */
export type ArchiveIntent =
  | EvaluateTruthIntent
  | GetRelatedIntent
  | FindSimilarIntent
  | QueryDomainIntent
  | QueryStratumIntent
  | SearchIntent;

export interface EvaluateTruthIntent {
  type: "evaluate_truth";
  factId: string;
}

export interface GetRelatedIntent {
  type: "get_related";
  factId: string;
  includeSemanticSimilar?: boolean;
  maxDepth?: number;
}

export interface FindSimilarIntent {
  type: "find_similar";
  factId?: string;
  query?: string;
  domain?: string;
  limit?: number;
}

export interface QueryDomainIntent {
  type: "query_domain";
  domain: string;
  includeDescendants?: boolean;
  minConfidence?: number;
  limit?: number;
}

export interface QueryStratumIntent {
  type: "query_stratum";
  stratum: OntologicalStratum;
  minConfidence?: number;
  maxConfidence?: number;
  limit?: number;
}

export interface SearchIntent {
  type: "search";
  query: string;
  domain?: string;
  stratum?: OntologicalStratum;
  limit?: number;
}

// ============================================================================
// Knowledge Archive API
// ============================================================================

/**
 * Knowledge Archive API - Intent-based interface for AI agents.
 *
 * Provides:
 * - Truth evaluation with confidence ratios
 * - Related statement discovery
 * - Semantic similarity search
 * - Domain and stratum-based queries
 */
export class KnowledgeArchiveAPI {
  private archive: KnowledgeArchive;
  private sessions: Map<string, ArchiveSession> = new Map();

  constructor(archive: KnowledgeArchive) {
    this.archive = archive;
  }

  // ============================================================================
  // Session Management
  // ============================================================================

  /**
   * Create a new archive session.
   */
  createSession(agentDid: DID): ArchiveSession {
    const sessionId = `archive:${agentDid}:${Date.now()}`;

    const session: ArchiveSession = {
      sessionId,
      agentDid,
      queryHistory: [],
      startedAt: Date.now(),
      lastActivityAt: Date.now(),
      totalQueries: 0,
      totalFacts: 0,
    };

    this.sessions.set(sessionId, session);
    return session;
  }

  /**
   * Get an existing session.
   */
  getSession(sessionId: string): ArchiveSession | undefined {
    return this.sessions.get(sessionId);
  }

  // ============================================================================
  // Intent-Based Queries
  // ============================================================================

  /**
   * Process an archive intent.
   */
  async processIntent(
    sessionId: string,
    intent: ArchiveIntent,
  ): Promise<ArchiveResponse<unknown>> {
    const session = this.sessions.get(sessionId);
    if (!session) {
      return this.errorResponse("SESSION_NOT_FOUND", "Session not found");
    }

    const startTime = Date.now();

    try {
      let result: ArchiveResponse<unknown>;

      switch (intent.type) {
        case "evaluate_truth":
          result = await this.handleEvaluateTruth(intent);
          break;
        case "get_related":
          result = await this.handleGetRelated(intent);
          break;
        case "find_similar":
          result = await this.handleFindSimilar(intent);
          break;
        case "query_domain":
          result = await this.handleQueryDomain(intent);
          break;
        case "query_stratum":
          result = await this.handleQueryStratum(intent);
          break;
        case "search":
          result = await this.handleSearch(intent);
          break;
        default:
          result = this.errorResponse("UNKNOWN_INTENT", "Unknown intent type");
      }

      // Update session
      session.lastActivityAt = Date.now();
      session.totalQueries++;
      session.queryHistory.push({
        queryId: result.metadata.queryId,
        type: intent.type,
        timestamp: startTime,
        executionTime: Date.now() - startTime,
        resultCount: Array.isArray(result.result) ? result.result.length : 1,
      });

      return result;
    } catch (error) {
      return this.errorResponse("INTERNAL_ERROR", String(error));
    }
  }

  // ============================================================================
  // Intent Handlers
  // ============================================================================

  /**
   * Handle truth evaluation intent.
   */
  private async handleEvaluateTruth(
    intent: EvaluateTruthIntent,
  ): Promise<ArchiveResponse<TruthEvaluation>> {
    const startTime = Date.now();
    const queryId = this.generateQueryId();

    try {
      const evaluation = await this.archive.evaluateTruth(intent.factId);

      const pp = createPPMetadata(
        {
          confidence: evaluation.confidence,
          domain: "truth_evaluation",
          citationCount:
            evaluation.supportingFacts.length +
            evaluation.contradictingFacts.length,
        },
        evaluation.confidence,
        false,
        evaluation.supportingFacts.map((f, i) => ({
          factId: f.factId,
          claim: f.claim,
          confidence: f.confidence,
          depth: i,
          relationship: f.referenceType,
        })),
      );

      return {
        success: true,
        result: evaluation,
        metadata: {
          queryId,
          executionTime: Date.now() - startTime,
          archiveBlockNumber: this.archive.getSyncStatus().currentBlock,
        },
        pp,
      };
    } catch (error) {
      return this.errorResponse("FACT_NOT_FOUND", String(error));
    }
  }

  /**
   * Handle get related intent.
   */
  private async handleGetRelated(
    intent: GetRelatedIntent,
  ): Promise<ArchiveResponse<RelatedStatements>> {
    const startTime = Date.now();
    const queryId = this.generateQueryId();

    try {
      const related = await this.archive.getRelatedStatements(intent.factId, {
        includeReferences: true,
        includeOntology: true,
        includeSimilarity: intent.includeSemanticSimilar ?? true,
        maxDepth: intent.maxDepth ?? 3,
        maxSimilarityResults: 10,
      });

      const pp = createPPMetadata(
        {
          confidence: 0.85,
          domain: "related_statements",
          citationCount: related.statistics.totalRelated,
        },
        0.85,
        false,
        [...related.supports, ...related.contradicts]
          .slice(0, 5)
          .map((f, i) => ({
            factId: f.factId,
            claim: f.claim,
            confidence: f.confidence,
            depth: i,
            relationship: "supports" as const,
          })),
      );

      return {
        success: true,
        result: related,
        metadata: {
          queryId,
          executionTime: Date.now() - startTime,
          archiveBlockNumber: this.archive.getSyncStatus().currentBlock,
        },
        pp,
      };
    } catch (error) {
      return this.errorResponse("FACT_NOT_FOUND", String(error));
    }
  }

  /**
   * Handle find similar intent.
   */
  private async handleFindSimilar(
    intent: FindSimilarIntent,
  ): Promise<ArchiveResponse<SemanticMatch[]>> {
    const startTime = Date.now();
    const queryId = this.generateQueryId();

    try {
      let matches: SemanticMatch[];

      if (intent.factId) {
        matches = await this.archive.findSimilar(intent.factId, {
          domain: intent.domain,
          limit: intent.limit ?? 10,
        });
      } else if (intent.query) {
        matches = await this.archive.search(intent.query, {
          domain: intent.domain,
          limit: intent.limit ?? 10,
        });
      } else {
        return this.errorResponse(
          "INVALID_QUERY",
          "Either factId or query required",
        );
      }

      const avgConfidence =
        matches.length > 0
          ? matches.reduce((sum, m) => sum + m.confidence, 0) / matches.length
          : 0;

      const pp = createPPMetadata(
        {
          confidence: avgConfidence,
          domain: intent.domain ?? "semantic_search",
          citationCount: matches.length,
        },
        avgConfidence,
        false,
        matches.slice(0, 5).map((m, i) => ({
          factId: m.factId,
          claim: m.claim,
          confidence: m.confidence,
          depth: i,
          relationship: "supports" as const,
        })),
      );

      return {
        success: true,
        result: matches,
        pagination: {
          hasMore: false,
          totalEstimate: matches.length,
        },
        metadata: {
          queryId,
          executionTime: Date.now() - startTime,
          archiveBlockNumber: this.archive.getSyncStatus().currentBlock,
        },
        pp,
      };
    } catch (error) {
      return this.errorResponse("SEARCH_ERROR", String(error));
    }
  }

  /**
   * Handle query domain intent.
   */
  private async handleQueryDomain(
    intent: QueryDomainIntent,
  ): Promise<ArchiveResponse<ArchiveFactEntry[]>> {
    const startTime = Date.now();
    const queryId = this.generateQueryId();

    const options: DomainQueryOptions & { pagination?: { limit: number } } = {
      includeDescendants: intent.includeDescendants ?? true,
      minConfidence: intent.minConfidence,
      pagination: {
        limit: intent.limit ?? 100,
      },
    };

    const result = await this.archive.queryByDomain(intent.domain, options);

    if (!result.success) {
      return this.errorResponse(
        result.error?.code ?? "QUERY_ERROR",
        result.error?.message ?? "Query failed",
      );
    }

    const avgConfidence =
      result.items.length > 0
        ? result.items.reduce((sum, e) => sum + e.fact.confidence, 0) /
          result.items.length
        : 0;

    const pp = createPPMetadata(
      {
        confidence: avgConfidence,
        domain: intent.domain,
        citationCount: result.items.length,
      },
      avgConfidence,
      false,
      result.items.slice(0, 5).map((e, i) => ({
        factId: e.fact.id,
        claim: e.fact.id,
        confidence: e.fact.confidence,
        depth: i,
        relationship: "supports" as const,
      })),
    );

    return {
      success: true,
      result: result.items,
      pagination: result.pagination,
      metadata: {
        queryId,
        executionTime: Date.now() - startTime,
        archiveBlockNumber: this.archive.getSyncStatus().currentBlock,
      },
      pp,
    };
  }

  /**
   * Handle query stratum intent.
   */
  private async handleQueryStratum(
    intent: QueryStratumIntent,
  ): Promise<ArchiveResponse<ArchiveFactEntry[]>> {
    const startTime = Date.now();
    const queryId = this.generateQueryId();

    const options: ConfidenceQueryOptions & { pagination?: { limit: number } } =
      {
        minConfidence: intent.minConfidence,
        maxConfidence: intent.maxConfidence,
        pagination: {
          limit: intent.limit ?? 100,
        },
      };

    const result = await this.archive.queryByStratum(intent.stratum, options);

    if (!result.success) {
      return this.errorResponse(
        result.error?.code ?? "QUERY_ERROR",
        result.error?.message ?? "Query failed",
      );
    }

    const avgConfidence =
      result.items.length > 0
        ? result.items.reduce((sum, e) => sum + e.fact.confidence, 0) /
          result.items.length
        : 0;

    const pp = createPPMetadata(
      {
        confidence: avgConfidence,
        domain: intent.stratum,
        citationCount: result.items.length,
      },
      avgConfidence,
      false,
      result.items.slice(0, 5).map((e, i) => ({
        factId: e.fact.id,
        claim: e.fact.id,
        confidence: e.fact.confidence,
        depth: i,
        relationship: "supports" as const,
      })),
    );

    return {
      success: true,
      result: result.items,
      pagination: result.pagination,
      metadata: {
        queryId,
        executionTime: Date.now() - startTime,
        archiveBlockNumber: this.archive.getSyncStatus().currentBlock,
      },
      pp,
    };
  }

  /**
   * Handle search intent.
   */
  private async handleSearch(
    intent: SearchIntent,
  ): Promise<ArchiveResponse<SemanticMatch[]>> {
    const startTime = Date.now();
    const queryId = this.generateQueryId();

    const matches = await this.archive.search(intent.query, {
      domain: intent.domain,
      stratum: intent.stratum,
      limit: intent.limit ?? 20,
    });

    const avgSimilarity =
      matches.length > 0
        ? matches.reduce((sum, m) => sum + m.similarityScore, 0) /
          matches.length
        : 0;

    const pp = createPPMetadata(
      {
        confidence: avgSimilarity,
        domain: intent.domain ?? "search",
        citationCount: matches.length,
      },
      avgSimilarity,
      false,
      matches.slice(0, 5).map((m, i) => ({
        factId: m.factId,
        claim: m.claim,
        confidence: m.confidence,
        depth: i,
        relationship: "supports" as const,
      })),
    );

    return {
      success: true,
      result: matches,
      pagination: {
        hasMore: false,
        totalEstimate: matches.length,
      },
      metadata: {
        queryId,
        executionTime: Date.now() - startTime,
        archiveBlockNumber: this.archive.getSyncStatus().currentBlock,
      },
      pp,
    };
  }

  // ============================================================================
  // Direct Archive Access
  // ============================================================================

  /**
   * Get archive sync status.
   */
  getSyncStatus(): ArchiveSyncStatus {
    return this.archive.getSyncStatus();
  }

  /**
   * Get archive statistics.
   */
  getStatistics(): ArchiveStatistics {
    return this.archive.getStatistics();
  }

  /**
   * Execute a batch query.
   */
  async batchQuery(batch: BatchArchiveQuery): Promise<BatchQueryResult> {
    return this.archive.batchQuery(batch);
  }

  /**
   * Get a fact by ID.
   */
  getFact(factId: string): ArchiveFactEntry | undefined {
    return this.archive.getFact(factId);
  }

  // ============================================================================
  // Utility Methods
  // ============================================================================

  /**
   * Generate a unique query ID.
   */
  private generateQueryId(): string {
    return `aq_${Date.now()}_${Math.random().toString(36).substring(2, 8)}`;
  }

  /**
   * Create an error response.
   */
  private errorResponse<T>(code: string, message: string): ArchiveResponse<T> {
    return {
      success: false,
      error: { code, message },
      metadata: {
        queryId: this.generateQueryId(),
        executionTime: 0,
        archiveBlockNumber: this.archive.getSyncStatus().currentBlock,
      },
      pp: createPPMetadata(
        { confidence: 0, domain: "error", citationCount: 0 },
        0,
        false,
        [],
      ),
    };
  }
}
