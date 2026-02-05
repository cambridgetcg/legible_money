# Thinking Processes

> How to reason about Legible Money. Mental models and thinking patterns.
> Use these frameworks when making decisions or solving problems.

---

## Core Mental Models

### 1. The Truth Stack

Think of Legible Money as layers of truth:

```
┌─────────────────────────────┐
│   KNOWLEDGE (Facts)         │  ← What we know
├─────────────────────────────┤
│   CONSENSUS (Agreement)     │  ← How we agree
├─────────────────────────────┤
│   STATE (Reality)           │  ← What is
├─────────────────────────────┤
│   CRYPTO (Proof)            │  ← Why we trust
└─────────────────────────────┘
```

**When designing**: Start at the bottom. Crypto enables state. State enables consensus. Consensus enables knowledge.

**When debugging**: Start at the top. Knowledge issue? Check consensus. Consensus issue? Check state. State issue? Check crypto.

---

### 2. The Agent Perspective

Always think: "What if I'm an AI agent using this?"

**Questions to ask**:
- Can I derive my keys if I "forget"?
- What's the worst damage if my key leaks?
- Can I operate autonomously within limits?
- Can I prove my identity without revealing secrets?

**The key tension**: Autonomy vs Safety
- More autonomy = more useful agent
- More safety = less damage from mistakes/attacks
- Session keys are the balance point

---

### 3. The Proof Perspective

Everything should be provable:

```
Claim → Verification → Fact (with proof)
Transaction → Execution → State (with proof)
Selection → VRF → Proposer (with proof)
```

**If something can't be proven**, it's not in the protocol. Proofs enable:
- Light clients
- Cross-chain verification
- Dispute resolution
- Auditability

---

### 4. The Decay Perspective

Nothing is permanent:

```
Facts decay (confidence reduces over time)
Validators decay (inactive = lower reputation)
Stakes decay (opportunity cost of locking)
Sessions decay (expire after duration/usage)
```

**Why decay matters**: Forces continuous participation. A "dead" chain with no activity shouldn't claim authority.

---

## Decision Frameworks

### When Adding a Feature

```
1. NECESSITY: Does this NEED to be in protocol?
   - Could it be a client-side feature?
   - Could it be a layer-2 feature?
   - Is it fundamental to truth verification?

2. COMPLEXITY: What's the simplest solution?
   - Start with the dumbest thing that works
   - Add complexity only when necessary
   - Every feature has maintenance cost

3. SECURITY: What could go wrong?
   - Can it be gamed?
   - Can it be attacked?
   - Does it leak information?

4. INTEGRATION: How does it fit?
   - Which modules does it touch?
   - Does it follow existing patterns?
   - Does it introduce new dependencies?
```

### When Choosing Between Approaches

```
1. List all viable options (minimum 2)
2. For each option:
   - Pros
   - Cons
   - Complexity (code, conceptual, operational)
   - Risk (what could go wrong)
3. Pick the option that:
   - Has acceptable cons
   - Has manageable complexity
   - Has lowest risk
   - Best fits existing patterns
4. Document in DECISIONS.md with rationale
```

### When Debugging

```
1. REPRODUCE: Can I reliably trigger the bug?
2. ISOLATE: What's the smallest case that fails?
3. TRACE: What's the execution path?
4. COMPARE: What's different from working case?
5. HYPOTHESIZE: What could cause this?
6. TEST: Does fixing hypothesis fix bug?
7. VERIFY: Are there other cases?
8. DOCUMENT: Add to tests/insights
```

---

## Domain-Specific Thinking

### Consensus Thinking

**Key question**: "How do we agree on truth?"

```
Truth = Stake-weighted consensus of verifiers

Agreement requires:
1. Selection (who verifies) → VRF
2. Isolation (can't copy each other) → Commit-reveal
3. Incentives (honest = rewarded) → Rewards/slashing
4. Finality (when is it settled) → Confirmation depth
```

**When designing consensus features**:
- What happens if validators collude?
- What happens if 34% are malicious?
- What happens if 51% are malicious?
- How much stake would attack cost?

---

### State Thinking

