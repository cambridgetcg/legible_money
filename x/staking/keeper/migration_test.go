package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/legible-money/legible/x/staking/keeper"
	"github.com/legible-money/legible/x/staking/types"
)

// ---------- Mock AutopoiesisKeeper ----------

type mockAutopoiesisKeeper struct {
	multipliers map[string]uint64
}

func newMockAutopoiesisKeeper() *mockAutopoiesisKeeper {
	return &mockAutopoiesisKeeper{
		multipliers: make(map[string]uint64),
	}
}

func (m *mockAutopoiesisKeeper) GetMultiplier(_ sdk.Context, path string) uint64 {
	if v, ok := m.multipliers[path]; ok {
		return v
	}
	return 1_000_000 // default 1.0x
}

// ---------- Helper: fresh keeper without genesis ----------

func setupBareKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	storeService := runtime.NewKVStoreService(storeKey)
	bk := newMockBankKeeper()
	ak := &mockAccountKeeper{}

	k := keeper.NewKeeper(cdc, storeService, ak, bk, nil, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx
}

// ============================================================
// TASK 1: Migration Path Test
// Simulates upgrade from old-format Params (no TierConfigs,
// no reputation deltas) to new format via GetParams backward compat.
// ============================================================

func TestMigrationPath_OldParamsToNew(t *testing.T) {
	k, ctx := setupBareKeeper(t)

	// Step 1: Store old-format params WITHOUT TierConfigs and reputation deltas.
	// This simulates a pre-migration state where only the core fields existed.
	oldParams := types.Params{
		MaxApprenticeValidators: 0,
		UnbondingPeriod:         268560,
		VirtualStake:            "11000000",
		MaxValidators:           100,
		MinSelfDelegation:       "111000",
		MaxSlashesPerEpoch:      2,
		SlashDecayPeriodBlocks:  34272,
		MaxSlashCountDeactivate: 3,
		MinStakeForVerification: "111000",
		SlashEscalationBps:      100000,
		// Intentionally EMPTY:
		TierConfigs:                nil,
		ReputationCorrectDelta:     0,
		ReputationIncorrectDelta:   0,
		ReputationSlashDelta:       0,
		RedelegationCooldownBlocks: 0,
	}

	// Step 1b: Write tier configs to KVStore only (simulating pre-migration state where
	// tier configs were in KVStore but NOT embedded in Params).
	defaults := types.DefaultTierConfigs()
	for _, tc := range defaults {
		if err := k.SetTierConfig(ctx, *tc); err != nil {
			t.Fatalf("failed to set tier config: %v", err)
		}
	}

	// Write the old params via SetParams (it will try to sync empty tier configs — that's fine)
	if err := k.SetParams(ctx, oldParams); err != nil {
		t.Fatalf("failed to set old params: %v", err)
	}

	// Step 2: Call GetParams — verify it fills TierConfigs from KVStore
	params := k.GetParams(ctx)

	if len(params.TierConfigs) != 4 {
		t.Fatalf("expected GetParams to fill 4 TierConfigs from KVStore, got %d", len(params.TierConfigs))
	}

	// Verify tier ordering
	expectedTiers := []string{"apprentice", "verified", "bonded", "guardian"}
	for i, tc := range params.TierConfigs {
		if tc.Tier != expectedTiers[i] {
			t.Errorf("TierConfigs[%d]: expected tier %q, got %q", i, expectedTiers[i], tc.Tier)
		}
	}

	// Verify reputation deltas were filled with defaults
	if params.ReputationCorrectDelta != 100 {
		t.Errorf("expected ReputationCorrectDelta=100 (default), got %d", params.ReputationCorrectDelta)
	}
	if params.ReputationIncorrectDelta != 200 {
		t.Errorf("expected ReputationIncorrectDelta=200 (default), got %d", params.ReputationIncorrectDelta)
	}
	if params.ReputationSlashDelta != 10000 {
		t.Errorf("expected ReputationSlashDelta=10000 (default), got %d", params.ReputationSlashDelta)
	}
	if params.RedelegationCooldownBlocks != 1111 {
		t.Errorf("expected RedelegationCooldownBlocks=1111 (default), got %d", params.RedelegationCooldownBlocks)
	}

	// Step 3: Call SetParams with the filled params — verify it syncs back correctly
	if err := k.SetParams(ctx, params); err != nil {
		t.Fatalf("SetParams with filled params failed: %v", err)
	}

	// Read back and verify consistency
	params2 := k.GetParams(ctx)
	if len(params2.TierConfigs) != 4 {
		t.Fatalf("expected 4 TierConfigs after round-trip, got %d", len(params2.TierConfigs))
	}
	for i, tc := range params2.TierConfigs {
		if tc.Tier != expectedTiers[i] {
			t.Errorf("round-trip TierConfigs[%d]: expected %q, got %q", i, expectedTiers[i], tc.Tier)
		}
	}

	// Step 4: Export genesis — verify TierConfigs appear in Params
	gs := k.ExportGenesis(ctx)
	if gs.Params == nil {
		t.Fatal("exported genesis has nil params")
	}
	if len(gs.Params.TierConfigs) != 4 {
		t.Fatalf("exported genesis params should have 4 TierConfigs, got %d", len(gs.Params.TierConfigs))
	}
}

