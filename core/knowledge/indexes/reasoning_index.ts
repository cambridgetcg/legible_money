/**
 * Domain Reasoning Index
 *
 * Optimized index for AI agent access to knowledge during reasoning.
 * Provides O(buckets) confidence-filtered queries, cross-stratum dependency
 * tracking, and efficient domain traversal.
 *
 * Key optimizations:
 * - Confidence-bucketed fact lookups: O(buckets) vs O(n)
 * - Cross-stratum dependency graph: O(depth) BFS traversal
 * - Gateway domain caching: O(1) for pre-computed paths
 * - Incremental maintenance: O(1) per fact mutation
 *
 * PP HIERARCHY:
 *   TRUTH: Index grounded in verified on-chain facts
 *   UNDERSTANDING: Enables deep comprehension of domain relationships
 *   BEAUTY: Elegant O(buckets) query complexity
 *   JUSTICE: Correctly placed in core/knowledge/indexes
 *   CREATIVITY: Novel gateway concept for cross-stratum reasoning
 */

import type {
  OntologicalStratum,
  OntologicalDomain,
} from "../ontology/types.js";
import { STRATUM_HIERARCHY } from "../ontology/types.js";
import {
  type ConfidenceTrajectory,
  type CrossStratumDependency,
  type TransitiveDependencyResult,
  type DependencyPath,
  type FactsByDomainOptions,
  type FactsByDomainResult,
  type DependentFactsOptions,
  type DependentFactsResult,
  type ReasoningIndexConfig,
  DEFAULT_REASONING_INDEX_CONFIG,
  confidenceToBucket,
  computeDomainHealthScore,
} from "./types.js";

// ============================================================================
// Domain Reasoning Index
// ============================================================================

/**
 * DomainReasoningIndex provides optimized queries for AI agent reasoning.
 */
export class DomainReasoningIndex {
  private config: ReasoningIndexConfig;

  // === Primary Indexes ===

  /** Domain → fact IDs (all facts in domain) */
  private factsByDomain: Map<string, Set<string>> = new Map();

  /** Domain → confidence bucket → fact IDs */
  private factsByDomainAndConfidence: Map<string, Map<number, Set<string>>> =
    new Map();

  /** Stratum → fact IDs */
  private factsByStratum: Map<OntologicalStratum, Set<string>> = new Map();

  /** Stratum → confidence bucket → fact IDs */
  private factsByStratumAndConfidence: Map<
    OntologicalStratum,
    Map<number, Set<string>>
  > = new Map();

  // === Cross-Stratum Dependency Index ===

  /** Fact → facts it depends on (lower strata) */
  private dependencies: Map<string, Set<string>> = new Map();

  /** Fact → facts that depend on it (higher strata) */
  private reverseDependencies: Map<string, Set<string>> = new Map();

  /** Fact → dependency metadata */
  private dependencyMetadata: Map<string, CrossStratumDependency[]> = new Map();

  // === Confidence Trajectory Cache ===

  /** Domain → confidence trajectory */
  private trajectoryCache: Map<
    string,
    { trajectory: ConfidenceTrajectory; computedAt: number }
  > = new Map();

  // === Metadata ===

  /** Fact → domain ID */
  private factDomainMap: Map<string, string> = new Map();

  /** Fact → stratum */
  private factStratumMap: Map<string, OntologicalStratum> = new Map();

  /** Domain → domain data */
  private domainData: Map<string, OntologicalDomain> = new Map();

  /** Last update block */
  private lastUpdateBlock: bigint = 0n;

  /** Optional provider for active challenge counts per domain */
  private challengeCountProvider?: (domainId: string) => number;

  constructor(config: ReasoningIndexConfig = DEFAULT_REASONING_INDEX_CONFIG) {
    this.config = { ...config };

    // Initialize stratum indexes
    for (const stratum of Object.keys(
      STRATUM_HIERARCHY,
    ) as OntologicalStratum[]) {
      this.factsByStratum.set(stratum, new Set());
      this.factsByStratumAndConfidence.set(stratum, new Map());
    }
  }

  /**
   * Set a provider function that returns active challenge count for a domain.
   * Called during trajectory computation to feed real dispute data into health score.
   */
  setChallengeCountProvider(provider: (domainId: string) => number): void {
    this.challengeCountProvider = provider;
  }

