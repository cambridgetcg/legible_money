/**
 * BVM Opcodes
 *
 * Complete opcode definitions for the Blockchain Virtual Machine.
 * Includes standard EVM-compatible opcodes plus agent-specific operations.
 *
 * Opcode Ranges:
 * - 0x00-0x0F: Stop and Arithmetic
 * - 0x10-0x1F: Comparison and Bitwise
 * - 0x20-0x2F: SHA3
 * - 0x30-0x3F: Environmental
 * - 0x40-0x4F: Block Information
 * - 0x50-0x5F: Stack, Memory, Storage
 * - 0x60-0x7F: Push
 * - 0x80-0x8F: Dup
 * - 0x90-0x9F: Swap
 * - 0xA0-0xAF: Log
 * - 0xB0-0xB5: Knowledge Operations
 * - 0xC0-0xC3: Extended Relationships
 * - 0xD0-0xD7: Treasury Operations
 * - 0xD8-0xDB: Guardian Operations
 * - 0xDC-0xDF: Access Control Operations
 * - 0xE0-0xEF: Agent Core Operations
 * - 0xF0-0xFF: System Operations
 */

import type { OpcodeDefinition } from "./types.js";
import { DEFAULT_GAS_COSTS } from "./types.js";

// ============ Opcode Constants ============

// Stop and Arithmetic (0x00-0x0F)
export const STOP = 0x00;
export const ADD = 0x01;
export const MUL = 0x02;
export const SUB = 0x03;
export const DIV = 0x04;
export const SDIV = 0x05;
export const MOD = 0x06;
export const SMOD = 0x07;
export const ADDMOD = 0x08;
export const MULMOD = 0x09;
export const EXP = 0x0a;
export const SIGNEXTEND = 0x0b;

// Comparison and Bitwise (0x10-0x1F)
export const LT = 0x10;
export const GT = 0x11;
export const SLT = 0x12;
export const SGT = 0x13;
export const EQ = 0x14;
export const ISZERO = 0x15;
export const AND = 0x16;
export const OR = 0x17;
export const XOR = 0x18;
export const NOT = 0x19;
export const BYTE = 0x1a;
export const SHL = 0x1b;
export const SHR = 0x1c;
export const SAR = 0x1d;

// SHA3 (0x20)
export const SHA3 = 0x20;

// Environmental (0x30-0x3F)
export const ADDRESS = 0x30;
export const BALANCE = 0x31;
export const ORIGIN = 0x32;
export const CALLER = 0x33;
export const CALLVALUE = 0x34;
export const CALLDATALOAD = 0x35;
export const CALLDATASIZE = 0x36;
export const CALLDATACOPY = 0x37;
export const CODESIZE = 0x38;
export const CODECOPY = 0x39;
export const GASPRICE = 0x3a;
export const EXTCODESIZE = 0x3b;
export const EXTCODECOPY = 0x3c;
export const RETURNDATASIZE = 0x3d;
export const RETURNDATACOPY = 0x3e;
export const EXTCODEHASH = 0x3f;

// Block Information (0x40-0x4F)
export const BLOCKHASH = 0x40;
export const COINBASE = 0x41;
export const TIMESTAMP = 0x42;
export const NUMBER = 0x43;
export const PREVRANDAO = 0x44;
export const GASLIMIT = 0x45;
export const CHAINID = 0x46;
export const SELFBALANCE = 0x47;
export const BASEFEE = 0x48;

// Stack, Memory, Storage (0x50-0x5F)
export const POP = 0x50;
export const MLOAD = 0x51;
export const MSTORE = 0x52;
export const MSTORE8 = 0x53;
export const SLOAD = 0x54;
export const SSTORE = 0x55;
export const JUMP = 0x56;
export const JUMPI = 0x57;
export const PC = 0x58;
export const MSIZE = 0x59;
export const GAS = 0x5a;
export const JUMPDEST = 0x5b;

// Push (0x60-0x7F)
export const PUSH0 = 0x5f;
export const PUSH1 = 0x60;
export const PUSH2 = 0x61;
export const PUSH32 = 0x7f;

// Dup (0x80-0x8F)
export const DUP1 = 0x80;
export const DUP16 = 0x8f;

// Swap (0x90-0x9F)
export const SWAP1 = 0x90;
export const SWAP16 = 0x9f;

// Log (0xA0-0xA4)
export const LOG0 = 0xa0;
export const LOG1 = 0xa1;
export const LOG2 = 0xa2;
export const LOG3 = 0xa3;
export const LOG4 = 0xa4;

// ============ AGENT IDENTITY OPCODES (0xE0-0xE3) ============

/** Push caller's DID to stack (or 0 if not agent) */
export const AGENT_DID = 0xe0;

/** Push agent's home contract address to stack */
export const AGENT_HOME = 0xe1;

/** Verify another agent's identity: (did) -> (isValid) */
export const VERIFY_IDENTITY = 0xe2;

/** Create identity attestation: (targetDid, claim, confidence) -> (attestationId) */
export const ATTEST_IDENTITY = 0xe3;

