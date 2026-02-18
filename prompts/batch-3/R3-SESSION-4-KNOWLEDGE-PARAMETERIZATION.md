# Session 4: x/knowledge Constant Parameterization (P0)

## Objective
x/knowledge has 60+ hardcoded constants in `types/types.go` that should be governance-adjustable via the Params struct. This is P0 from the Module Upgrade Plans.

## Reference
- `docs/MODULE_UPGRADE_PLANS.md` — P0-A section has the full constant list
- `x/knowledge/types/types.go` — current hardcoded constants
- `x/knowledge/types/params.go` — existing Params struct
- `x/knowledge/keeper/` — uses these constants directly

## Tasks

### 1. Expand the Params Proto
Edit `proto/legible/knowledge/v1/genesis.proto` (or wherever Params is defined) to add fields for:

**Group 1: Category Decay Profiles** (lines 55-68 of types.go)
- Add `repeated CategoryDecayProfile category_decay_profiles` to Params
- Define `CategoryDecayProfile` message with `category`, `half_life_blocks`, `max_confidence`

**Group 2: Evidence Type Strengths** (lines 189-197)
- Add `repeated EvidenceTypeStrength evidence_type_strengths` to Params
- Define `EvidenceTypeStrength` message with `type`, `strength`

**Group 3: Challenge Window Config** (lines 117-154)
- Add individual fields: `provisional_threshold`, `reject_threshold`, `base_challenge_duration`, `min_challenge_stake`, `failed_challenge_slash_bps`, `successful_challenge_reward_bps`, `confidence_growth_per_epoch`, `max_survival_confidence`, `base_cooldown_blocks`, `max_cooldown_blocks`, `max_reverifications_per_block`

**Group 4: Maturity Criteria** (lines 200-207)
- Add: `maturity_established_min_conf`, `maturity_established_min_age`, `maturity_established_min_verifications`

**Group 5: Remaining Constants**
- Scan types.go for any other hardcoded values used in consensus logic
- Add them to Params with sensible defaults matching current values

### 2. Regenerate Proto
```bash
make proto-gen
```

### 3. Update DefaultParams
In `x/knowledge/types/params.go`, update `DefaultParams()` to populate all new fields with the **exact current hardcoded values** (no behavioral change).

### 4. Update Keeper
- Replace all direct constant references in `x/knowledge/keeper/` with `k.GetParams(ctx).FieldName`
- Helper methods are fine (e.g., `k.getCategoryDecay(ctx, category)`)
- **Critical**: The default values must produce identical behavior to current hardcoded constants

### 5. Update Validation
- Add bounds validation in `Params.Validate()` for all new fields
- Prevent obviously dangerous values (e.g., zero challenge duration, negative thresholds)

### 6. Migration
- No state migration needed — existing chains will get defaults on upgrade
- But ensure `DefaultGenesis()` includes all new fields

## Verification
```bash
go build ./...
go test ./x/knowledge/...
# Cross-stack vectors should still pass:
go test ./tests/cross_stack/...
```

## Report
Write completion report to `/reports/SESSION-R3-4-KNOWLEDGE-PARAMS.md` with:
- Full list of constants moved to Params (name, old location, new Params field, default value)
- Count of keeper call sites updated
- Test results including cross-stack golden vectors
- Any constants intentionally left hardcoded and why
