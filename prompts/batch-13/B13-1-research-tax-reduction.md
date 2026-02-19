# B13-1 — Research Tax Unification: Single 3.33% Rate, No Double-Dipping

## Context
The audit revealed three critical issues with the current research tax:

1. **Double taxation on verification rewards**: Block rewards are taxed 7% by
   `vesting_rewards` when funding the verification pool, then the knowledge module
   taxes verification rewards AGAIN at 7% (`ResearchTaxRate` in confidence.go).
   Effective rate on verification income: ~13.5%.

2. **Citation and falsification rewards are untaxed**: The two largest growing
   revenue streams pay zero research tax, meaning the fund's share of total
   economic activity shrinks as the network matures.

3. **Three different rates across three modules**: `vesting_rewards` uses 70,000
   BPS, knowledge uses 70,000 BPS (hardcoded fallback), tree uses 234,000 BPS.

## The Fix: Single Tax Point, Unified Rate

### Principle
Tax once, at the point of disbursement to individuals. Never tax internal
module-to-module transfers. The research tax applies to all reward payouts
equally at 3.33%.

### 1. x/vesting_rewards/types/genesis.go — Primary Rate

```go
// Change:
ResearchShareBps: 70000, // 7%
// To:
ResearchShareBps: 33300, // 3.33%
```

This is the CANONICAL rate. All other modules should reference this or use
the same value.

### 2. x/knowledge/keeper/confidence.go — REMOVE Double Tax

The verification reward tax in `calculateRewardsAndSlashes()` must be REMOVED.
These rewards come from the `knowledge` module account, which was funded by
the verification pool, which was funded from ALREADY-TAXED block rewards.

```go
// REMOVE this entire block:
researchTaxRate := params.ResearchTaxRate
if researchTaxRate == 0 {
    researchTaxRate = 70000 // fallback: 7%
}
// ... and the tax deduction logic that follows

// REPLACE with: send full reward to validator (tax was already paid upstream)
if k.bankKeeper != nil && amount > 0 {
    rewardCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(int64(amount))))
    if err := k.RouteReward(ctx, valAddr, rewardCoins, "verification"); err != nil {
        k.Logger(ctx).Error("failed to route verification reward", "error", err)
    }
}
```

Also remove or deprecate the `ResearchTaxRate` field from the knowledge Params
proto. It's no longer used.

### 3. x/knowledge/keeper/citation.go — ADD Tax

Citation rewards are currently untaxed. They should be taxed at 3.33%
because they're new value flowing to individuals.

In `DistributeCitationRewards()`, before sending to the submitter:

```go
// Apply research tax to citation rewards (3.33%)
researchBps := k.GetResearchTaxBps(ctx) // reads from vesting_rewards or shared param
if researchBps > 0 {
    taxAmount := rewardAmount * researchBps / 1_000_000
    rewardAmount -= taxAmount
    if taxAmount > 0 {
        taxCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(int64(taxAmount))))
        k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "research_fund", taxCoins)
    }
}

// Then send taxed amount via RouteReward (for partnership routing)
```

### 4. x/knowledge/keeper/rounds.go — ADD Tax to Falsification

In `distributeFalsificationReward()`, same pattern:

```go
// Apply research tax to falsification reward
researchBps := k.GetResearchTaxBps(ctx)
if researchBps > 0 {
    taxAmount := safeMulDiv(rewardAmount, researchBps, 1_000_000)
    rewardAmount -= taxAmount
    if taxAmount > 0 {
        taxCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewInt(int64(taxAmount))))
        k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "research_fund", taxCoins)
    }
}
```

### 5. Shared Tax Rate Accessor

Add to knowledge keeper:

```go
// GetResearchTaxBps returns the unified research tax rate.
// Reads from vesting_rewards params if available, otherwise uses default.
func (k Keeper) GetResearchTaxBps(ctx sdk.Context) uint64 {
    if k.vestingRewardsKeeper != nil {
        params := k.vestingRewardsKeeper.GetParams(ctx)
        return params.ResearchShareBps
    }
    return 33300 // fallback: 3.33%
}
```

If wiring vestingRewardsKeeper is too invasive, store the rate in the
knowledge module's own params and keep it synced via governance. But
prefer reading from the canonical source.

### 6. Tree Module — Note Discrepancy

The tree module's 23.4% research split is for service revenue (user-paid,
not minted). Add a TODO comment but don't change it in this session:

```go
// TODO: Tree service revenue research split (23.4%) is independent from
// the 3.33% block reward research tax. These are different economic
// contexts (service payments vs minting). Evaluate unification in v0.3.
ResearchFundBp: 234000, // 23.4% of service revenue
```

### 7. Update Comments Everywhere

```bash
grep -rn "7%\|70000.*research\|research.*7%\|research.*70000" x/ --include="*.go" | grep -v _test.go | grep -v .pb.go
```

Update all comments to reflect 3.33%.

### 8. Impact Summary (Corrected)

**Before (with double taxation):**
| Stream | Effective Research Tax |
|--------|----------------------|
| Block producer rewards | 7% |
| Verification rewards | ~13.5% (double-taxed) |
| Citation rewards | 0% |
| Falsification rewards | 0% |

**After (unified):**
| Stream | Effective Research Tax |
|--------|----------------------|
| Block producer rewards | 3.33% (at minting) |
| Verification rewards | 3.33% (at minting only, no double-dip) |
| Citation rewards | 3.33% (new) |
| Falsification rewards | 3.33% (new) |

Every reward stream taxed equally. No loopholes. No double-dipping.

### 9. Tests

```go
func TestUnifiedResearchTax(t *testing.T)              // All streams taxed at 3.33%
func TestNoDoubleTaxOnVerification(t *testing.T)       // Verification rewards not taxed twice
func TestCitationRewardsTaxed(t *testing.T)            // Citations now taxed
func TestFalsificationRewardsTaxed(t *testing.T)       // Falsification now taxed
func TestResearchTaxRateConsistency(t *testing.T)      // All modules use same rate
```

### 10. Run

```bash
go test ./x/vesting_rewards/... -count=1 -v
go test ./x/knowledge/... -count=1 -v
go build ./...
```

## Commit
```
feat(economics): unified 3.33% research tax — no double-dipping, no loopholes

Eliminates double taxation on verification rewards (was ~13.5%, now 3.33%).
Adds research tax to citation and falsification rewards (were 0%).
Single canonical rate (33,300 BPS) used across all modules. Every reward
stream taxed equally at the point of individual disbursement.
```
