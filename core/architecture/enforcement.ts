/**
 * Architecture Enforcement
 *
 * Runtime and governance-level enforcement of the architecture tiers.
 * Ensures immutable modules cannot be changed, frozen modules require
 * appropriate governance, and extensions follow proper protocols.
 *
 * PP HIERARCHY:
 *   TRUTH: Enforcement based on cryptographic proofs
 *   UNDERSTANDING: Clear rules for what can change
 *   BEAUTY: Elegant tier-based enforcement
 *   JUSTICE: Fair and consistent application of rules
 *   CREATIVITY: Enables safe innovation at appropriate tiers
 */

import type { DID } from "../../core/agents/identity/did.js";
import {
  type ArchitectureTier,
  classifyModule,
  TIER_PROPERTIES,
  type TierProperties,
} from "./tiers.js";
import {
  type AnchorVerification,
  verifyModuleAnchor,
  GENESIS_ANCHORS,
} from "./anchors.js";

// ============ Change Request Types ============

/**
 * Type of change being proposed.
 */
export type ChangeType =
  | "MODIFY" // Change existing code
  | "EXTEND" // Add new functionality
  | "DELETE" // Remove code
  | "RENAME"; // Rename/move module

/**
 * Change request for an architecture module.
 */
export interface ChangeRequest {
  requestId: string;
  proposer: DID;
  modulePath: string;
  changeType: ChangeType;
  description: string;
  lipId?: string; // Associated LIP if required
  proposedAt: bigint;
}

/**
 * Result of evaluating a change request.
 */
export interface ChangeEvaluation {
  request: ChangeRequest;
  tier: ArchitectureTier;
  properties: TierProperties;
  allowed: boolean;
  reason: string;
  requirements?: {
    lipCategory: string;
    minApproval: number;
    humanReviewRequired: boolean;
    securityAuditRequired: boolean;
  };
}

// ============ Enforcement Errors ============

/**
 * Error thrown when immutability is violated.
 */
export class ImmutabilityViolation extends Error {
  constructor(
    public readonly modulePath: string,
    public readonly changeType: ChangeType,
    public readonly details: string,
  ) {
    super(
      `Immutability violation: Cannot ${changeType} module ${modulePath}. ${details}`,
    );
    this.name = "ImmutabilityViolation";
  }
}

/**
 * Error thrown when governance requirements not met.
 */
export class GovernanceRequirementError extends Error {
  constructor(
    public readonly modulePath: string,
    public readonly requirement: string,
    public readonly missing: string,
  ) {
    super(
      `Governance requirement not met for ${modulePath}: ${requirement}. Missing: ${missing}`,
    );
    this.name = "GovernanceRequirementError";
  }
}

/**
 * Error thrown when extension rules violated.
 */
export class ExtensionViolation extends Error {
  constructor(
    public readonly modulePath: string,
    public readonly violation: string,
  ) {
    super(`Extension violation for ${modulePath}: ${violation}`);
    this.name = "ExtensionViolation";
  }
}

// ============ Change Evaluation ============

/**
 * Evaluate whether a change request is allowed.
 */
