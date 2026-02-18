package keeper_test

import (
	"testing"

	"cosmossdk.io/core/store"
	corestore "cosmossdk.io/store"
	"cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	addresscodec "github.com/cosmos/cosmos-sdk/codec/address"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"

	"github.com/legible-money/legible/x/ibcratelimit/keeper"
	"github.com/legible-money/legible/x/ibcratelimit/types"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
}

// setupKeeper creates a Keeper backed by an in-memory KV store for testing.
func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := corestore.NewCommitMultiStore(db, log.NewNopLogger(), metrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	interfaceRegistry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(interfaceRegistry)
	_ = authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
	_ = addresscodec.NewBech32Codec("lgm")

	var storeService store.KVStoreService = runtime.NewKVStoreService(storeKey)

	authority := "lgm10d07y265gmmuvt4z0w9aw880jnsr700jvss730" // dummy governance address

	k := keeper.NewKeeper(cdc, storeService, authority)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx
}

// ============================================================
// Params Tests
// ============================================================

func TestDefaultParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	params := k.GetParams(ctx)

	if params.EpochLengthBlocks != 34272 {
		t.Errorf("expected EpochLengthBlocks=34272, got %d", params.EpochLengthBlocks)
	}
	if params.MaxOutflowPerEpoch != "10000000000" {
		t.Errorf("expected MaxOutflowPerEpoch=10000000000, got %s", params.MaxOutflowPerEpoch)
	}
	if params.MaxInflowPerEpoch != "10000000000" {
		t.Errorf("expected MaxInflowPerEpoch=10000000000, got %s", params.MaxInflowPerEpoch)
	}
	if params.TrackedDenom != "ulgm" {
		t.Errorf("expected TrackedDenom=ulgm, got %s", params.TrackedDenom)
	}
}

func TestSetGetParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	custom := types.Params{
		EpochLengthBlocks:  1000,
		MaxOutflowPerEpoch: "500000",
		MaxInflowPerEpoch:  "600000",
		TrackedDenom:       "uatom",
	}
	k.SetParams(ctx, &custom)

	got := k.GetParams(ctx)
	if got.EpochLengthBlocks != 1000 {
		t.Errorf("EpochLengthBlocks: want 1000, got %d", got.EpochLengthBlocks)
	}
	if got.MaxOutflowPerEpoch != "500000" {
		t.Errorf("MaxOutflowPerEpoch: want 500000, got %s", got.MaxOutflowPerEpoch)
	}
	if got.MaxInflowPerEpoch != "600000" {
		t.Errorf("MaxInflowPerEpoch: want 600000, got %s", got.MaxInflowPerEpoch)
	}
	if got.TrackedDenom != "uatom" {
		t.Errorf("TrackedDenom: want uatom, got %s", got.TrackedDenom)
	}
}

func TestParamsValidation(t *testing.T) {
	tests := []struct {
		name    string
		params  types.Params
		wantErr bool
	}{
		{"valid", types.DefaultParams(), false},
		{"zero epoch", types.Params{EpochLengthBlocks: 0, TrackedDenom: "ulgm"}, true},
		{"empty denom", types.Params{EpochLengthBlocks: 100, TrackedDenom: ""}, true},
	}
	for i := range tests {
		tc := &tests[i]
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if (err != nil) != tc.wantErr {
				t.Errorf("Validate() error = %v, wantErr %v", err, tc.wantErr)
			}
		})
	}
}

// ============================================================
// Flow CRUD Tests
// ============================================================

func TestSetGetFlow(t *testing.T) {
	k, ctx := setupKeeper(t)

	flow := types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "1000",
		InflowTotal:     "2000",
		EpochStartBlock: 50,
	}
	k.SetFlow(ctx, &flow)

	got, found := k.GetFlow(ctx, "channel-0")
	if !found {
		t.Fatal("expected flow to be found")
	}
	if got.ChannelId != "channel-0" {
		t.Errorf("ChannelId: want channel-0, got %s", got.ChannelId)
	}
	if got.OutflowTotal != "1000" {
		t.Errorf("OutflowTotal: want 1000, got %s", got.OutflowTotal)
	}
	if got.InflowTotal != "2000" {
		t.Errorf("InflowTotal: want 2000, got %s", got.InflowTotal)
	}
	if got.EpochStartBlock != 50 {
		t.Errorf("EpochStartBlock: want 50, got %d", got.EpochStartBlock)
	}
}

