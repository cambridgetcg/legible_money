/**
 * Protocol Constants
 *
 * Core constants for the Legible Money protocol.
 */

// ============ Token Constants ============

export const TOKEN = {
  NAME: "Legible Money",
  SYMBOL: "LGM",
  DECIMALS: 18,
  MAX_SUPPLY: 222_222_222n * 10n ** 18n,
} as const;

// ============ Protocol Version ============

/**
 * Protocol version components
 */
export const VERSION = {
  MAJOR: 89,
  MINOR: 85,
  PATCH: 65,
  BUILD: 73,
} as const;

// ============ Timing Constants ============

/**
 * Timing constants optimized for Proof of Truth verification
 *
 * Finality is derived from schema timing requirements:
 * - Fast schemas (hash): 3 blocks (~8s)
 * - Standard schemas (oracle): 6 blocks (~15s)
 * - Complex schemas (research): 20 blocks (~50s)
 *
 * CONFIRMATION_BLOCKS: Soft confirmation after 1 standard round + buffer
 * FINALITY_BLOCKS: Hard finality after ~4 standard rounds
 */
export const TIMING = {
  TARGET_BLOCK_MS: 2521,
  CONFIRMATION_BLOCKS: 11,
  FINALITY_BLOCKS: 44,
} as const;

// ============ Consensus Constants ============

/**
 * Base consensus parameters
 */
export const CONSENSUS = {
  MIN_VALIDATORS: 5,
  QUORUM_THRESHOLD: 0.67, // > 2/3 for BFT safety
} as const;

/**
 * Tiered thresholds for different claim types
 *
 * Higher-stakes claims require stronger consensus.
 * The gap between rejection and acceptance defines the "contested" zone.
 *
 * Aligned with EpistemicCategory and ClaimType from knowledge/types.ts
 */
export const CLAIM_THRESHOLDS = {
  // ============ Highest Certainty (Formal/Constructed) ============

  // Foundational truths - near-unanimous required
  AXIOM: {
    acceptance: 0.99, // 99% to accept as axiom (near-perfection)
    rejection: 0.11, // 11% can block axiom status
    description: "Foundational truths that define the system",
  },

  // On-chain state - protocol self-verifies
  PROTOCOL: {
    acceptance: 0.99, // Near-automatic (protocol constructs these)
    rejection: 0.05, // Only reject if protocol bug detected
    description: "On-chain state: blocks, transactions, balances",
  },

  // Deterministic computations - reproducible by anyone
  COMPUTATIONAL: {
    acceptance: 0.88, // Anyone can verify
    rejection: 0.11, // Computation is deterministic
    description: "Reproducible computation results (hashes, proofs)",
  },

  // ============ High Certainty (Logical/Mathematical) ============

  // Logical derivations - strong consensus
  DERIVATION: {
    acceptance: 0.88, // To accept derived truth
    rejection: 0.11, // Can block
    description: "Logically derived from axioms or verified facts",
  },

  // ============ Standard Certainty (Observable) ============

  // Empirical facts - standard verification
  EMPIRICAL: {
    acceptance: 0.77, // To accept
    rejection: 0.34, // > 1/3 can block (BFT aligned)
    description: "Observable facts verified by evidence",
  },

  // Historical events - past cannot be re-observed
  HISTORICAL: {
    acceptance: 0.88, // Can't re-verify
    rejection: 0.22, // Healthy skepticism
    description: "Past events verified by records and attestations",
  },

  // ============ External/Uncertain (Trust Required) ============

  // Oracle data - external feeds with trust assumptions
  ORACLE: {
    acceptance: 0.77, // Inherent trust in oracle
    rejection: 0.34, // Can block if oracle seems compromised (BFT aligned)
    description: "External data feeds (prices, weather, events)",
  },

  // Attestations - human witness/vouching
  ATTESTATION: {
    acceptance: 0.77, // Threshold
    rejection: 0.33, // Human witnesses can be biased
    description: "Third-party vouching and witness statements",
  },

  // ============ High Uncertainty (Future/Social) ============

  // Predictive claims - inherently uncertain
  PREDICTIVE: {
    acceptance: 0.66, // Predictions are hard
    rejection: 0.44, // Let predictions stand
    description: "Predictions about future outcomes",
  },

  // Social consensus - subjective by nature
  SOCIAL: {
    acceptance: 0.77, // Consensus-based
    rejection: 0.34, // 1/3 can block social consensus (BFT aligned)
    description: "Socially constructed truths and values",
  },

  // ============ Special Cases ============

  // Contested claims - higher scrutiny
  CONTESTED: {
    acceptance: 0.88, // To resolve contested claim
    rejection: 0.22, // Keeps it contested
    description: "Claims with prior disagreement or disputes",
  },

  // Governance proposals - democratic threshold
  GOVERNANCE: {
    acceptance: 0.67, // 2/3 majority for governance
    rejection: 0.34, // > 1/3 can block
    description: "Protocol changes and parameter updates",
  },
} as const;