  // ============================================================================
  // Indexing Operations
  // ============================================================================

  /**
   * Index a domain.
   */
  indexDomain(domain: OntologicalDomain): void {
    this.domainData.set(domain.id, domain);

    // Ensure bucket maps exist
    if (!this.factsByDomainAndConfidence.has(domain.id)) {
      this.factsByDomainAndConfidence.set(domain.id, new Map());
    }
    if (!this.factsByDomain.has(domain.id)) {
      this.factsByDomain.set(domain.id, new Set());
    }
  }

  /**
   * Index a fact in its domain with confidence bucket.
   * O(1) operation.
   */
  indexFact(
    factId: string,
    domainId: string,
    stratum: OntologicalStratum,
    confidence: number,
    blockNumber: bigint,
  ): void {
    const bucket = confidenceToBucket(
      confidence,
      this.config.confidenceBuckets.bucketCount,
    );

    // Store fact metadata
    this.factDomainMap.set(factId, domainId);
    this.factStratumMap.set(factId, stratum);

    // Add to domain → facts
    let domainFacts = this.factsByDomain.get(domainId);
    if (!domainFacts) {
      domainFacts = new Set();
      this.factsByDomain.set(domainId, domainFacts);
    }
    domainFacts.add(factId);

    // Add to domain → bucket → facts
    let domainBuckets = this.factsByDomainAndConfidence.get(domainId);
    if (!domainBuckets) {
      domainBuckets = new Map();
      this.factsByDomainAndConfidence.set(domainId, domainBuckets);
    }
    let bucketFacts = domainBuckets.get(bucket);
    if (!bucketFacts) {
      bucketFacts = new Set();
      domainBuckets.set(bucket, bucketFacts);
    }
    bucketFacts.add(factId);

    // Add to stratum → facts
    const stratumFacts = this.factsByStratum.get(stratum);
    if (stratumFacts) {
      stratumFacts.add(factId);
    }

    // Add to stratum → bucket → facts
    let stratumBuckets = this.factsByStratumAndConfidence.get(stratum);
    if (!stratumBuckets) {
      stratumBuckets = new Map();
      this.factsByStratumAndConfidence.set(stratum, stratumBuckets);
    }
    let stratumBucketFacts = stratumBuckets.get(bucket);
    if (!stratumBucketFacts) {
      stratumBucketFacts = new Set();
      stratumBuckets.set(bucket, stratumBucketFacts);
    }
    stratumBucketFacts.add(factId);

    // Invalidate trajectory cache for this domain
    this.trajectoryCache.delete(domainId);

    this.lastUpdateBlock = blockNumber;
  }

  /**
   * Remove a fact from the index.
   * O(1) operation.
   */
  removeFact(factId: string): void {
    const domainId = this.factDomainMap.get(factId);
    const stratum = this.factStratumMap.get(factId);

    if (domainId) {
      // Remove from domain → facts
      this.factsByDomain.get(domainId)?.delete(factId);

      // Remove from all buckets in domain (we don't know which bucket it's in)
      const domainBuckets = this.factsByDomainAndConfidence.get(domainId);
      if (domainBuckets) {
        for (const bucketFacts of domainBuckets.values()) {
          bucketFacts.delete(factId);
        }
      }

      // Invalidate trajectory cache
      this.trajectoryCache.delete(domainId);
    }

    if (stratum) {
      // Remove from stratum → facts
      this.factsByStratum.get(stratum)?.delete(factId);

      // Remove from all buckets in stratum
      const stratumBuckets = this.factsByStratumAndConfidence.get(stratum);
      if (stratumBuckets) {
        for (const bucketFacts of stratumBuckets.values()) {
          bucketFacts.delete(factId);
        }
      }
    }

    // Remove dependencies
    this.removeDependenciesForFact(factId);

    // Remove metadata
    this.factDomainMap.delete(factId);
    this.factStratumMap.delete(factId);
  }

