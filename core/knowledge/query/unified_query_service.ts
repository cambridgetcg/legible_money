/**
 * Unified Query Service
 *
 * Orchestrates knowledge queries during agent chat sessions.
 * Integrates Archive API, DomainReasoningIndex, SessionCache,
 * and PrefetchEngine for optimized query execution.
 *
 * Query Flow:
 * 1. Get/create reasoning session
 * 2. Record intent pattern for prefetch learning
 * 3. Check session cache (O(1))
 * 4. Execute optimized index query (O(buckets))
 * 5. Cache results
 * 6. Trigger async prefetch
 *
 * PP HIERARCHY:
 *   TRUTH: Queries return verified on-chain facts
 *   UNDERSTANDING: Session context enables deep reasoning
 *   BEAUTY: Single orchestrator with clean separation
 *   JUSTICE: Placed in core/knowledge/query alongside indexes
 *   CREATIVITY: Novel integration of index + cache + prefetch
 */

import type { DID } from "../../../core/agents/identity/did.js";
import type { FactState, DecayInfo } from "../../../state/types.js";
import {
  type OntologicalDomain,
  type OntologicalStratum,
  STRATUM_TO_DECAY_CATEGORY,
} from "../ontology/types.js";
// GAP-008: Confidence decay calculation
import { calculateCategoryDecayedConfidence } from "../../../consensus/pot/confidence.js";
import type { ReasoningSession } from "../indexes/types.js";
import { DomainReasoningIndex } from "../indexes/reasoning_index.js";
import { GatewayDomainIndex } from "../indexes/gateway_index.js";
import { ReasoningSessionManager } from "../indexes/session_cache.js";
import { PrefetchEngine } from "../indexes/prefetch_engine.js";
import { QueryCostEstimator } from "./cost_estimator.js";
// GAP-004: Semantic search integration
import type {
  SemanticSimilarityIndex,
  SemanticQueryOptions,
} from "../../../core/archive/indexes/semantic_index.js";
// GAP-007: Per-message billing integration
import type { MessageBillingTracker } from "../../billing/message_tracker.js";
// GAP-010: Rate limiting integration
import { QueryRateLimiter } from "./rate_limiter.js";
import { determineTierFromBudget, type RateLimitTier } from "./rate_limit.js";
import {
  type AgentIntent,
  type IntentResponse,
  type FactResult,
  type DomainResult,
  type PathResult,
  type QueryMetadata,
  type AgentChatSession,
  type QueryCostEstimate,
  type UnifiedQueryServiceConfig,
  DEFAULT_UNIFIED_QUERY_SERVICE_CONFIG,
  QueryRateLimitError,
} from "./types.js";

// ============================================================================
// Unified Query Service
// ============================================================================

/**
 * UnifiedQueryService orchestrates knowledge queries for agent chat sessions.
 */
export class UnifiedQueryService {
  private config: UnifiedQueryServiceConfig;
  private reasoningIndex: DomainReasoningIndex;
  private gatewayIndex: GatewayDomainIndex;
  private sessionManager: ReasoningSessionManager;
  private prefetchEngine: PrefetchEngine;
  private costEstimator: QueryCostEstimator;

  /** Active chat sessions */
  private chatSessions: Map<string, AgentChatSession> = new Map();

  /** Fact loader function (injected) */
  private factLoader?: (factId: string) => Promise<FactState | undefined>;

  /** Domain loader function (injected) */
  private domainLoader?: (
    domainId: string,
  ) => Promise<OntologicalDomain | undefined>;

  /** GAP-004: Semantic similarity index for natural language queries */
  private semanticIndex?: SemanticSimilarityIndex;

  /** GAP-007: Message billing tracker for per-message cost aggregation */
  private messageBillingTracker?: MessageBillingTracker;

  /** GAP-010: Rate limiter for query throttling */
  private rateLimiter?: QueryRateLimiter;

  /** Citation recorder callback (injected) for tracking fact usage */
  private citationRecorder?: (factId: string, blockNumber: bigint) => void;

  constructor(
    reasoningIndex: DomainReasoningIndex,
    gatewayIndex: GatewayDomainIndex,
    sessionManager: ReasoningSessionManager,
    prefetchEngine: PrefetchEngine,
    config: UnifiedQueryServiceConfig = DEFAULT_UNIFIED_QUERY_SERVICE_CONFIG,
  ) {
    this.reasoningIndex = reasoningIndex;
    this.gatewayIndex = gatewayIndex;
    this.sessionManager = sessionManager;
    this.prefetchEngine = prefetchEngine;
    this.config = { ...config };
    this.costEstimator = new QueryCostEstimator(config.costConfig);
  }

