package keeper_test

import (
	"crypto/rand"
	"fmt"
	"strings"
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

	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), nil, "lgm1authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Set default params
	dp := types.DefaultParams()
	k.SetParams(ctx, &dp)

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
	k.SetParams(ctx, &params)

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
		Cid:    "QmYwAPJzv5CZsnN625s3Xf2nemtYgPpHdWEz79ojWnPbdG",
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
	k.SetParams(ctx, &params)

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

	_, err := msgServer.ConfigureGuardian(ctx, &types.MsgConfigureGuardian{
		Owner:           owner1,
		HomeId:          homeResp.HomeId,
		DefenseStrategy: string(types.DefenseAggressive),
		AutoDefend:      true,
		Deadman: &types.DeadmanConfig{
			Enabled:             true,
			InactivityThreshold: 50000,
			Action:              string(types.DeadmanAlertGuardians),
		},
		RecoveryDids:      []string{"did:lgm:recovery1", "did:lgm:recovery2"},
		RecoveryThreshold: 2,
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
	k.SetHome(ctx, &home)

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
	k.SetHome(ctx, &activeHome)

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
	k.IterateAlerts(ctx, homeResp.HomeId, func(alert *types.Alert) bool {
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
		PeriodBlocks: 34272,     // 1 day
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

	k.InitGenesis(ctx, &gs)

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

// ==================== Spending Limit Enforcement ====================

func TestCheckAndDebitSpendingLimit_WithinLimit(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1, Name: "SpendLimit Home",
	})

	// Set a 10 LGM limit per day
	_, _ = msgServer.SetSpendingLimit(ctx, &types.MsgSetSpendingLimit{
		Owner: owner1, HomeId: homeResp.HomeId,
		KeyType: "operator", MaxAmount: "10000000", PeriodBlocks: 34272,
	})

	// Spend 3 LGM — should succeed
	err := k.CheckAndDebitSpendingLimit(ctx, homeResp.HomeId, "operator", "3000000")
	if err != nil {
		t.Fatalf("expected spend within limit to succeed: %v", err)
	}

	// Verify spent counter updated
	limit, _ := k.GetSpendingLimit(ctx, homeResp.HomeId, "operator")
	if limit.SpentInPeriod != "3000000" {
		t.Errorf("expected spent 3000000, got %s", limit.SpentInPeriod)
	}

	// Spend another 5 LGM — should succeed (total 8)
	err = k.CheckAndDebitSpendingLimit(ctx, homeResp.HomeId, "operator", "5000000")
	if err != nil {
		t.Fatalf("expected second spend to succeed: %v", err)
	}

	limit, _ = k.GetSpendingLimit(ctx, homeResp.HomeId, "operator")
	if limit.SpentInPeriod != "8000000" {
		t.Errorf("expected spent 8000000, got %s", limit.SpentInPeriod)
	}
}

func TestCheckAndDebitSpendingLimit_ExceedsLimit(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1, Name: "SpendLimit Exceed Home",
	})

	_, _ = msgServer.SetSpendingLimit(ctx, &types.MsgSetSpendingLimit{
		Owner: owner1, HomeId: homeResp.HomeId,
		KeyType: "operator", MaxAmount: "5000000", PeriodBlocks: 34272,
	})

	// Spend 3 LGM — ok
	_ = k.CheckAndDebitSpendingLimit(ctx, homeResp.HomeId, "operator", "3000000")

	// Spend another 3 LGM — should fail (total 6 > 5)
	err := k.CheckAndDebitSpendingLimit(ctx, homeResp.HomeId, "operator", "3000000")
	if err == nil {
		t.Fatal("expected error when exceeding spending limit")
	}

	// Verify spent counter was NOT updated (still 3)
	limit, _ := k.GetSpendingLimit(ctx, homeResp.HomeId, "operator")
	if limit.SpentInPeriod != "3000000" {
		t.Errorf("expected spent still 3000000 after rejection, got %s", limit.SpentInPeriod)
	}

	// Verify alert was created
	var alertCount uint32
	k.IterateAlerts(ctx, homeResp.HomeId, func(alert *types.Alert) bool {
		if alert.Type == string(types.AlertSpendingLimitHit) {
			alertCount++
		}
		return false
	})
	if alertCount != 1 {
		t.Errorf("expected 1 spending limit alert, got %d", alertCount)
	}
}

func TestCheckAndDebitSpendingLimit_PeriodReset(t *testing.T) {
	k, ctx := setupKeeper(t)

	homeId := "home-period-reset"
	k.SetHome(ctx, &types.AgentHome{
		HomeId: homeId, OwnerAddress: owner1, Name: "Period Reset",
		Status:   string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	})

	// Set limit with period starting at block 50, period of 20 blocks
	k.SetSpendingLimit(ctx, homeId, &types.SpendingLimit{
		KeyType: "operator", MaxAmount: "5000000",
		PeriodBlocks: 20, SpentInPeriod: "4000000", PeriodStart: 50,
	})

	// Context is at block 100. Period was 50+20=70, so 100 >= 70: period should reset.
	err := k.CheckAndDebitSpendingLimit(ctx, homeId, "operator", "3000000")
	if err != nil {
		t.Fatalf("expected spend after period reset to succeed: %v", err)
	}

	limit, _ := k.GetSpendingLimit(ctx, homeId, "operator")
	if limit.SpentInPeriod != "3000000" {
		t.Errorf("expected spent 3000000 after reset, got %s", limit.SpentInPeriod)
	}
	if limit.PeriodStart != 100 {
		t.Errorf("expected period start reset to 100, got %d", limit.PeriodStart)
	}
}

func TestCheckAndDebitSpendingLimit_NoLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	homeId := "home-no-limit"
	k.SetHome(ctx, &types.AgentHome{
		HomeId: homeId, OwnerAddress: owner1, Name: "No Limit",
		Status:   string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	})

	// No spending limit configured — should allow any amount
	err := k.CheckAndDebitSpendingLimit(ctx, homeId, "operator", "999999999")
	if err != nil {
		t.Fatalf("expected no-limit spend to succeed: %v", err)
	}
}

