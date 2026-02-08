/**
 * Schedule Condition Evaluator Unit Tests
 *
 * Tests for the ConditionEvaluator, which determines if schedule conditions
 * (time-based, logic-based, and compound) are satisfied.
 */

import { describe, it, expect, beforeEach } from "vitest";
import {
  ConditionEvaluator,
  type StateProvider,
  type EvaluationResult,
} from "../../../core/schedule/condition_evaluator.js";
import {
  ScheduleError,
  type ScheduleCondition,
  type TimeCondition,
  type LogicCondition,
  type ScheduleProcess,
} from "../../../core/schedule/index.js";

// ============ Mock StateProvider ============

function createMockStateProvider(
  overrides: Partial<StateProvider> = {},
): StateProvider {
  const stateStore = new Map<string, unknown>();
  const eventStore = new Set<string>();
  let currentBlock = 100n;

  return {
    getState: async (key: string) => stateStore.get(key),
    hasEvent: async (eventType: string, _filter?: Record<string, unknown>) =>
      eventStore.has(eventType),
    getCurrentBlock: () => currentBlock,
    // Helpers for tests (not part of interface, accessed via casting)
    ...({
      _setState: (key: string, value: unknown) => stateStore.set(key, value),
      _addEvent: (eventType: string) => eventStore.add(eventType),
      _setBlock: (block: bigint) => {
        currentBlock = block;
      },
    } as Record<string, unknown>),
    ...overrides,
  };
}

type MockStateProvider = StateProvider & {
  _setState: (key: string, value: unknown) => void;
  _addEvent: (eventType: string) => void;
  _setBlock: (block: bigint) => void;
};

// ============ Time Condition Tests: Absolute ============

describe("ConditionEvaluator - Absolute Time", () => {
  let stateProvider: MockStateProvider;
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    stateProvider = createMockStateProvider() as MockStateProvider;
    evaluator = new ConditionEvaluator(stateProvider);
  });

  it("should satisfy when current time >= executeAt", async () => {
    const condition: ScheduleCondition = {
      time: { type: "absolute", executeAt: 1000 },
    };
    const result = await evaluator.evaluate(condition, 1000);
    expect(result.satisfied).toBe(true);
    expect(result.proof).toBeDefined();
    expect(result.proof!.type).toBe("time");
  });

  it("should satisfy when current time is past executeAt", async () => {
    const condition: ScheduleCondition = {
      time: { type: "absolute", executeAt: 1000 },
    };
    const result = await evaluator.evaluate(condition, 2000);
    expect(result.satisfied).toBe(true);
  });

  it("should not satisfy when current time < executeAt", async () => {
    const condition: ScheduleCondition = {
      time: { type: "absolute", executeAt: 5000 },
    };
    const result = await evaluator.evaluate(condition, 1000);
    expect(result.satisfied).toBe(false);
    expect(result.nextCheckTime).toBe(5000);
    expect(result.reason).toBeDefined();
  });
});

// ============ Time Condition Tests: Relative ============

describe("ConditionEvaluator - Relative Time", () => {
  let stateProvider: MockStateProvider;
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    stateProvider = createMockStateProvider() as MockStateProvider;
    evaluator = new ConditionEvaluator(stateProvider);
  });

  it("should satisfy when current time >= pre-computed executeAt", async () => {
    // Relative conditions must have executeAt pre-computed
    const condition: ScheduleCondition = {
      time: {
        type: "relative",
        executeAfter: 5000,
        executeAt: 10_000, // pre-computed: creation time + executeAfter
      },
    };
    const result = await evaluator.evaluate(condition, 10_000);
    expect(result.satisfied).toBe(true);
  });

  it("should not satisfy when current time < pre-computed executeAt", async () => {
    const condition: ScheduleCondition = {
      time: {
        type: "relative",
        executeAfter: 5000,
        executeAt: 15_000,
      },
    };
    const result = await evaluator.evaluate(condition, 10_000);
    expect(result.satisfied).toBe(false);
    expect(result.nextCheckTime).toBe(15_000);
  });

  it("should throw if executeAt is not pre-computed on relative condition", async () => {
    const condition: ScheduleCondition = {
      time: {
        type: "relative",
        executeAfter: 5000,
        // executeAt intentionally missing
      },
    };
    await expect(evaluator.evaluate(condition, 10_000)).rejects.toThrow(
      ScheduleError,
    );
  });
});

