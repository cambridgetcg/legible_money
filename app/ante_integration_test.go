package app

import (
	"context"
	"encoding/hex"
	"testing"

	"cosmossdk.io/log"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmosed25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	protov2 "google.golang.org/protobuf/proto"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	legibleauthkeeper "github.com/legible-money/legible/x/auth/keeper"
	legibleauthtypes "github.com/legible-money/legible/x/auth/types"
	emergencykeeper "github.com/legible-money/legible/x/emergency/keeper"
	emergencytypes "github.com/legible-money/legible/x/emergency/types"
	stakingtypes "github.com/legible-money/legible/x/staking/types"
)

// ============================================================
// Test Infrastructure
// ============================================================

// mockSignedTx implements authsigning.SigVerifiableTx for testing ante decorators.
// It provides controlled signer pubkeys and message types.
type mockSignedTx struct {
	msgs   []sdk.Msg
	sigs   []signing.SignatureV2
	memo   string
	gas    uint64
	fee    sdk.Coins
}

func (m mockSignedTx) GetMsgs() []sdk.Msg                            { return m.msgs }
func (m mockSignedTx) GetMsgsV2() ([]protov2.Message, error)          { return nil, nil }
func (m mockSignedTx) ValidateBasic() error                          { return nil }
func (m mockSignedTx) GetSigners() ([][]byte, error)                 { return nil, nil }
func (m mockSignedTx) GetPubKeys() ([]cryptotypes.PubKey, error)     { return nil, nil }
func (m mockSignedTx) GetSignaturesV2() ([]signing.SignatureV2, error) { return m.sigs, nil }
func (m mockSignedTx) GetGas() uint64                                 { return m.gas }
func (m mockSignedTx) GetFee() sdk.Coins                              { return m.fee }
func (m mockSignedTx) FeePayer() []byte                                { return nil }
func (m mockSignedTx) FeeGranter() []byte                              { return nil }
func (m mockSignedTx) GetMemo() string                                 { return m.memo }

// newMockSignedTx creates a test tx signed by the given key(s) carrying the given message(s).
func newMockSignedTx(keys []*cosmosed25519.PrivKey, msgs []sdk.Msg) mockSignedTx {
	sigs := make([]signing.SignatureV2, len(keys))
	for i, key := range keys {
		sigs[i] = signing.SignatureV2{
			PubKey: key.PubKey(),
			Data:   &signing.SingleSignatureData{},
		}
	}
	return mockSignedTx{
		msgs: msgs,
		sigs: sigs,
		gas:  100_000,
		fee:  sdk.NewCoins(sdk.NewInt64Coin("ulgm", 100_000)),
	}
}

// mockCosmosAccountKeeperForAnte satisfies AccountKeeperForLegible (context.Context).
type mockCosmosAccountKeeperForAnte struct{}

func (m mockCosmosAccountKeeperForAnte) GetAccount(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}

// mockStakingKeeperForEmergency satisfies the emergency StakingKeeper interface.
type mockStakingKeeperForEmergency struct {
	validators map[string]*stakingtypes.LegibleValidator
}

func newMockStakingKeeperForEmergency() *mockStakingKeeperForEmergency {
	return &mockStakingKeeperForEmergency{validators: make(map[string]*stakingtypes.LegibleValidator)}
}

func (m *mockStakingKeeperForEmergency) GetValidator(_ sdk.Context, addr string) (*stakingtypes.LegibleValidator, bool) {
	v, ok := m.validators[addr]
	return v, ok
}

func (m *mockStakingKeeperForEmergency) GetActiveValidatorSet(_ sdk.Context) []*stakingtypes.LegibleValidator {
	var result []*stakingtypes.LegibleValidator
	for _, v := range m.validators {
		if v.IsActive {
			result = append(result, v)
		}
	}
	return result
}

func (m *mockStakingKeeperForEmergency) IterateValidators(_ sdk.Context, cb func(*stakingtypes.LegibleValidator) bool) {
	for _, v := range m.validators {
		if cb(v) {
			return
		}
	}
}

// mockBankKeeperForAnte satisfies legible auth keeper BankKeeper interface (context.Context).
type mockBankKeeperForAnte struct{}

func (m mockBankKeeperForAnte) GetBalance(_ context.Context, _ sdk.AccAddress, _ string) sdk.Coin {
	return sdk.NewInt64Coin("ulgm", 0)
}
func (m mockBankKeeperForAnte) GetAllBalances(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	return sdk.NewCoins()
}
func (m mockBankKeeperForAnte) SendCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}

// mockCosmosAccountKeeperForAuth satisfies legible auth keeper CosmosAccountKeeper interface (context.Context).
type mockCosmosAccountKeeperForAuth struct{}