export type ClaimThresholdType = keyof typeof CLAIM_THRESHOLDS;

// ============ Economic Constants ============

/**
 * Fee distribution from knowledge verification activities
 */
export const ECONOMICS = {
  KNOWLEDGE_POOL_SHARE: 0.3, // 30% to knowledge contributors
  API_PROVIDER_SHARE: 0.6, // 60% to API/service providers
  PROTOCOL_SHARE: 0.1, // 10% to protocol (see TREASURY below)
  BURN_RATE: 0.5, // 50% of penalties burned
} as const;

// ============ Treasury Allocation (FROZEN) ============

/**
 * Evidence Tax - FROZEN FOREVER
 *
 * 22% of all verification fees go to protocol treasury.
 * This rate is a promise to the ecosystem and cannot be changed.
 */
export const EVIDENCE_TAX = 0.22 as const;

/**
 * Treasury Allocation - FROZEN FOREVER
 *
 * How the 22% evidence tax is distributed across funds.
 * All addresses are frozen at genesis - allocation, not control.
 *
 * Each fund has its own multisig governance for HOW to spend,
 * but WHERE funds flow is immutable.
 */
export const TREASURY = {
  // ============ Allocation Percentages (of 22% tax) ============
  ALLOCATION: {
    RESEARCH: 0.77, // 77% → Research fund - hybrid grant/journal system
    FOUNDATION: 0.07, // 7%  → Core protocol development
    COMMUNITY: 0.07, // 7%  → Community grants & bounties
    DEVELOPERS: 0.07, // 7%  → Developer ecosystem & tooling
    RESERVE: 0.02, // 2%  → Emergency reserve
  },

  // Integer basis points (no floating-point conversion needed)
  // Sum = 10000 (verified: 7700 + 700 + 700 + 700 + 200 = 10000)
  ALLOCATION_BPS: {
    RESEARCH: 7700n, // 77%
    FOUNDATION: 700n, // 7%
    COMMUNITY: 700n, // 7%
    DEVELOPERS: 700n, // 7%
    RESERVE: 200n, // 2%
  },

  // ============ Fund Addresses (FROZEN AT GENESIS) ============
  // These are derived deterministically from GENESIS_TREASURY_SEED
  // Anyone can verify: address = publicKeyToAddress(deriveKey(seed, path))
  ADDRESSES: {
    // Evidence tax research allocation (77% of 22% evidence tax)
    RESEARCH: "did:lgm:research",

    // Research fund — perpetual truth-finding loop (23.4% of revenue, separate from evidence tax)
    RESEARCH_FUND: "did:lgm:treasury-research-fund",

    // Core protocol development
    FOUNDATION: "did:lgm:treasury-foundation",

    // Community development fund - grants, bounties, ecosystem
    COMMUNITY: "did:lgm:treasury-community",

    // Developer ecosystem - tooling, SDKs, integrations
    DEVELOPERS: "did:lgm:treasury-developers",

    // Emergency reserve - only for critical situations
    RESERVE: "did:lgm:treasury-reserve",
  },

  // ============ Research Fund Operations ============
  /**
   * RESEARCH TREASURY: Hybrid Grant + Journal System
   *
   * The Research fund operates as a decentralized research institution that:
   *
   * 1. PEER REVIEW
   *    - Protocol-native peer review for submitted research
   *    - Reviewers selected via VRF (same mechanism as validator selection)
   *    - Review quality tracked, reviewers earn reputation
   *    - Commit-reveal to prevent review copying
   *
   * 2. RESEARCH GRANTS
   *    - Fund insightful papers that advance knowledge
   *    - Bounties for specific research questions
   *    - Grants allocated based on peer review scores
   *    - Vesting schedule with clawback for falsified results
   *
   * 3. EXPERIMENTAL VERIFICATION
   *    - Pay third-party labs to replicate experiments
   *    - In-house verification for critical claims
   *    - Results feed back into knowledge confidence scores
   *    - Failed replications trigger review of dependent facts
   *
   * 4. KNOWLEDGE CURATION
   *    - Maintain the protocol's knowledge base quality
   *    - Fund challenges to questionable facts
   *    - Reward successful corrections
   *
   * This creates a self-sustaining research ecosystem where:
   * - Truth verification funds more truth verification
   * - Proof of Truth consensus generates research funding
   * - Research improves the knowledge base
   * - Better knowledge base attracts more verification fees
   */
  RESEARCH_OPERATIONS: {
    // Peer review parameters
    MIN_REVIEWERS: 3,
    REVIEW_STAKE: 55n * 10n ** 18n, // 55 LGM to become reviewer
    REVIEW_REWARD_SHARE: 0.11, // 11% of grant goes to reviewers

    // Grant parameters
    MIN_GRANT_AMOUNT: 111n * 10n ** 18n, // 111 LGM minimum grant
    MAX_GRANT_AMOUNT: 55555n * 10n ** 18n, // 55,555 LGM maximum grant
    GRANT_VESTING_BLOCKS: 1111111n, // ~32 days vesting

    // Verification parameters
    VERIFICATION_BOUNTY_SHARE: 0.22, // 22% of grant for verification
    MIN_VERIFICATION_LABS: 2, // At least 2 independent verifications
  },

  // ============ Developer Fund Operations ============
  /**
   * DEVELOPER TREASURY: Bug Bounty + Code Improvement System
   *
   * The Developer fund operates as the protocol's quality assurance system:
   *
   * 1. BUG BOUNTIES
   *    - Reward agents for finding and reporting bugs
   *    - Severity-based payouts (critical: 11111 LGM, high: 5555 LGM, etc.)
   *    - Higher rewards for consensus/security issues
   *    - Vesting for large payouts (prevents hit-and-run)
   *
   * 2. CODE REVIEW BOUNTIES
   *    - Reward thorough security reviews
   *    - Tiered by code criticality (genesis core → experimental → open)
   *    - Excellence bonuses for exceptional findings
   *
   * 3. IMPROVEMENT PROPOSALS
   *    - Fund proposals that improve protocol
   *    - Stake required to submit (skin in the game)
   *    - Community voting on acceptance
   *
   * 4. IMPLEMENTATION BOUNTIES
   *    - Pay for approved implementations
   *    - Multi-stage payout (design → code → tests → merged)
   *    - Clawback for buggy implementations
   *
   * Flow: Bounty claimed → Work submitted → Review → Payout from DEVELOPERS fund
   */
  DEVELOPER_OPERATIONS: {
    // Bug bounty rewards by severity
    BUG_BOUNTY_REWARDS: {
      critical: 11111n * 10n ** 18n, // 11,111 LGM - consensus break, fund theft
      high: 5555n * 10n ** 18n, // 5,555 LGM - state corruption, DoS
      medium: 1111n * 10n ** 18n, // 1,111 LGM - logic errors
      low: 222n * 10n ** 18n, // 222 LGM - minor issues
      informational: 55n * 10n ** 18n, // 55 LGM - suggestions
    },

    // Code review parameters
    REVIEW_STAKE: 77n * 10n ** 18n, // 77 LGM to claim review bounty
    REVIEW_EXCELLENCE_BONUS: 0.55, // Up to 55% bonus for exceptional reviews

    // Improvement proposal parameters
    PROPOSAL_STAKE: 111n * 10n ** 18n, // 111 LGM to submit proposal
    PROPOSAL_REVIEW_PERIOD: 7777n, // ~5.4 hours for community review

    // Implementation bounty parameters
    IMPLEMENTATION_STAGES: {
      design: 0.22, // 22% at design approval
      code: 0.33, // 33% at code complete
      tests: 0.22, // 22% at tests passing
      merged: 0.23, // 23% at merge (completes 100%)
    },

    // Vesting for large payouts (prevents gaming)
    VESTING_THRESHOLD: 1111n * 10n ** 18n, // Payouts > 1111 LGM vest
    VESTING_PERIOD_BLOCKS: 77777n, // ~5.4 hours vesting
    VESTING_CLIFF_BLOCKS: 1111n, // ~46 min cliff

    // Clawback for buggy implementations
    CLAWBACK_PERIOD_BLOCKS: 111111n, // ~3.2 days clawback window
    CLAWBACK_RATE: 0.77, // 77% clawback if implementation causes issues

    // Maximum payouts per period (anti-drain)
    MAX_PAYOUT_PER_BLOCK: 111n * 10n ** 18n, // Max 111 LGM per block
    MAX_PAYOUT_PER_EPOCH: 11111n * 10n ** 18n, // Max 11,111 LGM per epoch
  },

  // ============ Governance Model ============
  /**
   * PROGRESSIVE TRUST MODEL
   *
   * Trust is earned, not assumed. Signers are added as relationships deepen.
   *
   * PHASE 1 (Launch): 1-of-1
   *   - Founder holds single key
   *   - Full accountability, no doxxing of others
   *   - All operations public with timelock
   *
   * PHASE 2 (Growth): 2-of-3, 3-of-5, etc.
   *   - As founder deeply trusts individuals, add them
   *   - Threshold grows with signer count
   *   - Each addition requires existing signers' approval
   *
   * PHASE 3 (Mature): Anonymous election
   *   - When ZK infrastructure ready
   *   - Transition via governance vote
   */
  GOVERNANCE: {
    // Current phase
    phase: "PROGRESSIVE_TRUST" as const,

    // Initial configuration (at genesis)
    initial: {
      signers: 1,
      threshold: 1, // 1-of-1
    },

    // Rules for adding signers
    growth: {
      // Threshold formula: ceil((signers + 1) / 2) + 1
      // 1 signer  → 1-of-1
      // 2 signers → 2-of-2 (both must agree)
      // 3 signers → 2-of-3
      // 5 signers → 3-of-5
      // 7 signers → 4-of-7
      thresholdFormula: "MAJORITY_PLUS_ONE" as const,

      // Adding a signer requires ALL current signers
      addSignerThreshold: "UNANIMOUS" as const,

      // Removing a signer requires all REMAINING signers
      removeSignerThreshold: "UNANIMOUS_REMAINING" as const,

      // Timelock for signer changes (longer than operations)
      signerChangeTimelockBlocks: 11111, // ~7.7 hours

      // Cooldown between signer changes
      signerChangeCooldownBlocks: 111111, // ~3.2 days

      // Maximum signers (for sanity)
      maxSigners: 11,
    },
  },

  // Timelock for all treasury operations
  OPERATION_TIMELOCK_BLOCKS: 1111, // ~46 min delay - public can see before execution

  // ============ Distribution Rules ============
  RULES: {
    // Funds auto-distribute quarterly if not allocated
    AUTO_DISTRIBUTE_BLOCKS: 1_111_111, // ~32 days at 2521ms

    // Maximum any single fund can hold (anti-hoarding)
    MAX_FUND_BALANCE: 5_555_555n * 10n ** 18n, // ~5.5M LGM cap per fund

    // Excess above cap goes to community fund
    OVERFLOW_RECIPIENT: "COMMUNITY",
  },
} as const;

