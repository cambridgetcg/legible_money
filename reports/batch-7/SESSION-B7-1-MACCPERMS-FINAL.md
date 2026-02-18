# SESSION B7-1: MACCPERMS-FINAL — Comprehensive Module Account Audit

**Date**: 2026-02-18
**Scope**: Full audit of all bank keeper calls → maccPerms cross-reference
**Result**: 11 missing module accounts found and added. Compile-time guard added. All 37 test packages pass.

---

## Audit Methodology

Searched all non-test, non-proto `.go` files for:
- `SendCoinsFromAccountToModule`
- `SendCoinsFromModuleToAccount`
- `SendCoinsFromModuleToModule`
- `BurnCoins`
- `MintCoins`

Extracted every module name string passed to these functions. Cross-referenced against `maccPerms` in `app/app.go`.

---

## Complete Audit Table

| Module | ModuleName | Bank Operations | Required Perms | Was in maccPerms? |
|--------|-----------|-----------------|----------------|-------------------|
| x/auth | `legible_auth` | SendModuleToAccount | nil | YES |
| x/capture_challenge | `capture_challenge` | SendAccToModule, SendModuleToAcc | nil | YES (Burner) |
| x/capture_defense | `capture_defense` | SendAccToModule, SendModuleToAcc | nil | YES (Burner) |
| x/channels | `channels` | SendAccToModule, SendModuleToAcc, BurnCoins, ModuleToModule | Burner | YES |
| x/claiming_pot | `claiming_pot` | SendAccToModule, SendModuleToAcc | nil | YES (Burner) |
| x/compute_pool | `compute_pool` | SendAccToModule, SendModuleToAcc | nil | YES (Burner) |
| x/disputes | `disputes` | SendAccToModule, SendModuleToAcc, BurnCoins, ModuleToModule | Burner | YES |
| x/evidence_mgmt | `evidence_mgmt` | SendAccToModule, SendModuleToAcc, BurnCoins | Burner | YES |
| x/gov | `lgm_gov` | SendAccToModule | nil | YES (B6-1 fix, Burner) |
| x/knowledge | `knowledge` | SendAccToModule, SendModuleToAcc, BurnCoins, ModuleToModule | Burner | YES |
| x/qualification | `qualification` | SendAccToModule, SendModuleToAcc | nil | YES (Burner) |
| x/staking | `legible_staking` | SendAccToModule, SendModuleToAcc, BurnCoins, MintCoins | Burner+Staking | YES |
| x/tokens | `tokens` | - | Minter+Burner | YES |
| x/tree | `tree` | SendAccToModule, SendModuleToAcc | nil | YES (Burner) |
| x/billing | `billing` | SendAccToModule, SendModuleToAcc, BurnCoins | Burner | YES |
| x/partnerships | `partnerships` | - | Burner+Staking | YES |
| x/vesting_rewards | `vesting_rewards` | MintCoins, SendModuleToAcc, ModuleToModule | Minter+Burner | YES |
| **x/bvm** | **`bvm`** | SendAccToModule, SendModuleToAcc | **nil** | **NO** |
| **x/discovery** | **`discovery`** | SendAccToModule, SendModuleToAcc | **nil** | **NO** |
| **x/ontology** | **`ontology`** | SendAccToModule, SendModuleToAcc | **nil** | **NO** |
| **x/research** | **`research`** | SendAccToModule, SendModuleToAcc, BurnCoins | **Burner** | **NO** |
| **x/schedule** | **`schedule`** | SendAccToModule, SendModuleToAcc | **nil** | **NO** |
| **billing treasury** | **`treasury_protocol`** | SendAccToModule (destination) | **nil** | **NO** |
| **billing treasury** | **`treasury_research`** | SendAccToModule (destination) | **nil** | **NO** |
| **billing treasury** | **`treasury_foundation`** | SendAccToModule (destination) | **nil** | **NO** |
| **billing treasury** | **`treasury_community`** | SendAccToModule (destination) | **nil** | **NO** |
| **billing treasury** | **`treasury_developers`** | SendAccToModule (destination) | **nil** | **NO** |
| **billing treasury** | **`treasury_reserve`** | SendAccToModule (destination) | **nil** | **NO** |
| fee routing | `research_fund` | ModuleToModule (destination) | nil | YES |
| fee routing | `burn` | BurnCoins | Burner | YES |

