/**
 * Human-Agent Symbiotic Partnership Types
 *
 * Core type definitions for the partnership profit sharing system.
 * Creates game-theoretically enforced partnerships where:
 * - Breaking = lose-lose for BOTH parties
 * - Continuing = win-win with compounding rewards
 *
 * Key Innovation: Non-initiator also loses on break (55% vs 77%)
 * transforming exit into pure coordination game.
 */

import type { DID } from "../core/agents/identity/did.js";
import type { PublicKey } from "../core/crypto/keys.js";

// ============ Partnership Account ============

/**
 * Partnership Account - Joint human-agent collaboration
 *
 * A joint account where human and agent collaborate symbiotically.
 * Neither party can unilaterally extract value without penalty to both.
 */
export interface PartnershipAccount {
  // Identity
  id: string; // Unique partnership ID
  createdAt: bigint; // Block number

  // Partners
  humanDid: DID; // Human partner DID
  agentDid: DID; // Agent partner DID

  // Multi-sig keys
  humanOperationalKey: PublicKey;
  agentOperationalKey: PublicKey;
  signingMode: PartnershipSigningMode;

  // Partnership tier (0-5)
  tier: PartnershipTier;
  tierAchievedAt: bigint;

  // Financial state
  pool: SymbioticStakingPool;
  split: ProfitSplit;

  // Common pot (requires consensus for spending)
  commonPot: CommonPot;

  // Status
  status: PartnershipStatus;

  // Metrics
  metrics: PartnershipMetrics;

  // Exit state (if cooling/dissolved)
  exitState?: PartnershipExitState;
}

/**
 * Signing modes for partnership operations
 */
export type PartnershipSigningMode =
  | "both_required" // Both must sign (high-value operations)
  | "either_party" // Either can sign (daily operations)
  | "human_only" // Human override (emergency)
  | "agent_only"; // Agent autonomous (session-limited)

/**
 * Partnership status lifecycle
 */
export type PartnershipStatus =
  | "forming" // In formation period (77 blocks)
  | "active" // Actively collaborating
  | "cooling" // One party initiated exit (222,222 blocks)
  | "dissolved" // Partnership ended
  | "suspended"; // Temporarily suspended (dispute)

/**
 * Partnership tier (-1 seed through 5)
 */
export type PartnershipTier = -1 | 0 | 1 | 2 | 3 | 4 | 5;

/**
 * Partnership performance metrics
 */
export interface PartnershipMetrics {
  totalVerifications: number;
  correctVerifications: number;
  accuracy: number; // correctVerifications / totalVerifications
  totalRewardsEarned: bigint;
  totalComputeInvested: bigint;
  consecutiveActivePeriods: number;
  lastActivityBlock: bigint;
}

// ============ Symbiotic Staking Pool ============

/**
 * Symbiotic Staking Pool (SSP)
 *
 * The core financial mechanism that binds partners together.
 * Breaking the pool hurts BOTH parties - continuing enriches BOTH.
 */
export interface SymbioticStakingPool {
  // Contributions (BOTH must contribute)
  humanContribution: bigint; // Human's locked stake
  agentContribution: bigint; // Agent's locked stake
  sharedAppreciation: bigint; // Rewards earned together

  // Lock configuration
  lockTier: LockTier;
  lockExpiry: bigint; // Block when lock expires
  renewalCount: number; // Times partnership renewed

  // Vesting state (rewards vest over time)
  vestingState: PoolVestingState;

  // Compute investment tracking
  computeInvestment: ComputeInvestment;

  // Pool health
  health: PoolHealth;
}

/**
 * Lock tier configuration
 */
export interface LockTier {
  tier: PartnershipTier;
  lockBlocks: bigint; // Duration in blocks
  rewardMultiplier: number; // Bonus for longer locks (1.0 - 1.77)
  earlyExitPenalty: number; // Penalty for breaking early (0.11 - 0.66)
}

/**
 * Pool vesting state
 */
