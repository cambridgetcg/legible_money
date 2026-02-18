package keeper_test

import (
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/runtime"
	sdk "github.com/cosmos/cosmos-sdk/types"

	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"github.com/legible-money/legible/x/channels/keeper"
	"github.com/legible-money/legible/x/channels/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

const testChainID = "legible-test-1"

// ---------- Ed25519 Test Key Infrastructure ----------

// testParty holds an Ed25519 key pair and derived bech32 address.
type testParty struct {
	Name    string
	PubKey  ed25519.PublicKey
	PrivKey ed25519.PrivateKey
	Addr    string
}

// newTestParty derives a deterministic Ed25519 key pair from a name.
func newTestParty(name string) testParty {
	seed := sha256.Sum256([]byte("test_seed:" + name))
	privKey := ed25519.NewKeyFromSeed(seed[:])
	pubKey := privKey.Public().(ed25519.PublicKey)
	addrHash := sha256.Sum256(pubKey)
	addr := sdk.AccAddress(addrHash[:20]).String()
	return testParty{Name: name, PubKey: pubKey, PrivKey: privKey, Addr: addr}
}

// signPacked creates a packed [32-byte pubkey][64-byte sig] for a channel operation.
// Uses testChainID for domain separation.
func signPacked(party testParty, operation, channelId string, sequence uint64, spent string) []byte {
	return signPackedWithChain(party, operation, testChainID, channelId, sequence, spent)
}

// signPackedWithChain creates a packed signature with an explicit chain ID.
func signPackedWithChain(party testParty, operation, chainId, channelId string, sequence uint64, spent string) []byte {
	payload := types.ChannelSigningPayload(operation, chainId, channelId, sequence, spent)
	sig := ed25519.Sign(party.PrivKey, payload)
	return types.PackSignature(party.PubKey, sig)
}

// signHex creates a hex-encoded packed signature for ChannelStateUpdate fields.
func signHex(party testParty, operation, channelId string, sequence uint64, spent string) string {
	return hex.EncodeToString(signPacked(party, operation, channelId, sequence, spent))
}

// ---------- Mock BankKeeper ----------

type mockBankKeeper struct {
	balances       map[string]map[string]int64
	moduleBalances map[string]map[string]int64
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances:       make(map[string]map[string]int64),
		moduleBalances: make(map[string]map[string]int64),
	}
}

func (m *mockBankKeeper) setBalance(addr, denom string, amount int64) {
	if m.balances[addr] == nil {
		m.balances[addr] = make(map[string]int64)
	}
	m.balances[addr][denom] = amount
}

