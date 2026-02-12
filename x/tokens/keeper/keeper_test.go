package keeper_test

import (
	"context"
	"fmt"
	"math/big"
	"testing"

	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
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

// -----------------------------------------------------------------------
// Mock BankKeeper
// -----------------------------------------------------------------------

type mockBankKeeper struct {
	balances map[string]sdkmath.Int // addr+denom -> amount
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]sdkmath.Int),
	}
}

func (m *mockBankKeeper) setBalance(addr sdk.AccAddress, denom string, amount sdkmath.Int) {
	m.balances[addr.String()+"/"+denom] = amount
}

func (m *mockBankKeeper) getBalance(addr string, denom string) sdkmath.Int {
	key := addr + "/" + denom
	if v, ok := m.balances[key]; ok {
		return v
	}
	return sdkmath.ZeroInt()
}

func (m *mockBankKeeper) SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		fromKey := fromAddr.String() + "/" + coin.Denom
		toKey := toAddr.String() + "/" + coin.Denom

		fromBal, ok := m.balances[fromKey]
		if !ok {
			fromBal = sdkmath.ZeroInt()
		}
		if fromBal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance")
		}
		m.balances[fromKey] = fromBal.Sub(coin.Amount)

		toBal, ok := m.balances[toKey]
		if !ok {
			toBal = sdkmath.ZeroInt()
		}
		m.balances[toKey] = toBal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error {
	for _, coin := range amt {
		key := recipientAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		m.balances[key] = bal.Add(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error {
	for _, coin := range amt {
		key := senderAddr.String() + "/" + coin.Denom
		bal, ok := m.balances[key]
		if !ok {
			bal = sdkmath.ZeroInt()
		}
		if bal.LT(coin.Amount) {
			return fmt.Errorf("insufficient balance for send to module")
		}
		m.balances[key] = bal.Sub(coin.Amount)
	}
	return nil
}

func (m *mockBankKeeper) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	key := addr.String() + "/" + denom
	if v, ok := m.balances[key]; ok {
		return sdk.NewCoin(denom, v)
	}
	return sdk.NewCoin(denom, sdkmath.ZeroInt())
}

func (m *mockBankKeeper) MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

func (m *mockBankKeeper) BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error {
	return nil
}

// -----------------------------------------------------------------------
// Test Setup
// -----------------------------------------------------------------------

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.SetBech32PrefixForConsensusNode("lgmvalcons", "lgmvalconspub")
}

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context, *mockBankKeeper) {
	t.Helper()

	storeKey := storetypes.NewKVStoreKey(types.StoreKey)
	db := dbm.NewMemDB()
	stateStore := store.NewCommitMultiStore(db, log.NewNopLogger(), storemetrics.NewNoOpMetrics())
	stateStore.MountStoreWithDB(storeKey, storetypes.StoreTypeIAVL, db)
	if err := stateStore.LoadLatestVersion(); err != nil {
		t.Fatalf("failed to load latest version: %v", err)
	}

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)

	bk := newMockBankKeeper()

	authority := sdk.AccAddress([]byte("authority-addr------")).String()
	k := keeper.NewKeeper(cdc, runtime.NewKVStoreService(storeKey), bk, authority)

	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	return k, ctx, bk
}

func testAddr(i int) string {
	addr := sdk.AccAddress([]byte(fmt.Sprintf("test-addr-%010d", i)))
	return addr.String()
}

// -----------------------------------------------------------------------
// Tests: Params
// -----------------------------------------------------------------------

func TestParamsDefault(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	params := k.GetParams(ctx)
	if params.PeriodBlocks != 34272 {
		t.Errorf("expected PeriodBlocks 34272, got %d", params.PeriodBlocks)
	}
	if params.PeriodsUntilExpiry != 365 {
		t.Errorf("expected PeriodsUntilExpiry 365, got %d", params.PeriodsUntilExpiry)
	}
	if params.FreshExpirationOnTransfer {
		t.Error("expected FreshExpirationOnTransfer false")
	}
}

func TestParamsSetGet(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	custom := types.Params{
		PeriodBlocks:              1000,
		PeriodsUntilExpiry:        10,
		DustThreshold:             "500",
		SweepBatchSize:            50,
		RaiDistributionRate:       2000,
		FreshExpirationOnTransfer: true,
	}
	k.SetParams(ctx, custom)
	got := k.GetParams(ctx)
	if got.PeriodBlocks != 1000 {
		t.Errorf("expected PeriodBlocks 1000, got %d", got.PeriodBlocks)
	}
	if !got.FreshExpirationOnTransfer {
		t.Error("expected FreshExpirationOnTransfer true")
	}
}

