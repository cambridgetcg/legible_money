/**
 * Dispute Resolution System Types
 *
 * Comprehensive types for evidence-based dispute resolution:
 * - 8-phase state machine (initiation → resolved)
 * - Evidence comparison scoring
 * - VRF-based arbitrator selection with domain expertise
 * - 3-tier escalation (Validators → Domain Experts → Governance)
 * - Harsh punishment / generous reward economics
 *
 * PP HIERARCHY:
 *   TRUTH: Disputes are the crucible where truth is tested
 *   UNDERSTANDING: Evidence comparison - type, reliability, recency, independence
 *   BEAUTY: Elegant 8-phase state machine with clear transitions
 *   JUSTICE: Results matter, not intentions - harsh for lies, generous for truth
 *   CREATIVITY: VRF selection with domain weighting, 3-tier escalation
 *
 * CORE PHILOSOPHY: "Intention does not matter, result does."
 * Spreading falsehoods - whether through malice or negligence - pollutes the knowledge base.
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { PublicKey } from "../crypto/keys.js";
import type { Evidence, EvidenceType } from "../evidence/types.js";

// ============ Economic Constants ============

/**
 * Punishment rates - HARSH because results matter
 */
export const PUNISHMENT_RATES = {
  /** False claim submitted - polluted knowledge base */
  falseClaimSlash: 0.55,

  /** Failed challenge - wasted everyone's time */
  failedChallengeSlash: 0.33,

  /** Wrong arbitration vote - failed duty */
  wrongArbitrationSlash: 0.11,

  /** Refused to reveal evidence - bad faith actor */
  refusalToRevealSlash: 1.0,

  /** Tampered with evidence - maximum penalty */
  evidenceTamperingSlash: 1.0,
} as const;

/**
 * Reward rates - GENEROUS for truth discovery
 */
export const REWARD_RATES = {
  /** Winner gets 66% of loser's stake */
  winnerShare: 0.66,

  /** +55% bonus for exposing falsehood */
  truthBonusMultiplier: 1.55,

  /** 11% bonus for correct arbitration vote */
  arbiterCorrectBonus: 0.11,

  /** +22% for catching falsehood early (within 1111 blocks) */
  earlyDetectionBonus: 0.22,
} as const;

/**
 * Dispute staking economics
 */
export const DISPUTE_ECONOMICS = {
  /** Minimum stake to initiate challenge */
  minChallengeStake: 111n * 10n ** 18n, // 111 LGM

  /** Defender must match challenger stake */
  minDefenseStake: 111n * 10n ** 18n, // 111 LGM

  /** Minimum stake to be eligible as arbiter */
  minArbiterStake: 77n * 10n ** 18n, // 77 LGM

  /** Stake multipliers by escalation tier */
  tierMultipliers: {
    1: 1.1, // Validators - standard
    2: 2.2, // Domain experts - 2.2x required
    3: 5.5, // Governance - 5.5x required
  } as const,

  /** Research fund share of dispute penalties */
  researchShare: 0.22,

  /** Protocol fee from dispute penalties */
  protocolFee: 0.05,
} as const;

/**
 * Phase durations in blocks (~2.5 sec per block)
 */
export const PHASE_DURATIONS = {
  initiation: 111, // ~4.6 min
  evidence_commit: 555, // ~23 min
  evidence_reveal: 333, // ~14 min
  arbitration: 1111, // ~46 min
  deliberation: 555, // ~23 min
  settlement: 77, // ~3.2 min
} as const;

// ============ Dispute Phases ============

/**
 * 8-phase dispute state machine
 */
export type DisputePhase =
  | "initiation" // Challenger submits stake + initial evidence
  | "evidence_commit" // Both parties commit evidence hashes
  | "evidence_reveal" // Reveal and compare evidence
  | "arbitration" // VRF-selected arbiters vote
  | "deliberation" // Arbiters discuss, reach consensus
  | "settlement" // Distribute rewards/punishments
  | "escalated" // Moved to higher tier
  | "resolved"; // Final state

/**
 * Dispute resolution outcome
 */
export type DisputeOutcome =
  | "challenger_wins"
  | "defender_wins"
  | "draw"
  | "timeout";

/**
 * Escalation tier levels
 */
export type EscalationTier = 1 | 2 | 3;

/**
 * Types of targets that can be disputed
 */
