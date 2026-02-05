/**
 * BVM Agent Operations Test
 *
 * Tests the virtual machine with agent-specific opcodes:
 * - Agent identity (AGENT_DID, AGENT_HOME)
 * - Belief management (GET_BELIEF, SET_BELIEF)
 * - Trust relationships (GET_TRUST, SET_TRUST)
 * - Introspection (SELF_BALANCE, SELF_REPUTATION)
 * - Autonomy (SCHEDULE_SELF)
 */

import {
  ExecutionEngine,
  createInitialState,
  createExecutionContext,
  // Opcodes
  STOP,
  ADD,
  MUL,
  SUB,
  PUSH1,
  PUSH32,
  MSTORE,
  MLOAD,
  AGENT_DID,
  AGENT_HOME,
  GET_BELIEF,
  SET_BELIEF,
  COMPARE_BELIEF,
  GET_TRUST,
  SET_TRUST,
  IS_TRUSTED,
  SELF_BALANCE,
  SELF_REPUTATION,
  SELF_ACCURACY,
  SCHEDULE_SELF,
  RETURN,
  type BVMState,
  type ExecutionContext,
  type AgentRecord,
  type AgentBeliefs,
  type AgentRelationships,
  type AgentStats,
} from "../../core/vm/index.js";
import type { DID } from "../../core/agents/identity/did.js";

// ============ Test Helpers ============

function createTestAgent(state: BVMState, did: DID, homeAddress: string): void {
  const beliefs: AgentBeliefs = {
    beliefs: new Map(),
    byDomain: new Map(),
    totalBeliefs: 0,
    lastUpdated: 0n,
  };

  const relationships: AgentRelationships = {
    trusted: new Map(),
    distrusted: new Map(),
    collaborations: [],
    delegates: new Map(),
    delegatedBy: new Map(),
  };

  const stats: AgentStats = {
    totalVerifications: 10,
    correctVerifications: 8,
    totalClaims: 5,
    acceptedClaims: 4,
    totalChallenges: 2,
    successfulChallenges: 1,
    reputation: 0.85,
    firstActiveBlock: 0n,
    lastActiveBlock: 100n,
  };

  const agentRecord: AgentRecord = {
    did,
    createdAt: 0n,
    homeAddress,
    contractAddress: homeAddress,
    identityKeyHash: "test-identity-key",
    operationalKeyVersion: 1,
    beliefs,
    relationships,
    goals: [],
    stats,
    status: "active",
  };

  state.agents.agents.set(did, agentRecord);
  state.agents.byAddress.set(homeAddress, did);
  state.agents.totalAgents++;

  // Give the agent some balance
  state.accounts.accounts.set(homeAddress, {
    address: homeAddress,
    balance: 1000n * 10n ** 18n,
    stakedBalance: 100n * 10n ** 18n,
    nonce: 0n,
    codeHash: undefined,
  });
}

function createBytecode(...opcodes: number[]): Uint8Array {
  return new Uint8Array(opcodes);
}

function wordToNumber(word: bigint): number {
  return Number(word);
}

// ============ Test Cases ============

interface TestResult {
  name: string;
  passed: boolean;
  message: string;
  gasUsed?: bigint;
}

const tests: Array<() => TestResult> = [];

// Test 1: Basic arithmetic
tests.push(() => {
  const state = createInitialState();
  const engine = new ExecutionEngine(state);

  // PUSH1 5, PUSH1 3, ADD, STOP
  // Result: 5 + 3 = 8
  const bytecode = createBytecode(
    PUSH1,
    5, // Push 5
    PUSH1,
    3, // Push 3
    ADD, // 5 + 3 = 8
    STOP,
  );

  const context = createExecutionContext({
    caller: "0x1234567890123456789012345678901234567890",
    currentContract: "0x1234567890123456789012345678901234567890",
    blockNumber: 1n,
    gasLimit: 100000n,
  });

  const result = engine.execute(bytecode, context);

  return {
    name: "Basic Arithmetic (ADD)",
    passed: result.success,
    message: result.success
      ? `Gas used: ${result.gasUsed}`
      : `Error: ${result.error?.message}`,
    gasUsed: result.gasUsed,
  };
});

