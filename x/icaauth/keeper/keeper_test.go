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

	legibleauthtypes "github.com/legible-money/legible/x/auth/types"
	"github.com/legible-money/legible/x/icaauth/keeper"
	"github.com/legible-money/legible/x/icaauth/types"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
}

// ============================================================
// Mock keepers
// ============================================================

// mockLegibleAuthKeeper implements types.LegibleAuthKeeper for testing.
type mockLegibleAuthKeeper struct {
	accounts map[string]*legibleauthtypes.LegibleAccount // address -> account
}

func newMockLegibleAuthKeeper() *mockLegibleAuthKeeper {
	return &mockLegibleAuthKeeper{
		accounts: make(map[string]*legibleauthtypes.LegibleAccount),
	}
}

func (m *mockLegibleAuthKeeper) GetAccount(_ sdk.Context, address string) (*legibleauthtypes.LegibleAccount, bool) {
	acc, ok := m.accounts[address]
	return acc, ok
}

func (m *mockLegibleAuthKeeper) GetAccountByDID(_ sdk.Context, did string) (*legibleauthtypes.LegibleAccount, bool) {
	for _, acc := range m.accounts {
		if acc.Did == did {
			return acc, true
		}
	}
	return nil, false
}

func (m *mockLegibleAuthKeeper) GetAddressForDID(_ sdk.Context, did string) (string, bool) {
	for _, acc := range m.accounts {
		if acc.Did == did {
			return acc.Address, true
		}
	}
	return "", false
}

// mockICAControllerKeeper implements types.ICAControllerKeeper for testing.
type mockICAControllerKeeper struct {
	registered map[string]bool // "connectionID:owner" -> true
	failNext   bool
}

func newMockICAControllerKeeper() *mockICAControllerKeeper {
	return &mockICAControllerKeeper{
		registered: make(map[string]bool),
	}
}

func (m *mockICAControllerKeeper) RegisterInterchainAccount(_ sdk.Context, connectionID, owner, _ string) error {
	if m.failNext {
		m.failNext = false
		return types.ErrNotRegistered // simulate failure
	}
	m.registered[connectionID+":"+owner] = true
	return nil
}

func (m *mockICAControllerKeeper) GetInterchainAccountAddress(_ sdk.Context, connectionID, portID string) (string, bool) {
	key := connectionID + ":" + portID
	if m.registered[key] {
		return "cosmos1remote" + connectionID, true
	}
	return "", false
}

func (m *mockICAControllerKeeper) GetOpenActiveChannel(_ sdk.Context, connectionID, portID string) (string, bool) {
	key := connectionID + ":" + portID
	if m.registered[key] {
		return "channel-remote-0", true
	}
	return "", false
}

// ============================================================
// Test setup
// ============================================================

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockLegibleAuthKeeper, *mockICAControllerKeeper) {
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

	authority := "lgm10d07y265gmmuvt4z0w9aw880jnsr700jvss730"

	mockAuth := newMockLegibleAuthKeeper()
	mockICA := newMockICAControllerKeeper()

	k := keeper.NewKeeper(cdc, storeService, mockAuth, mockICA, authority)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx, mockAuth, mockICA
}

// testDID is a valid DID for testing (32 hex chars suffix).
const testDID = "did:lgm:aabbccddaabbccddaabbccddaabbccdd"

// testAddress is a valid lgm bech32 address for testing.
var testAddress string

func init() {
	// Generate a valid lgm address for testing
	addr := sdk.AccAddress([]byte("test_icaauth_address"))
	testAddress = addr.String()
}

// ============================================================
// Record CRUD Tests
// ============================================================

func TestSetGetRecord(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	record := types.InterchainAccountRecord{
		Did: testDID,
		Accounts: []*types.RemoteAccount{
			{
				ConnectionId: "connection-0",
				PortId:       "icacontroller-test",
				OwnerDid:     testDID,
				Active:       false,
			},
		},
	}
	k.SetRecord(ctx, record)

	got, found := k.GetRecord(ctx, testDID)
	if !found {
		t.Fatal("expected record to be found")
	}
	if got.Did != testDID {
		t.Errorf("DID: want %s, got %s", testDID, got.Did)
	}
	if len(got.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(got.Accounts))
	}
	if got.Accounts[0].ConnectionId != "connection-0" {
		t.Errorf("ConnectionId: want connection-0, got %s", got.Accounts[0].ConnectionId)
	}
}

func TestGetRecordNotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	_, found := k.GetRecord(ctx, "did:lgm:nonexistent000000000000000000")
	if found {
		t.Error("expected record not to be found")
	}
}

// ============================================================
// AddRemoteAccount Tests
// ============================================================