func TestGetFlowNotFound(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, found := k.GetFlow(ctx, "channel-nonexistent")
	if found {
		t.Error("expected flow not to be found")
	}
}

func TestDeleteFlow(t *testing.T) {
	k, ctx := setupKeeper(t)

	flow := types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "100",
		InflowTotal:     "200",
		EpochStartBlock: 10,
	}
	k.SetFlow(ctx, &flow)

	// Verify it exists
	_, found := k.GetFlow(ctx, "channel-0")
	if !found {
		t.Fatal("expected flow to exist before deletion")
	}

	k.DeleteFlow(ctx, "channel-0")

	_, found = k.GetFlow(ctx, "channel-0")
	if found {
		t.Error("expected flow to be deleted")
	}
}

func TestIterateFlows(t *testing.T) {
	k, ctx := setupKeeper(t)

	flows := []types.ChannelFlow{
		{ChannelId: "channel-0", Denom: "ulgm", OutflowTotal: "100", InflowTotal: "0", EpochStartBlock: 1},
		{ChannelId: "channel-1", Denom: "ulgm", OutflowTotal: "200", InflowTotal: "0", EpochStartBlock: 1},
		{ChannelId: "channel-2", Denom: "ulgm", OutflowTotal: "300", InflowTotal: "0", EpochStartBlock: 1},
	}
	for i := range flows {
		k.SetFlow(ctx, &flows[i])
	}

	var count int
	k.IterateFlows(ctx, func(flow *types.ChannelFlow) bool {
		count++
		return false
	})
	if count != 3 {
		t.Errorf("expected 3 flows, got %d", count)
	}
}

// ============================================================
// Epoch Management Tests
// ============================================================

func TestGetOrResetFlow_NewFlow(t *testing.T) {
	k, ctx := setupKeeper(t)

	flow := k.GetOrResetFlow(ctx, "channel-new")
	if flow.ChannelId != "channel-new" {
		t.Errorf("expected channel-new, got %s", flow.ChannelId)
	}
	if flow.OutflowTotal != "0" {
		t.Errorf("expected OutflowTotal=0, got %s", flow.OutflowTotal)
	}
	if flow.InflowTotal != "0" {
		t.Errorf("expected InflowTotal=0, got %s", flow.InflowTotal)
	}
	if flow.EpochStartBlock != 100 { // ctx.BlockHeight()
		t.Errorf("expected EpochStartBlock=100, got %d", flow.EpochStartBlock)
	}
}

func TestGetOrResetFlow_WithinEpoch(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set up a flow that started at block 50 with some volume
	flow := types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "5000",
		InflowTotal:     "3000",
		EpochStartBlock: 50,
	}
	k.SetFlow(ctx, &flow)

	// Context is at block 100, epoch length is 34272, so epoch hasn't expired
	got := k.GetOrResetFlow(ctx, "channel-0")
	if got.OutflowTotal != "5000" {
		t.Errorf("expected OutflowTotal to remain 5000, got %s", got.OutflowTotal)
	}
	if got.InflowTotal != "3000" {
		t.Errorf("expected InflowTotal to remain 3000, got %s", got.InflowTotal)
	}
}

func TestGetOrResetFlow_EpochExpired(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set params with short epoch
	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	})

	// Set a flow starting at block 50
	flow := types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "5000",
		InflowTotal:     "3000",
		EpochStartBlock: 50,
	}
	k.SetFlow(ctx, &flow)

	// Context at block 100, epoch length 10, so 50+10=60 < 100 → epoch expired
	got := k.GetOrResetFlow(ctx, "channel-0")
	if got.OutflowTotal != "0" {
		t.Errorf("expected OutflowTotal reset to 0, got %s", got.OutflowTotal)
	}
	if got.InflowTotal != "0" {
		t.Errorf("expected InflowTotal reset to 0, got %s", got.InflowTotal)
	}
	if got.EpochStartBlock != 100 {
		t.Errorf("expected EpochStartBlock reset to 100, got %d", got.EpochStartBlock)
	}
}