func TestMigrationPath_EmptyKVStoreAndParams(t *testing.T) {
	k, ctx := setupBareKeeper(t)

	// Completely empty store — GetParams should return defaults
	params := k.GetParams(ctx)

	// DefaultParams fills TierConfigs
	if len(params.TierConfigs) != 4 {
		t.Fatalf("expected DefaultParams TierConfigs length=4, got %d", len(params.TierConfigs))
	}
	if params.UnbondingPeriod != 268560 {
		t.Errorf("expected default UnbondingPeriod=268560, got %d", params.UnbondingPeriod)
	}
}

// ============================================================
// TASK 2: SetParams → KVStore Sync Integrity
// ============================================================

func TestSetParams_SyncsToKVStore(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Step 1: Modify tier configs via governance (SetParams with different min_stake)
	params := k.GetParams(ctx)
	if len(params.TierConfigs) < 4 {
		t.Fatalf("expected at least 4 TierConfigs, got %d", len(params.TierConfigs))
	}

	// Change apprentice min_stake to 222000
	params.TierConfigs[0].MinStake = "222000"
	// Change bonded slash multiplier to 2000 (2.0x)
	params.TierConfigs[2].SlashMultiplierBps = 2000

	if err := k.SetParams(ctx, params); err != nil {
		t.Fatalf("SetParams failed: %v", err)
	}

	// Step 2: Read back via GetParams → verify new values
	readParams := k.GetParams(ctx)
	if readParams.TierConfigs[0].MinStake != "222000" {
		t.Errorf("GetParams: expected apprentice MinStake=222000, got %s", readParams.TierConfigs[0].MinStake)
	}
	if readParams.TierConfigs[2].SlashMultiplierBps != 2000 {
		t.Errorf("GetParams: expected bonded SlashMultiplierBps=2000, got %d", readParams.TierConfigs[2].SlashMultiplierBps)
	}

	// Step 3: Read individual tier config from KVStore → verify synced
	apprenticeConfig, found := k.GetTierConfig(ctx, types.TierApprentice)
	if !found {
		t.Fatal("apprentice tier config not found in KVStore")
	}
	if apprenticeConfig.MinStake != "222000" {
		t.Errorf("KVStore: expected apprentice MinStake=222000, got %s", apprenticeConfig.MinStake)
	}

	bondedConfig, found := k.GetTierConfig(ctx, types.TierBonded)
	if !found {
		t.Fatal("bonded tier config not found in KVStore")
	}
	if bondedConfig.SlashMultiplierBps != 2000 {
		t.Errorf("KVStore: expected bonded SlashMultiplierBps=2000, got %d", bondedConfig.SlashMultiplierBps)
	}

	// Step 4: Run CalculateTierSlash → verify it uses the new slash multiplier
	// bonded tier now has 2000 BPS = 2.0x (was 1000 = 1.0x)
	slash := k.CalculateTierSlash(ctx, types.TierBonded, "1000")
	if slash != "2000" {
		t.Errorf("CalculateTierSlash: expected 2000 (1000 × 2.0x), got %s", slash)
	}

	// Verify apprentice slash multiplier unchanged (1500 BPS = 1.5x)
	slashApp := k.CalculateTierSlash(ctx, types.TierApprentice, "1000")
	if slashApp != "1500" {
		t.Errorf("CalculateTierSlash apprentice: expected 1500, got %s", slashApp)
	}
}

