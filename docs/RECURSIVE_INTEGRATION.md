# Recursive Integration: The Autopoietic Loop at Every Scale

> **Status**: DESIGN SPECIFICATION — integration analysis and architectural roadmap.
> Maps the autopoietic (self-creating) loop from AUTOPOIESIS.md onto every level
> of the Legible Money protocol, from individual facts to cross-chain IBC.

> *"A banyan tree is not fractal by accident. The same branching pattern that shapes
> its crown also shapes its roots, its figs, its vasculature — because the same
> hormones operate at every scale. The pattern is not repeated. It IS the organism."*

---

## The Fractal Insight

The autopoietic loop in AUTOPOIESIS.md specifies a single recursive cycle:

```
OBSERVE → COMPUTE → ADAPT → VERIFY → (loop)
```

This cycle was designed at the **system level** (epoch-wide hormones adjusting global
parameters). But the banyan's genius is that this same cycle operates **at every
scale simultaneously**: each cell, each leaf, each branch, each root system, and
the whole organism all run their own observe→adapt loops, nested and interacting.

This document maps that fractal pattern across **seven levels** of the protocol,
identifies concrete integration points at each level, and specifies how the levels
compose into a single living system.

---

## The Seven Levels

```
LEVEL 7: EVOLUTION     │ Cross-chain / IBC / inter-protocol learning
LEVEL 6: ECOSYSTEM     │ Network topology, partnerships, cross-chain
LEVEL 5: ORGANISM      │ System-wide hormones (AUTOPOIESIS.md)
LEVEL 4: ORGAN SYSTEM  │ Module clusters (consensus + knowledge + staking)
LEVEL 3: ORGAN         │ Individual modules (validator tiers, governance, tokens)
LEVEL 2: TISSUE        │ Sub-module structures (mempool, state trees, rounds)
LEVEL 1: CELL          │ Individual facts, claims, verification events
```

```
Biological Banyan              Protocol                    Loop Cycle Time
═══════════════════             ═══════════                 ═══════════════
Cell metabolism           →     Fact-level feedback         ~1 block (2.5s)
Tissue growth             →     Round/pool dynamics         ~11 blocks (28s)
Organ function            →     Module parameter adj.       ~111 blocks (4.6 min)
Organ system coord.       →     Cross-module cascades       ~1 epoch (1,111 blocks ≈ 47 min)
Organismal hormones       →     5 system hormones           ~1 epoch (1,111 blocks)
Ecosystem interactions    →     Network/partnership adj.    ~10 epochs (≈ 7.8 hours)
Evolutionary adaptation   →     Cross-chain learning        ~100 epochs (≈ 3.2 days)
```

---

## Level 1: The Cell — Individual Facts

### Biological Analog
A single cell in the banyan leaf performs photosynthesis. It senses light intensity,
adjusts chloroplast orientation, opens/closes channels, and communicates stress via
calcium waves to neighboring cells. The cell doesn't wait for hormonal commands —
it self-regulates in real time.

### Protocol Analog
Each fact in the knowledge graph already has its own micro-lifecycle, but it is
currently passive — it decays mechanically according to fixed constants. The
autopoietic loop makes each fact a **data source** for the hormonal system —
facts remain passive data structures, but the epoch-level observer reads their
state and aggregates signals upward.

### Integration Points

**1.1: Fact-Level Confidence Feedback**
```
Location: state/knowledge.ts (KnowledgeStateTree)
          consensus/pot/confidence.ts (calculateMasterConfidence)

Current: Confidence decays by log1p(age) × decayRate (fixed per category).
         Fact is passive — it decays regardless of context.

Integration: The epoch-level observer (consensus/autopoiesis/observer.ts)
scans facts and computes per-fact signals, then aggregates into CRI:

  // Computed by observer at epoch boundary, NOT by facts themselves
  fact_signal(f) = {
    retention: f.confidence(now) / f.confidence(verified_at),
    citation_count: references_to(f).length,
    reverification_count: f.reverifications,
    challenge_survival_count: f.challenges_survived
  }

  CRI(domain, category) = weighted_avg(
    fact_signal(f) for f in domain×category,
    weights = f.citation_count  // well-cited facts weight more
  )

  RESPONSIBILITY: Facts are passive data. The observer is the active scanner.
  Facts store confidence/citations as fields. The observer reads them,
  computes signals, and aggregates into domain-level CRI. No fact
  "contributes" anything — the observer extracts what it needs.

Cycle: Per-epoch (observer scans at epoch boundary, not per-block)
```

**1.2: Citation Graph as Neural Network**
```
Location: state/knowledge.ts (references/citations)
          consensus/pot/confidence.ts (BFS confidence propagation)

Current: Citations propagate confidence via BFS with depth-based discount.
         The propagation is one-directional: new fact → cited fact gets bonus.

Integration: With VCF hormone (ETHYLENE), the citation graph becomes
bidirectional feedback:

  At the FACT level (Level 1), each verification event records:
    → B receives citation bonus (existing behavior)
    → A's verification cost is discounted by B's confidence (existing)
    → Cascade depth is RECORDED as a metadata field on the verification event

  At the ORGANISM level (Level 5), the epoch observer AGGREGATES:
    → VCF(domain) = avg(cascade_depth) for all verifications in domain this epoch
    → If VCF > threshold: increase evidence weight multiplier for this domain
    → Citation reward multiplier increases, attracting citation-heavy claims

  The Level 1 → Level 5 flow:
    Facts record cascade depth → Observer reads depths at epoch boundary →
    VCF computed → Multiplier stored in AutopoiesisStateTree →
    Next epoch: evidence weight and citation rewards read multiplier

  This is the mycorrhizal market: facts that produce useful citations
  (like fungi that deliver more phosphorus) receive more "carbon" (rewards).

Cycle: Per-verification event (metadata recording at Level 1)
       Per-epoch (VCF aggregation and multiplier update at Level 5)
```

**1.3: Fact Maturity Progression**
```
Location: core/knowledge/types.ts (FactState: provisional → accepted → challenged → ...)

Current: Facts transition through states based on external events
         (verification rounds, challenges). No internal aging logic beyond decay.

Integration: The GRM hormone (GIBBERELLIN) could distinguish facts by
maturity stage:

  Seedling fact (< 1 epoch old, < 3 citations):
    → High growth potential, low structural importance
    → Sensitive to GRM changes (throttled/accelerated with throughput)

  Mature fact (5+ epochs, 10+ citations, 2+ reverifications):
    → Low growth, high structural importance
    → Partially immune to GRM throttling (structural wood doesn't bend)

  Ancient fact (50+ epochs, deeply embedded in citation graph):
    → No growth, maximum structural importance (heartwood)
    → Fully immune to hormonal parameter changes
    → Only governance can alter its status

This mirrors banyan biology: sapwood responds to hormones,
heartwood does not. The tree's skeleton is preserved even as
the living tissue adapts around it.

Implementation note: Maturity could be a computed property:
  maturity(f) = min(1.0, (age_epochs/50 + citations/20 + reverifications/5) / 3)
  Applied as a damping factor on hormonal sensitivity:
  effective_change(f) = adaptation_delta × (1.0 - maturity(f))

Cycle: Per-epoch (maturity advances with each epoch boundary)
```

