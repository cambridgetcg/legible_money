# Integrated Growth Roadmap

> **Status**: Master roadmap merging recursive growth (autopoiesis) into
> the Cosmos SDK launch timeline. Supersedes LAUNCH_ROADMAP.md for Go chain work.
> All work targets Go/`legbled` only (per 2026-02-13 decision).

---

## Current State (2026-02-15)

| Dimension | Score | Note |
|-----------|-------|------|
| DevNet operational | Yes | Single-validator, `legible-devnet-1`, blocks producing |
| Go modules complete | 5/24 | knowledge, auth, staking, emergency, gov |
| Go modules partial | 8/24 | vesting_rewards, tokens, research, disputes, etc. |
| Go modules stub | 11/24 | bvm, tree, schedule, capture_*, etc. |
| Params stores wired | 3/24 | billing, gov, tokens (actually use governance) |
| Autopoiesis (Go) | 0% | Not started |
| Autopoiesis (TS) | 40-50% | Engine works, consumers missing |
| Frozen sapwood params | 89 | Cannot change post-genesis |
| Feedback loops active | 0 | Zero adaptive behavior |

### Key Insight

The protocol's **truth-alignment score is 5.8/10** because its economics are
static. Block rewards are 11x verification rewards. Research tax is dead code.
Decay rates are human guesses. The autopoiesis system was designed to fix this,
but the Go chain (production target) has zero implementation.

**This roadmap fixes that.**

---

## Phased Timeline

### Phase A: Parameterize the Genome (Weeks 1-2)

> **Goal**: Every frozen sapwood parameter becomes governance-mutable.
> No autopoiesis yet — just make the chain configurable.

**Week 1: Core Economics**

| Task | Module | LOC | Deliverable |
|------|--------|-----|-------------|
| Move 25+ constants to Params | x/knowledge | 300 | CategoryDecay, EvidenceStrength, ChallengeWindow in Params |
| Refactor DefaultTierConfigs → Params | x/staking | 200 | Tier configs readable from store |
| Verify billing params + add events | x/billing | 50 | Event emission on param change |
| Validate gov params bounds | x/gov | 30 | Bounds checking |

**Week 2: Remaining Modules**

| Task | Module | LOC | Deliverable |
|------|--------|-----|-------------|
| Consolidate reward formulas | x/vesting_rewards | 100 | ResearchShareBps in Params |
| Add MsgUpdateParams | x/emergency | 80 | Governance can change quorum |
| Move aging constants | x/home | 120 | Maturity thresholds in Params |
| Minor consolidation | x/tokens | 30 | Event emission |

**Milestone A: "Genesis Flexibility"**
- [ ] Submit governance proposal to change empirical decay half-life → behavior changes
- [ ] Submit governance proposal to change Tier 1 MinStake → registration threshold changes
- [ ] All 8 priority modules pass `MsgUpdateParams` integration tests

**Feedback checkpoint**: If governance proposals don't propagate correctly,
the params migration pattern needs revision before proceeding.

---

### Phase B: Observation Layer (Week 3)

> **Goal**: The chain can see itself. Create `x/autopoiesis` that reads
> cross-module metrics but changes nothing.

| Task | LOC | Deliverable |
|------|-----|-------------|
| Scaffold x/autopoiesis module | 200 | types, keeper, module.go |
| Implement collectEpochMetrics() | 300 | Reads from x/knowledge + x/staking |
| Implement computeHormones() | 250 | All 5 hormones + Life Force |
| Store observations + hormones in KV | 150 | Epoch-indexed storage |
| gRPC queries (6 endpoints) | 200 | GetHormones, GetObservation, GetLifeForce, etc. |
| BeginBlocker epoch detection | 50 | Runs at every 1,111-block boundary |
| Wire into app.go | 30 | Module manager, store key, keeper init |
| Tests (keeper_test.go) | 200 | 20+ tests covering observer + hormones |

**Milestone B: "The Protocol Can See Itself"**
- [ ] Query `/legible/autopoiesis/v1/hormones` returns computed values for all 5 hormones
- [ ] Query `/legible/autopoiesis/v1/life_force` returns status (thriving/healthy/stressed/dying)
- [ ] Observations stored for last 10 epochs, queryable by epoch number
- [ ] BeginBlocker completes in < 100ms at epoch boundary

