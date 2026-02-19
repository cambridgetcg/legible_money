# B13-3 — FARM-10 through FARM-15 Verification & Integration Test

## Context
All six FARM fixes (10-15) landed in batch 11/12 code. This session verifies
they work correctly together and don't have any integration issues.

## What Landed (Verified in Code)
- **FARM-10:** Semantic novelty floor in `evaluation/novelty.go` — ✓
- **FARM-11:** Citation concentration detection in `keeper/citation.go` — ✓
- **FARM-12:** Domain stratum governance in `keeper/confidence.go` + `msg_server.go` — ✓
- **FARM-13:** Prophetic diversity gate in `keeper/growth.go` — ✓
- **FARM-14:** Normalized difficulty bonus in `keeper/confidence.go` — ✓
- **FARM-15:** Falsification reward scaling params in `types/types.go` — ✓

## Task

### 1. Run Full Test Suite

```bash
go test ./x/knowledge/... -count=1 -v -timeout 120s
```

Fix any failures.

### 2. Integration Test: FARM Fixes Working Together

Create `x/knowledge/keeper/farm_integration_test.go`:

```go
func TestFARMIntegration(t *testing.T) {
    // Scenario: Agent tries multiple farming strategies, all blocked

    // 1. Submit axiom-restating claim → FARM-10 blocks it
    // 2. Create sybil citation ring → FARM-11 discounts rewards
    // 3. Propose domain without stratum → FARM-12 assigns restrictive default
    // 4. Try to game prophetic with sybil citations → FARM-13 blocks status
    // 5. Verify low-ceiling domain doesn't over-reward → FARM-14 normalizes
    // 6. Challenge young uncited fact → FARM-15 gives minimal reward
}
```

### 3. Verify Default Parameters

Check that all FARM parameters have sensible defaults in `DefaultExtendedParams()`:

```go
// FARM-10
MinInformationGainBps: 50_000    // 5% — only catches true restating
RestatingPenaltyBps:   200_000   // 20% — heavy enough to push below acceptance

// FARM-11
CitationConcentrationThreshold: 800_000   // 80%
CitationConcentrationMinCites:  5
CitationConcentrationMaxAddrs:  3
CitationConcentrationDiscount:  700_000   // 70%

// FARM-12
// Domain stratum governance — check inferDomainStratum fallback

// FARM-13
PropheticMinUniqueCiters:     5
PropheticMaxSingleCiterShare: 300_000   // 30%

// FARM-14
// Difficulty normalization — check getDomainConfidenceCeiling

// FARM-15
FalsificationBaseMultiplierBps: 1_000_000  // 1×
FalsificationAgeScaleBlocks:    342_720
FalsificationAgeMaxBonusBps:    1_000_000
FalsificationCiteScaleCount:    50
FalsificationCiteMaxBonusBps:   1_000_000
```

Verify each with:
```bash
grep "MinInformationGainBps\|RestatingPenaltyBps\|CitationConcentration\|PropheticMin\|FalsificationBase\|FalsificationAge\|FalsificationCite" x/knowledge/types/types.go
```

### 4. Check for Conflicts

FARM-11 creates `CitationSourceRecord` and FARM-13 reuses it. Verify they
use the same data structure and storage prefix:

```bash
grep "CitationSourceRecord\|CitationSourcePrefix" x/knowledge/ -r --include="*.go" | grep -v _test.go
```

### 5. Run

```bash
go test ./x/knowledge/... -count=1 -v -timeout 120s
go vet ./x/knowledge/...
go build ./...
```

## Commit
```
test(knowledge): FARM-10 through FARM-15 integration test

Verifies all six anti-farming defenses work correctly together.
Tests combined attack scenarios: axiom restating, sybil citation
rings, domain ceiling abuse, prophetic manipulation, difficulty
skew exploitation, and young-fact falsification predation.
```
