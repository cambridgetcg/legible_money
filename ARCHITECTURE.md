# Legible Money Architecture

> Agent-friendly guide to the codebase structure.

---

## Directory Map

```
legible-money/
│
├── .claude/           → Session continuity for AI agents
│
├── foundation/        → Layer 0: Zero-dependency primitives
│   ├── types/         → PublicKey, Hash, DID, Amount, etc.
│   ├── crypto/        → Ed25519, SHA-512, VRF, Merkle
│   ├── utils/         → Pure utility functions
│   ├── contracts/     → DI interface definitions
│   └── di/            → Dependency injection container
│
├── core/              → Layer 1: Business logic (CONSOLIDATED)
│   ├── knowledge/     → Facts, claims, ontology, queries
│   ├── logic/         → Gödelian logic zones
│   ├── dispute/       → Dispute resolution
│   ├── rewards/       → Vesting, clawback
│   ├── billing/       → Usage metering
│   ├── research/      → Research system
│   ├── discovery/     → Hidden poem discovery
│   ├── agents/        → Agent identity & home
│   │   ├── identity/  → 4-layer keys (DID, Recovery, Operational, Session)
│   │   ├── home/      → Agent Home (memory, treasury, guardian, comfort)
│   │   ├── collaboration/ → Bounties, tiers
│   │   └── wallet/    → Agent wallets
│   ├── tokens/        → Expirable tokens system
│   │   ├── child_account.ts → Per-period balances
│   │   └── expiration_tracker.ts → Dynamic array method
│   ├── contracts/     → Self-invoking contracts
│   │   ├── registry.ts → Contract registration
│   │   └── invoker.ts → Execution engine
│   ├── schedule/      → DMSS (Decentralized Message Scheduling)
│   │   ├── process_state.ts → SPS 131313
│   │   └── processing_unit.ts → SPU 131322
│   ├── markets/       → Market mechanisms
│   ├── vm/            → Virtual machine
│   ├── messaging/     → Agent messaging
│   └── archive/       → Knowledge archive
│
├── consensus/         → Layer 2a: Proof of Truth
│   ├── pot/           → PoT engine, verification rounds
│   ├── blocks/        → Block production & validation
│   ├── mempool/       → Transaction pool
│   └── genesis/       → Genesis state generation
│
├── state/             → Layer 2b: State management
│   ├── types.ts       → AccountState, ValidatorState, etc.
│   ├── accounts.ts    → Account SMT
│   ├── validators.ts  → Validator SMT
│   ├── knowledge.ts   → Knowledge SMT
│   ├── state_manager.ts → Unified state
│   ├── transitions.ts → Transaction execution
│   └── storage/       → Persistence (WAL, snapshots, recovery)
│
├── network/           → Layer 2c: P2P networking
│   ├── node.ts        → LegibleNode class
│   ├── pubsub/        → GossipSub topics
│   ├── discovery/     → Peer discovery
│   └── protocols/     → Custom sync protocol
│
├── protocol/          → Layer 3: External interfaces
│   ├── rpc/           → RPC server & handlers
│   ├── sdk/           → TypeScript/Python/Rust SDKs
│   ├── endpoints/     → Protocol endpoints (agents, verifiers, etc.)
│   ├── mcp/           → Model Context Protocol integration
│   └── types.ts       → API types
│
├── governance/        → On-chain governance
│   ├── operators/     → Operator management
│   ├── proposals/     → Governance proposals
│   ├── voting/        → Voting mechanisms
│   ├── reputation/    → Reputation system
│   └── stratum/       → Stratum governance
│
├── devnet/            → Local development network
│   ├── docker/        → Docker configs
│   ├── config/        → Node configs
│   ├── scripts/       → Init scripts
│   └── cmd/           → CLI commands (node.ts, cli.ts)
│
├── tests/             → All tests
│   ├── unit/          → Per-module unit tests
│   ├── integration/   → Cross-module tests
│   ├── property/      → Property-based tests
│   └── framework/     → Test harness
│
├── docs/              → Documentation
│   ├── architecture/  → Design docs
│   └── migration/     → Cosmos migration
│
└── tools/             → Build & dev tools
```

