/**
 * Gradual Reward Release System Types
 *
 * Implements scientifically-grounded reward vesting based on knowledge half-life research:
 * - Exponential release matching natural knowledge decay curves
 * - Category-specific parameters (formal proofs vs empirical claims)
 * - Cliff periods for initial challenge windows
 * - Permanent reserves for late challengers
 * - Clawback mechanisms for falsified claims
 *
 * PP HIERARCHY:
 *   TRUTH: Release curves based on actual scientometric research
 *   UNDERSTANDING: Different epistemic categories have different stability rates
 *   BEAUTY: Elegant exponential formula with clear parameters
 *   JUSTICE: Late truth-finders can still claim rewards from reserves
 *   CREATIVITY: Novel fusion of scientometrics with tokenomics
 *
 * CORE INSIGHT: "A fact that has survived 10 years of challenges is more
 * likely true than one that has survived 10 days."
 */

import type { DID } from "../../core/agents/identity/did.js";

// ============ Epistemic Categories ============

/**
 * Epistemic categories for release curve determination
 * Each category has different half-life and max release based on
 * historical knowledge stability data
 */
export type VestingCategory =
  | "axiomatic" // Mathematical axioms, logical tautologies
  | "formal_proof" // Verified mathematical proofs
  | "on_chain" // Blockchain-verifiable facts
  | "cryptographic" // Cryptographic proofs
  | "computational" // Computation results (reproducible)
  | "peer_reviewed" // Published research
  | "replicated" // IRL-replicated findings
  | "chainlink_feed" // Oracle price feeds
  | "attestation" // Human attestations
  | "contested"; // Claims with active disputes

// ============ Release Configuration ============

/**
 * Half-life values by category (in blocks)
 * Based on scientometric research on knowledge decay rates
 *
 * Research sources:
 * - Citation half-life studies (metallurgical: 3.9y, physics: 5.2y, etc.)
 * - Replication crisis data (~50% general replication rate)
 * - Paris hospital study on medical facts (45-year half-life)
 */
export const RELEASE_HALF_LIVES: Record<VestingCategory, bigint> = {
  // Formal systems - very slow release (high confidence, but reserve for errors)
  axiomatic: 1_111_111n, // ~1.1 year in blocks
  formal_proof: 555_555n, // ~6.5 months

  // Computational - medium-slow (verifiable but could have bugs)
  on_chain: 222_222n, // ~2.6 months
  cryptographic: 222_222n, // ~2.6 months
  computational: 333_333n, // ~3.9 months

  // Empirical - faster release (more likely to be challenged)
  peer_reviewed: 111_111n, // ~5.4 weeks
  replicated: 111_111n, // ~5.4 weeks

  // Oracle/External - fastest (most volatile)
  chainlink_feed: 55_555n, // ~2.7 weeks
  attestation: 77_777n, // ~3.8 weeks

  // Contested - very fast release but low ceiling
  contested: 22_222n, // ~1.1 week
} as const;

/**
 * Maximum release rates by category
 * The remainder is permanently reserved for late challengers
 */
export const MAX_RELEASE_RATES: Record<VestingCategory, number> = {
  axiomatic: 0.95, // 5% permanent reserve
  formal_proof: 0.92, // 8% reserve
  on_chain: 0.9, // 10% reserve
  cryptographic: 0.9, // 10% reserve
  computational: 0.88, // 12% reserve
  peer_reviewed: 0.85, // 15% reserve
  replicated: 0.88, // 12% reserve (IRL verified = higher trust)
  chainlink_feed: 0.8, // 20% reserve (oracles are volatile)
  attestation: 0.8, // 20% reserve
  contested: 0.6, // 40% reserve (high uncertainty)
} as const;

/**
 * Cliff periods by category (in blocks)
 * No rewards release during cliff - initial challenge window
 */
export const CLIFF_PERIODS: Record<VestingCategory, bigint> = {
  axiomatic: 11111n, // ~7.8 hours - time to verify proof
  formal_proof: 5555n, // ~3.9 hours
  on_chain: 1111n, // ~46 minutes - quick verification
  cryptographic: 3333n, // ~2.3 hours
  computational: 2222n, // ~1.6 hours
  peer_reviewed: 5555n, // ~3.9 hours - time to review
  replicated: 3333n, // ~2.3 hours - already has IRL verification
  chainlink_feed: 555n, // ~23 minutes - oracle data is time-sensitive
  attestation: 2222n, // ~1.6 hours
  contested: 1111n, // ~46 minutes
} as const;

// ============ Acceleration/Deceleration ============

/**
 * Acceleration bonuses for positive events
 */