func (m *mockBankKeeper) SendCoins(_ context.Context, fromAddr, toAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		from := fromAddr.String()
		to := toAddr.String()
		if m.balances[from] == nil {
			m.balances[from] = make(map[string]int64)
		}
		if m.balances[to] == nil {
			m.balances[to] = make(map[string]int64)
		}
		m.balances[from][coin.Denom] -= coin.Amount.Int64()
		m.balances[to][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	if bal, ok := m.balances[addr.String()]; ok {
		if amt, exists := bal[denom]; exists {
			return sdk.NewCoin(denom, sdkmath.NewInt(amt))
		}
	}
	return sdk.NewCoin(denom, sdkmath.ZeroInt())
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		from := senderAddr.String()
		if m.balances[from] == nil {
			m.balances[from] = make(map[string]int64)
		}
		if m.moduleBalances[recipientModule] == nil {
			m.moduleBalances[recipientModule] = make(map[string]int64)
		}
		m.balances[from][coin.Denom] -= coin.Amount.Int64()
		m.moduleBalances[recipientModule][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		to := recipientAddr.String()
		if m.moduleBalances[senderModule] == nil {
			m.moduleBalances[senderModule] = make(map[string]int64)
		}
		if m.balances[to] == nil {
			m.balances[to] = make(map[string]int64)
		}
		m.moduleBalances[senderModule][coin.Denom] -= coin.Amount.Int64()
		m.balances[to][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) BurnCoins(_ context.Context, moduleName string, amt sdk.Coins) error {
	for _, coin := range amt {
		if m.moduleBalances[moduleName] == nil {
			m.moduleBalances[moduleName] = make(map[string]int64)
		}
		m.moduleBalances[moduleName][coin.Denom] -= coin.Amount.Int64()
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToModule(_ context.Context, senderModule, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		if m.moduleBalances[senderModule] == nil {
			m.moduleBalances[senderModule] = make(map[string]int64)
		}
		if m.moduleBalances[recipientModule] == nil {
			m.moduleBalances[recipientModule] = make(map[string]int64)
		}
		m.moduleBalances[senderModule][coin.Denom] -= coin.Amount.Int64()
		m.moduleBalances[recipientModule][coin.Denom] += coin.Amount.Int64()
	}
	return nil
}

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
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

	mockBK := newMockBankKeeper()

	storeService := runtime.NewKVStoreService(storeKey)
	k := keeper.NewKeeper(cdc, storeService, mockBK, "lgm1authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100, ChainID: testChainID}, false, log.NewNopLogger())

	return k, ctx, mockBK
}

func testAddr(name string) string {
	return sdk.AccAddress([]byte(name + "_address_padding")).String()
}

var (
	payer1Party    testParty
	payer2Party    testParty
	provider1Party testParty
	provider2Party testParty

	payer1    string
	payer2    string
	provider1 string
	provider2 string
)

func initAddresses() {
	payer1Party = newTestParty("payer001")
	payer2Party = newTestParty("payer002")
	provider1Party = newTestParty("provider1")
	provider2Party = newTestParty("provider2")

	payer1 = payer1Party.Addr
	payer2 = payer2Party.Addr
	provider1 = provider1Party.Addr
	provider2 = provider2Party.Addr
}

// ==================== Params Tests ====================

func TestDefaultParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	params := k.GetParams(ctx)
	if params.MaxChannelDuration != 1036800 {
		t.Errorf("expected max duration 1036800, got %d", params.MaxChannelDuration)
	}
	if params.MinDeposit != "1000000" {
		t.Errorf("expected min deposit 1000000, got %s", params.MinDeposit)
	}
	if params.MaxChannelsPerPayer != 50 {
		t.Errorf("expected max channels 50, got %d", params.MaxChannelsPerPayer)
	}
}

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	custom := types.Params{
		MinDeposit:            "500000",
		MaxChannelDuration:    200000,
		DefaultSettlementFreq: 50,
		DisputePeriodBlocks:   500,
		ExpiryGracePeriod:     200,
		FraudPenaltyBps:       1000,
		MaxChannelsPerPayer:   10,
	}
	k.SetParams(ctx, custom)
	got := k.GetParams(ctx)
	if got.MinDeposit != "500000" {
		t.Errorf("expected min deposit 500000, got %s", got.MinDeposit)
	}
	if got.MaxChannelsPerPayer != 10 {
		t.Errorf("expected max channels 10, got %d", got.MaxChannelsPerPayer)
	}
}

// ==================== Channel CRUD Tests ====================

func TestSetGetChannel(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ch := types.PaymentChannel{
		ChannelId: "pc-100-1",
		Payer:     payer1,
		Provider:  provider1,
		Deposited: "5000000",
		Spent:     "0",
		Available: "5000000",
		Status:    types.ChannelStatusOpen,
		Nonce:     0,
	}
	k.SetChannel(ctx, ch)

	got, found := k.GetChannel(ctx, "pc-100-1")
	if !found {
		t.Fatal("channel not found")
	}
	if got.Payer != payer1 {
		t.Errorf("expected payer %s, got %s", payer1, got.Payer)
	}
	if got.Deposited != "5000000" {
		t.Errorf("expected deposited 5000000, got %s", got.Deposited)
	}
}

func TestGetChannelNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	_, found := k.GetChannel(ctx, "nonexistent")
	if found {
		t.Error("expected channel not found")
	}
}

func TestDeleteChannel(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ch := types.PaymentChannel{ChannelId: "pc-del", Payer: payer1, Provider: provider1}
	k.SetChannel(ctx, ch)
	k.DeleteChannel(ctx, ch)
	_, found := k.GetChannel(ctx, "pc-del")
	if found {
		t.Error("channel should have been deleted")
	}
}

func TestGetChannelsByPayer(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-p1-a", Payer: payer1, Provider: provider1, Status: types.ChannelStatusOpen})
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-p1-b", Payer: payer1, Provider: provider2, Status: types.ChannelStatusOpen})
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-p2-a", Payer: payer2, Provider: provider1, Status: types.ChannelStatusOpen})

	channels := k.GetChannelsByPayer(ctx, payer1)
	if len(channels) != 2 {
		t.Errorf("expected 2 channels for payer1, got %d", len(channels))
	}
}

func TestGetChannelsByProvider(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-pr1-a", Payer: payer1, Provider: provider1, Status: types.ChannelStatusOpen})
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-pr1-b", Payer: payer2, Provider: provider1, Status: types.ChannelStatusOpen})
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-pr2-a", Payer: payer1, Provider: provider2, Status: types.ChannelStatusOpen})

	channels := k.GetChannelsByProvider(ctx, provider1)
	if len(channels) != 2 {
		t.Errorf("expected 2 channels for provider1, got %d", len(channels))
	}
}

func TestGetOpenChannelCountForPayer(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-oc-1", Payer: payer1, Provider: provider1, Status: types.ChannelStatusOpen})
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-oc-2", Payer: payer1, Provider: provider2, Status: types.ChannelStatusClosed})
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-oc-3", Payer: payer1, Provider: provider1, Status: types.ChannelStatusOpen})

	count := k.GetOpenChannelCountForPayer(ctx, payer1)
	if count != 2 {
		t.Errorf("expected 2 open channels, got %d", count)
	}
}

// ==================== Dispute CRUD Tests ====================

func TestSetGetDispute(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	d := types.ChannelDispute{
		ChannelId:     "pc-100-1",
		Disputer:      payer1,
		DisputedNonce: 5,
		DisputedSpent: "1000000",
		DeadlineBlock: 200,
	}
	k.SetDispute(ctx, d)

	got, found := k.GetDispute(ctx, "pc-100-1")
	if !found {
		t.Fatal("dispute not found")
	}
	if got.DisputedNonce != 5 {
		t.Errorf("expected nonce 5, got %d", got.DisputedNonce)
	}
}

func TestDeleteDispute(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	d := types.ChannelDispute{ChannelId: "pc-dd"}
	k.SetDispute(ctx, d)
	k.DeleteDispute(ctx, "pc-dd")
	_, found := k.GetDispute(ctx, "pc-dd")
	if found {
		t.Error("dispute should have been deleted")
	}
}

func TestGetExpiredDisputes(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetDispute(ctx, types.ChannelDispute{ChannelId: "pc-exp-1", DeadlineBlock: 50, Resolved: false})
	k.SetDispute(ctx, types.ChannelDispute{ChannelId: "pc-exp-2", DeadlineBlock: 150, Resolved: false})
	k.SetDispute(ctx, types.ChannelDispute{ChannelId: "pc-exp-3", DeadlineBlock: 50, Resolved: true})

	expired := k.GetExpiredDisputes(ctx, 100)
	if len(expired) != 1 {
		t.Errorf("expected 1 expired dispute, got %d", len(expired))
	}
	if expired[0].ChannelId != "pc-exp-1" {
		t.Errorf("expected pc-exp-1, got %s", expired[0].ChannelId)
	}
}

// ==================== Channel Counter Tests ====================

func TestGetNextChannelId(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	id1 := k.GetNextChannelId(ctx)
	id2 := k.GetNextChannelId(ctx)
	if id1 != 1 || id2 != 2 {
		t.Errorf("expected 1, 2, got %d, %d", id1, id2)
	}
}

// ==================== OpenChannel Tests ====================

func TestOpenChannel_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 10000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, err := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "5000000",
		TimeoutHeight: 10000,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ChannelId == "" {
		t.Error("expected channel ID")
	}

	ch, found := k.GetChannel(ctx, resp.ChannelId)
	if !found {
		t.Fatal("channel not found after open")
	}
	if ch.Deposited != "5000000" {
		t.Errorf("expected deposited 5000000, got %s", ch.Deposited)
	}
	if ch.Status != types.ChannelStatusOpen {
		t.Errorf("expected open status, got %s", ch.Status)
	}
	if ch.ExpiresAtBlock != uint64(ctx.BlockHeight())+10000 {
		t.Errorf("unexpected expiry block")
	}
}

func TestOpenChannel_InsufficientDeposit(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 10000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	_, err := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "100", // below min 1000000
		TimeoutHeight: 10000,
	})
	if err == nil {
		t.Error("expected insufficient deposit error")
	}
}

func TestOpenChannel_DurationExceeded(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 10000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	_, err := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "5000000",
		TimeoutHeight: 9999999, // above max
	})
	if err == nil {
		t.Error("expected duration exceeded error")
	}
}

func TestOpenChannel_SelfChannel(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 10000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	_, err := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      payer1,
		Deposit:       "5000000",
		TimeoutHeight: 10000,
	})
	if err == nil {
		t.Error("expected self-channel error")
	}
}

func TestOpenChannel_MaxChannelsExceeded(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 1000000000)

	params := types.DefaultParams()
	params.MaxChannelsPerPayer = 2
	k.SetParams(ctx, *params)

	msgServer := keeper.NewFullMsgServerImpl(k)
	for i := 0; i < 2; i++ {
		_, err := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
			Sender:        payer1,
			Receiver:      provider1,
			Deposit:       "1000000",
			TimeoutHeight: 10000,
		})
		if err != nil {
			t.Fatalf("failed to open channel %d: %v", i, err)
		}
	}

	_, err := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender:        payer1,
		Receiver:      provider1,
		Deposit:       "1000000",
		TimeoutHeight: 10000,
	})
	if err == nil {
		t.Error("expected max channels exceeded error")
	}
}

// ==================== DepositChannel Tests ====================

func TestDepositChannel_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.DepositChannel(ctx, &types.MsgDepositChannel{
		Depositor: payer1,
		ChannelId: resp.ChannelId,
		Amount:    "3000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Deposited != "8000000" {
		t.Errorf("expected deposited 8000000, got %s", ch.Deposited)
	}
	if ch.Available != "8000000" {
		t.Errorf("expected available 8000000, got %s", ch.Available)
	}
}

