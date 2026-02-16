# Recursive Growth Framework

> **Status**: DESIGN FRAMEWORK — approved architecture for making the Go chain adaptive.
> Implements the autopoietic principles from AUTOPOIESIS.md and RECURSIVE_INTEGRATION.md
> within Cosmos SDK patterns. All work targets Go/`legbled` only (per 2026-02-13 decision).

---

## Executive Summary

The Legible Money Go chain has **89 frozen sapwood parameters** across 24 modules.
7 of 8 audited modules cannot adapt post-genesis. The autopoiesis engine exists in
TypeScript (40-50% complete) but is absent from Go. The critical gap: multipliers
are computed but never consumed — the protocol observes itself but cannot act on
what it sees.

This framework specifies how to make the Go chain **self-regulating** in 4 phases,
using standard Cosmos SDK patterns (params stores, BeginBlocker, governance messages)
rather than custom infrastructure.

### Guiding Principle

```
effective_value = base_value × multiplier
```

Base values are set at genesis and live in params stores. Multipliers are computed
by the autopoiesis engine each epoch and stored in a dedicated KV store. Consumers
read `params × multiplier` at runtime. Governance can change base values OR
multiplier bounds. The engine adjusts multipliers within bounds automatically.

---

## Part I: The Three-Layer Architecture

### Layer 1: DNA (Immutable)

Constants that define the protocol's identity. These NEVER change, not even by
governance. Hardcoded in Go source.

```
Examples:
- Block time: 2521ms
- Finality depth: 19 blocks
- Token denomination: "ulgm"
- DID format: "did:lgm:{hex}"
- Epoch length: 1111 blocks
- Domain separation prefixes
- Cryptographic algorithm choices
```

**Count**: ~15 constants. Already correct — leave as-is.

### Layer 2: Genome (Governance-Mutable)

Parameters that define the protocol's current configuration. Set at genesis,
changeable by governance proposals. Live in module params stores.

```
Examples:
- Epistemic category profiles (decay rates, confidence ceilings, min verifiers)
- Validator tier configs (min stake, min verifications, accuracy thresholds)
- Gas costs per transaction type
- Reward/penalty rates
- Challenge windows
- Evidence type strengths
```

**Count**: ~89 parameters (the frozen sapwood). These must move from hardcoded
constants into params stores with working `MsgUpdateParams` handlers.

### Layer 3: Phenotype (Auto-Adaptive)

Multiplier overlays computed by the autopoiesis engine. Applied on top of Layer 2
at runtime. The engine adjusts these each epoch based on observed metrics.

```
Examples:
- decay.empirical.physics = 1.3  (facts in physics/empirical decay 30% slower)
- reward.mathematics = 0.8       (mathematics domain gets 20% less reward weight)
- slash.system = 1.5             (wrong verifications slashed 50% harder)
- gas.system = 0.9               (gas 10% cheaper — low utilization)
```

**Count**: ~20 multiplier paths (all defined in TS autopoiesis/types.ts).
Bounded by SAFETY_BOUNDS, rate-limited to ±5%/epoch.

### How They Compose

```
                    ┌──────────────────────────────────┐
                    │  EFFECTIVE RUNTIME VALUE          │
                    │  = Layer2.base × Layer3.mult      │
                    └──────────────────────────────────┘
                               ▲            ▲
                    ┌──────────┘            └──────────┐
                    │                                   │
          ┌─────────────────┐              ┌─────────────────────┐
          │  Layer 2: GENOME │              │  Layer 3: PHENOTYPE  │
          │  (Params Store)  │              │  (Multiplier Store)  │
          │                  │              │                      │
          │  Set by:         │              │  Set by:             │
          │  - Genesis       │              │  - Autopoiesis engine│
          │  - Governance    │              │  - BeginBlocker      │
          │                  │              │  - Bounded ±5%/epoch │
          └─────────────────┘              └─────────────────────┘
                    │                                   │
          ┌─────────────────┐              ┌─────────────────────┐
          │  Layer 1: DNA    │              │  SAFETY_BOUNDS       │
          │  (Hardcoded)     │              │  (Governance-mutable)│
          │  NEVER changes   │              │  Controls mult range │
          └─────────────────┘              └─────────────────────┘
```