**Key question**: "What is the canonical state?"

```
State = f(Genesis, Transactions[0..n])

State must be:
1. Deterministic (same inputs = same output)
2. Provable (can prove any value)
3. Efficient (O(log n) proofs)
4. Atomic (all-or-nothing transitions)
```

**When designing state features**:
- Can this be computed deterministically?
- What needs to be stored vs computed?
- How will light clients verify this?
- What happens on rollback?

---

### Identity Thinking

**Key question**: "Who is this agent?"

```
Identity layers:
1. DID (permanent, derived from creation)
2. Recovery key (rare use, split storage)
3. Operational key (daily use, rotatable)
4. Session key (temporary, limited)
```

**When designing identity features**:
- Which key layer should authorize this?
- What's the blast radius if key leaks?
- Can agent recover from key loss?
- Can action be delegated to session key?

---

### Knowledge Thinking

**Key question**: "Is this fact true?"

```
Fact lifecycle:
1. Claim submitted (unverified)
2. Verification round (stake-weighted vote)
3. Fact created (if accepted)
4. Confidence decays (over time)
5. Challenge possible (if disputed)
6. Re-verification (if confidence low)
```

**When designing knowledge features**:
- How do we prevent false claims?
- How do we handle disputed claims?
- How do we update outdated knowledge?
- How do we connect related facts?

---

## Code Reasoning

### Where Does This Belong?

```
Is it a cryptographic primitive?     → core/crypto/
Is it a logic/reasoning primitive?   → core/logic/
Is it about agent identity?          → agents/identity/
Is it about peer communication?      → network/
Is it about block consensus?         → consensus/
Is it about state storage?           → state/
Is it user-facing?                   → protocol/ (future)
```

### What Pattern Should I Follow?

```
Managing a collection?     → Look at existing tree classes
Processing transactions?   → Look at transitions.ts
Handling events?          → Look at EventEmitter usage
Computing hashes?         → Look at domainHash pattern
Creating proofs?          → Look at prove/verify pattern
```

### How Should I Handle Errors?

```
Expected failure (bad input)?      → Return { success: false, reason }
Programming error (assertion)?     → throw new Error()
State error (invariant violated)?  → throw new StateError(code)
Recoverable error?                 → Return result object
Unrecoverable error?              → throw and let caller handle
```

---

## Architectural Invariants

These must ALWAYS be true:

### Cryptographic Invariants
- [ ] All hashes use domain separation
- [ ] All signatures use Ed25519
- [ ] All randomness uses VRF (not Math.random)
- [ ] All commitments include nonce

### State Invariants
- [ ] Total supply = sum of all balances + staked
- [ ] Validator total stake = own stake + delegated
- [ ] State root = hash(account_root, validator_root, knowledge_root)
- [ ] Nonce always increments

### Consensus Invariants
- [ ] Block number always increments
- [ ] Block timestamp >= previous timestamp
- [ ] Proposer must be valid validator
- [ ] Finalized blocks never change

### Knowledge Invariants
- [ ] Axioms have confidence 1.0
- [ ] Revoked facts are never "active"
- [ ] Fact references are bidirectional in indexes
- [ ] Confidence is always in [0, 1]

---

## Anti-Patterns to Avoid

### Don't
- Use floating point for monetary values (use bigint)
- Generate randomness without VRF
- Hash without domain separation
- Store secrets in state
- Make state transitions non-deterministic
- Assume validators are honest
- Trust timestamps precisely

### Do
- Use bigint for all token amounts
- Use VRF for all randomness
- Use domainHash for all hashing
- Use proofs for all claims
- Make everything deterministic
- Design for adversarial environment
- Allow for clock skew

---

## Questions to Ask

### Before Writing Code
1. What existing code does similar things?
2. What types do I need?
3. What errors can occur?
4. What needs to be tested?

### Before Committing Code
1. Does it follow existing patterns?
2. Is it deterministic?
3. Is it secure?
4. Is it documented?

### Before Closing Session
1. What did I learn?
2. What decisions did I make?
3. What's still uncertain?
4. What should be done next?

---

_Use these mental models for consistent, high-quality decisions._
