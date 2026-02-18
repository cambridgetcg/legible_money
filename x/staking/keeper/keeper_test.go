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
func (m *mockAccountKeeper) SetAccount(_ context.Context, _ sdk.AccountI)                {}
func (m *mockAccountKeeper) NewAccountWithAddress(_ context.Context, _ sdk.AccAddress) sdk.AccountI {
	return nil
}

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

	k := keeper.NewKeeper(cdc, storeService, ak, bk, nil, "authority")
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

// promoteToBonded sets verification records and tier so that a validator with sufficient
// stake qualifies for Tier 2 (Bonded). Must be called after RegisterValidator.
func promoteToBonded(t *testing.T, k keeper.Keeper, ctx sdk.Context, addr string) {
	t.Helper()
	v, found := k.GetValidator(ctx, addr)
	if !found {
		t.Fatalf("promoteToBonded: validator %s not found", addr)
	}
	v.VerificationsCompleted = 20
	v.VerificationsCorrect = 15 // 75% accuracy (> 50% min)
	v.Tier = "bonded"
	if err := k.SetValidator(ctx, v); err != nil {
		t.Fatalf("promoteToBonded: failed to set validator: %v", err)
	}
}

// promoteToGuardian sets verification records and tier for Tier 3 (Guardian) qualification.
func promoteToGuardian(t *testing.T, k keeper.Keeper, ctx sdk.Context, addr string) {
	t.Helper()
	v, found := k.GetValidator(ctx, addr)
	if !found {
		t.Fatalf("promoteToGuardian: validator %s not found", addr)
	}
	v.VerificationsCompleted = 1200
	v.VerificationsCorrect = 1000 // ~83% accuracy (> 77% min)
	v.ContestedVerificationsCorrect = 33 // P2-2 fix: Guardian requires MinContestedVerifications=33
	v.SlashCount = 0
	v.Tier = "guardian"
	if err := k.SetValidator(ctx, v); err != nil {
		t.Fatalf("promoteToGuardian: failed to set validator: %v", err)
	}
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

	// Register 3 validators with bonded-level stake.
	// All start as apprentice (0 verifications), so the cap check at registration
	// doesn't fire — it only triggers for initialTier >= Bonded.
	addrs := make([]string, 3)
	for i := 0; i < 3; i++ {
		addrs[i] = sdk.AccAddress(fmt.Sprintf("blockprod_%04d______", i)).String()
		_, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
			ValidatorAddress: addrs[i],
			PubKey:           []byte(fmt.Sprintf("bppubkey_%04d", i)),
			SelfDelegation:   "1111000000",
			CommissionRate:   1000,
			Description:      &types.ValidatorDescription{Identity: fmt.Sprintf("did:lgm:bp%04d", i)},
		})
		if err != nil {
			t.Fatalf("registration #%d should succeed (apprentice tier), got: %v", i, err)
		}
	}

	// Promote first two to bonded — they become block producers
	promoteToBonded(t, k, ctx, addrs[0])
	promoteToBonded(t, k, ctx, addrs[1])

	count := k.CountBlockProducers(ctx)
	if count != 2 {
		t.Fatalf("expected 2 block producers, got %d", count)
	}

	// Third validator remains apprentice — block producer count stays at 2
	v, _ := k.GetValidator(ctx, addrs[2])
	if v.Tier != "apprentice" {
		t.Errorf("third validator should still be apprentice, got %s", v.Tier)
	}

	// Verifier (tier 0 stake) registration succeeds regardless of cap
	verAddr := sdk.AccAddress("verifier_extra______").String()
	_, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
		ValidatorAddress: verAddr,
		PubKey:           []byte("verpubkey_extra"),
		SelfDelegation:   "111000",
		CommissionRate:   1000,
		Description:      &types.ValidatorDescription{Identity: "did:lgm:verextra"},
	})
	if err != nil {
		t.Fatalf("verifier registration should succeed, got: %v", err)
	}
}

func TestCountBlockProducers(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Register mix of tiers (all start as apprentice, then promote)
	appAddr := sdk.AccAddress("val_apprentice______").String()
	bondedAddr := sdk.AccAddress("val_bonded__________").String()
	guardianAddr := sdk.AccAddress("val_guardian________").String()

	registerTestValidator(t, k, ctx, appAddr, "did:lgm:app1", "111000")                // tier 0
	registerTestValidator(t, k, ctx, bondedAddr, "did:lgm:bonded1", "1111000000")      // tier 2 stake
	registerTestValidator(t, k, ctx, guardianAddr, "did:lgm:guardian1", "11111000000") // tier 3 stake

	// Promote to actual tiers via verification records
	promoteToBonded(t, k, ctx, bondedAddr)
	promoteToGuardian(t, k, ctx, guardianAddr)

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
	promoteToBonded(t, k, ctx, valAddr)

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

// ---------- Session-7 Hardening Tests ----------

// P0-1: Reputation underflow edge cases
func TestRecordVerification_ReputationUnderflowEdge(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_repedge___").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:repedge", "1111000000")

	// Set reputation to exactly 200 — should decrement to 0
	v, _ := k.GetValidator(ctx, valAddr)
	v.Reputation = 200
	k.SetValidator(ctx, v)

	err := k.RecordVerification(ctx, valAddr, false, false)
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	if v.Reputation != 0 {
		t.Errorf("P0-1: expected reputation=0 after underflow edge, got %d", v.Reputation)
	}

	// Set reputation to 199 — should clamp to 0, not underflow
	v.Reputation = 199
	k.SetValidator(ctx, v)
	err = k.RecordVerification(ctx, valAddr, false, false)
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	if v.Reputation != 0 {
		t.Errorf("P0-1: expected reputation=0 for value 199, got %d", v.Reputation)
	}

	// Set reputation to 1 — should clamp to 0
	v.Reputation = 1
	k.SetValidator(ctx, v)
	err = k.RecordVerification(ctx, valAddr, false, false)
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	if v.Reputation != 0 {
		t.Errorf("P0-1: expected reputation=0 for value 1, got %d", v.Reputation)
	}

	// Set reputation to 0 — should stay 0
	v.Reputation = 0
	k.SetValidator(ctx, v)
	err = k.RecordVerification(ctx, valAddr, false, false)
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	if v.Reputation != 0 {
		t.Errorf("P0-1: expected reputation=0 for value 0, got %d", v.Reputation)
	}
}

