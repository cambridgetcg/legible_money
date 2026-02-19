# B14-2 — Billing Module: Simplify to Clean 4-Way Split

## Current State

The billing module (`x/billing/keeper/distribution.go`) has its own parallel
revenue split with a progressive ramp AND a 5-way research sub-split:

```
Billing Payment
├── 5-7% Research Share (ramps up over time)
│   └── 5-way: 77% research, 7% foundation, 7% community, 7% devs, 2% reserve
└── Remaining
    ├── 60% Provider
    ├── 30% Knowledge Pool (weighted by confidence × log2(citations))
    └── 10% Protocol (50% burn, 50% treasury)
```

This creates 6 separate module accounts:
- `treasury_research`, `treasury_foundation`, `treasury_community`
- `treasury_developers`, `treasury_reserve`, `treasury_protocol`

## Target State

```
Billing Payment (100%)
├── 55.0% → Provider (service operator)
├── 22.0% → Knowledge Pool (weighted by referenced facts)
├── 13.0% → Research Fund (module account: research_fund)
└── 10.0% → Protocol (5% burn + 5% treasury_protocol)
```

This matches the tree module's simplified split (B14-1) in philosophy:
builders/providers get majority, research gets a fair share, and there's
deflationary pressure.

## Changes

### 1. Kill the 5-Way Research Sub-Split

In `x/billing/keeper/distribution.go`:

**DELETE** the 5-way constants:
```go
// DELETE:
ResearchTreasury   = "treasury_research"
FoundationTreasury = "treasury_foundation"
CommunityTreasury  = "treasury_community"
DevelopersTreasury = "treasury_developers"
ReserveTreasury    = "treasury_reserve"
```

**KEEP only:**
```go
const (
    ProtocolTreasury = "treasury_protocol"
    ResearchFund     = "research_fund"
)
```

### 2. Rewrite CalculateDistribution

```go
func (k Keeper) CalculateDistribution(ctx sdk.Context, totalPayment *big.Int, factIds []string) *types.PaymentDistribution {
    params := k.GetParams(ctx)

    // Clean 4-way split
    providerBps := uint64(550000)    // 55%
    knowledgeBps := uint64(220000)   // 22%
    researchBps := uint64(130000)    // 13%
    // Burn+treasury = 10% (remainder)

    providerShare := new(big.Int).Mul(totalPayment, big.NewInt(int64(providerBps)))
    providerShare.Div(providerShare, bpsBasis)

    knowledgePool := new(big.Int).Mul(totalPayment, big.NewInt(int64(knowledgeBps)))
    knowledgePool.Div(knowledgePool, bpsBasis)

    researchShare := new(big.Int).Mul(totalPayment, big.NewInt(int64(researchBps)))
    researchShare.Div(researchShare, bpsBasis)

    // Protocol = remainder (absorbs rounding)
    protocolTotal := new(big.Int).Set(totalPayment)
    protocolTotal.Sub(protocolTotal, providerShare)
    protocolTotal.Sub(protocolTotal, knowledgePool)
    protocolTotal.Sub(protocolTotal, researchShare)

    // Protocol split: 50% burn, 50% treasury
    protocolBurn := new(big.Int).Div(protocolTotal, big.NewInt(2))
    protocolTreasury := new(big.Int).Sub(protocolTotal, protocolBurn)

    // Knowledge pool weighted distribution (keep existing logic)
    knowledgeEntries := k.calculateKnowledgePoolDistribution(ctx, knowledgePool, factIds)

    return &types.PaymentDistribution{
        TotalPayment:     totalPayment.String(),
        ProviderShare:    providerShare.String(),
        ResearchShare:    researchShare.String(),
        KnowledgePool:    knowledgeEntries,
        ProtocolBurn:     protocolBurn.String(),
        ProtocolTreasury: protocolTreasury.String(),
        // DELETE: ResearchSplit field entirely
    }
}
```

### 3. Simplify ExecuteDistribution

```go
func (k Keeper) ExecuteDistribution(ctx sdk.Context, callerAddr sdk.AccAddress, providerAddr sdk.AccAddress, distribution *types.PaymentDistribution) error {
    // 1. Research fund → single module account
    researchAmt := new(big.Int)
    researchAmt.SetString(distribution.ResearchShare, 10)
    if researchAmt.Sign() > 0 {
        coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(researchAmt)))
        if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, ResearchFund, coins); err != nil {
            return fmt.Errorf("research fund transfer failed: %w", err)
        }
    }

    // 2. Provider share directly
    providerAmt := new(big.Int)
    providerAmt.SetString(distribution.ProviderShare, 10)
    if providerAmt.Sign() > 0 {
        coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(providerAmt)))
        if err := k.bankKeeper.SendCoins(ctx, callerAddr, providerAddr, coins); err != nil {
            return fmt.Errorf("provider payment failed: %w", err)
        }
    }

    // 3. Knowledge pool to fact submitters
    for _, entry := range distribution.KnowledgePool {
        // ... keep existing knowledge pool logic ...
    }

    // 4. Burn
    burnAmt := new(big.Int)
    burnAmt.SetString(distribution.ProtocolBurn, 10)
    if burnAmt.Sign() > 0 {
        coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(burnAmt)))
        if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, types.ModuleName, coins); err != nil {
            return fmt.Errorf("burn transfer failed: %w", err)
        }
        if err := k.bankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
            return fmt.Errorf("burn failed: %w", err)
        }
    }

    // 5. Protocol treasury
    treasuryAmt := new(big.Int)
    treasuryAmt.SetString(distribution.ProtocolTreasury, 10)
    if treasuryAmt.Sign() > 0 {
        coins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(treasuryAmt)))
        if err := k.bankKeeper.SendCoinsFromAccountToModule(ctx, callerAddr, ProtocolTreasury, coins); err != nil {
            return fmt.Errorf("treasury transfer failed: %w", err)
        }
    }

    return nil
}
```

### 4. Kill Progressive Ramp

`GetEffectiveResearchShareBps()` is no longer needed — the research share
is a fixed percentage of the total, not a progressive ramp on top of a
different split. Remove the function or deprecate it.

### 5. Update Proto

In `x/billing/types/`:
- Remove `ResearchFundSplit` message from proto (the 5-way split type)
- Remove `research_split` field from `PaymentDistribution`
- Remove `FoundationAddress` from Params
- Remove `ResearchRampBlocks`, `ResearchMinBps` from Params (no more ramp)

### 6. Tests

```go
func TestSimplifiedBillingDistribution(t *testing.T)  // 55/22/13/10 split
func TestNoMore5WaySplit(t *testing.T)                 // ResearchSplit field gone
func TestBillingBurn(t *testing.T)                     // 5% actually burned
func TestKnowledgePoolPreserved(t *testing.T)          // Weighted distribution still works
func TestBillingModuleAccounts(t *testing.T)           // Uses research_fund + treasury_protocol
```

### 7. Run

```bash
go test ./x/billing/... -count=1 -v
go build ./...
```

## Commit
```
refactor(billing): simplify to clean 4-way split, kill 5-way research sub-split

Replace progressive ramp + 5-way research split with clean 55/22/13/10
(provider/knowledge/research/protocol). Eliminates 5 separate treasury
module accounts. Research fund receives 13% via single research_fund
module account. Knowledge pool weighted distribution preserved.
```