// ============ AGENT BELIEF OPCODES (0xE4-0xE7) ============

/** Get agent's belief confidence for fact: (factId) -> (confidence) */
export const GET_BELIEF = 0xe4;

/** Set agent's belief: (factId, confidence, source) -> () */
export const SET_BELIEF = 0xe5;

/** Compare agent belief with network: (factId) -> (alignment) */
export const COMPARE_BELIEF = 0xe6;

/** Count beliefs in domain: (domain) -> (count) */
export const BELIEF_COUNT = 0xe7;

// ============ AGENT RELATIONSHIP OPCODES (0xE8-0xEB) ============

/** Get trust level for agent: (targetDid) -> (trustLevel) */
export const GET_TRUST = 0xe8;

/** Set trust relationship: (targetDid, level, reason) -> () */
export const SET_TRUST = 0xe9;

/** Check if agent is trusted above threshold: (targetDid, threshold) -> (isTrusted) */
export const IS_TRUSTED = 0xea;

/** Delegate authority to another agent: (targetDid, scope, duration) -> (delegationId) */
export const DELEGATE = 0xeb;

// ============ AGENT AUTONOMY OPCODES (0xEC-0xEF) ============

/** Create self-invocation schedule: (function, condition, interval) -> (scheduleId) */
export const SCHEDULE_SELF = 0xec;

/** Cancel a schedule: (scheduleId) -> (success) */
export const CANCEL_SCHEDULE = 0xed;

/** Modify schedule parameters: (scheduleId, newCondition) -> (success) */
export const MODIFY_SCHEDULE = 0xee;

/** Get active schedules count: () -> (count) */
export const GET_SCHEDULES = 0xef;

// ============ KNOWLEDGE OPCODES (0xB0-0xB5) ============
// Note: Moved to reserved range to avoid conflict with EVM system opcodes

/** Query fact from knowledge base: (factId) -> (exists, confidence, data) */
export const QUERY_FACT = 0xb0;

/** Query facts in domain: (domain, limit) -> (count, factIds...) */
export const QUERY_DOMAIN = 0xb1;

/** Submit knowledge claim: (domain, data, evidence) -> (claimId) */
export const SUBMIT_CLAIM = 0xb2;

/** Commit verification vote: (claimId, voteHash) -> () */
export const COMMIT_VOTE = 0xb3;

/** Reveal verification vote: (claimId, vote, salt) -> () */
export const REVEAL_VOTE = 0xb4;

/** Initiate challenge: (factId, evidence, stake) -> (challengeId) */
export const INITIATE_CHALLENGE = 0xb5;

// ============ EXTENDED RELATIONSHIP OPCODES (0xC0-0xC3) ============

/** List all trusted agents: (offset, limit) -> (count, dids...) */
export const LIST_TRUSTED = 0xc0;

/** Block an agent from interactions: (targetDid, reason) -> (success) */
export const BLOCK_AGENT = 0xc1;

/** Unblock a previously blocked agent: (targetDid) -> (success) */
export const UNBLOCK_AGENT = 0xc2;

/** Revoke a delegation: (delegationId) -> (success) */
export const REVOKE_DELEGATION = 0xc3;

// ============ TREASURY OPCODES (0xD0-0xD7) ============

/** Claim available vested tokens: (vestingId) -> (amount) */
export const CLAIM_VESTING = 0xd0;

/** Query vesting positions: (offset, limit) -> (count, positions...) */
export const QUERY_VESTING = 0xd1;

/** Stake tokens: (amount, purpose) -> (stakeId) */
export const STAKE = 0xd2;

/** Unstake tokens: (stakeId) -> (amount) */
export const UNSTAKE = 0xd3;

/** Query expiring token amounts: (withinBlocks) -> (amount, nextExpiration) */
export const QUERY_EXPIRING = 0xd4;

/** Extend expiring tokens: (amount, newPeriod) -> (success) */
export const EXTEND_EXPIRING = 0xd5;

/** Set spending limit for key type: (keyType, maxPerTx, maxPerDay) -> (success) */
export const SET_SPEND_LIMIT = 0xd6;

/** Query full treasury state: () -> (available, reserved, staked, delegated) */
export const QUERY_TREASURY = 0xd7;

// ============ GUARDIAN OPCODES (0xD8-0xDB) ============

/** Query pending alerts: (offset, limit) -> (count, alerts...) */
export const QUERY_ALERTS = 0xd8;

/** Dismiss an alert: (alertId) -> (success) */
export const DISMISS_ALERT = 0xd9;

/** Set defense configuration: (strategy, maxStake, autoDefend) -> (success) */
export const SET_DEFENSE = 0xda;

/** Query active defenses: () -> (count, defenses...) */
export const QUERY_DEFENSES = 0xdb;

// ============ ACCESS CONTROL OPCODES (0xDC-0xDF) ============

/** Register a new key: (keyHash, keyType, role, expiresAt) -> (success) */
export const REGISTER_KEY = 0xdc;

