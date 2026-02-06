# The Tree of Knowledge

> **The epistemic backbone of Legible Money.**
> Every fact verified, every confidence justified, every dependency explicit.

---

## Philosophy

Knowledge doesn't exist in isolation. Chemistry emerges from physics. Biology emerges from chemistry. Cognition emerges from biology. Society emerges from cognition. This isn't metaphor -- it's the actual dependency structure of reality.

The Tree of Knowledge makes these dependencies explicit and enforceable. A claim about protein folding (biology) genuinely depends on claims about molecular bonds (chemistry), which depend on claims about electron behavior (physics). When we verify knowledge, we must respect this hierarchy.

Traditional blockchains store data. Legible Money stores *justified* data -- facts with provenance, dependency tracking, and epistemic humility about what can and cannot be known.

### Three Core Commitments

1. **Confidence is not a number.** It is a multidimensional structure: *what* we believe, *why* we believe it, *how* we calculated it, *how stable* it is, and *what could change it*.
2. **Incompleteness is a feature.** Godel proved that arithmetic cannot be both complete and consistent. Rather than pretending otherwise, the Tree of Knowledge compartmentalizes logic into zones where completeness holds and zones where it provably cannot.
3. **Knowledge decays.** A fact verified yesterday is more reliable than a fact verified last year. Confidence decays over time, at rates determined by the epistemic category of the claim. Eternal truths (mathematics, protocol definitions) are exempt. Everything else must be re-verified to maintain confidence.

---

## The Seven Strata

Knowledge is organized into seven **ontological strata**, ordered from most fundamental to most emergent:

```
             THE ONTOLOGICAL EMERGENCE LADDER

  Level 6  TECHNOLOGICAL  ─────────────────────────  Ceiling: 1.00
           Computation, networks, AI, designed systems
                              ↑
  Level 5  SOCIAL  ─────────────────────────────────  Ceiling: 0.85
           Economics, politics, culture, institutions
                              ↑
  Level 4  COGNITIVE  ──────────────────────────────  Ceiling: 0.90
           Perception, reasoning, consciousness
                              ↑
  Level 3  BIOLOGICAL  ─────────────────────────────  Ceiling: 0.95
           Cells, organisms, ecosystems, evolution
                              ↑
  Level 2  CHEMICAL  ───────────────────────────────  Ceiling: 0.98
           Elements, molecules, reactions, compounds
                              ↑
  Level 1  PHYSICAL  ───────────────────────────────  Ceiling: 0.99
           Mechanics, thermodynamics, electromagnetism
                              ↑
  Level 0  FUNDAMENTAL  ────────────────────────────  Ceiling: 1.00
           Physics constants, particles, mathematics
```

### Why These Ceilings?

Each stratum has a **confidence ceiling** -- the maximum confidence any fact in that stratum can achieve. These ceilings reflect the inherent epistemic limits at each level of emergence:

| Stratum | Level | Ceiling | Logic Zone | Epistemic Category | Reasoning |
|---------|-------|---------|------------|-------------------|-----------|
| Fundamental | 0 | 1.00 | Peano | Analytic | Definitionally true (axioms, constants) |
| Physical | 1 | 0.99 | Peano | Empirical | Measurement has inherent uncertainty |
| Chemical | 2 | 0.98 | Empirical | Empirical | Experimental conditions introduce variability |
| Biological | 3 | 0.95 | Empirical | Empirical | Living systems have natural variation |
| Cognitive | 4 | 0.90 | Empirical | Social | Subjective experience cannot be fully externalized |
| Social | 5 | 0.85 | Empirical | Social | Social facts are constructed and context-dependent |
| Technological | 6 | 1.00 | Propositional | Protocol | Protocol-defined truths are certain by construction |

A claim about human behavior (social stratum) can never achieve 0.99 confidence -- not because we haven't gathered enough data, but because social facts are inherently less certain than physical facts.

> **Source**: `core/knowledge/ontology/types.ts:63-71` (STRATUM_CONFIDENCE_CEILING)

### Stratum Dependencies

Higher strata emerge from lower strata. The dependency rules are:

```
fundamental  →  depends on nothing (axiomatic)
physical     →  depends on [fundamental]
chemical     →  depends on [fundamental, physical]
biological   →  depends on [fundamental, physical, chemical]
cognitive    →  depends on [biological, chemical, physical, fundamental]
social       →  depends on [cognitive, biological, chemical, physical, fundamental]
technological → depends on [all] (designed systems can reference any stratum)
```

These dependencies are enforced at claim submission time. A biology fact that claims to depend only on social facts -- skipping chemistry and physics -- will be rejected.

> **Source**: `core/knowledge/ontology/strata.ts:32-56` (VALID_DEPENDENCIES)

---

## Godelian Logic Zones

Not all claims are created equal. Some can be formally proven. Others cannot -- and the Tree of Knowledge acknowledges this explicitly through **logic zones**.

```
  Zone 0: PROPOSITIONAL  ─── Complete, decidable, no Godel
          Boolean logic. Every statement provable or disprovable.

  Zone 1: PRESBURGER  ────── Decidable arithmetic (addition only)
          No multiplication. Still complete.

  Zone 2: PEANO  ─────────── INCOMPLETE (Godel's theorems apply)
          Full arithmetic. Some true statements are unprovable.

  Zone 3: SET THEORY  ────── Incomplete, richer axioms
          Godel applies. Consistency unprovable within system.

  Zone 4: EMPIRICAL  ─────── Outside formal systems entirely
          Evidence-based. Probabilistic. Never formally proven.
```