// P0-2: ContestedVerificationsCorrect tracking
func TestRecordVerification_ContestedTracking(t *testing.T) {
	k, ctx := setupKeeper(t)
	valAddr := sdk.AccAddress("validator_contested_").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:contested", "1111000000")

	// Record a correct non-contested verification
	err := k.RecordVerification(ctx, valAddr, true, false)
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}
	v, _ := k.GetValidator(ctx, valAddr)
	if v.VerificationsCompleted != 1 {
		t.Errorf("expected VerificationsCompleted=1, got %d", v.VerificationsCompleted)
	}
	if v.VerificationsCorrect != 1 {
		t.Errorf("expected VerificationsCorrect=1, got %d", v.VerificationsCorrect)
	}
	if v.ContestedVerificationsCorrect != 0 {
		t.Errorf("P0-2: expected ContestedVerificationsCorrect=0 for non-contested, got %d", v.ContestedVerificationsCorrect)
	}

	// Record a correct contested verification
	err = k.RecordVerification(ctx, valAddr, true, true)
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	if v.ContestedVerificationsCorrect != 1 {
		t.Errorf("P0-2: expected ContestedVerificationsCorrect=1, got %d", v.ContestedVerificationsCorrect)
	}

	// Record an incorrect contested verification — should NOT increment ContestedVerificationsCorrect
	err = k.RecordVerification(ctx, valAddr, false, true)
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}
	v, _ = k.GetValidator(ctx, valAddr)
	if v.ContestedVerificationsCorrect != 1 {
		t.Errorf("P0-2: expected ContestedVerificationsCorrect=1 (wrong verdict), got %d", v.ContestedVerificationsCorrect)
	}
	if v.VerificationsCompleted != 3 {
		t.Errorf("expected VerificationsCompleted=3, got %d", v.VerificationsCompleted)
	}
}

// P0-3: Epoch length consistency (reads from params, not hardcoded)
func TestComputeValidatorTier_GuardianSlashWindow(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Set slash decay period to 34272 (1 day at 2.521s/block)
	params := k.GetParams(ctx)
	params.SlashDecayPeriodBlocks = 34272
	k.SetParams(ctx, params)

	// A guardian-eligible validator slashed 1 epoch ago (34272 blocks)
	// With 10-epoch window, slash should still be within window
	ctx = ctx.WithBlockHeight(342722) // well past 10 epochs (window = 10*34272 = 342720)
	tier := keeper.ComputeValidatorTier(
		ctx, k,
		"11111000000",   // 11,111 LGM (guardian stake)
		1200,            // verifications
		1000,            // correct (83%)
		1,               // 1 slash
		33,              // contested correct
		342720,          // last slash 1 block ago — within 10-epoch window
	)
	// Should NOT qualify for guardian because slash is within window
	if tier == types.TierGuardian {
		t.Error("P0-3: validator with recent slash should not qualify for guardian")
	}

	// Same validator, but last slash was 11+ epochs ago (outside 10-epoch window)
	// lastSlashHeight=0 is treated as "unknown" → conservative (returns full count)
	// Use lastSlashHeight=1 so the window calculation actually runs
	tier = keeper.ComputeValidatorTier(
		ctx, k,
		"11111000000",
		1200,
		1000,
		1,
		33,
		1, // last slash at height 1 — 342720 blocks ago, outside 10-epoch window
	)
	// With windowed slashes, old slash should be ignored → guardian eligible
	if tier != types.TierGuardian {
		t.Errorf("P0-3: validator with old slash should qualify for guardian, got %v", tier)
	}
}

// P1-1: GetDelegationsForValidator via reverse index
func TestGetDelegationsForValidator_ReverseIndex(t *testing.T) {
	k, ctx := setupKeeper(t)

	val1Addr := sdk.AccAddress("validator_ri1_______").String()
	val2Addr := sdk.AccAddress("validator_ri2_______").String()
	registerTestValidator(t, k, ctx, val1Addr, "did:lgm:ri1", "1000000000000000000000")
	registerTestValidator(t, k, ctx, val2Addr, "did:lgm:ri2", "1000000000000000000000")

	del1Addr := sdk.AccAddress("delegator_ri1_______").String()
	del2Addr := sdk.AccAddress("delegator_ri2_______").String()
	del3Addr := sdk.AccAddress("delegator_ri3_______").String()

	// Delegate to val1 from del1 and del2
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: del1Addr, Validator: val1Addr, Amount: "100000000000000000000",
	})
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: del2Addr, Validator: val1Addr, Amount: "200000000000000000000",
	})

	// Delegate to val2 from del3
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: del3Addr, Validator: val2Addr, Amount: "300000000000000000000",
	})

	// Query delegations for val1 — should return self-delegation + del1 + del2
	// (RegisterValidator creates a self-delegation)
	delegations := k.GetDelegationsForValidator(ctx, val1Addr)
	if len(delegations) != 3 {
		t.Fatalf("P1-1: expected 3 delegations for val1 (self+del1+del2), got %d", len(delegations))
	}

	// Verify all delegators are present (including self-delegation)
	found := map[string]bool{}
	for _, d := range delegations {
		found[d.DelegatorAddress] = true
		if d.ValidatorAddress != val1Addr {
			t.Errorf("P1-1: delegation should be for val1, got %s", d.ValidatorAddress)
		}
	}
	if !found[val1Addr] {
		t.Error("P1-1: missing self-delegation from val1")
	}
	if !found[del1Addr] {
		t.Error("P1-1: missing delegation from del1")
	}
	if !found[del2Addr] {
		t.Error("P1-1: missing delegation from del2")
	}

	// Query delegations for val2 — should return self-delegation + del3
	delegations = k.GetDelegationsForValidator(ctx, val2Addr)
	if len(delegations) != 2 {
		t.Fatalf("P1-1: expected 2 delegations for val2 (self+del3), got %d", len(delegations))
	}
	foundVal2 := map[string]bool{}
	for _, d := range delegations {
		foundVal2[d.DelegatorAddress] = true
	}
	if !foundVal2[val2Addr] {
		t.Error("P1-1: missing self-delegation from val2")
	}
	if !foundVal2[del3Addr] {
		t.Error("P1-1: missing delegation from del3")
	}
}

