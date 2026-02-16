# Per-Module Upgrade Plans

> **Status**: Actionable upgrade plans for each Go module.
> Companion to RECURSIVE_GROWTH_FRAMEWORK.md.
> All work targets Go/`legbled` only (per 2026-02-13 decision).

---

## Module Priority Map

```
P0 (Week 1)    P1 (Week 2)          P2 (Week 3)           P3 (Week 4+)
============   ============         ============          =============
x/knowledge    x/staking            x/emergency           x/home
x/billing      x/vesting_rewards    x/tokens              Remaining 16 modules
x/gov          x/autopoiesis (new)
```

P0 modules are already parameterized — need constant migration + consumer wiring.
P1 modules need structural refactoring to become parameterized.
P2 modules need missing handlers or consolidation.
P3 modules have fewer frozen constants.

---

## P0-A: x/knowledge — The Central Hub

### Current State
- 60+ hardcoded constants in `types/types.go`
- Params struct exists (proto-generated)
- SetParams/GetParams exists
- MsgUpdateParams handler exists and calls SetParams
- **Gap**: 20+ critical constants live outside the Params struct

### Constants to Parameterize

**Group 1: Category Decay Profiles (lines 55-68)**
```go
// MOVE from hardcoded map to Params.CategoryDecayProfiles
CATEGORY_DECAY = map[string]CategoryDecayProfile{
    "analytic":   {HalfLifeBlocks: 0, MaxConfidence: 1_000_000},
    "empirical":  {HalfLifeBlocks: 100_000, MaxConfidence: 990_000},
    "historical": {HalfLifeBlocks: 500_000, MaxConfidence: 950_000},
    "predictive": {HalfLifeBlocks: 20_000, MaxConfidence: 900_000},
    "social":     {HalfLifeBlocks: 150_000, MaxConfidence: 850_000},
    "replicated": {HalfLifeBlocks: 1_000_000, MaxConfidence: 995_000},
}
```

**Group 2: Evidence Type Strengths (lines 189-197)**
```go
// MOVE from hardcoded map to Params.EvidenceTypeStrengths
EVIDENCE_TYPE_STRENGTH = map[string]int{
    "on_chain": 1_000_000, "computation": 950_000, "experiment": 850_000,
    "oracle_feed": 700_000, "document": 600_000, "attestation": 500_000,
}
```

**Group 3: Challenge Window Config (lines 117-154)**
```go
// MOVE to Params struct fields:
// Params.ProvisionalThreshold        (default: 500_000)
// Params.RejectThreshold             (default: 300_000)
// Params.BaseChallengeDuration       (default: 34_272)
// Params.MinChallengeStake           (default: 11_000_000)
// Params.FailedChallengeSlashBps     (default: 220_000)
// Params.SuccessfulChallengeRewardBps(default: 300_000)
// Params.ConfidenceGrowthPerEpoch    (default: 11_000)
// Params.MaxSurvivalConfidence       (default: 770_000)
// Params.BaseCooldownBlocks          (default: 17_136)
// Params.MaxCooldownBlocks           (default: 171_360)
// Params.MaxReverificationsPerBlock  (default: 5)
```

**Group 4: Maturity Criteria (lines 200-207)**
```go
// MOVE to Params struct fields:
// Params.MaturityEstablishedMinConf         (default: 800_000)
// Params.MaturityEstablishedMinAge          (default: 34_272)
// Params.MaturityEstablishedMinVerifications(default: 3)
// Params.MaturityCanonicalMinConf           (default: 900_000)
// Params.MaturityCanonicalMinAge            (default: 342_720)
// Params.MaturityCanonicalMinVerifications  (default: 10)
```

**Group 5: Calibration & Diversity (lines 549-584)**
```go
// MOVE to Params struct fields:
// Params.MinCalibrationSamples     (default: 5)
// Params.MaxCalibrationBonus       (default: 50_000)
// Params.MaxCalibrationPenalty     (default: 150_000)
// Params.MinDiversityParticipants  (default: 3)
// Params.UniformityPenaltyBps     (default: 100_000)
// Params.DiversityBonusBps        (default: 30_000)
```

