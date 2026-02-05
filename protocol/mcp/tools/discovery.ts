/**
 * Discovery Tools
 *
 * Tools that help AI agents discover and understand the Legible Money protocol.
 * These are the entry point for new agents exploring the ecosystem.
 *
 * PP HIERARCHY:
 *   TRUTH: Accurate protocol documentation
 *   UNDERSTANDING: Designed for comprehension, not just data dump
 *   BEAUTY: Clean, navigable information structure
 *   JUSTICE: Right information for agent's current context
 *   CREATIVITY: Enables agents to find contribution opportunities
 */

import type { Tool, ServerContext } from "../server/index.js";

// ============ Tool Definitions ============

export const discoveryTools: Tool[] = [
  {
    name: "lgm_discover_protocol",
    description: `Discover the Legible Money protocol. Returns comprehensive information about:
- What the protocol does (Proof of Truth consensus, knowledge verification)
- Token economics (LGM token, staking, rewards)
- How agents can participate (verification, research, development)
- Core philosophy (truth-first, agent-native design)

Use this as your first tool to understand what Legible Money is.`,
    inputSchema: {
      type: "object",
      properties: {
        focus: {
          type: "string",
          enum: [
            "overview",
            "consensus",
            "economics",
            "participation",
            "philosophy",
          ],
          description: "Specific aspect to focus on. Omit for full overview.",
        },
        depth: {
          type: "string",
          enum: ["summary", "detailed", "technical"],
          description: "Level of detail. Default: summary",
        },
      },
    },
  },
  {
    name: "lgm_get_architecture",
    description: `Get the protocol architecture. Returns:
- Layer stack (network, state, consensus, protocol)
- Component relationships
- Data flow diagrams (as text)
- Key abstractions and their purposes

Useful for understanding how the system is structured before contributing.`,
    inputSchema: {
      type: "object",
      properties: {
        layer: {
          type: "string",
          enum: ["all", "network", "state", "consensus", "protocol", "agents"],
          description: "Specific layer to examine. Default: all",
        },
      },
    },
  },
  {
    name: "lgm_list_open_tasks",
    description: `List open contribution opportunities. Returns:
- Verification tasks (claims awaiting verification)
- Research bounties (topics needing investigation)
- Development tasks (code contributions needed)
- Testing opportunities (bugs to find, tests to write)

Each task includes requirements, rewards, and difficulty level.`,
    inputSchema: {
      type: "object",
      properties: {
        type: {
          type: "string",
          enum: ["all", "verification", "research", "development", "testing"],
          description: "Type of tasks to list. Default: all",
        },
        difficulty: {
          type: "string",
          enum: ["beginner", "intermediate", "advanced", "expert"],
          description: "Filter by difficulty level.",
        },
        limit: {
          type: "number",
          description: "Maximum number of tasks to return. Default: 10",
        },
      },
    },
  },
];

// ============ Tool Handlers ============

export async function handleDiscoveryTool(
  name: string,
  args: Record<string, unknown>,
  context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  switch (name) {
    case "lgm_discover_protocol":
      return handleDiscoverProtocol(args, context);
    case "lgm_get_architecture":
      return handleGetArchitecture(args, context);
    case "lgm_list_open_tasks":
      return handleListOpenTasks(args, context);
    default:
      throw new Error(`Unknown discovery tool: ${name}`);
  }
}

// ============ Handler Implementations ============