  /**
   * Update fact confidence (moves between buckets if needed).
   * O(1) operation.
   */
  updateConfidence(
    factId: string,
    oldConfidence: number,
    newConfidence: number,
  ): void {
    const domainId = this.factDomainMap.get(factId);
    const stratum = this.factStratumMap.get(factId);
    if (!domainId || !stratum) return;

    const bucketCount = this.config.confidenceBuckets.bucketCount;
    const oldBucket = confidenceToBucket(oldConfidence, bucketCount);
    const newBucket = confidenceToBucket(newConfidence, bucketCount);

    if (oldBucket !== newBucket) {
      // Remove from old bucket
      this.factsByDomainAndConfidence
        .get(domainId)
        ?.get(oldBucket)
        ?.delete(factId);
      this.factsByStratumAndConfidence
        .get(stratum)
        ?.get(oldBucket)
        ?.delete(factId);

      // Add to new bucket
      const domainBuckets = this.factsByDomainAndConfidence.get(domainId);
      if (domainBuckets) {
        let newBucketFacts = domainBuckets.get(newBucket);
        if (!newBucketFacts) {
          newBucketFacts = new Set();
          domainBuckets.set(newBucket, newBucketFacts);
        }
        newBucketFacts.add(factId);
      }

      const stratumBuckets = this.factsByStratumAndConfidence.get(stratum);
      if (stratumBuckets) {
        let newBucketFacts = stratumBuckets.get(newBucket);
        if (!newBucketFacts) {
          newBucketFacts = new Set();
          stratumBuckets.set(newBucket, newBucketFacts);
        }
        newBucketFacts.add(factId);
      }
    }

    // Invalidate trajectory cache
    this.trajectoryCache.delete(domainId);
  }

  // ============================================================================
  // Cross-Stratum Dependency Operations
  // ============================================================================

  /**
   * Index a cross-stratum dependency.
   */
  indexDependency(dependency: CrossStratumDependency): void {
    const { fromFactId, toFactId } = dependency;

    // Add to forward dependencies
    let deps = this.dependencies.get(fromFactId);
    if (!deps) {
      deps = new Set();
      this.dependencies.set(fromFactId, deps);
    }
    deps.add(toFactId);

    // Add to reverse dependencies
    let reverseDeps = this.reverseDependencies.get(toFactId);
    if (!reverseDeps) {
      reverseDeps = new Set();
      this.reverseDependencies.set(toFactId, reverseDeps);
    }
    reverseDeps.add(fromFactId);

    // Store metadata
    let metadata = this.dependencyMetadata.get(fromFactId);
    if (!metadata) {
      metadata = [];
      this.dependencyMetadata.set(fromFactId, metadata);
    }
    metadata.push(dependency);
  }

  /**
   * Remove all dependencies for a fact.
   */
  private removeDependenciesForFact(factId: string): void {
    // Remove forward dependencies
    const deps = this.dependencies.get(factId);
    if (deps) {
      for (const depId of deps) {
        this.reverseDependencies.get(depId)?.delete(factId);
      }
    }
    this.dependencies.delete(factId);

    // Remove reverse dependencies
    const reverseDeps = this.reverseDependencies.get(factId);
    if (reverseDeps) {
      for (const depId of reverseDeps) {
        this.dependencies.get(depId)?.delete(factId);
      }
    }
    this.reverseDependencies.delete(factId);

    // Remove metadata
    this.dependencyMetadata.delete(factId);
  }

  // ============================================================================
  // Query Operations
  // ============================================================================

  /**
   * Get facts in a domain with optional confidence filtering.
   * Complexity: O(buckets above threshold) instead of O(n).
   */
  getFactsInDomain(
    domainId: string,
    options: FactsByDomainOptions = {},
  ): FactsByDomainResult {
    const {
      minConfidence = 0,
      maxConfidence = 1,
      includeDescendants = false,
      limit,
      offset = 0,
    } = options;

    const bucketCount = this.config.confidenceBuckets.bucketCount;
    const minBucket = confidenceToBucket(minConfidence, bucketCount);
    const maxBucket = confidenceToBucket(maxConfidence, bucketCount);

    const factIds: Set<string> = new Set();
    const domainsIncluded: string[] = [domainId];
    let bucketsScanned = 0;

    // Helper to collect facts from a domain
    const collectFromDomain = (dId: string) => {
      const domainBuckets = this.factsByDomainAndConfidence.get(dId);
      if (!domainBuckets) return;

      for (let bucket = minBucket; bucket <= maxBucket; bucket++) {
        const bucketFacts = domainBuckets.get(bucket);
        if (bucketFacts) {
          bucketsScanned++;
          for (const factId of bucketFacts) {
            factIds.add(factId);
          }
        }
      }
    };

    // Collect from primary domain
    collectFromDomain(domainId);

    // Collect from descendants if requested
    if (includeDescendants) {
      const descendants = this.getDescendantDomains(domainId);
      for (const descId of descendants) {
        domainsIncluded.push(descId);
        collectFromDomain(descId);
      }
    }

    // Convert to array and apply pagination
    let result = Array.from(factIds);
    const totalCount = result.length;

    if (offset > 0) {
      result = result.slice(offset);
    }
    if (limit !== undefined && result.length > limit) {
      result = result.slice(0, limit);
    }

    return {
      factIds: result,
      totalCount,
      domainsIncluded,
      bucketsScanned,
    };
  }

