/**
 * Knowledge State Tree
 *
 * Sparse Merkle Tree for verified facts (knowledge).
 * This is the core state that distinguishes Legible Money.
 *
 * Key: hash(factId) - 32 bytes
 * Value: serialized FactState
 *
 * Provides:
 * - Cryptographic proof of knowledge existence
 * - Knowledge graph structure via references
 * - Challenge and dispute tracking
 * - Confidence decay over time
 */

import { InMemorySMT, type MerkleRoot } from "../core/crypto/merkle.js";
import { sha512_256, sha512_256Hex } from "../core/crypto/hashing.js";
import {
  type Confidence,
  type EpistemicCategory,
  type EpistemicProfile,
  EPISTEMIC_PROFILES,
} from "../core/knowledge/types.js";
import {
  type OntologicalStratum,
  STRATUM_TO_DECAY_CATEGORY,
} from "../core/knowledge/ontology/types.js";
import { calculateCategoryDecayedConfidence } from "../consensus/pot/confidence.js";
import {
  type FactState,
  type FactReference,
  type FactStatus,
  type ReferenceType,
  type DomainState,
  type FactProof,
  type DecayInfo,
  StateError,
} from "./types.js";

// ============ Serialization ============

function serializeFactState(fact: FactState): Uint8Array {
  const json = JSON.stringify({
    id: fact.id,
    claim: fact.claim,
    domain: fact.domain,
    confidence: fact.confidence,
    verifiedAt: fact.verifiedAt.toString(),
    verifierCount: fact.verifierCount,
    challengeCount: fact.challengeCount,
    lastChallengedAt: fact.lastChallengedAt?.toString(),
    status: fact.status,
    referenceRoot: fact.referenceRoot,
    // GAP-009: Expiration fields
    expiresAt: fact.expiresAt?.toString(),
    lastReverifiedAt: fact.lastReverifiedAt?.toString(),
    // Citation economics
    citationCount: fact.citationCount,
    lastCitedBlock: fact.lastCitedBlock?.toString(),
    totalCitationRewards: fact.totalCitationRewards?.toString(),
    // Ontological metadata
    epistemicCategory: fact.epistemicCategory,
    stratum: fact.stratum,
    // Gödelian zone metadata
    logicZone: fact.logicZone,
    decidability: fact.decidability,
    // U3/U7: stratum ceiling + undecidable resolution
    stratumCeiling: fact.stratumCeiling,
    undecidableResolution: fact.undecidableResolution
      ? {
          ...fact.undecidableResolution,
          resolvedAt: fact.undecidableResolution.resolvedAt.toString(),
        }
      : undefined,
    // Multidimensional confidence (all fields are number/string — no BigInt)
    multidimensionalConfidence: fact.multidimensionalConfidence,
  });
  return new TextEncoder().encode(json);
}

function deserializeFactState(bytes: Uint8Array): FactState {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    id: obj.id,
    claim: obj.claim,
    domain: obj.domain,
    confidence: obj.confidence as Confidence,
    verifiedAt: BigInt(obj.verifiedAt),
    verifierCount: obj.verifierCount,
    challengeCount: obj.challengeCount,
    lastChallengedAt: obj.lastChallengedAt
      ? BigInt(obj.lastChallengedAt)
      : undefined,
    status: obj.status as FactStatus,
    referenceRoot: obj.referenceRoot,
    // GAP-009: Expiration fields
    expiresAt: obj.expiresAt ? BigInt(obj.expiresAt) : undefined,
    lastReverifiedAt: obj.lastReverifiedAt
      ? BigInt(obj.lastReverifiedAt)
      : undefined,
    // Citation economics (backward compat: default citationCount to 0)
    citationCount: obj.citationCount ?? 0,
    lastCitedBlock: obj.lastCitedBlock ? BigInt(obj.lastCitedBlock) : undefined,
    totalCitationRewards: obj.totalCitationRewards
      ? BigInt(obj.totalCitationRewards)
      : undefined,
    // Ontological metadata
    epistemicCategory: obj.epistemicCategory, // undefined for legacy facts (backward compat)
    stratum: obj.stratum,
    // Gödelian zone metadata (undefined for legacy facts — backward compat)
    logicZone: obj.logicZone,
    decidability: obj.decidability,
    // U3/U7: stratum ceiling + undecidable resolution
    stratumCeiling: obj.stratumCeiling,
    undecidableResolution: obj.undecidableResolution
      ? {
          ...obj.undecidableResolution,
          resolvedAt: BigInt(obj.undecidableResolution.resolvedAt),
        }
      : undefined,
    // Multidimensional confidence (backward compat: undefined for legacy facts)
    multidimensionalConfidence: obj.multidimensionalConfidence,
  };
}

function serializeReference(ref: FactReference): Uint8Array {
  const json = JSON.stringify({
    fromFactId: ref.fromFactId,
    toFactId: ref.toFactId,
    referenceType: ref.referenceType,
    strength: ref.strength,
  });
  return new TextEncoder().encode(json);
}

function deserializeReference(bytes: Uint8Array): FactReference {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    fromFactId: obj.fromFactId,
    toFactId: obj.toFactId,
    referenceType: obj.referenceType as ReferenceType,
    strength: obj.strength,
  };
}

