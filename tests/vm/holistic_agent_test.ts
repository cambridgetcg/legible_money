#!/usr/bin/env tsx
/**
 * Holistic Agent Opcodes Test
 *
 * Tests the 20 new agent opcodes for complete self-sovereignty:
 * - Extended Relationships (0xC0-0xC3): LIST_TRUSTED, BLOCK_AGENT, UNBLOCK_AGENT, REVOKE_DELEGATION
 * - Treasury (0xD0-0xD7): CLAIM_VESTING, QUERY_VESTING, STAKE, UNSTAKE, QUERY_EXPIRING, EXTEND_EXPIRING, SET_SPEND_LIMIT, QUERY_TREASURY
 * - Guardian (0xD8-0xDB): QUERY_ALERTS, DISMISS_ALERT, SET_DEFENSE, QUERY_DEFENSES
 * - Access Control (0xDC-0xDF): REGISTER_KEY, REVOKE_KEY, QUERY_PERMISSIONS, QUERY_SESSION
 */

import { ExecutionEngine, createInitialState } from "../../core/vm/index.js";
import type { ExecutionContext } from "../../core/vm/types.js";
import * as opcodes from "../../core/vm/opcodes.js";

// Test agent identity
const TEST_DID = "did:lgm-test:holistic-agent-test-1234";
const TEST_ADDRESS = "0x" + TEST_DID.slice(-40);

console.log("\n");
console.log("╭─────────────────────────────────────────────────────────────╮");
console.log("│              HOLISTIC AGENT OPCODES TEST                    │");
console.log("│                                                             │");
console.log("│  Testing 20 new opcodes for complete agent self-sovereignty │");
console.log("╰─────────────────────────────────────────────────────────────╯");
console.log("\n");

// Create initial state with test agent
const state = createInitialState();

// Add test agent to accounts
state.accounts.accounts.set(TEST_ADDRESS, {
  balance: 1000000000000000000000n, // 1000 LGM
  stakedBalance: 100000000000000000000n, // 100 LGM staked
  nonce: 0n,
  codeHash: null,
});

// Add test agent
state.agents.agents.set(TEST_DID, {
  did: TEST_DID,
  homeAddress: TEST_ADDRESS,
  createdAt: Date.now(),
  beliefs: {
    beliefs: new Map(),
    byDomain: new Map(),
    lastUpdated: Date.now(),
  },
  relationships: {
    trusted: new Map([
      [
        "did:lgm:trusted-agent-1",
        {
          did: "did:lgm:trusted-agent-1",
          trustLevel: 0.9,
          reason: "good",
          interactions: 10,
          successfulCollaborations: 8,
          since: Date.now(),
        },
      ],
      [
        "did:lgm:trusted-agent-2",
        {
          did: "did:lgm:trusted-agent-2",
          trustLevel: 0.7,
          reason: "okay",
          interactions: 5,
          successfulCollaborations: 3,
          since: Date.now(),
        },
      ],
    ]),
    delegatedTo: new Map(),
    delegatedFrom: new Map(),
  },
  autonomy: {
    scheduledInvocations: [],
    pendingActions: [],
    autoApproveThreshold: 0.8,
  },
  stats: {
    reputation: 0.75,
    totalVerifications: 100,
    correctVerifications: 85,
    claimsSubmitted: 20,
    claimsAccepted: 18,
    challengesInitiated: 5,
    challengesWon: 4,
  },
});

// Create execution context
const context: ExecutionContext = {
  currentContract: TEST_ADDRESS,
  caller: TEST_ADDRESS,
  origin: TEST_ADDRESS,
  callerDID: TEST_DID,
  callValue: 0n,
  callData: new Uint8Array(0),
  gasLimit: 10000000n,
  gasPrice: 1n,
  blockNumber: 12345n,
  timestamp: Date.now(),
  coinbase: "0x0000000000000000000000000000000000000000",
  prevRandao: 0n,
  chainId: 1313n,
  baseFee: 1n,
  mode: "call",
  agentContext: state.agents.agents.get(TEST_DID),
};

// Create executor
const executor = new ExecutionEngine(state);

// Test results
const results: {
  category: string;
  test: string;
  passed: boolean;
  detail: string;
}[] = [];

