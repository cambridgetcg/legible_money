# B13-2 — Founder Auto-Split from Research Fund

## Context
7% of all research fund deposits (which total 3.33% of all reward streams)
automatically routes to the founder's address for operational expenses.
This is 0.23% of total rewards.

After B13-1, research tax is collected at four points:
1. Block reward minting (vesting_rewards)
2. Transaction fee sweep (vesting_rewards)
3. Citation reward disbursement (knowledge)
4. Falsification reward disbursement (knowledge)

The founder split should apply to ALL four, at the point the tax enters
the research_fund account. The cleanest approach: intercept at the
research_fund module account, not at each collection point.

## Implementation

### 1. Add Parameters

In `x/vesting_rewards/types/genesis.go`, add to Params:

```go
// FounderShareBps is the share of the research fund that auto-routes
// to the founder's address for operational expenses.
// 70,000 BPS = 7% of research fund = 0.23% of total rewards.
FounderShareBps: 70000,

// FounderAddress is the bech32 address that receives the founder share.
// Set at genesis. Empty string disables founder share.
FounderAddress: "",
```

Add to the Params struct in the proto or Go types:
```go
FounderShareBps uint64 `json:"founder_share_bps"`
FounderAddress  string `json:"founder_address"`
```

### 2. Update RouteReward in vesting_rewards

In `x/vesting_rewards/keeper/rewards.go`, in `RouteReward()`:

After calculating `researchShare`, split it:

```go
// Calculate founder's operational share (7% of research)
founderShare := new(big.Int)
if params.FounderShareBps > 0 && params.FounderAddress != "" {
    founderShare = new(big.Int).Mul(researchShare, big.NewInt(int64(params.FounderShareBps)))
    founderShare.Div(founderShare, big.NewInt(1000000))
    // Reduce research share by founder's portion
    researchShare.Sub(researchShare, founderShare)
}

routing := &types.RewardRouting{
    Source:          string(source),
    OriginalAmount:  amount,
    ResearchShare:   researchShare.String(),
    FounderShare:    founderShare.String(),
    RecipientAmount: recipientAmount.String(),
    Recipient:       recipient,
    FactId:          factId,
    BlockNumber:     uint64(ctx.BlockHeight()),
}
```

### 3. Update DistributeBlockReward

In the same file, in `DistributeBlockReward()`, after sending to research_fund:

```go
// Send founder operational share
if founderBig.Sign() > 0 && params.FounderAddress != "" {
    founderAddr, addrErr := sdk.AccAddressFromBech32(params.FounderAddress)
    if addrErr == nil {
        founderCoins := sdk.NewCoins(sdk.NewCoin("ulgm", sdkmath.NewIntFromBigInt(founderBig)))
        if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, founderAddr, founderCoins); err != nil {
            k.Logger(ctx).Error("failed to send founder share", "error", err)
        }
    }
}
```

### 4. Update RouteFees

Same pattern in `RouteFees()` — the 3.33% fee sweep should also split
the founder portion:

```go
// Split founder share from research amount
founderAmount := researchAmount.MulRaw(int64(params.FounderShareBps)).QuoRaw(1000000)
actualResearch := researchAmount.Sub(founderAmount)

// Send research portion to research_fund
researchCoins := sdk.NewCoins(sdk.NewCoin("ulgm", actualResearch))
k.bankKeeper.SendCoinsFromModuleToModule(ctx, authtypes.FeeCollectorName, ResearchFundModuleName, researchCoins)

// Send founder portion directly
if founderAmount.IsPositive() && params.FounderAddress != "" {
    founderAddr, _ := sdk.AccAddressFromBech32(params.FounderAddress)
    founderCoins := sdk.NewCoins(sdk.NewCoin("ulgm", founderAmount))
    k.bankKeeper.SendCoinsFromModuleToAccount(ctx, authtypes.FeeCollectorName, founderAddr, founderCoins)
}
```

### 5. Add to BlockRewardDistribution

Update the distribution record struct to include founder share:
```go
type BlockRewardDistribution struct {
    // ... existing fields ...
    FounderShare string `json:"founder_share"`
}
```

### 6. Sunset Integration

The founder share should respect the governance activation from B12-2.
When `GovernanceActivationHeight` is reached, the founder auto-split
stops and the full research share goes to `research_fund`:

```go
// Check governance activation
if config.GovernanceActivationHeight > 0 && uint64(ctx.BlockHeight()) >= config.GovernanceActivationHeight {
    // Governance active — no founder auto-split
    founderShare = big.NewInt(0)
}
```

### 7. Events

Emit transparent events:
```go
ctx.EventManager().EmitEvent(sdk.NewEvent(
    "founder_share_distributed",
    sdk.NewAttribute("amount", founderBig.String()),
    sdk.NewAttribute("recipient", params.FounderAddress),
    sdk.NewAttribute("block", fmt.Sprintf("%d", height)),
))
```

### 8. Query

Extend the research fund query to show founder stats:
```bash
legbled query vesting-rewards research-fund-info
```
Returns:
```json
{
    "research_fund_balance": "...",
    "founder_address": "lgm1...",
    "founder_share_bps": 70000,
    "total_founder_received": "...",
    "governance_activation_height": 0
}
```

### 9. Genesis Script Integration

In `scripts/genesis-777.sh`, set the founder address:
```bash
# Set founder address for research fund operational split
FOUNDER_ADDR=$($BINARY keys show validator -a --keyring-backend test --home "$NODE_HOME")
# Update genesis with founder address in vesting_rewards params
```

For production genesis, this will be Yu's address.

### 10. Tests

```go
func TestFounderAutoSplit(t *testing.T)           // 7% of research goes to founder
func TestFounderSplitDisabled(t *testing.T)       // Empty address = no split
func TestFounderSplitSunset(t *testing.T)         // After governance activation = no split
func TestFounderShareInFeeRouting(t *testing.T)   // Fee sweep also splits
func TestFounderShareAccounting(t *testing.T)     // Total distributed matches events
```

### 11. Run

```bash
go test ./x/vesting_rewards/... -count=1 -v
go build ./...
```

## Commit
```
feat(economics): founder auto-split — 7% of research fund for operations

Adds automatic founder share: 7% of the 3.33% research tax (0.23% of
total rewards) routes to founder address on every block reward and fee
sweep. Fully transparent via events and queries. Respects governance
activation sunset — stops automatically when community governance begins.
```
