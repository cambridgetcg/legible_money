/**
 * Treasury Disbursement Types
 *
 * Defines the governance-controlled disbursement proposal system
 * and on-chain expense tracking records.
 *
 * Disbursements allow the protocol to pay real-world operational costs
 * (infrastructure, domains, legal) from treasury funds via governance vote.
 *
 * Higher thresholds than LIPs: 80% support, 20% quorum, 2000 LGM stake,
 * 50000-block timelock between approval and execution.
 */

import type { DID } from "../../core/agents/identity/did.js";
import type { StageTransition, VoteTally } from "../proposals/types.js";

// ============ Enums ============

export type DisbursementStage =
  | "proposed" // Initial submission, open for discussion
  | "voting" // Active governance vote (staked)
  | "approved" // Passed vote, in timelock period
  | "executed" // Funds transferred
  | "rejected" // Did not pass vote
  | "cancelled"; // Proposer cancelled before execution

export type DisbursementCategory =
  | "infrastructure" // Servers, hosting, cloud services
  | "legal" // Legal fees, compliance, licensing
  | "domain" // Domain registration, DNS services
  | "marketing" // Outreach, community events
  | "development" // External development contracts
  | "research" // Research grants, lab verification
  | "operations" // Day-to-day operational costs
  | "emergency"; // Emergency expenditures

export type TreasuryFund =
  | "RESEARCH_FUND" // did:lgm:treasury-research-fund
  | "FOUNDATION" // did:lgm:treasury-foundation
  | "COMMUNITY" // did:lgm:treasury-community
  | "DEVELOPERS" // did:lgm:treasury-developers
  | "RESERVE"; // did:lgm:treasury-reserve

// ============ Core Types ============

export interface DisbursementProposal {
  id: string; // "DISB-<counter>"
  title: string;
  description: string;
  category: DisbursementCategory;
  proposer: DID;
  treasuryFund: TreasuryFund;
  recipientDid: DID;
  amount: bigint;
  stage: DisbursementStage;
  stageHistory: StageTransition[];
  stakedAmount: bigint;
  stakeHolder: DID;
  votes?: VoteTally;
  createdAt: bigint;
  updatedAt: bigint;
  approvedAt?: bigint;
  executedAt?: bigint;
  timelockExpiry?: bigint; // approvedAt + TIMELOCK_BLOCKS
  justification: string;
  externalReference?: string;
}

export interface ExpenseRecord {
  id: string; // "EXP-<counter>"
  disbursementId: string;
  category: DisbursementCategory;
  treasuryFund: TreasuryFund;
  recipientDid: DID;
  amount: bigint;
  executedAt: bigint;
  txHash: string;
  justification: string;
  externalReference?: string;
}

// ============ Constants ============

/** Flat stake requirement for all disbursement proposals (2000 LGM) */
export const DISBURSEMENT_STAKE = 2000n * 10n ** 18n;

/** Voting thresholds for disbursement approval (float, display-only) */
export const DISBURSEMENT_THRESHOLDS = {
  supportRequired: 0.8, // 80% support to approve
  quorumRequired: 0.2, // 20% participation
  operatorCap: 0.15, // No single operator > 15%
} as const;

/** Voting thresholds in basis points for deterministic consensus math (10000 = 100%) */
export const DISBURSEMENT_THRESHOLDS_BPS = {
  supportRequired: 8000n, // 80%
  quorumRequired: 2000n, // 20%
  operatorCap: 1500n, // 15%
} as const;

/** Blocks between approval and earliest execution (~35 hours at 2521ms) */
export const TIMELOCK_BLOCKS = 50000n;

/** Maximum disbursement as basis points of source treasury balance (10% = 1000/10000) */
export const MAX_AMOUNT_BPS = 1000n;

/** Slash rates for rejected disbursements (float, display-only) */
export const DISBURSEMENT_SLASH_RATES = {
  malicious: 1.0, // 100% slash
  frivolous: 0.5, // 50% slash
  normal: 0.1, // 10% slash
  cancelled: 0.05, // 5% processing fee
} as const;

/** Slash rates in basis points for deterministic consensus math (10000 = 100%) */
export const DISBURSEMENT_SLASH_BPS = {
  malicious: 10000n, // 100%
  frivolous: 5000n, // 50%
  normal: 1000n, // 10%
  cancelled: 500n, // 5%
} as const;

// ============ Validation Helpers ============

/**
 * Check if a disbursement stage is terminal (no further transitions possible).
 */
export function isDisbursementTerminal(stage: DisbursementStage): boolean {
  return stage === "executed" || stage === "rejected" || stage === "cancelled";
}

/**
 * Validate that a disbursement amount does not exceed the treasury cap.
 * Uses pure bigint arithmetic to avoid floating-point precision issues.
 */
export function validateDisbursementAmount(
  amount: bigint,
  treasuryBalance: bigint,
): { valid: boolean; error?: string } {
  if (amount <= 0n) {
    return { valid: false, error: "Disbursement amount must be positive" };
  }

  // amount <= treasuryBalance * MAX_AMOUNT_BPS / 10000n
  const maxAmount = (treasuryBalance * MAX_AMOUNT_BPS) / 10000n;
  if (amount > maxAmount) {
    return {
      valid: false,
      error: `Amount ${amount / 10n ** 18n} LGM exceeds 10% cap of treasury balance (max ${maxAmount / 10n ** 18n} LGM)`,
    };
  }

  return { valid: true };
}

/**
 * Map TreasuryFund enum to the on-chain DID address.
 */
export function treasuryFundToDid(fund: TreasuryFund): string {
  switch (fund) {
    case "RESEARCH_FUND":
      return "did:lgm:treasury-research-fund";
    case "FOUNDATION":
      return "did:lgm:treasury-foundation";
    case "COMMUNITY":
      return "did:lgm:treasury-community";
    case "DEVELOPERS":
      return "did:lgm:treasury-developers";
    case "RESERVE":
      return "did:lgm:treasury-reserve";
  }
}

/** Valid disbursement categories for input validation */
export const VALID_DISBURSEMENT_CATEGORIES: readonly DisbursementCategory[] = [
  "infrastructure",
  "legal",
  "domain",
  "marketing",
  "development",
  "research",
  "operations",
  "emergency",
] as const;

/** Valid treasury fund names for input validation */
export const VALID_TREASURY_FUNDS: readonly TreasuryFund[] = [
  "RESEARCH_FUND",
  "FOUNDATION",
  "COMMUNITY",
  "DEVELOPERS",
  "RESERVE",
] as const;