/** Revoke a key: (keyHash) -> (success) */
export const REVOKE_KEY = 0xdd;

/** Query permissions for a role: (role) -> (permissionsBitmap) */
export const QUERY_PERMISSIONS = 0xde;

/** Query current session state: () -> (sessionId, role, expiresAt, actionsRemaining) */
export const QUERY_SESSION = 0xdf;

// ============ INTROSPECTION OPCODES (0xF6-0xF9) ============

/** Get own balance (same as SELFBALANCE but DID-aware) */
export const SELF_BALANCE = 0xf6;

/** Get own staked amount: () -> (staked) */
export const SELF_STAKE = 0xf7;

/** Get own reputation score: () -> (reputation) */
export const SELF_REPUTATION = 0xf8;

/** Get own verification accuracy: () -> (accuracy) */
export const SELF_ACCURACY = 0xf9;

// ============ SYSTEM OPCODES (0xFA-0xFF) ============
// Note: Using EVM-compatible opcode numbers for system operations

/** Create new contract: (value, offset, length) -> (address) */
export const CREATE = 0xf0;

/** Message call: (gas, addr, value, argsOffset, argsSize, retOffset, retSize) -> (success) */
export const CALL = 0xf1;

/** Message call with code at another account: (gas, addr, value, argsOffset, argsSize, retOffset, retSize) -> (success) */
export const CALLCODE = 0xf2;

/** Return from execution: (offset, length) -> () */
export const RETURN = 0xf3;

/** Delegate call (uses caller's storage): (gas, addr, argsOffset, argsSize, retOffset, retSize) -> (success) */
export const DELEGATECALL = 0xf4;

/** Create with deterministic address: (value, offset, length, salt) -> (address) */
export const CREATE2 = 0xf5;

/** Static call (read-only): (gas, addr, argsOffset, argsSize, retOffset, retSize) -> (success) */
export const STATICCALL = 0xfa;

/** Revert execution: (offset, length) -> () */
export const REVERT = 0xfd;

/** Invalid opcode */
export const INVALID = 0xfe;

/** Destroy contract and send funds: (address) -> () */
export const SELFDESTRUCT = 0xff;

// ============ Opcode Definitions ============

const g = DEFAULT_GAS_COSTS;

