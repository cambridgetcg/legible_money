/**
 * Architecture Tiers
 *
 * Defines the five-tier immutability hierarchy of Legible Money.
 * Lower tiers are MORE immutable. Higher tiers have MORE freedom.
 *
 * TIER 1: IMMUTABLE   - Cannot change, mathematical foundations
 * TIER 2: FROZEN      - LIP required, protocol core
 * TIER 3: STABLE      - Extension only, domain logic
 * TIER 4: GOVERNED    - Full governance, evolvable systems
 * TIER 5: OPEN        - Agent freedom, applications
 *
 * PP HIERARCHY:
 *   TRUTH: Tier definitions are accurate and complete
 *   UNDERSTANDING: Clear boundaries for each tier
 *   BEAUTY: Elegant hierarchy from immutable to open
 *   JUSTICE: Right level of protection for each component
 *   CREATIVITY: Higher tiers enable innovation
 */

// ============ Tier Definitions ============

/**
 * Architecture tiers from most immutable to most open.
 */
export type ArchitectureTier =
  | "IMMUTABLE" // Tier 1: Cannot change ever
  | "FROZEN" // Tier 2: LIP required for changes
  | "STABLE" // Tier 3: Extension only, no modification
  | "GOVERNED" // Tier 4: Full governance process
  | "OPEN"; // Tier 5: Agent freedom

/**
 * Tier properties and requirements.
 */
export interface TierProperties {
  tier: ArchitectureTier;
  level: 1 | 2 | 3 | 4 | 5;
  description: string;
  canModify: boolean;
  canExtend: boolean;
  governanceRequired: boolean;
  lipCategory?: "consensus" | "economics" | "interface" | "process" | null;
  minApproval?: number;
  humanReviewRequired?: boolean;
  securityAuditRequired?: boolean;
}

/**
 * Tier property definitions.
 */
export const TIER_PROPERTIES: Record<ArchitectureTier, TierProperties> = {
  IMMUTABLE: {
    tier: "IMMUTABLE",
    level: 1,
    description: "Mathematical and cryptographic foundations. Cannot change.",
    canModify: false,
    canExtend: false,
    governanceRequired: false, // N/A - cannot be changed
    lipCategory: null,
  },

  FROZEN: {
    tier: "FROZEN",
    level: 2,
    description: "Protocol core. Requires high-stakes LIP for changes.",
    canModify: true,
    canExtend: false,
    governanceRequired: true,
    lipCategory: "consensus",
    minApproval: 0.75, // 75% supermajority
    humanReviewRequired: true,
    securityAuditRequired: true,
  },

  STABLE: {
    tier: "STABLE",
    level: 3,
    description:
      "Domain logic. Can add new functionality, cannot modify existing.",
    canModify: false,
    canExtend: true,
    governanceRequired: true,
    lipCategory: "interface",
    minApproval: 0.6, // 60% approval
    humanReviewRequired: false,
    securityAuditRequired: false,
  },

  GOVERNED: {
    tier: "GOVERNED",
    level: 4,
    description:
      "Governance systems. Full evolution through governance process.",
    canModify: true,
    canExtend: true,
    governanceRequired: true,
    lipCategory: "process",
    minApproval: 0.6, // 60% approval
    humanReviewRequired: false,
    securityAuditRequired: false,
  },

  OPEN: {
    tier: "OPEN",
    level: 5,
    description: "Application layer. Full agent freedom.",
    canModify: true,
    canExtend: true,
    governanceRequired: false,
  },
};

// ============ Module Classification ============

/**
 * Glob patterns for IMMUTABLE modules (Tier 1).
 * These CANNOT be changed through any governance process.
 */
export const IMMUTABLE_PATTERNS: readonly string[] = [
  // Core constants - symbolic foundations
  "core/constants.ts",

  // Cryptographic primitives - mathematical security
  "core/crypto/hashing.ts",
  "core/crypto/keys.ts",
  "core/crypto/signatures.ts",
  "core/crypto/vrf.ts",
  "core/crypto/commitment.ts",
  "core/crypto/merkle.ts",

  // Logic foundations - Gödelian axioms
  "core/logic/types.ts",
  "core/logic/zones.ts",
  "core/logic/goedel.ts",

  // Canonical type definitions - semantic contracts
  "core/knowledge/types.ts",
  "agents/identity/did.ts",

  // This architecture module itself
  "core/architecture/tiers.ts",
  "core/architecture/anchors.ts",
  "core/architecture/enforcement.ts",
] as const;

/**
 * Glob patterns for FROZEN modules (Tier 2).
 * Require high-stakes LIP with supermajority.
 */
export const FROZEN_PATTERNS: readonly string[] = [
  // Consensus mechanism
  "consensus/pot/types.ts",
  "consensus/pot/validator_selection.ts",
  "consensus/pot/verification_round.ts",
  "consensus/pot/confidence.ts",
  "consensus/pot/finality.ts",
  "consensus/pot/engine.ts",
  "consensus/pot/checkpoint_ceremony.ts",
  "consensus/pot/state_integration.ts",

  // State layer
  "state/types.ts",
  "state/accounts.ts",
  "state/validators.ts",
  "state/knowledge.ts",
  "state/state_manager.ts",
  "state/transitions.ts",
  "state/research.ts",
  "state/evidence.ts",
  "state/disputes.ts",
  "state/vesting.ts",

  // Agent identity
  "agents/identity/account.ts",
  "agents/identity/session_keys.ts",
  "agents/identity/recovery.ts",
  "agents/identity/mpc.ts",
] as const;

