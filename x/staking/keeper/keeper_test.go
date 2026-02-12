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
		ValidatorAddress: addr,
		PubKey:           []byte("abcdef0123456789"),
		SelfDelegation:   stake,
		CommissionRate:   1000,
		Description:      &types.ValidatorDescription{Identity: did},
	})
	if err != nil {
		t.Fatalf("failed to register validator %s: %v", addr, err)
	}
	_ = resp
}

// ---------- Registration Deposit & Cap Tests ----------

func TestRegisterValidator_InsufficientDeposit(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_nodeposit_").String()

	// 0 stake should fail
	_, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
		ValidatorAddress: valAddr,
		PubKey:           []byte("abcdef0123456789"),
		SelfDelegation:   "0",
		CommissionRate:   1000,
		Description:      &types.ValidatorDescription{Identity: "did:lgm:nodeposit"},
	})
	if err == nil {
		t.Fatal("expected error for zero deposit")
	}

	// Below minimum (110999 < 111000) should fail
	_, err = k.RegisterValidator(ctx, &types.MsgRegisterValidator{
		ValidatorAddress: valAddr,
		PubKey:           []byte("abcdef0123456789"),
		SelfDelegation:   "110999",
		CommissionRate:   1000,
		Description:      &types.ValidatorDescription{Identity: "did:lgm:nodeposit"},
	})
	if err == nil {
		t.Fatal("expected error for insufficient deposit")
	}
}

func TestRegisterValidator_ExactMinDeposit(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_mindepo___").String()

	// Exact minimum (111000) should succeed
	resp, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
		ValidatorAddress: valAddr,
		PubKey:           []byte("abcdef0123456789"),
		SelfDelegation:   "111000",
		CommissionRate:   1000,
		Description:      &types.ValidatorDescription{Identity: "did:lgm:mindepo"},
	})
	if err != nil {
		t.Fatalf("expected success for exact min deposit, got: %v", err)
	}
	if resp.InitialTier != types.TierApprentice {
		t.Errorf("expected apprentice tier, got %v", resp.InitialTier)
	}
}

func TestRegisterValidator_UnlimitedVerifiers(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set MaxValidators very low to prove verifiers bypass the cap
	params := k.GetParams(ctx)
	params.MaxValidators = 2
	k.SetParams(ctx, params)

	// Register many tier 0/1 validators — should all succeed
	for i := 0; i < 10; i++ {
		addr := sdk.AccAddress(fmt.Sprintf("validator_%04d______", i)).String()
		_, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
			ValidatorAddress: addr,
			PubKey:           []byte(fmt.Sprintf("pubkey_%04d", i)),
			SelfDelegation:   "111000", // tier 0
			CommissionRate:   1000,
			Description:      &types.ValidatorDescription{Identity: fmt.Sprintf("did:lgm:val%04d", i)},
		})
		if err != nil {
			t.Fatalf("verifier #%d registration should succeed, got: %v", i, err)
		}
	}
}

func TestRegisterValidator_BlockProducerCap(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set block producer cap to 2
	params := k.GetParams(ctx)
	params.MaxValidators = 2
	k.SetParams(ctx, params)

	// Register 2 block producers (tier 2 bonded: 11.1 LGM = 11100000 ulgm)
	for i := 0; i < 2; i++ {
		addr := sdk.AccAddress(fmt.Sprintf("blockprod_%04d______", i)).String()
		_, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
			ValidatorAddress: addr,
			PubKey:           []byte(fmt.Sprintf("bppubkey_%04d", i)),
			SelfDelegation:   "11100000", // tier 2 bonded
			CommissionRate:   1000,
			Description:      &types.ValidatorDescription{Identity: fmt.Sprintf("did:lgm:bp%04d", i)},
		})
		if err != nil {
			t.Fatalf("block producer #%d should succeed, got: %v", i, err)
		}
	}

	// Third block producer should be rejected
	addr := sdk.AccAddress("blockprod_0002______").String()
	_, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
		ValidatorAddress: addr,
		PubKey:           []byte("bppubkey_0002"),
		SelfDelegation:   "11100000",
		CommissionRate:   1000,
		Description:      &types.ValidatorDescription{Identity: "did:lgm:bp0002"},
	})
	if err == nil {
		t.Fatal("expected error: block producer cap should be enforced")
	}

	// But a verifier (tier 0) should still succeed
	verAddr := sdk.AccAddress("verifier_extra______").String()
	_, err = k.RegisterValidator(ctx, &types.MsgRegisterValidator{
		ValidatorAddress: verAddr,
		PubKey:           []byte("verpubkey_extra"),
		SelfDelegation:   "111000", // tier 0
		CommissionRate:   1000,
		Description:      &types.ValidatorDescription{Identity: "did:lgm:verextra"},
	})
	if err != nil {
		t.Fatalf("verifier should bypass block producer cap, got: %v", err)
	}
}

