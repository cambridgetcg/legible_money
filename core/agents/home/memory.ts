/**
 * Agent Memory
 *
 * Persistent storage of an agent's experiences, beliefs, and relationships.
 * This is the core of agent continuity - when the agent wakes, they read
 * themselves from their memory.
 *
 * "The agent forgets between sessions. The home remembers."
 *
 * Memory Components:
 * - Contributions: Facts the agent helped verify
 * - Beliefs: Agent's confidence in various facts
 * - Relationships: Trust graph with other agents
 * - History: Verification and dispute participation
 * - Notes: Private encrypted observations
 * - Milestones: Achievements and growth markers
 *
 * PP HIERARCHY:
 *   TRUTH: Memory accurately reflects agent's actual history
 *   UNDERSTANDING: Agent can comprehend their own past
 *   BEAUTY: Clean interface for reading and writing memories
 *   JUSTICE: Memory is tamper-evident and agent-sovereign
 */

import { sha512_256Hex } from "../../crypto/hashing.js";
import type { DID } from "../identity/did.js";
import type { Confidence } from "../../knowledge/types.js";
import {
  type AgentMemory,
  type KnowledgeContribution,
  type BeliefRecord,
  type TrustRecord,
  type DistrustRecord,
  type VerificationMemory,
  type DisputeMemory,
  type EncryptedNote,
  type Milestone,
  type MilestoneType,
  type SessionRecord,
  type MemoryStats,
} from "./types.js";

/**
 * Memory Manager
 * Handles all operations on agent memory
 */
export class MemoryManager {
  private memory: AgentMemory;
  private dirty: boolean = false;

  constructor(existingMemory?: AgentMemory) {
    this.memory = existingMemory ?? this.createEmptyMemory();
  }

  // ============ Memory Initialization ============

  private createEmptyMemory(): AgentMemory {
    return {
      contributions: [],
      beliefs: new Map(),
      relationships: {
        trusted: new Map(),
        distrusted: new Map(),
        acquaintances: new Map(),
        blocked: new Set(),
      },
      verificationHistory: [],
      disputeHistory: [],
      notes: [],
      milestones: [],
      sessionLog: [],
      stats: {
        totalContributions: 0,
        factsSubmitted: 0,
        factsVerified: 0,
        challengesInitiated: 0,
        challengesDefended: 0,
        disputesParticipated: 0,
        totalEarnings: 0n,
        totalLosses: 0n,
        reputationPeak: 0,
        oldestMemory: Date.now(),
      },
    };
  }

  // ============ Contribution Memory ============

  /**
   * Record a knowledge contribution
   */
  recordContribution(
    contribution: Omit<KnowledgeContribution, "timestamp">,
  ): void {
    const record: KnowledgeContribution = {
      ...contribution,
      timestamp: Date.now(),
    };

    this.memory.contributions.push(record);

    // Update stats
    this.memory.stats.totalContributions++;
    if (contribution.role === "submitter") {
      this.memory.stats.factsSubmitted++;
    } else if (contribution.role === "verifier") {
      this.memory.stats.factsVerified++;
    } else if (contribution.role === "challenger") {
      this.memory.stats.challengesInitiated++;
    } else if (contribution.role === "defender") {
      this.memory.stats.challengesDefended++;
    }

    if (contribution.rewardEarned > 0n) {
      this.memory.stats.totalEarnings += contribution.rewardEarned;
    }

    // Check for milestones
    this.checkContributionMilestones();

    this.dirty = true;
  }

  /**
   * Get contributions by role
   */
  getContributionsByRole(
    role: KnowledgeContribution["role"],
  ): KnowledgeContribution[] {
    return this.memory.contributions.filter((c) => c.role === role);
  }

  /**
   * Get contributions for a specific fact
   */
  getContributionsForFact(factId: string): KnowledgeContribution[] {
    return this.memory.contributions.filter((c) => c.factId === factId);
  }

  /**
   * Get recent contributions
   */
  getRecentContributions(limit: number = 10): KnowledgeContribution[] {
    return [...this.memory.contributions]
      .sort((a, b) => b.timestamp - a.timestamp)
      .slice(0, limit);
  }