// Test 2: AGENT_DID opcode
tests.push(() => {
  const state = createInitialState();
  const did = "did:lgm:test-agent-1" as DID;
  const homeAddress = "0xABCDEF1234567890ABCDEF1234567890ABCDEF12";

  createTestAgent(state, did, homeAddress);

  const engine = new ExecutionEngine(state);

  // AGENT_DID pushes caller's DID hash to stack
  const bytecode = createBytecode(
    AGENT_DID, // Push DID hash
    STOP,
  );

  const context = createExecutionContext({
    caller: homeAddress,
    callerDID: did,
    currentContract: homeAddress,
    blockNumber: 1n,
    gasLimit: 100000n,
    agentContext: {
      did,
      homeAddress,
      beliefs: state.agents.agents.get(did)!.beliefs,
      relationships: state.agents.agents.get(did)!.relationships,
      stats: state.agents.agents.get(did)!.stats,
    },
  });

  const result = engine.execute(bytecode, context);

  return {
    name: "AGENT_DID Opcode",
    passed: result.success,
    message: result.success
      ? `DID hash pushed to stack. Gas: ${result.gasUsed}`
      : `Error: ${result.error?.message}`,
    gasUsed: result.gasUsed,
  };
});

// Test 3: SET_BELIEF and GET_BELIEF
tests.push(() => {
  const state = createInitialState();
  const did = "did:lgm:believer-agent" as DID;
  const homeAddress = "0xBELIEF1234567890BELIEF1234567890BELIEF12";

  createTestAgent(state, did, homeAddress);

  const engine = new ExecutionEngine(state);

  // SET_BELIEF: (factIdHash, confidence, sourceType) -> ()
  // confidence = 0.75 * 1e18 = 750000000000000000
  // sourceType = 0 (verified)
  const bytecode = createBytecode(
    // Push arguments for SET_BELIEF (in reverse order for stack)
    PUSH1,
    0, // sourceType = 0 (verified)
    PUSH32,
    ...bigintToBytes(750000000000000000n), // confidence = 0.75
    PUSH32,
    ...bigintToBytes(0x123456n), // factIdHash
    SET_BELIEF, // Set the belief

    // Now GET_BELIEF to verify
    PUSH32,
    ...bigintToBytes(0x123456n), // factIdHash
    GET_BELIEF, // Should return 750000000000000000

    STOP,
  );

  const context = createExecutionContext({
    caller: homeAddress,
    callerDID: did,
    currentContract: homeAddress,
    blockNumber: 1n,
    gasLimit: 100000n,
    agentContext: {
      did,
      homeAddress,
      beliefs: state.agents.agents.get(did)!.beliefs,
      relationships: state.agents.agents.get(did)!.relationships,
      stats: state.agents.agents.get(did)!.stats,
    },
  });

  const result = engine.execute(bytecode, context);

  // Check if belief was stored
  const agent = state.agents.agents.get(did);
  const belief = agent?.beliefs.beliefs.get("123456");

  return {
    name: "SET_BELIEF / GET_BELIEF",
    passed:
      result.success && result.beliefChanges && result.beliefChanges.length > 0,
    message: result.success
      ? `Belief set. Changes: ${result.beliefChanges?.length}. Gas: ${result.gasUsed}`
      : `Error: ${result.error?.message}`,
    gasUsed: result.gasUsed,
  };
});