function factIdToKey(factId: string): Uint8Array {
  return sha512_256(factId);
}

function referenceToKey(fromFactId: string, toFactId: string): Uint8Array {
  return sha512_256(`${fromFactId}:${toFactId}`);
}

// ============ Knowledge State Tree ============

export class KnowledgeStateTree {
  private factSmt: InMemorySMT;
  private referenceSmt: InMemorySMT;
  private factCache: Map<string, FactState> = new Map();
  private referenceCache: Map<string, FactReference> = new Map();

  // Domain tracking
  private domains: Map<string, DomainState> = new Map();

  // Indexes for graph traversal
  private outgoingRefs: Map<string, Set<string>> = new Map(); // factId -> toFactIds
  private incomingRefs: Map<string, Set<string>> = new Map(); // factId -> fromFactIds

  // Statistics
  private factCount: number = 0;
  private activeFactCount: number = 0;

  constructor() {
    this.factSmt = new InMemorySMT();
    this.referenceSmt = new InMemorySMT();
  }

  // ============ Roots ============

  getFactRoot(): MerkleRoot {
    return this.factSmt.getRoot();
  }

  getReferenceRoot(): MerkleRoot {
    return this.referenceSmt.getRoot();
  }

  /**
   * Combined knowledge root
   */
  getKnowledgeRoot(): MerkleRoot {
    const combined = `${this.getFactRoot()}:${this.getReferenceRoot()}`;
    return sha512_256Hex(combined);
  }

  // ============ Fact Operations ============

  /**
   * Get fact by ID
   */
  getFact(factId: string): FactState | null {
    const cached = this.factCache.get(factId);
    if (cached) return cached;

    const key = factIdToKey(factId);
    const value = this.factSmt.get(key);
    if (!value) return null;

    const fact = deserializeFactState(value);
    this.factCache.set(factId, fact);
    return fact;
  }

  /**
   * Check if fact exists
   */
  hasFact(factId: string): boolean {
    return this.getFact(factId) !== null;
  }

  /**
   * Check if fact is active (not revoked/superseded)
   */
  isActiveFact(factId: string): boolean {
    const fact = this.getFact(factId);
    return fact?.status === "active";
  }

  /**
   * Add a new verified fact
   */
  addFact(fact: Omit<FactState, "referenceRoot">): MerkleRoot {
    if (this.hasFact(fact.id)) {
      throw new StateError("Fact already exists", "INVALID_TRANSITION", {
        factId: fact.id,
      });
    }

    const factState: FactState = {
      ...fact,
      referenceRoot: "0".repeat(64), // Empty reference tree initially
    };

    const key = factIdToKey(fact.id);
    const value = serializeFactState(factState);
    const newRoot = this.factSmt.set(key, value);

    this.factCache.set(fact.id, factState);
    this.factCount++;
    if (fact.status === "active") {
      this.activeFactCount++;
    }

    // Update domain stats
    this.updateDomain(fact.domain, 1);

    return newRoot;
  }

  /**
   * Update fact state
   */
  updateFact(factId: string, update: Partial<FactState>): MerkleRoot {
    const existing = this.getFact(factId);
    if (!existing) {
      throw new StateError("Fact not found", "FACT_NOT_FOUND", { factId });
    }

    const wasActive = existing.status === "active";
    const updated: FactState = {
      ...existing,
      ...update,
    };
    const isActive = updated.status === "active";

    const key = factIdToKey(factId);
    const value = serializeFactState(updated);
    const newRoot = this.factSmt.set(key, value);

    this.factCache.set(factId, updated);

    // Update active count
    if (wasActive && !isActive) {
      this.activeFactCount--;
    } else if (!wasActive && isActive) {
      this.activeFactCount++;
    }

    return newRoot;
  }

  // ============ Citation Economics ============

  /**
   * Increment citation count when a fact is queried/used.
   * Only increments for active facts (silently skips inactive).
   */
  incrementCitationCount(factId: string, blockNumber: bigint): MerkleRoot {
    const fact = this.getFact(factId);
    if (!fact) {
      throw new StateError("Fact not found", "FACT_NOT_FOUND", { factId });
    }
    // Only track citations for active facts
    if (fact.status !== "active") {
      return this.getKnowledgeRoot();
    }
    return this.updateFact(factId, {
      citationCount: fact.citationCount + 1,
      lastCitedBlock: blockNumber,
    });
  }

  /**
   * Record citation rewards earned by a fact.
   */
  recordCitationReward(factId: string, amount: bigint): MerkleRoot {
    const fact = this.getFact(factId);
    if (!fact) {
      throw new StateError("Fact not found", "FACT_NOT_FOUND", { factId });
    }
    return this.updateFact(factId, {
      totalCitationRewards: (fact.totalCitationRewards ?? 0n) + amount,
    });
  }

  /**
   * Challenge a fact
   */
  challengeFact(factId: string, blockNumber: bigint): MerkleRoot {
    const fact = this.getFact(factId);
    if (!fact) {
      throw new StateError("Fact not found", "FACT_NOT_FOUND", { factId });
    }

    if (fact.status !== "active") {
      throw new StateError(
        "Can only challenge active facts",
        "INVALID_TRANSITION",
        {
          factId,
          status: fact.status,
        },
      );
    }

    return this.updateFact(factId, {
      status: "challenged",
      challengeCount: fact.challengeCount + 1,
      lastChallengedAt: blockNumber,
    });
  }

