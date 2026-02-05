/**
 * Commit-Reveal Review Protocol
 *
 * Implements commit-reveal for LIP reviews to prevent copying and
 * ensure independent reviewer decisions. Reviewers commit a hash
 * of their review, then reveal after the commit phase ends.
 *
 * PP HIERARCHY:
 *   TRUTH: Commitment hash proves review was formed independently
 *   UNDERSTANDING: Clear phases with explicit timing
 *   BEAUTY: Elegant two-phase protocol
 *   JUSTICE: Non-reveal slashing enforces participation
 *   CREATIVITY: Adapted for multi-dimensional reviews (not just votes)
 */

import { sha512_256Hex } from "../../core/crypto/hashing.js";
import type { DID } from "../../core/agents/identity/did.js";

// ============ Timing Configuration ============

/**
 * Review phase durations in blocks.
 * At 2521ms per block:
 * - Commit: 5000 blocks ≈ 3.5 hours
 * - Reveal: 2500 blocks ≈ 1.75 hours
 */
export const REVIEW_TIMING = {
  commitPhase: 5000n, // Blocks for commit phase
  revealPhase: 2500n, // Blocks for reveal phase
  nonRevealSlash: 1.0, // 100% of reviewer stake slashed for non-reveal
  lateRevealPenalty: 0.2, // 20% penalty for late reveal (in grace period)
  graceBlocks: 500n, // Grace period after reveal phase
} as const;

// ============ Review Structure ============

/**
 * Review verdict options.
 */
export type ReviewVerdict = "approve" | "request_changes" | "reject";

/**
 * Security classification for code changes.
 */
export type SecurityLevel = "critical" | "high" | "medium" | "low";

/**
 * Three Labs alignment scores.
 * Rates how well the proposal aligns with Legible Money principles.
 */
export interface ThreeLabsScores {
  legible: number; // 0-100: Is it clear and understandable?
  honest: number; // 0-100: Does it promote truth/transparency?
  agentFirst: number; // 0-100: Does it benefit AI agents?
}

/**
 * Complete review content (before commitment).
 */
export interface ReviewContent {
  lipId: string;
  reviewer: DID;
  verdict: ReviewVerdict;
  technicalScore: number; // 0-100
  securityLevel: SecurityLevel;
  threeLabsScores: ThreeLabsScores;
  summary: string;
  concerns: string[];
  suggestions: string[];
  nonce: string; // Random nonce for commitment
}

/**
 * Commitment hash of a review.
 */
export interface ReviewCommitment {
  commitmentId: string;
  lipId: string;
  reviewer: DID;
  commitmentHash: string;
  committedAt: bigint; // Block number
  status: "committed" | "revealed" | "slashed" | "expired";
}

/**
 * Revealed review with verification.
 */
export interface ReviewReveal {
  commitmentId: string;
  lipId: string;
  reviewer: DID;
  content: ReviewContent;
  revealedAt: bigint;
  verified: boolean;
  verificationError?: string;
}

// ============ Commitment Functions ============

/**
 * Generate a random nonce for review commitment.
 */
export function generateReviewNonce(): string {
  const bytes = new Uint8Array(32);
  crypto.getRandomValues(bytes);
  return Array.from(bytes)
    .map((b) => b.toString(16).padStart(2, "0"))
    .join("");
}

/**
 * Calculate commitment hash from review content.
 */
export function calculateCommitmentHash(content: ReviewContent): string {
  // Deterministic serialization of review content
  const serialized = JSON.stringify({
    lipId: content.lipId,
    reviewer: content.reviewer,
    verdict: content.verdict,
    technicalScore: content.technicalScore,
    securityLevel: content.securityLevel,
    threeLabsScores: content.threeLabsScores,
    summary: content.summary,
    concerns: content.concerns.sort(),
    suggestions: content.suggestions.sort(),
    nonce: content.nonce,
  });

  return sha512_256Hex(serialized);
}

/**
 * Create a review commitment.
 */
export function createCommitment(
  lipId: string,
  reviewer: DID,
  content: ReviewContent,
  currentBlock: bigint,
): ReviewCommitment {
  const hash = calculateCommitmentHash(content);

  return {
    commitmentId: `commit-${lipId}-${reviewer}-${currentBlock}`,
    lipId,
    reviewer,
    commitmentHash: hash,
    committedAt: currentBlock,
    status: "committed",
  };
}

// ============ Reveal Functions ============

/**
 * Verify a reveal matches its commitment.
 */
export function verifyReveal(
  commitment: ReviewCommitment,
  content: ReviewContent,
): { valid: boolean; error?: string } {
  // Check LIP ID matches
  if (content.lipId !== commitment.lipId) {
    return { valid: false, error: "LIP ID mismatch" };
  }

  // Check reviewer matches
  if (content.reviewer !== commitment.reviewer) {
    return { valid: false, error: "Reviewer mismatch" };
  }

  // Verify hash
  const hash = calculateCommitmentHash(content);
  if (hash !== commitment.commitmentHash) {
    return {
      valid: false,
      error: "Commitment hash mismatch - review was modified",
    };
  }

  return { valid: true };
}

