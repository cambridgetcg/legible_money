# Session 5: x/staking Tier Config Parameterization (P1)

## Objective
Refactor x/staking's `DefaultTierConfigs` and related hardcoded constants into governance-adjustable Params. This is P1 from the Module Upgrade Plans.

## Reference
- `docs/MODULE_UPGRADE_PLANS.md` — P1 section for x/staking
- `x/staking/types/types.go` — tier configs, constants
- `x/staking/types/params.go` — existing Params
- `x/staking/keeper/` — uses tier configs

## Tasks

### 1. Audit Current Constants
- Read `x/staking/types/types.go` and catalog all hardcoded values:
  - 4-tier validator config (Apprentice, Verified, Bonded, Guardian)
  - Per-tier: min stake, max validators, reward multiplier, VRF weight
  - Slash percentages, jail durations
  - Promotion/demotion thresholds
  - Any other constants used in consensus

### 2. Expand Params Proto
Edit the staking proto to add:
- `repeated TierConfig tier_configs` with per-tier fields
- Slash/jail parameters
- Promotion/demotion thresholds
- Define `TierConfig` message type

### 3. Regenerate Proto
```bash
make proto-gen
```

### 4. Update DefaultParams
- `DefaultParams()` must return the **exact current values** from `DefaultTierConfigs`
- Zero behavioral change

### 5. Update Keeper
- Replace `DefaultTierConfigs` lookups with params-based lookups
- Replace hardcoded slash/jail values with params
- Helper: `k.getTierConfig(ctx, tier) TierConfig`

### 6. Validation
- Tier configs must have exactly 4 tiers (or validate tier ordering)
- Min stake must be > 0
- Reward multipliers must be positive
- VRF weights must be positive
- Slash percentages must be 0-100%

### 7. Ensure Autopoiesis Compatibility
- x/autopoiesis applies SSI multipliers to staking slash values
- Verify the multiplier path still works with parameterized values
- The formula `effective_slash = params.slash * autopoiesis_multiplier` must hold

## Verification
```bash
go build ./...
go test ./x/staking/...
go test ./x/autopoiesis/...  # SSI consumer integration
go test ./tests/cross_stack/...  # validator tier golden vectors
```

## Report
Write completion report to `/reports/SESSION-R3-5-STAKING-PARAMS.md` with:
- Full constant inventory (name, old value, new Params field)
- TierConfig proto schema
- Keeper call sites updated
- Autopoiesis integration verified
- Test results including cross-stack golden vectors
