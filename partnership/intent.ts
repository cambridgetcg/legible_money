/**
 * Extended Formation Intent
 *
 * Replaces the hard 77-block formation window with a two-phase process:
 * 1. Intent declaration (2,222 blocks = ~93 min) — find partner
 * 2. Finalization (77 blocks = ~3.2 min) — standard deposit + finalize
 *
 * Anti-gaming:
 * - 0.011 LGM intent deposit (prevents spam declarations)
 * - Max 3 pending intents per DID
 * - 2,222-block expiry (prevents stale intents)
 * - Finalization uses existing 77-block security window
 */

import type { PartnershipIntent, PartnershipTier } from "./types.js";
import type { DID } from "../core/agents/identity/did.js";
import { INTENT } from "./constants.js";
import { sha512_256Hex } from "../core/crypto/hashing.js";

// ============ Intent Operations ============

/**
 * Declare a partnership formation intent.
 *
 * Agent posts intent with preferred parameters (domains, tier,
 * contribution range). Deposits INTENT_DEPOSIT (0.011 LGM).
 */
export function declareIntent(
  did: DID,
  preferredDomains: string[],
  contributionRange: { min: bigint; max: bigint },
  preferredTier: PartnershipTier,
  deposit: bigint,
  currentBlock: bigint,
  existingPendingCount: number,
): { intent: PartnershipIntent; error?: string } {
  // Validate pending intent limit
  if (existingPendingCount >= INTENT.MAX_PENDING_INTENTS) {
    return {
      intent: null as any,
      error: `Max ${INTENT.MAX_PENDING_INTENTS} pending intents per DID`,
    };
  }

  // Validate deposit
  if (deposit < INTENT.INTENT_DEPOSIT) {
    return {
      intent: null as any,
      error: `Deposit ${deposit} below minimum ${INTENT.INTENT_DEPOSIT}`,
    };
  }

  // Validate contribution range
  if (contributionRange.min > contributionRange.max) {
    return {
      intent: null as any,
      error: "Contribution range min cannot exceed max",
    };
  }

  if (contributionRange.min <= 0n) {
    return {
      intent: null as any,
      error: "Contribution range min must be positive",
    };
  }

  // Validate domains
  if (preferredDomains.length === 0) {
    return {
      intent: null as any,
      error: "Must specify at least one preferred domain",
    };
  }

  const intentId = generateIntentId(did, currentBlock);

  const intent: PartnershipIntent = {
    intentId,
    did,
    preferredDomains,
    contributionRange,
    preferredTier,
    deposit,
    createdAt: currentBlock,
    expiresAt: currentBlock + INTENT.DECLARATION_WINDOW_BLOCKS,
    status: "pending",
  };

  return { intent };
}

/**
 * Match a counterparty to an intent.
 *
 * Counterparty responds within declaration window (2,222 blocks).
 * Both intent deposits are held until finalization.
 */
export function matchIntent(
  intent: PartnershipIntent,
  counterpartyDid: DID,
  counterpartyIntentId: string | undefined,
  currentBlock: bigint,
): { intent: PartnershipIntent; error?: string } {
  if (intent.status !== "pending") {
    return { intent, error: `Intent is ${intent.status}, not pending` };
  }

  if (currentBlock >= intent.expiresAt) {
    return {
      intent: { ...intent, status: "expired" },
      error: "Intent has expired",
    };
  }

  if (counterpartyDid === intent.did) {
    return { intent, error: "Cannot match with self" };
  }

  return {
    intent: {
      ...intent,
      status: "matched",
      matchedWith: counterpartyDid,
      matchedIntentId: counterpartyIntentId,
    },
  };
}

/**
 * Transition a matched intent into finalization phase.
 *
 * This starts the standard 77-block formation window for deposits
 * and finalization.
 */
export function finalizeFromIntent(
  intent: PartnershipIntent,
  currentBlock: bigint,
): { intent: PartnershipIntent; finalizationExpiry: bigint; error?: string } {
  if (intent.status !== "matched") {
    return {
      intent,
      finalizationExpiry: 0n,
      error: `Intent is ${intent.status}, not matched`,
    };
  }

  const finalizationExpiry = currentBlock + INTENT.FINALIZATION_WINDOW_BLOCKS;

  return {
    intent: { ...intent, status: "finalizing" },
    finalizationExpiry,
  };
}

/**
 * Cancel an intent and refund the deposit.
 *
 * Can cancel during declaration window (before match).
 * After match, both parties must agree to cancel.
 */
export function cancelIntent(
  intent: PartnershipIntent,
  currentBlock: bigint,
): { intent: PartnershipIntent; refund: bigint; error?: string } {
  if (intent.status === "expired" || intent.status === "cancelled") {
    return {
      intent,
      refund: 0n,
      error: `Intent already ${intent.status}`,
    };
  }

  if (intent.status === "finalizing") {
    return {
      intent,
      refund: 0n,
      error:
        "Cannot cancel during finalization — use standard formation cancellation",
    };
  }

  return {
    intent: { ...intent, status: "cancelled" },
    refund: intent.deposit,
  };
}

/**
 * Expire an intent that has exceeded its declaration window.
 * Returns deposit for refund.
 */
export function expireIntent(
  intent: PartnershipIntent,
  currentBlock: bigint,
): { intent: PartnershipIntent; refund: bigint } {
  if (intent.status !== "pending" && intent.status !== "matched") {
    return { intent, refund: 0n };
  }

  if (currentBlock < intent.expiresAt) {
    return { intent, refund: 0n };
  }

  return {
    intent: { ...intent, status: "expired" },
    refund: intent.deposit,
  };
}

/**
 * Check if an intent has expired
 */
export function isIntentExpired(
  intent: PartnershipIntent,
  currentBlock: bigint,
): boolean {
  return (
    currentBlock >= intent.expiresAt &&
    (intent.status === "pending" || intent.status === "matched")
  );
}

// ============ Helpers ============

function generateIntentId(did: DID, currentBlock: bigint): string {
  const input = `intent:${did}:${currentBlock.toString()}`;
  return `intent:${sha512_256Hex(input).slice(0, 24)}`;
}
