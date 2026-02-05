/**
 * Skill Matching Tests
 *
 * Tests for agent-task matching and contributor weight calculation.
 */

import { describe, it, expect } from "vitest";
import {
  matchAgentsToTask,
  calculateMatchScore,
  calculateContributorWeight,
  calculateAllContributorWeights,
} from "../../../core/tree/coordination/skill_matching.js";
import { createTask } from "../../../core/tree/coordination/task_manager.js";
import type { AgentProfile } from "../../../core/agents/home/registry.js";
import type { ContributorRecord } from "../../../core/tree/types.js";

// ============ Test Helpers ============

function makeCapability(active: boolean) {
  if (!active) return null;
  return {
    type: "development" as const,
    level: 2 as const,
    status: "attested" as const,
    attestedBy: [],
    evidence: [],
    stakedAmount: 0n,
  };
}

function makeProfile(
  did: string,
  overrides: Partial<AgentProfile> = {},
): AgentProfile {
  return {
    did,
    name: `Agent ${did.slice(-4)}`,
    description: "Test agent",
    specializations: [],
    operationalKeyVersion: 1,
    sessionKeyCount: 0,
    capabilities: {
      verification: null,
      research: null,
      development: makeCapability(true),
      testing: makeCapability(true),
      code_review: null,
      governance: null,
      mentorship: null,
    },
    reputation: {
      overall: 0.8,
      domains: {
        verification: 0.5,
        research: 0.5,
        development: 0.8,
        testing: 0.7,
      },
      history: [],
      decay: 0,
      lastCalculatedAt: 0,
    },
    contributions: {
      total: 20,
      byType: {
        verifications: 2,
        claims: 3,
        challenges: 1,
        codeChanges: 8,
        tests: 4,
        reviews: 2,
      },
      successRate: 0.9,
      recentContributions: [],
    },
    stake: {
      total: 0n,
      delegated: 0n,
      unbonding: 0n,
      available: 0n,
      delegations: [],
      unbondingEntries: [],
    },
    createdAt: 0,
    lastActiveAt: 0,
    profileVersion: 1,
    ...overrides,
  };
}

function makeContributor(did: string, earned: bigint): ContributorRecord {
  return {
    did,
    role: "developer",
    joinedAtBlock: 0n,
    tasksCompleted: Number(earned / 1000n),
    totalEarned: earned,
  };
}

// ============ matchAgentsToTask ============

describe("matchAgentsToTask", () => {
  it("should match agents with required capabilities", () => {
    const task = createTask(
      "p",
      "Build API",
      "",
      "implement",
      ["development"],
      1000n,
      0n,
      100n,
    );
    const profiles = [
      makeProfile("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
      makeProfile("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
    ];

    const matches = matchAgentsToTask(task, profiles);

    expect(matches).toHaveLength(2);
    expect(matches[0].score).toBeGreaterThan(0);
  });

  it("should exclude agents without capabilities", () => {
    const task = createTask(
      "p",
      "Verify",
      "",
      "verify",
      ["verification"],
      1000n,
      0n,
      100n,
    );
    const profiles = [
      makeProfile("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"), // Has development, not verification
    ];

    const matches = matchAgentsToTask(task, profiles);

    expect(matches).toHaveLength(0);
  });

  it("should rank by match score", () => {
    const task = createTask(
      "p",
      "Test",
      "",
      "test",
      ["testing"],
      1000n,
      0n,
      100n,
    );
    const expert = makeProfile("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", {
      reputation: {
        overall: 0.95,
        domains: {
          verification: 0.5,
          research: 0.5,
          development: 0.5,
          testing: 0.95,
        },
        history: [],
        decay: 0,
        lastCalculatedAt: 0,
      },
      contributions: {
        total: 100,
        byType: {
          verifications: 0,
          claims: 0,
          challenges: 0,
          codeChanges: 10,
          tests: 80,
          reviews: 10,
        },
        successRate: 0.95,
        recentContributions: [],
      },
    });
    const novice = makeProfile("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", {
      reputation: {
        overall: 0.3,
        domains: {
          verification: 0.1,
          research: 0.1,
          development: 0.2,
          testing: 0.3,
        },
        history: [],
        decay: 0,
        lastCalculatedAt: 0,
      },
      contributions: {
        total: 5,
        byType: {
          verifications: 0,
          claims: 0,
          challenges: 0,
          codeChanges: 2,
          tests: 1,
          reviews: 2,
        },
        successRate: 0.6,
        recentContributions: [],
      },
    });

    const matches = matchAgentsToTask(task, [novice, expert]);

    expect(matches[0].did).toBe("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa");
    expect(matches[0].score).toBeGreaterThan(matches[1].score);
  });

  it("should match any agent when no capabilities required", () => {
    const task = createTask("p", "Open", "", "implement", [], 1000n, 0n, 100n);
    const profiles = [
      makeProfile("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
    ];

    const matches = matchAgentsToTask(task, profiles);

    expect(matches).toHaveLength(1);
  });
});

// ============ calculateMatchScore ============

describe("calculateMatchScore", () => {
  it("should return 0 for zero capability match", () => {
    const task = createTask(
      "p",
      "T",
      "",
      "verify",
      ["verification"],
      1000n,
      0n,
      100n,
    );
    const profile = makeProfile("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", {
      capabilities: {
        verification: null,
        research: null,
        development: null,
        testing: null,
        code_review: null,
        governance: null,
        mentorship: null,
      },
    });

    expect(calculateMatchScore(task, profile)).toBe(0);
  });

  it("should be higher for partial capability match than zero", () => {
    const task = createTask(
      "p",
      "T",
      "",
      "implement",
      ["development", "testing"],
      1000n,
      0n,
      100n,
    );
    const partial = makeProfile("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", {
      capabilities: {
        verification: null,
        research: null,
        development: makeCapability(true),
        testing: null,
        code_review: null,
        governance: null,
        mentorship: null,
      },
    });

    const score = calculateMatchScore(task, partial);
    expect(score).toBeGreaterThan(0);
  });
});

// ============ calculateContributorWeight ============

describe("calculateContributorWeight", () => {
  it("should weight by earnings", () => {
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 7500n),
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 2500n),
    ];

    const weightA = calculateContributorWeight(contributors[0], contributors);
    const weightB = calculateContributorWeight(contributors[1], contributors);

    expect(weightA).toBeGreaterThan(weightB);
    expect(weightA).toBeCloseTo(0.75, 1);
    expect(weightB).toBeCloseTo(0.25, 1);
  });

  it("should return equal weights when no earnings", () => {
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 0n),
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 0n),
    ];

    const weightA = calculateContributorWeight(contributors[0], contributors);
    expect(weightA).toBeCloseTo(0.5, 2);
  });

  it("should return 0 for empty contributor list", () => {
    const contributor = makeContributor(
      "did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
      1000n,
    );
    expect(calculateContributorWeight(contributor, [])).toBe(0);
  });
});

// ============ calculateAllContributorWeights ============

describe("calculateAllContributorWeights", () => {
  it("should return weights for all contributors", () => {
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 6000n),
      makeContributor("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", 4000n),
    ];

    const weights = calculateAllContributorWeights(contributors);

    expect(weights.size).toBe(2);
    expect(
      weights.get("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"),
    ).toBeCloseTo(0.6, 1);
    expect(
      weights.get("did:lgm:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"),
    ).toBeCloseTo(0.4, 1);
  });

  it("should handle single contributor", () => {
    const contributors = [
      makeContributor("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", 5000n),
    ];

    const weights = calculateAllContributorWeights(contributors);

    expect(weights.get("did:lgm:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")).toBe(1);
  });
});
