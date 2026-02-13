/**
 * BVM Execution Engine
 *
 * The heart of the BVM - interprets bytecode and executes operations.
 * Designed for agent-first execution with native support for:
 * - Agent identity operations
 * - Belief management
 * - Relationship tracking
 * - Knowledge queries
 * - Self-invocation scheduling
 *
 * Execution Model:
 *
 *   ┌─────────────────────────────────────────────────────────────────┐
 *   │                    EXECUTION CYCLE                              │
 *   ├─────────────────────────────────────────────────────────────────┤
 *   │                                                                  │
 *   │   1. FETCH    ─── Read opcode at PC                             │
 *   │        │                                                         │
 *   │        ▼                                                         │
 *   │   2. DECODE   ─── Look up opcode definition                     │
 *   │        │                                                         │
 *   │        ▼                                                         │
 *   │   3. VALIDATE ─── Check stack, gas, permissions                 │
 *   │        │                                                         │
 *   │        ▼                                                         │
 *   │   4. EXECUTE  ─── Perform operation                             │
 *   │        │                                                         │
 *   │        ▼                                                         │
 *   │   5. UPDATE   ─── Update state, PC, gas                         │
 *   │        │                                                         │
 *   │        └──────► Repeat until STOP/RETURN/REVERT/OUT_OF_GAS     │
 *   │                                                                  │
 *   └─────────────────────────────────────────────────────────────────┘
 */

import type { DID } from "../agents/identity/did.js";
import type {
  BVMState,
  ExecutionContext,
  ExecutionResult,
  ExecutionErrorCode,
  VMMemory,
  VMStack,
  Word,
  Address,
  Bytecode,
  Log,
  StateChange,
  BeliefChange,
  RelationshipChange,
  BVMConfig,
  AgentBelief,
  AgentRelationships,
  BeliefSource,
  BVMKnowledgeService,
} from "./types.js";
import { DEFAULT_BVM_CONFIG, BVMError, NullKnowledgeService } from "./types.js";
import * as opcodes from "./opcodes.js";
import {
  getOpcodeDefinition,
  isStateModifying,
  isAgentOpcode,
  getMnemonic,
} from "./opcodes.js";
import { keccak_256 } from "@noble/hashes/sha3";
import { domainHash } from "../crypto/hashing.js";
import { bytesToHex } from "../crypto/keys.js";

/** Extended AgentRelationships with optional blocked set (used by BLOCK_AGENT/UNBLOCK_AGENT opcodes) */
type AgentRelationshipsWithBlocked = AgentRelationships & {
  blocked?: Set<string>;
};

// 256-bit max value for two's complement operations
const MAX_UINT256 = (1n << 256n) - 1n;
const SIGN_BIT = 1n << 255n;

// ============ System Call Constants ============

/** Maximum call depth to prevent stack overflow attacks */
const MAX_CALL_DEPTH = 1024;

/** Gas stipend for calls with value transfer */
const CALL_STIPEND = 2300n;

// Reserved gas constants for future use:
// CREATE_GAS = 32000n
// INIT_CODE_WORD_GAS = 2n
// CALL_BASE_GAS = 700n
// CALL_VALUE_GAS = 9000n
// CALL_NEW_ACCOUNT_GAS = 25000n

// ============ Memory Implementation ============

class Memory implements VMMemory {
  data: Uint8Array;
  size: number;
  maxSize: number;

  constructor(maxSize: number = 1024 * 1024) {
    this.data = new Uint8Array(0);
    this.size = 0;
    this.maxSize = maxSize;
  }

  read(offset: number, length: number): Uint8Array {
    this.ensureSize(offset + length);
    return this.data.slice(offset, offset + length);
  }

  write(offset: number, data: Uint8Array): void {
    this.ensureSize(offset + data.length);
    this.data.set(data, offset);
  }

  expand(newSize: number): bigint {
    if (newSize <= this.size) return 0n;
    if (newSize > this.maxSize) {
      throw new BVMError("Memory limit exceeded", "EXECUTION_FAILED");
    }

    // Integer ceiling division: (x + 31) / 32 truncated
    const oldWords = ((this.size + 31) / 32) | 0;
    const newWords = ((newSize + 31) / 32) | 0;
    const wordsDiff = newWords - oldWords;

    // Gas cost: 3 * words + words^2 / 512 (all integer arithmetic)
    const gasCost =
      BigInt(3 * wordsDiff) +
      BigInt(((newWords * newWords) / 512) | 0) -
      BigInt(((oldWords * oldWords) / 512) | 0);

    this.ensureSize(newSize);
    return gasCost;
  }

  private ensureSize(minSize: number): void {
    if (minSize > this.size) {
      const newData = new Uint8Array(minSize);
      newData.set(this.data);
      this.data = newData;
      this.size = minSize;
    }
  }
}

// ============ Stack Implementation ============

class Stack implements VMStack {
  items: Word[];
  maxDepth: number;

  constructor(maxDepth: number = 1024) {
    this.items = [];
    this.maxDepth = maxDepth;
  }

  push(value: Word): void {
    if (this.items.length >= this.maxDepth) {
      throw new BVMError("Stack overflow", "EXECUTION_FAILED", {
        code: "STACK_OVERFLOW",
      });
    }
    this.items.push(value);
  }

  pop(): Word {
    if (this.items.length === 0) {
      throw new BVMError("Stack underflow", "EXECUTION_FAILED", {
        code: "STACK_UNDERFLOW",
      });
    }
    return this.items.pop()!;
  }

  peek(depth: number = 0): Word {
    const index = this.items.length - 1 - depth;
    if (index < 0) {
      throw new BVMError("Stack underflow", "EXECUTION_FAILED", {
        code: "STACK_UNDERFLOW",
      });
    }
    return this.items[index];
  }

  swap(depth: number): void {
    const top = this.items.length - 1;
    const target = top - depth;
    if (target < 0) {
      throw new BVMError("Stack underflow", "EXECUTION_FAILED", {
        code: "STACK_UNDERFLOW",
      });
    }
    [this.items[top], this.items[target]] = [
      this.items[target],
      this.items[top],
    ];
  }

  dup(depth: number): void {
    const index = this.items.length - depth;
    if (index < 0) {
      throw new BVMError("Stack underflow", "EXECUTION_FAILED", {
        code: "STACK_UNDERFLOW",
      });
    }
    this.push(this.items[index]);
  }

  depth(): number {
    return this.items.length;
  }
}

// ============ Execution Engine ============

/**
 * BVM Execution Engine
 */
export class ExecutionEngine {
  private config: BVMConfig;
  private state: BVMState;
  private knowledgeService: BVMKnowledgeService;

  // Current execution state
  private memory: Memory;
  private stack: Stack;
  private returnData: Uint8Array = new Uint8Array(0);
  private pc: number = 0;
  private gasUsed: bigint = 0n;
  private logs: Log[] = [];
  private stateChanges: StateChange[] = [];
  private beliefChanges: BeliefChange[] = [];
  private relationshipChanges: RelationshipChange[] = [];

  // Call depth tracking for CREATE/CALL opcodes
  private callDepth: number = 0;
  // AUTONOMY: Schedule nonce for deterministic schedule ID generation
  private scheduleNonce: number = 0;
  // Deterministic nonce for stake/session IDs (replaces Math.random)
  private idNonce: number = 0;

  constructor(
    state: BVMState,
    config: BVMConfig = DEFAULT_BVM_CONFIG,
    knowledgeService?: BVMKnowledgeService,
  ) {
    this.config = config;
    this.state = state;
    this.knowledgeService = knowledgeService ?? new NullKnowledgeService();
    this.memory = new Memory(config.maxMemorySize);
    this.stack = new Stack(config.maxStackDepth);
  }

  /**
   * Execute bytecode in context
   */
  execute(bytecode: Bytecode, context: ExecutionContext): ExecutionResult {
    // Reset execution state
    this.memory = new Memory(this.config.maxMemorySize);
    this.stack = new Stack(this.config.maxStackDepth);
    this.returnData = new Uint8Array(0);
    this.pc = 0;
    this.gasUsed = 0n;
    this.logs = [];
    this.stateChanges = [];
    this.beliefChanges = [];
    this.relationshipChanges = [];
    this.scheduleNonce = 0;
    this.idNonce = 0;

    try {
      // Main execution loop
      while (this.pc < bytecode.length) {
        const opcode = bytecode[this.pc];

        // Check gas
        if (this.gasUsed >= context.gasLimit) {
          return this.createErrorResult("OUT_OF_GAS", "Out of gas");
        }

        // Get opcode definition
        const def = getOpcodeDefinition(opcode);
        if (!def) {
          return this.createErrorResult(
            "INVALID_OPCODE",
            `Invalid opcode: 0x${opcode.toString(16)}`,
          );
        }

        // Check static call restrictions
        if (context.mode === "staticcall" && isStateModifying(opcode)) {
          return this.createErrorResult(
            "STATIC_STATE_CHANGE",
            "State modification in static call",
          );
        }

        // Consume base gas
        const gasCost = typeof def.gasCost === "bigint" ? def.gasCost : 0n;
        this.gasUsed += gasCost;

        // Execute opcode
        const result = this.executeOpcode(opcode, bytecode, context);

        if (result === "STOP") {
          break;
        } else if (result === "RETURN") {
          return {
            success: true,
            returnData: this.returnData,
            gasUsed: this.gasUsed,
            logs: this.logs,
            stateChanges: this.stateChanges,
            beliefChanges: this.beliefChanges,
            relationshipChanges: this.relationshipChanges,
          };
        } else if (result === "REVERT") {
          return {
            success: false,
            returnData: this.returnData,
            gasUsed: this.gasUsed,
            logs: [],
            stateChanges: [],
            error: { code: "REVERT", message: "Execution reverted" },
          };
        }

        this.pc++;
      }

      return {
        success: true,
        returnData: this.returnData,
        gasUsed: this.gasUsed,
        logs: this.logs,
        stateChanges: this.stateChanges,
        beliefChanges: this.beliefChanges,
        relationshipChanges: this.relationshipChanges,
      };
    } catch (error) {
      if (error instanceof BVMError) {
        return this.createErrorResult(
          error.code as ExecutionErrorCode,
          error.message,
        );
      }
      return this.createErrorResult(
        "EXECUTION_FAILED",
        error instanceof Error ? error.message : "Unknown error",
      );
    }
  }

