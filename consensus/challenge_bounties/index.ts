/**
 * Challenge Bounties System
 *
 * Creates economic incentives to monitor domains and expose capture
 * attempts through bounties that reward successful challengers.
 *
 * Key features:
 * - 77 LGM minimum challenge stake
 * - 7.7x reward multiplier on success
 * - 777 LGM capture exposure bonus
 * - Auto-detection at HHI > 0.40
 * - 30% validator slash on upheld challenges
 *
 * @module consensus/challenge_bounties
 */

// Types
export type {
  CaptureEvidenceType,
  EvidenceProofType,
  CaptureEvidence,
  CaptureChallengeStatus,
  CaptureChallenge,
  DomainRecoveryAction,
  CaptureResolution,
  DomainBountyPool,
  CaptureMetrics,
  VerificationHistoryEntry,
  CaptureRiskAnalysis,
  ChallengeValidation,
  ChallengeSubmissionInput,
} from "./types.js";

export {
  CHALLENGE_BOUNTY_CONFIG,
  LGM,
  createEmptyCaptureMetrics,
  createEmptyBountyPool,
  generateChallengeId,
  isChallengeActive,
  isChallengeResolved,
} from "./types.js";

// Challenge Flow
export {
  validateChallengeSubmission,
  submitCaptureChallenge,
  addEvidence,
  advanceToReview,
  checkChallengeExpiration,
  calculateChallengerReward,
  calculateValidatorSlashes,
  calculateFailedChallengePenalty,
  resolveCaptureChallenge,
  finalizeChallenge,
  getOrCreateBountyPool,
  fundBountyPool,
  replenishBountyPool,
  autoFundBountyPool,
} from "./challenge_flow.js";

// Detection
export {
  calculateHerfindahlIndex,
  calculateTopNShare,
  calculateTimingCorrelation,
  calculateVerdictCorrelation,
  analyzeCaptureRisk,
  getRiskLevelDescription,
  analyzeValidatorPair,
  findSuspiciousPairs,
} from "./detection.js";
