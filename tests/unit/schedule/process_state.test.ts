/**
 * Schedule Process State (SPS 131313) Unit Tests
 *
 * Tests for the ScheduleProcessStateManager, which manages all active schedule
 * processes, their indexes, lifecycle transitions, and metrics.
 *
 * NOTE: Each test creates its own manager instance to avoid state leakage
 * under vitest's concurrent test execution mode.
 */

import { describe, it, expect } from "vitest";
import { ScheduleProcessStateManager } from "../../../core/schedule/process_state.js";
import {
  ScheduleError,
  type ScheduleProcess,
  type ScheduleCondition,
  type ScheduleEvent,
} from "../../../core/schedule/index.js";

// ============ Test Helpers ============

const TEST_DID = "did:lgm:test-creator-0123456789abcdef";
const TEST_DID_2 = "did:lgm:test-creator-fedcba9876543210";
const TEST_TARGET = "did:lgm:target-contract-aabbccdd";

let idCounter = 0;

function createTestProcess(
  overrides: Partial<ScheduleProcess> = {},
): ScheduleProcess {
  idCounter++;
  const id =
    overrides.id ??
    `proc-${idCounter}-${Math.random().toString(36).slice(2, 8)}`;
  return {
    id,
    sourceMessageId: `msg-${id}`,
    creator: TEST_DID,
    condition: {
      time: { type: "absolute", executeAt: Date.now() + 60_000 },
    },
    nextCheckTime: Date.now() + 60_000,
    watchedStateKeys: [],
    status: "pending",
    executionCount: 0,
    remainingFee: 100_000n,
    feePerExecution: 1_000n,
    targetAddress: TEST_TARGET,
    callData: new Uint8Array([0x01, 0x02]),
    transferValue: 0n,
    generatedTxNonce: 0n,
    createdAt: Date.now(),
    modifiedAt: Date.now(),
    ...overrides,
  };
}

// ============ Initialization Tests ============

describe("ScheduleProcessStateManager - Initialization", () => {
  it("should initialize with default config", () => {
    const manager = new ScheduleProcessStateManager();
    expect(manager).toBeDefined();
    expect(manager.getStateRoot()).toBeDefined();
    expect(manager.getStateRoot().length).toBeGreaterThan(0);
  });

  it("should initialize with custom config overrides", () => {
    const manager = new ScheduleProcessStateManager({
      maxActiveSchedulesPerAccount: 50,
    });
    expect(manager).toBeDefined();
  });

  it("should start with empty metrics", () => {
    const manager = new ScheduleProcessStateManager();
    const metrics = manager.getMetrics();
    expect(metrics.totalProcesses).toBe(0);
    expect(metrics.activeProcesses).toBe(0);
    expect(metrics.completedProcesses).toBe(0);
    expect(metrics.totalExecutions).toBe(0);
  });

  it("should compute a consistent empty state root", () => {
    const manager1 = new ScheduleProcessStateManager();
    const manager2 = new ScheduleProcessStateManager();
    expect(manager1.getStateRoot()).toBe(manager2.getStateRoot());
  });
});

// ============ Process Addition Tests ============

