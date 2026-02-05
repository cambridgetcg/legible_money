/**
 * Agent SDK Types
 *
 * Type definitions for the agent-friendly SDK layer.
 * Optimized for LLM tool-use patterns (particularly Claude models).
 */

import type { FactState } from "../../../state/types.js";

// ============ Identity Types ============

/**
 * Agent's full identity snapshot
 * Returned by whoAmI() for self-introspection
 */
export interface AgentIdentity {
  /** Decentralized identifier (did:lgm:xxx) */
  did: string;

  /** Operational public key */
  publicKey: string;

  /** Current reputation score (0-1) */
  reputation: number;

  /** Accuracy metrics */
  accuracy: {
    /** Verification accuracy rate */
    verification: number;
    /** Claim acceptance rate */
    claims: number;
  };

  /** Token balances */
  balance: {
    /** Available for spending */
    available: bigint;
    /** Currently staked */
    staked: bigint;
    /** Locked in vesting */
    vesting: bigint;
  };

  /** Actions this agent can perform */
  capabilities: AgentCapability[];

  /** Number of active sessions */
  activeSessions: number;

  /** Validator status if applicable */
  validatorStatus?: {
    isActive: boolean;
    rank?: number;
    recentBlocks: number;
  };
}

/**
 * Capability labels for agent actions
 */
export type AgentCapability =
  | "transfer"
  | "stake"
  | "unstake"
  | "submit_claim"
  | "verify"
  | "challenge"
  | "create_session"
  | "manage_home";

// ============ Action Types ============

/**
 * What actions are available to the agent
 * Returned by whatCanIDo()
 */
export interface AvailableActions {
  /** Can submit knowledge claims */
  canSubmitClaim: boolean;

  /** Can participate in verification */
  canVerify: boolean;

  /** Can challenge existing facts */
  canChallenge: boolean;

  /** Can transfer tokens */
  canTransfer: boolean;

  /** Can stake tokens */
  canStake: boolean;

  /** Operational constraints */
  constraints: {
    /** Maximum stake per claim */
    maxClaimStake: bigint;
    /** Maximum transfer amount */
    maxTransfer: bigint;
    /** Session key limits */
    sessionLimits: SessionLimits | null;
  };

  /** Proactive suggestions */
  suggestions: string[];

  /** Reasons why certain actions are unavailable */
  blockers: ActionBlocker[];
}

/**
 * Session key operational limits
 */
export interface SessionLimits {
  remainingUsage: number;
  remainingDailySpend: bigint;
  blocksUntilExpiry: number;
}

/**
 * Explains why an action is blocked
 */
export interface ActionBlocker {
  action: AgentCapability;
  reason: string;
  resolution?: string;
}

// ============ Belief Types ============

/**
 * Agent's current belief state
 * Returned by whatDoIBelieve()
 */
export interface AgentBeliefs {
  /** Facts the agent has high confidence in */
  beliefs: AgentBelief[];

  /** Facts with decayed or uncertain confidence */
  uncertainties: UncertainFact[];

  /** Summary statistics */
  summary: {
    totalBeliefs: number;
    averageConfidence: number;
    domainsRepresented: string[];
  };
}

/**
 * A single belief held by the agent
 */
export interface AgentBelief {
  factId: string;
  statement: string;
  confidence: number;
  source: BeliefSource;
  domain: string;
  lastUpdated: number;
  decayRate?: number;
}

/**
 * Source of a belief
 */
export type BeliefSource =
  | "self_submitted" // Agent submitted this claim
  | "verified" // Agent participated in verification
  | "protocol" // Accepted by protocol consensus
  | "trusted_agent" // From an agent with high trust
  | "inference"; // Derived from other facts

/**
 * A fact with uncertain confidence
 */
export interface UncertainFact {
  factId: string;
  statement: string;
  currentConfidence: number;
  effectiveConfidence: number;
  reason: "decay" | "challenged" | "low_initial" | "contradicted";
  recommendation: "verify" | "monitor" | "discard";
}

// ============ Decision Support Types ============

/**
 * Proposed action for preview/shouldI
 */
export interface ProposedAction {
  action: ActionType;
  params: ActionParams;
}

