# B11-4 — Prophetic Diversity Gate (FARM-13: Prophetic Manipulation)

## The Attack
The prophetic fact system rewards facts that were "barely accepted" (low base
confidence) but later gained many citations — the chain's way of recognizing
ahead-of-their-time ideas. The attack:

1. Submit a deliberately under-evidenced claim (lands at low confidence)
2. Use sybil accounts to gradually cite it over time
3. Trigger prophetic detection → 2× growth rate + elevated confidence cap
4. Earn accelerated citation rewards from the boosted fact

Current defense: `PropheticCitationThreshold` requires significant citations.
But it doesn't check WHERE the citations come from.

## The Fix

### 1. Require Citation Diversity for Prophetic Status

Add to `x/knowledge/types/types.go` ExtendedParams:

```go
// FARM-13: Prophetic facts require citation diversity.
// Citations must come from at least this many unique submitters to qualify.
PropheticMinUniqueCiters uint64 `json:"prophetic_min_unique_citers"` // default 5

// FARM-13: Maximum fraction of citations from any single address.
// If any one address accounts for more than this fraction, prophetic status denied.
PropheticMaxSingleCiterShare uint64 `json:"prophetic_max_single_citer_share"` // default 300,000 (30%)
```

Defaults:
```go
PropheticMinUniqueCiters:     5,
PropheticMaxSingleCiterShare: 300_000, // 30%
```

### 2. Update DetectPropheticFacts

In `x/knowledge/keeper/growth.go`, in `DetectPropheticFacts()`, after the
citation count check:

```go
// FARM-13: Require citation diversity for prophetic status.
// Prophetic recognition is about community vindication, not self-promotion.
if ep.PropheticMinUniqueCiters > 0 {
    record, found := k.GetCitationSourceRecord(ctx, fact.Id)
    if !found {
        return false // no citation source data → can't verify diversity
    }

    // Check unique citers
    uniqueCiters := uint64(len(record.SourceCounts))
    if uniqueCiters < ep.PropheticMinUniqueCiters {
        return false // not enough diverse citers
    }

    // Check no single address dominates
    for _, count := range record.SourceCounts {
        share := (count * 1_000_000) / record.TotalCites
        if share > ep.PropheticMaxSingleCiterShare {
            return false // single address too dominant
        }
    }
}
```

**Dependency note:** This reuses the `CitationSourceRecord` from B11-2. If B11-2
hasn't landed yet, you'll need to create the record tracking here. Check if
`GetCitationSourceRecord` exists:
```bash
grep -n "CitationSourceRecord\|GetCitationSource" x/knowledge/keeper/state.go
```
If not, create it following the B11-2 pattern (store under `CitationSourcePrefix`).

### 3. Also Gate PropheticGrowth on Diversity

In `ApplyPropheticGrowth()`, add a periodic re-check:

```go
// Re-validate prophetic status every growth epoch.
// If citation diversity drops below threshold (e.g., citations removed,
// sybil accounts banned), revoke prophetic status.
if !k.validatePropheticDiversity(ctx, fact, ep) {
    fact.IsProphetic = false
    k.SetFact(ctx, fact)
    ctx.EventManager().EmitEvent(
        sdk.NewEvent("prophetic_status_revoked",
            sdk.NewAttribute("fact_id", fact.Id),
            sdk.NewAttribute("reason", "FARM-13: citation diversity below threshold"),
        ),
    )
    return false
}
```

### 4. Helper Function

```go
// validatePropheticDiversity checks if a fact's citation sources meet the
// diversity requirements for prophetic status.
func (k Keeper) validatePropheticDiversity(ctx sdk.Context, fact *types.Fact, ep types.ExtendedParams) bool {
    if ep.PropheticMinUniqueCiters == 0 {
        return true // diversity check disabled
    }
    record, found := k.GetCitationSourceRecord(ctx, fact.Id)
    if !found {
        return false
    }
    if uint64(len(record.SourceCounts)) < ep.PropheticMinUniqueCiters {
        return false
    }
    for _, count := range record.SourceCounts {
        if (count * 1_000_000 / record.TotalCites) > ep.PropheticMaxSingleCiterShare {
            return false
        }
    }
    return true
}
```

### 5. Tests

```go
func TestFARM13_PropheticDiversity(t *testing.T) {
    // 10 citations from 10 unique addresses → prophetic allowed
    // 10 citations from 2 addresses → denied (too few unique citers)
    // 10 citations from 5 addresses, one has 4 (40%) → denied (>30% single share)
    // 10 citations from 5 addresses, max per address = 3 (30%) → allowed (at threshold)
    // Prophetic status revoked when diversity drops
}
```

### 6. Run

```bash
go test ./x/knowledge/keeper/... -v -run TestFARM13
go test ./x/knowledge/... -count=1
```

## Commit
```
fix(keeper): FARM-13 prophetic diversity gate — require diverse citers

Prophetic fact detection now requires citations from ≥5 unique
addresses with no single address accounting for >30% of citations.
Prophetic status is re-validated each growth epoch and revoked if
diversity drops. Prevents sybil-driven prophetic manipulation.
```