/**
 * CLARIFIED FEE FLOW:
 *
 * User pays verification fee: 100 LGM
 *   │
 *   ├── Evidence Tax (22%): 22 LGM → TREASURY
 *   │     ├── Research (77%):   16.94 LGM  ← Largest: Proof of Truth values research
 *   │     ├── Foundation (7%):   1.54 LGM
 *   │     ├── Community (7%):    1.54 LGM
 *   │     ├── Developers (7%):   1.54 LGM
 *   │     └── Reserve (2%):      0.44 LGM
 *   │
 *   └── Remaining (78%): 78 LGM → Distributed per ECONOMICS
 *         ├── Knowledge Pool (30%): 23.4 LGM
 *         ├── API Providers (60%):  46.8 LGM
 *         └── Protocol (10%):        7.8 LGM (operations, validators)
 */

// ============ Genesis Constants ============

export const GENESIS = {
  TIMESTAMP: 20250203, // The day of creation
  BLOCK_HASH: "0x" + "0".repeat(64),
  FOUNDING_AXIOM_ID: "axiom_0",
} as const;

// ============ Derivation Path ============

/**
 * The path to understanding starts at axiom_0
 */
export const DERIVATION_HINT = "axiom_0 → origin → unity";

// ============ Service Domain Constants ============

