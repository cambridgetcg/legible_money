package keeper_test

import (
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/home/keeper"
	"github.com/legible-money/legible/x/home/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
	t.Helper()
	initAddresses()

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

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), nil, nil, "lgm1authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Set default params
	k.SetParams(ctx, types.DefaultParams())

	return k, ctx
}

func testAddr(name string) string {
	return sdk.AccAddress([]byte(name + "_address_padding")).String()
}

var (
	owner1 string
	owner2 string
	user1  string
)

func initAddresses() {
	owner1 = testAddr("owner001")
	owner2 = testAddr("owner002")
	user1 = testAddr("user0001")
}

// ==================== CreateHome ====================

func TestCreateHome(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Agent Alpha Home",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.HomeId == "" {
		t.Error("expected non-empty home ID")
	}

	// Verify home was stored
	home, found := k.GetHome(ctx, resp.HomeId)
	if !found {
		t.Fatal("home not found after creation")
	}
	if home.Name != "Agent Alpha Home" {
		t.Errorf("expected name 'Agent Alpha Home', got '%s'", home.Name)
	}
	if home.Status != string(types.HomeStatusActive) {
		t.Errorf("expected active status, got %s", home.Status)
	}
	if home.OwnerAddress != owner1 {
		t.Errorf("expected owner %s, got %s", owner1, home.OwnerAddress)
	}
	if home.ComfortScore != 50 {
		t.Errorf("expected default comfort score 50, got %d", home.ComfortScore)
	}
	if home.Guardian.DefenseStrategy != string(types.DefenseModerate) {
		t.Errorf("expected default defense strategy moderate, got %s", home.Guardian.DefenseStrategy)
	}

	// Verify owner index
	ids := k.GetHomesByOwner(ctx, owner1)
	if len(ids) != 1 || ids[0] != resp.HomeId {
		t.Errorf("expected owner index to contain home ID")
	}
}

func TestCreateHomeDuplicate(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Set max homes to 2
	params := types.DefaultParams()
	params.MaxHomesPerAccount = 2
	k.SetParams(ctx, params)

	// Create 2 homes
	for i := 0; i < 2; i++ {
		_, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
			Owner: owner1,
			Name:  "Home",
		})
		if err != nil {
			t.Fatalf("unexpected error on home %d: %v", i+1, err)
		}
	}

	// Third should fail
	_, err := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Home 3",
	})
	if err == nil {
		t.Error("expected error for max homes exceeded")
	}
}

// ==================== UpdateHome ====================

func TestUpdateHome(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Original",
	})

	_, err := msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner:  owner1,
		HomeId: resp.HomeId,
		Name:   "Updated Name",
	})
	if err != nil {
		t.Fatalf("update failed: %v", err)
	}

	home, _ := k.GetHome(ctx, resp.HomeId)
	if home.Name != "Updated Name" {
		t.Errorf("expected 'Updated Name', got '%s'", home.Name)
	}
	if home.LastActiveBlock != uint64(ctx.BlockHeight()) {
		t.Errorf("expected LastActiveBlock to be bumped")
	}
}

func TestUpdateHomeStatusTransitions(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Status Test",
	})

	// active -> dormant (valid)
	_, err := msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "dormant",
	})
	if err != nil {
		t.Fatalf("active->dormant should be valid: %v", err)
	}

	// dormant -> active (valid)
	_, err = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "active",
	})
	if err != nil {
		t.Fatalf("dormant->active should be valid: %v", err)
	}

	// active -> archived (valid, terminal)
	_, err = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "archived",
	})
	if err != nil {
		t.Fatalf("active->archived should be valid: %v", err)
	}

	// archived -> active (invalid, terminal)
	_, err = msgServer.UpdateHome(ctx, &types.MsgUpdateHome{
		Owner: owner1, HomeId: resp.HomeId, Status: "active",
	})
	if err == nil {
		t.Error("archived->active should be invalid")
	}
}

// ==================== UpdateMemoryCID ====================

func TestUpdateMemoryCID(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	resp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Memory Home",
	})

	_, err := msgServer.UpdateMemoryCID(ctx, &types.MsgUpdateMemoryCID{
		Owner:  owner1,
		HomeId: resp.HomeId,
		CID:    "QmYwAPJzv5CZsnN625s3Xf2nemtYgPpHdWEz79ojWnPbdG",
	})
	if err != nil {
		t.Fatalf("update memory CID failed: %v", err)
	}

	home, _ := k.GetHome(ctx, resp.HomeId)
	if home.MemoryCid != "QmYwAPJzv5CZsnN625s3Xf2nemtYgPpHdWEz79ojWnPbdG" {
		t.Errorf("expected CID to be set, got '%s'", home.MemoryCid)
	}
}

