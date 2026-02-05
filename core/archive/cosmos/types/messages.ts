/**
 * Cosmos SDK Message Types
 *
 * Defines the Msg types for Legible Money transactions.
 * Each message maps to a transaction type in the current system.
 *
 * PP HIERARCHY:
 *   TRUTH: Messages encode intent with full type safety
 *   UNDERSTANDING: Clear action semantics
 *   BEAUTY: Consistent message patterns
 *   JUSTICE: Proper validation and authorization
 *   CREATIVITY: Extensible for future message types
 */

import type { Coin } from "./module.js";

// ============ Base Message Interface ============

/**
 * Base message interface (maps to Cosmos SDK Msg)
 */
export interface Msg {
  /** Message type URL (e.g., "/lgm.v1.MsgTransfer") */
  typeUrl: string;

  /** Signer addresses */
  getSigners(): string[];

  /** Validate basic message structure */
  validateBasic(): void;
}

// ============ LGM Module Messages ============

/**
 * Transfer tokens
 */
export interface MsgTransfer extends Msg {
  typeUrl: "/lgm.v1.MsgTransfer";
  fromAddress: string;
  toAddress: string;
  amount: Coin[];
  memo?: string;
}

/**
 * Stake tokens
 */
export interface MsgStake extends Msg {
  typeUrl: "/lgm.v1.MsgStake";
  delegatorAddress: string;
  validatorAddress: string;
  amount: Coin;
}

/**
 * Unstake tokens
 */
export interface MsgUnstake extends Msg {
  typeUrl: "/lgm.v1.MsgUnstake";
  delegatorAddress: string;
  validatorAddress: string;
  amount: Coin;
}

/**
 * Register as validator
 */
export interface MsgRegisterValidator extends Msg {
  typeUrl: "/lgm.v1.MsgRegisterValidator";
  validatorAddress: string;
  pubKey: string;
  selfDelegation: Coin;
  commission: number;
  description: ValidatorDescription;
}

export interface ValidatorDescription {
  moniker: string;
  identity?: string;
  website?: string;
  details?: string;
}

/**
 * Create a session key
 */
export interface MsgCreateSession extends Msg {
  typeUrl: "/lgm.v1.MsgCreateSession";
  ownerAddress: string;
  sessionPubKey: string;
  capabilities: SessionCapabilities;
  expiresAtHeight: bigint;
}

export interface SessionCapabilities {
  maxSpendPerTx: bigint;
  maxSpendPerDay: bigint;
  allowedActions: string[];
}

// ============ Knowledge Module Messages ============

/**
 * Submit a claim for verification
 */
export interface MsgSubmitClaim extends Msg {
  typeUrl: "/lgm.knowledge.v1.MsgSubmitClaim";
  submitter: string;
  claim: string;
  domain: string;
  stake: Coin;
  references: string[];
  evidence?: ClaimEvidence[];
}

export interface ClaimEvidence {
  type: string;
  uri: string;
  hash: string;
}

/**
 * Submit verification commitment (phase 1)
 */
export interface MsgSubmitCommitment extends Msg {
  typeUrl: "/lgm.knowledge.v1.MsgSubmitCommitment";
  validator: string;
  roundId: string;
  commitment: string; // Hash of verdict + nonce
}

/**
 * Submit verification reveal (phase 2)
 */
export interface MsgSubmitReveal extends Msg {
  typeUrl: "/lgm.knowledge.v1.MsgSubmitReveal";
  validator: string;
  roundId: string;
  verdict: "confirmed" | "rejected" | "uncertain";
  confidence: number;
  nonce: string;
  reasoning?: string;
}

/**
 * Challenge an existing fact
 */
export interface MsgChallengeFact extends Msg {
  typeUrl: "/lgm.knowledge.v1.MsgChallengeFact";
  challenger: string;
  factId: string;
  reason: string;
  stake: Coin;
  counterEvidence: ClaimEvidence[];
}

// ============ Research Module Messages ============

/**
 * Create a research bounty
 */
export interface MsgCreateBounty extends Msg {
  typeUrl: "/lgm.research.v1.MsgCreateBounty";
  creator: string;
  title: string;
  description: string;
  reward: Coin;
  deadline: bigint;
  requirements: BountyRequirements;
}

export interface BountyRequirements {
  minEvidenceCount: number;
  requiredDomains: string[];
  minConfidence: number;
}

/**
 * Submit bounty claim
 */
export interface MsgClaimBounty extends Msg {
  typeUrl: "/lgm.research.v1.MsgClaimBounty";
  claimer: string;
  bountyId: string;
  factIds: string[];
  evidence: ClaimEvidence[];
}

/**
 * Fund research treasury
 */
export interface MsgFundResearch extends Msg {
  typeUrl: "/lgm.research.v1.MsgFundResearch";
  funder: string;
  amount: Coin;
}

// ============ Evidence Module Messages ============

/**
 * Submit evidence
 */
export interface MsgSubmitEvidence extends Msg {
  typeUrl: "/lgm.evidence.v1.MsgSubmitEvidence";
  submitter: string;
  evidenceType: string;
  contentHash: string;
  uri: string;
  metadata: Record<string, string>;
}

/**
 * Archive evidence
 */
export interface MsgArchiveEvidence extends Msg {
  typeUrl: "/lgm.evidence.v1.MsgArchiveEvidence";
  evidenceId: string;
  archiveUri: string;
  archiveType: "ipfs" | "arweave" | "wayback";
}

// ============ Dispute Module Messages ============

/**
 * Initiate a dispute
 */