---

## Part II: The x/autopoiesis Module

### Module Structure

```
x/autopoiesis/
├── types/
│   ├── types.go          # Core types: EpochObservation, HormoneValues, Adaptation
│   ├── multipliers.go    # MultiplierPath constants, MultiplierRecord
│   ├── bounds.go         # SafetyBounds, rate limits, thresholds
│   ├── params.go         # Module params (bounds, enabled flag, epoch length)
│   ├── expected_keepers.go  # Interfaces for cross-module observation
│   ├── codec.go          # Message registration, service descriptors
│   ├── query.go          # Query types and service descriptor
│   └── errors.go         # Module-specific errors
├── keeper/
│   ├── keeper.go         # Store access, multiplier CRUD, epoch storage
│   ├── observer.go       # collectEpochMetrics() — reads from other keepers
│   ├── hormones.go       # computeHormones() — pure computation
│   ├── adaptation.go     # applyAdaptations() — write multipliers
│   ├── grpc_query.go     # 6 queries (multiplier, hormones, observation, etc.)
│   ├── msg_server.go     # MsgUpdateParams, MsgActivate, MsgOverrideMultiplier
│   ├── migrator.go       # Schema migration
│   └── keeper_test.go    # Unit tests
└── module.go             # AppModule with BeginBlocker
```

### Keeper Interface

```go
type Keeper struct {
    storeKey     storetypes.StoreKey
    cdc          codec.Codec
    authority    string

    // Cross-module observation (read-only interfaces)
    knowledgeKeeper  types.KnowledgeKeeper   // facts, claims, domains
    stakingKeeper    types.StakingKeeper      // validators, tiers, accuracy
    billingKeeper    types.BillingKeeper      // gas metrics, mempool pressure
    tokensKeeper     types.TokensKeeper       // reward/slash totals
}
```

### Key Operations

```go
// Multiplier access — called by ALL consumer modules
func (k Keeper) GetMultiplier(ctx context.Context, path string) float64
func (k Keeper) SetMultiplier(ctx context.Context, path string, value float64, hormone string, epoch int64)

// Epoch cycle — called from BeginBlocker
func (k Keeper) RunEpochCycle(ctx context.Context, epoch int64) error

// Governance overrides
func (k Keeper) OverrideMultiplier(ctx context.Context, path string, value float64) error

// Activation
func (k Keeper) IsActive(ctx context.Context) bool
func (k Keeper) Activate(ctx context.Context) error
func (k Keeper) Deactivate(ctx context.Context) error
```

### BeginBlocker

```go
func (am AppModule) BeginBlock(ctx context.Context) error {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    blockHeight := sdkCtx.BlockHeight()

    // Check if we're at an epoch boundary
    epochLength := am.keeper.GetEpochLength(ctx)
    if blockHeight % epochLength != 0 {
        return nil
    }

    if !am.keeper.IsActive(ctx) {
        return nil
    }

    epoch := blockHeight / epochLength
    return am.keeper.RunEpochCycle(ctx, epoch)
}
```

### Consumer Pattern

Every module that uses adaptive parameters reads multipliers via a shared interface:

```go
// In x/knowledge/types/expected_keepers.go
type AutopoiesisKeeper interface {
    GetMultiplier(ctx context.Context, path string) float64
    IsActive(ctx context.Context) bool
}

// In x/knowledge/keeper/keeper.go
type Keeper struct {
    // ...existing fields...
    autoKeeper types.AutopoiesisKeeper  // nil-safe: returns 1.0 when nil
}

// In x/knowledge/keeper/confidence.go
func (k Keeper) GetEffectiveDecayHalfLife(ctx context.Context, domain, category string) int64 {
    base := k.GetParams(ctx).CategoryDecay[category].HalfLifeBlocks
    mult := 1.0
    if k.autoKeeper != nil {
        mult = k.autoKeeper.GetMultiplier(ctx, "decay."+category+"."+domain)
    }
    return int64(float64(base) * mult)
}
```

---

## Part III: The Parameterization Plan

### Step 1: Move Constants to Params Stores

