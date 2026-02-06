/**
 * Evidence State Tree
 *
 * Sparse Merkle Tree for evidence storage and management.
 * This is the on-chain state for the Evidence Management System.
 *
 * Key: hash(evidenceId) - 32 bytes
 * Value: serialized EvidenceState
 *
 * Supports:
 * - Evidence storage with version chains
 * - Archive location tracking
 * - Deduplication fingerprinting
 * - Expiry processing
 * - Chainlink feed registry
 */

import { InMemorySMT, type MerkleRoot } from "../core/crypto/merkle.js";
import { sha512_256, sha512_256Hex } from "../core/crypto/hashing.js";
import type { DID } from "../core/agents/identity/did.js";
import type {
  Evidence,
  EvidenceType,
  EvidenceStatus,
  ArchiveLocation,
  ArchiveRequest,
  ChainlinkFeedRegistration,
  ChainlinkFeedType,
} from "../core/evidence/types.js";
import { StateError } from "./types.js";
import {
  DeduplicationIndex,
  generateFingerprint,
  detectDuplicate,
} from "../core/evidence/deduplication.js";
import {
  calculateFreshness,
  processExpirations,
} from "../core/evidence/expiry.js";
import {
  ArchiveQueue,
  calculateArchiveBonus,
} from "../core/evidence/archiving.js";
import { ChainlinkFeedRegistry } from "../core/evidence/chainlink.js";

// ============ Serialization ============

function serializeEvidence(evidence: Evidence): Uint8Array {
  const json = JSON.stringify({
    id: evidence.id,
    type: evidence.type,
    originalSource: evidence.originalSource,
    contentHash: evidence.contentHash,
    perceptualHash: evidence.perceptualHash,
    archives: evidence.archives.map((a) => ({
      ...a,
      archivedAtBlock: a.archivedAtBlock.toString(),
    })),
    firstSubmitter: evidence.firstSubmitter,
    confirmedBy: evidence.confirmedBy,
    status: evidence.status,
    reliabilityScore: evidence.reliabilityScore,
    relatedFactIds: evidence.relatedFactIds,
    submittedAt: evidence.submittedAt,
    submittedAtBlock: evidence.submittedAtBlock.toString(),
    version: {
      ...evidence.version,
      versionChain: evidence.version.versionChain,
    },
    expiry: {
      ...evidence.expiry,
      heartbeatConfig: evidence.expiry.heartbeatConfig
        ? {
            ...evidence.expiry.heartbeatConfig,
            lastHeartbeatBlock:
              evidence.expiry.heartbeatConfig.lastHeartbeatBlock.toString(),
          }
        : undefined,
    },
    oracleMetadata: evidence.oracleMetadata
      ? {
          ...evidence.oracleMetadata,
          roundId: evidence.oracleMetadata.roundId.toString(),
          answeredInRound: evidence.oracleMetadata.answeredInRound.toString(),
        }
      : undefined,
  });
  return new TextEncoder().encode(json);
}

function deserializeEvidence(bytes: Uint8Array): Evidence {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    ...obj,
    archives: obj.archives.map((a: { archivedAtBlock: string }) => ({
      ...a,
      archivedAtBlock: BigInt(a.archivedAtBlock),
    })) as ArchiveLocation[],
    firstSubmitter: obj.firstSubmitter as DID,
    confirmedBy: obj.confirmedBy as DID[],
    status: obj.status as EvidenceStatus,
    submittedAtBlock: BigInt(obj.submittedAtBlock),
    expiry: {
      ...obj.expiry,
      heartbeatConfig: obj.expiry.heartbeatConfig
        ? {
            ...obj.expiry.heartbeatConfig,
            lastHeartbeatBlock: BigInt(
              obj.expiry.heartbeatConfig.lastHeartbeatBlock,
            ),
          }
        : undefined,
    },
    oracleMetadata: obj.oracleMetadata
      ? {
          ...obj.oracleMetadata,
          roundId: BigInt(obj.oracleMetadata.roundId),
          answeredInRound: BigInt(obj.oracleMetadata.answeredInRound),
        }
      : undefined,
  } as Evidence;
}

