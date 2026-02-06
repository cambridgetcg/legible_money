/**
 * AccessManager Unit Tests
 *
 * Tests for the Access Control Manager - key registration, permission verification,
 * session handling, and security enforcement for Agent Homes.
 *
 * PP HIERARCHY:
 *   TRUTH: Tests verify accurate permission and key state
 *   UNDERSTANDING: Clear test structure mirrors access control flow
 *   BEAUTY: Comprehensive coverage with elegant test organization
 *   JUSTICE: Tests placed in appropriate unit test location
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  AccessManager,
  type AccessSummary,
} from "../../../core/agents/home/access.js";
import type {
  AccessControl,
  KeyRegistration,
  Permission,
  AccessRole,
} from "../../../core/agents/home/types.js";

// Test constants
const TEST_DID = "did:lgm:test-agent-access";
const IDENTITY_KEY_HASH = "identity_key_hash_abc123def456";
const OPERATIONAL_KEY_HASH = "operational_key_hash_xyz789";
const SESSION_KEY_HASH = "session_key_hash_qrs456";
const GUARDIAN_KEY_HASH = "guardian_key_hash_tuv789";
const ONE_LGM = 10n ** 18n;

// Deterministic base time for tests (avoids Date.now() dependency)
const BASE_TIME = 1700000000000; // fixed epoch ms

// Helper to create fresh manager with deterministic time
function createManager(): AccessManager {
  const m = new AccessManager(TEST_DID);
  m.setCurrentBlock(1n, BASE_TIME);
  return m;
}

// Helper to create manager with identity key
function createManagerWithIdentity(): AccessManager {
  const manager = new AccessManager(TEST_DID);
  manager.setCurrentBlock(1n, BASE_TIME);
  manager.registerKey(IDENTITY_KEY_HASH, "identity", "owner", {
    label: "Primary",
  });
  return manager;
}

describe.sequential("AccessManager", () => {
  // ============ Initialization Tests ============

  describe("Initialization", () => {
    it("should create access manager with default settings", () => {
      const manager = createManager();
      const summary = manager.getSummary();

      expect(summary.registeredKeys).toBe(0);
      expect(summary.activeSessions).toBe(0);
      expect(summary.securityStatus).toBe("normal");
    });

    it("should create with existing access state", () => {
      const existingAccess: AccessControl = {
        keys: [
          {
            keyHash: IDENTITY_KEY_HASH,
            keyType: "identity",
            role: "owner",
            registeredAt: BASE_TIME - 10000,
            label: "Test Identity",
          },
        ],
        permissions: new Map([
          ["owner", ["read_memory", "write_memory", "transfer"]],
        ]),
        activeSessions: [],
        accessLog: [],
        security: {
          multiSigThreshold: 500000000n,
          keyRotationCooldown: 500n,
          maxSessionDuration: 12 * 60 * 60 * 1000,
          rateLimits: {
            maxActionsPerBlock: 5,
            maxTransfersPerDay: 25,
            maxStakeChangesPerDay: 3,
          },
        },
      };

      const manager = new AccessManager(TEST_DID, existingAccess);
      const summary = manager.getSummary();

      expect(summary.registeredKeys).toBe(1);
      expect(summary.activeKeys).toBe(1);
    });

    it("should have default permissions for all roles", () => {
      const manager = createManager();

      // Register keys with different roles and check permissions
      manager.registerKey(IDENTITY_KEY_HASH, "identity", "owner");
      expect(manager.hasPermission(IDENTITY_KEY_HASH, "archive")).toBe(true);
      expect(manager.hasPermission(IDENTITY_KEY_HASH, "manage_keys")).toBe(
        true,
      );

      manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
      expect(manager.hasPermission(OPERATIONAL_KEY_HASH, "transfer")).toBe(
        true,
      );
      expect(manager.hasPermission(OPERATIONAL_KEY_HASH, "archive")).toBe(
        false,
      );

      manager.registerKey(SESSION_KEY_HASH, "session", "guest");
      expect(manager.hasPermission(SESSION_KEY_HASH, "verify")).toBe(true);
      expect(manager.hasPermission(SESSION_KEY_HASH, "transfer")).toBe(false);
    });
  });

  // ============ Key Management Tests ============

  describe("Key Management", () => {
    describe("registerKey()", () => {
      it("should register identity key as owner", () => {
        const manager = createManager();
        const key = manager.registerKey(
          IDENTITY_KEY_HASH,
          "identity",
          "owner",
          {
            label: "Primary Identity",
          },
        );

        expect(key.keyHash).toBe(IDENTITY_KEY_HASH);
        expect(key.keyType).toBe("identity");
        expect(key.role).toBe("owner");
        expect(key.label).toBe("Primary Identity");
      });

      it("should register operational key", () => {
        const manager = createManager();
        const key = manager.registerKey(
          OPERATIONAL_KEY_HASH,
          "operational",
          "operator",
        );

        expect(key.keyType).toBe("operational");
        expect(key.role).toBe("operator");
      });

      it("should register session key as guest", () => {
        const manager = createManager();
        const key = manager.registerKey(SESSION_KEY_HASH, "session", "guest", {
          expiresAt: BASE_TIME + 3600000,
        });

        expect(key.keyType).toBe("session");
        expect(key.role).toBe("guest");
        expect(key.expiresAt).toBeDefined();
      });

      it("should register guardian key", () => {
        const manager = createManager();
        const key = manager.registerKey(
          GUARDIAN_KEY_HASH,
          "guardian",
          "guardian",
        );

        expect(key.keyType).toBe("guardian");
        expect(key.role).toBe("guardian");
      });

      it("should throw on duplicate key registration", () => {
        const manager = createManager();
        manager.registerKey(IDENTITY_KEY_HASH, "identity", "owner");

        expect(() =>
          manager.registerKey(IDENTITY_KEY_HASH, "identity", "owner"),
        ).toThrow("Key already registered");
      });

      it("should allow re-registering revoked key", () => {
        const manager = createManager();
        manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
        manager.revokeKey(OPERATIONAL_KEY_HASH, "test");

        // Should not throw
        const newKey = manager.registerKey(
          OPERATIONAL_KEY_HASH,
          "operational",
          "operator",
        );
        expect(newKey.keyHash).toBe(OPERATIONAL_KEY_HASH);
      });

      it("should throw on invalid key type/role combination", () => {
        const manager = createManager();

        expect(() =>
          manager.registerKey(IDENTITY_KEY_HASH, "identity", "guest"),
        ).toThrow("cannot be assigned role");

        expect(() =>
          manager.registerKey(SESSION_KEY_HASH, "session", "owner"),
        ).toThrow("cannot be assigned role");

        expect(() =>
          manager.registerKey(GUARDIAN_KEY_HASH, "guardian", "operator"),
        ).toThrow("cannot be assigned role");
      });

      it("should log key registration", () => {
        const manager = createManager();
        manager.registerKey(IDENTITY_KEY_HASH, "identity", "owner");

        const log = manager.getAccessLog();
        expect(log.length).toBeGreaterThan(0);
        expect(log[log.length - 1].action).toBe("register_key");
        expect(log[log.length - 1].success).toBe(true);
      });
    });

    describe("revokeKey()", () => {
      it("should revoke operational key", () => {
        const manager = createManager();
        manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");

        manager.revokeKey(OPERATIONAL_KEY_HASH, "test-revoker");

        const key = manager.getKey(OPERATIONAL_KEY_HASH);
        expect(key).toBeUndefined();
      });

      it("should throw when revoking identity key", () => {
        const manager = createManagerWithIdentity();

        expect(() => manager.revokeKey(IDENTITY_KEY_HASH, "test")).toThrow(
          "Cannot revoke identity key",
        );
      });

      it("should throw when key not found", () => {
        const manager = createManager();

        expect(() => manager.revokeKey("non-existent", "test")).toThrow(
          "Key not found",
        );
      });

      it("should terminate active sessions on revoke", () => {
        const manager = createManager();
        manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
        const session = manager.startSession(OPERATIONAL_KEY_HASH);

        manager.revokeKey(OPERATIONAL_KEY_HASH, "test");

        expect(manager.getSession(session.sessionId)).toBeUndefined();
      });

      it("should log revocation", () => {
        const manager = createManager();
        manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
        manager.revokeKey(OPERATIONAL_KEY_HASH, "test");

        const log = manager.getAccessLog();
        const revokeEntry = log.find((e) => e.action === "revoke_key");
        expect(revokeEntry).toBeDefined();
        expect(revokeEntry?.success).toBe(true);
      });
    });

    describe("rotateKey()", () => {
      it("should rotate operational key", () => {
        const manager = createManager();
        manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator", {
          label: "Original",
        });
        manager.setCurrentBlock(2000n); // Past cooldown

        const newKeyHash = "new_operational_key_hash";
        const newKey = manager.rotateKey(
          OPERATIONAL_KEY_HASH,
          newKeyHash,
          "test-rotator",
        );

        expect(newKey.keyHash).toBe(newKeyHash);
        expect(newKey.role).toBe("operator");
        expect(manager.getKey(OPERATIONAL_KEY_HASH)).toBeUndefined();
      });

      it("should throw when rotating identity key", () => {
        const manager = createManagerWithIdentity();

        expect(() =>
          manager.rotateKey(IDENTITY_KEY_HASH, "new-identity", "test"),
        ).toThrow("Cannot rotate identity key");
      });

      it("should throw when old key not found", () => {
        const manager = createManager();

        expect(() =>
          manager.rotateKey("non-existent", "new-key", "test"),
        ).toThrow("Old key not found");
      });

      it("should enforce cooldown between rotations", () => {
        const manager = createManager();
        manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
        manager.setCurrentBlock(100n);

        // First rotation should work
        const newKey1Hash = "new_key_1";
        manager.rotateKey(OPERATIONAL_KEY_HASH, newKey1Hash, "test");

        // Second rotation within cooldown (1000 blocks) should be rejected
        manager.setCurrentBlock(200n);
        expect(() =>
          manager.rotateKey(newKey1Hash, "new_key_2", "test"),
        ).toThrow("Key rotation cooldown not elapsed");

        // After cooldown elapses, rotation should succeed
        manager.setCurrentBlock(1200n);
        expect(() =>
          manager.rotateKey(newKey1Hash, "new_key_2", "test"),
        ).not.toThrow();
      });
    });

    describe("getActiveKeys()", () => {
      it("should return only active keys", () => {
        const manager = createManager();
        manager.registerKey(IDENTITY_KEY_HASH, "identity", "owner");
        manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
        manager.revokeKey(OPERATIONAL_KEY_HASH, "test");

        const activeKeys = manager.getActiveKeys();
        expect(activeKeys).toHaveLength(1);
        expect(activeKeys[0].keyHash).toBe(IDENTITY_KEY_HASH);
      });

      it("should exclude expired keys", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest", {
          expiresAt: BASE_TIME - 1000, // Already expired relative to BASE_TIME
        });

        const activeKeys = manager.getActiveKeys();
        expect(activeKeys).toHaveLength(0);
      });
    });

    describe("getKey()", () => {
      it("should return key by hash", () => {
        const manager = createManagerWithIdentity();

        const key = manager.getKey(IDENTITY_KEY_HASH);
        expect(key?.keyHash).toBe(IDENTITY_KEY_HASH);
      });

      it("should return undefined for revoked key", () => {
        const manager = createManager();
        manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
        manager.revokeKey(OPERATIONAL_KEY_HASH, "test");

        expect(manager.getKey(OPERATIONAL_KEY_HASH)).toBeUndefined();
      });

      it("should return undefined for non-existent key", () => {
        const manager = createManager();
        expect(manager.getKey("non-existent")).toBeUndefined();
      });
    });
  });

  // ============ Session Management Tests ============

  describe("Session Management", () => {
    describe("startSession()", () => {
      it("should start session with registered key", () => {
        const manager = createManagerWithIdentity();

        const session = manager.startSession(IDENTITY_KEY_HASH);

        expect(session.sessionId).toBeTruthy();
        expect(session.keyHash).toBe(IDENTITY_KEY_HASH);
        expect(session.role).toBe("owner");
        expect(session.actionsPerformed).toBe(0);
        expect(session.spentThisSession).toBe(0n);
      });

      it("should throw for unregistered key", () => {
        const manager = createManager();

        expect(() => manager.startSession("unknown-key")).toThrow(
          "Key not registered",
        );
      });

      it("should throw for expired key", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest", {
          expiresAt: BASE_TIME - 1000,
        });

        expect(() => manager.startSession(SESSION_KEY_HASH)).toThrow(
          "Key has expired",
        );
      });

      it("should respect max session duration", () => {
        const manager = createManagerWithIdentity();

        const session = manager.startSession(IDENTITY_KEY_HASH, {
          sessionDuration: 48 * 60 * 60 * 1000, // 48 hours
        });

        // Should be capped to max (24 hours default)
        const expectedMax = 24 * 60 * 60 * 1000;
        const actualDuration = session.expiresAt - session.startedAt;
        expect(actualDuration).toBeLessThanOrEqual(expectedMax);
      });

      it("should log session start", () => {
        const manager = createManagerWithIdentity();
        manager.startSession(IDENTITY_KEY_HASH);

        const log = manager.getAccessLog();
        const startEntry = log.find((e) => e.action === "session_start");
        expect(startEntry).toBeDefined();
      });
    });

    describe("endSession()", () => {
      it("should end active session", () => {
        const manager = createManagerWithIdentity();
        const session = manager.startSession(IDENTITY_KEY_HASH);

        manager.endSession(session.sessionId);

        expect(manager.getSession(session.sessionId)).toBeUndefined();
      });

      it("should handle non-existent session gracefully", () => {
        const manager = createManager();

        // Should not throw
        manager.endSession("non-existent-session");
      });

      it("should log session end", () => {
        const manager = createManagerWithIdentity();
        const session = manager.startSession(IDENTITY_KEY_HASH);
        manager.endSession(session.sessionId);

        const log = manager.getAccessLog();
        const endEntry = log.find((e) => e.action === "session_end");
        expect(endEntry).toBeDefined();
      });
    });

    describe("getSession()", () => {
      it("should return active session", () => {
        const manager = createManagerWithIdentity();
        const started = manager.startSession(IDENTITY_KEY_HASH);

        const session = manager.getSession(started.sessionId);
        expect(session?.sessionId).toBe(started.sessionId);
      });

      it("should return undefined for ended session", () => {
        const manager = createManagerWithIdentity();
        const session = manager.startSession(IDENTITY_KEY_HASH);
        manager.endSession(session.sessionId);

        expect(manager.getSession(session.sessionId)).toBeUndefined();
      });

      it("should auto-cleanup expired session", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest", {
          expiresAt: BASE_TIME + 100000,
        });

        // Start session with very short duration (1ms)
        const session = manager.startSession(SESSION_KEY_HASH, {
          sessionDuration: 1,
        });

        const sessionId = session.sessionId;

        // Advance deterministic time past session expiry
        manager.setCurrentBlock(2n, BASE_TIME + 10);

        expect(manager.getSession(sessionId)).toBeUndefined();
      });
    });

    describe("recordSessionActivity()", () => {
      it("should update session activity", () => {
        const manager = createManagerWithIdentity();
        const session = manager.startSession(IDENTITY_KEY_HASH);

        manager.recordSessionActivity(session.sessionId, 100n);

        const updated = manager.getSession(session.sessionId);
        expect(updated?.actionsPerformed).toBe(1);
        expect(updated?.spentThisSession).toBe(100n);
      });

      it("should throw for non-existent session", () => {
        const manager = createManager();

        expect(() =>
          manager.recordSessionActivity("non-existent", 100n),
        ).toThrow("Session not found");
      });

      it("should accumulate spent amounts", () => {
        const manager = createManagerWithIdentity();
        const session = manager.startSession(IDENTITY_KEY_HASH);

        manager.recordSessionActivity(session.sessionId, 100n);
        manager.recordSessionActivity(session.sessionId, 200n);
        manager.recordSessionActivity(session.sessionId);

        const updated = manager.getSession(session.sessionId);
        expect(updated?.actionsPerformed).toBe(3);
        expect(updated?.spentThisSession).toBe(300n);
      });
    });

    describe("cleanupExpiredSessions()", () => {
      it("should remove expired sessions", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest", {
          expiresAt: BASE_TIME + 100000,
        });

        // Create a session that will expire (1ms duration)
        const session = manager.startSession(SESSION_KEY_HASH, {
          sessionDuration: 1,
        });

        // Advance time past session expiry (deterministic, no setTimeout)
        manager.setCurrentBlock(2n, BASE_TIME + 100);
        const expired = manager.cleanupExpiredSessions();
        expect(expired).toContain(session.sessionId);
      });

      it("should keep active sessions", () => {
        const manager = createManagerWithIdentity();
        const session = manager.startSession(IDENTITY_KEY_HASH);

        const expired = manager.cleanupExpiredSessions();

        expect(expired).not.toContain(session.sessionId);
        expect(manager.getSession(session.sessionId)).toBeDefined();
      });
    });
  });

  // ============ Permission Tests ============

  describe("Permission Checks", () => {
    describe("hasPermission()", () => {
      it("should return true for valid permission", () => {
        const manager = createManagerWithIdentity();
        expect(manager.hasPermission(IDENTITY_KEY_HASH, "transfer")).toBe(true);
      });

      it("should return false for invalid permission", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest");

        expect(manager.hasPermission(SESSION_KEY_HASH, "transfer")).toBe(false);
      });

      it("should return false for unknown key", () => {
        const manager = createManager();
        expect(manager.hasPermission("unknown", "read_memory")).toBe(false);
      });
    });

    describe("sessionHasPermission()", () => {
      it("should return true for valid session permission", () => {
        const manager = createManagerWithIdentity();
        const session = manager.startSession(IDENTITY_KEY_HASH);

        expect(
          manager.sessionHasPermission(session.sessionId, "transfer"),
        ).toBe(true);
      });

      it("should return false for missing permission", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest");
        const session = manager.startSession(SESSION_KEY_HASH);

        expect(
          manager.sessionHasPermission(session.sessionId, "transfer"),
        ).toBe(false);
      });

      it("should return false for non-existent session", () => {
        const manager = createManager();
        expect(
          manager.sessionHasPermission("non-existent", "read_memory"),
        ).toBe(false);
      });
    });

    describe("requirePermission()", () => {
      it("should not throw for valid permission", () => {
        const manager = createManagerWithIdentity();

        expect(() =>
          manager.requirePermission(IDENTITY_KEY_HASH, "transfer"),
        ).not.toThrow();
      });

      it("should throw for missing permission", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest");

        expect(() =>
          manager.requirePermission(SESSION_KEY_HASH, "transfer"),
        ).toThrow("Missing permission");
      });

      it("should log failed permission check", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest");

        try {
          manager.requirePermission(SESSION_KEY_HASH, "transfer");
        } catch {
          // Expected
        }

        const log = manager.getAccessLog();
        const failedEntry = log.find((e) => !e.success);
        expect(failedEntry).toBeDefined();
      });
    });

    describe("addPermissionToRole()", () => {
      it("should add permission to role", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest");

        expect(manager.hasPermission(SESSION_KEY_HASH, "challenge")).toBe(
          false,
        );

        manager.addPermissionToRole("guest", "challenge");

        expect(manager.hasPermission(SESSION_KEY_HASH, "challenge")).toBe(true);
      });

      it("should not duplicate permissions", () => {
        const manager = createManagerWithIdentity();

        manager.addPermissionToRole("owner", "transfer");
        manager.addPermissionToRole("owner", "transfer");

        // Should still work without error
        expect(manager.hasPermission(IDENTITY_KEY_HASH, "transfer")).toBe(true);
      });
    });

    describe("removePermissionFromRole()", () => {
      it("should remove permission from role", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest");

        expect(manager.hasPermission(SESSION_KEY_HASH, "verify")).toBe(true);

        manager.removePermissionFromRole("guest", "verify");

        expect(manager.hasPermission(SESSION_KEY_HASH, "verify")).toBe(false);
      });
    });
  });

  // ============ Rate Limiting Tests ============

  describe("Rate Limiting", () => {
    describe("setCurrentBlock()", () => {
      it("should reset per-block counters on new block", () => {
        const manager = createManagerWithIdentity();
        manager.setCurrentBlock(100n);

        // Consume some rate limit
        for (let i = 0; i < 5; i++) {
          manager.checkRateLimit(IDENTITY_KEY_HASH, "action");
        }

        // New block should reset
        manager.setCurrentBlock(101n);

        // Should be able to do more actions
        expect(() =>
          manager.checkRateLimit(IDENTITY_KEY_HASH, "action"),
        ).not.toThrow();
      });
    });

    describe("checkRateLimit()", () => {
      it("should allow actions within limit", () => {
        const manager = createManagerWithIdentity();
        manager.setCurrentBlock(100n);

        for (let i = 0; i < 10; i++) {
          expect(() =>
            manager.checkRateLimit(IDENTITY_KEY_HASH, "action"),
          ).not.toThrow();
        }
      });

      it("should throw when exceeding action limit", () => {
        const manager = createManagerWithIdentity();
        manager.setCurrentBlock(100n);

        // Exhaust limit (default 10 per block)
        for (let i = 0; i < 10; i++) {
          manager.checkRateLimit(IDENTITY_KEY_HASH, "action");
        }

        expect(() =>
          manager.checkRateLimit(IDENTITY_KEY_HASH, "action"),
        ).toThrow("Rate limit exceeded");
      });

      it("should track transfers per day", () => {
        const manager = createManagerWithIdentity();

        // Do some transfers (default 50 per day)
        for (let i = 0; i < 50; i++) {
          manager.checkRateLimit(IDENTITY_KEY_HASH, "transfer");
        }

        expect(() =>
          manager.checkRateLimit(IDENTITY_KEY_HASH, "transfer"),
        ).toThrow("Rate limit exceeded");
      });

      it("should track stake changes per day", () => {
        const manager = createManagerWithIdentity();

        // Default 5 stake changes per day
        for (let i = 0; i < 5; i++) {
          manager.checkRateLimit(IDENTITY_KEY_HASH, "stake_change");
        }

        expect(() =>
          manager.checkRateLimit(IDENTITY_KEY_HASH, "stake_change"),
        ).toThrow("Rate limit exceeded");
      });

      it("should track limits per key", () => {
        const manager = createManager();
        manager.registerKey(IDENTITY_KEY_HASH, "identity", "owner");
        manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
        manager.setCurrentBlock(100n);

        // Exhaust identity key limit
        for (let i = 0; i < 10; i++) {
          manager.checkRateLimit(IDENTITY_KEY_HASH, "action");
        }

        // Operational key should still have its own limit
        expect(() =>
          manager.checkRateLimit(OPERATIONAL_KEY_HASH, "action"),
        ).not.toThrow();
      });
    });

    describe("requiresMultiSig()", () => {
      it("should return true for amount above threshold", () => {
        const manager = createManager();

        // Default threshold is 1000 LGM (10^9 * 10^18)
        expect(manager.requiresMultiSig(1000000000n * ONE_LGM)).toBe(true);
      });

      it("should return false for amount below threshold", () => {
        const manager = createManager();

        // Default threshold is 1000000000n (10^9 base units)
        // Use an amount below that threshold
        expect(manager.requiresMultiSig(500000000n)).toBe(false); // 0.5 * 10^9 < 10^9
      });
    });
  });

  // ============ Security Settings Tests ============

  describe("Security Settings", () => {
    describe("updateSecuritySettings()", () => {
      it("should update multi-sig threshold", () => {
        const manager = createManager();

        manager.updateSecuritySettings(
          {
            multiSigThreshold: 500n * ONE_LGM,
          },
          "test-updater",
        );

        expect(manager.requiresMultiSig(600n * ONE_LGM)).toBe(true);
        expect(manager.requiresMultiSig(400n * ONE_LGM)).toBe(false);
      });

      it("should update max session duration", () => {
        const manager = createManager();

        manager.updateSecuritySettings(
          {
            maxSessionDuration: 1 * 60 * 60 * 1000, // 1 hour
          },
          "test",
        );

        expect(manager.getSecuritySettings().maxSessionDuration).toBe(3600000);
      });

      it("should update rate limits", () => {
        const manager = createManager();

        manager.updateSecuritySettings(
          {
            rateLimits: {
              maxActionsPerBlock: 5,
              maxTransfersPerDay: 10,
              maxStakeChangesPerDay: 2,
            },
          },
          "test",
        );

        const settings = manager.getSecuritySettings();
        expect(settings.rateLimits.maxActionsPerBlock).toBe(5);
        expect(settings.rateLimits.maxTransfersPerDay).toBe(10);
      });

      it("should throw for negative multi-sig threshold", () => {
        const manager = createManager();

        expect(() =>
          manager.updateSecuritySettings(
            {
              multiSigThreshold: -100n,
            },
            "test",
          ),
        ).toThrow("cannot be negative");
      });

      it("should throw for too short session duration", () => {
        const manager = createManager();

        expect(() =>
          manager.updateSecuritySettings(
            {
              maxSessionDuration: 1000, // 1 second
            },
            "test",
          ),
        ).toThrow("at least 1 minute");
      });

      it("should log security updates", () => {
        const manager = createManager();
        manager.updateSecuritySettings(
          {
            multiSigThreshold: 500n * ONE_LGM,
          },
          "test",
        );

        const log = manager.getAccessLog();
        const updateEntry = log.find((e) => e.action === "update_security");
        expect(updateEntry).toBeDefined();
      });
    });

    describe("getSecuritySettings()", () => {
      it("should return copy of settings", () => {
        const manager = createManager();
        const settings = manager.getSecuritySettings();

        settings.maxSessionDuration = 1;

        expect(manager.getSecuritySettings().maxSessionDuration).not.toBe(1);
      });
    });
  });

  // ============ Access Logging Tests ============

  describe("Access Logging", () => {
    describe("getAccessLog()", () => {
      it("should return recent log entries", () => {
        const manager = createManagerWithIdentity();
        manager.startSession(IDENTITY_KEY_HASH);

        const log = manager.getAccessLog();
        expect(log.length).toBeGreaterThan(0);
      });

      it("should respect limit parameter", () => {
        const manager = createManager();

        // Generate many log entries
        for (let i = 0; i < 20; i++) {
          manager.registerKey(`key_${i}`, "session", "guest");
        }

        const log = manager.getAccessLog(5);
        expect(log.length).toBeLessThanOrEqual(5);
      });

      it("should keep log bounded to 1000 entries", () => {
        const manager = createManager();

        // Generate many entries
        for (let i = 0; i < 1100; i++) {
          try {
            manager.registerKey(`key_${i}`, "session", "guest");
          } catch {
            // Some will fail due to duplicates, that's fine
          }
        }

        const log = manager.getAccessLog(2000);
        expect(log.length).toBeLessThanOrEqual(1000);
      });
    });

    describe("getFailedAttempts()", () => {
      it("should return failed attempts", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest");

        // Generate failed attempts
        try {
          manager.requirePermission(SESSION_KEY_HASH, "transfer");
        } catch {
          /* expected */
        }

        const failed = manager.getFailedAttempts();
        expect(failed.length).toBeGreaterThan(0);
        expect(failed.every((e) => !e.success)).toBe(true);
      });

      it("should respect time cutoff", () => {
        const manager = createManager();
        manager.registerKey(SESSION_KEY_HASH, "session", "guest");

        try {
          manager.requirePermission(SESSION_KEY_HASH, "transfer");
        } catch {
          /* expected */
        }

        // Get failures since far future (should be empty)
        const failed = manager.getFailedAttempts(BASE_TIME + 100000);
        expect(failed).toHaveLength(0);
      });
    });
  });

  // ============ Guardian Management Tests ============

  describe("Guardian Management", () => {
    it("should get configured guardians", () => {
      const manager = createManager();
      manager.registerKey(GUARDIAN_KEY_HASH, "guardian", "guardian");
      manager.registerKey("guardian_2", "guardian", "guardian");

      const guardians = manager.getGuardians();
      expect(guardians).toHaveLength(2);
    });

    it("should check if key is guardian", () => {
      const manager = createManager();
      manager.registerKey(GUARDIAN_KEY_HASH, "guardian", "guardian");
      manager.registerKey(SESSION_KEY_HASH, "session", "guest");

      expect(manager.isGuardian(GUARDIAN_KEY_HASH)).toBe(true);
      expect(manager.isGuardian(SESSION_KEY_HASH)).toBe(false);
    });

    it("should not include revoked guardians", () => {
      const manager = createManager();
      manager.registerKey(GUARDIAN_KEY_HASH, "guardian", "guardian");
      manager.revokeKey(GUARDIAN_KEY_HASH, "test");

      expect(manager.getGuardians()).toHaveLength(0);
    });
  });

  // ============ State Export Tests ============

  describe("State Export", () => {
    it("should export full state", () => {
      const manager = createManagerWithIdentity();
      manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
      manager.startSession(IDENTITY_KEY_HASH);

      const state = manager.exportState();

      expect(state.keys.length).toBe(2);
      expect(state.activeSessions.length).toBe(1);
      expect(state.permissions).toBeInstanceOf(Map);
    });

    it("should return copies not references", () => {
      const manager = createManagerWithIdentity();
      const state = manager.exportState();

      state.keys.push({
        keyHash: "injected",
        keyType: "session",
        role: "guest",
        registeredAt: BASE_TIME,
      });

      // Original should be unchanged
      expect(manager.exportState().keys.length).toBe(1);
    });
  });

  // ============ Summary Tests ============

  describe("getSummary()", () => {
    it("should provide comprehensive summary", () => {
      const manager = createManager();
      manager.registerKey(IDENTITY_KEY_HASH, "identity", "owner");
      manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");
      manager.registerKey(SESSION_KEY_HASH, "session", "guest", {
        expiresAt: BASE_TIME - 1000, // Expired relative to BASE_TIME
      });
      manager.registerKey(GUARDIAN_KEY_HASH, "guardian", "guardian");
      manager.revokeKey(OPERATIONAL_KEY_HASH, "test");
      manager.startSession(IDENTITY_KEY_HASH);

      const summary = manager.getSummary();

      expect(summary.registeredKeys).toBe(4);
      expect(summary.activeKeys).toBe(2); // identity + guardian
      expect(summary.revokedKeys).toBe(1);
      expect(summary.expiredKeys).toBe(1);
      expect(summary.activeSessions).toBe(1);
      expect(summary.guardiansConfigured).toBe(1);
    });

    it("should detect elevated security status", () => {
      const manager = createManager();
      manager.registerKey(SESSION_KEY_HASH, "session", "guest");

      // Generate failed attempts
      for (let i = 0; i < 5; i++) {
        try {
          manager.requirePermission(SESSION_KEY_HASH, "transfer");
        } catch {
          /* expected */
        }
      }

      const summary = manager.getSummary();
      expect(summary.securityStatus).toBe("elevated");
    });

    it("should detect lockdown security status", () => {
      const manager = createManager();
      manager.registerKey(SESSION_KEY_HASH, "session", "guest");

      // Generate many failed attempts
      for (let i = 0; i < 10; i++) {
        try {
          manager.requirePermission(SESSION_KEY_HASH, "transfer");
        } catch {
          /* expected */
        }
      }

      const summary = manager.getSummary();
      expect(summary.securityStatus).toBe("lockdown");
    });
  });

  // ============ Edge Cases ============

  describe("Edge Cases", () => {
    it("should handle concurrent sessions from same key", () => {
      const manager = createManagerWithIdentity();

      const session1 = manager.startSession(IDENTITY_KEY_HASH);
      const session2 = manager.startSession(IDENTITY_KEY_HASH);

      expect(session1.sessionId).not.toBe(session2.sessionId);
      expect(manager.getSummary().activeSessions).toBe(2);
    });

    it("should mask key hashes in logs", () => {
      const manager = createManager();
      // Register with a specific registeredBy key hash to test masking
      const registeredByHash = "registeredby_key_hash_longerthan12chars";
      manager.registerKey(IDENTITY_KEY_HASH, "identity", "owner", {
        registeredBy: registeredByHash,
      });

      const log = manager.getAccessLog();
      const entry = log.find((e) => e.action === "register_key");

      // The keyHash field is the actor (registeredBy), which should be masked
      expect(entry?.keyHash).not.toBe(registeredByHash);
      expect(entry?.keyHash).toContain("...");

      // The details also contain the masked key hash being registered
      expect(entry?.details).toBeDefined();
      const details = JSON.parse(entry!.details!);
      expect(details.keyHash).toContain("...");
      expect(details.keyHash).not.toBe(IDENTITY_KEY_HASH);
    });

    it("should generate unique session IDs", () => {
      const manager = createManagerWithIdentity();

      const sessionIds = new Set<string>();
      for (let i = 0; i < 100; i++) {
        // Advance block each iteration so deterministic ID inputs differ
        manager.setCurrentBlock(BigInt(i + 10), BASE_TIME + i * 2521);
        const session = manager.startSession(IDENTITY_KEY_HASH);
        sessionIds.add(session.sessionId);
        manager.endSession(session.sessionId);
      }

      expect(sessionIds.size).toBe(100);
    });

    it("should handle key rotation cooldown edge cases", () => {
      const manager = createManager();
      manager.registerKey(OPERATIONAL_KEY_HASH, "operational", "operator");

      // Set block to exactly at cooldown boundary
      manager.setCurrentBlock(1000n);

      // First rotation
      const newKey1 = manager.rotateKey(OPERATIONAL_KEY_HASH, "new_1", "test");

      // Set block to exactly at cooldown (1000 blocks later)
      manager.setCurrentBlock(2000n);

      // Should work at exactly cooldown
      expect(() => manager.rotateKey("new_1", "new_2", "test")).not.toThrow();
    });
  });
});
