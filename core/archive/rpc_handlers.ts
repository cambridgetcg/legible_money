/**
 * Archive RPC Handlers
 *
 * JSON-RPC handlers for the Knowledge Archive API.
 * Provides external access to archive queries for AI agents.
 */

import type {
  ArchiveFactEntry,
  TruthEvaluation,
  RelatedStatements,
  SemanticMatch,
  BatchArchiveQuery,
  BatchQueryResult,
  ArchiveSyncStatus,
  ArchiveStatistics,
  ArchiveQuery,
} from "./types.js";
import type { OntologicalStratum } from "../knowledge/ontology/types.js";
import {
  KnowledgeArchiveAPI,
  type ArchiveIntent,
  type ArchiveResponse,
} from "./archive_api.js";

// ============================================================================
// RPC Request/Response Types
// ============================================================================

/**
 * JSON-RPC request structure.
 */
export interface RPCRequest<P = unknown> {
  jsonrpc: "2.0";
  id: string | number;
  method: string;
  params?: P;
}

/**
 * JSON-RPC response structure.
 */
export interface RPCResponse<R = unknown> {
  jsonrpc: "2.0";
  id: string | number;
  result?: R;
  error?: RPCError;
}

/**
 * JSON-RPC error structure.
 */
export interface RPCError {
  code: number;
  message: string;
  data?: unknown;
}

// ============================================================================
// RPC Method Parameter Types
// ============================================================================

export interface CreateSessionParams {
  agentDid: string;
}

export interface EvaluateTruthParams {
  sessionId: string;
  factId: string;
}

export interface GetRelatedParams {
  sessionId: string;
  factId: string;
  includeSemanticSimilar?: boolean;
  maxDepth?: number;
}

export interface FindSimilarParams {
  sessionId: string;
  factId?: string;
  query?: string;
  domain?: string;
  limit?: number;
}

export interface QueryDomainParams {
  sessionId: string;
  domain: string;
  includeDescendants?: boolean;
  minConfidence?: number;
  limit?: number;
}

export interface QueryStratumParams {
  sessionId: string;
  stratum: OntologicalStratum;
  minConfidence?: number;
  maxConfidence?: number;
  limit?: number;
}

export interface SearchParams {
  sessionId: string;
  query: string;
  domain?: string;
  stratum?: OntologicalStratum;
  limit?: number;
}

export interface BatchQueryParams {
  sessionId: string;
  queries: ArchiveQuery[];
  parallel?: boolean;
  maxParallelism?: number;
}

export interface GetFactParams {
  factId: string;
}

// ============================================================================
// RPC Error Codes
// ============================================================================

export const RPC_ERRORS = {
  PARSE_ERROR: { code: -32700, message: "Parse error" },
  INVALID_REQUEST: { code: -32600, message: "Invalid request" },
  METHOD_NOT_FOUND: { code: -32601, message: "Method not found" },
  INVALID_PARAMS: { code: -32602, message: "Invalid params" },
  INTERNAL_ERROR: { code: -32603, message: "Internal error" },
  // Custom error codes
  SESSION_NOT_FOUND: { code: -32001, message: "Session not found" },
  FACT_NOT_FOUND: { code: -32002, message: "Fact not found" },
  DOMAIN_NOT_FOUND: { code: -32003, message: "Domain not found" },
  QUERY_TIMEOUT: { code: -32004, message: "Query timeout" },
} as const;

// ============================================================================
// Archive RPC Handler
// ============================================================================

/**
 * Handles JSON-RPC requests for the Knowledge Archive.
 */
export class ArchiveRPCHandler {
  private api: KnowledgeArchiveAPI;

  constructor(api: KnowledgeArchiveAPI) {
    this.api = api;
  }

  /**
   * Handle a JSON-RPC request.
   */
  async handleRequest(request: RPCRequest): Promise<RPCResponse> {
    try {
      // Validate request
      if (request.jsonrpc !== "2.0") {
        return this.errorResponse(request.id, RPC_ERRORS.INVALID_REQUEST);
      }

      // Route to handler
      const result = await this.routeMethod(request.method, request.params);

      return {
        jsonrpc: "2.0",
        id: request.id,
        result,
      };
    } catch (error) {
      return this.errorResponse(request.id, {
        code: RPC_ERRORS.INTERNAL_ERROR.code,
        message: String(error),
      });
    }
  }

