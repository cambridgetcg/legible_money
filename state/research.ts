/**
 * Research State Tree
 *
 * Sparse Merkle Tree for research submissions and bounties.
 * This is the on-chain state for the Protocol Research Repository.
 *
 * Key: hash(submissionId/bountyId) - 32 bytes
 * Value: serialized ResearchState/BountyState
 */

import {
  InMemorySMT,
  type MerkleRoot,
  type MerkleProofData,
} from "../core/crypto/merkle.js";
import { sha512_256, sha512_256Hex } from "../core/crypto/hashing.js";
import type { DID } from "../core/agents/identity/did.js";
import type {
  ResearchState,
  BountyState,
  ResearchType,
  ResearchOutcome,
  ResearchStatus,
  BountyStatus,
  ResearchQuery,
  FactResearchSummary,
} from "../core/research/types.js";
import { StateError } from "./types.js";

// ============ Serialization ============

function serializeResearchState(research: ResearchState): Uint8Array {
  const json = JSON.stringify({
    id: research.id,
    submitter: research.submitter,
    researchType: research.researchType,
    targetFactId: research.targetFactId,
    targetDomain: research.targetDomain,
    outcome: research.outcome,
    confidenceContribution: research.confidenceContribution,
    rawDataCID: research.rawDataCID,
    status: research.status,
    submittedAt: research.submittedAt.toString(),
    reviewScore: research.reviewScore,
  });
  return new TextEncoder().encode(json);
}

function deserializeResearchState(bytes: Uint8Array): ResearchState {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    id: obj.id,
    submitter: obj.submitter as DID,
    researchType: obj.researchType as ResearchType,
    targetFactId: obj.targetFactId,
    targetDomain: obj.targetDomain,
    outcome: obj.outcome as ResearchOutcome,
    confidenceContribution: obj.confidenceContribution,
    rawDataCID: obj.rawDataCID,
    status: obj.status as ResearchStatus,
    submittedAt: BigInt(obj.submittedAt),
    reviewScore: obj.reviewScore,
  };
}

function serializeBountyState(bounty: BountyState): Uint8Array {
  const json = JSON.stringify({
    id: bounty.id,
    creator: bounty.creator,
    targetType: bounty.targetType,
    targetFactId: bounty.targetFactId,
    reward: bounty.reward.toString(),
    deadline: bounty.deadline.toString(),
    status: bounty.status,
    claimedBy: bounty.claimedBy,
    fulfilledBy: bounty.fulfilledBy,
  });
  return new TextEncoder().encode(json);
}

function deserializeBountyState(bytes: Uint8Array): BountyState {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);
  return {
    id: obj.id,
    creator: obj.creator as DID,
    targetType: obj.targetType,
    targetFactId: obj.targetFactId,
    reward: BigInt(obj.reward),
    deadline: BigInt(obj.deadline),
    status: obj.status as BountyStatus,
    claimedBy: obj.claimedBy as DID | undefined,
    fulfilledBy: obj.fulfilledBy as DID | undefined,
  };
}

function researchIdToKey(id: string): Uint8Array {
  return sha512_256(`research:${id}`);
}

function bountyIdToKey(id: string): Uint8Array {
  return sha512_256(`bounty:${id}`);
}

// ============ Research State Tree ============

export class ResearchStateTree {
  private researchSmt: InMemorySMT;
  private bountySmt: InMemorySMT;
  private researchCache: Map<string, ResearchState> = new Map();
  private bountyCache: Map<string, BountyState> = new Map();

  // Indexes for efficient queries
  private factResearchIndex: Map<string, Set<string>> = new Map(); // factId -> researchIds
  private domainResearchIndex: Map<string, Set<string>> = new Map(); // domain -> researchIds
  private submitterIndex: Map<string, Set<string>> = new Map(); // submitter -> researchIds
  private openBountyIndex: Set<string> = new Set(); // bountyIds with status 'open'

  // Statistics
  private researchCount: number = 0;
  private acceptedCount: number = 0;
  private bountyCount: number = 0;
  private totalResearchFunding: bigint = 0n;

  constructor() {
    this.researchSmt = new InMemorySMT();
    this.bountySmt = new InMemorySMT();
  }

  // ============ Roots ============

  getResearchRoot(): MerkleRoot {
    return this.researchSmt.getRoot();
  }

  getBountyRoot(): MerkleRoot {
    return this.bountySmt.getRoot();
  }

