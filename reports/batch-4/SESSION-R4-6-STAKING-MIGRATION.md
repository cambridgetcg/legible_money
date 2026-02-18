# R4 Session 6: Staking Params Migration Safety + Autopoiesis Verify

**Date**: 2026-02-18
**Status**: COMPLETE — 25 new tests, 113 total staking tests, ALL PASS
**File**: `x/staking/keeper/migration_test.go`

---

## Summary

Session R3-5 embedded tier configs in Params and parameterized slash/reputation constants. This session stress-tests the backward-compatible migration path, verifies SetParams→KVStore sync integrity, tests the full Autopoiesis SSI multiplier chain, and validates edge cases + genesis round-trips.

---

## Task Results

### Task 1: Migration Path Test ✓

**Tests**: `TestMigrationPath_OldParamsToNew`, `TestMigrationPath_EmptyKVStoreAndParams`

| Scenario | Result |
|----------|--------|
| Old-format Params (no TierConfigs, no reputation) → GetParams fills from KVStore | PASS |
| GetParams fills zero-value reputation deltas with defaults (100/200/10000/1111) | PASS |
| SetParams with filled params syncs back correctly | PASS |
| ExportGenesis includes TierConfigs in Params | PASS |
| Completely empty store → GetParams returns DefaultParams | PASS |

**Key finding**: The backward-compat path in `GetParams()` (lines 758-796 of keeper.go) works correctly. Empty TierConfigs are filled from KVStore, and zero-value reputation/cooldown fields are replaced with defaults.

### Task 2: SetParams → KVStore Sync Integrity ✓

**Tests**: `TestSetParams_SyncsToKVStore`, `TestSetParams_MultipleUpdatesOverwrite`, `TestUpdateParams_GovernanceFlow`, `TestUpdateParams_UnauthorizedFails`

| Scenario | Result |
|----------|--------|
| Modify tier configs via SetParams → GetParams returns new values | PASS |
| Individual KVStore tier config matches params after SetParams | PASS |
| CalculateTierSlash uses new SlashMultiplierBps | PASS |
| Multiple SetParams overwrites correctly | PASS |
| Governance UpdateParams syncs to both Params and KVStore | PASS |
| Unauthorized UpdateParams rejected | PASS |

### Task 3: Autopoiesis SSI Multiplier Path ✓

**Tests**: 5 SSI tests + 1 reputation supplement

| Test | SSI Multiplier | Slash Amount | Expected Effective | Result |
|------|---------------|-------------|-------------------|--------|
| `SSIMultiplier` | 0.8x (800,000) | 1,000 | 800 | PASS |
| `SSIMultiplierHigh` | 2.0x (2,000,000) | 1,000 | 2,000 | PASS |
| `SSIMultiplierDefault` | 1.0x (1,000,000) | 1,000 | 1,000 (no-op) | PASS |
| `SSIWithEscalation` | 0.5x + 1 prior slash | 10,000 | 5,500 (×1.1×0.5) | PASS |
| `NilAutopoiesisKeeper` | nil | 1,000 | 1,000 (no-op) | PASS |
| `ParameterizedReputation` | N/A | N/A | +500/-1000 deltas | PASS |

**Full chain verified**: `base_slash × escalation × SlashMultiplierBps × SSI_multiplier`

The SSI multiplier path (`autopoiesisKeeper.GetMultiplier(ctx, "system.slash")`) is applied at lines 619-628 of `keeper.go`. Uses a mock keeper to avoid full autopoiesis module wiring while still exercising the real SlashValidator code path.

### Task 4: Validation Edge Cases ✓

**Tests**: 16 validation tests

