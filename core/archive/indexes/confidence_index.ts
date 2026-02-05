/**
 * Confidence Ranking Index
 *
 * Index for efficient confidence-based queries. Uses bucketed keys
 * for range queries and supports filtering by domain and stratum.
 */

import type { ConfidenceBucketEntry } from "../types.js";
import type { OntologicalStratum } from "../../knowledge/ontology/types.js";
import type { FactState } from "../../../state/types.js";
import {
  confidenceToBucket,
  getBucketRange,
  getBucketsInRange,
  makeConfidenceKey,
  DEFAULT_ARCHIVE_CONFIG,
  type ArchiveConfig,
} from "../config.js";

// ============================================================================
// Confidence Index Types
// ============================================================================

/**
 * Options for confidence queries.
 */
export interface ConfidenceQueryOptions {
  /** Minimum confidence threshold */
  minConfidence?: number;

  /** Maximum confidence threshold */
  maxConfidence?: number;

  /** Domain filter */
  domain?: string;

  /** Stratum filter */
  stratum?: OntologicalStratum;

  /** Maximum results */
  limit?: number;

  /** Sort order */
  sortOrder?: "asc" | "desc";

  /** Cursor for pagination */
  cursor?: string;
}

/**
 * Entry in the confidence index.
 */
export interface ConfidenceIndexEntry {
  /** Fact ID */
  factId: string;

  /** Confidence value */
  confidence: number;

  /** Confidence bucket */
  bucket: number;

  /** Domain ID */
  domainId: string;

  /** Stratum */
  stratum: OntologicalStratum;

  /** Block when indexed */
  indexedAt: bigint;
}

// ============================================================================
// Confidence Ranking Index
// ============================================================================

/**
 * Index for confidence-based queries.
 *
 * Key structures:
 * - conf:<bucket>:<fact_id> → ConfidenceIndexEntry
 * - conf_domain:<domain>:<bucket>:<fact_id> → factId
 * - conf_stratum:<stratum>:<bucket>:<fact_id> → factId
 */
export class ConfidenceRankingIndex {
  private config: ArchiveConfig;

  // In-memory indexes (would be backed by LevelDB in production)
  private globalIndex: Map<number, Set<string>> = new Map(); // bucket → factIds
  private domainIndex: Map<string, Map<number, Set<string>>> = new Map(); // domain → bucket → factIds
  private stratumIndex: Map<OntologicalStratum, Map<number, Set<string>>> =
    new Map(); // stratum → bucket → factIds
  private factEntries: Map<string, ConfidenceIndexEntry> = new Map(); // factId → entry

  constructor(config: ArchiveConfig = DEFAULT_ARCHIVE_CONFIG) {
    this.config = config;

    // Initialize buckets
    for (let i = 0; i < config.confidenceBuckets; i++) {
      this.globalIndex.set(i, new Set());
    }
  }

  // ============================================================================
  // Indexing Operations
  // ============================================================================

  /**
   * Index a fact by confidence.
   */
  indexFact(
    fact: FactState,
    domainId: string,
    stratum: OntologicalStratum,
    blockNumber: bigint,
  ): void {
    const bucket = confidenceToBucket(
      fact.confidence,
      this.config.confidenceBuckets,
    );

    // Remove from old bucket if exists
    const existingEntry = this.factEntries.get(fact.id);
    if (existingEntry) {
      this.removeFromBucket(
        fact.id,
        existingEntry.bucket,
        existingEntry.domainId,
        existingEntry.stratum,
      );
    }

    // Create new entry
    const entry: ConfidenceIndexEntry = {
      factId: fact.id,
      confidence: fact.confidence,
      bucket,
      domainId,
      stratum,
      indexedAt: blockNumber,
    };
    this.factEntries.set(fact.id, entry);

    // Add to global index
    let bucketSet = this.globalIndex.get(bucket);
    if (!bucketSet) {
      bucketSet = new Set();
      this.globalIndex.set(bucket, bucketSet);
    }
    bucketSet.add(fact.id);

    // Add to domain index
    let domainMap = this.domainIndex.get(domainId);
    if (!domainMap) {
      domainMap = new Map();
      this.domainIndex.set(domainId, domainMap);
    }
    let domainBucketSet = domainMap.get(bucket);
    if (!domainBucketSet) {
      domainBucketSet = new Set();
      domainMap.set(bucket, domainBucketSet);
    }
    domainBucketSet.add(fact.id);

    // Add to stratum index
    let stratumMap = this.stratumIndex.get(stratum);
    if (!stratumMap) {
      stratumMap = new Map();
      this.stratumIndex.set(stratum, stratumMap);
    }
    let stratumBucketSet = stratumMap.get(bucket);
    if (!stratumBucketSet) {
      stratumBucketSet = new Set();
      stratumMap.set(bucket, stratumBucketSet);
    }
    stratumBucketSet.add(fact.id);
  }

  /**
   * Remove a fact from its bucket.
   */
  private removeFromBucket(
    factId: string,
    bucket: number,
    domainId: string,
    stratum: OntologicalStratum,
  ): void {
    // Remove from global
    this.globalIndex.get(bucket)?.delete(factId);

    // Remove from domain
    this.domainIndex.get(domainId)?.get(bucket)?.delete(factId);

    // Remove from stratum
    this.stratumIndex.get(stratum)?.get(bucket)?.delete(factId);
  }