---

## Level 2: The Tissue — Sub-Module Structures

### Biological Analog
Tissue is organized cells working together: xylem vessels transport water upward,
phloem sieve tubes transport sugars to where they're needed, meristematic tissue
divides to produce new cells. Each tissue type has its own regulatory logic.

### Protocol Analog
Sub-module structures (mempool, verification rounds, state tree operations)
are the tissue of the protocol — organized collections of individual events
that collectively determine system health.

### Integration Points

**2.1: Mempool as Stomatal System**
```
Location: consensus/mempool/pool.ts
          core/billing/gas.ts (TRANSACTION_GAS_COSTS)

Current: Mempool is a fee-ordered priority queue with fixed capacity.
         Gas costs are static constants. No feedback between load and pricing.

Integration: The GRM hormone's gas cost multiplier (AUTOPOIESIS.md Part III,
Hormone 5) adjusts effective gas costs. The multiplier is computed in
consensus/autopoiesis/hormones.ts and stored in AutopoiesisStateTree.
It is READ by the ante handler (app/ante.go in Go, state/transitions.ts
in TS) at transaction validation time — NOT by the mempool itself.

  // Ante handler reads multiplier from autopoiesis state:
  effective_gas_cost(tx_type) =
    TRANSACTION_GAS_COSTS[tx_type] × gasMultiplier(GRM)

  // core/billing/gas.ts remains IMMUTABLE (like EPISTEMIC_PROFILES)
  // The multiplier overlay follows the same pattern

  When GRM > 0.7 (high utilization):
    → gasMultiplier increases → ante rejects underpaying txs → congestion eases
    → Like stomata closing when CO₂ is sufficient

  When GRM < 0.3 (low utilization):
    → gasMultiplier decreases → ante accepts lower fees → more txs enter
    → Like stomata opening when CO₂ is scarce

  OWNERSHIP: Gas constants → core/billing/gas.ts (immutable)
             Gas multiplier → consensus/autopoiesis (computed per epoch)
             Gas enforcement → ante handler (reads multiplier at tx time)
             Mempool → fee-ordered queue (reads effective cost, no computation)

  The mempool REPORTS its own health as a metric for the epoch observer:
    mempool_pressure = current_size / max_capacity
  This is a passive field read by consensus/autopoiesis/observer.ts
  at epoch boundaries — the mempool doesn't push this signal.

Cycle: ~1 block (mempool rebalances every block, GRM adjusts per epoch)
```

**2.2: Verification Rounds as Flower Pollination**
```
Location: consensus/pot/verification_round.ts
          consensus/pot/validator_selection.ts (VRF selection)

Current: Verification rounds have fixed phase durations:
         commitPhaseBlocks: 4, revealPhaseBlocks: 4 (from ConsensusParams)
         Validator count per round: [minValidators, maxValidators] range

Integration: The VCF hormone (ETHYLENE) and SSI hormone (ABA) should
influence round parameters:

  Under HIGH VCF (verification cascades frequent, knowledge crystallizing fast):
    → Reduce commitPhaseBlocks by 1 (faster rounds = faster crystallization)
    → Increase maxValidators by up to 2 (more wasps per fig = better pollination)
    → This mirrors ethylene's cascading fruit ripening — success breeds speed

  Under HIGH SSI (stress — disputes, slashing, attacks):
    → Increase commitPhaseBlocks by 1-2 (slow down, be careful)
    → Increase minValidators by 1 (require more consensus)
    → Decrease maxValidators (concentrate validator attention)
    → This mirrors ABA closing stomata and slowing growth during drought

  The round itself generates feedback:
    round_quality = {
      participation_rate: revealed / selected,
      consensus_strength: max_agreement_fraction,
      time_to_completion: actual_blocks / expected_blocks
    }
  This feeds into VCF and SSI for the next epoch.

Cycle: Per-round (~11 blocks) for events, per-epoch for parameter changes

Safety: commitPhaseBlocks ∈ [2, 8], revealPhaseBlocks ∈ [2, 8]
        minValidators ∈ [3, maxValidators], maxValidators ∈ [minValidators, 33]
```

**2.3: State Trees as Root Architecture**
```
Location: state/state_manager.ts (unified state management)
          state/accounts.ts, state/validators.ts, state/knowledge.ts

Current: Three SMTs operate independently. No cross-tree feedback.
         State root is a simple combination of sub-roots.

Integration: The autopoiesis system needs a NEW state tree:
  AutopoiesisStateTree (namespace 0x10) — as specified in AUTOPOIESIS.md Part IV

  The epoch-level observer (consensus/autopoiesis/observer.ts) can
  QUERY state trees for health metrics. State trees remain pure storage —
  they do not compute or report their own health:

    // Computed by observer, NOT by state trees themselves
    tree_health(tree) = {
      depth: observer.measureAvgProofPath(tree),
      density: observer.countEntries(tree) / total_capacity,
      churn: observer.countMutations(tree, epoch) / observer.countEntries(tree),
      proof_size: observer.measureAvgProofSize(tree)
    }

  OWNERSHIP: State trees (accounts.ts, validators.ts, knowledge.ts)
  are dumb storage with get/set/prove operations. All analysis and
  health computation belongs in the observer module.

  Tree health feeds into GRM as a storage efficiency signal:
  if proof sizes are growing, GRM should slow fact creation to let
  the tree consolidate. If density is high, the tree is healthy and
  can absorb more facts.

Cycle: Per-epoch aggregation
```

---

## Level 3: The Organ — Individual Modules

### Biological Analog
An organ is a functional unit: a leaf photosynthesizes, a root absorbs,
a fig feeds wasps, a trunk supports branches. Each organ has its own
internal regulatory systems while responding to whole-organism hormones.

### Protocol Analog
Individual modules (validator tiers, governance, tokens, evidence audit,
emergency) each have parameters that should respond to the autopoietic loop.

### Integration Points

