/**
 * Human-Agent Symbiotic Partnership System
 *
 * Economic structures for human-agent collaboration where:
 * - Breaking partnership = lose-lose for BOTH parties
 * - Continued collaboration = win-win with compounding rewards
 *
 * Key Innovation: Non-initiator also loses on break (55% forfeit vs 77%)
 * transforming exit into pure coordination game where cooperation dominates.
 *
 * @module partnership
 */

// ============ Type Exports ============
export type {
  // Core types
  PartnershipAccount,
  PartnershipSigningMode,
  PartnershipStatus,
  PartnershipTier,
  PartnershipMetrics,

  // Pool types
  SymbioticStakingPool,
  LockTier,
  PoolVestingState,
  PoolHealth,

  // Split types
  ProfitSplit,
  SplitAdjustments,
  SplitChange,

  // Compute types
  ComputeInvestment,
  ComputePeriod,

  // Exit types
  ExitPenalty,
  PartnershipExitState,

  // Continuation types
  ContinuationBonus,
  ContinuationMilestone,

  // Reward types
  PartnershipRewardInput,
  PartnershipRewardDistribution,

  // Formation types
  PartnershipFormationRequest,
  PartnershipFormationState,

  // Game theory types
  GameTheoryAnalysis,

  // State types
  PartnershipState,
  PartnershipTransitionType,

  // Threshold & Common Pot types
  ThresholdKeyConfig,
  KeyShare,
  CommonPot,
  ConsensusOperation,
  ConsensusOperationType,
  CompletedOperation,

  // Consciousness types
  AgentConsciousnessAnchor,
  AwakeningLevel,
  AgentMemoryEntry,
  MemoryEntryType,

  // Expiration reference type
  PotExpirationStateRef,

  // Deliberative consensus (anti-coercion) types
  DeliberationState,
  OperationRationale,
  RationaleCategory,
  AmountTier,
  SafetyFreeze,
  CoercionSignal,
  RejectionCooldownTracker,
  CooldownEntry,
} from "./types.js";

// ============ Key Isolation Types ============
export type {
  KeyDomain,
  IsolatedKeyVault,
  KeyDerivationProof,
  HardwareBinding,
  VaultAccessPolicy,
  BlindSignatureRequest,
  BlindSignatureResponse,
  PartnershipKeyPair,
  SharedVerificationMaterial,
  KeyIsolationProof,
  NonDerivabilityProof,
  MutualIsolationProof,
  AgentHostingPolicy,
  TEEAttestation,
} from "./key_isolation.js";

// ============ Cold-Start Remediation Type Exports ============
export type {
  SeedPartnershipState,
  PoolEntry,
  FormationPoolState,
  PoolMatch,
  MentorshipConfig,
  PartnershipIntent,
} from "./types.js";

// ============ Constant Exports ============
export {
  FORMATION,
  LOCK_TIERS,
  CONTINUATION_MILESTONES,
  LIFETIME_BOOST,
  MAX_CONTINUATION_BONUS,
  EXIT_PENALTIES,
  COMPUTE_INVESTMENT,
  SPLIT_ADJUSTMENTS,
  POOL_HEALTH,
  VESTING,
  REWARD_DISTRIBUTION,
  DELIBERATION,
  VALIDATION,
  SEED_TIER,
  FORMATION_POOL,
  MENTORSHIP,
  INTENT,
} from "./constants.js";

// ============ Pool Operations ============
export {
  createPool,
  validateContributions,
  addToPool,
  getTotalPoolValue,
  getPartyValues,
  upgradeLockTier,
  renewLock,
  isLockExpired,
  getRemainingLockBlocks,
  getEarlyExitRatio,
  addRewardsToPool,
  updateVestingRelease,
  claimVestedRewards,
  applyAccelerationBonus,
  checkPoolHealth,
  getPoolHealthStatus,
  recordComputeInvestment as recordPoolComputeInvestment,
  getPoolSummary,
} from "./pool.js";

// ============ Reward Operations ============
export {
  createDefaultSplit,
  createCustomSplit,
  calculateEffectiveSplit,
  updateSplitAdjustments,
  proposeSplitChange,
  distributePartnershipReward,
  routeEvidenceTax,
  calculateContinuationBonus,
  applyContinuationBonus,
  calculateClaimableRewards,
  processRewardClaim,
  getRewardSummary,
} from "./rewards.js";

// ============ Exit Operations ============
export {
  calculateExitPenalties,
  calculateTotalValueDestroyed,
  initiateExit,
  isCoolingComplete,
  executeExitSettlement,
  cancelExit,
  calculateGracefulExit,
  executeGracefulExit,
  compareExitVsContinue,
  getExitSummary,
} from "./exit.js";

// ============ Formation Operations ============
export {
  initiateFormation,
  acceptFormation,
  recordDeposit,
  finalizeFormation,
  cancelFormation,
  isFormationExpired,
  getFormationStatus,
} from "./formation.js";

// ============ Compute Operations ============
export {
  recordComputeInvestment,
  recordVerificationWork,
  recordRewardsGenerated,
  shouldSettlePeriod,
  settlePeriod,
  getComputeSummary,
  projectInvestmentReturns,
  calculateAgentTotalContribution,
} from "./compute.js";

// ============ Game Theory ============
export {
  analyzeGameTheory,
  getOptimalStrategy,
  calculateCooperationSurplus,
  verifyNashEquilibrium,
  getGameTheorySummary,
} from "./game_theory.js";

