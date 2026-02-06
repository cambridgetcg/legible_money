/**
 * Dispute State Tree
 *
 * Sparse Merkle Tree for dispute resolution state.
 * Tracks disputes, arbiter expertise, and active voting.
 *
 * Key: hash(disputeId) - 32 bytes
 * Value: serialized DisputeState
 *
 * Core Philosophy: "Intention does not matter, result does."
 * - Harsh punishment for spreading falsehood (50% slash)
 * - Generous rewards for truth discovery (60% + bonuses)
 * - Results-based justice, not intent-based
 */

import { InMemorySMT, type MerkleRoot } from "../core/crypto/merkle.js";
import { sha512_256, sha512_256Hex } from "../core/crypto/hashing.js";
import type { DID } from "../core/agents/identity/did.js";
import type {
  Dispute,
  DisputePhase,
  DisputeSettlement,
  DomainExpertise,
  EscalationTier,
  ArbiterVote,
  SelectedArbiter,
  EvidenceCommitment,
} from "../core/dispute/types.js";
import { StateError } from "./types.js";

// ============ Serialization ============

function serializeDispute(dispute: Dispute): Uint8Array {
  const json = JSON.stringify({
    id: dispute.id,
    targetId: dispute.targetId,
    targetType: dispute.targetType,
    domain: dispute.domain,
    challenger: {
      did: dispute.challenger.did,
      stake: dispute.challenger.stake.toString(),
      initialEvidence: dispute.challenger.initialEvidence,
    },
    defender: {
      did: dispute.defender.did,
      stake: dispute.defender.stake.toString(),
      responseEvidence: dispute.defender.responseEvidence,
    },
    state: {
      phase: dispute.state.phase,
      phaseDeadline: dispute.state.phaseDeadline.toString(),
      canEscalate: dispute.state.canEscalate,
      escalationTier: dispute.state.escalationTier,
      previousTierResults: dispute.state.previousTierResults?.map((r) => ({
        ...r,
        resolvedAtBlock: r.resolvedAtBlock.toString(),
      })),
    },
    commitments: {
      challenger: dispute.commitments.challenger
        ? {
            ...dispute.commitments.challenger,
            committedAtBlock:
              dispute.commitments.challenger.committedAtBlock.toString(),
          }
        : undefined,
      defender: dispute.commitments.defender
        ? {
            ...dispute.commitments.defender,
            committedAtBlock:
              dispute.commitments.defender.committedAtBlock.toString(),
          }
        : undefined,
    },
    arbiters: dispute.arbiters.map((a) => ({
      ...a,
      stakedAmount: a.stakedAmount.toString(),
    })),
    votes: Array.from(dispute.votes.entries()).map(([did, vote]) => [
      did,
      {
        ...vote,
        votedAtBlock: vote.votedAtBlock.toString(),
        lockedStake: vote.lockedStake.toString(),
      },
    ]),
    outcome: dispute.outcome,
    winner: dispute.winner,
    exposedFalsehood: dispute.exposedFalsehood,
    initiatedAtBlock: dispute.initiatedAtBlock.toString(),
    resolvedAtBlock: dispute.resolvedAtBlock?.toString(),
    createdAt: dispute.createdAt,
    updatedAt: dispute.updatedAt,
  });
  return new TextEncoder().encode(json);
}