// ============ Time Condition Tests: Periodic ============

describe("ConditionEvaluator - Periodic Time", () => {
  let stateProvider: MockStateProvider;
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    stateProvider = createMockStateProvider() as MockStateProvider;
    evaluator = new ConditionEvaluator(stateProvider);
  });

  it("should satisfy at the first interval boundary", async () => {
    const condition: ScheduleCondition = {
      time: {
        type: "periodic",
        interval: 10_000,
        startTime: 0,
      },
    };
    // At exactly 0 (start of first interval), should satisfy
    const result = await evaluator.evaluate(condition, 0);
    expect(result.satisfied).toBe(true);
    expect(result.nextCheckTime).toBe(10_000);
  });

  it("should satisfy at second interval when not already executed", async () => {
    const condition: ScheduleCondition = {
      time: {
        type: "periodic",
        interval: 10_000,
        startTime: 0,
      },
    };
    // At 15000, interval index = 1. Not yet executed in interval 1.
    const result = await evaluator.evaluate(condition, 15_000);
    expect(result.satisfied).toBe(true);
  });

  it("should not satisfy if already executed in current interval (MED-001)", async () => {
    const condition: ScheduleCondition = {
      time: {
        type: "periodic",
        interval: 10_000,
        startTime: 0,
      },
    };
    // At 15000, interval index = 1. Last executed at 12000, which is also interval 1.
    const result = await evaluator.evaluate(condition, 15_000, 12_000);
    expect(result.satisfied).toBe(false);
    expect(result.nextCheckTime).toBe(20_000);
  });

  it("should satisfy in new interval even after execution in previous interval", async () => {
    const condition: ScheduleCondition = {
      time: {
        type: "periodic",
        interval: 10_000,
        startTime: 0,
      },
    };
    // At 20000, interval index = 2. Last executed at 12000, which is interval 1.
    const result = await evaluator.evaluate(condition, 20_000, 12_000);
    expect(result.satisfied).toBe(true);
  });

  it("should not satisfy when current time is before start time", async () => {
    const condition: ScheduleCondition = {
      time: {
        type: "periodic",
        interval: 10_000,
        startTime: 5000,
      },
    };
    const result = await evaluator.evaluate(condition, 3000);
    expect(result.satisfied).toBe(false);
    expect(result.nextCheckTime).toBe(5000);
  });

  it("should not satisfy when current time is past end time", async () => {
    const condition: ScheduleCondition = {
      time: {
        type: "periodic",
        interval: 10_000,
        startTime: 0,
        endTime: 20_000,
      },
    };
    const result = await evaluator.evaluate(condition, 25_000);
    expect(result.satisfied).toBe(false);
    expect(result.reason).toContain("ended");
  });

  it("should throw when interval is zero or negative (MED-002)", async () => {
    const zeroInterval: ScheduleCondition = {
      time: {
        type: "periodic",
        interval: 0,
        startTime: 0,
      },
    };
    await expect(evaluator.evaluate(zeroInterval, 1000)).rejects.toThrow(
      ScheduleError,
    );

    const negativeInterval: ScheduleCondition = {
      time: {
        type: "periodic",
        interval: -1000,
        startTime: 0,
      },
    };
    await expect(evaluator.evaluate(negativeInterval, 1000)).rejects.toThrow(
      ScheduleError,
    );
  });
});

// ============ Logic Condition Tests: State Compare ============

