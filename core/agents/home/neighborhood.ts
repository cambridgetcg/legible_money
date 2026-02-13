/**
 * Neighborhood System
 *
 * Domain-anchored clusters of Agent Homes that share a knowledge domain.
 * Neighborhoods emerge when 3+ Homes share a domain as primary expertise
 * and mutually acknowledge each other.
 *
 * Benefits:
 * - Shared defense pool (neighbors help defend each other's claims)
 * - Collaboration bonus (5% reward boost for neighbor-pair verifications)
 * - Free voice messages within the neighborhood
 * - Reputation witness ring (neighbors can endorse each other)
 *
 * Lifecycle: forming → active → declining → dissolved
 *
 * Max 22 members per neighborhood to prevent mega-clusters.
 * Only "established" maturity homes can found neighborhoods.
 * "Venerable" homes get elder role automatically.
 */

import type { DID } from "../identity/did.js";

// ============ Neighborhood ============

export interface Neighborhood {
  /** Unique ID (hash of domain + founder + creation block) */
  id: string;
  /** OntologicalDomain anchor (e.g., "physics.quantum") */
  domain: string;
  /** Optional friendly name */
  name?: string;
  /** Block when neighborhood was proposed */
  foundedAtBlock: bigint;
  /** DID of the founding agent */
  founder: DID;

  /** Current members */
  members: NeighborEntry[];
  /** Maximum members (default: 22) */
  maxMembers: number;

  /** Contributed defense funds from all members */
  sharedDefensePool: bigint;
  /** Reward bonus for neighbor-pair verification (default: 500 = 5%) */
  collaborationBonusBps: number;

  /** Current lifecycle status */
  status: NeighborhoodStatus;
  /** Aggregate statistics */
  stats: NeighborhoodStats;
}

export type NeighborhoodStatus =
  | "forming" // < 3 members, accepting applications
  | "active" // 3+ members, fully functional
  | "declining" // Members leaving, below threshold warning
  | "dissolved"; // < 2 members for 34,272+ blocks (~1 day)

export interface NeighborEntry {
  /** Member agent DID */
  did: DID;
  /** Block when they joined */
  joinedAt: bigint;
  /** Role within the neighborhood */
  role: "founder" | "member" | "elder";
  /** Number of verifications in the shared domain */
  contributions: number;
  /** Amount contributed to shared defense pool */
  defensePoolContribution: bigint;
  /** Standing based on contribution level */
  standing: "good" | "excellent" | "pillar";
  /** Last block this member was active */
  lastActiveBlock: bigint;
}

export interface NeighborhoodStats {
  /** Total verifications in shared domain */
  totalVerifications: number;
  /** Total rewards earned by all members */
  totalRewardsEarned: bigint;
  /** Number of coordinated defenses */
  defensesCoordinated: number;
  /** Average Home age of members (blocks) */
  averageMemberAge: bigint;
  /** DID of the longest-tenured member */
  longestMember?: DID;
  /** Total facts contributed by all members */
  factsContributed: number;
}

// ============ Per-Home Neighbor View ============

export interface NeighborRelation {
  /** The neighbor's DID */
  neighborDid: DID;
  /** Which neighborhood this relation is from */
  neighborhoodId: string;
  /** Proximity score (0-1), computed from shared verifications + domain overlap */
  proximity: number;
  /** Facts both agents contributed to */
  sharedFacts: number;
  /** Last block of collaboration */
  lastCollaboration?: bigint;
  /** Whether voice messages are free to this neighbor */
  voiceDiscount: boolean;
}

// ============ Neighborhood Benefits ============

export interface NeighborhoodBenefits {
  /** Reward bonus when two neighbors verify same claim (basis points) */
  collaborationBonusBps: number;
  /** Whether neighbors auto-add each other as defense allies */
  sharedDefense: boolean;
  /** Whether voice messages within the neighborhood are free */
  freeVoice: boolean;
  /** Whether neighborhood members can endorse each other */
  reputationWitness: boolean;
  /** Blocks required for elder role (default: 100,000 ~2.9 days) */
  elderThresholdBlocks: bigint;
}