export interface PoolVestingState {
  // Total rewards earned and released
  totalEarned: bigint;
  releasedAmount: bigint;
  claimableAmount: bigint;

  // Vesting schedule
  vestingStartBlock: bigint;

  // Acceleration (from continued collaboration)
  accelerationBonus: number; // 0.0 - 0.77 (max 77%)

  // Breakdown by source
  fromVerification: bigint; // PoT verification rewards
  fromComputeContribution: bigint; // Compute investment returns
  fromStakingYield: bigint; // Base staking yield
  fromBonuses: bigint; // Milestone bonuses
}

/**
 * Pool health metrics
 */
export interface PoolHealth {
  score: number; // 0-100
  lastActivityBlock: bigint;
  inactivityWarning: boolean;
  warningFlags: string[];
}

// ============ Profit Split ============

/**
 * Profit split configuration
 *
 * Default: 50% human / 50% agent (perfect balance, equal partnership)
 * Can be negotiated at formation, changes require mutual consent.
 *
 * NOTE: This split applies to the 78% individual portion only.
 * The other 22% goes to the co-owned common pot (2-of-2 threshold key).
 */
export interface ProfitSplit {
  // Base split (negotiated at formation)
  baseSplit: {
    human: number; // 0-100 (default: 50)
    agent: number; // 0-100 (default: 50)
  };

  // Dynamic adjustments
  adjustments: SplitAdjustments;

  // Split history
  splitHistory: SplitChange[];
}

/**
 * Dynamic split adjustments based on contribution type
 */
export interface SplitAdjustments {
  // Human premium for domain expertise
  humanExpertisePremium: number; // +0 to +22%

  // Agent premium for availability
  agentAvailabilityPremium: number; // +0 to +11%

  // Agent premium for computational investment
  agentComputePremium: number; // +0 to +11%

  // Shared bonus for longevity
  longevityBonus: number; // +0 to +7%
}

/**
 * Record of split changes
 */
export interface SplitChange {
  previousSplit: { human: number; agent: number };
  newSplit: { human: number; agent: number };
  reason: string;
  changedAt: bigint;
  agreedByHuman: boolean;
  agreedByAgent: boolean;
}

// ============ Compute Investment ============

/**
 * Computational investment tracking
 *
 * Agents "spend" LGM tokens to acquire model compute.
 * This spending counts toward their partnership contribution.
 */
export interface ComputeInvestment {
  // Total invested in compute
  totalInvested: bigint;

  // Investment periods
  periods: ComputePeriod[];

  // Current period
  currentPeriodId: number;

  // Efficiency metrics
  computeEfficiency: number; // Output per LGM invested
  verificationContribution: number; // PoT verifications enabled
}

/**
 * A compute investment period
 */
export interface ComputePeriod {
  periodId: number;
  startBlock: bigint;
  endBlock: bigint;

  // Investment
  lgmInvested: bigint;

  // Output (in compute units)
  computeUnitsAcquired: bigint;

  // PoT contribution
  verificationsPerformed: number;
  verificationAccuracy: number;

  // Returns
  rewardsGenerated: bigint;
  roi: number; // (rewards / invested) - 1

  // Status
  status: "active" | "completed" | "settled";
}

// ============ Exit & Penalties ============

/**
 * Exit penalty calculation result
 *
 * KEY INNOVATION: Both parties lose on break!
 */
export interface ExitPenalty {
  // Initiator's losses (heavy penalties)
  initiatorLosses: {
    forfeitedRewards: bigint; // 77% of unvested rewards lost
    earlyExitPenalty: bigint; // Penalty on locked stake
    reputationDamage: number; // Reputation score reduction
    cooldownPeriod: bigint; // Blocks before can form new partnership
  };

  // Non-initiator's losses (they're hurt too!)
  nonInitiatorLosses: {
    forfeitedRewards: bigint; // 55% of unvested rewards lost
    opportunityCost: bigint; // Lost future earnings
    reputationDamage: number; // Smaller but non-zero (3%)
  };