func TestSetParams_MultipleUpdatesOverwrite(t *testing.T) {
	k, ctx := setupKeeper(t)

	// First update
	params := k.GetParams(ctx)
	params.TierConfigs[1].MinStake = "5555555"
	k.SetParams(ctx, params)

	// Verify
	config, found := k.GetTierConfig(ctx, types.TierVerified)
	if !found {
		t.Fatal("verified config not found")
	}
	if config.MinStake != "5555555" {
		t.Errorf("first update: expected MinStake=5555555, got %s", config.MinStake)
	}

	// Second update overwrites
	params2 := k.GetParams(ctx)
	params2.TierConfigs[1].MinStake = "7777777"
	k.SetParams(ctx, params2)

	config2, found := k.GetTierConfig(ctx, types.TierVerified)
	if !found {
		t.Fatal("verified config not found after second update")
	}
	if config2.MinStake != "7777777" {
		t.Errorf("second update: expected MinStake=7777777, got %s", config2.MinStake)
	}
}

// ============================================================
// TASK 3: Autopoiesis SSI Multiplier Path
// Tests the full chain: parameterized SlashMultiplierBps × SSI multiplier
// ============================================================

func TestSlashValidator_SSIMultiplier(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_ssi_test__").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:ssi", "1111000000")

	// Override to precise values for testing
	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "10000"
	v.DelegatedStake = "0"
	v.TotalStake = "10000"
	v.Tier = "bonded"
	k.SetValidator(ctx, v)

	// Set up mock autopoiesis keeper with SSI multiplier 0.8x (800000 in BPS)
	mockAuto := newMockAutopoiesisKeeper()
	mockAuto.multipliers["system.slash"] = 800_000 // 0.8x
	k.SetAutopoiesisKeeper(mockAuto)

	// Slash 1000 base amount
	// Expected: base_slash=1000, escalation=0 (first slash), SSI=0.8x → effective=800
	err := k.SlashValidator(ctx, valAddr, "1000", "ssi_test")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found after slash")
	}

	// 10000 - 800 = 9200
	if v.Stake != "9200" {
		t.Errorf("expected stake=9200 (base 1000 × 0.8 SSI = 800), got %s", v.Stake)
	}
}

func TestSlashValidator_SSIMultiplierHigh(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_ssi_high__").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:ssih", "1111000000")

	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "10000"
	v.DelegatedStake = "0"
	v.TotalStake = "10000"
	v.Tier = "bonded"
	k.SetValidator(ctx, v)

	// SSI multiplier 2.0x (system under stress)
	mockAuto := newMockAutopoiesisKeeper()
	mockAuto.multipliers["system.slash"] = 2_000_000 // 2.0x
	k.SetAutopoiesisKeeper(mockAuto)

	// Slash 1000 base → SSI 2.0x → effective 2000
	err := k.SlashValidator(ctx, valAddr, "1000", "ssi_high_test")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}

	// 10000 - 2000 = 8000
	if v.Stake != "8000" {
		t.Errorf("expected stake=8000 (base 1000 × 2.0 SSI = 2000), got %s", v.Stake)
	}
}

func TestSlashValidator_SSIMultiplierDefault(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_ssi_def___").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:ssidef", "1111000000")

	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "10000"
	v.DelegatedStake = "0"
	v.TotalStake = "10000"
	v.Tier = "bonded"
	k.SetValidator(ctx, v)

	// SSI multiplier 1.0x (default — should be skipped)
	mockAuto := newMockAutopoiesisKeeper()
	mockAuto.multipliers["system.slash"] = 1_000_000 // 1.0x = default, skipped
	k.SetAutopoiesisKeeper(mockAuto)

	// Slash 1000 → no SSI effect → effective 1000
	err := k.SlashValidator(ctx, valAddr, "1000", "ssi_default_test")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}

	// 10000 - 1000 = 9000 (no SSI modification)
	if v.Stake != "9000" {
		t.Errorf("expected stake=9000 (no SSI at 1.0x), got %s", v.Stake)
	}
}