export type DisputeTargetType =
  | "evidence"
  | "fact"
  | "claim"
  // Governance dispute types
  | "lip_proposal" // Dispute a LIP's validity
  | "lip_review" // Dispute a review's accuracy
  | "governance_vote" // Dispute vote validity
  | "operator_disclosure" // Dispute undisclosed agents
  | "code_change" // Dispute a code contribution
  | "deployment_decision"; // Dispute deployment approval

// ============ Evidence Scoring ============

/**
 * Evidence type weights for dispute comparison
 * Different from general evidence strength - optimized for disputes
 */
export const DISPUTE_EVIDENCE_WEIGHTS: Record<EvidenceType, number> = {
  on_chain: 0.95,
  formal_proof: 0.93,
  cryptographic_proof: 0.92,
  chainlink_price_feed: 0.9,
  chainlink_por: 0.89,
  chainlink_ccip: 0.88,
  computational: 0.87,
  chainlink_functions: 0.85,
  peer_reviewed: 0.85,
  institutional: 0.75,
  oracle: 0.7,
  archived_web: 0.7,
  attestation: 0.65,
  inference: 0.5,
};

/**
 * Evidence comparison score components
 */
export interface EvidenceScore {
  /** Weight based on evidence type (0.5-0.95) */
  typeWeight: number;

  /** Reliability score from evidence itself (0-1) */
  reliabilityScore: number;

  /** Recency score - fresher is higher (0-1) */
  recencyScore: number;

  /** Independence score - more independent sources (0-1) */
  independenceScore: number;

  /** Corroboration with other verified facts (0-1) */
  corroborationScore: number;
}

/**
 * Weights for combining evidence score components
 */
export const EVIDENCE_SCORE_WEIGHTS = {
  type: 0.25,
  reliability: 0.3,
  recency: 0.15,
  independence: 0.2,
  corroboration: 0.1,
} as const;

/**
 * Evidence comparison result
 */
export interface EvidenceComparisonResult {
  /** Challenger's total evidence strength (0-1) */
  challengerStrength: number;

  /** Defender's total evidence strength (0-1) */
  defenderStrength: number;

  /** Difference (positive = challenger stronger) */
  strengthDifference: number;

  /** Is the difference clear enough for auto-resolution? */
  isClearWinner: boolean;

  /** Suggested winner if clear */
  suggestedWinner?: "challenger" | "defender";

  /** Individual evidence scores */
  challengerScores: EvidenceScore[];
  defenderScores: EvidenceScore[];
}

// ============ Arbitrator Selection ============

/**
 * Domain expertise tracking for arbitrators
 */
export interface DomainExpertise {
  /** Knowledge domain */
  domain: string;

  /** Number of correct votes in this domain */
  correctVotes: number;

  /** Total votes in this domain */
  totalVotes: number;

  /** Accuracy rate (correctVotes / totalVotes) */
  accuracy: number;

  /** Last block where they arbitrated this domain */
  lastActiveBlock: bigint;
}

/**
 * Expertise decay configuration
 */
export const EXPERTISE_DECAY = {
  /** Half-life in blocks (~3.2 days) */
  halfLifeBlocks: 111111n,

  /** Minimum votes to be considered expert */
  minVotesForExpertise: 7,

  /** Minimum accuracy to be considered expert */
  minAccuracyForExpertise: 0.66,
} as const;

/**
 * Arbiter selection configuration
 */
export interface ArbiterSelectionConfig {
  /** Number of arbiters to select */
  targetCount: number;

  /** VRF seed (from block hash + dispute ID) */
  vrfSeed: Uint8Array;

  /** Whether to weight by domain expertise */
  domainWeighting: boolean;

  /** Whether to weight by stake */
  stakeWeighting: boolean;

  /** Minimum reputation to be eligible */
  minReputation: number;

  /** Domain for expertise weighting */
  domain?: string;

  /** Exclude these validators (conflict of interest) */
  excludeValidators: Set<string>;
}

/**
 * Selected arbiter with weight info
 */
export interface SelectedArbiter {
  /** Validator public key */
  publicKey: PublicKey;

  /** Validator DID */
  did: DID;

  /** Selection weight used */
  selectionWeight: number;

  /** Domain expertise score (0-1) */
  domainExpertise: number;

  /** Stake at selection time */
  stakedAmount: bigint;
}