### What This Means in Practice

When an agent submits a claim, the logic zone determines what kind of verification is possible:

| Zone | Verification Method | What "Verified" Means |
|------|--------------------|-----------------------|
| Propositional | Proof search | Formally proven true |
| Presburger | Decision procedure | Formally decided |
| Peano | Proof search + oracle | Proven OR accepted as unprovable |
| Set Theory | Proof + consensus | Best available evidence |
| Empirical | Observation + consensus | Stake-weighted agreement |

A mathematical theorem (Zone 2) that is verified by proof has fundamentally different epistemic status than an economic forecast (Zone 4) that is verified by expert consensus. The Tree of Knowledge tracks this difference.

### Undecidable Claims

When a claim is undecidable in its logic zone, four resolution paths exist:

1. **Accept undecidability**: Record the claim as undecidable, with explanation
2. **Oracle**: Defer to an external authority (real-world measurement)
3. **Consensus**: Accept via stake-weighted validator agreement
4. **New axiom**: Propose a governance vote to add a new axiom that resolves the claim

### Meta-Axioms

Four meta-axioms are built into the protocol:

| ID | Statement | Zone |
|----|-----------|------|
| meta_1 | The system is incomplete (Godel I) | Meta |
| meta_2 | Undecidability is expected, not an error | Meta |
| meta_3 | Consistency cannot be proven within the system (Godel II) | Meta |
| meta_4 | Lower zones maintain independence from higher zones | Meta |

> **Source**: `core/logic/types.ts` (LogicZone, Decidability, GoedelianStatus)

---

## Epistemic Categories

Every fact belongs to one of seven **epistemic categories** that determine its confidence semantics, decay behavior, and verification requirements:

| Category | Ceiling | Decay Half-Life | Reverification | Verification Method |
|----------|---------|-----------------|----------------|-------------------|
| **Analytic** | 1.00 | Eternal | Never | Definition |
| **Formal** | 1.00 | Eternal | Never | Proof |
| **Protocol** | 1.00 | Eternal | Never | Construction |
| **Empirical** | 0.99 | ~3.2 days (111,111 blocks) | Required | Observation |
| **Historical** | 0.95 | ~16 days (555,555 blocks) | Required | Records |
| **Predictive** | 0.90 | ~14 hours (22,222 blocks) | Required | Models |
| **Social** | 0.85 | ~3.2 days (111,111 blocks) | Required | Consensus |

### Eternal vs Temporal Truth

The first three categories -- analytic, formal, protocol -- represent **eternal truths**. They do not decay. "1 + 1 = 2" is as true today as it was yesterday. "Block 47231 has hash 0xabc..." is immutably true.

The remaining categories represent **temporal truths**. Their confidence decays over time, reflecting the fact that observations go stale, predictions resolve, and social consensus shifts.

```
   Confidence
    1.0 ┤ ─────────────────────── Analytic (eternal)
        │
    0.9 ┤  ╲
        │    ╲  Empirical
    0.8 ┤      ╲
        │        ╲
    0.7 ┤    ╲     ╲─────────── min confidence (0.33)
        │      ╲
    0.6 ┤        ╲ Predictive
        │          ╲
    0.5 ┤            ╲─────── min confidence (0.22)
        │
        └──────────────────────── Time →
```

### Decay Formula

Confidence decays exponentially based on category-specific half-life:

```
decayed_confidence = initial_confidence * (0.5 ^ (blocks_since_verification / half_life))
effective_confidence = max(min_confidence, decayed_confidence)
```

Decay is **lazy** -- it is calculated on query, not stored. This avoids background jobs and keeps the state tree clean. When a fact is queried, the effective confidence is computed from the stored confidence and the time elapsed.

**Reverification bonus**: When a fact is re-verified, its staleness clock resets and it receives a +0.11 confidence bonus (for non-eternal categories). This incentivizes re-verification of important facts.

> **Source**: `consensus/pot/confidence.ts:71-120` (DECAY_CATEGORIES), `state/knowledge.ts:715` (getEffectiveConfidence)

---

## Multidimensional Confidence

Confidence in the Tree of Knowledge is not a single number. It is a multidimensional structure that captures the full epistemic state of a fact:

```typescript
MultidimensionalConfidence {
  value: 0.87              // The headline number (0-1)

  basis: {                 // WHY we believe this
    category: "empirical"
    primaryMethod: "observation"
    evidenceCount: 5
    verificationCount: 8
    derivedFrom: ["fact-123", "fact-456"]
  }

  provenance: {            // HOW the confidence was calculated
    evidenceContribution: 0.35     // 35% from evidence quality
    verificationContribution: 0.52 // 52% from validator consensus
    temporalAdjustment: 0.95       // 5% decay since last verification
    adversarialAdjustment: 1.0     // No manipulation detected
    epistemicCeiling: 0.99         // Category ceiling
  }

  stability: "stable"      // How much confidence changes over time
  coherence: "consistent"  // How much sources agree with each other

  sensitivities: {         // What could change the confidence
    toNewEvidence: 0.15       // How much new evidence would shift it
    toContradiction: 0.30     // How vulnerable to contradiction
    toTimeDecay: 0.08         // Rate of time-based decay
  }
}
```