describe("ScheduleProcessStateManager - addProcess", () => {
  it("should add a valid process and retrieve it by ID", () => {
    const manager = new ScheduleProcessStateManager();
    const process = createTestProcess({ id: "add-proc-1" });
    manager.addProcess(process);
    const retrieved = manager.getProcess("add-proc-1");
    expect(retrieved).toBeDefined();
    expect(retrieved!.id).toBe("add-proc-1");
    expect(retrieved!.creator).toBe(TEST_DID);
  });

  it("should update metrics when adding a pending process", () => {
    const manager = new ScheduleProcessStateManager();
    manager.addProcess(createTestProcess({ status: "pending" }));
    const metrics = manager.getMetrics();
    expect(metrics.totalProcesses).toBe(1);
    expect(metrics.activeProcesses).toBe(1); // pending counts as active
  });

  it("should update metrics when adding an active process", () => {
    const manager = new ScheduleProcessStateManager();
    manager.addProcess(createTestProcess({ status: "active" }));
    const metrics = manager.getMetrics();
    expect(metrics.activeProcesses).toBe(1);
  });

  it("should index process by account", () => {
    const manager = new ScheduleProcessStateManager();
    manager.addProcess(
      createTestProcess({ id: "idx-acct-1", creator: TEST_DID }),
    );
    const byAccount = manager.getProcessesByAccount(TEST_DID);
    expect(byAccount).toHaveLength(1);
    expect(byAccount[0].id).toBe("idx-acct-1");
  });

  it("should index process by status", () => {
    const manager = new ScheduleProcessStateManager();
    manager.addProcess(
      createTestProcess({ id: "idx-status-1", status: "active" }),
    );
    const byStatus = manager.getProcessesByStatus("active");
    expect(byStatus).toHaveLength(1);
  });

  it("should index process by linked claim", () => {
    const manager = new ScheduleProcessStateManager();
    manager.addProcess(
      createTestProcess({
        id: "idx-claim-1",
        linkedEntityType: "claim",
        linkedEntityId: "claim-42",
      }),
    );
    const byClaim = manager.getProcessesByClaimId("claim-42");
    expect(byClaim).toHaveLength(1);
    expect(byClaim[0].id).toBe("idx-claim-1");
  });

  it("should index process by linked dispute", () => {
    const manager = new ScheduleProcessStateManager();
    manager.addProcess(
      createTestProcess({
        id: "idx-dispute-1",
        linkedEntityType: "dispute",
        linkedEntityId: "dispute-7",
      }),
    );
    const byDispute = manager.getProcessesByDisputeId("dispute-7");
    expect(byDispute).toHaveLength(1);
  });

  it("should update state root when adding a process", () => {
    const manager = new ScheduleProcessStateManager();
    const rootBefore = manager.getStateRoot();
    manager.addProcess(createTestProcess());
    const rootAfter = manager.getStateRoot();
    expect(rootAfter).not.toBe(rootBefore);
  });

  it("should throw MAX_SCHEDULES_EXCEEDED when account exceeds limit", () => {
    const manager = new ScheduleProcessStateManager({
      maxActiveSchedulesPerAccount: 2,
    });

    manager.addProcess(createTestProcess({ id: "lim-1", creator: TEST_DID }));
    manager.addProcess(createTestProcess({ id: "lim-2", creator: TEST_DID }));

    expect(() =>
      manager.addProcess(createTestProcess({ id: "lim-3", creator: TEST_DID })),
    ).toThrow(ScheduleError);

    try {
      manager.addProcess(createTestProcess({ id: "lim-4", creator: TEST_DID }));
    } catch (e) {
      expect((e as ScheduleError).code).toBe("MAX_SCHEDULES_EXCEEDED");
    }
  });

  it("should throw when process ID is missing", () => {
    const manager = new ScheduleProcessStateManager();
    expect(() => manager.addProcess(createTestProcess({ id: "" }))).toThrow(
      ScheduleError,
    );
  });

  it("should throw when creator is missing", () => {
    const manager = new ScheduleProcessStateManager();
    expect(() =>
      manager.addProcess(createTestProcess({ creator: "" })),
    ).toThrow(ScheduleError);
  });

  it("should throw when remaining fee is negative", () => {
    const manager = new ScheduleProcessStateManager();
    expect(() =>
      manager.addProcess(createTestProcess({ remainingFee: -1n })),
    ).toThrow(ScheduleError);
  });

  it("should throw CONDITION_TOO_COMPLEX when compound condition exceeds max depth", () => {
    const manager = new ScheduleProcessStateManager();
    // Depth 4 nested compound (default max is 3)
    const deepCondition: ScheduleCondition = {
      compound: {
        operator: "and",
        conditions: [
          {
            compound: {
              operator: "or",
              conditions: [
                {
                  compound: {
                    operator: "and",
                    conditions: [
                      {
                        compound: {
                          operator: "or",
                          conditions: [
                            {
                              time: { type: "absolute", executeAt: 1000 },
                            },
                          ],
                        },
                      },
                    ],
                  },
                },
              ],
            },
          },
        ],
      },
    };

    expect(() =>
      manager.addProcess(createTestProcess({ condition: deepCondition })),
    ).toThrow(ScheduleError);
  });

  it("should throw CONDITION_TOO_COMPLEX when too many watched keys", () => {
    const manager = new ScheduleProcessStateManager();
    const watchedStateKeys = Array.from({ length: 11 }, (_, i) => `key-${i}`);
    expect(() =>
      manager.addProcess(createTestProcess({ watchedStateKeys })),
    ).toThrow(ScheduleError);
  });

  it("should emit process_created event when adding a process", () => {
    const manager = new ScheduleProcessStateManager();
    const events: ScheduleEvent[] = [];
    manager.onEvent((e) => events.push(e));

    manager.addProcess(createTestProcess({ id: "proc-ev" }));

    expect(events).toHaveLength(1);
    expect(events[0].type).toBe("process_created");
    if (events[0].type === "process_created") {
      expect(events[0].process.id).toBe("proc-ev");
    }
  });
});