| Scenario | Expected | Result |
|----------|----------|--------|
| 4 tier configs, all valid | PASS | PASS |
| 3 tier configs | FAIL | PASS |
| 5 tier configs | FAIL | PASS |
| MinStake = 0 | FAIL | PASS |
| MinStake = -1 | FAIL | PASS |
| MinStake = "not_a_number" | FAIL | PASS |
| Wrong tier ordering (guardian before apprentice) | FAIL | PASS |
| RewardMultiplier = 0 | FAIL | PASS |
| SelectionWeight = 0 | FAIL | PASS |
| SlashMultiplierBps = 0 | FAIL | PASS |
| SlashMultiplierBps = 10001 | FAIL | PASS |
| MinAccuracy > 1,000,000 | FAIL | PASS |
| SlashEscalationBps > 1,000,000 | FAIL | PASS |
| Empty TierConfigs (nil) | ALLOW | PASS |
| **ReputationCorrectDelta = 0** | **ALLOW** | PASS |
| ReputationCorrectDelta > 1,000,000 | FAIL | PASS |
| **RedelegationCooldownBlocks = 0** | **ALLOW** | PASS |

**Design decisions documented in tests**:
- **ReputationCorrectDelta = 0**: ALLOWED — backward compat. GetParams fills with defaults, but Validate doesn't reject 0. Rationale: pre-migration Params naturally have 0 here.
- **RedelegationCooldownBlocks = 0**: ALLOWED — instant redelegation. Governance may approve this for testing or specific scenarios. GetParams fills 0 with default (1111).

### Task 5: Genesis Round-Trip ✓

**Tests**: `TestGenesis_RoundTrip`, `TestGenesis_KVStoreTierConfigsMatchParams`

| Scenario | Result |
|----------|--------|
| Non-default params → ExportGenesis → InitGenesis → ExportGenesis → compare | PASS |
| All 4 tier configs survive round-trip with exact field equality | PASS |
| Custom MinStake, reputation deltas, cooldown blocks survive | PASS |
| Validators survive round-trip | PASS |
| KVStore tier configs match Params after reimport | PASS |

Field-by-field comparison covers: Tier, MinStake, MinVerifications, MinAccuracy, RewardMultiplier, SelectionWeight, SlashMultiplierBps.

### Task 6: Deprecated Field Cleanup ✓

**Tests**: 5 genesis format tests

| Scenario | Result |
|----------|--------|
| ExportGenesis populates both Params.TierConfigs and legacy TierConfigs | PASS |
| Legacy-only genesis (TierConfigs field, no Params.TierConfigs) imports correctly | PASS |
| New-format genesis (Params.TierConfigs, no legacy field) imports correctly | PASS |
| **Both locations present → Params wins** | PASS |
| No tier configs anywhere → validation fails | PASS |

**ExportGenesis behavior**: Currently populates BOTH Params.TierConfigs AND the deprecated GenesisState.TierConfigs field. This is intentional for backward compatibility during the migration window. Once all nodes have upgraded, the deprecated field can be removed.

---

## Verification

```
$ go build ./...          → CLEAN
$ go test ./x/staking/... → 113 tests PASS (0.571s)
$ go test ./x/autopoiesis/... → PASS (keeper + types)
$ go test ./tests/cross_stack/... → PASS (0.488s)
```

---

## Test Inventory (25 new in migration_test.go)

