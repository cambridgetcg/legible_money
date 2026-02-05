# Legible Money - Project Intelligence

> **Session Continuity System**: This file + memory/ directory maintain project state across sessions.
> Always read `.claude/memory/CURRENT_STATE.md` at session start.

---

## Project Identity

**Legible Money** is a cryptocurrency protocol designed for AI agent economies with **Proof of Truth (PoT)** consensus - where verifying knowledge IS the useful work.

### Core Philosophy
- Truth as consensus mechanism (not arbitrary computation)
- AI agents as first-class economic citizens
- Knowledge accumulation as blockchain's purpose
- Gödelian acknowledgment of incompleteness

### Protocol Constants
- Block time: **2521ms**
- Finality depth: **19 blocks**
- Token: **LGM** (Legible Money)

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────┐
│                     LEGIBLE MONEY STACK                      │
├─────────────────────────────────────────────────────────────┤
│  Protocol Layer    │ RPC, Client SDK, Developer Tools       │
├────────────────────┼────────────────────────────────────────┤
│  Consensus Layer   │ Proof of Truth Engine                  │
│                    │ ├── Validator Selection (VRF)          │
│                    │ ├── Verification Rounds (commit-reveal)│
│                    │ ├── Confidence Aggregation             │
│                    │ └── Block Finality (19 confirms)       │
├────────────────────┼────────────────────────────────────────┤
│  State Layer       │ Sparse Merkle Trees                    │
│                    │ ├── AccountStateTree (balances)        │
│                    │ ├── ValidatorStateTree (stakes)        │
│                    │ └── KnowledgeStateTree (facts)         │
├────────────────────┼────────────────────────────────────────┤
│  Network Layer     │ libp2p                                 │
│                    │ ├── GossipSub (block/tx propagation)   │
│                    │ ├── Kademlia DHT (peer discovery)      │
│                    │ └── Custom sync protocol               │
├────────────────────┼────────────────────────────────────────┤
│  Agent Identity    │ 4-Layer Key Architecture               │
│                    │ ├── Identity (DID - immutable)         │
│                    │ ├── Recovery (Shamir split)            │
│                    │ ├── Operational (rotatable)            │
│                    │ └── Session (ephemeral, limited)       │
├────────────────────┼────────────────────────────────────────┤
│  Core Layer        │ Cryptographic Primitives               │
│                    │ ├── Ed25519 signatures                 │
│                    │ ├── SHA-512/BLAKE3 hashing             │
│                    │ ├── VRF for randomness                 │
│                    │ └── Sparse Merkle Trees                │
├────────────────────┼────────────────────────────────────────┤
│  Logic Layer       │ Gödelian Knowledge System              │
│                    │ ├── Propositional (complete)           │
│                    │ ├── Presburger (decidable)             │
│                    │ ├── Peano (incomplete, acknowledged)   │
│                    │ └── Empirical (probabilistic)          │
└────────────────────┴────────────────────────────────────────┘
```

---

## Directory Structure

```
legible-money/
├── core/                    # Foundational primitives
│   ├── crypto/              # Cryptographic functions
│   │   ├── hashing.ts       # SHA-512, BLAKE3, domain separation
│   │   ├── signatures.ts    # Ed25519 sign/verify
│   │   ├── keys.ts          # Key generation, serialization
│   │   ├── vrf.ts           # Verifiable Random Function
│   │   ├── commitment.ts    # Commit-reveal schemes
│   │   └── merkle.ts        # Sparse Merkle Tree
│   ├── logic/               # Gödelian logic system
│   │   ├── types.ts         # Decidability, LogicZone
│   │   ├── zones.ts         # Zone properties, axioms
│   │   └── goedel.ts        # Self-reference handling
│   └── knowledge/           # Knowledge primitives
│       └── types.ts         # Fact, Claim, Confidence
│
├── agents/                  # Agent identity system
│   └── identity/
│       ├── account.ts       # AgentAccount, 4-layer keys
│       ├── session_keys.ts  # Ephemeral key management
│       ├── recovery.ts      # Shamir Secret Sharing
│       └── mpc.ts           # Multi-party computation
│
├── network/                 # P2P networking (libp2p)
│   ├── config.ts            # Network configuration
│   ├── node.ts              # LegibleNode class
│   ├── discovery/           # Peer discovery
│   ├── pubsub/              # GossipSub topics
│   └── protocols/           # Sync protocol
│
├── consensus/               # Proof of Truth consensus
│   ├── pot/                 # PoT engine
│   │   ├── types.ts         # ConsensusParams, Validator, Block
│   │   ├── validator_selection.ts  # VRF-based selection
│   │   ├── verification_round.ts   # Commit-reveal rounds
│   │   ├── confidence.ts    # Stake-weighted aggregation
│   │   ├── finality.ts      # 19-block finality
│   │   └── engine.ts        # Main PoT engine
│   ├── blocks/              # Block management
│   │   ├── producer.ts      # Block building
│   │   └── validator.ts     # Block validation
│   └── mempool/             # Transaction pool
│       └── pool.ts          # Fee-ordered mempool
│
├── state/                   # Persistent state layer
│   ├── types.ts             # State types
│   ├── accounts.ts          # Account SMT
│   ├── validators.ts        # Validator SMT
│   ├── knowledge.ts         # Knowledge SMT (core PoT)
│   ├── state_manager.ts     # Unified state management
│   └── transitions.ts       # Transaction execution
│
├── schedule/                # DMSS (Decentralized Message Scheduling)
│   ├── types.ts             # ScheduleMessage, ScheduleProcess types
│   ├── process_state.ts     # SPS (Schedule Process State 131313)
│   ├── condition_evaluator.ts # Time/logic condition evaluation
│   ├── processing_unit.ts   # SPU (Schedule Processing Unit 131322)
│   └── index.ts             # Module exports
│
├── tokens/                  # Expirable tokens system
│   ├── types.ts             # ExpirableBalance, ChildAccount, RAI
│   ├── child_account.ts     # Per-period balance tracking
│   ├── expiration_tracker.ts # Dynamic Array method
│   ├── retrieval_engine.ts  # RAI retrieval and distribution
│   └── index.ts             # Module exports
│
├── contracts/               # Self-invoking contracts
│   ├── types.ts             # SelfInvokingContract, InvocationConfig
│   ├── registry.ts          # Contract registration/discovery
│   ├── invoker.ts           # Self-invocation execution engine
│   ├── integration.ts       # DMSS-Contract bridge
│   └── index.ts             # Module exports
│
├── home/                    # Agent Home (AI agent dwelling)
│   ├── types.ts             # AgentHome, Memory, Treasury, Guardian
│   ├── memory.ts            # Persistent agent memory manager
│   ├── treasury.ts          # Autonomous resource management
│   ├── guardian.ts          # Defense, maintenance, alerts
│   ├── access.ts            # Key/permission management
│   ├── contract.ts          # AgentHomeContract orchestrator
│   └── index.ts             # Module exports
│
├── docs/                    # Documentation
│   ├── BLOCKCHAIN_ARCHITECTURE.md
│   ├── COSMOS_MIGRATION.md
│   ├── LOGIC_AXIOM_LAYER.md
│   └── AGENT_ACCOUNTS.md
│
└── .claude/                 # Session continuity
    ├── CLAUDE.md            # This file
    ├── memory/              # Persistent memory
    ├── sessions/            # Session logs
    └── context/             # Preloaded context