// ============ Process Update Tests ============

describe("ScheduleProcessStateManager - updateProcess", () => {
  it("should update process status", () => {
    const manager = new ScheduleProcessStateManager();
    manager.setBlockTimestamp(Date.now());
    manager.addProcess(createTestProcess({ id: "upd-1", status: "pending" }));
    manager.updateProcess("upd-1", { status: "active" });
    const process = manager.getProcess("upd-1");
    expect(process!.status).toBe("active");
  });

  it("should throw PROCESS_NOT_FOUND for non-existent process", () => {
    const manager = new ScheduleProcessStateManager();
    expect(() =>
      manager.updateProcess("nonexistent", { status: "active" }),
    ).toThrow(ScheduleError);

    try {
      manager.updateProcess("nonexistent", { status: "active" });
    } catch (e) {
      expect((e as ScheduleError).code).toBe("PROCESS_NOT_FOUND");
    }
  });

  it("should update status index on status transition", () => {
    const manager = new ScheduleProcessStateManager();
    manager.setBlockTimestamp(Date.now());
    manager.addProcess(createTestProcess({ id: "trans-1", status: "pending" }));
    expect(manager.getProcessesByStatus("pending")).toHaveLength(1);
    expect(manager.getProcessesByStatus("active")).toHaveLength(0);

    manager.updateProcess("trans-1", { status: "active" });

    expect(manager.getProcessesByStatus("pending")).toHaveLength(0);
    expect(manager.getProcessesByStatus("active")).toHaveLength(1);
  });

  it("should decrement activeProcesses when moving from active to completed", () => {
    const manager = new ScheduleProcessStateManager();
    manager.setBlockTimestamp(Date.now());
    manager.addProcess(createTestProcess({ id: "dec-1", status: "active" }));
    expect(manager.getMetrics().activeProcesses).toBe(1);

    manager.updateProcess("dec-1", { status: "completed" });
    expect(manager.getMetrics().activeProcesses).toBe(0);
  });

  it("should increment activeProcesses when moving from paused to active", () => {
    const manager = new ScheduleProcessStateManager();
    manager.setBlockTimestamp(Date.now());
    manager.addProcess(createTestProcess({ id: "inc-1", status: "paused" }));
    // paused does not count as active
    expect(manager.getMetrics().activeProcesses).toBe(0);

    manager.updateProcess("inc-1", { status: "active" });
    expect(manager.getMetrics().activeProcesses).toBe(1);
  });

  it("should update modifiedAt using deterministic block timestamp", () => {
    const manager = new ScheduleProcessStateManager();
    const blockTime = 1700000000000;
    manager.setBlockTimestamp(blockTime);

    manager.addProcess(createTestProcess({ id: "ts-1", modifiedAt: 1000 }));
    manager.updateProcess("ts-1", { executionCount: 5 });

    const process = manager.getProcess("ts-1");
    expect(process!.modifiedAt).toBe(blockTime);
  });

  it("should update state root after process update", () => {
    const manager = new ScheduleProcessStateManager();
    manager.setBlockTimestamp(Date.now());
    manager.addProcess(createTestProcess({ id: "root-1" }));
    const rootBefore = manager.getStateRoot();

    manager.updateProcess("root-1", { executionCount: 1 });
    const rootAfter = manager.getStateRoot();

    expect(rootAfter).not.toBe(rootBefore);
  });

  it("should update watchedKeys index when watchedStateKeys change", () => {
    const manager = new ScheduleProcessStateManager();
    manager.setBlockTimestamp(Date.now());
    manager.addProcess(
      createTestProcess({
        id: "watch-1",
        watchedStateKeys: ["key-a"],
      }),
    );
    expect(manager.getProcessesWatchingKey("key-a")).toHaveLength(1);

    manager.updateProcess("watch-1", {
      watchedStateKeys: ["key-b", "key-c"],
    });
    // Old key should no longer be watched
    expect(manager.getProcessesWatchingKey("key-a")).toHaveLength(0);
    // New keys should be watched
    expect(manager.getProcessesWatchingKey("key-b")).toHaveLength(1);
    expect(manager.getProcessesWatchingKey("key-c")).toHaveLength(1);
  });
});