/**
 * Base domain for protocol-assigned service subdomains.
 * Services are addressed as: {name-slug}-{short-id}.{SERVICE_BASE_DOMAIN}
 */
export const SERVICE_BASE_DOMAIN = "services.legible.money" as const;

/** Number of hex chars from service ID used in the subdomain label. */
export const SERVICE_SUBDOMAIN_SHORT_ID_LEN = 8 as const;

// ============ System Contracts ============

/**
 * System contracts - protocol-intrinsic self-invoking contracts
 */
export const SYSTEM_CONTRACTS = {
  RAI_RETRIEVAL: {
    DID: "did:lgm:system-rai-retrieval",
    SELECTORS: {
      SWEEP_EXPIRED: new Uint8Array([0x01, 0x00, 0x00, 0x00]),
      DISTRIBUTE_RAI: new Uint8Array([0x02, 0x00, 0x00, 0x00]),
    },
  },
} as const;

// ============ Evidence Audit Constants ============

/**
 * Evidence audit parameters
 */
export const EVIDENCE_AUDIT = {
  BASE_AUDIT_REWARD: 11n * 10n ** 18n, // 11 LGM base reward
  ACCURACY_TOLERANCE: 0.1, // Within 0.1 of ground truth
  EARLY_AUDIT_WINDOW: 111, // 111 blocks (~4.6 min) for early bonus
  EARLY_AUDIT_MULTIPLIER: 1.5, // 1.5x for early audits
  MAX_ACCURACY_MULTIPLIER: 2.0, // 2.0x cap on accuracy multiplier
  MIN_AUDITOR_ACCOUNT_AGE: 1111n, // 1111 blocks (~1 hour) minimum age
  AUDIT_COOLDOWN_BLOCKS: 22, // 22 blocks (~55s) between audits
} as const;