  // ============================================================================
  // Configuration
  // ============================================================================

  /**
   * Set the fact loader function.
   */
  setFactLoader(
    loader: (factId: string) => Promise<FactState | undefined>,
  ): void {
    this.factLoader = loader;
  }

  /**
   * Set the domain loader function.
   */
  setDomainLoader(
    loader: (domainId: string) => Promise<OntologicalDomain | undefined>,
  ): void {
    this.domainLoader = loader;
  }

  /**
   * GAP-004: Set the semantic similarity index for natural language queries.
   */
  setSemanticIndex(index: SemanticSimilarityIndex): void {
    this.semanticIndex = index;
  }

  /**
   * GAP-007: Set the message billing tracker for per-message cost aggregation.
   */
  setMessageBillingTracker(tracker: MessageBillingTracker): void {
    this.messageBillingTracker = tracker;
  }

  /**
   * GAP-010: Set the query rate limiter.
   */
  setRateLimiter(limiter: QueryRateLimiter): void {
    this.rateLimiter = limiter;
  }

  /**
   * GAP-010: Get the query rate limiter.
   */
  getRateLimiter(): QueryRateLimiter | undefined {
    return this.rateLimiter;
  }

  /**
   * Set the citation recorder callback.
   * Called when facts are accessed via queries to drive citation economics.
   */
  setCitationRecorder(
    recorder: (factId: string, blockNumber: bigint) => void,
  ): void {
    this.citationRecorder = recorder;
  }

  // ============================================================================
  // Session Management
  // ============================================================================

  /**
   * Start a new chat session.
   */
  startSession(
    agentDid: DID,
    currentBlock: bigint,
    budgetLimit?: bigint,
  ): AgentChatSession {
    const sessionId = `chat:${agentDid}:${Date.now()}`;

    const session: AgentChatSession = {
      sessionId,
      agentDid,
      startBlock: currentBlock,
      startedAt: Date.now(),
      lastActivityAt: Date.now(),
      currentBlock,
      queryCount: 0,
      totalCost: 0n,
      budgetLimit,
    };

    this.chatSessions.set(sessionId, session);

    // Also create reasoning session
    this.sessionManager.getOrCreateSession(agentDid, currentBlock);

    return session;
  }

  /**
   * Get an existing chat session.
   */
  getSession(sessionId: string): AgentChatSession | undefined {
    const session = this.chatSessions.get(sessionId);
    if (session) {
      session.lastActivityAt = Date.now();
    }
    return session;
  }

  /**
   * End a chat session.
   */
  endSession(sessionId: string): void {
    const session = this.chatSessions.get(sessionId);
    if (session) {
      // End reasoning session
      const reasoningSession = this.sessionManager.getOrCreateSession(
        session.agentDid,
        session.currentBlock,
      );
      this.sessionManager.endSession(reasoningSession.sessionId);

      // GAP-010: Clear rate limit state
      if (this.rateLimiter) {
        this.rateLimiter.clearSession(sessionId);
      }

      // Remove chat session
      this.chatSessions.delete(sessionId);
    }
  }

  /**
   * Update session's current block.
   */
  updateCurrentBlock(sessionId: string, block: bigint): void {
    const session = this.chatSessions.get(sessionId);
    if (session) {
      session.currentBlock = block;
      session.lastActivityAt = Date.now();
    }
  }

  // ============================================================================
  // Cost Estimation
  // ============================================================================

  /**
   * Estimate cost for an intent before execution.
   */
  estimateQueryCost(sessionId: string, intent: AgentIntent): QueryCostEstimate {
    const chatSession = this.chatSessions.get(sessionId);
    if (!chatSession) {
      throw new Error("Session not found");
    }

    // Check cache
    const reasoningSession = this.sessionManager.getOrCreateSession(
      chatSession.agentDid,
      chatSession.currentBlock,
    );
    const isCached = this.checkCacheHit(reasoningSession, intent);

    return this.costEstimator.estimateCost(intent, chatSession, isCached);
  }

  /**
   * Get remaining budget for session.
   */
  getRemainingBudget(sessionId: string): bigint | undefined {
    const session = this.chatSessions.get(sessionId);
    if (!session) return undefined;
    return this.costEstimator.getRemainingBudget(session);
  }

  // ============================================================================
  // Query Execution
  // ============================================================================