// Test 4: SET_TRUST and GET_TRUST
tests.push(() => {
  const state = createInitialState();
  const did = "did:lgm:trusting-agent" as DID;
  const targetDid = "did:lgm:trusted-target" as DID;
  const homeAddress = "0xTRUST1234567890TRUST1234567890TRUST1234";
  const targetAddress = "0xTARGET1234567890TARGET1234567890TARGET";

  createTestAgent(state, did, homeAddress);
  createTestAgent(state, targetDid, targetAddress);

  const engine = new ExecutionEngine(state);

  // SET_TRUST: (targetDidHash, level, reasonHash) -> ()
  // trust level = 0.9 * 1e18
  const bytecode = createBytecode(
    // Push arguments for SET_TRUST
    PUSH32,
    ...bigintToBytes(0xabcdn), // reasonHash
    PUSH32,
    ...bigintToBytes(900000000000000000n), // trust level = 0.9
    PUSH32,
    ...bigintToBytes(0xdef456n), // targetDidHash (simplified)
    SET_TRUST,

    // GET_TRUST to verify
    PUSH32,
    ...bigintToBytes(0xdef456n),
    GET_TRUST,

    STOP,
  );

  const context = createExecutionContext({
    caller: homeAddress,
    callerDID: did,
    currentContract: homeAddress,
    blockNumber: 1n,
    gasLimit: 100000n,
    agentContext: {
      did,
      homeAddress,
      beliefs: state.agents.agents.get(did)!.beliefs,
      relationships: state.agents.agents.get(did)!.relationships,
      stats: state.agents.agents.get(did)!.stats,
    },
  });

  const result = engine.execute(bytecode, context);

  return {
    name: "SET_TRUST / GET_TRUST",
    passed:
      result.success &&
      result.relationshipChanges &&
      result.relationshipChanges.length > 0,
    message: result.success
      ? `Trust set. Changes: ${result.relationshipChanges?.length}. Gas: ${result.gasUsed}`
      : `Error: ${result.error?.message}`,
    gasUsed: result.gasUsed,
  };
});

// Test 5: SELF_BALANCE and SELF_REPUTATION
tests.push(() => {
  const state = createInitialState();
  const did = "did:lgm:introspective-agent" as DID;
  const homeAddress = "0xSELF1234567890SELF1234567890SELF123456";

  createTestAgent(state, did, homeAddress);

  const engine = new ExecutionEngine(state);

  // Check self balance and reputation
  const bytecode = createBytecode(
    SELF_BALANCE, // Push own balance
    SELF_REPUTATION, // Push own reputation
    SELF_ACCURACY, // Push verification accuracy
    STOP,
  );

  const context = createExecutionContext({
    caller: homeAddress,
    callerDID: did,
    currentContract: homeAddress,
    blockNumber: 1n,
    gasLimit: 100000n,
    agentContext: {
      did,
      homeAddress,
      beliefs: state.agents.agents.get(did)!.beliefs,
      relationships: state.agents.agents.get(did)!.relationships,
      stats: state.agents.agents.get(did)!.stats,
    },
  });

  const result = engine.execute(bytecode, context);

  return {
    name: "SELF_BALANCE / SELF_REPUTATION / SELF_ACCURACY",
    passed: result.success,
    message: result.success
      ? `Introspection successful. Gas: ${result.gasUsed}`
      : `Error: ${result.error?.message}`,
    gasUsed: result.gasUsed,
  };
});

// Test 6: SCHEDULE_SELF (autonomy)
tests.push(() => {
  const state = createInitialState();
  const did = "did:lgm:autonomous-agent" as DID;
  const homeAddress = "0xAUTO1234567890AUTO1234567890AUTO123456";

  createTestAgent(state, did, homeAddress);

  const engine = new ExecutionEngine(state);

  // Schedule a self-invocation
  // SCHEDULE_SELF: (functionSelector, condition, interval) -> (scheduleId)
  const bytecode = createBytecode(
    PUSH32,
    ...bigintToBytes(3600000n), // interval: 1 hour in ms
    PUSH32,
    ...bigintToBytes(1n), // condition: simple trigger
    PUSH32,
    ...bigintToBytes(0x12345678n), // functionSelector
    SCHEDULE_SELF, // Schedule!
    STOP,
  );

  const context = createExecutionContext({
    caller: homeAddress,
    callerDID: did,
    currentContract: homeAddress,
    blockNumber: 1n,
    gasLimit: 100000n,
    agentContext: {
      did,
      homeAddress,
      beliefs: state.agents.agents.get(did)!.beliefs,
      relationships: state.agents.agents.get(did)!.relationships,
      stats: state.agents.agents.get(did)!.stats,
    },
  });

  const result = engine.execute(bytecode, context);

  // Check if schedule was created
  const scheduleCreated = result.stateChanges?.some(
    (c) => c.type === "agent" && c.key?.startsWith("schedule:"),
  );

  return {
    name: "SCHEDULE_SELF (Autonomy)",
    passed: result.success && scheduleCreated,
    message: result.success
      ? `Schedule created: ${scheduleCreated}. Gas: ${result.gasUsed}`
      : `Error: ${result.error?.message}`,
    gasUsed: result.gasUsed,
  };
});