func TestDepositChannel_NotPayer(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)
	bk.setBalance(provider1, "ulgm", 5000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.DepositChannel(ctx, &types.MsgDepositChannel{
		Depositor: provider1,
		ChannelId: resp.ChannelId,
		Amount:    "1000000",
	})
	if err == nil {
		t.Error("expected not payer error")
	}
}

func TestDepositChannel_ClosedChannel(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ch := types.PaymentChannel{
		ChannelId: "pc-closed", Payer: payer1, Provider: provider1,
		Status: types.ChannelStatusClosed, Deposited: "5000000", Spent: "0", Available: "5000000",
	}
	k.SetChannel(ctx, ch)

	msgServer := keeper.NewFullMsgServerImpl(k)
	_, err := msgServer.DepositChannel(ctx, &types.MsgDepositChannel{
		Depositor: payer1, ChannelId: "pc-closed", Amount: "1000000",
	})
	if err == nil {
		t.Error("expected channel not open error")
	}
}

// ==================== UpdateState Tests (with signatures) ====================

func TestUpdateState_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender:    payer1,
		ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			ChannelId:         resp.ChannelId,
			Nonce:             1,
			Spent:             "2000000",
			StateHash:         "abc123",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "2000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "2000000"),
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Nonce != 1 {
		t.Errorf("expected nonce 1, got %d", ch.Nonce)
	}
	if ch.Spent != "2000000" {
		t.Errorf("expected spent 2000000, got %s", ch.Spent)
	}
	if ch.Available != "3000000" {
		t.Errorf("expected available 3000000, got %s", ch.Available)
	}
}

func TestUpdateState_MissingSignature(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// No signatures at all
	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender:    payer1,
		ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce: 1, Spent: "2000000",
		},
	})
	if err == nil {
		t.Error("expected signature error for missing signatures")
	}
}

func TestUpdateState_InvalidNonce(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Update with nonce 1
	msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce:             1,
			Spent:             "1000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "1000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "1000000"),
		},
	})

	// Try nonce 1 again (not higher) — sig is valid but nonce check fails
	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce:             1,
			Spent:             "2000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "2000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "2000000"),
		},
	})
	if err == nil {
		t.Error("expected invalid nonce error")
	}
}

func TestUpdateState_SpentExceedsDeposit(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce:             1,
			Spent:             "99999999",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "99999999"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "99999999"),
		},
	})
	if err == nil {
		t.Error("expected spent exceeds deposit error")
	}
}

func TestUpdateState_NotParty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender:    payer2, // not a party
		ChannelId: resp.ChannelId,
		Update:    types.ChannelStateUpdate{Nonce: 1, Spent: "1000000"},
	})
	if err == nil {
		t.Error("expected not party error")
	}
}

// ==================== DisputeChannel Tests (with signatures) ====================

func TestDisputeChannel_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Payer disputes, proving provider signed this state
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer:               payer1,
		ChannelId:              resp.ChannelId,
		Sequence:               1,
		ClaimedBalanceReceiver: "2000000",
		ProofSignature:         signPacked(provider1Party, "dispute", resp.ChannelId, 1, "2000000"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusDisputed {
		t.Errorf("expected disputed status, got %s", ch.Status)
	}

	d, found := k.GetDispute(ctx, resp.ChannelId)
	if !found {
		t.Fatal("dispute not found")
	}
	if d.DisputedNonce != 1 {
		t.Errorf("expected nonce 1, got %d", d.DisputedNonce)
	}
}

func TestDisputeChannel_MissingSignature(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// No proof signature
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer:               payer1,
		ChannelId:              resp.ChannelId,
		Sequence:               1,
		ClaimedBalanceReceiver: "2000000",
	})
	if err == nil {
		t.Error("expected signature error for missing proof")
	}
}

func TestDisputeChannel_AlreadyDisputed(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// First dispute
	msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: payer1, ChannelId: resp.ChannelId, Sequence: 1,
		ClaimedBalanceReceiver: "1000000",
		ProofSignature:         signPacked(provider1Party, "dispute", resp.ChannelId, 1, "1000000"),
	})

	// Second dispute with same sequence (should fail)
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: provider1, ChannelId: resp.ChannelId, Sequence: 1,
		ClaimedBalanceReceiver: "500000",
		ProofSignature:         signPacked(payer1Party, "dispute", resp.ChannelId, 1, "500000"),
	})
	if err == nil {
		t.Error("expected already disputed error")
	}
}

// ==================== SettleChannel Tests ====================

func TestSettleChannel_OpensClosingPeriod(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Update state with signatures
	msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce:             1,
			Spent:             "2000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "2000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "2000000"),
		},
	})

	// SettleChannel on open → should transition to closing
	sresp, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No funds distributed yet
	if sresp.AmountToSender != "0" || sresp.AmountToReceiver != "0" {
		t.Errorf("expected zero distribution during closing, got %s/%s", sresp.AmountToSender, sresp.AmountToReceiver)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosing {
		t.Errorf("expected closing status, got %s", ch.Status)
	}
	if ch.DisputeDeadline != uint64(ctx.BlockHeight())+1000 {
		t.Errorf("expected deadline %d, got %d", uint64(ctx.BlockHeight())+1000, ch.DisputeDeadline)
	}
}

func TestSettleChannel_ClosingToFinal(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Update state
	msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce:             1,
			Spent:             "2000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "2000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "2000000"),
		},
	})

	// Initiate closing
	msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})

	// Fund module account for distribution
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}

	// Advance past deadline
	ctx = ctx.WithBlockHeight(int64(100 + 1001))

	// Finalize settlement
	sresp, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sresp.AmountToSender != "3000000" {
		t.Errorf("expected amount to sender 3000000, got %s", sresp.AmountToSender)
	}
	if sresp.AmountToReceiver != "2000000" {
		t.Errorf("expected amount to receiver 2000000, got %s", sresp.AmountToReceiver)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosed {
		t.Errorf("expected closed status, got %s", ch.Status)
	}
}

func TestSettleChannel_ClosingBeforeDeadline(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Initiate closing
	msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})

	// Try to finalize before deadline
	_, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err == nil {
		t.Error("expected dispute period active error")
	}
}

func TestSettleChannel_AfterDispute(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Provider disputes with sequence 1 and claimed balance 3000000
	msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: provider1, ChannelId: resp.ChannelId,
		Sequence: 1, ClaimedBalanceReceiver: "3000000",
		ProofSignature: signPacked(payer1Party, "dispute", resp.ChannelId, 1, "3000000"),
	})

	// Advance past deadline
	ctx = ctx.WithBlockHeight(int64(100 + 1001))

	// Fund module account
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}

	// Settle
	sresp, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if sresp.AmountToSender != "2000000" {
		t.Errorf("expected amount to sender 2000000, got %s", sresp.AmountToSender)
	}
	if sresp.AmountToReceiver != "3000000" {
		t.Errorf("expected amount to receiver 3000000, got %s", sresp.AmountToReceiver)
	}
}

// ==================== CloseChannel Tests (with signatures) ====================