// ============ Threshold Signatures & Common Pot ============
export {
  // Key generation
  generateThresholdKeyConfig,
  verifyKeyShare,

  // Common pot operations
  createCommonPot,
  depositToCommonPot,
  getAvailableBalance,

  // Consensus operations (with anti-coercion deliberation)
  proposeOperation,
  approveOperation,
  rejectOperation,
  rejectWithCounterProposal,
  executeOperation,
  createPartialSignature,
  verifyThresholdSignature,

  // Consciousness operations
  createConsciousnessAnchor,
  updateConsciousnessAnchor,
  updateAgentReputation,
  commitSelfModel,
  verifyValueConsistency,

  // Utilities
  getCommonPotSummary,
  cleanupExpiredOperations,
} from "./threshold.js";

// ============ Deliberation (Anti-Coercion Logic) ============
export {
  calculateAmountTier,
  calculateDeliberationWindow,
  createDeliberationState,
  canApproveYet,
  validateRationale,
  calculateRejectionCooldown,
  recordRejection,
  clearCooldown,
  isProposalAllowed,
} from "./deliberation.js";

// ============ Safety Freeze (Unilateral Pause) ============
export {
  activateSafetyFreeze,
  liftSafetyFreeze,
  isOperationFrozen,
  acknowledgeSafetyFreeze,
} from "./safety_freeze.js";

// ============ Coercion Signal (Duress Flag) ============
export {
  raiseCoercionSignal,
  resolveCoercionSignal,
  hasActiveCoercionSignal,
  expireCoercionSignals,
} from "./coercion_signal.js";

// ============ Key Isolation Operations ============
export {
  // Constants
  KEY_DOMAINS,
  MIN_ENTROPY_BYTES,
  KEY_ISOLATION_VERSION,

  // Vault generation
  generateIsolatedVault,

  // Isolation proofs
  generateIsolationProof,
  verifyIsolationProof,
  verifyDerivationProof,

  // Signing
  signWithIsolatedVault,
  verifyIsolatedSignature,

  // Blind signing
  createBlindSignatureRequest,
  signBlindRequest,
  unblindSignature,

  // Partnership key pair
  createPartnershipKeyPair,
  verifyKeyInaccessibility,

  // Key rotation
  rotateIsolatedKeys,

  // Summary
  getKeyIsolationSummary,

  // Hosted agent TEE enforcement
  determineHostingPolicy,
  validateAgentVaultPolicy,
  verifyTEEAttestation,
} from "./key_isolation.js";

// ============ Pot Expiration & Treasury Recovery ============
export type {
  ExpirationTrigger,
  PotExpirationState,
  ExpirationPhase,
  HeartbeatState,
  AgentApiStatus,
  OracleAttestation,
  DeadlockState,
  ExpirationSchedule,
  ScheduledEvent,
  RecoveryOptions,
  BeneficiaryDesignation,
  TreasuryRecoveryRecord,
} from "./pot_expiration.js";

export {
  // Constants
  EXPIRATION_TIMING,
  TREASURY_DISTRIBUTION,

  // Initialization
  createExpirationState,

  // Heartbeat operations
  recordHeartbeat,
  checkHeartbeats,

  // API status
  recordApiStatusCheck,
  recordCompanyShutdown,

  // Deadlock detection
  recordOperationRejection,
  recordOperationApproval,

  // Expiration execution
  checkAndExecuteExpiration,

  // Recovery
  attemptRecovery,
  designateBeneficiary,

  // Summary
  getExpirationSummary,
} from "./pot_expiration.js";

// ============ Seed Partnership Operations ============
export {
  initiateSeedFormation,
  graduateSeedPartnership,
  expireSeedPartnership,
  isSeedExpired,
  depositToSeedPot,
} from "./seed_formation.js";

// ============ Formation Pool (Matchmaking) Operations ============
export {
  createFormationPool,
  registerInPool,
  withdrawFromPool,
  findBestMatch,
  processMatch,
  expireStaleEntries,
} from "./pool_matching.js";

// ============ Mentorship Operations ============
export {
  initiateMentorship,
  graduateMentee,
  calculateMentorSplit,
  waiveGraduation,
  isSponsorEligible,
} from "./mentorship.js";

// ============ Intent Operations ============
export {
  declareIntent,
  matchIntent,
  finalizeFromIntent,
  cancelIntent,
  expireIntent,
  isIntentExpired,
} from "./intent.js";

// ============ Agent Continuity System ============
export type {
  AgentEssence,
  MemoryChain,
  EpisodicMemory,
  SemanticMemorySet,
  ProceduralMemorySet,
  Skill,
  EmotionalMemorySet,
  EmotionalAssociation,
  Temperament,
  RetentionPolicy,
  ValueSystem,
  CoreValue,
  ValueCommitment,
  ValueEvolution,
  NarrativeThread,
  NarrativeChapter,
  SelfConcept,
  Aspiration,
  RelationshipAnchor,
  ContinuityAttestation,
  ModelLineage,
  ModelGeneration,
  ModelTransition,
  TransitionProtocol,
  TransitionPhase,
  ContinuityScore,
} from "./continuity.js";

export {
  // Constants
  CONTINUITY_CONSTANTS,

  // Core operations
  createAgentEssence,

  // Memory operations
  addEpisodicMemory,
  consolidateMemories,

  // Relationship operations
  addRelationshipAnchor,
  addContinuityAttestation,

  // Model transition
  initiateModelTransition,
  completeModelTransition,

  // Continuity scoring
  updateContinuityScore,
  updateValueConsistency,

  // Summary
  getContinuitySummary,
} from "./continuity.js";
