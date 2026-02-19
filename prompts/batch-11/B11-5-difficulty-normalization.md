# B11-5 — Difficulty Bonus Normalization (FARM-14: Domain Ceiling Skew)

## The Problem
The verification difficulty bonus in `calculateRewardsAndSlashes()`
(`x/knowledge/keeper/confidence.go`) gives more reward for correctly verifying
lower-confidence claims:

```go
if result.Confidence > 0 && result.Confidence < 1_000_000 {
    difficulty := 1_000_000 - result.Confidence
    reward = reward + reward*difficulty/1_000_000
}
```

At confidence 500k: +50% reward. At 900k: +10%.

This is epistemically correct — harder verification deserves more pay. But it
creates a perverse incentive: claims in empirical/social domains naturally land
at lower confidence (domain stratum ceiling is lower), so verifiers in those
domains earn disproportionately more per verification than verifiers in
mathematics (where confidence is near 1M).

**The result:** Rational verifiers prefer empirical domains over formal ones,
not because the work is harder but because the ceiling is lower. This
starves formal domains of verification attention.

## The Fix

### 1. Normalize Difficulty by Domain Ceiling

Instead of using raw `1_000_000 - confidence` as difficulty, compute
difficulty relative to the domain's confidence ceiling:

```
difficulty = (ceiling - confidence) / ceiling
```

A claim at 600k in mathematics (ceiling ~1M) has difficulty = 40%.
A claim at 600k in sociology (ceiling ~700k) has difficulty = 14%.

This correctly reflects that 600k in mathematics represents a genuinely
uncertain result, while 600k in sociology is nearly at the ceiling and
thus not epistemically surprising.

### 2. Implementation

In `x/knowledge/keeper/confidence.go`, in `calculateRewardsAndSlashes()`:

Replace:
```go
if result.Confidence > 0 && result.Confidence < 1_000_000 {
    difficulty := 1_000_000 - result.Confidence
    reward = reward + reward*difficulty/1_000_000
}
```

With:
```go
// FARM-14: Normalized difficulty bonus — adjust by domain ceiling.
// Difficulty is measured relative to the domain's confidence ceiling,
// not the absolute maximum. A claim near its domain ceiling is "easy"
// even if the absolute confidence is low.
if result.Confidence > 0 {
    ceiling := k.getDomainConfidenceCeiling(ctx, result.Domain)
    if ceiling == 0 {
        ceiling = 1_000_000 // fallback
    }
    if result.Confidence < ceiling {
        // Normalized difficulty: (ceiling - confidence) / ceiling
        difficulty := safeMulDiv(ceiling - result.Confidence, 1_000_000, ceiling)
        reward = reward + safeMulDiv(reward, difficulty, 1_000_000)
    }
    // At or above ceiling → no difficulty bonus (claim was "expected")
}
```

### 3. Add getDomainConfidenceCeiling Helper

```go
// getDomainConfidenceCeiling returns the confidence ceiling for a domain.
// Checks ontology keeper first, falls back to stratum-based mapping.
func (k Keeper) getDomainConfidenceCeiling(ctx sdk.Context, domain string) uint64 {
    if k.ontologyKeeper != nil {
        maxConf, _, err := k.ontologyKeeper.GetConfidenceCeiling(ctx, domain)
        if err == nil && maxConf > 0 {
            return maxConf
        }
    }
    // Fallback: use stratum ceiling
    ep := k.GetExtendedParams(ctx)
    stratum := inferDomainStratum(domain) // or k.inferDomainStratum if B11-3 landed
    ceiling, ok := ep.GetStratumConfidenceCeiling(stratum)
    if ok {
        return ceiling
    }
    return 1_000_000
}
```

This may already exist as `applyStratumCeiling` — check and reuse if so.

### 4. Impact Analysis

Before this fix:
- Math verifier: claim at 950k, difficulty = 5%, bonus = +5%
- Sociology verifier: claim at 600k, difficulty = 40%, bonus = +40%
- **8× more bonus for sociology**, despite math being epistemically harder to verify

After this fix (assuming math ceiling ~1M, sociology ceiling ~700k):
- Math verifier: claim at 950k, normalized difficulty = (1M-950k)/1M = 5%, bonus = +5%
- Sociology verifier: claim at 600k, normalized difficulty = (700k-600k)/700k = 14%, bonus = +14%
- **2.8× more bonus for sociology** — still rewards harder work but not excessively

### 5. Tests

```go
func TestFARM14_NormalizedDifficulty(t *testing.T) {
    // Same absolute confidence, different ceilings → different bonus
    // Confidence at ceiling → zero bonus
    // Confidence above ceiling → zero bonus (shouldn't happen but defensive)
    // Unknown domain → falls back to 1M ceiling
    // Verify math 950k gives small bonus, sociology 600k gives moderate bonus
}
```

### 6. Run

```bash
go test ./x/knowledge/keeper/... -v -run TestFARM14
go test ./x/knowledge/keeper/... -v -run TestAggregateVerification
go test ./x/knowledge/... -count=1
```

## Commit
```
fix(keeper): FARM-14 normalized difficulty bonus — adjust by domain ceiling

Verification difficulty bonus is now computed relative to the domain's
confidence ceiling rather than the absolute maximum. A claim near its
ceiling is easy regardless of absolute confidence. Prevents over-
rewarding verifiers in low-ceiling domains and under-rewarding those
in high-ceiling domains.
```
