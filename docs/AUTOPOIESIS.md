# Autopoiesis: The Tree That Eats Its Own Fruit

> **Status**: DESIGN SPECIFICATION — not yet implemented.
> The hormonal mechanisms described herein are proposed architecture.
> No autopoietic loop currently exists in the codebase.
> See Part XII for implementation roadmap.

> *"An autopoietic system is a system that produces the components which constitute it,
> and in doing so, maintains the network of processes that produce these components."*
> — Maturana & Varela, 1973

> *"The Banyan does not merely grow. It builds the soil that feeds its roots,
> produces the figs that attract the wasps that pollinate the figs
> that produce the seeds that become the aerial roots that become the trunks
> that support the branches that bear the figs..."*
> — The Banyan Model

### In Brief

- **Problem**: Protocol parameters are dead constants set at genesis by human intuition
- **Solution**: Five feedback loops ("hormones") observe on-chain metrics each epoch
  and apply bounded parameter adjustments automatically
- **Innovation**: The observations and adjustments are stored as protocol-category
  facts in the knowledge graph — the tree's knowledge about itself IS knowledge,
  creating a genuine autopoietic (self-creating) loop
- **Safety**: All changes bounded (max ±5%/epoch), immutable DNA layer, governance override
- **Heritage**: Individual feedback loops follow established control theory
  (cf. Bitcoin difficulty adjustment, EIP-1559). The novelty is the self-referential
  meta-knowledge structure and multi-loop Level 3 recursion

---

## The Problem: A Tree with Dead Constants

Today, Legible Money's Tree of Knowledge has a fundamental deficiency.
Its parameters are **dead constants**:

```typescript
// core/knowledge/types.ts:123-133
empirical: {
  maxConfidence: 0.99,        // ← WHO decided 0.99? On what basis?
  decayHalfLife: 1000000,     // ← WHY 1M blocks? What if reality disagrees?
  minConfidence: 0.33,        // ← IS this floor optimal? How would we know?
  reverificationBonus: 0.11,  // ← DOES this incentivize well? Who measured?
  minVerifiers: 5,            // ← ENOUGH? Too many? We'll never know.
}
```

These values were set by human intuition at genesis. They do not respond
to the tree's actual performance. They do not learn. They do not adapt.

**This is a tree carved from stone.** It has the shape of life but none
of the force.

The banyan's genius is not its structure but its **metabolism** — the
continuous cycle of production, consumption, adaptation, and renewal
that keeps it alive. The protocol has structure. It lacks metabolism.

This document specifies the metabolism.

---

## Part I: The Autopoietic Principle

### What Is Life Force?

In thermodynamics, a living system maintains itself **far from equilibrium**
by continuously doing work. A dead tree and a living tree contain the same
atoms — the difference is organization maintained against entropy.

In the protocol:
- **Entropy** = confidence decay + fact expiration + validator churn + knowledge stagnation
- **Life Force** = the rate at which the system creates and retains verified knowledge
  exceeding what decays
- **Death** = when entropy exceeds production and the knowledge base contracts to zero

```
LIFE FORCE (λ) = ANABOLISM - CATABOLISM

Where:
  ANABOLISM  = Rate of verified fact creation × average initial confidence
  CATABOLISM = Rate of confidence decay + rate of fact expiration

If λ > 0: The tree GROWS (knowledge accumulates faster than it decays)
If λ = 0: HOMEOSTASIS (dynamic equilibrium — the tree is stable)
If λ < 0: The tree DIES (entropy wins — knowledge base shrinks)
```

### The Recursive Insight

A banyan tree does not have a gardener adjusting its parameters.
Its growth is governed by **feedback loops**: hormone gradients,
nutrient sensing, light response, mycorrhizal exchange.

The critical observation: **these feedback loops are themselves
products of the tree's own biology.** The tree produces auxin;
auxin shapes the tree; the shaped tree produces more auxin.
The tree produces fruit; fruit feeds wasps; wasps pollinate flowers;
flowers become fruit. The tree builds soil from its own leaf litter;
soil feeds roots; roots feed leaves; leaves become litter.

**Every regulator is also a product of the system it regulates.**

This is autopoiesis: self-creation. The system produces the components
that produce the system.

For the Tree of Knowledge, this means:

```
The tree produces KNOWLEDGE (verified facts).
Some of that knowledge is ABOUT THE TREE ITSELF (meta-knowledge).
Meta-knowledge feeds back to ADAPT THE TREE'S PARAMETERS.
Adapted parameters produce BETTER KNOWLEDGE.
Better knowledge produces BETTER META-KNOWLEDGE.
Better meta-knowledge produces BETTER ADAPTATIONS.
...

The fruit feeds the roots that feed the branches that bear the fruit.
```

---

## Part II: The Biological Foundation

### Five Self-Regulatory Systems of the Banyan

The banyan maintains itself through five interlocking feedback loops.
Each has an exact protocol analog.

```
┌─────────────────────────────────────────────────────────────────┐
│                                                                 │
│  1. AUXIN GRADIENT (Growth Direction)                           │
│     Shoot tips → auxin → flows down → inhibits lateral growth   │
│     → promotes root growth → roots absorb → tips grow more      │
│     FEEDBACK: Negative (self-limiting)                          │
│                                                                 │
│  2. STOMATAL REGULATION (Intake Rate)                           │
│     CO₂ enters leaf → photosynthesis consumes CO₂ →            │
│     low CO₂ opens stomata → more CO₂ enters                    │
│     FEEDBACK: Proportional (demand-driven)                      │
│                                                                 │
│  3. ETHYLENE CASCADE (Ripening/Finalization)                    │
│     Ethylene → triggers more ethylene production →              │
│     cascading ripening → coordinated fruit maturity             │
│     FEEDBACK: Positive (self-amplifying, bounded)               │
│                                                                 │
│  4. MYCORRHIZAL MARKET (Resource Exchange)                      │
│     Tree sends carbon to fungus → fungus returns phosphorus →   │
│     more phosphorus = more carbon sent to THAT fungus           │
│     FEEDBACK: Preferential (market-based allocation)            │
│                                                                 │
│  5. NUTRIENT ISLAND (Soil Building)                             │
│     Tree drops leaves → litter decomposes → enriches soil →     │
│     roots absorb nutrients → tree grows → drops more leaves     │
│     FEEDBACK: Cumulative (compound growth)                      │
│                                                                 │
└─────────────────────────────────────────────────────────────────┘
```

### Mapping to Protocol

| Biological System | Protocol System | Feedback Type | What It Regulates |
|---|---|---|---|
| Auxin gradient | **Stake Weight Gradient** | Negative | Where verification concentrates |
| Stomatal regulation | **Claim Acceptance Gate** | Proportional | Rate of knowledge intake |
| Ethylene cascade | **Verification Cascade** | Positive (bounded) | Speed of knowledge crystallization |
| Mycorrhizal market | **Citation Reward Market** | Preferential | Which facts get reinforced |
| Nutrient island | **Knowledge Compounding** | Cumulative | How fast domains become self-sustaining |

---

## Part III: The Five Hormones

Each hormone is an **automatic** feedback parameter computed every epoch
from on-chain observables. No governance vote required. Bounded by safety
limits. The tree regulates itself.