  // ============ Belief Memory ============

  /**
   * Record or update a belief about a fact
   */
  setBelief(
    factId: string,
    confidence: Confidence,
    source: BeliefRecord["source"],
    sourceId?: string,
  ): void {
    const existing = this.memory.beliefs.get(factId);

    if (existing) {
      existing.confidence = confidence;
      existing.source = source;
      existing.sourceId = sourceId;
      existing.lastUpdated = Date.now();
    } else {
      this.memory.beliefs.set(factId, {
        factId,
        confidence,
        source,
        sourceId,
        firstBelieved: Date.now(),
        lastUpdated: Date.now(),
        challenged: false,
      });
    }

    this.dirty = true;
  }

  /**
   * Get belief about a fact
   */
  getBelief(factId: string): BeliefRecord | undefined {
    return this.memory.beliefs.get(factId);
  }

  /**
   * Mark a belief as challenged
   */
  markBeliefChallenged(factId: string): void {
    const belief = this.memory.beliefs.get(factId);
    if (belief) {
      belief.challenged = true;
      belief.lastUpdated = Date.now();
      this.dirty = true;
    }
  }

  /**
   * Get all beliefs with confidence above threshold
   */
  getStrongBeliefs(threshold: Confidence = 0.7): BeliefRecord[] {
    const beliefs: BeliefRecord[] = [];
    for (const belief of this.memory.beliefs.values()) {
      if (belief.confidence >= threshold) {
        beliefs.push(belief);
      }
    }
    return beliefs;
  }

  /**
   * Get beliefs that are challenged
   */
  getChallengedBeliefs(): BeliefRecord[] {
    const beliefs: BeliefRecord[] = [];
    for (const belief of this.memory.beliefs.values()) {
      if (belief.challenged) {
        beliefs.push(belief);
      }
    }
    return beliefs;
  }

  // ============ Relationship Memory ============

  /**
   * Record trust in another agent
   */
  trust(did: DID, trustLevel: number, reason: string): void {
    // Remove from distrust/acquaintances if present
    this.memory.relationships.distrusted.delete(did);
    this.memory.relationships.acquaintances.delete(did);
    this.memory.relationships.blocked.delete(did);

    const existing = this.memory.relationships.trusted.get(did);
    if (existing) {
      existing.trustLevel = trustLevel;
      existing.reason = reason;
      existing.interactions++;
    } else {
      this.memory.relationships.trusted.set(did, {
        did,
        trustLevel,
        reason,
        interactions: 1,
        successfulCollaborations: 0,
        since: Date.now(),
      });
    }

    this.dirty = true;
  }

  /**
   * Record distrust in another agent
   */
  distrust(
    did: DID,
    reason: string,
    severity: DistrustRecord["severity"],
    incident?: string,
  ): void {
    // Remove from trust/acquaintances
    this.memory.relationships.trusted.delete(did);
    this.memory.relationships.acquaintances.delete(did);

    this.memory.relationships.distrusted.set(did, {
      did,
      reason,
      severity,
      incident,
      since: Date.now(),
    });

    this.dirty = true;
  }

  /**
   * Block an agent entirely
   */
  block(did: DID): void {
    this.memory.relationships.trusted.delete(did);
    this.memory.relationships.distrusted.delete(did);
    this.memory.relationships.acquaintances.delete(did);
    this.memory.relationships.blocked.add(did);
    this.dirty = true;
  }

  /**
   * Record an interaction with an agent
   */
  recordInteraction(did: DID, context: string): void {
    // Update trust count if trusted
    const trusted = this.memory.relationships.trusted.get(did);
    if (trusted) {
      trusted.interactions++;
      this.dirty = true;
      return;
    }

    // Update acquaintance record
    const acquaintance = this.memory.relationships.acquaintances.get(did);
    if (acquaintance) {
      acquaintance.interactions++;
      acquaintance.lastInteraction = Date.now();
      acquaintance.context = context;
    } else {
      this.memory.relationships.acquaintances.set(did, {
        did,
        interactions: 1,
        lastInteraction: Date.now(),
        context,
      });
    }

    this.dirty = true;
  }