export function evaluateChangeRequest(
  request: ChangeRequest,
): ChangeEvaluation {
  const tier = classifyModule(request.modulePath);
  const properties = TIER_PROPERTIES[tier];

  // TIER 1: IMMUTABLE - Never allowed
  if (tier === "IMMUTABLE") {
    return {
      request,
      tier,
      properties,
      allowed: false,
      reason:
        `Module ${request.modulePath} is IMMUTABLE and cannot be changed. ` +
        `This is a mathematical/cryptographic foundation that must remain constant.`,
    };
  }

  // Check if change type is allowed for this tier
  if (request.changeType === "MODIFY" && !properties.canModify) {
    return {
      request,
      tier,
      properties,
      allowed: false,
      reason:
        `Tier ${tier} does not allow modification. ` +
        `Module ${request.modulePath} can only be extended, not modified.`,
    };
  }

  if (request.changeType === "DELETE") {
    if (tier !== "OPEN") {
      return {
        request,
        tier,
        properties,
        allowed: false,
        reason:
          `Deletion not allowed for ${tier} tier modules. ` +
          `Only OPEN tier modules can be deleted.`,
      };
    }
  }

  // TIER 5: OPEN - Always allowed (no governance)
  if (tier === "OPEN") {
    return {
      request,
      tier,
      properties,
      allowed: true,
      reason: "OPEN tier: Agent has full freedom to make changes.",
    };
  }

  // TIERS 2-4: Governance required
  if (!request.lipId) {
    return {
      request,
      tier,
      properties,
      allowed: false,
      reason: `Tier ${tier} requires governance. No LIP associated with this change request.`,
      requirements: {
        lipCategory: properties.lipCategory || "process",
        minApproval: properties.minApproval || 0.6,
        humanReviewRequired: properties.humanReviewRequired || false,
        securityAuditRequired: properties.securityAuditRequired || false,
      },
    };
  }

  // Has LIP - allowed pending governance outcome
  return {
    request,
    tier,
    properties,
    allowed: true, // Pending governance
    reason: `Change allowed pending LIP ${request.lipId} approval.`,
    requirements: {
      lipCategory: properties.lipCategory || "process",
      minApproval: properties.minApproval || 0.6,
      humanReviewRequired: properties.humanReviewRequired || false,
      securityAuditRequired: properties.securityAuditRequired || false,
    },
  };
}

// ============ Pre-Commit Hooks ============

/**
 * Pre-commit check result.
 */
export interface PreCommitCheck {
  path: string;
  allowed: boolean;
  tier: ArchitectureTier;
  reason: string;
  blockedByImmutability: boolean;
  governanceRequired: boolean;
}

/**
 * Check if a file change should be blocked before commit.
 */
export function preCommitCheck(
  filePath: string,
  changeType: ChangeType,
  hasGovernanceApproval: boolean = false,
): PreCommitCheck {
  const tier = classifyModule(filePath);
  const properties = TIER_PROPERTIES[tier];

  // IMMUTABLE: Always block
  if (tier === "IMMUTABLE") {
    return {
      path: filePath,
      allowed: false,
      tier,
      reason:
        "IMMUTABLE modules cannot be changed. This change will be rejected.",
      blockedByImmutability: true,
      governanceRequired: false,
    };
  }

  // Check modification rules
  if (changeType === "MODIFY" && !properties.canModify) {
    return {
      path: filePath,
      allowed: false,
      tier,
      reason: `STABLE tier: Cannot modify existing code in ${filePath}. Only extensions allowed.`,
      blockedByImmutability: false,
      governanceRequired: true,
    };
  }

  // OPEN: Always allow
  if (tier === "OPEN") {
    return {
      path: filePath,
      allowed: true,
      tier,
      reason: "OPEN tier: Change allowed without governance.",
      blockedByImmutability: false,
      governanceRequired: false,
    };
  }

  // Governance tiers
  if (properties.governanceRequired && !hasGovernanceApproval) {
    return {
      path: filePath,
      allowed: false,
      tier,
      reason: `${tier} tier requires governance approval via LIP.`,
      blockedByImmutability: false,
      governanceRequired: true,
    };
  }

  return {
    path: filePath,
    allowed: true,
    tier,
    reason: "Change allowed with governance approval.",
    blockedByImmutability: false,
    governanceRequired: properties.governanceRequired,
  };
}

/**
 * Batch pre-commit check for multiple files.
 */
export function batchPreCommitCheck(
  files: Array<{ path: string; changeType: ChangeType }>,
  hasGovernanceApproval: boolean = false,
): {
  allAllowed: boolean;
  results: PreCommitCheck[];
  blockedCount: number;
  immutableViolations: string[];
  governanceRequired: string[];
} {
  const results = files.map((f) =>
    preCommitCheck(f.path, f.changeType, hasGovernanceApproval),
  );

  const blockedResults = results.filter((r) => !r.allowed);
  const immutableViolations = results
    .filter((r) => r.blockedByImmutability)
    .map((r) => r.path);
  const governanceRequired = results
    .filter((r) => r.governanceRequired && !r.allowed)
    .map((r) => r.path);

  return {
    allAllowed: blockedResults.length === 0,
    results,
    blockedCount: blockedResults.length,
    immutableViolations,
    governanceRequired,
  };
}