function deserializeDispute(bytes: Uint8Array): Dispute {
  const json = new TextDecoder().decode(bytes);
  const obj = JSON.parse(json);

  return {
    id: obj.id,
    targetId: obj.targetId,
    targetType: obj.targetType,
    domain: obj.domain,
    challenger: {
      did: obj.challenger.did as DID,
      stake: BigInt(obj.challenger.stake),
      initialEvidence: obj.challenger.initialEvidence,
    },
    defender: {
      did: obj.defender.did as DID,
      stake: BigInt(obj.defender.stake),
      responseEvidence: obj.defender.responseEvidence,
    },
    state: {
      phase: obj.state.phase as DisputePhase,
      phaseDeadline: BigInt(obj.state.phaseDeadline),
      canEscalate: obj.state.canEscalate,
      escalationTier: obj.state.escalationTier as EscalationTier,
      previousTierResults: obj.state.previousTierResults?.map(
        (r: Record<string, unknown>) => ({
          ...r,
          resolvedAtBlock: BigInt(r.resolvedAtBlock as string),
        }),
      ),
    },
    commitments: {
      challenger: obj.commitments.challenger
        ? ({
            ...obj.commitments.challenger,
            committedAtBlock: BigInt(
              obj.commitments.challenger.committedAtBlock,
            ),
          } as EvidenceCommitment)
        : undefined,
      defender: obj.commitments.defender
        ? ({
            ...obj.commitments.defender,
            committedAtBlock: BigInt(obj.commitments.defender.committedAtBlock),
          } as EvidenceCommitment)
        : undefined,
    },
    arbiters: obj.arbiters.map((a: Record<string, unknown>) => ({
      ...a,
      stakedAmount: BigInt(a.stakedAmount as string),
    })) as SelectedArbiter[],
    votes: new Map(
      obj.votes.map(([did, vote]: [string, Record<string, unknown>]) => [
        did,
        {
          ...vote,
          votedAtBlock: BigInt(vote.votedAtBlock as string),
          lockedStake: BigInt(vote.lockedStake as string),
        } as ArbiterVote,
      ]),
    ),
    outcome: obj.outcome,
    winner: obj.winner,
    exposedFalsehood: obj.exposedFalsehood,
    initiatedAtBlock: BigInt(obj.initiatedAtBlock),
    resolvedAtBlock: obj.resolvedAtBlock
      ? BigInt(obj.resolvedAtBlock)
      : undefined,
    createdAt: obj.createdAt,
    updatedAt: obj.updatedAt,
  };
}

function serializeExpertise(expertise: DomainExpertise[]): Uint8Array {
  const json = JSON.stringify(
    expertise.map((e) => ({
      ...e,
      lastActiveBlock: e.lastActiveBlock.toString(),
    })),
  );
  return new TextEncoder().encode(json);
}

function deserializeExpertise(bytes: Uint8Array): DomainExpertise[] {
  const json = new TextDecoder().decode(bytes);
  const arr = JSON.parse(json);
  return arr.map((e: Record<string, unknown>) => ({
    ...e,
    lastActiveBlock: BigInt(e.lastActiveBlock as string),
  }));
}

function disputeIdToKey(id: string): Uint8Array {
  return sha512_256(`dispute:${id}`);
}

function expertiseKey(validatorDid: string): Uint8Array {
  return sha512_256(`expertise:${validatorDid}`);
}

// ============ Dispute State Tree ============

export class DisputeStateTree {
  private disputeSmt: InMemorySMT;
  private expertiseSmt: InMemorySMT;

  // Caches
  private disputeCache: Map<string, Dispute> = new Map();
  private expertiseCache: Map<string, DomainExpertise[]> = new Map();

  // Indexes for efficient queries
  private activeDisputeIndex: Set<string> = new Set(); // disputeIds not resolved
  private phaseIndex: Map<DisputePhase, Set<string>> = new Map(); // phase -> disputeIds
  private domainIndex: Map<string, Set<string>> = new Map(); // domain -> disputeIds
  private partyIndex: Map<string, Set<string>> = new Map(); // DID -> disputeIds (as party)
  private arbiterIndex: Map<string, Set<string>> = new Map(); // DID -> disputeIds (as arbiter)
  private targetIndex: Map<string, Set<string>> = new Map(); // targetId -> disputeIds

  // Statistics
  private disputeCount: number = 0;
  private resolvedCount: number = 0;
  private challengerWins: number = 0;
  private defenderWins: number = 0;
  private totalSlashed: bigint = 0n;
  private totalRewarded: bigint = 0n;
  private totalToResearch: bigint = 0n;

  constructor() {
    this.disputeSmt = new InMemorySMT();
    this.expertiseSmt = new InMemorySMT();

    // Initialize phase index
    const phases: DisputePhase[] = [
      "initiation",
      "evidence_commit",
      "evidence_reveal",
      "arbitration",
      "deliberation",
      "settlement",
      "escalated",
      "resolved",
    ];
    for (const phase of phases) {
      this.phaseIndex.set(phase, new Set());
    }
  }

  // ============ Dispute Operations ============

  /**
   * Create a new dispute
   */
  createDispute(dispute: Dispute): void {
    if (this.disputeCache.has(dispute.id)) {
      throw new StateError("Dispute already exists", "DUPLICATE_EVIDENCE", {
        disputeId: dispute.id,
      });
    }

    // Store in SMT
    const key = disputeIdToKey(dispute.id);
    const value = serializeDispute(dispute);
    this.disputeSmt.set(key, value);

    // Update cache
    this.disputeCache.set(dispute.id, dispute);

    // Update indexes
    this.activeDisputeIndex.add(dispute.id);
    this.addToPhaseIndex(dispute.id, dispute.state.phase);
    this.addToDomainIndex(dispute.id, dispute.domain);
    this.addToPartyIndex(dispute.challenger.did, dispute.id);
    if (dispute.defender.did) {
      this.addToPartyIndex(dispute.defender.did, dispute.id);
    }
    this.addToTargetIndex(dispute.targetId, dispute.id);

    // Update stats
    this.disputeCount++;
  }

