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

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx, mockBK
}

func testAddr(name string) string {
	return sdk.AccAddress([]byte(name + "_address_padding")).String()
}

var (
	payer1    string
	payer2    string
	provider1 string
	provider2 string
)

func initAddresses() {
	payer1 = testAddr("payer001")
	payer2 = testAddr("payer002")
	provider1 = testAddr("provider1")
	provider2 = testAddr("provider2")
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

	// Verify channel stored
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

	// Set max channels to 2
	params := types.DefaultParams()
	params.MaxChannelsPerPayer = 2
	k.SetParams(ctx, *params)

	msgServer := keeper.NewFullMsgServerImpl(k)
	// Open 2 channels
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

	// Third should fail
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
		Depositor: provider1, // not payer
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

// ==================== UpdateState Tests ====================

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
			ChannelId: resp.ChannelId,
			Nonce:     1,
			Spent:     "2000000",
			StateHash: "abc123",
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
		Update: types.ChannelStateUpdate{Nonce: 1, Spent: "1000000"},
	})

	// Try nonce 1 again (not higher)
	_, err := msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{Nonce: 1, Spent: "2000000"},
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
		Update: types.ChannelStateUpdate{Nonce: 1, Spent: "99999999"},
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

// ==================== DisputeChannel Tests ====================

func TestDisputeChannel_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer:               payer1,
		ChannelId:              resp.ChannelId,
		Sequence:               1,
		ClaimedBalanceReceiver: "2000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify channel is disputed
	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusDisputed {
		t.Errorf("expected disputed status, got %s", ch.Status)
	}

	// Verify dispute record
	d, found := k.GetDispute(ctx, resp.ChannelId)
	if !found {
		t.Fatal("dispute not found")
	}
	if d.DisputedNonce != 1 {
		t.Errorf("expected nonce 1, got %d", d.DisputedNonce)
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
		Disputer: payer1, ChannelId: resp.ChannelId, Sequence: 1, ClaimedBalanceReceiver: "1000000",
	})

	// Second dispute with same sequence (should fail)
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: provider1, ChannelId: resp.ChannelId, Sequence: 1, ClaimedBalanceReceiver: "500000",
	})
	if err == nil {
		t.Error("expected already disputed error")
	}
}

// ==================== SettleChannel Tests ====================

func TestSettleChannel_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Update state to spend some
	msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{Nonce: 1, Spent: "2000000"},
	})

	// Fund module account for distribution
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}

	// Settle
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

	// Verify closed
	ch, _ := k.GetChannel(ctx, resp.ChannelId)
	if ch.Status != types.ChannelStatusClosed {
		t.Errorf("expected closed status, got %s", ch.Status)
	}
}

func TestSettleChannel_AfterDispute(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 10000,
	})

	// Dispute with sequence 1 and claimed balance receiver 3000000
	msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: provider1, ChannelId: resp.ChannelId,
		Sequence: 1, ClaimedBalanceReceiver: "3000000",
	})

	// Advance past deadline
	ctx = ctx.WithBlockHeight(int64(100 + 1001)) // past dispute period

	// Fund module account
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}

	// Settle
	sresp, err := msgServer.SettleChannel(ctx, &types.MsgSettleChannel{
		Authority: payer1, ChannelId: resp.ChannelId,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Disputed state: spent=3000000, refund=2000000
	if sresp.AmountToSender != "2000000" {
		t.Errorf("expected amount to sender 2000000, got %s", sresp.AmountToSender)
	}
	if sresp.AmountToReceiver != "3000000" {
		t.Errorf("expected amount to receiver 3000000, got %s", sresp.AmountToReceiver)
	}
}

// ==================== CloseChannel Tests ====================

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
		CounterpartySignature: []byte("sig_provider"),
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

// ==================== ClaimExpired Tests ====================

func TestClaimExpired_Success(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(payer1, "ulgm", 20000000)

	msgServer := keeper.NewFullMsgServerImpl(k)
	resp, _ := msgServer.OpenChannel(ctx, &types.MsgOpenChannel{
		Sender: payer1, Receiver: provider1, Deposit: "5000000", TimeoutHeight: 100,
	})

	// Update state to spend some
	msgServer.UpdateState(ctx, &types.MsgUpdateState{
		Sender: payer1, ChannelId: resp.ChannelId,
		Update: types.ChannelStateUpdate{Nonce: 1, Spent: "1000000"},
	})

	// Fund module account
	bk.moduleBalances[types.ModuleName] = map[string]int64{"ulgm": 5000000}

	// Advance past expiry + grace period (100 + 100 + 500 = 701)
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

	// Don't advance past expiry
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

	// Advance past expiry
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
		Sender:              payer1,
		ChannelId:           resp.ChannelId,
		Sequence:            1,
		NewBalanceReceiver:  "2000000",
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
	})

	// Counter-dispute with higher sequence should succeed
	_, err := msgServer.DisputeChannel(ctx, &types.MsgDisputeChannel{
		Disputer: provider1, ChannelId: resp.ChannelId,
		Sequence: 5, ClaimedBalanceReceiver: "3000000",
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

	// Set some data
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

	// Export
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

	// Re-init in fresh context
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, gs)

	// Verify data round-tripped
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
	k, ctx, _ := setupKeeper(t)

	// Create a channel in disputed state
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

	// Advance past deadline
	ctx = ctx.WithBlockHeight(200)

	// Simulate BeginBlock
	disputes := k.GetExpiredDisputes(ctx, uint64(ctx.BlockHeight()))
	if len(disputes) != 1 {
		t.Fatalf("expected 1 expired dispute, got %d", len(disputes))
	}

	// Resolve (as module.go BeginBlock would)
	for _, dispute := range disputes {
		gotCh, found := k.GetChannel(ctx, dispute.ChannelId)
		if !found {
			continue
		}
		gotCh.Spent = dispute.DisputedSpent
		gotCh.Nonce = dispute.DisputedNonce
		gotCh.Status = types.ChannelStatusOpen
		gotCh.DisputeDeadline = 0
		k.SetChannel(ctx, gotCh)

		dispute.Resolved = true
		dispute.Resolution = "timeout"
		k.SetDispute(ctx, dispute)
	}

	// Verify resolved
	updatedCh, _ := k.GetChannel(ctx, "pc-bb-1")
	if updatedCh.Status != types.ChannelStatusOpen {
		t.Errorf("expected open status after resolve, got %s", updatedCh.Status)
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

	// Fund the channels module account so it can transfer
	mockBK.moduleBalances["channels"] = map[string]int64{"ulgm": 100000}

	// Create a channel
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

	// Spend from channel
	err := k.SpendFromChannel(ctx, "pc-spend-1", "3000", "tree")
	if err != nil {
		t.Fatalf("spend from channel: %v", err)
	}

	// Verify channel state updated
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

	// Verify module balance transfer
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

	// Try to spend more than available
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
