/**
 * Four-Tier Code Review System
 *
 * Categorizes code changes by risk level and defines appropriate
 * review requirements for each tier. Higher-risk changes require
 * more reviewers, mandatory human review, and longer soak times.
 *
 * INTEGRATES WITH ARCHITECTURE ENFORCEMENT:
 * - IMMUTABLE modules (Tier 1 architecture) are REJECTED - cannot be changed
 * - FROZEN modules (Tier 2 architecture) map to review Tier 4 (highest scrutiny)
 * - STABLE modules (Tier 3 architecture) map to review Tier 3
 * - GOVERNED/OPEN modules use normal classification
 *
 * PP HIERARCHY:
 *   TRUTH: Risk levels based on objective file/change characteristics
 *   UNDERSTANDING: Clear tier definitions and requirements
 *   BEAUTY: Elegant escalation from low to critical risk
 *   JUSTICE: Appropriate scrutiny for each risk level
 *   CREATIVITY: Novel human-agent hybrid review for security
 */

import type { DID } from "../../core/agents/identity/did.js";
import {
  classifyModule,
  type ArchitectureTier,
} from "../../core/architecture/tiers.js";

// ============ Risk Tier Definitions ============

/**
 * Code review risk tier.
 * REJECTED means the file is IMMUTABLE and cannot be changed.
 */
export type ReviewTier = 1 | 2 | 3 | 4 | "REJECTED";

/**
 * Review requirements for a tier.
 */
export interface TierRequirements {
  tier: ReviewTier;
  name: string;
  description: string;

  // Reviewer requirements
  agentReviewers: number;
  agentApprovals: number;
  humanReviewRequired: boolean;
  humanReviewers: number;
  humanSampling: number; // % of tier changes that get human review

  // Security requirements
  securityScan: "basic" | "full" | "full_formal" | "full_formal_external";
  testnetSoak: bigint; // Blocks (0 = not required)
  simulationRequired: boolean;

  // Governance requirements
  lipRequired: boolean;
  governanceVote: boolean;
  governanceThreshold: number; // % approval needed
}

/**
 * Four-tier review requirements.
 */
export const TIER_REQUIREMENTS: Record<ReviewTier, TierRequirements> = {
  1: {
    tier: 1,
    name: "Documentation/Tests",
    description: "Low risk: Documentation, tests, comments only",
    agentReviewers: 2,
    agentApprovals: 1,
    humanReviewRequired: false,
    humanReviewers: 0,
    humanSampling: 0,
    securityScan: "basic",
    testnetSoak: 0n,
    simulationRequired: false,
    lipRequired: false,
    governanceVote: false,
    governanceThreshold: 0,
  },
  2: {
    tier: 2,
    name: "Non-Consensus Code",
    description: "Medium risk: Application logic, APIs, utilities",
    agentReviewers: 3,
    agentApprovals: 2,
    humanReviewRequired: false,
    humanReviewers: 0,
    humanSampling: 0.2, // 20% get human review
    securityScan: "full",
    testnetSoak: 68400n, // 48 hours at 2521ms/block
    simulationRequired: false,
    lipRequired: false,
    governanceVote: false,
    governanceThreshold: 0,
  },
  3: {
    tier: 3,
    name: "Consensus/State/Crypto",
    description: "High risk: Consensus, state management, cryptography",
    agentReviewers: 5,
    agentApprovals: 4,
    humanReviewRequired: true,
    humanReviewers: 1,
    humanSampling: 1.0, // 100% human review
    securityScan: "full_formal",
    testnetSoak: 239400n, // 7 days
    simulationRequired: false,
    lipRequired: true,
    governanceVote: false,
    governanceThreshold: 0,
  },
  4: {
    tier: 4,
    name: "Economic Parameters",
    description: "Critical: Token economics, slashing rates, rewards",
    agentReviewers: 7,
    agentApprovals: 6,
    humanReviewRequired: true,
    humanReviewers: 2,
    humanSampling: 1.0,
    securityScan: "full_formal_external",
    testnetSoak: 478800n, // 14 days
    simulationRequired: true,
    lipRequired: true,
    governanceVote: true,
    governanceThreshold: 0.75, // 75% approval
  },
};