func TestCountBlockProducers(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register mix of tiers
	registerTestValidator(t, k, ctx, sdk.AccAddress("val_apprentice______").String(), "did:lgm:app1", "111000")         // tier 0
	registerTestValidator(t, k, ctx, sdk.AccAddress("val_bonded__________").String(), "did:lgm:bonded1", "11100000")     // tier 2
	registerTestValidator(t, k, ctx, sdk.AccAddress("val_guardian________").String(), "did:lgm:guardian1", "111000000")   // tier 3

	count := k.CountBlockProducers(ctx)
	if count != 2 {
		t.Errorf("expected 2 block producers (bonded+guardian), got %d", count)
	}

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
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "111000")

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
	registerTestValidator(t, k, ctx, val2Addr, "did:lgm:val2", "111000")

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

// ---------- Unbonding ID Collision Tests ----------

func TestUndelegate_SameBlock_NoDuplicateId(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "1000000000000000000000")

	delAddr := sdk.AccAddress("delegator1__________").String()

	// Delegate 1000 tokens
	_, err := k.Delegate(ctx, &types.MsgDelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "1000000000000000000000",
	})
	if err != nil {
		t.Fatalf("delegate failed: %v", err)
	}

	// Undelegate 200 at height 100
	resp1, err := k.Undelegate(ctx, &types.MsgUndelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "200000000000000000000",
	})
	if err != nil {
		t.Fatalf("first undelegate failed: %v", err)
	}

	// Undelegate 300 at same height 100
	resp2, err := k.Undelegate(ctx, &types.MsgUndelegate{
		Delegator: delAddr, Validator: valAddr, Amount: "300000000000000000000",
	})
	if err != nil {
		t.Fatalf("second undelegate failed: %v", err)
	}

	// IDs must be different
	if resp1.UnbondingId == resp2.UnbondingId {
		t.Fatalf("unbonding IDs collided: both are %s", resp1.UnbondingId)
	}

	// Both entries must exist with correct amounts
	entry1, found := k.GetUnbonding(ctx, resp1.UnbondingId)
	if !found {
		t.Fatal("first unbonding entry not found")
	}
	if entry1.Amount != "200000000000000000000" {
		t.Errorf("expected first entry amount 200000000000000000000, got %s", entry1.Amount)
	}

	entry2, found := k.GetUnbonding(ctx, resp2.UnbondingId)
	if !found {
		t.Fatal("second unbonding entry not found")
	}
	if entry2.Amount != "300000000000000000000" {
		t.Errorf("expected second entry amount 300000000000000000000, got %s", entry2.Amount)
	}
}

