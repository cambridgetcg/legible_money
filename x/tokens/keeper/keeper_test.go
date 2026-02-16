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

	"github.com/legible-money/legible/x/tokens/keeper"
	"github.com/legible-money/legible/x/tokens/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
}

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatal(err)
	}

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	storeService := runtime.NewKVStoreService(storeKey)

	k := keeper.NewKeeper(cdc, storeService, nil, "lgm1authority")
	return k, ctx
}

func TestKeeperCreation(t *testing.T) {
	k, _ := setupKeeper(t)
	if k.GetAuthority() != "lgm1authority" {
		t.Fatalf("expected authority lgm1authority, got %s", k.GetAuthority())
	}
}

func TestParamsGetSet(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Default params
	params := k.GetParams(ctx)
	if params.PeriodBlocks != 0 {
		t.Fatalf("expected zeroed PeriodBlocks, got %d", params.PeriodBlocks)
	}

	// Set and retrieve
	params.PeriodBlocks = 12345
	k.SetParams(ctx, params)
	got := k.GetParams(ctx)
	if got.PeriodBlocks != 12345 {
		t.Fatalf("expected PeriodBlocks 12345, got %d", got.PeriodBlocks)
	}
}

func TestInitExportGenesis(t *testing.T) {
	k, ctx := setupKeeper(t)

	genState := types.DefaultGenesis()
	k.InitGenesis(ctx, *genState)

	exported := k.ExportGenesis(ctx)
	if exported.Params == nil {
		t.Fatal("exported params is nil")
	}
	if len(exported.Accounts) != 0 {
		t.Fatalf("expected 0 accounts, got %d", len(exported.Accounts))
	}
}

func TestMsgServerImpl(t *testing.T) {
	k, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	if srv == nil {
		t.Fatal("NewMsgServerImpl returned nil")
	}
}

func TestQueryServerImpl(t *testing.T) {
	k, ctx := setupKeeper(t)
	srv := keeper.NewQueryServerImpl(k)
	resp, err := srv.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("Params query failed: %v", err)
	}
	if resp == nil {
		t.Fatal("Params response is nil")
	}
}