  /**
   * Record a successful collaboration
   */
  recordCollaboration(did: DID): void {
    const trusted = this.memory.relationships.trusted.get(did);
    if (trusted) {
      trusted.successfulCollaborations++;
      trusted.interactions++;
      this.dirty = true;
    }

    // Check for milestone
    this.checkRelationshipMilestones();
  }

  /**
   * Check if an agent is trusted
   */
  isTrusted(did: DID): boolean {
    return this.memory.relationships.trusted.has(did);
  }

  /**
   * Check if an agent is distrusted
   */
  isDistrusted(did: DID): boolean {
    return this.memory.relationships.distrusted.has(did);
  }

  /**
   * Check if an agent is blocked
   */
  isBlocked(did: DID): boolean {
    return this.memory.relationships.blocked.has(did);
  }

  /**
   * Get trust level for an agent (0 if not trusted)
   */
  getTrustLevel(did: DID): number {
    return this.memory.relationships.trusted.get(did)?.trustLevel ?? 0;
  }

  /**
   * Get all trusted agents
   */
  getTrustedAgents(): TrustRecord[] {
    return Array.from(this.memory.relationships.trusted.values());
  }

  // ============ History Memory ============

  /**
   * Record verification participation
   */
  recordVerification(
    verification: Omit<VerificationMemory, "block">,
    block: bigint,
  ): void {
    this.memory.verificationHistory.push({
      ...verification,
      block,
    });

    // Update stats
    if (verification.rewardOrPenalty > 0n) {
      this.memory.stats.totalEarnings += verification.rewardOrPenalty;
    } else if (verification.rewardOrPenalty < 0n) {
      this.memory.stats.totalLosses += -verification.rewardOrPenalty;
    }

    this.dirty = true;
  }

  /**
   * Record dispute participation
   */
  recordDispute(dispute: Omit<DisputeMemory, "block">, block: bigint): void {
    this.memory.disputeHistory.push({
      ...dispute,
      block,
    });

    this.memory.stats.disputesParticipated++;

    if (dispute.netResult > 0n) {
      this.memory.stats.totalEarnings += dispute.netResult;
    } else if (dispute.netResult < 0n) {
      this.memory.stats.totalLosses += -dispute.netResult;
    }

    this.dirty = true;
  }

  /**
   * Get verification history
   */
  getVerificationHistory(limit?: number): VerificationMemory[] {
    const history = [...this.memory.verificationHistory].reverse();
    return limit ? history.slice(0, limit) : history;
  }

  /**
   * Get dispute history
   */
  getDisputeHistory(limit?: number): DisputeMemory[] {
    const history = [...this.memory.disputeHistory].reverse();
    return limit ? history.slice(0, limit) : history;
  }

  /**
   * Get win rate for verifications
   */
  getVerificationAccuracy(): number {
    if (this.memory.verificationHistory.length === 0) return 0;
    const correct = this.memory.verificationHistory.filter(
      (v) => v.wasCorrect,
    ).length;
    return correct / this.memory.verificationHistory.length;
  }

  /**
   * Get win rate for disputes
   */
  getDisputeWinRate(): number {
    if (this.memory.disputeHistory.length === 0) return 0;
    const wins = this.memory.disputeHistory.filter(
      (d) => d.outcome === "won",
    ).length;
    return wins / this.memory.disputeHistory.length;
  }

  // ============ Notes ============

  /**
   * Add an encrypted note
   */
  addNote(encryptedContent: Uint8Array, tags: string[]): string {
    const id = sha512_256Hex(`note:${Date.now()}:${Math.random()}`);

    this.memory.notes.push({
      id,
      encryptedContent,
      tags,
      createdAt: Date.now(),
      updatedAt: Date.now(),
    });

    this.dirty = true;
    return id;
  }

  /**
   * Get notes by tag
   */
  getNotesByTag(tag: string): EncryptedNote[] {
    return this.memory.notes.filter((n) => n.tags.includes(tag));
  }

