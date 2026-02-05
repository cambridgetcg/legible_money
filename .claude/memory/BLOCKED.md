# Blockers & Open Questions

> Active blockers and questions that need resolution.
> Remove items when resolved (move to DECISIONS.md if significant).

---

## Blockers

_None currently._

---

## Resolved Blockers

### B1: Network Layer is Placeholder (RESOLVED 2026-02-03)
**Was Blocking**: Multi-node devnet, integration testing, consensus simulation

**Resolution**: Implemented full libp2p integration:
- `network/node.ts` - Real libp2p with tcp, websockets, noise, yamux, gossipsub
- `network/protocols/sync.ts` - Added SyncProtocolHandler
- `cmd/node.ts` - Wired network to consensus
- Created consensus-network bridge adapter
- Added unit and integration tests

---

## Open Questions

### Q1: Persistence Backend
**Question**: LevelDB vs RocksDB vs SQLite for SMT persistence?

**Context**: Currently in-memory only. Need durable storage.

**Considerations**:
- LevelDB: Simple, proven in blockchain (Bitcoin, Ethereum)
- RocksDB: Better performance, more features, larger
- SQLite: SQL queries useful for debugging, but overhead

**Status**: Pending research

---

### Q2: Cosmos SDK Module Structure
**Question**: How to map current TypeScript modules to Cosmos SDK modules?

**Mapping Draft**:
- `x/knowledge` - Knowledge SMT, fact management
- `x/verification` - PoT verification rounds
- `x/agent` - Agent identity management

**Status**: Low priority until production migration

---

### Q3: Test Framework
**Question**: Vitest vs Jest for TypeScript tests?

**Leaning**: Vitest (faster, native ESM support)

**Status**: Decide when starting test phase

---

## Deferred Items

### D1: Binary Serialization
Currently using JSON for SMT values. Should migrate to:
- Protocol Buffers, or
- Custom binary encoding

**Why Deferred**: JSON works for prototype, optimize later.

---

### D2: Light Client Protocol
Need to define:
- State proof format
- Block header sync
- Verification proof format

**Why Deferred**: Focus on full node first.

---

_Last reviewed: 2026-02-03 (PP implementation review)_