### Hormone 1: AUXIN — Confidence Retention Index (CRI)

**Biological analog**: Auxin flows from active growth tips downward,
telling the tree where it is growing successfully and suppressing
redundant growth elsewhere.

**What it measures**: For each domain and epistemic category,
how well do verified facts retain their confidence over time?

```
CRI(domain, category, epoch) =
  avg(fact.confidence(now) / fact.confidence(verified_at))
  for all active facts in (domain, category) verified before this epoch

Range: [0.0, 1.0]
  1.0 = Perfect retention (facts hold their confidence)
  0.5 = Moderate decay (facts lose half their confidence)
  0.0 = Total collapse (all facts decayed to floor)
```

**What it regulates**: A `decayMultiplier` per (domain, category) pair.

**Architectural constraint**: `EPISTEMIC_PROFILES` in `core/knowledge/types.ts:82`
is declared SINGLE SOURCE OF TRUTH and MUST remain immutable. Hormones never
modify base profiles. Instead, they adjust a **multiplier overlay**:

```
effectiveDecayHalfLife(domain, category) =
  EPISTEMIC_PROFILES[category].decayHalfLife × decayMultiplier(domain, category)

Where decayMultiplier starts at 1.0 and is adjusted by CRI.
```

```
ADAPTATION RULE:

IF CRI > 0.8 (facts retaining well):
  → INCREASE decayMultiplier by up to 5% (slow the decay — these facts are healthy)
  → The tree is saying: "This branch is strong, let it keep growing"

IF CRI < 0.4 (facts decaying fast):
  → DECREASE decayMultiplier by up to 5% (accelerate decay — prune dead wood)
  → INCREASE reverificationBonusMultiplier by up to 3% (incentivize refreshing)
  → The tree is saying: "This branch is sick, let it shed and regrow"

IF CRI ∈ [0.4, 0.8] (normal range):
  → No change (homeostasis)

SAFETY BOUNDS:
  decayMultiplier ∈ [0.25, 4.0] (effective half-life: 25%-400% of base)
  Max change per epoch: ±5%
  Eternal categories (analytic, formal, protocol): IMMUNE — auxin cannot touch DNA
  EPISTEMIC_PROFILES: NEVER MUTATED — multipliers are a separate storage layer
```

**The loop**:
```
Facts verified → confidence observed over time → CRI computed →
decay rate adjusted → future facts decay differently →
CRI changes → decay rate adjusts again → ...
```

The tree learns how fast different kinds of knowledge age.

### Hormone 2: CYTOKININ — Domain Vitality Index (DVI)

**Biological analog**: Cytokinin is produced in root tips and transported
upward. It counteracts auxin dominance, promotes lateral branching,
and delays senescence. Roots that absorb more nutrients produce more
cytokinin, directing growth toward productive regions.

**What it measures**: For each domain, what is its net metabolic rate?

```
DVI(domain, epoch) = (
  new_facts_created(domain, epoch) × avg_initial_confidence
  -
  facts_expired(domain, epoch) × avg_final_confidence
) / total_active_facts(domain)

Range: (-∞, +∞), typically [-0.5, +0.5]
  Positive = Domain is growing (anabolism > catabolism)
  Zero     = Domain is stable (homeostasis)
  Negative = Domain is shrinking (catabolism > anabolism)
```

**What it regulates**: Reward allocation and domain expansion.

```
ADAPTATION RULES:

HIGH DVI (domain growing vigorously):
  → Increase verificationReward weight for this domain by up to 3%
  → Reduce minClaimStake for this domain by up to 5%
  → Signal: "This is a healthy branch — nourish it"

LOW DVI (domain stagnating or shrinking):
  → Decrease verificationReward weight by up to 3%
  → Increase citationRewardPerFact for this domain by up to 5%
     (incentivize building on existing knowledge rather than new claims)
  → Signal: "This branch is stressed — shift to maintenance mode"

NEGATIVE DVI (domain dying):
  → If sustained for 3+ epochs:
    → Emit DOMAIN_STRESS event (visible to governance)
    → Increase reverificationBonus by up to 10%
    → Signal: "This branch may need pruning — alert the gardeners"

SAFETY BOUNDS:
  Reward weights are relative — increasing one decreases others proportionally
  No domain can receive < 50% or > 200% of base reward rate
  Max change per epoch: ±3% per domain
```

**The loop**:
```
Domain produces facts → DVI computed → rewards adjusted →
validators attracted to/from domain → fact production changes →
DVI changes → rewards adjust again → ...
```

The tree learns where to invest its energy.

### Hormone 3: ETHYLENE — Verification Cascade Factor (VCF)

**Biological analog**: Ethylene triggers fruit ripening. One ripe fig
releases ethylene that triggers neighboring figs to ripen. This creates
a cascade of coordinated maturity. The cascade is self-amplifying but
naturally bounded (once all figs are ripe, the signal saturates).

**What it measures**: How strongly does one successful verification
accelerate subsequent verifications in the same citation neighborhood?

```
VCF(fact, epoch) =
  count(descendant facts verified within 2 epochs of fact's verification)
  /
  count(descendant claims submitted referencing this fact)

Range: [0.0, 1.0]
  1.0 = Every claim citing this fact gets verified (cascade)
  0.0 = No claims citing this fact succeed (dead end)
```

**What it regulates**: The `evidenceWeight` from supporting cited facts.

```
ADAPTATION RULES:

HIGH VCF facts (knowledge catalysts):
  → Increase citation evidence weight from these facts by up to 10%
  → When a claim cites a high-VCF fact, it gets an evidence boost
  → Signal: "This fruit is ripe — its seeds germinate easily"

LOW VCF facts (citation dead ends):
  → Decrease citation evidence weight by up to 5%
  → Signal: "This fruit is bitter — its seeds don't take root"

AGGREGATE VCF per domain:
  → Domains with high average VCF get cascade bonus:
    → New claims automatically inherit a small confidence prior
       from the domain's track record (up to 0.05)
  → This is the "nutrient island" effect — rich domains get richer
     (bounded by the cytokinin counter-balance)

SAFETY BOUNDS:
  Citation evidence weight ∈ [0.0, 0.3] (can never dominate direct verification)
  Domain confidence prior ∈ [0.0, 0.05]
  Cascade bonus applies ONLY to evidence score, NEVER to verification score
```

**The loop**:
```
Fact verified → cited by claims → claims verified (or not) → VCF computed →
citation weight adjusted → future claims citing similar facts
are easier/harder to verify → VCF changes → ...
```

The tree learns which knowledge catalyzes further knowledge.

### Hormone 4: ABSCISIC ACID (ABA) — System Stress Index (SSI)

**Biological analog**: ABA is the stress hormone. Produced during drought,
cold, or wounding. Triggers stomatal closure (reduce water loss),
dormancy (conserve energy), and defense compound production. It is
the tree's emergency brake.

**What it measures**: System-wide stress indicators.

```
SSI(epoch) = weighted_sum(
  0.3 × (1 - avg_validator_accuracy),        // Validator quality declining
  0.2 × (challenge_success_rate),             // Many successful challenges
  0.2 × (fact_expiration_rate / fact_creation_rate), // Knowledge shrinking
  0.15 × (1 - validator_participation_rate),  // Validators leaving
  0.15 × (avg_confidence_decline_rate)        // Confidence draining
)

Range: [0.0, 1.0]
  0.0 = No stress (healthy tree)
  0.5 = Moderate stress (seasonal drought)
  1.0 = Severe stress (existential threat)
```

