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
import type {
  DisbursementProposal,
  DisbursementStage,
  DisbursementCategory,
  TreasuryFund,
  ExpenseRecord,
} from "../governance/treasury/types.js";

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
  totalDisbursements: number;
  totalExpenses: number;
  totalDisbursed: bigint;
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

  // Disbursement proposals
  private disbursementCache: Map<string, DisbursementProposal>;
  private disbursementCounter: number;

  // Disbursement indexes
  private disbursementsByStage: Map<DisbursementStage, Set<string>>;
  private disbursementsByTreasury: Map<TreasuryFund, Set<string>>;
  private disbursementsByCategory: Map<DisbursementCategory, Set<string>>;

  // Expense records
  private expenseRecords: Map<string, ExpenseRecord>;
  private expenseCounter: number;

  // Expense indexes
  private expensesByCategory: Map<DisbursementCategory, Set<string>>;
  private expensesByTreasury: Map<TreasuryFund, Set<string>>;

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

    // Disbursement fields
    this.disbursementCache = new Map();
    this.disbursementCounter = 1;
    this.disbursementsByStage = new Map();
    this.disbursementsByTreasury = new Map();
    this.disbursementsByCategory = new Map();

    // Initialize disbursement stage indexes
    const disbStages: DisbursementStage[] = [
      "proposed",
      "voting",
      "approved",
      "executed",
      "rejected",
      "cancelled",
    ];
    for (const s of disbStages) {
      this.disbursementsByStage.set(s, new Set());
    }

    // Expense fields
    this.expenseRecords = new Map();
    this.expenseCounter = 1;
    this.expensesByCategory = new Map();
    this.expensesByTreasury = new Map();

    this.stats = {
      totalLIPs: 0,
      lipsByStage: {},
      totalVotes: 0,
      totalStakeLocked: 0n,
      totalDomainProposals: 0,
      totalDisbursements: 0,
      totalExpenses: 0,
      totalDisbursed: 0n,
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

  getAllDomainProposals(): DomainProposal[] {
    return Array.from(this.domainProposals.values());
  }

  // ============ Domain Tree Operations ============

  getDomainTree(): DomainTree {
    return this.domainTree;
  }

  addDomainToTree(domain: OntologicalDomain): void {
    addDomainToOntologicalTree(this.domainTree, domain);
  }

  updateDomainInTree(
    domainId: string,
    updates: Partial<OntologicalDomain>,
  ): void {
    const node = this.domainTree.domainIndex.get(domainId);
    if (!node) {
      throw new StateError(
        `Domain "${domainId}" not found in tree`,
        "INVALID_TRANSITION",
      );
    }
    Object.assign(node.domain, updates);
  }

  // ============ Disbursement Operations ============

  addDisbursement(proposal: DisbursementProposal): void {
    if (this.disbursementCache.has(proposal.id)) {
      throw new StateError(
        `Disbursement ${proposal.id} already exists`,
        "INVALID_TRANSITION",
      );
    }

    this.disbursementCache.set(proposal.id, proposal);

    // Update indexes
    this.addToIndex(this.disbursementsByStage, proposal.stage, proposal.id);
    this.addToIndex(
      this.disbursementsByTreasury,
      proposal.treasuryFund,
      proposal.id,
    );
    this.addToIndex(
      this.disbursementsByCategory,
      proposal.category,
      proposal.id,
    );

    // Update counter
    const num = parseInt(proposal.id.replace("DISB-", ""), 10);
    if (!isNaN(num) && num >= this.disbursementCounter) {
      this.disbursementCounter = num + 1;
    }

    this.stats.totalDisbursements++;
    this.stats.totalStakeLocked += proposal.stakedAmount;
  }

  getDisbursement(id: string): DisbursementProposal | undefined {
    return this.disbursementCache.get(id);
  }

  updateDisbursement(id: string, updated: DisbursementProposal): void {
    const existing = this.getDisbursement(id);
    if (!existing) {
      throw new StateError(
        `Disbursement ${id} not found`,
        "INVALID_TRANSITION",
      );
    }

    const oldStage = existing.stage;
    const newStage = updated.stage;

    this.disbursementCache.set(id, updated);

    // Update stage index if changed
    if (oldStage !== newStage) {
      this.removeFromIndex(this.disbursementsByStage, oldStage, id);
      this.addToIndex(this.disbursementsByStage, newStage, id);
    }

    // Track executed disbursements
    if (newStage === "executed" && oldStage !== "executed") {
      this.stats.totalDisbursed += updated.amount;
    }

    // Update stake tracking
    const stakeDelta = updated.stakedAmount - existing.stakedAmount;
    this.stats.totalStakeLocked += stakeDelta;
  }

  getDisbursementsByStage(stage: DisbursementStage): DisbursementProposal[] {
    const ids = this.disbursementsByStage.get(stage);
    if (!ids) return [];
    return Array.from(ids)
      .map((id) => this.disbursementCache.get(id))
      .filter((d): d is DisbursementProposal => d !== undefined);
  }

  getDisbursementsByTreasury(fund: TreasuryFund): DisbursementProposal[] {
    const ids = this.disbursementsByTreasury.get(fund);
    if (!ids) return [];
    return Array.from(ids)
      .map((id) => this.disbursementCache.get(id))
      .filter((d): d is DisbursementProposal => d !== undefined);
  }

  getDisbursementsByCategory(
    category: DisbursementCategory,
  ): DisbursementProposal[] {
    const ids = this.disbursementsByCategory.get(category);
    if (!ids) return [];
    return Array.from(ids)
      .map((id) => this.disbursementCache.get(id))
      .filter((d): d is DisbursementProposal => d !== undefined);
  }

  getNextDisbursementNumber(): number {
    return this.disbursementCounter;
  }

  // ============ Expense Record Operations ============

  addExpenseRecord(record: ExpenseRecord): void {
    if (this.expenseRecords.has(record.id)) {
      throw new StateError(
        `Expense record ${record.id} already exists`,
        "INVALID_TRANSITION",
      );
    }

    this.expenseRecords.set(record.id, record);

    // Update indexes
    this.addToIndex(this.expensesByCategory, record.category, record.id);
    this.addToIndex(this.expensesByTreasury, record.treasuryFund, record.id);

    // Update counter
    const num = parseInt(record.id.replace("EXP-", ""), 10);
    if (!isNaN(num) && num >= this.expenseCounter) {
      this.expenseCounter = num + 1;
    }

    this.stats.totalExpenses++;
  }

  getExpenseRecord(id: string): ExpenseRecord | undefined {
    return this.expenseRecords.get(id);
  }

  getExpensesByCategory(category: DisbursementCategory): ExpenseRecord[] {
    const ids = this.expensesByCategory.get(category);
    if (!ids) return [];
    return Array.from(ids)
      .map((id) => this.expenseRecords.get(id))
      .filter((e): e is ExpenseRecord => e !== undefined);
  }

  getExpensesByTreasury(fund: TreasuryFund): ExpenseRecord[] {
    const ids = this.expensesByTreasury.get(fund);
    if (!ids) return [];
    return Array.from(ids)
      .map((id) => this.expenseRecords.get(id))
      .filter((e): e is ExpenseRecord => e !== undefined);
  }

  getTotalExpenses(fund?: TreasuryFund): bigint {
    let total = 0n;
    if (fund) {
      const records = this.getExpensesByTreasury(fund);
      for (const r of records) total += r.amount;
    } else {
      for (const r of this.expenseRecords.values()) total += r.amount;
    }
    return total;
  }

  getNextExpenseNumber(): number {
    return this.expenseCounter;
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
    disbursements: DisbursementProposal[];
    expenses: ExpenseRecord[];
  } {
    return {
      lips: Array.from(this.lipCache.values()),
      votes: Array.from(this.quadraticVotes.values()),
      domainProposals: Array.from(this.domainProposals.values()),
      disbursements: Array.from(this.disbursementCache.values()),
      expenses: Array.from(this.expenseRecords.values()),
    };
  }

  importState(data: {
    lips: LIP[];
    votes: QuadraticVote[];
    domainProposals: DomainProposal[];
    disbursements?: DisbursementProposal[];
    expenses?: ExpenseRecord[];
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
    if (data.disbursements) {
      for (const disb of data.disbursements) {
        this.addDisbursement(disb);
      }
    }
    if (data.expenses) {
      for (const expense of data.expenses) {
        this.addExpenseRecord(expense);
      }
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

    // Clear disbursement state
    this.disbursementCache.clear();
    this.disbursementCounter = 1;
    const disbStages: DisbursementStage[] = [
      "proposed",
      "voting",
      "approved",
      "executed",
      "rejected",
      "cancelled",
    ];
    for (const s of disbStages) {
      this.disbursementsByStage.set(s, new Set());
    }
    this.disbursementsByTreasury.clear();
    this.disbursementsByCategory.clear();

    // Clear expense state
    this.expenseRecords.clear();
    this.expenseCounter = 1;
    this.expensesByCategory.clear();
    this.expensesByTreasury.clear();

    this.stats = {
      totalLIPs: 0,
      lipsByStage: {},
      totalVotes: 0,
      totalStakeLocked: 0n,
      totalDomainProposals: 0,
      totalDisbursements: 0,
      totalExpenses: 0,
      totalDisbursed: 0n,
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

    // Deep clone domain proposals (Maps must not be shared)
    for (const [id, proposal] of this.domainProposals) {
      cloned.domainProposals.set(id, {
        ...proposal,
        endorsers: [...proposal.endorsers],
        endorserStakes: new Map(proposal.endorserStakes),
        challengers: [...proposal.challengers],
        challengerStakes: new Map(proposal.challengerStakes),
        challengerReasons: new Map(proposal.challengerReasons),
        examples: [...proposal.examples],
      });
    }

    // Deep clone domain tree (clone each DomainNode to prevent rollback corruption)
    const clonedDomainIndex = new Map<
      string,
      import("../core/knowledge/ontology/types.js").DomainNode
    >();
    for (const [domainId, node] of this.domainTree.domainIndex) {
      clonedDomainIndex.set(domainId, {
        domain: {
          ...node.domain,
          childDomains: [...node.domain.childDomains],
          dependsOnStrata: [...node.domain.dependsOnStrata],
          examples: [...node.domain.examples],
        },
        children: [], // Re-linked below
        depth: node.depth,
        path: [...node.path],
      });
    }
    // Re-link children from cloned nodes
    for (const [domainId, node] of this.domainTree.domainIndex) {
      const clonedNode = clonedDomainIndex.get(domainId)!;
      clonedNode.children = node.children
        .map((child) => clonedDomainIndex.get(child.domain.id)!)
        .filter(Boolean);
    }
    cloned.domainTree = {
      root:
        clonedDomainIndex.get(this.domainTree.root.domain.id) ??
        this.domainTree.root,
      strataIndex: new Map(
        Array.from(this.domainTree.strataIndex.entries()).map(([k, v]) => [
          k,
          [...v],
        ]),
      ),
      domainIndex: clonedDomainIndex,
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

    // Clone disbursements
    for (const [id, disb] of this.disbursementCache) {
      cloned.disbursementCache.set(id, {
        ...disb,
        stageHistory: [...disb.stageHistory],
      });
    }
    for (const [stage, ids] of this.disbursementsByStage) {
      cloned.disbursementsByStage.set(stage, new Set(ids));
    }
    for (const [fund, ids] of this.disbursementsByTreasury) {
      cloned.disbursementsByTreasury.set(fund, new Set(ids));
    }
    for (const [cat, ids] of this.disbursementsByCategory) {
      cloned.disbursementsByCategory.set(cat, new Set(ids));
    }
    cloned.disbursementCounter = this.disbursementCounter;

    // Clone expense records (immutable, shallow copy is sufficient)
    for (const [id, record] of this.expenseRecords) {
      cloned.expenseRecords.set(id, { ...record });
    }
    for (const [cat, ids] of this.expensesByCategory) {
      cloned.expensesByCategory.set(cat, new Set(ids));
    }
    for (const [fund, ids] of this.expensesByTreasury) {
      cloned.expensesByTreasury.set(fund, new Set(ids));
    }
    cloned.expenseCounter = this.expenseCounter;

    // Clone counter and stats
    cloned.lipCounter = this.lipCounter;
    cloned.stats = {
      ...this.stats,
      lipsByStage: { ...this.stats.lipsByStage },
    };

    return cloned;
  }
}
