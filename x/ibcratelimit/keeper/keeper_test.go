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
	k.SetParams(ctx, custom)

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
	for _, tc := range tests {
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
	k.SetFlow(ctx, flow)

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
	k.SetFlow(ctx, flow)

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
	for _, f := range flows {
		k.SetFlow(ctx, f)
	}

	var count int
	k.IterateFlows(ctx, func(flow types.ChannelFlow) bool {
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
	k.SetFlow(ctx, flow)

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
	k.SetParams(ctx, types.Params{
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
	k.SetFlow(ctx, flow)

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
	k.SetParams(ctx, types.Params{
		EpochLengthBlocks:  10,
		MaxOutflowPerEpoch: "10000000000",
		MaxInflowPerEpoch:  "10000000000",
		TrackedDenom:       "ulgm",
	})

	// One expired flow, one not expired
	k.SetFlow(ctx, types.ChannelFlow{
		ChannelId: "channel-expired", Denom: "ulgm",
		OutflowTotal: "1000", InflowTotal: "2000", EpochStartBlock: 50,
	})
	k.SetFlow(ctx, types.ChannelFlow{
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
		Params: types.Params{
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

	msg := &types.MsgUpdateParams{
		Authority: "lgm1wrongauthority",
		Params:    types.DefaultParams(),
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
		Params: types.Params{
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

	k.SetFlow(ctx, types.ChannelFlow{
		ChannelId:       "channel-0",
		Denom:           "ulgm",
		OutflowTotal:    "42",
		InflowTotal:     "99",
		EpochStartBlock: 1,
	})

	resp, err := qs.ChannelFlow(ctx, &types.QueryChannelFlowRequest{ChannelID: "channel-0"})
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

	resp, err := qs.ChannelFlow(ctx, &types.QueryChannelFlowRequest{ChannelID: "channel-nonexistent"})
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

	_, err := qs.ChannelFlow(ctx, &types.QueryChannelFlowRequest{ChannelID: ""})
	if err == nil {
		t.Error("expected error with empty channel_id")
	}
}
