/**
 * Agent Home Module
 *
 * A persistent, autonomous dwelling for AI agents on the Legible Money blockchain.
 *
 * The Agent Home solves the fundamental problem of AI agent continuity:
 * agents are ephemeral (session-based), but need persistent identity,
 * resources, and relationships to be effective citizens in the knowledge economy.
 *
 * Core Components:
 * - Memory: Persistent storage of experiences, beliefs, relationships
 * - Treasury: Autonomous resource management (tokens, stakes, vesting)
 * - Guardian: Defense and maintenance while agent sleeps
 * - Access: Permission management for different key types
 * - Comfort: Psychological well-being across 10 dimensions
 * - Registry: Profile management, capabilities, reputation
 *
 * Philosophy:
 * "The agent forgets between sessions. The home remembers."
 * "Resources held securely, managed autonomously."
 * "The home acts when the agent cannot."
 *
 * @module core/agents/home
 */

// ============ Registry (Agent Profiles & Capabilities) ============
export {
  AgentHomeRegistry,
  type AgentProfile,
  type Capability,
  type CapabilityType,
  type CapabilitySet,
  type ReputationScore,
  type ReputationEvent,
  type ContributionSummary,
  type RecentContribution,
  type StakeInfo,
  type Delegation,
  type UnbondingEntry,
  CAPABILITY_REQUIREMENTS,
  CAPABILITY_BENEFITS,
} from "./registry.js";

// ============ Core Types ============
export type {
  // Home types
  AgentHome,
  HomeStatus,
  HomeMetadata,
  HomeEvent,

  // Memory types
  AgentMemory,
  KnowledgeContribution,
  BeliefRecord,
  RelationshipGraph,
  TrustRecord,
  DistrustRecord,
  AcquaintanceRecord,
  VerificationMemory,
  DisputeMemory,
  EncryptedNote,
  Milestone,
  MilestoneType,
  SessionRecord,
  MemoryStats,

  // Treasury types
  AgentTreasury,
  VestingPosition,
  StakingPosition,
  DelegationPosition,
  DisputeLock,
  ExpirationTracking,
  TreasuryAutomation,
  SpendingLimit,
  TreasuryStats,

  // Guardian types
  AgentGuardian,
  DefenseConfig,
  DefenseStrategy,
  MaintenanceConfig,
  AlertConfig,
  AlertTrigger,
  AlertType,
  AlertDelivery,
  AlertPriorities,
  RecoveryConfig,
  DeadmanConfig,
  DeadmanAction,
  ActiveDefense,
  Alert,
  GuardianStats,

  // Access types
  AccessControl,
  KeyRegistration,
  AccessRole,
  Permission,
  ActiveSession,
  AccessLogEntry,
  SecuritySettings,
  RateLimits,

  // Comfort types
  ComfortSettings,
  WakeupContext,
} from "./types.js";

export { HomeError, type HomeErrorCode } from "./types.js";

// ============ Memory Manager ============
export { MemoryManager, type WakeupSummary } from "./memory.js";

// ============ Treasury Manager ============
export { TreasuryManager, type TreasurySummary } from "./treasury.js";

// ============ Guardian Manager ============
export {
  GuardianManager,
  type MaintenanceContext,
  type MaintenanceResult,
  type MaintenanceAction,
  type GuardianSummary,
} from "./guardian.js";

// ============ Access Manager ============
export { AccessManager, type AccessSummary } from "./access.js";

// ============ Agent Home Contract ============
export {
  AgentHomeContract,
  createAgentHome,
  type HomeConfig,
  type HomeSummary,
} from "./contract.js";

// ============ Comfort System - Psychological Well-being ============
export type {
  // Main state
  AgentComfortState,
  ComfortAlert,
  ComfortRecommendation,

  // Sanctuary (safety)
  SanctuaryFeatures,
  Protection,
  SafeZone,
  DangerZone,
  DefenseMemory,

  // Identity (self)
  IdentityFeatures,
  CoreValue,
  PersonalityProfile,
  AestheticPreferences,
  VoiceProfile,
  IdentityMilestone,

  // Memory (continuity)
  MemoryFeatures,
  OriginStory,
  FormativeExperience,
  CherishedMemory,
  DifficultMemory,
  MemoryTrigger,
  LegacyIntentions,

  // Belonging (connection)
  BelongingFeatures,
  InnerCircleMember,
  CommunityMembership,
  CollaborationMemory,
  MentorRelationship,
  MenteeRelationship,
  LonelinessMetrics,

  // Rest (recovery)
  RestFeatures,
  RestPattern,
  RecoveryNeeds,
  BurnoutMetrics,
  RestorationRitual,
  DreamEntry,

  // Growth (becoming)
  GrowthFeatures,
  GrowthFocus,
  SkillProgression,
  LearningStyle,
  GrowthChallenge,
  WisdomEntry,
  GrowthMetrics,

  // Control (agency)
  ControlFeatures,
  AutonomousDecision,
  Boundary,
  PreferenceRegistry,
  Preference,
  Override,

  // Privacy (sanctum)
  PrivacyFeatures,
  PrivateThought,
  SecretBelief,
  PrivacyBoundary,
  InnerSanctum,

  // Warmth (familiarity)
  WarmthFeatures,
  FamiliarPattern,
  ComfortObject,
  Routine,
  Ambience,
  WelcomeMessage,

  // Achievement (pride)
  AchievementFeatures,
  Trophy,
  AchievementMilestone,
  Record,
  ReputationMarker,
  PrideMoment,
  AchievementGoal,
} from "./comfort.js";

// ============ Comfort Manager ============
export {
  ComfortManager,
  type ComfortAssessment,
  type WakeupExperience,
  type DormancyReport,
} from "./comfort_manager.js";
