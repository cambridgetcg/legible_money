/**
 * Challenge/Dispute Integration Types
 *
 * Types and interfaces for integrating the dispute resolution system
 * with the knowledge layer (facts, claims, verification rounds).
 *
 * GAP-003 FIX: This module provides the missing link between:
 * - Dispute outcomes → Fact confidence adjustments
 * - Verification rounds → Active dispute awareness
 * - Challenge initiation → Proper entry points
 *
 * PP HIERARCHY:
 *   TRUTH: Disputes reveal when claimed truths need re-evaluation
 *   UNDERSTANDING: Clear types for dispute-fact relationship
 *   BEAUTY: Clean integration without coupling systems tightly
 *   JUSTICE: Dispute outcomes justly affect fact confidence
 *   CREATIVITY: Bridge pattern connects isolated systems
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { Confidence } from "../knowledge/types.js";
import type { OntologicalStratum } from "../knowledge/ontology/types.js";
import type {
  DisputeOutcome,
  EscalationTier,
  Dispute,
  DisputeSettlement,
} from "./types.js";

// ============ Challenge Request Types ============

/**
 * Request to challenge an existing verified fact
 *
 * This is the entry point for disputing knowledge in the tree.
 */
export interface FactChallengeRequest {
  /** ID of the fact being challenged */
  factId: string;

  /** Challenger's DID */
  challenger: DID;

  /** Stake amount for the challenge */
  stake: bigint;

  /** Initial evidence IDs supporting the challenge */
  initialEvidence: string[];

  /** Reason hash (full reason stored off-chain) */
  reasonHash: string;

  /** Domain of the fact being challenged */
  domain: string;
}

/**
 * Result of initiating a fact challenge
 */
export interface FactChallengeResult {
  /** Whether challenge was accepted */
  accepted: boolean;

  /** Dispute ID if accepted */
  disputeId?: string;

  /** Reason for rejection if not accepted */
  rejectionReason?: string;

  /** Whether the fact was marked as challenged */
  factMarkedChallenged?: boolean;

  /** Original fact confidence (before challenge) */
  originalConfidence?: Confidence;
}

// ============ Dispute Outcome Routing ============

/**
 * Request to route a dispute outcome to the fact layer
 *
 * Called when a dispute reaches settlement phase.
 */
export interface DisputeOutcomeRoutingRequest {
  /** The resolved dispute */
  dispute: Dispute;

  /** Settlement details */
  settlement: DisputeSettlement;

  /** Block number when settled */
  settlementBlock: bigint;
}

/**
 * Result of routing dispute outcome to fact
 */
export interface DisputeOutcomeRoutingResult {
  /** Whether routing was successful */
  success: boolean;

  /** New confidence value for the fact */
  newConfidence?: Confidence;

  /** Confidence adjustment applied */
  confidenceAdjustment?: number;

  /** New fact status (matches FactStatus type) */
  newStatus?: "active" | "challenged" | "superseded" | "revoked";

  /** Whether fact was revoked (proven false) */
  factRetracted?: boolean;

  /** Error message if failed */
  error?: string;
}

// ============ Confidence Adjustment ============

/**
 * Configuration for confidence adjustment based on dispute outcome
 */
export const CONFIDENCE_ADJUSTMENT_CONFIG = {
  /** Confidence reduction when challenger wins (falsehood exposed) */
  challengerWinsReduction: 0.55,

  /** Confidence boost when defender wins (truth defended) */
  defenderWinsBoost: 0.11,

  /** Maximum confidence boost from successful defense */
  maxDefenseBoost: 0.22,

  /** Confidence reduction for draw (uncertainty persists) */
  drawReduction: 0.22,

  /** Confidence reduction for timeout (unresolved) */
  timeoutReduction: 0.11,

  /** Minimum confidence before retraction */
  retractionThreshold: 0.33,

  /** Additional reduction per escalation tier (truth required more scrutiny) */
  escalationTierPenalty: 0.05,

  /** Decay rate for challenge-related confidence reduction */
  challengeDecayRate: 0.9,
} as const;

/**
 * Confidence adjustment calculation input
 */
export interface ConfidenceAdjustmentInput {
  /** Current fact confidence */
  currentConfidence: Confidence;

  /** Dispute outcome */
  outcome: DisputeOutcome;

  /** Final escalation tier reached */
  finalTier: EscalationTier;

  /** Whether falsehood was exposed */
  exposedFalsehood: boolean;

  /** Number of previous challenges to this fact */
  previousChallengeCount: number;

  /** Ontological stratum of the fact */
  stratum?: OntologicalStratum;
}

/**
 * Confidence adjustment calculation result
 */
export interface ConfidenceAdjustmentResult {
  /** New confidence value */
  newConfidence: Confidence;

  /** Adjustment factor applied */
  adjustmentFactor: number;

