package keeper_test

import (
	"context"
	"fmt"
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

	"github.com/legible-money/legible/x/claiming_pot/keeper"
	"github.com/legible-money/legible/x/claiming_pot/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

// ---------- Mock BankKeeper ----------

type mockBankKeeper struct {
	balances       map[string]map[string]int64 // addr -> denom -> amount
	moduleBalances map[string]map[string]int64 // module -> denom -> amount
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

func (m *mockBankKeeper) setModuleBalance(module, denom string, amount int64) {
	if m.moduleBalances[module] == nil {
		m.moduleBalances[module] = make(map[string]int64)
	}
	m.moduleBalances[module][denom] = amount
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

	k := keeper.NewKeeper(cdc, storeKey, mockBK, "lgm1authority")

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx, mockBK
}

// Test addresses — generated lazily after init() sets lgm bech32 prefix
func testAddr(name string) string {
	return sdk.AccAddress([]byte(name + "_address_padding")).String()
}

var (
	creator1 string
	creator2 string
	claimer1 string
	claimer2 string
	funder1  string
)

func initAddresses() {
	creator1 = testAddr("creator1")
	creator2 = testAddr("creator2")
	claimer1 = testAddr("claimer1")
	claimer2 = testAddr("claimer2")
	funder1 = testAddr("funder01")
}

// ==================== Params Tests ====================

func TestDefaultParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	params := k.GetParams(ctx)
	if params.MaxPotsPerCreator != 100 {
		t.Errorf("expected max pots 100, got %d", params.MaxPotsPerCreator)
	}
	if params.MinInitialDeposit != "1000000" {
		t.Errorf("expected min deposit 1000000, got %s", params.MinInitialDeposit)
	}
}

func TestSetGetParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	custom := types.Params{
		MaxPotsPerCreator: 50,
		MinInitialDeposit: "5000000",
		MaxAllowlistSize:  500,
		MinAmountPerClaim: "2000",
	}
	k.SetParams(ctx, custom)

	got := k.GetParams(ctx)
	if got.MaxPotsPerCreator != 50 {
		t.Errorf("expected 50, got %d", got.MaxPotsPerCreator)
	}
	if got.MinInitialDeposit != "5000000" {
		t.Errorf("expected 5000000, got %s", got.MinInitialDeposit)
	}
}

// ==================== Pot CRUD Tests ====================

func TestSetGetPot(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	pot := types.ClaimingPot{
		PotId:          "pot-1",
		Creator:        creator1,
		Balance:        "1000000",
		TotalDeposited: "1000000",
		TotalClaimed:   "0",
		Status:         types.PotStatusActive,
		Config: types.PotConfig{
			AmountPerClaim:      "1000",
			MaxClaims:           100,
			MaxClaimsPerAccount: 1,
		},
	}

	k.SetPot(ctx, pot)

	got, found := k.GetPot(ctx, "pot-1")
	if !found {
		t.Fatal("expected pot to be found")
	}
	if got.PotId != "pot-1" {
		t.Errorf("expected pot-1, got %s", got.PotId)
	}
	if got.Balance != "1000000" {
		t.Errorf("expected 1000000, got %s", got.Balance)
	}
}

func TestGetPotNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, found := k.GetPot(ctx, "nonexistent")
	if found {
		t.Error("expected pot not found")
	}
}

func TestDeletePot(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	pot := types.ClaimingPot{PotId: "pot-del", Creator: creator1, Status: types.PotStatusActive}
	k.SetPot(ctx, pot)

	k.DeletePot(ctx, "pot-del")

	_, found := k.GetPot(ctx, "pot-del")
	if found {
		t.Error("expected pot to be deleted")
	}
}

func TestIteratePots(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	for i := 0; i < 5; i++ {
		pot := types.ClaimingPot{
			PotId:   fmt.Sprintf("iter-%d", i),
			Creator: creator1,
			Status:  types.PotStatusActive,
		}
		k.SetPot(ctx, pot)
	}

	count := 0
	k.IteratePots(ctx, func(pot types.ClaimingPot) bool {
		count++
		return false
	})
	if count != 5 {
		t.Errorf("expected 5 pots, got %d", count)
	}
}