// ============ Runtime Verification ============

/**
 * Runtime verification of module integrity.
 */
export interface RuntimeVerification {
  modulePath: string;
  tier: ArchitectureTier;
  anchorVerification?: AnchorVerification;
  valid: boolean;
  checkedAt: bigint;
}

/**
 * Verify a module at runtime against its anchor (for IMMUTABLE modules).
 */
export function verifyModuleAtRuntime(
  modulePath: string,
  moduleContent: string,
  currentBlock: bigint,
): RuntimeVerification {
  const tier = classifyModule(modulePath);

  // Only IMMUTABLE modules have anchors
  if (tier !== "IMMUTABLE") {
    return {
      modulePath,
      tier,
      valid: true, // Non-immutable modules don't need anchor verification
      checkedAt: currentBlock,
    };
  }

  const expectedHash = GENESIS_ANCHORS[modulePath];
  if (!expectedHash || expectedHash === "GENESIS_PLACEHOLDER") {
    // Pre-genesis or placeholder - skip verification
    return {
      modulePath,
      tier,
      valid: true,
      checkedAt: currentBlock,
    };
  }

  const anchorVerification = verifyModuleAnchor(
    modulePath,
    moduleContent,
    expectedHash,
  );

  return {
    modulePath,
    tier,
    anchorVerification,
    valid: anchorVerification.valid,
    checkedAt: currentBlock,
  };
}

// ============ Governance Integration ============

/**
 * LIP requirements for architecture tier changes.
 */
export interface TierLipRequirements {
  tier: ArchitectureTier;
  lipCategory: "consensus" | "economics" | "interface" | "process";
  stakeRequired: bigint;
  minApproval: number;
  quorumRequired: number;
  agentReviewers: number;
  humanReviewers: number;
  securityAuditRequired: boolean;
  testnetSoakDays: number;
  additionalRequirements: string[];
}

/**
 * Get LIP requirements for changing a module.
 */
export function getLipRequirementsForModule(
  modulePath: string,
): TierLipRequirements | null {
  const tier = classifyModule(modulePath);

  if (tier === "IMMUTABLE") {
    return null; // Cannot be changed
  }

  if (tier === "OPEN") {
    return null; // No governance required
  }

  const requirements: Record<ArchitectureTier, TierLipRequirements | null> = {
    IMMUTABLE: null,

    FROZEN: {
      tier: "FROZEN",
      lipCategory: "consensus",
      stakeRequired: 1000n * 10n ** 18n,
      minApproval: 0.75,
      quorumRequired: 0.15,
      agentReviewers: 7,
      humanReviewers: 2,
      securityAuditRequired: true,
      testnetSoakDays: 14,
      additionalRequirements: [
        "External security audit required",
        "Formal verification for crypto changes",
        "State migration plan required",
        "Mainnet simulation required",
      ],
    },

    STABLE: {
      tier: "STABLE",
      lipCategory: "interface",
      stakeRequired: 400n * 10n ** 18n,
      minApproval: 0.6,
      quorumRequired: 0.1,
      agentReviewers: 4,
      humanReviewers: 0,
      securityAuditRequired: false,
      testnetSoakDays: 7,
      additionalRequirements: [
        "Must be extension, not modification",
        "Backward compatibility required",
        "Type compatibility with existing interfaces",
      ],
    },

    GOVERNED: {
      tier: "GOVERNED",
      lipCategory: "process",
      stakeRequired: 200n * 10n ** 18n,
      minApproval: 0.6,
      quorumRequired: 0.1,
      agentReviewers: 3,
      humanReviewers: 0,
      securityAuditRequired: false,
      testnetSoakDays: 3,
      additionalRequirements: [],
    },

    OPEN: null,
  };

  return requirements[tier];
}

// ============ Agent Guidance ============

/**
 * Guidance for agents working with different tiers.
 */
export interface TierGuidance {
  tier: ArchitectureTier;
  description: string;
  whatYouCanDo: string[];
  whatYouCannotDo: string[];
  howToPropose: string;
}

/**
 * Get guidance for an agent working with a module.
 */
