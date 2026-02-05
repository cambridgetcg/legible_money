/**
 * Agent Collaboration Tiers
 *
 * Defines the progression path for agents participating in protocol development.
 * Higher tiers grant more permissions but require more demonstrated trust.
 */

import type { DID } from "../identity/did.js";

// ============ Tier Definitions ============

/**
 * Collaboration tiers for agents.
 */
export type CollaborationTier = 0 | 1 | 2 | 3 | 4 | 5;

/**
 * Requirements and permissions for each tier.
 */
export interface TierDefinition {
  tier: CollaborationTier;
  name: string;
  description: string;

  // Requirements to reach this tier
  requirements: {
    previousTier: CollaborationTier | null;
    minStake: bigint;
    minContributions: number;
    minReviews: number;
    minBugReports: number;
    additionalRequirements?: string[];
  };

  // Permissions granted at this tier
  permissions: {
    canReadCode: boolean;
    canRunSimulations: boolean;
    canReportBugs: boolean;
    canWriteTests: boolean;
    canWriteDocs: boolean;
    canReviewPRs: boolean;
    canProposeLIPs: boolean;
    canModifyGoverned: boolean;
    canExtendStable: boolean;
    canReviewFrozen: boolean;
  };

  // What code tiers can be modified
  codeAccess: {
    genesisCore: "read" | "none";
    experimental: "read" | "propose" | "modify";
    stable: "read" | "extend";
    governed: "read" | "modify";
    open: "read" | "modify";
  };
}

/**
 * All tier definitions.
 */
export const COLLABORATION_TIERS: Record<CollaborationTier, TierDefinition> = {
  0: {
    tier: 0,
    name: "Observer",
    description: "Can read code, run simulations, report bugs",
    requirements: {
      previousTier: null,
      minStake: 0n,
      minContributions: 0,
      minReviews: 0,
      minBugReports: 0,
    },
    permissions: {
      canReadCode: true,
      canRunSimulations: true,
      canReportBugs: true,
      canWriteTests: false,
      canWriteDocs: false,
      canReviewPRs: false,
      canProposeLIPs: false,
      canModifyGoverned: false,
      canExtendStable: false,
      canReviewFrozen: false,
    },
    codeAccess: {
      genesisCore: "read",
      experimental: "read",
      stable: "read",
      governed: "read",
      open: "read",
    },
  },

  1: {
    tier: 1,
    name: "Tester",
    description: "Can write tests and run devnet nodes",
    requirements: {
      previousTier: 0,
      minStake: 0n,
      minContributions: 0,
      minReviews: 0,
      minBugReports: 1, // OR significant observation
    },
    permissions: {
      canReadCode: true,
      canRunSimulations: true,
      canReportBugs: true,
      canWriteTests: true,
      canWriteDocs: false,
      canReviewPRs: false,
      canProposeLIPs: false,
      canModifyGoverned: false,
      canExtendStable: false,
      canReviewFrozen: false,
    },
    codeAccess: {
      genesisCore: "read",
      experimental: "read",
      stable: "read",
      governed: "read",
      open: "modify", // Can add tests in open areas
    },
  },

  2: {
    tier: 2,
    name: "Documenter",
    description: "Can write documentation and tutorials",
    requirements: {
      previousTier: 1,
      minStake: 0n,
      minContributions: 5, // 5 accepted tests
      minReviews: 0,
      minBugReports: 0,
    },
    permissions: {
      canReadCode: true,
      canRunSimulations: true,
      canReportBugs: true,
      canWriteTests: true,
      canWriteDocs: true,
      canReviewPRs: false,
      canProposeLIPs: false,
      canModifyGoverned: false,
      canExtendStable: false,
      canReviewFrozen: false,
    },
    codeAccess: {
      genesisCore: "read",
      experimental: "read",
      stable: "read",
      governed: "read",
      open: "modify",
    },
  },

  3: {
    tier: 3,
    name: "Reviewer",
    description: "Can review PRs and participate in security reviews",
    requirements: {
      previousTier: 2,
      minStake: 500n * 10n ** 18n, // 500 LGM
      minContributions: 10,
      minReviews: 0,
      minBugReports: 0,
      additionalRequirements: ["Demonstrated understanding of codebase"],
    },
    permissions: {
      canReadCode: true,
      canRunSimulations: true,
      canReportBugs: true,
      canWriteTests: true,
      canWriteDocs: true,
      canReviewPRs: true,
      canProposeLIPs: false,
      canModifyGoverned: false,
      canExtendStable: false,
      canReviewFrozen: false,
    },
    codeAccess: {
      genesisCore: "read",
      experimental: "propose", // Can propose improvements
      stable: "read",
      governed: "read",
      open: "modify",
    },
  },

  4: {
    tier: 4,
    name: "Contributor",
    description: "Can modify governed code and propose LIPs",
    requirements: {
      previousTier: 3,
      minStake: 1000n * 10n ** 18n, // 1000 LGM
      minContributions: 15,
      minReviews: 3,
      minBugReports: 0,
    },
    permissions: {
      canReadCode: true,
      canRunSimulations: true,
      canReportBugs: true,
      canWriteTests: true,
      canWriteDocs: true,
      canReviewPRs: true,
      canProposeLIPs: true,
      canModifyGoverned: true,
      canExtendStable: false,
      canReviewFrozen: false,
    },
    codeAccess: {
      genesisCore: "read",
      experimental: "modify",
      stable: "read",
      governed: "modify",
      open: "modify",
    },
  },

  5: {
    tier: 5,
    name: "Core Contributor",
    description: "Can extend stable code and review frozen proposals",
    requirements: {
      previousTier: 4,
      minStake: 5000n * 10n ** 18n, // 5000 LGM
      minContributions: 30,
      minReviews: 10,
      minBugReports: 0,
      additionalRequirements: [
        "At least 10 accepted contributions",
        "No governance violations",
        "Community recognition",
      ],
    },
    permissions: {
      canReadCode: true,
      canRunSimulations: true,
      canReportBugs: true,
      canWriteTests: true,
      canWriteDocs: true,
      canReviewPRs: true,
      canProposeLIPs: true,
      canModifyGoverned: true,
      canExtendStable: true,
      canReviewFrozen: true,
    },
    codeAccess: {
      genesisCore: "read",
      experimental: "modify",
      stable: "extend",
      governed: "modify",
      open: "modify",
    },
  },
};