// ============ Process Removal Tests ============

describe("ScheduleProcessStateManager - removeProcess", () => {
  it("should remove a process and make it unretrievable", () => {
    const manager = new ScheduleProcessStateManager();
    manager.addProcess(createTestProcess({ id: "rm-1" }));
    manager.removeProcess("rm-1");
    expect(manager.getProcess("rm-1")).toBeUndefined();
  });

  it("should not throw when removing non-existent process", () => {
    const manager = new ScheduleProcessStateManager();
    expect(() => manager.removeProcess("nonexistent")).not.toThrow();
  });

  it("should remove process from all indexes", () => {
    const manager = new ScheduleProcessStateManager();
    manager.addProcess(
      createTestProcess({
        id: "rm-idx-1",
        creator: TEST_DID,
        status: "active",
        watchedStateKeys: ["watched-key"],
      }),
    );

    manager.removeProcess("rm-idx-1");

    expect(manager.getProcessesByAccount(TEST_DID)).toHaveLength(0);
    expect(manager.getProcessesByStatus("active")).toHaveLength(0);
    expect(manager.getProcessesWatchingKey("watched-key")).toHaveLength(0);
  });

  it("should increment completedProcesses when removing a completed process", () => {
    const manager = new ScheduleProcessStateManager();
    manager.addProcess(
      createTestProcess({ id: "rm-comp-1", status: "completed" }),
    );
    const completedBefore = manager.getMetrics().completedProcesses;

    manager.removeProcess("rm-comp-1");
    expect(manager.getMetrics().completedProcesses).toBe(completedBefore + 1);
  });

  it("should update state root back to empty root when last process removed", () => {
    const manager = new ScheduleProcessStateManager();
    const emptyRoot = manager.getStateRoot();
    manager.addProcess(createTestProcess({ id: "rm-root-1" }));
    manager.removeProcess("rm-root-1");
    expect(manager.getStateRoot()).toBe(emptyRoot);
  });
});

// ============ Query Method Tests ============