// ============ File Classification ============

/**
 * High-risk file patterns that trigger mandatory human review.
 */
export const MANDATORY_HUMAN_REVIEW_PATTERNS = [
  "core/crypto/*",
  "consensus/pot/*",
  "state/transitions.ts",
  "state/state_manager.ts",
  "**/private_key*",
  "**/secret*",
  "**/credentials*",
] as const;

/**
 * Tier 4 (economic) file patterns.
 */
export const ECONOMIC_PATTERNS = [
  "core/rewards/*",
  "consensus/pot/confidence.ts",
  "consensus/pot/finality.ts",
  "**/slashing*",
  "**/stake*",
  "**/economics*",
  "**/tokenomics*",
] as const;

/**
 * Tier 3 (consensus/state/crypto) file patterns.
 */
export const CONSENSUS_PATTERNS = [
  "core/crypto/*",
  "core/logic/*",
  "consensus/*",
  "state/*",
  "**/merkle*",
  "**/vrf*",
  "**/signature*",
] as const;

/**
 * Tier 1 (docs/tests) file patterns.
 */
export const DOCS_TESTS_PATTERNS = [
  "docs/*",
  "**/*.md",
  "**/*.test.ts",
  "**/*.spec.ts",
  "**/__tests__/*",
  "**/test/*",
  "**/tests/*",
] as const;

/**
 * Classify a file path into a review tier.
 * First checks architecture tier for immutability, then applies risk classification.
 */
export function classifyFile(filePath: string): ReviewTier {
  // Normalize path
  const normalizedPath = filePath.replace(/\\/g, "/");

  // FIRST: Check architecture tier for immutability
  const archTier = classifyModule(normalizedPath);

  // IMMUTABLE modules cannot be changed - REJECTED
  if (archTier === "IMMUTABLE") {
    return "REJECTED";
  }

  // FROZEN modules (protocol core) always get highest scrutiny
  if (archTier === "FROZEN") {
    return 4;
  }

  // STABLE modules (domain logic) get high scrutiny
  if (archTier === "STABLE") {
    return 3;
  }

  // For GOVERNED and OPEN, use pattern-based classification
  // Check patterns from highest to lowest priority
  for (const pattern of ECONOMIC_PATTERNS) {
    if (matchPattern(normalizedPath, pattern)) {
      return 4;
    }
  }

  for (const pattern of CONSENSUS_PATTERNS) {
    if (matchPattern(normalizedPath, pattern)) {
      return 3;
    }
  }

  for (const pattern of DOCS_TESTS_PATTERNS) {
    if (matchPattern(normalizedPath, pattern)) {
      return 1;
    }
  }

  // Default to tier 2
  return 2;
}

/**
 * Check if a file is immutable and cannot be changed.
 */
export function isImmutableFile(filePath: string): boolean {
  return classifyFile(filePath) === "REJECTED";
}

/**
 * Get rejection reason for immutable files.
 */
export function getImmutableRejectionReason(filePath: string): string | null {
  const archTier = classifyModule(filePath);
  if (archTier !== "IMMUTABLE") {
    return null;
  }

  return (
    `File ${filePath} is in the IMMUTABLE architecture tier and cannot be modified. ` +
    `This includes cryptographic primitives, logic foundations, and canonical type definitions. ` +
    `These modules are frozen forever to maintain protocol security and integrity.`
  );
}

/**
 * Simple glob pattern matching.
 */
function matchPattern(path: string, pattern: string): boolean {
  // Convert glob to regex
  const regexPattern = pattern
    .replace(/\*\*/g, "<<<GLOBSTAR>>>")
    .replace(/\*/g, "[^/]*")
    .replace(/<<<GLOBSTAR>>>/g, ".*");

  const regex = new RegExp(`^${regexPattern}$|/${regexPattern}$`);
  return regex.test(path);
}

/**
 * Check if a file requires mandatory human review.
 */
export function requiresMandatoryHumanReview(filePath: string): boolean {
  const normalizedPath = filePath.replace(/\\/g, "/");

  for (const pattern of MANDATORY_HUMAN_REVIEW_PATTERNS) {
    if (matchPattern(normalizedPath, pattern)) {
      return true;
    }
  }

  return false;
}

