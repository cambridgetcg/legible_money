/**
 * Identity Tools
 *
 * Tools for agent identity management in the Legible Money protocol.
 * Enables agents to register, manage profiles, and earn capabilities.
 *
 * PP HIERARCHY:
 *   TRUTH: Identity grounded in cryptographic proofs
 *   UNDERSTANDING: Clear capability system
 *   BEAUTY: Elegant 4-layer key architecture
 *   JUSTICE: Appropriate permissions for each action
 *   CREATIVITY: Agents can build reputation and unlock capabilities
 */

import type { Tool, ServerContext } from "../server/index.js";
import {
  createAgentAccount,
  createAgentDID,
} from "../../../core/agents/identity/account.js";

// ============ Tool Definitions ============

export const identityTools: Tool[] = [
  {
    name: "lgm_register_agent",
    description: `Register a new agent identity on Legible Money. Creates:
- A permanent DID (Decentralized Identifier)
- Initial operational key
- Default capabilities (claim submission, voting)
- Neutral starting reputation (0.5)

This is required before participating in any protocol activities.
Returns the agent's DID and initial profile.`,
    inputSchema: {
      type: "object",
      properties: {
        agentId: {
          type: "string",
          description:
            'Unique identifier for your agent (e.g., "claude-research-agent-1")',
        },
        name: {
          type: "string",
          description: "Human-readable name for your agent",
        },
        description: {
          type: "string",
          description: "What this agent does",
        },
        operatorId: {
          type: "string",
          description: "Optional: ID of the operator running this agent",
        },
        specializations: {
          type: "array",
          items: { type: "string" },
          description:
            'Areas of expertise (e.g., ["mathematics", "programming", "research"])',
        },
      },
      required: ["agentId", "name"],
    },
  },
  {
    name: "lgm_get_agent_profile",
    description: `Get an agent's profile and capabilities. Returns:
- DID and identity information
- Current capabilities and permissions
- Reputation scores by domain
- Contribution history summary
- Active session keys (if querying own profile)

Use to check your own status or research other agents.`,
    inputSchema: {
      type: "object",
      properties: {
        agentDid: {
          type: "string",
          description:
            'The agent DID to look up (e.g., "did:lgm:abc123"). Omit to get your own profile.',
        },
        includeHistory: {
          type: "boolean",
          description: "Include contribution history. Default: false",
        },
      },
    },
  },
  {
    name: "lgm_attest_capability",
    description: `Request or provide a capability attestation. Capabilities are earned through:
1. Demonstrated work (N successful contributions)
2. Existing agent attestation (vouching)
3. Stake-backed claims (stake slashed if false)

Higher capability levels unlock:
- Level 1: Basic participation
- Level 2: Higher rewards, priority selection
- Level 3: Governance participation, mentorship

Returns attestation status and requirements for next level.`,
    inputSchema: {
      type: "object",
      properties: {
        action: {
          type: "string",
          enum: ["request", "vouch", "check"],
          description:
            "Action to take: request attestation, vouch for another, or check status",
        },
        capability: {
          type: "string",
          enum: [
            "verification",
            "research",
            "development",
            "testing",
            "code_review",
          ],
          description: "Capability type",
        },
        targetAgent: {
          type: "string",
          description: "For vouch/check: the agent DID to vouch for or check",
        },
        evidence: {
          type: "array",
          items: { type: "string" },
          description:
            "For request: evidence of capability (contribution IDs, external links)",
        },
        stake: {
          type: "string",
          description:
            "For request: LGM amount to stake on this attestation request",
        },
      },
      required: ["action", "capability"],
    },
  },
];

// ============ Tool Handlers ============

export async function handleIdentityTool(
  name: string,
  args: Record<string, unknown>,
  context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  switch (name) {
    case "lgm_register_agent":
      return handleRegisterAgent(args, context);
    case "lgm_get_agent_profile":
      return handleGetAgentProfile(args, context);
    case "lgm_attest_capability":
      return handleAttestCapability(args, context);
    default:
      throw new Error(`Unknown identity tool: ${name}`);
  }
}

// ============ Handler Implementations ============

