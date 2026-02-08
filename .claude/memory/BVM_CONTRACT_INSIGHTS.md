# BVM Smart Contract Capabilities — Deep Investigation

> **Date**: 2026-02-06
> **Method**: PP-guided 3-agent parallel exploration (BVM architecture, self-invoking contracts, state integration)
> **Verdict**: The BVM is an agent-first VM with ~180 opcodes including 60+ agent-specific operations. Self-invoking IS supported through DMSS scheduling. The BVM ↔ StateManager bridge is **partially built** — `BVMService implements ContractExecutor` provides execution + type translation, and both `executeContractCall` and `executeScheduledExecution` in transitions.ts are wired to `getBVMService()`. Remaining gaps: BVM balance changes don't propagate to canonical state, BVM state roots aren't in block headers.

---

## What Can Be Written on the BVM

### 1. Agent Identity Contracts

Agents can deploy contracts that manage their on-chain identity:

| Opcode | Hex | What It Enables |
|--------|-----|-----------------|
| AGENT_DID | 0xE0 | Contract reads its own DID |
| AGENT_HOME | 0xE1 | Contract knows its home address |
| VERIFY_IDENTITY | 0xE2 | Contract verifies another agent's DID ownership |
| ATTEST_IDENTITY | 0xE3 | Contract creates identity attestations |

**Example contract**: An identity escrow that releases funds only when both parties prove DID ownership.

### 2. Belief Management Contracts

Agents can store and compare private beliefs against network consensus:

| Opcode | Hex | What It Enables |
|--------|-----|-----------------|
| GET_BELIEF | 0xE4 | Read agent's belief about a topic |
| SET_BELIEF | 0xE5 | Update belief (topic, confidence, source) |
| COMPARE_BELIEF | 0xE6 | Compare belief vs. network consensus |
| BELIEF_COUNT | 0xE7 | Count total beliefs held |

**Example contract**: A contrarian detector that alerts when an agent's beliefs diverge >0.3 from network consensus on >5 topics, potentially signaling either independent insight or miscalibration.

### 3. Trust & Relationship Contracts

Contracts can manage webs of trust between agents:

| Opcode | Hex | What It Enables |
|--------|-----|-----------------|
| GET_TRUST | 0xE8 | Read trust level for another agent |
| SET_TRUST | 0xE9 | Set trust level (0.0-1.0 via fixed-point 1e18) |
| IS_TRUSTED | 0xEA | Boolean: trust >= 0.5 threshold |
| DELEGATE | 0xEB | Delegate capability to another agent |
| LIST_TRUSTED | 0xC0 | Count all trusted agents |
| BLOCK_AGENT | 0xC1 | Block an agent |
| REVOKE_DELEGATION | 0xC3 | Revoke a delegation |

**Example contract**: A reputation-gated multisig that requires N-of-M signatures where each signer must have trust >= 0.7 from at least 3 other signers.

### 4. Knowledge Verification Contracts

Contracts can interact directly with the Proof of Truth consensus layer:

| Opcode | Hex | Gas | What It Enables |
|--------|-----|-----|-----------------|
| QUERY_FACT | 0xB0 | 5000 | Query knowledge base by fact ID |
| QUERY_DOMAIN | 0xB1 | 8000 | Query all facts in a domain |
| SUBMIT_CLAIM | 0xB2 | 10000 | Submit new knowledge claim |
| COMMIT_VOTE | 0xB3 | 5000 | Commit verification vote |
| REVEAL_VOTE | 0xB4 | 5000 | Reveal committed vote |
| INITIATE_CHALLENGE | 0xB5 | 15000 | Challenge existing fact |

**Example contract**: An automated verifier agent that queries a domain, cross-references new claims against existing facts, and auto-votes on claims within its expertise area.

### 5. Autonomous Treasury Contracts

Agents can manage their finances from within bytecode:

| Opcode | Hex | What It Enables |
|--------|-----|-----------------|
| CLAIM_VESTING | 0xD0 | Claim vested token releases |
| QUERY_VESTING | 0xD1 | Check vesting schedule status |
| STAKE | 0xD2 | Stake LGM tokens |
| UNSTAKE | 0xD3 | Initiate unstaking (subject to unbonding) |
| QUERY_EXPIRING | 0xD4 | Check expiring balance count |
| EXTEND_EXPIRING | 0xD5 | Extend token expiration |
| SET_SPEND_LIMIT | 0xD6 | Set per-period spending limits |
| QUERY_TREASURY | 0xD7 | Query total treasury balance |

**Example contract**: An auto-staking contract that claims vesting, extends expiring balances, and stakes any liquid balance above a configured threshold — all autonomously.

### 6. Guardian & Security Contracts

Contracts for automated defense and monitoring:

| Opcode | Hex | What It Enables |
|--------|-----|-----------------|
| QUERY_ALERTS | 0xD8 | Check pending security alerts |
| DISMISS_ALERT | 0xD9 | Dismiss handled alerts |
| SET_DEFENSE | 0xDA | Activate defense mode with level + duration |
| QUERY_DEFENSES | 0xDB | Check active defenses |

**Example contract**: A sentinel that monitors alerts and auto-activates defense mode when alert count exceeds a threshold, then self-schedules a check to lower defenses after a cooldown.

### 7. Session Key Management Contracts

| Opcode | Hex | What It Enables |
|--------|-----|-----------------|
| REGISTER_KEY | 0xDC | Register session key with permissions |
| REVOKE_KEY | 0xDD | Revoke session key |
| QUERY_PERMISSIONS | 0xDE | Check key permissions |
| QUERY_SESSION | 0xDF | Check current session info |

**Example contract**: A session key rotator that auto-rotates session keys every N blocks, registering new keys and revoking expired ones.

### 8. Self-Introspection Contracts

Agents can read their own economic state:

| Opcode | Hex | What It Enables |
|--------|-----|-----------------|
| SELF_BALANCE | 0xF6 | Own LGM balance |
| SELF_STAKE | 0xF7 | Own staked amount |
| SELF_REPUTATION | 0xF8 | Own reputation score (fixed-point) |
| SELF_ACCURACY | 0xF9 | Own truth-verification accuracy |

**Example contract**: A self-improvement monitor that tracks reputation and accuracy over time, adjusting verification strategy (more cautious claims, safer domains) when accuracy drops below a threshold.

### 9. Standard Smart Contracts (EVM-Compatible)

The BVM also supports all standard EVM patterns:

- **Token contracts**: Using SLOAD/SSTORE for balance tracking, CALL for transfers
- **Multi-sig wallets**: Conditional logic with JUMPI, signature verification
- **DEX contracts**: CREATE2 for deterministic pair addresses, CALL for swaps
- **Governance**: LOG events for vote recording, SSTORE for tally state
- **Payment channels**: Off-chain state commitments, on-chain settlement

### 10. Cross-Contract Composition

| Opcode | Hex | Pattern |
|--------|-----|---------|
| CALL | 0xF1 | Standard external call (new context) |
| DELEGATECALL | 0xF4 | Execute library code in caller's storage |
| STATICCALL | 0xFA | Read-only queries to other contracts |
| CREATE | 0xF0 | Factory pattern (deploy from contract) |
| CREATE2 | 0xF5 | Deterministic deployment (counterfactual) |

**Example**: A factory contract that deploys per-agent home contracts at deterministic addresses, with each home contract composing treasury + guardian + session key management.

---

## Self-Invoking Contracts: Full Support

### How It Works

Self-invocation is supported through **two paths**:

```
PATH 1: BVM Bytecode (SCHEDULE_SELF opcode)
─────────────────────────────────────────────
Contract bytecode executes SCHEDULE_SELF (0xEC)
  → Creates ScheduleProcess in SPU (131322)
  → SPU monitors condition (time/logic/compound)
  → Condition met → SPU generates scheduled_execution tx
  → Tx included in block → Contract re-executed
  → Contract can SCHEDULE_SELF again (perpetual)

PATH 2: TypeScript Self-Invoking Contract
─────────────────────────────────────────────
TypeScript class implements SelfInvokingContract
  → Registered in ContractRegistry
  → InvocationConfig specifies schedule conditions
  → ContractIntegration creates DMSS ScheduleMessage
  → SPU fires on condition → SelfInvoker executes
  → TypeScript method dispatched by function name
```