  /**
   * Get facts in a stratum with optional confidence filtering.
   */
  getFactsInStratum(
    stratum: OntologicalStratum,
    options: {
      minConfidence?: number;
      maxConfidence?: number;
      limit?: number;
    } = {},
  ): string[] {
    const { minConfidence = 0, maxConfidence = 1, limit } = options;

    const bucketCount = this.config.confidenceBuckets.bucketCount;
    const minBucket = confidenceToBucket(minConfidence, bucketCount);
    const maxBucket = confidenceToBucket(maxConfidence, bucketCount);

    const factIds: Set<string> = new Set();
    const stratumBuckets = this.factsByStratumAndConfidence.get(stratum);

    if (stratumBuckets) {
      for (let bucket = minBucket; bucket <= maxBucket; bucket++) {
        const bucketFacts = stratumBuckets.get(bucket);
        if (bucketFacts) {
          for (const factId of bucketFacts) {
            factIds.add(factId);
            if (limit !== undefined && factIds.size >= limit) {
              return Array.from(factIds);
            }
          }
        }
      }
    }

    return Array.from(factIds);
  }

  /**
   * Get facts that depend on a domain (BFS traversal).
   * Complexity: O(d) where d = depth of dependency graph.
   */
  getDependentFacts(
    domainId: string,
    options: DependentFactsOptions = {},
  ): DependentFactsResult {
    const {
      maxDepth = this.config.maxDependencyDepth,
      strataFilter,
      minStrength = 0,
      limit,
    } = options;

    const results: Array<{
      factId: string;
      depth: number;
      path: string[];
      stratum: OntologicalStratum;
    }> = [];

    const visited = new Set<string>();

    // Get all facts in the source domain
    const sourceFacts = this.factsByDomain.get(domainId) ?? new Set();

    // BFS from source facts through reverse dependency graph
    const queue: Array<{ factId: string; depth: number; path: string[] }> = [];

    for (const factId of sourceFacts) {
      queue.push({ factId, depth: 0, path: [factId] });
    }

    while (queue.length > 0) {
      const { factId, depth, path } = queue.shift()!;

      if (visited.has(factId) || depth > maxDepth) continue;
      visited.add(factId);

      // Get facts that depend on this fact
      const dependents = this.reverseDependencies.get(factId);
      if (dependents) {
        for (const depFactId of dependents) {
          const depStratum = this.factStratumMap.get(depFactId);
          if (!depStratum) continue;

          // Apply stratum filter
          if (strataFilter && !strataFilter.includes(depStratum)) continue;

          // Apply strength filter
          const metadata = this.dependencyMetadata.get(depFactId);
          const relevantDep = metadata?.find((m) => m.toFactId === factId);
          if (relevantDep && relevantDep.strength < minStrength) continue;

          results.push({
            factId: depFactId,
            depth: depth + 1,
            path: [...path, depFactId],
            stratum: depStratum,
          });

          if (limit !== undefined && results.length >= limit) {
            return { facts: results, totalCount: results.length };
          }

          queue.push({
            factId: depFactId,
            depth: depth + 1,
            path: [...path, depFactId],
          });
        }
      }
    }

    return { facts: results, totalCount: results.length };
  }

