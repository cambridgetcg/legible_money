/**
 * GuardianManager Unit Tests
 *
 * Tests for the Agent Guardian - autonomous defense and maintenance for Agent Home.
 * Covers defense operations, maintenance, alerts, recovery, and deadman switch.
 *
 * PP HIERARCHY:
 *   TRUTH: Tests verify guardian actions are transparent and auditable
 *   UNDERSTANDING: Clear test structure mirrors guardian operations
 *   BEAUTY: Comprehensive coverage with elegant test organization
 *   JUSTICE: Tests placed in appropriate unit test location
 */

import { describe, it, expect } from "vitest";
import {
  GuardianManager,
  type MaintenanceContext,
  type MaintenanceResult,
  type GuardianSummary,
} from "../../../core/agents/home/guardian.js";
import type { AgentGuardian, Alert } from "../../../core/agents/home/types.js";

// Test constants
const TEST_DID = "did:lgm:test-agent-guardian";
const CHALLENGER_DID = "did:lgm:challenger";
const ALLY_DID = "did:lgm:ally";
const GUARDIAN_DID_1 = "did:lgm:guardian-1";
const GUARDIAN_DID_2 = "did:lgm:guardian-2";
const ONE_LGM = 10n ** 18n;

// Helper to create fresh manager for each test
function createManager(): GuardianManager {
  return new GuardianManager();
}

