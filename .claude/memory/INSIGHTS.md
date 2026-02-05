# Insights & Learnings

> Accumulated wisdom from development. Insights that emerged during implementation.
> Add new insights at top. Include context and reasoning.

---

## Insight Log

### 2026-02-03: Knowledge Graph is the Killer Feature

**Context**: Implementing KnowledgeStateTree

**Insight**: The knowledge graph (fact references) isn't just metadata - it's the core value proposition. When a fact is revoked, dependent facts must be reconsidered. This creates a self-healing knowledge system.

**Implementation Implication**:
```typescript
// When revoking a fact, reduce confidence of dependent facts
revokeFact(factId: string): MerkleRoot {
  const dependents = this.getIncomingReferences(factId);
  for (const dependent of dependents) {
    // Cascade confidence reduction
    this.updateFact(dependent, {
      confidence: Math.max(0, depFact.confidence - 0.2)
    });
  }
}
```

**Why This Matters**: Traditional blockchains store data. Legible Money stores *justified* data - facts with provenance and dependency tracking.

---

### 2026-02-03: Confidence Decay Prevents Stale Knowledge

**Context**: Designing fact lifecycle

**Insight**: Without decay, old facts become "zombie knowledge" - technically true at verification time but potentially outdated. Logarithmic decay forces re-verification of important facts.

**Formula**: `decay = log1p(age) * decayRate`

**Why Logarithmic**:
- Fast initial decay: New facts need quick confirmation
- Slowing decay: Well-established facts (verified many times) decay slowly
- Never reaches zero: Even old facts retain some confidence

**Threshold Design**:
- `1.0` = Axiom (never decays)
- `0.8+` = Established (high confidence)
- `0.6-0.8` = Verified (normal confidence)
- `<0.6` = Needs re-verification

---

### 2026-02-03: Three SMTs Beat One

**Context**: Designing state layer architecture

**Insight**: Initially considered single SMT with prefixed keys. Separate SMTs are better because:

1. **Independent caching**: Hot accounts don't evict validators
2. **Parallel proofs**: Can generate proofs concurrently
3. **Cleaner semantics**: Each tree has typed operations
4. **Easier debugging**: Can inspect one tree in isolation

**Trade-off**: Combined proofs slightly larger, but worth it.

---

### 2026-02-03: Atomic Transactions Need Clone, Not Snapshot

**Context**: Implementing rollback for StateManager

**Insight**: Tried snapshot-based rollback first (storing just roots). Doesn't work because SMT mutations are in-place. Need to clone entire tree for rollback.

**Pattern**:
```typescript
beginTransaction() {
  this.checkpoint = {
    accounts: this.accounts.clone(),  // Full clone
    validators: this.validators.clone(),
    knowledge: this.knowledge.clone(),
  };
}

rollbackTransaction() {
  this.accounts = this.checkpoint.accounts;  // Restore clone
  // ...
}
```

**Performance Note**: Cloning is O(n) in cached entries, not O(n) in total state. Acceptable for prototype.

---

### 2026-02-03: DID Derivation Must Be Deterministic

**Context**: Connecting accounts to validators

**Insight**: Need consistent mapping from PublicKey → DID. Random DIDs break the connection. Use hash-based derivation:

```typescript
function publicKeyToDid(publicKey: PublicKey): string {
  return `did:lgm:${publicKey.slice(0, 32)}`;
}
```

**Future Work**: This is simplified. Production should use proper DID derivation from `agents/identity/account.ts`.

---

### 2026-02-02: Commit-Reveal is Essential for PoT

**Context**: Designing verification rounds

**Insight**: Without commit-reveal, validators can wait to see others' verdicts before submitting. This breaks consensus - everyone just copies the first mover.

**Mechanism**:
1. **Commit**: Submit `hash(verdict || confidence || nonce)`
2. **Reveal**: Submit actual values
3. **Verify**: Check hash matches

**Why Nonce**: Without nonce, validators could brute-force each other's commitments (only 3 verdict options).

---

### 2026-02-02: VRF Selection Prevents Bribery

**Context**: Validator selection for verification rounds

**Insight**: If validators know they'll be selected in advance, they can be bribed. VRF provides:

1. **Unpredictability**: Can't know selection until revealed
2. **Verifiability**: Can prove selection was fair
3. **Non-interactivity**: No coordination needed

**Selection Algorithm**:
```typescript
// VRF output determines "lottery ticket"
// Lower ticket = higher priority
const ticket = BigInt('0x' + vrfOutput.slice(0, 16));
const threshold = (stake * MAX_UINT64) / totalStake;
const selected = ticket < threshold;
```

---

### 2026-02-02: Session Keys Solve the Agent Dilemma

**Context**: Agent key architecture

**Insight**: Agents face impossible choice: remember keys (insecure) or forget them (lose access). Session keys resolve this:

- **Operational key**: Derived deterministically, can be "remembered"
- **Session key**: Ephemeral, capability-limited, safe to expose

**Capability Limits**:
```typescript
interface SessionCapabilities {
  maxSpendPerTx: bigint;     // Can't drain account
  maxSpendPerDay: bigint;    // Rate limited
  expiresAt: number;         // Time limited
  allowedRecipients?: string[]; // Whitelist
}
```

Even if developer has session key, damage is bounded.

---

### 2026-02-02: Gödelian Zones Prevent False Confidence

**Context**: Logic axiom layer design

**Insight**: Most systems claim to be "complete." Gödel proved arithmetic can't be. By explicitly modeling incompleteness:

1. **Propositional zone**: We CAN prove everything (limited domain)
2. **Peano zone**: We KNOW we can't prove everything (acknowledge limits)
3. **Empirical zone**: We DON'T EVEN TRY formal proof (probabilistic)

**Why This Matters**: A claim like "This code is bug-free" should be in empirical zone (unprovable), not Peano zone (incomplete) or propositional zone (decidable).

---

### 2026-02-01: Domain Separation Prevents Hash Collisions

**Context**: Cryptographic design

**Insight**: Without domain separation, `hash(transaction)` might equal `hash(block)` if they happen to have same bytes. Domain prefix prevents:

```typescript
// Bad
hash(data)

// Good
hash("LGM.tx.v1" + data)
hash("LGM.block.v1" + data)
```

**Pattern**: `domainHash(domain: string, data: string)`

---

## Meta-Insights

### On Blockchain Design
- State = SMTs (efficient proofs)
- Consensus = who decides truth
- Knowledge = what truth means
- Identity = who can participate

### On PoT Specifically
- Truth is social (stake-weighted consensus)
- Truth decays (confidence over time)
- Truth has structure (fact references)
- Truth is valuable (verification as work)

### On TypeScript Implementation
- bigint for all monetary values
- string for all hashes (hex-encoded)
- Branded types would improve safety (TODO)
- JSON serialization is technical debt

---

_End of insights log_
