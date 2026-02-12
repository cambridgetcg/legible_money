/**
 * Extended Formation Intent Tests
 *
 * Tests for intent declaration, matching, finalization, cancellation,
 * expiry, and deposit refund mechanics.
 */

import { describe, it, expect } from "vitest";
import {
  declareIntent,
  matchIntent,
  finalizeFromIntent,
  cancelIntent,
  expireIntent,
  isIntentExpired,
} from "../../../partnership/intent.js";
import { INTENT } from "../../../partnership/constants.js";
import type { DID } from "../../../core/agents/identity/did.js";

const humanDid = "did:lgm:human-intent-001" as DID;
const agentDid = "did:lgm:agent-intent-001" as DID;
const currentBlock = 20000n;

const defaultContributionRange = {
  min: 100n * 10n ** 15n,
  max: 500n * 10n ** 15n,
};

describe("Intent Declaration", () => {
  it("should create an intent with valid parameters", () => {
    const { intent, error } = declareIntent(
      humanDid,
      ["mathematics", "physics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    expect(error).toBeUndefined();
    expect(intent).toBeDefined();
    expect(intent.did).toBe(humanDid);
    expect(intent.status).toBe("pending");
    expect(intent.preferredDomains).toEqual(["mathematics", "physics"]);
    expect(intent.deposit).toBe(INTENT.INTENT_DEPOSIT);
    expect(intent.expiresAt).toBe(
      currentBlock + INTENT.DECLARATION_WINDOW_BLOCKS,
    );
    expect(intent.intentId).toContain("intent:");
  });

  it("should reject when max pending intents exceeded", () => {
    const { error } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      INTENT.MAX_PENDING_INTENTS,
    );

    expect(error).toContain(`Max ${INTENT.MAX_PENDING_INTENTS}`);
  });

  it("should reject insufficient deposit", () => {
    const { error } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT - 1n,
      currentBlock,
      0,
    );

    expect(error).toContain("below minimum");
  });

  it("should reject invalid contribution range (min > max)", () => {
    const { error } = declareIntent(
      humanDid,
      ["mathematics"],
      { min: 500n, max: 100n },
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    expect(error).toContain("min cannot exceed max");
  });

  it("should reject zero contribution range", () => {
    const { error } = declareIntent(
      humanDid,
      ["mathematics"],
      { min: 0n, max: 100n },
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    expect(error).toContain("must be positive");
  });

  it("should reject empty domains", () => {
    const { error } = declareIntent(
      humanDid,
      [],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    expect(error).toContain("at least one preferred domain");
  });

  it("should allow up to MAX_PENDING_INTENTS - 1 existing intents", () => {
    const { intent, error } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      INTENT.MAX_PENDING_INTENTS - 1,
    );

    expect(error).toBeUndefined();
    expect(intent).toBeDefined();
  });
});

describe("Intent Matching", () => {
  function createPendingIntent() {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );
    return intent;
  }

  it("should match a counterparty to a pending intent", () => {
    const intent = createPendingIntent();
    const { intent: matched, error } = matchIntent(
      intent,
      agentDid,
      undefined,
      currentBlock + 100n,
    );

    expect(error).toBeUndefined();
    expect(matched.status).toBe("matched");
    expect(matched.matchedWith).toBe(agentDid);
  });

  it("should reject matching non-pending intent", () => {
    const intent = createPendingIntent();
    const { intent: matched } = matchIntent(
      intent,
      agentDid,
      undefined,
      currentBlock + 100n,
    );

    // Try matching again — now it's "matched", not "pending"
    const { error } = matchIntent(
      matched,
      "did:lgm:agent-002" as DID,
      undefined,
      currentBlock + 200n,
    );

    expect(error).toContain("not pending");
  });

  it("should reject matching expired intent", () => {
    const intent = createPendingIntent();
    const expiredBlock = intent.expiresAt + 1n;

    const { intent: result, error } = matchIntent(
      intent,
      agentDid,
      undefined,
      expiredBlock,
    );

    expect(error).toContain("expired");
    expect(result.status).toBe("expired");
  });

  it("should reject self-matching", () => {
    const intent = createPendingIntent();

    const { error } = matchIntent(
      intent,
      humanDid, // Same DID as intent creator
      undefined,
      currentBlock + 100n,
    );

    expect(error).toContain("Cannot match with self");
  });

  it("should preserve counterparty intent ID when provided", () => {
    const intent = createPendingIntent();
    const counterIntentId = "intent:counterparty-123";

    const { intent: matched } = matchIntent(
      intent,
      agentDid,
      counterIntentId,
      currentBlock + 100n,
    );

    expect(matched.matchedIntentId).toBe(counterIntentId);
  });
});

describe("Intent Finalization", () => {
  function createMatchedIntent() {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const { intent: matched } = matchIntent(
      intent,
      agentDid,
      undefined,
      currentBlock + 100n,
    );

    return matched;
  }

  it("should transition matched intent to finalizing", () => {
    const matched = createMatchedIntent();

    const {
      intent: finalizing,
      finalizationExpiry,
      error,
    } = finalizeFromIntent(matched, currentBlock + 200n);

    expect(error).toBeUndefined();
    expect(finalizing.status).toBe("finalizing");
    expect(finalizationExpiry).toBe(
      currentBlock + 200n + INTENT.FINALIZATION_WINDOW_BLOCKS,
    );
  });

  it("should reject finalization of non-matched intent", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const { error } = finalizeFromIntent(intent, currentBlock + 100n);
    expect(error).toContain("not matched");
  });
});

