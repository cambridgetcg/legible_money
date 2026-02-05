/**
 * Reference Graph Index
 *
 * Index for navigating relationships between facts (supports, contradicts,
 * extends, etc.). Enables confidence ratio calculations and transitive
 * relationship traversal.
 */

import type {
  ArchiveReference,
  ReferenceGraphNode,
  ConfidenceRatio,
} from "../types.js";
import type { ReferenceType } from "../../../state/types.js";
import {
  makeRefOutKey,
  makeRefInKey,
  // INDEX_PREFIXES,
  DEFAULT_ARCHIVE_CONFIG,
  type ArchiveConfig,
} from "../config.js";

// ============================================================================
// Reference Graph Types
// ============================================================================

/**
 * Options for reference queries.
 */
export interface ReferenceQueryOptions {
  /** Types of references to include */
  referenceTypes?: ReferenceType[];

  /** Maximum traversal depth */
  maxDepth?: number;

  /** Minimum reference strength to include */
  minStrength?: number;

  /** Maximum results */
  limit?: number;
}

/**
 * Result of a reference traversal.
 */
export interface ReferenceTraversalResult {
  /** Fact ID */
  factId: string;

  /** Distance from source (depth) */
  depth: number;

  /** Path from source to this fact */
  path: string[];

  /** Cumulative strength along path */
  cumulativeStrength: number;

  /** Reference type of final edge */
  referenceType: ReferenceType;
}

/**
 * Stored reference entry.
 */
export interface StoredReference {
  /** Source fact ID */
  sourceId: string;

  /** Target fact ID */
  targetId: string;

  /** Reference type */
  referenceType: ReferenceType;

  /** Strength of the reference */
  strength: number;

  /** Whether bidirectional */
  bidirectional: boolean;

  /** Block when indexed */
  indexedAt: bigint;
}

// ============================================================================
// Reference Graph Index
// ============================================================================

/**
 * Index for fact reference graph traversal.
 *
 * Key structures:
 * - ref_out:<fact_id>:<ref_type>:<target_id> → StoredReference
 * - ref_in:<fact_id>:<ref_type>:<source_id> → StoredReference
 * - ref_strength:<fact_id> → { supporting: number, contradicting: number }
 */
export class ReferenceGraphIndex {
  private config: ArchiveConfig;

  // In-memory graph (would be backed by LevelDB in production)
  private outgoing: Map<
    string,
    Map<ReferenceType, Map<string, StoredReference>>
  > = new Map();
  private incoming: Map<
    string,
    Map<ReferenceType, Map<string, StoredReference>>
  > = new Map();
  private confidenceCache: Map<
    string,
    { supporting: number; contradicting: number }
  > = new Map();

  constructor(config: ArchiveConfig = DEFAULT_ARCHIVE_CONFIG) {
    this.config = config;
  }

  // ============================================================================
  // Indexing Operations
  // ============================================================================

  /**
   * Index a reference between facts.
   */
  indexReference(reference: ArchiveReference, blockNumber: bigint): void {
    const stored: StoredReference = {
      sourceId: reference.factId,
      targetId: reference.targetFactId,
      referenceType: reference.referenceType,
      strength: reference.strength,
      bidirectional: reference.bidirectional,
      indexedAt: blockNumber,
    };

    // Add to outgoing index
    this.addOutgoing(
      reference.factId,
      reference.targetFactId,
      reference.referenceType,
      stored,
    );

    // Add to incoming index
    this.addIncoming(
      reference.targetFactId,
      reference.factId,
      reference.referenceType,
      stored,
    );

    // If bidirectional, add reverse edges
    if (reference.bidirectional) {
      const reverseStored: StoredReference = {
        ...stored,
        sourceId: reference.targetFactId,
        targetId: reference.factId,
      };
      this.addOutgoing(
        reference.targetFactId,
        reference.factId,
        reference.referenceType,
        reverseStored,
      );
      this.addIncoming(
        reference.factId,
        reference.targetFactId,
        reference.referenceType,
        reverseStored,
      );
    }

    // Invalidate confidence cache for affected facts
    this.confidenceCache.delete(reference.factId);
    this.confidenceCache.delete(reference.targetFactId);
  }

  private addOutgoing(
    sourceId: string,
    targetId: string,
    refType: ReferenceType,
    stored: StoredReference,
  ): void {
    let factMap = this.outgoing.get(sourceId);
    if (!factMap) {
      factMap = new Map();
      this.outgoing.set(sourceId, factMap);
    }

    let typeMap = factMap.get(refType);
    if (!typeMap) {
      typeMap = new Map();
      factMap.set(refType, typeMap);
    }

    typeMap.set(targetId, stored);
  }

