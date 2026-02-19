# B14-3 — Founder Auto-Split on Service Revenue Research Deposits

## Context
Batch 13 added the founder auto-split (7% of research fund) for block
rewards and tx fees in `x/vesting_rewards`. But service revenue sends to
the `research_fund` module account via the tree and billing modules —
those deposits don't pass through vesting_rewards.

The founder's 7% share must apply to ALL research fund deposits, regardless
of source. Otherwise service revenue research deposits bypass the founder
split, creating an inconsistency.

## The Fix

### Option A: Intercept at research_fund module account (cleanest)

Create a hook or wrapper that applies the founder split whenever ANY module
deposits into `research_fund`. This way, tree module, billing module, and
any future module all get the split automatically.

### Option B: Split at each deposit point (fragile)

Add founder split logic in tree/msg_server.go and billing/distribution.go.
This works but requires updating every new module that deposits to research_fund.

### Recommendation: Option A

### Implementation

#### 1. Create ResearchFundRouter

In `x/vesting_rewards/keeper/` (or a shared location), add:

```go
// DepositToResearchFund deposits amount to the research fund, applying the
// founder auto-split. This is the ONLY way modules should send to research_fund.
//
// Split: (1 - FounderShareBps) → research_fund, FounderShareBps → founder address
func (k Keeper) DepositToResearchFund(ctx sdk.Context, fromModule string, amount sdk.Coins) error {
    params := k.GetParams(ctx)

    totalAmount := amount.AmountOf("ulgm")
    if !totalAmount.IsPositive() {
        return nil
    }

    founderAmount := sdkmath.ZeroInt()
    if params.FounderShareBps > 0 && params.FounderAddress != "" {
        // Check governance activation sunset
        config := k.GetResearchFundConfig(ctx) // if accessible
        if config == nil || config.GovernanceActivationHeight == 0 ||
            uint64(ctx.BlockHeight()) < config.GovernanceActivationHeight {
            founderAmount = totalAmount.MulRaw(int64(params.FounderShareBps)).QuoRaw(1_000_000)
        }
    }

    researchAmount := totalAmount.Sub(founderAmount)

    // Send research portion to research_fund module
    if researchAmount.IsPositive() {
        researchCoins := sdk.NewCoins(sdk.NewCoin("ulgm", researchAmount))
        if err := k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromModule, "research_fund", researchCoins); err != nil {
            return err
        }
    }

    // Send founder portion directly
    if founderAmount.IsPositive() && params.FounderAddress != "" {
        founderAddr, err := sdk.AccAddressFromBech32(params.FounderAddress)
        if err != nil {
            // If founder address is invalid, all goes to research fund
            allCoins := sdk.NewCoins(sdk.NewCoin("ulgm", totalAmount))
            return k.bankKeeper.SendCoinsFromModuleToModule(ctx, fromModule, "research_fund", allCoins)
        }
        founderCoins := sdk.NewCoins(sdk.NewCoin("ulgm", founderAmount))
        if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, fromModule, founderAddr, founderCoins); err != nil {
            return err
        }
    }

    ctx.EventManager().EmitEvent(sdk.NewEvent(
        "research_fund_deposit",
        sdk.NewAttribute("source_module", fromModule),
        sdk.NewAttribute("total", totalAmount.String()),
        sdk.NewAttribute("research", researchAmount.String()),
        sdk.NewAttribute("founder", founderAmount.String()),
    ))

    return nil
}
```

#### 2. Expose via Interface

In a shared types package or via keeper interface:

```go
// ResearchFundDepositor routes deposits to the research fund with founder split.
type ResearchFundDepositor interface {
    DepositToResearchFund(ctx sdk.Context, fromModule string, amount sdk.Coins) error
}
```

#### 3. Wire into Tree Module

In `x/tree/keeper/msg_server.go`, replace:
```go
k.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "research_fund", coins)
```
With:
```go
k.vestingRewardsKeeper.DepositToResearchFund(ctx, types.ModuleName, coins)
```

Add `vestingRewardsKeeper` to the tree keeper constructor.

#### 4. Wire into Billing Module

Same pattern in `x/billing/keeper/distribution.go`:
```go
// Replace:
k.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, ResearchFund, coins)
// With:
// First: move from caller to billing module
k.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, types.ModuleName, coins)
// Then: route through depositor
k.vestingRewardsKeeper.DepositToResearchFund(ctx, types.ModuleName, coins)
```

#### 5. Wire into Knowledge Module

The knowledge module's citation and falsification tax (from B13-1) also
sends to research_fund. Replace those with the depositor too:
```go
// In citation.go and rounds.go:
k.vestingRewardsKeeper.DepositToResearchFund(ctx, types.ModuleName, taxCoins)
```

#### 6. Single Source of Truth

After this change, EVERY deposit to research_fund goes through
`DepositToResearchFund()`. The founder split is guaranteed regardless of
which module generates the revenue. Adding a new module that generates
research tax? Just call `DepositToResearchFund()`. Done.

### Tests

```go
func TestFounderSplitOnTreeRevenue(t *testing.T)      // Tree service revenue → founder gets 7%
func TestFounderSplitOnBillingRevenue(t *testing.T)    // Billing payment → founder gets 7%
func TestFounderSplitOnKnowledgeTax(t *testing.T)      // Citation/falsification tax → founder gets 7%
func TestFounderSplitOnBlockReward(t *testing.T)       // Block rewards → founder gets 7% (existing)
func TestFounderSplitSunset(t *testing.T)              // After governance activation → no split
func TestAllResearchDepositsRouted(t *testing.T)       // No direct sends to research_fund
```

### Run

```bash
go test ./x/vesting_rewards/... -count=1 -v
go test ./x/tree/... -count=1 -v
go test ./x/billing/... -count=1 -v
go test ./x/knowledge/... -count=1 -v
go build ./...
```

## Commit
```
feat(economics): unified research fund depositor with guaranteed founder split

All research fund deposits from all modules now route through
DepositToResearchFund(), which applies the 7% founder auto-split
consistently. Block rewards, tx fees, service revenue, citation tax,
and falsification tax all treated equally. Single point of control
for the founder split, with governance sunset respected.
```