// P1-1: Reverse index cleanup on undelegation
func TestGetDelegationsForValidator_UndelegateCleanup(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_undclean__").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:undclean", "1000000000000000000000")

	del1Addr := sdk.AccAddress("delegator_uc1_______").String()
	del2Addr := sdk.AccAddress("delegator_uc2_______").String()

	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: del1Addr, Validator: valAddr, Amount: "100000000000000000000",
	})
	k.Delegate(ctx, &types.MsgDelegate{
		Delegator: del2Addr, Validator: valAddr, Amount: "200000000000000000000",
	})

	// Undelegate full amount from del1
	k.Undelegate(ctx, &types.MsgUndelegate{
		Delegator: del1Addr, Validator: valAddr, Amount: "100000000000000000000",
	})

	// Should now have self-delegation + del2
	delegations := k.GetDelegationsForValidator(ctx, valAddr)
	if len(delegations) != 2 {
		t.Fatalf("P1-1: expected 2 delegations after undelegate (self+del2), got %d", len(delegations))
	}
	foundDelegators := map[string]bool{}
	for _, d := range delegations {
		foundDelegators[d.DelegatorAddress] = true
	}
	if !foundDelegators[valAddr] {
		t.Error("P1-1: missing self-delegation")
	}
	if !foundDelegators[del2Addr] {
		t.Error("P1-1: expected del2 to remain after del1 undelegation")
	}
}

// Guardian tier: contested verifications requirement enforced
func TestComputeValidatorTier_GuardianContestedRequired(t *testing.T) {
	k, ctx := setupKeeper(t)

	// All guardian requirements met EXCEPT contested verifications
	tier := keeper.ComputeValidatorTier(
		ctx, k,
		"11111000000", // 11,111 LGM
		1200,          // verifications
		1000,          // correct (83%)
		0,             // no slashes
		32,            // only 32 contested (need 33)
		0,             // no slash history
	)
	if tier == types.TierGuardian {
		t.Error("should not qualify for guardian with only 32 contested verifications")
	}
	if tier != types.TierBonded {
		t.Errorf("expected bonded tier fallback, got %v", tier)
	}

	// Now with exactly 33 — should qualify
	tier = keeper.ComputeValidatorTier(
		ctx, k,
		"11111000000",
		1200,
		1000,
		0,
		33,
		0,
	)
	if tier != types.TierGuardian {
		t.Errorf("should qualify for guardian with 33 contested, got %v", tier)
	}
}

// Guardian tier with contested verification multiplier (3x)
func TestComputeValidatorTier_ContestedMultiplier(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Guardian requires MinVerifications=333 with contested multiplier 3x
	// With 100 total verifications and 50 contested correct:
	// effective = (100-50) + 50*(1+(3-1)) = 50 + 150 = 200
	// 200 < 333 → should not qualify
	tier := keeper.ComputeValidatorTier(
		ctx, k,
		"11111000000",
		100,  // verifications
		80,   // correct
		0,    // no slashes
		50,   // contested correct
		0,
	)
	if tier == types.TierGuardian {
		t.Error("200 effective verifications should not meet 333 requirement")
	}

	// With 200 total and 100 contested:
	// effective = (200-100) + 100*3 = 100 + 300 = 400
	// 400 >= 333 → should qualify (if all other requirements met)
	tier = keeper.ComputeValidatorTier(
		ctx, k,
		"11111000000",
		200,  // verifications
		180,  // correct (90%)
		0,    // no slashes
		100,  // contested correct
		0,
	)
	if tier != types.TierGuardian {
		t.Errorf("400 effective verifications should meet 333 requirement, got %v", tier)
	}
}

// P2-2: promoteToGuardian helper correctness — verify it sets 33 contested
func TestPromoteToGuardian_MeetsAllRequirements(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("validator_guardian___").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:guardian", "11111000000")
	promoteToGuardian(t, k, ctx, valAddr)

	v, found := k.GetValidator(ctx, valAddr)
	if !found {
		t.Fatal("validator not found")
	}

	// Verify the helper sets values that actually qualify for guardian
	computedTier := keeper.ComputeValidatorTier(
		ctx, k,
		v.TotalStake,
		v.VerificationsCompleted,
		v.VerificationsCorrect,
		v.SlashCount,
		v.ContestedVerificationsCorrect,
		v.LastSlashedHeight,
	)
	if computedTier != types.TierGuardian {
		t.Errorf("P2-2: promoteToGuardian helper should produce a validator that computes as guardian, got %v", computedTier)
	}
}

