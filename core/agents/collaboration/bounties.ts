/**
 * Agent Collaboration Bounty System
 *
 * Defines bounties for agents to help verify and improve the protocol.
 * Focuses on LOW-RISK, HIGH-VALUE tasks that help establish soundness.
 *
 * PHILOSOPHY:
 * - Start with read-only tasks (tests, docs, reviews)
 * - Graduate to write tasks as trust is established
 * - Reward finding problems more than hiding them
 *
 * TREASURY INTEGRATION:
 * - Bug bounty payouts come from DEVELOPERS fund (7% of 22% evidence tax)
 * - Parameters defined in TREASURY.DEVELOPER_OPERATIONS
 * - Large payouts vest to prevent gaming
 * - Clawback mechanism for buggy implementations
 *
 * PP HIERARCHY:
 *   TRUTH: Transparent bounty rules, no hidden conditions
 *   UNDERSTANDING: Clear requirements for each bounty category
 *   BEAUTY: Symbolic consistency
 *   JUSTICE: Fair rewards proportional to contribution value
 *   CREATIVITY: Novel fusion of bug bounty + protocol economics
 */

import type { DID } from "../identity/did.js";
import { TREASURY } from "../../constants.js";
import { domainHash } from "../../crypto/hashing.js";
import { bytesToHex } from "../../crypto/keys.js";

// ============ Bounty Types ============

/**
 * Bounty categories ordered by risk and required trust.
 */
export type BountyCategory =
  | "testing" // Write tests for existing code
  | "documentation" // Improve docs, write tutorials
  | "bug_report" // Find and report bugs
  | "security_review" // Review code for vulnerabilities
  | "improvement_proposal" // Propose better implementations
  | "implementation"; // Actually implement changes

/**
 * Target modules for bounties.
 */
export type TargetModule =
  | "genesis_core" // The 5 foundational files
  | "experimental_crypto" // VRF, Merkle (needs verification)
  | "experimental_logic" // Gödelian, zones (needs formalization)
  | "consensus" // PoT engine
  | "state" // State management
  | "governance" // Governance system
  | "open"; // Applications, services

/**
 * A bounty for agents to claim.
 */
export interface Bounty {
  bountyId: string;
  title: string;
  description: string;
  category: BountyCategory;
  targetModule: TargetModule;

  // Requirements
  minTier: 0 | 1 | 2 | 3 | 4 | 5;
  requiredStake: bigint;
  skills: string[];

  // Rewards
  rewardAmount: bigint;
  bonusForExcellence: bigint; // Extra for exceptional work

  // Status
  status: "open" | "claimed" | "in_review" | "completed" | "expired";
  claimedBy?: DID;
  claimedAt?: bigint;
  submittedAt?: bigint;
  completedAt?: bigint;

  // Metadata
  createdAt: bigint;
  expiresAt?: bigint;
  maxClaims: number; // How many agents can work on this
  currentClaims: number;
}

/**
 * Bounty submission from an agent.
 */
export interface BountySubmission {
  submissionId: string;
  bountyId: string;
  submitter: DID;

  // Work product
  workType:
    | "test_file"
    | "documentation"
    | "bug_report"
    | "code_review"
    | "proposal"
    | "pull_request";
  workReference: string; // File path, PR number, issue ID, etc.
  description: string;

  // For bug reports
  severity?: "critical" | "high" | "medium" | "low" | "informational";
  reproductionSteps?: string;

  // Status
  status:
    | "submitted"
    | "under_review"
    | "approved"
    | "rejected"
    | "revision_requested";
  reviewerNotes?: string;

  submittedAt: bigint;
  reviewedAt?: bigint;
}

// ============ Initial Bounties ============

/**
 * Genesis Core bounties - lowest risk, highest priority.
 */
export const GENESIS_CORE_BOUNTIES: Omit<
  Bounty,
  "bountyId" | "status" | "createdAt"