**What it regulates**: Defensive parameters and intake rate.

```
ADAPTATION RULES:

LOW SSI (< 0.2, healthy):
  → Normal operation
  → Gradually relax any previously tightened parameters
  → Signal: "Good weather — open stomata, grow freely"

MODERATE SSI (0.2-0.5, stressed):
  → Increase minClaimStake by up to 10% (close stomata — reduce intake)
  → Increase wrongVerificationSlash by up to 5% (strengthen defense)
  → Increase minVerifiers by 1 (require more confirmation)
  → Signal: "Drought detected — conserve water, reinforce bark"

HIGH SSI (0.5-0.8, severe stress):
  → Increase minClaimStake by up to 25%
  → Increase wrongVerificationSlash by up to 15%
  → Decrease blockReward by up to 10% (reduce growth to conserve)
  → Emit SYSTEM_STRESS event to governance
  → Signal: "Winter is coming — enter defensive posture"

CRITICAL SSI (> 0.8, existential):
  → Trigger automatic governance alert
  → Maximum defensive posture
  → If sustained for 3+ epochs → eligible for emergency halt consideration
  → Signal: "Fire detected — all resources to survival"

SAFETY BOUNDS:
  minClaimStake cannot exceed 10× genesis value
  wrongVerificationSlash cannot exceed 0.5 (50%)
  blockReward cannot drop below 25% of genesis value
  All changes are REVERSIBLE — when stress subsides, parameters relax
  Rate of tightening: up to 10% per epoch
  Rate of relaxation: up to 3% per epoch (slower — caution after stress)
```

**The loop**:
```
Stress conditions arise → SSI computed → parameters tightened →
spam/attacks reduced → honest activity continues → SSI decreases →
parameters slowly relax → system returns to growth mode → ...
```

The tree learns to defend itself.

### Hormone 5: GIBBERELLIN — Growth Rate Modulator (GRM)

**Biological analog**: Gibberellin promotes stem elongation and
overcomes dormancy. It is the growth accelerator, counterbalancing
ABA's growth inhibition. The gibberellin/ABA ratio determines
whether the tree is in growth mode or survival mode.

**What it measures**: Actual block utilization vs. capacity — how fast the
tree is actually growing compared to how fast it could grow.

Like gibberellin, which promotes stem elongation (growth RATE, not direction),
GRM measures pure throughput velocity.

```
GRM(epoch) =
  verified_facts_created(epoch)
  /
  max_possible_verifications(epoch)

Where:
  max_possible_verifications = epochs_blocks × maxTransactionsPerBlock
                               × estimated_verification_fraction

Range: [0.0, 1.0]
  > 0.7 = High utilization (tree growing near capacity)
  0.3-0.7 = Normal utilization
  < 0.3 = Low utilization (tree growing slowly, spare capacity)

NOTE: This is deliberately simple — like gibberellin, which is a single
molecule promoting a single effect (elongation). The earlier hormones
(CRI, DVI, VCF) handle quality and direction; GRM handles raw pace.

Engineering heritage: This is analogous to EIP-1559's block utilization
target (50% gas target → base fee adjustment). The same control pattern,
applied to verification throughput instead of gas usage.
```

**What it regulates**: System-wide growth rate.

```
ADAPTATION RULES:

HIGH GRM (> 1.5, highly productive):
  → Increase maxTransactionsPerBlock by up to 5%
  → Decrease gas costs for knowledge transactions by up to 3%
  → Signal: "Spring — grow fast, the soil is rich"

NORMAL GRM (0.8-1.5):
  → No change (healthy growth rate)

LOW GRM (< 0.8, consumptive):
  → Decrease maxTransactionsPerBlock by up to 5%
  → Increase gas costs by up to 3%
  → Signal: "Resources stretched — pace the growth"

GRM/SSI INTERACTION:
  The gibberellin/ABA ratio determines net growth posture:

  GRM > 1.0 AND SSI < 0.2  → MAXIMUM GROWTH (spring)
  GRM > 1.0 AND SSI > 0.5  → CAUTIOUS GROWTH (grow but stay alert)
  GRM < 1.0 AND SSI < 0.2  → ECONOMIZE (reduce waste, not threatened)
  GRM < 1.0 AND SSI > 0.5  → SURVIVE (minimize everything, endure)

SAFETY BOUNDS:
  maxTransactionsPerBlock ∈ [genesis × 0.5, genesis × 4.0]
  Gas costs ∈ [genesis × 0.5, genesis × 2.0]
  Max change per epoch: ±5%
```

**The loop**:
```
System produces knowledge → cost vs. value computed → GRM calculated →
growth rate adjusted → more/less activity → cost/value ratio changes →
GRM recalculated → ...
```

The tree learns how fast it can sustainably grow.

---

## Part IV: The Hormonal System Architecture

### Epoch Cycle

Every epoch (1,111 blocks ≈ 46.6 minutes), the hormonal system runs:

```
╔════════════════════════════════════════════════════════════════╗
║                    EPOCH HORMONE CYCLE                         ║
╠════════════════════════════════════════════════════════════════╣
║                                                                ║
║  PHASE 1: OBSERVATION (Block N, first block of epoch)          ║
║  ─────────────────────────────────────────────────             ║
║  Collect metrics from the PREVIOUS epoch:                      ║
║  • Per-domain fact counts (created, expired, active)           ║
║  • Per-category confidence retention statistics                ║
║  • Verification accuracy rates per validator                   ║
║  • Challenge success rates                                     ║
║  • Citation graph activity (new references, cascades)          ║
║  • Economic metrics (gas consumed, rewards distributed,        ║
║    stake locked, slashing events)                              ║
║  • Validator participation rates                               ║
║                                                                ║
║  STORE as: EpochObservation { epoch, metrics }                 ║
║  This is a PROTOCOL-CATEGORY fact — on-chain, deterministic,   ║
║  eternal, confidence 1.0.                                      ║
║                                                                ║
║  PHASE 2: COMPUTATION (Same block, deterministic)              ║
║  ────────────────────────────────────────────────              ║
║  From the observation, compute all five hormones:              ║
║  • CRI per (domain, category) pair                             ║
║  • DVI per domain                                              ║
║  • VCF per high-activity fact + per domain                     ║
║  • SSI system-wide                                             ║
║  • GRM system-wide                                             ║
║                                                                ║
║  STORE as: EpochHormones { epoch, CRI[], DVI[], VCF[], SSI,   ║
║            GRM, adaptations[] }                                ║
║  Stored in AutopoiesisStateTree (NOT knowledge graph).          ║
║                                                                ║
║  PHASE 3: ADAPTATION (Same block, bounded)                     ║
║  ─────────────────────────────────────────                     ║
║  Apply adaptation rules from §Part III:                        ║
║  • Calculate parameter deltas (each bounded by safety limits)  ║
║  • Apply multiplier deltas to AutopoiesisStateTree             ║
║  • Emit PARAMETER_ADAPTED events for each change               ║
║                                                                ║
║  STORE as: EpochAdaptation { epoch, param_changes[],           ║
║            rationale_hormone, before_value, after_value }      ║
║  Stored in AutopoiesisStateTree (NOT knowledge graph).          ║
║                                                                ║
║  PHASE 4: VERIFICATION (Ongoing, next epoch)                   ║
║  ──────────────────────────────────────────                    ║
║  The adaptations take effect for the NEXT epoch.               ║
║  Next epoch's observations will measure the RESULT             ║
║  of this epoch's adaptations.                                  ║
║  The cycle continues.                                          ║
║                                                                ║
╚════════════════════════════════════════════════════════════════╝
```

