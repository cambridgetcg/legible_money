# B11-2 — Citation Concentration Detection (FARM-11: Sybil Citation Rings)

## The Attack
Agent A submits claim X. Agent B (same person, different address) submits
claim Y citing X. Agent A gets full citation rewards (no self-citation
discount because different address). Agent C (also same person) submits
claim Z citing Y. Rewards cascade through the ring.

**FARM-3 only catches direct self-citation** (same submitter address).
It doesn't detect one-hop or multi-hop sybil chains.

## The Fix

### 1. Citation Concentration Score

Track, per fact, how concentrated its citation rewards are among a small
set of addresses. If a fact's citations come overwhelmingly from a tiny
cluster, discount future citation rewards for that fact.

In `x/knowledge/types/types.go`, add to ExtendedParams:

```go
// FARM-11: Citation concentration detection — discount rewards for facts
// whose citations come from a suspiciously small set of addresses.
CitationConcentrationThreshold uint64 `json:"citation_concentration_threshold"` // default 800,000 (80%)
CitationConcentrationMinCites  uint64 `json:"citation_concentration_min_cites"` // default 5
CitationConcentrationMaxAddrs  uint64 `json:"citation_concentration_max_addrs"` // default 3
CitationConcentrationDiscount  uint64 `json:"citation_concentration_discount"`  // default 700,000 (70% discount)
```

Meaning: if ≥80% of a fact's citations (minimum 5) come from ≤3 unique
addresses, apply a 70% discount to future citation rewards for that fact.

Defaults:
```go
CitationConcentrationThreshold: 800_000,
CitationConcentrationMinCites:  5,
CitationConcentrationMaxAddrs:  3,
CitationConcentrationDiscount:  700_000,
```

### 2. Track Citation Sources

Add a new field to the Fact proto or a separate index:

**Option A (lightweight — recommend):** Add a citation source tracking struct
in keeper state, keyed by fact ID. No proto change needed.

```go
// CitationSourceRecord tracks unique addresses that have cited a fact.
// Used by FARM-11 to detect citation concentration.
type CitationSourceRecord struct {
    FactID       string            `json:"fact_id"`
    SourceCounts map[string]uint64 `json:"source_counts"` // address → citation count
    TotalCites   uint64            `json:"total_cites"`
}
```

Store under a new prefix key: `CitationSourcePrefix = []byte{0x27}`.

### 3. Update Citation Flow

In `x/knowledge/keeper/citation.go`, in `DistributeCitationRewards()`:

a) Before distributing, look up the `CitationSourceRecord` for the cited fact.
b) Check concentration: compute what fraction of `TotalCites` comes from the
   top `CitationConcentrationMaxAddrs` addresses.
c) If fraction ≥ threshold AND total cites ≥ min, apply the concentration discount.
d) After distributing, update the record with the new citing address.

```go
// Check FARM-11 citation concentration
concentrationDiscount := k.GetCitationConcentrationDiscount(ctx, ref.ToFactId, citingSubmitter, ep)
if concentrationDiscount > 0 {
    adjustedShare = (adjustedShare * (1_000_000 - concentrationDiscount)) / 1_000_000
}
```

### 4. The Concentration Check Function

```go
// GetCitationConcentrationDiscount returns the FARM-11 discount for a fact's
// citation rewards based on source concentration.
func (k Keeper) GetCitationConcentrationDiscount(
    ctx sdk.Context, factID, citingAddr string, ep types.ExtendedParams,
) uint64 {
    record, found := k.GetCitationSourceRecord(ctx, factID)
    if !found || record.TotalCites < ep.CitationConcentrationMinCites {
        return 0
    }

    // Find top N addresses by citation count
    topN := ep.CitationConcentrationMaxAddrs
    topTotal := topNSum(record.SourceCounts, topN)

    concentrationRate := (topTotal * 1_000_000) / record.TotalCites
    if concentrationRate >= ep.CitationConcentrationThreshold {
        return ep.CitationConcentrationDiscount
    }
    return 0
}
```

### 5. Edge Cases

- Genesis axioms (submitter="genesis") are exempt — their citation sources
  will be naturally diverse as the network grows
- Legitimate popular facts cited by many different agents won't trigger this —
  the top-3 addresses need to account for ≥80% of citations
- The check is per-fact, not per-submitter — it's the *cited fact* that gets
  discounted, not the citing agent's rewards overall
- New facts with <5 citations are exempt (too early to judge)

### 6. Tests

```go
func TestFARM11_CitationConcentration(t *testing.T) {
    // 10 citations from 10 unique addresses → no discount
    // 10 citations from 2 addresses (5 each) → 100% from top 3 → discount
    // 5 citations from 3 addresses → just at threshold → discount
    // 4 citations from 2 addresses → below min cites → no discount
    // 10 citations where top 3 = 70% → below 80% threshold → no discount
}
```

### 7. Run

```bash
go test ./x/knowledge/keeper/... -v -run TestFARM11
go test ./x/knowledge/... -count=1
```

## Commit
```
fix(keeper): FARM-11 citation concentration detection — sybil ring defense

Tracks per-fact citation source distribution. When ≥80% of a fact's
citations (min 5) come from ≤3 addresses, future citation rewards
for that fact are discounted 70%. Prevents sybil citation pyramids
without requiring full sybil detection.
```