function evidenceIdToKey(id: string): Uint8Array {
  return sha512_256(`evidence:${id}`);
}

// ============ Evidence Query Types ============

export interface EvidenceQuery {
  type?: EvidenceType;
  status?: EvidenceStatus;
  submitter?: DID;
  factId?: string;
  minReliability?: number;
  includeExpired?: boolean;
  limit?: number;
  offset?: number;
}

export interface EvidenceStatistics {
  totalEvidence: number;
  byType: Record<EvidenceType, number>;
  byStatus: Record<EvidenceStatus, number>;
  totalArchives: number;
  verifiedArchives: number;
  pendingArchiveRequests: number;
  expiringEvidence: number;
  chainlinkFeeds: number;
}

// ============ Evidence State Tree ============

export class EvidenceStateTree {
  private evidenceSmt: InMemorySMT;
  private archiveRequestSmt: InMemorySMT;
  private evidenceCache: Map<string, Evidence> = new Map();

  // Deduplication index
  private deduplicationIndex: DeduplicationIndex;

  // Archive queue
  private archiveQueue: ArchiveQueue;

  // Chainlink feed registry
  private chainlinkRegistry: ChainlinkFeedRegistry;

  // Indexes for efficient queries
  private typeIndex: Map<EvidenceType, Set<string>> = new Map();
  private statusIndex: Map<EvidenceStatus, Set<string>> = new Map();
  private factIndex: Map<string, Set<string>> = new Map();
  private submitterIndex: Map<string, Set<string>> = new Map();
  private versionChainIndex: Map<string, string[]> = new Map(); // original -> versions

  // Statistics
  private evidenceCount: number = 0;
  private archiveCount: number = 0;
  private verifiedArchiveCount: number = 0;

  constructor() {
    this.evidenceSmt = new InMemorySMT();
    this.archiveRequestSmt = new InMemorySMT();
    this.deduplicationIndex = new DeduplicationIndex();
    this.archiveQueue = new ArchiveQueue();
    this.chainlinkRegistry = new ChainlinkFeedRegistry();
  }

  // ============ Roots ============

  getEvidenceRoot(): MerkleRoot {
    return this.evidenceSmt.getRoot();
  }

  getArchiveRequestRoot(): MerkleRoot {
    return this.archiveRequestSmt.getRoot();
  }

  /**
   * Combined evidence layer root
   */
  getRoot(): MerkleRoot {
    const combined = `${this.getEvidenceRoot()}:${this.getArchiveRequestRoot()}`;
    return sha512_256Hex(combined);
  }

  // ============ Evidence Operations ============

  /**
   * Get evidence by ID
   */
  getEvidence(id: string): Evidence | null {
    // Check cache first
    const cached = this.evidenceCache.get(id);
    if (cached) {
      return cached;
    }

    // Check SMT
    const key = evidenceIdToKey(id);
    const value = this.evidenceSmt.get(key);
    if (!value) {
      return null;
    }

    const evidence = deserializeEvidence(value);
    this.evidenceCache.set(id, evidence);
    return evidence;
  }

