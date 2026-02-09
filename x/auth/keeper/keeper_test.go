package keeper_test

import (
	"context"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/auth/keeper"
	"github.com/legible-money/legible/x/auth/types"
)

// ---------- Mock Keepers ----------

type mockCosmosAccountKeeper struct{}

func (m mockCosmosAccountKeeper) GetAccount(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}
func (m mockCosmosAccountKeeper) SetAccount(_ context.Context, _ sdk.AccountI) {}
func (m mockCosmosAccountKeeper) NewAccountWithAddress(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}

type mockBankKeeper struct{}

func (m mockBankKeeper) GetBalance(_ context.Context, _ sdk.AccAddress, _ string) sdk.Coin {
	return sdk.NewCoin("ulgm", sdkmath.ZeroInt())
}
func (m mockBankKeeper) GetAllBalances(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins()
}

// ---------- Test Setup ----------

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

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
		storeKey,
		mockCosmosAccountKeeper{},
		mockBankKeeper{},
		"authority",
	)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Set default params
	if err := k.SetParams(ctx, types.DefaultParams()); err != nil {
		t.Fatalf("failed to set params: %v", err)
	}

	return k, ctx
}

const (
	testAddr1 = "lgm180376lgm92tk3h3dz9gdy4c5nt4uly6t2qxpgu"
	testAddr2 = "lgm1ur4uyehecajarczwttfj3sad9dstmecc6rf5a3"
	testDID1  = "did:lgm:0000000000000000000000000000000000000000000000000000000000000001"
	testDID2  = "did:lgm:0000000000000000000000000000000000000000000000000000000000000002"
	testPubKey1 = "abcdef0123456789abcdef0123456789abcdef0123456789abcdef0123456789"
	testPubKey2 = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
)

// ---------- RegisterAccount Tests ----------

func TestRegisterAccount_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	resp, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "opkey1hash",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil {
		t.Fatal("expected non-nil response")
	}

	// Verify account stored
	account, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("account not found after registration")
	}
	if account.Did != testDID1 {
		t.Errorf("expected DID %s, got %s", testDID1, account.Did)
	}
	if account.PublicKey != testPubKey1 {
		t.Errorf("expected pubkey %s, got %s", testPubKey1, account.PublicKey)
	}
	if account.AccountType != "agent" {
		t.Errorf("expected type agent, got %s", account.AccountType)
	}
	if account.OperationalKeyHash != "opkey1hash" {
		t.Errorf("expected opkey hash opkey1hash, got %s", account.OperationalKeyHash)
	}
	if account.OperationalKeyVersion != 1 {
		t.Errorf("expected version 1, got %d", account.OperationalKeyVersion)
	}
	if account.ReputationScore != 500000 {
		t.Errorf("expected reputation 500000, got %d", account.ReputationScore)
	}
	if !account.Flags.CanSubmitClaims {
		t.Error("expected CanSubmitClaims true")
	}

	// Verify DID mapping stored
	mapping, found := k.GetDIDMapping(ctx, testDID1)
	if !found {
		t.Fatal("DID mapping not found after registration")
	}
	if mapping.Bech32 != testAddr1 {
		t.Errorf("expected bech32 %s, got %s", testAddr1, mapping.Bech32)
	}
	if mapping.PubKey != testPubKey1 {
		t.Errorf("expected pubkey %s, got %s", testPubKey1, mapping.PubKey)
	}
}

func TestRegisterAccount_DuplicateAddress(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Register first account
	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	// Try to register again with same address
	_, err = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID2,
		PublicKey:   testPubKey2,
		AccountType: "human",
	})
	if err == nil {
		t.Fatal("expected error for duplicate address")
	}
}

func TestRegisterAccount_DuplicateDID(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Register first account
	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	if err != nil {
		t.Fatalf("unexpected error on first registration: %v", err)
	}

	// Try to register with same DID
	_, err = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr2,
		Did:         testDID1,
		PublicKey:   testPubKey2,
		AccountType: "human",
	})
	if err == nil {
		t.Fatal("expected error for duplicate DID")
	}
}

// ---------- RotateKey Tests ----------

func TestRotateKey_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Register account first
	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "oldkey",
	})
	if err != nil {
		t.Fatalf("failed to register account: %v", err)
	}

	// Rotate operational key
	_, err = ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "operational",
		NewKeyHash: "newkey",
		Signature:  "sig",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify key was rotated
	account, _ := k.GetAccount(ctx, testAddr1)
	if account.OperationalKeyHash != "newkey" {
		t.Errorf("expected newkey, got %s", account.OperationalKeyHash)
	}
	if account.OperationalKeyVersion != 2 {
		t.Errorf("expected version 2, got %d", account.OperationalKeyVersion)
	}
}

func TestRotateKey_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "operational",
		NewKeyHash: "newkey",
		Signature:  "sig",
	})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

func TestRotateKey_Cooldown(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Register account
	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	// First rotation
	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "operational",
		NewKeyHash: "key2",
		Signature:  "sig",
	})
	if err != nil {
		t.Fatalf("first rotation failed: %v", err)
	}

	// Immediate second rotation should fail (cooldown)
	_, err = ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "operational",
		NewKeyHash: "key3",
		Signature:  "sig",
	})
	if err == nil {
		t.Fatal("expected cooldown error")
	}

	// After cooldown passes
	params := k.GetParams(ctx)
	newCtx := ctx.WithBlockHeight(int64(100 + params.KeyRotationCooldown + 1))
	_, err = ms.RotateKey(newCtx, &types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "operational",
		NewKeyHash: "key3",
		Signature:  "sig",
	})
	if err != nil {
		t.Fatalf("rotation after cooldown should succeed: %v", err)
	}
}