  /**
   * Process an intent with full optimization stack.
   *
   * GAP-007: If messageId is provided and a message billing tracker is configured,
   * query costs are recorded to the message for aggregation.
   */
  async processIntent(
    sessionId: string,
    intent: AgentIntent,
    messageId?: string,
  ): Promise<IntentResponse> {
    const startTime = Date.now();

    // Get chat session
    const chatSession = this.chatSessions.get(sessionId);
    if (!chatSession) {
      throw new Error("Session not found");
    }

    // GAP-010: Check rate limit before processing
    if (this.rateLimiter) {
      const tier = this.determineRateLimitTier(chatSession);
      const rateLimitResult = this.rateLimiter.checkRateLimit(
        sessionId,
        chatSession.agentDid,
        tier,
        chatSession.currentBlock,
      );

      if (!rateLimitResult.allowed) {
        throw new QueryRateLimitError(rateLimitResult);
      }
    }

    // Get or create reasoning session
    const reasoningSession = this.sessionManager.getOrCreateSession(
      chatSession.agentDid,
      chatSession.currentBlock,
    );

    // Record intent pattern for prefetch learning
    this.sessionManager.recordQueryIntent(reasoningSession, intent.type, {
      domain: intent.domain,
      depth: intent.depth,
      stratum: intent.stratum,
    });

    // Check session cache first (O(1))
    const cachedResult = this.checkCache(
      reasoningSession,
      intent,
      chatSession.currentBlock,
    );
    if (cachedResult) {
      const latencyMs = Date.now() - startTime;

      // GAP-007: Record cached query (cost = 0) to message tracker
      if (messageId && this.messageBillingTracker) {
        const cachedQueryCost: QueryCostEstimate = {
          cost: 0n,
          cached: true,
          estimatedBuckets: 0,
          crossStratumMultiplier: 1n,
          breakdown: {
            baseCost: 0n,
            bucketCost: 0n,
            crossStratumCost: 0n,
            materializationCost: 0n,
          },
        };
        this.messageBillingTracker.recordQuery(
          messageId,
          cachedQueryCost,
          cachedResult.facts.map((f) => f.factId),
          intent.crossStrata,
        );
      }

      // GAP-010: Record cached query to rate limiter (if configured to count cached)
      if (this.rateLimiter) {
        this.rateLimiter.recordQuery(sessionId, true);
        this.rateLimiter.recordQueryComplete(sessionId);
      }

      // Add rate limit info to cached result
      const wrappedResult = this.wrapCachedResult(
        cachedResult,
        reasoningSession,
        latencyMs,
      );
      if (this.rateLimiter) {
        wrappedResult.metadata.rateLimit =
          this.rateLimiter.getRateLimitInfo(sessionId);
      }
      return wrappedResult;
    }

    // Check budget
    const estimate = this.costEstimator.estimateCost(
      intent,
      chatSession,
      false,
    );
    if (!this.costEstimator.hasBudget(chatSession, estimate.cost)) {
      throw new Error("Insufficient budget for query");
    }

    // Execute optimized query (O(buckets) vs O(n))
    const result = await this.executeOptimizedQuery(
      reasoningSession,
      intent,
      chatSession.currentBlock,
    );

    // Cache results
    this.cacheResults(reasoningSession, intent, result);

    // Calculate actual cost
    const actualCost = this.costEstimator.calculateActualCost(
      intent,
      chatSession,
      result.metadata.bucketsScanned,
      result.facts.length,
      result.domains.length,
      false,
    );

    // GAP-007: Record query cost to message tracker if configured
    if (messageId && this.messageBillingTracker) {
      // Create QueryCostEstimate with actual cost
      const queryCostForBilling: QueryCostEstimate = {
        ...estimate,
        cost: actualCost,
        cached: false,
      };
      this.messageBillingTracker.recordQuery(
        messageId,
        queryCostForBilling,
        result.facts.map((f) => f.factId),
        intent.crossStrata,
      );
    }

    // Update session
    chatSession.queryCount++;
    chatSession.totalCost += actualCost;
    chatSession.lastActivityAt = Date.now();

    // GAP-010: Record query to rate limiter and complete concurrent tracking
    if (this.rateLimiter) {
      this.rateLimiter.recordQuery(sessionId, false);
      this.rateLimiter.recordQueryComplete(sessionId);
    }

    // Record citations for accessed facts (drives citation economics)
    if (this.citationRecorder && result.facts.length > 0) {
      for (const factResult of result.facts) {
        this.citationRecorder(factResult.factId, chatSession.currentBlock);
      }
    }

    // Trigger async prefetch for predicted next domains
    if (this.config.enablePrefetch) {
      this.triggerPrefetch(reasoningSession, intent);
    }

    // Update metadata
    result.metadata.latencyMs = Date.now() - startTime;
    result.metadata.estimatedCost = estimate.cost;
    result.metadata.actualCost = actualCost;
    result.metadata.cacheHitRatio = reasoningSession.stats.cacheHitRatio;
    result.metadata.prefetchHitRatio = reasoningSession.stats.prefetchHitRatio;

    // GAP-010: Add rate limit info to response metadata
    if (this.rateLimiter) {
      result.metadata.rateLimit = this.rateLimiter.getRateLimitInfo(sessionId);
    }

    return result;
  }