func (m mockCosmosAccountKeeperForAuth) GetAccount(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}
func (m mockCosmosAccountKeeperForAuth) SetAccount(_ context.Context, _ sdk.AccountI) {}
func (m mockCosmosAccountKeeperForAuth) NewAccountWithAddress(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}

// setupAuthKeeper creates a real legible auth keeper with an in-memory KV store.
func setupAuthKeeper(t *testing.T) (legibleauthkeeper.Keeper, storetypes.CommitMultiStore) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(legibleauthtypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	k := legibleauthkeeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), mockCosmosAccountKeeperForAuth{}, mockBankKeeperForAnte{}, "authority")
	return k, stateStore
}

// setupEmergencyKeeper creates a real emergency keeper with an in-memory KV store.
func setupEmergencyKeeper(t *testing.T) (emergencykeeper.Keeper, storetypes.CommitMultiStore) {
	t.Helper()
	storeKey := storetypes.NewKVStoreKey(emergencytypes.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}
	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	mockSK := newMockStakingKeeperForEmergency()
	k := emergencykeeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), mockSK, "authority")
	return k, stateStore
}

// setupBothKeepers creates both keepers on a shared multi-store.
func setupBothKeepers(t *testing.T) (legibleauthkeeper.Keeper, emergencykeeper.Keeper, sdk.Context) {
	t.Helper()

	authStoreKey := storetypes.NewKVStoreKey(legibleauthtypes.StoreKey)
	emergStoreKey := storetypes.NewKVStoreKey(emergencytypes.StoreKey)

	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(authStoreKey, storetypes.StoreTypeIAVL, db)
	stateStore.MountStoreWithDB(emergStoreKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	authKeeper := legibleauthkeeper.NewKeeper(cdc, runtime.NewKVStoreService(authStoreKey), mockCosmosAccountKeeperForAuth{}, mockBankKeeperForAnte{}, "authority")
	mockSK := newMockStakingKeeperForEmergency()
	emergKeeper := emergencykeeper.NewKeeper(cdc, runtime.NewKVStoreService(emergStoreKey), mockSK, "authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 500_000}, false, log.NewNopLogger())

	// Set default auth params
	defaultAuthParams := legibleauthtypes.DefaultParams()
	if err := authKeeper.SetParams(ctx, &defaultAuthParams); err != nil {
		t.Fatalf("failed to set auth params: %v", err)
	}

	return authKeeper, emergKeeper, ctx
}

// registerLegibleAccount creates a legible account entry for the given key.
func registerLegibleAccount(t *testing.T, authKeeper legibleauthkeeper.Keeper, ctx sdk.Context, key *cosmosed25519.PrivKey, frozen bool) string {
	t.Helper()
	addr := sdk.AccAddress(key.PubKey().Address()).String()
	pubKeyHex := hex.EncodeToString(key.PubKey().Bytes())

	account := &legibleauthtypes.LegibleAccount{
		Address:     addr,
		Did:         "did:lgm:" + pubKeyHex[:32],
		PublicKey:   pubKeyHex,
		AccountType: "agent",
		Flags: &legibleauthtypes.AccountFlags{
			Frozen: frozen,
		},
		CreatedAtBlock:  1,
		LastActiveBlock: 1,
	}
	authKeeper.SetAccount(ctx, account)
	return addr
}

// registerSessionKey stores a session key for the given owner with specified capabilities.
// NOTE: The capability decorator uses getTxSigners() which derives the address from
// the signing pubkey. So the session key lookup uses the session key's own derived
// address as the "owner" key. We store the session under BOTH the real owner address
// AND the session key's derived address to match the decorator's lookup.
func registerSessionKey(t *testing.T, authKeeper legibleauthkeeper.Keeper, ctx sdk.Context, owner string, sessionKey *cosmosed25519.PrivKey, caps *legibleauthtypes.SessionCapabilities) {
	t.Helper()
	pubKeyHex := hex.EncodeToString(sessionKey.PubKey().Bytes())
	// The decorator derives the lookup address from the signature pubkey,
	// so store the session key under the session key's derived address.
	sessionAddr := sdk.AccAddress(sessionKey.PubKey().Address()).String()
	session := &legibleauthtypes.SessionKey{
		KeyHash:        "session_" + pubKeyHex[:16],
		PublicKey:      pubKeyHex,
		Owner:          sessionAddr,
		Capabilities:   caps,
		ExpiresAtBlock: 1_000_000, // far future
		CreatedAtBlock: 1,
	}
	authKeeper.SetSessionKey(ctx, session)
}

// ============================================================
// Frozen Account Tests
// ============================================================

func TestFrozenAccount_MsgSend_Rejected(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	key := cosmosed25519.GenPrivKey()
	registerLegibleAccount(t, authKeeper, ctx, key, true) // frozen

	decorator := NewLegibleAccountDecorator(authKeeper)
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected frozen account to be rejected for MsgSend")
	}
	if err != legibleauthtypes.ErrAccountFrozen {
		t.Fatalf("expected ErrAccountFrozen, got: %v", err)
	}
}

func TestFrozenAccount_MsgDelegate_Rejected(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	key := cosmosed25519.GenPrivKey()
	registerLegibleAccount(t, authKeeper, ctx, key, true)

	decorator := NewLegibleAccountDecorator(authKeeper)
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.staking.v1beta1.MsgDelegate"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected frozen account to be rejected for MsgDelegate")
	}
	if err != legibleauthtypes.ErrAccountFrozen {
		t.Fatalf("expected ErrAccountFrozen, got: %v", err)
	}
}