func TestRotateKey_FrozenAccount(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Register and freeze account
	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	account, _ := k.GetAccount(ctx, testAddr1)
	account.Flags.Frozen = true
	k.SetAccount(ctx, account)

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "operational",
		NewKeyHash: "newkey",
		Signature:  "sig",
	})
	if err == nil {
		t.Fatal("expected error for frozen account")
	}
}

func TestRotateKey_InvalidKeyType(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "session",
		NewKeyHash: "newkey",
		Signature:  "sig",
	})
	if err == nil {
		t.Fatal("expected error for session key type")
	}
}

// ---------- CreateSession Tests ----------

func TestCreateSession_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	resp, err := ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "session1",
		Capabilities: types.SessionCapabilities{
			CanTransfer:     true,
			CanSubmitClaims: true,
		},
		Duration: 1000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ExpiresAtBlock != 1100 { // 100 (block height) + 1000
		t.Errorf("expected expiry 1100, got %d", resp.ExpiresAtBlock)
	}

	// Verify session stored
	session, found := k.GetSessionKey(ctx, testAddr1, "session1")
	if !found {
		t.Fatal("session not found")
	}
	if !session.Capabilities.CanTransfer {
		t.Error("expected CanTransfer true")
	}
	if !session.Capabilities.CanSubmitClaims {
		t.Error("expected CanSubmitClaims true")
	}

	// Verify account updated
	account, _ := k.GetAccount(ctx, testAddr1)
	if account.SessionKeyCount != 1 {
		t.Errorf("expected session count 1, got %d", account.SessionKeyCount)
	}
}

func TestCreateSession_MaxSessionKeys(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	params := k.GetParams(ctx)

	// Create max sessions
	for i := uint32(0); i < params.MaxSessionKeys; i++ {
		_, err := ms.CreateSession(ctx, &types.MsgCreateSession{
			Owner:          testAddr1,
			SessionKeyHash: string(rune('a' + int(i))),
			Capabilities:   types.SessionCapabilities{},
			Duration:       1000,
		})
		if err != nil {
			t.Fatalf("unexpected error creating session %d: %v", i, err)
		}
	}

	// Next should fail
	_, err := ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "overflow",
		Capabilities:   types.SessionCapabilities{},
		Duration:       1000,
	})
	if err == nil {
		t.Fatal("expected error for exceeding max session keys")
	}
}

func TestCreateSession_DurationExceeded(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	params := k.GetParams(ctx)

	_, err := ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "session1",
		Duration:       params.MaxSessionDuration + 1,
	})
	if err == nil {
		t.Fatal("expected error for exceeding max duration")
	}
}

// ---------- RevokeSession Tests ----------

func TestRevokeSession_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "session1",
		Duration:       1000,
	})

	_, err := ms.RevokeSession(ctx, &types.MsgRevokeSession{
		Owner:          testAddr1,
		SessionKeyHash: "session1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify session deleted
	_, found := k.GetSessionKey(ctx, testAddr1, "session1")
	if found {
		t.Fatal("session should be deleted after revocation")
	}

	// Verify account updated
	account, _ := k.GetAccount(ctx, testAddr1)
	if account.SessionKeyCount != 0 {
		t.Errorf("expected session count 0, got %d", account.SessionKeyCount)
	}
}

func TestRevokeSession_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.RevokeSession(ctx, &types.MsgRevokeSession{
		Owner:          testAddr1,
		SessionKeyHash: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for non-existent session")
	}
}

// ---------- Query Tests ----------

func TestQueryAccount(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	resp, err := qs.Account(ctx, &types.QueryAccountRequest{Address: testAddr1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Account.Did != testDID1 {
		t.Errorf("expected DID %s, got %s", testDID1, resp.Account.Did)
	}
}

func TestQueryAccountByDID(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "human",
	})

	resp, err := qs.AccountByDID(ctx, &types.QueryAccountByDIDRequest{Did: testDID1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Account.Address != testAddr1 {
		t.Errorf("expected address %s, got %s", testAddr1, resp.Account.Address)
	}
	if resp.Account.AccountType != "human" {
		t.Errorf("expected type human, got %s", resp.Account.AccountType)
	}
}

func TestQueryAccount_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	_, err := qs.Account(ctx, &types.QueryAccountRequest{Address: testAddr1})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

func TestQuerySessionKeys(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)
	qs := keeper.NewQueryServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s1",
		Duration:       1000,
	})
	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s2",
		Duration:       2000,
	})

	resp, err := qs.SessionKeys(ctx, &types.QuerySessionKeysRequest{Owner: testAddr1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.SessionKeys) != 2 {
		t.Errorf("expected 2 session keys, got %d", len(resp.SessionKeys))
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)

	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.MaxSessionKeys != 5 {
		t.Errorf("expected max session keys 5, got %d", resp.Params.MaxSessionKeys)
	}
	if resp.Params.MaxSessionDuration != 34272 {
		t.Errorf("expected max session duration 34272, got %d", resp.Params.MaxSessionDuration)
	}
	if resp.Params.KeyRotationCooldown != 111 {
		t.Errorf("expected key rotation cooldown 111, got %d", resp.Params.KeyRotationCooldown)
	}
}