func TestCheckAndDebitSpendingLimit_InvalidAmount(t *testing.T) {
	k, ctx := setupKeeper(t)

	homeId := "home-invalid-amt"
	k.SetHome(ctx, &types.AgentHome{
		HomeId: homeId, OwnerAddress: owner1, Name: "Invalid Amt",
		Status:   string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	})

	k.SetSpendingLimit(ctx, homeId, &types.SpendingLimit{
		KeyType: "operator", MaxAmount: "5000000",
		PeriodBlocks: 34272, SpentInPeriod: "0", PeriodStart: 100,
	})

	// Zero amount
	err := k.CheckAndDebitSpendingLimit(ctx, homeId, "operator", "0")
	if err == nil {
		t.Error("expected error for zero amount")
	}

	// Negative amount
	err = k.CheckAndDebitSpendingLimit(ctx, homeId, "operator", "-100")
	if err == nil {
		t.Error("expected error for negative amount")
	}

	// Non-numeric
	err = k.CheckAndDebitSpendingLimit(ctx, homeId, "operator", "abc")
	if err == nil {
		t.Error("expected error for non-numeric amount")
	}
}

func TestGetRemainingSpendingLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	homeId := "home-remaining"
	k.SetHome(ctx, &types.AgentHome{
		HomeId: homeId, OwnerAddress: owner1, Name: "Remaining",
		Status:   string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	})

	// No limit — returns -1
	remaining := k.GetRemainingSpendingLimit(ctx, homeId, "operator")
	if remaining != "-1" {
		t.Errorf("expected -1 for no limit, got %s", remaining)
	}

	// Set limit: max 10, spent 3, period still active
	k.SetSpendingLimit(ctx, homeId, &types.SpendingLimit{
		KeyType: "operator", MaxAmount: "10000000",
		PeriodBlocks: 34272, SpentInPeriod: "3000000", PeriodStart: 90,
	})

	remaining = k.GetRemainingSpendingLimit(ctx, homeId, "operator")
	if remaining != "7000000" {
		t.Errorf("expected remaining 7000000, got %s", remaining)
	}

	// Period expired — should return full max amount
	k.SetSpendingLimit(ctx, homeId, &types.SpendingLimit{
		KeyType: "operator", MaxAmount: "10000000",
		PeriodBlocks: 5, SpentInPeriod: "8000000", PeriodStart: 50,
	})

	remaining = k.GetRemainingSpendingLimit(ctx, homeId, "operator")
	if remaining != "10000000" {
		t.Errorf("expected full limit after period reset, got %s", remaining)
	}
}

// ==================== Treasury Automation ====================

func TestProcessTreasuryAutomation_AutoCompound(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create home with auto-compound enabled
	home := types.AgentHome{
		HomeId: "home-autocompound", OwnerAddress: owner1, Name: "AutoCompound",
		Status:   string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
		Treasury: &types.HomeTreasury{
			ReservedBalance: "0",
			Automation: &types.TreasuryAutomation{
				AutoCompoundRewards: true,
			},
		},
	}
	k.SetHome(ctx, &home)

	// Run automation — should emit event (no bank keeper needed)
	k.ProcessTreasuryAutomation(ctx)

	// Verify event was emitted
	events := ctx.EventManager().Events()
	found := false
	for _, ev := range events {
		if ev.Type == "treasury_auto_compound" {
			found = true
			for _, attr := range ev.Attributes {
				if attr.Key == "home_id" && attr.Value == "home-autocompound" {
					break
				}
			}
		}
	}
	if !found {
		t.Error("expected treasury_auto_compound event to be emitted")
	}
}

func TestProcessTreasuryAutomation_SkipsDormant(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create a dormant home with automation
	home := types.AgentHome{
		HomeId: "home-dormant-auto", OwnerAddress: owner1, Name: "DormantAuto",
		Status:   string(types.HomeStatusDormant),
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
		Treasury: &types.HomeTreasury{
			Automation: &types.TreasuryAutomation{AutoCompoundRewards: true},
		},
	}
	k.SetHome(ctx, &home)

	k.ProcessTreasuryAutomation(ctx)

	// No events should be emitted for dormant homes
	events := ctx.EventManager().Events()
	for _, ev := range events {
		if ev.Type == "treasury_auto_compound" {
			t.Error("should not emit auto-compound for dormant home")
		}
	}
}

func TestProcessTreasuryAutomation_NoAutomation(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Home without automation configured
	home := types.AgentHome{
		HomeId: "home-no-auto", OwnerAddress: owner1, Name: "NoAuto",
		Status:   string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
		Treasury: &types.HomeTreasury{ReservedBalance: "0"},
	}
	k.SetHome(ctx, &home)

	// Should not panic and not emit events
	k.ProcessTreasuryAutomation(ctx)

	events := ctx.EventManager().Events()
	for _, ev := range events {
		if ev.Type == "treasury_auto_compound" {
			t.Error("should not emit events for home without automation")
		}
	}
}

// ==================== Aging / Maturity System ====================

func TestComputeMaturityStage(t *testing.T) {
	tests := []struct {
		ageBlocks uint64
		expected  string
	}{
		{0, keeper.MaturityFresh},
		{1000, keeper.MaturityFresh},
		{keeper.ThresholdEstablished - 1, keeper.MaturityFresh},
		{keeper.ThresholdEstablished, keeper.MaturityEstablished},
		{keeper.ThresholdEstablished + 1, keeper.MaturityEstablished},
		{keeper.ThresholdMature - 1, keeper.MaturityEstablished},
		{keeper.ThresholdMature, keeper.MaturityMature},
		{keeper.ThresholdVenerable - 1, keeper.MaturityMature},
		{keeper.ThresholdVenerable, keeper.MaturityVenerable},
		{keeper.ThresholdAncient - 1, keeper.MaturityVenerable},
		{keeper.ThresholdAncient, keeper.MaturityAncient},
		{keeper.ThresholdAncient + 1000000, keeper.MaturityAncient},
	}

	for _, tc := range tests {
		result := keeper.ComputeMaturityStage(tc.ageBlocks)
		if result != tc.expected {
			t.Errorf("ComputeMaturityStage(%d) = %s, want %s", tc.ageBlocks, result, tc.expected)
		}
	}
}

