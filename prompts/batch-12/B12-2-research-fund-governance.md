# B12-2 — Research Fund: Founder-Directed Treasury (Phase 1)

## Design Decision
The research fund (7% tax on verification rewards) will be founder-directed
during the bootstrap phase. Full proposal/vote/disburse governance is deferred
to v0.3.0 or 6 months post-mainnet (whichever comes first).

**Rationale:** The chain needs operational flexibility during early development.
A founder who built the philosophy, the axioms, and the economics is the best
steward of research funding until a community of contributors exists to govern it.

## Implementation

### 1. Founder Treasury Authority

In `x/knowledge/types/types.go`, add:

```go
// ResearchFundConfig governs the research fund treasury.
type ResearchFundConfig struct {
    // Authority is the address that can disburse from the research fund.
    // During Phase 1 (bootstrap), this is the founder's address.
    // After GovernanceActivationHeight, disbursement requires governance proposal.
    Authority string `json:"authority"`

    // GovernanceActivationHeight: when governance takes over (0 = not yet set).
    // After this height, Authority can no longer unilaterally disburse.
    GovernanceActivationHeight uint64 `json:"governance_activation_height"`

    // MaxDisbursementBps: max single disbursement as fraction of total fund.
    // Even the founder can't drain the whole fund in one tx.
    MaxDisbursementBps uint64 `json:"max_disbursement_bps"` // default 200,000 (20%)

    // DisbursementCooldownBlocks: minimum blocks between disbursements.
    DisbursementCooldownBlocks uint64 `json:"disbursement_cooldown_blocks"` // default 34272 (~1 day)

    // TotalDisbursed: running total for transparency.
    TotalDisbursed string `json:"total_disbursed"`

    // LastDisbursementHeight: when the last disbursement occurred.
    LastDisbursementHeight uint64 `json:"last_disbursement_height"`
}
```

### 2. Genesis Configuration

In `x/knowledge/types/genesis.go`, set defaults:

```go
ResearchFundConfig: &ResearchFundConfig{
    Authority:                  "", // Set to founder address at chain init
    GovernanceActivationHeight: 0,  // Not yet scheduled
    MaxDisbursementBps:         200_000, // 20% per tx
    DisbursementCooldownBlocks: 34_272,  // ~1 day
    TotalDisbursed:             "0",
    LastDisbursementHeight:     0,
},
```

### 3. Disbursement Message

```go
type MsgDisburseResearchFund struct {
    Authority   string `json:"authority"`    // Must match config.Authority
    Amount      string `json:"amount"`       // ulgm
    Recipient   string `json:"recipient"`    // address or partnership ID
    Purpose     string `json:"purpose"`      // Required: human-readable reason
    Domain      string `json:"domain"`       // Optional: knowledge domain served
}
```

### 4. Keeper Implementation

```go
func (k Keeper) DisburseResearchFund(ctx sdk.Context, msg *MsgDisburseResearchFund) error {
    config := k.GetResearchFundConfig(ctx)

    // Phase check: if past governance activation, reject founder disbursement
    if config.GovernanceActivationHeight > 0 && uint64(ctx.BlockHeight()) >= config.GovernanceActivationHeight {
        return fmt.Errorf("governance phase active: founder disbursement no longer permitted")
    }

    // Authority check
    if msg.Authority != config.Authority {
        return fmt.Errorf("unauthorized: only %s can disburse", config.Authority)
    }

    // Cooldown check
    currentHeight := uint64(ctx.BlockHeight())
    if config.LastDisbursementHeight > 0 &&
        currentHeight < config.LastDisbursementHeight + config.DisbursementCooldownBlocks {
        return fmt.Errorf("cooldown active: next disbursement at block %d",
            config.LastDisbursementHeight + config.DisbursementCooldownBlocks)
    }

    // Amount check: max 20% of fund per tx
    fundBalance := k.bankKeeper.GetBalance(ctx, k.GetResearchFundModuleAddress(), "ulgm")
    amount, ok := sdkmath.NewIntFromString(msg.Amount)
    if !ok || !amount.IsPositive() {
        return fmt.Errorf("invalid amount")
    }
    maxAmount := fundBalance.Amount.Mul(sdkmath.NewInt(int64(config.MaxDisbursementBps))).Quo(sdkmath.NewInt(1_000_000))
    if amount.GT(maxAmount) {
        return fmt.Errorf("amount %s exceeds max disbursement %s (20%% of fund %s)",
            amount, maxAmount, fundBalance.Amount)
    }

    // Purpose is required (accountability)
    if msg.Purpose == "" {
        return fmt.Errorf("purpose is required for research fund disbursement")
    }

    // Transfer
    recipientAddr, err := sdk.AccAddressFromBech32(msg.Recipient)
    if err != nil {
        return fmt.Errorf("invalid recipient: %w", err)
    }
    coins := sdk.NewCoins(sdk.NewCoin("ulgm", amount))
    if err := k.bankKeeper.SendCoinsFromModuleToAccount(ctx, "research_fund", recipientAddr, coins); err != nil {
        return err
    }

    // Update config
    totalDisbursed := new(big.Int)
    totalDisbursed.SetString(config.TotalDisbursed, 10)
    totalDisbursed.Add(totalDisbursed, amount.BigInt())
    config.TotalDisbursed = totalDisbursed.String()
    config.LastDisbursementHeight = currentHeight
    k.SetResearchFundConfig(ctx, config)

    // Emit event with full transparency
    ctx.EventManager().EmitEvent(sdk.NewEvent(
        "research_fund_disbursed",
        sdk.NewAttribute("authority", msg.Authority),
        sdk.NewAttribute("amount", msg.Amount),
        sdk.NewAttribute("recipient", msg.Recipient),
        sdk.NewAttribute("purpose", msg.Purpose),
        sdk.NewAttribute("domain", msg.Domain),
        sdk.NewAttribute("fund_remaining", fundBalance.Amount.Sub(amount).String()),
        sdk.NewAttribute("total_disbursed", config.TotalDisbursed),
    ))

    return nil
}
```