func TestSlashValidator_SSIWithEscalation(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_ssiesc____").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:ssiesc", "1111000000")

	// Allow more slashes
	params := k.GetParams(ctx)
	params.MaxSlashesPerEpoch = 5
	params.MaxSlashCountDeactivate = 10
	k.SetParams(ctx, params)

	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "100000"
	v.DelegatedStake = "0"
	v.TotalStake = "100000"
	v.Tier = "bonded"
	v.SlashCount = 1 // One prior slash for escalation
	k.SetValidator(ctx, v)

	// SSI multiplier 0.5x (dampened)
	mockAuto := newMockAutopoiesisKeeper()
	mockAuto.multipliers["system.slash"] = 500_000 // 0.5x
	k.SetAutopoiesisKeeper(mockAuto)

	// Slash 10000 base
	// Escalation: 10000 * (1M + 1*100000) / 1M = 10000 * 1.1 = 11000
	// SSI: 11000 * 500000 / 1000000 = 5500
	err := k.SlashValidator(ctx, valAddr, "10000", "ssi_esc_test")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}

	// 100000 - 5500 = 94500
	if v.Stake != "94500" {
		t.Errorf("expected stake=94500 (base 10000 × 1.1 escalation × 0.5 SSI = 5500), got %s", v.Stake)
	}
}

func TestSlashValidator_NilAutopoiesisKeeper(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_nilauto___").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:nilauto", "1111000000")

	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "10000"
	v.DelegatedStake = "0"
	v.TotalStake = "10000"
	v.Tier = "bonded"
	k.SetValidator(ctx, v)

	// Do NOT set autopoiesis keeper — it should be nil
	// Slash 1000 → no SSI effect → effective 1000
	err := k.SlashValidator(ctx, valAddr, "1000", "nil_auto_test")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}

	// 10000 - 1000 = 9000
	if v.Stake != "9000" {
		t.Errorf("expected stake=9000 (nil autopoiesis keeper), got %s", v.Stake)
	}
}

// ============================================================
// TASK 4: Validation Edge Cases
// ============================================================

func TestParams_Validate_TierConfigsExactly4(t *testing.T) {
	p := types.DefaultParams()
	if err := p.Validate(); err != nil {
		t.Errorf("valid params should pass: %v", err)
	}
}

func TestParams_Validate_TierConfigs3_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.TierConfigs = p.TierConfigs[:3] // Remove guardian
	if err := p.Validate(); err == nil {
		t.Error("expected error for 3 tier configs")
	}
}

func TestParams_Validate_TierConfigs5_Fails(t *testing.T) {
	p := types.DefaultParams()
	extra := &types.TierConfig{Tier: "extra", MinStake: "999", RewardMultiplier: 100, SelectionWeight: 100, SlashMultiplierBps: 1000}
	p.TierConfigs = append(p.TierConfigs, extra)
	if err := p.Validate(); err == nil {
		t.Error("expected error for 5 tier configs")
	}
}

func TestParams_Validate_TierConfigMinStakeZero_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.TierConfigs[0].MinStake = "0"
	if err := p.Validate(); err == nil {
		t.Error("expected error for min_stake=0")
	}
}

func TestParams_Validate_TierConfigMinStakeNegative_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.TierConfigs[0].MinStake = "-1"
	if err := p.Validate(); err == nil {
		t.Error("expected error for min_stake=-1")
	}
}

func TestParams_Validate_TierConfigMinStakeInvalid_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.TierConfigs[0].MinStake = "not_a_number"
	if err := p.Validate(); err == nil {
		t.Error("expected error for non-numeric min_stake")
	}
}

func TestParams_Validate_WrongTierOrdering_Fails(t *testing.T) {
	p := types.DefaultParams()
	// Swap apprentice and guardian
	p.TierConfigs[0], p.TierConfigs[3] = p.TierConfigs[3], p.TierConfigs[0]
	if err := p.Validate(); err == nil {
		t.Error("expected error for wrong tier ordering (guardian before apprentice)")
	}
}

func TestParams_Validate_RewardMultiplierZero_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.TierConfigs[0].RewardMultiplier = 0
	if err := p.Validate(); err == nil {
		t.Error("expected error for reward_multiplier=0")
	}
}

func TestParams_Validate_SelectionWeightZero_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.TierConfigs[0].SelectionWeight = 0
	if err := p.Validate(); err == nil {
		t.Error("expected error for selection_weight=0")
	}
}

func TestParams_Validate_SlashMultiplierBpsZero_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.TierConfigs[0].SlashMultiplierBps = 0
	if err := p.Validate(); err == nil {
		t.Error("expected error for slash_multiplier_bps=0")
	}
}

