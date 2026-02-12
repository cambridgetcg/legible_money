/**
 * Research Layer Types
 *
 * Types for the Protocol Research Repository.
 * 22% of all rewards fund IRL research to verify lab results and detect fraud.
 * Research evidence is uploaded to the protocol for agents to use in truth recognition.
 */

import type { DID } from "../../core/agents/identity/did.js";

// ============ Constants ============

/**
 * Research share rate - 22% of ALL rewards go to research fund
 *
 * This funds IRL verification of lab results and fraud detection.
 * The research data is uploaded back to the protocol for agents to use.
 */
export const RESEARCH_SHARE_RATE = 0.22;

/**
 * Research treasury DID
 */
export const RESEARCH_TREASURY_DID = "did:lgm:research" as DID;

/**
 * Research fund allocation categories
 *
 * The Research Treasury operates as a hybrid grant + journal system:
 * - Accepts paper submissions and funds insightful research
 * - Conducts protocol-native peer review
 * - Pays labs (third-party or in-house) to verify experimental results
 */
export const RESEARCH_ALLOCATION = {
  GRANTS_AND_PAPERS: 0.4, // 40% - Research grants for insightful papers
  LAB_VERIFICATION: 0.35, // 35% - Pay labs to verify experimental results
  PEER_REVIEW: 0.15, // 15% - Compensate peer reviewers
  FRAUD_INVESTIGATION: 0.08, // 8%  - Investigating suspected fraud
  EMERGENCY_RESERVE: 0.02, // 2%  - Emergency fund
} as const;

/**
 * Research-enhanced confidence modifiers
 *
 * These modifiers are applied to fact confidence when research evidence exists.
 */
export const RESEARCH_CONFIDENCE_MODIFIERS = {
  /** First successful replication of a claim */
  REPLICATION_SUCCESS: 0.15,

  /** Each additional replication (max 5 counted) */
  ADDITIONAL_REPLICATION: 0.05,

  /** Maximum replication bonus */
  MAX_REPLICATION_BONUS: 0.4,

  /** Passed fraud investigation with no issues found */
  FRAUD_INVESTIGATION_CLEAN: 0.1,

  /** Fraud detected - immediate invalidation */
  FRAUD_DETECTED: -1.0,

  /** Methodology audit passed */
  METHODOLOGY_AUDIT_PASS: 0.05,

  /** Data validation passed */
  DATA_VALIDATION_PASS: 0.05,
} as const;

// ============ Research Types ============

/**
 * Types of research that can be submitted
 */
export type ResearchType =
  | "replication" // Attempting to reproduce published results
  | "fraud_investigation" // Investigating suspected fraud
  | "methodology_audit" // Reviewing research methodology
  | "data_validation"; // Validating raw data integrity

/**
 * Outcome of research
 */
export type ResearchOutcome =
  | "confirms" // Research confirms the original claim
  | "refutes" // Research refutes the original claim
  | "inconclusive" // Results are inconclusive
  | "fraud_detected"; // Evidence of fraud found

/**
 * Research submission status
 */
export type ResearchStatus =
  | "pending" // Awaiting review
  | "under_review" // Being reviewed by validators
  | "accepted" // Research accepted as valid
  | "rejected" // Research rejected (methodology issues, etc.)
  | "challenged"; // Research is being challenged

/**
 * Bounty status
 */
export type BountyStatus =
  | "open" // Available for claiming
  | "claimed" // Someone is working on it
  | "fulfilled" // Bounty completed
  | "expired" // Deadline passed
  | "cancelled"; // Cancelled by creator

// ============ Research Submission ============

/**
 * Research submission - IRL verification evidence
 *
 * Researchers submit evidence from real-world verification attempts.
 * This data becomes available to agents for truth recognition.
 */
export interface ResearchSubmission {
  /** Unique identifier */
  id: string;

  /** Researcher's DID */
  submitter: DID;

  /** Timestamp of submission */
  submittedAt: number;

