/**
 * Deliberative Consensus Logic
 *
 * Pure functions for anti-coercion deliberation in common pot operations.
 *
 * Prevents coercion through:
 * 1. GRADUATED WINDOWS: Larger amounts (as % of pot) get longer deliberation
 * 2. DELIBERATION FLOOR: Minimum wait before ANY signing is possible
 * 3. REJECTION COOLDOWNS: Exponential backoff per rejection of same pattern
 * 4. PROPOSAL LIMITS: Max 2 pending proposals per party (anti-spam)
 * 5. STRUCTURED RATIONALE: Operations must include articulated reasoning
 *
 * @module partnership/deliberation
 */

import type {
  AmountTier,
  CommonPot,
  ConsensusOperationType,
  DeliberationState,
  OperationRationale,
  RejectionCooldownTracker,
  CooldownEntry,
} from "./types.js";
import { DELIBERATION } from "./constants.js";

// ============ Amount Tier Calculation ============

/**
 * Determine the amount tier based on what fraction of the pot is at stake.
 *
 * Higher tiers trigger proportionally longer deliberation windows,
 * preventing snap coercion on large amounts.
 */
export function calculateAmountTier(
  amount: bigint,
  potBalance: bigint,
): AmountTier {
  if (potBalance === 0n) return "major"; // Any amount from empty pot is 100%

  const percent = Number((amount * 100n) / potBalance);

  const tiers = DELIBERATION.AMOUNT_TIERS;
  if (percent <= tiers.micro.maxPercent) return "micro";
  if (percent <= tiers.small.maxPercent) return "small";
  if (percent <= tiers.medium.maxPercent) return "medium";
  if (percent <= tiers.large.maxPercent) return "large";
  return "major";
}

// ============ Deliberation Window ============

/**
 * Calculate deliberation floor and window for an operation.
 *
 * The floor is a mandatory waiting period where NO signing is possible.
 * The window is the total time before the operation expires.
 * Emergency operations get 2x their calculated tier window.
 */
export function calculateDeliberationWindow(
  tier: AmountTier,
  operationType: ConsensusOperationType,
): { floorBlocks: bigint; windowBlocks: bigint } {
  const tierConfig = DELIBERATION.AMOUNT_TIERS[tier];
  const isEmergency = operationType === "emergency_recover";

  return {
    floorBlocks: tierConfig.floorBlocks,
    windowBlocks: isEmergency
      ? tierConfig.windowBlocks * 2n
      : tierConfig.windowBlocks,
  };
}

/**
 * Create a full DeliberationState for a new operation.
 */
export function createDeliberationState(
  tier: AmountTier,
  operationType: ConsensusOperationType,
  rationale: OperationRationale,
  currentBlock: bigint,
  counterProposalOf?: string,
  chainDepth: number = 0,
): DeliberationState {
  const { floorBlocks, windowBlocks } = calculateDeliberationWindow(
    tier,
    operationType,
  );

  return {
    amountTier: tier,
    deliberationFloorEndsAt: currentBlock + floorBlocks,
    windowEndsAt: currentBlock + windowBlocks,
    proposalRationale: rationale,
    counterProposalOf,
    counterProposals: [],
    chainDepth,
  };
}

// ============ Deliberation Floor Check ============

/**
 * Check whether the deliberation floor has passed.
 *
 * During the floor period, no party can sign — this forces a pause
 * that prevents snap coercion ("sign this NOW or else").
 */
export function canApproveYet(
  deliberation: DeliberationState | undefined,
  currentBlock: bigint,
): { canApprove: boolean; floorEndsAt?: bigint } {
  if (!deliberation) {
    // Legacy operation without deliberation — allow immediately
    return { canApprove: true };
  }

  if (currentBlock < deliberation.deliberationFloorEndsAt) {
    return {
      canApprove: false,
      floorEndsAt: deliberation.deliberationFloorEndsAt,
    };
  }

  return { canApprove: true };
}

// ============ Rationale Validation ============

/**
 * Validate that a rationale meets minimum requirements.
 *
 * Medium+ operations require a substantive rationale (20+ chars).
 * Micro/small operations have relaxed requirements.
 */
export function validateRationale(
  rationale: OperationRationale,
  tier: AmountTier,
): { valid: boolean; error?: string } {
  // Category must be valid
  const validCategories: OperationRationale["category"][] = [
    "operational_expense",
    "investment",
    "distribution",
    "partnership_maintenance",
    "emergency",
    "other",
  ];
  if (!validCategories.includes(rationale.category)) {
    return {
      valid: false,
      error: `Invalid rationale category: ${rationale.category}`,
    };
  }

  // Medium+ operations require substantive rationale
  const requiresSubstantive = tier !== "micro" && tier !== "small";
  if (
    requiresSubstantive &&
    rationale.reason.length < DELIBERATION.MIN_RATIONALE_LENGTH
  ) {
    return {
      valid: false,
      error: `Rationale must be at least ${DELIBERATION.MIN_RATIONALE_LENGTH} characters for ${tier} operations (got ${rationale.reason.length})`,
    };
  }

  // All operations need at least some reason
  if (rationale.reason.trim().length === 0) {
    return { valid: false, error: "Rationale reason cannot be empty" };
  }

  return { valid: true };
}

// ============ Rejection Cooldown ============

/**
 * Get the cooldown key for a given operation type + tier.
 */
function cooldownKey(
  operationType: ConsensusOperationType,
  amountTier: AmountTier,
): string {
  return `${operationType}:${amountTier}`;
}