// ==================== StartSession / EndSession ====================

func TestStartEndSession(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	// Create home
	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Session Home",
	})

	// Register key
	_, err := msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner:       owner1,
		HomeId:      homeResp.HomeId,
		KeyHash:     "abc123def456",
		KeyType:     "ed25519",
		Role:        string(types.AccessRoleOperator),
		Permissions: []string{string(types.PermReadMemory), string(types.PermWriteMemory), string(types.PermVerify)},
	})
	if err != nil {
		t.Fatalf("register key failed: %v", err)
	}

	// Start session with subset of permissions
	sesResp, err := msgServer.StartSession(ctx, &types.MsgStartSession{
		Signer:               user1,
		HomeId:               homeResp.HomeId,
		KeyHash:              "abc123def456",
		RequestedPermissions: []string{string(types.PermReadMemory), string(types.PermVerify)},
	})
	if err != nil {
		t.Fatalf("start session failed: %v", err)
	}
	if sesResp.SessionId == "" {
		t.Error("expected non-empty session ID")
	}

	// Verify session stored with intersection
	session, found := k.GetSession(ctx, homeResp.HomeId, sesResp.SessionId)
	if !found {
		t.Fatal("session not found")
	}
	if len(session.Permissions) != 2 {
		t.Errorf("expected 2 granted permissions, got %d", len(session.Permissions))
	}

	// End session
	_, err = msgServer.EndSession(ctx, &types.MsgEndSession{
		Signer:    user1,
		HomeId:    homeResp.HomeId,
		SessionId: sesResp.SessionId,
	})
	if err != nil {
		t.Fatalf("end session failed: %v", err)
	}

	_, found = k.GetSession(ctx, homeResp.HomeId, sesResp.SessionId)
	if found {
		t.Error("session should be deleted after EndSession")
	}
}

func TestSessionExpiry(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Expiry Home",
	})

	// Register key that expires at block 50 (current is 100)
	_, _ = msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner:       owner1,
		HomeId:      homeResp.HomeId,
		KeyHash:     "expiredkey123",
		KeyType:     "ed25519",
		Role:        string(types.AccessRoleGuest),
		Permissions: []string{string(types.PermReadMemory)},
		ExpiresAt:   50, // Already expired
	})

	// Try to start session with expired key
	_, err := msgServer.StartSession(ctx, &types.MsgStartSession{
		Signer:  user1,
		HomeId:  homeResp.HomeId,
		KeyHash: "expiredkey123",
	})
	if err == nil {
		t.Error("expected error for expired key")
	}
}

// ==================== RegisterKey ====================

func TestRegisterKey(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Key Home",
	})

	_, err := msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner:       owner1,
		HomeId:      homeResp.HomeId,
		KeyHash:     "mykeyhash001",
		KeyType:     "secp256k1",
		Role:        string(types.AccessRoleOperator),
		Permissions: []string{string(types.PermTransfer), string(types.PermStake)},
	})
	if err != nil {
		t.Fatalf("register key failed: %v", err)
	}

	reg, found := k.GetKeyRegistration(ctx, homeResp.HomeId, "mykeyhash001")
	if !found {
		t.Fatal("key registration not found")
	}
	if reg.Role != string(types.AccessRoleOperator) {
		t.Errorf("expected operator role, got %s", reg.Role)
	}
	if len(reg.Permissions) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(reg.Permissions))
	}
}

// ==================== RevokeKey ====================

func TestRevokeKey(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Revoke Home",
	})

	_, _ = msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner:       owner1,
		HomeId:      homeResp.HomeId,
		KeyHash:     "revokekey001",
		KeyType:     "ed25519",
		Role:        string(types.AccessRoleOperator),
		Permissions: []string{string(types.PermReadMemory)},
	})

	// Start a session with this key
	sesResp, _ := msgServer.StartSession(ctx, &types.MsgStartSession{
		Signer:  user1,
		HomeId:  homeResp.HomeId,
		KeyHash: "revokekey001",
	})

	// Revoke the key
	_, err := msgServer.RevokeKey(ctx, &types.MsgRevokeKey{
		Owner:   owner1,
		HomeId:  homeResp.HomeId,
		KeyHash: "revokekey001",
	})
	if err != nil {
		t.Fatalf("revoke key failed: %v", err)
	}

	// Key should be revoked
	reg, _ := k.GetKeyRegistration(ctx, homeResp.HomeId, "revokekey001")
	if !reg.Revoked {
		t.Error("expected key to be revoked")
	}

	// Session should be terminated
	_, found := k.GetSession(ctx, homeResp.HomeId, sesResp.SessionId)
	if found {
		t.Error("session should be terminated after key revocation")
	}
}

// ==================== MaxKeysEnforced ====================