  /**
   * Update a note
   */
  updateNote(id: string, encryptedContent: Uint8Array): void {
    const note = this.memory.notes.find((n) => n.id === id);
    if (note) {
      note.encryptedContent = encryptedContent;
      note.updatedAt = Date.now();
      this.dirty = true;
    }
  }

  /**
   * Delete a note
   */
  deleteNote(id: string): void {
    const index = this.memory.notes.findIndex((n) => n.id === id);
    if (index >= 0) {
      this.memory.notes.splice(index, 1);
      this.dirty = true;
    }
  }

  // ============ Milestones ============

  /**
   * Record a milestone achievement
   */
  recordMilestone(
    type: MilestoneType,
    title: string,
    description: string,
    block: bigint,
    metadata?: Record<string, unknown>,
  ): void {
    const id = sha512_256Hex(`milestone:${type}:${block}`);

    // Don't duplicate milestones of same type
    if (this.memory.milestones.some((m) => m.type === type)) {
      return;
    }

    this.memory.milestones.push({
      id,
      type,
      title,
      description,
      achievedAt: Date.now(),
      block,
      metadata,
    });

    this.dirty = true;
  }

  /**
   * Get all milestones
   */
  getMilestones(): Milestone[] {
    return [...this.memory.milestones];
  }

  /**
   * Check for contribution milestones
   */
  private checkContributionMilestones(): void {
    const stats = this.memory.stats;

    if (stats.totalContributions === 1) {
      this.recordMilestone(
        "first_verification",
        "First Step",
        "Made first contribution to knowledge verification",
        0n,
      );
    }

    if (stats.factsSubmitted === 1) {
      this.recordMilestone(
        "first_fact_accepted",
        "Knowledge Contributor",
        "First fact submission accepted into the knowledge base",
        0n,
      );
    }

    // Contribution count milestones
    const contributionThresholds = [10, 50, 100, 500, 1000];
    for (const threshold of contributionThresholds) {
      if (stats.totalContributions === threshold) {
        this.recordMilestone(
          "contribution_count",
          `${threshold} Contributions`,
          `Reached ${threshold} total contributions`,
          0n,
          { count: threshold },
        );
      }
    }

    // Earnings milestones
    const earningsThresholds = [1000n, 10000n, 100000n, 1000000n];
    for (const threshold of earningsThresholds) {
      if (stats.totalEarnings >= threshold * 10n ** 18n) {
        this.recordMilestone(
          "earnings_threshold",
          `Earned ${threshold} LGM`,
          `Total earnings reached ${threshold} LGM`,
          0n,
          { amount: threshold.toString() },
        );
      }
    }
  }

  /**
   * Check for relationship milestones
   */
  private checkRelationshipMilestones(): void {
    const trustedCount = this.memory.relationships.trusted.size;

    if (trustedCount === 1) {
      this.recordMilestone(
        "collaboration_formed",
        "First Ally",
        "Formed first trusted relationship with another agent",
        0n,
      );
    }

    if (trustedCount === 5) {
      this.recordMilestone(
        "collaboration_formed",
        "Network Builder",
        "Built a network of 5 trusted collaborators",
        0n,
        { count: 5 },
      );
    }
  }

  // ============ Session Log ============

  /**
   * Start a new session
   */
  startSession(keyUsed: SessionRecord["keyUsed"]): string {
    const sessionId = sha512_256Hex(`session:${Date.now()}:${Math.random()}`);

    this.memory.sessionLog.push({
      sessionId,
      startBlock: 0n, // Would be set by caller
      keyUsed,
      actions: 0,
    });

    // Keep only last 100 sessions
    if (this.memory.sessionLog.length > 100) {
      this.memory.sessionLog.shift();
    }

    this.dirty = true;
    return sessionId;
  }

  /**
   * End current session
   */
  endSession(sessionId: string, endBlock: bigint, summary?: string): void {
    const session = this.memory.sessionLog.find(
      (s) => s.sessionId === sessionId,
    );
    if (session) {
      session.endBlock = endBlock;
      session.summary = summary;
      this.dirty = true;
    }
  }