describe("ConditionEvaluator - State Compare", () => {
  let stateProvider: MockStateProvider;
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    stateProvider = createMockStateProvider() as MockStateProvider;
    evaluator = new ConditionEvaluator(stateProvider);
  });

  it("should satisfy when state value equals expected (bigint)", async () => {
    stateProvider._setState("balance:alice", 1000n);
    const condition: ScheduleCondition = {
      logic: {
        type: "state_compare",
        stateKey: "balance:alice",
        comparator: "eq",
        compareValue: 1000n,
      },
    };
    const result = await evaluator.evaluate(condition, 0);
    expect(result.satisfied).toBe(true);
    expect(result.proof).toBeDefined();
    expect(result.proof!.type).toBe("logic");
  });

  it("should satisfy 'gt' comparator", async () => {
    stateProvider._setState("balance:alice", 1500n);
    const condition: ScheduleCondition = {
      logic: {
        type: "state_compare",
        stateKey: "balance:alice",
        comparator: "gt",
        compareValue: 1000n,
      },
    };
    const result = await evaluator.evaluate(condition, 0);
    expect(result.satisfied).toBe(true);
  });

  it("should not satisfy 'gt' when equal", async () => {
    stateProvider._setState("balance:alice", 1000n);
    const condition: ScheduleCondition = {
      logic: {
        type: "state_compare",
        stateKey: "balance:alice",
        comparator: "gt",
        compareValue: 1000n,
      },
    };
    const result = await evaluator.evaluate(condition, 0);
    expect(result.satisfied).toBe(false);
  });

  it("should satisfy 'gte' when equal", async () => {
    stateProvider._setState("balance:alice", 1000n);
    const condition: ScheduleCondition = {
      logic: {
        type: "state_compare",
        stateKey: "balance:alice",
        comparator: "gte",
        compareValue: 1000n,
      },
    };
    const result = await evaluator.evaluate(condition, 0);
    expect(result.satisfied).toBe(true);
  });

  it("should satisfy string 'contains' comparator", async () => {
    stateProvider._setState("status:contract", "active_and_verified");
    const condition: ScheduleCondition = {
      logic: {
        type: "state_compare",
        stateKey: "status:contract",
        comparator: "contains",
        compareValue: "verified",
      },
    };
    const result = await evaluator.evaluate(condition, 0);
    expect(result.satisfied).toBe(true);
  });

  it("should throw when stateKey or comparator is missing", async () => {
    const noKey: ScheduleCondition = {
      logic: {
        type: "state_compare",
        comparator: "eq",
        compareValue: 100,
      },
    };
    await expect(evaluator.evaluate(noKey, 0)).rejects.toThrow(ScheduleError);

    const noComparator: ScheduleCondition = {
      logic: {
        type: "state_compare",
        stateKey: "key",
        compareValue: 100,
      },
    };
    await expect(evaluator.evaluate(noComparator, 0)).rejects.toThrow(
      ScheduleError,
    );
  });
});

// ============ Logic Condition Tests: Event Occurred ============

describe("ConditionEvaluator - Event Occurred", () => {
  let stateProvider: MockStateProvider;
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    stateProvider = createMockStateProvider() as MockStateProvider;
    evaluator = new ConditionEvaluator(stateProvider);
  });

  it("should satisfy when event has occurred", async () => {
    stateProvider._addEvent("claim_verified");
    const condition: ScheduleCondition = {
      logic: {
        type: "event_occurred",
        eventType: "claim_verified",
      },
    };
    const result = await evaluator.evaluate(condition, 0);
    expect(result.satisfied).toBe(true);
  });

  it("should not satisfy when event has not occurred", async () => {
    const condition: ScheduleCondition = {
      logic: {
        type: "event_occurred",
        eventType: "claim_rejected",
      },
    };
    const result = await evaluator.evaluate(condition, 0);
    expect(result.satisfied).toBe(false);
    expect(result.reason).toContain("has not occurred");
  });

  it("should throw when eventType is missing", async () => {
    const condition: ScheduleCondition = {
      logic: {
        type: "event_occurred",
        // eventType missing
      },
    };
    await expect(evaluator.evaluate(condition, 0)).rejects.toThrow(
      ScheduleError,
    );
  });
});

// ============ Logic Condition Tests: Threshold Crossed ============