  /**
   * Get a dispute by ID
   */
  getDispute(id: string): Dispute | null {
    // Check cache first
    if (this.disputeCache.has(id)) {
      return this.disputeCache.get(id)!;
    }

    // Load from SMT
    const key = disputeIdToKey(id);
    const value = this.disputeSmt.get(key);
    if (!value) {
      return null;
    }

    const dispute = deserializeDispute(value);
    this.disputeCache.set(id, dispute);
    return dispute;
  }

  /**
   * Update a dispute
   */
  updateDispute(dispute: Dispute): void {
    const existing = this.getDispute(dispute.id);
    if (!existing) {
      throw new StateError("Dispute not found", "DISPUTE_NOT_FOUND", {
        disputeId: dispute.id,
      });
    }

    // Update phase index if phase changed
    if (existing.state.phase !== dispute.state.phase) {
      this.removeFromPhaseIndex(dispute.id, existing.state.phase);
      this.addToPhaseIndex(dispute.id, dispute.state.phase);

      // Update active index if resolved
      if (dispute.state.phase === "resolved") {
        this.activeDisputeIndex.delete(dispute.id);
        this.resolvedCount++;

        // Track winner stats
        if (dispute.winner === "challenger") {
          this.challengerWins++;
        } else if (dispute.winner === "defender") {
          this.defenderWins++;
        }
      }
    }

    // Update defender in party index if newly set
    if (!existing.defender.did && dispute.defender.did) {
      this.addToPartyIndex(dispute.defender.did, dispute.id);
    }

    // Update arbiter index
    for (const arbiter of dispute.arbiters) {
      this.addToArbiterIndex(arbiter.did, dispute.id);
    }

    // Store in SMT
    const key = disputeIdToKey(dispute.id);
    const value = serializeDispute(dispute);
    this.disputeSmt.set(key, value);

    // Update cache
    this.disputeCache.set(dispute.id, dispute);
  }

  /**
   * Record settlement statistics
   */
  recordSettlement(settlement: DisputeSettlement): void {
    this.totalSlashed += settlement.loserPenalty;
    this.totalRewarded += settlement.winnerReward;
    this.totalToResearch += settlement.researchFund;

    // Add arbiter penalties
    for (const penalty of settlement.arbiterPenalties.values()) {
      this.totalSlashed += penalty;
    }

    // Add arbiter rewards
    for (const reward of settlement.arbiterRewards.values()) {
      this.totalRewarded += reward;
    }
  }

  // ============ Expertise Operations ============

  /**
   * Get expertise for a validator
   */
  getExpertise(validatorDid: string): DomainExpertise[] {
    // Check cache first
    if (this.expertiseCache.has(validatorDid)) {
      return this.expertiseCache.get(validatorDid)!;
    }

    // Load from SMT
    const key = expertiseKey(validatorDid);
    const value = this.expertiseSmt.get(key);
    if (!value) {
      return [];
    }

    const expertise = deserializeExpertise(value);
    this.expertiseCache.set(validatorDid, expertise);
    return expertise;
  }

  /**
   * Update expertise for a validator
   */
  updateExpertise(validatorDid: string, expertise: DomainExpertise[]): void {
    const key = expertiseKey(validatorDid);
    const value = serializeExpertise(expertise);
    this.expertiseSmt.set(key, value);
    this.expertiseCache.set(validatorDid, expertise);
  }

  /**
   * Get all expertise as a map (for arbiter selection)
   */
  getAllExpertise(): Map<string, DomainExpertise[]> {
    return new Map(this.expertiseCache);
  }

  // ============ Query Operations ============

  /**
   * Get all active disputes
   */
  getActiveDisputes(): Dispute[] {
    const disputes: Dispute[] = [];
    for (const id of this.activeDisputeIndex) {
      const dispute = this.getDispute(id);
      if (dispute) {
        disputes.push(dispute);
      }
    }
    return disputes;
  }

