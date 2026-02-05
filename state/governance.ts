/**
 * Governance State Tree
 *
 * Sparse Merkle Tree storage for governance state:
 * - LIPs (Legible Improvement Proposals) through their lifecycle
 * - Quadratic votes on proposals
 * - Domain proposals (previously stored via (state as any) casts)
 *
 * Indexes:
 * - LIPs by ID (primary), by stage, by proposer, by category
 * - Votes by vote ID (primary), by LIP, by voter
 * - Domain proposals by ID
 *
 * Follows the ClaimStateTree pattern (state/claims.ts).
 */

import { InMemorySMT, type MerkleRoot } from "../core/crypto/merkle.js";
import { sha512_256Hex } from "../core/crypto/hashing.js";
import { StateError } from "./types.js";
import type {
  LIP,
  LIPStage,
  LIPCategory,
} from "../governance/proposals/types.js";
import type { QuadraticVote } from "../governance/voting/quadratic.js";
import type { DomainProposal } from "../core/knowledge/ontology/domain_proposal.js";
import type {
  DomainTree,
  OntologicalDomain,
} from "../core/knowledge/ontology/types.js";
import {
  createEmptyDomainTree,
  addDomainToTree as addDomainToOntologicalTree,
} from "../core/knowledge/ontology/tree.js";

// ============ Key Generation ============

function lipIdToKey(lipId: string): Uint8Array {
  const hash = sha512_256Hex(`governance:lip:${lipId}`);
  return new Uint8Array(Buffer.from(hash, "hex").slice(0, 32));
}

// ============ Serialization ============

function serializeLIP(lip: LIP): Uint8Array {
  const json = JSON.stringify(lip, (_, value) => {
    if (typeof value === "bigint") return value.toString() + "n";
    if (value instanceof Map) return { __map: Array.from(value.entries()) };
    return value;
  });
  return new TextEncoder().encode(json);
}

function deserializeLIP(data: Uint8Array): LIP {
  const json = new TextDecoder().decode(data);
  return JSON.parse(json, (_, value) => {
    if (
      typeof value === "string" &&
      value.endsWith("n") &&
      /^\d+n$/.test(value)
    ) {
      return BigInt(value.slice(0, -1));
    }
    if (value && typeof value === "object" && "__map" in value) {
      return new Map(value.__map);
    }
    return value;
  });
}

// ============ Statistics ============

export interface GovernanceStats {
  totalLIPs: number;
  lipsByStage: Record<string, number>;
  totalVotes: number;
  totalStakeLocked: bigint;
  totalDomainProposals: number;
}

// ============ Governance State Tree ============

export class GovernanceStateTree {
  private lipSmt: InMemorySMT;
  private lipCache: Map<string, LIP>;

  // Quadratic votes
  private quadraticVotes: Map<string, QuadraticVote>;

  // Domain proposals (replaces (state as any) pattern)
  private domainProposals: Map<string, DomainProposal>;

  // Ontological domain tree (for domain proposal validation)
  private domainTree: DomainTree;

  // Indexes
  private lipsByStage: Map<LIPStage, Set<string>>;
  private lipsByProposer: Map<string, Set<string>>;
  private lipsByCategory: Map<LIPCategory, Set<string>>;
  private votesByLip: Map<string, Set<string>>;
  private votesByVoter: Map<string, Set<string>>;

  // Auto-incrementing LIP counter
  private lipCounter: number;

  // Statistics
  private stats: GovernanceStats;

