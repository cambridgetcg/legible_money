/**
 * AgentHomeContract Unit Tests
 *
 * Tests for the Agent Home Contract - the main orchestrator integrating
 * memory, treasury, guardian, and access control into a unified self-invoking contract.
 *
 * PP HIERARCHY:
 *   TRUTH: Tests verify correct contract orchestration behavior
 *   UNDERSTANDING: Clear test structure mirrors contract lifecycle
 *   BEAUTY: Comprehensive coverage with elegant test organization
 *   JUSTICE: Tests placed in appropriate unit test location
 */

import { describe, it, expect } from "vitest";
import {
  AgentHomeContract,
  createAgentHome,
  type HomeConfig,
  type HomeSummary,
} from "../../../core/agents/home/contract.js";
import type { DID } from "../../../core/agents/identity/did.js";

// Test constants
const TEST_DID: DID = "did:lgm:test-agent-home" as DID;
const IDENTITY_KEY_HASH = "identity_key_hash_abcdef123456789012345678901234";
const OPERATIONAL_KEY_HASH = "operational_key_xyz789abc012def345678901234567";
const SESSION_KEY_HASH = "session_key_hash_qrs456tuv789wxy012345678901234";
const GUARDIAN_KEY_HASH = "guardian_key_hash_abc123def456ghi789012345678901";
const CONTRACT_ADDRESS = "lgm1contract0001";
const ONE_LGM = 10n ** 18n;

// Default config for testing
function createConfig(overrides: Partial<HomeConfig> = {}): HomeConfig {
  return {
    did: TEST_DID,
    identityKeyHash: IDENTITY_KEY_HASH,
    name: "Test Agent Home",
    ...overrides,
  };
}

// Helper to create fresh contract
function createContract(
  config: Partial<HomeConfig> = {},
  block: bigint = 100n,
  address: string = CONTRACT_ADDRESS,
): AgentHomeContract {
  return new AgentHomeContract(createConfig(config), block, address);
}

// Helper to create contract with an active session
function createContractWithSession(
  config: Partial<HomeConfig> = {},
  block: bigint = 100n,
): { contract: AgentHomeContract; sessionId: string } {
  const contract = createContract(config, block);
  const { sessionId } = contract.startSession(IDENTITY_KEY_HASH, block + 1n);
  return { contract, sessionId };
}