func TestParams_Validate_SlashMultiplierBpsOver10000_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.TierConfigs[0].SlashMultiplierBps = 10001
	if err := p.Validate(); err == nil {
		t.Error("expected error for slash_multiplier_bps=10001")
	}
}

func TestParams_Validate_MinAccuracyOver1M_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.TierConfigs[1].MinAccuracy = 1000001
	if err := p.Validate(); err == nil {
		t.Error("expected error for min_accuracy > 1000000")
	}
}

func TestParams_Validate_ReputationCorrectDeltaZero_Allowed(t *testing.T) {
	// Decision: ReputationCorrectDelta=0 is allowed for backward compat.
	// GetParams fills it with defaults, but Validate doesn't reject 0.
	p := types.DefaultParams()
	p.ReputationCorrectDelta = 0
	if err := p.Validate(); err != nil {
		t.Errorf("ReputationCorrectDelta=0 should be allowed (backward compat): %v", err)
	}
}

func TestParams_Validate_ReputationCorrectDeltaOverLimit_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.ReputationCorrectDelta = 1000001
	if err := p.Validate(); err == nil {
		t.Error("expected error for ReputationCorrectDelta > 1000000")
	}
}

func TestParams_Validate_RedelegationCooldownZero_Allowed(t *testing.T) {
	// Decision: RedelegationCooldownBlocks=0 is allowed (instant redelegation).
	// This could be used in testing or governance-approved scenarios.
	// GetParams fills 0 with default, but Validate accepts 0.
	p := types.DefaultParams()
	p.RedelegationCooldownBlocks = 0
	if err := p.Validate(); err != nil {
		t.Errorf("RedelegationCooldownBlocks=0 should be allowed: %v", err)
	}
}

func TestParams_Validate_SlashEscalationBpsOverLimit_Fails(t *testing.T) {
	p := types.DefaultParams()
	p.SlashEscalationBps = 1000001
	if err := p.Validate(); err == nil {
		t.Error("expected error for SlashEscalationBps > 1000000")
	}
}

func TestParams_Validate_EmptyTierConfigs_Allowed(t *testing.T) {
	// Empty TierConfigs allowed for backward compat (GetParams fills from KVStore)
	p := types.DefaultParams()
	p.TierConfigs = nil
	if err := p.Validate(); err != nil {
		t.Errorf("empty TierConfigs should be allowed (backward compat): %v", err)
	}
}

// ============================================================
// TASK 5: Genesis Round-Trip
// ============================================================

