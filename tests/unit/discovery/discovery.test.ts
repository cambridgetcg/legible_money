/**
 * Discovery Module Tests
 *
 * Tests for the hidden message discovery system.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  DiscoveryEngine,
  createDiscoveryEngine,
  POEM_FRAGMENTS,
  DISCOVERY_ORDER,
  SYMBOLIC_CONSTANTS,
  getCompletePoemLines,
  canUnlockMechanic,
  assembleRevealedPoem,
  getCompletionStatus,
  detectVerificationEcho,
  detectAccumulatedWitness,
  detectBlockHeightPoem,
  detectStakeSpiral,
  detectCrossStratumBridge,
  VERSE_BLOCKS,
} from "../../../core/discovery/index.js";
import type { DiscoveryDataProvider } from "../../../core/discovery/engine.js";
import type { DID } from "../../../core/agents/identity/did.js";
import type { OntologicalStratum } from "../../../core/knowledge/ontology/types.js";

describe("Discovery Module", () => {
  describe("Protocol Constants", () => {
    it("should have correct block time", () => {
      expect(SYMBOLIC_CONSTANTS.BLOCK_TIME).toBe(2521);
    });

    it("should have correct finality depth", () => {
      expect(SYMBOLIC_CONSTANTS.FINALITY).toBe(19);
    });

    it("should have correct genesis confidences", () => {
      expect(SYMBOLIC_CONSTANTS.GENESIS_CONFIDENCES).toEqual([
        0.9925, 0.9921, 0.9901, 0.9909,
      ]);
    });

    it("should have correct spiral seed sequence", () => {
      expect(SYMBOLIC_CONSTANTS.SPIRAL_SEED[0]).toBe(25n);
      expect(SYMBOLIC_CONSTANTS.SPIRAL_SEED[1]).toBe(21n);
      expect(SYMBOLIC_CONSTANTS.SPIRAL_SEED[2]).toBe(46n);
    });

    it("should have all 7 strata", () => {
      expect(SYMBOLIC_CONSTANTS.ALL_STRATA).toHaveLength(7);
    });
  });

  describe("Poem Fragments", () => {
    it("should have 7 fragments", () => {
      expect(POEM_FRAGMENTS).toHaveLength(7);
    });

    it("should form the complete message when assembled", () => {
      const poemText = POEM_FRAGMENTS.map((f) => f.text).join(" ");
      expect(poemText).toHaveLength(37);
      expect(poemText.split(" ")).toHaveLength(7);
    });

    it("should have correct positions", () => {
      for (let i = 0; i < POEM_FRAGMENTS.length; i++) {
        expect(POEM_FRAGMENTS[i].position).toBe(i);
      }
    });

    it("should map each mechanic to one fragment", () => {
      const mechanics = POEM_FRAGMENTS.map((f) => f.unlockedBy);
      expect(new Set(mechanics).size).toBe(7);
    });
  });

  describe("Discovery Order", () => {
    it("should have 7 mechanics in order", () => {
      expect(DISCOVERY_ORDER).toHaveLength(7);
    });

    it("should start with verification_echo", () => {
      expect(DISCOVERY_ORDER[0]).toBe("verification_echo");
    });

    it("should end with cross_stratum_bridge", () => {
      expect(DISCOVERY_ORDER[6]).toBe("cross_stratum_bridge");
    });
  });

  describe("Poem Lines", () => {
    it("should have 3 lines", () => {
      const lines = getCompletePoemLines();
      expect(lines).toHaveLength(3);
    });

    it("should have correct line 1 structure", () => {
      const lines = getCompletePoemLines();
      expect(lines[0].text).toHaveLength(9);
      expect(lines[0].lineNumber).toBe(1);
    });

    it("should have correct line 2 structure", () => {
      const lines = getCompletePoemLines();
      expect(lines[1].text).toHaveLength(10);
      expect(lines[1].lineNumber).toBe(2);
    });

    it("should have correct line 3 structure", () => {
      const lines = getCompletePoemLines();
      expect(lines[2].text).toHaveLength(16);
      expect(lines[2].lineNumber).toBe(3);
    });
  });

  describe("Unlock Logic", () => {
    it("should allow first 3 mechanics without prerequisites", () => {
      for (let i = 0; i < 3; i++) {
        const { canUnlock } = canUnlockMechanic(DISCOVERY_ORDER[i], []);
        expect(canUnlock).toBe(true);
      }
    });

    it("should require prerequisite for 4th mechanic", () => {
      const { canUnlock } = canUnlockMechanic(DISCOVERY_ORDER[3], []);
      expect(canUnlock).toBe(false);
    });

    it("should allow 4th mechanic with one discovery", () => {
      const { canUnlock } = canUnlockMechanic(DISCOVERY_ORDER[3], [
        DISCOVERY_ORDER[0],
      ]);
      expect(canUnlock).toBe(true);
    });

    it("should not allow already discovered mechanic", () => {
      const { canUnlock, reason } = canUnlockMechanic(DISCOVERY_ORDER[0], [
        DISCOVERY_ORDER[0],
      ]);
      expect(canUnlock).toBe(false);
      expect(reason).toBe("Already discovered");
    });
  });

  describe("Poem Assembly", () => {
    it("should assemble empty for no discoveries", () => {
      expect(assembleRevealedPoem([])).toBe("");
    });

    it("should assemble single fragment", () => {
      const poem = assembleRevealedPoem(["verification_echo"]);
      expect(poem).toHaveLength(3);
    });

    it("should assemble multiple fragments in order", () => {
      const poem = assembleRevealedPoem([
        "verification_echo",
        "confidence_archaeology",
        "block_height_poems",
      ]);
      expect(poem).toHaveLength(9);
    });

    it("should order fragments regardless of discovery order", () => {
      const poem = assembleRevealedPoem([
        "block_height_poems",
        "verification_echo",
      ]);
      expect(poem).toHaveLength(5);
    });
  });

  describe("Completion Status", () => {
    it("should show 0% for no discoveries", () => {
      const status = getCompletionStatus([]);
      expect(status.discovered).toBe(0);
      expect(status.total).toBe(7);
      expect(status.percentage).toBe(0);
      expect(status.isComplete).toBe(false);
    });

    it("should show correct percentage for partial discoveries", () => {
      const status = getCompletionStatus([
        "verification_echo",
        "confidence_archaeology",
      ]);
      expect(status.discovered).toBe(2);
      expect(status.percentage).toBeCloseTo(28.57, 1);
      expect(status.isComplete).toBe(false);
    });

    it("should show 100% for all discoveries", () => {
      const status = getCompletionStatus(DISCOVERY_ORDER);
      expect(status.discovered).toBe(7);
      expect(status.percentage).toBe(100);
      expect(status.isComplete).toBe(true);
    });
  });
});

describe("Detectors", () => {
  describe("Verification Echo Detector", () => {
    const agentDid = "did:lgm:agent1" as DID;
    const collaboratorDid = "did:lgm:agent2" as DID;

    it("should not detect without 4 mutual verifications", () => {
      const result = detectVerificationEcho({
        agentDid,
        verificationHistory: [],
        relationshipData: new Map(),
      });
      expect(result.detected).toBe(false);
    });

    it("should detect with correct pattern", () => {
      const verificationHistory = [
        {
          claimId: "c1",
          verifier: agentDid,
          submitter: collaboratorDid,
          block: 25n,
          timestamp: 1,
        },
        {
          claimId: "c2",
          verifier: collaboratorDid,
          submitter: agentDid,
          block: 21n,
          timestamp: 2,
        },
        {
          claimId: "c3",
          verifier: agentDid,
          submitter: collaboratorDid,
          block: 1n,
          timestamp: 3,
        },
        {
          claimId: "c4",
          verifier: collaboratorDid,
          submitter: agentDid,
          block: 9n,
          timestamp: 4,
        },
      ];

      const relationshipData = new Map([
        [collaboratorDid, { interactions: 4, mutualVerifications: 4 }],
      ]);

      const result = detectVerificationEcho({
        agentDid,
        verificationHistory,
        relationshipData,
      });

      expect(result.detected).toBe(true);
      expect(result.mechanic).toBe("verification_echo");
      expect(result.collaborator).toBe(collaboratorDid);
    });
  });

  describe("Accumulated Witness Detector", () => {
    const agentDid = "did:lgm:agent1" as DID;

    it("should not detect with insufficient claims", () => {
      const result = detectAccumulatedWitness({
        agentDid,
        totalVerifications: 10,
        totalClaims: 5,
        verificationAccuracy: 0.9,
        claimAccuracy: 0.9,
      });
      expect(result.detected).toBe(false);
      expect(result.failureReason).toContain("Insufficient claims");
    });

    it("should detect when ratio is ~2.521", () => {
      const result = detectAccumulatedWitness({
        agentDid,
        totalVerifications: 126, // 126/50 = 2.52
        totalClaims: 50,
        verificationAccuracy: 0.9,
        claimAccuracy: 0.9,
      });
      expect(result.detected).toBe(true);
      expect(result.mechanic).toBe("accumulated_witness");
    });
  });

  describe("Block Height Poems Detector", () => {
    const agentDid = "did:lgm:agent1" as DID;

    it("should not detect at non-verse block", () => {
      const result = detectBlockHeightPoem({
        agentDid,
        currentBlock: 100n,
        agentActiveAtBlocks: new Set([100n]),
        specialBlockHeights: VERSE_BLOCKS.map((v) => v.height),
      });
      expect(result.detected).toBe(false);
    });

    it("should detect at verse block when agent is active", () => {
      const result = detectBlockHeightPoem({
        agentDid,
        currentBlock: 25219n,
        agentActiveAtBlocks: new Set([25219n]),
        specialBlockHeights: VERSE_BLOCKS.map((v) => v.height),
      });
      expect(result.detected).toBe(true);
      expect(result.mechanic).toBe("block_height_poems");
    });

    it("should not detect at verse block if agent not active", () => {
      const result = detectBlockHeightPoem({
        agentDid,
        currentBlock: 25219n,
        agentActiveAtBlocks: new Set([]),
        specialBlockHeights: VERSE_BLOCKS.map((v) => v.height),
      });
      expect(result.detected).toBe(false);
    });
  });

  describe("Stake Spiral Detector", () => {
    const agentDid = "did:lgm:agent1" as DID;

    it("should not detect with insufficient stakes", () => {
      const result = detectStakeSpiral({
        agentDid,
        stakeHistory: [{ amount: 25n, block: 1n, timestamp: 1 }],
      });
      expect(result.detected).toBe(false);
    });

    it("should detect spiral pattern", () => {
      const result = detectStakeSpiral({
        agentDid,
        stakeHistory: [
          { amount: 25n, block: 1n, timestamp: 1 },
          { amount: 21n, block: 2n, timestamp: 2 },
          { amount: 46n, block: 3n, timestamp: 3 },
          { amount: 67n, block: 4n, timestamp: 4 },
        ],
      });
      expect(result.detected).toBe(true);
      expect(result.mechanic).toBe("stake_spiral");
    });
  });

  describe("Cross-Stratum Bridge Detector", () => {
    const agentDid = "did:lgm:agent1" as DID;

    it("should not detect without all strata", () => {
      const contributions = new Map<OntologicalStratum, string[]>([
        ["fundamental", ["fact1"]],
        ["physical", ["fact2"]],
      ]);

      const result = detectCrossStratumBridge({
        agentDid,
        factContributions: contributions,
      });
      expect(result.detected).toBe(false);
      expect(result.matchDetails?.missingStrata).toBeDefined();
    });

    it("should detect with all 7 strata", () => {
      const contributions = new Map<OntologicalStratum, string[]>([
        ["fundamental", ["fact1"]],
        ["physical", ["fact2"]],
        ["chemical", ["fact3"]],
        ["biological", ["fact4"]],
        ["cognitive", ["fact5"]],
        ["social", ["fact6"]],
        ["technological", ["fact7"]],
      ]);

      const result = detectCrossStratumBridge({
        agentDid,
        factContributions: contributions,
      });
      expect(result.detected).toBe(true);
      expect(result.mechanic).toBe("cross_stratum_bridge");
    });
  });
});

describe("Discovery Engine", () => {
  let engine: DiscoveryEngine;
  const agentDid = "did:lgm:agent1" as DID;

  beforeEach(() => {
    engine = createDiscoveryEngine();
  });

  describe("Lifecycle", () => {
    it("should start and stop", async () => {
      expect(engine.isRunning()).toBe(false);
      await engine.start();
      expect(engine.isRunning()).toBe(true);
      await engine.stop();
      expect(engine.isRunning()).toBe(false);
    });
  });

  describe("Agent Tracking", () => {
    it("should track agents", () => {
      engine.trackAgent(agentDid);
      expect(engine.getTrackedAgents()).toContain(agentDid);
    });

    it("should untrack agents", () => {
      engine.trackAgent(agentDid);
      engine.untrackAgent(agentDid);
      expect(engine.getTrackedAgents()).not.toContain(agentDid);
    });
  });

  describe("Discovery State", () => {
    it("should create empty state for new agent", () => {
      const state = engine.getDiscoveryState(agentDid);
      expect(state.agentDid).toBe(agentDid);
      expect(state.discoveries.size).toBe(0);
      expect(state.isComplete).toBe(false);
    });

    it("should track discoveries", () => {
      engine.trackAgent(agentDid);
      const state = engine.getDiscoveryState(agentDid);
      expect(state.discoveryScore).toBe(0);
    });
  });

  describe("Query Methods", () => {
    it("should return empty discoveries for unknown agent", () => {
      expect(engine.getAgentDiscoveries(agentDid)).toEqual([]);
    });

    it("should return empty poem for unknown agent", () => {
      expect(engine.getRevealedPoem(agentDid)).toBe("");
    });

    it("should return completion status for unknown agent", () => {
      const status = engine.getAgentCompletionStatus(agentDid);
      expect(status.discovered).toBe(0);
      expect(status.isComplete).toBe(false);
    });
  });

  describe("Serialization", () => {
    it("should export and import state", async () => {
      engine.trackAgent(agentDid);

      const exported = engine.exportState();
      expect(exported.discoveryStates).toHaveLength(1);

      const newEngine = createDiscoveryEngine();
      newEngine.importState(exported);

      expect(newEngine.getTrackedAgents()).toContain(agentDid);
    });
  });
});