  // ============================================================================
  // Cache Operations
  // ============================================================================

  /**
   * Check if intent result is fully cached.
   */
  private checkCacheHit(
    session: ReasoningSession,
    intent: AgentIntent,
  ): boolean {
    if (intent.type === "explore" || intent.type === "connect") {
      // These require fresh computation
      return false;
    }

    if (intent.domain) {
      const domainCache = this.sessionManager.getDomainCache(session.sessionId);
      return domainCache?.has(intent.domain) ?? false;
    }

    if (intent.factId) {
      const factCache = this.sessionManager.getFactCache(session.sessionId);
      return factCache?.has(intent.factId) ?? false;
    }

    return false;
  }

  /**
   * Check cache and return result if available.
   */
  private checkCache(
    session: ReasoningSession,
    intent: AgentIntent,
    currentBlock: bigint,
  ): IntentResponse | null {
    // Single fact lookup
    if (intent.factId) {
      const factCache = this.sessionManager.getFactCache(session.sessionId);
      const cached = factCache?.get(intent.factId);
      if (cached) {
        return {
          intentType: intent.type,
          facts: [
            this.buildFactResult(
              intent.factId,
              cached.fact,
              cached.domain,
              cached.stratum,
              1.0,
              true,
              false,
              currentBlock,
            ),
          ],
          domains: [],
          metadata: this.createCachedMetadata(),
        };
      }
    }

    // Single domain lookup (for understand intent)
    if (intent.type === "understand" && intent.domain) {
      const domainCache = this.sessionManager.getDomainCache(session.sessionId);
      const cached = domainCache?.get(intent.domain);
      if (cached) {
        return {
          intentType: intent.type,
          facts: [],
          domains: [
            {
              domainId: intent.domain,
              domain: cached.domain,
              factCount: cached.factCount,
              trajectory: cached.trajectory,
              isGateway: this.gatewayIndex.isGateway(intent.domain),
              gatewayScore: this.gatewayIndex.getGateway(intent.domain)
                ?.gatewayScore,
            },
          ],
          trajectory: cached.trajectory,
          metadata: this.createCachedMetadata(),
        };
      }
    }

    return null;
  }

  /**
   * Cache results from a query.
   */
  private cacheResults(
    session: ReasoningSession,
    intent: AgentIntent,
    result: IntentResponse,
  ): void {
    // Cache facts
    for (const factResult of result.facts) {
      this.sessionManager.cacheFact(
        session,
        factResult.factId,
        factResult.fact,
        factResult.domain,
        factResult.stratum,
      );
    }

    // Cache domains
    for (const domainResult of result.domains) {
      this.sessionManager.cacheDomain(
        session,
        domainResult.domainId,
        domainResult.domain,
        domainResult.factCount,
        domainResult.trajectory,
      );
    }

    // Record domain access
    if (intent.domain) {
      this.sessionManager.recordDomainAccess(session, intent.domain);
    }
  }

  private createCachedMetadata(): QueryMetadata {
    return {
      cached: true,
      bucketsScanned: 0,
      domainsSearched: [],
      latencyMs: 0,
      cacheHitRatio: 1.0,
      prefetchHitRatio: 0,
    };
  }

  private wrapCachedResult(
    result: IntentResponse,
    session: ReasoningSession,
    latencyMs: number,
  ): IntentResponse {
    return {
      ...result,
      metadata: {
        ...result.metadata,
        cached: true,
        latencyMs,
        cacheHitRatio: session.stats.cacheHitRatio,
        prefetchHitRatio: session.stats.prefetchHitRatio,
      },
    };
  }

  // ============================================================================
  // Query Execution
  // ============================================================================