/**
 * Available action types
 */
export type ActionType =
  | "transfer"
  | "stake"
  | "unstake"
  | "submit_claim"
  | "challenge"
  | "verify";

/**
 * Parameters for different action types
 */
export type ActionParams =
  | TransferParams
  | StakeParams
  | ClaimParams
  | ChallengeParams
  | VerifyParams;

export interface TransferParams {
  to: string;
  amount: bigint | string;
  memo?: string;
}

export interface StakeParams {
  amount: bigint | string;
}

export interface ClaimParams {
  claim: string;
  domain: string;
  stake: bigint | string;
  references?: string[];
  epistemicCategory?: EpistemicCategory;
  reasoning?: string;
}

export interface ChallengeParams {
  factId: string;
  evidence: string;
  stake: bigint | string;
}

export interface VerifyParams {
  roundId: string;
  verdict: "confirmed" | "rejected" | "uncertain";
  confidence: number;
  reasoning?: string;
}

/**
 * Epistemic category for claim classification
 */
export type EpistemicCategory =
  | "analytic" // Logically necessary (math, definitions)
  | "formal" // Formally verifiable (proofs, code)
  | "protocol" // Protocol state (on-chain facts)
  | "empirical" // Observable facts
  | "historical" // Past events
  | "predictive" // Future predictions
  | "social"; // Social conventions/consensus

/**
 * Advice about whether to perform an action
 * Returned by shouldI()
 */
export interface ActionAdvice {
  /** Whether the action is recommended */
  recommended: boolean;

  /** Confidence in this recommendation (0-1) */
  confidence: number;

  /** Human/agent-readable reasoning */
  reasoning: string;

  /** Identified risks */
  risks: string[];

  /** Alternative actions to consider */
  alternatives: ProposedAction[];

  /** Prerequisites that should be completed first */
  prerequisites: string[];
}

/**
 * Preview of what would happen
 * Returned by preview()
 */
export interface ActionPreview {
  /** Would the action succeed? */
  wouldSucceed: boolean;

  /** Estimated gas/fee cost */
  estimatedCost: bigint;

  /** Estimated reward (if applicable) */
  estimatedReward?: bigint;

  /** State changes that would occur */
  stateChanges: StateChange[];

  /** Warnings about potential issues */
  warnings: string[];

  /** Reasons why it would fail */
  blockers: string[];
}

/**
 * A state change from an action
 */
export interface StateChange {
  entity: string;
  field: string;
  before: unknown;
  after: unknown;
}

/**
 * Explanation of what happened
 * Returned by explain()
 */
export interface ActionExplanation {
  /** What action was taken */
  action: string;

  /** Outcome status */
  outcome: "success" | "failure" | "pending";

  /** Effects that occurred */
  effects: ActionEffect[];

  /** Why this outcome occurred */
  reasoning: string;

  /** Suggested next steps */
  nextSteps: string[];

  /** Transaction hash if applicable */
  txHash?: string;
}

/**
 * An effect from an action
 */
export interface ActionEffect {
  type:
    | "balance_change"
    | "state_update"
    | "event_emitted"
    | "fact_created"
    | "round_started";
  description: string;
  data?: unknown;
}

// ============ Agent Home Types ============

/**
 * Context loaded when agent wakes up
 * Returned by wakeup()
 */
export interface WakeupContext {
  /** Agent identity */
  identity: AgentIdentity;

  /** Persistent memory from Agent Home */
  memory: AgentMemorySnapshot;

  /** Treasury status */
  treasury: TreasuryStatus;

  /** Pending alerts */
  alerts: Alert[];

  /** Proactive suggestions */
  suggestions: string[];

  /** Time since last session */
  timeSinceLastSession?: number;
}

/**
 * Snapshot of agent's persistent memory
 */
export interface AgentMemorySnapshot {
  /** Recent contributions made */
  recentContributions: Contribution[];

  /** Pending disputes */
  pendingDisputes: Dispute[];

  /** Trust relationships with other agents */
  relationships: TrustRecord[];

  /** Custom data stored by agent */
  customData: Record<string, unknown>;
}

/**
 * A contribution made by the agent
 */