  /**
   * Resolve a challenge (fact upheld or revoked)
   */
  resolveChallenge(
    factId: string,
    upheld: boolean,
    newConfidence?: Confidence,
  ): MerkleRoot {
    const fact = this.getFact(factId);
    if (!fact) {
      throw new StateError("Fact not found", "FACT_NOT_FOUND", { factId });
    }

    if (fact.status !== "challenged") {
      throw new StateError(
        "Fact is not under challenge",
        "INVALID_TRANSITION",
        {
          factId,
          status: fact.status,
        },
      );
    }

    if (upheld) {
      return this.updateFact(factId, {
        status: "active",
        confidence: newConfidence ?? fact.confidence,
      });
    } else {
      return this.revokeFact(factId);
    }
  }

  /**
   * Revoke a fact (proven false)
   */
  revokeFact(factId: string): MerkleRoot {
    const fact = this.getFact(factId);
    if (!fact) {
      throw new StateError("Fact not found", "FACT_NOT_FOUND", { factId });
    }

    const oldConfidence = fact.confidence;

    // Revoke the fact first (confidence effectively becomes 0)
    const root = this.updateFact(factId, {
      status: "revoked",
    });

    // Propagate the confidence drop through the knowledge graph via BFS
    // This replaces the naive flat -0.2 with proper graph-aware propagation
    this.propagateConfidenceChange(factId, oldConfidence, 0 as Confidence);

    return root;
  }

  /**
   * Supersede a fact with a newer one
   */
  supersedeFact(oldFactId: string, newFactId: string): MerkleRoot {
    const oldFact = this.getFact(oldFactId);
    if (!oldFact) {
      throw new StateError("Old fact not found", "FACT_NOT_FOUND", {
        factId: oldFactId,
      });
    }

    const newFact = this.getFact(newFactId);
    if (!newFact) {
      throw new StateError("New fact not found", "FACT_NOT_FOUND", {
        factId: newFactId,
      });
    }

    // Mark old as superseded
    this.updateFact(oldFactId, { status: "superseded" });

    // Add supersedes reference
    this.addReference({
      fromFactId: newFactId,
      toFactId: oldFactId,
      referenceType: "extends",
      strength: 1.0,
    });

    return this.getKnowledgeRoot();
  }

  // ============ Reference Operations ============

  /**
   * Get reference between two facts
   */
  getReference(fromFactId: string, toFactId: string): FactReference | null {
    const cacheKey = `${fromFactId}:${toFactId}`;
    const cached = this.referenceCache.get(cacheKey);
    if (cached) return cached;

    const key = referenceToKey(fromFactId, toFactId);
    const value = this.referenceSmt.get(key);
    if (!value) return null;

    const ref = deserializeReference(value);
    this.referenceCache.set(cacheKey, ref);
    return ref;
  }

  /**
   * Add a reference between facts
   */
  addReference(ref: FactReference): MerkleRoot {
    // Verify both facts exist
    if (!this.hasFact(ref.fromFactId)) {
      throw new StateError("Source fact not found", "FACT_NOT_FOUND", {
        factId: ref.fromFactId,
      });
    }
    if (!this.hasFact(ref.toFactId)) {
      throw new StateError("Target fact not found", "FACT_NOT_FOUND", {
        factId: ref.toFactId,
      });
    }

    const key = referenceToKey(ref.fromFactId, ref.toFactId);
    const value = serializeReference(ref);
    const newRoot = this.referenceSmt.set(key, value);

    const cacheKey = `${ref.fromFactId}:${ref.toFactId}`;
    this.referenceCache.set(cacheKey, ref);

    // Update indexes
    if (!this.outgoingRefs.has(ref.fromFactId)) {
      this.outgoingRefs.set(ref.fromFactId, new Set());
    }
    this.outgoingRefs.get(ref.fromFactId)!.add(ref.toFactId);

    if (!this.incomingRefs.has(ref.toFactId)) {
      this.incomingRefs.set(ref.toFactId, new Set());
    }
    this.incomingRefs.get(ref.toFactId)!.add(ref.fromFactId);

    return newRoot;
  }

  /**
   * Get all outgoing references from a fact
   */
  getOutgoingReferences(factId: string): string[] {
    return Array.from(this.outgoingRefs.get(factId) ?? []);
  }

  /**
   * Get all incoming references to a fact
   */
  getIncomingReferences(factId: string): string[] {
    return Array.from(this.incomingRefs.get(factId) ?? []);
  }

  /**
   * Get facts that support a given fact
   */
  getSupportingFacts(factId: string): FactState[] {
    const incoming = this.getIncomingReferences(factId);
    const supporting: FactState[] = [];

    for (const fromId of incoming) {
      const ref = this.getReference(fromId, factId);
      if (ref?.referenceType === "supports") {
        const fact = this.getFact(fromId);
        if (fact && fact.status === "active") {
          supporting.push(fact);
        }
      }
    }

    return supporting;
  }