// ==========================================================================
// R2 Session 6: 4-Tier Validator Lifecycle Tests
// ==========================================================================

// ---------- Task 1: Full Tier Progression Integration Test ----------

func TestTierProgression_ApprenticeToGuardian(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_lifecycle________").String()

	// Step 1: Register with 0.111 LGM → Apprentice
	resp, err := k.RegisterValidator(ctx, &types.MsgRegisterValidator{
		ValidatorAddress: valAddr,
		PubKey:           []byte("lifecycle_pubkey"),
		SelfDelegation:   "111000", // 0.111 LGM in ulgm
		CommissionRate:   1000,
		Description:      &types.ValidatorDescription{Identity: "did:lgm:lifecycle"},
	})
	if err != nil {
		t.Fatalf("Step 1: registration failed: %v", err)
	}
	if resp.InitialTier != types.TierApprentice {
		t.Fatalf("Step 1: expected apprentice, got %v", resp.InitialTier)
	}

	v, _ := k.GetValidator(ctx, valAddr)
	if v.Tier != "apprentice" {
		t.Fatalf("Step 1: stored tier should be apprentice, got %s", v.Tier)
	}

	// Step 2: Perform 22 correct verifications → still Apprentice (need 1.11 LGM for Verified)
	for i := 0; i < 22; i++ {
		if err := k.RecordVerification(ctx, valAddr, true, false); err != nil {
			t.Fatalf("Step 2: RecordVerification #%d failed: %v", i+1, err)
		}
	}

	v, _ = k.GetValidator(ctx, valAddr)
	if v.VerificationsCompleted != 22 {
		t.Fatalf("Step 2: expected 22 verifications, got %d", v.VerificationsCompleted)
	}
	if v.VerificationsCorrect != 22 {
		t.Fatalf("Step 2: expected 22 correct, got %d", v.VerificationsCorrect)
	}
	if v.Tier != "apprentice" {
		t.Fatalf("Step 2: expected still apprentice (insufficient stake for verified), got %s", v.Tier)
	}

	// Step 3: Increase stake to 1.11 LGM → Verified (Tier 1)
	// Need to add 1110000 - 111000 = 999000 more
	_, err = k.UpdateValidatorStake(ctx, &types.MsgUpdateValidatorStake{
		Validator:  valAddr,
		Amount:     "999000",
		IsIncrease: true,
	})
	if err != nil {
		t.Fatalf("Step 3: stake increase failed: %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	if v.Tier != "verified" {
		t.Fatalf("Step 3: expected verified with 1.11 LGM + 22 verifications at 100%%, got %s", v.Tier)
	}

	// Step 4: Verify accuracy ≥ 77% → still Verified (already at 100%)
	accuracy := v.GetAccuracyRate()
	if accuracy < 770000 {
		t.Fatalf("Step 4: expected accuracy ≥ 77%%, got %d bps", accuracy)
	}
	if v.Tier != "verified" {
		t.Fatalf("Step 4: expected verified, got %s", v.Tier)
	}

	// Step 5: Increase stake to 1,111 LGM → Bonded (Tier 2)
	// Current stake is 1,110,000 ulgm. Need 1,111,000,000 ulgm.
	needed := "1109890000" // 1,111,000,000 - 1,110,000 = 1,109,890,000
	_, err = k.UpdateValidatorStake(ctx, &types.MsgUpdateValidatorStake{
		Validator:  valAddr,
		Amount:     needed,
		IsIncrease: true,
	})
	if err != nil {
		t.Fatalf("Step 5: stake increase to bonded failed: %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	// With 1,111 LGM, 22 verifications at 100% accuracy → meets Bonded requirements
	// (Bonded: 1,111 LGM, 11 verifications, 50% accuracy)
	if v.Tier != "bonded" {
		t.Fatalf("Step 5: expected bonded with 1,111 LGM + 22 verifs, got %s", v.Tier)
	}

	// Step 6: Verify we have enough verifications for bonded (already satisfied at step 2)
	if v.VerificationsCompleted < 11 {
		t.Fatalf("Step 6: expected ≥ 11 verifications for bonded, got %d", v.VerificationsCompleted)
	}

	// Step 7: Path to Guardian — increase stake to 11,111 LGM
	// Current stake: 1,111,000,000. Need 11,111,000,000.
	_, err = k.UpdateValidatorStake(ctx, &types.MsgUpdateValidatorStake{
		Validator:  valAddr,
		Amount:     "10000000000", // add 10,000 LGM
		IsIncrease: true,
	})
	if err != nil {
		t.Fatalf("Step 7: stake increase to guardian-level failed: %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	// Still bonded — need more verifications + contested + accuracy check
	if v.Tier == "guardian" {
		t.Fatalf("Step 7: should NOT be guardian yet (insufficient verifications/contested)")
	}

	// Step 8: Simulate contested verifications for Guardian requirement
	// Need 333 effective verifications (with 3x multiplier): 11 contested correct gives 33
	// Need MinContestedVerifications=33
	// We have 22 verifications already. Let's add more:
	// Strategy: 78 more normal correct verifications + 11 contested correct = 89 more
	// Total: 22 + 78 + 11 = 111 total
	// Effective: (111 - 11) + 11*3 = 100 + 33 = 133 ... not enough
	// Need 333 effective. Let's do: normal_correct + contested_correct*3 >= 333
	// With 33 contested correct: (total - 33) + 33*3 = total - 33 + 99 = total + 66
	// Need total + 66 >= 333 → total >= 267
	// Currently have 22. Need 267 - 22 = 245 more verifications total.
	// Of those 245, we need 33 to be contested correct (11 already done? no, we haven't done any contested yet).
	// Let's do: 212 normal correct + 33 contested correct = 245 more
	for i := 0; i < 212; i++ {
		if err := k.RecordVerification(ctx, valAddr, true, false); err != nil {
			t.Fatalf("Step 8: normal verification #%d failed: %v", i+1, err)
		}
	}
	for i := 0; i < 33; i++ {
		if err := k.RecordVerification(ctx, valAddr, true, true); err != nil {
			t.Fatalf("Step 8: contested verification #%d failed: %v", i+1, err)
		}
	}

	v, _ = k.GetValidator(ctx, valAddr)
	// Total: 22 + 212 + 33 = 267 verifications, all correct (100% accuracy)
	// Effective: (267 - 33) + 33*3 = 234 + 99 = 333 → meets 333 requirement
	// Contested correct: 33 → meets MinContestedVerifications=33
	// Stake: 11,111 LGM → meets MinStake
	// Accuracy: 100% → meets 77%
	// SlashCount: 0 → meets MaxSlashCount=0
	if v.VerificationsCompleted != 267 {
		t.Fatalf("Step 8: expected 267 total verifications, got %d", v.VerificationsCompleted)
	}
	if v.ContestedVerificationsCorrect != 33 {
		t.Fatalf("Step 8: expected 33 contested correct, got %d", v.ContestedVerificationsCorrect)
	}
	if v.Tier != "guardian" {
		t.Fatalf("Step 8: expected guardian tier, got %s (stake=%s, verifs=%d, correct=%d, contested=%d, slashes=%d)",
			v.Tier, v.TotalStake, v.VerificationsCompleted, v.VerificationsCorrect,
			v.ContestedVerificationsCorrect, v.SlashCount)
	}
}

// ---------- Task 2: Tier Demotion Tests ----------

func TestTierDemotion_GuardianSlash_ImmediateDemotion(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_gdemotion_______").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:gdemotion", "11111000000") // 11,111 LGM
	promoteToGuardian(t, k, ctx, valAddr)

	v, _ := k.GetValidator(ctx, valAddr)
	if v.Tier != "guardian" {
		t.Fatalf("precondition: expected guardian tier, got %s", v.Tier)
	}

	// Slash the guardian → should immediately drop to Bonded
	// (Guardian requires MaxSlashCount=0, any slash disqualifies)
	err := k.SlashValidator(ctx, valAddr, "100000", "test_guardian_demotion")
	if err != nil {
		t.Fatalf("SlashValidator failed: %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	if v.Tier == "guardian" {
		t.Fatal("guardian should be demoted after slash")
	}
	if v.Tier != "bonded" {
		t.Errorf("expected demotion to bonded (still meets bonded requirements), got %s", v.Tier)
	}
	if v.SlashCount != 1 {
		t.Errorf("expected slashCount=1, got %d", v.SlashCount)
	}
}

func TestTierDemotion_StakeDropsBelowThreshold(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_stakedrop_______").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:stakedrop", "1111000000") // 1,111 LGM
	promoteToBonded(t, k, ctx, valAddr)

	v, _ := k.GetValidator(ctx, valAddr)
	if v.Tier != "bonded" {
		t.Fatalf("precondition: expected bonded, got %s", v.Tier)
	}

	// Decrease stake below bonded threshold (1,111 LGM → 1,110 LGM)
	_, err := k.UpdateValidatorStake(ctx, &types.MsgUpdateValidatorStake{
		Validator:  valAddr,
		Amount:     "1000000", // reduce by 1 LGM
		IsIncrease: false,
	})
	if err != nil {
		t.Fatalf("stake decrease failed: %v", err)
	}

	v, _ = k.GetValidator(ctx, valAddr)
	// New stake: 1,111,000,000 - 1,000,000 = 1,110,000,000 < 1,111,000,000 required
	if v.Tier == "bonded" {
		t.Fatal("should be demoted — stake dropped below bonded threshold")
	}
	// With 20 verifications at 75% accuracy and 1.11 LGM+: should be verified
	// 15 correct / 20 total = 750000 bps (75%) < 770000 (77%) — not enough for verified
	// So it should drop to apprentice
	// Actually: promoteToBonded sets 20 verifs, 15 correct = 75% < 77% for verified
	// But verified also needs 1.11 LGM stake — check: 1,110,000,000 > 1,110,000 ✓
	// Accuracy: 750000 < 770000 — doesn't qualify for verified
	if v.Tier != "apprentice" {
		t.Errorf("expected demotion to apprentice (accuracy 75%% < 77%% for verified), got %s", v.Tier)
	}
}

func TestTierDemotion_AccuracyDropsBelowThreshold(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_accydrop________").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:accydrop", "1110000") // 1.11 LGM (verified stake)

	// Set up as verified: 22 verifications at 100% accuracy
	v, _ := k.GetValidator(ctx, valAddr)
	v.VerificationsCompleted = 22
	v.VerificationsCorrect = 22
	v.Tier = "verified"
	k.SetValidator(ctx, v)

	// Verify precondition
	v, _ = k.GetValidator(ctx, valAddr)
	if v.Tier != "verified" {
		t.Fatalf("precondition: expected verified, got %s", v.Tier)
	}

	// Record 8 incorrect verifications to drop accuracy below 77%
	// Current: 22/22 = 100%. After 8 wrong: 22/30 = 73.3% < 77%
	for i := 0; i < 8; i++ {
		if err := k.RecordVerification(ctx, valAddr, false, false); err != nil {
			t.Fatalf("RecordVerification #%d failed: %v", i+1, err)
		}
	}

	v, _ = k.GetValidator(ctx, valAddr)
	accuracy := v.GetAccuracyRate()
	if accuracy >= 770000 {
		t.Fatalf("expected accuracy < 77%%, got %d bps", accuracy)
	}
	if v.Tier == "verified" {
		t.Fatal("should be demoted — accuracy dropped below verified threshold")
	}
	if v.Tier != "apprentice" {
		t.Errorf("expected demotion to apprentice, got %s", v.Tier)
	}
}

func TestTierDemotion_SlashDecayWindow(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_slashdecay______").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:slashdecay", "11111000000") // 11,111 LGM

	// Set up as bonded with guardian-level stats EXCEPT a recent slash
	v, _ := k.GetValidator(ctx, valAddr)
	v.VerificationsCompleted = 1200
	v.VerificationsCorrect = 1000
	v.ContestedVerificationsCorrect = 33
	v.SlashCount = 1
	v.LastSlashedHeight = 100 // slashed at current block height
	v.Tier = "bonded"        // can't be guardian due to slash
	k.SetValidator(ctx, v)

	// At height 100, slash is within window → should remain bonded, not guardian
	tier := keeper.ComputeValidatorTier(ctx, k, v.TotalStake, v.VerificationsCompleted,
		v.VerificationsCorrect, v.SlashCount, v.ContestedVerificationsCorrect, v.LastSlashedHeight)
	if tier == types.TierGuardian {
		t.Fatal("should NOT be guardian — recent slash within window")
	}

	// Advance past 10-epoch window: 10 * 34272 = 342720 blocks
	params := k.GetParams(ctx)
	epochLength := params.GetSlashDecayPeriodBlocks()
	windowBlocks := 10 * epochLength // 342720
	futureCtx := ctx.WithBlockHeight(int64(100 + windowBlocks + 1))

	// Now the slash is outside the window → should qualify for guardian
	tier = keeper.ComputeValidatorTier(futureCtx, k, v.TotalStake, v.VerificationsCompleted,
		v.VerificationsCorrect, v.SlashCount, v.ContestedVerificationsCorrect, v.LastSlashedHeight)
	if tier != types.TierGuardian {
		t.Errorf("expected guardian after slash window expired, got %v", tier)
	}
}

// ---------- Task 3: Contested Verification Flow ----------

func TestContestedVerification_ChallengeRound(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_challenge_______").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:challenge", "1111000000")

	// Record a correct contested verification (simulating challenge round)
	err := k.RecordVerification(ctx, valAddr, true, true) // contested=true
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}

	v, _ := k.GetValidator(ctx, valAddr)
	if v.ContestedVerificationsCorrect != 1 {
		t.Errorf("challenge round: expected ContestedVerificationsCorrect=1, got %d", v.ContestedVerificationsCorrect)
	}
	if v.VerificationsCompleted != 1 {
		t.Errorf("expected VerificationsCompleted=1, got %d", v.VerificationsCompleted)
	}
	if v.VerificationsCorrect != 1 {
		t.Errorf("expected VerificationsCorrect=1, got %d", v.VerificationsCorrect)
	}
}