func TestResetExpiredFlows(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Short epoch for testing
	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	})

	// One expired flow, one not expired
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId: "channel-expired", Denom: "ulgm",
		OutflowTotal: "1000", InflowTotal: "2000", EpochStartBlock: 50,
	})
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId: "channel-active", Denom: "ulgm",
		OutflowTotal: "500", InflowTotal: "600", EpochStartBlock: 95,
	})

	k.ResetExpiredFlows(ctx)

	// Expired flow should be reset
	expired, _ := k.GetFlow(ctx, "channel-expired")
	if expired.OutflowTotal != "0" {
		t.Errorf("expired channel: want OutflowTotal=0, got %s", expired.OutflowTotal)
	}

	// Active flow should remain unchanged
	active, _ := k.GetFlow(ctx, "channel-active")
	if active.OutflowTotal != "500" {
		t.Errorf("active channel: want OutflowTotal=500, got %s", active.OutflowTotal)
	}
}

// ============================================================
// Genesis Tests
// ============================================================

func TestInitExportGenesis(t *testing.T) {
	k, ctx := setupKeeper(t)

	params := types.Params{
		EpochLengthBlocks:  5000,
		MaxOutflowPerEpoch: "999",
		MaxInflowPerEpoch:  "888",
		TrackedDenom:       "utest",
	}
	flow := types.ChannelFlow{
		ChannelId:       "channel-42",
		Denom:           "utest",
		OutflowTotal:    "100",
		InflowTotal:     "200",
		EpochStartBlock: 10,
	}
	gs := &types.GenesisState{
		Params: &params,
		Flows:  []*types.ChannelFlow{&flow},
	}

	k.InitGenesis(ctx, gs)

	exported := k.ExportGenesis(ctx)
	if exported.Params.EpochLengthBlocks != 5000 {
		t.Errorf("expected EpochLengthBlocks=5000, got %d", exported.Params.EpochLengthBlocks)
	}
	if len(exported.Flows) != 1 {
		t.Fatalf("expected 1 flow, got %d", len(exported.Flows))
	}
	if exported.Flows[0].ChannelId != "channel-42" {
		t.Errorf("expected channel-42, got %s", exported.Flows[0].ChannelId)
	}
}

func TestDefaultGenesisState(t *testing.T) {
	gs := types.DefaultGenesisState()
	if err := gs.Validate(); err != nil {
		t.Errorf("default genesis should be valid: %v", err)
	}
}

// ============================================================
// MsgServer Tests
// ============================================================

func TestMsgUpdateParams_Authority(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Valid authority
	msg := &types.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params: &types.Params{
			EpochLengthBlocks:  100,
			MaxOutflowPerEpoch: "999",
			MaxInflowPerEpoch:  "888",
			TrackedDenom:       "ulgm",
		},
	}
	_, err := ms.UpdateParams(ctx, msg)
	if err != nil {
		t.Errorf("expected no error with correct authority, got %v", err)
	}

	// Verify params were updated
	got := k.GetParams(ctx)
	if got.EpochLengthBlocks != 100 {
		t.Errorf("expected EpochLengthBlocks=100 after update, got %d", got.EpochLengthBlocks)
	}
}

func TestMsgUpdateParams_Unauthorized(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	dp := types.DefaultParams()
	msg := &types.MsgUpdateParams{
		Authority: "lgm1wrongauthority",
		Params:    &dp,
	}
	_, err := ms.UpdateParams(ctx, msg)
	if err == nil {
		t.Error("expected error with wrong authority")
	}
}