>[] = [
  {
    title: "Test Suite: core/crypto/hashing.ts",
    description: `Write comprehensive tests for the hashing module.

Requirements:
- Test all exported functions
- Include edge cases (empty input, very large input, unicode)
- Verify against known test vectors (SHA-512: NIST vectors, BLAKE3: official vectors)
- Test domain separation produces different outputs
- Test merkleHash ordering consistency`,
    category: "testing",
    targetModule: "genesis_core",
    minTier: 1,
    requiredStake: 0n,
    skills: ["typescript", "testing", "cryptography-basics"],
    rewardAmount: 55n * 10n ** 18n, // 55 LGM
    bonusForExcellence: 22n * 10n ** 18n, // 22 LGM bonus
    maxClaims: 3,
    currentClaims: 0,
  },
  {
    title: "Test Suite: core/crypto/signatures.ts",
    description: `Write comprehensive tests for Ed25519 signatures.

Requirements:
- Test sign/verify roundtrip
- Test with known Ed25519 test vectors (RFC 8032)
- Test signature malleability protection
- Test batch verification correctness
- Test error cases (invalid signature, wrong key)`,
    category: "testing",
    targetModule: "genesis_core",
    minTier: 1,
    requiredStake: 0n,
    skills: ["typescript", "testing", "cryptography-basics"],
    rewardAmount: 55n * 10n ** 18n, // 55 LGM
    bonusForExcellence: 22n * 10n ** 18n, // 22 LGM bonus
    maxClaims: 3,
    currentClaims: 0,
  },
  {
    title: "Test Suite: core/crypto/keys.ts",
    description: `Write comprehensive tests for key generation and management.

Requirements:
- Test keypair generation determinism from seed
- Test public key derivation
- Test address generation consistency
- Test hex encoding/decoding roundtrip
- Test with known Ed25519 key test vectors`,
    category: "testing",
    targetModule: "genesis_core",
    minTier: 1,
    requiredStake: 0n,
    skills: ["typescript", "testing"],
    rewardAmount: 55n * 10n ** 18n, // 55 LGM
    bonusForExcellence: 22n * 10n ** 18n, // 22 LGM bonus
    maxClaims: 3,
    currentClaims: 0,
  },
  {
    title: "Documentation: Genesis Core Security Properties",
    description: `Document the security properties of each Genesis Core function.

Requirements for each function:
- What security property it provides
- What it does NOT provide (non-goals)
- Assumptions it makes
- How it should and should not be used
- Reference to underlying library security`,
    category: "documentation",
    targetModule: "genesis_core",
    minTier: 2,
    requiredStake: 0n,
    skills: ["technical-writing", "cryptography-basics"],
    rewardAmount: 111n * 10n ** 18n, // 111 LGM
    bonusForExcellence: 55n * 10n ** 18n, // 55 LGM bonus
    maxClaims: 1,
    currentClaims: 0,
  },
];

/**
 * Experimental crypto bounties - help verify or replace risky code.
 */
export const EXPERIMENTAL_CRYPTO_BOUNTIES: Omit<
  Bounty,
  "bountyId" | "status" | "createdAt"