func TestCloseChannel_Cooperative(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Fund module account
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}

	_, err := msgServer.CloseChannel(ctx, &types.MsgCloseChannel{
		Closer:                payer1,
		ChannelId:             resp.ChannelId,
		Sequence:              5,
		FinalBalanceReceiver:  "4000000",
		FinalBalanceSender:    "1000000",
		CounterpartySignature: signPacked(provider1Party, "close", resp.ChannelId, 5, "4000000"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosed {
		t.Errorf("expected closed, got %s", ch.Status)
	}
	if ch.Nonce != 5 {
		t.Errorf("expected nonce 5, got %d", ch.Nonce)
	}
	if ch.Spent != "4000000" {
		t.Errorf("expected spent 4000000, got %s", ch.Spent)
	}
	if ch.Available != "1000000" {
		t.Errorf("expected available 1000000, got %s", ch.Available)
	}
}

func TestCloseChannel_MissingSignature(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// No counterparty signature
	_, err := msgServer.CloseChannel(ctx, &types.MsgCloseChannel{
		Closer:               payer1,
		ChannelId:            resp.ChannelId,
		Sequence:             5,
		FinalBalanceReceiver: "4000000",
	})
	if err == nil {
		t.Error("expected signature error for missing counterparty signature")
	}
}

func TestCloseChannel_WrongSignerAddress(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Payer closes but provides own signature instead of counterparty's
	_, err := msgServer.CloseChannel(ctx, &types.MsgCloseChannel{
		Closer:                payer1,
		ChannelId:             resp.ChannelId,
		Sequence:              5,
		FinalBalanceReceiver:  "4000000",
		CounterpartySignature: signPacked(payer1Party, "close", resp.ChannelId, 5, "4000000"), // wrong signer
	})
	if err == nil {
		t.Error("expected signature error: payer signed instead of provider")
	}
}

// ==================== ClaimExpired Tests ====================

func TestClaimExpired_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 100,
	})

	// Update state with signatures
	msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce:             1,
			Spent:             "1000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "1000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "1000000"),
		},
	})

	// Fund module account
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}

	// Advance past expiry + grace period
	ctx = ctx.WithBlockHeight(701)

	cresp, err := msgServer.ClaimExpired(ctx, &types.MsgClaimExpired{
		Claimer:   payer1,
		ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cresp.RefundedAmount != "4000000" {
		t.Errorf("expected refund 4000000, got %s", cresp.RefundedAmount)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosed {
		t.Errorf("expected closed, got %s", ch.Status)
	}
}

func TestClaimExpired_NotExpired(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 100000,
	})

	_, err := msgServer.ClaimExpired(ctx, &types.MsgClaimExpired{
		Claimer:   payer1,
		ChannelId: resp.ChannelId,
	})
	if err == nil {
		t.Error("expected not expired error")
	}
}

func TestClaimExpired_NotPayer(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 100,
	})

	ctx = ctx.WithBlockHeight(701)

	_, err := msgServer.ClaimExpired(ctx, &types.MsgClaimExpired{
		Claimer:   provider1, // not payer
		ChannelId: resp.ChannelId,
	})
	if err == nil {
		t.Error("expected not payer error")
	}
}

// ==================== UpdateParams Tests ====================

func TestUpdateParams_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewFullMsgServerImpl(k)

	newParams := &types.Params{
		MinDeposit:            "500000",
		MaxChannelDuration:    200000,
		DefaultSettlementFreq: 50,
		DisputePeriodBlocks:   500,
		ExpiryGracePeriod:     200,
		FraudPenaltyBps:       1000,
		MaxChannelsPerPayer:   10,
	}
	_, err := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: "lgm1authority",
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := k.GetParams(ctx)
	if got.MinDeposit != "500000" {
		t.Errorf("expected min deposit 500000, got %s", got.MinDeposit)
	}
	if got.MaxChannelsPerPayer != 10 {
		t.Errorf("expected max channels 10, got %d", got.MaxChannelsPerPayer)
	}
}

func TestUpdateParams_Unauthorized(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	msgServer := keeper.NewFullMsgServerImpl(k)

	_, err := msgServer.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: payer1, // not the authority
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Error("expected unauthorized error")
	}
}

// ==================== UpdateChannel (protobuf) Tests ====================

func TestUpdateChannel_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.UpdateChannel(ctx, &types.MsgUpdateChannel{
		Sender:             payer1,
		ChannelId:          resp.ChannelId,
		Sequence:           1,
		NewBalanceReceiver: "2000000",
		Signature:          signPacked(provider1Party, "update", resp.ChannelId, 1, "2000000"),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Nonce != 1 {
		t.Errorf("expected nonce 1, got %d", ch.Nonce)
	}
	if ch.Spent != "2000000" {
		t.Errorf("expected spent 2000000, got %s", ch.Spent)
	}
	if ch.Available != "3000000" {
		t.Errorf("expected available 3000000, got %s", ch.Available)
	}
}

func TestUpdateChannel_MissingSignature(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.UpdateChannel(ctx, &types.MsgUpdateChannel{
		Sender: payer1, ChannelId: resp.ChannelId,
		Sequence: 1, NewBalanceReceiver: "2000000",
		// No signature
	})
	if err == nil {
		t.Error("expected signature error")
	}
}

func TestUpdateChannel_SpentExceedsDeposit(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.UpdateChannel(ctx, &types.MsgUpdateChannel{
		Sender: payer1, ChannelId: resp.ChannelId,
		Sequence: 1, NewBalanceReceiver: "99999999",
		Signature: signPacked(provider1Party, "update", resp.ChannelId, 1, "99999999"),
	})
	if err == nil {
		t.Error("expected spent exceeds deposit error")
	}
}

func TestUpdateChannel_NotParty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.UpdateChannel(ctx, &types.MsgUpdateChannel{
		Sender: payer2, ChannelId: resp.ChannelId,
		Sequence: 1, NewBalanceReceiver: "1000000",
	})
	if err == nil {
		t.Error("expected not party error")
	}
}

// ==================== DisputeChannel Counter Tests ====================

func TestDisputeChannel_HigherSequenceSupersedes(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// First dispute with sequence 1
	msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: payer1, ChannelId: resp.ChannelId,
		Sequence: 1, ClaimedBalanceReceiver: "1000000",
		ProofSignature: signPacked(provider1Party, "dispute", resp.ChannelId, 1, "1000000"),
	})

	// Counter-dispute with higher sequence
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: provider1, ChannelId: resp.ChannelId,
		Sequence: 5, ClaimedBalanceReceiver: "3000000",
		ProofSignature: signPacked(payer1Party, "dispute", resp.ChannelId, 5, "3000000"),
	})
	if err != nil {
		t.Fatalf("expected higher sequence dispute to succeed: %v", err)
	}

	d, found := k.GetDispute(ctx, resp.ChannelId)
	if !found {
		t.Fatal("dispute not found")
	}
	if d.DisputedNonce != 5 {
		t.Errorf("expected nonce 5, got %d", d.DisputedNonce)
	}
	if d.DisputedSpent != "3000000" {
		t.Errorf("expected disputed spent 3000000, got %s", d.DisputedSpent)
	}
}