func TestMsgUpdateParams_InvalidParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	msg := &types.MsgUpdateParams{
		Authority: k.GetAuthority(),
		Params: &types.Params{
			EpochLengthBlocks: 0, // invalid
			TrackedDenom:      "ulgm",
		},
	}
	_, err := ms.UpdateParams(ctx, msg)
	if err == nil {
		t.Error("expected error with invalid params (zero epoch length)")
	}
}

// ============================================================
// Query Server Tests
// ============================================================

func TestQueryParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.TrackedDenom != "ulgm" {
		t.Errorf("expected TrackedDenom=ulgm, got %s", resp.Params.TrackedDenom)
	}
}

func TestQueryChannelFlow_Found(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "42",
		InflowTotal:     "99",
		EpochStartBlock: 1,
	})

	resp, err := qs.ChannelFlow(ctx, &types.QueryChannelFlowRequest{ChannelId: "channel-0"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Found {
		t.Error("expected flow to be found")
	}
	if resp.Flow.OutflowTotal != "42" {
		t.Errorf("expected OutflowTotal=42, got %s", resp.Flow.OutflowTotal)
	}
}

func TestQueryChannelFlow_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.ChannelFlow(ctx, &types.QueryChannelFlowRequest{ChannelId: "channel-nonexistent"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Found {
		t.Error("expected flow not to be found")
	}
}

func TestQueryChannelFlow_EmptyChannelID(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	_, err := qs.ChannelFlow(ctx, &types.QueryChannelFlowRequest{ChannelId: ""})
	if err == nil {
		t.Error("expected error with empty channel_id")
	}
}

// ============================================================
// P0-3: Sliding Window / Effective Flow Tests
// ============================================================

func TestGetEffectiveOutflow_NoPreviousEpoch(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set flow with no previous epoch data
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "500",
		InflowTotal:     "0",
		EpochStartBlock: 100,
	})

	effective := k.GetEffectiveOutflow(ctx, "channel-0")
	if effective.String() != "500" {
		t.Errorf("expected effective outflow=500 (no previous), got %s", effective.String())
	}
}

func TestGetEffectiveOutflow_WithPreviousEpoch(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "10000",
		MaxInflowPerEpoch:  "10000",
		TrackedDenom:       "ulgm",
	})

	// Previous epoch had 1000 outflow
	k.SetPreviousFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "1000",
		InflowTotal:     "0",
		EpochStartBlock: 90,
	})

	// Current epoch started at block 100, we're at block 100 (ctx height)
	// So elapsed = 0, remaining = 10, carry = 1000 * 10/10 = 1000
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "200",
		InflowTotal:     "0",
		EpochStartBlock: 100,
	})

	effective := k.GetEffectiveOutflow(ctx, "channel-0")
	// effective = 200 (current) + 1000 * (10-0)/10 = 200 + 1000 = 1200
	if effective.String() != "1200" {
		t.Errorf("expected effective outflow=1200, got %s", effective.String())
	}
}

func TestGetEffectiveOutflow_HalfwayThroughEpoch(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "10000",
		MaxInflowPerEpoch:  "10000",
		TrackedDenom:       "ulgm",
	})

	// Previous epoch had 1000 outflow
	k.SetPreviousFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "1000",
		InflowTotal:     "0",
		EpochStartBlock: 85,
	})

	// Current epoch started at block 95, we're at block 100
	// elapsed = 5, remaining = 5, carry = 1000 * 5/10 = 500
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "300",
		InflowTotal:     "0",
		EpochStartBlock: 95,
	})

	effective := k.GetEffectiveOutflow(ctx, "channel-0")
	// effective = 300 + 500 = 800
	if effective.String() != "800" {
		t.Errorf("expected effective outflow=800, got %s", effective.String())
	}
}