func TestAddRemoteAccount_Success(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	account := &types.RemoteAccount{
		ConnectionId: "connection-0",
		PortId:       "icacontroller-test",
		OwnerDid:     testDID,
		Active:       false,
	}
	err := k.AddRemoteAccount(ctx, testDID, account)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	accounts := k.GetRemoteAccountsForDID(ctx, testDID)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(accounts))
	}
}

func TestAddRemoteAccount_MaxReached(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	// Set max to 2 for testing
	err := k.SetParams(ctx, types.Params{
		MaxRemoteAccountsPerDid: 2,
		RegistrationCooldown:    100,
	})
	if err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	// Add two accounts
	for i := 0; i < 2; i++ {
		acc := &types.RemoteAccount{
			ConnectionId: "connection-" + string(rune('0'+i)),
			OwnerDid:     testDID,
		}
		if err := k.AddRemoteAccount(ctx, testDID, acc); err != nil {
			t.Fatalf("failed to add account %d: %v", i, err)
		}
	}

	// Third should fail
	acc := &types.RemoteAccount{ConnectionId: "connection-2", OwnerDid: testDID}
	err = k.AddRemoteAccount(ctx, testDID, acc)
	if err == nil {
		t.Error("expected error for exceeding max accounts")
	}
}

func TestAddRemoteAccount_DuplicateConnection(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	acc := &types.RemoteAccount{ConnectionId: "connection-0", OwnerDid: testDID}
	if err := k.AddRemoteAccount(ctx, testDID, acc); err != nil {
		t.Fatalf("first add failed: %v", err)
	}

	// Duplicate connection
	acc2 := &types.RemoteAccount{ConnectionId: "connection-0", OwnerDid: testDID}
	err := k.AddRemoteAccount(ctx, testDID, acc2)
	if err == nil {
		t.Error("expected error for duplicate connection")
	}
}

// ============================================================
// GetRemoteAccountByConnection Tests
// ============================================================

func TestGetRemoteAccountByConnection(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	acc := &types.RemoteAccount{ConnectionId: "connection-0", OwnerDid: testDID}
	_ = k.AddRemoteAccount(ctx, testDID, acc)

	got, found := k.GetRemoteAccountByConnection(ctx, testDID, "connection-0")
	if !found {
		t.Fatal("expected account to be found")
	}
	if got.ConnectionId != "connection-0" {
		t.Errorf("ConnectionId: want connection-0, got %s", got.ConnectionId)
	}
}

func TestGetRemoteAccountByConnection_NotFound(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	_, found := k.GetRemoteAccountByConnection(ctx, testDID, "connection-missing")
	if found {
		t.Error("expected account not to be found")
	}
}

// ============================================================
// UpdateRemoteAccountAddress Tests
// ============================================================

func TestUpdateRemoteAccountAddress(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	acc := &types.RemoteAccount{ConnectionId: "connection-0", OwnerDid: testDID, Active: false}
	_ = k.AddRemoteAccount(ctx, testDID, acc)

	k.UpdateRemoteAccountAddress(ctx, testDID, "connection-0", "cosmos1remoteaddr")

	got, found := k.GetRemoteAccountByConnection(ctx, testDID, "connection-0")
	if !found {
		t.Fatal("expected account to be found after update")
	}
	if got.RemoteAddress != "cosmos1remoteaddr" {
		t.Errorf("RemoteAddress: want cosmos1remoteaddr, got %s", got.RemoteAddress)
	}
	if !got.Active {
		t.Error("expected account to be active after address update")
	}
}

// ============================================================
// Params Tests
// ============================================================

func TestSetGetParams(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	params := types.Params{
		MaxRemoteAccountsPerDid: 5,
		AllowedHostMsgTypes:     []string{"/cosmos.bank.v1beta1.MsgSend"},
		RegistrationCooldown:    50,
	}
	if err := k.SetParams(ctx, params); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	got := k.GetParams(ctx)
	if got.MaxRemoteAccountsPerDid != 5 {
		t.Errorf("MaxRemoteAccountsPerDid: want 5, got %d", got.MaxRemoteAccountsPerDid)
	}
	if got.RegistrationCooldown != 50 {
		t.Errorf("RegistrationCooldown: want 50, got %d", got.RegistrationCooldown)
	}
}

func TestSetParams_Invalid(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	err := k.SetParams(ctx, types.Params{
		MaxRemoteAccountsPerDid: 0, // invalid
		RegistrationCooldown:    100,
	})
	if err == nil {
		t.Error("expected error for invalid params (zero max accounts)")
	}
}

