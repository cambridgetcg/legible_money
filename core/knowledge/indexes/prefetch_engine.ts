/**
 * Prefetch Engine
 *
 * Predictive prefetching for AI agent reasoning sessions.
 * Analyzes query patterns to prefetch likely-needed domains
 * before the agent requests them.
 *
 * Prefetch Strategies:
 * 1. ExploreIntent → prefetch siblings and children
 * 2. UnderstandIntent (deep) → prefetch ancestors
 * 3. Cross-stratum interest → prefetch gateway domains
 * 4. Sequential pattern → prefetch next in sequence
 *
 * PP HIERARCHY:
 *   TRUTH: Predictions grounded in actual query patterns
 *   UNDERSTANDING: Anticipates agent's reasoning needs
 *   BEAUTY: Elegant pattern matching for prediction
 *   JUSTICE: Prefetch benefits agent without wasting resources
 *   CREATIVITY: Novel predictive optimization for reasoning
 */

import type { FactState } from "../../../state/types.js";
import type {
  OntologicalStratum,
  OntologicalDomain,
} from "../ontology/types.js";
import {
  type ReasoningSession,
  type QueryIntentType,
  type DomainPrediction,
  type ReasoningIndexConfig,
  DEFAULT_REASONING_INDEX_CONFIG,
} from "./types.js";
import { DomainReasoningIndex } from "./reasoning_index.js";
import { GatewayDomainIndex } from "./gateway_index.js";
import { ReasoningSessionManager } from "./session_cache.js";

// ============================================================================
// Prefetch Engine
// ============================================================================

/**
 * Predictive prefetching engine for reasoning sessions.
 */
export class PrefetchEngine {
  private config: ReasoningIndexConfig;
  private reasoningIndex: DomainReasoningIndex;
  private gatewayIndex: GatewayDomainIndex;
  private sessionManager: ReasoningSessionManager;

  constructor(
    reasoningIndex: DomainReasoningIndex,
    gatewayIndex: GatewayDomainIndex,
    sessionManager: ReasoningSessionManager,
    config: ReasoningIndexConfig = DEFAULT_REASONING_INDEX_CONFIG,
  ) {
    this.reasoningIndex = reasoningIndex;
    this.gatewayIndex = gatewayIndex;
    this.sessionManager = sessionManager;
    this.config = { ...config };
  }

  // ============================================================================
  // Prediction
  // ============================================================================

  /**
   * Predict next domains agent might need based on current query.
   */
  predictNextDomains(
    session: ReasoningSession,
    currentIntent: QueryIntentType,
    currentDomain?: string,
    metadata?: {
      depth?: number;
      direction?: string;
      stratum?: OntologicalStratum;
    },
  ): DomainPrediction[] {
    const predictions: DomainPrediction[] = [];
    const pattern = session.queryPattern;

    // Strategy 1: Based on intent type
    switch (currentIntent) {
      case "explore":
        if (currentDomain) {
          predictions.push(
            ...this.predictForExplore(currentDomain, metadata?.direction),
          );
        }
        break;

      case "understand":
        if (currentDomain) {
          predictions.push(
            ...this.predictForUnderstand(currentDomain, metadata?.depth),
          );
        }
        break;

      case "connect":
        // For connect, predict gateway domains
        predictions.push(...this.predictGateways(pattern));
        break;

      case "verify":
        if (currentDomain) {
          // For verify, predict related domains for context
          predictions.push(...this.predictRelated(currentDomain));
        }
        break;
    }

    // Strategy 2: Based on sequential pattern
    predictions.push(...this.predictFromSequence(pattern.domainSequence));

    // Strategy 3: Based on stratum interest
    predictions.push(...this.predictFromStratumInterest(pattern));

    // Deduplicate and filter out already-working domains
    const seen = new Set<string>();
    const filtered: DomainPrediction[] = [];

    for (const pred of predictions) {
      if (
        !seen.has(pred.domainId) &&
        !session.workingDomains.has(pred.domainId) &&
        !session.prefetchedDomains.has(pred.domainId)
      ) {
        seen.add(pred.domainId);
        filtered.push(pred);
      }
    }

    // Sort by confidence and limit
    return filtered
      .sort((a, b) => b.confidence - a.confidence)
      .slice(0, this.config.prefetchCount);
  }