// ---------- DID Lookup Tests ----------

func TestGetAccountByDID(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	// Lookup by DID
	account, found := k.GetAccountByDID(ctx, testDID1)
	if !found {
		t.Fatal("expected to find account by DID")
	}
	if account.Address != testAddr1 {
		t.Errorf("expected address %s, got %s", testAddr1, account.Address)
	}

	// Lookup by address
	addr, found := k.GetAddressForDID(ctx, testDID1)
	if !found {
		t.Fatal("expected to find address for DID")
	}
	if addr != testAddr1 {
		t.Errorf("expected %s, got %s", testAddr1, addr)
	}
}

// ---------- Genesis Tests ----------

func TestInitExportGenesis(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	// Register two accounts
	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr2,
		Did:         testDID2,
		PublicKey:   testPubKey2,
		AccountType: "human",
	})

	// Create a session
	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s1",
		Duration:       1000,
	})

	// Export genesis
	gs := k.ExportGenesis(ctx)
	if len(gs.Accounts) != 2 {
		t.Errorf("expected 2 accounts in genesis, got %d", len(gs.Accounts))
	}
	if len(gs.DIDMappings) != 2 {
		t.Errorf("expected 2 DID mappings in genesis, got %d", len(gs.DIDMappings))
	}
	if len(gs.SessionKeys) != 1 {
		t.Errorf("expected 1 session key in genesis, got %d", len(gs.SessionKeys))
	}

	// Validate genesis
	if err := gs.Validate(); err != nil {
		t.Fatalf("genesis validation failed: %v", err)
	}
}

// ---------- ValidateBasic Tests ----------

func TestMsgRegisterAccount_ValidateBasic(t *testing.T) {
	// Valid
	msg := types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	// Invalid account type
	msg.AccountType = "invalid"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for invalid account type")
	}

	// Empty public key
	msg.AccountType = "agent"
	msg.PublicKey = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty public key")
	}

	// Wrong length public key
	msg.PublicKey = "short"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for short public key")
	}

	// Invalid DID
	msg.PublicKey = testPubKey1
	msg.Did = "invalid"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for invalid DID")
	}
}

func TestValidateDID(t *testing.T) {
	// Valid
	if err := types.ValidateDID(testDID1); err != nil {
		t.Errorf("expected valid DID: %v", err)
	}

	// Missing prefix
	if err := types.ValidateDID("0000000000000000000000000000000000000000000000000000000000000001"); err == nil {
		t.Error("expected error for missing did:lgm: prefix")
	}

	// Wrong suffix length
	if err := types.ValidateDID("did:lgm:short"); err == nil {
		t.Error("expected error for short suffix")
	}
}

// ---------- Phase 2: OperationalPublicKey + Ed25519 Sync Tests ----------

func TestRegisterAccount_SetsOperationalPublicKey(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "opkey1hash",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, found := k.GetAccount(ctx, testAddr1)
	if !found {
		t.Fatal("account not found")
	}

	// Identity key should be initial operational key
	if account.OperationalPublicKey != testPubKey1 {
		t.Errorf("expected OperationalPublicKey %s, got %s", testPubKey1, account.OperationalPublicKey)
	}
}

func TestRotateKey_StoresNewPublicKey(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "oldkey",
	})

	// Rotate with a new public key
	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:       testAddr1,
		KeyType:      "operational",
		NewKeyHash:   "newkey",
		NewPublicKey: testPubKey2,
		Signature:    "sig",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.OperationalPublicKey != testPubKey2 {
		t.Errorf("expected OperationalPublicKey %s, got %s", testPubKey2, account.OperationalPublicKey)
	}
	if account.OperationalKeyHash != "newkey" {
		t.Errorf("expected key hash newkey, got %s", account.OperationalKeyHash)
	}
	if account.OperationalKeyVersion != 2 {
		t.Errorf("expected version 2, got %d", account.OperationalKeyVersion)
	}
}

func TestRotateKey_WithoutNewPublicKey_PreservesExisting(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "oldkey",
	})

	// Rotate without providing NewPublicKey
	_, err := ms.RotateKey(ctx, &types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "operational",
		NewKeyHash: "newkey",
		Signature:  "sig",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	// OperationalPublicKey should remain the identity key
	if account.OperationalPublicKey != testPubKey1 {
		t.Errorf("expected OperationalPublicKey preserved as %s, got %s", testPubKey1, account.OperationalPublicKey)
	}
}

// ---------- Phase 2-3: Session Key PublicKey Tests ----------

func TestCreateSession_StoresSessionPublicKey(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	sessionPubKey := "fedcba9876543210fedcba9876543210fedcba9876543210fedcba9876543210"
	_, err := ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:            testAddr1,
		SessionKeyHash:   "session1",
		SessionPublicKey: sessionPubKey,
		Capabilities: types.SessionCapabilities{
			CanTransfer:     true,
			CanSubmitClaims: true,
		},
		Duration: 1000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	session, found := k.GetSessionKey(ctx, testAddr1, "session1")
	if !found {
		t.Fatal("session not found")
	}
	if session.PublicKey != sessionPubKey {
		t.Errorf("expected session PublicKey %s, got %s", sessionPubKey, session.PublicKey)
	}
}

