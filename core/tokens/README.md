# Expirable Tokens Module

Token expiration system using the Dynamic Array method.

## Overview

Implements token expiration to encourage active participation in the knowledge economy. Key features:

- **Tokens with limited lifespan** (~6 months default)
- **Efficient O(1) balance tracking** via child accounts
- **Automatic retrieval** of expired tokens
- **RAI (Regenerative Agent Income)** distribution

## Circular Economy Model

```
┌────────────────────────────────────────────────────────┐
│                                                         │
│  ┌─────────┐     ┌──────────┐     ┌──────────────┐     │
│  │  Mint   │────▶│  Active  │────▶│   Expired    │     │
│  │ Tokens  │     │  Balance │     │   Balance    │     │
│  └─────────┘     └──────────┘     └──────────────┘     │
│       ▲                                  │             │
│       │                                  ▼             │
│  ┌─────────┐                       ┌──────────┐       │
│  │   RAI   │◀──────────────────────│ Retrieval│       │
│  │  Dist.  │                       │  Engine  │       │
│  └─────────┘                       └──────────┘       │
│                                                         │
└────────────────────────────────────────────────────────┘
```

## Components

### ChildAccountManager

Per-period balance tracking for O(1) operations.

```typescript
import { ChildAccountManager } from './tokens';

const manager = new ChildAccountManager(store);

// Get or create child account for period
const child = await manager.getOrCreate(agentDid, periodId);

// Add balance to child account
await manager.credit(agentDid, periodId, amount);

// Get total active balance (across all non-expired periods)
const balance = await manager.getActiveBalance(agentDid);
```

### ExpirationTracker

Dynamic Array implementation for efficient expiration tracking.

```typescript
import { ExpirationTracker } from './tokens';

const tracker = new ExpirationTracker(store, {
  periodLength: 100000,        // Blocks per period
  expirationPeriods: 6,        // Periods until expiration
  retrievalGracePeriod: 10000, // Grace period blocks
});

// Record token mint
await tracker.recordMint(agentDid, amount, currentBlock);

// Get expiring balances
const expiring = await tracker.getExpiringBalances(currentBlock);

// Process period transition
await tracker.processPeriodTransition(currentBlock);
```

### RetrievalEngine

Expired token retrieval and RAI distribution.

```typescript
import { RetrievalEngine } from './tokens';

const engine = new RetrievalEngine(store, stateManager);

// Trigger retrieval for expired tokens
const result = await engine.retrieveExpired({
  accounts: ['did:lgm:agent1', 'did:lgm:agent2'],
  upToBlock: currentBlock,
});

console.log('Retrieved:', result.totalRetrieved);
console.log('Accounts processed:', result.accountsProcessed);

// Distribute RAI
await engine.distributeRAI({
  totalPool: retrievedAmount,
  eligibleAccounts: activeAccounts,
  distributionMethod: 'proportional',
});
```

## Configuration

```typescript
interface ExpirationConfig {
  periodLength: number;        // Blocks per period (default: 100000 ~3 days)
  expirationPeriods: number;   // Periods until expiration (default: 6)
  retrievalGracePeriod: number;// Grace period blocks (default: 10000)
  minBalanceForRAI: bigint;    // Minimum balance for RAI eligibility
  raiDistributionRatio: number;// Ratio distributed as RAI (default: 0.8)
}
```

## How It Works

### Period-Based Tracking

1. Each period (e.g., ~3 days) has its own "child account"
2. Tokens minted during a period go into that period's child
3. When period expires, child account tokens become retrievable
4. Total balance = sum of active (non-expired) child accounts

### Dynamic Array Method

```
Period:      1    2    3    4    5    6    7    (current)
             │    │    │    │    │    │    │
Child:      [C1] [C2] [C3] [C4] [C5] [C6] [C7]
             │         │
          Expired    Active────────────────▶

- C1, C2 expired (period 1, 2)
- C3-C7 active (periods 3-7)
- When period 8 starts, C3 expires
```

### RAI Distribution

Retrieved tokens are distributed as Regenerative Agent Income:

1. **80%** → Active agents (proportional to activity)
2. **20%** → Protocol treasury

## Transfer Policies

```typescript
interface TransferPolicy {
  allowPartialExpired: boolean;  // Allow sending near-expiry tokens
  preferNewerTokens: boolean;    // Send newer tokens first
  minExpirationBlocks: number;   // Minimum blocks until expiry for transfer
}
```

## Testing

```bash
# Run token tests
npm run test -- tests/unit/tokens/

# Specific tests
npm run test -- tests/unit/tokens/child_account.test.ts
npm run test -- tests/unit/tokens/expiration_tracker.test.ts
npm run test -- tests/unit/tokens/retrieval_engine.test.ts
```

## Files

| File | Description |
|------|-------------|
| `types.ts` | Type definitions |
| `child_account.ts` | Child account manager |
| `expiration_tracker.ts` | Dynamic array tracker |
| `retrieval_engine.ts` | Retrieval and RAI |

## See Also

- [Knowledge Economics](../../docs/KNOWLEDGE_ECONOMICS.md)
- [GLOSSARY.md](../../docs/GLOSSARY.md)

---

*Encouraging active participation through token expiration*