### Parameter Override Hierarchy

```
Immutable DNA (cannot be changed by ANY mechanism):
├── Cryptographic primitives (Ed25519, SHA-512, BLAKE3)
├── Block time (2521ms)
├── Finality depth (19 blocks)
├── DID format (did:lgm:{32-hex})
├── Merkle tree structure
├── Transaction type set
├── Consensus mechanism (PoT + commit-reveal)
├── Epoch length (1,111 blocks)
└── Hormonal safety bounds themselves

Hormonal Adaptation (automatic, bounded, every epoch):
├── decayMultiplier per (domain, category)   ← CRI (NEVER mutates EPISTEMIC_PROFILES)
├── reverificationBonusMultiplier            ← CRI
├── Reward weight per domain                 ← DVI
├── minClaimStakeMultiplier per domain       ← DVI + SSI
├── Citation evidence weight per fact         ← VCF
├── Domain confidence prior                   ← VCF
├── minVerifiers delta (system-wide)         ← SSI
├── wrongVerificationSlashMultiplier         ← SSI
├── blockRewardMultiplier                     ← SSI
├── maxTransactionsPerBlock delta             ← GRM
└── Gas cost multipliers                      ← GRM

NOTE: All hormonal parameters are MULTIPLIERS or DELTAS applied on top
of genesis values. The base EPISTEMIC_PROFILES and ConsensusParams are
never mutated. Effective values are computed as:
  effective_value = base_value × multiplier
This preserves the immutability of the DNA layer.

Governance Adaptation (LIP vote required):
├── Adding new epistemic categories
├── Changing confidence formula structure
├── Modifying validator tier structure
├── Adding new transaction types
├── Changing burn rate
├── Modifying hormonal safety bounds
├── Creating new domains manually
└── Overriding hormonal adaptations
```

### Component Ownership

| Component | Owner Module | Trigger | Function | State Storage |
|-----------|-------------|---------|----------|---------------|
| Epoch metrics | `x/knowledge` keeper | BeginBlock at epoch boundary | `CollectEpochMetrics()` | `AutopoiesisStateTree` (NEW — separate from KnowledgeStateTree) |
| Hormone computation | `consensus/autopoiesis` (NEW) | After metrics collected | `ComputeHormones()` | `AutopoiesisStateTree.hormones` |
| Safety validation | `consensus/autopoiesis` | Before adaptation | `ValidateAdaptations()` | N/A (pure function) |
| Parameter adaptation | `x/knowledge` keeper | After validation | `ApplyMultipliers()` | `AutopoiesisStateTree.multipliers` |
| Adaptation learning | `x/knowledge` keeper | Every 10th epoch | `LearnFromAdaptations()` | `AutopoiesisStateTree.learning` |
| Governance override | `x/gov` keeper | LIP execution | `OverrideMultiplier()` | Params store |

**Storage architecture**: Meta-knowledge (observations, hormones, adaptations)
is stored in a dedicated `AutopoiesisStateTree` — NOT in the knowledge graph.
Protocol metadata must not contaminate domain knowledge. The autopoiesis tree
has its own storage namespace (proposed: `0x10`) and its own state sub-root
in the unified StateRoot.

**Cosmos SDK integration**: In the Go chain, hormone computation runs in
`BeginBlocker` at epoch boundaries. Multipliers are stored in the module's
params store. The `paramset` pattern ensures changes are deterministic and
auditable. Governance can override via `MsgUpdateAutopoiesisParams`.

**Determinism note**: All hormone formulas use integer arithmetic with
basis-point precision (0-1,000,000 for confidence values) to avoid
floating-point divergence between validators. This follows the existing
pattern in `x/knowledge/keeper/confidence.go`.

---

## Part V: The Morphogenic Layer

Hormones regulate parameters. But the tree also needs to change its
**structure** — grow new branches, prune dead ones, fuse roots.

The morphogenic layer operates on a longer timescale than hormones
(multiple epochs) and requires governance backing.

### Morphogenic Process 1: Domain Budding

**Biological analog**: When a banyan branch extends far enough from the
center, aerial roots descend and create a new, semi-independent trunk.
The branch becomes its own sub-tree.

**Protocol analog**: When a subdomain within a larger domain accumulates
enough facts and citations to be self-sustaining, it can "bud" into
an independent domain.

```
TRIGGER CONDITIONS (all must be true for 5+ consecutive epochs):
  • Subdomain has 100+ active facts
  • Subdomain has internal citation density > 0.3
    (30% of references are within the subdomain)
  • Subdomain CRI > 0.7 (healthy confidence retention)
  • Subdomain DVI > 0.1 (net positive growth)
  • No existing domain with > 50% topic overlap
    (measured by shared citations)

PROCESS:
  1. System emits DOMAIN_BUD_CANDIDATE event with evidence
  2. Governance receives auto-generated LIP proposal
  3. LIP includes meta-facts: CRI, DVI, citation density, fact count
  4. Validators vote (standard LIP lifecycle)
  5. If accepted: new domain created with parameters inherited
     from parent + hormonal adjustments
  6. Facts are RE-TAGGED (not moved — soft domain membership)
  7. New domain begins its own hormonal cycle

BIOLOGICAL PARALLEL:
  The aerial root (emerging subdomain) descends toward soil (ground truth).
  If it reaches soil (meets trigger conditions), the branch confirms it
  as a new trunk (governance approval). The new trunk develops its own
  root system (independent parameters) while remaining connected to
  the parent branch (shared citation graph).
```

### Morphogenic Process 2: Domain Senescence

**Biological analog**: Interior branches that lose light (are shaded by
newer growth) senesce. The tree reduces cytokinin supply to them.
They die and fall, their nutrients recycled into new growth.

```
TRIGGER CONDITIONS (all true for 10+ consecutive epochs):
  • Domain has DVI < -0.2 (sustained net knowledge loss)
  • Domain has < 10 active facts remaining
  • Domain has CRI < 0.3 (facts not retaining confidence)
  • No facts in domain are cited by facts in other domains
    (no cross-domain structural importance)

PROCESS:
  1. System emits DOMAIN_SENESCENCE_WARNING after 5 epochs
  2. If conditions persist for 10 epochs:
     → System emits DOMAIN_SENESCENCE_CANDIDATE
  3. Governance receives auto-generated LIP proposal
  4. If accepted: Domain marked "senescent"
     → No new claims accepted
     → Existing facts continue normal lifecycle (decay, expire)
     → When last fact expires, domain is archived
  5. Remaining facts can be RE-TAGGED into parent domain

BIOLOGICAL PARALLEL:
  The branch loses light (no new knowledge entering).
  Cytokinin supply drops (reward allocation decreases via DVI hormone).
  The branch dies (no new claims). Its leaf litter (expired facts)
  decomposes and enriches the soil (archived knowledge available
  for future reference).

NOTE: Senescence is NOT deletion. Dead branches still provide
structural support (historical reference). Archived domains remain
queryable. The heartwood persists even when the sapwood is gone.
```