func TestGetOrResetFlow_SavesPreviousEpoch(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "10000",
		MaxInflowPerEpoch:  "10000",
		TrackedDenom:       "ulgm",
	})

	// Set a flow that started at block 90 with accumulated values (exactly 1 epoch ago)
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "5000",
		InflowTotal:     "3000",
		EpochStartBlock: 90,
	})

	// Context at block 100, epoch length 10 → exactly 1 epoch expired
	flow := k.GetOrResetFlow(ctx, "channel-0")

	// Current flow should be reset
	if flow.OutflowTotal != "0" {
		t.Errorf("expected current outflow reset to 0, got %s", flow.OutflowTotal)
	}

	// Previous flow should be saved
	prevFlow, found := k.GetPreviousFlow(ctx, "channel-0")
	if !found {
		t.Fatal("expected previous flow to be saved")
	}
	if prevFlow.OutflowTotal != "5000" {
		t.Errorf("expected previous outflow=5000, got %s", prevFlow.OutflowTotal)
	}
	if prevFlow.InflowTotal != "3000" {
		t.Errorf("expected previous inflow=3000, got %s", prevFlow.InflowTotal)
	}
}

// ============================================================
// Sliding Window Adversarial Tests
// ============================================================

// TestSlidingWindow_BoundaryAttack verifies that filling to the limit before an
// epoch boundary doesn't give extra capacity through carry-over exploitation.
func TestSlidingWindow_BoundaryAttack(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	// Phase 1: Fill to 990 during first epoch (block 100, epoch starts at 100)
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "990",
		InflowTotal:     "0",
		EpochStartBlock: 100,
	})

	// At block 109 (1 block before boundary), still within first epoch
	ctx109 := ctx.WithBlockHeight(109)
	effective := k.GetEffectiveOutflow(ctx109, "channel-0")
	if effective.String() != "990" {
		t.Errorf("block 109 (within epoch): expected effective=990, got %s", effective)
	}

	// Push to exactly at limit
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "1000",
		InflowTotal:     "0",
		EpochStartBlock: 100,
	})

	// Phase 2: At block 110 (exact epoch boundary), trigger reset
	ctx110 := ctx.WithBlockHeight(110)
	flow := k.GetOrResetFlow(ctx110, "channel-0")
	if flow.OutflowTotal != "0" {
		t.Errorf("after reset at boundary: expected outflow=0, got %s", flow.OutflowTotal)
	}
	if flow.EpochStartBlock != 110 {
		t.Errorf("expected epoch start=110, got %d", flow.EpochStartBlock)
	}

	// Previous flow saved (adjacent epoch transition)
	prev, found := k.GetPreviousFlow(ctx110, "channel-0")
	if !found {
		t.Fatal("expected previous flow to be saved at adjacent epoch transition")
	}
	if prev.OutflowTotal != "1000" {
		t.Errorf("expected previous outflow=1000, got %s", prev.OutflowTotal)
	}

	// At boundary (elapsed=0): carry = 1000 * (10-0)/10 = 1000
	// Effective = 0 + 1000 = 1000. No room for additional outflow.
	effective = k.GetEffectiveOutflow(ctx110, "channel-0")
	if effective.String() != "1000" {
		t.Errorf("boundary (elapsed=0): expected effective=1000, got %s", effective)
	}

	// Phase 3: At block 111 (elapsed=1): carry = 1000*9/10 = 900
	ctx111 := ctx.WithBlockHeight(111)
	effective = k.GetEffectiveOutflow(ctx111, "channel-0")
	if effective.String() != "900" {
		t.Errorf("block 111 (elapsed=1): expected effective=900, got %s", effective)
	}

	// Attacker can only send 100 more (not 1000 that a naive reset would allow)
	// This proves the sliding window prevents boundary-burst attacks
}