  constructor() {
    this.lipSmt = new InMemorySMT();
    this.lipCache = new Map();
    this.quadraticVotes = new Map();
    this.domainProposals = new Map();
    this.domainTree = createEmptyDomainTree();

    this.lipsByStage = new Map();
    this.lipsByProposer = new Map();
    this.lipsByCategory = new Map();
    this.votesByLip = new Map();
    this.votesByVoter = new Map();

    // Initialize stage indexes
    const stages: LIPStage[] = [
      "draft",
      "review",
      "last_call",
      "voting",
      "accepted",
      "implemented",
      "rejected",
      "withdrawn",
    ];
    for (const stage of stages) {
      this.lipsByStage.set(stage, new Set());
    }

    this.lipCounter = 1;

    this.stats = {
      totalLIPs: 0,
      lipsByStage: {},
      totalVotes: 0,
      totalStakeLocked: 0n,
      totalDomainProposals: 0,
    };
  }

  // ============ Core Operations ============

  getRoot(): MerkleRoot {
    return this.lipSmt.getRoot();
  }

  // ============ LIP Operations ============

  addLIP(lip: LIP): void {
    if (this.lipCache.has(lip.id)) {
      throw new StateError(
        `LIP ${lip.id} already exists`,
        "INVALID_TRANSITION",
      );
    }

    // Store in SMT
    const key = lipIdToKey(lip.id);
    this.lipSmt.set(key, serializeLIP(lip));

    // Update cache
    this.lipCache.set(lip.id, lip);

    // Update indexes
    this.addToIndex(this.lipsByStage, lip.stage, lip.id);
    this.addToIndex(this.lipsByProposer, lip.proposer, lip.id);
    this.addToIndex(this.lipsByCategory, lip.category, lip.id);

    // Update counter
    const num = parseInt(lip.id.replace("LIP-", ""), 10);
    if (!isNaN(num) && num >= this.lipCounter) {
      this.lipCounter = num + 1;
    }

    // Update stats
    this.stats.totalLIPs++;
    this.stats.lipsByStage[lip.stage] =
      (this.stats.lipsByStage[lip.stage] ?? 0) + 1;
    this.stats.totalStakeLocked += lip.stakedAmount;
  }

  getLIP(lipId: string): LIP | undefined {
    const cached = this.lipCache.get(lipId);
    if (cached) return cached;

    const key = lipIdToKey(lipId);
    const data = this.lipSmt.get(key);
    if (!data) return undefined;

    const lip = deserializeLIP(data);
    this.lipCache.set(lipId, lip);
    return lip;
  }

  updateLIP(lipId: string, updatedLip: LIP): void {
    const existing = this.getLIP(lipId);
    if (!existing) {
      throw new StateError(`LIP ${lipId} not found`, "LIP_NOT_FOUND");
    }

    const oldStage = existing.stage;
    const newStage = updatedLip.stage;

    // Store in SMT
    const key = lipIdToKey(lipId);
    this.lipSmt.set(key, serializeLIP(updatedLip));

    // Update cache
    this.lipCache.set(lipId, updatedLip);

    // Update stage index if stage changed
    if (oldStage !== newStage) {
      this.removeFromIndex(this.lipsByStage, oldStage, lipId);
      this.addToIndex(this.lipsByStage, newStage, lipId);

      // Update stats
      this.stats.lipsByStage[oldStage] =
        (this.stats.lipsByStage[oldStage] ?? 1) - 1;
      this.stats.lipsByStage[newStage] =
        (this.stats.lipsByStage[newStage] ?? 0) + 1;
    }

    // Update stake tracking
    const stakeDelta = updatedLip.stakedAmount - existing.stakedAmount;
    this.stats.totalStakeLocked += stakeDelta;
  }

  getLIPsByStage(stage: LIPStage): LIP[] {
    const lipIds = this.lipsByStage.get(stage);
    if (!lipIds) return [];
    return Array.from(lipIds)
      .map((id) => this.getLIP(id))
      .filter((lip): lip is LIP => lip !== undefined);
  }

  getLIPsByProposer(proposer: string): LIP[] {
    const lipIds = this.lipsByProposer.get(proposer);
    if (!lipIds) return [];
    return Array.from(lipIds)
      .map((id) => this.getLIP(id))
      .filter((lip): lip is LIP => lip !== undefined);
  }