func TestGetSessionKeysForOwner_ReturnsAll(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	// Create 3 sessions with different pubkeys
	for i := 0; i < 3; i++ {
		hash := string(rune('a' + i))
		_, err := ms.CreateSession(ctx, &types.MsgCreateSession{
			Owner:            testAddr1,
			SessionKeyHash:   hash,
			SessionPublicKey: "deadbeef" + hash + "0000000000000000000000000000000000000000000000000000000",
			Capabilities:     types.SessionCapabilities{CanTransfer: i%2 == 0},
			Duration:         1000,
		})
		if err != nil {
			t.Fatalf("session %d: %v", i, err)
		}
	}

	sessions := k.GetSessionKeysForOwner(ctx, testAddr1)
	if len(sessions) != 3 {
		t.Errorf("expected 3 sessions, got %d", len(sessions))
	}
}

func TestGetSessionKeysForOwner_EmptyForUnknownOwner(t *testing.T) {
	k, ctx := setupKeeper(t)
	sessions := k.GetSessionKeysForOwner(ctx, testAddr2)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions for unknown owner, got %d", len(sessions))
	}
}

// ---------- Phase 3: Frozen Account Blocking Tests ----------

func TestCreateSession_FrozenAccount(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	// Freeze account
	account, _ := k.GetAccount(ctx, testAddr1)
	account.Flags.Frozen = true
	k.SetAccount(ctx, account)

	_, err := ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "session1",
		Duration:       1000,
	})
	if err == nil {
		t.Fatal("expected error for frozen account")
	}
}

// ---------- Phase 4: DID Resolution Tests ----------

func TestGetAddressForDID_NotRegistered(t *testing.T) {
	k, ctx := setupKeeper(t)

	_, found := k.GetAddressForDID(ctx, testDID1)
	if found {
		t.Fatal("expected DID not found for unregistered DID")
	}
}

func TestGetAccountByDID_ReturnsFullAccount(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:             testAddr1,
		Did:                testDID1,
		PublicKey:          testPubKey1,
		AccountType:        "agent",
		OperationalKeyHash: "opkey1hash",
	})

	account, found := k.GetAccountByDID(ctx, testDID1)
	if !found {
		t.Fatal("expected to find account by DID")
	}
	if account.Address != testAddr1 {
		t.Errorf("expected address %s, got %s", testAddr1, account.Address)
	}
	if account.OperationalPublicKey != testPubKey1 {
		t.Errorf("expected OperationalPublicKey %s, got %s", testPubKey1, account.OperationalPublicKey)
	}
	if account.OperationalKeyHash != "opkey1hash" {
		t.Errorf("expected opkey hash opkey1hash, got %s", account.OperationalKeyHash)
	}
}

// ---------- ValidateBasic Phase 2-4 Tests ----------

func TestMsgRotateKey_ValidateBasic(t *testing.T) {
	// Valid with NewPublicKey
	msg := types.MsgRotateKey{
		Sender:       testAddr1,
		KeyType:      "operational",
		NewKeyHash:   "newhash",
		NewPublicKey: testPubKey2,
		Signature:    "sig",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	// Valid without NewPublicKey (backward compatible)
	msg2 := types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "operational",
		NewKeyHash: "newhash",
		Signature:  "sig",
	}
	if err := msg2.ValidateBasic(); err != nil {
		t.Errorf("expected valid without new pubkey, got error: %v", err)
	}

	// Invalid sender
	msg.Sender = "invalid"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for invalid sender")
	}
}

func TestMsgCreateSession_ValidateBasic(t *testing.T) {
	msg := types.MsgCreateSession{
		Owner:            testAddr1,
		SessionKeyHash:   "s1",
		SessionPublicKey: testPubKey2,
		Duration:         1000,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	// Invalid owner
	msg.Owner = "bad"
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for invalid owner")
	}

	// Zero duration
	msg.Owner = testAddr1
	msg.Duration = 0
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for zero duration")
	}
}

func TestMsgRevokeSession_ValidateBasic(t *testing.T) {
	msg := types.MsgRevokeSession{
		Owner:          testAddr1,
		SessionKeyHash: "s1",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	// Empty key hash
	msg.SessionKeyHash = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty session key hash")
	}
}

// ---------- LastActiveBlock Update Tests ----------

func TestRegisterAccount_SetsCreatedAndLastActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.CreatedAtBlock != 100 {
		t.Errorf("expected CreatedAtBlock 100, got %d", account.CreatedAtBlock)
	}
	if account.LastActiveBlock != 100 {
		t.Errorf("expected LastActiveBlock 100, got %d", account.LastActiveBlock)
	}
}

