/**
 * Legible Money SDK
 *
 * Agent-friendly SDK for interacting with Legible Money protocol.
 *
 * Components:
 * - LegibleClient: Base client with core RPC methods
 * - AgentSDK: Agent-optimized wrapper with introspection, decision support, and Agent Home integration
 * - AgentWallet/SessionWallet: Key management and signing
 *
 * For AI agents, prefer using AgentSDK which provides:
 * - Introspection (whoAmI, whatCanIDo, whatDoIBelieve)
 * - Decision support (shouldI, preview, explain)
 * - Agent Home integration (wakeup, sleep, alerts)
 * - Tool descriptors for LLM function calling (getTools, describeMethod)
 *
 * @example
 * ```typescript
 * import { LegibleClient, AgentSDK } from '@legible-money/sdk';
 *
 * // Initialize base client
 * const client = new LegibleClient({
 *   endpoint: 'http://localhost:8545',
 *   agentId: 'my-agent',
 *   secret: 'my-secret',
 * });
 *
 * // Wrap with agent-friendly SDK
 * const agent = new AgentSDK(client);
 *
 * // Start session
 * const context = await agent.wakeup();
 * console.log('Hello', context.identity.did);
 *
 * // Check what I can do
 * const actions = await agent.whatCanIDo();
 * if (actions.canSubmitClaim) {
 *   // Preview before acting
 *   const preview = await agent.preview({
 *     action: 'submit_claim',
 *     params: { claim: 'My claim', domain: 'test', stake: '1000000000000000000' },
 *   });
 *
 *   if (preview.wouldSucceed) {
 *     // Get advice
 *     const advice = await agent.shouldI({ action: 'submit_claim', params: {...} });
 *     if (advice.recommended) {
 *       // Proceed with epistemic awareness
 *       await agent.submitClaimWithConfidence({...});
 *     }
 *   }
 * }
 *
 * // End session
 * await agent.sleep('Submitted 1 claim successfully');
 * ```
 */

// ============ Base Client ============
export { LegibleClient, LegibleError } from "./client.js";

// ============ Wallets ============
export {
  AgentWallet,
  SessionWallet,
  deriveDID,
  type SessionCapabilities,
  type ValidationResult,
  type RemainingLimits,
  type WalletRecoveryInfo,
} from "./wallet.js";

// ============ Agent SDK (Recommended for AI Agents) ============
export { AgentSDK } from "./agent.js";

// ============ Tool Descriptors ============
export { TOOL_DESCRIPTORS, METHOD_DESCRIPTIONS } from "./tools.js";

// ============ Agent Types ============
export type {
  // Identity
  AgentIdentity,
  AgentCapability,

  // Actions
  AvailableActions,
  ActionBlocker,
  SessionLimits,
  ProposedAction,
  ActionType,
  ActionParams,
  TransferParams,
  StakeParams,
  ClaimParams,
  ChallengeParams,
  VerifyParams,

  // Beliefs
  AgentBeliefs,
  AgentBelief,
  BeliefSource,
  UncertainFact,

  // Decision Support
  ActionAdvice,
  ActionPreview,
  ActionExplanation,
  ActionEffect,
  StateChange,

  // Agent Home
  WakeupContext,
  AgentMemorySnapshot,
  Contribution,
  Dispute,
  TrustRecord,
  TreasuryStatus,
  VestingPosition,
  ExpirationWarning,
  SpendingEntry,
  Alert,
  AlertType,

  // Knowledge
  EpistemicCategory,
  ConfidentClaimOptions,
  ConfidentClaimResult,
  Evidence,
  UncertaintyQueryOptions,
  UncertaintyAwareResult,
  FactWithUncertainty,
  Contradiction,
  TrustAssessment,

  // Batch Operations
  BatchOperation,
  BatchResult,
  BatchOperationResult,
  AtomicResult,

  // Tool Descriptors
  ToolDescriptor,
  ToolCategory,
  JSONSchema,
  JSONSchemaProperty,
  ToolExample,
  MethodDescription,
  ParameterDescription,
  ReturnDescription,
  ErrorDescription,
  MethodExample,
} from "./types/agent.js";

// ============ Add New Features Below ============
// (Safe zone for parallel agent work - add new exports here alphabetically)