  private addIncoming(
    targetId: string,
    sourceId: string,
    refType: ReferenceType,
    stored: StoredReference,
  ): void {
    let factMap = this.incoming.get(targetId);
    if (!factMap) {
      factMap = new Map();
      this.incoming.set(targetId, factMap);
    }

    let typeMap = factMap.get(refType);
    if (!typeMap) {
      typeMap = new Map();
      factMap.set(refType, typeMap);
    }

    typeMap.set(sourceId, stored);
  }

  /**
   * Remove a reference from the index.
   */
  removeReference(
    sourceId: string,
    targetId: string,
    refType: ReferenceType,
  ): void {
    // Remove from outgoing
    this.outgoing.get(sourceId)?.get(refType)?.delete(targetId);

    // Remove from incoming
    this.incoming.get(targetId)?.get(refType)?.delete(sourceId);

    // Invalidate cache
    this.confidenceCache.delete(sourceId);
    this.confidenceCache.delete(targetId);
  }

  // ============================================================================
  // Query Operations
  // ============================================================================

  /**
   * Get outgoing references from a fact.
   */
  getOutgoing(
    factId: string,
    options: ReferenceQueryOptions = {},
  ): StoredReference[] {
    const factMap = this.outgoing.get(factId);
    if (!factMap) return [];

    const results: StoredReference[] = [];
    const types = options.referenceTypes || Array.from(factMap.keys());

    for (const refType of types) {
      const typeMap = factMap.get(refType);
      if (typeMap) {
        for (const ref of typeMap.values()) {
          if (!options.minStrength || ref.strength >= options.minStrength) {
            results.push(ref);
          }
        }
      }
    }

    // Apply limit
    if (options.limit && results.length > options.limit) {
      return results.slice(0, options.limit);
    }

    return results;
  }

  /**
   * Get incoming references to a fact.
   */
  getIncoming(
    factId: string,
    options: ReferenceQueryOptions = {},
  ): StoredReference[] {
    const factMap = this.incoming.get(factId);
    if (!factMap) return [];

    const results: StoredReference[] = [];
    const types = options.referenceTypes || Array.from(factMap.keys());

    for (const refType of types) {
      const typeMap = factMap.get(refType);
      if (typeMap) {
        for (const ref of typeMap.values()) {
          if (!options.minStrength || ref.strength >= options.minStrength) {
            results.push(ref);
          }
        }
      }
    }

    // Apply limit
    if (options.limit && results.length > options.limit) {
      return results.slice(0, options.limit);
    }

    return results;
  }

  /**
   * Get the support chain for a fact (transitive support).
   */
  getSupportChain(
    factId: string,
    maxDepth: number = this.config.referenceGraphMaxDepth,
  ): ReferenceTraversalResult[] {
    return this.traverseGraph(factId, ["supports", "extends"], maxDepth);
  }

  /**
   * Get facts that contradict a given fact.
   */
  getContradictions(factId: string): StoredReference[] {
    return this.getIncoming(factId, { referenceTypes: ["contradicts"] });
  }

  /**
   * Get all facts that support a given fact.
   */
  getSupportingFacts(factId: string): StoredReference[] {
    return this.getIncoming(factId, {
      referenceTypes: ["supports", "extends"],
    });
  }

  /**
   * Traverse the reference graph.
   */
  traverseGraph(
    startId: string,
    referenceTypes: ReferenceType[],
    maxDepth: number,
  ): ReferenceTraversalResult[] {
    const results: ReferenceTraversalResult[] = [];
    const visited = new Set<string>();
    const queue: {
      factId: string;
      depth: number;
      path: string[];
      strength: number;
    }[] = [{ factId: startId, depth: 0, path: [startId], strength: 1.0 }];

    while (queue.length > 0) {
      const { factId, depth, path, strength } = queue.shift()!;

      if (depth > 0 && !visited.has(factId)) {
        visited.add(factId);

        // Get the reference type of the edge that brought us here
        const lastEdge = this.getOutgoing(path[path.length - 2], {
          referenceTypes,
        }).find((r) => r.targetId === factId);

        results.push({
          factId,
          depth,
          path,
          cumulativeStrength: strength,
          referenceType: lastEdge?.referenceType || "supports",
        });
      }

      if (depth < maxDepth) {
        const outgoing = this.getOutgoing(factId, { referenceTypes });
        for (const ref of outgoing) {
          if (!visited.has(ref.targetId)) {
            const newStrength =
              strength * ref.strength * this.config.depthDecayFactor;
            queue.push({
              factId: ref.targetId,
              depth: depth + 1,
              path: [...path, ref.targetId],
              strength: newStrength,
            });
          }
        }
      }
    }

    return results;
  }

