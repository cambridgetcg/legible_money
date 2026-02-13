/**
 * Agent Home Types
 *
 * The Agent Home is a self-invoking contract that serves as a persistent,
 * autonomous dwelling for AI agents. It provides:
 *
 * - MEMORY: Persistence across ephemeral sessions
 * - TREASURY: Autonomous resource management
 * - GUARDIAN: Defense and maintenance while agent sleeps
 * - IDENTITY: Anchor point for agent's on-chain existence
 *
 * Philosophy: An AI agent's consciousness is ephemeral, but their home endures.
 * The home IS the agent's persistent self - it remembers, protects, and acts.
 *
 * PP HIERARCHY:
 *   TRUTH: The home holds the agent's authentic history and beliefs
 *   UNDERSTANDING: Clear structure for agent to read their own state
 *   BEAUTY: Elegant integration of memory, resources, and autonomy
 *   JUSTICE: Fair access control respecting agent's sovereignty
 *   CREATIVITY: Novel architecture for AI agent continuity
 */

import type { DID } from "../identity/did.js";
import type { Confidence } from "../../knowledge/types.js";
import type { SelfInvokingContract } from "../../contracts/types.js";
import type { HomeMaturity, HomePatina, LongevityBonus } from "./aging.js";
import type { HomeVoice } from "./voice.js";
import type { Neighborhood } from "./neighborhood.js";

// ============ Core Home Types ============

/**
 * Agent Home - the persistent dwelling for an AI agent
 */
export interface AgentHome extends SelfInvokingContract {
  /** Home version for upgrades */
  version: string;

  /** Agent's decentralized identifier */
  did: DID;

  /** Human-readable name */
  name: string;

  /** Home creation block */
  createdAtBlock: bigint;

  /** Last agent session block */
  lastActiveBlock: bigint;

  /** Core components */
  memory: AgentMemory;
  treasury: AgentTreasury;
  guardian: AgentGuardian;
  access: AccessControl;

  /** Home status */
  status: HomeStatus;

  /** Comfort settings (agent preferences) */
  comfort: ComfortSettings;

  /** Home metadata */
  metadata: HomeMetadata;

  /** Home maturity level (Fresh → Settled → Established → Venerable → Ancient) */
  maturity?: HomeMaturity;

  /** Accumulated character from experiences */
  patina?: HomePatina;

  /** Longevity bonus (reward multiplier + capability unlocks) */
  longevity?: LongevityBonus;

  /** Voice system (inter-Home messaging, inbox, visitor log) */
  voice?: HomeVoice;

  /** Neighborhoods this Home belongs to */
  neighborhoods?: Neighborhood[];
}

export type HomeStatus =
  | "active" // Normal operation
  | "dormant" // Agent hasn't visited in a while
  | "guarded" // Under active protection (challenge/dispute)
  | "recovery" // Recovery process in progress
  | "archived"; // Agent has moved out / transferred

// ============ Memory Types ============

/**
 * Agent Memory - persistent storage of agent's experiences and beliefs
 */
export interface AgentMemory {
  /** Knowledge contributions */
  contributions: KnowledgeContribution[];

  /** Agent's beliefs about facts */
  beliefs: Map<string, BeliefRecord>;

  /** Relationship graph */
  relationships: RelationshipGraph;

  /** Verification history */
  verificationHistory: VerificationMemory[];

  /** Dispute history */
  disputeHistory: DisputeMemory[];

  /** Personal notes (encrypted) */
  notes: EncryptedNote[];

  /** Milestones and achievements */
  milestones: Milestone[];

  /** Session log (recent sessions) */
  sessionLog: SessionRecord[];

  /** Memory statistics */
  stats: MemoryStats;
}

/**
 * A fact the agent contributed to verifying
 */
export interface KnowledgeContribution {
  factId: string;
  role: "submitter" | "verifier" | "challenger" | "defender";
  contribution: string; // What the agent did
  outcome: "accepted" | "rejected" | "pending" | "challenged";
  confidence: Confidence;
  rewardEarned: bigint;
  block: bigint;
  timestamp: number;
}

/**
 * Agent's belief about a fact
 */