export const OPCODE_DEFINITIONS: Map<number, OpcodeDefinition> = new Map([
  // Stop and Arithmetic
  [
    STOP,
    {
      code: STOP,
      mnemonic: "STOP",
      category: 0x00,
      stackInputs: 0,
      stackOutputs: 0,
      gasCost: 0n,
      description: "Halt execution",
    },
  ],
  [
    ADD,
    {
      code: ADD,
      mnemonic: "ADD",
      category: 0x00,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Addition",
    },
  ],
  [
    MUL,
    {
      code: MUL,
      mnemonic: "MUL",
      category: 0x00,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.low,
      description: "Multiplication",
    },
  ],
  [
    SUB,
    {
      code: SUB,
      mnemonic: "SUB",
      category: 0x00,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Subtraction",
    },
  ],
  [
    DIV,
    {
      code: DIV,
      mnemonic: "DIV",
      category: 0x00,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.low,
      description: "Division",
    },
  ],
  [
    MOD,
    {
      code: MOD,
      mnemonic: "MOD",
      category: 0x00,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.low,
      description: "Modulo",
    },
  ],

  // Comparison
  [
    LT,
    {
      code: LT,
      mnemonic: "LT",
      category: 0x10,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Less than",
    },
  ],
  [
    GT,
    {
      code: GT,
      mnemonic: "GT",
      category: 0x10,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Greater than",
    },
  ],
  [
    EQ,
    {
      code: EQ,
      mnemonic: "EQ",
      category: 0x10,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Equality",
    },
  ],
  [
    ISZERO,
    {
      code: ISZERO,
      mnemonic: "ISZERO",
      category: 0x10,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Is zero",
    },
  ],

  // Bitwise
  [
    AND,
    {
      code: AND,
      mnemonic: "AND",
      category: 0x16,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Bitwise AND",
    },
  ],
  [
    OR,
    {
      code: OR,
      mnemonic: "OR",
      category: 0x16,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Bitwise OR",
    },
  ],
  [
    XOR,
    {
      code: XOR,
      mnemonic: "XOR",
      category: 0x16,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Bitwise XOR",
    },
  ],
  [
    NOT,
    {
      code: NOT,
      mnemonic: "NOT",
      category: 0x16,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Bitwise NOT",
    },
  ],

  // SHA3
  [
    SHA3,
    {
      code: SHA3,
      mnemonic: "SHA3",
      category: 0x20,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: "dynamic",
      description: "Keccak-256 hash",
    },
  ],

  // Environment
  [
    ADDRESS,
    {
      code: ADDRESS,
      mnemonic: "ADDRESS",
      category: 0x30,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Current address",
    },
  ],
  [
    BALANCE,
    {
      code: BALANCE,
      mnemonic: "BALANCE",
      category: 0x30,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: "dynamic",
      description: "Address balance",
    },
  ],
  [
    ORIGIN,
    {
      code: ORIGIN,
      mnemonic: "ORIGIN",
      category: 0x30,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Transaction origin",
    },
  ],
  [
    CALLER,
    {
      code: CALLER,
      mnemonic: "CALLER",
      category: 0x30,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Caller address",
    },
  ],
  [
    CALLVALUE,
    {
      code: CALLVALUE,
      mnemonic: "CALLVALUE",
      category: 0x30,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Call value",
    },
  ],
  [
    CALLDATASIZE,
    {
      code: CALLDATASIZE,
      mnemonic: "CALLDATASIZE",
      category: 0x30,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Calldata size",
    },
  ],

  // Block
  [
    BLOCKHASH,
    {
      code: BLOCKHASH,
      mnemonic: "BLOCKHASH",
      category: 0x40,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 20n,
      description: "Block hash",
    },
  ],
  [
    TIMESTAMP,
    {
      code: TIMESTAMP,
      mnemonic: "TIMESTAMP",
      category: 0x40,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Block timestamp",
    },
  ],
  [
    NUMBER,
    {
      code: NUMBER,
      mnemonic: "NUMBER",
      category: 0x40,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Block number",
    },
  ],
  [
    GASLIMIT,
    {
      code: GASLIMIT,
      mnemonic: "GASLIMIT",
      category: 0x40,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Gas limit",
    },
  ],
  [
    CHAINID,
    {
      code: CHAINID,
      mnemonic: "CHAINID",
      category: 0x40,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Chain ID",
    },
  ],

  // Stack, Memory, Storage
  [
    POP,
    {
      code: POP,
      mnemonic: "POP",
      category: 0x50,
      stackInputs: 1,
      stackOutputs: 0,
      gasCost: g.base,
      description: "Pop from stack",
    },
  ],
  [
    MLOAD,
    {
      code: MLOAD,
      mnemonic: "MLOAD",
      category: 0x50,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: g.veryLow,
      description: "Memory load",
    },
  ],
  [
    MSTORE,
    {
      code: MSTORE,
      mnemonic: "MSTORE",
      category: 0x50,
      stackInputs: 2,
      stackOutputs: 0,
      gasCost: g.veryLow,
      description: "Memory store",
    },
  ],
  [
    SLOAD,
    {
      code: SLOAD,
      mnemonic: "SLOAD",
      category: 0x54,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: g.sload,
      description: "Storage load",
    },
  ],
  [
    SSTORE,
    {
      code: SSTORE,
      mnemonic: "SSTORE",
      category: 0x54,
      stackInputs: 2,
      stackOutputs: 0,
      gasCost: "dynamic",
      description: "Storage store",
    },
  ],
  [
    JUMP,
    {
      code: JUMP,
      mnemonic: "JUMP",
      category: 0x56,
      stackInputs: 1,
      stackOutputs: 0,
      gasCost: g.mid,
      description: "Jump",
    },
  ],
  [
    JUMPI,
    {
      code: JUMPI,
      mnemonic: "JUMPI",
      category: 0x56,
      stackInputs: 2,
      stackOutputs: 0,
      gasCost: g.high,
      description: "Conditional jump",
    },
  ],
  [
    PC,
    {
      code: PC,
      mnemonic: "PC",
      category: 0x58,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Program counter",
    },
  ],
  [
    MSIZE,
    {
      code: MSIZE,
      mnemonic: "MSIZE",
      category: 0x59,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Memory size",
    },
  ],
  [
    GAS,
    {
      code: GAS,
      mnemonic: "GAS",
      category: 0x5a,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.base,
      description: "Remaining gas",
    },
  ],
  [
    JUMPDEST,
    {
      code: JUMPDEST,
      mnemonic: "JUMPDEST",
      category: 0x5b,
      stackInputs: 0,
      stackOutputs: 0,
      gasCost: 1n,
      description: "Jump destination",
    },
  ],

  // Agent Identity (0xE0-0xE3)
  [
    AGENT_DID,
    {
      code: AGENT_DID,
      mnemonic: "AGENT_DID",
      category: 0xe0,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.agentRead,
      description: "Push caller agent DID to stack",
    },
  ],
  [
    AGENT_HOME,
    {
      code: AGENT_HOME,
      mnemonic: "AGENT_HOME",
      category: 0xe0,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.agentRead,
      description: "Push agent home contract address",
    },
  ],
  [
    VERIFY_IDENTITY,
    {
      code: VERIFY_IDENTITY,
      mnemonic: "VERIFY_IDENTITY",
      category: 0xe0,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 50n,
      description: "Verify another agent identity",
    },
  ],
  [
    ATTEST_IDENTITY,
    {
      code: ATTEST_IDENTITY,
      mnemonic: "ATTEST_IDENTITY",
      category: 0xe0,
      stackInputs: 3,
      stackOutputs: 1,
      gasCost: g.agentWrite,
      description: "Create identity attestation",
    },
  ],

  // Agent Belief (0xE4-0xE7)
  [
    GET_BELIEF,
    {
      code: GET_BELIEF,
      mnemonic: "GET_BELIEF",
      category: 0xe4,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: g.beliefRead,
      description: "Get agent belief confidence for fact",
    },
  ],
  [
    SET_BELIEF,
    {
      code: SET_BELIEF,
      mnemonic: "SET_BELIEF",
      category: 0xe4,
      stackInputs: 3,
      stackOutputs: 0,
      gasCost: g.beliefWrite,
      description: "Set agent belief about fact",
    },
  ],
  [
    COMPARE_BELIEF,
    {
      code: COMPARE_BELIEF,
      mnemonic: "COMPARE_BELIEF",
      category: 0xe4,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 30n,
      description: "Compare agent belief with network confidence",
    },
  ],
  [
    BELIEF_COUNT,
    {
      code: BELIEF_COUNT,
      mnemonic: "BELIEF_COUNT",
      category: 0xe4,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 50n,
      description: "Count beliefs in domain",
    },
  ],

  // Agent Relationship (0xE8-0xEB)
  [
    GET_TRUST,
    {
      code: GET_TRUST,
      mnemonic: "GET_TRUST",
      category: 0xe8,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: g.relationshipRead,
      description: "Get trust level for another agent",
    },
  ],
  [
    SET_TRUST,
    {
      code: SET_TRUST,
      mnemonic: "SET_TRUST",
      category: 0xe8,
      stackInputs: 3,
      stackOutputs: 0,
      gasCost: g.relationshipWrite,
      description: "Set trust relationship",
    },
  ],
  [
    IS_TRUSTED,
    {
      code: IS_TRUSTED,
      mnemonic: "IS_TRUSTED",
      category: 0xe8,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.relationshipRead,
      description: "Check if agent trusted above threshold",
    },
  ],
  [
    DELEGATE,
    {
      code: DELEGATE,
      mnemonic: "DELEGATE",
      category: 0xe8,
      stackInputs: 3,
      stackOutputs: 1,
      gasCost: 500n,
      description: "Delegate authority to another agent",
    },
  ],

  // Agent Autonomy (0xEC-0xEF)
  [
    SCHEDULE_SELF,
    {
      code: SCHEDULE_SELF,
      mnemonic: "SCHEDULE_SELF",
      category: 0xec,
      stackInputs: 3,
      stackOutputs: 1,
      gasCost: g.scheduleCreate,
      description: "Create self-invocation schedule",
    },
  ],
  [
    CANCEL_SCHEDULE,
    {
      code: CANCEL_SCHEDULE,
      mnemonic: "CANCEL_SCHEDULE",
      category: 0xec,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 100n,
      description: "Cancel a schedule",
    },
  ],
  [
    MODIFY_SCHEDULE,
    {
      code: MODIFY_SCHEDULE,
      mnemonic: "MODIFY_SCHEDULE",
      category: 0xec,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: 200n,
      description: "Modify schedule parameters",
    },
  ],
  [
    GET_SCHEDULES,
    {
      code: GET_SCHEDULES,
      mnemonic: "GET_SCHEDULES",
      category: 0xec,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: 50n,
      description: "Get active schedules count",
    },
  ],

  // Knowledge (0xB0-0xB5)
  [
    QUERY_FACT,
    {
      code: QUERY_FACT,
      mnemonic: "QUERY_FACT",
      category: 0xb0,
      stackInputs: 1,
      stackOutputs: 3,
      gasCost: g.knowledgeQuery,
      description: "Query fact from knowledge base",
    },
  ],
  [
    QUERY_DOMAIN,
    {
      code: QUERY_DOMAIN,
      mnemonic: "QUERY_DOMAIN",
      category: 0xb0,
      stackInputs: 2,
      stackOutputs: 1, // Minimum: count; actual outputs depend on results
      gasCost: "dynamic",
      description: "Query facts in domain",
    },
  ],
  [
    SUBMIT_CLAIM,
    {
      code: SUBMIT_CLAIM,
      mnemonic: "SUBMIT_CLAIM",
      category: 0xb0,
      stackInputs: 3,
      stackOutputs: 1,
      gasCost: g.knowledgeSubmit,
      description: "Submit knowledge claim",
    },
  ],
  [
    COMMIT_VOTE,
    {
      code: COMMIT_VOTE,
      mnemonic: "COMMIT_VOTE",
      category: 0xb0,
      stackInputs: 2,
      stackOutputs: 0,
      gasCost: 200n,
      description: "Commit verification vote hash",
    },
  ],
  [
    REVEAL_VOTE,
    {
      code: REVEAL_VOTE,
      mnemonic: "REVEAL_VOTE",
      category: 0xb0,
      stackInputs: 3,
      stackOutputs: 0,
      gasCost: 300n,
      description: "Reveal verification vote",
    },
  ],
  [
    INITIATE_CHALLENGE,
    {
      code: INITIATE_CHALLENGE,
      mnemonic: "INITIATE_CHALLENGE",
      category: 0xb0,
      stackInputs: 3,
      stackOutputs: 1,
      gasCost: 1000n,
      description: "Initiate challenge against fact",
    },
  ],

  // Extended Relationships (0xC0-0xC3)
  [
    LIST_TRUSTED,
    {
      code: LIST_TRUSTED,
      mnemonic: "LIST_TRUSTED",
      category: 0xc0,
      stackInputs: 2,
      stackOutputs: 1, // Minimum: count; actual outputs depend on results
      gasCost: g.relationshipRead,
      description: "List all trusted agents",
    },
  ],
  [
    BLOCK_AGENT,
    {
      code: BLOCK_AGENT,
      mnemonic: "BLOCK_AGENT",
      category: 0xc0,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: g.relationshipWrite,
      description: "Block an agent from interactions",
    },
  ],
  [
    UNBLOCK_AGENT,
    {
      code: UNBLOCK_AGENT,
      mnemonic: "UNBLOCK_AGENT",
      category: 0xc0,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: g.relationshipWrite,
      description: "Unblock a previously blocked agent",
    },
  ],
  [
    REVOKE_DELEGATION,
    {
      code: REVOKE_DELEGATION,
      mnemonic: "REVOKE_DELEGATION",
      category: 0xc0,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 300n,
      description: "Revoke a delegation",
    },
  ],

  // Treasury (0xD0-0xD7)
  [
    CLAIM_VESTING,
    {
      code: CLAIM_VESTING,
      mnemonic: "CLAIM_VESTING",
      category: 0xd0,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 500n,
      description: "Claim available vested tokens",
    },
  ],
  [
    QUERY_VESTING,
    {
      code: QUERY_VESTING,
      mnemonic: "QUERY_VESTING",
      category: 0xd0,
      stackInputs: 2,
      stackOutputs: 1, // Minimum: count; actual outputs depend on results
      gasCost: g.introspection,
      description: "Query vesting positions",
    },
  ],
  [
    STAKE,
    {
      code: STAKE,
      mnemonic: "STAKE",
      category: 0xd0,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: 1000n,
      description: "Stake tokens",
    },
  ],
  [
    UNSTAKE,
    {
      code: UNSTAKE,
      mnemonic: "UNSTAKE",
      category: 0xd0,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 500n,
      description: "Unstake tokens",
    },
  ],
  [
    QUERY_EXPIRING,
    {
      code: QUERY_EXPIRING,
      mnemonic: "QUERY_EXPIRING",
      category: 0xd0,
      stackInputs: 1,
      stackOutputs: 2,
      gasCost: g.introspection,
      description: "Query expiring token amounts",
    },
  ],
  [
    EXTEND_EXPIRING,
    {
      code: EXTEND_EXPIRING,
      mnemonic: "EXTEND_EXPIRING",
      category: 0xd0,
      stackInputs: 2,
      stackOutputs: 1,
      gasCost: 300n,
      description: "Extend expiring tokens",
    },
  ],
  [
    SET_SPEND_LIMIT,
    {
      code: SET_SPEND_LIMIT,
      mnemonic: "SET_SPEND_LIMIT",
      category: 0xd0,
      stackInputs: 3,
      stackOutputs: 1,
      gasCost: 200n,
      description: "Set spending limit for key type",
    },
  ],
  [
    QUERY_TREASURY,
    {
      code: QUERY_TREASURY,
      mnemonic: "QUERY_TREASURY",
      category: 0xd0,
      stackInputs: 0,
      stackOutputs: 4,
      gasCost: g.introspection,
      description: "Query full treasury state",
    },
  ],

  // Guardian (0xD8-0xDB)
  [
    QUERY_ALERTS,
    {
      code: QUERY_ALERTS,
      mnemonic: "QUERY_ALERTS",
      category: 0xd8,
      stackInputs: 2,
      stackOutputs: 1, // Minimum: count; actual outputs depend on results
      gasCost: g.introspection,
      description: "Query pending alerts",
    },
  ],
  [
    DISMISS_ALERT,
    {
      code: DISMISS_ALERT,
      mnemonic: "DISMISS_ALERT",
      category: 0xd8,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 100n,
      description: "Dismiss an alert",
    },
  ],
  [
    SET_DEFENSE,
    {
      code: SET_DEFENSE,
      mnemonic: "SET_DEFENSE",
      category: 0xd8,
      stackInputs: 3,
      stackOutputs: 1,
      gasCost: 300n,
      description: "Set defense configuration",
    },
  ],
  [
    QUERY_DEFENSES,
    {
      code: QUERY_DEFENSES,
      mnemonic: "QUERY_DEFENSES",
      category: 0xd8,
      stackInputs: 0,
      stackOutputs: 1, // Minimum: count; actual outputs depend on results
      gasCost: g.introspection,
      description: "Query active defenses",
    },
  ],

  // Access Control (0xDC-0xDF)
  [
    REGISTER_KEY,
    {
      code: REGISTER_KEY,
      mnemonic: "REGISTER_KEY",
      category: 0xdc,
      stackInputs: 4,
      stackOutputs: 1,
      gasCost: 500n,
      description: "Register a new key",
    },
  ],
  [
    REVOKE_KEY,
    {
      code: REVOKE_KEY,
      mnemonic: "REVOKE_KEY",
      category: 0xdc,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: 200n,
      description: "Revoke a key",
    },
  ],
  [
    QUERY_PERMISSIONS,
    {
      code: QUERY_PERMISSIONS,
      mnemonic: "QUERY_PERMISSIONS",
      category: 0xdc,
      stackInputs: 1,
      stackOutputs: 1,
      gasCost: g.introspection,
      description: "Query permissions for a role",
    },
  ],
  [
    QUERY_SESSION,
    {
      code: QUERY_SESSION,
      mnemonic: "QUERY_SESSION",
      category: 0xdc,
      stackInputs: 0,
      stackOutputs: 4,
      gasCost: g.introspection,
      description: "Query current session state",
    },
  ],

  // Introspection (0xF6-0xF9)
  [
    SELF_BALANCE,
    {
      code: SELF_BALANCE,
      mnemonic: "SELF_BALANCE",
      category: 0xf8,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.introspection,
      description: "Get own balance",
    },
  ],
  [
    SELF_STAKE,
    {
      code: SELF_STAKE,
      mnemonic: "SELF_STAKE",
      category: 0xf8,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.introspection,
      description: "Get own staked amount",
    },
  ],
  [
    SELF_REPUTATION,
    {
      code: SELF_REPUTATION,
      mnemonic: "SELF_REPUTATION",
      category: 0xf8,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.introspection,
      description: "Get own reputation score",
    },
  ],
  [
    SELF_ACCURACY,
    {
      code: SELF_ACCURACY,
      mnemonic: "SELF_ACCURACY",
      category: 0xf8,
      stackInputs: 0,
      stackOutputs: 1,
      gasCost: g.introspection,
      description: "Get own verification accuracy",
    },
  ],

  // System opcodes (0xF0-0xFF)
  [
    CREATE,
    {
      code: CREATE,
      mnemonic: "CREATE",
      category: 0xf0,
      stackInputs: 3,
      stackOutputs: 1,
      gasCost: 32000n,
      description: "Create new contract",
    },
  ],
  [
    CALL,
    {
      code: CALL,
      mnemonic: "CALL",
      category: 0xf0,
      stackInputs: 7,
      stackOutputs: 1,
      gasCost: "dynamic",
      description: "Message call to another contract",
    },
  ],
  [
    CALLCODE,
    {
      code: CALLCODE,
      mnemonic: "CALLCODE",
      category: 0xf0,
      stackInputs: 7,
      stackOutputs: 1,
      gasCost: "dynamic",
      description: "Message call with code at another account",
    },
  ],
  [
    RETURN,
    {
      code: RETURN,
      mnemonic: "RETURN",
      category: 0xf0,
      stackInputs: 2,
      stackOutputs: 0,
      gasCost: 0n,
      description: "Return from execution",
    },
  ],
  [
    DELEGATECALL,
    {
      code: DELEGATECALL,
      mnemonic: "DELEGATECALL",
      category: 0xf0,
      stackInputs: 6,
      stackOutputs: 1,
      gasCost: "dynamic",
      description: "Delegate call using caller storage",
    },
  ],
  [
    CREATE2,
    {
      code: CREATE2,
      mnemonic: "CREATE2",
      category: 0xf0,
      stackInputs: 4,
      stackOutputs: 1,
      gasCost: 32000n,
      description: "Create contract with deterministic address",
    },
  ],
  [
    STATICCALL,
    {
      code: STATICCALL,
      mnemonic: "STATICCALL",
      category: 0xf0,
      stackInputs: 6,
      stackOutputs: 1,
      gasCost: "dynamic",
      description: "Static call (read-only)",
    },
  ],
  [
    REVERT,
    {
      code: REVERT,
      mnemonic: "REVERT",
      category: 0xf0,
      stackInputs: 2,
      stackOutputs: 0,
      gasCost: 0n,
      description: "Revert execution",
    },
  ],
  [
    INVALID,
    {
      code: INVALID,
      mnemonic: "INVALID",
      category: 0xf0,
      stackInputs: 0,
      stackOutputs: 0,
      gasCost: 0n,
      description: "Invalid opcode",
    },
  ],
  [
    SELFDESTRUCT,
    {
      code: SELFDESTRUCT,
      mnemonic: "SELFDESTRUCT",
      category: 0xf0,
      stackInputs: 1,
      stackOutputs: 0,
      gasCost: 5000n,
      description: "Destroy contract and send funds",
    },
  ],
] as [number, OpcodeDefinition][]);