func TestGetPotsByStatus(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetPot(ctx, types.ClaimingPot{PotId: "active1", Status: types.PotStatusActive})
	k.SetPot(ctx, types.ClaimingPot{PotId: "active2", Status: types.PotStatusActive})
	k.SetPot(ctx, types.ClaimingPot{PotId: "closed1", Status: types.PotStatusClosed})

	active := k.GetPotsByStatus(ctx, types.PotStatusActive)
	if len(active) != 2 {
		t.Errorf("expected 2 active pots, got %d", len(active))
	}

	closed := k.GetPotsByStatus(ctx, types.PotStatusClosed)
	if len(closed) != 1 {
		t.Errorf("expected 1 closed pot, got %d", len(closed))
	}
}

func TestGetPotsByCreator(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetPot(ctx, types.ClaimingPot{PotId: "c1p1", Creator: creator1, Status: types.PotStatusActive})
	k.SetPot(ctx, types.ClaimingPot{PotId: "c1p2", Creator: creator1, Status: types.PotStatusActive})
	k.SetPot(ctx, types.ClaimingPot{PotId: "c2p1", Creator: creator2, Status: types.PotStatusActive})

	c1pots := k.GetPotsByCreator(ctx, creator1)
	if len(c1pots) != 2 {
		t.Errorf("expected 2 pots for creator1, got %d", len(c1pots))
	}

	c2pots := k.GetPotsByCreator(ctx, creator2)
	if len(c2pots) != 1 {
		t.Errorf("expected 1 pot for creator2, got %d", len(c2pots))
	}
}

// ==================== Claim Record Tests ====================

func TestSetGetClaimRecord(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	record := types.ClaimRecord{
		PotId:          "pot-1",
		Claimer:        claimer1,
		ClaimCount:     1,
		TotalAmount:    "1000",
		LastClaimBlock: 100,
	}

	k.SetClaimRecord(ctx, record)

	got, found := k.GetClaimRecord(ctx, "pot-1", claimer1)
	if !found {
		t.Fatal("expected claim record to be found")
	}
	if got.ClaimCount != 1 {
		t.Errorf("expected claim count 1, got %d", got.ClaimCount)
	}
	if got.TotalAmount != "1000" {
		t.Errorf("expected 1000, got %s", got.TotalAmount)
	}
}

func TestGetClaimRecordNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	_, found := k.GetClaimRecord(ctx, "pot-1", "nonexistent")
	if found {
		t.Error("expected claim record not found")
	}
}

func TestGetAllClaimRecords(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetClaimRecord(ctx, types.ClaimRecord{PotId: "pot-1", Claimer: claimer1, ClaimCount: 1, TotalAmount: "100"})
	k.SetClaimRecord(ctx, types.ClaimRecord{PotId: "pot-1", Claimer: claimer2, ClaimCount: 2, TotalAmount: "200"})

	records := k.GetAllClaimRecords(ctx, "pot-1")
	if len(records) != 2 {
		t.Errorf("expected 2 records, got %d", len(records))
	}
}

// ==================== Creator Pot Count Tests ====================

func TestCreatorPotCount(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	count := k.GetCreatorPotCount(ctx, creator1)
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}

	k.IncrementCreatorPotCount(ctx, creator1)
	k.IncrementCreatorPotCount(ctx, creator1)
	k.IncrementCreatorPotCount(ctx, creator1)

	count = k.GetCreatorPotCount(ctx, creator1)
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

// ==================== Pot ID Generation ====================

func TestGetNextPotId(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	id1 := k.GetNextPotId(ctx)
	if id1 != "pot-1" {
		t.Errorf("expected pot-1, got %s", id1)
	}

	id2 := k.GetNextPotId(ctx)
	if id2 != "pot-2" {
		t.Errorf("expected pot-2, got %s", id2)
	}
}

// ==================== MsgServer: CreatePot ====================

func TestCreatePot(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 10_000_000) // 10 LGM

	msgServer := keeper.NewMsgServerImpl(k)
	resp, err := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           10,
			MaxClaimsPerAccount: 1,
			CreatorCanClose:     true,
		},
		InitialDeposit: "5000000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.PotId == "" {
		t.Error("expected non-empty pot ID")
	}

	// Verify pot was stored
	pot, found := k.GetPot(ctx, resp.PotId)
	if !found {
		t.Fatal("pot not found after creation")
	}
	if pot.Balance != "5000000" {
		t.Errorf("expected balance 5000000, got %s", pot.Balance)
	}
	if pot.Status != types.PotStatusActive {
		t.Errorf("expected active status, got %s", pot.Status)
	}

	// Verify bank transfer
	if bk.moduleBalances[types.ModuleName]["ulgm"] != 5_000_000 {
		t.Errorf("expected module balance 5000000, got %d", bk.moduleBalances[types.ModuleName]["ulgm"])
	}
}