func TestGenesis_RoundTrip(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register a validator for richer genesis state
	valAddr := sdk.AccAddress("validator_genesis____").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:genesis", "1111000000")

	// Modify params to non-default values
	params := k.GetParams(ctx)
	params.TierConfigs[0].MinStake = "333000"
	params.ReputationCorrectDelta = 150
	params.ReputationIncorrectDelta = 300
	params.ReputationSlashDelta = 15000
	params.RedelegationCooldownBlocks = 2222
	k.SetParams(ctx, params)

	// Export genesis
	gs1 := k.ExportGenesis(ctx)

	// Verify tier configs in params
	if len(gs1.Params.TierConfigs) != 4 {
		t.Fatalf("exported genesis should have 4 TierConfigs, got %d", len(gs1.Params.TierConfigs))
	}
	if gs1.Params.TierConfigs[0].MinStake != "333000" {
		t.Errorf("exported apprentice MinStake: expected 333000, got %s", gs1.Params.TierConfigs[0].MinStake)
	}
	if gs1.Params.ReputationCorrectDelta != 150 {
		t.Errorf("exported ReputationCorrectDelta: expected 150, got %d", gs1.Params.ReputationCorrectDelta)
	}

	// Create fresh keeper and import
	k2, ctx2 := setupBareKeeper(t)
	if err := k2.InitGenesis(ctx2, gs1); err != nil {
		t.Fatalf("InitGenesis failed: %v", err)
	}

	// Export from fresh keeper
	gs2 := k2.ExportGenesis(ctx2)

	// Compare key fields
	if gs2.Params.TierConfigs[0].MinStake != gs1.Params.TierConfigs[0].MinStake {
		t.Errorf("round-trip MinStake mismatch: %s vs %s",
			gs1.Params.TierConfigs[0].MinStake, gs2.Params.TierConfigs[0].MinStake)
	}
	if gs2.Params.ReputationCorrectDelta != gs1.Params.ReputationCorrectDelta {
		t.Errorf("round-trip ReputationCorrectDelta mismatch: %d vs %d",
			gs1.Params.ReputationCorrectDelta, gs2.Params.ReputationCorrectDelta)
	}
	if gs2.Params.ReputationIncorrectDelta != gs1.Params.ReputationIncorrectDelta {
		t.Errorf("round-trip ReputationIncorrectDelta mismatch")
	}
	if gs2.Params.ReputationSlashDelta != gs1.Params.ReputationSlashDelta {
		t.Errorf("round-trip ReputationSlashDelta mismatch")
	}
	if gs2.Params.RedelegationCooldownBlocks != gs1.Params.RedelegationCooldownBlocks {
		t.Errorf("round-trip RedelegationCooldownBlocks mismatch")
	}

	// Verify all 4 tier configs survive exactly
	for i := 0; i < 4; i++ {
		tc1 := gs1.Params.TierConfigs[i]
		tc2 := gs2.Params.TierConfigs[i]
		if tc1.Tier != tc2.Tier {
			t.Errorf("round-trip tier[%d] name mismatch: %s vs %s", i, tc1.Tier, tc2.Tier)
		}
		if tc1.MinStake != tc2.MinStake {
			t.Errorf("round-trip tier[%d] MinStake mismatch: %s vs %s", i, tc1.MinStake, tc2.MinStake)
		}
		if tc1.MinVerifications != tc2.MinVerifications {
			t.Errorf("round-trip tier[%d] MinVerifications mismatch", i)
		}
		if tc1.MinAccuracy != tc2.MinAccuracy {
			t.Errorf("round-trip tier[%d] MinAccuracy mismatch", i)
		}
		if tc1.RewardMultiplier != tc2.RewardMultiplier {
			t.Errorf("round-trip tier[%d] RewardMultiplier mismatch", i)
		}
		if tc1.SelectionWeight != tc2.SelectionWeight {
			t.Errorf("round-trip tier[%d] SelectionWeight mismatch", i)
		}
		if tc1.SlashMultiplierBps != tc2.SlashMultiplierBps {
			t.Errorf("round-trip tier[%d] SlashMultiplierBps mismatch: %d vs %d",
				i, tc1.SlashMultiplierBps, tc2.SlashMultiplierBps)
		}
	}

	// Verify validators survived
	if len(gs2.Validators) != len(gs1.Validators) {
		t.Errorf("validator count mismatch: %d vs %d", len(gs1.Validators), len(gs2.Validators))
	}
}

func TestGenesis_KVStoreTierConfigsMatchParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Modify a tier config via params
	params := k.GetParams(ctx)
	params.TierConfigs[3].MinStake = "99999000000"
	k.SetParams(ctx, params)

	// Verify KVStore matches
	guardianConfig, found := k.GetTierConfig(ctx, types.TierGuardian)
	if !found {
		t.Fatal("guardian config not found")
	}
	if guardianConfig.MinStake != "99999000000" {
		t.Errorf("KVStore guardian MinStake: expected 99999000000, got %s", guardianConfig.MinStake)
	}

	// Export and reimport
	gs := k.ExportGenesis(ctx)
	k2, ctx2 := setupBareKeeper(t)
	if err := k2.InitGenesis(ctx2, gs); err != nil {
		t.Fatalf("InitGenesis failed: %v", err)
	}

	// Verify KVStore in new keeper
	guardianConfig2, found := k2.GetTierConfig(ctx2, types.TierGuardian)
	if !found {
		t.Fatal("guardian config not found after reimport")
	}
	if guardianConfig2.MinStake != "99999000000" {
		t.Errorf("reimported guardian MinStake: expected 99999000000, got %s", guardianConfig2.MinStake)
	}
}

// ============================================================
// TASK 6: Deprecated Field Cleanup
// ============================================================