func TestParamsValidation(t *testing.T) {
	cases := []struct {
		name    string
		params  types.Params
		wantErr bool
	}{
		{"valid", types.DefaultParams(), false},
		{"zero period blocks", types.Params{PeriodBlocks: 0, PeriodsUntilExpiry: 365, DustThreshold: "1000", SweepBatchSize: 100, RaiDistributionRate: 34272}, true},
		{"zero periods until expiry", types.Params{PeriodBlocks: 34272, PeriodsUntilExpiry: 0, DustThreshold: "1000", SweepBatchSize: 100, RaiDistributionRate: 34272}, true},
		{"zero sweep batch", types.Params{PeriodBlocks: 34272, PeriodsUntilExpiry: 365, DustThreshold: "1000", SweepBatchSize: 0, RaiDistributionRate: 34272}, true},
		{"zero rai rate", types.Params{PeriodBlocks: 34272, PeriodsUntilExpiry: 365, DustThreshold: "1000", SweepBatchSize: 100, RaiDistributionRate: 0}, true},
		{"negative dust", types.Params{PeriodBlocks: 34272, PeriodsUntilExpiry: 365, DustThreshold: "-1", SweepBatchSize: 100, RaiDistributionRate: 34272}, true},
		{"invalid dust", types.Params{PeriodBlocks: 34272, PeriodsUntilExpiry: 365, DustThreshold: "abc", SweepBatchSize: 100, RaiDistributionRate: 34272}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.params.Validate()
			if tc.wantErr && err == nil {
				t.Error("expected error but got nil")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

// -----------------------------------------------------------------------
// Tests: Period Helpers
// -----------------------------------------------------------------------

func TestGetCurrentPeriod(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "1000",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})
	// Height is 100, period = 100/10 = 10
	period := k.GetCurrentPeriod(ctx)
	if period != 10 {
		t.Errorf("expected period 10, got %d", period)
	}
}

func TestIsExpired(t *testing.T) {
	k, _, _ := setupKeeper(t)
	if k.IsExpired(5, 10, 10) {
		t.Error("period 5 should not be expired at current 10 with expiry 10")
	}
	if !k.IsExpired(5, 20, 10) {
		t.Error("period 5 should be expired at current 20 with expiry 10")
	}
	if k.IsExpired(10, 10, 5) {
		t.Error("current period should not be expired")
	}
}

func TestPeriodToExpirationBlock(t *testing.T) {
	k, _, _ := setupKeeper(t)
	params := types.Params{
		PeriodBlocks:       10,
		PeriodsUntilExpiry: 5,
	}
	// period 3 expires at (3+5+1)*10 = 90
	block := k.PeriodToExpirationBlock(3, params)
	if block != 90 {
		t.Errorf("expected 90, got %d", block)
	}
}

// -----------------------------------------------------------------------
// Tests: ExpirableAccount CRUD
// -----------------------------------------------------------------------

func TestExpirableAccountCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	addr := testAddr(1)

	// Not found initially
	_, found := k.GetExpirableAccount(ctx, addr)
	if found {
		t.Error("expected account not found")
	}

	// Set
	acct := types.ExpirableAccount{
		Address:      addr,
		TotalBalance: "1000000",
		PeriodBalances: []*types.PeriodBalance{
			&types.PeriodBalance{Period: 0, Balance: "1000000", ExpiresAt: 100000, Swept: false},
		},
		EarliestPeriod: 0,
		LatestPeriod:   0,
		TotalExpired:   "0",
		AccountType:    "standard",
	}
	k.SetExpirableAccount(ctx, acct)

	// Get
	got, found := k.GetExpirableAccount(ctx, addr)
	if !found {
		t.Fatal("expected account to be found")
	}
	if got.TotalBalance != "1000000" {
		t.Errorf("expected balance 1000000, got %s", got.TotalBalance)
	}

	// Delete
	k.DeleteExpirableAccount(ctx, addr)
	_, found = k.GetExpirableAccount(ctx, addr)
	if found {
		t.Error("expected account to be deleted")
	}
}

func TestGetAllExpirableAccounts(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	for i := 0; i < 3; i++ {
		k.SetExpirableAccount(ctx, types.ExpirableAccount{
			Address:      testAddr(i),
			TotalBalance: "100",
			AccountType:  "standard",
			TotalExpired: "0",
		})
	}
	all := k.GetAllExpirableAccounts(ctx)
	if len(all) != 3 {
		t.Errorf("expected 3 accounts, got %d", len(all))
	}
}

// -----------------------------------------------------------------------
// Tests: CreditBalance
// -----------------------------------------------------------------------

func TestCreditBalance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)
	err := k.CreditBalance(ctx, addr, "5000", 100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	acct, found := k.GetExpirableAccount(ctx, addr)
	if !found {
		t.Fatal("account not found after credit")
	}
	if acct.TotalBalance != "5000" {
		t.Errorf("expected total 5000, got %s", acct.TotalBalance)
	}
	if len(acct.PeriodBalances) != 1 {
		t.Fatalf("expected 1 period, got %d", len(acct.PeriodBalances))
	}
	if acct.PeriodBalances[0].Period != 10 {
		t.Errorf("expected period 10, got %d", acct.PeriodBalances[0].Period)
	}
}

func TestCreditBalanceMultiplePeriods(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)
	// Period 10
	if err := k.CreditBalance(ctx, addr, "5000", 100); err != nil {
		t.Fatalf("credit 1: %v", err)
	}
	// Period 20
	if err := k.CreditBalance(ctx, addr, "3000", 200); err != nil {
		t.Fatalf("credit 2: %v", err)
	}
	// Period 10 again (same period accumulates)
	if err := k.CreditBalance(ctx, addr, "2000", 105); err != nil {
		t.Fatalf("credit 3: %v", err)
	}

	acct, _ := k.GetExpirableAccount(ctx, addr)
	if acct.TotalBalance != "10000" {
		t.Errorf("expected 10000, got %s", acct.TotalBalance)
	}
	if len(acct.PeriodBalances) != 2 {
		t.Errorf("expected 2 periods, got %d", len(acct.PeriodBalances))
	}
	// Period 10 should have 7000
	if acct.PeriodBalances[0].Balance != "7000" {
		t.Errorf("expected period 10 balance 7000, got %s", acct.PeriodBalances[0].Balance)
	}
}

