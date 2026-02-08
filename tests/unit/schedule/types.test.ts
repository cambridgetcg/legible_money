/**
 * Schedule Types Unit Tests
 *
 * Tests for DMSS type definitions, exported constants, and error classes.
 * Validates the structural contracts that the rest of the schedule module depends on.
 */

import { describe, it, expect } from "vitest";
import {
  DEFAULT_SCHEDULE_CONFIG,
  ScheduleError,
  type ScheduleConfig,
  type ScheduleMessage,
  type ScheduleProcess,
  type ScheduleCondition,
  type TimeCondition,
  type LogicCondition,
  type CompoundCondition,
  type ScheduleProcessStatus,
  type ScheduleType,
  type Comparator,
  type ExecutionResult,
  type LinkedEntityType,
  type ScheduleEvent,
  type ScheduleModification,
  type ScheduleModificationAction,
  type ConditionProof,
  type ScheduleErrorCode,
} from "../../../core/schedule/index.js";

// ============ DEFAULT_SCHEDULE_CONFIG Tests ============

describe("DEFAULT_SCHEDULE_CONFIG", () => {
  it("should export a valid default configuration object", () => {
    expect(DEFAULT_SCHEDULE_CONFIG).toBeDefined();
    expect(typeof DEFAULT_SCHEDULE_CONFIG).toBe("object");
  });

  it("should have correct processing limits", () => {
    expect(DEFAULT_SCHEDULE_CONFIG.maxActiveSchedulesPerAccount).toBe(100);
    expect(DEFAULT_SCHEDULE_CONFIG.maxScheduleExecutionGas).toBe(1_000_000n);
    expect(DEFAULT_SCHEDULE_CONFIG.maxWatchedKeysPerProcess).toBe(10);
    expect(DEFAULT_SCHEDULE_CONFIG.maxCompoundConditionDepth).toBe(3);
  });

  it("should have correct timing defaults", () => {
    // Minimum interval should equal one block time (2521ms)
    expect(DEFAULT_SCHEDULE_CONFIG.minScheduleIntervalMs).toBe(2521);
    // 1-minute time buckets
    expect(DEFAULT_SCHEDULE_CONFIG.timeBucketSizeMs).toBe(60_000);
    // 1-year maximum future schedule
    expect(DEFAULT_SCHEDULE_CONFIG.maxFutureScheduleMs).toBe(
      365 * 24 * 60 * 60 * 1000,
    );
  });

  it("should have correct fee defaults", () => {
    expect(DEFAULT_SCHEDULE_CONFIG.minPrepaidExecutions).toBe(10);
    expect(DEFAULT_SCHEDULE_CONFIG.feeBufferFactor).toBe(1.5);
  });

  it("should have correct cleanup defaults", () => {
    expect(DEFAULT_SCHEDULE_CONFIG.expiredProcessRetentionBlocks).toBe(1000);
  });

  it("should have all required ScheduleConfig fields", () => {
    const config: ScheduleConfig = DEFAULT_SCHEDULE_CONFIG;
    const requiredKeys: Array<keyof ScheduleConfig> = [
      "maxActiveSchedulesPerAccount",
      "maxScheduleExecutionGas",
      "maxWatchedKeysPerProcess",
      "maxCompoundConditionDepth",
      "minScheduleIntervalMs",
      "timeBucketSizeMs",
      "maxFutureScheduleMs",
      "minPrepaidExecutions",
      "feeBufferFactor",
      "expiredProcessRetentionBlocks",
    ];
    for (const key of requiredKeys) {
      expect(config[key]).toBeDefined();
    }
  });
});

// ============ ScheduleError Tests ============

