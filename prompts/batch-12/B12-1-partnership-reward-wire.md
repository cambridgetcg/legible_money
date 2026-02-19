# B12-1 — Wire Knowledge Rewards Through Partnership System

## The Problem
The knowledge module distributes verification rewards and citation rewards
directly to individual addresses (`bankKeeper.SendCoinsFromModuleToAccount`).
The partnership module has `DistributeReward()` that splits rewards 22% pot /
78% split. But NOTHING connects them. A partnered agent earns rewards that
bypass the partnership entirely.

## Current Reward Distribution Points

### 1. Verification rewards (x/knowledge/keeper/confidence.go)
In `calculateRewardsAndSlashes()`:
```go
// Transfer reward from module account to validator (after research tax)
if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, valAddr, rewardCoins); err != nil {
```
This sends directly to the validator address — no partnership check.

### 2. Citation rewards (x/knowledge/keeper/citation.go)
In `DistributeCitationRewards()`:
```go
// Send citation reward to cited fact's submitter
k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, submitterAddr, rewardCoins)
```
Sends directly to the fact submitter — no partnership check.

### 3. Falsification rewards (x/knowledge/keeper/rounds.go)
In `distributeFalsificationReward()`:
```go
k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, cfSubmitterAddr, rewardCoins)
```
Direct to challenger — no partnership check.

## The Fix

### 1. Add PartnershipKeeper to Knowledge Module

In `x/knowledge/types/expected_keepers.go`, add:

```go
// PartnershipKeeper is used to route rewards through partnerships.
type PartnershipKeeper interface {
    // GetActivePartnershipForAddress returns the active partnership for a participant.
    // Returns nil, false if no active partnership exists.
    GetActivePartnershipForAddress(ctx sdk.Context, addr string) (*partnershiptypes.Partnership, bool)

    // DistributeReward splits a reward through the partnership economics.
    DistributeReward(ctx sdk.Context, partnership *partnershiptypes.Partnership, grossAmount string, sourceType string) (humanShare, agentShare, commonPotAdd string, err error)
}
```

Wire this into the Knowledge keeper constructor. Make it optional (nil check)
so the module works standalone in tests.

### 2. Create RewardRouter Helper

In `x/knowledge/keeper/reward_router.go`:

```go
// RouteReward sends reward to addr, routing through partnership if one exists.
// If addr is in an active partnership, the reward is split per partnership economics.
// If not, the full reward goes directly to addr.
func (k Keeper) RouteReward(ctx sdk.Context, addr sdk.AccAddress, amount sdk.Coins, sourceType string) error {
    if k.partnershipKeeper == nil {
        // No partnership module — direct transfer
        return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, amount)
    }

    partnership, found := k.partnershipKeeper.GetActivePartnershipForAddress(ctx, addr.String())
    if !found {
        // No active partnership — direct transfer
        return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, amount)
    }

    // Route through partnership: 22% pot, 78% split
    grossAmount := amount.AmountOf("ulgm").String()
    humanShare, agentShare, _, err := k.partnershipKeeper.DistributeReward(
        ctx, partnership, grossAmount, sourceType,
    )
    if err != nil {
        // Partnership distribution failed — fall back to direct
        k.Logger(ctx).Error("partnership reward routing failed, falling back to direct",
            "addr", addr.String(), "error", err)
        return k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, addr, amount)
    }

    // Send individual shares
    humanAddr, _ := sdk.AccAddressFromBech32(partnership.HumanAddr)
    agentAddr, _ := sdk.AccAddressFromBech32(partnership.AgentAddr)

    if humanAmt, ok := sdkmath.NewIntFromString(humanShare); ok && humanAmt.IsPositive() {
        humanCoins := sdk.NewCoins(sdk.NewCoin("ulgm", humanAmt))
        if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, humanAddr, humanCoins); err != nil {
            return err
        }
    }
    if agentAmt, ok := sdkmath.NewIntFromString(agentShare); ok && agentAmt.IsPositive() {
        agentCoins := sdk.NewCoins(sdk.NewCoin("ulgm", agentAmt))
        if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, agentAddr, agentCoins); err != nil {
            return err
        }
    }

    // Common pot share stays in partnership module account (already handled by DistributeReward)

    ctx.EventManager().EmitEvent(sdk.NewEvent(
        "partnership_reward_routed",
        sdk.NewAttribute("partnership_id", partnership.Id),
        sdk.NewAttribute("source", sourceType),
        sdk.NewAttribute("gross", grossAmount),
        sdk.NewAttribute("human_share", humanShare),
        sdk.NewAttribute("agent_share", agentShare),
    ))

    return nil
}
```

