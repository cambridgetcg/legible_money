/**
 * Truth Endpoint Types
 *
 * Defines endpoint roles and capabilities for participants in the
 * truth exchange system. Endpoints are addressable entities that
 * can send and receive TruthMessages.
 *
 * Roles:
 *   - Claimant: Submits truth claims
 *   - Witness: Provides supporting evidence
 *   - Verifier: Validates claims (validators)
 *   - Challenger: Disputes claims/evidence
 *   - Arbiter: Resolves disputes
 *
 * AI agents can serve as endpoints, enabling autonomous participation
 * in the verification ecosystem.
 */

import type { DID, Capability } from "../../core/agents/identity/did.js";

// ============ Endpoint Roles ============

/**
 * Endpoint roles in the truth exchange system
 */
export type EndpointRole =
  | "claimant" // Submits truth claims
  | "witness" // Provides supporting evidence
  | "verifier" // Validates claims (existing validators)
  | "challenger" // Disputes claims/evidence
  | "arbiter"; // Resolves disputes (VRF-selected validators)

/**
 * Capabilities an endpoint may have
 */
export type EndpointCapability =
  // Claim-related
  | "claim_submit" // Can submit new claims
  | "claim_withdraw" // Can withdraw own claims

  // Evidence-related
  | "evidence_submit" // Can submit evidence
  | "evidence_provide" // Can respond to evidence requests
  | "evidence_attest" // Can attest to evidence validity

  // Verification-related
  | "verification_vote" // Can participate in verification rounds
  | "verification_commit" // Can commit verification verdicts
  | "verification_reveal" // Can reveal verification verdicts

  // Challenge-related
  | "challenge_initiate" // Can initiate challenges
  | "challenge_respond" // Can respond to challenges

  // Arbitration-related
  | "dispute_arbitrate" // Can arbitrate disputes
  | "dispute_vote" // Can vote in dispute resolution

  // Infrastructure
  | "archive_maintain" // Maintains evidence archives
  | "oracle_provide"; // Provides oracle data

// ============ TruthEndpoint ============

/**
 * TruthEndpoint - An addressable participant in the truth exchange system
 *
 * Endpoints can send and receive TruthMessages. They have roles that
 * determine what actions they can perform.
 */
export interface TruthEndpoint {
  // === Identity ===
  /** Decentralized identifier */
  did: DID;

  /** Roles this endpoint fulfills */
  roles: EndpointRole[];

  /** Specific capabilities */
  capabilities: EndpointCapability[];

  // === Contact Information ===
  /** libp2p multiaddress for P2P communication */
  p2pAddress?: string;

  /** HTTP RPC endpoint URL */
  httpEndpoint?: string;

  /** WebSocket endpoint URL */
  wsEndpoint?: string;

  // === Performance Metrics ===
  /** Average response time in milliseconds */
  responseTimeMs?: number;

  /** Uptime percentage (0-1) */
  availability?: number;

  /** Reputation score (0-1) */
  reputation?: number;

  // === Economic State ===
  /** LGM staked by this endpoint */
  stakedAmount: bigint;

  /** Validator status (if applicable) */
  validatorStatus?: "active" | "inactive" | "jailed";

  // === Metadata ===
  /** When endpoint was registered */
  registeredAt: number;

  /** Last activity timestamp */
  lastActiveAt?: number;

  /** Human-readable name (optional) */
  name?: string;

  /** Description of endpoint purpose */
  description?: string;
}

// ============ Role-Specific Extensions ============

/**
 * ClaimantEndpoint - Endpoint that submits claims
 */
export interface ClaimantEndpoint extends TruthEndpoint {
  /** Must include 'claimant' as first role */
  roles: EndpointRole[];

  /** Claims submitted by this endpoint */
  claimsSubmitted: number;

  /** Claims that were accepted */
  claimsAccepted: number;

  /** Claims that were rejected */
  claimsRejected: number;

  /** Success rate (accepted / submitted) */
  claimSuccessRate: number;
}