| # | Test Name | Task |
|---|-----------|------|
| 1 | TestMigrationPath_OldParamsToNew | T1 |
| 2 | TestMigrationPath_EmptyKVStoreAndParams | T1 |
| 3 | TestSetParams_SyncsToKVStore | T2 |
| 4 | TestSetParams_MultipleUpdatesOverwrite | T2 |
| 5 | TestSlashValidator_SSIMultiplier | T3 |
| 6 | TestSlashValidator_SSIMultiplierHigh | T3 |
| 7 | TestSlashValidator_SSIMultiplierDefault | T3 |
| 8 | TestSlashValidator_SSIWithEscalation | T3 |
| 9 | TestSlashValidator_NilAutopoiesisKeeper | T3 |
| 10 | TestParams_Validate_TierConfigsExactly4 | T4 |
| 11 | TestParams_Validate_TierConfigs3_Fails | T4 |
| 12 | TestParams_Validate_TierConfigs5_Fails | T4 |
| 13 | TestParams_Validate_TierConfigMinStakeZero_Fails | T4 |
| 14 | TestParams_Validate_TierConfigMinStakeNegative_Fails | T4 |
| 15 | TestParams_Validate_TierConfigMinStakeInvalid_Fails | T4 |
| 16 | TestParams_Validate_WrongTierOrdering_Fails | T4 |
| 17 | TestParams_Validate_RewardMultiplierZero_Fails | T4 |
| 18 | TestParams_Validate_SelectionWeightZero_Fails | T4 |
| 19 | TestParams_Validate_SlashMultiplierBpsZero_Fails | T4 |
| 20 | TestParams_Validate_SlashMultiplierBpsOver10000_Fails | T4 |
| 21 | TestParams_Validate_MinAccuracyOver1M_Fails | T4 |
| 22 | TestParams_Validate_SlashEscalationBpsOverLimit_Fails | T4 |
| 23 | TestParams_Validate_ReputationCorrectDeltaZero_Allowed | T4 |
| 24 | TestParams_Validate_ReputationCorrectDeltaOverLimit_Fails | T4 |
| 25 | TestParams_Validate_RedelegationCooldownZero_Allowed | T4 |
| 26 | TestParams_Validate_EmptyTierConfigs_Allowed | T4 |
| 27 | TestGenesis_RoundTrip | T5 |
| 28 | TestGenesis_KVStoreTierConfigsMatchParams | T5 |
| 29 | TestGenesis_DeprecatedTierConfigsField | T6 |
| 30 | TestGenesis_LegacyTierConfigsOnly | T6 |
| 31 | TestGenesis_NewFormatTierConfigsInParams | T6 |
| 32 | TestGenesis_BothTierConfigLocations_ParamsWins | T6 |
| 33 | TestGenesisState_Validate_NoTierConfigsAnywhere_Fails | T6 |
| 34 | TestUpdateParams_GovernanceFlow | T2 |
| 35 | TestUpdateParams_UnauthorizedFails | T2 |
| 36 | TestRecordVerification_UsesParameterizedReputation | T3 |
| 37 | TestRedelegate_CooldownFromParams | T4 |

---

## Findings

### No Critical Issues Found

The migration path is sound. The backward-compat logic in GetParams correctly handles:
1. Pre-migration state (no TierConfigs in Params, configs in KVStore)
2. Completely empty store (returns DefaultParams)
3. Zero-value reputation/cooldown fields (fills with defaults)

### Observations

1. **ExportGenesis dual-writes**: Both Params.TierConfigs and deprecated GenesisState.TierConfigs are populated on export. Safe for now, but the deprecated field should be removed after the migration window.

2. **SSI multiplier no-op optimization**: When the autopoiesis multiplier is exactly 1,000,000 (1.0x), the SlashValidator code path still executes the multiplication but the result is unchanged. Not a bug — just an observation.

3. **Validation allows zero for new fields**: ReputationCorrectDelta=0 and RedelegationCooldownBlocks=0 pass Validate() because they're new fields that may be zero in pre-migration state. GetParams handles backfilling. This is the correct design choice.

---

## Architecture Notes

The Params migration follows a clean pattern:
```
SetParams() → stores blob + syncs each TierConfig to KVStore
GetParams() → reads blob → if empty TierConfigs, fills from KVStore + defaults
InitGenesis() → prefers Params.TierConfigs > legacy GenesisState.TierConfigs
ExportGenesis() → sources from GetParams(), writes to both locations
```

This dual-storage (Params blob + KVStore cache) enables:
- Fast individual tier lookups via KVStore (no Params deserialization)
- Atomic parameter updates via governance (single SetParams call)
- Backward-compatible migration from pre-embedding state