  /**
   * Predict domains for explore intent.
   */
  private predictForExplore(
    domainId: string,
    direction?: string,
  ): DomainPrediction[] {
    const predictions: DomainPrediction[] = [];
    const domain = this.reasoningIndex.getDomain(domainId);
    if (!domain) return predictions;

    // Predict children (most likely for exploration)
    if (domain.childDomains) {
      for (const childId of domain.childDomains.slice(0, 3)) {
        predictions.push({
          domainId: childId,
          confidence: 0.8,
          reason: "child",
        });
      }
    }

    // Predict siblings (domains with same parent)
    if (domain.parentDomain) {
      const parent = this.reasoningIndex.getDomain(domain.parentDomain);
      if (parent?.childDomains) {
        for (const siblingId of parent.childDomains) {
          if (siblingId !== domainId) {
            predictions.push({
              domainId: siblingId,
              confidence: 0.7,
              reason: "sibling",
            });
          }
        }
      }
    }

    // If exploring causes/effects, predict cross-stratum
    if (direction === "causes" || direction === "effects") {
      const gateways = this.gatewayIndex.getGatewaysForStratum(domain.stratum);
      for (const gateway of gateways.slice(0, 2)) {
        predictions.push({
          domainId: gateway.domainId,
          confidence: 0.6,
          reason: "gateway",
        });
      }
    }

    return predictions;
  }

  /**
   * Predict domains for understand intent.
   */
  private predictForUnderstand(
    domainId: string,
    depth?: number,
  ): DomainPrediction[] {
    const predictions: DomainPrediction[] = [];
    const domain = this.reasoningIndex.getDomain(domainId);
    if (!domain) return predictions;

    // Deep understanding needs ancestors for context
    if (depth !== undefined && depth >= 3) {
      // Predict parent chain
      let currentDomain = domain;
      let currentConfidence = 0.7;

      while (currentDomain.parentDomain && currentConfidence > 0.3) {
        predictions.push({
          domainId: currentDomain.parentDomain,
          confidence: currentConfidence,
          reason: "ancestor",
        });
        currentDomain =
          this.reasoningIndex.getDomain(currentDomain.parentDomain) ??
          currentDomain;
        currentConfidence -= 0.15;
      }
    }

    // Also predict some related high-confidence domains
    predictions.push(...this.predictRelated(domainId));

    return predictions;
  }

  /**
   * Predict related domains.
   */
  private predictRelated(domainId: string): DomainPrediction[] {
    const predictions: DomainPrediction[] = [];
    const domain = this.reasoningIndex.getDomain(domainId);
    if (!domain) return predictions;

    // Get facts in this domain
    const factsResult = this.reasoningIndex.getFactsInDomain(domainId, {
      minConfidence: 0.7,
      limit: 10,
    });

    // For each fact, check dependencies
    for (const factId of factsResult.factIds.slice(0, 5)) {
      const deps = this.reasoningIndex.getTransitiveDependencies(factId, 2);
      for (const depFactId of deps.dependencies.slice(0, 2)) {
        const depDomain = this.reasoningIndex.getFactDomain(depFactId);
        if (depDomain && depDomain !== domainId) {
          predictions.push({
            domainId: depDomain,
            confidence: 0.5,
            reason: "related",
          });
        }
      }
    }

    return predictions;
  }

  /**
   * Predict gateways based on stratum interest pattern.
   */
  private predictGateways(
    pattern: ReasoningSession["queryPattern"],
  ): DomainPrediction[] {
    const predictions: DomainPrediction[] = [];

    // Find most interested strata
    const sortedStrata = Array.from(pattern.stratumInterest.entries())
      .sort((a, b) => b[1] - a[1])
      .slice(0, 2);

    if (sortedStrata.length >= 2) {
      // Find gateways between top two strata of interest
      const result = this.gatewayIndex.findGatewaysBetweenStrata(
        sortedStrata[0][0],
        sortedStrata[1][0],
      );

      for (const gateway of result.gateways.slice(0, 3)) {
        predictions.push({
          domainId: gateway.domainId,
          confidence: 0.6 * gateway.gatewayScore,
          reason: "gateway",
        });
      }
    } else if (sortedStrata.length === 1) {
      // Get gateways for the interested stratum
      const gateways = this.gatewayIndex.getGatewaysForStratum(
        sortedStrata[0][0],
      );
      for (const gateway of gateways.slice(0, 3)) {
        predictions.push({
          domainId: gateway.domainId,
          confidence: 0.5 * gateway.gatewayScore,
          reason: "gateway",
        });
      }
    }

    return predictions;
  }