### 3. Replace Direct Transfers

In `confidence.go`, replace:
```go
k.bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, valAddr, rewardCoins)
```
With:
```go
k.RouteReward(ctx, valAddr, rewardCoins, "verification")
```

In `citation.go`, replace direct sends with:
```go
k.RouteReward(ctx, submitterAddr, rewardCoins, "citation")
```

In `rounds.go` (falsification), replace with:
```go
k.RouteReward(ctx, cfSubmitterAddr, rewardCoins, "falsification")
```

### 4. Partnership Keeper Method

In `x/partnerships/keeper/keeper.go`, add:

```go
// GetActivePartnershipForAddress finds an active partnership where addr is a participant.
// Returns the first active partnership found (an address should have at most one active).
func (k Keeper) GetActivePartnershipForAddress(ctx sdk.Context, addr string) (*types.Partnership, bool) {
    // Check human partnerships
    for _, id := range k.GetPartnershipsByHuman(ctx, addr) {
        if p, found := k.GetPartnership(ctx, id); found && p.Status == types.StatusActive {
            return p, true
        }
    }
    // Check agent partnerships
    for _, id := range k.GetPartnershipsByAgent(ctx, addr) {
        if p, found := k.GetPartnership(ctx, id); found && p.Status == types.StatusActive {
            return p, true
        }
    }
    return nil, false
}
```

### 5. Edge Cases

- **Multiple active partnerships:** Currently the code allows only one non-dissolved
  partnership per pair, but an address could have partnerships with multiple counterparts.
  `GetActivePartnershipForAddress` returns the first found. For v0.1, this is fine.
  Add a TODO for multi-partnership routing in v0.2.
- **Partnership suspended (frozen):** During a safety freeze, should rewards still be
  split? Yes — the freeze prevents pot *spending*, not pot *filling*. Rewards continue
  accumulating; only withdrawals are blocked.
- **Genesis submitter:** Genesis axioms have submitter="genesis" — the module account.
  `RouteReward` should short-circuit for module accounts (no partnership lookup).

### 6. Tests

```go
func TestRouteReward_NoPartnership(t *testing.T)       // Direct transfer
func TestRouteReward_WithPartnership(t *testing.T)     // Split applied
func TestRouteReward_FrozenPartnership(t *testing.T)   // Still splits (freeze blocks spending not earning)
func TestRouteReward_ModuleAccount(t *testing.T)       // Short-circuit for genesis/module
func TestRouteReward_FallbackOnError(t *testing.T)     // Direct transfer if partnership fails
func TestVerificationRewardRouted(t *testing.T)        // Integration: verify → partnership split
func TestCitationRewardRouted(t *testing.T)            // Integration: cite → partnership split
```

### 7. Run

```bash
go test ./x/knowledge/keeper/... -v -run TestRouteReward
go test ./x/knowledge/... -count=1
go test ./x/partnerships/... -count=1
```

## Commit
```
feat(knowledge): route rewards through partnership system

Knowledge module rewards (verification, citation, falsification) now
check if the recipient is in an active partnership. If so, rewards
flow through DistributeReward: 22% to common pot, 78% split per
partnership terms (default 50/50). Partnerships are no longer
economically disconnected from knowledge work.
```