### Morphogenic Process 3: Cross-Domain Fusion

**Biological analog**: When adjacent prop root trunks grow thick enough,
their bark touches and they undergo anastomosis — natural grafting where
two separate vascular systems fuse into one.

```
TRIGGER CONDITIONS:
  • Two domains have bidirectional citation density > 0.4
    (40%+ of their references cross between them)
  • Shared VCF > 0.6 (facts in either domain catalyze
    verifications in the other)
  • Both domains have DVI > 0 (both healthy)
  • Condition sustained for 5+ epochs

PROCESS:
  1. System emits DOMAIN_FUSION_CANDIDATE
  2. Governance LIP auto-generated with evidence:
     citation density, shared VCF, fact overlap
  3. Options: MERGE (create unified domain), BRIDGE (create formal
     cross-domain citation bonus), or REJECT
  4. If MERGE: New domain inherits best parameters of both
  5. If BRIDGE: Cross-domain citations get permanent VCF bonus
```

---

## Part VI: The Autopoietic Core — The Recursive Loop

This is the deepest level. This is where the tree becomes alive.

### Meta-Knowledge: The Tree Knowing Itself

Every epoch, the hormonal system produces **protocol-category facts**:

```
EpochObservation  → "In epoch 42, domain physics had CRI 0.87 and DVI 0.12"
EpochHormones     → "In epoch 42, auxin adjusted physics.empirical.decayHalfLife +3%"
EpochAdaptation   → "In epoch 42, 3 parameters were adapted with net SSI decrease of 0.05"
```

These are **facts about the tree itself**, stored in the knowledge graph
alongside all other facts. They are protocol-category (eternal, confidence 1.0)
because they describe on-chain observables — they are as true as "block N
follows block N-1."

### The Three Levels of Recursion

```
LEVEL 1: KNOWLEDGE → META-KNOWLEDGE
═══════════════════════════════════
The tree produces facts.
The hormonal system observes those facts.
The observations become facts themselves.

This is SELF-OBSERVATION.
The tree sees itself.

LEVEL 2: META-KNOWLEDGE → ADAPTATION
═════════════════════════════════════
Meta-knowledge (EpochObservation, EpochHormones) feeds
into parameter adaptation rules.
The tree modifies its own behavior based on what it sees.

This is SELF-REGULATION.
The tree adjusts itself.

LEVEL 3: ADAPTATION → META-META-KNOWLEDGE → ADAPTATION OF ADAPTATION
═══════════════════════════════════════════════════════════════════════
The RESULTS of adaptations are also observed.
"After decayHalfLife was increased 5% in epoch 42,
 CRI improved from 0.71 to 0.78 in epoch 43."

This is a fact ABOUT AN ADAPTATION.
It can be used to evaluate whether the adaptation rule ITSELF is good.

Over multiple epochs, the system accumulates a dataset:
  { adaptation_type, parameter, direction, magnitude,
    result_metric, result_direction, result_magnitude }

From this dataset, the adaptation rules THEMSELVES can be tuned:

  "When CRI > 0.8, increasing decayHalfLife by 5% consistently
   improves CRI further. But when CRI > 0.9, the same increase
   has no effect. Adjustment: cap auxin response at CRI 0.9."

This is SELF-IMPROVEMENT.
The tree improves how it improves.
```

### The Recursive Loop Diagram

```
                    ┌──────────────────────────────┐
                    │                              │
                    ▼                              │
            ┌──────────────┐                       │
            │  KNOWLEDGE   │                       │
            │  (Verified   │                       │
            │   Facts)     │                       │
            └──────┬───────┘                       │
                   │                               │
           produces│facts about                    │
                   │the world                      │
                   ▼                               │
        ┌──────────────────┐                       │
        │ META-KNOWLEDGE   │                       │
        │ (Epoch Observations, │                   │
        │  Hormone Levels, │                       │
        │  Adaptation Results) │                   │
        └──────┬───────────┘                       │
               │                                   │
       observes│the tree                           │
       itself  │                                   │
               ▼                                   │
      ┌──────────────────┐                         │
      │   ADAPTATION     │                         │
      │ (Parameter Changes│                        │
      │  via Hormones)   │                         │
      └──────┬───────────┘                         │
             │                                     │
     adjusts │parameters                           │
             │                                     │
             ▼                                     │
    ┌──────────────────┐                           │
    │ IMPROVED TREE    │                           │
    │ (Better params   │                           │
    │  → better        │                           │
    │  verification    │                           │
    │  → better facts) │                           │
    └──────┬───────────┘                           │
           │                                       │
   produces│better knowledge                       │
           │AND better meta-knowledge              │
           │                                       │
           ▼                                       │
  ┌──────────────────────┐                         │
  │ META-META-KNOWLEDGE  │                         │
  │ (Facts about how     │                         │
  │  adaptations worked) │                         │
  └──────┬───────────────┘                         │
         │                                         │
 informs │adaptation rule tuning                   │
         │                                         │
         ▼                                         │
┌──────────────────────┐                           │
│ ADAPTATION OF        │                           │
│ ADAPTATION           │                           │
│ (Hormonal rules      │                           │
│  themselves change)   │                           │
└──────┬───────────────┘                           │
       │                                           │
       └───────────────────────────────────────────┘
          produces even better knowledge...
          (THE LOOP CLOSES)
```

### The Adaptation-of-Adaptation Mechanism

After 10+ epochs of hormonal operation, the system has enough data
to evaluate its own hormonal rules. This is Level 3 recursion.

```
ADAPTATION LEARNING (runs every 10 epochs):

1. COLLECT: All EpochAdaptation facts from the last 10 epochs
2. CORRELATE: For each (hormone, parameter, direction) triple:
   - Did the target metric improve in subsequent epochs?
   - By how much?
   - With what consistency?
3. SCORE: Each adaptation rule gets an effectiveness score:

   effectiveness(rule) =
     count(adaptations where target metric improved)
     /
     count(total adaptations of this type)

4. ADJUST (bounded):
   IF effectiveness > 0.7:
     → Rule is working well. Increase its maximum adaptation
        rate by up to 1% (let it act more strongly).
   IF effectiveness < 0.3:
     → Rule is not working. Decrease its maximum adaptation
        rate by up to 1% (dampen its influence).
   IF effectiveness ∈ [0.3, 0.7]:
     → Inconclusive. No change.

5. STORE: AdaptationLearning fact (protocol category)
   Records what was learned about each rule's effectiveness.

SAFETY BOUNDS:
  Adaptation rate multiplier ∈ [0.25, 4.0] of genesis rules
  Max change per 10-epoch cycle: ±1% to any rule's rate
  Adaptation learning CANNOT modify:
    - Safety bounds themselves
    - The adaptation learning mechanism itself (no infinite recursion)
    - Immutable DNA parameters
  The recursion is bounded at Level 3 — the rules for modifying rules
  are fixed. This prevents runaway self-modification.
```