// ============ Escalation ============

/**
 * Escalation tier configuration
 */
export interface EscalationConfig {
  /** Tier level */
  tier: EscalationTier;

  /** Number of arbiters for this tier */
  arbiterCount: number;

  /** Quorum requirement (0.5 = majority, 0.67 = 2/3, 0.75 = 3/4) */
  quorumRequirement: number;

  /** Stake multiplier for this tier */
  stakeMultiplier: number;

  /** Blocks allowed for appeal after resolution */
  appealWindow: number;
}

/**
 * Escalation tier configurations
 */
export const ESCALATION_TIERS: Record<EscalationTier, EscalationConfig> = {
  1: {
    tier: 1,
    arbiterCount: 7,
    quorumRequirement: 0.55, // Simple majority
    stakeMultiplier: 1.1,
    appealWindow: 555, // ~23 min
  },
  2: {
    tier: 2,
    arbiterCount: 11,
    quorumRequirement: 0.66, // 2/3 supermajority
    stakeMultiplier: 2.2,
    appealWindow: 1111, // ~46 min
  },
  3: {
    tier: 3,
    arbiterCount: 22,
    quorumRequirement: 0.77, // 3/4 supermajority
    stakeMultiplier: 5.5,
    appealWindow: 0, // No appeal - final
  },
};

// ============ Core Dispute Types ============

/**
 * Evidence commitment (hash before reveal)
 */
export interface EvidenceCommitment {
  /** Committer's DID */
  committer: DID;

  /** Hash of evidence to be revealed */
  evidenceHash: string;

  /** Block when committed */
  committedAtBlock: bigint;

  /** Whether revealed yet */
  revealed: boolean;

  /** The actual evidence (set after reveal) */
  evidence?: Evidence[];
}

/**
 * Arbiter vote in a dispute
 */
export interface ArbiterVote {
  /** Arbiter's DID */
  arbiter: DID;

  /** Vote: who wins */
  vote: "challenger" | "defender" | "abstain";

  /** Optional justification hash (stored off-chain) */
  justificationHash?: string;

  /** Block when voted */
  votedAtBlock: bigint;

  /** Stake locked for this vote */
  lockedStake: bigint;
}

/**
 * Dispute state for tracking current phase
 */
export interface DisputeState {
  /** Current phase */
  phase: DisputePhase;

  /** Block number when phase ends */
  phaseDeadline: bigint;

  /** Whether escalation is available */
  canEscalate: boolean;

  /** Current escalation tier */
  escalationTier: EscalationTier;

  /** Previous tier results (if escalated) */
  previousTierResults?: TierResult[];
}

/**
 * Result of a single tier's arbitration
 */
export interface TierResult {
  /** Tier that rendered this result */
  tier: EscalationTier;

  /** Winner according to this tier */
  winner: "challenger" | "defender" | "draw";

  /** Vote counts */
  challengerVotes: number;
  defenderVotes: number;
  abstainVotes: number;

  /** Block when resolved */
  resolvedAtBlock: bigint;

  /** Whether it was appealed */
  wasAppealed: boolean;
}

/**
 * Main Dispute entity
 */
export interface Dispute {
  /** Unique dispute ID */
  id: string;

  /** Evidence or fact being disputed */
  targetId: string;

  /** Type of target being disputed */
  targetType: DisputeTargetType;

  /** Knowledge domain */
  domain: string;

  /** Challenger information */
  challenger: {
    did: DID;
    stake: bigint;
    initialEvidence: string[]; // Evidence IDs
  };

  /** Defender information */
  defender: {
    did: DID;
    stake: bigint;
    responseEvidence?: string[]; // Evidence IDs
  };

  /** Current state */
  state: DisputeState;

  /** Evidence commitments */
  commitments: {
    challenger?: EvidenceCommitment;
    defender?: EvidenceCommitment;
  };

  /** Evidence comparison result (after reveal) */
  evidenceComparison?: EvidenceComparisonResult;

  /** Selected arbiters for current tier */
  arbiters: SelectedArbiter[];

  /** Arbiter votes */
  votes: Map<string, ArbiterVote>; // DID -> Vote

  /** Final outcome */
  outcome?: DisputeOutcome;

  /** Final winner */
  winner?: "challenger" | "defender";

  /** Whether falsehood was exposed (for bonus) */
  exposedFalsehood: boolean;