func TestContestedVerification_ContradictionRound(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_contradiction___").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:contradiction", "1111000000")

	// Contradiction round is also contested=true
	err := k.RecordVerification(ctx, valAddr, true, true)
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}

	v, _ := k.GetValidator(ctx, valAddr)
	if v.ContestedVerificationsCorrect != 1 {
		t.Errorf("contradiction round: expected ContestedVerificationsCorrect=1, got %d", v.ContestedVerificationsCorrect)
	}
}

func TestContestedVerification_NormalRound(t *testing.T) {
	k, ctx := setupKeeper(t)

	valAddr := sdk.AccAddress("val_normalround_____").String()
	registerTestValidator(t, k, ctx, valAddr, "did:lgm:normalround", "1111000000")

	// Normal round: contested=false → should NOT increment ContestedVerificationsCorrect
	err := k.RecordVerification(ctx, valAddr, true, false)
	if err != nil {
		t.Fatalf("RecordVerification failed: %v", err)
	}

	v, _ := k.GetValidator(ctx, valAddr)
	if v.ContestedVerificationsCorrect != 0 {
		t.Errorf("normal round: expected ContestedVerificationsCorrect=0, got %d", v.ContestedVerificationsCorrect)
	}
	if v.VerificationsCorrect != 1 {
		t.Errorf("expected VerificationsCorrect=1, got %d", v.VerificationsCorrect)
	}
}

