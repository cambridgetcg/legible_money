# Architecture Rationale

> Why the system is designed the way it is.
> Read this to understand the big picture.

---

## The Core Insight

**Traditional blockchains**: Consensus produces security. Security enables transactions.

**Legible Money**: Consensus produces TRUTH. Truth enables knowledge. Knowledge is valuable.

```
Traditional: Mining/Staking → Security → Transactions → ?
Legible Money: Verification → Truth → Knowledge → Value
```

The work of reaching consensus IS the valuable output.

---

## System Architecture

### Layer Diagram

```
┌──────────────────────────────────────────────────────────────┐
│                       APPLICATION                             │
│  (Future: dApps, AI agent services, knowledge markets)        │
├──────────────────────────────────────────────────────────────┤
│                       PROTOCOL                                │
│  RPC │ SDK │ CLI                                    [PENDING] │
├──────────────────────────────────────────────────────────────┤
│                       CONSENSUS                               │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│  │ Validator   │  │ Verification│  │ Block       │           │
│  │ Selection   │  │ Rounds      │  │ Finality    │           │
│  │ (VRF)       │  │ (Commit-    │  │ (19 blocks) │           │
│  │             │  │  Reveal)    │  │             │           │
│  └─────────────┘  └─────────────┘  └─────────────┘           │
├──────────────────────────────────────────────────────────────┤
│                       STATE                                   │
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐           │
│  │ Account     │  │ Validator   │  │ Knowledge   │           │
│  │ State Tree  │  │ State Tree  │  │ State Tree  │           │
│  │ (balances)  │  │ (stakes)    │  │ (facts)     │           │
│  └─────────────┘  └─────────────┘  └─────────────┘           │
├──────────────────────────────────────────────────────────────┤
│                       IDENTITY                                │
│  ┌─────────────────────────────────────────────────┐         │
│  │ DID → Recovery → Operational → Session          │         │
│  │ (permanent)  (split)    (rotatable)  (limited)  │         │
│  └─────────────────────────────────────────────────┘         │
├──────────────────────────────────────────────────────────────┤
│                       NETWORK                                 │
│  libp2p │ GossipSub │ Kademlia │ Sync Protocol               │
├──────────────────────────────────────────────────────────────┤
│                       CRYPTO                                  │
│  Ed25519 │ SHA-512/BLAKE3 │ VRF │ SMT │ Commitments          │
├──────────────────────────────────────────────────────────────┤
│                       LOGIC                                   │
│  Propositional │ Presburger │ Peano │ Empirical              │
│  (complete)      (decidable)  (incomplete) (probabilistic)   │
└──────────────────────────────────────────────────────────────┘
```

---

## Why Each Layer Exists

### Logic Layer
**Purpose**: Define what "truth" means in different domains.

**Without it**: All claims treated equally. Mathematical facts and empirical observations get same treatment. Can't express uncertainty properly.

**With it**: Claims categorized by provability. System acknowledges what can and can't be formally proven.

### Crypto Layer
**Purpose**: Provide mathematical guarantees.

**Primitives**:
- Signatures: Prove who authorized something
- Hashes: Create unforgeable commitments
- VRF: Generate provable randomness
- SMT: Prove state inclusion/exclusion

### Network Layer
**Purpose**: Connect nodes, propagate messages.

**Why libp2p**: Battle-tested, modular, supports our needs (pubsub, DHT, custom protocols).

### Identity Layer
**Purpose**: Define who participants are.

**Why 4 layers**: Different security needs for different actions. Session keys enable safe automation.

### State Layer
**Purpose**: Track canonical state of everything.

**Three trees because**: Different access patterns, different caching needs, cleaner semantics.

### Consensus Layer
**Purpose**: Agree on what's true.

**Proof of Truth because**: The work of verification IS the value. Not wasted computation.

---

## Data Flow

### Claim Submission Flow

```
1. Agent submits claim
   ├── claim: "The sky is blue"
   ├── domain: "observations.visual"
   ├── stake: 100 LGM
   └── references: []

2. Claim enters verification queue
   └── Mempool holds until round starts

3. Verification round begins
   ├── VRF selects N validators
   ├── Commit phase: validators submit hash(verdict)
   └── Reveal phase: validators reveal verdicts

4. Aggregation
   ├── Stake-weighted confidence computed
   ├── C = Σ(stake × reputation × verdict) / Σ(stake × reputation)
   └── Outcome determined (accepted/rejected/contested)

5. If accepted
   ├── Fact created in KnowledgeStateTree
   ├── Claimant rewarded
   └── Correct validators rewarded

6. If rejected
   ├── Claimant stake slashed
   └── Minority validators slashed
```