func TestGetRewardMultiplierBps(t *testing.T) {
	tests := []struct {
		stage    string
		expected uint32
	}{
		{keeper.MaturityFresh, keeper.RewardMultiplierFresh},
		{keeper.MaturityEstablished, keeper.RewardMultiplierEstablished},
		{keeper.MaturityMature, keeper.RewardMultiplierMature},
		{keeper.MaturityVenerable, keeper.RewardMultiplierVenerable},
		{keeper.MaturityAncient, keeper.RewardMultiplierAncient},
		{"unknown", keeper.RewardMultiplierFresh}, // default to fresh
	}

	for _, tc := range tests {
		result := keeper.GetRewardMultiplierBps(tc.stage)
		if result != tc.expected {
			t.Errorf("GetRewardMultiplierBps(%s) = %d, want %d", tc.stage, result, tc.expected)
		}
	}
}

func TestComputePatina(t *testing.T) {
	// Zero age, zero epistemic score, stable (0 status changes) → stability factor = 20
	patina := keeper.ComputePatina(0, 0, 0)
	if patina != 20 {
		t.Errorf("expected patina 20 for zero-age stable home, got %d", patina)
	}

	// 1 day old, moderate epistemic score (50/100), no status changes — mid-range
	patina = keeper.ComputePatina(keeper.BlocksPerDay, 50, 0)
	if patina == 0 {
		t.Error("expected non-zero patina for 1-day-old active home")
	}

	// Very old (365 days), high epistemic score (90/100), no status changes — should be high
	age365 := uint64(keeper.BlocksPerDay * 365)
	patina = keeper.ComputePatina(age365, 90, 0)
	if patina < 80 {
		t.Errorf("expected high patina for 365-day active home, got %d", patina)
	}

	// Old but lots of status changes — stability factor hurts
	patina10changes := keeper.ComputePatina(age365, 90, 10)
	patinaStable := keeper.ComputePatina(age365, 90, 0)
	if patina10changes >= patinaStable {
		t.Errorf("expected lower patina with 10 status changes: %d >= %d", patina10changes, patinaStable)
	}

	// Old but no epistemic activity — epistemic factor hurts
	patinaInactive := keeper.ComputePatina(age365, 0, 0)
	patinaActive := keeper.ComputePatina(age365, 90, 0)
	if patinaInactive >= patinaActive {
		t.Errorf("expected lower patina when no epistemic activity: %d >= %d", patinaInactive, patinaActive)
	}

	// Patina capped at 100
	huge := uint64(keeper.BlocksPerDay * 10000)
	patinaCapped := keeper.ComputePatina(huge, 100, 0)
	if patinaCapped > 100 {
		t.Errorf("expected patina capped at 100, got %d", patinaCapped)
	}
}

func TestUpdateHomeMaturity(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create home at block 10. Context is at block 100, so age = 90.
	// 90 blocks is well below ThresholdEstablished (34272), so stage = fresh.
	home := types.AgentHome{
		HomeId: "home-maturity", OwnerAddress: owner1, Name: "Maturity Test",
		Status:         string(types.HomeStatusActive),
		CreatedAtBlock: 10,
		Guardian:       &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	}
	k.SetHome(ctx, &home)

	changed := k.UpdateHomeMaturity(ctx, &home)
	// Stage is fresh, default is also fresh → no change
	if changed {
		t.Error("expected no change for fresh home at low age")
	}

	stage := k.GetHomeMaturity(ctx, "home-maturity")
	if stage != keeper.MaturityFresh {
		t.Errorf("expected fresh, got %s", stage)
	}

	// Now simulate an old home (created at block 0, current at block 100000)
	oldCtx := ctx.WithBlockHeight(100000)
	oldHome := types.AgentHome{
		HomeId: "home-old", OwnerAddress: owner1, Name: "Old Home",
		Status:         string(types.HomeStatusActive),
		CreatedAtBlock: 0,
		Guardian:       &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	}
	k.SetHome(oldCtx, &oldHome)

	changed = k.UpdateHomeMaturity(oldCtx, &oldHome)
	if !changed {
		t.Error("expected maturity change for old home")
	}

	stage = k.GetHomeMaturity(oldCtx, "home-old")
	// 100000 blocks > ThresholdMature (342720)? No, 100000 < 342720.
	// 100000 > ThresholdEstablished (34272)? Yes.
	if stage != keeper.MaturityEstablished {
		t.Errorf("expected established for 100000-block-old home, got %s", stage)
	}
}

func TestUpdateHomeMaturity_AllStages(t *testing.T) {
	k, ctx := setupKeeper(t)

	stages := []struct {
		height   int64
		created  uint64
		expected string
	}{
		{100, 99, keeper.MaturityFresh},                                       // 1 block old
		{keeper.ThresholdEstablished + 10, 10, keeper.MaturityEstablished},    // just past established
		{int64(keeper.ThresholdMature) + 10, 10, keeper.MaturityMature},       // just past mature
		{int64(keeper.ThresholdVenerable) + 10, 10, keeper.MaturityVenerable}, // just past venerable
		{int64(keeper.ThresholdAncient) + 10, 10, keeper.MaturityAncient},     // just past ancient
	}

	for i, tc := range stages {
		stageCtx := ctx.WithBlockHeight(tc.height)
		homeId := fmt.Sprintf("home-stage-%d", i)
		home := types.AgentHome{
			HomeId: homeId, OwnerAddress: owner1, Name: "Stage Test",
			Status:         string(types.HomeStatusActive),
			CreatedAtBlock: tc.created,
			Guardian:       &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
		}
		k.SetHome(stageCtx, &home)
		k.UpdateHomeMaturity(stageCtx, &home)

		got := k.GetHomeMaturity(stageCtx, homeId)
		if got != tc.expected {
			t.Errorf("stage %d: expected %s, got %s (height=%d, created=%d)", i, tc.expected, got, tc.height, tc.created)
		}
	}
}