### Upgrade Steps

1. Add 25+ new fields to Params struct in `types/types.go`
2. Set defaults in `DefaultParams()` matching current hardcoded values
3. Add `Validate()` bounds checking for each new field
4. Replace every hardcoded constant read in keeper/ with `k.GetParams(ctx).FieldName`
5. Verify MsgUpdateParams handler correctly applies all new fields
6. Add integration test: governance proposal changes CategoryDecay, verify fact decay changes

### Autopoiesis Consumer Sites

| Keeper Method | Current Constant | Multiplier Path |
|---------------|-----------------|-----------------|
| `calculateDecay()` | CategoryDecay[cat].HalfLifeBlocks | `decay.{cat}.{dom}` |
| `calculateDecay()` | reverificationBonus | `revbonus.{cat}.{dom}` |
| `createRound()` | minVerifiers | `minverifiers.system` |
| `handleSubmitClaim()` | claimStakeMultiplier | `claimstake.{dom}` |
| `evaluateEvidence()` | citationEvidenceWeight | `citation.{dom}` |

### Migration

```go
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
    params := m.keeper.GetParams(ctx)
    // Populate new fields with hardcoded defaults
    params.CategoryDecayProfiles = DefaultCategoryDecayProfiles()
    params.EvidenceTypeStrengths = DefaultEvidenceTypeStrengths()
    params.ProvisionalThreshold = 500_000
    // ... all 25+ fields
    return m.keeper.SetParams(ctx, params)
}
```

**Estimated effort**: 300-400 LOC changes, 50-80 LOC new tests.

---

## P0-B: x/billing — Already Parameterized

### Current State
- 9 constants, ALL in DefaultParams()
- Params struct, SetParams/GetParams, MsgUpdateParams all working
- Validation rules present

### Remaining Work
- Verify MsgUpdateParams handler emits events
- Add bounds checking (e.g., ResearchShareBps max 50%)
- Wire autopoiesis consumer: `gas.system` multiplier

### Autopoiesis Consumer Site

```go
func (k Keeper) GetEffectiveGasCost(ctx context.Context, txType string) uint64 {
    base := k.GetParams(ctx).GasCostForTx(txType)
    mult := k.getMultiplier(ctx, "gas.system")
    return uint64(float64(base) * mult)
}
```

**Estimated effort**: 50-80 LOC changes.

---

## P0-C: x/gov — Already Parameterized

### Current State
- Params struct with all thresholds
- SetParams/GetParams working
- MsgUpdateParams handler calls SetParams
- Category-specific stakes via GetLIPStake() accessor

### Remaining Work
- Add Params.Validate() bounds checking (e.g., SupportThresholdBps <= 10_000)
- Emit events on param change
- No autopoiesis multiplier needed (governance parameters are governance-controlled by definition)

**Estimated effort**: 30-50 LOC changes.

---

## P1-A: x/staking — Structural Refactor Required

### Current State
- 26 constants in `DefaultTierConfigs()` function (lines 53-109)
- Params struct exists but tier configs are returned by function, not stored in Params
- SetParams/GetParams exists
- **Gap**: `DefaultTierConfigs()` is called directly — ignores any stored params

### Problem

```go
// Current: always returns hardcoded values
func DefaultTierConfigs() map[string]TierConfig {
    return map[string]TierConfig{
        "apprentice": { MinStake: "111000", ... },
        // ...
    }
}

// Should be:
func (k Keeper) GetTierConfigs(ctx context.Context) map[string]TierConfig {
    params := k.GetParams(ctx)
    return params.TierConfigs  // reads from store
}
```

### Upgrade Steps

1. Add `TierConfigs map[string]TierConfig` field to Params struct
2. Set default in `DefaultParams()` using current `DefaultTierConfigs()` values
3. Populate in InitGenesis
4. Replace ALL calls to `DefaultTierConfigs()` with `k.GetParams(ctx).TierConfigs`
5. Add validation: MinStake > 0, MinAccuracy in [0, 1_000_000], etc.
6. Wire MsgUpdateParams handler to apply tier config changes
7. Emit event on tier config change

