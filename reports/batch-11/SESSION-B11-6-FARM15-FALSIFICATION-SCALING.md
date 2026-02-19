# B11-6: FARM-15 Falsification Reward Scaling — Session Report

## Summary

Implemented age×citation weighted falsification reward scaling. Rewards now scale dynamically from 1× to 3× the challenger's stake based on the target fact's age and citation count. Young uncited facts yield minimal reward (deterring economic predation); old widely-cited errors yield maximum reward (incentivizing genuine truth correction).

---

## Changes

### `x/knowledge/types/types.go`

**New constants** (after `FalsificationPoolShareBps`):
```go
FalsificationBaseMultiplierBps = 1_000_000  // 1× base
FalsificationAgeScaleBlocks    = 342_720    // ~10 days for max age bonus
FalsificationAgeMaxBonusBps    = 1_000_000  // +1× max age bonus
FalsificationCiteScaleCount    = 50         // 50 citations for max citation bonus
FalsificationCiteMaxBonusBps   = 1_000_000  // +1× max citation bonus
```

**New ExtendedParams fields**: 5 new fields mirror the constants with JSON tags.

**New validation**: Sum of base + age_max + cite_max must not exceed `FalsificationRewardCapBps` (3×). Age scale and cite scale must be > 0.

**New defaults** in `DefaultExtendedParams()`.

### `x/knowledge/keeper/rounds.go`

**Modified `distributeFalsificationReward()`**: Replaced flat `types.FalsificationRewardCapBps` (3×) cap with dynamic calculation:

```
dynamicCapBps = baseMultiplier + ageBonus + citationBonus

ageBonus    = min(ageMaxBonusBps, ageMaxBonusBps * ageBlocks / ageScaleBlocks)
citationBonus = min(citeMaxBonusBps, citeMaxBonusBps * citationCount / citeScaleCount)

rewardCap = cfStake * dynamicCapBps / 1_000_000
```

Uses `safeMulDiv()` for overflow-safe arithmetic. Fetches params via `k.GetExtendedParams(ctx)`.

### `x/knowledge/keeper/keeper_test.go`

**Updated 2 existing tests**:
- `TestContradictionAccepted_StakeReturnedAndRewarded` — added `CitationCount: 50` so dynamic cap doesn't bind on the 16 LGM reward
- `TestFalsificationReward_CappedAt3x` — added `CitationCount: 50` + high block height (500k) so dynamic cap = 3× matches old behavior

**Added 4 new FARM-15 tests**:

| Test | Scenario | Expected |
|------|----------|----------|
| `TestFARM15_NewFactMinimalReward` | age=100 blocks, 0 citations | ~1× (10M on 10M stake) |
| `TestFARM15_MediumFactMediumReward` | age=171,360 (~5 days), 25 citations | 2× (20M on 10M stake) |
| `TestFARM15_OldHighlyCitedFactMaxReward` | age=500k, 100 citations | 3× (30M on 10M stake) |
| `TestFARM15_VeryNewFactMinimumReward` | age=10 blocks, 0 citations | ~1× (10,000,290 on 10M) |

---

## Behavioral Impact

| Scenario | Old reward | New reward |
|----------|-----------|------------|
| 1-day-old, 0 citations | 3× stake | ~1.1× stake |
| 5-day-old, 10 citations | 3× stake | ~1.7× stake |
| 10-day-old, 50+ citations | 3× stake | 3.0× stake |
| 30-day-old, 100 citations | 3× stake | 3.0× stake (capped) |

---

## Verification

```
go build ./...                                                    # Clean
go test ./x/knowledge/types/... -count=1                          # PASS
go test ./x/knowledge/keeper/... -run "TestFARM15|TestContradiction|TestFalsificationReward|TestFalsificationEpoch" -count=1
  # 9/9 PASS (5 existing + 4 new)
```

Pre-existing failures in `TestFARM14_*` and `TestInferStratum` are unrelated (from unstaged FARM-14 work on main).

---

## Design Notes

- The epoch pool cap (`FalsificationPoolShareBps = 15%`) remains unchanged — it acts as a global safety valve regardless of per-fact dynamic caps
- `safeMulDiv()` prevents uint64 overflow in age and citation bonus calculations
- The validation constraint `base + age_max + cite_max <= reward_cap` ensures governance can't accidentally set params that exceed the overall cap
- `FalsificationRewardCapBps` is preserved as an overall ceiling — dynamic cap can never exceed it even with governance param changes