func TestGetHomeRewardMultiplierBps(t *testing.T) {
	k, ctx := setupKeeper(t)

	homeId := "home-reward-mult"
	// Default (no maturity stored) — should be fresh multiplier
	mult := k.GetHomeRewardMultiplierBps(ctx, homeId)
	if mult != keeper.RewardMultiplierFresh {
		t.Errorf("expected fresh multiplier %d, got %d", keeper.RewardMultiplierFresh, mult)
	}

	// Set to venerable
	k.SetHomeMaturity(ctx, homeId, keeper.MaturityVenerable)
	mult = k.GetHomeRewardMultiplierBps(ctx, homeId)
	if mult != keeper.RewardMultiplierVenerable {
		t.Errorf("expected venerable multiplier %d, got %d", keeper.RewardMultiplierVenerable, mult)
	}
}

func TestProcessHomeAging_Periodic(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create an active home
	home := types.AgentHome{
		HomeId: "home-aging-periodic", OwnerAddress: owner1, Name: "Aging Test",
		Status:         string(types.HomeStatusActive),
		CreatedAtBlock: 0,
		Guardian:       &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	}
	k.SetHome(ctx, &home)

	// Block 100 — 100 % 1428 != 0, so aging should NOT run
	k.ProcessHomeAging(ctx)
	stage := k.GetHomeMaturity(ctx, "home-aging-periodic")
	if stage != keeper.MaturityFresh {
		t.Errorf("expected fresh (aging should not run at non-periodic block), got %s", stage)
	}

	// Block 1428 — should run aging
	periodicCtx := ctx.WithBlockHeight(1428)
	k.ProcessHomeAging(periodicCtx)
	stage = k.GetHomeMaturity(periodicCtx, "home-aging-periodic")
	// Age at 1428 blocks is still below ThresholdEstablished (34272), so still fresh
	if stage != keeper.MaturityFresh {
		t.Errorf("expected fresh at 1428 blocks, got %s", stage)
	}

	// Block 1428 * 25 = 35700 — just past ThresholdEstablished (34272)
	olderCtx := ctx.WithBlockHeight(35700)
	// Need to reconfigure — block height 35700, home created at 0, age = 35700
	k.ProcessHomeAging(olderCtx)
	stage = k.GetHomeMaturity(olderCtx, "home-aging-periodic")
	// 35700 >= 34272 → established
	if stage != keeper.MaturityEstablished {
		t.Errorf("expected established at 35700 blocks, got %s", stage)
	}
}

func TestProcessHomeAging_SkipsArchived(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create an archived home that would otherwise be established by age
	home := types.AgentHome{
		HomeId: "home-archived-aging", OwnerAddress: owner1, Name: "Archived",
		Status:         string(types.HomeStatusArchived),
		CreatedAtBlock: 0,
		Guardian:       &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	}
	archivedCtx := ctx.WithBlockHeight(int64(keeper.ThresholdEstablished) + 1428)
	k.SetHome(archivedCtx, &home)

	// Run at periodic block
	blockHeight := int64(keeper.ThresholdEstablished) + 1428
	// Ensure it's divisible by 1428
	blockHeight = (blockHeight / 1428) * 1428
	periodicCtx := ctx.WithBlockHeight(blockHeight)
	k.ProcessHomeAging(periodicCtx)

	// Archived home should not have maturity updated
	stage := k.GetHomeMaturity(periodicCtx, "home-archived-aging")
	if stage != keeper.MaturityFresh {
		t.Errorf("expected fresh for archived home (should be skipped), got %s", stage)
	}
}

func TestMaturityKVStore(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Default returns fresh
	stage := k.GetHomeMaturity(ctx, "nonexistent")
	if stage != keeper.MaturityFresh {
		t.Errorf("expected fresh for nonexistent home, got %s", stage)
	}

	// Set and get
	k.SetHomeMaturity(ctx, "test-home", keeper.MaturityMature)
	stage = k.GetHomeMaturity(ctx, "test-home")
	if stage != keeper.MaturityMature {
		t.Errorf("expected mature, got %s", stage)
	}

	// Override
	k.SetHomeMaturity(ctx, "test-home", keeper.MaturityAncient)
	stage = k.GetHomeMaturity(ctx, "test-home")
	if stage != keeper.MaturityAncient {
		t.Errorf("expected ancient after override, got %s", stage)
	}
}

func TestMaturityConstants(t *testing.T) {
	// Verify threshold ordering
	if keeper.ThresholdEstablished >= keeper.ThresholdMature {
		t.Error("established threshold should be less than mature")
	}
	if keeper.ThresholdMature >= keeper.ThresholdVenerable {
		t.Error("mature threshold should be less than venerable")
	}
	if keeper.ThresholdVenerable >= keeper.ThresholdAncient {
		t.Error("venerable threshold should be less than ancient")
	}

	// Verify reward multiplier ordering
	if keeper.RewardMultiplierFresh >= keeper.RewardMultiplierEstablished {
		t.Error("fresh multiplier should be less than established")
	}
	if keeper.RewardMultiplierEstablished >= keeper.RewardMultiplierMature {
		t.Error("established multiplier should be less than mature")
	}
	if keeper.RewardMultiplierMature >= keeper.RewardMultiplierVenerable {
		t.Error("mature multiplier should be less than venerable")
	}
	if keeper.RewardMultiplierVenerable >= keeper.RewardMultiplierAncient {
		t.Error("venerable multiplier should be less than ancient")
	}

	// Verify BlocksPerDay matches protocol constant
	if keeper.BlocksPerDay != 34272 {
		t.Errorf("expected BlocksPerDay 34272, got %d", keeper.BlocksPerDay)
	}
}

// ==================== Session Cleanup ====================

