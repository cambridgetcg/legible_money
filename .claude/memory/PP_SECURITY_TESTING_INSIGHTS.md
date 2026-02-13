# PP Security Testing Insights

> Generated 2026-02-13 via full PP hierarchy review (TRUTH → UNDERSTANDING → BEAUTY → JUSTICE → CREATIVITY)

## Overview

8 security dimensions identified, 7 new test files created, filling gaps in D1, D2, D6, D7, D8.
**Result: 418 security tests across 23 files, 0 failures.**

## Dimension Coverage Matrix

| Dimension | Pre-existing Files | New Files | Total Tests |
|-----------|-------------------|-----------|-------------|
| D1: Cryptographic | 2 (KATs, properties) | 1 (differential) | 48 |
| D2: Consensus | 2 (byzantine, fairness) | 1 (chaos scenarios) | 50 |
| D3: Economic | 3 (invariants, game theory, loopholes) | 0 | ~80 |
| D4: Identity | 1 (DID injection) | 0 | 30 |
| D5: Network | 2 (RPC, mempool) | 0 | 51 |
| D6: Knowledge | 1 (adversarial claims) | 1 (counter-fact game) | ~46 |
| D7: State | **0** | **3** (atomicity, merkle, determinism) | **29** |
| D8: BVM | **0** | **2** (caller identity, gas exhaustion) | **14** |

## Key Insights by PP Dimension

### TRUTH (Grounding)
1. **SMT `get()` has shallow node traversal** — The InMemorySMT stores intermediate nodes with only `{ hash }` children, so `get()` can only traverse 1 level deep in a 256-depth tree. The `prove()` method correctly generates proofs (via `nodes.get()` lookup at each level), but returns `null` for the value. Existing tests handle this by supplying the known value when verifying inclusion proofs.

2. **State root determinism depends on version, not just data** — Two StateManagers with identical account balances but different commit counts produce different state roots because version is included in the root hash. This is correct behavior (prevents replay across versions).

### UNDERSTANDING (Depth)
3. **Transaction atomicity clones ALL 15+ sub-trees** — `beginTransaction()` calls `.clone()` on every sub-tree (accounts, validators, knowledge, claims, research, evidence, disputes, vesting, paymentChannels, projects, services, tasks, seeds, governance) plus schedule exports and subscription maps. This is expensive but correct for full rollback safety.

4. **Counter-fact economics require ≥10% honest success rate to deter spam** — At 10% success rate with 50% slash / 3× reward cap, expected value for random attacks is negative (-0.15× stake). Honest falsifiers profit at ≥50% success rate. The incentive structure works.

### BEAUTY (Elegance)
5. **BVM execution context cleanly separates identity from execution** — CallerDID is a context field, not controllable by bytecode. The AGENT_DID opcode reads from context, not from the stack. This prevents spoofing at the execution layer.

6. **Domain separation with fixed-length concatenation prevents collision** — The state root computation uses 64-char hex strings for each sub-root + 16-char version, with domain hashing. Fixed-length prevents a critical class of hash collision attacks where `root_A + root_B` could equal `root_C + root_D`.

### JUSTICE (Placement)
7. **Gas metering must reject at gas=0** — Even trivially simple programs (PUSH1, POP, STOP) consume gas. A gasLimit of 0n correctly fails, preventing zero-cost DoS. Gas metering is deterministic — identical programs consume identical gas.

8. **Static call mode correctly prevents SSTORE** — The BVM enforces read-only execution by checking `mode === "staticcall"` before allowing state-modifying opcodes, matching EVM safety guarantees.

### CREATIVITY (New Paths)
9. **Negative knowledge penalties are tiered by confidence** — High (≥0.7) → 8% penalty, Medium (≥0.4) → 4% penalty, Low → 1% penalty. This graduated response prevents over-punishment of uncertain contradictions while making high-confidence falsifications impactful.

10. **Chaos testing reveals boundary discipline** — Phase transitions in verification rounds strictly respect block deadlines. `transitionToReveal()` is a no-op if not in commit phase or before deadline. This prevents timing-based manipulation of the commit-reveal protocol.

## Patterns for Future Tests

### What works well
- **Boundary-value analysis** on bidirectional confidence thresholds (exact 0.6, -0.6, 0.2, -0.2 boundaries)
- **Differential testing** (run same operations on two fresh states, compare roots)
- **Fault injection** (rollback after multiple mutations, verify zero residue)
- **Economic invariant checking** (expected-value calculations for attack strategies)

### Known limitations to address
- SMT `get()` shallow traversal limits some proof-of-inclusion test patterns
- BVM TS engine doesn't have full host function integration (agent opcodes need mock hosts)
- No fuzz testing yet (would benefit from fast-check or similar)
- No cross-stack differential tests (TS vs Go) — blocked on Go test harness alignment

## File Inventory

```
tests/security/
├── d1-cryptographic/
│   ├── crypto-kats.test.ts          (22 tests)
│   ├── crypto-properties.test.ts    (16 tests)
│   └── crypto-differential.test.ts  (10 tests) ← NEW
├── d2-consensus/
│   ├── byzantine-fault.test.ts      (22 tests)
│   ├── selection-fairness.test.ts   (18 tests)
│   └── chaos-scenarios.test.ts      (10 tests) ← NEW
├── d3-economic/
│   ├── invariants.test.ts           (16 tests)
│   ├── game-theory.test.ts          (~20 tests)
│   └── loophole-*.test.ts           (76 tests across 3 files)
├── d4-identity/
│   └── did-injection.test.ts        (30 tests)
├── d5-network/
│   ├── rpc-security.test.ts         (27 tests)
│   └── mempool-network.test.ts      (24 tests)
├── d6-knowledge/
│   ├── adversarial-claims.test.ts   (~28 tests)
│   └── counter-fact-game.test.ts    (18 tests) ← NEW
├── d7-state/                        ← NEW DIRECTORY
│   ├── atomicity.test.ts            (13 tests)
│   ├── merkle-soundness.test.ts     (10 tests)
│   └── determinism.test.ts          (6 tests)
└── d8-bvm/                          ← NEW DIRECTORY
    ├── caller-identity.test.ts      (6 tests)
    └── gas-exhaustion.test.ts       (8 tests)
```