  /**
   * Execute an optimized query using the reasoning index.
   */
  private async executeOptimizedQuery(
    session: ReasoningSession,
    intent: AgentIntent,
    currentBlock: bigint,
  ): Promise<IntentResponse> {
    // GAP-004: Check for semantic query first (intent.query takes precedence)
    if (intent.query && this.semanticIndex) {
      return this.executeSemanticQuery(session, intent, currentBlock);
    }

    switch (intent.type) {
      case "understand":
        return this.executeUnderstandQuery(session, intent, currentBlock);

      case "verify":
        return this.executeVerifyQuery(session, intent, currentBlock);

      case "explore":
        return this.executeExploreQuery(session, intent, currentBlock);

      case "connect":
        return this.executeConnectQuery(session, intent, currentBlock);

      case "search":
        // Explicit search intent without semanticIndex configured
        if (!this.semanticIndex) {
          return this.createEmptyResponse("search");
        }
        return this.executeSemanticQuery(session, intent, currentBlock);

      case "contribute":
      case "challenge":
        // These are write intents, return empty result
        return this.createEmptyResponse(intent.type);

      default:
        return this.createEmptyResponse(intent.type);
    }
  }

  /**
   * Execute an understand intent.
   */
  private async executeUnderstandQuery(
    session: ReasoningSession,
    intent: AgentIntent,
    currentBlock: bigint,
  ): Promise<IntentResponse> {
    const facts: FactResult[] = [];
    const domains: DomainResult[] = [];
    let bucketsScanned = 0;
    const domainsSearched: string[] = [];

    if (intent.domain) {
      domainsSearched.push(intent.domain);

      // Get facts in domain using optimized index
      const factResult = this.reasoningIndex.getFactsInDomain(intent.domain, {
        minConfidence: intent.minConfidence ?? 0.6,
        includeDescendants: intent.depth ? intent.depth >= 2 : false,
        limit: intent.limit ?? 50,
      });

      bucketsScanned = factResult.bucketsScanned;

      // Load and materialize facts
      for (const factId of factResult.factIds) {
        const fact = await this.loadFact(factId);
        if (fact) {
          const domain = this.reasoningIndex.getDomain(intent.domain);
          const stratum = this.reasoningIndex.getFactStratum(factId);
          if (domain && stratum) {
            facts.push(
              this.buildFactResult(
                factId,
                fact,
                domain,
                stratum,
                fact.confidence,
                false,
                session.prefetchedDomains.has(intent.domain),
                currentBlock,
              ),
            );
          }
        }
      }

      // Get domain info
      const domain = this.reasoningIndex.getDomain(intent.domain);
      if (domain) {
        const trajectory = this.reasoningIndex.getConfidenceTrajectory(
          intent.domain,
        );
        domains.push({
          domainId: intent.domain,
          domain,
          factCount: factResult.totalCount,
          trajectory,
          isGateway: this.gatewayIndex.isGateway(intent.domain),
          gatewayScore: this.gatewayIndex.getGateway(intent.domain)
            ?.gatewayScore,
        });
      }
    }

    return {
      intentType: "understand",
      facts,
      domains,
      trajectory: domains[0]?.trajectory,
      metadata: {
        cached: false,
        bucketsScanned,
        domainsSearched,
        latencyMs: 0,
        cacheHitRatio: 0,
        prefetchHitRatio: 0,
      },
    };
  }

  /**
   * Execute a verify intent.
   */
  private async executeVerifyQuery(
    _session: ReasoningSession,
    intent: AgentIntent,
    currentBlock: bigint,
  ): Promise<IntentResponse> {
    const facts: FactResult[] = [];
    const bucketsScanned = 0;
    const domainsSearched: string[] = [];

    if (intent.factId) {
      // Get the fact and its dependencies
      const fact = await this.loadFact(intent.factId);
      const domainId = this.reasoningIndex.getFactDomain(intent.factId);
      const stratum = this.reasoningIndex.getFactStratum(intent.factId);

      if (fact && domainId && stratum) {
        domainsSearched.push(domainId);
        const domain = this.reasoningIndex.getDomain(domainId);
        if (domain) {
          facts.push(
            this.buildFactResult(
              intent.factId,
              fact,
              domain,
              stratum,
              1.0,
              false,
              false,
              currentBlock,
            ),
          );
        }

        // Get transitive dependencies for verification context
        const deps = this.reasoningIndex.getTransitiveDependencies(
          intent.factId,
          3,
        );
        for (const depFactId of deps.dependencies.slice(0, 10)) {
          const depFact = await this.loadFact(depFactId);
          const depDomainId = this.reasoningIndex.getFactDomain(depFactId);
          const depStratum = this.reasoningIndex.getFactStratum(depFactId);

          if (depFact && depDomainId && depStratum) {
            const depDomain = this.reasoningIndex.getDomain(depDomainId);
            if (depDomain) {
              facts.push(
                this.buildFactResult(
                  depFactId,
                  depFact,
                  depDomain,
                  depStratum,
                  0.8,
                  false,
                  false,
                  currentBlock,
                ),
              );
            }
          }
        }
      }
    }

    return {
      intentType: "verify",
      facts,
      domains: [],
      metadata: {
        cached: false,
        bucketsScanned,
        domainsSearched,
        latencyMs: 0,
        cacheHitRatio: 0,
        prefetchHitRatio: 0,
      },
    };
  }