func TestCleanupExpiredSessions(t *testing.T) {
	k, ctx := setupKeeper(t)

	homeId := "home-session-cleanup"
	k.SetHome(ctx, &types.AgentHome{
		HomeId: homeId, OwnerAddress: owner1, Name: "Session Cleanup",
		Status:   string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	})

	// Create an expired session (expires at block 50, current is 100)
	k.SetSession(ctx, &types.ActiveSession{
		SessionId: "expired-1", HomeId: homeId,
		StartedAt: 40, ExpiresAt: 50, KeyHash: "key1",
	})

	// Create a valid session (expires at block 200)
	k.SetSession(ctx, &types.ActiveSession{
		SessionId: "valid-1", HomeId: homeId,
		StartedAt: 90, ExpiresAt: 200, KeyHash: "key2",
	})

	k.CleanupExpiredSessions(ctx)

	// Expired session should be deleted
	_, found := k.GetSession(ctx, homeId, "expired-1")
	if found {
		t.Error("expired session should be deleted")
	}

	// Valid session should remain
	_, found = k.GetSession(ctx, homeId, "valid-1")
	if !found {
		t.Error("valid session should remain")
	}
}

// ==================== Alert Cleanup ====================

func TestCleanupOldAlerts(t *testing.T) {
	k, ctx := setupKeeper(t)

	params := types.DefaultParams()
	params.AlertRetentionBlocks = 50
	k.SetParams(ctx, &params)

	homeId := "home-alert-cleanup"
	k.SetHome(ctx, &types.AgentHome{
		HomeId: homeId, OwnerAddress: owner1, Name: "Alert Cleanup",
		Status:   string(types.HomeStatusActive),
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	})

	// Old acknowledged alert (created at block 10, retention 50, current 100: 10+50=60 < 100)
	k.SetAlert(ctx, &types.Alert{
		AlertId: "old-ack", HomeId: homeId, Type: string(types.AlertDefenseActivated),
		Priority: string(types.AlertPriorityLow), CreatedAt: 10, Acknowledged: true,
	})

	// Recent acknowledged alert (created at block 80, 80+50=130 > 100)
	k.SetAlert(ctx, &types.Alert{
		AlertId: "recent-ack", HomeId: homeId, Type: string(types.AlertDefenseActivated),
		Priority: string(types.AlertPriorityLow), CreatedAt: 80, Acknowledged: true,
	})

	// Old unacknowledged alert — should NOT be cleaned up
	k.SetAlert(ctx, &types.Alert{
		AlertId: "old-unack", HomeId: homeId, Type: string(types.AlertDefenseActivated),
		Priority: string(types.AlertPriorityHigh), CreatedAt: 10, Acknowledged: false,
	})

	k.CleanupOldAlerts(ctx)

	// Old acknowledged should be deleted
	_, found := k.GetAlert(ctx, homeId, "old-ack")
	if found {
		t.Error("old acknowledged alert should be cleaned up")
	}

	// Recent acknowledged should remain
	_, found = k.GetAlert(ctx, homeId, "recent-ack")
	if !found {
		t.Error("recent acknowledged alert should remain")
	}

	// Old unacknowledged should remain
	_, found = k.GetAlert(ctx, homeId, "old-unack")
	if !found {
		t.Error("old unacknowledged alert should remain")
	}
}

// ==================== Comfort Dimensions ====================

func TestComfortDimensions_DefaultHome(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Minimal home — low scores across the board
	home := types.AgentHome{
		HomeId: "home-comfort-default", OwnerAddress: owner1, Name: "Comfort Test",
		Status: string(types.HomeStatusActive), CreatedAtBlock: 90,
		Guardian: &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
		Treasury: &types.HomeTreasury{ReservedBalance: "0"},
	}
	k.SetHome(ctx, &home)

	dims := k.ComputeComfortDimensions(ctx, &home)

	// Sanctuary: active=80, no autodefend → 80
	if dims.Sanctuary != 80 {
		t.Errorf("expected sanctuary 80, got %d", dims.Sanctuary)
	}
	// Identity: 30 (name) + 0 (no DID) + 0 (fresh) = 30
	if dims.Identity != 30 {
		t.Errorf("expected identity 30, got %d", dims.Identity)
	}
	// Memory: no CID, no epistemic = 0
	if dims.Memory != 0 {
		t.Errorf("expected memory 0, got %d", dims.Memory)
	}
	// Growth: fresh = 20
	if dims.Growth != 20 {
		t.Errorf("expected growth 20, got %d", dims.Growth)
	}
	// Privacy: 30 + moderate(30) = 60
	if dims.Privacy != 60 {
		t.Errorf("expected privacy 60, got %d", dims.Privacy)
	}

	composite := dims.CompositeScore()
	if composite == 0 {
		t.Error("expected non-zero composite score")
	}
}