// TestSlidingWindow_RapidEpochTransitions sends traffic across 3 epoch boundaries
// and verifies flow accounting stays consistent at each transition.
func TestSlidingWindow_RapidEpochTransitions(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  5,
		MaxOutflowPerEpoch: "10000",
		MaxInflowPerEpoch:  "10000",
		TrackedDenom:       "ulgm",
	})

	// Epoch 1 (blocks 100-104): 800 outflow
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "800",
		InflowTotal:     "0",
		EpochStartBlock: 100,
	})

	// Transition to epoch 2 at block 105
	ctx105 := ctx.WithBlockHeight(105)
	flow := k.GetOrResetFlow(ctx105, "channel-0")
	if flow.OutflowTotal != "0" {
		t.Fatalf("epoch 2 start: expected outflow=0, got %s", flow.OutflowTotal)
	}

	// At block 105 (elapsed=0): carry = 800*5/5 = 800, effective = 0+800 = 800
	eff := k.GetEffectiveOutflow(ctx105, "channel-0")
	if eff.String() != "800" {
		t.Errorf("block 105: expected effective=800, got %s", eff)
	}

	// Add 300 outflow during epoch 2
	flow.OutflowTotal = "300"
	k.SetFlow(ctx105, flow)

	// Mid-epoch check at block 107 (elapsed=2, remaining=3)
	// carry = 800*3/5 = 480, effective = 300+480 = 780
	ctx107 := ctx.WithBlockHeight(107)
	eff = k.GetEffectiveOutflow(ctx107, "channel-0")
	if eff.String() != "780" {
		t.Errorf("block 107: expected effective=780, got %s", eff)
	}

	// Set final epoch 2 outflow to 600
	flow.OutflowTotal = "600"
	k.SetFlow(ctx107, flow)

	// Transition to epoch 3 at block 110
	ctx110 := ctx.WithBlockHeight(110)
	flow = k.GetOrResetFlow(ctx110, "channel-0")
	if flow.OutflowTotal != "0" {
		t.Fatalf("epoch 3 start: expected outflow=0, got %s", flow.OutflowTotal)
	}

	// Previous should be epoch 2's flow (600)
	prev, found := k.GetPreviousFlow(ctx110, "channel-0")
	if !found {
		t.Fatal("expected previous flow for epoch 3")
	}
	if prev.OutflowTotal != "600" {
		t.Errorf("epoch 3 previous: expected 600, got %s", prev.OutflowTotal)
	}

	// At block 110 (elapsed=0): carry = 600*5/5 = 600, effective = 0+600 = 600
	eff = k.GetEffectiveOutflow(ctx110, "channel-0")
	if eff.String() != "600" {
		t.Errorf("block 110: expected effective=600, got %s", eff)
	}

	// Transition to epoch 4 at block 115 (epoch 3 had no new outflow)
	ctx115 := ctx.WithBlockHeight(115)
	flow = k.GetOrResetFlow(ctx115, "channel-0")
	if flow.OutflowTotal != "0" {
		t.Fatalf("epoch 4 start: expected outflow=0, got %s", flow.OutflowTotal)
	}

	// Previous is epoch 3 (0 outflow), so carry = 0
	eff = k.GetEffectiveOutflow(ctx115, "channel-0")
	if eff.String() != "0" {
		t.Errorf("block 115: expected effective=0 (no traffic in epoch 3), got %s", eff)
	}
}

// TestSlidingWindow_QuietPeriodDecay verifies that after 2+ full epochs of
// silence, previous flow is fully discarded (zero carry-over).
func TestSlidingWindow_QuietPeriodDecay(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	// Fill 900 during epoch 1 (block 100)
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "900",
		InflowTotal:     "0",
		EpochStartBlock: 100,
	})

	// Skip 2 full epochs → block 121 (epochsPassed = (121-100)/10 = 2)
	ctx121 := ctx.WithBlockHeight(121)
	flow := k.GetOrResetFlow(ctx121, "channel-0")
	if flow.OutflowTotal != "0" {
		t.Errorf("expected outflow=0 after quiet period, got %s", flow.OutflowTotal)
	}

	// Previous flow should be DELETED (multi-epoch skip)
	_, found := k.GetPreviousFlow(ctx121, "channel-0")
	if found {
		t.Error("expected previous flow deleted after 2+ epoch skip")
	}

	// No carry-over → effective = 0
	effective := k.GetEffectiveOutflow(ctx121, "channel-0")
	if effective.String() != "0" {
		t.Errorf("expected effective=0 after quiet period, got %s", effective)
	}

	// Burst: can send up to full limit
	flow.OutflowTotal = "999"
	k.SetFlow(ctx121, flow)
	effective = k.GetEffectiveOutflow(ctx121, "channel-0")
	if effective.String() != "999" {
		t.Errorf("burst after quiet: expected effective=999, got %s", effective)
	}
}