func TestDisputeChannel_NotParty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: payer2, ChannelId: resp.ChannelId,
		Sequence: 1, ClaimedBalanceReceiver: "1000000",
	})
	if err == nil {
		t.Error("expected not party error")
	}
}

// ==================== SettleChannel Edge Cases ====================

func TestSettleChannel_AlreadyClosed(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ch := types.PaymentChannel{
		ChannelId: "pc-sc", Payer: payer1, Provider: provider1,
		Status: types.ChannelStatusClosed, Deposited: "5000000", Spent: "0", Available: "5000000",
	}
	k.SetChannel(ctx, ch)

	msgServer := keeper.NewFullMsgServerImpl(k)
	_, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: "pc-sc",
	})
	if err == nil {
		t.Error("expected already closed error")
	}
}

func TestSettleChannel_DisputePeriodStillActive(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Dispute
	msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: payer1, ChannelId: resp.ChannelId,
		Sequence: 1, ClaimedBalanceReceiver: "2000000",
		ProofSignature: signPacked(provider1Party, "dispute", resp.ChannelId, 1, "2000000"),
	})

	// Try to settle without advancing past deadline
	_, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err == nil {
		t.Error("expected dispute period active error")
	}
}

// ==================== CloseChannel Edge Cases ====================

func TestCloseChannel_SpentExceedsDeposit(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.CloseChannel(ctx, &types.MsgCloseChannel{
		Closer: payer1, ChannelId: resp.ChannelId,
		Sequence: 1, FinalBalanceReceiver: "99999999",
		CounterpartySignature: signPacked(provider1Party, "close", resp.ChannelId, 1, "99999999"),
	})
	if err == nil {
		t.Error("expected spent exceeds deposit error")
	}
}

func TestCloseChannel_NotParty(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.CloseChannel(ctx, &types.MsgCloseChannel{
		Closer: payer2, ChannelId: resp.ChannelId,
		Sequence: 1, FinalBalanceReceiver: "1000000",
	})
	if err == nil {
		t.Error("expected not party error")
	}
}

// ==================== QueryChannelsByProvider Tests ====================

func TestQueryChannelsByProvider(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-qpr-1", Payer: payer1, Provider: provider1})
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-qpr-2", Payer: payer2, Provider: provider1})
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-qpr-3", Payer: payer1, Provider: provider2})

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.ChannelsByProvider(ctx, &types.QueryChannelsByProviderRequest{Provider: provider1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}
}

// ==================== Genesis Tests ====================

func TestGenesisRoundtrip(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	params := types.Params{
		MinDeposit:            "2000000",
		MaxChannelDuration:    500000,
		DefaultSettlementFreq: 200,
		DisputePeriodBlocks:   800,
		ExpiryGracePeriod:     300,
		FraudPenaltyBps:       750,
		MaxChannelsPerPayer:   25,
	}
	k.SetParams(ctx, params)

	ch1 := types.PaymentChannel{
		ChannelId: "pc-gen-1", Payer: payer1, Provider: provider1,
		Deposited: "5000000", Spent: "1000000", Available: "4000000",
		Status: types.ChannelStatusOpen, Nonce: 3,
	}
	ch2 := types.PaymentChannel{
		ChannelId: "pc-gen-2", Payer: payer2, Provider: provider2,
		Deposited: "10000000", Spent: "0", Available: "10000000",
		Status: types.ChannelStatusClosed, Nonce: 0,
	}
	k.SetChannel(ctx, ch1)
	k.SetChannel(ctx, ch2)

	d1 := types.ChannelDispute{
		ChannelId: "pc-gen-1", Disputer: provider1,
		DisputedNonce: 2, DisputedSpent: "500000", DeadlineBlock: 200,
	}
	k.SetDispute(ctx, d1)

	gs := k.ExportGenesis(ctx)
	if len(gs.Channels) != 2 {
		t.Errorf("expected 2 channels, got %d", len(gs.Channels))
	}
	if len(gs.Disputes) != 1 {
		t.Errorf("expected 1 dispute, got %d", len(gs.Disputes))
	}
	if gs.Params.MinDeposit != "2000000" {
		t.Errorf("expected min deposit 2000000, got %s", gs.Params.MinDeposit)
	}

	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, gs)

	gotCh, found := k2.GetChannel(ctx2, "pc-gen-1")
	if !found {
		t.Fatal("channel pc-gen-1 not found after re-init")
	}
	if gotCh.Nonce != 3 {
		t.Errorf("expected nonce 3, got %d", gotCh.Nonce)
	}

	gotD, found := k2.GetDispute(ctx2, "pc-gen-1")
	if !found {
		t.Fatal("dispute not found after re-init")
	}
	if gotD.DisputedNonce != 2 {
		t.Errorf("expected disputed nonce 2, got %d", gotD.DisputedNonce)
	}
}

// ==================== Query Tests ====================

