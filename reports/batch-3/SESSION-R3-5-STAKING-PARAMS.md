# SESSION-R3-5: Staking Tier Config Parameterization (P1-A)

**Date**: 2026-02-18
**Scope**: x/staking — embed tier configs in Params, parameterize hardcoded constants
**Result**: 76/76 tests PASS (8 new), zero behavioral change with default values

---

## Summary

Tier configs (4 tiers with ~12 fields each) were stored in KVStore but **not part of Params** — governance via `MsgUpdateParams` could not update them. Several constants were hardcoded in keeper code (slash multipliers, reputation deltas, redelegation cooldown) with no governance path. This session embeds tier configs in Params and parameterizes all remaining hardcoded constants.

---

## Changes

### Proto Schema

| File | Change |
|------|--------|
| `proto/legible/staking/v1/state.proto` | Added `slash_multiplier_bps` (field 12) to TierConfig |
| `proto/legible/staking/v1/genesis.proto` | Added fields 11-15 to Params: `tier_configs`, `reputation_correct_delta`, `reputation_incorrect_delta`, `reputation_slash_delta`, `redelegation_cooldown_blocks` |
| `proto/legible/staking/v1/genesis.proto` | Marked `GenesisState.tier_configs` (field 5) as deprecated |

### Generated Code

| File | Change |
|------|--------|
| `x/staking/types/state.pb.go` | TierConfig.SlashMultiplierBps field + getter |
| `x/staking/types/genesis.pb.go` | Params.TierConfigs, reputation deltas, cooldown fields + getters |

### Types

| File | Change |
|------|--------|
| `x/staking/types/types.go` | DefaultTierConfigs: added SlashMultiplierBps (1500/1200/1000/1000) |
| `x/staking/types/types.go` | Params.Validate: tier config validation (count, order, ranges), reputation/cooldown bounds |
| `x/staking/types/genesis.go` | DefaultParams: tier configs + reputation deltas + cooldown embedded |
| `x/staking/types/genesis.go` | DefaultGenesisState: TierConfigs sourced from params |
| `x/staking/types/genesis.go` | GenesisState.Validate: accepts tier configs from either params or legacy field |
| `x/staking/types/keys.go` | RedelegationCooldownBlocks constant marked deprecated |

### Keeper

| File | Change |
|------|--------|
| `x/staking/keeper/keeper.go` | SetParams: syncs TierConfigs to KVStore after writing params |
| `x/staking/keeper/keeper.go` | GetParams: backward-compat fill for empty TierConfigs and zero-value fields |
| `x/staking/keeper/keeper.go` | InitGenesis: SetParams handles tier config sync; legacy fallback for pre-migration genesis |
| `x/staking/keeper/keeper.go` | ExportGenesis: TierConfigs sourced from params (no separate GetAllTierConfigs) |
| `x/staking/keeper/keeper.go` | RecordVerification: reputation deltas from params instead of hardcoded 100/200 |
| `x/staking/keeper/keeper.go` | SlashValidator: reputation slash delta from params instead of hardcoded 10000 |
| `x/staking/keeper/tiers.go` | CalculateTierSlash: reads SlashMultiplierBps from TierConfig instead of switch |
| `x/staking/keeper/msg_server.go` | Redelegate: cooldown from params instead of types.RedelegationCooldownBlocks constant |
| `x/staking/keeper/grpc_query.go` | Params query: simplified, tier configs from params directly |

---

## Constant Inventory

| Constant | Old Location | New Location | Default |
|----------|-------------|-------------|---------|
| Slash multiplier (per-tier) | `tiers.go` switch statement | `TierConfig.SlashMultiplierBps` | 1500/1200/1000/1000 |
| Reputation correct delta | `keeper.go:547` hardcoded `100` | `Params.ReputationCorrectDelta` | 100 |
| Reputation incorrect delta | `keeper.go:551-554` hardcoded `200` | `Params.ReputationIncorrectDelta` | 200 |
| Reputation slash delta | `keeper.go:674` hardcoded `10000` | `Params.ReputationSlashDelta` | 10000 |
| Redelegation cooldown | `keys.go` const `1111` | `Params.RedelegationCooldownBlocks` | 1111 |
| All tier config fields (11) | `GenesisState.TierConfigs` only | `Params.TierConfigs` (canonical) | DefaultTierConfigs() |

---

## Autopoiesis Compatibility

The SSI multiplier path (`k.autopoiesisKeeper.GetMultiplier(ctx, "system.slash")` at `keeper.go:614`) is a **runtime adaptive value**, not a governance param. It multiplies the already-parameterized slash amount. No change needed — the formula `effective_slash = base * escalation * ssiMult` still holds with parameterized base values.

---

## Backward Compatibility

- **GetParams**: If unmarshaled Params has empty TierConfigs (pre-migration), fills from KVStore via GetAllTierConfigs. Zero-value reputation/cooldown fields filled with defaults.
- **InitGenesis**: If Params.TierConfigs is populated, SetParams handles everything (syncs to KVStore). Else falls back to GenesisState.TierConfigs (legacy path).
- **GenesisState.Validate**: Accepts tier configs from either Params or legacy field.
- **keys.go**: RedelegationCooldownBlocks const retained as fallback default, marked deprecated.

---

## Test Results

```
go test ./x/staking/... -count=1     # 76/76 PASS (68 existing + 8 new)
go test ./tests/cross_stack/... -count=1  # PASS (golden vectors)
go build ./x/staking/...             # CLEAN
```

### New Tests (8)

| # | Test | Verifies |
|---|------|----------|
| 1 | TestParamsIncludesTierConfigs | DefaultParams has 4 tier configs with correct SlashMultiplierBps defaults |
| 2 | TestSetParamsSyncsTierConfigs | SetParams with modified configs syncs to KVStore |
| 3 | TestSlashMultiplierFromTierConfig | CalculateTierSlash reads from TierConfig, responds to governance changes |
| 4 | TestReputationDeltasFromParams | RecordVerification uses parameterized deltas |
| 5 | TestRedelegationCooldownFromParams | Redelegate uses params cooldown, governance-adjustable |
| 6 | TestUpdateParamsWithTierConfigs | MsgUpdateParams flows through to KVStore |
| 7 | TestParamsValidation_BadTierConfigs | Validation rejects bad configs (count, order, ranges) |
| 8 | TestInitGenesisBackwardCompat | Legacy genesis (params without tier configs) works correctly |

### Pre-existing Issues (not introduced by this session)

- `go vet` warnings: proto-generated types contain `sync.Mutex` via `protoimpl.MessageState` — all pre-existing
- `go build ./...` fails in x/knowledge (missing ExtendedParams arguments) — pre-existing, unrelated to staking
- `go test ./app/...` fails due to x/knowledge build errors — pre-existing

---

## Files Modified (14)

1. `proto/legible/staking/v1/state.proto`
2. `proto/legible/staking/v1/genesis.proto`
3. `x/staking/types/state.pb.go` (generated)
4. `x/staking/types/genesis.pb.go` (generated)
5. `x/staking/types/types.go`
6. `x/staking/types/genesis.go`
7. `x/staking/types/keys.go`
8. `x/staking/keeper/keeper.go`
9. `x/staking/keeper/tiers.go`
10. `x/staking/keeper/msg_server.go`
11. `x/staking/keeper/grpc_query.go`
12. `x/staking/keeper/keeper_test.go`