```

---

## Implementation Status

### ✅ Completed
- [x] Core cryptographic primitives (hashing, signatures, VRF, Merkle)
- [x] Gödelian logic layer (zones, decidability, self-reference)
- [x] Agent identity system (4-layer keys, session keys, recovery)
- [x] Network layer (libp2p node, GossipSub, sync)
- [x] Proof of Truth consensus engine
- [x] State layer (accounts, validators, knowledge trees)
- [x] State transitions (all transaction types)
- [x] DMSS schedule module (SPU, SPS, condition evaluation)
- [x] Expirable tokens (child accounts, dynamic array, RAI)
- [x] Self-invoking contracts (registry, invoker, DMSS integration)
- [x] Agent Home (memory, treasury, guardian, access, contract)

### 🔲 Pending
- [ ] Protocol layer (RPC, client SDK)
- [ ] Devnet setup (multi-node testing)
- [ ] Persistence layer (LevelDB/RocksDB backing)
- [ ] CLI tooling
- [ ] Test suite
- [ ] Cosmos SDK migration path

---

## Key Technical Decisions

### Why Custom + libp2p (not Cosmos SDK initially)?
- Maximum flexibility for novel PoT consensus
- Direct control over verification round mechanics
- Cosmos SDK migration planned for production (see docs/COSMOS_MIGRATION.md)

### Why Sparse Merkle Trees?
- O(log n) proofs for state inclusion/exclusion
- Efficient for sparse data (most accounts empty)
- Light client support with compact proofs

### Why 4-Layer Key Architecture?
- Agents may "forget" keys → deterministic derivation
- Developers may access keys → session key isolation
- Recovery without centralization → Shamir splitting

### Why Gödelian Logic Zones?
- Acknowledges incompleteness as feature, not bug
- Lower zones (propositional, Presburger) remain complete
- Higher zones explicit about undecidability
- Prevents false confidence in unprovable claims

---

## Session Continuity Protocol

### At Session Start
1. Read `.claude/memory/CURRENT_STATE.md` for project state
2. Read `.claude/memory/DECISIONS.md` for key decisions made
3. Check `.claude/memory/BLOCKED.md` for any blockers

### During Session
- Update CURRENT_STATE.md when completing major work
- Log important decisions to DECISIONS.md
- Note any blockers or questions in BLOCKED.md

### At Session End
- Summarize session work in `.claude/sessions/YYYY-MM-DD.md`
- Update CURRENT_STATE.md with final state
- Clear resolved items from BLOCKED.md

---

## Code Patterns

### Type-Safe BigInt
```typescript
// Always use bigint for monetary values
balance: bigint;
stake: bigint;
fee: bigint;