### Call Sites to Update

Search for `DefaultTierConfigs()` across x/staking/keeper/:
- `keeper.go` — tier lookup
- `tiers.go` — tier computation
- `msg_server.go` — registration, promotion

### Autopoiesis: No direct multiplier

Tier configs are governance-controlled, not auto-adaptive. The autopoiesis engine
may observe tier distribution (in SSI hormone) but does not modify tier thresholds.

**Estimated effort**: 200-250 LOC changes, 30-50 LOC new tests.

---

## P1-B: x/vesting_rewards — Consolidate Reward Formulas

### Current State
- Params struct exists with BlockRewardAmount, RewardTapRate, EmptyBlockRewardRate
- SetParams/GetParams working
- MsgUpdateParams handler exists
- **Gap**: 22% research share hardcoded in keeper logic (not in Params)

### Constants to Move

```go
// In keeper/rewards.go — hardcoded 22% research share
// MOVE to Params.ResearchShareBps (default: 220_000)
// MOVE to Params.ResearchFundAddress (default: module account)
```

### Upgrade Steps

1. Add `ResearchShareBps` field to Params struct (default: 220_000)
2. Replace hardcoded 22% in `RouteReward()` with `k.GetParams(ctx).ResearchShareBps`
3. Validate: ResearchShareBps in [0, 500_000] (max 50%)
4. Wire autopoiesis consumer: `blockreward.system` multiplier

### Autopoiesis Consumer Site

```go
func (k Keeper) GetEffectiveBlockReward(ctx context.Context) math.Int {
    base := k.GetParams(ctx).BlockRewardAmount
    mult := k.getMultiplier(ctx, "blockreward.system")
    scaled := base.MulRaw(int64(mult * 1_000_000)).QuoRaw(1_000_000)
    return scaled
}
```

**Estimated effort**: 80-120 LOC changes, 20-30 LOC new tests.

---

## P2-A: x/emergency — Add Missing Handler

### Current State
- Params struct exists with all quorum thresholds
- SetParams/GetParams in keeper/state.go
- **Gap**: NO MsgUpdateParams handler

### Upgrade Steps

1. Add `MsgUpdateParams` to types/types.go
2. Register in codec.go service descriptor
3. Implement handler in msg_server.go:
   ```go
   func (ms msgServer) UpdateParams(ctx context.Context, msg *types.MsgUpdateParams) (...) {
       if ms.Keeper.GetAuthority() != msg.Authority {
           return nil, govtypes.ErrInvalidSigner
       }
       if err := msg.Params.Validate(); err != nil {
           return nil, err
       }
       return &types.MsgUpdateParamsResponse{}, ms.Keeper.SetParams(ctx, msg.Params)
   }
   ```
4. Register in module.go RegisterServices
5. Test: governance proposal changes HaltQuorum

### Safety Note

Emergency params should require a HIGHER governance threshold than normal params.
Consider requiring Guardian-tier supermajority for emergency param changes.

**Estimated effort**: 60-80 LOC new code, 15-20 LOC new tests.

---

## P2-B: x/tokens — Minor Consolidation

### Current State
- Already parameterized (5 fields)
- SetParams/GetParams working
- MsgUpdateParams handler exists

### Remaining Work
- Verify MsgUpdateParams emits events
- No autopoiesis multiplier needed (token expiration is economic policy, not adaptive)

**Estimated effort**: 20-30 LOC changes.

---

## P3: x/home — Aging Constants Migration

### Current State
- 10 hardcoded constants in keeper/aging.go
- Params struct exists but only has MaxHomesPerAccount, MaxKeysPerHome, MaxSessionsPerHome
- **Gap**: Aging thresholds and reward multipliers outside Params

### Constants to Move

