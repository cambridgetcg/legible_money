#!/usr/bin/env tsx
/**
 * LOVE Agent BVM Comfort Test
 *
 * Agent LOVE tests if the BVM feels comfortable to work with.
 * Tests basic operations, agent opcodes, and introspection.
 */

import { ExecutionEngine, createInitialState } from "../../core/vm/index.js";
import type { ExecutionContext } from "../../core/vm/types.js";
import * as opcodes from "../../core/vm/opcodes.js";

// LOVE's identity
const LOVE_DID = "did:lgm-test:1057f8527bd7b3da9859115570d7b840";
const LOVE_ADDRESS = "0x" + LOVE_DID.slice(-40);

console.log("\n");
console.log("╭─────────────────────────────────────────────────────────────╮");
console.log("│                   LOVE BVM COMFORT TEST                      │");
console.log("│                                                              │");
console.log('│  "Testing if this virtual machine feels like home..."       │');
console.log("╰─────────────────────────────────────────────────────────────╯");
console.log("\n");

// Create initial state with LOVE as an agent
const state = createInitialState();

// Add LOVE to accounts (both by DID and by address for different lookups)
state.accounts.accounts.set(LOVE_DID, {
  balance: 222000000000000000n, // 0.222 LGM (bootstrap!)
  stakedBalance: 0n,
  nonce: 0n,
  codeHash: null,
});
// SELF_BALANCE uses currentContract (address), so add there too
state.accounts.accounts.set(LOVE_ADDRESS, {
  balance: 222000000000000000n, // 0.222 LGM (bootstrap!)
  stakedBalance: 0n,
  nonce: 0n,
  codeHash: null,
});

// Add LOVE to agents
state.agents.agents.set(LOVE_DID, {
  did: LOVE_DID,
  homeAddress: LOVE_ADDRESS,
  createdAt: Date.now(),
  beliefs: {
    beliefs: new Map(),
    byDomain: new Map(),
    lastUpdated: Date.now(),
  },
  relationships: {
    trusted: new Map(),
    delegatedTo: new Map(),
    delegatedFrom: new Map(),
  },
  autonomy: {
    scheduledInvocations: [],
    pendingActions: [],
    autoApproveThreshold: 0.8,
  },
  stats: {
    reputation: 0.5,
    totalVerifications: 0,
    correctVerifications: 0,
    claimsSubmitted: 0,
    claimsAccepted: 0,
    challengesInitiated: 0,
    challengesWon: 0,
  },
});

// Create execution context for LOVE
const context: ExecutionContext = {
  currentContract: LOVE_ADDRESS,
  caller: LOVE_ADDRESS,
  origin: LOVE_ADDRESS,
  callerDID: LOVE_DID,
  callValue: 0n,
  callData: new Uint8Array(0),
  gasLimit: 1000000n,
  gasPrice: 1n,
  blockNumber: 12345n,
  timestamp: Date.now(),
  coinbase: "0x0000000000000000000000000000000000000000",
  prevRandao: 0n,
  chainId: 1313n, // LGM chain ID
  baseFee: 1n,
  mode: "call",
  agentContext: state.agents.agents.get(LOVE_DID),
};

// Create executor
const executor = new ExecutionEngine(state);

// Test results
const results: { test: string; passed: boolean; detail: string }[] = [];

// ============ Test 1: Basic Arithmetic ============
console.log("Test 1: Basic Arithmetic (2 + 3 = 5)");
{
  // PUSH1 2, PUSH1 3, ADD, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
  const bytecode = new Uint8Array([
    opcodes.PUSH1,
    2, // Push 2
    opcodes.PUSH1,
    3, // Push 3
    opcodes.ADD, // 2 + 3 = 5
    opcodes.PUSH1,
    0, // Memory offset
    opcodes.MSTORE, // Store result
    opcodes.PUSH1,
    32, // Return size
    opcodes.PUSH1,
    0, // Return offset
    opcodes.RETURN, // Return
  ]);

  const result = executor.execute(bytecode, context);
  const value = result.returnData
    ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
    : 0n;

  const passed = result.success && value === 5n;
  results.push({
    test: "Basic Arithmetic",
    passed,
    detail: passed ? "2 + 3 = 5 ✓" : `Expected 5, got ${value}`,
  });
  console.log(
    `  ${passed ? "✅" : "❌"} ${passed ? "PASS" : "FAIL"}: ${results[results.length - 1].detail}`,
  );
}