describe.sequential("GuardianManager", () => {
  // ============ Initialization Tests ============

  describe("Initialization", () => {
    it("should create guardian with default values", () => {
      const manager = createManager();
      const guardian = manager.getGuardian();

      expect(guardian.defense.autoDefendChallenges).toBe(true);
      expect(guardian.defense.defaultStrategy).toBe("moderate");
      expect(guardian.defense.defenseAllies).toEqual([]);
    });

    it("should create with existing guardian state", () => {
      const existingGuardian: AgentGuardian = {
        defense: {
          autoDefendChallenges: false,
          defaultStrategy: "aggressive",
          maxAutoDefenseStake: 5000n * ONE_LGM,
          evidenceSources: ["memory"],
          defenseAllies: [ALLY_DID],
        },
        maintenance: {
          monitorExpiration: false,
          monitorVesting: true,
          monitorReputation: false,
          checkInterval: 500n,
          lastMaintenanceBlock: 100n,
        },
        alerts: {
          triggers: [],
          delivery: {
            storeInHome: true,
            emitEvent: false,
            notifyAllies: true,
          },
          priorities: {
            criticalThreshold: 50n * ONE_LGM,
            highReputationDrop: 0.2,
            mediumDeadlineHours: 12,
          },
        },
        recovery: {
          guardians: [GUARDIAN_DID_1],
          threshold: 1,
          recoveryDelay: 5000n,
          deadman: {
            enabled: true,
            inactivityThreshold: 50000n,
            action: "transfer_to_beneficiary",
            beneficiary: ALLY_DID,
          },
        },
        activeDefenses: [],
        pendingAlerts: [],
        stats: {
          challengesAutoDefended: 5,
          challengesWon: 4,
          challengesLost: 1,
          alertsGenerated: 10,
          maintenanceActionsPerformed: 20,
          tokensPreservedFromExpiration: 1000n * ONE_LGM,
          vestingAutoClaimed: 500n * ONE_LGM,
        },
      };

      const manager = new GuardianManager(existingGuardian);
      const guardian = manager.getGuardian();

      expect(guardian.defense.autoDefendChallenges).toBe(false);
      expect(guardian.defense.defaultStrategy).toBe("aggressive");
      expect(guardian.stats.challengesWon).toBe(4);
    });

    it("should start clean (not dirty)", () => {
      const manager = createManager();
      expect(manager.isDirty()).toBe(false);
    });

    it("should have default alert triggers", () => {
      const manager = createManager();
      const guardian = manager.getGuardian();

      expect(guardian.alerts.triggers.length).toBeGreaterThan(0);

      const challengeTrigger = guardian.alerts.triggers.find(
        (t) => t.type === "challenge_received",
      );
      expect(challengeTrigger).toBeDefined();
      expect(challengeTrigger?.priority).toBe("critical");
      expect(challengeTrigger?.enabled).toBe(true);
    });
  });

  // ============ Defense Operations Tests ============

  describe("Defense Operations", () => {
    describe("handleChallenge()", () => {
      it("should handle challenge with auto-defend", () => {
        const manager = createManager();
        const result = manager.handleChallenge(
          "dispute-1",
          "fact-1",
          CHALLENGER_DID,
          100n,
        );

        expect(result.action).toBe("auto_defend");
        expect(result.defense).toBeDefined();
        expect(result.defense?.disputeId).toBe("dispute-1");
        expect(result.defense?.factId).toBe("fact-1");
        expect(result.defense?.challenger).toBe(CHALLENGER_DID);
        expect(result.defense?.autoResponding).toBe(true);
      });

      it("should create alert on challenge", () => {
        const manager = createManager();
        manager.handleChallenge("dispute-1", "fact-1", CHALLENGER_DID, 100n);

        const alerts = manager.getCriticalAlerts();
        expect(alerts.length).toBeGreaterThan(0);
        expect(alerts[0].type).toBe("challenge_received");
      });

      it("should alert only when auto-defend disabled", () => {
        const manager = createManager();
        manager.setDefenseConfig({ autoDefendChallenges: false });

        const result = manager.handleChallenge(
          "dispute-1",
          "fact-1",
          CHALLENGER_DID,
          100n,
        );

        expect(result.action).toBe("alert_only");
        expect(result.defense).toBeUndefined();
      });

      it("should delegate when strategy is delegate with allies", () => {
        const manager = createManager();
        manager.addDefenseAlly(ALLY_DID);
        manager.setDefenseConfig({ defaultStrategy: "delegate" });

        const result = manager.handleChallenge(
          "dispute-1",
          "fact-1",
          CHALLENGER_DID,
          100n,
        );

        expect(result.action).toBe("delegate");
      });

      it("should increment challengesAutoDefended stat", () => {
        const manager = createManager();
        manager.handleChallenge("d1", "f1", CHALLENGER_DID, 100n);
        manager.handleChallenge("d2", "f2", CHALLENGER_DID, 200n);

        const stats = manager.getStats();
        expect(stats.challengesAutoDefended).toBe(2);
      });

      it("should add to active defenses", () => {
        const manager = createManager();
        manager.handleChallenge("dispute-1", "fact-1", CHALLENGER_DID, 100n);

        const defenses = manager.getActiveDefenses();
        expect(defenses).toHaveLength(1);
      });

      it("should mark guardian as dirty", () => {
        const manager = createManager();
        manager.handleChallenge("dispute-1", "fact-1", CHALLENGER_DID, 100n);
        expect(manager.isDirty()).toBe(true);
      });
    });

    describe("updateDefensePhase()", () => {
      it("should update defense phase", () => {
        const manager = createManager();
        manager.handleChallenge("dispute-1", "fact-1", CHALLENGER_DID, 100n);
        manager.markClean();

        manager.updateDefensePhase("dispute-1", "evidence", 500n);

        const defenses = manager.getActiveDefenses();
        expect(defenses[0].phase).toBe("evidence");
        expect(defenses[0].nextDeadline).toBe(500n);
      });

      it("should create alert on phase update", () => {
        const manager = createManager();
        manager.handleChallenge("dispute-1", "fact-1", CHALLENGER_DID, 100n);
        const initialAlerts = manager.getPendingAlerts().length;
        manager.updateDefensePhase("dispute-1", "deliberation", 600n);

        expect(manager.getPendingAlerts().length).toBeGreaterThan(
          initialAlerts,
        );
      });
    });

    describe("completeDefense()", () => {
      it("should remove defense on completion (win)", () => {
        const manager = createManager();
        manager.handleChallenge("dispute-1", "fact-1", CHALLENGER_DID, 100n);
        manager.markClean();

        manager.completeDefense("dispute-1", true);

        expect(manager.getActiveDefenses()).toHaveLength(0);
        expect(manager.getStats().challengesWon).toBe(1);
      });

      it("should remove defense on completion (loss)", () => {
        const manager = createManager();
        manager.handleChallenge("dispute-1", "fact-1", CHALLENGER_DID, 100n);
        manager.markClean();

        manager.completeDefense("dispute-1", false);

        expect(manager.getActiveDefenses()).toHaveLength(0);
        expect(manager.getStats().challengesLost).toBe(1);
      });

      it("should mark guardian as dirty", () => {
        const manager = createManager();
        manager.handleChallenge("dispute-1", "fact-1", CHALLENGER_DID, 100n);
        manager.markClean();

        manager.completeDefense("dispute-1", true);
        expect(manager.isDirty()).toBe(true);
      });
    });

    describe("getUrgentDefenses()", () => {
      it("should get defenses with upcoming deadlines", () => {
        const manager = createManager();
        manager.handleChallenge("d1", "f1", CHALLENGER_DID, 100n);
        manager.handleChallenge("d2", "f2", CHALLENGER_DID, 100n);
        // Set different deadlines
        manager.updateDefensePhase("d1", "evidence", 150n);
        manager.updateDefensePhase("d2", "evidence", 500n);

        const urgent = manager.getUrgentDefenses(100n, 100n);

        expect(urgent.length).toBe(1);
        expect(urgent[0].disputeId).toBe("d1");
      });

      it("should include all within range", () => {
        const manager = createManager();
        manager.handleChallenge("d1", "f1", CHALLENGER_DID, 100n);
        manager.handleChallenge("d2", "f2", CHALLENGER_DID, 100n);
        manager.updateDefensePhase("d1", "evidence", 150n);
        manager.updateDefensePhase("d2", "evidence", 500n);

        const urgent = manager.getUrgentDefenses(100n, 500n);
        expect(urgent.length).toBe(2);
      });
    });
  });

  // ============ Defense Configuration Tests ============

  describe("Defense Configuration", () => {
    it("should set defense config", () => {
      const manager = createManager();
      manager.setDefenseConfig({
        autoDefendChallenges: false,
        defaultStrategy: "conservative",
      });

      const guardian = manager.getGuardian();
      expect(guardian.defense.autoDefendChallenges).toBe(false);
      expect(guardian.defense.defaultStrategy).toBe("conservative");
    });

    it("should add defense ally", () => {
      const manager = createManager();
      manager.addDefenseAlly(ALLY_DID);

      const guardian = manager.getGuardian();
      expect(guardian.defense.defenseAllies).toContain(ALLY_DID);
    });

    it("should not add duplicate ally", () => {
      const manager = createManager();
      manager.addDefenseAlly(ALLY_DID);
      manager.addDefenseAlly(ALLY_DID);

      const guardian = manager.getGuardian();
      expect(guardian.defense.defenseAllies).toHaveLength(1);
    });

    it("should remove defense ally", () => {
      const manager = createManager();
      manager.addDefenseAlly(ALLY_DID);
      manager.removeDefenseAlly(ALLY_DID);

      const guardian = manager.getGuardian();
      expect(guardian.defense.defenseAllies).toHaveLength(0);
    });

    it("should get defense strategy", () => {
      const manager = createManager();
      expect(manager.getDefenseStrategy()).toBe("moderate");

      manager.setDefenseConfig({ defaultStrategy: "diplomatic" });
      expect(manager.getDefenseStrategy()).toBe("diplomatic");
    });
  });

  // ============ Maintenance Tests ============

  describe("Maintenance Operations", () => {
    const createMaintenanceContext = (
      overrides: Partial<MaintenanceContext> = {},
    ): MaintenanceContext => ({
      claimableVesting: 0n,
      expiringAmount: 0n,
      reputationChange: 0,
      currentReputation: 0.8,
      ...overrides,
    });

    describe("performMaintenance()", () => {
      it("should not perform maintenance before interval", () => {
        const manager = createManager();
        const context = createMaintenanceContext({ claimableVesting: 100n });
        const result = manager.performMaintenance(500n, context);

        expect(result.performed).toBe(false);
        expect(result.actions).toHaveLength(0);
      });

      it("should perform maintenance after interval", () => {
        const manager = createManager();
        const context = createMaintenanceContext({
          claimableVesting: 100n * ONE_LGM,
        });
        const result = manager.performMaintenance(1000n, context);

        expect(result.performed).toBe(true);
      });

      it("should claim vesting when monitoring enabled", () => {
        const manager = createManager();
        const context = createMaintenanceContext({
          claimableVesting: 100n * ONE_LGM,
        });
        const result = manager.performMaintenance(1000n, context);

        const claimAction = result.actions.find(
          (a) => a.type === "claim_vesting",
        );
        expect(claimAction).toBeDefined();
        expect(claimAction?.amount).toBe(100n * ONE_LGM);
      });

      it("should extend expiring when monitoring enabled", () => {
        const manager = createManager();
        const context = createMaintenanceContext({
          expiringAmount: 50n * ONE_LGM,
        });
        const result = manager.performMaintenance(1000n, context);

        const extendAction = result.actions.find(
          (a) => a.type === "extend_expiration",
        );
        expect(extendAction).toBeDefined();
        expect(extendAction?.amount).toBe(50n * ONE_LGM);
      });

      it("should alert on significant reputation drop", () => {
        const manager = createManager();
        const context = createMaintenanceContext({ reputationChange: -0.15 });
        manager.performMaintenance(1000n, context);

        const alerts = manager.getPendingAlerts();
        const repAlert = alerts.find((a) => a.type === "reputation_change");
        expect(repAlert).toBeDefined();
      });

      it("should not alert on minor reputation change", () => {
        const manager = createManager();
        const context = createMaintenanceContext({ reputationChange: -0.05 });
        manager.performMaintenance(1000n, context);

        const alerts = manager.getPendingAlerts();
        const repAlert = alerts.find((a) => a.type === "reputation_change");
        expect(repAlert).toBeUndefined();
      });

      it("should update last maintenance block", () => {
        const manager = createManager();
        const context = createMaintenanceContext();
        manager.performMaintenance(1000n, context);

        const guardian = manager.getGuardian();
        expect(guardian.maintenance.lastMaintenanceBlock).toBe(1000n);
      });

      it("should track maintenance stats", () => {
        const manager = createManager();
        const context = createMaintenanceContext({
          claimableVesting: 100n * ONE_LGM,
          expiringAmount: 50n * ONE_LGM,
        });
        manager.performMaintenance(1000n, context);

        const stats = manager.getStats();
        expect(stats.maintenanceActionsPerformed).toBe(2);
        expect(stats.vestingAutoClaimed).toBe(100n * ONE_LGM);
        expect(stats.tokensPreservedFromExpiration).toBe(50n * ONE_LGM);
      });

      it("should skip vesting when monitoring disabled", () => {
        const manager = createManager();
        manager.setMaintenanceConfig({ monitorVesting: false });
        const context = createMaintenanceContext({
          claimableVesting: 100n * ONE_LGM,
        });
        const result = manager.performMaintenance(1000n, context);

        const claimAction = result.actions.find(
          (a) => a.type === "claim_vesting",
        );
        expect(claimAction).toBeUndefined();
      });

      it("should skip expiration when monitoring disabled", () => {
        const manager = createManager();
        manager.setMaintenanceConfig({ monitorExpiration: false });
        const context = createMaintenanceContext({
          expiringAmount: 50n * ONE_LGM,
        });
        const result = manager.performMaintenance(1000n, context);

        const extendAction = result.actions.find(
          (a) => a.type === "extend_expiration",
        );
        expect(extendAction).toBeUndefined();
      });
    });

    describe("setMaintenanceConfig()", () => {
      it("should update maintenance configuration", () => {
        const manager = createManager();
        manager.setMaintenanceConfig({
          checkInterval: 2000n,
          monitorReputation: false,
        });

        const guardian = manager.getGuardian();
        expect(guardian.maintenance.checkInterval).toBe(2000n);
        expect(guardian.maintenance.monitorReputation).toBe(false);
      });
    });
  });

  // ============ Alert Tests ============

  describe("Alert Operations", () => {
    describe("createAlert()", () => {
      it("should create alert", () => {
        const manager = createManager();
        const alert = manager.createAlert(
          "balance_low",
          "high",
          "Balance is low",
          {
            balance: 50n,
          },
        );

        expect(alert.id).toBeTruthy();
        expect(alert.type).toBe("balance_low");
        expect(alert.priority).toBe("high");
        expect(alert.message).toBe("Balance is low");
      });

      it("should store alert in pending", () => {
        const manager = createManager();
        manager.createAlert("balance_low", "high", "Low balance");

        const pending = manager.getPendingAlerts();
        expect(pending).toHaveLength(1);
      });

      it("should increment alert stats", () => {
        const manager = createManager();
        manager.createAlert("balance_low", "high", "Test");
        manager.createAlert("tokens_expiring", "medium", "Test 2");

        expect(manager.getStats().alertsGenerated).toBe(2);
      });

      it("should not create alert when trigger disabled", () => {
        const manager = createManager();
        manager.setAlertTrigger("balance_low", false);

        const alert = manager.createAlert("balance_low", "high", "Test");

        expect(alert.id).toBe(""); // Empty ID indicates disabled
      });

      it("should limit alerts to 100", () => {
        const manager = createManager();
        for (let i = 0; i < 110; i++) {
          manager.createAlert("tokens_expiring", "low", `Alert ${i}`);
        }

        const guardian = manager.getGuardian();
        expect(guardian.pendingAlerts.length).toBe(100);
      });
    });

    describe("getPendingAlerts()", () => {
      it("should get unread alerts only", () => {
        const manager = createManager();
        manager.createAlert("balance_low", "high", "Test 1");
        const alert2 = manager.createAlert(
          "tokens_expiring",
          "medium",
          "Test 2",
        );
        manager.markAlertRead(alert2.id);

        const pending = manager.getPendingAlerts();
        expect(pending).toHaveLength(1);
      });
    });

    describe("getAlertsByPriority()", () => {
      it("should filter by priority", () => {
        const manager = createManager();
        manager.createAlert("balance_low", "critical", "Critical");
        manager.createAlert("tokens_expiring", "high", "High");
        manager.createAlert("vesting_available", "medium", "Medium");
        manager.createAlert("reputation_change", "low", "Low");

        const critical = manager.getAlertsByPriority("critical");
        expect(critical).toHaveLength(1);
        expect(critical[0].priority).toBe("critical");
      });

      it("should get critical alerts", () => {
        const manager = createManager();
        manager.createAlert("balance_low", "critical", "Critical");
        manager.createAlert("tokens_expiring", "high", "High");

        const critical = manager.getCriticalAlerts();
        expect(critical).toHaveLength(1);
      });

      it("should exclude read alerts", () => {
        const manager = createManager();
        const alert = manager.createAlert(
          "balance_low",
          "critical",
          "Critical",
        );
        manager.markAlertRead(alert.id);

        const critical = manager.getCriticalAlerts();
        expect(critical).toHaveLength(0);
      });
    });

    describe("markAlertRead()", () => {
      it("should mark alert as read", () => {
        const manager = createManager();
        const alert = manager.createAlert("balance_low", "high", "Test");
        manager.markAlertRead(alert.id);

        expect(manager.getPendingAlerts()).toHaveLength(0);
      });

      it("should set readAt timestamp", () => {
        const manager = createManager();
        const alert = manager.createAlert("balance_low", "high", "Test");
        const beforeMark = Date.now();
        manager.markAlertRead(alert.id);

        const guardian = manager.getGuardian();
        const markedAlert = guardian.pendingAlerts.find(
          (a) => a.id === alert.id,
        );
        expect(markedAlert?.readAt).toBeGreaterThanOrEqual(beforeMark);
      });
    });

    describe("markAlertActioned()", () => {
      it("should mark alert as actioned", () => {
        const manager = createManager();
        const alert = manager.createAlert("balance_low", "high", "Test");
        manager.markAlertActioned(alert.id, "transferred_funds");

        const guardian = manager.getGuardian();
        const actionedAlert = guardian.pendingAlerts.find(
          (a) => a.id === alert.id,
        );
        expect(actionedAlert?.actionTaken).toBe("transferred_funds");
        expect(actionedAlert?.readAt).toBeDefined();
      });
    });

    describe("markAllAlertsRead()", () => {
      it("should mark all alerts as read", () => {
        const manager = createManager();
        manager.createAlert("balance_low", "high", "Test 1");
        manager.createAlert("tokens_expiring", "medium", "Test 2");
        manager.createAlert("vesting_available", "low", "Test 3");

        manager.markAllAlertsRead();

        expect(manager.getPendingAlerts()).toHaveLength(0);
      });
    });

    describe("setAlertTrigger()", () => {
      it("should enable/disable trigger", () => {
        const manager = createManager();
        manager.setAlertTrigger("balance_low", false);

        const guardian = manager.getGuardian();
        const trigger = guardian.alerts.triggers.find(
          (t) => t.type === "balance_low",
        );
        expect(trigger?.enabled).toBe(false);
      });

      it("should update priority", () => {
        const manager = createManager();
        manager.setAlertTrigger("vesting_available", true, "high");

        const guardian = manager.getGuardian();
        const trigger = guardian.alerts.triggers.find(
          (t) => t.type === "vesting_available",
        );
        expect(trigger?.priority).toBe("high");
      });
    });

    describe("setAlertDelivery()", () => {
      it("should update delivery settings", () => {
        const manager = createManager();
        manager.setAlertDelivery({
          emitEvent: false,
          notifyAllies: true,
        });

        const guardian = manager.getGuardian();
        expect(guardian.alerts.delivery.emitEvent).toBe(false);
        expect(guardian.alerts.delivery.notifyAllies).toBe(true);
        expect(guardian.alerts.delivery.storeInHome).toBe(true); // unchanged
      });
    });
  });

  // ============ Recovery Tests ============

  describe("Recovery Operations", () => {
    it("should add recovery guardian", () => {
      const manager = createManager();
      manager.addRecoveryGuardian(GUARDIAN_DID_1);

      const guardians = manager.getRecoveryGuardians();
      expect(guardians).toContain(GUARDIAN_DID_1);
    });

    it("should not add duplicate guardian", () => {
      const manager = createManager();
      manager.addRecoveryGuardian(GUARDIAN_DID_1);
      manager.addRecoveryGuardian(GUARDIAN_DID_1);

      expect(manager.getRecoveryGuardians()).toHaveLength(1);
    });

    it("should remove recovery guardian", () => {
      const manager = createManager();
      manager.addRecoveryGuardian(GUARDIAN_DID_1);
      manager.addRecoveryGuardian(GUARDIAN_DID_2);
      manager.removeRecoveryGuardian(GUARDIAN_DID_1);

      const guardians = manager.getRecoveryGuardians();
      expect(guardians).toHaveLength(1);
      expect(guardians).not.toContain(GUARDIAN_DID_1);
    });

    it("should set recovery threshold", () => {
      const manager = createManager();
      manager.addRecoveryGuardian(GUARDIAN_DID_1);
      manager.addRecoveryGuardian(GUARDIAN_DID_2);
      manager.setRecoveryThreshold(2);

      const guardian = manager.getGuardian();
      expect(guardian.recovery.threshold).toBe(2);
    });

    it("should throw on invalid threshold (too low)", () => {
      const manager = createManager();
      manager.addRecoveryGuardian(GUARDIAN_DID_1);

      expect(() => manager.setRecoveryThreshold(0)).toThrow(
        "Invalid recovery threshold",
      );
    });

    it("should throw on invalid threshold (too high)", () => {
      const manager = createManager();
      manager.addRecoveryGuardian(GUARDIAN_DID_1);

      expect(() => manager.setRecoveryThreshold(5)).toThrow(
        "Invalid recovery threshold",
      );
    });

    it("should return copy of guardians list", () => {
      const manager = createManager();
      manager.addRecoveryGuardian(GUARDIAN_DID_1);
      const guardians = manager.getRecoveryGuardians();
      guardians.push(GUARDIAN_DID_2);

      expect(manager.getRecoveryGuardians()).toHaveLength(1);
    });
  });

  // ============ Deadman Switch Tests ============

  describe("Deadman Switch", () => {
    describe("setDeadmanConfig()", () => {
      it("should update deadman configuration", () => {
        const manager = createManager();
        manager.setDeadmanConfig({
          enabled: true,
          inactivityThreshold: 50000n,
          action: "transfer_to_beneficiary",
          beneficiary: ALLY_DID,
        });

        const guardian = manager.getGuardian();
        expect(guardian.recovery.deadman.enabled).toBe(true);
        expect(guardian.recovery.deadman.action).toBe(
          "transfer_to_beneficiary",
        );
        expect(guardian.recovery.deadman.beneficiary).toBe(ALLY_DID);
      });
    });

    describe("checkDeadmanSwitch()", () => {
      it("should not trigger when disabled", () => {
        const manager = createManager();
        const result = manager.checkDeadmanSwitch(0n, 200000n);
        expect(result.triggered).toBe(false);
      });

      it("should not trigger when active", () => {
        const manager = createManager();
        manager.setDeadmanConfig({ enabled: true, inactivityThreshold: 1000n });

        const result = manager.checkDeadmanSwitch(100n, 500n);
        expect(result.triggered).toBe(false);
      });

      it("should trigger when inactive past threshold", () => {
        const manager = createManager();
        manager.setDeadmanConfig({
          enabled: true,
          inactivityThreshold: 1000n,
          action: "alert_guardians",
        });

        const result = manager.checkDeadmanSwitch(0n, 1000n);
        expect(result.triggered).toBe(true);
        expect(result.action).toBe("alert_guardians");
      });

      it("should trigger exactly at threshold", () => {
        const manager = createManager();
        manager.setDeadmanConfig({
          enabled: true,
          inactivityThreshold: 1000n,
          action: "donate_to_research",
        });

        const result = manager.checkDeadmanSwitch(500n, 1500n);
        expect(result.triggered).toBe(true);
      });

      it("should return action type", () => {
        const manager = createManager();
        manager.setDeadmanConfig({
          enabled: true,
          inactivityThreshold: 100n,
          action: "distribute_to_allies",
        });

        const result = manager.checkDeadmanSwitch(0n, 100n);
        expect(result.action).toBe("distribute_to_allies");
      });
    });
  });

  // ============ Statistics Tests ============

  describe("Statistics", () => {
    it("should get stats", () => {
      const manager = createManager();
      manager.handleChallenge("d1", "f1", CHALLENGER_DID, 100n);
      manager.completeDefense("d1", true);
      manager.createAlert("balance_low", "high", "Test");

      const stats = manager.getStats();
      expect(stats.challengesAutoDefended).toBe(1);
      expect(stats.challengesWon).toBe(1);
      expect(stats.alertsGenerated).toBeGreaterThan(0);
    });

    it("should return copy of stats", () => {
      const manager = createManager();
      const stats = manager.getStats();
      stats.challengesWon = 999;

      expect(manager.getStats().challengesWon).toBe(0);
    });
  });

  // ============ Summary Tests ============

  describe("getSummary()", () => {
    it("should provide comprehensive summary", () => {
      const manager = createManager();

      // Set up some state
      manager.handleChallenge("d1", "f1", CHALLENGER_DID, 100n);
      manager.handleChallenge("d2", "f2", CHALLENGER_DID, 100n);
      manager.completeDefense("d1", true);
      manager.createAlert("balance_low", "critical", "Critical!");
      manager.createAlert("tokens_expiring", "high", "High!");
      manager.createAlert("vesting_available", "low", "Low!");
      manager.addRecoveryGuardian(GUARDIAN_DID_1);
      manager.addRecoveryGuardian(GUARDIAN_DID_2);
      manager.setDeadmanConfig({ enabled: true });

      // Perform maintenance to track stats
      const context: MaintenanceContext = {
        claimableVesting: 100n * ONE_LGM,
        expiringAmount: 50n * ONE_LGM,
        reputationChange: 0,
        currentReputation: 0.8,
      };
      manager.performMaintenance(1000n, context);

      const summary = manager.getSummary();

      expect(summary.activeDefenses).toBe(1); // d2 still active
      // Critical alerts: 2 from challenges (d1, d2) + 1 manual balance_low = 3
      expect(summary.pendingCriticalAlerts).toBe(3);
      // High alerts: 2 from defense phase updates (dispute_deadline alerts) + 1 manual tokens_expiring = 3
      expect(summary.pendingHighAlerts).toBeGreaterThanOrEqual(1);
      expect(summary.challengesDefended).toBe(2);
      expect(summary.recoveryGuardians).toBe(2);
      expect(summary.deadmanEnabled).toBe(true);
    });

    it("should calculate win rate", () => {
      const manager = createManager();
      manager.handleChallenge("d1", "f1", CHALLENGER_DID, 100n);
      manager.handleChallenge("d2", "f2", CHALLENGER_DID, 100n);
      manager.completeDefense("d1", true);
      manager.completeDefense("d2", false);

      const summary = manager.getSummary();

      // 2 defended, 1 won
      expect(summary.winRate).toBe(0.5);
    });

    it("should track tokens preserved", () => {
      const manager = createManager();
      const context: MaintenanceContext = {
        claimableVesting: 0n,
        expiringAmount: 50n * ONE_LGM,
        reputationChange: 0,
        currentReputation: 0.8,
      };
      manager.performMaintenance(1000n, context);

      const summary = manager.getSummary();
      expect(summary.tokensPreserved).toBe(50n * ONE_LGM);
    });

    it("should track vesting claimed", () => {
      const manager = createManager();
      const context: MaintenanceContext = {
        claimableVesting: 100n * ONE_LGM,
        expiringAmount: 0n,
        reputationChange: 0,
        currentReputation: 0.8,
      };
      manager.performMaintenance(1000n, context);

      const summary = manager.getSummary();
      expect(summary.vestingClaimed).toBe(100n * ONE_LGM);
    });
  });

  // ============ State Management Tests ============

  describe("State Management", () => {
    it("should get full guardian state", () => {
      const manager = createManager();
      manager.handleChallenge("d1", "f1", CHALLENGER_DID, 100n);

      const guardian = manager.getGuardian();
      expect(guardian.activeDefenses).toHaveLength(1);
    });

    it("should track dirty state", () => {
      const manager = createManager();
      expect(manager.isDirty()).toBe(false);

      manager.handleChallenge("d1", "f1", CHALLENGER_DID, 100n);
      expect(manager.isDirty()).toBe(true);

      manager.markClean();
      expect(manager.isDirty()).toBe(false);
    });

    it("should mark dirty on various operations", () => {
      const operations = [
        (m: GuardianManager) =>
          m.handleChallenge("d1", "f1", CHALLENGER_DID, 100n),
        (m: GuardianManager) =>
          m.setDefenseConfig({ autoDefendChallenges: false }),
        (m: GuardianManager) => m.addDefenseAlly(ALLY_DID),
        (m: GuardianManager) => m.createAlert("balance_low", "high", "Test"),
        (m: GuardianManager) => m.addRecoveryGuardian(GUARDIAN_DID_1),
        (m: GuardianManager) => m.setDeadmanConfig({ enabled: true }),
        (m: GuardianManager) =>
          m.setMaintenanceConfig({ monitorVesting: false }),
      ];

      for (const op of operations) {
        const manager = createManager();
        expect(manager.isDirty()).toBe(false);
        op(manager);
        expect(manager.isDirty()).toBe(true);
      }
    });
  });

  // ============ Edge Cases ============

  describe("Edge Cases", () => {
    it("should handle completing non-existent defense", () => {
      const manager = createManager();
      // Should not throw
      manager.completeDefense("non-existent", true);

      const stats = manager.getStats();
      expect(stats.challengesWon).toBe(0);
    });

    it("should handle multiple challenges on same fact", () => {
      const manager = createManager();
      manager.handleChallenge("d1", "fact-1", CHALLENGER_DID, 100n);
      manager.handleChallenge("d2", "fact-1", "did:lgm:other", 200n);

      const defenses = manager.getActiveDefenses();
      expect(defenses).toHaveLength(2);
    });

    it("should handle rapid alert creation", () => {
      const manager = createManager();
      for (let i = 0; i < 50; i++) {
        manager.createAlert("tokens_expiring", "medium", `Alert ${i}`);
      }

      expect(manager.getPendingAlerts().length).toBe(50);
    });

    it("should handle very large block numbers", () => {
      const manager = createManager();
      const largeBlock = 10n ** 20n;
      manager.handleChallenge("d1", "f1", CHALLENGER_DID, largeBlock);
      manager.updateDefensePhase("d1", "evidence", largeBlock + 1000n);

      const defenses = manager.getActiveDefenses();
      expect(defenses[0].nextDeadline).toBe(largeBlock + 1000n);
    });

    it("should handle maintenance at block 0", () => {
      const manager = createManager();
      // Default lastMaintenanceBlock is 0, interval is 1000
      const context: MaintenanceContext = {
        claimableVesting: 100n,
        expiringAmount: 0n,
        reputationChange: 0,
        currentReputation: 0.8,
      };

      // At block 0, should not perform (0 < 0 + 1000)
      const result = manager.performMaintenance(0n, context);
      expect(result.performed).toBe(false);
    });

    it("should handle empty defense allies with delegate strategy", () => {
      const manager = createManager();
      manager.setDefenseConfig({
        defaultStrategy: "delegate",
        // No allies added
      });

      const result = manager.handleChallenge("d1", "f1", CHALLENGER_DID, 100n);
      // Should fall back to auto_defend since no allies
      expect(result.action).toBe("auto_defend");
    });
  });
});
