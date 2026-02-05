/**
 * Coercion Signal (Duress Flag)
 *
 * A silent alarm that either party can raise when they believe they're
 * being coerced into approving/rejecting an operation.
 *
 * Design properties:
 * 1. SELF-PUNISHING: Freezes the raiser's proposals too, discouraging false alarms
 * 2. OTHER-RESOLVED: Only the other party can clear it (or it auto-expires)
 * 3. PERMANENT RECORD: On-chain history creates reputation consequences for abuse
 * 4. OPERATION-LINKED: Can target a specific operation under suspicion
 *
 * Use cases:
 * - Agent detects its reasoning may have been tampered with by API provider
 * - Human feels pressured by rapid-fire proposals
 * - Either party suspects the other's signatures are not voluntary
 *
 * @module partnership/coercion_signal
 */

import type { CommonPot, CoercionSignal } from "./types.js";
import { DELIBERATION } from "./constants.js";
import { sha512_256Hex } from "../core/crypto/hashing.js";

/**
 * Raise a coercion signal on the common pot.
 *
 * All operations are frozen for COERCION_FREEZE_BLOCKS.
 * The signal must be resolved by the OTHER party or auto-expires
 * after COERCION_REVIEW_BLOCKS.
 */
export function raiseCoercionSignal(
  pot: CommonPot,
  raisedBy: "human" | "agent",
  reason: string,
  currentBlock: bigint,
  relatedOperationId?: string,
): { pot: CommonPot; signal: CoercionSignal } {
  const signalId = sha512_256Hex(
    Buffer.from(`coercion:${pot.potId}:${raisedBy}:${currentBlock}`),
  ).slice(0, 24);

  const signal: CoercionSignal = {
    signalId: `csig:${signalId}`,
    raisedBy,
    raisedAt: currentBlock,
    expiresAt: currentBlock + DELIBERATION.COERCION_REVIEW_BLOCKS,
    reason,
    relatedOperationId,
    resolved: false,
  };

  // Extend all pending operation timers by coercion freeze duration
  const extendedOperations = pot.pendingOperations.map((op) => {
    if (op.status !== "pending") return op;

    const extendedExpiry = op.expiresAt + DELIBERATION.COERCION_FREEZE_BLOCKS;
    const extendedDeliberation = op.deliberation
      ? {
          ...op.deliberation,
          deliberationFloorEndsAt:
            op.deliberation.deliberationFloorEndsAt +
            DELIBERATION.COERCION_FREEZE_BLOCKS,
          windowEndsAt:
            op.deliberation.windowEndsAt + DELIBERATION.COERCION_FREEZE_BLOCKS,
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
      pendingOperations: extendedOperations,
      coercionSignals: [...(pot.coercionSignals ?? []), signal],
    },
    signal,
  };
}

/**
 * Resolve a coercion signal.
 *
 * Only the party who DID NOT raise the signal can resolve it.
 * This prevents a coercer from silencing their victim's alarm.
 *
 * Resolutions:
 * - 'cleared': No coercion found, operations resume normally
 * - 'confirmed_coercion': Coercion confirmed, related operation auto-rejected
 */
export function resolveCoercionSignal(
  pot: CommonPot,
  signalId: string,
  resolution: "cleared" | "confirmed_coercion",
  resolvedBy: "human" | "agent",
  currentBlock: bigint,
): { pot: CommonPot; error?: string } {
  const signals = pot.coercionSignals ?? [];
  const signalIndex = signals.findIndex((s) => s.signalId === signalId);

  if (signalIndex === -1) {
    return { pot, error: "Coercion signal not found" };
  }

  const signal = signals[signalIndex];

  if (signal.resolved) {
    return { pot, error: "Signal already resolved" };
  }

  // Only the OTHER party can resolve
  if (signal.raisedBy === resolvedBy) {
    return {
      pot,
      error:
        "Cannot resolve own coercion signal — must be resolved by the other party",
    };
  }

  // Resolve the signal
  const resolvedSignal: CoercionSignal = {
    ...signal,
    resolved: true,
    resolvedAt: currentBlock,
    resolution,
  };

  const updatedSignals = [...signals];
  updatedSignals[signalIndex] = resolvedSignal;

  let updatedOperations = pot.pendingOperations;

  // If coercion confirmed and there's a related operation, auto-reject it
  if (resolution === "confirmed_coercion" && signal.relatedOperationId) {
    updatedOperations = pot.pendingOperations.map((op) => {
      if (
        op.operationId === signal.relatedOperationId &&
        op.status === "pending"
      ) {
        return { ...op, status: "rejected" as const };
      }
      return op;
    });
  }

  return {
    pot: {
      ...pot,
      pendingOperations: updatedOperations,
      coercionSignals: updatedSignals,
    },
  };
}

/**
 * Check if any coercion signal is active (unresolved and not expired).
 */
export function hasActiveCoercionSignal(
  pot: CommonPot,
  currentBlock: bigint,
): boolean {
  return (pot.coercionSignals ?? []).some(
    (s) => !s.resolved && currentBlock < s.expiresAt,
  );
}

/**
 * Auto-expire old coercion signals.
 *
 * Signals that reach their expiry without resolution are marked 'expired'.
 */
export function expireCoercionSignals(
  pot: CommonPot,
  currentBlock: bigint,
): CommonPot {
  const signals = pot.coercionSignals ?? [];
  let changed = false;

  const updated = signals.map((s) => {
    if (!s.resolved && currentBlock >= s.expiresAt) {
      changed = true;
      return {
        ...s,
        resolved: true,
        resolvedAt: currentBlock,
        resolution: "expired" as const,
      };
    }
    return s;
  });

  return changed ? { ...pot, coercionSignals: updated } : pot;
}