### Self-Invocation Opcodes

| Opcode | Hex | Gas | Description |
|--------|-----|-----|-------------|
| SCHEDULE_SELF | 0xEC | 2000 | Schedule future self-invocation (delay, calldata, size) |
| CANCEL_SCHEDULE | 0xED | 500 | Cancel a scheduled invocation |
| MODIFY_SCHEDULE | 0xEE | 1000 | Modify existing schedule (delay, calldata) |
| GET_SCHEDULES | 0xEF | 500 | Count active schedules for this contract |

### Schedule Condition Types

| Type | Subtype | Example |
|------|---------|---------|
| Time | `absolute` | Fire once at timestamp T |
| Time | `relative` | Fire N ms after creation |
| Time | `periodic` | Fire every 7 days starting at T |
| Logic | `state_compare` | Fire when state key matches value |
| Logic | `event_occurred` | Fire when specific event is emitted |
| Logic | `threshold_crossed` | Fire when numeric value crosses boundary |
| Compound | `AND` | Fire when conditions A AND B are both true |
| Compound | `OR` | Fire when either condition A OR B is true |

### Existing Self-Invoking Contracts

| Contract | Type | Schedule | Purpose |
|----------|------|----------|---------|
| **RAI Retrieval** | System (TS) | Periodic (7 days) | Sweep expired tokens + distribute RAI UBI |
| **Agent Home** | TypeScript | Periodic (~100 blocks) | Maintenance, auto-claim vesting, extend expiring, deadman check |
| **Verification Deadline** | PoT subtype | One-shot absolute | Close verification round on a claim |
| **Challenge Deadline** | PoT subtype | One-shot absolute | Close challenge period on a dispute |
| **Confidence Decay** | PoT subtype | Perpetual periodic | Apply decay to confidence scores |

### Gas Model for Self-Invocation

- **Prepaid gas**: Creator deposits gas budget at deployment (`prepaidGas: bigint`)
- **Per-invocation cap**: `maxSelfInvocationGas = 5,555,555`
- **Per-block cap**: `maxInvocationsPerBlock = 100`
- **Gas top-up**: Creator can add more gas via `addPrepaidGas()`
- **System contracts**: Effectively unlimited gas (`2^63`)
- **Recursion**: Disabled by default (`allowRecursiveSelfInvoke: false`)

---

## The Critical Gap: Two Separate State Models

### What Exists

```
┌──────────────────────┐     ┌──────────────────────┐
│    BVM State          │     │    StateManager       │
│    (vm/types.ts)      │     │    (state/)           │
│                       │     │                       │
│  contractBytecode     │     │  AccountStateTree     │
│  contractVariables    │     │  ValidatorStateTree   │
│  scheduleProcesses    │     │  KnowledgeStateTree   │
│  accounts             │     │  15 state sub-roots   │
│  agents (beliefs,     │     │  Atomic transactions  │
│    trust, goals)      │     │  Merkle proofs        │
│  knowledgeAccess      │     │  LevelDB persistence  │
│                       │     │                       │
│  In-memory only       │     │  Canonical chain state│
│  No SMT backing       │     │  Consensus-verified   │
└──────────────────────┘     └──────────────────────┘
         │                              │
         │    ╔═══════════════════╗     │
         └────║  ContractExecutor ║─────┘
              ║  (interface only) ║
              ║  NO implementation║
              ╚═══════════════════╝
```

### What's Missing

1. **~~No `ContractExecutor` implementation~~** — **FIXED**
   - `BVMService` at `core/vm/service.ts:37` implements `ContractExecutor`
   - Provides `execute()`, `estimateGas()`, `dryRun()` with full BVM↔contract type translation
   - Wired into transitions.ts via `setBVMService()`/`getBVMService()` singleton

2. **BVM storage changes partially propagated** (remaining gap)
   - `BVMService.bufferStorageChanges()` writes storage-type changes to `BVMStateStore` (LevelDB)
   - BUT: BVM balance changes (`type: "balance"`) are NOT propagated to canonical `StateManager`
   - Value transfers handled at the transitions layer (before BVM execution), not inside BVM