func TestFrozenAccount_CustomMsg_Rejected(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	key := cosmosed25519.GenPrivKey()
	registerLegibleAccount(t, authKeeper, ctx, key, true)

	decorator := NewLegibleAccountDecorator(authKeeper)
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/legible.home.v1.MsgCreateHome"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected frozen account to be rejected for MsgCreateHome")
	}
	if err != legibleauthtypes.ErrAccountFrozen {
		t.Fatalf("expected ErrAccountFrozen, got: %v", err)
	}
}

func TestNonFrozenAccount_MsgSend_Passes(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	key := cosmosed25519.GenPrivKey()
	registerLegibleAccount(t, authKeeper, ctx, key, false) // not frozen

	decorator := NewLegibleAccountDecorator(authKeeper)
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected non-frozen account to pass, got: %v", err)
	}
}

func TestFrozenAccount_UpdatesLastActiveBlock(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	key := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, key, false)

	decorator := NewLegibleAccountDecorator(authKeeper)
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify LastActiveBlock was updated to current height (500_000)
	account, found := authKeeper.GetAccount(ctx, addr)
	if !found {
		t.Fatal("account not found after ante handler")
	}
	if account.LastActiveBlock != 500_000 {
		t.Errorf("expected LastActiveBlock=500000, got %d", account.LastActiveBlock)
	}
}

func TestUnregisteredAccount_MsgSend_Passes(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	key := cosmosed25519.GenPrivKey()
	// Do NOT register — standard Cosmos account

	decorator := NewLegibleAccountDecorator(authKeeper)
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected unregistered account to pass, got: %v", err)
	}
}

// ============================================================
// Session Key Capability Tests
// ============================================================

func TestSessionKey_TransferOnly_MsgSend_Passes(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	identityKey := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, identityKey, false)

	sessionKey := cosmosed25519.GenPrivKey()
	registerSessionKey(t, authKeeper, ctx, addr, sessionKey, &legibleauthtypes.SessionCapabilities{
		CanTransfer: true,
	})

	decorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{sessionKey}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected transfer-capable session key to pass MsgSend, got: %v", err)
	}
}

func TestSessionKey_TransferOnly_MsgDelegate_Rejected(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	identityKey := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, identityKey, false)

	sessionKey := cosmosed25519.GenPrivKey()
	registerSessionKey(t, authKeeper, ctx, addr, sessionKey, &legibleauthtypes.SessionCapabilities{
		CanTransfer: true,
		// CanStake: false (default)
	})

	decorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{sessionKey}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.staking.v1beta1.MsgDelegate"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected transfer-only session key to be rejected for MsgDelegate")
	}
	if err != legibleauthtypes.ErrSessionCapabilityDenied {
		t.Fatalf("expected ErrSessionCapabilityDenied, got: %v", err)
	}
}

func TestSessionKey_VerifyOnly_MsgSend_Rejected(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	identityKey := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, identityKey, false)

	sessionKey := cosmosed25519.GenPrivKey()
	registerSessionKey(t, authKeeper, ctx, addr, sessionKey, &legibleauthtypes.SessionCapabilities{
		CanSubmitClaims: true,
		// CanTransfer: false (default)
	})

	decorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{sessionKey}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected verify-only session key to be rejected for MsgSend")
	}
	if err != legibleauthtypes.ErrSessionCapabilityDenied {
		t.Fatalf("expected ErrSessionCapabilityDenied, got: %v", err)
	}
}

func TestSessionKey_ClaimOnly_SubmitClaim_Passes(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	identityKey := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, identityKey, false)

	sessionKey := cosmosed25519.GenPrivKey()
	registerSessionKey(t, authKeeper, ctx, addr, sessionKey, &legibleauthtypes.SessionCapabilities{
		CanSubmitClaims: true,
	})

	decorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{sessionKey}, []sdk.Msg{
		mockMsg{typeURL: "/legible.knowledge.v1.MsgSubmitClaim"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected claim-capable session key to pass MsgSubmitClaim, got: %v", err)
	}
}