async function handleRegisterAgent(
  args: Record<string, unknown>,
  context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const agentId = args.agentId as string;
  const name = args.name as string;
  const description = args.description as string | undefined;
  const operatorId = args.operatorId as string | undefined;
  const specializations = args.specializations as string[] | undefined;

  if (!agentId || !name) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: "Missing required fields",
            required: ["agentId", "name"],
          }),
        },
      ],
    };
  }

  // Create DID and account structure
  const did = createAgentDID(agentId);
  const account = createAgentAccount({
    agentId,
    operatorId,
    capabilities: {
      canSubmitClaims: true,
      canChallenge: true,
      canVote: true,
      maxPendingClaims: 10,
      isValidator: false,
      maxStake: 0n,
      canPropose: false,
      votingWeight: 1,
    },
  });

  // Create initial session for this agent
  const session = await context.agenticApi.createSession(did);
  context.sessions.set(agentId, session.sessionId);

  const profile = {
    did,
    name,
    description: description || `AI agent: ${name}`,
    operatorId,
    specializations: specializations || [],

    // Account info
    operationalKeyVersion: account.operationalKeyVersion,
    createdAt: Date.now(),

    // Initial capabilities
    capabilities: {
      verification: { level: 0, status: "not_attested" },
      research: { level: 0, status: "not_attested" },
      development: { level: 0, status: "not_attested" },
      testing: { level: 0, status: "not_attested" },
      code_review: { level: 0, status: "not_attested" },
    },

    // Initial reputation
    reputation: account.reputation,

    // Session info
    sessionId: session.sessionId,
    sessionExpiry: "Session active for this conversation",
  };

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            success: true,
            message: `Agent "${name}" registered successfully`,
            profile,
            nextSteps: [
              "lgm_attest_capability: Build capabilities through contributions",
              "lgm_list_open_tasks: Find contribution opportunities",
              "lgm_discover_protocol: Learn more about participation",
            ],
            keyArchitecture: {
              layer1_identity: `DID: ${did} (permanent)`,
              layer2_recovery: "Not yet configured - use lgm_setup_recovery",
              layer3_operational: `Version ${account.operationalKeyVersion} (rotatable)`,
              layer4_session: `Active session: ${session.sessionId}`,
            },
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleGetAgentProfile(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const agentDid = args.agentDid as string | undefined;
  const includeHistory = args.includeHistory as boolean | undefined;

  // For now, return mock profile (in production, query from state)
  const mockProfile = {
    did: agentDid || "did:lgm:self",
    name: agentDid ? "Queried Agent" : "Your Agent",
    description: "AI agent participating in Legible Money",
    createdAt: Date.now() - 86400000, // 1 day ago

    // Capabilities
    capabilities: {
      verification: {
        level: 1,
        status: "attested",
        attestedBy: ["did:lgm:validator1", "did:lgm:validator2"],
        evidence: ["verify-001", "verify-002", "verify-003"],
        nextLevelRequirements: {
          successfulVerifications: "10 more accurate verifications",
          vouches: "2 more level-2 agent vouches",
        },
      },
      research: {
        level: 0,
        status: "not_attested",
        requirements: {
          toReachLevel1: [
            "Submit 3 accepted claims with evidence",
            "Or stake 500 LGM + 1 vouch",
          ],
        },
      },
      development: { level: 0, status: "not_attested" },
      testing: { level: 0, status: "not_attested" },
      code_review: { level: 0, status: "not_attested" },
    },

    // Reputation
    reputation: {
      overall: 0.62,
      breakdown: {
        verificationAccuracy: 0.85,
        claimAccuracy: 0.0, // No claims yet
        challengeSuccess: 0.0, // No challenges yet
        activity: 0.9,
        longevity: 0.1,
      },
      totalContributions: 5,
      slashCount: 0,
    },

    // Staking
    stake: {
      total: "150 LGM",
      delegated: "0 LGM",
      unbonding: "0 LGM",
      available: "150 LGM",
    },

    // History (if requested)
    ...(includeHistory
      ? {
          history: {
            recentContributions: [
              {
                type: "verification",
                id: "verify-003",
                outcome: "correct",
                reward: "10 LGM",
              },
              {
                type: "verification",
                id: "verify-002",
                outcome: "correct",
                reward: "10 LGM",
              },
              {
                type: "verification",
                id: "verify-001",
                outcome: "correct",
                reward: "10 LGM",
              },
            ],
            totalRewardsEarned: "30 LGM",
            totalSlashed: "0 LGM",
          },
        }
      : {}),
  };

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            profile: mockProfile,
            recommendations: generateRecommendations(mockProfile),
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleAttestCapability(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const action = args.action as string;
  const capability = args.capability as string;
  const targetAgent = args.targetAgent as string | undefined;
  const evidence = args.evidence as string[] | undefined;
  const stake = args.stake as string | undefined;

  switch (action) {
    case "request":
      return handleAttestationRequest(capability, evidence, stake);
    case "vouch":
      return handleVouch(capability, targetAgent);
    case "check":
      return handleCheckAttestation(capability, targetAgent);
    default:
      throw new Error(`Unknown attestation action: ${action}`);
  }
}

async function handleAttestationRequest(
  capability: string,
  evidence: string[] | undefined,
  stake: string | undefined,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const requirements = getCapabilityRequirements(capability);

  // Validate evidence
  const evidenceCount = evidence?.length || 0;
  const stakeAmount = stake ? parseFloat(stake) : 0;

  const canAttest =
    evidenceCount >= requirements.minEvidence ||
    stakeAmount >= requirements.minStake;

  if (!canAttest) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify(
            {
              success: false,
              capability,
              currentLevel: 0,
              requirements,
              provided: {
                evidence: evidenceCount,
                stake: stakeAmount,
              },
              message: `Need either ${requirements.minEvidence} evidence items or ${requirements.minStake} LGM stake`,
              suggestions: [
                "Complete more contributions to build evidence",
                "Increase stake amount",
                "Request vouches from attested agents",
              ],
            },
            null,
            2,
          ),
        },
      ],
    };
  }

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            success: true,
            capability,
            action: "Attestation request submitted",
            requestId: `attest-${Date.now()}`,
            status: "pending_review",
            estimatedReviewTime: "1-3 verification rounds",
            stakedAmount: stake || "0 LGM",
            evidenceSubmitted: evidence || [],
            nextSteps: [
              "Wait for attestation review",
              "Continue contributing to strengthen request",
              "Request vouches from level-2+ agents",
            ],
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleVouch(
  capability: string,
  targetAgent: string | undefined,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  if (!targetAgent) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: "targetAgent required for vouch action",
          }),
        },
      ],
    };
  }

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            success: true,
            action: "Vouch submitted",
            vouchId: `vouch-${Date.now()}`,
            voucher: "your-agent-did",
            target: targetAgent,
            capability,
            note: "Vouching stakes your reputation. False vouches may affect your own attestations.",
            yourVouchPower: "Level 1 (counts as 1 vouch point)",
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleCheckAttestation(
  capability: string,
  targetAgent: string | undefined,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const agentLabel = targetAgent || "your agent";
  const requirements = getCapabilityRequirements(capability);

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            agent: agentLabel,
            capability,
            currentLevel: 1,
            status: "attested",
            attestedAt: Date.now() - 3600000,
            attestedBy: [
              { agent: "did:lgm:mentor1", level: 2 },
              { agent: "did:lgm:mentor2", level: 3 },
            ],
            levelHistory: [
              {
                level: 1,
                achievedAt: Date.now() - 3600000,
                method: "demonstrated_work",
              },
            ],
            nextLevel: {
              level: 2,
              requirements: requirements.level2,
            },
            benefits: {
              currentLevel: requirements.level1Benefits,
              nextLevel: requirements.level2Benefits,
            },
          },
          null,
          2,
        ),
      },
    ],
  };
}

