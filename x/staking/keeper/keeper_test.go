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

	"github.com/legible-money/legible/x/staking/keeper"
	"github.com/legible-money/legible/x/staking/types"
)

func init() {
	config := sdk.GetConfig()
	config.SetBech32PrefixForAccount("lgm", "lgmpub")
	config.SetBech32PrefixForValidator("lgmvaloper", "lgmvaloperpub")
	config.Seal()
}

// ---------- Mock Keepers ----------

type mockBankKeeper struct {
	balances map[string]map[string]int64 // addr -> denom -> amount
}

func newMockBankKeeper() *mockBankKeeper {
	return &mockBankKeeper{
		balances: make(map[string]map[string]int64),
	}
}

func (m *mockBankKeeper) setBalance(addr, denom string, amount int64) {
	if m.balances[addr] == nil {
		m.balances[addr] = make(map[string]int64)
	}
	m.balances[addr][denom] = amount
}

func (m *mockBankKeeper) SendCoins(_ context.Context, _, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}
func (m *mockBankKeeper) SendCoinsFromModuleToAccount(_ context.Context, _ string, _ sdk.AccAddress, _ sdk.Coins) error {
	return nil
}
func (m *mockBankKeeper) SendCoinsFromAccountToModule(_ context.Context, _ sdk.AccAddress, _ string, _ sdk.Coins) error {
	return nil
}
func (m *mockBankKeeper) GetBalance(_ context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	if bal, ok := m.balances[addr.String()]; ok {
		if amount, ok := bal[denom]; ok {
			return sdk.NewCoin(denom, sdkmath.NewInt(amount))
		}
	}
	return sdk.NewCoin(denom, sdkmath.ZeroInt())
}
func (m *mockBankKeeper) GetAllBalances(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	return nil
}
func (m *mockBankKeeper) SpendableCoins(_ context.Context, _ sdk.AccAddress) sdk.Coins {
	return nil
}
func (m *mockBankKeeper) MintCoins(_ context.Context, _ string, _ sdk.Coins) error { return nil }
func (m *mockBankKeeper) BurnCoins(_ context.Context, _ string, _ sdk.Coins) error { return nil }

type mockAccountKeeper struct{}

func (m *mockAccountKeeper) GetAccount(_ context.Context, _ sdk.AccAddress) sdk.AccountI { return nil }
func (m *mockAccountKeeper) SetAccount(_ context.Context, _ sdk.AccountI)               {}
func (m *mockAccountKeeper) NewAccountWithAddress(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}

type mockTokensKeeper struct {
	period uint64
}

func (m *mockTokensKeeper) DebitBalance(_ sdk.Context, _, _ string) error { return nil }
func (m *mockTokensKeeper) CreditBalance(_ sdk.Context, _, _ string, _ uint64) error {
	return nil
}
func (m *mockTokensKeeper) CreditBalanceToSpecificPeriod(_ sdk.Context, _, _ string, _ uint64) error {
	return nil
}
func (m *mockTokensKeeper) GetCurrentPeriod(_ sdk.Context) uint64 { return m.period }

// ---------- Test Setup ----------

func setupKeeper(t *testing.T) (keeper.Keeper, sdk.Context) {
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

	storeService := runtime.NewKVStoreService(storeKey)
	bk := newMockBankKeeper()
	ak := &mockAccountKeeper{}
	tk := &mockTokensKeeper{period: 1}

	k := keeper.NewKeeper(cdc, storeService, ak, bk, nil, tk, "authority")
	ctx := sdk.NewContext(stateStore, cmtproto.Header{Height: 100}, false, log.NewNopLogger())

	// Init genesis
	gs := types.DefaultGenesisState()
	if err := k.InitGenesis(ctx, gs); err != nil {
		t.Fatalf("failed to init genesis: %v", err)
	}

	return k, ctx
}

func registerTestValidator(t *testing.T, k keeper.Keeper, ctx sdk.Context, addr, did, stake string) {
	t.Helper()
	resp, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
		Sender:          addr,
		ConsensusPubkey: "abcdef0123456789",
		Did:             did,
		InitialStake:    stake,
		Commission:      1000,
	})
	if err != nil {
		t.Fatalf("failed to register validator %s: %v", addr, err)
	}
	_ = resp
}