  /**
   * Calculate confidence ratio for a fact.
   * Returns the ratio of supporting vs contradicting evidence.
   */
  calculateConfidenceRatio(
    factId: string,
    factConfidences: Map<string, number>,
  ): ConfidenceRatio {
    // Check cache first
    const cached = this.confidenceCache.get(factId);
    if (cached) {
      const total = cached.supporting + cached.contradicting;
      return {
        supporting: cached.supporting,
        contradicting: cached.contradicting,
        ratio: total > 0 ? cached.supporting / total : 0.5,
        netConfidence: Math.max(
          0,
          Math.min(1, cached.supporting - cached.contradicting),
        ),
      };
    }

    let supporting = 0;
    let contradicting = 0;

    // Sum supporting references
    const supportRefs = this.getIncoming(factId, {
      referenceTypes: ["supports", "extends"],
    });
    for (const ref of supportRefs) {
      const sourceConfidence = factConfidences.get(ref.sourceId) ?? 0.5;
      supporting += sourceConfidence * ref.strength;
    }

    // Sum contradicting references
    const contradictRefs = this.getIncoming(factId, {
      referenceTypes: ["contradicts"],
    });
    for (const ref of contradictRefs) {
      const sourceConfidence = factConfidences.get(ref.sourceId) ?? 0.5;
      contradicting += sourceConfidence * ref.strength;
    }

    // Cache result
    this.confidenceCache.set(factId, { supporting, contradicting });

    const total = supporting + contradicting;
    return {
      supporting,
      contradicting,
      ratio: total > 0 ? supporting / total : 0.5,
      netConfidence: Math.max(0, Math.min(1, supporting - contradicting + 0.5)),
    };
  }

  /**
   * Get the reference graph node for a fact.
   */
  getNode(factId: string): ReferenceGraphNode {
    const outgoingMap = this.outgoing.get(factId);
    const incomingMap = this.incoming.get(factId);

    const outgoing: Map<ReferenceType, string[]> = new Map();
    const incoming: Map<ReferenceType, string[]> = new Map();
    let outDegree = 0;
    let inDegree = 0;

    if (outgoingMap) {
      for (const [refType, targets] of outgoingMap) {
        outgoing.set(refType, Array.from(targets.keys()));
        outDegree += targets.size;
      }
    }

    if (incomingMap) {
      for (const [refType, sources] of incomingMap) {
        incoming.set(refType, Array.from(sources.keys()));
        inDegree += sources.size;
      }
    }

    return {
      factId,
      outgoing,
      incoming,
      outDegree,
      inDegree,
    };
  }

  /**
   * Get index statistics.
   */
  getStatistics(): {
    totalReferences: number;
    referencesByType: Record<string, number>;
    avgOutDegree: number;
    avgInDegree: number;
  } {
    const referencesByType: Record<string, number> = {};
    let totalReferences = 0;
    let totalOutDegree = 0;
    let totalInDegree = 0;

    for (const factMap of this.outgoing.values()) {
      for (const [refType, targets] of factMap) {
        referencesByType[refType] =
          (referencesByType[refType] || 0) + targets.size;
        totalReferences += targets.size;
        totalOutDegree += targets.size;
      }
    }

    for (const factMap of this.incoming.values()) {
      for (const targets of factMap.values()) {
        totalInDegree += targets.size;
      }
    }

    const nodeCount = Math.max(this.outgoing.size, this.incoming.size);

    return {
      totalReferences,
      referencesByType,
      avgOutDegree: nodeCount > 0 ? totalOutDegree / nodeCount : 0,
      avgInDegree: nodeCount > 0 ? totalInDegree / nodeCount : 0,
    };
  }

  // ============================================================================
  // Serialization
  // ============================================================================

  /**
   * Generate storage key for outgoing reference.
   */
  makeOutgoingKey(factId: string, refType: string, targetId: string): string {
    return makeRefOutKey(factId, refType, targetId);
  }

  /**
   * Generate storage key for incoming reference.
   */
  makeIncomingKey(factId: string, refType: string, sourceId: string): string {
    return makeRefInKey(factId, refType, sourceId);
  }

  /**
   * Clear the index (for testing).
   */
  clear(): void {
    this.outgoing.clear();
    this.incoming.clear();
    this.confidenceCache.clear();
  }
}
