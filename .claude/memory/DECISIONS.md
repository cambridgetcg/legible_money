# Technical Decisions Log

> Immutable log of key technical decisions made during development.
> Add new decisions at the top. Never delete entries.

---

## 2026-02-03: Confidence Quantification Framework

**Decision**: Implement multidimensional confidence with epistemic categories

**Context**: Confidence is not a single number. A mathematical proof and a weather observation cannot have the same confidence semantics. Need to encode epistemic humility into the protocol.

**Key Design Choices**:

1. **Epistemic Categories**: Seven categories (analytic, formal, protocol, empirical, historical, predictive, social) each with:
   - Maximum confidence ceiling (eternal truths get 1.0, social consensus caps at 0.85)
   - Decay half-life (eternal truths don't decay, predictions decay fast)
   - Minimum verifiers required
   - Verification method

2. **Bayesian Evidence Aggregation**: Evidence updates belief, doesn't replace it
   - Likelihood ratios for each piece of evidence
   - Independence modeling (correlated evidence weighted lower)
   - Contradiction handling (conflicts cap maximum achievable confidence)

3. **Multidimensional Output**: Rich confidence structure including:
   - Value (the headline number)
   - Provenance (how it was calculated)
   - Stability (how volatile over time)
   - Coherence (how much sources agree)
   - Sensitivities (what could change it)

4. **Fact Maturity Levels**:
   - Emerging: Can be freely challenged
   - Established: Requires evidence to challenge
   - Canonical: Requires governance to dispute

5. **Confidence Propagation**: When a fact's confidence changes:
   - Propagate to derived facts (5% threshold)
   - Detect circular dependencies with penalty
   - Limit propagation depth (10 hops)

**Master Formula**:
```
C_final = min(C_max, C_base × M_temporal × M_adversarial)
```

**Rationale**:
- AI agents need to understand not just "what we know" but "why we think we know it"
- Enables rational economic decisions in uncertain environments
- Honest uncertainty is more valuable than false certainty

**Files**:
- `core/knowledge/types.ts` - Types and categories
- `consensus/pot/confidence.ts` - Calculations
- `state/knowledge.ts` - Propagation

---

## 2026-02-03: State Layer Architecture

**Decision**: Three separate SMTs combined into unified StateRoot

**Context**: Need efficient state proofs for accounts, validators, and knowledge.

**Options Considered**:
1. Single SMT with prefixed keys
2. Separate SMTs with combined root
3. Nested SMT structure

**Choice**: Option 2 - Separate SMTs

**Rationale**:
- Cleaner separation of concerns
- Independent caching strategies
- Parallel proof generation
- Easier to reason about

**Trade-offs**:
- Slightly larger combined proofs
- More complex root computation

---

## 2026-02-03: Knowledge Confidence Decay

**Decision**: Logarithmic confidence decay over time

**Context**: Facts should require re-verification to maintain high confidence.

**Formula**: `decay = log1p(age) * decayRate`

**Rationale**:
- Fast initial decay encourages fresh verification
- Slows over time (well-established facts decay slowly)
- Prevents "set and forget" knowledge

**Parameters**:
- `decayRate`: 0.001 (tunable)
- `REVERIFICATION_THRESHOLD`: 0.6

---

## 2026-02-03: Transaction Gas Costs

**Decision**: Fixed gas costs per transaction type (for now)

**Costs**:
- Transfer: 21,000
- Stake/Unstake: 50,000
- Delegate: 60,000
- Claim Submit: 100,000
- Challenge: 80,000
- Governance Propose: 150,000

**Note**: These are placeholder values. Production should compute based on actual execution cost.

---

## 2026-02-02: Proof of Truth Consensus

**Decision**: Commit-reveal verification rounds with VRF selection

**Context**: Prevent validators from copying each other's verdicts.

**Mechanism**:
1. Validators selected via VRF (stake-weighted)
2. Commit phase: Submit hash(verdict, confidence, nonce)
3. Reveal phase: Submit actual values
4. Aggregation: Stake-weighted confidence

**Confidence Formula**:
```
C_final = Σ(stake_i × reputation_i × verdict_i) / Σ(stake_i × reputation_i)
```

---

## 2026-02-02: Block Timing Constants

**Decision**: Protocol timing constants

**Values**:
- Block time: 2521ms
- Finality: 19 blocks

**Rationale**: Optimized for Proof of Truth verification rounds.

---

## 2026-02-02: Agent Key Architecture

**Decision**: 4-layer key hierarchy

**Layers**:
1. **Identity** (DID) - Immutable, derived from agent ID
2. **Recovery** - Cold, Shamir-split (2-of-3)
3. **Operational** - Warm, rotatable
4. **Session** - Hot, ephemeral, capability-limited

**Problem Solved**: Agents may forget keys, developers may have access.

**Solution**: Deterministic derivation + session key isolation.

---

## 2026-02-02: Blockchain Stack Choice

**Decision**: Custom TypeScript + libp2p, migrate to Cosmos SDK later

**Options Evaluated**:
1. Cosmos SDK (Go)
2. Substrate (Rust)
3. Custom + libp2p
4. Ethereum L2
5. Solana
6. Custom from scratch

**Choice**: Option 3 for prototype, Option 1 for production

**Rationale**:
- Maximum flexibility for novel PoT consensus
- TypeScript for rapid iteration
- Cosmos SDK for production (IBC, validator tooling)

**Migration Plan**: See `docs/COSMOS_MIGRATION.md`

---

## 2026-02-01: Gödelian Logic Zones

**Decision**: Compartmentalized logic zones acknowledging incompleteness

**Zones**:
1. **Propositional** - Complete, decidable (Boolean)
2. **Presburger** - Decidable arithmetic (no multiplication)
3. **Peano** - Incomplete (full arithmetic)
4. **Empirical** - Probabilistic, not formally provable

**Key Insight**: Incompleteness is a feature, not a bug. Lower zones maintain full provability while higher zones explicitly acknowledge limits.

---

## 2026-02-01: Cryptographic Primitives

**Decision**: Ed25519 + SHA-512/BLAKE3

**Signatures**: Ed25519
- Fast, compact (64 byte signatures)
- Deterministic
- Well-audited implementations

**Hashing**:
- SHA-512/256 for compatibility
- BLAKE3 for performance-critical paths

**Domain Separation**: All hashes include domain prefix
- Pattern: `domainHash('LGM.purpose.v1', data)`

---

_End of decisions log_