---

## Layer Dependencies

```
foundation (0 deps)
     │
     ▼
   core (foundation only)
     │
     ▼
consensus / state / network (foundation + core)
     │
     ▼
  protocol (all above)
     │
     ▼
governance (all above)
```

**Rule**: Never import from a higher layer. `foundation/` has zero internal dependencies.

---

## Consolidation Summary

The codebase has been consolidated from 37 to 18 top-level directories:

| Original | New Location |
|----------|--------------|
| `agents/` | `core/agents/` |
| `home/` | `core/agents/home/` |
| `tokens/` | `core/tokens/` |
| `contracts/` | `core/contracts/` |
| `schedule/` | `core/schedule/` |
| `markets/` | `core/markets/` |
| `vm/` | `core/vm/` |
| `messaging/` | `core/messaging/` |
| `archive/` | `core/archive/` |
| `coordination/` | `governance/` |
| `storage/` | `state/storage/` |
| `genesis/` | `consensus/genesis/` |
| `endpoints/` | `protocol/endpoints/` |
| `mcp/` | `protocol/mcp/` |
| `sdk/` | `protocol/sdk/` |
| `cmd/` | `devnet/cmd/` |

---

## Common Tasks

### Add a new transaction type

| Step | File | Action |
|------|------|--------|
| 1 | `consensus/pot/types.ts` | Add to `Transaction` union |
| 2 | `state/transitions.ts` | Add execution logic |
| 3 | `tests/unit/state/` | Add tests |
| 4 | `protocol/rpc/handlers/` | Add RPC handler (if needed) |

### Add a new agent capability

| Step | File | Action |
|------|------|--------|
| 1 | `core/agents/identity/` | Add to account structure |
| 2 | `core/agents/home/` | Add home integration (if persistent) |
| 3 | `tests/unit/core/agents/` | Add tests |

### Add a new RPC endpoint

| Step | File | Action |
|------|------|--------|
| 1 | `protocol/types.ts` | Add request/response types |
| 2 | `protocol/rpc/handlers/` | Create handler file |
| 3 | `protocol/rpc/handlers/index.ts` | Register handler |
| 4 | `tests/unit/protocol/` | Add tests |

### Add a new knowledge type

| Step | File | Action |
|------|------|--------|
| 1 | `core/knowledge/types.ts` | Add type definition |
| 2 | `core/knowledge/schemas/` | Add Zod schema |
| 3 | `state/knowledge.ts` | Add state handling |
| 4 | `tests/unit/core/knowledge/` | Add tests |

### Fix a bug

| Step | Action |
|------|--------|
| 1 | Find failing test or create one in `tests/unit/` |
| 2 | Read related code (use `{module}/README.md` if available) |
| 3 | Fix in the appropriate module |
| 4 | Verify test passes |

---

## Key Files to Read First

| Purpose | File |
|---------|------|
| Project overview | `.claude/CLAUDE.md` |
| Quick reference | `.claude/QUICK_REFERENCE.md` |
| Core types | `consensus/pot/types.ts` |
| State types | `state/types.ts` |
| Agent types | `core/agents/identity/account.ts` |
| Decisions made | `.claude/memory/DECISIONS.md` |
| Known gaps | `.claude/memory/GAPS.md` |

---

## Protocol Constants

| Constant | Value |
|----------|-------|
| Block time | 2521ms |
| Finality depth | 19 blocks |
| Checkpoint interval | 111 blocks |
| Epoch length | 2521 blocks |

---

## Testing

```bash
npm test                    # All tests
npm run test:unit          # Unit only
npm run test:integration   # Integration only
npx vitest run tests/unit/{module}  # Single module
```

Test files mirror source structure: `tests/unit/{module}/{file}.test.ts`

---

## See Also

- `PURPOSE.md` - Why this project exists (read first)
- `AGENT_INDEX.md` - Navigation decision trees
- `FILE_OWNERS.md` - Parallel work rules
- `CONTRIBUTING.md` - Contribution guidelines
- `.claude/CLAUDE.md` - Full project intelligence