func TestQueryChannel(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	ch := types.PaymentChannel{ChannelId: "pc-q-1", Payer: payer1, Provider: provider1, Deposited: "5000000"}
	k.SetChannel(ctx, ch)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Channel(ctx, &types.QueryChannelRequest{ChannelId: "pc-q-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Channel.Deposited != "5000000" {
		t.Errorf("expected deposited 5000000, got %s", resp.Channel.Deposited)
	}
}

func TestQueryChannelNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Channel(ctx, &types.QueryChannelRequest{ChannelId: "nonexistent"})
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestQueryChannelsByPayer(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-qp-1", Payer: payer1, Provider: provider1})
	k.SetChannel(ctx, types.PaymentChannel{ChannelId: "pc-qp-2", Payer: payer1, Provider: provider2})

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.ChannelsByPayer(ctx, &types.QueryChannelsByPayerRequest{Payer: payer1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected total 2, got %d", resp.Total)
	}
}

func TestQueryActiveDispute(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	d := types.ChannelDispute{ChannelId: "pc-qd-1", DisputedNonce: 3, DeadlineBlock: 200}
	k.SetDispute(ctx, d)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.ActiveDispute(ctx, &types.QueryActiveDisputeRequest{ChannelId: "pc-qd-1"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.Found {
		t.Error("expected dispute found")
	}
	if resp.Dispute.DisputedNonce != 3 {
		t.Errorf("expected nonce 3, got %d", resp.Dispute.DisputedNonce)
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.DisputePeriodBlocks != 1000 {
		t.Errorf("expected dispute period 1000, got %d", resp.Params.DisputePeriodBlocks)
	}
}

// ==================== BeginBlock Tests ====================

func TestBeginBlock_ResolvesExpiredDisputes(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	ch := types.PaymentChannel{
		ChannelId: "pc-bb-1", Payer: payer1, Provider: provider1,
		Deposited: "5000000", Spent: "1000000", Available: "4000000",
		Status: types.ChannelStatusDisputed, Nonce: 0,
		DisputeDeadline: 150, DisputeNonce: 2,
	}
	k.SetChannel(ctx, ch)

	d := types.ChannelDispute{
		ChannelId: "pc-bb-1", Disputer: provider1,
		DisputedNonce: 2, DisputedSpent: "3000000",
		DeadlineBlock: 150, Resolved: false,
	}
	k.SetDispute(ctx, d)

	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}
	ctx = ctx.WithBlockHeight(200)

	// New behavior: auto-settle expired channels (closing or disputed)
	expired := k.GetExpiredChannels(ctx, uint64(ctx.BlockHeight()))
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired channel, got %d", len(expired))
	}
	for _, ech := range expired {
		k.AutoSettleChannel(ctx, ech)
	}

	updatedCh, _ := k.GetChannel(ctx, "pc-bb-1")
	if updatedCh.Status != types.ChannelStatusClosed {
		t.Errorf("expected closed status after auto-settle, got %s", updatedCh.Status)
	}
	if updatedCh.Spent != "3000000" {
		t.Errorf("expected spent 3000000, got %s", updatedCh.Spent)
	}
	if updatedCh.Nonce != 2 {
		t.Errorf("expected nonce 2, got %d", updatedCh.Nonce)
	}

	updatedD, _ := k.GetDispute(ctx, "pc-bb-1")
	if !updatedD.Resolved {
		t.Error("expected dispute to be resolved")
	}
}

// ======================== Integration: SpendFromChannel ========================

func TestSpendFromChannel_Success(t *testing.T) {
	k, ctx, mockBK := setupKeeper(t)
	mockBK.moduleBalances["channels"] = map[string]int64{"ulgm": 100000}

	ch := types.PaymentChannel{
		ChannelId:      "pc-spend-1",
		Payer:          payer1,
		Provider:       provider1,
		Deposited:      "10000",
		Spent:          "0",
		Available:      "10000",
		Status:         types.ChannelStatusOpen,
		OpenedAtBlock:  100,
		ExpiresAtBlock: 200,
	}
	k.SetChannel(ctx, ch)

	err := k.SpendFromChannel(ctx, "pc-spend-1", "3000", "tree")
	if err != nil {
		t.Fatalf("spend from channel: %v", err)
	}

	updated, found := k.GetChannel(ctx, "pc-spend-1")
	if !found {
		t.Fatal("channel not found after spend")
	}
	if updated.Spent != "3000" {
		t.Errorf("expected spent 3000, got %s", updated.Spent)
	}
	if updated.Available != "7000" {
		t.Errorf("expected available 7000, got %s", updated.Available)
	}
	if updated.Nonce != 1 {
		t.Errorf("expected nonce 1, got %d", updated.Nonce)
	}

	if mockBK.moduleBalances["channels"]["ulgm"] != 97000 {
		t.Errorf("expected channels module balance 97000, got %d", mockBK.moduleBalances["channels"]["ulgm"])
	}
	if mockBK.moduleBalances["tree"]["ulgm"] != 3000 {
		t.Errorf("expected tree module balance 3000, got %d", mockBK.moduleBalances["tree"]["ulgm"])
	}
}

func TestSpendFromChannel_InsufficientBalance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ch := types.PaymentChannel{
		ChannelId:      "pc-spend-2",
		Payer:          payer1,
		Provider:       provider1,
		Deposited:      "1000",
		Spent:          "800",
		Available:      "200",
		Status:         types.ChannelStatusOpen,
		OpenedAtBlock:  100,
		ExpiresAtBlock: 200,
	}
	k.SetChannel(ctx, ch)

	err := k.SpendFromChannel(ctx, "pc-spend-2", "500", "tree")
	if err == nil {
		t.Error("expected error for insufficient channel balance")
	}
}

func TestSpendFromChannel_ClosedChannel(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ch := types.PaymentChannel{
		ChannelId:      "pc-spend-3",
		Payer:          payer1,
		Provider:       provider1,
		Deposited:      "10000",
		Spent:          "0",
		Available:      "10000",
		Status:         types.ChannelStatusClosed,
		OpenedAtBlock:  100,
		ExpiresAtBlock: 200,
	}
	k.SetChannel(ctx, ch)

	err := k.SpendFromChannel(ctx, "pc-spend-3", "1000", "tree")
	if err == nil {
		t.Error("expected error for closed channel")
	}
}

func TestSpendFromChannel_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	err := k.SpendFromChannel(ctx, "nonexistent", "1000", "tree")
	if err == nil {
		t.Error("expected error for non-existent channel")
	}
}

func TestGetChannelInfo_Success(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	ch := types.PaymentChannel{
		ChannelId: "pc-info-1",
		Payer:     payer1,
		Provider:  provider1,
		Deposited: "5000",
		Spent:     "2000",
		Available: "3000",
		Status:    types.ChannelStatusOpen,
	}
	k.SetChannel(ctx, ch)

	payer, provider, available, status, found := k.GetChannelInfo(ctx, "pc-info-1")
	if !found {
		t.Fatal("expected channel to be found")
	}
	if payer != payer1 {
		t.Errorf("expected payer %s, got %s", payer1, payer)
	}
	if provider != provider1 {
		t.Errorf("expected provider %s, got %s", provider1, provider)
	}
	if available != "3000" {
		t.Errorf("expected available 3000, got %s", available)
	}
	if status != "open" {
		t.Errorf("expected status open, got %s", status)
	}
}

func TestGetChannelInfo_NotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, _, _, _, found := k.GetChannelInfo(ctx, "nonexistent")
	if found {
		t.Error("expected channel not found")
	}
}

// ==================== Signature Verification Tests ====================

func TestSignatureVerification_ValidSignature(t *testing.T) {
	initAddresses()
	payload := types.ChannelSigningPayload("update", testChainID, "pc-100-1", 5, "3000000")
	packed := signPacked(provider1Party, "update", "pc-100-1", 5, "3000000")

	err := types.VerifyPackedSignature(payload, packed, provider1Party.Addr)
	if err != nil {
		t.Fatalf("valid signature rejected: %v", err)
	}
}

func TestSignatureVerification_WrongAddress(t *testing.T) {
	initAddresses()
	payload := types.ChannelSigningPayload("update", testChainID, "pc-100-1", 5, "3000000")
	packed := signPacked(provider1Party, "update", "pc-100-1", 5, "3000000")

	err := types.VerifyPackedSignature(payload, packed, payer1Party.Addr) // wrong address
	if err == nil {
		t.Error("expected error: signature from wrong address should be rejected")
	}
}

func TestSignatureVerification_TamperedPayload(t *testing.T) {
	initAddresses()
	// Sign one payload, verify against different payload
	packed := signPacked(provider1Party, "update", "pc-100-1", 5, "3000000")
	tamperedPayload := types.ChannelSigningPayload("update", testChainID, "pc-100-1", 5, "9999999") // different amount

	err := types.VerifyPackedSignature(tamperedPayload, packed, provider1Party.Addr)
	if err == nil {
		t.Error("expected error: tampered payload should be rejected")
	}
}

func TestSignatureVerification_TruncatedSignature(t *testing.T) {
	initAddresses()
	payload := types.ChannelSigningPayload("update", testChainID, "pc-100-1", 5, "3000000")
	packed := signPacked(provider1Party, "update", "pc-100-1", 5, "3000000")

	err := types.VerifyPackedSignature(payload, packed[:50], provider1Party.Addr) // truncated
	if err == nil {
		t.Error("expected error: truncated signature should be rejected")
	}
}