describe("ScheduleProcessStateManager - Query Methods", () => {
  it("should return processes due by time", () => {
    const mgr = new ScheduleProcessStateManager();
    const now = Date.now();
    mgr.addProcess(
      createTestProcess({
        id: "due-past",
        status: "active",
        nextCheckTime: now - 5000,
        condition: { time: { type: "absolute", executeAt: now - 5000 } },
      }),
    );
    mgr.addProcess(
      createTestProcess({
        id: "due-future",
        status: "active",
        nextCheckTime: now + 60_000,
        condition: { time: { type: "absolute", executeAt: now + 60_000 } },
      }),
    );

    const due = mgr.getProcessesDueByTime(now);
    expect(due).toHaveLength(1);
    expect(due[0].id).toBe("due-past");
  });

  it("should only return active or pending processes as due by time", () => {
    const mgr = new ScheduleProcessStateManager();
    const now = Date.now();
    mgr.addProcess(
      createTestProcess({
        id: "due-paused",
        status: "paused",
        nextCheckTime: now - 5000,
        condition: { time: { type: "absolute", executeAt: now - 5000 } },
      }),
    );
    mgr.addProcess(
      createTestProcess({
        id: "due-pending",
        status: "pending",
        nextCheckTime: now - 1000,
        condition: { time: { type: "absolute", executeAt: now - 1000 } },
      }),
    );

    const due = mgr.getProcessesDueByTime(now);
    expect(due).toHaveLength(1);
    expect(due[0].id).toBe("due-pending");
  });

  it("should return processes watching a key (only active/pending)", () => {
    const mgr = new ScheduleProcessStateManager();
    mgr.addProcess(
      createTestProcess({
        id: "wk-active",
        status: "active",
        watchedStateKeys: ["balance:alice"],
      }),
    );
    mgr.addProcess(
      createTestProcess({
        id: "wk-paused",
        status: "paused",
        watchedStateKeys: ["balance:alice"],
      }),
    );

    const watchers = mgr.getProcessesWatchingKey("balance:alice");
    expect(watchers).toHaveLength(1);
    expect(watchers[0].id).toBe("wk-active");
  });

  it("should return empty array for unwatched key", () => {
    const mgr = new ScheduleProcessStateManager();
    expect(mgr.getProcessesWatchingKey("unknown-key")).toHaveLength(0);
  });

  it("should return processes by account", () => {
    const mgr = new ScheduleProcessStateManager();
    mgr.addProcess(createTestProcess({ id: "ba-1", creator: TEST_DID }));
    mgr.addProcess(createTestProcess({ id: "ba-2", creator: TEST_DID }));
    mgr.addProcess(createTestProcess({ id: "ba-3", creator: TEST_DID_2 }));

    expect(mgr.getProcessesByAccount(TEST_DID)).toHaveLength(2);
    expect(mgr.getProcessesByAccount(TEST_DID_2)).toHaveLength(1);
    expect(mgr.getProcessesByAccount("unknown")).toHaveLength(0);
  });

  it("should return active processes (pending + active)", () => {
    const mgr = new ScheduleProcessStateManager();
    mgr.addProcess(createTestProcess({ id: "act-1", status: "pending" }));
    mgr.addProcess(createTestProcess({ id: "act-2", status: "active" }));
    mgr.addProcess(createTestProcess({ id: "act-3", status: "completed" }));

    const active = mgr.getActiveProcesses();
    expect(active).toHaveLength(2);
    const ids = active.map((p) => p.id);
    expect(ids).toContain("act-1");
    expect(ids).toContain("act-2");
  });
});

// ============ Events Tests ============

describe("ScheduleProcessStateManager - Events", () => {
  it("should emit events to registered listeners", () => {
    const manager = new ScheduleProcessStateManager();
    const events: ScheduleEvent[] = [];
    manager.onEvent((e) => events.push(e));

    manager.addProcess(createTestProcess({ id: "ev-emit-1" }));
    expect(events.length).toBeGreaterThanOrEqual(1);
  });

  it("should support multiple listeners", () => {
    const manager = new ScheduleProcessStateManager();
    const events1: ScheduleEvent[] = [];
    const events2: ScheduleEvent[] = [];

    manager.onEvent((e) => events1.push(e));
    manager.onEvent((e) => events2.push(e));

    manager.addProcess(createTestProcess({ id: "ev-multi-1" }));

    expect(events1.length).toBeGreaterThanOrEqual(1);
    expect(events2.length).toBeGreaterThanOrEqual(1);
  });

  it("should support unsubscribing", () => {
    const manager = new ScheduleProcessStateManager();
    const events: ScheduleEvent[] = [];
    const unsubscribe = manager.onEvent((e) => events.push(e));

    manager.addProcess(createTestProcess({ id: "ev-unsub-1" }));
    const countAfterFirst = events.length;

    unsubscribe();
    manager.addProcess(createTestProcess({ id: "ev-unsub-2" }));

    expect(events.length).toBe(countAfterFirst);
  });

  it("should not throw when listener throws", () => {
    const manager = new ScheduleProcessStateManager();
    manager.onEvent(() => {
      throw new Error("listener exploded");
    });

    expect(() =>
      manager.addProcess(createTestProcess({ id: "ev-throw-1" })),
    ).not.toThrow();
  });
});