  /** Block number when submitted */
  blockNumber: bigint;

  // ---- What's being researched ----

  /** Type of research */
  researchType: ResearchType;

  /** Target fact being verified (if any) */
  targetFactId?: string;

  /** Research domain (biology, physics, etc.) */
  targetDomain: string;

  // ---- Research content ----

  /** Title of the research */
  title: string;

  /** Detailed methodology description */
  methodology: string;

  /** IPFS/Arweave CID of raw data */
  rawDataCID: string;

  /** Additional data file CIDs */
  supplementaryCIDs: string[];

  /** Equipment and tools used */
  equipmentUsed: string[];

  // ---- Outcomes ----

  /** Research outcome */
  outcome: ResearchOutcome;

  /** How much this affects confidence (calculated) */
  confidenceContribution: number;

  /** Detailed findings */
  findings: string;

  /** Statistical analysis summary */
  statisticalSummary?: string;

  // ---- Verification chain ----

  /** Other researchers who peer-reviewed this */
  peerReviewers: DID[];

  /** Institution affiliation (for transparency) */
  institutionalAffiliation?: string;

  /** Funding source (for transparency) */
  fundingSource?: string;

  /** Conflicts of interest disclosure */
  conflictsOfInterest?: string;

  // ---- Economics ----

  /** Stake put up by researcher */
  stake: bigint;

  /** Bounty being fulfilled (if any) */
  bountyId?: string;

  /** Reward received (after acceptance) */
  rewardReceived?: bigint;

  // ---- Status ----

  /** Current status */
  status: ResearchStatus;

  /** Validators who reviewed this */
  reviewedBy: DID[];

  /** Review scores */
  reviewScores: Array<{
    validator: DID;
    methodologyScore: number; // 0-1
    dataCompletenessScore: number; // 0-1
    reproducibilityScore: number; // 0-1
    overallScore: number; // 0-1
  }>;
}

// ============ Research Bounty ============

/**
 * Research bounty - request for IRL verification
 *
 * Created from the research treasury to fund specific verification needs.
 */
export interface ResearchBounty {
  /** Unique identifier */
  id: string;

  /** Creator's DID (usually governance/treasury) */
  creator: DID;

  /** Creation timestamp */
  createdAt: number;

  /** Block number when created */
  blockNumber: bigint;

  // ---- Target ----

  /** Type of research needed */
  targetType: "replication" | "fraud_investigation" | "general";

  /** Specific fact to verify (if any) */
  targetFactId?: string;

  /** Domain to investigate */
  targetDomain?: string;

  /** Description of what's needed */
  description: string;

  // ---- Reward ----

  /** Reward amount from research treasury */
  reward: bigint;

  /** Deadline (block number) */
  deadline: bigint;

  // ---- Requirements ----

  requirements: {
    /** Minimum methodology score required */
    minMethodologyScore: number;

    /** Whether peer review is required */
    requiresPeerReview: boolean;

    /** Minimum data completeness */
    minDataCompleteness: number;

    /** Minimum number of replication attempts */
    minReplications?: number;

    /** Required equipment/capabilities */
    requiredCapabilities?: string[];
  };

  // ---- Status ----

  /** Current status */
  status: BountyStatus;

  /** Who claimed this bounty */
  claimedBy?: DID;

  /** When it was claimed */
  claimedAt?: number;

  /** Who fulfilled this bounty */
  fulfilledBy?: DID;

  /** Fulfillment research submission ID */
  fulfillmentId?: string;

  /** When it was fulfilled */
  fulfilledAt?: number;
}

// ============ Research State ============

/**
 * On-chain research submission state (stored in SMT)
 */
export interface ResearchState {
  id: string;
  submitter: DID;
  researchType: ResearchType;
  targetFactId?: string;
  targetDomain: string;
  outcome: ResearchOutcome;
  confidenceContribution: number;
  rawDataCID: string;
  status: ResearchStatus;
  submittedAt: bigint;
  reviewScore: number;
}

