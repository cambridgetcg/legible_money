/**
 * Skill Matching
 *
 * Matches agent capabilities to project task requirements.
 * Uses AgentProfile from core/agents/home/registry.ts for capability and reputation data.
 */

import type { DID } from "../../agents/identity/did.js";
import type {
  AgentProfile,
  Capability,
  CapabilitySet,
} from "../../agents/home/registry.js";
import type { ProjectTask, ContributorRecord } from "../types.js";

// ============ Matching Weight Constants ============

/** Weights for overall match score (sum = 1.0) */
const MATCH_WEIGHTS = {
  CAPABILITY: 0.5,
  REPUTATION: 0.3,
  EXPERIENCE: 0.2,
} as const;

/** Weights for experience sub-score (sum = 1.0) */
const EXPERIENCE_WEIGHTS = {
  RELEVANCE: 0.6,
  SUCCESS: 0.25,
  RECENCY: 0.15,
} as const;

/** Recency half-life: 3 days in milliseconds */
const RECENCY_HALF_LIFE_MS = 3 * 24 * 60 * 60 * 1000;

// ============ Capability-to-Task Name Mapping ============

/**
 * Maps each CapabilitySet key to the task capability names it satisfies.
 * Single source of truth — add new capabilities here when extending CapabilityType.
 */
const CAPABILITY_TASK_NAMES: Record<keyof CapabilitySet, string[]> = {
  verification: ["verification", "verify"],
  research: ["research"],
  development: ["development", "implement", "deploy"],
  testing: ["testing", "test"],
  code_review: ["code_review", "review"],
  governance: ["governance"],
  mentorship: ["mentorship", "document"],
};

/**
 * Composite capabilities that require multiple base capabilities.
 */
const COMPOSITE_CAPABILITIES: Array<{
  name: string;
  requires: (keyof CapabilitySet)[];
}> = [{ name: "design", requires: ["research", "development"] }];

/**
 * All valid task capability names that can appear in ProjectTask.requiredCapabilities.
 * Derived from CAPABILITY_TASK_NAMES values + COMPOSITE_CAPABILITIES names.
 */
export const VALID_TASK_CAPABILITY_NAMES: ReadonlySet<string> = new Set([
  ...Object.values(CAPABILITY_TASK_NAMES).flat(),
  ...COMPOSITE_CAPABILITIES.map((c) => c.name),
]);

// ============ Skill Matching ============

/**
 * Match agents to a task, ranked by suitability.
 * Returns DIDs sorted by match score (best first).
 */
export function matchAgentsToTask(
  task: ProjectTask,
  profiles: AgentProfile[],
): AgentMatch[] {
  const matches: AgentMatch[] = [];

  for (const profile of profiles) {
    const score = calculateMatchScore(task, profile);
    if (score > 0) {
      matches.push({
        did: profile.did,
        score,
        matchedCapabilities: getMatchedCapabilities(task, profile),
        reputationScore: profile.reputation.overall,
      });
    }
  }

  // Sort by score descending
  return matches.sort((a, b) => b.score - a.score);
}

/**
 * Calculate how well an agent matches a task (0-1).
 */
export function calculateMatchScore(
  task: ProjectTask,
  profile: AgentProfile,
): number {
  const {
    CAPABILITY: CAPABILITY_WEIGHT,
    REPUTATION: REPUTATION_WEIGHT,
    EXPERIENCE: EXPERIENCE_WEIGHT,
  } = MATCH_WEIGHTS;

  // 1. Capability match (0-1): what fraction of required capabilities does agent have,
  //    weighted by capability level?
  const capabilityScore = calculateCapabilityMatch(
    task.requiredCapabilities,
    profile.capabilities,
  );

  // Must have at least some capability match
  if (capabilityScore === 0) return 0;

  // 2. Reputation score (0-1): agent's overall reputation
  const reputationScore = profile.reputation.overall;

  // 3. Experience score (0-1): relevant contribution history with recency weighting
  const experienceScore = calculateExperienceScore(task, profile);

  return (
    capabilityScore * CAPABILITY_WEIGHT +
    reputationScore * REPUTATION_WEIGHT +
    experienceScore * EXPERIENCE_WEIGHT
  );
}