  /**
   * Route method to appropriate handler.
   */
  private async routeMethod(method: string, params: unknown): Promise<unknown> {
    switch (method) {
      // Session management
      case "archive_createSession":
        return this.handleCreateSession(params as CreateSessionParams);

      // Truth evaluation
      case "archive_evaluateTruth":
        return this.handleEvaluateTruth(params as EvaluateTruthParams);

      // Related statements
      case "archive_getRelated":
        return this.handleGetRelated(params as GetRelatedParams);

      // Similarity search
      case "archive_findSimilar":
        return this.handleFindSimilar(params as FindSimilarParams);

      // Domain queries
      case "archive_queryDomain":
        return this.handleQueryDomain(params as QueryDomainParams);

      // Stratum queries
      case "archive_queryStratum":
        return this.handleQueryStratum(params as QueryStratumParams);

      // Semantic search
      case "archive_search":
        return this.handleSearch(params as SearchParams);

      // Batch queries
      case "archive_batchQuery":
        return this.handleBatchQuery(params as BatchQueryParams);

      // Direct access
      case "archive_getFact":
        return this.handleGetFact(params as GetFactParams);

      case "archive_getSyncStatus":
        return this.handleGetSyncStatus();

      case "archive_getStatistics":
        return this.handleGetStatistics();

      default:
        throw new Error(`Unknown method: ${method}`);
    }
  }

  // ============================================================================
  // Handler Methods
  // ============================================================================

  /**
   * Create a new session.
   */
  private async handleCreateSession(
    params: CreateSessionParams,
  ): Promise<{ sessionId: string }> {
    this.validateParams(params, ["agentDid"]);
    const session = this.api.createSession(params.agentDid);
    return { sessionId: session.sessionId };
  }

  /**
   * Evaluate truth of a fact.
   */
  private async handleEvaluateTruth(
    params: EvaluateTruthParams,
  ): Promise<ArchiveResponse<TruthEvaluation>> {
    this.validateParams(params, ["sessionId", "factId"]);

    const intent: ArchiveIntent = {
      type: "evaluate_truth",
      factId: params.factId,
    };

    return this.api.processIntent(params.sessionId, intent) as Promise<
      ArchiveResponse<TruthEvaluation>
    >;
  }

  /**
   * Get related statements.
   */
  private async handleGetRelated(
    params: GetRelatedParams,
  ): Promise<ArchiveResponse<RelatedStatements>> {
    this.validateParams(params, ["sessionId", "factId"]);

    const intent: ArchiveIntent = {
      type: "get_related",
      factId: params.factId,
      includeSemanticSimilar: params.includeSemanticSimilar,
      maxDepth: params.maxDepth,
    };

    return this.api.processIntent(params.sessionId, intent) as Promise<
      ArchiveResponse<RelatedStatements>
    >;
  }

  /**
   * Find similar facts.
   */
  private async handleFindSimilar(
    params: FindSimilarParams,
  ): Promise<ArchiveResponse<SemanticMatch[]>> {
    this.validateParams(params, ["sessionId"]);

    if (!params.factId && !params.query) {
      throw new Error("Either factId or query required");
    }

    const intent: ArchiveIntent = {
      type: "find_similar",
      factId: params.factId,
      query: params.query,
      domain: params.domain,
      limit: params.limit,
    };

    return this.api.processIntent(params.sessionId, intent) as Promise<
      ArchiveResponse<SemanticMatch[]>
    >;
  }

  /**
   * Query by domain.
   */
  private async handleQueryDomain(
    params: QueryDomainParams,
  ): Promise<ArchiveResponse<ArchiveFactEntry[]>> {
    this.validateParams(params, ["sessionId", "domain"]);

    const intent: ArchiveIntent = {
      type: "query_domain",
      domain: params.domain,
      includeDescendants: params.includeDescendants,
      minConfidence: params.minConfidence,
      limit: params.limit,
    };

    return this.api.processIntent(params.sessionId, intent) as Promise<
      ArchiveResponse<ArchiveFactEntry[]>
    >;
  }

  /**
   * Query by stratum.
   */
  private async handleQueryStratum(
    params: QueryStratumParams,
  ): Promise<ArchiveResponse<ArchiveFactEntry[]>> {
    this.validateParams(params, ["sessionId", "stratum"]);

    const intent: ArchiveIntent = {
      type: "query_stratum",
      stratum: params.stratum,
      minConfidence: params.minConfidence,
      maxConfidence: params.maxConfidence,
      limit: params.limit,
    };

    return this.api.processIntent(params.sessionId, intent) as Promise<
      ArchiveResponse<ArchiveFactEntry[]>
    >;
  }