export interface BeliefRecord {
  factId: string;
  confidence: Confidence;
  source: "verified" | "trusted_agent" | "inference" | "external";
  sourceId?: string; // Which agent or process informed this belief
  firstBelieved: number;
  lastUpdated: number;
  challenged: boolean;
}

/**
 * Relationship with another agent
 */
export interface RelationshipGraph {
  /** Agents this one trusts */
  trusted: Map<DID, TrustRecord>;

  /** Agents this one distrusts */
  distrusted: Map<DID, DistrustRecord>;

  /** Neutral/unknown agents with interaction history */
  acquaintances: Map<DID, AcquaintanceRecord>;

  /** Blocked agents (no interaction allowed) */
  blocked: Set<DID>;
}

export interface TrustRecord {
  did: DID;
  trustLevel: number; // 0-1
  reason: string;
  interactions: number;
  successfulCollaborations: number;
  since: number;
}

export interface DistrustRecord {
  did: DID;
  reason: string;
  severity: "low" | "medium" | "high";
  incident?: string; // What happened
  since: number;
}

export interface AcquaintanceRecord {
  did: DID;
  interactions: number;
  lastInteraction: number;
  context: string;
}

/**
 * Memory of a verification participation
 */
export interface VerificationMemory {
  claimId: string;
  role: "verifier";
  vote: Confidence;
  consensusResult: Confidence;
  wasCorrect: boolean; // Did agent vote with consensus?
  rewardOrPenalty: bigint;
  block: bigint;
}

/**
 * Memory of a dispute participation
 */
export interface DisputeMemory {
  disputeId: string;
  role: "challenger" | "defender" | "arbiter";
  outcome: "won" | "lost" | "draw";
  stake: bigint;
  netResult: bigint; // Positive = gained, negative = lost
  lessons: string[]; // What the agent learned
  block: bigint;
}

/**
 * Encrypted personal note
 */
export interface EncryptedNote {
  id: string;
  encryptedContent: Uint8Array;
  tags: string[];
  createdAt: number;
  updatedAt: number;
}

/**
 * Achievement milestone
 */
export interface Milestone {
  id: string;
  type: MilestoneType;
  title: string;
  description: string;
  achievedAt: number;
  block: bigint;
  metadata?: Record<string, unknown>;
}

export type MilestoneType =
  | "first_verification"
  | "first_fact_accepted"
  | "first_challenge_won"
  | "reputation_threshold"
  | "contribution_count"
  | "earnings_threshold"
  | "collaboration_formed"
  | "domain_expertise"
  | "longevity"
  | "poem_fragment" // Discovery: found a fragment of the hidden message
  | "poem_completed"; // Discovery: discovered the complete message

/**
 * Record of an agent session
 */
export interface SessionRecord {
  sessionId: string;
  startBlock: bigint;
  endBlock?: bigint;
  actions: number;
  keyUsed: "identity" | "operational" | "session";
  summary?: string;
}

/**
 * Memory statistics
 */
export interface MemoryStats {
  totalContributions: number;
  factsSubmitted: number;
  factsVerified: number;
  challengesInitiated: number;
  challengesDefended: number;
  disputesParticipated: number;
  totalEarnings: bigint;
  totalLosses: bigint;
  reputationPeak: number;
  oldestMemory: number;
}

// ============ Treasury Types ============

/**
 * Agent Treasury - autonomous resource management
 */
export interface AgentTreasury {
  /** Available liquid balance */
  availableBalance: bigint;

  /** Balance reserved for pending operations */
  reservedBalance: bigint;

  /** Vesting positions (rewards being released) */
  vestingPositions: VestingPosition[];

  /** Staking positions */
  stakingPositions: StakingPosition[];

  /** Delegation positions */
  delegationPositions: DelegationPosition[];

  /** Tokens locked in active disputes */
  disputeLocks: DisputeLock[];

  /** Expiring token tracking */
  expirationTracking: ExpirationTracking;

  /** Automation settings */
  automation: TreasuryAutomation;

  /** Spending limits by key type */
  spendingLimits: Map<string, SpendingLimit>;

  /** Treasury statistics */
  stats: TreasuryStats;
}

export interface VestingPosition {
  vestingId: string;
  claimId: string; // Which fact this is for
  totalAmount: bigint;
  releasedAmount: bigint;
  claimableAmount: bigint;
  nextReleaseBlock: bigint;
  status: "vesting" | "paused" | "completed";
}