/**
 * Calculate contributor weight for revenue distribution.
 * Based on tasks completed and bounties earned within a project.
 *
 * Uses 1_000_000 basis points (6 decimals) for precision to prevent
 * cascading precision loss in bigint revenue splits.
 */
export function calculateContributorWeight(
  contributor: ContributorRecord,
  allContributors: ContributorRecord[],
): number {
  if (allContributors.length === 0) return 0;

  // Weight by total earned (which reflects both task count and task value)
  const totalEarned = allContributors.reduce(
    (sum, c) => sum + c.totalEarned,
    0n,
  );

  if (totalEarned === 0n) {
    // Equal weight if no earnings yet
    return 1 / allContributors.length;
  }

  // Use 1_000_000 BP (6 decimals) to minimize precision loss
  return (
    Number((contributor.totalEarned * 1_000_000n) / totalEarned) / 1_000_000
  );
}

/**
 * Calculate contributor weights for all contributors in a project.
 * Computes totalEarned once (O(N)) rather than per-contributor (O(N²)).
 */
export function calculateAllContributorWeights(
  contributors: ContributorRecord[],
): Map<DID, number> {
  const weights = new Map<DID, number>();
  if (contributors.length === 0) return weights;

  const totalEarned = contributors.reduce((sum, c) => sum + c.totalEarned, 0n);

  for (const contributor of contributors) {
    if (totalEarned === 0n) {
      weights.set(contributor.did, 1 / contributors.length);
    } else {
      weights.set(
        contributor.did,
        Number((contributor.totalEarned * 1_000_000n) / totalEarned) /
          1_000_000,
      );
    }
  }

  return weights;
}

// ============ Internal Helpers ============

/**
 * Check if a capability is active (attested, not revoked, and not expired).
 * Uses provided timestamp instead of Date.now() for consensus determinism.
 * When called off-chain (matching queries), pass Date.now().
 * When called on-chain (state transitions), pass block timestamp.
 */
function isActiveCapability(
  cap: Capability | null,
  nowMs?: number,
): cap is Capability {
  if (cap === null || cap.status !== "attested") return false;
  if (cap.expiresAt !== undefined) {
    const ts = nowMs ?? Date.now();
    if (cap.expiresAt <= ts) return false;
  }
  return true;
}

/**
 * Get the level multiplier for a capability (1-3 → 0.6-1.0).
 * Level 1 = 0.6, Level 2 = 0.8, Level 3 = 1.0
 */
function levelMultiplier(cap: Capability): number {
  return 0.4 + cap.level * 0.2;
}

/**
 * Calculate capability match score (0-1), weighted by capability level.
 *
 * Each matched capability contributes its level-adjusted score.
 * A level-3 expert (1.0) scores higher than a level-1 novice (0.6).
 */
function calculateCapabilityMatch(
  required: string[],
  capabilities: CapabilitySet,
): number {
  if (required.length === 0) return 1; // No requirements = anyone can do it

  const activeCapMap = getActiveCapabilityMap(capabilities);
  let totalScore = 0;

  for (const req of required) {
    const cap = activeCapMap.get(req);
    if (cap) {
      totalScore += levelMultiplier(cap);
    }
  }

  return totalScore / required.length;
}

/**
 * Build a map from task capability name → source Capability (with level).
 * Only includes attested, non-revoked capabilities.
 */
function getActiveCapabilityMap(
  capabilities: CapabilitySet,
): Map<string, Capability> {
  const map = new Map<string, Capability>();

  for (const [key, taskNames] of Object.entries(CAPABILITY_TASK_NAMES)) {
    const cap = capabilities[key as keyof CapabilitySet];
    if (isActiveCapability(cap)) {
      for (const name of taskNames) {
        map.set(name, cap);
      }
    }
  }

  // Composite capabilities: all required base capabilities must be active
  for (const composite of COMPOSITE_CAPABILITIES) {
    const baseCaps = composite.requires.map((k) => capabilities[k]);
    if (baseCaps.every(isActiveCapability)) {
      // Use the minimum level among required capabilities
      const minLevel = Math.min(...baseCaps.map((c) => c!.level)) as 1 | 2 | 3;
      map.set(composite.name, { ...baseCaps[0]!, level: minLevel });
    }
  }

  return map;
}

