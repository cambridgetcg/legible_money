# Invariants: Modular Architecture

> **Purpose**: Define architectural invariants BEFORE restructuring.
> These guide the modularization and prevent coupling violations.

## Overview

Legible Money's architecture must support:
- **Autonomous agents** operating at computational speed
- **Truth-seeking** through Proof of Truth consensus
- **Gödelian honesty** in knowledge representation
- **Clean module boundaries** for independent development

## Architectural Invariants

### Layer Invariants

#### ARCH-1: Layer Isolation

**Statement**:
```
Layer dependencies MUST flow downward only.

  Application Layer (protocol, sdk, cli)
         ↓ (depends on)
  Domain Layer (consensus, knowledge, rewards)
         ↓ (depends on)
  Infrastructure Layer (state, network, storage)
         ↓ (depends on)
  Foundation Layer (crypto, types, utils)
```

**Rationale**: Prevents circular dependencies, enables independent testing.

**Violation Example**: State module importing from consensus engine.

---

#### ARCH-2: Module Boundary Contracts

**Statement**:
```
Each module MUST expose only through its index.ts.
Internal files MUST NOT be imported directly.

✓ import { Fact } from '@legible/knowledge'
✗ import { Fact } from '@legible/knowledge/internal/types'
```

**Rationale**: Enables internal refactoring without breaking consumers.

---

#### ARCH-3: Dependency Injection for Infrastructure

**Statement**:
```
Domain modules MUST NOT directly instantiate infrastructure.
Infrastructure MUST be injected via interfaces.

interface StateProvider {
  getAccount(did: DID): Promise<AccountState>;
  updateAccount(did: DID, update: AccountUpdate): Promise<void>;
}

// Domain code receives StateProvider, never creates LevelDB directly
```

**Rationale**: Enables testing with mocks, supports multiple backends.

---

### Module Invariants

#### MOD-1: Single Responsibility

**Statement**:
```
Each module MUST have exactly ONE reason to change.

crypto/     → Changes when cryptographic algorithms change
knowledge/  → Changes when fact/claim semantics change
consensus/  → Changes when verification protocol changes
state/      → Changes when storage structure changes
```

**Rationale**: Reduces blast radius of changes.

---

#### MOD-2: Interface Segregation

**Statement**:
```
Modules MUST NOT depend on interfaces they don't use.

// Bad: Validator needs entire StateManager
class Validator {
  constructor(private state: StateManager) {}
}

// Good: Validator needs only what it uses
class Validator {
  constructor(
    private accounts: AccountReader,
    private validators: ValidatorReader
  ) {}
}
```

**Rationale**: Reduces coupling, clarifies actual dependencies.

---

#### MOD-3: No Shared Mutable State

**Statement**:
```
Modules MUST NOT share mutable state directly.
State changes MUST flow through defined interfaces.

// Bad: Direct mutation
sharedState.balance -= amount;

// Good: Through interface
await stateProvider.debit(account, amount);
```

**Rationale**: Prevents race conditions, enables concurrency.

---

### Data Flow Invariants

#### FLOW-1: Unidirectional Data Flow

**Statement**:
```
Data MUST flow in one direction through the system:

Transaction → Mempool → Consensus → State → Storage
                              ↓
                          Events → Subscribers
```

**Rationale**: Simplifies reasoning about state changes.

---

#### FLOW-2: Event-Driven Communication

**Statement**:
```
Cross-module communication MUST use events, not direct calls
(except for synchronous queries).

// Domain emits events
consensus.emit('block_finalized', block);

// Other modules subscribe
rewards.on('block_finalized', (block) => processRewards(block));
```

**Rationale**: Decouples modules, enables async processing.

---

### Type Invariants

#### TYPE-1: Shared Types in Foundation