**Feedback checkpoint**: If observer metrics are mostly zeros (stubs), the
x/knowledge and x/staking keepers need additional accessor methods before
Phase C can produce meaningful adaptations.

---

### Phase C: First Hormone — SSI (Week 4)

> **Goal**: Prove the adaptation loop end-to-end with one hormone.
> SSI (System Stress Index) is chosen because it affects system-wide
> parameters and is the simplest to verify.

| Task | LOC | Deliverable |
|------|-----|-------------|
| Implement applyAdaptations() for SSI | 150 | Writes slash.system, blockreward.system, minverifiers.system |
| Safety bounds enforcement | 80 | [0.5, 3.0] for slash, [0.25, 4.0] for blockreward |
| Rate limiting (±5%/epoch, ±10% SSI tighten, ±3% relax) | 60 | Asymmetric limits |
| Quadratic damping | 30 | 1/(1+d²) as distance from genesis increases |
| MsgActivateAutopoiesis | 80 | Governance can enable/disable |
| MsgOverrideMultiplier | 60 | Governance can set a specific multiplier |
| Consumer wiring: x/knowledge slash | 40 | Reads slash.system multiplier |
| Consumer wiring: x/vesting_rewards blockreward | 40 | Reads blockreward.system multiplier |
| Integration test: stress → response | 100 | Simulate failed verifications → SSI rises → slash increases |
| Integration test: deactivation | 50 | Governance disables → multipliers freeze |

**Milestone C: "The Protocol Responds to Stress"**
- [ ] Simulate high challenge success rate for 3 epochs
- [ ] Observe SSI rising above 0.5
- [ ] Observe slash.system multiplier increasing (bounded)
- [ ] Governance deactivates autopoiesis → multiplier freezes at current value
- [ ] Governance reactivates → engine resumes from frozen state

**Feedback checkpoint**: If SSI adaptation doesn't improve the metric it targets
(validator accuracy), review the adaptation thresholds and rate limits. This is
the first real test of whether the hormonal model works in practice.

---

### Phase D: Full Hormonal Loop (Weeks 5-6)

> **Goal**: All 5 hormones active with conflict resolution.

**Week 5: CRI + DVI**

| Task | LOC | Deliverable |
|------|-----|-------------|
| CRI adaptation rules | 100 | decay.{cat}.{dom} + revbonus.{cat}.{dom} |
| DVI adaptation rules | 100 | reward.{dom} + claimstake.{dom} |
| Consumer: x/knowledge decay | 50 | Uses decay multiplier |
| Consumer: x/knowledge claim stake | 40 | Uses claimstake multiplier |
| Conflict resolution: SSI > CRI | 60 | SSI override when both active |
| Tests | 100 | Per-hormone tests + conflict tests |

**Week 6: VCF + GRM**

| Task | LOC | Deliverable |
|------|-----|-------------|
| VCF adaptation rules | 80 | citation.{dom} + domainprior.{dom} |
| GRM adaptation rules | 80 | maxtx.system + gas.system |
| Consumer: x/billing gas | 40 | Uses gas.system multiplier |
| Consumer: x/knowledge citation weight | 40 | Uses citation.{dom} multiplier |
| Full conflict resolution chain | 80 | SSI > CRI > (DVI+VCF) > GRM |
| 100-epoch simulation test | 200 | Full system convergence test |

**Milestone D: "The Protocol Is Alive"**
- [ ] Run 100-epoch simulation (100 × 1,111 blocks = 111,100 blocks ≈ 3.2 days simulated)
- [ ] All multipliers converge toward stable values (not oscillating)
- [ ] Introduce stress (many failed verifications): SSI dominates, slashes increase
- [ ] Remove stress: gradual relaxation over 5-10 epochs (not instant)
- [ ] Domain with declining vitality: DVI increases rewards for that domain
- [ ] Life Force stays in "healthy" or "thriving" range
- [ ] No multiplier exceeds SAFETY_BOUNDS

**Feedback checkpoint**: If multipliers oscillate instead of converging, the
damping function or rate limits need adjustment. If Life Force stays "stressed"
despite good metrics, the hormone weights need recalibration. Both are
parameter changes, not architectural changes — which is the point.