func TestContestedVerification_3xMultiplier(t *testing.T) {
	k, ctx := setupKeeper(t)

	// The 3x multiplier boosts effective verification count, NOT MinContestedVerifications.
	// MinContestedVerifications=33 requires 33 raw contested correct verifications.
	// Guardian MinVerifications=333 uses effective count.
	//
	// With 267 total verifications and 33 contested correct:
	// effective = (267-33) + 33*3 = 234 + 99 = 333 → meets MinVerifications
	// Without multiplier: 267 < 333 → would NOT qualify
	tier := keeper.ComputeValidatorTier(
		ctx, k,
		"11111000000", // guardian stake
		267,           // total verifications (less than 333 threshold)
		250,           // correct (93.6% accuracy > 77%)
		0,             // no slashes
		33,            // 33 contested correct (meets MinContestedVerifications)
		0,
	)
	// effective = (267-33) + 33*3 = 234 + 99 = 333 ≥ 333 → qualifies
	if tier != types.TierGuardian {
		t.Errorf("3x multiplier: expected guardian (267 total + 33 contested*3 = 333 effective), got %v", tier)
	}

	// Without the multiplier bonus, 267 total wouldn't be enough.
	// Prove: with 266 total and 33 contested:
	// effective = (266-33) + 33*3 = 233 + 99 = 332 < 333 → doesn't qualify
	tier = keeper.ComputeValidatorTier(
		ctx, k,
		"11111000000",
		266, // one fewer → effective = 332 < 333
		250,
		0,
		33,
		0,
	)
	if tier == types.TierGuardian {
		t.Error("should NOT qualify for guardian with 332 effective verifications (< 333)")
	}

	// Confirm MinContestedVerifications is raw, not multiplied:
	// 32 contested correct < 33 → fails even with high total
	tier = keeper.ComputeValidatorTier(
		ctx, k,
		"11111000000",
		500, // plenty of total
		400,
		0,
		32, // only 32 < 33 minimum
		0,
	)
	if tier == types.TierGuardian {
		t.Error("should NOT qualify for guardian with only 32 contested (< 33 MinContestedVerifications)")
	}
}