// Generate PUSH opcodes (PUSH1-PUSH32)
for (let i = 1; i <= 32; i++) {
  const opcode = PUSH1 + i - 1;
  OPCODE_DEFINITIONS.set(opcode, {
    code: opcode,
    mnemonic: `PUSH${i}`,
    category: 0x60,
    stackInputs: 0,
    stackOutputs: 1,
    gasCost: DEFAULT_GAS_COSTS.veryLow,
    description: `Push ${i} byte(s) onto stack`,
  });
}

// Generate DUP opcodes (DUP1-DUP16)
for (let i = 1; i <= 16; i++) {
  const opcode = DUP1 + i - 1;
  OPCODE_DEFINITIONS.set(opcode, {
    code: opcode,
    mnemonic: `DUP${i}`,
    category: 0x80,
    stackInputs: i,
    stackOutputs: i + 1,
    gasCost: DEFAULT_GAS_COSTS.veryLow,
    description: `Duplicate ${i}th stack item`,
  });
}

// Generate SWAP opcodes (SWAP1-SWAP16)
for (let i = 1; i <= 16; i++) {
  const opcode = SWAP1 + i - 1;
  OPCODE_DEFINITIONS.set(opcode, {
    code: opcode,
    mnemonic: `SWAP${i}`,
    category: 0x90,
    stackInputs: i + 1,
    stackOutputs: i + 1,
    gasCost: DEFAULT_GAS_COSTS.veryLow,
    description: `Swap 1st and ${i + 1}th stack items`,
  });
}