func TestComfortDimensions_RichHome(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1, Name: "Rich Home",
		InitialGuardianConfig: &types.HomeGuardian{
			DefenseStrategy: string(types.DefenseAggressive),
			AutoDefend:      true,
			RecoveryDids:    []string{"did:1", "did:2", "did:3", "did:4"},
			RecoveryThreshold: 2,
			Deadman: &types.DeadmanConfig{
				Enabled: true, InactivityThreshold: 50000,
				Action: string(types.DeadmanAlertGuardians),
			},
		},
	})

	// Set memory CID
	msgServer.UpdateMemoryCID(ctx, &types.MsgUpdateMemoryCID{
		Owner: owner1, HomeId: homeResp.HomeId, Cid: "QmTest123",
	})

	// Register keys
	msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner: owner1, HomeId: homeResp.HomeId, KeyHash: "key1",
		KeyType: "ed25519", Role: string(types.AccessRoleOperator),
		Permissions: []string{string(types.PermReadMemory)},
	})
	msgServer.RegisterKey(ctx, &types.MsgRegisterKey{
		Owner: owner1, HomeId: homeResp.HomeId, KeyHash: "key2",
		KeyType: "ed25519", Role: string(types.AccessRoleGuest),
		Permissions: []string{string(types.PermReadMemory)},
	})

	// Set spending limit
	msgServer.SetSpendingLimit(ctx, &types.MsgSetSpendingLimit{
		Owner: owner1, HomeId: homeResp.HomeId,
		KeyType: "operator", MaxAmount: "5000000", PeriodBlocks: 34272,
	})

	// Record epistemic activity
	for i := 0; i < 20; i++ {
		k.RecordVerification(ctx, owner1, "science", true)
	}

	home, _ := k.GetHome(ctx, homeResp.HomeId)
	dims := k.ComputeComfortDimensions(ctx, home)

	// Sanctuary: active(80) + autodefend(20) = 100
	if dims.Sanctuary != 100 {
		t.Errorf("expected sanctuary 100, got %d", dims.Sanctuary)
	}
	// Belonging: 4 recovery DIDs * 25 = 100, + threshold bonus capped
	if dims.Belonging < 100 {
		t.Errorf("expected belonging >= 100, got %d", dims.Belonging)
	}
	// Rest: 30 + deadman(40) = 70 (no age bonus, only 10 blocks old)
	if dims.Rest != 70 {
		t.Errorf("expected rest 70, got %d", dims.Rest)
	}
	// Achievement should be > 0
	if dims.Achievement == 0 {
		t.Error("expected non-zero achievement with epistemic activity")
	}
	// Memory: CID(50) + epistemic bonus
	if dims.Memory < 50 {
		t.Errorf("expected memory >= 50, got %d", dims.Memory)
	}
}

func TestComfortDimensions_StorageRoundTrip(t *testing.T) {
	k, ctx := setupKeeper(t)

	dims := types.ComfortDimensions{
		Sanctuary: 80, Identity: 60, Memory: 50, Belonging: 40, Rest: 70,
		Growth: 30, Control: 55, Privacy: 65, Warmth: 45, Achievement: 35,
	}
	k.SetComfortDimensions(ctx, "test-home", dims)

	got := k.GetComfortDimensions(ctx, "test-home")
	if got != dims {
		t.Errorf("round-trip mismatch: got %+v", got)
	}
}

func TestComfortDimensions_CompositeScore(t *testing.T) {
	dims := types.ComfortDimensions{
		Sanctuary: 100, Identity: 100, Memory: 100, Belonging: 100, Rest: 100,
		Growth: 100, Control: 100, Privacy: 100, Warmth: 100, Achievement: 100,
	}
	if dims.CompositeScore() != 100 {
		t.Errorf("expected composite 100, got %d", dims.CompositeScore())
	}

	zero := types.ComfortDimensions{}
	if zero.CompositeScore() != 0 {
		t.Errorf("expected composite 0, got %d", zero.CompositeScore())
	}
}

// ==================== Epistemic Stats ====================

func TestRecordVerification(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	homeResp, _ := msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1, Name: "Epistemic Home",
	})
	_ = homeResp

	// Record verifications
	k.RecordVerification(ctx, owner1, "physics", true)
	k.RecordVerification(ctx, owner1, "physics", true)
	k.RecordVerification(ctx, owner1, "physics", false)

	homeIds := k.GetHomesByOwner(ctx, owner1)
	stats := k.GetEpistemicStats(ctx, homeIds[0])

	if stats.VerificationsDone != 3 {
		t.Errorf("expected 3 verifications, got %d", stats.VerificationsDone)
	}
	if stats.AccurateVerdicts != 2 {
		t.Errorf("expected 2 accurate verdicts, got %d", stats.AccurateVerdicts)
	}
	if stats.TotalVerdicts != 3 {
		t.Errorf("expected 3 total verdicts, got %d", stats.TotalVerdicts)
	}
	if stats.AccuracyRate() != 66 {
		t.Errorf("expected accuracy 66%%, got %d%%", stats.AccuracyRate())
	}
}

func TestRecordClaimAndChallenge(t *testing.T) {
	k, ctx := setupKeeper(t)
	msgServer := keeper.NewMsgServerImpl(k)

	msgServer.CreateHome(ctx, &types.MsgCreateHome{
		Owner: owner1, Name: "Claims Home",
	})

	k.RecordClaimSubmission(ctx, owner1)
	k.RecordClaimSubmission(ctx, owner1)
	k.RecordChallenge(ctx, owner1)

	homeIds := k.GetHomesByOwner(ctx, owner1)
	stats := k.GetEpistemicStats(ctx, homeIds[0])

	if stats.ClaimsSubmitted != 2 {
		t.Errorf("expected 2 claims, got %d", stats.ClaimsSubmitted)
	}
	if stats.ChallengesInitiated != 1 {
		t.Errorf("expected 1 challenge, got %d", stats.ChallengesInitiated)
	}
}

func TestEpistemicScore(t *testing.T) {
	// Zero stats
	s := types.EpistemicStats{}
	if s.EpistemicScore() != 0 {
		t.Errorf("expected 0, got %d", s.EpistemicScore())
	}

	// Active agent
	s = types.EpistemicStats{
		VerificationsDone: 100,
		AccurateVerdicts:  90,
		TotalVerdicts:     100,
		DomainsActive:     5,
	}
	score := s.EpistemicScore()
	if score < 50 {
		t.Errorf("expected score > 50 for active agent, got %d", score)
	}
}

func TestRecordVerification_NoHome(t *testing.T) {
	k, ctx := setupKeeper(t)
	// Should not panic when owner has no home
	k.RecordVerification(ctx, owner2, "math", true)
}

// ==================== UpdateParams ====================

func TestUpdateParams(t *testing.T) {
	k, ctx := setupKeeper(t)

	newParams := types.DefaultParams()
	newParams.MaxHomesPerAccount = 10
	newParams.MaxKeysPerHome = 50
	k.SetParams(ctx, &newParams)

	got := k.GetParams(ctx)
	if got.MaxHomesPerAccount != 10 {
		t.Errorf("expected 10, got %d", got.MaxHomesPerAccount)
	}
	if got.MaxKeysPerHome != 50 {
		t.Errorf("expected 50, got %d", got.MaxKeysPerHome)
	}
}