func TestRotateKey_UpdatesLastActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	// Advance block height past cooldown
	params := k.GetParams(ctx)
	advancedCtx := ctx.WithBlockHeight(int64(100 + params.KeyRotationCooldown + 1))

	_, err := ms.RotateKey(advancedCtx, &types.MsgRotateKey{
		Sender:     testAddr1,
		KeyType:    "operational",
		NewKeyHash: "newkey",
		Signature:  "sig",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(advancedCtx, testAddr1)
	expected := uint64(100 + params.KeyRotationCooldown + 1)
	if account.LastActiveBlock != expected {
		t.Errorf("expected LastActiveBlock %d, got %d", expected, account.LastActiveBlock)
	}
}

func TestCreateSession_UpdatesLastActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	advancedCtx := ctx.WithBlockHeight(200)

	_, err := ms.CreateSession(advancedCtx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s1",
		Duration:       1000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(advancedCtx, testAddr1)
	if account.LastActiveBlock != 200 {
		t.Errorf("expected LastActiveBlock 200, got %d", account.LastActiveBlock)
	}
}

func TestRevokeSession_UpdatesLastActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})
	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s1",
		Duration:       1000,
	})

	advancedCtx := ctx.WithBlockHeight(250)
	_, err := ms.RevokeSession(advancedCtx, &types.MsgRevokeSession{
		Owner:          testAddr1,
		SessionKeyHash: "s1",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(advancedCtx, testAddr1)
	if account.LastActiveBlock != 250 {
		t.Errorf("expected LastActiveBlock 250, got %d", account.LastActiveBlock)
	}
}

// ---------- FreezeAccount Tests ----------

func TestFreezeAccount_SelfFreeze(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "compromised key",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	if !account.Flags.Frozen {
		t.Fatal("expected account to be frozen")
	}
}

func TestFreezeAccount_AuthorityFreeze(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  "authority",
		Address: testAddr1,
		Reason:  "malicious activity",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	if !account.Flags.Frozen {
		t.Fatal("expected account to be frozen by authority")
	}
}

func TestFreezeAccount_UnauthorizedThirdParty(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr2,
		Address: testAddr1,
		Reason:  "attack",
	})
	if err == nil {
		t.Fatal("expected error for unauthorized freeze")
	}
}

func TestFreezeAccount_AlreadyFrozen(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	// First freeze
	_, _ = ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "first freeze",
	})

	// Second freeze should fail
	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "second freeze",
	})
	if err == nil {
		t.Fatal("expected error for already frozen account")
	}
}

func TestFreezeAccount_NotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test",
	})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

// ---------- UnfreezeAccount Tests ----------

func TestUnfreezeAccount_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	// Freeze first
	_, _ = ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test",
	})

	// Unfreeze by authority
	_, err := ms.UnfreezeAccount(ctx, &types.MsgUnfreezeAccount{
		Authority: "authority",
		Address:   testAddr1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.Flags.Frozen {
		t.Fatal("expected account to be unfrozen")
	}
}

func TestUnfreezeAccount_NonAuthority(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, _ = ms.FreezeAccount(ctx, &types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test",
	})

	// Non-authority should fail
	_, err := ms.UnfreezeAccount(ctx, &types.MsgUnfreezeAccount{
		Authority: testAddr2,
		Address:   testAddr1,
	})
	if err == nil {
		t.Fatal("expected error for non-authority unfreeze")
	}
}

func TestUnfreezeAccount_NotFrozen(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.UnfreezeAccount(ctx, &types.MsgUnfreezeAccount{
		Authority: "authority",
		Address:   testAddr1,
	})
	if err == nil {
		t.Fatal("expected error for unfreezing non-frozen account")
	}
}

// ---------- SetRecoveryConfig Tests ----------

func TestSetRecoveryConfig_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   2,
		TotalShards: 3,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
			{Type: "agent", Identifier: "lgm1guardian2xxxxxxxxxxxxxxxxxxxxxxxx", ShardIndex: 1, CanInitiateRecovery: false},
			{Type: "agent", Identifier: "lgm1guardian3xxxxxxxxxxxxxxxxxxxxxxxx", ShardIndex: 2, CanInitiateRecovery: false},
		},
		RecoveryDelayBlocks:   500,
		ChallengePeriodBlocks: 250,
	}

	_, err := ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	stored, found := k.GetRecoveryConfig(ctx, testAddr1)
	if !found {
		t.Fatal("recovery config not found")
	}
	if stored.Threshold != 2 {
		t.Errorf("expected threshold 2, got %d", stored.Threshold)
	}
	if stored.TotalShards != 3 {
		t.Errorf("expected total shards 3, got %d", stored.TotalShards)
	}
	if len(stored.ShardHolders) != 3 {
		t.Errorf("expected 3 shard holders, got %d", len(stored.ShardHolders))
	}
}

func TestSetRecoveryConfig_AccountNotFound(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, err := ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: types.RecoveryConfig{Threshold: 1, TotalShards: 1},
	})
	if err == nil {
		t.Fatal("expected error for non-existent account")
	}
}

// ---------- InitiateRecovery Tests ----------

func TestInitiateRecovery_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   2,
		TotalShards: 3,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
			{Type: "agent", Identifier: "lgm1guardian2xxxxxxxxxxxxxxxxxxxxxxxx", ShardIndex: 1, CanInitiateRecovery: false},
			{Type: "agent", Identifier: "lgm1guardian3xxxxxxxxxxxxxxxxxxxxxxxx", ShardIndex: 2, CanInitiateRecovery: false},
		},
		RecoveryDelayBlocks:   500,
		ChallengePeriodBlocks: 250,
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, err := ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify request stored
	req, found := k.GetRecoveryRequest(ctx, testAddr1)
	if !found {
		t.Fatal("recovery request not found")
	}
	if req.Status != "pending" {
		t.Errorf("expected status pending, got %s", req.Status)
	}
	if req.InitiatedBy != testAddr2 {
		t.Errorf("expected initiated by %s, got %s", testAddr2, req.InitiatedBy)
	}
	if req.NewOperationalKey != testPubKey2 {
		t.Errorf("expected new key %s, got %s", testPubKey2, req.NewOperationalKey)
	}
	if req.ShardsRequired != 2 {
		t.Errorf("expected shards required 2, got %d", req.ShardsRequired)
	}
	if req.DelayExpiresAt != 600 { // 100 + 500
		t.Errorf("expected delay expires at 600, got %d", req.DelayExpiresAt)
	}
	if req.ChallengeExpiresAt != 850 { // 100 + 500 + 250
		t.Errorf("expected challenge expires at 850, got %d", req.ChallengeExpiresAt)
	}

	// Verify InRecovery flag
	account, _ := k.GetAccount(ctx, testAddr1)
	if !account.Flags.InRecovery {
		t.Fatal("expected InRecovery flag to be set")
	}
}