func TestCreditBalanceBelowDust(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "1000",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	err := k.CreditBalance(ctx, testAddr(1), "999", 100)
	if err == nil {
		t.Error("expected dust error")
	}
}

func TestCreditBalanceInvalidAmount(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	err := k.CreditBalance(ctx, testAddr(1), "0", 100)
	if err == nil {
		t.Error("expected error for zero amount")
	}
	err = k.CreditBalance(ctx, testAddr(1), "-100", 100)
	if err == nil {
		t.Error("expected error for negative amount")
	}
	err = k.CreditBalance(ctx, testAddr(1), "abc", 100)
	if err == nil {
		t.Error("expected error for non-numeric amount")
	}
}

// -----------------------------------------------------------------------
// Tests: CreditBalanceToSpecificPeriod
// -----------------------------------------------------------------------

func TestCreditBalanceToSpecificPeriod(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)
	err := k.CreditBalanceToSpecificPeriod(ctx, addr, "5000", 5)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	acct, _ := k.GetExpirableAccount(ctx, addr)
	if acct.PeriodBalances[0].Period != 5 {
		t.Errorf("expected period 5, got %d", acct.PeriodBalances[0].Period)
	}
	if acct.EarliestPeriod != 5 {
		t.Errorf("expected earliest 5, got %d", acct.EarliestPeriod)
	}
}

// -----------------------------------------------------------------------
// Tests: DebitBalance
// -----------------------------------------------------------------------

func TestDebitBalance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)
	k.CreditBalance(ctx, addr, "10000", 100)
	err := k.DebitBalance(ctx, addr, "3000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bal := k.GetBalance(ctx, addr)
	if bal != "7000" {
		t.Errorf("expected 7000, got %s", bal)
	}
}

func TestDebitBalanceFIFO(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)
	// Period 5
	k.CreditBalanceToSpecificPeriod(ctx, addr, "5000", 5)
	// Period 10
	k.CreditBalanceToSpecificPeriod(ctx, addr, "5000", 10)

	// Debit 7000 — should consume all of period 5 (5000) + 2000 from period 10
	err := k.DebitBalance(ctx, addr, "7000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	acct, _ := k.GetExpirableAccount(ctx, addr)
	if len(acct.PeriodBalances) != 1 {
		t.Fatalf("expected 1 remaining period, got %d", len(acct.PeriodBalances))
	}
	if acct.PeriodBalances[0].Period != 10 {
		t.Errorf("expected remaining period 10, got %d", acct.PeriodBalances[0].Period)
	}
	if acct.PeriodBalances[0].Balance != "3000" {
		t.Errorf("expected remaining 3000, got %s", acct.PeriodBalances[0].Balance)
	}
}

func TestDebitBalanceInsufficientFunds(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)
	k.CreditBalance(ctx, addr, "1000", 100)
	err := k.DebitBalance(ctx, addr, "2000")
	if err == nil {
		t.Error("expected insufficient balance error")
	}
}

func TestDebitBalanceAccountNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	err := k.DebitBalance(ctx, testAddr(99), "1000")
	if err == nil {
		t.Error("expected account not found error")
	}
}

// -----------------------------------------------------------------------
// Tests: GetBalance
// -----------------------------------------------------------------------

func TestGetBalanceExcludesExpired(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	params := types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  5,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	}
	k.SetParams(ctx, params)

	addr := testAddr(1)
	// Directly set up account with a pre-expired period balance.
	// CreditBalanceToSpecificPeriod would redirect expired periods to current,
	// so we use SetExpirableAccount to place balances at already-expired periods.
	// Period 0 — expired at current period 10 (10-0 > 5)
	// Period 8 — active at current period 10 (10-8 <= 5)
	k.SetExpirableAccount(ctx, types.ExpirableAccount{
		Address:      addr,
		TotalBalance: "8000",
		PeriodBalances: []*types.PeriodBalance{
			{Period: 0, Balance: "5000", ExpiresAt: 60, Swept: false},
			{Period: 8, Balance: "3000", ExpiresAt: 140, Swept: false},
		},
		EarliestPeriod: 0,
		LatestPeriod:   8,
		TotalExpired:   "0",
		AccountType:    "standard",
	})

	bal := k.GetBalance(ctx, addr)
	if bal != "3000" {
		t.Errorf("expected 3000, got %s", bal)
	}
}

func TestGetBalanceEmpty(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	bal := k.GetBalance(ctx, testAddr(99))
	if bal != "0" {
		t.Errorf("expected 0 for non-existent account, got %s", bal)
	}
}

// -----------------------------------------------------------------------
// Tests: TransferPreservingPeriods
// -----------------------------------------------------------------------