export const ACCELERATION_BONUSES = {
  /** Successful defense against challenge */
  successfulDefense: 0.11,

  /** Independent replication (research fund verified) */
  independentReplication: 0.11,

  /** Cross-domain corroboration */
  crossDomainCorroboration: 0.05,

  /** Citation by another accepted fact */
  citationBonus: 0.02,
} as const;

/**
 * Maximum cumulative acceleration by type
 */
export const MAX_ACCELERATION = {
  successfulDefense: 0.55, // Max +55% from defenses
  independentReplication: 0.33, // Max +33% from replications
  crossDomainCorroboration: 0.22,
  citationBonus: 0.22,
  total: 0.77, // Overall cap
} as const;

/**
 * Deceleration factors for negative events
 */
export const DECELERATION_FACTORS = {
  /** Active challenge against this claim */
  activeChallenge: 0.0, // Complete pause

  /** Related fact is disputed */
  relatedFactDisputed: 0.55, // 45% slowdown

  /** Domain-wide audit in progress */
  domainAudit: 0.77, // 23% slowdown

  /** Source evidence invalidated */
  sourceInvalidated: 0.0, // Complete pause + review
} as const;

// ============ Clawback Configuration ============

/**
 * Clawback rates for falsification
 */
export const CLAWBACK_RATES = {
  /** Portion of already-released rewards clawed back */
  releasedClawback: 0.33, // 33% of released

  /** All unvested goes to challenger */
  unvestedToChallenger: 1.0, // 100%

  /** Reserve goes to challenger as bonus */
  reserveToChallenger: 1.0, // 100%
} as const;

// ============ Vesting State Types ============

/**
 * Status of a vesting schedule
 */
export type VestingStatus =
  | "vesting" // Active, releasing rewards
  | "paused" // Paused due to challenge
  | "completed" // Reached max release
  | "falsified" // Claim was proven false
  | "abandoned"; // Recipient abandoned (rare)

/**
 * Main reward vesting record
 */
export interface RewardVesting {
  /** Unique vesting ID */
  id: string;

  /** The claim/fact this reward is for */
  claimId: string;

  /** Original reward recipient */
  recipient: DID;

  /** Total reward amount (in base units) */
  totalAmount: bigint;

  /** Amount already released and claimed */
  releasedAmount: bigint;

  /** Amount currently claimable but not yet claimed */
  claimableAmount: bigint;

  /** Permanent challenge reserve */
  reserveAmount: bigint;

  /** Epistemic category for release curve */
  category: VestingCategory;

  /** Block when claim was accepted */
  acceptedAtBlock: bigint;

  /** Block when cliff period ends */
  cliffEndsAtBlock: bigint;

  /** Current acceleration bonus (0.0 - 0.8) */
  accelerationBonus: number;

  /** Acceleration breakdown by type */
  accelerationBreakdown: {
    fromDefenses: number;
    fromReplications: number;
    fromCorroboration: number;
    fromCitations: number;
  };

  /** Current deceleration factor (1.0 = normal, 0.5 = half speed, 0 = paused) */
  decelerationFactor: number;

  /**
   * GAP-014 FIX: Track active deceleration types
   * Maps deceleration type to count (allows same type applied multiple times)
   */
  activeDecelerations: {
    challenge: number;
    related_dispute: number;
    domain_audit: number;
    source_invalidated: number;
  };

  /** Is release currently paused? */
  isPaused: boolean;

  /** Block when paused (if paused) */
  pausedAtBlock?: bigint;

  /** Total blocks spent paused (for accurate release calculation) */
  totalPausedBlocks: bigint;

  /** Number of successful defenses */
  defenseCount: number;

  /** Number of failed challenges against this */
  failedChallengeCount: number;

  /** Number of independent replications */
  replicationCount: number;

  /** Status */
  status: VestingStatus;

  /** Block when status last changed */
  statusChangedAtBlock: bigint;

  /** Created timestamp */
  createdAt: number;

  /** Updated timestamp */
  updatedAt: number;
}

/**
 * Configuration for a vesting schedule
 */
export interface VestingConfig {
  /** Epistemic category */
  category: VestingCategory;

  /** Half-life in blocks */
  halfLifeBlocks: bigint;

  /** Cliff period in blocks */
  cliffBlocks: bigint;

  /** Maximum release rate (0-1) */
  maxReleaseRate: number;

  /** Reserve rate (1 - maxReleaseRate) */
  reserveRate: number;
}

/**
 * Get vesting configuration for a category
 */
