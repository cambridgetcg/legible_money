# Self-Invoking Contracts Module

Autonomous on-chain agents that invoke their own functions.

## Overview

Self-invoking contracts can autonomously execute based on:

- **Time-based conditions** (absolute, relative, periodic)
- **Logic-based conditions** (state comparison, events, thresholds)
- **Compound conditions** (AND/OR combinations)

This enables truly autonomous on-chain agents for the knowledge economy.

## Architecture

```
┌────────────────────────────────────────────────────────────┐
│                   Contract Deployment                       │
│                                                             │
│  Deploy Request ──▶ Registry ──▶ DMSS Registration         │
│                        │                 │                  │
│                        ▼                 ▼                  │
│                   Contract         Schedule Process         │
└────────────────────────────────────────────────────────────┘
                            │                 │
                            │                 │
                            ▼                 ▼
┌────────────────────────────────────────────────────────────┐
│                   Invocation Flow                           │
│                                                             │
│  SPU monitors ──▶ Condition Met ──▶ Invoker Queue          │
│                                           │                 │
│                                           ▼                 │
│                                     BVM Execution           │
│                                           │                 │
│                                           ▼                 │
│                                    State Update             │
└────────────────────────────────────────────────────────────┘
```

## Components

### ContractRegistry

Contract registration and discovery.

```typescript
import { ContractRegistry } from './contracts';

const registry = new ContractRegistry(store);

// Register contract
await registry.register({
  owner: 'did:lgm:owner',
  code: contractBytecode,
  abi: contractABI,
  invocationConfig: {
    conditions: [
      { type: 'time', schedule: 'periodic', intervalBlocks: 1000 },
    ],
    functions: ['processQueue', 'updateState'],
  },
});

// Find contracts by owner
const contracts = await registry.findByOwner('did:lgm:owner');

// Get contract metadata
const metadata = await registry.getMetadata(contractId);
```

### SelfInvoker

Self-invocation execution engine.

```typescript
import { SelfInvoker } from './contracts';

const invoker = new SelfInvoker(registry, executor, stateManager);

// Process pending invocations
const results = await invoker.processPending(currentBlock);

// Manual invocation (for testing)
const result = await invoker.invoke(contractId, 'processQueue', []);

console.log('Success:', result.success);
console.log('Gas used:', result.gasUsed);
console.log('State changes:', result.stateChanges);
```

### ContractIntegration

DMSS-Contract integration bridge.

```typescript
import { ContractIntegration } from './contracts';

const integration = new ContractIntegration(registry, dmss);

// Deploy with scheduling
await integration.deployWithSchedule({
  contract: myContract,
  schedules: [
    {
      function: 'dailyMaintenance',
      condition: { type: 'time', schedule: 'periodic', intervalBlocks: 34560 },
    },
    {
      function: 'onBalanceLow',
      condition: { type: 'logic', operator: 'lt', field: 'balance', value: 100n },
    },
  ],
});

// Sync contract state with DMSS
await integration.syncSchedules(contractId);
```

## PoT-Specific Contracts

### VerificationContract

Automatic verification deadline handling.

```typescript
const verificationContract: VerificationContract = {
  roundId: 'round:123',
  deadlines: {
    commitDeadline: currentBlock + 100,
    revealDeadline: currentBlock + 200,
  },
  methods: {
    onCommitDeadline: 'processCommits',
    onRevealDeadline: 'processReveals',
  },
};
```

### DisputeContract

Challenge window management.

```typescript
const disputeContract: DisputeContract = {
  factId: 'fact:456',
  challengeWindow: 10000,  // blocks
  methods: {
    onChallengeStart: 'notifyStakeholders',
    onChallengeEnd: 'resolveDispute',
  },
};
```

### VestingContract

Vesting schedule execution.

```typescript
const vestingContract: VestingContract = {
  beneficiary: 'did:lgm:agent',
  totalAmount: 1000n * 10n ** 18n,
  schedule: {
    cliffBlocks: 10000,
    vestingBlocks: 100000,
    intervalBlocks: 1000,
  },
  methods: {
    onVestingTick: 'releaseVestedTokens',
    onFullyVested: 'completeVesting',
  },
};
```

## Condition Types

### Time Conditions

```typescript
// Absolute time
{ type: 'time', schedule: 'absolute', blockNumber: 100000 }

// Relative time (from deployment)
{ type: 'time', schedule: 'relative', offsetBlocks: 1000 }

// Periodic
{ type: 'time', schedule: 'periodic', intervalBlocks: 1000, startBlock: 50000 }
```

### Logic Conditions

```typescript
// State comparison
{ type: 'logic', operator: 'gt', field: 'balance', value: 1000n }

// Event trigger
{ type: 'logic', operator: 'eq', field: 'lastEvent', value: 'challenge_submitted' }

// Threshold
{ type: 'logic', operator: 'gte', field: 'pendingRewards', value: 100n }
```

### Compound Conditions

```typescript
// AND condition
{
  type: 'compound',
  operator: 'and',
  conditions: [
    { type: 'time', schedule: 'periodic', intervalBlocks: 1000 },
    { type: 'logic', operator: 'gt', field: 'queueSize', value: 10 },
  ]
}

// OR condition
{
  type: 'compound',
  operator: 'or',
  conditions: [
    { type: 'logic', operator: 'lt', field: 'balance', value: 100n },
    { type: 'time', schedule: 'absolute', blockNumber: 200000 },
  ]
}
```

## Testing

```bash
# Run contract tests
npm run test -- tests/unit/contracts/
```

## Files

| File | Description |
|------|-------------|
| `types.ts` | Type definitions |
| `registry.ts` | Contract registry |
| `invoker.ts` | Self-invocation engine |
| `integration.ts` | DMSS integration |

## See Also

- [Schedule Module](../schedule/README.md) - DMSS system
- [GLOSSARY.md](../../docs/GLOSSARY.md) - Terminology

---

*Enabling truly autonomous on-chain agents*