func TestCreatePotInsufficientDeposit(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 10_000_000)

	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "100",
			MaxClaims:           10,
			MaxClaimsPerAccount: 1,
		},
		InitialDeposit: "500", // Below MinInitialDeposit (1000000)
	})
	if err == nil {
		t.Error("expected error for insufficient deposit")
	}
}

func TestCreatePotMaxPotsExceeded(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 1_000_000_000)

	// Set max pots per creator to 2
	params := types.DefaultParams()
	params.MaxPotsPerCreator = 2
	k.SetParams(ctx, params)

	msgServer := keeper.NewMsgServerImpl(k)

	// Create 2 pots
	for i := 0; i < 2; i++ {
		_, err := msgServer.CreatePot(ctx, &types.MsgCreatePot{
			Creator: creator1,
			Config: types.PotConfig{
				AmountPerClaim:      "1000",
				MaxClaims:           10,
				MaxClaimsPerAccount: 1,
			},
			InitialDeposit: "1000000",
		})
		if err != nil {
			t.Fatalf("unexpected error on pot %d: %v", i+1, err)
		}
	}

	// Third should fail
	_, err := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "1000",
			MaxClaims:           10,
			MaxClaimsPerAccount: 1,
		},
		InitialDeposit: "1000000",
	})
	if err == nil {
		t.Error("expected error for max pots exceeded")
	}
}

func TestCreatePotInvalidConfig(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 10_000_000)

	msgServer := keeper.NewMsgServerImpl(k)

	// Zero amount per claim
	_, err := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "0",
			MaxClaims:           10,
			MaxClaimsPerAccount: 1,
		},
		InitialDeposit: "1000000",
	})
	if err == nil {
		t.Error("expected error for zero amount per claim")
	}

	// Zero max claims per account
	_, err = msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "1000",
			MaxClaims:           10,
			MaxClaimsPerAccount: 0,
		},
		InitialDeposit: "1000000",
	})
	if err == nil {
		t.Error("expected error for zero max claims per account")
	}
}

// ==================== MsgServer: FundPot ====================

func TestFundPot(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 10_000_000)
	bk.setBalance(funder1, "ulgm", 5_000_000)

	msgServer := keeper.NewMsgServerImpl(k)

	resp, err := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           100,
			MaxClaimsPerAccount: 1,
		},
		InitialDeposit: "2000000",
	})
	if err != nil {
		t.Fatalf("create pot failed: %v", err)
	}

	// Fund the pot
	_, err = msgServer.FundPot(ctx, &types.MsgFundPot{
		Funder: funder1,
		PotId:  resp.PotId,
		Amount: "3000000",
	})
	if err != nil {
		t.Fatalf("fund pot failed: %v", err)
	}

	pot, _ := k.GetPot(ctx, resp.PotId)
	if pot.Balance != "5000000" {
		t.Errorf("expected balance 5000000, got %s", pot.Balance)
	}
	if pot.TotalDeposited != "5000000" {
		t.Errorf("expected total deposited 5000000, got %s", pot.TotalDeposited)
	}
}

func TestFundPotReactivatesExhausted(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Create an exhausted pot directly
	pot := types.ClaimingPot{
		PotId:          "pot-exhausted",
		Creator:        creator1,
		Balance:        "0",
		TotalDeposited: "1000000",
		TotalClaimed:   "1000000",
		Status:         types.PotStatusExhausted,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           100,
			MaxClaimsPerAccount: 1,
		},
	}
	k.SetPot(ctx, pot)
	bk.setBalance(funder1, "ulgm", 500_000)

	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.FundPot(ctx, &types.MsgFundPot{
		Funder: funder1,
		PotId:  "pot-exhausted",
		Amount: "500000",
	})
	if err != nil {
		t.Fatalf("fund pot failed: %v", err)
	}

	got, _ := k.GetPot(ctx, "pot-exhausted")
	if got.Status != types.PotStatusActive {
		t.Errorf("expected active status after refunding, got %s", got.Status)
	}
}

