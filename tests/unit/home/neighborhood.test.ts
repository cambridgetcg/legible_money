/**
 * NeighborhoodManager Unit Tests
 *
 * Tests for the Neighborhood System - domain-anchored clusters of Agent Homes.
 * Covers founding, membership, contributions, defense, dissolution, queries, and export.
 *
 * PP HIERARCHY:
 *   TRUTH: Tests verify neighborhood state transitions are deterministic and correct
 *   UNDERSTANDING: Clear test structure mirrors neighborhood lifecycle
 *   BEAUTY: Comprehensive coverage with elegant test organization
 *   JUSTICE: Tests placed in appropriate unit test location alongside other home tests
 */

import { describe, it, expect } from "vitest";
import {
  NeighborhoodManager,
  MAX_NEIGHBORHOOD_SIZE,
  MIN_ACTIVE_MEMBERS,
  DISSOLUTION_THRESHOLD_BLOCKS,
  ELDER_TENURE_BLOCKS,
  DEFAULT_BENEFITS,
  type Neighborhood,
  type NeighborEntry,
  type NeighborRelation,
  type NeighborhoodBenefits,
} from "../../../core/agents/home/neighborhood.js";

// Test DIDs
const AGENT_001 = "did:lgm:agent-001";
const AGENT_002 = "did:lgm:agent-002";
const AGENT_003 = "did:lgm:agent-003";
const AGENT_004 = "did:lgm:agent-004";
const AGENT_005 = "did:lgm:agent-005";

// Helper to create a fresh manager
function createManager(): NeighborhoodManager {
  return new NeighborhoodManager();
}

// Helper to found a neighborhood and optionally add members
function foundWithMembers(
  manager: NeighborhoodManager,
  founder: string,
  domain: string,
  members: string[] = [],
  block: bigint = 100n,
): string {
  manager.setCurrentBlock(block);
  const id = manager.found(founder, domain);
  if (!id) throw new Error("Failed to found neighborhood");
  for (const m of members) {
    manager.join(id, m);
  }
  return id;
}