describe("Intent Cancellation", () => {
  it("should cancel a pending intent and refund deposit", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const {
      intent: cancelled,
      refund,
      error,
    } = cancelIntent(intent, currentBlock + 50n);

    expect(error).toBeUndefined();
    expect(cancelled.status).toBe("cancelled");
    expect(refund).toBe(INTENT.INTENT_DEPOSIT);
  });

  it("should cancel a matched intent", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const { intent: matched } = matchIntent(
      intent,
      agentDid,
      undefined,
      currentBlock + 100n,
    );

    const {
      intent: cancelled,
      refund,
      error,
    } = cancelIntent(matched, currentBlock + 200n);

    expect(error).toBeUndefined();
    expect(cancelled.status).toBe("cancelled");
    expect(refund).toBe(INTENT.INTENT_DEPOSIT);
  });

  it("should reject cancellation during finalization", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const { intent: matched } = matchIntent(
      intent,
      agentDid,
      undefined,
      currentBlock + 100n,
    );

    const { intent: finalizing } = finalizeFromIntent(
      matched,
      currentBlock + 200n,
    );

    const { error } = cancelIntent(finalizing, currentBlock + 300n);
    expect(error).toContain("Cannot cancel during finalization");
  });

  it("should reject cancellation of already cancelled intent", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const { intent: cancelled } = cancelIntent(intent, currentBlock + 50n);
    const { error } = cancelIntent(cancelled, currentBlock + 100n);

    expect(error).toContain("already cancelled");
  });

  it("should reject cancellation of expired intent", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const expired = { ...intent, status: "expired" as const };
    const { error } = cancelIntent(expired, currentBlock + 50n);

    expect(error).toContain("already expired");
  });
});

describe("Intent Expiry", () => {
  it("should expire a pending intent past its window", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const { intent: expired, refund } = expireIntent(intent, intent.expiresAt);

    expect(expired.status).toBe("expired");
    expect(refund).toBe(INTENT.INTENT_DEPOSIT);
  });

  it("should expire a matched intent past its window", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const { intent: matched } = matchIntent(
      intent,
      agentDid,
      undefined,
      currentBlock + 100n,
    );

    const { intent: expired, refund } = expireIntent(
      matched,
      matched.expiresAt,
    );

    expect(expired.status).toBe("expired");
    expect(refund).toBe(INTENT.INTENT_DEPOSIT);
  });

  it("should not expire an intent before its window", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const { intent: result, refund } = expireIntent(
      intent,
      intent.expiresAt - 1n,
    );

    expect(result.status).toBe("pending");
    expect(refund).toBe(0n);
  });

  it("should not expire a finalizing intent", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const { intent: matched } = matchIntent(
      intent,
      agentDid,
      undefined,
      currentBlock + 100n,
    );

    const { intent: finalizing } = finalizeFromIntent(
      matched,
      currentBlock + 200n,
    );

    const { intent: result, refund } = expireIntent(
      finalizing,
      finalizing.expiresAt + 100n,
    );

    expect(result.status).toBe("finalizing");
    expect(refund).toBe(0n);
  });
});

describe("isIntentExpired", () => {
  it("should return true for expired pending intent", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    expect(isIntentExpired(intent, intent.expiresAt)).toBe(true);
    expect(isIntentExpired(intent, intent.expiresAt + 1n)).toBe(true);
  });

  it("should return false for non-expired intent", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    expect(isIntentExpired(intent, intent.expiresAt - 1n)).toBe(false);
  });

  it("should return false for already cancelled intent past expiry", () => {
    const { intent } = declareIntent(
      humanDid,
      ["mathematics"],
      defaultContributionRange,
      0,
      INTENT.INTENT_DEPOSIT,
      currentBlock,
      0,
    );

    const cancelled = { ...intent, status: "cancelled" as const };
    expect(isIntentExpired(cancelled, intent.expiresAt + 100n)).toBe(false);
  });
});