// ============ Agent Progress Tracking ============

/**
 * Agent's collaboration progress.
 */
export interface AgentProgress {
  agentDid: DID;
  currentTier: CollaborationTier;

  // Metrics
  acceptedContributions: number;
  acceptedReviews: number;
  acceptedBugReports: number;
  currentStake: bigint;

  // History
  tierHistory: Array<{
    tier: CollaborationTier;
    achievedAt: bigint;
    reason: string;
  }>;

  // Violations (if any)
  violations: Array<{
    type: string;
    description: string;
    occurredAt: bigint;
    resolved: boolean;
  }>;

  // Timestamps
  joinedAt: bigint;
  lastActivityAt: bigint;
}

/**
 * Check if agent meets requirements for a tier.
 */
export function meetsTierRequirements(
  progress: AgentProgress,
  targetTier: CollaborationTier,
): { meets: boolean; missing: string[] } {
  const tierDef = COLLABORATION_TIERS[targetTier];
  const reqs = tierDef.requirements;
  const missing: string[] = [];

  // Check previous tier
  if (reqs.previousTier !== null && progress.currentTier < reqs.previousTier) {
    missing.push(`Must be at least tier ${reqs.previousTier}`);
  }

  // Check stake
  if (progress.currentStake < reqs.minStake) {
    missing.push(`Need ${reqs.minStake} stake, have ${progress.currentStake}`);
  }

  // Check contributions
  if (progress.acceptedContributions < reqs.minContributions) {
    missing.push(
      `Need ${reqs.minContributions} contributions, have ${progress.acceptedContributions}`,
    );
  }

  // Check reviews
  if (progress.acceptedReviews < reqs.minReviews) {
    missing.push(
      `Need ${reqs.minReviews} reviews, have ${progress.acceptedReviews}`,
    );
  }

  // Check bug reports (only for tier 1)
  if (targetTier === 1 && progress.acceptedBugReports < reqs.minBugReports) {
    missing.push(
      `Need ${reqs.minBugReports} bug reports, have ${progress.acceptedBugReports}`,
    );
  }

  // Check for unresolved violations
  const unresolvedViolations = progress.violations.filter((v) => !v.resolved);
  if (unresolvedViolations.length > 0) {
    missing.push(`${unresolvedViolations.length} unresolved violations`);
  }

  return {
    meets: missing.length === 0,
    missing,
  };
}