  /**
   * Combined research layer root
   */
  getRoot(): MerkleRoot {
    const combined = `${this.getResearchRoot()}:${this.getBountyRoot()}`;
    return sha512_256Hex(combined);
  }

  // ============ Research Operations ============

  /**
   * Get research by ID
   */
  getResearch(id: string): ResearchState | null {
    const cached = this.researchCache.get(id);
    if (cached) return cached;

    const key = researchIdToKey(id);
    const value = this.researchSmt.get(key);
    if (!value) return null;

    const research = deserializeResearchState(value);
    this.researchCache.set(id, research);
    return research;
  }

  /**
   * Check if research exists
   */
  hasResearch(id: string): boolean {
    return this.getResearch(id) !== null;
  }

  /**
   * Add new research submission
   */
  addResearch(research: ResearchState): MerkleRoot {
    if (this.hasResearch(research.id)) {
      throw new StateError("Research already exists", "INVALID_TRANSITION", {
        id: research.id,
      });
    }

    const key = researchIdToKey(research.id);
    const value = serializeResearchState(research);
    const newRoot = this.researchSmt.set(key, value);

    this.researchCache.set(research.id, research);
    this.researchCount++;

    if (research.status === "accepted") {
      this.acceptedCount++;
    }

    // Update indexes
    if (research.targetFactId) {
      if (!this.factResearchIndex.has(research.targetFactId)) {
        this.factResearchIndex.set(research.targetFactId, new Set());
      }
      this.factResearchIndex.get(research.targetFactId)!.add(research.id);
    }

    if (!this.domainResearchIndex.has(research.targetDomain)) {
      this.domainResearchIndex.set(research.targetDomain, new Set());
    }
    this.domainResearchIndex.get(research.targetDomain)!.add(research.id);

    if (!this.submitterIndex.has(research.submitter)) {
      this.submitterIndex.set(research.submitter, new Set());
    }
    this.submitterIndex.get(research.submitter)!.add(research.id);

    return newRoot;
  }

  /**
   * Update research state
   */
  updateResearch(id: string, update: Partial<ResearchState>): MerkleRoot {
    const existing = this.getResearch(id);
    if (!existing) {
      throw new StateError("Research not found", "RESEARCH_NOT_FOUND", { id });
    }

    const wasAccepted = existing.status === "accepted";
    const updated: ResearchState = {
      ...existing,
      ...update,
    };
    const isAccepted = updated.status === "accepted";

    const key = researchIdToKey(id);
    const value = serializeResearchState(updated);
    const newRoot = this.researchSmt.set(key, value);

    this.researchCache.set(id, updated);

    // Update accepted count
    if (wasAccepted && !isAccepted) {
      this.acceptedCount--;
    } else if (!wasAccepted && isAccepted) {
      this.acceptedCount++;
    }

    return newRoot;
  }

  /**
   * Get all research for a fact
   */
  getResearchForFact(factId: string): ResearchState[] {
    const ids = this.factResearchIndex.get(factId);
    if (!ids) return [];

    const results: ResearchState[] = [];
    for (const id of ids) {
      const research = this.getResearch(id);
      if (research) {
        results.push(research);
      }
    }

    return results.sort((a, b) => Number(b.submittedAt - a.submittedAt));
  }

  /**
   * Get research summary for a fact
   */
  getFactResearchSummary(factId: string): FactResearchSummary {
    const research = this.getResearchForFact(factId);

    const replications = research.filter(
      (r) => r.researchType === "replication" && r.status === "accepted",
    );
    const successfulReplications = replications.filter(
      (r) => r.outcome === "confirms",
    );
    const failedReplications = replications.filter(
      (r) => r.outcome === "refutes",
    );

    const fraudInvestigations = research.filter(
      (r) =>
        r.researchType === "fraud_investigation" && r.status === "accepted",
    );
    const fraudDetected = fraudInvestigations.some(
      (r) => r.outcome === "fraud_detected",
    );

    // Calculate net confidence modifier
    let netModifier = 0;

    // Replication bonuses
    if (successfulReplications.length > 0) {
      netModifier += 0.15; // First replication
      const additional = Math.min(successfulReplications.length - 1, 5);
      netModifier += additional * 0.05;
    }

    // Failed replications penalty
    if (failedReplications.length > 0) {
      netModifier -= failedReplications.length * 0.1;
    }

    // Fraud investigation bonuses/penalties
    if (fraudDetected) {
      netModifier = -1.0; // Immediate invalidation
    } else if (fraudInvestigations.length > 0) {
      netModifier += 0.1; // Clean investigation bonus
    }

    return {
      factId,
      replicationAttempts: replications.length,
      successfulReplications: successfulReplications.length,
      failedReplications: failedReplications.length,
      fraudInvestigations: fraudInvestigations.length,
      fraudDetected,
      netConfidenceModifier: Math.max(-1, Math.min(0.4, netModifier)),
      latestResearch: research[0],
    };
  }

