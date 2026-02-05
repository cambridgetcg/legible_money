# Agent Navigation Index

> Decision trees for AI agents working on Legible Money.

---

## "Where do I add a new..."

### Type

```
Is it a primitive (used everywhere)?
├── YES → foundation/types/
│         Examples: PublicKey, Hash, DID, Timestamp, Amount
│
└── NO → Is it consensus-related?
    ├── YES → consensus/pot/types.ts
    │         Examples: Block, Transaction, Validator, VerificationRound
    │
    └── NO → Is it state-related?
        ├── YES → state/types.ts
        │         Examples: AccountState, ValidatorState
        │
        └── NO → {module}/types.ts
                  Examples: Fact → core/knowledge/types.ts
                           AgentHome → core/agents/home/types.ts
```

### Transaction Type

```
consensus/pot/types.ts     → Add to Transaction union type
        ↓
state/transitions.ts       → Add execution logic in applyTransaction()
        ↓
tests/unit/state/          → Add test file
```

### Agent Feature

```
Is it identity-related (keys, DID, recovery)?
├── YES → core/agents/identity/
│
└── NO → Is it persistent across sessions?
    ├── YES → core/agents/home/
    │         memory.ts (data), treasury.ts (funds), guardian.ts (defense)
    │
    └── NO → Is it collaboration-related?
        ├── YES → core/agents/collaboration/
        │
        └── NO → core/agents/ (create new subdirectory)
```

### RPC Endpoint

```
protocol/types.ts              → Add request/response types
        ↓
protocol/rpc/handlers/         → Create new handler file
        ↓
protocol/rpc/handlers/index.ts → Register in handlers object
        ↓
tests/unit/protocol/rpc/       → Add test
```

### Test

```
What type of test?
├── Unit (single module) → tests/unit/{module}/
│   Mirror source: core/agents/identity/account.ts → tests/unit/core/agents/account.test.ts
│
├── Integration (cross-module) → tests/integration/
│   Example: tests/integration/consensus/full_flow.test.ts
│
├── Property-based → tests/property/
│   Example: tests/property/crypto/signatures.test.ts
│
└── End-to-end → tests/e2e/
    Example: tests/e2e/devnet_scenario.test.ts
```

### Knowledge/Fact Feature

```
core/knowledge/types.ts        → Add type definition
        ↓
core/knowledge/schemas/        → Add Zod schema (if validatable)
        ↓
core/knowledge/ontology/       → Add ontology mapping (if domain-specific)
        ↓
state/knowledge.ts             → Add state tree handling
```

### Governance Feature

```
Is it proposal-related?
├── YES → governance/proposals/
│
└── NO → Is it voting-related?
    ├── YES → governance/voting/
    │
    └── NO → Is it operator-related?
        ├── YES → governance/operators/
        │
        └── NO → Is it reputation-related?
            ├── YES → governance/reputation/
            │
            └── NO → governance/ (evaluate fit)
```

### Token/Financial Feature

```
Is it about expirable tokens?
├── YES → core/tokens/
│         child_account.ts (per-period), expiration_tracker.ts (tracking)
│
└── NO → Is it about self-invoking contracts?
    ├── YES → core/contracts/
    │         registry.ts (register), invoker.ts (execute)
    │
    └── NO → Is it about scheduling?
        ├── YES → core/schedule/
        │
        └── NO → core/ (evaluate fit)
```

---

## "How do I understand..."

### The overall architecture

```
Read in order:
1. PURPOSE.md (why this exists - start here)
2. ARCHITECTURE.md (how it's organized)
3. .claude/CLAUDE.md (full project context)
4. .claude/QUICK_REFERENCE.md (one-page mental model)
```

### A specific module

```
{module}/README.md     → Module orientation (if exists)
        ↓
{module}/types.ts      → Type definitions
        ↓
{module}/index.ts      → Public API
        ↓
tests/unit/{module}/   → Usage examples in tests
```

### Why a decision was made

