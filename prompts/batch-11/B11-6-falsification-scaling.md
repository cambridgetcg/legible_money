# B11-6 — Falsification Reward Scaling (FARM-15: Age×Citation Weighted)

## The Problem
Falsification rewards are currently capped at 3× the challenger's stake
(`FalsificationRewardCapBps = 3_000_000`), regardless of what was disproved.
This creates two issues:

**Issue A — Old errors are under-rewarded:** Disproving a 1-day-old claim with
0 citations pays the same as disproving a 100-day-old claim with 50 citations.
The latter is far more valuable — you've corrected a long-standing error that
many people built on.

**Issue B — Young facts are over-targeted:** Whale challengers can stake large
and challenge young, borderline facts for the 3× payoff. This isn't truth-
seeking — it's economic predation on new knowledge. FARM-9's 50% minimum
stake helps but doesn't fully address the incentive.

## The Fix

### 1. Scale Reward by Fact Age and Citation Impact

Replace the flat 3× cap with a dynamic cap based on the target fact's age
and citation count:

```
rewardMultiplier = baseMultiplier + ageBonus + citationBonus
```

Where:
- `baseMultiplier` = 1.0× (always get your stake back + 100%)
- `ageBonus` = min(1.0, ageBlocks / ageScaleBlocks) × 1.0 → up to +1.0×
- `citationBonus` = min(1.0, citationCount / citationScaleCount) × 1.0 → up to +1.0×

So the total range is 1.0× to 3.0× — same maximum as before, but you only
reach 3× by disproving something old AND widely-cited.

### 2. Add Parameters

In `x/knowledge/types/types.go`, add to ExtendedParams:

```go
// FARM-15: Falsification reward scaling.
// Rewards scale with the target fact's age and citation impact.
FalsificationBaseMultiplierBps uint64 `json:"falsification_base_multiplier_bps"` // default 1,000,000 (1×)
FalsificationAgeScaleBlocks    uint64 `json:"falsification_age_scale_blocks"`    // default 342,720 (~10 days at 2.5s blocks)
FalsificationAgeMaxBonusBps    uint64 `json:"falsification_age_max_bonus_bps"`   // default 1,000,000 (1×)
FalsificationCiteScaleCount    uint64 `json:"falsification_cite_scale_count"`    // default 50
FalsificationCiteMaxBonusBps   uint64 `json:"falsification_cite_max_bonus_bps"`  // default 1,000,000 (1×)
```

Defaults:
```go
FalsificationBaseMultiplierBps: 1_000_000,  // 1×
FalsificationAgeScaleBlocks:    342_720,    // ~10 days
FalsificationAgeMaxBonusBps:    1_000_000,  // +1× max age bonus
FalsificationCiteScaleCount:    50,         // 50 citations for max bonus
FalsificationCiteMaxBonusBps:   1_000_000,  // +1× max citation bonus
```

### 3. Implementation

In `x/knowledge/keeper/rounds.go`, in `distributeFalsificationReward()`:

Replace the flat `FalsificationRewardCapBps` usage with:

```go
// FARM-15: Dynamic reward cap based on target fact's age and citation impact.
currentHeight := uint64(ctx.BlockHeight())
ageBlocks := uint64(0)
if currentHeight > targetFact.CreatedAtHeight {
    ageBlocks = currentHeight - targetFact.CreatedAtHeight
}

// Age bonus: linear ramp from 0 to FalsificationAgeMaxBonusBps
ageBonusBps := ep.FalsificationAgeMaxBonusBps
if ageBlocks < ep.FalsificationAgeScaleBlocks {
    ageBonusBps = safeMulDiv(ep.FalsificationAgeMaxBonusBps, ageBlocks, ep.FalsificationAgeScaleBlocks)
}

// Citation bonus: linear ramp from 0 to FalsificationCiteMaxBonusBps
citeBonusBps := ep.FalsificationCiteMaxBonusBps
if targetFact.CitationCount < ep.FalsificationCiteScaleCount {
    citeBonusBps = safeMulDiv(ep.FalsificationCiteMaxBonusBps, targetFact.CitationCount, ep.FalsificationCiteScaleCount)
}

// Total multiplier: base + age + citation
dynamicCapBps := ep.FalsificationBaseMultiplierBps + ageBonusBps + citeBonusBps

// Use dynamic cap instead of flat FalsificationRewardCapBps
rewardCap := new(big.Int).Mul(cfStake, big.NewInt(int64(dynamicCapBps)))
rewardCap.Div(rewardCap, big.NewInt(1_000_000))
```

### 4. Behavioral Impact

| Scenario | Old reward | New reward |
|----------|-----------|------------|
| 1-day old, 0 citations | 3× stake | 1.1× stake |
| 5-day old, 10 citations | 3× stake | 1.7× stake |
| 10-day old, 50 citations | 3× stake | 3.0× stake |
| 30-day old, 100 citations | 3× stake | 3.0× stake (capped) |

This makes it uneconomical to target young facts for profit — the reward
barely covers the risk. But disproving established, widely-cited falsehoods
pays the full maximum, which is exactly what truth-seeking requires.

### 5. Keep the Epoch Pool Cap

The existing `FalsificationPoolShareBps` (15% of epoch reward pool) remains
as a global cap regardless of the per-fact dynamic cap. This prevents any
single epoch from over-spending on falsification rewards.

### 6. Tests

```go
func TestFARM15_FalsificationScaling(t *testing.T) {
    // New fact (age=100 blocks, 0 citations) → ~1× reward
    // Medium fact (age=171,360 blocks, 25 citations) → ~2× reward
    // Old fact (age=500,000 blocks, 100 citations) → 3× reward (max)
    // Very new fact (age=10 blocks, 0 citations) → ~1× reward (minimum)
    // Epoch pool cap still enforced
}
```

### 7. Run

```bash
go test ./x/knowledge/keeper/... -v -run TestFARM15
go test ./x/knowledge/keeper/... -v -run TestDistributeFalsification
go test ./x/knowledge/... -count=1
```

## Commit
```
fix(keeper): FARM-15 falsification reward scaling — age×citation weighted

Falsification rewards now scale dynamically: base 1× + up to 1× for
fact age + up to 1× for citation count. Young uncited facts yield
minimal reward (deterring economic predation); old widely-cited errors
yield maximum reward (incentivizing genuine truth correction). Total
range preserved at 1×–3×. Epoch pool cap unchanged.
```