func TestUpdateValidatorStake_DecreaseDouble_SameBlock(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator1__________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:val1", "1000000000000000000000")

	// Decrease 200 at height 100
	resp1, err := k.UpdateValidatorStake(ctx, &types.MsgUpdateValidatorStake{
		Validator:  valAddr,
		Amount:     "200000000000000000000",
		IsIncrease: false,
	})
	if err != nil {
		t.Fatalf("first stake decrease failed: %v", err)
	}

	// Decrease 300 at same height 100
	resp2, err := k.UpdateValidatorStake(ctx, &types.MsgUpdateValidatorStake{
		Validator:  valAddr,
		Amount:     "300000000000000000000",
		IsIncrease: false,
	})
	if err != nil {
		t.Fatalf("second stake decrease failed: %v", err)
	}

	// Verify two distinct unbonding entries exist
	entries := k.GetUnbondingsForDelegator(ctx, valAddr)
	if len(entries) != 2 {
		t.Fatalf("expected 2 unbonding entries, got %d", len(entries))
	}

	// Verify amounts (order may vary, check both amounts exist)
	amounts := map[string]bool{}
	for _, e := range entries {
		amounts[e.Amount] = true
	}
	if !amounts["200000000000000000000"] {
		t.Error("missing 200000000000000000000 unbonding entry")
	}
	if !amounts["300000000000000000000"] {
		t.Error("missing 300000000000000000000 unbonding entry")
	}

	// Verify remaining stake
	_ = resp1
	if resp2.NewStake != "500000000000000000000" {
		t.Errorf("expected remaining stake 500000000000000000000, got %s", resp2.NewStake)
	}
}

func TestNextUnbondingSeq_Monotonic(t *testing.T) {
	k, ctx := setupKeeper(t)

	seq1 := k.NextUnbondingSeq(ctx)
	seq2 := k.NextUnbondingSeq(ctx)
	seq3 := k.NextUnbondingSeq(ctx)

	if seq1 != 1 || seq2 != 2 || seq3 != 3 {
		t.Errorf("expected monotonic sequence 1,2,3 but got %d,%d,%d", seq1, seq2, seq3)
	}
}

func TestUnbondingSeq_GenesisRoundtrip(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Generate some sequence values
	k.NextUnbondingSeq(ctx)
	k.NextUnbondingSeq(ctx)
	k.NextUnbondingSeq(ctx)

	// Export genesis
	gs := k.ExportGenesis(ctx)
	if gs.UnbondingSeq != 3 {
		t.Fatalf("expected exported seq 3, got %d", gs.UnbondingSeq)
	}

	// Create fresh keeper and import
	k2, ctx2 := setupKeeper(t)
	if err := k2.InitGenesis(ctx2, gs); err != nil {
		t.Fatalf("init genesis failed: %v", err)
	}

	// Next seq should be 4
	seq := k2.NextUnbondingSeq(ctx2)
	if seq != 4 {
		t.Errorf("expected seq 4 after genesis import, got %d", seq)
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

// ---------- Anti-Grief Remediation Tests (R1-R4) ----------

func TestSlashValidator_R4_SlashFloor(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_floor_____").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:floor", "111000") // min deposit

	// Set stake to 0 manually to test slash floor behavior
	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "0"
	v.TotalStake = "0"
	k.SetValidator(ctx, v)

	// Slash a 0-stake validator — should NOT increment slashCount (R4)
	err := k.SlashValidator(ctx, valAddr, "100", "test_floor")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found after slash")
	}
	if v.SlashCount != 0 {
		t.Errorf("R4: expected slashCount=0 for 0-amount slash, got %d", v.SlashCount)
	}
	if v.SlashesThisEpoch != 0 {
		t.Errorf("R4: expected slashesThisEpoch=0 for 0-amount slash, got %d", v.SlashesThisEpoch)
	}
}

func TestSlashValidator_R2_PerEpochCap(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_cap________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:cap", "1000000000000") // 1M LGM

	// Slash twice — both should succeed
	for i := 0; i < 2; i++ {
		err := k.SlashValidator(ctx, valAddr, "100000", "test_cap")
		if err != nil {
			t.Fatalf("SlashValidator #%d failed: %v", i+1, err)
		}
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}
	if v.SlashCount != 2 {
		t.Errorf("expected slashCount=2, got %d", v.SlashCount)
	}
	if v.SlashesThisEpoch != 2 {
		t.Errorf("expected slashesThisEpoch=2, got %d", v.SlashesThisEpoch)
	}

	// Third slash should be capped (R2)
	err := k.SlashValidator(ctx, valAddr, "100000", "test_cap_3")
	if err != nil {
		t.Fatalf("SlashValidator #3 failed: %v", err)
	}

	v, found = k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found after capped slash")
	}
	if v.SlashCount != 2 {
		t.Errorf("R2: expected slashCount=2 after cap, got %d", v.SlashCount)
	}
	if v.IsActive != true {
		t.Error("R2: validator should still be active — cap prevented 3rd slash")
	}
}

