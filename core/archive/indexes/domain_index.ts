/**
 * Domain Hierarchy Index
 *
 * Index for efficient domain tree traversal and fact retrieval by domain.
 * Supports pattern matching (e.g., 'physics.*') and hierarchical queries.
 */

import type { DomainIndexEntry } from "../types.js";
import type {
  OntologicalDomain,
  OntologicalStratum,
} from "../../knowledge/ontology/types.js";
import type { FactState } from "../../../state/types.js";
import {
  makeDomainKey,
  makeDomainFactsKey,
  confidenceToBucket,
  DEFAULT_ARCHIVE_CONFIG,
  type ArchiveConfig,
} from "../config.js";

// ============================================================================
// Domain Index Interface
// ============================================================================

/**
 * Options for domain queries.
 */
export interface DomainQueryOptions {
  /** Include facts from descendant domains */
  includeDescendants?: boolean;

  /** Minimum confidence threshold */
  minConfidence?: number;

  /** Maximum confidence threshold */
  maxConfidence?: number;

  /** Stratum filter */
  stratum?: OntologicalStratum;

  /** Maximum results */
  limit?: number;

  /** Cursor for pagination */
  cursor?: string;
}

/**
 * Result of a domain pattern match.
 */
export interface DomainPatternMatch {
  /** Matched domain ID */
  domainId: string;

  /** The domain data */
  domain: OntologicalDomain;

  /** Fact count in this domain */
  factCount: number;

  /** Whether this is an exact match or pattern match */
  matchType: "exact" | "pattern";
}

// ============================================================================
// Domain Hierarchy Index
// ============================================================================

/**
 * Index for domain hierarchy traversal and fact retrieval.
 *
 * Key structures:
 * - domain:<domain_id> → DomainIndexEntry
 * - domain_facts:<domain_id>:<confidence_bucket>:<fact_id> → factId
 * - domain_children:<domain_id> → string[]
 */
export class DomainHierarchyIndex {
  private config: ArchiveConfig;

  // In-memory caches (would be backed by LevelDB in production)
  private domainIndex: Map<string, DomainIndexEntry> = new Map();
  private domainFacts: Map<string, Set<string>> = new Map(); // domainId → factIds
  private domainChildren: Map<string, Set<string>> = new Map(); // domainId → childDomainIds

  constructor(config: ArchiveConfig = DEFAULT_ARCHIVE_CONFIG) {
    this.config = config;
  }

  // ============================================================================
  // Indexing Operations
  // ============================================================================

  /**
   * Index a domain entry.
   */
  indexDomain(domain: OntologicalDomain, blockNumber: bigint): void {
    const entry: DomainIndexEntry = {
      domainId: domain.id,
      domain,
      factCount: 0,
      avgConfidence: 0,
      childDomains: [],
      lastIndexed: blockNumber,
    };

    this.domainIndex.set(domain.id, entry);

    // Index parent-child relationship
    if (domain.parentDomain) {
      let children = this.domainChildren.get(domain.parentDomain);
      if (!children) {
        children = new Set();
        this.domainChildren.set(domain.parentDomain, children);
      }
      children.add(domain.id);

      // Update parent's childDomains list
      const parentEntry = this.domainIndex.get(domain.parentDomain);
      if (parentEntry) {
        parentEntry.childDomains = Array.from(children);
      }
    }
  }

  /**
   * Index a fact in its domain.
   */
  indexFact(fact: FactState, domainId: string, blockNumber: bigint): void {
    // Add to domain's fact set
    let factSet = this.domainFacts.get(domainId);
    if (!factSet) {
      factSet = new Set();
      this.domainFacts.set(domainId, factSet);
    }
    factSet.add(fact.id);

    // Update domain entry statistics
    const entry = this.domainIndex.get(domainId);
    if (entry) {
      entry.factCount = factSet.size;
      entry.lastIndexed = blockNumber;

      // Update average confidence (simplified - would use running average in production)
      // This is a placeholder - real implementation would track sum and count
    }
  }

  /**
   * Remove a fact from its domain index.
   */
  removeFact(factId: string, domainId: string): void {
    const factSet = this.domainFacts.get(domainId);
    if (factSet) {
      factSet.delete(factId);

      const entry = this.domainIndex.get(domainId);
      if (entry) {
        entry.factCount = factSet.size;
      }
    }
  }

  // ============================================================================
  // Query Operations
  // ============================================================================

  /**
   * Get a domain entry by ID.
   */
  getDomain(domainId: string): DomainIndexEntry | undefined {
    return this.domainIndex.get(domainId);
  }

