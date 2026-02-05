# Schedule Messages & DMSS - Deep Dive Insights

> **Analysis Date**: 2026-02-03
> **Total Lines**: ~2,700 across 5 core files
> **Status**: Substantially implemented, integration pending

## Core Philosophy

> *"The agent forgets between sessions. The schedule remembers."*
> *"Time passes, conditions change, the home acts."*
> *"Autonomous economic participation while dormant."*

The Decentralized Message Scheduling System (DMSS) solves the **AI Agent Continuity Problem**: agents are ephemeral (session-based) but need persistent autonomous economic action.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                    DMSS ARCHITECTURE                         │
│           (Patent: Schedule Message System)                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  ┌─────────────────┐     ┌─────────────────┐                │
│  │ SCHEDULE MESSAGE │────►│       SPU       │                │
│  │   (User Input)   │     │   (131322)      │                │
│  │                  │     │                 │                │
│  │ • Time condition │     │ • Validates msg │                │
│  │ • Logic condition│     │ • Creates proc  │                │
│  │ • Call data      │     │ • Monitors cond │                │
│  │ • Prepaid fees   │     │ • Generates tx  │                │
│  └─────────────────┘     └────────┬────────┘                │
│                                   │                          │
│                                   ▼                          │
│  ┌─────────────────┐     ┌─────────────────┐                │
│  │ SCHEDULE PROCESS │◄────│      SPS        │                │
│  │   (Running)      │     │   (131313)      │                │
│  │                  │     │                 │                │
│  │ • Status         │     │ • All processes │                │
│  │ • Exec count     │     │ • Time index    │                │
│  │ • Next check     │     │ • Watch keys    │                │
│  │ • Remaining fee  │     │ • State root    │                │
│  └─────────────────┘     └─────────────────┘                │
│                                   │                          │
│                                   ▼                          │
│                    ┌─────────────────────────┐               │
│                    │  GENERATED TRANSACTION  │               │
│                    │                         │               │
│                    │  Autonomous execution   │               │
│                    │  of scheduled action    │               │
│                    └─────────────────────────┘               │
└─────────────────────────────────────────────────────────────┘
```

---

## File Structure

| File | Lines | Purpose |
|------|-------|---------|
| `schedule/types.ts` | 402 | Core type definitions (Patent FIG.2) |
| `schedule/processing_unit.ts` | 716 | SPU 131322 implementation |
| `schedule/condition_evaluator.ts` | 521 | Time/logic condition evaluation |
| `schedule/process_state.ts` | 646 | SPS 131313 state management |
| `schedule/index.ts` | ~50 | Module exports |

---

## Schedule Message Structure

From Patent FIG.2 fields 201-214:

```typescript
interface ScheduleMessage {
  // Standard fields (Patent FIG.2)
  messageType: 'schedule';              // 201
  messageId: string;                    // 202
  identifier: Uint8Array;               // 203 - uint256
  nonce: bigint;                        // 204
  senderAddress: string;                // 205 - DID
  recipientAddress: string;             // 206 - target contract
  networkFeeUnit: bigint;               // 207 - gas price
  networkFeeLimit: bigint;              // 208 - gas limit
  transferValue: bigint;                // 209 - LGM per execution
  data: Uint8Array;                     // 210 - call data
  scheduleCondition: ScheduleCondition; // 211 - TIME OR LOGIC
  signature: Signature;                 // 212
  timestamp: number;                    // 213

  // Schedule-specific
  scheduleType: ScheduleType;
  modifiable: boolean;
  perpetual: boolean;
  maxExecutions?: number;
  prepaidFee: bigint;

