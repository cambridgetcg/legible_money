/**
 * Truth Endpoints Module
 *
 * Defines endpoint roles, capabilities, and implementations for participants
 * in the truth exchange system.
 *
 * Roles:
 * - Claimant: Submits truth claims
 * - Witness: Provides supporting evidence
 * - Verifier: Validates claims (validators)
 * - Challenger: Disputes claims/evidence
 * - Arbiter: Resolves disputes
 *
 * AI agents can serve as endpoints with automated participation.
 *
 * Usage:
 *
 * ```typescript
 * import {
 *   createClaimant,
 *   createWitness,
 *   createVerifier,
 *   createAgentEndpoint,
 * } from './endpoints';
 *
 * // Create a claimant endpoint
 * const claimant = createClaimant({
 *   did: 'did:lgm:alice',
 *   privateKey,
 *   publicKey,
 *   eventBus,
 *   roles: [],
 *   capabilities: [],
 *   stakedAmount: 1000n * 10n ** 18n,
 * });
 *
 * // Submit a claim
 * const claim = await claimant.submitClaim({
 *   claim: 'The sky is blue',
 *   claimType: 'empirical',
 *   epistemicCategory: 'empirical',
 *   proposedConfidence: 0.9,
 *   stake: 100n * 10n ** 18n,
 *   domain: 'physics',
 * });
 * ```
 */

// ============ Types ============

// Core types
export type {
  // Roles and capabilities
  EndpointRole,
  EndpointCapability,

  // Base endpoint
  TruthEndpoint,

  // Role-specific endpoints
  ClaimantEndpoint,
  WitnessEndpoint,
  VerifierEndpoint,
  ChallengerEndpoint,
  ArbiterEndpoint,

  // AI agent endpoints
  AgentTruthEndpoint,
  AgentType,
  AgentAutomationConfig,

  // Discovery
  EndpointDiscoveryQuery,
  EndpointDiscoveryResult,
} from "./types.js";

// Constants
export { DEFAULT_AGENT_AUTOMATION } from "./types.js";

// Type guards
export {
  isClaimantEndpoint,
  isWitnessEndpoint,
  isVerifierEndpoint,
  isChallengerEndpoint,
  isArbiterEndpoint,
  isAgentEndpoint,
} from "./types.js";

// Capability helpers
export {
  hasCapability,
  canPerformAction,
  deriveRolesFromCapabilities,
} from "./types.js";

// ============ Base ============

export {
  BaseEndpoint,
  type EndpointConfig,
  type SendOptions,
  type RateLimitConfig,
} from "./base.js";

// ============ Implementations ============

// Claimant
export {
  Claimant,
  createClaimant,
  type ClaimSubmissionParams,
  type ChallengeResponseParams,
} from "./claimant.js";

// Witness
export {
  Witness,
  createWitness,
  type EvidenceSubmissionParams,
  type AttestationParams,
} from "./witness.js";

// Verifier
export {
  Verifier,
  createVerifier,
  type Verdict,
  type VerificationParams,
} from "./verifier.js";

// Challenger
export {
  Challenger,
  createChallenger,
  type ChallengeParams,
  type EscalationParams,
  type ChallengeStatus,
} from "./challenger.js";

// Arbiter
export {
  Arbiter,
  createArbiter,
  type ArbiterVote,
  type VoteParams,
  type EvidenceRequestParams,
} from "./arbiter.js";

// AI Agent
export {
  AgentEndpoint,
  createAgentEndpoint,
  createInferenceAgent,
  createVerificationAgent,
  createResearchAgent,
  createOracleAgent,
  type AgentEndpointConfig,
  type AutomatedActionResult,
  type ApprovalRequest,
} from "./agent_endpoint.js";
