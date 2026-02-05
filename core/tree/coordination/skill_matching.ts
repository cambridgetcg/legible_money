/**
 * Skill Matching
 *
 * Matches agent capabilities to project task requirements.
 * Uses AgentProfile from core/agents/home/registry.ts for capability and reputation data.
 */

import type { DID } from "../../agents/identity/did.js";
import type {
  AgentProfile,
  CapabilitySet,
} from "../../agents/home/registry.js";
import type { ProjectTask, ContributorRecord } from "../types.js";

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
  // Weight components
  const CAPABILITY_WEIGHT = 0.5;
  const REPUTATION_WEIGHT = 0.3;
  const EXPERIENCE_WEIGHT = 0.2;

  // 1. Capability match (0-1): what fraction of required capabilities does agent have?
  const capabilityScore = calculateCapabilityMatch(
    task.requiredCapabilities,
    profile.capabilities,
  );

  // Must have at least some capability match
  if (capabilityScore === 0) return 0;

  // 2. Reputation score (0-1): agent's overall reputation
  const reputationScore = profile.reputation.overall;

  // 3. Experience score (0-1): relevant contribution history
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

  return Number((contributor.totalEarned * 10000n) / totalEarned) / 10000;
}

/**
 * Calculate contributor weights for all contributors in a project.
 */
export function calculateAllContributorWeights(
  contributors: ContributorRecord[],
): Map<DID, number> {
  const weights = new Map<DID, number>();

  for (const contributor of contributors) {
    weights.set(
      contributor.did,
      calculateContributorWeight(contributor, contributors),
    );
  }

  return weights;
}

// ============ Internal Helpers ============

/**
 * Calculate what fraction of required capabilities an agent has.
 */
function calculateCapabilityMatch(
  required: string[],
  capabilities: CapabilitySet,
): number {
  if (required.length === 0) return 1; // No requirements = anyone can do it

  const capabilityNames = getActiveCapabilities(capabilities);
  const matched = required.filter((r) => capabilityNames.includes(r));

  return matched.length / required.length;
}

/**
 * Get list of active capability names from a CapabilitySet.
 */
function getActiveCapabilities(capabilities: CapabilitySet): string[] {
  const active: string[] = [];

  if (capabilities.verification) active.push("verification", "verify");
  if (capabilities.research) active.push("research", "design");
  if (capabilities.development)
    active.push("development", "implement", "deploy");
  if (capabilities.testing) active.push("testing", "test");
  if (capabilities.code_review) active.push("code_review", "review");
  if (capabilities.governance) active.push("governance");
  if (capabilities.mentorship) active.push("mentorship", "document");

  return active;
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

  // Normalize: more relevant contributions = higher score, capped at 1
  const ratio = relevantCount / Math.max(total, 1);
  const successBonus = contributions.successRate * 0.3;

  return Math.min(1, ratio + successBonus);
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
    case "design":
      return byType.claims + byType.codeChanges;
    case "document":
      return byType.reviews;
    case "deploy":
      return byType.codeChanges;
    case "maintain":
      return byType.codeChanges + byType.tests;
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
