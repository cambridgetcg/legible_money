/**
 * Challenge Bounties System - Types
 *
 * Creates economic incentives to monitor domains and expose capture
 * attempts through bounties that reward successful challengers.
 *
 * Symbolic Constants:
 * - 77 LGM minimum challenge stake (lucky 7s)
 * - 7.7x successful challenge multiplier (lucky 7s)
 * - 777 LGM capture exposure bonus (ultimate luck)
 * - 777 blocks challenge period (~32 minutes)
 * - 7777 blocks review period (~5.4 hours)
 * - 2222 LGM base bounty pool (growth pattern)
 */

import type { OntologicalStratum } from "../../core/knowledge/ontology/types.js";

// ============================================================================
// Configuration
// ============================================================================

/**
 * Challenge bounty configuration with symbolic constants
 */
export const CHALLENGE_BOUNTY_CONFIG = {
  // Stakes
  minChallengeStake: 77n * 10n ** 18n, // 77 LGM
  challengeStakeMultiplierOnSuccess: 7.7, // 7.7x return
  captureExposureBonus: 777n * 10n ** 18n, // 777 LGM bonus

  // Evidence
  minEvidencePieces: 3,
  evidencePeriodBlocks: 777n, // ~32 minutes

  // Review
  reviewPeriodBlocks: 7777n, // ~5.4 hours

  // Bounty pools
  baseBountyPool: 2222n * 10n ** 18n, // 2222 LGM
  bountyReplenishmentRate: 0.01, // 1% of verification rewards

  // Recovery
  domainPauseOnUpheldChallenge: true,
  domainPauseDurationBlocks: 777n,

  // Slashing
  validatorSlashOnUpheldChallenge: 0.3, // 30% of stake
  endorserSlashOnUpheldChallenge: 0.15, // 15% of endorsement stake

  // Thresholds for automatic detection
  herfindahlTriggerThreshold: 0.4, // Auto-create bounty above this
  minVerificationsForAnalysis: 19, // Need 19 verifications to analyze

  // Failed challenge penalty
  failedChallengePenalty: 0.77, // 77% of stake goes to bounty pool
} as const;

/**
 * LGM token unit
 */
export const LGM = 10n ** 18n;

// ============================================================================
// Evidence Types
// ============================================================================

/**
 * Types of capture evidence
 */
export type CaptureEvidenceType =
  | "coordinated_voting" // Validators voting identically
  | "timing_correlation" // Suspicious timing patterns
  | "off_chain_coordination" // Evidence of external coordination
  | "false_fact_accepted" // Demonstrably false fact passed verification
  | "stake_manipulation" // Documented stake manipulation
  | "qualification_fraud" // Fake qualifications
  | "systematic_bias"; // Pattern of biased verdicts

/**
 * Proof type for evidence
 */
export type EvidenceProofType =
  | "merkle" // Merkle proof of on-chain data
  | "signature" // Signed attestation
  | "statistical" // Statistical analysis
  | "on_chain"; // Direct on-chain reference

/**
 * Evidence submitted with a capture challenge
 */
export interface CaptureEvidence {
  id: string;
  type: CaptureEvidenceType;
  description: string;

  // Supporting data
  factIds?: string[]; // Facts that should be re-verified
  validatorIds?: string[]; // Validators implicated
  blockRange?: { start: bigint; end: bigint };

  // Cryptographic proof
  dataHash: string;
  proofType: EvidenceProofType;
  proof: string;

  // Submission metadata
  submittedAt: bigint;
  submittedBy: string;
}

// ============================================================================
// Challenge Types
// ============================================================================

/**
 * Status of a capture challenge
 */
export type CaptureChallengeStatus =
  | "submitted" // Initial submission
  | "evidence_period" // Gathering additional evidence
  | "under_review" // Governance review
  | "extended_verification" // Re-verifying challenged facts
  | "upheld" // Challenge successful
  | "rejected" // Challenge failed
  | "expired"; // No resolution within deadline

/**
 * A challenge against domain capture
 */
export interface CaptureChallenge {
  id: string;
  domain: string;
  stratum: OntologicalStratum;

  // Challenger
  challenger: string; // DID
  challengeStake: bigint;

  // Evidence
  evidence: CaptureEvidence[];
  minEvidenceRequired: number;

  // Accused validators
  accusedValidators: string[];

  // Timing
  submittedAt: bigint;
  evidenceDeadline: bigint;
  reviewDeadline: bigint;

  // Status
  status: CaptureChallengeStatus;

  // Resolution
  resolution?: CaptureResolution;
}

/**
 * Actions to recover a captured domain
 */