func TestUpdateParams_Authority(t *testing.T) {
	k, _ := setupKeeper(t)

	if k.GetAuthority() != "lgm1authority" {
		t.Errorf("expected lgm1authority, got %s", k.GetAuthority())
	}
}

// ==================== DimensionNames ====================

func TestDimensionNames(t *testing.T) {
	names := types.DimensionNames()
	if len(names) != 10 {
		t.Errorf("expected 10 dimensions, got %d", len(names))
	}
	expected := []string{"sanctuary", "identity", "memory", "belonging", "rest",
		"growth", "control", "privacy", "warmth", "achievement"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("dimension %d: expected %s, got %s", i, name, names[i])
		}
	}
}

// ==================== Memory Encryption (AES-256-GCM) ====================

func TestEncryptDecryptRoundtrip(t *testing.T) {
	k, ctx := setupKeeper(t)

	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		t.Fatal(err)
	}

	plainCID := "QmYwAPJzv5CZsnA625s3Xf2nemtYgPpHdWEz79ojWnPbdG"
	encrypted, err := k.EncryptMemoryCID(ctx, "home-1", plainCID, key)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	if encrypted == "" || encrypted == plainCID {
		t.Error("expected non-empty ciphertext different from plaintext")
	}

	decrypted, err := k.DecryptMemoryCID(ctx, "home-1", encrypted, key)
	if err != nil {
		t.Fatalf("decrypt failed: %v", err)
	}
	if decrypted != plainCID {
		t.Errorf("roundtrip mismatch: got %s, want %s", decrypted, plainCID)
	}
}

func TestDecryptWrongKeyFails(t *testing.T) {
	k, ctx := setupKeeper(t)

	keyA := make([]byte, 32)
	keyB := make([]byte, 32)
	rand.Read(keyA)
	rand.Read(keyB)

	encrypted, err := k.EncryptMemoryCID(ctx, "home-1", "QmTestCID", keyA)
	if err != nil {
		t.Fatal(err)
	}

	_, err = k.DecryptMemoryCID(ctx, "home-1", encrypted, keyB)
	if err == nil {
		t.Error("expected decryption to fail with wrong key")
	}
}

func TestDecryptTamperedCiphertextFails(t *testing.T) {
	k, ctx := setupKeeper(t)

	key := make([]byte, 32)
	rand.Read(key)

	encrypted, _ := k.EncryptMemoryCID(ctx, "home-1", "QmTestCID", key)

	// Tamper with the base64 content (flip a character in the middle)
	runes := []rune(encrypted)
	if len(runes) > 10 {
		if runes[10] == 'A' {
			runes[10] = 'B'
		} else {
			runes[10] = 'A'
		}
	}
	tampered := string(runes)

	_, err := k.DecryptMemoryCID(ctx, "home-1", tampered, key)
	if err == nil {
		t.Error("expected decryption to fail with tampered ciphertext")
	}
}

func TestEncryptEmptyCID(t *testing.T) {
	k, ctx := setupKeeper(t)

	key := make([]byte, 32)
	rand.Read(key)

	_, err := k.EncryptMemoryCID(ctx, "home-1", "", key)
	if err == nil {
		t.Error("expected error for empty CID")
	}
	if !strings.Contains(err.Error(), "empty") {
		t.Errorf("expected empty CID error, got: %v", err)
	}
}

func TestEncryptInvalidKeyLength(t *testing.T) {
	k, ctx := setupKeeper(t)

	// nil key
	_, err := k.EncryptMemoryCID(ctx, "home-1", "QmTest", nil)
	if err == nil {
		t.Error("expected error for nil key")
	}

	// 16-byte key (too short)
	shortKey := make([]byte, 16)
	_, err = k.EncryptMemoryCID(ctx, "home-1", "QmTest", shortKey)
	if err == nil {
		t.Error("expected error for 16-byte key")
	}

	// Decrypt with nil key
	_, err = k.DecryptMemoryCID(ctx, "home-1", "dGVzdA==", nil)
	if err == nil {
		t.Error("expected error for nil key on decrypt")
	}
}

// ==================== Defense Strategy Enactment ====================

func TestDefenseRateLimit(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create home with aggressive strategy (5 ops per 100 blocks)
	home := &types.AgentHome{
		HomeId:       "home-defense-1",
		OwnerDid:     "did:lgm:defender1",
		OwnerAddress: owner1,
		Name:         "Defense Home",
		Status:       string(types.HomeStatusActive),
		Guardian:     &types.HomeGuardian{DefenseStrategy: string(types.DefenseAggressive)},
	}
	k.SetHome(ctx, home)

	meta := map[string]string{"2fa_verified": "true"}

	// First 5 calls should succeed
	for i := 0; i < 5; i++ {
		err := k.EnforceDefenseStrategy(ctx, "home-defense-1", "key1", "transfer", meta)
		if err != nil {
			t.Fatalf("call %d should have succeeded: %v", i+1, err)
		}
		// Advance block past cooldown (aggressive = 10 blocks)
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 11)
	}

	// 6th call in same window should be rate-limited
	err := k.EnforceDefenseStrategy(ctx, "home-defense-1", "key1", "transfer", meta)
	if err == nil {
		t.Error("expected rate limit error on 6th call")
	}
	if !strings.Contains(err.Error(), "rate limit") {
		t.Errorf("expected rate limit error, got: %v", err)
	}
}

func TestDefenseCooldown(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create home with moderate strategy (cooldown = 3 blocks)
	home := &types.AgentHome{
		HomeId:       "home-defense-2",
		OwnerDid:     "did:lgm:defender2",
		OwnerAddress: owner1,
		Name:         "Cooldown Home",
		Status:       string(types.HomeStatusActive),
		Guardian:     &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	}
	k.SetHome(ctx, home)

	// First call succeeds
	err := k.EnforceDefenseStrategy(ctx, "home-defense-2", "key1", "transfer", nil)
	if err != nil {
		t.Fatalf("first call should succeed: %v", err)
	}

	// Immediate second call (same block) should fail cooldown
	err = k.EnforceDefenseStrategy(ctx, "home-defense-2", "key1", "transfer", nil)
	if err == nil {
		t.Error("expected cooldown error")
	}
	if !strings.Contains(err.Error(), "cooldown") {
		t.Errorf("expected cooldown error, got: %v", err)
	}

	// Advance past cooldown (3 blocks)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 4)
	err = k.EnforceDefenseStrategy(ctx, "home-defense-2", "key1", "transfer", nil)
	if err != nil {
		t.Fatalf("call after cooldown should succeed: %v", err)
	}
}