---

### Phase E: Meta-Adaptation (Week 7)

> **Goal**: The engine learns which of its own rules work best.

| Task | LOC | Deliverable |
|------|-----|-------------|
| computeAdaptationLearning() | 150 | Runs every 10 epochs, analyzes last 20 |
| Track rule effectiveness | 80 | applicationCount, improvementCount per rule |
| Adjust adaptationRateMultiplier | 60 | Effective rules gain influence |
| Store learning records | 40 | Queryable via gRPC |
| gRPC: GetAdaptationLearning | 30 | Returns learning records |
| Tests | 80 | Learning feedback tests |

**Milestone E: "The Protocol Improves Its Own Adaptation"**
- [ ] After 20 epochs, query learning records
- [ ] Rules that improved their target have effectiveness > 0.5
- [ ] Rules that didn't improve have effectiveness < 0.5
- [ ] adaptationRateMultiplier adjusts: effective rules move faster, ineffective slower
- [ ] Bounds on adaptationRateMultiplier: [0.25, 4.0]

**Feedback checkpoint**: If all rules show low effectiveness, the observation
metrics may be too noisy. Consider increasing the learning window or adding
smoothing to the effectiveness calculation.

---

### Phase F: Cross-Stack Validation (Week 7, parallel)

> **Goal**: Prove Go implementation matches TS reference spec.

| Task | LOC | Deliverable |
|------|-----|-------------|
| Export golden vectors from TS tests | 100 | JSON files with input/output pairs |
| Go test harness for golden vectors | 80 | Reads JSON, computes, asserts match |
| Test all 5 hormones | 50 | Matching within ε=1e-9 |
| Test adaptation rules | 50 | Matching multiplier deltas |
| Test damping + rate limiting | 30 | Matching bounds enforcement |

**Milestone F: "Go Matches TS Semantics"**
- [ ] All golden vector tests pass
- [ ] Hormone computation matches TS within ε=1e-9
- [ ] Adaptation deltas match TS within ε=1e-9

---

### Phase G: Morphogenic Processes (Weeks 8-10)

> **Goal**: Domain-level structural changes driven by hormones.
> This is the most speculative phase — proceed only if Phases C-E succeed.

| Task | LOC | Deliverable |
|------|-----|-------------|
| Domain Budding detection | 200 | VCF consistently high in fact sub-cluster |
| Domain Senescence detection | 150 | DVI negative for 10+ epochs |
| Cross-Domain Fusion suggestion | 200 | Citation patterns bridge two domains |
| Governance proposals (auto-generated) | 150 | LIP auto-submitted for domain changes |
| Tests | 150 | Morphogenic process tests |

**Milestone G: "The Knowledge Graph Restructures Itself"**
- [ ] Heavy cross-citation between physics and mathematics triggers fusion suggestion
- [ ] Domain with zero activity for 10 epochs triggers senescence warning
- [ ] Auto-generated LIP for domain change appears in governance queue

**Feedback checkpoint**: If morphogenic proposals are too noisy (frequent, low-quality),
increase the thresholds or require longer observation windows. If they never trigger,
the protocol may need more active domains — this is a testnet-era feature.

---

## Success Metrics by Phase

### Quantitative

| Metric | Phase A | Phase B | Phase C | Phase D | Phase E |
|--------|---------|---------|---------|---------|---------|
| Frozen params | 89 → 0 | 0 | 0 | 0 | 0 |
| Governance-mutable params | 0 → 89 | 89 | 89 | 89 | 89 |
| Observable hormones | 0 | 5 | 5 | 5 | 5 |
| Active hormones | 0 | 0 | 1 | 5 | 5 |
| Active multiplier paths | 0 | 0 | 3 | 11 | 11 |
| Consumer integration sites | 0 | 0 | 2 | 6 | 6 |
| Feedback loops | 0 | 0 | 1 | 5 | 5 + meta |
| Life Force status | N/A | Visible | Visible | Target: healthy | Target: thriving |
| Adaptation learning | No | No | No | No | Yes |

### Qualitative