export type DomainRecoveryAction =
  | { type: "revoke_qualifications"; validators: string[] }
  | {
      type: "increase_diversity_requirements";
      newRequirements: Record<string, unknown>;
    }
  | { type: "pause_new_claims"; durationBlocks: bigint }
  | { type: "require_governance_review"; claimTypes: string[] }
  | { type: "slash_endorsers"; endorsers: string[] };

/**
 * Resolution of a capture challenge
 */
export interface CaptureResolution {
  outcome: "upheld" | "rejected" | "partial";

  // Rewards/penalties
  challengerReward: bigint;
  validatorSlashes: Map<string, bigint>;

  // Fact changes
  factsInvalidated: string[];
  factsRequeuedForVerification: string[];

  // Domain recovery
  domainRecoveryActions: DomainRecoveryAction[];

  // Metadata
  resolvedAt: bigint;
  resolvedBy: "governance" | "automatic" | "settlement";
  justification: string;
}

// ============================================================================
// Bounty Pool Types
// ============================================================================

/**
 * Domain bounty pool
 */
export interface DomainBountyPool {
  domain: string;
  balance: bigint;

  // History
  challengesSubmitted: number;
  challengesUpheld: number;
  challengesRejected: number;
  totalRewardsPaid: bigint;

  // Auto-detection
  captureRiskScore: number;
  lastAnalyzedBlock: bigint;
  autoGeneratedBounty: boolean;

  // Timing
  createdAt: bigint;
  lastUpdatedAt: bigint;
}

// ============================================================================
// Detection Types
// ============================================================================

/**
 * Capture risk metrics for a domain
 */
export interface CaptureMetrics {
  herfindahlIndex: number; // Concentration measure (0-1)
  timingCorrelation: number; // How correlated verification timing is
  verdictCorrelation: number; // How correlated verdicts are
  uniqueValidators: number; // Number of distinct validators
  totalVerifications: number; // Total verifications analyzed
  top3ValidatorShare: number; // Share of verifications by top 3
}

/**
 * Verification history entry for analysis
 */
export interface VerificationHistoryEntry {
  claimId: string;
  blockNumber: bigint;
  validators: string[];
  verdicts: Array<"confirmed" | "rejected" | "uncertain">;
  finalOutcome: "accepted" | "rejected" | "contested";
  submissionTimestamps: bigint[];
}

/**
 * Capture risk analysis result
 */
export interface CaptureRiskAnalysis {
  domain: string;
  riskScore: number; // 0-1, higher = more risk
  metrics: CaptureMetrics;
  shouldCreateBounty: boolean;
  suggestedBountySize: bigint;
  analyzedAt: bigint;
  verificationCount: number;
}

// ============================================================================
// Validation Types
// ============================================================================

/**
 * Challenge validation result
 */
export interface ChallengeValidation {
  valid: boolean;
  errors: string[];
}

/**
 * Challenge submission input
 */
export interface ChallengeSubmissionInput {
  challenger: string;
  domain: string;
  evidence: CaptureEvidence[];
  stake: bigint;
}

// ============================================================================
// Helper Functions
// ============================================================================

/**
 * Create empty capture metrics
 */
export function createEmptyCaptureMetrics(): CaptureMetrics {
  return {
    herfindahlIndex: 0,
    timingCorrelation: 0,
    verdictCorrelation: 0,
    uniqueValidators: 0,
    totalVerifications: 0,
    top3ValidatorShare: 0,
  };
}

/**
 * Create empty bounty pool
 */
export function createEmptyBountyPool(
  domain: string,
  currentBlock: bigint,
): DomainBountyPool {
  return {
    domain,
    balance: 0n,
    challengesSubmitted: 0,
    challengesUpheld: 0,
    challengesRejected: 0,
    totalRewardsPaid: 0n,
    captureRiskScore: 0,
    lastAnalyzedBlock: currentBlock,
    autoGeneratedBounty: false,
    createdAt: currentBlock,
    lastUpdatedAt: currentBlock,
  };
}

/**
 * Generate challenge ID
 */
export function generateChallengeId(
  _challenger: string,
  domain: string,
  currentBlock: bigint,
): string {
  // Simple hash-like ID generation (in production, use proper hashing)
  return `cc-${domain}-${currentBlock.toString(16).slice(-8)}`;
}

/**
 * Check if challenge is still active
 */
export function isChallengeActive(challenge: CaptureChallenge): boolean {
  return (
    challenge.status === "submitted" ||
    challenge.status === "evidence_period" ||
    challenge.status === "under_review" ||
    challenge.status === "extended_verification"
  );
}

/**
 * Check if challenge is resolved
 */
export function isChallengeResolved(challenge: CaptureChallenge): boolean {
  return (
    challenge.status === "upheld" ||
    challenge.status === "rejected" ||
    challenge.status === "expired"
  );
}
