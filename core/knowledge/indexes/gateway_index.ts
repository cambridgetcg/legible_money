/**
 * Gateway Domain Index
 *
 * Identifies and indexes "gateway domains" - domains that bridge
 * knowledge across strata. High gateway score domains are important
 * for cross-stratum reasoning.
 *
 * A domain becomes a gateway when:
 * 1. Facts in higher strata depend on facts in this domain
 * 2. Facts in this domain depend on lower strata
 * 3. The domain serves as a knowledge bridge
 *
 * PP HIERARCHY:
 *   TRUTH: Gateway scores grounded in actual dependency patterns
 *   UNDERSTANDING: Enables comprehension of cross-stratum knowledge flow
 *   BEAUTY: Elegant concept capturing knowledge connectivity
 *   JUSTICE: Identifies domains that deserve recognition as bridges
 *   CREATIVITY: Novel pattern for reasoning optimization
 */

import type {
  OntologicalStratum,
  OntologicalDomain,
} from "../ontology/types.js";
import { STRATUM_HIERARCHY } from "../ontology/types.js";
import {
  type GatewayDomainInfo,
  type GatewayQueryResult,
  type ReasoningIndexConfig,
  DEFAULT_REASONING_INDEX_CONFIG,
} from "./types.js";

// ============================================================================
// Gateway Domain Index
// ============================================================================

/**
 * Index for identifying and querying gateway domains.
 */
export class GatewayDomainIndex {
  private config: ReasoningIndexConfig;

  /** Gateway domain information */
  private gateways: Map<string, GatewayDomainInfo> = new Map();

  /** Pre-computed gateway paths between strata pairs */
  private gatewayPaths: Map<string, string[]> = new Map();

  /** Domains by stratum for quick lookup */
  private domainsByStratum: Map<OntologicalStratum, Set<string>> = new Map();

  /** Inbound cross-stratum connections (domain → domains that depend on it) */
  private inboundConnections: Map<string, Set<string>> = new Map();

  /** Outbound cross-stratum connections (domain → domains it depends on) */
  private outboundConnections: Map<string, Set<string>> = new Map();

  constructor(config: ReasoningIndexConfig = DEFAULT_REASONING_INDEX_CONFIG) {
    this.config = { ...config };

    // Initialize stratum sets
    for (const stratum of Object.keys(
      STRATUM_HIERARCHY,
    ) as OntologicalStratum[]) {
      this.domainsByStratum.set(stratum, new Set());
    }
  }

  // ============================================================================
  // Domain Registration
  // ============================================================================

  /**
   * Register a domain in the index.
   */
  registerDomain(domain: OntologicalDomain): void {
    const stratumSet = this.domainsByStratum.get(domain.stratum);
    if (stratumSet) {
      stratumSet.add(domain.id);
    }
  }

  /**
   * Record a cross-stratum connection (fact dependency).
   */
  recordConnection(
    fromDomainId: string,
    fromStratum: OntologicalStratum,
    toDomainId: string,
    toStratum: OntologicalStratum,
  ): void {
    // Only record if crossing strata
    if (fromStratum === toStratum) return;

    // Record outbound from higher stratum domain
    let outbound = this.outboundConnections.get(fromDomainId);
    if (!outbound) {
      outbound = new Set();
      this.outboundConnections.set(fromDomainId, outbound);
    }
    outbound.add(toDomainId);

    // Record inbound to lower stratum domain
    let inbound = this.inboundConnections.get(toDomainId);
    if (!inbound) {
      inbound = new Set();
      this.inboundConnections.set(toDomainId, inbound);
    }
    inbound.add(fromDomainId);

    // Re-evaluate gateway status for both domains
    this.evaluateGatewayStatus(fromDomainId, fromStratum);
    this.evaluateGatewayStatus(toDomainId, toStratum);

    // Invalidate pre-computed paths involving these strata
    this.invalidatePathsForStrata(fromStratum, toStratum);
  }

  // ============================================================================
  // Gateway Evaluation
  // ============================================================================