  /**
   * Submit new evidence
   * Handles deduplication automatically
   */
  submitEvidence(
    evidence: Evidence,
    content: string,
    currentBlock: bigint,
  ): {
    success: boolean;
    evidenceId: string;
    isDuplicate: boolean;
    duplicateOf?: string;
    error?: string;
  } {
    // Generate fingerprint for deduplication
    const fingerprint = generateFingerprint(
      content,
      evidence.originalSource,
      evidence.firstSubmitter,
      evidence.submittedAt,
    );

    // Check for duplicates
    const existingFingerprints = this.deduplicationIndex.getAllFingerprints();
    const duplicateResult = detectDuplicate(
      fingerprint,
      existingFingerprints,
      currentBlock,
    );

    if (
      duplicateResult.isDuplicate &&
      duplicateResult.duplicationType === "exact"
    ) {
      // Exact duplicate - add as confirmer to existing
      const originalId = duplicateResult.matchingEvidenceIds[0];
      const original = this.getEvidence(originalId);
      if (original) {
        // Add confirmer
        const updated = {
          ...original,
          confirmedBy: [...original.confirmedBy, evidence.firstSubmitter],
        };
        this.updateEvidence(updated);

        return {
          success: true,
          evidenceId: originalId,
          isDuplicate: true,
          duplicateOf: originalId,
        };
      }
    }

    // Store new evidence
    const key = evidenceIdToKey(evidence.id);
    const value = serializeEvidence(evidence);
    this.evidenceSmt.set(key, value);

    // Update cache
    this.evidenceCache.set(evidence.id, evidence);

    // Update indexes
    this.addToTypeIndex(evidence.type, evidence.id);
    this.addToStatusIndex(evidence.status, evidence.id);
    this.addToSubmitterIndex(evidence.firstSubmitter, evidence.id);
    for (const factId of evidence.relatedFactIds) {
      this.addToFactIndex(factId, evidence.id);
    }

    // Add to deduplication index
    this.deduplicationIndex.addEvidence(evidence, fingerprint);

    // Update stats
    this.evidenceCount++;
    this.archiveCount += evidence.archives.length;
    this.verifiedArchiveCount += evidence.archives.filter(
      (a) => a.verified,
    ).length;

    return {
      success: true,
      evidenceId: evidence.id,
      isDuplicate: duplicateResult.isDuplicate,
      duplicateOf: duplicateResult.isDuplicate
        ? duplicateResult.matchingEvidenceIds[0]
        : undefined,
    };
  }

  /**
   * Update existing evidence
   */
  updateEvidence(evidence: Evidence): void {
    const existing = this.getEvidence(evidence.id);
    if (!existing) {
      throw new StateError(
        `Evidence ${evidence.id} not found`,
        "EVIDENCE_NOT_FOUND",
      );
    }

    // Update SMT
    const key = evidenceIdToKey(evidence.id);
    const value = serializeEvidence(evidence);
    this.evidenceSmt.set(key, value);

    // Update cache
    this.evidenceCache.set(evidence.id, evidence);

    // Update indexes if status changed
    if (existing.status !== evidence.status) {
      this.removeFromStatusIndex(existing.status, evidence.id);
      this.addToStatusIndex(evidence.status, evidence.id);
    }

    // Update archive stats
    const existingVerified = existing.archives.filter((a) => a.verified).length;
    const newVerified = evidence.archives.filter((a) => a.verified).length;
    this.verifiedArchiveCount += newVerified - existingVerified;
    this.archiveCount += evidence.archives.length - existing.archives.length;
  }

  /**
   * Add archive location to evidence
   */
  addArchive(evidenceId: string, archive: ArchiveLocation): void {
    const evidence = this.getEvidence(evidenceId);
    if (!evidence) {
      throw new StateError(
        `Evidence ${evidenceId} not found`,
        "EVIDENCE_NOT_FOUND",
      );
    }

    // Check if archive already exists
    const exists = evidence.archives.some(
      (a) =>
        a.provider === archive.provider && a.identifier === archive.identifier,
    );
    if (exists) {
      return; // Already have this archive
    }

    // Add archive
    const updated = {
      ...evidence,
      archives: [...evidence.archives, archive],
      status:
        evidence.status === "pending_archive"
          ? ("archived" as EvidenceStatus)
          : evidence.status,
    };

    this.updateEvidence(updated);
  }

  /**
   * Verify an archive
   */
  verifyArchive(
    evidenceId: string,
    archiveIndex: number,
    currentTime?: number,
  ): void {
    const evidence = this.getEvidence(evidenceId);
    if (!evidence) {
      throw new StateError(
        `Evidence ${evidenceId} not found`,
        "EVIDENCE_NOT_FOUND",
      );
    }

    if (archiveIndex >= evidence.archives.length) {
      throw new StateError(
        `Archive index ${archiveIndex} not found`,
        "ARCHIVE_NOT_FOUND",
      );
    }

    const archives = [...evidence.archives];
    archives[archiveIndex] = {
      ...archives[archiveIndex],
      verified: true,
      verifiedAt: currentTime ?? 0,
    };

    this.updateEvidence({ ...evidence, archives });
  }

  // ============ Version Chain Operations ============