func TestSignatureVerification_EmptySignature(t *testing.T) {
	initAddresses()
	payload := types.ChannelSigningPayload("update", testChainID, "pc-100-1", 5, "3000000")

	err := types.VerifyPackedSignature(payload, nil, provider1Party.Addr)
	if err == nil {
		t.Error("expected error: nil signature should be rejected")
	}

	err = types.VerifyPackedSignature(payload, []byte{}, provider1Party.Addr)
	if err == nil {
		t.Error("expected error: empty signature should be rejected")
	}
}

func TestSignatureVerification_HexRoundtrip(t *testing.T) {
	initAddresses()
	payload := types.ChannelSigningPayload("state", testChainID, "pc-100-1", 1, "2000000")
	hexSig := signHex(payer1Party, "state", "pc-100-1", 1, "2000000")

	err := types.VerifyHexPackedSignature(payload, hexSig, payer1Party.Addr)
	if err != nil {
		t.Fatalf("valid hex signature rejected: %v", err)
	}
}

// ==================== State Machine: Closing Flow Tests ====================

func TestStateMachine_DisputeDuringClosing(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Initiate unilateral close → closing
	msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosing {
		t.Fatalf("expected closing, got %s", ch.Status)
	}

	// Provider disputes during closing period with higher nonce
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: provider1, ChannelId: resp.ChannelId,
		Sequence: 3, ClaimedBalanceReceiver: "4000000",
		ProofSignature: signPacked(payer1Party, "dispute", resp.ChannelId, 3, "4000000"),
	})
	if err != nil {
		t.Fatalf("dispute during closing should succeed: %v", err)
	}

	ch, _ = k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusDisputed {
		t.Errorf("expected disputed, got %s", ch.Status)
	}
}

func TestStateMachine_CannotUpdateDuringClosing(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Initiate closing
	msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})

	// Try to update state during closing
	_, err := msgServer.UpdateChannel(ctx, &types.MsgUpdateChannel{
		Sender: payer1, ChannelId: resp.ChannelId,
		Sequence: 1, NewBalanceReceiver: "1000000",
		Signature: signPacked(provider1Party, "update", resp.ChannelId, 1, "1000000"),
	})
	if err == nil {
		t.Error("expected error: cannot update channel during closing")
	}
}

// ==================== Dispute Window Timing Tests ====================

func TestDisputeWindow_CannotSettleAtExactDeadline(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Dispute at block 100, deadline = 100 + 1000 = 1100
	msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: payer1, ChannelId: resp.ChannelId,
		Sequence: 1, ClaimedBalanceReceiver: "2000000",
		ProofSignature: signPacked(provider1Party, "dispute", resp.ChannelId, 1, "2000000"),
	})

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	deadline := ch.DisputeDeadline // 1100

	// At exact deadline: should be rejected (blockHeight <= deadline)
	ctx = ctx.WithBlockHeight(int64(deadline))
	_, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err == nil {
		t.Error("expected error: settlement at exact deadline should be rejected")
	}

	// One block after deadline: should succeed
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}
	ctx = ctx.WithBlockHeight(int64(deadline + 1))
	_, err = msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("settlement after deadline should succeed: %v", err)
	}
}

// ==================== Replay Protection Tests ====================

func TestReplay_SameChannelDifferentNonce(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Valid update at nonce 5
	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce: 5, Spent: "1000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 5, "1000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 5, "1000000"),
		},
	})
	if err != nil {
		t.Fatalf("initial update should succeed: %v", err)
	}

	// Replay the nonce-5 signature — sigs are valid but nonce guard rejects
	_, err = msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce: 5, Spent: "1000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 5, "1000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 5, "1000000"),
		},
	})
	if err == nil {
		t.Error("expected error: replayed nonce should be rejected")
	}
}

func TestReplay_SameNonceDifferentChannel(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 30000000)

	msgServer := keeper.NewFullMsgServerImpl(k)

	// Open channel A
	respA, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})
	// Open channel B
	respB, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Sign for channel A
	payerSigA := signHex(payer1Party, "state", respA.ChannelId, 1, "1000000")
	providerSigA := signHex(provider1Party, "state", respA.ChannelId, 1, "1000000")

	// Try to use channel A's signatures on channel B — channelId is in payload
	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: respB.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce: 1, Spent: "1000000",
			PayerSignature:    payerSigA,
			ProviderSignature: providerSigA,
		},
	})
	if err == nil {
		t.Error("expected error: signature from channel A should be rejected on channel B")
	}
}

func TestReplay_SameOperationDifferentChain(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Sign with a different chain ID (simulating testnet signature)
	wrongChainSig := signPackedWithChain(provider1Party, "update", "legible-testnet-1", resp.ChannelId, 1, "2000000")

	// Try to use the testnet signature on the current chain
	_, err := msgServer.UpdateChannel(ctx, &types.MsgUpdateChannel{
		Sender: payer1, ChannelId: resp.ChannelId,
		Sequence: 1, NewBalanceReceiver: "2000000",
		Signature: wrongChainSig,
	})
	if err == nil {
		t.Error("expected error: signature from different chain should be rejected")
	}
}

func TestReplay_UpdateStateDualSig(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Sign for nonce=1, spent=1000000
	payerSig1 := signHex(payer1Party, "state", resp.ChannelId, 1, "1000000")
	providerSig1 := signHex(provider1Party, "state", resp.ChannelId, 1, "1000000")

	// Try to use those sigs but with different spent amount — payload mismatch
	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce: 1, Spent: "2000000", // different from what was signed (1000000)
			PayerSignature:    payerSig1,
			ProviderSignature: providerSig1,
		},
	})
	if err == nil {
		t.Error("expected error: sigs signed for 1000000 should not validate against 2000000")
	}
}

// ==================== Cooperative Close Flow Tests ====================

func TestCooperativeClose_HappyPath(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Update state to reflect some spending
	msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{
			Nonce: 1, Spent: "2000000",
			PayerSignature:    signHex(payer1Party, "state", resp.ChannelId, 1, "2000000"),
			ProviderSignature: signHex(provider1Party, "state", resp.ChannelId, 1, "2000000"),
		},
	})

	// Step 1: Initiate settle → closing
	sresp, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("initiate close failed: %v", err)
	}
	if sresp.AmountToSender != "0" || sresp.AmountToReceiver != "0" {
		t.Error("expected zero distribution during closing period")
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosing {
		t.Fatalf("expected closing, got %s", ch.Status)
	}

	// Step 2: Wait past deadline
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}
	ctx = ctx.WithBlockHeight(int64(ch.DisputeDeadline + 1))

	// Step 3: Finalize → closed with fund distribution
	sresp, err = msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("finalize close failed: %v", err)
	}
	if sresp.AmountToSender != "3000000" {
		t.Errorf("expected payer refund 3000000, got %s", sresp.AmountToSender)
	}
	if sresp.AmountToReceiver != "2000000" {
		t.Errorf("expected provider payout 2000000, got %s", sresp.AmountToReceiver)
	}

	ch, _ = k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosed {
		t.Errorf("expected closed, got %s", ch.Status)
	}
}