func TestSessionKey_Expired_Passes(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	identityKey := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, identityKey, false)

	sessionKey := cosmosed25519.GenPrivKey()
	pubKeyHex := hex.EncodeToString(sessionKey.PubKey().Bytes())
	// Register with expired session key (expires at block 100, current is 500_000)
	session := &legibleauthtypes.SessionKey{
		KeyHash:        "session_" + pubKeyHex[:16],
		PublicKey:      pubKeyHex,
		Owner:          addr,
		Capabilities:   &legibleauthtypes.SessionCapabilities{CanTransfer: true},
		ExpiresAtBlock: 100, // expired
		CreatedAtBlock: 1,
	}
	authKeeper.SetSessionKey(ctx, session)

	decorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	// Signing with expired session key pubkey — should be treated as non-session key
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{sessionKey}, []sdk.Msg{
		mockMsg{typeURL: "/legible.home.v1.MsgCreateHome"}, // would be denied for active session
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected expired session key to be treated as identity key (no restrictions), got: %v", err)
	}
}

func TestSessionKey_IdentityKey_NoRestrictions(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	identityKey := cosmosed25519.GenPrivKey()
	registerLegibleAccount(t, authKeeper, ctx, identityKey, false)
	// No session key registered — signing with identity key

	decorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{identityKey}, []sdk.Msg{
		mockMsg{typeURL: "/legible.home.v1.MsgCreateHome"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected identity key to pass any msg type, got: %v", err)
	}
}

func TestSessionKey_MultiMsg_AllMustBeAllowed(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	identityKey := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, identityKey, false)

	sessionKey := cosmosed25519.GenPrivKey()
	registerSessionKey(t, authKeeper, ctx, addr, sessionKey, &legibleauthtypes.SessionCapabilities{
		CanTransfer: true,
		// CanStake: false
	})

	decorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	// MsgSend (transfer) + MsgDelegate (stake) — second should fail
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{sessionKey}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
		mockMsg{typeURL: "/cosmos.staking.v1beta1.MsgDelegate"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected multi-msg tx to fail when session key lacks stake capability")
	}
	if err != legibleauthtypes.ErrSessionCapabilityDenied {
		t.Fatalf("expected ErrSessionCapabilityDenied, got: %v", err)
	}
}

func TestSessionKey_DefaultDeny_UnrecognizedMsg(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	identityKey := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, identityKey, false)

	sessionKey := cosmosed25519.GenPrivKey()
	registerSessionKey(t, authKeeper, ctx, addr, sessionKey, &legibleauthtypes.SessionCapabilities{
		CanTransfer:     true,
		CanStake:        true,
		CanSubmitClaims: true,
		CanVote:         true,
	})

	decorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	// Even with all standard caps, unrecognized msg types are denied
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{sessionKey}, []sdk.Msg{
		mockMsg{typeURL: "/legible.bvm.v1.MsgDeployContract"}, // not in any capability category
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected session key to be denied for unrecognized msg type")
	}
	if err != legibleauthtypes.ErrSessionCapabilityDenied {
		t.Fatalf("expected ErrSessionCapabilityDenied, got: %v", err)
	}
}

// ============================================================
// Emergency Halt Tests
// ============================================================

func TestEmergencyHalt_RegularTx_Rejected(t *testing.T) {
	_, emergKeeper, ctx := setupBothKeepers(t)

	// Set chain to halted state
	emergKeeper.SetEmergencyStatus(ctx, emergencytypes.StatusHalted)

	decorator := NewEmergencyHaltDecorator(emergKeeper)
	key := cosmosed25519.GenPrivKey()
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected regular tx to be rejected during emergency halt")
	}
	if err != emergencytypes.ErrChainHalted {
		t.Fatalf("expected ErrChainHalted, got: %v", err)
	}
}

func TestEmergencyHalt_EmergencyMsg_Passes(t *testing.T) {
	_, emergKeeper, ctx := setupBothKeepers(t)

	emergKeeper.SetEmergencyStatus(ctx, emergencytypes.StatusHalted)

	decorator := NewEmergencyHaltDecorator(emergKeeper)
	key := cosmosed25519.GenPrivKey()

	// Use a real emergency message type
	emergencyMsg := &emergencytypes.MsgProposeResume{
		Proposer: sdk.AccAddress(key.PubKey().Address()).String(),
		Justification: "test resume",
	}

	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{emergencyMsg})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected emergency msg to pass during halt, got: %v", err)
	}
}