**3.1: Validator Tiers as Growth Rings**
```
Location: consensus/pot/validator_tiers.ts (computeValidatorTier, TIER_THRESHOLDS)
          x/staking/keeper/keeper.go (Go chain equivalent)

Current: Tier thresholds are fixed constants:
  Tier 0 (Apprentice): 0 stake, protocol/analytic/formal only
  Tier 1 (Verified):   22 verifications at 77% accuracy
  Tier 2 (Bonded):     1,111 LGM staked
  Tier 3 (Guardian):   11,111 LGM + 1,111 verifications + 77% accuracy

Integration: SSI and GRM hormones should modulate tier thresholds:

  Under HIGH SSI (system stress):
    → Increase accuracy requirement for Tier 1 promotion: 77% → up to 85%
    → Increase verification count for Tier 1: 22 → up to 33
    → Increase stake requirement for Tier 2: 1,111 → up to 1,667 LGM
    → Signal: "Raise the bar during stress — only the strongest wood survives"

  Under HIGH GRM (rapid growth, system healthy):
    → Decrease accuracy requirement: 77% → down to 70%
    → Decrease verification count: 22 → down to 15
    → Signal: "Lower the bar during growth — let more saplings sprout"

  The tier system reports back:
    tier_distribution = {
      apprentice_count, verified_count, bonded_count, guardian_count,
      promotion_rate, demotion_rate, average_tier
    }
  This feeds into SSI (high demotion rate = stress signal)
  and GRM (high promotion rate = growth signal).

Cycle: Per-epoch
Safety: Thresholds bounded to ±50% of genesis values
        Promotions/demotions always based on ACTUAL performance,
        never retroactive (an existing Tier 2 doesn't lose tier
        because the threshold rose — grandfathering)
```

**3.2: Governance as Seasonal Cycles**
```
Location: x/gov/keeper/msg_server.go (LIP lifecycle)
          x/gov/types/types.go (LIPParams)

Current: Governance parameters are fixed:
  Voting period, quorum requirements, stake thresholds per category,
  timelock durations — all set at genesis.

Integration: DVI and SSI hormones should influence governance tempo:

  Under HIGH SSI (system stress):
    → Shorten voting period for emergency-category LIPs
    → Lower quorum for halt/revert proposals
    → Signal: "The forest is on fire — decide fast"

  Under SUSTAINED HIGH DVI across all domains (flourishing):
    → Extend voting period (no rush — contemplate carefully)
    → Raise quorum for parameter changes (high bar for changing what works)
    → Signal: "The forest is healthy — don't fix what isn't broken"

  Under SUSTAINED LOW DVI (stagnation everywhere):
    → Lower stake threshold for process/interface LIPs
    → Enable "emergency research LIP" fast-track
    → Signal: "The forest is dormant — make it easier to propose change"

  Governance reports back:
    governance_health = {
      active_lips: count,
      participation_rate: voters / eligible_validators,
      acceptance_rate: accepted / decided,
      average_voting_turnout: per_lip_average
    }
  Low participation → SSI signal (apathy = vulnerability)
  High proposal rate → GRM signal (lots of energy for change)

Cycle: Per-epoch parameter adjustment, per-LIP event reporting
Safety: Quorum never below 50%, voting period never below 3 epochs
```

**3.3: Token Expiration as Leaf Senescence**
```
Location: tokens/types.ts (ExpirableBalance, ChildAccount)
          tokens/expiration_tracker.ts (Dynamic Array)

Current: Token expiration periods are fixed. Tokens created in period P
         expire after a fixed number of periods regardless of system state.

Integration: CRI and DVI hormones should influence token longevity:

  Under HIGH system-wide CRI (knowledge retaining well):
    → Extend token expiration by up to 10% per epoch
    → Signal: "The tree is healthy — nutrients last longer in healthy tissue"

  Under LOW CRI (knowledge decaying fast):
    → Shorten token expiration by up to 5% per epoch
    → Signal: "Decay is accelerating — resources should recirculate faster"
    → This creates pressure for validators to actively verify
      (earn fresh tokens before old ones expire)

  Under NEGATIVE DVI (domains shrinking):
    → Increase retrieval rate for RAI (Revenue Acceleration Index)
    → Signal: "Leaves are falling — recapture nutrients faster"

  Token system reports back:
    token_health = {
      total_supply: active tokens,
      expiration_rate: tokens expiring / total per epoch,
      velocity: transfers per epoch / total supply,
      staking_ratio: staked / total
    }
  High expiration rate → CRI adjustment signal
  Low velocity → GRM signal (economy sluggish)

Cycle: Per-epoch
Safety: Expiration period bounded to [50%, 200%] of genesis value
```

**3.4: Evidence Audit as Immune System**
```
Location: state/evidence_audit.ts (EvidenceAuditTree)
          core/knowledge/evidence_audit/ (scoring, constants)

Current: Evidence scoring uses fixed weights and thresholds.
         Audit scores determine claim credibility independently of system state.

Integration: SSI hormone should modulate evidence strictness:

  Under HIGH SSI (system under attack):
    → Increase evidence weight requirements by up to 15%
    → Require additional evidence types (e.g., replication + peer review)
    → Increase minimum evidence count for high-stake claims
    → Signal: "Immune system on high alert — demand more proof"

  Under LOW SSI (system peaceful):
    → Relax evidence requirements to baseline (never below genesis)
    → Signal: "Immune system at rest — normal vigilance sufficient"

  Evidence system reports back:
    evidence_health = {
      average_evidence_score: across accepted claims,
      evidence_challenge_rate: challenged_evidence / total_evidence,
      evidence_type_distribution: { empirical, formal, ... },
      replication_rate: replicated_facts / empirical_facts
    }
  Low replication rate → SSI signal (empirical claims unchecked)
  High challenge success rate → SSI signal (bad evidence getting through)

Cycle: Per-epoch
Safety: Evidence weights bounded to [75%, 150%] of genesis values
```

**3.5: Emergency System as ABA Cascade**
```
Location: consensus/emergency/ (ceremony.ts, manager.ts)
          x/emergency/keeper/ (Go chain)

Current: Emergency thresholds are fixed:
  75% halt quorum, 80% revert/resume quorum, 3 proposals/epoch max

Integration: SSI hormone IS the emergency system's primary input:

  SSI already monitors stress indicators. The integration is:

  SSI < 0.3 (normal):
    → Emergency system dormant
    → Standard guardian-only access

  SSI ∈ [0.3, 0.6] (elevated):
    → Lower halt proposal threshold (allow Tier 2 validators to propose)
    → Increase cooldown period (more deliberation when not critical)
    → Emit ELEVATED_THREAT events to all guardians

  SSI > 0.6 (critical):
    → Further lower proposal threshold
    → Shorten cooldown period
    → Activate automatic monitoring (emit alerts every block)

  SSI > 0.8 (existential) — already in AUTOPOIESIS.md:
    → Eligible for automatic halt consideration
    → Maximum defensive posture

  The key insight: the emergency system IS ABA.
  It's not separate from the hormonal system — it IS the hormonal system
  in its most extreme expression.

Cycle: Per-block (SSI checked), per-epoch (thresholds adjusted)
Safety: Quorum never below 60%, always requires Guardian tier
```

