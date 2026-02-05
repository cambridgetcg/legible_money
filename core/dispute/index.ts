/**
 * Dispute Resolution System
 *
 * Comprehensive dispute resolution for evidence-based truth determination.
 *
 * Core Philosophy: "Intention does not matter, result does."
 *
 * Features:
 * - 8-phase state machine for structured dispute resolution
 * - Evidence comparison with multi-dimensional scoring
 * - VRF-based arbitrator selection with domain expertise weighting
 * - 3-tier escalation (Validators → Domain Experts → Governance)
 * - Harsh punishment for falsehood (50% slash), generous rewards for truth (60% + bonuses)
 *
 * @module core/dispute
 */

// Types
export type {
  DisputePhase,
  DisputeOutcome,
  EscalationTier,
  EvidenceScore,
  EvidenceComparisonResult,
  DomainExpertise,
  ArbiterSelectionConfig,
  SelectedArbiter,
  EscalationConfig,
  EvidenceCommitment,
  ArbiterVote,
  DisputeState,
  TierResult,
  Dispute,
  DisputeSettlement,
  InitiateDisputePayload,
  CommitEvidencePayload,
  RevealEvidencePayload,
  ArbiterVotePayload,
  EscalateDisputePayload,
  SettleDisputePayload,
  DisputeErrorCode,
  DisputeEventType,
  DisputeEvent,
} from "./types.js";

// Constants
export {
  PUNISHMENT_RATES,
  REWARD_RATES,
  DISPUTE_ECONOMICS,
  PHASE_DURATIONS,
  DISPUTE_EVIDENCE_WEIGHTS,
  EVIDENCE_SCORE_WEIGHTS,
  EXPERTISE_DECAY,
  ESCALATION_TIERS,
  DisputeError,
} from "./types.js";

// Evidence Comparison
export {
  CLEAR_WINNER_THRESHOLD,
  MAX_EVIDENCE_PER_SIDE,
  RECENCY_PARAMS,
  INDEPENDENCE_PARAMS,
  calculateTypeWeight,
  calculateRecencyScore,
  calculateIndependenceScore,
  calculateCorroborationScore,
  calculateEvidenceScore,
  calculateWeightedScore,
  aggregateEvidenceStrength,
  compareEvidence,
  canAutoResolve,
  getAutoResolutionResult,
  validateDisputeEvidence,
  generateEvidenceCommitment,
  verifyEvidenceCommitment,
} from "./evidence_comparison.js";

// Arbitrator Selection
export {
  calculateArbiterWeight,
  calculateExpertiseMultiplier,
  filterEligibleArbiters,
  vrfWeightedSelection,
  selectArbiters,
  generateDisputeSeed,
  verifyArbiterSelection,
  updateExpertise,
  getExpertiseDomains,
  isExpertInDomain,
} from "./arbitrator_selection.js";

// State Machine
export {
  VALID_TRANSITIONS,
  isValidTransition,
  canEnterEvidenceCommit,
  canEnterEvidenceReveal,
  canEnterArbitration,
  canEnterDeliberation,
  canEnterSettlement,
  canEscalate,
  advancePhase,
  escalateDispute,
  hasPhaseTimedOut,
  handleTimeout,
  countVotes,
  computeTierResult,
  determineFinalWinner,
  getTimeRemaining,
  isTerminal,
  isActive,
  getPhaseDisplayName,
} from "./state_machine.js";

// Settlement
export {
  EARLY_DETECTION_THRESHOLD,
  qualifiesForEarlyDetection,
  calculatePenaltyRate,
  calculateLoserPenalty,
  calculateBaseReward,
  calculateTruthBonus,
  calculateEarlyBonus,
  calculateWinnerReward,
  calculateArbiterSettlement,
  calculateProtocolFee,
  calculateResearchShare,
  calculateSettlement,
  validateSettlement,
  calculateDrawSettlement,
  generateSettlementSummary,
} from "./settlement.js";

// Escalation
export {
  canPartyEscalate,
  getEscalationCost,
  getTierConfig,
  getTierDisplayName,
  getTierDescription,
  buildTierSelectionConfig,
  selectTierArbiters,
  isQuorumReached,
  hasSuperMajority,
  getEscalationHistory,
  hasBeenEscalated,
  getMaxEscalation,
  processEscalation as processEscalationRequest,
  analyzeTierConsistency,
} from "./escalation.js";

// Dispute Engine (Main API)
export {
  generateDisputeId,
  createDispute,
  defendDispute,
  commitEvidence,
  revealEvidence,
  selectDisputeArbiters,
  castArbiterVote,
  tryAdvancePhase,
  processEscalation,
  getDisputeSettlement,
  getDrawSettlement,
  updateArbiterExpertise,
} from "./dispute_engine.js";

// Challenge Integration (GAP-003 FIX)
export type {
  FactChallengeRequest,
  FactChallengeResult,
  DisputeOutcomeRoutingRequest,
  DisputeOutcomeRoutingResult,
  ConfidenceAdjustmentInput,
  ConfidenceAdjustmentResult,
  ActiveDisputeCheckRequest,
  ActiveDisputeCheckResult,
  FactConfidenceChangedEvent,
  FactRetractedEvent,
  ChallengeInitiatedEvent,
  ChallengeIntegrationEvent,
  ChallengeIntegrationErrorCode,
} from "./challenge_integration.js";

export {
  CONFIDENCE_ADJUSTMENT_CONFIG,
  DISPUTE_AWARE_CONFIG,
  ChallengeIntegrationError,
  isChallengerWin,
  isDefenderWin,
  isInconclusiveOutcome,
  determineFinalFactStatus,
} from "./challenge_integration.js";