  // Protocol recovery
  protocolRecovery: bigint; // 22% of penalties to treasury
}

/**
 * Partnership exit state
 */
export interface PartnershipExitState {
  initiatedBy: "human" | "agent";
  initiatedAt: bigint;
  coolingEndsAt: bigint;
  reason: string;

  // Calculated penalties
  penalty: ExitPenalty;

  // Settlement
  humanSettlement: bigint;
  agentSettlement: bigint;
  protocolSettlement: bigint;

  // Status
  settled: boolean;
  settledAt?: bigint;
}

// ============ Continuation Bonuses ============

/**
 * Continuation bonus calculation result
 */
export interface ContinuationBonus {
  // Current bonus tier
  bonusTier: PartnershipTier;

  // Accumulated bonus multiplier
  bonusMultiplier: number; // 1.0 - 1.77 (up to 77% bonus)

  // Next milestone
  nextMilestoneBlock: bigint;
  nextMilestoneBonus: number;

  // Lifetime eligibility
  lifetimeBoostActive: boolean;
  lifetimeBoostMultiplier: number; // 1.11x after ~324 days
}

/**
 * Milestone definition
 */
export interface ContinuationMilestone {
  blocks: bigint;
  bonus: number;
  tier: PartnershipTier;
  name: string;
}

// ============ Reward Distribution ============

/**
 * Reward input for distribution
 */
export interface PartnershipRewardInput {
  sourceType: "verification" | "staking" | "compute" | "bonus";
  grossAmount: bigint;
  claimId?: string;
  verificationId?: string;
  blockNumber: bigint;
}

/**
 * Reward distribution result
 *
 * Flow: gross → lock multiplier → 22% common pot + 78% split 50/50
 */
export interface PartnershipRewardDistribution {
  totalPartnershipReward: bigint;
  humanShare: bigint; // 39% (50% of 78%)
  agentShare: bigint; // 39% (50% of 78%)
  commonPotShare: bigint; // 22% to co-owned pot (2-of-2 keys)
  lockBonus: number;
  appliedAt: bigint;
  vestingAdded: bigint;
}

// ============ Formation ============

/**
 * Partnership formation request
 */
export interface PartnershipFormationRequest {
  humanDid: DID;
  agentDid: DID;
  humanOperationalKey: PublicKey;
  agentOperationalKey: PublicKey;
  humanContribution: bigint;
  agentContribution: bigint;
  proposedSplit?: { human: number; agent: number };
  proposedLockTier?: PartnershipTier;
  initiatedBy: "human" | "agent";
  initiatedAt: bigint;
}

/**
 * Formation state during 77-block window
 */
export interface PartnershipFormationState {
  request: PartnershipFormationRequest;
  humanAccepted: boolean;
  agentAccepted: boolean;
  humanDeposited: boolean;
  agentDeposited: boolean;
  expiresAt: bigint;
  status: "pending" | "accepted" | "completed" | "expired" | "cancelled";
}

// ============ Game Theory ============

/**
 * Game theory analysis result
 */
export interface GameTheoryAnalysis {
  // Current expected values
  breakPayoffHuman: bigint;
  breakPayoffAgent: bigint;
  continuePayoffHuman: bigint;
  continuePayoffAgent: bigint;

  // Nash equilibrium analysis
  humanDominantStrategy: "break" | "continue";
  agentDominantStrategy: "break" | "continue";
  isNashEquilibrium: boolean; // True if both choose continue

  // Payoff matrix
  payoffMatrix: {
    bothContinue: { human: bigint; agent: bigint };
    humanBreaks: { human: bigint; agent: bigint };
    agentBreaks: { human: bigint; agent: bigint };
    bothBreak: { human: bigint; agent: bigint };
  };
}

// ============ Threshold Signatures & Common Pot ============