  /**
   * Get facts that contradict a given fact
   */
  getContradictingFacts(factId: string): FactState[] {
    const allRefs = [
      ...this.getIncomingReferences(factId),
      ...this.getOutgoingReferences(factId),
    ];

    const contradicting: FactState[] = [];
    const seen = new Set<string>();

    for (const otherId of allRefs) {
      if (seen.has(otherId)) continue;
      seen.add(otherId);

      const ref =
        this.getReference(otherId, factId) ??
        this.getReference(factId, otherId);
      if (ref?.referenceType === "contradicts") {
        const fact = this.getFact(otherId);
        if (fact && fact.status === "active") {
          contradicting.push(fact);
        }
      }
    }

    return contradicting;
  }

  /**
   * Get facts that extend a given fact
   */
  getExtendingFacts(factId: string): FactState[] {
    const incoming = this.getIncomingReferences(factId);
    const extending: FactState[] = [];

    for (const fromId of incoming) {
      const ref = this.getReference(fromId, factId);
      if (ref?.referenceType === "extends") {
        const fact = this.getFact(fromId);
        if (fact && fact.status === "active") {
          extending.push(fact);
        }
      }
    }

    return extending;
  }

  // ============ Queries ============

  /**
   * Get facts by domain
   */
  getFactsByDomain(domain: string): FactState[] {
    const facts: FactState[] = [];
    for (const fact of this.factCache.values()) {
      if (fact.domain === domain && fact.status === "active") {
        facts.push(fact);
      }
    }
    return facts;
  }

  /**
   * Get facts by status
   */
  getFactsByStatus(status: FactStatus): FactState[] {
    const facts: FactState[] = [];
    for (const fact of this.factCache.values()) {
      if (fact.status === status) {
        facts.push(fact);
      }
    }
    return facts;
  }

  /**
   * Get all references in the graph (for persistence)
   */
  getAllReferences(): FactReference[] {
    return Array.from(this.referenceCache.values());
  }

  /**
   * Search facts by claim text (simple substring match)
   */
  searchFacts(query: string): FactState[] {
    const lowerQuery = query.toLowerCase();
    const results: FactState[] = [];

    for (const fact of this.factCache.values()) {
      if (
        fact.claim.toLowerCase().includes(lowerQuery) &&
        fact.status === "active"
      ) {
        results.push(fact);
      }
    }

    return results;
  }

  /**
   * Get total fact count
   */
  getFactCount(): number {
    return this.factCount;
  }

  /**
   * Get active fact count
   */
  getActiveFactCount(): number {
    return this.activeFactCount;
  }

  /**
   * Get all domains
   */
  getDomains(): DomainState[] {
    return Array.from(this.domains.values());
  }

  // ============ Proofs ============

  /**
   * Generate proof of fact existence
   */
  proveFact(factId: string, blockNumber: bigint): FactProof {
    const fact = this.getFact(factId);
    const key = factIdToKey(factId);
    const proof = this.factSmt.prove(key);

    if (!fact) {
      throw new StateError("Fact not found", "FACT_NOT_FOUND", { factId });
    }

    return {
      factId,
      fact,
      stateRoot: this.getKnowledgeRoot(),
      blockNumber,
      proof,
    };
  }

  /**
   * Verify a fact proof
   */
  static verifyFactProof(proof: FactProof): boolean {
    // Verify Merkle proof
    const validProof = InMemorySMT.verify(proof.proof);
    if (!validProof) return false;

    // Verify fact content matches
    if (proof.proof.value) {
      const fact = deserializeFactState(proof.proof.value);
      return (
        fact.id === proof.fact.id &&
        fact.claim === proof.fact.claim &&
        fact.status === proof.fact.status
      );
    }

    return false;
  }

  // ============ Lazy Confidence Decay (GAP-008) ============

  // STRATUM_TO_DECAY_CATEGORY imported from core/knowledge/ontology/types.ts