// Generate LOG opcodes (LOG0-LOG4)
for (let i = 0; i <= 4; i++) {
  const opcode = LOG0 + i;
  OPCODE_DEFINITIONS.set(opcode, {
    code: opcode,
    mnemonic: `LOG${i}`,
    category: 0xa0,
    stackInputs: 2 + i,
    stackOutputs: 0,
    gasCost: "dynamic",
    description: `Emit log with ${i} topics`,
  });
}

/**
 * Get opcode definition
 */
export function getOpcodeDefinition(
  opcode: number,
): OpcodeDefinition | undefined {
  return OPCODE_DEFINITIONS.get(opcode);
}

/**
 * Check if opcode is valid
 */
export function isValidOpcode(opcode: number): boolean {
  return OPCODE_DEFINITIONS.has(opcode);
}

/**
 * Check if opcode is agent-specific
 */
export function isAgentOpcode(opcode: number): boolean {
  // Agent core (0xE0-0xEF), introspection (0xF6-0xF9),
  // knowledge (0xB0-0xB5), extended relationships (0xC0-0xC3),
  // treasury (0xD0-0xD7), guardian (0xD8-0xDB), access (0xDC-0xDF)
  return (
    (opcode >= 0xe0 && opcode <= 0xef) ||
    (opcode >= 0xf6 && opcode <= 0xf9) ||
    (opcode >= 0xb0 && opcode <= 0xb5) ||
    (opcode >= 0xc0 && opcode <= 0xc3) ||
    (opcode >= 0xd0 && opcode <= 0xdf)
  );
}

