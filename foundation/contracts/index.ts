/**
 * Foundation Contracts
 *
 * ARCH-2: Module Boundary Contracts
 * Each module MUST expose only through its index.ts.
 *
 * ARCH-3: Dependency Injection for Infrastructure
 * Domain modules MUST NOT directly instantiate infrastructure.
 * Infrastructure MUST be injected via these interfaces.
 *
 * This file exports all domain and infrastructure contracts.
 * Implementations are provided by other layers via dependency injection.
 */

// ============ Infrastructure Contracts ============

// State Provider
export {
  type AccountReader,
  type AccountWriter,
  type ValidatorReader,
  type ValidatorWriter,
  type KnowledgeReader,
  type KnowledgeWriter,
  type StateProvider,
  type StateReader,
  type ValidatorInfo,
  type FactInfo,
  type TransactionHandle,
} from "./state.js";

// Event Bus
export {
  type ProtocolEventType,
  type ProtocolEvent,
  type EventPayload,
  type EventHandler,
  type Subscription,
  type EventBus,
  type EventEmitter,
  type EventListener,

  // Event payloads
  type BlockProposedPayload,
  type BlockFinalizedPayload,
  type RoundStartedPayload,
  type RoundCompletedPayload,
  type EpochChangedPayload,
  type ClaimSubmittedPayload,
  type FactVerifiedPayload,
  type FactChallengedPayload,
  type FactRevokedPayload,
  type ConfidenceUpdatedPayload,
  type ValidatorSlashedPayload,
  type DisputeInitiatedPayload,
  type DisputeResolvedPayload,
  type RewardDistributedPayload,
  type VestingCreatedPayload,
  type ClawbackExecutedPayload,
  createEvent,
} from "./events.js";

// ============ Domain Contracts ============

// Knowledge Service
export {
  type KnowledgeService,
  type KnowledgeReader as KnowledgeServiceReader,
  type KnowledgeSubscriber,
  type KnowledgeEvent,
  type KnowledgeEventType,

  // Knowledge types
  type EpistemicCategory,
  type ClaimType,
  type FactMaturity,
  type FactStatus,
  type InferenceType,
  type ClaimSubmission,
  type Fact,
  type KnowledgeQuery,
  type KnowledgeResult,
  type Challenge,
} from "./knowledge.js";

// Consensus Engine
export {
  type ConsensusEngine,
  type ConsensusReader,
  type ConsensusEventType,

  // Consensus types
  type RoundPhase,
  type Verdict,
  type RoundOutcome,
  type ValidatorCommitment,
  type ValidatorReveal,
  type VerificationRound,
  type ValidatorSelection,
  type VerificationResult,
  type BlockHeader,
  type Block,
  type BlockAttestation,
  type Transaction,
  type ConsensusParams,
} from "./consensus.js";

// Rewards Engine
export {
  type RewardsEngine,
  type RewardsReader,
  type RewardsEventType,

  // Rewards types
  type VestingStatus,
  type VestingCategory,
  type VestingSchedule,
  type Clawback,
  type RewardDistribution,
  type RewardReason,
  type AccelerationEvent,
  type AccelerationType,
  type VestingParams,
} from "./rewards.js";

// Dispute Resolver
export {
  type DisputeResolver,
  type DisputeReader,
  type DisputeEventType,

  // Dispute types
  type DisputePhase,
  type DisputeOutcome,
  type EscalationTier,
  type DisputeTargetType,
  type ChallengeType,
  type ArbiterVote,
  type SelectedArbiter,
  type EvidenceCommitment,
  type EvidenceComparison,
  type DisputeState,
  type Dispute,
  type DisputeSettlement,
  type DisputeParams,
} from "./disputes.js";

// Identity Service
export {
  type IdentityService,
  type IdentityReader,
  type IdentityEventType,

  // Identity types
  type AgentAccount,
  type AccountFlags,
  type SessionKey,
  type SessionPermissions,
  type RecoveryShare,
  type RecoveryRequest,
  type KeyRotation,
  type IdentityVerification,
  type Attestation,
} from "./identity.js";
