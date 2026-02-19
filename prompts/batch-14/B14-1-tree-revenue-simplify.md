# B14-1 — Tree Module: Simplify to Clean 4-Way Split

## Current State (Overcomplicated)

Tree service revenue currently has a 4-way primary split PLUS a 5-way
sub-split on the evidence tax portion:

```
Service Payment (100%)
├── 22.0% Evidence Tax → 5-way: 77% research, 7% foundation, 7% community, 7% devs, 2% reserve
├── 39.0% Contributors
├── 23.4% Research Fund
└── 15.6% Protocol Treasury
```

Research fund effectively gets 40.3% (more than the builders). The 5-way
evidence tax sub-split sends to 5 different addresses/accounts for no
real governance benefit — they're all controlled the same way.

## Target State (Clean)

```
Service Payment (100%)
├── 55.0% → Contributors (pro-rata by tasks completed)
├── 22.0% → Protocol Treasury (module account: treasury_protocol)
├── 13.0% → Research Fund (module account: research_fund)
└── 10.0% → Burn (deflationary)
```

## Changes

### 1. x/tree/types/genesis.go — Update Defaults

```go
func DefaultParams() *Params {
    return &Params{
        // ... existing params ...
        
        // Old 4+5 split (DELETE):
        // EvidenceTaxBp:      220000,
        // ContributorsBp:     390000,
        // ResearchFundBp:     234000,
        // ProtocolTreasuryBp: 156000,
        
        // New clean 4-way split:
        ContributorsBp:     550000,  // 55%
        ProtocolTreasuryBp: 220000,  // 22%
        ResearchFundBp:     130000,  // 13%
        BurnBp:             100000,  // 10%
    }
}
```

Remove `EvidenceTaxBp` from the params entirely. Update the `Validate()`
function to check the new 4-way sum:

```go
func (p *Params) Validate() error {
    // ...
    bpSum := p.ContributorsBp + p.ProtocolTreasuryBp + p.ResearchFundBp + p.BurnBp
    if bpSum != 0 && bpSum != 1000000 {
        return fmt.Errorf("revenue basis points must sum to 1000000, got %d", bpSum)
    }
    return nil
}
```

### 2. x/tree/keeper/revenue.go — Simplify CalculateRevenue

Replace the entire function:

```go
type RevenueDistribution struct {
    ContributorPool  int64
    ResearchFund     int64
    ProtocolTreasury int64
    Burn             int64
    ContributorShares []ContributorShare
}

func CalculateRevenue(
    total int64,
    contributorsBp, treasuryBp, researchBp, burnBp uint32,
    contributors []*types.ContributorRecord,
) RevenueDistribution {
    if total <= 0 {
        return RevenueDistribution{}
    }

    dist := RevenueDistribution{}
    dist.ContributorPool = total * int64(contributorsBp) / BpDenominator
    dist.ResearchFund = total * int64(researchBp) / BpDenominator
    dist.Burn = total * int64(burnBp) / BpDenominator
    // Treasury absorbs remainder (rounding dust)
    dist.ProtocolTreasury = total - dist.ContributorPool - dist.ResearchFund - dist.Burn

    // Contributor pro-rata split (unchanged logic)
    // ... keep existing contributor distribution code ...

    return dist
}
```

**DELETE** all evidence tax sub-split fields and constants:
- `TaxResearchBp`, `TaxFoundationBp`, `TaxCommunityBp`, `TaxDevelopersBp`, `TaxReserveBp`
- `dist.TaxResearch`, `dist.TaxFoundation`, `dist.TaxCommunity`, `dist.TaxDevelopers`, `dist.TaxReserve`
- `dist.EvidenceTax`

### 3. x/tree/keeper/msg_server.go — Simplify Distribution Code

In `CallService()`, replace the distribution section:

```go
params := m.Keeper.GetParams(ctx)

dist := CalculateRevenue(
    price.Int64(),
    params.ContributorsBp,
    params.ProtocolTreasuryBp,
    params.ResearchFundBp,
    params.BurnBp,
    project.Contributors,
)

// Contributors
for _, share := range dist.ContributorShares {
    if share.Amount > 0 {
        addr, _ := sdk.AccAddressFromBech32(share.Address)
        coins := sdk.NewCoins(sdk.NewCoin("ulgm", math.NewInt(share.Amount)))
        if err := m.Keeper.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, coins); err != nil {
            return nil, fmt.Errorf("contributor payment failed: %w", err)
        }
    }
}

// Research fund → module account (NOT configurable address)
if dist.ResearchFund > 0 {
    coins := sdk.NewCoins(sdk.NewCoin("ulgm", math.NewInt(dist.ResearchFund)))
    if err := m.Keeper.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "research_fund", coins); err != nil {
        return nil, fmt.Errorf("research fund transfer failed: %w", err)
    }
}

// Protocol treasury → module account
if dist.ProtocolTreasury > 0 {
    coins := sdk.NewCoins(sdk.NewCoin("ulgm", math.NewInt(dist.ProtocolTreasury)))
    if err := m.Keeper.bankKeeper.SendCoinsFromModuleToModule(ctx, types.ModuleName, "treasury_protocol", coins); err != nil {
        return nil, fmt.Errorf("treasury transfer failed: %w", err)
    }
}

// Burn
if dist.Burn > 0 {
    coins := sdk.NewCoins(sdk.NewCoin("ulgm", math.NewInt(dist.Burn)))
    if err := m.Keeper.bankKeeper.BurnCoins(ctx, types.ModuleName, coins); err != nil {
        return nil, fmt.Errorf("burn failed: %w", err)
    }
}
```

**DELETE** the entire `taxRecipients` loop and the `ResearchFundAddr`/`ProtocolTreasuryAddr` usage.

### 4. Remove Configurable Addresses

The tree module currently sends to configurable bech32 addresses
(`ResearchFundAddr`, `ProtocolTreasuryAddr`). Replace with module accounts:
- `research_fund` (already exists)
- `treasury_protocol` (already exists in billing module)

Deprecate or remove the address fields from the proto:
```go
// In params proto: mark as deprecated
// string research_fund_addr = 14 [deprecated = true];
// string protocol_treasury_addr = 15 [deprecated = true];
```

For proto backwards compat, keep the fields but ignore them.

### 5. Proto Changes

If `EvidenceTaxBp` is a proto field, mark it deprecated:
```protobuf
uint32 evidence_tax_bp = 9 [deprecated = true]; // replaced by 4-way split
```

Add `BurnBp`:
```protobuf
uint32 burn_bp = 16; // burn share of service revenue (default 100000 = 10%)
```

### 6. Tests

```go
func TestSimplifiedRevenue(t *testing.T)          // 55/22/13/10 split
func TestRevenueBurnActuallyBurns(t *testing.T)   // Supply decreases
func TestNoMoreEvidenceTax(t *testing.T)          // Old 5-way sub-split removed
func TestRevenueModuleAccounts(t *testing.T)      // Sends to module accounts not addresses
func TestNoContributorsRedirect(t *testing.T)     // 0 contributors → treasury gets their share
func TestRevenueBpsValidation(t *testing.T)       // Must sum to 1M
```

### 7. Run

```bash
go test ./x/tree/... -count=1 -v
go test ./x/tree/keeper/... -v -run TestRevenue
go build ./...
```

## Commit
```
refactor(tree): simplify service revenue to clean 4-way split

Replace overcomplicated 4+5 split (evidence tax sub-split) with
clean 55/22/13/10 (contributors/treasury/research/burn). Contributors
get majority. Evidence tax concept eliminated. Configurable addresses
replaced with module accounts. 10% burn adds deflation.
```
