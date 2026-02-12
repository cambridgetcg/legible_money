/**
 * Session Keys Unit Tests
 *
 * Tests ephemeral keys with limited capabilities for agent operations.
 * Session keys are designed to be safe even if exposed to developers.
 *
 * PP HIERARCHY:
 *   TRUTH: Verify cryptographic correctness and limit enforcement
 *   UNDERSTANDING: Test all session key operations and edge cases
 *   BEAUTY: Clear test structure matching module organization
 *   JUSTICE: Tests in proper unit test location
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import {
  generateSessionKey,
  deriveSessionKey,
  authorizeSessionKey,
  verifySessionKeyAuthorization,
  validateSessionAction,
  updateSessionUsage,
  generateIntentChallenge,
  createIntentProof,
  verifyIntentProof,
  shouldDiscardSession,
  getSessionRemaining,
  DEFAULT_SESSION_LIMITS,
  CONSERVATIVE_SESSION_LIMITS,
  type SessionKey,
  type SessionLimits,
  type SessionAction,
} from "../../../../core/agents/identity/session_keys.js";
import { generateKeypair } from "../../../../core/crypto/keys.js";
import { deriveOperationalKey } from "../../../../core/agents/identity/account.js";

// ============ Generate Session Key ============

describe.sequential("generateSessionKey", () => {
  it("should generate unique session key", () => {
    const session1 = generateSessionKey();
    const session2 = generateSessionKey();

    expect(session1.publicKey).not.toBe(session2.publicKey);
    expect(session1.privateKey).not.toBe(session2.privateKey);
    expect(session1.id).not.toBe(session2.id);
  });

  it("should set default limits", () => {
    const session = generateSessionKey();

    expect(session.limits.maxPerTransaction).toBe(
      DEFAULT_SESSION_LIMITS.maxPerTransaction,
    );
    expect(session.limits.maxPerDay).toBe(DEFAULT_SESSION_LIMITS.maxPerDay);
    expect(session.limits.maxTransactions).toBe(
      DEFAULT_SESSION_LIMITS.maxTransactions,
    );
  });

  it("should override limits", () => {
    const customLimits: Partial<SessionLimits> = {
      maxPerTransaction: 50n * 10n ** 15n,
      maxTransactions: 10,
    };

    const session = generateSessionKey(customLimits);

    expect(session.limits.maxPerTransaction).toBe(50n * 10n ** 15n);
    expect(session.limits.maxTransactions).toBe(10);
    // Non-overridden defaults still apply
    expect(session.limits.maxPerDay).toBe(DEFAULT_SESSION_LIMITS.maxPerDay);
  });

  it("should set expiration", () => {
    const now = Date.now();
    const expiresInMs = 7200000; // 2 hours

    const session = generateSessionKey({}, expiresInMs);

    expect(session.expiresAt).toBeGreaterThanOrEqual(now + expiresInMs);
    expect(session.expiresAt).toBeLessThan(now + expiresInMs + 1000);
  });

  it("should initialize usage to zero", () => {
    const session = generateSessionKey();

    expect(session.usage.transactionCount).toBe(0);
    expect(session.usage.totalSpent).toBe(0n);
    expect(session.usage.dailySpent).toBe(0n);
    expect(session.usage.lastUsed).toBe(0);
  });

  it("should generate valid 32-char hex session ID", () => {
    const session = generateSessionKey();

    expect(session.id).toMatch(/^[a-f0-9]{32}$/);
  });
});

// ============ Derive Session Key ============

describe.sequential("deriveSessionKey", () => {
  let operationalKey: { publicKey: string; privateKey: string };

  beforeEach(() => {
    operationalKey = deriveOperationalKey("test-agent", "secret-123");
  });

  it("should derive deterministic session key", () => {
    const session1 = deriveSessionKey(
      operationalKey.privateKey,
      "session-seed-1",
    );
    const session2 = deriveSessionKey(
      operationalKey.privateKey,
      "session-seed-1",
    );

    expect(session1.publicKey).toBe(session2.publicKey);
    expect(session1.id).toBe(session2.id);
  });

  it("should derive different keys for different seeds", () => {
    const session1 = deriveSessionKey(operationalKey.privateKey, "seed-a");
    const session2 = deriveSessionKey(operationalKey.privateKey, "seed-b");

    expect(session1.publicKey).not.toBe(session2.publicKey);
  });

  it("should derive different keys from different operational keys", () => {
    const opKey2 = deriveOperationalKey("other-agent", "secret");

    const session1 = deriveSessionKey(operationalKey.privateKey, "same-seed");
    const session2 = deriveSessionKey(opKey2.privateKey, "same-seed");

    expect(session1.publicKey).not.toBe(session2.publicKey);
  });

  it("should apply custom limits", () => {
    const session = deriveSessionKey(operationalKey.privateKey, "seed", {
      maxPerTransaction: 25n * 10n ** 15n,
    });

    expect(session.limits.maxPerTransaction).toBe(25n * 10n ** 15n);
  });

  it("should set custom expiration", () => {
    const now = Date.now();
    const expiresInMs = 1800000; // 30 minutes

    const session = deriveSessionKey(
      operationalKey.privateKey,
      "seed",
      {},
      expiresInMs,
    );

    expect(session.expiresAt).toBeGreaterThanOrEqual(now + expiresInMs);
  });

  it("should generate deterministic session ID from seed", () => {
    const session1 = deriveSessionKey(operationalKey.privateKey, "test-seed");
    const session2 = deriveSessionKey(operationalKey.privateKey, "test-seed");

    expect(session1.id).toBe(session2.id);
  });
});

// ============ Session Key Authorization ============

describe.sequential("authorizeSessionKey", () => {
  let operationalKey: { publicKey: string; privateKey: string };
  let sessionKey: SessionKey;

  beforeEach(() => {
    operationalKey = deriveOperationalKey("test-agent", "secret");
    sessionKey = generateSessionKey();
  });

  it("should create valid authorization", () => {
    const auth = authorizeSessionKey(
      sessionKey.publicKey,
      "did:lgm:testnet:abc123",
      sessionKey.limits,
      operationalKey.privateKey,
    );

    expect(auth.sessionPublicKey).toBe(sessionKey.publicKey);
    expect(auth.agentDid).toBe("did:lgm:testnet:abc123");
    expect(auth.signature).toBeTruthy();
  });

  it("should include limits in authorization", () => {
    const limits: SessionLimits = {
      ...DEFAULT_SESSION_LIMITS,
      maxPerTransaction: 200n * 10n ** 15n,
    };

    const auth = authorizeSessionKey(
      sessionKey.publicKey,
      "did:lgm:test",
      limits,
      operationalKey.privateKey,
    );

    expect(auth.limits.maxPerTransaction).toBe(200n * 10n ** 15n);
  });

  it("should set expiration correctly", () => {
    const now = Date.now();
    const expiresInMs = 1800000;

    const auth = authorizeSessionKey(
      sessionKey.publicKey,
      "did:lgm:test",
      sessionKey.limits,
      operationalKey.privateKey,
      expiresInMs,
    );

    expect(auth.expiresAt).toBeGreaterThanOrEqual(now + expiresInMs);
    expect(auth.expiresAt).toBeLessThan(now + expiresInMs + 1000);
  });
});

// ============ Verify Session Key Authorization ============

describe.sequential("verifySessionKeyAuthorization", () => {
  let operationalKey: { publicKey: string; privateKey: string };
  let sessionKey: SessionKey;

  beforeEach(() => {
    operationalKey = deriveOperationalKey("test-agent", "secret");
    sessionKey = generateSessionKey({}, 3600000); // 1 hour
  });

  it("should verify valid authorization", () => {
    const auth = authorizeSessionKey(
      sessionKey.publicKey,
      "did:lgm:test",
      sessionKey.limits,
      operationalKey.privateKey,
    );

    const result = verifySessionKeyAuthorization(
      auth,
      operationalKey.publicKey,
    );

    expect(result.valid).toBe(true);
  });

  it("should reject expired authorization", () => {
    const auth = authorizeSessionKey(
      sessionKey.publicKey,
      "did:lgm:test",
      sessionKey.limits,
      operationalKey.privateKey,
      -1000, // Already expired
    );

    const result = verifySessionKeyAuthorization(
      auth,
      operationalKey.publicKey,
    );

    expect(result.valid).toBe(false);
    expect(result.reason).toBe("Authorization expired");
  });

  it("should reject authorization with wrong operational key", () => {
    const otherKey = deriveOperationalKey("other-agent", "secret");

    const auth = authorizeSessionKey(
      sessionKey.publicKey,
      "did:lgm:test",
      sessionKey.limits,
      operationalKey.privateKey,
    );

    const result = verifySessionKeyAuthorization(auth, otherKey.publicKey);

    expect(result.valid).toBe(false);
    expect(result.reason).toBe("Invalid signature");
  });

  it("should reject tampered authorization", () => {
    const auth = authorizeSessionKey(
      sessionKey.publicKey,
      "did:lgm:test",
      sessionKey.limits,
      operationalKey.privateKey,
    );

    // Tamper with limits
    auth.limits.maxPerTransaction = 9999n * 10n ** 15n;

    const result = verifySessionKeyAuthorization(
      auth,
      operationalKey.publicKey,
    );

    expect(result.valid).toBe(false);
    expect(result.reason).toBe("Invalid signature");
  });
});

// ============ Validate Session Action ============

describe.sequential("validateSessionAction", () => {
  let session: SessionKey;

  beforeEach(() => {
    session = generateSessionKey(
      {
        allowedActions: ["transfer", "query", "submit_claim"],
        maxPerTransaction: 100n * 10n ** 15n,
        maxPerDay: 500n * 10n ** 15n,
        maxTotal: 1000n * 10n ** 15n,
        maxTransactions: 50,
      },
      3600000,
    );
  });

  it("should allow valid action", () => {
    const result = validateSessionAction(session, "transfer", 50n * 10n ** 15n);

    expect(result.allowed).toBe(true);
  });

  it("should reject expired session", () => {
    session.expiresAt = Date.now() - 1000;

    const result = validateSessionAction(session, "transfer");

    expect(result.allowed).toBe(false);
    expect(result.reason).toContain("Session expired");
  });

  it("should reject when transaction limit reached", () => {
    session.usage.transactionCount = 50;

    const result = validateSessionAction(session, "query");

    expect(result.allowed).toBe(false);
    expect(result.reason).toBe("Transaction limit reached");
  });

  it("should reject disallowed action", () => {
    const result = validateSessionAction(session, "stake");

    expect(result.allowed).toBe(false);
    expect(result.reason).toBe("Action 'stake' not allowed");
  });

  it("should reject transfer exceeding per-tx limit", () => {
    const amount = 150n * 10n ** 15n; // Over 100 LGM limit

    const result = validateSessionAction(session, "transfer", amount);

    expect(result.allowed).toBe(false);
    expect(result.reason).toBe("Exceeds per-transaction limit");
  });

  it("should reject transfer exceeding daily limit", () => {
    session.usage.dailySpent = 450n * 10n ** 15n;
    const amount = 60n * 10n ** 15n; // Would exceed 500 LGM daily

    const result = validateSessionAction(session, "transfer", amount);

    expect(result.allowed).toBe(false);
    expect(result.reason).toBe("Exceeds daily limit");
  });

  it("should reject transfer exceeding total limit", () => {
    session.usage.totalSpent = 950n * 10n ** 15n;
    const amount = 60n * 10n ** 15n; // Would exceed 1000 LGM total

    const result = validateSessionAction(session, "transfer", amount);

    expect(result.allowed).toBe(false);
    expect(result.reason).toBe("Exceeds total limit");
  });

  it("should reset daily spending after 24 hours", () => {
    session.usage.dailySpent = 400n * 10n ** 15n;
    session.usage.lastDailyReset = Date.now() - 86400001; // Over 24 hours ago

    const result = validateSessionAction(session, "transfer", 60n * 10n ** 15n);

    expect(result.allowed).toBe(true);
  });

  it("should allow query without amount check", () => {
    const result = validateSessionAction(session, "query");

    expect(result.allowed).toBe(true);
  });

  it("should allow submit_claim without amount check", () => {
    const result = validateSessionAction(session, "submit_claim");

    expect(result.allowed).toBe(true);
  });
});

// ============ Update Session Usage ============

describe.sequential("updateSessionUsage", () => {
  let session: SessionKey;

  beforeEach(() => {
    session = generateSessionKey();
  });

  it("should increment transaction count", () => {
    const updated = updateSessionUsage(session, "query");

    expect(updated.usage.transactionCount).toBe(1);
  });

  it("should track spending", () => {
    const amount = 50n * 10n ** 15n;
    const updated = updateSessionUsage(session, "transfer", amount);

    expect(updated.usage.dailySpent).toBe(amount);
    expect(updated.usage.totalSpent).toBe(amount);
  });

  it("should accumulate spending", () => {
    const amount = 50n * 10n ** 15n;
    let updated = updateSessionUsage(session, "transfer", amount);
    updated = updateSessionUsage(updated, "transfer", amount);

    expect(updated.usage.dailySpent).toBe(amount * 2n);
    expect(updated.usage.totalSpent).toBe(amount * 2n);
    expect(updated.usage.transactionCount).toBe(2);
  });

  it("should update lastUsed timestamp", () => {
    const now = Date.now();
    const updated = updateSessionUsage(session, "query");

    expect(updated.usage.lastUsed).toBeGreaterThanOrEqual(now);
  });

  it("should reset daily spending after 24 hours", () => {
    session.usage.dailySpent = 100n * 10n ** 15n;
    session.usage.lastDailyReset = Date.now() - 86400001; // Over 24 hours ago

    const updated = updateSessionUsage(session, "transfer", 50n * 10n ** 15n);

    expect(updated.usage.dailySpent).toBe(50n * 10n ** 15n);
  });

  it("should not modify original session", () => {
    const originalCount = session.usage.transactionCount;
    updateSessionUsage(session, "query");

    expect(session.usage.transactionCount).toBe(originalCount);
  });
});

// ============ Intent Proof ============

describe.sequential("Intent Proof", () => {
  let sessionKey: SessionKey;

  beforeEach(() => {
    sessionKey = generateSessionKey();
  });

  describe.sequential("generateIntentChallenge", () => {
    it("should generate deterministic challenge for same block", () => {
      const challenge1 = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce1",
        1000,
      );
      const challenge2 = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce1",
        1000,
      );

      expect(challenge1).toMatch(/^[a-f0-9]{64}$/);
      expect(challenge1).toBe(challenge2);
    });

    it("should generate different challenges for different agents", () => {
      const challenge1 = generateIntentChallenge(
        "did:lgm:agent1",
        "transfer",
        "nonce",
        1000,
      );
      const challenge2 = generateIntentChallenge(
        "did:lgm:agent2",
        "transfer",
        "nonce",
        1000,
      );

      expect(challenge1).not.toBe(challenge2);
    });

    it("should generate different challenges for different actions", () => {
      const challenge1 = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce",
        1000,
      );
      const challenge2 = generateIntentChallenge(
        "did:lgm:test",
        "stake",
        "nonce",
        1000,
      );

      expect(challenge1).not.toBe(challenge2);
    });

    it("should generate different challenges for different blocks", () => {
      const challenge1 = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce",
        1000,
      );
      const challenge2 = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce",
        1001,
      );

      expect(challenge1).not.toBe(challenge2);
    });
  });

  describe.sequential("createIntentProof", () => {
    it("should create valid intent proof", () => {
      const challenge = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce",
        1000,
      );
      const proof = createIntentProof(
        "transfer",
        "send 100 LGM",
        challenge,
        sessionKey.privateKey,
      );

      expect(proof.action).toBe("transfer");
      expect(proof.details).toBe("send 100 LGM");
      expect(proof.challenge).toBe(challenge);
      expect(proof.response).toBeTruthy();
      expect(proof.timestamp).toBeGreaterThan(0);
    });
  });

  describe.sequential("verifyIntentProof", () => {
    it("should verify valid intent proof", () => {
      const challenge = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce",
        1000,
      );
      const proof = createIntentProof(
        "transfer",
        "send 100 LGM",
        challenge,
        sessionKey.privateKey,
      );

      const result = verifyIntentProof(proof, sessionKey.publicKey);

      expect(result.valid).toBe(true);
    });

    it("should reject expired intent proof", () => {
      const challenge = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce",
        1000,
      );
      const proof = createIntentProof(
        "transfer",
        "send 100 LGM",
        challenge,
        sessionKey.privateKey,
      );
      proof.timestamp = Date.now() - 120000; // 2 minutes ago

      const result = verifyIntentProof(proof, sessionKey.publicKey, 60000); // 1 minute max age

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Intent proof expired");
    });

    it("should reject proof with wrong session key", () => {
      const otherSession = generateSessionKey();
      const challenge = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce",
        1000,
      );
      const proof = createIntentProof(
        "transfer",
        "send 100 LGM",
        challenge,
        sessionKey.privateKey,
      );

      const result = verifyIntentProof(proof, otherSession.publicKey);

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Invalid intent proof signature");
    });

    it("should reject tampered intent proof", () => {
      const challenge = generateIntentChallenge(
        "did:lgm:test",
        "transfer",
        "nonce",
        1000,
      );
      const proof = createIntentProof(
        "transfer",
        "send 100 LGM",
        challenge,
        sessionKey.privateKey,
      );
      proof.details = "send 1000000 LGM"; // Tampered

      const result = verifyIntentProof(proof, sessionKey.publicKey);

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Invalid intent proof signature");
    });
  });
});

// ============ Session Cleanup ============

describe.sequential("shouldDiscardSession", () => {
  let session: SessionKey;

  beforeEach(() => {
    session = generateSessionKey(
      {
        maxTransactions: 10,
        maxTotal: 100n * 10n ** 15n,
      },
      3600000,
    );
  });

  it("should discard expired session", () => {
    session.expiresAt = Date.now() - 1000;

    expect(shouldDiscardSession(session)).toBe(true);
  });

  it("should discard session with exhausted transactions", () => {
    session.usage.transactionCount = 10;

    expect(shouldDiscardSession(session)).toBe(true);
  });

  it("should discard session with exhausted spending", () => {
    session.usage.totalSpent = 100n * 10n ** 15n;

    expect(shouldDiscardSession(session)).toBe(true);
  });

  it("should keep valid session", () => {
    session.usage.transactionCount = 5;
    session.usage.totalSpent = 50n * 10n ** 15n;

    expect(shouldDiscardSession(session)).toBe(false);
  });
});

// ============ Get Session Remaining ============

describe.sequential("getSessionRemaining", () => {
  let session: SessionKey;

  beforeEach(() => {
    session = generateSessionKey(
      {
        maxTransactions: 100,
        maxTotal: 1000n * 10n ** 15n,
      },
      3600000,
    );
  });

  it("should calculate remaining time", () => {
    const remaining = getSessionRemaining(session);

    expect(remaining.timeMs).toBeGreaterThan(3590000);
    expect(remaining.timeMs).toBeLessThanOrEqual(3600000);
  });

  it("should calculate remaining transactions", () => {
    session.usage.transactionCount = 30;

    const remaining = getSessionRemaining(session);

    expect(remaining.transactions).toBe(70);
  });

  it("should calculate remaining spending", () => {
    session.usage.totalSpent = 400n * 10n ** 15n;

    const remaining = getSessionRemaining(session);

    expect(remaining.spending).toBe(600n * 10n ** 15n);
  });

  it("should return zero for expired session", () => {
    session.expiresAt = Date.now() - 1000;

    const remaining = getSessionRemaining(session);

    expect(remaining.timeMs).toBe(0);
  });

  it("should return zero for exhausted transactions", () => {
    session.usage.transactionCount = 150; // Over limit

    const remaining = getSessionRemaining(session);

    expect(remaining.transactions).toBe(0);
  });
});

// ============ Default Limits ============

describe.sequential("Default Limits", () => {
  it("DEFAULT_SESSION_LIMITS should have correct values", () => {
    expect(DEFAULT_SESSION_LIMITS.maxPerTransaction).toBe(100n * 10n ** 15n);
    expect(DEFAULT_SESSION_LIMITS.maxPerDay).toBe(1000n * 10n ** 15n);
    expect(DEFAULT_SESSION_LIMITS.maxTotal).toBe(5000n * 10n ** 15n);
    expect(DEFAULT_SESSION_LIMITS.maxTransactions).toBe(100);
    expect(DEFAULT_SESSION_LIMITS.maxDurationMs).toBe(3600000);
    expect(DEFAULT_SESSION_LIMITS.requireIntentProof).toBe(false);
  });

  it("CONSERVATIVE_SESSION_LIMITS should be more restrictive", () => {
    expect(CONSERVATIVE_SESSION_LIMITS.maxPerTransaction).toBeLessThan(
      DEFAULT_SESSION_LIMITS.maxPerTransaction,
    );
    expect(CONSERVATIVE_SESSION_LIMITS.maxPerDay).toBeLessThan(
      DEFAULT_SESSION_LIMITS.maxPerDay,
    );
    expect(CONSERVATIVE_SESSION_LIMITS.maxTransactions).toBeLessThan(
      DEFAULT_SESSION_LIMITS.maxTransactions,
    );
    expect(CONSERVATIVE_SESSION_LIMITS.requireIntentProof).toBe(true);
  });
});

// ============ Edge Cases ============

describe.sequential("Session Keys Edge Cases", () => {
  it("should handle zero amount transfers", () => {
    const session = generateSessionKey();

    const result = validateSessionAction(session, "transfer", 0n);

    expect(result.allowed).toBe(true);
  });

  it("should handle very large amounts correctly", () => {
    const session = generateSessionKey({
      maxPerTransaction: 10n ** 30n,
      maxPerDay: 10n ** 31n,
      maxTotal: 10n ** 32n,
    });

    const result = validateSessionAction(session, "transfer", 10n ** 29n);

    expect(result.allowed).toBe(true);
  });

  it("should handle empty allowed actions", () => {
    const session = generateSessionKey({
      allowedActions: [],
    });

    const result = validateSessionAction(session, "transfer");

    expect(result.allowed).toBe(false);
  });

  it("should handle session at exact expiration boundary", () => {
    const session = generateSessionKey({}, 0);
    session.expiresAt = Date.now() - 1; // 1ms in the past to ensure expiry

    const result = validateSessionAction(session, "query");
    // Session should be expired
    expect(result.allowed).toBe(false);
    expect(result.reason).toContain("Session expired");
  });

  it("should handle max transactions at exact limit", () => {
    const session = generateSessionKey({
      maxTransactions: 1,
    });
    session.usage.transactionCount = 0;

    const result1 = validateSessionAction(session, "query");
    expect(result1.allowed).toBe(true);

    session.usage.transactionCount = 1;
    const result2 = validateSessionAction(session, "query");
    expect(result2.allowed).toBe(false);
  });
});
