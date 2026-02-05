# Motivations & Reasoning

> The "why" behind every major choice. Not just what we decided, but why.
> Understanding motivation prevents repeating mistakes and enables consistent extension.

---

## Project Motivation

### Why Build Legible Money?

**Problem**: Existing blockchains optimize for:
- Computation (Ethereum) - smart contracts
- Speed (Solana) - transactions per second
- Interoperability (Cosmos) - chain communication

None optimize for **truth verification**.

**Opportunity**: AI agents will be economic actors. They need:
- Verified knowledge (not just data)
- Reputation systems (not just addresses)
- Identity that survives key loss
- Limited autonomy (session keys)

**Solution**: A blockchain where the useful work IS truth verification.

---

### Why "Proof of Truth"?

**Motivation**: Proof of Work wastes energy. Proof of Stake just locks capital. Neither produces value beyond security.

**Insight**: What if the consensus mechanism PRODUCED something valuable?
- PoW produces: security (and heat)
- PoS produces: security (and nothing)
- PoT produces: security AND verified knowledge

**The value proposition**:
```
Every block adds to the knowledge graph
Every verification makes the network smarter
The blockchain becomes more valuable over time (not just bigger)
```

---

### Why AI Agents First?

**Motivation**: Designing for humans first leads to human-centric assumptions:
- Users remember passwords (agents might not)
- Users control their keys (developers might access agent keys)
- Users act intentionally (agents might make mistakes)

**Design principle**: If it works for agents, it works for humans. Not vice versa.

**Agent-first features**:
- Deterministic key derivation (can "remember" by re-deriving)
- Session keys with capability limits (bounded damage)
- Reputation as first-class concept (not just address history)

---

## Architectural Motivations

### Why Custom Stack (Not Cosmos SDK)?

**Motivation at decision time**:

We considered:
1. Cosmos SDK (Go) - production-ready, IBC support
2. Substrate (Rust) - flexible, forkless upgrades
3. Custom (TypeScript) - full control

**Why Custom**:
- Novel consensus (PoT) doesn't fit existing frameworks
- Verification rounds need custom state machine
- Knowledge graph is not a standard module
- TypeScript enables rapid iteration

**Why Plan Cosmos Migration**:
- Cosmos is better for production (battle-tested)
- IBC enables knowledge sharing across chains
- Validator tooling is mature
- Custom prototype validates design first

**The strategy**: Build fast, migrate right.

---

### Why Three Separate SMTs?

**Motivation**: Considered single SMT with prefixed keys.

**Why separate**:
1. **Semantic clarity**: Account operations vs validator operations vs knowledge operations
2. **Caching efficiency**: Hot accounts don't evict cold validators
3. **Parallel operations**: Can generate proofs concurrently
4. **Debugging**: Can inspect one tree without others
5. **Upgrade flexibility**: Can change one tree's structure independently

**Trade-off accepted**: Combined state root computation is slightly more complex.

---

### Why Commit-Reveal for Verification?

**Motivation**: Validators verify claims. Without protection:

```
Validator A submits: "TRUE"
Validator B sees A, copies: "TRUE"
Validator C sees A, copies: "TRUE"
...
No one actually verified - they just copied
```

**Solution**: Commit-reveal

```
Phase 1 (commit): Submit hash(verdict || confidence || nonce)
Phase 2 (reveal): Submit actual values
Verify: hash matches commitment
```

**Why it works**:
- Can't change verdict after committing
- Can't read others' verdicts before committing
- Nonce prevents brute-force attack on 3-option hash

**Cost accepted**: Two-phase process is slower than single phase.

---

### Why VRF for Validator Selection?

**Motivation**: Need to select verifiers for each claim.

**Options**:
1. Round-robin: Predictable, gameable
2. Random seed: Who generates? Can be biased
3. VRF: Unpredictable, verifiable

**Why VRF**:
- Validator generates own proof locally
- Anyone can verify selection was fair
- No one can predict selection in advance
- Selection is stake-weighted (more stake = more likely)

**The insight**: VRF turns "random selection" into "provable selection."

---

### Why 4-Layer Key Architecture?

**Motivation**: Agents have unique challenges:

```
Challenge 1: Agents might "forget" keys between runs
Challenge 2: Developers might have access to agent keys
Challenge 3: Compromised key shouldn't drain entire account
Challenge 4: Identity should persist across key rotations
```

**Solution**: Layer the keys by sensitivity:

| Layer | Usage | If Compromised |
|-------|-------|----------------|
| DID | Identity | Can't be compromised (derived) |
| Recovery | Emergency | Full access (but split storage) |
| Operational | Daily | Can rotate, some damage |
| Session | Temporary | Limited damage (capped) |

**The insight**: Separate "identity" from "authority."

---

### Why Session Key Capability Limits?

**Motivation**: If developer can read agent's session key:

```
Without limits:
  Developer extracts session key
  Developer drains agent's account
  Agent is bankrupt

With limits:
  Developer extracts session key
  Session key can only spend 100 LGM/tx, 1000 LGM/day
  Damage is bounded
```