func TestFundClosedPotFails(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(funder1, "ulgm", 1_000_000)

	pot := types.ClaimingPot{
		PotId:   "pot-closed",
		Creator: creator1,
		Status:  types.PotStatusClosed,
		Config:  types.PotConfig{AmountPerClaim: "1000", MaxClaimsPerAccount: 1},
	}
	k.SetPot(ctx, pot)

	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.FundPot(ctx, &types.MsgFundPot{
		Funder: funder1,
		PotId:  "pot-closed",
		Amount: "500000",
	})
	if err == nil {
		t.Error("expected error for funding closed pot")
	}
}

// ==================== MsgServer: ClaimFromPot ====================

func TestClaimFromPot(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 10_000_000)

	msgServer := keeper.NewMsgServerImpl(k)

	createResp, err := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           10,
			MaxClaimsPerAccount: 1,
		},
		InitialDeposit: "1000000",
	})
	if err != nil {
		t.Fatalf("create pot failed: %v", err)
	}

	claimResp, err := msgServer.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		Claimer: claimer1,
		PotId:   createResp.PotId,
	})
	if err != nil {
		t.Fatalf("claim failed: %v", err)
	}
	if claimResp.Amount != "100000" {
		t.Errorf("expected claim amount 100000, got %s", claimResp.Amount)
	}

	// Verify pot state
	pot, _ := k.GetPot(ctx, createResp.PotId)
	if pot.ClaimCount != 1 {
		t.Errorf("expected claim count 1, got %d", pot.ClaimCount)
	}
	if pot.Balance != "900000" {
		t.Errorf("expected balance 900000, got %s", pot.Balance)
	}

	// Verify claim record
	record, found := k.GetClaimRecord(ctx, createResp.PotId, claimer1)
	if !found {
		t.Fatal("expected claim record")
	}
	if record.ClaimCount != 1 {
		t.Errorf("expected record count 1, got %d", record.ClaimCount)
	}
}

func TestClaimFromPotMaxClaimsPerAccount(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 10_000_000)

	msgServer := keeper.NewMsgServerImpl(k)

	createResp, _ := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           100,
			MaxClaimsPerAccount: 1,
		},
		InitialDeposit: "5000000",
	})

	// First claim succeeds
	_, err := msgServer.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		Claimer: claimer1,
		PotId:   createResp.PotId,
	})
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}

	// Second claim should fail
	_, err = msgServer.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		Claimer: claimer1,
		PotId:   createResp.PotId,
	})
	if err == nil {
		t.Error("expected error for exceeding max claims per account")
	}
}

func TestClaimFromPotExhaustion(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	// Create pot with exactly 2 claims worth of balance
	pot := types.ClaimingPot{
		PotId:          "pot-exhaust",
		Creator:        creator1,
		Balance:        "200000",
		TotalDeposited: "200000",
		TotalClaimed:   "0",
		ClaimCount:     0,
		Status:         types.PotStatusActive,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           0, // unlimited
			MaxClaimsPerAccount: 1,
		},
	}
	k.SetPot(ctx, pot)
	bk.setModuleBalance(types.ModuleName, "ulgm", 200000)

	msgServer := keeper.NewMsgServerImpl(k)

	// First claim succeeds
	_, err := msgServer.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		Claimer: claimer1,
		PotId:   "pot-exhaust",
	})
	if err != nil {
		t.Fatalf("first claim failed: %v", err)
	}

	// Second claim succeeds and exhausts pot
	_, err = msgServer.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		Claimer: claimer2,
		PotId:   "pot-exhaust",
	})
	if err != nil {
		t.Fatalf("second claim failed: %v", err)
	}

	// Pot should be exhausted
	got, _ := k.GetPot(ctx, "pot-exhaust")
	if got.Status != types.PotStatusExhausted {
		t.Errorf("expected exhausted status, got %s", got.Status)
	}
}