  /**
   * Create new version of evidence
   */
  createVersion(
    originalId: string,
    newEvidence: Evidence,
    content: string,
    currentBlock: bigint,
  ): string {
    const original = this.getEvidence(originalId);
    if (!original) {
      throw new StateError(
        `Original evidence ${originalId} not found`,
        "EVIDENCE_NOT_FOUND",
      );
    }

    // Check version limits
    if (original.version.versionNumber >= 10) {
      throw new StateError(
        "Maximum 10 versions per evidence chain",
        "VERSION_LIMIT_EXCEEDED",
      );
    }

    // Set version info
    const versionedEvidence: Evidence = {
      ...newEvidence,
      version: {
        versionNumber: original.version.versionNumber + 1,
        previousVersionHash: original.contentHash,
        supersededBy: null,
        versionChain: [...original.version.versionChain, original.id],
        createdAt: Number(currentBlock),
        confidenceImpact: newEvidence.version.confidenceImpact || 0,
      },
    };

    // Mark original as superseded
    const updatedOriginal = {
      ...original,
      version: {
        ...original.version,
        supersededBy: newEvidence.id,
      },
    };
    this.updateEvidence(updatedOriginal);

    // Submit new version
    const result = this.submitEvidence(
      versionedEvidence,
      content,
      currentBlock,
    );

    // Update version chain index
    const chainRoot = original.version.versionChain[0] || originalId;
    const chain = this.versionChainIndex.get(chainRoot) || [chainRoot];
    chain.push(newEvidence.id);
    this.versionChainIndex.set(chainRoot, chain);

    return result.evidenceId;
  }

  /**
   * Get full version chain for evidence
   */
  getVersionChain(evidenceId: string): Evidence[] {
    const evidence = this.getEvidence(evidenceId);
    if (!evidence) {
      return [];
    }

    const chainRoot = evidence.version.versionChain[0] || evidenceId;
    const chain = this.versionChainIndex.get(chainRoot) || [evidenceId];

    return chain
      .map((id) => this.getEvidence(id))
      .filter((e): e is Evidence => e !== null);
  }

  // ============ Query Operations ============

  /**
   * Query evidence with filters
   */
  queryEvidence(query: EvidenceQuery): Evidence[] {
    let candidateIds: Set<string>;

    // Start with most selective index
    if (query.factId) {
      candidateIds = this.factIndex.get(query.factId) || new Set();
    } else if (query.submitter) {
      candidateIds = this.submitterIndex.get(query.submitter) || new Set();
    } else if (query.type) {
      candidateIds = this.typeIndex.get(query.type) || new Set();
    } else if (query.status) {
      candidateIds = this.statusIndex.get(query.status) || new Set();
    } else {
      // Get all evidence IDs
      candidateIds = new Set(this.evidenceCache.keys());
    }

    // Filter results
    const results: Evidence[] = [];
    for (const id of candidateIds) {
      const evidence = this.getEvidence(id);
      if (!evidence) continue;

      // Apply filters
      if (query.type && evidence.type !== query.type) continue;
      if (query.status && evidence.status !== query.status) continue;
      if (query.submitter && evidence.firstSubmitter !== query.submitter)
        continue;
      if (
        query.minReliability &&
        evidence.reliabilityScore < query.minReliability
      )
        continue;
      if (!query.includeExpired && evidence.expiry.isExpired) continue;

      results.push(evidence);
    }

    // Apply pagination
    const offset = query.offset || 0;
    const limit = query.limit || 100;
    return results.slice(offset, offset + limit);
  }

  /**
   * Get evidence for a specific fact
   */
  getEvidenceForFact(factId: string): Evidence[] {
    const evidenceIds = this.factIndex.get(factId) || new Set();
    return Array.from(evidenceIds)
      .map((id) => this.getEvidence(id))
      .filter((e): e is Evidence => e !== null);
  }

  // ============ Expiry Processing ============