/**
 * Get list of active capability names from a CapabilitySet.
 * Only includes attested capabilities.
 */
function getActiveCapabilities(capabilities: CapabilitySet): string[] {
  return Array.from(getActiveCapabilityMap(capabilities).keys());
}

/**
 * Get the capabilities from the task that the agent matches.
 */
function getMatchedCapabilities(
  task: ProjectTask,
  profile: AgentProfile,
): string[] {
  const capabilityNames = getActiveCapabilities(profile.capabilities);
  return task.requiredCapabilities.filter((r) => capabilityNames.includes(r));
}

/**
 * Calculate experience relevance score (0-1).
 *
 * Components:
 * - Relevance ratio: fraction of contributions in the task's domain (0-1)
 * - Success bonus: scaled by success rate (0-0.2)
 * - Recency bonus: recent activity boosts score (0-0.15)
 *
 * Uses weighted combination instead of additive clamping to preserve
 * differentiation across the full range.
 */
function calculateExperienceScore(
  task: ProjectTask,
  profile: AgentProfile,
): number {
  const contributions = profile.contributions;

  // Map task type to contribution type
  const relevantCount = getRelevantContributionCount(
    task.taskType,
    contributions.byType,
  );
  const total = contributions.total;

  if (total === 0) return 0;

  // Relevance ratio (0-1): what fraction of contributions are relevant
  const relevanceRatio = relevantCount / total;

  // Success factor (0-1): contribution success rate
  const successFactor = contributions.successRate;

  // Recency factor (0-1): how recently active the agent was
  const recencyFactor = calculateRecencyFactor(profile.lastActiveAt);

  // Weighted combination (weights sum to 1.0 — no clamping needed)
  return (
    relevanceRatio * EXPERIENCE_WEIGHTS.RELEVANCE +
    successFactor * EXPERIENCE_WEIGHTS.SUCCESS +
    recencyFactor * EXPERIENCE_WEIGHTS.RECENCY
  );
}

/**
 * Calculate recency factor (0-1) based on last activity time.
 * More recent activity scores higher.
 *
 * - Active within last hour: 1.0
 * - Active within last day: ~0.8
 * - Active within last week: ~0.5
 * - Inactive > 30 days: ~0.1
 */
function calculateRecencyFactor(lastActiveAt: number, nowMs?: number): number {
  if (lastActiveAt === 0) return 0;

  const now = nowMs ?? Date.now();
  const elapsed = Math.max(0, now - lastActiveAt);

  return Math.pow(0.5, elapsed / RECENCY_HALF_LIFE_MS);
}

/**
 * Get count of contributions relevant to a task type.
 */
function getRelevantContributionCount(
  taskType: string,
  byType: {
    verifications: number;
    claims: number;
    challenges: number;
    codeChanges: number;
    tests: number;
    reviews: number;
  },
): number {
  switch (taskType) {
    case "verify":
      return byType.verifications;
    case "implement":
      return byType.codeChanges;
    case "test":
      return byType.tests;
    case "review":
      return byType.reviews;
    case "research":
      return byType.claims + byType.verifications;
    case "design":
      return byType.claims + byType.codeChanges;
    case "document":
      return byType.reviews;
    case "deploy":
      return byType.codeChanges;
    case "maintain":
      return byType.codeChanges + byType.tests;
    case "governance":
      return byType.reviews + byType.verifications;
    default:
      return 0;
  }
}

// ============ Types ============

/**
 * Result of matching an agent to a task
 */
export interface AgentMatch {
  did: DID;
  score: number;
  matchedCapabilities: string[];
  reputationScore: number;
}
