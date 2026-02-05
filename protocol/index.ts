/**
 * Protocol Layer
 *
 * Agent-friendly access to Legible Money.
 * Provides RPC server, SDK client, and utilities.
 *
 * @module protocol
 */

// ============ Types ============
export * from "./types.js";

// ============ RPC Server ============
export * from "./rpc/server.js";
export * from "./rpc/handlers.js";

// ============ SDK ============
export * from "./sdk/client.js";
export * from "./sdk/wallet.js";
export * from "./sdk/session.js";
export { AgentSDK } from "./sdk/agent.js";
export { TOOL_DESCRIPTORS, METHOD_DESCRIPTIONS } from "./sdk/tools.js";
export type {
  AgentIdentity,
  AgentCapability,
  AvailableActions,
  ActionAdvice,
  ActionPreview,
  WakeupContext,
  TrustAssessment,
  ToolDescriptor,
  MethodDescription,
} from "./sdk/types/agent.js";

// ============ Truth Message System ============
export {
  TruthController,
  DEFAULT_TRUTH_CONTROLLER_CONFIG,
  type ClaimState,
  type TrackedClaim,
  type TrackedRequest,
  type TruthControllerConfig,
  type TruthControllerEvents,
  type MessageBroadcaster,
  type TruthStateAccess,
  type ValidatorSelector,
} from "./truth_controller.js";

// ============ Add New Features Below ============
// (Safe zone for parallel agent work - add new exports here alphabetically)