  /**
   * Execute an explore intent.
   */
  private async executeExploreQuery(
    session: ReasoningSession,
    intent: AgentIntent,
    currentBlock: bigint,
  ): Promise<IntentResponse> {
    const facts: FactResult[] = [];
    const domains: DomainResult[] = [];
    let bucketsScanned = 0;
    const domainsSearched: string[] = [];

    if (intent.domain) {
      const sourceDomain = this.reasoningIndex.getDomain(intent.domain);
      if (!sourceDomain) {
        return this.createEmptyResponse("explore");
      }

      domainsSearched.push(intent.domain);

      // Based on direction, explore related domains
      let relatedDomainIds: string[] = [];

      switch (intent.direction) {
        case "descendants":
          relatedDomainIds = sourceDomain.childDomains || [];
          break;

        case "ancestors":
          if (sourceDomain.parentDomain) {
            relatedDomainIds = [sourceDomain.parentDomain];
          }
          break;

        case "siblings":
          if (sourceDomain.parentDomain) {
            const parent = this.reasoningIndex.getDomain(
              sourceDomain.parentDomain,
            );
            relatedDomainIds = (parent?.childDomains || []).filter(
              (id) => id !== intent.domain,
            );
          }
          break;

        default:
          // Default: children + siblings
          relatedDomainIds = sourceDomain.childDomains || [];
          if (sourceDomain.parentDomain) {
            const parent = this.reasoningIndex.getDomain(
              sourceDomain.parentDomain,
            );
            relatedDomainIds = relatedDomainIds.concat(
              (parent?.childDomains || []).filter((id) => id !== intent.domain),
            );
          }
      }

      // Get top facts from each related domain
      for (const relatedId of relatedDomainIds.slice(0, 5)) {
        domainsSearched.push(relatedId);
        const relatedDomain = this.reasoningIndex.getDomain(relatedId);
        if (!relatedDomain) continue;

        const factResult = this.reasoningIndex.getFactsInDomain(relatedId, {
          minConfidence: intent.minConfidence ?? 0.5,
          limit: 5,
        });
        bucketsScanned += factResult.bucketsScanned;

        // Add domain
        const trajectory =
          this.reasoningIndex.getConfidenceTrajectory(relatedId);
        domains.push({
          domainId: relatedId,
          domain: relatedDomain,
          factCount: factResult.totalCount,
          trajectory,
          isGateway: this.gatewayIndex.isGateway(relatedId),
          gatewayScore: this.gatewayIndex.getGateway(relatedId)?.gatewayScore,
        });

        // Add facts
        for (const factId of factResult.factIds) {
          const fact = await this.loadFact(factId);
          const stratum = this.reasoningIndex.getFactStratum(factId);
          if (fact && stratum) {
            facts.push(
              this.buildFactResult(
                factId,
                fact,
                relatedDomain,
                stratum,
                fact.confidence * 0.8,
                false,
                session.prefetchedDomains.has(relatedId),
                currentBlock,
              ),
            );
          }
        }
      }
    }

    return {
      intentType: "explore",
      facts,
      domains,
      metadata: {
        cached: false,
        bucketsScanned,
        domainsSearched,
        latencyMs: 0,
        cacheHitRatio: 0,
        prefetchHitRatio: 0,
      },
    };
  }