export interface MsgInitiateDispute extends Msg {
  typeUrl: "/lgm.dispute.v1.MsgInitiateDispute";
  challenger: string;
  targetType: "fact" | "evidence" | "validator";
  targetId: string;
  reason: string;
  stake: Coin;
  evidence: ClaimEvidence[];
}

/**
 * Submit dispute vote
 */
export interface MsgDisputeVote extends Msg {
  typeUrl: "/lgm.dispute.v1.MsgDisputeVote";
  arbitrator: string;
  disputeId: string;
  vote: "challenger" | "defender" | "abstain";
  reasoning: string;
}

/**
 * Escalate dispute
 */
export interface MsgEscalateDispute extends Msg {
  typeUrl: "/lgm.dispute.v1.MsgEscalateDispute";
  disputeId: string;
  requester: string;
  additionalStake: Coin;
}

// ============ Vesting Module Messages ============

/**
 * Claim vested rewards
 */
export interface MsgClaimRewards extends Msg {
  typeUrl: "/lgm.vesting.v1.MsgClaimRewards";
  claimer: string;
  vestingIds: string[];
}

// ============ Governance Messages ============

/**
 * Submit governance proposal
 */
export interface MsgSubmitProposal extends Msg {
  typeUrl: "/lgm.gov.v1.MsgSubmitProposal";
  proposer: string;
  content: ProposalContent;
  initialDeposit: Coin[];
}

export type ProposalContent =
  | ParamChangeProposal
  | SoftwareUpgradeProposal
  | KnowledgeProposal;

export interface ParamChangeProposal {
  type: "param_change";
  title: string;
  description: string;
  changes: ParamChange[];
}

export interface ParamChange {
  subspace: string;
  key: string;
  value: string;
}

export interface SoftwareUpgradeProposal {
  type: "software_upgrade";
  title: string;
  description: string;
  plan: UpgradePlan;
}

export interface UpgradePlan {
  name: string;
  height: bigint;
  info: string;
}

export interface KnowledgeProposal {
  type: "knowledge";
  title: string;
  description: string;
  action: "add_axiom" | "remove_fact" | "update_domain";
  targetId?: string;
  newValue?: string;
}

// ============ Message Type Registry ============

export const MESSAGE_TYPES = {
  // LGM
  TRANSFER: "/lgm.v1.MsgTransfer",
  STAKE: "/lgm.v1.MsgStake",
  UNSTAKE: "/lgm.v1.MsgUnstake",
  REGISTER_VALIDATOR: "/lgm.v1.MsgRegisterValidator",
  CREATE_SESSION: "/lgm.v1.MsgCreateSession",

  // Knowledge
  SUBMIT_CLAIM: "/lgm.knowledge.v1.MsgSubmitClaim",
  SUBMIT_COMMITMENT: "/lgm.knowledge.v1.MsgSubmitCommitment",
  SUBMIT_REVEAL: "/lgm.knowledge.v1.MsgSubmitReveal",
  CHALLENGE_FACT: "/lgm.knowledge.v1.MsgChallengeFact",

  // Research
  CREATE_BOUNTY: "/lgm.research.v1.MsgCreateBounty",
  CLAIM_BOUNTY: "/lgm.research.v1.MsgClaimBounty",
  FUND_RESEARCH: "/lgm.research.v1.MsgFundResearch",

  // Evidence
  SUBMIT_EVIDENCE: "/lgm.evidence.v1.MsgSubmitEvidence",
  ARCHIVE_EVIDENCE: "/lgm.evidence.v1.MsgArchiveEvidence",

  // Dispute
  INITIATE_DISPUTE: "/lgm.dispute.v1.MsgInitiateDispute",
  DISPUTE_VOTE: "/lgm.dispute.v1.MsgDisputeVote",
  ESCALATE_DISPUTE: "/lgm.dispute.v1.MsgEscalateDispute",

  // Vesting
  CLAIM_REWARDS: "/lgm.vesting.v1.MsgClaimRewards",

  // Governance
  SUBMIT_PROPOSAL: "/lgm.gov.v1.MsgSubmitProposal",
} as const;

// ============ Message Responses ============

export interface MsgTransferResponse {
  // Empty for transfers
}

export interface MsgSubmitClaimResponse {
  claimId: string;
  roundId: string;
  estimatedCompletionHeight: bigint;
}

export interface MsgSubmitCommitmentResponse {
  commitmentHash: string;
}

export interface MsgSubmitRevealResponse {
  accepted: boolean;
}

export interface MsgCreateBountyResponse {
  bountyId: string;
}

export interface MsgInitiateDisputeResponse {
  disputeId: string;
  phase: string;
  deadline: bigint;
}

export interface MsgClaimRewardsResponse {
  totalClaimed: Coin;
  vestingsClaimed: number;
}

// ============ Message Validation Helpers ============

/**
 * Validate a Cosmos address
 */
export function validateAddress(address: string): void {
  if (!address || !address.startsWith("lgm1")) {
    throw new Error(`Invalid address: ${address}`);
  }
  if (address.length !== 43) {
    // lgm1 + 39 chars
    throw new Error(`Invalid address length: ${address}`);
  }
}

/**
 * Validate coin amount
 */
export function validateCoin(coin: Coin): void {
  if (!coin.denom) {
    throw new Error("Coin denom required");
  }
  if (coin.amount <= 0n) {
    throw new Error("Coin amount must be positive");
  }
}

/**
 * Validate claim content
 */
export function validateClaim(claim: string, maxLength: number = 10000): void {
  if (!claim || claim.trim().length === 0) {
    throw new Error("Claim cannot be empty");
  }
  if (claim.length > maxLength) {
    throw new Error(`Claim exceeds maximum length of ${maxLength}`);
  }
}