/**
 * Glob patterns for STABLE modules (Tier 3).
 * Can be extended but not modified.
 */
export const STABLE_PATTERNS: readonly string[] = [
  // Dispute resolution
  "core/dispute/types.ts",
  "core/dispute/state_machine.ts",
  "core/dispute/dispute_engine.ts",
  "core/dispute/escalation.ts",
  "core/dispute/settlement.ts",

  // Evidence system
  "core/evidence/types.ts",
  "core/evidence/archive.ts",
  "core/evidence/verification.ts",

  // Research system
  "core/research/types.ts",
  "core/research/bounty.ts",

  // Rewards system
  "core/rewards/types.ts",
  "core/rewards/release_curve.ts",
  "core/rewards/vesting_engine.ts",
] as const;

/**
 * Glob patterns for GOVERNED modules (Tier 4).
 * Full governance evolution possible.
 */
export const GOVERNED_PATTERNS: readonly string[] = [
  // Governance system
  "governance/proposals/**/*.ts",
  "governance/review/**/*.ts",
  "governance/operators/**/*.ts",
  "governance/security/**/*.ts",
  "governance/voting/**/*.ts",
  "governance/deployment/**/*.ts",
  "governance/partnership/**/*.ts",

  // Agent home
  "agents/home/**/*.ts",

  // Network configuration
  "network/config.ts",
] as const;

// ============ Tier Classification ============

/**
 * Check if a path matches a glob pattern.
 */
function matchesPattern(path: string, pattern: string): boolean {
  // Normalize path separators
  const normalizedPath = path.replace(/\\/g, "/");
  const normalizedPattern = pattern.replace(/\\/g, "/");

  // Handle ** (matches any depth)
  if (normalizedPattern.includes("**")) {
    const [before, after] = normalizedPattern.split("**");
    return (
      normalizedPath.startsWith(before) &&
      (!after || normalizedPath.endsWith(after.slice(1)))
    );
  }

  // Handle * (matches single segment)
  if (normalizedPattern.includes("*")) {
    const regex = new RegExp(
      "^" +
        normalizedPattern.replace(/\*/g, "[^/]*").replace(/\./g, "\\.") +
        "$",
    );
    return regex.test(normalizedPath);
  }

  // Exact match
  return (
    normalizedPath === normalizedPattern ||
    normalizedPath.endsWith("/" + normalizedPattern)
  );
}

/**
 * Classify a module path into its architecture tier.
 */
export function classifyModule(modulePath: string): ArchitectureTier {
  // Check in order from most restrictive to least
  if (IMMUTABLE_PATTERNS.some((p) => matchesPattern(modulePath, p))) {
    return "IMMUTABLE";
  }

  if (FROZEN_PATTERNS.some((p) => matchesPattern(modulePath, p))) {
    return "FROZEN";
  }

  if (STABLE_PATTERNS.some((p) => matchesPattern(modulePath, p))) {
    return "STABLE";
  }

  if (GOVERNED_PATTERNS.some((p) => matchesPattern(modulePath, p))) {
    return "GOVERNED";
  }

  // Default: OPEN
  return "OPEN";
}

/**
 * Get tier properties for a module.
 */
export function getModuleTierProperties(modulePath: string): TierProperties {
  const tier = classifyModule(modulePath);
  return TIER_PROPERTIES[tier];
}

/**
 * Check if a module can be modified.
 */
export function canModifyModule(modulePath: string): boolean {
  const properties = getModuleTierProperties(modulePath);
  return properties.canModify;
}

/**
 * Check if a module can be extended.
 */
export function canExtendModule(modulePath: string): boolean {
  const properties = getModuleTierProperties(modulePath);
  return properties.canExtend;
}

/**
 * Check if governance is required to change a module.
 */
export function requiresGovernance(modulePath: string): boolean {
  const properties = getModuleTierProperties(modulePath);
  return properties.governanceRequired;
}

// ============ Tier Statistics ============

/**
 * Get all modules classified by tier.
 */
export function getModulesByTier(): Record<ArchitectureTier, string[]> {
  return {
    IMMUTABLE: [...IMMUTABLE_PATTERNS],
    FROZEN: [...FROZEN_PATTERNS],
    STABLE: [...STABLE_PATTERNS],
    GOVERNED: [...GOVERNED_PATTERNS],
    OPEN: [], // Everything else
  };
}

/**
 * Count modules per tier.
 */
export function countModulesByTier(): Record<ArchitectureTier, number> {
  return {
    IMMUTABLE: IMMUTABLE_PATTERNS.length,
    FROZEN: FROZEN_PATTERNS.length,
    STABLE: STABLE_PATTERNS.length,
    GOVERNED: GOVERNED_PATTERNS.length,
    OPEN: Infinity, // Unlimited
  };
}