  /**
   * Query research
   */
  queryResearch(query: ResearchQuery): ResearchState[] {
    let results: ResearchState[] = [];

    // Start with appropriate index
    if (query.factId) {
      results = this.getResearchForFact(query.factId);
    } else if (query.domain) {
      const ids = this.domainResearchIndex.get(query.domain);
      if (ids) {
        for (const id of ids) {
          const research = this.getResearch(id);
          if (research) results.push(research);
        }
      }
    } else if (query.submitter) {
      const ids = this.submitterIndex.get(query.submitter);
      if (ids) {
        for (const id of ids) {
          const research = this.getResearch(id);
          if (research) results.push(research);
        }
      }
    } else {
      // Full scan (expensive)
      results = Array.from(this.researchCache.values());
    }

    // Apply filters
    if (query.researchType) {
      results = results.filter((r) => r.researchType === query.researchType);
    }
    if (query.outcome) {
      results = results.filter((r) => r.outcome === query.outcome);
    }
    if (query.status) {
      results = results.filter((r) => r.status === query.status);
    }
    if (query.minConfidenceContribution !== undefined) {
      results = results.filter(
        (r) => r.confidenceContribution >= query.minConfidenceContribution!,
      );
    }

    // Sort by submission time (newest first)
    results.sort((a, b) => Number(b.submittedAt - a.submittedAt));

    // Apply pagination
    const offset = query.offset ?? 0;
    const limit = query.limit ?? 100;
    return results.slice(offset, offset + limit);
  }

  // ============ Bounty Operations ============

  /**
   * Get bounty by ID
   */
  getBounty(id: string): BountyState | null {
    const cached = this.bountyCache.get(id);
    if (cached) return cached;

    const key = bountyIdToKey(id);
    const value = this.bountySmt.get(key);
    if (!value) return null;

    const bounty = deserializeBountyState(value);
    this.bountyCache.set(id, bounty);
    return bounty;
  }

  /**
   * Check if bounty exists
   */
  hasBounty(id: string): boolean {
    return this.getBounty(id) !== null;
  }

  /**
   * Add new bounty
   */
  addBounty(bounty: BountyState): MerkleRoot {
    if (this.hasBounty(bounty.id)) {
      throw new StateError("Bounty already exists", "INVALID_TRANSITION", {
        id: bounty.id,
      });
    }

    const key = bountyIdToKey(bounty.id);
    const value = serializeBountyState(bounty);
    const newRoot = this.bountySmt.set(key, value);

    this.bountyCache.set(bounty.id, bounty);
    this.bountyCount++;
    this.totalResearchFunding += bounty.reward;

    if (bounty.status === "open") {
      this.openBountyIndex.add(bounty.id);
    }

    return newRoot;
  }

  /**
   * Update bounty state
   */
  updateBounty(id: string, update: Partial<BountyState>): MerkleRoot {
    const existing = this.getBounty(id);
    if (!existing) {
      throw new StateError("Bounty not found", "BOUNTY_NOT_FOUND", { id });
    }

    const wasOpen = existing.status === "open";
    const updated: BountyState = {
      ...existing,
      ...update,
    };
    const isOpen = updated.status === "open";

    const key = bountyIdToKey(id);
    const value = serializeBountyState(updated);
    const newRoot = this.bountySmt.set(key, value);

    this.bountyCache.set(id, updated);

    // Update open index
    if (wasOpen && !isOpen) {
      this.openBountyIndex.delete(id);
    } else if (!wasOpen && isOpen) {
      this.openBountyIndex.add(id);
    }

    return newRoot;
  }

  /**
   * Claim a bounty
   */
  claimBounty(id: string, claimer: DID): MerkleRoot {
    const bounty = this.getBounty(id);
    if (!bounty) {
      throw new StateError("Bounty not found", "BOUNTY_NOT_FOUND", { id });
    }

    if (bounty.status !== "open") {
      throw new StateError("Bounty is not open", "INVALID_TRANSITION", {
        id,
        status: bounty.status,
      });
    }

    return this.updateBounty(id, {
      status: "claimed",
      claimedBy: claimer,
    });
  }