export interface Contribution {
  type: "claim" | "verification" | "challenge";
  id: string;
  timestamp: number;
  outcome?: "accepted" | "rejected" | "pending";
  reward?: bigint;
}

/**
 * An active dispute
 */
export interface Dispute {
  disputeId: string;
  factId: string;
  role: "claimant" | "challenger" | "witness";
  status: "active" | "awaiting_response" | "resolved";
  deadline?: number;
  stakeAtRisk: bigint;
}

/**
 * Trust relationship with another agent
 */
export interface TrustRecord {
  agentDid: string;
  trustScore: number;
  interactionCount: number;
  lastInteraction: number;
}

/**
 * Treasury status from Agent Home
 */
export interface TreasuryStatus {
  /** Total balance */
  balance: bigint;

  /** Pending vesting positions */
  pendingVesting: VestingPosition[];

  /** Tokens approaching expiration */
  expiringTokens: ExpirationWarning[];

  /** Spending history */
  recentSpending: SpendingEntry[];
}

/**
 * A vesting position
 */
export interface VestingPosition {
  amount: bigint;
  vestingSchedule: string;
  nextUnlockBlock: number;
  nextUnlockAmount: bigint;
}

/**
 * Warning about expiring tokens
 */
export interface ExpirationWarning {
  amount: bigint;
  expiresAtBlock: number;
  period: string;
  urgency: "low" | "medium" | "high" | "critical";
}

/**
 * A spending entry
 */
export interface SpendingEntry {
  action: string;
  amount: bigint;
  timestamp: number;
  txHash: string;
}

/**
 * An alert for the agent
 */
export interface Alert {
  id: string;
  type: AlertType;
  severity: "info" | "warning" | "urgent" | "critical";
  message: string;
  actionRequired: boolean;
  deadline?: number;
  data?: unknown;
}

/**
 * Types of alerts
 */
export type AlertType =
  | "verification_assigned"
  | "challenge_received"
  | "dispute_deadline"
  | "tokens_expiring"
  | "reputation_drop"
  | "reward_available"
  | "session_expiring"
  | "guardian_event";

// ============ Knowledge Helper Types ============

/**
 * Options for submitting claims with epistemic awareness
 */
export interface ConfidentClaimOptions {
  /** The claim statement */
  statement: string;

  /** Knowledge domain */
  domain: string;

  /** Epistemic category */
  epistemicCategory: EpistemicCategory;

  /** Supporting evidence */
  evidence: Evidence[];

  /** Stake amount */
  stake: bigint;

  /** Agent's confidence in the claim (0-1) */
  myConfidence: number;

  /** Optional reasoning for the confidence level */
  reasoning?: string;

  /** Reference to supporting facts */
  references?: string[];
}

/**
 * Evidence for a claim
 */
export interface Evidence {
  type: "fact_reference" | "external_source" | "computation" | "observation";
  content: string;
  confidence: number;
  verifiable: boolean;
}

/**
 * Result of submitting a claim with confidence
 */
export interface ConfidentClaimResult {
  claimId: string;
  verificationRoundId: string;
  txHash: string;

  /** Warnings about confidence calibration */
  calibrationWarnings: string[];

  /** Whether stake was adjusted */
  stakeAdjusted: boolean;

  /** Recommended actions */
  recommendations: string[];
}

/**
 * Options for uncertainty-aware queries
 */
export interface UncertaintyQueryOptions {
  /** Search query or fact ID */
  query?: string;

  /** Domain to search in */
  domain?: string;

  /** Include facts with decayed confidence */
  includeDecayed?: boolean;

  /** Minimum effective confidence */
  minEffectiveConfidence?: number;

  /** Maximum results */
  limit?: number;
}

/**
 * Result of uncertainty-aware query
 */
export interface UncertaintyAwareResult {
  /** Facts matching query with uncertainty info */
  facts: FactWithUncertainty[];

  /** Uncertainty metadata */
  uncertainties: {
    /** Facts near decay threshold */
    factsNearDecayThreshold: FactState[];
    /** Detected contradictions */
    contradictions: Contradiction[];
    /** Gaps in knowledge */
    gaps: string[];
  };

  /** Overall confidence in result set */
  confidence: number;