  /**
   * Get transitive dependencies of a fact.
   */
  getTransitiveDependencies(
    factId: string,
    maxDepth?: number,
  ): TransitiveDependencyResult {
    const depth = maxDepth ?? this.config.maxDependencyDepth;
    const dependencies: string[] = [];
    const strataTraversed: Set<OntologicalStratum> = new Set();
    const paths: DependencyPath[] = [];
    const visited = new Set<string>();

    const traverse = (
      fId: string,
      currentDepth: number,
      currentPath: string[],
      currentStrength: number,
    ): void => {
      if (visited.has(fId) || currentDepth > depth) return;
      visited.add(fId);

      const deps = this.dependencies.get(fId);
      if (deps) {
        for (const depFactId of deps) {
          dependencies.push(depFactId);

          const depStratum = this.factStratumMap.get(depFactId);
          if (depStratum) {
            strataTraversed.add(depStratum);
          }

          const newPath = [...currentPath, depFactId];
          const metadata = this.dependencyMetadata.get(fId);
          const depMeta = metadata?.find((m) => m.toFactId === depFactId);
          const newStrength = currentStrength * (depMeta?.strength ?? 1);

          paths.push({
            facts: newPath,
            cumulativeStrength: newStrength,
            strata: newPath
              .map((f) => this.factStratumMap.get(f))
              .filter((s): s is OntologicalStratum => s !== undefined),
          });

          traverse(depFactId, currentDepth + 1, newPath, newStrength);
        }
      }
    };

    traverse(factId, 0, [factId], 1);

    return {
      dependencies: [...new Set(dependencies)],
      strataTraversed: Array.from(strataTraversed),
      maxDepthReached: visited.size >= depth,
      paths,
    };
  }

  // ============================================================================
  // Confidence Trajectory
  // ============================================================================

  /**
   * Get confidence trajectory for a domain.
   * Cached with TTL.
   */
  getConfidenceTrajectory(
    domainId: string,
    factConfidences?: Map<string, number>,
    activeChallengeCount?: number,
  ): ConfidenceTrajectory {
    // Check cache
    const cached = this.trajectoryCache.get(domainId);
    const ttl = this.config.confidenceBuckets.cacheTtlMs;
    if (cached && Date.now() - cached.computedAt < ttl) {
      return cached.trajectory;
    }

    // Compute trajectory
    const facts = this.factsByDomain.get(domainId);
    if (!facts || facts.size === 0) {
      return this.createEmptyTrajectory(domainId);
    }

    // Collect confidence values
    const confidences: number[] = [];
    if (factConfidences) {
      for (const factId of facts) {
        const conf = factConfidences.get(factId);
        if (conf !== undefined) {
          confidences.push(conf);
        }
      }
    }

    // If no confidences provided, estimate from bucket distribution
    if (confidences.length === 0) {
      const domainBuckets = this.factsByDomainAndConfidence.get(domainId);
      if (domainBuckets) {
        const bucketCount = this.config.confidenceBuckets.bucketCount;
        for (const [bucket, bucketFacts] of domainBuckets) {
          const midpoint = (bucket + 0.5) / bucketCount;
          for (let i = 0; i < bucketFacts.size; i++) {
            confidences.push(midpoint);
          }
        }
      }
    }

    if (confidences.length === 0) {
      return this.createEmptyTrajectory(domainId);
    }

    // Calculate statistics
    const avgConfidence =
      confidences.reduce((a, b) => a + b, 0) / confidences.length;
    const variance =
      confidences.reduce((sum, c) => sum + Math.pow(c - avgConfidence, 2), 0) /
      confidences.length;
    const confidenceStdDev = Math.sqrt(variance);

    // Get history and calculate trend
    const existingHistory = cached?.trajectory.history ?? [];
    const trend =
      existingHistory.length >= 2
        ? existingHistory[existingHistory.length - 1].avgConfidence -
          existingHistory[0].avgConfidence
        : 0;

    // Use caller-provided count, injected provider, or 0 as last resort
    const activeChallenges =
      activeChallengeCount ?? this.challengeCountProvider?.(domainId) ?? 0;

    const healthScore = computeDomainHealthScore(
      avgConfidence,
      confidenceStdDev,
      trend,
      activeChallenges,
    );

    const trajectory: ConfidenceTrajectory = {
      domainId,
      avgConfidence,
      confidenceStdDev,
      trend,
      activeChallenges,
      history: existingHistory,
      healthScore,
      computedAt: Date.now(),
    };

    // Cache result
    this.trajectoryCache.set(domainId, {
      trajectory,
      computedAt: Date.now(),
    });

    return trajectory;
  }