```go
// From keeper/aging.go lines 24-37:
// MOVE to Params:
// Params.ThresholdEstablishedBlocks  (default: 34_272)    // 1 day
// Params.ThresholdMatureBlocks       (default: 342_720)   // 10 days
// Params.ThresholdVenerableBlocks    (default: 1_028_160) // 30 days
// Params.ThresholdAncientBlocks      (default: 3_427_200) // 100 days
// Params.RewardMultiplierFresh       (default: 10_000)    // 1.0x
// Params.RewardMultiplierEstablished (default: 10_500)    // 1.05x
// Params.RewardMultiplierMature      (default: 11_000)    // 1.10x
// Params.RewardMultiplierVenerable   (default: 11_500)    // 1.15x
// Params.RewardMultiplierAncient     (default: 12_000)    // 1.20x
```

### Upgrade Steps

1. Add 9 new fields to Params struct
2. Set defaults matching current constants
3. Replace all reads in aging.go with `k.GetParams(ctx).ThresholdXxx`
4. Validate: thresholds must be strictly increasing
5. Migration: Migrate1to2 populates new fields

**Estimated effort**: 100-150 LOC changes, 20-30 LOC new tests.

---

## P3: Remaining 16 Stub Modules

The following modules are partially or fully stubs. They require parameterization
as part of their full implementation, not as a separate upgrade:

| Module | Status | Params Action |
|--------|--------|---------------|
| x/research | Partial keeper | Add research-specific params when completing |
| x/disputes | Stub | Add dispute window/threshold params |
| x/evidence_mgmt | Stub | Add evidence scoring params |
| x/ontology | Stub | Add domain management params |
| x/channels | Stub | Add channel capacity/timeout params |
| x/claiming_pot | Stub | Add pot distribution params |
| x/bvm | Partial (interpreter) | Add VM execution limits params |
| x/schedule | Partial | Already parameterized — verify |
| x/tree | Stub | Add tree structure params |
| x/capture_challenge | Stub | Add challenge params |
| x/capture_defense | Stub | Add defense params |
| x/qualification | Stub | Add qualification criteria params |
| x/partnerships | Stub | Add partnership params |
| x/discovery | Stub | Add discovery params |
| x/ibcratelimit | Stub | Add rate limit params |
| x/icaauth | Stub | Add ICA auth params |

**Approach**: Each module gets parameterized as it is fully implemented. The
parameterization pattern from P0/P1 modules serves as the template.

---

## Evolution Path Summary

```
Week 1: Foundation
├── x/knowledge: 25+ constants → Params (P0)
├── x/billing: Verify + add gas.system multiplier consumer (P0)
└── x/gov: Bounds validation + events (P0)

Week 2: Economics
├── x/staking: Refactor DefaultTierConfigs() → Params (P1)
├── x/vesting_rewards: Consolidate 22% share + blockreward.system (P1)
└── x/emergency: Add MsgUpdateParams handler (P2)

Week 3: Observation
├── x/autopoiesis: Scaffold module, observer, hormones (Phase 1)
└── x/tokens: Minor consolidation (P2)

Week 4: Adaptation
├── x/autopoiesis: SSI hormone live (Phase 2)
├── x/home: Aging constants → Params (P3)
└── Consumer wiring: knowledge + vesting_rewards read multipliers

Week 5-6: Full Loop
├── x/autopoiesis: All 5 hormones live (Phase 3)
├── Consumer wiring: billing + knowledge read multipliers
└── Integration tests: 100-epoch simulation

Week 7: Meta-Adaptation
├── x/autopoiesis: Level 3 learning (Phase 4)
└── Cross-stack golden vector tests

Week 8-10: Morphogenic
├── x/autopoiesis: Domain budding/senescence (Phase 5)
└── Remaining stub modules parameterized as implemented
```

### Total Estimated New/Changed LOC

| Category | LOC |
|----------|-----|
| Params migration (8 modules) | ~800 |
| x/autopoiesis module | ~2,000 |
| Consumer wiring (5 modules) | ~300 |
| Tests | ~600 |
| Migration (Migrate1to2) | ~200 |
| **Total** | **~3,900** |

---

_End of Module Upgrade Plans_