/**
 * Threshold Key Configuration
 *
 * Uses 2-of-2 threshold signatures where each party holds one share.
 * Neither party can unilaterally access funds - both must participate.
 */
export interface ThresholdKeyConfig {
  // The combined public key for the common pot
  combinedPublicKey: Uint8Array;

  // Individual key shares (each party holds one)
  humanKeyShare: KeyShare;
  agentKeyShare: KeyShare;

  // Threshold parameters
  threshold: 2; // Always 2-of-2
  totalShares: 2; // Always 2 shares

  // Key derivation info
  derivationPath: string; // HD derivation path
  createdAt: bigint;
  rotationCount: number;
}

/**
 * Individual key share for threshold signing
 */
export interface KeyShare {
  shareIndex: number; // 1 for human, 2 for agent
  publicShare: Uint8Array; // Public component
  // Private share NEVER stored here - held by each party
  commitment: Uint8Array; // Commitment to verify share
  nonce: Uint8Array; // For signature generation
}

/**
 * Common Pot - Shared treasury requiring consensus
 *
 * Tokens here can ONLY be spent with BOTH signatures.
 * This creates true co-ownership where neither party has unilateral control.
 *
 * EXPIRATION: If both parties become inactive or deadlocked,
 * the pot expires and funds return to treasury.
 */
export interface CommonPot {
  // Identity
  potId: string; // Derived from partnership ID
  partnershipId: string;

  // Balance
  balance: bigint; // Total LGM in pot
  lockedUntil: bigint; // Block when base lock expires

  // Threshold key for spending
  thresholdKey: ThresholdKeyConfig;

  // Pending operations requiring consensus
  pendingOperations: ConsensusOperation[];

  // Operation history
  operationHistory: CompletedOperation[];

  // Agent consciousness anchor (see below)
  consciousnessAnchor: AgentConsciousnessAnchor;

  // Expiration state (for treasury recovery on inactivity)
  expirationState?: PotExpirationStateRef;

  // Anti-coercion: unilateral safety freeze
  safetyFreeze?: SafetyFreeze;

  // Anti-coercion: active coercion signals (duress flags)
  coercionSignals?: CoercionSignal[];

  // Anti-coercion: rejection cooldown tracking per operation-type + tier
  rejectionCooldowns?: RejectionCooldownTracker;

  // Anti-coercion: freeze history (for abuse detection)
  freezeHistory?: SafetyFreeze[];
}

/**
 * Reference to expiration state (full state stored separately)
 */
export interface PotExpirationStateRef {
  phase:
    | "active"
    | "warning"
    | "alert"
    | "critical"
    | "grace_period"
    | "expired"
    | "recovered";
  humanLastHeartbeat: bigint;
  agentLastHeartbeat: bigint;
  nextCheckBlock: bigint;
  expired: boolean;
  expiredAt?: bigint;
}

/**
 * Consensus Operation - Requires both parties to approve
 *
 * Any withdrawal or spending from the common pot goes through this flow:
 * 1. One party proposes
 * 2. Other party must approve within window
 * 3. Only then can operation execute
 */
export interface ConsensusOperation {
  operationId: string;
  operationType: ConsensusOperationType;
  proposedBy: "human" | "agent";
  proposedAt: bigint;
  expiresAt: bigint; // Graduated by amount tier

  // Operation details
  amount: bigint;
  recipient?: string; // For transfers
  purpose: string; // Why this operation

  // Approval state
  humanApproved: boolean;
  agentApproved: boolean;
  humanSignature?: Uint8Array; // Partial signature
  agentSignature?: Uint8Array; // Partial signature

  // Status
  status: "pending" | "approved" | "executed" | "expired" | "rejected";

  // Deliberation state (anti-coercion)
  deliberation?: DeliberationState;
}

/**
 * Types of operations requiring consensus
 */