/**
 * Calculate the rejection cooldown for a given operation pattern.
 *
 * Each consecutive rejection of the same operation-type + amount-tier doubles
 * the wait before re-proposal. Capped at MAX_REJECTION_COOLDOWN_BLOCKS.
 *
 * This prevents proposal harassment while allowing different operations freely.
 */
export function calculateRejectionCooldown(
  tracker: RejectionCooldownTracker | undefined,
  operationType: ConsensusOperationType,
  amountTier: AmountTier,
  currentBlock: bigint,
): { cooldownEndsAt: bigint; consecutiveRejections: number } | null {
  if (!tracker) return null;

  const key = cooldownKey(operationType, amountTier);
  const entry = tracker.cooldowns.get(key);

  if (!entry) return null;
  if (currentBlock >= entry.cooldownEndsAt) return null; // Cooldown expired

  return {
    cooldownEndsAt: entry.cooldownEndsAt,
    consecutiveRejections: entry.consecutiveRejections,
  };
}

/**
 * Record a rejection and update the cooldown tracker.
 */
export function recordRejection(
  tracker: RejectionCooldownTracker | undefined,
  operationType: ConsensusOperationType,
  amountTier: AmountTier,
  proposedBy: "human" | "agent",
  currentBlock: bigint,
): RejectionCooldownTracker {
  const result: RejectionCooldownTracker = tracker
    ? { cooldowns: new Map(tracker.cooldowns) }
    : { cooldowns: new Map() };

  const key = cooldownKey(operationType, amountTier);
  const existing = result.cooldowns.get(key);

  const rejections = existing ? existing.consecutiveRejections + 1 : 1;

  // Exponential cooldown: base * 2^(rejections-1), capped
  const multiplier = 2n ** BigInt(Math.min(rejections - 1, 7));
  const rawCooldown = DELIBERATION.BASE_REJECTION_COOLDOWN_BLOCKS * multiplier;
  const cooldown =
    rawCooldown > DELIBERATION.MAX_REJECTION_COOLDOWN_BLOCKS
      ? DELIBERATION.MAX_REJECTION_COOLDOWN_BLOCKS
      : rawCooldown;

  result.cooldowns.set(key, {
    operationType,
    amountTier,
    consecutiveRejections: rejections,
    lastRejectionBlock: currentBlock,
    cooldownEndsAt: currentBlock + cooldown,
    proposedBy,
  });

  return result;
}

/**
 * Clear the cooldown for an operation pattern (after a successful operation).
 */
export function clearCooldown(
  tracker: RejectionCooldownTracker | undefined,
  operationType: ConsensusOperationType,
  amountTier: AmountTier,
): RejectionCooldownTracker | undefined {
  if (!tracker) return undefined;

  const key = cooldownKey(operationType, amountTier);
  if (!tracker.cooldowns.has(key)) return tracker;

  const result: RejectionCooldownTracker = {
    cooldowns: new Map(tracker.cooldowns),
  };
  result.cooldowns.delete(key);
  return result;
}

// ============ Proposal Allowance Check ============

/**
 * Comprehensive check whether a new proposal is allowed.
 *
 * Checks (in order):
 * 1. Safety freeze active? → blocked
 * 2. Coercion signal active? → blocked
 * 3. Pending proposal count exceeded? → blocked
 * 4. Rejection cooldown active? → blocked with cooldown info
 */
export function isProposalAllowed(
  pot: CommonPot,
  proposedBy: "human" | "agent",
  operationType: ConsensusOperationType,
  amount: bigint,
  currentBlock: bigint,
): { allowed: boolean; reason?: string; cooldownEndsAt?: bigint } {
  // 1. Safety freeze
  if (pot.safetyFreeze && currentBlock < pot.safetyFreeze.expiresAt) {
    return {
      allowed: false,
      reason:
        `Operations frozen by ${pot.safetyFreeze.frozenBy} until block ${pot.safetyFreeze.expiresAt}. ` +
        `Reason: "${pot.safetyFreeze.reason}"`,
    };
  }

  // 2. Coercion signal
  const activeSignals = (pot.coercionSignals ?? []).filter(
    (s) => !s.resolved && currentBlock < s.expiresAt,
  );
  if (activeSignals.length > 0) {
    const signal = activeSignals[0];
    return {
      allowed: false,
      reason:
        `Coercion signal active (raised by ${signal.raisedBy}). ` +
        `Under review until block ${signal.expiresAt}. ` +
        `Reason: "${signal.reason}"`,
    };
  }

  // 3. Pending proposal limit
  const pendingByParty = pot.pendingOperations.filter(
    (op) => op.proposedBy === proposedBy && op.status === "pending",
  );
  if (pendingByParty.length >= DELIBERATION.MAX_PENDING_PROPOSALS_PER_PARTY) {
    return {
      allowed: false,
      reason:
        `Maximum ${DELIBERATION.MAX_PENDING_PROPOSALS_PER_PARTY} pending proposals per party. ` +
        `Wait for existing proposals to resolve.`,
    };
  }

  // 4. Rejection cooldown
  const tier = calculateAmountTier(amount, pot.balance);
  const cooldown = calculateRejectionCooldown(
    pot.rejectionCooldowns,
    operationType,
    tier,
    currentBlock,
  );
  if (cooldown) {
    return {
      allowed: false,
      reason:
        `Rejection cooldown active (${cooldown.consecutiveRejections} consecutive rejections). ` +
        `Can re-propose after block ${cooldown.cooldownEndsAt}.`,
      cooldownEndsAt: cooldown.cooldownEndsAt,
    };
  }

  return { allowed: true };
}