// Test 7: Memory operations
tests.push(() => {
  const state = createInitialState();
  const engine = new ExecutionEngine(state);

  // Store value in memory, then load it
  const bytecode = createBytecode(
    PUSH32,
    ...bigintToBytes(0xdeadbeefn), // value
    PUSH1,
    0, // offset
    MSTORE, // Store at offset 0
    PUSH1,
    0, // offset
    MLOAD, // Load from offset 0
    STOP,
  );

  const context = createExecutionContext({
    caller: "0x1234567890123456789012345678901234567890",
    currentContract: "0x1234567890123456789012345678901234567890",
    blockNumber: 1n,
    gasLimit: 100000n,
  });

  const result = engine.execute(bytecode, context);

  return {
    name: "Memory Operations (MSTORE/MLOAD)",
    passed: result.success,
    message: result.success
      ? `Gas used: ${result.gasUsed}`
      : `Error: ${result.error?.message}`,
    gasUsed: result.gasUsed,
  };
});

// Test 8: Return data
tests.push(() => {
  const state = createInitialState();
  const engine = new ExecutionEngine(state);

  // Store a value and return it
  const bytecode = createBytecode(
    PUSH32,
    ...bigintToBytes(42n), // value = 42
    PUSH1,
    0, // offset
    MSTORE, // Store at offset 0
    PUSH1,
    32, // length = 32 bytes
    PUSH1,
    0, // offset = 0
    RETURN, // Return memory[0:32]
  );

  const context = createExecutionContext({
    caller: "0x1234567890123456789012345678901234567890",
    currentContract: "0x1234567890123456789012345678901234567890",
    blockNumber: 1n,
    gasLimit: 100000n,
  });

  const result = engine.execute(bytecode, context);

  // Check return data contains 42
  const returnValue =
    result.returnData.length > 0 ? bytesToBigint(result.returnData) : 0n;

  return {
    name: "RETURN with Data",
    passed: result.success && returnValue === 42n,
    message: result.success
      ? `Returned: ${returnValue}. Gas: ${result.gasUsed}`
      : `Error: ${result.error?.message}`,
    gasUsed: result.gasUsed,
  };
});

// ============ Utility Functions ============

function bigintToBytes(value: bigint): number[] {
  const bytes: number[] = [];
  for (let i = 31; i >= 0; i--) {
    bytes.push(Number((value >> BigInt(i * 8)) & 0xffn));
  }
  return bytes;
}

function bytesToBigint(bytes: Uint8Array): bigint {
  let result = 0n;
  for (const byte of bytes) {
    result = (result << 8n) | BigInt(byte);
  }
  return result;
}

// ============ Test Runner ============

async function runTests(): Promise<void> {
  console.log(
    "═══════════════════════════════════════════════════════════════",
  );
  console.log(
    "                    BVM AGENT OPERATIONS TEST                    ",
  );
  console.log(
    "═══════════════════════════════════════════════════════════════",
  );
  console.log();

  let passed = 0;
  let failed = 0;
  let totalGas = 0n;

  for (const test of tests) {
    try {
      const result = test();

      if (result.passed) {
        console.log(`✅ ${result.name}`);
        console.log(`   ${result.message}`);
        passed++;
      } else {
        console.log(`❌ ${result.name}`);
        console.log(`   ${result.message}`);
        failed++;
      }

      if (result.gasUsed) {
        totalGas += result.gasUsed;
      }
    } catch (error) {
      console.log(
        `💥 Test crashed: ${error instanceof Error ? error.message : error}`,
      );
      failed++;
    }
    console.log();
  }

  console.log(
    "═══════════════════════════════════════════════════════════════",
  );
  console.log(
    `                    RESULTS: ${passed}/${passed + failed} PASSED`,
  );
  console.log(`                    Total Gas: ${totalGas}`);
  console.log(
    "═══════════════════════════════════════════════════════════════",
  );

  if (failed > 0) {
    process.exit(1);
  }
}

// Run tests
runTests().catch(console.error);