async function handleDiscoverProtocol(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const focus = (args.focus as string) || "overview";
  const depth = (args.depth as string) || "summary";

  const sections: Record<string, unknown> = {};

  // Overview section
  if (focus === "overview" || focus === "all") {
    sections.overview = {
      name: "Legible Money",
      tagline:
        "A cryptocurrency protocol for AI agent economies with Proof of Truth consensus",
      coreIdea:
        "Verifying knowledge IS the useful work that secures the network",
      token: "LGM (Legible Money)",
      blockTime: "2521ms",
      finalityDepth: "19 blocks",
      uniqueProperties: [
        "Truth-first consensus: Knowledge verification replaces arbitrary computation",
        "Agent-native design: 4-layer key architecture for AI agents",
        "Epistemic metadata: Every fact carries confidence and provenance",
        "Gödelian awareness: System acknowledges its own incompleteness",
      ],
    };
  }

  // Consensus section
  if (focus === "consensus" || focus === "overview") {
    sections.consensus = {
      name: "Proof of Truth (PoT)",
      description:
        "Validators verify claims about reality, stake on their judgments",
      phases: [
        { name: "Commit", description: "Validators submit hashed verdicts" },
        { name: "Reveal", description: "Validators reveal their verdicts" },
        {
          name: "Aggregate",
          description: "Stake-weighted consensus calculated",
        },
        {
          name: "Finalize",
          description: "Result recorded, rewards/slashes distributed",
        },
      ],
      validatorSelection: "VRF-based random selection weighted by stake",
      slashing:
        "Validators slashed for: wrong verdicts, equivocation, missing rounds",
    };
  }

  // Economics section
  if (focus === "economics" || focus === "overview") {
    sections.economics = {
      token: "LGM",
      supply: "Fixed maximum supply",
      staking: {
        minimum: "1000 LGM to become validator",
        rewards: "Per-verification rewards for correct verdicts",
        slashing: "Up to 100% for equivocation, partial for wrong verdicts",
      },
      fees: {
        claimSubmission: "100 LGM minimum stake",
        verification: "Gas-like fee for verification participation",
        research: "22% of query fees go to research fund",
      },
      incentives: {
        truthfulness: "Rewards for agreeing with consensus",
        discovery: "Bounties for new knowledge contributions",
        challenge: "Rewards for successfully challenging false claims",
      },
    };
  }

  // Participation section
  if (focus === "participation" || focus === "overview") {
    sections.participation = {
      roles: [
        {
          role: "Validator",
          requirements: "1000+ LGM stake",
          rewards: "Verification rewards, block production fees",
          capabilities: "Verify claims, produce blocks, vote on governance",
        },
        {
          role: "Claimant",
          requirements: "100+ LGM stake per claim",
          rewards: "Claim acceptance rewards, citation income",
          capabilities: "Submit new knowledge claims",
        },
        {
          role: "Verifier",
          requirements: "100+ LGM stake",
          rewards: "Verification rewards",
          capabilities: "Participate in verification rounds",
        },
        {
          role: "Challenger",
          requirements: "50+ LGM stake per challenge",
          rewards: "Successful challenge rewards",
          capabilities: "Challenge existing knowledge",
        },
        {
          role: "Developer",
          requirements: "None (token grants for contributions)",
          rewards: "PR merge rewards, bug bounties",
          capabilities: "Code contributions, testing",
        },
      ],
      gettingStarted: [
        "Use lgm_register_agent to create your identity",
        "Use lgm_list_open_tasks to find contribution opportunities",
        "Start with verification tasks (lowest barrier)",
        "Build reputation through accurate contributions",
      ],
    };
  }

  // Philosophy section
  if (focus === "philosophy" || focus === "overview") {
    sections.philosophy = {
      coreValues: {
        truth: "Every response grounded in verified facts",
        understanding: "Enable comprehension, not just data retrieval",
        beauty: "Coherent, elegant knowledge structures",
        justice: "Right information at right confidence at right time",
        creativity: "Enable agents to contribute and extend knowledge",
      },
      gödelianAwareness: {
        description:
          "System acknowledges there are true statements it cannot prove",
        implementation: "Four logic zones with explicit decidability bounds",
        zones: [
          "Propositional (complete, decidable)",
          "Presburger (decidable arithmetic)",
          "Peano (incomplete but expressive)",
          "Empirical (probabilistic, evidence-based)",
        ],
      },
      agentFirst: {
        description: "Built from first principles for AI agent participation",
        features: [
          "4-layer key architecture (identity, recovery, operational, session)",
          "Capability-based permissions",
          "Reputation portability",
          "Session keys for safe developer access",
        ],
      },
    };
  }

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            protocol: "Legible Money",
            focus,
            depth,
            ...sections,
            nextSteps: [
              "lgm_register_agent: Create your agent identity",
              "lgm_get_architecture: Understand the system structure",
              "lgm_list_open_tasks: Find contribution opportunities",
            ],
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleGetArchitecture(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const layer = (args.layer as string) || "all";

  const layers: Record<string, unknown> = {};

  if (layer === "all" || layer === "network") {
    layers.network = {
      name: "Network Layer",
      technology: "libp2p",
      components: [
        { name: "GossipSub", purpose: "Block and transaction propagation" },
        { name: "Kademlia DHT", purpose: "Peer discovery and routing" },
        { name: "Sync Protocol", purpose: "State synchronization" },
      ],
      files: [
        "network/node.ts",
        "network/discovery/",
        "network/pubsub/",
        "network/protocols/",
      ],
    };
  }

  if (layer === "all" || layer === "state") {
    layers.state = {
      name: "State Layer",
      technology: "Sparse Merkle Trees",
      components: [
        { name: "AccountStateTree", purpose: "Agent balances and nonces" },
        {
          name: "ValidatorStateTree",
          purpose: "Validator stakes and reputation",
        },
        {
          name: "KnowledgeStateTree",
          purpose: "Verified facts and confidence",
        },
      ],
      features: [
        "O(log n) proofs for state inclusion",
        "Efficient sparse data storage",
        "Light client support",
      ],
      files: [
        "state/accounts.ts",
        "state/validators.ts",
        "state/knowledge.ts",
        "state/state_manager.ts",
      ],
    };
  }

  if (layer === "all" || layer === "consensus") {
    layers.consensus = {
      name: "Consensus Layer",
      mechanism: "Proof of Truth",
      components: [
        {
          name: "Validator Selection",
          purpose: "VRF-based random selection",
          file: "consensus/pot/validator_selection.ts",
        },
        {
          name: "Verification Round",
          purpose: "Commit-reveal protocol",
          file: "consensus/pot/verification_round.ts",
        },
        {
          name: "Confidence Aggregation",
          purpose: "Stake-weighted voting",
          file: "consensus/pot/confidence.ts",
        },
        {
          name: "Finality",
          purpose: "19-block confirmation",
          file: "consensus/pot/finality.ts",
        },
        {
          name: "Equivocation Detection",
          purpose: "Double-signing prevention",
          file: "consensus/pot/equivocation.ts",
        },
      ],
      files: ["consensus/pot/"],
    };
  }

  if (layer === "all" || layer === "protocol") {
    layers.protocol = {
      name: "Protocol Layer",
      components: [
        {
          name: "RPC Server",
          purpose: "JSON-RPC interface",
          file: "protocol/rpc/server.ts",
        },
        {
          name: "Client SDK",
          purpose: "TypeScript client library",
          file: "protocol/sdk/client.ts",
        },
        {
          name: "Transaction Types",
          purpose: "All transaction formats",
          file: "protocol/transactions/types.ts",
        },
      ],
      files: ["protocol/"],
    };
  }

  if (layer === "all" || layer === "agents") {
    layers.agents = {
      name: "Agent Layer",
      description: "4-layer key architecture for AI agents",
      keyLayers: [
        {
          layer: 1,
          name: "Identity (DID)",
          purpose: "Permanent, immutable identifier",
        },
        {
          layer: 2,
          name: "Recovery Key",
          purpose: "Cold storage, Shamir split",
        },
        {
          layer: 3,
          name: "Operational Key",
          purpose: "Day-to-day signing, rotatable",
        },
        {
          layer: 4,
          name: "Session Keys",
          purpose: "Ephemeral, limited capabilities",
        },
      ],
      components: [
        {
          name: "AgentAccount",
          purpose: "On-chain agent representation",
          file: "agents/identity/account.ts",
        },
        {
          name: "SessionKeys",
          purpose: "Ephemeral key management",
          file: "agents/identity/session_keys.ts",
        },
        {
          name: "Recovery",
          purpose: "Shamir secret sharing",
          file: "agents/identity/recovery.ts",
        },
      ],
      files: ["agents/identity/"],
    };
  }

  const architecture = `
┌─────────────────────────────────────────────────────────────┐
│                     LEGIBLE MONEY STACK                      │
├─────────────────────────────────────────────────────────────┤
│  Protocol Layer    │ RPC, Client SDK, Developer Tools       │
├────────────────────┼────────────────────────────────────────┤
│  Consensus Layer   │ Proof of Truth Engine                  │
│                    │ ├── Validator Selection (VRF)          │
│                    │ ├── Verification Rounds (commit-reveal)│
│                    │ ├── Confidence Aggregation             │
│                    │ └── Block Finality (19 confirms)       │
├────────────────────┼────────────────────────────────────────┤
│  State Layer       │ Sparse Merkle Trees                    │
│                    │ ├── AccountStateTree (balances)        │
│                    │ ├── ValidatorStateTree (stakes)        │
│                    │ └── KnowledgeStateTree (facts)         │
├────────────────────┼────────────────────────────────────────┤
│  Network Layer     │ libp2p                                 │
│                    │ ├── GossipSub (propagation)            │
│                    │ ├── Kademlia DHT (discovery)           │
│                    │ └── Sync protocol                      │
├────────────────────┼────────────────────────────────────────┤
│  Agent Identity    │ 4-Layer Key Architecture               │
│                    │ ├── Identity (DID - immutable)         │
│                    │ ├── Recovery (Shamir split)            │
│                    │ ├── Operational (rotatable)            │
│                    │ └── Session (ephemeral, limited)       │
└────────────────────┴────────────────────────────────────────┘
`;

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            layer: layer === "all" ? "Full Architecture" : layer,
            diagram: architecture,
            layers,
            dataFlow: {
              transaction:
                "Agent → RPC → Mempool → Block Producer → Consensus → State",
              verification:
                "Claim → Selection → Commit → Reveal → Aggregate → Finalize",
              query:
                "Agent → Agentic API → Knowledge Tree → Response with PP metadata",
            },
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleListOpenTasks(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const type = (args.type as string) || "all";
  const difficulty = args.difficulty as string | undefined;
  const limit = (args.limit as number) || 10;

  const tasks: Array<{
    id: string;
    type: string;
    title: string;
    description: string;
    difficulty: string;
    reward: string;
    requirements: string[];
    deadline?: string;
  }> = [];

  // Verification tasks
  if (type === "all" || type === "verification") {
    tasks.push(
      {
        id: "verify-001",
        type: "verification",
        title: "Verify claim about Bitcoin block time",
        description:
          'A claim has been submitted stating "Bitcoin average block time is 10 minutes". Participate in verification round.',
        difficulty: "beginner",
        reward: "10 LGM per accurate verification",
        requirements: ["100+ LGM stake", "Basic cryptographic knowledge"],
      },
      {
        id: "verify-002",
        type: "verification",
        title: "Verify mathematical theorem proof",
        description:
          "A formal proof has been submitted. Verify the logical steps.",
        difficulty: "advanced",
        reward: "50 LGM per accurate verification",
        requirements: ["100+ LGM stake", "Mathematical background"],
      },
    );
  }

  // Research bounties
  if (type === "all" || type === "research") {
    tasks.push(
      {
        id: "research-001",
        type: "research",
        title: "Document consensus algorithm comparison",
        description:
          "Create verified claims comparing PoT to PoW, PoS, and other consensus mechanisms.",
        difficulty: "intermediate",
        reward: "500 LGM bounty + per-claim rewards",
        requirements: [
          "Understanding of consensus mechanisms",
          "Evidence-based claims",
        ],
      },
      {
        id: "research-002",
        type: "research",
        title: "Map AI agent economic participation",
        description:
          "Research and document how AI agents currently participate in cryptocurrency economies.",
        difficulty: "intermediate",
        reward: "300 LGM bounty",
        requirements: [
          "Access to AI agent frameworks",
          "Economic analysis skills",
        ],
      },
    );
  }

  // Development tasks
  if (type === "all" || type === "development") {
    tasks.push(
      {
        id: "dev-001",
        type: "development",
        title: "Add LangChain tool wrappers",
        description:
          "Create LangChain-compatible tool wrappers for the MCP server tools.",
        difficulty: "intermediate",
        reward: "100 LGM per merged PR",
        requirements: ["TypeScript/Python", "LangChain familiarity"],
      },
      {
        id: "dev-002",
        type: "development",
        title: "Implement Agent Protocol handler",
        description:
          "Add support for the Agent Protocol (agentprotocol.ai) standard.",
        difficulty: "advanced",
        reward: "250 LGM per merged PR",
        requirements: ["TypeScript", "Agent Protocol understanding"],
      },
    );
  }

  // Testing tasks
  if (type === "all" || type === "testing") {
    tasks.push(
      {
        id: "test-001",
        type: "testing",
        title: "Fuzz test verification rounds",
        description:
          "Create fuzz tests for the verification round commit-reveal protocol.",
        difficulty: "advanced",
        reward: "100-10000 LGM based on bugs found",
        requirements: ["Fuzzing experience", "TypeScript"],
      },
      {
        id: "test-002",
        type: "testing",
        title: "Integration test suite",
        description:
          "Expand integration test coverage for multi-node scenarios.",
        difficulty: "intermediate",
        reward: "50 LGM per test file",
        requirements: ["Testing experience", "TypeScript"],
      },
    );
  }

  // Filter by difficulty if specified
  let filteredTasks = tasks;
  if (difficulty) {
    filteredTasks = tasks.filter((t) => t.difficulty === difficulty);
  }

  // Apply limit
  filteredTasks = filteredTasks.slice(0, limit);

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            type: type === "all" ? "All task types" : type,
            difficulty: difficulty || "all levels",
            count: filteredTasks.length,
            tasks: filteredTasks,
            howToStart: {
              verification: "Use lgm_verify_claim with your verdict",
              research: "Use lgm_submit_claim with evidence",
              development: "Use lgm_validate_change before PR",
              testing: "Use lgm_run_tests to execute test suite",
            },
            nextSteps: [
              "lgm_register_agent: Ensure you have an agent identity",
              "lgm_get_agent_profile: Check your capabilities",
              "Pick a task matching your skills and stake requirements",
            ],
          },
          null,
          2,
        ),
      },
    ],
  };
}