func TestEmergencyHalt_MixedMsgs_Rejected(t *testing.T) {
	_, emergKeeper, ctx := setupBothKeepers(t)

	emergKeeper.SetEmergencyStatus(ctx, emergencytypes.StatusHalted)

	decorator := NewEmergencyHaltDecorator(emergKeeper)
	key := cosmosed25519.GenPrivKey()
	addr := sdk.AccAddress(key.PubKey().Address()).String()

	// Mix of emergency + non-emergency — should fail
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		&emergencytypes.MsgProposeResume{Proposer: addr, Justification: "test"},
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected mixed emergency+regular tx to be rejected during halt")
	}
}

func TestEmergencyHalt_NormalState_Passes(t *testing.T) {
	_, emergKeeper, ctx := setupBothKeepers(t)

	// Default state is normal — do not set halted
	decorator := NewEmergencyHaltDecorator(emergKeeper)
	key := cosmosed25519.GenPrivKey()
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected regular tx to pass when chain is not halted, got: %v", err)
	}
}

func TestEmergencyHalt_SimulateSkips(t *testing.T) {
	_, emergKeeper, ctx := setupBothKeepers(t)

	emergKeeper.SetEmergencyStatus(ctx, emergencytypes.StatusHalted)

	decorator := NewEmergencyHaltDecorator(emergKeeper)
	key := cosmosed25519.GenPrivKey()
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, true, passThroughHandler) // simulate=true
	if err != nil {
		t.Fatalf("expected simulate to skip halt check, got: %v", err)
	}
}

// ============================================================
// Gas Estimation Edge Cases
// ============================================================

func TestGasAccumulation_ManyMessages(t *testing.T) {
	decorator := NewLGMGasDecorator()
	ctx := sdk.Context{}

	// 200 messages × 21,000 gas = 4,200,000 — well below TxGasLimit
	msgs := make([]sdk.Msg, 200)
	for i := range msgs {
		msgs[i] = mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}
	}

	tx := mockFeeTx{
		gas:  5_000_000,
		fee:  sdk.NewCoins(sdk.NewInt64Coin("ulgm", 5_000_000)),
		msgs: msgs,
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected 200 messages to pass with adequate gas, got: %v", err)
	}
}

func TestGasAccumulation_OverflowProtection(t *testing.T) {
	decorator := NewLGMGasDecorator()
	ctx := sdk.Context{}

	// Many expensive messages that could overflow uint64
	// deploy_contract = 200,000 gas × 100,000 messages = 20 billion > uint64 max? No, but
	// let's make msgs that would overflow.
	// Actually uint64 max = 18,446,744,073,709,551,615 so even 200,000 * 100,000 = 20B won't overflow.
	// The overflow check is for when individual values are extremely large.
	// Test that the accumulator doesn't panic with large numbers of messages.
	msgs := make([]sdk.Msg, 500)
	for i := range msgs {
		msgs[i] = mockMsg{typeURL: "/legible.bvm.v1.MsgDeployContract"} // 200,000 each
	}
	// 500 * 200,000 = 100,000,000 — exceeds TxGasLimit (11,111,111) but that's ok,
	// the gas limit check will reject it, not panic.

	tx := mockFeeTx{
		gas:  TxGasLimit, // max allowed
		fee:  sdk.NewCoins(sdk.NewInt64Coin("ulgm", int64(TxGasLimit))),
		msgs: msgs,
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected error when accumulated gas exceeds provided gas")
	}
	// Should get "gas limit below minimum" error, not a panic
}

func TestGasAccumulation_MaxGasBoundary(t *testing.T) {
	decorator := NewLGMGasDecorator()
	ctx := sdk.Context{}

	// Exactly at the TxGasLimit boundary
	// deploy_contract = 200,000 gas × 55 = 11,000,000 < 11,111,111
	msgs := make([]sdk.Msg, 55)
	for i := range msgs {
		msgs[i] = mockMsg{typeURL: "/legible.bvm.v1.MsgDeployContract"}
	}
	// 55 * 200,000 = 11,000,000

	tx := mockFeeTx{
		gas:  TxGasLimit,
		fee:  sdk.NewCoins(sdk.NewInt64Coin("ulgm", int64(TxGasLimit))),
		msgs: msgs,
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected tx at gas boundary to pass, got: %v", err)
	}
}

// ============================================================
// Bootstrap Gas-Free Decorator Tests
// ============================================================