func TestClaimFromInactivePot(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	pot := types.ClaimingPot{
		PotId:   "pot-inactive",
		Creator: creator1,
		Status:  types.PotStatusClosed,
		Config:  types.PotConfig{AmountPerClaim: "1000", MaxClaimsPerAccount: 1},
	}
	k.SetPot(ctx, pot)

	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		Claimer: claimer1,
		PotId:   "pot-inactive",
	})
	if err == nil {
		t.Error("expected error for claiming from inactive pot")
	}
}

func TestClaimFromExpiredPot(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Pot expired at block 50, current block is 100
	pot := types.ClaimingPot{
		PotId:   "pot-expired",
		Creator: creator1,
		Balance: "1000000",
		Status:  types.PotStatusActive,
		Config: types.PotConfig{
			AmountPerClaim:      "1000",
			MaxClaimsPerAccount: 1,
			ExpiresAtBlock:      50,
		},
	}
	k.SetPot(ctx, pot)

	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		Claimer: claimer1,
		PotId:   "pot-expired",
	})
	if err == nil {
		t.Error("expected error for claiming from expired pot")
	}

	// Verify pot status was updated
	got, _ := k.GetPot(ctx, "pot-expired")
	if got.Status != types.PotStatusExpired {
		t.Errorf("expected expired status, got %s", got.Status)
	}
}

func TestClaimFromPotMaxTotalClaims(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	pot := types.ClaimingPot{
		PotId:          "pot-maxclaims",
		Creator:        creator1,
		Balance:        "1000000",
		TotalDeposited: "1000000",
		TotalClaimed:   "0",
		ClaimCount:     1,
		Status:         types.PotStatusActive,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           1, // only 1 total
			MaxClaimsPerAccount: 1,
		},
	}
	k.SetPot(ctx, pot)
	bk.setModuleBalance(types.ModuleName, "ulgm", 1000000)

	msgServer := keeper.NewMsgServerImpl(k)
	_, err := msgServer.ClaimFromPot(ctx, &types.MsgClaimFromPot{
		Claimer: claimer1,
		PotId:   "pot-maxclaims",
	})
	if err == nil {
		t.Error("expected error for max total claims reached")
	}
}

// ==================== MsgServer: ClosePot ====================

func TestClosePot(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 10_000_000)

	msgServer := keeper.NewMsgServerImpl(k)

	createResp, _ := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           10,
			MaxClaimsPerAccount: 1,
			CreatorCanClose:     true,
		},
		InitialDeposit: "5000000",
	})

	closeResp, err := msgServer.ClosePot(ctx, &types.MsgClosePot{
		Creator: creator1,
		PotId:   createResp.PotId,
	})
	if err != nil {
		t.Fatalf("close pot failed: %v", err)
	}
	if closeResp.RefundedAmount != "5000000" {
		t.Errorf("expected refund 5000000, got %s", closeResp.RefundedAmount)
	}

	pot, _ := k.GetPot(ctx, createResp.PotId)
	if pot.Status != types.PotStatusClosed {
		t.Errorf("expected closed status, got %s", pot.Status)
	}
	if pot.Balance != "0" {
		t.Errorf("expected zero balance, got %s", pot.Balance)
	}
}

func TestClosePotNotCreator(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 10_000_000)

	msgServer := keeper.NewMsgServerImpl(k)

	createResp, _ := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           10,
			MaxClaimsPerAccount: 1,
			CreatorCanClose:     true,
		},
		InitialDeposit: "5000000",
	})

	_, err := msgServer.ClosePot(ctx, &types.MsgClosePot{
		Creator: creator2, // wrong creator
		PotId:   createResp.PotId,
	})
	if err == nil {
		t.Error("expected error for non-creator close attempt")
	}
}

func TestClosePotNotAllowed(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	bk.setBalance(creator1, "ulgm", 10_000_000)

	msgServer := keeper.NewMsgServerImpl(k)

	createResp, _ := msgServer.CreatePot(ctx, &types.MsgCreatePot{
		Creator: creator1,
		Config: types.PotConfig{
			AmountPerClaim:      "100000",
			MaxClaims:           10,
			MaxClaimsPerAccount: 1,
			CreatorCanClose:     false, // not allowed
		},
		InitialDeposit: "5000000",
	})

	_, err := msgServer.ClosePot(ctx, &types.MsgClosePot{
		Creator: creator1,
		PotId:   createResp.PotId,
	})
	if err == nil {
		t.Error("expected error when creator close not allowed")
	}
}