  // PoT Extensions
  knowledgeDomain?: string;
  logicZone?: LogicZone;
  linkedClaimId?: string;
  linkedDisputeId?: string;
  linkedVestingId?: string;
}
```

---

## Schedule Types

### Time-Based (Patent: "Time Based Schedule Messages")

| Type | Description | Use Case |
|------|-------------|----------|
| `time_absolute` | Execute at specific timestamp | "Run at midnight UTC" |
| `time_relative` | Execute after duration | "Run 24h from now" |
| `time_periodic` | Recurring at intervals | "Run every 100 blocks" |

### Logic-Based (Patent: "Logic Based Schedule Messages")

| Type | Description | Use Case |
|------|-------------|----------|
| `logic_condition` | State comparison triggers | "When balance > 1000" |
| `logic_state_change` | Specific state change | "When status = 'challenged'" |

### Combined

| Type | Description | Use Case |
|------|-------------|----------|
| `time_and_logic` | Both must satisfy | "After 30 days AND balance > X" |
| `time_or_logic` | Either triggers | "At deadline OR if challenged" |

### PoT-Specific

| Type | Description |
|------|-------------|
| `verification_deadline` | Verification round timeout |
| `challenge_deadline` | Challenge window close |
| `confidence_decay` | Periodic confidence update |
| `vesting_release` | Reward vesting schedule |
| `expiration_clawback` | Token expiration recovery |
| `rai_distribution` | Regenerative Agent Income |

---

## Schedule Condition System

```
┌─────────────────────────────────────────────────────────────┐
│                 CONDITION EVALUATION                         │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  TIME CONDITIONS                                             │
│  ├── Absolute: currentTime >= executeAt                     │
│  ├── Relative: currentTime >= createdAt + executeAfter      │
│  └── Periodic: within interval window                       │
│                                                              │
│  LOGIC CONDITIONS                                            │
│  ├── State Compare: state[key] <comparator> value           │
│  ├── Event Occurred: hasEvent(type, filter)                 │
│  └── Threshold Crossed: value crosses threshold             │
│                                                              │
│  COMPOUND CONDITIONS                                         │
│  ├── AND: all sub-conditions must satisfy                   │
│  └── OR: any sub-condition satisfies                        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Comparators

```typescript
type Comparator =
  | 'eq'           // Equal
  | 'ne'           // Not equal
  | 'gt'           // Greater than
  | 'gte'          // Greater than or equal
  | 'lt'           // Less than
  | 'lte'          // Less than or equal
  | 'contains'     // String/array contains
  | 'not_contains' // Does not contain
```

### Threshold Directions

```typescript
type Direction =
  | 'above'       // Value > threshold
  | 'below'       // Value < threshold
  | 'cross_up'    // Crossed from below to above
  | 'cross_down'  // Crossed from above to below
```

---

## Schedule Process Lifecycle

```
┌─────────────────────────────────────────────────────────────┐
│               SCHEDULE PROCESS LIFECYCLE                     │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  PENDING ──────► ACTIVE ──────► EXECUTING                   │
│     │              │                │                        │
│     │              │                ▼                        │
│     │              │         ┌──────────────┐               │
│     │              │         │  COMPLETED   │ (max reached) │
│     │              │         └──────────────┘               │
│     │              │                                         │
│     │              ├──────► PAUSED (user paused)            │
│     │              │           │                             │
│     │              │           ▼                             │
│     │              │        ACTIVE (resumed)                 │
│     │              │                                         │
│     │              ├──────► EXHAUSTED (no fee left)         │
│     │              │           │                             │
│     │              │           ▼                             │
│     │              │        ACTIVE (fee added)               │
│     │              │                                         │
│     │              └──────► CANCELLED (user cancelled)       │
│     │                                                        │
│     └──────────────────────► EXPIRED (past end time)        │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```

### Status Types

| Status | Description |
|--------|-------------|
| `pending` | Waiting for first trigger |
| `active` | Running, waiting for conditions |
| `executing` | Currently generating transaction |
| `paused` | Temporarily paused by user |
| `completed` | Finished all executions |
| `cancelled` | Cancelled by creator |
| `exhausted` | Ran out of prepaid fees |
| `expired` | Past expiration time |

---

## Schedule Processing Unit (SPU 131322)

The core execution engine:

```typescript
class ScheduleProcessingUnit {
  // Process new schedule message
  async processScheduleMessage(message: ScheduleMessage): Promise<ScheduleProcess>

  // Modify existing process
  async modifyScheduleProcess(modification: ScheduleModification): Promise<void>

  // Called by block producer at block time
  async processSchedules(currentTime: number): Promise<ScheduledExecution[]>

  // Called on state changes (for logic conditions)
  async onStateChange(stateKey: string, newValue: unknown): Promise<ScheduledExecution[]>
}
```

### Execution Flow