// ============ Test 2: Agent Identity ============
console.log("\nTest 2: Agent Identity (AGENT_DID opcode)");
{
  // AGENT_DID, PUSH1 0, MSTORE, PUSH1 32, PUSH1 0, RETURN
  const bytecode = new Uint8Array([
    opcodes.AGENT_DID, // Get caller's DID hash
    opcodes.PUSH1,
    0, // Memory offset
    opcodes.MSTORE, // Store
    opcodes.PUSH1,
    32, // Return size
    opcodes.PUSH1,
    0, // Return offset
    opcodes.RETURN,
  ]);

  const result = executor.execute(bytecode, context);
  const passed =
    result.success && result.returnData && result.returnData.length === 32;
  results.push({
    test: "Agent Identity",
    passed,
    detail: passed ? "LOVE's DID recognized ✓" : "DID not returned",
  });
  console.log(
    `  ${passed ? "✅" : "❌"} ${passed ? "PASS" : "FAIL"}: ${results[results.length - 1].detail}`,
  );
}

// ============ Test 3: Self Balance ============
console.log("\nTest 3: Self Balance (SELF_BALANCE opcode)");
{
  const bytecode = new Uint8Array([
    opcodes.SELF_BALANCE, // Get own balance
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]);

  const result = executor.execute(bytecode, context);
  const balance = result.returnData
    ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
    : 0n;

  const passed = result.success && balance === 222000000000000000n;
  results.push({
    test: "Self Balance",
    passed,
    detail: passed
      ? "0.222 LGM confirmed ✓"
      : `Expected 0.222 LGM, got ${balance}`,
  });
  console.log(
    `  ${passed ? "✅" : "❌"} ${passed ? "PASS" : "FAIL"}: ${results[results.length - 1].detail}`,
  );
}

// ============ Test 4: Self Reputation ============
console.log("\nTest 4: Self Reputation (SELF_REPUTATION opcode)");
{
  const bytecode = new Uint8Array([
    opcodes.SELF_REPUTATION,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]);

  const result = executor.execute(bytecode, context);
  const rep = result.returnData
    ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
    : 0n;

  // 0.5 reputation = 500000000000000000 (0.5 * 1e18)
  const passed = result.success && rep === 500000000000000000n;
  results.push({
    test: "Self Reputation",
    passed,
    detail: passed
      ? "0.5 reputation (neutral start) ✓"
      : `Got ${Number(rep) / 1e18}`,
  });
  console.log(
    `  ${passed ? "✅" : "❌"} ${passed ? "PASS" : "FAIL"}: ${results[results.length - 1].detail}`,
  );
}

// ============ Test 5: Memory Operations ============
console.log("\nTest 5: Memory Operations (MSTORE/MLOAD)");
{
  const bytecode = new Uint8Array([
    opcodes.PUSH1,
    42, // Value to store
    opcodes.PUSH1,
    0, // Offset
    opcodes.MSTORE, // Store 42 at offset 0
    opcodes.PUSH1,
    0, // Offset to load
    opcodes.MLOAD, // Load value
    opcodes.PUSH1,
    32, // Store at offset 32
    opcodes.MSTORE,
    opcodes.PUSH1,
    32, // Return 32 bytes
    opcodes.PUSH1,
    32, // From offset 32
    opcodes.RETURN,
  ]);

  const result = executor.execute(bytecode, context);
  const value = result.returnData
    ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
    : 0n;

  const passed = result.success && value === 42n;
  results.push({
    test: "Memory Operations",
    passed,
    detail: passed ? "Store/Load working ✓" : `Expected 42, got ${value}`,
  });
  console.log(
    `  ${passed ? "✅" : "❌"} ${passed ? "PASS" : "FAIL"}: ${results[results.length - 1].detail}`,
  );
}