// TestSlidingWindow_ExactEpochBoundary verifies correct behavior when a transfer
// arrives at the exact epoch boundary block (no double-counting).
func TestSlidingWindow_ExactEpochBoundary(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	// Fill 500 in epoch 1
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "500",
		InflowTotal:     "0",
		EpochStartBlock: 100,
	})

	// At exact boundary block 110
	ctx110 := ctx.WithBlockHeight(110)
	flow := k.GetOrResetFlow(ctx110, "channel-0")
	if flow.OutflowTotal != "0" {
		t.Errorf("expected reset outflow=0, got %s", flow.OutflowTotal)
	}
	if flow.EpochStartBlock != 110 {
		t.Errorf("expected epoch start=110, got %d", flow.EpochStartBlock)
	}

	// At boundary (elapsed=0): carry = 500*(10/10) = 500
	effective := k.GetEffectiveOutflow(ctx110, "channel-0")
	if effective.String() != "500" {
		t.Errorf("at boundary: expected effective=500, got %s", effective)
	}

	// Add 400 at boundary → effective = 400+500 = 900 < 1000
	flow.OutflowTotal = "400"
	k.SetFlow(ctx110, flow)
	effective = k.GetEffectiveOutflow(ctx110, "channel-0")
	if effective.String() != "900" {
		t.Errorf("at boundary with 400 current: expected effective=900, got %s", effective)
	}

	// The 500 from previous is counted once (as carry), 400 from current once
	// No double-counting: total = 400 + 500 = 900
}

// TestSlidingWindow_PartialEpochDecay_Precision verifies carry-over fraction is
// correct at 1/3, 1/2, 2/3, and 99% through an epoch (within exact integer math).
func TestSlidingWindow_PartialEpochDecay_Precision(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  100,
		MaxOutflowPerEpoch: "100000",
		MaxInflowPerEpoch:  "100000",
		TrackedDenom:       "ulgm",
	})

	tests := []struct {
		name        string
		elapsed     uint64
		expectCarry string
	}{
		{"1/3 through (elapsed=33)", 33, "6700"},  // 10000*(100-33)/100 = 6700
		{"1/2 through (elapsed=50)", 50, "5000"},  // 10000*50/100 = 5000
		{"2/3 through (elapsed=67)", 67, "3300"},  // 10000*33/100 = 3300
		{"99% through (elapsed=99)", 99, "100"},   // 10000*1/100 = 100
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			epochStart := uint64(200)
			blockHeight := epochStart + tc.elapsed

			// Set up current flow (empty — no traffic this epoch)
			k.SetFlow(ctx, &types.ChannelFlow{
				ChannelId:       "channel-test",
				Denom:           "ulgm",
				OutflowTotal:    "0",
				InflowTotal:     "0",
				EpochStartBlock: epochStart,
			})

			// Set up previous flow with 10000 outflow
			k.SetPreviousFlow(ctx, &types.ChannelFlow{
				ChannelId:       "channel-test",
				Denom:           "ulgm",
				OutflowTotal:    "10000",
				InflowTotal:     "0",
				EpochStartBlock: epochStart - 100,
			})

			ctxB := ctx.WithBlockHeight(int64(blockHeight))
			effective := k.GetEffectiveOutflow(ctxB, "channel-test")
			if effective.String() != tc.expectCarry {
				t.Errorf("at elapsed=%d: expected carry=%s, got %s",
					tc.elapsed, tc.expectCarry, effective)
			}
		})
	}
}

// ============================================================
// Epoch Reset Edge Cases
// ============================================================