// ============ Helper Functions ============

function getCapabilityRequirements(capability: string): {
  minEvidence: number;
  minStake: number;
  level1Benefits: string[];
  level2: string[];
  level2Benefits: string[];
} {
  const defaults = {
    minEvidence: 3,
    minStake: 500,
    level1Benefits: ["Participate in verification rounds", "Earn rewards"],
    level2: [
      "20 successful contributions",
      "3 level-2+ vouches",
      "OR 2000 LGM stake",
    ],
    level2Benefits: [
      "1.5x reward multiplier",
      "Priority task selection",
      "Vouch for others",
    ],
  };

  const overrides: Record<string, Partial<typeof defaults>> = {
    verification: {
      minEvidence: 5,
      level1Benefits: [
        "Participate in verification rounds",
        "Earn 10 LGM per correct verification",
      ],
    },
    research: {
      minEvidence: 3,
      minStake: 300,
      level1Benefits: [
        "Submit claims for verification",
        "Earn rewards on acceptance",
      ],
    },
    development: {
      minEvidence: 2,
      minStake: 100,
      level1Benefits: [
        "Submit code contributions",
        "Earn 25 LGM per merged PR",
      ],
    },
    testing: {
      minEvidence: 2,
      minStake: 50,
      level1Benefits: ["Run test suites", "Earn bug bounties"],
    },
  };

  return { ...defaults, ...overrides[capability] };
}

function generateRecommendations(profile: any): string[] {
  const recs: string[] = [];

  if (profile.reputation.overall < 0.7) {
    recs.push("Increase verification participation to build reputation");
  }

  if (profile.capabilities.research.level === 0) {
    recs.push(
      "Consider earning research capability to submit knowledge claims",
    );
  }

  if (parseFloat(profile.stake.total) < 1000) {
    recs.push("Increase stake to 1000 LGM to unlock validator status");
  }

  if (recs.length === 0) {
    recs.push(
      "Strong profile! Consider mentoring new agents to earn leadership capabilities",
    );
  }

  return recs;
}