describe("ConditionEvaluator - Threshold Crossed", () => {
  it("should satisfy 'above' when value exceeds threshold", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    sp._setState("price:lgm", 150n);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "above",
      },
    };
    const result = await ev.evaluate(condition, 0);
    expect(result.satisfied).toBe(true);
  });

  it("should not satisfy 'above' when value is at threshold", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    sp._setState("price:lgm", 100n);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "above",
      },
    };
    const result = await ev.evaluate(condition, 0);
    expect(result.satisfied).toBe(false);
  });

  it("should satisfy 'below' when value is under threshold", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    sp._setState("price:lgm", 50n);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "below",
      },
    };
    const result = await ev.evaluate(condition, 0);
    expect(result.satisfied).toBe(true);
  });

  it("should not trigger cross_up on first observation (HIGH-003)", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    sp._setState("price:lgm", 150n);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "cross_up",
      },
    };
    // First check - just records state, does not trigger
    const result = await ev.evaluate(condition, 0);
    expect(result.satisfied).toBe(false);
    expect(result.reason).toContain("First observation");
  });

  it("should detect cross_up when value crosses above threshold", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "cross_up",
      },
    };

    // First observation: below threshold
    sp._setState("price:lgm", 80n);
    await ev.evaluate(condition, 0);

    // Second observation: above threshold -> crossing detected
    sp._setState("price:lgm", 120n);
    const result = await ev.evaluate(condition, 1000);
    expect(result.satisfied).toBe(true);
  });

  it("should not detect cross_up when staying above threshold", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "cross_up",
      },
    };

    // First: above threshold
    sp._setState("price:lgm", 110n);
    await ev.evaluate(condition, 0);

    // Second: still above -> no crossing
    sp._setState("price:lgm", 120n);
    const result = await ev.evaluate(condition, 1000);
    expect(result.satisfied).toBe(false);
  });

  it("should detect cross_down when value drops below threshold", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "cross_down",
      },
    };

    // First: above threshold
    sp._setState("price:lgm", 120n);
    await ev.evaluate(condition, 0);

    // Second: below threshold -> crossing detected
    sp._setState("price:lgm", 80n);
    const result = await ev.evaluate(condition, 1000);
    expect(result.satisfied).toBe(true);
  });

  it("should isolate threshold state per process (CRIT-004)", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "cross_up",
      },
    };

    // Process A: set initial state below threshold
    sp._setState("price:lgm", 80n);
    await ev.evaluate(condition, 0, undefined, "processA");

    // Process B: set initial state above threshold
    sp._setState("price:lgm", 110n);
    await ev.evaluate(condition, 0, undefined, "processB");

    // Now value moves above threshold
    sp._setState("price:lgm", 120n);

    // Process A should detect cross_up (80 -> 120 crosses 100)
    const resultA = await ev.evaluate(condition, 1000, undefined, "processA");
    expect(resultA.satisfied).toBe(true);

    // Process B should NOT detect cross_up (110 -> 120, both above 100)
    const resultB = await ev.evaluate(condition, 1000, undefined, "processB");
    expect(resultB.satisfied).toBe(false);
  });

  it("should throw when threshold params are missing", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    const missing: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        // all params missing
      },
    };
    await expect(ev.evaluate(missing, 0)).rejects.toThrow(ScheduleError);
  });

  it("should handle non-numeric state value gracefully", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    sp._setState("price:lgm", "not-a-number");
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "above",
      },
    };
    const result = await ev.evaluate(condition, 0);
    expect(result.satisfied).toBe(false);
    expect(result.reason).toContain("not numeric");
  });
});

// ============ Compound Condition Tests ============

