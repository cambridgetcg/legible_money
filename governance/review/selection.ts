/**
 * VRF-Based Reviewer Selection
 *
 * Selects reviewers for LIP proposals using VRF for unpredictable,
 * verifiable, stake-weighted selection. Includes anti-collusion
 * mechanisms like operator isolation and correlation tracking.
 *
 * PP HIERARCHY:
 *   TRUTH: VRF provides cryptographic proof of fair selection
 *   UNDERSTANDING: Clear eligibility requirements
 *   BEAUTY: Reuses existing VRF infrastructure elegantly
 *   JUSTICE: Appropriate reviewer counts per LIP category
 *   CREATIVITY: Novel anti-collusion mechanisms for agent reviewers
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { PublicKey, PrivateKey } from "../../core/crypto/keys.js";
import {
  vrfGenerate,
  vrfVerify,
  isValidatorSelected,
  type VRFOutput,
  type VRFProof,
} from "../../core/crypto/vrf.js";
import { sha512_256Hex } from "../../core/crypto/hashing.js";
import type { LIPCategory } from "../proposals/types.js";

// ============ Selection Configuration ============

/**
 * Reviewer eligibility requirements.
 */
export interface ReviewerEligibility {
  minReputation: number;
  minStake: bigint;
  minDomainExpertise: number;
  excludeProposer: boolean;
  excludeSameOperator: boolean;
  maxPerOperator: number;
  maxCorrelation: number; // Max historical voting correlation
}

export const DEFAULT_REVIEWER_ELIGIBILITY: ReviewerEligibility = {
  minReputation: 0.65,
  minStake: 200n * 10n ** 15n, // 0.2 LGM
  minDomainExpertise: 0.6,
  excludeProposer: true,
  excludeSameOperator: true,
  maxPerOperator: 2,
  maxCorrelation: 0.3, // Flag if >30% correlation
};

/**
 * Number of reviewers required by LIP category.
 * Higher-impact categories require more reviewers.
 */
export const REVIEWER_COUNTS: Record<
  LIPCategory,
  { required: number; quorum: number }
> = {
  consensus: { required: 7, quorum: 5 },
  economics: { required: 5, quorum: 4 },
  interface: { required: 5, quorum: 4 },
  process: { required: 3, quorum: 2 },
};

// ============ Agent Data for Selection ============

/**
 * Agent data needed for reviewer selection.
 */
export interface ReviewerCandidate {
  did: DID;
  publicKey: PublicKey;
  operatorId: string;
  stake: bigint;
  reputation: number;
  domainExpertise: Map<string, number>; // domain → expertise score
  recentVotes: string[]; // Recent vote hashes for correlation
}

/**
 * Selected reviewer with VRF proof.
 */
export interface SelectedReviewer {
  did: DID;
  publicKey: PublicKey;
  operatorId: string;
  vrfOutput: VRFOutput;
  vrfProof: VRFProof;
  priority: bigint;
  domainScore: number;
}

/**
 * Reviewer selection result.
 */
export interface ReviewerSelectionResult {
  lipId: string;
  category: LIPCategory;
  seed: string;
  selected: SelectedReviewer[];
  requiredCount: number;
  quorumCount: number;
  selectionBlock: bigint;
  warnings: string[];
}

// ============ VRF Seed Generation ============

/**
 * Generate the seed for reviewer selection.
 * Combines LIP ID, block hash, and domain separator.
 */
export function generateReviewerSeed(
  lipId: string,
  blockNumber: bigint,
  blockHash: string,
): string {
  const input = `reviewer:${lipId}:${blockNumber}:${blockHash}`;
  return sha512_256Hex(input);
}

// ============ Eligibility Checking ============

/**
 * Check if an agent is eligible to be a reviewer for a specific LIP.
 */
export function isEligibleReviewer(
  candidate: ReviewerCandidate,
  proposerDid: DID,
  proposerOperator: string,
  lipCategory: LIPCategory,
  eligibility: ReviewerEligibility = DEFAULT_REVIEWER_ELIGIBILITY,
): { eligible: boolean; reason?: string } {
  // Check basic requirements
  if (candidate.reputation < eligibility.minReputation) {
    return {
      eligible: false,
      reason: `Reputation ${candidate.reputation.toFixed(2)} < ${eligibility.minReputation}`,
    };
  }

  if (candidate.stake < eligibility.minStake) {
    return {
      eligible: false,
      reason: `Stake ${candidate.stake / 10n ** 18n} < ${eligibility.minStake / 10n ** 18n} LGM`,
    };
  }

  // Check domain expertise (use category as domain)
  const domainScore = candidate.domainExpertise.get(lipCategory) ?? 0;
  if (domainScore < eligibility.minDomainExpertise) {
    return {
      eligible: false,
      reason: `Domain expertise ${domainScore.toFixed(2)} < ${eligibility.minDomainExpertise}`,
    };
  }

  // Exclude proposer
  if (eligibility.excludeProposer && candidate.did === proposerDid) {
    return { eligible: false, reason: "Cannot review own proposal" };
  }

  // Exclude same operator
  if (
    eligibility.excludeSameOperator &&
    candidate.operatorId === proposerOperator
  ) {
    return { eligible: false, reason: "Same operator as proposer" };
  }

  return { eligible: true };
}