  /**
   * Remove a fact from the index.
   */
  removeFact(factId: string): void {
    const entry = this.factEntries.get(factId);
    if (entry) {
      this.removeFromBucket(
        factId,
        entry.bucket,
        entry.domainId,
        entry.stratum,
      );
      this.factEntries.delete(factId);
    }
  }

  // ============================================================================
  // Query Operations
  // ============================================================================

  /**
   * Get facts by confidence range.
   */
  getByConfidenceRange(options: ConfidenceQueryOptions = {}): string[] {
    const minConf = options.minConfidence ?? 0;
    const maxConf = options.maxConfidence ?? 1;
    const buckets = getBucketsInRange(
      minConf,
      maxConf,
      this.config.confidenceBuckets,
    );

    const factIds: Set<string> = new Set();

    // Determine which index to use
    if (options.domain) {
      // Use domain index
      const domainMap = this.domainIndex.get(options.domain);
      if (domainMap) {
        for (const bucket of buckets) {
          const bucketSet = domainMap.get(bucket);
          if (bucketSet) {
            bucketSet.forEach((id) => factIds.add(id));
          }
        }
      }
    } else if (options.stratum) {
      // Use stratum index
      const stratumMap = this.stratumIndex.get(options.stratum);
      if (stratumMap) {
        for (const bucket of buckets) {
          const bucketSet = stratumMap.get(bucket);
          if (bucketSet) {
            bucketSet.forEach((id) => factIds.add(id));
          }
        }
      }
    } else {
      // Use global index
      for (const bucket of buckets) {
        const bucketSet = this.globalIndex.get(bucket);
        if (bucketSet) {
          bucketSet.forEach((id) => factIds.add(id));
        }
      }
    }

    // Convert to array and sort
    let result = Array.from(factIds);

    // Sort by confidence
    result.sort((a, b) => {
      const entryA = this.factEntries.get(a);
      const entryB = this.factEntries.get(b);
      if (!entryA || !entryB) return 0;

      return options.sortOrder === "asc"
        ? entryA.confidence - entryB.confidence
        : entryB.confidence - entryA.confidence;
    });

    // Apply limit
    if (options.limit && result.length > options.limit) {
      result = result.slice(0, options.limit);
    }

    return result;
  }

  /**
   * Get top facts by confidence in a domain.
   */
  getTopByDomain(domainId: string, limit: number = 10): string[] {
    return this.getByConfidenceRange({
      domain: domainId,
      minConfidence: 0,
      maxConfidence: 1,
      limit,
      sortOrder: "desc",
    });
  }

  /**
   * Get top facts by confidence in a stratum.
   */
  getTopByStratum(stratum: OntologicalStratum, limit: number = 10): string[] {
    return this.getByConfidenceRange({
      stratum,
      minConfidence: 0,
      maxConfidence: 1,
      limit,
      sortOrder: "desc",
    });
  }

  /**
   * Get the confidence entry for a fact.
   */
  getEntry(factId: string): ConfidenceIndexEntry | undefined {
    return this.factEntries.get(factId);
  }

  /**
   * Get bucket statistics.
   */
  getBucketStats(): ConfidenceBucketEntry[] {
    const stats: ConfidenceBucketEntry[] = [];

    for (let bucket = 0; bucket < this.config.confidenceBuckets; bucket++) {
      const bucketSet = this.globalIndex.get(bucket);
      const range = getBucketRange(bucket, this.config.confidenceBuckets);

      stats.push({
        bucket,
        factIds: bucketSet ? Array.from(bucketSet) : [],
        lowerBound: range.lower,
        upperBound: range.upper,
      });
    }

    return stats;
  }

  /**
   * Get facts with confidence above threshold.
   */
  getHighConfidenceFacts(threshold: number = 0.8): string[] {
    return this.getByConfidenceRange({
      minConfidence: threshold,
      maxConfidence: 1,
      sortOrder: "desc",
    });
  }

  /**
   * Get facts with low confidence (uncertain).
   */
  getLowConfidenceFacts(threshold: number = 0.5): string[] {
    return this.getByConfidenceRange({
      minConfidence: 0,
      maxConfidence: threshold,
      sortOrder: "asc",
    });
  }

  /**
   * Get index statistics.
   */
  getStatistics(): {
    totalFacts: number;
    avgConfidence: number;
    confidenceDistribution: { bucket: number; count: number }[];
  } {
    let totalConfidence = 0;
    const distribution: { bucket: number; count: number }[] = [];

    for (let bucket = 0; bucket < this.config.confidenceBuckets; bucket++) {
      const bucketSet = this.globalIndex.get(bucket);
      const count = bucketSet?.size ?? 0;
      distribution.push({ bucket, count });
    }

    for (const entry of this.factEntries.values()) {
      totalConfidence += entry.confidence;
    }

    const totalFacts = this.factEntries.size;

    return {
      totalFacts,
      avgConfidence: totalFacts > 0 ? totalConfidence / totalFacts : 0,
      confidenceDistribution: distribution.filter((d) => d.count > 0),
    };
  }

  // ============================================================================
  // Serialization
  // ============================================================================

  /**
   * Generate storage key for confidence entry.
   */
  makeStorageKey(bucket: number, factId: string): string {
    return makeConfidenceKey(bucket, factId);
  }

  /**
   * Clear the index (for testing).
   */
  clear(): void {
    for (const bucketSet of this.globalIndex.values()) {
      bucketSet.clear();
    }
    this.domainIndex.clear();
    this.stratumIndex.clear();
    this.factEntries.clear();
  }
}
