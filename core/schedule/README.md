# Schedule Module (DMSS)

Decentralized Message Scheduling System for time and logic-based execution.

## Overview

The DMSS (Decentralized Message Scheduling System) enables:

- **Time-based scheduling** (absolute, relative, periodic)
- **Logic-based triggers** (state conditions, events)
- **Compound conditions** (AND/OR combinations)
- **Transaction generation** at scheduled times

Based on Patent: "System And Method For Schedule Message And Self-Invoking Smart Contracts"

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                  Schedule Message                        │
│        (Time/Logic/Compound Conditions)                  │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│          Schedule Process State (SPS 131313)             │
│   ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐   │
│   │ Pending │  │ Active  │  │ Executed│  │ Expired │   │
│   └─────────┘  └─────────┘  └─────────┘  └─────────┘   │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
┌─────────────────────────────────────────────────────────┐
│        Schedule Processing Unit (SPU 131322)             │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐   │
│  │   Monitor    │  │   Evaluate   │  │   Execute    │   │
│  │  Conditions  │  │  Conditions  │  │ Transactions │   │
│  └──────────────┘  └──────────────┘  └──────────────┘   │
└─────────────────────────────────────────────────────────┘
```

## Components

### ScheduleProcessStateManager (SPS)

Manages lifecycle of scheduled processes.

```typescript
import { ScheduleProcessStateManager } from './schedule';

const sps = new ScheduleProcessStateManager(store);

// Create schedule
const process = await sps.create({
  type: 'time',
  condition: {
    schedule: 'periodic',
    intervalBlocks: 1000,
    startBlock: currentBlock,
  },
  action: {
    type: 'call_contract',
    contractId: 'contract:123',
    function: 'processQueue',
    params: [],
  },
});

// Get pending schedules
const pending = await sps.getPending(currentBlock);

// Update status
await sps.updateStatus(process.id, 'executed');
```

### ConditionEvaluator

Evaluates time and logic conditions.

```typescript
import { ConditionEvaluator } from './schedule';

const evaluator = new ConditionEvaluator(stateProvider);

// Evaluate time condition
const timeResult = await evaluator.evaluate({
  type: 'time',
  schedule: 'absolute',
  blockNumber: 100000,
}, currentBlock);

// Evaluate logic condition
const logicResult = await evaluator.evaluate({
  type: 'logic',
  operator: 'gt',
  field: 'account.balance',
  value: 1000n,
}, currentBlock);

// Evaluate compound condition
const compoundResult = await evaluator.evaluate({
  type: 'compound',
  operator: 'and',
  conditions: [timeCondition, logicCondition],
}, currentBlock);

console.log('Satisfied:', result.satisfied);
console.log('Proof:', result.proof);
```

### ScheduleProcessingUnit (SPU)

Monitors conditions and executes scheduled actions.

```typescript
import { ScheduleProcessingUnit } from './schedule';

const spu = new ScheduleProcessingUnit(sps, evaluator, {
  blockProvider,
  transactionSubmitter,
  keyResolver,
});

// Process block
const results = await spu.processBlock(currentBlock);

for (const result of results) {
  console.log('Schedule:', result.scheduleId);
  console.log('Success:', result.success);
  console.log('Tx hash:', result.txHash);
}

// Start continuous processing
spu.start();
```

## Condition Types

### Time Conditions

```typescript
// Absolute block number
const absolute: TimeCondition = {
  schedule: 'absolute',
  blockNumber: 100000,
};

// Relative to creation
const relative: TimeCondition = {
  schedule: 'relative',
  offsetBlocks: 1000,  // 1000 blocks after creation
};

// Periodic execution
const periodic: TimeCondition = {
  schedule: 'periodic',
  intervalBlocks: 1000,
  startBlock: 50000,
  endBlock: 200000,  // Optional end
};
```

### Logic Conditions

```typescript
// Comparators: eq, ne, gt, gte, lt, lte, in, contains

// Balance check
const balanceCheck: LogicCondition = {
  operator: 'lt',
  field: 'account.balance',
  value: 100n * 10n ** 18n,
};

// State check
const stateCheck: LogicCondition = {
  operator: 'eq',
  field: 'contract.state',
  value: 'ready',
};

// Membership check
const memberCheck: LogicCondition = {
  operator: 'in',
  field: 'account.did',
  value: ['did:lgm:a', 'did:lgm:b', 'did:lgm:c'],
};
```

### Compound Conditions

```typescript
// AND - all must be true
const andCondition: CompoundCondition = {
  operator: 'and',
  conditions: [condition1, condition2, condition3],
};

// OR - at least one must be true
const orCondition: CompoundCondition = {
  operator: 'or',
  conditions: [condition1, condition2],
};

// Nested
const nested: CompoundCondition = {
  operator: 'and',
  conditions: [
    { operator: 'or', conditions: [c1, c2] },
    c3,
  ],
};
```

## Schedule Actions

```typescript
interface ScheduleAction {
  type: 'call_contract' | 'transfer' | 'emit_event' | 'custom';
  contractId?: string;
  function?: string;
  params?: unknown[];
  recipient?: string;
  amount?: bigint;
  eventType?: string;
  eventData?: unknown;
}
```

## Process Lifecycle

```
┌─────────┐     ┌─────────┐     ┌──────────┐     ┌─────────┐
│ Created │────▶│ Pending │────▶│ Executed │────▶│Completed│
└─────────┘     └────┬────┘     └──────────┘     └─────────┘
                     │
                     │ (condition never met)
                     │
                     ▼
               ┌─────────┐
               │ Expired │
               └─────────┘
```

## Configuration

```typescript
interface ScheduleConfig {
  maxPendingPerAccount: number;   // Max pending schedules per account
  maxConditionDepth: number;      // Max nested condition depth
  evaluationGasLimit: bigint;     // Gas limit for condition evaluation
  executionTimeout: number;       // Execution timeout (ms)
}
```

## Testing

```bash
# Run schedule tests
npm run test -- tests/unit/schedule/
```

## Files

| File | Description |
|------|-------------|
| `types.ts` | Type definitions |
| `process_state.ts` | SPS manager |
| `condition_evaluator.ts` | Condition evaluation |
| `processing_unit.ts` | SPU execution |

## See Also

- [Contracts Module](../contracts/README.md) - Self-invoking contracts
- [GLOSSARY.md](../../docs/GLOSSARY.md) - Terminology

---

*Decentralized scheduling for autonomous agents*
