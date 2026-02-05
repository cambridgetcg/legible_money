# Project Evolution

> Chronological record of how the project evolved.
> What changed, when, and why.

---

## Timeline

### Phase 0: Conception

**Date**: Pre-2026-02-01

**State**: Idea only

**Key Concepts Established**:
- Proof of Truth consensus
- AI agent-first design
- Knowledge blockchain (not just data)

**Open Questions**:
- What stack to use?
- How exactly does verification work?
- How do agents manage keys?

---

### Phase 1: Foundation

**Date**: 2026-02-01

**State**: Core primitives implemented

**Work Done**:
- Cryptographic primitives (hashing, signatures, VRF)
- Gödelian logic zones
- Network layer (libp2p)

**Key Decisions**:
- Ed25519 for signatures
- SHA-512/BLAKE3 for hashing
- Domain-separated hashing
- Custom + libp2p stack

**Files Created**:
```
core/crypto/hashing.ts
core/crypto/signatures.ts
core/crypto/keys.ts
core/crypto/vrf.ts
core/crypto/commitment.ts
core/crypto/merkle.ts
core/logic/types.ts
core/logic/zones.ts
core/logic/goedel.ts
network/config.ts
network/node.ts
network/discovery/bootstrap.ts
network/pubsub/topics.ts
network/protocols/sync.ts
```

**Insight Gained**: Domain separation prevents subtle bugs.

---

### Phase 2: Identity

**Date**: 2026-02-02

**State**: Agent identity system implemented

**Work Done**:
- 4-layer key architecture
- Session key management
- Shamir secret sharing for recovery
- MPC wallet concept

**Key Decisions**:
- DID as permanent identity
- Session keys with capability limits
- Deterministic key derivation

**Files Created**:
```
agents/identity/account.ts
agents/identity/session_keys.ts
agents/identity/recovery.ts
agents/identity/mpc.ts
```

**Insight Gained**: Session keys solve the agent autonomy vs security dilemma.

**Problem Solved**: Agents can operate safely even if developers can read keys.

---

### Phase 3: Consensus

**Date**: 2026-02-02

**State**: Proof of Truth consensus engine implemented

**Work Done**:
- Validator selection via VRF
- Verification rounds with commit-reveal
- Confidence aggregation
- Block finality

**Key Decisions**:
- 2521ms block time
- 19-block finality
- Stake-weighted confidence formula
- Commit-reveal to prevent copying

**Files Created**:
```
consensus/pot/types.ts
consensus/pot/validator_selection.ts
consensus/pot/verification_round.ts
consensus/pot/confidence.ts
consensus/pot/finality.ts
consensus/pot/engine.ts
consensus/blocks/producer.ts
consensus/blocks/validator.ts
consensus/mempool/pool.ts
```

**Insight Gained**: VRF makes selection both random and provable.

**Problem Solved**: Validators can't be bribed in advance (unpredictable selection).

---

### Phase 4: State Layer

**Date**: 2026-02-03

**State**: Full state layer implemented

**Work Done**:
- Account state tree (balances, nonces)
- Validator state tree (stakes, delegations)
- Knowledge state tree (facts, references)
- Unified state manager
- All transaction types

**Key Decisions**:
- Three separate SMTs (not one)
- Confidence decay for facts
- Knowledge graph via references
- Clone-based rollback

**Files Created**:
```
state/types.ts
state/accounts.ts
state/validators.ts
state/knowledge.ts
state/state_manager.ts
state/transitions.ts
```

**Insight Gained**: Knowledge graph makes facts self-healing (revoke propagates).

**Problem Solved**: State is provable, atomic, and rollback-safe.

---

### Phase 5: Session Infrastructure

**Date**: 2026-02-03

**State**: Session continuity system implemented

**Work Done**:
- Project CLAUDE.md
- Memory system (state, decisions, insights)
- Prompts library
- Thinking frameworks
- Motivation documentation

**Files Created**:
```
.claude/CLAUDE.md
.claude/STARTUP.md
.claude/settings.json
.claude/memory/CURRENT_STATE.md
.claude/memory/DECISIONS.md
.claude/memory/BLOCKED.md
.claude/memory/INSIGHTS.md
.claude/memory/PROMPTS.md
.claude/memory/THINKING.md
.claude/memory/MOTIVATIONS.md
.claude/memory/EVOLUTION.md
.claude/context/CORE_TYPES.md
.claude/context/API_PATTERNS.md
.claude/sessions/2026-02-03.md
```

**Insight Gained**: Documentation is memory. Comprehensive docs = no context loss.

---

## Next Phases (Planned)

### Phase 6: Protocol Layer
- JSON-RPC server
- WebSocket subscriptions
- TypeScript client SDK
- Transaction builder

### Phase 7: Devnet
- Docker compose setup
- Genesis configuration
- Bootstrap nodes
- Basic block explorer

### Phase 8: Testing
- Unit tests per module
- Integration tests
- Consensus simulation
- State transition fuzzing

### Phase 9: Persistence
- LevelDB/RocksDB backing
- State snapshots
- Pruning strategy
- Recovery procedures

### Phase 10: Production
- Cosmos SDK migration
- Validator documentation
- Security audit
- Mainnet launch

---

## Evolution Patterns

### What Gets Better Over Time
- Code organization (refactor as patterns emerge)
- Error handling (add cases as they're discovered)
- Documentation (capture knowledge as it's gained)
- Type safety (add branded types, stricter checks)

### What Stays Fixed
- Core cryptographic choices
- Fundamental data structures (SMTs)
- Key consensus parameters
- Identity architecture

### What Might Change
- Serialization format (JSON → binary)
- Gas costs (placeholder → computed)
- Network protocol details
- Client SDK design

---

## Lessons Learned

### Technical Lessons
1. Start with types, implement later
2. Clone is simpler than copy-on-write
3. Separate SMTs are cleaner than prefixed keys
4. Domain separation prevents subtle bugs

### Process Lessons
1. Document decisions immediately
2. Capture motivations, not just choices
3. Use tasks for complex multi-step work
4. Update state at end of session

### Design Lessons
1. Agent-first thinking reveals edge cases
2. Decay prevents stale state
3. Proofs enable everything else
4. Simplicity is better than cleverness

---

_This is the story of the project. Add to it as we go._