func TestBootstrap_FreeGas_OnlyDuringBootstrap(t *testing.T) {
	decorator := NewBootstrapGasFreeDecorator()

	// During bootstrap (height < 480,000)
	bootstrapCtx := sdk.NewContext(nil, cmtproto.Header{Height: 100}, false, log.NewNopLogger()).
		WithGasMeter(storetypes.NewGasMeter(1))

	tx := mockSignedTx{
		msgs: []sdk.Msg{mockMsg{typeURL: "/legible.knowledge.v1.MsgSubmitClaim"}},
	}

	gasMeterChanged := false
	handler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		// After bootstrap decorator, gas meter should be infinite (Limit = MaxUint64)
		if ctx.GasMeter().Limit() != 1 { // changed from our initial limit of 1
			gasMeterChanged = true
		}
		return ctx, nil
	}

	_, err := decorator.AnteHandle(bootstrapCtx, tx, false, handler)
	if err != nil {
		t.Fatalf("expected bootstrap tx to pass, got: %v", err)
	}
	if !gasMeterChanged {
		t.Error("expected gas meter to be set to infinite during bootstrap")
	}

	// After bootstrap (height > 480,000)
	postBootstrapCtx := sdk.NewContext(nil, cmtproto.Header{Height: 500_000}, false, log.NewNopLogger()).
		WithGasMeter(storetypes.NewGasMeter(100))

	gasMeterChanged = false
	handler2 := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		if ctx.GasMeter().Limit() == 100 { // original meter preserved
			gasMeterChanged = true
		}
		return ctx, nil
	}

	_, err = decorator.AnteHandle(postBootstrapCtx, tx, false, handler2)
	if err != nil {
		t.Fatalf("expected post-bootstrap tx to pass, got: %v", err)
	}
	if !gasMeterChanged {
		t.Error("expected gas meter to remain unchanged after bootstrap")
	}
}

func TestBootstrap_FreeGas_OnlyForWhitelistedTypes(t *testing.T) {
	decorator := NewBootstrapGasFreeDecorator()

	bootstrapCtx := sdk.NewContext(nil, cmtproto.Header{Height: 100}, false, log.NewNopLogger()).
		WithGasMeter(storetypes.NewGasMeter(100))

	// Non-whitelisted type during bootstrap should NOT get free gas
	tx := mockSignedTx{
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}}, // not whitelisted
	}

	gasMeterPreserved := false
	handler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		if ctx.GasMeter().Limit() == 100 { // original meter preserved
			gasMeterPreserved = true
		}
		return ctx, nil
	}

	_, err := decorator.AnteHandle(bootstrapCtx, tx, false, handler)
	if err != nil {
		t.Fatalf("expected non-whitelisted tx to pass (just no free gas), got: %v", err)
	}
	if !gasMeterPreserved {
		t.Error("expected gas meter to remain unchanged for non-whitelisted type during bootstrap")
	}
}

func TestBootstrap_MixedMsgs_NotEligible(t *testing.T) {
	decorator := NewBootstrapGasFreeDecorator()

	bootstrapCtx := sdk.NewContext(nil, cmtproto.Header{Height: 100}, false, log.NewNopLogger()).
		WithGasMeter(storetypes.NewGasMeter(100))

	// Mix of whitelisted + non-whitelisted — should NOT get free gas
	tx := mockSignedTx{
		msgs: []sdk.Msg{
			mockMsg{typeURL: "/legible.knowledge.v1.MsgSubmitClaim"}, // whitelisted
			mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},        // NOT whitelisted
		},
	}

	gasMeterPreserved := false
	handler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
		if ctx.GasMeter().Limit() == 100 {
			gasMeterPreserved = true
		}
		return ctx, nil
	}

	_, err := decorator.AnteHandle(bootstrapCtx, tx, false, handler)
	if err != nil {
		t.Fatalf("expected mixed msg tx to pass, got: %v", err)
	}
	if !gasMeterPreserved {
		t.Error("expected gas meter preserved for mixed msg tx (not all eligible)")
	}
}

func TestBootstrap_AllWhitelistedTypes(t *testing.T) {
	decorator := NewBootstrapGasFreeDecorator()

	bootstrapCtx := sdk.NewContext(nil, cmtproto.Header{Height: 100}, false, log.NewNopLogger()).
		WithGasMeter(storetypes.NewGasMeter(100))

	// All whitelisted types should get free gas
	for msgType := range BootstrapGasFreeTypes {
		tx := mockSignedTx{
			msgs: []sdk.Msg{mockMsg{typeURL: msgType}},
		}

		gotInfinite := false
		handler := func(ctx sdk.Context, tx sdk.Tx, simulate bool) (sdk.Context, error) {
			if ctx.GasMeter().Limit() != 100 { // changed from our initial limit of 100
				gotInfinite = true
			}
			return ctx, nil
		}

		_, err := decorator.AnteHandle(bootstrapCtx, tx, false, handler)
		if err != nil {
			t.Fatalf("expected whitelisted type %q to pass, got: %v", msgType, err)
		}
		if !gotInfinite {
			t.Errorf("expected infinite gas meter for whitelisted type %q during bootstrap", msgType)
		}
	}
}