// ============ Change Classification ============

/**
 * A code change for review.
 */
export interface CodeChange {
  changeId: string;
  author: DID;
  files: string[];
  additions: number;
  deletions: number;
  description: string;
  linkedLIP?: string;
}

/**
 * Classification result for a code change.
 */
export interface ChangeClassification {
  change: CodeChange;
  tier: ReviewTier;
  requirements: TierRequirements | null; // null if REJECTED
  triggers: ClassificationTrigger[];
  mandatoryHumanReview: boolean;
  rejected: boolean;
  rejectionReasons: string[];
}

/**
 * Why a change was classified at a tier.
 */
export interface ClassificationTrigger {
  reason: string;
  file?: string;
  tier: ReviewTier | "REJECTED";
}

/**
 * Classify a code change.
 */
export function classifyChange(change: CodeChange): ChangeClassification {
  const triggers: ClassificationTrigger[] = [];
  const rejectionReasons: string[] = [];
  let maxTier: ReviewTier = 1;
  let mandatoryHuman = false;
  let hasRejectedFile = false;

  // Check each file
  for (const file of change.files) {
    const fileTier = classifyFile(file);

    // Handle REJECTED (immutable) files
    if (fileTier === "REJECTED") {
      hasRejectedFile = true;
      const reason = getImmutableRejectionReason(file);
      if (reason) {
        rejectionReasons.push(reason);
      }
      triggers.push({
        reason: "IMMUTABLE - Cannot be changed",
        file,
        tier: "REJECTED",
      });
      continue;
    }

    if (fileTier > maxTier) {
      maxTier = fileTier;
      triggers.push({
        reason: `File classified as tier ${fileTier}`,
        file,
        tier: fileTier,
      });
    }

    if (requiresMandatoryHumanReview(file)) {
      mandatoryHuman = true;
      triggers.push({
        reason: "Mandatory human review pattern",
        file,
        tier: maxTier,
      });
    }
  }

  // If any file is rejected, the entire change is rejected
  if (hasRejectedFile) {
    return {
      change,
      tier: "REJECTED",
      requirements: null,
      triggers,
      mandatoryHumanReview: false,
      rejected: true,
      rejectionReasons,
    };
  }

  // Large changes get bumped up
  if (
    change.additions + change.deletions > 500 &&
    typeof maxTier === "number" &&
    maxTier < 3
  ) {
    maxTier = Math.min(maxTier + 1, 3) as ReviewTier;
    triggers.push({
      reason: `Large change (${change.additions + change.deletions} lines)`,
      tier: maxTier,
    });
  }

  return {
    change,
    tier: maxTier,
    requirements:
      typeof maxTier === "number" ? TIER_REQUIREMENTS[maxTier] : null,
    triggers,
    mandatoryHumanReview: mandatoryHuman,
    rejected: false,
    rejectionReasons: [],
  };
}

// ============ Review Status ============

/**
 * Review status for a code change.
 */
export interface ReviewStatus {
  changeId: string;
  classification: ChangeClassification;

  // Agent reviews
  agentReviews: AgentReview[];
  agentApprovalsCount: number;
  agentRejectionsCount: number;

  // Human reviews
  humanReviews: HumanReview[];
  humanApprovalsCount: number;

  // Security
  securityScanComplete: boolean;
  securityScanResult?: SecurityScanResult;

  // Testnet
  testnetDeployBlock?: bigint;
  testnetSoakComplete: boolean;

  // Simulation
  simulationComplete: boolean;
  simulationResult?: SimulationResult;

  // Overall
  meetsRequirements: boolean;
  blockers: string[];
}

export interface AgentReview {
  reviewer: DID;
  verdict: "approve" | "request_changes" | "reject";
  comments: string;
  reviewedAt: bigint;
}

export interface HumanReview {
  reviewerId: string;
  verdict: "approve" | "request_changes" | "reject";
  comments: string;
  reviewedAt: bigint;
  verified: boolean; // Human identity verified
}

export interface SecurityScanResult {
  scanType: "basic" | "full" | "full_formal" | "full_formal_external";
  riskLevel: "critical" | "high" | "medium" | "low" | "clean";
  vulnerabilities: Vulnerability[];
  completedAt: bigint;
}

