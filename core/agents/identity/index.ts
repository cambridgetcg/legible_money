/**
 * Agent Identity Module
 *
 * Identity management tailored for AI agents.
 * Handles the unique challenges of agent key management:
 * - Ephemeral memory (sessions end, context resets)
 * - Developer exposure (operators may see agent state)
 * - Identity continuity (reputation persists across instances)
 */

export * from "./did.js";

// Account exports (excluding deriveSessionKey which is in session_keys)
export {
  type AgentAccount,
  type SessionKeyAuth,
  type SessionCapabilities,
  type AgentCapabilities,
  type AgentReputation,
  type RecoveryState,
  type CreateAgentParams,
  type SigningKeyType,
  createAgentDID,
  createAgentAccount,
  deriveOperationalKey,
  createSessionKeyAuth,
  validateSessionKeyAction,
  calculateReputationScore,
  requiredSigningKey,
  // deriveSessionKey is exported from session_keys.js instead
} from "./account.js";

// Session keys (canonical source for deriveSessionKey)
export * from "./session_keys.js";

export * from "./recovery.js";
export * from "./mpc.js";