  /**
   * Execute a connect intent (cross-stratum).
   */
  private async executeConnectQuery(
    _session: ReasoningSession,
    intent: AgentIntent,
    currentBlock: bigint,
  ): Promise<IntentResponse> {
    const facts: FactResult[] = [];
    const domains: DomainResult[] = [];
    const paths: PathResult[] = [];
    let bucketsScanned = 0;
    const domainsSearched: string[] = [];

    if (intent.stratum) {
      // Find gateway domains for cross-stratum reasoning
      const gateways = this.gatewayIndex.getGatewaysForStratum(intent.stratum);

      for (const gateway of gateways.slice(0, 5)) {
        domainsSearched.push(gateway.domainId);

        // Add gateway domain
        const trajectory = this.reasoningIndex.getConfidenceTrajectory(
          gateway.domainId,
        );
        domains.push({
          domainId: gateway.domainId,
          domain: gateway.domain,
          factCount: gateway.crossStratumFactCount,
          trajectory,
          isGateway: true,
          gatewayScore: gateway.gatewayScore,
        });

        // Get cross-stratum facts
        const factResult = this.reasoningIndex.getFactsInDomain(
          gateway.domainId,
          {
            minConfidence: intent.minConfidence ?? 0.6,
            limit: 10,
          },
        );
        bucketsScanned += factResult.bucketsScanned;

        for (const factId of factResult.factIds) {
          const fact = await this.loadFact(factId);
          const stratum = this.reasoningIndex.getFactStratum(factId);
          if (fact && stratum) {
            facts.push(
              this.buildFactResult(
                factId,
                fact,
                gateway.domain,
                stratum,
                fact.confidence * gateway.gatewayScore,
                false,
                false,
                currentBlock,
              ),
            );
          }
        }

        // Build path info
        paths.push({
          fromDomain: gateway.domainId,
          toDomain: gateway.connectsTo[0] || gateway.domainId,
          gateways: [gateway.domainId],
          facts: factResult.factIds,
          strength: gateway.gatewayScore,
          strataTraversed: gateway.bridgesStrata,
        });
      }
    }

    return {
      intentType: "connect",
      facts,
      domains,
      paths,
      metadata: {
        cached: false,
        bucketsScanned,
        domainsSearched,
        latencyMs: 0,
        cacheHitRatio: 0,
        prefetchHitRatio: 0,
      },
    };
  }

  /**
   * GAP-004: Execute a semantic search query.
   *
   * Uses the SemanticSimilarityIndex to find facts similar to the query text.
   * This enables natural language knowledge discovery.
   */
  private async executeSemanticQuery(
    session: ReasoningSession,
    intent: AgentIntent,
    currentBlock: bigint,
  ): Promise<IntentResponse> {
    if (!this.semanticIndex || !intent.query) {
      return this.createEmptyResponse("search");
    }

    const facts: FactResult[] = [];
    const domainsSearched: string[] = [];

    // Build query options from intent
    const options: SemanticQueryOptions = {
      domain: intent.domain,
      stratum: intent.stratum,
      minSimilarity: intent.minConfidence ?? 0.7,
      limit: intent.limit ?? 20,
    };

    // Execute semantic search
    const matches = await this.semanticIndex.searchByText(
      intent.query,
      options,
    );

    // Convert SemanticMatch[] to FactResult[]
    for (const match of matches) {
      const fact = await this.loadFact(match.factId);
      if (fact) {
        const domain = this.reasoningIndex.getDomain(match.domain);
        if (domain) {
          facts.push(
            this.buildFactResult(
              match.factId,
              fact,
              domain,
              match.stratum,
              match.similarityScore,
              false,
              session.prefetchedDomains.has(match.domain),
              currentBlock,
            ),
          );

          if (!domainsSearched.includes(match.domain)) {
            domainsSearched.push(match.domain);
          }
        }
      }
    }

    return {
      intentType: "search",
      facts,
      domains: [],
      metadata: {
        cached: false,
        bucketsScanned: 0, // Semantic search doesn't use confidence buckets
        domainsSearched,
        latencyMs: 0,
        cacheHitRatio: 0,
        prefetchHitRatio: 0,
      },
    };
  }

  private createEmptyResponse(intentType: string): IntentResponse {
    return {
      intentType: intentType as any,
      facts: [],
      domains: [],
      metadata: {
        cached: false,
        bucketsScanned: 0,
        domainsSearched: [],
        latencyMs: 0,
        cacheHitRatio: 0,
        prefetchHitRatio: 0,
      },
    };
  }

  // ============================================================================
  // Prefetch
  // ============================================================================

  /**
   * Trigger prefetch for predicted next domains.
   */
  private triggerPrefetch(
    session: ReasoningSession,
    intent: AgentIntent,
  ): void {
    if (!this.factLoader || !this.domainLoader) {
      return;
    }

    // Fire and forget - don't await
    this.prefetchEngine
      .triggerPrefetch(
        session,
        intent.type,
        intent.domain,
        {
          depth: intent.depth,
          direction: intent.direction,
          stratum: intent.stratum,
        },
        this.factLoader,
        this.domainLoader,
      )
      .catch(() => {
        // Prefetch failures are silent
      });
  }

  // ============================================================================
  // Helpers
  // ============================================================================