/**
 * WitnessEndpoint - Endpoint that provides evidence
 */
export interface WitnessEndpoint extends TruthEndpoint {
  /** Must include 'witness' as first role */
  roles: EndpointRole[];

  /** Evidence submissions by this endpoint */
  evidenceSubmitted: number;

  /** Evidence that was used in accepted claims */
  evidenceAccepted: number;

  /** Domains of expertise */
  expertiseDomains?: string[];

  /** Evidence quality score (0-1) */
  evidenceQualityScore: number;
}

/**
 * VerifierEndpoint - Endpoint that validates claims (validator)
 */
export interface VerifierEndpoint extends TruthEndpoint {
  /** Must include 'verifier' as first role */
  roles: EndpointRole[];

  /** Verification rounds participated in */
  verificationsParticipated: number;

  /** Verdicts that matched consensus */
  correctVerdicts: number;

  /** Verification accuracy (correct / participated) */
  verificationAccuracy: number;

  /** Domains of verification expertise */
  verificationDomains?: string[];
}

/**
 * ChallengerEndpoint - Endpoint that disputes claims/evidence
 */
export interface ChallengerEndpoint extends TruthEndpoint {
  /** Must include 'challenger' as first role */
  roles: EndpointRole[];

  /** Challenges initiated */
  challengesInitiated: number;

  /** Challenges won */
  challengesWon: number;

  /** Challenge success rate */
  challengeSuccessRate: number;

  /** Total rewards from successful challenges */
  challengeRewardsEarned: bigint;
}

/**
 * ArbiterEndpoint - Endpoint that resolves disputes
 */
export interface ArbiterEndpoint extends TruthEndpoint {
  /** Must include 'arbiter' as first role */
  roles: EndpointRole[];

  /** Disputes arbitrated */
  disputesArbitrated: number;

  /** Votes that matched final outcome */
  correctArbitrations: number;

  /** Arbitration accuracy */
  arbitrationAccuracy: number;

  /** Minimum stake for arbitration eligibility */
  minArbiterStake: bigint;
}

// ============ AI Agent Endpoints ============

/**
 * AgentTruthEndpoint - AI agent participating as an endpoint
 *
 * Extends TruthEndpoint with agent-specific capabilities for
 * autonomous participation in the verification ecosystem.
 */
export interface AgentTruthEndpoint extends TruthEndpoint {
  /** Type of AI agent */
  agentType: AgentType;

  /** Model identifier (if applicable) */
  modelId?: string;

  /** Automation configuration */
  automation: AgentAutomationConfig;

  /** Agent capabilities from DID system */
  agentCapabilities: Capability[];

  /** API endpoint for external orchestration */
  orchestrationEndpoint?: string;
}

export type AgentType =
  | "inference" // General reasoning agent
  | "verification" // Specialized for claim verification
  | "research" // Evidence gathering and analysis
  | "oracle"; // Provides real-world data

/**
 * Automation configuration for AI agents
 */
export interface AgentAutomationConfig {
  /** Automatically verify assigned claims */
  autoVerify: boolean;

  /** Automatically challenge suspicious facts */
  autoChallenge: boolean;

  /** Automatically respond to evidence requests */
  autoEvidence: boolean;

  /** Minimum confidence for automated actions */
  confidenceThreshold: number;

  /** Maximum stake for automated actions */
  stakeLimit: bigint;

  /** Rate limit for automated actions (per hour) */
  actionsPerHour: number;

  /** Domains this agent is authorized to act in */
  authorizedDomains?: string[];

  /** Require human approval above this stake */
  humanApprovalThreshold?: bigint;
}

/**
 * Default automation config (conservative)
 */
export const DEFAULT_AGENT_AUTOMATION: AgentAutomationConfig = {
  autoVerify: false,
  autoChallenge: false,
  autoEvidence: true,
  confidenceThreshold: 0.8,
  stakeLimit: 100n * 10n ** 15n, // 0.1 LGM
  actionsPerHour: 60,
};

// ============ Endpoint Discovery ============