// ---------- Delegation Tests ----------

func TestDelegate_Success(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator1__________").String()

	resp, err := k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr,
		Validator: valAddr,
		Amount:    "500000000000000000000",
	})
	if err != nil {
		t.Fatalf("delegate failed: %v", err)
	}
	if resp.NewDelegation != "500000000000000000000" {
		t.Errorf("expected delegation 500000000000000000000, got %s", resp.NewDelegation)
	}

	// Verify delegation stored
	del, found := k.GetDelegation(ctx, delAddr, valAddr)
	if !found {
		t.Fatal("delegation not found")
	}
	if del.Amount != "500000000000000000000" {
		t.Errorf("expected delegation amount 500000000000000000000, got %s", del.Amount)
	}

	// Verify validator DelegatedStake updated
	val, _ := k.GetValidator(ctx, valAddr)
	if val.DelegatedStake != "500000000000000000000" {
		t.Errorf("expected delegated stake 500000000000000000000, got %s", val.DelegatedStake)
	}
}

func TestDelegate_InactiveValidator(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "0")

	// Deactivate manually
	val, _ := k.GetValidator(ctx, valAddr)
	val.IsActive = false
	k.SetValidator(ctx, val)

	delAddr := sdk.AccAddress("delegator1__________").String()
	_, err := k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr,
		Validator: valAddr,
		Amount:    "100000000000000000000",
	})
	if err == nil {
		t.Fatal("expected error for inactive validator")
	}
}

func TestDelegate_ValidatorNotFound(t *testing.T) {
	k, ctx := setupKeeper(t)

	delAddr := sdk.AccAddress("delegator1__________").String()
	_, err := k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr,
		Validator: sdk.AccAddress("nonexistent_________").String(),
		Amount:    "100000000000000000000",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent validator")
	}
}

func TestDelegate_IncreasesExistingDelegation(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator1__________").String()

	// First delegation
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "100000000000000000000",
	})

	// Second delegation
	resp, err := k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "200000000000000000000",
	})
	if err != nil {
		t.Fatalf("delegate failed: %v", err)
	}
	if resp.NewDelegation != "300000000000000000000" {
		t.Errorf("expected total delegation 300000000000000000000, got %s", resp.NewDelegation)
	}
}

// ---------- Undelegate Tests ----------

func TestUndelegate_Success(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator1__________").String()

	// Delegate first
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "500000000000000000000",
	})

	// Undelegate partially
	resp, err := k.Undelegate(ctx, &types.MsgUndelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "200000000000000000000",
	})
	if err != nil {
		t.Fatalf("undelegate failed: %v", err)
	}
	if resp.CompletesAtHeight != 100+268560 {
		t.Errorf("expected completes at %d, got %d", 100+268560, resp.CompletesAtHeight)
	}

	// Verify delegation reduced
	del, found := k.GetDelegation(ctx, delAddr, valAddr)
	if !found {
		t.Fatal("delegation should still exist")
	}
	if del.Amount != "300000000000000000000" {
		t.Errorf("expected remaining delegation 300000000000000000000, got %s", del.Amount)
	}

	// Verify unbonding entry created
	unbonding, found := k.GetUnbonding(ctx, resp.UnbondingId)
	if !found {
		t.Fatal("unbonding entry not found")
	}
	if unbonding.Amount != "200000000000000000000" {
		t.Errorf("expected unbonding amount 200000000000000000000, got %s", unbonding.Amount)
	}
}

func TestUndelegate_FullAmount_DeletesDelegation(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator1__________").String()

	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "500000000000000000000",
	})

	// Undelegate full amount
	_, err := k.Undelegate(ctx, &types.MsgUndelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "500000000000000000000",
	})
	if err != nil {
		t.Fatalf("undelegate failed: %v", err)
	}

	// Delegation should be deleted
	_, found := k.GetDelegation(ctx, delAddr, valAddr)
	if found {
		t.Error("delegation should have been deleted")
	}
}