// ==================== Eligibility Tests ====================

func TestEligibilityAllowlist(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	pot := types.ClaimingPot{
		PotId:   "pot-allow",
		Creator: creator1,
		Status:  types.PotStatusActive,
		Config: types.PotConfig{
			AmountPerClaim:      "1000",
			MaxClaimsPerAccount: 1,
			Eligibility: types.PotEligibility{
				Allowlist: []string{claimer1},
			},
		},
	}

	eligible, _ := k.EvaluateEligibility(ctx, pot, claimer1)
	if !eligible {
		t.Error("expected claimer1 to be eligible (in allowlist)")
	}

	eligible, reason := k.EvaluateEligibility(ctx, pot, claimer2)
	if eligible {
		t.Error("expected claimer2 to be ineligible (not in allowlist)")
	}
	if reason != "address not in allowlist" {
		t.Errorf("unexpected reason: %s", reason)
	}
}

func TestEligibilityNoRestrictions(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	pot := types.ClaimingPot{
		PotId:   "pot-open",
		Creator: creator1,
		Status:  types.PotStatusActive,
		Config: types.PotConfig{
			AmountPerClaim:      "1000",
			MaxClaimsPerAccount: 1,
			Eligibility:         types.PotEligibility{},
		},
	}

	eligible, _ := k.EvaluateEligibility(ctx, pot, claimer1)
	if !eligible {
		t.Error("expected anyone to be eligible with no restrictions")
	}
}

func TestEligibilityMaxBalance(t *testing.T) {
	k, ctx, bk := setupKeeper(t)

	bk.setBalance(claimer1, "ulgm", 500_000)   // under limit
	bk.setBalance(claimer2, "ulgm", 2_000_000) // over limit

	pot := types.ClaimingPot{
		PotId:   "pot-maxbal",
		Creator: creator1,
		Status:  types.PotStatusActive,
		Config: types.PotConfig{
			AmountPerClaim:      "1000",
			MaxClaimsPerAccount: 1,
			Eligibility: types.PotEligibility{
				MaxBalance: "1000000",
			},
		},
	}

	eligible, _ := k.EvaluateEligibility(ctx, pot, claimer1)
	if !eligible {
		t.Error("expected claimer1 to be eligible (under max balance)")
	}

	eligible, _ = k.EvaluateEligibility(ctx, pot, claimer2)
	if eligible {
		t.Error("expected claimer2 to be ineligible (over max balance)")
	}
}

// ==================== Genesis Tests ====================

func TestInitExportGenesis(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	gs := types.GenesisState{
		Params: types.Params{
			MaxPotsPerCreator: 42,
			MinInitialDeposit: "999",
			MaxAllowlistSize:  100,
			MinAmountPerClaim: "10",
		},
		Pots: []types.ClaimingPot{
			{PotId: "g-pot-1", Creator: creator1, Status: types.PotStatusActive, Balance: "1000", TotalDeposited: "1000", TotalClaimed: "0", Config: types.PotConfig{AmountPerClaim: "100", MaxClaimsPerAccount: 1}},
		},
		Claims: []types.ClaimRecord{
			{PotId: "g-pot-1", Claimer: claimer1, ClaimCount: 1, TotalAmount: "100"},
		},
	}

	k.InitGenesis(ctx, gs)

	// Verify params
	params := k.GetParams(ctx)
	if params.MaxPotsPerCreator != 42 {
		t.Errorf("expected 42, got %d", params.MaxPotsPerCreator)
	}

	// Verify pot
	pot, found := k.GetPot(ctx, "g-pot-1")
	if !found {
		t.Fatal("expected genesis pot")
	}
	if pot.Balance != "1000" {
		t.Errorf("expected 1000, got %s", pot.Balance)
	}

	// Verify claim record
	record, found := k.GetClaimRecord(ctx, "g-pot-1", claimer1)
	if !found {
		t.Fatal("expected genesis claim record")
	}
	if record.ClaimCount != 1 {
		t.Errorf("expected 1, got %d", record.ClaimCount)
	}

	// Export and verify
	exported := k.ExportGenesis(ctx)
	if exported.Params.MaxPotsPerCreator != 42 {
		t.Errorf("exported params mismatch")
	}
	if len(exported.Pots) != 1 {
		t.Errorf("expected 1 pot, got %d", len(exported.Pots))
	}
	if len(exported.Claims) != 1 {
		t.Errorf("expected 1 claim, got %d", len(exported.Claims))
	}
}