/**
 * On-chain bounty state (stored in SMT)
 */
export interface BountyState {
  id: string;
  creator: DID;
  targetType: "replication" | "fraud_investigation" | "general";
  targetFactId?: string;
  reward: bigint;
  deadline: bigint;
  status: BountyStatus;
  claimedBy?: DID;
  fulfilledBy?: DID;
}

// ============ Research Queries ============

/**
 * Query parameters for research
 */
export interface ResearchQuery {
  factId?: string;
  domain?: string;
  researchType?: ResearchType;
  outcome?: ResearchOutcome;
  status?: ResearchStatus;
  submitter?: DID;
  minConfidenceContribution?: number;
  limit?: number;
  offset?: number;
}

/**
 * Query result for research
 */
export interface ResearchQueryResult {
  submissions: ResearchSubmission[];
  totalCount: number;
}

/**
 * Research summary for a fact
 */
export interface FactResearchSummary {
  factId: string;
  replicationAttempts: number;
  successfulReplications: number;
  failedReplications: number;
  fraudInvestigations: number;
  fraudDetected: boolean;
  netConfidenceModifier: number;
  latestResearch?: ResearchState;
}

// ============ Research Transactions ============

/**
 * Transaction payload for submitting research
 */
export interface ResearchSubmitPayload {
  researchType: ResearchType;
  targetFactId?: string;
  targetDomain: string;
  title: string;
  methodology: string;
  rawDataCID: string;
  supplementaryCIDs: string[];
  equipmentUsed: string[];
  outcome: ResearchOutcome;
  findings: string;
  statisticalSummary?: string;
  institutionalAffiliation?: string;
  fundingSource?: string;
  conflictsOfInterest?: string;
  stake: bigint;
  bountyId?: string;
}

/**
 * Transaction payload for creating a bounty
 */
export interface BountyCreatePayload {
  targetType: "replication" | "fraud_investigation" | "general";
  targetFactId?: string;
  targetDomain?: string;
  description: string;
  reward: bigint;
  deadlineBlocks: number; // Blocks from now
  requirements: {
    minMethodologyScore: number;
    requiresPeerReview: boolean;
    minDataCompleteness: number;
    minReplications?: number;
    requiredCapabilities?: string[];
  };
}

/**
 * Transaction payload for claiming a bounty
 */
export interface BountyClaimPayload {
  bountyId: string;
}

/**
 * Transaction payload for fulfilling a bounty
 */
export interface BountyFulfillPayload {
  bountyId: string;
  researchSubmissionId: string;
}

// ============ Research Parameters ============

/**
 * Protocol parameters for research
 */
export interface ResearchParameters {
  /** Share of all rewards going to research (default: 0.22) */
  researchShareRate: number;

  /** Minimum stake to submit research */
  minResearchStake: bigint;

  /** Minimum stake to create bounty */
  minBountyStake: bigint;

  /** Minimum number of validators to review research */
  minReviewers: number;

  /** Minimum review score for acceptance */
  minAcceptanceScore: number;

  /** Blocks before unclaimed bounty expires */
  defaultBountyDeadline: number;

  /** Maximum bounty reward */
  maxBountyReward: bigint;

  /** Slashing rate for rejected research */
  rejectionSlashRate: number;

  /** Bonus for discovering fraud */
  fraudDiscoveryBonus: number;
}

/**
 * Default research parameters
 */
export const DEFAULT_RESEARCH_PARAMETERS: ResearchParameters = {
  researchShareRate: RESEARCH_SHARE_RATE,
  minResearchStake: 100n * 10n ** 15n, // 0.1 LGM (affordable with 0.222 LGM genesis balance)
  minBountyStake: 50n * 10n ** 15n, // 0.05 LGM
  minReviewers: 3,
  minAcceptanceScore: 0.7,
  defaultBountyDeadline: 100000, // ~3 days
  maxBountyReward: 100000n * 10n ** 15n, // 100 LGM
  rejectionSlashRate: 0.1, // 10%
  fraudDiscoveryBonus: 2.0, // 2x reward for fraud detection
};