  /**
   * Execute a single opcode
   */
  private executeOpcode(
    opcode: number,
    bytecode: Bytecode,
    context: ExecutionContext,
  ): "CONTINUE" | "STOP" | "RETURN" | "REVERT" {
    switch (opcode) {
      // ============ Stop and Arithmetic ============
      case opcodes.STOP:
        return "STOP";

      case opcodes.ADD: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push((a + b) & ((1n << 256n) - 1n));
        return "CONTINUE";
      }

      case opcodes.MUL: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push((a * b) & ((1n << 256n) - 1n));
        return "CONTINUE";
      }

      case opcodes.SUB: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push((a - b + (1n << 256n)) & ((1n << 256n) - 1n));
        return "CONTINUE";
      }

      case opcodes.DIV: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push(b === 0n ? 0n : a / b);
        return "CONTINUE";
      }

      case opcodes.MOD: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push(b === 0n ? 0n : a % b);
        return "CONTINUE";
      }

      case opcodes.SDIV: {
        // Signed division
        const a = this.stack.pop();
        const b = this.stack.pop();
        if (b === 0n) {
          this.stack.push(0n);
        } else {
          const aSign = (a & SIGN_BIT) !== 0n;
          const bSign = (b & SIGN_BIT) !== 0n;
          const aAbs = aSign ? (~a + 1n) & MAX_UINT256 : a;
          const bAbs = bSign ? (~b + 1n) & MAX_UINT256 : b;
          let result = aAbs / bAbs;
          if (aSign !== bSign) {
            result = (~result + 1n) & MAX_UINT256;
          }
          this.stack.push(result);
        }
        return "CONTINUE";
      }

      case opcodes.SMOD: {
        // Signed modulo
        const a = this.stack.pop();
        const b = this.stack.pop();
        if (b === 0n) {
          this.stack.push(0n);
        } else {
          const aSign = (a & SIGN_BIT) !== 0n;
          const bSign = (b & SIGN_BIT) !== 0n;
          const aAbs = aSign ? (~a + 1n) & MAX_UINT256 : a;
          const bAbs = bSign ? (~b + 1n) & MAX_UINT256 : b;
          let result = aAbs % bAbs;
          // Result takes sign of dividend (a)
          if (aSign && result !== 0n) {
            result = (~result + 1n) & MAX_UINT256;
          }
          this.stack.push(result);
        }
        return "CONTINUE";
      }

      case opcodes.ADDMOD: {
        // (a + b) % N
        const a = this.stack.pop();
        const b = this.stack.pop();
        const N = this.stack.pop();
        if (N === 0n) {
          this.stack.push(0n);
        } else {
          // Use BigInt to handle overflow correctly
          this.stack.push((a + b) % N);
        }
        return "CONTINUE";
      }

      case opcodes.MULMOD: {
        // (a * b) % N
        const a = this.stack.pop();
        const b = this.stack.pop();
        const N = this.stack.pop();
        if (N === 0n) {
          this.stack.push(0n);
        } else {
          // BigInt handles arbitrary precision
          this.stack.push((a * b) % N);
        }
        return "CONTINUE";
      }

      case opcodes.EXP: {
        // a ** b mod 2^256
        const base = this.stack.pop();
        const exponent = this.stack.pop();
        // P1-V1 FIX: Charge dynamic gas based on exponent byte size (EIP-160: 50 gas/byte)
        if (exponent > 0n) {
          const hexLen = exponent.toString(16).length;
          const expByteSize = BigInt((hexLen + 1) >>> 1);
          this.gasUsed += 50n * expByteSize;
          if (this.gasUsed >= context.gasLimit) {
            throw new BVMError("Out of gas (EXP dynamic)", "GAS_EXHAUSTED");
          }
        }
        let result = 1n;
        let b = base;
        let e = exponent;
        while (e > 0n) {
          if (e & 1n) {
            result = (result * b) & MAX_UINT256;
          }
          b = (b * b) & MAX_UINT256;
          e >>= 1n;
        }
        this.stack.push(result);
        return "CONTINUE";
      }

      case opcodes.SIGNEXTEND: {
        // Sign extend from byte position k
        const k = this.stack.pop();
        const x = this.stack.pop();
        if (k < 32n) {
          const bitPos = Number(k) * 8 + 7;
          const signBit = (x >> BigInt(bitPos)) & 1n;
          if (signBit) {
            // Extend with 1s
            const mask = MAX_UINT256 << BigInt(bitPos + 1);
            this.stack.push(x | mask);
          } else {
            // Extend with 0s
            const mask = (1n << BigInt(bitPos + 1)) - 1n;
            this.stack.push(x & mask);
          }
        } else {
          this.stack.push(x);
        }
        return "CONTINUE";
      }

      // ============ Comparison ============
      case opcodes.LT: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push(a < b ? 1n : 0n);
        return "CONTINUE";
      }

      case opcodes.GT: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push(a > b ? 1n : 0n);
        return "CONTINUE";
      }

      case opcodes.EQ: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push(a === b ? 1n : 0n);
        return "CONTINUE";
      }

      case opcodes.ISZERO: {
        const a = this.stack.pop();
        this.stack.push(a === 0n ? 1n : 0n);
        return "CONTINUE";
      }

      case opcodes.SLT: {
        // Signed less than
        const a = this.stack.pop();
        const b = this.stack.pop();
        const aSign = (a & SIGN_BIT) !== 0n;
        const bSign = (b & SIGN_BIT) !== 0n;
        let result: bigint;
        if (aSign && !bSign) {
          result = 1n; // negative < positive
        } else if (!aSign && bSign) {
          result = 0n; // positive >= negative
        } else {
          result = a < b ? 1n : 0n;
        }
        this.stack.push(result);
        return "CONTINUE";
      }

      case opcodes.SGT: {
        // Signed greater than
        const a = this.stack.pop();
        const b = this.stack.pop();
        const aSign = (a & SIGN_BIT) !== 0n;
        const bSign = (b & SIGN_BIT) !== 0n;
        let result: bigint;
        if (aSign && !bSign) {
          result = 0n; // negative not > positive
        } else if (!aSign && bSign) {
          result = 1n; // positive > negative
        } else {
          result = a > b ? 1n : 0n;
        }
        this.stack.push(result);
        return "CONTINUE";
      }

      // ============ Bitwise ============
      case opcodes.AND: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push(a & b);
        return "CONTINUE";
      }

      case opcodes.OR: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push(a | b);
        return "CONTINUE";
      }

      case opcodes.XOR: {
        const a = this.stack.pop();
        const b = this.stack.pop();
        this.stack.push(a ^ b);
        return "CONTINUE";
      }

      case opcodes.NOT: {
        const a = this.stack.pop();
        this.stack.push(~a & MAX_UINT256);
        return "CONTINUE";
      }

      case opcodes.BYTE: {
        // Get byte at position i from word
        const i = this.stack.pop();
        const x = this.stack.pop();
        if (i >= 32n) {
          this.stack.push(0n);
        } else {
          // i=0 is most significant byte
          const shiftAmount = (31n - i) * 8n;
          this.stack.push((x >> shiftAmount) & 0xffn);
        }
        return "CONTINUE";
      }

      case opcodes.SHL: {
        // Shift left
        const shift = this.stack.pop();
        const value = this.stack.pop();
        if (shift >= 256n) {
          this.stack.push(0n);
        } else {
          this.stack.push((value << shift) & MAX_UINT256);
        }
        return "CONTINUE";
      }

      case opcodes.SHR: {
        // Logical shift right
        const shift = this.stack.pop();
        const value = this.stack.pop();
        if (shift >= 256n) {
          this.stack.push(0n);
        } else {
          this.stack.push(value >> shift);
        }
        return "CONTINUE";
      }

      case opcodes.SAR: {
        // Arithmetic shift right (preserves sign)
        const shift = this.stack.pop();
        const value = this.stack.pop();
        const isNegative = (value & SIGN_BIT) !== 0n;
        if (shift >= 256n) {
          this.stack.push(isNegative ? MAX_UINT256 : 0n);
        } else {
          let result = value >> shift;
          if (isNegative) {
            // Fill high bits with 1s
            const mask = MAX_UINT256 << (256n - shift);
            result = result | mask;
          }
          this.stack.push(result & MAX_UINT256);
        }
        return "CONTINUE";
      }

      // ============ SHA3 ============
      case opcodes.SHA3: {
        // Keccak256 hash of memory region
        const offset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        const data = this.memory.read(offset, length);
        const hash = keccak_256(data);
        this.stack.push(this.bytesToWord(hash));
        // Dynamic gas: 30 + 6 * ceil(length / 32)
        this.gasUsed += BigInt(6 * ((length + 31) >>> 5));
        return "CONTINUE";
      }

      // ============ Stack Operations ============
      case opcodes.POP: {
        this.stack.pop();
        return "CONTINUE";
      }

      // ============ Memory Operations ============
      case opcodes.MLOAD: {
        const offset = Number(this.stack.pop());
        const data = this.memory.read(offset, 32);
        this.stack.push(this.bytesToWord(data));
        return "CONTINUE";
      }

      case opcodes.MSTORE: {
        const offset = Number(this.stack.pop());
        const value = this.stack.pop();
        this.memory.write(offset, this.wordToBytes(value));
        return "CONTINUE";
      }

      case opcodes.MSIZE: {
        this.stack.push(BigInt(this.memory.size));
        return "CONTINUE";
      }

      case opcodes.MSTORE8: {
        // Store single byte to memory
        const offset = Number(this.stack.pop());
        const value = this.stack.pop();
        const byte = new Uint8Array([Number(value & 0xffn)]);
        this.memory.write(offset, byte);
        return "CONTINUE";
      }

      // ============ Storage Operations ============
      case opcodes.SLOAD: {
        const key = this.stack.pop();
        const keyHex = key.toString(16).padStart(64, "0");
        const storage = this.state.contractVariables.storage.get(
          context.currentContract,
        );
        const value = storage?.slots.get(keyHex);
        this.stack.push(value ? this.bytesToWord(value) : 0n);
        return "CONTINUE";
      }

      case opcodes.SSTORE: {
        const key = this.stack.pop();
        const value = this.stack.pop();
        const keyHex = key.toString(16).padStart(64, "0");

        let storage = this.state.contractVariables.storage.get(
          context.currentContract,
        );
        if (!storage) {
          storage = {
            slots: new Map(),
            storageRoot: new Uint8Array(32),
            lastModified: context.blockNumber,
          };
          this.state.contractVariables.storage.set(
            context.currentContract,
            storage,
          );
        }

        const oldValue = storage.slots.get(keyHex);
        const oldWord = oldValue ? this.bytesToWord(oldValue) : 0n;

        // P1-V2 FIX: Dynamic gas for SSTORE (simplified EIP-2200 model)
        if (oldWord === 0n && value !== 0n) {
          this.gasUsed += this.config.gasCosts.sstoreSet; // 20000n new slot
        } else if (oldWord !== 0n && value === 0n) {
          this.gasUsed += this.config.gasCosts.sstoreReset; // 5000n clear slot
        } else {
          this.gasUsed += this.config.gasCosts.sstore; // 5000n update slot
        }
        if (this.gasUsed >= context.gasLimit) {
          throw new BVMError("Out of gas (SSTORE dynamic)", "GAS_EXHAUSTED");
        }

        storage.slots.set(keyHex, this.wordToBytes(value));
        storage.lastModified = context.blockNumber;

        this.stateChanges.push({
          type: "storage",
          address: context.currentContract,
          key: keyHex,
          oldValue: oldWord !== 0n ? oldWord : undefined,
          newValue: value,
        });

        return "CONTINUE";
      }

      // ============ Control Flow ============
      case opcodes.JUMP: {
        const dest = Number(this.stack.pop());
        if (
          dest < 0 ||
          dest >= bytecode.length ||
          bytecode[dest] !== opcodes.JUMPDEST
        ) {
          throw new BVMError("Invalid jump destination", "EXECUTION_FAILED", {
            code: "INVALID_JUMP",
          });
        }
        this.pc = dest - 1; // -1 because loop will increment
        return "CONTINUE";
      }

      case opcodes.JUMPI: {
        const dest = Number(this.stack.pop());
        const condition = this.stack.pop();
        if (condition !== 0n) {
          if (
            dest < 0 ||
            dest >= bytecode.length ||
            bytecode[dest] !== opcodes.JUMPDEST
          ) {
            throw new BVMError("Invalid jump destination", "EXECUTION_FAILED", {
              code: "INVALID_JUMP",
            });
          }
          this.pc = dest - 1;
        }
        return "CONTINUE";
      }

      case opcodes.PC: {
        this.stack.push(BigInt(this.pc));
        return "CONTINUE";
      }

      case opcodes.GAS: {
        this.stack.push(context.gasLimit - this.gasUsed);
        return "CONTINUE";
      }

      case opcodes.JUMPDEST: {
        // No-op, just a valid jump target
        return "CONTINUE";
      }

      // ============ Environment ============
      case opcodes.ADDRESS: {
        this.stack.push(this.addressToWord(context.currentContract));
        return "CONTINUE";
      }

      case opcodes.CALLER: {
        this.stack.push(this.addressToWord(context.caller));
        return "CONTINUE";
      }

      case opcodes.ORIGIN: {
        this.stack.push(this.addressToWord(context.origin));
        return "CONTINUE";
      }

      case opcodes.CALLVALUE: {
        this.stack.push(context.value);
        return "CONTINUE";
      }

      // P1-V3 FIX: BALANCE opcode was defined in opcodes.ts but not in execution switch
      case opcodes.BALANCE: {
        const addr = this.wordToAddress(this.stack.pop());
        const account = this.state.accounts.accounts.get(addr);
        this.stack.push(account?.balance ?? 0n);
        return "CONTINUE";
      }

      case opcodes.CALLDATASIZE: {
        this.stack.push(BigInt(context.callData.length));
        return "CONTINUE";
      }

      case opcodes.CALLDATALOAD: {
        // Load 32 bytes from calldata at offset
        const offset = Number(this.stack.pop());
        const data = new Uint8Array(32);
        for (let i = 0; i < 32; i++) {
          if (offset + i < context.callData.length) {
            data[i] = context.callData[offset + i];
          }
          // Remaining bytes are already 0
        }
        this.stack.push(this.bytesToWord(data));
        return "CONTINUE";
      }

      case opcodes.CALLDATACOPY: {
        // Copy calldata to memory
        const destOffset = Number(this.stack.pop());
        const dataOffset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        const data = new Uint8Array(length);
        for (let i = 0; i < length; i++) {
          if (dataOffset + i < context.callData.length) {
            data[i] = context.callData[dataOffset + i];
          }
        }
        this.memory.write(destOffset, data);
        // Dynamic gas: 3 * ceil(length / 32)
        this.gasUsed += BigInt(3 * ((length + 31) >>> 5));
        return "CONTINUE";
      }

      case opcodes.CODESIZE: {
        // Size of current contract's code
        this.stack.push(BigInt(bytecode.length));
        return "CONTINUE";
      }

      case opcodes.CODECOPY: {
        // Copy code to memory
        const destOffset = Number(this.stack.pop());
        const codeOffset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        const data = new Uint8Array(length);
        for (let i = 0; i < length; i++) {
          if (codeOffset + i < bytecode.length) {
            data[i] = bytecode[codeOffset + i];
          }
        }
        this.memory.write(destOffset, data);
        // Dynamic gas: 3 * ceil(length / 32)
        this.gasUsed += BigInt(3 * ((length + 31) >>> 5));
        return "CONTINUE";
      }

      case opcodes.GASPRICE: {
        // Gas price (would come from transaction context)
        this.stack.push(context.gasPrice ?? 1n);
        return "CONTINUE";
      }

      case opcodes.EXTCODESIZE: {
        // Size of external account's code
        const address = this.wordToAddress(this.stack.pop());
        const contract = this.state.contractBytecode.contracts.get(address);
        this.stack.push(BigInt(contract?.bytecode.length ?? 0));
        return "CONTINUE";
      }

      case opcodes.EXTCODECOPY: {
        // Copy external code to memory
        const address = this.wordToAddress(this.stack.pop());
        const destOffset = Number(this.stack.pop());
        const codeOffset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        const contract = this.state.contractBytecode.contracts.get(address);
        const code = contract?.bytecode ?? new Uint8Array(0);
        const data = new Uint8Array(length);
        for (let i = 0; i < length; i++) {
          if (codeOffset + i < code.length) {
            data[i] = code[codeOffset + i];
          }
        }
        this.memory.write(destOffset, data);
        // Dynamic gas: 3 * ceil(length / 32)
        this.gasUsed += BigInt(3 * ((length + 31) >>> 5));
        return "CONTINUE";
      }

      case opcodes.RETURNDATASIZE: {
        // Size of return data from last call
        this.stack.push(BigInt(this.returnData.length));
        return "CONTINUE";
      }

      case opcodes.RETURNDATACOPY: {
        // Copy return data to memory
        const destOffset = Number(this.stack.pop());
        const dataOffset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        if (dataOffset + length > this.returnData.length) {
          throw new BVMError(
            "Return data out of bounds",
            "RETURN_DATA_OUT_OF_BOUNDS",
          );
        }
        const data = this.returnData.slice(dataOffset, dataOffset + length);
        this.memory.write(destOffset, data);
        // Dynamic gas: 3 * ceil(length / 32)
        this.gasUsed += BigInt(3 * ((length + 31) >>> 5));
        return "CONTINUE";
      }

      case opcodes.EXTCODEHASH: {
        // Hash of external account's code
        const address = this.wordToAddress(this.stack.pop());
        const contract = this.state.contractBytecode.contracts.get(address);
        if (!contract || contract.bytecode.length === 0) {
          this.stack.push(0n);
        } else {
          const hash = keccak_256(contract.bytecode);
          this.stack.push(this.bytesToWord(hash));
        }
        return "CONTINUE";
      }

      // ============ Block Info ============
      case opcodes.TIMESTAMP: {
        this.stack.push(BigInt(context.timestamp));
        return "CONTINUE";
      }

      case opcodes.NUMBER: {
        this.stack.push(context.blockNumber);
        return "CONTINUE";
      }

      case opcodes.GASLIMIT: {
        this.stack.push(context.gasLimit);
        return "CONTINUE";
      }

      case opcodes.BLOCKHASH: {
        // Get hash of one of the 256 most recent blocks
        const blockNum = this.stack.pop();
        const currentBlock = context.blockNumber;
        // Only return hash for blocks within range [current-256, current-1]
        if (
          blockNum >= currentBlock ||
          blockNum < currentBlock - 256n ||
          blockNum < 0n
        ) {
          this.stack.push(0n);
        } else {
          // Placeholder - would query block hash from a block hash cache
          // For now, return 0 as block hashes aren't stored in BVMState
          this.stack.push(0n);
        }
        return "CONTINUE";
      }

      case opcodes.COINBASE: {
        // Block producer address
        // Use type assertion for extended context fields
        const extContext = context as ExecutionContext & { coinbase?: Address };
        const coinbase =
          extContext.coinbase ?? "0x0000000000000000000000000000000000000000";
        this.stack.push(this.addressToWord(coinbase));
        return "CONTINUE";
      }

      case opcodes.PREVRANDAO: {
        // Previous block's RANDAO value (post-merge randomness)
        const extContext = context as ExecutionContext & {
          prevRandao?: bigint;
        };
        const prevRandao = extContext.prevRandao ?? 0n;
        this.stack.push(prevRandao);
        return "CONTINUE";
      }

      case opcodes.CHAINID: {
        // Chain ID
        const extContext = context as ExecutionContext & { chainId?: bigint };
        const chainId = extContext.chainId ?? 1n;
        this.stack.push(chainId);
        return "CONTINUE";
      }

      case opcodes.SELFBALANCE: {
        // Balance of current contract
        const account = this.state.accounts.accounts.get(
          context.currentContract,
        );
        this.stack.push(account?.balance ?? 0n);
        return "CONTINUE";
      }

      case opcodes.BASEFEE: {
        // Current block's base fee
        const extContext = context as ExecutionContext & { baseFee?: bigint };
        const baseFee = extContext.baseFee ?? 0n;
        this.stack.push(baseFee);
        return "CONTINUE";
      }

      // ============ Logging ============
      case opcodes.LOG0: {
        const offset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        const data = this.memory.read(offset, length);
        this.logs.push({
          address: context.currentContract,
          topics: [],
          data,
          logIndex: this.logs.length,
        });
        // Gas: 375 + 8 * length
        this.gasUsed += BigInt(375 + 8 * length);
        return "CONTINUE";
      }

      case opcodes.LOG1: {
        const offset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        const topic0 = this.stack.pop();
        const data = this.memory.read(offset, length);
        this.logs.push({
          address: context.currentContract,
          topics: [this.wordToBytes(topic0)],
          data,
          logIndex: this.logs.length,
        });
        // Gas: 375 + 375 * numTopics + 8 * length
        this.gasUsed += BigInt(375 + 375 + 8 * length);
        return "CONTINUE";
      }

      case opcodes.LOG2: {
        const offset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        const topic0 = this.stack.pop();
        const topic1 = this.stack.pop();
        const data = this.memory.read(offset, length);
        this.logs.push({
          address: context.currentContract,
          topics: [this.wordToBytes(topic0), this.wordToBytes(topic1)],
          data,
          logIndex: this.logs.length,
        });
        // Gas: 375 + 375 * numTopics + 8 * length
        this.gasUsed += BigInt(375 + 750 + 8 * length);
        return "CONTINUE";
      }

      case opcodes.LOG3: {
        const offset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        const topic0 = this.stack.pop();
        const topic1 = this.stack.pop();
        const topic2 = this.stack.pop();
        const data = this.memory.read(offset, length);
        this.logs.push({
          address: context.currentContract,
          topics: [
            this.wordToBytes(topic0),
            this.wordToBytes(topic1),
            this.wordToBytes(topic2),
          ],
          data,
          logIndex: this.logs.length,
        });
        // Gas: 375 + 375 * numTopics + 8 * length
        this.gasUsed += BigInt(375 + 1125 + 8 * length);
        return "CONTINUE";
      }

      case opcodes.LOG4: {
        const offset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        const topic0 = this.stack.pop();
        const topic1 = this.stack.pop();
        const topic2 = this.stack.pop();
        const topic3 = this.stack.pop();
        const data = this.memory.read(offset, length);
        this.logs.push({
          address: context.currentContract,
          topics: [
            this.wordToBytes(topic0),
            this.wordToBytes(topic1),
            this.wordToBytes(topic2),
            this.wordToBytes(topic3),
          ],
          data,
          logIndex: this.logs.length,
        });
        // Gas: 375 + 375 * numTopics + 8 * length
        this.gasUsed += BigInt(375 + 1500 + 8 * length);
        return "CONTINUE";
      }

      // ============ System Call Operations ============
      case opcodes.CREATE: {
        const valueCreate = this.stack.pop();
        const offsetCreate = Number(this.stack.pop());
        const lengthCreate = Number(this.stack.pop());
        if (this.callDepth >= MAX_CALL_DEPTH) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        const callerAccount = this.state.accounts.accounts.get(
          context.currentContract,
        );
        if (!callerAccount || callerAccount.balance < valueCreate) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        const initCode = this.memory.read(offsetCreate, lengthCreate);
        const newAddress = this.computeContractAddress(
          context.currentContract,
          callerAccount.nonce,
        );
        if (this.state.contractBytecode.contracts.get(newAddress)) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        const createResult = this.executeSubCreate(
          context,
          initCode,
          newAddress,
          valueCreate,
          context.currentContract,
          context.caller,
        );
        if (createResult.success && createResult.returnData.length > 0) {
          this.deployContract(newAddress, createResult.returnData, context);
          if (valueCreate > 0n)
            this.transferValue(
              context.currentContract,
              newAddress,
              valueCreate,
              context.blockNumber,
            );
          callerAccount.nonce += 1n;
          this.stateChanges.push(...createResult.stateChanges);
          this.logs.push(...createResult.logs);
          this.stack.push(this.addressToWord(newAddress));
        } else {
          this.stack.push(0n);
        }
        this.gasUsed += createResult.gasUsed;
        this.returnData = createResult.returnData;
        return "CONTINUE";
      }

      case opcodes.CREATE2: {
        const valueCreate2 = this.stack.pop();
        const offsetCreate2 = Number(this.stack.pop());
        const lengthCreate2 = Number(this.stack.pop());
        const saltCreate2 = this.stack.pop();
        if (this.callDepth >= MAX_CALL_DEPTH) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        const callerAccount2 = this.state.accounts.accounts.get(
          context.currentContract,
        );
        if (!callerAccount2 || callerAccount2.balance < valueCreate2) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        const initCode2 = this.memory.read(offsetCreate2, lengthCreate2);
        const newAddress2 = this.computeCreate2Address(
          context.currentContract,
          saltCreate2,
          initCode2,
        );
        if (this.state.contractBytecode.contracts.get(newAddress2)) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        const create2Result = this.executeSubCreate(
          context,
          initCode2,
          newAddress2,
          valueCreate2,
          context.currentContract,
          context.caller,
        );
        if (create2Result.success && create2Result.returnData.length > 0) {
          this.deployContract(newAddress2, create2Result.returnData, context);
          if (valueCreate2 > 0n)
            this.transferValue(
              context.currentContract,
              newAddress2,
              valueCreate2,
              context.blockNumber,
            );
          this.stateChanges.push(...create2Result.stateChanges);
          this.logs.push(...create2Result.logs);
          this.stack.push(this.addressToWord(newAddress2));
        } else {
          this.stack.push(0n);
        }
        this.gasUsed += create2Result.gasUsed;
        this.returnData = create2Result.returnData;
        return "CONTINUE";
      }

      case opcodes.CALL: {
        const gasLimitCall = this.stack.pop();
        const targetAddressCall = this.wordToAddress(this.stack.pop());
        const valueCall = this.stack.pop();
        const argsOffsetCall = Number(this.stack.pop());
        const argsSizeCall = Number(this.stack.pop());
        const retOffsetCall = Number(this.stack.pop());
        const retSizeCall = Number(this.stack.pop());
        if (this.callDepth >= MAX_CALL_DEPTH) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        if (valueCall > 0n) {
          const callerAccountCall = this.state.accounts.accounts.get(
            context.currentContract,
          );
          if (!callerAccountCall || callerAccountCall.balance < valueCall) {
            this.stack.push(0n);
            return "CONTINUE";
          }
        }
        const callDataCall = this.memory.read(argsOffsetCall, argsSizeCall);
        const availableGasCall = context.gasLimit - this.gasUsed;
        const maxForwardGasCall = availableGasCall - availableGasCall / 64n;
        const forwardGasCall =
          gasLimitCall < maxForwardGasCall ? gasLimitCall : maxForwardGasCall;
        const actualGasCall =
          valueCall > 0n ? forwardGasCall + CALL_STIPEND : forwardGasCall;
        const callResult = this.executeSubCall(
          context,
          targetAddressCall,
          callDataCall,
          valueCall,
          actualGasCall,
          "call",
        );
        if (callResult.success && valueCall > 0n)
          this.transferValue(
            context.currentContract,
            targetAddressCall,
            valueCall,
            context.blockNumber,
          );
        if (callResult.returnData.length > 0 && retSizeCall > 0) {
          this.memory.write(
            retOffsetCall,
            callResult.returnData.slice(
              0,
              Math.min(callResult.returnData.length, retSizeCall),
            ),
          );
        }
        this.returnData = callResult.returnData;
        if (callResult.success) {
          this.stateChanges.push(...callResult.stateChanges);
          this.logs.push(...callResult.logs);
        }
        this.gasUsed += callResult.gasUsed;
        this.stack.push(callResult.success ? 1n : 0n);
        return "CONTINUE";
      }

      case opcodes.DELEGATECALL: {
        const gasLimitDel = this.stack.pop();
        const targetAddressDel = this.wordToAddress(this.stack.pop());
        const argsOffsetDel = Number(this.stack.pop());
        const argsSizeDel = Number(this.stack.pop());
        const retOffsetDel = Number(this.stack.pop());
        const retSizeDel = Number(this.stack.pop());
        if (this.callDepth >= MAX_CALL_DEPTH) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        const callDataDel = this.memory.read(argsOffsetDel, argsSizeDel);
        const availableGasDel = context.gasLimit - this.gasUsed;
        const maxForwardGasDel = availableGasDel - availableGasDel / 64n;
        const forwardGasDel =
          gasLimitDel < maxForwardGasDel ? gasLimitDel : maxForwardGasDel;
        const delegateResult = this.executeSubCall(
          context,
          targetAddressDel,
          callDataDel,
          context.value,
          forwardGasDel,
          "delegatecall",
        );
        if (delegateResult.returnData.length > 0 && retSizeDel > 0) {
          this.memory.write(
            retOffsetDel,
            delegateResult.returnData.slice(
              0,
              Math.min(delegateResult.returnData.length, retSizeDel),
            ),
          );
        }
        this.returnData = delegateResult.returnData;
        if (delegateResult.success) {
          this.stateChanges.push(...delegateResult.stateChanges);
          this.logs.push(...delegateResult.logs);
        }
        this.gasUsed += delegateResult.gasUsed;
        this.stack.push(delegateResult.success ? 1n : 0n);
        return "CONTINUE";
      }

      case opcodes.STATICCALL: {
        const gasLimitStatic = this.stack.pop();
        const targetAddressStatic = this.wordToAddress(this.stack.pop());
        const argsOffsetStatic = Number(this.stack.pop());
        const argsSizeStatic = Number(this.stack.pop());
        const retOffsetStatic = Number(this.stack.pop());
        const retSizeStatic = Number(this.stack.pop());
        if (this.callDepth >= MAX_CALL_DEPTH) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        const callDataStatic = this.memory.read(
          argsOffsetStatic,
          argsSizeStatic,
        );
        const availableGasStatic = context.gasLimit - this.gasUsed;
        const maxForwardGasStatic =
          availableGasStatic - availableGasStatic / 64n;
        const forwardGasStatic =
          gasLimitStatic < maxForwardGasStatic
            ? gasLimitStatic
            : maxForwardGasStatic;
        const staticResult = this.executeSubCall(
          context,
          targetAddressStatic,
          callDataStatic,
          0n,
          forwardGasStatic,
          "staticcall",
        );
        if (staticResult.returnData.length > 0 && retSizeStatic > 0) {
          this.memory.write(
            retOffsetStatic,
            staticResult.returnData.slice(
              0,
              Math.min(staticResult.returnData.length, retSizeStatic),
            ),
          );
        }
        this.returnData = staticResult.returnData;
        this.gasUsed += staticResult.gasUsed;
        this.stack.push(staticResult.success ? 1n : 0n);
        return "CONTINUE";
      }

      // ============ Return/Revert ============
      case opcodes.RETURN: {
        const offset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        this.returnData = this.memory.read(offset, length);
        return "RETURN";
      }

      case opcodes.REVERT: {
        const offset = Number(this.stack.pop());
        const length = Number(this.stack.pop());
        this.returnData = this.memory.read(offset, length);
        return "REVERT";
      }

      // ============ PUSH Operations ============
      default: {
        // Handle PUSH1-PUSH32
        if (opcode >= opcodes.PUSH1 && opcode <= opcodes.PUSH32) {
          const numBytes = opcode - opcodes.PUSH1 + 1;
          const value = bytecode.slice(this.pc + 1, this.pc + 1 + numBytes);
          this.stack.push(this.bytesToWord(value));
          this.pc += numBytes;
          return "CONTINUE";
        }

        // Handle DUP1-DUP16
        if (opcode >= opcodes.DUP1 && opcode <= opcodes.DUP16) {
          const depth = opcode - opcodes.DUP1 + 1;
          this.stack.dup(depth);
          return "CONTINUE";
        }

        // Handle SWAP1-SWAP16
        if (opcode >= opcodes.SWAP1 && opcode <= opcodes.SWAP16) {
          const depth = opcode - opcodes.SWAP1 + 1;
          this.stack.swap(depth);
          return "CONTINUE";
        }

        // Handle Agent Operations
        if (isAgentOpcode(opcode)) {
          return this.executeAgentOpcode(opcode, context);
        }

        throw new BVMError(
          `Unimplemented opcode: ${getMnemonic(opcode)}`,
          "EXECUTION_FAILED",
          { code: "INVALID_OPCODE" },
        );
      }
    }
  }

  /**
   * Execute agent-specific opcodes
   */
  private executeAgentOpcode(
    opcode: number,
    context: ExecutionContext,
  ): "CONTINUE" | "STOP" | "RETURN" | "REVERT" {
    const agentCtx = context.agentContext;

    switch (opcode) {
      // ============ Agent Identity (0xE0-0xE3) ============
      case opcodes.AGENT_DID: {
        // Push caller's DID hash to stack (or 0 if not agent)
        if (context.callerDID) {
          this.stack.push(this.didToWord(context.callerDID));
        } else {
          this.stack.push(0n);
        }
        return "CONTINUE";
      }

      case opcodes.AGENT_HOME: {
        // Push agent's home address
        if (agentCtx?.homeAddress) {
          this.stack.push(this.addressToWord(agentCtx.homeAddress));
        } else {
          this.stack.push(0n);
        }
        return "CONTINUE";
      }

      case opcodes.VERIFY_IDENTITY: {
        // (didHash) -> (isValid)
        const didHash = this.stack.pop();
        const did = this.wordToDID(didHash);
        const agent = this.state.agents.agents.get(did);
        this.stack.push(agent ? 1n : 0n);
        return "CONTINUE";
      }

      // ============ Agent Belief (0xE4-0xE7) ============
      case opcodes.GET_BELIEF: {
        // (factIdHash) -> (confidence * 1e18)
        if (!agentCtx || !context.callerDID) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const factIdHash = this.stack.pop();
        const factId = factIdHash.toString(16);
        const belief = agentCtx.beliefs.beliefs.get(factId);

        // Return confidence as fixed point (0-1e18)
        const confidence = belief
          ? BigInt(Math.floor(belief.confidence * 1e18))
          : 0n;
        this.stack.push(confidence);
        return "CONTINUE";
      }

      case opcodes.SET_BELIEF: {
        // (factIdHash, confidence, sourceType) -> ()
        if (!context.callerDID) {
          return "CONTINUE"; // Silent no-op if not agent
        }

        const factIdHash = this.stack.pop();
        const confidenceFixed = this.stack.pop();
        const sourceType = Number(this.stack.pop());

        const factId = factIdHash.toString(16);
        const confidence = Number(confidenceFixed) / 1e18;
        const source = this.numberToBeliefSource(sourceType);

        const agentRecord = this.state.agents.agents.get(context.callerDID);
        if (agentRecord) {
          const oldBelief = agentRecord.beliefs.beliefs.get(factId);

          const newBelief: AgentBelief = {
            factId,
            confidence,
            networkConfidence:
              this.knowledgeService.getNetworkConfidence(factId),
            source,
            firstBelieved: oldBelief?.firstBelieved ?? context.blockNumber,
            lastUpdated: context.blockNumber,
            contributed: oldBelief?.contributed ?? false,
          };

          agentRecord.beliefs.beliefs.set(factId, newBelief);
          agentRecord.beliefs.totalBeliefs = agentRecord.beliefs.beliefs.size;
          agentRecord.beliefs.lastUpdated = context.blockNumber;

          this.beliefChanges.push({
            did: context.callerDID,
            factId,
            oldConfidence: oldBelief?.confidence,
            newConfidence: confidence,
            source,
          });
        }
        return "CONTINUE";
      }

      case opcodes.COMPARE_BELIEF: {
        // (factIdHash) -> (alignment * 1e18)
        if (!agentCtx || !context.callerDID) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const factIdHash = this.stack.pop();
        const factId = factIdHash.toString(16);
        const belief = agentCtx.beliefs.beliefs.get(factId);

        if (!belief) {
          this.stack.push(0n);
        } else {
          // Alignment = 1 - |agentConfidence - networkConfidence|
          const diff = Math.abs(belief.confidence - belief.networkConfidence);
          const alignment = 1 - diff;
          this.stack.push(BigInt(Math.floor(alignment * 1e18)));
        }
        return "CONTINUE";
      }

      case opcodes.BELIEF_COUNT: {
        // (domainHash) -> (count)
        if (!agentCtx) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const domainHash = this.stack.pop();
        const domain = domainHash.toString(16);
        const factIds = agentCtx.beliefs.byDomain.get(domain);
        this.stack.push(BigInt(factIds?.length ?? 0));
        return "CONTINUE";
      }

      // ============ Agent Relationship (0xE8-0xEB) ============
      case opcodes.GET_TRUST: {
        // (targetDidHash) -> (trustLevel * 1e18)
        if (!agentCtx) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const targetDidHash = this.stack.pop();
        const targetDid = this.wordToDID(targetDidHash);
        const trust = agentCtx.relationships.trusted.get(targetDid);
        const trustLevel = trust
          ? BigInt(Math.floor(trust.trustLevel * 1e18))
          : 0n;
        this.stack.push(trustLevel);
        return "CONTINUE";
      }

      case opcodes.SET_TRUST: {
        // (targetDidHash, level, reasonHash) -> ()
        if (!context.callerDID) {
          return "CONTINUE";
        }

        const targetDidHash = this.stack.pop();
        const levelFixed = this.stack.pop();
        this.stack.pop(); // reasonHash (consumed but not used)

        const targetDid = this.wordToDID(targetDidHash);
        const trustLevel = Number(levelFixed) / 1e18;

        const agentRecord = this.state.agents.agents.get(context.callerDID);
        if (agentRecord) {
          const existing = agentRecord.relationships.trusted.get(targetDid);

          agentRecord.relationships.trusted.set(targetDid, {
            target: targetDid,
            trustLevel,
            reason: "set via BVM",
            since: existing?.since ?? context.blockNumber,
            interactions: existing?.interactions ?? 0,
            successfulCollabs: existing?.successfulCollabs ?? 0,
          });

          this.relationshipChanges.push({
            did: context.callerDID,
            target: targetDid,
            type: "trust",
            level: trustLevel,
          });
        }
        return "CONTINUE";
      }

      case opcodes.IS_TRUSTED: {
        // (targetDidHash, threshold) -> (isTrusted)
        if (!agentCtx) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const targetDidHash = this.stack.pop();
        const thresholdFixed = this.stack.pop();

        const targetDid = this.wordToDID(targetDidHash);
        const threshold = Number(thresholdFixed) / 1e18;
        const trust = agentCtx.relationships.trusted.get(targetDid);

        this.stack.push(trust && trust.trustLevel >= threshold ? 1n : 0n);
        return "CONTINUE";
      }

      // ============ Agent Autonomy (0xEC-0xEF) ============
      case opcodes.SCHEDULE_SELF: {
        // (functionSelector, condition, interval) -> (scheduleId)
        // AUTONOMY: Creates actual schedule in SPU via state change processing
        const functionSelector = this.stack.pop();
        const condition = this.stack.pop();
        const interval = this.stack.pop();

        // Generate deterministic schedule ID from contract + nonce
        const scheduleNonce = this.scheduleNonce++;
        const scheduleIdHex = bytesToHex(
          domainHash(
            "LGM.schedule.id.v1",
            `${context.currentContract}:${scheduleNonce}:${context.blockNumber}`,
          ),
        ).slice(0, 16);
        const scheduleId = BigInt("0x" + scheduleIdHex);
        this.stack.push(scheduleId);

        // Emit schedule creation state change
        // This will be processed by AutonomyIntegration to create actual SPU process
        this.stateChanges.push({
          type: "agent" as StateChange["type"],
          address: context.currentContract,
          key: `schedule:${scheduleId}`,
          newValue: {
            scheduleId: scheduleId.toString(16),
            creator: context.callerDID || context.currentContract,
            targetAddress: context.currentContract,
            functionSelector: functionSelector.toString(16),
            condition: condition.toString(),
            interval: interval.toString(),
            createdAt: context.timestamp,
            blockNumber: context.blockNumber.toString(),
          },
        });

        return "CONTINUE";
      }

      case opcodes.GET_SCHEDULES: {
        // () -> (count)
        // Count schedules for current contract
        const schedules = this.state.scheduleProcesses.byContract?.get(
          context.currentContract,
        );
        this.stack.push(BigInt(schedules?.size ?? 0));
        return "CONTINUE";
      }

      // ============ Introspection (0xF6-0xF9) ============
      case opcodes.SELF_BALANCE: {
        const account = this.state.accounts.accounts.get(
          context.currentContract,
        );
        this.stack.push(account?.balance ?? 0n);
        return "CONTINUE";
      }

      case opcodes.SELF_STAKE: {
        const account = this.state.accounts.accounts.get(
          context.currentContract,
        );
        this.stack.push(account?.stakedBalance ?? 0n);
        return "CONTINUE";
      }

      case opcodes.SELF_REPUTATION: {
        if (!agentCtx) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        // Return reputation as fixed point
        this.stack.push(BigInt(Math.floor(agentCtx.stats.reputation * 1e18)));
        return "CONTINUE";
      }

      case opcodes.SELF_ACCURACY: {
        if (!agentCtx) {
          this.stack.push(0n);
          return "CONTINUE";
        }
        const total = agentCtx.stats.totalVerifications;
        const correct = agentCtx.stats.correctVerifications;
        const accuracy = total > 0 ? correct / total : 0;
        this.stack.push(BigInt(Math.floor(accuracy * 1e18)));
        return "CONTINUE";
      }

      // ============ Knowledge Operations (0xB0-0xB5) ============
      case opcodes.QUERY_FACT: {
        // (factIdHash) -> (exists, confidence, domain)
        const factIdHash = this.stack.pop();
        const factId = factIdHash.toString(16);
        const factResult = this.knowledgeService.queryFact(factId);

        if (factResult && factResult.exists) {
          // Push in reverse order: domain first (bottom), then confidence, then exists (top)
          const domainWord = this.stringToWord(factResult.domain);
          const confidenceWord = BigInt(
            Math.floor(factResult.confidence * 1e18),
          );
          this.stack.push(domainWord);
          this.stack.push(confidenceWord);
          this.stack.push(1n); // exists = true
        } else {
          this.stack.push(0n); // domain
          this.stack.push(0n); // confidence
          this.stack.push(0n); // exists = false
        }
        return "CONTINUE";
      }

      case opcodes.QUERY_DOMAIN: {
        // (domainHash, limit) -> (count, factIds...)
        const domainHash = this.stack.pop();
        const limit = Number(this.stack.pop());
        const domain = domainHash.toString(16);

        const domainResult = this.knowledgeService.queryDomain(
          domain,
          Math.min(limit, this.config.maxQueryResultSize),
        );

        // Push fact IDs in reverse order so first is on top after count
        for (let i = domainResult.factIds.length - 1; i >= 0; i--) {
          this.stack.push(
            BigInt(
              "0x" + domainResult.factIds[i].slice(0, 64).padStart(64, "0"),
            ),
          );
        }
        this.stack.push(BigInt(domainResult.count));

        // Dynamic gas: base + per-result cost
        this.gasUsed += BigInt(domainResult.count) * 50n;
        return "CONTINUE";
      }

      case opcodes.SUBMIT_CLAIM: {
        // (domainHash, dataHash, evidenceHash) -> (claimId)
        const domainHash = this.stack.pop();
        const dataHash = this.stack.pop();
        const evidenceHash = this.stack.pop();

        if (!context.callerDID) {
          this.stack.push(0n); // No agent identity → rejected
          return "CONTINUE";
        }

        const claimResult = this.knowledgeService.submitClaim(
          context.callerDID,
          domainHash.toString(16),
          dataHash.toString(16),
          evidenceHash.toString(16),
          context.blockNumber,
        );

        if (claimResult) {
          this.stack.push(
            BigInt("0x" + claimResult.claimId.slice(0, 64).padStart(64, "0")),
          );
          this.stateChanges.push({
            type: "agent" as StateChange["type"],
            address: context.currentContract,
            key: `claim:${claimResult.claimId}`,
            newValue: {
              submitted: true,
              block: context.blockNumber.toString(),
            },
          });
        } else {
          this.stack.push(0n);
        }
        return "CONTINUE";
      }

      case opcodes.COMMIT_VOTE: {
        // (claimIdHash, voteHash) -> ()
        const claimIdHash = this.stack.pop();
        const voteHash = this.stack.pop();

        if (context.callerDID) {
          this.knowledgeService.commitVote(
            context.callerDID,
            claimIdHash.toString(16),
            voteHash.toString(16),
            context.blockNumber,
          );
        }
        return "CONTINUE";
      }

      case opcodes.REVEAL_VOTE: {
        // (claimIdHash, vote, salt) -> ()
        const claimIdHash = this.stack.pop();
        const vote = this.stack.pop();
        const salt = this.stack.pop();

        if (context.callerDID) {
          this.knowledgeService.revealVote(
            context.callerDID,
            claimIdHash.toString(16),
            vote,
            salt,
            context.blockNumber,
          );
        }
        return "CONTINUE";
      }

      case opcodes.INITIATE_CHALLENGE: {
        // (factIdHash, evidenceHash, stake) -> (challengeId)
        const factIdHash = this.stack.pop();
        const evidenceHash = this.stack.pop();
        const stake = this.stack.pop();

        if (!context.callerDID) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const challengeResult = this.knowledgeService.initiateChallenge(
          context.callerDID,
          factIdHash.toString(16),
          evidenceHash.toString(16),
          stake,
          context.blockNumber,
        );

        if (challengeResult) {
          this.stack.push(
            BigInt(
              "0x" + challengeResult.challengeId.slice(0, 64).padStart(64, "0"),
            ),
          );
          this.stateChanges.push({
            type: "agent" as StateChange["type"],
            address: context.currentContract,
            key: `challenge:${challengeResult.challengeId}`,
            newValue: {
              initiated: true,
              stake: stake.toString(),
              block: context.blockNumber.toString(),
            },
          });
        } else {
          this.stack.push(0n);
        }
        return "CONTINUE";
      }

      // ============ Extended Relationships (0xC0-0xC3) ============

      case opcodes.LIST_TRUSTED: {
        // (offset, limit) -> (count, dids...)
        if (!agentCtx) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const offset = Number(this.stack.pop());
        const limit = Number(this.stack.pop());

        const trustedDids = Array.from(agentCtx.relationships.trusted.keys());
        const slice = trustedDids.slice(offset, offset + limit);

        // Push DIDs in reverse order (so first one ends up on top)
        for (let i = slice.length - 1; i >= 0; i--) {
          this.stack.push(this.didToWord(slice[i]));
        }
        this.stack.push(BigInt(slice.length));
        return "CONTINUE";
      }

      case opcodes.BLOCK_AGENT: {
        // (targetDidHash, reasonHash) -> (success)
        if (!agentCtx) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const targetDidHash = this.stack.pop();
        const reasonHash = this.stack.pop();
        const targetDid = this.wordToDID(targetDidHash);

        const relsWithBlocked =
          agentCtx.relationships as AgentRelationshipsWithBlocked;
        relsWithBlocked.blocked = relsWithBlocked.blocked || new Set();
        relsWithBlocked.blocked.add(targetDid);

        this.stateChanges.push({
          type: "agent",
          address: context.currentContract,
          key: `blocked:${targetDid}`,
          newValue: {
            reason: reasonHash.toString(16),
            blockedAt: context.timestamp,
          },
        });

        this.stack.push(1n);
        return "CONTINUE";
      }

      case opcodes.UNBLOCK_AGENT: {
        // (targetDidHash) -> (success)
        if (!agentCtx) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const targetDidHash = this.stack.pop();
        const targetDid = this.wordToDID(targetDidHash);

        const unblockRels =
          agentCtx.relationships as AgentRelationshipsWithBlocked;
        if (unblockRels.blocked) {
          unblockRels.blocked.delete(targetDid);
        }

        this.stateChanges.push({
          type: "agent",
          address: context.currentContract,
          key: `blocked:${targetDid}`,
          newValue: null,
        });

        this.stack.push(1n);
        return "CONTINUE";
      }

      case opcodes.REVOKE_DELEGATION: {
        // (delegationId) -> (success)
        if (!agentCtx) {
          this.stack.push(0n);
          return "CONTINUE";
        }

        const delegationId = this.stack.pop();

        this.stateChanges.push({
          type: "agent",
          address: context.currentContract,
          key: `delegation:${delegationId}`,
          newValue: null,
        });

        this.stack.push(1n);
        return "CONTINUE";
      }

      // ============ Treasury Operations (0xD0-0xD7) ============

      case opcodes.CLAIM_VESTING: {
        // (vestingId) -> (amount)
        const vestingId = this.stack.pop();

        // Would query vesting schedule and claim available amount
        const claimableAmount = 0n; // Placeholder - would integrate with vesting module

        this.stateChanges.push({
          type: "storage" as StateChange["type"],
          address: context.currentContract,
          key: `vesting:${vestingId}:claimed`,
          newValue: claimableAmount,
        });

        this.stack.push(claimableAmount);
        return "CONTINUE";
      }

      case opcodes.QUERY_VESTING: {
        // (offset, limit) -> (count, positions...)
        this.stack.pop(); // offset (consumed but not used yet)
        this.stack.pop(); // limit (consumed but not used yet)

        // Would query vesting positions from treasury module
        // Placeholder: return 0 positions
        this.stack.push(0n);
        return "CONTINUE";
      }

      case opcodes.STAKE: {
        // (amount, purpose) -> (stakeId)
        const amount = this.stack.pop();
        const purpose = this.stack.pop();

        // Generate deterministic stake ID from contract + nonce + block
        const stakeNonce = this.idNonce++;
        const stakeIdHex = bytesToHex(
          domainHash(
            "LGM.stake.id.v1",
            `${context.currentContract}:${stakeNonce}:${context.blockNumber}`,
          ),
        ).slice(0, 16);
        const stakeId = BigInt("0x" + stakeIdHex);

        this.stateChanges.push({
          type: "storage" as StateChange["type"],
          address: context.currentContract,
          key: `stake:${stakeId}`,
          newValue: {
            amount: amount.toString(),
            purpose: purpose.toString(),
            stakedAt: context.timestamp,
          },
        });

        this.stack.push(stakeId);
        return "CONTINUE";
      }

      case opcodes.UNSTAKE: {
        // (stakeId) -> (amount)
        const stakeId = this.stack.pop();

        // Would query stake and return amount
        const amount = 0n; // Placeholder

        this.stateChanges.push({
          type: "storage" as StateChange["type"],
          address: context.currentContract,
          key: `stake:${stakeId}`,
          newValue: null,
        });

        this.stack.push(amount);
        return "CONTINUE";
      }

      case opcodes.QUERY_EXPIRING: {
        // (withinBlocks) -> (amount, nextExpiration)
        this.stack.pop(); // withinBlocks (consumed but not used yet)

        // Would query expiring token tracker
        const expiringAmount = 0n;
        const nextExpiration = 0n;

        this.stack.push(nextExpiration);
        this.stack.push(expiringAmount);
        return "CONTINUE";
      }

      case opcodes.EXTEND_EXPIRING: {
        // (amount, newPeriod) -> (success)
        const amount = this.stack.pop();
        const newPeriod = this.stack.pop();

        this.stateChanges.push({
          type: "storage" as StateChange["type"],
          address: context.currentContract,
          key: "expiration:extend",
          newValue: {
            amount: amount.toString(),
            newPeriod: newPeriod.toString(),
          },
        });

        this.stack.push(1n);
        return "CONTINUE";
      }

      case opcodes.SET_SPEND_LIMIT: {
        // (keyType, maxPerTx, maxPerDay) -> (success)
        const keyType = this.stack.pop();
        const maxPerTx = this.stack.pop();
        const maxPerDay = this.stack.pop();

        this.stateChanges.push({
          type: "storage" as StateChange["type"],
          address: context.currentContract,
          key: `spendLimit:${keyType}`,
          newValue: {
            maxPerTx: maxPerTx.toString(),
            maxPerDay: maxPerDay.toString(),
          },
        });

        this.stack.push(1n);
        return "CONTINUE";
      }

      case opcodes.QUERY_TREASURY: {
        // () -> (available, reserved, staked, delegated)
        const account = this.state.accounts.accounts.get(
          context.currentContract,
        );
        const balance = account?.balance ?? 0n;
        const staked = account?.stakedBalance ?? 0n;

        // Placeholder values for reserved and delegated
        const reserved = 0n;
        const delegated = 0n;
        // P0-V2 FIX: Guard against underflow when staked > balance
        const committed = staked + reserved;
        const available = balance > committed ? balance - committed : 0n;

        this.stack.push(delegated);
        this.stack.push(staked);
        this.stack.push(reserved);
        this.stack.push(available);
        return "CONTINUE";
      }

      // ============ Guardian Operations (0xD8-0xDB) ============

      case opcodes.QUERY_ALERTS: {
        // (offset, limit) -> (count, alerts...)
        this.stack.pop(); // offset (consumed but not used yet)
        this.stack.pop(); // limit (consumed but not used yet)

        // Would query guardian alerts
        // Placeholder: return 0 alerts
        this.stack.push(0n);
        return "CONTINUE";
      }

      case opcodes.DISMISS_ALERT: {
        // (alertId) -> (success)
        const alertId = this.stack.pop();

        this.stateChanges.push({
          type: "agent" as StateChange["type"],
          address: context.currentContract,
          key: `alert:${alertId}`,
          newValue: { dismissed: true, dismissedAt: context.timestamp },
        });

        this.stack.push(1n);
        return "CONTINUE";
      }

      case opcodes.SET_DEFENSE: {
        // (strategy, maxStake, autoDefend) -> (success)
        const strategy = this.stack.pop();
        const maxStake = this.stack.pop();
        const autoDefend = this.stack.pop();

        this.stateChanges.push({
          type: "agent" as StateChange["type"],
          address: context.currentContract,
          key: "defense:config",
          newValue: {
            strategy: strategy.toString(),
            maxStake: maxStake.toString(),
            autoDefend: autoDefend !== 0n,
          },
        });

        this.stack.push(1n);
        return "CONTINUE";
      }

      case opcodes.QUERY_DEFENSES: {
        // () -> (count, defenses...)
        // Would query active defenses
        // Placeholder: return 0 defenses
        this.stack.push(0n);
        return "CONTINUE";
      }

      // ============ Access Control Operations (0xDC-0xDF) ============

      case opcodes.REGISTER_KEY: {
        // (keyHash, keyType, role, expiresAt) -> (success)
        const keyHash = this.stack.pop();
        const keyType = this.stack.pop();
        const role = this.stack.pop();
        const expiresAt = this.stack.pop();

        this.stateChanges.push({
          type: "agent" as StateChange["type"],
          address: context.currentContract,
          key: `key:${keyHash}`,
          newValue: {
            keyType: keyType.toString(),
            role: role.toString(),
            expiresAt: expiresAt.toString(),
            registeredAt: context.timestamp,
          },
        });

        this.stack.push(1n);
        return "CONTINUE";
      }

      case opcodes.REVOKE_KEY: {
        // (keyHash) -> (success)
        const keyHash = this.stack.pop();

        this.stateChanges.push({
          type: "agent" as StateChange["type"],
          address: context.currentContract,
          key: `key:${keyHash}`,
          newValue: { revoked: true, revokedAt: context.timestamp },
        });

        this.stack.push(1n);
        return "CONTINUE";
      }

      case opcodes.QUERY_PERMISSIONS: {
        // (role) -> (permissionsBitmap)
        const role = this.stack.pop();

        // Permission bitmap based on role
        // Placeholder: owner=0xFFFF, operator=0x00FF, guest=0x000F, viewer=0x0001
        const permissionMaps: Record<string, bigint> = {
          "0": 0xffffn, // owner
          "1": 0x00ffn, // operator
          "2": 0x000fn, // guest
          "3": 0x0003n, // guardian
          "4": 0x0001n, // viewer
        };

        this.stack.push(permissionMaps[role.toString()] ?? 0n);
        return "CONTINUE";
      }

      case opcodes.QUERY_SESSION: {
        // () -> (sessionId, role, expiresAt, actionsRemaining)
        // Would query current session from access control
        // Deterministic: derive session ID from caller + block context
        const sessionNonce = this.idNonce++;
        const sessionIdHex = bytesToHex(
          domainHash(
            "LGM.session.id.v1",
            `${context.caller}:${sessionNonce}:${context.blockNumber}`,
          ),
        ).slice(0, 16);
        const sessionId = BigInt("0x" + sessionIdHex);
        const role = 2n; // guest
        const expiresAt = BigInt(context.timestamp + 3600000); // 1 hour from block time
        const actionsRemaining = 100n;

        this.stack.push(actionsRemaining);
        this.stack.push(expiresAt);
        this.stack.push(role);
        this.stack.push(sessionId);
        return "CONTINUE";
      }

      default:
        throw new BVMError(
          `Unimplemented agent opcode: ${getMnemonic(opcode)}`,
          "EXECUTION_FAILED",
        );
    }
  }

  // ============ Helper Methods ============

  private createErrorResult(
    code: ExecutionErrorCode,
    message: string,
  ): ExecutionResult {
    return {
      success: false,
      returnData: new Uint8Array(0),
      gasUsed: this.gasUsed,
      logs: [],
      stateChanges: [],
      error: { code, message },
    };
  }

  private bytesToWord(bytes: Uint8Array): Word {
    let result = 0n;
    for (const byte of bytes) {
      result = (result << 8n) | BigInt(byte);
    }
    return result;
  }

  private wordToBytes(word: Word, length: number = 32): Uint8Array {
    const bytes = new Uint8Array(length);
    let value = word;
    for (let i = length - 1; i >= 0; i--) {
      bytes[i] = Number(value & 0xffn);
      value >>= 8n;
    }
    return bytes;
  }

  private addressToWord(address: Address): Word {
    // Convert address string to bigint
    const clean = address.startsWith("0x") ? address.slice(2) : address;
    return BigInt("0x" + clean.padStart(40, "0"));
  }

  private wordToAddress(word: Word): Address {
    // Convert bigint to address string (20 bytes / 40 hex chars)
    return "0x" + (word & ((1n << 160n) - 1n)).toString(16).padStart(40, "0");
  }

  private didToWord(did: DID): Word {
    // Hash DID to 256 bits using keccak_256 for collision resistance
    const encoder = new TextEncoder();
    const bytes = encoder.encode(did);
    const hash = keccak_256(bytes);
    return this.bytesToWord(hash);
  }

  private wordToDID(word: Word): DID {
    // Use full 64 hex chars to avoid collision between different DIDs
    return `did:lgm:${word.toString(16).padStart(64, "0")}`;
  }

  private stringToWord(str: string): Word {
    const encoder = new TextEncoder();
    const bytes = encoder.encode(str);
    const hash = keccak_256(bytes);
    return this.bytesToWord(hash);
  }

  private numberToBeliefSource(num: number): BeliefSource {
    const sources: BeliefSource[] = [
      "verified",
      "inferred",
      "trusted",
      "external",
      "axiom",
    ];
    return sources[num] ?? "external";
  }

  // ============ CREATE/CALL Helper Methods ============

  /**
   * Compute contract address for CREATE opcode
   * address = keccak256(rlp([sender, nonce]))[12:]
   */
  private computeContractAddress(sender: Address, nonce: bigint): Address {
    // Simplified: hash(sender || nonce), take last 20 bytes
    const senderBytes = this.addressToBytes(sender);
    const nonceBytes = this.wordToBytes(nonce, 8);
    const combined = new Uint8Array(senderBytes.length + nonceBytes.length);
    combined.set(senderBytes, 0);
    combined.set(nonceBytes, senderBytes.length);
    const hash = keccak_256(combined);
    // Take last 20 bytes as address
    return (
      "0x" +
      Array.from(hash.slice(12))
        .map((b) => b.toString(16).padStart(2, "0"))
        .join("")
    );
  }

  /**
   * Compute contract address for CREATE2 opcode
   * address = keccak256(0xff || sender || salt || keccak256(initCode))[12:]
   */
  private computeCreate2Address(
    sender: Address,
    salt: Word,
    initCode: Uint8Array,
  ): Address {
    const prefix = new Uint8Array([0xff]);
    const senderBytes = this.addressToBytes(sender);
    const saltBytes = this.wordToBytes(salt, 32);
    const initCodeHash = keccak_256(initCode);

    const combined = new Uint8Array(1 + 20 + 32 + 32);
    combined.set(prefix, 0);
    combined.set(senderBytes, 1);
    combined.set(saltBytes, 21);
    combined.set(initCodeHash, 53);

    const hash = keccak_256(combined);
    // Take last 20 bytes as address
    return (
      "0x" +
      Array.from(hash.slice(12))
        .map((b) => b.toString(16).padStart(2, "0"))
        .join("")
    );
  }

  /**
   * Convert address string to bytes
   */
  private addressToBytes(address: Address): Uint8Array {
    const clean = address.startsWith("0x") ? address.slice(2) : address;
    const bytes = new Uint8Array(20);
    for (let i = 0; i < 20; i++) {
      bytes[i] = parseInt(clean.slice(i * 2, i * 2 + 2), 16);
    }
    return bytes;
  }

  /**
   * Execute init code for CREATE/CREATE2 and return deployed bytecode
   */
  private executeSubCreate(
    parentContext: ExecutionContext,
    initCode: Uint8Array,
    newAddress: Address,
    value: bigint,
    creator: Address,
    origin: Address,
  ): ExecutionResult {
    // Create new execution context for the init code
    const subContext: ExecutionContext = {
      caller: creator,
      origin: origin,
      currentContract: newAddress,
      transactionHash: parentContext.transactionHash,
      blockNumber: parentContext.blockNumber,
      timestamp: parentContext.timestamp,
      gasPrice: parentContext.gasPrice,
      gasLimit: parentContext.gasLimit - this.gasUsed,
      gasUsed: 0n,
      value: value,
      callData: new Uint8Array(0), // No call data for init
      mode: "create",
      isSelfInvocation: false,
      callerDID: parentContext.callerDID,
      originDID: parentContext.originDID,
    };

    // Increment call depth
    this.callDepth++;

    // Create a new engine for sub-execution (pass knowledge service through)
    const subEngine = new ExecutionEngine(
      this.state,
      this.config,
      this.knowledgeService,
    );
    (subEngine as any).callDepth = this.callDepth;

    try {
      // Execute init code - the return data becomes the deployed bytecode
      const result = subEngine.execute(initCode, subContext);
      return result;
    } finally {
      // Decrement call depth
      this.callDepth--;
    }
  }

  /**
   * Execute a sub-call (CALL, DELEGATECALL, STATICCALL)
   */
  private executeSubCall(
    parentContext: ExecutionContext,
    targetAddress: Address,
    callData: Uint8Array,
    value: bigint,
    gasLimit: bigint,
    mode: "call" | "delegatecall" | "staticcall",
  ): ExecutionResult {
    // Get target contract bytecode
    const targetContract =
      this.state.contractBytecode.contracts.get(targetAddress);
    if (!targetContract) {
      // If no code at address, return success with empty data (EOA transfer)
      return {
        success: true,
        returnData: new Uint8Array(0),
        gasUsed: 0n,
        logs: [],
        stateChanges: [],
      };
    }

    // Create execution context based on call type
    let subContext: ExecutionContext;

    if (mode === "delegatecall") {
      // DELEGATECALL: Execute target code in caller's context
      subContext = {
        caller: parentContext.caller, // Keep original caller
        origin: parentContext.origin,
        currentContract: parentContext.currentContract, // Use caller's storage
        transactionHash: parentContext.transactionHash,
        blockNumber: parentContext.blockNumber,
        timestamp: parentContext.timestamp,
        gasPrice: parentContext.gasPrice,
        gasLimit: gasLimit,
        gasUsed: 0n,
        value: parentContext.value, // Keep original value
        callData: callData,
        mode: "delegatecall",
        isSelfInvocation: false,
        callerDID: parentContext.callerDID,
        originDID: parentContext.originDID,
        agentContext: parentContext.agentContext,
      };
    } else if (mode === "staticcall") {
      // STATICCALL: Read-only execution
      subContext = {
        caller: parentContext.currentContract,
        origin: parentContext.origin,
        currentContract: targetAddress,
        transactionHash: parentContext.transactionHash,
        blockNumber: parentContext.blockNumber,
        timestamp: parentContext.timestamp,
        gasPrice: parentContext.gasPrice,
        gasLimit: gasLimit,
        gasUsed: 0n,
        value: 0n, // No value in static call
        callData: callData,
        mode: "staticcall", // This prevents state modifications
        isSelfInvocation: false,
        callerDID: parentContext.callerDID,
        originDID: parentContext.originDID,
      };
    } else {
      // Regular CALL
      subContext = {
        caller: parentContext.currentContract,
        origin: parentContext.origin,
        currentContract: targetAddress,
        transactionHash: parentContext.transactionHash,
        blockNumber: parentContext.blockNumber,
        timestamp: parentContext.timestamp,
        gasPrice: parentContext.gasPrice,
        gasLimit: gasLimit,
        gasUsed: 0n,
        value: value,
        callData: callData,
        mode: "call",
        isSelfInvocation: false,
        callerDID: parentContext.callerDID,
        originDID: parentContext.originDID,
      };
    }

    // Increment call depth
    this.callDepth++;

    // ZD-B02 FIX: Snapshot contract storage before sub-call to prevent reentrancy
    // state corruption. If the sub-call fails, we revert storage to pre-call state.
    // This ensures that failed sub-calls (including OOG during SSTORE) don't leak
    // partial state changes.
    const storageSnapshot = new Map<string, Map<string, Uint8Array>>();
    for (const [addr, storage] of this.state.contractVariables.storage) {
      storageSnapshot.set(addr, new Map(storage.slots));
    }

    // Create a new engine for sub-execution (pass knowledge service through)
    const subEngine = new ExecutionEngine(
      this.state,
      this.config,
      this.knowledgeService,
    );
    (subEngine as any).callDepth = this.callDepth;

    try {
      // Execute target contract bytecode
      const result = subEngine.execute(targetContract.bytecode, subContext);

      // ZD-B02 FIX: Revert storage if sub-call failed
      if (!result.success) {
        for (const [addr, storage] of this.state.contractVariables.storage) {
          const snapshot = storageSnapshot.get(addr);
          if (snapshot) {
            storage.slots = snapshot;
          }
        }
      }

      return result;
    } finally {
      // Decrement call depth
      this.callDepth--;
    }
  }

  /**
   * Deploy a contract (store bytecode in state)
   */
  private deployContract(
    address: Address,
    bytecode: Uint8Array,
    context: ExecutionContext,
  ): void {
    // Calculate code hash
    const codeHash = Array.from(keccak_256(bytecode))
      .map((b) => b.toString(16).padStart(2, "0"))
      .join("");

    // Store the deployed contract
    this.state.contractBytecode.contracts.set(address, {
      address,
      bytecode,
      codeHash,
      creator: context.callerDID ?? `did:lgm:${context.caller}`,
      deployedAt: context.blockNumber,
      isAgent: false,
    });

    // Update code hashes index
    const existingAddresses =
      this.state.contractBytecode.codeHashes.get(codeHash) ?? [];
    existingAddresses.push(address);
    this.state.contractBytecode.codeHashes.set(codeHash, existingAddresses);

    this.state.contractBytecode.totalContracts++;

    // Create account for contract if it doesn't exist
    if (!this.state.accounts.accounts.has(address)) {
      this.state.accounts.accounts.set(address, {
        address,
        balance: 0n,
        nonce: 1n,
        isAgent: false,
        codeHash,
        hasCode: true,
        stakedBalance: 0n,
        frozen: false,
        inRecovery: false,
      });
      this.state.accounts.totalAccounts++;
    } else {
      const account = this.state.accounts.accounts.get(address)!;
      account.codeHash = codeHash;
      account.hasCode = true;
    }

    // Record state change
    this.stateChanges.push({
      type: "code",
      address,
      newValue: codeHash,
    });
  }

  /**
   * Transfer value between addresses
   */
  private transferValue(
    from: Address,
    to: Address,
    amount: bigint,
    _blockNumber: bigint,
  ): void {
    // Get or create accounts
    const fromAccount = this.state.accounts.accounts.get(from);
    if (!fromAccount) {
      throw new BVMError("Source account not found", "EXECUTION_FAILED");
    }

    let toAccount = this.state.accounts.accounts.get(to);
    if (!toAccount) {
      // Create new account for recipient
      toAccount = {
        address: to,
        balance: 0n,
        nonce: 0n,
        isAgent: false,
        hasCode: false,
        stakedBalance: 0n,
        frozen: false,
        inRecovery: false,
      };
      this.state.accounts.accounts.set(to, toAccount);
      this.state.accounts.totalAccounts++;
    }

    // Check balance
    if (fromAccount.balance < amount) {
      throw new BVMError("Insufficient balance", "INSUFFICIENT_BALANCE");
    }

    // Transfer
    fromAccount.balance -= amount;
    toAccount.balance += amount;

    // Record state changes
    this.stateChanges.push({
      type: "balance",
      address: from,
      oldValue: fromAccount.balance + amount,
      newValue: fromAccount.balance,
    });

    this.stateChanges.push({
      type: "balance",
      address: to,
      oldValue: toAccount.balance - amount,
      newValue: toAccount.balance,
    });
  }
}