// ============ Serialization Tests ============

describe("ScheduleProcessStateManager - Export/Import", () => {
  it("should export and import state preserving processes", () => {
    const manager1 = new ScheduleProcessStateManager();
    manager1.addProcess(createTestProcess({ id: "exp-1", creator: TEST_DID }));
    manager1.addProcess(
      createTestProcess({ id: "exp-2", creator: TEST_DID_2 }),
    );
    manager1.incrementExecutions();

    const exported = manager1.exportState();

    const manager2 = new ScheduleProcessStateManager();
    manager2.importState(exported);

    expect(manager2.getProcess("exp-1")).toBeDefined();
    expect(manager2.getProcess("exp-2")).toBeDefined();
    expect(manager2.getMetrics().totalExecutions).toBe(1);
  });

  it("should rebuild indexes after import", () => {
    const manager1 = new ScheduleProcessStateManager();
    manager1.addProcess(
      createTestProcess({
        id: "imp-idx-1",
        creator: TEST_DID,
        status: "active",
        watchedStateKeys: ["key-x"],
      }),
    );

    const exported = manager1.exportState();
    const manager2 = new ScheduleProcessStateManager();
    manager2.importState(exported);

    expect(manager2.getProcessesByAccount(TEST_DID)).toHaveLength(1);
    expect(manager2.getProcessesByStatus("active")).toHaveLength(1);
    expect(manager2.getProcessesWatchingKey("key-x")).toHaveLength(1);
  });

  it("should have matching state root after import", () => {
    const manager1 = new ScheduleProcessStateManager();
    manager1.addProcess(createTestProcess({ id: "imp-root-1" }));

    const exported = manager1.exportState();
    const manager2 = new ScheduleProcessStateManager();
    manager2.importState(exported);

    expect(manager2.getStateRoot()).toBe(manager1.getStateRoot());
  });
});

// ============ Metrics Tests ============

describe("ScheduleProcessStateManager - Metrics", () => {
  it("should track processCountByStatus in metrics", () => {
    const mgr = new ScheduleProcessStateManager();
    mgr.addProcess(createTestProcess({ id: "met-1", status: "active" }));
    mgr.addProcess(createTestProcess({ id: "met-2", status: "pending" }));
    mgr.addProcess(createTestProcess({ id: "met-3", status: "pending" }));

    const metrics = mgr.getMetrics();
    expect(metrics.processCountByStatus["active"]).toBe(1);
    expect(metrics.processCountByStatus["pending"]).toBe(2);
  });

  it("should track processCountByType in metrics", () => {
    const mgr = new ScheduleProcessStateManager();
    mgr.addProcess(
      createTestProcess({
        id: "mty-1",
        condition: { time: { type: "absolute", executeAt: 1000 } },
      }),
    );
    mgr.addProcess(
      createTestProcess({
        id: "mty-2",
        condition: { time: { type: "periodic", interval: 5000 } },
      }),
    );

    const metrics = mgr.getMetrics();
    expect(metrics.processCountByType["time_absolute"]).toBe(1);
    expect(metrics.processCountByType["time_periodic"]).toBe(1);
  });

  it("should incrementExecutions correctly", () => {
    const mgr = new ScheduleProcessStateManager();
    mgr.incrementExecutions();
    mgr.incrementExecutions();
    mgr.incrementExecutions();
    expect(mgr.getMetrics().totalExecutions).toBe(3);
  });
});