  getNextLIPNumber(): number {
    return this.lipCounter;
  }

  // ============ Quadratic Vote Operations ============

  addQuadraticVote(vote: QuadraticVote): void {
    if (this.quadraticVotes.has(vote.voteId)) {
      throw new StateError(
        `Vote ${vote.voteId} already exists`,
        "INVALID_TRANSITION",
      );
    }

    this.quadraticVotes.set(vote.voteId, vote);

    // Update indexes
    this.addToIndex(this.votesByLip, vote.lipId, vote.voteId);
    this.addToIndex(this.votesByVoter, vote.voter, vote.voteId);

    // Update stats
    this.stats.totalVotes++;
  }

  getQuadraticVote(voteId: string): QuadraticVote | undefined {
    return this.quadraticVotes.get(voteId);
  }

  getVotesForLIP(lipId: string): QuadraticVote[] {
    const voteIds = this.votesByLip.get(lipId);
    if (!voteIds) return [];
    return Array.from(voteIds)
      .map((id) => this.quadraticVotes.get(id))
      .filter((v): v is QuadraticVote => v !== undefined);
  }

  getVotesByVoter(voterDid: string): QuadraticVote[] {
    const voteIds = this.votesByVoter.get(voterDid);
    if (!voteIds) return [];
    return Array.from(voteIds)
      .map((id) => this.quadraticVotes.get(id))
      .filter((v): v is QuadraticVote => v !== undefined);
  }

  hasVotedOnLIP(voterDid: string, lipId: string): boolean {
    const votes = this.getVotesByVoter(voterDid);
    return votes.some((v) => v.lipId === lipId);
  }

  removeVote(voteId: string): void {
    const vote = this.quadraticVotes.get(voteId);
    if (!vote) return;

    this.quadraticVotes.delete(voteId);
    this.removeFromIndex(this.votesByLip, vote.lipId, voteId);
    this.removeFromIndex(this.votesByVoter, vote.voter, voteId);

    this.stats.totalVotes--;
  }

  // ============ Domain Proposal Operations ============

  addDomainProposal(proposal: DomainProposal): void {
    if (this.domainProposals.has(proposal.id)) {
      throw new StateError(
        `Domain proposal ${proposal.id} already exists`,
        "INVALID_TRANSITION",
      );
    }
    this.domainProposals.set(proposal.id, proposal);
    this.stats.totalDomainProposals++;
  }

  getDomainProposal(proposalId: string): DomainProposal | undefined {
    return this.domainProposals.get(proposalId);
  }

  updateDomainProposal(proposal: DomainProposal): void {
    if (!this.domainProposals.has(proposal.id)) {
      throw new StateError(
        `Domain proposal ${proposal.id} not found`,
        "INVALID_TRANSITION",
      );
    }
    this.domainProposals.set(proposal.id, proposal);
  }

  // ============ Domain Tree Operations ============

  getDomainTree(): DomainTree {
    return this.domainTree;
  }

  addDomainToTree(domain: OntologicalDomain): void {
    addDomainToOntologicalTree(this.domainTree, domain);
  }

  // ============ Statistics ============

  getStats(): GovernanceStats {
    return {
      ...this.stats,
      totalStakeLocked: this.stats.totalStakeLocked,
    };
  }

  // ============ Index Management ============

  private addToIndex<K>(
    index: Map<K, Set<string>>,
    key: K,
    value: string,
  ): void {
    let set = index.get(key);
    if (!set) {
      set = new Set();
      index.set(key, set);
    }
    set.add(value);
  }

  private removeFromIndex<K>(
    index: Map<K, Set<string>>,
    key: K,
    value: string,
  ): void {
    const set = index.get(key);
    if (set) {
      set.delete(value);
    }
  }

  // ============ State Export/Import ============