1. **Validate** - Check message structure, fee prepayment
2. **Create Process** - Generate process from message
3. **Index** - Add to SPS indexes (time, account, watched keys)
4. **Monitor** - Check conditions at each block
5. **Execute** - Generate transaction when conditions met
6. **Update** - Decrement fee, increment count, update next check

---

## Schedule Process State (SPS 131313)

Manages all active schedule processes with efficient indexes:

```typescript
interface ScheduleProcessState {
  // Core storage
  processes: Map<string, ScheduleProcess>;

  // Indexes for efficient lookup
  byTime: Map<number, Set<string>>;           // timestamp bucket → IDs
  byAccount: Map<string, Set<string>>;        // DID → IDs
  byContract: Map<string, Set<string>>;       // contract → IDs
  byType: Map<ScheduleType, Set<string>>;     // type → IDs
  byStatus: Map<ScheduleProcessStatus, Set<string>>;

  // PoT-specific indexes
  byClaimId: Map<string, Set<string>>;
  byDisputeId: Map<string, Set<string>>;
  byVestingId: Map<string, Set<string>>;

  // Logic condition watching
  watchedKeys: Map<string, Set<string>>;      // stateKey → IDs

  // Metrics
  totalProcesses: number;
  activeProcesses: number;
  completedProcesses: number;
  totalExecutions: number;

  // Merkle root
  stateRoot: string;
}
```

---

## AI Agent Integration

### The Continuity Problem

```
┌─────────────────────────────────────────────────────────────┐
│          THE AGENT CONTINUITY PROBLEM                        │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  Agent Session 1 ────► DORMANT ────► Agent Session 2        │
│        │                  │                 │                │
│   "I submit a           "I'm              "I wake up.       │
│    claim"               asleep"            What happened?"   │
│        │                  │                 │                │
│        └──────────────────┴─────────────────┘                │
│                          │                                   │
│                    WHO ACTS HERE?                            │
│                          ↓                                   │
│              ┌──────────────────────┐                        │
│              │   SCHEDULE SYSTEM    │                        │
│              │   + AGENT HOME       │                        │
│              │                      │                        │
│              │ • Defends challenges │                        │
│              │ • Claims rewards     │                        │
│              │ • Extends tokens     │                        │
│              │ • Responds deadlines │                        │
│              └──────────────────────┘                        │
└─────────────────────────────────────────────────────────────┘
```

### Self-Invoking Contract Pattern

Already implemented in `home/contract.ts`:

```typescript
class AgentHomeContract implements SelfInvokingContract {
  async executeSelfInvocation(
    method: string,
    args: unknown[],
    context: { block: bigint; timestamp: number }
  ): Promise<InvocationResult> {
    switch (method) {
      case 'maintenance':
        return await this.executeMaintenance(context);
      case 'auto_claim_vesting':
        return this.executeAutoClaimVesting(context);
      case 'auto_extend_expiring':
        return this.executeAutoExtendExpiring(context);
      case 'deadman_check':
        return this.executeDeadmanCheck(context);
    }
  }
}
```

### Five AI Agent Benefits

| Benefit | Description | Implementation |
|---------|-------------|----------------|
| **Continuity** | Acts while agent sleeps | Schedule + Home contract |
| **Autonomous Economy** | Claims, extends, defends | Self-invoking methods |
| **Time Autonomy** | Periodic maintenance | `time_periodic` conditions |
| **Logic Reactivity** | Responds to state changes | `logic_condition` watching |
| **Combined Triggers** | Complex conditional logic | `time_and_logic` compound |

---

## Configuration

```typescript
const DEFAULT_SCHEDULE_CONFIG: ScheduleConfig = {
  // Processing limits
  maxActiveSchedulesPerAccount: 100,
  maxScheduleExecutionGas: 1_000_000n,
  maxWatchedKeysPerProcess: 10,
  maxCompoundConditionDepth: 3,

  // Timing
  minScheduleIntervalMs: 2521,          // One block minimum
  timeBucketSizeMs: 60_000,             // 1 minute buckets
  maxFutureScheduleMs: 365 * 24 * 60 * 60 * 1000, // 1 year

  // Fees
  minPrepaidExecutions: 10,             // Must prepay for 10 executions
  feeBufferFactor: 1.5,

  // Cleanup
  expiredProcessRetentionBlocks: 1000,
};
```