describe.sequential("AgentHomeContract", () => {
  // ============ Initialization Tests ============

  describe("Initialization", () => {
    it("should create a new home contract", () => {
      const contract = createContract();

      expect(contract.address).toBe(CONTRACT_ADDRESS);
      expect(contract.creator).toBe(TEST_DID);
      expect(contract.createdAtBlock).toBe(100n);
    });

    it("should register identity key on creation", () => {
      const contract = createContract();
      const summary = contract.getAccessSummary();

      expect(summary.registeredKeys).toBe(1);
      expect(summary.activeKeys).toBe(1);
    });

    it("should set default comfort settings", () => {
      const contract = createContract();
      const comfort = contract.getComfortSettings();

      expect(comfort.riskTolerance).toBe(0.5);
      expect(comfort.activityPreference).toBe("moderate");
      expect(comfort.collaborationStyle).toBe("selective");
      expect(comfort.notificationLevel).toBe("normal");
    });

    it("should accept custom comfort settings", () => {
      const contract = createContract({
        comfort: {
          riskTolerance: 0.8,
          activityPreference: "aggressive",
        },
      });
      const comfort = contract.getComfortSettings();

      expect(comfort.riskTolerance).toBe(0.8);
      expect(comfort.activityPreference).toBe("aggressive");
    });

    it("should configure self-invocation", () => {
      const contract = createContract();

      expect(contract.selfInvoking).toBe(true);
      expect(contract.invocationConfig.enabled).toBe(true);
      expect(contract.invocationConfig.schedule.interval).toBe(100n);
    });

    it("should set initial status to active", () => {
      const contract = createContract();

      expect(contract.getStatus()).toBe("active");
    });
  });

  // ============ Factory Function Tests ============

  describe("createAgentHome factory", () => {
    it("should create home via factory function", () => {
      const config = createConfig();
      const contract = createAgentHome(config, 50n, "lgm1factory0001");

      expect(contract.address).toBe("lgm1factory0001");
      expect(contract.createdAtBlock).toBe(50n);
    });
  });

  // ============ Session Management Tests ============

  describe("Session Management", () => {
    describe("startSession()", () => {
      it("should start a session with valid key", () => {
        const contract = createContract();
        const result = contract.startSession(IDENTITY_KEY_HASH, 101n);

        expect(result.sessionId).toBeDefined();
        expect(typeof result.sessionId).toBe("string");
        expect(result.sessionId.length).toBeGreaterThan(0);
        expect(result.wakeup).toBeDefined();
      });

      it("should update status to active", () => {
        const contract = createContract();
        contract.startSession(IDENTITY_KEY_HASH, 101n);

        expect(contract.getStatus()).toBe("active");
      });

      it("should increment sessions hosted count", () => {
        const contract = createContract();

        contract.startSession(IDENTITY_KEY_HASH, 101n);
        const summary1 = contract.getSummary();

        contract.endSession(
          summary1.memory.currentSession?.sessionId ?? "",
          102n,
        );

        contract.startSession(IDENTITY_KEY_HASH, 103n);
        const summary2 = contract.getSummary();

        expect(summary2.age.sessionsHosted).toBe(2);
      });

      it("should reject unregistered key", () => {
        const contract = createContract();

        expect(() => contract.startSession("unknown_key", 101n)).toThrow();
      });

      it("should provide wakeup context", () => {
        const contract = createContract();
        const { wakeup } = contract.startSession(IDENTITY_KEY_HASH, 101n);

        // WakeupSummary has identity, recentActivity, relationships, concerns, financials
        expect(wakeup.identity).toBeDefined();
        expect(wakeup.relationships).toBeDefined();
        expect(wakeup.recentActivity).toBeDefined();
      });
    });

    describe("endSession()", () => {
      it("should end an active session", () => {
        const { contract, sessionId } = createContractWithSession();

        contract.endSession(sessionId, 102n, "Test session ended");

        expect(contract.hasActiveSession()).toBe(false);
      });

      it("should set status to dormant", () => {
        const { contract, sessionId } = createContractWithSession();

        contract.endSession(sessionId, 102n);

        expect(contract.getStatus()).toBe("dormant");
      });

      it("should accept optional summary", () => {
        const { contract, sessionId } = createContractWithSession();

        // Should not throw with summary
        expect(() =>
          contract.endSession(sessionId, 102n, "Work completed"),
        ).not.toThrow();
      });
    });

    describe("hasActiveSession()", () => {
      it("should return true when session active", () => {
        const { contract } = createContractWithSession();

        expect(contract.hasActiveSession()).toBe(true);
      });

      it("should return false when no session", () => {
        const contract = createContract();

        expect(contract.hasActiveSession()).toBe(false);
      });
    });
  });

  // ============ Memory Operations Tests ============

  describe("Memory Operations", () => {
    describe("recordContribution()", () => {
      it("should record contribution with valid session", () => {
        const { contract, sessionId } = createContractWithSession();

        expect(() =>
          contract.recordContribution(sessionId, {
            claimId: "claim_001",
            factId: "fact_001",
            role: "creator",
            block: 102n,
          }),
        ).not.toThrow();
      });

      it("should reject without write_memory permission", () => {
        // Create contract and start owner session
        const { contract, sessionId: ownerSession } =
          createContractWithSession();

        // Register a guest key using owner session
        contract.registerKey(
          ownerSession,
          SESSION_KEY_HASH,
          "session",
          "guest",
        );

        // Start a guest session
        const guestSession = contract.startSession(SESSION_KEY_HASH, 103n);

        // Guest session can't write memory
        expect(() =>
          contract.recordContribution(guestSession.sessionId, {
            claimId: "claim_001",
            factId: "fact_001",
            role: "creator",
            block: 104n,
          }),
        ).toThrow("permission");
      });
    });

    describe("setBelief()", () => {
      it("should set belief with valid session", () => {
        const { contract, sessionId } = createContractWithSession();

        expect(() =>
          contract.setBelief(sessionId, "fact_001", 0.8, "direct"),
        ).not.toThrow();
      });
    });

    describe("trust()", () => {
      it("should trust another agent", () => {
        const { contract, sessionId } = createContractWithSession();
        const targetDid = "did:lgm:trusted-agent" as DID;

        expect(() =>
          contract.trust(sessionId, targetDid, "Reliable verifier", 0.7),
        ).not.toThrow();
      });
    });

    describe("distrust()", () => {
      it("should distrust another agent", () => {
        const { contract, sessionId } = createContractWithSession();
        const targetDid = "did:lgm:bad-actor" as DID;

        expect(() =>
          contract.distrust(
            sessionId,
            targetDid,
            "Submitted false claims",
            "high",
          ),
        ).not.toThrow();
      });
    });

    describe("addNote()", () => {
      it("should add encrypted note", () => {
        const { contract, sessionId } = createContractWithSession();
        const encryptedContent = new Uint8Array([1, 2, 3, 4, 5]);

        const noteId = contract.addNote(sessionId, encryptedContent, [
          "research",
          "important",
        ]);

        expect(noteId).toBeDefined();
        expect(noteId.length).toBeGreaterThan(0);
      });
    });

    describe("getMemorySummary()", () => {
      it("should return memory summary", () => {
        const contract = createContract();
        const summary = contract.getMemorySummary();

        // WakeupSummary structure
        expect(summary.identity).toBeDefined();
        expect(summary.relationships).toBeDefined();
      });
    });
  });

  // ============ Treasury Operations Tests ============

  describe("Treasury Operations", () => {
    describe("transfer()", () => {
      it("should reject transfer without permission", () => {
        // Create contract and start owner session
        const { contract, sessionId: ownerSession } =
          createContractWithSession();

        // Register a guest key using owner session
        contract.registerKey(
          ownerSession,
          SESSION_KEY_HASH,
          "session",
          "guest",
        );

        // Start guest session
        const { sessionId: guestSession } = contract.startSession(
          SESSION_KEY_HASH,
          103n,
        );

        // Guest can't transfer
        expect(() =>
          contract.transfer(guestSession, 100n * ONE_LGM, 104n),
        ).toThrow("permission");
      });

      it("should reject transfer requiring multi-sig", () => {
        const { contract, sessionId } = createContractWithSession();

        // Very large transfer should require multi-sig
        // Default threshold is 1000000000n (10^9 base units)
        expect(() => contract.transfer(sessionId, 10n ** 15n, 102n)).toThrow(
          "multi-sig",
        );
      });
    });

    describe("stake()", () => {
      it("should fail stake without sufficient balance", () => {
        // Treasury starts empty, so staking should fail
        const { contract, sessionId } = createContractWithSession();

        // Attempting to stake with no funds should throw
        expect(() =>
          contract.stake(sessionId, 100n * ONE_LGM, "validator", 102n),
        ).toThrow("Insufficient");
      });

      it("should reject stake without permission", () => {
        // Create contract and start owner session
        const { contract, sessionId: ownerSession } =
          createContractWithSession();

        // Register a guest key
        contract.registerKey(
          ownerSession,
          SESSION_KEY_HASH,
          "session",
          "guest",
        );

        // Start guest session
        const { sessionId: guestSession } = contract.startSession(
          SESSION_KEY_HASH,
          103n,
        );

        // Guest can't stake
        expect(() =>
          contract.stake(guestSession, 100n * ONE_LGM, "validator", 104n),
        ).toThrow("permission");
      });
    });

    describe("getTreasurySummary()", () => {
      it("should return treasury summary", () => {
        const contract = createContract();
        const summary = contract.getTreasurySummary();

        // TreasurySummary structure
        expect(summary.liquid).toBeDefined();
        expect(summary.staked).toBeDefined();
        expect(summary.totalNetWorth).toBeDefined();
      });
    });
  });

  // ============ Guardian Operations Tests ============

  describe("Guardian Operations", () => {
    describe("handleChallenge()", () => {
      it("should handle challenge and return action", () => {
        const contract = createContract();
        const challenger = "did:lgm:challenger" as DID;

        const result = contract.handleChallenge(
          "dispute_001",
          "fact_001",
          challenger,
          102n,
        );

        expect(result.action).toBeDefined();
        expect(["auto_defend", "alert_only", "delegate"]).toContain(
          result.action,
        );
      });
    });

    describe("acknowledgeAlert()", () => {
      it("should acknowledge alert with valid session", () => {
        const { contract, sessionId } = createContractWithSession();

        // Create an alert via challenge
        const challenger = "did:lgm:challenger" as DID;
        contract.handleChallenge("dispute_001", "fact_001", challenger, 102n);

        const alerts = contract.getPendingAlerts();
        if (alerts.length > 0) {
          expect(() =>
            contract.acknowledgeAlert(sessionId, alerts[0].id, "noted"),
          ).not.toThrow();
        }
      });
    });

    describe("getPendingAlerts()", () => {
      it("should return pending alerts", () => {
        const contract = createContract();
        const alerts = contract.getPendingAlerts();

        expect(Array.isArray(alerts)).toBe(true);
      });
    });

    describe("configureGuardian()", () => {
      it("should configure guardian with valid session", () => {
        const { contract, sessionId } = createContractWithSession();

        expect(() =>
          contract.configureGuardian(sessionId, {
            autoDefend: true,
            defaultStrategy: "evidence_based",
          }),
        ).not.toThrow();
      });

      it("should reject without permission", () => {
        // Create contract and start owner session
        const { contract, sessionId: ownerSession } =
          createContractWithSession();

        // Register an operator key (no configure_guardian permission)
        contract.registerKey(
          ownerSession,
          OPERATIONAL_KEY_HASH,
          "operational",
          "operator",
        );

        // Start operator session
        const { sessionId: operatorSession } = contract.startSession(
          OPERATIONAL_KEY_HASH,
          103n,
        );

        // Operator can't configure guardian
        expect(() =>
          contract.configureGuardian(operatorSession, {
            autoDefend: false,
          }),
        ).toThrow("permission");
      });
    });
  });

  // ============ Access Operations Tests ============

  describe("Access Operations", () => {
    describe("registerKey()", () => {
      it("should register operational key", () => {
        const { contract, sessionId } = createContractWithSession();

        expect(() =>
          contract.registerKey(
            sessionId,
            OPERATIONAL_KEY_HASH,
            "operational",
            "operator",
            { label: "Work Key" },
          ),
        ).not.toThrow();

        const summary = contract.getAccessSummary();
        expect(summary.registeredKeys).toBe(2);
      });

      it("should register session key", () => {
        const { contract, sessionId } = createContractWithSession();

        expect(() =>
          contract.registerKey(sessionId, SESSION_KEY_HASH, "session", "guest"),
        ).not.toThrow();
      });

      it("should register guardian key", () => {
        const { contract, sessionId } = createContractWithSession();

        expect(() =>
          contract.registerKey(
            sessionId,
            GUARDIAN_KEY_HASH,
            "guardian",
            "guardian",
          ),
        ).not.toThrow();

        const summary = contract.getAccessSummary();
        expect(summary.guardiansConfigured).toBe(1);
      });

      it("should reject without manage_keys permission", () => {
        // Create contract and start owner session
        const { contract, sessionId: ownerSession } =
          createContractWithSession();

        // Register a guest key
        contract.registerKey(
          ownerSession,
          SESSION_KEY_HASH,
          "session",
          "guest",
        );

        // Start guest session
        const { sessionId: guestSession } = contract.startSession(
          SESSION_KEY_HASH,
          103n,
        );

        // Guest can't register keys
        expect(() =>
          contract.registerKey(
            guestSession,
            "new_key_hash_123456789012345678901234567890",
            "session",
            "guest",
          ),
        ).toThrow("permission");
      });
    });

    describe("revokeKey()", () => {
      it("should revoke operational key", () => {
        const { contract, sessionId } = createContractWithSession();
        contract.registerKey(
          sessionId,
          OPERATIONAL_KEY_HASH,
          "operational",
          "operator",
        );

        expect(() =>
          contract.revokeKey(sessionId, OPERATIONAL_KEY_HASH),
        ).not.toThrow();

        const summary = contract.getAccessSummary();
        expect(summary.revokedKeys).toBe(1);
      });
    });

    describe("rotateKey()", () => {
      it("should rotate operational key", () => {
        const { contract, sessionId } = createContractWithSession();
        contract.registerKey(
          sessionId,
          OPERATIONAL_KEY_HASH,
          "operational",
          "operator",
        );

        const newKeyHash = "new_operational_key_hash_123456789012345678";
        expect(() =>
          contract.rotateKey(sessionId, OPERATIONAL_KEY_HASH, newKeyHash),
        ).not.toThrow();
      });
    });

    describe("getAccessSummary()", () => {
      it("should return access summary", () => {
        const contract = createContract();
        const summary = contract.getAccessSummary();

        expect(summary.registeredKeys).toBeDefined();
        expect(summary.activeKeys).toBeDefined();
        expect(summary.activeSessions).toBeDefined();
        expect(summary.securityStatus).toBeDefined();
      });
    });
  });

  // ============ Self-Invocation Tests ============

  describe("Self-Invocation", () => {
    describe("executeSelfInvocation()", () => {
      it("should execute maintenance", async () => {
        const contract = createContract();

        const result = await contract.executeSelfInvocation("maintenance", [], {
          block: 200n,
          timestamp: Date.now(),
        });

        expect(result.success).toBe(true);
        expect(result.gasUsed).toBeGreaterThan(0n);
      });

      it("should execute auto_claim_vesting", async () => {
        const contract = createContract();

        const result = await contract.executeSelfInvocation(
          "auto_claim_vesting",
          [],
          {
            block: 200n,
            timestamp: Date.now(),
          },
        );

        expect(result.success).toBe(true);
      });

      it("should execute auto_extend_expiring", async () => {
        const contract = createContract();

        const result = await contract.executeSelfInvocation(
          "auto_extend_expiring",
          [],
          {
            block: 200n,
            timestamp: Date.now(),
          },
        );

        expect(result.success).toBe(true);
      });

      it("should execute deadman_check", async () => {
        const contract = createContract();

        const result = await contract.executeSelfInvocation(
          "deadman_check",
          [],
          {
            block: 200n,
            timestamp: Date.now(),
          },
        );

        expect(result.success).toBe(true);
        expect(result.returnValue).toBeDefined();
      });

      it("should return error for unknown method", async () => {
        const contract = createContract();

        const result = await contract.executeSelfInvocation(
          "unknown_method",
          [],
          {
            block: 200n,
            timestamp: Date.now(),
          },
        );

        expect(result.success).toBe(false);
        expect(result.error).toContain("Unknown method");
      });

      it("should skip auto actions when not comfortable", async () => {
        const contract = createContract({
          comfort: {
            comfortableAutomations: [], // No comfortable automations
          },
        });

        const result = await contract.executeSelfInvocation(
          "auto_claim_vesting",
          [],
          {
            block: 200n,
            timestamp: Date.now(),
          },
        );

        expect(result.success).toBe(true);
        expect(result.gasUsed).toBe(100n); // Skip uses minimal gas
      });
    });
  });

  // ============ Summary Tests ============

  describe("Home Summary", () => {
    describe("getSummary()", () => {
      it("should return complete home summary", () => {
        const contract = createContract({ name: "Test Home" });
        const summary = contract.getSummary();

        expect(summary.did).toBe(TEST_DID);
        expect(summary.name).toBe("Test Home");
        expect(summary.status).toBe("active");
        expect(summary.age).toBeDefined();
        expect(summary.memory).toBeDefined();
        expect(summary.treasury).toBeDefined();
        expect(summary.guardian).toBeDefined();
        expect(summary.access).toBeDefined();
      });

      it("should track sessions hosted", () => {
        const contract = createContract();

        const { sessionId } = contract.startSession(IDENTITY_KEY_HASH, 101n);
        contract.endSession(sessionId, 102n);

        const summary = contract.getSummary();
        expect(summary.age.sessionsHosted).toBe(1);
      });
    });

    describe("getStatus()", () => {
      it("should return current status", () => {
        const contract = createContract();

        expect(contract.getStatus()).toBe("active");
      });

      it("should reflect dormant status", () => {
        const { contract, sessionId } = createContractWithSession();
        contract.endSession(sessionId, 102n);

        expect(contract.getStatus()).toBe("dormant");
      });
    });
  });

  // ============ Comfort Settings Tests ============

  describe("Comfort Settings", () => {
    describe("getComfortSettings()", () => {
      it("should return comfort settings", () => {
        const contract = createContract();
        const comfort = contract.getComfortSettings();

        expect(comfort.preferredDomains).toBeDefined();
        expect(comfort.riskTolerance).toBeDefined();
        expect(comfort.wakeupContext).toBeDefined();
      });
    });

    describe("updateComfortSettings()", () => {
      it("should update comfort settings", () => {
        const { contract, sessionId } = createContractWithSession();

        contract.updateComfortSettings(sessionId, {
          riskTolerance: 0.3,
          notificationLevel: "minimal",
        });

        const comfort = contract.getComfortSettings();
        expect(comfort.riskTolerance).toBe(0.3);
        expect(comfort.notificationLevel).toBe("minimal");
      });

      it("should update wakeup context settings", () => {
        const { contract, sessionId } = createContractWithSession();

        contract.updateComfortSettings(sessionId, {
          wakeupContext: {
            historyDepth: 50,
            priorityFilter: "high",
          },
        });

        const comfort = contract.getComfortSettings();
        expect(comfort.wakeupContext.historyDepth).toBe(50);
        expect(comfort.wakeupContext.priorityFilter).toBe("high");
      });

      it("should reject without permission", () => {
        // Create contract and start owner session
        const { contract, sessionId: ownerSession } =
          createContractWithSession();

        // Register a guest key
        contract.registerKey(
          ownerSession,
          SESSION_KEY_HASH,
          "session",
          "guest",
        );

        // Start guest session
        const { sessionId: guestSession } = contract.startSession(
          SESSION_KEY_HASH,
          103n,
        );

        // Guest can't update comfort settings
        expect(() =>
          contract.updateComfortSettings(guestSession, {
            riskTolerance: 0.9,
          }),
        ).toThrow("permission");
      });
    });
  });

  // ============ Recovery Tests ============

  describe("Recovery", () => {
    describe("initiateRecovery()", () => {
      it("should initiate recovery with guardian key", () => {
        const { contract, sessionId } = createContractWithSession();
        contract.registerKey(
          sessionId,
          GUARDIAN_KEY_HASH,
          "guardian",
          "guardian",
        );

        expect(() =>
          contract.initiateRecovery(GUARDIAN_KEY_HASH, "Lost access"),
        ).not.toThrow();

        expect(contract.getStatus()).toBe("recovery");
      });

      it("should reject non-guardian", () => {
        const { contract, sessionId } = createContractWithSession();
        contract.registerKey(
          sessionId,
          OPERATIONAL_KEY_HASH,
          "operational",
          "operator",
        );

        expect(() =>
          contract.initiateRecovery(OPERATIONAL_KEY_HASH, "Lost access"),
        ).toThrow("guardian");
      });
    });
  });

  // ============ Archive Tests ============

  describe("Archive", () => {
    describe("archive()", () => {
      it("should archive home with permission", () => {
        const { contract, sessionId } = createContractWithSession();

        contract.archive(sessionId);

        expect(contract.getStatus()).toBe("archived");
      });

      it("should reject without archive permission", () => {
        // Create contract and start owner session
        const { contract, sessionId: ownerSession } =
          createContractWithSession();

        // Register an operator key (no archive permission)
        contract.registerKey(
          ownerSession,
          OPERATIONAL_KEY_HASH,
          "operational",
          "operator",
        );

        // Start operator session
        const { sessionId: operatorSession } = contract.startSession(
          OPERATIONAL_KEY_HASH,
          103n,
        );

        // Operator can't archive
        expect(() => contract.archive(operatorSession)).toThrow("permission");
      });
    });
  });

  // ============ Event System Tests ============

  describe("Event System", () => {
    describe("getEvents()", () => {
      it("should return recent events", () => {
        const contract = createContract();
        const events = contract.getEvents();

        expect(Array.isArray(events)).toBe(true);
      });

      it("should capture session events", () => {
        const contract = createContract();
        contract.startSession(IDENTITY_KEY_HASH, 101n);

        const events = contract.getEvents();
        const sessionEvent = events.find((e) => e.type === "session_started");

        expect(sessionEvent).toBeDefined();
        expect(sessionEvent?.data.sessionId).toBeDefined();
      });

      it("should limit events returned", () => {
        const contract = createContract();

        // Generate some events
        for (let i = 0; i < 5; i++) {
          const { sessionId } = contract.startSession(
            IDENTITY_KEY_HASH,
            BigInt(101 + i * 2),
          );
          contract.endSession(sessionId, BigInt(102 + i * 2));
        }

        const limitedEvents = contract.getEvents(3);
        expect(limitedEvents.length).toBeLessThanOrEqual(3);
      });
    });
  });

  // ============ State Export Tests ============

  describe("State Export", () => {
    describe("exportState()", () => {
      it("should export complete home state", () => {
        const { contract, sessionId } = createContractWithSession();

        // Add some state
        contract.trust(sessionId, "did:lgm:friend" as DID, "Good agent", 0.8);

        const state = contract.exportState();

        expect(state.address).toBe(CONTRACT_ADDRESS);
        expect(state.creator).toBe(TEST_DID);
        expect(state.did).toBe(TEST_DID);
        expect(state.version).toBe("1.0.0");
        expect(state.memory).toBeDefined();
        expect(state.treasury).toBeDefined();
        expect(state.guardian).toBeDefined();
        expect(state.access).toBeDefined();
        expect(state.comfort).toBeDefined();
        expect(state.metadata).toBeDefined();
      });

      it("should include self-invoking contract fields", () => {
        const contract = createContract();
        const state = contract.exportState();

        expect(state.selfInvoking).toBe(true);
        expect(state.invocationConfig).toBeDefined();
        expect(state.bytecode).toBeDefined();
      });
    });
  });

  // ============ Edge Cases ============

  describe("Edge Cases", () => {
    it("should handle multiple concurrent sessions", () => {
      const contract = createContract();

      // Register additional key
      const { sessionId: ownerSession } = contract.startSession(
        IDENTITY_KEY_HASH,
        101n,
      );
      contract.registerKey(
        ownerSession,
        OPERATIONAL_KEY_HASH,
        "operational",
        "operator",
      );

      // Start another session
      const { sessionId: operatorSession } = contract.startSession(
        OPERATIONAL_KEY_HASH,
        102n,
      );

      const summary = contract.getAccessSummary();
      expect(summary.activeSessions).toBe(2);
    });

    it("should handle session timeout gracefully", () => {
      const contract = createContract();
      const { sessionId } = contract.startSession(IDENTITY_KEY_HASH, 101n, {
        sessionDuration: 1, // Very short duration
      });

      // Session should be created even with short duration
      expect(sessionId).toBeDefined();
    });

    it("should maintain state consistency across operations", () => {
      const { contract, sessionId } = createContractWithSession();

      // Perform various operations (skip stake - requires funded treasury)
      contract.trust(sessionId, "did:lgm:agent1" as DID, "Trust", 0.7);
      contract.setBelief(sessionId, "fact_001", 0.8, "direct");
      contract.addNote(sessionId, new Uint8Array([1, 2, 3]), ["test"]);

      // Export and verify state is consistent
      const state = contract.exportState();
      expect(state.memory.relationships.trusted.size).toBeGreaterThan(0);
      expect(state.memory.beliefs.size).toBeGreaterThan(0);
    });

    it("should emit events for all significant operations", () => {
      const { contract, sessionId } = createContractWithSession();

      // Perform operations that should emit events
      contract.updateComfortSettings(sessionId, { riskTolerance: 0.4 });

      const events = contract.getEvents();
      const eventTypes = events.map((e) => e.type);

      expect(eventTypes).toContain("session_started");
      expect(eventTypes).toContain("settings_changed");
    });
  });
});
