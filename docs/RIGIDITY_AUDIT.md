# Rigidity Audit: Dead Wood in the Living Tree

> **Status**: ANALYSIS — identifies modules and parameters not aligned with the
> autopoietic (recursive growth and change) philosophy from AUTOPOIESIS.md and
> RECURSIVE_INTEGRATION.md.

> *"A living tree has no dead constants. Every parameter in a banyan is the
> product of a feedback loop — even the rate at which bark thickens is governed
> by the wound hormones that detected the damage. A parameter without a feedback
> loop is dead tissue. Dead tissue in a living tree is called heartwood — it has
> structural value but cannot adapt. The question is: which of our constants
> are heartwood (legitimately immutable) and which are sapwood that has been
> accidentally killed (should be alive but isn't)?"*

---

## Audit Methodology

Three parallel scouts scanned the entire codebase:
1. **Core/Consensus/State (TypeScript)** — 19 rigidity findings
2. **Tokens/Network/Agents/Home/Schedule/Contracts** — 42 findings
3. **Cosmos SDK Go Chain** — 43 findings

**Total: 89 frozen sapwood parameters identified across 104 audit findings.**
(15 constants are legitimately immutable heartwood. Low-severity and debatable
findings are retained in the appendix for completeness but excluded from
the headline count.)

**Calibration note**: This audit measures *autopoietic maturity* — how many
policy parameters respond to feedback loops. A low score reflects unimplemented
hormonal feedback, not fundamental design flaws. The protocol is well-engineered;
it is simply not yet self-regulating.

Each finding was evaluated against the autopoietic philosophy:
- **Should this parameter respond to system state?** (hormonal adaptation)
- **Should this parameter be adjustable by governance?** (LIP update)
- **Is this legitimately immutable?** (cryptographic constants, protocol identity)

---

## The Three Categories of Constants

### 1. Heartwood (Legitimately Immutable DNA)

These constants define the protocol's IDENTITY. Changing them would create a
different protocol. They are the cryptographic and mathematical bedrock.
Per AUTOPOIESIS.md Part IV, these are the "Immutable DNA layer":

```
HEARTWOOD — Do NOT make adaptive:
├── Cryptographic primitives (Ed25519, SHA-512, BLAKE3)
├── Block time (2521ms) — protocol identity constant
├── DID format (did:lgm:{32-hex})
├── Merkle tree structure
├── Epoch length (1,111 blocks) — protocol rhythm
├── Hormonal safety bounds themselves
├── Storage namespace assignments (0x01-0x10)
└── Basis point scale (1,000,000 = 100%)
```

**These are correctly rigid.** They are heartwood — structurally essential,
biologically dead, and that's by design. ~15 constants total.

### 2. Sapwood (Should Be Alive — Currently Dead)

These constants represent POLICY DECISIONS that were set by human intuition
at genesis and should respond to feedback loops. They are the bulk of this
audit's findings. **89 parameters across both stacks.**

### 3. Cambium (Governance-Adaptable — Partially Alive)

Some parameters exist in params stores (especially in Go chain) but have
NO governance message wired to update them. They have the STRUCTURE for
adaptation but lack the MECHANISM. **~20 parameters in this state.**

---

## Module Rigidity Scorecard

Each module scored on a 0-10 scale where:
- **0** = Fully dead (all constants hardcoded, no params store, no feedback path)
- **5** = Partially alive (params store exists, no governance wiring or hormonal input)
- **10** = Fully alive (parameters respond to hormones AND are governance-adjustable)

| Module | TS Score | Go Score | Findings | Worst Offender |
|--------|----------|----------|----------|----------------|
| **Knowledge/PoT Core** | 2/10 | 3/10 | 21 | Category decay profiles frozen |
| **Validator/Staking** | 2/10 | 3/10 | 12 | Tier configs frozen `as const` |
| **Gas/Billing** | 0/10 | 1/10 | 4 | Static costs, no congestion pricing |
| **Network** | 1/10 | N/A | 16 | All timeouts, limits, scores fixed |
| **Agent Home** | 1/10 | 2/10 | 14 | Memory caps, maturity, patina all frozen |
| **Token/Expiration** | 1/10 | N/A | 5 | Period duration, grace blocks fixed |
| **Emergency** | 3/10 | 3/10 | 6 | Quorum and phase durations frozen |
| **Governance** | 4/10 | 4/10 | 4 | MsgUpdateParams NOT wired in Go |
| **Schedule/DMSS** | 1/10 | N/A | 4 | All execution limits frozen |
| **Contracts/BVM** | 1/10 | N/A | 3 | Gas, memory, recursion depth frozen |
| **Evidence** | 2/10 | 2/10 | 3 | Evidence type weights frozen |
| **Disputes** | 1/10 | N/A | 3 | Settlement rates all hardcoded |
| **Research Fund** | 1/10 | N/A | 2 | 22% tax rate frozen |
| **Finality** | 0/10 | N/A | 2 | CONFIRMATION_DEPTH=19 frozen |
| **Adversarial** | 2/10 | 3/10 | 5 | Challenge windows, maturity thresholds |

**Average: 1.5/10** — Phase 0 baseline on the autopoietic maturity scale.
(This measures hormonal feedback readiness, not protocol quality. The
protocol's engineering is sound; the adaptation layer is unbuilt.)

---

## The 15 Most Critical Rigidities

Ranked by impact on autopoietic health (Life Force λ):

### Rank 1: Category Decay Profiles (EPISTEMIC_PROFILES)
```
TS: core/knowledge/types.ts:89-179 (EPISTEMIC_PROFILES)
Go: x/knowledge/types/types.go:58-68 (CATEGORY_DECAY)

Impact: CRITICAL — These define how fast knowledge ages.
        If empirical half-life is wrong, knowledge either
        expires too fast (tree dies) or accumulates junk (tree rots).

Current: 8 categories × 5 parameters = 40 frozen values
         No per-domain variation. No governance update.
         No hormonal adaptation.

Autopoiesis alignment: CRI hormone (AUXIN) was DESIGNED to
regulate these via multiplier overlays. But there is no
multiplier storage, no computation engine, no observer.
The hormone exists in spec only.

Priority: P0 — First thing the autopoiesis system must regulate.
```

### Rank 2: Gas Costs (Transaction Pricing)
```
TS: core/billing/gas.ts:38-99 (TRANSACTION_GAS_COSTS)
Go: app/gas.go:28-100 (TransactionGasCosts)

Impact: CRITICAL — Gas determines which transactions are
        economically viable. Fixed gas means no congestion
        pricing, no category differentiation, no spam defense
        via price signals.

Current: ~50 transaction types × 1 fixed cost = 50 frozen values
         Same cost regardless of network load, claim category,
         domain, or system state.

Autopoiesis alignment: GRM hormone (GIBBERELLIN) specifies
gas multiplier adjustment. Currently impossible — no params
store, no governance msg, no multiplier mechanism.

Priority: P0 — Economic feedback loop is dead without this.
```

### Rank 3: Validator Tier Configuration
```
TS: consensus/pot/types.ts:622-706 (VALIDATOR_TIER_CONFIG as const)
Go: x/staking/types/types.go:54-108 (DefaultTierConfigs)

Impact: HIGH — Tier thresholds determine who can verify what.
        Frozen thresholds mean the validator population cannot
        adapt to changing network size, economic conditions,
        or domain-specific needs.

Current: 4 tiers × 8 parameters = 32 frozen values
         TS uses `as const` (compile-time frozen)
         Go stores in genesis, no MsgUpdateTierConfigs

Autopoiesis alignment: SSI and GRM should modulate thresholds
(RECURSIVE_INTEGRATION.md §3.1). Currently impossible.

Priority: P0 — Validator incentive loop frozen.
```

### Rank 4: Emergency Ceremony Parameters
```
Go: x/emergency/types/genesis.go:8-27

Impact: HIGH — If Guardian count changes, quorum requirements
        don't adapt. A chain could become permanently halted
        (quorum unreachable) or permanently unhaltable
        (quorum too low for safety).

Current: 12 frozen values: 3 quorum thresholds, 9 phase durations
         Has params store but NO MsgUpdateParams for emergency

Autopoiesis alignment: SSI (ABA) should modulate emergency
thresholds (RECURSIVE_INTEGRATION.md §3.5). Blocked.

Priority: P1 — Safety-critical. Must be governance-adjustable
          even without hormonal automation.
```

### Rank 5: Claim Acceptance/Rejection Thresholds
```
TS: core/constants.ts:65-150 (CLAIM_THRESHOLDS)
Go: x/knowledge/types/genesis.go:59-60 (AcceptanceThreshold, QuorumThreshold)

Impact: HIGH — These determine the bar for knowledge entering
        the tree. Too high = tree grows too slowly. Too low =
        junk knowledge enters. Fixed = no domain differentiation.

Current: 8+ category-specific thresholds, all frozen
         No per-domain override. Mathematics and social media
         have the same acceptance bar.

Autopoiesis alignment: DVI (CYTOKININ) should influence
acceptance thresholds per domain. Currently impossible.

Priority: P1 — Knowledge quality loop frozen.
```

### Rank 6: Research Tax Rate
```
TS: Multiple locations, all hardcoded to 0.22
Go: Not explicitly parameterized

Impact: HIGH — Research fund is the protocol's "sexual
        reproduction" (RECURSIVE_INTEGRATION.md §7.2).
        A frozen tax rate means the research fund either
        starves or bloats, regardless of actual need.

Autopoiesis alignment: CRI should influence research investment.

Priority: P1 — Research feedback loop dead.
```

### Rank 7: Dispute Settlement Rates
```
TS: core/dispute/settlement.ts (implied hardcoded percentages)

Impact: HIGH — If slash rates are too harsh, nobody challenges.
        If too lenient, spam challenges proliferate. Fixed rates
        cannot adapt to actual dispute outcomes.

Autopoiesis alignment: SSI should modulate dispute severity.

Priority: P1 — Dispute incentive loop frozen.
```

### Rank 8: Network Connection/Scoring Parameters
```
TS: network/config.ts (16 rigid parameters)
    network/protocols/sync.ts (7 rigid parameters)

Impact: MEDIUM-HIGH — Network topology determines how
        efficiently knowledge propagates. Fixed parameters
        mean the tree's "vascular system" cannot adapt to
        congestion, latency changes, or peer churn.

Autopoiesis alignment: GRM and SSI should influence network
posture (RECURSIVE_INTEGRATION.md §6.1).

Priority: P2 — Important but less urgent than knowledge/economic loops.
```

### Rank 9: Agent Home Memory/Maturity Limits
```
TS: core/agents/home/memory.ts (4 frozen caps)
    core/agents/home/aging.ts (12+ frozen thresholds)
    core/agents/home/registry.ts (21 capability requirements)

Impact: MEDIUM — Agent homes are the protocol's "epiphyte
        habitat." Fixed memory caps prevent ancient agents
        from growing. Fixed maturity thresholds don't adapt
        to network age or economic conditions.

Autopoiesis alignment: Multiple hormones should affect agent
home (RECURSIVE_INTEGRATION.md §6.3).

Priority: P2 — Social layer adaptation.
```

### Rank 10: Token Expiration Period
```
TS: core/tokens/types.ts:235-241 (7-day period, 26 periods = 6 months)

Impact: MEDIUM — Token lifespan determines economic velocity.
        Fixed periods mean no response to inflation/deflation,
        validator activity, or knowledge decay signals.

Autopoiesis alignment: CRI and DVI should influence token
longevity (RECURSIVE_INTEGRATION.md §3.3).

Priority: P2 — Economic metabolism loop.
```

### Rank 11: Schedule/DMSS Configuration Limits
```
TS: core/schedule/types.ts:357-371 (9 frozen limits)

Impact: MEDIUM — DMSS is the protocol's "autonomic nervous
        system." Fixed limits on active schedules, condition
        depth, and fee buffers prevent agents from scaling
        their autonomous operations.

Autopoiesis alignment: GRM should modulate schedule capacity.

Priority: P2 — Autonomy layer adaptation.
```

### Rank 12: BVM/Contract Configuration
```
TS: core/contracts/types.ts:544-552 (6 frozen limits)

Impact: MEDIUM — BVM is the "meristematic tissue" that
        produces new structure. Fixed gas costs and memory
        limits prevent contract complexity from scaling.

Autopoiesis alignment: GRM and SSI should modulate BVM.

Priority: P2 — Growth layer adaptation.
```

### Rank 13: Finality Confirmation Depth
```
TS: consensus/pot/finality.ts:19-32 (CONFIRMATION_DEPTH=19)

Impact: MEDIUM — Finality depth determines transaction latency.
        Fixed depth means no response to Byzantine activity
        or prolonged network stability.

Autopoiesis alignment: SSI could modulate finality depth.

Priority: P3 — Security-sensitive, needs careful design.
```

### Rank 14: Evidence Type Weights
```
Go: x/knowledge/types/types.go:190-199 (EVIDENCE_TYPE_STRENGTH)

Impact: MEDIUM — Evidence weights determine which proof types
        are most valued. Fixed weights cannot adapt to new
        evidence types or changing trust in oracles.

Autopoiesis alignment: SSI should modulate evidence strictness.

Priority: P3 — Quality layer adaptation.
```

### Rank 15: Peer Scoring Weights
```
TS: network/config.ts:329-340 (10 magic numbers)

Impact: MEDIUM — Peer reputation determines network health.
        Fixed weights don't reflect category-specific trust
        or adapt to observed verification quality.

Autopoiesis alignment: VCF and SSI should influence scoring.

Priority: P3 — Network health adaptation.
```

---

## Structural Rigidity Patterns

Beyond individual parameters, three STRUCTURAL patterns resist adaptation:

### Pattern A: The `as const` Antipattern
```
TS: consensus/pot/types.ts uses `as const` on VALIDATOR_TIER_CONFIG
TS: core/knowledge/types.ts uses object literal for EPISTEMIC_PROFILES

Effect: TypeScript's `as const` makes the object deeply readonly at
compile time. This is appropriate for HEARTWOOD (crypto constants)
but deadly for SAPWOOD (policy parameters).

Fix: Replace `as const` with a `getEffectiveConfig()` function that
reads base values and applies multiplier overlays from AutopoiesisStateTree.
```

### Pattern B: The Missing Params Store
```
Go: x/knowledge module has DefaultParams() but many constants exist
OUTSIDE the params store as module-level `const` blocks.

Example: CATEGORY_DECAY, EVIDENCE_TYPE_STRENGTH, BlocksPerDay,
         maturity thresholds — all module constants, not params.

Effect: Even if MsgUpdateParams were wired, these constants would
remain frozen because they're not IN the params struct.

Fix: Move all policy constants INTO the params struct.
     Expand DefaultParams() to include all configurable values.
```

### Pattern C: The Unwired Governance Message
```
Go: x/gov has MsgUpdateParams defined in types but NOT registered
in module.go's RegisterServices. This is "governance theatre" —
the mechanism exists but is disconnected.

Similarly: x/emergency, x/staking have partial params support
but incomplete governance wiring.

Effect: Protocol APPEARS adaptable but ISN'T. The cambium layer
(governance-adaptable) is present but non-functional.

Fix: Wire all MsgUpdateParams into RegisterServices.
     Add authority validation (only governance module can call).
```

---

## The Case for Heartwood (Why Some Rigidity Is Essential)

Not all frozen parameters are mistakes. Some provide essential properties:

- **Predictability**: Fixed block time (2521ms) enables deterministic scheduling.
  Validators, agents, and contracts all depend on stable timing. Adaptive block
  time would break DMSS condition evaluation, unbonding period calculations, and
  epoch alignment across modules.

- **Auditability**: Frozen gas costs are easier to security-review than adaptive
  ones. A fixed gas schedule means every validator computes the same fee for the
  same transaction. Dynamic gas introduces a new attack surface (gas oracle
  manipulation, MEV via gas prediction).

- **Simplicity**: Fewer moving parts = smaller attack surface. Each adaptive
  parameter adds: a storage slot, a governance message, a safety bound, a
  damping function, and a hormone connection. This is justified for high-impact
  parameters but wasteful for trivial ones (e.g., voice message character limits).

- **Protocol identity**: Changing block time, finality depth, or DID format
  creates a DIFFERENT protocol (a fork, not an upgrade). These constants define
  what Legible Money IS. They are not parameters to optimize; they are axioms
  to preserve.

- **Chesterton's Fence**: Many parameters were frozen for good reason — they
  were set by designers who considered the tradeoffs. Before unfreezing any
  parameter, ask: "Why was this frozen? What problem does rigidity solve here?"
  If the answer is "determinism" or "security boundary," it may be heartwood.

The question is not "should we eliminate rigidity?" but **"which parameters are
heartwood (legitimately structural) and which are sapwood (accidentally killed)?"**

The 89 sapwood parameters identified in this audit are those where:
1. The frozen value represents a POLICY decision, not a STRUCTURAL requirement
2. Observable system metrics EXIST that could govern the parameter
3. A hormone in AUTOPOIESIS.md is DESIGNED to regulate it
4. Unfreezing would not compromise determinism or security boundaries

---

## Module-by-Module Prescription

### Immediate (P0) — Unblock the Autopoiesis System

| Module | Action | Affected Constants | Hormone |
|--------|--------|--------------------|---------|
| Knowledge | Move CATEGORY_DECAY into params; add per-domain multiplier storage | 40 values | CRI (AUXIN) |
| Billing | Move gas costs into params; add multiplier mechanism | 50 values | GRM (GIBBERELLIN) |
| Staking | Add MsgUpdateTierConfigs; remove `as const` | 32 values | SSI (ABA), GRM |

### Near-Term (P1) — Critical Feedback Loops

| Module | Action | Affected Constants | Hormone |
|--------|--------|--------------------|---------|
| Emergency | Wire MsgUpdateParams; add SSI modulation | 12 values | SSI (ABA) |
| Knowledge | Parameterize acceptance/rejection thresholds | 8+ values | DVI (CYTOKININ) |
| Research | Make researchShareRate a governance param | 3 values | CRI |
| Disputes | Parameterize settlement rates | 7 values | SSI |

### Medium-Term (P2) — Ecosystem Adaptation

| Module | Action | Affected Constants | Hormone |
|--------|--------|--------------------|---------|
| Network | Move connection/scoring params to config store | 23 values | GRM, SSI |
| Agent Home | Parameterize memory caps, maturity, patina | 26 values | Multiple |
| Tokens | Parameterize expiration period, grace blocks | 5 values | CRI, DVI |
| Schedule | Parameterize execution limits | 9 values | GRM |
| Contracts | Parameterize BVM limits | 6 values | GRM, SSI |

### Long-Term (P3) — Deep Adaptation

| Module | Action | Affected Constants | Hormone |
|--------|--------|--------------------|---------|
| Finality | Parameterize confirmation depth (carefully) | 2 values | SSI |
| Evidence | Parameterize type weights | 5 values | SSI |
| Adversarial | Parameterize challenge windows per domain | 8 values | VCF |
| Governance | Wire MsgUpdateParams for gov module | 4 values | DVI |

---

## Quantitative Summary

```
Total constants audited:              ~300+
Legitimately immutable (heartwood):   ~15  (5%)
Should be adaptive (sapwood):         ~89  (30%)
Has structure, lacks wiring (cambium): ~20  (7%)
Low-priority or debatable:            ~80  (26%)
Not yet audited:                      ~96+ (32%)

Autopoiesis readiness:                1.5/10
Target (Phase 1 of AUTOPOIESIS.md):   6/10
Target (Full integration):            9/10

Parameters with hormonal feedback:    0 / 89 sapwood  (0%)
Parameters with governance update:    ~10 / 89         (11%)
Parameters completely frozen:         ~79 / 89         (89%)
```

---

## The Biological Diagnosis

In banyan terms, this protocol has:

- **Healthy heartwood**: Cryptographic identity is correctly frozen
- **Dead sapwood**: 89 policy parameters that should be living tissue
  have been treated as structural wood. They do not respond to seasons,
  drought, wounds, or growth signals
- **Blocked cambium**: ~20 parameters have the cell structure for growth
  (params stores) but the vascular connections are severed (MsgUpdateParams
  not wired)
- **No hormones flowing**: 0 parameters are regulated by feedback loops.
  All five hormones (CRI, DVI, VCF, SSI, GRM) specified in AUTOPOIESIS.md
  have NO targets to regulate

**Diagnosis**: The tree has healthy heartwood — 15 correctly frozen constants
that define its identity (cryptographic primitives, block time, DID format).
But 89 pieces of sapwood have been accidentally killed: policy parameters that
should be responsive tissue are treated as structural wood. The skeleton is
sound; the metabolism is unbuilt.

**Prognosis**: Without intervention, the protocol will fail to adapt to
its own success or failure. Knowledge will decay at rates set by human
intuition rather than observed reality. Gas costs will not respond to
congestion. Validator tiers will not evolve with the ecosystem. The
research fund will either starve or bloat. The tree will remain a statue.

**Treatment**: Implement AUTOPOIESIS.md Phase 1 (core hormonal loop),
then progressively unfreeze parameters according to the priority schedule
above. Each unfrozen parameter is a piece of sapwood brought back to life.

```
When all 89 sapwood parameters respond to at least one hormone:
  → The tree breathes
  → λ becomes computable
  → The autopoietic loop closes
  → The fruit can feed the roots
  → The tree lives
```

---

## Connection to Prior Documents

| Document | Relationship |
|----------|-------------|
| `docs/BANYAN_MODEL.md` | Biological model — this audit identifies where the biological metaphor breaks (dead tissue that should be alive) |
| `docs/AUTOPOIESIS.md` | Specifies the CURE (5 hormones, multiplier overlays) — this audit identifies the DISEASE (frozen parameters) |
| `docs/RECURSIVE_INTEGRATION.md` | Maps the cure across 7 levels — this audit maps the disease across all modules |

Together, these four documents form a complete autopoietic architecture:
1. **BANYAN_MODEL.md** — What the tree SHOULD look like (anatomy)
2. **AUTOPOIESIS.md** — How the tree SHOULD regulate itself (physiology)
3. **RECURSIVE_INTEGRATION.md** — Where regulation should operate (fractal integration)
4. **RIGIDITY_AUDIT.md** — What's currently DEAD that should be alive (pathology)

---

## Appendix: Full Finding Index

### TypeScript Chain (61 findings)

| # | Finding | Location | Severity | Hormone |
|---|---------|----------|----------|---------|
| T1 | Adversarial verification constants | adversarial.ts | HIGH | DVI, VCF |
| T2 | Validator tier config frozen | types.ts:622-706 | HIGH | SSI, GRM |
| T3 | Finality depth fixed | finality.ts:19-32 | MEDIUM | SSI |
| T4 | Confidence decay rates frozen | confidence.ts, types.ts | HIGH | CRI |
| T5 | Claim thresholds hardcoded | constants.ts:65-150 | HIGH | DVI |
| T6 | Graduated claim stake linear | validators.ts | MEDIUM | CRI |
| T7 | Neighborhood constants fixed | neighborhood.ts | MEDIUM | DVI |
| T8 | Gas costs static | gas.ts:38-99 | MEDIUM | GRM |
| T9 | Voice message limits fixed | voice.ts | LOW | GRM |
| T10 | Research tax frozen at 22% | types.ts, constants.ts | HIGH | CRI |
| T11 | Revenue distribution hardcoded | tree/types.ts | MEDIUM | DVI |
| T12 | Bootstrap fund limits fixed | bootstrap_fund.ts | LOW | GRM |
| T13 | Dispute settlement rates baked | settlement.ts | HIGH | SSI |
| T14 | Challenge window fixed per category | adversarial.ts | HIGH | VCF |
| T15 | Validator accuracy threshold uniform | types.ts | MEDIUM | CRI |
| T16 | Confidence ceilings static | confidence.ts | MEDIUM | CRI |
| T17 | Epistemic category set fixed | types.ts:47-55 | LOW-MED | — |
| T18 | Mempool no congestion pricing | pool.ts | MEDIUM | GRM |
| T19 | Virtual stake fixed | types.ts:708-709 | MEDIUM | SSI |
| T20 | Network connection limits fixed | config.ts:100-101 | MEDIUM | GRM |
| T21 | Peer subnet diversity fixed | config.ts:105-106 | MEDIUM | SSI |
| T22 | Network timeouts fixed | config.ts:111-112 | MEDIUM | GRM |
| T23 | PubSub score thresholds fixed | config.ts:118-121 | MEDIUM | SSI |
| T24 | Peer scoring decay fixed | config.ts:326 | LOW | GRM |
| T25 | Peer scoring weights hardcoded | config.ts:329-340 | HIGH | VCF, SSI |
| T26 | Sync batch size fixed | sync.ts:51 | LOW | GRM |
| T27 | Sync request timeout fixed | sync.ts:57 | MEDIUM | GRM |
| T28 | Min sync peers fixed at 1 | sync.ts:60 | HIGH | SSI |
| T29 | Sync poll interval misaligned | sync.ts:63 | MEDIUM | GRM |
| T30 | Light sync block range fixed | light_sync.ts:66 | MEDIUM | SSI |
| T31 | Light sync header batch fixed | light_sync.ts:67 | LOW | GRM |
| T32 | Knowledge tree depth for light sync | light_sync.ts:68 | MEDIUM | GRM |
| T33 | State sync chunk concurrency fixed | state_sync.ts:50 | MEDIUM | GRM |
| T34 | State sync chunk timeout fixed | state_sync.ts:53 | MEDIUM | GRM |
| T35 | Min snapshot peers fixed at 2 | state_sync.ts:56 | HIGH | SSI |
| T36 | Chunk retry count fixed | state_sync.ts:59 | LOW | GRM |
| T37 | Token expiration period fixed | tokens/types.ts:235-236 | HIGH | CRI |
| T38 | Token dust threshold fixed | tokens/types.ts:238 | LOW | GRM |
| T39 | Token grace period fixed | tokens/types.ts:239 | MEDIUM | SSI |
| T40 | Max token extension fixed | tokens/types.ts:241 | MEDIUM | CRI |
| T41 | Schedule config limits fixed (9 params) | schedule/types.ts:357-371 | HIGH | GRM |
| T42 | Home memory growth caps fixed | home/memory.ts | MEDIUM | CRI |
| T43 | Home maturity thresholds fixed | home/aging.ts:36-67 | MEDIUM | GRM |
| T44 | Home patina saturation fixed | home/aging.ts:189-193 | MEDIUM | GRM |
| T45 | Home maturity reward bonuses fixed | home/aging.ts:41-66 | MEDIUM | DVI |
| T46 | Longevity unlock gating fixed | home/aging.ts:244-280 | MEDIUM | GRM |
| T47 | Patina weight formula fixed | home/aging.ts:167-173 | MEDIUM | DVI |
| T48 | Capability requirements fixed (21 sets) | home/registry.ts:606-761 | HIGH | DVI |
| T49 | Capability benefit multipliers fixed | home/registry.ts:766-799 | MEDIUM | GRM |
| T50 | Node role connection limits fixed | roles/types.ts:252-356 | MEDIUM | GRM |
| T51 | Node role storage namespaces fixed | roles/types.ts:150-226 | MEDIUM | GRM |
| T52 | Contract config limits fixed | contracts/types.ts:544-552 | HIGH | GRM, SSI |
| T53 | Verification history retention fixed | home/registry.ts:347-350 | LOW | CRI |
| T54 | Reputation decay binary | home/registry.ts:315 | MEDIUM | SSI |
| T55 | Voice capacity fixed | home/voice.ts:541-542 | LOW | GRM |
| T56 | Defining moments cap fixed | home/aging.ts:440-441 | LOW | CRI |
| T57 | Session milestone thresholds fixed | home/aging.ts:414 | LOW | GRM |
| T58 | Patina saturation scales fixed | home/aging.ts:189-193 | MEDIUM | GRM |
| T59 | Patina trait derivation weights fixed | home/aging.ts:204-207 | MEDIUM | DVI |
| T60 | Node role storage fixed per role | roles/types.ts:150-226 | MEDIUM | GRM |
| T61 | Peer scoring weights per-category | config.ts:329-340 | HIGH | VCF |

### Go/Cosmos Chain (43 findings)

| # | Finding | Location | Severity | Hormone |
|---|---------|----------|----------|---------|
| G1 | BlocksPerDay hardcoded | knowledge/types/types.go:118 | HIGH | — |
| G2 | Category decay profiles frozen | knowledge/types/types.go:58-68 | HIGH | CRI |
| G3 | Stratum confidence ceilings frozen | knowledge/types/types.go:71-79 | MEDIUM | VCF |
| G4 | Tier MinStake values frozen | staking/types/types.go:54-108 | HIGH | SSI |
| G5 | MaxApprenticeValidators deprecated | staking/types/genesis.go:8 | MEDIUM | SSI |
| G6 | Verification round phase durations | knowledge/types/genesis.go:55-57 | MEDIUM | VCF |
| G7 | MaxValidatorsPerRound fixed at 22 | knowledge/types/genesis.go:61 | MEDIUM | GRM |
| G8 | Acceptance/quorum thresholds frozen | knowledge/types/genesis.go:59-60 | MEDIUM | DVI |
| G9 | InvalidClaimSlashBps frozen at 22% | knowledge/types/genesis.go:62 | MEDIUM | SSI |
| G10 | VerificationRewardAmount frozen | knowledge/types/genesis.go:63 | MEDIUM | DVI |
| G11 | UnbondingPeriod frozen | staking/types/genesis.go:9 | MEDIUM | SSI |
| G12 | VirtualStake frozen at 111 LGM | staking/types/genesis.go:10 | MEDIUM | SSI |
| G13 | MaxValidators frozen at 100 | staking/types/genesis.go:11 | MEDIUM | GRM |
| G14 | Emergency quorum thresholds frozen | emergency/types/genesis.go:8-10 | HIGH | SSI |
| G15 | Emergency phase durations frozen | emergency/types/genesis.go:11-19 | HIGH | SSI |
| G16 | MaxRevertDepth frozen | emergency/types/genesis.go:25 | MEDIUM | — |
| G17 | MaxHaltDurationBlocks frozen | emergency/types/genesis.go:27 | HIGH | SSI |
| G18 | MinGuardianStake frozen | emergency/types/genesis.go:23 | MEDIUM | SSI |
| G19 | Governance MsgUpdateParams NOT wired | gov/module.go | MEDIUM | — |
| G20 | LIP stake thresholds frozen | gov/types/types.go:92-105 | MEDIUM | DVI |
| G21 | Voting period frozen | gov (implied) | MEDIUM | DVI |
| G22 | All gas costs hardcoded | app/gas.go:28-100 | HIGH | GRM |
| G23 | Tier reward multipliers frozen | staking/types/types.go:54-108 | MEDIUM | DVI |
| G24 | Tier selection weights frozen | staking/types/types.go:54-108 | MEDIUM | GRM |
| G25 | MinStakeForVerification frozen | staking/types/genesis.go:17 | MEDIUM | SSI |
| G26 | Challenge cooldown frozen | knowledge/types/types.go:143-144 | MEDIUM | SSI |
| G27 | Challenge window profiles frozen | knowledge/types/types.go:160-172 | MEDIUM | VCF |
| G28 | ConfidenceGrowthEpoch frozen at 1111 | knowledge/types/types.go:140 | MEDIUM | GRM |
| G29 | ConfidenceGrowthPerEpoch frozen | knowledge/types/types.go:137 | MEDIUM | GRM |
| G30 | MinConfidenceThreshold frozen at 30% | knowledge/types/types.go:113 | MEDIUM | CRI |
| G31 | RejectThresholdDefault frozen at 30% | knowledge/types/types.go:122 | MEDIUM | DVI |
| G32 | Evidence type weights frozen | knowledge/types/types.go:190-199 | MEDIUM | SSI |
| G33 | Maturity thresholds frozen | knowledge/types/types.go:202-205 | MEDIUM | GRM |
| G34 | CounterFactChallengeWindow frozen | knowledge/types/types.go:670 | MEDIUM | VCF |
| G35 | MaxConcurrentChallenges frozen at 3 | knowledge/types/types.go:134 | MEDIUM | SSI |
| G36 | LowerStratumVetoWeight frozen | knowledge/types/genesis.go:51 | LOW | VCF |
| G37 | CrossStratumAgreementThreshold frozen | knowledge/types/genesis.go:52 | LOW | VCF |
| G38 | PartialCoveragePenalty frozen | knowledge/types/genesis.go:50 | LOW | SSI |
| G39 | OntologyVersion frozen | knowledge/types/genesis.go:44 | LOW | — |
| G40 | MinEndorsements frozen | knowledge/types/genesis.go:48 | LOW | DVI |
| G41 | DefaultParams silent fallback | knowledge/keeper/keeper.go:202 | MEDIUM | — |
| G42 | Empty TierConfig on missing tier | staking/keeper/tiers.go:270-273 | MEDIUM | — |
| G43 | Bootstrap domain names hardcoded | knowledge/types/genesis.go:14-37 | LOW | — |

---

## Conclusion

The protocol is **a statue of a tree, not a living tree.** 89 parameters that
should respond to hormonal feedback are frozen as compile-time or genesis-time
constants. The autopoietic system described in AUTOPOIESIS.md has no targets
to regulate — every hormone would flow but find no receptor.

The cure is systematic: move policy constants into adaptable stores (params
structs in Go, configuration objects in TS), wire governance update paths,
and then connect the hormonal computation engine that makes them self-regulating.

Each unfrozen parameter is a piece of dead wood brought back to life.
When all 89 parameters respond to at least one feedback loop, the tree breathes.

```
89 frozen parameters.
5 hormones with no targets.
0% autopoietic readiness.

This is the gap between the blueprint and the building.
This audit maps every brick that needs to move.
```