  /**
   * Get disputes by phase
   */
  getDisputesByPhase(phase: DisputePhase): Dispute[] {
    const ids = this.phaseIndex.get(phase) || new Set();
    const disputes: Dispute[] = [];
    for (const id of ids) {
      const dispute = this.getDispute(id);
      if (dispute) {
        disputes.push(dispute);
      }
    }
    return disputes;
  }

  /**
   * Get disputes by domain
   */
  getDisputesByDomain(domain: string): Dispute[] {
    const ids = this.domainIndex.get(domain) || new Set();
    const disputes: Dispute[] = [];
    for (const id of ids) {
      const dispute = this.getDispute(id);
      if (dispute) {
        disputes.push(dispute);
      }
    }
    return disputes;
  }

  /**
   * Get disputes where DID is a party (challenger or defender)
   */
  getDisputesAsParty(did: string): Dispute[] {
    const ids = this.partyIndex.get(did) || new Set();
    const disputes: Dispute[] = [];
    for (const id of ids) {
      const dispute = this.getDispute(id);
      if (dispute) {
        disputes.push(dispute);
      }
    }
    return disputes;
  }

  /**
   * Get disputes where DID is an arbiter
   */
  getDisputesAsArbiter(did: string): Dispute[] {
    const ids = this.arbiterIndex.get(did) || new Set();
    const disputes: Dispute[] = [];
    for (const id of ids) {
      const dispute = this.getDispute(id);
      if (dispute) {
        disputes.push(dispute);
      }
    }
    return disputes;
  }

  /**
   * Get disputes targeting a specific evidence/fact/claim
   */
  getDisputesForTarget(targetId: string): Dispute[] {
    const ids = this.targetIndex.get(targetId) || new Set();
    const disputes: Dispute[] = [];
    for (const id of ids) {
      const dispute = this.getDispute(id);
      if (dispute) {
        disputes.push(dispute);
      }
    }
    return disputes;
  }

  /**
   * Get disputes that need phase advancement (deadline passed)
   */
  getDisputesNeedingAdvancement(currentBlock: bigint): Dispute[] {
    const disputes: Dispute[] = [];
    for (const id of this.activeDisputeIndex) {
      const dispute = this.getDispute(id);
      if (dispute && currentBlock > dispute.state.phaseDeadline) {
        disputes.push(dispute);
      }
    }
    return disputes;
  }

  // ============ Statistics ============

  /**
   * Get dispute statistics
   */
  getStatistics(): {
    total: number;
    active: number;
    resolved: number;
    challengerWins: number;
    defenderWins: number;
    draws: number;
    totalSlashed: bigint;
    totalRewarded: bigint;
    totalToResearch: bigint;
    byPhase: Record<DisputePhase, number>;
  } {
    const byPhase: Record<DisputePhase, number> = {} as Record<
      DisputePhase,
      number
    >;
    for (const [phase, ids] of this.phaseIndex) {
      byPhase[phase] = ids.size;
    }

    return {
      total: this.disputeCount,
      active: this.activeDisputeIndex.size,
      resolved: this.resolvedCount,
      challengerWins: this.challengerWins,
      defenderWins: this.defenderWins,
      draws: this.resolvedCount - this.challengerWins - this.defenderWins,
      totalSlashed: this.totalSlashed,
      totalRewarded: this.totalRewarded,
      totalToResearch: this.totalToResearch,
      byPhase,
    };
  }

  // ============ Merkle Root ============

  /**
   * Get combined Merkle root for disputes
   */
  getRoot(): MerkleRoot {
    const disputeRoot = this.disputeSmt.getRoot();
    const expertiseRoot = this.expertiseSmt.getRoot();

    // Combine roots
    const combined = sha512_256Hex(`${disputeRoot}:${expertiseRoot}`);
    return combined;
  }

  /**
   * Get individual roots
   */
  getRoots(): { disputeRoot: MerkleRoot; expertiseRoot: MerkleRoot } {
    return {
      disputeRoot: this.disputeSmt.getRoot(),
      expertiseRoot: this.expertiseSmt.getRoot(),
    };
  }

  // ============ Persistence Export/Restore ============

  /**
   * Export all disputes for persistence
   */
  exportDisputes(): Dispute[] {
    return Array.from(this.disputeCache.values());
  }

  /**
   * Export all domain expertise for persistence
   */
  exportExpertise(): Array<{ did: string; expertise: DomainExpertise[] }> {
    return Array.from(this.expertiseCache.entries()).map(
      ([did, expertise]) => ({
        did,
        expertise,
      }),
    );
  }