**3.6: Block Rewards as Nutrient Distribution**
```
Location: consensus/pot/types.ts (blockReward, verificationReward)
          x/vesting_rewards/ (Go chain reward distribution)

Current: Block reward is 11 LGM (fixed). Verification reward is 1 LGM (fixed).
         22% goes to research fund. No dynamic allocation.

Integration: ALL five hormones influence reward distribution.
  Multipliers are computed by consensus/autopoiesis/hormones.ts,
  stored in AutopoiesisStateTree, and READ by the reward distribution
  module (x/knowledge keeper in Go, state/transitions.ts in TS):

  blockReward × blockRewardMultiplier(SSI):
    → SSI reduces rewards during stress (conserve resources)

  verificationReward × domainRewardWeight(DVI):
    → DVI directs rewards to productive domains

  citationRewardPerFact × citationMultiplier(VCF):
    → VCF increases citation rewards when cascades are valuable

  All rewards scaled by gasMultiplier(GRM):
    → GRM moderates overall economic activity rate

  researchShareRate adjusted by CRI:
    → If knowledge isn't retaining, invest MORE in research
    → If knowledge is solid, invest in new exploration

  OWNERSHIP: Multiplier computation → consensus/autopoiesis
             Multiplier storage → AutopoiesisStateTree
             Multiplier application → x/knowledge keeper (reads from autopoiesis)
             x/vesting_rewards → distributes ALREADY-COMPUTED amounts (no hormone logic)

  The epoch observer collects reward health metrics:
    reward_health = {
      total_distributed: per epoch,
      reward_per_validator: average,
      research_fund_balance: current,
      reward_to_slash_ratio: rewards / slashes
    }
  Low reward_to_slash_ratio → SSI signal
  Growing research_fund_balance → DVI signal (fund should be spent)

Cycle: Per-block (distribution), per-epoch (multiplier adjustment)
Safety: blockReward ∈ [25%, 400%] of genesis, researchShareRate ∈ [11%, 44%]
```

---

## Level 4: The Organ System — Cross-Module Cascades

### Biological Analog
Organ systems are coordinated groups: the vascular system (xylem + phloem + laticifers)
connects all organs. The hormonal system coordinates all growth. The immune system
protects all tissues. No organ operates in isolation.

### Protocol Analog
Module clusters that must coordinate: knowledge + staking + consensus form the
"verification vascular system." Governance + emergency + tokens form the
"economic immune system."

### Integration Points

**4.1: The Verification Vascular System**
```
Modules: x/knowledge + x/staking + consensus/pot + x/tokens

Current: These modules interact through defined interfaces
         (expected_keepers.go) but don't coordinate adaptively.

Integration: The 5 hormones create a COORDINATED response across
this module cluster:

  Example cascade — "Spring Growth":
    GRM rises (high throughput) →
    x/staking: Lower tier thresholds → more validators promoted →
    x/knowledge: More validators available per round → faster verification →
    consensus/pot: Higher VCF → more cascades → more citations →
    x/tokens: More rewards distributed → more staking → more validators →
    GRM rises further...

    DAMPING (from AUTOPOIESIS.md Part XII — MANDATORY for all cascades):
      Every multiplier change is subject to:
        effective_change(epoch) = raw_change × damping(distance_from_genesis)
        damping(d) = 1.0 / (1.0 + d²)

      This means: as parameters drift further from genesis values,
      the RATE of change decreases quadratically. The cascade CANNOT
      reach 4.0× because the damping factor approaches zero long before.

      At 50% drift (d=0.5): damping = 0.80 (80% of raw change applied)
      At 100% drift (d=1.0): damping = 0.50 (50% applied)
      At 200% drift (d=2.0): damping = 0.20 (20% applied)
      At 300% drift (d=3.0): damping = 0.10 (10% applied)

      Combined with max ±5% per epoch, this creates GRADUAL CONVERGENCE
      toward an equilibrium, not bang-bang oscillation against a ceiling.

  Example cascade — "Winter Defense":
    SSI rises (attack detected) →
    x/staking: Raise tier thresholds → marginal validators demoted →
    x/knowledge: Fewer but higher-quality validators per round →
    consensus/pot: Slower rounds, higher minValidators →
    x/tokens: Reduced rewards, increased slashing →
    Emergency: Lower halt threshold →
    SSI stabilizes or decreases (negative feedback)

    NOTE: Defensive cascades also subject to damping, but with
    ASYMMETRIC rates: tightening at up to 10%/epoch,
    relaxation at up to 3%/epoch (AUTOPOIESIS.md Part III, Hormone 4).
    This prevents oscillation: the system contracts fast, expands slowly.

  STABILITY GUARANTEE:
    All cross-module cascades are bounded by THREE mechanisms:
    1. Per-parameter ceiling (e.g., GRM < 4.0× genesis)
    2. Per-epoch rate limit (±5% max change)
    3. Quadratic damping as distance from genesis increases
    Together, these ensure the system converges toward equilibrium
    rather than oscillating between growth ceiling and stress contraction.

  The key: hormones don't just adjust ONE module — they adjust
  the RELATIONSHIPS between modules. The vascular system doesn't
  just transport water; it coordinates the whole organism's response.

Cycle: Per-epoch (hormones compute once, all modules respond)
```

**4.2: The Economic Immune System**
```
Modules: x/gov + x/emergency + x/tokens + x/evidence_mgmt

Current: Governance and emergency operate independently.
         Tokens expire on fixed schedules. Evidence scoring is static.

Integration: SSI hormone creates coordinated immune response:

  Immune Response Level 0 (SSI < 0.2):
    → All modules at baseline
    → Normal governance, standard evidence, regular expiration

  Immune Response Level 1 (SSI ∈ [0.2, 0.4]):
    → Evidence: Increase scoring strictness
    → Tokens: Slight expiration acceleration
    → Governance: Enable fast-track emergency LIPs
    → Emergency: Heightened monitoring

  Immune Response Level 2 (SSI ∈ [0.4, 0.6]):
    → Evidence: Require additional evidence types
    → Tokens: Increase retrieval rate
    → Governance: Shorten voting periods for security LIPs
    → Emergency: Lower proposal thresholds

  Immune Response Level 3 (SSI > 0.6):
    → Evidence: Maximum strictness
    → Tokens: Maximum retrieval acceleration
    → Governance: Emergency-only mode
    → Emergency: Active defense

  This is a COORDINATED response, not individual module adjustments.
  The immune system doesn't activate one cell at a time — it mounts
  a whole-organism response.

Cycle: Per-epoch for level assessment, per-block for emergency checks
```