  /**
   * Sample trajectory history (called periodically).
   */
  sampleTrajectoryHistory(
    domainId: string,
    blockNumber: bigint,
    avgConfidence: number,
  ): void {
    const cached = this.trajectoryCache.get(domainId);
    const history = cached?.trajectory.history ?? [];

    history.push({ block: blockNumber, avgConfidence });

    // Trim to max history
    while (history.length > this.config.maxTrajectoryHistory) {
      history.shift();
    }

    if (cached) {
      cached.trajectory.history = history;
    }
  }

  private createEmptyTrajectory(domainId: string): ConfidenceTrajectory {
    return {
      domainId,
      avgConfidence: 0,
      confidenceStdDev: 0,
      trend: 0,
      activeChallenges: 0,
      history: [],
      healthScore: 0,
      computedAt: Date.now(),
    };
  }

  // ============================================================================
  // Helper Methods
  // ============================================================================

  /**
   * Get all descendant domain IDs (BFS).
   */
  private getDescendantDomains(domainId: string): string[] {
    const descendants: string[] = [];
    const domain = this.domainData.get(domainId);
    if (!domain) return descendants;

    const queue = [...(domain.childDomains || [])];
    const visited = new Set<string>();

    while (queue.length > 0) {
      const childId = queue.shift()!;
      if (visited.has(childId)) continue;
      visited.add(childId);
      descendants.push(childId);

      const childDomain = this.domainData.get(childId);
      if (childDomain?.childDomains) {
        queue.push(...childDomain.childDomains);
      }
    }

    return descendants;
  }

  /**
   * Get fact's stratum.
   */
  getFactStratum(factId: string): OntologicalStratum | undefined {
    return this.factStratumMap.get(factId);
  }

  /**
   * Get fact's domain.
   */
  getFactDomain(factId: string): string | undefined {
    return this.factDomainMap.get(factId);
  }

  /**
   * Get domain data.
   */
  getDomain(domainId: string): OntologicalDomain | undefined {
    return this.domainData.get(domainId);
  }

  /**
   * Get top facts in domain by confidence bucket.
   */
  getTopFactsInDomain(domainId: string, count: number): string[] {
    const domainBuckets = this.factsByDomainAndConfidence.get(domainId);
    if (!domainBuckets) return [];

    const result: string[] = [];
    const bucketCount = this.config.confidenceBuckets.bucketCount;

    // Start from highest bucket
    for (
      let bucket = bucketCount - 1;
      bucket >= 0 && result.length < count;
      bucket--
    ) {
      const bucketFacts = domainBuckets.get(bucket);
      if (bucketFacts) {
        for (const factId of bucketFacts) {
          result.push(factId);
          if (result.length >= count) break;
        }
      }
    }

    return result;
  }

  // ============================================================================
  // Statistics
  // ============================================================================

  /**
   * Get index statistics.
   */
  getStatistics(): {
    totalFacts: number;
    totalDomains: number;
    factsByStratum: Record<string, number>;
    dependencyCount: number;
    lastUpdateBlock: bigint;
  } {
    const factsByStratum: Record<string, number> = {};
    for (const [stratum, facts] of this.factsByStratum) {
      factsByStratum[stratum] = facts.size;
    }

    let totalFacts = 0;
    for (const facts of this.factsByDomain.values()) {
      totalFacts += facts.size;
    }

    let dependencyCount = 0;
    for (const deps of this.dependencies.values()) {
      dependencyCount += deps.size;
    }

    return {
      totalFacts,
      totalDomains: this.domainData.size,
      factsByStratum,
      dependencyCount,
      lastUpdateBlock: this.lastUpdateBlock,
    };
  }

  /**
   * Clear the index (for testing).
   */
  clear(): void {
    this.factsByDomain.clear();
    this.factsByDomainAndConfidence.clear();
    this.factsByStratum.clear();
    this.factsByStratumAndConfidence.clear();
    this.dependencies.clear();
    this.reverseDependencies.clear();
    this.dependencyMetadata.clear();
    this.trajectoryCache.clear();
    this.factDomainMap.clear();
    this.factStratumMap.clear();
    this.domainData.clear();

    // Re-initialize stratum indexes
    for (const stratum of Object.keys(
      STRATUM_HIERARCHY,
    ) as OntologicalStratum[]) {
      this.factsByStratum.set(stratum, new Set());
      this.factsByStratumAndConfidence.set(stratum, new Map());
    }
  }
}