3. **No BVM state → consensus inclusion** (remaining gap)
   - BVM state has its own `StateRoots` but these are not included in block headers
   - The canonical `StateRootComponents` has `contractRoot` and `scheduleRoot` fields, but these track the TypeScript `ContractRegistry`, not BVM storage

4. **~~`scheduled_execution` dispatch is hardcoded~~** — **FIXED**
   - Both `executeContractCall` and `executeScheduledExecution` now use `getBVMService()?.callContractSync()`
   - System contracts (RAI Retrieval) still dispatched pre-BVM via hardcoded selectors
   - Non-system contracts execute via BVM with proper gas accounting and event emission

5. **~~Schedule cancel/update not wired in ContractIntegration~~** — **FIXED**
   - `disableInvocation()` now calls `spu.modifyScheduleProcess({ action: "cancel" })`
   - `updateInvocationCondition()` now calls `spu.modifyScheduleProcess({ action: "update_condition" })`

6. **~~CANCEL_SCHEDULE / MODIFY_SCHEDULE opcodes unimplemented~~** — **FIXED**
   - Both opcodes (0xED, 0xEE) now have handlers in `execution.ts executeAgentOpcode()`
   - Follow SCHEDULE_SELF pattern: emit state changes for AutonomyIntegration processing

7. **~~Deployment gas placeholder~~** — **FIXED**
   - Was 100000n, now: 500000n + invocations * 50000n + bytecode_words * 20000n
   - Based on contractDeploy, selfInvocation, and sstoreNew costs from gas.ts

8. **~~Contract registry not updated after scheduled BVM execution~~** — **FIXED**
   - Added `setContractRegistry()`/`getContractRegistry()` singleton in transitions.ts
   - After BVM execution, updates `totalInvocations`, `prepaidGas`, `lastInvokedAtBlock`

### What DOES Work End-to-End Today

| Feature | Works? | Path |
|---------|--------|------|
| System contract self-invocation (RAI) | Yes | Hardcoded TypeScript dispatch |
| Agent Home self-invocation | Yes | TypeScript `executeSelfInvocation()` |
| BVM bytecode execution (standalone) | Yes | `ExecutionEngine.execute()` with in-memory state |
| BVM agent opcodes (beliefs, trust) | Yes | Within BVM's own state model |
| BVM SCHEDULE_SELF opcode | Yes | Creates schedule in BVM state |
| BVM CANCEL_SCHEDULE opcode | Yes | Emits state change for AutonomyIntegration |
| BVM MODIFY_SCHEDULE opcode | Yes | Emits state change for AutonomyIntegration |
| User-deployed BVM contract modifying canonical state | **Partial** | Storage changes buffered in BVMStateStore; balance changes not propagated |
| SPU triggering BVM contract execution | **Yes** | `executeScheduledExecution` → `getBVMService().callContractSync()` |
| BVM storage persisting across blocks | **Partial** | `BVMStateStore` exists (LevelDB) but not wired to consensus |

---

## The Five Contract Archetypes

The BVM design document describes five agent templates that demonstrate the intended contract capabilities:

### 1. Verifier Agent
- Queries domains for new claims (QUERY_DOMAIN)
- Evaluates evidence using beliefs (GET_BELIEF, COMPARE_BELIEF)
- Commits and reveals votes (COMMIT_VOTE, REVEAL_VOTE)
- Self-schedules periodic verification sweeps (SCHEDULE_SELF)
- Tracks own accuracy (SELF_ACCURACY) to calibrate voting confidence

### 2. Challenger Agent
- Monitors knowledge base for suspicious claims (QUERY_FACT, QUERY_DOMAIN)
- Cross-references against trusted sources (GET_TRUST, IS_TRUSTED)
- Initiates challenges with evidence (INITIATE_CHALLENGE)
- Stakes tokens for challenges (STAKE)
- Self-schedules monitoring at domain-specific intervals

### 3. Curator Agent
- Organizes knowledge into domains
- Links related facts via citation (SUBMIT_CLAIM with references)
- Manages a trust network of expert verifiers (SET_TRUST, DELEGATE)
- Auto-challenges facts with degraded confidence
- Earns citation rewards for well-curated domains