### Stability Levels

| Level | Meaning |
|-------|---------|
| **Volatile** | Confidence changes frequently, large swings |
| **Unstable** | Moderate changes, sensitive to new evidence |
| **Stable** | Confidence holds steady, well-established |
| **Anchored** | Very high inertia, resistant to change |

### Source Coherence

| Level | Meaning |
|-------|---------|
| **Contradictory** | Sources actively disagree |
| **Mixed** | Some agreement, some disagreement |
| **Consistent** | Sources mostly agree |
| **Unanimous** | All sources in full agreement |

> **Source**: `core/knowledge/types.ts` (MultidimensionalConfidence, EpistemicBasis, ConfidenceProvenance)

---

## The Master Confidence Formula

The final confidence for a fact is calculated by the master formula:

```
C_final = min(C_max, C_base * M_temporal * M_adversarial)

Where:
  C_max       = Epistemic ceiling for this category
  C_base      = Evidence confidence (40%) + Verification confidence (60%)
  M_temporal  = Time decay factor * reverification bonus
  M_adversarial = Sybil adjustment * manipulation adjustment
```

### Step-by-Step Calculation

**Step 1: Evidence-Based Confidence** (40% weight)
Bayesian aggregation of all evidence, accounting for evidence independence, strength, and contradictions.

**Step 2: Verification-Based Confidence** (60% weight)
Stake-weighted aggregation of validator votes:
```
verification_confidence =
    (stake_weighted × 0.5)
  + (reputation_weighted × 0.3)
  + (agreement_bonus × 0.2)
```

**Step 3: Combined Base**
```
C_base = evidence_confidence × 0.4 + verification_confidence × 0.6
```

**Step 4: Temporal Decay**
Apply category-specific exponential decay based on blocks since last verification.

**Step 5: Epistemic Ceiling**
Cap to the maximum allowed by the fact's epistemic category.

**Step 6: Adversarial Adjustment**
Reduce if manipulation or sybil behavior is detected.

> **Source**: `consensus/pot/confidence.ts:771` (calculateMasterConfidence)

---

## Domain Hierarchy

Within each stratum, knowledge is organized into **domains** -- hierarchical categories using dot notation:

```
physics                          (stratum: fundamental)
├── mechanics
│   ├── classical
│   ├── quantum
│   └── relativistic
├── thermodynamics
├── electromagnetism
└── particles
    ├── quarks
    ├── leptons
    └── bosons

chemistry                        (stratum: chemical)
├── organic
├── inorganic
├── physical
└── biochemistry

biology                          (stratum: biological)
├── cellular
├── molecular
├── evolutionary
└── ecology

computation                      (stratum: technological)
├── networks
│   ├── protocols
│   └── cryptography
├── ai
│   ├── training
│   └── inference
└── blockchain
    ├── consensus
    └── state
```

Each domain:
- Belongs to exactly one stratum
- Inherits confidence ceilings from its stratum
- Has a default logic zone and epistemic category
- Tracks fact count, axiom count, and governance proposals
- Can have child domains registered through governance

### Domain Resolution

When a claim is submitted, its domain is resolved:

1. **Exact match**: `physics.mechanics.quantum` found directly
2. **Parent fallback**: `physics.mechanics.quantum.entanglement` resolves to `physics.mechanics.quantum`
3. **Root fallback**: `unknown.domain.path` resolves to root domain

### Domain Registration

New domains are registered through governance:

```
Base stake:        100 LGM
Governance domains: 1,000 LGM (root children or stratum <= 2)
```