```
.claude/memory/DECISIONS.md    → Technical decisions log
        ↓
docs/architecture/             → Architecture docs
        ↓
.claude/context/ARCHITECTURE_RATIONALE.md → Design philosophy
```

### What's been implemented

```
.claude/memory/CURRENT_STATE.md → Implementation status
        ↓
.claude/memory/GAPS.md          → Known gaps
        ↓
.claude/memory/TEST_PROGRESS.md → Test coverage
```

### How to use an API

```
{module}/index.ts              → See what's exported
        ↓
.claude/context/API_PATTERNS.md → Common usage patterns
        ↓
tests/unit/{module}/           → Working examples
```

---

## "I'm getting an error in..."

### foundation/

```
Common issues:
- Type mismatch → Check foundation/types/ for correct types
- Crypto error → Verify domain separation (use domainHash, not raw hash)
- Serialization → BigInt must be serialized as string
```

### core/

```
Common issues:
- Missing dependency → Check layer rules (core depends only on foundation)
- Type error → Import types from {module}/types.ts, not index.ts
- Module moved → See ARCHITECTURE.md "Consolidation Summary" for new paths
```

### consensus/

```
Common issues:
- Validator selection → Check stake values are bigint, not number
- Block validation → Verify previous hash matches
- VRF issues → Check seed derivation
```

### state/

```
Common issues:
- Transaction failed → Check state/transitions.ts execution logic
- State root mismatch → Verify atomic transaction (beginTransaction/commit)
- Missing account → Accounts are created on first use
- Storage issues → Check state/storage/ for persistence
```

### network/

```
Common issues:
- Connection failed → Check libp2p configuration in network/config.ts
- Message not received → Verify pubsub topic subscription
```

### protocol/

```
Common issues:
- RPC handler not found → Check registration in handlers/index.ts
- Invalid request → Verify request matches type in protocol/types.ts
- Endpoint issues → Check protocol/endpoints/
```

---

## Module Quick Reference

| Module | Purpose | Key Files |
|--------|---------|-----------|
| `foundation/` | Primitives | `types/`, `crypto/` |
| `core/knowledge/` | Facts & claims | `types.ts`, `schemas/` |
| `core/agents/identity/` | Agent keys | `account.ts`, `session_keys.ts` |
| `core/agents/home/` | Agent dwelling | `memory.ts`, `treasury.ts` |
| `core/tokens/` | Expirable tokens | `child_account.ts`, `expiration_tracker.ts` |
| `core/contracts/` | Self-invoking | `registry.ts`, `invoker.ts` |
| `core/schedule/` | DMSS scheduling | `processing_unit.ts`, `process_state.ts` |
| `consensus/pot/` | PoT engine | `engine.ts`, `types.ts` |
| `consensus/genesis/` | Genesis state | `index.ts`, `treasury.ts` |
| `state/` | State trees | `state_manager.ts`, `transitions.ts` |
| `state/storage/` | Persistence | `snapshots.ts`, `recovery.ts` |
| `network/` | P2P | `node.ts`, `pubsub/` |
| `protocol/rpc/` | RPC API | `handlers/`, `server.ts` |
| `protocol/sdk/` | Client SDKs | `typescript/`, `python/`, `rust/` |
| `protocol/endpoints/` | Protocol endpoints | `agent_endpoint.ts`, `verifier.ts` |
| `governance/` | On-chain governance | `proposals/`, `voting/`, `reputation/` |

---

## Session Start Checklist

```
1. Read .claude/memory/CURRENT_STATE.md   → What's built
2. Read .claude/memory/BLOCKED.md         → Active blockers
3. Check .claude/plans/                   → Active plans
4. Run: npm run typecheck                 → Verify clean state
```

---

## See Also

- `PURPOSE.md` - Why this project exists
- `ARCHITECTURE.md` - Directory structure
- `FILE_OWNERS.md` - Parallel work rules
- `CONTRIBUTING.md` - How to contribute