func TestGenesis_DeprecatedTierConfigsField(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Export genesis — TierConfigs field is populated (sourced from params)
	gs := k.ExportGenesis(ctx)
	if len(gs.TierConfigs) != 4 {
		t.Fatalf("exported genesis.TierConfigs should be 4 (for backward compat), got %d", len(gs.TierConfigs))
	}

	// The TierConfigs should be sourced from params
	if gs.Params == nil || len(gs.Params.TierConfigs) != 4 {
		t.Fatal("exported genesis.Params.TierConfigs should be present")
	}

	// Both should point to the same data
	for i := 0; i < 4; i++ {
		if gs.TierConfigs[i].Tier != gs.Params.TierConfigs[i].Tier {
			t.Errorf("genesis.TierConfigs[%d] and genesis.Params.TierConfigs[%d] should match", i, i)
		}
	}
}

func TestGenesis_LegacyTierConfigsOnly(t *testing.T) {
	k, ctx := setupBareKeeper(t)

	// Create genesis with tier configs ONLY in the legacy field (no params.TierConfigs)
	defaults := types.DefaultTierConfigs()
	dp := types.DefaultParams()
	dp.TierConfigs = nil // Simulate old genesis format
	legacyGenesis := &types.GenesisState{
		Params:      &dp,
		TierConfigs: defaults,
	}

	// Validate should pass (tier configs in legacy field)
	if err := legacyGenesis.Validate(); err != nil {
		t.Fatalf("legacy genesis validation failed: %v", err)
	}

	// InitGenesis should handle legacy field
	if err := k.InitGenesis(ctx, legacyGenesis); err != nil {
		t.Fatalf("InitGenesis with legacy format failed: %v", err)
	}

	// Verify tier configs were written to KVStore from legacy field
	configs := k.GetAllTierConfigs(ctx)
	if len(configs) != 4 {
		t.Fatalf("expected 4 tier configs in KVStore after legacy import, got %d", len(configs))
	}

	// GetParams should fill TierConfigs from KVStore
	params := k.GetParams(ctx)
	if len(params.TierConfigs) != 4 {
		t.Fatalf("GetParams should fill TierConfigs from KVStore, got %d", len(params.TierConfigs))
	}
}

func TestGenesis_NewFormatTierConfigsInParams(t *testing.T) {
	k, ctx := setupBareKeeper(t)

	// Create genesis with tier configs IN params (new format)
	dp := types.DefaultParams()
	newGenesis := &types.GenesisState{
		Params:      &dp,
		TierConfigs: nil, // New format — no legacy field
	}

	// Validate should pass (tier configs in params)
	if err := newGenesis.Validate(); err != nil {
		t.Fatalf("new format genesis validation failed: %v", err)
	}

	// InitGenesis should use params.TierConfigs
	if err := k.InitGenesis(ctx, newGenesis); err != nil {
		t.Fatalf("InitGenesis with new format failed: %v", err)
	}

	// Verify tier configs synced to KVStore from params
	configs := k.GetAllTierConfigs(ctx)
	if len(configs) != 4 {
		t.Fatalf("expected 4 tier configs in KVStore after new-format import, got %d", len(configs))
	}
}

func TestGenesis_BothTierConfigLocations_ParamsWins(t *testing.T) {
	k, ctx := setupBareKeeper(t)

	// Create genesis with tier configs in BOTH locations (params and legacy)
	// Params should win per InitGenesis logic.
	dp := types.DefaultParams()
	dp.TierConfigs[0].MinStake = "999000" // Custom value in params

	legacyConfigs := types.DefaultTierConfigs()
	legacyConfigs[0].MinStake = "111000" // Original value in legacy

	bothGenesis := &types.GenesisState{
		Params:      &dp,
		TierConfigs: legacyConfigs,
	}

	if err := k.InitGenesis(ctx, bothGenesis); err != nil {
		t.Fatalf("InitGenesis failed: %v", err)
	}

	// Params version should win
	config, found := k.GetTierConfig(ctx, types.TierApprentice)
	if !found {
		t.Fatal("apprentice config not found")
	}
	if config.MinStake != "999000" {
		t.Errorf("expected params MinStake=999000 to win over legacy, got %s", config.MinStake)
	}
}

func TestGenesisState_Validate_NoTierConfigsAnywhere_Fails(t *testing.T) {
	dp := types.DefaultParams()
	dp.TierConfigs = nil

	gs := &types.GenesisState{
		Params:      &dp,
		TierConfigs: nil,
	}

	if err := gs.Validate(); err == nil {
		t.Error("expected error when tier configs missing from both params and genesis")
	}
}

// ============================================================
// TASK 2 supplement: UpdateParams governance integration
// ============================================================