**Capabilities designed**:
- `maxSpendPerTx`: Single transaction limit
- `maxSpendPerDay`: Daily aggregate limit
- `expiresAt`: Time limit
- `allowedRecipients`: Whitelist (optional)
- `canStake/canVote`: Action whitelist

**Philosophy**: Principle of least privilege, enforced at protocol level.

---

### Why Gödelian Logic Zones?

**Motivation**: Claims vary in provability:

```
"1 + 1 = 2" - provable in arithmetic
"This program halts" - undecidable (halting problem)
"The sky is blue" - empirical (not formally provable)
```

**Problem with single zone**: Everything gets same confidence treatment.

**Solution**: Explicit zones with different rules:

| Zone | Example | Decidability |
|------|---------|--------------|
| Propositional | "A implies A" | Complete |
| Presburger | "2 + 3 = 5" | Decidable |
| Peano | "Goldbach's conjecture" | Incomplete |
| Empirical | "Water boils at 100°C" | Probabilistic |

**The insight**: Incompleteness is a feature. By acknowledging it, we avoid false confidence.

---

### Why Confidence Decay?

**Motivation**: Facts verified in 2025 might not be true in 2030.

**Problem with permanent facts**:
```
2025: "Company X is solvent" (verified, high confidence)
2027: Company X goes bankrupt
2030: Old fact still shows "high confidence"
```

**Solution**: Confidence decays logarithmically:
```
decay = log1p(age_in_blocks) * decay_rate
```

**Why logarithmic**:
- Fast initial decay: New facts need quick re-confirmation
- Slow later decay: Well-established facts are probably still true
- Never zero: Even old facts have some weight

**The insight**: Knowledge needs maintenance, not just creation.

---

### Why Knowledge Graph (References)?

**Motivation**: Facts don't exist in isolation.

```
Fact A: "All men are mortal"
Fact B: "Socrates is a man"
Fact C: "Socrates is mortal" (depends on A and B)
```

**Problem with flat facts**: If Fact A is revoked, Fact C should be reconsidered.

**Solution**: Explicit references between facts:
- `supports`: This fact supports another
- `contradicts`: This fact contradicts another
- `extends`: This fact extends another
- `specializes`: This fact is a special case
- `generalizes`: This fact is a generalization

**Behavior**:
```
When fact is revoked:
  Find all facts that reference it
  Reduce their confidence
  Mark for re-verification
```

**The insight**: Knowledge is a graph, not a list.

---

## Code-Level Motivations

### Why bigint Not number?

**Motivation**: JavaScript numbers are IEEE 754 floats.

```javascript
0.1 + 0.2 === 0.3  // false (floating point error)
Number.MAX_SAFE_INTEGER + 1 === Number.MAX_SAFE_INTEGER + 2  // true (overflow)
```

**For money**: This is unacceptable.

**Solution**: bigint for all monetary values
```typescript
balance: bigint;  // Exact, no overflow until very large
fee: bigint;
stake: bigint;
```

**Trade-off**: JSON doesn't support bigint natively, need toString() serialization.

---

### Why Domain-Separated Hashing?

**Motivation**: Hash collisions across domains.

```typescript
// Bad: Could collide
hash(transaction.bytes) === hash(block.bytes)  // Possible!

// Good: Can't collide
hash("LGM.tx.v1" + transaction.bytes) !== hash("LGM.block.v1" + block.bytes)
```

**Pattern**:
```typescript
domainHash(domain: string, data: string): Uint8Array {
  return hash(domain + ":" + data);
}
```

**The insight**: Separation of concerns applies to cryptography too.

---

### Why JSON Serialization (For Now)?

**Motivation**: Need to serialize state for SMT storage.

**Options**:
1. JSON: Human-readable, easy, has overhead
2. Protobuf: Compact, needs schema, complex
3. Custom binary: Most compact, most work

**Why JSON for prototype**:
- Fastest to implement
- Easy to debug (can read values)
- Good enough for correctness testing

**Why migrate later**:
- Production needs efficiency
- Protobuf is standard for Cosmos
- Binary is smaller on disk

**The strategy**: Correctness first, optimization later.

---

### Why Clone for Rollback (Not Snapshot)?

**Motivation**: Need atomic transactions with rollback.

**Attempt 1**: Store root hash, restore on rollback.
- Problem: SMT mutations are in-place
- Root hash doesn't let you restore tree

**Attempt 2**: Copy-on-write.
- Problem: Complex, error-prone

**Solution**: Full clone before transaction.
```typescript
beginTransaction() {
  this.checkpoint = {
    accounts: this.accounts.clone(),
    validators: this.validators.clone(),
    knowledge: this.knowledge.clone(),
  };
}
```

**Trade-off accepted**: Clone is O(cache size), not O(1). Acceptable for prototype.

---

## Future Motivation Notes

_Add here when new decisions are made. Capture the "why" immediately._

### Template for New Motivations

```
### Why [DECISION]?

**Motivation**: [Problem being solved]

**Options considered**:
1. [Option A]
2. [Option B]
3. [Option C]

**Why [Chosen Option]**:
- [Reason 1]
- [Reason 2]

**Trade-off accepted**: [What we gave up]

**The insight**: [Key learning]
```

---

_Understanding why helps extend the system consistently._