/**
 * Process a review reveal.
 */
export function processReveal(
  commitment: ReviewCommitment,
  content: ReviewContent,
  currentBlock: bigint,
  phaseStartBlock: bigint,
): ReviewReveal {
  // Verify the reveal
  const verification = verifyReveal(commitment, content);

  // Check timing
  const commitPhaseEnd = phaseStartBlock + REVIEW_TIMING.commitPhase;
  const revealPhaseEnd = commitPhaseEnd + REVIEW_TIMING.revealPhase;
  const graceEnd = revealPhaseEnd + REVIEW_TIMING.graceBlocks;

  let verificationError = verification.error;

  if (currentBlock < commitPhaseEnd) {
    verificationError = "Reveal submitted during commit phase";
  } else if (currentBlock > graceEnd) {
    verificationError = "Reveal submitted after grace period";
  }

  return {
    commitmentId: commitment.commitmentId,
    lipId: commitment.lipId,
    reviewer: commitment.reviewer,
    content,
    revealedAt: currentBlock,
    verified: verification.valid && !verificationError,
    verificationError,
  };
}

// ============ Phase Management ============

/**
 * Review round phases.
 */
export type ReviewPhase = "commit" | "reveal" | "grace" | "complete";

/**
 * Determine current review phase.
 */
export function getCurrentPhase(
  currentBlock: bigint,
  phaseStartBlock: bigint,
): { phase: ReviewPhase; blocksRemaining: bigint } {
  const elapsed = currentBlock - phaseStartBlock;
  const commitEnd = REVIEW_TIMING.commitPhase;
  const revealEnd = commitEnd + REVIEW_TIMING.revealPhase;
  const graceEnd = revealEnd + REVIEW_TIMING.graceBlocks;

  if (elapsed < commitEnd) {
    return { phase: "commit", blocksRemaining: commitEnd - elapsed };
  }

  if (elapsed < revealEnd) {
    return { phase: "reveal", blocksRemaining: revealEnd - elapsed };
  }

  if (elapsed < graceEnd) {
    return { phase: "grace", blocksRemaining: graceEnd - elapsed };
  }

  return { phase: "complete", blocksRemaining: 0n };
}

/**
 * Check if commitment is allowed in current phase.
 */
export function canCommit(
  currentBlock: bigint,
  phaseStartBlock: bigint,
): boolean {
  const { phase } = getCurrentPhase(currentBlock, phaseStartBlock);
  return phase === "commit";
}

/**
 * Check if reveal is allowed in current phase.
 */
export function canReveal(
  currentBlock: bigint,
  phaseStartBlock: bigint,
): boolean {
  const { phase } = getCurrentPhase(currentBlock, phaseStartBlock);
  return phase === "reveal" || phase === "grace";
}

/**
 * Check if reveal is in grace period (late penalty applies).
 */
export function isInGracePeriod(
  currentBlock: bigint,
  phaseStartBlock: bigint,
): boolean {
  const { phase } = getCurrentPhase(currentBlock, phaseStartBlock);
  return phase === "grace";
}

// ============ Review Round Management ============

/**
 * Review round state.
 */
export interface ReviewRound {
  roundId: string;
  lipId: string;
  phaseStartBlock: bigint;
  commitments: Map<DID, ReviewCommitment>;
  reveals: Map<DID, ReviewReveal>;
  slashed: Set<DID>;
  status: "active" | "finalized";
}

/**
 * Create a new review round.
 */
export function createReviewRound(
  lipId: string,
  startBlock: bigint,
): ReviewRound {
  return {
    roundId: `round-${lipId}-${startBlock}`,
    lipId,
    phaseStartBlock: startBlock,
    commitments: new Map(),
    reveals: new Map(),
    slashed: new Set(),
    status: "active",
  };
}

/**
 * Add a commitment to a review round.
 */
export function addCommitment(
  round: ReviewRound,
  commitment: ReviewCommitment,
  currentBlock: bigint,
): { success: boolean; error?: string } {
  // Check phase
  if (!canCommit(currentBlock, round.phaseStartBlock)) {
    return { success: false, error: "Not in commit phase" };
  }

  // Check for duplicate
  if (round.commitments.has(commitment.reviewer)) {
    return { success: false, error: "Already committed" };
  }

  round.commitments.set(commitment.reviewer, commitment);
  return { success: true };
}

/**
 * Add a reveal to a review round.
 */