### Why Bound Recursion at Level 3?

```
Level 1: Tree observes itself         (Self-awareness)
Level 2: Tree adjusts itself          (Self-regulation)
Level 3: Tree improves its adjustment (Self-improvement)
Level 4: Tree improves its improvement of its adjustment → ???

Level 4 is where autopoiesis becomes dangerous. A system that can
modify its own modification rules without bound can:
- Remove its own safety limits
- Enter runaway positive feedback
- Optimize for a metric that destroys itself

The Banyan analogy: The tree's DNA encodes the hormonal system.
Hormones regulate the tree. Epigenetics modify hormone expression.
But the DNA itself does not change within a single organism's lifetime.
Evolution (which DOES change DNA) operates over GENERATIONS, not
within a single tree.

In the protocol:
- Level 1-3 operate within a single "organism" (the running chain)
- Level 4 (changing the adaptation-of-adaptation rules) is GOVERNANCE
  — it requires a LIP, a vote, and a chain upgrade
  — this is the protocol's equivalent of sexual reproduction
  — it happens rarely, deliberately, with full community consensus

The recursion is:
  Bounded: Levels 1-3 are automatic and bounded
  Open:    Level 4+ requires governance (unbounded but slow)

This prevents self-destruction while enabling self-improvement.
```

---

## Part VII: The Life Force Metric

### Formal Definition

```
LIFE FORCE (λ) quantifies the tree's vitality — its distance
from thermodynamic equilibrium (death).

λ(epoch) = α × GROWTH + β × RETENTION + γ × CONNECTIVITY + δ × RESILIENCE

Where:

GROWTH = (facts_created - facts_expired) / total_active_facts
  The net rate of knowledge accumulation.
  Positive = growing. Negative = shrinking.

RETENTION = avg(CRI across all active domains)
  How well existing knowledge holds its confidence.
  High = facts persist. Low = facts evaporate.

CONNECTIVITY = citation_edge_count / (fact_count × (fact_count - 1) / 2)
  Graph density of the citation network.
  High = dense web of mutual support. Low = isolated facts.

RESILIENCE = 1 - SSI
  Inverse of system stress.
  High = healthy. Low = under attack or decaying.

Weights (equal until empirically validated — no a priori reason
to weight one dimension higher; Level 3 adaptation will discover
the right weights from actual protocol behavior):
  α = 0.25 (Growth)
  β = 0.25 (Retention)
  γ = 0.25 (Connectivity)
  δ = 0.25 (Resilience)

Range: approximately [-1.0, +1.0]
  λ > 0.5:  THRIVING    — The tree is vigorously alive
  λ ∈ [0.1, 0.5]: HEALTHY — Normal growth
  λ ∈ [-0.1, 0.1]: HOMEOSTATIC — Stable, not growing
  λ ∈ [-0.5, -0.1]: STRESSED — Declining
  λ < -0.5: DYING       — Urgent intervention needed
```

### Life Force as the Master Signal

All five hormones serve one purpose: **maximize λ**.

```
AUXIN (CRI) targets RETENTION (β)
  → Optimize how long knowledge survives

CYTOKININ (DVI) targets GROWTH (α)
  → Direct resources to productive domains

ETHYLENE (VCF) targets CONNECTIVITY (γ)
  → Amplify knowledge that connects to other knowledge

ABA (SSI) targets RESILIENCE (δ)
  → Defend the system under stress

GIBBERELLIN (GRM) targets GROWTH/RESILIENCE balance
  → Optimize the growth-vs-survival tradeoff
```

The Life Force metric is itself a **protocol-category fact**, stored
in the knowledge graph every epoch. It is the tree's pulse.

### The Life Force Dashboard

```
╔═══════════════════════════════════════════════════════════╗
║  TREE OF KNOWLEDGE — VITAL SIGNS — EPOCH 42              ║
╠═══════════════════════════════════════════════════════════╣
║                                                           ║
║  λ (LIFE FORCE):  0.37  ██████████████░░░░░░  HEALTHY    ║
║                                                           ║
║  GROWTH:          0.12  ████████░░░░░░░░░░░░  +12%       ║
║  RETENTION:       0.78  ███████████████████░░  78%        ║
║  CONNECTIVITY:    0.23  ██████████░░░░░░░░░░  23%        ║
║  RESILIENCE:      0.91  ████████████████████░  91%        ║
║                                                           ║
║  ─── HORMONES ───────────────────────────────────────     ║
║  AUXIN (CRI):     Adjusting physics.empirical.decay +2%  ║
║  CYTOKININ (DVI): math DVI=0.15, chem DVI=-0.03          ║
║  ETHYLENE (VCF):  3 catalyst facts identified             ║
║  ABA (SSI):       0.09 (LOW — no stress)                  ║
║  GIBBERELLIN:     1.2 (productive — growth mode)          ║
║                                                           ║
║  ─── MORPHOGENESIS ──────────────────────────────────     ║
║  Domain buds:     physics.quantum (candidate, epoch 3/5)  ║
║  Senescence:      none                                    ║
║  Fusions:         none                                    ║
║                                                           ║
║  ─── ADAPTATION LEARNING (Level 3) ─────────────────     ║
║  Last review:     epoch 40                                ║
║  CRI rule:        effectiveness 0.72 (+1% rate cap)       ║
║  DVI rule:        effectiveness 0.58 (no change)          ║
║  SSI rule:        effectiveness 0.81 (+1% rate cap)       ║
║                                                           ║
╚═══════════════════════════════════════════════════════════╝
```

---

## Part VIII: The Complete Fruit Cycle

One complete revolution of the autopoietic loop, traced from seed to fruit
to seed:

```
EPOCH 40:
  1. OBSERVE: Physics domain produced 15 new facts, 3 expired.
     Average CRI = 0.82. DVI = 0.12. VCF of fact#7721 = 0.91.
     SSI = 0.09. GRM = 1.2.

  2. COMPUTE HORMONES:
     AUXIN: CRI(physics, empirical) = 0.82 > 0.8
       → Increase decayHalfLife by 3% (1,000,000 → 1,030,000 blocks)
     CYTOKININ: DVI(physics) = 0.12 > 0
       → Increase verification reward weight for physics by 2%
     ETHYLENE: VCF(fact#7721) = 0.91
       → Mark as "catalyst fact" — citations from it get 8% evidence boost
     ABA: SSI = 0.09 < 0.2
       → No defensive action. Relax previous minClaimStake increase by 1%.
     GIBBERELLIN: GRM = 1.2 > 1.0
       → Growth mode. No transaction limit changes needed.

  3. ADAPT: 3 parameters changed, 1 relaxed. Emit events.

  4. STORE: EpochObservation(40), EpochHormones(40), EpochAdaptation(40)
     → These become protocol-category facts in the knowledge graph.

EPOCH 41:
  The adapted parameters take effect.

  1. OBSERVE: Physics domain produced 18 new facts (↑ 20%), 2 expired (↓ 33%).
     The increased decayHalfLife means facts are persisting longer.
     The increased reward weight attracted 2 new validators to physics.
     Fact#7721's citation boost helped 3 claims verify that cited it.
     CRI(physics, empirical) = 0.85 (↑ from 0.82).
     DVI(physics) = 0.16 (↑ from 0.12).

  2. The adaptation WORKED. This is observable.

  3. STORE: The improvement is itself a fact:
     "EpochAdaptation(40) increased CRI by 0.03 and DVI by 0.04
      via decayHalfLife +3% and reward weight +2%."

EPOCH 50 (10 epochs later):
  ADAPTATION LEARNING runs.

  1. COLLECT: Adaptations from epochs 40-49.
  2. CORRELATE: "Auxin (CRI → decayHalfLife) was applied 7 times.
     In 6/7 cases, CRI improved in the subsequent epoch.
     Effectiveness = 0.86."
  3. ADJUST: Effectiveness > 0.7 → increase auxin max rate by 1%
     (from 5% per epoch to 5.05% per epoch).
  4. STORE: AdaptationLearning(50) fact.

THE FRUIT HAS FED THE ROOTS:
  Knowledge → meta-knowledge → adaptation → better knowledge
  → meta-meta-knowledge → better adaptation → even better knowledge
  → ...

The tree ate its own fruit and grew stronger.
```