  /**
   * Predict from domain access sequence (Markov-like).
   */
  private predictFromSequence(sequence: string[]): DomainPrediction[] {
    const predictions: DomainPrediction[] = [];
    if (sequence.length < 3) return predictions;

    // Simple bigram model: what domain typically follows after domain X?
    const transitions: Map<string, Map<string, number>> = new Map();

    for (let i = 0; i < sequence.length - 1; i++) {
      const from = sequence[i];
      const to = sequence[i + 1];

      let fromMap = transitions.get(from);
      if (!fromMap) {
        fromMap = new Map();
        transitions.set(from, fromMap);
      }

      const count = fromMap.get(to) ?? 0;
      fromMap.set(to, count + 1);
    }

    // Get predictions based on current domain
    const current = sequence[sequence.length - 1];
    const nextCounts = transitions.get(current);

    if (nextCounts) {
      const total = Array.from(nextCounts.values()).reduce((a, b) => a + b, 0);
      for (const [nextDomain, count] of nextCounts) {
        if (nextDomain !== current) {
          predictions.push({
            domainId: nextDomain,
            confidence: (count / total) * 0.5, // Scale down pattern-based predictions
            reason: "pattern",
          });
        }
      }
    }

    return predictions;
  }

  /**
   * Predict based on stratum interest.
   */
  private predictFromStratumInterest(
    pattern: ReasoningSession["queryPattern"],
  ): DomainPrediction[] {
    const predictions: DomainPrediction[] = [];

    // Get top stratum of interest
    const topStratum = Array.from(pattern.stratumInterest.entries()).sort(
      (a, b) => b[1] - a[1],
    )[0];

    if (topStratum) {
      // Get facts in that stratum with high confidence
      const facts = this.reasoningIndex.getFactsInStratum(topStratum[0], {
        minConfidence: 0.8,
        limit: 10,
      });

      // Group by domain and pick top domains
      const domainCounts: Map<string, number> = new Map();
      for (const factId of facts) {
        const domain = this.reasoningIndex.getFactDomain(factId);
        if (domain) {
          domainCounts.set(domain, (domainCounts.get(domain) ?? 0) + 1);
        }
      }

      for (const [domainId, count] of domainCounts) {
        predictions.push({
          domainId,
          confidence: Math.min(0.4, count * 0.1), // Low confidence for general interest
          reason: "related",
        });
      }
    }

    return predictions;
  }

  // ============================================================================
  // Prefetch Execution
  // ============================================================================

  /**
   * Execute prefetch for predicted domains.
   * Returns the domains that were prefetched.
   */
  async executePrefetch(
    session: ReasoningSession,
    predictions: DomainPrediction[],
    loadFact: (factId: string) => Promise<FactState | undefined>,
    loadDomain: (domainId: string) => Promise<OntologicalDomain | undefined>,
  ): Promise<string[]> {
    const prefetched: string[] = [];

    for (const prediction of predictions) {
      try {
        // Load domain
        const domain = await loadDomain(prediction.domainId);
        if (!domain) continue;

        // Get top facts in domain
        const topFacts = this.reasoningIndex.getTopFactsInDomain(
          prediction.domainId,
          5,
        );

        // Load facts into session cache
        for (const factId of topFacts) {
          const fact = await loadFact(factId);
          if (fact) {
            this.sessionManager.cacheFact(
              session,
              factId,
              fact,
              domain,
              domain.stratum,
            );
          }
        }

        // Cache domain with trajectory
        const trajectory = this.reasoningIndex.getConfidenceTrajectory(
          prediction.domainId,
        );
        const factCount = this.reasoningIndex.getFactsInDomain(
          prediction.domainId,
        ).totalCount;
        this.sessionManager.cacheDomain(
          session,
          prediction.domainId,
          domain,
          factCount,
          trajectory,
        );

        prefetched.push(prediction.domainId);
      } catch {
        // Prefetch failures are silent - not critical
      }
    }

    // Mark as prefetched in session
    this.sessionManager.markPrefetched(session, prefetched);

    // Update prefetch hit stats (will be calculated on actual access)
    return prefetched;
  }

  /**
   * Trigger prefetch based on query.
   */
  async triggerPrefetch(
    session: ReasoningSession,
    intentType: QueryIntentType,
    currentDomain?: string,
    metadata?: {
      depth?: number;
      direction?: string;
      stratum?: OntologicalStratum;
    },
    loadFact?: (factId: string) => Promise<FactState | undefined>,
    loadDomain?: (domainId: string) => Promise<OntologicalDomain | undefined>,
  ): Promise<string[]> {
    // Predict domains
    const predictions = this.predictNextDomains(
      session,
      intentType,
      currentDomain,
      metadata,
    );

    if (predictions.length === 0) return [];

    // Execute prefetch if loaders provided
    if (loadFact && loadDomain) {
      return this.executePrefetch(session, predictions, loadFact, loadDomain);
    }

    // Just return predictions if no loaders
    return predictions.map((p) => p.domainId);
  }
}