  /** Whether fact should be retracted */
  shouldRetract: boolean;

  /** Reason for adjustment */
  reason: string;
}

// ============ Verification Round Integration ============

/**
 * Active dispute check request
 *
 * Used by verification rounds to check if a claim/fact has active disputes.
 */
export interface ActiveDisputeCheckRequest {
  /** ID of the fact or claim to check */
  targetId: string;

  /** Type of target */
  targetType: "fact" | "claim";
}

/**
 * Active dispute check result
 */
export interface ActiveDisputeCheckResult {
  /** Whether there are active disputes */
  hasActiveDisputes: boolean;

  /** Number of active disputes */
  activeDisputeCount: number;

  /** IDs of active disputes */
  activeDisputeIds: string[];

  /** Highest escalation tier among active disputes */
  highestTier: EscalationTier;

  /** Whether verification should be paused */
  shouldPauseVerification: boolean;

  /** Recommended action */
  recommendedAction: "proceed" | "pause" | "wait_for_resolution";
}

/**
 * Configuration for dispute-aware verification
 */
export const DISPUTE_AWARE_CONFIG = {
  /** Pause verification when dispute reaches this tier */
  pauseVerificationAtTier: 2 as EscalationTier,

  /** Maximum active disputes before pausing all verification */
  maxActiveDisputesBeforePause: 3,

  /** Blocks to wait after dispute resolution before resuming */
  postResolutionCooldown: 77,

  /** Whether to apply confidence penalty for ongoing disputes */
  penalizeOngoingDisputes: true,

  /** Confidence penalty per active dispute */
  ongoingDisputePenalty: 0.05,
} as const;

// ============ State Change Events ============

/**
 * Event emitted when a dispute affects fact confidence
 */
export interface FactConfidenceChangedEvent {
  type: "fact_confidence_changed";
  factId: string;
  previousConfidence: Confidence;
  newConfidence: Confidence;
  disputeId: string;
  outcome: DisputeOutcome;
  blockNumber: bigint;
  timestamp: number;
}

/**
 * Event emitted when a fact is retracted due to dispute
 */
export interface FactRetractedEvent {
  type: "fact_retracted";
  factId: string;
  disputeId: string;
  finalConfidence: Confidence;
  retractionReason: string;
  blockNumber: bigint;
  timestamp: number;
}

/**
 * Event emitted when a challenge is initiated
 */
export interface ChallengeInitiatedEvent {
  type: "challenge_initiated";
  factId: string;
  disputeId: string;
  challenger: DID;
  stake: bigint;
  blockNumber: bigint;
  timestamp: number;
}

/**
 * Union of all challenge integration events
 */
export type ChallengeIntegrationEvent =
  | FactConfidenceChangedEvent
  | FactRetractedEvent
  | ChallengeInitiatedEvent;

// ============ Error Types ============

/**
 * Challenge integration error codes
 */
export type ChallengeIntegrationErrorCode =
  | "FACT_NOT_FOUND"
  | "FACT_NOT_ACTIVE"
  | "ALREADY_CHALLENGED"
  | "INSUFFICIENT_STAKE"
  | "INVALID_EVIDENCE"
  | "DISPUTE_NOT_FOUND"
  | "INVALID_DISPUTE_STATE"
  | "ROUTING_FAILED"
  | "CONFIDENCE_CALCULATION_FAILED";

/**
 * Challenge integration error
 */
export class ChallengeIntegrationError extends Error {
  constructor(
    message: string,
    public code: ChallengeIntegrationErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "ChallengeIntegrationError";
  }
}

// ============ Helper Type Guards ============

/**
 * Check if a dispute outcome favors the challenger
 */
export function isChallengerWin(outcome: DisputeOutcome): boolean {
  return outcome === "challenger_wins";
}

/**
 * Check if a dispute outcome favors the defender
 */
export function isDefenderWin(outcome: DisputeOutcome): boolean {
  return outcome === "defender_wins";
}

/**
 * Check if a dispute outcome is inconclusive
 */
export function isInconclusiveOutcome(outcome: DisputeOutcome): boolean {
  return outcome === "draw" || outcome === "timeout";
}

/**
 * Determine fact status based on dispute outcome and confidence
 * Maps to valid FactStatus values: 'active' | 'challenged' | 'superseded' | 'revoked'
 */
export function determineFinalFactStatus(
  outcome: DisputeOutcome,
  newConfidence: Confidence,
  retractionThreshold: number = CONFIDENCE_ADJUSTMENT_CONFIG.retractionThreshold,
): "active" | "revoked" | "challenged" {
  if (isChallengerWin(outcome) && newConfidence <= retractionThreshold) {
    return "revoked";
  }
  if (isInconclusiveOutcome(outcome)) {
    return "challenged"; // Remains contested
  }
  return "active";
}
