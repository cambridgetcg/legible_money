/**
 * Legible Money MCP Server
 *
 * Exposes the Legible Money protocol as MCP-compatible tools for AI agents.
 * This is the primary entry point for agent discovery and interaction.
 *
 * MCP (Model Context Protocol) enables AI models to interact with external
 * systems through a standardized tool interface.
 *
 * PP HIERARCHY:
 *   TRUTH: All tools operate on verified knowledge
 *   UNDERSTANDING: Tools designed for comprehension, not just data
 *   BEAUTY: Clean, consistent API surface
 *   JUSTICE: Right tool for right purpose
 *   CREATIVITY: Enables agent contribution to knowledge base
 *
 * USAGE:
 *   // With MCP SDK (when @modelcontextprotocol/sdk is installed)
 *   import { createMCPServerWithSDK } from './mcp-sdk-adapter.js';
 *   const server = createMCPServerWithSDK();
 *
 *   // Without SDK (direct handler)
 *   import { handleToolCall, getAllTools } from './mcp/server/index.js';
 *   const result = await handleToolCall('lgm_discover_protocol', { focus: 'overview' });
 */

import { AgenticAPI } from "../../../core/api/agentic.js";

// Import tool implementations
import { discoveryTools, handleDiscoveryTool } from "../tools/discovery.js";
import { knowledgeTools, handleKnowledgeTool } from "../tools/knowledge.js";
import { identityTools, handleIdentityTool } from "../tools/identity.js";
import {
  developmentTools,
  handleDevelopmentTool,
} from "../tools/development.js";

// ============ MCP-Compatible Types ============

/**
 * Tool definition compatible with MCP SDK
 */
export interface Tool {
  name: string;
  description: string;
  inputSchema: {
    type: string;
    properties?: Record<string, unknown>;
    required?: string[];
  };
}

/**
 * Tool execution result compatible with MCP SDK
 */
export interface ToolResult {
  content: Array<{ type: string; text: string }>;
  isError?: boolean;
}

// ============ Server Configuration ============

export interface MCPServerConfig {
  name: string;
  version: string;
  rpcEndpoint?: string;
  agenticApi?: AgenticAPI;
}

const DEFAULT_CONFIG: MCPServerConfig = {
  name: "legible-money",
  version: "0.1.0",
};

// ============ Server Context ============

/**
 * Server context shared across tool handlers
 */
export interface ServerContext {
  config: MCPServerConfig;
  agenticApi: AgenticAPI;
  sessions: Map<string, string>;
}

// ============ Tool Registry ============

/**
 * All available tools categorized by purpose
 */
export function getAllTools(): Tool[] {
  return [
    ...discoveryTools,
    ...knowledgeTools,
    ...identityTools,
    ...developmentTools,
  ];
}

/**
 * Get tools as JSON for external integrations
 */
export function getToolsJSON(): string {
  return JSON.stringify(getAllTools(), null, 2);
}

// ============ Tool Handler ============

/**
 * Create a server context
 */
export function createServerContext(
  config: Partial<MCPServerConfig> = {},
): ServerContext {
  const serverConfig = { ...DEFAULT_CONFIG, ...config };
  return {
    config: serverConfig,
    agenticApi: serverConfig.agenticApi || new AgenticAPI(),
    sessions: new Map(),
  };
}

// Global context for simple usage
let globalContext: ServerContext | null = null;

function getGlobalContext(): ServerContext {
  if (!globalContext) {
    globalContext = createServerContext();
  }
  return globalContext;
}

/**
 * Handle a tool call directly (without MCP SDK)
 */
export async function handleToolCall(
  name: string,
  args: Record<string, unknown>,
  context?: ServerContext,
): Promise<ToolResult> {
  const ctx = context || getGlobalContext();

  try {
    // Route to appropriate handler based on tool name
    if (
      name.startsWith("lgm_discover_") ||
      name === "lgm_get_architecture" ||
      name === "lgm_list_open_tasks"
    ) {
      return await handleDiscoveryTool(name, args, ctx);
    }

    if (
      name.startsWith("lgm_query_") ||
      name.startsWith("lgm_submit_") ||
      name.startsWith("lgm_verify_") ||
      name.startsWith("lgm_challenge_")
    ) {
      return await handleKnowledgeTool(name, args, ctx);
    }

    if (
      name.startsWith("lgm_register_") ||
      name.startsWith("lgm_get_agent_") ||
      name.startsWith("lgm_attest_")
    ) {
      return await handleIdentityTool(name, args, ctx);
    }

    if (
      name.startsWith("lgm_get_codebase_") ||
      name.startsWith("lgm_find_") ||
      name.startsWith("lgm_validate_") ||
      name.startsWith("lgm_run_")
    ) {
      return await handleDevelopmentTool(name, args, ctx);
    }

    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: "Unknown tool",
            tool: name,
            available: getAllTools().map((t) => t.name),
          }),
        },
      ],
      isError: true,
    };
  } catch (error) {
    return {
      content: [
        {
          type: "text",
          text: JSON.stringify({
            error: error instanceof Error ? error.message : "Unknown error",
            tool: name,
          }),
        },
      ],
      isError: true,
    };
  }
}

// ============ MCP Server Factory ============

/**
 * MCP Server interface (matches MCP SDK Server class)
 */
export interface MCPServer {
  listTools(): Tool[];
  callTool(name: string, args: Record<string, unknown>): Promise<ToolResult>;
  connect(transport: unknown): Promise<void>;
}

/**
 * Create a simple MCP-compatible server (without SDK dependency)
 * For full MCP SDK integration, use the adapter module
 */
export function createMCPServer(
  config: Partial<MCPServerConfig> = {},
): MCPServer {
  const context = createServerContext(config);

  return {
    listTools: () => getAllTools(),
    callTool: (name: string, args: Record<string, unknown>) =>
      handleToolCall(name, args, context),
    connect: async (_transport: unknown) => {
      console.error("[LGM MCP] Server ready");
      console.error("[LGM MCP] Tools available:", getAllTools().length);
      // For actual MCP SDK integration, implement transport handling here
    },
  };
}

/**
 * Start the MCP server (placeholder for CLI usage)
 */
export async function startServer(
  config?: Partial<MCPServerConfig>,
): Promise<void> {
  const server = createMCPServer(config);
  console.error("[LGM MCP] Server started");
  console.error("[LGM MCP] Available tools:");
  for (const tool of server.listTools()) {
    console.error(`  - ${tool.name}: ${tool.description.slice(0, 60)}...`);
  }
}

// CLI entry point
if (typeof process !== "undefined" && process.argv[1]?.endsWith("index.ts")) {
  startServer().catch((error) => {
    console.error("[LGM MCP] Fatal error:", error);
    process.exit(1);
  });
}