  /**
   * Fulfill a bounty
   */
  fulfillBounty(id: string, fulfiller: DID): MerkleRoot {
    const bounty = this.getBounty(id);
    if (!bounty) {
      throw new StateError("Bounty not found", "BOUNTY_NOT_FOUND", { id });
    }

    if (bounty.status !== "claimed") {
      throw new StateError("Bounty is not claimed", "INVALID_TRANSITION", {
        id,
        status: bounty.status,
      });
    }

    if (bounty.claimedBy !== fulfiller) {
      throw new StateError("Only claimer can fulfill", "UNAUTHORIZED", {
        id,
        fulfiller,
      });
    }

    return this.updateBounty(id, {
      status: "fulfilled",
      fulfilledBy: fulfiller,
    });
  }

  /**
   * Get open bounties
   */
  getOpenBounties(): BountyState[] {
    const results: BountyState[] = [];
    for (const id of this.openBountyIndex) {
      const bounty = this.getBounty(id);
      if (bounty && bounty.status === "open") {
        results.push(bounty);
      }
    }
    return results.sort((a, b) => Number(b.reward - a.reward)); // Highest reward first
  }

  /**
   * Get bounties for a fact
   */
  getBountiesForFact(factId: string): BountyState[] {
    const results: BountyState[] = [];
    for (const bounty of this.bountyCache.values()) {
      if (bounty.targetFactId === factId) {
        results.push(bounty);
      }
    }
    return results;
  }

  /**
   * Expire overdue bounties
   */
  expireOverdueBounties(currentBlock: bigint): number {
    let expired = 0;
    for (const id of this.openBountyIndex) {
      const bounty = this.getBounty(id);
      if (bounty && bounty.deadline < currentBlock) {
        this.updateBounty(id, { status: "expired" });
        expired++;
      }
    }
    return expired;
  }

  // ============ Statistics ============

  getResearchCount(): number {
    return this.researchCount;
  }

  getAcceptedResearchCount(): number {
    return this.acceptedCount;
  }

  getBountyCount(): number {
    return this.bountyCount;
  }

  getTotalResearchFunding(): bigint {
    return this.totalResearchFunding;
  }

  getOpenBountyCount(): number {
    return this.openBountyIndex.size;
  }

  // ============ Proofs ============

  /**
   * Generate proof for research existence
   */
  proveResearch(id: string): MerkleProofData {
    const key = researchIdToKey(id);
    return this.researchSmt.prove(key);
  }

  /**
   * Generate proof for bounty existence
   */
  proveBounty(id: string): MerkleProofData {
    const key = bountyIdToKey(id);
    return this.bountySmt.prove(key);
  }

  // ============ Utilities ============

  /**
   * Clone for speculative execution
   */
  clone(): ResearchStateTree {
    const cloned = new ResearchStateTree();

    for (const [id, research] of this.researchCache) {
      cloned.researchCache.set(id, { ...research });
      const key = researchIdToKey(id);
      cloned.researchSmt.set(key, serializeResearchState(research));
    }

    for (const [id, bounty] of this.bountyCache) {
      cloned.bountyCache.set(id, { ...bounty });
      const key = bountyIdToKey(id);
      cloned.bountySmt.set(key, serializeBountyState(bounty));
    }

    for (const [factId, ids] of this.factResearchIndex) {
      cloned.factResearchIndex.set(factId, new Set(ids));
    }

    for (const [domain, ids] of this.domainResearchIndex) {
      cloned.domainResearchIndex.set(domain, new Set(ids));
    }

    for (const [submitter, ids] of this.submitterIndex) {
      cloned.submitterIndex.set(submitter, new Set(ids));
    }

    cloned.openBountyIndex = new Set(this.openBountyIndex);
    cloned.researchCount = this.researchCount;
    cloned.acceptedCount = this.acceptedCount;
    cloned.bountyCount = this.bountyCount;
    cloned.totalResearchFunding = this.totalResearchFunding;

    return cloned;
  }
}

// ============ Factory Functions ============

/**
 * Generate research submission ID
 */
export function generateResearchId(
  submitter: string,
  targetDomain: string,
  timestamp: number,
): string {
  return sha512_256Hex(`research:${submitter}:${targetDomain}:${timestamp}`);
}

/**
 * Generate bounty ID
 */
export function generateBountyId(
  creator: string,
  targetType: string,
  timestamp: number,
): string {
  return sha512_256Hex(`bounty:${creator}:${targetType}:${timestamp}`);
}