  /**
   * Get effective (time-decayed) confidence for a fact
   *
   * GAP-008: Lazy decay calculation - doesn't modify stored values
   *
   * Following GAP-009's lazy evaluation pattern, this calculates the
   * effective confidence based on time elapsed without modifying the
   * stored confidence value. Consumers can use this to get accurate
   * confidence while preserving the original verification record.
   *
   * @param fact - The fact to calculate decay for
   * @param currentBlock - Current block number
   * @returns Decay information including effective confidence
   */
  getEffectiveConfidence(fact: FactState, currentBlock: bigint): DecayInfo {
    const blocksSinceVerification = Number(currentBlock - fact.verifiedAt);

    // Prefer fact-level epistemicCategory (set at creation from domain config),
    // fall back to stratum→category mapping for legacy facts without it.
    // Final fallback: "empirical" (temporal, decaying) — safe default.
    const category: EpistemicCategory =
      fact.epistemicCategory ??
      (fact.stratum
        ? (STRATUM_TO_DECAY_CATEGORY[fact.stratum] ?? "empirical")
        : "empirical");

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
   * Get fact with effective confidence calculated
   *
   * GAP-008: Convenience method that returns both the fact and its
   * decay information in a single call.
   *
   * @param factId - ID of the fact to retrieve
   * @param currentBlock - Current block number for decay calculation
   * @returns Fact with decay info, or undefined if fact doesn't exist
   */
  getFactWithDecay(
    factId: string,
    currentBlock: bigint,
  ): { fact: FactState; decayInfo: DecayInfo } | undefined {
    const fact = this.getFact(factId);
    if (!fact) {
      return undefined;
    }

    const decayInfo = this.getEffectiveConfidence(fact, currentBlock);
    return { fact, decayInfo };
  }

  /**
   * Get multiple facts with effective confidence
   *
   * GAP-008: Batch version of getFactWithDecay for efficiency.
   *
   * @param factIds - IDs of facts to retrieve
   * @param currentBlock - Current block number for decay calculation
   * @returns Array of facts with decay info (excludes missing facts)
   */
  getFactsWithDecay(
    factIds: string[],
    currentBlock: bigint,
  ): Array<{ factId: string; fact: FactState; decayInfo: DecayInfo }> {
    const results: Array<{
      factId: string;
      fact: FactState;
      decayInfo: DecayInfo;
    }> = [];

    for (const factId of factIds) {
      const result = this.getFactWithDecay(factId, currentBlock);
      if (result) {
        results.push({ factId, ...result });
      }
    }

    return results;
  }

  // ============ GAP-009: Fact Expiration ============

  /**
   * Mapping from ontological stratum to epistemic profile
   *
   * This maps the stratum types used in FactState to the epistemic profiles
   * that define decay rates and reverification requirements.
   */
  private static readonly STRATUM_TO_PROFILE: Record<
    OntologicalStratum,
    EpistemicProfile
  > = {
    fundamental: EPISTEMIC_PROFILES.formal,
    physical: EPISTEMIC_PROFILES.empirical,
    chemical: EPISTEMIC_PROFILES.empirical,
    biological: EPISTEMIC_PROFILES.empirical,
    cognitive: EPISTEMIC_PROFILES.social,
    social: EPISTEMIC_PROFILES.social,
    technological: EPISTEMIC_PROFILES.protocol,
  };

  /**
   * Check if a fact is expired
   *
   * GAP-009: Expiration conditions (any triggers expiration):
   * 1. Explicit expiration: expiresAt < currentBlock
   * 2. Confidence decay: confidence < minConfidenceThreshold (0.3)
   * 3. Reverification required but stale: age > 2 * decayHalfLife without reverification
   *
   * @param fact - The fact to check
   * @param currentBlock - Current block number
   * @param params - Expiration parameters (optional, defaults to DEFAULT_EXPIRATION_PARAMS)
   * @returns ExpirationResult with expired status and reason
   */
  isFactExpired(
    fact: FactState,
    currentBlock: bigint,
    params: ExpirationParams = DEFAULT_EXPIRATION_PARAMS,
  ): ExpirationResult {
    // Facts that are already not active cannot be "expired"
    if (fact.status !== "active") {
      return { expired: false };
    }

    // Condition 1: Explicit expiration
    if (fact.expiresAt !== undefined && currentBlock >= fact.expiresAt) {
      return {
        expired: true,
        reason: "explicit_expiration",
        details: {
          blocksFromExpiration: currentBlock - fact.expiresAt,
        },
      };
    }

    // Condition 2: Effective (decayed) confidence below threshold
    // Must use getEffectiveConfidence() instead of stored confidence,
    // otherwise a fact verified at 0.9 that has decayed to 0.07 would
    // appear alive because stored confidence is still 0.9.
    const { effectiveConfidence } = this.getEffectiveConfidence(
      fact,
      currentBlock,
    );
    if (effectiveConfidence < params.minConfidenceThreshold) {
      return {
        expired: true,
        reason: "confidence_decay",
        details: {
          currentConfidence: effectiveConfidence,
          threshold: params.minConfidenceThreshold,
        },
      };
    }

    // Condition 3: Reverification required but stale
    if (params.useStratumPolicy && fact.stratum) {
      const profile = KnowledgeStateTree.STRATUM_TO_PROFILE[fact.stratum];

      if (profile?.requiresReverification && profile.decayHalfLife !== null) {
        // Use lastReverifiedAt if available, otherwise use verifiedAt
        const lastVerification = fact.lastReverifiedAt ?? fact.verifiedAt;
        const age = Number(currentBlock - lastVerification);
        const staleThreshold =
          profile.decayHalfLife * params.stalenessMultiplier;

        if (age > staleThreshold) {
          return {
            expired: true,
            reason: "reverification_required",
            details: {
              blocksFromExpiration: BigInt(Math.floor(age - staleThreshold)),
              threshold: staleThreshold,
            },
          };
        }
      }
    }

    return { expired: false };
  }

  /**
   * Mark a fact as expired
   *
   * @param factId - ID of the fact to expire
   * @param blockNumber - Block number when expiration occurred
   * @param reason - Reason for expiration
   * @returns New merkle root
   */
  expireFact(
    factId: string,
    _blockNumber: bigint,
    _reason: ExpirationReason,
  ): MerkleRoot {
    return this.updateFact(factId, {
      status: "expired",
    });
  }

  /**
   * Get all facts that are expired or near expiration
   *
   * @param currentBlock - Current block number
   * @param params - Expiration parameters
   * @returns Array of expired facts with expiration details
   */
  getExpiredFacts(
    currentBlock: bigint,
    params: ExpirationParams = DEFAULT_EXPIRATION_PARAMS,
  ): Array<{ fact: FactState; result: ExpirationResult }> {
    const expired: Array<{ fact: FactState; result: ExpirationResult }> = [];

    for (const [_factId, fact] of this.factCache) {
      // Include already-expired facts
      if (fact.status === "expired") {
        expired.push({
          fact,
          result: { expired: true, reason: "explicit_expiration" },
        });
        continue;
      }

      // Check active facts for expiration
      if (fact.status === "active") {
        const result = this.isFactExpired(fact, currentBlock, params);
        if (result.expired) {
          expired.push({ fact, result });
        }
      }
    }

    return expired;
  }

  /**
   * Get fact by ID with optional expiration checking
   *
   * GAP-009: When checkExpiration is true, this method will:
   * 1. Check if the fact is expired
   * 2. If expired and updateIfExpired is true, update the status to 'expired'
   * 3. Return the fact (possibly with updated status)
   *
   * @param factId - ID of the fact to retrieve
   * @param options - Options for expiration checking
   * @returns Fact state or null if not found
   */
  getFactWithExpiration(
    factId: string,
    options: GetFactOptions,
  ): FactState | null {
    const fact = this.getFact(factId);
    if (!fact) return null;

    // If not checking expiration, return as-is
    if (!options.checkExpiration || options.currentBlock === undefined) {
      return fact;
    }

    const params = options.expirationParams ?? DEFAULT_EXPIRATION_PARAMS;
    const result = this.isFactExpired(fact, options.currentBlock, params);

    if (result.expired && fact.status === "active") {
      // Lazily update status if requested
      if (options.updateIfExpired && params.lazyExpiration) {
        this.expireFact(factId, options.currentBlock, result.reason!);
        return { ...fact, status: "expired" };
      }
    }

    return fact;
  }

  /**
   * Re-verify a fact (reset reverification timestamp)
   *
   * Called when a fact passes re-verification to reset its staleness clock.
   *
   * @param factId - ID of the fact
   * @param blockNumber - Block number of re-verification
   * @param newConfidence - Updated confidence (optional)
   * @returns New merkle root
   */
  reverifyFact(
    factId: string,
    blockNumber: bigint,
    newConfidence?: Confidence,
  ): MerkleRoot {
    const fact = this.getFact(factId);
    if (!fact) {
      throw new StateError("Fact not found", "FACT_NOT_FOUND", { factId });
    }

    const updates: Partial<FactState> = {
      lastReverifiedAt: blockNumber,
      verifierCount: fact.verifierCount + 1,
    };

    // If fact was expired, reactivate it
    if (fact.status === "expired") {
      updates.status = "active";
    }

    // Update confidence if provided
    if (newConfidence !== undefined) {
      updates.confidence = newConfidence;
    }

    return this.updateFact(factId, updates);
  }

  // ============ Confidence Propagation ============

  /**
   * Propagation constants
   */
  private static readonly PROPAGATION_THRESHOLD = 0.01; // 1% change triggers propagation
  private static readonly MAX_PROPAGATION_DEPTH = 10; // Max hops in dependency chain

  /**
   * Propagate confidence change to derived facts
   *
   * When a fact's confidence changes, facts derived from it may need updating.
   * Uses BFS to propagate changes up the dependency graph.
   */
  propagateConfidenceChange(
    factId: string,
    oldConfidence: Confidence,
    newConfidence: Confidence,
  ): PropagationResult {
    const changeMagnitude = Math.abs(newConfidence - oldConfidence);

    // Only propagate significant changes
    if (changeMagnitude < KnowledgeStateTree.PROPAGATION_THRESHOLD) {
      return {
        updatedFacts: [],
        cyclesDetected: [],
        depthReached: 0,
      };
    }

    const result: PropagationResult = {
      updatedFacts: [],
      cyclesDetected: [],
      depthReached: 0,
    };

    // BFS through derived facts
    const visited = new Set<string>();
    const queue: Array<{ id: string; depth: number }> = [];

    // Start with facts that cite/derive from this one
    const dependents = this.getIncomingReferences(factId);
    for (const depId of dependents) {
      queue.push({ id: depId, depth: 1 });
    }

    while (queue.length > 0) {
      const { id, depth } = queue.shift()!;

      // Skip if already visited (cycle detection)
      if (visited.has(id)) {
        if (!result.cyclesDetected.includes(id)) {
          result.cyclesDetected.push(id);
        }
        continue;
      }

      // Respect depth limit
      if (depth > KnowledgeStateTree.MAX_PROPAGATION_DEPTH) {
        result.depthReached = depth;
        continue;
      }

      visited.add(id);
      result.depthReached = Math.max(result.depthReached, depth);

      const depFact = this.getFact(id);
      if (!depFact || depFact.status !== "active") continue;

      // Calculate new confidence for dependent fact
      const newDepConfidence = this.calculateDerivedConfidence(id);

      if (newDepConfidence !== null) {
        const depChange = Math.abs(newDepConfidence - depFact.confidence);

        if (depChange >= KnowledgeStateTree.PROPAGATION_THRESHOLD) {
          // Update the fact
          this.updateFact(id, {
            confidence: newDepConfidence as Confidence,
          });

          result.updatedFacts.push({
            factId: id,
            oldConfidence: depFact.confidence,
            newConfidence: newDepConfidence,
            depth,
          });

          // Continue propagation to facts derived from this one
          const nextDependents = this.getIncomingReferences(id);
          for (const nextId of nextDependents) {
            if (!visited.has(nextId)) {
              queue.push({ id: nextId, depth: depth + 1 });
            }
          }
        }
      }
    }

    return result;
  }

  /**
   * Calculate confidence for a derived fact based on its premises
   *
   * Uses weakest-link principle: confidence = min(premise_confidences) × inference_strength
   */
  calculateDerivedConfidence(factId: string): Confidence | null {
    const outgoing = this.getOutgoingReferences(factId);
    if (outgoing.length === 0) return null;

    // Get premise confidences
    const premiseConfidences: number[] = [];
    let inferenceStrength = 0.99; // Default to deductive
    let hasSupportingRefs = false;

    for (const toId of outgoing) {
      const ref = this.getReference(factId, toId);
      if (!ref) continue;

      // Only consider derivation relationships (supports, extends, derives_from)
      if (
        ref.referenceType === "supports" ||
        ref.referenceType === "extends" ||
        ref.referenceType === "derives_from"
      ) {
        hasSupportingRefs = true;
        const premise = this.getFact(toId);
        if (premise && premise.status === "active") {
          premiseConfidences.push(premise.confidence);
          // Use lowest inference strength
          inferenceStrength = Math.min(inferenceStrength, ref.strength);
        }
      }
    }

    // No derivation relationships at all - not a derived fact
    if (!hasSupportingRefs) return null;

    // Had supporting refs but all premises are revoked/inactive - derivation chain broken
    if (premiseConfidences.length === 0) return 0 as Confidence;

    // Weakest link principle
    const minPremise = Math.min(...premiseConfidences);
    return (minPremise * inferenceStrength) as Confidence;
  }

  /**
   * Detect circular dependencies starting from a fact
   *
   * Uses Tarjan's algorithm for cycle detection
   */
  detectCircularDependencies(
    startFactId: string,
  ): CircularDependencyInfo | null {
    const visited = new Set<string>();
    const recursionStack = new Set<string>();
    const path: string[] = [];

    const dfs = (factId: string): string[] | null => {
      visited.add(factId);
      recursionStack.add(factId);
      path.push(factId);

      const outgoing = this.getOutgoingReferences(factId);
      for (const toId of outgoing) {
        if (!visited.has(toId)) {
          const cycle = dfs(toId);
          if (cycle) return cycle;
        } else if (recursionStack.has(toId)) {
          // Found cycle - return path from cycle start
          const cycleStart = path.indexOf(toId);
          return path.slice(cycleStart);
        }
      }

      path.pop();
      recursionStack.delete(factId);
      return null;
    };

    const cycle = dfs(startFactId);
    if (cycle) {
      return {
        facts: cycle,
        tightness: cycle.length <= 3 ? 1.0 : 1.0 / Math.log2(cycle.length),
        penalty:
          0.4 * (cycle.length <= 3 ? 1.0 : 1.0 / Math.log2(cycle.length)),
      };
    }

    return null;
  }

  /**
   * Update fact confidence with propagation
   *
   * Higher-level API that handles both the update and propagation
   */
  updateFactConfidenceWithPropagation(
    factId: string,
    newConfidence: Confidence,
  ): {
    root: MerkleRoot;
    propagation: PropagationResult;
  } {
    const existing = this.getFact(factId);
    if (!existing) {
      throw new StateError("Fact not found", "FACT_NOT_FOUND", { factId });
    }

    const oldConfidence = existing.confidence;

    // Check for circular dependencies
    const cycle = this.detectCircularDependencies(factId);
    if (cycle) {
      // Apply cycle penalty
      newConfidence = Math.max(
        0.1,
        newConfidence * (1 - cycle.penalty),
      ) as Confidence;
    }

    // Update the fact
    const root = this.updateFact(factId, { confidence: newConfidence });

    // Propagate to dependents
    const propagation = this.propagateConfidenceChange(
      factId,
      oldConfidence,
      newConfidence,
    );

    return { root, propagation };
  }

  // ============ Domain Management ============

  private updateDomain(domainId: string, factDelta: number): void {
    const existing = this.domains.get(domainId);
    if (existing) {
      existing.factCount += factDelta;
    } else {
      this.domains.set(domainId, {
        id: domainId,
        name: domainId,
        factCount: 1,
      });
    }
  }

  /**
   * Create or update a domain
   */
  setDomain(domain: DomainState): void {
    this.domains.set(domain.id, domain);
  }

  // ============ Utilities ============

  /**
   * Clone for speculative execution
   */
  clone(): KnowledgeStateTree {
    const cloned = new KnowledgeStateTree();

    for (const [factId, fact] of this.factCache) {
      cloned.factCache.set(factId, { ...fact });
      const key = factIdToKey(factId);
      cloned.factSmt.set(key, serializeFactState(fact));
    }

    for (const [refKey, ref] of this.referenceCache) {
      cloned.referenceCache.set(refKey, { ...ref });
      const key = referenceToKey(ref.fromFactId, ref.toFactId);
      cloned.referenceSmt.set(key, serializeReference(ref));
    }

    for (const [factId, refs] of this.outgoingRefs) {
      cloned.outgoingRefs.set(factId, new Set(refs));
    }

    for (const [factId, refs] of this.incomingRefs) {
      cloned.incomingRefs.set(factId, new Set(refs));
    }

    for (const [domainId, domain] of this.domains) {
      cloned.domains.set(domainId, { ...domain });
    }

    cloned.factCount = this.factCount;
    cloned.activeFactCount = this.activeFactCount;

    return cloned;
  }
}

// ============ Factory Functions ============

/**
 * Generate fact ID from claim content
 */
export function generateFactId(
  claim: string,
  domain: string,
  submitter: string,
): string {
  return sha512_256Hex(`fact:${domain}:${claim}:${submitter}`);
}

/**
 * Create knowledge tree with genesis axioms
 */
export function createKnowledgeTreeWithAxioms(
  axioms: Array<{ id: string; claim: string; domain: string }>,
): KnowledgeStateTree {
  const tree = new KnowledgeStateTree();

  for (const axiom of axioms) {
    tree.addFact({
      id: axiom.id,
      claim: axiom.claim,
      domain: axiom.domain,
      confidence: 1.0 as Confidence, // Axioms have full confidence
      verifiedAt: 0n,
      verifierCount: 0, // Axioms don't need verification
      challengeCount: 0,
      citationCount: 0,
      status: "active",
    });
  }

  return tree;
}

// ============ Constants ============

/**
 * Minimum confidence to consider a fact "established"
 */
export const ESTABLISHED_CONFIDENCE = 0.8;

/**
 * Confidence below which facts should be re-verified
 */
export const REVERIFICATION_THRESHOLD = 0.6;

/**
 * Maximum references per fact (to prevent spam)
 */
export const MAX_REFERENCES_PER_FACT = 100;

// ============ Propagation Types ============

/**
 * Result of confidence propagation
 */
export interface PropagationResult {
  /** Facts that were updated during propagation */
  updatedFacts: Array<{
    factId: string;
    oldConfidence: Confidence;
    newConfidence: Confidence;
    depth: number;
  }>;