func TestDefaultParams(t *testing.T) {
	params := types.DefaultParams()
	if params.MaxRemoteAccountsPerDid != 10 {
		t.Errorf("expected MaxRemoteAccountsPerDid=10, got %d", params.MaxRemoteAccountsPerDid)
	}
	if params.RegistrationCooldown != 100 {
		t.Errorf("expected RegistrationCooldown=100, got %d", params.RegistrationCooldown)
	}
}

// ============================================================
// Genesis Tests
// ============================================================

func TestInitExportGenesis(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	params := types.Params{
		MaxRemoteAccountsPerDid: 3,
		AllowedHostMsgTypes:     []string{"/cosmos.bank.v1beta1.MsgSend"},
		RegistrationCooldown:    200,
	}
	record := &types.InterchainAccountRecord{
		Did: testDID,
		Accounts: []*types.RemoteAccount{
			{ConnectionId: "connection-0", OwnerDid: testDID, Active: true, RemoteAddress: "cosmos1addr"},
		},
	}
	gs := types.GenesisState{
		Params:  &params,
		Records: []*types.InterchainAccountRecord{record},
	}

	if err := k.InitGenesis(ctx, gs); err != nil {
		t.Fatalf("InitGenesis failed: %v", err)
	}

	exported := k.ExportGenesis(ctx)
	if exported.Params.MaxRemoteAccountsPerDid != 3 {
		t.Errorf("exported params: want MaxRemoteAccountsPerDid=3, got %d", exported.Params.MaxRemoteAccountsPerDid)
	}
	if len(exported.Records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(exported.Records))
	}
	if exported.Records[0].Did != testDID {
		t.Errorf("expected DID %s, got %s", testDID, exported.Records[0].Did)
	}
}

func TestEmptyGenesis(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)

	gs := types.GenesisState{
		Params:  nil,
		Records: nil,
	}
	if err := k.InitGenesis(ctx, gs); err != nil {
		t.Fatalf("InitGenesis with empty state failed: %v", err)
	}

	exported := k.ExportGenesis(ctx)
	// Should get default params
	if exported.Params.MaxRemoteAccountsPerDid != 10 {
		t.Errorf("expected default MaxRemoteAccountsPerDid=10, got %d", exported.Params.MaxRemoteAccountsPerDid)
	}
}

// ============================================================
// MsgServer Tests
// ============================================================