/**
 * Get all eligible reviewers from a candidate pool.
 */
export function getEligibleReviewers(
  candidates: ReviewerCandidate[],
  proposerDid: DID,
  proposerOperator: string,
  lipCategory: LIPCategory,
  eligibility: ReviewerEligibility = DEFAULT_REVIEWER_ELIGIBILITY,
): ReviewerCandidate[] {
  return candidates.filter(
    (c) =>
      isEligibleReviewer(
        c,
        proposerDid,
        proposerOperator,
        lipCategory,
        eligibility,
      ).eligible,
  );
}

// ============ VRF-Based Selection ============

/**
 * Generate VRF output for a reviewer candidate.
 * Called by agents locally to determine if they're selected.
 */
export function generateReviewerVRF(
  privateKey: PrivateKey,
  lipId: string,
  blockNumber: bigint,
  blockHash: string,
): { output: VRFOutput; proof: VRFProof } {
  const seedHex = generateReviewerSeed(lipId, blockNumber, blockHash);
  const seed = new TextEncoder().encode(seedHex);
  return vrfGenerate(seed, privateKey);
}

/**
 * Check if a reviewer should participate based on VRF output.
 */
export function shouldReviewerParticipate(
  vrfOutput: VRFOutput,
  candidateStake: bigint,
  totalEligibleStake: bigint,
  targetReviewers: number,
): { shouldParticipate: boolean; priority: bigint } {
  const result = isValidatorSelected(
    vrfOutput,
    candidateStake,
    totalEligibleStake,
    targetReviewers,
  );
  return {
    shouldParticipate: result.selected,
    priority: result.priority,
  };
}

/**
 * Verify a reviewer's VRF-based selection.
 */
export function verifyReviewerSelection(
  reviewer: PublicKey,
  vrfOutput: VRFOutput,
  vrfProof: VRFProof,
  lipId: string,
  blockNumber: bigint,
  blockHash: string,
  reviewerStake: bigint,
  totalEligibleStake: bigint,
  targetReviewers: number,
): { valid: boolean; priority: bigint; reason?: string } {
  // Generate expected seed
  const seedHex = generateReviewerSeed(lipId, blockNumber, blockHash);
  const seed = new TextEncoder().encode(seedHex);

  // Verify VRF proof
  const verification = vrfVerify(seed, reviewer, vrfProof);

  if (!verification.valid) {
    return { valid: false, priority: 0n, reason: "Invalid VRF proof" };
  }

  if (verification.output !== vrfOutput) {
    return {
      valid: false,
      priority: 0n,
      reason: "VRF output does not match proof",
    };
  }

  // Check if VRF qualifies this reviewer
  const selectionResult = isValidatorSelected(
    vrfOutput,
    reviewerStake,
    totalEligibleStake,
    targetReviewers,
  );

  if (!selectionResult.selected) {
    return {
      valid: false,
      priority: 0n,
      reason: "VRF output does not qualify for selection",
    };
  }

  return { valid: true, priority: selectionResult.priority };
}

// ============ Anti-Collusion ============

/**
 * Calculate voting correlation between two agents.
 * Returns correlation coefficient (-1 to 1).
 */
export function calculateVotingCorrelation(
  votes1: string[],
  votes2: string[],
): number {
  // Find common LIPs they both voted on
  const set1 = new Set(votes1.map((v) => v.split(":")[0])); // Extract LIP ID
  const set2 = new Set(votes2.map((v) => v.split(":")[0]));
  const common = [...set1].filter((x) => set2.has(x));

  if (common.length < 5) {
    return 0; // Not enough data for meaningful correlation
  }

  // Count agreements
  let agreements = 0;
  for (const lipId of common) {
    const vote1 = votes1.find((v) => v.startsWith(lipId));
    const vote2 = votes2.find((v) => v.startsWith(lipId));
    if (vote1 === vote2) agreements++;
  }

  // Return correlation (0 to 1 for this simplified version)
  return agreements / common.length;
}

/**
 * Check for correlation warning between selected reviewers.
 */