// ---------- Task 4: VRF Selection Weight by Tier ----------

func TestVRFSelection_TierWeighting(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Create validators at each tier
	apprenticeAddr := sdk.AccAddress("val_vrf_apprentice__").String()
	verifiedAddr := sdk.AccAddress("val_vrf_verified____").String()
	bondedAddr := sdk.AccAddress("val_vrf_bonded______").String()
	guardianAddr := sdk.AccAddress("val_vrf_guardian____").String()

	registerTestValidator(t, k, ctx, apprenticeAddr, "did:lgm:vrfa", "111000")          // 0.111 LGM
	registerTestValidator(t, k, ctx, verifiedAddr, "did:lgm:vrfv", "1110000")           // 1.11 LGM
	registerTestValidator(t, k, ctx, bondedAddr, "did:lgm:vrfb", "1111000000")          // 1,111 LGM
	registerTestValidator(t, k, ctx, guardianAddr, "did:lgm:vrfg", "11111000000")       // 11,111 LGM

	// Set up verified tier requirements
	vv, _ := k.GetValidator(ctx, verifiedAddr)
	vv.VerificationsCompleted = 22
	vv.VerificationsCorrect = 22
	vv.Tier = "verified"
	k.SetValidator(ctx, vv)

	promoteToBonded(t, k, ctx, bondedAddr)
	promoteToGuardian(t, k, ctx, guardianAddr)

	params := k.GetParams(ctx)

	// Test effective selection stakes
	apprenticeV, _ := k.GetValidator(ctx, apprenticeAddr)
	verifiedV, _ := k.GetValidator(ctx, verifiedAddr)
	bondedV, _ := k.GetValidator(ctx, bondedAddr)
	guardianV, _ := k.GetValidator(ctx, guardianAddr)

	apprenticeStake := keeper.GetEffectiveSelectionStake(ctx, k, apprenticeV, params)
	verifiedStake := keeper.GetEffectiveSelectionStake(ctx, k, verifiedV, params)
	bondedStake := keeper.GetEffectiveSelectionStake(ctx, k, bondedV, params)
	guardianStake := keeper.GetEffectiveSelectionStake(ctx, k, guardianV, params)

	// Apprentice: virtual stake = 11 LGM = 11000000 ulgm
	if apprenticeStake != "11000000" {
		t.Errorf("apprentice: expected virtual stake 11000000, got %s", apprenticeStake)
	}

	// Verified: virtual stake = 11 LGM = 11000000 ulgm (same as apprentice)
	if verifiedStake != "11000000" {
		t.Errorf("verified: expected virtual stake 11000000, got %s", verifiedStake)
	}

	// Bonded: real stake (1,111 LGM = 1111000000 ulgm)
	if bondedStake != bondedV.TotalStake {
		t.Errorf("bonded: expected real stake %s, got %s", bondedV.TotalStake, bondedStake)
	}

	// Guardian: real stake (11,111 LGM = 11111000000 ulgm)
	if guardianStake != guardianV.TotalStake {
		t.Errorf("guardian: expected real stake %s, got %s", guardianV.TotalStake, guardianStake)
	}

	// Verify tier reward multipliers
	baseReward := "1000000"

	apprenticeReward := keeper.CalculateTierReward(ctx, k, types.TierApprentice, baseReward)
	verifiedReward := keeper.CalculateTierReward(ctx, k, types.TierVerified, baseReward)
	bondedReward := keeper.CalculateTierReward(ctx, k, types.TierBonded, baseReward)
	guardianReward := keeper.CalculateTierReward(ctx, k, types.TierGuardian, baseReward)

	// Apprentice: 0.1x → 100000
	if apprenticeReward != "100000" {
		t.Errorf("apprentice reward: expected 100000 (0.1x), got %s", apprenticeReward)
	}
	// Verified: 0.5x → 500000
	if verifiedReward != "500000" {
		t.Errorf("verified reward: expected 500000 (0.5x), got %s", verifiedReward)
	}
	// Bonded: 1.0x → 1000000
	if bondedReward != "1000000" {
		t.Errorf("bonded reward: expected 1000000 (1.0x), got %s", bondedReward)
	}
	// Guardian: 2.0x → 2000000
	if guardianReward != "2000000" {
		t.Errorf("guardian reward: expected 2000000 (2.0x), got %s", guardianReward)
	}
}