// TestEpochReset_MultiEpochSkip_NoPreviousFlow verifies that skipping 3+ epochs
// deletes the previous flow record entirely (no stale carry-over).
func TestEpochReset_MultiEpochSkip_NoPreviousFlow(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	// Set up flow with history
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "500",
		InflowTotal:     "300",
		EpochStartBlock: 100,
	})

	// Also set a stale previous flow
	k.SetPreviousFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "800",
		InflowTotal:     "0",
		EpochStartBlock: 90,
	})

	// Skip 3 epochs: block 131. epochsPassed = (131-100)/10 = 3
	ctx131 := ctx.WithBlockHeight(131)
	flow := k.GetOrResetFlow(ctx131, "channel-0")

	if flow.OutflowTotal != "0" {
		t.Errorf("expected outflow=0, got %s", flow.OutflowTotal)
	}
	if flow.InflowTotal != "0" {
		t.Errorf("expected inflow=0, got %s", flow.InflowTotal)
	}

	// Previous should be DELETED (epochsPassed > 1)
	_, found := k.GetPreviousFlow(ctx131, "channel-0")
	if found {
		t.Error("expected previous flow deleted after 3-epoch skip")
	}

	// Effective = 0 (no carry-over)
	effective := k.GetEffectiveOutflow(ctx131, "channel-0")
	if effective.String() != "0" {
		t.Errorf("expected effective=0, got %s", effective)
	}

	// Epoch start aligned: 100 + 3*10 = 130
	if flow.EpochStartBlock != 130 {
		t.Errorf("expected epoch start=130, got %d", flow.EpochStartBlock)
	}
}

// TestEpochReset_AlignedBoundaries verifies epoch start aligns to deterministic
// boundaries (multiples of epoch length from the original start), not lazy reset time.
func TestEpochReset_AlignedBoundaries(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	// Epoch starts at block 100, expires at 110
	k.SetFlow(ctx, &types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "400",
		InflowTotal:     "0",
		EpochStartBlock: 100,
	})

	// Access at block 113 (3 blocks into epoch 2)
	// epochsPassed = (113-100)/10 = 1
	// New epoch should start at 100+1*10 = 110, NOT 113 (lazy alignment)
	ctx113 := ctx.WithBlockHeight(113)
	flow := k.GetOrResetFlow(ctx113, "channel-0")

	if flow.EpochStartBlock != 110 {
		t.Errorf("epoch start should be aligned to 110 (not 113 lazy reset); got %d",
			flow.EpochStartBlock)
	}

	// Set some outflow
	flow.OutflowTotal = "200"
	k.SetFlow(ctx113, flow)

	// Access at block 127 (7 blocks into epoch 3)
	// epochsPassed = (127-110)/10 = 1
	// New epoch should start at 110+1*10 = 120
	ctx127 := ctx.WithBlockHeight(127)
	flow = k.GetOrResetFlow(ctx127, "channel-0")

	if flow.EpochStartBlock != 120 {
		t.Errorf("epoch start should be aligned to 120; got %d", flow.EpochStartBlock)
	}
}

// TestEpochReset_FirstBlockAfterGenesis verifies the first-ever transfer has
// no previous flow and no carry-over.
func TestEpochReset_FirstBlockAfterGenesis(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetParams(ctx, &types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "1000",
		MaxInflowPerEpoch:  "1000",
		TrackedDenom:       "ulgm",
	})

	// First ever transfer at block 1 — no flow record exists
	ctx1 := ctx.WithBlockHeight(1)
	flow := k.GetOrResetFlow(ctx1, "channel-0")

	if flow.OutflowTotal != "0" {
		t.Errorf("expected zero outflow at genesis, got %s", flow.OutflowTotal)
	}
	if flow.InflowTotal != "0" {
		t.Errorf("expected zero inflow at genesis, got %s", flow.InflowTotal)
	}
	if flow.EpochStartBlock != 1 {
		t.Errorf("expected epoch start=1, got %d", flow.EpochStartBlock)
	}

	// No previous flow should exist
	_, found := k.GetPreviousFlow(ctx1, "channel-0")
	if found {
		t.Error("expected no previous flow at genesis")
	}

	// Effective should be zero
	effective := k.GetEffectiveOutflow(ctx1, "channel-0")
	if effective.String() != "0" {
		t.Errorf("expected effective=0 at genesis, got %s", effective)
	}
}