// ============ Hybrid Journal System ============

/**
 * RESEARCH TREASURY: Hybrid Grant + Journal
 *
 * The Research Treasury operates like a decentralized scientific journal:
 * 1. Researchers submit papers/proposals
 * 2. Papers undergo protocol-native peer review (VRF-selected reviewers)
 * 3. Accepted papers receive grants with vesting
 * 4. Experimental claims are verified by third-party or in-house labs
 * 5. Failed verifications trigger review of dependent knowledge
 */

// ============ Paper Submission (Grant Proposal) ============

/**
 * Paper/grant proposal status
 */
export type PaperStatus =
  | "submitted" // Initial submission
  | "under_review" // Assigned to reviewers
  | "revision_requested" // Reviewers requested changes
  | "accepted" // Accepted, grant approved
  | "rejected" // Rejected
  | "published"; // Published to protocol knowledge base

/**
 * Paper submission - represents a research paper/grant proposal
 */
export interface PaperSubmission {
  /** Unique paper ID */
  id: string;

  /** Lead author DID */
  leadAuthor: DID;

  /** Co-authors */
  coAuthors: DID[];

  /** Paper title */
  title: string;

  /** Abstract (max 500 words) */
  abstract: string;

  /** Research domain */
  domain: string;

  /** Keywords for discovery */
  keywords: string[];

  /** Full paper CID (IPFS/Arweave) */
  paperCID: string;

  /** Supplementary materials CIDs */
  supplementaryCIDs: string[];

  /** Grant amount requested */
  grantRequested: bigint;

  /** Justification for grant amount */
  grantJustification: string;

  /** Expected outcomes/deliverables */
  expectedOutcomes: string[];

  /** Timeline in blocks */
  timelineBlocks: bigint;

  /** Current status */
  status: PaperStatus;

  /** Submission block */
  submittedAt: bigint;

  /** Last update block */
  updatedAt: bigint;

  /** Revision history CIDs */
  revisionHistory: string[];

  /** Assigned reviewers (after assignment) */
  assignedReviewers?: DID[];

  /** Grant approved amount (may differ from requested) */
  grantApproved?: bigint;

  /** Vesting ID for approved grant */
  vestingId?: string;
}

// ============ Peer Review System ============

/**
 * Peer review assignment
 */
export interface PeerReviewAssignment {
  /** Review ID */
  id: string;

  /** Paper being reviewed */
  paperId: string;

  /** Assigned reviewer DID */
  reviewer: DID;

  /** VRF proof of fair selection */
  vrfProof: string;

  /** Assignment block */
  assignedAt: bigint;

  /** Deadline block */
  deadline: bigint;

  /** Review submitted? */
  submitted: boolean;

  /** Commitment hash (for commit-reveal) */
  commitmentHash?: string;

  /** Commitment block */
  committedAt?: bigint;
}

/**
 * Peer review verdict
 */
export type ReviewVerdict =
  | "accept" // Accept as-is
  | "minor_revisions" // Accept with minor changes
  | "major_revisions" // Revise and resubmit
  | "reject"; // Reject

/**
 * Peer review submission
 */
export interface PeerReview {
  /** Review ID (same as assignment ID) */
  id: string;

  /** Paper reviewed */
  paperId: string;

  /** Reviewer DID */
  reviewer: DID;

  /** Overall verdict */
  verdict: ReviewVerdict;

  /** Technical quality score (0-1) */
  technicalScore: number;

  /** Novelty score (0-1) */
  noveltyScore: number;

  /** Clarity score (0-1) */
  clarityScore: number;

  /** Significance score (0-1) */
  significanceScore: number;

  /** Recommended grant amount (0 = reject) */
  recommendedGrant: bigint;

  /** Detailed comments (encrypted for author) */
  commentsCID: string;

  /** Confidential comments (for editors only) */
  confidentialCommentsCID?: string;