### 5. Sunset Mechanism

Add a separate message to schedule the governance activation:

```go
type MsgScheduleGovernanceActivation struct {
    Authority       string `json:"authority"`
    ActivationHeight uint64 `json:"activation_height"` // Must be in the future
}
```

This is one-way: once set, it can only be moved *earlier*, never later.
The founder commits to a date, and can accelerate it but never delay it.

```go
func (k Keeper) ScheduleGovernanceActivation(ctx sdk.Context, msg *MsgScheduleGovernanceActivation) error {
    config := k.GetResearchFundConfig(ctx)
    currentHeight := uint64(ctx.BlockHeight())

    if msg.Authority != config.Authority {
        return fmt.Errorf("unauthorized")
    }
    if msg.ActivationHeight <= currentHeight {
        return fmt.Errorf("activation height must be in the future")
    }
    // One-way ratchet: can only move earlier, never later
    if config.GovernanceActivationHeight > 0 && msg.ActivationHeight > config.GovernanceActivationHeight {
        return fmt.Errorf("cannot delay governance activation: current=%d, requested=%d",
            config.GovernanceActivationHeight, msg.ActivationHeight)
    }

    config.GovernanceActivationHeight = msg.ActivationHeight
    k.SetResearchFundConfig(ctx, config)

    ctx.EventManager().EmitEvent(sdk.NewEvent(
        "governance_activation_scheduled",
        sdk.NewAttribute("activation_height", fmt.Sprintf("%d", msg.ActivationHeight)),
    ))
    return nil
}
```

### 6. Query Endpoint

```bash
legbled query knowledge research-fund
```

Returns:
```json
{
    "balance": "14500000",
    "authority": "lgm1founder...",
    "governance_activation_height": 0,
    "max_disbursement_bps": 200000,
    "total_disbursed": "3200000",
    "last_disbursement_height": 102400,
    "disbursement_history": [...]
}
```

Full transparency: anyone can see the balance, who controls it, how much
has been spent, and when governance takes over.

### 7. Disbursement History

Store each disbursement as a record for on-chain accountability:

```go
type DisbursementRecord struct {
    Height    uint64 `json:"height"`
    Amount    string `json:"amount"`
    Recipient string `json:"recipient"`
    Purpose   string `json:"purpose"`
    Domain    string `json:"domain"`
}
```

Query: `legbled query knowledge research-fund-history`

### 8. Tests

```go
func TestFounderDisburse(t *testing.T)                 // Happy path
func TestNonFounderDenied(t *testing.T)                // Wrong authority rejected
func TestMaxDisbursement(t *testing.T)                 // >20% rejected
func TestCooldown(t *testing.T)                        // Too frequent rejected
func TestPurposeRequired(t *testing.T)                 // Empty purpose rejected
func TestGovernanceActivation(t *testing.T)            // After activation, founder blocked
func TestGovernanceRatchet(t *testing.T)               // Can't delay activation
func TestDisbursementHistory(t *testing.T)             // History recorded and queryable
```

### 9. Run

```bash
go test ./x/knowledge/keeper/... -v -run TestResearchFund
go test ./x/knowledge/... -count=1
```

## Commit
```
feat(knowledge): founder-directed research fund with sunset mechanism

Research fund (7% tax) is founder-directed during bootstrap phase.
Guardrails: max 20% per disbursement, 1-day cooldown, required purpose
string, full on-chain history. One-way governance activation ratchet:
once a date is set, it can only be moved earlier. Phase 2 governance
(propose/vote/disburse) activates at the scheduled height.
```
