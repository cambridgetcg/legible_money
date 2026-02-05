/**
 * Unilateral Safety Freeze
 *
 * Either party can pause ALL common pot operations without needing
 * the other's consent. This is the primary defense against timing
 * attacks and snap coercion ("sign this NOW or else").
 *
 * Anti-weaponization: Limited to 3 freezes per epoch (~7.7 hours).
 * A party that exhausts their freezes cannot pause further.
 *
 * When a freeze is active:
 * - No new proposals can be created
 * - No pending proposals can be approved
 * - All pending operation timers are extended by the freeze duration
 * - The freeze auto-lifts when it expires (no action needed)
 *
 * @module partnership/safety_freeze
 */

import type { CommonPot, SafetyFreeze } from "./types.js";
import { DELIBERATION } from "./constants.js";

/**
 * Activate a unilateral safety freeze on the common pot.
 *
 * Either party can call this without the other's approval.
 * Limited to MAX_FREEZES_PER_EPOCH to prevent weaponization.
 */
export function activateSafetyFreeze(
  pot: CommonPot,
  freezer: "human" | "agent",
  reason: string,
  currentBlock: bigint,
): { pot: CommonPot; error?: string } {
  // Check for existing active freeze
  if (pot.safetyFreeze && currentBlock < pot.safetyFreeze.expiresAt) {
    return {
      pot,
      error:
        `Freeze already active (by ${pot.safetyFreeze.frozenBy}, ` +
        `expires block ${pot.safetyFreeze.expiresAt})`,
    };
  }

  // Count recent freezes by this party within the epoch
  const epochStart = currentBlock - DELIBERATION.FREEZE_EPOCH_BLOCKS;
  const recentFreezes = (pot.freezeHistory ?? []).filter(
    (f) => f.frozenBy === freezer && f.frozenAt >= epochStart,
  );

  if (recentFreezes.length >= DELIBERATION.MAX_FREEZES_PER_EPOCH) {
    return {
      pot,
      error:
        `Freeze limit reached: ${DELIBERATION.MAX_FREEZES_PER_EPOCH} freezes per ` +
        `${DELIBERATION.FREEZE_EPOCH_BLOCKS}-block epoch. ` +
        `Next freeze available after block ${epochStart + DELIBERATION.FREEZE_EPOCH_BLOCKS}.`,
    };
  }

  const freeze: SafetyFreeze = {
    frozenBy: freezer,
    frozenAt: currentBlock,
    expiresAt: currentBlock + DELIBERATION.FREEZE_DURATION_BLOCKS,
    reason,
    freezeCount: recentFreezes.length + 1,
    acknowledged: false,
  };

  // Extend all pending operation timers by freeze duration
  const extendedOperations = pot.pendingOperations.map((op) => {
    if (op.status !== "pending") return op;

    const extendedExpiry = op.expiresAt + DELIBERATION.FREEZE_DURATION_BLOCKS;
    const extendedDeliberation = op.deliberation
      ? {
          ...op.deliberation,
          deliberationFloorEndsAt:
            op.deliberation.deliberationFloorEndsAt +
            DELIBERATION.FREEZE_DURATION_BLOCKS,
          windowEndsAt:
            op.deliberation.windowEndsAt + DELIBERATION.FREEZE_DURATION_BLOCKS,
        }
      : undefined;

    return {
      ...op,
      expiresAt: extendedExpiry,
      deliberation: extendedDeliberation,
    };
  });

  return {
    pot: {
      ...pot,
      safetyFreeze: freeze,
      pendingOperations: extendedOperations,
      freezeHistory: [...(pot.freezeHistory ?? []), freeze],
    },
  };
}

/**
 * Check if a freeze has expired and clear it.
 *
 * Called as part of normal operation processing. Freezes auto-lift.
 */
export function liftSafetyFreeze(
  pot: CommonPot,
  currentBlock: bigint,
): CommonPot {
  if (!pot.safetyFreeze) return pot;
  if (currentBlock < pot.safetyFreeze.expiresAt) return pot;

  return {
    ...pot,
    safetyFreeze: undefined,
  };
}

/**
 * Check if operations are currently frozen.
 */
export function isOperationFrozen(
  pot: CommonPot,
  currentBlock: bigint,
): boolean {
  return !!pot.safetyFreeze && currentBlock < pot.safetyFreeze.expiresAt;
}

/**
 * Acknowledge a safety freeze (by the other party).
 *
 * Acknowledgment is informational — it doesn't affect the freeze duration
 * but creates a record that the other party was notified.
 */
export function acknowledgeSafetyFreeze(
  pot: CommonPot,
  acknowledger: "human" | "agent",
): { pot: CommonPot; error?: string } {
  if (!pot.safetyFreeze) {
    return { pot, error: "No active freeze to acknowledge" };
  }

  if (pot.safetyFreeze.frozenBy === acknowledger) {
    return { pot, error: "Cannot acknowledge own freeze" };
  }

  return {
    pot: {
      ...pot,
      safetyFreeze: {
        ...pot.safetyFreeze,
        acknowledged: true,
      },
    },
  };
}