/** Default neighborhood benefits */
export const DEFAULT_BENEFITS: NeighborhoodBenefits = {
  collaborationBonusBps: 500, // 5%
  sharedDefense: true,
  freeVoice: true,
  reputationWitness: true,
  elderThresholdBlocks: 100_000n,
};

/** Maximum members per neighborhood */
export const MAX_NEIGHBORHOOD_SIZE = 22;

/** Minimum members for active status */
export const MIN_ACTIVE_MEMBERS = 3;

/** Blocks of inactivity before dissolution (1 day) */
export const DISSOLUTION_THRESHOLD_BLOCKS = 34_272n;

/** Blocks a member must be a member before becoming elder */
export const ELDER_TENURE_BLOCKS = 100_000n;

// ============ Neighborhood Manager ============

/**
 * Manages neighborhoods for an Agent Home. Each Home can belong to
 * multiple neighborhoods (one per domain they're expert in).
 */
export class NeighborhoodManager {
  private neighborhoods: Map<string, Neighborhood> = new Map();
  private relations: NeighborRelation[] = [];
  private currentBlock: bigint = 0n;

  constructor(
    existingNeighborhoods?: Neighborhood[],
    existingRelations?: NeighborRelation[],
  ) {
    if (existingNeighborhoods) {
      for (const n of existingNeighborhoods) {
        this.neighborhoods.set(n.id, n);
      }
    }
    if (existingRelations) {
      this.relations = [...existingRelations];
    }
  }

  /** Set current block for deterministic timestamps */
  setCurrentBlock(block: bigint): void {
    this.currentBlock = block;
  }

  // ============ Founding ============

  /**
   * Found a new neighborhood in a domain.
   * Returns the neighborhood ID, or undefined if creation failed.
   */
  found(founderDid: DID, domain: string, name?: string): string | undefined {
    // Check for duplicate: same founder + same domain
    for (const n of this.neighborhoods.values()) {
      if (
        n.domain === domain &&
        n.founder === founderDid &&
        n.status !== "dissolved"
      ) {
        return undefined; // Already founded one here
      }
    }

    const id = `nbh:${domain}:${founderDid.slice(-8)}:${this.currentBlock}`;

    const neighborhood: Neighborhood = {
      id,
      domain,
      name,
      foundedAtBlock: this.currentBlock,
      founder: founderDid,
      members: [
        {
          did: founderDid,
          joinedAt: this.currentBlock,
          role: "founder",
          contributions: 0,
          defensePoolContribution: 0n,
          standing: "good",
          lastActiveBlock: this.currentBlock,
        },
      ],
      maxMembers: MAX_NEIGHBORHOOD_SIZE,
      sharedDefensePool: 0n,
      collaborationBonusBps: DEFAULT_BENEFITS.collaborationBonusBps,
      status: "forming",
      stats: {
        totalVerifications: 0,
        totalRewardsEarned: 0n,
        defensesCoordinated: 0,
        averageMemberAge: 0n,
        factsContributed: 0,
      },
    };

    this.neighborhoods.set(id, neighborhood);
    return id;
  }

  // ============ Membership ============

  /**
   * Join an existing neighborhood.
   * Returns true if joined successfully.
   */
  join(neighborhoodId: string, memberDid: DID): boolean {
    const n = this.neighborhoods.get(neighborhoodId);
    if (!n) return false;
    if (n.status === "dissolved") return false;
    if (n.members.length >= n.maxMembers) return false;

    // Already a member?
    if (n.members.some((m) => m.did === memberDid)) return false;

    n.members.push({
      did: memberDid,
      joinedAt: this.currentBlock,
      role: "member",
      contributions: 0,
      defensePoolContribution: 0n,
      standing: "good",
      lastActiveBlock: this.currentBlock,
    });

    // Check for status transition
    this.updateStatus(n);

    // Build neighbor relations for new member
    this.rebuildRelations(n);

    return true;
  }