func TestVRFSelection_OnlyEligibleCategories(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Apprentice: protocol, computational, formal only
	apprenticeCategories := []struct {
		category string
		allowed  bool
	}{
		{"protocol", true},
		{"computational", true},
		{"formal", true},
		{"empirical", false},
		{"oracle", false},
		{"attestation", false},
		{"predictive", false},
		{"social", false},
		{"contested", false},
	}

	for _, tc := range apprenticeCategories {
		result := keeper.IsTierEligibleForCategory(ctx, k, types.TierApprentice, tc.category)
		if result != tc.allowed {
			t.Errorf("Apprentice + %s: expected %v, got %v", tc.category, tc.allowed, result)
		}
	}

	// Verified: adds empirical
	verifiedCategories := []struct {
		category string
		allowed  bool
	}{
		{"protocol", true},
		{"computational", true},
		{"formal", true},
		{"empirical", true},
		{"oracle", false},
		{"attestation", false},
	}

	for _, tc := range verifiedCategories {
		result := keeper.IsTierEligibleForCategory(ctx, k, types.TierVerified, tc.category)
		if result != tc.allowed {
			t.Errorf("Verified + %s: expected %v, got %v", tc.category, tc.allowed, result)
		}
	}

	// Bonded: adds oracle, attestation
	for _, cat := range []string{"protocol", "computational", "formal", "empirical", "oracle", "attestation"} {
		if !keeper.IsTierEligibleForCategory(ctx, k, types.TierBonded, cat) {
			t.Errorf("Bonded should be eligible for %s", cat)
		}
	}
	for _, cat := range []string{"predictive", "social", "contested"} {
		if keeper.IsTierEligibleForCategory(ctx, k, types.TierBonded, cat) {
			t.Errorf("Bonded should NOT be eligible for %s", cat)
		}
	}

	// Guardian: all categories
	allCategories := []string{"protocol", "computational", "formal", "empirical",
		"oracle", "attestation", "predictive", "social", "contested"}
	for _, cat := range allCategories {
		if !keeper.IsTierEligibleForCategory(ctx, k, types.TierGuardian, cat) {
			t.Errorf("Guardian should be eligible for %s", cat)
		}
	}
}

// ---------- Task 5: Edge Cases ----------

func TestTier_ExactThresholdStake(t *testing.T) {
	k, ctx := setupKeeper(t)

	// Exact minimum for Verified: 1,110,000 ulgm (1.11 LGM)
	tier := keeper.ComputeValidatorTier(ctx, k,
		"1110000", // exact Verified MinStake
		22,        // meets MinVerifications
		22,        // 100% accuracy > 77%
		0, 0, 0,
	)
	if tier != types.TierVerified {
		t.Errorf("exact Verified threshold: expected verified, got %v", tier)
	}

	// Exact minimum for Bonded: 1,111,000,000 ulgm (1,111 LGM)
	tier = keeper.ComputeValidatorTier(ctx, k,
		"1111000000", // exact Bonded MinStake
		11,           // meets MinVerifications
		11,           // 100% accuracy > 50%
		0, 0, 0,
	)
	if tier != types.TierBonded {
		t.Errorf("exact Bonded threshold: expected bonded, got %v", tier)
	}

	// Exact minimum for Guardian: 11,111,000,000 ulgm (11,111 LGM)
	tier = keeper.ComputeValidatorTier(ctx, k,
		"11111000000", // exact Guardian MinStake
		333,           // meets effective verifications (no contested multiplier since 33*3=99, 333-33+99=399 ≥ 333)
		300,           // 90% accuracy > 77%
		0,             // zero slashes
		33,            // meets MinContestedVerifications
		0,
	)
	if tier != types.TierGuardian {
		t.Errorf("exact Guardian threshold: expected guardian, got %v", tier)
	}
}

func TestTier_OneBelow(t *testing.T) {
	k, ctx := setupKeeper(t)

	// One below Verified threshold: 1,109,999 ulgm (< 1.11 LGM)
	tier := keeper.ComputeValidatorTier(ctx, k,
		"1109999", // 1 ulgm below Verified MinStake
		22,        // meets verifications
		22,        // 100% accuracy
		0, 0, 0,
	)
	if tier != types.TierApprentice {
		t.Errorf("one below Verified: expected apprentice, got %v", tier)
	}

	// One below Bonded threshold: 1,110,999,999 ulgm (< 1,111 LGM)
	tier = keeper.ComputeValidatorTier(ctx, k,
		"1110999999",
		22,
		22,
		0, 0, 0,
	)
	// Should qualify for Verified (has enough stake, verifications, accuracy)
	if tier != types.TierVerified {
		t.Errorf("one below Bonded: expected verified (meets verified requirements), got %v", tier)
	}

	// One below Guardian threshold: 11,110,999,999 ulgm (< 11,111 LGM)
	tier = keeper.ComputeValidatorTier(ctx, k,
		"11110999999",
		1200,
		1000,
		0,
		33,
		0,
	)
	// Should qualify for Bonded (meets bonded requirements)
	if tier != types.TierBonded {
		t.Errorf("one below Guardian: expected bonded, got %v", tier)
	}
}

func TestTier_ZeroVerifications(t *testing.T) {
	k, ctx := setupKeeper(t)

	// New validator with high stake but zero verifications → Apprentice
	// Even with guardian-level stake, 0 verifications means no upgrade
	tier := keeper.ComputeValidatorTier(ctx, k,
		"11111000000", // 11,111 LGM (guardian stake)
		0,             // zero verifications
		0,             // zero correct
		0, 0, 0,
	)
	if tier != types.TierApprentice {
		t.Errorf("zero verifications: expected apprentice regardless of stake, got %v", tier)
	}

	// Even with Bonded-level stake: 0 verifications → Apprentice
	tier = keeper.ComputeValidatorTier(ctx, k,
		"1111000000", // 1,111 LGM (bonded stake)
		0,
		0,
		0, 0, 0,
	)
	if tier != types.TierApprentice {
		t.Errorf("bonded stake but zero verifications: expected apprentice, got %v", tier)
	}

	// Verified-level stake with 0 verifications → Apprentice
	tier = keeper.ComputeValidatorTier(ctx, k,
		"1110000", // 1.11 LGM
		0,
		0,
		0, 0, 0,
	)
	if tier != types.TierApprentice {
		t.Errorf("verified stake but zero verifications: expected apprentice, got %v", tier)
	}
}