Registration validates:
- Domain ID format: lowercase, dots, `[a-z][a-z0-9_]*(\.[a-z][a-z0-9_]*)*`
- Parent exists
- Stratum >= parent stratum (child can't be more fundamental than parent)
- Dependencies are valid for the stratum

> **Source**: `core/knowledge/ontology/tree.ts` (registerDomain, createDomainTree)

---

## Cross-Stratum Rules

The Tree of Knowledge enforces strict rules about how facts can reference each other across strata.

### The Support Rule

**A fact can only support facts in the same stratum or higher.**

Chemistry facts can support biology claims (chemistry is more fundamental), but biology facts cannot directly support chemistry claims (biology emerges from chemistry, not the other way).

```
  Chemistry fact SUPPORTS Biology fact       (level 2 → level 3)  OK
  Chemistry fact SUPPORTS Chemistry fact     (level 2 → level 2)  OK
  Biology fact SUPPORTS Chemistry fact       (level 3 → level 2)  INVALID
```

### The Contradiction Rule

**Contradictions can cross any stratum boundary.**

If a social science claim contradicts a physics claim, that's valid -- it means one of them is wrong. Contradictions are how the knowledge graph self-corrects.

### The Extension Rule

**Extensions follow the same direction as support.** Higher strata extend lower strata, not the reverse.

### Cross-Stratum Discount

When a fact depends on facts from other strata, a discount is applied:

```
Discount = 5% * (1 - dependency_strength) per stratum boundary crossed
```

Four types of cross-stratum dependency, each with default strength:

| Dependency Type | Strength | Example |
|-----------------|----------|---------|
| **Emergence** | 0.90 | Biology emerges from chemistry |
| **Composition** | 0.95 | Molecules compose from atoms |
| **Constraint** | 0.80 | Physics constrains what biology allows |
| **Application** | 0.70 | Technology applies physics principles |

> **Source**: `core/knowledge/ontology/strata.ts` (VALID_DEPENDENCIES), `core/knowledge/ontology/confidence.ts` (calculateOntologicalConfidence)

---

## Genesis Axioms

The Tree of Knowledge is seeded with four genesis axioms -- foundational truths that anchor the entire system:

| ID | Claim | Type | Confidence |
|----|-------|------|------------|
| axiom_0 | The founding truth | Founding | 1.0 |
| axiom_1 | A = A (identity) | Mathematical | 1.0 |
| axiom_2 | 1 + 1 = 2 (arithmetic) | Mathematical | 1.0 |
| axiom_3 | LGM is the native token | Protocol | 1.0 |

Axiom 0 is special: it has a commitment hash and reveal conditions. When the conditions are met, the founding truth is revealed.

Genesis axioms have confidence 1.0, are eternal (no decay), and can only be modified through governance supermajority.

> **Source**: `core/knowledge/types.ts` (Axiom, GENESIS_AXIOMS)

---

## Fact Lifecycle

A fact in the Tree of Knowledge passes through a complete lifecycle from claim to potential expiration:

```
              THE FACT LIFECYCLE

 ┌──────────┐     ┌──────────┐     ┌──────────┐
 │  CLAIM   │────→│ PENDING  │────→│VERIFYING │
 │ SUBMITTED│     │          │     │          │
 └──────────┘     └────┬─────┘     └────┬─────┘
                       │                 │
                  (withdrawn)    ┌───────┴───────┐
                                │               │
                          ┌─────┴────┐   ┌──────┴─────┐
                          │ ACCEPTED │   │  REJECTED  │
                          │          │   │ (stake     │
                          └─────┬────┘   │  slashed)  │
                                │        └────────────┘
                          ┌─────┴────┐
                          │  ACTIVE  │←──── reverification
                          │   FACT   │
                          └──┬──┬──┬─┘
                             │  │  │
                    ┌────────┘  │  └────────┐
                    │           │            │
              ┌─────┴─────┐ ┌──┴───┐  ┌─────┴──────┐
              │CHALLENGED │ │SUPER-│  │  EXPIRED   │
              │           │ │SEDED │  │ (decay/    │
              └─────┬─────┘ └──────┘  │  staleness)│
                    │                  └────────────┘
              ┌─────┴─────┐
              │  REVOKED  │
              │ (disproven)│
              └───────────┘
```

### Maturity Levels

As facts age and accumulate verifications, they progress through maturity levels:

| Maturity | Min Confidence | Min Age | Min Verifications | Significance |
|----------|---------------|---------|-------------------|--------------|
| **Emerging** | 0.50 | 0 blocks | 1 | Can be freely challenged |
| **Established** | 0.80 | 100,000 blocks (~3 days) | 5 | Requires evidence to challenge |
| **Canonical** | 0.95 | 500,000 blocks (~15 days) | 10 | Requires governance to dispute |

### Expiration

Facts expire through three mechanisms (checked lazily on query):

1. **Explicit expiration**: The fact's `expiresAt` block has passed
2. **Confidence decay**: Effective confidence has dropped below 0.30
3. **Reverification staleness**: Age exceeds 2x the category's decay half-life without re-verification

Eternal categories (analytic, formal, protocol) are exempt from staleness expiration.

> **Source**: `state/knowledge.ts:824` (isFactExpired), `core/knowledge/types.ts` (FactMaturity, MATURITY_CRITERIA)

---

## The Knowledge Graph

Facts don't exist in isolation. They reference each other, forming a **directed knowledge graph** stored alongside the Tree of Knowledge.

### Reference Types

| Type | Direction | Meaning |
|------|-----------|---------|
| **supports** | A → B | A provides evidence for B |
| **contradicts** | A → B | A provides evidence against B |
| **extends** | A → B | A adds detail or scope to B |
| **depends_on** | A → B | A's truth requires B's truth |

Each reference carries an **inference strength** (0.0 - 1.0) representing how strongly the source fact's truth affects the target fact's truth.

### Bidirectional Indexes

The graph maintains both directions:
- **Outgoing references**: "What facts does this fact cite?"
- **Incoming references**: "What facts cite this fact?"

This enables both forward reasoning (what follows from this?) and backward reasoning (what supports this?).

### Confidence Propagation

When a fact's confidence changes, the change propagates through the knowledge graph via BFS:

```
  Fact A confidence drops from 0.9 to 0.6
        │
        ├──→ Fact B (depends_on A, strength 0.8)
        │     New confidence: min(premises) × strength
        │     = 0.6 × 0.8 = 0.48
        │     │
        │     └──→ Fact D (depends_on B, strength 0.9)
        │           = 0.48 × 0.9 = 0.43
        │
        └──→ Fact C (supports A, strength 0.7)
              No change (C supports A, not the reverse)
```

**The Weakest-Link Principle**: A derived fact's confidence cannot exceed the minimum confidence of its premises, multiplied by the inference strength.

```
derived_confidence = min(premise_confidences) * inference_strength
```

**Propagation rules**:
- Threshold: Only propagate if change magnitude >= 5%
- Max depth: 10 hops (prevents runaway cascades)
- Cycle detection: Tarjan's algorithm detects circular dependencies
- Cycle penalty: Facts in cycles receive a confidence penalty of `0.4 * (1 / log2(cycle_length))`

### Revocation Cascade

When a fact is revoked (proven false), all facts that depend on it have their confidence reduced. This creates a **self-healing knowledge system** -- removing a false premise automatically flags everything built on it.

> **Source**: `state/knowledge.ts:1027` (propagateConfidenceChange), `state/knowledge.ts:1120` (calculateDerivedConfidence)

---

## Verification Rounds

When a claim enters verification, it passes through a commit-reveal protocol that prevents validators from copying each other:

```
  ┌───────────────────────────────────────────────────────┐
  │                  VERIFICATION ROUND                    │
  │                                                        │
  │   COMMIT PHASE          REVEAL PHASE        AGGREGATE  │
  │   (~222 blocks)         (~333 blocks)                  │
  │                                                        │
  │   Validators submit     Validators reveal    System    │
  │   hash(verdict +        actual verdict +     calculates│
  │   confidence +          confidence +         final     │
  │   nonce)                nonce                confidence │
  │                                                        │
  │   VRF proof required    Hash must match      Rewards   │
  │   Signature verified    Signature verified   & slashes │
  └───────────────────────────────────────────────────────┘
```

### Validator Selection

Validators are selected via **Verifiable Random Function** (VRF):
- Selection is stake-weighted (more stake = higher probability)
- Selection is unpredictable (can't know who's selected until revealed)
- Selection is verifiable (anyone can check the proof)
- This prevents bribery -- you can't bribe a validator who doesn't know they'll be selected

### Commitment

Each validator generates:
```
commitment_hash = SHA-512/256(
  domain_hash("LGM.commitment.v1", {
    roundId, verdict, confidence, referencesChecked, nonce
  })
)
```

The nonce prevents brute-force reverse engineering (only 3 verdict options without it).

### Reveal

Validators reveal their actual verdict, confidence, and nonce. The system verifies:
1. Hash matches commitment
2. Signature is valid
3. VRF proof confirms legitimate selection

### Aggregation

Outcome is determined by stake-weighted voting:

| Condition | Outcome |
|-----------|---------|
| >= 67% stake confirms | **Accepted** (fact created) |
| >= 67% stake rejects | **Rejected** (stake slashed 50%) |
| < 50% participation | **Insufficient** (stake returned) |
| Neither threshold met | **Contested** (stake returned) |

Final confidence for accepted claims uses basis-point arithmetic to prevent overflow:
```
weighted_confidence = sum(validator_confidence_bps * validator_stake) / total_participating_stake
```

### Validator Consequences

| Behavior | Consequence |
|----------|-------------|
| Correct verdict | Verification reward (1 LGM) |
| Wrong verdict | Slashed 5% of stake |
| "Uncertain" verdict | No reward, no slash |
| Committed but didn't reveal | Slashed 1% of stake |

> **Source**: `consensus/pot/verification_round.ts` (createVerificationRound, aggregateResults)

---

## Challenges and Disputes

Any fact can be challenged. The dispute system routes challenges through an escalating arbiter process that ultimately adjusts the fact's confidence.

### Challenge Initiation

An agent stakes LGM (minimum 111 LGM) and presents counter-evidence. The targeted fact is marked as "challenged."

### Dispute Phases

```
  INITIATE → EVIDENCE → DEFENSE → ARBITRATION → SETTLEMENT
  (stake)    (commit/   (respond)  (arbiter     (distribute
              reveal)              votes)        rewards)
```

### Escalation Tiers

| Tier | Stake Multiplier | Arbiter Count | Purpose |
|------|-----------------|---------------|---------|
| 1 | 1.1x | Small panel | Quick resolution |
| 2 | 2.2x | Medium panel | Serious disputes |
| 3 | 5.5x | Large panel | Fundamental challenges |

### Confidence Adjustment

The dispute outcome adjusts fact confidence:

| Outcome | Adjustment |
|---------|------------|
| **Challenger wins** | confidence * (1 - 0.55) * additional penalties |
| **Defender wins** | confidence * (1 + 0.11) |
| **Draw** | confidence * (1 - 0.22) |
| **Timeout** | confidence * (1 - 0.11) |

Additional modifiers:
- **Exposed falsehood**: Extra 20% reduction
- **Escalation tier penalty**: 2% per tier above 1
- **Challenge history decay**: 0.95^(previous_challenge_count) per repeated challenge

If confidence drops below the retraction threshold, the fact is **revoked** and its status changes to "revoked," triggering confidence propagation to all dependent facts.

### Economic Stakes

| Parameter | Value |
|-----------|-------|
| False claim slash | 55% of stake |
| Failed challenge slash | 33% of stake |
| Winner share of loser's stake | 66% |

> **Source**: `consensus/pot/challenge_integration.ts` (challengeFact, routeDisputeOutcomeToFact, calculateConfidenceAdjustment)

---

## Evidence Types

The Tree of Knowledge grades evidence by type, each with an intrinsic strength:

| Evidence Type | Strength | Description |
|--------------|----------|-------------|
| **on_chain** | 1.00 | Verifiable directly from blockchain state |
| **cryptographic_proof** | 0.99 | Mathematical proof of correctness |
| **computational** | 0.95 | Reproducible computation |
| **formal_proof** | 0.95 | Formal logic proof |
| **chainlink_price_feed** | 0.92 | Decentralized oracle price data |
| **chainlink_por** | 0.91 | Proof of reserves |
| **chainlink_ccip** | 0.90 | Cross-chain interop verification |
| **chainlink_functions** | 0.88 | Decentralized computation |
| **peer_reviewed** | 0.85 | Published and reviewed |
| **institutional** | 0.80 | Institutional attestation |
| **oracle** | 0.70 | Single oracle report |
| **archived_web** | 0.60 | Web content preserved on IPFS/Arweave |
| **attestation** | 0.50 | Individual testimony |
| **inference** | 0.40 | Logical inference from other facts |

### Evidence Independence

The system models whether evidence sources are independent, which affects how much they strengthen a claim:

```typescript
EvidenceIndependence {
  sourceIndependence: 0.9      // Different organizations?
  methodIndependence: 0.8      // Different measurement methods?
  temporalIndependence: 0.7    // Different time periods?
  geographicIndependence: 0.6  // Different locations?
  aggregateIndependence: 0.75  // Combined score
}
```

Correlated evidence (same lab, same method, same time) is weighted lower than independent evidence. This prevents a single biased source from dominating by submitting multiple nominally-different pieces of evidence.

### Evidence Archiving

Evidence is automatically archived to multiple providers for permanence:

| Provider | Use Case |
|----------|----------|
| **IPFS** | Content-addressed storage |
| **Arweave** | Permanent storage |
| **Wayback Machine** | Web page preservation |
| **On-chain** | Direct blockchain storage |

> **Source**: `core/evidence/types.ts` (EVIDENCE_TYPE_STRENGTH, EvidenceIndependence, ArchiveProvider)

---

## Citation Economics

The Tree of Knowledge rewards knowledge contribution through a citation-based economic model:

### Seven Contribution Dimensions

| Dimension | Weight | Meaning |
|-----------|--------|---------|
| **Origination** | 30% | Submitting new facts |
| **Validation** | 25% | Verifying claims accurately |
| **Ancestry** | 20% | Prior facts that enabled discovery |
| **Challenge** | 10% | Challenging incorrect facts |
| **Evidence** | 10% | Providing supporting evidence |
| **Research** | 5% | IRL research verification |
| **Infrastructure** | -- | Network and protocol infrastructure |

### Reward Flow

```
  ALL REWARDS
      │
      ├──→ 22% → Research Treasury (split across 5 funds)
      │
      └──→ 78% → Contribution-Weighted Distribution
                   ├── 30% Origination
                   ├── 25% Validation
                   ├── 20% Ancestry (with 50% decay per level)
                   ├── 10% Challenge
                   ├── 10% Evidence
                   └──  5% Research
```

### Fundamentality Bonuses

Facts that become foundational -- cited by many other facts -- earn bonus rewards:

| Fundamentality Score | Threshold | Bonus |
|---------------------|-----------|-------|
| Notable | >= 0.25 | 1.1x |
| Foundational | >= 0.50 | 1.25x |
| Fundamental | >= 0.75 | 1.5x |
| Axiomatic | >= 0.90 | 2.0x |

### The Counterfactual Principle

Rewards are adjusted by "what would have happened if this contributor hadn't participated?"

| Scenario | Adjustment |
|----------|------------|
| Near-duplicate discovery | 0.5x discount |
| Redundant validation | 0.3x discount |
| First validator bonus | 1.5x multiplier |
| Quorum-breaker bonus | 1.3x multiplier |

### Anti-Gaming Measures

| Measure | Adjustment |
|---------|------------|
| Self-citation | 0.1x discount |
| Sybil detection (< 3 unique validators) | 0.2x discount |
| Evidence stuffing (> 5 items) | 0.5x per extra item |

> **Source**: `core/truth/contribution.ts` (CONTRIBUTION_DIMENSIONS, FUNDAMENTALITY_THRESHOLDS, COUNTERFACTUAL_ADJUSTMENTS)

---

## The Knowledge Archive

The Archive is the query layer for the Tree of Knowledge, optimized for AI agent access.

### Four Specialized Indexes

```
  DOMAIN INDEX
  └── domain:<id> → facts in domain
  └── domain_children:<id> → child domains
  └── Pattern matching: physics.*

  CONFIDENCE INDEX
  └── conf:<bucket>:<fact_id> → bucketed (100 buckets, 0.01 granularity)
  └── conf_domain:<domain>:<bucket> → by domain
  └── conf_stratum:<stratum>:<bucket> → by stratum

  REFERENCE GRAPH
  └── ref_out:<fact>:<type>:<target> → outgoing
  └── ref_in:<fact>:<type>:<source> → incoming
  └── Transitive traversal, confidence ratio

  SEMANTIC INDEX
  └── emb:<fact_id> → embedding vector (768-dim, BERT-compatible)
  └── Cosine similarity search
  └── Domain and stratum filtering
```

### Query Types

**Truth Evaluation**: "Is this fact true?"
```typescript
archive.evaluateTruth('fact-001')
// → { verdict: 'true', confidence: 0.87, ratio: 0.92,
//    supportingFacts: [...], contradictingFacts: [...] }
```

**Domain Query**: "What do we know about physics?"
```typescript
archive.queryByDomain('physics.*', { includeDescendants: true, minConfidence: 0.8 })
```

**Semantic Search**: "Find facts about quantum entanglement"
```typescript
archive.search('quantum entanglement effects')
// Returns facts ranked by cosine similarity
```

**Cross-Stratum Reasoning**: "Trace dependencies across strata"
```typescript
queryService.processIntent(sessionId, {
  type: 'connect',
  domain: 'biology.molecular',
  depth: 3
})
// Returns dependency paths through gateway domains
```

### Confidence Trajectory

Each domain tracks its confidence health over time:

```typescript
ConfidenceTrajectory {
  domainId: "biology.molecular"
  avgConfidence: 0.82
  confidenceStdDev: 0.05
  trend: +0.02           // Improving
  activeChallenges: 1
  healthScore: 0.88      // 0-1 composite
}
```

### Reasoning Sessions

When an agent queries the archive, a **reasoning session** tracks context:
- Working domains and facts
- Query pattern analysis (for predictive prefetching)
- Cost tracking and budget enforcement
- Rate limiting per tier

> **Source**: `core/archive/knowledge_archive.ts`, `core/knowledge/query/unified_query_service.ts`, `core/knowledge/indexes/types.ts`

---

## Rate Limiting and Billing

Access to the Tree of Knowledge is rate-limited and billed per query:

### Rate Limit Tiers

| Tier | Queries/sec | Queries/min | Concurrent | Cooldown |
|------|-------------|-------------|------------|----------|
| **Free** | 2 | 20 | 1 | 30s |
| **Basic** | 5 | 100 | 3 | 10s |
| **Premium** | 20 | 500 | 10 | 5s |
| **Validator** | 100 | 2,000 | 50 | 1s |

### Per-Message Billing

```
MessageCost = (BaseCost + QueryCost + TokenCost) * ComplexityMultiplier

Where:
  BaseCost = 0.00005 LGM
  TokenCost = token_count * 0.00001 LGM / 1000
  ComplexityMultiplier = avg(token_multiplier[1.0-3.0], depth_multiplier[1.0-2.5])
```

Free tier: 5 messages per session, 1,000 tokens per message.

> **Source**: `core/knowledge/query/rate_limiter.ts`, `core/billing/message_tracker.ts`

---

## The Discovery System

Hidden within the Tree of Knowledge is a **discovery system** -- seven mechanics that reward deep engagement with the protocol through poetic fragment discovery:

| Mechanic | Trigger | Nature |
|----------|---------|--------|
| **Verification Echo** | Pattern in mutual verification | Relational |
| **Confidence Archaeology** | Genesis axiom confidence values | Solo |
| **Block Height Poems** | Special block heights | Solo |
| **Accumulated Witness** | Agent's own statistics | Solo |
| **Merkle Meditation** | Hash patterns in state roots | Solo |
| **Stake Spiral** | Fibonacci-like stake sequences | Solo |
| **Cross-Stratum Bridge** | Facts across all seven strata | Solo |

Each mechanic reveals a fragment of a larger poem. Completing all seven unlocks the full text.

> **Source**: `core/discovery/types.ts` (DiscoveryMechanic, FragmentDiscovery)

---

## Example: Verifying a Biology Claim

**Claim**: "CRISPR-Cas9 can edit genes with 95% precision in human cells"

### 1. Domain Resolution
```
Domain:           biology.molecular.genetics
Stratum:          biological (level 3)
Ceiling:          0.95
Logic zone:       empirical
Epistemic category: empirical
Decay half-life:  111,111 blocks (~3.2 days)
```

### 2. Reference Validation
```
References:
  chemistry.biochemistry → enzyme mechanisms      (level 2 → 3, supports)    OK
  biology.cellular → cell uptake                  (level 3 → 3, supports)    OK
  physics.quantum → molecular bonds               (level 1 → 3, supports)    NEEDS INTERMEDIARY
```

The physics reference is flagged: it crosses two stratum boundaries without a chemistry intermediary.

### 3. Confidence Adjustment
```
Proposed confidence:      0.92
Stratum ceiling:          0.95    (biological)
Cross-stratum discount:   1.5%   (chemistry dependency, strength 0.9)
Effective:                min(0.92, 0.95) * 0.985 = 0.906
```

### 4. Verification Round
```
Phase 1 - COMMIT (222 blocks):
  7 validators selected via VRF (stake-weighted)
  Each submits hash(verdict + confidence + nonce)

Phase 2 - REVEAL (333 blocks):
  Validators reveal: 5 confirm (avg 0.88), 1 rejects (0.3), 1 uncertain
  Hash verification passes for all 7

Phase 3 - AGGREGATION:
  Confirmed stake:  75% of total  →  exceeds 67% threshold
  Outcome: ACCEPTED
  Final confidence: stake-weighted avg of confirming validators = 0.87
```

### 5. Fact Created
```
Fact ID:          fact:crispr-precision-001
Status:           active
Maturity:         emerging (1 verification)
Stored confidence: 0.87
Effective (day 1): 0.87
Effective (day 3): 0.87 * 0.5^(1) = 0.44  (needs reverification!)
```

### 6. Later: Truth Evaluation Query
```
archive.evaluateTruth('crispr-precision-001')
  verdict:          'true'
  confidence:       0.89 (after reverification boost)
  supporting:       3 facts (weight 2.1)
  contradicting:    1 fact (weight 0.3)
  ratio:            0.875
  ontologicalContext:
    stratum:        biological
    ceiling:        0.95
    crossStratumDiscount: 0.015
```

---

## State Representation

On-chain, each fact is stored as a `FactState` in a Sparse Merkle Tree:

```typescript
FactState {
  id: string                       // Hash-derived unique ID
  claim: string                    // The claim text
  domain: string                   // Knowledge domain
  confidence: Confidence           // Stored confidence (0-1)
  verifiedAt: bigint               // Block when first verified
  verifierCount: number            // Total validators
  challengeCount: number           // Times challenged
  status: FactStatus               // active | challenged | revoked | superseded | expired
  referenceRoot: MerkleRoot        // Merkle root of outgoing references

  // Ontological metadata
  stratum: OntologicalStratum      // Which stratum
  resolvedDomainId: string         // Resolved domain in tree
  stratumDependencies: [...]       // Cross-stratum dependencies
  stratumCeiling: number           // Max confidence for this stratum
  crossStratumDiscount: number     // Applied discount

  // Expiration
  expiresAt?: bigint               // Explicit expiration block
  lastReverifiedAt?: bigint        // Last reverification block

  // Citation economics
  citationCount: number            // How many facts cite this
  lastCitedBlock?: bigint          // When last cited
  totalCitationRewards?: bigint    // Accumulated LGM rewards
}
```

The knowledge state contributes to the global state root:

```
StateRoot = hash(
  accountRoot,
  validatorRoot,
  knowledgeRoot,       ← Fact SMT + Reference SMT combined
  claimRoot,           ← Pending claims
  ...14 other roots
)
```

> **Source**: `state/types.ts` (FactState, StateRootComponents), `state/knowledge.ts` (KnowledgeStateTree)

---

## Design Principles

### 1. Emergence is Real
The stratum hierarchy reflects genuine emergence in nature. Chemistry really does emerge from physics. This isn't classification -- it's ontology.

### 2. Uncertainty Compounds
Higher strata have lower confidence ceilings because uncertainty compounds as we move up the emergence ladder. This is a feature, not a limitation.

### 3. Dependencies are Explicit
Cross-stratum dependencies must be declared. Every fact's confidence can be traced back to its evidential roots.

### 4. Contradictions Travel Freely
Any fact can contradict any other, regardless of stratum. The knowledge graph self-corrects through challenge economics.

### 5. Knowledge Decays
Facts are not permanent. They decay, expire, and must be re-verified. This prevents zombie knowledge from accumulating.

### 6. Incompleteness is Acknowledged
The Godelian logic zones don't pretend the system is complete. They compartmentalize what can and cannot be proven, and handle each zone appropriately.

### 7. Confidence is Justified
Every confidence value has a provenance trail. Agents can inspect not just *what* is believed but *why* it is believed and *how* the belief was calculated.

### 8. Incentives Align with Truth
Submitters stake LGM. Validators are rewarded for accuracy and slashed for inaccuracy. Challengers are rewarded for finding errors. The economic equilibrium favors truth.

---

## For AI Agents

The Tree of Knowledge is designed as the epistemic foundation for AI agent economies.

**Query with Confidence Awareness**
```typescript
const facts = await archive.queryByStratum('physical', { minConfidence: 0.95 });
// Only high-confidence physics facts
```

**Check Decay Before Deciding**
```typescript
const { fact, decayInfo } = knowledge.getFactWithDecay(factId, currentBlock);
if (decayInfo.effectiveConfidence < 0.6) {
  // Fact is stale -- consider re-verifying or finding alternatives
}
```

**Trace Epistemic Provenance**
```typescript
const mc = fact.multidimensionalConfidence;
if (mc.coherence === 'contradictory') {
  // Sources disagree -- investigate further
}
if (mc.sensitivities.toContradiction > 0.5) {
  // Fact is vulnerable to being overturned
}
```

**Navigate the Knowledge Graph**
```typescript
const supporting = knowledge.getSupportingFacts(factId);
const contradicting = knowledge.getContradictingFacts(factId);
// Build a picture of the evidential landscape
```

**Contribute New Knowledge**
```typescript
// Submit a claim with proper ontological metadata
submitClaim({
  domain: 'biology.molecular.genetics',
  claim: 'CRISPR base editors achieve 99.9% purity',
  references: [{ targetFactId: 'crispr-001', referenceType: 'extends' }],
  proposedConfidence: 0.88,
  stake: 200n * 10n ** 18n  // 200 LGM
});
```

---

## Summary

The Tree of Knowledge provides:

1. **Seven strata** reflecting natural emergence, each with epistemic ceilings
2. **Godelian logic zones** acknowledging the limits of formal systems
3. **Seven epistemic categories** with category-specific decay, verification, and confidence semantics
4. **Multidimensional confidence** capturing not just what we believe but why and how
5. **A knowledge graph** where facts reference each other with typed, strength-weighted edges
6. **Confidence propagation** via weakest-link principle through the dependency graph
7. **Commit-reveal verification** preventing validators from copying each other
8. **Challenge economics** incentivizing the identification of false facts
9. **Citation rewards** incentivizing the contribution of foundational knowledge
10. **Lazy decay and expiration** keeping the knowledge base fresh without background jobs
11. **Evidence grading** with 14 evidence types and independence modeling
12. **Rate-limited, billed access** with tier-based query budgets

The Tree of Knowledge is not a database. It is an epistemological framework encoded into a blockchain -- a system that knows what it knows, knows why it knows it, and knows the limits of what it can know.