func TestCooperativeClose_DisputeDuringClosing(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Initiate settle → closing
	msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})

	// Provider disputes during closing with proof of higher state
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: provider1, ChannelId: resp.ChannelId,
		Sequence: 3, ClaimedBalanceReceiver: "4000000",
		ProofSignature: signPacked(payer1Party, "dispute", resp.ChannelId, 3, "4000000"),
	})
	if err != nil {
		t.Fatalf("dispute during closing should succeed: %v", err)
	}

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusDisputed {
		t.Fatalf("expected disputed, got %s", ch.Status)
	}

	// Wait past new dispute deadline and settle
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}
	ctx = ctx.WithBlockHeight(int64(ch.DisputeDeadline + 1))

	sresp, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("settle after dispute should succeed: %v", err)
	}
	// Disputed state: provider claimed 4000000
	if sresp.AmountToReceiver != "4000000" {
		t.Errorf("expected receiver 4000000, got %s", sresp.AmountToReceiver)
	}
	if sresp.AmountToSender != "1000000" {
		t.Errorf("expected sender 1000000, got %s", sresp.AmountToSender)
	}
}

func TestCooperativeClose_CounterpartyCanDispute(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Provider initiates unilateral close
	msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: provider1, ChannelId: resp.ChannelId,
	})

	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosing {
		t.Fatalf("expected closing, got %s", ch.Status)
	}

	// Payer disputes with proof of an updated state
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: payer1, ChannelId: resp.ChannelId,
		Sequence: 5, ClaimedBalanceReceiver: "1000000",
		ProofSignature: signPacked(provider1Party, "dispute", resp.ChannelId, 5, "1000000"),
	})
	if err != nil {
		t.Fatalf("counterparty dispute should succeed: %v", err)
	}

	ch, _ = k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusDisputed {
		t.Errorf("expected disputed, got %s", ch.Status)
	}

	// Settle after deadline
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}
	ctx = ctx.WithBlockHeight(int64(ch.DisputeDeadline + 1))

	sresp, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("settle should succeed: %v", err)
	}
	if sresp.AmountToReceiver != "1000000" {
		t.Errorf("expected receiver 1000000, got %s", sresp.AmountToReceiver)
	}
	if sresp.AmountToSender != "4000000" {
		t.Errorf("expected sender 4000000, got %s", sresp.AmountToSender)
	}
}

func TestCooperativeClose_FundsDistribution(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Cooperative close with agreed final balance
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}

	_, err := msgServer.CloseChannel(ctx, &types.MsgCloseChannel{
		Closer:                payer1,
		ChannelId:             resp.ChannelId,
		Sequence:              1,
		FinalBalanceReceiver:  "3000000",
		CounterpartySignature: signPacked(provider1Party, "close", resp.ChannelId, 1, "3000000"),
	})
	if err != nil {
		t.Fatalf("cooperative close failed: %v", err)
	}

	// Verify channel is closed
	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosed {
		t.Errorf("expected closed, got %s", ch.Status)
	}

	// Verify bank balances
	// Provider got 3000000 payout
	if bk.balances[provider1]["ulgm"] != 3000000 {
		t.Errorf("expected provider balance 3000000, got %d", bk.balances[provider1]["ulgm"])
	}
	// Payer: started 20000000, deposited 5000000 → 15000000, refund 2000000 → 17000000
	payerExpected := int64(20000000 - 5000000 + 2000000)
	if bk.balances[payer1]["ulgm"] != payerExpected {
		t.Errorf("expected payer balance %d, got %d", payerExpected, bk.balances[payer1]["ulgm"])
	}
}

// ==================== BeginBlock Auto-Resolution Tests ====================

func TestBeginBlock_AutoResolvesExpiredClosing(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Manually create a channel in closing state past deadline
	ch := types.PaymentChannel{
		ChannelId: "pc-auto-1", Payer: payer1, Provider: provider1,
		Deposited: "5000000", Spent: "2000000", Available: "3000000",
		Status: types.ChannelStatusClosing, DisputeDeadline: 150, Nonce: 3,
	}
	k.SetChannel(ctx, ch)
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}

	ctx = ctx.WithBlockHeight(200) // past deadline of 150

	// Simulate BeginBlock: get expired channels and auto-settle
	expired := k.GetExpiredChannels(ctx, uint64(ctx.BlockHeight()))
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired channel, got %d", len(expired))
	}
	for _, ech := range expired {
		k.AutoSettleChannel(ctx, ech)
	}

	// Verify channel is closed
	updated, _ := k.GetChannel(ctx, "pc-auto-1")
	if updated.Status != types.ChannelStatusClosed {
		t.Errorf("expected closed, got %s", updated.Status)
	}

	// Verify funds distributed: payer gets 3000000 refund, provider gets 2000000
	if bk.balances[payer1]["ulgm"] != 3000000 {
		t.Errorf("expected payer balance 3000000, got %d", bk.balances[payer1]["ulgm"])
	}
	if bk.balances[provider1]["ulgm"] != 2000000 {
		t.Errorf("expected provider balance 2000000, got %d", bk.balances[provider1]["ulgm"])
	}
}

func TestBeginBlock_AutoResolvesExpiredDispute(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Manually create a disputed channel past deadline
	ch := types.PaymentChannel{
		ChannelId: "pc-auto-2", Payer: payer1, Provider: provider1,
		Deposited: "5000000", Spent: "1000000", Available: "4000000",
		Status: types.ChannelStatusDisputed, DisputeDeadline: 150,
		Nonce: 1, DisputeNonce: 3,
	}
	k.SetChannel(ctx, ch)

	d := types.ChannelDispute{
		ChannelId: "pc-auto-2", Disputer: provider1,
		DisputedNonce: 3, DisputedSpent: "3000000",
		DeadlineBlock: 150, Resolved: false,
	}
	k.SetDispute(ctx, d)

	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}
	ctx = ctx.WithBlockHeight(200) // past deadline

	// Simulate BeginBlock
	expired := k.GetExpiredChannels(ctx, uint64(ctx.BlockHeight()))
	if len(expired) != 1 {
		t.Fatalf("expected 1 expired channel, got %d", len(expired))
	}
	for _, ech := range expired {
		k.AutoSettleChannel(ctx, ech)
	}

	// Verify channel closed with disputed state
	updated, _ := k.GetChannel(ctx, "pc-auto-2")
	if updated.Status != types.ChannelStatusClosed {
		t.Errorf("expected closed, got %s", updated.Status)
	}
	if updated.Spent != "3000000" {
		t.Errorf("expected spent to be disputed amount 3000000, got %s", updated.Spent)
	}

	// Verify dispute resolved
	updatedD, _ := k.GetDispute(ctx, "pc-auto-2")
	if !updatedD.Resolved {
		t.Error("expected dispute to be resolved")
	}

	// Verify funds: payer gets 2000000 (5M-3M), provider gets 3000000
	if bk.balances[payer1]["ulgm"] != 2000000 {
		t.Errorf("expected payer balance 2000000, got %d", bk.balances[payer1]["ulgm"])
	}
	if bk.balances[provider1]["ulgm"] != 3000000 {
		t.Errorf("expected provider balance 3000000, got %d", bk.balances[provider1]["ulgm"])
	}
}