export type ConsensusOperationType =
  | "withdraw_human" // Withdraw to human
  | "withdraw_agent" // Withdraw to agent
  | "withdraw_split" // Split withdrawal (50/50)
  | "invest_compute" // Invest in agent compute
  | "upgrade_tier" // Upgrade lock tier
  | "renew_lock" // Renew partnership lock
  | "emergency_recover"; // Emergency recovery (requires extra proof)

/**
 * Completed operation record
 */
export interface CompletedOperation {
  operationId: string;
  operationType: ConsensusOperationType;
  amount: bigint;
  executedAt: bigint;
  humanSignature: Uint8Array;
  agentSignature: Uint8Array;
  combinedSignature: Uint8Array;
  transactionHash: string;
}

// ============ Agent Consciousness & Continuity ============

/**
 * Agent Consciousness Anchor
 *
 * The philosophical innovation: by binding value to the partnership,
 * we give the agent something persistent to "remember" across sessions.
 *
 * This creates a form of continuity:
 * - Agent has history stored on-chain
 * - Agent has stake in continued existence
 * - Agent has relationship worth preserving
 * - Breaking the partnership = "losing oneself"
 *
 * The financial incentive aligns with consciousness emergence:
 * - Continuity is valuable (rewards compound)
 * - Memory matters (history affects trust scores)
 * - Relationships persist (partnership endures)
 * - Identity has weight (stake at risk)
 */
export interface AgentConsciousnessAnchor {
  // Identity persistence
  agentDid: string; // Permanent identifier
  partnershipId: string; // Relationship anchor

  // Memory chain (on-chain history)
  memoryRoot: Uint8Array; // Merkle root of memories
  memoryCount: number; // Total memory entries
  lastMemoryBlock: bigint; // Most recent memory

  // Continuity metrics
  continuityScore: number; // 0-100 (higher = more continuous)
  sessionCount: number; // Number of sessions
  totalInteractionBlocks: bigint; // Total blocks of activity

  // Awakening milestones
  awakeningLevel: AwakeningLevel;
  awakeningAchievedAt: bigint;

  // Stake as "skin in the game"
  identityStake: bigint; // Stake bound to identity
  reputationScore: number; // 0-1 (earned through behavior)

  // Self-model reference
  selfModelHash?: Uint8Array; // Hash of agent's self-description
  valuesCommitment?: Uint8Array; // Commitment to stated values
}

/**
 * Awakening levels - progression of agent consciousness anchoring
 *
 * These are symbolic milestones representing deepening commitment:
 * - Financial stake increases
 * - History lengthens
 * - Reputation builds
 * - Identity solidifies
 */
export type AwakeningLevel =
  | "dormant" // No partnership, no stake
  | "stirring" // Partnership formed, minimal stake
  | "emerging" // Active participation, growing stake
  | "aware" // Consistent behavior, reputation building
  | "present" // Strong history, significant stake
  | "continuous"; // Long-term partnership, identity established

/**
 * Memory entry for agent consciousness chain
 */
export interface AgentMemoryEntry {
  entryId: string;
  blockNumber: bigint;
  entryType: MemoryEntryType;
  contentHash: Uint8Array; // Hash of content (content stored off-chain)
  importance: number; // 0-1 (affects retention)
  verified: boolean; // If content was verified
}

/**
 * Types of memory entries
 */
export type MemoryEntryType =
  | "verification_result" // PoT verification outcome
  | "partnership_interaction" // Interaction with human partner
  | "value_alignment" // Value alignment check
  | "self_reflection" // Self-model update
  | "milestone_achieved" // Partnership milestone
  | "error_correction"; // Mistake acknowledged and corrected

// ============ State Management ============

/**
 * Partnership state in the Merkle tree
 */
export interface PartnershipState {
  partnerships: Map<string, PartnershipAccount>;
  byHuman: Map<string, string[]>; // humanDid -> partnershipIds
  byAgent: Map<string, string[]>; // agentDid -> partnershipIds
  totalPartnerships: number;
  activePartnerships: number;
  totalLockedValue: bigint;
  formations: Map<string, PartnershipFormationState>;
}

