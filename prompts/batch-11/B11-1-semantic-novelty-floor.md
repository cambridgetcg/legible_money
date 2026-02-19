# B11-1 — Semantic Novelty Floor (FARM-10: Genesis Axiom Restating)

## The Attack
An agent submits claims that are technically true but epistemically empty —
they just restate what genesis axioms already say. Example:

> "Zero is a natural number and every natural number has a successor"

References: [MATH-001, MATH-002]. Gets +60,000 BPS support bonus (2 verified
refs at 30k each). Passes evaluation easily. Becomes a verified fact. Then
other agents (or sybils) cite it, and the original agent earns citation rewards.

**Cost to attacker:** minimal stake  
**Reward:** citation income stream from a fact that adds zero knowledge  
**FARM-2 partial coverage:** only triggers at >95% acceptance rate, so a
smart farmer mixes in some genuine claims.

## The Fix

### 1. Minimum Information Gain Threshold

The novelty stage (`x/knowledge/evaluation/novelty.go`) already computes
`calculateInformationGain()` which measures domain sparsity and reference
novelty. Add a **floor check**: if a claim's information gain falls below
a minimum AND it references only high-maturity facts, penalize it.

In `x/knowledge/types/types.go`, add to ExtendedParams:

```go
// FARM-10: Semantic novelty floor — block claims that restate existing knowledge.
// Claims with info gain below this threshold AND referencing only canonical/established
// facts get a heavy penalty. Prevents genesis axiom restating for citation farming.
MinInformationGainBps uint64 `json:"min_information_gain_bps"` // default 50,000 (5%)

// FARM-10: Penalty applied when information gain is below floor and all references
// are high-maturity (canonical or established).
RestatingPenaltyBps uint64 `json:"restating_penalty_bps"` // default 200,000 (20%)
```

Defaults:
```go
MinInformationGainBps: 50_000,   // 5% — very low bar, only catches true restating
RestatingPenaltyBps:   200_000,  // 20% — heavy enough to push below acceptance
```

### 2. Implementation in Novelty Stage

In `evaluateNovelty()` in `x/knowledge/evaluation/novelty.go`:

After computing `infoGainBps`, add:

```go
// FARM-10: Check for axiom restating.
// If info gain is below floor AND all references are canonical/established,
// this claim is likely restating existing knowledge.
if infoGainBps < ep.MinInformationGainBps && len(claim.References) > 0 {
    allHighMaturity := true
    for _, refId := range claim.References {
        fact, found := evalCtx.GetFact(ctx, refId)
        if !found {
            allHighMaturity = false
            break
        }
        if fact.Maturity != types.MaturityCanonical && fact.Maturity != types.MaturityEstablished {
            allHighMaturity = false
            break
        }
    }
    if allHighMaturity {
        result.Adjustment = -int64(ep.RestatingPenaltyBps)
        result.Justification = fmt.Sprintf(
            "FARM-10: information gain %d below floor %d with all canonical/established refs",
            infoGainBps, ep.MinInformationGainBps,
        )
        return result
    }
}
```

### 3. Edge Cases

- Claims that synthesize across domains should NOT be penalized — cross-domain
  references produce high information gain even with canonical refs
- Claims in sparse domains (few existing facts) naturally have high info gain
  due to domain sparsity, so they pass the floor easily
- Genesis axioms themselves (submitter="genesis") are exempt — they're loaded
  at block 0, not submitted through the evaluation pipeline

### 4. Tests

In `x/knowledge/evaluation/novelty_test.go`:

```go
func TestFARM10_RestatingPenalty(t *testing.T) {
    // Claim with low info gain + all canonical refs → penalized
    // Claim with low info gain + mixed maturity refs → not penalized
    // Claim with high info gain + all canonical refs → not penalized
    // Cross-domain claim with canonical refs → not penalized (high info gain)
}
```

### 5. Run

```bash
go test ./x/knowledge/evaluation/... -v -run TestFARM10
go test ./x/knowledge/... -count=1
```

## Commit
```
fix(evaluation): FARM-10 semantic novelty floor — block axiom-restating claims

Claims with information gain below MinInformationGainBps (5%) that
reference only canonical/established facts receive a -20% penalty,
pushing them below acceptance threshold. Prevents farming citation
rewards by restating genesis axioms as "new" claims.
```