func TestTransferPreservingPeriods(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	from := testAddr(1)
	to := testAddr(2)

	// Fund sender at periods 5 and 8
	k.CreditBalanceToSpecificPeriod(ctx, from, "5000", 5)
	k.CreditBalanceToSpecificPeriod(ctx, from, "5000", 8)

	// Transfer 7000 (FIFO: 5000 from period 5 + 2000 from period 8)
	err := k.TransferPreservingPeriods(ctx, from, to, "7000")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Sender should have 3000 left at period 8
	senderBal := k.GetBalance(ctx, from)
	if senderBal != "3000" {
		t.Errorf("sender expected 3000, got %s", senderBal)
	}

	// Receiver should have tokens at sender's periods
	receiverAcct, _ := k.GetExpirableAccount(ctx, to)
	if len(receiverAcct.PeriodBalances) != 2 {
		t.Fatalf("expected 2 periods on receiver, got %d", len(receiverAcct.PeriodBalances))
	}
	if receiverAcct.PeriodBalances[0].Period != 5 || receiverAcct.PeriodBalances[0].Balance != "5000" {
		t.Errorf("expected period 5 with 5000, got period %d with %s", receiverAcct.PeriodBalances[0].Period, receiverAcct.PeriodBalances[0].Balance)
	}
	if receiverAcct.PeriodBalances[1].Period != 8 || receiverAcct.PeriodBalances[1].Balance != "2000" {
		t.Errorf("expected period 8 with 2000, got period %d with %s", receiverAcct.PeriodBalances[1].Period, receiverAcct.PeriodBalances[1].Balance)
	}
}

func TestTransferPreservingPeriodsInsufficientBalance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	from := testAddr(1)
	k.CreditBalanceToSpecificPeriod(ctx, from, "1000", 5)

	err := k.TransferPreservingPeriods(ctx, from, testAddr(2), "5000")
	if err == nil {
		t.Error("expected insufficient balance error")
	}
}

// -----------------------------------------------------------------------
// Tests: Allowance
// -----------------------------------------------------------------------

func TestAllowanceCRUD(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	owner := testAddr(1)
	spender := testAddr(2)

	// Default is "0"
	allowance := k.GetAllowance(ctx, owner, spender)
	if allowance != "0" {
		t.Errorf("expected 0, got %s", allowance)
	}

	// Set
	k.SetAllowance(ctx, owner, spender, "5000")
	allowance = k.GetAllowance(ctx, owner, spender)
	if allowance != "5000" {
		t.Errorf("expected 5000, got %s", allowance)
	}

	// Delete
	k.DeleteAllowance(ctx, owner, spender)
	allowance = k.GetAllowance(ctx, owner, spender)
	if allowance != "0" {
		t.Errorf("expected 0 after delete, got %s", allowance)
	}
}

// -----------------------------------------------------------------------
// Tests: RAI
// -----------------------------------------------------------------------

func TestRAIAccountDefault(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	rai := k.GetRAIAccount(ctx)
	if rai.Balance != "0" {
		t.Errorf("expected 0 balance, got %s", rai.Balance)
	}
	if rai.TotalReceived != "0" {
		t.Errorf("expected 0 total received, got %s", rai.TotalReceived)
	}
}

func TestAddToRAIPool(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.AddToRAIPool(ctx, "5000")
	k.AddToRAIPool(ctx, "3000")

	rai := k.GetRAIAccount(ctx)
	if rai.Balance != "8000" {
		t.Errorf("expected 8000, got %s", rai.Balance)
	}
	if rai.TotalReceived != "8000" {
		t.Errorf("expected total received 8000, got %s", rai.TotalReceived)
	}
}

func TestDistributeRAI(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	k.AddToRAIPool(ctx, "9000")

	recipients := []string{testAddr(1), testAddr(2), testAddr(3)}
	dist, err := k.DistributeRAI(ctx, recipients)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 9000 / 3 = 3000 per recipient
	if dist.PerRecipient != "3000" {
		t.Errorf("expected 3000 per recipient, got %s", dist.PerRecipient)
	}
	if dist.RecipientCount != 3 {
		t.Errorf("expected 3 recipients, got %d", dist.RecipientCount)
	}

	// Check recipient balances
	for _, r := range recipients {
		bal := k.GetBalance(ctx, r)
		if bal != "3000" {
			t.Errorf("expected recipient balance 3000, got %s for %s", bal, r)
		}
	}

	// RAI pool should be empty
	rai := k.GetRAIAccount(ctx)
	if rai.Balance != "0" {
		t.Errorf("expected RAI balance 0, got %s", rai.Balance)
	}
}

func TestDistributeRAIEmpty(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	_, err := k.DistributeRAI(ctx, []string{testAddr(1)})
	if err == nil {
		t.Error("expected error for empty RAI pool")
	}
}

func TestDistributeRAINoRecipients(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.AddToRAIPool(ctx, "5000")
	_, err := k.DistributeRAI(ctx, []string{})
	if err == nil {
		t.Error("expected error for no recipients")
	}
}