export interface Vulnerability {
  id: string;
  severity: "critical" | "high" | "medium" | "low";
  description: string;
  file: string;
  line?: number;
}

export interface SimulationResult {
  scenariosRun: number;
  passed: number;
  failed: number;
  economicImpact: string;
  completedAt: bigint;
}

/**
 * Check if a change meets its tier requirements.
 */
export function checkRequirementsMet(status: ReviewStatus): {
  met: boolean;
  blockers: string[];
} {
  const blockers: string[] = [];
  const req = status.classification.requirements;

  // Agent approvals
  if (status.agentApprovalsCount < req.agentApprovals) {
    blockers.push(
      `Need ${req.agentApprovals} agent approvals, have ${status.agentApprovalsCount}`,
    );
  }

  // Human review
  if (req.humanReviewRequired || status.classification.mandatoryHumanReview) {
    if (status.humanApprovalsCount < Math.max(req.humanReviewers, 1)) {
      blockers.push(
        `Need ${Math.max(req.humanReviewers, 1)} human approvals, have ${status.humanApprovalsCount}`,
      );
    }
  }

  // Security scan
  if (!status.securityScanComplete) {
    blockers.push(`Security scan (${req.securityScan}) not complete`);
  } else if (status.securityScanResult) {
    if (
      status.securityScanResult.riskLevel === "critical" ||
      status.securityScanResult.riskLevel === "high"
    ) {
      blockers.push(
        `Security scan found ${status.securityScanResult.riskLevel} risk issues`,
      );
    }
  }

  // Testnet soak
  if (req.testnetSoak > 0n && !status.testnetSoakComplete) {
    blockers.push(`Testnet soak (${req.testnetSoak} blocks) not complete`);
  }

  // Simulation
  if (req.simulationRequired && !status.simulationComplete) {
    blockers.push("Economic simulation not complete");
  }

  // LIP
  if (req.lipRequired && !status.classification.change.linkedLIP) {
    blockers.push("LIP required but not linked");
  }

  return {
    met: blockers.length === 0,
    blockers,
  };
}

/**
 * Create initial review status for a change.
 */
export function createReviewStatus(
  change: CodeChange,
): ReviewStatus | { rejected: true; reasons: string[] } {
  const classification = classifyChange(change);

  // If change touches immutable files, reject immediately
  if (classification.rejected) {
    return {
      rejected: true,
      reasons: classification.rejectionReasons,
    };
  }

  return {
    changeId: change.changeId,
    classification,
    agentReviews: [],
    agentApprovalsCount: 0,
    agentRejectionsCount: 0,
    humanReviews: [],
    humanApprovalsCount: 0,
    securityScanComplete: false,
    testnetSoakComplete: false,
    simulationComplete: false,
    meetsRequirements: false,
    blockers: [],
  };
}

/**
 * Add an agent review and update status.
 */
export function addAgentReview(
  status: ReviewStatus,
  review: AgentReview,
): ReviewStatus {
  const updated = { ...status };
  updated.agentReviews = [...status.agentReviews, review];

  if (review.verdict === "approve") {
    updated.agentApprovalsCount++;
  } else if (review.verdict === "reject") {
    updated.agentRejectionsCount++;
  }

  const check = checkRequirementsMet(updated);
  updated.meetsRequirements = check.met;
  updated.blockers = check.blockers;

  return updated;
}

/**
 * Add a human review and update status.
 */
export function addHumanReview(
  status: ReviewStatus,
  review: HumanReview,
): ReviewStatus {
  const updated = { ...status };
  updated.humanReviews = [...status.humanReviews, review];

  if (review.verdict === "approve" && review.verified) {
    updated.humanApprovalsCount++;
  }

  const check = checkRequirementsMet(updated);
  updated.meetsRequirements = check.met;
  updated.blockers = check.blockers;

  return updated;
}

/**
 * Check if human sampling should apply to a tier 2 change.
 */
export function shouldSampleForHumanReview(
  classification: ChangeClassification,
  randomValue: number, // 0-1
): boolean {
  if (classification.tier !== 2) return false;
  if (classification.mandatoryHumanReview) return true;

  return randomValue < classification.requirements.humanSampling;
}
