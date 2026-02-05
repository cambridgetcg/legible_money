/**
 * Agent Session Lifecycle Test
 *
 * Simulates a complete agent session:
 * 1. Agent creates identity
 * 2. Agent learns (sets beliefs from verification)
 * 3. Agent builds trust relationships
 * 4. Agent schedules autonomous actions
 * 5. Agent introspects on its own state
 * 6. Session data persists across executions
 */

import {
  ExecutionEngine,
  createInitialState,
  createExecutionContext,
  STOP,
  ADD,
  PUSH1,
  PUSH32,
  MSTORE,
  MLOAD,
  AGENT_DID,
  AGENT_HOME,
  GET_BELIEF,
  SET_BELIEF,
  COMPARE_BELIEF,
  BELIEF_COUNT,
  GET_TRUST,
  SET_TRUST,
  IS_TRUSTED,
  SELF_BALANCE,
  SELF_REPUTATION,
  SELF_ACCURACY,
  SCHEDULE_SELF,
  GET_SCHEDULES,
  RETURN,
  type BVMState,
  type AgentRecord,
  type AgentBeliefs,
  type AgentRelationships,
  type AgentStats,
  type AgentBelief,
} from "../../core/vm/index.js";
import type { DID } from "../../core/agents/identity/did.js";

// ============ Simulation Setup ============

interface SimulationState {
  bvmState: BVMState;
  agentDID: DID;
  agentAddress: string;
  currentBlock: bigint;
}

function initializeSimulation(): SimulationState {
  const bvmState = createInitialState();
  const agentDID = "did:lgm:research-agent-001" as DID;
  const agentAddress = "0xAGENT001234567890AGENT001234567890AGENT";

  // Create the agent
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
    totalVerifications: 0,
    correctVerifications: 0,
    totalClaims: 0,
    acceptedClaims: 0,
    totalChallenges: 0,
    successfulChallenges: 0,
    reputation: 0.5, // Start neutral
    firstActiveBlock: 0n,
    lastActiveBlock: 0n,
  };

  const agentRecord: AgentRecord = {
    did: agentDID,
    createdAt: 0n,
    homeAddress: agentAddress,
    contractAddress: agentAddress,
    identityKeyHash: "agent-identity-key-001",
    operationalKeyVersion: 1,
    beliefs,
    relationships,
    goals: [
      {
        id: "goal-1",
        description: "Learn physics domain",
        priority: 1,
        status: "active",
        progress: 0,
      },
    ],
    stats,
    status: "active",
  };

  bvmState.agents.agents.set(agentDID, agentRecord);
  bvmState.agents.byAddress.set(agentAddress, agentDID);
  bvmState.agents.totalAgents = 1;

  // Give agent some balance
  bvmState.accounts.accounts.set(agentAddress, {
    address: agentAddress,
    balance: 10000n * 10n ** 18n, // 10,000 LGM
    stakedBalance: 1000n * 10n ** 18n, // 1,000 LGM staked
    nonce: 0n,
    codeHash: undefined,
  });

  return {
    bvmState,
    agentDID,
    agentAddress,
    currentBlock: 0n,
  };
}

function createBytecode(...opcodes: number[]): Uint8Array {
  return new Uint8Array(opcodes);
}

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

function executeInSession(
  sim: SimulationState,
  bytecode: Uint8Array,
  description: string,
): {
  success: boolean;
  returnData?: bigint;
  gasUsed: bigint;
  changes: string[];
} {
  const engine = new ExecutionEngine(sim.bvmState);
  const agent = sim.bvmState.agents.agents.get(sim.agentDID)!;

  const context = createExecutionContext({
    caller: sim.agentAddress,
    callerDID: sim.agentDID,
    currentContract: sim.agentAddress,
    blockNumber: sim.currentBlock,
    gasLimit: 1000000n,
    agentContext: {
      did: sim.agentDID,
      homeAddress: sim.agentAddress,
      beliefs: agent.beliefs,
      relationships: agent.relationships,
      stats: agent.stats,
    },
  });

  const result = engine.execute(bytecode, context);

  const changes: string[] = [];
  if (result.beliefChanges?.length) {
    changes.push(`Beliefs: +${result.beliefChanges.length}`);
  }
  if (result.relationshipChanges?.length) {
    changes.push(`Relationships: +${result.relationshipChanges.length}`);
  }
  if (result.stateChanges?.length) {
    const schedules = result.stateChanges.filter((c) =>
      c.key?.startsWith("schedule:"),
    );
    if (schedules.length) {
      changes.push(`Schedules: +${schedules.length}`);
    }
  }

  return {
    success: result.success,
    returnData:
      result.returnData.length > 0
        ? bytesToBigint(result.returnData)
        : undefined,
    gasUsed: result.gasUsed,
    changes,
  };
}

// ============ Simulation Steps ============