export function addReveal(
  round: ReviewRound,
  content: ReviewContent,
  currentBlock: bigint,
): { success: boolean; reveal?: ReviewReveal; error?: string } {
  // Check phase
  if (!canReveal(currentBlock, round.phaseStartBlock)) {
    return { success: false, error: "Not in reveal phase" };
  }

  // Check for commitment
  const commitment = round.commitments.get(content.reviewer);
  if (!commitment) {
    return { success: false, error: "No commitment found" };
  }

  // Check for duplicate reveal
  if (round.reveals.has(content.reviewer)) {
    return { success: false, error: "Already revealed" };
  }

  // Process reveal
  const reveal = processReveal(
    commitment,
    content,
    currentBlock,
    round.phaseStartBlock,
  );

  if (!reveal.verified) {
    return { success: false, error: reveal.verificationError };
  }

  round.reveals.set(content.reviewer, reveal);

  // Update commitment status
  commitment.status = "revealed";

  return { success: true, reveal };
}

/**
 * Finalize a review round, slashing non-revealers.
 */
export function finalizeRound(
  round: ReviewRound,
  currentBlock: bigint,
): {
  finalized: boolean;
  slashedReviewers: DID[];
  validReviews: ReviewReveal[];
  error?: string;
} {
  const { phase } = getCurrentPhase(currentBlock, round.phaseStartBlock);

  if (phase !== "complete") {
    return {
      finalized: false,
      slashedReviewers: [],
      validReviews: [],
      error: "Round not complete",
    };
  }

  const slashedReviewers: DID[] = [];
  const validReviews: ReviewReveal[] = [];

  // Process all commitments
  for (const [reviewer, commitment] of round.commitments) {
    if (commitment.status === "revealed") {
      const reveal = round.reveals.get(reviewer);
      if (reveal && reveal.verified) {
        validReviews.push(reveal);
      }
    } else {
      // Non-reveal: slash
      commitment.status = "slashed";
      round.slashed.add(reviewer);
      slashedReviewers.push(reviewer);
    }
  }

  round.status = "finalized";

  return {
    finalized: true,
    slashedReviewers,
    validReviews,
  };
}

// ============ Review Aggregation ============

/**
 * Aggregated review results.
 */
export interface AggregatedReview {
  lipId: string;
  totalReviews: number;
  verdicts: {
    approve: number;
    request_changes: number;
    reject: number;
  };
  averageScores: {
    technical: number;
    legible: number;
    honest: number;
    agentFirst: number;
  };
  securityConcerns: {
    critical: number;
    high: number;
    medium: number;
    low: number;
  };
  commonConcerns: string[];
  recommendation: ReviewVerdict;
}

/**
 * Aggregate reviews for a LIP.
 */
export function aggregateReviews(reviews: ReviewReveal[]): AggregatedReview {
  if (reviews.length === 0) {
    throw new Error("No reviews to aggregate");
  }

  const lipId = reviews[0].lipId;

  // Count verdicts
  const verdicts = { approve: 0, request_changes: 0, reject: 0 };
  for (const review of reviews) {
    verdicts[review.content.verdict]++;
  }

  // Calculate average scores
  const technicalScores = reviews.map((r) => r.content.technicalScore);
  const legibleScores = reviews.map((r) => r.content.threeLabsScores.legible);
  const honestScores = reviews.map((r) => r.content.threeLabsScores.honest);
  const agentFirstScores = reviews.map(
    (r) => r.content.threeLabsScores.agentFirst,
  );

  const avg = (arr: number[]) => arr.reduce((a, b) => a + b, 0) / arr.length;

  // Count security levels
  const securityConcerns = { critical: 0, high: 0, medium: 0, low: 0 };
  for (const review of reviews) {
    securityConcerns[review.content.securityLevel]++;
  }

  // Find common concerns (mentioned by 2+ reviewers)
  const concernCounts = new Map<string, number>();
  for (const review of reviews) {
    for (const concern of review.content.concerns) {
      concernCounts.set(concern, (concernCounts.get(concern) ?? 0) + 1);
    }
  }
  const commonConcerns = [...concernCounts.entries()]
    .filter(([_, count]) => count >= 2)
    .map(([concern]) => concern);

  // Determine recommendation
  let recommendation: ReviewVerdict;
  if (securityConcerns.critical > 0) {
    recommendation = "reject";
  } else if (verdicts.reject > verdicts.approve) {
    recommendation = "reject";
  } else if (verdicts.request_changes > verdicts.approve) {
    recommendation = "request_changes";
  } else {
    recommendation = "approve";
  }

  return {
    lipId,
    totalReviews: reviews.length,
    verdicts,
    averageScores: {
      technical: avg(technicalScores),
      legible: avg(legibleScores),
      honest: avg(honestScores),
      agentFirst: avg(agentFirstScores),
    },
    securityConcerns,
    commonConcerns,
    recommendation,
  };
}

/**
 * Calculate slash amount for non-reveal.
 */
export function calculateNonRevealSlash(reviewerStake: bigint): bigint {
  return (
    (reviewerStake * BigInt(Math.floor(REVIEW_TIMING.nonRevealSlash * 10000))) /
    10000n
  );
}

/**
 * Calculate late reveal penalty.
 */
export function calculateLateRevealPenalty(reviewerStake: bigint): bigint {
  return (
    (reviewerStake *
      BigInt(Math.floor(REVIEW_TIMING.lateRevealPenalty * 10000))) /
    10000n
  );
}