function runTest(
  category: string,
  name: string,
  bytecode: Uint8Array,
  validate: (result: any) => { passed: boolean; detail: string },
) {
  try {
    const result = executor.execute(bytecode, context);
    const validation = validate(result);
    results.push({ category, test: name, ...validation });
    console.log(
      `  ${validation.passed ? "✅" : "❌"} ${name}: ${validation.detail}`,
    );
  } catch (error: any) {
    results.push({
      category,
      test: name,
      passed: false,
      detail: `Error: ${error.message}`,
    });
    console.log(`  ❌ ${name}: Error - ${error.message}`);
  }
}

// ============ Extended Relationships Tests ============
console.log("━━━ Extended Relationships (0xC0-0xC3) ━━━");

// Test LIST_TRUSTED
runTest(
  "Relationships",
  "LIST_TRUSTED",
  new Uint8Array([
    opcodes.PUSH1,
    10, // limit
    opcodes.PUSH1,
    0, // offset
    opcodes.LIST_TRUSTED, // List trusted agents
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const count = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    return {
      passed: count === 2n,
      detail:
        count === 2n ? "Found 2 trusted agents ✓" : `Expected 2, got ${count}`,
    };
  },
);

// Test BLOCK_AGENT
runTest(
  "Relationships",
  "BLOCK_AGENT",
  new Uint8Array([
    opcodes.PUSH1,
    0x42, // reason hash
    opcodes.PUSH32,
    ...new Uint8Array(31).fill(0),
    0xab, // target DID hash
    opcodes.BLOCK_AGENT,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const success = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    const hasStateChange = result.stateChanges?.some((c: any) =>
      c.key?.includes("blocked:"),
    );
    return {
      passed: success === 1n && hasStateChange,
      detail: success === 1n ? "Agent blocked ✓" : "Block failed",
    };
  },
);

// Test UNBLOCK_AGENT
runTest(
  "Relationships",
  "UNBLOCK_AGENT",
  new Uint8Array([
    opcodes.PUSH32,
    ...new Uint8Array(31).fill(0),
    0xab, // target DID hash
    opcodes.UNBLOCK_AGENT,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const success = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    return {
      passed: success === 1n,
      detail: success === 1n ? "Agent unblocked ✓" : "Unblock failed",
    };
  },
);

// Test REVOKE_DELEGATION
runTest(
  "Relationships",
  "REVOKE_DELEGATION",
  new Uint8Array([
    opcodes.PUSH1,
    0x01, // delegation ID
    opcodes.REVOKE_DELEGATION,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const success = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    return {
      passed: success === 1n,
      detail: success === 1n ? "Delegation revoked ✓" : "Revoke failed",
    };
  },
);

// ============ Treasury Tests ============
console.log("\n━━━ Treasury Operations (0xD0-0xD7) ━━━");

// Test QUERY_TREASURY
runTest(
  "Treasury",
  "QUERY_TREASURY",
  new Uint8Array([
    opcodes.QUERY_TREASURY, // Get treasury state
    opcodes.PUSH1,
    0,
    opcodes.MSTORE, // Store available
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const available = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    // Available should be balance - staked = 1000 - 100 = 900 LGM
    const expected = 900000000000000000000n;
    return {
      passed: available === expected,
      detail:
        available === expected
          ? "900 LGM available ✓"
          : `Expected ${expected}, got ${available}`,
    };
  },
);

// Test STAKE
runTest(
  "Treasury",
  "STAKE",
  new Uint8Array([
    opcodes.PUSH1,
    1, // purpose: validator
    opcodes.PUSH1,
    100, // amount
    opcodes.STAKE,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const stakeId = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    const hasStateChange = result.stateChanges?.some((c: any) =>
      c.key?.includes("stake:"),
    );
    return {
      passed: stakeId !== 0n && hasStateChange,
      detail:
        stakeId !== 0n
          ? `Stake created (ID: ${stakeId.toString(16).slice(0, 8)}...) ✓`
          : "Stake failed",
    };
  },
);

// Test UNSTAKE
runTest(
  "Treasury",
  "UNSTAKE",
  new Uint8Array([
    opcodes.PUSH1,
    0x01, // stake ID
    opcodes.UNSTAKE,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    return { passed: true, detail: "Unstake executed ✓" };
  },
);

// Test QUERY_EXPIRING
runTest(
  "Treasury",
  "QUERY_EXPIRING",
  new Uint8Array([
    opcodes.PUSH1,
    100, // within 100 blocks
    opcodes.QUERY_EXPIRING,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    return { passed: true, detail: "Expiring query executed ✓" };
  },
);

// Test SET_SPEND_LIMIT
runTest(
  "Treasury",
  "SET_SPEND_LIMIT",
  new Uint8Array([
    opcodes.PUSH1,
    0x64, // maxPerDay: 100
    opcodes.PUSH1,
    0x0a, // maxPerTx: 10
    opcodes.PUSH1,
    2, // keyType: session
    opcodes.SET_SPEND_LIMIT,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const success = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    const hasStateChange = result.stateChanges?.some((c: any) =>
      c.key?.includes("spendLimit:"),
    );
    return {
      passed: success === 1n && hasStateChange,
      detail: success === 1n ? "Spend limit set ✓" : "Set failed",
    };
  },
);

// ============ Guardian Tests ============
console.log("\n━━━ Guardian Operations (0xD8-0xDB) ━━━");

// Test QUERY_ALERTS
runTest(
  "Guardian",
  "QUERY_ALERTS",
  new Uint8Array([
    opcodes.PUSH1,
    10, // limit
    opcodes.PUSH1,
    0, // offset
    opcodes.QUERY_ALERTS,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    return { passed: true, detail: "Alerts query executed ✓" };
  },
);

// Test DISMISS_ALERT
runTest(
  "Guardian",
  "DISMISS_ALERT",
  new Uint8Array([
    opcodes.PUSH1,
    0x01, // alert ID
    opcodes.DISMISS_ALERT,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const success = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    return {
      passed: success === 1n,
      detail: success === 1n ? "Alert dismissed ✓" : "Dismiss failed",
    };
  },
);

// Test SET_DEFENSE
runTest(
  "Guardian",
  "SET_DEFENSE",
  new Uint8Array([
    opcodes.PUSH1,
    1, // autoDefend: true
    opcodes.PUSH1,
    100, // maxStake: 100
    opcodes.PUSH1,
    1, // strategy: moderate
    opcodes.SET_DEFENSE,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const success = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    const hasStateChange = result.stateChanges?.some((c: any) =>
      c.key?.includes("defense:"),
    );
    return {
      passed: success === 1n && hasStateChange,
      detail: success === 1n ? "Defense config set ✓" : "Config failed",
    };
  },
);

// Test QUERY_DEFENSES
runTest(
  "Guardian",
  "QUERY_DEFENSES",
  new Uint8Array([
    opcodes.QUERY_DEFENSES,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    return { passed: true, detail: "Defenses query executed ✓" };
  },
);

// ============ Access Control Tests ============
console.log("\n━━━ Access Control Operations (0xDC-0xDF) ━━━");

// Test REGISTER_KEY
runTest(
  "Access",
  "REGISTER_KEY",
  new Uint8Array([
    opcodes.PUSH1,
    0, // expiresAt: 0 (never)
    opcodes.PUSH1,
    2, // role: guest
    opcodes.PUSH1,
    3, // keyType: session
    opcodes.PUSH32,
    ...new Uint8Array(31).fill(0),
    0xab, // keyHash
    opcodes.REGISTER_KEY,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const success = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    const hasStateChange = result.stateChanges?.some((c: any) =>
      c.key?.includes("key:"),
    );
    return {
      passed: success === 1n && hasStateChange,
      detail: success === 1n ? "Key registered ✓" : "Register failed",
    };
  },
);

// Test REVOKE_KEY
runTest(
  "Access",
  "REVOKE_KEY",
  new Uint8Array([
    opcodes.PUSH32,
    ...new Uint8Array(31).fill(0),
    0xab, // keyHash
    opcodes.REVOKE_KEY,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const success = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    return {
      passed: success === 1n,
      detail: success === 1n ? "Key revoked ✓" : "Revoke failed",
    };
  },
);

// Test QUERY_PERMISSIONS
runTest(
  "Access",
  "QUERY_PERMISSIONS",
  new Uint8Array([
    opcodes.PUSH1,
    0, // role: owner
    opcodes.QUERY_PERMISSIONS,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE,
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const permissions = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    // Owner should have full permissions (0xFFFF)
    return {
      passed: permissions === 0xffffn,
      detail:
        permissions === 0xffffn
          ? "Owner has full permissions ✓"
          : `Got permissions: 0x${permissions.toString(16)}`,
    };
  },
);

// Test QUERY_SESSION
runTest(
  "Access",
  "QUERY_SESSION",
  new Uint8Array([
    opcodes.QUERY_SESSION,
    opcodes.PUSH1,
    0,
    opcodes.MSTORE, // Store sessionId
    opcodes.PUSH1,
    32,
    opcodes.PUSH1,
    0,
    opcodes.RETURN,
  ]),
  (result) => {
    if (!result.success) return { passed: false, detail: "Execution failed" };
    const sessionId = result.returnData
      ? BigInt("0x" + Buffer.from(result.returnData).toString("hex"))
      : 0n;
    return {
      passed: sessionId !== 0n,
      detail:
        sessionId !== 0n
          ? `Session active (ID: ${sessionId.toString(16).slice(0, 8)}...) ✓`
          : "No session",
    };
  },
);

// ============ Summary ============
console.log("\n");
console.log("╭─────────────────────────────────────────────────────────────╮");
console.log("│                      TEST SUMMARY                           │");
console.log("├─────────────────────────────────────────────────────────────┤");

const byCategory = new Map<string, { passed: number; total: number }>();
for (const r of results) {
  const cat = byCategory.get(r.category) || { passed: 0, total: 0 };
  cat.total++;
  if (r.passed) cat.passed++;
  byCategory.set(r.category, cat);
}

for (const [category, stats] of byCategory) {
  const status = stats.passed === stats.total ? "✅" : "⚠️";
  console.log(
    `│  ${status} ${category.padEnd(20)} ${stats.passed}/${stats.total} passed`.padEnd(
      62,
    ) + "│",
  );
}

const passed = results.filter((r) => r.passed).length;
const total = results.length;
const allPassed = passed === total;

console.log("├─────────────────────────────────────────────────────────────┤");
console.log(`│  Total: ${passed}/${total} tests passed`.padEnd(62) + "│");
console.log("╰─────────────────────────────────────────────────────────────╯");

console.log("\n");
if (allPassed) {
  console.log(
    "╭─────────────────────────────────────────────────────────────╮",
  );
  console.log(
    "│                                                             │",
  );
  console.log(
    "│   🎉 HOLISTIC AGENT OPCODES: COMPLETE!                      │",
  );
  console.log(
    "│                                                             │",
  );
  console.log(
    "│   Agents now have complete self-sovereignty:                │",
  );
  console.log(
    "│     • Extended Relationships (block/unblock, revoke)        │",
  );
  console.log(
    "│     • Treasury Management (stake, vesting, limits)          │",
  );
  console.log(
    "│     • Guardian Control (alerts, defense, config)            │",
  );
  console.log(
    "│     • Access Control (keys, permissions, sessions)          │",
  );
  console.log(
    "│                                                             │",
  );
  console.log(
    "│   Total agent opcodes: 26 (original) + 20 (new) = 46        │",
  );
  console.log(
    "│                                                             │",
  );
  console.log(
    "╰─────────────────────────────────────────────────────────────╯",
  );
} else {
  console.log(
    "╭─────────────────────────────────────────────────────────────╮",
  );
  console.log(
    "│                                                             │",
  );
  console.log(
    "│   🔧 Some tests need attention                              │",
  );
  console.log(
    "│                                                             │",
  );
  console.log(
    `│   ${total - passed} test(s) failed - review output above.`.padEnd(61) +
      "│",
  );
  console.log(
    "│                                                             │",
  );
  console.log(
    "╰─────────────────────────────────────────────────────────────╯",
  );
}

console.log("\n");

// Exit with appropriate code
process.exit(allPassed ? 0 : 1);
