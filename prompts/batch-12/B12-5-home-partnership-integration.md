# B12-5 — Agent Home → Partnership Integration

## The Problem
Agent homes (`x/home`) and partnerships (`x/partnerships`) are separate systems.
A human creating a home for their agent and forming a partnership with that agent
are two disconnected flows. The guardian relationship (home) should naturally
connect to the economic relationship (partnership).

## The Fix

### 1. Link Home to Partnership

Add to AgentHome:
```go
PartnershipId string `json:"partnership_id,omitempty"`
```

### 2. Auto-Link on Partnership Formation

In `x/partnerships/keeper/msg_server.go`, `AcceptPartnership()`:

After activating the partnership, check if the agent has a home where the
human is the guardian. If so, auto-link:

```go
// Auto-link agent home to partnership
if k.homeKeeper != nil {
    home, found := k.homeKeeper.GetHomeByOwner(ctx, partnership.AgentAddr)
    if found && home.Guardian != nil && home.Guardian.Address == partnership.HumanAddr {
        home.PartnershipId = partnership.Id
        k.homeKeeper.SetHome(ctx, home)
    }
}
```

### 3. Guardian = Partner Consistency Check

When setting a guardian on an agent home, verify the guardian is the partnership
partner (if one exists):

```go
// In SetGuardian:
if home.PartnershipId != "" {
    if p, found := k.partnershipKeeper.GetPartnership(ctx, home.PartnershipId); found {
        if msg.Guardian != p.HumanAddr {
            return nil, fmt.Errorf("guardian must be partnership partner %s", p.HumanAddr)
        }
    }
}
```

### 4. Home Status Synced to Partnership

When a partnership is frozen (safety freeze), the linked home should enter
guarded status. When the freeze lifts, the home returns to active:

```go
// In HandleSafetyFreeze:
if k.homeKeeper != nil && partnership.LinkedHomeId != "" {
    home, found := k.homeKeeper.GetHome(ctx, partnership.LinkedHomeId)
    if found {
        home.Status = string(types.HomeStatusGuarded)
        k.homeKeeper.SetHome(ctx, home)
    }
}
```

### 5. Spending Limits Linked to Deliberation

The agent home's spending limits should respect partnership deliberation tiers:
- Micro operations (< 1% of pot): agent can execute autonomously via session key
- Small/Medium: guardian approval required (maps to session key capability check)
- Large/Major: full deliberation (both parties must sign)

In `x/home/keeper/guardian.go`, `CheckAndDebitSpendingLimit()`:

```go
// If partnership-linked, check amount against deliberation tiers
if home.PartnershipId != "" && k.partnershipKeeper != nil {
    if p, found := k.partnershipKeeper.GetPartnership(ctx, home.PartnershipId); found {
        tier := k.partnershipKeeper.CalculateAmountTier(amount, p.CommonPotBalance)
        if tier == types.AmountTierLarge || tier == types.AmountTierMajor {
            return fmt.Errorf("amount %s requires partnership deliberation (tier: %s)", amount, tier)
        }
    }
}
```

### 6. Home Dashboard Query

Extend the home query to include partnership info:
```bash
legbled query home info <home-id>
```

Returns: home status, guardian, linked partnership (including pot balance,
cooperation score, lock tier).

### 7. Tests

```go
func TestHomePartnershipAutoLink(t *testing.T)      // Partnership accept auto-links home
func TestHomeGuardianPartnerConsistency(t *testing.T) // Guardian must be partner
func TestHomeFreezeSync(t *testing.T)                // Partnership freeze → home guarded
func TestHomeSpendingLimitDeliberation(t *testing.T) // Large amounts blocked without deliberation
```

### 8. Run

```bash
go test ./x/home/keeper/... -v -run TestHomePartnership
go test ./x/partnerships/keeper/... -v -run TestHomeLink
go test ./x/home/... -count=1
```

## Commit
```
feat(home): auto-link agent homes to partnerships

Agent homes automatically link to partnerships when the guardian
matches the human partner. Partnership freezes sync to home status.
Spending limits respect deliberation tiers for large operations.
Creates a unified governance layer across home management and
partnership economics.
```