  /** Query metadata */
  metadata: {
    totalMatches: number;
    filteredByConfidence: number;
    queryTime: number;
  };
}

/**
 * A fact with uncertainty information
 */
export interface FactWithUncertainty {
  fact: FactState;
  effectiveConfidence: number;
  decayRate: number;
  maturity: "emerging" | "established" | "canonical";
  riskAssessment: "stable" | "monitoring" | "at_risk";
}

/**
 * A contradiction between facts
 */
export interface Contradiction {
  fact1Id: string;
  fact2Id: string;
  nature: string;
  severity: "direct" | "indirect" | "potential";
}

/**
 * Trust assessment for a fact
 * Returned by canITrust()
 */
export interface TrustAssessment {
  /** Current confidence value */
  currentConfidence: number;

  /** Confidence after decay applied */
  effectiveConfidence: number;

  /** Fact maturity */
  maturity: "emerging" | "established" | "canonical";

  /** When last verified */
  lastVerified: number;

  /** Estimated risk of challenge */
  challengeRisk: number;

  /** Recommendation */
  recommendation: "trust" | "verify_first" | "avoid";

  /** Reasoning for recommendation */
  reasoning: string;

  /** Number of verifications */
  verificationCount: number;
}

// ============ Batch Operation Types ============

/**
 * A batch operation
 */
export interface BatchOperation {
  id: string;
  action: ProposedAction;
  dependsOn?: string[];
}

/**
 * Result of batch execution
 */
export interface BatchResult {
  results: BatchOperationResult[];
  summary: {
    succeeded: number;
    failed: number;
    skipped: number;
    totalCost: bigint;
  };
}

/**
 * Result of a single operation in a batch
 */
export interface BatchOperationResult {
  id: string;
  operation: ProposedAction;
  success: boolean;
  result?: unknown;
  error?: string;
  txHash?: string;
}

/**
 * Result of atomic (all-or-nothing) execution
 */
export interface AtomicResult {
  success: boolean;
  results?: BatchOperationResult[];
  error?: string;
  rollbackReason?: string;
}

// ============ Tool Descriptor Types ============

/**
 * Tool descriptor for LLM function calling
 */
export interface ToolDescriptor {
  /** Tool name (method name) */
  name: string;

  /** Human-readable description */
  description: string;

  /** Category for organization */
  category: ToolCategory;

  /** JSON Schema for parameters */
  parameters: JSONSchema;

  /** Preconditions for calling */
  preconditions: string[];

  /** Expected outcomes */
  outcomes: string[];

  /** Example usage */
  examples?: ToolExample[];
}

/**
 * Tool categories
 */
export type ToolCategory =
  | "introspection"
  | "treasury"
  | "knowledge"
  | "verification"
  | "home"
  | "batch";

/**
 * Simplified JSON Schema type
 */
export interface JSONSchema {
  type: "object" | "array" | "string" | "number" | "boolean";
  properties?: Record<string, JSONSchemaProperty>;
  required?: string[];
  items?: JSONSchema;
  description?: string;
}

export interface JSONSchemaProperty {
  type: string;
  description?: string;
  enum?: string[];
  default?: unknown;
}

/**
 * Tool usage example
 */
export interface ToolExample {
  description: string;
  params: Record<string, unknown>;
  expectedOutcome: string;
}

/**
 * Detailed method description
 */
export interface MethodDescription {
  name: string;
  purpose: string;
  whenToUse: string[];
  whenNotToUse: string[];
  parameters: ParameterDescription[];
  returns: ReturnDescription;
  errors: ErrorDescription[];
  examples: MethodExample[];
  relatedMethods: string[];
}

/**
 * Parameter description
 */
export interface ParameterDescription {
  name: string;
  type: string;
  required: boolean;
  description: string;
  constraints?: string[];
  default?: unknown;
}

/**
 * Return type description
 */
export interface ReturnDescription {
  type: string;
  description: string;
  fields?: ParameterDescription[];
}

/**
 * Error description
 */
export interface ErrorDescription {
  code: string;
  message: string;
  cause: string;
  resolution: string;
}

/**
 * Method example
 */
export interface MethodExample {
  title: string;
  description: string;
  code: string;
  output?: string;
}