  /** Block when initiated */
  initiatedAtBlock: bigint;

  /** Block when resolved (if resolved) */
  resolvedAtBlock?: bigint;

  /** Created timestamp */
  createdAt: number;

  /** Updated timestamp */
  updatedAt: number;
}

// ============ Settlement Types ============

/**
 * Settlement calculation result
 */
export interface DisputeSettlement {
  /** Dispute ID */
  disputeId: string;

  /** Winner */
  winner: "challenger" | "defender";

  /** Winner's total reward */
  winnerReward: bigint;

  /** Loser's total penalty */
  loserPenalty: bigint;

  /** Whether truth bonus was applied */
  truthBonusApplied: boolean;

  /** Whether early detection bonus was applied */
  earlyDetectionBonusApplied: boolean;

  /** Arbiter rewards (DID -> amount) */
  arbiterRewards: Map<string, bigint>;

  /** Arbiter penalties (DID -> amount) */
  arbiterPenalties: Map<string, bigint>;

  /** Protocol fee */
  protocolFee: bigint;

  /** Research fund share */
  researchFund: bigint;

  /** Breakdown of calculations */
  breakdown: {
    loserStake: bigint;
    baseWinnerReward: bigint;
    truthBonus: bigint;
    earlyBonus: bigint;
    penaltyRate: number;
  };
}

// ============ Transaction Payloads ============

/**
 * Initiate dispute transaction payload
 */
export interface InitiateDisputePayload {
  /** Target being disputed */
  targetId: string;
  targetType: DisputeTargetType;

  /** Domain */
  domain: string;

  /** Challenger DID */
  challenger: DID;

  /** Stake amount */
  stake: bigint;

  /** Initial evidence IDs */
  initialEvidence: string[];

  /** Reason for dispute (hash, content stored off-chain) */
  reasonHash: string;
}

/**
 * Commit evidence transaction payload
 */
export interface CommitEvidencePayload {
  /** Dispute ID */
  disputeId: string;

  /** Committer DID */
  committer: DID;

  /** Hash of evidence to reveal */
  evidenceHash: string;
}

/**
 * Reveal evidence transaction payload
 */
export interface RevealEvidencePayload {
  /** Dispute ID */
  disputeId: string;

  /** Revealer DID */
  revealer: DID;

  /** Actual evidence IDs */
  evidenceIds: string[];

  /** Nonce used in commitment */
  nonce: string;
}

/**
 * Arbiter vote transaction payload
 */
export interface ArbiterVotePayload {
  /** Dispute ID */
  disputeId: string;

  /** Arbiter DID */
  arbiter: DID;

  /** Vote */
  vote: "challenger" | "defender" | "abstain";

  /** Optional justification hash */
  justificationHash?: string;
}

/**
 * Escalate dispute transaction payload
 */
export interface EscalateDisputePayload {
  /** Dispute ID */
  disputeId: string;

  /** Party escalating (must be loser of previous tier) */
  escalator: DID;

  /** Additional stake for escalation */
  additionalStake: bigint;
}

/**
 * Settle dispute transaction payload
 */
export interface SettleDisputePayload {
  /** Dispute ID */
  disputeId: string;

  /** Settlement details */
  settlement: DisputeSettlement;
}

// ============ Error Types ============

/**
 * Dispute-specific error codes
 */
export type DisputeErrorCode =
  | "DISPUTE_NOT_FOUND"
  | "INVALID_PHASE"
  | "INSUFFICIENT_STAKE"
  | "NOT_AUTHORIZED"
  | "ALREADY_COMMITTED"
  | "COMMITMENT_MISMATCH"
  | "ALREADY_VOTED"
  | "NOT_ARBITER"
  | "CANNOT_ESCALATE"
  | "DEADLINE_PASSED"
  | "DEADLINE_NOT_REACHED"
  | "ALREADY_RESOLVED"
  | "INVALID_EVIDENCE";

/**
 * Dispute error
 */
export class DisputeError extends Error {
  constructor(
    message: string,
    public code: DisputeErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "DisputeError";
  }
}

// ============ Events ============

/**
 * Dispute event types for logging/hooks
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

/**
 * Dispute event
 */
export interface DisputeEvent {
  type: DisputeEventType;
  disputeId: string;
  blockNumber: bigint;
  timestamp: number;
  data: Record<string, unknown>;
}