describe("ScheduleError", () => {
  it("should be an instance of Error", () => {
    const err = new ScheduleError("test error", "INVALID_CONDITION");
    expect(err).toBeInstanceOf(Error);
    expect(err).toBeInstanceOf(ScheduleError);
  });

  it("should store message and error code", () => {
    const err = new ScheduleError("something went wrong", "PROCESS_NOT_FOUND");
    expect(err.message).toBe("something went wrong");
    expect(err.code).toBe("PROCESS_NOT_FOUND");
  });

  it("should have name set to ScheduleError", () => {
    const err = new ScheduleError("test", "INVALID_CONDITION");
    expect(err.name).toBe("ScheduleError");
  });

  it("should store optional details", () => {
    const details = { limit: 100, current: 101 };
    const err = new ScheduleError(
      "limit exceeded",
      "MAX_SCHEDULES_EXCEEDED",
      details,
    );
    expect(err.details).toEqual(details);
  });

  it("should have undefined details when not provided", () => {
    const err = new ScheduleError("no details", "UNAUTHORIZED");
    expect(err.details).toBeUndefined();
  });

  it("should support all defined error codes", () => {
    const codes: ScheduleErrorCode[] = [
      "INVALID_CONDITION",
      "INVALID_SCHEDULE_TYPE",
      "INSUFFICIENT_FEE",
      "MAX_SCHEDULES_EXCEEDED",
      "PROCESS_NOT_FOUND",
      "PROCESS_NOT_MODIFIABLE",
      "UNAUTHORIZED",
      "CONDITION_TOO_COMPLEX",
      "INTERVAL_TOO_SHORT",
      "SCHEDULE_TOO_FAR_FUTURE",
      "ALREADY_PAUSED",
      "NOT_PAUSED",
      "ALREADY_COMPLETED",
      "EXECUTION_FAILED",
      "INVALID_FEE",
    ];

    for (const code of codes) {
      const err = new ScheduleError(`error: ${code}`, code);
      expect(err.code).toBe(code);
    }
  });

  it("should have a proper stack trace", () => {
    const err = new ScheduleError("stack check", "INVALID_CONDITION");
    expect(err.stack).toBeDefined();
    expect(err.stack).toContain("ScheduleError");
  });
});

// ============ Type Shape Validation Tests ============

describe("Type shape validation", () => {
  it("should allow constructing a valid ScheduleCondition with time only", () => {
    const condition: ScheduleCondition = {
      time: {
        type: "absolute",
        executeAt: Date.now() + 10_000,
      },
    };
    expect(condition.time).toBeDefined();
    expect(condition.logic).toBeUndefined();
    expect(condition.compound).toBeUndefined();
  });

  it("should allow constructing a valid ScheduleCondition with logic only", () => {
    const condition: ScheduleCondition = {
      logic: {
        type: "state_compare",
        stateKey: "balance:account1",
        comparator: "gt",
        compareValue: 100n,
      },
    };
    expect(condition.logic).toBeDefined();
    expect(condition.time).toBeUndefined();
  });

  it("should allow constructing a compound condition", () => {
    const condition: ScheduleCondition = {
      compound: {
        operator: "and",
        conditions: [
          {
            time: { type: "absolute", executeAt: 1000 },
          },
          {
            logic: {
              type: "state_compare",
              stateKey: "key",
              comparator: "eq",
              compareValue: 42,
            },
          },
        ],
      },
    };
    expect(condition.compound).toBeDefined();
    expect(condition.compound!.conditions).toHaveLength(2);
  });

  it("should allow constructing ExecutionResult success variant", () => {
    const result: ExecutionResult = {
      success: true,
      txId: "abc123",
      gasUsed: 5000n,
    };
    expect(result.success).toBe(true);
    if (result.success) {
      expect(result.txId).toBe("abc123");
      expect(result.gasUsed).toBe(5000n);
    }
  });

  it("should allow constructing ExecutionResult failure variant", () => {
    const result: ExecutionResult = {
      success: false,
      error: "out of gas",
      gasUsed: 0n,
    };
    expect(result.success).toBe(false);
    if (!result.success) {
      expect(result.error).toBe("out of gas");
    }
  });

  it("should allow all Comparator values", () => {
    const comparators: Comparator[] = [
      "eq",
      "ne",
      "gt",
      "gte",
      "lt",
      "lte",
      "contains",
      "not_contains",
    ];
    expect(comparators).toHaveLength(8);
  });

  it("should allow all ScheduleProcessStatus values", () => {
    const statuses: ScheduleProcessStatus[] = [
      "pending",
      "active",
      "executing",
      "paused",
      "completed",
      "cancelled",
      "exhausted",
      "expired",
    ];
    expect(statuses).toHaveLength(8);
  });

  it("should allow all ScheduleType values", () => {
    const types: ScheduleType[] = [
      "time_absolute",
      "time_relative",
      "time_periodic",
      "logic_condition",
      "logic_state_change",
      "time_and_logic",
      "time_or_logic",
      "verification_deadline",
      "challenge_deadline",
      "confidence_decay",
      "vesting_release",
      "expiration_clawback",
      "rai_distribution",
    ];
    expect(types).toHaveLength(13);
  });

  it("should allow all LinkedEntityType values", () => {
    const entityTypes: LinkedEntityType[] = [
      "claim",
      "dispute",
      "vesting",
      "validator",
      "bounty",
      "rai",
      "service",
    ];
    expect(entityTypes).toHaveLength(7);
  });
});