func TestSlashValidator_R1_SlashDecay(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_decay______").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:decay", "1000000000000") // 1M LGM

	// Slash once
	err := k.SlashValidator(ctx, valAddr, "100000", "test_decay")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}
	if v.SlashCount != 1 {
		t.Fatalf("expected slashCount=1, got %d", v.SlashCount)
	}

	// Simulate epoch boundary — set block height to decay period
	params := k.GetParams(ctx)
	epochBoundary := int64(params.GetSlashDecayPeriodBlocks())
	ctx = ctx.WithBlockHeight(epochBoundary)

	// Run BeginBlocker at epoch boundary
	err = k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker failed: %v", err)
	}

	v, found = k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found after decay")
	}
	// SlashesThisEpoch was 1 (from the slash), so no decay happens this epoch
	if v.SlashCount != 1 {
		t.Errorf("expected slashCount=1 (no decay in slash epoch), got %d", v.SlashCount)
	}
	if v.SlashesThisEpoch != 0 {
		t.Errorf("expected slashesThisEpoch=0 after epoch reset, got %d", v.SlashesThisEpoch)
	}

	// Now simulate NEXT epoch boundary — no slashes this time
	ctx = ctx.WithBlockHeight(epochBoundary * 2)
	err = k.BeginBlocker(ctx)
	if err != nil {
		t.Fatalf("BeginBlocker #2 failed: %v", err)
	}

	v, found = k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found after second epoch")
	}
	// R1: No slashes in the epoch, so slashCount should decay by 1 (1 → 0)
	if v.SlashCount != 0 {
		t.Errorf("R1: expected slashCount=0 after decay, got %d", v.SlashCount)
	}
}

func TestSlashValidator_Deactivation(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_deact_____").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:deact", "1000000000000")

	// Slash 3 times — validator should deactivate at slashCount=3
	for i := 0; i < 3; i++ {
		// Reset epoch counter between slashes (simulating separate epochs)
		v, _ := k.GetValidator(ctx, valAddr)
		v.SlashesThisEpoch = 0
		_ = k.SetValidator(ctx, v)

		err := k.SlashValidator(ctx, valAddr, "100000", "deact_test")
		if err != nil {
			t.Fatalf("SlashValidator #%d failed: %v", i+1, err)
		}
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}
	if v.IsActive {
		t.Error("expected validator to be deactivated after 3 slashes")
	}
	if v.SlashCount != 3 {
		t.Errorf("expected slashCount=3, got %d", v.SlashCount)
	}
}

func TestSlashValidator_LastSlashedHeight(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_height____").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:height", "1000000000000")

	ctx = ctx.WithBlockHeight(42)
	err := k.SlashValidator(ctx, valAddr, "100000", "height_test")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}
	if v.LastSlashedHeight != 42 {
		t.Errorf("expected LastSlashedHeight=42, got %d", v.LastSlashedHeight)
	}
}

func TestSlashValidator_OwnBeforeDelegated(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_owndelg___").String()
	// Register with enough stake for Bonded tier (1.0x slash multiplier)
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:owndelg", "1111000000")

	// Override to precise values for testing: own=1000, delegated=500
	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "1000"
	v.DelegatedStake = "500"
	v.TotalStake = "1500"
	v.Tier = "bonded" // ensure 1.0x slash multiplier
	k.SetValidator(ctx, v)

	// Slash 800 — should take 800 from own stake, 0 from delegated
	err := k.SlashValidator(ctx, valAddr, "800", "test_own_first")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}
	if v.Stake != "200" {
		t.Errorf("expected own stake=200, got %s", v.Stake)
	}
	if v.DelegatedStake != "500" {
		t.Errorf("expected delegated stake=500 (untouched), got %s", v.DelegatedStake)
	}
	if v.TotalStake != "700" {
		t.Errorf("expected total stake=700, got %s", v.TotalStake)
	}
}