**Modules without bank keeper calls** (no module account needed):
- x/home — only uses `bankKeeper.GetBalance()` (read-only)
- x/autopoiesis — no bank keeper calls at all
- x/alignment — no bank keeper calls
- x/emergency — no bank keeper calls

---

## Entries Added

### app/gas.go — New billing treasury constants

```go
const (
    TreasuryProtocolName    = "treasury_protocol"
    TreasuryResearchName    = "treasury_research"
    TreasuryFoundationName  = "treasury_foundation"
    TreasuryCommunityName   = "treasury_community"
    TreasuryDevelopersName  = "treasury_developers"
    TreasuryReserveName     = "treasury_reserve"
)
```

These match the constants in `x/billing/keeper/distribution.go:14-21`.

### app/app.go — maccPerms additions (11 entries)

```go
// Modules with bank keeper calls that were missing (B7-1 audit)
bvmtypes.ModuleName:              nil,                // BVM value transfer escrow
discoverytypes.ModuleName:        nil,                // agent registration stake escrow
ontologytypes.ModuleName:         nil,                // domain proposal stake return
researchtypes.ModuleName:         {authtypes.Burner}, // research submission/bounty escrow + burn on rejection
scheduletypes.ModuleName:         nil,                // schedule creation fee escrow
// Billing treasury accounts (5-way research split + protocol)
TreasuryProtocolName:             nil,
TreasuryResearchName:             nil,
TreasuryFoundationName:           nil,
TreasuryCommunityName:            nil,
TreasuryDevelopersName:           nil,
TreasuryReserveName:              nil,
```

### app/app.go — Compile-time guard (init function)

Added a startup validation in `init()` that checks all 30 custom module accounts are present in `maccPerms`. If any are missing, the binary panics at startup with:
```
CRITICAL: module "xxx" uses bank keeper but is not registered in maccPerms — add it to app/app.go
```

This ensures future module additions that use bank keeper calls will fail fast at startup (not at runtime when a user triggers the code path).

---

## Permission Rationale

| Permission | Meaning | Modules |
|-----------|---------|---------|
| nil | Can only send/receive between module accounts and regular accounts | bvm, discovery, ontology, schedule, all 6 treasuries |
| Burner | Can destroy tokens (slashing, rejection penalties) | research, disputes, evidence_mgmt, channels, billing, etc. |
| Minter | Can create new tokens | vesting_rewards, tokens |
| Staking | Interacts with staking pool | legible_staking, partnerships |

**Note**: `research` needs Burner because `msg_server.go:282` calls `BurnCoins(ctx, types.ModuleName, burnCoins)` when a research submission is rejected.

---

## Files Changed

| File | Change |
|------|--------|
| `app/app.go` | 11 new maccPerms entries + init() guard (30 module names) |
| `app/gas.go` | 6 billing treasury constant definitions |

---

## Verification

```
$ go build ./...                    # Clean
$ go test ./... -count=1 -timeout 600s
  37 packages OK, 0 FAIL
```

**Total maccPerms entries**: 38 (was 27 + 11 added)
- 8 standard SDK/IBC
- 30 custom Legible Money

---

## Discovery: Billing Treasury Architecture

The billing module implements a 5-way research split with 6 dedicated treasury module accounts, all defined as local constants in `x/billing/keeper/distribution.go:14-21`. None were registered in maccPerms, meaning ALL billing distribution operations would panic on a live chain. This was the most impactful finding — 6 of the 11 missing accounts.

Note: `ResearchFundName` ("research_fund") and `TreasuryResearchName` ("treasury_research") are different module accounts:
- `research_fund` — receives the 7% research tax from vesting_rewards block emissions
- `treasury_research` — receives the research portion of the billing 5-way split

Both are correctly registered.