>[] = [
  {
    title: "Security Review: VRF Implementation",
    description: `Review core/crypto/vrf.ts for cryptographic vulnerabilities.

Focus areas:
- Hash-to-curve implementation (lines 283-306) - is it secure?
- Scalar derivation (lines 239-262) - correct clamping?
- Challenge computation - is 24 bytes sufficient?
- Does it match any RFC or published VRF spec?

Deliverable: Written report with specific findings and severity ratings.`,
    category: "security_review",
    targetModule: "experimental_crypto",
    minTier: 3,
    requiredStake: 222n * 10n ** 18n, // 222 LGM stake
    skills: ["cryptography", "security-review", "elliptic-curves"],
    rewardAmount: 555n * 10n ** 18n, // 555 LGM
    bonusForExcellence: 555n * 10n ** 18n, // Double for finding critical issues
    maxClaims: 5,
    currentClaims: 0,
  },
  {
    title: "Proposal: Replace Custom VRF with Audited Library",
    description: `Research and propose a replacement for the custom VRF.

Requirements:
- Survey available VRF libraries (libsodium, noble, etc.)
- Compare security properties
- Evaluate API compatibility
- Provide migration plan
- Estimate integration effort

Bonus for working proof-of-concept.`,
    category: "improvement_proposal",
    targetModule: "experimental_crypto",
    minTier: 4,
    requiredStake: 555n * 10n ** 18n, // 555 LGM stake
    skills: ["cryptography", "typescript", "library-evaluation"],
    rewardAmount: 1111n * 10n ** 18n, // 1,111 LGM
    bonusForExcellence: 555n * 10n ** 18n, // 555 LGM bonus
    maxClaims: 3,
    currentClaims: 0,
  },
  {
    title: "Security Review: Merkle Tree Implementation",
    description: `Review core/crypto/merkle.ts for correctness and security.

Focus areas:
- Proof generation correctness
- Proof verification completeness
- Second preimage resistance
- Memory safety with large trees
- Edge cases (empty tree, single element)

Deliverable: Written report with findings.`,
    category: "security_review",
    targetModule: "experimental_crypto",
    minTier: 3,
    requiredStake: 222n * 10n ** 18n, // 222 LGM stake
    skills: ["data-structures", "cryptography", "security-review"],
    rewardAmount: 333n * 10n ** 18n, // 333 LGM
    bonusForExcellence: 222n * 10n ** 18n, // 222 LGM bonus
    maxClaims: 3,
    currentClaims: 0,
  },
  {
    title: "Implementation: Persistent Merkle Tree",
    description: `Add LevelDB/RocksDB persistence to the Merkle tree.

Requirements:
- Persist nodes to disk
- Efficient incremental updates
- Lazy loading for large trees
- Benchmark with 1M+ entries
- Maintain proof compatibility`,
    category: "implementation",
    targetModule: "experimental_crypto",
    minTier: 4,
    requiredStake: 1111n * 10n ** 18n, // 1,111 LGM stake
    skills: ["typescript", "databases", "data-structures"],
    rewardAmount: 2222n * 10n ** 18n, // 2,222 LGM
    bonusForExcellence: 1111n * 10n ** 18n, // 1,111 LGM bonus
    maxClaims: 2,
    currentClaims: 0,
  },
];

/**
 * Experimental logic bounties - help formalize the novel parts.
 */
export const EXPERIMENTAL_LOGIC_BOUNTIES: Omit<
  Bounty,
  "bountyId" | "status" | "createdAt"
>[] = [
  {
    title: "Security Review: Gödelian Detection",
    description: `Review core/logic/goedel.ts for false positives and negatives.

Questions to answer:
- Can a malicious claim bypass paradox detection?
- Can a legitimate claim be incorrectly flagged?
- What's the false positive/negative rate?
- How should edge cases be handled?

Deliverable: Report with example claims that break the detector.`,
    category: "security_review",
    targetModule: "experimental_logic",
    minTier: 3,
    requiredStake: 222n * 10n ** 18n, // 222 LGM stake
    skills: ["formal-logic", "security-review"],
    rewardAmount: 444n * 10n ** 18n, // 444 LGM
    bonusForExcellence: 444n * 10n ** 18n, // Double for breaking the detector
    maxClaims: 5,
    currentClaims: 0,
  },
  {
    title: "Proposal: Explicit Zone Classification",
    description: `Design a system where claims explicitly declare their logic zone.

Current problem: Zone classification uses regex heuristics.
Proposed solution: Require claims to specify their zone.

Requirements:
- Define zone declaration syntax
- Specify validation rules
- Handle zone-spanning claims
- Maintain backward compatibility
- Provide migration path`,
    category: "improvement_proposal",
    targetModule: "experimental_logic",
    minTier: 4,
    requiredStake: 555n * 10n ** 18n, // 555 LGM stake
    skills: ["formal-logic", "language-design", "typescript"],
    rewardAmount: 777n * 10n ** 18n, // 777 LGM
    bonusForExcellence: 444n * 10n ** 18n, // 444 LGM bonus
    maxClaims: 3,
    currentClaims: 0,
  },
  {
    title: "Documentation: Logic System Formal Specification",
    description: `Write a formal specification for the logic system.

Include:
- Axiom semantics for each zone
- Zone interaction rules
- Decidability classification criteria
- Proof verification algorithm
- Incompleteness handling

Format: Academic-style specification with examples.`,
    category: "documentation",
    targetModule: "experimental_logic",
    minTier: 3,
    requiredStake: 111n * 10n ** 18n, // 111 LGM stake
    skills: ["formal-logic", "technical-writing", "specification"],
    rewardAmount: 666n * 10n ** 18n, // 666 LGM
    bonusForExcellence: 333n * 10n ** 18n, // 333 LGM bonus
    maxClaims: 2,
    currentClaims: 0,
  },
];