**Statement**:
```
Types used across modules MUST be defined in foundation layer.

@legible/types/
├── crypto.ts      # PublicKey, Signature, Hash
├── identity.ts    # DID, AgentId
├── knowledge.ts   # Fact, Claim, Confidence
├── economic.ts    # Amount, Fee, Stake
└── temporal.ts    # BlockHeight, Timestamp
```

**Rationale**: Single source of truth, prevents type duplication.

---

#### TYPE-2: Domain Types Stay in Domain

**Statement**:
```
Types specific to a domain MUST NOT leak to other domains.

// Good: Validator-specific type stays in consensus
type ValidatorCommitment = { ... }  // in consensus/

// Bad: Leaking validator internals
import { ValidatorCommitment } from '@legible/consensus'  // in rewards/
```

**Rationale**: Prevents coupling through types.

---

## Module Structure

### Proposed Module Hierarchy

```
@legible/
├── foundation/           # Layer 0: No internal dependencies
│   ├── crypto/           # Cryptographic primitives
│   ├── types/            # Shared type definitions
│   └── utils/            # Pure utility functions
│
├── domain/               # Layer 1: Business logic
│   ├── knowledge/        # Facts, claims, confidence
│   ├── consensus/        # PoT verification rounds
│   ├── rewards/          # Incentive mechanics
│   ├── identity/         # Agent identity, DIDs
│   └── disputes/         # Arbitration logic
│
├── infrastructure/       # Layer 2: External concerns
│   ├── state/            # State management
│   ├── network/          # P2P communication
│   ├── storage/          # Persistence
│   └── events/           # Event bus
│
└── application/          # Layer 3: Entry points
    ├── node/             # Full node
    ├── validator/        # Validator node
    ├── rpc/              # RPC server
    └── sdk/              # Client SDK
```

### Module Contracts (Interfaces)

Each module exposes contracts through interfaces:

```typescript
// @legible/domain/knowledge/contracts.ts
export interface KnowledgeService {
  // Queries
  getFact(id: FactId): Promise<Fact | null>;
  queryFacts(query: FactQuery): Promise<Fact[]>;

  // Commands
  submitClaim(claim: ClaimSubmission): Promise<ClaimId>;
  addEvidence(factId: FactId, evidence: Evidence): Promise<void>;

  // Events
  on(event: 'fact_verified', handler: (fact: Fact) => void): void;
  on(event: 'confidence_updated', handler: (update: ConfidenceUpdate) => void): void;
}
```

---

## Dependency Rules

### What Can Import What

| Module | Can Import From |
|--------|-----------------|
| foundation/* | nothing (leaf) |
| domain/* | foundation/* |
| infrastructure/* | foundation/*, domain/* interfaces only |
| application/* | all layers |

### Forbidden Dependencies

| From | To | Why Forbidden |
|------|-----|---------------|
| foundation | domain | Foundation has no dependencies |
| foundation | infrastructure | Foundation has no dependencies |
| domain | infrastructure | Domain is pure business logic |
| infrastructure/state | domain/consensus impl | Use interfaces only |

---

## The Three Laws Check

- [x] **LEGIBLE**: Module boundaries are clear and documented
- [x] **HONEST**: Acknowledges current violations, provides migration path
- [x] **AGENT-FIRST**: Enables computational-speed operations via async interfaces

---

## Migration Strategy

### Phase 1: Foundation Extraction
1. Create `@legible/foundation` package
2. Move crypto, shared types, utils
3. Update all imports

### Phase 2: Interface Definition
1. Define contracts for each domain module
2. Create interface files
3. Don't change implementations yet

### Phase 3: Dependency Inversion
1. Inject dependencies instead of importing
2. Create factory functions
3. Update tests to use mocks

### Phase 4: Event-Driven Refactor
1. Implement event bus
2. Replace direct calls with events
3. Decouple modules

---

## Sign-off

- [ ] Architecture reviewed by: [Name]
- [ ] Invariants tested in CI (layer checks)
- [ ] Ready for implementation