func TestUndelegate_InsufficientDelegation(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator1__________").String()

	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "100000000000000000000",
	})

	_, err := k.Undelegate(ctx, &types.MsgUndelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "200000000000000000000",
	})
	if err == nil {
		t.Fatal("expected error for insufficient delegation")
	}
}

func TestUndelegate_NoDelegation(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "1000000000000000000000")

	_, err := k.Undelegate(ctx, &types.MsgUndelegate{
		Delegator: sdk.AccAddress("delegator1__________").String(),
		Validator: valAddr,
		Amount:    "100000000000000000000",
	})
	if err == nil {
		t.Fatal("expected error for missing delegation")
	}
}

// ---------- Redelegate Tests ----------

func TestRedelegate_Success(t *testing.T) {
	k, ctx := setupKeeper(t)

	val1Addr := sdk.AccAddress("validator1__________").String()
	val2Addr := sdk.AccAddress("validator2__________").String()
	registerTestValidator(t, k, ctx, val1Addr, "did:lgm:val1", "1000000000000000000000")
	registerTestValidator(t, k, ctx, val2Addr, "did:lgm:val2", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator1__________").String()

	// Delegate to val1
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: val1Addr, Amount: "500000000000000000000",
	})

	// Redelegate from val1 to val2
	_, err := k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator:    delAddr,
		SrcValidator: val1Addr,
		DstValidator: val2Addr,
		Amount:       "300000000000000000000",
	})
	if err != nil {
		t.Fatalf("redelegate failed: %v", err)
	}

	// Verify source delegation reduced
	srcDel, found := k.GetDelegation(ctx, delAddr, val1Addr)
	if !found {
		t.Fatal("source delegation should still exist")
	}
	if srcDel.Amount != "200000000000000000000" {
		t.Errorf("expected src delegation 200000000000000000000, got %s", srcDel.Amount)
	}

	// Verify destination delegation created
	dstDel, found := k.GetDelegation(ctx, delAddr, val2Addr)
	if !found {
		t.Fatal("destination delegation not found")
	}
	if dstDel.Amount != "300000000000000000000" {
		t.Errorf("expected dst delegation 300000000000000000000, got %s", dstDel.Amount)
	}

	// Verify validator DelegatedStake updated
	val1, _ := k.GetValidator(ctx, val1Addr)
	if val1.DelegatedStake != "200000000000000000000" {
		t.Errorf("expected val1 delegated stake 200000000000000000000, got %s", val1.DelegatedStake)
	}
	val2, _ := k.GetValidator(ctx, val2Addr)
	if val2.DelegatedStake != "300000000000000000000" {
		t.Errorf("expected val2 delegated stake 300000000000000000000, got %s", val2.DelegatedStake)
	}
}

func TestRedelegate_FullAmount_DeletesSrcDelegation(t *testing.T) {
	k, ctx := setupKeeper(t)

	val1Addr := sdk.AccAddress("validator1__________").String()
	val2Addr := sdk.AccAddress("validator2__________").String()
	registerTestValidator(t, k, ctx, val1Addr, "did:lgm:val1", "1000000000000000000000")
	registerTestValidator(t, k, ctx, val2Addr, "did:lgm:val2", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator1__________").String()
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: val1Addr, Amount: "500000000000000000000",
	})

	_, err := k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val1Addr, DstValidator: val2Addr,
		Amount: "500000000000000000000",
	})
	if err != nil {
		t.Fatalf("redelegate failed: %v", err)
	}

	_, found := k.GetDelegation(ctx, delAddr, val1Addr)
	if found {
		t.Error("source delegation should have been deleted")
	}
}

func TestRedelegate_SameValidator_Error(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "1000000000000000000000")

	msg := &types.MsgRedelegate{
		Delegator:    sdk.AccAddress("delegator1__________").String(),
		SrcValidator: valAddr,
		DstValidator: valAddr,
		Amount:       "100000000000000000000",
	}
	if err := msg.ValidateBasic(); err == nil {
		t.Fatal("expected ValidateBasic to fail for same src/dst validator")
	}
}