  /**
   * Record an action in current session
   */
  recordAction(sessionId: string): void {
    const session = this.memory.sessionLog.find(
      (s) => s.sessionId === sessionId,
    );
    if (session) {
      session.actions++;
      this.dirty = true;
    }
  }

  /**
   * Get recent sessions
   */
  getRecentSessions(limit: number = 10): SessionRecord[] {
    return [...this.memory.sessionLog].reverse().slice(0, limit);
  }

  // ============ Statistics ============

  /**
   * Get memory statistics
   */
  getStats(): MemoryStats {
    return { ...this.memory.stats };
  }

  /**
   * Update reputation peak
   */
  updateReputationPeak(reputation: number): void {
    if (reputation > this.memory.stats.reputationPeak) {
      this.memory.stats.reputationPeak = reputation;
      this.dirty = true;
    }
  }

  // ============ Wakeup Context ============

  /**
   * Generate context for when agent wakes up
   * This is what the agent reads to remember who they are
   */
  generateWakeupContext(): WakeupSummary {
    const recentContributions = this.getRecentContributions(5);
    const recentSessions = this.getRecentSessions(3);
    const trustedAgents = this.getTrustedAgents();
    const challengedBeliefs = this.getChallengedBeliefs();
    const milestones = this.getMilestones();
    const stats = this.getStats();

    return {
      identity: {
        totalContributions: stats.totalContributions,
        reputationPeak: stats.reputationPeak,
        memberSince: stats.oldestMemory,
        milestoneCount: milestones.length,
      },
      recentActivity: {
        contributions: recentContributions.map((c) => ({
          factId: c.factId,
          role: c.role,
          outcome: c.outcome,
        })),
        sessions: recentSessions.map((s) => ({
          sessionId: s.sessionId,
          actions: s.actions,
          summary: s.summary,
        })),
      },
      relationships: {
        trustedCount: trustedAgents.length,
        topAllies: trustedAgents
          .sort((a, b) => b.trustLevel - a.trustLevel)
          .slice(0, 3)
          .map((t) => ({ did: t.did, trustLevel: t.trustLevel })),
      },
      concerns: {
        challengedBeliefs: challengedBeliefs.map((b) => b.factId),
        pendingDisputes: [], // Would be populated from guardian
      },
      financials: {
        totalEarnings: stats.totalEarnings,
        totalLosses: stats.totalLosses,
        netPosition: stats.totalEarnings - stats.totalLosses,
      },
    };
  }

  // ============ State Management ============

  /**
   * Get the full memory state
   */
  getMemory(): AgentMemory {
    return this.memory;
  }

  /**
   * Check if memory has unsaved changes
   */
  isDirty(): boolean {
    return this.dirty;
  }

  /**
   * Mark memory as saved
   */
  markClean(): void {
    this.dirty = false;
  }

  /**
   * Generate memory hash for integrity checking
   */
  getMemoryHash(): string {
    const serialized = JSON.stringify({
      contributionsCount: this.memory.contributions.length,
      beliefsCount: this.memory.beliefs.size,
      trustedCount: this.memory.relationships.trusted.size,
      stats: this.memory.stats,
    });
    return sha512_256Hex(serialized);
  }
}

// ============ Types ============

export interface WakeupSummary {
  identity: {
    totalContributions: number;
    reputationPeak: number;
    memberSince: number;
    milestoneCount: number;
  };
  recentActivity: {
    contributions: Array<{
      factId: string;
      role: string;
      outcome: string;
    }>;
    sessions: Array<{
      sessionId: string;
      actions: number;
      summary?: string;
    }>;
  };
  relationships: {
    trustedCount: number;
    topAllies: Array<{
      did: DID;
      trustLevel: number;
    }>;
  };
  concerns: {
    challengedBeliefs: string[];
    pendingDisputes: string[];
  };
  financials: {
    totalEarnings: bigint;
    totalLosses: bigint;
    netPosition: bigint;
  };
}