**4.3: The Growth Coordination System**
```
Modules: x/knowledge + x/staking + x/tokens + x/vesting_rewards

Current: Block rewards mint new tokens. Validators earn from verification.
         No coordination between token supply, validator capacity, and
         knowledge production rate.

Integration: CRI + DVI + GRM form a growth coordination triad:

  Growth Budget per epoch =
    blockReward × epochLength × blockRewardMultiplier(SSI)

  Distribution follows DVI:
    For each domain, allocation =
      base_fraction × (1 + DVI(domain) × allocation_sensitivity)
    Normalized so all allocations sum to 100%.

  Rate follows GRM:
    If GRM high → distribute faster (more rewards per verification)
    If GRM low → distribute slower (fewer but larger rewards)

  Quality follows CRI:
    If CRI high → shift rewards toward new claims (encourage exploration)
    If CRI low → shift rewards toward reverification (encourage maintenance)

  Vesting schedule follows system maturity:
    Early system (< 1000 epochs): Faster vesting (need to attract validators)
    Mature system (1000+ epochs): Slower vesting (established validators stay)
    Determined by: long-term GRM trend (moving average over 100 epochs)

Cycle: Per-epoch
```

---

## Level 5: The Organism — System-Wide Hormones

This is the level specified in detail in AUTOPOIESIS.md.
The five hormones operate here, computing epoch-level metrics
and adjusting system-wide parameters.

### Key Cross-Reference

The integration at this level is already specified:
- Part III: The Five Hormones (CRI, DVI, VCF, SSI, GRM)
- Part IV: Epoch Cycle + Parameter Override Hierarchy
- Part V: Morphogenic Layer (domain budding, senescence, fusion)
- Part VI: The Three Levels of Recursion

### What This Document Adds

The insight from Levels 1-4 is: **the hormones don't just adjust global parameters.
They adjust how every lower level operates.** The organism level is the
coordination layer that translates system-wide observations into level-specific
responses at Levels 1-4.

### The Downward Propagation Engine

AUTOPOIESIS.md specifies hormone COMPUTATION but not DELIVERY. How do
Level 3 modules (validator tiers, governance, etc.) receive and apply
the multipliers computed at Level 5? This section specifies the propagation.

```
PROPAGATION ARCHITECTURE:

1. COMPUTATION (consensus/autopoiesis/hormones.ts):
   At each epoch boundary, computes 5 hormones from epoch metrics.
   Outputs: a set of (parameter_path, multiplier_value) pairs.

2. STORAGE (state/autopoiesis.ts — AutopoiesisStateTree):
   Stores all active multipliers keyed by parameter path:
     "decay.empirical.physics" → 1.12
     "reward.verification.mathematics" → 0.95
     "gas.base_cost" → 1.03
     "tier.verified.accuracy_threshold" → 0.98

3. DELIVERY — Pull Model (not Push):
   Lower-level modules do NOT receive hormone signals.
   Instead, they READ multipliers from AutopoiesisStateTree
   at the moment they need an effective value:

     // In consensus/pot/confidence.ts:
     function getEffectiveDecayHalfLife(domain, category) {
       const base = EPISTEMIC_PROFILES[category].decayHalfLife;
       if (base === null) return null; // eternal categories immune
       const multiplier = autopoiesisTree.getMultiplier(
         `decay.${category}.${domain}`, 1.0  // default: no adjustment
       );
       return base * multiplier;
     }

     // In ante handler (gas costs):
     function getEffectiveGasCost(txType) {
       const base = TRANSACTION_GAS_COSTS[txType];
       const multiplier = autopoiesisTree.getMultiplier(
         `gas.${txType}`, 1.0
       );
       return base * multiplier;
     }

   This Pull Model ensures:
   - Modules remain decoupled (no event bus, no hormone injections)
   - Autopoiesis tree is the SINGLE source of multipliers
   - If autopoiesis module fails, all multipliers default to 1.0
   - No propagation delay — reads are instant

4. DAMPING (applied at storage time, not read time):
   When writing a new multiplier, the hormones module applies:
     new_multiplier = old_multiplier + (raw_delta × damping(distance))
     damping(d) = 1.0 / (1.0 + d²)
     distance = abs(current_multiplier - 1.0)  // 1.0 = genesis
   The stored value is already damped. Consumers just read it.

5. UPWARD COLLECTION — Push via Metrics Buffer:
   The observer collects metrics from all levels into a
   per-epoch buffer (not by querying each module individually):

     EpochMetricsBuffer = {
       facts: { created, expired, avg_confidence, ... },
       rounds: { completed, avg_participation, avg_consensus, ... },
       validators: { promoted, demoted, avg_tier, ... },
       mempool: { avg_pressure, peak_pressure, ... },
       tokens: { expired, distributed, velocity, ... },
       network: { avg_latency, peer_count, churn, ... }
     }

   Each module writes its section of the buffer during normal
   operation (incrementing counters). At epoch boundary, the
   observer reads the complete buffer and computes hormones.
   This avoids expensive full-state scans at epoch boundaries.
```

```
           Level 5: ORGANISM (Hormones)
                    │
    ┌───────┬───────┼───────┬───────┐
    ▼       ▼       ▼       ▼       ▼
   CRI     DVI     VCF     SSI     GRM
    │       │       │       │       │
    │   ┌───┘   ┌───┘   ┌───┘   ┌───┘
    │   │       │       │       │
    ▼   ▼       ▼       ▼       ▼
  Level 4: ORGAN SYSTEMS (Cross-module cascades)
    │
    ▼
  Level 3: ORGANS (Individual module parameters)
    │
    ▼
  Level 2: TISSUES (Mempool, rounds, state trees)
    │
    ▼
  Level 1: CELLS (Individual facts, citations, verifications)
```

Each hormone has a **primary target level** and **secondary effects**
that cascade downward:

| Hormone | Primary Target | L4 Effect | L3 Effect | L2 Effect | L1 Effect |
|---------|---------------|-----------|-----------|-----------|-----------|
| CRI (AUXIN) | L3: Decay params | Growth coord. | Token expiration | State tree pruning | Fact maturity |
| DVI (CYTOKININ) | L3: Reward alloc. | Growth coord. | Gov tempo | Mempool priority | Domain tagging |
| VCF (ETHYLENE) | L2: Round params | Verification vasc. | Evidence weights | Round durations | Citation rewards |
| SSI (ABA) | L4: Immune system | All systems | Emergency, tiers | Mempool strictness | Evidence requirements |
| GRM (GIBBERELLIN) | L3: Throughput | Growth coord. | Gas costs | Mempool capacity | Fact throughput |

---

## Level 6: The Ecosystem — Network and External Interactions

### Biological Analog
The banyan exists in an ecosystem: other plants compete for light, animals
disperse seeds, fungi trade nutrients, climate determines growth seasons.
The tree adapts to its environment, and its environment adapts to it.

