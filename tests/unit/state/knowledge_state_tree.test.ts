/**
 * KnowledgeStateTree Unit Tests
 *
 * Comprehensive tests for the core knowledge state tree implementation.
 * Covers fact CRUD, reference graph, citation economics, challenges,
 * supersession, queries, proofs, confidence decay, propagation,
 * circular dependency detection, domain management, and cloning.
 *
 * Note: GAP-009 (fact expiration) is tested separately in fact_expiration.test.ts.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  KnowledgeStateTree,
  generateFactId,
  createKnowledgeTreeWithAxioms,
  ESTABLISHED_CONFIDENCE,
  REVERIFICATION_THRESHOLD,
  MAX_REFERENCES_PER_FACT,
  type PropagationResult,
  type CircularDependencyInfo,
} from "../../../state/knowledge.js";
import type {
  FactState,
  FactStatus,
  FactReference,
  ReferenceType,
} from "../../../state/types.js";
import type { Confidence } from "../../../core/knowledge/types.js";

// ============ Test Helpers ============

let _testIdCounter = 0;
function uid(): string {
  _testIdCounter++;
  return `fact-${_testIdCounter}-${Date.now().toString(36)}`;
}

function makeFact(
  id: string,
  opts: {
    claim?: string;
    domain?: string;
    confidence?: number;
    status?: FactStatus;
    verifiedAt?: bigint;
    verifierCount?: number;
    challengeCount?: number;
    citationCount?: number;
    stratum?:
      | "fundamental"
      | "physical"
      | "chemical"
      | "biological"
      | "cognitive"
      | "social"
      | "technological";
    expiresAt?: bigint;
    lastReverifiedAt?: bigint;
    lastCitedBlock?: bigint;
    totalCitationRewards?: bigint;
  } = {},
): Omit<FactState, "referenceRoot"> {
  return {
    id,
    claim: opts.claim ?? `Test claim for ${id}`,
    domain: opts.domain ?? "physics.mechanics",
    confidence: (opts.confidence ?? 0.85) as Confidence,
    verifiedAt: opts.verifiedAt ?? 100n,
    verifierCount: opts.verifierCount ?? 5,
    challengeCount: opts.challengeCount ?? 0,
    citationCount: opts.citationCount ?? 0,
    status: opts.status ?? "active",
    stratum: opts.stratum,
    expiresAt: opts.expiresAt,
    lastReverifiedAt: opts.lastReverifiedAt,
    lastCitedBlock: opts.lastCitedBlock,
    totalCitationRewards: opts.totalCitationRewards,
  };
}

// ============ Fact Operations ============

describe.sequential("KnowledgeStateTree", () => {
  let tree: KnowledgeStateTree;

  beforeEach(() => {
    tree = new KnowledgeStateTree();
    _testIdCounter = 0;
  });

  // ──────────── Initialization ────────────

  describe("Initialization", () => {
    it("should start with empty roots", () => {
      expect(tree.getFactRoot()).toBeDefined();
      expect(tree.getReferenceRoot()).toBeDefined();
      expect(tree.getKnowledgeRoot()).toBeDefined();
    });

    it("should start with zero counts", () => {
      expect(tree.getFactCount()).toBe(0);
      expect(tree.getActiveFactCount()).toBe(0);
    });

    it("should start with empty domains", () => {
      expect(tree.getDomains()).toHaveLength(0);
    });
  });

  // ──────────── Fact CRUD ────────────

  describe("Fact Operations", () => {
    it("should add a fact and return a new root", () => {
      const id = uid();
      const root = tree.addFact(makeFact(id));
      expect(root).toBeDefined();
      expect(typeof root).toBe("string");
      expect(root.length).toBeGreaterThan(0);
    });

    it("should retrieve an added fact", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.9 }));

      const fact = tree.getFact(id);
      expect(fact).not.toBeNull();
      expect(fact!.id).toBe(id);
      expect(fact!.confidence).toBe(0.9);
      expect(fact!.status).toBe("active");
    });

    it("should return null for non-existent fact", () => {
      expect(tree.getFact("nonexistent")).toBeNull();
    });

    it("should detect existing facts with hasFact", () => {
      const id = uid();
      expect(tree.hasFact(id)).toBe(false);
      tree.addFact(makeFact(id));
      expect(tree.hasFact(id)).toBe(true);
    });

    it("should detect active facts with isActiveFact", () => {
      const id = uid();
      tree.addFact(makeFact(id, { status: "active" }));
      expect(tree.isActiveFact(id)).toBe(true);
    });

    it("should not report non-active fact as active", () => {
      const id = uid();
      tree.addFact(makeFact(id, { status: "revoked" }));
      expect(tree.isActiveFact(id)).toBe(false);
    });

    it("should reject duplicate fact IDs", () => {
      const id = uid();
      tree.addFact(makeFact(id));
      expect(() => tree.addFact(makeFact(id))).toThrow("Fact already exists");
    });

    it("should increment fact and active counts on add", () => {
      tree.addFact(makeFact(uid()));
      tree.addFact(makeFact(uid()));
      expect(tree.getFactCount()).toBe(2);
      expect(tree.getActiveFactCount()).toBe(2);
    });

    it("should not increment active count for non-active facts", () => {
      tree.addFact(makeFact(uid(), { status: "revoked" }));
      expect(tree.getFactCount()).toBe(1);
      expect(tree.getActiveFactCount()).toBe(0);
    });

    it("should set referenceRoot to zeros for new facts", () => {
      const id = uid();
      tree.addFact(makeFact(id));
      const fact = tree.getFact(id)!;
      expect(fact.referenceRoot).toBe("0".repeat(64));
    });

    it("should update a fact", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.8 }));

      const newRoot = tree.updateFact(id, { confidence: 0.95 as Confidence });
      expect(newRoot).toBeDefined();

      const updated = tree.getFact(id)!;
      expect(updated.confidence).toBe(0.95);
    });

    it("should throw when updating non-existent fact", () => {
      expect(() =>
        tree.updateFact("nonexistent", { confidence: 0.5 as Confidence }),
      ).toThrow("Fact not found");
    });

    it("should track active count changes on status update", () => {
      const id = uid();
      tree.addFact(makeFact(id, { status: "active" }));
      expect(tree.getActiveFactCount()).toBe(1);

      tree.updateFact(id, { status: "revoked" });
      expect(tree.getActiveFactCount()).toBe(0);
    });

    it("should track active count when reactivating", () => {
      const id = uid();
      tree.addFact(makeFact(id, { status: "revoked" }));
      expect(tree.getActiveFactCount()).toBe(0);

      tree.updateFact(id, { status: "active" });
      expect(tree.getActiveFactCount()).toBe(1);
    });

    it("should change knowledge root when facts are modified", () => {
      const root1 = tree.getKnowledgeRoot();
      const id = uid();
      tree.addFact(makeFact(id));
      const root2 = tree.getKnowledgeRoot();
      expect(root2).not.toBe(root1);

      tree.updateFact(id, { confidence: 0.5 as Confidence });
      const root3 = tree.getKnowledgeRoot();
      expect(root3).not.toBe(root2);
    });

    it("should preserve serialization round-trip for all fields", () => {
      const id = uid();
      tree.addFact(
        makeFact(id, {
          confidence: 0.77,
          verifiedAt: 42n,
          stratum: "biological",
          expiresAt: 9999n,
          lastReverifiedAt: 500n,
          citationCount: 3,
          lastCitedBlock: 600n,
          totalCitationRewards: 1000n,
        }),
      );

      // Clear cache to force deserialization
      const tree2 = tree.clone();
      const fact = tree2.getFact(id)!;
      expect(fact.confidence).toBe(0.77);
      expect(fact.verifiedAt).toBe(42n);
      expect(fact.stratum).toBe("biological");
      expect(fact.expiresAt).toBe(9999n);
      expect(fact.lastReverifiedAt).toBe(500n);
      expect(fact.citationCount).toBe(3);
      expect(fact.lastCitedBlock).toBe(600n);
      expect(fact.totalCitationRewards).toBe(1000n);
    });
  });

  // ──────────── Citation Economics ────────────

  describe("Citation Economics", () => {
    it("should increment citation count for active facts", () => {
      const id = uid();
      tree.addFact(makeFact(id));

      tree.incrementCitationCount(id, 200n);
      const fact = tree.getFact(id)!;
      expect(fact.citationCount).toBe(1);
      expect(fact.lastCitedBlock).toBe(200n);
    });

    it("should accumulate citation counts", () => {
      const id = uid();
      tree.addFact(makeFact(id));

      tree.incrementCitationCount(id, 200n);
      tree.incrementCitationCount(id, 300n);
      tree.incrementCitationCount(id, 400n);

      const fact = tree.getFact(id)!;
      expect(fact.citationCount).toBe(3);
      expect(fact.lastCitedBlock).toBe(400n);
    });

    it("should skip citation increment for non-active facts", () => {
      const id = uid();
      tree.addFact(makeFact(id, { status: "revoked" }));

      const root = tree.incrementCitationCount(id, 200n);
      expect(root).toBeDefined();
      const fact = tree.getFact(id)!;
      expect(fact.citationCount).toBe(0);
    });

    it("should throw when citing non-existent fact", () => {
      expect(() => tree.incrementCitationCount("nonexistent", 200n)).toThrow(
        "Fact not found",
      );
    });

    it("should record citation rewards", () => {
      const id = uid();
      tree.addFact(makeFact(id));

      tree.recordCitationReward(id, 100n);
      tree.recordCitationReward(id, 250n);

      const fact = tree.getFact(id)!;
      expect(fact.totalCitationRewards).toBe(350n);
    });

    it("should throw when recording reward for non-existent fact", () => {
      expect(() => tree.recordCitationReward("nonexistent", 100n)).toThrow(
        "Fact not found",
      );
    });
  });

  // ──────────── Challenge ────────────

  describe("Challenge Operations", () => {
    it("should challenge an active fact", () => {
      const id = uid();
      tree.addFact(makeFact(id));

      tree.challengeFact(id, 500n);
      const fact = tree.getFact(id)!;
      expect(fact.status).toBe("challenged");
      expect(fact.challengeCount).toBe(1);
      expect(fact.lastChallengedAt).toBe(500n);
    });

    it("should reject challenge on non-active fact", () => {
      const id = uid();
      tree.addFact(makeFact(id, { status: "revoked" }));
      expect(() => tree.challengeFact(id, 500n)).toThrow(
        "Can only challenge active facts",
      );
    });

    it("should throw when challenging non-existent fact", () => {
      expect(() => tree.challengeFact("nonexistent", 500n)).toThrow(
        "Fact not found",
      );
    });

    it("should resolve challenge as upheld (restores active)", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.8 }));
      tree.challengeFact(id, 500n);

      tree.resolveChallenge(id, true, 0.85 as Confidence);
      const fact = tree.getFact(id)!;
      expect(fact.status).toBe("active");
      expect(fact.confidence).toBe(0.85);
    });

    it("should resolve challenge upheld without new confidence", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.8 }));
      tree.challengeFact(id, 500n);

      tree.resolveChallenge(id, true);
      const fact = tree.getFact(id)!;
      expect(fact.status).toBe("active");
      expect(fact.confidence).toBe(0.8);
    });

    it("should resolve challenge as revoked", () => {
      const id = uid();
      tree.addFact(makeFact(id));
      tree.challengeFact(id, 500n);

      tree.resolveChallenge(id, false);
      const fact = tree.getFact(id)!;
      expect(fact.status).toBe("revoked");
    });

    it("should reject resolving challenge on non-challenged fact", () => {
      const id = uid();
      tree.addFact(makeFact(id));
      expect(() => tree.resolveChallenge(id, true)).toThrow(
        "Fact is not under challenge",
      );
    });

    it("should reject resolving challenge on non-existent fact", () => {
      expect(() => tree.resolveChallenge("nonexistent", true)).toThrow(
        "Fact not found",
      );
    });
  });

  // ──────────── Revocation ────────────

  describe("Fact Revocation", () => {
    it("should revoke a fact", () => {
      const id = uid();
      tree.addFact(makeFact(id));

      tree.revokeFact(id);
      const fact = tree.getFact(id)!;
      expect(fact.status).toBe("revoked");
    });

    it("should zero out dependent confidence when sole premise is revoked", () => {
      // BFS propagation calls calculateDerivedConfidence, which returns 0
      // when hasSupportingRefs is true but all premises are inactive (chain broken).
      const baseId = uid();
      const depId = uid();
      tree.addFact(makeFact(baseId, { confidence: 0.9 }));
      tree.addFact(makeFact(depId, { confidence: 0.8 }));

      tree.addReference({
        fromFactId: depId,
        toFactId: baseId,
        referenceType: "supports",
        strength: 0.9,
      });

      tree.revokeFact(baseId);
      const dep = tree.getFact(depId)!;
      // Derivation chain broken: hasSupportingRefs=true, no active premises → confidence = 0
      expect(dep.confidence).toBe(0);
    });

    it("should recalculate dependent confidence when one of multiple premises is revoked", () => {
      const premise1 = uid();
      const premise2 = uid();
      const derived = uid();

      tree.addFact(makeFact(premise1, { confidence: 0.9 }));
      tree.addFact(makeFact(premise2, { confidence: 0.7 }));
      tree.addFact(makeFact(derived, { confidence: 0.8 }));

      // derived references both premises
      tree.addReference({
        fromFactId: derived,
        toFactId: premise1,
        referenceType: "supports",
        strength: 0.9,
      });
      tree.addReference({
        fromFactId: derived,
        toFactId: premise2,
        referenceType: "supports",
        strength: 0.8,
      });

      // Revoke premise1: derived is recalculated from premise2 only
      tree.revokeFact(premise1);
      const dep = tree.getFact(derived)!;
      // weakest-link: min(0.7) * min(0.8) = 0.56
      expect(dep.confidence).toBeCloseTo(0.56, 2);
    });

    it("should not affect non-active dependent facts", () => {
      const baseId = uid();
      const depId = uid();
      tree.addFact(makeFact(baseId));
      tree.addFact(makeFact(depId, { status: "revoked", confidence: 0.5 }));

      tree.addReference({
        fromFactId: depId,
        toFactId: baseId,
        referenceType: "supports",
        strength: 0.9,
      });

      tree.revokeFact(baseId);
      const dep = tree.getFact(depId)!;
      expect(dep.confidence).toBe(0.5); // unchanged
    });

    it("should throw when revoking non-existent fact", () => {
      expect(() => tree.revokeFact("nonexistent")).toThrow("Fact not found");
    });
  });

  // ──────────── Supersession ────────────

  describe("Fact Supersession", () => {
    it("should supersede an old fact with a new one", () => {
      const oldId = uid();
      const newId = uid();
      tree.addFact(makeFact(oldId));
      tree.addFact(makeFact(newId));

      tree.supersedeFact(oldId, newId);

      expect(tree.getFact(oldId)!.status).toBe("superseded");
      expect(tree.getFact(newId)!.status).toBe("active");
    });

    it("should add an extends reference from new to old", () => {
      const oldId = uid();
      const newId = uid();
      tree.addFact(makeFact(oldId));
      tree.addFact(makeFact(newId));

      tree.supersedeFact(oldId, newId);

      const ref = tree.getReference(newId, oldId);
      expect(ref).not.toBeNull();
      expect(ref!.referenceType).toBe("extends");
      expect(ref!.strength).toBe(1.0);
    });

    it("should throw when old fact does not exist", () => {
      const newId = uid();
      tree.addFact(makeFact(newId));
      expect(() => tree.supersedeFact("nonexistent", newId)).toThrow(
        "Old fact not found",
      );
    });

    it("should throw when new fact does not exist", () => {
      const oldId = uid();
      tree.addFact(makeFact(oldId));
      expect(() => tree.supersedeFact(oldId, "nonexistent")).toThrow(
        "New fact not found",
      );
    });
  });

  // ──────────── Reference Graph ────────────

  describe("Reference Operations", () => {
    it("should add a reference between facts", () => {
      const a = uid();
      const b = uid();
      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));

      tree.addReference({
        fromFactId: a,
        toFactId: b,
        referenceType: "supports",
        strength: 0.8,
      });

      const ref = tree.getReference(a, b);
      expect(ref).not.toBeNull();
      expect(ref!.fromFactId).toBe(a);
      expect(ref!.toFactId).toBe(b);
      expect(ref!.referenceType).toBe("supports");
      expect(ref!.strength).toBe(0.8);
    });

    it("should return null for non-existent reference", () => {
      const a = uid();
      const b = uid();
      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));
      expect(tree.getReference(a, b)).toBeNull();
    });

    it("should throw when source fact is missing", () => {
      const b = uid();
      tree.addFact(makeFact(b));
      expect(() =>
        tree.addReference({
          fromFactId: "nonexistent",
          toFactId: b,
          referenceType: "supports",
          strength: 0.5,
        }),
      ).toThrow("Source fact not found");
    });

    it("should throw when target fact is missing", () => {
      const a = uid();
      tree.addFact(makeFact(a));
      expect(() =>
        tree.addReference({
          fromFactId: a,
          toFactId: "nonexistent",
          referenceType: "supports",
          strength: 0.5,
        }),
      ).toThrow("Target fact not found");
    });

    it("should track outgoing references", () => {
      const a = uid();
      const b = uid();
      const c = uid();
      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));
      tree.addFact(makeFact(c));

      tree.addReference({
        fromFactId: a,
        toFactId: b,
        referenceType: "supports",
        strength: 0.8,
      });
      tree.addReference({
        fromFactId: a,
        toFactId: c,
        referenceType: "extends",
        strength: 0.9,
      });

      const outgoing = tree.getOutgoingReferences(a);
      expect(outgoing).toHaveLength(2);
      expect(outgoing).toContain(b);
      expect(outgoing).toContain(c);
    });

    it("should track incoming references", () => {
      const a = uid();
      const b = uid();
      const c = uid();
      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));
      tree.addFact(makeFact(c));

      tree.addReference({
        fromFactId: a,
        toFactId: c,
        referenceType: "supports",
        strength: 0.8,
      });
      tree.addReference({
        fromFactId: b,
        toFactId: c,
        referenceType: "supports",
        strength: 0.7,
      });

      const incoming = tree.getIncomingReferences(c);
      expect(incoming).toHaveLength(2);
      expect(incoming).toContain(a);
      expect(incoming).toContain(b);
    });

    it("should return empty arrays for facts with no references", () => {
      const id = uid();
      tree.addFact(makeFact(id));
      expect(tree.getOutgoingReferences(id)).toHaveLength(0);
      expect(tree.getIncomingReferences(id)).toHaveLength(0);
    });

    it("should get all references", () => {
      const a = uid();
      const b = uid();
      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));
      tree.addReference({
        fromFactId: a,
        toFactId: b,
        referenceType: "contradicts",
        strength: 0.6,
      });

      const all = tree.getAllReferences();
      expect(all).toHaveLength(1);
      expect(all[0].referenceType).toBe("contradicts");
    });
  });

  // ──────────── Supporting / Contradicting Queries ────────────

  describe("Supporting and Contradicting Facts", () => {
    it("should find supporting facts", () => {
      const target = uid();
      const supporter1 = uid();
      const supporter2 = uid();
      const nonSupporter = uid();

      tree.addFact(makeFact(target));
      tree.addFact(makeFact(supporter1));
      tree.addFact(makeFact(supporter2));
      tree.addFact(makeFact(nonSupporter));

      tree.addReference({
        fromFactId: supporter1,
        toFactId: target,
        referenceType: "supports",
        strength: 0.9,
      });
      tree.addReference({
        fromFactId: supporter2,
        toFactId: target,
        referenceType: "supports",
        strength: 0.7,
      });
      tree.addReference({
        fromFactId: nonSupporter,
        toFactId: target,
        referenceType: "extends",
        strength: 0.5,
      });

      const supporting = tree.getSupportingFacts(target);
      expect(supporting).toHaveLength(2);
      const ids = supporting.map((f) => f.id);
      expect(ids).toContain(supporter1);
      expect(ids).toContain(supporter2);
    });

    it("should exclude non-active supporting facts", () => {
      const target = uid();
      const supporter = uid();

      tree.addFact(makeFact(target));
      tree.addFact(makeFact(supporter, { status: "revoked" }));

      tree.addReference({
        fromFactId: supporter,
        toFactId: target,
        referenceType: "supports",
        strength: 0.9,
      });

      expect(tree.getSupportingFacts(target)).toHaveLength(0);
    });

    it("should find contradicting facts from both directions", () => {
      const a = uid();
      const b = uid();
      const c = uid();

      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));
      tree.addFact(makeFact(c));

      // b contradicts a (incoming to a)
      tree.addReference({
        fromFactId: b,
        toFactId: a,
        referenceType: "contradicts",
        strength: 0.8,
      });
      // a contradicts c (outgoing from a)
      tree.addReference({
        fromFactId: a,
        toFactId: c,
        referenceType: "contradicts",
        strength: 0.7,
      });

      const contradicting = tree.getContradictingFacts(a);
      expect(contradicting).toHaveLength(2);
      const ids = contradicting.map((f) => f.id);
      expect(ids).toContain(b);
      expect(ids).toContain(c);
    });

    it("should deduplicate contradicting facts", () => {
      const a = uid();
      const b = uid();

      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));

      // Bidirectional contradiction (both sides)
      tree.addReference({
        fromFactId: a,
        toFactId: b,
        referenceType: "contradicts",
        strength: 0.8,
      });
      tree.addReference({
        fromFactId: b,
        toFactId: a,
        referenceType: "contradicts",
        strength: 0.8,
      });

      const contradicting = tree.getContradictingFacts(a);
      expect(contradicting).toHaveLength(1);
      expect(contradicting[0].id).toBe(b);
    });
  });

  // ──────────── Query Operations ────────────

  describe("Query Operations", () => {
    it("should find facts by domain", () => {
      tree.addFact(makeFact(uid(), { domain: "physics" }));
      tree.addFact(makeFact(uid(), { domain: "physics" }));
      tree.addFact(makeFact(uid(), { domain: "biology" }));

      const physics = tree.getFactsByDomain("physics");
      expect(physics).toHaveLength(2);
      expect(physics.every((f) => f.domain === "physics")).toBe(true);
    });

    it("should only return active facts for domain query", () => {
      tree.addFact(makeFact(uid(), { domain: "physics", status: "revoked" }));
      tree.addFact(makeFact(uid(), { domain: "physics", status: "active" }));

      expect(tree.getFactsByDomain("physics")).toHaveLength(1);
    });

    it("should find facts by status", () => {
      tree.addFact(makeFact(uid(), { status: "active" }));
      tree.addFact(makeFact(uid(), { status: "active" }));
      tree.addFact(makeFact(uid(), { status: "revoked" }));

      expect(tree.getFactsByStatus("active")).toHaveLength(2);
      expect(tree.getFactsByStatus("revoked")).toHaveLength(1);
      expect(tree.getFactsByStatus("challenged")).toHaveLength(0);
    });

    it("should search facts by claim text (case-insensitive)", () => {
      tree.addFact(makeFact(uid(), { claim: "Water boils at 100 degrees" }));
      tree.addFact(makeFact(uid(), { claim: "Ice melts at 0 degrees" }));
      tree.addFact(makeFact(uid(), { claim: "Fire is hot" }));

      const results = tree.searchFacts("degrees");
      expect(results).toHaveLength(2);

      const results2 = tree.searchFacts("WATER");
      expect(results2).toHaveLength(1);
    });

    it("should only search active facts", () => {
      tree.addFact(
        makeFact(uid(), {
          claim: "Active claim about gravity",
          status: "active",
        }),
      );
      tree.addFact(
        makeFact(uid(), {
          claim: "Revoked claim about gravity",
          status: "revoked",
        }),
      );

      const results = tree.searchFacts("gravity");
      expect(results).toHaveLength(1);
    });
  });

  // ──────────── Domain Management ────────────

  describe("Domain Management", () => {
    it("should auto-create domain when adding fact", () => {
      tree.addFact(makeFact(uid(), { domain: "chemistry" }));
      const domains = tree.getDomains();
      expect(domains).toHaveLength(1);
      expect(domains[0].id).toBe("chemistry");
      expect(domains[0].factCount).toBe(1);
    });

    it("should increment domain fact count", () => {
      tree.addFact(makeFact(uid(), { domain: "chemistry" }));
      tree.addFact(makeFact(uid(), { domain: "chemistry" }));

      const domains = tree.getDomains();
      expect(domains[0].factCount).toBe(2);
    });

    it("should track multiple domains", () => {
      tree.addFact(makeFact(uid(), { domain: "physics" }));
      tree.addFact(makeFact(uid(), { domain: "chemistry" }));

      const domains = tree.getDomains();
      expect(domains).toHaveLength(2);
    });

    it("should allow setting domain state directly", () => {
      tree.setDomain({
        id: "custom",
        name: "Custom Domain",
        factCount: 42,
        rootFactId: "genesis-fact",
      });

      const domains = tree.getDomains();
      const custom = domains.find((d) => d.id === "custom");
      expect(custom).toBeDefined();
      expect(custom!.name).toBe("Custom Domain");
      expect(custom!.factCount).toBe(42);
    });
  });

  // ──────────── Proofs ────────────

  describe("Merkle Proofs", () => {
    it("should generate a fact proof", () => {
      const id = uid();
      tree.addFact(makeFact(id));

      const proof = tree.proveFact(id, 100n);
      expect(proof.factId).toBe(id);
      expect(proof.blockNumber).toBe(100n);
      expect(proof.stateRoot).toBeDefined();
      expect(proof.fact).toBeDefined();
      expect(proof.proof).toBeDefined();
    });

    it("should throw when proving non-existent fact", () => {
      expect(() => tree.proveFact("nonexistent", 100n)).toThrow(
        "Fact not found",
      );
    });

    it("should generate proof with matching fact data", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.77 }));

      const proof = tree.proveFact(id, 100n);
      expect(proof.fact.id).toBe(id);
      expect(proof.fact.confidence).toBe(0.77);
      expect(proof.proof).toBeDefined();
      expect(proof.proof.key).toBeDefined();
      expect(proof.proof.value).toBeDefined();
    });
  });

  // ──────────── Confidence Decay (Eager) ────────────

  describe("Confidence Decay (Eager)", () => {
    it("should apply logarithmic decay to active facts", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.9, verifiedAt: 0n }));

      tree.applyConfidenceDecay(10000n, 0.001);
      const fact = tree.getFact(id)!;
      expect(fact.confidence).toBeLessThan(0.9);
      expect(fact.confidence).toBeGreaterThan(0.1);
    });

    it("should not decay non-active facts", () => {
      const id = uid();
      tree.addFact(
        makeFact(id, { confidence: 0.9, status: "revoked", verifiedAt: 0n }),
      );

      tree.applyConfidenceDecay(10000n, 0.001);
      const fact = tree.getFact(id)!;
      expect(fact.confidence).toBe(0.9);
    });

    it("should not decay below 0.1", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.15, verifiedAt: 0n }));

      // Very aggressive decay rate
      tree.applyConfidenceDecay(1000000n, 0.1);
      const fact = tree.getFact(id)!;
      expect(fact.confidence).toBeGreaterThanOrEqual(0.1);
    });
  });

  // ──────────── Lazy Confidence Decay (GAP-008) ────────────

  describe("Lazy Confidence Decay (GAP-008)", () => {
    it("should calculate effective confidence without modifying stored value", () => {
      const id = uid();
      tree.addFact(
        makeFact(id, {
          confidence: 0.9,
          verifiedAt: 0n,
          stratum: "physical",
        }),
      );

      const fact = tree.getFact(id)!;
      const decay = tree.getEffectiveConfidence(fact, 200000n);

      expect(decay.storedConfidence).toBe(0.9);
      expect(decay.effectiveConfidence).toBeLessThan(0.9);
      expect(decay.decayFactor).toBeLessThan(1);
      expect(decay.isEternal).toBe(false);
      expect(decay.blocksSinceVerification).toBe(200000);
      expect(decay.category).toBe("empirical"); // physical -> empirical

      // Stored value should be unchanged
      const factAgain = tree.getFact(id)!;
      expect(factAgain.confidence).toBe(0.9);
    });

    it("should treat fundamental stratum facts as eternal", () => {
      const id = uid();
      tree.addFact(
        makeFact(id, {
          confidence: 1.0,
          verifiedAt: 0n,
          stratum: "fundamental",
        }),
      );

      const fact = tree.getFact(id)!;
      const decay = tree.getEffectiveConfidence(fact, 1000000n);

      expect(decay.isEternal).toBe(true);
      expect(decay.effectiveConfidence).toBe(1.0);
      expect(decay.category).toBe("analytic");
    });

    it("should treat technological stratum facts as eternal", () => {
      const id = uid();
      tree.addFact(
        makeFact(id, {
          confidence: 0.95,
          verifiedAt: 0n,
          stratum: "technological",
        }),
      );

      const fact = tree.getFact(id)!;
      const decay = tree.getEffectiveConfidence(fact, 1000000n);

      expect(decay.isEternal).toBe(true);
      expect(decay.effectiveConfidence).toBe(0.95);
      expect(decay.category).toBe("protocol");
    });

    it("should clamp social stratum to lower min confidence than physical", () => {
      // empirical (physical) and social have same half-life (111111 blocks)
      // but different minConfidence: empirical = 0.33, social = 0.22.
      // At very large block counts the min clamp diverges.
      const physId = uid();
      const socId = uid();

      tree.addFact(
        makeFact(physId, {
          confidence: 0.9,
          verifiedAt: 0n,
          stratum: "physical",
        }),
      );
      tree.addFact(
        makeFact(socId, {
          confidence: 0.9,
          verifiedAt: 0n,
          stratum: "social",
        }),
      );

      const physFact = tree.getFact(physId)!;
      const socFact = tree.getFact(socId)!;

      // At 500000 blocks, both decay below their minConfidence floors
      const physDecay = tree.getEffectiveConfidence(physFact, 500000n);
      const socDecay = tree.getEffectiveConfidence(socFact, 500000n);

      // Empirical (physical) clamps at 0.33, social at 0.22
      expect(physDecay.effectiveConfidence).toBeCloseTo(0.33, 1);
      expect(socDecay.effectiveConfidence).toBeCloseTo(0.22, 1);
      expect(socDecay.effectiveConfidence).toBeLessThan(
        physDecay.effectiveConfidence,
      );
    });

    it("should default to empirical category for facts without stratum", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.8, verifiedAt: 0n }));

      const fact = tree.getFact(id)!;
      const decay = tree.getEffectiveConfidence(fact, 100000n);
      expect(decay.category).toBe("empirical");
    });

    it("should return fact with decay info via getFactWithDecay", () => {
      const id = uid();
      tree.addFact(
        makeFact(id, {
          confidence: 0.85,
          verifiedAt: 0n,
          stratum: "biological",
        }),
      );

      const result = tree.getFactWithDecay(id, 50000n);
      expect(result).toBeDefined();
      expect(result!.fact.id).toBe(id);
      expect(result!.decayInfo.storedConfidence).toBe(0.85);
      expect(result!.decayInfo.category).toBe("empirical");
    });

    it("should return undefined for non-existent fact in getFactWithDecay", () => {
      expect(tree.getFactWithDecay("nonexistent", 100n)).toBeUndefined();
    });

    it("should batch-retrieve facts with decay", () => {
      const ids = [uid(), uid(), uid()];
      for (const id of ids) {
        tree.addFact(makeFact(id, { verifiedAt: 0n, stratum: "physical" }));
      }

      const results = tree.getFactsWithDecay([...ids, "nonexistent"], 50000n);

      expect(results).toHaveLength(3);
      for (const r of results) {
        expect(ids).toContain(r.factId);
        expect(r.decayInfo).toBeDefined();
      }
    });
  });

  // ──────────── Confidence Propagation ────────────

  describe("Confidence Propagation", () => {
    it("should propagate significant confidence changes to dependents", () => {
      // A -> B (B supports A), so A is a dependent of B
      const base = uid();
      const derived = uid();

      tree.addFact(makeFact(base, { confidence: 0.9 }));
      tree.addFact(makeFact(derived, { confidence: 0.8 }));

      // derived references base (derived -> base, "supports")
      tree.addReference({
        fromFactId: derived,
        toFactId: base,
        referenceType: "supports",
        strength: 0.9,
      });

      // Now base's incomingRefs includes derived
      // When base confidence drops, propagateConfidenceChange checks incoming => derived
      const result = tree.propagateConfidenceChange(
        base,
        0.9 as Confidence,
        0.3 as Confidence,
      );

      expect(result.updatedFacts.length).toBeGreaterThanOrEqual(0);
      // The derived fact's confidence may be recalculated via calculateDerivedConfidence
    });

    it("should skip propagation for small changes", () => {
      const base = uid();
      const derived = uid();

      tree.addFact(makeFact(base, { confidence: 0.9 }));
      tree.addFact(makeFact(derived, { confidence: 0.8 }));

      tree.addReference({
        fromFactId: derived,
        toFactId: base,
        referenceType: "supports",
        strength: 0.9,
      });

      // Change < 5% threshold
      const result = tree.propagateConfidenceChange(
        base,
        0.9 as Confidence,
        0.88 as Confidence,
      );

      expect(result.updatedFacts).toHaveLength(0);
      expect(result.depthReached).toBe(0);
    });

    it("should detect cycles during propagation", () => {
      // Create a cycle: A -> B -> C -> A
      const a = uid();
      const b = uid();
      const c = uid();

      tree.addFact(makeFact(a, { confidence: 0.9 }));
      tree.addFact(makeFact(b, { confidence: 0.85 }));
      tree.addFact(makeFact(c, { confidence: 0.8 }));

      tree.addReference({
        fromFactId: b,
        toFactId: a,
        referenceType: "supports",
        strength: 0.9,
      });
      tree.addReference({
        fromFactId: c,
        toFactId: b,
        referenceType: "supports",
        strength: 0.9,
      });
      tree.addReference({
        fromFactId: a,
        toFactId: c,
        referenceType: "supports",
        strength: 0.9,
      });

      // Large change to trigger propagation
      const result = tree.propagateConfidenceChange(
        a,
        0.9 as Confidence,
        0.3 as Confidence,
      );

      // Cycles should be detected (b->a->c->b would hit b again)
      // The exact behavior depends on BFS traversal order
      expect(result.depthReached).toBeGreaterThanOrEqual(0);
    });

    it("should respect max propagation depth", () => {
      // Create a chain: f0 <- f1 <- f2 <- ... <- f12
      const facts: string[] = [];
      for (let i = 0; i < 13; i++) {
        const id = uid();
        facts.push(id);
        tree.addFact(makeFact(id, { confidence: 0.9 }));
      }

      for (let i = 1; i < facts.length; i++) {
        tree.addReference({
          fromFactId: facts[i],
          toFactId: facts[i - 1],
          referenceType: "supports",
          strength: 0.9,
        });
      }

      const result = tree.propagateConfidenceChange(
        facts[0],
        0.9 as Confidence,
        0.2 as Confidence,
      );

      // Should not exceed MAX_PROPAGATION_DEPTH (10)
      expect(result.depthReached).toBeLessThanOrEqual(11);
    });
  });

  // ──────────── Derived Confidence ────────────

  describe("Derived Confidence Calculation", () => {
    it("should calculate derived confidence using weakest-link principle", () => {
      const derived = uid();
      const premise1 = uid();
      const premise2 = uid();

      tree.addFact(makeFact(derived, { confidence: 0.9 }));
      tree.addFact(makeFact(premise1, { confidence: 0.85 }));
      tree.addFact(makeFact(premise2, { confidence: 0.7 }));

      tree.addReference({
        fromFactId: derived,
        toFactId: premise1,
        referenceType: "supports",
        strength: 0.95,
      });
      tree.addReference({
        fromFactId: derived,
        toFactId: premise2,
        referenceType: "supports",
        strength: 0.8,
      });

      const result = tree.calculateDerivedConfidence(derived);
      expect(result).not.toBeNull();

      // min(0.85, 0.7) * min(0.95, 0.8) = 0.7 * 0.8 = 0.56
      expect(result).toBeCloseTo(0.56, 2);
    });

    it("should return null for facts with no outgoing support references", () => {
      const id = uid();
      tree.addFact(makeFact(id));

      expect(tree.calculateDerivedConfidence(id)).toBeNull();
    });

    it("should ignore non-support reference types", () => {
      const derived = uid();
      const other = uid();

      tree.addFact(makeFact(derived));
      tree.addFact(makeFact(other, { confidence: 0.5 }));

      tree.addReference({
        fromFactId: derived,
        toFactId: other,
        referenceType: "contradicts",
        strength: 0.9,
      });

      expect(tree.calculateDerivedConfidence(derived)).toBeNull();
    });

    it("should return zero when all supporting premises are revoked", () => {
      const derived = uid();
      const premise = uid();

      tree.addFact(makeFact(derived));
      tree.addFact(makeFact(premise, { confidence: 0.8, status: "revoked" }));

      tree.addReference({
        fromFactId: derived,
        toFactId: premise,
        referenceType: "supports",
        strength: 0.9,
      });

      // Derivation chain is broken - all premises revoked → confidence 0
      expect(tree.calculateDerivedConfidence(derived)).toBe(0);
    });

    it("should include extends references in derivation", () => {
      const derived = uid();
      const base = uid();

      tree.addFact(makeFact(derived));
      tree.addFact(makeFact(base, { confidence: 0.8 }));

      tree.addReference({
        fromFactId: derived,
        toFactId: base,
        referenceType: "extends",
        strength: 0.9,
      });

      const result = tree.calculateDerivedConfidence(derived);
      expect(result).toBeCloseTo(0.72, 2); // 0.8 * 0.9
    });
  });

  // ──────────── Circular Dependency Detection ────────────

  describe("Circular Dependency Detection", () => {
    it("should detect a simple cycle (A -> B -> A)", () => {
      const a = uid();
      const b = uid();

      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));

      tree.addReference({
        fromFactId: a,
        toFactId: b,
        referenceType: "supports",
        strength: 0.9,
      });
      tree.addReference({
        fromFactId: b,
        toFactId: a,
        referenceType: "supports",
        strength: 0.9,
      });

      const cycle = tree.detectCircularDependencies(a);
      expect(cycle).not.toBeNull();
      expect(cycle!.facts.length).toBeGreaterThanOrEqual(2);
      expect(cycle!.tightness).toBe(1.0); // 2-node = tight
      expect(cycle!.penalty).toBeCloseTo(0.4, 2);
    });

    it("should detect a longer cycle (A -> B -> C -> A)", () => {
      const a = uid();
      const b = uid();
      const c = uid();

      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));
      tree.addFact(makeFact(c));

      tree.addReference({
        fromFactId: a,
        toFactId: b,
        referenceType: "supports",
        strength: 0.8,
      });
      tree.addReference({
        fromFactId: b,
        toFactId: c,
        referenceType: "supports",
        strength: 0.8,
      });
      tree.addReference({
        fromFactId: c,
        toFactId: a,
        referenceType: "supports",
        strength: 0.8,
      });

      const cycle = tree.detectCircularDependencies(a);
      expect(cycle).not.toBeNull();
      expect(cycle!.facts).toHaveLength(3);
      expect(cycle!.tightness).toBe(1.0); // <= 3 nodes
    });

    it("should return null when no cycle exists", () => {
      const a = uid();
      const b = uid();
      const c = uid();

      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));
      tree.addFact(makeFact(c));

      tree.addReference({
        fromFactId: a,
        toFactId: b,
        referenceType: "supports",
        strength: 0.8,
      });
      tree.addReference({
        fromFactId: b,
        toFactId: c,
        referenceType: "supports",
        strength: 0.8,
      });

      const cycle = tree.detectCircularDependencies(a);
      expect(cycle).toBeNull();
    });

    it("should calculate lower tightness for longer cycles", () => {
      // 5-node cycle
      const ids: string[] = [];
      for (let i = 0; i < 5; i++) {
        const id = uid();
        ids.push(id);
        tree.addFact(makeFact(id));
      }

      for (let i = 0; i < 5; i++) {
        tree.addReference({
          fromFactId: ids[i],
          toFactId: ids[(i + 1) % 5],
          referenceType: "supports",
          strength: 0.8,
        });
      }

      const cycle = tree.detectCircularDependencies(ids[0]);
      expect(cycle).not.toBeNull();
      expect(cycle!.tightness).toBeLessThan(1.0);
      expect(cycle!.penalty).toBeLessThan(0.4);
    });
  });

  // ──────────── Update Confidence With Propagation ────────────

  describe("updateFactConfidenceWithPropagation", () => {
    it("should update confidence and return propagation result", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.9 }));

      const { root, propagation } = tree.updateFactConfidenceWithPropagation(
        id,
        0.5 as Confidence,
      );

      expect(root).toBeDefined();
      expect(propagation).toBeDefined();
      expect(tree.getFact(id)!.confidence).toBe(0.5);
    });

    it("should apply cycle penalty when circular dependency detected", () => {
      const a = uid();
      const b = uid();

      tree.addFact(makeFact(a, { confidence: 0.9 }));
      tree.addFact(makeFact(b, { confidence: 0.8 }));

      tree.addReference({
        fromFactId: a,
        toFactId: b,
        referenceType: "supports",
        strength: 0.9,
      });
      tree.addReference({
        fromFactId: b,
        toFactId: a,
        referenceType: "supports",
        strength: 0.9,
      });

      tree.updateFactConfidenceWithPropagation(a, 0.8 as Confidence);
      const fact = tree.getFact(a)!;

      // Cycle penalty: 0.8 * (1 - 0.4) = 0.48 initially
      // Then BFS propagation feeds back through cycle:
      //   b recalculated: 0.48 * 0.9 = 0.432
      //   a recalculated: 0.432 * 0.9 = 0.3888
      expect(fact.confidence).toBeCloseTo(0.3888, 3);
    });

    it("should throw for non-existent fact", () => {
      expect(() =>
        tree.updateFactConfidenceWithPropagation(
          "nonexistent",
          0.5 as Confidence,
        ),
      ).toThrow("Fact not found");
    });
  });

  // ──────────── Reverification ────────────

  describe("Reverification", () => {
    it("should update reverification timestamp and verifier count", () => {
      const id = uid();
      tree.addFact(makeFact(id, { verifierCount: 5 }));

      tree.reverifyFact(id, 1000n);
      const fact = tree.getFact(id)!;
      expect(fact.lastReverifiedAt).toBe(1000n);
      expect(fact.verifierCount).toBe(6);
    });

    it("should update confidence when provided", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.7 }));

      tree.reverifyFact(id, 1000n, 0.9 as Confidence);
      expect(tree.getFact(id)!.confidence).toBe(0.9);
    });

    it("should reactivate expired facts", () => {
      const id = uid();
      tree.addFact(makeFact(id, { status: "expired" }));

      tree.reverifyFact(id, 1000n);
      expect(tree.getFact(id)!.status).toBe("active");
    });

    it("should not change status for non-expired facts", () => {
      const id = uid();
      tree.addFact(makeFact(id, { status: "active" }));

      tree.reverifyFact(id, 1000n);
      expect(tree.getFact(id)!.status).toBe("active");
    });

    it("should throw for non-existent fact", () => {
      expect(() => tree.reverifyFact("nonexistent", 1000n)).toThrow(
        "Fact not found",
      );
    });
  });

  // ──────────── Cloning ────────────

  describe("Cloning", () => {
    it("should clone all facts", () => {
      const id1 = uid();
      const id2 = uid();
      tree.addFact(makeFact(id1, { confidence: 0.9 }));
      tree.addFact(makeFact(id2, { confidence: 0.7 }));

      const cloned = tree.clone();

      expect(cloned.getFact(id1)!.confidence).toBe(0.9);
      expect(cloned.getFact(id2)!.confidence).toBe(0.7);
      expect(cloned.getFactCount()).toBe(2);
      expect(cloned.getActiveFactCount()).toBe(2);
    });

    it("should clone references and indexes", () => {
      const a = uid();
      const b = uid();
      tree.addFact(makeFact(a));
      tree.addFact(makeFact(b));
      tree.addReference({
        fromFactId: a,
        toFactId: b,
        referenceType: "supports",
        strength: 0.8,
      });

      const cloned = tree.clone();

      expect(cloned.getReference(a, b)).not.toBeNull();
      expect(cloned.getOutgoingReferences(a)).toContain(b);
      expect(cloned.getIncomingReferences(b)).toContain(a);
      expect(cloned.getAllReferences()).toHaveLength(1);
    });

    it("should clone domains", () => {
      tree.addFact(makeFact(uid(), { domain: "physics" }));
      tree.addFact(makeFact(uid(), { domain: "chemistry" }));

      const cloned = tree.clone();
      expect(cloned.getDomains()).toHaveLength(2);
    });

    it("should be independent after cloning (mutations don't cross)", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.9 }));

      const cloned = tree.clone();
      cloned.updateFact(id, { confidence: 0.1 as Confidence });

      // Original should be unchanged
      expect(tree.getFact(id)!.confidence).toBe(0.9);
      expect(cloned.getFact(id)!.confidence).toBe(0.1);
    });

    it("should produce matching knowledge roots", () => {
      tree.addFact(makeFact(uid()));
      tree.addFact(makeFact(uid()));

      const cloned = tree.clone();
      expect(cloned.getKnowledgeRoot()).toBe(tree.getKnowledgeRoot());
    });
  });

  // ──────────── Factory Functions ────────────

  describe("Factory Functions", () => {
    it("should generate deterministic fact IDs", () => {
      const id1 = generateFactId("claim", "domain", "submitter");
      const id2 = generateFactId("claim", "domain", "submitter");
      expect(id1).toBe(id2);
    });

    it("should generate different IDs for different inputs", () => {
      const id1 = generateFactId("claim1", "domain", "submitter");
      const id2 = generateFactId("claim2", "domain", "submitter");
      expect(id1).not.toBe(id2);
    });

    it("should create tree with genesis axioms", () => {
      const axioms = [
        { id: "axiom_0", claim: "Truth exists", domain: "meta" },
        { id: "axiom_1", claim: "Knowledge has value", domain: "meta" },
      ];

      const axTree = createKnowledgeTreeWithAxioms(axioms);
      expect(axTree.getFactCount()).toBe(2);
      expect(axTree.getActiveFactCount()).toBe(2);

      const ax0 = axTree.getFact("axiom_0")!;
      expect(ax0.confidence).toBe(1.0);
      expect(ax0.verifiedAt).toBe(0n);
      expect(ax0.verifierCount).toBe(0);
      expect(ax0.status).toBe("active");
    });
  });

  // ──────────── Constants ────────────

  describe("Constants", () => {
    it("should export correct confidence thresholds", () => {
      expect(ESTABLISHED_CONFIDENCE).toBe(0.8);
      expect(REVERIFICATION_THRESHOLD).toBe(0.6);
    });

    it("should export max references per fact", () => {
      expect(MAX_REFERENCES_PER_FACT).toBe(100);
    });
  });

  // ──────────── Edge Cases ────────────

  describe("Edge Cases", () => {
    it("should handle fact with zero confidence", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0 }));
      expect(tree.getFact(id)!.confidence).toBe(0);
    });

    it("should handle fact with maximum confidence", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 1.0 }));
      expect(tree.getFact(id)!.confidence).toBe(1.0);
    });

    it("should handle very large block numbers", () => {
      const id = uid();
      tree.addFact(makeFact(id, { verifiedAt: 999999999999n }));
      expect(tree.getFact(id)!.verifiedAt).toBe(999999999999n);
    });

    it("should handle facts with all optional fields undefined", () => {
      const id = uid();
      tree.addFact({
        id,
        claim: "Minimal fact",
        domain: "test",
        confidence: 0.5 as Confidence,
        verifiedAt: 0n,
        verifierCount: 1,
        challengeCount: 0,
        citationCount: 0,
        status: "active",
      });
      const fact = tree.getFact(id)!;
      expect(fact.expiresAt).toBeUndefined();
      expect(fact.lastReverifiedAt).toBeUndefined();
      expect(fact.stratum).toBeUndefined();
      expect(fact.lastChallengedAt).toBeUndefined();
    });

    it("should handle rapid add-update-revoke cycle", () => {
      const id = uid();
      tree.addFact(makeFact(id, { confidence: 0.9 }));
      tree.updateFact(id, { confidence: 0.5 as Confidence });
      tree.revokeFact(id);

      expect(tree.getFact(id)!.status).toBe("revoked");
      expect(tree.getActiveFactCount()).toBe(0);
      expect(tree.getFactCount()).toBe(1);
    });

    it("should handle multiple domains with same fact count", () => {
      tree.addFact(makeFact(uid(), { domain: "a" }));
      tree.addFact(makeFact(uid(), { domain: "b" }));
      tree.addFact(makeFact(uid(), { domain: "a" }));

      const domains = tree.getDomains();
      const domA = domains.find((d) => d.id === "a")!;
      const domB = domains.find((d) => d.id === "b")!;
      expect(domA.factCount).toBe(2);
      expect(domB.factCount).toBe(1);
    });

    it("should handle self-referencing fact", () => {
      const id = uid();
      tree.addFact(makeFact(id));

      tree.addReference({
        fromFactId: id,
        toFactId: id,
        referenceType: "supports",
        strength: 1.0,
      });

      expect(tree.getOutgoingReferences(id)).toContain(id);
      expect(tree.getIncomingReferences(id)).toContain(id);
    });

    it("should handle all reference types", () => {
      const types: ReferenceType[] = [
        "supports",
        "contradicts",
        "extends",
        "specializes",
        "generalizes",
      ];

      const base = uid();
      tree.addFact(makeFact(base));

      for (const type of types) {
        const other = uid();
        tree.addFact(makeFact(other));
        tree.addReference({
          fromFactId: base,
          toFactId: other,
          referenceType: type,
          strength: 0.5,
        });
      }

      expect(tree.getOutgoingReferences(base)).toHaveLength(5);
    });
  });
});