func TestIsEligibleForRAI(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)

	// Not eligible with no account
	if k.IsEligibleForRAI(ctx, addr) {
		t.Error("should not be eligible without account")
	}

	// Standard account with balance
	k.CreditBalance(ctx, addr, "5000", 100)
	if !k.IsEligibleForRAI(ctx, addr) {
		t.Error("should be eligible with active balance")
	}

	// Foundation accounts are not eligible
	foundAddr := testAddr(2)
	k.SetExpirableAccount(ctx, types.ExpirableAccount{
		Address:      foundAddr,
		TotalBalance: "5000",
		PeriodBalances: []*types.PeriodBalance{
			&types.PeriodBalance{Period: 10, Balance: "5000", ExpiresAt: 100000},
		},
		AccountType:  "foundation",
		TotalExpired: "0",
	})
	if k.IsEligibleForRAI(ctx, foundAddr) {
		t.Error("foundation account should not be eligible for RAI")
	}
}

// -----------------------------------------------------------------------
// Tests: SweepExpiredTokens
// -----------------------------------------------------------------------

func TestSweepExpiredTokens(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	params := types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  5,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	}
	k.SetParams(ctx, params)

	addr := testAddr(1)
	// Directly set up account with a pre-expired period balance.
	// CreditBalanceToSpecificPeriod would redirect expired periods to current,
	// so we use SetExpirableAccount + AddToSweepIndex to place balances at
	// already-expired periods with the correct sweep index entries.
	// Period 0 — expired at height 100 (period 10, expiry 5: 10-0 > 5)
	// Period 8 — still active at period 10 (10-8 <= 5)
	k.SetExpirableAccount(ctx, types.ExpirableAccount{
		Address:      addr,
		TotalBalance: "8000",
		PeriodBalances: []*types.PeriodBalance{
			{Period: 0, Balance: "5000", ExpiresAt: 60, Swept: false},
			{Period: 8, Balance: "3000", ExpiresAt: 140, Swept: false},
		},
		EarliestPeriod: 0,
		LatestPeriod:   8,
		TotalExpired:   "0",
		AccountType:    "standard",
	})
	k.AddToSweepIndex(ctx, 0, addr)
	k.AddToSweepIndex(ctx, 8, addr)

	totalSwept, accountsProcessed := k.SweepExpiredTokens(ctx, 100)
	sweptInt := new(big.Int)
	sweptInt.SetString(totalSwept, 10)
	if sweptInt.Cmp(big.NewInt(5000)) != 0 {
		t.Errorf("expected 5000 swept, got %s", totalSwept)
	}
	if accountsProcessed != 1 {
		t.Errorf("expected 1 account processed, got %d", accountsProcessed)
	}

	// RAI pool should have the swept amount
	rai := k.GetRAIAccount(ctx)
	if rai.Balance != "5000" {
		t.Errorf("expected RAI balance 5000, got %s", rai.Balance)
	}

	// Account should only have the active period
	acct, _ := k.GetExpirableAccount(ctx, addr)
	if len(acct.PeriodBalances) != 1 {
		t.Fatalf("expected 1 period remaining, got %d", len(acct.PeriodBalances))
	}
	if acct.PeriodBalances[0].Period != 8 {
		t.Errorf("expected remaining period 8, got %d", acct.PeriodBalances[0].Period)
	}
}

func TestSweepNoExpiredTokens(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	k.CreditBalanceToSpecificPeriod(ctx, testAddr(1), "5000", 8)
	totalSwept, accountsProcessed := k.SweepExpiredTokens(ctx, 100)
	if totalSwept != "0" {
		t.Errorf("expected 0 swept, got %s", totalSwept)
	}
	if accountsProcessed != 0 {
		t.Errorf("expected 0 accounts processed, got %d", accountsProcessed)
	}
}

// -----------------------------------------------------------------------
// Tests: Sweep Index
// -----------------------------------------------------------------------

func TestSweepIndex(t *testing.T) {
	k, ctx, _ := setupKeeper(t)

	k.AddToSweepIndex(ctx, 5, testAddr(1))
	k.AddToSweepIndex(ctx, 5, testAddr(2))
	k.AddToSweepIndex(ctx, 10, testAddr(3))

	addrs := k.GetSweepAddresses(ctx, 5)
	if len(addrs) != 2 {
		t.Errorf("expected 2 addresses for period 5, got %d", len(addrs))
	}

	addrs = k.GetSweepAddresses(ctx, 10)
	if len(addrs) != 1 {
		t.Errorf("expected 1 address for period 10, got %d", len(addrs))
	}

	k.RemoveFromSweepIndex(ctx, 5, testAddr(1))
	addrs = k.GetSweepAddresses(ctx, 5)
	if len(addrs) != 1 {
		t.Errorf("expected 1 address after removal, got %d", len(addrs))
	}
}

// -----------------------------------------------------------------------
// Tests: MsgServer
// -----------------------------------------------------------------------