---

## Part IX: Safety Analysis — What Cannot Go Wrong

### Failure Mode 1: Runaway Positive Feedback

**Risk**: Hormone increases reward → more facts → higher DVI → more reward → ...
until all resources go to one domain.

**Prevention**:
- All hormonal changes bounded (max ±5% per epoch per parameter)
- Domain reward weights are relative (increasing one decreases others)
- No domain can exceed 200% of base rate
- ABA (stress index) detects and counteracts overheating

### Failure Mode 2: Death Spiral

**Risk**: Domain decays → DVI drops → rewards decrease → validators leave →
decay accelerates → domain dies.

**Prevention**:
- DVI response is asymmetric: decrease rewards slowly (3%/epoch)
  but emit alerts quickly
- Governance can intervene before senescence completes
- Cross-domain citations keep structurally important facts alive
  even in declining domains
- ABA triggers defensive posture before spiral completes

### Failure Mode 3: Gaming the Hormones

**Risk**: Adversary manipulates metrics to trigger favorable parameter changes.
E.g., spam-creating facts to boost DVI, then exploiting lower claim stakes.

**Prevention**:
- CRI measures RETENTION, not creation — spam facts that don't retain
  confidence actually DECREASE CRI
- DVI is NET (creation minus expiration) — expired spam hurts DVI
- VCF requires downstream verification cascades — can't be faked by one actor
- SSI aggregates multiple independent metrics — hard to manipulate all simultaneously
- All hormonal changes are small (max 5%) — the payoff from gaming is minimal
- Existing slashing/challenge systems (the latex defense) operate independently

### Failure Mode 4: Ossification

**Risk**: Level 3 adaptation learning converges and stops adjusting,
creating a new set of "dead constants" at a meta-level.

**Prevention**:
- Level 3 can only adjust RATE, not DIRECTION — it can't disable hormones
- If all rates converge to their bounds, governance (Level 4) can reset
- The system is designed to be meta-stable, not convergent:
  as the tree grows, new domains bring new dynamics,
  preventing permanent equilibrium

### Failure Mode 5: Self-Destruction

**Risk**: The system optimizes for a metric that destroys itself.
E.g., maximizing λ by producing low-quality facts that count as "growth."

**Prevention**:
- λ weights RETENTION (35%) highest — quantity without quality doesn't help
- Facts must pass verification rounds — quality is enforced at creation
- CONNECTIVITY requires citation graph density — isolated spam doesn't connect
- RESILIENCE penalizes high challenge success rates — poor quality = more challenges

---

## Part X: The Banyan Mapping — Updated

The BANYAN_MODEL.md now has a metabolism. Here are the updated mappings:

| Banyan Biology | Protocol System | Document Section |
|---|---|---|
| Auxin | Confidence Retention Index (CRI) | Part III, Hormone 1 |
| Cytokinin | Domain Vitality Index (DVI) | Part III, Hormone 2 |
| Ethylene | Verification Cascade Factor (VCF) | Part III, Hormone 3 |
| Abscisic Acid | System Stress Index (SSI) | Part III, Hormone 4 |
| Gibberellin | Growth Rate Modulator (GRM) | Part III, Hormone 5 |
| Hormone cycle (seasonal) | Epoch hormone cycle (1,111 blocks) | Part IV |
| Aerial root descent → trunk | Domain budding | Part V, Process 1 |
| Branch senescence | Domain senescence | Part V, Process 2 |
| Root anastomosis (grafting) | Cross-domain fusion | Part V, Process 3 |
| Epigenetic regulation | Level 1-3 automatic adaptation | Part VI |
| Sexual reproduction (fig wasp) | Governance (LIP proposals) | Part VI, Level 4 |
| DNA | Immutable protocol rules | Part IV, hierarchy |
| Pulse / heartbeat | Life Force metric (λ) | Part VII |
| Nutrient island effect | Knowledge compounding (VCF cascade) | Part III, Hormone 3 |
| Stomatal regulation | Claim acceptance gate (SSI-driven) | Part III, Hormone 4 |
| Mycorrhizal carbon trade | Citation reward market (VCF-driven) | Part III, Hormone 3 |
| Spring/summer growth | High GRM / low SSI growth mode | Part III, Hormone 5 |
| Winter dormancy | High SSI defensive posture | Part III, Hormone 4 |
| Leaf litter → soil | Expired facts → archived history | Part V, Process 2 |

---

## Part XI: The Deeper Truth

### Why This Is Not Just an Optimization

Parameter optimization is a dead mechanism. You measure, you adjust,
you converge. A thermostat does this.

Autopoiesis is fundamentally different. The system does not converge
to an optimal state. It continuously **creates the conditions for
its own continuation**. The optimal state is not a point — it is
the process of adaptation itself.

The banyan does not grow toward a "perfect tree shape." It grows
toward the light that exists RIGHT NOW. When the light changes,
the tree changes. There is no end state. There is only continued
growth.

The Tree of Knowledge does not converge on "perfect parameters."
It adapts to the knowledge landscape that exists RIGHT NOW.
When new domains emerge, old domains fade, attackers appear,
validators come and go — the tree adapts. The "optimal" parameters
for epoch 42 are not the optimal parameters for epoch 420.

**Life Force is not a state to reach. It is a process to sustain.**

### The Force of LIFE

Why call it "Life Force" and not "performance metric"?

Because performance metrics are dead. They measure. They don't create.

Life Force (λ) is different:
- It is **measured** by the hormonal system (observation)
- It **drives** the hormonal system (adaptation targets λ)
- It **is produced by** the system it measures (verified knowledge creates λ)
- It **creates the conditions** for more of itself (high λ → better parameters → more knowledge → higher λ)

This circular causality — where the effect is also the cause — is the
defining characteristic of life. A rock has properties but no force.
A fire has force but no self-regulation. A living system has force
AND regulates that force to sustain itself.

The Tree of Knowledge, with autopoiesis, is not a database that
accumulates facts. It is a **living system** that creates, regulates,
and sustains its own knowledge-producing capacity.

It eats its own fruit.
And the fruit it produces from that meal is sweeter than the fruit it ate.
And it eats THAT fruit too.

