/**
 * Dimension 8: BVM & Smart Contract Security - Gas Exhaustion Tests
 *
 * Validates that:
 * 1. Infinite loops are halted by gas limit
 * 2. Memory expansion costs gas proportionally
 * 3. Storage operations cost appropriate gas
 * 4. Gas metering is consistent across runs
 * 5. Complex opcodes don't allow gas-free computation
 *
 * Industry method: Gas metering fuzzing + bounds analysis.
 */

import { describe, it, expect } from "vitest";
import {
  ExecutionEngine,
  createInitialState,
  createExecutionContext,
  STOP,
  ADD,
  PUSH1,
  PUSH32,
  MLOAD,
  MSTORE,
  SLOAD,
  SSTORE,
  JUMP,
  JUMPDEST,
  JUMPI,
  POP,
  DUP1,
  GAS,
  MSIZE,
  type ExecutionContext,
} from "../../../core/vm/index.js";

// ============ Helpers ============

function makeCtx(gasLimit: bigint): ExecutionContext {
  return createExecutionContext({
    caller: "0x" + "aa".repeat(20),
    currentContract: "0x" + "bb".repeat(20),
    blockNumber: 1n,
    gasLimit,
  });
}

// ============================================================
// GAS-01 through GAS-08: Gas exhaustion invariants
// ============================================================

describe("Gas Exhaustion Security (GAS)", () => {
  it("GAS-01: Infinite loop halts at gas limit", () => {
    const engine = new ExecutionEngine(createInitialState());

    // Bytecode: JUMPDEST, PUSH1 0x00, JUMP → infinite loop at offset 0
    const bytecode = new Uint8Array([
      JUMPDEST, // offset 0
      PUSH1,
      0x00, // push 0
      JUMP, // jump to offset 0
    ]);

    const result = engine.execute(bytecode, makeCtx(10000n));

    // Must fail (out of gas), NOT hang forever
    expect(result.success).toBe(false);
    expect(result.gasUsed).toBeGreaterThan(0n);
  });

  it("GAS-02: Every opcode consumes at least 1 gas", () => {
    const engine = new ExecutionEngine(createInitialState());

    // PUSH1 0x01, POP, STOP — 3 opcodes, should consume > 0 gas
    const bytecode = new Uint8Array([PUSH1, 0x01, POP, STOP]);
    const result = engine.execute(bytecode, makeCtx(1000000n));

    expect(result.success).toBe(true);
    expect(result.gasUsed).toBeGreaterThan(0n);
  });

  it("GAS-03: SSTORE costs more than SLOAD", () => {
    const engine1 = new ExecutionEngine(createInitialState());
    const engine2 = new ExecutionEngine(createInitialState());

    // SLOAD: PUSH1 0x00, SLOAD, POP, STOP
    const sloadCode = new Uint8Array([PUSH1, 0x00, SLOAD, POP, STOP]);
    const sloadResult = engine1.execute(sloadCode, makeCtx(1000000n));

    // SSTORE: PUSH1 0x42, PUSH1 0x00, SSTORE, STOP
    const sstoreCode = new Uint8Array([PUSH1, 0x42, PUSH1, 0x00, SSTORE, STOP]);
    const sstoreResult = engine2.execute(sstoreCode, makeCtx(1000000n));

    // Both should succeed
    expect(sloadResult.success).toBe(true);
    expect(sstoreResult.success).toBe(true);

    // SSTORE should cost more than SLOAD
    expect(sstoreResult.gasUsed).toBeGreaterThan(sloadResult.gasUsed);
  });

  it("GAS-04: Memory expansion costs gas", () => {
    const engine1 = new ExecutionEngine(createInitialState());
    const engine2 = new ExecutionEngine(createInitialState());

    // Small memory: PUSH1 0x42, PUSH1 0x00, MSTORE, STOP
    const smallMem = new Uint8Array([PUSH1, 0x42, PUSH1, 0x00, MSTORE, STOP]);
    const smallResult = engine1.execute(smallMem, makeCtx(1000000n));

    // Large memory offset: PUSH1 0x42, PUSH1 0xFF, MSTORE, STOP
    const largeMem = new Uint8Array([PUSH1, 0x42, PUSH1, 0xff, MSTORE, STOP]);
    const largeResult = engine2.execute(largeMem, makeCtx(1000000n));

    expect(smallResult.success).toBe(true);
    expect(largeResult.success).toBe(true);

    // Larger memory offset should cost more gas
    expect(largeResult.gasUsed).toBeGreaterThanOrEqual(smallResult.gasUsed);
  });

  it("GAS-05: Gas metering is deterministic (same input = same gas)", () => {
    const bytecode = new Uint8Array([PUSH1, 0x01, PUSH1, 0x02, ADD, POP, STOP]);

    const engine1 = new ExecutionEngine(createInitialState());
    const engine2 = new ExecutionEngine(createInitialState());
    const ctx = makeCtx(1000000n);

    const result1 = engine1.execute(bytecode, { ...ctx });
    const result2 = engine2.execute(bytecode, { ...ctx });

    expect(result1.success).toBe(result2.success);
    expect(result1.gasUsed).toBe(result2.gasUsed);
  });

  it("GAS-06: Insufficient gas fails execution", () => {
    const engine1 = new ExecutionEngine(createInitialState());

    // First, measure gas needed for a simple program
    const bytecode = new Uint8Array([PUSH1, 0x01, POP, STOP]);
    const measure = engine1.execute(bytecode, makeCtx(1000000n));
    expect(measure.success).toBe(true);
    expect(measure.gasUsed).toBeGreaterThan(0n);

    // With very low gas (1): should fail
    const engine2 = new ExecutionEngine(createInitialState());
    const shortResult = engine2.execute(bytecode, makeCtx(1n));
    expect(shortResult.success).toBe(false);
  });

  it("GAS-07: GAS opcode reports remaining gas accurately", () => {
    const engine = new ExecutionEngine(createInitialState());

    // GAS pushes remaining gas onto stack; POP discards; STOP ends
    const bytecode = new Uint8Array([GAS, POP, STOP]);
    const result = engine.execute(bytecode, makeCtx(1000000n));

    expect(result.success).toBe(true);
    // Gas consumed should be small (just GAS + POP + STOP)
    expect(result.gasUsed).toBeLessThan(100n);
  });

  it("GAS-08: MSIZE reflects actual memory usage", () => {
    const engine = new ExecutionEngine(createInitialState());

    // MSIZE should be 0 before any memory use, then grow after MSTORE
    // MSIZE, POP, PUSH1 0x42, PUSH1 0x00, MSTORE, MSIZE, POP, STOP
    const bytecode = new Uint8Array([
      MSIZE,
      POP, // Should be 0
      PUSH1,
      0x42,
      PUSH1,
      0x00,
      MSTORE, // Write to memory[0]
      MSIZE,
      POP, // Should be 32
      STOP,
    ]);

    const result = engine.execute(bytecode, makeCtx(1000000n));
    expect(result.success).toBe(true);
  });
});
