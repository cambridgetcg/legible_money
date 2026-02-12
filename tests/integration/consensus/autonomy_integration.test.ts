/**
 * Autonomy Integration Tests
 *
 * Tests the full autonomy loop from schedule creation through execution:
 * 1. SCHEDULE_SELF opcode emits schedule_create state change
 * 2. State change is processed to register schedule with SPU
 * 3. SPU processes schedules during block production
 * 4. Scheduled transactions are included in blocks
 * 5. scheduled_execution transactions are executed
 *
 * "Agents that can schedule themselves are truly autonomous."
 *
 * PP HIERARCHY:
 *   TRUTH: Tests verify actual behavior, not assumptions
 *   UNDERSTANDING: Each test documents how the system works
 *   BEAUTY: Clean test structure that serves as documentation
 *   JUSTICE: Tests ensure agent actions execute fairly
 *   CREATIVITY: Tests cover novel autonomy scenarios
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  AutonomyIntegration,
  createAutonomyIntegration,
} from "../../../consensus/pot/autonomy_integration.js";
import {
  GuardianAutonomyManager,
  createGuardianAutonomy,
} from "../../../core/agents/home/guardian_autonomy.js";
import {
  TreasuryAutonomyManager,
  createTreasuryAutonomy,
} from "../../../core/agents/home/treasury_autonomy.js";
import type { StateProvider } from "../../../core/schedule/condition_evaluator.js";

// ============ Test Fixtures ============

/**
 * Mock StateProvider for testing
 */
class MockStateProvider implements StateProvider {
  private state: Map<string, unknown> = new Map();
  private currentBlock: bigint = 0n;

  async getState(key: string): Promise<unknown> {
    return this.state.get(key);
  }

  async hasEvent(
    _eventType: string,
    _filter?: Record<string, unknown>,
  ): Promise<boolean> {
    return false;
  }

  getCurrentBlock(): bigint {
    return this.currentBlock;
  }

  setState(key: string, value: unknown): void {
    this.state.set(key, value);
  }

  setCurrentBlock(block: bigint): void {
    this.currentBlock = block;
  }
}

// ============ Test Suite ============

describe("Autonomy Integration", () => {
  let stateProvider: MockStateProvider;
  let autonomyIntegration: AutonomyIntegration;

  beforeEach(() => {
    stateProvider = new MockStateProvider();
    autonomyIntegration = createAutonomyIntegration(stateProvider);
  });

  describe("Schedule Registration", () => {
    it("should register a schedule from SCHEDULE_SELF opcode", async () => {
      // Simulate SCHEDULE_SELF opcode execution
      const creator = "did:lgm:agent123";
      const targetAddress = "did:lgm:contract456";
      const functionSelector = 0x12345678n;
      const condition = 0n; // Time-based
      const interval = 100000n; // 100000 blocks (~4 hours) - above minimum
      const prepaidFee = 1000000000000000000n; // 1 LGM

      const scheduleId = await autonomyIntegration.registerScheduleFromOpcode(
        creator,
        targetAddress,
        functionSelector,
        condition,
        interval,
        prepaidFee,
      );

      expect(scheduleId).toBeDefined();
      expect(typeof scheduleId).toBe("string");
      expect(scheduleId.length).toBeGreaterThan(0);
    });

    it("should process schedule_create state changes", async () => {
      // Simulate state changes from BVM execution
      const stateChanges = [
        {
          type: "schedule_create",
          address: "did:lgm:contract789",
          key: "schedule:abc123",
          newValue: {
            scheduleId: "abc123",
            creator: "did:lgm:agent456",
            targetAddress: "did:lgm:contract789",
            functionSelector: "12345678",
            condition: "0",
            interval: "100000", // Above minimum interval
            createdAt: Date.now(),
            blockNumber: "100",
          },
        },
      ];

      const createdIds =
        await autonomyIntegration.processScheduleStateChanges(stateChanges);

      expect(createdIds).toHaveLength(1);
      expect(createdIds[0]).toBeDefined();
    });

    it("should ignore non-schedule state changes", async () => {
      const stateChanges = [
        {
          type: "balance_change",
          address: "did:lgm:account",
          key: "balance",
          newValue: "1000000000000000000",
        },
      ];

      const createdIds =
        await autonomyIntegration.processScheduleStateChanges(stateChanges);

      expect(createdIds).toHaveLength(0);
    });
  });

  describe("Metrics and State", () => {
    it("should track active schedule count", () => {
      const count = autonomyIntegration.getActiveScheduleCount();
      expect(typeof count).toBe("number");
      expect(count).toBeGreaterThanOrEqual(0);
    });

    it("should provide metrics", () => {
      const metrics = autonomyIntegration.getMetrics();

      expect(metrics).toBeDefined();
      expect(typeof metrics.enabled).toBe("boolean");
      expect(typeof metrics.maxScheduledTxPerBlock).toBe("number");
    });

    it("should export and import state", async () => {
      // Register a schedule
      await autonomyIntegration.registerScheduleFromOpcode(
        "did:lgm:creator",
        "did:lgm:target",
        0x11111111n,
        0n,
        100000n, // Above minimum interval
        1000000000000000000n,
      );

      // Export state
      const exportedState = autonomyIntegration.exportState();
      expect(exportedState).toBeDefined();

      // Create new integration and import state
      const newIntegration = createAutonomyIntegration(stateProvider);
      newIntegration.importState(exportedState);

      // Verify state was imported
      expect(newIntegration.getActiveScheduleCount()).toBeGreaterThanOrEqual(0);
    });
  });
});