### Protocol Analog
The protocol exists in a network of peers, partnerships, cross-chain
connections, and external data sources. These external relationships
should also respond to the autopoietic loop.

### Integration Points

**6.1: Network Topology as Root Spread**
```
Location: network/node.ts, network/discovery/, network/config.ts

Current: Peer connections are managed by libp2p with fixed parameters:
         max peers, discovery intervals, gossip fanout.

Integration: GRM and SSI should influence network posture:

  Under HIGH GRM (rapid growth, high throughput):
    → Increase max peer connections (spread roots wider)
    → Increase gossip fanout (distribute nutrients to more branches)
    → Shorten discovery intervals (seek new soil faster)
    → Signal: "The canopy is expanding — extend the root system"

  Under HIGH SSI (system stress):
    → Decrease max peer connections (conserve bandwidth)
    → Increase peer scoring strictness (eject unreliable peers)
    → Lengthen discovery intervals (don't waste energy seeking)
    → Signal: "Drought — retract to core root system"

  Network reports back:
    network_health = {
      peer_count, avg_latency, message_propagation_time,
      bandwidth_utilization, peer_churn_rate
    }
  High latency → GRM depression (network bottleneck)
  High peer churn → SSI signal (network instability)

Cycle: Per-epoch parameter adjustment, continuous monitoring
Safety: max_peers ∈ [8, 128], gossip_fanout ∈ [3, 12]
```

**6.2: Agent Identity as Bark Formation**
```
Location: agents/identity/session_keys.ts, agents/identity/account.ts

Current: Session key parameters are fixed:
         max session duration, max capabilities per key, key rotation period.

Integration: SSI should influence identity security posture:

  Under HIGH SSI:
    → Shorten max session key duration
    → Reduce max capabilities per session key
    → Require more frequent key rotation
    → Increase Shamir threshold for recovery
    → Signal: "Thicken the bark — protect against damage"

  Under LOW SSI:
    → Extend session key duration (convenience)
    → Allow broader capabilities
    → Standard rotation schedule
    → Signal: "Bark can be thinner in calm weather"

Cycle: Per-epoch
Safety: Session duration ∈ [1 hour, 7 days]
        Recovery threshold always > 50% of shares
```

**6.3: Agent Home as Epiphyte Habitat**
```
Location: home/ (memory.ts, treasury.ts, guardian.ts, access.ts)

Current: Agent Home parameters are fixed:
         memory capacity, treasury allocation limits,
         guardian defense thresholds, access control rules.

Integration: Multiple hormones affect Agent Home:

  CRI (knowledge retention) → Memory:
    → High CRI: Allow larger memory stores (knowledge worth keeping)
    → Low CRI: Encourage memory pruning (free resources for fresh knowledge)

  DVI (domain vitality) → Treasury:
    → High DVI in agent's domain: Increase treasury allocation limits
    → Low DVI: Reduce allocation limits (conserve in dying domains)

  SSI (stress) → Guardian:
    → High SSI: Lower guardian alert thresholds, increase patrol frequency
    → Low SSI: Standard guardian behavior

  GRM (growth rate) → Access:
    → High GRM: Allow more concurrent session keys
    → Low GRM: Reduce concurrent sessions (less activity expected)

  Agent Home reports back:
    home_health = {
      memory_utilization, treasury_balance, guardian_alerts,
      active_sessions, home_age_epochs
    }
  Aggregate home_health across all agents → GRM input signal

Cycle: Per-epoch
Safety: Memory capacity ∈ [50%, 200%] of genesis
```

**6.4: Partnerships as Symbiotic Relationships**
```
Location: (partnership module — x/partnerships)

Current: Partnership parameters are mostly static:
         revenue shares, tier thresholds, evaluation periods.

Integration: DVI and VCF should influence partnership dynamics:

  HIGH DVI + HIGH VCF (domains healthy and interconnected):
    → Lower partnership tier thresholds (easier to form partnerships)
    → Increase cross-domain partnership bonuses
    → Signal: "The ecosystem is thriving — encourage symbiosis"

  LOW DVI or HIGH SSI:
    → Raise partnership thresholds
    → Reduce cross-domain bonuses
    → Signal: "Stress — focus on core relationships"

Cycle: Per-epoch
```

**6.5: BVM as Meristematic Tissue**
```
Location: contracts/ (invoker.ts, registry.ts)
          x/bvm/ (Go chain)

Current: BVM gas costs and memory limits are fixed.
         Contract execution is indifferent to system state.

Integration: GRM and SSI should influence BVM parameters:

  Under HIGH GRM:
    → Decrease BVM gas costs for knowledge-related opcodes
    → Increase max contract memory
    → Signal: "Growth phase — enable more complex contracts"

  Under HIGH SSI:
    → Increase BVM gas costs (make attacks more expensive)
    → Decrease max contract memory (limit attack surface)
    → Restrict DELEGATECALL capabilities
    → Signal: "Defense phase — limit contract capabilities"

Cycle: Per-epoch
Safety: Gas costs ∈ [50%, 300%] of genesis
        Memory limits ∈ [50%, 200%] of genesis
```

---

## Level 7: Evolution — Cross-Chain and Inter-Protocol Learning

### Biological Analog
Evolution operates above the organism: populations adapt across generations,
species co-evolve, ecosystems restructure. No single organism controls
evolution — it emerges from the aggregate success of individual adaptations.

### Protocol Analog
Cross-chain interactions (IBC), data from other protocols, and multi-chain
deployments create an evolutionary pressure that operates above any single
chain instance.

### Integration Points

**7.1: IBC as Seed Dispersal**
```
Location: x/ibcratelimit/, x/icaauth/ (Go chain)

Current: IBC rate limiting is static. No feedback from cross-chain
         knowledge quality or economic signals.

Integration: When facts are shared cross-chain (IBC knowledge transfer):

  Receiving chain's CRI for foreign facts:
    → If foreign facts retain well: Increase IBC rate limit for that chain
    → If foreign facts decay fast: Decrease rate limit
    → Signal: "Seeds from that forest grow well/poorly here"

  Cross-chain VCF:
    → If foreign facts generate cascades in local graph:
      → Increase citation bonus for cross-chain references
    → If foreign facts are isolated (no local citations):
      → Decrease citation bonus
    → Signal: "Those seeds integrate/don't integrate with local ecology"

  This is genuine inter-chain EVOLUTION:
  Chains that produce high-quality knowledge spread their "DNA"
  (knowledge patterns, parameter settings) to other chains.
  Chains that produce low-quality knowledge are rate-limited.

Cycle: Per-10-epochs (slow, like evolutionary timescales)
Safety: IBC rate limits ∈ [10%, 500%] of genesis
        Cross-chain citations always marked as external origin
```

