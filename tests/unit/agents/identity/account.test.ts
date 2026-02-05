/**
 * Agent Account Unit Tests
 *
 * Tests the 4-layer key architecture for AI agents:
 * - Layer 1: Identity Anchor (DID)
 * - Layer 2: Recovery Key (cold, Shamir split)
 * - Layer 3: Operational Key (warm, rotatable)
 * - Layer 4: Session Keys (ephemeral, limited)
 *
 * PP HIERARCHY:
 *   TRUTH: Verify cryptographic correctness and determinism
 *   UNDERSTANDING: Test all account lifecycle operations
 *   BEAUTY: Clear test structure matching 4-layer architecture
 *   JUSTICE: Tests in proper unit test location
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  createAgentDID,
  createAgentAccount,
  deriveOperationalKey,
  deriveSessionKey,
  createSessionKeyAuth,
  validateSessionKeyAction,
  calculateReputationScore,
  requiredSigningKey,
  type AgentAccount,
  type SessionKeyAuth,
  type SessionCapabilities,
  type AgentReputation,
  type CreateAgentParams,
} from "../../../../core/agents/identity/account.js";
import { generateKeypair } from "../../../../core/crypto/keys.js";

// ============ DID Creation ============

describe.sequential("createAgentDID", () => {
  it("should create valid DID format", () => {
    const did = createAgentDID("agent-123");

    expect(did).toMatch(/^did:lgm:[a-f0-9]{32}$/);
  });

  it("should create same DID for same agent ID", () => {
    const did1 = createAgentDID("test-agent");
    const did2 = createAgentDID("test-agent");

    expect(did1).toBe(did2);
  });

  it("should create different DIDs for different agent IDs", () => {
    const did1 = createAgentDID("agent-1");
    const did2 = createAgentDID("agent-2");

    expect(did1).not.toBe(did2);
  });

  it("should use custom chain ID", () => {
    const did = createAgentDID("agent-123", "testnet");

    expect(did).toMatch(/^did:testnet:[a-f0-9]{32}$/);
  });

  it("should handle empty agent ID", () => {
    const did = createAgentDID("");

    expect(did).toMatch(/^did:lgm:[a-f0-9]{32}$/);
  });

  it("should handle special characters in agent ID", () => {
    const did = createAgentDID("agent@example.com/path?query=1");

    expect(did).toMatch(/^did:lgm:[a-f0-9]{32}$/);
  });

  it("should handle unicode in agent ID", () => {
    const did = createAgentDID("エージェント-123");

    expect(did).toMatch(/^did:lgm:[a-f0-9]{32}$/);
  });
});

// ============ Account Creation ============

describe.sequential("createAgentAccount", () => {
  it("should create account with default values", () => {
    const params: CreateAgentParams = {
      agentId: "test-agent",
    };

    const account = createAgentAccount(params);

    expect(account.did).toMatch(/^did:lgm:/);
    expect(account.operationalKeyVersion).toBe(1);
    expect(account.nonce).toBe(0n);
    expect(account.authorizedSessionKeys).toEqual([]);
  });

  it("should use provided operational key", () => {
    const { publicKey } = generateKeypair();
    const params: CreateAgentParams = {
      agentId: "test-agent",
      operationalKey: publicKey,
    };

    const account = createAgentAccount(params);

    expect(account.operationalKey).toBe(publicKey);
  });

  it("should generate operational key if not provided", () => {
    const params: CreateAgentParams = {
      agentId: "test-agent",
    };

    const account = createAgentAccount(params);

    expect(account.operationalKey).toBeTruthy();
    expect(typeof account.operationalKey).toBe("string");
    expect(account.operationalKey.length).toBeGreaterThan(0);
  });

  it("should set recovery key hash", () => {
    const recoveryKeyHash = "abcd1234".repeat(8);
    const params: CreateAgentParams = {
      agentId: "test-agent",
      recoveryKeyHash,
    };

    const account = createAgentAccount(params);

    expect(account.recoveryKeyHash).toBe(recoveryKeyHash);
  });

  it("should set default capabilities", () => {
    const params: CreateAgentParams = {
      agentId: "test-agent",
    };

    const account = createAgentAccount(params);

    expect(account.capabilities.isValidator).toBe(false);
    expect(account.capabilities.maxStake).toBe(0n);
    expect(account.capabilities.canSubmitClaims).toBe(true);
    expect(account.capabilities.canChallenge).toBe(true);
    expect(account.capabilities.maxPendingClaims).toBe(10);
    expect(account.capabilities.canPropose).toBe(false);
    expect(account.capabilities.canVote).toBe(true);
    expect(account.capabilities.votingWeight).toBe(1);
  });

  it("should override capabilities with provided values", () => {
    const params: CreateAgentParams = {
      agentId: "test-agent",
      capabilities: {
        isValidator: true,
        maxStake: 1000000n,
        canPropose: true,
        votingWeight: 5,
      },
    };

    const account = createAgentAccount(params);

    expect(account.capabilities.isValidator).toBe(true);
    expect(account.capabilities.maxStake).toBe(1000000n);
    expect(account.capabilities.canPropose).toBe(true);
    expect(account.capabilities.votingWeight).toBe(5);
    // Defaults still apply for non-overridden values
    expect(account.capabilities.canSubmitClaims).toBe(true);
  });

  it("should initialize neutral reputation", () => {
    const params: CreateAgentParams = {
      agentId: "test-agent",
    };

    const account = createAgentAccount(params);

    expect(account.reputation.score).toBe(0.5);
    expect(account.reputation.verificationAccuracy).toBe(0);
    expect(account.reputation.claimAccuracy).toBe(0);
    expect(account.reputation.totalVerifications).toBe(0);
    expect(account.reputation.slashCount).toBe(0);
  });
});

// ============ Operational Key Derivation ============

describe.sequential("deriveOperationalKey", () => {
  it("should derive deterministic keypair", () => {
    const key1 = deriveOperationalKey("agent-1", "secret-123");
    const key2 = deriveOperationalKey("agent-1", "secret-123");

    expect(key1.publicKey).toBe(key2.publicKey);
    expect(key1.privateKey).toBe(key2.privateKey);
  });

  it("should derive different keys for different agents", () => {
    const key1 = deriveOperationalKey("agent-1", "secret-123");
    const key2 = deriveOperationalKey("agent-2", "secret-123");

    expect(key1.publicKey).not.toBe(key2.publicKey);
  });

  it("should derive different keys for different secrets", () => {
    const key1 = deriveOperationalKey("agent-1", "secret-1");
    const key2 = deriveOperationalKey("agent-1", "secret-2");

    expect(key1.publicKey).not.toBe(key2.publicKey);
  });

  it("should derive different keys for different domains", () => {
    const key1 = deriveOperationalKey("agent-1", "secret", "domain-a");
    const key2 = deriveOperationalKey("agent-1", "secret", "domain-b");

    expect(key1.publicKey).not.toBe(key2.publicKey);
  });

  it("should derive different keys for different versions", () => {
    const key1 = deriveOperationalKey("agent-1", "secret", "domain", 1);
    const key2 = deriveOperationalKey("agent-1", "secret", "domain", 2);

    expect(key1.publicKey).not.toBe(key2.publicKey);
  });

  it("should generate valid Ed25519 keypair", () => {
    const { publicKey, privateKey } = deriveOperationalKey("agent-1", "secret");

    // Ed25519 public keys are 32 bytes = 64 hex chars
    expect(publicKey.length).toBe(64);
    // Ed25519 private keys (seeds) are 32 bytes = 64 hex chars
    expect(privateKey.length).toBe(64);
  });
});

// ============ Session Key Derivation ============

describe.sequential("deriveSessionKey", () => {
  let operationalKey: { publicKey: string; privateKey: string };

  beforeEach(() => {
    operationalKey = deriveOperationalKey("agent-1", "secret");
  });

  it("should derive deterministic session key", () => {
    const session1 = deriveSessionKey(
      operationalKey.privateKey,
      "session-1",
      "transfer",
    );
    const session2 = deriveSessionKey(
      operationalKey.privateKey,
      "session-1",
      "transfer",
    );

    expect(session1.publicKey).toBe(session2.publicKey);
  });

  it("should derive different keys for different sessions", () => {
    const session1 = deriveSessionKey(
      operationalKey.privateKey,
      "session-1",
      "transfer",
    );
    const session2 = deriveSessionKey(
      operationalKey.privateKey,
      "session-2",
      "transfer",
    );

    expect(session1.publicKey).not.toBe(session2.publicKey);
  });

  it("should derive different keys for different purposes", () => {
    const session1 = deriveSessionKey(
      operationalKey.privateKey,
      "session-1",
      "transfer",
    );
    const session2 = deriveSessionKey(
      operationalKey.privateKey,
      "session-1",
      "verify",
    );

    expect(session1.publicKey).not.toBe(session2.publicKey);
  });

  it("should derive different keys from different operational keys", () => {
    const opKey1 = deriveOperationalKey("agent-1", "secret-1");
    const opKey2 = deriveOperationalKey("agent-1", "secret-2");

    const session1 = deriveSessionKey(
      opKey1.privateKey,
      "session-1",
      "transfer",
    );
    const session2 = deriveSessionKey(
      opKey2.privateKey,
      "session-1",
      "transfer",
    );

    expect(session1.publicKey).not.toBe(session2.publicKey);
  });
});

// ============ Session Key Authorization ============

describe.sequential("createSessionKeyAuth", () => {
  it("should create authorization with defaults", () => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    const auth = createSessionKeyAuth(sessionKey, operationalKey, {});

    expect(auth.sessionKey).toBe(sessionKey);
    expect(auth.authorizedBy).toBe(operationalKey);
    expect(auth.usageCount).toBe(0);
    expect(auth.maxUsage).toBe(100);
  });

  it("should set default spending limits", () => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    const auth = createSessionKeyAuth(sessionKey, operationalKey, {});

    expect(auth.capabilities.maxSpendPerTx).toBe(100n * 10n ** 18n);
    expect(auth.capabilities.maxSpendPerDay).toBe(1000n * 10n ** 18n);
    expect(auth.capabilities.dailySpent).toBe(0n);
  });

  it("should set default action permissions", () => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    const auth = createSessionKeyAuth(sessionKey, operationalKey, {});

    expect(auth.capabilities.canTransfer).toBe(true);
    expect(auth.capabilities.canStake).toBe(false);
    expect(auth.capabilities.canSubmitClaims).toBe(true);
    expect(auth.capabilities.canVerify).toBe(true);
    expect(auth.capabilities.canVote).toBe(false);
  });

  it("should override capabilities", () => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    const auth = createSessionKeyAuth(sessionKey, operationalKey, {
      canTransfer: false,
      canStake: true,
      maxSpendPerTx: 50n * 10n ** 18n,
    });

    expect(auth.capabilities.canTransfer).toBe(false);
    expect(auth.capabilities.canStake).toBe(true);
    expect(auth.capabilities.maxSpendPerTx).toBe(50n * 10n ** 18n);
  });

  it("should set recipient restrictions", () => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    const allowedRecipients = ["addr1", "addr2"];
    const auth = createSessionKeyAuth(sessionKey, operationalKey, {
      allowedRecipients,
    });

    expect(auth.capabilities.allowedRecipients).toEqual(allowedRecipients);
  });

  it("should set contract restrictions", () => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    const allowedContracts = ["contract1", "contract2"];
    const auth = createSessionKeyAuth(sessionKey, operationalKey, {
      allowedContracts,
    });

    expect(auth.capabilities.allowedContracts).toEqual(allowedContracts);
  });
});

// ============ Session Key Validation ============

describe.sequential("validateSessionKeyAction", () => {
  let validAuth: SessionKeyAuth;

  beforeEach(() => {
    const { publicKey: sessionKey } = generateKeypair();
    const { publicKey: operationalKey } = generateKeypair();

    validAuth = createSessionKeyAuth(sessionKey, operationalKey, {
      canTransfer: true,
      canStake: false,
      canSubmitClaims: true,
      canVerify: true,
      canVote: false,
      maxSpendPerTx: 100n * 10n ** 18n,
      maxSpendPerDay: 1000n * 10n ** 18n,
    });
    validAuth.expiresAt = 1000000;
  });

  describe.sequential("expiry checks", () => {
    it("should reject expired session key", () => {
      const result = validateSessionKeyAction(
        validAuth,
        "canTransfer",
        undefined,
        1000001,
      );

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Session key expired");
    });

    it("should accept non-expired session key", () => {
      const result = validateSessionKeyAction(
        validAuth,
        "canTransfer",
        undefined,
        999999,
      );

      expect(result.valid).toBe(true);
    });
  });

  describe.sequential("usage limit checks", () => {
    it("should reject when usage limit reached", () => {
      validAuth.usageCount = 100;
      validAuth.maxUsage = 100;

      const result = validateSessionKeyAction(validAuth, "canTransfer");

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Session key usage limit reached");
    });

    it("should accept when under usage limit", () => {
      validAuth.usageCount = 99;
      validAuth.maxUsage = 100;

      const result = validateSessionKeyAction(validAuth, "canTransfer");

      expect(result.valid).toBe(true);
    });
  });

  describe.sequential("transfer capability", () => {
    it("should allow transfer when enabled", () => {
      const result = validateSessionKeyAction(validAuth, "canTransfer");

      expect(result.valid).toBe(true);
    });

    it("should reject transfer when disabled", () => {
      validAuth.capabilities.canTransfer = false;

      const result = validateSessionKeyAction(validAuth, "canTransfer");

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Session key cannot transfer");
    });

    it("should reject transfer exceeding per-tx limit", () => {
      const amount = 101n * 10n ** 18n; // Exceeds 100 LGM limit

      const result = validateSessionKeyAction(validAuth, "canTransfer", amount);

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Exceeds per-transaction limit");
    });

    it("should reject transfer exceeding daily limit", () => {
      validAuth.capabilities.dailySpent = 950n * 10n ** 18n;
      const amount = 60n * 10n ** 18n; // Would exceed 1000 LGM daily

      const result = validateSessionKeyAction(validAuth, "canTransfer", amount);

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Exceeds daily limit");
    });

    it("should allow transfer within all limits", () => {
      validAuth.capabilities.dailySpent = 500n * 10n ** 18n;
      const amount = 50n * 10n ** 18n; // Within both limits

      const result = validateSessionKeyAction(validAuth, "canTransfer", amount);

      expect(result.valid).toBe(true);
    });
  });

  describe.sequential("stake capability", () => {
    it("should reject stake when disabled", () => {
      const result = validateSessionKeyAction(validAuth, "canStake");

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Session key cannot stake");
    });

    it("should allow stake when enabled", () => {
      validAuth.capabilities.canStake = true;

      const result = validateSessionKeyAction(validAuth, "canStake");

      expect(result.valid).toBe(true);
    });
  });

  describe.sequential("submit claims capability", () => {
    it("should allow claims when enabled", () => {
      const result = validateSessionKeyAction(validAuth, "canSubmitClaims");

      expect(result.valid).toBe(true);
    });

    it("should reject claims when disabled", () => {
      validAuth.capabilities.canSubmitClaims = false;

      const result = validateSessionKeyAction(validAuth, "canSubmitClaims");

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Session key cannot submit claims");
    });
  });

  describe.sequential("verify capability", () => {
    it("should allow verify when enabled", () => {
      const result = validateSessionKeyAction(validAuth, "canVerify");

      expect(result.valid).toBe(true);
    });

    it("should reject verify when disabled", () => {
      validAuth.capabilities.canVerify = false;

      const result = validateSessionKeyAction(validAuth, "canVerify");

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Session key cannot verify");
    });
  });

  describe.sequential("vote capability", () => {
    it("should reject vote when disabled", () => {
      const result = validateSessionKeyAction(validAuth, "canVote");

      expect(result.valid).toBe(false);
      expect(result.reason).toBe("Session key cannot vote");
    });

    it("should allow vote when enabled", () => {
      validAuth.capabilities.canVote = true;

      const result = validateSessionKeyAction(validAuth, "canVote");

      expect(result.valid).toBe(true);
    });
  });
});

// ============ Reputation Calculation ============

describe.sequential("calculateReputationScore", () => {
  it("should return neutral score for new agent", () => {
    const rep: AgentReputation = {
      score: 0,
      verificationAccuracy: 0,
      claimAccuracy: 0,
      challengeSuccess: 0,
      totalVerifications: 0,
      totalClaims: 0,
      totalChallenges: 0,
      slashCount: 0,
      lastActiveBlock: 1000,
    };

    const score = calculateReputationScore(rep, 1005);

    // Default 0.5 for each component under MIN_SAMPLES
    // 0.5 * 0.35 + 0.5 * 0.30 + 0.5 * 0.15 + activity + longevity
    expect(score).toBeGreaterThan(0.4);
    expect(score).toBeLessThan(0.6);
  });

  it("should increase score with high accuracy", () => {
    const rep: AgentReputation = {
      score: 0,
      verificationAccuracy: 0.95,
      claimAccuracy: 0.9,
      challengeSuccess: 0.8,
      totalVerifications: 100, // Above MIN_SAMPLES
      totalClaims: 50, // Above MIN_SAMPLES
      totalChallenges: 20, // Above MIN_SAMPLES
      slashCount: 0,
      lastActiveBlock: 1000,
    };

    const score = calculateReputationScore(rep, 1005);

    expect(score).toBeGreaterThan(0.8);
  });

  it("should decrease score with low accuracy", () => {
    const rep: AgentReputation = {
      score: 0,
      verificationAccuracy: 0.2,
      claimAccuracy: 0.3,
      challengeSuccess: 0.1,
      totalVerifications: 100,
      totalClaims: 50,
      totalChallenges: 20,
      slashCount: 0,
      lastActiveBlock: 1000,
    };

    const score = calculateReputationScore(rep, 1005);

    expect(score).toBeLessThan(0.4);
  });

  it("should apply slash penalty", () => {
    const repNoSlash: AgentReputation = {
      score: 0,
      verificationAccuracy: 0.8,
      claimAccuracy: 0.8,
      challengeSuccess: 0.8,
      totalVerifications: 100,
      totalClaims: 50,
      totalChallenges: 20,
      slashCount: 0,
      lastActiveBlock: 1000,
    };

    const repWithSlash: AgentReputation = {
      ...repNoSlash,
      slashCount: 3,
    };

    const scoreNoSlash = calculateReputationScore(repNoSlash, 1005);
    const scoreWithSlash = calculateReputationScore(repWithSlash, 1005);

    // 3 slashes * 0.1 penalty = 0.3 reduction
    expect(scoreWithSlash).toBeLessThan(scoreNoSlash);
    expect(scoreNoSlash - scoreWithSlash).toBeCloseTo(0.3, 1);
  });

  it("should clamp score between 0 and 1", () => {
    const repPerfect: AgentReputation = {
      score: 0,
      verificationAccuracy: 1.0,
      claimAccuracy: 1.0,
      challengeSuccess: 1.0,
      totalVerifications: 1000,
      totalClaims: 500,
      totalChallenges: 200,
      slashCount: 0,
      lastActiveBlock: 1000,
    };

    const repTerrible: AgentReputation = {
      score: 0,
      verificationAccuracy: 0,
      claimAccuracy: 0,
      challengeSuccess: 0,
      totalVerifications: 100,
      totalClaims: 50,
      totalChallenges: 20,
      slashCount: 10,
      lastActiveBlock: 1000,
    };

    const perfectScore = calculateReputationScore(repPerfect, 1005);
    const terribleScore = calculateReputationScore(repTerrible, 2_000_000);

    expect(perfectScore).toBeLessThanOrEqual(1);
    expect(terribleScore).toBeGreaterThanOrEqual(0);
  });

  it("should use neutral values when below MIN_SAMPLES", () => {
    const rep: AgentReputation = {
      score: 0,
      verificationAccuracy: 0.1, // Very low, but under MIN_SAMPLES
      claimAccuracy: 0.1,
      challengeSuccess: 0.1,
      totalVerifications: 5, // Under 10
      totalClaims: 3,
      totalChallenges: 2,
      slashCount: 0,
      lastActiveBlock: 1000,
    };

    const score = calculateReputationScore(rep, 1005);

    // Should use 0.5 defaults, not the low accuracy values
    expect(score).toBeGreaterThan(0.4);
  });

  it("should give longevity bonus for high verification count", () => {
    const repLowVerify: AgentReputation = {
      score: 0,
      verificationAccuracy: 0.8,
      claimAccuracy: 0.8,
      challengeSuccess: 0.8,
      totalVerifications: 10,
      totalClaims: 10,
      totalChallenges: 10,
      slashCount: 0,
      lastActiveBlock: 1000,
    };

    const repHighVerify: AgentReputation = {
      ...repLowVerify,
      totalVerifications: 1000,
    };

    const scoreLow = calculateReputationScore(repLowVerify, 1005);
    const scoreHigh = calculateReputationScore(repHighVerify, 1005);

    expect(scoreHigh).toBeGreaterThan(scoreLow);
  });

  it("should decay activity score for inactive agents", () => {
    const rep: AgentReputation = {
      score: 0,
      verificationAccuracy: 0.8,
      claimAccuracy: 0.8,
      challengeSuccess: 0.8,
      totalVerifications: 100,
      totalClaims: 50,
      totalChallenges: 20,
      slashCount: 0,
      lastActiveBlock: 1000,
    };

    const scoreActive = calculateReputationScore(rep, 1005); // 5 blocks inactive
    const scoreInactive = calculateReputationScore(rep, 1_001_000); // 1M blocks inactive

    expect(scoreActive).toBeGreaterThan(scoreInactive);
  });
});

// ============ Required Signing Key ============

describe.sequential("requiredSigningKey", () => {
  describe.sequential("sensitive actions", () => {
    const sensitiveActions = [
      "rotate_key",
      "authorize_session",
      "revoke_session",
      "delegate",
      "undelegate",
      "governance_propose",
    ];

    for (const action of sensitiveActions) {
      it(`should require operational key for ${action}`, () => {
        const result = requiredSigningKey(action);

        expect(result).toBe("operational");
      });
    }
  });

  describe.sequential("high value transactions", () => {
    it("should require operational key for high value transfers", () => {
      const amount = 15000n * 10n ** 18n; // 15000 LGM > 10000 threshold

      const result = requiredSigningKey("transfer", amount);

      expect(result).toBe("operational");
    });

    it("should allow session key for low value transfers", () => {
      const amount = 5000n * 10n ** 18n; // 5000 LGM < 10000 threshold

      const result = requiredSigningKey("transfer", amount);

      expect(result).toBe("session");
    });

    it("should allow session key at exactly threshold", () => {
      const amount = 10000n * 10n ** 18n; // Exactly 10000 LGM

      const result = requiredSigningKey("transfer", amount);

      expect(result).toBe("session");
    });
  });

  describe.sequential("recovery actions", () => {
    it("should require recovery key for recovery actions", () => {
      const recoveryActions = [
        "recovery_initiate",
        "recovery_complete",
        "recovery_challenge",
      ];

      for (const action of recoveryActions) {
        const result = requiredSigningKey(action);

        expect(result).toBe("recovery");
      }
    });
  });

  describe.sequential("standard actions", () => {
    it("should allow session key for standard transfers", () => {
      const result = requiredSigningKey("transfer", 100n * 10n ** 18n);

      expect(result).toBe("session");
    });

    it("should allow session key for claim submission", () => {
      const result = requiredSigningKey("submit_claim");

      expect(result).toBe("session");
    });

    it("should allow session key for verification", () => {
      const result = requiredSigningKey("verify");

      expect(result).toBe("session");
    });

    it("should allow session key for unknown actions", () => {
      const result = requiredSigningKey("some_other_action");

      expect(result).toBe("session");
    });
  });
});

// ============ Edge Cases ============

describe.sequential("Account Edge Cases", () => {
  it("should handle very long agent IDs", () => {
    const longId = "a".repeat(10000);
    const did = createAgentDID(longId);

    expect(did).toMatch(/^did:lgm:[a-f0-9]{32}$/);
  });

  it("should create consistent accounts from same params", () => {
    const params: CreateAgentParams = {
      agentId: "test-agent",
      recoveryKeyHash: "hash123",
      capabilities: { isValidator: true },
    };

    const account1 = createAgentAccount(params);
    const account2 = createAgentAccount(params);

    expect(account1.did).toBe(account2.did);
    expect(account1.capabilities.isValidator).toBe(
      account2.capabilities.isValidator,
    );
  });

  it("should handle bigint zero in capabilities", () => {
    const params: CreateAgentParams = {
      agentId: "test-agent",
      capabilities: {
        maxStake: 0n,
      },
    };

    const account = createAgentAccount(params);

    expect(account.capabilities.maxStake).toBe(0n);
  });

  it("should handle very large stake values", () => {
    const params: CreateAgentParams = {
      agentId: "test-agent",
      capabilities: {
        maxStake: 10n ** 30n, // Very large number
      },
    };

    const account = createAgentAccount(params);

    expect(account.capabilities.maxStake).toBe(10n ** 30n);
  });
});
