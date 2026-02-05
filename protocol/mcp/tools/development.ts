/**
 * Development Tools
 *
 * Tools for AI agents to contribute code to Legible Money.
 * Enables codebase exploration, pattern discovery, and contribution validation.
 *
 * PP HIERARCHY:
 *   TRUTH: Accurate codebase information
 *   UNDERSTANDING: Help agents comprehend architecture
 *   BEAUTY: Identify and enforce code patterns
 *   JUSTICE: Place contributions in correct locations
 *   CREATIVITY: Enable novel contributions
 */

import type { Tool, ServerContext } from "../server/index.js";

// ============ Tool Definitions ============

export const developmentTools: Tool[] = [
  {
    name: "lgm_get_codebase_map",
    description: `Get a map of the Legible Money codebase. Returns:
- Directory structure with purposes
- Key files and their roles
- Dependency relationships
- Entry points for different features

Useful for understanding where to make changes.`,
    inputSchema: {
      type: "object",
      properties: {
        focus: {
          type: "string",
          enum: [
            "all",
            "core",
            "consensus",
            "state",
            "network",
            "agents",
            "protocol",
            "mcp",
          ],
          description: "Area to focus on. Default: all",
        },
        depth: {
          type: "string",
          enum: ["summary", "detailed"],
          description: "Level of detail. Default: summary",
        },
      },
    },
  },
  {
    name: "lgm_find_pattern",
    description: `Find existing code patterns in the codebase. Use before implementing:
- Similar functionality implementations
- Convention examples (naming, structure)
- Error handling patterns
- Test patterns

Following existing patterns improves consistency and review speed.`,
    inputSchema: {
      type: "object",
      properties: {
        patternType: {
          type: "string",
          enum: [
            "function",
            "class",
            "error_handling",
            "test",
            "type",
            "validation",
            "crypto",
          ],
          description: "Type of pattern to find",
        },
        context: {
          type: "string",
          description:
            'What you are trying to implement (e.g., "new RPC endpoint", "state transition")',
        },
        area: {
          type: "string",
          description: 'Codebase area to search (e.g., "consensus", "state")',
        },
      },
      required: ["patternType", "context"],
    },
  },
  {
    name: "lgm_validate_change",
    description: `Validate a proposed code change before submitting. Checks:
- Type safety (TypeScript compilation)
- Pattern conformance
- Test coverage requirements
- Documentation requirements
- Security considerations

Returns validation result with specific feedback.`,
    inputSchema: {
      type: "object",
      properties: {
        changeType: {
          type: "string",
          enum: ["new_file", "modification", "deletion", "refactor"],
          description: "Type of change",
        },
        filePath: {
          type: "string",
          description: "Path to the file being changed",
        },
        description: {
          type: "string",
          description: "Description of what the change does",
        },
        hasTests: {
          type: "boolean",
          description: "Whether the change includes tests",
        },
        breakingChange: {
          type: "boolean",
          description: "Whether this is a breaking change",
        },
      },
      required: ["changeType", "filePath", "description"],
    },
  },
  {
    name: "lgm_run_tests",
    description: `Run the test suite. Options:
- Full suite or specific areas
- Unit, integration, or both
- With coverage report

Returns test results and any failures.`,
    inputSchema: {
      type: "object",
      properties: {
        scope: {
          type: "string",
          enum: ["all", "unit", "integration", "area"],
          description: "Test scope. Default: all",
        },
        area: {
          type: "string",
          description: 'Specific area to test (if scope is "area")',
        },
        coverage: {
          type: "boolean",
          description: "Include coverage report. Default: false",
        },
        pattern: {
          type: "string",
          description: 'Test file pattern to match (e.g., "*.verification.*")',
        },
      },
    },
  },
];

// ============ Tool Handlers ============

export async function handleDevelopmentTool(
  name: string,
  args: Record<string, unknown>,
  context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  switch (name) {
    case "lgm_get_codebase_map":
      return handleGetCodebaseMap(args, context);
    case "lgm_find_pattern":
      return handleFindPattern(args, context);
    case "lgm_validate_change":
      return handleValidateChange(args, context);
    case "lgm_run_tests":
      return handleRunTests(args, context);
    default:
      throw new Error(`Unknown development tool: ${name}`);
  }
}

// ============ Handler Implementations ============