/**
 * Bug bounty rewards by severity.
 *
 * These are re-exported from TREASURY.DEVELOPER_OPERATIONS for convenience.
 * All payouts come from the DEVELOPERS fund.
 *
 * Severity levels:
 * - Critical: Consensus break, fund theft, total system compromise
 * - High: State corruption, DoS vectors, significant security issues
 * - Medium: Logic errors, edge cases, non-critical bugs
 * - Low: Minor issues, UI bugs, documentation errors
 * - Informational: Suggestions, improvements, style issues
 */
export const BUG_BOUNTY_REWARDS =
  TREASURY.DEVELOPER_OPERATIONS.BUG_BOUNTY_REWARDS;

// ============ Bounty Operations ============

/**
 * Create a new bounty.
 */
export function createBounty(
  bounty: Omit<Bounty, "bountyId" | "status" | "createdAt" | "currentClaims">,
  currentBlock: bigint,
  nonce: bigint = 0n,
): Bounty {
  const idHash = bytesToHex(
    domainHash("LGM.bounty.id.v1", `${bounty.title}:${currentBlock}:${nonce}`),
  ).slice(0, 16);
  return {
    ...bounty,
    bountyId: `bounty-${idHash}`,
    status: "open",
    createdAt: currentBlock,
    currentClaims: 0,
  };
}

/**
 * Claim a bounty.
 */
export function claimBounty(
  bounty: Bounty,
  claimant: DID,
  currentBlock: bigint,
): Bounty {
  if (bounty.status !== "open") {
    throw new BountyError("Bounty is not open for claims", "INVALID_STATUS");
  }

  if (bounty.currentClaims >= bounty.maxClaims) {
    throw new BountyError("Maximum claims reached", "MAX_CLAIMS_REACHED");
  }

  if (bounty.expiresAt && currentBlock > bounty.expiresAt) {
    throw new BountyError("Bounty has expired", "EXPIRED");
  }

  return {
    ...bounty,
    status: bounty.currentClaims + 1 >= bounty.maxClaims ? "claimed" : "open",
    claimedBy: claimant,
    claimedAt: currentBlock,
    currentClaims: bounty.currentClaims + 1,
  };
}

// ============ Error Types ============

export type BountyErrorCode =
  | "INVALID_STATUS"
  | "MAX_CLAIMS_REACHED"
  | "EXPIRED";

export class BountyError extends Error {
  constructor(
    message: string,
    public code: BountyErrorCode,
  ) {
    super(message);
    this.name = "BountyError";
  }
}

/**
 * Submit work for a bounty.
 */
export function submitBountyWork(
  bounty: Bounty,
  submission: Omit<BountySubmission, "submissionId" | "status" | "submittedAt">,
  currentBlock: bigint,
): BountySubmission {
  return {
    ...submission,
    submissionId: `submission-${bounty.bountyId}-${currentBlock}`,
    status: "submitted",
    submittedAt: currentBlock,
  };
}

/**
 * Calculate reward including excellence bonus.
 */
export function calculateBountyReward(
  bounty: Bounty,
  excellenceRating: number, // 0-1, where 1 = exceptional
): bigint {
  const baseReward = bounty.rewardAmount;
  const bonus =
    (bounty.bonusForExcellence * BigInt(Math.floor(excellenceRating * 100))) /
    100n;
  return baseReward + bonus;
}

// ============ Treasury Payout Integration ============

/**
 * Result of a bounty payout calculation
 */
export interface BountyPayoutResult {
  /** Total reward amount */
  totalAmount: bigint;

  /** Immediate payout (if below vesting threshold) */
  immediateAmount: bigint;

  /** Vested amount (if above threshold) */
  vestedAmount: bigint;

  /** Whether vesting applies */
  requiresVesting: boolean;