  /**
   * Process expirations for all evidence
   */
  processExpirations(currentBlock: bigint): {
    processed: number;
    newlyExpired: string[];
  } {
    const allEvidence = Array.from(this.evidenceCache.values()).filter(
      (e) => !e.expiry.isExpired,
    );

    const results = processExpirations(allEvidence, currentBlock);

    const newlyExpired: string[] = [];
    for (const result of results) {
      if (result.isNewlyExpired) {
        const evidence = this.getEvidence(result.evidenceId);
        if (evidence) {
          const updated = {
            ...evidence,
            expiry: {
              ...evidence.expiry,
              isExpired: true,
              expiredAt: Number(currentBlock),
              freshnessScore: 0,
            },
            status: "expired" as EvidenceStatus,
          };
          this.updateEvidence(updated);
          newlyExpired.push(result.evidenceId);
        }
      }
    }

    return {
      processed: results.length,
      newlyExpired,
    };
  }

  /**
   * Calculate effective confidence with freshness
   */
  calculateEffectiveConfidence(
    evidenceId: string,
    currentBlock: bigint,
  ): number {
    const evidence = this.getEvidence(evidenceId);
    if (!evidence) {
      return 0;
    }

    // Base confidence from reliability
    let confidence = evidence.reliabilityScore;

    // Apply freshness
    const freshness = calculateFreshness(evidence, currentBlock);
    confidence *= Math.sqrt(freshness); // Gentler penalty

    // Apply archive bonus
    const archiveBonus = calculateArchiveBonus(evidence.archives);
    confidence = Math.min(1, Math.max(0, confidence + archiveBonus));

    return confidence;
  }

  // ============ Chainlink Operations ============

  /**
   * Register a Chainlink feed
   */
  registerChainlinkFeed(registration: ChainlinkFeedRegistration): void {
    this.chainlinkRegistry.registerFeed(registration);
  }

  /**
   * Get Chainlink feed by address
   */
  getChainlinkFeed(
    feedAddress: string,
    chainId: number,
  ): ChainlinkFeedRegistration | undefined {
    return this.chainlinkRegistry.getFeed(chainId, feedAddress);
  }

  /**
   * Get feeds by type
   */
  getChainlinkFeedsByType(
    feedType: ChainlinkFeedType,
  ): ChainlinkFeedRegistration[] {
    return this.chainlinkRegistry.getFeedsByType(feedType);
  }

  // ============ Archive Queue Operations ============

  /**
   * Get next archive request to process
   */
  getNextArchiveRequest(): ArchiveRequest | undefined {
    return this.archiveQueue.dequeue();
  }

  /**
   * Add archive request to queue
   */
  queueArchiveRequest(request: ArchiveRequest): void {
    this.archiveQueue.enqueue(request);
  }

  /**
   * Get archive queue statistics
   */
  getArchiveQueueStats(): ReturnType<ArchiveQueue["getStats"]> {
    return this.archiveQueue.getStats();
  }

  // ============ Statistics ============

  /**
   * Get comprehensive statistics
   */
  getStatistics(): EvidenceStatistics {
    const byType: Record<EvidenceType, number> = {} as Record<
      EvidenceType,
      number
    >;
    const byStatus: Record<EvidenceStatus, number> = {} as Record<
      EvidenceStatus,
      number
    >;

    for (const [type, ids] of this.typeIndex) {
      byType[type] = ids.size;
    }

    for (const [status, ids] of this.statusIndex) {
      byStatus[status] = ids.size;
    }

    const queueStats = this.archiveQueue.getStats();

    return {
      totalEvidence: this.evidenceCount,
      byType,
      byStatus,
      totalArchives: this.archiveCount,
      verifiedArchives: this.verifiedArchiveCount,
      pendingArchiveRequests: queueStats.pending,
      expiringEvidence: Array.from(this.evidenceCache.values()).filter(
        (e) => !e.expiry.isExpired && e.expiry.expiryType !== "eternal",
      ).length,
      chainlinkFeeds: this.chainlinkRegistry.getStats().totalFeeds,
    };
  }

  // ============ Merkle Proofs ============

  /**
   * Generate proof that evidence exists
   * Note: Full proof generation pending InMemorySMT enhancement
   */
  getEvidenceProof(
    evidenceId: string,
  ): { exists: boolean; root: MerkleRoot } | null {
    const evidence = this.getEvidence(evidenceId);
    if (!evidence) {
      return null;
    }
    return {
      exists: true,
      root: this.getEvidenceRoot(),
    };
  }