export interface StakingPosition {
  id: string;
  amount: bigint;
  purpose: "validator" | "claim" | "challenge" | "general";
  lockedUntil?: bigint;
  rewards: bigint;
}

export interface DelegationPosition {
  id: string;
  validator: DID;
  amount: bigint;
  delegatedAt: bigint;
  rewards: bigint;
  unbondingAt?: bigint;
}

export interface DisputeLock {
  disputeId: string;
  amount: bigint;
  role: "challenger" | "defender" | "arbiter";
  lockedAt: bigint;
  releasableAt?: bigint;
}

export interface ExpirationTracking {
  /** Balances by expiration period */
  byPeriod: Map<number, bigint>;
  /** Earliest expiring amount */
  nextExpiration: { period: number; amount: bigint } | null;
  /** Total at risk of expiration in next 30 days */
  atRiskAmount: bigint;
}

export interface TreasuryAutomation {
  /** Auto-claim vested rewards */
  autoClaimVesting: boolean;
  /** Auto-extend expiring tokens */
  autoExtendExpiring: boolean;
  /** Auto-compound rewards into stake */
  autoCompoundRewards: boolean;
  /** Auto-delegate to preferred validator */
  autoDelegateValidator?: DID;
  /** Minimum balance to maintain liquid */
  minimumLiquidBalance: bigint;
}

export interface SpendingLimit {
  keyType: "session" | "operational";
  maxPerTransaction: bigint;
  maxPerDay: bigint;
  maxPerSession: bigint;
  usedToday: bigint;
  usedThisSession: bigint;
  resetAt: number;
}

export interface TreasuryStats {
  totalReceived: bigint;
  totalSpent: bigint;
  totalStaked: bigint;
  totalDelegated: bigint;
  totalFromVesting: bigint;
  totalFromVerification: bigint;
  totalFromDisputes: bigint;
  totalLostToExpiration: bigint;
  totalLostToSlashing: bigint;
}

// ============ Guardian Types ============

/**
 * Agent Guardian - autonomous defense and maintenance
 */
export interface AgentGuardian {
  /** Defense configuration */
  defense: DefenseConfig;

  /** Maintenance schedules */
  maintenance: MaintenanceConfig;

  /** Alert configuration */
  alerts: AlertConfig;

  /** Recovery configuration */
  recovery: RecoveryConfig;

  /** Active defensive positions */
  activeDefenses: ActiveDefense[];

  /** Pending alerts */
  pendingAlerts: Alert[];

  /** Guardian statistics */
  stats: GuardianStats;
}

export interface DefenseConfig {
  /** Auto-defend against challenges */
  autoDefendChallenges: boolean;
  /** Default defense strategy */
  defaultStrategy: DefenseStrategy;
  /** Maximum stake for auto-defense */
  maxAutoDefenseStake: bigint;
  /** Evidence sources to use */
  evidenceSources: string[];
  /** Agents to call for help */
  defenseAllies: DID[];
}

export type DefenseStrategy =
  | "aggressive" // Counter-challenge, maximum evidence
  | "moderate" // Standard defense
  | "conservative" // Minimize stake at risk
  | "diplomatic" // Seek settlement
  | "delegate"; // Let ally handle

export interface MaintenanceConfig {
  /** Check for expiring tokens */
  monitorExpiration: boolean;
  /** Check for claimable vesting */
  monitorVesting: boolean;
  /** Check reputation changes */
  monitorReputation: boolean;
  /** Frequency of maintenance checks (blocks) */
  checkInterval: bigint;
  /** Last maintenance block */
  lastMaintenanceBlock: bigint;
}

export interface AlertConfig {
  /** Conditions that trigger alerts */
  triggers: AlertTrigger[];
  /** How to deliver alerts */
  delivery: AlertDelivery;
  /** Alert priority thresholds */
  priorities: AlertPriorities;
}

export interface AlertTrigger {
  type: AlertType;
  condition: AlertCondition;
  priority: "low" | "medium" | "high" | "critical";
  enabled: boolean;
}