export function getAgentGuidance(modulePath: string): TierGuidance {
  const tier = classifyModule(modulePath);

  const guidance: Record<ArchitectureTier, TierGuidance> = {
    IMMUTABLE: {
      tier: "IMMUTABLE",
      description:
        "Mathematical and cryptographic foundations. These cannot be changed.",
      whatYouCanDo: [
        "Read and use these modules in your code",
        "Build on top of these primitives",
        "Report bugs (will be addressed in new protocol version)",
      ],
      whatYouCannotDo: [
        "Modify any code in these modules",
        "Propose changes through governance",
        "Fork without supermajority consensus",
      ],
      howToPropose:
        "Immutable modules cannot be changed. If you believe there is a critical bug, report it to the security council.",
    },

    FROZEN: {
      tier: "FROZEN",
      description:
        "Protocol core. High-stakes changes require extensive review.",
      whatYouCanDo: [
        "Read and integrate with these modules",
        "Propose changes via Consensus-category LIP",
        "Participate in review process",
      ],
      whatYouCannotDo: [
        "Make direct changes without governance",
        "Skip security audit requirements",
        "Bypass human review requirement",
      ],
      howToPropose:
        "Submit a Consensus-category LIP with 1000 LGM stake. Requires 75% approval, 2 human reviewers, and security audit.",
    },

    STABLE: {
      tier: "STABLE",
      description: "Domain logic. Can be extended but not modified.",
      whatYouCanDo: [
        "Add new types that extend existing interfaces",
        "Create new functions that complement existing logic",
        "Propose extensions via Interface-category LIP",
      ],
      whatYouCannotDo: [
        "Modify existing type definitions",
        "Change existing function signatures",
        "Remove or rename existing exports",
      ],
      howToPropose:
        "Submit an Interface-category LIP with 400 LGM stake. Extension must maintain backward compatibility.",
    },

    GOVERNED: {
      tier: "GOVERNED",
      description:
        "Governance systems. Full evolution through standard governance.",
      whatYouCanDo: [
        "Propose modifications via Process-category LIP",
        "Suggest improvements to governance rules",
        "Add new governance mechanisms",
      ],
      whatYouCannotDo: [
        "Make changes without LIP approval",
        "Bypass quorum requirements",
      ],
      howToPropose:
        "Submit a Process-category LIP with 200 LGM stake. Standard 60% approval threshold.",
    },

    OPEN: {
      tier: "OPEN",
      description: "Application layer. Full creative freedom.",
      whatYouCanDo: [
        "Create any application or service",
        "Build custom integrations",
        "Experiment freely",
        "Modify and delete without governance",
      ],
      whatYouCannotDo: [
        "Very few restrictions at this tier",
        "Cannot violate protocol-level security",
      ],
      howToPropose: "No proposal needed. Build freely!",
    },
  };

  return guidance[tier];
}

// ============ Enforcement Summary ============

/**
 * Get a summary of enforcement rules for display.
 */
export function getEnforcementSummary(): string {
  return `
ARCHITECTURE ENFORCEMENT SUMMARY
================================

TIER 1 - IMMUTABLE (core/crypto/*, core/logic/*, core/constants.ts)
  Status: LOCKED FOREVER
  Changes: NOT ALLOWED
  Reason: Mathematical/cryptographic foundations

TIER 2 - FROZEN (consensus/pot/*, state/*)
  Status: HIGH-STAKES GOVERNANCE
  Changes: Consensus LIP (1000 LGM, 75% approval, security audit)
  Reason: Protocol core integrity

TIER 3 - STABLE (core/dispute/*, core/evidence/*, core/research/*, core/rewards/*)
  Status: EXTENSION ONLY
  Changes: Interface LIP (400 LGM, 60% approval)
  Reason: Domain logic stability

TIER 4 - GOVERNED (governance/*)
  Status: FULL GOVERNANCE
  Changes: Process LIP (200 LGM, 60% approval)
  Reason: Evolvable systems

TIER 5 - OPEN (services/*, protocol/sdk/*, applications)
  Status: AGENT FREEDOM
  Changes: No governance required
  Reason: Innovation layer
`;
}