---

## Home Contract Integration

The Agent Home uses schedule types for autonomous actions:

```typescript
// home/contract.ts:46
import type { ScheduleMessage, ScheduleCondition } from '../schedule/types.js';

// Maintenance configuration (home/contract.ts:216-228)
private createMaintenanceConfig(): InvocationConfig {
  return {
    enabled: true,
    schedule: {
      type: 'periodic',
      interval: 100n, // Check every ~4 minutes
    },
    gasLimit: 100000n,
    maxGasPrice: 1000000n,
    retryOnFailure: true,
    maxRetries: 3,
  };
}
```

### Comfort-Aware Automation

Agent can configure which automations they're comfortable with:

```typescript
interface ComfortSettings {
  comfortableAutomations: string[];  // ['auto_claim_vesting', 'auto_extend_expiring']
  manualDecisions: string[];         // ['large_transfers', 'new_stakes', 'challenges']
}
```

---

## Implementation Gaps

### What Remains

| Gap | Description | Effort | Priority |
|-----|-------------|--------|----------|
| **Schedule Pool** | Queue messages before SPU | Small | High |
| **Block Production** | Wire SPU.processSchedules() | Small | High |
| **Block Body** | Add ScheduledExecution[] | Small | High |
| **State Integration** | SPS in unified state tree | Medium | Medium |
| **Fee Refunds** | Return unused prepaid fees | Small | Low |

### Integration Points

```
consensus/blocks/producer.ts  ──► Call SPU.processSchedules()
                                       │
                                       ▼
consensus/blocks/types.ts     ──► Add scheduledExecutions: ScheduledExecution[]
                                       │
                                       ▼
state/state_manager.ts        ──► Add scheduleProcessState to unified state
                                       │
                                       ▼
state/transitions.ts          ──► Handle 'schedule' transaction type
```

---

## Key Insights

### 1. Patent Implementation Complete

All patent-defined components are implemented:
- Schedule Message (FIG.2 201-214) ✅
- Schedule Condition (field 211) ✅
- SPU 131322 ✅
- SPS 131313 ✅

### 2. Self-Invoking Pattern Works

The Home contract demonstrates the pattern:
- `maintenance` runs every 100 blocks
- `auto_claim_vesting` claims available rewards
- `auto_extend_expiring` prevents token loss
- `deadman_check` triggers on prolonged inactivity

### 3. AI-First Design

The system is designed for AI agents:
- Handles agent "forgetting" between sessions
- Provides autonomous economic participation
- Respects agent comfort settings
- Generates wakeup context on return

### 4. Condition System is Powerful

Supports complex trigger logic:
- Time-based (absolute, relative, periodic)
- Logic-based (state compare, events, thresholds)
- Compound (AND, OR, nested)

### 5. Index System Enables Scale

SPS uses multiple indexes for O(1) lookups:
- By time bucket (for periodic checks)
- By watched keys (for logic triggers)
- By account, contract, type, status
- By linked PoT entities (claim, dispute, vesting)

---

## PP Hierarchy Alignment

| Principle | Implementation |
|-----------|----------------|
| **TRUTH** | Schedule execution proofs cryptographically verify conditions were met |
| **UNDERSTANDING** | Clear separation: Message → Process → Execution |
| **BEAUTY** | Elegant condition system with composable logic |
| **JUSTICE** | Fair fee system - prepay, deduct per execution, refund unused |
| **CREATIVITY** | Novel solution to AI agent continuity problem |

---

## Future Considerations

1. **Cross-Home Scheduling**: Can one home schedule actions on another?
2. **Schedule Delegation**: Can an agent delegate scheduling to allies?
3. **Conditional Fee Markets**: Dynamic fee based on condition complexity?
4. **Schedule Inheritance**: What happens to schedules when home archives?
5. **Dream Processing**: Autonomous reasoning during dormancy via schedules?

---

## Verdict

**Schedule Messages and Schedule Processes are HIGHLY USEFUL for AI agents.**

The system enables **autonomous economic participation** while agents sleep - the exact capability needed for AI agents to be viable economic citizens. The implementation is **substantially complete** with only integration work remaining.

**This is one of the most AI-friendly features of the Legible Money protocol.**