func TestDefense2FA(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create home with aggressive strategy (requires 2FA)
	home := &types.AgentHome{
		HomeId:       "home-defense-3",
		OwnerDid:     "did:lgm:defender3",
		OwnerAddress: owner1,
		Name:         "2FA Home",
		Status:       string(types.HomeStatusActive),
		Guardian:     &types.HomeGuardian{DefenseStrategy: string(types.DefenseAggressive)},
	}
	k.SetHome(ctx, home)

	// Without 2FA should fail
	err := k.EnforceDefenseStrategy(ctx, "home-defense-3", "key1", "transfer", nil)
	if err == nil {
		t.Error("expected 2FA error")
	}
	if !strings.Contains(err.Error(), "2FA") {
		t.Errorf("expected 2FA error, got: %v", err)
	}

	// With 2FA should succeed
	meta := map[string]string{"2fa_verified": "true"}
	err = k.EnforceDefenseStrategy(ctx, "home-defense-3", "key1", "transfer", meta)
	if err != nil {
		t.Fatalf("call with 2FA should succeed: %v", err)
	}
}

func TestDefenseBlacklist(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create home with moderate strategy (uses blacklist)
	home := &types.AgentHome{
		HomeId:       "home-defense-4",
		OwnerDid:     "did:lgm:defender4",
		OwnerAddress: owner1,
		Name:         "Blacklist Home",
		Status:       string(types.HomeStatusActive),
		Guardian:     &types.HomeGuardian{DefenseStrategy: string(types.DefenseModerate)},
	}
	k.SetHome(ctx, home)

	badAddr := "lgm1badactor"
	k.SetDefenseList(ctx, "home-defense-4", "blacklist", []string{badAddr})

	// Transfer to blacklisted address should fail
	meta := map[string]string{"target_address": badAddr}
	err := k.EnforceDefenseStrategy(ctx, "home-defense-4", "key1", "transfer", meta)
	if err == nil {
		t.Error("expected blacklist error")
	}
	if !strings.Contains(err.Error(), "blacklisted") {
		t.Errorf("expected blacklist error, got: %v", err)
	}

	// Transfer to non-blacklisted address should succeed (advance past cooldown)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 4)
	goodMeta := map[string]string{"target_address": "lgm1goodactor"}
	err = k.EnforceDefenseStrategy(ctx, "home-defense-4", "key1", "transfer", goodMeta)
	if err != nil {
		t.Fatalf("non-blacklisted transfer should succeed: %v", err)
	}
}

func TestDefenseWhitelist(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create home with conservative strategy (uses whitelist)
	home := &types.AgentHome{
		HomeId:       "home-defense-5",
		OwnerDid:     "did:lgm:defender5",
		OwnerAddress: owner1,
		Name:         "Whitelist Home",
		Status:       string(types.HomeStatusActive),
		Guardian:     &types.HomeGuardian{DefenseStrategy: string(types.DefenseConservative)},
	}
	k.SetHome(ctx, home)

	allowedAddr := "lgm1allowed"
	k.SetDefenseList(ctx, "home-defense-5", "whitelist", []string{allowedAddr})

	// Transfer to non-whitelisted address should fail
	meta := map[string]string{"target_address": "lgm1stranger"}
	err := k.EnforceDefenseStrategy(ctx, "home-defense-5", "key1", "transfer", meta)
	if err == nil {
		t.Error("expected whitelist error for non-whitelisted address")
	}
	if !strings.Contains(err.Error(), "not in whitelist") {
		t.Errorf("expected whitelist error, got: %v", err)
	}

	// Transfer to whitelisted address should succeed
	goodMeta := map[string]string{"target_address": allowedAddr}
	// Advance past cooldown for conservative (5 blocks)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 6)
	err = k.EnforceDefenseStrategy(ctx, "home-defense-5", "key1", "transfer", goodMeta)
	if err != nil {
		t.Fatalf("whitelisted transfer should succeed: %v", err)
	}
}

func TestDefenseDiplomaticAllowsMost(t *testing.T) {
	k, ctx := setupKeeper(t)

	home := &types.AgentHome{
		HomeId:       "home-defense-6",
		OwnerDid:     "did:lgm:diplomat",
		OwnerAddress: owner1,
		Name:         "Diplomatic Home",
		Status:       string(types.HomeStatusActive),
		Guardian:     &types.HomeGuardian{DefenseStrategy: string(types.DefenseDiplomatic)},
	}
	k.SetHome(ctx, home)

	// Rapid-fire calls should all succeed (no rate limit or cooldown)
	for i := 0; i < 20; i++ {
		err := k.EnforceDefenseStrategy(ctx, "home-defense-6", "key1", "transfer", nil)
		if err != nil {
			t.Fatalf("diplomatic call %d should succeed: %v", i+1, err)
		}
	}
}

func TestDefenseNoGuardianAllows(t *testing.T) {
	k, ctx := setupKeeper(t)

	home := &types.AgentHome{
		HomeId:       "home-defense-7",
		OwnerDid:     "did:lgm:noguard",
		OwnerAddress: owner1,
		Name:         "No Guardian Home",
		Status:       string(types.HomeStatusActive),
	}
	k.SetHome(ctx, home)

	err := k.EnforceDefenseStrategy(ctx, "home-defense-7", "key1", "transfer", nil)
	if err != nil {
		t.Fatalf("no guardian should allow all: %v", err)
	}
}

// ---------- Helpers ----------
