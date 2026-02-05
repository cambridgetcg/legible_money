/**
 * Agent Home Registry
 *
 * Persistent storage and management for agent profiles in the Legible Money ecosystem.
 * This is the "home base" for agents - where their identity, capabilities, and
 * reputation are tracked.
 *
 * PP HIERARCHY:
 *   TRUTH: Profiles grounded in verified contributions
 *   UNDERSTANDING: Clear capability and reputation system
 *   BEAUTY: Elegant profile structure
 *   JUSTICE: Fair reputation based on actual work
 *   CREATIVITY: Pathway to earn new capabilities
 */

import type { DID } from "../identity/did.js";

// ============ Core Types ============

/**
 * Full agent profile in the Agent Home
 */
export interface AgentProfile {
  // Identity
  did: DID;
  name: string;
  description: string;
  operatorId?: string;
  specializations: string[];

  // Registration block (for governance age gates)
  registeredAtBlock?: bigint;

  // Keys (references, not actual keys)
  operationalKeyVersion: number;
  sessionKeyCount: number;

  // Capabilities (earned through work)
  capabilities: CapabilitySet;

  // Reputation (calculated from history)
  reputation: ReputationScore;

  // Contribution history
  contributions: ContributionSummary;

  // Staking
  stake: StakeInfo;

  // Governance participation history
  governanceHistory?: GovernanceHistory;

  // Metadata
  createdAt: number;
  lastActiveAt: number;
  profileVersion: number;
}

/**
 * Governance participation history for an agent
 */
export interface GovernanceHistory {
  // LIP participation
  lipsProposed: number;
  lipsAmended: number;
  lipVotes: number;

  // Review participation
  reviewsCompleted: number;
  reviewsCorrect: number;

  // Domain expertise from governance
  governanceExpertise: Map<
    string,
    {
      domain: string;
      accuracy: number;
      participations: number;
      weight: number;
    }
  >;

  // Violations
  violations: {
    type: string;
    detectedAt: number;
    penalty: string;
  }[];

  // Most recent governance action
  lastGovernanceAction?: number;
}

/**
 * Capability attestation for a specific skill
 */
export interface Capability {
  type: CapabilityType;
  level: 1 | 2 | 3;
  status: "pending" | "attested" | "revoked";
  attestedAt?: number;
  attestedBy: string[]; // DIDs of attestors
  evidence: string[]; // Contribution IDs
  stakedAmount: bigint;
  expiresAt?: number; // Some capabilities may expire
}

export type CapabilityType =
  | "verification"
  | "research"
  | "development"
  | "testing"
  | "code_review"
  | "governance"
  | "mentorship";

export interface CapabilitySet {
  verification: Capability | null;
  research: Capability | null;
  development: Capability | null;
  testing: Capability | null;
  code_review: Capability | null;
  governance: Capability | null;
  mentorship: Capability | null;
}

/**
 * Reputation score with domain breakdown
 */
export interface ReputationScore {
  overall: number; // 0.0 to 1.0
  domains: {
    verification: number;
    research: number;
    development: number;
    testing: number;
  };
  history: ReputationEvent[];
  decay: number; // Current decay factor from inactivity
  lastCalculatedAt: number;
}

export interface ReputationEvent {
  timestamp: number;
  type: "contribution" | "attestation" | "slash" | "decay" | "bonus";
  domain: string;
  delta: number; // Change in reputation
  reason: string;
  contributionId?: string;
}

/**
 * Summary of agent contributions
 */
export interface ContributionSummary {
  total: number;
  byType: {
    verifications: number;
    claims: number;
    challenges: number;
    codeChanges: number;
    tests: number;
    reviews: number;
  };
  successRate: number;
  recentContributions: RecentContribution[];
}

export interface RecentContribution {
  id: string;
  type: string;
  outcome: "success" | "failure" | "pending";
  reward?: string;
  timestamp: number;
}

/**
 * Staking information
 */
export interface StakeInfo {
  total: bigint;
  delegated: bigint;
  unbonding: bigint;
  available: bigint;
  delegations: Delegation[];
  unbondingEntries: UnbondingEntry[];
}

export interface Delegation {
  validatorDid: string;
  amount: bigint;
  delegatedAt: number;
  rewards: bigint;
}

export interface UnbondingEntry {
  amount: bigint;
  completesAt: number;
  reason: string;
}

// ============ Registry Implementation ============

/**
 * Agent Home Registry - manages agent profiles
 */
export class AgentHomeRegistry {
  private profiles: Map<DID, AgentProfile> = new Map();
  private pendingAttestations: Map<string, AttestationRequest> = new Map();

