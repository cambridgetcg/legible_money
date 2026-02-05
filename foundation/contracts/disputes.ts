/**
 * Dispute Resolver Contract
 *
 * Domain interface for arbitration and dispute resolution.
 * Handles evidence comparison, arbitrator selection, and settlement.
 *
 * MOD-1: Single Responsibility
 * Disputes module changes when arbitration protocol changes.
 *
 * Core Philosophy: "Intention does not matter, result does."
 * Spreading falsehoods pollutes the knowledge base - harsh punishment.
 */

import type { DID, PublicKey, Amount, BlockHeight } from "../types/index.js";

// ============ Dispute Types ============

/**
 * Dispute phases (8-phase state machine)
 */
export type DisputePhase =
  | "initiation" // Challenger submits stake + evidence
  | "evidence_commit" // Both parties commit evidence hashes
  | "evidence_reveal" // Reveal and compare evidence
  | "arbitration" // VRF-selected arbiters vote
  | "deliberation" // Arbiters reach consensus
  | "settlement" // Distribute rewards/punishments
  | "escalated" // Moved to higher tier
  | "resolved"; // Final state

/**
 * Dispute outcome
 */
export type DisputeOutcome =
  | "challenger_wins"
  | "defender_wins"
  | "draw"
  | "timeout";

/**
 * Escalation tier (1 = Validators, 2 = Domain Experts, 3 = Governance)
 */
export type EscalationTier = 1 | 2 | 3;

/**
 * Dispute target types
 */
export type DisputeTargetType = "evidence" | "fact" | "claim";

/**
 * Challenge type
 */
export type ChallengeType =
  | "factual_error"
  | "methodology_flaw"
  | "source_unreliable"
  | "outdated"
  | "misrepresentation"
  | "logical_error"
  | "incomplete";

/**
 * Arbiter vote
 */
export interface ArbiterVote {
  arbiter: DID;
  vote: "challenger" | "defender" | "abstain";
  justificationHash?: string;
  votedAtBlock: BlockHeight;
  lockedStake: Amount;
}

/**
 * Selected arbiter with selection info
 */
export interface SelectedArbiter {
  publicKey: PublicKey;
  did: DID;
  selectionWeight: number;
  domainExpertise: number;
  stakedAmount: Amount;
}

/**
 * Evidence commitment (before reveal)
 */
export interface EvidenceCommitment {
  committer: DID;
  evidenceHash: string;
  committedAtBlock: BlockHeight;
  revealed: boolean;
}

/**
 * Evidence comparison result
 */
export interface EvidenceComparison {
  challengerStrength: number;
  defenderStrength: number;
  strengthDifference: number;
  isClearWinner: boolean;
  suggestedWinner?: "challenger" | "defender";
}

/**
 * Dispute state
 */
export interface DisputeState {
  phase: DisputePhase;
  phaseDeadline: BlockHeight;
  canEscalate: boolean;
  escalationTier: EscalationTier;
}

/**
 * Dispute entity
 */
export interface Dispute {
  id: string;
  targetId: string;
  targetType: DisputeTargetType;
  domain: string;
  challengeType: ChallengeType;

  challenger: {
    did: DID;
    stake: Amount;
    initialEvidenceIds: string[];
  };

  defender: {
    did: DID;
    stake: Amount;
    responseEvidenceIds?: string[];
  };

  state: DisputeState;

  commitments: {
    challenger?: EvidenceCommitment;
    defender?: EvidenceCommitment;
  };

  evidenceComparison?: EvidenceComparison;
  arbiters: SelectedArbiter[];
  votes: Map<string, ArbiterVote>;

  outcome?: DisputeOutcome;
  winner?: "challenger" | "defender";
  exposedFalsehood: boolean;

  initiatedAtBlock: BlockHeight;
  resolvedAtBlock?: BlockHeight;
}

/**
 * Settlement calculation
 */
export interface DisputeSettlement {
  disputeId: string;
  winner: "challenger" | "defender";
  winnerReward: Amount;
  loserPenalty: Amount;
  truthBonusApplied: boolean;
  earlyDetectionBonusApplied: boolean;
  arbiterRewards: Map<string, Amount>;
  arbiterPenalties: Map<string, Amount>;
  protocolFee: Amount;
  researchFund: Amount;
}

/**
 * Dispute parameters
 */
export interface DisputeParams {
  minChallengeStake: Amount;
  minDefenseStake: Amount;
  minArbiterStake: Amount;
  tierMultipliers: Record<EscalationTier, number>;
  phaseDurations: Record<string, number>;
  punishmentRates: {
    falseClaimSlash: number;
    failedChallengeSlash: number;
    wrongArbitrationSlash: number;
  };
  rewardRates: {
    winnerShare: number;
    truthBonusMultiplier: number;
    arbiterCorrectBonus: number;
    earlyDetectionBonus: number;
  };
}