  /** Fact IDs where cycles were detected and skipped */
  cyclesDetected: string[];

  /** Maximum depth reached during propagation */
  depthReached: number;
}

/**
 * Information about circular dependencies
 */
export interface CircularDependencyInfo {
  /** Fact IDs in the cycle */
  facts: string[];

  /** How tight the cycle is (smaller = tighter, more problematic) */
  tightness: number;

  /** Confidence penalty to apply (0-1) */
  penalty: number;
}

// ============ GAP-009: Expiration Types ============

/**
 * Reason why a fact is considered expired
 */
export type ExpirationReason =
  | "explicit_expiration" // expiresAt < currentBlock
  | "confidence_decay" // confidence < minConfidenceThreshold
  | "reverification_required" // age > 2 * decayHalfLife without reverification
  | "stratum_policy"; // Stratum-specific expiration policy

/**
 * Result of checking if a fact is expired
 */
export interface ExpirationResult {
  /** Whether the fact is expired */
  expired: boolean;

  /** Reason for expiration (if expired) */
  reason?: ExpirationReason;

  /** Additional context about expiration */
  details?: {
    /** Blocks until/since expiration */
    blocksFromExpiration?: bigint;
    /** Current confidence level */
    currentConfidence?: number;
    /** Threshold that was violated */
    threshold?: number;
  };
}

/**
 * Parameters controlling fact expiration behavior
 */
export interface ExpirationParams {
  /** Minimum confidence before automatic expiration (default: 0.3) */
  minConfidenceThreshold: number;