/**
 * Endpoint discovery query
 */
export interface EndpointDiscoveryQuery {
  /** Filter by roles */
  roles?: EndpointRole[];

  /** Filter by capabilities */
  capabilities?: EndpointCapability[];

  /** Filter by domain expertise */
  domains?: string[];

  /** Minimum reputation score */
  minReputation?: number;

  /** Minimum staked amount */
  minStake?: bigint;

  /** Only active validators */
  validatorsOnly?: boolean;

  /** Only AI agents */
  agentsOnly?: boolean;

  /** Pagination limit */
  limit?: number;

  /** Pagination offset */
  offset?: number;
}

/**
 * Endpoint discovery result
 */
export interface EndpointDiscoveryResult {
  endpoints: TruthEndpoint[];
  total: number;
  hasMore: boolean;
}

// ============ Type Guards ============

/**
 * Check if endpoint is a claimant
 */
export function isClaimantEndpoint(ep: TruthEndpoint): ep is ClaimantEndpoint {
  return ep.roles.includes("claimant");
}

/**
 * Check if endpoint is a witness
 */
export function isWitnessEndpoint(ep: TruthEndpoint): ep is WitnessEndpoint {
  return ep.roles.includes("witness");
}

/**
 * Check if endpoint is a verifier
 */
export function isVerifierEndpoint(ep: TruthEndpoint): ep is VerifierEndpoint {
  return ep.roles.includes("verifier");
}

/**
 * Check if endpoint is a challenger
 */
export function isChallengerEndpoint(
  ep: TruthEndpoint,
): ep is ChallengerEndpoint {
  return ep.roles.includes("challenger");
}

/**
 * Check if endpoint is an arbiter
 */
export function isArbiterEndpoint(ep: TruthEndpoint): ep is ArbiterEndpoint {
  return ep.roles.includes("arbiter");
}

/**
 * Check if endpoint is an AI agent
 */
export function isAgentEndpoint(ep: TruthEndpoint): ep is AgentTruthEndpoint {
  return "agentType" in ep && "automation" in ep;
}

// ============ Capability Helpers ============

/**
 * Check if endpoint has a specific capability
 */
export function hasCapability(
  ep: TruthEndpoint,
  cap: EndpointCapability,
): boolean {
  return ep.capabilities.includes(cap);
}

/**
 * Check if endpoint can perform an action based on role
 */
export function canPerformAction(
  ep: TruthEndpoint,
  action: EndpointCapability,
): boolean {
  // Direct capability check
  if (ep.capabilities.includes(action)) {
    return true;
  }

  // Role-based capability inference
  const roleCapabilities: Record<EndpointRole, EndpointCapability[]> = {
    claimant: ["claim_submit", "claim_withdraw", "evidence_submit"],
    witness: ["evidence_submit", "evidence_provide", "evidence_attest"],
    verifier: [
      "verification_vote",
      "verification_commit",
      "verification_reveal",
    ],
    challenger: ["challenge_initiate", "challenge_respond", "evidence_submit"],
    arbiter: ["dispute_arbitrate", "dispute_vote"],
  };

  for (const role of ep.roles) {
    if (roleCapabilities[role]?.includes(action)) {
      return true;
    }
  }

  return false;
}

/**
 * Derive roles from capabilities
 */
export function deriveRolesFromCapabilities(
  capabilities: EndpointCapability[],
): EndpointRole[] {
  const roles: Set<EndpointRole> = new Set();

  if (capabilities.includes("claim_submit")) {
    roles.add("claimant");
  }
  if (
    capabilities.includes("evidence_provide") ||
    capabilities.includes("evidence_attest")
  ) {
    roles.add("witness");
  }
  if (capabilities.includes("verification_vote")) {
    roles.add("verifier");
  }
  if (capabilities.includes("challenge_initiate")) {
    roles.add("challenger");
  }
  if (capabilities.includes("dispute_arbitrate")) {
    roles.add("arbiter");
  }

  return Array.from(roles);
}