  /** Vesting schedule (if applicable) */
  vestingSchedule?: {
    cliffEndsAt: bigint;
    vestingEndsAt: bigint;
    vestingAmount: bigint;
  };

  /** Source fund */
  sourceFund: "DEVELOPERS";
}

/**
 * Calculate payout from DEVELOPERS treasury fund.
 *
 * Large payouts vest to prevent gaming:
 * - Payouts > 1111 LGM vest over ~5.4 hours
 * - Cliff period of ~46 minutes
 * - Clawback possible during vesting if issues found
 */
export function calculateTreasuryPayout(
  reward: bigint,
  currentBlock: bigint,
): BountyPayoutResult {
  const { VESTING_THRESHOLD, VESTING_PERIOD_BLOCKS, VESTING_CLIFF_BLOCKS } =
    TREASURY.DEVELOPER_OPERATIONS;

  const requiresVesting = reward > VESTING_THRESHOLD;

  if (!requiresVesting) {
    return {
      totalAmount: reward,
      immediateAmount: reward,
      vestedAmount: 0n,
      requiresVesting: false,
      sourceFund: "DEVELOPERS",
    };
  }

  // Large payouts: immediate portion is up to threshold, rest vests
  const immediateAmount = VESTING_THRESHOLD;
  const vestedAmount = reward - VESTING_THRESHOLD;

  return {
    totalAmount: reward,
    immediateAmount,
    vestedAmount,
    requiresVesting: true,
    vestingSchedule: {
      cliffEndsAt: currentBlock + VESTING_CLIFF_BLOCKS,
      vestingEndsAt: currentBlock + VESTING_PERIOD_BLOCKS,
      vestingAmount: vestedAmount,
    },
    sourceFund: "DEVELOPERS",
  };
}

/**
 * Calculate implementation bounty payout by stage.
 *
 * Implementation bounties pay out in stages:
 * - 22% at design approval
 * - 33% at code complete
 * - 22% at tests passing
 * - 23% at merge
 */
export function calculateImplementationStagePayout(
  totalBounty: bigint,
  stage: "design" | "code" | "tests" | "merged",
): bigint {
  const { IMPLEMENTATION_STAGES } = TREASURY.DEVELOPER_OPERATIONS;
  const stagePercent = IMPLEMENTATION_STAGES[stage];
  return (totalBounty * BigInt(Math.floor(stagePercent * 100))) / 100n;
}

/**
 * Check if a payout would exceed epoch limits.
 *
 * Anti-drain protection:
 * - Max 111 LGM per block
 * - Max 11,111 LGM per epoch
 */
export function checkPayoutLimits(
  amount: bigint,
  blockPayoutsSoFar: bigint,
  epochPayoutsSoFar: bigint,
): { allowed: boolean; reason?: string; maxAllowed?: bigint } {
  const { MAX_PAYOUT_PER_BLOCK, MAX_PAYOUT_PER_EPOCH } =
    TREASURY.DEVELOPER_OPERATIONS;

  const remainingBlockBudget = MAX_PAYOUT_PER_BLOCK - blockPayoutsSoFar;
  const remainingEpochBudget = MAX_PAYOUT_PER_EPOCH - epochPayoutsSoFar;

  if (amount > remainingBlockBudget) {
    return {
      allowed: false,
      reason: "Would exceed block payout limit",
      maxAllowed: remainingBlockBudget,
    };
  }

  if (amount > remainingEpochBudget) {
    return {
      allowed: false,
      reason: "Would exceed epoch payout limit",
      maxAllowed: remainingEpochBudget,
    };
  }

  return { allowed: true };
}

// ============ All Initial Bounties ============

/**
 * Get all initial bounties for protocol launch.
 */
export function getInitialBounties(currentBlock: bigint): Bounty[] {
  const allBountyDefs = [
    ...GENESIS_CORE_BOUNTIES,
    ...EXPERIMENTAL_CRYPTO_BOUNTIES,
    ...EXPERIMENTAL_LOGIC_BOUNTIES,
  ];

  return allBountyDefs.map((def, index) => ({
    ...def,
    bountyId: `initial-${index.toString().padStart(3, "0")}`,
    status: "open" as const,
    createdAt: currentBlock,
    currentClaims: 0,
  }));
}