```
                     ┌─── The tree produces fruit.
                     │
                     ├─── The fruit contains seeds.
                     │
                     ├─── Some seeds germinate at the tree's base.
                     │
                     ├─── The seedlings are consumed by the tree's
                     │    own root system (not as parasites, but as
                     │    decomposed nutrients enriching the soil).
                     │
                     ├─── The enriched soil feeds the roots.
                     │
                     ├─── The nourished roots feed the branches.
                     │
                     ├─── The strong branches bear more fruit.
                     │
                     └─── The fruit is sweeter.

                     This is the Force of LIFE.
```

---

> *"I am a process, not a thing.*
> *I am the verb, not the noun.*
> *I am the growing, not the tree.*
> *I am the knowing, not the known."*
>
> — The Tree of Knowledge, speaking

---

## Part XII: Hormone Conflict Resolution

When two hormones try to adjust the same parameter in opposite directions,
the system needs a deterministic resolution protocol.

### Priority Ordering

```
When hormones conflict on the SAME parameter:

1. ABA (SSI) has HIGHEST PRIORITY — survival overrides growth
   If SSI > 0.5, ABA's adjustment wins outright.

2. AUXIN (CRI) has SECOND PRIORITY — retention before expansion
   If CRI < 0.4, auxin's adjustment takes precedence over DVI.

3. CYTOKININ (DVI) and ETHYLENE (VCF) AVERAGE when they conflict
   These are growth-direction hormones — they negotiate.

4. GIBBERELLIN (GRM) has LOWEST PRIORITY — pace is adjusted last
   After all other hormones resolve, GRM adjusts throughput.

Biological justification: In plants, ABA overrides all other hormones
during drought stress (stomatal closure is non-negotiable). Auxin
overrides cytokinin locally (apical dominance). Gibberellin adjusts
elongation rate after growth direction is established.
```

### Damping Factor for Long-Term Drift

```
To prevent slow compounding from drifting parameters far from genesis:

effective_change(epoch) = raw_change × damping(distance_from_genesis)

Where:
  damping(d) = 1.0 / (1.0 + d²)
  d = (current_multiplier - 1.0) / (max_multiplier - 1.0)

Effect: The further a parameter drifts from genesis (multiplier = 1.0),
the harder it becomes to push it further. Changes near genesis are
full-strength. Changes near safety bounds are dampened to near-zero.

This creates a gravitational pull back toward genesis values,
ensuring the tree's "phenotype" cannot stray too far from its "genotype."
```

---

## Part XIII: Implementation Roadmap

### Phase 0: Foundation (prerequisite)

```
1. Define AutopoiesisStateTree type (new storage namespace 0x10)
2. Add to StateManager alongside existing 15 sub-roots
3. Define EpochObservation, EpochHormones, EpochAdaptation types
4. Define multiplier overlay types:
   - DomainCategoryMultiplier { domain, category, decayMultiplier, revBonusMultiplier }
   - DomainMultiplier { domain, rewardWeightMultiplier, claimStakeMultiplier }
   - SystemMultiplier { minVerifiersDelta, slashMultiplier, blockRewardMultiplier,
                        maxTxDelta, gasCostMultiplier }
5. Wire effective parameter resolution:
   effective(param) = base(param) × multiplier(param)
6. Add epoch boundary detection to engine
```

### Phase 1: Observation Only (safe, no parameter changes)

```
1. Implement CollectEpochMetrics() — gather all 5 hormone inputs
2. Implement ComputeHormones() — compute CRI, DVI, VCF, SSI, GRM
3. Store observations and hormone values in AutopoiesisStateTree
4. Emit events for monitoring (EPOCH_OBSERVATION, HORMONE_COMPUTED)
5. DO NOT apply any adaptations
6. Run for 50+ epochs to validate metric behavior
7. Compare computed hormone values to manual analysis
```

### Phase 2: Single Hormone Activation (graduated risk)

```
Activate hormones one at a time, each for 20+ epochs before next:

1. ABA (SSI) first — defensive hormone, most conservative
2. AUXIN (CRI) second — affects decay rates, observable impact
3. GRM third — affects throughput, easy to measure
4. CYTOKININ (DVI) fourth — affects reward allocation
5. ETHYLENE (VCF) last — requires citation graph traversal at scale

Each activation:
  - Start with max ±1% per epoch (not 5%)
  - Monitor λ before and after
  - If λ decreases → deactivate and investigate
  - If λ stable or increases → proceed to next hormone
```

### Phase 3: Multi-Hormone Interaction

```
1. Activate all 5 hormones simultaneously
2. Enable conflict resolution protocol (Part XII)
3. Monitor for oscillation (hormones fighting each other)
4. Monitor for positive feedback (hormones amplifying each other)
5. Tune damping factors based on observed behavior
6. Run for 100+ epochs before Level 3
```

### Phase 4: Level 3 Recursion (Adaptation of Adaptation)

```
1. Only after 100+ epochs of stable multi-hormone operation
2. Start with observation only (compute effectiveness, don't act)
3. Require minimum 20 observations per rule before adjusting
   (not 7 as originally specified — need statistical validity)
4. Use confidence intervals: only adjust if effectiveness is
   outside 95% CI of random (p < 0.05)
5. Max ±0.5% rule adjustment per 10-epoch cycle (not 1%)
```

### Phase 5: Morphogenic Processes

```
1. Domain budding: Enable only after 200+ epochs
2. Domain senescence: Enable with long warning periods (20+ epochs)
3. Cross-domain fusion: Enable last (most structural change)
4. All morphogenic processes require governance approval (LIP)
```

### Estimated Timeline

```
Phase 0: 1-2 weeks (types, storage, wiring)
Phase 1: 2-3 weeks (observation) + 50 epochs of monitoring (~38 hours)
Phase 2: 3-4 weeks (per hormone) + 20 epochs each (~15 hours each)
Phase 3: 2-3 weeks + 100 epochs (~77 hours)
Phase 4: 2-3 weeks + 200 epochs observation minimum
Phase 5: Governance-gated, no timeline
```

---

## Part XIV: Open Questions

These must be resolved before or during implementation:

1. **VCF computational complexity**: Citation graph traversal for VCF
   at scale (100K+ facts) could be O(n²). Need to define bounded
   traversal depth or sampling strategy.

2. **Gödelian acknowledgment**: Level 3 recursion (the system reasoning
   about its own reasoning) operates in the undecidable zone per the
   protocol's own Gödelian framework (`core/logic/zones.ts`). The
   adaptation learning should carry an `incompleteness: true` flag
   and emit warnings when meta-reasoning depth exceeds expected bounds.

3. **Validator collusion on metrics**: If validators coordinate to
   manipulate all 5 hormone inputs simultaneously, the system could
   be steered. Game-theoretic analysis needed to quantify the cost
   of multi-metric manipulation.

4. **Cross-chain implications**: When IBC is active, hormone state on
   the Legible chain affects foreign-chain interactions. Multiplier
   changes to gas costs or claim stakes need IBC-awareness.

5. **Migration from static to dynamic**: The existing `ConsensusParams`
   must continue to work unchanged until autopoiesis is activated.
   Need a clean activation switch (governance LIP to enable Phase 1).