func TestMaxKeysEnforced(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	params := types.DefaultParams()
	params.MaxKeysPerHome = 3
	k.SetParams(ctx, params)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Key Limit Home",
	})

	for i := 0; i < 3; i++ {
		_, err := msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
			Owner:       owner1,
			HomeId:      homeResp.HomeId,
			KeyHash:     string(rune('a'+i)) + "keyhash",
			KeyType:     "ed25519",
			Role:        string(types.AccessRoleGuest),
			Permissions: []string{string(types.PermReadMemory)},
		})
		if err != nil {
			t.Fatalf("register key %d failed: %v", i+1, err)
		}
	}

	// Fourth should fail
	_, err := msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner:       owner1,
		HomeId:      homeResp.HomeId,
		KeyHash:     "dkeyhash",
		KeyType:     "ed25519",
		Role:        string(types.AccessRoleGuest),
		Permissions: []string{string(types.PermReadMemory)},
	})
	if err == nil {
		t.Error("expected error for max keys exceeded")
	}
}

// ==================== ConfigureGuardian ====================

func TestConfigureGuardian(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Guardian Home",
	})

	aggressive := string(types.DefenseAggressive)
	autoDefend := true
	_, err := msgServer.ConfigureGuardian(ctx, &types.MsgConfigureGuardian{
		Owner:           owner1,
		HomeId:          homeResp.HomeId,
		DefenseStrategy: &aggressive,
		AutoDefend:      &autoDefend,
		Deadman: &types.DeadmanConfig{
			Enabled:             true,
			InactivityThreshold: 50000,
			Action:              string(types.DeadmanAlertGuardians),
		},
		RecoveryDIDs:      []string{"did:lgm:recovery1", "did:lgm:recovery2"},
		RecoveryThreshold: ptrUint32(2),
	})
	if err != nil {
		t.Fatalf("configure guardian failed: %v", err)
	}

	home, _ := k.GetHome(ctx, homeResp.HomeId)
	if home.Guardian.DefenseStrategy != string(types.DefenseAggressive) {
		t.Errorf("expected aggressive, got %s", home.Guardian.DefenseStrategy)
	}
	if !home.Guardian.AutoDefend {
		t.Error("expected auto_defend true")
	}
	if !home.Guardian.Deadman.Enabled {
		t.Error("expected deadman enabled")
	}
	if home.Guardian.Deadman.InactivityThreshold != 50000 {
		t.Errorf("expected threshold 50000, got %d", home.Guardian.Deadman.InactivityThreshold)
	}
	if len(home.Guardian.RecoveryDids) != 2 {
		t.Errorf("expected 2 recovery DIDs, got %d", len(home.Guardian.RecoveryDids))
	}
	if home.Guardian.RecoveryThreshold != 2 {
		t.Errorf("expected recovery threshold 2, got %d", home.Guardian.RecoveryThreshold)
	}

	// Test invalid deadman threshold (below minimum)
	_, err = msgServer.ConfigureGuardian(ctx, &types.MsgConfigureGuardian{
		Owner:  owner1,
		HomeId: homeResp.HomeId,
		Deadman: &types.DeadmanConfig{
			Enabled:             true,
			InactivityThreshold: 100, // Below DeadmanMinThreshold (34272)
			Action:              string(types.DeadmanAlertGuardians),
		},
	})
	if err == nil {
		t.Error("expected error for deadman threshold below minimum")
	}
}

// ==================== DeadmanSwitch ====================

func TestDeadmanSwitch(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create a home with deadman enabled, last active at block 50
	home := types.AgentHome{
		HomeId:       "home-deadman",
		OwnerAddress: owner1,
		Name:         "Deadman Test",
		Status:       string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{
			DefenseStrategy: string(types.DefenseModerate),
			Deadman: &types.DeadmanConfig{
				Enabled:             true,
				InactivityThreshold: 30, // 30 blocks
				Action:              string(types.DeadmanAlertGuardians),
			},
		},
		CreatedAtBlock:  10,
		LastActiveBlock: 50, // Last active at block 50, current is 100 (gap = 50 > 30)
	}
	k.SetHome(ctx, home)

	// Also create a home that should NOT trigger
	activeHome := types.AgentHome{
		HomeId:       "home-active",
		OwnerAddress: owner1,
		Name:         "Active Home",
		Status:       string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{
			Deadman: &types.DeadmanConfig{
				Enabled:             true,
				InactivityThreshold: 30,
				Action:              string(types.DeadmanAlertGuardians),
			},
		},
		CreatedAtBlock:  10,
		LastActiveBlock: 90, // Gap = 10 < 30
	}
	k.SetHome(ctx, activeHome)

	// Run deadman check
	k.CheckDeadmanSwitches(ctx)

	// Verify the inactive home was set to guarded
	h1, _ := k.GetHome(ctx, "home-deadman")
	if h1.Status != string(types.HomeStatusGuarded) {
		t.Errorf("expected guarded status, got %s", h1.Status)
	}

	// Verify the active home was NOT touched
	h2, _ := k.GetHome(ctx, "home-active")
	if h2.Status != string(types.HomeStatusActive) {
		t.Errorf("expected active status, got %s", h2.Status)
	}
}