| Phase | What Changes |
|-------|-------------|
| A | "The chain can be tuned without redeployment" |
| B | "We can see how the chain is doing in real time" |
| C | "The chain defends itself against stress" |
| D | "The chain nurtures knowledge domains that need help" |
| E | "The chain gets better at adapting over time" |
| G | "The knowledge graph restructures itself" |

---

## Risk Register

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Governance spam (too many param proposals) | Medium | Low | Proposal deposit requirement |
| Multiplier oscillation (feedback instability) | Medium | High | Quadratic damping, ±5% rate limit, SSI override |
| Observer returning garbage data | Low | High | Validate metrics bounds before computing hormones |
| Adaptation rule hurts instead of helps | Medium | Medium | Level 3 learning detects and dampens ineffective rules |
| Emergency during active autopoiesis | Low | High | Emergency freeze mechanism + skip missed epochs |
| Gas cost multiplier makes chain unusable | Low | Critical | gas.system bounded [0.5, 2.0] — max 2x increase |
| Block reward manipulation via SSI gaming | Medium | High | SSI uses 5-metric weighted composite — no single metric dominates |

---

## Relationship to Existing Roadmap

### What This Replaces

The original LAUNCH_ROADMAP.md focused on TS chain DevNet/Testnet/Mainnet.
Per the 2026-02-13 decision, all new work targets Go. This roadmap covers
the Go chain's evolution from static to adaptive.

### What This Complements

- **COSMOS_BRIDGE_ANALYSIS.md**: Module completion roadmap (Tiers 1-4)
- **MODULE_UPGRADABILITY_REVIEW.md**: Migration infrastructure (migrator scaffolding)
- **RECURSIVE_GROWTH_FRAMEWORK.md**: Architectural specification (3-layer model)
- **MODULE_UPGRADE_PLANS.md**: Per-module implementation details

### Integration Points

| Existing Roadmap Item | How Recursive Growth Integrates |
|----------------------|-------------------------------|
| Tier 1 (wire modules) | Phase A runs in parallel — parameterize as you wire |
| Tier 2 (complete modules) | Each module gets parameterized as part of completion |
| Tier 3 (new modules) | New modules follow the parameterized template from day 1 |
| Tier 4 (BVM, schedule, tree) | These complex modules get autopoiesis consumers last |
| DevNet testing | Phase B provides real-time health dashboard |
| Testnet launch | Phase C (single hormone) should be live before testnet |
| Security audit | Autopoiesis bounds and rate limits are audit targets |
| Mainnet launch | Phase D (full loop) should be live and validated before mainnet |

---

## Total Effort

| Phase | Duration | New LOC | Changed LOC | Tests |
|-------|----------|---------|-------------|-------|
| A: Parameterize | 2 weeks | 200 | 600 | 150 |
| B: Observation | 1 week | 1,200 | 50 | 200 |
| C: SSI Hormone | 1 week | 700 | 100 | 150 |
| D: Full Loop | 2 weeks | 800 | 200 | 350 |
| E: Meta-Adaptation | 1 week | 450 | 0 | 80 |
| F: Cross-Stack | 0.5 week | 300 | 0 | 300 |
| G: Morphogenic | 2-3 weeks | 850 | 0 | 150 |
| **Total** | **9.5-10.5 weeks** | **4,500** | **950** | **1,380** |

---

## Go/No-Go Decision Points

After each phase, evaluate before proceeding:

### Phase A → B
**Go if**: All 8 modules pass MsgUpdateParams integration tests.
**No-go if**: Params changes don't propagate to runtime behavior (pattern is wrong).

### Phase B → C
**Go if**: Observer produces meaningful non-zero metrics for all 5 hormones.
**No-go if**: More than 3 metrics are stubbed zeros (need more keeper accessors first).

### Phase C → D
**Go if**: SSI adaptation measurably improves the metric it targets over 10 epochs.
**No-go if**: Adaptation has no effect or worsens the target metric.

### Phase D → E
**Go if**: All 5 hormones converge in 100-epoch simulation without oscillation.
**No-go if**: Any hormone oscillates or any multiplier hits bounds repeatedly.

### Phase E → G
**Go if**: Learning correctly identifies effective vs ineffective rules.
**No-go if**: Effectiveness scores don't correlate with actual metric improvement.

---

_End of Integrated Growth Roadmap_