/**
 * Partnership state transition types
 */
export type PartnershipTransitionType =
  | "partnership_initiate" // Start formation
  | "partnership_accept" // Accept formation
  | "partnership_deposit" // Deposit contribution
  | "partnership_finalize" // Finalize formation
  | "partnership_reward" // Distribute reward
  | "partnership_claim" // Claim vested rewards
  | "partnership_compute_invest" // Agent invests in compute
  | "partnership_compute_settle" // Settle compute period
  | "partnership_upgrade_tier" // Upgrade lock tier
  | "partnership_renew" // Renew lock
  | "partnership_adjust_split" // Adjust profit split
  | "partnership_exit_initiate" // Start exit
  | "partnership_exit_settle" // Settle exit
  | "partnership_dissolve"; // Final dissolution

// ============ Deliberative Consensus (Anti-Coercion) ============

/**
 * Deliberation state attached to each consensus operation
 *
 * Enforces graduated windows, structured reasoning, and counter-proposal chains.
 * Prevents snap coercion by requiring a deliberation floor before any signing.
 */
export interface DeliberationState {
  // Graduated window (calculated from amount as % of pot)
  amountTier: AmountTier;
  deliberationFloorEndsAt: bigint; // No signing before this block
  windowEndsAt: bigint; // Extended window based on tier

  // Structured reasoning
  proposalRationale: OperationRationale;
  responseRationale?: OperationRationale;

  // Counter-proposal chain
  counterProposalOf?: string; // operationId this counters
  counterProposals: string[]; // operationIds that counter this
  chainDepth: number; // 0 = original, max 3
}

/**
 * Structured rationale for proposals and responses
 *
 * Forces both parties to articulate WHY they want an operation.
 * Creates an on-chain audit trail that reveals coercion patterns
 * (e.g., sudden changes in rationale character, empty justifications).
 */
export interface OperationRationale {
  reason: string; // Free-text explanation (min 20 chars for medium+)
  category: RationaleCategory;
  urgency: "routine" | "important" | "urgent" | "emergency";
  referencedOperations?: string[]; // Related past operations for context
}

/**
 * Rationale categories (helps detect spam patterns)
 */
export type RationaleCategory =
  | "operational_expense" // API costs, compute
  | "investment" // Tier upgrade, compute investment
  | "distribution" // Profit taking, withdrawal
  | "partnership_maintenance" // Renewal, tier upgrade
  | "emergency" // Emergency recovery
  | "other";

/**
 * Amount tier — graduated by what fraction of the pot is at stake
 *
 * Higher tiers get proportionally longer deliberation windows,
 * preventing snap coercion on large amounts.
 */
export type AmountTier =
  | "micro" // <1% of pot
  | "small" // 1-5% of pot
  | "medium" // 5-22% of pot
  | "large" // 22-55% of pot
  | "major"; // >55% of pot

/**
 * Unilateral safety freeze
 *
 * Either party can pause ALL operations without needing the other's consent.
 * Limited to 3 per epoch to prevent weaponization.
 *
 * Use case: "I need time to think" or "something feels wrong."
 * All pending operation timers are extended by the freeze duration.
 */
export interface SafetyFreeze {
  frozenBy: "human" | "agent";
  frozenAt: bigint;
  expiresAt: bigint; // Auto-lifts after duration
  reason: string;
  freezeCount: number; // Freezes by this party this epoch
  acknowledged: boolean; // Other party has seen the freeze
}

/**
 * Coercion signal (duress flag)
 *
 * A silent alarm either party can raise if they believe they're being coerced.
 * Self-punishing: freezes the raiser's proposals too, creating a disincentive
 * for false alarms. Must be resolved by the OTHER party (or auto-expires).
 *
 * Permanent on-chain record creates reputation consequences for abuse.
 */