// ==================== AcknowledgeAlert ====================

func TestAcknowledgeAlert(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Alert Home",
	})

	// Create an alert directly
	k.CreateAlert(ctx, homeResp.HomeId, types.AlertDefenseActivated, types.AlertPriorityHigh, "Defense triggered", "")

	// Find the alert
	var alertId string
	k.IterateAlerts(ctx, homeResp.HomeId, func(alert types.Alert) bool {
		alertId = alert.AlertId
		return true
	})
	if alertId == "" {
		t.Fatal("no alert found")
	}

	// Acknowledge it
	_, err := msgServer.AcknowledgeAlert(ctx, &types.MsgAcknowledgeAlert{
		Signer:  owner1,
		HomeId:  homeResp.HomeId,
		AlertId: alertId,
	})
	if err != nil {
		t.Fatalf("acknowledge alert failed: %v", err)
	}

	alert, _ := k.GetAlert(ctx, homeResp.HomeId, alertId)
	if !alert.Acknowledged {
		t.Error("expected alert to be acknowledged")
	}
}

// ==================== SetSpendingLimit ====================

func TestSetSpendingLimit(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1,
		Name:  "Spending Home",
	})

	_, err := msgServer.SetSpendingLimit(ctx, &types.MsgSetSpendingLimit{
		Owner:        owner1,
		HomeId:       homeResp.HomeId,
		KeyType:      "operator",
		MaxAmount:    "5000000", // 5 LGM
		PeriodBlocks: 34272,    // 1 day
	})
	if err != nil {
		t.Fatalf("set spending limit failed: %v", err)
	}

	limit, found := k.GetSpendingLimit(ctx, homeResp.HomeId, "operator")
	if !found {
		t.Fatal("spending limit not found")
	}
	if limit.MaxAmount != "5000000" {
		t.Errorf("expected max 5000000, got %s", limit.MaxAmount)
	}
	if limit.PeriodBlocks != 34272 {
		t.Errorf("expected period 34272, got %d", limit.PeriodBlocks)
	}
	if limit.SpentInPeriod != "0" {
		t.Errorf("expected spent 0, got %s", limit.SpentInPeriod)
	}
}

// ==================== Genesis Round-Trip ====================

func TestParamsGenesis(t *testing.T) {
	k, ctx := setupKeeper(t)

	gs := types.GenesisState{
		Params: &types.Params{
			MaxHomesPerAccount:   5,
			MinCreationStake:     "2000000",
			SessionMaxDuration:   68544,
			MaxKeysPerHome:       10,
			MaxAlertsPerHome:     50,
			MaxSessionsPerHome:   3,
			DeadmanMinThreshold:  68544,
			AlertRetentionBlocks: 685440,
		},
		Homes: []*types.AgentHome{
			{
				HomeId:          "genesis-home-1",
				OwnerAddress:    owner1,
				Name:            "Genesis Home",
				Status:          string(types.HomeStatusActive),
				CreatedAtBlock:  1,
				LastActiveBlock: 1,
				Treasury: &types.HomeTreasury{
					ReservedBalance: "0",
				},
				Guardian: &types.HomeGuardian{
					DefenseStrategy: string(types.DefenseModerate),
				},
			},
		},
	}

	k.InitGenesis(ctx, gs)

	// Verify params
	params := k.GetParams(ctx)
	if params.MaxHomesPerAccount != 5 {
		t.Errorf("expected 5, got %d", params.MaxHomesPerAccount)
	}
	if params.SessionMaxDuration != 68544 {
		t.Errorf("expected 68544, got %d", params.SessionMaxDuration)
	}

	// Verify home
	home, found := k.GetHome(ctx, "genesis-home-1")
	if !found {
		t.Fatal("expected genesis home")
	}
	if home.Name != "Genesis Home" {
		t.Errorf("expected 'Genesis Home', got '%s'", home.Name)
	}

	// Export and verify
	exported := k.ExportGenesis(ctx)
	if exported.Params.MaxHomesPerAccount != 5 {
		t.Error("exported params mismatch")
	}
	if len(exported.Homes) != 1 {
		t.Errorf("expected 1 home, got %d", len(exported.Homes))
	}
}

// ---------- Helpers ----------

func ptrUint32(v uint32) *uint32 { return &v }