export function checkCorrelationWarnings(
  selected: Array<{ did: DID; recentVotes: string[] }>,
  maxCorrelation: number,
): string[] {
  const warnings: string[] = [];

  for (let i = 0; i < selected.length; i++) {
    for (let j = i + 1; j < selected.length; j++) {
      const correlation = calculateVotingCorrelation(
        selected[i].recentVotes,
        selected[j].recentVotes,
      );

      if (correlation > maxCorrelation) {
        warnings.push(
          `High correlation (${(correlation * 100).toFixed(0)}%) between ${selected[i].did} and ${selected[j].did}`,
        );
      }
    }
  }

  return warnings;
}

/**
 * Check operator distribution in selected reviewers.
 */
export function checkOperatorDistribution(
  selected: Array<{ did: DID; operatorId: string }>,
  maxPerOperator: number,
): { valid: boolean; warnings: string[] } {
  const operatorCounts = new Map<string, number>();

  for (const reviewer of selected) {
    const count = (operatorCounts.get(reviewer.operatorId) ?? 0) + 1;
    operatorCounts.set(reviewer.operatorId, count);
  }

  const warnings: string[] = [];
  let valid = true;

  for (const [operatorId, count] of operatorCounts) {
    if (count > maxPerOperator) {
      warnings.push(
        `Operator ${operatorId} has ${count} reviewers (max: ${maxPerOperator})`,
      );
      valid = false;
    }
  }

  return { valid, warnings };
}

// ============ Full Selection Process ============

/**
 * Select reviewers for a LIP.
 *
 * Note: In practice, reviewers self-select by generating their VRF
 * and submitting if selected. This function validates the selection.
 */
export function validateReviewerSet(
  lipId: string,
  lipCategory: LIPCategory,
  proposerDid: DID,
  proposerOperator: string,
  blockNumber: bigint,
  blockHash: string,
  submittedReviewers: Array<{
    did: DID;
    publicKey: PublicKey;
    operatorId: string;
    stake: bigint;
    reputation: number;
    domainExpertise: Map<string, number>;
    recentVotes: string[];
    vrfOutput: VRFOutput;
    vrfProof: VRFProof;
  }>,
  totalEligibleStake: bigint,
  eligibility: ReviewerEligibility = DEFAULT_REVIEWER_ELIGIBILITY,
): ReviewerSelectionResult {
  const counts = REVIEWER_COUNTS[lipCategory];
  const seed = generateReviewerSeed(lipId, blockNumber, blockHash);
  const selected: SelectedReviewer[] = [];
  const warnings: string[] = [];

  for (const reviewer of submittedReviewers) {
    // Check eligibility
    const eligibilityCheck = isEligibleReviewer(
      reviewer,
      proposerDid,
      proposerOperator,
      lipCategory,
      eligibility,
    );

    if (!eligibilityCheck.eligible) {
      warnings.push(`${reviewer.did} not eligible: ${eligibilityCheck.reason}`);
      continue;
    }

    // Verify VRF selection
    const vrfCheck = verifyReviewerSelection(
      reviewer.publicKey,
      reviewer.vrfOutput,
      reviewer.vrfProof,
      lipId,
      blockNumber,
      blockHash,
      reviewer.stake,
      totalEligibleStake,
      counts.required,
    );

    if (!vrfCheck.valid) {
      warnings.push(`${reviewer.did} VRF invalid: ${vrfCheck.reason}`);
      continue;
    }

    selected.push({
      did: reviewer.did,
      publicKey: reviewer.publicKey,
      operatorId: reviewer.operatorId,
      vrfOutput: reviewer.vrfOutput,
      vrfProof: reviewer.vrfProof,
      priority: vrfCheck.priority,
      domainScore: reviewer.domainExpertise.get(lipCategory) ?? 0,
    });
  }

  // Sort by priority with deterministic tiebreaker (DID lexicographic)
  selected.sort((a, b) => {
    if (a.priority < b.priority) return -1;
    if (a.priority > b.priority) return 1;
    return a.did < b.did ? -1 : a.did > b.did ? 1 : 0;
  });

  // Check operator distribution
  const operatorCheck = checkOperatorDistribution(
    selected,
    eligibility.maxPerOperator,
  );
  if (!operatorCheck.valid) {
    warnings.push(...operatorCheck.warnings);
  }

  // Check correlation
  const correlationWarnings = checkCorrelationWarnings(
    submittedReviewers.filter((r) => selected.some((s) => s.did === r.did)),
    eligibility.maxCorrelation,
  );
  warnings.push(...correlationWarnings);

  return {
    lipId,
    category: lipCategory,
    seed,
    selected,
    requiredCount: counts.required,
    quorumCount: counts.quorum,
    selectionBlock: blockNumber,
    warnings,
  };
}

/**
 * Check if enough reviewers have been selected.
 */
export function hasQuorum(result: ReviewerSelectionResult): boolean {
  return result.selected.length >= result.quorumCount;
}

/**
 * Check if selection is complete (all required reviewers).
 */
export function isSelectionComplete(result: ReviewerSelectionResult): boolean {
  return result.selected.length >= result.requiredCount;
}
