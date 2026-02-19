# B13-1 — Research Tax 7% → 3.33% Across All Modules

## Context
The research tax currently takes 7% (70,000 BPS) from all reward streams.
This is being reduced to 3.33% (33,300 BPS) to put more capital into
verification rewards (truth-seeking incentives) and block production
(validator incentives).

## Files to Change

### 1. x/vesting_rewards/types/genesis.go
The primary research share parameter:
```go
// Change:
ResearchShareBps: 70000, // 7%
// To:
ResearchShareBps: 33300, // 3.33%
```

### 2. x/vesting_rewards/keeper/rewards.go
Check if `ResearchShareBps` is used directly or only via params. If there
are any hardcoded `70000` values, update those too.

```bash
grep -n "70000\|7%" x/vesting_rewards/keeper/rewards.go
```

Update comments that mention "7%" to "3.33%".

### 3. x/tree/types/genesis.go
The tree module has its own research fund split:
```go
// Current:
ResearchFundBp: 234000, // 23.4%
// This is tree-specific — evaluate whether this should also change.
```

The tree module's 23.4% is a different context (service revenue, not block
rewards). Decision: **keep the tree split as-is for now** — service revenue
is user-paid, not minted, so the research share from service calls is a
different economic question. Add a TODO comment noting the discrepancy.

### 4. x/vesting_rewards/types/keys.go
Update the comment on `VerificationPoolShareBps`:
```go
// The verification pool now receives 50% of 96.67% (previously 50% of 93%).
```

### 5. x/knowledge/keeper/confidence.go
Check if the knowledge module references the research tax rate directly:
```bash
grep -n "ResearchTaxRate\|research.*tax\|70000" x/knowledge/keeper/confidence.go
```
Update any hardcoded references.

### 6. Documentation
Update any docs that mention the 7% figure:
```bash
grep -rn "7%\|7 percent\|seven percent\|research.*7" docs/ README.md
```

### 7. Validation
The `ResearchShareBps` must be ≤ 1,000,000 and the genesis validation
already checks this. Verify:
```bash
grep "ResearchShareBps" x/vesting_rewards/types/genesis.go
```

### 8. Impact Summary

| Metric | Before (7%) | After (3.33%) | Change |
|--------|------------|---------------|--------|
| Research fund | 7.00% | 3.33% | -3.67% |
| Verification pool | 32.55% | 33.83% | +1.28% |
| Compute pool | 13.95% | 14.50% | +0.55% |
| Block producer | 46.50% | 48.34% | +1.84% |

More goes to truth-seeking (verification) and security (validators).

### 9. Tests

Update any test that asserts the 7% split:
```bash
grep -rn "70000\|ResearchShareBps.*70" x/vesting_rewards/ --include="*_test.go"
```

Change expected values from 70000 to 33300.

### 10. Run

```bash
go test ./x/vesting_rewards/... -count=1 -v
go test ./x/knowledge/... -count=1
go build ./...
```

## Commit
```
feat(economics): reduce research tax 7% → 3.33%

Research fund share reduced from 70,000 BPS (7%) to 33,300 BPS (3.33%).
Freed capital flows to verification rewards (+1.28%) and block producers
(+1.84%), strengthening truth-seeking and validator incentives.
Tree module service revenue split unchanged (separate economic context).
```