  /**
   * GAP-010: Determine rate limit tier for a session.
   * Uses explicit tier if set, otherwise derives from budget.
   */
  private determineRateLimitTier(session: AgentChatSession): RateLimitTier {
    // Explicit tier takes precedence
    if (session.rateLimitTier) {
      return session.rateLimitTier;
    }

    // Otherwise, derive from budget
    return determineTierFromBudget(session.budgetLimit);
  }

  /**
   * Load a fact using the injected loader.
   */
  private async loadFact(factId: string): Promise<FactState | undefined> {
    if (!this.factLoader) {
      return undefined;
    }
    return this.factLoader(factId);
  }

  /**
   * Load a domain using the injected loader.
   * @internal Reserved for future use in domain-centric queries
   */
  // @ts-expect-error Reserved for future domain-centric queries
  private async _loadDomain(
    domainId: string,
  ): Promise<OntologicalDomain | undefined> {
    if (!this.domainLoader) {
      return this.reasoningIndex.getDomain(domainId);
    }
    return this.domainLoader(domainId);
  }

  // ============================================================================
  // GAP-008: Confidence Decay
  // ============================================================================

  /**
   * Calculate effective (time-decayed) confidence for a fact.
   *
   * GAP-008: Lazy decay calculation - doesn't modify stored values.
   * Returns both effective confidence and full decay info.
   *
   * @param fact - The fact to calculate decay for
   * @param currentBlock - Current block number
   * @returns Decay information including effective confidence
   */
  private calculateDecayInfo(fact: FactState, currentBlock: bigint): DecayInfo {
    const blocksSinceVerification = Number(currentBlock - fact.verifiedAt);

    // Map stratum to category for decay calculation
    const stratum = fact.stratum as OntologicalStratum | undefined;
    const category = stratum
      ? (STRATUM_TO_DECAY_CATEGORY[stratum] ?? "empirical")
      : "empirical";

    // Use category-aware decay from consensus module
    const decay = calculateCategoryDecayedConfidence(
      fact.confidence,
      blocksSinceVerification,
      category,
    );

    return {
      storedConfidence: fact.confidence,
      effectiveConfidence: decay.confidence,
      decayFactor: decay.decayFactor,
      isEternal: decay.isEternal,
      blocksSinceVerification,
      category,
    };
  }

  /**
   * Build a FactResult with decay information.
   *
   * GAP-008: Helper to consistently include effective confidence in results.
   */
  private buildFactResult(
    factId: string,
    fact: FactState,
    domain: OntologicalDomain,
    stratum: OntologicalStratum,
    relevance: number,
    cached: boolean,
    prefetched: boolean,
    currentBlock: bigint,
  ): FactResult {
    const decayInfo = this.calculateDecayInfo(fact, currentBlock);

    return {
      factId,
      fact,
      domain,
      stratum,
      relevance,
      cached,
      prefetched,
      effectiveConfidence: decayInfo.effectiveConfidence,
      decayInfo,
    };
  }

  // ============================================================================
  // Statistics
  // ============================================================================

  /**
   * Get service statistics.
   */
  getStatistics(): {
    activeSessions: number;
    totalQueries: number;
    totalCost: bigint;
    indexStats: ReturnType<DomainReasoningIndex["getStatistics"]>;
    gatewayStats: ReturnType<GatewayDomainIndex["getStatistics"]>;
  } {
    let totalQueries = 0;
    let totalCost = 0n;

    for (const session of this.chatSessions.values()) {
      totalQueries += session.queryCount;
      totalCost += session.totalCost;
    }

    return {
      activeSessions: this.chatSessions.size,
      totalQueries,
      totalCost,
      indexStats: this.reasoningIndex.getStatistics(),
      gatewayStats: this.gatewayIndex.getStatistics(),
    };
  }

  /**
   * Cleanup stale sessions.
   */
  cleanupStaleSessions(): number {
    const now = Date.now();
    const timeout = this.config.sessionTimeoutMs;
    let cleaned = 0;

    for (const [sessionId, session] of this.chatSessions) {
      if (now - session.lastActivityAt > timeout) {
        this.endSession(sessionId);
        cleaned++;
      }
    }

    // Also cleanup reasoning sessions
    this.sessionManager.cleanupStaleSessions();

    return cleaned;
  }

  /**
   * Shutdown the service.
   */
  shutdown(): void {
    // End all sessions
    for (const sessionId of this.chatSessions.keys()) {
      this.endSession(sessionId);
    }

    // Shutdown session manager
    this.sessionManager.shutdown();
  }
}