  /**
   * Verify evidence exists at given root
   * Note: Full proof verification pending InMemorySMT enhancement
   */
  verifyEvidenceExists(evidenceId: string, expectedRoot: MerkleRoot): boolean {
    const evidence = this.getEvidence(evidenceId);
    if (!evidence) {
      return false;
    }
    return this.getEvidenceRoot() === expectedRoot;
  }

  // ============ Index Helpers ============

  private addToTypeIndex(type: EvidenceType, id: string): void {
    if (!this.typeIndex.has(type)) {
      this.typeIndex.set(type, new Set());
    }
    this.typeIndex.get(type)!.add(id);
  }

  private addToStatusIndex(status: EvidenceStatus, id: string): void {
    if (!this.statusIndex.has(status)) {
      this.statusIndex.set(status, new Set());
    }
    this.statusIndex.get(status)!.add(id);
  }

  private removeFromStatusIndex(status: EvidenceStatus, id: string): void {
    this.statusIndex.get(status)?.delete(id);
  }

  private addToFactIndex(factId: string, id: string): void {
    if (!this.factIndex.has(factId)) {
      this.factIndex.set(factId, new Set());
    }
    this.factIndex.get(factId)!.add(id);
  }

  private addToSubmitterIndex(submitter: DID, id: string): void {
    if (!this.submitterIndex.has(submitter)) {
      this.submitterIndex.set(submitter, new Set());
    }
    this.submitterIndex.get(submitter)!.add(id);
  }

  // ============ Persistence Export/Restore ============

  /**
   * Export all evidence for persistence
   */
  exportEvidence(): Evidence[] {
    return Array.from(this.evidenceCache.values());
  }

  /**
   * Restore evidence from persistence (bypasses deduplication and content hashing)
   */
  restoreEvidence(evidence: Evidence): void {
    // Store in SMT
    const key = evidenceIdToKey(evidence.id);
    const value = serializeEvidence(evidence);
    this.evidenceSmt.set(key, value);

    // Update cache
    this.evidenceCache.set(evidence.id, evidence);

    // Update indexes
    this.addToTypeIndex(evidence.type, evidence.id);
    this.addToStatusIndex(evidence.status, evidence.id);
    this.addToSubmitterIndex(evidence.firstSubmitter, evidence.id);
    for (const factId of evidence.relatedFactIds) {
      this.addToFactIndex(factId, evidence.id);
    }

    // Update stats
    this.evidenceCount++;
    this.archiveCount += evidence.archives.length;
    this.verifiedArchiveCount += evidence.archives.filter(
      (a) => a.verified,
    ).length;
  }

  // ============ Clone for Transaction Support ============

  /**
   * Create a shallow clone for transaction checkpointing
   * Note: This is a simplified clone for rollback support
   */
  clone(): EvidenceStateTree {
    const cloned = new EvidenceStateTree();

    // Clone SMTs (in-memory SMT has clone support)
    // Note: For full production use, implement deep clone
    // For now, copy the evidence cache
    for (const [id, evidence] of this.evidenceCache) {
      cloned.evidenceCache.set(id, { ...evidence });
    }

    // Copy indexes
    for (const [type, ids] of this.typeIndex) {
      cloned.typeIndex.set(type, new Set(ids));
    }
    for (const [status, ids] of this.statusIndex) {
      cloned.statusIndex.set(status, new Set(ids));
    }
    for (const [factId, ids] of this.factIndex) {
      cloned.factIndex.set(factId, new Set(ids));
    }
    for (const [submitter, ids] of this.submitterIndex) {
      cloned.submitterIndex.set(submitter, new Set(ids));
    }
    for (const [chainRoot, versions] of this.versionChainIndex) {
      cloned.versionChainIndex.set(chainRoot, [...versions]);
    }

    // Copy stats
    cloned.evidenceCount = this.evidenceCount;
    cloned.archiveCount = this.archiveCount;
    cloned.verifiedArchiveCount = this.verifiedArchiveCount;

    return cloned;
  }
}

// ============ Factory Functions ============

/**
 * Create empty evidence state tree
 */
export function createEvidenceTree(): EvidenceStateTree {
  return new EvidenceStateTree();
}