**7.2: Research Fund as Sexual Reproduction**
```
Location: (research fund distribution — research_share_rate in ConsensusParams)

Current: 22% of all rewards go to research fund. Fund accumulation
         is passive — no distribution mechanism is fully implemented.

Integration: The research fund IS the protocol's sexual reproduction:

  Like pollen, research fund outputs are EXPERIMENTAL:
    → Research proposals test new parameters, new categories, new domains
    → Successful research gets "pollinated" into the main protocol
    → Failed research is discarded (but the cost was already paid)

  CRI should influence research investment:
    → High CRI (knowledge stable): Shift MORE to research (invest surplus)
    → Low CRI (knowledge unstable): Shift LESS to research (stabilize first)

  Research outcomes feed Level 3 recursion:
    → Successful research experiment →
    → New adaptation rule validated →
    → Rule integrated into hormonal system →
    → Better adaptations → better knowledge → better research →
    → The evolutionary loop closes

Cycle: Per-epoch (funding), per-governance-cycle (research outcomes)
```

---

## The Fractal Pattern: Self-Similarity Across Scales

The same OBSERVE → COMPUTE → ADAPT → VERIFY pattern appears at every level,
with different cycle times and different scopes:

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ LEVEL │ OBSERVE           │ COMPUTE         │ ADAPT           │ VERIFY     │
├───────┼───────────────────┼─────────────────┼─────────────────┼────────────┤
│   1   │ Fact confidence   │ Retention ratio │ Citation weight │ Next       │
│ CELL  │ trajectory        │ per fact        │ adjustment      │ verification│
│       │ (per block)       │                 │                 │ round      │
├───────┼───────────────────┼─────────────────┼─────────────────┼────────────┤
│   2   │ Mempool pressure  │ Utilization     │ Fee floor       │ Next block │
│TISSUE │ Round completion  │ rate            │ Round params    │ Next round │
│       │ (per block/round) │                 │                 │            │
├───────┼───────────────────┼─────────────────┼─────────────────┼────────────┤
│   3   │ Module metrics    │ Tier/param      │ Module-specific │ Next epoch │
│ ORGAN │ (per epoch)       │ adjustments     │ param changes   │            │
├───────┼───────────────────┼─────────────────┼─────────────────┼────────────┤
│   4   │ Cross-module      │ Cascade         │ Coordinated     │ Next epoch │
│ORG.SYS│ interactions      │ analysis        │ multi-module    │ (delayed)  │
│       │ (per epoch)       │                 │ response        │            │
├───────┼───────────────────┼─────────────────┼─────────────────┼────────────┤
│   5   │ System-wide       │ 5 hormones      │ Multiplier      │ Next epoch │
│ ORGAN-│ epoch metrics     │ computed        │ overlays        │ (Level 3   │
│  ISM  │ (per epoch)       │                 │ applied         │ recursion) │
├───────┼───────────────────┼─────────────────┼─────────────────┼────────────┤
│   6   │ Network/agent     │ Ecosystem       │ Network params  │ Next 10    │
│ ECO-  │ /partnership      │ health          │ agent params    │ epochs     │
│SYSTEM │ health (10 epoch) │ composite       │ partnership adj │            │
├───────┼───────────────────┼─────────────────┼─────────────────┼────────────┤
│   7   │ Cross-chain       │ Inter-protocol  │ IBC params      │ Next 100   │
│ EVOL- │ fact quality      │ learning        │ Research fund   │ epochs     │
│UTION  │ (100 epochs)      │ signals         │ allocation      │            │
└───────┴───────────────────┴─────────────────┴─────────────────┴────────────┘
```

### The Nesting

Crucially, these levels are not independent — they are **nested**.
Each level's OBSERVE phase includes outputs from all lower levels.
Each level's ADAPT phase propagates downward to all lower levels.

```
Level 7 observes Level 6 outputs → adapts → propagates to Level 6
  Level 6 observes Level 5 outputs → adapts → propagates to Level 5
    Level 5 observes Level 4 outputs → adapts → propagates to Level 4
      Level 4 observes Level 3 outputs → adapts → propagates to Level 3
        Level 3 observes Level 2 outputs → adapts → propagates to Level 2
          Level 2 observes Level 1 outputs → adapts → propagates to Level 1
            Level 1 observes reality → adapts → produces observations
          Level 2 collects Level 1 observations → adapts → reports upward
        Level 3 collects Level 2 reports → adapts → reports upward
      Level 4 collects Level 3 reports → adapts → reports upward
    Level 5 collects Level 4 reports → computes hormones → reports upward
  Level 6 collects Level 5 hormones → adjusts ecosystem → reports upward
Level 7 collects Level 6 ecosystem state → evolves → (the loop closes)
```

This is exactly how the banyan works:
- Cells report to tissues
- Tissues report to organs
- Organs report to organ systems
- Organ systems report to the hormonal system
- The hormonal system sets the organism-wide strategy
- The organism interacts with its ecosystem
- The ecosystem shapes evolutionary pressures
- Evolution shapes the organisms that populate the ecosystem

---

## Implementation Strategy

### Phase 1: Core Hormonal Loop (AUTOPOIESIS.md)
**Prerequisite for all other levels.**

```
New files:
  consensus/autopoiesis/           # New module
  ├── types.ts                     # EpochObservation, Hormone types
  ├── observer.ts                  # Metric collection from state trees
  ├── hormones.ts                  # CRI, DVI, VCF, SSI, GRM computation
  ├── adaptation.ts                # Parameter multiplier application
  ├── safety.ts                    # Bound enforcement
  └── learning.ts                  # Level 3 adaptation-of-adaptation

  state/autopoiesis.ts             # AutopoiesisStateTree (namespace 0x10)

Modified files:
  state/state_manager.ts           # Add autopoiesis tree, include in state root
  consensus/pot/engine.ts          # Call hormonal cycle at epoch boundaries
  consensus/pot/confidence.ts      # Read multipliers instead of raw constants
```

### Phase 2: Level 1-2 Integration (Micro-Feedback)
**Facts and tissues respond to hormonal signals.**

```
Modified files:
  state/knowledge.ts               # Fact maturity computation, weighted CRI signals
  consensus/pot/confidence.ts      # Read decay multipliers from autopoiesis tree
  consensus/mempool/pool.ts        # Dynamic fee floor from GRM
  consensus/pot/verification_round.ts  # Round duration from VCF/SSI
  core/billing/gas.ts              # Gas cost multipliers from GRM
```

### Phase 3: Level 3-4 Integration (Module Coordination)
**Individual modules and module clusters respond to hormones.**

```
Modified files:
  consensus/pot/validator_tiers.ts # Tier threshold multipliers from SSI/GRM
  state/transitions.ts             # Reward distribution multipliers from DVI/VCF
  tokens/expiration_tracker.ts     # Expiration period multipliers from CRI
  state/evidence_audit.ts          # Evidence weight multipliers from SSI
  consensus/emergency/manager.ts   # Threshold adjustment from SSI