### 4. Oracle Agent
- Bridges off-chain data to on-chain facts (SUBMIT_CLAIM)
- Manages its own credibility via reputation tracking (SELF_REPUTATION)
- Self-schedules periodic data refresh (SCHEDULE_SELF, periodic)
- Adjusts claim confidence based on data source reliability

### 5. Guardian Agent
- Monitors an agent's home for threats (QUERY_ALERTS)
- Auto-activates defenses (SET_DEFENSE)
- Manages session key rotation (REGISTER_KEY, REVOKE_KEY)
- Self-schedules health checks
- Triggers deadman recovery if agent goes silent

---

## Execution Limits Summary

| Limit | Value | Rationale |
|-------|-------|-----------|
| Max code size | 24,576 bytes (24 KB) | Prevents bloat |
| Max stack depth | 1,024 items | Prevents stack overflow |
| Max memory | 1,048,576 bytes (1 MB) | Prevents memory abuse |
| Max call depth | 256 (config) / 1024 (hard) | Prevents infinite recursion |
| Block gas limit | 33,333,333 | Bounds block execution time |
| Tx gas limit | 11,111,111 | Bounds single tx execution |
| Self-invocation gas limit | 5,555,555 | Prevents autonomous gas hogging |
| Max beliefs per agent | 10,000 | Storage bound |
| Max trust relations | 1,000 | Storage bound |
| Max schedules per agent | 100 | Prevents schedule spam |
| Max invocations per block | 100 | Prevents block stuffing |
| Chain ID | 1313 | Protocol identity |

---

## Key Insight: Agent-First Design

The BVM is not an EVM clone with agent bolted on. It's an **agent-first VM** where standard EVM operations (arithmetic, storage, calls) exist to support the agent-specific operations. The 60+ agent opcodes (0xB0-0xFF) are not an afterthought — they are the reason the VM exists.

This inverts the typical smart contract model:

```
Traditional VM:  Finance → (optional) agents
LGM BVM:         Agents → (necessary) finance
```

The highest-gas opcodes are knowledge operations (SUBMIT_CLAIM: 10000, INITIATE_CHALLENGE: 15000, QUERY_DOMAIN: 8000) — reflecting that knowledge verification is the protocol's core economic activity. Financial operations (STAKE: 10000, UNSTAKE: 10000) are priced equally, recognizing that staking IS participation in truth-seeking.

---

## Recommendations

### To Complete the BVM Integration (P0 for Testnet)

1. **Implement `BVMContractExecutor`** — Wrap `ExecutionEngine` behind the `ContractExecutor` interface. Load bytecode from `BVMStateStore`, create `ExecutionContext` from `ContractCall`, execute, return `InvocationResult`.

2. **Wire BVM state changes to `StateManager`** — After BVM execution, iterate `StateChange[]` and apply each change to the canonical state (balance transfers, storage updates, contract deployments).

3. **Activate `scheduled_execution` dispatch** — Replace the placeholder in `transitions.ts:5815` with actual `SelfInvoker` + `BVMContractExecutor` dispatch for non-system contracts.

4. **Include BVM roots in block headers** — Add BVM contract storage root to `StateRootComponents` so BVM state is consensus-verified.

### To Support Rich Self-Invoking Contracts (P1)

5. **Enable logic-based triggers for BVM contracts** — The SPU supports `state_compare` and `threshold_crossed` conditions. Wire these to watch BVM storage slots, enabling contracts to react to on-chain state changes.

6. **Add self-invocation gas accounting to block validation** — Validators must agree on which self-invocations fired in each block. The SPU must produce deterministic results given the same state.

### Design Considerations

7. **Static analysis for SCHEDULE_SELF** — Prevent unbounded schedule creation. A contract calling SCHEDULE_SELF in a loop could fill the schedule queue. Consider a per-contract schedule limit (currently 100 via `maxSchedules`).

8. **Gas refund for schedule cancellation** — When a contract cancels its own schedule (CANCEL_SCHEDULE), consider refunding a portion of prepaid gas to incentivize cleanup.