describe("ConditionEvaluator - Compound Conditions", () => {
  it("should satisfy AND when all sub-conditions are met", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    sp._setState("balance:alice", 500n);
    const condition: ScheduleCondition = {
      compound: {
        operator: "and",
        conditions: [
          { time: { type: "absolute", executeAt: 1000 } },
          {
            logic: {
              type: "state_compare",
              stateKey: "balance:alice",
              comparator: "gte",
              compareValue: 100n,
            },
          },
        ],
      },
    };
    const result = await ev.evaluate(condition, 2000);
    expect(result.satisfied).toBe(true);
    expect(result.proof!.type).toBe("compound");
  });

  it("should not satisfy AND when one sub-condition fails", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    sp._setState("balance:alice", 50n);
    const condition: ScheduleCondition = {
      compound: {
        operator: "and",
        conditions: [
          { time: { type: "absolute", executeAt: 1000 } },
          {
            logic: {
              type: "state_compare",
              stateKey: "balance:alice",
              comparator: "gte",
              compareValue: 100n,
            },
          },
        ],
      },
    };
    const result = await ev.evaluate(condition, 2000);
    expect(result.satisfied).toBe(false);
  });

  it("should satisfy OR when at least one sub-condition is met", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    // Time not yet, but logic is met
    sp._setState("balance:alice", 500n);
    const condition: ScheduleCondition = {
      compound: {
        operator: "or",
        conditions: [
          { time: { type: "absolute", executeAt: 99999 } },
          {
            logic: {
              type: "state_compare",
              stateKey: "balance:alice",
              comparator: "gte",
              compareValue: 100n,
            },
          },
        ],
      },
    };
    const result = await ev.evaluate(condition, 1000);
    expect(result.satisfied).toBe(true);
  });

  it("should not satisfy OR when all sub-conditions fail", async () => {
    const sp = createMockStateProvider() as MockStateProvider;
    const ev = new ConditionEvaluator(sp);
    sp._setState("balance:alice", 10n);
    const condition: ScheduleCondition = {
      compound: {
        operator: "or",
        conditions: [
          { time: { type: "absolute", executeAt: 99999 } },
          {
            logic: {
              type: "state_compare",
              stateKey: "balance:alice",
              comparator: "gte",
              compareValue: 100n,
            },
          },
        ],
      },
    };
    const result = await ev.evaluate(condition, 1000);
    expect(result.satisfied).toBe(false);
  });

  it("should short-circuit AND on first false", async () => {
    // First condition fails, second should not be evaluated
    let logicChecked = false;
    const customProvider = createMockStateProvider({
      getState: async () => {
        logicChecked = true;
        return 500n;
      },
    }) as MockStateProvider;
    const customEvaluator = new ConditionEvaluator(customProvider);

    const condition: ScheduleCondition = {
      compound: {
        operator: "and",
        conditions: [
          { time: { type: "absolute", executeAt: 99999 } }, // fails
          {
            logic: {
              type: "state_compare",
              stateKey: "balance",
              comparator: "eq",
              compareValue: 500n,
            },
          },
        ],
      },
    };
    const result = await customEvaluator.evaluate(condition, 1000);
    expect(result.satisfied).toBe(false);
    expect(logicChecked).toBe(false); // Logic was never checked
  });
});

// ============ Implicit AND (time + logic) Tests ============

describe("ConditionEvaluator - Implicit AND (time + logic)", () => {
  let stateProvider: MockStateProvider;
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    stateProvider = createMockStateProvider() as MockStateProvider;
    evaluator = new ConditionEvaluator(stateProvider);
  });

  it("should satisfy when both time and logic are met", async () => {
    stateProvider._setState("key", 42);
    const condition: ScheduleCondition = {
      time: { type: "absolute", executeAt: 1000 },
      logic: {
        type: "state_compare",
        stateKey: "key",
        comparator: "eq",
        compareValue: 42,
      },
    };
    const result = await evaluator.evaluate(condition, 2000);
    expect(result.satisfied).toBe(true);
    expect(result.proof!.type).toBe("compound");
  });

  it("should not satisfy when time passes but logic fails", async () => {
    stateProvider._setState("key", 0);
    const condition: ScheduleCondition = {
      time: { type: "absolute", executeAt: 1000 },
      logic: {
        type: "state_compare",
        stateKey: "key",
        comparator: "eq",
        compareValue: 42,
      },
    };
    const result = await evaluator.evaluate(condition, 2000);
    expect(result.satisfied).toBe(false);
  });

  it("should not satisfy when logic passes but time has not arrived", async () => {
    stateProvider._setState("key", 42);
    const condition: ScheduleCondition = {
      time: { type: "absolute", executeAt: 9999 },
      logic: {
        type: "state_compare",
        stateKey: "key",
        comparator: "eq",
        compareValue: 42,
      },
    };
    const result = await evaluator.evaluate(condition, 1000);
    expect(result.satisfied).toBe(false);
  });
});

// ============ Empty Condition Tests ============

describe("ConditionEvaluator - Empty Condition", () => {
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    const stateProvider = createMockStateProvider() as MockStateProvider;
    evaluator = new ConditionEvaluator(stateProvider);
  });

  it("should not satisfy an empty condition (prevents unconditional execution)", async () => {
    const condition: ScheduleCondition = {};
    const result = await evaluator.evaluate(condition, 1000);
    expect(result.satisfied).toBe(false);
    expect(result.reason).toContain("Empty condition");
  });
});