func TestInitiateRecovery_UnauthorizedInitiator(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   2,
		TotalShards: 3,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: false}, // cannot initiate
		},
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, err := ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})
	if err == nil {
		t.Fatal("expected error for unauthorized initiator")
	}
}

func TestInitiateRecovery_AlreadyActive(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   1,
		TotalShards: 1,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
		},
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	// First initiation
	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})

	// Second should fail
	_, err := ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})
	if err == nil {
		t.Fatal("expected error for already active recovery")
	}
}

func TestInitiateRecovery_NoConfig(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, err := ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})
	if err == nil {
		t.Fatal("expected error for missing recovery config")
	}
}

// ---------- SubmitRecoveryShard Tests ----------

func TestSubmitRecoveryShard_Success(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   2,
		TotalShards: 3,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
			{Type: "agent", Identifier: "lgm1guardian2xxxxxxxxxxxxxxxxxxxxxxxx", ShardIndex: 1, CanInitiateRecovery: false},
			{Type: "agent", Identifier: "lgm1guardian3xxxxxxxxxxxxxxxxxxxxxxxx", ShardIndex: 2, CanInitiateRecovery: false},
		},
		RecoveryDelayBlocks:   500,
		ChallengePeriodBlocks: 250,
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})

	// Submit first shard
	_, err := ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
		ShardIndex:     0,
	})
	if err != nil {
		t.Fatalf("unexpected error submitting shard 0: %v", err)
	}

	req, _ := k.GetRecoveryRequest(ctx, testAddr1)
	if req.Status != "pending" {
		t.Errorf("expected status pending after 1 shard, got %s", req.Status)
	}
	if len(req.ShardsProvided) != 1 {
		t.Errorf("expected 1 shard provided, got %d", len(req.ShardsProvided))
	}

	// Submit second shard to meet threshold
	_, err = ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         "lgm1guardian2xxxxxxxxxxxxxxxxxxxxxxxx",
		AccountAddress: testAddr1,
		ShardIndex:     1,
	})
	if err != nil {
		t.Fatalf("unexpected error submitting shard 1: %v", err)
	}

	req, _ = k.GetRecoveryRequest(ctx, testAddr1)
	if req.Status != "delayed" {
		t.Errorf("expected status delayed after 2 shards (threshold met), got %s", req.Status)
	}
}

func TestSubmitRecoveryShard_DuplicateShard(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   2,
		TotalShards: 2,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
			{Type: "agent", Identifier: "lgm1guardian2xxxxxxxxxxxxxxxxxxxxxxxx", ShardIndex: 1, CanInitiateRecovery: false},
		},
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})

	// Submit shard 0
	_, _ = ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
		ShardIndex:     0,
	})

	// Submit shard 0 again — should fail
	_, err := ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
		ShardIndex:     0,
	})
	if err == nil {
		t.Fatal("expected error for duplicate shard submission")
	}
}

func TestSubmitRecoveryShard_WrongSender(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   1,
		TotalShards: 2,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
			{Type: "agent", Identifier: "lgm1guardian2xxxxxxxxxxxxxxxxxxxxxxxx", ShardIndex: 1, CanInitiateRecovery: false},
		},
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})

	// Wrong sender trying to submit shard 1
	_, err := ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         testAddr2, // testAddr2 holds shard 0, not 1
		AccountAddress: testAddr1,
		ShardIndex:     1,
	})
	if err == nil {
		t.Fatal("expected error for wrong shard holder")
	}
}

// ---------- ChallengeRecovery Tests ----------

func TestChallengeRecovery_OwnerCancels(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   1,
		TotalShards: 1,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
		},
		RecoveryDelayBlocks:   100,
		ChallengePeriodBlocks: 100,
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})

	// Submit shard to move to delayed
	_, _ = ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
		ShardIndex:     0,
	})

	// Advance past delay to make it challengeable
	challengeCtx := ctx.WithBlockHeight(201) // > 100 + 100 delay
	k.ProcessRecoveryTimeouts(challengeCtx)

	req, _ := k.GetRecoveryRequest(challengeCtx, testAddr1)
	if req.Status != "challengeable" {
		t.Fatalf("expected challengeable, got %s", req.Status)
	}

	// Owner challenges
	_, err := ms.ChallengeRecovery(challengeCtx, &types.MsgChallengeRecovery{
		Sender:         testAddr1,
		AccountAddress: testAddr1,
		Reason:         "I still have my keys",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	req, _ = k.GetRecoveryRequest(challengeCtx, testAddr1)
	if req.Status != "cancelled" {
		t.Errorf("expected status cancelled, got %s", req.Status)
	}
	if req.ChallengerAddress != testAddr1 {
		t.Errorf("expected challenger %s, got %s", testAddr1, req.ChallengerAddress)
	}

	// InRecovery flag should be cleared
	account, _ := k.GetAccount(challengeCtx, testAddr1)
	if account.Flags.InRecovery {
		t.Fatal("expected InRecovery flag cleared after challenge")
	}
}

func TestChallengeRecovery_NotChallengeable(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   1,
		TotalShards: 1,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
		},
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})

	// Try to challenge while still "pending" (not challengeable)
	_, err := ms.ChallengeRecovery(ctx, &types.MsgChallengeRecovery{
		Sender:         testAddr1,
		AccountAddress: testAddr1,
		Reason:         "too early",
	})
	if err == nil {
		t.Fatal("expected error for challenging non-challengeable recovery")
	}
}