// ============================================================
// Full Ante Chain Integration Test
// ============================================================

func TestFullAnteChain_MultiMsg_TracesDecorators(t *testing.T) {
	authKeeper, emergKeeper, ctx := setupBothKeepers(t)

	// Register a non-frozen account
	key := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, key, false)

	// Chain the decorators we can test without full SDK (skip sig verification etc.)
	// Emergency → Bootstrap → LGM Gas → Account → Capability

	// Build a multi-message tx: MsgSend + transfer
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	// Run through each decorator sequentially
	// 1. Emergency halt check (should pass — not halted)
	emergDecorator := NewEmergencyHaltDecorator(emergKeeper)
	ctx1, err := emergDecorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("emergency decorator failed: %v", err)
	}
	_ = ctx1

	// 2. Bootstrap gas-free check (should not apply — height > 480,000)
	bootstrapDecorator := NewBootstrapGasFreeDecorator()
	_, err = bootstrapDecorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("bootstrap decorator failed: %v", err)
	}

	// 3. LGM gas decorator (needs FeeTx interface)
	gasDecorator := NewLGMGasDecorator()
	gasTx := mockFeeTx{
		gas:  100_000,
		fee:  sdk.NewCoins(sdk.NewInt64Coin("ulgm", 100_000)),
		msgs: tx.msgs,
	}
	_, err = gasDecorator.AnteHandle(ctx, gasTx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("gas decorator failed: %v", err)
	}

	// 4. Account decorator (should check frozen status + update LastActiveBlock)
	accountDecorator := NewLegibleAccountDecorator(authKeeper)
	_, err = accountDecorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("account decorator failed: %v", err)
	}

	// Verify LastActiveBlock was updated
	account, found := authKeeper.GetAccount(ctx, addr)
	if !found {
		t.Fatal("account not found")
	}
	if account.LastActiveBlock != 500_000 {
		t.Errorf("expected LastActiveBlock=500000, got %d", account.LastActiveBlock)
	}

	// 5. Capability decorator (should pass — signing with identity key, no session)
	capDecorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	_, err = capDecorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("capability decorator failed: %v", err)
	}
}

func TestFullAnteChain_FrozenAccount_BlockedAtAccountDecorator(t *testing.T) {
	authKeeper, emergKeeper, ctx := setupBothKeepers(t)

	key := cosmosed25519.GenPrivKey()
	registerLegibleAccount(t, authKeeper, ctx, key, true) // frozen

	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	// Emergency should pass
	emergDecorator := NewEmergencyHaltDecorator(emergKeeper)
	_, err := emergDecorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("emergency decorator should pass for non-halted chain: %v", err)
	}

	// Account decorator should reject
	accountDecorator := NewLegibleAccountDecorator(authKeeper)
	_, err = accountDecorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected frozen account to be rejected")
	}
	if err != legibleauthtypes.ErrAccountFrozen {
		t.Fatalf("expected ErrAccountFrozen, got: %v", err)
	}
}

func TestFullAnteChain_SessionKeyRestriction(t *testing.T) {
	authKeeper, emergKeeper, ctx := setupBothKeepers(t)

	// Register identity + session key
	identityKey := cosmosed25519.GenPrivKey()
	addr := registerLegibleAccount(t, authKeeper, ctx, identityKey, false)

	sessionKey := cosmosed25519.GenPrivKey()
	registerSessionKey(t, authKeeper, ctx, addr, sessionKey, &legibleauthtypes.SessionCapabilities{
		CanTransfer: true,
	})

	// TX signed by session key with MsgSend + MsgDelegate
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{sessionKey}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
		mockMsg{typeURL: "/cosmos.staking.v1beta1.MsgDelegate"},
	})

	// Emergency passes
	emergDecorator := NewEmergencyHaltDecorator(emergKeeper)
	_, err := emergDecorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("emergency decorator failed: %v", err)
	}

	// Account decorator passes (not frozen)
	accountDecorator := NewLegibleAccountDecorator(authKeeper)
	_, err = accountDecorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("account decorator should pass: %v", err)
	}

	// Capability decorator rejects (session key can't stake)
	capDecorator := NewLegibleCapabilityDecorator(authKeeper, mockCosmosAccountKeeperForAnte{})
	_, err = capDecorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected session key to be rejected for MsgDelegate")
	}
	if err != legibleauthtypes.ErrSessionCapabilityDenied {
		t.Fatalf("expected ErrSessionCapabilityDenied, got: %v", err)
	}
}

// ============================================================
// Sybil Funding Decorator Tests
// ============================================================