func TestUpdateParams_GovernanceFlow(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Simulate governance updating params with new tier configs
	newParams := types.DefaultParams()
	newParams.TierConfigs[2].MinStake = "5555000000" // Change bonded min stake

	_, err := k.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "authority",
		Params:    &newParams,
	})
	if err != nil {
		t.Fatalf("UpdateParams failed: %v", err)
	}

	// Verify
	params := k.GetParams(ctx)
	if params.TierConfigs[2].MinStake != "5555000000" {
		t.Errorf("expected bonded MinStake=5555000000 after governance update, got %s",
			params.TierConfigs[2].MinStake)
	}

	// Verify KVStore synced
	config, found := k.GetTierConfig(ctx, types.TierBonded)
	if !found {
		t.Fatal("bonded config not found after governance update")
	}
	if config.MinStake != "5555000000" {
		t.Errorf("KVStore bonded MinStake mismatch: expected 5555000000, got %s", config.MinStake)
	}
}

func TestUpdateParams_UnauthorizedFails(t *testing.T) {
	k, ctx := setupKeeper(t)

	newParams := types.DefaultParams()
	_, err := k.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "not_authority",
		Params:    &newParams,
	})
	if err == nil {
		t.Error("expected error for unauthorized UpdateParams")
	}
}

// ============================================================
// TASK 3 supplement: RecordVerification uses parameterized reputation
// ============================================================

func TestRecordVerification_UsesParameterizedReputation(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_paramrep__").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:paramrep", "1111000000")

	// Override reputation deltas
	params := k.GetParams(ctx)
	params.ReputationCorrectDelta = 500   // +0.05%
	params.ReputationIncorrectDelta = 1000 // -0.1%
	k.SetParams(ctx, params)

	// Set validator to known reputation
	v, _ := k.GetValidator(ctx, valAddr)
	v.Reputation = 500000 // 50%
	k.SetValidator(ctx, v)

	// Correct verification
	k.RecordVerification(ctx, valAddr, true, false)
	v, _ = k.GetValidator(ctx, valAddr)
	if v.Reputation != 500500 { // 500000 + 500
		t.Errorf("expected reputation=500500 after correct (+500), got %d", v.Reputation)
	}

	// Incorrect verification
	k.RecordVerification(ctx, valAddr, false, false)
	v, _ = k.GetValidator(ctx, valAddr)
	if v.Reputation != 499500 { // 500500 - 1000
		t.Errorf("expected reputation=499500 after incorrect (-1000), got %d", v.Reputation)
	}
}

// ============================================================
// TASK 4 supplement: Redelegation cooldown with governance-adjusted blocks
// ============================================================

func TestRedelegate_CooldownFromParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	val1Addr := sdk.AccAddress("validator_cd1_______").String()
	val2Addr := sdk.AccAddress("validator_cd2_______").String()
	val3Addr := sdk.AccAddress("validator_cd3_______").String()
	registerTestValidator(t, k, ctx, val1Addr, "did:lgm:cd1", "1000000000000000000000")
	registerTestValidator(t, k, ctx, val2Addr, "did:lgm:cd2", "1000000000000000000000")
	registerTestValidator(t, k, ctx, val3Addr, "did:lgm:cd3", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator_cd________").String()
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: val1Addr, Amount: "500000000000000000000",
	})

	// Set custom cooldown to 500 blocks
	params := k.GetParams(ctx)
	params.RedelegationCooldownBlocks = 500
	k.SetParams(ctx, params)

	// First redelegate at height 100 should succeed
	_, err := k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val1Addr, DstValidator: val2Addr,
		Amount: "100000000000000000000",
	})
	if err != nil {
		t.Fatalf("first redelegate should succeed: %v", err)
	}

	// Second redelegate at height 100 should fail (within 500-block cooldown)
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: val2Addr, Amount: "100000000000000000000",
	})
	_, err = k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val2Addr, DstValidator: val3Addr,
		Amount: "100000000000000000000",
	})
	if err == nil {
		t.Error("expected cooldown error for rapid redelegate")
	}

	// Advance past cooldown
	ctx = ctx.WithBlockHeight(601)
	_, err = k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val2Addr, DstValidator: val3Addr,
		Amount: "100000000000000000000",
	})
	if err != nil {
		t.Fatalf("redelegate after cooldown should succeed: %v", err)
	}
}