  /**
   * Evaluate and update gateway status for a domain.
   */
  evaluateGatewayStatus(domainId: string, stratum: OntologicalStratum): void {
    const inbound = this.inboundConnections.get(domainId);
    const outbound = this.outboundConnections.get(domainId);

    const inboundCount = inbound?.size ?? 0;
    const outboundCount = outbound?.size ?? 0;

    // Calculate gateway score
    // A good gateway has both inbound (higher strata depend on it) and outbound (it depends on lower)
    const gatewayScore = this.calculateGatewayScore(
      inboundCount,
      outboundCount,
      stratum,
    );

    if (gatewayScore >= this.config.gatewayThreshold) {
      // This is a gateway domain
      const existingGateway = this.gateways.get(domainId);

      // Determine which strata this bridges
      const bridgesStrata = this.determineBridgeStrata(domainId, stratum);

      const gatewayInfo: GatewayDomainInfo = {
        domainId,
        domain: existingGateway?.domain ?? ({} as OntologicalDomain), // Will be set by caller
        gatewayScore,
        connectsTo: Array.from(inbound ?? []),
        connectsFrom: Array.from(outbound ?? []),
        crossStratumFactCount: inboundCount + outboundCount,
        bridgesStrata,
      };

      this.gateways.set(domainId, gatewayInfo);
    } else {
      // Not a gateway (anymore)
      this.gateways.delete(domainId);
    }
  }

  /**
   * Calculate gateway score based on connectivity.
   */
  private calculateGatewayScore(
    inboundCount: number,
    outboundCount: number,
    stratum: OntologicalStratum,
  ): number {
    // Base score from connectivity
    const connectivityScore = Math.min(1, (inboundCount + outboundCount) / 20);

    // Bonus for balanced connections (both in and out)
    const balance = inboundCount > 0 && outboundCount > 0 ? 0.2 : 0;

    // Bonus for middle strata (more likely to be bridges)
    const stratumLevel = STRATUM_HIERARCHY[stratum];
    const middleBonus = stratumLevel >= 2 && stratumLevel <= 4 ? 0.1 : 0;

    return Math.min(1, connectivityScore + balance + middleBonus);
  }

  /**
   * Determine which strata a domain bridges.
   */
  private determineBridgeStrata(
    domainId: string,
    stratum: OntologicalStratum,
  ): [OntologicalStratum, OntologicalStratum] {
    const inbound = this.inboundConnections.get(domainId);
    const outbound = this.outboundConnections.get(domainId);

    let highestInbound = stratum;
    let lowestOutbound = stratum;

    // Find highest stratum that depends on this domain
    if (inbound && inbound.size > 0) {
      // Look up stratum of dependent domain (simplified - would need domain data)
      // For now, assume it's one level higher
      const level = STRATUM_HIERARCHY[stratum];
      if (level < 6) {
        highestInbound = this.getStratumByLevel(level + 1) ?? stratum;
      }
    }

    // Find lowest stratum this domain depends on
    if (outbound && outbound.size > 0) {
      const level = STRATUM_HIERARCHY[stratum];
      if (level > 0) {
        lowestOutbound = this.getStratumByLevel(level - 1) ?? stratum;
      }
    }

    return [lowestOutbound, highestInbound];
  }

  private getStratumByLevel(level: number): OntologicalStratum | undefined {
    for (const [stratum, l] of Object.entries(STRATUM_HIERARCHY)) {
      if (l === level) return stratum as OntologicalStratum;
    }
    return undefined;
  }

  /**
   * Invalidate cached paths for affected strata.
   */
  private invalidatePathsForStrata(
    stratum1: OntologicalStratum,
    stratum2: OntologicalStratum,
  ): void {
    const keysToDelete: string[] = [];
    for (const key of this.gatewayPaths.keys()) {
      if (key.includes(stratum1) || key.includes(stratum2)) {
        keysToDelete.push(key);
      }
    }
    for (const key of keysToDelete) {
      this.gatewayPaths.delete(key);
    }
  }

  // ============================================================================
  // Query Operations
  // ============================================================================