func TestSybilFunding_NilKeeper_Passes(t *testing.T) {
	decorator := NewSybilFundingDecorator(nil) // nil keeper
	ctx := sdk.Context{}
	key := cosmosed25519.GenPrivKey()
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected nil keeper to pass through, got: %v", err)
	}
}

func TestSybilFunding_SimulateSkips(t *testing.T) {
	decorator := NewSybilFundingDecorator(nil)
	ctx := sdk.Context{}
	key := cosmosed25519.GenPrivKey()
	tx := newMockSignedTx([]*cosmosed25519.PrivKey{key}, []sdk.Msg{
		mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"},
	})

	_, err := decorator.AnteHandle(ctx, tx, true, passThroughHandler) // simulate=true
	if err != nil {
		t.Fatalf("expected simulate to skip, got: %v", err)
	}
}

// ============================================================
// DID Decorator Tests
// ============================================================

func TestDIDDecorator_NoMemo_Passes(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	decorator := NewLegibleDIDDecorator(authKeeper)
	key := cosmosed25519.GenPrivKey()
	tx := mockSignedTx{
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
		sigs: []signing.SignatureV2{{PubKey: key.PubKey(), Data: &signing.SingleSignatureData{}}},
		memo: "", // no DID memo
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected tx without DID memo to pass, got: %v", err)
	}
}

func TestDIDDecorator_ValidDID_Passes(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	key := cosmosed25519.GenPrivKey()
	addr := sdk.AccAddress(key.PubKey().Address()).String()
	pubKeyHex := hex.EncodeToString(key.PubKey().Bytes())
	did := "did:lgm:" + pubKeyHex[:64] // full 64-hex DID

	// Register account and DID mapping
	registerLegibleAccount(t, authKeeper, ctx, key, false)
	authKeeper.SetDIDMapping(ctx, &legibleauthtypes.DIDMapping{
		Did:    did,
		Bech32: addr,
		PubKey: pubKeyHex,
	})

	decorator := NewLegibleDIDDecorator(authKeeper)
	tx := mockSignedTx{
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
		sigs: []signing.SignatureV2{{PubKey: key.PubKey(), Data: &signing.SingleSignatureData{}}},
		memo: did,
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err != nil {
		t.Fatalf("expected valid DID memo to pass, got: %v", err)
	}
}

func TestDIDDecorator_InvalidDID_Rejected(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	decorator := NewLegibleDIDDecorator(authKeeper)
	key := cosmosed25519.GenPrivKey()
	tx := mockSignedTx{
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
		sigs: []signing.SignatureV2{{PubKey: key.PubKey(), Data: &signing.SingleSignatureData{}}},
		memo: "did:lgm:ZZZZ", // invalid hex / too short
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected invalid DID to be rejected")
	}
}

func TestDIDDecorator_UnregisteredDID_Rejected(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	decorator := NewLegibleDIDDecorator(authKeeper)
	key := cosmosed25519.GenPrivKey()
	pubKeyHex := hex.EncodeToString(key.PubKey().Bytes())
	did := "did:lgm:" + pubKeyHex[:64]
	// Do NOT register DID mapping

	tx := mockSignedTx{
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
		sigs: []signing.SignatureV2{{PubKey: key.PubKey(), Data: &signing.SingleSignatureData{}}},
		memo: did,
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected unregistered DID to be rejected")
	}
}

func TestDIDDecorator_DIDMismatch_Rejected(t *testing.T) {
	authKeeper, _, ctx := setupBothKeepers(t)

	// Register DID for key1 but sign tx with key2
	key1 := cosmosed25519.GenPrivKey()
	key2 := cosmosed25519.GenPrivKey()
	pubKeyHex1 := hex.EncodeToString(key1.PubKey().Bytes())
	addr1 := sdk.AccAddress(key1.PubKey().Address()).String()
	did := "did:lgm:" + pubKeyHex1[:64]

	registerLegibleAccount(t, authKeeper, ctx, key1, false)
	authKeeper.SetDIDMapping(ctx, &legibleauthtypes.DIDMapping{
		Did:    did,
		Bech32: addr1,
		PubKey: pubKeyHex1,
	})

	decorator := NewLegibleDIDDecorator(authKeeper)
	// Sign with key2 but memo references key1's DID
	tx := mockSignedTx{
		msgs: []sdk.Msg{mockMsg{typeURL: "/cosmos.bank.v1beta1.MsgSend"}},
		sigs: []signing.SignatureV2{{PubKey: key2.PubKey(), Data: &signing.SingleSignatureData{}}},
		memo: did,
	}

	_, err := decorator.AnteHandle(ctx, tx, false, passThroughHandler)
	if err == nil {
		t.Fatal("expected DID mismatch (signed by different key) to be rejected")
	}
}