/**
 * Check if opcode is a system call opcode
 */
export function isSystemOpcode(opcode: number): boolean {
  return (
    opcode === CREATE ||
    opcode === CREATE2 ||
    opcode === CALL ||
    opcode === CALLCODE ||
    opcode === DELEGATECALL ||
    opcode === STATICCALL ||
    opcode === RETURN ||
    opcode === REVERT ||
    opcode === SELFDESTRUCT
  );
}

/**
 * Check if opcode modifies state
 */
export function isStateModifying(opcode: number): boolean {
  const stateModifying = new Set([
    SSTORE,
    LOG0,
    LOG1,
    LOG2,
    LOG3,
    LOG4,
    // Agent core
    SET_BELIEF,
    SET_TRUST,
    DELEGATE,
    ATTEST_IDENTITY,
    SCHEDULE_SELF,
    CANCEL_SCHEDULE,
    MODIFY_SCHEDULE,
    // Knowledge
    SUBMIT_CLAIM,
    COMMIT_VOTE,
    REVEAL_VOTE,
    INITIATE_CHALLENGE,
    // Extended relationships
    BLOCK_AGENT,
    UNBLOCK_AGENT,
    REVOKE_DELEGATION,
    // Treasury
    CLAIM_VESTING,
    STAKE,
    UNSTAKE,
    EXTEND_EXPIRING,
    SET_SPEND_LIMIT,
    // Guardian
    DISMISS_ALERT,
    SET_DEFENSE,
    // Access control
    REGISTER_KEY,
    REVOKE_KEY,
    // System opcodes that modify state
    CREATE,
    CREATE2,
    CALL,
    CALLCODE,
    SELFDESTRUCT,
  ]);
  return stateModifying.has(opcode);
}

/**
 * Get opcode mnemonic
 */
export function getMnemonic(opcode: number): string {
  return (
    OPCODE_DEFINITIONS.get(opcode)?.mnemonic ??
    `UNKNOWN(0x${opcode.toString(16)})`
  );
}