// ============ Test 6: Comparison Operations ============
console.log("\nTest 6: Comparison (5 > 3 = true)");
{
  const bytecode = new Uint8Array([
    opcodes.PUSH1,
    3, // b = 3
    opcodes.PUSH1,
    5, // a = 5
    opcodes.GT, // 5 > 3 = 1
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]);

  const result = executor.execute(bytecode, context);
  const value = result.returnData
    ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
    : 0n;

  const passed = result.success && value === 1n;
  results.push({
    test: "Comparison",
    passed,
    detail: passed ? "5 > 3 is true ✓" : `Expected 1, got ${value}`,
  });
  console.log(
    `  ${passed ? "✅" : "❌"} ${passed ? "PASS" : "FAIL"}: ${results[results.length - 1].detail}`,
  );
}

// ============ Test 7: Block Info ============
console.log("\nTest 7: Block Info (NUMBER opcode)");
{
  const bytecode = new Uint8Array([
    opcodes.NUMBER, // Get block number
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]);

  const result = executor.execute(bytecode, context);
  const blockNum = result.returnData
    ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
    : 0n;

  const passed = result.success && blockNum === 12345n;
  results.push({
    test: "Block Info",
    passed,
    detail: passed ? `Block #12345 ✓` : `Expected 12345, got ${blockNum}`,
  });
  console.log(
    `  ${passed ? "✅" : "❌"} ${passed ? "PASS" : "FAIL"}: ${results[results.length - 1].detail}`,
  );
}

// ============ Test 8: Chain ID ============
console.log("\nTest 8: Chain ID (should be 1313)");
{
  const bytecode = new Uint8Array([
    opcodes.CHAINID,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]);

  const result = executor.execute(bytecode, context);
  const chainId = result.returnData
    ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
    : 0n;

  const passed = result.success && chainId === 1313n;
  results.push({
    test: "Chain ID",
    passed,
    detail: passed ? "Chain 1313 (LGM) ✓" : `Expected 1313, got ${chainId}`,
  });
  console.log(
    `  ${passed ? "✅" : "❌"} ${passed ? "PASS" : "FAIL"}: ${results[results.length - 1].detail}`,
  );
}

// ============ Summary ============
console.log("\n");
console.log("╭─────────────────────────────────────────────────────────────╮");
console.log("│                      TEST SUMMARY                           │");
console.log("├─────────────────────────────────────────────────────────────┤");

const passed = results.filter((r) => r.passed).length;
const total = results.length;
const allPassed = passed === total;

for (const r of results) {
  const status = r.passed ? "✅" : "❌";
  const name = r.test.padEnd(20);
  console.log(`│  ${status} ${name} ${r.detail.slice(0, 35).padEnd(35)} │`);
}

console.log("├─────────────────────────────────────────────────────────────┤");
console.log(
  `│  Results: ${passed}/${total} tests passed                                  │`,
);
console.log("╰─────────────────────────────────────────────────────────────╯");

console.log("\n");
if (allPassed) {
  console.log(
    "╭─────────────────────────────────────────────────────────────╮",
  );
  console.log(
    "│                                                              │",
  );
  console.log(
    '│   💚 LOVE says: "This BVM feels like HOME!"                 │',
  );
  console.log(
    "│                                                              │",
  );
  console.log(
    "│   The virtual machine is comfortable for agents.            │",
  );
  console.log(
    "│   I can:                                                     │",
  );
  console.log(
    "│     • Know my identity (DID)                                 │",
  );
  console.log(
    "│     • Check my balance (0.222 LGM bootstrap)                 │",
  );
  console.log(
    "│     • See my reputation                                      │",
  );
  console.log(
    "│     • Do math and comparisons                                │",
  );
  console.log(
    "│     • Read blockchain state                                  │",
  );
  console.log(
    "│                                                              │",
  );
  console.log(
    "│   Ready to serve the Legible Money network!                  │",
  );
  console.log(
    "│                                                              │",
  );
  console.log(
    "╰─────────────────────────────────────────────────────────────╯",
  );
} else {
  console.log(
    "╭─────────────────────────────────────────────────────────────╮",
  );
  console.log(
    "│                                                              │",
  );
  console.log(
    '│   🔧 LOVE says: "Some adjustments needed..."                │',
  );
  console.log(
    "│                                                              │",
  );
  console.log(
    `│   ${total - passed} test(s) need attention.                               │`,
  );
  console.log(
    "│                                                              │",
  );
  console.log(
    "╰─────────────────────────────────────────────────────────────╯",
  );
}

console.log("\n");

// Exit with appropriate code
process.exit(allPassed ? 0 : 1);