  /** Multiplier on decayHalfLife for staleness check (default: 2.0) */
  stalenessMultiplier: number;

  /** Whether to update status lazily on query (default: true) */
  lazyExpiration: boolean;

  /** Whether to include stratum in expiration calculation (default: true) */
  useStratumPolicy: boolean;
}

/**
 * Default expiration parameters
 * Aligned with DEFAULT_TRUTH_PARAMETERS.minConfidenceThreshold
 */
export const DEFAULT_EXPIRATION_PARAMS: ExpirationParams = {
  minConfidenceThreshold: 0.3, // Facts below 30% confidence are expired
  stalenessMultiplier: 2.0, // 2x half-life = ~25% original confidence
  lazyExpiration: true, // Update status on query
  useStratumPolicy: true, // Respect stratum-specific rules
};

/**
 * Options for getFact with expiration checking
 */
export interface GetFactOptions {
  /** Whether to check expiration status (default: false for backward compat) */
  checkExpiration?: boolean;

  /** Current block number (required if checkExpiration is true) */
  currentBlock?: bigint;

  /** Expiration parameters to use (defaults to DEFAULT_EXPIRATION_PARAMS) */
  expirationParams?: ExpirationParams;

  /** Whether to update fact status if expired (requires checkExpiration) */
  updateIfExpired?: boolean;
}