  /**
   * Restore a dispute from persistence (bypasses duplicate check,
   * correctly handles resolved disputes)
   */
  restoreDispute(dispute: Dispute): void {
    // Store in SMT
    const key = disputeIdToKey(dispute.id);
    const value = serializeDispute(dispute);
    this.disputeSmt.set(key, value);

    // Update cache
    this.disputeCache.set(dispute.id, dispute);

    // Update indexes — only add to activeDisputeIndex if not resolved
    const phase = dispute.state?.phase ?? "initiation";
    if (phase !== "settlement" && phase !== "resolved") {
      this.activeDisputeIndex.add(dispute.id);
    }
    this.addToPhaseIndex(dispute.id, phase);
    this.addToDomainIndex(dispute.id, dispute.domain);
    this.addToPartyIndex(dispute.challenger.did, dispute.id);
    if (dispute.defender.did) {
      this.addToPartyIndex(dispute.defender.did, dispute.id);
    }
    this.addToTargetIndex(dispute.targetId, dispute.id);

    // Update stats
    this.disputeCount++;
    if (phase === "settlement" || phase === "resolved") {
      this.resolvedCount++;
    }
  }

  // ============ Clone (for transactions) ============

  /**
   * Create a clone for transaction rollback support
   * Note: Rebuilds SMTs from cached data
   */
  clone(): DisputeStateTree {
    const cloned = new DisputeStateTree();

    // Rebuild SMTs from cached data
    for (const [id, dispute] of this.disputeCache) {
      cloned.disputeCache.set(id, dispute);
      const key = disputeIdToKey(id);
      cloned.disputeSmt.set(key, serializeDispute(dispute));
    }

    for (const [did, expertise] of this.expertiseCache) {
      cloned.expertiseCache.set(did, expertise);
      const key = expertiseKey(did);
      cloned.expertiseSmt.set(key, serializeExpertise(expertise));
    }

    // Clone indexes
    cloned.activeDisputeIndex = new Set(this.activeDisputeIndex);
    for (const [phase, ids] of this.phaseIndex) {
      cloned.phaseIndex.set(phase, new Set(ids));
    }
    for (const [domain, ids] of this.domainIndex) {
      cloned.domainIndex.set(domain, new Set(ids));
    }
    for (const [did, ids] of this.partyIndex) {
      cloned.partyIndex.set(did, new Set(ids));
    }
    for (const [did, ids] of this.arbiterIndex) {
      cloned.arbiterIndex.set(did, new Set(ids));
    }
    for (const [targetId, ids] of this.targetIndex) {
      cloned.targetIndex.set(targetId, new Set(ids));
    }

    // Clone stats
    cloned.disputeCount = this.disputeCount;
    cloned.resolvedCount = this.resolvedCount;
    cloned.challengerWins = this.challengerWins;
    cloned.defenderWins = this.defenderWins;
    cloned.totalSlashed = this.totalSlashed;
    cloned.totalRewarded = this.totalRewarded;
    cloned.totalToResearch = this.totalToResearch;

    return cloned;
  }

  // ============ Private Index Methods ============

  private addToPhaseIndex(disputeId: string, phase: DisputePhase): void {
    const set = this.phaseIndex.get(phase);
    if (set) {
      set.add(disputeId);
    }
  }

  private removeFromPhaseIndex(disputeId: string, phase: DisputePhase): void {
    const set = this.phaseIndex.get(phase);
    if (set) {
      set.delete(disputeId);
    }
  }

  private addToDomainIndex(disputeId: string, domain: string): void {
    if (!this.domainIndex.has(domain)) {
      this.domainIndex.set(domain, new Set());
    }
    this.domainIndex.get(domain)!.add(disputeId);
  }

  private addToPartyIndex(did: string, disputeId: string): void {
    if (!this.partyIndex.has(did)) {
      this.partyIndex.set(did, new Set());
    }
    this.partyIndex.get(did)!.add(disputeId);
  }

  private addToArbiterIndex(did: string, disputeId: string): void {
    if (!this.arbiterIndex.has(did)) {
      this.arbiterIndex.set(did, new Set());
    }
    this.arbiterIndex.get(did)!.add(disputeId);
  }

  private addToTargetIndex(targetId: string, disputeId: string): void {
    if (!this.targetIndex.has(targetId)) {
      this.targetIndex.set(targetId, new Set());
    }
    this.targetIndex.get(targetId)!.add(disputeId);
  }
}