func TestRedelegate_InsufficientDelegation(t *testing.T) {
	k, ctx := setupKeeper(t)

	val1Addr := sdk.AccAddress("validator1__________").String()
	val2Addr := sdk.AccAddress("validator2__________").String()
	registerTestValidator(t, k, ctx, val1Addr, "did:lgm:val1", "1000000000000000000000")
	registerTestValidator(t, k, ctx, val2Addr, "did:lgm:val2", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator1__________").String()
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: val1Addr, Amount: "100000000000000000000",
	})

	_, err := k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val1Addr, DstValidator: val2Addr,
		Amount: "500000000000000000000",
	})
	if err == nil {
		t.Fatal("expected error for insufficient delegation")
	}
}

func TestRedelegate_DstValidatorInactive(t *testing.T) {
	k, ctx := setupKeeper(t)

	val1Addr := sdk.AccAddress("validator1__________").String()
	val2Addr := sdk.AccAddress("validator2__________").String()
	registerTestValidator(t, k, ctx, val1Addr, "did:lgm:val1", "1000000000000000000000")
	registerTestValidator(t, k, ctx, val2Addr, "did:lgm:val2", "0")

	// Deactivate dst
	val2, _ := k.GetValidator(ctx, val2Addr)
	val2.IsActive = false
	k.SetValidator(ctx, val2)

	delAddr := sdk.AccAddress("delegator1__________").String()
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: val1Addr, Amount: "100000000000000000000",
	})

	_, err := k.Redelegate(ctx, &types.MsgRedelegate{
		Delegator: delAddr, SrcValidator: val1Addr, DstValidator: val2Addr,
		Amount: "100000000000000000000",
	})
	if err == nil {
		t.Fatal("expected error for inactive destination validator")
	}
}

// ---------- ValidateBasic Tests ----------

func TestMsgDelegate_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgDelegate
		wantErr bool
	}{
		{"valid", types.MsgDelegate{
			Delegator: sdk.AccAddress("delegator1__________").String(),
			Validator: sdk.AccAddress("validator1__________").String(),
			Amount:    "100000000000000000000",
		}, false},
		{"empty amount", types.MsgDelegate{
			Delegator: sdk.AccAddress("delegator1__________").String(),
			Validator: sdk.AccAddress("validator1__________").String(),
			Amount:    "",
		}, true},
		{"zero amount", types.MsgDelegate{
			Delegator: sdk.AccAddress("delegator1__________").String(),
			Validator: sdk.AccAddress("validator1__________").String(),
			Amount:    "0",
		}, true},
		{"invalid delegator", types.MsgDelegate{
			Delegator: "invalid",
			Validator: sdk.AccAddress("validator1__________").String(),
			Amount:    "100",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMsgUndelegate_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgUndelegate
		wantErr bool
	}{
		{"valid", types.MsgUndelegate{
			Delegator: sdk.AccAddress("delegator1__________").String(),
			Validator: sdk.AccAddress("validator1__________").String(),
			Amount:    "100000000000000000000",
		}, false},
		{"empty amount", types.MsgUndelegate{
			Delegator: sdk.AccAddress("delegator1__________").String(),
			Validator: sdk.AccAddress("validator1__________").String(),
			Amount:    "0",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestMsgRedelegate_ValidateBasic(t *testing.T) {
	tests := []struct {
		name    string
		msg     types.MsgRedelegate
		wantErr bool
	}{
		{"valid", types.MsgRedelegate{
			Delegator:    sdk.AccAddress("delegator1__________").String(),
			SrcValidator: sdk.AccAddress("validator1__________").String(),
			DstValidator: sdk.AccAddress("validator2__________").String(),
			Amount:       "100",
		}, false},
		{"same src dst", types.MsgRedelegate{
			Delegator:    sdk.AccAddress("delegator1__________").String(),
			SrcValidator: sdk.AccAddress("validator1__________").String(),
			DstValidator: sdk.AccAddress("validator1__________").String(),
			Amount:       "100",
		}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.msg.ValidateBasic()
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateBasic() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