func TestMsgRegisterInterchainAccount_Success(t *testing.T) {
	k, ctx, mockAuth, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Register a Legible account
	mockAuth.accounts[testAddress] = &legibleauthtypes.LegibleAccount{
		Address: testAddress,
		Did:     testDID,
	}

	msg := &types.MsgRegisterInterchainAccount{
		Owner:        testAddress,
		ConnectionId: "connection-0",
		Version:      "",
	}

	resp, err := ms.RegisterInterchainAccount(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PortId == "" {
		t.Error("expected non-empty PortId in response")
	}

	// Verify the account was stored
	accounts := k.GetRemoteAccountsForDID(ctx, testDID)
	if len(accounts) != 1 {
		t.Fatalf("expected 1 remote account, got %d", len(accounts))
	}
	if accounts[0].ConnectionId != "connection-0" {
		t.Errorf("ConnectionId: want connection-0, got %s", accounts[0].ConnectionId)
	}
	if accounts[0].Active {
		t.Error("expected account to be inactive (pending)")
	}
}

func TestMsgRegisterInterchainAccount_NoLegibleAccount(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	msg := &types.MsgRegisterInterchainAccount{
		Owner:        testAddress,
		ConnectionId: "connection-0",
	}

	_, err := ms.RegisterInterchainAccount(ctx, msg)
	if err == nil {
		t.Error("expected error for missing Legible account")
	}
}

func TestMsgRegisterInterchainAccount_MaxAccountsReached(t *testing.T) {
	k, ctx, mockAuth, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Set max to 1
	_ = k.SetParams(ctx, types.Params{
		MaxRemoteAccountsPerDid: 1,
		RegistrationCooldown:    100,
	})

	mockAuth.accounts[testAddress] = &legibleauthtypes.LegibleAccount{
		Address: testAddress,
		Did:     testDID,
	}

	// First registration should succeed
	msg := &types.MsgRegisterInterchainAccount{
		Owner:        testAddress,
		ConnectionId: "connection-0",
	}
	_, err := ms.RegisterInterchainAccount(ctx, msg)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Second should fail
	msg2 := &types.MsgRegisterInterchainAccount{
		Owner:        testAddress,
		ConnectionId: "connection-1",
	}
	_, err = ms.RegisterInterchainAccount(ctx, msg2)
	if err == nil {
		t.Error("expected error for exceeding max accounts")
	}
}

func TestMsgRegisterInterchainAccount_DuplicateConnection(t *testing.T) {
	k, ctx, mockAuth, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	mockAuth.accounts[testAddress] = &legibleauthtypes.LegibleAccount{
		Address: testAddress,
		Did:     testDID,
	}

	msg := &types.MsgRegisterInterchainAccount{
		Owner:        testAddress,
		ConnectionId: "connection-0",
	}
	_, err := ms.RegisterInterchainAccount(ctx, msg)
	if err != nil {
		t.Fatalf("first registration failed: %v", err)
	}

	// Duplicate
	_, err = ms.RegisterInterchainAccount(ctx, msg)
	if err == nil {
		t.Error("expected error for duplicate connection")
	}
}

func TestMsgRegisterInterchainAccount_ICAControllerError(t *testing.T) {
	k, ctx, mockAuth, mockICA := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	mockAuth.accounts[testAddress] = &legibleauthtypes.LegibleAccount{
		Address: testAddress,
		Did:     testDID,
	}
	mockICA.failNext = true

	msg := &types.MsgRegisterInterchainAccount{
		Owner:        testAddress,
		ConnectionId: "connection-0",
	}

	_, err := ms.RegisterInterchainAccount(ctx, msg)
	if err == nil {
		t.Error("expected error from ICA controller failure")
	}
}

func TestMsgSendICATransaction_Success(t *testing.T) {
	k, ctx, mockAuth, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	mockAuth.accounts[testAddress] = &legibleauthtypes.LegibleAccount{
		Address: testAddress,
		Did:     testDID,
	}

	// First register an account and activate it
	acc := &types.RemoteAccount{
		ConnectionId: "connection-0",
		PortId:       "icacontroller-test",
		OwnerDid:     testDID,
		Active:       true, // pre-activated
	}
	_ = k.AddRemoteAccount(ctx, testDID, acc)

	msg := &types.MsgSendICATransaction{
		Owner:        testAddress,
		ConnectionId: "connection-0",
		MsgBytes:     []byte("test-tx-bytes"),
	}

	_, err := ms.SendICATransaction(ctx, msg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestMsgSendICATransaction_NoLegibleAccount(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	msg := &types.MsgSendICATransaction{
		Owner:        testAddress,
		ConnectionId: "connection-0",
		MsgBytes:     []byte("test"),
	}
	_, err := ms.SendICATransaction(ctx, msg)
	if err == nil {
		t.Error("expected error for missing Legible account")
	}
}

func TestMsgSendICATransaction_NotRegistered(t *testing.T) {
	k, ctx, mockAuth, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	mockAuth.accounts[testAddress] = &legibleauthtypes.LegibleAccount{
		Address: testAddress,
		Did:     testDID,
	}

	msg := &types.MsgSendICATransaction{
		Owner:        testAddress,
		ConnectionId: "connection-0",
		MsgBytes:     []byte("test"),
	}
	_, err := ms.SendICATransaction(ctx, msg)
	if err == nil {
		t.Error("expected error for unregistered ICA")
	}
}

func TestMsgSendICATransaction_NotActive(t *testing.T) {
	k, ctx, mockAuth, _ := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	mockAuth.accounts[testAddress] = &legibleauthtypes.LegibleAccount{
		Address: testAddress,
		Did:     testDID,
	}

	// Register but don't activate
	acc := &types.RemoteAccount{
		ConnectionId: "connection-0",
		OwnerDid:     testDID,
		Active:       false,
	}
	_ = k.AddRemoteAccount(ctx, testDID, acc)

	msg := &types.MsgSendICATransaction{
		Owner:        testAddress,
		ConnectionId: "connection-0",
		MsgBytes:     []byte("test"),
	}
	_, err := ms.SendICATransaction(ctx, msg)
	if err == nil {
		t.Error("expected error for inactive ICA")
	}
}

// ============================================================
// Query Server Tests
// ============================================================

func TestQueryInterchainAccounts(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	acc := &types.RemoteAccount{ConnectionId: "connection-0", OwnerDid: testDID}
	_ = k.AddRemoteAccount(ctx, testDID, acc)

	resp, err := qs.InterchainAccounts(ctx, &types.QueryInterchainAccountsRequest{Did: testDID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accounts) != 1 {
		t.Fatalf("expected 1 account, got %d", len(resp.Accounts))
	}
}

func TestQueryInterchainAccounts_Empty(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.InterchainAccounts(ctx, &types.QueryInterchainAccountsRequest{Did: testDID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Accounts) != 0 {
		t.Errorf("expected 0 accounts, got %d", len(resp.Accounts))
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.MaxRemoteAccountsPerDid != 10 {
		t.Errorf("expected MaxRemoteAccountsPerDid=10, got %d", resp.Params.MaxRemoteAccountsPerDid)
	}
}
