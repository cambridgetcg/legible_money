# State Layer

> Persistent state management using Sparse Merkle Trees.

## Quick Reference

| Item | Value |
|------|-------|
| **Layer** | state (Layer 2b) |
| **Dependencies** | `foundation/`, `core/crypto/`, `consensus/pot/types` |
| **Dependents** | `consensus/pot/engine`, `protocol/`, `network/` |

## What This Module Does

The state layer:
1. Stores all on-chain state in Sparse Merkle Trees (SMTs)
2. Provides O(log n) proofs for inclusion/exclusion
3. Executes transactions atomically
4. Maintains multiple state trees under a unified root

```
                StateRoot
                    │
       ┌────────────┼────────────┐
       │            │            │
  AccountRoot  ValidatorRoot  KnowledgeRoot
       │            │            │
     [SMT]        [SMT]        [SMT]
```

## Key Types

```typescript
// Combined state root
StateRoot = string;  // Hex-encoded hash of all sub-roots

// Individual account state
AccountState {
  did: string,
  balance: bigint,
  stakedBalance: bigint,
  nonce: bigint,
  reputationScore: number,
  ...
}

// Validator state
ValidatorState {
  did: string,
  stake: bigint,
  status: 'active' | 'unbonding' | 'slashed',
  ...
}

// Knowledge state (facts)
KnowledgeState {
  factId: string,
  confidence: Confidence,
  submitter: DID,
  ...
}
```

## Key Files

| File | Purpose |
|------|---------|
| `types.ts` | All state type definitions |
| `state_manager.ts` | Unified StateManager class |
| `accounts.ts` | Account SMT operations |
| `validators.ts` | Validator SMT operations |
| `knowledge.ts` | Knowledge SMT operations |
| `transitions.ts` | Transaction execution logic |

## Common Tasks

### Read account balance

```typescript
import { StateManager } from './state_manager.js';

const state = new StateManager();
const account = await state.getAccount(did);
console.log('Balance:', account?.balance);
```

### Execute a transaction

```typescript
// Always use atomic transactions
state.beginTransaction();
try {
  await applyTransaction(state, tx, block);
  state.commitTransaction();
} catch (error) {
  state.rollbackTransaction();
  throw error;
}
```

### Add a new transaction type

1. Add type to `consensus/pot/types.ts`:
```typescript
export interface NewTransaction {
  type: 'new_type';
  // fields...
}
```

2. Add execution in `transitions.ts`:
```typescript
case 'new_type':
  return applyNewTransaction(state, tx as NewTransaction, context);
```

3. Implement `applyNewTransaction()` with state mutations

### Generate a Merkle proof

```typescript
const proof = await state.proveAccount(did);
// proof can be verified by light clients
```

## Where Am I?

```
legible-money/
└── state/             ← YOU ARE HERE
    ├── README.md
    ├── types.ts           # All state types
    ├── state_manager.ts   # Unified state management
    ├── accounts.ts        # Account SMT
    ├── validators.ts      # Validator SMT
    ├── knowledge.ts       # Knowledge SMT
    ├── transitions.ts     # Transaction execution
    └── snapshots.ts       # State snapshots
```

## Related

- **Tests**: `tests/unit/state/`
- **Consensus**: `consensus/pot/engine.ts` (uses state)
- **Storage**: `storage/` (persistence layer)
- **Patterns**: `.claude/context/API_PATTERNS.md`

## Architecture Notes

### Why Sparse Merkle Trees?

- **O(log n) proofs**: Efficient verification
- **Efficient for sparse data**: Most accounts empty
- **Light client support**: Compact inclusion proofs
- **256-bit keys**: Hash(did) as key

### Atomic Transactions

**Always use the pattern:**
```typescript
state.beginTransaction();
try {
  // All mutations here
  state.commitTransaction();
} catch {
  state.rollbackTransaction();
}
```

This ensures:
- All-or-nothing execution
- No partial state updates
- Consistent state root

### State Root Composition

```typescript
StateRoot = hash(
  accountRoot ||
  validatorRoot ||
  knowledgeRoot ||
  ...otherRoots
)
```

Each sub-root is updated independently, then combined.

## BigInt Rule

**Always use `bigint` for monetary values:**
```typescript
// CORRECT
balance: bigint = 1000n;

// WRONG - will overflow
balance: number = 1000;
```

Serialize as string for JSON:
```typescript
JSON.stringify({ balance: balance.toString() })
BigInt(obj.balance)  // Parse back
```
