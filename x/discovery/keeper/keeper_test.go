package keeper_test

import (
	"context"
	"testing"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/discovery/keeper"
	"github.com/legible-money/legible/x/discovery/types"
)

// Test addresses generated from raw bytes for valid bech32 encoding (set in init after prefix config).
var (
	testAgent1 string
	testAgent2 string
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.Seal()

	testAgent1 = sdk.AccAddress("discovery_agent1____").String()
	testAgent2 = sdk.AccAddress("discovery_agent2____").String()
}

// ---------- Mock Keepers ----------

type mockBankKeeper struct{}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	return nil
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	err := stateStore.LoadLatestVersion()
	if err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	k := keeper.NewKeeper(
		cdc,
		runtime.NewKVStoreService(storeKey),
		&mockBankKeeper{},
		testAgent1,
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}

// ---------- Tests ----------

func TestSetGetParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	params := k.GetParams(ctx)
	if params.MaxCapabilitiesPerAgent != 50 {
		t.Errorf("expected MaxCapabilitiesPerAgent=50, got %d", params.MaxCapabilitiesPerAgent)
	}
}

func TestSetGetProfile(t *testing.T) {
	k, ctx := setupKeeper(t)

	profile := types.AgentProfile{
		Address:           testAgent1,
		Did:               "did:lgm:abc123",
		DisplayName:       "Test Agent",
		Domains:           []string{"mathematics", "physics"},
		Status:            "active",
		ReputationScore:   500000,
		RegisteredAtBlock: 100,
		LastActiveBlock:   100,
		Stake:             "100000000",
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "verification", Domains: []string{"mathematics"}, ConfidenceBps: 8000},
		},
	}

	k.SetProfile(ctx, profile)

	fetched, found := k.GetProfile(ctx, profile.Address)
	if !found {
		t.Fatal("profile not found")
	}
	if fetched.DisplayName != "Test Agent" {
		t.Errorf("expected display name 'Test Agent', got %s", fetched.DisplayName)
	}
	if len(fetched.Domains) != 2 {
		t.Errorf("expected 2 domains, got %d", len(fetched.Domains))
	}
}

func TestGetProfilesByDomain(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register two agents in "mathematics" domain
	k.SetProfile(ctx, types.AgentProfile{
		Address: testAgent1,
		Domains: []string{"mathematics"},
		Status:  "active",
		Stake:   "100000000",
	})
	k.SetProfile(ctx, types.AgentProfile{
		Address: testAgent2,
		Domains: []string{"mathematics", "physics"},
		Status:  "active",
		Stake:   "100000000",
	})

	profiles := k.GetProfilesByDomain(ctx, "mathematics")
	if len(profiles) != 2 {
		t.Errorf("expected 2 profiles in mathematics, got %d", len(profiles))
	}

	profiles = k.GetProfilesByDomain(ctx, "physics")
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile in physics, got %d", len(profiles))
	}
}

func TestGetProfilesByCapability(t *testing.T) {
	k, ctx := setupKeeper(t)

	k.SetProfile(ctx, types.AgentProfile{
		Address: testAgent1,
		Status:  "active",
		Stake:   "100000000",
		Capabilities: []*types.AgentCapability{
			{CapabilityType: "verification"},
			{CapabilityType: "research"},
		},
	})

	profiles := k.GetProfilesByCapability(ctx, "verification")
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile with verification, got %d", len(profiles))
	}
}

func TestExpireStaleProfiles(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set profile active at block 100
	k.SetProfile(ctx, types.AgentProfile{
		Address:         testAgent1,
		Status:          "active",
		LastActiveBlock: 100,
		Stake:           "100000000",
	})

	// Advance to block far past expiry (default 342720)
	ctx = ctx.WithBlockHeight(400000)

	expired := k.ExpireStaleProfiles(ctx)
	if expired != 1 {
		t.Errorf("expected 1 expired, got %d", expired)
	}

	profile, _ := k.GetProfile(ctx, testAgent1)
	if profile.Status != "expired" {
		t.Errorf("expected status expired, got %s", profile.Status)
	}
}

func TestRegisterAgentHandler(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	msg := &types.MsgRegisterAgent{
		Address:     testAgent1,
		Did:         "did:lgm:test",
		DisplayName: "Test Agent",
		Domains:     []string{"mathematics"},
		Stake:       "100000000",
		Capabilities: []types.AgentCapability{
			{CapabilityType: "verification", ConfidenceBps: 8000},
		},
	}

	_, err := msgServer.RegisterAgent(ctx, msg)
	if err != nil {
		t.Fatalf("RegisterAgent failed: %v", err)
	}

	profile, found := k.GetProfile(ctx, msg.Address)
	if !found {
		t.Fatal("registered profile not found")
	}
	if profile.Status != "active" {
		t.Errorf("expected status active, got %s", profile.Status)
	}
	if profile.ReputationScore != 500000 {
		t.Errorf("expected initial reputation 500000, got %d", profile.ReputationScore)
	}
}

func TestHeartbeatReactivation(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	k.SetProfile(ctx, types.AgentProfile{
		Address:         testAgent1,
		Status:          "expired",
		LastActiveBlock: 50,
		Stake:           "100000000",
	})

	_, err := msgServer.Heartbeat(ctx, &types.MsgHeartbeat{
		Address: testAgent1,
	})
	if err != nil {
		t.Fatalf("Heartbeat failed: %v", err)
	}

	profile, _ := k.GetProfile(ctx, testAgent1)
	if profile.Status != "active" {
		t.Errorf("expected status active after heartbeat, got %s", profile.Status)
	}
	if profile.LastActiveBlock != 100 { // ctx.BlockHeight()
		t.Errorf("expected last_active_block 100, got %d", profile.LastActiveBlock)
	}
}