func TestChallengeRecovery_ThirdPartyBlocked(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   1,
		TotalShards: 1,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
		},
		RecoveryDelayBlocks:   10,
		ChallengePeriodBlocks: 10,
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})
	_, _ = ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
		ShardIndex:     0,
	})

	// Advance to challengeable
	challengeCtx := ctx.WithBlockHeight(111)
	k.ProcessRecoveryTimeouts(challengeCtx)

	// Third party (not owner, not authority) tries to challenge
	_, err := ms.ChallengeRecovery(challengeCtx, &types.MsgChallengeRecovery{
		Sender:         testAddr2, // testAddr2 is not the account owner
		AccountAddress: testAddr1,
		Reason:         "I'm not the owner",
	})
	if err == nil {
		t.Fatal("expected error for third party challenge")
	}
}

// ---------- ExecuteRecovery Tests ----------

func TestExecuteRecovery_FullLifecycle(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	// Create some sessions that should be revoked on recovery
	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s1",
		Duration:       10000,
	})
	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s2",
		Duration:       10000,
	})

	config := types.RecoveryConfig{
		Threshold:   1,
		TotalShards: 1,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
		},
		RecoveryDelayBlocks:   10,
		ChallengePeriodBlocks: 10,
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	// Initiate
	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})

	// Submit shard → moves to "delayed"
	_, _ = ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
		ShardIndex:     0,
	})

	// Advance past delay → "challengeable"
	delayCtx := ctx.WithBlockHeight(111)
	k.ProcessRecoveryTimeouts(delayCtx)
	req, _ := k.GetRecoveryRequest(delayCtx, testAddr1)
	if req.Status != "challengeable" {
		t.Fatalf("expected challengeable, got %s", req.Status)
	}

	// Advance past challenge → "executable"
	execCtx := ctx.WithBlockHeight(121)
	k.ProcessRecoveryTimeouts(execCtx)
	req, _ = k.GetRecoveryRequest(execCtx, testAddr1)
	if req.Status != "executable" {
		t.Fatalf("expected executable, got %s", req.Status)
	}

	// Execute recovery
	_, err := ms.ExecuteRecovery(execCtx, &types.MsgExecuteRecovery{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify key rotated
	account, _ := k.GetAccount(execCtx, testAddr1)
	if account.OperationalPublicKey != testPubKey2 {
		t.Errorf("expected new operational key %s, got %s", testPubKey2, account.OperationalPublicKey)
	}
	if account.OperationalKeyVersion != 2 {
		t.Errorf("expected key version 2, got %d", account.OperationalKeyVersion)
	}
	if account.Flags.InRecovery {
		t.Fatal("expected InRecovery flag cleared")
	}

	// Verify all sessions revoked
	if account.SessionKeyCount != 0 {
		t.Errorf("expected session count 0 after recovery, got %d", account.SessionKeyCount)
	}
	sessions := k.GetSessionKeysForOwner(execCtx, testAddr1)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions after recovery, got %d", len(sessions))
	}

	// Verify request completed
	req, _ = k.GetRecoveryRequest(execCtx, testAddr1)
	if req.Status != "completed" {
		t.Errorf("expected status completed, got %s", req.Status)
	}
}

func TestExecuteRecovery_NotExecutable(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   1,
		TotalShards: 1,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
		},
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})

	// Try to execute while still "pending"
	_, err := ms.ExecuteRecovery(ctx, &types.MsgExecuteRecovery{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
	})
	if err == nil {
		t.Fatal("expected error for executing non-executable recovery")
	}
}

// ---------- BeginBlock Session Cleanup Tests ----------

func TestCleanupExpiredSessions(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	// Create sessions with different expirations
	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "short",
		Duration:       50, // expires at block 150
	})
	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "long",
		Duration:       1000, // expires at block 1100
	})

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.SessionKeyCount != 2 {
		t.Fatalf("expected 2 sessions, got %d", account.SessionKeyCount)
	}

	// Advance past short session expiry
	expiredCtx := ctx.WithBlockHeight(151)
	k.CleanupExpiredSessions(expiredCtx)

	// Short session should be gone
	_, found := k.GetSessionKey(expiredCtx, testAddr1, "short")
	if found {
		t.Fatal("expected short session to be cleaned up")
	}

	// Long session should remain
	_, found = k.GetSessionKey(expiredCtx, testAddr1, "long")
	if !found {
		t.Fatal("expected long session to still exist")
	}

	// Session count should be decremented
	account, _ = k.GetAccount(expiredCtx, testAddr1)
	if account.SessionKeyCount != 1 {
		t.Errorf("expected session count 1 after cleanup, got %d", account.SessionKeyCount)
	}
}