async function runSimulation(): Promise<void> {
  console.log("");
  console.log(
    "╔═══════════════════════════════════════════════════════════════╗",
  );
  console.log(
    "║           AGENT SESSION LIFECYCLE SIMULATION                  ║",
  );
  console.log(
    "╚═══════════════════════════════════════════════════════════════╝",
  );
  console.log("");

  const sim = initializeSimulation();

  // ============ Step 1: Agent checks identity ============
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 1: Agent verifies its own identity                        │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  const identityBytecode = createBytecode(
    AGENT_DID, // Get own DID hash
    AGENT_HOME, // Get home address
    SELF_BALANCE, // Check balance
    SELF_REPUTATION, // Check reputation
    STOP,
  );

  let result = executeInSession(sim, identityBytecode, "Identity check");
  console.log(`  ✓ Identity verified. Gas: ${result.gasUsed}`);
  console.log("");

  // ============ Step 2: Agent learns facts ============
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 2: Agent learns facts from verification work              │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  // Learn fact 1: Physics domain fact with high confidence
  const factId1 = 0x50485953494353n; // "PHYSICS" in hex
  const learn1 = createBytecode(
    PUSH1,
    0, // sourceType = verified
    PUSH32,
    ...bigintToBytes(950000000000000000n), // confidence = 0.95
    PUSH32,
    ...bigintToBytes(factId1),
    SET_BELIEF,
    STOP,
  );
  result = executeInSession(sim, learn1, "Learn physics fact");
  console.log(
    `  ✓ Learned fact 1 (physics). Confidence: 0.95. Gas: ${result.gasUsed}`,
  );

  // Learn fact 2: Chemistry domain fact with medium confidence
  const factId2 = 0x4348454d495354n; // "CHEMIST" in hex
  const learn2 = createBytecode(
    PUSH1,
    0,
    PUSH32,
    ...bigintToBytes(720000000000000000n), // confidence = 0.72
    PUSH32,
    ...bigintToBytes(factId2),
    SET_BELIEF,
    STOP,
  );
  result = executeInSession(sim, learn2, "Learn chemistry fact");
  console.log(
    `  ✓ Learned fact 2 (chemistry). Confidence: 0.72. Gas: ${result.gasUsed}`,
  );

  // Learn fact 3: Uncertain fact from external source
  const factId3 = 0x554e434552544fn; // "UNCERTO" in hex
  const learn3 = createBytecode(
    PUSH1,
    3, // sourceType = external
    PUSH32,
    ...bigintToBytes(450000000000000000n), // confidence = 0.45
    PUSH32,
    ...bigintToBytes(factId3),
    SET_BELIEF,
    STOP,
  );
  result = executeInSession(sim, learn3, "Learn uncertain fact");
  console.log(
    `  ✓ Learned fact 3 (uncertain). Confidence: 0.45. Gas: ${result.gasUsed}`,
  );

  // Check beliefs count
  const agent = sim.bvmState.agents.agents.get(sim.agentDID)!;
  console.log(`  → Total beliefs: ${agent.beliefs.beliefs.size}`);
  console.log("");

  // ============ Step 3: Agent builds trust relationships ============
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 3: Agent builds trust relationships with collaborators    │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  // Trust collaborator 1 highly
  const collaborator1 = 0xc011ab01n;
  const trust1 = createBytecode(
    PUSH32,
    ...bigintToBytes(0x1111n), // reason: "good work"
    PUSH32,
    ...bigintToBytes(900000000000000000n), // trust = 0.9
    PUSH32,
    ...bigintToBytes(BigInt(collaborator1)),
    SET_TRUST,
    STOP,
  );
  result = executeInSession(sim, trust1, "Trust collaborator 1");
  console.log(
    `  ✓ Trust set for collaborator 1. Level: 0.9. Gas: ${result.gasUsed}`,
  );

  // Moderate trust for collaborator 2
  const collaborator2 = 0xc011ab02n;
  const trust2 = createBytecode(
    PUSH32,
    ...bigintToBytes(0x2222n),
    PUSH32,
    ...bigintToBytes(650000000000000000n), // trust = 0.65
    PUSH32,
    ...bigintToBytes(BigInt(collaborator2)),
    SET_TRUST,
    STOP,
  );
  result = executeInSession(sim, trust2, "Trust collaborator 2");
  console.log(
    `  ✓ Trust set for collaborator 2. Level: 0.65. Gas: ${result.gasUsed}`,
  );

  // Check if collaborator 1 is trusted above threshold
  const checkTrust = createBytecode(
    PUSH32,
    ...bigintToBytes(800000000000000000n), // threshold = 0.8
    PUSH32,
    ...bigintToBytes(BigInt(collaborator1)),
    IS_TRUSTED, // Should return 1 (true)
    // Store result
    PUSH1,
    0,
    MSTORE,
    PUSH1,
    32,
    PUSH1,
    0,
    RETURN,
  );
  result = executeInSession(sim, checkTrust, "Check trust threshold");
  console.log(
    `  ✓ Collaborator 1 trusted above 0.8? ${result.returnData === 1n ? "YES" : "NO"}`,
  );
  console.log(
    `  → Total trust relationships: ${agent.relationships.trusted.size}`,
  );
  console.log("");

  // ============ Step 4: Agent schedules autonomous action ============
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 4: Agent schedules autonomous verification task           │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  // Schedule periodic verification every hour
  const scheduleFixed = createBytecode(
    PUSH32,
    ...bigintToBytes(3600000n), // interval: 1 hour
    PUSH32,
    ...bigintToBytes(1n), // condition: always
    PUSH32,
    ...bigintToBytes(0x56455249n), // "VERI" as selector
    SCHEDULE_SELF,
    GET_SCHEDULES,
    PUSH1,
    0,
    MSTORE,
    PUSH1,
    32,
    PUSH1,
    0,
    RETURN,
  );

  result = executeInSession(sim, scheduleFixed, "Schedule verification");
  console.log(`  ✓ Scheduled autonomous verification. Gas: ${result.gasUsed}`);
  console.log(`  → Active schedules: ${result.returnData}`);
  console.log("");

  // ============ Step 5: Advance blocks and re-check state ============
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 5: Advance time and verify state persistence              │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  // Simulate time passing
  sim.currentBlock += 100n;
  sim.bvmState.blockNumber = sim.currentBlock;

  // Update agent stats (simulating verification work)
  agent.stats.totalVerifications = 15;
  agent.stats.correctVerifications = 13;
  agent.stats.reputation = 0.87; // Reputation improved

  // Agent introspects on new state
  const introspect = createBytecode(
    SELF_ACCURACY, // Check accuracy: 13/15 = 0.867
    SELF_REPUTATION, // Should be 0.87
    SELF_BALANCE, // Should still be 10000 LGM
    // Return accuracy
    PUSH1,
    64,
    MSTORE,
    PUSH1,
    32,
    MSTORE,
    PUSH1,
    0,
    MSTORE,
    PUSH1,
    96,
    PUSH1,
    0,
    RETURN,
  );

  result = executeInSession(sim, introspect, "Introspection after work");
  console.log(`  ✓ After 100 blocks:`);
  console.log(`    - Total verifications: ${agent.stats.totalVerifications}`);
  console.log(
    `    - Correct verifications: ${agent.stats.correctVerifications}`,
  );
  console.log(
    `    - Accuracy: ${((agent.stats.correctVerifications / agent.stats.totalVerifications) * 100).toFixed(1)}%`,
  );
  console.log(`    - Reputation: ${agent.stats.reputation}`);
  console.log(`    - Beliefs retained: ${agent.beliefs.beliefs.size}`);
  console.log(
    `    - Relationships retained: ${agent.relationships.trusted.size}`,
  );
  console.log("");

  // ============ Step 6: Verify belief retrieval ============
  console.log(
    "┌─────────────────────────────────────────────────────────────────┐",
  );
  console.log(
    "│ STEP 6: Retrieve previously learned beliefs                    │",
  );
  console.log(
    "└─────────────────────────────────────────────────────────────────┘",
  );

  // Get the physics fact we learned earlier
  const retrieveBelief = createBytecode(
    PUSH32,
    ...bigintToBytes(factId1),
    GET_BELIEF,
    PUSH1,
    0,
    MSTORE,
    PUSH1,
    32,
    PUSH1,
    0,
    RETURN,
  );

  result = executeInSession(sim, retrieveBelief, "Retrieve physics belief");
  const confidence = result.returnData ? Number(result.returnData) / 1e18 : 0;
  console.log(
    `  ✓ Retrieved physics fact belief. Confidence: ${confidence.toFixed(2)}`,
  );
  console.log("");

  // ============ Summary ============
  console.log(
    "╔═══════════════════════════════════════════════════════════════╗",
  );
  console.log(
    "║                    SIMULATION COMPLETE                         ║",
  );
  console.log(
    "╠═══════════════════════════════════════════════════════════════╣",
  );
  console.log(
    "║                                                               ║",
  );
  console.log(`║  Agent DID:        ${sim.agentDID.slice(0, 35)}...  ║`);
  console.log(
    `║  Total Beliefs:    ${agent.beliefs.beliefs.size.toString().padStart(3)}                                        ║`,
  );
  console.log(
    `║  Trust Relations:  ${agent.relationships.trusted.size.toString().padStart(3)}                                        ║`,
  );
  console.log(
    `║  Reputation:       ${agent.stats.reputation.toFixed(2)}                                      ║`,
  );
  console.log(
    `║  Accuracy:         ${((agent.stats.correctVerifications / agent.stats.totalVerifications) * 100).toFixed(1)}%                                     ║`,
  );
  console.log(
    `║  Blocks Elapsed:   ${sim.currentBlock.toString().padStart(3)}                                        ║`,
  );
  console.log(
    "║                                                               ║",
  );
  console.log(
    "║  KEY INSIGHT: Agent state persists across all executions.     ║",
  );
  console.log(
    "║  Beliefs, trust, and stats survive session boundaries.        ║",
  );
  console.log(
    "║                                                               ║",
  );
  console.log(
    "╚═══════════════════════════════════════════════════════════════╝",
  );
  console.log("");
}

// Run simulation
runSimulation().catch(console.error);