  /**
   * Search by text query.
   */
  private async handleSearch(
    params: SearchParams,
  ): Promise<ArchiveResponse<SemanticMatch[]>> {
    this.validateParams(params, ["sessionId", "query"]);

    const intent: ArchiveIntent = {
      type: "search",
      query: params.query,
      domain: params.domain,
      stratum: params.stratum,
      limit: params.limit,
    };

    return this.api.processIntent(params.sessionId, intent) as Promise<
      ArchiveResponse<SemanticMatch[]>
    >;
  }

  /**
   * Execute batch query.
   */
  private async handleBatchQuery(
    params: BatchQueryParams,
  ): Promise<BatchQueryResult> {
    this.validateParams(params, ["sessionId", "queries"]);

    const batch: BatchArchiveQuery = {
      queries: params.queries,
      options: {
        parallel: params.parallel ?? true,
        maxParallelism: params.maxParallelism ?? 5,
        timeout: 30000,
        streaming: false,
      },
    };

    return this.api.batchQuery(batch);
  }

  /**
   * Get a fact by ID.
   */
  private async handleGetFact(
    params: GetFactParams,
  ): Promise<ArchiveFactEntry | null> {
    this.validateParams(params, ["factId"]);
    return this.api.getFact(params.factId) ?? null;
  }

  /**
   * Get sync status.
   */
  private async handleGetSyncStatus(): Promise<ArchiveSyncStatus> {
    return this.api.getSyncStatus();
  }

  /**
   * Get statistics.
   */
  private async handleGetStatistics(): Promise<ArchiveStatistics> {
    return this.api.getStatistics();
  }

  // ============================================================================
  // Utility Methods
  // ============================================================================

  /**
   * Validate required parameters.
   */
  private validateParams(params: unknown, required: string[]): void {
    if (!params || typeof params !== "object") {
      throw new Error("Invalid params");
    }

    for (const key of required) {
      if (!(key in params)) {
        throw new Error(`Missing required parameter: ${key}`);
      }
    }
  }

  /**
   * Create an error response.
   */
  private errorResponse(
    id: string | number,
    error: { code: number; message: string },
  ): RPCResponse {
    return {
      jsonrpc: "2.0",
      id,
      error,
    };
  }
}

// ============================================================================
// RPC Method Registry
// ============================================================================

/**
 * Registry of available RPC methods with descriptions.
 */
export const ARCHIVE_RPC_METHODS = {
  // Session management
  archive_createSession: {
    description: "Create a new archive session for an agent",
    params: ["agentDid"],
    returns: "sessionId",
  },

  // Truth evaluation
  archive_evaluateTruth: {
    description: "Evaluate the truth of a fact with confidence ratio",
    params: ["sessionId", "factId"],
    returns:
      "TruthEvaluation with verdict (true/false/uncertain), confidence, supporting/contradicting facts",
  },

  // Related statements
  archive_getRelated: {
    description: "Get related statements for a fact",
    params: ["sessionId", "factId", "includeSemanticSimilar?", "maxDepth?"],
    returns:
      "RelatedStatements with supports, contradicts, extends, semanticallySimilar",
  },

  // Similarity search
  archive_findSimilar: {
    description: "Find semantically similar facts",
    params: ["sessionId", "factId?", "query?", "domain?", "limit?"],
    returns: "Array of SemanticMatch with similarity scores",
  },

  // Domain queries
  archive_queryDomain: {
    description: "Query facts by domain pattern",
    params: [
      "sessionId",
      "domain",
      "includeDescendants?",
      "minConfidence?",
      "limit?",
    ],
    returns: "Array of ArchiveFactEntry",
  },

  // Stratum queries
  archive_queryStratum: {
    description: "Query facts by ontological stratum",
    params: [
      "sessionId",
      "stratum",
      "minConfidence?",
      "maxConfidence?",
      "limit?",
    ],
    returns: "Array of ArchiveFactEntry",
  },

  // Semantic search
  archive_search: {
    description: "Search for facts by text query",
    params: ["sessionId", "query", "domain?", "stratum?", "limit?"],
    returns: "Array of SemanticMatch",
  },

  // Batch queries
  archive_batchQuery: {
    description: "Execute multiple queries in batch",
    params: ["sessionId", "queries", "parallel?", "maxParallelism?"],
    returns: "BatchQueryResult with results and errors",
  },

  // Direct access
  archive_getFact: {
    description: "Get a fact by ID",
    params: ["factId"],
    returns: "ArchiveFactEntry or null",
  },

  archive_getSyncStatus: {
    description: "Get archive sync status",
    params: [],
    returns: "ArchiveSyncStatus",
  },

  archive_getStatistics: {
    description: "Get archive statistics",
    params: [],
    returns: "ArchiveStatistics",
  },
} as const;