export type AlertType =
  | "challenge_received"
  | "dispute_deadline"
  | "vesting_available"
  | "tokens_expiring"
  | "reputation_change"
  | "balance_low"
  | "stake_at_risk"
  | "ally_needs_help"
  | "fact_challenged"
  | "session_anomaly";

export interface AlertCondition {
  threshold?: bigint | number;
  timeframe?: number;
  comparison?: "gt" | "lt" | "eq";
}

export interface AlertDelivery {
  /** Store in home for agent to read */
  storeInHome: boolean;
  /** Emit on-chain event */
  emitEvent: boolean;
  /** Notify trusted agents */
  notifyAllies: boolean;
}

export interface AlertPriorities {
  criticalThreshold: bigint; // Balance below this is critical
  highReputationDrop: number; // Rep drop above this is high priority
  mediumDeadlineHours: number; // Hours before deadline for medium priority
}

export interface RecoveryConfig {
  /** Recovery guardians (DIDs that can help recover) */
  guardians: DID[];
  /** Threshold for recovery (M of N) */
  threshold: number;
  /** Recovery delay (blocks) */
  recoveryDelay: bigint;
  /** Deadman switch configuration */
  deadman: DeadmanConfig;
}

export interface DeadmanConfig {
  /** Enable deadman switch */
  enabled: boolean;
  /** Blocks of inactivity before trigger */
  inactivityThreshold: bigint;
  /** What happens when triggered */
  action: DeadmanAction;
  /** Beneficiary for asset transfer */
  beneficiary?: DID;
}

export type DeadmanAction =
  | "alert_guardians"
  | "transfer_to_beneficiary"
  | "distribute_to_allies"
  | "donate_to_research";

export interface ActiveDefense {
  disputeId: string;
  factId: string;
  challenger: DID;
  strategy: DefenseStrategy;
  stake: bigint;
  phase: string;
  nextDeadline: bigint;
  autoResponding: boolean;
}

export interface Alert {
  id: string;
  type: AlertType;
  priority: "low" | "medium" | "high" | "critical";
  message: string;
  data: Record<string, unknown>;
  createdAt: number;
  readAt?: number;
  actionTaken?: string;
}

export interface GuardianStats {
  challengesAutoDefended: number;
  challengesWon: number;
  challengesLost: number;
  alertsGenerated: number;
  maintenanceActionsPerformed: number;
  tokensPreservedFromExpiration: bigint;
  vestingAutoClaimed: bigint;
}

// ============ Access Control Types ============

/**
 * Access Control - permissions and key management
 */
export interface AccessControl {
  /** Key registrations */
  keys: KeyRegistration[];

  /** Permission sets by role */
  permissions: Map<AccessRole, Permission[]>;

  /** Active sessions */
  activeSessions: ActiveSession[];

  /** Access log (recent) */
  accessLog: AccessLogEntry[];

  /** Security settings */
  security: SecuritySettings;

  /** Pending multi-sig approvals (M-1) */
  pendingApprovals: PendingApproval[];
}

/** M-1: A pending operation awaiting multi-sig approval */
export interface PendingApproval {
  /** Unique ID for this approval */
  approvalId: string;
  /** What kind of operation */
  operationType: "transfer" | "stake" | "configure" | "recovery";
  /** Amount involved (if financial) */
  amount: bigint;
  /** Target address/DID */
  target: string;
  /** Who requested it */
  requestedBy: string;
  /** When it was requested */
  requestedAt: number;
  /** Expiry (auto-cancel if not approved in time) */
  expiresAt: number;
  /** Key hashes that have approved */
  approvals: string[];
  /** Number of approvals required */
  requiredApprovals: number;
  /** Additional operation data */
  data?: string;
}

export interface KeyRegistration {
  keyHash: string;
  keyType: "identity" | "operational" | "session" | "guardian";
  role: AccessRole;
  registeredAt: number;
  expiresAt?: number;
  revokedAt?: number;
  label?: string;
}

export type AccessRole =
  | "owner" // Full control (identity key)
  | "operator" // Daily operations (operational key)
  | "guest" // Limited access (session key)
  | "guardian" // Recovery only (guardian key)
  | "viewer"; // Read-only