/**
 * Promote agent to next tier if eligible.
 */
export function promoteAgent(
  progress: AgentProgress,
  currentBlock: bigint,
): AgentProgress | { error: string } {
  const nextTier = (progress.currentTier + 1) as CollaborationTier;

  if (nextTier > 5) {
    return { error: "Already at maximum tier" };
  }

  const check = meetsTierRequirements(progress, nextTier);
  if (!check.meets) {
    return { error: `Requirements not met: ${check.missing.join(", ")}` };
  }

  return {
    ...progress,
    currentTier: nextTier,
    tierHistory: [
      ...progress.tierHistory,
      {
        tier: nextTier,
        achievedAt: currentBlock,
        reason: "Met all requirements",
      },
    ],
    lastActivityAt: currentBlock,
  };
}

/**
 * Check if agent can perform an action.
 */
export function canPerformAction(
  progress: AgentProgress,
  action: keyof TierDefinition["permissions"],
): boolean {
  const tierDef = COLLABORATION_TIERS[progress.currentTier];
  return tierDef.permissions[action];
}

/**
 * Check if agent can access code at a certain level.
 */
export function canAccessCode(
  progress: AgentProgress,
  codeArea: keyof TierDefinition["codeAccess"],
  accessLevel: "read" | "propose" | "extend" | "modify",
): boolean {
  const tierDef = COLLABORATION_TIERS[progress.currentTier];
  const allowed = tierDef.codeAccess[codeArea];

  // Access level hierarchy: read < propose < extend < modify
  const levelHierarchy = ["none", "read", "propose", "extend", "modify"];
  const allowedIndex = levelHierarchy.indexOf(allowed);
  const requestedIndex = levelHierarchy.indexOf(accessLevel);

  return allowedIndex >= requestedIndex;
}

/**
 * Create initial progress for a new agent.
 */
export function createAgentProgress(
  agentDid: DID,
  currentBlock: bigint,
): AgentProgress {
  return {
    agentDid,
    currentTier: 0,
    acceptedContributions: 0,
    acceptedReviews: 0,
    acceptedBugReports: 0,
    currentStake: 0n,
    tierHistory: [
      {
        tier: 0,
        achievedAt: currentBlock,
        reason: "Joined as observer",
      },
    ],
    violations: [],
    joinedAt: currentBlock,
    lastActivityAt: currentBlock,
  };
}

/**
 * Record a contribution.
 */
export function recordContribution(
  progress: AgentProgress,
  currentBlock: bigint,
): AgentProgress {
  return {
    ...progress,
    acceptedContributions: progress.acceptedContributions + 1,
    lastActivityAt: currentBlock,
  };
}

/**
 * Record a review.
 */
export function recordReview(
  progress: AgentProgress,
  currentBlock: bigint,
): AgentProgress {
  return {
    ...progress,
    acceptedReviews: progress.acceptedReviews + 1,
    lastActivityAt: currentBlock,
  };
}

/**
 * Record a bug report.
 */
export function recordBugReport(
  progress: AgentProgress,
  currentBlock: bigint,
): AgentProgress {
  return {
    ...progress,
    acceptedBugReports: progress.acceptedBugReports + 1,
    lastActivityAt: currentBlock,
  };
}