### Transaction Flow

```
1. Transaction created
   ├── type: "transfer"
   ├── from: DID
   ├── to: DID
   ├── amount: bigint
   ├── signature: Signature
   └── nonce: bigint

2. Transaction validated
   ├── Signature verified
   ├── Nonce checked
   ├── Balance checked
   └── Added to mempool

3. Block producer selected (VRF)
   └── Includes transaction in block

4. Block proposed
   ├── Transactions executed
   ├── State transitions applied
   └── New state root computed

5. Block validated by other nodes
   ├── All transactions valid?
   ├── State root correct?
   └── Proposer authorized?

6. Block finalized (19 confirmations)
   └── Transaction permanent
```

### State Transition Flow

```
1. Begin atomic transaction
   └── Clone current state for rollback

2. Execute transitions
   ├── Validate preconditions
   ├── Compute changes
   └── Apply to state trees

3. If success
   └── Commit: discard checkpoint

4. If failure
   └── Rollback: restore checkpoint
```

---

## Key Invariants

### Must Always Be True

**Cryptographic**:
- All hashes use domain separation
- All randomness from VRF
- All signatures Ed25519

**State**:
- Total supply = sum(balances) + sum(stakes)
- Block numbers strictly increase
- Nonces strictly increase per account

**Consensus**:
- Proposer is valid active validator
- Quorum is achieved before finality
- Commits match reveals

**Knowledge**:
- Axioms never decay
- Revoked facts never activate
- References are bidirectional in indexes

---

## Security Model

### Threat Model

**Assumed Honest**: >66% of stake
**Assumed Malicious**: <34% of stake

### Attack Vectors Considered

| Attack | Mitigation |
|--------|------------|
| Validator collusion | Commit-reveal (can't see others' votes) |
| Bribery | VRF selection (can't predict who to bribe) |
| Key theft | Session key limits (bounded damage) |
| Fake claims | Stake requirement + slashing |
| Sybil | Stake-weighted (need real stake) |
| Long-range | Checkpoints + finality |

### Trust Assumptions

1. Cryptographic primitives are secure (Ed25519, SHA-512)
2. Network eventually delivers messages
3. Supermajority is economically rational
4. Code is correct (needs auditing)

---

## Extension Points

### Where To Add Features

**New transaction type**:
1. Add type to `TransactionType` union
2. Add handler in `transitions.ts`
3. Add gas cost
4. Add tests

**New state**:
1. Add to appropriate SMT (or create new one)
2. Add state type
3. Add transition logic
4. Update state root computation if new SMT

**New consensus rule**:
1. Modify `consensus/pot/engine.ts`
2. Update validation in `blocks/validator.ts`
3. Consider fork implications

**New identity feature**:
1. Add to `AgentAccount` structure
2. Add management functions
3. Consider key layer implications

---

## Module Boundaries

### What Each Module Owns

**core/crypto**: Cryptographic primitives only. No business logic.

**core/logic**: Logic zones and decidability. No state.

**agents/identity**: Identity management. Doesn't know about consensus.

**network**: Peer communication. Doesn't know about consensus.

**consensus**: Block production and verification rounds. Uses state, doesn't own it.

**state**: Canonical state. Doesn't know about networking.

### Allowed Dependencies

```
crypto ← everything (foundation)
logic ← knowledge (for zone validation)
identity ← consensus, state (for account management)
network ← consensus (for message passing)
state ← consensus (for state updates)
consensus ← state (for reading/writing)
```

### Forbidden Dependencies

- crypto → anything (pure primitives)
- state → network (state is local)
- network → state (network is transport)

---

## Design Principles

### 1. Proofs Over Trust
Everything that can be proven should be proven. Don't trust, verify.

### 2. Explicit Over Implicit
Make assumptions explicit. If something decays, show it. If something is uncertain, say it.

### 3. Simple Over Clever
The simplest solution that works is best. Complexity is a liability.

### 4. Agent Over Human
Design for AI agents first. Humans can use agent-friendly systems. Not vice versa.

### 5. Decay Over Permanence
Nothing is forever. Built-in decay forces maintenance and prevents zombie state.

---

_This is why the system is the way it is._