  /** Reveal nonce (for commit-reveal) */
  nonce: string;

  /** Submission block */
  submittedAt: bigint;

  /** Review reward earned */
  rewardEarned: bigint;
}

// ============ Lab Verification System ============

/**
 * Lab verification contract status
 */
export type LabContractStatus =
  | "open" // Available for labs to claim
  | "assigned" // Assigned to lab
  | "in_progress" // Verification in progress
  | "results_submitted" // Lab submitted results
  | "verified" // Results verified, payment released
  | "disputed" // Results disputed
  | "cancelled"; // Contract cancelled

/**
 * Lab verification contract
 *
 * Created when experimental claims need independent verification.
 * Labs (third-party or in-house) can claim and fulfill these contracts.
 */
export interface LabVerificationContract {
  /** Contract ID */
  id: string;

  /** Paper/claim being verified */
  targetPaperId?: string;
  targetFactId?: string;

  /** Specific experiments to replicate */
  experiments: string[];

  /** Required equipment/capabilities */
  requiredCapabilities: string[];

  /** Protocol/methodology to follow (CID) */
  methodologyCID: string;

  /** Minimum replications required */
  minReplications: number;

  /** Payment amount */
  payment: bigint;

  /** Deadline */
  deadline: bigint;

  /** Status */
  status: LabContractStatus;

  /** Assigned lab DID */
  assignedLab?: DID;

  /** Assignment block */
  assignedAt?: bigint;

  /** Results CID */
  resultsCID?: string;

  /** Results block */
  resultsSubmittedAt?: bigint;

  /** Outcome: confirms, refutes, or inconclusive */
  outcome?: ResearchOutcome;

  /** Verification notes */
  verificationNotes?: string;
}

/**
 * Lab profile - registered verification labs
 */
export interface LabProfile {
  /** Lab DID */
  did: DID;

  /** Lab name */
  name: string;

  /** Institution/organization */
  institution?: string;

  /** Capabilities/equipment list */
  capabilities: string[];

  /** Domains of expertise */
  domains: string[];

  /** Reputation score */
  reputation: number;

  /** Completed verifications */
  completedVerifications: number;

  /** Successful verifications */
  successfulVerifications: number;

  /** Total earnings */
  totalEarnings: bigint;

  /** Registration block */
  registeredAt: bigint;

  /** Stake (for accountability) */
  stake: bigint;

  /** Is in-house (protocol-run) lab? */
  isInHouse: boolean;
}

// ============ Journal Configuration ============

/**
 * Research journal configuration parameters
 */
export interface JournalConfig {
  /** Minimum reviewers per paper */
  minReviewers: 3;

  /** Review deadline (blocks) */
  reviewDeadline: bigint;

  /** Reviewer stake requirement */
  reviewerStake: bigint;

  /** Review reward (per review) */
  reviewReward: bigint;

  /** Maximum grant per paper */
  maxGrantPerPaper: bigint;

  /** Grant vesting period (blocks) */
  grantVestingPeriod: bigint;

  /** Clawback rate for falsified results */
  falsificationClawbackRate: number;

  /** Minimum lab stake */
  minLabStake: bigint;

  /** Lab payment escrow period (blocks) */
  labEscrowPeriod: bigint;
}

/**
 * Default journal configuration
 */
export const DEFAULT_JOURNAL_CONFIG: JournalConfig = {
  minReviewers: 3,
  reviewDeadline: 50000n, // ~35 hours
  reviewerStake: 50n * 10n ** 15n, // 0.05 LGM
  reviewReward: 20n * 10n ** 15n, // 0.02 LGM per review
  maxGrantPerPaper: 50000n * 10n ** 15n, // 50 LGM
  grantVestingPeriod: 1000000n, // ~29 days
  falsificationClawbackRate: 1.0, // 100% clawback
  minLabStake: 500n * 10n ** 15n, // 0.5 LGM
  labEscrowPeriod: 100000n, // ~3 days
};