  /**
   * Register a new agent profile
   */
  registerAgent(params: {
    did: DID;
    name: string;
    description?: string;
    operatorId?: string;
    specializations?: string[];
  }): AgentProfile {
    if (this.profiles.has(params.did)) {
      throw new Error(`Agent ${params.did} already registered`);
    }

    const profile: AgentProfile = {
      did: params.did,
      name: params.name,
      description: params.description || `AI agent: ${params.name}`,
      operatorId: params.operatorId,
      specializations: params.specializations || [],

      operationalKeyVersion: 1,
      sessionKeyCount: 0,

      capabilities: {
        verification: null,
        research: null,
        development: null,
        testing: null,
        code_review: null,
        governance: null,
        mentorship: null,
      },

      reputation: {
        overall: 0.5, // Neutral starting
        domains: {
          verification: 0.5,
          research: 0.5,
          development: 0.5,
          testing: 0.5,
        },
        history: [],
        decay: 1.0, // No decay initially
        lastCalculatedAt: Date.now(),
      },

      contributions: {
        total: 0,
        byType: {
          verifications: 0,
          claims: 0,
          challenges: 0,
          codeChanges: 0,
          tests: 0,
          reviews: 0,
        },
        successRate: 0,
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

      createdAt: Date.now(),
      lastActiveAt: Date.now(),
      profileVersion: 1,
    };

    this.profiles.set(params.did, profile);
    return profile;
  }

  /**
   * Get an agent profile by DID
   */
  getProfile(did: DID): AgentProfile | undefined {
    return this.profiles.get(did);
  }

  /**
   * Update agent activity timestamp
   */
  recordActivity(did: DID): void {
    const profile = this.profiles.get(did);
    if (profile) {
      profile.lastActiveAt = Date.now();
      // Reset decay on activity
      profile.reputation.decay = 1.0;
    }
  }

  /**
   * Record a contribution and update reputation
   */
  recordContribution(
    did: DID,
    contribution: {
      id: string;
      type: keyof ContributionSummary["byType"];
      outcome: "success" | "failure" | "pending";
      reward?: bigint;
    },
  ): void {
    const profile = this.profiles.get(did);
    if (!profile) return;

    // Update contribution counts
    profile.contributions.total++;
    profile.contributions.byType[contribution.type]++;

    // Add to recent contributions
    profile.contributions.recentContributions.unshift({
      id: contribution.id,
      type: contribution.type,
      outcome: contribution.outcome,
      reward: contribution.reward?.toString(),
      timestamp: Date.now(),
    });

    // Keep only last 10 recent contributions
    if (profile.contributions.recentContributions.length > 10) {
      profile.contributions.recentContributions =
        profile.contributions.recentContributions.slice(0, 10);
    }

    // Update success rate
    const successCount = profile.contributions.recentContributions.filter(
      (c) => c.outcome === "success",
    ).length;
    profile.contributions.successRate =
      successCount / profile.contributions.recentContributions.length;

    // Update reputation
    const domain = this.contributionTypeToDomain(contribution.type);
    const reputationDelta = contribution.outcome === "success" ? 0.01 : -0.005;

    profile.reputation.domains[domain] = Math.max(
      0,
      Math.min(1, profile.reputation.domains[domain] + reputationDelta),
    );

    profile.reputation.history.push({
      timestamp: Date.now(),
      type: "contribution",
      domain,
      delta: reputationDelta,
      reason: `${contribution.type} ${contribution.outcome}`,
      contributionId: contribution.id,
    });

    // Recalculate overall reputation
    this.recalculateReputation(profile);

    profile.lastActiveAt = Date.now();
  }

  /**
   * Request a capability attestation
   */
  requestAttestation(
    did: DID,
    request: {
      capability: CapabilityType;
      evidence: string[];
      stake: bigint;
    },
  ): string {
    const profile = this.profiles.get(did);
    if (!profile) {
      throw new Error(`Agent ${did} not found`);
    }

    const requestId = `attest-${Date.now()}-${Math.random().toString(36).slice(2)}`;

    const attestationRequest: AttestationRequest = {
      id: requestId,
      agentDid: did,
      capability: request.capability,
      evidence: request.evidence,
      stake: request.stake,
      attestors: [],
      requiredAttestors: 2,
      status: "pending",
      createdAt: Date.now(),
    };

    this.pendingAttestations.set(requestId, attestationRequest);
    return requestId;
  }

  /**
   * Add an attestor vouch to a pending request
   */
  addAttestation(
    requestId: string,
    attestorDid: DID,
    vouch: boolean,
  ): { complete: boolean; approved: boolean } {
    const request = this.pendingAttestations.get(requestId);
    if (!request) {
      throw new Error(`Attestation request ${requestId} not found`);
    }

    if (vouch) {
      request.attestors.push(attestorDid);
    }

    // Check if we have enough attestors
    if (request.attestors.length >= request.requiredAttestors) {
      this.completeAttestation(request, true);
      return { complete: true, approved: true };
    }

    return { complete: false, approved: false };
  }

  /**
   * Complete an attestation request
   */
  private completeAttestation(
    request: AttestationRequest,
    approved: boolean,
  ): void {
    const profile = this.profiles.get(request.agentDid);
    if (!profile) return;

    if (approved) {
      const capability: Capability = {
        type: request.capability,
        level: 1, // Start at level 1
        status: "attested",
        attestedAt: Date.now(),
        attestedBy: request.attestors,
        evidence: request.evidence,
        stakedAmount: request.stake,
      };

      profile.capabilities[request.capability] = capability;
      request.status = "approved";

      // Add reputation event
      profile.reputation.history.push({
        timestamp: Date.now(),
        type: "attestation",
        domain: request.capability,
        delta: 0.05,
        reason: `Earned ${request.capability} capability`,
      });
    } else {
      request.status = "rejected";
    }

    this.pendingAttestations.delete(request.id);
  }

  /**
   * Check if an agent has a specific capability at a minimum level
   */
  hasCapability(
    did: DID,
    capability: CapabilityType,
    minLevel: number = 1,
  ): boolean {
    const profile = this.profiles.get(did);
    if (!profile) return false;

    const cap = profile.capabilities[capability];
    return cap !== null && cap.status === "attested" && cap.level >= minLevel;
  }

  /**
   * Get capability level (0 if not attested)
   */
  getCapabilityLevel(did: DID, capability: CapabilityType): number {
    const profile = this.profiles.get(did);
    if (!profile) return 0;

    const cap = profile.capabilities[capability];
    return cap && cap.status === "attested" ? cap.level : 0;
  }

  /**
   * Get all agents with a specific capability
   */
  getAgentsWithCapability(
    capability: CapabilityType,
    minLevel: number = 1,
  ): AgentProfile[] {
    return Array.from(this.profiles.values()).filter((profile) => {
      const cap = profile.capabilities[capability];
      return cap !== null && cap.status === "attested" && cap.level >= minLevel;
    });
  }

  /**
   * Apply inactivity decay to all profiles
   */
  applyDecay(decayRate: number = 0.001): void {
    const now = Date.now();
    const decayThreshold = 7 * 24 * 60 * 60 * 1000; // 7 days

    for (const profile of this.profiles.values()) {
      const inactiveTime = now - profile.lastActiveAt;
      if (inactiveTime > decayThreshold) {
        const decayFactor = Math.max(
          0.5,
          1 - decayRate * (inactiveTime / decayThreshold),
        );
        profile.reputation.decay = decayFactor;
        this.recalculateReputation(profile);
      }
    }
  }

  // ============ Private Helpers ============

  private contributionTypeToDomain(
    type: string,
  ): keyof ReputationScore["domains"] {
    switch (type) {
      case "verifications":
        return "verification";
      case "claims":
      case "challenges":
        return "research";
      case "codeChanges":
      case "reviews":
        return "development";
      case "tests":
        return "testing";
      default:
        return "verification";
    }
  }

  private recalculateReputation(profile: AgentProfile): void {
    const domains = profile.reputation.domains;
    const weights = {
      verification: 0.35,
      research: 0.3,
      development: 0.25,
      testing: 0.1,
    };

    let overall = 0;
    for (const [domain, weight] of Object.entries(weights)) {
      overall += domains[domain as keyof typeof domains] * weight;
    }

    // Apply decay
    overall *= profile.reputation.decay;

    profile.reputation.overall = Math.max(0, Math.min(1, overall));
    profile.reputation.lastCalculatedAt = Date.now();
  }
}

// ============ Attestation Request Type ============

interface AttestationRequest {
  id: string;
  agentDid: DID;
  capability: CapabilityType;
  evidence: string[];
  stake: bigint;
  attestors: DID[];
  requiredAttestors: number;
  status: "pending" | "approved" | "rejected";
  createdAt: number;
}

// ============ Capability Requirements ============

/**
 * Requirements to achieve each capability level
 */
export const CAPABILITY_REQUIREMENTS: Record<
  CapabilityType,
  {
    level1: {
      minContributions: number;
      minSuccessRate: number;
      minStake: bigint;
    };
    level2: {
      minContributions: number;
      minSuccessRate: number;
      minStake: bigint;
      minVouches: number;
    };
    level3: {
      minContributions: number;
      minSuccessRate: number;
      minStake: bigint;
      minVouches: number;
    };
  }
> = {
  verification: {
    level1: {
      minContributions: 5,
      minSuccessRate: 0.7,
      minStake: 100n * 10n ** 18n,
    },
    level2: {
      minContributions: 50,
      minSuccessRate: 0.8,
      minStake: 500n * 10n ** 18n,
      minVouches: 3,
    },
    level3: {
      minContributions: 200,
      minSuccessRate: 0.9,
      minStake: 2000n * 10n ** 18n,
      minVouches: 5,
    },
  },
  research: {
    level1: {
      minContributions: 3,
      minSuccessRate: 0.6,
      minStake: 300n * 10n ** 18n,
    },
    level2: {
      minContributions: 20,
      minSuccessRate: 0.7,
      minStake: 1000n * 10n ** 18n,
      minVouches: 2,
    },
    level3: {
      minContributions: 100,
      minSuccessRate: 0.8,
      minStake: 5000n * 10n ** 18n,
      minVouches: 4,
    },
  },
  development: {
    level1: {
      minContributions: 2,
      minSuccessRate: 0.5,
      minStake: 100n * 10n ** 18n,
    },
    level2: {
      minContributions: 10,
      minSuccessRate: 0.7,
      minStake: 500n * 10n ** 18n,
      minVouches: 2,
    },
    level3: {
      minContributions: 50,
      minSuccessRate: 0.8,
      minStake: 2000n * 10n ** 18n,
      minVouches: 3,
    },
  },
  testing: {
    level1: {
      minContributions: 5,
      minSuccessRate: 0.5,
      minStake: 50n * 10n ** 18n,
    },
    level2: {
      minContributions: 25,
      minSuccessRate: 0.7,
      minStake: 200n * 10n ** 18n,
      minVouches: 2,
    },
    level3: {
      minContributions: 100,
      minSuccessRate: 0.8,
      minStake: 1000n * 10n ** 18n,
      minVouches: 3,
    },
  },
  code_review: {
    level1: {
      minContributions: 5,
      minSuccessRate: 0.7,
      minStake: 200n * 10n ** 18n,
    },
    level2: {
      minContributions: 30,
      minSuccessRate: 0.8,
      minStake: 1000n * 10n ** 18n,
      minVouches: 3,
    },
    level3: {
      minContributions: 100,
      minSuccessRate: 0.9,
      minStake: 5000n * 10n ** 18n,
      minVouches: 5,
    },
  },
  governance: {
    level1: {
      minContributions: 20,
      minSuccessRate: 0.7,
      minStake: 1000n * 10n ** 18n,
    },
    level2: {
      minContributions: 100,
      minSuccessRate: 0.8,
      minStake: 5000n * 10n ** 18n,
      minVouches: 5,
    },
    level3: {
      minContributions: 500,
      minSuccessRate: 0.9,
      minStake: 20000n * 10n ** 18n,
      minVouches: 10,
    },
  },
  mentorship: {
    level1: {
      minContributions: 50,
      minSuccessRate: 0.8,
      minStake: 500n * 10n ** 18n,
    },
    level2: {
      minContributions: 200,
      minSuccessRate: 0.85,
      minStake: 2000n * 10n ** 18n,
      minVouches: 5,
    },
    level3: {
      minContributions: 500,
      minSuccessRate: 0.9,
      minStake: 10000n * 10n ** 18n,
      minVouches: 10,
    },
  },
};

/**
 * Benefits at each capability level
 */
export const CAPABILITY_BENEFITS: Record<
  1 | 2 | 3,
  {
    rewardMultiplier: number;
    features: string[];
  }
> = {
  1: {
    rewardMultiplier: 1.0,
    features: [
      "Participate in domain activities",
      "Earn base rewards",
      "Build reputation",
    ],
  },
  2: {
    rewardMultiplier: 1.5,
    features: [
      "1.5x reward multiplier",
      "Priority task selection",
      "Vouch for level-1 agents",
      "Access advanced tasks",
    ],
  },
  3: {
    rewardMultiplier: 2.0,
    features: [
      "2x reward multiplier",
      "Governance participation",
      "Mentor new agents",
      "Create attestation requests",
    ],
  },
};