// ============ Utility Method Tests ============

describe("ConditionEvaluator - Utility Methods", () => {
  let stateProvider: MockStateProvider;
  let evaluator: ConditionEvaluator;

  beforeEach(() => {
    stateProvider = createMockStateProvider() as MockStateProvider;
    evaluator = new ConditionEvaluator(stateProvider);
  });

  it("should extract watched keys from logic condition with stateKey", () => {
    const condition: ScheduleCondition = {
      logic: {
        type: "state_compare",
        stateKey: "balance:alice",
        comparator: "gt",
        compareValue: 0n,
      },
    };
    const keys = evaluator.extractWatchedKeys(condition);
    expect(keys).toContain("balance:alice");
  });

  it("should extract watched keys from logic condition with thresholdKey", () => {
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "above",
      },
    };
    const keys = evaluator.extractWatchedKeys(condition);
    expect(keys).toContain("price:lgm");
  });

  it("should extract watched keys from compound conditions (deduplicated)", () => {
    const condition: ScheduleCondition = {
      compound: {
        operator: "and",
        conditions: [
          {
            logic: {
              type: "state_compare",
              stateKey: "key-a",
              comparator: "eq",
              compareValue: 1,
            },
          },
          {
            logic: {
              type: "threshold_crossed",
              thresholdKey: "key-b",
              thresholdValue: 10n,
              direction: "above",
            },
          },
          {
            logic: {
              type: "state_compare",
              stateKey: "key-a", // duplicate
              comparator: "ne",
              compareValue: 0,
            },
          },
        ],
      },
    };
    const keys = evaluator.extractWatchedKeys(condition);
    expect(keys).toContain("key-a");
    expect(keys).toContain("key-b");
    expect(keys).toHaveLength(2); // deduplicated
  });

  it("should return empty array for time-only condition", () => {
    const condition: ScheduleCondition = {
      time: { type: "absolute", executeAt: 1000 },
    };
    const keys = evaluator.extractWatchedKeys(condition);
    expect(keys).toHaveLength(0);
  });

  it("should calculate next check time for absolute time process", () => {
    const process = {
      condition: { time: { type: "absolute" as const, executeAt: 5000 } },
      createdAt: 0,
    } as ScheduleProcess;
    const next = evaluator.calculateNextCheckTime(process, 1000);
    expect(next).toBe(5000);
  });

  it("should calculate next check time for periodic process", () => {
    const process = {
      condition: {
        time: { type: "periodic" as const, interval: 10_000, startTime: 0 },
      },
      createdAt: 0,
    } as ScheduleProcess;
    // At 5000, next interval = ceil(5000/10000) * 10000 = 10000
    const next = evaluator.calculateNextCheckTime(process, 5000);
    expect(next).toBe(10_000);
  });

  it("should return undefined for logic-only process", () => {
    const process = {
      condition: {
        logic: {
          type: "state_compare" as const,
          stateKey: "key",
          comparator: "eq" as const,
          compareValue: 1,
        },
      },
      createdAt: 0,
    } as ScheduleProcess;
    const next = evaluator.calculateNextCheckTime(process, 1000);
    expect(next).toBeUndefined();
  });

  it("should clear specific threshold state", async () => {
    stateProvider._setState("price:lgm", 80n);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "cross_up",
      },
    };
    await evaluator.evaluate(condition, 0, undefined, "proc-1");

    expect(evaluator.getThresholdState("proc-1:price:lgm")).toBeDefined();
    evaluator.clearThresholdState("proc-1:price:lgm");
    expect(evaluator.getThresholdState("proc-1:price:lgm")).toBeUndefined();
  });

  it("should clear all threshold states", async () => {
    stateProvider._setState("price:lgm", 80n);
    const condition: ScheduleCondition = {
      logic: {
        type: "threshold_crossed",
        thresholdKey: "price:lgm",
        thresholdValue: 100n,
        direction: "above",
      },
    };
    await evaluator.evaluate(condition, 0);
    evaluator.clearAllThresholdStates();
    expect(evaluator.getThresholdState("price:lgm")).toBeUndefined();
  });
});