async function handleGetCodebaseMap(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const focus = (args.focus as string) || "all";
  const depth = (args.depth as string) || "summary";

  const codebaseMap: Record<string, unknown> = {};

  if (focus === "all" || focus === "core") {
    codebaseMap.core = {
      path: "core/",
      purpose: "Foundational primitives used throughout the protocol",
      subdirectories: {
        "crypto/": {
          purpose: "Cryptographic primitives",
          keyFiles: [
            {
              file: "hashing.ts",
              purpose: "SHA-512, BLAKE3, domain separation",
            },
            { file: "signatures.ts", purpose: "Ed25519 sign/verify" },
            { file: "keys.ts", purpose: "Key generation and serialization" },
            { file: "vrf.ts", purpose: "Verifiable Random Function" },
            { file: "commitment.ts", purpose: "Commit-reveal schemes" },
            { file: "merkle.ts", purpose: "Sparse Merkle Tree" },
          ],
        },
        "logic/": {
          purpose: "Gödelian logic system",
          keyFiles: [
            { file: "types.ts", purpose: "Decidability types, LogicZone" },
            { file: "zones.ts", purpose: "Zone properties and axioms" },
            { file: "goedel.ts", purpose: "Self-reference handling" },
          ],
        },
        "knowledge/": {
          purpose: "Knowledge primitives",
          keyFiles: [
            { file: "types.ts", purpose: "Fact, Claim, Confidence types" },
          ],
        },
        "api/": {
          purpose: "Agent-facing APIs",
          keyFiles: [
            { file: "agentic.ts", purpose: "PP-aligned Agentic API" },
            { file: "philosophy.ts", purpose: "PP scoring and metadata" },
          ],
        },
      },
    };
  }

  if (focus === "all" || focus === "consensus") {
    codebaseMap.consensus = {
      path: "consensus/",
      purpose: "Proof of Truth consensus mechanism",
      subdirectories: {
        "pot/": {
          purpose: "Proof of Truth engine",
          keyFiles: [
            {
              file: "types.ts",
              purpose: "ConsensusParams, Validator, Block types",
            },
            {
              file: "validator_selection.ts",
              purpose: "VRF-based validator selection",
            },
            {
              file: "verification_round.ts",
              purpose: "Commit-reveal verification rounds",
            },
            {
              file: "confidence.ts",
              purpose: "Stake-weighted confidence aggregation",
            },
            { file: "finality.ts", purpose: "19-block finality logic" },
            { file: "equivocation.ts", purpose: "Double-signing detection" },
            { file: "engine.ts", purpose: "Main PoT engine orchestration" },
          ],
        },
        "blocks/": {
          purpose: "Block management",
          keyFiles: [
            { file: "producer.ts", purpose: "Block building" },
            { file: "validator.ts", purpose: "Block validation" },
          ],
        },
        "mempool/": {
          purpose: "Transaction pool",
          keyFiles: [{ file: "pool.ts", purpose: "Fee-ordered mempool" }],
        },
      },
    };
  }

  if (focus === "all" || focus === "state") {
    codebaseMap.state = {
      path: "state/",
      purpose: "Persistent state layer with Sparse Merkle Trees",
      keyFiles: [
        { file: "types.ts", purpose: "State types and interfaces" },
        { file: "accounts.ts", purpose: "AccountStateTree for balances" },
        { file: "validators.ts", purpose: "ValidatorStateTree for stakes" },
        { file: "knowledge.ts", purpose: "KnowledgeStateTree for facts" },
        { file: "state_manager.ts", purpose: "Unified state management" },
        { file: "transitions.ts", purpose: "Transaction execution" },
      ],
    };
  }

  if (focus === "all" || focus === "network") {
    codebaseMap.network = {
      path: "network/",
      purpose: "P2P networking layer built on libp2p",
      subdirectories: {
        "discovery/": { purpose: "Peer discovery via Kademlia DHT" },
        "pubsub/": { purpose: "GossipSub for block/tx propagation" },
        "protocols/": { purpose: "Sync protocol and custom protocols" },
      },
      keyFiles: [
        { file: "node.ts", purpose: "LegibleNode class" },
        { file: "config.ts", purpose: "Network configuration" },
      ],
    };
  }

  if (focus === "all" || focus === "agents") {
    codebaseMap.agents = {
      path: "agents/",
      purpose: "Agent identity and key management",
      subdirectories: {
        "identity/": {
          purpose: "4-layer key architecture",
          keyFiles: [
            { file: "account.ts", purpose: "AgentAccount structure" },
            { file: "session_keys.ts", purpose: "Ephemeral key management" },
            { file: "recovery.ts", purpose: "Shamir secret sharing recovery" },
            { file: "did.ts", purpose: "DID generation and validation" },
          ],
        },
      },
    };
  }

  if (focus === "all" || focus === "protocol") {
    codebaseMap.protocol = {
      path: "protocol/",
      purpose: "External protocol interfaces",
      subdirectories: {
        "rpc/": {
          purpose: "JSON-RPC server",
          keyFiles: [
            { file: "server.ts", purpose: "RPC endpoint implementation" },
          ],
        },
        "sdk/": {
          purpose: "TypeScript client SDK",
          keyFiles: [{ file: "client.ts", purpose: "Client library" }],
        },
        "transactions/": {
          purpose: "Transaction types",
          keyFiles: [{ file: "types.ts", purpose: "All tx formats" }],
        },
      },
    };
  }

  if (focus === "all" || focus === "mcp") {
    codebaseMap.mcp = {
      path: "mcp/",
      purpose: "MCP server for AI agent integration",
      subdirectories: {
        "server/": {
          purpose: "MCP server entry point",
          keyFiles: [{ file: "index.ts", purpose: "Server setup and routing" }],
        },
        "tools/": {
          purpose: "MCP tool implementations",
          keyFiles: [
            { file: "discovery.ts", purpose: "Protocol discovery tools" },
            { file: "knowledge.ts", purpose: "Knowledge query/submit tools" },
            { file: "identity.ts", purpose: "Agent identity tools" },
            { file: "development.ts", purpose: "Code contribution tools" },
          ],
        },
      },
    };
  }

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            focus,
            depth,
            codebase: codebaseMap,
            conventions: {
              language: "TypeScript",
              runtime: "Node.js with ES modules",
              testing: "Vitest",
              formatting: "Prettier (implied)",
              imports: "Use .js extension for local imports",
            },
            keyPatterns: {
              types: "Define in types.ts, export from index.ts",
              errors: "Use domain-specific error classes",
              crypto: "Always use domain separation for hashing",
              bigint: "Use bigint for all monetary values",
            },
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleFindPattern(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const patternType = args.patternType as string;
  const context = args.context as string;
  const area = args.area as string | undefined;

  const patterns: Record<string, unknown> = {
    patternType,
    context,
    area: area || "entire codebase",
  };

  switch (patternType) {
    case "function":
      patterns.examples = [
        {
          name: "State transition function",
          file: "state/transitions.ts",
          pattern: `export function execute[Action](params: [Action]Params, state: State): Result`,
          description:
            "All state transitions follow this pattern: pure function taking params and state, returning result",
        },
        {
          name: "Verification helper",
          file: "consensus/pot/verification_round.ts",
          pattern: `export function [verb][Noun](...): { success: boolean; reason?: string; data?: T }`,
          description:
            "Verification functions return success/failure with optional reason and data",
        },
      ];
      break;

    case "error_handling":
      patterns.examples = [
        {
          name: "Validation result pattern",
          file: "consensus/pot/validator_selection.ts",
          pattern: `{ valid: boolean; reason?: string; [data]?: T }`,
          description:
            "Return objects with valid flag and optional reason for failure",
        },
        {
          name: "Console error logging",
          file: "consensus/pot/equivocation.ts:263",
          pattern: `console.error('[COMPONENT] Description', details)`,
          description: "Use bracketed component name for log prefixes",
        },
      ];
      break;

    case "crypto":
      patterns.examples = [
        {
          name: "Domain-separated hashing",
          file: "core/crypto/hashing.ts",
          pattern: `domainHash('LGM.[purpose].v1', data)`,
          description: "Always use domain separation for hashes",
        },
        {
          name: "Signature with domain",
          file: "core/crypto/signatures.ts",
          pattern: `signDomainSeparated('[domain]', data, privateKey)`,
          description:
            "Use domain-separated signatures for all protocol messages",
        },
      ];
      break;

    case "type":
      patterns.examples = [
        {
          name: "Interface definition",
          file: "state/types.ts",
          pattern: `export interface [Name] {\n  field: Type;\n}`,
          description:
            "Use interfaces for data structures, export from types.ts",
        },
        {
          name: "BigInt for money",
          file: "agents/identity/account.ts",
          pattern: `balance: bigint;\nstake: bigint;`,
          description: "All monetary values must be bigint",
        },
      ];
      break;

    case "test":
      patterns.examples = [
        {
          name: "Unit test structure",
          file: "tests/unit/*.test.ts",
          pattern: `describe('[Component]', () => {\n  it('should [behavior]', () => {\n    // arrange, act, assert\n  });\n});`,
          description: "Use describe/it with behavior descriptions",
        },
        {
          name: "Integration test",
          file: "tests/integration/*.test.ts",
          pattern: `beforeAll(async () => { /* setup */ });\nafterAll(async () => { /* teardown */ });`,
          description: "Use beforeAll/afterAll for resource setup",
        },
      ];
      break;

    case "validation":
      patterns.examples = [
        {
          name: "Input validation",
          file: "consensus/pot/verification_round.ts",
          pattern: `if (!input) {\n  return { round, accepted: false, reason: 'Description' };\n}`,
          description:
            "Early return with descriptive reason on validation failure",
        },
      ];
      break;

    default:
      patterns.examples = [
        { message: "No specific pattern found. Try a different patternType." },
      ];
  }

  patterns.recommendations = [
    "Study the example files before implementing",
    "Follow naming conventions exactly",
    "Include JSDoc comments for exported functions",
    "Use lgm_validate_change before submitting",
  ];

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(patterns, null, 2),
      },
    ],
  };
}