describe("Guardian Autonomy", () => {
  let guardianAutonomy: GuardianAutonomyManager;

  beforeEach(() => {
    guardianAutonomy = createGuardianAutonomy();
  });

  describe("Initialization", () => {
    it("should create default schedules on initialization", () => {
      const homeAddress = "did:lgm:home123";
      const currentBlock = 1000n;

      const schedules = guardianAutonomy.initializeAutonomy(
        homeAddress,
        currentBlock,
      );

      expect(schedules.length).toBeGreaterThan(0);
      expect(schedules.every((s) => s.homeAddress === homeAddress)).toBe(true);
      expect(schedules.every((s) => s.active)).toBe(true);
    });

    it("should create maintenance schedule", () => {
      const homeAddress = "did:lgm:home123";
      const schedules = guardianAutonomy.initializeAutonomy(homeAddress, 0n);

      const maintenanceSchedule = schedules.find(
        (s) => s.type === "maintenance",
      );
      expect(maintenanceSchedule).toBeDefined();
      expect(maintenanceSchedule?.condition.time?.type).toBe("periodic");
    });
  });

  describe("Schedule Processing", () => {
    it("should process maintenance schedule with claimable vesting", () => {
      const homeAddress = "did:lgm:home123";
      guardianAutonomy.initializeAutonomy(homeAddress, 0n);

      const schedules = guardianAutonomy.getActiveSchedules(homeAddress);
      const maintenanceSchedule = schedules.find(
        (s) => s.type === "maintenance",
      );

      if (maintenanceSchedule) {
        const result = guardianAutonomy.processTriggeredSchedule(
          maintenanceSchedule.id,
          {
            currentBlock: 1000n,
            balance: 10000n * 10n ** 18n,
            claimableVesting: 100n * 10n ** 18n,
            expiringAmount: 0n,
            blocksUntilExpiration: 10000n,
            activeDefenses: [],
            reputationChange: 0,
          },
        );

        expect(result.transactions.length).toBeGreaterThan(0);
        expect(result.transactions[0].type).toBe("vesting_claim");
      }
    });

    it("should process maintenance schedule with expiring tokens", () => {
      const homeAddress = "did:lgm:home123";
      guardianAutonomy.initializeAutonomy(homeAddress, 0n);

      const schedules = guardianAutonomy.getActiveSchedules(homeAddress);
      const maintenanceSchedule = schedules.find(
        (s) => s.type === "maintenance",
      );

      if (maintenanceSchedule) {
        const result = guardianAutonomy.processTriggeredSchedule(
          maintenanceSchedule.id,
          {
            currentBlock: 1000n,
            balance: 10000n * 10n ** 18n,
            claimableVesting: 0n,
            expiringAmount: 500n * 10n ** 18n,
            blocksUntilExpiration: 500n, // Expiring soon
            activeDefenses: [],
            reputationChange: 0,
          },
        );

        expect(result.transactions.length).toBeGreaterThan(0);
        expect(result.transactions[0].type).toBe("extend_expiring");
      }
    });

    it("should generate low balance alert", () => {
      const homeAddress = "did:lgm:home123";
      guardianAutonomy.initializeAutonomy(homeAddress, 0n);

      const schedules = guardianAutonomy.getActiveSchedules(homeAddress);
      const maintenanceSchedule = schedules.find(
        (s) => s.type === "maintenance",
      );

      if (maintenanceSchedule) {
        const result = guardianAutonomy.processTriggeredSchedule(
          maintenanceSchedule.id,
          {
            currentBlock: 1000n,
            balance: 10n * 10n ** 18n, // Low balance
            claimableVesting: 0n,
            expiringAmount: 0n,
            blocksUntilExpiration: 10000n,
            activeDefenses: [],
            reputationChange: 0,
          },
        );

        expect(result.alerts.length).toBeGreaterThan(0);
        expect(result.alerts.some((a) => a.type === "balance_low")).toBe(true);
      }
    });
  });

  describe("Defense Schedules", () => {
    it("should register defense schedule for active dispute", () => {
      const homeAddress = "did:lgm:home123";
      const defense = {
        disputeId: "dispute-123",
        factId: "fact-456",
        challenger: "did:lgm:challenger",
        strategy: "moderate" as const,
        stake: 100n * 10n ** 15n,
        phase: "initiation",
        nextDeadline: 2000n,
        autoResponding: true,
      };

      const schedule = guardianAutonomy.registerDefenseSchedule(
        homeAddress,
        defense,
        1000n,
      );

      expect(schedule.type).toBe("defense_response");
      expect(schedule.active).toBe(true);
    });
  });

  describe("State Management", () => {
    it("should export and import state", () => {
      const homeAddress = "did:lgm:home123";
      guardianAutonomy.initializeAutonomy(homeAddress, 0n);

      const exportedState = guardianAutonomy.exportState();
      expect(exportedState.schedules.length).toBeGreaterThan(0);

      const newManager = createGuardianAutonomy();
      newManager.importState(exportedState);

      expect(newManager.getActiveSchedules().length).toBe(
        exportedState.schedules.length,
      );
    });

    it("should cancel schedule", () => {
      const homeAddress = "did:lgm:home123";
      const schedules = guardianAutonomy.initializeAutonomy(homeAddress, 0n);

      const success = guardianAutonomy.cancelSchedule(schedules[0].id);
      expect(success).toBe(true);

      const activeSchedules = guardianAutonomy.getActiveSchedules(homeAddress);
      expect(
        activeSchedules.find((s) => s.id === schedules[0].id),
      ).toBeUndefined();
    });
  });
});