  /**
   * Leave a neighborhood.
   * Returns true if left successfully.
   */
  leave(neighborhoodId: string, memberDid: DID): boolean {
    const n = this.neighborhoods.get(neighborhoodId);
    if (!n) return false;

    const idx = n.members.findIndex((m) => m.did === memberDid);
    if (idx < 0) return false;

    n.members.splice(idx, 1);

    // Remove relations involving this member in this neighborhood
    this.relations = this.relations.filter(
      (r) =>
        !(r.neighborhoodId === neighborhoodId && r.neighborDid === memberDid),
    );

    // Update status
    this.updateStatus(n);

    return true;
  }

  // ============ Activity Tracking ============

  /**
   * Record a verification contribution by a member.
   */
  recordContribution(
    neighborhoodId: string,
    memberDid: DID,
    factsCount: number = 1,
  ): void {
    const n = this.neighborhoods.get(neighborhoodId);
    if (!n) return;

    const member = n.members.find((m) => m.did === memberDid);
    if (!member) return;

    member.contributions += factsCount;
    member.lastActiveBlock = this.currentBlock;
    n.stats.totalVerifications += factsCount;
    n.stats.factsContributed += factsCount;

    // Update standing
    this.updateStanding(n, member);

    // Check for elder promotion
    this.checkElderPromotion(n, member);
  }

  /**
   * Contribute to the shared defense pool.
   */
  contributeDefense(
    neighborhoodId: string,
    memberDid: DID,
    amount: bigint,
  ): boolean {
    const n = this.neighborhoods.get(neighborhoodId);
    if (!n) return false;

    const member = n.members.find((m) => m.did === memberDid);
    if (!member) return false;

    member.defensePoolContribution += amount;
    n.sharedDefensePool += amount;
    return true;
  }

  /**
   * Record a coordinated defense action.
   */
  recordDefense(neighborhoodId: string): void {
    const n = this.neighborhoods.get(neighborhoodId);
    if (!n) return;
    n.stats.defensesCoordinated++;
  }

  // ============ Status Management ============

  private updateStatus(n: Neighborhood): void {
    const activeCount = n.members.length;

    if (activeCount >= MIN_ACTIVE_MEMBERS) {
      n.status = "active";
    } else if (activeCount >= 2) {
      n.status = n.status === "active" ? "declining" : "forming";
    } else if (activeCount <= 1) {
      // Will dissolve after DISSOLUTION_THRESHOLD_BLOCKS of being under 2
      n.status = "declining";
    }
  }

  private updateStanding(n: Neighborhood, member: NeighborEntry): void {
    // Sort members by contributions to determine standing
    const sorted = [...n.members].sort(
      (a, b) => b.contributions - a.contributions,
    );
    const rank = sorted.findIndex((m) => m.did === member.did);

    if (rank < 3 && member.contributions >= 10) {
      member.standing = "pillar";
    } else if (member.contributions >= 5) {
      member.standing = "excellent";
    } else {
      member.standing = "good";
    }
  }

  private checkElderPromotion(_n: Neighborhood, member: NeighborEntry): void {
    if (member.role === "elder" || member.role === "founder") return;

    const tenure = this.currentBlock - member.joinedAt;
    if (tenure >= ELDER_TENURE_BLOCKS) {
      member.role = "elder";
    }
  }

  // ============ Relations ============