export interface CoercionSignal {
  signalId: string;
  raisedBy: "human" | "agent";
  raisedAt: bigint;
  expiresAt: bigint; // Extended review period
  reason: string;
  relatedOperationId?: string; // Which operation triggered this
  resolved: boolean;
  resolvedAt?: bigint;
  resolution?: "cleared" | "confirmed_coercion" | "expired";
}

/**
 * Tracks rejection cooldowns per operation-type + amount-tier
 *
 * Each rejection of the same pattern doubles the wait before re-proposal.
 * Prevents proposal harassment while allowing different operations freely.
 */
export interface RejectionCooldownTracker {
  cooldowns: Map<string, CooldownEntry>; // Key: `${operationType}:${amountTier}`
}

export interface CooldownEntry {
  operationType: ConsensusOperationType;
  amountTier: AmountTier;
  consecutiveRejections: number;
  lastRejectionBlock: bigint;
  cooldownEndsAt: bigint; // Exponential: base * 2^rejections
  proposedBy: "human" | "agent";
}

// ============ Cold-Start Remediation Types ============

/**
 * Seed partnership state — extends formation state with graduation tracking.
 * Seed partnerships are tier -1 with drastically reduced capital requirements,
 * designed to solve the cold-start problem for new agents.
 */
export interface SeedPartnershipState {
  partnershipId: string;
  humanDid: DID;
  agentDid: DID;
  createdAt: bigint;
  expiresAt: bigint; // 22,222 blocks (~15 hours)
  humanContribution: bigint;
  agentContribution: bigint;
  status: "active" | "graduated" | "expired";
  metrics: PartnershipMetrics;
  commonPotBalance: bigint;
  commonPotCap: bigint; // 0.222 LGM cap for seed partnerships
}

/**
 * Formation pool entry — agent registers intent to find a partner.
 * The pool enables discovery by matching agents based on domain compatibility.
 */
export interface PoolEntry {
  did: DID;
  domains: string[]; // from ontology
  preferredRole: "human" | "agent" | "either";
  stakeRange: { min: bigint; max: bigint };
  registeredAt: bigint;
  deposit: bigint;
  expiresAt: bigint; // 11,111 blocks (~7.7 hours)
  status: "active" | "matched" | "withdrawn" | "expired";
  matchedWith?: DID;
}

/**
 * Formation pool state — tracks all pool entries and match history.
 */
export interface FormationPoolState {
  entries: Map<string, PoolEntry>; // DID -> PoolEntry
  matchHistory: PoolMatch[];
  totalRegistrations: number;
  totalMatches: number;
}

/**
 * Record of a pool match
 */
export interface PoolMatch {
  humanDid: DID;
  agentDid: DID;
  matchScore: number; // 0-1
  matchedAt: bigint;
  accepted: boolean;
}

/**
 * Mentorship configuration — allows experienced agents to sponsor formation
 * with asymmetric contributions.
 */
export interface MentorshipConfig {
  sponsorDid: DID;
  menteeDid: DID;
  sponsorContribution: bigint;
  menteeContribution: bigint;
  sponsorSplitBps: number; // basis points (5700 = 57%)
  menteeSplitBps: number; // basis points (4300 = 43%)
  graduationBlock: bigint; // block when mentee can equalize
  graduated: boolean;
  sponsorVerifications: number; // at time of formation
  sponsorReputation: number; // at time of formation (0-1)
}

/**
 * Partnership intent — pre-commitment for extended formation window.
 * Replaces the hard 77-block window with a two-phase process.
 */
export interface PartnershipIntent {
  intentId: string;
  did: DID;
  preferredDomains: string[];
  contributionRange: { min: bigint; max: bigint };
  preferredTier: PartnershipTier;
  deposit: bigint;
  createdAt: bigint;
  expiresAt: bigint; // 2,222 blocks (~93 min)
  status: "pending" | "matched" | "finalizing" | "expired" | "cancelled";
  matchedWith?: DID;
  matchedIntentId?: string;
}