func TestMsgTransfer(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:              10,
		PeriodsUntilExpiry:        365,
		DustThreshold:             "100",
		SweepBatchSize:            100,
		RaiDistributionRate:       100,
		FreshExpirationOnTransfer: false,
	})

	from := testAddr(1)
	to := testAddr(2)

	// Set up expirable balance and bank balance
	k.CreditBalanceToSpecificPeriod(ctx, from, "10000", 5)
	fromAddr, _ := sdk.AccAddressFromBech32(from)
	bk.setBalance(fromAddr, "ulgm", sdkmath.NewInt(10000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.Transfer(ctx, &types.MsgTransfer{
		Sender:   from,
		Receiver: to,
		Amount:   "5000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Expirable balance check
	senderBal := k.GetBalance(ctx, from)
	if senderBal != "5000" {
		t.Errorf("expected sender balance 5000, got %s", senderBal)
	}
	receiverBal := k.GetBalance(ctx, to)
	if receiverBal != "5000" {
		t.Errorf("expected receiver balance 5000, got %s", receiverBal)
	}
}

func TestMsgTransferFreshExpiration(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:              10,
		PeriodsUntilExpiry:        365,
		DustThreshold:             "100",
		SweepBatchSize:            100,
		RaiDistributionRate:       100,
		FreshExpirationOnTransfer: true,
	})

	from := testAddr(1)
	to := testAddr(2)

	// Fund at old period
	k.CreditBalanceToSpecificPeriod(ctx, from, "10000", 2)
	fromAddr, _ := sdk.AccAddressFromBech32(from)
	bk.setBalance(fromAddr, "ulgm", sdkmath.NewInt(10000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.Transfer(ctx, &types.MsgTransfer{
		Sender:   from,
		Receiver: to,
		Amount:   "5000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Receiver should have tokens at current period (10), not sender's period (2)
	receiverAcct, _ := k.GetExpirableAccount(ctx, to)
	if len(receiverAcct.PeriodBalances) != 1 {
		t.Fatalf("expected 1 period, got %d", len(receiverAcct.PeriodBalances))
	}
	if receiverAcct.PeriodBalances[0].Period != 10 {
		t.Errorf("expected period 10 (current), got %d", receiverAcct.PeriodBalances[0].Period)
	}
}

func TestMsgApproveAllowance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	owner := testAddr(1)
	spender := testAddr(2)

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.ApproveAllowance(ctx, &types.MsgApproveAllowance{
		Owner:   owner,
		Spender: spender,
		Amount:  "5000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	allowance := k.GetAllowance(ctx, owner, spender)
	if allowance != "5000" {
		t.Errorf("expected 5000, got %s", allowance)
	}
}

func TestMsgApproveAllowanceRevoke(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	owner := testAddr(1)
	spender := testAddr(2)

	srv := keeper.NewMsgServerImpl(k)
	srv.ApproveAllowance(ctx, &types.MsgApproveAllowance{
		Owner: owner, Spender: spender, Amount: "5000",
	})
	srv.ApproveAllowance(ctx, &types.MsgApproveAllowance{
		Owner: owner, Spender: spender, Amount: "0",
	})

	allowance := k.GetAllowance(ctx, owner, spender)
	if allowance != "0" {
		t.Errorf("expected 0 after revoke, got %s", allowance)
	}
}

func TestMsgBurn(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)
	k.CreditBalance(ctx, addr, "10000", 100)
	accAddr, _ := sdk.AccAddressFromBech32(addr)
	bk.setBalance(accAddr, "ulgm", sdkmath.NewInt(10000))

	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.Burn(ctx, &types.MsgBurn{
		Sender: addr,
		Amount: "3000",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	bal := k.GetBalance(ctx, addr)
	if bal != "7000" {
		t.Errorf("expected 7000 after burn, got %s", bal)
	}
}

func TestMsgUpdateParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	authority := k.GetAuthority()

	srv := keeper.NewMsgServerImpl(k)
	newParams := types.Params{
		PeriodBlocks:              5000,
		PeriodsUntilExpiry:        100,
		DustThreshold:             "500",
		SweepBatchSize:            200,
		RaiDistributionRate:       10000,
		FreshExpirationOnTransfer: true,
	}
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: authority,
		Params:    newParams,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := k.GetParams(ctx)
	if got.PeriodBlocks != 5000 {
		t.Errorf("expected 5000, got %d", got.PeriodBlocks)
	}
}

func TestMsgUpdateParamsUnauthorized(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.UpdateParams(ctx, &types.MsgUpdateParams{
		Authority: testAddr(99),
		Params:    types.DefaultParams(),
	})
	if err == nil {
		t.Error("expected unauthorized error")
	}
}

// -----------------------------------------------------------------------
// Tests: Queries
// -----------------------------------------------------------------------

func TestQueryAccount(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)
	k.CreditBalance(ctx, addr, "5000", 100)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Account(ctx, &types.QueryAccountRequest{Address: addr})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Account.TotalBalance != "5000" {
		t.Errorf("expected 5000, got %s", resp.Account.TotalBalance)
	}
}

func TestQueryAccountNotFound(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Account(ctx, &types.QueryAccountRequest{Address: testAddr(99)})
	if err == nil {
		t.Error("expected not found error")
	}
}

func TestQueryBalance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr := testAddr(1)
	k.CreditBalance(ctx, addr, "5000", 100)
	k.CreditBalance(ctx, addr, "3000", 200)

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Balance(ctx, &types.QueryBalanceRequest{Address: addr})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ActiveBalance != "8000" {
		t.Errorf("expected 8000, got %s", resp.ActiveBalance)
	}
	if len(resp.PeriodBalances) != 2 {
		t.Errorf("expected 2 period balances, got %d", len(resp.PeriodBalances))
	}
}

func TestQueryBalanceEmpty(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Balance(ctx, &types.QueryBalanceRequest{Address: testAddr(99)})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ActiveBalance != "0" {
		t.Errorf("expected 0, got %s", resp.ActiveBalance)
	}
}

func TestQueryAllowance(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	owner := testAddr(1)
	spender := testAddr(2)
	k.SetAllowance(ctx, owner, spender, "5000")

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Allowance(ctx, &types.QueryAllowanceRequest{Owner: owner, Spender: spender})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Allowance != "5000" {
		t.Errorf("expected 5000, got %s", resp.Allowance)
	}
}

func TestQueryRAI(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.AddToRAIPool(ctx, "10000")

	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.RAI(ctx, &types.QueryRAIRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.RAI.Balance != "10000" {
		t.Errorf("expected 10000, got %s", resp.RAI.Balance)
	}
}

func TestQueryParams(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	resp, err := qs.Params(ctx, &types.QueryParamsRequest{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Params.PeriodBlocks != 34272 {
		t.Errorf("expected 34272, got %d", resp.Params.PeriodBlocks)
	}
}

// -----------------------------------------------------------------------
// Tests: ValidateBasic
// -----------------------------------------------------------------------

func TestMsgTransferValidateBasic(t *testing.T) {
	from := testAddr(1)
	to := testAddr(2)

	cases := []struct {
		name    string
		msg     types.MsgTransfer
		wantErr bool
	}{
		{"valid", types.MsgTransfer{Sender: from, Receiver: to, Amount: "1000"}, false},
		{"empty sender", types.MsgTransfer{Sender: "", Receiver: to, Amount: "1000"}, true},
		{"empty receiver", types.MsgTransfer{Sender: from, Receiver: "", Amount: "1000"}, true},
		{"same sender receiver", types.MsgTransfer{Sender: from, Receiver: from, Amount: "1000"}, true},
		{"zero amount", types.MsgTransfer{Sender: from, Receiver: to, Amount: "0"}, true},
		{"negative amount", types.MsgTransfer{Sender: from, Receiver: to, Amount: "-1"}, true},
		{"invalid amount", types.MsgTransfer{Sender: from, Receiver: to, Amount: "abc"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMsgApproveAllowanceValidateBasic(t *testing.T) {
	owner := testAddr(1)
	spender := testAddr(2)

	cases := []struct {
		name    string
		msg     types.MsgApproveAllowance
		wantErr bool
	}{
		{"valid", types.MsgApproveAllowance{Owner: owner, Spender: spender, Amount: "1000"}, false},
		{"valid revoke", types.MsgApproveAllowance{Owner: owner, Spender: spender, Amount: "0"}, false},
		{"empty owner", types.MsgApproveAllowance{Owner: "", Spender: spender, Amount: "1000"}, true},
		{"empty spender", types.MsgApproveAllowance{Owner: owner, Spender: "", Amount: "1000"}, true},
		{"same owner spender", types.MsgApproveAllowance{Owner: owner, Spender: owner, Amount: "1000"}, true},
		{"negative amount", types.MsgApproveAllowance{Owner: owner, Spender: spender, Amount: "-1"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMsgBurnValidateBasic(t *testing.T) {
	addr := testAddr(1)

	cases := []struct {
		name    string
		msg     types.MsgBurn
		wantErr bool
	}{
		{"valid", types.MsgBurn{Sender: addr, Amount: "1000"}, false},
		{"empty sender", types.MsgBurn{Sender: "", Amount: "1000"}, true},
		{"zero amount", types.MsgBurn{Sender: addr, Amount: "0"}, true},
		{"negative", types.MsgBurn{Sender: addr, Amount: "-1"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.msg.ValidateBasic()
			if tc.wantErr && err == nil {
				t.Error("expected error")
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestMsgUpdateParamsValidateBasic(t *testing.T) {
	authority := testAddr(1)

	valid := types.MsgUpdateParams{Authority: authority, Params: types.DefaultParams()}
	if err := valid.ValidateBasic(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	invalid := types.MsgUpdateParams{Authority: "", Params: types.DefaultParams()}
	if err := invalid.ValidateBasic(); err == nil {
		t.Error("expected error for empty authority")
	}

	invalidParams := types.MsgUpdateParams{
		Authority: authority,
		Params:    types.Params{PeriodBlocks: 0},
	}
	if err := invalidParams.ValidateBasic(); err == nil {
		t.Error("expected error for invalid params")
	}
}

// -----------------------------------------------------------------------
// Tests: Genesis
// -----------------------------------------------------------------------

func TestGenesis(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:        10,
		PeriodsUntilExpiry:  365,
		DustThreshold:       "100",
		SweepBatchSize:      100,
		RaiDistributionRate: 100,
	})

	addr1 := testAddr(1)
	addr2 := testAddr(2)
	k.CreditBalanceToSpecificPeriod(ctx, addr1, "5000", 5)
	k.CreditBalanceToSpecificPeriod(ctx, addr2, "3000", 8)
	k.AddToRAIPool(ctx, "1000")

	// Export
	genState := k.ExportGenesis(ctx)
	if len(genState.Accounts) != 2 {
		t.Errorf("expected 2 accounts, got %d", len(genState.Accounts))
	}
	if genState.Rai.Balance != "1000" {
		t.Errorf("expected RAI balance 1000, got %s", genState.Rai.Balance)
	}

	// New keeper, import
	k2, ctx2, _ := setupKeeper(t)
	k2.InitGenesis(ctx2, *genState)

	got := k2.ExportGenesis(ctx2)
	if len(got.Accounts) != 2 {
		t.Errorf("expected 2 accounts after import, got %d", len(got.Accounts))
	}
	if got.Rai.Balance != "1000" {
		t.Errorf("expected RAI 1000 after import, got %s", got.Rai.Balance)
	}
}

func TestGenesisValidation(t *testing.T) {
	valid := types.DefaultGenesis()
	if err := valid.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Duplicate address
	dup := types.DefaultGenesis()
	dup.Accounts = []*types.ExpirableAccount{
		{Address: "lgm1abc", AccountType: "standard"},
		{Address: "lgm1abc", AccountType: "standard"},
	}
	if err := dup.Validate(); err == nil {
		t.Error("expected duplicate address error")
	}

	// Invalid account type
	badType := types.DefaultGenesis()
	badType.Accounts = []*types.ExpirableAccount{
		{Address: "lgm1xyz", AccountType: "invalid"},
	}
	if err := badType.Validate(); err == nil {
		t.Error("expected invalid account type error")
	}
}

// -----------------------------------------------------------------------
// Tests: Full Lifecycle
// -----------------------------------------------------------------------

func TestFullLifecycle(t *testing.T) {
	k, ctx, bk := setupKeeper(t)
	k.SetParams(ctx, types.Params{
		PeriodBlocks:              10,
		PeriodsUntilExpiry:        5,
		DustThreshold:             "100",
		SweepBatchSize:            100,
		RaiDistributionRate:       100,
		FreshExpirationOnTransfer: false,
	})

	alice := testAddr(1)
	bob := testAddr(2)
	aliceAddr, _ := sdk.AccAddressFromBech32(alice)

	// 1. Credit Alice at period 8 (active at period 10 with expiry 5: 10-8=2 <= 5)
	k.CreditBalanceToSpecificPeriod(ctx, alice, "10000", 8)
	bk.setBalance(aliceAddr, "ulgm", sdkmath.NewInt(10000))

	// 2. Transfer to Bob (period-preserving)
	srv := keeper.NewMsgServerImpl(k)
	_, err := srv.Transfer(ctx, &types.MsgTransfer{
		Sender: alice, Receiver: bob, Amount: "4000",
	})
	if err != nil {
		t.Fatalf("transfer failed: %v", err)
	}

	// 3. Verify period preservation — Bob gets tokens at Alice's period 8
	bobAcct, _ := k.GetExpirableAccount(ctx, bob)
	if bobAcct.PeriodBalances[0].Period != 8 {
		t.Errorf("expected Bob to have period 8 (preserved), got %d", bobAcct.PeriodBalances[0].Period)
	}

	// 4. Set allowance
	_, err = srv.ApproveAllowance(ctx, &types.MsgApproveAllowance{
		Owner: alice, Spender: bob, Amount: "2000",
	})
	if err != nil {
		t.Fatalf("allowance failed: %v", err)
	}
	if k.GetAllowance(ctx, alice, bob) != "2000" {
		t.Error("allowance not set")
	}

	// 5. Advance context to height 200 (period 20) so period 8 expires (20-8=12 > 5)
	ctx = ctx.WithBlockHeight(200)

	totalSwept, _ := k.SweepExpiredTokens(ctx, 100)
	sweptInt := new(big.Int)
	sweptInt.SetString(totalSwept, 10)
	if sweptInt.Cmp(big.NewInt(10000)) != 0 {
		t.Errorf("expected 10000 swept (6000 alice + 4000 bob), got %s", totalSwept)
	}

	// 6. Verify RAI pool got swept tokens
	rai := k.GetRAIAccount(ctx)
	if rai.Balance != "10000" {
		t.Errorf("expected RAI balance 10000, got %s", rai.Balance)
	}

	// 7. Distribute RAI to a new set of recipients
	charlie := testAddr(3)
	k.CreditBalanceToSpecificPeriod(ctx, charlie, "1000", 18) // active at period 20
	dist, err := k.DistributeRAI(ctx, []string{charlie})
	if err != nil {
		t.Fatalf("RAI distribution failed: %v", err)
	}
	if dist.PerRecipient != "10000" {
		t.Errorf("expected 10000 per recipient, got %s", dist.PerRecipient)
	}

	// 8. Export genesis for persistence check
	genState := k.ExportGenesis(ctx)
	if len(genState.Accounts) < 1 {
		t.Error("expected at least 1 account in genesis export")
	}
}

// -----------------------------------------------------------------------
// Tests: Query validation
// -----------------------------------------------------------------------

func TestQueryAccountNilRequest(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Account(ctx, nil)
	if err == nil {
		t.Error("expected error for nil request")
	}
}

func TestQueryBalanceNilRequest(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Balance(ctx, nil)
	if err == nil {
		t.Error("expected error for nil request")
	}
}

func TestQueryAllowanceNilRequest(t *testing.T) {
	k, ctx, _ := setupKeeper(t)
	qs := keeper.NewQueryServerImpl(k)
	_, err := qs.Allowance(ctx, nil)
	if err == nil {
		t.Error("expected error for nil request")
	}
}