  /**
   * Find gateway domains between two strata.
   * Returns cached result if available.
   */
  findGatewaysBetweenStrata(
    fromStratum: OntologicalStratum,
    toStratum: OntologicalStratum,
  ): GatewayQueryResult {
    const pathKey = `${fromStratum}:${toStratum}`;

    // Check cache
    const cached = this.gatewayPaths.get(pathKey);
    if (cached) {
      return {
        gateways: cached
          .map((id) => this.gateways.get(id))
          .filter((g): g is GatewayDomainInfo => g !== undefined),
        cached: true,
        fromStratum,
        toStratum,
      };
    }

    // Compute gateway path
    const fromLevel = STRATUM_HIERARCHY[fromStratum];
    const toLevel = STRATUM_HIERARCHY[toStratum];
    const minLevel = Math.min(fromLevel, toLevel);
    const maxLevel = Math.max(fromLevel, toLevel);

    // Find gateways that bridge the gap
    const matchingGateways: GatewayDomainInfo[] = [];

    for (const gateway of this.gateways.values()) {
      const gatewayLevel = STRATUM_HIERARCHY[gateway.bridgesStrata[0]];
      const bridgeToLevel = STRATUM_HIERARCHY[gateway.bridgesStrata[1]];

      // Gateway should be between or at the boundary of the two strata
      if (
        (gatewayLevel >= minLevel && gatewayLevel <= maxLevel) ||
        (bridgeToLevel >= minLevel && bridgeToLevel <= maxLevel)
      ) {
        matchingGateways.push(gateway);
      }
    }

    // Sort by gateway score with domainId tiebreaker for determinism
    matchingGateways.sort(
      (a, b) =>
        b.gatewayScore - a.gatewayScore || a.domainId.localeCompare(b.domainId),
    );

    // Cache the result
    this.gatewayPaths.set(
      pathKey,
      matchingGateways.map((g) => g.domainId),
    );

    return {
      gateways: matchingGateways,
      cached: false,
      fromStratum,
      toStratum,
    };
  }

  /**
   * Get all gateway domains for a stratum.
   */
  getGatewaysForStratum(stratum: OntologicalStratum): GatewayDomainInfo[] {
    const result: GatewayDomainInfo[] = [];
    const domains = this.domainsByStratum.get(stratum);

    if (domains) {
      for (const domainId of domains) {
        const gateway = this.gateways.get(domainId);
        if (gateway) {
          result.push(gateway);
        }
      }
    }

    return result.sort(
      (a, b) =>
        b.gatewayScore - a.gatewayScore || a.domainId.localeCompare(b.domainId),
    );
  }

  /**
   * Get gateway info for a specific domain.
   */
  getGateway(domainId: string): GatewayDomainInfo | undefined {
    return this.gateways.get(domainId);
  }

  /**
   * Check if a domain is a gateway.
   */
  isGateway(domainId: string): boolean {
    return this.gateways.has(domainId);
  }

  /**
   * Get top gateways by score.
   */
  getTopGateways(limit: number = 10): GatewayDomainInfo[] {
    return Array.from(this.gateways.values())
      .sort(
        (a, b) =>
          b.gatewayScore - a.gatewayScore ||
          a.domainId.localeCompare(b.domainId),
      )
      .slice(0, limit);
  }

  // ============================================================================
  // Statistics
  // ============================================================================

  /**
   * Get gateway index statistics.
   */
  getStatistics(): {
    totalGateways: number;
    gatewaysByStratum: Record<string, number>;
    avgGatewayScore: number;
    cachedPaths: number;
  } {
    const gatewaysByStratum: Record<string, number> = {};
    let totalScore = 0;

    for (const gateway of this.gateways.values()) {
      totalScore += gateway.gatewayScore;
      const stratum = gateway.bridgesStrata[0];
      gatewaysByStratum[stratum] = (gatewaysByStratum[stratum] ?? 0) + 1;
    }

    return {
      totalGateways: this.gateways.size,
      gatewaysByStratum,
      avgGatewayScore:
        this.gateways.size > 0 ? totalScore / this.gateways.size : 0,
      cachedPaths: this.gatewayPaths.size,
    };
  }

  /**
   * Clear the index (for testing).
   */
  clear(): void {
    this.gateways.clear();
    this.gatewayPaths.clear();
    this.inboundConnections.clear();
    this.outboundConnections.clear();

    // Re-initialize stratum sets
    for (const stratum of Object.keys(
      STRATUM_HIERARCHY,
    ) as OntologicalStratum[]) {
      this.domainsByStratum.set(stratum, new Set());
    }
  }
}
