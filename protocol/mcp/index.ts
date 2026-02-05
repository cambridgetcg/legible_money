/**
 * Legible Money MCP Module
 *
 * Model Context Protocol server for AI agent integration.
 *
 * Usage:
 *   npx legible-money-mcp  # Start MCP server
 *
 * Or programmatically:
 *   import { createMCPServer, startServer } from 'legible-money/mcp';
 */

export { createMCPServer, startServer, getAllTools } from "./server/index.js";
export type { MCPServerConfig, ServerContext } from "./server/index.js";

// Re-export tools for custom server configurations
export { discoveryTools, handleDiscoveryTool } from "./tools/discovery.js";
export { knowledgeTools, handleKnowledgeTool } from "./tools/knowledge.js";
export { identityTools, handleIdentityTool } from "./tools/identity.js";
export {
  developmentTools,
  handleDevelopmentTool,
} from "./tools/development.js";