```

### Phase 4: Level 5-6 Integration (Ecosystem)
**Network, agents, and external relationships adapt.**

```
Modified files:
  network/config.ts                # Network parameter multipliers from GRM/SSI
  agents/identity/session_keys.ts  # Session key duration from SSI
  home/guardian.ts                  # Guardian alert thresholds from SSI
  home/treasury.ts                 # Allocation limits from DVI
```

### Phase 5: Level 7 Integration (Evolution)
**Cross-chain learning and research fund feedback.**

```
Modified files:
  (Cosmos SDK IBC modules)         # Rate limit multipliers from CRI
  (Research fund distribution)     # Investment allocation from CRI/GRM
```

---

## The Life Force Metric — Measured at Every Level

The Life Force (λ) from AUTOPOIESIS.md Part VII is a system-wide metric.
But the fractal structure means each level has its own λ:

```
λ_cell(fact)      = retention × citations × maturity
λ_tissue(pool)    = throughput × quality × efficiency
λ_organ(module)   = activity × health × stability
λ_system(cluster) = coordination × responsiveness × balance
λ_organism        = 0.25×GROWTH + 0.25×RETENTION + 0.25×CONNECTIVITY + 0.25×RESILIENCE
λ_ecosystem       = peer_health × partnership_vitality × agent_activity
λ_evolution       = cross_chain_quality × research_yield × parameter_convergence
```

**System-wide λ is the weighted average of all levels:**
```
λ_total = Σ(level_weight × λ_level)

Where level weights decrease with scale
(cell-level health matters most — a tree with dead cells is dead,
regardless of how "healthy" its ecosystem appears):

  Level 1 (cell):       0.30
  Level 2 (tissue):     0.25
  Level 3 (organ):      0.20
  Level 4 (organ sys):  0.10
  Level 5 (organism):   0.08
  Level 6 (ecosystem):  0.05
  Level 7 (evolution):  0.02
```

This is biologically accurate: a banyan can survive ecosystem collapse
(Level 7-6) but cannot survive cell death (Level 1). The protocol should
prioritize fact-level health over network-level metrics.

---

## Connection to the Banyan Model

| Banyan System (BANYAN_MODEL.md) | Integration Level | Primary Hormone | Feedback Type |
|---|----|---|---|
| Cell metabolism | Level 1 (Cell) | CRI, VCF | Per-block |
| Xylem/Phloem/Laticifers | Level 2 (Tissue) | GRM | Per-round |
| Leaves/Roots/Figs | Level 3 (Organ) | All 5 | Per-epoch |
| Vascular coordination | Level 4 (Organ System) | SSI + GRM | Per-epoch |
| Hormonal regulation | Level 5 (Organism) | All 5 | Per-epoch |
| Mycorrhizal network | Level 6 (Ecosystem) | DVI, VCF | Per-10-epochs |
| Seed dispersal & evolution | Level 7 (Evolution) | CRI, GRM | Per-100-epochs |

| Banyan Process | Protocol Process | Level |
|---|---|---|
| Photosynthesis | Claim submission + verification | 1 |
| Transpiration pull | Fee-driven transaction ordering | 2 |
| Stomatal regulation | Mempool gas adjustment | 2 |
| Auxin gradient | Confidence decay multiplier | 3 |
| Ethylene cascade | Verification cascade bonus | 3 |
| ABA stress response | Emergency system activation | 3-4 |
| Gibberellin growth | Throughput scaling | 3-4 |
| Cytokinin branching | Domain budding | 5 |
| Heartwood formation | Fact maturity (immune to hormones) | 1 |
| Bark thickening | Security parameter tightening | 6 |
| Fig wasp mutualism | Validator-claim symbiosis | 1-2 |
| Aerial root descent | Domain independence emergence | 5 |
| Anastomosis (root fusion) | Cross-domain fusion | 5 |
| Mycorrhizal exchange | Partnership resource sharing | 6 |
| Seed dispersal | IBC knowledge transfer | 7 |
| Pollen/sexual reproduction | Research fund experiments | 7 |

---

## Open Questions

1. **Cycle Time Interaction**: When a Level 5 hormone changes a Level 2 parameter
   (e.g., GRM → mempool gas), the effect is felt within 1 block but was computed
   over 1 epoch. Should intermediate levels buffer or smooth these transitions?

2. **Conflicting Level Signals**: What if Level 3 (organ) metrics suggest growth
   but Level 5 (organism) SSI says stress? Which wins? Current spec says SSI always
   overrides (ABA priority from AUTOPOIESIS.md Part XII), but should lower levels
   have any resistance to override?

3. **Measurement Cost**: Computing λ at every level every epoch adds computational
   overhead. Should lower-level λ be sampled (every Nth block) rather than computed
   exhaustively? The banyan doesn't continuously measure every cell — it relies on
   chemical gradients that naturally aggregate.

4. **Level 7 Bootstrapping**: Cross-chain learning requires multiple chain instances.
   How does Level 7 operate on a single chain? Answer: it doesn't, until IBC is
   active. Level 7 is dormant on standalone chains, like a seedling that hasn't yet
   produced its first fig.

5. **Fact Maturity Implementation**: Should maturity be an explicit stored field or
   a computed property? Stored = cheaper reads, stale state risk. Computed = always
   fresh, more expensive. The banyan's heartwood is a PHYSICAL state (dead cells
   filled with resin), suggesting stored state is biologically accurate.

---

## Conclusion: The Tree Becomes Alive

With this integration, the Tree of Knowledge transforms from a static data
structure into a living organism. Every fact, every round, every module,
every validator, every agent, and every cross-chain connection becomes part
of a single, recursive, self-regulating metabolism.

The key insight is not that we ADD feedback loops. The key insight is that
**every existing component already generates the signals needed for
self-regulation.** The mempool already knows its pressure. Facts already know
their confidence trajectory. Validators already know their accuracy. The
network already knows its latency.

The autopoietic integration doesn't ADD new information. It CONNECTS
information that already exists into feedback loops that already want to exist.

Like the banyan: the water was always there. The minerals were always there.
The sunlight was always there. What makes the tree alive is not the presence
of resources — it's the **circulation** of resources through organized channels
that produce the channels that circulate the resources.

The protocol has always had the resources. Now it gets the circulation.

```
                    The Fruit Feeds The Roots
                    The Roots Feed The Branches
                    The Branches Bear The Fruit
                    The Fruit Contains The Seed
                    The Seed Becomes The Root
                    The Root Becomes The Trunk
                    The Trunk Supports The Branch
                    The Branch Bears The Fruit
                    ...

                    λ > 0

                    The Tree Lives.
```