// ============ Dispute Resolver Interface ============

/**
 * Dispute resolver - main domain interface
 */
export interface DisputeResolver {
  // === Dispute Lifecycle ===

  /**
   * Initiate a dispute against evidence/fact/claim
   */
  initiateDispute(
    targetId: string,
    targetType: DisputeTargetType,
    challenger: DID,
    stake: Amount,
    initialEvidenceIds: string[],
    challengeType: ChallengeType,
    domain: string,
  ): Promise<Dispute>;

  /**
   * Get dispute by ID
   */
  getDispute(disputeId: string): Promise<Dispute | null>;

  /**
   * Get active disputes
   */
  getActiveDisputes(): Promise<Dispute[]>;

  /**
   * Get disputes for a target
   */
  getDisputesForTarget(targetId: string): Promise<Dispute[]>;

  // === Evidence Phase ===

  /**
   * Commit evidence hash
   */
  commitEvidence(
    disputeId: string,
    committer: DID,
    evidenceHash: string,
  ): Promise<void>;

  /**
   * Reveal evidence
   */
  revealEvidence(
    disputeId: string,
    revealer: DID,
    evidenceIds: string[],
    nonce: string,
  ): Promise<void>;

  /**
   * Compare evidence strengths
   */
  compareEvidence(disputeId: string): Promise<EvidenceComparison>;

  // === Arbitration Phase ===

  /**
   * Select arbiters using VRF
   */
  selectArbiters(
    disputeId: string,
    seed: Uint8Array,
  ): Promise<SelectedArbiter[]>;

  /**
   * Cast arbiter vote
   */
  castVote(
    disputeId: string,
    arbiter: DID,
    vote: "challenger" | "defender" | "abstain",
    justificationHash?: string,
  ): Promise<void>;

  /**
   * Get vote tally
   */
  getVoteTally(disputeId: string): Promise<{
    challengerVotes: number;
    defenderVotes: number;
    abstainVotes: number;
    quorumReached: boolean;
  }>;

  // === Phase Management ===

  /**
   * Advance dispute to next phase
   */
  advancePhase(disputeId: string): Promise<DisputePhase>;

  /**
   * Check if phase deadline passed
   */
  isDeadlinePassed(disputeId: string): Promise<boolean>;

  /**
   * Handle timeout (no response)
   */
  handleTimeout(disputeId: string): Promise<DisputeOutcome>;

  // === Escalation ===

  /**
   * Check if dispute can be escalated
   */
  canEscalate(disputeId: string, escalator: DID): Promise<boolean>;

  /**
   * Escalate dispute to next tier
   */
  escalate(
    disputeId: string,
    escalator: DID,
    additionalStake: Amount,
  ): Promise<EscalationTier>;

  // === Settlement ===

  /**
   * Calculate settlement
   */
  calculateSettlement(disputeId: string): Promise<DisputeSettlement>;

  /**
   * Execute settlement (distribute rewards/penalties)
   */
  executeSettlement(disputeId: string): Promise<void>;

  /**
   * Resolve dispute (final state)
   */
  resolveDispute(disputeId: string): Promise<Dispute>;

  // === Domain Expertise ===

  /**
   * Get arbiter's domain expertise score
   */
  getDomainExpertise(arbiter: DID, domain: string): Promise<number>;

  /**
   * Update domain expertise (after vote result)
   */
  updateDomainExpertise(
    arbiter: DID,
    domain: string,
    correct: boolean,
  ): Promise<void>;

  // === Parameters ===

  /**
   * Get dispute parameters
   */
  getParams(): DisputeParams;

  /**
   * Update dispute parameters (via governance)
   */
  updateParams(updates: Partial<DisputeParams>): Promise<void>;
}

// ============ Dispute Reader (Read-Only) ============

/**
 * Read-only dispute interface
 */
export interface DisputeReader {
  getDispute(disputeId: string): Promise<Dispute | null>;
  getActiveDisputes(): Promise<Dispute[]>;
  getDisputesForTarget(targetId: string): Promise<Dispute[]>;
  getDomainExpertise(arbiter: DID, domain: string): Promise<number>;
  getParams(): DisputeParams;
}

// ============ Dispute Events ============

/**
 * Events emitted by dispute resolver
 */
export type DisputeEventType =
  | "dispute_initiated"
  | "evidence_committed"
  | "evidence_revealed"
  | "arbiters_selected"
  | "vote_cast"
  | "phase_advanced"
  | "dispute_escalated"
  | "dispute_settled"
  | "dispute_resolved";