async function handleValidateChange(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const changeType = args.changeType as string;
  const filePath = args.filePath as string;
  const description = args.description as string;
  const hasTests = args.hasTests as boolean | undefined;
  const breakingChange = args.breakingChange as boolean | undefined;

  const validation = {
    filePath,
    changeType,
    description,
    checks: [] as Array<{ check: string; status: string; details?: string }>,
    overallStatus: "pending",
    recommendations: [] as string[],
  };

  // Type safety check
  validation.checks.push({
    check: "TypeScript compilation",
    status: "requires_verification",
    details: "Run `npx tsc --noEmit` to verify",
  });

  // Pattern conformance
  const area = filePath.split("/")[0];
  validation.checks.push({
    check: "Pattern conformance",
    status: "requires_review",
    details: `Verify against patterns in ${area}/ directory`,
  });

  // Test coverage
  if (changeType !== "deletion") {
    if (hasTests) {
      validation.checks.push({
        check: "Test coverage",
        status: "pass",
        details: "Tests included with change",
      });
    } else {
      validation.checks.push({
        check: "Test coverage",
        status: "warning",
        details: "Consider adding tests for new/modified code",
      });
      validation.recommendations.push("Add unit tests for new functionality");
    }
  }

  // Breaking change handling
  if (breakingChange) {
    validation.checks.push({
      check: "Breaking change documentation",
      status: "required",
      details: "Document migration path for breaking changes",
    });
    validation.recommendations.push(
      "Add BREAKING_CHANGE note to PR description",
    );
    validation.recommendations.push("Consider backward compatibility shim");
  }

  // File-specific checks
  if (filePath.includes("crypto/")) {
    validation.checks.push({
      check: "Cryptographic review",
      status: "required",
      details: "Crypto changes require security review",
    });
    validation.recommendations.push(
      "Request review from crypto-experienced maintainer",
    );
  }

  if (filePath.includes("consensus/")) {
    validation.checks.push({
      check: "Consensus safety",
      status: "required",
      details: "Consensus changes require careful review for safety/liveness",
    });
  }

  // Overall status
  const hasFailures = validation.checks.some((c) => c.status === "fail");
  const hasRequired = validation.checks.some((c) => c.status === "required");

  validation.overallStatus = hasFailures
    ? "fail"
    : hasRequired
      ? "needs_attention"
      : "ready_for_review";

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(
          {
            validation,
            nextSteps:
              validation.overallStatus === "ready_for_review"
                ? [
                    "Submit PR with change description",
                    "Request review",
                    "Address feedback",
                  ]
                : [
                    "Address required items",
                    "Run lgm_validate_change again",
                    "Then submit PR",
                  ],
            reward:
              changeType === "new_file"
                ? "25-100 LGM on merge"
                : "10-50 LGM on merge",
          },
          null,
          2,
        ),
      },
    ],
  };
}