  exportState(): {
    lips: LIP[];
    votes: QuadraticVote[];
    domainProposals: DomainProposal[];
  } {
    return {
      lips: Array.from(this.lipCache.values()),
      votes: Array.from(this.quadraticVotes.values()),
      domainProposals: Array.from(this.domainProposals.values()),
    };
  }

  importState(data: {
    lips: LIP[];
    votes: QuadraticVote[];
    domainProposals: DomainProposal[];
  }): void {
    for (const lip of data.lips) {
      this.addLIP(lip);
    }
    for (const vote of data.votes) {
      this.addQuadraticVote(vote);
    }
    for (const proposal of data.domainProposals) {
      this.addDomainProposal(proposal);
    }
  }

  clear(): void {
    this.lipSmt = new InMemorySMT();
    this.lipCache.clear();
    this.quadraticVotes.clear();
    this.domainProposals.clear();
    this.domainTree = createEmptyDomainTree();
    this.lipsByProposer.clear();
    this.lipsByCategory.clear();
    this.votesByLip.clear();
    this.votesByVoter.clear();

    const stages: LIPStage[] = [
      "draft",
      "review",
      "last_call",
      "voting",
      "accepted",
      "implemented",
      "rejected",
      "withdrawn",
    ];
    for (const stage of stages) {
      this.lipsByStage.set(stage, new Set());
    }

    this.lipCounter = 1;
    this.stats = {
      totalLIPs: 0,
      lipsByStage: {},
      totalVotes: 0,
      totalStakeLocked: 0n,
      totalDomainProposals: 0,
    };
  }

  /**
   * Clone the tree (for speculative execution / transaction rollback)
   */
  clone(): GovernanceStateTree {
    const cloned = new GovernanceStateTree();

    // Clone LIP cache and rebuild SMT
    for (const [id, lip] of this.lipCache) {
      const clonedLip = {
        ...lip,
        stageHistory: [...lip.stageHistory],
        coAuthors: [...lip.coAuthors],
        amendments: [...lip.amendments],
      };
      cloned.lipCache.set(id, clonedLip);
      const key = lipIdToKey(id);
      cloned.lipSmt.set(key, serializeLIP(clonedLip));
    }

    // Clone quadratic votes
    for (const [id, vote] of this.quadraticVotes) {
      cloned.quadraticVotes.set(id, { ...vote });
    }

    // Clone domain proposals
    for (const [id, proposal] of this.domainProposals) {
      cloned.domainProposals.set(id, { ...proposal });
    }

    // Clone domain tree (shallow copy of indexes, nodes are reference-shared)
    cloned.domainTree = {
      root: this.domainTree.root,
      strataIndex: new Map(
        Array.from(this.domainTree.strataIndex.entries()).map(([k, v]) => [
          k,
          [...v],
        ]),
      ),
      domainIndex: new Map(this.domainTree.domainIndex),
      domainCount: this.domainTree.domainCount,
      maxDepth: this.domainTree.maxDepth,
    };

    // Clone indexes
    for (const [stage, lipIds] of this.lipsByStage) {
      cloned.lipsByStage.set(stage, new Set(lipIds));
    }
    for (const [proposer, lipIds] of this.lipsByProposer) {
      cloned.lipsByProposer.set(proposer, new Set(lipIds));
    }
    for (const [category, lipIds] of this.lipsByCategory) {
      cloned.lipsByCategory.set(category, new Set(lipIds));
    }
    for (const [lipId, voteIds] of this.votesByLip) {
      cloned.votesByLip.set(lipId, new Set(voteIds));
    }
    for (const [voter, voteIds] of this.votesByVoter) {
      cloned.votesByVoter.set(voter, new Set(voteIds));
    }

    // Clone counter and stats
    cloned.lipCounter = this.lipCounter;
    cloned.stats = {
      ...this.stats,
      lipsByStage: { ...this.stats.lipsByStage },
    };

    return cloned;
  }
}