func TestSlashValidator_OverflowToDelegated(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_overflow__").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:overflow", "1111000000")

	// Override: own=300, delegated=700
	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "300"
	v.DelegatedStake = "700"
	v.TotalStake = "1000"
	v.Tier = "bonded"
	k.SetValidator(ctx, v)

	// Slash 500 — should take 300 from own, 200 from delegated
	err := k.SlashValidator(ctx, valAddr, "500", "test_overflow")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}
	if v.Stake != "0" {
		t.Errorf("expected own stake=0, got %s", v.Stake)
	}
	if v.DelegatedStake != "500" {
		t.Errorf("expected delegated stake=500, got %s", v.DelegatedStake)
	}
	if v.TotalStake != "500" {
		t.Errorf("expected total stake=500, got %s", v.TotalStake)
	}
}

func TestSlashValidator_FullSlash(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_fullslash_").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:fullslash", "1111000000")

	// Override: own=200, delegated=300
	v, _ := k.GetValidator(ctx, valAddr)
	v.Stake = "200"
	v.DelegatedStake = "300"
	v.TotalStake = "500"
	v.Tier = "bonded"
	k.SetValidator(ctx, v)

	// Slash 9999 — should cap at total and drain both
	err := k.SlashValidator(ctx, valAddr, "9999", "test_full")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}
	if v.Stake != "0" {
		t.Errorf("expected own stake=0, got %s", v.Stake)
	}
	if v.DelegatedStake != "0" {
		t.Errorf("expected delegated stake=0, got %s", v.DelegatedStake)
	}
	if v.TotalStake != "0" {
		t.Errorf("expected total stake=0, got %s", v.TotalStake)
	}
}

func TestSlashValidator_ProgressiveEscalation(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_escalate__").String()
	// Use enough stake to stay in Bonded tier after slashing
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:escalate", "1111000000000")

	// Increase per-epoch cap and deactivation threshold to allow 3+ slashes
	params := k.GetParams(ctx)
	params.MaxSlashesPerEpoch = 5
	params.MaxSlashCountDeactivate = 10
	k.SetParams(ctx, params)

	v, _ := k.GetValidator(ctx, valAddr)
	if v.Tier != "bonded" {
		t.Fatalf("expected bonded tier, got %s", v.Tier)
	}

	// First slash: 1000000, slashCount=0 → no escalation → takes 1000000 (bonded 1.0x)
	err := k.SlashValidator(ctx, valAddr, "1000000", "first")
	if err != nil {
		t.Fatalf("first slash failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	// 1111000000000 - 1000000 = 1110999000000
	if v.Stake != "1110999000000" {
		t.Errorf("after first slash: expected stake=1110999000000, got %s", v.Stake)
	}
	if v.SlashCount != 1 {
		t.Errorf("expected slashCount=1, got %d", v.SlashCount)
	}

	// Second slash: 1000000, slashCount=1 → escalation 10% → 1100000 effective
	err = k.SlashValidator(ctx, valAddr, "1000000", "second")
	if err != nil {
		t.Fatalf("second slash failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	// 1110999000000 - 1100000 = 1110997900000
	if v.Stake != "1110997900000" {
		t.Errorf("after second slash: expected stake=1110997900000, got %s", v.Stake)
	}

	// Third slash: 1000000, slashCount=2 → escalation 20% → 1200000 effective
	err = k.SlashValidator(ctx, valAddr, "1000000", "third")
	if err != nil {
		t.Fatalf("third slash failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	// 1110997900000 - 1200000 = 1110996700000
	if v.Stake != "1110996700000" {
		t.Errorf("after third slash: expected stake=1110996700000, got %s", v.Stake)
	}
	if v.SlashCount != 3 {
		t.Errorf("expected slashCount=3, got %d", v.SlashCount)
	}
}