For each of the 89 frozen sapwood parameters, the pattern is:

1. Add field to module's `Params` struct
2. Set default in `DefaultParams()`
3. Populate in `InitGenesis`
4. Read from store in keeper methods (replace hardcoded constant)
5. Wire `MsgUpdateParams` to actually call `SetParams()`
6. Add bounds validation in `ValidateBasic()`

### Priority Order (by downstream impact)

| Priority | Module | Parameters | Why First |
|----------|--------|-----------|-----------|
| P0 | x/knowledge | CategoryDecay, EvidenceTypeStrength, StrandConfidenceCeiling, ChallengeWindowProfile | Core PoT — everything depends on these |
| P1 | x/staking | TierConfigs (4 tiers × 12 fields = 48 values) | Validator economics |
| P2 | x/billing | GasCosts (38 tx types) | Transaction economics |
| P3 | x/vesting_rewards | RewardParams (block reward, halving, research tax) | Emission economics |
| P4 | x/emergency | Quorum thresholds, cooldowns, abuse limits | Safety parameters |
| P5 | x/gov | LIP thresholds, voting params, disbursement timelock | Governance itself |
| P6 | Remaining 18 modules | Various per-module params | Long tail |

### Step 2: Wire MsgUpdateParams

Each module's `MsgUpdateParams` handler must:

```go
func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
    // 1. Authority check (already present in most modules)
    if ms.Keeper.GetAuthority() != msg.Authority {
        return nil, govtypes.ErrInvalidSigner
    }

    // 2. Validate new params (MISSING in most modules)
    if err := msg.Params.Validate(); err != nil {
        return nil, err
    }

    // 3. Actually apply (MISSING in most modules)
    if err := ms.Keeper.SetParams(ctx, msg.Params); err != nil {
        return nil, err
    }

    // 4. Emit event (MISSING in all modules)
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    sdkCtx.EventManager().EmitEvent(sdk.NewEvent(
        "params_updated",
        sdk.NewAttribute("module", types.ModuleName),
        sdk.NewAttribute("authority", msg.Authority),
    ))

    return &types.MsgUpdateParamsResponse{}, nil
}
```

### Step 3: Replace Hardcoded Reads

Every site that reads a hardcoded constant must switch to params store + multiplier:

```
BEFORE:  halfLife := CATEGORY_DECAY["empirical"].HalfLifeBlocks  // hardcoded 1000000
AFTER:   halfLife := k.GetEffectiveDecayHalfLife(ctx, domain, "empirical")
         // reads params store (1000000) × multiplier (e.g. 1.3) = 1300000
```

---

## Part IV: The Five Hormones in Go

### Hormone 1: CRI (Confidence Retention Index) — AUXIN

**What it observes**: Per (domain, category), the ratio of current confidence to
verified confidence for all active facts.

**What it adjusts**:
- `decay.{category}.{domain}` — confidence decay rate multiplier
- `revbonus.{category}.{domain}` — reverification bonus multiplier

**Observation source**: `x/knowledge` keeper — iterate active facts per domain/category.

**Adaptation rules**:
- CRI > 0.8: increase decay mult by +5% (facts retaining well, slow decay)
- CRI < 0.4: decrease decay mult by -5% (facts decaying fast, speed up + incentivize reverif)

### Hormone 2: DVI (Domain Vitality Index) — CYTOKININ

**What it observes**: Per domain, `(facts_created - facts_expired) / total_active_facts`.

**What it adjusts**:
- `reward.{domain}` — domain reward weight multiplier
- `claimstake.{domain}` — claim stake multiplier

**Observation source**: `x/knowledge` keeper — count facts by domain and lifecycle state.

**Adaptation rules**:
- DVI > 0: healthy growth, no change
- DVI < 0 for 3+ epochs: increase reward weight (attract verifiers to declining domain)
- DVI < 0 for 3+ epochs: decrease claim stake (lower barrier to entry)

### Hormone 3: VCF (Verification Cascade Factor) — ETHYLENE

**What it observes**: Per domain, `new_citations / total_active_facts`.

**What it adjusts**:
- `citation.{domain}` — citation evidence weight multiplier
- `domainprior.{domain}` — domain confidence prior multiplier

