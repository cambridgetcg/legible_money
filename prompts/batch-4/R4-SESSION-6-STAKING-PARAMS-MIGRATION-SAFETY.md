# R4 Session 6: Staking Params Migration Safety + Autopoiesis Verify

## Context
Session R3-5 embedded tier configs in Params and parameterized slash/reputation constants. The report notes a critical backward compatibility concern:

> "GetParams: If unmarshaled Params has empty TierConfigs (pre-migration), fills from KVStore via GetAllTierConfigs"

This migration path needs stress testing. Also:

> "go build ./... fails in x/knowledge (missing ExtendedParams arguments) — pre-existing"

This means the full build was broken at the time R3-5 ran. Need to verify staking changes work in the context of a fully building codebase.

## Tasks

### 1. Migration Path Test
Create a test that simulates upgrade from old Params (no TierConfigs field) to new:
- Store old-format Params (without TierConfigs, without reputation deltas)
- Call GetParams — verify it fills TierConfigs from KVStore and defaults for reputation/cooldown
- Call SetParams with the filled params — verify it syncs back correctly
- Export genesis — verify TierConfigs appear in Params

### 2. SetParams → KVStore Sync Integrity
When SetParams writes TierConfigs to Params AND syncs to KVStore:
- Modify tier configs via governance (SetParams with different min_stake)
- Read back via GetParams → verify new values
- Read individual tier config from KVStore → verify synced
- Run CalculateTierSlash → verify it uses the new slash multiplier

### 3. Autopoiesis SSI Multiplier Path
The SSI multiplier (`autopoiesisKeeper.GetMultiplier(ctx, "system.slash")`) multiplies the parameterized slash amount. Test the full chain:
- Set a custom SlashMultiplierBps for Guardian tier (e.g., 2000)
- Set SSI multiplier to 0.8 (dampen)
- SlashValidator → verify: `base_slash * 2000/10000 * 0.8`
- This is an integration test — may need both keepers wired

### 4. Validation Edge Cases
- TierConfigs with exactly 4 tiers, all valid → passes
- TierConfigs with 3 tiers → fails
- TierConfigs with 5 tiers → fails
- TierConfig with min_stake = 0 → fails
- TierConfigs with wrong tier ordering (Guardian before Apprentice) → fails
- ReputationCorrectDelta = 0 → should it be allowed? Document the decision
- RedelegationCooldownBlocks = 0 → should it be allowed? (instant redelegation)

### 5. Genesis Round-Trip
- Create genesis with new params format
- InitGenesis → ExportGenesis → compare
- Verify tier configs survive the round-trip exactly

### 6. Deprecated Field Cleanup
`GenesisState.tier_configs` (field 5) is marked deprecated. Verify:
- New genesis doesn't populate it
- Old genesis with it still works (InitGenesis handles both)
- ExportGenesis doesn't include deprecated field

## Verification
```bash
go build ./...
go test ./x/staking/... -v -count=1
go test ./x/autopoiesis/... -count=1
go test ./tests/cross_stack/... -count=1
```

## Report
Write to `/reports/batch-4/SESSION-R4-6-STAKING-MIGRATION.md`