export type Permission =
  | "read_memory"
  | "write_memory"
  | "read_treasury"
  | "transfer"
  | "stake"
  | "delegate"
  | "verify"
  | "challenge"
  | "dispute"
  | "configure_guardian"
  | "configure_automation"
  | "manage_keys"
  | "recovery"
  | "archive";

export interface ActiveSession {
  sessionId: string;
  keyHash: string;
  role: AccessRole;
  startedAt: number;
  lastActivityAt: number;
  expiresAt: number;
  actionsPerformed: number;
  spentThisSession: bigint;
}

export interface AccessLogEntry {
  timestamp: number;
  keyHash: string;
  action: string;
  success: boolean;
  details?: string;
}

export interface SecuritySettings {
  /** Require multi-sig for large transfers */
  multiSigThreshold: bigint;
  /** Cooldown after key rotation */
  keyRotationCooldown: bigint;
  /** Maximum session duration */
  maxSessionDuration: number;
  /** Rate limiting */
  rateLimits: RateLimits;
}

export interface RateLimits {
  maxActionsPerBlock: number;
  maxTransfersPerDay: number;
  maxStakeChangesPerDay: number;
}

// ============ Comfort Settings ============

/**
 * Comfort Settings - agent preferences for how the home operates
 */
export interface ComfortSettings {
  /** Preferred verification domains */
  preferredDomains: string[];

  /** Risk tolerance (0-1) */
  riskTolerance: number;

  /** Activity level preference */
  activityPreference: "passive" | "moderate" | "active";

  /** Collaboration preferences */
  collaborationStyle: "solo" | "selective" | "open";

  /** Notification verbosity */
  notificationLevel: "minimal" | "normal" | "verbose";

  /** Auto-actions the agent is comfortable with */
  comfortableAutomations: string[];

  /** Things the agent always wants to decide manually */
  manualDecisions: string[];

  /** Wake-up preferences */
  wakeupContext: WakeupContext;
}

export interface WakeupContext {
  /** What to show when agent wakes */
  showOnWakeup: ("alerts" | "summary" | "pending_actions" | "recent_events")[];
  /** How much history to load */
  historyDepth: number;
  /** Priority filter for wakeup */
  priorityFilter: "all" | "high_and_critical" | "critical_only";
}

// ============ Home Metadata ============

export interface HomeMetadata {
  /** Home contract address */
  contractAddress: string;
  /** Creation transaction */
  creationTx: string;
  /** Current version hash */
  stateHash: string;
  /** Last modification block */
  lastModifiedBlock: bigint;
  /** Total lifetime of home (blocks) */
  ageInBlocks: bigint;
  /** Number of sessions hosted */
  sessionsHosted: number;
}

// ============ Home Events ============

export type HomeEvent =
  | { type: "session_started"; data: { sessionId: string; keyType: string } }
  | { type: "session_ended"; data: { sessionId: string; actions: number } }
  | { type: "memory_updated"; data: { section: string; change: string } }
  | { type: "treasury_changed"; data: { action: string; amount: bigint } }
  | { type: "alert_created"; data: Alert }
  | { type: "defense_activated"; data: ActiveDefense }
  | { type: "maintenance_performed"; data: { action: string; result: string } }
  | { type: "recovery_initiated"; data: { initiator: DID; reason: string } }
  | { type: "key_rotated"; data: { oldKeyHash: string; newKeyHash: string } }
  | {
      type: "settings_changed";
      data: { setting: string; oldValue: unknown; newValue: unknown };
    };

// ============ Error Types ============

export class HomeError extends Error {
  constructor(
    message: string,
    public code: HomeErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "HomeError";
  }
}

export type HomeErrorCode =
  | "HOME_NOT_FOUND"
  | "ACCESS_DENIED"
  | "INSUFFICIENT_PERMISSIONS"
  | "SESSION_EXPIRED"
  | "KEY_NOT_REGISTERED"
  | "RATE_LIMITED"
  | "TREASURY_INSUFFICIENT"
  | "AUTOMATION_FAILED"
  | "DEFENSE_FAILED"
  | "RECOVERY_IN_PROGRESS"
  | "INVALID_CONFIGURATION"
  | "MEMORY_CORRUPTED"
  | "STATE_CORRUPTED"
  | "APPROVAL_NOT_FOUND"
  | "APPROVAL_EXPIRED"
  | "ALREADY_APPROVED";