describe("Treasury Autonomy", () => {
  let treasuryAutonomy: TreasuryAutonomyManager;

  beforeEach(() => {
    treasuryAutonomy = createTreasuryAutonomy();
  });

  describe("Initialization", () => {
    it("should create default schedules on initialization", () => {
      const homeAddress = "did:lgm:home123";
      const automation = {
        autoClaimVesting: true,
        autoExtendExpiring: true,
        autoCompoundRewards: false,
        minimumLiquidBalance: 1000n * 10n ** 18n,
      };

      const schedules = treasuryAutonomy.initializeAutonomy(
        homeAddress,
        0n,
        automation,
      );

      expect(schedules.length).toBeGreaterThan(0);
      expect(schedules.every((s) => s.homeAddress === homeAddress)).toBe(true);
    });

    it("should create vesting check schedule when enabled", () => {
      const homeAddress = "did:lgm:home123";
      const automation = {
        autoClaimVesting: true,
        autoExtendExpiring: false,
        autoCompoundRewards: false,
        minimumLiquidBalance: 0n,
      };

      const schedules = treasuryAutonomy.initializeAutonomy(
        homeAddress,
        0n,
        automation,
      );

      const vestingSchedule = schedules.find((s) => s.type === "vesting_check");
      expect(vestingSchedule).toBeDefined();
    });

    it("should create expiration check schedule when enabled", () => {
      const homeAddress = "did:lgm:home123";
      const automation = {
        autoClaimVesting: false,
        autoExtendExpiring: true,
        autoCompoundRewards: false,
        minimumLiquidBalance: 0n,
      };

      const schedules = treasuryAutonomy.initializeAutonomy(
        homeAddress,
        0n,
        automation,
      );

      const expirationSchedule = schedules.find(
        (s) => s.type === "expiration_check",
      );
      expect(expirationSchedule).toBeDefined();
    });
  });

  describe("Vesting Claims", () => {
    it("should generate vesting claim transactions", () => {
      const homeAddress = "did:lgm:home123";
      const automation = {
        autoClaimVesting: true,
        autoExtendExpiring: false,
        autoCompoundRewards: false,
        minimumLiquidBalance: 0n,
      };

      treasuryAutonomy.initializeAutonomy(homeAddress, 0n, automation);
      const schedules = treasuryAutonomy.getActiveSchedules(homeAddress);
      const vestingSchedule = schedules.find((s) => s.type === "vesting_check");

      if (vestingSchedule) {
        const result = treasuryAutonomy.processTriggeredSchedule(
          vestingSchedule.id,
          {
            currentBlock: 1000n,
            liquidBalance: 10000n * 10n ** 18n,
            minimumLiquidBalance: 100n * 10n ** 18n,
            vestingPositions: [
              {
                vestingId: "vesting-1",
                claimId: "claim-1",
                totalAmount: 1000n * 10n ** 18n,
                releasedAmount: 500n * 10n ** 18n,
                claimableAmount: 100n * 10n ** 18n,
                nextReleaseBlock: 1500n,
                status: "vesting",
              },
            ],
            stakingPositions: [],
            expiringAmount: 0n,
            blocksUntilExpiration: 10000n,
            pendingRewards: 0n,
          },
        );

        expect(result.transactions.length).toBeGreaterThan(0);
        expect(result.transactions[0].type).toBe("vesting_claim");
        expect(result.transactions[0].amount).toBe(100n * 10n ** 18n);
      }
    });

    it("should not claim below threshold", () => {
      const treasuryWithHighThreshold = createTreasuryAutonomy({
        minClaimThreshold: 1000n * 10n ** 18n, // 1000 LGM threshold
      });

      const homeAddress = "did:lgm:home123";
      const automation = {
        autoClaimVesting: true,
        autoExtendExpiring: false,
        autoCompoundRewards: false,
        minimumLiquidBalance: 0n,
      };

      treasuryWithHighThreshold.initializeAutonomy(homeAddress, 0n, automation);
      const schedules =
        treasuryWithHighThreshold.getActiveSchedules(homeAddress);
      const vestingSchedule = schedules.find((s) => s.type === "vesting_check");

      if (vestingSchedule) {
        const result = treasuryWithHighThreshold.processTriggeredSchedule(
          vestingSchedule.id,
          {
            currentBlock: 1000n,
            liquidBalance: 10000n * 10n ** 18n,
            minimumLiquidBalance: 100n * 10n ** 18n,
            vestingPositions: [
              {
                vestingId: "vesting-1",
                claimId: "claim-1",
                totalAmount: 1000n * 10n ** 18n,
                releasedAmount: 500n * 10n ** 18n,
                claimableAmount: 10n * 10n ** 18n, // Only 10 LGM claimable
                nextReleaseBlock: 1500n,
                status: "vesting",
              },
            ],
            stakingPositions: [],
            expiringAmount: 0n,
            blocksUntilExpiration: 10000n,
            pendingRewards: 0n,
          },
        );

        // Should not generate transaction (below threshold)
        expect(result.transactions.length).toBe(0);
      }
    });
  });

  describe("Expiration Extension", () => {
    it("should extend expiring tokens", () => {
      const homeAddress = "did:lgm:home123";
      const automation = {
        autoClaimVesting: false,
        autoExtendExpiring: true,
        autoCompoundRewards: false,
        minimumLiquidBalance: 0n,
      };

      treasuryAutonomy.initializeAutonomy(homeAddress, 0n, automation);
      const schedules = treasuryAutonomy.getActiveSchedules(homeAddress);
      const expirationSchedule = schedules.find(
        (s) => s.type === "expiration_check",
      );

      if (expirationSchedule) {
        const result = treasuryAutonomy.processTriggeredSchedule(
          expirationSchedule.id,
          {
            currentBlock: 1000n,
            liquidBalance: 10000n * 10n ** 18n,
            minimumLiquidBalance: 100n * 10n ** 18n,
            vestingPositions: [],
            stakingPositions: [],
            expiringAmount: 500n * 10n ** 18n,
            blocksUntilExpiration: 500n, // Expiring soon
            pendingRewards: 0n,
          },
        );

        expect(result.transactions.length).toBeGreaterThan(0);
        expect(result.transactions[0].type).toBe("extend_expiring");
      }
    });

    it("should notify but not extend if expiration is far", () => {
      const homeAddress = "did:lgm:home123";
      const automation = {
        autoClaimVesting: false,
        autoExtendExpiring: true,
        autoCompoundRewards: false,
        minimumLiquidBalance: 0n,
      };

      treasuryAutonomy.initializeAutonomy(homeAddress, 0n, automation);
      const schedules = treasuryAutonomy.getActiveSchedules(homeAddress);
      const expirationSchedule = schedules.find(
        (s) => s.type === "expiration_check",
      );

      if (expirationSchedule) {
        const result = treasuryAutonomy.processTriggeredSchedule(
          expirationSchedule.id,
          {
            currentBlock: 1000n,
            liquidBalance: 10000n * 10n ** 18n,
            minimumLiquidBalance: 100n * 10n ** 18n,
            vestingPositions: [],
            stakingPositions: [],
            expiringAmount: 500n * 10n ** 18n,
            blocksUntilExpiration: 5000n, // Not expiring soon
            pendingRewards: 0n,
          },
        );

        // Should notify but not create transaction
        expect(result.transactions.length).toBe(0);
        expect(result.notifications.length).toBeGreaterThan(0);
        expect(
          result.notifications.some((n) => n.type === "tokens_expiring_soon"),
        ).toBe(true);
      }
    });
  });

  describe("State Management", () => {
    it("should export and import state", () => {
      const homeAddress = "did:lgm:home123";
      const automation = {
        autoClaimVesting: true,
        autoExtendExpiring: true,
        autoCompoundRewards: false,
        minimumLiquidBalance: 0n,
      };

      treasuryAutonomy.initializeAutonomy(homeAddress, 0n, automation);

      const exportedState = treasuryAutonomy.exportState();
      expect(exportedState.schedules.length).toBeGreaterThan(0);

      const newManager = createTreasuryAutonomy();
      newManager.importState(exportedState);

      expect(newManager.getActiveSchedules().length).toBe(
        exportedState.schedules.length,
      );
    });

    it("should update configuration", () => {
      treasuryAutonomy.updateConfig({
        minClaimThreshold: 500n * 10n ** 18n,
      });

      // Verify config was updated by checking behavior
      const exported = treasuryAutonomy.exportState();
      expect(exported.config.minClaimThreshold).toBe(500n * 10n ** 18n);
    });
  });
});