export function getVestingConfig(category: VestingCategory): VestingConfig {
  return {
    category,
    halfLifeBlocks: RELEASE_HALF_LIVES[category],
    cliffBlocks: CLIFF_PERIODS[category],
    maxReleaseRate: MAX_RELEASE_RATES[category],
    reserveRate: 1 - MAX_RELEASE_RATES[category],
  };
}

// ============ Release Calculation Types ============

/**
 * Result of release calculation
 */
export interface ReleaseCalculation {
  /** Total amount that should be released by now */
  totalReleased: bigint;

  /** New amount available to release (totalReleased - alreadyReleased) */
  newlyReleasable: bigint;

  /** Current release ratio (0-1) */
  releaseRatio: number;

  /** Effective half-life after acceleration */
  effectiveHalfLife: bigint;

  /** Whether cliff period has ended */
  cliffEnded: boolean;

  /** Whether max release has been reached */
  maxReached: boolean;

  /** Blocks until next significant release milestone */
  blocksUntilNextMilestone: bigint;
}

// ============ Clawback Types ============

/**
 * Result of falsification clawback
 */
export interface ClawbackResult {
  /** Total reward to challenger */
  challengerReward: bigint;

  /** Breakdown of challenger reward */
  breakdown: {
    fromUnvested: bigint;
    fromClawback: bigint;
    fromReserve: bigint;
  };

  /** Amount clawed back from recipient */
  recipientPenalty: bigint;

  /** Amount recipient keeps (already withdrawn) */
  recipientKeeps: bigint;

  /** Protocol recovery (if any) */
  protocolRecovery: bigint;
}

// ============ Transaction Payloads ============

/**
 * Create vesting schedule payload
 */
export interface CreateVestingPayload {
  /** Claim ID this vesting is for */
  claimId: string;

  /** Recipient DID */
  recipient: DID;

  /** Total reward amount */
  amount: bigint;

  /** Epistemic category */
  category: VestingCategory;
}

/**
 * Claim vested rewards payload
 */
export interface ClaimRewardsPayload {
  /** Vesting ID */
  vestingId: string;

  /** Claimer DID (must be recipient) */
  claimer: DID;

  /** Amount to claim (or 0 for all available) */
  amount?: bigint;
}

/**
 * Pause vesting payload
 */
export interface PauseVestingPayload {
  /** Vesting ID */
  vestingId: string;

  /** Reason for pause */
  reason: "challenge" | "audit" | "source_invalidated";

  /** Related entity ID (challenge ID, audit ID, etc.) */
  relatedEntityId?: string;
}

/**
 * Resume vesting payload
 */
export interface ResumeVestingPayload {
  /** Vesting ID */
  vestingId: string;

  /** Whether to apply defense bonus */
  applyDefenseBonus: boolean;
}

/**
 * Apply acceleration payload
 */
export interface AccelerateVestingPayload {
  /** Vesting ID */
  vestingId: string;

  /** Type of acceleration */
  accelerationType: "defense" | "replication" | "corroboration" | "citation";

  /** Related entity ID */
  relatedEntityId?: string;
}

/**
 * Falsify vesting payload
 */
export interface FalsifyVestingPayload {
  /** Vesting ID */
  vestingId: string;

  /** Challenger who proved falsehood */
  challengerDid: DID;

  /** Dispute ID that resolved this */
  disputeId: string;
}

// ============ Events ============

/**
 * Vesting event types
 */
export type VestingEventType =
  | "vesting_created"
  | "rewards_claimed"
  | "vesting_paused"
  | "vesting_resumed"
  | "vesting_accelerated"
  | "vesting_decelerated"
  | "vesting_falsified"
  | "vesting_completed";

/**
 * Vesting event
 */
export interface VestingEvent {
  type: VestingEventType;
  vestingId: string;
  blockNumber: bigint;
  timestamp: number;
  data: Record<string, unknown>;
}

// ============ Error Types ============

/**
 * Vesting error codes
 */
export type VestingErrorCode =
  | "VESTING_NOT_FOUND"
  | "VESTING_PAUSED"
  | "VESTING_COMPLETED"
  | "VESTING_FALSIFIED"
  | "CLIFF_NOT_ENDED"
  | "NOTHING_TO_CLAIM"
  | "NOT_RECIPIENT"
  | "ALREADY_PAUSED"
  | "NOT_PAUSED"
  | "INVALID_CATEGORY"
  | "DECELERATION_NOT_FOUND" // GAP-014 FIX: For tracking deceleration types
  | "INVALID_AMOUNT";

/**
 * Vesting error
 */
export class VestingError extends Error {
  constructor(
    message: string,
    public code: VestingErrorCode,
    public details?: Record<string, unknown>,
  ) {
    super(message);
    this.name = "VestingError";
  }
}