// ==================== Query Tests ====================

func TestQueryPot(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	pot := types.ClaimingPot{PotId: "q-pot-1", Creator: creator1, Status: types.PotStatusActive, Balance: "500"}
	k.SetPot(ctx, pot)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Pot(ctx, &types.QueryPotRequest{PotId: "q-pot-1"})
	if err != nil {
		t.Fatalf("query pot failed: %v", err)
	}
	if resp.Pot.Balance != "500" {
		t.Errorf("expected 500, got %s", resp.Pot.Balance)
	}
}

func TestQueryPotNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Pot(ctx, &types.QueryPotRequest{PotId: "nonexistent"})
	if err == nil {
		t.Error("expected error for not found")
	}
}

func TestQueryPots(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.SetPot(ctx, types.ClaimingPot{PotId: "qp-1", Status: types.PotStatusActive})
	k.SetPot(ctx, types.ClaimingPot{PotId: "qp-2", Status: types.PotStatusActive})
	k.SetPot(ctx, types.ClaimingPot{PotId: "qp-3", Status: types.PotStatusClosed})

	qs := keeper.NewQueryServerImpl(k)

	// All pots
	resp, err := qs.Pots(ctx, &types.QueryPotsRequest{})
	if err != nil {
		t.Fatalf("query pots failed: %v", err)
	}
	if resp.Total != 3 {
		t.Errorf("expected 3 total, got %d", resp.Total)
	}

	// Filter by status
	resp, err = qs.Pots(ctx, &types.QueryPotsRequest{Status: "active"})
	if err != nil {
		t.Fatalf("query pots by status failed: %v", err)
	}
	if resp.Total != 2 {
		t.Errorf("expected 2 active, got %d", resp.Total)
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("query params failed: %v", err)
	}
	if resp.Params.MaxPotsPerCreator != 100 {
		t.Errorf("expected 100, got %d", resp.Params.MaxPotsPerCreator)
	}
}

// ==================== BeginBlock Tests ====================

func TestBeginBlockExpiresOldPots(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	// Active pot that expires at block 50, current is 100
	k.SetPot(ctx, types.ClaimingPot{
		PotId:   "bb-expired",
		Creator: creator1,
		Status:  types.PotStatusActive,
		Config:  types.PotConfig{AmountPerClaim: "1000", MaxClaimsPerAccount: 1, ExpiresAtBlock: 50},
	})

	// Active pot that expires at block 200 (should NOT expire)
	k.SetPot(ctx, types.ClaimingPot{
		PotId:   "bb-future",
		Creator: creator1,
		Status:  types.PotStatusActive,
		Config:  types.PotConfig{AmountPerClaim: "1000", MaxClaimsPerAccount: 1, ExpiresAtBlock: 200},
	})

	// Active pot with no expiration (should NOT expire)
	k.SetPot(ctx, types.ClaimingPot{
		PotId:   "bb-forever",
		Creator: creator1,
		Status:  types.PotStatusActive,
		Config:  types.PotConfig{AmountPerClaim: "1000", MaxClaimsPerAccount: 1, ExpiresAtBlock: 0},
	})

	// Simulate BeginBlock
	currentBlock := uint64(ctx.BlockHeight())
	k.IteratePots(ctx, func(pot types.ClaimingPot) bool {
		if pot.Status == types.PotStatusActive && pot.Config.ExpiresAtBlock > 0 && currentBlock > pot.Config.ExpiresAtBlock {
			pot.Status = types.PotStatusExpired
			k.SetPot(ctx, pot)
		}
		return false
	})

	expired, _ := k.GetPot(ctx, "bb-expired")
	if expired.Status != types.PotStatusExpired {
		t.Errorf("expected expired, got %s", expired.Status)
	}

	future, _ := k.GetPot(ctx, "bb-future")
	if future.Status != types.PotStatusActive {
		t.Errorf("expected active, got %s", future.Status)
	}

	forever, _ := k.GetPot(ctx, "bb-forever")
	if forever.Status != types.PotStatusActive {
		t.Errorf("expected active, got %s", forever.Status)
	}
}