  /**
   * Get all fact IDs in a domain.
   */
  getFactsInDomain(
    domainId: string,
    options: DomainQueryOptions = {},
  ): string[] {
    const factIds: Set<string> = new Set();

    // Get facts from this domain
    const directFacts = this.domainFacts.get(domainId);
    if (directFacts) {
      directFacts.forEach((id) => factIds.add(id));
    }

    // Include descendants if requested
    if (options.includeDescendants) {
      const descendants = this.getDescendants(domainId);
      for (const descId of descendants) {
        const descFacts = this.domainFacts.get(descId);
        if (descFacts) {
          descFacts.forEach((id) => factIds.add(id));
        }
      }
    }

    // Convert to array and apply limit
    let result = Array.from(factIds);

    if (options.limit && result.length > options.limit) {
      result = result.slice(0, options.limit);
    }

    return result;
  }

  /**
   * Match domains by pattern (e.g., 'physics.*').
   */
  matchDomainPattern(pattern: string): DomainPatternMatch[] {
    const matches: DomainPatternMatch[] = [];

    // Handle exact match
    if (!pattern.includes("*")) {
      const entry = this.domainIndex.get(pattern);
      if (entry) {
        matches.push({
          domainId: entry.domainId,
          domain: entry.domain,
          factCount: entry.factCount,
          matchType: "exact",
        });
      }
      return matches;
    }

    // Handle wildcard patterns
    const regexPattern = pattern.replace(/\./g, "\\.").replace(/\*/g, ".*");
    const regex = new RegExp(`^${regexPattern}$`);

    for (const [domainId, entry] of this.domainIndex) {
      if (regex.test(domainId)) {
        matches.push({
          domainId: entry.domainId,
          domain: entry.domain,
          factCount: entry.factCount,
          matchType: "pattern",
        });
      }
    }

    return matches;
  }

  /**
   * Get child domains of a domain.
   */
  getChildren(domainId: string): string[] {
    const children = this.domainChildren.get(domainId);
    return children ? Array.from(children) : [];
  }

  /**
   * Get all ancestor domains (path to root).
   */
  getAncestors(domainId: string): string[] {
    const ancestors: string[] = [];
    let current = this.domainIndex.get(domainId);

    while (current && current.domain.parentDomain) {
      ancestors.push(current.domain.parentDomain);
      current = this.domainIndex.get(current.domain.parentDomain);
    }

    return ancestors;
  }

  /**
   * Get all descendant domains.
   */
  getDescendants(domainId: string): string[] {
    const descendants: string[] = [];
    const queue = this.getChildren(domainId);

    while (queue.length > 0) {
      const childId = queue.shift()!;
      descendants.push(childId);
      queue.push(...this.getChildren(childId));
    }

    return descendants;
  }

  /**
   * Get the domain path from root to a domain.
   */
  getDomainPath(domainId: string): string[] {
    const ancestors = this.getAncestors(domainId);
    return [...ancestors.reverse(), domainId];
  }

  /**
   * Get domains by stratum.
   */
  getDomainsByStratum(stratum: OntologicalStratum): DomainIndexEntry[] {
    const result: DomainIndexEntry[] = [];

    for (const entry of this.domainIndex.values()) {
      if (entry.domain.stratum === stratum) {
        result.push(entry);
      }
    }

    return result;
  }

  /**
   * Get total fact count across all domains.
   */
  getTotalFactCount(): number {
    let total = 0;
    for (const entry of this.domainIndex.values()) {
      total += entry.factCount;
    }
    return total;
  }

  /**
   * Get domain statistics.
   */
  getStatistics(): {
    totalDomains: number;
    totalFacts: number;
    domainsByStratum: Record<string, number>;
  } {
    const domainsByStratum: Record<string, number> = {};

    for (const entry of this.domainIndex.values()) {
      const stratum = entry.domain.stratum;
      domainsByStratum[stratum] = (domainsByStratum[stratum] || 0) + 1;
    }

    return {
      totalDomains: this.domainIndex.size,
      totalFacts: this.getTotalFactCount(),
      domainsByStratum,
    };
  }

  // ============================================================================
  // Serialization
  // ============================================================================

  /**
   * Generate storage key for domain entry.
   */
  makeDomainStorageKey(domainId: string): string {
    return makeDomainKey(domainId);
  }

  /**
   * Generate storage key for domain fact entry.
   */
  makeDomainFactStorageKey(
    domainId: string,
    confidence: number,
    factId: string,
  ): string {
    const bucket = confidenceToBucket(
      confidence,
      this.config.confidenceBuckets,
    );
    return makeDomainFactsKey(domainId, bucket, factId);
  }

  /**
   * Clear the index (for testing).
   */
  clear(): void {
    this.domainIndex.clear();
    this.domainFacts.clear();
    this.domainChildren.clear();
  }
}