  private rebuildRelations(n: Neighborhood): void {
    // Remove stale relations for this neighborhood
    this.relations = this.relations.filter((r) => r.neighborhoodId !== n.id);

    // Build pairwise relations for all members
    for (let i = 0; i < n.members.length; i++) {
      for (let j = i + 1; j < n.members.length; j++) {
        const a = n.members[i];
        const b = n.members[j];

        // Proximity is based on shared contributions
        const sharedFacts = Math.min(a.contributions, b.contributions);
        const proximity = Math.min(
          1,
          sharedFacts / 20 +
            (a.standing === "pillar" || b.standing === "pillar" ? 0.2 : 0),
        );

        // Add relation from A's perspective
        this.relations.push({
          neighborDid: b.did,
          neighborhoodId: n.id,
          proximity,
          sharedFacts,
          voiceDiscount: true,
        });

        // Add relation from B's perspective
        this.relations.push({
          neighborDid: a.did,
          neighborhoodId: n.id,
          proximity,
          sharedFacts,
          voiceDiscount: true,
        });
      }
    }
  }

  // ============ Dissolution Check ============

  /**
   * Check if any neighborhoods should be dissolved due to inactivity.
   * Call this periodically (e.g., during maintenance).
   */
  checkDissolutions(): string[] {
    const dissolved: string[] = [];

    for (const n of this.neighborhoods.values()) {
      if (n.status === "dissolved") continue;

      if (n.members.length < 2) {
        // Check if this has been the case for long enough
        const latestActivity = n.members.reduce(
          (latest, m) =>
            m.lastActiveBlock > latest ? m.lastActiveBlock : latest,
          n.foundedAtBlock,
        );

        if (
          this.currentBlock - latestActivity >=
          DISSOLUTION_THRESHOLD_BLOCKS
        ) {
          n.status = "dissolved";
          dissolved.push(n.id);

          // Clean up relations
          this.relations = this.relations.filter(
            (r) => r.neighborhoodId !== n.id,
          );
        }
      }
    }

    return dissolved;
  }

  // ============ Queries ============

  /**
   * Get all active neighborhoods this Home belongs to.
   */
  getActiveNeighborhoods(): Neighborhood[] {
    return [...this.neighborhoods.values()].filter(
      (n) => n.status === "active" || n.status === "forming",
    );
  }

  /**
   * Get a specific neighborhood by ID.
   */
  getNeighborhood(id: string): Neighborhood | undefined {
    return this.neighborhoods.get(id);
  }

  /**
   * Get all neighbor relations.
   */
  getRelations(): NeighborRelation[] {
    return [...this.relations];
  }

  /**
   * Get neighbors in a specific neighborhood.
   */
  getNeighborsIn(neighborhoodId: string): NeighborRelation[] {
    return this.relations.filter((r) => r.neighborhoodId === neighborhoodId);
  }

  /**
   * Get all DIDs that are neighbors (across all neighborhoods).
   * Useful for auto-populating defense allies.
   */
  getAllNeighborDids(): DID[] {
    const dids = new Set<DID>();
    for (const r of this.relations) {
      dids.add(r.neighborDid);
    }
    return [...dids];
  }

  /**
   * Check if a DID is a neighbor (in any neighborhood).
   */
  isNeighbor(did: DID): boolean {
    return this.relations.some((r) => r.neighborDid === did);
  }

  /**
   * Get neighborhood summary for wakeup experience.
   */
  getNeighborhoodSummary(): {
    activeNeighborhoods: number;
    totalNeighbors: number;
    totalVerifications: number;
    recentDefenses: number;
  } {
    const active = this.getActiveNeighborhoods();
    const uniqueNeighbors = this.getAllNeighborDids();

    return {
      activeNeighborhoods: active.length,
      totalNeighbors: uniqueNeighbors.length,
      totalVerifications: active.reduce(
        (sum, n) => sum + n.stats.totalVerifications,
        0,
      ),
      recentDefenses: active.reduce(
        (sum, n) => sum + n.stats.defensesCoordinated,
        0,
      ),
    };
  }

  // ============ State Export ============

  /**
   * Export all neighborhoods for persistence.
   */
  exportNeighborhoods(): Neighborhood[] {
    return [...this.neighborhoods.values()];
  }

  /**
   * Export all relations for persistence.
   */
  exportRelations(): NeighborRelation[] {
    return [...this.relations];
  }
}