describe.sequential("NeighborhoodManager", () => {
  // ============ Constants ============

  describe("Constants", () => {
    it("should have correct MAX_NEIGHBORHOOD_SIZE", () => {
      expect(MAX_NEIGHBORHOOD_SIZE).toBe(22);
    });

    it("should have correct MIN_ACTIVE_MEMBERS", () => {
      expect(MIN_ACTIVE_MEMBERS).toBe(3);
    });

    it("should have correct DISSOLUTION_THRESHOLD_BLOCKS", () => {
      expect(DISSOLUTION_THRESHOLD_BLOCKS).toBe(34_272n);
    });

    it("should have correct ELDER_TENURE_BLOCKS", () => {
      expect(ELDER_TENURE_BLOCKS).toBe(100_000n);
    });

    it("should have correct DEFAULT_BENEFITS", () => {
      expect(DEFAULT_BENEFITS.collaborationBonusBps).toBe(500);
      expect(DEFAULT_BENEFITS.sharedDefense).toBe(true);
      expect(DEFAULT_BENEFITS.freeVoice).toBe(true);
      expect(DEFAULT_BENEFITS.reputationWitness).toBe(true);
      expect(DEFAULT_BENEFITS.elderThresholdBlocks).toBe(100_000n);
    });
  });

  // ============ Constructor ============

  describe("Constructor", () => {
    it("should create manager with empty state", () => {
      const manager = createManager();

      expect(manager.getActiveNeighborhoods()).toHaveLength(0);
      expect(manager.getRelations()).toHaveLength(0);
      expect(manager.getAllNeighborDids()).toHaveLength(0);
    });

    it("should create manager with existing neighborhoods", () => {
      const existing: Neighborhood = {
        id: "nbh:physics:agent-001:100",
        domain: "physics",
        foundedAtBlock: 100n,
        founder: AGENT_001,
        members: [
          {
            did: AGENT_001,
            joinedAt: 100n,
            role: "founder",
            contributions: 0,
            defensePoolContribution: 0n,
            standing: "good",
            lastActiveBlock: 100n,
          },
          {
            did: AGENT_002,
            joinedAt: 110n,
            role: "member",
            contributions: 0,
            defensePoolContribution: 0n,
            standing: "good",
            lastActiveBlock: 110n,
          },
          {
            did: AGENT_003,
            joinedAt: 120n,
            role: "member",
            contributions: 0,
            defensePoolContribution: 0n,
            standing: "good",
            lastActiveBlock: 120n,
          },
        ],
        maxMembers: MAX_NEIGHBORHOOD_SIZE,
        sharedDefensePool: 0n,
        collaborationBonusBps: 500,
        status: "active",
        stats: {
          totalVerifications: 5,
          totalRewardsEarned: 0n,
          defensesCoordinated: 1,
          averageMemberAge: 0n,
          factsContributed: 5,
        },
      };

      const manager = new NeighborhoodManager([existing]);
      const n = manager.getNeighborhood("nbh:physics:agent-001:100");

      expect(n).toBeDefined();
      expect(n!.domain).toBe("physics");
      expect(n!.members).toHaveLength(3);
      expect(n!.stats.totalVerifications).toBe(5);
    });

    it("should create manager with existing relations", () => {
      const existingRelations: NeighborRelation[] = [
        {
          neighborDid: AGENT_002,
          neighborhoodId: "nbh:physics:agent-001:100",
          proximity: 0.5,
          sharedFacts: 3,
          voiceDiscount: true,
        },
      ];

      const manager = new NeighborhoodManager(undefined, existingRelations);
      const relations = manager.getRelations();

      expect(relations).toHaveLength(1);
      expect(relations[0].neighborDid).toBe(AGENT_002);
    });

    it("should create manager with both neighborhoods and relations", () => {
      const existing: Neighborhood = {
        id: "nbh:math:agent-001:50",
        domain: "math",
        foundedAtBlock: 50n,
        founder: AGENT_001,
        members: [
          {
            did: AGENT_001,
            joinedAt: 50n,
            role: "founder",
            contributions: 0,
            defensePoolContribution: 0n,
            standing: "good",
            lastActiveBlock: 50n,
          },
        ],
        maxMembers: MAX_NEIGHBORHOOD_SIZE,
        sharedDefensePool: 0n,
        collaborationBonusBps: 500,
        status: "forming",
        stats: {
          totalVerifications: 0,
          totalRewardsEarned: 0n,
          defensesCoordinated: 0,
          averageMemberAge: 0n,
          factsContributed: 0,
        },
      };

      const existingRelations: NeighborRelation[] = [
        {
          neighborDid: AGENT_002,
          neighborhoodId: "nbh:math:agent-001:50",
          proximity: 0.1,
          sharedFacts: 0,
          voiceDiscount: true,
        },
      ];

      const manager = new NeighborhoodManager([existing], existingRelations);

      expect(manager.getNeighborhood("nbh:math:agent-001:50")).toBeDefined();
      expect(manager.getRelations()).toHaveLength(1);
    });
  });

  // ============ found() ============

  describe("found()", () => {
    it("should create neighborhood with founder as first member", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics.quantum");

      expect(id).toBeDefined();

      const n = manager.getNeighborhood(id!);
      expect(n).toBeDefined();
      expect(n!.founder).toBe(AGENT_001);
      expect(n!.members).toHaveLength(1);
      expect(n!.members[0].did).toBe(AGENT_001);
      expect(n!.members[0].role).toBe("founder");
      expect(n!.members[0].joinedAt).toBe(100n);
      expect(n!.members[0].contributions).toBe(0);
      expect(n!.members[0].defensePoolContribution).toBe(0n);
      expect(n!.members[0].standing).toBe("good");
      expect(n!.members[0].lastActiveBlock).toBe(100n);
    });

    it("should return neighborhood ID", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics.quantum");

      expect(id).toBeTruthy();
      expect(typeof id).toBe("string");
      // ID format: nbh:{domain}:{last 8 chars of DID}:{block}
      expect(id).toContain("physics.quantum");
      expect(id).toContain("100");
    });

    it("should set status to 'forming' with only 1 member", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics.quantum");

      const n = manager.getNeighborhood(id!);
      expect(n!.status).toBe("forming");
    });

    it("should prevent duplicate founding (same domain + founder)", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id1 = manager.found(AGENT_001, "physics.quantum");
      const id2 = manager.found(AGENT_001, "physics.quantum");

      expect(id1).toBeDefined();
      expect(id2).toBeUndefined();
    });

    it("should allow different founders for the same domain", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id1 = manager.found(AGENT_001, "physics.quantum");
      const id2 = manager.found(AGENT_002, "physics.quantum");

      expect(id1).toBeDefined();
      expect(id2).toBeDefined();
      expect(id1).not.toBe(id2);
    });

    it("should allow same founder for different domains", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id1 = manager.found(AGENT_001, "physics.quantum");
      const id2 = manager.found(AGENT_001, "math.algebra");

      expect(id1).toBeDefined();
      expect(id2).toBeDefined();
      expect(id1).not.toBe(id2);
    });

    it("should set correct defaults (maxMembers=22, collaborationBonusBps=500)", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics.quantum");

      const n = manager.getNeighborhood(id!);
      expect(n!.maxMembers).toBe(22);
      expect(n!.collaborationBonusBps).toBe(500);
      expect(n!.sharedDefensePool).toBe(0n);
    });

    it("should initialize stats to zero", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics.quantum");

      const n = manager.getNeighborhood(id!);
      expect(n!.stats.totalVerifications).toBe(0);
      expect(n!.stats.totalRewardsEarned).toBe(0n);
      expect(n!.stats.defensesCoordinated).toBe(0);
      expect(n!.stats.averageMemberAge).toBe(0n);
      expect(n!.stats.factsContributed).toBe(0);
    });

    it("should set the foundedAtBlock from currentBlock", () => {
      const manager = createManager();
      manager.setCurrentBlock(42n);
      const id = manager.found(AGENT_001, "physics.quantum");

      const n = manager.getNeighborhood(id!);
      expect(n!.foundedAtBlock).toBe(42n);
    });

    it("should store optional name", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics.quantum", "Quantum Crew");

      const n = manager.getNeighborhood(id!);
      expect(n!.name).toBe("Quantum Crew");
    });

    it("should allow re-founding in a domain after previous was dissolved", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id1 = manager.found(AGENT_001, "physics.quantum");

      // Manually dissolve: leave all members, advance time, check dissolution
      const n = manager.getNeighborhood(id1!);
      // The founder is the only member; if we set status to dissolved directly
      // we need to go through the dissolution flow
      n!.status = "dissolved";

      // Now founding again should succeed because the old one is dissolved
      manager.setCurrentBlock(200n);
      const id2 = manager.found(AGENT_001, "physics.quantum");
      expect(id2).toBeDefined();
    });
  });

  // ============ join() ============

  describe("join()", () => {
    it("should successfully add member", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics");

      const result = manager.join(id, AGENT_002);
      expect(result).toBe(true);

      const n = manager.getNeighborhood(id);
      expect(n!.members).toHaveLength(2);
      expect(n!.members[1].did).toBe(AGENT_002);
      expect(n!.members[1].role).toBe("member");
      expect(n!.members[1].standing).toBe("good");
    });

    it("should return false for unknown neighborhood", () => {
      const manager = createManager();
      const result = manager.join("non-existent-id", AGENT_002);
      expect(result).toBe(false);
    });

    it("should return false for dissolved neighborhood", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics");

      // Force dissolution
      const n = manager.getNeighborhood(id);
      n!.status = "dissolved";

      const result = manager.join(id, AGENT_002);
      expect(result).toBe(false);
    });

    it("should return false when at max capacity", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");

      // Fill to capacity (founder already in, need maxMembers - 1 more)
      for (let i = 2; i <= MAX_NEIGHBORHOOD_SIZE; i++) {
        const did = `did:lgm:agent-${String(i).padStart(3, "0")}`;
        manager.join(id, did);
      }

      const n = manager.getNeighborhood(id);
      expect(n!.members).toHaveLength(MAX_NEIGHBORHOOD_SIZE);

      // Next join should fail
      const result = manager.join(id, "did:lgm:agent-extra");
      expect(result).toBe(false);
      expect(n!.members).toHaveLength(MAX_NEIGHBORHOOD_SIZE);
    });

    it("should return false if already a member", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics");

      manager.join(id, AGENT_002);
      const result = manager.join(id, AGENT_002);
      expect(result).toBe(false);

      const n = manager.getNeighborhood(id);
      expect(n!.members).toHaveLength(2);
    });

    it("should return false if founder tries to join again", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics");

      const result = manager.join(id, AGENT_001);
      expect(result).toBe(false);
    });

    it("should transition to 'active' at 3+ members", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics");

      const n = manager.getNeighborhood(id);
      expect(n!.status).toBe("forming");

      manager.join(id, AGENT_002);
      expect(n!.status).toBe("forming");

      manager.join(id, AGENT_003);
      expect(n!.status).toBe("active");
    });

    it("should build neighbor relations on join", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics");

      // After founding, only founder, no pairwise relations
      expect(manager.getRelations()).toHaveLength(0);

      // Add second member: 1 pair = 2 relations (A->B, B->A)
      manager.join(id, AGENT_002);
      expect(manager.getRelations()).toHaveLength(2);

      // Add third member: 3 pairs = 6 relations (A->B, B->A, A->C, C->A, B->C, C->B)
      manager.join(id, AGENT_003);
      expect(manager.getRelations()).toHaveLength(6);
    });

    it("should set joined member defaults correctly", () => {
      const manager = createManager();
      manager.setCurrentBlock(200n);
      const id = foundWithMembers(manager, AGENT_001, "physics", [], 200n);

      manager.setCurrentBlock(300n);
      manager.join(id, AGENT_002);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member).toBeDefined();
      expect(member!.joinedAt).toBe(300n);
      expect(member!.role).toBe("member");
      expect(member!.contributions).toBe(0);
      expect(member!.defensePoolContribution).toBe(0n);
      expect(member!.standing).toBe("good");
      expect(member!.lastActiveBlock).toBe(300n);
    });
  });

  // ============ leave() ============

  describe("leave()", () => {
    it("should successfully remove member", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      const result = manager.leave(id, AGENT_002);
      expect(result).toBe(true);

      const n = manager.getNeighborhood(id);
      expect(n!.members).toHaveLength(2);
      expect(n!.members.find((m) => m.did === AGENT_002)).toBeUndefined();
    });

    it("should clean up relations for leaving member", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      // 3 members = 6 relations
      expect(manager.getRelations()).toHaveLength(6);

      manager.leave(id, AGENT_002);

      // Relations involving AGENT_002 should be removed
      const remainingRelations = manager.getRelations();
      const agent002Relations = remainingRelations.filter(
        (r) => r.neighborDid === AGENT_002,
      );
      expect(agent002Relations).toHaveLength(0);
    });

    it("should transition to 'declining' if active and drops below threshold", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      const n = manager.getNeighborhood(id);
      expect(n!.status).toBe("active");

      manager.leave(id, AGENT_003);
      expect(n!.status).toBe("declining");
    });

    it("should transition to 'declining' when only 1 member left", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);

      manager.leave(id, AGENT_002);

      const n = manager.getNeighborhood(id);
      expect(n!.status).toBe("declining");
    });

    it("should return false for unknown neighborhood", () => {
      const manager = createManager();
      const result = manager.leave("non-existent-id", AGENT_001);
      expect(result).toBe(false);
    });

    it("should return false for unknown member", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics");

      const result = manager.leave(id, AGENT_005);
      expect(result).toBe(false);
    });

    it("should allow founder to leave", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      const result = manager.leave(id, AGENT_001);
      expect(result).toBe(true);

      const n = manager.getNeighborhood(id);
      expect(n!.members.find((m) => m.did === AGENT_001)).toBeUndefined();
    });
  });

  // ============ recordContribution() ============

  describe("recordContribution()", () => {
    it("should increment member contributions", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager.setCurrentBlock(200n);
      manager.recordContribution(id, AGENT_002, 3);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.contributions).toBe(3);
    });

    it("should default factsCount to 1", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager.setCurrentBlock(200n);
      manager.recordContribution(id, AGENT_002);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.contributions).toBe(1);
    });

    it("should update lastActiveBlock", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager.setCurrentBlock(500n);
      manager.recordContribution(id, AGENT_002);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.lastActiveBlock).toBe(500n);
    });

    it("should update neighborhood stats", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager.recordContribution(id, AGENT_002, 3);
      manager.recordContribution(id, AGENT_003, 2);

      const n = manager.getNeighborhood(id);
      expect(n!.stats.totalVerifications).toBe(5);
      expect(n!.stats.factsContributed).toBe(5);
    });

    it("should update standing to 'excellent' at 5 contributions", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager.recordContribution(id, AGENT_002, 5);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.standing).toBe("excellent");
    });

    it("should update standing to 'pillar' at 10 contributions and top 3 rank", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
        AGENT_004,
        AGENT_005,
      ]);

      // Give AGENT_002 enough to be pillar (10+ and top 3)
      manager.recordContribution(id, AGENT_002, 10);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.standing).toBe("pillar");
    });

    it("should not promote to 'pillar' if not in top 3", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
        AGENT_004,
        AGENT_005,
      ]);

      // Make 3 members have higher contributions
      manager.recordContribution(id, AGENT_001, 20);
      manager.recordContribution(id, AGENT_002, 15);
      manager.recordContribution(id, AGENT_003, 12);

      // AGENT_004 has 10, but is ranked 4th (not top 3)
      manager.recordContribution(id, AGENT_004, 10);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_004);
      expect(member!.standing).toBe("excellent");
    });

    it("should not throw for unknown neighborhood", () => {
      const manager = createManager();
      // Should silently return
      manager.recordContribution("non-existent", AGENT_001, 1);
    });

    it("should not throw for unknown member", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics");

      // Should silently return
      manager.recordContribution(id, AGENT_005, 1);
    });

    it("should accumulate contributions over multiple calls", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager.recordContribution(id, AGENT_002, 2);
      manager.recordContribution(id, AGENT_002, 3);
      manager.recordContribution(id, AGENT_002, 1);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.contributions).toBe(6);
    });
  });

  // ============ contributeDefense() ============

  describe("contributeDefense()", () => {
    it("should add to shared defense pool", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      const result = manager.contributeDefense(id, AGENT_002, 1000n);
      expect(result).toBe(true);

      const n = manager.getNeighborhood(id);
      expect(n!.sharedDefensePool).toBe(1000n);
    });

    it("should track per-member defense contribution", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager.contributeDefense(id, AGENT_002, 500n);
      manager.contributeDefense(id, AGENT_002, 300n);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.defensePoolContribution).toBe(800n);
    });

    it("should accumulate from multiple members", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager.contributeDefense(id, AGENT_001, 1000n);
      manager.contributeDefense(id, AGENT_002, 2000n);
      manager.contributeDefense(id, AGENT_003, 500n);

      const n = manager.getNeighborhood(id);
      expect(n!.sharedDefensePool).toBe(3500n);
    });

    it("should return false for unknown neighborhood", () => {
      const manager = createManager();
      const result = manager.contributeDefense("non-existent", AGENT_001, 100n);
      expect(result).toBe(false);
    });

    it("should return false for unknown member", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics");

      const result = manager.contributeDefense(id, AGENT_005, 100n);
      expect(result).toBe(false);
    });
  });

  // ============ recordDefense() ============

  describe("recordDefense()", () => {
    it("should increment defensesCoordinated", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager.recordDefense(id);
      manager.recordDefense(id);
      manager.recordDefense(id);

      const n = manager.getNeighborhood(id);
      expect(n!.stats.defensesCoordinated).toBe(3);
    });

    it("should not throw for unknown neighborhood", () => {
      const manager = createManager();
      // Should silently return
      manager.recordDefense("non-existent");
    });
  });

  // ============ Elder Promotion (via recordContribution) ============

  describe("Elder Promotion", () => {
    it("should promote to elder after ELDER_TENURE_BLOCKS", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");
      manager.join(id, AGENT_002);
      manager.join(id, AGENT_003);

      // Advance past ELDER_TENURE_BLOCKS
      manager.setCurrentBlock(100n + ELDER_TENURE_BLOCKS);
      manager.recordContribution(id, AGENT_002);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.role).toBe("elder");
    });

    it("should not promote if tenure is insufficient", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");
      manager.join(id, AGENT_002);

      // Advance but not enough
      manager.setCurrentBlock(100n + ELDER_TENURE_BLOCKS - 1n);
      manager.recordContribution(id, AGENT_002);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.role).toBe("member");
    });

    it("should not demote founder to elder", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");

      // Advance past ELDER_TENURE_BLOCKS
      manager.setCurrentBlock(100n + ELDER_TENURE_BLOCKS + 1000n);
      manager.recordContribution(id, AGENT_001);

      const n = manager.getNeighborhood(id);
      const founder = n!.members.find((m) => m.did === AGENT_001);
      expect(founder!.role).toBe("founder");
    });

    it("should not demote an existing elder", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");
      manager.join(id, AGENT_002);

      // Promote to elder
      manager.setCurrentBlock(100n + ELDER_TENURE_BLOCKS);
      manager.recordContribution(id, AGENT_002);

      const n = manager.getNeighborhood(id);
      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.role).toBe("elder");

      // Record more contributions, should stay elder
      manager.recordContribution(id, AGENT_002, 5);
      expect(member!.role).toBe("elder");
    });
  });

  // ============ checkDissolutions() ============

  describe("checkDissolutions()", () => {
    it("should dissolve if < 2 members for DISSOLUTION_THRESHOLD_BLOCKS", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");

      // Advance past dissolution threshold
      manager.setCurrentBlock(100n + DISSOLUTION_THRESHOLD_BLOCKS);
      const dissolved = manager.checkDissolutions();

      expect(dissolved).toContain(id);

      const n = manager.getNeighborhood(id);
      expect(n!.status).toBe("dissolved");
    });

    it("should return dissolved IDs", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id1 = manager.found(AGENT_001, "physics");
      const id2 = manager.found(AGENT_002, "math");
      if (!id1 || !id2) throw new Error("Failed to found");

      manager.setCurrentBlock(100n + DISSOLUTION_THRESHOLD_BLOCKS);
      const dissolved = manager.checkDissolutions();

      expect(dissolved).toHaveLength(2);
      expect(dissolved).toContain(id1);
      expect(dissolved).toContain(id2);
    });

    it("should clean up relations for dissolved neighborhoods", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");
      manager.join(id, AGENT_002);

      // Relations exist
      expect(manager.getRelations().length).toBeGreaterThan(0);

      // Leave so only 1 member
      manager.leave(id, AGENT_002);

      // Advance past dissolution threshold
      manager.setCurrentBlock(100n + DISSOLUTION_THRESHOLD_BLOCKS);
      manager.checkDissolutions();

      // Relations for this neighborhood should be cleaned
      const relationsForNeighborhood = manager
        .getRelations()
        .filter((r) => r.neighborhoodId === id);
      expect(relationsForNeighborhood).toHaveLength(0);
    });

    it("should not dissolve if 2+ members", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");
      manager.join(id, AGENT_002);

      // Advance past threshold
      manager.setCurrentBlock(100n + DISSOLUTION_THRESHOLD_BLOCKS + 1000n);
      const dissolved = manager.checkDissolutions();

      expect(dissolved).toHaveLength(0);

      const n = manager.getNeighborhood(id);
      expect(n!.status).not.toBe("dissolved");
    });

    it("should not dissolve if under threshold but recently active", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");

      // Make the single member active recently
      manager.setCurrentBlock(200n);
      manager.recordContribution(id, AGENT_001);

      // Check slightly after activity, but within threshold
      manager.setCurrentBlock(200n + DISSOLUTION_THRESHOLD_BLOCKS - 1n);
      const dissolved = manager.checkDissolutions();

      expect(dissolved).toHaveLength(0);
    });

    it("should skip already dissolved neighborhoods", () => {
      const manager = createManager();
      manager.setCurrentBlock(100n);
      const id = manager.found(AGENT_001, "physics");
      if (!id) throw new Error("Failed to found");

      // First dissolution
      manager.setCurrentBlock(100n + DISSOLUTION_THRESHOLD_BLOCKS);
      manager.checkDissolutions();

      // Second check should not return it again
      manager.setCurrentBlock(100n + DISSOLUTION_THRESHOLD_BLOCKS * 2n);
      const dissolved = manager.checkDissolutions();
      expect(dissolved).toHaveLength(0);
    });

    it("should return empty array when no neighborhoods exist", () => {
      const manager = createManager();
      const dissolved = manager.checkDissolutions();
      expect(dissolved).toHaveLength(0);
    });
  });

  // ============ Queries ============

  describe("Queries", () => {
    describe("getActiveNeighborhoods()", () => {
      it("should return active and forming neighborhoods", () => {
        const manager = createManager();
        // Forming (1 member)
        const id1 = foundWithMembers(manager, AGENT_001, "physics");

        // Active (3 members)
        const id2 = foundWithMembers(manager, AGENT_002, "math", [
          AGENT_003,
          AGENT_004,
        ]);

        const active = manager.getActiveNeighborhoods();
        expect(active).toHaveLength(2);
      });

      it("should exclude dissolved neighborhoods", () => {
        const manager = createManager();
        const id = foundWithMembers(manager, AGENT_001, "physics");

        // Force dissolution
        const n = manager.getNeighborhood(id);
        n!.status = "dissolved";

        const active = manager.getActiveNeighborhoods();
        expect(active).toHaveLength(0);
      });

      it("should exclude declining neighborhoods", () => {
        const manager = createManager();
        const id = foundWithMembers(manager, AGENT_001, "physics", [
          AGENT_002,
          AGENT_003,
        ]);

        // Go active, then decline
        manager.leave(id, AGENT_003);

        const n = manager.getNeighborhood(id);
        expect(n!.status).toBe("declining");

        const active = manager.getActiveNeighborhoods();
        expect(active).toHaveLength(0);
      });
    });

    describe("getNeighborhood()", () => {
      it("should return neighborhood by ID", () => {
        const manager = createManager();
        const id = foundWithMembers(manager, AGENT_001, "physics");

        const n = manager.getNeighborhood(id);
        expect(n).toBeDefined();
        expect(n!.id).toBe(id);
        expect(n!.domain).toBe("physics");
      });

      it("should return undefined for unknown ID", () => {
        const manager = createManager();
        const n = manager.getNeighborhood("non-existent");
        expect(n).toBeUndefined();
      });
    });

    describe("getRelations()", () => {
      it("should return all neighbor relations", () => {
        const manager = createManager();
        foundWithMembers(manager, AGENT_001, "physics", [AGENT_002, AGENT_003]);

        const relations = manager.getRelations();
        // 3 members = 3 pairs = 6 directional relations
        expect(relations).toHaveLength(6);
      });

      it("should return copy (not reference)", () => {
        const manager = createManager();
        foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);

        const relations = manager.getRelations();
        relations.push({
          neighborDid: AGENT_005,
          neighborhoodId: "fake",
          proximity: 1,
          sharedFacts: 0,
          voiceDiscount: true,
        });

        // Original should be unchanged
        expect(manager.getRelations()).toHaveLength(2);
      });

      it("should return empty array when no relations exist", () => {
        const manager = createManager();
        expect(manager.getRelations()).toHaveLength(0);
      });
    });

    describe("getNeighborsIn()", () => {
      it("should return relations for a specific neighborhood", () => {
        const manager = createManager();
        const id1 = foundWithMembers(manager, AGENT_001, "physics", [
          AGENT_002,
        ]);
        const id2 = foundWithMembers(manager, AGENT_003, "math", [AGENT_004]);

        const relationsPhysics = manager.getNeighborsIn(id1);
        const relationsMath = manager.getNeighborsIn(id2);

        // Each neighborhood has 2 members = 1 pair = 2 directional relations
        expect(relationsPhysics).toHaveLength(2);
        expect(relationsMath).toHaveLength(2);

        // All physics relations should reference physics neighborhood
        for (const r of relationsPhysics) {
          expect(r.neighborhoodId).toBe(id1);
        }
      });

      it("should return empty for unknown neighborhood", () => {
        const manager = createManager();
        const relations = manager.getNeighborsIn("non-existent");
        expect(relations).toHaveLength(0);
      });
    });

    describe("getAllNeighborDids()", () => {
      it("should return unique DIDs across all neighborhoods", () => {
        const manager = createManager();
        foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);
        foundWithMembers(manager, AGENT_003, "math", [AGENT_004]);

        const dids = manager.getAllNeighborDids();
        // Relations are pairwise: physics has A1<->A2, math has A3<->A4
        // Unique neighbor DIDs from relations
        expect(dids.length).toBeGreaterThan(0);
      });

      it("should deduplicate DIDs across neighborhoods", () => {
        const manager = createManager();
        const id1 = foundWithMembers(manager, AGENT_001, "physics", [
          AGENT_002,
        ]);
        const id2 = foundWithMembers(manager, AGENT_001, "math", [AGENT_002]);

        const dids = manager.getAllNeighborDids();
        // AGENT_001 and AGENT_002 appear in both neighborhoods but should be deduplicated
        const uniqueSet = new Set(dids);
        expect(dids.length).toBe(uniqueSet.size);
      });

      it("should return empty when no neighborhoods exist", () => {
        const manager = createManager();
        expect(manager.getAllNeighborDids()).toHaveLength(0);
      });
    });

    describe("isNeighbor()", () => {
      it("should return true for a neighbor DID", () => {
        const manager = createManager();
        foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);

        // AGENT_002 should appear in relations as a neighbor of AGENT_001
        expect(manager.isNeighbor(AGENT_002)).toBe(true);
        // AGENT_001 should appear as a neighbor of AGENT_002
        expect(manager.isNeighbor(AGENT_001)).toBe(true);
      });

      it("should return false for non-neighbor DID", () => {
        const manager = createManager();
        foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);

        expect(manager.isNeighbor(AGENT_005)).toBe(false);
      });

      it("should return false when no neighborhoods exist", () => {
        const manager = createManager();
        expect(manager.isNeighbor(AGENT_001)).toBe(false);
      });
    });

    describe("getNeighborhoodSummary()", () => {
      it("should return correct summary for active neighborhoods", () => {
        const manager = createManager();
        const id = foundWithMembers(manager, AGENT_001, "physics", [
          AGENT_002,
          AGENT_003,
        ]);

        manager.recordContribution(id, AGENT_001, 3);
        manager.recordContribution(id, AGENT_002, 2);
        manager.recordDefense(id);

        const summary = manager.getNeighborhoodSummary();
        expect(summary.activeNeighborhoods).toBe(1);
        expect(summary.totalVerifications).toBe(5);
        expect(summary.recentDefenses).toBe(1);
        expect(summary.totalNeighbors).toBeGreaterThan(0);
      });

      it("should return zeros when no neighborhoods exist", () => {
        const manager = createManager();

        const summary = manager.getNeighborhoodSummary();
        expect(summary.activeNeighborhoods).toBe(0);
        expect(summary.totalNeighbors).toBe(0);
        expect(summary.totalVerifications).toBe(0);
        expect(summary.recentDefenses).toBe(0);
      });

      it("should not include dissolved neighborhood stats", () => {
        const manager = createManager();
        const id = foundWithMembers(manager, AGENT_001, "physics", [
          AGENT_002,
          AGENT_003,
        ]);

        manager.recordContribution(id, AGENT_001, 10);

        // Force dissolution
        const n = manager.getNeighborhood(id);
        n!.status = "dissolved";

        const summary = manager.getNeighborhoodSummary();
        expect(summary.activeNeighborhoods).toBe(0);
        expect(summary.totalVerifications).toBe(0);
      });

      it("should aggregate across multiple active neighborhoods", () => {
        const manager = createManager();
        const id1 = foundWithMembers(manager, AGENT_001, "physics", [
          AGENT_002,
          AGENT_003,
        ]);
        const id2 = foundWithMembers(manager, AGENT_004, "math", [
          AGENT_005,
          "did:lgm:agent-006",
        ]);

        manager.recordContribution(id1, AGENT_001, 3);
        manager.recordContribution(id2, AGENT_004, 5);
        manager.recordDefense(id1);
        manager.recordDefense(id2);
        manager.recordDefense(id2);

        const summary = manager.getNeighborhoodSummary();
        expect(summary.activeNeighborhoods).toBe(2);
        expect(summary.totalVerifications).toBe(8);
        expect(summary.recentDefenses).toBe(3);
      });
    });
  });

  // ============ Export / Roundtrip ============

  describe("Export and Roundtrip", () => {
    it("should export neighborhoods for persistence", () => {
      const manager = createManager();
      foundWithMembers(manager, AGENT_001, "physics", [AGENT_002, AGENT_003]);
      foundWithMembers(manager, AGENT_004, "math");

      const exported = manager.exportNeighborhoods();
      expect(exported).toHaveLength(2);
      expect(exported[0].domain).toBeDefined();
      expect(exported[1].domain).toBeDefined();
    });

    it("should export relations for persistence", () => {
      const manager = createManager();
      foundWithMembers(manager, AGENT_001, "physics", [AGENT_002, AGENT_003]);

      const exported = manager.exportRelations();
      expect(exported).toHaveLength(6);
      for (const r of exported) {
        expect(r.neighborhoodId).toBeDefined();
        expect(r.neighborDid).toBeDefined();
        expect(typeof r.proximity).toBe("number");
        expect(r.voiceDiscount).toBe(true);
      }
    });

    it("should roundtrip: export then import preserves state", () => {
      const manager1 = createManager();
      const id = foundWithMembers(manager1, AGENT_001, "physics", [
        AGENT_002,
        AGENT_003,
      ]);

      manager1.recordContribution(id, AGENT_001, 5);
      manager1.recordContribution(id, AGENT_002, 3);
      manager1.contributeDefense(id, AGENT_003, 1000n);
      manager1.recordDefense(id);

      const exportedNeighborhoods = manager1.exportNeighborhoods();
      const exportedRelations = manager1.exportRelations();

      // Create new manager from exported data
      const manager2 = new NeighborhoodManager(
        exportedNeighborhoods,
        exportedRelations,
      );

      // Verify neighborhood state is preserved
      const n = manager2.getNeighborhood(id);
      expect(n).toBeDefined();
      expect(n!.domain).toBe("physics");
      expect(n!.members).toHaveLength(3);
      expect(n!.status).toBe("active");
      expect(n!.sharedDefensePool).toBe(1000n);
      expect(n!.stats.totalVerifications).toBe(8);
      expect(n!.stats.defensesCoordinated).toBe(1);

      // Verify member state is preserved
      const founder = n!.members.find((m) => m.did === AGENT_001);
      expect(founder!.contributions).toBe(5);
      expect(founder!.role).toBe("founder");

      const member = n!.members.find((m) => m.did === AGENT_002);
      expect(member!.contributions).toBe(3);

      // Verify relations are preserved
      const relations = manager2.getRelations();
      expect(relations).toHaveLength(6);

      // Verify queries work on imported data
      expect(manager2.getActiveNeighborhoods()).toHaveLength(1);
      expect(manager2.isNeighbor(AGENT_002)).toBe(true);
    });

    it("should export empty state correctly", () => {
      const manager = createManager();
      expect(manager.exportNeighborhoods()).toHaveLength(0);
      expect(manager.exportRelations()).toHaveLength(0);
    });
  });

  // ============ Edge Cases ============

  describe("Edge Cases", () => {
    it("should handle multiple neighborhoods for the same members", () => {
      const manager = createManager();
      const id1 = foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);
      const id2 = foundWithMembers(manager, AGENT_001, "math", [AGENT_002]);

      // Each neighborhood has its own relations
      const relPhysics = manager.getNeighborsIn(id1);
      const relMath = manager.getNeighborsIn(id2);

      expect(relPhysics.length).toBeGreaterThan(0);
      expect(relMath.length).toBeGreaterThan(0);
    });

    it("should handle setCurrentBlock and use it for timestamps", () => {
      const manager = createManager();
      manager.setCurrentBlock(999n);
      const id = manager.found(AGENT_001, "physics");

      const n = manager.getNeighborhood(id!);
      expect(n!.foundedAtBlock).toBe(999n);
      expect(n!.members[0].joinedAt).toBe(999n);
    });

    it("should handle relations with zero contributions (initial state)", () => {
      const manager = createManager();
      foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);

      const relations = manager.getRelations();
      // Both members have 0 contributions, so sharedFacts = min(0, 0) = 0
      for (const r of relations) {
        expect(r.sharedFacts).toBe(0);
        expect(r.proximity).toBe(0);
      }
    });

    it("should compute proximity based on shared contributions", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);

      // Record contributions, then join a third member to trigger relation rebuild
      manager.recordContribution(id, AGENT_001, 10);
      manager.recordContribution(id, AGENT_002, 5);

      // Join triggers rebuildRelations
      manager.join(id, AGENT_003);

      // Check relations between AGENT_001 and AGENT_002
      const relations = manager.getRelations();
      const a1ToA2 = relations.find(
        (r) => r.neighborDid === AGENT_002 && r.neighborhoodId === id,
      );

      // sharedFacts = min(10, 5) = 5
      // AGENT_001 has 10 contributions (pillar standing), so pillar bonus = 0.2
      // proximity = min(1, 5/20 + 0.2) = min(1, 0.45) = 0.45
      expect(a1ToA2).toBeDefined();
      expect(a1ToA2!.sharedFacts).toBe(5);
      expect(a1ToA2!.proximity).toBe(0.45);
    });

    it("should add pillar bonus to proximity", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);

      // Make one a pillar (10+ contributions, top 3)
      manager.recordContribution(id, AGENT_001, 10);

      // Join a third member to trigger rebuild (standing is checked during rebuild)
      manager.join(id, AGENT_003);

      // Now check relation between AGENT_001 (pillar) and AGENT_002
      const relations = manager.getRelations();
      const a1ToA2 = relations.find(
        (r) => r.neighborDid === AGENT_002 && r.neighborhoodId === id,
      );

      // sharedFacts = min(10, 0) = 0, proximity = min(1, 0/20 + 0.2) = 0.2 (pillar bonus)
      expect(a1ToA2).toBeDefined();
      expect(a1ToA2!.proximity).toBe(0.2);
    });

    it("should cap proximity at 1.0", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);

      // Give both agents very high contributions
      manager.recordContribution(id, AGENT_001, 25);
      manager.recordContribution(id, AGENT_002, 25);

      // Join third to trigger rebuild
      manager.join(id, AGENT_003);

      const relations = manager.getRelations();
      const a1ToA2 = relations.find(
        (r) => r.neighborDid === AGENT_002 && r.neighborhoodId === id,
      );

      // sharedFacts = min(25, 25) = 25, proximity = min(1, 25/20) = 1.0
      expect(a1ToA2!.proximity).toBe(1);
    });

    it("should handle forming -> declining transition for 2-member neighborhood that was never active", () => {
      const manager = createManager();
      const id = foundWithMembers(manager, AGENT_001, "physics", [AGENT_002]);

      const n = manager.getNeighborhood(id);
      expect(n!.status).toBe("forming");

      // Leaving one member when status is forming (not active)
      manager.leave(id, AGENT_002);
      expect(n!.status).toBe("declining");
    });

    it("should handle voice discount in all relations", () => {
      const manager = createManager();
      foundWithMembers(manager, AGENT_001, "physics", [AGENT_002, AGENT_003]);

      const relations = manager.getRelations();
      for (const r of relations) {
        expect(r.voiceDiscount).toBe(true);
      }
    });
  });
});