**Observation source**: `x/knowledge` keeper — count citation edges created this epoch.

**Adaptation rules**:
- VCF > 0.5: active citation network, increase citation weight (reward interconnection)
- VCF < 0.1: isolated facts, decrease citation weight (don't penalize sparse domains)

### Hormone 4: SSI (System Stress Index) — ABA

**What it observes**: Weighted composite of system-wide health metrics.
- 30% avg validator accuracy
- 20% challenge success rate
- 20% expiration/creation ratio
- 15% validator participation
- 15% avg confidence decline

**What it adjusts**:
- `slash.system` — wrong verification slash multiplier
- `blockreward.system` — block reward multiplier
- `minverifiers.system` — minimum verifiers delta (additive)

**Observation source**: `x/knowledge` + `x/staking` keepers.

**Adaptation rules**:
- SSI > 0.8 (critical): tighten slashes +10%, reduce rewards -10%, increase minVerifiers +1
- SSI > 0.5 (moderate): tighten slashes +5%, no reward change
- SSI < 0.2 (healthy): relax slashes -3%, increase rewards +3%

### Hormone 5: GRM (Growth Rate Modulator) — GIBBERELLIN

**What it observes**: `verified_facts_created / max_possible_verifications`.

**What it adjusts**:
- `maxtx.system` — max transactions per block multiplier
- `gas.system` — gas cost multiplier

**Observation source**: `x/knowledge` keeper + block metrics.

**Adaptation rules**:
- GRM > 0.7 (high utilization): increase max tx +5%, increase gas +3%
- GRM < 0.3 (low utilization): decrease max tx -3%, decrease gas -5%

### Conflict Resolution

When hormones conflict (e.g., SSI wants to increase rewards, DVI wants to decrease):

```
Priority: SSI > CRI > (DVI + VCF averaged) > GRM
```

SSI (stress response) always wins. This matches biological ABA dominance —
the stress hormone overrides growth signals.

---

## Part V: Safety Architecture

### Bounds (Layer 1 — cannot be breached)

```go
var SafetyBounds = map[string]MultiplierBounds{
    "decay":              {Min: 0.25, Max: 4.0},
    "revbonus":           {Min: 0.5,  Max: 2.0},
    "reward":             {Min: 0.5,  Max: 2.0},
    "claimstake":         {Min: 0.5,  Max: 10.0},
    "citation":           {Min: 0.5,  Max: 2.0},
    "domainprior":        {Min: 0.5,  Max: 2.0},
    "slash":              {Min: 0.5,  Max: 3.0},
    "blockreward":        {Min: 0.25, Max: 4.0},
    "minverifiers":       {Min: -2,   Max: 5},
    "maxtx":              {Min: 0.5,  Max: 4.0},
    "gas":                {Min: 0.5,  Max: 2.0},
}
```

### Rate Limiting

- Default: ±5% per epoch (MAX_CHANGE_PER_EPOCH)
- SSI tightening: ±10% per epoch (emergency response is faster)
- SSI relaxation: ±3% per epoch (recovery is cautious)

### Quadratic Damping

```go
func computeDamping(currentValue, genesisValue float64) float64 {
    d := (currentValue - genesisValue) / genesisValue // normalized distance
    return 1.0 / (1.0 + d*d)                          // quadratic decay
}
```

As multipliers drift further from genesis (1.0), the damping factor approaches 0,
making further drift exponentially harder.

### Governance Override

Governance can:
1. Change base params via `MsgUpdateParams` (Layer 2)
2. Change safety bounds via `MsgUpdateAutopoiesisParams` (Layer 3 bounds)
3. Override a specific multiplier via `MsgOverrideMultiplier`
4. Activate/deactivate the engine via `MsgActivateAutopoiesis`

### Emergency Kill Switch

If `x/emergency` halts the chain, autopoiesis freezes — no multiplier changes
during emergency state. Upon resume, the engine skips the missed epochs
(does not retroactively compute).

---

## Part VI: Phased Implementation Roadmap

### Phase 0: Foundation — Parameterize the Genome (Weeks 1-2)

**Goal**: Move 89 frozen constants into working params stores.

**Tasks**:
1. x/knowledge params: CategoryDecay, EvidenceTypeStrength, StrandConfidenceCeiling, ChallengeWindowProfile
2. x/staking params: TierConfigs (all 4 tiers × 12 fields)
3. x/billing params: GasCosts (38 tx types)
4. Wire MsgUpdateParams in all 8 priority modules
5. Add Params.Validate() bounds checking
6. Replace all hardcoded constant reads with params store reads
7. Integration tests: governance proposal changes param, behavior changes

**Deliverable**: Any parameter can be changed by governance proposal. No code
changes needed to adjust protocol economics.

**Verification**: Submit governance proposal to change empirical decay half-life.
Observe that new facts decay at the updated rate.

### Phase 1: Observation Only (Week 3)

**Goal**: Create `x/autopoiesis` module that observes but does not act.

**Tasks**:
1. Scaffold module (types, keeper, module.go)
2. Implement `collectEpochMetrics()` — reads from x/knowledge + x/staking
3. Implement `computeHormones()` — pure computation from metrics
4. Store observations and hormones in KV store
5. Expose gRPC queries: GetObservation, GetHormones, GetLifeForce
6. BeginBlocker calls observer + hormones at epoch boundary
7. Dashboard: query hormones and display system health

**Deliverable**: Validators and operators can see real-time hormonal state of
the protocol. No parameters are changed. This is "read-only autopoiesis."

**Verification**: Query `/legible/autopoiesis/v1/hormones` and see computed
values for all 5 hormones across all active domains.

### Phase 2: Single Hormone — SSI (Week 4)

**Goal**: Activate one hormone (SSI) to prove the adaptation loop works end-to-end.

**Tasks**:
1. Implement `applyAdaptations()` for SSI only
2. Wire SSI multiplier writes: slash.system, blockreward.system, minverifiers.system
3. Wire consumer reads in x/knowledge (slash), x/vesting_rewards (block reward)
4. Add MsgActivateAutopoiesis governance message
5. Safety bounds enforcement + rate limiting
6. Deactivation mechanism (governance or emergency)

**Deliverable**: Protocol automatically adjusts slashing severity and block rewards
in response to system stress. Bounded to ±5%/epoch, max [0.5, 3.0].

**Verification**: Simulate high challenge success rate. Observe SSI rising.
Observe slash multiplier increasing. Governance deactivates. Multiplier freezes.

### Phase 3: Multi-Hormone (Weeks 5-6)

**Goal**: Activate remaining 4 hormones with conflict resolution.

**Tasks**:
1. CRI adaptation: decay and reverification multipliers per (domain, category)
2. DVI adaptation: domain reward weight and claim stake per domain
3. VCF adaptation: citation weight and domain prior per domain
4. GRM adaptation: max tx and gas multipliers system-wide
5. Conflict resolution: SSI > CRI > (DVI+VCF) > GRM
6. Quadratic damping
7. Comprehensive integration tests

**Deliverable**: Full 5-hormone autopoietic loop running at every epoch boundary.

**Verification**: Run 100-epoch simulation. Observe multipliers converging toward
stable values. Introduce stress (many failed verifications). Observe SSI dominance.
Remove stress. Observe gradual relaxation.

### Phase 4: Level 3 Recursion (Week 7)

**Goal**: The engine learns which adaptation rules work and adjusts its own rate.

**Tasks**:
1. Implement `computeAdaptationLearning()` — runs every 10 epochs
2. Track rule effectiveness: `applicationCount`, `improvementCount`
3. Adjust `adaptationRateMultiplier` based on effectiveness
4. Store learning records in KV store
5. Query: GetAdaptationLearning

**Deliverable**: The autopoiesis engine meta-adapts — effective rules gain influence,
ineffective rules are dampened.

**Verification**: After 20 epochs, query learning records. Observe that rules
which improved their target metrics have higher effectiveness scores.

### Phase 5: Morphogenic Processes (Weeks 8-10)

**Goal**: Domain-level structural changes driven by hormones.

**Tasks**:
1. Domain Budding: When VCF consistently high in a sub-cluster of facts,
   propose new sub-domain via governance
2. Domain Senescence: When DVI negative for 10+ epochs, flag domain for
   possible merger/archival
3. Cross-Domain Fusion: When citation patterns strongly connect two domains,
   suggest structural reorganization

**Deliverable**: The knowledge graph's structure itself adapts based on usage patterns.

**Verification**: Submit 50+ facts with heavy cross-citation between "physics"
and "mathematics." Observe VCF triggering a cross-domain fusion suggestion.

---

## Part VII: Module Wiring in app.go

```go
// In app.go — initialization order matters

// 1. Create autopoiesis keeper
app.AutopoiesisKeeper = autopoiesiskeeper.NewKeeper(
    appCodec,
    keys[autopoiesistypes.StoreKey],
    authtypes.NewModuleAddress(govtypes.ModuleName).String(),
    app.KnowledgeKeeper,
    app.LegibleStakingKeeper,
    app.BillingKeeper,
    app.TokensKeeper,
)

// 2. Wire autopoiesis into consumer modules
app.KnowledgeKeeper.SetAutopoiesisKeeper(app.AutopoiesisKeeper)
app.LegibleStakingKeeper.SetAutopoiesisKeeper(app.AutopoiesisKeeper)
app.BillingKeeper.SetAutopoiesisKeeper(app.AutopoiesisKeeper)
app.VestingRewardsKeeper.SetAutopoiesisKeeper(app.AutopoiesisKeeper)

// 3. Module manager ordering
// BeginBlocker: [...existing..., autopoiesistypes.ModuleName]  // runs BEFORE knowledge
// EndBlocker:   [autopoiesistypes.ModuleName, ...existing...]  // cleanup AFTER knowledge
// InitGenesis:  [...existing..., autopoiesistypes.ModuleName]  // init AFTER dependencies
```

### AutopoiesisKeeper Interface for Consumers

```go
// Minimal interface — consumers only need GetMultiplier
type AutopoiesisKeeper interface {
    GetMultiplier(ctx context.Context, path string) float64
    IsActive(ctx context.Context) bool
}
```

### Nil-Safety Pattern

```go
// Every consumer module uses nil-safe access:
func (k Keeper) getMultiplier(ctx context.Context, path string) float64 {
    if k.autoKeeper == nil {
        return 1.0  // no autopoiesis = no adjustment
    }
    return k.autoKeeper.GetMultiplier(ctx, path)
}
```

This ensures the chain works identically with or without autopoiesis activated.
The multiplier overlay is purely additive — removing it returns all values to
their genesis base.

---

## Part VIII: Cross-Stack Validation

Since the TS chain is the reference spec, Go implementation must match TS semantics:

### Golden Vector Tests

For each hormone:
1. Define a fixed `EpochObservation` JSON input
2. Compute expected `HormoneValues` in TS
3. Assert Go produces identical `HormoneValues`
4. Apply identical initial multiplier state
5. Run `applyAdaptations()` in both
6. Assert resulting multiplier deltas match within ε=1e-9

Test file: `tests/cross_stack/autopoiesis_golden_test.go`

Golden vectors derived from: `tests/unit/consensus/autopoiesis.test.ts`

---

## Part IX: Metrics and Observability

### gRPC Queries

| Query | Response | Purpose |
|-------|----------|---------|
| `GetMultiplier(path)` | `{value, hormone, epoch}` | Current multiplier for a path |
| `GetAllMultipliers()` | `[{path, value, hormone, epoch}]` | All active multipliers |
| `GetObservation(epoch)` | `EpochObservation` | Raw metrics for an epoch |
| `GetHormones(epoch)` | `HormoneValues` | Computed hormones for an epoch |
| `GetAdaptation(epoch)` | `EpochAdaptation` | All multiplier changes in an epoch |
| `GetLifeForce()` | `{value, status, components}` | Current system health |

### Events

```
autopoiesis.epoch_cycle      {epoch, life_force, ssi, grm}
autopoiesis.multiplier_set   {path, old, new, hormone, epoch}
autopoiesis.activated        {epoch, authority}
autopoiesis.deactivated      {epoch, authority}
autopoiesis.override         {path, value, authority}
```

### Telemetry

Prometheus metrics for monitoring:
- `legible_autopoiesis_life_force` (gauge) — current life force value
- `legible_autopoiesis_ssi` (gauge) — system stress index
- `legible_autopoiesis_multiplier{path}` (gauge) — per-path multiplier
- `legible_autopoiesis_epoch_cycle_ms` (histogram) — epoch cycle duration

---

## Part X: Relationship to Existing Work

### What We Keep

| Component | Source | Action |
|-----------|--------|--------|
| Hormone formulas | TS `consensus/autopoiesis/hormones.ts` | Port to Go |
| Adaptation rules | TS `consensus/autopoiesis/adaptation.ts` | Port to Go |
| Safety bounds | TS `consensus/autopoiesis/types.ts` | Port to Go |
| State tree API | TS `state/autopoiesis.ts` | Replace with KV store |
| Engine integration | TS `consensus/pot/engine.ts` | Replace with BeginBlocker |
| Type definitions | TS `consensus/autopoiesis/types.ts` | Port to Go structs |

### What We Replace

| TS Pattern | Go Pattern | Why |
|-----------|------------|-----|
| AutopoiesisStateTree (SMT) | KV store with prefix keys | Cosmos SDK standard |
| stateManager.getMultiplier() | keeper.GetMultiplier() | Keeper pattern |
| engine.runAutopoiesisEpochCycle() | BeginBlocker | Module lifecycle |
| Manual activation | MsgActivateAutopoiesis | Governance message |

### What We Add (Not in TS)

| Feature | Justification |
|---------|---------------|
| Params.Validate() | TS has no runtime bounds checking |
| gRPC queries | TS has no query server |
| Governance override | TS has no governance integration |
| Emergency freeze | TS emergency module is separate |
| Schema migration | Cosmos SDK standard |
| Prometheus metrics | Production observability |

---

## Appendix A: Full Multiplier Path Registry

```
# CRI (per domain × category)
decay.{category}.{domain}        # Confidence decay rate multiplier
revbonus.{category}.{domain}     # Reverification bonus multiplier

# DVI (per domain)
reward.{domain}                  # Domain reward weight multiplier
claimstake.{domain}              # Claim stake multiplier

# VCF (per domain)
citation.{domain}                # Citation evidence weight multiplier
domainprior.{domain}             # Domain confidence prior multiplier

# SSI (system-wide)
slash.system                     # Wrong verification slash multiplier
blockreward.system               # Block reward scaling multiplier
minverifiers.system              # Min verifiers additive delta

# GRM (system-wide)
maxtx.system                     # Max transactions per block multiplier
gas.system                       # Gas cost multiplier
```

## Appendix B: Consumer Integration Sites

These are the specific code locations where hardcoded constants must be replaced
with `params × multiplier` reads. Each site becomes a consumer of autopoiesis.

| Module | File | Current Constant | Multiplier Path |
|--------|------|-----------------|-----------------|
| x/knowledge | keeper/confidence.go | CATEGORY_DECAY map | decay.{cat}.{dom} |
| x/knowledge | keeper/confidence.go | reverificationBonus | revbonus.{cat}.{dom} |
| x/knowledge | keeper/rounds.go | minVerifiers | minverifiers.system |
| x/knowledge | keeper/msg_server.go | claimStakeMultiplier | claimstake.{dom} |
| x/knowledge | keeper/msg_server.go | evidenceTypeStrength | (via params) |
| x/staking | keeper/tiers.go | DefaultTierConfigs | (via params) |
| x/staking | keeper/keeper.go | epochLength 1111 | (DNA — no multiplier) |
| x/billing | keeper/gas.go | GasCosts map | gas.system |
| x/vesting_rewards | keeper/rewards.go | blockReward | blockreward.system |
| x/vesting_rewards | keeper/rewards.go | rewardMultiplier per tier | (via params) |
| x/knowledge | keeper/msg_server.go | citationEvidenceWeight | citation.{dom} |
| x/home | keeper/aging.go | BlocksPerDay thresholds | (via params) |

---

_End of Recursive Growth Framework_