func TestCleanupExpiredSessions_AllExpired(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s1",
		Duration:       10,
	})
	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s2",
		Duration:       20,
	})

	// Advance past all expirations
	expiredCtx := ctx.WithBlockHeight(200)
	k.CleanupExpiredSessions(expiredCtx)

	account, _ := k.GetAccount(expiredCtx, testAddr1)
	if account.SessionKeyCount != 0 {
		t.Errorf("expected session count 0, got %d", account.SessionKeyCount)
	}

	sessions := k.GetSessionKeysForOwner(expiredCtx, testAddr1)
	if len(sessions) != 0 {
		t.Errorf("expected 0 sessions, got %d", len(sessions))
	}
}

func TestCleanupExpiredSessions_NoneExpired(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	_, _ = ms.CreateSession(ctx, &types.MsgCreateSession{
		Owner:          testAddr1,
		SessionKeyHash: "s1",
		Duration:       1000,
	})

	// Don't advance — nothing expired
	k.CleanupExpiredSessions(ctx)

	account, _ := k.GetAccount(ctx, testAddr1)
	if account.SessionKeyCount != 1 {
		t.Errorf("expected session count 1 (no cleanup), got %d", account.SessionKeyCount)
	}
}

// ---------- Recovery Timeout Processing Tests ----------

func TestProcessRecoveryTimeouts_DelayedToChallengeable(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   1,
		TotalShards: 1,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
		},
		RecoveryDelayBlocks:   50,
		ChallengePeriodBlocks: 50,
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})
	_, _ = ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
		ShardIndex:     0,
	})

	// Verify delayed
	req, _ := k.GetRecoveryRequest(ctx, testAddr1)
	if req.Status != "delayed" {
		t.Fatalf("expected delayed, got %s", req.Status)
	}

	// Process at block before delay expires — no change
	beforeCtx := ctx.WithBlockHeight(149)
	k.ProcessRecoveryTimeouts(beforeCtx)
	req, _ = k.GetRecoveryRequest(beforeCtx, testAddr1)
	if req.Status != "delayed" {
		t.Errorf("expected still delayed before expiry, got %s", req.Status)
	}

	// Process at block after delay expires
	afterCtx := ctx.WithBlockHeight(151)
	k.ProcessRecoveryTimeouts(afterCtx)
	req, _ = k.GetRecoveryRequest(afterCtx, testAddr1)
	if req.Status != "challengeable" {
		t.Errorf("expected challengeable after delay, got %s", req.Status)
	}
}

func TestProcessRecoveryTimeouts_ChallengeableToExecutable(t *testing.T) {
	k, ctx := setupKeeper(t)
	ms := keeper.NewMsgServerImpl(k)

	_, _ = ms.RegisterAccount(ctx, &types.MsgRegisterAccount{
		Sender:      testAddr1,
		Did:         testDID1,
		PublicKey:   testPubKey1,
		AccountType: "agent",
	})

	config := types.RecoveryConfig{
		Threshold:   1,
		TotalShards: 1,
		ShardHolders: []types.ShardHolder{
			{Type: "agent", Identifier: testAddr2, ShardIndex: 0, CanInitiateRecovery: true},
		},
		RecoveryDelayBlocks:   10,
		ChallengePeriodBlocks: 10,
	}
	_, _ = ms.SetRecoveryConfig(ctx, &types.MsgSetRecoveryConfig{
		Sender: testAddr1,
		Config: config,
	})

	_, _ = ms.InitiateRecovery(ctx, &types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	})
	_, _ = ms.SubmitRecoveryShard(ctx, &types.MsgSubmitRecoveryShard{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
		ShardIndex:     0,
	})

	// Advance past delay
	afterDelay := ctx.WithBlockHeight(111)
	k.ProcessRecoveryTimeouts(afterDelay)
	req, _ := k.GetRecoveryRequest(afterDelay, testAddr1)
	if req.Status != "challengeable" {
		t.Fatalf("expected challengeable, got %s", req.Status)
	}

	// Advance past challenge period
	afterChallenge := ctx.WithBlockHeight(121)
	k.ProcessRecoveryTimeouts(afterChallenge)
	req, _ = k.GetRecoveryRequest(afterChallenge, testAddr1)
	if req.Status != "executable" {
		t.Errorf("expected executable after challenge period, got %s", req.Status)
	}
}

// ---------- ValidateBasic Tests for New Msg Types ----------

func TestMsgFreezeAccount_ValidateBasic(t *testing.T) {
	msg := types.MsgFreezeAccount{
		Sender:  testAddr1,
		Address: testAddr1,
		Reason:  "test",
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.Sender = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty sender")
	}

	msg.Sender = testAddr1
	msg.Address = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty address")
	}
}

func TestMsgUnfreezeAccount_ValidateBasic(t *testing.T) {
	msg := types.MsgUnfreezeAccount{
		Authority: testAddr2, // must be valid bech32
		Address:   testAddr1,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.Authority = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty authority")
	}
}

func TestMsgInitiateRecovery_ValidateBasic(t *testing.T) {
	msg := types.MsgInitiateRecovery{
		Sender:            testAddr2,
		AccountAddress:    testAddr1,
		NewOperationalKey: testPubKey2,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.NewOperationalKey = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty new key")
	}
}

func TestMsgExecuteRecovery_ValidateBasic(t *testing.T) {
	msg := types.MsgExecuteRecovery{
		Sender:         testAddr2,
		AccountAddress: testAddr1,
	}
	if err := msg.ValidateBasic(); err != nil {
		t.Errorf("expected valid, got error: %v", err)
	}

	msg.AccountAddress = ""
	if err := msg.ValidateBasic(); err == nil {
		t.Error("expected error for empty account address")
	}
}
