package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/autopoiesis/keeper"
	"github.com/legible-money/legible/x/autopoiesis/types"
)

const testAuthority = "lgm1authority"

func setupMsgServer(t *testing.T) (types.MsgServer, keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	mockSK := &mockStakingKeeper{activeValidators: 20, totalValidators: 22, avgAccuracy: 900_000}
	mockKK := &mockKnowledgeKeeper{activeFacts: 100, factsCreated: 10, factsExpired: 2}
	mockEK := &mockEmergencyKeeper{halted: false}
	storeService := runtime.NewKVStoreService(storeKey)

	k := keeper.NewKeeper(storeService, log.NewNopLogger(), mockSK, mockKK, mockEK, testAuthority)
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
	k.InitGenesis(ctx, types.DefaultGenesis())

	ms := keeper.NewMsgServerImpl(k)
	return ms, k, ctx
}

// ---------- UpdateParams Tests ----------

func TestUpdateParams_Success(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	newParams := types.Params{
		EpochLengthBlocks:       2222,
		MaxChangePerEpochBps:    100_000,
		SlashMultiplierMin:      500_000,
		SlashMultiplierMax:      3_000_000,
		SSICriticalThreshold:    900_000,
		SSIModerateThreshold:    600_000,
		SSILowModerateThreshold: 300_000,
		Enabled:                 true,
	}

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAuthority,
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("UpdateParams failed: %v", err)
	}

	stored := k.GetParams(ctx)
	if stored.EpochLengthBlocks != 2222 {
		t.Fatalf("expected epoch_length_blocks=2222, got %d", stored.EpochLengthBlocks)
	}
	if stored.SSICriticalThreshold != 900_000 {
		t.Fatalf("expected ssi_critical=900000, got %d", stored.SSICriticalThreshold)
	}
}

func TestUpdateParams_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	_, err := ms.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "lgm1wrongauthority",
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Fatal("expected error for unauthorized caller")
	}
}

func TestUpdateParams_InvalidParams(t *testing.T) {
	msg := &types.MsgUpdateParams{
		Authority: testAuthority,
		Params: types.Params{
			EpochLengthBlocks: 0, // invalid
		},
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Fatal("expected ValidateBasic to reject zero epoch length")
	}
}

// ---------- Activate Tests ----------

func TestActivate_Enable(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	// Deactivate first
	_, err := ms.Activate(ctx, &types.MsgActivateAutopoiesis{
		Authority: testAuthority,
		Active:    false,
	})
	if err != nil {
		t.Fatalf("Deactivate failed: %v", err)
	}
	if k.IsActive(ctx) {
		t.Fatal("expected autopoiesis to be deactivated")
	}

	// Re-activate
	_, err = ms.Activate(ctx, &types.MsgActivateAutopoiesis{
		Authority: testAuthority,
		Active:    true,
	})
	if err != nil {
		t.Fatalf("Activate failed: %v", err)
	}
	if !k.IsActive(ctx) {
		t.Fatal("expected autopoiesis to be activated")
	}
}

func TestActivate_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	_, err := ms.Activate(ctx, &types.MsgActivateAutopoiesis{
		Authority: "lgm1wrongauthority",
		Active:    false,
	})
	if err == nil {
		t.Fatal("expected error for unauthorized caller")
	}
}

// ---------- OverrideMultiplier Tests ----------

func TestOverrideMultiplier_Success(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	_, err := ms.OverrideMultiplier(ctx, &types.MsgOverrideMultiplier{
		Authority: testAuthority,
		Path:      types.PathSlashSystem,
		Value:     2_000_000, // 2.0x
	})
	if err != nil {
		t.Fatalf("OverrideMultiplier failed: %v", err)
	}

	val := k.GetMultiplier(ctx, types.PathSlashSystem)
	if val != 2_000_000 {
		t.Fatalf("expected multiplier=2000000, got %d", val)
	}

	entry := k.GetMultiplierEntry(ctx, types.PathSlashSystem)
	if entry == nil {
		t.Fatal("expected multiplier entry to exist")
	}
	if entry.Hormone != "governance" {
		t.Fatalf("expected hormone=governance, got %s", entry.Hormone)
	}
}

func TestOverrideMultiplier_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	_, err := ms.OverrideMultiplier(ctx, &types.MsgOverrideMultiplier{
		Authority: "lgm1wrongauthority",
		Path:      types.PathSlashSystem,
		Value:     2_000_000,
	})
	if err == nil {
		t.Fatal("expected error for unauthorized caller")
	}
}

func TestOverrideMultiplier_ValidateBasic(t *testing.T) {
	msg := &types.MsgOverrideMultiplier{
		Authority: testAuthority,
		Path:      "",
		Value:     1_000_000,
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Fatal("expected error for empty path")
	}

	msg.Path = types.PathSlashSystem
	msg.Value = 0
	if err := msg.ValidateBasic(); err == nil {
		t.Fatal("expected error for zero value")
	}
}

// ---------- FreezeMultiplier Tests ----------

func TestFreezeMultiplier_FreezeAndUnfreeze(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	// Freeze
	_, err := ms.FreezeMultiplier(ctx, &types.MsgFreezeMultiplier{
		Authority: testAuthority,
		Path:      types.PathSlashSystem,
		Frozen:    true,
	})
	if err != nil {
		t.Fatalf("FreezeMultiplier failed: %v", err)
	}
	if !k.IsMultiplierFrozen(ctx, types.PathSlashSystem) {
		t.Fatal("expected multiplier to be frozen")
	}

	// Unfreeze
	_, err = ms.FreezeMultiplier(ctx, &types.MsgFreezeMultiplier{
		Authority: testAuthority,
		Path:      types.PathSlashSystem,
		Frozen:    false,
	})
	if err != nil {
		t.Fatalf("UnfreezeMultiplier failed: %v", err)
	}
	if k.IsMultiplierFrozen(ctx, types.PathSlashSystem) {
		t.Fatal("expected multiplier to be unfrozen")
	}
}

func TestFreezeMultiplier_Unauthorized(t *testing.T) {
	ms, _, ctx := setupMsgServer(t)

	_, err := ms.FreezeMultiplier(ctx, &types.MsgFreezeMultiplier{
		Authority: "lgm1wrongauthority",
		Path:      types.PathSlashSystem,
		Frozen:    true,
	})
	if err == nil {
		t.Fatal("expected error for unauthorized caller")
	}
}

// ---------- Frozen Multiplier Skipped During Adaptation ----------

func TestFrozenMultiplierSkippedDuringAdaptation(t *testing.T) {
	ms, k, ctx := setupMsgServer(t)

	// Set an initial multiplier value
	k.SetMultiplier(ctx, types.MultiplierEntry{
		Path:    types.PathSlashSystem,
		Value:   1_000_000,
		Hormone: "SSI",
		Epoch:   0,
	})

	// Freeze it
	_, err := ms.FreezeMultiplier(ctx, &types.MsgFreezeMultiplier{
		Authority: testAuthority,
		Path:      types.PathSlashSystem,
		Frozen:    true,
	})
	if err != nil {
		t.Fatalf("FreezeMultiplier failed: %v", err)
	}

	// Verify frozen status
	if !k.IsMultiplierFrozen(ctx, types.PathSlashSystem) {
		t.Fatal("expected slash.system to be frozen")
	}

	// Non-frozen path should not be frozen
	if k.IsMultiplierFrozen(ctx, types.PathBlockRewardSystem) {
		t.Fatal("expected blockreward.system to NOT be frozen")
	}
}