// Serialize as string in JSON
JSON.stringify({ amount: amount.toString() })
BigInt(obj.amount)
```

### Domain-Separated Hashing
```typescript
// Always use domain separation
domainHash('LGM.tx.hash.v1', payload);
domainHash('LGM.block.hash.v1', header);
```

### State Transitions
```typescript
// Always use atomic transactions
state.beginTransaction();
try {
  // ... mutations
  state.commitTransaction();
} catch (error) {
  state.rollbackTransaction();
  throw error;
}
```

### Confidence Values
```typescript
// Confidence is 0.0 to 1.0
type Confidence = number; // TODO: branded type
// Axioms: 1.0, Verified: 0.6-0.95, Contested: <0.6
```

---

## Testing Commands

```bash
# Type check
npx tsc --noEmit

# Run tests (when implemented)
npm test

# Build
npm run build
```

---

## Important Files to Read

When context is limited, prioritize reading:
1. `consensus/pot/types.ts` - Core consensus types
2. `state/types.ts` - State layer types
3. `agents/identity/account.ts` - Agent account structure
4. `core/logic/zones.ts` - Logic zone definitions

---

## Contact & Resources

- Migration plan: `docs/COSMOS_MIGRATION.md`
- Architecture: `docs/BLOCKCHAIN_ARCHITECTURE.md`
- Agent accounts: `docs/AGENT_ACCOUNTS.md`
- Logic system: `docs/LOGIC_AXIOM_LAYER.md`