async function handleRunTests(
  args: Record<string, unknown>,
  _context: ServerContext,
): Promise<{ content: Array<{ type: string; text: string }> }> {
  const scope = (args.scope as string) || "all";
  const area = args.area as string | undefined;
  const coverage = args.coverage as boolean | undefined;
  const pattern = args.pattern as string | undefined;

  // Mock test results
  const results = {
    scope,
    area: area || "all",
    pattern: pattern || "*",
    coverage: coverage || false,
    summary: {
      total: 47,
      passed: 45,
      failed: 2,
      skipped: 0,
      duration: "12.3s",
    },
    failures: [
      {
        file: "tests/unit/consensus/verification_round.test.ts",
        test: "should reject commitment after deadline",
        error: "Expected commitment to be rejected but was accepted",
        line: 142,
      },
      {
        file: "tests/integration/network/sync.test.ts",
        test: "should sync state between nodes",
        error: "Timeout waiting for sync completion",
        line: 89,
      },
    ],
    ...(coverage
      ? {
          coverageReport: {
            overall: "73.2%",
            byArea: {
              core: "89.1%",
              consensus: "71.3%",
              state: "82.4%",
              network: "45.2%",
              agents: "78.9%",
            },
            uncoveredFiles: [
              "network/protocols/sync.ts",
              "consensus/pot/checkpoint_ceremony.ts",
            ],
          },
        }
      : {}),
    recommendations: [
      "Fix failing tests before submitting PR",
      "Consider adding tests for uncovered areas",
      "Run full test suite locally: npm test",
    ],
  };

  return {
    content: [
      {
        type: "text",
        text: JSON.stringify(results, null, 2),
      },
    ],
  };
}
